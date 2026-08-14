package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBrokerRejectsUnknownCapability(t *testing.T) {
	b := New(time.Second)
	_, err := b.Execute(context.Background(), Request{Agent: "main", Capability: "service.restart", Target: "nginx.service"})
	if !errors.Is(err, ErrUnknownCapability) {
		t.Fatalf("err=%v", err)
	}
}

func TestBrokerExecutesRegisteredHandler(t *testing.T) {
	b := New(time.Second)
	if err := b.Register("filesystem.read", HandlerFunc(func(ctx context.Context, req Request) (Result, error) {
		return Result{Data: json.RawMessage(`{"read":true}`)}, nil
	})); err != nil {
		t.Fatal(err)
	}
	out, err := b.Execute(context.Background(), Request{Agent: "main", Capability: "filesystem.read", Target: "/safe/file"})
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Agent != "main" || out.Capability != "filesystem.read" || out.Target != "/safe/file" {
		t.Fatalf("unexpected result: %#v", out)
	}
	if len(out.ExecutionID) != 32 {
		t.Fatalf("execution id=%q", out.ExecutionID)
	}
	if out.StartedAt.IsZero() || out.FinishedAt.IsZero() || out.FinishedAt.Before(out.StartedAt) || out.DurationMS < 0 {
		t.Fatalf("invalid timing: %#v", out)
	}
}

func TestBrokerGeneratesUniqueExecutionReceipts(t *testing.T) {
	b := New(time.Second)
	if err := b.Register("audit.read", HandlerFunc(func(context.Context, Request) (Result, error) { return Result{}, nil })); err != nil {
		t.Fatal(err)
	}
	first, err := b.Execute(context.Background(), Request{Agent: "main", Capability: "audit.read"})
	if err != nil { t.Fatal(err) }
	second, err := b.Execute(context.Background(), Request{Agent: "main", Capability: "audit.read"})
	if err != nil { t.Fatal(err) }
	if first.ExecutionID == second.ExecutionID {
		t.Fatal("execution ids must be unique")
	}
}

func TestBrokerRejectsOversizedInputs(t *testing.T) {
	b := New(time.Second)
	if err := b.Register("filesystem.read", HandlerFunc(func(context.Context, Request) (Result, error) { return Result{}, nil })); err != nil {
		t.Fatal(err)
	}
	cases := []Request{
		{Agent: strings.Repeat("a", MaxAgentBytes+1), Capability: "filesystem.read"},
		{Agent: "main", Capability: "filesystem.read", Target: strings.Repeat("x", MaxTargetBytes+1)},
		{Agent: "main", Capability: "filesystem.read", Arguments: json.RawMessage(`"` + strings.Repeat("x", MaxArgumentsBytes) + `"`)},
	}
	for i, req := range cases {
		if _, err := b.Execute(context.Background(), req); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("case %d err=%v", i, err)
		}
	}
}

func TestBrokerTimeout(t *testing.T) {
	b := New(10 * time.Millisecond)
	if err := b.Register("process.execute", HandlerFunc(func(ctx context.Context, req Request) (Result, error) {
		<-ctx.Done()
		return Result{}, ctx.Err()
	})); err != nil {
		t.Fatal(err)
	}
	out, err := b.Execute(context.Background(), Request{Agent: "system-ops", Capability: "process.execute"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v", err)
	}
	if out.ExecutionID == "" || out.OK {
		t.Fatalf("timeout receipt missing: %#v", out)
	}
}

func TestBrokerRejectsDuplicateRegistration(t *testing.T) {
	b := New(time.Second)
	h := HandlerFunc(func(context.Context, Request) (Result, error) { return Result{}, nil })
	if err := b.Register("filesystem.read", h); err != nil {
		t.Fatal(err)
	}
	if err := b.Register("filesystem.read", h); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

func TestValidators(t *testing.T) {
	if err := ValidateServiceUnit("nginx.service"); err != nil { t.Fatal(err) }
	if err := ValidateServiceUnit("../../bad"); err == nil { t.Fatal("expected invalid service unit") }
	if err := ValidatePackageName("curl"); err != nil { t.Fatal(err) }
	if err := ValidatePackageName("--option"); err == nil { t.Fatal("expected invalid package name") }
	root := t.TempDir()
	inside := root + "/a/b"
	if _, err := ValidatePathWithin(root, inside); err != nil { t.Fatal(err) }
	if _, err := ValidatePathWithin(root, root+"/../escape"); err == nil { t.Fatal("expected path escape rejection") }
}
