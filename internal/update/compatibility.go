package update

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var updateTargetIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,127}$`)

type TargetContext struct {
	Profile           string
	Arch              string
	Channel           string
	BoardID           string
	DeviceClass       string
	DevicePackIDs     []string
	AttestationMode   string
}

// CheckTargetCompatibility verifies that a signed update was published for the
// exact class of Edge device that is about to consume it. This is deliberately
// separate from artifact signature verification: both checks are required.
func CheckTargetCompatibility(manifest Manifest, target TargetContext) error {
	if manifest.Profile != "" && manifest.Profile != target.Profile {
		return fmt.Errorf("update profile %q does not match device profile %q", manifest.Profile, target.Profile)
	}
	if manifest.Arch != "" && manifest.Arch != target.Arch {
		return fmt.Errorf("update arch %q does not match device arch %q", manifest.Arch, target.Arch)
	}
	if manifest.Channel != "" && target.Channel != "" && manifest.Channel != target.Channel {
		return fmt.Errorf("update channel %q does not match device channel %q", manifest.Channel, target.Channel)
	}
	if len(manifest.BoardIDs) > 0 && !containsUpdateString(manifest.BoardIDs, target.BoardID) {
		return fmt.Errorf("update is not targeted to board %q", target.BoardID)
	}
	if len(manifest.DeviceClasses) > 0 && !containsUpdateString(manifest.DeviceClasses, target.DeviceClass) {
		return fmt.Errorf("update is not targeted to device class %q", target.DeviceClass)
	}
	for _, required := range manifest.RequiredDevicePacks {
		if !containsUpdateString(target.DevicePackIDs, required) {
			return fmt.Errorf("required Device Pack %q is not installed", required)
		}
	}
	if len(manifest.AttestationModes) > 0 && !containsUpdateString(manifest.AttestationModes, target.AttestationMode) {
		return fmt.Errorf("attestation mode %q is not accepted by update", target.AttestationMode)
	}
	return nil
}

func validateTargetConstraints(manifest Manifest) error {
	if err := validateUpdateTargetList("board_ids", manifest.BoardIDs, 64); err != nil { return err }
	if err := validateUpdateTargetList("device_classes", manifest.DeviceClasses, 16); err != nil { return err }
	if err := validateUpdateTargetList("required_device_packs", manifest.RequiredDevicePacks, 64); err != nil { return err }
	if err := validateUpdateTargetList("attestation_modes", manifest.AttestationModes, 8); err != nil { return err }
	for _, class := range manifest.DeviceClasses {
		switch class {
		case "gateway", "robot", "industrial-pc", "embedded", "appliance", "developer":
		default: return fmt.Errorf("unsupported device class %q", class)
		}
	}
	for _, mode := range manifest.AttestationModes {
		switch mode {
		case "none", "tpm2", "secure-element", "tee":
		default: return fmt.Errorf("unsupported attestation mode %q", mode)
		}
	}
	return nil
}

func validateUpdateTargetList(name string, values []string, max int) error {
	if len(values) > max { return fmt.Errorf("too many %s entries", name) }
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > 160 || strings.ContainsAny(value, "\x00\n\r") {
			return fmt.Errorf("invalid %s entry", name)
		}
		if name == "required_device_packs" && !updateTargetIDPattern.MatchString(value) {
			return errors.New("invalid required Device Pack id")
		}
		if _, exists := seen[value]; exists { return fmt.Errorf("duplicate %s entry %q", name, value) }
		seen[value] = struct{}{}
	}
	return nil
}

func containsUpdateString(values []string, value string) bool {
	for _, candidate := range values { if candidate == value { return true } }
	return false
}
