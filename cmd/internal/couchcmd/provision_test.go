package couchcmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

type readinessFixture struct {
	calls         []couchcore.ProvisionRequest
	ready         bool
	fail          bool
	cancellable   bool
	waitForCancel bool
}

func (f *readinessFixture) Ensure(ctx context.Context, req couchcore.ProvisionRequest) (couchcore.ProvisionResult, error) {
	f.calls = append(f.calls, req)
	f.cancellable = ctx.Done() != nil
	fmt.Fprintln(req.Progress, "preparing workspace")
	if f.waitForCancel {
		<-ctx.Done()
		return couchcore.ProvisionResult{}, ctx.Err()
	}
	if f.fail {
		return couchcore.ProvisionResult{}, errors.New("setup failed")
	}
	disposition := "created"
	if f.ready {
		disposition = "reused"
	}
	f.ready = true
	return couchcore.ProvisionResult{
		SchemaVersion: 1, Address: "repo:1", Path: "/fleet/worktree/repo-slot1/repo",
		RestingBranch: "main-slot1", BaselineSHA: strings.Repeat("a", 40), Disposition: disposition,
	}, nil
}

type provisionRT struct {
	testRT
	readiness couchcore.WorkspaceReadiness
}

func (r provisionRT) NewCouchWith(runner couchcore.Runner, namespace couchcore.CouchNamespace) (*couchcore.Couch, error) {
	c, err := r.testRT.NewCouchWith(runner, namespace)
	if err == nil {
		c.Workspaces = r.readiness
	}
	return c, err
}
func TestProvisionCLI(t *testing.T) {
	fixture := &readinessFixture{}
	rt := provisionRT{newRT(t), fixture}
	for _, want := range []string{"created", "reused"} {
		var out, diag bytes.Buffer
		code := RunWithRuntime([]string{"--internal", "provision-workspace", "/fleet/repo", "--slot=1", "--remote=upstream"}, strings.NewReader(""), &out, &diag, rt)
		if code != 0 {
			t.Fatalf("code=%d stderr=%s", code, &diag)
		}
		var result map[string]any
		decoder := json.NewDecoder(&out)
		if err := decoder.Decode(&result); err != nil {
			t.Fatal(err)
		}
		if result["disposition"] != want || result["schema_version"] != float64(1) {
			t.Fatalf("result=%v", result)
		}
		if err := decoder.Decode(&result); err != io.EOF {
			t.Fatalf("extra stdout: %v", err)
		}
		if !strings.Contains(diag.String(), "preparing workspace") {
			t.Fatalf("stderr=%s", &diag)
		}
	}
	if len(fixture.calls) != 2 || fixture.calls[0].Path != "/fleet/repo" || fmt.Sprint(fixture.calls[0].Slot) != "1" || fixture.calls[0].Remote != "upstream" {
		t.Fatalf("calls=%+v", fixture.calls)
	}
	if !fixture.cancellable {
		t.Fatal("CLI did not supply cancellable context")
	}
	if rt.supervisor.acquired != 0 || len(rt.registryRecords(t)) != 0 || len(rt.runner.Ops) != 0 {
		t.Fatal("provisioning acquired supervisor or registered an actor")
	}
	fixture.fail = true
	var out, diag bytes.Buffer
	if code := RunWithRuntime([]string{"--internal", "provision-workspace", "/fleet/repo", "--slot=2"}, strings.NewReader(""), &out, &diag, rt); code == 0 || out.Len() != 0 || !strings.Contains(diag.String(), "setup failed") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, &out, &diag)
	}
}
func TestProvisionCLIRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"/repo"}, {"/repo", "--slot"}, {"/repo", "--slot="}, {"/repo", "--slot=01"}, {"/repo", "--slot=0"}, {"/repo", "--slot=1", "--remote"}, {"/repo", "--slot=1", "--retry"}, {"--slot=1"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			fixture := &readinessFixture{}
			rt := provisionRT{newRT(t), fixture}
			var out, diag bytes.Buffer
			if code := RunWithRuntime(append([]string{"--internal", "provision-workspace"}, args...), strings.NewReader(""), &out, &diag, rt); code == 0 || len(fixture.calls) != 0 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, len(fixture.calls), &diag)
			}
		})
	}
}

// Exercise actual SIGTERM registration in a subprocess so the test runner's
// signal handlers and sibling tests are never changed by the test driver.
func TestProvisionCLICancellation(t *testing.T) {
	if os.Getenv("PAIR_PROVISION_SIGNAL_HELPER") == "1" {
		fixture := &readinessFixture{waitForCancel: true}
		rt := provisionRT{newRT(t), fixture}
		var out bytes.Buffer
		code := RunWithRuntime([]string{"--internal", "provision-workspace", "/repo", "--slot=1"}, strings.NewReader(""), &out, os.Stderr, rt)
		if code == 0 || out.Len() != 0 || !fixture.cancellable {
			t.Fatal("cancellation returned success or produced a result")
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProvisionCLICancellation$")
	cmd.Env = append(os.Environ(), "PAIR_PROVISION_SIGNAL_HELPER=1")
	pipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() || scanner.Text() != "preparing workspace" {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("missing setup barrier: %s", scanner.Text())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal(err)
	}
	var diagnostics strings.Builder
	for scanner.Scan() {
		diagnostics.WriteString(scanner.Text())
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper: %v stdout=%s stderr=%s", err, &output, &diagnostics)
	}
	if !strings.Contains(diagnostics.String(), "context canceled") {
		t.Fatalf("stderr=%s", &diagnostics)
	}
}

// Exercise the CLI wiring retained by subsequent menu operations, then the
// real command runner used for weave/Homebrew setup. A console owns all output.
func TestConsoleProvisionOutputDoesNotBypassRenderer(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%v", fail), func(t *testing.T) {
			rt := newRT(t, "/repo")
			var stdout, stderr bytes.Buffer
			finished := false
			finish := func(console *couchtty.Console, c *couchcore.Couch, _ couchcore.StartResult, _ io.Writer) int {
				finished = true
				if console == nil {
					t.Fatal("missing console")
				}
				command := "printf '\033[2Jbrew update\n'; printf 'weave build diagnostics\n' >&2"
				if fail {
					command += "; exit 7"
				}
				_, err := (couchcore.OSProvisionIO{}).Run(context.Background(), couchcore.ProvisionCommand{
					Program: "sh", Args: []string{"-c", command}, Progress: c.WorkspaceProgress, StreamOutput: true,
				})
				if fail && (err == nil || !strings.Contains(err.Error(), "weave build diagnostics")) {
					t.Fatalf("lost failure diagnostic: %v", err)
				}
				if !fail && err != nil {
					t.Fatal(err)
				}
				return 0
			}
			op, _ := Resolve("start")
			code := runTypedOperationWithConsole(op, map[string]string{}, map[string]string{"path": "/repo"}, true, "", nil, nil, strings.NewReader(""), &stdout, &stderr, rt, finish)
			if code != 0 || !finished {
				t.Fatalf("console not reached: code %d, stderr %s", code, &stderr)
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("setup bypassed renderer: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}
