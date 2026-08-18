package devicepack

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestDetachedSignatureBytes(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { t.Fatal(err) }
	manifest := []byte(`{"schema":2,"id":"kingai.test"}`)
	sig := ed25519.Sign(priv, manifest)
	envelope := DetachedSignature{Schema: 1, KeyID: "kingai-release", Signature: base64.StdEncoding.EncodeToString(sig)}
	if err := verifyDetachedSignatureBytes(manifest, envelope, pub); err != nil { t.Fatal(err) }
	if err := verifyDetachedSignatureBytes(append([]byte(nil), append(manifest, ' ')...), envelope, pub); err == nil {
		t.Fatal("tampered manifest must fail signature verification")
	}
}

func TestDetachedSignatureRejectsBadKeyID(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil { t.Fatal(err) }
	manifest := []byte(`{"schema":2}`)
	envelope := DetachedSignature{Schema: 1, KeyID: "../escape", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, manifest))}
	if err := verifyDetachedSignatureBytes(manifest, envelope, pub); err == nil {
		t.Fatal("unsafe key id must be rejected")
	}
}
