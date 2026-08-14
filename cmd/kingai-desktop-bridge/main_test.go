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

	"github.com/kingaiwork/KINGAIOS/internal/desktopbridge"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func TestRunOnceWritesServerRedactedPrivateSnapshot(t *testing.T) {
	tmp := t.TempDir()
	socket := filepath.Join(tmp, "kingaid.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil { t.Fatal(err) }
	defer ln.Close()

	now := time.Now().UTC()
	serverSnapshot := desktopbridge.Snapshot{
		Schema: desktopbridge.Schema,
		Product: "KINGAI OS Desktop",
		Version: "0.1-test",
		UserUID: uint32(os.Geteuid()),
		Tasks: []desktopbridge.TaskSummary{{ID:"mine", Goal:"user task", Agent:"main", Status:taskgraph.StatusWaiting, StepCount:2, DoneSteps:1, UpdatedAt:now}},
		Memory: memory.Summary{Total:2, ByLayer:map[string]int{"M1":1,"M4":1}, BySensitivity:map[string]int{"private":2}},
		UpdatedAt: now,
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/desktop/private" { http.NotFound(w, r); return }
		_ = json.NewEncoder(w).Encode(serverSnapshot)
	})}
	go func() { _ = server.Serve(ln) }()
	defer server.Shutdown(context.Background())

	output := filepath.Join(tmp, "runtime", "kingai", "desktop-private.json")
	t.Setenv("KINGAI_SOCKET", socket)
	t.Setenv("KINGAI_DESKTOP_PRIVATE_STATUS", output)
	if err := runOnce(); err != nil { t.Fatal(err) }

	b, err := os.ReadFile(output)
	if err != nil { t.Fatal(err) }
	var got desktopbridge.Snapshot
	if err := json.Unmarshal(b, &got); err != nil { t.Fatal(err) }
	if got.Schema != desktopbridge.Schema || got.Product != "KINGAI OS Desktop" || got.UserUID != uint32(os.Geteuid()) { t.Fatalf("unexpected snapshot: %#v", got) }
	if len(got.Tasks) != 1 || got.Tasks[0].ID != "mine" || got.Tasks[0].Goal != "user task" { t.Fatalf("unexpected tasks: %#v", got.Tasks) }
	if got.Memory.Total != 2 || got.Memory.ByLayer["M4"] != 1 { t.Fatalf("unexpected memory: %#v", got.Memory) }
	if strings.Contains(string(b), "/v1/tasks/list") || strings.Contains(string(b), "approval_id") { t.Fatalf("unexpected raw runtime field in bridge output: %s", b) }
	st, err := os.Stat(output)
	if err != nil { t.Fatal(err) }
	if st.Mode().Perm() != 0o600 { t.Fatalf("snapshot mode=%o want 600", st.Mode().Perm()) }
	parent, err := os.Stat(filepath.Dir(output))
	if err != nil { t.Fatal(err) }
	if parent.Mode().Perm() != 0o700 { t.Fatalf("snapshot dir mode=%o want 700", parent.Mode().Perm()) }
}

func TestRunOnceRejectsSnapshotForAnotherUID(t *testing.T) {
	tmp := t.TempDir()
	socket := filepath.Join(tmp, "kingaid.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil { t.Fatal(err) }
	defer ln.Close()
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(desktopbridge.Snapshot{Schema:desktopbridge.Schema, Product:"KINGAI OS Desktop", UserUID:uint32(os.Geteuid()+1), Tasks:[]desktopbridge.TaskSummary{}, Memory:memory.Summary{}, UpdatedAt:time.Now().UTC()})
	})}
	go func() { _ = server.Serve(ln) }()
	defer server.Shutdown(context.Background())
	t.Setenv("KINGAI_SOCKET", socket)
	t.Setenv("KINGAI_DESKTOP_PRIVATE_STATUS", filepath.Join(tmp,"out.json"))
	if err := runOnce(); err == nil || !strings.Contains(err.Error(), "does not match bridge uid") { t.Fatalf("expected uid mismatch, got %v", err) }
}

func TestPrivateSnapshotPathRejectsRelativeOverride(t *testing.T) {
	t.Setenv("KINGAI_DESKTOP_PRIVATE_STATUS", "relative.json")
	if _, err := privateSnapshotPath(); err == nil { t.Fatal("relative private status path must be rejected") }
}
