package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(context.Context, string, []string, []string) (string, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) Run(ctx context.Context, path string, args []string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append([]string(nil), env...)
	var output cappedBuffer
	output.max = 16 << 10
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

type NativeHandlers struct {
	Runner CommandRunner
}

func (h NativeHandlers) Register(b *Broker) error {
	if b == nil {
		return errors.New("broker is required")
	}
	if h.Runner == nil {
		h.Runner = OSCommandRunner{}
	}
	return b.Register("service.restart", HandlerFunc(h.restartService))
}

func (h NativeHandlers) restartService(ctx context.Context, req Request) (Result, error) {
	if err := ValidateServiceUnit(req.Target); err != nil {
		return Result{Capability: req.Capability, Target: req.Target, Message: err.Error()}, err
	}
	if h.Runner == nil {
		h.Runner = OSCommandRunner{}
	}
	output, err := h.Runner.Run(ctx, "/usr/bin/systemctl", []string{"restart", "--", req.Target}, []string{
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
	})
	message := strings.TrimSpace(output)
	if err != nil {
		if message == "" {
			message = "systemctl restart failed"
		}
		return Result{Capability: req.Capability, Target: req.Target, Message: message}, err
	}
	data, _ := json.Marshal(map[string]any{"restarted": true})
	return Result{Capability: req.Capability, Target: req.Target, Data: data, Message: message}, nil
}

type cappedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := b.max - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.buf.Write(p)
	}
	return original, nil
}

func (b *cappedBuffer) String() string { return b.buf.String() }
