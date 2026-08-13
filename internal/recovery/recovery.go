package recovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

type InspectResult struct {
	TargetDisk string `json:"target_disk"`
	ActiveSlot kingupdate.Slot `json:"active_slot"`
	PendingSlot kingupdate.Slot `json:"pending_slot,omitempty"`
	ActiveVersion string `json:"active_version"`
	PendingVersion string `json:"pending_version,omitempty"`
	RootAUUID string `json:"root_a_uuid"`
	RootBUUID string `json:"root_b_uuid"`
	State kingupdate.SlotState `json:"state"`
}

type Options struct { TargetDisk, StateKey, Confirmation string }

func Inspect(opts Options) (InspectResult,error) {
	ctx,err:=open(opts,false);if err!=nil{return InspectResult{},err};defer ctx.close()
	return ctx.inspect(),nil
}

func Rollback(opts Options)(InspectResult,error){
	if err:=writeGate(opts,"ROLLBACK:");err!=nil{return InspectResult{},err}
	ctx,err:=open(opts,true);if err!=nil{return InspectResult{},err};defer ctx.close()
	state:=ctx.state
	if state.PendingSlot=="" { return InspectResult{},errors.New("no pending slot exists; refusing unnecessary recovery rollback") }
	rolled,err:=state.Rollback();if err!=nil{return InspectResult{},err}
	if err:=kingupdate.RollbackBoot(ctx.rootA,state.ActiveSlot);err!=nil{return InspectResult{},err}
	if err:=kingupdate.SaveSlotStateFile(ctx.statePath,rolled);err!=nil{return InspectResult{},err}
	if err:=run("sync");err!=nil{return InspectResult{},err}
	ctx.state=rolled
	return ctx.inspect(),nil
}

func RepairBoot(opts Options)(InspectResult,error){
	if err:=writeGate(opts,"REPAIR:");err!=nil{return InspectResult{},err}
	ctx,err:=open(opts,true);if err!=nil{return InspectResult{},err};defer ctx.close()
	ctrl:=kingupdate.BootController{RootAPath:ctx.rootA,RootBPath:ctx.rootB,RootAUUID:ctx.aUUID,RootBUUID:ctx.bUUID}
	if err:=ctrl.WriteConfig();err!=nil{return InspectResult{},err}
	if ctx.state.PendingSlot!="" { if err:=kingupdate.SetPendingBoot(ctx.rootA,ctx.state.ActiveSlot,ctx.state.PendingSlot);err!=nil{return InspectResult{},err} } else { if err:=kingupdate.ConfirmBoot(ctx.rootA,ctx.state.ActiveSlot);err!=nil{return InspectResult{},err} }
	if err:=run("sync");err!=nil{return InspectResult{},err}
	return ctx.inspect(),nil
}

type context struct{target,rootA,rootB,stateMnt,statePath,mapper,aUUID,bUUID string; state kingupdate.SlotState; mounted []string; mapperOpen bool}
func (c *context) close(){for i:=len(c.mounted)-1;i>=0;i--{_ = exec.Command("umount",c.mounted[i]).Run()};if c.mapperOpen{_ = exec.Command("cryptsetup","close",c.mapper).Run()}}
func (c *context) inspect() InspectResult{return InspectResult{TargetDisk:c.target,ActiveSlot:c.state.ActiveSlot,PendingSlot:c.state.PendingSlot,ActiveVersion:c.state.ActiveVersion,PendingVersion:c.state.PendingVersion,RootAUUID:c.aUUID,RootBUUID:c.bUUID,State:c.state}}

func open(opts Options,write bool)(*context,error){
	if runtime.GOARCH!="amd64"{return nil,errors.New("recovery execution is currently reviewed only for amd64")}
	if os.Geteuid()!=0{return nil,errors.New("recovery requires root")}
	if opts.TargetDisk==""||opts.StateKey==""{return nil,errors.New("target-disk and state-key are required")}
	if os.Getenv("KINGAI_RECOVERY_CI")=="1"&&!strings.HasPrefix(opts.TargetDisk,"/dev/nbd"){return nil,errors.New("CI recovery is restricted to disposable /dev/nbd devices")}
	st,err:=os.Stat(opts.StateKey);if err!=nil{return nil,err};if !st.Mode().IsRegular()||st.Size()<32||st.Mode().Perm()&0o077!=0{return nil,errors.New("STATE key must be a private regular file of at least 32 bytes")}
	for _,tool:=range []string{"blkid","cryptsetup","mount","umount","grub-editenv","sync"}{if _,e:=exec.LookPath(tool);e!=nil{return nil,fmt.Errorf("required recovery tool missing: %s",tool)}}
	parts:=[]string{part(opts.TargetDisk,1),part(opts.TargetDisk,2),part(opts.TargetDisk,3),part(opts.TargetDisk,4)}
	for i,want:=range []string{"KINGAI_EFI","KINGAI_ROOT_A","KINGAI_ROOT_B"}{got,e:=out("blkid","-s","LABEL","-o","value",parts[i]);if e!=nil||got!=want{return nil,fmt.Errorf("partition %s is not %s",parts[i],want)}}
	if e:=exec.Command("cryptsetup","isLuks",parts[3]).Run();e!=nil{return nil,errors.New("KINGAI_STATE is not LUKS")}
	work,e:=os.MkdirTemp("","kingai-recovery-");if e!=nil{return nil,e}
	c:=&context{target:opts.TargetDisk,rootA:filepath.Join(work,"a"),rootB:filepath.Join(work,"b"),stateMnt:filepath.Join(work,"state"),mapper:fmt.Sprintf("kingai-recovery-%d",os.Getpid())}
	for _,p:=range []string{c.rootA,c.rootB,c.stateMnt}{_ = os.MkdirAll(p,0755)}
	if e:=run("cryptsetup","open","--key-file",opts.StateKey,parts[3],c.mapper);e!=nil{os.RemoveAll(work);return nil,e};c.mapperOpen=true
	for _,pair:=range [][2]string{{parts[1],c.rootA},{parts[2],c.rootB},{"/dev/mapper/"+c.mapper,c.stateMnt}}{args:=[]string{};if !write{args=append(args,"-o","ro")};args=append(args,pair[0],pair[1]);if e:=run("mount",args...);e!=nil{c.close();os.RemoveAll(work);return nil,e};c.mounted=append(c.mounted,pair[1])}
	c.statePath=filepath.Join(c.stateMnt,"kingai/update/slots.json");c.state,e=kingupdate.LoadSlotStateFile(c.statePath);if e!=nil{c.close();os.RemoveAll(work);return nil,e}
	c.aUUID,e=out("blkid","-s","UUID","-o","value",parts[1]);if e!=nil{c.close();return nil,e};c.bUUID,e=out("blkid","-s","UUID","-o","value",parts[2]);if e!=nil{c.close();return nil,e}
	return c,nil
}
func writeGate(opts Options,prefix string)error{if os.Getenv("KINGAI_RECOVERY_ALLOW_WRITE")!="1"{return errors.New("recovery writes are disabled; set KINGAI_RECOVERY_ALLOW_WRITE=1 explicitly")};if opts.Confirmation!=prefix+opts.TargetDisk{return fmt.Errorf("confirmation mismatch; expected %s%s",prefix,opts.TargetDisk)};return nil}
func part(t string,n int)string{if t==""{return ""};last:=t[len(t)-1];if last>='0'&&last<='9'{return fmt.Sprintf("%sp%d",t,n)};return fmt.Sprintf("%s%d",t,n)}
func run(n string,a ...string)error{cmd:=exec.Command(n,a...);cmd.Stdout=os.Stdout;cmd.Stderr=os.Stderr;if e:=cmd.Run();e!=nil{return fmt.Errorf("%s: %w",n,e)};return nil}
func out(n string,a ...string)(string,error){b,e:=exec.Command(n,a...).CombinedOutput();if e!=nil{return "",fmt.Errorf("%s: %w: %s",n,e,strings.TrimSpace(string(b)))};return strings.TrimSpace(string(b)),nil}
func Marshal(v any)string{b,_:=json.MarshalIndent(v,"","  ");return string(b)}
var _ = time.Now
