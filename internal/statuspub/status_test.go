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
	mustWriteJSON(t, filepath.Join(tasks, "one.json"), map[string]any{"status":"running","goal":"must never be published"})
	mustWriteJSON(t, filepath.Join(tasks, "two.json"), map[string]any{"status":"completed","goal":"private task text"})
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
	if got.ActiveTasks != 1 || got.PendingApprovals != 1 || got.ModelProviders != 2 { t.Fatalf("unexpected activity counts: %#v", got) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"prompt", "token", "secret", "memory_content", "api_key", "password", "must never be published", "private task text"} {
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
