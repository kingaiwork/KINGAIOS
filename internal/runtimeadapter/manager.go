package runtimeadapter

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

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
}

func NewManager() *Manager {
	return &Manager{adapters: make(map[string]Adapter)}
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
		health := adapter.Health(ctx)
		if !health.OK {
			continue
		}
		capabilities, err := adapter.Capabilities(ctx)
		if err != nil {
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
			health := adapter.Health(ctx)
			if !health.OK {
				return executor.Result{}, fmt.Errorf("adapter %s is unhealthy: %s", adapter.ID(), health.Status)
			}
		}
	}
	if err != nil {
		return executor.Result{}, err
	}
	return adapter.Execute(ctx, req)
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
