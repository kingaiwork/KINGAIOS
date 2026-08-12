package update

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type Manifest struct {
	Schema    int    `json:"schema"`
	Product   string `json:"product"`
	Version   string `json:"version"`
	Channel   string `json:"channel"`
	Profile   string `json:"profile"`
	Arch      string `json:"arch"`
	Artifact  string `json:"artifact"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type Envelope struct {
	KeyID     string   `json:"key_id"`
	Manifest  Manifest `json:"manifest"`
	Signature string   `json:"signature"`
}

func VerifyEnvelope(e Envelope, pub ed25519.PublicKey) error {
	if e.Manifest.Schema != 1 { return fmt.Errorf("unsupported manifest schema: %d", e.Manifest.Schema) }
	if e.Manifest.Product != "KINGAI OS" { return errors.New("unexpected update product") }
	if len(pub) != ed25519.PublicKeySize { return errors.New("invalid update public key") }
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil { return fmt.Errorf("decode signature: %w", err) }
	payload, err := json.Marshal(e.Manifest)
	if err != nil { return err }
	if !ed25519.Verify(pub, payload, sig) { return errors.New("invalid manifest signature") }
	return nil
}

func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil { return nil, err }
	raw, err := base64.StdEncoding.DecodeString(string(bytesTrimSpace(b)))
	if err != nil { return nil, fmt.Errorf("decode public key: %w", err) }
	if len(raw) != ed25519.PublicKeySize { return nil, errors.New("invalid update public key size") }
	return ed25519.PublicKey(raw), nil
}

func VerifyArtifact(path string, m Manifest) error {
	f, err := os.Open(path)
	if err != nil { return err }
	defer f.Close()
	st, err := f.Stat()
	if err != nil { return err }
	if st.Size() != m.SizeBytes { return fmt.Errorf("artifact size mismatch: got %d want %d", st.Size(), m.SizeBytes) }
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil { return err }
	got := hex.EncodeToString(h.Sum(nil))
	want, err := hex.DecodeString(m.SHA256)
	if err != nil || len(want) != sha256.Size { return errors.New("invalid manifest sha256") }
	gotRaw, _ := hex.DecodeString(got)
	if subtle.ConstantTimeCompare(gotRaw, want) != 1 { return errors.New("artifact sha256 mismatch") }
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && (b[start]==' ' || b[start]=='\n' || b[start]=='\r' || b[start]=='\t') { start++ }
	for end > start && (b[end-1]==' ' || b[end-1]=='\n' || b[end-1]=='\r' || b[end-1]=='\t') { end-- }
	return b[start:end]
}
