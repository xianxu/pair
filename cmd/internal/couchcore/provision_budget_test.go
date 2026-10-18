package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// slowWorktreeAdd delays `git worktree add` like a loaded machine, honoring
// the command's own budget the way OSProvisionIO does; when stderr is set, a
// timed-out add reports it as the command's diagnostic tail.
type slowWorktreeAdd struct {
	*ProvisionFixture
	delay  time.Duration
	stderr string
	budget []time.Duration
}

func (s *slowWorktreeAdd) Run(ctx context.Context, c ProvisionCommand) ([]byte, error) {
	if c.Program == "git" && len(c.Args) > 1 && c.Args[0] == "worktree" && c.Args[1] == "add" {
		s.budget = append(s.budget, c.Timeout)
		timer := time.NewTimer(s.delay)
		defer timer.Stop()
		var deadline <-chan time.Time
		if c.Timeout > 0 {
			deadline = time.After(c.Timeout)
		}
		select {
		case <-timer.C:
		case <-deadline:
			return nil, fmt.Errorf("%w: %s", errors.Join(errors.New("signal: terminated"), context.DeadlineExceeded), s.stderr)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.ProvisionFixture.Run(ctx, c)
}

// TestWorktreeAddUsesWriteBudget: a worktree add slower than the probe budget
// completes, because writing a checkout is not a probe (pair#442).
func TestWorktreeAddUsesWriteBudget(t *testing.T) {
	t.Parallel()
	slow := &slowWorktreeAdd{ProvisionFixture: newProvisionFixture(t)}
	p := NewWorkspaceProvisioner(slow)
	// The other probes run real git, so the probe budget keeps room for a
	// loaded test machine; the add alone is slower than it.
	p.ProbeTimeout, p.WriteTimeout = 2*time.Second, time.Minute
	slow.delay = p.ProbeTimeout + 300*time.Millisecond
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: slow.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.Disposition != "created" || len(slow.budget) != 1 || slow.budget[0] != p.WriteTimeout {
		t.Fatalf("result %+v, worktree add budgets %v", r, slow.budget)
	}
}

// TestFailedCreationIsRecordedAndRetryIsClean: a creation that times out
// leaves a readable record naming the step, command, budget and stderr; the
// next run, with nothing cleaned by hand, creates the slot and drops the
// record (pair#442).
func TestFailedCreationIsRecordedAndRetryIsClean(t *testing.T) {
	t.Parallel()
	slow := &slowWorktreeAdd{ProvisionFixture: newProvisionFixture(t), delay: time.Hour, stderr: "fatal: fixture checkout interrupted"}
	p := NewWorkspaceProvisioner(slow)
	//lint:allow test-shrunk-deadline -- the fake arms this budget only inside worktree-add; assertions pin its persisted 200ms diagnostic and clean retry
	p.WriteTimeout = 200 * time.Millisecond
	_, err := p.Ensure(context.Background(), ProvisionRequest{Path: slow.Primary, Slot: 1})
	var blocked *SlotReconcileError
	if !errors.As(err, &blocked) {
		t.Fatalf("want a blocking reconcile error, got %v", err)
	}
	if !strings.Contains(err.Error(), "step worktree-add") || !strings.Contains(err.Error(), "timed out after 200ms") {
		t.Fatalf("the error does not name the step and budget: %v", err)
	}
	common := slow.git(slow.Primary, "rev-parse", "--path-format=absolute", "--git-common-dir")
	layout := NewSlotLayout(slow.Primary, common, 1)
	rec, err := ReadSlotFailure(layout)
	if err != nil || rec == nil {
		t.Fatalf("no failure record: %v", err)
	}
	if rec.Step != StepWorktreeAdd || rec.Resource != ResourceHost || !rec.TimedOut || rec.Timeout != "200ms" ||
		!strings.HasPrefix(rec.Command, "git worktree add ") || !strings.Contains(rec.Cause, "fatal: fixture checkout interrupted") {
		t.Fatalf("record %+v", rec)
	}
	if _, err := os.Stat(layout.Env()); err != nil {
		t.Fatalf("the failed run should leave its environment, as production does: %v", err)
	}

	slow.delay, p.WriteTimeout = 0, time.Minute // the retry runs real git
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: slow.Primary, Slot: 1})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if r.Disposition != "created" || r.Path != slow.host(1) {
		t.Fatalf("retry result %+v", r)
	}
	if rec, err := ReadSlotFailure(layout); err != nil || rec != nil {
		t.Fatalf("a converged slot keeps its failure record: %+v %v", rec, err)
	}
}

// TestCommandErrorTimedOutIsTheCommandsOwnBudget: only the command's own
// budget expiring reads as its timeout; a caller's deadline does not.
func TestCommandErrorTimedOutIsTheCommandsOwnBudget(t *testing.T) {
	t.Parallel()
	waitForDone := recoveryIOFunc(func(ctx context.Context, c ProvisionCommand) ([]byte, error) {
		if c.Timeout == time.Millisecond {
			return nil, context.DeadlineExceeded // the command's budget expired
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	p := &WorkspaceProvisioner{IO: waitForDone}
	_, err := p.gitWithin(context.Background(), time.Millisecond, "/", nil, "status")
	var cmd *ProvisionCommandError
	if !errors.As(err, &cmd) || !cmd.TimedOut || cmd.Command != "git status" || !errors.Is(err, context.DeadlineExceeded) ||
		!strings.Contains(err.Error(), "git status: timed out after 1ms") {
		t.Fatalf("own budget: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = p.gitWithin(ctx, time.Hour, "/", nil, "status")
	if !errors.As(err, &cmd) || cmd.TimedOut || strings.Contains(err.Error(), "timed out") {
		t.Fatalf("caller deadline read as the command's timeout: %v", err)
	}
}

// TestSlotFailureRecordFitsTheStore: a cause with a full 64 KiB stderr tail
// is still recorded, keeping its head and the end of its stderr, however
// much JSON escaping grows it.
func TestSlotFailureRecordFitsTheStore(t *testing.T) {
	t.Parallel()
	for name, filler := range map[string]string{"plain": "x", "escaped": "\x1b[0m<"} {
		t.Run(name, func(t *testing.T) {
			common, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			layout := NewSlotLayout("/fleet/repo", common, 1)
			stderr := strings.Repeat(filler, (64<<10)/len(filler)) + "fatal: the last line"
			cmdErr := &ProvisionCommandError{Command: "git worktree add", Timeout: time.Second, TimedOut: true, Err: errors.New(stderr)}
			p := &WorkspaceProvisioner{Store: ProvisionStore{}}
			if err := p.recordSlotFailure(layout, &SlotReconcileError{Failure: ReconcileFailure{Resource: ResourceHost, Step: StepWorktreeAdd}, Err: cmdErr}); err != nil {
				t.Fatalf("an oversized cause was not recorded: %v", err)
			}
			rec, rerr := ReadSlotFailure(layout)
			if rerr != nil || rec == nil {
				t.Fatalf("read back: %v", rerr)
			}
			if !strings.HasPrefix(rec.Cause, "git worktree add: timed out after 1s") || !strings.HasSuffix(rec.Cause, "fatal: the last line") {
				t.Fatalf("cause %d bytes lost its head or tail", len(rec.Cause))
			}
		})
	}
}

// TestUnrecordedFailureIsSaid: a failure record that cannot be written is
// named in the caller's error, never dropped silently.
func TestUnrecordedFailureIsSaid(t *testing.T) {
	t.Parallel()
	e := &SlotReconcileError{Address: "repo:1", Repo: "repo", Failure: ReconcileFailure{Resource: ResourceHost, Step: StepWorktreeAdd, Class: FailureRetryable, Cause: "c"},
		RecordErr: errors.New("host creation busy")}
	if !strings.Contains(e.Error(), "slot repo:1: this failure could not be recorded: host creation busy") {
		t.Fatalf("%q", e.Error())
	}
}

// TestBoundCauseIsTotal: any limit yields a cause within it, never a panic.
func TestBoundCauseIsTotal(t *testing.T) {
	t.Parallel()
	cause := strings.Repeat("y", 100)
	for limit := 0; limit <= 120; limit++ {
		got := boundCause(cause, limit)
		if (limit >= len(cause) && got != cause) || (limit < len(cause) && len(got) > limit) {
			t.Fatalf("limit %d: %d bytes", limit, len(got))
		}
	}
}

// TestUnclearedFailureIsWarned: a converged slot whose old record cannot be
// removed says so instead of leaving --show to show a stale failure.
func TestUnclearedFailureIsWarned(t *testing.T) {
	t.Parallel()
	f := newProvisionFixture(t)
	p := NewWorkspaceProvisioner(f)
	if _, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1}); err != nil {
		t.Fatal(err)
	}
	common := f.git(f.Primary, "rev-parse", "--path-format=absolute", "--git-common-dir")
	path := SlotFailurePath(NewSlotLayout(f.Primary, common, 1))
	if err := os.MkdirAll(path, 0o700); err != nil { // not a regular file: Remove refuses it
		t.Fatal(err)
	}
	r, err := p.Ensure(context.Background(), ProvisionRequest{Path: f.Primary, Slot: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Warning, "last failure record could not be cleared") {
		t.Fatalf("warning %q", r.Warning)
	}
}
