package diagnostics

import (
	"fmt"
	"os"

	"github.com/kingaiwork/KINGAIOS/internal/tufclient"
)

type RepairResult struct {
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
	Summary  string `json:"summary"`
}

// ApplySafeRepairs executes only actions explicitly classified as safe-auto by
// the diagnostic engine. Every action is independently revalidated immediately
// before mutation; a stale report can never authorize a different state.
func ApplySafeRepairs(root string, report Report) []RepairResult {
	var results []RepairResult
	for _, check := range report.Checks {
		if check.Status != "fail" && check.Status != "warn" { continue }
		if check.RemediationMode != RemediationSafeAuto || check.ActionID == "" { continue }
		switch check.ActionID {
		case "tuf.harden-root-permissions":
			results = append(results, hardenTUFRootPermissions(root))
		default:
			results = append(results, RepairResult{ActionID: check.ActionID, Status: "blocked", Summary: "action is not present in the compiled safe-auto allowlist"})
		}
	}
	return results
}

func hardenTUFRootPermissions(root string) RepairResult {
	const action = "tuf.harden-root-permissions"
	path := rooted(root, "/etc/kingai/update/root.json")
	st, err := os.Lstat(path)
	if err != nil {
		return RepairResult{ActionID: action, Status: "failed", Summary: "cannot inspect TUF root before permission repair: " + err.Error()}
	}
	if !st.Mode().IsRegular() {
		return RepairResult{ActionID: action, Status: "blocked", Summary: "TUF root is not a regular file; safe permission repair refused"}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return RepairResult{ActionID: action, Status: "failed", Summary: "cannot read TUF root before permission repair: " + err.Error()}
	}
	if _, err := tufclient.ValidateTrustedRootIntegrity(b); err != nil {
		return RepairResult{ActionID: action, Status: "blocked", Summary: "TUF root signature threshold is not valid; permission-only repair refused: " + err.Error()}
	}
	oldMode := st.Mode().Perm()
	newMode := oldMode &^ 0o022
	if oldMode == newMode {
		return RepairResult{ActionID: action, Status: "noop", Summary: fmt.Sprintf("TUF root permissions are already hardened (%04o)", oldMode)}
	}
	if err := os.Chmod(path, newMode); err != nil {
		return RepairResult{ActionID: action, Status: "failed", Summary: "failed to harden TUF root permissions: " + err.Error()}
	}
	return RepairResult{ActionID: action, Status: "applied", Summary: fmt.Sprintf("TUF root permissions hardened from %04o to %04o after signature revalidation", oldMode, newMode)}
}
