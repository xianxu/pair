//go:build darwin

package vmcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeNativeDriver(t *testing.T) *nativeDriver {
	t.Helper()
	home := t.TempDir()
	var err error
	home, err = filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
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
 exec) test -f "$HOME/running" || exit 3; if test -f "$HOME/hold-first-probe"; then rm "$HOME/hold-first-probe"; sleep 30; fi; if test -f "$HOME/guest-script-enabled"; then shift 2; exec "$@"; fi; printf '%s\n' "$HOME"; test "$3" != fail || exit 7;;
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
	child, err := d.Launch(ctx, vm, "/input with space", &bytes.Buffer{})
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
	if strings.Contains(string(raw), "results:") {
		t.Fatalf("writable host results share: %s", raw)
	}
	if !strings.Contains(string(raw), "--no-graphics\n--no-audio\n--no-clipboard\n--dir\ninput:/input with space:ro\n") {
		t.Fatalf("bad shares: %s", raw)
	}
}

func TestNativeLaunchReturnsHandleWhenIdentityUnknown(t *testing.T) {
	d := fakeNativeDriver(t)
	d.identity = func(string) string { return "" }
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	child, err := d.Launch(ctx, ManagedVMName(AdhocLane), "/input", &bytes.Buffer{})
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
	child, err := d.Launch(context.Background(), ManagedVMName(AdhocLane), "/input", &bytes.Buffer{})
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
	child, err := d.Launch(ctx, vm, "/input", &bytes.Buffer{})
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

func TestNativeCloneCancellationConfirmsProcessCleanup(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "block"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := d.Clone(ctx, "pair-conformance-image", ManagedVMName(AdhocLane))
	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrUnconfirmedProcessCleanup) {
		t.Fatalf("cancellation failed to confirm cleanup: %v", err)
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
	handle, err := d.Launch(context.Background(), ManagedVMName(AdhocLane), "/input", writer)
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
	if err := child.process.AwaitTerminal(ctx); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("boot failure did not stop pinned child: %v", err)
	}
	terminal, observeErr := nativeChildTerminal(child.PID())
	for !terminal && observeErr == nil && ctx.Err() == nil {
		time.Sleep(time.Millisecond)
		terminal, observeErr = nativeChildTerminal(child.PID())
	}
	if !terminal || observeErr != nil {
		t.Fatalf("owned launch survived output failure: %v", observeErr)
	}
	if child.cmd.ProcessState != nil {
		t.Fatal("boot error reaped PID generation before Stop")
	}

}

func TestNativeReadinessDoesNotRetryUnconfirmedTransportCleanup(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "running"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	d.own = func(cmd *exec.Cmd) *ownedProcess {
		owner := configureSnapshotCommand(cmd)
		owner.groupPresent = func(int) (bool, error) { return false, errors.New("unavailable process group observation") }
		return owner
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := d.AwaitGuest(ctx, ManagedVMName(AdhocLane))
	if !errors.Is(err, ErrUnconfirmedProcessCleanup) {
		t.Fatalf("lost readiness uncertainty: %v", err)
	}
	raw, _ := os.ReadFile(filepath.Join(d.home, "argv"))
	if strings.Count(string(raw), "exec\n") != 1 {
		t.Fatalf("retried uncertain readiness: %s", raw)
	}
}

func TestNativeClonePreservesUnknownTransportCleanup(t *testing.T) {
	d := fakeNativeDriver(t)
	d.own = func(cmd *exec.Cmd) *ownedProcess {
		owner := configureSnapshotCommand(cmd)
		owner.groupPresent = func(int) (bool, error) { return false, errors.New("unavailable process group observation") }
		return owner
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := d.Clone(ctx, "pair-conformance-image", ManagedVMName(AdhocLane)); !errors.Is(err, ErrUnconfirmedProcessCleanup) {
		t.Fatalf("clone lost cleanup uncertainty: %v", err)
	}
}

func TestNativeResultsExportUsesBoundedLiteralTransport(t *testing.T) {
	d := fakeNativeDriver(t)
	guest := filepath.Join(d.home, "guest with space; literal")
	if err := os.Mkdir(guest, 0700); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{"running": "", "guest-script-enabled": "", "guest with space; literal/manifest.json": `{"version":1,"files":[{"path":"status","size":2}]}`, "guest with space; literal/status": "ok"} {
		if err := os.WriteFile(filepath.Join(d.home, path), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dest := t.TempDir()
	if err := d.ExportResults(ctx, ManagedVMName(AdhocLane), guest, dest, 1024); err != nil {
		t.Fatal(err)
	}
	status, err := os.ReadFile(filepath.Join(dest, "status"))
	if err != nil || string(status) != "ok" {
		t.Fatalf("export %q %v", status, err)
	}
	if err := os.WriteFile(filepath.Join(guest, "status"), []byte("oversized"), 0600); err != nil {
		t.Fatal(err)
	}
	overflowDest := t.TempDir()
	if err := d.ExportResults(ctx, ManagedVMName(AdhocLane), guest, overflowDest, 1024); err == nil {
		t.Fatal("oversized guest result accepted")
	}
	if _, err := os.Stat(filepath.Join(overflowDest, "status")); !os.IsNotExist(err) {
		t.Fatal("oversized status written to host")
	}
}

func TestNativeSlowReadinessProbeDoesNotStartFinalCleanupBudget(t *testing.T) {
	d := fakeNativeDriver(t)
	for _, name := range []string{"running", "hold-first-probe"} {
		if err := os.WriteFile(filepath.Join(d.home, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	bound, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	scope := &cleanupScope{base: bound}
	defer scope.close()
	ctx := context.WithValue(bound, cleanupScopeKey{}, scope)
	if err := d.AwaitGuest(ctx, ManagedVMName(AdhocLane)); err != nil {
		t.Fatal(err)
	}
	scope.mu.Lock()
	started := scope.ctx != nil
	scope.mu.Unlock()
	if started {
		t.Fatal("short readiness probe started final Run cleanup clock")
	}
	raw, _ := os.ReadFile(filepath.Join(d.home, "argv"))
	if strings.Count(string(raw), "exec\n") != 2 {
		t.Fatalf("expected one expired probe and one successful probe: %s", raw)
	}
}

func TestNativeResultsExportRejectsSymlinkCatalogPaths(t *testing.T) {
	for _, kind := range []string{"root", "ancestor", "leaf"} {
		t.Run(kind, func(t *testing.T) {
			d := fakeNativeDriver(t)
			guest := filepath.Join(d.home, "guest")
			if err := os.Mkdir(guest, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d.home, "running"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(d.home, "guest-script-enabled"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			outside := t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("ok"), 0600); err != nil {
				t.Fatal(err)
			}
			manifest := `{"version":1,"files":[{"path":"status","size":2}]}`
			if kind == "ancestor" {
				manifest = `{"version":1,"files":[{"path":"status","size":2},{"path":"diagnostics/zellij-0.log","size":2}]}`
			}
			if err := os.WriteFile(filepath.Join(guest, "manifest.json"), []byte(manifest), 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "leaf" {
				if err := os.Symlink(filepath.Join(outside, "sentinel"), filepath.Join(guest, "status")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(filepath.Join(guest, "status"), []byte("ok"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "ancestor" {
				if err := os.WriteFile(filepath.Join(outside, "zellij-0.log"), []byte("ok"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(guest, "diagnostics")); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "root" {
				alias := filepath.Join(d.home, "guest-alias")
				if err := os.Symlink(guest, alias); err != nil {
					t.Fatal(err)
				}
				guest = alias
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := d.ExportResults(ctx, ManagedVMName(AdhocLane), guest, t.TempDir(), 4096); err == nil {
				t.Fatal("guest symlink catalog path admitted")
			}
		})
	}
}

func TestNativeProvisionPinsBaseAndNeverPrunes(t *testing.T) {
	d := fakeNativeDriver(t)
	if err := os.WriteFile(filepath.Join(d.home, "list"), []byte(`[]`), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := d.Clone(ctx, "ghcr.io/example/base@sha256:"+strings.Repeat("a", 64), ManagedVMName(AdhocLane)); err != nil {
		t.Fatal(err)
	}
	if err := d.Clone(ctx, "ghcr.io/example/base:latest", ManagedVMName(AdhocLane)); err == nil {
		t.Fatal("floating base admitted")
	}
	if got := d.command(ctx, "list").Env; !strings.Contains(strings.Join(got, "\n"), "TART_NO_AUTO_PRUNE=1") {
		t.Fatal("automatic cache pruning enabled")
	}
	if err := d.SaveImage(ctx, "pair-test", "pair-conformance-test"); err == nil {
		t.Fatal("foreign image source admitted")
	}
	for _, state := range []string{"running", "mystery", "stopped"} {
		raw := `[{"Name":"pair-conformance-run-adhoc","State":"` + state + `","Running":false}]`
		if err := os.WriteFile(filepath.Join(d.home, "list"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		err := d.SaveImage(ctx, ManagedVMName(AdhocLane), "pair-conformance-qualified")
		if (err == nil) != (state == "stopped") {
			t.Fatalf("%s: %v", state, err)
		}
	}
}

func TestNativeProvisionCountsUnpublishedCandidates(t *testing.T) {
	d := fakeNativeDriver(t)
	raw := `[{"Name":"pair-conformance-run-adhoc","State":"stopped","Running":false},{"Name":"pair-conformance-orphan-one","State":"stopped","Running":false},{"Name":"pair-conformance-orphan-two","State":"running","Running":true}]`
	if err := os.WriteFile(filepath.Join(d.home, "list"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveImage(context.Background(), ManagedVMName(AdhocLane), "pair-conformance-new"); err == nil {
		t.Fatal("unpublished image population ignored")
	}
	if _, err := os.Stat(filepath.Join(d.home, "cloned")); !os.IsNotExist(err) {
		t.Fatal("cloned above image cap")
	}
}
