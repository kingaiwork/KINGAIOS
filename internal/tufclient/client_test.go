package tufclient

import (
	"crypto"
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
)

type testRepo struct {
	root      []byte
	timestamp []byte
	snapshot  []byte
	targets   []byte
	target    []byte
	targetKey string
}

func buildTestRepo(t *testing.T) testRepo {
	t.Helper()
	expires := time.Now().UTC().Add(24 * time.Hour)
	root := metadata.Root(expires.Add(30 * 24 * time.Hour))
	targets := metadata.Targets(expires)
	snapshot := metadata.Snapshot(expires)
	timestamp := metadata.Timestamp(expires)

	keys := map[string]ed25519.PrivateKey{}
	for _, role := range []string{metadata.ROOT, metadata.TARGETS, metadata.SNAPSHOT, metadata.TIMESTAMP} {
		_, private, err := ed25519.GenerateKey(nil)
		if err != nil { t.Fatal(err) }
		keys[role] = private
		key, err := metadata.KeyFromPublicKey(private.Public())
		if err != nil { t.Fatal(err) }
		if err := root.Signed.AddKey(key, role); err != nil { t.Fatal(err) }
	}

	targetBody := []byte("KINGAI-TUF-VERIFIED-TARGET\n")
	tmpTarget := filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(tmpTarget, targetBody, 0o600); err != nil { t.Fatal(err) }
	info, err := metadata.TargetFile().FromFile(tmpTarget, "sha256")
	if err != nil { t.Fatal(err) }
	targetName := "payload.bin"
	targets.Signed.Targets[targetName] = info

	sign := func(role string, m interface{ Sign(signature.Signer) (*metadata.Signature, error) }) {
		signer, err := signature.LoadSigner(keys[role], crypto.Hash(0))
		if err != nil { t.Fatal(err) }
		if _, err := m.Sign(signer); err != nil { t.Fatal(err) }
	}
	sign(metadata.ROOT, root)
	sign(metadata.TARGETS, targets)
	sign(metadata.SNAPSHOT, snapshot)
	sign(metadata.TIMESTAMP, timestamp)

	rootBytes, err := root.ToBytes(false); if err != nil { t.Fatal(err) }
	targetsBytes, err := targets.ToBytes(false); if err != nil { t.Fatal(err) }
	snapshotBytes, err := snapshot.ToBytes(false); if err != nil { t.Fatal(err) }
	timestampBytes, err := timestamp.ToBytes(false); if err != nil { t.Fatal(err) }
	hash := info.Hashes["sha256"]
	if len(hash) == 0 { t.Fatal("target sha256 missing") }
	return testRepo{
		root: rootBytes,
		timestamp: timestampBytes,
		snapshot: snapshotBytes,
		targets: targetsBytes,
		target: targetBody,
		targetKey: hex.EncodeToString(hash) + "." + targetName,
	}
}

func serveRepo(t *testing.T, repo testRepo, tamperMetadata, tamperTarget bool) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		switch r.URL.Path {
		case "/metadata/2.root.json":
			http.NotFound(w, r); return
		case "/metadata/timestamp.json":
			body = repo.timestamp
		case "/metadata/1.snapshot.json":
			body = repo.snapshot
		case "/metadata/1.targets.json":
			body = append([]byte(nil), repo.targets...)
			if tamperMetadata && len(body) > 16 { body[len(body)/2] ^= 1 }
		case "/targets/" + repo.targetKey:
			body = append([]byte(nil), repo.target...)
			if tamperTarget { body = append(body, []byte("TAMPERED")...) }
		default:
			http.NotFound(w, r); return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(body)
	})
	return httptest.NewTLSServer(h)
}

func writeTrustedRoot(t *testing.T, root []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "root.json")
	if err := os.WriteFile(path, root, 0o600); err != nil { t.Fatal(err) }
	return path
}

func TestFetchPinnedRootHTTPSAndRejectTampering(t *testing.T) {
	repo := buildTestRepo(t)
	rootPath := writeTrustedRoot(t, repo.root)

	t.Run("valid chain downloads verified target", func(t *testing.T) {
		srv := serveRepo(t, repo, false, false); defer srv.Close()
		path, err := fetch(Config{MetadataURL: srv.URL + "/metadata", TargetsURL: srv.URL + "/targets", TrustedRootPath: rootPath, StateDir: t.TempDir()}, "payload.bin", srv.Client())
		if err != nil { t.Fatalf("Fetch failed: %v", err) }
		got, err := os.ReadFile(path); if err != nil { t.Fatal(err) }
		if string(got) != string(repo.target) { t.Fatalf("downloaded target mismatch: %q", got) }
	})

	t.Run("tampered target is rejected", func(t *testing.T) {
		srv := serveRepo(t, repo, false, true); defer srv.Close()
		if _, err := fetch(Config{MetadataURL: srv.URL + "/metadata", TargetsURL: srv.URL + "/targets", TrustedRootPath: rootPath, StateDir: t.TempDir()}, "payload.bin", srv.Client()); err == nil {
			t.Fatal("tampered TUF target was accepted")
		}
	})

	t.Run("tampered signed metadata is rejected", func(t *testing.T) {
		srv := serveRepo(t, repo, true, false); defer srv.Close()
		if _, err := fetch(Config{MetadataURL: srv.URL + "/metadata", TargetsURL: srv.URL + "/targets", TrustedRootPath: rootPath, StateDir: t.TempDir()}, "payload.bin", srv.Client()); err == nil {
			t.Fatal("tampered TUF metadata was accepted")
		}
	})
}

func TestFetchRefusesNonHTTPS(t *testing.T) {
	root := writeTrustedRoot(t, make([]byte, 256))
	_, err := Fetch(Config{MetadataURL: "http://example.test/metadata", TargetsURL: "https://example.test/targets", TrustedRootPath: root, StateDir: t.TempDir()}, "payload.bin")
	if err == nil { t.Fatal("non-HTTPS metadata URL was accepted") }
}
