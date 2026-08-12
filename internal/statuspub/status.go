package statuspub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Snapshot is intentionally non-sensitive. It is safe for local desktop users
// to read and must never contain prompts, memory content, file paths, tokens,
// provider credentials, user identifiers, or raw audit events.
type Snapshot struct {
	Product          string    `json:"product"`
	Version          string    `json:"version"`
	Architecture     string    `json:"architecture"`
	Health           string    `json:"health"`
	Policy           string    `json:"policy"`
	RegisteredAgents int       `json:"registered_agents"`
	ModelStrategy    string    `json:"model_strategy"`
	ModelMode        string    `json:"model_mode"`
	MemoryMode       string    `json:"memory_mode"`
	CloudRequired    bool      `json:"cloud_required"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func Write(path string, s Snapshot) error {
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = time.Now().UTC()
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	// The daemon runs with a restrictive umask; explicitly expose only this
	// sanitized local status snapshot as world-readable.
	if err := os.Chmod(tmp, 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
