package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/desktopbridge"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func TestDesktopPrivateEndpointIsPeerScopedAndRedacted(t *testing.T) {
	root := t.TempDir()
	tasks := taskgraph.Store{Root: root + "/tasks"}
	memories := memory.FileStore{Root: root + "/memory"}
	registry := agent.New([]agent.Definition{
		{ID:"main", Role:"assistant", Capabilities:[]string{"filesystem.read","network.read"}},
		{ID:"system-ops", Role:"privileged-system", Capabilities:[]string{"filesystem.write","service.restart"}},
	})

	mine, err := tasks.Create("my private goal", "main", 1000, []taskgraph.Step{{ID:"one", Title:"private step", Capability:"filesystem.write", Target:"/home/me/private", Status:taskgraph.StatusCreated}})
	if err != nil { t.Fatal(err) }
	if _, err := tasks.Create("other user goal", "main", 2000, nil); err != nil { t.Fatal(err) }
	if _, err := memories.Put("uid-1000", "semantic", "private", json.RawMessage(`{"secret":"memory-body"}`)); err != nil { t.Fatal(err) }
	if _, err := memories.Put("uid-2000", "semantic", "private", json.RawMessage(`{"secret":"other-memory"}`)); err != nil { t.Fatal(err) }

	mux := http.NewServeMux()
	registerDesktopPrivateHandler(mux, tasks, memories, "0.1-test", registry)
	req := httptest.NewRequest(http.MethodGet, "/v1/desktop/private", nil)
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(1000)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK { t.Fatalf("status=%d body=%s", w.Code, w.Body.String()) }

	var snapshot desktopbridge.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil { t.Fatal(err) }
	if snapshot.UserUID != 1000 || len(snapshot.Tasks) != 1 || snapshot.Tasks[0].ID != mine.ID || snapshot.Tasks[0].Goal != "my private goal" {
		t.Fatalf("unexpected private snapshot: %#v", snapshot)
	}
	if snapshot.Memory.Total != 1 || snapshot.Memory.ByLayer["M4"] != 1 { t.Fatalf("unexpected memory summary: %#v", snapshot.Memory) }
	if len(snapshot.Agents) != 2 { t.Fatalf("unexpected Agent summaries: %#v", snapshot.Agents) }
	if snapshot.Agents[0].ID != "main" || !snapshot.Agents[0].Authorized || snapshot.Agents[0].CapabilityCount != 2 { t.Fatalf("unexpected main Agent summary: %#v", snapshot.Agents[0]) }
	if snapshot.Agents[1].ID != "system-ops" || snapshot.Agents[1].Authorized || snapshot.Agents[1].CapabilityCount != 2 { t.Fatalf("privileged Agent must remain locked for ordinary peer: %#v", snapshot.Agents[1]) }

	text := strings.ToLower(w.Body.String())
	for _, forbidden := range []string{"other user goal", "private step", "filesystem.write", "filesystem.read", "network.read", "service.restart", "/home/me/private", "memory-body", "other-memory", "approval_id", "result"} {
		if strings.Contains(text, strings.ToLower(forbidden)) { t.Fatalf("desktop endpoint leaked %q: %s", forbidden, text) }
	}
}

func TestDesktopPrivateEndpointRejectsMissingPeerIdentity(t *testing.T) {
	mux := http.NewServeMux()
	registerDesktopPrivateHandler(mux, taskgraph.Store{Root:t.TempDir()}, memory.FileStore{Root:t.TempDir()}, "test")
	req := httptest.NewRequest(http.MethodGet, "/v1/desktop/private", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden { t.Fatalf("status=%d want 403", w.Code) }
}

func TestDesktopPrivateEndpointIsReadOnly(t *testing.T) {
	mux := http.NewServeMux()
	registerDesktopPrivateHandler(mux, taskgraph.Store{Root:t.TempDir()}, memory.FileStore{Root:t.TempDir()}, "test")
	req := httptest.NewRequest(http.MethodPost, "/v1/desktop/private", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), peerKey{}, uint32(1000)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed { t.Fatalf("status=%d want 405", w.Code) }
}
