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

	"github.com/kingaiwork/KINGAIOS/internal/audit"
	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

var version = "0.1.0-dev"
type peerKey struct{}

func main() {
	socket := getenv("KINGAI_EXECD_SOCKET", "/run/kingai-execd/execd.sock")
	allowedUser := getenv("KINGAI_EXECD_ALLOWED_USER", "_kingai")
	auditPath := getenv("KINGAI_EXECD_AUDIT", "/var/log/kingai-execd/audit.jsonl")
	allowedUID, err := lookupUID(allowedUser)
	if err != nil { log.Fatalf("resolve allowed caller: %v", err) }

	broker := executor.New(30 * time.Second)
	if err := (executor.NativeHandlers{}).Register(broker); err != nil { log.Fatal(err) }

	if !filepath.IsAbs(socket) { log.Fatal("KINGAI_EXECD_SOCKET must be absolute") }
	dir := filepath.Dir(socket)
	if err := os.MkdirAll(dir, 0o711); err != nil { log.Fatal(err) }
	if err := os.Chmod(dir, 0o711); err != nil { log.Fatal(err) }
	_ = os.Remove(socket)
	ln, err := net.Listen("unix", socket)
	if err != nil { log.Fatal(err) }
	defer ln.Close()
	if err := os.Chown(socket, int(allowedUID), 0); err != nil { log.Fatal(err) }
	if err := os.Chmod(socket, 0o600); err != nil { log.Fatal(err) }

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		if peerUID(r.Context()) != allowedUID { http.Error(w, "caller not authorized", http.StatusForbidden); return }
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "kingai-execd", "version": version, "capabilities": broker.Capabilities()})
	})
	mux.HandleFunc("/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context())
		if uid != allowedUID { http.Error(w, "caller not authorized", http.StatusForbidden); return }
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var req executor.Request
		if err := dec.Decode(&req); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return }
		result, execErr := broker.Execute(r.Context(), req)
		reason := result.Message
		if execErr != nil && reason == "" { reason = execErr.Error() }
		if err := audit.Append(auditPath, audit.Event{Type: "execution.run", Agent: req.Agent, Capability: req.Capability, Allowed: execErr == nil && result.OK, PeerUID: uid, TargetHash: audit.HashTarget(req.Target), Reason: reason}); err != nil {
			log.Printf("execution audit append failed: %v", err)
		}
		writeJSON(w, http.StatusOK, result)
	})

	srv := &http.Server{
		Handler: mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout: 30 * time.Second,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context { return context.WithValue(ctx, peerKey{}, unixPeerUID(c)) },
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatalf("serve: %v", err) }
	}()
	log.Printf("kingai-execd %s listening on unix:%s for uid=%d", version, socket, allowedUID)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func lookupUID(name string) (uint32, error) {
	u, err := user.Lookup(name)
	if err != nil { return 0, err }
	id, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil { return 0, err }
	return uint32(id), nil
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
