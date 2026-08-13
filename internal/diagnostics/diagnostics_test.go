package diagnostics

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

func writeTestFile(t *testing.T, root, path string, data []byte) {
	t.Helper()
	p := rooted(root, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(p, data, 0o644); err != nil { t.Fatal(err) }
}

func healthyRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "/etc/os-release", []byte("NAME=\"KINGAI OS\"\nID=kingai\n"))
	for _, p := range []string{"/etc/kingai/policy.json", "/etc/kingai/system.json", "/etc/kingai/agents.json", "/etc/kingai/models.json"} {
		writeTestFile(t, root, p, []byte("{}"))
	}
	state, err := kingupdate.NewSlotState(kingupdate.SlotA, "0.1.0")
	if err != nil { t.Fatal(err) }
	b, _ := json.Marshal(state)
	writeTestFile(t, root, kingupdate.DefaultStatePath, b)
	writeTestFile(t, root, "/etc/kingai/update/root.json", []byte(`{"signed":{"_type":"root"}}`))
	writeTestFile(t, root, "/sys/firmware/efi/efivars/SecureBoot-test", []byte{0, 0, 0, 0, 1})
	return root
}

func TestHealthyReportScores100(t *testing.T) {
	root := healthyRoot(t)
	r := Run(Options{Root: root, Now: func() time.Time { return time.Unix(1, 0) }})
	if r.Status != "healthy" || r.Score != 100 {
		t.Fatalf("status=%s score=%d, want healthy/100: %+v", r.Status, r.Score, r.Checks)
	}
	if len(r.NextActions) != 0 { t.Fatalf("unexpected actions: %v", r.NextActions) }
}

func TestDaemonAndMalformedConfigBecomeCritical(t *testing.T) {
	root := healthyRoot(t)
	writeTestFile(t, root, "/etc/kingai/policy.json", []byte("{"))
	r := Run(Options{Root: root, DaemonError: errors.New("socket unavailable")})
	if r.Status != "critical" { t.Fatalf("status=%s, want critical", r.Status) }
	if r.Score >= 60 { t.Fatalf("score=%d, expected critical risk penalty", r.Score) }
	if len(r.NextActions) < 2 { t.Fatalf("expected remediation actions, got %v", r.NextActions) }
}

func TestPendingUpdateIsDegradedNotCritical(t *testing.T) {
	root := healthyRoot(t)
	state, err := kingupdate.NewSlotState(kingupdate.SlotA, "0.1.0")
	if err != nil { t.Fatal(err) }
	state, err = state.MarkPending("0.1.1")
	if err != nil { t.Fatal(err) }
	b, _ := json.Marshal(state)
	writeTestFile(t, root, kingupdate.DefaultStatePath, b)
	r := Run(Options{Root: root})
	if r.Status != "healthy" || r.Score != 92 {
		t.Fatalf("pending update should be visible but non-critical: status=%s score=%d", r.Status, r.Score)
	}
	found := false
	for _, c := range r.Checks { if c.ID == "ab-state" && c.Status == "warn" { found = true } }
	if !found { t.Fatal("missing A/B pending warning") }
}
