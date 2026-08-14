package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Health struct {
	OK        bool          `json:"ok"`
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency,omitempty"`
	Message   string        `json:"message,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

type ProviderStatus struct {
	ID                  string    `json:"id"`
	Health              Health    `json:"health"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Provider interface {
	ID() string
	Health(context.Context) Health
	Candidates(context.Context) ([]Candidate, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	status    map[string]ProviderStatus
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		status:    make(map[string]ProviderStatus),
	}
}

func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return errors.New("provider is required")
	}
	id := strings.TrimSpace(provider.ID())
	if !validProviderID(id) {
		return errors.New("invalid provider id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider already registered: %s", id)
	}
	r.providers[id] = provider
	now := time.Now().UTC()
	r.status[id] = ProviderStatus{
		ID: id,
		Health: Health{
			OK:        false,
			Status:    "registered",
			CheckedAt: now,
		},
		UpdatedAt: now,
	}
	return nil
}

func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) Status(id string) (ProviderStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status, ok := r.status[id]
	if !ok {
		return ProviderStatus{}, fmt.Errorf("provider not registered: %s", id)
	}
	return status, nil
}

func (r *Registry) Statuses() []ProviderStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ProviderStatus, 0, len(r.status))
	for _, status := range r.status {
		out = append(out, status)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Registry) Refresh(ctx context.Context, id string) (ProviderStatus, error) {
	provider, err := r.get(id)
	if err != nil {
		return ProviderStatus{}, err
	}
	health := normalizeProviderHealth(provider.Health(ctx))
	r.recordHealth(id, health, healthFailureMessage(health))
	return r.Status(id)
}

func (r *Registry) Health(ctx context.Context) map[string]Health {
	providers := r.snapshotProviders()
	ids := sortedProviderIDs(providers)
	out := make(map[string]Health, len(providers))
	for _, id := range ids {
		health := normalizeProviderHealth(providers[id].Health(ctx))
		r.recordHealth(id, health, healthFailureMessage(health))
		out[id] = health
	}
	return out
}

func (r *Registry) Candidates(ctx context.Context) ([]Candidate, error) {
	providers := r.snapshotProviders()
	ids := sortedProviderIDs(providers)
	out := make([]Candidate, 0)
	for _, id := range ids {
		provider := providers[id]
		health := normalizeProviderHealth(provider.Health(ctx))
		if !health.OK {
			r.recordHealth(id, health, healthFailureMessage(health))
			continue
		}

		candidates, err := provider.Candidates(ctx)
		if err != nil {
			r.recordFailure(id, health, err.Error())
			continue
		}
		r.recordHealth(id, health, "")
		for _, candidate := range candidates {
			if candidate.ID == "" {
				continue
			}
			if candidate.Provider == "" {
				candidate.Provider = id
			}
			candidate.Available = candidate.Available && health.OK
			if candidate.LatencyMS == 0 && health.Latency > 0 {
				candidate.LatencyMS = int(health.Latency.Milliseconds())
			}
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		return []Candidate{}, ErrNoModel
	}
	return out, nil
}

func (r *Registry) Select(ctx context.Context, req Request) (Candidate, error) {
	candidates, err := r.Candidates(ctx)
	if err != nil {
		return Candidate{}, err
	}
	return Select(req, candidates)
}

func (r *Registry) get(id string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider not registered: %s", id)
	}
	return provider, nil
}

func (r *Registry) snapshotProviders() map[string]Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	providers := make(map[string]Provider, len(r.providers))
	for id, provider := range r.providers {
		providers[id] = provider
	}
	return providers
}

func (r *Registry) recordHealth(id string, health Health, lastError string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	previous := r.status[id]
	failures := 0
	if !health.OK {
		failures = previous.ConsecutiveFailures + 1
	}
	r.status[id] = ProviderStatus{
		ID:                  id,
		Health:              health,
		ConsecutiveFailures: failures,
		LastError:           strings.TrimSpace(lastError),
		UpdatedAt:           time.Now().UTC(),
	}
}

func (r *Registry) recordFailure(id string, health Health, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	previous := r.status[id]
	health.OK = false
	if strings.TrimSpace(health.Status) == "" || health.Status == "ok" {
		health.Status = "provider_error"
	}
	health.Message = strings.TrimSpace(message)
	if health.CheckedAt.IsZero() {
		health.CheckedAt = time.Now().UTC()
	}
	r.status[id] = ProviderStatus{
		ID:                  id,
		Health:              health,
		ConsecutiveFailures: previous.ConsecutiveFailures + 1,
		LastError:           strings.TrimSpace(message),
		UpdatedAt:           time.Now().UTC(),
	}
}

func normalizeProviderHealth(health Health) Health {
	health.Status = strings.TrimSpace(health.Status)
	health.Message = strings.TrimSpace(health.Message)
	if health.CheckedAt.IsZero() {
		health.CheckedAt = time.Now().UTC()
	}
	if health.Status == "" {
		if health.OK {
			health.Status = "ok"
		} else {
			health.Status = "unhealthy"
		}
	}
	return health
}

func healthFailureMessage(health Health) string {
	if health.OK {
		return ""
	}
	if health.Message != "" {
		return health.Message
	}
	return health.Status
}

func sortedProviderIDs(providers map[string]Provider) []string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func validProviderID(v string) bool {
	if v == "" || len(v) > 64 || strings.HasPrefix(v, "-") {
		return false
	}
	for _, ch := range v {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') {
			return false
		}
	}
	return true
}
