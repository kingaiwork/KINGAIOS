package policy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnknownCapabilityDenied(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "main", Capability: "unknown.action"})
	if got.Allowed {
		t.Fatal("unknown capability must be denied")
	}
}

func TestSafeReadAllowed(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "main", Capability: "filesystem.read"})
	if !got.Allowed {
		t.Fatalf("filesystem.read should be allowed: %s", got.Reason)
	}
}

func TestCriticalNeedsApproval(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "system", Capability: "boot.modify"})
	if got.Allowed || !got.ApprovalRequired {
		t.Fatal("boot.modify must require approval")
	}
}

func TestTrustRootNeedsOwner(t *testing.T) {
	got := Default().Evaluate(Request{Agent: "system", Capability: "trust.modify", Approved: true})
	if got.Allowed {
		t.Fatal("trust.modify must remain owner-only")
	}
}

func TestTrustFlagsAreNotClientJSON(t *testing.T) {
	b, err := json.Marshal(Request{Agent: "main", Capability: "filesystem.read", Owner: true, Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "owner") || strings.Contains(string(b), "approved") {
		t.Fatalf("trust flags leaked into client JSON: %s", b)
	}
}

func TestRuntimeDeviceRuleCanAddCapability(t *testing.T) {
	t.Cleanup(func() { _ = SetRuntimeRules(nil) })
	if err := SetRuntimeRules(map[string]Rule{
		"device.telemetry.read": {Risk: RiskRead},
	}); err != nil {
		t.Fatal(err)
	}
	got := Default().Evaluate(Request{Agent: "system", Capability: "device.telemetry.read"})
	if !got.Allowed || got.Risk != RiskRead {
		t.Fatalf("runtime device rule should be allowed: %+v", got)
	}
}

func TestRuntimeRuleCannotWeakenStaticPolicy(t *testing.T) {
	t.Cleanup(func() { _ = SetRuntimeRules(nil) })
	if err := SetRuntimeRules(map[string]Rule{
		"device.power.control": {Risk: RiskUser},
	}); err != nil {
		t.Fatal(err)
	}
	p := Default()
	p.Rules["device.power.control"] = Rule{Risk: RiskSecurity, ApprovalRequired: true, OwnerOnly: true}
	got := p.Evaluate(Request{Agent: "system", Capability: "device.power.control", Approved: true})
	if got.Allowed || !got.ApprovalRequired || got.Risk != RiskSecurity || got.Reason != "owner authorization required" {
		t.Fatalf("runtime rule weakened static policy: %+v", got)
	}
}

func TestRuntimeRuleRejectsWildcard(t *testing.T) {
	t.Cleanup(func() { _ = SetRuntimeRules(nil) })
	if err := SetRuntimeRules(map[string]Rule{"device.*": {Risk: RiskRead}}); err == nil {
		t.Fatal("runtime policy must contain exact device capabilities, not wildcards")
	}
}
