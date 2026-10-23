//go:build darwin || linux

package vmcmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestPositionProcessHelper(t *testing.T) {
	root := os.Getenv("PAIR_VM_POSITION_HELPER_ROOT")
	if root == "" {
		return
	}
	p, err := AcquirePosition(context.Background(), root, TrainLane, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	fmt.Fprintln(os.Stdout, "READY")
	io.Copy(io.Discard, os.Stdin)
}
func TestPositionOwnerDeathReleasesKernelLock(t *testing.T) {
	root := filepath.Join(t.TempDir(), "positions")
	ctx, cancel := context.WithTimeout(context.Background(), testwait.Budget(t))
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPositionProcessHelper$")
	cmd.Env = append(os.Environ(), "PAIR_VM_POSITION_HELPER_ROOT="+root)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	ready := make(chan string, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		ready <- fmt.Sprintf("%s|%v", line, err)
	}()
	select {
	case line := <-ready:
		if line != "READY\n|<nil>" {
			t.Fatal(line)
		}
	case <-testwait.After(t):
		t.Fatal("owner never acquired")
	}
	before, err := os.Stat(filepath.Join(root, "train", "position.lock"))
	if err != nil {
		t.Fatal(err)
	}
	progress := &positionProgress{waiting: make(chan struct{})}
	acquired := make(chan *Position, 1)
	errs := make(chan error, 1)
	go func() {
		p, err := AcquirePosition(ctx, root, TrainLane, progress)
		if err != nil {
			errs <- err
			return
		}
		acquired <- p
	}()
	select {
	case <-progress.waiting:
	case err := <-errs:
		t.Fatal(err)
	case <-testwait.After(t):
		t.Fatal("contender never observed owner")
	}
	other, err := AcquirePosition(ctx, root, AdhocLane, nil)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	waited = true
	select {
	case p := <-acquired:
		defer p.Close()
	case err := <-errs:
		t.Fatal(err)
	case <-testwait.After(t):
		t.Fatal("kernel lock survived owner death")
	}
	after, err := os.Stat(filepath.Join(root, "train", "position.lock"))
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("owner death replaced lock inode", err)
	}
}
func TestPositionFailedWritePreservesClaim(t *testing.T) {
	p, _ := positionFixture(t, TrainLane)
	c := positionClaim(TrainLane)
	if err := p.WriteClaim(c); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(p.Dir(), "claim.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Unknown = map[string]json.RawMessage{"huge": json.RawMessage(`"` + strings.Repeat("x", 65536) + `"`)}
	if err := p.WriteClaim(c); err == nil {
		t.Fatal("oversized write accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("oversized write changed claim")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory write permissions")
	}
	c.Unknown = nil
	c.Stage = ClaimLaunched
	c.PID = 11
	if err := os.Chmod(p.Dir(), 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(p.Dir(), 0700)
	if err := p.WriteClaim(c); err == nil {
		t.Fatal("write into readonly directory accepted")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed write changed old bytes")
	}
}
