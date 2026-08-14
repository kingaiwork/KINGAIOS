package planner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

var ErrInvalidPlan = errors.New("invalid plan")

type Request struct {
	Goal        string `json:"goal"`
	Agent       string `json:"agent"`
	MaxSteps    int    `json:"max_steps,omitempty"`
	Offline     bool   `json:"offline,omitempty"`
	Private     bool   `json:"private,omitempty"`
}

type Plan struct {
	Goal    string           `json:"goal"`
	Agent   string           `json:"agent"`
	Summary string           `json:"summary,omitempty"`
	Steps   []taskgraph.Step `json:"steps"`
}

type Planner interface {
	Plan(context.Context, Request) (Plan, error)
}

type CapabilityChecker interface {
	Allows(string, string) bool
}

type Validator struct {
	Registry CapabilityChecker
	MaxSteps int
}

func (v Validator) Validate(plan Plan) (Plan, error) {
	plan.Goal = strings.TrimSpace(plan.Goal)
	plan.Agent = strings.TrimSpace(plan.Agent)
	plan.Summary = strings.TrimSpace(plan.Summary)
	if plan.Goal == "" || plan.Agent == "" { return Plan{}, fmt.Errorf("%w: goal and agent are required", ErrInvalidPlan) }
	if len(plan.Goal) > 4096 || len(plan.Summary) > 8192 { return Plan{}, fmt.Errorf("%w: plan text exceeds limits", ErrInvalidPlan) }
	max := v.MaxSteps
	if max <= 0 { max = 64 }
	if max > 256 { max = 256 }
	if len(plan.Steps) == 0 || len(plan.Steps) > max { return Plan{}, fmt.Errorf("%w: plan must contain 1-%d steps", ErrInvalidPlan, max) }

	ids := make(map[string]struct{}, len(plan.Steps))
	for i := range plan.Steps {
		step := &plan.Steps[i]
		step.ID = strings.TrimSpace(step.ID)
		step.Title = strings.TrimSpace(step.Title)
		step.Capability = strings.TrimSpace(step.Capability)
		step.Target = strings.TrimSpace(step.Target)
		if step.ID == "" { step.ID = fmt.Sprintf("step-%d", i+1) }
		if step.Title == "" { step.Title = step.ID }
		if len(step.ID) > 128 || len(step.Title) > 512 || len(step.Target) > 4096 { return Plan{}, fmt.Errorf("%w: step exceeds limits", ErrInvalidPlan) }
		if _, exists := ids[step.ID]; exists { return Plan{}, fmt.Errorf("%w: duplicate step id %s", ErrInvalidPlan, step.ID) }
		ids[step.ID] = struct{}{}
		if step.Status == "" { step.Status = taskgraph.StatusCreated }
		if step.Status != taskgraph.StatusCreated { return Plan{}, fmt.Errorf("%w: planner may only emit created steps", ErrInvalidPlan) }
		if step.Capability != "" && v.Registry != nil && !v.Registry.Allows(plan.Agent, step.Capability) { return Plan{}, fmt.Errorf("%w: agent %s does not declare capability %s", ErrInvalidPlan, plan.Agent, step.Capability) }
		step.ApprovalID = ""
		step.Result = nil
		step.Error = ""
		step.StartedAt = nil
		step.FinishedAt = nil
	}

	for _, step := range plan.Steps {
		for _, dep := range step.DependsOn {
			if dep == step.ID { return Plan{}, fmt.Errorf("%w: step %s depends on itself", ErrInvalidPlan, step.ID) }
			if _, ok := ids[dep]; !ok { return Plan{}, fmt.Errorf("%w: step %s depends on unknown step %s", ErrInvalidPlan, step.ID, dep) }
		}
	}
	if hasCycle(plan.Steps) { return Plan{}, fmt.Errorf("%w: dependency cycle", ErrInvalidPlan) }
	return plan, nil
}

func hasCycle(steps []taskgraph.Step) bool {
	deps := make(map[string][]string, len(steps))
	for _, step := range steps { deps[step.ID] = append([]string(nil), step.DependsOn...) }
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] { return true }
		if visited[id] { return false }
		visiting[id] = true
		for _, dep := range deps[id] { if visit(dep) { return true } }
		visiting[id] = false
		visited[id] = true
		return false
	}
	for id := range deps { if visit(id) { return true } }
	return false
}
