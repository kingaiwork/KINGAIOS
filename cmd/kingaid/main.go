package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/audit"
	"github.com/kingaiwork/KINGAIOS/internal/policy"
)

var version = "0.1.0-dev"
type peerKey struct{}

func main() {
	socket := getenv("KINGAI_SOCKET", "/run/kingai/kingaid.sock")
	policyPath := getenv("KINGAI_POLICY", "/etc/kingai/policy.json")
	agentPath := getenv("KINGAI_AGENTS", "/etc/kingai/agents.json")
	auditPath := getenv("KINGAI_AUDIT", "/var/log/kingai/audit.jsonl")

	p := policy.Default()
	if loaded, err := policy.Load(policyPath); err == nil {
		p = loaded
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("policy load failed, using safe defaults: %v", err)
	}
	registry := agent.Default()
	if loaded, err := agent.Load(agentPath); err == nil {
		registry = loaded
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("agent registry load failed, using safe fallback: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil { log.Fatal(err) }
	_ = os.Remove(socket)
	ln, err := net.Listen("unix", socket)
	if err != nil { log.Fatal(err) }
	defer ln.Close()
	if err := os.Chmod(socket, 0o660); err != nil { log.Fatal(err) }

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "kingaid", "version": version})
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		writeJSON(w, http.StatusOK, map[string]any{"name": "KINGAI OS", "architecture": "D4", "local_first": true, "policy": "enabled", "agent_registry": "enabled", "audit": "enabled", "version": version})
	})
	mux.HandleFunc("/v1/policy/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var req policy.Request
		if err := dec.Decode(&req); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return }

		uid := peerUID(r.Context())
		username := usernameForUID(uid)
		var res policy.Result
		if !agentIdentityAllowed(req.Agent, username, uid) {
			res = policy.Result{Reason: "agent identity is not authorized for this local peer"}
		} else if !registry.Has(req.Agent) {
			res = policy.Result{Reason: "unknown agent: default deny"}
		} else if !registry.Allows(req.Agent, req.Capability) {
			res = policy.Result{Reason: "capability not declared by agent manifest"}
		} else {
			req.Owner = uid == 0
			req.Approved = false // Future approval broker supplies cryptographically bound approvals.
			res = p.Evaluate(req)
		}
		if err := audit.Append(auditPath, audit.Event{Type: "policy.evaluate", Agent: req.Agent, Capability: req.Capability, Allowed: res.Allowed, ApprovalRequired: res.ApprovalRequired, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(req.Target), Reason: res.Reason}); err != nil {
			log.Printf("audit append failed: %v", err)
		}
		writeJSON(w, http.StatusOK, res)
	})

	srv := &http.Server{
		Handler: mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout: 30 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, peerKey{}, unixPeerUID(c))
		},
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatalf("serve: %v", err) }
	}()
	log.Printf("kingaid %s listening on unix:%s", version, socket)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func agentIdentityAllowed(requested, username string, uid uint32) bool {
	if requested == "main" { return true }
	if uid == 0 { return requested == "system-ops" || requested == "sec-ops" }
	switch requested {
	case "system-ops": return username == "_kingai-system"
	case "sec-ops": return username == "_kingai-sec"
	default: return false
	}
}

func usernameForUID(uid uint32) string {
	if uid == ^uint32(0) { return "" }
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil { return "" }
	return u.Username
}

func unixPeerUID(c net.Conn) uint32 {
	uc, ok := c.(*net.UnixConn)
	if !ok { return ^uint32(0) }
	raw, err := uc.SyscallConn()
	if err != nil { return ^uint32(0) }
	uid := ^uint32(0)
	_ = raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err == nil { uid = cred.Uid }
	})
	return uid
}
func peerUID(ctx context.Context) uint32 { if uid, ok := ctx.Value(peerKey{}).(uint32); ok { return uid }; return ^uint32(0) }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(code); _ = json.NewEncoder(w).Encode(v) }
func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
