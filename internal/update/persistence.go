package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	stateRuntimeMarkerRel = "kingai/runtime/.layout-v1-ready"
	stateRuntimeLibRel    = "kingai/runtime/lib"
	stateRuntimeLogRel    = "kingai/runtime/log"
	migrationConfigRel    = "etc/kingai/state-migration.conf"
	migrationUnitRel      = "usr/lib/systemd/system/kingai-state-migrate.service"
	migrationDropInRel    = "usr/lib/systemd/system/kingaid.service.d/05-state-migrate.conf"
)

func prepareTargetRuntimePersistence(targetRoot, stateRoot string, activeSlot Slot, rootAUUID, rootBUUID string) error {
	for _, rel := range []string{"kingai/update", stateRuntimeLibRel, stateRuntimeLogRel} {
		p := filepath.Join(stateRoot, rel)
		if err := os.MkdirAll(p, 0o700); err != nil {
			return fmt.Errorf("create encrypted STATE path %s: %w", p, err)
		}
		if err := os.Chmod(p, 0o700); err != nil {
			return fmt.Errorf("harden encrypted STATE path %s: %w", p, err)
		}
	}
	for _, rel := range []string{"var/lib/kingai-state", "var/lib/kingai", "var/log/kingai"} {
		if err := os.MkdirAll(filepath.Join(targetRoot, rel), 0o700); err != nil {
			return err
		}
	}
	if err := rewriteRuntimeFstab(targetRoot); err != nil {
		return err
	}

	marker := filepath.Join(stateRoot, stateRuntimeMarkerRel)
	if _, err := os.Stat(marker); err == nil {
		return clearMigrationBootGuard(targetRoot)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	sourceUUID := rootAUUID
	if activeSlot == SlotB {
		sourceUUID = rootBUUID
	}
	if sourceUUID == "" {
		return errors.New("legacy STATE migration source UUID is empty")
	}
	return installMigrationBootGuard(targetRoot, sourceUUID, activeSlot)
}

func rewriteRuntimeFstab(root string) error {
	path := filepath.Join(root, "etc/fstab")
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read target fstab: %w", err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			switch fields[1] {
			case "/var/lib/kingai-state", "/var/lib/kingai", "/var/log/kingai":
				continue
			}
		}
		out = append(out, line)
	}
	out = append(out,
		"/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults 0 2",
		"/var/lib/kingai-state/kingai/runtime/lib /var/lib/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
		"/var/lib/kingai-state/kingai/runtime/log /var/log/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
	)
	return atomicWriteDurable(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func installMigrationBootGuard(root, sourceUUID string, sourceSlot Slot) error {
	if sourceSlot != SlotA && sourceSlot != SlotB {
		return fmt.Errorf("invalid migration source slot %q", sourceSlot)
	}
	conf := fmt.Sprintf("SOURCE_ROOT_UUID=%s\nSOURCE_SLOT=%s\n", sourceUUID, sourceSlot)
	unit := `[Unit]
Description=KINGAI OS Encrypted Runtime State Migration
After=local-fs.target
Before=kingaid.service
ConditionPathExists=/etc/kingai/state-migration.conf

[Service]
Type=oneshot
ExecStart=/usr/lib/kingai/kingai-update migrate-state
TimeoutStartSec=5min
UMask=0077
NoNewPrivileges=yes
PrivateTmp=yes
ProtectHome=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
ProtectClock=yes
ProtectHostname=yes
RestrictSUIDSGID=yes
RestrictRealtime=yes
LockPersonality=yes
SystemCallArchitectures=native
CapabilityBoundingSet=CAP_SYS_ADMIN CAP_DAC_OVERRIDE CAP_FOWNER CAP_CHOWN
AmbientCapabilities=
`
	dropIn := `[Unit]
Requires=kingai-state-migrate.service
After=kingai-state-migrate.service
`
	for _, spec := range []struct {
		rel  string
		data []byte
		mode os.FileMode
	}{
		{migrationConfigRel, []byte(conf), 0o600},
		{migrationUnitRel, []byte(unit), 0o644},
		{migrationDropInRel, []byte(dropIn), 0o644},
	} {
		path := filepath.Join(root, spec.rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := atomicWriteDurable(path, spec.data, spec.mode); err != nil {
			return err
		}
	}
	return nil
}

func clearMigrationBootGuard(root string) error {
	for _, rel := range []string{migrationConfigRel, migrationUnitRel, migrationDropInRel} {
		path := filepath.Join(root, rel)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func MigrateRuntimeState() error {
	if os.Geteuid() != 0 {
		return errors.New("STATE migration requires root")
	}
	const stateRoot = "/var/lib/kingai-state"
	marker := filepath.Join(stateRoot, stateRuntimeMarkerRel)
	confPath := "/etc/kingai/state-migration.conf"
	if _, err := os.Stat(marker); err == nil {
		if err := os.Remove(confPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	for _, p := range []string{stateRoot, "/var/lib/kingai", "/var/log/kingai"} {
		ok, err := exactMountpoint(p)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("required encrypted runtime mount is not active: %s", p)
		}
	}
	b, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read migration metadata: %w", err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || (k != "SOURCE_ROOT_UUID" && k != "SOURCE_SLOT") {
			return errors.New("invalid STATE migration metadata")
		}
		values[k] = v
	}
	uuid := values["SOURCE_ROOT_UUID"]
	slot := Slot(values["SOURCE_SLOT"])
	if uuid == "" || strings.Trim(uuid, "0123456789abcdefABCDEF-") != "" {
		return errors.New("invalid migration source root UUID")
	}
	if slot != SlotA && slot != SlotB {
		return errors.New("invalid migration source slot")
	}
	sourceDevice, err := commandOutput("blkid", "-U", uuid)
	if err != nil || sourceDevice == "" {
		return errors.New("migration source root device cannot be resolved")
	}
	if st, err := os.Stat(sourceDevice); err != nil || st.Mode()&os.ModeDevice == 0 {
		return errors.New("migration source root is not a block device")
	}
	if current, err := commandOutput("findmnt", "-rn", "-T", "/", "-o", "SOURCE"); err == nil {
		if sameDevice(current, sourceDevice) {
			return errors.New("migration source unexpectedly resolves to the current root")
		}
	}

	work := "/run/kingai-state-migrate"
	sourceMount := filepath.Join(work, "source")
	if err := os.MkdirAll(sourceMount, 0o700); err != nil {
		return err
	}
	if err := run("mount", "-o", "ro", sourceDevice, sourceMount); err != nil {
		return err
	}
	mounted := true
	defer func() {
		if mounted {
			_ = run("umount", sourceMount)
		}
		_ = os.RemoveAll(work)
	}()
	identity, err := os.ReadFile(filepath.Join(sourceMount, "usr/lib/os-release"))
	if err != nil || !strings.Contains(string(identity), `NAME="KINGAI OS"`) {
		return errors.New("migration source is not a KINGAI OS root")
	}
	for _, pair := range [][2]string{
		{filepath.Join(sourceMount, "var/lib/kingai"), "/var/lib/kingai"},
		{filepath.Join(sourceMount, "var/log/kingai"), "/var/log/kingai"},
	} {
		if st, err := os.Stat(pair[0]); err == nil && st.IsDir() {
			if err := run("rsync", "-aHAX", "--numeric-ids", filepath.Clean(pair[0])+"/", filepath.Clean(pair[1])+"/"); err != nil {
				return err
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := run("sync"); err != nil {
		return err
	}
	markerData := []byte(fmt.Sprintf("layout=1\nsource_slot=%s\nsource_root_uuid=%s\n", slot, uuid))
	if err := atomicWriteDurable(marker, markerData, 0o600); err != nil {
		return err
	}
	if err := os.Remove(confPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncDir(filepath.Dir(confPath)); err != nil {
		return err
	}
	if err := run("umount", sourceMount); err != nil {
		return err
	}
	mounted = false
	return nil
}

func exactMountpoint(path string) (bool, error) {
	out, err := commandOutput("findmnt", "-rn", "-T", path, "-o", "TARGET")
	if err != nil {
		return false, err
	}
	return filepath.Clean(out) == filepath.Clean(path), nil
}

func sameDevice(a, b string) bool {
	ar, errA := filepath.EvalSymlinks(a)
	br, errB := filepath.EvalSymlinks(b)
	if errA == nil && errB == nil {
		return ar == br
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func atomicWriteDurable(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kingai-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func syncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
