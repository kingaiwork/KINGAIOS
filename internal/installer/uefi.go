package installer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FinalizeUEFIBoot replaces the removable-media fallback EFI image with a
// deterministic GRUB image that embeds only a tiny relay configuration. The
// relay discovers ROOT_A by filesystem UUID and then hands control to the real
// /boot/grub/grub.cfg on that verified root filesystem.
//
// This is intentionally separate from generic disk copying so the fallback
// boot path can be tested and failed closed without changing partition data.
func FinalizeUEFIBoot(res InstallResult) error {
	if runtime.GOARCH != "amd64" {
		return errors.New("UEFI fallback finalization is currently reviewed only for amd64")
	}
	if res.RootAPart == "" || res.EFIPart == "" || res.RootAUUID == "" {
		return errors.New("installer result is missing ROOT_A/EFI boot identity")
	}
	if _, err := exec.LookPath("grub-mkimage"); err != nil {
		return errors.New("required UEFI finalizer tool missing: grub-mkimage")
	}
	for _, p := range []string{res.RootAPart, res.EFIPart} {
		st, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("boot partition stat %s: %w", p, err)
		}
		if st.Mode()&os.ModeDevice == 0 {
			return fmt.Errorf("boot partition is not a block device: %s", p)
		}
	}

	mnt, err := os.MkdirTemp("", "kingai-uefi-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mnt)
	rootMnt := filepath.Join(mnt, "root")
	efiMnt := filepath.Join(mnt, "efi")
	if err := os.MkdirAll(rootMnt, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(efiMnt, 0o755); err != nil {
		return err
	}

	if err := runBootTool("mount", res.RootAPart, rootMnt); err != nil {
		return err
	}
	rootMounted := true
	defer func() {
		if rootMounted {
			_ = runBootTool("umount", rootMnt)
		}
	}()
	if err := runBootTool("mount", res.EFIPart, efiMnt); err != nil {
		return err
	}
	efiMounted := true
	defer func() {
		if efiMounted {
			_ = runBootTool("umount", efiMnt)
		}
	}()

	grubCfg := filepath.Join(rootMnt, "boot/grub/grub.cfg")
	if st, err := os.Stat(grubCfg); err != nil || !st.Mode().IsRegular() {
		return errors.New("ROOT_A is missing the real /boot/grub/grub.cfg")
	}

	relay := filepath.Join(mnt, "relay.cfg")
	relayBody := fmt.Sprintf("search.fs_uuid %s root\nset prefix=($root)/boot/grub\n", res.RootAUUID)
	if err := os.WriteFile(relay, []byte(relayBody), 0o600); err != nil {
		return err
	}

	fallbackDir := filepath.Join(efiMnt, "EFI/BOOT")
	if err := os.MkdirAll(fallbackDir, 0o755); err != nil {
		return err
	}
	fallback := filepath.Join(fallbackDir, "BOOTX64.EFI")
	modules := []string{
		"part_gpt",
		"ext2",
		"search",
		"search_fs_uuid",
		"normal",
		"configfile",
		"linux",
	}
	args := []string{
		"-O", "x86_64-efi",
		"-o", fallback,
		"-p", "/boot/grub",
		"-c", relay,
	}
	args = append(args, modules...)
	if err := runBootTool("grub-mkimage", args...); err != nil {
		return err
	}
	st, err := os.Stat(fallback)
	if err != nil {
		return fmt.Errorf("fallback EFI image missing after grub-mkimage: %w", err)
	}
	if st.Size() < 64*1024 {
		return fmt.Errorf("fallback EFI image is unexpectedly small: %d bytes", st.Size())
	}
	if err := runBootTool("sync"); err != nil {
		return err
	}

	if err := runBootTool("umount", efiMnt); err != nil {
		return err
	}
	efiMounted = false
	if err := runBootTool("umount", rootMnt); err != nil {
		return err
	}
	rootMounted = false
	return nil
}

func runBootTool(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
