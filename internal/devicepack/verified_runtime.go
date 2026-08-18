package devicepack

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// LoadVerifiedRuntime is the production Device Pack loader. In addition to the
// manifest/schema checks performed by LoadRuntime it requires a detached
// Ed25519 signature and verifies every declared installed artifact before any
// device capability becomes executable.
func LoadVerifiedRuntime(manifestDir, artifactRoot, keyDir, socketRoot, boardID string, timeout time.Duration) (*Runtime, error) {
	for label, path := range map[string]string{
		"device-pack directory": manifestDir,
		"device-pack artifact root": artifactRoot,
		"device-pack key directory": keyDir,
		"device handler socket root": socketRoot,
	} {
		if err := validateAbsolutePath(path); err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
	}
	if err := validateTrustedDirectory(socketRoot); err != nil {
		return nil, fmt.Errorf("device handler socket root: %w", err)
	}
	if err := validateTrustedDirectory(keyDir); err != nil {
		return nil, fmt.Errorf("device-pack key directory: %w", err)
	}
	if err := validateTrustedDirectory(artifactRoot); err != nil {
		return nil, fmt.Errorf("device-pack artifact root: %w", err)
	}

	entries, err := os.ReadDir(manifestDir)
	if errors.Is(err, os.ErrNotExist) {
		return NewRuntime(nil, socketRoot, boardID, timeout)
	}
	if err != nil {
		return nil, err
	}
	if err := validateTrustedDirectory(manifestDir); err != nil {
		return nil, fmt.Errorf("device-pack directory: %w", err)
	}

	manifests := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		manifestPath := filepath.Join(manifestDir, entry.Name())
		if err := validateTrustedManifestFile(manifestPath); err != nil {
			return nil, err
		}
		signaturePath := manifestPath + ".sig"
		if err := VerifyDetachedSignature(manifestPath, signaturePath, keyDir); err != nil {
			return nil, fmt.Errorf("verify %s: %w", entry.Name(), err)
		}
		manifest, err := Load(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		if err := VerifyInstalledArtifacts(manifest, artifactRoot); err != nil {
			return nil, fmt.Errorf("verify artifacts for %s: %w", manifest.ID, err)
		}
		manifests = append(manifests, manifest)
	}
	return NewRuntime(manifests, socketRoot, boardID, timeout)
}

func VerifyInstalledArtifacts(manifest Manifest, artifactRoot string) error {
	if err := validateAbsolutePath(artifactRoot); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	packRoot := filepath.Join(artifactRoot, manifest.ID)
	if filepath.Dir(packRoot) != filepath.Clean(artifactRoot) {
		return errors.New("device-pack artifact path escaped artifact root")
	}
	if err := validateTrustedDirectory(packRoot); err != nil {
		return fmt.Errorf("artifact directory: %w", err)
	}

	artifacts := append([]Artifact(nil), manifest.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	for _, artifact := range artifacts {
		if strings.ContainsAny(artifact.Name, "/\\") || artifact.Name == "." || artifact.Name == ".." {
			return fmt.Errorf("artifact %q is not a safe basename", artifact.Name)
		}
		path := filepath.Join(packRoot, artifact.Name)
		if filepath.Dir(path) != packRoot {
			return fmt.Errorf("artifact %q escaped pack directory", artifact.Name)
		}
		if err := verifyArtifactFile(path, artifact); err != nil {
			return err
		}
	}
	return nil
}

func verifyArtifactFile(path string, artifact Artifact) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("artifact %q: %w", artifact.Name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("artifact %q must be a regular non-symlink file", artifact.Name)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("artifact %q must not be group/world writable", artifact.Name)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return fmt.Errorf("artifact %q must be owned by root", artifact.Name)
	}
	if info.Size() != artifact.SizeBytes {
		return fmt.Errorf("artifact %q size mismatch", artifact.Name)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != artifact.SHA256 {
		return fmt.Errorf("artifact %q sha256 mismatch", artifact.Name)
	}
	return nil
}
