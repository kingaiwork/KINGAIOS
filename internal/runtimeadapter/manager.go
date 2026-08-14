package runtimeadapter

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

var (
	ErrUnknownAdapter = errors.New("runtime adapter is not registered")
	ErrNoAdapter      = errors.New("no runtime adapter supports the capability")
)

type Health struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Lifecycle string

const (
	LifecycleRegistered Lifecycle = "registered"
	LifecycleStarting   Lifecycle = "starting"
	LifecycleHealthy    Lifecycle = "healthy"
	LifecycleDegraded   Lifecycle = "degraded"
	LifecycleStopping   Lifecycle = "stopping"
	LifecycleStopped    Lifecycle = "stopped"
)

type Status struct {
	ID        string    `json:"id"`
	State     Lifecycle `json:"state"`
	Health    Health    `json:"health"`
	LastError string    `json:"last_error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Adapter interface {
	ID() string
	Start(context.Context) error
	Stop(context.Context) error
	Health(context.Context) Health
	Capabilities(context.Context) ([]string, error)
	Execute(context.Context, executor.Request) (executor.Result, error)
	Cancel(context.Context, string) error
}

type Manager struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
	status   map[string]Status
}

func NewManager() *Manager {
	return &Manager{
		adapters: make(map[string]Adapter),
		status:   make(map[string]Status),
	}
}

func (m *Manager) Register(adapter Adapter) error {
	if adapter == nil {
		return errors.New("adapter is required")
	}
	id := strings.TrimSpace(adapter.ID())
	if !validID(id) {
		return errors.New("invalid adapter id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.adapters[id]; exists {
		return fmt.Errorf("adapter already registered: %s", id)
	}
	m.adapters[id] = adapter
	m.status[id] = Status{ID: id, State: LifecycleRegistered, UpdatedAt: time.Now().UTC()}
	return nil
}

func (m *Manager) IDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.adapters))
	for id := range m.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *Manager) Get(id string) (Adapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, ok := m.adapters[id]
	if !ok {
		return nil, ErrUnknownAdapter
	}
	return adapter, nil
}

func (m *Manager) Start(ctx context.Context, id string) error {
	adapter, err := m.Get(id)
	if err != nil {
		return err
	}
	m.setStatus(id, LifecycleStarting, Health{Status: "starting"}, "")
	if err := adapter.Start(ctx); err != nil {
		m.setStatus(id, LifecycleDegraded, Health{OK: false, Status: "start_failed", Message: err.Error()}, err.Error())
		return err
	}
	health := normalizeHealth(adapter.Health(ctx))
	if health.OK {
		m.setStatus(id, LifecycleHealthy, health, "")
	} else {
		m.setStatus(id, LifecycleDegraded, health, healthMessage(health))
	}
	return nil
}

func (m *Manager) Stop(ctx context.Context, id string) error {
	adapter, err := m.Get(id)
	if err != nil {
		return err
	}
	m.setStatus(id, LifecycleStopping, Health{Status: "stopping"}, "")
	if err := adapter.Stop(ctx); err != nil {
		m.setStatus(id, LifecycleDegraded, Health{OK: false, Status: "stop_failed", Message: err.Error()}, err.Error())
		return err
	}
	m.setStatus(id, LifecycleStopped, Health{OK: false, Status: "stopped"}, "")
	return nil
}

func (m *Manager) Refresh(ctx context.Context, id string) (Status, error) {
	adapter, err := m.Get(id)
	if err != nil {
		return Status{}, err
	}
	health := normalizeHealth(adapter.Health(ctx))
	if health.OK {
		m.setStatus(id, LifecycleHealthy, health, "")
	} else {
		m.setStatus(id, LifecycleDegraded, health, healthMessage(health))
	}
	return m.Status(id)
}

func (m *Manager) Status(id string) (Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.status[id]
	if !ok {
		return Status{}, ErrUnknownAdapter
	}
	return status, nil
}

func (m *Manager) Statuses() []Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Status, 0, len(m.status))
	for _, status := range m.status {
		out = append(out, status)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) Select(ctx context.Context, capability string) (Adapter, error) {
	capability = strings.TrimSpace(capability)
	m.mu.RLock()
	adapters := make([]Adapter, 0, len(m.adapters))
	for _, adapter := range m.adapters {
		adapters = append(adapters, adapter)
	}
	m.mu.RUnlock()

	sort.Slice(adapters, func(i, j int) bool { return adapters[i].ID() < adapters[j].ID() })
	for _, adapter := range adapters {
		health := normalizeHealth(adapter.Health(ctx))
		if !health.OK {
			m.setStatus(adapter.ID(), LifecycleDegraded, health, healthMessage(health))
			continue
		}
		m.setStatus(adapter.ID(), LifecycleHealthy, health, "")
		capabilities, err := adapter.Capabilities(ctx)
		if err != nil {
			m.setStatus(adapter.ID(), LifecycleDegraded, health, err.Error())
			continue
		}
		for _, supported := range capabilities {
			if supported == capability || supported == "*" {
				return adapter, nil
			}
		}
	}
	return nil, ErrNoAdapter
}

func (m *Manager) Execute(ctx context.Context, adapterID string, req executor.Request) (executor.Result, error) {
	var adapter Adapter
	var err error
	if strings.TrimSpace(adapterID) == "" {
		adapter, err = m.Select(ctx, req.Capability)
	} else {
		adapter, err = m.Get(adapterID)
		if err == nil {
			health := normalizeHealth(adapter.Health(ctx))
			if !health.OK {
				m.setStatus(adapter.ID(), LifecycleDegraded, health, healthMessage(health))
				return executor.Result{}, fmt.Errorf("adapter %s is unhealthy: %s", adapter.ID(), health.Status)
			}
			m.setStatus(adapter.ID(), LifecycleHealthy, health, "")
		}
	}
	if err != nil {
		return executor.Result{}, err
	}
	result, execErr := adapter.Execute(ctx, req)
	if execErr != nil {
		status, _ := m.Status(adapter.ID())
		m.setStatus(adapter.ID(), LifecycleDegraded, status.Health, execErr.Error())
	}
	return result, execErr
}

func (m *Manager) setStatus(id string, state Lifecycle, health Health, lastError string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status[id] = Status{
		ID:        id,
		State:     state,
		Health:    health,
		LastError: strings.TrimSpace(lastError),
		UpdatedAt: time.Now().UTC(),
	}
}

func normalizeHealth(health Health) Health {
	health.Status = strings.TrimSpace(health.Status)
	health.Message = strings.TrimSpace(health.Message)
	if health.Status == "" {
		if health.OK {
			health.Status = "ok"
		} else {
			health.Status = "unhealthy"
		}
	}
	return health
}

func healthMessage(health Health) string {
	if health.Message != "" {
		return health.Message
	}
	return health.Status
}

func validID(v string) bool {
	if v == "" || len(v) > 64 || strings.HasPrefix(v, "-") {
		return false
	}
	for _, r := range v {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
