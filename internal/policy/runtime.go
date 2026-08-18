package policy

import (
	"errors"
	"strings"
	"sync"
)

var runtimePolicyRules struct {
	sync.RWMutex
	rules map[string]Rule
}

// SetRuntimeRules installs exact, runtime-discovered device capability rules.
// Runtime rules can add new device.* capabilities, but Evaluate always merges
// them with any static rule using the stricter risk/approval/owner settings.
func SetRuntimeRules(rules map[string]Rule) error {
	next := make(map[string]Rule, len(rules))
	for capability, rule := range rules {
		if !validRuntimeCapability(capability) {
			return errors.New("runtime policy capability must be an exact device.* id")
		}
		if rule.Risk < RiskRead || rule.Risk > RiskTrustRoot {
			return errors.New("runtime policy risk is out of range")
		}
		if rule.OwnerOnly {
			rule.ApprovalRequired = true
		}
		next[capability] = rule
	}
	runtimePolicyRules.Lock()
	runtimePolicyRules.rules = next
	runtimePolicyRules.Unlock()
	return nil
}

func effectiveRule(static map[string]Rule, capability string) (Rule, bool) {
	base, baseOK := static[capability]
	runtimeRule, runtimeOK := getRuntimeRule(capability)
	if !runtimeOK {
		return base, baseOK
	}
	if !baseOK {
		return runtimeRule, true
	}
	if runtimeRule.Risk > base.Risk {
		base.Risk = runtimeRule.Risk
	}
	base.ApprovalRequired = base.ApprovalRequired || runtimeRule.ApprovalRequired
	base.OwnerOnly = base.OwnerOnly || runtimeRule.OwnerOnly
	if base.OwnerOnly {
		base.ApprovalRequired = true
	}
	return base, true
}

func getRuntimeRule(capability string) (Rule, bool) {
	runtimePolicyRules.RLock()
	defer runtimePolicyRules.RUnlock()
	rule, ok := runtimePolicyRules.rules[capability]
	return rule, ok
}

func validRuntimeCapability(capability string) bool {
	if !strings.HasPrefix(capability, "device.") || len(capability) <= len("device.") || len(capability) > 128 {
		return false
	}
	if strings.Contains(capability, "..") || strings.ContainsAny(capability, "* /\\;$`?[]{}\x00\n\r") {
		return false
	}
	for _, r := range capability {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
