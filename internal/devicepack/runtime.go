package devicepack

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/executor"
)

var (
	ErrUnknownDeviceCapability = errors.New("device capability is not registered")
	ErrResourceNotDeclared     = errors.New("device resource is not declared")
	ErrHandlerUnavailable      = errors.New("device handler is unavailable")
)

const (
	maxManifestBytes = 1 << 20
	maxResponseBytes = 256 << 10
)

type RuntimeCapability struct {
	ID               string   `json:"id"`
	PackID           string   `json:"pack_id"`
	Handler          string   `json:"handler"`
	Operation        string   `json:"operation"`
	Risk             string   `json:"risk"`
	ApprovalRequired bool     `json:"approval_required"`
	Resources        []string `json:"resources"`
}

type Runtime struct {
	socketRoot   string
	timeout      time.Duration
	capabilities map[string]RuntimeCapability
	packIDs      []string
}

// LoadRuntime loads root-provisioned Device Pack manifests for the current CPU
// architecture. A pack with board_ids is accepted only when boardID exactly
// matches one of them. This prevents a signed pack for one board from silently
// becoming authoritative on another board family.
func LoadRuntime(manifestDir, socketRoot, boardID string, timeout time.Duration) (*Runtime, error) {
	if err := validateAbsolutePath(manifestDir); err != nil {
		return nil, fmt.Errorf("device-pack directory: %w", err)
	}
	if err := validateAbsolutePath(socketRoot); err != nil {
		return nil, fmt.Errorf("device handler socket root: %w", err)
	}
	if err := validateTrustedDirectory(socketRoot); err != nil {
		return nil, fmt.Errorf("device handler socket root: %w", err)
	}

	entries, err := os.ReadDir(manifestDir)
	if errors.Is(err, os.ErrNotExist) {
		return NewRuntime(nil, socketRoot, boardID, timeout)
	}
	if err != nil {
		return nil, err
	}
	if err := validateTrustedDirectory(manifestDir); err != nil {
		return nil, fmt.Errorf("device-pack directory: %w", err)
	}

	manifests := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(manifestDir, entry.Name())
		if err := validateTrustedManifestFile(path); err != nil {
			return nil, err
		}
		manifest, err := Load(path)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		manifests = append(manifests, manifest)
	}
	return NewRuntime(manifests, socketRoot, boardID, timeout)
}

// NewRuntime builds an in-memory capability index. It is intentionally useful
// in unit tests and tooling; filesystem trust checks are performed by
// LoadRuntime before production manifests enter this constructor.
func NewRuntime(manifests []Manifest, socketRoot, boardID string, timeout time.Duration) (*Runtime, error) {
	if err := validateAbsolutePath(socketRoot); err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	if timeout > 2*time.Minute {
		timeout = 2 * time.Minute
	}

	r := &Runtime{
		socketRoot:   filepath.Clean(socketRoot),
		timeout:      timeout,
		capabilities: make(map[string]RuntimeCapability),
	}
	seenPacks := make(map[string]struct{}, len(manifests))
	for _, manifest := range manifests {
		if err := manifest.Validate(); err != nil {
			return nil, fmt.Errorf("device pack %q: %w", manifest.ID, err)
		}
		if manifest.Arch != runtime.GOARCH {
			return nil, fmt.Errorf("device pack %q targets %s, runtime is %s", manifest.ID, manifest.Arch, runtime.GOARCH)
		}
		if len(manifest.BoardIDs) > 0 && !containsString(manifest.BoardIDs, boardID) {
			if strings.TrimSpace(boardID) == "" {
				return nil, fmt.Errorf("device pack %q requires an explicit board id", manifest.ID)
			}
			return nil, fmt.Errorf("device pack %q does not match board id %q", manifest.ID, boardID)
		}
		if _, exists := seenPacks[manifest.ID]; exists {
			return nil, fmt.Errorf("duplicate device pack id %q", manifest.ID)
		}
		seenPacks[manifest.ID] = struct{}{}
		r.packIDs = append(r.packIDs, manifest.ID)

		for _, capability := range manifest.Capabilities {
			if _, exists := r.capabilities[capability.ID]; exists {
				return nil, fmt.Errorf("device capability %q is declared by more than one pack", capability.ID)
			}
			resources := append([]string(nil), capability.Resources...)
			sort.Strings(resources)
			r.capabilities[capability.ID] = RuntimeCapability{
				ID:               capability.ID,
				PackID:           manifest.ID,
				Handler:          capability.Handler,
				Operation:        capability.Operation,
				Risk:             capability.Risk,
				ApprovalRequired: capability.ApprovalRequired,
				Resources:        resources,
			}
		}
	}
	sort.Strings(r.packIDs)
	return r, nil
}

func (r *Runtime) PackIDs() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.packIDs...)
}

func (r *Runtime) Capabilities() []RuntimeCapability {
	if r == nil {
		return nil
	}
	out := make([]RuntimeCapability, 0, len(r.capabilities))
	for _, capability := range r.capabilities {
		capability.Resources = append([]string(nil), capability.Resources...)
		out = append(out, capability)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Runtime) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	if r == nil {
		return executor.Result{}, ErrUnknownDeviceCapability
	}
	capability, ok := r.capabilities[strings.TrimSpace(req.Capability)]
	if !ok {
		return executor.Result{}, ErrUnknownDeviceCapability
	}
	if strings.TrimSpace(req.Agent) == "" || len(req.Agent) > executor.MaxAgentBytes {
		return executor.Result{}, executor.ErrInvalidRequest
	}
	if len(req.Target) > executor.MaxTargetBytes || strings.ContainsRune(req.Target, '\x00') {
		return executor.Result{}, executor.ErrInvalidRequest
	}
	if len(req.Arguments) > executor.MaxArgumentsBytes || (len(req.Arguments) > 0 && !json.Valid(req.Arguments)) {
		return executor.Result{}, executor.ErrInvalidRequest
	}
	if !containsString(capability.Resources, req.Target) {
		return executor.Result{}, fmt.Errorf("%w: %s", ErrResourceNotDeclared, req.Target)
	}

	socketPath := filepath.Join(r.socketRoot, capability.Handler+".sock")
	if filepath.Dir(socketPath) != r.socketRoot {
		return executor.Result{}, ErrHandlerUnavailable
	}
	if err := validateHandlerSocket(socketPath); err != nil {
		return executor.Result{}, err
	}

	executionID, err := newRuntimeExecutionID()
	if err != nil {
		return executor.Result{}, err
	}
	started := time.Now().UTC()
	result := executor.Result{
		ExecutionID: executionID,
		Agent:       req.Agent,
		Capability:  capability.ID,
		Target:      req.Target,
		StartedAt:   started,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return result, err
	}
	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: r.timeout}
	httpReq, err := http.NewRequestWithContext(execCtx, http.MethodPost, "http://kingai-device/v1/execute", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
	if err != nil {
		return finishRuntimeResult(result, false, "device handler request failed"), fmt.Errorf("%w: %v", ErrHandlerUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		messageBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		message := strings.TrimSpace(string(messageBytes))
		if message == "" {
			message = resp.Status
		}
		return finishRuntimeResult(result, false, message), fmt.Errorf("device handler returned %s", resp.Status)
	}

	var handlerResult executor.Result
	dec := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	if err := dec.Decode(&handlerResult); err != nil {
		return finishRuntimeResult(result, false, "invalid device handler response"), err
	}
	result.Data = handlerResult.Data
	result.Message = handlerResult.Message
	result = finishRuntimeResult(result, handlerResult.OK, result.Message)
	if !handlerResult.OK {
		if result.Message == "" {
			result.Message = "device handler reported failure"
		}
		return result, errors.New(result.Message)
	}
	return result, nil
}

func finishRuntimeResult(result executor.Result, ok bool, message string) executor.Result {
	result.OK = ok
	result.Message = strings.TrimSpace(message)
	result.FinishedAt = time.Now().UTC()
	result.DurationMS = result.FinishedAt.Sub(result.StartedAt).Milliseconds()
	return result
}

func validateAbsolutePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "\x00\n\r") {
		return errors.New("path must be a clean absolute local path")
	}
	return nil
}

func validateTrustedDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("directory must not be group/world writable")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return errors.New("directory must be owned by root")
	}
	return nil
}

func validateTrustedManifestFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("device-pack manifest %s must be a regular file", filepath.Base(path))
	}
	if info.Size() <= 0 || info.Size() > maxManifestBytes {
		return fmt.Errorf("device-pack manifest %s has invalid size", filepath.Base(path))
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("device-pack manifest %s must not be group/world writable", filepath.Base(path))
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return fmt.Errorf("device-pack manifest %s must be owned by root", filepath.Base(path))
	}
	return nil
}

func validateHandlerSocket(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHandlerUnavailable, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%w: handler endpoint is not a unix socket", ErrHandlerUnavailable)
	}
	// The socket is intentionally private to the unprivileged kingaid identity.
	// A privileged handler may bind it and chown it to _kingai before serving.
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: handler socket must not grant group/world access", ErrHandlerUnavailable)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("%w: handler socket must be owned by the kingaid uid", ErrHandlerUnavailable)
	}
	return nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func newRuntimeExecutionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
