package main

import "testing"

func TestAgentIdentityBinding(t *testing.T) {
	tests := []struct {
		name, requested, username string
		uid uint32
		want bool
	}{
		{"normal user main", "main", "alice", 1000, true},
		{"normal user cannot spoof system", "system-ops", "alice", 1000, false},
		{"system service identity", "system-ops", "_kingai-system", 991, true},
		{"security service identity", "sec-ops", "_kingai-sec", 992, true},
		{"wrong service identity", "sec-ops", "_kingai-system", 991, false},
		{"root system ops", "system-ops", "root", 0, true},
		{"root unknown identity", "unknown", "root", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := agentIdentityAllowed(tt.requested, tt.username, tt.uid); got != tt.want { t.Fatalf("got %v want %v", got, tt.want) }
		})
	}
}
