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

type Layer string

const (
	LayerContext      Layer = "M0"
	LayerWorking      Layer = "M1"
	LayerTask         Layer = "M2"
	LayerEpisodic     Layer = "M3"
	LayerSemantic     Layer = "M4"
	LayerUserOrg      Layer = "M5"
	LayerEvolution    Layer = "M6"
)

type Metadata struct {
	Agent           string     `json:"agent,omitempty"`
	Namespace       string     `json:"namespace,omitempty"`
	Layer           Layer      `json:"layer,omitempty"`
	Source          string     `json:"source,omitempty"`
	Confidence      float64    `json:"confidence,omitempty"`
	Importance      float64    `json:"importance,omitempty"`
	RetentionPolicy string     `json:"retention_policy,omitempty"`
	Jurisdiction    string     `json:"jurisdiction,omitempty"`
	CloudPolicy     string     `json:"cloud_policy,omitempty"`
	ModelPolicy     string     `json:"model_policy,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
}

type Record struct {
	ID              string          `json:"id"`
	Owner           string          `json:"owner"`
	Agent           string          `json:"agent,omitempty"`
	Namespace       string          `json:"namespace,omitempty"`
	Layer           Layer           `json:"layer"`
	Kind            string          `json:"kind"`
	Sensitivity     string          `json:"sensitivity"`
	Source          string          `json:"source,omitempty"`
	Confidence      float64         `json:"confidence,omitempty"`
	Importance      float64         `json:"importance,omitempty"`
	RetentionPolicy string          `json:"retention_policy,omitempty"`
	Jurisdiction    string          `json:"jurisdiction,omitempty"`
	CloudPolicy     string          `json:"cloud_policy,omitempty"`
	ModelPolicy     string          `json:"model_policy,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	Data            json.RawMessage `json:"data"`
}

type Query struct {
	Text        string `json:"text,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Layer       Layer  `json:"layer,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Sensitivity string `json:"sensitivity,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

type FileStore struct{ Root string }

func (s FileStore) Put(owner, kind, sensitivity string, data json.RawMessage) (Record, error) {
	return s.PutWithMetadata(owner, kind, sensitivity, Metadata{Layer: inferLayer(kind)}, data)
}

func (s FileStore) PutWithMetadata(owner, kind, sensitivity string, meta Metadata, data json.RawMessage) (Record, error) {
	if !safe(owner) { return Record{}, errors.New("invalid owner") }
	kind = strings.TrimSpace(kind)
	if kind == "" { kind = "semantic" }
	sensitivity = strings.TrimSpace(sensitivity)
	if sensitivity == "" { sensitivity = "private" }
	if len(data) == 0 || !json.Valid(data) { return Record{}, errors.New("memory data must be valid JSON") }
	if meta.Agent != "" && !safe(meta.Agent) { return Record{}, errors.New("invalid agent") }
	if meta.Namespace != "" && !safe(meta.Namespace) { return Record{}, errors.New("invalid namespace") }
	if meta.Layer == "" { meta.Layer = inferLayer(kind) }
	if !validLayer(meta.Layer) { return Record{}, errors.New("invalid memory layer") }
	if meta.Confidence < 0 || meta.Confidence > 1 || meta.Importance < 0 || meta.Importance > 1 { return Record{}, errors.New("confidence and importance must be between 0 and 1") }
	if err := validateMetadataText(meta); err != nil { return Record{}, err }
	id, err := randomID(); if err != nil { return Record{}, err }
	now := time.Now().UTC()
	r := Record{
		ID: id, Owner: owner, Agent: meta.Agent, Namespace: meta.Namespace, Layer: meta.Layer, Kind: kind, Sensitivity: sensitivity,
		Source: meta.Source, Confidence: meta.Confidence, Importance: meta.Importance, RetentionPolicy: meta.RetentionPolicy,
		Jurisdiction: meta.Jurisdiction, CloudPolicy: meta.CloudPolicy, ModelPolicy: meta.ModelPolicy,
		CreatedAt: now, UpdatedAt: now, ExpiresAt: meta.ExpiresAt, Data: append(json.RawMessage(nil), data...),
	}
	if err := s.write(r); err != nil { return Record{}, err }
	return r, nil
}

func (s FileStore) Get(owner, id string) (Record, error) {
	if !safe(owner) || !safe(id) { return Record{}, errors.New("invalid memory identifier") }
	b, err := os.ReadFile(filepath.Join(s.Root, owner, id+".json"))
	if err != nil { return Record{}, err }
	var r Record
	if err := json.Unmarshal(b, &r); err != nil { return Record{}, fmt.Errorf("decode memory: %w", err) }
	normalize(&r)
	if r.Owner != owner { return Record{}, errors.New("memory owner mismatch") }
	if expired(r, time.Now().UTC()) { return Record{}, os.ErrNotExist }
	return r, nil
}

func (s FileStore) List(owner string, limit int) ([]Record, error) {
	return s.Search(owner, Query{Limit: limit})
}

func (s FileStore) Search(owner string, q Query) ([]Record, error) {
	if !safe(owner) { return nil, errors.New("invalid owner") }
	if q.Agent != "" && !safe(q.Agent) { return nil, errors.New("invalid agent") }
	if q.Namespace != "" && !safe(q.Namespace) { return nil, errors.New("invalid namespace") }
	if q.Layer != "" && !validLayer(q.Layer) { return nil, errors.New("invalid memory layer") }
	if q.Limit <= 0 || q.Limit > 500 { q.Limit = 100 }
	dir := filepath.Join(s.Root, owner)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) { return []Record{}, nil }
	if err != nil { return nil, err }
	now := time.Now().UTC()
	needle := strings.ToLower(strings.TrimSpace(q.Text))
	out := make([]Record, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") { continue }
		b, err := os.ReadFile(filepath.Join(dir, e.Name())); if err != nil { return nil, err }
		var r Record; if err := json.Unmarshal(b, &r); err != nil { return nil, fmt.Errorf("decode %s: %w", e.Name(), err) }
		normalize(&r)
		if r.Owner != owner || expired(r, now) { continue }
		if q.Agent != "" && r.Agent != q.Agent { continue }
		if q.Namespace != "" && r.Namespace != q.Namespace { continue }
		if q.Layer != "" && r.Layer != q.Layer { continue }
		if q.Kind != "" && r.Kind != q.Kind { continue }
		if q.Sensitivity != "" && r.Sensitivity != q.Sensitivity { continue }
		if needle != "" {
			haystack := strings.ToLower(strings.Join([]string{r.Kind, r.Source, r.Agent, r.Namespace, string(r.Data)}, "\n"))
			if !strings.Contains(haystack, needle) { continue }
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Importance == out[j].Importance { return out[i].CreatedAt.After(out[j].CreatedAt) }
		return out[i].Importance > out[j].Importance
	})
	if len(out) > q.Limit { out = out[:q.Limit] }
	return out, nil
}

func (s FileStore) Promote(owner, id string, next Layer) (Record, error) {
	if !validLayer(next) { return Record{}, errors.New("invalid target memory layer") }
	r, err := s.Get(owner, id)
	if err != nil { return Record{}, err }
	if layerIndex(next) != layerIndex(r.Layer)+1 { return Record{}, fmt.Errorf("memory promotion must be adjacent: %s -> %s", r.Layer, next) }
	r.Layer = next
	r.UpdatedAt = time.Now().UTC()
	if err := s.write(r); err != nil { return Record{}, err }
	return r, nil
}

func (s FileStore) Delete(owner, id string) error {
	if !safe(owner) || !safe(id) { return errors.New("invalid memory identifier") }
	err := os.Remove(filepath.Join(s.Root, owner, id+".json"))
	if errors.Is(err, os.ErrNotExist) { return nil }
	return err
}

func (s FileStore) PurgeExpired(owner string) (int, error) {
	if !safe(owner) { return 0, errors.New("invalid owner") }
	dir := filepath.Join(s.Root, owner)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) { return 0, nil }
	if err != nil { return 0, err }
	now := time.Now().UTC(); removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") { continue }
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path); if err != nil { return removed, err }
		var r Record; if err := json.Unmarshal(b, &r); err != nil { return removed, err }
		if expired(r, now) { if err := os.Remove(path); err != nil { return removed, err }; removed++ }
	}
	return removed, nil
}

func (s FileStore) write(r Record) error {
	if s.Root == "" || !safe(r.Owner) || !safe(r.ID) { return errors.New("invalid memory storage path") }
	dir := filepath.Join(s.Root, r.Owner)
	if err := os.MkdirAll(dir, 0o700); err != nil { return err }
	b, err := json.MarshalIndent(r, "", "  "); if err != nil { return err }
	p := filepath.Join(dir, r.ID+".json"); tmp := p+".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil { return err }
	return os.Rename(tmp, p)
}

func normalize(r *Record) {
	if r.Layer == "" { r.Layer = inferLayer(r.Kind) }
	if r.UpdatedAt.IsZero() { r.UpdatedAt = r.CreatedAt }
}

func inferLayer(kind string) Layer {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "context": return LayerContext
	case "working": return LayerWorking
	case "task": return LayerTask
	case "episodic", "episode": return LayerEpisodic
	case "user", "organization", "org", "profile": return LayerUserOrg
	case "evolution": return LayerEvolution
	default: return LayerSemantic
	}
}

func validLayer(v Layer) bool { return layerIndex(v) >= 0 }
func layerIndex(v Layer) int {
	switch v { case LayerContext: return 0; case LayerWorking: return 1; case LayerTask: return 2; case LayerEpisodic: return 3; case LayerSemantic: return 4; case LayerUserOrg: return 5; case LayerEvolution: return 6; default: return -1 }
}
func expired(r Record, now time.Time) bool { return r.ExpiresAt != nil && !r.ExpiresAt.After(now) }

func validateMetadataText(meta Metadata) error {
	for name, value := range map[string]string{"source": meta.Source, "retention_policy": meta.RetentionPolicy, "jurisdiction": meta.Jurisdiction, "cloud_policy": meta.CloudPolicy, "model_policy": meta.ModelPolicy} {
		if len(value) > 512 || strings.ContainsRune(value, '\x00') { return fmt.Errorf("invalid %s", name) }
	}
	return nil
}

func safe(v string) bool {
	if v == "" || v == "." || v == ".." || strings.ContainsAny(v, "/\\") { return false }
	for _, r := range v { if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) { return false } }
	return true
}

func randomID() (string, error) { b := make([]byte, 16); if _, err := rand.Read(b); err != nil { return "", err }; return hex.EncodeToString(b), nil }
