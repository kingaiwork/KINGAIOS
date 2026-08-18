package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	stateRuntimeMarkerRel = "kingai/runtime/.layout-v1-ready"
	stateRuntimeLibRel    = "kingai/runtime/lib"
	stateRuntimeLogRel    = "kingai/runtime/log"
	stateHomeRel          = "home"
	migrationConfigRel    = "etc/kingai/state-migration.conf"
	migrationUnitRel      = "usr/lib/systemd/system/kingai-state-migrate.service"
	migrationDropInRel    = "usr/lib/systemd/system/kingaid.service.d/05-state-migrate.conf"
)

func prepareTargetRuntimePersistence(targetRoot, activeRoot, stateRoot string, activeSlot Slot, rootAUUID, rootBUUID string) error {
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

	persistentHome := false
	if st, err := os.Stat(filepath.Join(stateRoot, stateHomeRel)); err == nil && st.IsDir() {
		persistentHome = true
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect encrypted Desktop home: %w", err)
	}
	if persistentHome {
		if err := os.MkdirAll(filepath.Join(targetRoot, "home"), 0o755); err != nil {
			return err
		}
	}
	if err := rewriteRuntimeFstab(targetRoot, persistentHome); err != nil {
		return err
	}
	if persistentHome {
		if err := preserveDesktopIdentity(activeRoot, targetRoot, stateRoot); err != nil {
			return fmt.Errorf("preserve Desktop identity: %w", err)
		}
		// The installer entry belongs only to a Live medium. A fresh Desktop
		// source image contains it, so remove it again from the staged slot.
		if err := os.Remove(filepath.Join(targetRoot, "usr/share/applications/kingai-installer.desktop")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
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

func rewriteRuntimeFstab(root string, persistentHome bool) error {
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
			case "/var/lib/kingai-state", "/var/lib/kingai", "/var/log/kingai", "/home":
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
	if persistentHome {
		out = append(out, "/var/lib/kingai-state/home /home none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0")
	}
	return atomicWriteDurable(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

// preserveDesktopIdentity merges the installed human identity and machine-local
// settings from the active A/B root into a freshly staged Desktop root. The
// source image remains authoritative for system/service accounts; only UID 1000
// and its explicit memberships are carried forward.
func preserveDesktopIdentity(activeRoot, targetRoot, stateRoot string) error {
	if activeRoot == "" {
		return errors.New("active root is required for Desktop identity preservation")
	}
	passwdPath := filepath.Join(activeRoot, "etc/passwd")
	b, err := os.ReadFile(passwdPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	userLine, username, uid, gid, home, ok, err := humanPasswdRecord(string(b))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if home != "/home/"+username {
		return fmt.Errorf("unsupported Desktop home for %s: %s", username, home)
	}

	if err := mergePasswdRecord(filepath.Join(targetRoot, "etc/passwd"), userLine, username, uid); err != nil {
		return err
	}
	if err := copyNamedRecord(filepath.Join(activeRoot, "etc/shadow"), filepath.Join(targetRoot, "etc/shadow"), username, 0, 0o600, true); err != nil {
		return err
	}
	if err := mergeDesktopGroups(activeRoot, targetRoot, username, gid); err != nil {
		return err
	}
	for _, rel := range []string{"etc/subuid", "etc/subgid"} {
		if err := copyNamedRecord(filepath.Join(activeRoot, rel), filepath.Join(targetRoot, rel), username, 0, 0o644, false); err != nil {
			return err
		}
	}

	homeDir := filepath.Join(stateRoot, stateHomeRel, username)
	if err := os.MkdirAll(homeDir, 0o700); err != nil {
		return err
	}
	if err := os.Chown(homeDir, uid, gid); err != nil {
		return fmt.Errorf("own encrypted Desktop home: %w", err)
	}
	if err := os.Chmod(homeDir, 0o700); err != nil {
		return err
	}

	for _, rel := range []string{
		"etc/hostname",
		"etc/hosts",
		"etc/default/locale",
		"etc/timezone",
		"etc/localtime",
		"etc/machine-id",
	} {
		if err := copyPathIfPresent(filepath.Join(activeRoot, rel), filepath.Join(targetRoot, rel)); err != nil {
			return fmt.Errorf("preserve %s: %w", rel, err)
		}
	}
	for _, rel := range []string{
		"etc/NetworkManager/system-connections",
		"var/lib/bluetooth",
		"var/lib/fprint",
		"etc/cups/ppd",
	} {
		if err := replaceTreeIfPresent(filepath.Join(activeRoot, rel), filepath.Join(targetRoot, rel)); err != nil {
			return fmt.Errorf("preserve %s: %w", rel, err)
		}
	}
	if err := copyPathIfPresent(filepath.Join(activeRoot, "etc/cups/printers.conf"), filepath.Join(targetRoot, "etc/cups/printers.conf")); err != nil {
		return err
	}
	return nil
}

func humanPasswdRecord(passwd string) (line, username string, uid, gid int, home string, ok bool, err error) {
	for _, candidate := range strings.Split(strings.TrimRight(passwd, "\n"), "\n") {
		fields := strings.Split(candidate, ":")
		if len(fields) != 7 {
			continue
		}
		u, parseErr := strconv.Atoi(fields[2])
		if parseErr != nil || u != 1000 {
			continue
		}
		g, parseErr := strconv.Atoi(fields[3])
		if parseErr != nil {
			return "", "", 0, 0, "", false, errors.New("invalid Desktop primary GID")
		}
		if fields[0] == "" || strings.ContainsAny(fields[0], "\r\n:/") {
			return "", "", 0, 0, "", false, errors.New("invalid Desktop username in passwd")
		}
		return candidate, fields[0], u, g, fields[5], true, nil
	}
	return "", "", 0, 0, "", false, nil
}

func mergePasswdRecord(path, sourceLine, username string, uid int) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var out []string
	replaced := false
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			out = append(out, line)
			continue
		}
		lineUID, _ := strconv.Atoi(fields[2])
		if fields[0] == username || lineUID == uid {
			if fields[0] != username && lineUID == uid {
				return fmt.Errorf("target UID %d belongs to %s", uid, fields[0])
			}
			if fields[0] == username && lineUID != uid {
				return fmt.Errorf("target user %s has unexpected UID %s", username, fields[2])
			}
			if !replaced {
				out = append(out, sourceLine)
				replaced = true
			}
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, sourceLine)
	}
	return atomicWriteDurable(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func copyNamedRecord(sourcePath, targetPath, name string, field int, mode os.FileMode, required bool) error {
	src, err := os.ReadFile(sourcePath)
	if errors.Is(err, os.ErrNotExist) && !required {
		return nil
	}
	if err != nil {
		return err
	}
	var record string
	for _, line := range strings.Split(strings.TrimRight(string(src), "\n"), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) > field && parts[field] == name {
			record = line
			break
		}
	}
	if record == "" {
		if required {
			return fmt.Errorf("required record %s is missing from %s", name, sourcePath)
		}
		return nil
	}
	target, err := os.ReadFile(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		target = nil
	} else if err != nil {
		return err
	}
	var out []string
	replaced := false
	for _, line := range strings.Split(strings.TrimRight(string(target), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) > field && parts[field] == name {
			if !replaced {
				out = append(out, record)
				replaced = true
			}
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, record)
	}
	return atomicWriteDurable(targetPath, []byte(strings.Join(out, "\n")+"\n"), mode)
}

func mergeDesktopGroups(activeRoot, targetRoot, username string, primaryGID int) error {
	sourceGroupPath := filepath.Join(activeRoot, "etc/group")
	source, err := os.ReadFile(sourceGroupPath)
	if err != nil {
		return err
	}
	targetGroupPath := filepath.Join(targetRoot, "etc/group")
	target, err := os.ReadFile(targetGroupPath)
	if err != nil {
		return err
	}

	sourceGroups := map[string]string{}
	primaryName := ""
	memberGroups := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(string(source), "\n"), "\n") {
		f := strings.Split(line, ":")
		if len(f) != 4 {
			continue
		}
		gid, _ := strconv.Atoi(f[2])
		if gid == primaryGID {
			primaryName = f[0]
			sourceGroups[f[0]] = line
		}
		for _, member := range strings.Split(f[3], ",") {
			if member == username {
				memberGroups[f[0]] = true
				sourceGroups[f[0]] = line
			}
		}
	}
	if primaryName == "" {
		return fmt.Errorf("primary group GID %d is missing", primaryGID)
	}

	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(string(target), "\n"), "\n") {
		f := strings.Split(line, ":")
		if len(f) != 4 {
			out = append(out, line)
			continue
		}
		gid, _ := strconv.Atoi(f[2])
		if gid == primaryGID && f[0] != primaryName {
			return fmt.Errorf("target GID %d belongs to %s", primaryGID, f[0])
		}
		if f[0] == primaryName {
			if gid != primaryGID {
				return fmt.Errorf("target primary group %s has unexpected GID %s", primaryName, f[2])
			}
			out = append(out, sourceGroups[primaryName])
			seen[primaryName] = true
			continue
		}
		if memberGroups[f[0]] {
			members := splitMembers(f[3])
			members[username] = true
			f[3] = joinMembers(members)
			out = append(out, strings.Join(f, ":"))
			seen[f[0]] = true
			continue
		}
		out = append(out, line)
	}
	for name := range sourceGroups {
		if !seen[name] {
			out = append(out, sourceGroups[name])
		}
	}
	if err := atomicWriteDurable(targetGroupPath, []byte(strings.Join(out, "\n")+"\n"), 0o644); err != nil {
		return err
	}

	// gshadow is optional in minimal images. Preserve password/admin fields from
	// the new source while restoring only this human user's memberships.
	targetGShadow := filepath.Join(targetRoot, "etc/gshadow")
	if _, err := os.Stat(targetGShadow); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	gb, err := os.ReadFile(targetGShadow)
	if err != nil {
		return err
	}
	var gout []string
	seenG := map[string]bool{}
	for _, line := range strings.Split(strings.TrimRight(string(gb), "\n"), "\n") {
		f := strings.Split(line, ":")
		if len(f) != 4 {
			gout = append(gout, line)
			continue
		}
		if f[0] == primaryName || memberGroups[f[0]] {
			members := splitMembers(f[3])
			if memberGroups[f[0]] {
				members[username] = true
			}
			f[3] = joinMembers(members)
			seenG[f[0]] = true
		}
		gout = append(gout, strings.Join(f, ":"))
	}
	// If a carried supplementary group is absent from the new source gshadow,
	// create a locked group-shadow record rather than copying any old group password.
	for name := range memberGroups {
		if !seenG[name] {
			gout = append(gout, fmt.Sprintf("%s:!::%s", name, username))
		}
	}
	if !seenG[primaryName] {
		gout = append(gout, fmt.Sprintf("%s:!::", primaryName))
	}
	return atomicWriteDurable(targetGShadow, []byte(strings.Join(gout, "\n")+"\n"), 0o600)
}

func splitMembers(s string) map[string]bool {
	out := map[string]bool{}
	for _, item := range strings.Split(s, ",") {
		if item != "" {
			out[item] = true
		}
	}
	return out
}

func joinMembers(m map[string]bool) string {
	// Keep deterministic output without importing sort by using the small fixed
	// account set in lexical insertion-independent order via repeated selection.
	items := make([]string, 0, len(m))
	for item := range m {
		items = append(items, item)
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j] < items[i] {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return strings.Join(items, ",")
}

func copyPathIfPresent(src, dst string) error {
	info, err := os.Lstat(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(target, dst)
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return atomicWriteDurable(dst, b, info.Mode().Perm())
}

func replaceTreeIfPresent(src, dst string) error {
	info, err := os.Stat(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return copyTree(src, dst)
}

func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyTree(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if info.Mode().IsRegular() {
		return copyPathIfPresent(src, dst)
	}
	return nil
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
