package devicepack

import "testing"

func TestCapabilityRiskFloors(t *testing.T) {
	tests := []struct{
		name string
		cap Capability
		want int
	}{
		{"telemetry read", Capability{Operation:"read", Resources:[]string{"sensor:temperature"}}, 0},
		{"gpio write", Capability{Operation:"write", Resources:[]string{"gpio:17"}}, 1},
		{"camera read", Capability{Operation:"read", Resources:[]string{"camera:front"}}, 1},
		{"npu compute", Capability{Operation:"compute", Resources:[]string{"npu:0"}}, 1},
		{"motor control", Capability{Operation:"control", Resources:[]string{"motor:left"}}, 4},
		{"power control", Capability{Operation:"control", Resources:[]string{"power:system"}}, 4},
		{"firmware update", Capability{Operation:"update", Resources:[]string{"firmware:board"}}, 5},
	}
	for _, tt := range tests { t.Run(tt.name, func(t *testing.T){ if got:=minimumCapabilityRisk(tt.cap); got!=tt.want { t.Fatalf("got %d want %d",got,tt.want) } }) }
}

func TestUnderstatedPhysicalRiskRejected(t *testing.T) {
	c := Capability{ID:"device.motor.control", Handler:"motor-control", Operation:"control", Risk:"L1", ApprovalRequired:false, Resources:[]string{"motor:left"}}
	if err := validateCapability(c); err == nil { t.Fatal("understated motor risk must be rejected") }
}

func TestHighRiskMutationRequiresApproval(t *testing.T) {
	c := Capability{ID:"device.firmware.update", Handler:"firmware-update", Operation:"update", Risk:"L5", ApprovalRequired:false, Resources:[]string{"firmware:board"}}
	if err := validateCapability(c); err == nil { t.Fatal("firmware update must require approval") }
}

func TestHighRiskComputeCannotBypassApproval(t *testing.T) {
	c := Capability{ID:"device.power.compute", Handler:"power-compute", Operation:"compute", Risk:"L4", ApprovalRequired:false, Resources:[]string{"power:system"}}
	if err := validateCapability(c); err == nil { t.Fatal("compute label must not bypass high-risk resource approval") }
	c = Capability{ID:"device.npu.compute", Handler:"npu-compute", Operation:"compute", Risk:"L1", ApprovalRequired:false, Resources:[]string{"npu:0"}}
	if err := validateCapability(c); err != nil { t.Fatalf("ordinary L1 NPU compute should remain allowed: %v",err) }
}
