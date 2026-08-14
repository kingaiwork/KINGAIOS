package taskgraph

import (
	"encoding/json"
	"testing"
)

func TestTaskLifecycle(t *testing.T) {
	s := Store{Root: t.TempDir()}
	task, err := s.Create("restart web service", "system-ops", []Step{{ID: "inspect", Title: "inspect"}, {ID: "restart", Title: "restart", Capability: "service.restart", DependsOn: []string{"inspect"}}})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusCreated {
		t.Fatalf("status=%s", task.Status)
	}
	for _, next := range []Status{StatusPlanning, StatusWaitingApproval, StatusRunning} {
		task, err = s.Transition(task.ID, next, nil, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	result := json.RawMessage(`{"ok":true}`)
	task, err = s.Transition(task.ID, StatusCompleted, result, "")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusCompleted || string(task.Result) != string(result) {
		t.Fatalf("unexpected completed task: %#v", task)
	}
}

func TestTaskRejectsInvalidTransition(t *testing.T) {
	s := Store{Root: t.TempDir()}
	task, err := s.Create("goal", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Transition(task.ID, StatusCompleted, nil, ""); err == nil {
		t.Fatal("expected invalid transition to fail")
	}
}

func TestTaskValidatesDependencies(t *testing.T) {
	s := Store{Root: t.TempDir()}
	_, err := s.Create("goal", "main", []Step{{ID: "two", Title: "two", DependsOn: []string{"missing"}}})
	if err == nil {
		t.Fatal("expected unknown dependency to fail")
	}
}
