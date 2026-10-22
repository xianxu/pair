package couchcore

import (
	"io"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

type helperBudgetAck struct {
	io.Reader
	closed bool
	close  func()
}

type helperBudgetDrainingAck struct {
	*io.PipeReader
	finished chan struct{}
}

func (a *helperBudgetDrainingAck) Read(p []byte) (int, error) {
	n, err := a.PipeReader.Read(p)
	// Make joining the read observable independently of closing its pipe.
	// Observe this timer in testing/synctest virtual time; no wall-clock guess.
	<-time.After(time.Second)
	close(a.finished)
	return n, err
}

func (a *helperBudgetAck) Close() error {
	a.closed = true
	if a.close != nil {
		a.close()
	}
	return nil
}

func TestLaunchHelperStartupDeadlineRejectsBeforeExec(t *testing.T) {
	for _, raw := range []string{"invalid", "", startupbudget.Encode(time.Now().Add(-time.Second)), "0001-01-01T00:00:00Z"} {
		t.Run(raw, func(t *testing.T) {
			ack := &helperBudgetAck{Reader: strings.NewReader(string([]byte{launchAckByte}))}
			called := false
			err := RunLaunchHelper(ack, time.Minute, []string{"target"}, []string{startupbudget.DeadlineEnv + "=" + raw}, func([]string, []string) error { called = true; return nil })
			if err == nil || called || !ack.closed {
				t.Fatalf("err=%v exec=%v closed=%v", err, called, ack.closed)
			}
		})
	}
}

func TestLaunchHelperStartupDeadlineCapsDelayedAcknowledgement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reader, writer := io.Pipe()
		defer writer.Close()
		ack := &helperBudgetDrainingAck{PipeReader: reader, finished: make(chan struct{})}
		started := time.Now()
		deadline := started.Add(3 * time.Second)
		done := make(chan error, 1)
		called := false
		go func() {
			done <- RunLaunchHelper(ack, time.Minute, []string{"target"}, []string{startupbudget.DeadlineEnv + "=" + startupbudget.Encode(deadline)}, func([]string, []string) error { called = true; return nil })
		}()
		synctest.Wait()
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		<-time.After(4 * time.Second)
		synctest.Wait()
		select {
		case err := <-done:
			if err == nil || called {
				t.Fatalf("err=%v exec=%v", err, called)
			}
			select {
			case <-ack.finished:
			default:
				t.Fatal("helper returned without joining acknowledgement reader")
			}
		default:
			t.Fatal("helper extended original deadline while awaiting acknowledgement")
		}
	})
}

func TestLaunchHelperStartupDeadlineCheckedAfterAcknowledgementClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deadline := time.Now().Add(time.Second)
		// Observe this timer in testing/synctest virtual time; no wall-clock guess.
		ack := &helperBudgetAck{Reader: strings.NewReader(string([]byte{launchAckByte})), close: func() { <-time.After(2 * time.Second) }}
		called := false
		err := RunLaunchHelper(ack, time.Minute, []string{"target"}, []string{startupbudget.DeadlineEnv + "=" + startupbudget.Encode(deadline)}, func([]string, []string) error { called = true; return nil })
		if err == nil || called {
			t.Fatalf("err=%v exec=%v", err, called)
		}
	})
}

func TestLaunchHelperStartupDeadlinePreservesEnvironment(t *testing.T) {
	raw := startupbudget.Encode(time.Now().Add(time.Minute))
	env := []string{"A=B", startupbudget.DeadlineEnv + "=" + raw}
	ack := &helperBudgetAck{Reader: strings.NewReader(string([]byte{launchAckByte}))}
	err := RunLaunchHelper(ack, time.Minute, []string{"target"}, env, func(_ []string, got []string) error {
		if len(got) != len(env) || got[1] != env[1] {
			t.Fatalf("lost startup deadline: %v", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
