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
	env := newTestEnv(t, "/repo")
	env.Couch.resumeRegistrationTimeout = 150 * time.Millisecond
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	env.Artifacts.SetNativeBinding(parked.Address, "claude", sessioninventory.BindingEstablished, "native-root-1")

	_, _, err := env.Couch.Resume(parked.Address)
	text := errorText(err)
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
			var mu sync.Mutex
			var bornAt time.Time
			env.Artifacts.PaneSidecarsHook = func(address ThreadAddress) error {
				mu.Lock()
				defer mu.Unlock()
				if bornAt.IsZero() && env.Artifacts.PaneQueries() > 0 {
					bornAt = time.Now()
					env.Artifacts.SetPaneSidecar(address, "claude")
				}
				return nil
			}
			env.Artifacts.BeforePairSession = func(ThreadAddress) error {
				mu.Lock()
				defer mu.Unlock()
				if !bornAt.IsZero() && time.Since(bornAt) < 200*time.Millisecond {
					return c.checkErr
				}
				return nil
			}
			_, _, err := env.Couch.Resume(parked.Address)
			text := errorText(err)
			for _, want := range []string{"ownership poll consumed the budget", c.last, "(waited "} {
				if !strings.Contains(text, want) {
					t.Fatalf("error %q lacks %q", text, want)
				}
			}
			polls := regexp.MustCompile(`([0-9]+) ownership polls`).FindStringSubmatch(text)
			if polls == nil || polls[1] == "0" {
				t.Fatalf("error %q lacks a non-zero ownership poll count", text)
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
