package taskgraph

import (
	"encoding/json"
	"testing"
)

func TestTaskLifecycle(t *testing.T) {
	s := Store{Root: t.TempDir()}
	task, err := s.Create("restart web service", "system-ops", 1000, []Step{{ID: "inspect", Title: "inspect"}, {ID: "restart", Title: "restart", Capability: "service.restart", DependsOn: []string{"inspect"}}})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusCreated || task.PeerUID != 1000 {
		t.Fatalf("unexpected task: %#v", task)
	}
	for _, next := range []Status{StatusPlanning, StatusWaitingApproval, StatusRunning} {
		task, err = s.TransitionForPeer(task.ID, 1000, next, nil, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	result := json.RawMessage(`{"ok":true}`)
	task, err = s.TransitionForPeer(task.ID, 1000, StatusCompleted, result, "")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusCompleted || string(task.Result) != string(result) {
		t.Fatalf("unexpected completed task: %#v", task)
	}
}

func TestTaskRejectsInvalidTransition(t *testing.T) {
	s := Store{Root: t.TempDir()}
	task, err := s.Create("goal", "main", 1000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransitionForPeer(task.ID, 1000, StatusCompleted, nil, ""); err == nil {
		t.Fatal("expected invalid transition to fail")
	}
}

func TestTaskValidatesDependencies(t *testing.T) {
	s := Store{Root: t.TempDir()}
	_, err := s.Create("goal", "main", 1000, []Step{{ID: "two", Title: "two", DependsOn: []string{"missing"}}})
	if err == nil {
		t.Fatal("expected unknown dependency to fail")
	}
}

func TestTaskPeerIsolation(t *testing.T) {
	s := Store{Root: t.TempDir()}
	task, err := s.Create("goal", "main", 1000, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransitionForPeer(task.ID, 1001, StatusPlanning, nil, ""); err == nil {
		t.Fatal("expected peer mismatch")
	}
	mine, err := s.ListForPeer(1000, 10)
	if err != nil || len(mine) != 1 {
		t.Fatalf("mine=%v err=%v", mine, err)
	}
	other, err := s.ListForPeer(1001, 10)
	if err != nil || len(other) != 0 {
		t.Fatalf("other=%v err=%v", other, err)
	}
}
