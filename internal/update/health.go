package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const DefaultStatePath = "/var/lib/kingai-state/kingai/update/slots.json"

// ReconcileHealthyBoot is intentionally called only after the core daemon and
// local filesystems are healthy. If the running root is the pending slot it
// confirms the update. If GRUB has already consumed a failed one-shot pending
// boot and returned to the previous active slot, it reconciles STATE to a
// rollback. No network access or cloud approval is involved.
func ReconcileHealthyBoot(statePath string) (string, error) {
	if os.Geteuid() != 0 { return "", errors.New("boot health reconciliation requires root") }
	if statePath == "" { statePath = DefaultStatePath }
	state, err := loadSlotState(statePath)
	if err != nil { return "", err }
	running, err := runningSlot()
	if err != nil { return "", err }
	rootADevice, err := deviceByLabel("KINGAI_ROOT_A")
	if err != nil { return "", err }
	rootA, cleanup, err := mountControllerRoot(rootADevice, running)
	if err != nil { return "", err }
	defer cleanup()
	env, err := ReadBootEnv(rootA)
	if err != nil { return "", err }

	if state.PendingSlot == "" {
		if running != state.ActiveSlot { return "", fmt.Errorf("running slot %s does not match confirmed active slot %s", running, state.ActiveSlot) }
		if env["kingai_active"] != string(state.ActiveSlot) {
			if err := ConfirmBoot(rootA, state.ActiveSlot); err != nil { return "", err }
		}
		return "healthy-confirmed", nil
	}
	if running == state.PendingSlot {
		confirmed, err := state.ConfirmPending()
		if err != nil { return "", err }
		if err := ConfirmBoot(rootA, confirmed.ActiveSlot); err != nil { return "", err }
		if err := saveSlotState(statePath, confirmed); err != nil { return "", err }
		return "update-confirmed", nil
	}
	if running == state.ActiveSlot && env["kingai_pending"] == "" {
		rolled, err := state.Rollback()
		if err != nil { return "", err }
		if err := RollbackBoot(rootA, state.ActiveSlot); err != nil { return "", err }
		if err := saveSlotState(statePath, rolled); err != nil { return "", err }
		return "update-rolled-back", nil
	}
	return "", fmt.Errorf("inconsistent A/B boot state: running=%s active=%s pending=%s grub_pending=%s", running, state.ActiveSlot, state.PendingSlot, env["kingai_pending"])
}

func runningSlot() (Slot, error) {
	out, err := exec.Command("findmnt", "-n", "-o", "SOURCE", "/").CombinedOutput()
	if err != nil { return "", fmt.Errorf("find running root: %w: %s", err, strings.TrimSpace(string(out))) }
	source := strings.TrimSpace(string(out))
	labelOut, err := exec.Command("blkid", "-s", "LABEL", "-o", "value", source).CombinedOutput()
	if err != nil { return "", fmt.Errorf("read running root label: %w: %s", err, strings.TrimSpace(string(labelOut))) }
	switch strings.TrimSpace(string(labelOut)) {
	case "KINGAI_ROOT_A": return SlotA, nil
	case "KINGAI_ROOT_B": return SlotB, nil
	default: return "", fmt.Errorf("running root is not a KINGAI A/B slot: %s", strings.TrimSpace(string(labelOut)))
	}
}

func deviceByLabel(label string) (string,error) {
	p := filepath.Join("/dev/disk/by-label", label)
	resolved,err:=filepath.EvalSymlinks(p);if err!=nil{return "",fmt.Errorf("resolve %s: %w",label,err)};return resolved,nil
}

func mountControllerRoot(rootADevice string, running Slot)(string,func(),error) {
	if running==SlotA { return "/", func(){}, nil }
	mnt,err:=os.MkdirTemp("","kingai-bootctrl-");if err!=nil{return "",nil,err}
	if out,e:=exec.Command("mount",rootADevice,mnt).CombinedOutput();e!=nil{os.RemoveAll(mnt);return "",nil,fmt.Errorf("mount boot controller: %w: %s",e,strings.TrimSpace(string(out)))}
	cleanup:=func(){_ = exec.Command("umount",mnt).Run();_ = os.RemoveAll(mnt)}
	return mnt,cleanup,nil
}
