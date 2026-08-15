package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNaturalCompareKernelVersions(t *testing.T) {
	cases := []struct{ a, b string; want int }{
		{"vmlinuz-6.11.0", "vmlinuz-6.9.0", 1},
		{"vmlinuz-6.11.0-10", "vmlinuz-6.11.0-9", 1},
		{"initrd.img-6.14.0-1009-generic", "initrd.img-6.14.0-1008-generic", 1},
		{"vmlinuz-6.11.0", "vmlinuz-6.11.0", 0},
	}
	for _, tc := range cases {
		got := naturalCompare(tc.a, tc.b)
		if tc.want < 0 && got >= 0 || tc.want == 0 && got != 0 || tc.want > 0 && got <= 0 {
			t.Fatalf("naturalCompare(%q,%q)=%d, want sign %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNewestBootFileUsesVersionOrdering(t *testing.T) {
	root := t.TempDir()
	boot := filepath.Join(root, "boot")
	if err := os.MkdirAll(boot, 0o755); err != nil { t.Fatal(err) }
	for _, name := range []string{"vmlinuz-6.9.0-99", "vmlinuz-6.11.0-9", "vmlinuz-6.11.0-10"} {
		if err := os.WriteFile(filepath.Join(boot, name), []byte("x"), 0o644); err != nil { t.Fatal(err) }
	}
	got, err := newestBootFile(root, "vmlinuz-*")
	if err != nil { t.Fatal(err) }
	if want := "vmlinuz-6.11.0-10"; filepath.Base(got) != want {
		t.Fatalf("newestBootFile=%q, want %q", filepath.Base(got), want)
	}
}

func TestBootControllerDisablesPlymouthForHeadlessSlots(t *testing.T) {
	a, b := makeBootRoots(t, false)
	controller := BootController{RootAPath: a, RootBPath: b, RootAUUID: "root-a", RootBUUID: "root-b"}
	if err := controller.WriteConfig(); err != nil { t.Fatal(err) }
	cfg, err := os.ReadFile(filepath.Join(a, "boot/grub/grub.cfg"))
	if err != nil { t.Fatal(err) }
	text := string(cfg)
	if !strings.Contains(text, "console=ttyS0,115200n8 plymouth.enable=0") {
		t.Fatalf("headless grub config must keep serial console and disable Plymouth: %s", text)
	}
}

func TestBootControllerKeepsPlymouthForGraphicalSlots(t *testing.T) {
	a, b := makeBootRoots(t, true)
	controller := BootController{RootAPath: a, RootBPath: b, RootAUUID: "root-a", RootBUUID: "root-b"}
	if err := controller.WriteConfig(); err != nil { t.Fatal(err) }
	cfg, err := os.ReadFile(filepath.Join(a, "boot/grub/grub.cfg"))
	if err != nil { t.Fatal(err) }
	text := string(cfg)
	if strings.Contains(text, "plymouth.enable=0") {
		t.Fatalf("graphical grub config must preserve Plymouth: %s", text)
	}
	if !strings.Contains(text, "console=tty0 console=ttyS0,115200n8") {
		t.Fatalf("graphical grub config must preserve console arguments: %s", text)
	}
}

func makeBootRoots(t *testing.T, graphical bool) (string, string) {
	t.Helper()
	roots := []string{filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")}
	for _, root := range roots {
		boot := filepath.Join(root, "boot")
		if err := os.MkdirAll(boot, 0o755); err != nil { t.Fatal(err) }
		if err := os.WriteFile(filepath.Join(boot, "vmlinuz-6.11.0"), []byte("kernel"), 0o644); err != nil { t.Fatal(err) }
		if err := os.WriteFile(filepath.Join(boot, "initrd.img-6.11.0"), []byte("initrd"), 0o644); err != nil { t.Fatal(err) }
		if graphical {
			unit := filepath.Join(root, "usr/lib/systemd/system/sddm.service")
			if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil { t.Fatal(err) }
			if err := os.WriteFile(unit, []byte("[Unit]\nDescription=SDDM\n"), 0o644); err != nil { t.Fatal(err) }
		}
	}
	return roots[0], roots[1]
}
