package deviceidentity

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
)

const SchemaVersion = 1
const maxIdentityBytes = 64 << 10

var (
	deviceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{7,63}$`)
	labelKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}$`)
)

type Identity struct {
	Schema        int               `json:"schema"`
	DeviceID      string            `json:"device_id"`
	BoardID       string            `json:"board_id"`
	Class         string            `json:"class"`
	Fleet         string            `json:"fleet,omitempty"`
	UpdateChannel string            `json:"update_channel"`
	Provisioning  string            `json:"provisioning"`
	Attestation   string            `json:"attestation"`
	Labels        map[string]string `json:"labels,omitempty"`
}

func LoadTrusted(path string) (Identity, error) {
	info, err := os.Lstat(path)
	if err != nil { return Identity{}, err }
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Identity{}, errors.New("device identity must be a regular non-symlink file")
	}
	if info.Size() <= 0 || info.Size() > maxIdentityBytes {
		return Identity{}, errors.New("device identity has invalid size")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return Identity{}, errors.New("device identity must not be group/world writable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return Identity{}, errors.New("device identity must be owned by root")
	}
	b, err := os.ReadFile(path)
	if err != nil { return Identity{}, err }
	return Parse(b)
}

func Parse(raw []byte) (Identity, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var identity Identity
	if err := dec.Decode(&identity); err != nil {
		return Identity{}, fmt.Errorf("decode device identity: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Identity{}, errors.New("device identity must contain exactly one JSON object")
	}
	if err := identity.Validate(); err != nil { return Identity{}, err }
	return identity, nil
}

func (i Identity) Validate() error {
	if i.Schema != SchemaVersion { return fmt.Errorf("unsupported device identity schema %d", i.Schema) }
	if !deviceIDPattern.MatchString(i.DeviceID) { return errors.New("invalid device_id") }
	if strings.TrimSpace(i.BoardID) == "" || len(i.BoardID) > 160 || containsControl(i.BoardID) {
		return errors.New("invalid board_id")
	}
	switch i.Class {
	case "gateway", "robot", "industrial-pc", "embedded", "appliance", "developer":
	default:
		return errors.New("invalid device class")
	}
	if len(i.Fleet) > 128 || containsControl(i.Fleet) { return errors.New("invalid fleet") }
	switch i.UpdateChannel {
	case "stable", "beta", "dev", "pinned":
	default:
		return errors.New("invalid update_channel")
	}
	switch i.Provisioning {
	case "factory", "manual", "fleet":
	default:
		return errors.New("invalid provisioning mode")
	}
	switch i.Attestation {
	case "none", "tpm2", "secure-element", "tee":
	default:
		return errors.New("invalid attestation mode")
	}
	if len(i.Labels) > 32 { return errors.New("too many device labels") }
	for key, value := range i.Labels {
		if !labelKeyPattern.MatchString(key) || len(value) > 160 || containsControl(value) {
			return errors.New("invalid device label")
		}
	}
	return nil
}

func containsControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f { return true }
	}
	return false
}
