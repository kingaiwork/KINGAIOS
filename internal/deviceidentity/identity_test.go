package deviceidentity

import "testing"

func TestParseIdentity(t *testing.T) {
	raw := []byte(`{"schema":1,"device_id":"edge-00000001","board_id":"generic-uefi-amd64","class":"gateway","fleet":"lab","update_channel":"beta","provisioning":"manual","attestation":"tpm2","labels":{"site":"la"}}`)
	identity, err := Parse(raw)
	if err != nil { t.Fatal(err) }
	if identity.DeviceID != "edge-00000001" || identity.BoardID != "generic-uefi-amd64" { t.Fatalf("unexpected identity: %+v", identity) }
}

func TestIdentityRejectsUnknownField(t *testing.T) {
	_, err := Parse([]byte(`{"schema":1,"device_id":"edge-00000001","board_id":"board","class":"gateway","update_channel":"stable","provisioning":"manual","attestation":"none","unknown":true}`))
	if err == nil { t.Fatal("unknown field must be rejected") }
}

func TestIdentityRejectsUnsafeValues(t *testing.T) {
	i := Identity{Schema:1, DeviceID:"edge-00000001", BoardID:"board", Class:"robot", UpdateChannel:"stable", Provisioning:"factory", Attestation:"none", Labels:map[string]string{"../bad":"x"}}
	if err := i.Validate(); err == nil { t.Fatal("unsafe label key must be rejected") }
}
