package installer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

type ExecuteOptions struct {
	Target       string
	Profile      string
	SourceRoot   string
	StateKey     string
	Confirmation string
}

type InstallResult struct {
	Target        string    `json:"target"`
	Profile       string    `json:"profile"`
	EFIPart       string    `json:"efi_partition"`
	RootAPart     string    `json:"root_a_partition"`
	RootBPart     string    `json:"root_b_partition"`
	StatePart     string    `json:"state_partition"`
	RootAUUID     string    `json:"root_a_uuid"`
	RootBUUID     string    `json:"root_b_uuid"`
	EFIUUID       string    `json:"efi_uuid"`
	StateLUKSUUID string    `json:"state_luks_uuid"`
	ActiveSlot    string    `json:"active_slot"`
	CompletedAt   time.Time `json:"completed_at"`
}

type commandRunner interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
}

type osRunner struct{}

func (osRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (osRunner) Output(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}

func Execute(opts ExecuteOptions) (InstallResult, error) {
	if runtime.GOARCH != "amd64" {
		return InstallResult{}, errors.New("destructive installer execution is currently reviewed only for amd64 UEFI systems")
	}
	if opts.Profile != "server" && opts.Profile != "desktop" {
		return InstallResult{}, errors.New("destructive installer execution is currently enabled only for server and desktop profiles")
	}
	if os.Geteuid() != 0 {
		return InstallResult{}, errors.New("installer execute requires root")
	}
	if os.Getenv("KINGAI_INSTALLER_ALLOW_WRITE") != "1" {
		return InstallResult{}, errors.New("disk writing is disabled; set KINGAI_INSTALLER_ALLOW_WRITE=1 explicitly")
	}
	if opts.Confirmation != "ERASE:"+opts.Target {
		return InstallResult{}, errors.New("confirmation mismatch; expected ERASE:<exact target>")
	}
	devs, err := Discover()
	if err != nil {
		return InstallResult{}, err
	}
	plan, err := BuildPlan(devs, opts.Target, opts.Profile)
	if err != nil {
		return InstallResult{}, err
	}
	if err := validateExecutionInputs(opts); err != nil {
		return InstallResult{}, err
	}
	return executePlan(osRunner{}, plan, opts)
}

func validateExecutionInputs(opts ExecuteOptions) error {
	if opts.Target == "" || opts.SourceRoot == "" || opts.StateKey == "" {
		return errors.New("target, source-root and state-key are required")
	}
	st, err := os.Stat(opts.Target)
	if err != nil {
		return fmt.Errorf("target stat: %w", err)
	}
	if st.Mode()&os.ModeDevice == 0 {
		return errors.New("target must be a block device")
	}
	root, err := filepath.EvalSymlinks(opts.SourceRoot)
	if err != nil {
		return fmt.Errorf("source root: %w", err)
	}
	b, err := os.ReadFile(filepath.Join(root, "usr/lib/os-release"))
	if err != nil {
		return errors.New("source root is not a KINGAI OS root filesystem")
	}
	if !strings.Contains(string(b), `NAME="KINGAI OS"`) {
		return errors.New("source root product identity is not KINGAI OS")
	}
	keyInfo, err := os.Stat(opts.StateKey)
	if err != nil {
		return fmt.Errorf("state key: %w", err)
	}
	if !keyInfo.Mode().IsRegular() || keyInfo.Size() < 32 {
		return errors.New("state key must be a regular file of at least 32 bytes")
	}
	if keyInfo.Mode().Perm()&0o077 != 0 {
		return errors.New("state key permissions must not grant group/other access")
	}
	for _, tool := range []string{"sgdisk", "partprobe", "udevadm", "wipefs", "mkfs.vfat", "mkfs.ext4", "cryptsetup", "rsync", "mount", "umount", "blkid", "grub-install", "grub-editenv", "sync"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("required installer tool missing: %s", tool)
		}
	}
	return nil
}

func executePlan(r commandRunner, plan Plan, opts ExecuteOptions) (res InstallResult, retErr error) {
	parts := []string{partitionPath(plan.Target, 1), partitionPath(plan.Target, 2), partitionPath(plan.Target, 3), partitionPath(plan.Target, 4)}
	mapper := "kingai-state-installer-" + strconv.Itoa(os.Getpid())
	mapperPath := "/dev/mapper/" + mapper
	mnt, err := os.MkdirTemp("", "kingai-install-")
	if err != nil {
		return res, err
	}
	rootA, rootB, stateMnt := filepath.Join(mnt, "root-a"), filepath.Join(mnt, "root-b"), filepath.Join(mnt, "state")
	for _, p := range []string{rootA, rootB, stateMnt} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return res, err
		}
	}
	mapperOpen := false
	mounted := []string{}
	defer func() {
		for i := len(mounted) - 1; i >= 0; i-- {
			_ = r.Run("umount", mounted[i])
		}
		if mapperOpen {
			_ = r.Run("cryptsetup", "close", mapper)
		}
		_ = os.RemoveAll(mnt)
	}()

	rootMiB := plan.Partitions[1].SizeBytes / MiB
	if err := r.Run("wipefs", "-a", plan.Target); err != nil {
		return res, err
	}
	if err := r.Run("sgdisk", "--zap-all", plan.Target); err != nil {
		return res, err
	}
	if err := r.Run("sgdisk", "--clear",
		"-n1:1MiB:+512MiB", "-t1:EF00", "-c1:KINGAI_EFI",
		fmt.Sprintf("-n2:0:+%dMiB", rootMiB), "-t2:8304", "-c2:KINGAI_ROOT_A",
		fmt.Sprintf("-n3:0:+%dMiB", rootMiB), "-t3:8304", "-c3:KINGAI_ROOT_B",
		"-n4:0:0", "-t4:8309", "-c4:KINGAI_STATE", plan.Target); err != nil {
		return res, err
	}
	if err := r.Run("partprobe", plan.Target); err != nil {
		return res, err
	}
	if err := r.Run("udevadm", "settle"); err != nil {
		return res, err
	}
	for _, p := range parts {
		if _, err := os.Stat(p); err != nil {
			return res, fmt.Errorf("partition missing after GPT creation: %s", p)
		}
	}

	if err := r.Run("mkfs.vfat", "-F", "32", "-n", "KINGAI_EFI", parts[0]); err != nil {
		return res, err
	}
	if err := r.Run("mkfs.ext4", "-F", "-L", "KINGAI_ROOT_A", parts[1]); err != nil {
		return res, err
	}
	if err := r.Run("mkfs.ext4", "-F", "-L", "KINGAI_ROOT_B", parts[2]); err != nil {
		return res, err
	}
	if err := r.Run("cryptsetup", "luksFormat", "--type", "luks2", "--batch-mode", "--key-file", opts.StateKey, parts[3]); err != nil {
		return res, err
	}
	if err := r.Run("cryptsetup", "open", "--key-file", opts.StateKey, parts[3], mapper); err != nil {
		return res, err
	}
	mapperOpen = true
	if err := r.Run("mkfs.ext4", "-F", "-L", "KINGAI_STATE", mapperPath); err != nil {
		return res, err
	}

	if err := r.Run("mount", parts[1], rootA); err != nil {
		return res, err
	}
	mounted = append(mounted, rootA)
	if err := r.Run("mount", parts[2], rootB); err != nil {
		return res, err
	}
	mounted = append(mounted, rootB)
	if err := r.Run("mount", mapperPath, stateMnt); err != nil {
		return res, err
	}
	mounted = append(mounted, stateMnt)
	for _, dst := range []string{rootA, rootB} {
		if err := copyRoot(r, opts.SourceRoot, dst); err != nil {
			return res, err
		}
	}
	if err := preparePersistentStateLayout(rootA, rootB, stateMnt); err != nil {
		return res, err
	}
	for _, dst := range []string{rootA, rootB} {
		if err := os.MkdirAll(filepath.Join(dst, "boot/efi"), 0o755); err != nil {
			return res, err
		}
	}
	if err := r.Run("mount", parts[0], filepath.Join(rootA, "boot/efi")); err != nil {
		return res, err
	}
	mounted = append(mounted, filepath.Join(rootA, "boot/efi"))

	efiUUID, err := r.Output("blkid", "-s", "UUID", "-o", "value", parts[0])
	if err != nil {
		return res, err
	}
	aUUID, err := r.Output("blkid", "-s", "UUID", "-o", "value", parts[1])
	if err != nil {
		return res, err
	}
	bUUID, err := r.Output("blkid", "-s", "UUID", "-o", "value", parts[2])
	if err != nil {
		return res, err
	}
	luksUUID, err := r.Output("blkid", "-s", "UUID", "-o", "value", parts[3])
	if err != nil {
		return res, err
	}

	persistKey := false
	if os.Getenv("KINGAI_INSTALLER_TEST_PERSIST_KEY") == "1" {
		if os.Getenv("KINGAI_INSTALLER_CI") != "1" || !strings.HasPrefix(plan.Target, "/dev/nbd") {
			return res, errors.New("test STATE-key persistence is restricted to explicit CI mode on /dev/nbd devices")
		}
		persistKey = true
	}
	for idx, dst := range []string{rootA, rootB} {
		rootUUID := aUUID
		if idx == 1 {
			rootUUID = bUUID
		}
		if err := writeInstalledConfig(dst, rootUUID, efiUUID, luksUUID, persistKey, opts.StateKey); err != nil {
			return res, err
		}
	}
	slotState, err := kingupdate.NewSlotState(kingupdate.SlotA, installedVersion(rootA))
	if err != nil {
		return res, fmt.Errorf("initialize A/B slot state: %w", err)
	}
	if err := kingupdate.SaveSlotStateFile(filepath.Join(stateMnt, "kingai/update/slots.json"), slotState); err != nil {
		return res, fmt.Errorf("durably initialize A/B slot state: %w", err)
	}

	if err := installGRUB(r, rootA, rootB, aUUID, bUUID); err != nil {
		return res, err
	}
	if err := r.Run("sync"); err != nil {
		return res, err
	}
	res = InstallResult{Target: plan.Target, Profile: plan.Profile, EFIPart: parts[0], RootAPart: parts[1], RootBPart: parts[2], StatePart: parts[3], RootAUUID: aUUID, RootBUUID: bUUID, EFIUUID: efiUUID, StateLUKSUUID: luksUUID, ActiveSlot: "A", CompletedAt: time.Now().UTC()}
	return res, nil
}

func copyRoot(r commandRunner, src, dst string) error {
	return r.Run("rsync", "-aHAX", "--numeric-ids", "--delete",
		"--exclude=/dev/*", "--exclude=/proc/*", "--exclude=/sys/*", "--exclude=/run/*", "--exclude=/tmp/*", "--exclude=/mnt/*", "--exclude=/media/*", "--exclude=/boot/efi/*",
		filepath.Clean(src)+"/", filepath.Clean(dst)+"/")
}

func preparePersistentStateLayout(rootA, rootB, stateRoot string) error {
	for _, p := range []string{
		filepath.Join(stateRoot, "kingai/update"),
		filepath.Join(stateRoot, "kingai/runtime/lib"),
		filepath.Join(stateRoot, "kingai/runtime/log"),
	} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			return fmt.Errorf("create encrypted STATE runtime directory %s: %w", p, err)
		}
		if err := os.Chmod(p, 0o700); err != nil {
			return fmt.Errorf("harden encrypted STATE runtime directory %s: %w", p, err)
		}
	}
	if err := writeFreshStateMarker(stateRoot); err != nil {
		return err
	}
	for _, root := range []string{rootA, rootB} {
		for _, rel := range []string{"var/lib/kingai-state", "var/lib/kingai", "var/log/kingai"} {
			p := filepath.Join(root, rel)
			if err := os.MkdirAll(p, 0o700); err != nil {
				return fmt.Errorf("create installed persistent mountpoint %s: %w", p, err)
			}
		}
	}
	return nil
}

func writeFreshStateMarker(stateRoot string) error {
	marker := filepath.Join(stateRoot, "kingai/runtime/.layout-v1-ready")
	f, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create encrypted STATE layout marker: %w", err)
	}
	if _, err := f.WriteString("layout=1\norigin=fresh-install\n"); err != nil {
		_ = f.Close()
		return fmt.Errorf("write encrypted STATE layout marker: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync encrypted STATE layout marker: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close encrypted STATE layout marker: %w", err)
	}
	d, err := os.Open(filepath.Dir(marker))
	if err != nil {
		return fmt.Errorf("open encrypted STATE marker directory: %w", err)
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return fmt.Errorf("sync encrypted STATE marker directory: %w", err)
	}
	return nil
}

func writeInstalledConfig(root, rootUUID, efiUUID, luksUUID string, persistKey bool, keyPath string) error {
	fstab := fmt.Sprintf(
		"UUID=%s / ext4 defaults,errors=remount-ro 0 1\n"+
			"UUID=%s /boot/efi vfat umask=0077 0 2\n"+
			"/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults 0 2\n"+
			"/var/lib/kingai-state/kingai/runtime/lib /var/lib/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0\n"+
			"/var/lib/kingai-state/kingai/runtime/log /var/log/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0\n",
		rootUUID, efiUUID,
	)
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte(fstab), 0o644); err != nil {
		return err
	}
	keySpec := "none"
	if persistKey {
		dst := filepath.Join(root, "etc/kingai/state-test.key")
		b, err := os.ReadFile(keyPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			return err
		}
		keySpec = "/etc/kingai/state-test.key"
	}
	crypttab := fmt.Sprintf("KINGAI_STATE UUID=%s %s luks\n", luksUUID, keySpec)
	if err := os.WriteFile(filepath.Join(root, "etc/crypttab"), []byte(crypttab), 0o600); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "var/lib/kingai-state"), 0o700); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(root, "etc/machine-id"))
	if err := os.WriteFile(filepath.Join(root, "etc/machine-id"), nil, 0o444); err != nil {
		return err
	}
	return nil
}

func installGRUB(r commandRunner, rootA, rootB, rootAUUID, rootBUUID string) error {
	efi := filepath.Join(rootA, "boot/efi")
	if err := r.Run("grub-install", "--target=x86_64-efi", "--efi-directory="+efi, "--boot-directory="+filepath.Join(rootA, "boot"), "--removable", "--no-nvram"); err != nil {
		return err
	}
	controller := kingupdate.BootController{
		RootAPath: rootA,
		RootBPath: rootB,
		RootAUUID: rootAUUID,
		RootBUUID: rootBUUID,
	}
	if err := controller.WriteConfig(); err != nil {
		return fmt.Errorf("write A/B boot controller: %w", err)
	}
	if err := kingupdate.ConfirmBoot(rootA, kingupdate.SlotA); err != nil {
		return fmt.Errorf("initialize A/B grub environment: %w", err)
	}
	return nil
}

func installedVersion(root string) string {
	b, err := os.ReadFile(filepath.Join(root, "usr/lib/os-release"))
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VERSION_ID=") {
			return strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	return "unknown"
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
