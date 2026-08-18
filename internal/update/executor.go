package update

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
)

type ExecuteOptions struct {
	TargetDisk    string
	SourceRoot    string
	StateKey      string
	TargetVersion string
	Confirmation  string
}

type ExecuteResult struct {
	TargetDisk    string    `json:"target_disk"`
	FromSlot      Slot      `json:"from_slot"`
	TargetSlot    Slot      `json:"target_slot"`
	TargetVersion string    `json:"target_version"`
	CompletedAt   time.Time `json:"completed_at"`
}

func ExecuteStage(opts ExecuteOptions) (ExecuteResult, error) {
	if runtime.GOARCH != "amd64" {
		return ExecuteResult{}, errors.New("A/B update execution is currently reviewed only for amd64")
	}
	if os.Geteuid() != 0 {
		return ExecuteResult{}, errors.New("A/B update execution requires root")
	}
	if os.Getenv("KINGAI_UPDATE_ALLOW_WRITE") != "1" {
		return ExecuteResult{}, errors.New("A/B update writes are disabled; set KINGAI_UPDATE_ALLOW_WRITE=1 explicitly")
	}
	if opts.TargetDisk == "" || opts.SourceRoot == "" || opts.StateKey == "" || opts.TargetVersion == "" {
		return ExecuteResult{}, errors.New("target-disk, source-root, state-key and target-version are required")
	}
	if opts.Confirmation != "UPDATE:"+opts.TargetDisk {
		return ExecuteResult{}, errors.New("confirmation mismatch; expected UPDATE:<exact target disk>")
	}
	if os.Getenv("KINGAI_UPDATE_CI") == "1" && !strings.HasPrefix(opts.TargetDisk, "/dev/nbd") {
		return ExecuteResult{}, errors.New("CI update execution is restricted to disposable /dev/nbd devices")
	}
	if err := validateSourceRoot(opts.SourceRoot, opts.TargetVersion); err != nil {
		return ExecuteResult{}, err
	}
	if err := validatePrivateKey(opts.StateKey); err != nil {
		return ExecuteResult{}, err
	}
	for _, tool := range []string{"blkid", "cryptsetup", "mount", "umount", "rsync", "grub-editenv", "findmnt", "sync"} {
		if _, err := exec.LookPath(tool); err != nil {
			return ExecuteResult{}, fmt.Errorf("required update tool missing: %s", tool)
		}
	}

	parts := []string{
		partitionPath(opts.TargetDisk, 1),
		partitionPath(opts.TargetDisk, 2),
		partitionPath(opts.TargetDisk, 3),
		partitionPath(opts.TargetDisk, 4),
	}
	labels := []string{"KINGAI_EFI", "KINGAI_ROOT_A", "KINGAI_ROOT_B"}
	for i, want := range labels {
		got, err := commandOutput("blkid", "-s", "LABEL", "-o", "value", parts[i])
		if err != nil || got != want {
			return ExecuteResult{}, fmt.Errorf("partition %s is not %s", parts[i], want)
		}
	}
	if err := exec.Command("cryptsetup", "isLuks", parts[3]).Run(); err != nil {
		return ExecuteResult{}, errors.New("KINGAI_STATE partition is not LUKS")
	}

	work, err := os.MkdirTemp("", "kingai-update-")
	if err != nil {
		return ExecuteResult{}, err
	}
	defer os.RemoveAll(work)
	rootA := filepath.Join(work, "root-a")
	rootB := filepath.Join(work, "root-b")
	stateMnt := filepath.Join(work, "state")
	for _, p := range []string{rootA, rootB, stateMnt} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return ExecuteResult{}, err
		}
	}
	mapper := fmt.Sprintf("kingai-state-update-%d", os.Getpid())
	mapperPath := "/dev/mapper/" + mapper
	if err := run("cryptsetup", "open", "--key-file", opts.StateKey, parts[3], mapper); err != nil {
		return ExecuteResult{}, err
	}
	mapperOpen := true
	mounted := []string{}
	defer func() {
		for i := len(mounted) - 1; i >= 0; i-- {
			_ = exec.Command("umount", mounted[i]).Run()
		}
		if mapperOpen {
			_ = exec.Command("cryptsetup", "close", mapper).Run()
		}
	}()
	if err := run("mount", parts[1], rootA); err != nil {
		return ExecuteResult{}, err
	}
	mounted = append(mounted, rootA)
	if err := run("mount", parts[2], rootB); err != nil {
		return ExecuteResult{}, err
	}
	mounted = append(mounted, rootB)
	if err := run("mount", mapperPath, stateMnt); err != nil {
		return ExecuteResult{}, err
	}
	mounted = append(mounted, stateMnt)

	statePath := filepath.Join(stateMnt, "kingai/update/slots.json")
	state, err := loadSlotState(statePath)
	if err != nil {
		return ExecuteResult{}, err
	}
	plan, err := state.PlanStage(opts.TargetVersion)
	if err != nil {
		return ExecuteResult{}, err
	}
	activeRoot := rootA
	if state.ActiveSlot == SlotB {
		activeRoot = rootB
	}
	targetPart := parts[2]
	targetRoot := rootB
	if plan.TargetSlot == SlotA {
		targetPart, targetRoot = parts[1], rootA
	}
	if filepath.Clean(activeRoot) == filepath.Clean(targetRoot) {
		return ExecuteResult{}, errors.New("A/B update target unexpectedly equals active root")
	}
	if isMountedElsewhere(targetPart, targetRoot) {
		return ExecuteResult{}, fmt.Errorf("inactive target slot %s is mounted elsewhere", plan.TargetSlot)
	}

	args := []string{
		"-aHAX", "--numeric-ids", "--delete",
		"--exclude=/dev/*", "--exclude=/proc/*", "--exclude=/sys/*", "--exclude=/run/*", "--exclude=/tmp/*", "--exclude=/mnt/*", "--exclude=/media/*", "--exclude=/boot/efi/*",
		"--exclude=/home/*",
		"--exclude=/etc/fstab", "--exclude=/etc/crypttab", "--exclude=/etc/machine-id", "--exclude=/etc/kingai/state-test.key",
		"--exclude=/usr/lib/systemd/system/kingai-state-unlock-ci.service", "--exclude=/etc/systemd/system/local-fs-pre.target.wants/kingai-state-unlock-ci.service",
		"--exclude=/etc/systemd/system/casper-md5check.service", "--exclude=/etc/systemd/system/default.target",
	}
	if plan.TargetSlot == SlotA {
		args = append(args, "--exclude=/boot/grub/*")
	}
	args = append(args, filepath.Clean(opts.SourceRoot)+"/", filepath.Clean(targetRoot)+"/")
	if err := run("rsync", args...); err != nil {
		return ExecuteResult{}, err
	}

	aUUID, err := commandOutput("blkid", "-s", "UUID", "-o", "value", parts[1])
	if err != nil {
		return ExecuteResult{}, err
	}
	bUUID, err := commandOutput("blkid", "-s", "UUID", "-o", "value", parts[2])
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := prepareTargetRuntimePersistence(targetRoot, activeRoot, stateMnt, state.ActiveSlot, aUUID, bUUID); err != nil {
		return ExecuteResult{}, fmt.Errorf("prepare encrypted runtime persistence: %w", err)
	}

	controller := BootController{RootAPath: rootA, RootBPath: rootB, RootAUUID: aUUID, RootBUUID: bUUID}
	if err := controller.WriteConfig(); err != nil {
		return ExecuteResult{}, err
	}
	pending, err := state.MarkPending(opts.TargetVersion)
	if err != nil {
		return ExecuteResult{}, err
	}
	if err := SetPendingBoot(rootA, state.ActiveSlot, plan.TargetSlot); err != nil {
		return ExecuteResult{}, err
	}
	if err := saveSlotState(statePath, pending); err != nil {
		return ExecuteResult{}, err
	}
	if err := run("sync"); err != nil {
		return ExecuteResult{}, err
	}
	mapperOpen = false
	for i := len(mounted) - 1; i >= 0; i-- {
		if err := run("umount", mounted[i]); err != nil {
			return ExecuteResult{}, err
		}
	}
	mounted = nil
	if err := run("cryptsetup", "close", mapper); err != nil {
		return ExecuteResult{}, err
	}
	return ExecuteResult{
		TargetDisk:    opts.TargetDisk,
		FromSlot:      state.ActiveSlot,
		TargetSlot:    plan.TargetSlot,
		TargetVersion: opts.TargetVersion,
		CompletedAt:   time.Now().UTC(),
	}, nil
}

func loadSlotState(path string) (SlotState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return SlotState{}, fmt.Errorf("read slot state: %w", err)
	}
	var s SlotState
	if err := json.Unmarshal(b, &s); err != nil {
		return SlotState{}, fmt.Errorf("decode slot state: %w", err)
	}
	if err := s.Validate(); err != nil {
		return SlotState{}, err
	}
	return s, nil
}

func saveSlotState(path string, s SlotState) error {
	if err := s.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := atomicWriteDurable(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("durably persist slot state: %w", err)
	}
	return nil
}

func validateSourceRoot(root, targetVersion string) error {
	b, err := os.ReadFile(filepath.Join(root, "usr/lib/os-release"))
	if err != nil {
		return errors.New("source root is not a KINGAI OS root filesystem")
	}
	if !strings.Contains(string(b), `NAME="KINGAI OS"`) {
		return errors.New("source root product identity is not KINGAI OS")
	}
	if !strings.Contains(string(b), "VERSION_ID=\""+targetVersion+"\"") && !strings.Contains(string(b), "VERSION_ID="+targetVersion) {
		return errors.New("source root VERSION_ID does not match target-version")
	}
	return nil
}

func validatePrivateKey(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() < 32 {
		return errors.New("STATE key must be a regular file of at least 32 bytes")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return errors.New("STATE key must be private (0600 or stricter)")
	}
	return nil
}

func partitionPath(target string, n int) string {
	if target == "" {
		return ""
	}
	last := target[len(target)-1]
	if last >= '0' && last <= '9' {
		return fmt.Sprintf("%sp%d", target, n)
	}
	return fmt.Sprintf("%s%d", target, n)
}

func commandOutput(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func isMountedElsewhere(device, expected string) bool {
	out, err := commandOutput("findmnt", "-rn", "-S", device, "-o", "TARGET")
	if err != nil {
		return false
	}
	for _, p := range strings.Fields(out) {
		if filepath.Clean(p) != filepath.Clean(expected) {
			return true
		}
	}
	return false
}
