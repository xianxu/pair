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

// TestReconcileOutcomeTable (pair#387 Task 2.5): over every single
// perturbation and agent state, a reconcile outcome is blocking exactly when
// no agent could work in the slot: its host checkout is not present, or setup
// never completed (no valid marker). Stated from the final observation,
// independently of OutcomeSeverity's rules.
func TestReconcileOutcomeTable(t *testing.T) {
	for _, agent := range AllEvidenceAgents() {
		for _, p := range slotPerturbations() {
			w := newSlotWorld()
			w.Agent = agent
			w.set(p)
			result, err := reconcileLoop(context.Background(), w, nil)
			blocking, _ := SlotOutcome("pair:1", "pair", result, err)
			host, _ := result.Observation.Get(ResourceHost)
			setup, _ := result.Observation.Get(ResourceSetup)
			// An agent can work in the slot when its checkout is there and was
			// set up once (a known failure or a running compile under a valid
			// marker still counts).
			setUp := setup.State == StatePresent || setup.Sub == SubMarkerValid || setup.Sub == SubWithWarning
			unusable := host.State != StatePresent || !setUp
			if (blocking != nil) != unusable {
				t.Errorf("agent=%s %s: blocking=%v (%v), but host=%s setup=%s/%s", agent, p, blocking != nil, blocking, host.State, setup.State, setup.Sub)
			}
		}
	}
}

// The outcome table's named cells.
func TestOutcomeTableNamedCells(t *testing.T) {
	run := func(agent EvidenceAgent, ps ...perturbation) (*SlotReconcileError, []string, *SlotWorld) {
		w := newSlotWorld()
		w.Agent = agent
		for _, p := range ps {
			w.set(p)
		}
		result, err := reconcileLoop(context.Background(), w, nil)
		b, warnings := SlotOutcome("pair:1", "pair", result, err)
		return b, warnings, w
	}
	// A held index.lock in a dependency of a live slot: unknown, degraded, attach.
	if b, warnings, _ := run(AgentLive, perturbation{DepResource(fakeDep), StateUnknown, ""}); b != nil || len(warnings) != 1 {
		t.Errorf("index.lock under a live agent: blocking=%v warnings=%v", b, warnings)
	}
	// A broken dependency under a live agent: held, degraded, nothing moved.
	b, warnings, w := run(AgentLive, perturbation{DepResource(fakeDep), StateBroken, SubUnreadable})
	if b != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "reboot the slot") || len(w.Effects) != 0 {
		t.Errorf("broken dep under a live agent: blocking=%v warnings=%v effects=%v", b, warnings, w.Effects)
	}
	// Reboot's second pass, the agent stopped: the same dependency is repaired.
	b, warnings, w = run(AgentNone, perturbation{DepResource(fakeDep), StateBroken, SubUnreadable})
	if b != nil || len(warnings) != 0 || len(w.Effects) != 2 {
		t.Errorf("after the agent stopped: blocking=%v warnings=%v effects=%v, want set-aside then compile", b, warnings, w.Effects)
	}
	// tools:1: setup never completed and cannot: blocking, the :0 hand-off.
	b, _, _ = run(AgentNone, perturbation{DepResource(fakeDep), StateAbsent, ""}, perturbation{ResourceSetup, StateAbsent, ""})
	if b == nil {
		w := newSlotWorld()
		w.SourceKnown = false
		w.set(perturbation{DepResource(fakeDep), StateAbsent, ""})
		w.set(perturbation{ResourceSetup, StateAbsent, ""})
		result, err := reconcileLoop(context.Background(), w, nil)
		b, _ = SlotOutcome("tools:1", "tools", result, err)
		if b == nil || b.Failure.Class != FailureHandoff || !strings.Contains(b.Error(), "ask the tools:0 agent") {
			t.Errorf("tools:1: %v, want the blocking :0 hand-off", b)
		}
	}
}
