package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BootController keeps the GRUB control plane on ROOT_A while allowing either
// root slot to be the running system. ROOT_A's /boot/grub is therefore never
// replaced by an A/B payload update.
type BootController struct {
	RootAPath string
	RootBPath string
	RootAUUID string
	RootBUUID string
}

func (b BootController) Validate() error {
	if b.RootAPath == "" || b.RootBPath == "" || b.RootAUUID == "" || b.RootBUUID == "" {
		return errors.New("boot controller requires both mounted roots and UUIDs")
	}
	if b.RootAUUID == b.RootBUUID {
		return errors.New("A and B root UUIDs must differ")
	}
	return nil
}

func (b BootController) WriteConfig() error {
	if err := b.Validate(); err != nil {
		return err
	}
	aKernel, err := newestBootFile(b.RootAPath, "vmlinuz-*")
	if err != nil { return fmt.Errorf("slot A kernel: %w", err) }
	aInitrd, err := newestBootFile(b.RootAPath, "initrd.img-*")
	if err != nil { return fmt.Errorf("slot A initrd: %w", err) }
	bKernel, err := newestBootFile(b.RootBPath, "vmlinuz-*")
	if err != nil { return fmt.Errorf("slot B kernel: %w", err) }
	bInitrd, err := newestBootFile(b.RootBPath, "initrd.img-*")
	if err != nil { return fmt.Errorf("slot B initrd: %w", err) }

	grubDir := filepath.Join(b.RootAPath, "boot/grub")
	if err := os.MkdirAll(grubDir, 0o755); err != nil { return err }
	cfg := fmt.Sprintf(`set timeout=2
set default=slotA
if [ -s $prefix/grubenv ]; then
  load_env
fi
if [ "$kingai_active" = "B" ]; then
  set default=slotB
fi
if [ "$kingai_pending" = "A" -o "$kingai_pending" = "B" ]; then
  if [ "$kingai_attempted" = "1" ]; then
    # The pending slot failed to reach userspace health confirmation on the
    # previous boot. Return to the last confirmed active slot immediately.
    set kingai_pending=
    set kingai_attempted=
    save_env kingai_pending kingai_attempted
  else
    set default=slot$kingai_pending
    set kingai_attempted=1
    save_env kingai_attempted
  fi
fi
menuentry 'KINGAI OS — Slot A' --id slotA {
  search --no-floppy --fs-uuid --set=root %s
  linux /boot/%s root=UUID=%s ro console=tty0 console=ttyS0,115200n8
  initrd /boot/%s
}
menuentry 'KINGAI OS — Slot B' --id slotB {
  search --no-floppy --fs-uuid --set=root %s
  linux /boot/%s root=UUID=%s ro console=tty0 console=ttyS0,115200n8
  initrd /boot/%s
}
`, b.RootAUUID, filepath.Base(aKernel), b.RootAUUID, filepath.Base(aInitrd), b.RootBUUID, filepath.Base(bKernel), b.RootBUUID, filepath.Base(bInitrd))
	return os.WriteFile(filepath.Join(grubDir, "grub.cfg"), []byte(cfg), 0o644)
}

func SetPendingBoot(rootA string, active, pending Slot) error {
	if !validSlot(active) || !validSlot(pending) || active == pending {
		return errors.New("invalid active/pending boot slots")
	}
	return setGRUBEnv(rootA, map[string]string{
		"kingai_active": string(active),
		"kingai_pending": string(pending),
		"kingai_attempted": "0",
	})
}

func ConfirmBoot(rootA string, active Slot) error {
	if !validSlot(active) { return errors.New("invalid confirmed boot slot") }
	if err := setGRUBEnv(rootA, map[string]string{"kingai_active": string(active)}); err != nil { return err }
	return unsetGRUBEnv(rootA, "kingai_pending", "kingai_attempted")
}

func RollbackBoot(rootA string, active Slot) error {
	if !validSlot(active) { return errors.New("invalid rollback boot slot") }
	if err := setGRUBEnv(rootA, map[string]string{"kingai_active": string(active)}); err != nil { return err }
	return unsetGRUBEnv(rootA, "kingai_pending", "kingai_attempted")
}

func ReadBootEnv(rootA string) (map[string]string, error) {
	envPath := filepath.Join(rootA, "boot/grub/grubenv")
	out, err := exec.Command("grub-editenv", envPath, "list").CombinedOutput()
	if err != nil { return nil, fmt.Errorf("grub-editenv list: %w: %s", err, strings.TrimSpace(string(out))) }
	result := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 { result[parts[0]] = parts[1] }
	}
	return result, nil
}

func setGRUBEnv(rootA string, values map[string]string) error {
	envPath := filepath.Join(rootA, "boot/grub/grubenv")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil { return err }
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if out, e := exec.Command("grub-editenv", envPath, "create").CombinedOutput(); e != nil {
			return fmt.Errorf("grub-editenv create: %w: %s", e, strings.TrimSpace(string(out)))
		}
	}
	args := []string{envPath, "set"}
	for k, v := range values {
		if strings.ContainsAny(k+v, "\n\r\x00") { return errors.New("invalid grub environment value") }
		args = append(args, k+"="+v)
	}
	if out, err := exec.Command("grub-editenv", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("grub-editenv set: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func unsetGRUBEnv(rootA string, keys ...string) error {
	envPath := filepath.Join(rootA, "boot/grub/grubenv")
	args := append([]string{envPath, "unset"}, keys...)
	if out, err := exec.Command("grub-editenv", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("grub-editenv unset: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func newestBootFile(root, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "boot", pattern))
	if err != nil { return "", err }
	if len(matches) == 0 { return "", fmt.Errorf("no boot file matches %s", pattern) }
	best := matches[0]
	for _, p := range matches[1:] { if p > best { best = p } }
	return best, nil
}
