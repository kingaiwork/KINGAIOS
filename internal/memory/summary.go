package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Summary is metadata-only. It is intended for trusted local UI surfaces that
// need to explain Memory state without receiving record payloads.
type Summary struct {
	Total         int            `json:"total"`
	ByLayer       map[string]int `json:"by_layer"`
	BySensitivity map[string]int `json:"by_sensitivity"`
	Expiring      int            `json:"expiring"`
}

type summaryRecord struct {
	Owner       string     `json:"owner"`
	Layer       Layer      `json:"layer"`
	Kind        string     `json:"kind"`
	Sensitivity string     `json:"sensitivity"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Summarize scans one owner namespace and counts only non-expired metadata.
// Record payloads are intentionally not represented in summaryRecord and
// therefore cannot be returned by this API.
func (s FileStore) Summarize(owner string) (Summary, error) {
	if !safe(owner) {
		return Summary{}, errors.New("invalid owner")
	}
	out := Summary{ByLayer: map[string]int{}, BySensitivity: map[string]int{}}
	dir := filepath.Join(s.Root, owner)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return Summary{}, err
	}
	now := time.Now().UTC()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return Summary{}, err
		}
		var record summaryRecord
		if err := json.Unmarshal(b, &record); err != nil {
			return Summary{}, err
		}
		if record.Owner != owner {
			continue
		}
		if record.ExpiresAt != nil && !record.ExpiresAt.After(now) {
			continue
		}
		if record.Layer == "" {
			record.Layer = inferLayer(record.Kind)
		}
		if !validLayer(record.Layer) {
			continue
		}
		sensitivity := strings.TrimSpace(record.Sensitivity)
		if sensitivity == "" {
			sensitivity = "private"
		}
		out.Total++
		out.ByLayer[string(record.Layer)]++
		out.BySensitivity[sensitivity]++
		if record.ExpiresAt != nil {
			out.Expiring++
		}
	}
	return out, nil
}
