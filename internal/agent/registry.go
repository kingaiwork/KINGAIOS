package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type Definition struct {
	ID           string   `json:"id"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities"`
}

type Config struct {
	Agents []Definition `json:"agents"`
	Rule   string       `json:"rule,omitempty"`
}

type Registry struct {
	agents map[string]map[string]struct{}
}

func Default() Registry {
	return New([]Definition{{ID: "main", Role: "safe-fallback", Capabilities: []string{"filesystem.read", "network.read"}}})
}

func New(defs []Definition) Registry {
	r := Registry{agents: make(map[string]map[string]struct{}, len(defs))}
	for _, d := range defs {
		if d.ID == "" { continue }
		caps := make(map[string]struct{}, len(d.Capabilities))
		for _, c := range d.Capabilities {
			if c != "" { caps[c] = struct{}{} }
		}
		r.agents[d.ID] = caps
	}
	return r
}

func Load(path string) (Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil { return Registry{}, err }
	var c Config
	if err := json.Unmarshal(b, &c); err != nil { return Registry{}, fmt.Errorf("decode agent registry: %w", err) }
	if len(c.Agents) == 0 { return Registry{}, errors.New("agent registry has no agents") }
	r := New(c.Agents)
	if len(r.agents) == 0 { return Registry{}, errors.New("agent registry has no valid agents") }
	return r, nil
}

func (r Registry) Allows(agentID, capability string) bool {
	caps, ok := r.agents[agentID]
	if !ok { return false }
	_, ok = caps[capability]
	return ok
}

func (r Registry) Has(agentID string) bool {
	_, ok := r.agents[agentID]
	return ok
}
