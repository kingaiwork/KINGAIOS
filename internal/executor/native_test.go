package executor

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeRunner struct {
	path string
	args []string
	env  []string
	out  string
	err  error
}

func (f *fakeRunner) Run(ctx context.Context, path string, args []string, env []string) (string, error) {
	f.path = path
	f.args = append([]string(nil), args...)
	f.env = append([]string(nil), env...)
	return f.out, f.err
}

func TestNativeRestartUsesFixedSystemctlInvocation(t *testing.T) {
	runner := &fakeRunner{out: "ok"}
	b := New(time.Second)
	if err := (NativeHandlers{Runner: runner}).Register(b); err != nil { t.Fatal(err) }
	out, err := b.Execute(context.Background(), Request{Agent: "system-ops", Capability: "service.restart", Target: "nginx.service"})
	if err != nil { t.Fatal(err) }
	if !out.OK { t.Fatalf("out=%#v", out) }
	if runner.path != "/usr/bin/systemctl" { t.Fatalf("path=%s", runner.path) }
	want := []string{"restart", "--", "nginx.service"}
	if !reflect.DeepEqual(runner.args, want) { t.Fatalf("args=%v want=%v", runner.args, want) }
}

func TestNativeRestartRejectsInjectionLikeTarget(t *testing.T) {
	runner := &fakeRunner{}
	b := New(time.Second)
	if err := (NativeHandlers{Runner: runner}).Register(b); err != nil { t.Fatal(err) }
	_, err := b.Execute(context.Background(), Request{Agent: "system-ops", Capability: "service.restart", Target: "nginx.service;touch /tmp/pwned"})
	if err == nil { t.Fatal("expected target validation failure") }
	if runner.path != "" { t.Fatal("runner must not execute invalid target") }
}

func TestNativeRestartPropagatesRunnerFailure(t *testing.T) {
	runner := &fakeRunner{out: "failed", err: errors.New("exit 1")}
	b := New(time.Second)
	if err := (NativeHandlers{Runner: runner}).Register(b); err != nil { t.Fatal(err) }
	out, err := b.Execute(context.Background(), Request{Agent: "system-ops", Capability: "service.restart", Target: "nginx.service"})
	if err == nil { t.Fatal("expected failure") }
	if out.OK || out.Message != "failed" { t.Fatalf("out=%#v", out) }
}
