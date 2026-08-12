package model

import (
	"errors"
	"sort"
)

type Candidate struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Local        bool     `json:"local"`
	Available    bool     `json:"available"`
	Capabilities []string `json:"capabilities"`
	Priority     int      `json:"priority"`
	LatencyMS    int      `json:"latency_ms,omitempty"`
	CostClass    int      `json:"cost_class,omitempty"`
}

type Request struct {
	Capability string `json:"capability"`
	Private    bool   `json:"private"`
	Offline    bool   `json:"offline"`
}

var ErrNoModel = errors.New("no eligible model")

func Select(req Request, candidates []Candidate) (Candidate, error) {
	eligible := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if !c.Available || c.ID == "" || !supports(c, req.Capability) {
			continue
		}
		if (req.Private || req.Offline) && !c.Local {
			continue
		}
		eligible = append(eligible, c)
	}
	if len(eligible) == 0 {
		return Candidate{}, ErrNoModel
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		a, b := score(req, eligible[i]), score(req, eligible[j])
		if a == b { return eligible[i].ID < eligible[j].ID }
		return a > b
	})
	return eligible[0], nil
}

func supports(c Candidate, capability string) bool {
	if capability == "" { return true }
	for _, v := range c.Capabilities {
		if v == capability || v == "*" { return true }
	}
	return false
}

func score(req Request, c Candidate) int {
	s := c.Priority * 1000
	if c.Local { s += 50 }
	if req.Private && c.Local { s += 5000 }
	s -= c.LatencyMS
	s -= c.CostClass * 100
	return s
}
