package devicepack

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

func runtimeManifest() Manifest {
	return Manifest{
		Schema:  SchemaVersion,
		ID:      "kingai.test-device",
		Name:    "KINGAI Test Device",
		Version: "0.1.0",
		Arch:    runtime.GOARCH,
		Vendor:  "KINGAI",
		Boot:    Boot{Method: "uefi"},
		Artifacts: []Artifact{{
			Name:      "device.json",
			Kind:      "config",
			SHA256:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			SizeBytes: 1,
		}},
		Capabilities: []Capability{{
			ID:               "device.gpio.read",
			Handler:          "gpio-read",
			Operation:        "read",
			Risk:             "L1",
			ApprovalRequired: false,
			Resources:        []string{"gpio:17"},
		}},
		Security: Security{SignedManifest: true, RedistributionReviewed: true},
	}
}

func TestRuntimeExecutesDeclaredResource(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(root, "gpio-read.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil { t.Fatal(err) }
	defer listener.Close()
	if err := os.Chmod(socket, 0o600); err != nil { t.Fatal(err) }

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/execute" { http.Error(w, "not found", http.StatusNotFound); return }
		var req executor.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "bad request", http.StatusBadRequest); return }
		if req.Capability != "device.gpio.read" || req.Target != "gpio:17" { http.Error(w, "unexpected request", http.StatusBadRequest); return }
		payload, _ := json.Marshal(map[string]int{"value": 1})
		_ = json.NewEncoder(w).Encode(executor.Result{OK: true, Data: payload})
	})}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	rt, err := NewRuntime([]Manifest{runtimeManifest()}, root, "", time.Second)
	if err != nil { t.Fatal(err) }
	result, err := rt.Execute(context.Background(), executor.Request{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:17"})
	if err != nil { t.Fatal(err) }
	if !result.OK || result.ExecutionID == "" || result.Capability != "device.gpio.read" { t.Fatalf("unexpected result: %+v", result) }
}

func TestRuntimeRejectsUndeclaredResource(t *testing.T) {
	rt, err := NewRuntime([]Manifest{runtimeManifest()}, t.TempDir(), "", time.Second)
	if err != nil { t.Fatal(err) }
	_, err = rt.Execute(context.Background(), executor.Request{Agent:"system-ops",Capability:"device.gpio.read",Target:"gpio:18"})
	if err == nil { t.Fatal("undeclared device resource must be rejected") }
}

func TestRuntimeRejectsDuplicateCapability(t *testing.T) {
	first := runtimeManifest(); second := runtimeManifest(); second.ID = "kingai.second-device"
	if _, err := NewRuntime([]Manifest{first, second}, t.TempDir(), "", time.Second); err == nil { t.Fatal("duplicate device capability must be rejected") }
}

func TestRuntimeRequiresBoardMatch(t *testing.T) {
	manifest := runtimeManifest(); manifest.BoardIDs = []string{"board-a"}
	if _, err := NewRuntime([]Manifest{manifest}, t.TempDir(), "board-b", time.Second); err == nil { t.Fatal("device pack must not load on a different board id") }
	if _, err := NewRuntime([]Manifest{manifest}, t.TempDir(), "board-a", time.Second); err != nil { t.Fatalf("matching board id should load: %v", err) }
}
