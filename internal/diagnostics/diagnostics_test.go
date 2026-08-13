package diagnostics

import (
	"crypto"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"

	kingupdate "github.com/kingaiwork/KINGAIOS/internal/update"
)

func writeTestFile(t *testing.T, root, path string, data []byte) {
	t.Helper()
	p := rooted(root, path)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(p, data, 0o644); err != nil { t.Fatal(err) }
}

func signedRoot(t *testing.T, expires time.Time) []byte {
	t.Helper()
	root := metadata.Root(expires.UTC())
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil { t.Fatal(err) }
	key, err := metadata.KeyFromPublicKey(private.Public())
	if err != nil { t.Fatal(err) }
	if err := root.Signed.AddKey(key, metadata.ROOT); err != nil { t.Fatal(err) }
	signer, err := signature.LoadSigner(private, crypto.Hash(0))
	if err != nil { t.Fatal(err) }
	if _, err := root.Sign(signer); err != nil { t.Fatal(err) }
	b, err := root.ToBytes(false)
	if err != nil { t.Fatal(err) }
	return b
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
	writeTestFile(t, root, "/etc/kingai/update/root.json", signedRoot(t, time.Now().UTC().Add(365*24*time.Hour)))
	writeTestFile(t, root, "/sys/firmware/efi/efivars/SecureBoot-test", []byte{0, 0, 0, 0, 1})
	return root
}

func findCheck(t *testing.T, report Report, id string) Check {
	t.Helper()
	for _, c := range report.Checks { if c.ID == id { return c } }
	t.Fatalf("check %q not found", id)
	return Check{}
}

func tamperRootVersion(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil { t.Fatal(err) }
	signed := doc["signed"].(map[string]any)
	signed["version"] = float64(99)
	tampered, _ := json.Marshal(doc)
	if err := os.WriteFile(path, tampered, 0o644); err != nil { t.Fatal(err) }
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
	if c := findCheck(t, r, "daemon"); c.ActionID != "kingaid.recover" || c.RemediationMode != RemediationApproval {
		t.Fatalf("daemon remediation classification is not approval-gated: %+v", c)
	}
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
	if r.Status != "degraded" || r.Score != 92 {
		t.Fatalf("pending update should be visible and degraded but non-critical: status=%s score=%d", r.Status, r.Score)
	}
	c := findCheck(t, r, "ab-state")
	if c.Status != "warn" || c.ActionID != "update.await-health" || c.RemediationMode != RemediationObserve {
		t.Fatalf("pending update remediation classification is unsafe: %+v", c)
	}
}

func TestTamperedTUFRootMakesReportCritical(t *testing.T) {
	root := healthyRoot(t)
	path := rooted(root, "/etc/kingai/update/root.json")
	tamperRootVersion(t, path)
	r := Run(Options{Root: root})
	if r.Status != "critical" { t.Fatalf("tampered trust root must be critical: %+v", r.Checks) }
	c := findCheck(t, r, "tuf-root")
	if c.Status != "fail" || c.Severity != "critical" || c.ActionID != "tuf.restore-root" || c.RemediationMode != RemediationManual {
		t.Fatalf("tampered TUF root must remain manual/critical: %+v", c)
	}
}

func TestSafeAutoHardensOnlySignatureValidTUFRoot(t *testing.T) {
	root := healthyRoot(t)
	path := rooted(root, "/etc/kingai/update/root.json")
	if err := os.Chmod(path, 0o666); err != nil { t.Fatal(err) }
	r := Run(Options{Root: root})
	c := findCheck(t, r, "tuf-root")
	if r.Status != "critical" || c.ActionID != "tuf.harden-root-permissions" || c.RemediationMode != RemediationSafeAuto {
		t.Fatalf("valid but writable TUF root should expose only safe permission hardening: %+v", c)
	}
	repairs := ApplySafeRepairs(root, r)
	if len(repairs) != 1 || repairs[0].Status != "applied" { t.Fatalf("safe repair not applied: %+v", repairs) }
	st, err := os.Stat(path); if err != nil { t.Fatal(err) }
	if st.Mode().Perm()&0o022 != 0 { t.Fatalf("unsafe write bits remain after repair: %04o", st.Mode().Perm()) }
	after := Run(Options{Root: root})
	if after.Status != "healthy" { t.Fatalf("signature-valid root should recover after permission hardening: %+v", after.Checks) }
}

func TestSafeAutoNeverMasksTamperedWritableTUFRoot(t *testing.T) {
	root := healthyRoot(t)
	path := rooted(root, "/etc/kingai/update/root.json")
	tamperRootVersion(t, path)
	if err := os.Chmod(path, 0o666); err != nil { t.Fatal(err) }
	r := Run(Options{Root: root})
	c := findCheck(t, r, "tuf-root")
	if c.ActionID != "tuf.restore-root" || c.RemediationMode != RemediationManual {
		t.Fatalf("tampered root was incorrectly offered safe-auto repair: %+v", c)
	}
	if repairs := ApplySafeRepairs(root, r); len(repairs) != 0 { t.Fatalf("tampered root triggered automatic mutation: %+v", repairs) }
	st, err := os.Stat(path); if err != nil { t.Fatal(err) }
	if st.Mode().Perm() != 0o666 { t.Fatalf("tampered root permissions were unexpectedly changed: %04o", st.Mode().Perm()) }
}

func TestExpiredSignedTUFRootIsDegradedWithRotationAdvice(t *testing.T) {
	root := healthyRoot(t)
	now := time.Now().UTC()
	writeTestFile(t, root, "/etc/kingai/update/root.json", signedRoot(t, now.Add(-time.Hour)))
	r := Run(Options{Root: root, Now: func() time.Time { return now }})
	if r.Status != "degraded" { t.Fatalf("expired but signature-valid root should be degraded, got %s", r.Status) }
	c := findCheck(t, r, "tuf-root")
	if c.Status != "warn" || c.ActionID != "tuf.rotate-root" || c.RemediationMode != RemediationApproval || c.Recommendation == "" {
		t.Fatalf("expired TUF root must request approval-gated rotation: %+v", c)
	}
}
