package devicepack

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,127}$`)

type Boot struct {
	Method            string `json:"method"`
	FirmwareRequired  bool   `json:"firmware_required,omitempty"`
	DTBRequired       bool   `json:"dtb_required,omitempty"`
	SecureBootCapable bool   `json:"secure_boot_capable,omitempty"`
}

type Artifact struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	License   string `json:"license,omitempty"`
	Source    string `json:"source,omitempty"`
}

type Security struct {
	SignedManifest         bool   `json:"signed_manifest"`
	RedistributionReviewed bool   `json:"redistribution_reviewed"`
	MinimumFirmwareVersion string `json:"minimum_firmware_version,omitempty"`
	Notes                  string `json:"notes,omitempty"`
}

type Manifest struct {
	Schema    int        `json:"schema"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Arch      string     `json:"arch"`
	Vendor    string     `json:"vendor"`
	BoardIDs  []string   `json:"board_ids,omitempty"`
	Boot      Boot       `json:"boot"`
	Artifacts []Artifact `json:"artifacts"`
	Security  Security   `json:"security"`
}

func Load(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decode device pack: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m Manifest) Validate() error {
	if m.Schema != 1 {
		return fmt.Errorf("unsupported device-pack schema %d", m.Schema)
	}
	if !idPattern.MatchString(m.ID) {
		return errors.New("invalid device-pack id")
	}
	if strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" || strings.TrimSpace(m.Vendor) == "" {
		return errors.New("name, version and vendor are required")
	}
	if m.Arch != "amd64" && m.Arch != "arm64" {
		return errors.New("arch must be amd64 or arm64")
	}
	switch m.Boot.Method {
	case "uefi", "uboot", "vendor":
	default:
		return errors.New("boot method must be uefi, uboot or vendor")
	}
	if len(m.Artifacts) == 0 {
		return errors.New("at least one device-pack artifact is required")
	}
	seen := map[string]struct{}{}
	for _, a := range m.Artifacts {
		if strings.TrimSpace(a.Name) == "" {
			return errors.New("artifact name is required")
		}
		if _, ok := seen[a.Name]; ok {
			return fmt.Errorf("duplicate artifact %q", a.Name)
		}
		seen[a.Name] = struct{}{}
		switch a.Kind {
		case "kernel", "initrd", "dtb", "firmware", "bootloader", "driver", "config":
		default:
			return fmt.Errorf("unsupported artifact kind %q", a.Kind)
		}
		hash, err := hex.DecodeString(a.SHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(a.SHA256) != a.SHA256 {
			return fmt.Errorf("artifact %q has invalid sha256", a.Name)
		}
		if a.SizeBytes < 0 {
			return fmt.Errorf("artifact %q has invalid size", a.Name)
		}
		// Firmware and vendor boot components are high-risk redistribution items.
		if (a.Kind == "firmware" || a.Kind == "bootloader") && strings.TrimSpace(a.License) == "" {
			return fmt.Errorf("artifact %q requires an explicit license field", a.Name)
		}
	}
	if !m.Security.SignedManifest {
		return errors.New("device-pack manifest must be signed before release")
	}
	if !m.Security.RedistributionReviewed {
		return errors.New("device-pack redistribution review is required")
	}
	return nil
}
