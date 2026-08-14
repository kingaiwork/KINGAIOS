package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadQtSettingsExperience(t *testing.T) {
	p := filepath.Join(t.TempDir(), "desktop.ini")
	if err := os.WriteFile(p, []byte("[General]\nexperience=kingai-flow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readExperience(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "kingai-flow" {
		t.Fatalf("got %q", got)
	}
}

func TestReadRejectsUnknownExperience(t *testing.T) {
	p := filepath.Join(t.TempDir(), "desktop.ini")
	_ = os.WriteFile(p, []byte("[General]\nexperience=third-party\n"), 0o600)
	if _, err := readExperience(p); err == nil {
		t.Fatal("unknown experience must fail closed")
	}
}

func TestDefaultExperienceIsIntelligence(t *testing.T) {
	got := Default()
	if got.ID != "kingai-intelligence" || !got.Default {
		t.Fatalf("unexpected default experience: %#v", got)
	}
	defaults := 0
	for _, e := range List() {
		if e.Default {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatalf("want exactly one default experience, got %d", defaults)
	}
}

func TestSetPersistsWithoutApplying(t *testing.T) {
	p := filepath.Join(t.TempDir(), "kingai-desktop.ini")
	t.Setenv("KINGAI_DESKTOP_CONFIG", p)
	if err := Set("kingai-classic", false); err != nil {
		t.Fatal(err)
	}
	got, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	if got != "kingai-classic" {
		t.Fatalf("got %q", got)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o", st.Mode().Perm())
	}
}

func TestValidateInstalledExperienceAssets(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KINGAI_DESKTOP_ROOT", root)
	for _, e := range List() {
		manifest := filepath.Join(root, experienceManifestPath(e))
		if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(manifest, data, 0o644); err != nil {
			t.Fatal(err)
		}
		themeManifest := rooted(root, filepath.Join("/usr/share/plasma/look-and-feel", e.Theme, "manifest.json"))
		if err := os.MkdirAll(filepath.Dir(themeManifest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(themeManifest, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		layout := rooted(root, filepath.Join("/usr/share/kingai/desktop/layouts", e.Layout))
		if err := os.MkdirAll(filepath.Dir(layout), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(layout, []byte("// layout"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := ValidateInstalled(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateInstalledRejectsManifestMismatch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KINGAI_DESKTOP_ROOT", root)
	e := Default()
	manifest := rooted(root, experienceManifestPath(e))
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	e.Theme = "org.example.untrusted"
	data, _ := json.Marshal(e)
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInstalled(); err == nil {
		t.Fatal("mismatched desktop manifest must fail")
	}
}
