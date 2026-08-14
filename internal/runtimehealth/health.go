package runtimehealth

import (
	"sort"
	"strings"
	"time"
)

type Component struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	OK       bool   `json:"ok"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

type Snapshot struct {
	Ready      bool        `json:"ready"`
	Status     string      `json:"status"`
	Components []Component `json:"components"`
	CheckedAt  time.Time   `json:"checked_at"`
}

// Build converts component health into one deterministic readiness snapshot.
// Only a failed required component blocks readiness. Optional components can be
// unavailable without making the local governance/runtime core unusable.
func Build(components ...Component) Snapshot {
	items := append([]Component(nil), components...)
	for i := range items {
		items[i].Name = strings.TrimSpace(items[i].Name)
		items[i].Status = strings.TrimSpace(items[i].Status)
		items[i].Message = strings.TrimSpace(items[i].Message)
		if items[i].Status == "" {
			if items[i].OK {
				items[i].Status = "ok"
			} else {
				items[i].Status = "unavailable"
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	ready := true
	optionalFailure := false
	for _, item := range items {
		if item.Required && !item.OK {
			ready = false
		}
		if !item.Required && !item.OK {
			optionalFailure = true
		}
	}
	status := "ready"
	if !ready {
		status = "blocked"
	} else if optionalFailure {
		status = "degraded"
	}
	return Snapshot{Ready: ready, Status: status, Components: items, CheckedAt: time.Now().UTC()}
}
