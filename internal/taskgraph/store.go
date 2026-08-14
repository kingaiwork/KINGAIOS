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
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Capability string   `json:"capability,omitempty"`
	DependsOn  []string `json:"depends_on,omitempty"`
	ApprovalID string   `json:"approval_id,omitempty"`
	Status     Status   `json:"status"`
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
		if steps[i].ID == "" {
			steps[i].ID = fmt.Sprintf("step-%d", i+1)
		}
		if _, ok := seen[steps[i].ID]; ok {
			return Task{}, fmt.Errorf("duplicate step id: %s", steps[i].ID)
		}
		seen[steps[i].ID] = struct{}{}
		if steps[i].Status == "" {
			steps[i].Status = StatusCreated
		}
	}
	for _, step := range steps {
		for _, dep := range step.DependsOn {
			if _, ok := seen[dep]; !ok {
				return Task{}, fmt.Errorf("step %s depends on unknown step %s", step.ID, dep)
			}
		}
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
	if peerUID != 0 && t.PeerUID != peerUID {
		return Task{}, errors.New("task owner mismatch")
	}
	if !canTransition(t.Status, next) {
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
	if peerUID != 0 && t.PeerUID != peerUID {
		return Task{}, errors.New("task owner mismatch")
	}
	found := false
	for i := range t.Steps {
		if t.Steps[i].ID == stepID {
			t.Steps[i].ApprovalID = approvalID
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

func canTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed := map[Status]map[Status]bool{
		StatusCreated:         {StatusPlanning: true, StatusCancelled: true},
		StatusPlanning:        {StatusWaiting: true, StatusWaitingApproval: true, StatusRunning: true, StatusBlocked: true, StatusFailed: true, StatusCancelled: true},
		StatusWaiting:         {StatusPlanning: true, StatusRunning: true, StatusBlocked: true, StatusCancelled: true},
		StatusWaitingApproval: {StatusRunning: true, StatusBlocked: true, StatusCancelled: true},
		StatusRunning:         {StatusPaused: true, StatusWaiting: true, StatusWaitingApproval: true, StatusBlocked: true, StatusFailed: true, StatusCompleted: true, StatusCancelled: true},
		StatusPaused:          {StatusRunning: true, StatusCancelled: true},
		StatusBlocked:         {StatusPlanning: true, StatusRunning: true, StatusFailed: true, StatusCancelled: true},
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
