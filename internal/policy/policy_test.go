package policy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnknownCapabilityDenied(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "main", Capability: "unknown.action"})
	if got.Allowed { t.Fatal("unknown capability must be denied") }
}

func TestSafeReadAllowed(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "main", Capability: "filesystem.read"})
	if !got.Allowed { t.Fatalf("filesystem.read should be allowed: %s", got.Reason) }
}

func TestCriticalNeedsApproval(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "system", Capability: "boot.modify"})
	if got.Allowed || !got.ApprovalRequired { t.Fatal("boot.modify must require approval") }
}

func TestTrustRootNeedsOwner(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "system", Capability: "trust.modify", Approved: true})
	if got.Allowed { t.Fatal("trust.modify must remain owner-only") }
}

func TestTrustFlagsAreNotClientJSON(t *testing.T) {
	b, err := json.Marshal(Request{Agent:"main", Capability:"filesystem.read", Owner:true, Approved:true})
	if err != nil { t.Fatal(err) }
	if strings.Contains(string(b), "owner") || strings.Contains(string(b), "approved") {
		t.Fatalf("trust flags leaked into client JSON: %s", b)
	}
}
