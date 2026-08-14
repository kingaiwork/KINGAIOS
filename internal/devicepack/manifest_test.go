package devicepack

import "testing"

func validManifest() Manifest {
	return Manifest{
		Schema: SchemaVersion,
		ID: "kingai.generic-uefi-arm64",
		Name: "KINGAI Generic UEFI ARM64",
		Version: "0.1.0",
		Arch: "arm64",
		Vendor: "KINGAI",
		Boot: Boot{Method: "uefi", SecureBootCapable: true},
		Artifacts: []Artifact{{
			Name: "Image", Kind: "kernel",
			SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			SizeBytes: 1,
			License: "GPL-2.0-only",
		}},
		Security: Security{SignedManifest: true, RedistributionReviewed: true},
	}
}

func TestValidDevicePack(t *testing.T) {
	if err := validManifest().Validate(); err != nil { t.Fatal(err) }
}

func TestUnsignedDevicePackRejected(t *testing.T) {
	m := validManifest(); m.Security.SignedManifest = false
	if err := m.Validate(); err == nil { t.Fatal("unsigned pack must be rejected") }
}

func TestFirmwareWithoutLicenseRejected(t *testing.T) {
	m := validManifest(); m.Artifacts[0].Kind = "firmware"; m.Artifacts[0].License = ""
	if err := m.Validate(); err == nil { t.Fatal("firmware without explicit license must be rejected") }
}

func TestInvalidHashRejected(t *testing.T) {
	m := validManifest(); m.Artifacts[0].SHA256 = "not-a-sha256"
	if err := m.Validate(); err == nil { t.Fatal("invalid hash must be rejected") }
}
