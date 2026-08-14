package taskgraph

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Status string

const (
	StatusCreated         Status = "created"
	StatusPlanning        Status = "planning"
	StatusWaiting         Status = "waiting"
	StatusWaitingApproval Status = "waiting_approval"
	StatusRunning         Status = "running"
	StatusPaused          Status = "paused"
	StatusBlocked         Status = "blocked"
	StatusFailed          Status = "failed"
	StatusCompleted       Status = "completed"
	StatusCancelled       Status = "cancelled"
)

type Step struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	Capability string          `json:"capability,omitempty"`
	Target     string          `json:"target,omitempty"`
	DependsOn  []string        `json:"depends_on,omitempty"`
	ApprovalID string          `json:"approval_id,omitempty"`
	Status     Status          `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	StartedAt  *time.Time      `json:"started_at,omitempty"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

type Task struct {
	ID        string          `json:"id"`
	Goal      string          `json:"goal"`
	Agent     string          `json:"agent"`
	PeerUID   uint32          `json:"peer_uid"`
	Status    Status          `json:"status"`
	Steps     []Step          `json:"steps,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Store struct{ Root string }

func (s Store) Create(goal, agent string, peerUID uint32, steps []Step) (Task, error) {
	goal = strings.TrimSpace(goal)
	agent = strings.TrimSpace(agent)
	if goal == "" || agent == "" {
		return Task{}, errors.New("goal and agent are required")
	}
	id, err := randomID()
	if err != nil {
		return Task{}, err
	}
	seen := map[string]struct{}{}
	for i := range steps {
		steps[i].ID = strings.TrimSpace(steps[i].ID)
		steps[i].Title = strings.TrimSpace(steps[i].Title)
		if steps[i].ID == "" {
			steps[i].ID = fmt.Sprintf("step-%d", i+1)
		}
		if steps[i].Title == "" {
			steps[i].Title = steps[i].ID
		}
		if _, ok := seen[steps[i].ID]; ok {
			return Task{}, fmt.Errorf("duplicate step id: %s", steps[i].ID)
		}
		seen[steps[i].ID] = struct{}{}
		if steps[i].Status == "" {
			steps[i].Status = StatusCreated
		}
		if !validStatus(steps[i].Status) {
			return Task{}, fmt.Errorf("invalid initial step status: %s", steps[i].Status)
		}
	}
	for _, step := range steps {
		for _, dep := range step.DependsOn {
			if dep == step.ID {
				return Task{}, fmt.Errorf("step %s cannot depend on itself", step.ID)
			}
			if _, ok := seen[dep]; !ok {
				return Task{}, fmt.Errorf("step %s depends on unknown step %s", step.ID, dep)
			}
		}
	}
	if hasDependencyCycle(steps) {
		return Task{}, errors.New("task step dependency cycle detected")
	}
	now := time.Now().UTC()
	t := Task{ID: id, Goal: goal, Agent: agent, PeerUID: peerUID, Status: StatusCreated, Steps: steps, CreatedAt: now, UpdatedAt: now}
	if err := s.write(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s Store) Get(id string) (Task, error) {
	if !safeID(id) {
		return Task{}, errors.New("invalid task id")
	}
	b, err := os.ReadFile(filepath.Join(s.Root, id+".json"))
	if err != nil {
		return Task{}, err
	}
	var t Task
	if err := json.Unmarshal(b, &t); err != nil {
		return Task{}, fmt.Errorf("decode task: %w", err)
	}
	return t, nil
}

func (s Store) ListForPeer(peerUID uint32, limit int) ([]Task, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	entries, err := os.ReadDir(s.Root)
	if errors.Is(err, os.ErrNotExist) {
		return []Task{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		t, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		if peerUID != 0 && t.PeerUID != peerUID {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s Store) TransitionForPeer(id string, peerUID uint32, next Status, result json.RawMessage, failure string) (Task, error) {
	t, err := s.Get(id)
	if err != nil {
		return Task{}, err
	}
	if err := authorizePeer(t, peerUID); err != nil {
		return Task{}, err
	}
	if !validStatus(next) || !canTransition(t.Status, next) {
		return Task{}, fmt.Errorf("invalid task transition: %s -> %s", t.Status, next)
	}
	t.Status = next
	t.UpdatedAt = time.Now().UTC()
	if next == StatusCompleted {
		t.Result = result
		t.Error = ""
	}
	if next == StatusFailed || next == StatusBlocked {
		t.Error = strings.TrimSpace(failure)
	}
	if err := s.write(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s Store) SetStepApprovalForPeer(id string, peerUID uint32, stepID, approvalID string) (Task, error) {
	t, err := s.Get(id)
	if err != nil {
		return Task{}, err
	}
	if err := authorizePeer(t, peerUID); err != nil {
		return Task{}, err
	}
	found := false
	for i := range t.Steps {
		if t.Steps[i].ID == stepID {
			if terminal(t.Steps[i].Status) {
				return Task{}, errors.New("cannot attach approval to terminal step")
			}
			t.Steps[i].ApprovalID = strings.TrimSpace(approvalID)
			t.Steps[i].Status = StatusWaitingApproval
			found = true
			break
		}
	}
	if !found {
		return Task{}, errors.New("task step not found")
	}
	t.Status = StatusWaitingApproval
	t.UpdatedAt = time.Now().UTC()
	if err := s.write(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s Store) TransitionStepForPeer(id string, peerUID uint32, stepID string, next Status, result json.RawMessage, failure string) (Task, error) {
	t, err := s.Get(id)
	if err != nil {
		return Task{}, err
	}
	if err := authorizePeer(t, peerUID); err != nil {
		return Task{}, err
	}
	if terminal(t.Status) {
		return Task{}, errors.New("cannot transition a step in a terminal task")
	}
	idx := -1
	for i := range t.Steps {
		if t.Steps[i].ID == stepID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Task{}, errors.New("task step not found")
	}
	step := &t.Steps[idx]
	if !validStatus(next) || !canTransition(step.Status, next) {
		return Task{}, fmt.Errorf("invalid step transition: %s -> %s", step.Status, next)
	}
	if next == StatusRunning && !dependenciesCompleted(t, *step) {
		return Task{}, errors.New("step dependencies are not completed")
	}

	now := time.Now().UTC()
	step.Status = next
	if next == StatusRunning && step.StartedAt == nil {
		step.StartedAt = &now
	}
	if next == StatusCompleted {
		step.Result = result
		step.Error = ""
		step.FinishedAt = &now
	}
	if next == StatusFailed || next == StatusBlocked {
		step.Error = strings.TrimSpace(failure)
		if next == StatusFailed {
			step.FinishedAt = &now
		}
	}
	if next == StatusCancelled {
		step.FinishedAt = &now
	}

	t.UpdatedAt = now
	reconcileTaskStatus(&t)
	if err := s.write(t); err != nil {
		return Task{}, err
	}
	return t, nil
}

func ReadySteps(t Task) []Step {
	if terminal(t.Status) {
		return []Step{}
	}
	out := make([]Step, 0)
	for _, step := range t.Steps {
		if (step.Status == StatusCreated || step.Status == StatusWaiting) && dependenciesCompleted(t, step) {
			out = append(out, step)
		}
	}
	return out
}

func authorizePeer(t Task, peerUID uint32) error {
	if peerUID != 0 && t.PeerUID != peerUID {
		return errors.New("task owner mismatch")
	}
	return nil
}

func dependenciesCompleted(t Task, step Step) bool {
	if len(step.DependsOn) == 0 {
		return true
	}
	status := make(map[string]Status, len(t.Steps))
	for _, candidate := range t.Steps {
		status[candidate.ID] = candidate.Status
	}
	for _, dep := range step.DependsOn {
		if status[dep] != StatusCompleted {
			return false
		}
	}
	return true
}

func reconcileTaskStatus(t *Task) {
	if len(t.Steps) == 0 {
		return
	}
	allComplete := true
	for _, step := range t.Steps {
		switch step.Status {
		case StatusFailed:
			t.Status = StatusFailed
			t.Error = step.Error
			return
		case StatusBlocked:
			t.Status = StatusBlocked
			t.Error = step.Error
			return
		case StatusWaitingApproval:
			allComplete = false
			t.Status = StatusWaitingApproval
		case StatusRunning:
			allComplete = false
			t.Status = StatusRunning
		case StatusCreated, StatusPlanning, StatusWaiting, StatusPaused:
			allComplete = false
			if t.Status != StatusWaitingApproval && t.Status != StatusRunning {
				t.Status = StatusWaiting
			}
		case StatusCancelled:
			allComplete = false
		}
	}
	if allComplete {
		t.Status = StatusCompleted
		t.Error = ""
	}
}

func hasDependencyCycle(steps []Step) bool {
	deps := make(map[string][]string, len(steps))
	for _, step := range steps {
		deps[step.ID] = append([]string(nil), step.DependsOn...)
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if visit(dep) {
				return true
			}
		}
		visiting[id] = false
		visited[id] = true
		return false
	}
	for id := range deps {
		if visit(id) {
			return true
		}
	}
	return false
}

func terminal(status Status) bool {
	return status == StatusCompleted || status == StatusFailed || status == StatusCancelled
}

func validStatus(status Status) bool {
	switch status {
	case StatusCreated, StatusPlanning, StatusWaiting, StatusWaitingApproval, StatusRunning, StatusPaused, StatusBlocked, StatusFailed, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

func canTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed := map[Status]map[Status]bool{
		StatusCreated:         {StatusPlanning: true, StatusWaiting: true, StatusWaitingApproval: true, StatusRunning: true, StatusCancelled: true},
		StatusPlanning:        {StatusWaiting: true, StatusWaitingApproval: true, StatusRunning: true, StatusBlocked: true, StatusFailed: true, StatusCancelled: true},
		StatusWaiting:         {StatusPlanning: true, StatusWaitingApproval: true, StatusRunning: true, StatusBlocked: true, StatusCancelled: true},
		StatusWaitingApproval: {StatusRunning: true, StatusBlocked: true, StatusCancelled: true},
		StatusRunning:         {StatusPaused: true, StatusWaiting: true, StatusWaitingApproval: true, StatusBlocked: true, StatusFailed: true, StatusCompleted: true, StatusCancelled: true},
		StatusPaused:          {StatusRunning: true, StatusCancelled: true},
		StatusBlocked:         {StatusPlanning: true, StatusWaiting: true, StatusRunning: true, StatusFailed: true, StatusCancelled: true},
	}
	return allowed[from][to]
}

func (s Store) write(t Task) error {
	if s.Root == "" {
		return errors.New("task root is required")
	}
	if !safeID(t.ID) {
		return errors.New("invalid task id")
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(s.Root, t.ID+".json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func safeID(v string) bool {
	if len(v) != 32 {
		return false
	}
	for _, r := range v {
		if !((r >= 'a' && r <= 'f') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
