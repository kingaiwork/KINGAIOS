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

type Provider interface {
	ID() string
	Health(context.Context) Health
	Candidates(context.Context) ([]Candidate, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry { return &Registry{providers: make(map[string]Provider)} }

func (r *Registry) Register(provider Provider) error {
	if provider == nil { return errors.New("provider is required") }
	id := strings.TrimSpace(provider.ID())
	if !validProviderID(id) { return errors.New("invalid provider id") }
	r.mu.Lock(); defer r.mu.Unlock()
	if _, exists := r.providers[id]; exists { return fmt.Errorf("provider already registered: %s", id) }
	r.providers[id] = provider
	return nil
}

func (r *Registry) IDs() []string {
	r.mu.RLock(); defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers { ids = append(ids, id) }
	sort.Strings(ids)
	return ids
}

func (r *Registry) Health(ctx context.Context) map[string]Health {
	r.mu.RLock(); providers := make(map[string]Provider, len(r.providers)); for id, p := range r.providers { providers[id] = p }; r.mu.RUnlock()
	out := make(map[string]Health, len(providers))
	for id, provider := range providers {
		health := provider.Health(ctx)
		if health.CheckedAt.IsZero() { health.CheckedAt = time.Now().UTC() }
		out[id] = health
	}
	return out
}

func (r *Registry) Candidates(ctx context.Context) ([]Candidate, error) {
	r.mu.RLock(); providers := make(map[string]Provider, len(r.providers)); for id, p := range r.providers { providers[id] = p }; r.mu.RUnlock()
	ids := make([]string, 0, len(providers)); for id := range providers { ids = append(ids, id) }; sort.Strings(ids)
	out := make([]Candidate, 0)
	for _, id := range ids {
		provider := providers[id]
		health := provider.Health(ctx)
		if !health.OK { continue }
		candidates, err := provider.Candidates(ctx)
		if err != nil { continue }
		for _, candidate := range candidates {
			if candidate.ID == "" { continue }
			if candidate.Provider == "" { candidate.Provider = id }
			candidate.Available = candidate.Available && health.OK
			if candidate.LatencyMS == 0 && health.Latency > 0 { candidate.LatencyMS = int(health.Latency.Milliseconds()) }
			out = append(out, candidate)
		}
	}
	if len(out) == 0 { return []Candidate{}, ErrNoModel }
	return out, nil
}

func (r *Registry) Select(ctx context.Context, req Request) (Candidate, error) {
	candidates, err := r.Candidates(ctx)
	if err != nil { return Candidate{}, err }
	return Select(req, candidates)
}

func validProviderID(v string) bool {
	if v == "" || len(v) > 64 || strings.HasPrefix(v, "-") { return false }
	for _, ch := range v {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_') { return false }
	}
	return true
}
