package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

type fakeDispatcher struct {
	calls int
	err   error
}

func (d *fakeDispatcher) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	d.calls++
	if d.err != nil { return executor.Result{Capability: req.Capability, Target: req.Target, Message: "failed"}, d.err }
	return executor.Result{Capability: req.Capability, Target: req.Target, OK: true}, nil
}

func TestSchedulerRunsReadyStepsInDependencyOrder(t *testing.T) {
	store := taskgraph.Store{Root: t.TempDir()}
	task, err := store.Create("restart", "system-ops", 1000, []taskgraph.Step{
		{ID: "one", Capability: "service.restart", Target: "one.service"},
		{ID: "two", Capability: "service.restart", Target: "two.service", DependsOn: []string{"one"}},
	})
	if err != nil { t.Fatal(err) }
	dispatcher := &fakeDispatcher{}
	s := Scheduler{
		Store: store,
		Authorizer: AuthorizerFunc(func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error) {
			return Authorization{Allowed: true}, nil
		}),
		Dispatcher: dispatcher,
	}
	out, err := s.RunReady(context.Background(), task.ID, 1000)
	if err != nil { t.Fatal(err) }
	if out.Status != taskgraph.StatusCompleted || dispatcher.calls != 2 {
		t.Fatalf("out=%#v calls=%d", out, dispatcher.calls)
	}
}

func TestSchedulerStopsForApproval(t *testing.T) {
	store := taskgraph.Store{Root: t.TempDir()}
	task, err := store.Create("restart", "system-ops", 1000, []taskgraph.Step{{ID: "one", Capability: "service.restart", Target: "nginx.service"}})
	if err != nil { t.Fatal(err) }
	dispatcher := &fakeDispatcher{}
	s := Scheduler{
		Store: store,
		Authorizer: AuthorizerFunc(func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error) {
			return Authorization{ApprovalRequired: true, ApprovalID: "approval-1", Reason: "approval required"}, nil
		}),
		Dispatcher: dispatcher,
	}
	out, err := s.RunReady(context.Background(), task.ID, 1000)
	if err != nil { t.Fatal(err) }
	if out.Status != taskgraph.StatusWaitingApproval || out.Steps[0].ApprovalID != "approval-1" || dispatcher.calls != 0 {
		t.Fatalf("out=%#v calls=%d", out, dispatcher.calls)
	}
}

func TestSchedulerBlocksDeniedStep(t *testing.T) {
	store := taskgraph.Store{Root: t.TempDir()}
	task, err := store.Create("restart", "system-ops", 1000, []taskgraph.Step{{ID: "one", Capability: "service.restart"}})
	if err != nil { t.Fatal(err) }
	s := Scheduler{
		Store: store,
		Authorizer: AuthorizerFunc(func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error) {
			return Authorization{Reason: "denied"}, nil
		}),
		Dispatcher: &fakeDispatcher{},
	}
	out, err := s.RunReady(context.Background(), task.ID, 1000)
	if err != nil { t.Fatal(err) }
	if out.Status != taskgraph.StatusBlocked || out.Steps[0].Error != "denied" {
		t.Fatalf("out=%#v", out)
	}
}

func TestSchedulerFailsTaskWhenExecutionFails(t *testing.T) {
	store := taskgraph.Store{Root: t.TempDir()}
	task, err := store.Create("restart", "system-ops", 1000, []taskgraph.Step{{ID: "one", Capability: "service.restart"}})
	if err != nil { t.Fatal(err) }
	s := Scheduler{
		Store: store,
		Authorizer: AuthorizerFunc(func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error) { return Authorization{Allowed: true}, nil }),
		Dispatcher: &fakeDispatcher{err: errors.New("boom")},
	}
	out, err := s.RunReady(context.Background(), task.ID, 1000)
	if err == nil { t.Fatal("expected execution error") }
	if out.Status != taskgraph.StatusFailed { t.Fatalf("out=%#v", out) }
}

func TestSchedulerLeavesNonExecutablePlanningStepReady(t *testing.T) {
	store := taskgraph.Store{Root: t.TempDir()}
	task, err := store.Create("plan", "main", 1000, []taskgraph.Step{{ID: "think", Title: "think"}})
	if err != nil { t.Fatal(err) }
	dispatcher := &fakeDispatcher{}
	s := Scheduler{
		Store: store,
		Authorizer: AuthorizerFunc(func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error) { return Authorization{Allowed: true}, nil }),
		Dispatcher: dispatcher,
	}
	out, err := s.RunReady(context.Background(), task.ID, 1000)
	if err != nil { t.Fatal(err) }
	if out.Status != taskgraph.StatusCreated || dispatcher.calls != 0 { t.Fatalf("out=%#v calls=%d", out, dispatcher.calls) }
	_ = json.Valid(nil)
}
