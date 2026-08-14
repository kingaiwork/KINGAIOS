package model

import (
	"context"
	"testing"
	"time"
)

type fakeProvider struct {
	id         string
	health     Health
	candidates []Candidate
}

func (f *fakeProvider) ID() string { return f.id }
func (f *fakeProvider) Health(context.Context) Health { return f.health }
func (f *fakeProvider) Candidates(context.Context) ([]Candidate, error) { return append([]Candidate(nil), f.candidates...), nil }

func TestProviderRegistrySelectsHealthyCandidate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeProvider{id: "local", health: Health{OK: true, Status: "ok", Latency: 5 * time.Millisecond}, candidates: []Candidate{{ID: "local-chat", Local: true, Available: true, Capabilities: []string{"chat"}, Priority: 5}}}); err != nil { t.Fatal(err) }
	if err := r.Register(&fakeProvider{id: "cloud", health: Health{OK: false, Status: "down"}, candidates: []Candidate{{ID: "cloud-chat", Available: true, Capabilities: []string{"chat"}, Priority: 100}}}); err != nil { t.Fatal(err) }
	selected, err := r.Select(context.Background(), Request{Capability: "chat", Private: true})
	if err != nil { t.Fatal(err) }
	if selected.ID != "local-chat" || selected.Provider != "local" || selected.LatencyMS != 5 { t.Fatalf("selected=%#v", selected) }
}

func TestProviderRegistryRejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{id: "local", health: Health{OK: true}}
	if err := r.Register(p); err != nil { t.Fatal(err) }
	if err := r.Register(p); err == nil { t.Fatal("expected duplicate provider rejection") }
}

func TestProviderRegistryFailsClosedWithoutHealthyCandidates(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeProvider{id: "cloud", health: Health{OK: false, Status: "down"}}); err != nil { t.Fatal(err) }
	if _, err := r.Select(context.Background(), Request{Capability: "chat"}); err != ErrNoModel { t.Fatalf("err=%v", err) }
}
