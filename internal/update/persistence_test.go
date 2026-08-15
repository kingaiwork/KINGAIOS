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
	if err := rewriteRuntimeFstab(root); err != nil {
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
}

func TestPrepareTargetRuntimePersistenceInstallsLegacyBootGuard(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc/fstab"), []byte("UUID=root / ext4 defaults 0 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepareTargetRuntimePersistence(root, state, SlotA, "aaaa-bbbb", "cccc-dddd"); err != nil {
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
	if err := prepareTargetRuntimePersistence(root, state, SlotB, "aaaa", "bbbb"); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{migrationConfigRel, migrationUnitRel, migrationDropInRel} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("stale migration guard survived marker: %s err=%v", rel, err)
		}
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
