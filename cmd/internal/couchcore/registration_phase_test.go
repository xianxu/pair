package couchcore

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// A cold resume's registration wait has two phases: the pane-birth wait, then
// the zellij ownership poll. Its timeout names the phase that consumed the
// budget, both phases' elapsed time, how many ownership polls ran and the last
// check's result, so a failure says where to look (pair#367 smoke test: a
// remote cold resume timed out with only "context deadline exceeded").
func TestColdResumeRegistrationTimeoutNamesThePaneBirthPhase(t *testing.T) {
	t.Parallel()

	env := newTestEnv(t, "/repo")
	env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")

	_, _, err := env.Couch.Resume(parked.Address)
	text := retainedLaunchError(t, env.Couch, err)
	// Every step of the resume is timed, so an earlier slow step cannot
	// hide behind the one that timed out.
	if !regexp.MustCompile(`\[steps: claim [0-9.]+m?s, checkout [0-9.]+m?s, prepare [0-9.]+m?s, spawn [0-9.]+m?s, record\+baseline [0-9.]+m?s, ack [0-9.]+m?s, registration [0-9.]+m?s\]`).MatchString(text) {
		t.Fatalf("error %q lacks the per-step timings", text)
	}
	for _, want := range []string{"pane-birth wait consumed the budget", "0 ownership polls", "(waited "} {
		if !strings.Contains(text, want) {
			t.Fatalf("error %q lacks %q", text, want)
		}
	}
	if !regexp.MustCompile(`pane birth [0-9.]+m?s`).MatchString(text) {
		t.Fatalf("error %q lacks the pane-birth elapsed time", text)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the phase report lost the deadline cause: %v", err)
	}
}

func TestColdResumeRegistrationTimeoutNamesTheOwnershipPhase(t *testing.T) {
	for _, c := range []struct {
		name     string
		checkErr error
		last     string
	}{
		{"never owned", nil, "last ownership check: not owned"},
		{"check failing", errors.New("zellij list-panes timed out"), "last ownership check: zellij list-panes timed out"},
	} {
		t.Run(c.name, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
			parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")
			// The pane is born on the first wait poll; the session never
			// proves owned (or its check keeps failing) from then on.
			// The failing check lasts as long as the wait does: afterwards the
			// post-ack cleanup needs an answer before it may retire the start.
			// The cleanup's first act is to time its TERM through Couch.after,
			// so that call, not a wall-clock guess, ends the fault (#480).
			var mu sync.Mutex
			var born, cleaning bool
			env.Artifacts.PaneSidecarsHook = func(address ThreadAddress) error {
				mu.Lock()
				defer mu.Unlock()
				if !born && env.Artifacts.PaneQueries() > 0 {
					born = true
					env.Artifacts.SetPaneSidecar(address, "claude")
				}
				return nil
			}
			// The budget is spent when the test says: on the second ownership
			// check, so exactly one completed poll precedes it however slowly
			// the launch before it ran.
			budget := newSpendableBudget()
			env.Couch.registrationDeadline = func(parent context.Context, _ time.Duration) (context.Context, context.CancelFunc) {
				return budget.bind(parent)
			}
			checks := 0
			env.Couch.after = func(d time.Duration) <-chan time.Time {
				mu.Lock()
				cleaning = true
				mu.Unlock()
				return time.After(d)
			}
			env.Artifacts.BeforePairSession = func(ThreadAddress) error {
				mu.Lock()
				defer mu.Unlock()
				if born && !cleaning {
					if checks++; checks == 2 {
						budget.spend()
					}
					return c.checkErr
				}
				return nil
			}
			_, _, err := env.Couch.Resume(parked.Address)
			text := retainedLaunchError(t, env.Couch, err)
			for _, want := range []string{"ownership poll consumed the budget", c.last, "(waited "} {
				if !strings.Contains(text, want) {
					t.Fatalf("error %q lacks %q", text, want)
				}
			}
			if !strings.Contains(text, " 1 ownership polls;") {
				t.Fatalf("error %q lacks the one ownership poll that ran", text)
			}
			if !regexp.MustCompile(`ownership poll [0-9.]+m?s`).MatchString(text) {
				t.Fatalf("error %q lacks the ownership-poll elapsed time", text)
			}
		})
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// spendableBudget is a deadline the test spends: its context
// reports DeadlineExceeded, as a lapsed WithTimeout would, once spend runs.
type spendableBudget struct {
	once  sync.Once
	spent chan struct{}
}

func newSpendableBudget() *spendableBudget { return &spendableBudget{spent: make(chan struct{})} }

func (b *spendableBudget) spend() { b.once.Do(func() { close(b.spent) }) }

func (b *spendableBudget) bind(parent context.Context) (context.Context, context.CancelFunc) {
	return &budgetContext{parent, b.spent, time.Now().Add(time.Hour)}, func() {}
}

// budgetContext is its parent with a deadline that lapses when spent closes.
// The parent here is never cancelled, so Done need not merge it. The pane-birth
// wait demands a Deadline to size its own bound; a distant one leaves spend
// the only thing that ends the wait.
type budgetContext struct {
	context.Context
	spent chan struct{}
	far   time.Time
}

func (c *budgetContext) Deadline() (time.Time, bool) { return c.far, true }

func (c *budgetContext) Done() <-chan struct{} { return c.spent }

func (c *budgetContext) Err() error {
	select {
	case <-c.spent:
		return context.DeadlineExceeded
	default:
		return c.Context.Err()
	}
}

// Timing detail moved from the one-line operator error into retained diagnostics.
func retainedLaunchError(t *testing.T, c *Couch, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected launch error")
	}
	records, readErr := c.LaunchFailures.Read()
	if readErr != nil || len(records) != 1 {
		t.Fatalf("retained evidence: %v %+v", readErr, records)
	}
	return records[0].LaunchError
}
