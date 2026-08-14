package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var version = "dev"

const (
	defaultBind = "127.0.0.1:9443"
	webRoot     = "/usr/share/kingai/sentinel/web"
	feedRoot    = "/var/lib/kingai/sentinel/feeds"
	scopePath   = "/etc/kingai/sentinel-scope.json"
)

type feedState struct {
	ID      string `json:"id"`
	Present bool   `json:"present"`
	Size    int64  `json:"size_bytes,omitempty"`
	Updated string `json:"updated_at,omitempty"`
}

type scopeFile struct {
	ExpiresAt string            `json:"expires_at"`
	Assets    []json.RawMessage `json:"assets"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func uptimeSeconds() int64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(f)
}

func feedStates() []feedState {
	files := []struct {
		id   string
		name string
	}{
		{"cisa-kev", "cisa-kev.json"},
		{"first-epss", "first-epss.csv.gz"},
		{"nvd-kev", "nvd-kev.json"},
	}
	out := make([]feedState, 0, len(files))
	for _, f := range files {
		s := feedState{ID: f.id}
		if info, err := os.Stat(filepath.Join(feedRoot, f.name)); err == nil {
			s.Present = true
			s.Size = info.Size()
			s.Updated = info.ModTime().UTC().Format(time.RFC3339)
		}
		out = append(out, s)
	}
	return out
}

func authState() map[string]any {
	state := map[string]any{
		"mode":                "authorized-assets-only",
		"default":             "deny",
		"scope_present":       false,
		"asset_count":         0,
		"privileged_actions":  "approval-required",
		"unknown_targets":     "deny",
	}
	b, err := os.ReadFile(scopePath)
	if err != nil {
		return state
	}
	var scope scopeFile
	if json.Unmarshal(b, &scope) != nil {
		state["scope_error"] = "invalid-json"
		return state
	}
	state["scope_present"] = true
	state["asset_count"] = len(scope.Assets)
	if scope.ExpiresAt != "" {
		state["expires_at"] = scope.ExpiresAt
		if t, err := time.Parse(time.RFC3339, scope.ExpiresAt); err == nil {
			state["expired"] = time.Now().After(t)
		}
	}
	return state
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func main() {
	bind := os.Getenv("KINGAI_SENTINEL_BIND")
	if bind == "" {
		bind = defaultBind
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "kingai-sentinel", "version": version})
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, _ *http.Request) {
		host, _ := os.Hostname()
		feeds := feedStates()
		ready := 0
		for _, f := range feeds {
			if f.Present {
				ready++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"product":          "KINGAI OS Sentinel",
			"version":          version,
			"hostname":         host,
			"uptime_seconds":   uptimeSeconds(),
			"mode":             "defensive",
			"ai_runtime":       "kingaid",
			"dashboard_bind":   bind,
			"feeds_ready":      ready,
			"feeds_total":      len(feeds),
			"execution_policy": "governed-sandbox",
		})
	})
	mux.HandleFunc("/api/feeds", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, feedStates())
	})
	mux.HandleFunc("/api/authorization", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, authState())
	})
	mux.Handle("/", http.FileServer(http.Dir(webRoot)))

	srv := &http.Server{
		Addr:              bind,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("KINGAI OS Sentinel console listening on %s", bind)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Errorf("sentinel console: %w", err))
	}
}
