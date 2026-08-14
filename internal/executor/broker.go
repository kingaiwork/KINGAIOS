package executor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnknownCapability = errors.New("execution capability is not registered")
	ErrInvalidRequest     = errors.New("invalid execution request")
)

const (
	MaxAgentBytes     = 128
	MaxTargetBytes    = 4096
	MaxArgumentsBytes = 32 << 10
)

type Request struct {
	Agent      string          `json:"agent"`
	Capability string          `json:"capability"`
	Target     string          `json:"target,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

type Result struct {
	ExecutionID string          `json:"execution_id"`
	Agent       string          `json:"agent"`
	Capability  string          `json:"capability"`
	Target      string          `json:"target,omitempty"`
	OK          bool            `json:"ok"`
	Data        json.RawMessage `json:"data,omitempty"`
	Message     string          `json:"message,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	FinishedAt  time.Time       `json:"finished_at"`
	DurationMS  int64           `json:"duration_ms"`
}

type Handler interface {
	Execute(context.Context, Request) (Result, error)
}

type HandlerFunc func(context.Context, Request) (Result, error)

func (f HandlerFunc) Execute(ctx context.Context, req Request) (Result, error) {
	return f(ctx, req)
}

type Broker struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	timeout  time.Duration
}

func New(timeout time.Duration) *Broker {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout > 10*time.Minute {
		timeout = 10 * time.Minute
	}
	return &Broker{handlers: make(map[string]Handler), timeout: timeout}
}

func (b *Broker) Register(capability string, handler Handler) error {
	capability = strings.TrimSpace(capability)
	if !validCapability(capability) || handler == nil {
		return ErrInvalidRequest
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, exists := b.handlers[capability]; exists {
		return fmt.Errorf("capability already registered: %s", capability)
	}
	b.handlers[capability] = handler
	return nil
}

func (b *Broker) Capabilities() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.handlers))
	for capability := range b.handlers {
		out = append(out, capability)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (b *Broker) Execute(ctx context.Context, req Request) (Result, error) {
	req.Agent = strings.TrimSpace(req.Agent)
	req.Capability = strings.TrimSpace(req.Capability)
	if req.Agent == "" || len(req.Agent) > MaxAgentBytes ||
		!validCapability(req.Capability) || len(req.Target) > MaxTargetBytes ||
		strings.ContainsRune(req.Target, '\x00') || len(req.Arguments) > MaxArgumentsBytes {
		return Result{}, ErrInvalidRequest
	}
	if len(req.Arguments) > 0 && !json.Valid(req.Arguments) {
		return Result{}, ErrInvalidRequest
	}

	b.mu.RLock()
	handler, ok := b.handlers[req.Capability]
	b.mu.RUnlock()
	if !ok {
		return Result{}, ErrUnknownCapability
	}

	executionID, err := newExecutionID()
	if err != nil {
		return Result{}, fmt.Errorf("create execution id: %w", err)
	}
	execCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	started := time.Now().UTC()
	result, execErr := handler.Execute(execCtx, req)
	finished := time.Now().UTC()
	result.ExecutionID = executionID
	result.Agent = req.Agent
	if result.Capability == "" {
		result.Capability = req.Capability
	}
	if result.Target == "" {
		result.Target = req.Target
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = started
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = finished
	}
	result.DurationMS = finished.Sub(started).Milliseconds()
	if execErr != nil {
		result.OK = false
		return result, execErr
	}
	result.OK = true
	return result, nil
}

func newExecutionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func validCapability(v string) bool {
	if v == "" || len(v) > 128 || strings.HasPrefix(v, ".") || strings.HasSuffix(v, ".") || strings.Contains(v, "..") {
		return false
	}
	for _, r := range v {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
