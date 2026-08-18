package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type RiskLevel int

const (
	RiskRead RiskLevel = iota
	RiskUser
	RiskApplication
	RiskSystem
	RiskSecurity
	RiskCritical
	RiskTrustRoot
)

type Rule struct {
	Risk             RiskLevel `json:"risk"`
	ApprovalRequired bool      `json:"approval_required"`
	OwnerOnly        bool      `json:"owner_only"`
}

type Policy struct {
	Rules map[string]Rule `json:"rules"`
}

// Request contains the public policy request plus daemon-populated trust context.
// Owner and Approved are intentionally excluded from JSON so an untrusted client
// cannot self-assert authorization.
type Request struct {
	Agent      string `json:"agent"`
	Capability string `json:"capability"`
	Target     string `json:"target,omitempty"`
	Owner      bool   `json:"-"`
	Approved   bool   `json:"-"`
}

type Result struct {
	Allowed          bool      `json:"allowed"`
	ApprovalRequired bool      `json:"approval_required"`
	Risk             RiskLevel `json:"risk"`
	Reason           string    `json:"reason"`
}

func Default() Policy {
	return Policy{Rules: map[string]Rule{
		"filesystem.read":  {Risk: RiskRead},
		"network.read":     {Risk: RiskRead},
		"audit.read":       {Risk: RiskRead},
		"filesystem.write": {Risk: RiskUser},
		"process.execute":  {Risk: RiskApplication},
		"service.restart":  {Risk: RiskApplication},
		"package.install":  {Risk: RiskSystem, ApprovalRequired: true},
		"network.modify":   {Risk: RiskSecurity, ApprovalRequired: true},
		"security.modify":  {Risk: RiskSecurity, ApprovalRequired: true},
		"boot.modify":      {Risk: RiskCritical, ApprovalRequired: true},
		"disk.raw":         {Risk: RiskCritical, ApprovalRequired: true},
		"trust.modify":     {Risk: RiskTrustRoot, ApprovalRequired: true, OwnerOnly: true},
	}}
}

func Load(path string) (Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	if err := json.Unmarshal(b, &p); err != nil {
		return Policy{}, fmt.Errorf("decode policy: %w", err)
	}
	if len(p.Rules) == 0 {
		return Policy{}, errors.New("policy has no rules")
	}
	return p, nil
}

func (p Policy) Evaluate(r Request) Result {
	if r.Agent == "" {
		return Result{Reason: "agent identity is required"}
	}
	if r.Capability == "" {
		return Result{Reason: "capability is required"}
	}
	rule, ok := effectiveRule(p.Rules, r.Capability)
	if !ok {
		return Result{Reason: "unknown capability: default deny"}
	}
	res := Result{Risk: rule.Risk}
	if rule.OwnerOnly && !r.Owner {
		res.ApprovalRequired = true
		res.Reason = "owner authorization required"
		return res
	}
	if rule.ApprovalRequired && !r.Approved {
		res.ApprovalRequired = true
		res.Reason = "explicit approval required"
		return res
	}
	res.Allowed = true
	res.Reason = "allowed by policy"
	return res
}
