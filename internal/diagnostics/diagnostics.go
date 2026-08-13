package diagnostics

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/tufclient"
	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

const (
	RemediationObserve  = "observe"
	RemediationSafeAuto = "safe-auto"
	RemediationApproval = "approval-required"
	RemediationManual   = "manual"
)

type Check struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	Severity        string `json:"severity"`
	Summary         string `json:"summary"`
	Recommendation  string `json:"recommendation,omitempty"`
	ActionID        string `json:"action_id,omitempty"`
	RemediationMode string `json:"remediation_mode,omitempty"`
}

type Report struct {
	Schema      int            `json:"schema"`
	Product     string         `json:"product"`
	GeneratedAt time.Time      `json:"generated_at"`
	Score       int            `json:"score"`
	Status      string         `json:"status"`
	Checks      []Check        `json:"checks"`
	NextActions []string       `json:"next_actions,omitempty"`
	Repairs     []RepairResult `json:"repairs,omitempty"`
}

type Options struct {
	Root        string
	DaemonError error
	Now         func() time.Time
}

func Run(opts Options) Report {
	now := time.Now
	if opts.Now != nil { now = opts.Now }
	referenceTime := now().UTC()
	r := Report{Schema: 1, Product: "KINGAI OS", GeneratedAt: referenceTime, Score: 100}

	if opts.DaemonError != nil {
		r.add(Check{ID: "daemon", Status: "fail", Severity: "critical", Summary: "kingaid is not reachable: " + opts.DaemonError.Error(), Recommendation: "Inspect systemctl status kingaid and journalctl -u kingaid before allowing autonomous execution.", ActionID: "kingaid.recover", RemediationMode: RemediationApproval})
	} else {
		r.add(Check{ID: "daemon", Status: "pass", Severity: "critical", Summary: "kingaid local health endpoint is reachable"})
	}

	osRelease := rooted(opts.Root, "/etc/os-release")
	if b, err := os.ReadFile(osRelease); err != nil {
		r.add(Check{ID: "os-identity", Status: statusForMissing(err), Severity: "warning", Summary: "KINGAI OS identity could not be verified", Recommendation: "Verify /etc/os-release is installed from the KINGAI image.", ActionID: "os.identity.verify", RemediationMode: RemediationManual})
	} else if !strings.Contains(string(b), "ID=kingai") || !strings.Contains(string(b), "NAME=\"KINGAI OS\"") {
		r.add(Check{ID: "os-identity", Status: "warn", Severity: "warning", Summary: "host does not identify as a complete KINGAI OS installation", Recommendation: "Use this result as informational in a development checkout; installed images should contain KINGAI os-release metadata.", ActionID: "os.identity.verify", RemediationMode: RemediationManual})
	} else {
		r.add(Check{ID: "os-identity", Status: "pass", Severity: "warning", Summary: "KINGAI OS identity is consistent"})
	}

	for _, item := range []struct{ id, path string }{
		{"policy-config", "/etc/kingai/policy.json"},
		{"system-config", "/etc/kingai/system.json"},
		{"agent-config", "/etc/kingai/agents.json"},
		{"model-config", "/etc/kingai/models.json"},
	} {
		r.add(checkJSON(rooted(opts.Root, item.path), item.id))
	}

	r.add(checkABState(rooted(opts.Root, kingupdate.DefaultStatePath)))
	r.add(checkTUFRoot(rooted(opts.Root, "/etc/kingai/update/root.json"), referenceTime))
	r.add(checkSecureBoot(opts.Root))

	r.finalize()
	return r
}

func (r *Report) add(c Check) {
	r.Checks = append(r.Checks, c)
	switch c.Status {
	case "warn":
		r.Score -= 8
	case "fail":
		if c.Severity == "critical" { r.Score -= 25 } else { r.Score -= 15 }
	}
	if r.Score < 0 { r.Score = 0 }
}

func (r *Report) finalize() {
	criticalFailure := false
	hasProblem := false
	for _, c := range r.Checks {
		if c.Status == "warn" || c.Status == "fail" { hasProblem = true }
		if c.Status == "fail" && c.Severity == "critical" { criticalFailure = true }
		if c.Recommendation != "" && (c.Status == "warn" || c.Status == "fail") {
			r.NextActions = append(r.NextActions, c.Recommendation)
		}
	}
	if criticalFailure || r.Score < 60 {
		r.Status = "critical"
	} else if hasProblem || r.Score < 90 {
		r.Status = "degraded"
	} else {
		r.Status = "healthy"
	}
}

func checkJSON(path, id string) Check {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return Check{ID: id, Status: "warn", Severity: "warning", Summary: "configuration exists but is not readable by this user", Recommendation: "Run kingai doctor with sufficient local privileges to validate protected configuration.", ActionID: "config.inspect", RemediationMode: RemediationManual}
		}
		return Check{ID: id, Status: "fail", Severity: "warning", Summary: fmt.Sprintf("required configuration is unavailable: %s", path), Recommendation: "Restore the missing KINGAI configuration from the installed image or a verified configuration backup.", ActionID: "config.restore", RemediationMode: RemediationApproval}
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return Check{ID: id, Status: "fail", Severity: "critical", Summary: "configuration contains invalid JSON: " + err.Error(), Recommendation: "Do not start autonomous agents until the malformed configuration is replaced with a validated copy.", ActionID: "config.replace", RemediationMode: RemediationApproval}
	}
	return Check{ID: id, Status: "pass", Severity: "warning", Summary: "configuration is present and valid JSON"}
}

func checkABState(path string) Check {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Check{ID: "ab-state", Status: "info", Severity: "warning", Summary: "A/B state is not present (normal for live/development environments)"}
		}
		if errors.Is(err, os.ErrPermission) {
			return Check{ID: "ab-state", Status: "warn", Severity: "warning", Summary: "A/B state is protected and could not be inspected", Recommendation: "Run the diagnostic locally with root privileges before staging or approving an update.", ActionID: "update.state.inspect", RemediationMode: RemediationManual}
		}
		return Check{ID: "ab-state", Status: "fail", Severity: "warning", Summary: "A/B state could not be read: " + err.Error(), Recommendation: "Inspect encrypted STATE availability before attempting an update.", ActionID: "update.state.inspect", RemediationMode: RemediationManual}
	}
	var s kingupdate.SlotState
	if err := json.Unmarshal(b, &s); err != nil {
		return Check{ID: "ab-state", Status: "fail", Severity: "critical", Summary: "A/B state is malformed JSON", Recommendation: "Use KINGAI Recovery to inspect STATE; do not manually switch boot slots.", ActionID: "recovery.inspect-state", RemediationMode: RemediationApproval}
	}
	if err := s.Validate(); err != nil {
		return Check{ID: "ab-state", Status: "fail", Severity: "critical", Summary: "A/B state violates invariants: " + err.Error(), Recommendation: "Boot KINGAI Recovery and inspect/repair the boot state before any further update.", ActionID: "recovery.inspect-state", RemediationMode: RemediationApproval}
	}
	if s.RollbackRequired {
		return Check{ID: "ab-state", Status: "fail", Severity: "critical", Summary: fmt.Sprintf("slot %s requires rollback after %d boot attempts", s.PendingSlot, s.BootAttempts), Recommendation: "Use the controlled rollback/recovery path; do not confirm the pending slot.", ActionID: "update.rollback", RemediationMode: RemediationApproval}
	}
	if s.PendingSlot != "" {
		return Check{ID: "ab-state", Status: "warn", Severity: "warning", Summary: fmt.Sprintf("update %s is pending on slot %s", s.PendingVersion, s.PendingSlot), Recommendation: "Allow the pending slot to reach the userspace health checkpoint; if it cannot, the boot controller should fall back automatically.", ActionID: "update.await-health", RemediationMode: RemediationObserve}
	}
	return Check{ID: "ab-state", Status: "pass", Severity: "critical", Summary: fmt.Sprintf("slot %s (%s) is confirmed", s.ActiveSlot, s.ActiveVersion)}
}

func checkTUFRoot(path string, now time.Time) Check {
	st, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Check{ID: "tuf-root", Status: "warn", Severity: "warning", Summary: "production TUF bootstrap root is not provisioned", Recommendation: "Provision the pinned TUF root out-of-band before enabling production network updates.", ActionID: "tuf.provision-root", RemediationMode: RemediationManual}
		}
		if errors.Is(err, os.ErrPermission) {
			return Check{ID: "tuf-root", Status: "warn", Severity: "warning", Summary: "TUF root exists but is not readable by this diagnostic user", Recommendation: "Validate the protected TUF root locally before enabling update checks.", ActionID: "tuf.inspect-root", RemediationMode: RemediationManual}
		}
		return Check{ID: "tuf-root", Status: "fail", Severity: "warning", Summary: "TUF root cannot be inspected: " + err.Error(), Recommendation: "Restore the trusted root only from an authenticated offline source.", ActionID: "tuf.restore-root", RemediationMode: RemediationManual}
	}
	if !st.Mode().IsRegular() {
		return Check{ID: "tuf-root", Status: "fail", Severity: "critical", Summary: "TUF bootstrap root is not a regular file (symlinks and special files are refused)", Recommendation: "Disable network updates and restore root.json as a verified regular file from an offline source.", ActionID: "tuf.restore-root", RemediationMode: RemediationManual}
	}
	if dst, err := os.Stat(filepath.Dir(path)); err != nil {
		return Check{ID: "tuf-root", Status: "fail", Severity: "critical", Summary: "TUF trust directory cannot be inspected: " + err.Error(), Recommendation: "Inspect and restore the protected TUF trust directory before enabling updates.", ActionID: "tuf.restore-root", RemediationMode: RemediationManual}
	} else if dst.Mode().Perm()&0o022 != 0 {
		return Check{ID: "tuf-root", Status: "fail", Severity: "critical", Summary: fmt.Sprintf("TUF trust directory is writable by group/other (mode %04o)", dst.Mode().Perm()), Recommendation: "Remove group/other write access from the TUF trust directory and re-verify the pinned root against the offline trust source.", ActionID: "tuf.harden-trust-directory", RemediationMode: RemediationApproval}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return Check{ID: "tuf-root", Status: "warn", Severity: "warning", Summary: "TUF root exists but is not readable by this diagnostic user", Recommendation: "Validate the protected TUF root locally before enabling update checks.", ActionID: "tuf.inspect-root", RemediationMode: RemediationManual}
		}
		return Check{ID: "tuf-root", Status: "fail", Severity: "warning", Summary: "TUF root cannot be read: " + err.Error(), Recommendation: "Restore the trusted root only from an authenticated offline source.", ActionID: "tuf.restore-root", RemediationMode: RemediationManual}
	}
	info, err := tufclient.ValidateTrustedRootIntegrity(b)
	if err != nil {
		return Check{ID: "tuf-root", Status: "fail", Severity: "critical", Summary: "TUF bootstrap root signature threshold validation failed: " + err.Error(), Recommendation: "Disable network updates and restore a self-consistent, threshold-signed root.json from the authenticated offline trust source.", ActionID: "tuf.restore-root", RemediationMode: RemediationManual}
	}
	if st.Mode().Perm()&0o022 != 0 {
		return Check{ID: "tuf-root", Status: "fail", Severity: "critical", Summary: fmt.Sprintf("signature-valid TUF bootstrap root is writable by group/other (mode %04o)", st.Mode().Perm()), Recommendation: "Remove group/other write permission; the compiled safe-auto repair may do this only after revalidating the root signature threshold.", ActionID: "tuf.harden-root-permissions", RemediationMode: RemediationSafeAuto}
	}
	if !info.Expires.After(now) {
		return Check{ID: "tuf-root", Status: "warn", Severity: "warning", Summary: fmt.Sprintf("TUF root v%d is signature-valid but expired at %s", info.Version, info.Expires.Format(time.RFC3339)), Recommendation: "Allow only verified TUF root rotation to a newer root; do not replace the trust anchor from the network or bypass TUF verification.", ActionID: "tuf.rotate-root", RemediationMode: RemediationApproval}
	}
	if info.Expires.Sub(now) <= 30*24*time.Hour {
		return Check{ID: "tuf-root", Status: "warn", Severity: "warning", Summary: fmt.Sprintf("TUF root v%d is signature-valid but expires soon at %s", info.Version, info.Expires.Format(time.RFC3339)), Recommendation: "Prepare and verify the next offline-signed TUF root rotation before this root expires.", ActionID: "tuf.rotate-root", RemediationMode: RemediationApproval}
	}
	return Check{ID: "tuf-root", Status: "pass", Severity: "critical", Summary: fmt.Sprintf("TUF root v%d passes self-signature threshold %d with %d trusted keys; expires %s", info.Version, info.Threshold, info.KeyCount, info.Expires.Format(time.RFC3339))}
}

func checkSecureBoot(root string) Check {
	efi := rooted(root, "/sys/firmware/efi")
	if _, err := os.Stat(efi); err != nil {
		return Check{ID: "secure-boot", Status: "info", Severity: "warning", Summary: "EFI runtime is not visible; Secure Boot state cannot be measured here", ActionID: "secure-boot.measure", RemediationMode: RemediationObserve}
	}
	matches, _ := filepath.Glob(filepath.Join(efi, "efivars", "SecureBoot-*"))
	if len(matches) == 0 {
		return Check{ID: "secure-boot", Status: "warn", Severity: "warning", Summary: "EFI is active but the SecureBoot variable is unavailable", Recommendation: "Verify firmware Secure Boot state before promoting this machine to a production execution node.", ActionID: "secure-boot.verify", RemediationMode: RemediationManual}
	}
	b, err := os.ReadFile(matches[0])
	if err != nil || len(b) < 5 {
		return Check{ID: "secure-boot", Status: "warn", Severity: "warning", Summary: "Secure Boot variable could not be decoded", Recommendation: "Verify Secure Boot from firmware or mokutil before production use.", ActionID: "secure-boot.verify", RemediationMode: RemediationManual}
	}
	if b[4] != 1 {
		return Check{ID: "secure-boot", Status: "warn", Severity: "warning", Summary: "Secure Boot is disabled", Recommendation: "Enable Secure Boot for production nodes after validating the signed KINGAI boot chain.", ActionID: "secure-boot.enable", RemediationMode: RemediationManual}
	}
	return Check{ID: "secure-boot", Status: "pass", Severity: "critical", Summary: "firmware reports Secure Boot enabled"}
}

func rooted(root, p string) string {
	if root == "" { return p }
	return filepath.Join(root, strings.TrimPrefix(filepath.Clean(p), string(filepath.Separator)))
}

func statusForMissing(err error) string {
	if errors.Is(err, os.ErrPermission) { return "warn" }
	return "info"
}
