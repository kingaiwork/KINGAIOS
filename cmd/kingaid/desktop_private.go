package main

import (
	"net/http"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/desktopbridge"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func registerDesktopPrivateHandler(mux *http.ServeMux, taskStore taskgraph.Store, memoryStore memory.FileStore, buildVersion string, suppliedRegistry ...agent.Registry) {
	registry := agent.Default()
	if len(suppliedRegistry) > 0 {
		registry = suppliedRegistry[0]
	} else if loaded, err := agent.Load(getenv("KINGAI_AGENTS", "/etc/kingai/agents.json")); err == nil {
		registry = loaded
	}

	mux.HandleFunc("/v1/desktop/private", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		uid := peerUID(r.Context())
		if uid == invalidUID() {
			http.Error(w, "peer identity unavailable", http.StatusForbidden)
			return
		}
		tasks, err := taskStore.ListForPeer(uid, 100)
		if err != nil {
			http.Error(w, "unable to summarize tasks", http.StatusInternalServerError)
			return
		}
		memorySummary, err := memoryStore.Summarize(ownerForUID(uid))
		if err != nil {
			http.Error(w, "unable to summarize memory", http.StatusInternalServerError)
			return
		}
		username := usernameForUID(uid)
		snapshot := desktopbridge.Build(uid, buildVersion, tasks, memorySummary, time.Now().UTC())
		snapshot.Agents = desktopbridge.SummarizeAgents(registry.Definitions(), func(agentID string) bool {
			return agentIdentityAllowed(agentID, username, uid)
		})
		writeJSON(w, http.StatusOK, snapshot)
	})
}
