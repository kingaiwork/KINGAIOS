package deviceidentity

import (
	"encoding/json"
	"testing"
)

func TestParseIdentity(t *testing.T) {
	want := Identity{
		Schema: SchemaVersion, DeviceID: "edge-00000001", BoardID: "generic-uefi-amd64",
		Class: "gateway", Fleet: "lab", UpdateChannel: "beta", Provisioning: "manual",
		Attestation: "tpm2", Labels: map[string]string{"site": "la"},
	}
	raw, err := json.Marshal(want)
	if err != nil { t.Fatal(err) }
	identity, err := Parse(raw)
	if err != nil { t.Fatal(err) }
	if identity.DeviceID != want.DeviceID || identity.BoardID != want.BoardID || identity.Attestation != want.Attestation {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestIdentityRejectsUnknownField(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"schema": SchemaVersion, "device_id": "edge-00000001", "board_id": "board",
		"class": "gateway", "update_channel": "stable", "provisioning": "manual",
		"attestation": "none", "unknown": true,
	})
	if err != nil { t.Fatal(err) }
	if _, err := Parse(raw); err == nil { t.Fatal("unknown field must be rejected") }
}

func TestIdentityRejectsUnsafeValues(t *testing.T) {
	i := Identity{Schema:SchemaVersion, DeviceID:"edge-00000001", BoardID:"board", Class:"robot", UpdateChannel:"stable", Provisioning:"factory", Attestation:"none", Labels:map[string]string{"../bad":"x"}}
	if err := i.Validate(); err == nil { t.Fatal("unsafe label key must be rejected") }
}
