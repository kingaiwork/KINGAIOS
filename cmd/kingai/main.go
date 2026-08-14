package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kingaiwork/KINGAIOS/internal/approval"
	"github.com/kingaiwork/KINGAIOS/internal/desktop"
	"github.com/kingaiwork/KINGAIOS/internal/diagnostics"
	"github.com/kingaiwork/KINGAIOS/internal/executor"
	"github.com/kingaiwork/KINGAIOS/internal/memory"
	"github.com/kingaiwork/KINGAIOS/internal/model"
	"github.com/kingaiwork/KINGAIOS/internal/policy"
	"github.com/kingaiwork/KINGAIOS/internal/taskgraph"
)

var version = "0.1.0-dev"

func usage() {
	fmt.Print(`KINGAI OS CLI

Usage:
  kingai version
  kingai status [--json]
  kingai doctor [--json] [--repair-safe]
  kingai policy check <capability> [target]
  kingai approval request <agent> <capability> [target]
  kingai approval list
  kingai approval approve <id>
  kingai approval deny <id>
  kingai execution run <agent> <capability> <target> [approval-id]
  kingai memory put <kind> <json>
  kingai memory list
  kingai memory search <text...>
  kingai memory delete <id>
  kingai model select <capability> [--private] [--offline]
  kingai task create <agent> <goal...>
  kingai task list
  kingai task run <id>
  kingai task transition <id> <status>
  kingai task step <id> <step-id> <status>
  kingai desktop list
  kingai desktop show
  kingai desktop set <kingai-intelligence|kingai-flow|kingai-classic>
  kingai desktop apply
`)
}

func main() {
	if len(os.Args) < 2 { usage(); return }
	switch os.Args[1] {
	case "version": fmt.Printf("KINGAI OS %s\n", version)
	case "status": status(os.Args[2:])
	case "doctor": doctor(os.Args[2:])
	case "policy": policyCmd(os.Args[2:])
	case "approval": approvalCmd(os.Args[2:])
	case "execution": executionCmd(os.Args[2:])
	case "memory": memoryCmd(os.Args[2:])
	case "model": modelCmd(os.Args[2:])
	case "task": taskCmd(os.Args[2:])
	case "desktop": desktopCmd(os.Args[2:])
	default: usage(); os.Exit(2)
	}
}

func status(args []string) {
	var remote map[string]any
	if err := daemonJSON(http.MethodGet, "/v1/status", nil, &remote); err == nil {
		if len(args) > 0 && strings.EqualFold(args[0], "--json") { _ = json.NewEncoder(os.Stdout).Encode(remote); return }
		fmt.Printf("System:       %v\nVersion:      %v\nArchitecture: %v\nPolicy:       %v\nExecution:    %v\n", remote["name"], remote["version"], remote["architecture"], remote["policy"], remote["execution_broker"])
		return
	}
	fallback := map[string]any{"name":"KINGAI OS","version":version,"channel":"dev","platform":runtime.GOOS+"/"+runtime.GOARCH,"daemon":"offline"}
	if len(args) > 0 && strings.EqualFold(args[0], "--json") { _ = json.NewEncoder(os.Stdout).Encode(fallback); return }
	fmt.Printf("System:  KINGAI OS\nVersion: %s\nPlatform: %s/%s\nDaemon:  offline\n", version, runtime.GOOS, runtime.GOARCH)
}

func doctor(args []string) {
	jsonMode := false
	repairSafe := false
	for _, arg := range args { switch arg { case "--json": jsonMode = true; case "--repair-safe": repairSafe = true; default: usage(); os.Exit(2) } }
	if repairSafe && os.Geteuid() != 0 { fmt.Fprintln(os.Stderr, "kingai doctor --repair-safe requires root; diagnostic-only doctor does not"); os.Exit(5) }
	root := os.Getenv("KINGAI_DIAGNOSTIC_ROOT")
	var health map[string]any
	daemonErr := daemonJSON(http.MethodGet, "/healthz", nil, &health)
	report := diagnostics.Run(diagnostics.Options{Root: root, DaemonError: daemonErr})
	if repairSafe {
		repairs := diagnostics.ApplySafeRepairs(root, report)
		if len(repairs) > 0 { report = diagnostics.Run(diagnostics.Options{Root: root, DaemonError: daemonErr}); report.Repairs = repairs }
	}
	if jsonMode {
		enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(report)
	} else {
		fmt.Println("KINGAI Health Intelligence")
		fmt.Printf("Status: %s   Score: %d/100\n", strings.ToUpper(report.Status), report.Score)
		for _, c := range report.Checks {
			fmt.Printf("[%s] %-16s %s\n", c.Status, c.ID, c.Summary)
			if c.ActionID != "" && (c.Status == "warn" || c.Status == "fail") { fmt.Printf("       action=%s  mode=%s\n", c.ActionID, c.RemediationMode) }
		}
		if len(report.Repairs) > 0 { fmt.Println("Safe repairs:"); for _, r := range report.Repairs { fmt.Printf("  [%s] %s: %s\n", r.Status, r.ActionID, r.Summary) } }
		if len(report.NextActions) > 0 { fmt.Println("Recommended next actions:"); for i, a := range report.NextActions { fmt.Printf("  %d. %s\n", i+1, a) } }
	}
	if report.Status == "critical" { os.Exit(4) }
}

func policyCmd(args []string) {
	if len(args) < 2 || args[0] != "check" { usage(); os.Exit(2) }
	req := policy.Request{Agent: "main", Capability: args[1]}
	if len(args) > 2 { req.Target = args[2] }
	var out policy.Result
	if err := daemonJSON(http.MethodPost, "/v1/policy/evaluate", req, &out); err != nil { out = policy.Default().Evaluate(req) }
	_ = json.NewEncoder(os.Stdout).Encode(out)
	if !out.Allowed { os.Exit(3) }
}

func approvalCmd(args []string) {
	if len(args) < 1 { usage(); os.Exit(2) }
	switch args[0] {
	case "request":
		if len(args) < 3 { usage(); os.Exit(2) }
		body := map[string]any{"agent": args[1], "capability": args[2]}
		if len(args) > 3 { body["target"] = args[3] }
		var out approval.Request
		if err := daemonJSON(http.MethodPost, "/v1/approval/request", body, &out); err != nil { fail(err) }
		printJSON(out)
	case "list":
		var out []approval.Request
		if err := daemonJSON(http.MethodGet, "/v1/approval/list", nil, &out); err != nil { fail(err) }
		printJSON(out)
	case "approve", "deny":
		if len(args) != 2 { usage(); os.Exit(2) }
		var out approval.Request
		if err := daemonJSON(http.MethodPost, "/v1/approval/decision", map[string]any{"id": args[1], "action": args[0]}, &out); err != nil { fail(err) }
		printJSON(out)
	default: usage(); os.Exit(2)
	}
}

func executionCmd(args []string) {
	if len(args) < 4 || args[0] != "run" { usage(); os.Exit(2) }
	body := map[string]any{"agent": args[1], "capability": args[2], "target": args[3]}
	if len(args) > 4 { body["approval_id"] = args[4] }
	var out executor.Result
	if err := daemonJSONWithTimeout(http.MethodPost, "/v1/execution/run", body, &out, 40*time.Second); err != nil { fail(err) }
	printJSON(out)
}

func memoryCmd(args []string) {
	if len(args) < 1 { usage(); os.Exit(2) }
	switch args[0] {
	case "put":
		if len(args) != 3 || !json.Valid([]byte(args[2])) { fmt.Fprintln(os.Stderr, "memory data must be valid JSON"); os.Exit(2) }
		body := map[string]any{"agent": "main", "kind": args[1], "sensitivity": "private", "data": json.RawMessage(args[2])}
		var out memory.Record
		if err := daemonJSON(http.MethodPost, "/v1/memory/put", body, &out); err != nil { fail(err) }
		printJSON(out)
	case "list":
		var out []memory.Record
		if err := daemonJSON(http.MethodGet, "/v1/memory/list", nil, &out); err != nil { fail(err) }
		printJSON(out)
	case "search":
		if len(args) < 2 { usage(); os.Exit(2) }
		var out []memory.Record
		if err := daemonJSON(http.MethodPost, "/v1/memory/search", memory.Query{Text: strings.Join(args[1:], " "), Limit: 100}, &out); err != nil { fail(err) }
		printJSON(out)
	case "delete":
		if len(args) != 2 { usage(); os.Exit(2) }
		var out map[string]any
		if err := daemonJSON(http.MethodPost, "/v1/memory/delete", map[string]any{"id": args[1]}, &out); err != nil { fail(err) }
		printJSON(out)
	default: usage(); os.Exit(2)
	}
}

func modelCmd(args []string) {
	if len(args) < 2 || args[0] != "select" { usage(); os.Exit(2) }
	req := model.Request{Capability: args[1]}
	for _, arg := range args[2:] { switch arg { case "--private": req.Private = true; case "--offline": req.Offline = true; default: usage(); os.Exit(2) } }
	var out model.Candidate
	if err := daemonJSON(http.MethodPost, "/v1/model/select", req, &out); err != nil { fail(err) }
	printJSON(out)
}

func taskCmd(args []string) {
	if len(args) < 1 { usage(); os.Exit(2) }
	switch args[0] {
	case "create":
		if len(args) < 3 { usage(); os.Exit(2) }
		body := map[string]any{"agent": args[1], "goal": strings.Join(args[2:], " ")}
		var out taskgraph.Task
		if err := daemonJSON(http.MethodPost, "/v1/tasks/create", body, &out); err != nil { fail(err) }
		printJSON(out)
	case "list":
		var out []taskgraph.Task
		if err := daemonJSON(http.MethodGet, "/v1/tasks/list", nil, &out); err != nil { fail(err) }
		printJSON(out)
	case "run":
		if len(args) != 2 { usage(); os.Exit(2) }
		var out taskgraph.Task
		if err := daemonJSONWithTimeout(http.MethodPost, "/v1/tasks/run", map[string]any{"id": args[1]}, &out, 40*time.Second); err != nil { fail(err) }
		printJSON(out)
	case "transition":
		if len(args) != 3 { usage(); os.Exit(2) }
		var out taskgraph.Task
		if err := daemonJSON(http.MethodPost, "/v1/tasks/transition", map[string]any{"id": args[1], "status": args[2]}, &out); err != nil { fail(err) }
		printJSON(out)
	case "step":
		if len(args) != 4 { usage(); os.Exit(2) }
		var out taskgraph.Task
		body := map[string]any{"id": args[1], "step_id": args[2], "status": args[3]}
		if err := daemonJSON(http.MethodPost, "/v1/tasks/step/transition", body, &out); err != nil { fail(err) }
		printJSON(out)
	default: usage(); os.Exit(2)
	}
}

func desktopCmd(args []string) {
	if len(args) < 1 { usage(); os.Exit(2) }
	switch args[0] {
	case "list": for _, e := range desktop.List() { fmt.Printf("%s\t%s\n", e.ID, e.Name) }
	case "show":
		v, err := desktop.Current(); if err != nil { fail(err) }
		if v == "" { fmt.Println("unselected") } else { fmt.Println(v) }
	case "set":
		if len(args) != 2 { usage(); os.Exit(2) }
		if err := desktop.Set(args[1], true); err != nil { fail(err) }
		fmt.Printf("desktop experience set to %s\n", args[1])
	case "apply":
		if err := desktop.ApplyCurrent(); err != nil { fail(err) }
		fmt.Println("desktop experience applied")
	default: usage(); os.Exit(2)
	}
}

func printJSON(v any) { enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); _ = enc.Encode(v) }
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
func daemonJSON(method, path string, body any, out any) error { return daemonJSONWithTimeout(method, path, body, out, 2*time.Second) }

func daemonJSONWithTimeout(method, path string, body any, out any, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout); defer cancel()
	socketPath := os.Getenv("KINGAI_SOCKET"); if socketPath == "" { socketPath = "/run/kingai/kingaid.sock" }
	if !filepath.IsAbs(socketPath) || strings.ContainsAny(socketPath, "\x00\n\r") { return fmt.Errorf("KINGAI_SOCKET must be an absolute local unix-socket path") }
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return (&net.Dialer{}).DialContext(ctx, "unix", socketPath) }}; defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: timeout}
	var reader io.Reader
	if body != nil { b, err := json.Marshal(body); if err != nil { return err }; reader = bytes.NewReader(b) }
	req, err := http.NewRequestWithContext(ctx, method, "http://kingai"+path, reader); if err != nil { return err }
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	resp, err := client.Do(req); if err != nil { return err }; defer resp.Body.Close()
	if resp.StatusCode >= 300 { msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096)); return fmt.Errorf("daemon returned %s: %s", resp.Status, strings.TrimSpace(string(msg))) }
	return json.NewDecoder(resp.Body).Decode(out)
}
