package approval

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
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusDenied   Status = "denied"
	StatusConsumed Status = "consumed"
	StatusExpired  Status = "expired"
)

type Request struct {
	ID         string     `json:"id"`
	Agent      string     `json:"agent"`
	Capability string     `json:"capability"`
	TargetHash string     `json:"target_hash,omitempty"`
	PeerUID    uint32     `json:"peer_uid"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Status     Status     `json:"status"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	DecidedBy  *uint32    `json:"decided_by,omitempty"`
}

type Store struct{ Root string }

func (s Store) Create(agent, capability, targetHash string, peerUID uint32, ttl time.Duration) (Request, error) {
	if strings.TrimSpace(agent) == "" || strings.TrimSpace(capability) == "" {
		return Request{}, errors.New("agent and capability are required")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if ttl > 30*time.Minute {
		ttl = 30 * time.Minute
	}
	id, err := randomID()
	if err != nil {
		return Request{}, err
	}
	now := time.Now().UTC()
	r := Request{ID: id, Agent: agent, Capability: capability, TargetHash: targetHash, PeerUID: peerUID, CreatedAt: now, ExpiresAt: now.Add(ttl), Status: StatusPending}
	if err := s.write(r); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (s Store) Get(id string) (Request, error) {
	if !safeID(id) {
		return Request{}, errors.New("invalid approval id")
	}
	b, err := os.ReadFile(filepath.Join(s.Root, id+".json"))
	if err != nil {
		return Request{}, err
	}
	var r Request
	if err := json.Unmarshal(b, &r); err != nil {
		return Request{}, fmt.Errorf("decode approval: %w", err)
	}
	if (r.Status == StatusPending || r.Status == StatusApproved) && time.Now().UTC().After(r.ExpiresAt) {
		r.Status = StatusExpired
	}
	return r, nil
}

func (s Store) List(limit int) ([]Request, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	entries, err := os.ReadDir(s.Root)
	if errors.Is(err, os.ErrNotExist) {
		return []Request{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]Request, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		r, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s Store) Decide(id string, approve bool, actorUID uint32) (Request, error) {
	r, err := s.Get(id)
	if err != nil {
		return Request{}, err
	}
	if r.Status != StatusPending {
		return Request{}, fmt.Errorf("approval is %s", r.Status)
	}
	now := time.Now().UTC()
	if now.After(r.ExpiresAt) {
		r.Status = StatusExpired
		_ = s.write(r)
		return Request{}, errors.New("approval expired")
	}
	if approve {
		r.Status = StatusApproved
	} else {
		r.Status = StatusDenied
	}
	r.DecidedAt = &now
	r.DecidedBy = &actorUID
	if err := s.write(r); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (s Store) Consume(id, agent, capability, targetHash string, peerUID uint32) (Request, error) {
	r, err := s.Get(id)
	if err != nil {
		return Request{}, err
	}
	if r.Status != StatusApproved {
		return Request{}, fmt.Errorf("approval is %s", r.Status)
	}
	if time.Now().UTC().After(r.ExpiresAt) {
		r.Status = StatusExpired
		_ = s.write(r)
		return Request{}, errors.New("approval expired")
	}
	if r.Agent != agent || r.Capability != capability || r.TargetHash != targetHash || r.PeerUID != peerUID {
		return Request{}, errors.New("approval binding mismatch")
	}
	r.Status = StatusConsumed
	if err := s.write(r); err != nil {
		return Request{}, err
	}
	return r, nil
}

func (s Store) write(r Request) error {
	if s.Root == "" {
		return errors.New("approval root is required")
	}
	if !safeID(r.ID) {
		return errors.New("invalid approval id")
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	p := filepath.Join(s.Root, r.ID+".json")
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
