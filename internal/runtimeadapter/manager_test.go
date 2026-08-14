package runtimeadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

type fakeAdapter struct {
	id      string
	health  Health
	caps    []string
	execErr error
}

func (f *fakeAdapter) ID() string { return f.id }
func (f *fakeAdapter) Start(context.Context) error { return nil }
func (f *fakeAdapter) Stop(context.Context) error { return nil }
func (f *fakeAdapter) Health(context.Context) Health { return f.health }
func (f *fakeAdapter) Capabilities(context.Context) ([]string, error) { return append([]string(nil), f.caps...), nil }
func (f *fakeAdapter) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	if f.execErr != nil { return executor.Result{}, f.execErr }
	return executor.Result{Capability: req.Capability, Target: req.Target, OK: true, Data: json.RawMessage(`{"adapter":"` + f.id + `"}`)}, nil
}
func (f *fakeAdapter) Cancel(context.Context, string) error { return nil }

func TestManagerSelectsHealthyAdapter(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "broken", health: Health{OK: false, Status: "down"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	if err := m.Register(&fakeAdapter{id: "native", health: Health{OK: true, Status: "ok"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	adapter, err := m.Select(context.Background(), "filesystem.read")
	if err != nil { t.Fatal(err) }
	if adapter.ID() != "native" { t.Fatalf("id=%s", adapter.ID()) }
}

func TestManagerFailsWhenNoAdapterSupportsCapability(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "native", health: Health{OK: true, Status: "ok"}, caps: []string{"filesystem.read"}}); err != nil { t.Fatal(err) }
	_, err := m.Select(context.Background(), "service.restart")
	if !errors.Is(err, ErrNoAdapter) { t.Fatalf("err=%v", err) }
}

func TestManagerExecuteExplicitAdapter(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "mcp", health: Health{OK: true, Status: "ok"}, caps: []string{"network.read"}}); err != nil { t.Fatal(err) }
	out, err := m.Execute(context.Background(), "mcp", executor.Request{Agent: "main", Capability: "network.read", Target: "example"})
	if err != nil { t.Fatal(err) }
	if !out.OK || out.Capability != "network.read" { t.Fatalf("out=%#v", out) }
}

func TestManagerRejectsDuplicateAndInvalidIDs(t *testing.T) {
	m := NewManager()
	if err := m.Register(&fakeAdapter{id: "Native", health: Health{OK: true}}); err == nil { t.Fatal("expected invalid id") }
	adapter := &fakeAdapter{id: "native", health: Health{OK: true}}
	if err := m.Register(adapter); err != nil { t.Fatal(err) }
	if err := m.Register(adapter); err == nil { t.Fatal("expected duplicate") }
}
