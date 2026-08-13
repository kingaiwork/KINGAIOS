package tufclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata"
	"github.com/theupdateframework/go-tuf/v2/metadata/config"
	"github.com/theupdateframework/go-tuf/v2/metadata/updater"
)

type Config struct {
	MetadataURL     string
	TargetsURL      string
	TrustedRootPath string
	StateDir        string
}

type RootInfo struct {
	Version   int64     `json:"version"`
	Expires   time.Time `json:"expires"`
	Threshold int       `json:"threshold"`
	KeyCount  int       `json:"key_count"`
}

// ValidateTrustedRootIntegrity parses pinned root metadata and verifies that it
// is actually authorized by its own root role threshold. Expiration is returned
// to the caller instead of being rejected here: TUF intentionally allows an
// expired initial root to bootstrap a verified root rotation before freshness is
// enforced against the final root during timestamp processing.
func ValidateTrustedRootIntegrity(data []byte) (RootInfo, error) {
	if len(data) < 128 {
		return RootInfo{}, errors.New("trusted TUF root is unexpectedly small")
	}
	root, err := metadata.Root().FromBytes(data)
	if err != nil {
		return RootInfo{}, fmt.Errorf("parse trusted TUF root: %w", err)
	}
	if root.Signed.Type != metadata.ROOT {
		return RootInfo{}, fmt.Errorf("trusted metadata type is %q, expected root", root.Signed.Type)
	}
	role, ok := root.Signed.Roles[metadata.ROOT]
	if !ok || role == nil {
		return RootInfo{}, errors.New("trusted TUF root does not define the root role")
	}
	if role.Threshold < 1 {
		return RootInfo{}, errors.New("trusted TUF root has an invalid signing threshold")
	}
	if err := root.VerifyDelegate(metadata.ROOT, root); err != nil {
		return RootInfo{}, fmt.Errorf("verify trusted TUF root self-signature threshold: %w", err)
	}
	return RootInfo{
		Version:   root.Signed.Version,
		Expires:   root.Signed.Expires.UTC(),
		Threshold: role.Threshold,
		KeyCount:  len(root.Signed.Keys),
	}, nil
}

// Fetch verifies the complete TUF metadata chain from a trusted root that must
// already be installed out-of-band with KINGAI OS. This function deliberately
// does not implement TOFU: a remote server can never choose the first root of
// trust for an installed system.
func Fetch(cfg Config, target string) (string, error) {
	return fetch(cfg, target, nil)
}

// fetch accepts a custom HTTP client only as an internal test seam so tests can
// exercise a real TLS server with an ephemeral CA. Production callers always
// enter through Fetch and therefore use go-tuf's default verified HTTPS client.
func fetch(cfg Config, target string, testHTTPClient *http.Client) (string, error) {
	if target == "" { return "", errors.New("TUF target name is required") }
	if err := validateHTTPS(cfg.MetadataURL, "metadata URL"); err != nil { return "", err }
	if err := validateHTTPS(cfg.TargetsURL, "targets URL"); err != nil { return "", err }
	if cfg.TrustedRootPath == "" || cfg.StateDir == "" { return "", errors.New("trusted root path and TUF state directory are required") }
	rootBytes, err := os.ReadFile(cfg.TrustedRootPath)
	if err != nil { return "", fmt.Errorf("read trusted TUF root: %w", err) }
	if _, err := ValidateTrustedRootIntegrity(rootBytes); err != nil {
		return "", err
	}
	metadataDir := filepath.Join(cfg.StateDir, "metadata")
	targetsDir := filepath.Join(cfg.StateDir, "targets")
	if err := os.MkdirAll(metadataDir, 0o700); err != nil { return "", err }
	if err := os.MkdirAll(targetsDir, 0o700); err != nil { return "", err }

	ucfg, err := config.New(cfg.MetadataURL, rootBytes)
	if err != nil { return "", fmt.Errorf("create TUF configuration: %w", err) }
	ucfg.LocalMetadataDir = metadataDir
	ucfg.LocalTargetsDir = targetsDir
	ucfg.RemoteTargetsURL = cfg.TargetsURL
	ucfg.PrefixTargetsWithHash = true
	if testHTTPClient != nil {
		if err := ucfg.SetDefaultFetcherHTTPClient(testHTTPClient); err != nil { return "", fmt.Errorf("configure TUF test TLS client: %w", err) }
	}
	up, err := updater.New(ucfg)
	if err != nil { return "", fmt.Errorf("create TUF updater: %w", err) }
	if err := up.Refresh(); err != nil { return "", fmt.Errorf("refresh TUF metadata: %w", err) }
	info, err := up.GetTargetInfo(target)
	if err != nil { return "", fmt.Errorf("trusted TUF target %q not found: %w", target, err) }
	if path, _, err := up.FindCachedTarget(info, ""); err != nil {
		return "", fmt.Errorf("validate cached TUF target: %w", err)
	} else if path != "" { return path, nil }
	path, _, err := up.DownloadTarget(info, "", "")
	if err != nil { return "", fmt.Errorf("download trusted TUF target: %w", err) }
	return path, nil
}

func validateHTTPS(raw, label string) error {
	u, err := url.Parse(raw)
	if err != nil { return fmt.Errorf("invalid %s: %w", label, err) }
	if u.Scheme != "https" || u.Host == "" { return fmt.Errorf("%s must use https with an explicit host", label) }
	if u.User != nil || u.Fragment != "" { return fmt.Errorf("%s must not contain credentials or fragments", label) }
	return nil
}
