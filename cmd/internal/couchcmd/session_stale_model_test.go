package couchcmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

func staleTestObservation() staleReminderObservation {
	return staleReminderObservation{document: sessionsync.DocumentInfo{ID: "doc", File: "/tl.md", Owner: "ops:0", SessionSync: true}, live: couchmessage.SlotLiveness{Binding: couchmessage.Binding{Slot: "ops:0", Nonce: "one"}, Session: 1, Settled: boolp(true), Picker: boolp(false)}, hub: true, known: true}
}

func TestSessionStaleTransitionCooldown(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	observation := staleTestObservation()
	var state staleReminderState
	if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: observation}); !fx.check {
		t.Fatal("no initial check")
	}
	if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: observation}); fx.check {
		t.Fatal("overlapping check")
	}
	fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: now, observation: observation, body: "stale output"})
	if !fx.send {
		t.Fatal("stale result not sent")
	}
	for _, err := range []error{nil, errors.New("uncertain")} {
		advanceStaleReminder(&state, staleReminderEvent{kind: staleSent, at: now, err: err})
		if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now.Add(29 * time.Minute), observation: observation}); fx.check || fx.send {
			t.Fatal("send result refunded allowance")
		}
	}
	if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now.Add(30 * time.Minute), observation: observation}); !fx.check {
		t.Fatal("allowance never renewed")
	}
	if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: now.Add(30 * time.Minute), observation: observation}); fx.send {
		t.Fatal("fresh result sent")
	}
}

func TestSessionStaleTransitionEligibilityAndInvalidation(t *testing.T) {
	for _, change := range []struct {
		name string
		edit func(*staleReminderObservation)
	}{
		{"not session", func(o *staleReminderObservation) { o.document.SessionSync = false }},
		{"operator lock", func(o *staleReminderObservation) { o.document.Holder = sessionsync.HolderOperator }},
		{"not hub", func(o *staleReminderObservation) { o.hub = false }},
		{"busy", func(o *staleReminderObservation) { o.live.Settled = boolp(false) }},
		{"unknown settled", func(o *staleReminderObservation) { o.live.Settled = nil }},
		{"picker", func(o *staleReminderObservation) { o.live.Picker = boolp(true) }},
		{"unknown picker", func(o *staleReminderObservation) { o.live.Picker = nil }},
		{"disconnected", func(o *staleReminderObservation) { o.known = false }},
		{"pending", func(o *staleReminderObservation) { o.pending = true }},
		{"different owner", func(o *staleReminderObservation) { o.document.Owner = "ops:1" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			now := time.Now()
			good := staleTestObservation()
			bad := good
			change.edit(&bad)
			var state staleReminderState
			if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: bad}); fx.check {
				t.Fatal("ineligible check")
			}
			advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: good})
			if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: now, observation: bad, body: "stale"}); fx.send {
				t.Fatal("invalidated result sent")
			}
			if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: good}); !fx.check {
				t.Fatal("invalidated check blocked next eligible tick")
			}
		})
	}
	for _, change := range []func(*staleReminderObservation){
		func(o *staleReminderObservation) { o.live.Session++ },
		func(o *staleReminderObservation) { o.live.Binding.Nonce = "replacement" },
		func(o *staleReminderObservation) { o.document.ID = "new registration" },
		func(o *staleReminderObservation) { o.document.File = "/renamed.md" },
	} {
		now := time.Now()
		good := staleTestObservation()
		changed := good
		change(&changed)
		var state staleReminderState
		advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: good})
		if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: now, observation: changed, body: "stale"}); fx.send {
			t.Fatal("generation changed during check")
		}
	}
}

func TestSessionStaleTransitionFailurePruneAndCancel(t *testing.T) {
	now := time.Now()
	o := staleTestObservation()
	var state staleReminderState
	for i := 0; i < 2; i++ {
		at := now.Add(time.Duration(i) * 30 * time.Minute)
		advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: at, observation: o})
		fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: at, err: errors.New("missing tl-lead")})
		if fx.send || fx.log != (i == 0) {
			t.Fatalf("failure effects %+v", fx)
		}
	}
	advanceStaleReminder(&state, staleReminderEvent{kind: staleRegistry, ids: map[string]bool{}})
	if len(state.next) != 0 {
		t.Fatal("forgotten registrations retained")
	}
	advanceStaleReminder(&state, staleReminderEvent{kind: staleTick, at: now, observation: o})
	advanceStaleReminder(&state, staleReminderEvent{kind: staleCancelled})
	if fx := advanceStaleReminder(&state, staleReminderEvent{kind: staleChecked, at: now, observation: o, body: "stale"}); fx.send || fx.log {
		t.Fatal("effect after cancellation")
	}
}

func TestSessionStaleOutputContract(t *testing.T) {
	for _, raw := range []string{"fresh: every budgeted section\n", "stale: ## Threads (3h old)\nstale: ## Slots (unstamped)\n"} {
		body, err := staleReminderBody("/tl.md", raw)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(raw, "fresh:") && body != "" {
			t.Fatal("fresh sent")
		}
		if body != "" && !strings.HasPrefix(body, sessionsync.PingPrefix) {
			t.Fatal("not identity-excluded ping")
		}
	}
	for _, raw := range []string{"", "garbage", "check: landing unknown\nstale: ## Threads (old)", "stale: ## Threads\x1b[0m", strings.Repeat("x", 9000)} {
		if _, err := staleReminderBody("/tl.md", raw); err == nil {
			t.Fatalf("accepted %q", raw[:min(len(raw), 80)])
		}
	}
}

func FuzzSessionStaleOutput(f *testing.F) {
	f.Add("/tl.md", "stale: ## Threads (unstamped)\n")
	f.Add("/tl.md", "fresh: every budgeted section\n")
	f.Fuzz(func(t *testing.T, path, raw string) {
		body, err := staleReminderBody(path, raw)
		if err == nil && body != "" {
			if err := couchmessage.ValidateBody(body); err != nil {
				t.Fatalf("unsafe body %v", err)
			}
			if !strings.HasPrefix(body, sessionsync.PingPrefix+path+"\n") {
				t.Fatal("path or identity prefix lost")
			}
		}
	})
}

func TestSessionStaleTransitionDiagnosticEpisode(t *testing.T) {
	var s staleReminderState
	for i := 0; i < 2; i++ {
		fx := advanceStaleReminder(&s, staleReminderEvent{kind: staleFault, err: errors.New("inspection failed")})
		if fx.log != (i == 0) {
			t.Fatalf("diagnostic effects %+v", fx)
		}
	}
	o := staleTestObservation()
	advanceStaleReminder(&s, staleReminderEvent{kind: staleTick, at: time.Now(), observation: o})
	advanceStaleReminder(&s, staleReminderEvent{kind: staleChecked, at: time.Now(), observation: o})
	if !advanceStaleReminder(&s, staleReminderEvent{kind: staleFault, err: errors.New("failed again")}).log {
		t.Fatal("success did not clear episode")
	}
}
