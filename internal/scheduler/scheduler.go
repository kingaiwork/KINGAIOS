package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

type Authorization struct {
	Allowed          bool
	ApprovalRequired bool
	ApprovalID       string
	Reason           string
}

type Authorizer interface {
	Authorize(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error)
}

type AuthorizerFunc func(context.Context, taskgraph.Task, taskgraph.Step, uint32) (Authorization, error)

func (f AuthorizerFunc) Authorize(ctx context.Context, task taskgraph.Task, step taskgraph.Step, peerUID uint32) (Authorization, error) {
	return f(ctx, task, step, peerUID)
}

type Dispatcher interface {
	Execute(context.Context, executor.Request) (executor.Result, error)
}

type TaskStore interface {
	Get(string) (taskgraph.Task, error)
	TransitionStepForPeer(string, uint32, string, taskgraph.Status, json.RawMessage, string) (taskgraph.Task, error)
	SetStepApprovalForPeer(string, uint32, string, string) (taskgraph.Task, error)
}

type Scheduler struct {
	Store      TaskStore
	Authorizer Authorizer
	Dispatcher Dispatcher
}

func (s Scheduler) RunReady(ctx context.Context, taskID string, peerUID uint32) (taskgraph.Task, error) {
	if s.Store == nil || s.Authorizer == nil || s.Dispatcher == nil {
		return taskgraph.Task{}, errors.New("scheduler dependencies are required")
	}
	task, err := s.Store.Get(taskID)
	if err != nil { return taskgraph.Task{}, err }
	if peerUID != 0 && task.PeerUID != peerUID { return taskgraph.Task{}, errors.New("task owner mismatch") }

	for {
		ready := taskgraph.ReadySteps(task)
		if len(ready) == 0 { return task, nil }
		progressed := false
		for _, step := range ready {
			if step.Capability == "" {
				continue
			}
			auth, err := s.Authorizer.Authorize(ctx, task, step, peerUID)
			if err != nil { return task, err }
			if !auth.Allowed {
				if auth.ApprovalRequired {
					if auth.ApprovalID != "" {
						task, err = s.Store.SetStepApprovalForPeer(task.ID, peerUID, step.ID, auth.ApprovalID)
					} else {
						task, err = s.Store.TransitionStepForPeer(task.ID, peerUID, step.ID, taskgraph.StatusWaitingApproval, nil, auth.Reason)
					}
					if err != nil { return task, err }
					return task, nil
				}
				reason := auth.Reason
				if reason == "" { reason = "execution denied by policy" }
				task, err = s.Store.TransitionStepForPeer(task.ID, peerUID, step.ID, taskgraph.StatusBlocked, nil, reason)
				if err != nil { return task, err }
				return task, nil
			}

			task, err = s.Store.TransitionStepForPeer(task.ID, peerUID, step.ID, taskgraph.StatusRunning, nil, "")
			if err != nil { return task, err }
			result, execErr := s.Dispatcher.Execute(ctx, executor.Request{Agent: task.Agent, Capability: step.Capability, Target: step.Target})
			if execErr != nil {
				reason := result.Message
				if reason == "" { reason = execErr.Error() }
				task, err = s.Store.TransitionStepForPeer(task.ID, peerUID, step.ID, taskgraph.StatusFailed, nil, reason)
				if err != nil { return task, err }
				return task, fmt.Errorf("step %s execution failed: %w", step.ID, execErr)
			}
			payload, err := json.Marshal(result)
			if err != nil { return task, err }
			task, err = s.Store.TransitionStepForPeer(task.ID, peerUID, step.ID, taskgraph.StatusCompleted, payload, "")
			if err != nil { return task, err }
			progressed = true
		}
		if !progressed { return task, nil }
	}
}
