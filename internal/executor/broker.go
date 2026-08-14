package executor

import (
	"context"
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

type Request struct {
	Agent      string          `json:"agent"`
	Capability string          `json:"capability"`
	Target     string          `json:"target,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
}

type Result struct {
	Capability string          `json:"capability"`
	Target     string          `json:"target,omitempty"`
	OK         bool            `json:"ok"`
	Data       json.RawMessage `json:"data,omitempty"`
	Message    string          `json:"message,omitempty"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at"`
}

type Handler interface {
	Execute(context.Context, Request) (Result, error)
}

type HandlerFunc func(context.Context, Request) (Result, error)

func (f HandlerFunc) Execute(ctx context.Context, req Request) (Result, error) {
	return f(ctx, req)
}

type Broker struct {
	mu      sync.RWMutex
	handlers map[string]Handler
	timeout time.Duration
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
	if req.Agent == "" || !validCapability(req.Capability) || strings.ContainsRune(req.Target, '\x00') {
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

	execCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()
	started := time.Now().UTC()
	result, err := handler.Execute(execCtx, req)
	finished := time.Now().UTC()
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
	if err != nil {
		result.OK = false
		return result, err
	}
	result.OK = true
	return result, nil
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
