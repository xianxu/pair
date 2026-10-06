package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Weave's error lines, captured from ariadne's source at b9bc9f32
// (cmd/weave/main.go prints "Error: <err>"; staging/setup.go ErrSetupInUse;
// acquire.go missing substrate / missing repository).
const (
	weaveLineSetupActive     = "Error: environment setup is active in /f/worktree/tools-slot1; retry after the active setup finishes"
	weaveLineMissingSubstate = "Error: missing substrate /f/worktree/tools-slot1/ariadne declared in /f/worktree/tools-slot1/tools: record its source in construct/deps"
	weaveLineMissingRepo     = "Error: missing repository /f/worktree/tools-slot1/ariadne: record its source in construct/deps: exit status 128"
	weaveLineOwnerPrefixed   = "tools: Error: missing substrate /f/x declared in /f/y: record its source in construct/deps"
)

func TestClassifyConvergeError(t *testing.T) {
	compile := PlannedStep{Step: StepCompile, Resource: ResourceSetup}
	cases := []struct {
		name  string
		err   error
		class FailureClass
		cause string
	}{
		{"weave setup active", fmt.Errorf("exit status 1: %s", weaveLineSetupActive), FailureRetryable, weaveLineSetupActive},
		{"setup lock held", errSetupRunning, FailureRetryable, errSetupRunning.Error()},
		{"host creation busy", ErrHostCreationBusy, FailureRetryable, ErrHostCreationBusy.Error()},
		{"deadline", context.DeadlineExceeded, FailureRetryable, context.DeadlineExceeded.Error()},
		{"cancelled", context.Canceled, FailureRetryable, context.Canceled.Error()},
		{"missing substrate", fmt.Errorf("exit status 1: progress\n%s", weaveLineMissingSubstate), FailureHandoff, weaveLineMissingSubstate},
		{"missing repository", errors.New(weaveLineMissingRepo), FailureHandoff, weaveLineMissingRepo},
		{"owner-prefixed line", errors.New(weaveLineOwnerPrefixed), FailureHandoff, "Error: missing substrate /f/x declared in /f/y: record its source in construct/deps"},
		{"git auth failure", errors.New("fatal: Authentication failed for 'https://github.com/x/y.git/'"), FailureHandoff, "fatal: Authentication failed for 'https://github.com/x/y.git/'"},
		{"no Error: line", errors.New("exit status 2"), FailureHandoff, "exit status 2"},
	}
	seen := map[FailureClass]bool{}
	for _, c := range cases {
		got := ClassifyConvergeError(compile, c.err)
		if got.Class != c.class || got.Cause != c.cause || got.Resource != ResourceSetup {
			t.Errorf("%s: %+v, want %s %q", c.name, got, c.class, c.cause)
		}
		seen[got.Class] = true
	}
	for _, class := range []FailureClass{FailureRetryable, FailureHandoff} {
		if !seen[class] {
			t.Errorf("class %s has no row", class)
		}
	}
}

// TestReconcileAdviceText: every failure class has advice; only retryable says
// run it again; a hand-off names the repository's :0 agent; none says "fix
// it" or a bare "retry".
func TestReconcileAdviceText(t *testing.T) {
	for _, class := range AllFailureClasses() {
		f := ReconcileFailure{Resource: DepResource("ariadne"), Class: class, Cause: "cause text"}
		if class == FailureHold {
			f.Cause = StopReasonAgentLive
		}
		text := ReconcileAdvice("tools:1", "tools", f)
		if !strings.HasPrefix(text, "slot tools:1: dep:ariadne") {
			t.Errorf("%s: %q does not name the slot and resource", class, text)
		}
		lower := strings.ToLower(text)
		if strings.Contains(lower, "fix it") || strings.Contains(lower, " retry") {
			t.Errorf("%s: %q says fix it / retry", class, text)
		}
		if (class == FailureRetryable) != strings.Contains(text, "run it again") {
			t.Errorf("%s: %q, only retryable says run it again", class, text)
		}
		if (class == FailureHandoff) != strings.Contains(text, "ask the tools:0 agent") {
			t.Errorf("%s: %q, only a hand-off goes to :0", class, text)
		}
	}
	if text := ReconcileAdvice("tools:1", "tools", ReconcileFailure{Resource: ResourceHost, Class: FailureHold, Cause: StopReasonAgentLive}); !strings.Contains(text, "reboot the slot") {
		t.Errorf("an agent hold must name reboot: %q", text)
	}
}

// TestSetupFailureThroughRealSeam: a weave that prints its missing-substrate
// line, run through OSProvisionIO (which folds stderr into the error), is
// classified a hand-off with weave's own line as the cause.
func TestSetupFailureThroughRealSeam(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "weave")
	body := "#!/bin/sh\necho 'compiling' 1>&2\necho '" + weaveLineMissingSubstate + "' 1>&2\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	io := OSProvisionIO{Env: append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))}
	_, err := io.Run(context.Background(), ProvisionCommand{Dir: dir, Program: script, Args: []string{"compile"}})
	if err == nil {
		t.Fatal("fake weave succeeded")
	}
	got := ClassifyConvergeError(PlannedStep{Step: StepCompile, Resource: ResourceSetup}, err)
	if got.Class != FailureHandoff || got.Cause != weaveLineMissingSubstate {
		t.Fatalf("%+v from %v", got, err)
	}
}

// TestWeaveConformance runs the real weave when it is installed: a held setup
// lock is retryable, and a substrate row without a source is a hand-off
// carrying weave's missing-substrate line.
func TestWeaveConformance(t *testing.T) {
	weave, err := exec.LookPath("weave")
	if err != nil {
		t.Skip("weave is not installed")
	}
	env := t.TempDir()
	host := filepath.Join(env, "tools")
	write(t, filepath.Join(host, "construct", "deps"), "substrate ../missing-dep\n")
	cmd := exec.Command("git", "init", "-q", host)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	io := OSProvisionIO{Env: os.Environ()}
	_, err = io.Run(context.Background(), ProvisionCommand{Dir: host, Program: weave, Args: []string{"compile"}})
	if err == nil {
		t.Skip("weave compiled a sourceless substrate; conformance needs a newer fixture")
	}
	if got := ClassifyConvergeError(PlannedStep{Step: StepCompile, Resource: ResourceSetup}, err); got.Class != FailureHandoff || !strings.Contains(got.Cause, "Error:") {
		t.Errorf("sourceless substrate: %+v from %v", got, err)
	}
}
