package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func TestSanitizeTasksKeepsUserFacingTaskButDropsSensitiveStepData(t *testing.T) {
	now := time.Now().UTC()
	tasks := []taskgraph.Task{{
		ID: "task-1", Goal: "prepare private project", Agent: "main", PeerUID: 1234,
		Status: taskgraph.StatusRunning, CreatedAt: now.Add(-time.Minute), UpdatedAt: now,
		Steps: []taskgraph.Step{
			{ID: "one", Title: "read", Capability: "filesystem.read", Target: "/home/user/private", ApprovalID: "approval-secret", Status: taskgraph.StatusCompleted, Result: json.RawMessage(`{"secret":"result"}`)},
			{ID: "two", Title: "write", Capability: "filesystem.write", Target: "/home/user/output", Status: taskgraph.StatusBlocked, Error: "private failure"},
		},
	}}

	got := sanitizeTasks(tasks)
	if len(got) != 1 { t.Fatalf("len=%d", len(got)) }
	if got[0].ID != "task-1" || got[0].Goal != "prepare private project" || got[0].Agent != "main" || got[0].Status != taskgraph.StatusRunning {
		t.Fatalf("unexpected task: %#v", got[0])
	}
	if got[0].StepCount != 2 || got[0].DoneSteps != 1 || got[0].FailedSteps != 1 {
		t.Fatalf("unexpected step summary: %#v", got[0])
	}
	b, err := json.Marshal(got)
	if err != nil { t.Fatal(err) }
	text := strings.ToLower(string(b))
	for _, forbidden := range []string{"filesystem.read", "/home/user/private", "approval-secret", "result", "private failure", "peer_uid", "target", "capability", "approval_id"} {
		if strings.Contains(text, strings.ToLower(forbidden)) { t.Fatalf("private desktop snapshot leaked raw step data %q: %s", forbidden, text) }
	}
}

func TestRunOnceWritesPrivateSnapshotFromUnixPeerAPI(t *testing.T) {
	tmp := t.TempDir()
	socket := filepath.Join(tmp, "kingaid.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil { t.Fatal(err) }
	defer ln.Close()

	now := time.Now().UTC()
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/list" { http.NotFound(w, r); return }
		_ = json.NewEncoder(w).Encode([]taskgraph.Task{{
			ID: "mine", Goal: "user task", Agent: "main", PeerUID: uint32(os.Geteuid()), Status: taskgraph.StatusWaiting,
			Steps: []taskgraph.Step{{ID: "s1", Title: "private title", Target: "/private/target", Status: taskgraph.StatusCreated}},
			CreatedAt: now, UpdatedAt: now,
		}})
	})}
	go func() { _ = server.Serve(ln) }()
	defer server.Shutdown(context.Background())

	output := filepath.Join(tmp, "runtime", "kingai", "desktop-private.json")
	t.Setenv("KINGAI_SOCKET", socket)
	t.Setenv("KINGAI_DESKTOP_PRIVATE_STATUS", output)
	if err := runOnce(); err != nil { t.Fatal(err) }

	b, err := os.ReadFile(output)
	if err != nil { t.Fatal(err) }
	var got privateSnapshot
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.Schema != snapshotSchema || got.Product != "KINGAI OS Desktop" || got.UserUID != uint32(os.Geteuid()) { t.Fatalf("unexpected snapshot: %#v", got) }
	if len(got.Tasks) != 1 || got.Tasks[0].ID != "mine" || got.Tasks[0].Goal != "user task" { t.Fatalf("unexpected tasks: %#v", got.Tasks) }
	if strings.Contains(string(b), "/private/target") || strings.Contains(string(b), "private title") { t.Fatalf("raw step content leaked: %s", b) }
	st, err := os.Stat(output)
	if err != nil { t.Fatal(err) }
	if st.Mode().Perm() != 0o600 { t.Fatalf("snapshot mode=%o want 600", st.Mode().Perm()) }
	parent, err := os.Stat(filepath.Dir(output))
	if err != nil { t.Fatal(err) }
	if parent.Mode().Perm() != 0o700 { t.Fatalf("snapshot dir mode=%o want 700", parent.Mode().Perm()) }
}

func TestPrivateSnapshotPathRejectsRelativeOverride(t *testing.T) {
	t.Setenv("KINGAI_DESKTOP_PRIVATE_STATUS", "relative.json")
	if _, err := privateSnapshotPath(); err == nil { t.Fatal("relative private status path must be rejected") }
}
