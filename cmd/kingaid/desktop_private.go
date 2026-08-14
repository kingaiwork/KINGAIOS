package main

import (
	"net/http"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/desktopbridge"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func registerDesktopPrivateHandler(mux *http.ServeMux, taskStore taskgraph.Store, memoryStore memory.FileStore, buildVersion string) {
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
		snapshot := desktopbridge.Build(uid, buildVersion, tasks, memorySummary, time.Now().UTC())
		writeJSON(w, http.StatusOK, snapshot)
	})
}
