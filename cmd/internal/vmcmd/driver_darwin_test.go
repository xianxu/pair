//go:build darwin

package vmcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func fakeNativeDriver(t *testing.T) *nativeDriver {
	t.Helper()
	home := t.TempDir()
	executable := filepath.Join(home, "tart")
	script := `#!/bin/sh
set -eu
test "${TART_HOME-unset}" = unset
test "${OPENAI_API_KEY-unset}" = unset
printf '%s\n' "$@" >> "$HOME/argv"
case "$1" in
 list)
  if test -f "$HOME/list"; then cat "$HOME/list"; elif test -f "$HOME/running"; then printf '[{"Name":"pair-conformance-run-adhoc","Running":true,"State":"running"}]'; else printf '[{"Name":"pair-conformance-run-adhoc","Running":false,"State":"stopped","Other":12}]'; fi;;
 clone) touch "$HOME/cloned"; if test -f "$HOME/block"; then sleep 30; fi;;
 run) touch "$HOME/running"; if test -f "$HOME/boot-output"; then printf boot-output; fi; while test -f "$HOME/running"; do if test -f "$HOME/no-descendants"; then :; else sleep 0.01; fi; done;;
 exec) test -f "$HOME/running" || exit 3; printf '%s\n' "$HOME"; test "$3" != fail || exit 7;;
 stop) rm -f "$HOME/running";;
 delete) touch "$HOME/deleted";;
 *) exit 9;;
esac
`
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return newNativeDriver(executable, home, os.Getenv("PATH"))
}

func TestNativeInspectStrictList(t *testing.T) {
	d := fakeNativeDriver(t)
	for _, tc := range []struct {
		raw  string
		want VMStatus
		fail bool
	}{
		{`[]`, VMStatus{}, false},
		{`[{"Name":"pair-conformance-run-adhoc","Running":false,"State":"stopped","Source":"local"}]`, VMStatus{true, VMStopped}, false},
		{`[{"Name":"pair-conformance-run-adhoc","Running":false,"State":"mystery"}]`, VMStatus{true, VMUnknown}, false},
		{`[{"Name":"pair-conformance-run-adhoc","Running":"false","State":"stopped"}]`, VMStatus{}, true},
		{`[{"Name":"pair-conformance-run-adhoc","Name":"other","Running":false,"State":"stopped"}]`, VMStatus{}, true},
		{`[{"Name":"pair-conformance-run-adhoc","Running":false,"State":"stopped"},{"Name":"pair-conformance-run-adhoc","Running":false,"State":"stopped"}]`, VMStatus{}, true},
		{`null`, VMStatus{}, true},
	} {
		if err := os.WriteFile(filepath.Join(d.home, "list"), []byte(tc.raw), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := d.Inspect(context.Background(), ManagedVMName(AdhocLane))
		if (err != nil) != tc.fail || (!tc.fail && got != tc.want) {
			t.Errorf("%s: got %+v %v", tc.raw, got, err)
		}
	}
}

func TestNativeLifecycleOwnsChildAndSanitizesTransport(t *testing.T) {
	d := fakeNativeDriver(t)
	vm := ManagedVMName(AdhocLane)
	t.Setenv("HOME", "/poison")
	t.Setenv("TART_HOME", "/poison")
	t.Setenv("OPENAI_API_KEY", "poison")
	ctx, cancel := context.WithCancel(context.Background())
	if err := d.Clone(ctx, "pair-conformance-image", vm); err != nil {
		t.Fatal(err)
	}
	child, err := d.Launch(ctx, vm, "/input with space", "/results", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	bound, done := context.WithTimeout(context.Background(), 3*time.Second)
	defer done()
	if err := d.AwaitGuest(bound, vm); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := d.Execute(bound, vm, []string{"literal; $(touch /poison)", "a b"}, &out, &out)
	if err != nil || code != 0 || strings.TrimSpace(out.String()) != d.home {
		t.Fatalf("exec %d %v %q", code, err, out.String())
	}
	if child.PID() <= 0 || child.Identity() == "" {
		t.Fatalf("unowned child %v", child)
	}
	if err := d.Stop(bound, vm, child); err != nil {
		t.Fatal(err)
	}
	proof, err := d.Recover(bound, LaunchClaim{Version: 1, Lane: AdhocLane, VM: vm, Nonce: strings.Repeat("a", 32), Stage: ClaimLaunched, PID: child.PID(), StartIdentity: child.Identity()})
	if err != nil || proof.Process != ProcessGone || proof.VM != VMStopped {
		t.Fatalf("proof %+v %v", proof, err)
	}
	if err := d.Delete(bound, "pair-test"); err == nil {
		t.Fatal("deleted foreign VM")
	}
	if err := d.Delete(bound, vm); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(d.home, "argv"))
	if !strings.Contains(string(raw), "--no-graphics\n--no-audio\n--no-clipboard\n--dir\ninput:/input with space:ro\n--dir\nresults:/results\n") {
		t.Fatalf("bad shares: %s", raw)
	}
}

func TestNativeLaunchReturnsHandleWhenIdentityUnknown(t *testing.T) {
	d := fakeNativeDriver(t)
	d.identity = func(string) string { return "" }
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	child, err := d.Launch(ctx, ManagedVMName(AdhocLane), "/input", "/results", &bytes.Buffer{})
	if err == nil || child == nil || child.PID() == 0 {
		t.Fatalf("lost started child: %v %v", child, err)
	}
	if err := d.AwaitGuest(ctx, ManagedVMName(AdhocLane)); err != nil {
		t.Fatal(err)
	}
	if err := d.Stop(ctx, ManagedVMName(AdhocLane), child); err != nil {
		t.Fatal(err)
	}
}

func TestNativePreservesUnknownRecoveryAndRefusesUnownedStop(t *testing.T) {
	d := fakeNativeDriver(t)
	vm := ManagedVMName(AdhocLane)
	claim := LaunchClaim{Version: 1, Lane: AdhocLane, VM: vm, Nonce: strings.Repeat("b", 32), Stage: ClaimLaunched, PID: os.Getpid()}
	proof, err := d.Recover(context.Background(), claim)
	if err != nil || proof.Process != ProcessUnknown {
		t.Fatalf("invented absence: %+v %v", proof, err)
	}
	if err := os.WriteFile(filepath.Join(d.home, "running"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := d.Stop(context.Background(), vm, nil); err == nil {
		t.Fatal("accepted running VM without owned child")
	}
	raw, _ := os.ReadFile(filepath.Join(d.home, "argv"))
	if strings.Contains(string(raw), "stop\n") {
		t.Fatal("attempted to stop unowned VM")
	}
}

func TestNativeLaunchStartFailureHasNoHandle(t *testing.T) {
	d := fakeNativeDriver(t)
	d.executable = filepath.Join(d.home, "missing")
	child, err := d.Launch(context.Background(), ManagedVMName(AdhocLane), "/input", "/results", &bytes.Buffer{})
	if err == nil || child != nil {
		t.Fatalf("start failure: %v %v", child, err)
	}
}

func TestNativeExecutionFailureIsNotRetried(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "running"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	code, err := d.Execute(context.Background(), ManagedVMName(AdhocLane), []string{"fail"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 7 {
		t.Fatalf("lost guest failure: %d %v", code, err)
	}
	raw, _ := os.ReadFile(filepath.Join(d.home, "argv"))
	if strings.Count(string(raw), "exec\n") != 1 {
		t.Fatalf("retried execution: %s", raw)
	}
}

func TestNativeStopRejectsOtherDriversChild(t *testing.T) {
	d := fakeNativeDriver(t)
	other := fakeNativeDriver(t)
	vm := ManagedVMName(AdhocLane)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	child, err := d.Launch(ctx, vm, "/input", "/results", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Stop(ctx, vm, child)
	if err := d.AwaitGuest(ctx, vm); err != nil {
		t.Fatal(err)
	}
	if err := other.Stop(ctx, vm, child); err == nil {
		t.Fatal("another driver joined foreign child")
	}
}

func TestNativeClonePreservesUnconfirmedProcessCleanup(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "block"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := d.Clone(ctx, "pair-conformance-image", ManagedVMName(AdhocLane))
	if !errors.Is(err, ErrUnconfirmedProcessCleanup) {
		t.Fatalf("cleanup uncertainty not preserved: %v", err)
	}
}

type rejectingBootWriter struct{ seen chan struct{} }

func (w rejectingBootWriter) Write(p []byte) (int, error) {
	select {
	case w.seen <- struct{}{}:
	default:
	}
	return 0, io.ErrShortWrite
}

func TestNativeBootWriterErrorSignalsAndJoinsOwnedGroup(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "boot-output"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	writer := rejectingBootWriter{seen: make(chan struct{}, 1)}
	handle, err := d.Launch(context.Background(), ManagedVMName(AdhocLane), "/input", "/results", writer)
	if handle == nil {
		t.Fatalf("lost launch handle: %v", err)
	}
	child := handle.(*nativeChild)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer func() {
		_ = os.Remove(filepath.Join(d.home, "running"))
		_ = d.Stop(ctx, ManagedVMName(AdhocLane), child)
	}()
	select {
	case <-writer.seen:
	case <-ctx.Done():
		t.Fatal("boot output never reached writer")
	}
	select {
	case <-child.done:
	case <-ctx.Done():
		t.Fatal("writer error left owned launch blocked")
	}
	if err := nativeGroupCleanup(ctx, child.cmd); err != nil {
		t.Fatalf("owned group survived boot failure: %v", err)
	}
}

// Darwin can return EPERM for a group containing only non-signalable zombies.
// The signal result is ambiguous; the owned process table is the proof.
func TestNativeGroupCleanupConfirmsGoneAfterAmbiguousSignal(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "no-descendants"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	handle, err := d.Launch(ctx, ManagedVMName(AdhocLane), "/input", "/results", io.Discard)
	if handle == nil {
		t.Fatalf("launch: %v", err)
	}
	child := handle.(*nativeChild)
	signal := child.cmd.Cancel
	defer func() {
		child.cmd.Cancel = signal
		_ = os.Remove(filepath.Join(d.home, "running"))
		_ = d.Stop(ctx, ManagedVMName(AdhocLane), child)
	}()
	if err := d.AwaitGuest(ctx, ManagedVMName(AdhocLane)); err != nil {
		t.Fatal(err)
	}
	child.cmd.Cancel = func() error { _ = signal(); return syscall.EPERM }
	if err := nativeGroupCleanup(ctx, child.cmd); err != nil {
		t.Fatalf("did not confirm disappearance after ambiguous signal: %v", err)
	}
	select {
	case <-child.done:
	case <-ctx.Done():
		t.Fatal("owned child not joined")
	}
}

func TestNativeGroupCleanupRetainsUnconfirmedPresentGroup(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "no-descendants"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	handle, err := d.Launch(ctx, ManagedVMName(AdhocLane), "/input", "/results", io.Discard)
	if handle == nil {
		t.Fatalf("launch: %v", err)
	}
	child := handle.(*nativeChild)
	signal := child.cmd.Cancel
	defer func() {
		child.cmd.Cancel = signal
		_ = os.Remove(filepath.Join(d.home, "running"))
		_ = d.Stop(ctx, ManagedVMName(AdhocLane), child)
	}()
	if err := d.AwaitGuest(ctx, ManagedVMName(AdhocLane)); err != nil {
		t.Fatal(err)
	}
	child.cmd.Cancel = func() error { return syscall.EPERM }
	short, done := context.WithTimeout(ctx, 30*time.Millisecond)
	defer done()
	if err := nativeGroupCleanup(short, child.cmd); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("did not wait boundedly for positive absence: %v", err)
	}
	select {
	case <-child.done:
		t.Fatal("claimed absence of live child")
	default:
	}
}
