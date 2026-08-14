package edgehandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultSocketRoot = "/run/kingai-device"
	MaxBodyBytes      = 64 << 10
)

type Request struct {
	Agent      string          `json:"agent"`
	Capability string          `json:"capability"`
	Target     string          `json:"target,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

type Result struct {
	OK      bool            `json:"ok"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

type Handler interface {
	Execute(context.Context, Request) (Result, error)
}

type HandlerFunc func(context.Context, Request) (Result, error)
func (f HandlerFunc) Execute(ctx context.Context, req Request) (Result, error) { return f(ctx, req) }

type Config struct {
	SocketPath   string
	SocketOwner  int
	Capabilities map[string][]string
	Timeout      time.Duration
}

// Serve exposes one hardware handler over a private Unix socket. It provides a
// second exact capability/resource allowlist below the Device Broker so a bug
// or stale manifest cannot turn a board handler into a generic device API.
func Serve(ctx context.Context, cfg Config, handler Handler) error {
	if handler == nil { return errors.New("edge handler implementation is required") }
	if err := validateConfig(cfg); err != nil { return err }
	if cfg.Timeout <= 0 { cfg.Timeout = 20 * time.Second }
	if cfg.Timeout > 2*time.Minute { cfg.Timeout = 2*time.Minute }

	if err := prepareSocket(cfg.SocketPath); err != nil { return err }
	listener, err := net.Listen("unix", cfg.SocketPath)
	if err != nil { return err }
	defer func(){ _ = listener.Close(); _ = os.Remove(cfg.SocketPath) }()
	if err := os.Chmod(cfg.SocketPath, 0o600); err != nil { return err }
	if cfg.SocketOwner >= 0 {
		if err := os.Chown(cfg.SocketPath, cfg.SocketOwner, -1); err != nil { return fmt.Errorf("chown handler socket: %w", err) }
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w,"method not allowed",http.StatusMethodNotAllowed); return }
		writeJSON(w,http.StatusOK,map[string]any{"ok":true,"service":"kingai-edge-handler","capabilities":len(cfg.Capabilities)})
	})
	mux.HandleFunc("/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w,"method not allowed",http.StatusMethodNotAllowed); return }
		r.Body = http.MaxBytesReader(w,r.Body,MaxBodyBytes)
		dec := json.NewDecoder(r.Body); dec.DisallowUnknownFields()
		var req Request
		if err := dec.Decode(&req); err != nil { http.Error(w,"invalid request",http.StatusBadRequest); return }
		if err := authorize(cfg.Capabilities,req); err != nil { writeJSON(w,http.StatusForbidden,Result{OK:false,Message:err.Error()}); return }
		execCtx,cancel := context.WithTimeout(r.Context(),cfg.Timeout); defer cancel()
		result,err := handler.Execute(execCtx,req)
		if err != nil { result.OK=false; if strings.TrimSpace(result.Message)=="" { result.Message="handler execution failed" }; writeJSON(w,http.StatusBadGateway,result); return }
		result.OK=true
		writeJSON(w,http.StatusOK,result)
	})

	server := &http.Server{Handler:mux,ReadHeaderTimeout:3*time.Second,IdleTimeout:15*time.Second}
	errCh := make(chan error,1)
	go func(){ errCh <- server.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdown,cancel:=context.WithTimeout(context.Background(),5*time.Second); defer cancel()
		_ = server.Shutdown(shutdown)
		return nil
	case err := <-errCh:
		if errors.Is(err,http.ErrServerClosed) { return nil }
		return err
	}
}

func authorize(capabilities map[string][]string, req Request) error {
	if strings.TrimSpace(req.Agent)=="" || len(req.Agent)>128 { return errors.New("invalid agent") }
	if strings.TrimSpace(req.Capability)=="" || len(req.Capability)>128 || !strings.HasPrefix(req.Capability,"device.") || strings.ContainsAny(req.Capability,"* /\\;$`?[]{}\x00\n\r") {
		return errors.New("invalid device capability")
	}
	if len(req.Target)>4096 || strings.ContainsAny(req.Target,"\x00\n\r") { return errors.New("invalid target") }
	if len(req.Arguments)>32<<10 || (len(req.Arguments)>0 && !json.Valid(req.Arguments)) { return errors.New("invalid arguments") }
	resources,ok := capabilities[req.Capability]
	if !ok { return errors.New("capability is not implemented by this handler") }
	for _,resource := range resources { if resource==req.Target { return nil } }
	return errors.New("resource is not implemented by this handler")
}

func validateConfig(cfg Config) error {
	if !filepath.IsAbs(cfg.SocketPath) || filepath.Clean(cfg.SocketPath)!=cfg.SocketPath || filepath.Dir(cfg.SocketPath)!=DefaultSocketRoot || !strings.HasSuffix(cfg.SocketPath,".sock") || strings.ContainsAny(cfg.SocketPath,"\x00\n\r") {
		return errors.New("handler socket must be a clean .sock path directly under /run/kingai-device")
	}
	if len(cfg.Capabilities)==0 || len(cfg.Capabilities)>64 { return errors.New("handler must declare 1-64 capabilities") }
	for capability,resources := range cfg.Capabilities {
		if !strings.HasPrefix(capability,"device.") || strings.ContainsAny(capability,"* /\\;$`?[]{}\x00\n\r") || len(resources)==0 || len(resources)>32 { return errors.New("invalid handler capability declaration") }
		seen:=map[string]struct{}{}
		for _,resource:=range resources {
			if resource=="" || len(resource)>256 || strings.ContainsAny(resource,"*?[]{};$`\\\x00\n\r") { return errors.New("invalid handler resource declaration") }
			if _,exists:=seen[resource];exists{return errors.New("duplicate handler resource")};seen[resource]=struct{}{}
		}
	}
	return nil
}

func prepareSocket(path string) error {
	parent:=filepath.Dir(path)
	info,err:=os.Stat(parent); if err!=nil{return fmt.Errorf("handler socket root: %w",err)}
	if !info.IsDir() || info.Mode().Perm()&0o022!=0 { return errors.New("handler socket root must be a protected directory") }
	stat,ok:=info.Sys().(*syscall.Stat_t); if !ok || stat.Uid!=0 { return errors.New("handler socket root must be root-owned") }
	if existing,err:=os.Lstat(path);err==nil {
		if existing.Mode()&os.ModeSymlink!=0 || existing.Mode()&os.ModeSocket==0 { return errors.New("refusing to replace non-socket handler path") }
		if err:=os.Remove(path);err!=nil{return err}
	} else if !errors.Is(err,os.ErrNotExist) { return err }
	return nil
}

func writeJSON(w http.ResponseWriter,code int,value any){w.Header().Set("Content-Type","application/json");w.WriteHeader(code);_ = json.NewEncoder(w).Encode(value)}
