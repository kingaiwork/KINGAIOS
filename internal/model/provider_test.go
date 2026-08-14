package model

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvider struct {
	id         string
	health     Health
	candidates []Candidate
	candidateErr error
}

func (f *fakeProvider) ID() string { return f.id }
func (f *fakeProvider) Health(context.Context) Health { return f.health }
func (f *fakeProvider) Candidates(context.Context) ([]Candidate, error) {
	if f.candidateErr != nil { return nil, f.candidateErr }
	return append([]Candidate(nil), f.candidates...), nil
}

func TestProviderRegistryTracksRegisteredAndHealthyStatus(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{id: "local", health: Health{OK: true, Status: "ok"}}
	if err := r.Register(p); err != nil { t.Fatal(err) }
	status, err := r.Status("local")
	if err != nil { t.Fatal(err) }
	if status.Health.Status != "registered" || status.ConsecutiveFailures != 0 { t.Fatalf("status=%#v", status) }

	status, err = r.Refresh(context.Background(), "local")
	if err != nil { t.Fatal(err) }
	if !status.Health.OK || status.Health.Status != "ok" || status.ConsecutiveFailures != 0 || status.LastError != "" {
		t.Fatalf("status=%#v", status)
	}
}

func TestProviderRegistrySelectsHealthyCandidate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(&fakeProvider{id: "local", health: Health{OK: true, Status: "ok", Latency: 5 * time.Millisecond}, candidates: []Candidate{{ID: "local-chat", Local: true, Available: true, Capabilities: []string{"chat"}, Priority: 5}}}); err != nil { t.Fatal(err) }
	if err := r.Register(&fakeProvider{id: "cloud", health: Health{OK: false, Status: "down"}, candidates: []Candidate{{ID: "cloud-chat", Available: true, Capabilities: []string{"chat"}, Priority: 100}}}); err != nil { t.Fatal(err) }
	selected, err := r.Select(context.Background(), Request{Capability: "chat", Private: true})
	if err != nil { t.Fatal(err) }
	if selected.ID != "local-chat" || selected.Provider != "local" || selected.LatencyMS != 5 { t.Fatalf("selected=%#v", selected) }
	local, _ := r.Status("local")
	if local.ConsecutiveFailures != 0 || !local.Health.OK { t.Fatalf("local=%#v", local) }
	cloud, _ := r.Status("cloud")
	if cloud.ConsecutiveFailures != 1 || cloud.LastError != "down" { t.Fatalf("cloud=%#v", cloud) }
}

func TestProviderRegistryFailureCounterResetsAfterRecovery(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{id: "local", health: Health{OK: false, Status: "down", Message: "socket unavailable"}}
	if err := r.Register(p); err != nil { t.Fatal(err) }
	if _, err := r.Refresh(context.Background(), "local"); err != nil { t.Fatal(err) }
	if _, err := r.Refresh(context.Background(), "local"); err != nil { t.Fatal(err) }
	status, _ := r.Status("local")
	if status.ConsecutiveFailures != 2 || status.LastError != "socket unavailable" { t.Fatalf("status=%#v", status) }

	p.health = Health{OK: true, Status: "ok"}
	if _, err := r.Refresh(context.Background(), "local"); err != nil { t.Fatal(err) }
	status, _ = r.Status("local")
	if status.ConsecutiveFailures != 0 || status.LastError != "" || !status.Health.OK { t.Fatalf("status=%#v", status) }
}

func TestProviderRegistryCandidateFailureIsRecorded(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{id: "cloud", health: Health{OK: true, Status: "ok"}, candidateErr: errors.New("catalog unavailable")}
	if err := r.Register(p); err != nil { t.Fatal(err) }
	if _, err := r.Candidates(context.Background()); err != ErrNoModel { t.Fatalf("err=%v", err) }
	status, _ := r.Status("cloud")
	if status.ConsecutiveFailures != 1 || status.LastError != "catalog unavailable" || status.Health.Status != "provider_error" {
		t.Fatalf("status=%#v", status)
	}
}

func TestProviderRegistryStatusesAreSorted(t *testing.T) {
	r := NewRegistry()
	for _, id := range []string{"remote", "local", "edge"} {
		if err := r.Register(&fakeProvider{id: id}); err != nil { t.Fatal(err) }
	}
	statuses := r.Statuses()
	if len(statuses) != 3 || statuses[0].ID != "edge" || statuses[1].ID != "local" || statuses[2].ID != "remote" {
		t.Fatalf("statuses=%#v", statuses)
	}
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
