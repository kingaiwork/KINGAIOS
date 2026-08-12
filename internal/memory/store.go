package memory

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

type Record struct {
	ID          string          `json:"id"`
	Owner       string          `json:"owner"`
	Kind        string          `json:"kind"`
	Sensitivity string          `json:"sensitivity"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	Data        json.RawMessage `json:"data"`
}

type FileStore struct{ Root string }

func (s FileStore) Put(owner, kind, sensitivity string, data json.RawMessage) (Record, error) {
	if !safe(owner) { return Record{}, errors.New("invalid owner") }
	if kind == "" { kind = "semantic" }
	if sensitivity == "" { sensitivity = "private" }
	id, err := randomID(); if err != nil { return Record{}, err }
	r := Record{ID:id, Owner:owner, Kind:kind, Sensitivity:sensitivity, CreatedAt:time.Now().UTC(), Data:data}
	b, err := json.MarshalIndent(r, "", "  "); if err != nil { return Record{}, err }
	dir := filepath.Join(s.Root, owner)
	if err := os.MkdirAll(dir, 0o700); err != nil { return Record{}, err }
	p := filepath.Join(dir, id+".json"); tmp := p+".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil { return Record{}, err }
	if err := os.Rename(tmp, p); err != nil { return Record{}, err }
	return r, nil
}

func (s FileStore) List(owner string, limit int) ([]Record, error) {
	if !safe(owner) { return nil, errors.New("invalid owner") }
	if limit <= 0 || limit > 500 { limit = 100 }
	dir := filepath.Join(s.Root, owner)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) { return []Record{}, nil }
	if err != nil { return nil, err }
	out := make([]Record, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") { continue }
		b, err := os.ReadFile(filepath.Join(dir, e.Name())); if err != nil { return nil, err }
		var r Record; if err := json.Unmarshal(b, &r); err != nil { return nil, fmt.Errorf("decode %s: %w", e.Name(), err) }
		if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) { continue }
		out = append(out, r)
	}
	sort.Slice(out, func(i,j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit { out = out[:limit] }
	return out, nil
}

func (s FileStore) Delete(owner, id string) error {
	if !safe(owner) || !safe(id) { return errors.New("invalid memory identifier") }
	err := os.Remove(filepath.Join(s.Root, owner, id+".json"))
	if errors.Is(err, os.ErrNotExist) { return nil }
	return err
}

func safe(v string) bool {
	if v == "" || v == "." || v == ".." || strings.ContainsAny(v, "/\\") { return false }
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) { return false }
	}
	return true
}

func randomID() (string, error) {
	b := make([]byte, 16); if _, err := rand.Read(b); err != nil { return "", err }
	return hex.EncodeToString(b), nil
}
