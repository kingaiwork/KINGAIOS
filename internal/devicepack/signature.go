package devicepack

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const maxSignatureBytes = 16 << 10

var keyIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

type DetachedSignature struct {
	Schema    int    `json:"schema"`
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

// VerifyDetachedSignature verifies a Device Pack manifest against a detached
// Ed25519 signature and a root-provisioned public key. The signature covers the
// exact manifest bytes on disk so formatting changes are also detected.
func VerifyDetachedSignature(manifestPath, signaturePath, keyDir string) error {
	for _, path := range []string{manifestPath, signaturePath, keyDir} {
		if err := validateAbsolutePath(path); err != nil {
			return err
		}
	}
	if err := validateTrustedDirectory(keyDir); err != nil {
		return fmt.Errorf("device-pack trust directory: %w", err)
	}
	if err := validateTrustedRegularFile(manifestPath, maxManifestBytes); err != nil {
		return fmt.Errorf("device-pack manifest trust check: %w", err)
	}
	if err := validateTrustedRegularFile(signaturePath, maxSignatureBytes); err != nil {
		return fmt.Errorf("device-pack signature trust check: %w", err)
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	signatureBytes, err := os.ReadFile(signaturePath)
	if err != nil {
		return err
	}
	envelope, err := decodeDetachedSignature(signatureBytes)
	if err != nil {
		return err
	}

	keyPath := filepath.Join(keyDir, envelope.KeyID+".pub")
	if filepath.Dir(keyPath) != filepath.Clean(keyDir) {
		return errors.New("device-pack key path escaped trust directory")
	}
	if err := validateTrustedRegularFile(keyPath, 4096); err != nil {
		return fmt.Errorf("device-pack public key trust check: %w", err)
	}
	encodedKey, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	publicKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encodedKey)))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid device-pack Ed25519 public key")
	}
	return verifyDetachedSignatureBytes(manifestBytes, envelope, ed25519.PublicKey(publicKey))
}

func decodeDetachedSignature(raw []byte) (DetachedSignature, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var envelope DetachedSignature
	if err := dec.Decode(&envelope); err != nil {
		return DetachedSignature{}, fmt.Errorf("decode device-pack signature: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return DetachedSignature{}, errors.New("device-pack signature must contain exactly one JSON object")
	}
	if envelope.Schema != 1 {
		return DetachedSignature{}, fmt.Errorf("unsupported device-pack signature schema %d", envelope.Schema)
	}
	if !keyIDPattern.MatchString(envelope.KeyID) || strings.Contains(envelope.KeyID, "..") {
		return DetachedSignature{}, errors.New("invalid device-pack signature key id")
	}
	return envelope, nil
}

func verifyDetachedSignatureBytes(manifest []byte, envelope DetachedSignature, publicKey ed25519.PublicKey) error {
	if envelope.Schema != 1 || !keyIDPattern.MatchString(envelope.KeyID) || strings.Contains(envelope.KeyID, "..") {
		return errors.New("invalid device-pack signature envelope")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid device-pack Ed25519 public key")
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(envelope.Signature))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("invalid device-pack Ed25519 signature")
	}
	if !ed25519.Verify(publicKey, manifest, signature) {
		return errors.New("device-pack signature verification failed")
	}
	return nil
}

func validateTrustedRegularFile(path string, maxBytes int64) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("trusted file must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > maxBytes {
		return errors.New("trusted file has invalid size")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("trusted file must not be group/world writable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return errors.New("trusted file must be owned by root")
	}
	return nil
}
