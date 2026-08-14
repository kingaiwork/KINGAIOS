package devicepackadmin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/audit"
	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
)

var packIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,127}$`)

var (
	ErrRootRequired = errors.New("Device Pack lifecycle changes require root")
	ErrBusy         = errors.New("another Device Pack lifecycle operation is active")
)

type Paths struct {
	ManifestDir  string
	ArtifactRoot string
	TrustDir     string
	LockPath     string
	AdminAudit   string
}

func DefaultPaths() Paths {
	return Paths{
		ManifestDir: "/etc/kingai/device-packs",
		ArtifactRoot: "/usr/lib/kingai/device-packs",
		TrustDir: "/etc/kingai/trust/device-pack-keys",
		LockPath: "/run/lock/kingai-device-pack.lock",
		AdminAudit: "/var/log/kingai/device-pack-admin.jsonl",
	}
}

type InstallOptions struct {
	ManifestPath  string
	SignaturePath string
	ArtifactDir   string
	BoardID       string
	Confirmation  string
}

type Result struct {
	Action            string   `json:"action"`
	PackID            string   `json:"pack_id"`
	Version           string   `json:"version,omitempty"`
	ManifestPath      string   `json:"manifest_path,omitempty"`
	ArtifactPath      string   `json:"artifact_path,omitempty"`
	Backups           []string `json:"backups,omitempty"`
	RestartRequired   bool     `json:"restart_required"`
}

type Installed struct {
	PackID          string `json:"pack_id"`
	Version         string `json:"version"`
	Arch            string `json:"arch"`
	ManifestPath    string `json:"manifest_path"`
	SignatureOK     bool   `json:"signature_ok"`
	ArtifactsOK     bool   `json:"artifacts_ok"`
	Error           string `json:"error,omitempty"`
}

// Install activates a signed Device Pack only after staging and verifying the
// exact manifest, signature and artifact bytes inside root-owned destination
// files. The manifest is renamed into place last, making activation fail-closed.
func Install(paths Paths, opts InstallOptions) (Result, error) {
	if os.Geteuid() != 0 { return Result{}, ErrRootRequired }
	if err := validatePaths(paths); err != nil { return Result{}, err }
	unlock, err := lockLifecycle(paths.LockPath)
	if err != nil { return Result{}, err }
	defer unlock()

	for label, path := range map[string]string{"manifest":opts.ManifestPath,"signature":opts.SignaturePath,"artifact directory":opts.ArtifactDir} {
		if err := cleanAbsolute(path); err != nil { return Result{}, fmt.Errorf("%s source: %w",label,err) }
	}
	sourceManifest, err := devicepack.Load(opts.ManifestPath)
	if err != nil { return Result{}, fmt.Errorf("parse source manifest: %w",err) }
	if opts.Confirmation != "INSTALL:"+sourceManifest.ID { return Result{}, errors.New("confirmation must equal INSTALL:<pack-id>") }
	if sourceManifest.Arch != runtime.GOARCH { return Result{}, fmt.Errorf("Device Pack targets %s, host is %s",sourceManifest.Arch,runtime.GOARCH) }
	if err := boardCompatible(sourceManifest,opts.BoardID); err != nil { return Result{}, err }
	if err := rejectDuplicateActiveID(paths.ManifestDir,sourceManifest.ID); err != nil { return Result{}, err }

	manifestStage, err := os.MkdirTemp(paths.ManifestDir,".install-")
	if err != nil { return Result{}, err }
	artifactStage, err := os.MkdirTemp(paths.ArtifactRoot,".install-")
	if err != nil { _ = os.RemoveAll(manifestStage); return Result{}, err }
	defer os.RemoveAll(manifestStage)
	defer os.RemoveAll(artifactStage)
	_ = os.Chmod(manifestStage,0o700)
	_ = os.Chmod(artifactStage,0o700)

	stagedManifestPath:=filepath.Join(manifestStage,sourceManifest.ID+".json")
	stagedSignaturePath:=stagedManifestPath+".sig"
	if err:=copyRegular(opts.ManifestPath,stagedManifestPath,0o644,1<<20);err!=nil{return Result{},err}
	if err:=copyRegular(opts.SignaturePath,stagedSignaturePath,0o644,16<<10);err!=nil{return Result{},err}
	stagedPackDir:=filepath.Join(artifactStage,sourceManifest.ID)
	if err:=os.Mkdir(stagedPackDir,0o755);err!=nil{return Result{},err}
	for _,artifact:=range sourceManifest.Artifacts{
		source:=filepath.Join(opts.ArtifactDir,artifact.Name)
		if filepath.Dir(source)!=filepath.Clean(opts.ArtifactDir){return Result{},fmt.Errorf("artifact %q escaped source directory",artifact.Name)}
		dest:=filepath.Join(stagedPackDir,artifact.Name)
		if err:=copyAndVerifyArtifact(source,dest,artifact);err!=nil{return Result{},err}
	}
	if err:=devicepack.VerifyDetachedSignature(stagedManifestPath,stagedSignaturePath,paths.TrustDir);err!=nil{return Result{},err}
	stagedManifest,err:=devicepack.Load(stagedManifestPath);if err!=nil{return Result{},err}
	if stagedManifest.ID!=sourceManifest.ID||stagedManifest.Version!=sourceManifest.Version{return Result{},errors.New("staged manifest identity changed during copy")}
	if err:=devicepack.VerifyInstalledArtifacts(stagedManifest,artifactStage);err!=nil{return Result{},err}

	if err:=appendAdminAudit(paths.AdminAudit,"devicepack.install.prepare",sourceManifest.ID,sourceManifest.Version,true,"verified staged release");err!=nil{return Result{},fmt.Errorf("admin audit unavailable before activation: %w",err)}
	result,err:=promote(paths,stagedManifestPath,stagedSignaturePath,stagedPackDir,sourceManifest)
	if err!=nil{_ = appendAdminAudit(paths.AdminAudit,"devicepack.install",sourceManifest.ID,sourceManifest.Version,false,err.Error());return Result{},err}
	if err:=appendAdminAudit(paths.AdminAudit,"devicepack.install",sourceManifest.ID,sourceManifest.Version,true,"activated; kingaid restart required");err!=nil{return result,fmt.Errorf("Device Pack activated but completion audit failed: %w",err)}
	return result,nil
}

func List(paths Paths) ([]Installed,error){
	if err:=validateReadPaths(paths);err!=nil{return nil,err}
	entries,err:=os.ReadDir(paths.ManifestDir);if err!=nil{return nil,err}
	out:=make([]Installed,0)
	for _,entry:=range entries{
		if entry.IsDir()||!strings.HasSuffix(entry.Name(),".json"){continue}
		path:=filepath.Join(paths.ManifestDir,entry.Name())
		item:=Installed{ManifestPath:path}
		manifest,loadErr:=devicepack.Load(path)
		if loadErr!=nil{item.Error=loadErr.Error();out=append(out,item);continue}
		item.PackID=manifest.ID;item.Version=manifest.Version;item.Arch=manifest.Arch
		if err:=devicepack.VerifyDetachedSignature(path,path+".sig",paths.TrustDir);err!=nil{item.Error=err.Error();out=append(out,item);continue};item.SignatureOK=true
		if err:=devicepack.VerifyInstalledArtifacts(manifest,paths.ArtifactRoot);err!=nil{item.Error=err.Error();out=append(out,item);continue};item.ArtifactsOK=true
		out=append(out,item)
	}
	sort.Slice(out,func(i,j int)bool{return out[i].PackID<out[j].PackID})
	return out,nil
}

// Deactivate removes the manifest first so a restart can never load a partially
// removed pack. Artifacts are moved to a hidden backup rather than destroyed.
func Deactivate(paths Paths,packID,confirmation string)(Result,error){
	if os.Geteuid()!=0{return Result{},ErrRootRequired}
	if !packIDPattern.MatchString(packID)||strings.Contains(packID,".."){return Result{},errors.New("invalid pack id")}
	if confirmation!="DEACTIVATE:"+packID{return Result{},errors.New("confirmation must equal DEACTIVATE:<pack-id>")}
	if err:=validatePaths(paths);err!=nil{return Result{},err}
	unlock,err:=lockLifecycle(paths.LockPath);if err!=nil{return Result{},err};defer unlock()
	manifestPath:=filepath.Join(paths.ManifestDir,packID+".json")
	manifest,err:=devicepack.Load(manifestPath);if err!=nil{return Result{},err}
	if manifest.ID!=packID{return Result{},errors.New("canonical manifest pack id mismatch")}
	if err:=appendAdminAudit(paths.AdminAudit,"devicepack.deactivate.prepare",packID,manifest.Version,true,"root confirmed deactivation");err!=nil{return Result{},err}
	token,err:=backupToken();if err!=nil{return Result{},err}
	backupManifest:=filepath.Join(paths.ManifestDir,".disabled-"+packID+"-"+token+".manifest")
	backupSignature:=backupManifest+".sig"
	backupArtifacts:=filepath.Join(paths.ArtifactRoot,".disabled-"+packID+"-"+token)
	if err:=os.Rename(manifestPath,backupManifest);err!=nil{return Result{},err}
	backups:=[]string{backupManifest}
	if _,err:=os.Lstat(manifestPath+".sig");err==nil{if err:=os.Rename(manifestPath+".sig",backupSignature);err!=nil{return Result{},err};backups=append(backups,backupSignature)}
	artifactPath:=filepath.Join(paths.ArtifactRoot,packID)
	if _,err:=os.Lstat(artifactPath);err==nil{if err:=os.Rename(artifactPath,backupArtifacts);err!=nil{return Result{},err};backups=append(backups,backupArtifacts)}
	_ = syncDir(paths.ManifestDir);_ = syncDir(paths.ArtifactRoot)
	result:=Result{Action:"deactivated",PackID:packID,Version:manifest.Version,Backups:backups,RestartRequired:true}
	if err:=appendAdminAudit(paths.AdminAudit,"devicepack.deactivate",packID,manifest.Version,true,"manifest deactivated; backup retained; kingaid restart required");err!=nil{return result,err}
	return result,nil
}

func promote(paths Paths,stagedManifest,stagedSignature,stagedPackDir string,manifest devicepack.Manifest)(Result,error){
	token,err:=backupToken();if err!=nil{return Result{},err}
	finalManifest:=filepath.Join(paths.ManifestDir,manifest.ID+".json")
	finalSignature:=finalManifest+".sig"
	finalArtifacts:=filepath.Join(paths.ArtifactRoot,manifest.ID)
	backupManifest:=filepath.Join(paths.ManifestDir,".backup-"+manifest.ID+"-"+token+".manifest")
	backupSignature:=backupManifest+".sig"
	backupArtifacts:=filepath.Join(paths.ArtifactRoot,".backup-"+manifest.ID+"-"+token)
	backups:=[]string{}
	oldArtifacts:=false;oldManifest:=false;oldSignature:=false;newArtifacts:=false;newSignature:=false
	rollback:=func(){
		if newSignature{_ = os.Remove(finalSignature)}
		_ = os.Remove(finalManifest)
		if newArtifacts{_ = os.RemoveAll(finalArtifacts)}
		if oldArtifacts{_ = os.Rename(backupArtifacts,finalArtifacts)}
		if oldSignature{_ = os.Rename(backupSignature,finalSignature)}
		if oldManifest{_ = os.Rename(backupManifest,finalManifest)}
		_ = syncDir(paths.ManifestDir);_ = syncDir(paths.ArtifactRoot)
	}
	if _,err:=os.Lstat(finalArtifacts);err==nil{if err:=os.Rename(finalArtifacts,backupArtifacts);err!=nil{return Result{},err};oldArtifacts=true;backups=append(backups,backupArtifacts)} else if !errors.Is(err,os.ErrNotExist){return Result{},err}
	if err:=os.Rename(stagedPackDir,finalArtifacts);err!=nil{rollback();return Result{},err};newArtifacts=true
	if _,err:=os.Lstat(finalManifest);err==nil{if err:=os.Rename(finalManifest,backupManifest);err!=nil{rollback();return Result{},err};oldManifest=true;backups=append(backups,backupManifest)} else if !errors.Is(err,os.ErrNotExist){rollback();return Result{},err}
	if _,err:=os.Lstat(finalSignature);err==nil{if err:=os.Rename(finalSignature,backupSignature);err!=nil{rollback();return Result{},err};oldSignature=true;backups=append(backups,backupSignature)} else if !errors.Is(err,os.ErrNotExist){rollback();return Result{},err}
	if err:=os.Rename(stagedSignature,finalSignature);err!=nil{rollback();return Result{},err};newSignature=true
	if err:=os.Rename(stagedManifest,finalManifest);err!=nil{rollback();return Result{},err}
	if err:=syncDir(paths.ArtifactRoot);err!=nil{return Result{},err}
	if err:=syncDir(paths.ManifestDir);err!=nil{return Result{},err}
	return Result{Action:"installed",PackID:manifest.ID,Version:manifest.Version,ManifestPath:finalManifest,ArtifactPath:finalArtifacts,Backups:backups,RestartRequired:true},nil
}

func rejectDuplicateActiveID(manifestDir,packID string)error{
	entries,err:=os.ReadDir(manifestDir);if err!=nil{return err}
	canonical:=packID+".json"
	for _,entry:=range entries{
		if entry.IsDir()||!strings.HasSuffix(entry.Name(),".json")||entry.Name()==canonical{continue}
		manifest,err:=devicepack.Load(filepath.Join(manifestDir,entry.Name()));if err!=nil{continue}
		if manifest.ID==packID{return fmt.Errorf("active Device Pack %q already exists as noncanonical manifest %s",packID,entry.Name())}
	}
	return nil
}

func boardCompatible(manifest devicepack.Manifest,boardID string)error{
	if len(manifest.BoardIDs)==0{return nil}
	for _,candidate:=range manifest.BoardIDs{if candidate==boardID{return nil}}
	if strings.TrimSpace(boardID)==""{return fmt.Errorf("Device Pack %q requires a trusted board id",manifest.ID)}
	return fmt.Errorf("Device Pack %q does not match board id %q",manifest.ID,boardID)
}

func validatePaths(paths Paths)error{
	if err:=validateReadPaths(paths);err!=nil{return err}
	for _,path:=range []string{paths.ManifestDir,paths.ArtifactRoot,paths.TrustDir}{if err:=trustedRootDir(path);err!=nil{return err}}
	if err:=cleanAbsolute(paths.LockPath);err!=nil{return err}
	if err:=cleanAbsolute(paths.AdminAudit);err!=nil{return err}
	return nil
}
func validateReadPaths(paths Paths)error{
	for _,path:=range []string{paths.ManifestDir,paths.ArtifactRoot,paths.TrustDir}{if err:=cleanAbsolute(path);err!=nil{return err}}
	return nil
}
func trustedRootDir(path string)error{
	if err:=cleanAbsolute(path);err!=nil{return err}
	info,err:=os.Stat(path);if err!=nil{return err};if !info.IsDir(){return errors.New("trusted path is not a directory")};if info.Mode().Perm()&0o022!=0{return errors.New("trusted directory must not be group/world writable")}
	stat,ok:=info.Sys().(*syscall.Stat_t);if !ok||stat.Uid!=0{return errors.New("trusted directory must be root-owned")};return nil
}
func cleanAbsolute(path string)error{if !filepath.IsAbs(path)||filepath.Clean(path)!=path||strings.ContainsAny(path,"\x00\n\r"){return errors.New("path must be a clean absolute path")};return nil}

func copyRegular(source,dest string,mode os.FileMode,max int64)error{
	info,err:=os.Lstat(source);if err!=nil{return err};if info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){return errors.New("source must be a regular non-symlink file")};if info.Size()<=0||info.Size()>max{return errors.New("source file size is outside allowed bounds")}
	in,err:=os.Open(source);if err!=nil{return err};defer in.Close();out,err:=os.OpenFile(dest,os.O_CREATE|os.O_EXCL|os.O_WRONLY,mode);if err!=nil{return err}
	ok:=false;defer func(){_ = out.Close();if !ok{_ = os.Remove(dest)}}();n,err:=io.Copy(out,io.LimitReader(in,max+1));if err!=nil{return err};if n!=info.Size(){return errors.New("source changed while copying")};if err:=out.Sync();err!=nil{return err};if err:=out.Close();err!=nil{return err};ok=true;return nil
}

func copyAndVerifyArtifact(source,dest string,artifact devicepack.Artifact)error{
	info,err:=os.Lstat(source);if err!=nil{return err};if info.Mode()&os.ModeSymlink!=0||!info.Mode().IsRegular(){return fmt.Errorf("artifact %q must be a regular non-symlink file",artifact.Name)};if info.Size()!=artifact.SizeBytes{return fmt.Errorf("artifact %q size mismatch",artifact.Name)}
	in,err:=os.Open(source);if err!=nil{return err};defer in.Close();out,err:=os.OpenFile(dest,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0o644);if err!=nil{return err};ok:=false;defer func(){_ = out.Close();if !ok{_ = os.Remove(dest)}}()
	h:=sha256.New();n,err:=io.Copy(io.MultiWriter(out,h),in);if err!=nil{return err};if n!=artifact.SizeBytes{return errors.New("artifact changed while copying")};if hex.EncodeToString(h.Sum(nil))!=artifact.SHA256{return fmt.Errorf("artifact %q sha256 mismatch",artifact.Name)};if err:=out.Sync();err!=nil{return err};if err:=out.Close();err!=nil{return err};ok=true;return nil
}

func lockLifecycle(path string)(func(),error){
	if err:=os.MkdirAll(filepath.Dir(path),0o755);err!=nil{return nil,err};f,err:=os.OpenFile(path,os.O_CREATE|os.O_RDWR,0o600);if err!=nil{return nil,err};if err:=syscall.Flock(int(f.Fd()),syscall.LOCK_EX|syscall.LOCK_NB);err!=nil{_ = f.Close();return nil,ErrBusy};return func(){_ = syscall.Flock(int(f.Fd()),syscall.LOCK_UN);_ = f.Close()},nil
}
func backupToken()(string,error){var raw [8]byte;if _,err:=rand.Read(raw[:]);err!=nil{return "",err};return time.Now().UTC().Format("20060102T150405Z")+"-"+hex.EncodeToString(raw[:]),nil}
func syncDir(path string)error{f,err:=os.Open(path);if err!=nil{return err};defer f.Close();return f.Sync()}
func appendAdminAudit(path,eventType,packID,version string,allowed bool,reason string)error{return audit.Append(path,audit.Event{Type:eventType,Capability:"trust.modify",Allowed:allowed,Risk:6,PeerUID:0,TargetHash:audit.HashTarget(packID),Reason:"pack="+packID+" version="+version+" "+reason})}

func MarshalIndented(value any)([]byte,error){b,err:=json.MarshalIndent(value,"","  ");if err!=nil{return nil,err};return append(b,'\n'),nil}
