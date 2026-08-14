package statuspub

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Snapshot is intentionally non-sensitive. It is safe for local desktop users
// to read and must never contain prompts, memory content, file paths, tokens,
// provider credentials, user identifiers, task goals, approval targets, or raw
// audit events.
type Snapshot struct {
	Product          string    `json:"product"`
	Version          string    `json:"version"`
	Architecture     string    `json:"architecture"`
	Health           string    `json:"health"`
	Policy           string    `json:"policy"`
	RegisteredAgents int       `json:"registered_agents"`
	ActiveTasks      int       `json:"active_tasks"`
	PendingApprovals int       `json:"pending_approvals"`
	ModelProviders   int       `json:"model_providers"`
	ModelStrategy    string    `json:"model_strategy"`
	ModelMode        string    `json:"model_mode"`
	MemoryMode       string    `json:"memory_mode"`
	CloudRequired    bool      `json:"cloud_required"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type approvalSummary struct {
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type taskSummary struct {
	Status string `json:"status"`
}

type modelsSummary struct {
	Providers []struct {
		Provider string `json:"provider"`
	} `json:"providers"`
}

func Write(path string, s Snapshot) error {
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = time.Now().UTC()
	}
	// Enrichment is count-only and intentionally discards identities, task
	// content, targets and memory. Failures leave counts at zero rather than
	// making the public status path a runtime dependency.
	s.ActiveTasks = countActiveTasks(envOr("KINGAI_TASK_ROOT", "/var/lib/kingai/tasks"))
	s.PendingApprovals = countPendingApprovals(envOr("KINGAI_APPROVAL_ROOT", "/var/lib/kingai/approvals"), s.UpdatedAt)
	s.ModelProviders = countModelProviders(envOr("KINGAI_MODELS", "/etc/kingai/models.json"))

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

func countActiveTasks(root string) int {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) || err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		var task taskSummary
		if json.Unmarshal(b, &task) != nil {
			continue
		}
		switch task.Status {
		case "completed", "failed", "cancelled":
			continue
		default:
			count++
		}
	}
	return count
}

func countPendingApprovals(root string, now time.Time) int {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) || err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		var approval approvalSummary
		if json.Unmarshal(b, &approval) != nil {
			continue
		}
		if approval.Status == "pending" && (approval.ExpiresAt.IsZero() || now.Before(approval.ExpiresAt)) {
			count++
		}
	}
	return count
}

func countModelProviders(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var cfg modelsSummary
	if json.Unmarshal(b, &cfg) != nil {
		return 0
	}
	ids := make(map[string]struct{})
	for _, provider := range cfg.Providers {
		id := strings.TrimSpace(provider.Provider)
		if id == "" {
			id = "configured"
		}
		ids[id] = struct{}{}
	}
	return len(ids)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
