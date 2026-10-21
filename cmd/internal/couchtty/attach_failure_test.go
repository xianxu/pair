package couchtty

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestAttachFailureSnapshotsBeforeRollback(t *testing.T) {
	for _, exited := range []bool{false, true} {
		t.Run(map[bool]string{false: "unobserved-exit", true: "observed-exit"}[exited], func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
			t.Cleanup(func() { c.Stop(); _ = c.release() })
			raw := append(bytes.Repeat([]byte("x"), couchcore.LaunchFailureOutputLimit+50), []byte("final\xffbytes")...)
			child := ptychild.NewFakeChild(raw)
			t.Cleanup(func() { _ = child.Close() })
			child.Endpoint().EndInput()
			if exited {
				child.Exit(23)
				child.Wait()
			}
			err := c.installObservedThreadActor(context.Background(), "failed", "actor", menuAddress("failed"), "/repo", "repo", child, couchcore.ProcessIdentity{}, true)
			var evidence interface {
				TerminalFailureEvidence() couchcore.LifecycleFailure
			}
			if !errors.As(err, &evidence) {
				t.Fatalf("attach lost typed evidence: %v", err)
			}
			store := couchcore.NewFakeLaunchFailureStore()
			recorder := couchcore.NewLifecycleFailureRecorder(store, time.Now, func() (string, error) { return "operation-owned", nil })
			attempt, beginErr := recorder.Begin("reboot", couchcore.LifecycleFailureTarget{Target: "fixture:3"})
			if beginErr != nil {
				t.Fatal(beginErr)
			}
			_, recordedErr := attempt.Finish(nil, err)
			history, readErr := store.Read()
			if readErr != nil || len(history) != 1 || couchcore.FailureNonce(recordedErr) != "operation-owned" || history[0].Nonce != "operation-owned" || history[0].Stage != "attach-preflight" || !bytes.Equal(history[0].Output, raw[len(raw)-couchcore.LaunchFailureOutputLimit:]) {
				t.Fatalf("actual attach producer did not survive recorder: history=%+v error=%v recorded=%v", history, readErr, recordedErr)
			}
			before := evidence.TerminalFailureEvidence()
			if before.Stage != "attach-preflight" || before.Stream != "pty stdout/stderr" || !bytes.Equal(before.Output, raw[len(raw)-couchcore.LaunchFailureOutputLimit:]) {
				t.Fatalf("missing site/output: %+v", before)
			}
			if !strings.Contains(before.LaunchError, child.Endpoint().ID()) || !strings.Contains(before.LaunchError, "input_ended=true") || !strings.Contains(before.LaunchError, "termination_cause=unavailable") {
				t.Fatalf("missing actual endpoint observation: %q", before.LaunchError)
			}
			if exited && (before.BeforeExit == nil || *before.BeforeExit != 23) || !exited && before.BeforeExit != nil {
				t.Fatalf("invented/lost exit: %+v", before.BeforeExit)
			}
			_ = child.Close()
			before.Output[0] = 'z'
			if before.BeforeExit != nil {
				*before.BeforeExit = 99
			}
			after := evidence.TerminalFailureEvidence()
			if after.Output[0] != 'x' || !exited && after.BeforeExit != nil || exited && *after.BeforeExit != 23 || after.AfterExit != nil {
				t.Fatalf("rollback/caller mutation changed original evidence: %+v", after)
			}
		})
	}
}

func TestAttachFailureValueProjectionOwnsCopies(t *testing.T) {
	cause := errors.New("attach rejected")
	code := 7
	e := &attachFailure{cause: cause, evidence: couchcore.LifecycleFailure{Stage: "attach-precommit", LaunchError: "termination_cause=unavailable", Stream: "pty stdout/stderr", Output: []byte{0xff, 'x'}, BeforeExit: &code}}
	if e.Error() != cause.Error() || e.Unwrap() != cause || !errors.Is(e, cause) {
		t.Fatal("error projection lost cause")
	}
	first := e.TerminalFailureEvidence()
	first.Output[0] = 0
	*first.BeforeExit = 99
	first.Stage = "changed"
	second := e.TerminalFailureEvidence()
	if second.Stage != "attach-precommit" || second.LaunchError != "termination_cause=unavailable" || !bytes.Equal(second.Output, []byte{0xff, 'x'}) || second.BeforeExit == nil || *second.BeforeExit != 7 || second.AfterExit != nil {
		t.Fatalf("projection aliases evidence: %+v", second)
	}
}

func TestAttachFailurePrecommitSnapshotsBeforeRetirement(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	selected := ptychild.NewFakeChild(nil)
	if err := c.presenter.Select(t.Context(), selected.Endpoint(), terminal.Geometry{Cols: 80, Rows: 24}, nil); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	resizeDone := make(chan error, 1)
	go func() {
		resizeDone <- c.presenter.Resize(t.Context(), terminal.Geometry{Cols: 81, Rows: 24}, func(terminal.Geometry) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	child := ptychild.NewFakeChild([]byte("precommit last bytes"))
	done := make(chan error, 1)
	go func() {
		done <- c.installObservedThreadActor(t.Context(), "pending", "actor", menuAddress("pending"), "/repo", "repo", child, couchcore.ProcessIdentity{}, true)
	}()
	// The presenter barrier holds Register while the already-admitted child
	// loses input. It remains unreaped, so this is not the exited-child branch.
	testwait.Until(t, "installer resize before registration", func() bool { return child.Size() == (ptychild.Size{Rows: 23, Cols: 80}) })
	child.Endpoint().EndInput()
	close(release)
	if err := <-resizeDone; err != nil {
		t.Fatal(err)
	}
	err := <-done
	t.Cleanup(func() { c.Stop(); c.workers.Wait(); _ = c.release(); _ = child.Close(); _ = selected.Close() })
	var evidence interface {
		TerminalFailureEvidence() couchcore.LifecycleFailure
	}
	if !errors.As(err, &evidence) {
		t.Fatalf("missing precommit evidence: %v", err)
	}
	f := evidence.TerminalFailureEvidence()
	if f.Stage != "attach-precommit" || string(f.Output) != "precommit last bytes" || f.BeforeExit != nil || f.AfterExit != nil {
		t.Fatalf("incorrect precommit snapshot: %+v", f)
	}
	if err := c.presenter.Retire(t.Context(), child.Endpoint()); err == nil {
		t.Fatal("rejected registration was not retired")
	}
	_ = child.Close()
	if got := evidence.TerminalFailureEvidence(); got.BeforeExit != nil || got.AfterExit != nil {
		t.Fatalf("cleanup exit attributed to attach: %+v", got)
	}
}
