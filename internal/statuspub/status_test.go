package statuspub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPublicStatusIsSanitizedAndReadable(t *testing.T) {
	root := t.TempDir()
	approvals := filepath.Join(root, "approvals")
	tasks := filepath.Join(root, "tasks")
	models := filepath.Join(root, "models.json")
	if err := os.MkdirAll(approvals, 0o700); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(tasks, 0o700); err != nil { t.Fatal(err) }
	now := time.Now().UTC()
	mustWriteJSON(t, filepath.Join(approvals, "one.json"), map[string]any{"status":"pending","expires_at":now.Add(time.Minute)})
	mustWriteJSON(t, filepath.Join(approvals, "two.json"), map[string]any{"status":"pending","expires_at":now.Add(-time.Minute)})
	mustWriteJSON(t, filepath.Join(tasks, "running.json"), map[string]any{"status":"running","goal":"must never be published"})
	mustWriteJSON(t, filepath.Join(tasks, "waiting.json"), map[string]any{"status":"waiting","goal":"private waiting task text"})
	mustWriteJSON(t, filepath.Join(tasks, "approval.json"), map[string]any{"status":"waiting_approval","target":"approval-target-secret"})
	mustWriteJSON(t, filepath.Join(tasks, "blocked.json"), map[string]any{"status":"blocked","goal":"private blocked task text"})
	mustWriteJSON(t, filepath.Join(tasks, "paused.json"), map[string]any{"status":"paused","goal":"private paused task text"})
	mustWriteJSON(t, filepath.Join(tasks, "planning.json"), map[string]any{"status":"planning","goal":"private planning task text"})
	mustWriteJSON(t, filepath.Join(tasks, "completed.json"), map[string]any{"status":"completed","goal":"private completed task text"})
	mustWriteJSON(t, filepath.Join(tasks, "failed.json"), map[string]any{"status":"failed","error":"private failure text"})
	mustWriteJSON(t, models, map[string]any{"providers":[]map[string]any{{"provider":"local"},{"provider":"local"},{"provider":"cloud"}}})
	t.Setenv("KINGAI_APPROVAL_ROOT", approvals)
	t.Setenv("KINGAI_TASK_ROOT", tasks)
	t.Setenv("KINGAI_MODELS", models)

	p := filepath.Join(root, "status.json")
	s := Snapshot{Product:"KINGAI OS", Version:"0.1", Architecture:"D5", Health:"ok", Policy:"enabled", RegisteredAgents:3, ModelStrategy:"provider-neutral", ModelMode:"hybrid", MemoryMode:"local-first", UpdatedAt:now}
	if err := Write(p, s); err != nil { t.Fatal(err) }
	b, err := os.ReadFile(p); if err != nil { t.Fatal(err) }
	var got Snapshot
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.Product != "KINGAI OS" || got.RegisteredAgents != 3 { t.Fatalf("unexpected status: %#v", got) }
	if got.ActiveTasks != 6 || got.PendingApprovals != 1 || got.ModelProviders != 2 { t.Fatalf("unexpected activity counts: %#v", got) }
	if got.RunningTasks != 1 || got.WaitingTasks != 1 || got.WaitingApprovalTasks != 1 || got.BlockedTasks != 1 || got.PausedTasks != 1 || got.PlanningTasks != 1 {
		t.Fatalf("unexpected task state counts: %#v", got)
	}
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"prompt", "token", "secret", "memory_content", "api_key", "password", "must never be published", "private waiting task text", "private blocked task text", "private paused task text", "private planning task text", "private completed task text", "private failure text"} {
		if strings.Contains(text, forbidden) { t.Fatalf("public status contains forbidden content %q", forbidden) }
	}
	st, err := os.Stat(p); if err != nil { t.Fatal(err) }
	if st.Mode().Perm() != 0o644 { t.Fatalf("status mode=%o want 644", st.Mode().Perm()) }
}

func mustWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, b, 0o600); err != nil { t.Fatal(err) }
}
