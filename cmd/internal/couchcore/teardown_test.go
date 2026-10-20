package couchcore

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/lifetimelease"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// teardownFixture is a slot tree over the fakes: a table whose rows are
// exactly the processes FakeProcOps still lists, and lease holders read from
// the same truth.
type teardownFixture struct {
	env     *testEnv
	thread  ThreadRecord
	rows    []procutil.Process
	holders []int // pids that hold the lifetime lease while alive
}

const teardownTTY = "268435639"

func newTeardownFixture(t *testing.T) *teardownFixture {
	t.Helper()
	env := newTestEnv(t, "/repo")
	f := &teardownFixture{env: env}
	thread := validThreadRecord(t)
	thread.StartingPath, thread.WorkingPath = "/w/pair-slot3/pair", "/w/pair-slot3/pair"
	thread.LatestLaunchProfile = &LaunchProfile{Agent: "claude", Argv: []string{}}
	f.thread = thread
	env.Couch.ProcessTable = func() ([]procutil.Process, error) {
		var out []procutil.Process
		for _, row := range f.rows {
			if env.Proc.Exists(row.PID) == Live {
				out = append(out, row)
			}
		}
		return out, nil
	}
	env.Couch.LeaseHolders = func(context.Context, string) ([]int, error) {
		var out []int
		for _, pid := range f.holders {
			if env.Proc.Exists(pid) == Live {
				out = append(out, pid)
			}
		}
		return out, nil
	}
	return f
}

func (f *teardownFixture) process(pid, ppid int, tty, name string, dies bool) {
	identity := "id-" + name
	f.env.Proc.Set(pid, identity)
	if dies {
		f.env.Proc.DiesOn[pid] = TermSignal
	}
	f.rows = append(f.rows, procutil.Process{PID: pid, PPID: ppid, Identity: identity, Name: name, TTY: tty})
}

// liveTree: the wrapper (a lease holder), the agent on its pty with a tool
// command, and a setsid'd runner the agent started that left the terminal.
func (f *teardownFixture) liveTree() {
	f.process(400, 1, "", "pair", true)
	f.process(500, 400, teardownTTY, "claude", true)
	f.process(501, 500, teardownTTY, "npm", true)
	f.process(502, 500, "", "sdlc", true)
	f.holders = []int{400}
	f.env.Artifacts.SetAgentTerminal(f.thread.Address, AgentTerminal{Process: ProcessIdentity{PID: 500, Identity: "id-claude"}, TTY: teardownTTY})
}

func shortTeardownWaits(t *testing.T) {
	t.Helper()
	term, kill := teardownTermWait, teardownKillWait
	teardownTermWait, teardownKillWait = 100*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { teardownTermWait, teardownKillWait = term, kill })
}

func TestTeardownProvesALiveTreeGoneAndNamesWhatLeft(t *testing.T) {
	f := newTeardownFixture(t)
	f.liveTree()
	proof, err := f.env.Couch.Teardown(context.Background(), f.thread, "op-1")
	if err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	for _, pid := range []int{400, 500, 501} {
		if f.env.Proc.Exists(pid) != Dead {
			t.Errorf("pid %d survived teardown", pid)
		}
	}
	if len(proof.LeftRunning) != 1 || proof.LeftRunning[0].Process.PID != 502 || proof.LeftRunning[0].Operation != "op-1" {
		t.Fatalf("LeftRunning = %+v, want the setsid'd runner", proof.LeftRunning)
	}
	if got := f.env.Proc.Signals[502]; len(got) != 0 {
		t.Fatalf("teardown signalled the detached runner: %v", got)
	}
	if proof.AgentTTY != teardownTTY || proof.provenAt.IsZero() || proof.Address().Key() != "slot:/w/pair-slot3/pair" {
		t.Fatalf("proof = %+v", proof)
	}
}

// On a slot already down teardown signals nothing and still proves.
func TestTeardownIsIdempotentOnADownSlot(t *testing.T) {
	f := newTeardownFixture(t)
	if _, err := f.env.Couch.Teardown(context.Background(), f.thread, ""); err != nil {
		t.Fatalf("Teardown of a down slot: %v", err)
	}
	if len(f.env.Proc.Signals) != 0 {
		t.Fatalf("a down slot was signalled: %v", f.env.Proc.Signals)
	}
}

// A holder that survives SIGKILL is never assumed gone: teardown is
// unproven, names it, and says which phase failed through the details
// #683's recorder reads.
func TestTeardownUnprovenRecordsPhaseAndHolders(t *testing.T) {
	shortTeardownWaits(t)
	f := newTeardownFixture(t)
	f.liveTree()
	delete(f.env.Proc.DiesOn, 501) // the tool command ignores every signal
	_, err := f.env.Couch.Teardown(context.Background(), f.thread, "")
	var details LifecycleFailureDetails
	if !errors.As(err, &details) {
		t.Fatalf("err = %v, want lifecycle failure details", err)
	}
	if details.FailurePhase() != "sweep" || details.FailureOutcome() != "unproven" {
		t.Fatalf("phase %q outcome %q", details.FailurePhase(), details.FailureOutcome())
	}
	holders := details.FailureHolders()
	if len(holders) != 1 || holders[0].Process.PID != 501 {
		t.Fatalf("holders = %+v, want only the process on the agent's terminal", holders)
	}
	got := f.env.Proc.Signals[501]
	if len(got) != 2 || got[0] != TermSignal || got[1] != syscall.SIGKILL {
		t.Fatalf("signals to the holder = %v, want SIGTERM then SIGKILL", got)
	}
	// Through #683's real completion recorder: the failure persists with the
	// phase, the outcome and the named holder, with no capture site in
	// teardown itself.
	f.env.Couch.LaunchFailures = NewFakeLaunchFailureStore()
	attempt, beginErr := f.env.Couch.BeginOperationAttempt("archive", LifecycleFailureTarget{Address: f.thread.Address, StartingPath: f.thread.StartingPath})
	if beginErr != nil {
		t.Fatal(beginErr)
	}
	_, _ = attempt.Finish(nil, err)
	history, snapErr := f.env.Couch.lifecycleFailureRecorder().Snapshot()
	if snapErr != nil || len(history) != 1 {
		t.Fatalf("recorded %+v, %v", history, snapErr)
	}
	if r := history[0]; r.Phase != "sweep" || r.Outcome != "unproven" || len(r.Holders) != 1 || r.Holders[0].Process.PID != 501 || r.Holders[0].Kind != "process" {
		t.Fatalf("recorded failure phase %q outcome %q holders %+v", r.Phase, r.Outcome, r.Holders)
	}
}

// A listing that omits a live server is unknown, never absence (#692): the
// stop refuses and is never retried.
func TestTeardownUnprovenWhenListingOmitsLiveServer(t *testing.T) {
	f := newTeardownFixture(t)
	f.liveTree()
	f.thread.SessionBinding = &couchidentity.SessionBinding{Name: "pair-x", ScopeKey: f.thread.Address.RepoScope, Tag: string(f.thread.Address.Tag), StartNonce: "start", Legacy: true}
	f.env.Artifacts.SetPairSession(f.thread.Address, "pair-x", true)
	f.env.Artifacts.QuiesceNamedErr = launcher.ErrListingOmittedLiveServer
	_, err := f.env.Couch.Teardown(context.Background(), f.thread, "")
	var unprovenErr *ErrTeardownUnproven
	if !errors.As(err, &unprovenErr) || unprovenErr.Phase != "stop" || !errors.Is(err, launcher.ErrListingOmittedLiveServer) {
		t.Fatalf("err = %v, want unproven at stop wrapping the listing omission", err)
	}
}

// ProveDown keeps "start only on a down slot": a survivor is named, never
// signalled.
func TestProveDownNeverStops(t *testing.T) {
	f := newTeardownFixture(t)
	f.liveTree()
	_, err := f.env.Couch.ProveDown(context.Background(), f.thread)
	var unprovenErr *ErrTeardownUnproven
	if !errors.As(err, &unprovenErr) || unprovenErr.Phase != "prove" || len(unprovenErr.Holders) == 0 {
		t.Fatalf("ProveDown of a live slot = %v", err)
	}
	if len(f.env.Proc.Signals) != 0 {
		t.Fatalf("ProveDown signalled: %v", f.env.Proc.Signals)
	}
}

// The proof removes the lease file it proved free, in every mode, since every
// caller retires the record on it; the next launch recreates it.
func TestTeardownRemovesTheProvenLease(t *testing.T) {
	for name, prove := range map[string]func(*Couch, ThreadRecord) (TeardownProof, error){
		"teardown":   func(c *Couch, r ThreadRecord) (TeardownProof, error) { return c.Teardown(context.Background(), r, "") },
		"prove down": func(c *Couch, r ThreadRecord) (TeardownProof, error) { return c.ProveDown(context.Background(), r) },
		"prove and fold": func(c *Couch, r ThreadRecord) (TeardownProof, error) {
			return c.proveDownAndFold(context.Background(), r, "")
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newTeardownFixture(t)
			entry, err := f.env.Couch.lifetimeLeaseEnv(f.thread)
			if err != nil {
				t.Fatal(err)
			}
			path := entry[len(lifetimelease.Env)+1:]
			if _, err := prove(f.env.Couch, f.thread); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("lease after a proof: %v", err)
			}
		})
	}
}

// The published agent is a witness by exact identity even with no terminal
// published (a pre-tty wrapper): teardown stops it, and ProveDown names it
// instead of proving a live agent's slot down.
func TestTeardownWitnessesAnAgentWithoutATerminal(t *testing.T) {
	f := newTeardownFixture(t)
	f.process(500, 1, "", "claude", true)
	f.env.Artifacts.SetAgentTerminal(f.thread.Address, AgentTerminal{Process: ProcessIdentity{PID: 500, Identity: "id-claude"}})
	_, err := f.env.Couch.ProveDown(context.Background(), f.thread)
	var unprovenErr *ErrTeardownUnproven
	if !errors.As(err, &unprovenErr) || len(unprovenErr.Holders) != 1 || unprovenErr.Holders[0].Kind != "agent" {
		t.Fatalf("ProveDown with a live agent and no tty = %v", err)
	}
	if _, err := f.env.Couch.Teardown(context.Background(), f.thread, ""); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if f.env.Proc.Exists(500) != Dead {
		t.Fatal("teardown proved the slot down with its agent alive")
	}
}

// A dead agent's terminal device may already belong to another terminal:
// with the tree down (agent gone, lease free) teardown never sweeps it, and
// says so.
func TestTeardownNeverSweepsAReusedTerminal(t *testing.T) {
	f := newTeardownFixture(t)
	f.process(900, 1, teardownTTY, "vim", true) // someone else's editor on the reused device
	f.env.Artifacts.SetAgentTerminal(f.thread.Address, AgentTerminal{Process: ProcessIdentity{PID: 500, Identity: "id-claude"}, TTY: teardownTTY})
	proof, err := f.env.Couch.Teardown(context.Background(), f.thread, "")
	if err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if len(f.env.Proc.Signals[900]) != 0 || f.env.Proc.Exists(900) != Live {
		t.Fatalf("teardown signalled a process on a reused terminal: %v", f.env.Proc.Signals)
	}
	if len(proof.Notes) == 0 || !strings.Contains(proof.Notes[0], "not checked") {
		t.Fatalf("notes = %q, want the unchecked-terminal note", proof.Notes)
	}
}

// A tool reparented to init after its agent died is still on the agent's
// terminal while the tree is up, so the sweep reaches it by tty, not lineage.
func TestTeardownSweepsReparentedHolders(t *testing.T) {
	f := newTeardownFixture(t)
	f.liveTree()
	f.process(503, 1, teardownTTY, "make", true) // orphaned to init, still on the pty
	if _, err := f.env.Couch.Teardown(context.Background(), f.thread, ""); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if f.env.Proc.Exists(503) != Dead {
		t.Fatal("a reparented process on the agent's terminal survived")
	}
}
