package update

import "testing"

func TestEdgeTargetCompatibility(t *testing.T) {
	m := Manifest{
		Profile:"iot", Arch:"arm64", Channel:"stable",
		BoardIDs:[]string{"raspberry-pi-5"}, DeviceClasses:[]string{"gateway"},
		RequiredDevicePacks:[]string{"kingai.raspberry-pi-5"}, AttestationModes:[]string{"tpm2"},
	}
	ctx := TargetContext{Profile:"iot",Arch:"arm64",Channel:"stable",BoardID:"raspberry-pi-5",DeviceClass:"gateway",DevicePackIDs:[]string{"kingai.raspberry-pi-5"},AttestationMode:"tpm2"}
	if err := CheckTargetCompatibility(m,ctx); err != nil { t.Fatal(err) }
	ctx.BoardID="nvidia-jetson-orin-nano"
	if err := CheckTargetCompatibility(m,ctx); err == nil { t.Fatal("wrong board must be rejected") }
}

func TestEdgeTargetRequiresPack(t *testing.T) {
	m := Manifest{RequiredDevicePacks:[]string{"kingai.board-pack"}}
	if err := CheckTargetCompatibility(m,TargetContext{}); err == nil { t.Fatal("missing Device Pack must be rejected") }
}

func TestInvalidTargetConstraintsRejected(t *testing.T) {
	m := Manifest{RequiredDevicePacks:[]string{"../escape"}}
	if err := validateTargetConstraints(m); err == nil { t.Fatal("unsafe Device Pack id must be rejected") }
	m = Manifest{DeviceClasses:[]string{"unknown"}}
	if err := validateTargetConstraints(m); err == nil { t.Fatal("unknown class must be rejected") }
}
