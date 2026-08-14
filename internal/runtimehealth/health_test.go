package runtimehealth

import "testing"

func TestBuildReadyWhenRequiredComponentsHealthy(t *testing.T) {
	s := Build(
		Component{Name: "policy", Required: true, OK: true},
		Component{Name: "agents", Required: true, OK: true},
		Component{Name: "models", Required: false, OK: true, Status: "not_configured"},
	)
	if !s.Ready || s.Status != "ready" {
		t.Fatalf("snapshot=%#v", s)
	}
	if len(s.Components) != 3 || s.Components[0].Name != "agents" || s.Components[1].Name != "models" || s.Components[2].Name != "policy" {
		t.Fatalf("components=%#v", s.Components)
	}
}

func TestBuildDegradedWhenOptionalComponentUnavailable(t *testing.T) {
	s := Build(
		Component{Name: "policy", Required: true, OK: true},
		Component{Name: "execd", Required: false, OK: false, Status: "offline"},
	)
	if !s.Ready || s.Status != "degraded" {
		t.Fatalf("snapshot=%#v", s)
	}
}

func TestBuildBlockedWhenRequiredComponentUnavailable(t *testing.T) {
	s := Build(
		Component{Name: "policy", Required: true, OK: true},
		Component{Name: "execd", Required: true, OK: false, Status: "offline"},
	)
	if s.Ready || s.Status != "blocked" {
		t.Fatalf("snapshot=%#v", s)
	}
}
