//go:build darwin

package vmcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type nativeDriver struct {
	executable, home string
	env              []string
	identity         func(string) string
	own              func(*exec.Cmd) *ownedProcess
}

func NewNativeDriver() (VMDriver, error) {
	account, err := user.Current()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(account.HomeDir) || account.HomeDir == "/" {
		return nil, fmt.Errorf("native account HOME unavailable")
	}
	executable, err := exec.LookPath("tart")
	if err != nil {
		return nil, fmt.Errorf("Tart unavailable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return nil, err
	}
	return newNativeDriver(executable, account.HomeDir, os.Getenv("PATH")), nil
}

func newNativeDriver(executable, home, path string) *nativeDriver {
	return &nativeDriver{executable: executable, home: home, env: []string{"HOME=" + home, "PATH=" + path, "LANG=C", "LC_ALL=C"}, identity: procutil.Identity, own: configureSnapshotCommand}
}

func managedVM(name string) bool {
	return name == ManagedVMName(TrainLane) || name == ManagedVMName(AdhocLane)
}
func (d *nativeDriver) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, d.executable, args...)
	cmd.Env = append([]string(nil), d.env...)
	cmd.Dir = d.home
	return cmd
}

func (d *nativeDriver) run(ctx context.Context, out, stderr io.Writer, args ...string) (int, error) {
	return d.runOwned(ctx, nil, out, stderr, args...)
}
func (d *nativeDriver) runOwned(ctx, cleanupBound context.Context, out, stderr io.Writer, args ...string) (int, error) {
	cmd := d.command(ctx, args...)
	owner := d.own(cmd)
	cmd.Stdout, cmd.Stderr = owner.Writer(out), owner.Writer(stderr)
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	observeErr := owner.AwaitTerminal(ctx)
	cleanupCtx := ctx
	stopCleanup := func() {}
	if cleanupBound != nil {
		cleanupCtx = cleanupBound
	} else if ctx.Err() != nil {
		cleanupCtx, stopCleanup = cleanupContext(ctx)
	}
	waitErr, cleanupErr := owner.Finish(cleanupCtx)
	stopCleanup()
	if cleanupErr != nil {
		return -1, errors.Join(observeErr, cleanupErr)
	}
	if observeErr != nil {
		return -1, observeErr
	}
	if waitErr == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(waitErr, &exit) && exit.ExitCode() >= 0 {
		return exit.ExitCode(), nil
	}
	return -1, waitErr
}

func (d *nativeDriver) operation(ctx context.Context, args ...string) error {
	return d.operationOwned(ctx, nil, args...)
}
func (d *nativeDriver) operationOwned(ctx, cleanupBound context.Context, args ...string) error {
	// Bound external diagnostics; never retain an unlimited Tart list/error log.
	inner, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &snapshotOutput{remaining: 4 << 20, cancel: cancel}
	code, err := d.runOwned(inner, cleanupBound, out, out, args...)
	if err != nil || code != 0 {
		return fmt.Errorf("Tart %s failed (exit %d): %w", args[0], code, errors.Join(err, fmt.Errorf("%s", out.buffer.String())))
	}
	return nil
}

func (d *nativeDriver) Inspect(ctx context.Context, name string) (VMStatus, error) {
	return d.inspectOwned(ctx, nil, name)
}
func (d *nativeDriver) inspectOwned(ctx, cleanupBound context.Context, name string) (VMStatus, error) {
	inner, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &snapshotOutput{remaining: 4 << 20, cancel: cancel}
	var stderr bytes.Buffer
	code, err := d.runOwned(inner, cleanupBound, out, &snapshotLimitedWriter{writer: &stderr, remaining: 65536}, "list", "--format", "json")
	if err != nil || code != 0 {
		return VMStatus{}, fmt.Errorf("Tart list unavailable (exit %d): %w", code, errors.Join(err, fmt.Errorf("%s", stderr.String())))
	}
	raw := out.buffer.Bytes()
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return VMStatus{}, err
	}
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '[' {
		return VMStatus{}, fmt.Errorf("Tart list must be an array")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return VMStatus{}, err
	}
	known := map[string]bool{"Name": true, "Running": true, "State": true}
	found := VMStatus{}
	for _, entry := range entries {
		fields, _, err := requiredObject(entry, known)
		if err != nil {
			return VMStatus{}, err
		}
		encoded, err := strictjson.EncodeObject(fields, nil, known)
		if err != nil {
			return VMStatus{}, err
		}
		var item struct {
			Name    string
			Running bool
			State   string
		}
		if err := strictjson.Decode(encoded, &item); err != nil {
			return VMStatus{}, err
		}
		if item.Name != name {
			continue
		}
		if found.Exists {
			return VMStatus{}, fmt.Errorf("ambiguous duplicate Tart VM %q", name)
		}
		found.Exists = true
		switch {
		case item.Running && strings.EqualFold(item.State, "running"):
			found.State = VMRunning
		case !item.Running && strings.EqualFold(item.State, "stopped"):
			found.State = VMStopped
		}
	}
	return found, nil
}

func (d *nativeDriver) Clone(ctx context.Context, image, name string) error {
	if !managedVM(name) || !imageNamePattern.MatchString(image) || managedVM(image) {
		return fmt.Errorf("refusing unmanaged VM clone")
	}
	return d.operation(ctx, "clone", image, name)
}

type nativeChild struct {
	process  *ownedProcess
	owner    *nativeDriver
	vm       string
	cmd      *exec.Cmd
	identity string
	done     chan struct{}
}

func (c *nativeChild) PID() int         { return c.cmd.Process.Pid }
func (c *nativeChild) Identity() string { return c.identity }

func (d *nativeDriver) Launch(ctx context.Context, name, input string, boot io.Writer) (VMChild, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !managedVM(name) {
		return nil, fmt.Errorf("refusing unmanaged VM launch")
	}
	for _, path := range []string{input} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ":\x00") {
			return nil, fmt.Errorf("invalid Tart share path")
		}
	}
	// Caller cancellation must not abandon the VM before host-owned Stop joins it.
	cmd := d.command(context.Background(), "run", "--no-graphics", "--no-audio", "--no-clipboard", "--dir", "input:"+input+":ro", name)
	process := d.own(cmd)
	cmd.Stdout, cmd.Stderr = process.Writer(boot), process.Writer(boot)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	child := &nativeChild{owner: d, vm: name, cmd: cmd, identity: d.identity(strconv.Itoa(cmd.Process.Pid)), process: process, done: process.done}
	if child.identity == "" {
		return child, fmt.Errorf("Tart child %d started without observable start identity", child.PID())
	}
	return child, nil
}

func (d *nativeDriver) AwaitGuest(ctx context.Context, name string) error {
	if !managedVM(name) {
		return fmt.Errorf("invalid VM readiness query")
	}
	bound, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		probe, stop := context.WithTimeout(bound, time.Second)
		code, err := d.runOwned(probe, bound, io.Discard, io.Discard, "exec", name, "/usr/bin/true")
		stop()
		if errors.Is(err, ErrUnconfirmedProcessCleanup) {
			return err
		}
		if code == 0 && err == nil {
			return nil
		}
		select {
		case <-bound.Done():
			return fmt.Errorf("guest-agent readiness: %w", bound.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func (d *nativeDriver) Execute(ctx context.Context, name string, argv []string, out, stderr io.Writer) (int, error) {
	if !managedVM(name) || len(argv) == 0 {
		return -1, fmt.Errorf("invalid VM execution")
	}
	return d.run(ctx, out, stderr, append([]string{"exec", name}, argv...)...)
}

func (d *nativeDriver) Stop(ctx context.Context, name string, handle VMChild) error {
	if !managedVM(name) {
		return fmt.Errorf("refusing unmanaged VM stop")
	}
	if handle == nil {
		status, err := d.inspectOwned(ctx, ctx, name)
		if err != nil {
			return err
		}
		if status.Exists && status.State != VMStopped {
			return fmt.Errorf("VM %s is not confirmed stopped", name)
		}
		return nil
	}
	child, ok := handle.(*nativeChild)
	if !ok || child.owner != d || child.vm != name {
		return fmt.Errorf("VM child is not owned by native driver")
	}
	// Leave part of the caller's bounded cleanup time for forced join/inspection.
	grace, cancel := context.WithTimeout(ctx, time.Second)
	stopErr := d.operationOwned(grace, ctx, "stop", name)
	cancel()
	_, cleanupErr := child.process.Finish(ctx)
	if cleanupErr != nil {
		return errors.Join(stopErr, cleanupErr)
	}
	status, err := d.inspectOwned(ctx, ctx, name)
	if err != nil {
		return errors.Join(stopErr, err)
	}
	if status.Exists && status.State != VMStopped {
		return errors.Join(stopErr, fmt.Errorf("VM %s stop unconfirmed", name))
	}
	if errors.Is(stopErr, ErrUnconfirmedProcessCleanup) {
		return stopErr
	}
	return nil
}
func (d *nativeDriver) Delete(ctx context.Context, name string) error {
	if !managedVM(name) {
		return fmt.Errorf("refusing unmanaged VM delete")
	}
	return d.operationOwned(ctx, ctx, "delete", name)
}
func (d *nativeDriver) Recover(ctx context.Context, claim LaunchClaim) (RecoveryEvidence, error) {
	proof := RecoveryEvidence{PID: claim.PID, Identity: claim.StartIdentity}
	if err := validateLaunchClaim(claim); err != nil {
		return proof, err
	}
	if claim.PID > 0 && claim.StartIdentity != "" {
		err := syscall.Kill(claim.PID, 0)
		if errors.Is(err, syscall.ESRCH) {
			proof.Process = ProcessGone
		} else if err == nil && d.identity(strconv.Itoa(claim.PID)) == claim.StartIdentity {
			proof.Process = ProcessAlive
		}
		present, groupErr := nativeGroupPresent(claim.PID)
		if groupErr != nil {
			return proof, groupErr
		}
		if present {
			proof.Group = ProcessAlive
		} else {
			proof.Group = ProcessGone
		}
	}
	status, err := d.Inspect(ctx, claim.VM)
	if err != nil {
		return proof, err
	}
	proof.VM = status.State
	if !status.Exists {
		proof.VM = VMStopped
	}
	return proof, nil
}

func (d *nativeDriver) ExportResults(ctx context.Context, name, guestRoot, destination string, remaining int64) error {
	if !managedVM(name) {
		return fmt.Errorf("refusing unmanaged VM results export")
	}
	return exportGuestResults(ctx, guestRoot, destination, remaining, func(ctx context.Context, path string, limit int64) ([]byte, bool, error) {
		inner, cancel := context.WithCancel(ctx)
		defer cancel()
		out := &snapshotOutput{remaining: int(limit + 1), cancel: cancel}
		var stderr bytes.Buffer
		code, err := d.runOwned(inner, ctx, out, &snapshotLimitedWriter{writer: &stderr, remaining: 65536}, "exec", name, "/bin/sh", "-c", `path=$1
case "$path" in /*) ;; *) exit 45;; esac
rest=${path#/}
current=
while [ -n "$rest" ]; do
 component=${rest%%/*}
 case "$component" in ''|.|..) exit 45;; esac
 case "$rest" in */*) rest=${rest#*/};; *) rest=;; esac
 current=$current/$component
 if [ -L "$current" ]; then exit 45; fi
done
if [ ! -f "$path" ]; then exit 44; fi
/usr/bin/head -c "$2" "$path"`, "sh", path, strconv.FormatInt(limit+1, 10))
		if err != nil {
			return nil, false, err
		}
		if code == 44 {
			return nil, false, nil
		}
		if code != 0 {
			return nil, false, fmt.Errorf("guest results fetch exit %d: %s", code, stderr.String())
		}
		return append([]byte(nil), out.buffer.Bytes()...), true, nil
	})
}
