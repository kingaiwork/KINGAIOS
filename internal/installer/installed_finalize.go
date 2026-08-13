package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FinalizeInstalledSystem applies installed-system-only policy after the root
// filesystem copies are complete. It intentionally does not weaken the
// production STATE trust model: plaintext key persistence remains forbidden
// outside the explicit disposable-NBD CI path.
func FinalizeInstalledSystem(res InstallResult) error {
	if res.RootAPart == "" || res.RootBPart == "" || res.StateLUKSUUID == "" {
		return errors.New("installer result is missing installed-system identity")
	}
	for _, rootPart := range []string{res.RootAPart, res.RootBPart} {
		if err := finalizeInstalledRoot(rootPart, res); err != nil {
			return err
		}
	}
	return nil
}

func finalizeInstalledRoot(rootPart string, res InstallResult) error {
	st, err := os.Stat(rootPart)
	if err != nil {
		return fmt.Errorf("installed root stat %s: %w", rootPart, err)
	}
	if st.Mode()&os.ModeDevice == 0 {
		return fmt.Errorf("installed root is not a block device: %s", rootPart)
	}

	mnt, err := os.MkdirTemp("", "kingai-installed-root-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mnt)
	if err := runBootTool("mount", rootPart, mnt); err != nil {
		return err
	}
	mounted := true
	defer func() {
		if mounted {
			_ = runBootTool("umount", mnt)
		}
	}()

	// A copied Live root must not run media-integrity services after installation.
	systemDir := filepath.Join(mnt, "etc/systemd/system")
	if err := os.MkdirAll(systemDir, 0o755); err != nil {
		return err
	}
	for _, liveUnit := range []string{"casper-md5check.service"} {
		p := filepath.Join(systemDir, liveUnit)
		_ = os.Remove(p)
		if err := os.Symlink("/dev/null", p); err != nil {
			return fmt.Errorf("mask installed-system live unit %s: %w", liveUnit, err)
		}
	}

	// Keep Server headless by default; Desktop keeps the graphical target that
	// was established by the Desktop rootfs builder.
	if res.Profile == "server" {
		defaultTarget := filepath.Join(systemDir, "default.target")
		_ = os.Remove(defaultTarget)
		if err := os.Symlink("/usr/lib/systemd/system/multi-user.target", defaultTarget); err != nil {
			return fmt.Errorf("set server default target: %w", err)
		}
	}

	if ciStateUnlockAllowed(res) {
		if err := installCIStateUnlock(mnt, res.StateLUKSUUID); err != nil {
			return err
		}
	}

	if err := runBootTool("sync"); err != nil {
		return err
	}
	if err := runBootTool("umount", mnt); err != nil {
		return err
	}
	mounted = false
	return nil
}

func ciStateUnlockAllowed(res InstallResult) bool {
	return os.Getenv("KINGAI_INSTALLER_CI") == "1" &&
		os.Getenv("KINGAI_INSTALLER_TEST_PERSIST_KEY") == "1" &&
		strings.HasPrefix(res.Target, "/dev/nbd")
}

func installCIStateUnlock(root, luksUUID string) error {
	key := filepath.Join(root, "etc/kingai/state-test.key")
	info, err := os.Stat(key)
	if err != nil {
		return fmt.Errorf("CI STATE key missing: %w", err)
	}
	if info.Mode().Perm() != 0o600 || info.Size() < 32 {
		return errors.New("CI STATE key does not meet private-key fixture requirements")
	}
	if strings.ContainsAny(luksUUID, " \t\n/'\"") {
		return errors.New("invalid LUKS UUID in installer result")
	}

	unitPath := filepath.Join(root, "usr/lib/systemd/system/kingai-state-unlock-ci.service")
	unit := fmt.Sprintf(`[Unit]
Description=KINGAI CI-only encrypted STATE unlock
DefaultDependencies=no
After=systemd-udev-trigger.service
Before=local-fs-pre.target
ConditionPathExists=/etc/kingai/state-test.key

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStartPre=/usr/bin/udevadm settle
ExecStart=/bin/sh -c 'test -e /dev/mapper/KINGAI_STATE || /usr/sbin/cryptsetup open --type luks --key-file /etc/kingai/state-test.key /dev/disk/by-uuid/%s KINGAI_STATE'
ExecStop=/bin/sh -c 'test ! -e /dev/mapper/KINGAI_STATE || /usr/sbin/cryptsetup close KINGAI_STATE'

[Install]
WantedBy=local-fs-pre.target
`, luksUUID)
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("write CI STATE unlock unit: %w", err)
	}
	wants := filepath.Join(root, "etc/systemd/system/local-fs-pre.target.wants")
	if err := os.MkdirAll(wants, 0o755); err != nil {
		return err
	}
	link := filepath.Join(wants, "kingai-state-unlock-ci.service")
	_ = os.Remove(link)
	if err := os.Symlink("/usr/lib/systemd/system/kingai-state-unlock-ci.service", link); err != nil {
		return fmt.Errorf("enable CI STATE unlock unit: %w", err)
	}
	return nil
}
