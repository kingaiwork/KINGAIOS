package planner

import (
	"testing"

	"github.com/kingaiwork/KINGAIOS/internal/agent"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

func TestValidatorAcceptsDeclaredCapabilityPlan(t *testing.T) {
	registry := agent.New([]agent.Definition{{ID: "main", Capabilities: []string{"filesystem.read"}}})
	v := Validator{Registry: registry, MaxSteps: 8}
	plan, err := v.Validate(Plan{Goal: "inspect workspace", Agent: "main", Steps: []taskgraph.Step{{ID: "inspect", Capability: "filesystem.read", Target: "/workspace"}}})
	if err != nil { t.Fatal(err) }
	if plan.Steps[0].Status != taskgraph.StatusCreated { t.Fatalf("status=%s", plan.Steps[0].Status) }
}

func TestValidatorRejectsUndeclaredCapability(t *testing.T) {
	registry := agent.New([]agent.Definition{{ID: "main", Capabilities: []string{"filesystem.read"}}})
	v := Validator{Registry: registry}
	_, err := v.Validate(Plan{Goal: "restart", Agent: "main", Steps: []taskgraph.Step{{ID: "restart", Capability: "service.restart"}}})
	if err == nil { t.Fatal("expected undeclared capability rejection") }
}

func TestValidatorRejectsCycleAndRuntimeState(t *testing.T) {
	v := Validator{}
	_, err := v.Validate(Plan{Goal: "cycle", Agent: "main", Steps: []taskgraph.Step{{ID: "one", DependsOn: []string{"two"}}, {ID: "two", DependsOn: []string{"one"}}}})
	if err == nil { t.Fatal("expected cycle rejection") }
	_, err = v.Validate(Plan{Goal: "bad state", Agent: "main", Steps: []taskgraph.Step{{ID: "one", Status: taskgraph.StatusRunning}}})
	if err == nil { t.Fatal("expected runtime state rejection") }
}

func TestValidatorClearsPlannerSuppliedRuntimeFields(t *testing.T) {
	v := Validator{}
	plan, err := v.Validate(Plan{Goal: "safe", Agent: "main", Steps: []taskgraph.Step{{ID: "one", ApprovalID: "forged", Error: "forged"}}})
	if err != nil { t.Fatal(err) }
	if plan.Steps[0].ApprovalID != "" || plan.Steps[0].Error != "" { t.Fatalf("runtime fields were not cleared: %#v", plan.Steps[0]) }
}
