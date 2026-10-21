package launcher

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"
)

func TestZellijKnownBugReadRecoversEmptyAndCrossedReplies(t *testing.T) {
	for _, tc := range []struct {
		name          string
		args          []string
		good, crossed string
	}{
		{"panes", []string{"--session", "pair-a", "action", "list-panes", "--json", "--command"}, "[]\n", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"},
		{"clients", []string{"--session", "pair-a", "action", "list-clients"}, "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", "[]\n"},
		{"sessions", []string{"list-sessions", "--no-formatting"}, "pair-a [Created 0s ago]\n", "[]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEmptyReplyFake(t, emptyReplyFrame{Args: tc.args}, emptyReplyFrame{Args: tc.args, Stdout: tc.crossed}, emptyReplyFrame{Args: tc.args, Stdout: tc.good})
			var events []ZellijQueryObservation
			ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { events = append(events, e) })
			out, err := f.source.runContext(ctx, tc.args...)
			if err != nil || string(out) != tc.good || f.calls(t) != 3 {
				t.Fatalf("reply=%q err=%v calls=%d", out, err, f.calls(t))
			}
			if len(events) != 3 || !errors.Is(events[0].Err, ErrZellijNoReply) || events[1].Err == nil || events[2].Err != nil {
				t.Fatalf("attempt evidence=%+v", events)
			}
			for _, event := range events {
				if !event.EffectiveDeadline.Equal(events[0].EffectiveDeadline) {
					t.Fatal("attempt refreshed deadline")
				}
			}
		})
	}
}

func TestZellijKnownBugReadBoundAndNonRetryableResults(t *testing.T) {
	args := []string{"--session", "pair-a", "action", "list-panes", "--json", "--command"}
	for _, tc := range []struct {
		name, raw   string
		exit, calls int
		args        []string
	}{
		{"empty", "", 0, 3, args}, {"crossed", "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", 0, 3, args},
		{"malformed", "[{", 0, 1, args}, {"invalid-pane", "[{\"id\":0}]", 0, 1, args},
		{"nonzero", "", 7, 1, args}, {"mutation", "", 0, 1, []string{"--session", "pair-a", "action", "close-pane"}},
		{"unknown-flags", "", 0, 1, []string{"--session", "pair-a", "action", "list-panes", "--json", "--future"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := make([]emptyReplyFrame, tc.calls)
			for i := range frames {
				frames[i] = emptyReplyFrame{Args: tc.args, Stdout: tc.raw, Exit: tc.exit}
			}
			f := newEmptyReplyFake(t, frames...)
			_, err := f.source.runContext(t.Context(), tc.args...)
			if f.calls(t) != tc.calls {
				t.Fatalf("calls=%d want=%d", f.calls(t), tc.calls)
			}
			if tc.calls == 3 && err == nil {
				t.Fatal("exhausted invalid replies granted authority")
			}
		})
	}
}

func TestZellijKnownBugReadCancellationStopsBeforeNextAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	source := ZellijSource{queryRun: func(cmd *exec.Cmd) error { calls++; cancel(); return nil }}
	_, err := source.runContext(ctx, "list-sessions", "--no-formatting")
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestZellijKnownBugReadWorkIsBoundedLinearly(t *testing.T) {
	for _, n := range []int{8, 16} {
		calls := 0
		source := ZellijSource{queryRun: func(cmd *exec.Cmd) error {
			calls++
			if calls%3 == 0 {
				_, _ = io.WriteString(cmd.Stdout, "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n")
			}
			return nil
		}}
		for range n {
			if _, err := source.ClientCountContext(t.Context(), "pair-a"); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 3*n {
			t.Fatalf("%d reads used %d processes", n, calls)
		}
	}
}
