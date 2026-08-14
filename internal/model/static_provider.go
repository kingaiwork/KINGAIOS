package model

import (
	"context"
	"sort"
	"strings"
	"time"
)

// StaticProvider adapts configured Candidate records into the Provider contract.
// It performs no network I/O and adds no provider SDK dependency.
type StaticProvider struct {
	id         string
	candidates []Candidate
}

func NewStaticProvider(id string, candidates []Candidate) *StaticProvider {
	copied := append([]Candidate(nil), candidates...)
	for i := range copied {
		if strings.TrimSpace(copied[i].Provider) == "" {
			copied[i].Provider = id
		}
	}
	return &StaticProvider{id: strings.TrimSpace(id), candidates: copied}
}

func (p *StaticProvider) ID() string { return p.id }

func (p *StaticProvider) Health(context.Context) Health {
	available := 0
	for _, candidate := range p.candidates {
		if candidate.ID != "" && candidate.Available {
			available++
		}
	}
	if available == 0 {
		return Health{OK: false, Status: "no_available_models", Message: "no configured model is currently available", CheckedAt: time.Now().UTC()}
	}
	return Health{OK: true, Status: "ok", CheckedAt: time.Now().UTC()}
}

func (p *StaticProvider) Candidates(context.Context) ([]Candidate, error) {
	out := append([]Candidate(nil), p.candidates...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// RegistryFromCandidates groups configured candidates by provider ID. Candidates
// without an explicit provider are placed under the neutral "configured" ID.
// An empty candidate set produces an empty Registry and therefore fails closed.
func RegistryFromCandidates(candidates []Candidate) (*Registry, error) {
	registry := NewRegistry()
	groups := make(map[string][]Candidate)
	for _, candidate := range candidates {
		id := strings.TrimSpace(candidate.Provider)
		if id == "" {
			id = "configured"
		}
		groups[id] = append(groups[id], candidate)
	}
	ids := make([]string, 0, len(groups))
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := registry.Register(NewStaticProvider(id, groups[id])); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
