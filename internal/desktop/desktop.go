package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type Config struct {
	Experience string `json:"experience"`
}

func List() []Experience { return append([]Experience(nil), experiences...) }

func find(id string) (Experience, bool) {
	for _, e := range experiences {
		if e.ID == id {
			return e, true
		}
	}
	return Experience{}, false
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kingai", "desktop.json"), nil
}

func Current() (string, error) {
	p, err := configPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return "", err
	}
	return c.Experience, nil
}

func Set(id string, apply bool) error {
	e, ok := find(id)
	if !ok {
		return fmt.Errorf("unknown desktop experience %q", id)
	}
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(Config{Experience: id}, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		return err
	}
	if !apply {
		return nil
	}
	commands := [][]string{{"plasma-apply-lookandfeel", "-a", e.Theme}, {"lookandfeeltool", "-a", e.Theme}}
	for _, c := range commands {
		if path, err := exec.LookPath(c[0]); err == nil {
			cmd := exec.Command(path, c[1:]...)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("apply desktop profile: %w: %s", err, string(out))
			}
			return nil
		}
	}
	return nil
}
