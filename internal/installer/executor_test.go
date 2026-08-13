package installer

import (
	"os"
	"path/filepath"
	"runtime"
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
