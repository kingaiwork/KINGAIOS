package desktop

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Experience struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Theme       string `json:"theme"`
}

var experiences = []Experience{
	{ID: "kingai-intelligence", Name: "KINGAI Intelligence", Description: "AI-first workspace for agents, memory, knowledge and automation", Theme: "org.kingai.intelligence"},
	{ID: "kingai-flow", Name: "KINGAI Flow", Description: "Modern dock-oriented spatial workflow", Theme: "org.kingai.flow"},
	{ID: "kingai-classic", Name: "KINGAI Classic", Description: "Traditional taskbar and application-menu workflow", Theme: "org.kingai.classic"},
}

func List() []Experience { return append([]Experience(nil), experiences...) }
func find(id string) (Experience, bool) { for _, e := range experiences { if e.ID == id { return e, true } }; return Experience{}, false }

func configPath() (string, error) { dir, err := os.UserConfigDir(); if err != nil { return "", err }; return filepath.Join(dir, "kingai-desktop.ini"), nil }
func Current() (string, error) { p, err := configPath(); if err != nil { return "", err }; return readExperience(p) }

func readExperience(path string) (string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) { return "", nil }
	if err != nil { return "", err }
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "experience=") {
			id := strings.Trim(strings.TrimPrefix(line, "experience="), " \t\r\n\"")
			if _, ok := find(id); ok { return id, nil }
			return "", fmt.Errorf("unknown desktop experience in config: %s", id)
		}
	}
	return "", s.Err()
}

func Set(id string, apply bool) error {
	e, ok := find(id)
	if !ok { return fmt.Errorf("unknown desktop experience %q", id) }
	if apply {
		if err := Apply(e); err != nil { return err }
	}
	p, err := configPath()
	if err != nil { return err }
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil { return err }
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte("[General]\nexperience="+id+"\n"), 0o600); err != nil { return err }
	return os.Rename(tmp, p)
}

func ApplyCurrent() error {
	id, err := Current()
	if err != nil { return err }
	if id == "" { return errors.New("desktop experience has not been selected") }
	e, _ := find(id)
	return Apply(e)
}

func Apply(e Experience) error {
	themeDir := filepath.Join("/usr/share/plasma/look-and-feel", e.Theme)
	if _, err := os.Stat(filepath.Join(themeDir, "manifest.json")); err != nil {
		return fmt.Errorf("desktop theme is not installed or is not a Plasma 6 package: %s", e.Theme)
	}

	var themeApplied bool
	for _, c := range [][]string{{"plasma-apply-lookandfeel", "-a", e.Theme}, {"lookandfeeltool", "-a", e.Theme}} {
		if path, err := exec.LookPath(c[0]); err == nil {
			cmd := exec.Command(path, c[1:]...)
			if out, err := cmd.CombinedOutput(); err != nil { return fmt.Errorf("apply desktop theme: %w: %s", err, string(out)) }
			themeApplied = true
			break
		}
	}
	if !themeApplied { return errors.New("Plasma look-and-feel tool is unavailable") }

	layoutPath := filepath.Join("/usr/share/kingai/desktop/layouts", e.ID+".js")
	layout, err := os.ReadFile(layoutPath)
	if err != nil { return fmt.Errorf("read desktop layout: %w", err) }
	for _, name := range []string{"qdbus6", "/usr/lib/qt6/bin/qdbus", "qdbus"} {
		path := name
		if !filepath.IsAbs(path) {
			p, err := exec.LookPath(name); if err != nil { continue }; path = p
		} else if _, err := os.Stat(path); err != nil { continue }
		cmd := exec.Command(path, "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", string(layout))
		if out, err := cmd.CombinedOutput(); err != nil { return fmt.Errorf("apply Plasma layout: %w: %s", err, string(out)) }
		return nil
	}
	return errors.New("Qt 6 D-Bus client is unavailable")
}
