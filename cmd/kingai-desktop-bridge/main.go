package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

var version = "0.1.0-dev"

const snapshotSchema = 1

type privateTask struct {
	ID         string           `json:"id"`
	Goal       string           `json:"goal"`
	Agent      string           `json:"agent"`
	Status     taskgraph.Status `json:"status"`
	StepCount  int              `json:"step_count"`
	DoneSteps  int              `json:"done_steps"`
	FailedSteps int             `json:"failed_steps"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type privateSnapshot struct {
	Schema    int           `json:"schema"`
	Product   string        `json:"product"`
	Version   string        `json:"version"`
	UserUID   uint32        `json:"user_uid"`
	Tasks     []privateTask `json:"tasks"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version":
			fmt.Printf("KINGAI Desktop Bridge %s\n", version)
			return
		case "--once":
			if err := runOnce(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		default:
			fmt.Fprintln(os.Stderr, "usage: kingai-desktop-bridge [--once|version]")
			os.Exit(2)
		}
	}

	if os.Geteuid() == 0 && os.Getenv("KINGAI_DESKTOP_BRIDGE_ALLOW_ROOT") != "1" {
		fmt.Fprintln(os.Stderr, "kingai-desktop-bridge is a per-user service and refuses to run as root")
		os.Exit(5)
	}

	interval := 3 * time.Second
	if raw := strings.TrimSpace(os.Getenv("KINGAI_DESKTOP_BRIDGE_INTERVAL")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 1 && seconds <= 300 {
			interval = time.Duration(seconds) * time.Second
		}
	}

	for {
		if err := runOnce(); err != nil {
			fmt.Fprintln(os.Stderr, "KINGAI Desktop Bridge:", err)
		}
		time.Sleep(interval)
	}
}

func runOnce() error {
	socket := strings.TrimSpace(os.Getenv("KINGAI_SOCKET"))
	if socket == "" {
		socket = "/run/kingai/kingaid.sock"
	}
	if !filepath.IsAbs(socket) || strings.ContainsAny(socket, "\x00\n\r") {
		return errors.New("KINGAI_SOCKET must be an absolute local unix-socket path")
	}

	output, err := privateSnapshotPath()
	if err != nil {
		return err
	}

	client := unixHTTPClient(socket, 2500*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	var tasks []taskgraph.Task
	if err := getJSON(ctx, client, "/v1/tasks/list", &tasks); err != nil {
		return fmt.Errorf("fetch private tasks: %w", err)
	}

	snapshot := privateSnapshot{
		Schema:    snapshotSchema,
		Product:   "KINGAI OS Desktop",
		Version:   version,
		UserUID:   uint32(os.Geteuid()),
		Tasks:     sanitizeTasks(tasks),
		UpdatedAt: time.Now().UTC(),
	}
	return writePrivateSnapshot(output, snapshot)
}

func sanitizeTasks(tasks []taskgraph.Task) []privateTask {
	out := make([]privateTask, 0, len(tasks))
	for _, task := range tasks {
		item := privateTask{
			ID:        task.ID,
			Goal:      strings.TrimSpace(task.Goal),
			Agent:     strings.TrimSpace(task.Agent),
			Status:    task.Status,
			StepCount: len(task.Steps),
			CreatedAt: task.CreatedAt,
			UpdatedAt: task.UpdatedAt,
		}
		for _, step := range task.Steps {
			switch step.Status {
			case taskgraph.StatusCompleted:
				item.DoneSteps++
			case taskgraph.StatusFailed, taskgraph.StatusBlocked:
				item.FailedSteps++
			}
		}
		out = append(out, item)
	}
	return out
}

func privateSnapshotPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("KINGAI_DESKTOP_PRIVATE_STATUS")); override != "" {
		if !filepath.IsAbs(override) || strings.ContainsAny(override, "\x00\n\r") {
			return "", errors.New("KINGAI_DESKTOP_PRIVATE_STATUS must be an absolute path")
		}
		return override, nil
	}

	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Geteuid())
	}
	if !filepath.IsAbs(runtimeDir) || strings.ContainsAny(runtimeDir, "\x00\n\r") {
		return "", errors.New("XDG_RUNTIME_DIR must be an absolute path")
	}
	if err := validateRuntimeDir(runtimeDir); err != nil {
		return "", err
	}
	return filepath.Join(runtimeDir, "kingai", "desktop-private.json"), nil
}

func validateRuntimeDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("runtime directory unavailable: %w", err)
	}
	if !info.IsDir() {
		return errors.New("runtime path is not a directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("runtime directory ownership unavailable")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("runtime directory owner uid %d does not match bridge uid %d", stat.Uid, os.Geteuid())
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("runtime directory permissions %o are too broad", info.Mode().Perm())
	}
	return nil
}

func writePrivateSnapshot(path string, snapshot privateSnapshot) error {
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func unixHTTPClient(socket string, timeout time.Duration) *http.Client {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &http.Client{Transport: transport, Timeout: timeout}
}

func getJSON(ctx context.Context, client *http.Client, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://kingai"+path, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}
