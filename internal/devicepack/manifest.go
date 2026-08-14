package devicepack

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const SchemaVersion = 2

var (
	idPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,127}$`)
	capabilityPattern = regexp.MustCompile(`^device\.[a-z0-9][a-z0-9._-]{1,126}$`)
	handlerPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	resourcePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@+\-]{0,255}$`)
)

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

type Capability struct {
	ID               string   `json:"id"`
	Handler          string   `json:"handler"`
	Operation        string   `json:"operation"`
	Risk             string   `json:"risk"`
	ApprovalRequired bool     `json:"approval_required"`
	Resources        []string `json:"resources"`
	Description      string   `json:"description,omitempty"`
}

type Security struct {
	SignedManifest         bool   `json:"signed_manifest"`
	RedistributionReviewed bool   `json:"redistribution_reviewed"`
	MinimumFirmwareVersion string `json:"minimum_firmware_version,omitempty"`
	Notes                  string `json:"notes,omitempty"`
}

type Manifest struct {
	Schema       int          `json:"schema"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Arch         string       `json:"arch"`
	Vendor       string       `json:"vendor"`
	BoardIDs     []string     `json:"board_ids,omitempty"`
	Boot         Boot         `json:"boot"`
	Artifacts    []Artifact   `json:"artifacts"`
	Capabilities []Capability `json:"capabilities"`
	Security     Security     `json:"security"`
}

func Load(path string) (Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil { return Manifest{}, err }
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil { return Manifest{}, fmt.Errorf("decode device pack: %w", err) }
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) { return Manifest{}, errors.New("device pack must contain exactly one JSON object") }
	if err := m.Validate(); err != nil { return Manifest{}, err }
	return m, nil
}

func (m Manifest) Validate() error {
	if m.Schema != SchemaVersion { return fmt.Errorf("unsupported device-pack schema %d", m.Schema) }
	if !idPattern.MatchString(m.ID) { return errors.New("invalid device-pack id") }
	if strings.TrimSpace(m.Name) == "" || len(m.Name) > 160 || strings.TrimSpace(m.Version) == "" || len(m.Version) > 64 || strings.TrimSpace(m.Vendor) == "" || len(m.Vendor) > 120 {
		return errors.New("invalid name, version or vendor")
	}
	if m.Arch != "amd64" && m.Arch != "arm64" { return errors.New("arch must be amd64 or arm64") }
	if len(m.BoardIDs) > 64 || duplicateStrings(m.BoardIDs) { return errors.New("invalid board id set") }
	for _, boardID := range m.BoardIDs {
		if strings.TrimSpace(boardID) == "" || len(boardID) > 160 || containsControl(boardID) { return errors.New("invalid board id") }
	}
	switch m.Boot.Method { case "uefi", "uboot", "vendor": default: return errors.New("boot method must be uefi, uboot or vendor") }
	if len(m.Artifacts) == 0 { return errors.New("at least one device-pack artifact is required") }
	seenArtifacts := map[string]struct{}{}
	for _, a := range m.Artifacts {
		if strings.TrimSpace(a.Name) == "" || len(a.Name) > 255 || containsControl(a.Name) || strings.ContainsAny(a.Name, "/\\") || a.Name == "." || a.Name == ".." {
			return errors.New("artifact name is invalid")
		}
		if _, ok := seenArtifacts[a.Name]; ok { return fmt.Errorf("duplicate artifact %q", a.Name) }
		seenArtifacts[a.Name] = struct{}{}
		switch a.Kind { case "kernel", "initrd", "dtb", "firmware", "bootloader", "driver", "config": default: return fmt.Errorf("unsupported artifact kind %q", a.Kind) }
		hash, err := hex.DecodeString(a.SHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(a.SHA256) != a.SHA256 { return fmt.Errorf("artifact %q has invalid sha256", a.Name) }
		if a.SizeBytes < 0 { return fmt.Errorf("artifact %q has invalid size", a.Name) }
		if len(a.License) > 160 || len(a.Source) > 2048 || containsUnsafeText(a.License) || containsUnsafeText(a.Source) { return fmt.Errorf("artifact %q has invalid metadata", a.Name) }
		if a.Kind == "firmware" || a.Kind == "bootloader" || a.Kind == "driver" {
			if strings.TrimSpace(a.License) == "" || strings.TrimSpace(a.Source) == "" { return fmt.Errorf("artifact %q requires explicit license and source", a.Name) }
		}
	}
	if len(m.Capabilities) > 128 { return errors.New("too many device capabilities") }
	seenCapabilities := map[string]struct{}{}
	for _, capability := range m.Capabilities {
		if _, ok := seenCapabilities[capability.ID]; ok { return fmt.Errorf("duplicate capability %q", capability.ID) }
		seenCapabilities[capability.ID] = struct{}{}
		if err := validateCapability(capability); err != nil { return fmt.Errorf("capability %q: %w", capability.ID, err) }
	}
	if len(m.Security.MinimumFirmwareVersion) > 128 || len(m.Security.Notes) > 2000 || containsUnsafeText(m.Security.MinimumFirmwareVersion) || containsUnsafeText(m.Security.Notes) {
		return errors.New("invalid security metadata")
	}
	if !m.Security.SignedManifest { return errors.New("device-pack manifest must be signed before release") }
	if !m.Security.RedistributionReviewed { return errors.New("device-pack redistribution review is required") }
	return nil
}

func CapabilityIDs(m Manifest) []string {
	ids := make([]string, 0, len(m.Capabilities))
	for _, capability := range m.Capabilities { ids = append(ids, capability.ID) }
	sort.Strings(ids)
	return ids
}

func validateCapability(c Capability) error {
	if !capabilityPattern.MatchString(c.ID) || strings.Contains(c.ID, "..") || strings.Contains(c.ID, "*") { return errors.New("invalid capability id") }
	if !handlerPattern.MatchString(c.Handler) || strings.Contains(c.Handler, "..") || strings.ContainsAny(c.Handler, "/\\ ;$`*?") { return errors.New("handler must be a logical registered id, not a command or path") }
	switch c.Operation { case "read", "write", "control", "reset", "update": default: return errors.New("invalid operation") }
	rank := riskRank(c.Risk)
	if rank < 0 { return errors.New("risk must be L0 through L6") }
	if len(c.Resources) == 0 || len(c.Resources) > 32 || duplicateStrings(c.Resources) { return errors.New("invalid resource set") }
	for _, resource := range c.Resources {
		if !resourcePattern.MatchString(resource) || strings.ContainsAny(resource, "*?[]{};$`\\\n\r\x00") { return errors.New("resource contains wildcard, shell or control syntax") }
	}
	floor := minimumCapabilityRisk(c)
	if rank < floor { return fmt.Errorf("declared risk %s is below required L%d safety floor", c.Risk, floor) }
	if len(c.Description) > 500 || containsUnsafeText(c.Description) { return errors.New("invalid capability description") }
	if c.Operation != "read" && rank >= 3 && !c.ApprovalRequired { return errors.New("high-risk mutating capability requires approval") }
	if rank >= 5 && !c.ApprovalRequired { return errors.New("critical device capability requires approval") }
	return nil
}

func riskRank(risk string) int {
	if len(risk) != 2 || risk[0] != 'L' || risk[1] < '0' || risk[1] > '6' { return -1 }
	return int(risk[1] - '0')
}

func duplicateStrings(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values { if _, exists := seen[value]; exists { return true }; seen[value] = struct{}{} }
	return false
}

func containsControl(value string) bool {
	for _, r := range value { if r < 0x20 || r == 0x7f { return true } }
	return false
}

func containsUnsafeText(value string) bool {
	for _, r := range value { if r == 0 || r == 0x7f || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') { return true } }
	return false
}
