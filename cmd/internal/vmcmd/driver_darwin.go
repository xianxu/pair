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
	return &nativeDriver{executable: executable, home: home, env: []string{"HOME=" + home, "PATH=" + path, "LANG=C", "LC_ALL=C"}, identity: procutil.Identity}
}

func managedVM(name string) bool {
	return name == ManagedVMName(TrainLane) || name == ManagedVMName(AdhocLane)
}
func (d *nativeDriver) command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, d.executable, args...)
	cmd.Env = append([]string(nil), d.env...)
	cmd.Dir = d.home
	configureSnapshotCommand(cmd)
	cmd.WaitDelay = 100 * time.Millisecond
	return cmd
}

// Reuse the snapshot subprocess owner's process-group setup and signal path,
// but confirm group disappearance against the caller's cleanup deadline.
func nativeGroupCleanup(ctx context.Context, cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return confirmOwnedGroupCleanup(ctx, cmd.Cancel, func() (bool, error) { return procutil.ProcessGroupMembers(cmd.Process.Pid) })
}

func (d *nativeDriver) run(ctx context.Context, out, stderr io.Writer, args ...string) (int, error) {
	cmd := d.command(ctx, args...)
	cmd.Stdout, cmd.Stderr = out, stderr
	err := cmd.Run()
	cleanupErr := nativeGroupCleanup(ctx, cmd)
	if cleanupErr != nil {
		cleanupErr = errors.Join(ErrUnconfirmedProcessCleanup, cleanupErr)
	}
	if ctx.Err() != nil {
		return -1, errors.Join(ctx.Err(), cleanupErr)
	}
	if cleanupErr != nil {
		return -1, cleanupErr
	}
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() >= 0 {
		return exit.ExitCode(), nil
	}
	return -1, err
}

func (d *nativeDriver) operation(ctx context.Context, args ...string) error {
	// Bound external diagnostics; never retain an unlimited Tart list/error log.
	inner, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &snapshotOutput{remaining: 4 << 20, cancel: cancel}
	code, err := d.run(inner, out, out, args...)
	if err != nil || code != 0 {
		return fmt.Errorf("Tart %s failed (exit %d): %w", args[0], code, errors.Join(err, fmt.Errorf("%s", out.buffer.String())))
	}
	return nil
}

func (d *nativeDriver) Inspect(ctx context.Context, name string) (VMStatus, error) {
	inner, cancel := context.WithCancel(ctx)
	defer cancel()
	out := &snapshotOutput{remaining: 4 << 20, cancel: cancel}
	var stderr bytes.Buffer
	code, err := d.run(inner, out, &snapshotLimitedWriter{writer: &stderr, remaining: 65536}, "list", "--format", "json")
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
	owner    *nativeDriver
	vm       string
	cmd      *exec.Cmd
	identity string
	done     chan struct{}
}

func (c *nativeChild) PID() int         { return c.cmd.Process.Pid }
func (c *nativeChild) Identity() string { return c.identity }

// A failed boot sink must not strand Tart behind a pipe nobody reads. The
// owner still retains the child handle and confirms stop/join separately.
type launchBootWriter struct {
	writer io.Writer
	stop   func() error
}

func (w launchBootWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return n, errors.Join(err, w.stop())
	}
	return n, nil
}

func (d *nativeDriver) Launch(ctx context.Context, name, input, results string, boot io.Writer) (VMChild, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !managedVM(name) {
		return nil, fmt.Errorf("refusing unmanaged VM launch")
	}
	for _, path := range []string{input, results} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ":\x00") {
			return nil, fmt.Errorf("invalid Tart share path")
		}
	}
	// Caller cancellation must not abandon the VM before host-owned Stop joins it.
	cmd := d.command(context.Background(), "run", "--no-graphics", "--no-audio", "--no-clipboard", "--dir", "input:"+input+":ro", "--dir", "results:"+results, name)
	bootOutput := launchBootWriter{writer: boot, stop: cmd.Cancel}
	cmd.Stdout, cmd.Stderr = bootOutput, bootOutput
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	child := &nativeChild{owner: d, vm: name, cmd: cmd, identity: d.identity(strconv.Itoa(cmd.Process.Pid)), done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(child.done) }()
	if child.identity == "" {
		return child, fmt.Errorf("Tart child %d started without observable start identity", child.PID())
	}
	return child, nil
}

func (d *nativeDriver) AwaitGuest(ctx context.Context, name string) error {
	bound, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		probe, stop := context.WithTimeout(bound, time.Second)
		code, err := d.Execute(probe, name, []string{"/usr/bin/true"}, io.Discard, io.Discard)
		stop()
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
		status, err := d.Inspect(ctx, name)
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
	stopErr := d.operation(grace, "stop", name)
	cancel()
	if err := nativeGroupCleanup(ctx, child.cmd); err != nil {
		return errors.Join(stopErr, err)
	}
	select {
	case <-child.done:
	case <-ctx.Done():
		return errors.Join(stopErr, ctx.Err())
	}
	status, err := d.Inspect(ctx, name)
	if err != nil {
		return errors.Join(stopErr, err)
	}
	if status.Exists && status.State != VMStopped {
		return errors.Join(stopErr, fmt.Errorf("VM %s stop unconfirmed", name))
	}
	return nil
}
func (d *nativeDriver) Delete(ctx context.Context, name string) error {
	if !managedVM(name) {
		return fmt.Errorf("refusing unmanaged VM delete")
	}
	return d.operation(ctx, "delete", name)
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
