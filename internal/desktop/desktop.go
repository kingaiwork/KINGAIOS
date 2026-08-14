package desktop

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultExperienceID = "kingai-intelligence"

type Experience struct {
	Schema          int      `json:"schema"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Theme           string   `json:"theme"`
	Layout          string   `json:"layout"`
	Mode            string   `json:"mode"`
	Default         bool     `json:"default"`
	PrimarySurfaces []string `json:"primary_surfaces"`
}

var experiences = []Experience{
	{Schema: 1, ID: "kingai-intelligence", Name: "KINGAI Intelligence", Description: "AI-first workspace centered on agents, tasks, memory, knowledge, models and automation.", Theme: "org.kingai.intelligence", Layout: "kingai-intelligence.js", Mode: "ai-first", Default: true, PrimarySurfaces: []string{"agents", "tasks", "approvals", "memory", "knowledge", "models", "automation"}},
	{Schema: 1, ID: "kingai-flow", Name: "KINGAI Flow", Description: "Modern spatial workflow with a centered dock and workspace-first interaction.", Theme: "org.kingai.flow", Layout: "kingai-flow.js", Mode: "dock-spatial", PrimarySurfaces: []string{"workspace", "search", "agents", "tasks", "dock"}},
	{Schema: 1, ID: "kingai-classic", Name: "KINGAI Classic", Description: "Familiar personal-computer workflow with an application menu, taskbar and system tray.", Theme: "org.kingai.classic", Layout: "kingai-classic.js", Mode: "taskbar-menu", PrimarySurfaces: []string{"application-menu", "taskbar", "agents", "tasks", "system-tray"}},
}

func cloneExperience(e Experience) Experience {
	e.PrimarySurfaces = append([]string(nil), e.PrimarySurfaces...)
	return e
}

func List() []Experience {
	out := make([]Experience, len(experiences))
	for i, e := range experiences {
		out[i] = cloneExperience(e)
	}
	return out
}

func find(id string) (Experience, bool) {
	for _, e := range experiences {
		if e.ID == id {
			return cloneExperience(e), true
		}
	}
	return Experience{}, false
}

func Describe(id string) (Experience, error) {
	e, ok := find(id)
	if !ok {
		return Experience{}, fmt.Errorf("unknown desktop experience %q", id)
	}
	return e, nil
}

func Default() Experience {
	e, _ := find(defaultExperienceID)
	return e
}

func configPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KINGAI_DESKTOP_CONFIG")); override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("KINGAI_DESKTOP_CONFIG must be an absolute path")
		}
		return filepath.Clean(override), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kingai-desktop.ini"), nil
}

func Current() (string, error) {
	p, err := configPath()
	if err != nil {
		return "", err
	}
	return readExperience(p)
}

func readExperience(path string) (string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "experience=") {
			id := strings.Trim(strings.TrimPrefix(line, "experience="), " \t\r\n\"")
			if _, ok := find(id); ok {
				return id, nil
			}
			return "", fmt.Errorf("unknown desktop experience in config: %s", id)
		}
	}
	return "", s.Err()
}

func writeExperience(path, id string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := fmt.Fprintf(f, "[General]\nexperience=%s\n", id); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func Set(id string, apply bool) error {
	e, ok := find(id)
	if !ok {
		return fmt.Errorf("unknown desktop experience %q", id)
	}
	if apply {
		if err := Apply(e); err != nil {
			return err
		}
	}
	p, err := configPath()
	if err != nil {
		return err
	}
	return writeExperience(p, id)
}

func ApplyCurrent() error {
	id, err := Current()
	if err != nil {
		return err
	}
	if id == "" {
		return errors.New("desktop experience has not been selected")
	}
	e, _ := find(id)
	return Apply(e)
}

func desktopRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("KINGAI_DESKTOP_ROOT"))
	if root == "" {
		return "/", nil
	}
	if !filepath.IsAbs(root) {
		return "", errors.New("KINGAI_DESKTOP_ROOT must be an absolute path")
	}
	return filepath.Clean(root), nil
}

func rooted(root, path string) string {
	if root == "/" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func experienceManifestPath(e Experience) string {
	name := strings.TrimPrefix(e.ID, "kingai-") + ".json"
	return filepath.Join("/usr/share/kingai/desktop/experiences", name)
}

func validateExperienceAssets(root string, e Experience) error {
	manifestPath := rooted(root, experienceManifestPath(e))
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read desktop experience manifest %s: %w", e.ID, err)
	}
	var manifest Experience
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse desktop experience manifest %s: %w", e.ID, err)
	}
	if manifest.Schema != 1 || manifest.ID != e.ID || manifest.Theme != e.Theme || manifest.Layout != e.Layout {
		return fmt.Errorf("desktop experience manifest mismatch for %s", e.ID)
	}
	if _, err := os.Stat(rooted(root, filepath.Join("/usr/share/plasma/look-and-feel", e.Theme, "manifest.json"))); err != nil {
		return fmt.Errorf("desktop theme is not installed for %s: %w", e.ID, err)
	}
	if _, err := os.Stat(rooted(root, filepath.Join("/usr/share/kingai/desktop/layouts", e.Layout))); err != nil {
		return fmt.Errorf("desktop layout is not installed for %s: %w", e.ID, err)
	}
	return nil
}

func ValidateInstalled() error {
	root, err := desktopRoot()
	if err != nil {
		return err
	}
	var problems []error
	for _, e := range experiences {
		if err := validateExperienceAssets(root, e); err != nil {
			problems = append(problems, err)
		}
	}
	return errors.Join(problems...)
}

func findCommand(candidates ...string) (string, error) {
	for _, name := range candidates {
		if filepath.IsAbs(name) {
			if st, err := os.Stat(name); err == nil && st.Mode().IsRegular() && st.Mode()&0o111 != 0 {
				return name, nil
			}
			continue
		}
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("command is unavailable")
}

func Apply(e Experience) error {
	known, ok := find(e.ID)
	if !ok || known.Theme != e.Theme || known.Layout != e.Layout {
		return fmt.Errorf("untrusted desktop experience %q", e.ID)
	}
	if err := validateExperienceAssets("/", known); err != nil {
		return err
	}

	layoutPath := filepath.Join("/usr/share/kingai/desktop/layouts", known.Layout)
	layout, err := os.ReadFile(layoutPath)
	if err != nil {
		return fmt.Errorf("read desktop layout: %w", err)
	}
	themeTool, err := findCommand("plasma-apply-lookandfeel", "lookandfeeltool")
	if err != nil {
		return errors.New("Plasma look-and-feel tool is unavailable")
	}
	qdbus, err := findCommand("qdbus6", "/usr/lib/qt6/bin/qdbus", "qdbus")
	if err != nil {
		return errors.New("Qt 6 D-Bus client is unavailable")
	}

	cmd := exec.Command(themeTool, "-a", known.Theme)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apply desktop theme: %w: %s", err, strings.TrimSpace(string(out)))
	}
	cmd = exec.Command(qdbus, "org.kde.plasmashell", "/PlasmaShell", "org.kde.PlasmaShell.evaluateScript", string(layout))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apply Plasma layout: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
