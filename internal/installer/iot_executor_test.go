package installer

import (
	"runtime"
	"testing"
)

func TestExecuteIoTRejectsWrongProfileBeforeWrite(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("IoT destructive execution is intentionally amd64-only")
	}
	_, err := ExecuteIoT(ExecuteOptions{Profile: "server", Target: "/dev/never"})
	if err == nil {
		t.Fatal("expected non-IoT profile to be rejected")
	}
}
