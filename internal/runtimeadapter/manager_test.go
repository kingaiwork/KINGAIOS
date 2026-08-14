package runtimeadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

type fakeAdapter struct {
	id       string
	health   Health
	caps     []string
	startErr error
	stopErr  error
	capsErr  error
	execErr  error
}

func (f *fakeAdapter) ID() string { return f.id }
func (f *fakeAdapter) Start(context.Context) error { return f.startErr }
func (f *fakeAdapter) Stop(context.Context) error { return f.stopErr }
func (f *fakeAdapter) Health(context.Context) Health { return f.health }
func (f *fakeAdapter) Capabilities(context.Context) ([]string, error) {
	if f.capsErr != nil { return nil, f.capsErr }
	return append([]string(nil), f.caps...), nil
}
func (f *fakeAdapter) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	if f.execErr != nil { return executor.Result{}, f.execErr }
	return executor.Result{Capability: req.Capability, Target: req.Target, OK: true, Data: json.RawMessage(`{"adapter":"` + f.id + `"}`)}, nil
}
func (f *fakeAdapter) Cancel(context.Context, string) error { return nil }

func TestManagerTracksLifecycle(t *testing.T) {
	m := NewManager()
	adapter := &fakeAdapter{id: "native", health: Health{OK: true, Status: "ok"}}
	if err := m.Register(adapter); err != nil { t.Fatal(err) }
	status, err := m.Status("native")
	if err != nil { t.Fatal(err) }
	if status.State != LifecycleRegistered { t.Fatalf("state=%s", status.State) }

	if err := m.Start(context.Background(), "native"); err != nil { t.Fatal(err) }
	status, _ = m.Status("native")
	if status.State != LifecycleHealthy || !status.Health.OK { t.Fatalf("status=%#v", status) }

	if err := m.Stop(context.Background(), "native"); err != nil { t.Fatal(err) }
	status, _ = m.Status("native")
	if status.State != LifecycleStopped || status.Health.Status != "stopped" { t.Fatalf("status=%#v", status) }
}

func TestManagerMarksStartFailureDegraded(t *testing.T) {
	m := NewManager()
	adapter := &fakeAdapter{id: "mcp", startErr: errors.New("startup failed")}
	if err := m.Register(adapter); err != nil { t.Fatal(err) }
	if err := m.Start(context.Background(), "mcp"); err == nil { t.Fatal("expected start error") }
	status, _ := m.Status("mcp")
	if status.State != LifecycleDegraded || status.LastError != "startup failed" { t.Fatalf("status=%#v", status) }
}

func TestManagerSelectsHealthyAdapter(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "broken", health: Health{OK: false, Status: "down"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	if err := m.Register(&fakeAdapter{id: "native", health: Health{OK: true, Status: "ok"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	adapter, err := m.Select(context.Background(), "filesystem.read")
	if err != nil { t.Fatal(err) }
	if adapter.ID() != "native" { t.Fatalf("id=%s", adapter.ID()) }
	broken, _ := m.Status("broken")
	if broken.State != LifecycleDegraded { t.Fatalf("broken status=%#v", broken) }
	native, _ := m.Status("native")
	if native.State != LifecycleHealthy { t.Fatalf("native status=%#v", native) }
}

func TestManagerFailsWhenNoAdapterSupportsCapability(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "native", health: Health{OK: true, Status: "ok"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	_, err := m.Select(context.Background(), "service.restart")
	if !errors.Is(err, ErrNoAdapter) { t.Fatalf("err=%v", err) }
}

func TestManagerCapabilityFailureMarksDegraded(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "mcp", health: Health{OK: true, Status: "ok"}, capsErr: errors.New("capability probe failed")}); err != nil { t.Fatal(err) }
	_, err := m.Select(context.Background(), "network.read")
	if !errors.Is(err, ErrNoAdapter) { t.Fatalf("err=%v", err) }
	status, _ := m.Status("mcp")
	if status.State != LifecycleDegraded || status.LastError != "capability probe failed" { t.Fatalf("status=%#v", status) }
}

func TestManagerExecuteExplicitAdapter(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "mcp", health: Health{OK: true, Status: "ok"}, caps: []string{"network.read"}}); err != nil { t.Fatal(err) }
	out, err := m.Execute(context.Background(), "mcp", executor.Request{Agent: "main", Capability: "network.read", Target: "example"})
	if err != nil { t.Fatal(err) }
	if !out.OK || out.Capability != "network.read" { t.Fatalf("out=%#v", out) }
	status, _ := m.Status("mcp")
	if status.State != LifecycleHealthy { t.Fatalf("status=%#v", status) }
}

func TestManagerExecutionFailureMarksDegraded(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "browser", health: Health{OK: true, Status: "ok"}, caps: []string{"network.read"}, execErr: errors.New("execution failed")}); err != nil { t.Fatal(err) }
	_, err := m.Execute(context.Background(), "browser", executor.Request{Agent: "main", Capability: "network.read"})
	if err == nil { t.Fatal("expected execution error") }
	status, _ := m.Status("browser")
	if status.State != LifecycleDegraded || status.LastError != "execution failed" { t.Fatalf("status=%#v", status) }
}

func TestManagerStatusesAreSorted(t *testing.T) {
	m := NewManager()
	for _, id := range []string{"mcp", "browser", "native"} {
		if err := m.Register(&fakeAdapter{id: id}); err != nil { t.Fatal(err) }
	}
	statuses := m.Statuses()
	if len(statuses) != 3 || statuses[0].ID != "browser" || statuses[1].ID != "mcp" || statuses[2].ID != "native" {
		t.Fatalf("statuses=%#v", statuses)
	}
}

func TestManagerRejectsDuplicateAndInvalidIDs(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "Native", health: Health{OK: true}}); err == nil { t.Fatal("expected invalid id") }
	adapter := &fakeAdapter{id: "native", health: Health{OK: true}}
	if err := m.Register(adapter); err != nil { t.Fatal(err) }
	if err := m.Register(adapter); err == nil { t.Fatal("expected duplicate") }
}
