package main

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/deviceidentity"
	"github.com/kingaiwork/KINGAIOS/internal/devicepack"
	"github.com/kingaiwork/KINGAIOS/internal/executor"
	"github.com/kingaiwork/KINGAIOS/internal/policy"
)

// IoT uses an in-process, non-privileged Device Broker that speaks the same
// constrained AF_UNIX execution protocol as the regular execution client. It
// does not execute shell commands and it never opens hardware devices itself;
// exact device.* capabilities are forwarded only to verified Device Pack handlers.
func init() {
	if !getenvBool("KINGAI_DEVICE_RUNTIME_ENABLED", false) { return }

	manifestDir := getenv("KINGAI_DEVICE_PACK_DIR", "/etc/kingai/device-packs")
	artifactRoot := getenv("KINGAI_DEVICE_ARTIFACT_ROOT", "/usr/lib/kingai/device-packs")
	trustDir := getenv("KINGAI_DEVICE_TRUST_DIR", "/etc/kingai/trust/device-pack-keys")
	handlerRoot := getenv("KINGAI_DEVICE_HANDLER_ROOT", "/run/kingai-device")
	identityPath := getenv("KINGAI_DEVICE_IDENTITY", "/etc/kingai/device.json")
	boardID := strings.TrimSpace(os.Getenv("KINGAI_DEVICE_BOARD_ID"))
	deviceID := "unprovisioned"
	deviceClass := "unprovisioned"
	if identity, err := deviceidentity.LoadTrusted(identityPath); err == nil {
		if boardID != "" && boardID != identity.BoardID {
			log.Fatalf("device identity board_id %q conflicts with KINGAI_DEVICE_BOARD_ID %q", identity.BoardID, boardID)
		}
		boardID = identity.BoardID
		deviceID = identity.DeviceID
		deviceClass = identity.Class
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("device identity initialization failed: %v", err)
	}
	timeout := time.Duration(getenvInt("KINGAI_DEVICE_HANDLER_TIMEOUT_SECONDS", 20, 1, 120)) * time.Second

	runtime, err := devicepack.LoadVerifiedRuntime(manifestDir, artifactRoot, trustDir, handlerRoot, boardID, timeout)
	if err != nil { log.Fatalf("device runtime initialization failed: %v", err) }

	rules := make(map[string]policy.Rule)
	for _, capability := range runtime.Capabilities() {
		risk := deviceRiskLevel(capability.Risk)
		ownerOnly := risk == policy.RiskTrustRoot
		rules[capability.ID] = policy.Rule{Risk:risk, ApprovalRequired:capability.ApprovalRequired || ownerOnly, OwnerOnly:ownerOnly}
	}
	if err := policy.SetRuntimeRules(rules); err != nil { log.Fatalf("device runtime policy registration failed: %v", err) }

	brokerSocket := getenv("KINGAI_DEVICE_BROKER_SOCKET", "/run/kingai/device-broker.sock")
	if !filepath.IsAbs(brokerSocket) || filepath.Clean(brokerSocket) != brokerSocket || filepath.Dir(brokerSocket) != "/run/kingai" || strings.ContainsAny(brokerSocket, "\x00\n\r") {
		log.Fatal("device broker socket must be a clean absolute path directly under /run/kingai")
	}
	if err := os.Setenv("KINGAI_EXECD_SOCKET", brokerSocket); err != nil { log.Fatalf("set device broker execution socket: %v", err) }
	if err := os.Remove(brokerSocket); err != nil && !errors.Is(err, os.ErrNotExist) { log.Fatalf("remove stale device broker socket: %v", err) }
	listener, err := net.Listen("unix", brokerSocket)
	if err != nil { log.Fatalf("listen on device broker socket: %v", err) }
	if err := os.Chmod(brokerSocket, 0o600); err != nil { _ = listener.Close(); log.Fatalf("protect device broker socket: %v", err) }

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		writeDeviceJSON(w, http.StatusOK, map[string]any{
			"ok": true, "service": "kingai-device-broker", "verification": "ed25519+sha256",
			"device_id": deviceID, "device_class": deviceClass, "board_id": boardID,
			"device_packs": len(runtime.PackIDs()), "capabilities": len(runtime.Capabilities()),
		})
	})
	mux.HandleFunc("/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		dec := json.NewDecoder(r.Body); dec.DisallowUnknownFields()
		var req executor.Request
		if err := dec.Decode(&req); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return }
		result, err := runtime.Execute(r.Context(), req)
		if err != nil {
			code := http.StatusBadGateway
			switch {
			case errors.Is(err, devicepack.ErrUnknownDeviceCapability): code = http.StatusNotFound
			case errors.Is(err, devicepack.ErrResourceNotDeclared), errors.Is(err, executor.ErrInvalidRequest): code = http.StatusForbidden
			case errors.Is(err, devicepack.ErrHandlerUnavailable): code = http.StatusServiceUnavailable
			}
			writeDeviceJSON(w, code, result); return
		}
		writeDeviceJSON(w, http.StatusOK, result)
	})

	server := &http.Server{Handler:mux, ReadHeaderTimeout:3*time.Second, IdleTimeout:15*time.Second}
	go func() { if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatalf("device broker serve: %v", err) } }()
	log.Printf("KINGAI IoT Device Broker enabled: device=%s board=%s packs=%d capabilities=%d verification=ed25519+sha256", deviceID, boardID, len(runtime.PackIDs()), len(runtime.Capabilities()))
}

func deviceRiskLevel(risk string) policy.RiskLevel {
	if len(risk) == 2 && risk[0] == 'L' && risk[1] >= '0' && risk[1] <= '6' { return policy.RiskLevel(risk[1] - '0') }
	return policy.RiskTrustRoot
}

func writeDeviceJSON(w http.ResponseWriter, code int, value any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(code); _ = json.NewEncoder(w).Encode(value) }
