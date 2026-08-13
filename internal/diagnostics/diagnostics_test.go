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
	if r.Status != "degraded" || r.Score != 92 {
		t.Fatalf("pending update should be visible and degraded but non-critical: status=%s score=%d", r.Status, r.Score)
	}
	found := false
	for _, c := range r.Checks { if c.ID == "ab-state" && c.Status == "warn" { found = true } }
	if !found { t.Fatal("missing A/B pending warning") }
}

func TestTamperedTUFRootMakesReportCritical(t *testing.T) {
	root := healthyRoot(t)
	path := rooted(root, "/etc/kingai/update/root.json")
	b, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil { t.Fatal(err) }
	signed := doc["signed"].(map[string]any)
	signed["version"] = float64(99)
	tampered, _ := json.Marshal(doc)
	if err := os.WriteFile(path, tampered, 0o644); err != nil { t.Fatal(err) }
	r := Run(Options{Root: root})
	if r.Status != "critical" { t.Fatalf("tampered trust root must be critical: %+v", r.Checks) }
	found := false
	for _, c := range r.Checks { if c.ID == "tuf-root" && c.Status == "fail" && c.Severity == "critical" { found = true } }
	if !found { t.Fatal("tampered TUF root was not identified as a critical failure") }
}

func TestWritableTUFRootMakesReportCritical(t *testing.T) {
	root := healthyRoot(t)
	path := rooted(root, "/etc/kingai/update/root.json")
	if err := os.Chmod(path, 0o666); err != nil { t.Fatal(err) }
	r := Run(Options{Root: root})
	if r.Status != "critical" { t.Fatalf("writable trust root must be critical: %+v", r.Checks) }
}

func TestExpiredSignedTUFRootIsDegradedWithRotationAdvice(t *testing.T) {
	root := healthyRoot(t)
	now := time.Now().UTC()
	writeTestFile(t, root, "/etc/kingai/update/root.json", signedRoot(t, now.Add(-time.Hour)))
	r := Run(Options{Root: root, Now: func() time.Time { return now }})
	if r.Status != "degraded" { t.Fatalf("expired but signature-valid root should be degraded, got %s", r.Status) }
	found := false
	for _, c := range r.Checks {
		if c.ID == "tuf-root" && c.Status == "warn" && c.Recommendation != "" { found = true }
	}
	if !found { t.Fatal("expired TUF root warning/rotation guidance missing") }
}
