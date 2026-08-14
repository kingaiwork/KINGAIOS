package model

import (
	"context"
	"testing"
)

func TestStaticProviderHealthReflectsAvailability(t *testing.T) {
	p := NewStaticProvider("local", []Candidate{{ID: "local-chat", Available: true, Local: true}})
	health := p.Health(context.Background())
	if !health.OK || health.Status != "ok" { t.Fatalf("health=%#v", health) }

	p = NewStaticProvider("local", []Candidate{{ID: "local-chat", Available: false, Local: true}})
	health = p.Health(context.Background())
	if health.OK || health.Status != "no_available_models" { t.Fatalf("health=%#v", health) }
}

func TestRegistryFromCandidatesGroupsProviders(t *testing.T) {
	registry, err := RegistryFromCandidates([]Candidate{
		{ID: "local-chat", Provider: "local", Local: true, Available: true, Capabilities: []string{"chat"}},
		{ID: "cloud-chat", Provider: "cloud", Available: true, Capabilities: []string{"chat"}},
		{ID: "fallback", Available: true, Capabilities: []string{"chat"}},
	})
	if err != nil { t.Fatal(err) }
	ids := registry.IDs()
	if len(ids) != 3 || ids[0] != "cloud" || ids[1] != "configured" || ids[2] != "local" { t.Fatalf("ids=%v", ids) }
	selected, err := registry.Select(context.Background(), Request{Capability: "chat", Private: true})
	if err != nil { t.Fatal(err) }
	if selected.ID != "local-chat" { t.Fatalf("selected=%#v", selected) }
}

func TestRegistryFromCandidatesEmptyFailsClosed(t *testing.T) {
	registry, err := RegistryFromCandidates(nil)
	if err != nil { t.Fatal(err) }
	if len(registry.IDs()) != 0 { t.Fatalf("ids=%v", registry.IDs()) }
	if _, err := registry.Select(context.Background(), Request{Capability: "chat"}); err != ErrNoModel { t.Fatalf("err=%v", err) }
}

func TestStaticProviderFillsMissingProviderID(t *testing.T) {
	p := NewStaticProvider("configured", []Candidate{{ID: "x", Available: true}})
	items, err := p.Candidates(context.Background())
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Provider != "configured" { t.Fatalf("items=%#v", items) }
}
