package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FinalizeUEFIBoot installs the removable-media UEFI fallback path.
// Production-capable images prefer Ubuntu's Microsoft-signed shim followed by
// Ubuntu's signed GRUB. Developer images retain an unsigned grub-mkimage
// fallback only when the reviewed signed packages are unavailable.
func FinalizeUEFIBoot(res InstallResult) error {
	if runtime.GOARCH != "amd64" { return errors.New("UEFI fallback finalization is currently reviewed only for amd64") }
	if res.RootAPart == "" || res.EFIPart == "" || res.RootAUUID == "" { return errors.New("installer result is missing ROOT_A/EFI boot identity") }
	for _, p := range []string{res.RootAPart,res.EFIPart}{st,err:=os.Stat(p);if err!=nil{return fmt.Errorf("boot partition stat %s: %w",p,err)};if st.Mode()&os.ModeDevice==0{return fmt.Errorf("boot partition is not a block device: %s",p)}}

	mnt,err:=os.MkdirTemp("","kingai-uefi-");if err!=nil{return err};defer os.RemoveAll(mnt)
	rootMnt,efiMnt:=filepath.Join(mnt,"root"),filepath.Join(mnt,"efi");_ = os.MkdirAll(rootMnt,0755);_ = os.MkdirAll(efiMnt,0755)
	if err:=runBootTool("mount",res.RootAPart,rootMnt);err!=nil{return err};rootMounted:=true;defer func(){if rootMounted{_ = runBootTool("umount",rootMnt)}}()
	if err:=runBootTool("mount",res.EFIPart,efiMnt);err!=nil{return err};efiMounted:=true;defer func(){if efiMounted{_ = runBootTool("umount",efiMnt)}}()
	grubCfg:=filepath.Join(rootMnt,"boot/grub/grub.cfg");if st,err:=os.Stat(grubCfg);err!=nil||!st.Mode().IsRegular(){return errors.New("ROOT_A is missing the real /boot/grub/grub.cfg")}

	fallbackDir:=filepath.Join(efiMnt,"EFI/BOOT");if err:=os.MkdirAll(fallbackDir,0755);err!=nil{return err}
	relayBody:=fmt.Sprintf("search.fs_uuid %s root\nset prefix=($root)/boot/grub\nconfigfile $prefix/grub.cfg\n",res.RootAUUID)

	shim := "/usr/lib/shim/shimx64.efi.signed.latest"
	signedGrub := "/usr/lib/grub/x86_64-efi-signed/grubx64.efi.signed"
	if regular(shim) && regular(signedGrub) {
		if _,err:=exec.LookPath("sbverify");err!=nil{return errors.New("signed UEFI chain present but sbverify is missing")}
		if err:=verifyPESignature(shim);err!=nil{return fmt.Errorf("shim signature verification: %w",err)}
		if err:=verifyPESignature(signedGrub);err!=nil{return fmt.Errorf("GRUB signature verification: %w",err)}
		if err:=copyFile(shim,filepath.Join(fallbackDir,"BOOTX64.EFI"),0644);err!=nil{return err}
		if err:=copyFile(signedGrub,filepath.Join(fallbackDir,"grubx64.efi"),0644);err!=nil{return err}
		if err:=os.WriteFile(filepath.Join(fallbackDir,"grub.cfg"),[]byte(relayBody),0644);err!=nil{return err}
		marker := "chain=ubuntu-microsoft-shim+ubuntu-signed-grub\nsecure_boot_compatible=true\nproduction_key_material_embedded=false\n"
		if err:=os.WriteFile(filepath.Join(fallbackDir,"KINGAI-SECURE-BOOT"),[]byte(marker),0644);err!=nil{return err}
	} else {
		if _,err:=exec.LookPath("grub-mkimage");err!=nil{return errors.New("signed boot chain unavailable and developer fallback grub-mkimage is missing")}
		relay:=filepath.Join(mnt,"relay.cfg");if err:=os.WriteFile(relay,[]byte(relayBody),0600);err!=nil{return err}
		fallback:=filepath.Join(fallbackDir,"BOOTX64.EFI")
		mods:=[]string{"part_gpt","ext2","search","search_fs_uuid","normal","configfile","linux"}
		args:=[]string{"-O","x86_64-efi","-o",fallback,"-p","/boot/grub","-c",relay};args=append(args,mods...)
		if err:=runBootTool("grub-mkimage",args...);err!=nil{return err}
		if st,err:=os.Stat(fallback);err!=nil{return fmt.Errorf("fallback EFI image missing: %w",err)}else if st.Size()<64*1024{return fmt.Errorf("fallback EFI image is unexpectedly small: %d bytes",st.Size())}
		if err:=os.WriteFile(filepath.Join(fallbackDir,"KINGAI-DEVELOPER-UNSIGNED-BOOT"),[]byte("secure_boot_compatible=false\n"),0644);err!=nil{return err}
	}
	if err:=runBootTool("sync");err!=nil{return err}
	if err:=runBootTool("umount",efiMnt);err!=nil{return err};efiMounted=false
	if err:=runBootTool("umount",rootMnt);err!=nil{return err};rootMounted=false
	return nil
}

func verifyPESignature(path string) error { out,err:=exec.Command("sbverify","--list",path).CombinedOutput();if err!=nil{return fmt.Errorf("sbverify: %w: %s",err,strings.TrimSpace(string(out)))};if !strings.Contains(strings.ToLower(string(out)),"signature"){return errors.New("no PE signature reported")};return nil }
func regular(path string)bool{st,err:=os.Stat(path);return err==nil&&st.Mode().IsRegular()&&st.Size()>64*1024}
func copyFile(src,dst string,mode os.FileMode)error{in,err:=os.Open(src);if err!=nil{return err};defer in.Close();out,err:=os.OpenFile(dst,os.O_CREATE|os.O_TRUNC|os.O_WRONLY,mode);if err!=nil{return err};_,cpErr:=io.Copy(out,in);closeErr:=out.Close();if cpErr!=nil{return cpErr};return closeErr}
func runBootTool(name string,args ...string)error{cmd:=exec.Command(name,args...);cmd.Stdout=os.Stderr;cmd.Stderr=os.Stderr;if err:=cmd.Run();err!=nil{return fmt.Errorf("%s %s: %w",name,strings.Join(args," "),err)};return nil}
