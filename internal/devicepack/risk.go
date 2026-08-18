package devicepack

import "strings"

// minimumCapabilityRisk prevents a Device Pack from self-declaring a dangerous
// physical operation as low risk. The declared risk may always be higher.
func minimumCapabilityRisk(c Capability) int {
	floor := operationRiskFloor(c.Operation)
	for _, resource := range c.Resources {
		if candidate := resourceRiskFloor(resource); candidate > floor { floor = candidate }
	}
	return floor
}

func operationRiskFloor(operation string) int {
	switch operation {
	case "read": return 0
	case "write", "compute": return 1
	case "control": return 3
	case "reset": return 4
	case "update": return 5
	default: return 6
	}
}

func resourceRiskFloor(resource string) int {
	prefix := resource
	if i := strings.IndexByte(prefix, ':'); i >= 0 { prefix = prefix[:i] }
	prefix = strings.ToLower(prefix)
	switch prefix {
	case "sensor", "telemetry": return 0
	case "gpio", "i2c", "spi", "uart", "gpu", "npu", "accelerator", "camera": return 1
	case "microphone", "location", "network", "storage": return 2
	case "safety": return 3
	case "motor", "actuator", "relay", "power": return 4
	case "firmware", "boot", "flash": return 5
	default: return 0
	}
}
