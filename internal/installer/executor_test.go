package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPartitionPath(t *testing.T) {
	cases := map[string]string{
		"/dev/sda":     "/dev/sda2",
		"/dev/vda":     "/dev/vda2",
		"/dev/nvme0n1": "/dev/nvme0n1p2",
		"/dev/nbd0":    "/dev/nbd0p2",
	}
	for dev, want := range cases {
		if got := partitionPath(dev, 2); got != want {
			t.Fatalf("partitionPath(%q,2)=%q want %q", dev, got, want)
		}
	}
}

func TestValidateExecutionInputsRejectsNonBlockTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "disk.raw")
	if err := os.WriteFile(target, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "root")
	if err := os.MkdirAll(filepath.Join(root, "usr/lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr/lib/os-release"), []byte("NAME=\"KINGAI OS\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(dir, "state.key")
	if err := os.WriteFile(key, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateExecutionInputs(ExecuteOptions{Target: target, SourceRoot: root, StateKey: key}); err == nil {
		t.Fatal("expected regular-file target to be rejected")
	}
}

func TestValidateExecutionInputsRejectsWeakKeyPermissions(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux device semantics")
	}
	dir := t.TempDir()
	key := filepath.Join(dir, "state.key")
	if err := os.WriteFile(key, make([]byte, 64), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(key)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 == 0 {
		t.Fatal("test key unexpectedly private")
	}
}

func TestPreparePersistentStateLayout(t *testing.T) {
	rootA := filepath.Join(t.TempDir(), "root-a")
	rootB := filepath.Join(t.TempDir(), "root-b")
	state := filepath.Join(t.TempDir(), "state")
	for _, p := range []string{rootA, rootB, state} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := preparePersistentStateLayout(rootA, rootB, state); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"kingai/update", "kingai/runtime/lib", "kingai/runtime/log"} {
		info, err := os.Stat(filepath.Join(state, rel))
		if err != nil {
			t.Fatalf("missing STATE path %s: %v", rel, err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("STATE path %s mode=%o want 700", rel, got)
		}
	}
	for _, root := range []string{rootA, rootB} {
		for _, rel := range []string{"var/lib/kingai-state", "var/lib/kingai", "var/log/kingai"} {
			if info, err := os.Stat(filepath.Join(root, rel)); err != nil || !info.IsDir() {
				t.Fatalf("missing installed mountpoint %s: %v", filepath.Join(root, rel), err)
			}
		}
	}
}

func TestWriteInstalledConfigRequiresEncryptedRuntimeState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "etc/kingai"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "var/lib/kingai-state"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeInstalledConfig(root, "root-uuid", "efi-uuid", "luks-uuid", false, ""); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "etc/fstab"))
	if err != nil {
		t.Fatal(err)
	}
	fstab := string(b)
	for _, want := range []string{
		"/dev/mapper/KINGAI_STATE /var/lib/kingai-state ext4 defaults 0 2",
		"/var/lib/kingai-state/kingai/runtime/lib /var/lib/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
		"/var/lib/kingai-state/kingai/runtime/log /var/log/kingai none bind,x-systemd.requires-mounts-for=/var/lib/kingai-state 0 0",
	} {
		if !strings.Contains(fstab, want) {
			t.Fatalf("fstab missing %q:\n%s", want, fstab)
		}
	}
	if strings.Contains(fstab, "KINGAI_STATE /var/lib/kingai-state ext4 defaults,nofail") {
		t.Fatalf("encrypted STATE must not be optional:\n%s", fstab)
	}
}

func TestInstalledVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usr/lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr/lib/os-release"), []byte("NAME=\"KINGAI OS\"\nVERSION_ID=\"0.1.0-dev\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := installedVersion(root); got != "0.1.0-dev" {
		t.Fatalf("installedVersion=%q", got)
	}
}
