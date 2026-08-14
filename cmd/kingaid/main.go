package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/approval"
	"github.com/kingaiwork/KINGAIOS/internal/audit"
	"github.com/kingaiwork/KINGAIOS/internal/executor"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/model"
	"github.com/kingaiwork/KINGAIOS/internal/policy"
	"github.com/kingaiwork/KINGAIOS/internal/scheduler"
	"github.com/kingaiwork/KINGAIOS/internal/statuspub"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

var version = "0.1.0-dev"
type peerKey struct{}

type modelSummary struct {
	Strategy    string `json:"strategy"`
	DefaultMode string `json:"default_mode"`
}

type modelConfig struct {
	Strategy    string            `json:"strategy"`
	DefaultMode string            `json:"default_mode"`
	Providers   []model.Candidate `json:"providers"`
}

type policyEvaluateRequest struct {
	Agent      string `json:"agent"`
	Capability string `json:"capability"`
	Target     string `json:"target,omitempty"`
	ApprovalID string `json:"approval_id,omitempty"`
}

type executionRequest struct {
	Agent      string          `json:"agent"`
	Capability string          `json:"capability"`
	Target     string          `json:"target,omitempty"`
	ApprovalID string          `json:"approval_id,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

type approvalRequest struct {
	Agent      string `json:"agent"`
	Capability string `json:"capability"`
	Target     string `json:"target,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
}

type approvalDecisionRequest struct {
	ID     string `json:"id"`
	Action string `json:"action"`
}

type memoryPutRequest struct {
	Agent       string          `json:"agent"`
	Kind        string          `json:"kind,omitempty"`
	Sensitivity string          `json:"sensitivity,omitempty"`
	Data        json.RawMessage `json:"data"`
}

type idRequest struct { ID string `json:"id"` }

type taskCreateRequest struct {
	Goal  string           `json:"goal"`
	Agent string           `json:"agent"`
	Steps []taskgraph.Step `json:"steps,omitempty"`
}

type taskTransitionRequest struct {
	ID     string           `json:"id"`
	Status taskgraph.Status `json:"status"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  string           `json:"error,omitempty"`
}

type taskStepTransitionRequest struct {
	ID     string           `json:"id"`
	StepID string           `json:"step_id"`
	Status taskgraph.Status `json:"status"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  string           `json:"error,omitempty"`
}

func main() {
	socket := getenv("KINGAI_SOCKET", "/run/kingai/kingaid.sock")
	policyPath := getenv("KINGAI_POLICY", "/etc/kingai/policy.json")
	agentPath := getenv("KINGAI_AGENTS", "/etc/kingai/agents.json")
	modelPath := getenv("KINGAI_MODELS", "/etc/kingai/models.json")
	auditPath := getenv("KINGAI_AUDIT", "/var/log/kingai/audit.jsonl")
	publicStatusPath := getenv("KINGAI_PUBLIC_STATUS", "/run/kingai/public-status.json")
	approvalRoot := getenv("KINGAI_APPROVAL_ROOT", "/var/lib/kingai/approvals")
	memoryRoot := getenv("KINGAI_MEMORY_ROOT", "/var/lib/kingai/memory")
	taskRoot := getenv("KINGAI_TASK_ROOT", "/var/lib/kingai/tasks")
	execdSocket := getenv("KINGAI_EXECD_SOCKET", "/run/kingai-execd/execd.sock")

	p := policy.Default()
	if loaded, err := policy.Load(policyPath); err == nil { p = loaded } else if !errors.Is(err, os.ErrNotExist) { log.Printf("policy load failed, using safe defaults: %v", err) }
	registry := agent.Default()
	if loaded, err := agent.Load(agentPath); err == nil { registry = loaded } else if !errors.Is(err, os.ErrNotExist) { log.Printf("agent registry load failed, using safe fallback: %v", err) }
	modelCfg := loadModelConfig(modelPath)
	models := modelSummary{Strategy: modelCfg.Strategy, DefaultMode: modelCfg.DefaultMode}
	approvalStore := approval.Store{Root: approvalRoot}
	memoryStore := memory.FileStore{Root: memoryRoot}
	taskStore := taskgraph.Store{Root: taskRoot}
	execClient := executor.Client{Socket: execdSocket, Timeout: 35 * time.Second}
	taskScheduler := scheduler.Scheduler{
		Store: taskStore,
		Authorizer: scheduler.AuthorizerFunc(func(ctx context.Context, task taskgraph.Task, step taskgraph.Step, uid uint32) (scheduler.Authorization, error) {
			res := evaluatePolicy(p, registry, approvalStore, task.Agent, step.Capability, step.Target, step.ApprovalID, uid)
			if res.Allowed { return scheduler.Authorization{Allowed: true}, nil }
			if res.ApprovalRequired {
				approvalID := step.ApprovalID
				if approvalID == "" {
					created, err := approvalStore.Create(task.Agent, step.Capability, audit.HashTarget(step.Target), uid, 5*time.Minute)
					if err != nil { return scheduler.Authorization{}, err }
					approvalID = created.ID
					appendAudit(auditPath, audit.Event{Type: "approval.request", Agent: task.Agent, Capability: step.Capability, ApprovalRequired: true, PeerUID: uid, TargetHash: audit.HashTarget(step.Target), Reason: "created by task scheduler"})
				}
				return scheduler.Authorization{ApprovalRequired: true, ApprovalID: approvalID, Reason: res.Reason}, nil
			}
			return scheduler.Authorization{Reason: res.Reason}, nil
		}),
		Dispatcher: execClient,
	}

	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil { log.Fatal(err) }
	_ = os.Remove(socket)
	ln, err := net.Listen("unix", socket)
	if err != nil { log.Fatal(err) }
	defer ln.Close()
	if err := os.Chmod(socket, 0o660); err != nil { log.Fatal(err) }

	publishStatus := func(health string) {
		err := statuspub.Write(publicStatusPath, statuspub.Snapshot{Product: "KINGAI OS", Version: version, Architecture: "D5-preview", Health: health, Policy: "enabled", RegisteredAgents: registry.Count(), ModelStrategy: models.Strategy, ModelMode: models.DefaultMode, MemoryMode: "local-first", CloudRequired: false})
		if err != nil { log.Printf("public status publish failed: %v", err) }
	}
	publishStatus("ok")
	statusCtx, statusCancel := context.WithCancel(context.Background())
	defer statusCancel()
	go func() {
		t := time.NewTicker(5 * time.Second); defer t.Stop()
		for { select { case <-statusCtx.Done(): return; case <-t.C: publishStatus("ok") } }
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "kingaid", "version": version, "architecture": "D5-preview"})
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		execState := "offline"
		healthCtx, cancel := context.WithTimeout(r.Context(), 150*time.Millisecond)
		if execClient.Health(healthCtx) == nil { execState = "ready" }
		cancel()
		writeJSON(w, http.StatusOK, map[string]any{"name": "KINGAI OS", "architecture": "D5-preview", "local_first": true, "policy": "enabled", "approval_broker": "enabled", "task_graph": "enabled", "task_scheduler": "enabled", "memory_service": "enabled", "model_router": "enabled", "execution_broker": execState, "agent_registry": "enabled", "registered_agents": registry.Count(), "model_strategy": models.Strategy, "model_mode": models.DefaultMode, "model_candidates": len(modelCfg.Providers), "audit": "enabled", "version": version})
	})

	mux.HandleFunc("/v1/policy/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in policyEvaluateRequest
		if !decodeJSON(w, r, 64<<10, &in) { return }
		uid := peerUID(r.Context())
		res := evaluatePolicy(p, registry, approvalStore, in.Agent, in.Capability, in.Target, in.ApprovalID, uid)
		appendAudit(auditPath, audit.Event{Type: "policy.evaluate", Agent: in.Agent, Capability: in.Capability, Allowed: res.Allowed, ApprovalRequired: res.ApprovalRequired, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(in.Target), Reason: res.Reason})
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/v1/execution/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in executionRequest
		if !decodeJSON(w, r, 128<<10, &in) { return }
		uid := peerUID(r.Context())
		res := evaluatePolicy(p, registry, approvalStore, in.Agent, in.Capability, in.Target, in.ApprovalID, uid)
		if !res.Allowed {
			appendAudit(auditPath, audit.Event{Type: "execution.denied", Agent: in.Agent, Capability: in.Capability, Allowed: false, ApprovalRequired: res.ApprovalRequired, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(in.Target), Reason: res.Reason})
			writeJSON(w, http.StatusForbidden, res); return
		}
		if len(in.Arguments) > 0 && !json.Valid(in.Arguments) { http.Error(w, "arguments must be valid JSON", http.StatusBadRequest); return }
		result, err := execClient.Execute(r.Context(), executor.Request{Agent: in.Agent, Capability: in.Capability, Target: in.Target, Arguments: in.Arguments})
		if err != nil {
			appendAudit(auditPath, audit.Event{Type: "execution.dispatch", Agent: in.Agent, Capability: in.Capability, Allowed: false, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(in.Target), Reason: err.Error()})
			writeJSON(w, http.StatusBadGateway, result); return
		}
		appendAudit(auditPath, audit.Event{Type: "execution.dispatch", Agent: in.Agent, Capability: in.Capability, Allowed: true, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(in.Target), Reason: "executed by constrained broker"})
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("/v1/approval/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in approvalRequest
		if !decodeJSON(w, r, 64<<10, &in) { return }
		uid := peerUID(r.Context())
		if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		if !agentIdentityAllowed(in.Agent, usernameForUID(uid), uid) || !registry.Has(in.Agent) || !registry.Allows(in.Agent, in.Capability) { http.Error(w, "agent or capability is not authorized", http.StatusForbidden); return }
		base := policy.Request{Agent: in.Agent, Capability: in.Capability, Target: in.Target, Owner: uid == 0}
		res := p.Evaluate(base)
		if res.Allowed { http.Error(w, "approval is not required", http.StatusConflict); return }
		if !res.ApprovalRequired { http.Error(w, "request is denied by policy", http.StatusForbidden); return }
		if res.Reason == "owner authorization required" && uid != 0 { http.Error(w, "owner authorization required", http.StatusForbidden); return }
		created, err := approvalStore.Create(in.Agent, in.Capability, audit.HashTarget(in.Target), uid, time.Duration(in.TTLSeconds)*time.Second)
		if err != nil { http.Error(w, "unable to create approval", http.StatusInternalServerError); return }
		appendAudit(auditPath, audit.Event{Type: "approval.request", Agent: in.Agent, Capability: in.Capability, ApprovalRequired: true, Risk: int(res.Risk), PeerUID: uid, TargetHash: audit.HashTarget(in.Target), Reason: res.Reason})
		writeJSON(w, http.StatusCreated, created)
	})
	mux.HandleFunc("/v1/approval/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		if peerUID(r.Context()) != 0 { http.Error(w, "owner authorization required", http.StatusForbidden); return }
		items, err := approvalStore.List(100); if err != nil { http.Error(w, "unable to list approvals", http.StatusInternalServerError); return }; writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("/v1/approval/decision", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid != 0 { http.Error(w, "owner authorization required", http.StatusForbidden); return }
		var in approvalDecisionRequest; if !decodeJSON(w, r, 16<<10, &in) { return }
		approve := strings.EqualFold(in.Action, "approve"); if !approve && !strings.EqualFold(in.Action, "deny") { http.Error(w, "action must be approve or deny", http.StatusBadRequest); return }
		item, err := approvalStore.Decide(in.ID, approve, uid); if err != nil { http.Error(w, err.Error(), http.StatusConflict); return }
		appendAudit(auditPath, audit.Event{Type: "approval.decision", Agent: item.Agent, Capability: item.Capability, Allowed: approve, PeerUID: uid, TargetHash: item.TargetHash, Reason: string(item.Status)}); writeJSON(w, http.StatusOK, item)
	})

	mux.HandleFunc("/v1/memory/put", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in memoryPutRequest; if !decodeJSON(w, r, 256<<10, &in) { return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		if !agentIdentityAllowed(in.Agent, usernameForUID(uid), uid) || !registry.Has(in.Agent) { http.Error(w, "agent identity is not authorized", http.StatusForbidden); return }
		if len(in.Data) == 0 || !json.Valid(in.Data) { http.Error(w, "data must be valid JSON", http.StatusBadRequest); return }
		record, err := memoryStore.Put(ownerForUID(uid), in.Kind, in.Sensitivity, in.Data); if err != nil { http.Error(w, "unable to store memory", http.StatusInternalServerError); return }
		appendAudit(auditPath, audit.Event{Type: "memory.put", Agent: in.Agent, Allowed: true, PeerUID: uid}); writeJSON(w, http.StatusCreated, record)
	})
	mux.HandleFunc("/v1/memory/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		items, err := memoryStore.List(ownerForUID(uid), 100); if err != nil { http.Error(w, "unable to list memory", http.StatusInternalServerError); return }; writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("/v1/memory/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		var in idRequest; if !decodeJSON(w, r, 16<<10, &in) { return }
		if err := memoryStore.Delete(ownerForUID(uid), in.ID); err != nil { http.Error(w, "unable to delete memory", http.StatusBadRequest); return }
		appendAudit(auditPath, audit.Event{Type: "memory.delete", Allowed: true, PeerUID: uid}); writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	})

	mux.HandleFunc("/v1/model/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in model.Request; if !decodeJSON(w, r, 32<<10, &in) { return }
		selected, err := model.Select(in, modelCfg.Providers); if err != nil { http.Error(w, "no eligible model", http.StatusServiceUnavailable); return }; writeJSON(w, http.StatusOK, selected)
	})

	mux.HandleFunc("/v1/tasks/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		var in taskCreateRequest; if !decodeJSON(w, r, 128<<10, &in) { return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		if !agentIdentityAllowed(in.Agent, usernameForUID(uid), uid) || !registry.Has(in.Agent) { http.Error(w, "agent identity is not authorized", http.StatusForbidden); return }
		task, err := taskStore.Create(in.Goal, in.Agent, uid, in.Steps); if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
		appendAudit(auditPath, audit.Event{Type: "task.create", Agent: in.Agent, Allowed: true, PeerUID: uid}); writeJSON(w, http.StatusCreated, task)
	})
	mux.HandleFunc("/v1/tasks/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		items, err := taskStore.ListForPeer(uid, 100); if err != nil { http.Error(w, "unable to list tasks", http.StatusInternalServerError); return }; writeJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("/v1/tasks/transition", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		var in taskTransitionRequest; if !decodeJSON(w, r, 128<<10, &in) { return }
		task, err := taskStore.TransitionForPeer(in.ID, uid, in.Status, in.Result, in.Error); if err != nil { http.Error(w, err.Error(), http.StatusConflict); return }
		appendAudit(auditPath, audit.Event{Type: "task.transition", Agent: task.Agent, Allowed: true, PeerUID: uid, Reason: string(task.Status)}); writeJSON(w, http.StatusOK, task)
	})
	mux.HandleFunc("/v1/tasks/step/transition", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		var in taskStepTransitionRequest; if !decodeJSON(w, r, 128<<10, &in) { return }
		task, err := taskStore.TransitionStepForPeer(in.ID, uid, in.StepID, in.Status, in.Result, in.Error); if err != nil { http.Error(w, err.Error(), http.StatusConflict); return }
		appendAudit(auditPath, audit.Event{Type: "task.step.transition", Agent: task.Agent, Allowed: true, PeerUID: uid, Reason: in.StepID + ":" + string(in.Status)}); writeJSON(w, http.StatusOK, task)
	})
	mux.HandleFunc("/v1/tasks/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		uid := peerUID(r.Context()); if uid == invalidUID() { http.Error(w, "peer identity unavailable", http.StatusForbidden); return }
		var in idRequest; if !decodeJSON(w, r, 16<<10, &in) { return }
		task, err := taskScheduler.RunReady(r.Context(), in.ID, uid)
		if err != nil {
			appendAudit(auditPath, audit.Event{Type: "task.run", Agent: task.Agent, Allowed: false, PeerUID: uid, Reason: err.Error()}); writeJSON(w, http.StatusBadGateway, task); return
		}
		appendAudit(auditPath, audit.Event{Type: "task.run", Agent: task.Agent, Allowed: true, PeerUID: uid, Reason: string(task.Status)}); writeJSON(w, http.StatusOK, task)
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, ConnContext: func(ctx context.Context, c net.Conn) context.Context { return context.WithValue(ctx, peerKey{}, unixPeerUID(c)) }}
	go func() { if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatalf("serve: %v", err) } }()
	log.Printf("kingaid %s listening on unix:%s", version, socket)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM); defer stop(); <-ctx.Done(); statusCancel(); publishStatus("stopping")
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel(); _ = srv.Shutdown(shutdown)
}

func evaluatePolicy(p policy.Policy, registry agent.Registry, approvals approval.Store, agentID, capability, target, approvalID string, uid uint32) policy.Result {
	pReq := policy.Request{Agent: agentID, Capability: capability, Target: target, Owner: uid == 0}
	if uid == invalidUID() { return policy.Result{Reason: "unable to establish local peer identity"} }
	if !agentIdentityAllowed(agentID, usernameForUID(uid), uid) { return policy.Result{Reason: "agent identity is not authorized for this local peer"} }
	if !registry.Has(agentID) { return policy.Result{Reason: "unknown agent: default deny"} }
	if !registry.Allows(agentID, capability) { return policy.Result{Reason: "capability not declared by agent manifest"} }
	res := p.Evaluate(pReq)
	if approvalID != "" && res.ApprovalRequired {
		if _, err := approvals.Consume(approvalID, agentID, capability, audit.HashTarget(target), uid); err == nil { pReq.Approved = true; res = p.Evaluate(pReq) } else { res.Allowed = false; res.ApprovalRequired = true; res.Reason = "approval token rejected" }
	}
	return res
}

func loadModelConfig(path string) modelConfig {
	m := modelConfig{Strategy: "provider-neutral", DefaultMode: "hybrid", Providers: []model.Candidate{}}
	b, err := os.ReadFile(path); if err != nil { return m }
	var configured modelConfig; if json.Unmarshal(b, &configured) != nil { return m }
	if configured.Strategy != "" { m.Strategy = configured.Strategy }; if configured.DefaultMode != "" { m.DefaultMode = configured.DefaultMode }; if configured.Providers != nil { m.Providers = configured.Providers }; return m
}
func decodeJSON(w http.ResponseWriter, r *http.Request, max int64, out any) bool { r.Body = http.MaxBytesReader(w, r.Body, max); dec := json.NewDecoder(r.Body); dec.DisallowUnknownFields(); if err := dec.Decode(out); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return false }; return true }
func appendAudit(path string, event audit.Event) { if err := audit.Append(path, event); err != nil { log.Printf("audit append failed: %v", err) } }
func ownerForUID(uid uint32) string { return fmt.Sprintf("uid-%d", uid) }
func invalidUID() uint32 { return ^uint32(0) }
func agentIdentityAllowed(requested, username string, uid uint32) bool { if uid == invalidUID() { return false }; if requested == "main" { return true }; if uid == 0 { return requested == "system-ops" || requested == "sec-ops" }; switch requested { case "system-ops": return username == "_kingai-system"; case "sec-ops": return username == "_kingai-sec"; default: return false } }
func usernameForUID(uid uint32) string { if uid == invalidUID() { return "" }; u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); if err != nil { return "" }; return u.Username }
func unixPeerUID(c net.Conn) uint32 { uc, ok := c.(*net.UnixConn); if !ok { return invalidUID() }; raw, err := uc.SyscallConn(); if err != nil { return invalidUID() }; uid := invalidUID(); _ = raw.Control(func(fd uintptr) { cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED); if err == nil { uid = cred.Uid } }); return uid }
func peerUID(ctx context.Context) uint32 { if uid, ok := ctx.Value(peerKey{}).(uint32); ok { return uid }; return invalidUID() }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(code); _ = json.NewEncoder(w).Encode(v) }
func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
