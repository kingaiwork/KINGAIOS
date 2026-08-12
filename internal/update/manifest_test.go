package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSignedManifestAndArtifact(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader); if err != nil { t.Fatal(err) }
	data := []byte("kingai-update-test")
	sum := sha256.Sum256(data)
	m := Manifest{Schema:1,Product:"KINGAI OS",Version:"0.1.1",Channel:"dev",Profile:"server",Arch:"amd64",Artifact:"test.img",URL:"https://os.kingai.work/updates/test.img",SHA256:hex.EncodeToString(sum[:]),SizeBytes:int64(len(data))}
	payload, _ := json.Marshal(m)
	e := Envelope{KeyID:"test",Manifest:m,Signature:base64.StdEncoding.EncodeToString(ed25519.Sign(priv,payload))}
	if err := VerifyEnvelope(e,pub); err != nil { t.Fatal(err) }
	p := filepath.Join(t.TempDir(),"test.img"); if err := os.WriteFile(p,data,0o600); err != nil { t.Fatal(err) }
	if err := VerifyArtifact(p,m); err != nil { t.Fatal(err) }
}

func TestTamperedManifestRejected(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	m := Manifest{Schema:1,Product:"KINGAI OS",Version:"0.1.1"}
	payload,_ := json.Marshal(m)
	e := Envelope{Manifest:m,Signature:base64.StdEncoding.EncodeToString(ed25519.Sign(priv,payload))}
	e.Manifest.Version="9.9.9"
	if err := VerifyEnvelope(e,pub); err == nil { t.Fatal("tampered manifest must be rejected") }
}
