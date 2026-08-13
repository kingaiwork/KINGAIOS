package update

import (
	"os"
	"path/filepath"
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
