package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteRuntimeFstabReplacesLegacyOptionalState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "UUID=root / ext4 defaults 0 1\n" +
		"UUID=efi /boot/efi vfat umask=0077 0 2\n" +
		"/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults,nofail 0 2\n"
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rewriteRuntimeFstab(root, false); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "etc/fstab"))
	if err != nil {
		t.Fatal(err)
	}
	fstab := string(b)
	if strings.Contains(fstab, "defaults,nofail") {
		t.Fatalf("legacy optional STATE mount survived rewrite:\n%s", fstab)
	}
	for _, want := range []string{
		"/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults 0 2",
		"/var/lib/kingai-state/kingai/runtime/lib /var/lib/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
		"/var/lib/kingai-state/kingai/runtime/log /var/log/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
	} {
		if !strings.Contains(fstab, want) {
			t.Fatalf("rewritten fstab missing %q:\n%s", want, fstab)
		}
	}
	if strings.Contains(fstab, " /home ") {
		t.Fatalf("server/runtime rewrite unexpectedly enabled persistent home:\n%s", fstab)
	}
}

func TestRewriteRuntimeFstabAddsEncryptedDesktopHome(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "UUID=root / ext4 defaults 0 1\n" +
		"/old/home /home none bind 0 0\n"
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rewriteRuntimeFstab(root, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "etc/fstab"))
	if err != nil {
		t.Fatal(err)
	}
	fstab := string(b)
	want := "/var/lib/kingai-state/home /home none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0"
	if strings.Count(fstab, " /home ") != 1 || !strings.Contains(fstab, want) {
		t.Fatalf("persistent Desktop home not canonicalized:\n%s", fstab)
	}
}

func TestPrepareTargetRuntimePersistenceInstallsLegacyBootGuard(t *testing.T) {
	root := t.TempDir()
	active := t.TempDir()
	state := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte("UUID=root / ext4 defaults 0 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepareTargetRuntimePersistence(root, active, state, SlotA, "aaaa-bbbb", "cccc-dddd"); err != nil {
		t.Fatal(err)
	}
	conf, err := os.ReadFile(filepath.Join(root, migrationConfigRel))
	if err != nil {
		t.Fatal(err)
	}
	if string(conf) != "SOURCE_ROOT_UUID=aaaa-bbbb\nSOURCE_SLOT=A\n" {
		t.Fatalf("unexpected migration metadata: %q", conf)
	}
	unit, err := os.ReadFile(filepath.Join(root, migrationUnitRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unit), "ExecStart=/usr/lib/kingai/kingai-update migrate-state") {
		t.Fatalf("migration unit does not invoke update binary:\n%s", unit)
	}
	dropIn, err := os.ReadFile(filepath.Join(root, migrationDropInRel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dropIn), "Requires=kingai-state-migrate.service") {
		t.Fatalf("kingaid migration dependency missing:\n%s", dropIn)
	}
}

func TestPrepareTargetRuntimePersistenceSkipsMigrationAfterMarker(t *testing.T) {
	root := t.TempDir()
	active := t.TempDir()
	state := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte("UUID=root / ext4 defaults 0 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(state, stateRuntimeMarkerRel)
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("layout=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{migrationConfigRel, migrationUnitRel, migrationDropInRel} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("stale"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := prepareTargetRuntimePersistence(root, active, state, SlotB, "aaaa", "bbbb"); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{migrationConfigRel, migrationUnitRel, migrationDropInRel} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("stale migration guard survived marker: %s err=%v", rel, err)
		}
	}
}

func TestPreserveDesktopIdentityAcrossABSlot(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root to validate encrypted home ownership")
	}
	active := t.TempDir()
	target := t.TempDir()
	state := t.TempDir()
	for _, root := range []string{active, target} {
		for _, dir := range []string{"etc", "etc/default", "etc/NetworkManager/system-connections", "usr/share/applications"} {
			if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
				t.Fatal(err)
			}
	}
	}
	if err := os.MkdirAll(filepath.Join(state, stateHomeRel), 0o755); err != nil {
		t.Fatal(err)
	}

	write := func(root, rel, data string, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(active, "etc/passwd", "root:x:0:0:root:/root:/bin/bash\nkingai:x:1000:1000:King:/home/kingai:/bin/bash\n", 0o644)
	write(active, "etc/shadow", "root:*:1:0:99999:7:::\nkingai:$6$activehash:1:0:99999:7:::\n", 0o600)
	write(active, "etc/group", "root:x:0:\nsudo:x:27:kingai\nplugdev:x:46:kingai\nkingai:x:1000:\n", 0o644)
	write(active, "etc/gshadow", "root:*::\nsudo:*::kingai\nplugdev:*::kingai\nkingai:!::\n", 0o600)
	write(active, "etc/subuid", "kingai:100000:65536\n", 0o644)
	write(active, "etc/subgid", "kingai:100000:65536\n", 0o644)
	write(active, "etc/hostname", "kingai-laptop\n", 0o644)
	write(active, "etc/hosts", "127.0.0.1 localhost\n127.0.1.1 kingai-laptop\n", 0o644)
	write(active, "etc/default/locale", "LANG=en_US.UTF-8\n", 0o644)
	write(active, "etc/timezone", "America/Los_Angeles\n", 0o644)
	write(active, "etc/machine-id", "0123456789abcdef0123456789abcdef\n", 0o444)
	write(active, "etc/NetworkManager/system-connections/home.nmconnection", "[connection]\nid=Home\n", 0o600)

	write(target, "etc/passwd", "root:x:0:0:root:/root:/bin/bash\n", 0o644)
	write(target, "etc/shadow", "root:*:1:0:99999:7:::\n", 0o600)
	write(target, "etc/group", "root:x:0:\nsudo:x:27:\nplugdev:x:46:\n", 0o644)
	write(target, "etc/gshadow", "root:*::\nsudo:*::\nplugdev:*::\n", 0o600)
	write(target, "usr/share/applications/kingai-installer.desktop", "live only\n", 0o644)
	write(target, "etc/fstab", "UUID=root / ext4 defaults 0 1\n", 0o644)

	marker := filepath.Join(state, stateRuntimeMarkerRel)
	if err := os.MkdirAll(filepath.Dir(marker), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("layout=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := prepareTargetRuntimePersistence(target, active, state, SlotA, "aaaa", "bbbb"); err != nil {
		t.Fatal(err)
	}

	mustContain := func(rel, needle string) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), needle) {
			t.Fatalf("%s missing %q:\n%s", rel, needle, b)
		}
	}
	mustContain("etc/passwd", "kingai:x:1000:1000:King:/home/kingai:/bin/bash")
	mustContain("etc/shadow", "kingai:$6$activehash:")
	mustContain("etc/group", "sudo:x:27:kingai")
	mustContain("etc/group", "kingai:x:1000:")
	mustContain("etc/fstab", "/var/lib/kingai-state/home /home none bind,")
	mustContain("etc/hostname", "kingai-laptop")
	mustContain("etc/machine-id", "0123456789abcdef0123456789abcdef")
	mustContain("etc/NetworkManager/system-connections/home.nmconnection", "id=Home")
	if _, err := os.Stat(filepath.Join(target, "usr/share/applications/kingai-installer.desktop")); !os.IsNotExist(err) {
		t.Fatalf("Live installer launcher survived staged Desktop update: %v", err)
	}
	home, err := os.Stat(filepath.Join(state, stateHomeRel, "kingai"))
	if err != nil {
		t.Fatal(err)
	}
	if got := home.Mode().Perm(); got != 0o700 {
		t.Fatalf("encrypted home mode=%o want 700", got)
	}
}

func TestAtomicWriteDurableReplacesAndHardensMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "slots.json")
	if err := atomicWriteDurable(path, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteDurable(path, []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "second\n" {
		t.Fatalf("unexpected durable state: %q", b)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Fatalf("durable state mode=%o want 600", got)
	}
}
