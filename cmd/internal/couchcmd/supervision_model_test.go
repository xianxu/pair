package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"
	"time"
)

func TestSupervisionOvernightCompletion(t *testing.T) {
	now := time.Unix(1000, 0)
	b := couchmessage.Binding{Slot: "pair:7", Nonce: "one"}
	s := newSupervisionState()
	s.report(b, "report", now, 3, true)
	f := supervisionFacts{binding: b, known: true, settled: true, quiet: time.Minute, resting: true, submission: 3}
	if fx := s.tick(now.Add(15*time.Minute-time.Nanosecond), []supervisionFacts{f}, true); countSupervision(fx, supervisionNudge) != 0 {
		t.Fatal(fx)
	}
	fx := s.tick(now.Add(15*time.Minute), []supervisionFacts{f}, true)
	if countSupervision(fx, supervisionNudge) != 1 {
		t.Fatal(fx)
	}
	s.nudged(b, "report")
	if fx = s.tick(now.Add(29*time.Minute), []supervisionFacts{f}, true); countSupervision(fx, supervisionNudge)+countSupervision(fx, supervisionEscalate) != 0 {
		t.Fatal(fx)
	}
	fx = s.tick(now.Add(30*time.Minute), []supervisionFacts{f}, false)
	if countSupervision(fx, supervisionEscalate) != 1 {
		t.Fatal(fx)
	}
	if fx = s.tick(now.Add(time.Hour), []supervisionFacts{f}, true); len(fx) != 0 {
		t.Fatal(fx)
	}
	f.submission++
	s.tick(now.Add(time.Hour), []supervisionFacts{f}, true)
	if s.unanswered(b) {
		t.Fatal("operator submission left episode")
	}
}
func countSupervision(fx []supervisionEffect, kind supervisionEffectKind) int {
	n := 0
	for _, f := range fx {
		if f.kind == kind {
			n++
		}
	}
	return n
}

func TestSupervisionSuppressionsAndResets(t *testing.T) {
	now := time.Unix(1000, 0)
	b := couchmessage.Binding{Slot: "pair:7"}
	base := supervisionFacts{binding: b, known: true, settled: true, quiet: time.Minute}
	for _, name := range []string{"unknown", "active", "pending", "resting", "unsettled"} {
		t.Run(name, func(t *testing.T) {
			s := newSupervisionState()
			s.report(b, "one", now, 0, false)
			f := base
			switch name {
			case "unknown":
				f.known = false
			case "active":
				f.quiet = 0
			case "pending":
				f.pending = true
			case "resting":
				f.resting = true
			case "unsettled":
				f.settled = false
			}
			if fx := s.tick(now.Add(time.Hour), []supervisionFacts{f}, true); countSupervision(fx, supervisionNudge)+countSupervision(fx, supervisionEscalate) != 0 {
				t.Fatal(fx)
			}
		})
	}
	s := newSupervisionState()
	s.report(b, "one", now, 0, false)
	s.report(b, "one", now.Add(time.Hour), 0, false)
	if fx := s.tick(now.Add(15*time.Minute), []supervisionFacts{base}, true); countSupervision(fx, supervisionNudge) != 1 {
		t.Fatal("duplicate reset age", fx)
	}
	s.report(b, "two", now.Add(16*time.Minute), 0, false)
	s.nudged(b, "one")
	s.reply(b, now.Add(15*time.Minute))
	if !s.unanswered(b) {
		t.Fatal("old reply cleared new report")
	}
	if fx := s.tick(now.Add(31*time.Minute), []supervisionFacts{base}, true); countSupervision(fx, supervisionNudge) != 1 {
		t.Fatal("old effect suppressed new report", fx)
	}
	s.reply(b, now.Add(17*time.Minute))
	if s.unanswered(b) {
		t.Fatal("reply did not clear")
	}
}

func TestSupervisionStartupOncePerAddress(t *testing.T) {
	s := newSupervisionState()
	now := time.Unix(1000, 0)
	f := supervisionFacts{binding: couchmessage.Binding{Slot: "pair:7", Nonce: "a"}}
	if fx := s.tick(now, []supervisionFacts{f}, false); len(fx) != 0 {
		t.Fatal(fx)
	}
	if fx := s.tick(now, []supervisionFacts{f}, true); countSupervision(fx, supervisionPrompt) != 1 {
		t.Fatal(fx)
	}
	s.tick(now, nil, true)
	f.binding.Nonce = "b"
	if fx := s.tick(now, []supervisionFacts{f}, true); len(fx) != 0 {
		t.Fatal("replacement prompted twice", fx)
	}
	s = newSupervisionState()
	if fx := s.tick(now, []supervisionFacts{f}, true); countSupervision(fx, supervisionPrompt) != 1 {
		t.Fatal(fx)
	}
}
func TestSupervisionCoalescesEarlierWaitNotice(t *testing.T) {
	s := newSupervisionState()
	now := time.Unix(1000, 0)
	b := couchmessage.Binding{Slot: "pair:7"}
	s.report(b, "r", now, 0, false)
	s.nudged(b, "r")
	f := supervisionFacts{binding: b, known: true, settled: true, quiet: time.Hour}
	fx := s.tick(now.Add(30*time.Minute), []supervisionFacts{f}, true)
	if countSupervision(fx, supervisionNudge) != 0 || countSupervision(fx, supervisionEscalate) != 1 {
		t.Fatal(fx)
	}
}
func TestSupervisionCompletionHints(t *testing.T) {
	for _, text := range []string{"#530 landed: PR #123", "Merged as PR #123; back on main", "#530: landed PR #123"} {
		if !completionHint(text) {
			t.Errorf("missed %q", text)
		}
	}
	for _, text := range []string{"Not merged", "#530 not landed", "Will merge tomorrow", "The plan says landed", "> #530 landed", "#530: tests passed\nlanded"} {
		if completionHint(text) {
			t.Errorf("false hint %q", text)
		}
	}
}

func TestSupervisionRepositoryQualifiedCompletion(t *testing.T) {
	if !completionHint("pair#530 landed: PR #123") {
		t.Fatal("qualified issue landing was not recognized")
	}
}

func TestSupervisionSnapshotCannotPruneNewerReport(t *testing.T) {
	now := time.Unix(1000, 0)
	s := newSupervisionState()
	b := couchmessage.Binding{Slot: "pair:7"}
	s.report(b, "new", now.Add(time.Second), 0, true)
	// The tick began before a newly connected worker submitted its report.
	s.tick(now, nil, false)
	if !s.unanswered(b) {
		t.Fatal("older liveness snapshot discarded a newer report")
	}
	s.tick(now.Add(2*time.Second), nil, false)
	if s.unanswered(b) {
		t.Fatal("later disconnected snapshot retained old report")
	}
}

func TestHubSupervisionDelayedHubAvailability(t *testing.T) {
	at := time.Unix(1000, 0)
	a, b := couchmessage.Binding{Slot: "pair:1"}, couchmessage.Binding{Slot: "pair:2"}
	state := newSupervisionState()
	facts := []supervisionFacts{{binding: a, known: true, settled: true, quiet: time.Minute}, {binding: b, known: true, settled: true, quiet: time.Minute}}
	state.report(a, "a", at, 0, false)
	state.report(b, "b", at, 0, false)
	available := map[couchmessage.Binding]bool{a: true, b: false}
	effects := state.tick(at.Add(15*time.Minute), facts, true, available)
	for _, e := range effects {
		if e.binding != a {
			t.Fatalf("offline own hub received effect: %+v", e)
		}
	}
	if state.attempted[b.Slot] {
		t.Fatal("startup prompt consumed before own hub connected")
	}
	available[b] = true
	effects = state.tick(at.Add(16*time.Minute), facts, true, available)
	prompts := 0
	for _, e := range effects {
		if e.kind == supervisionPrompt {
			prompts++
			if e.binding != b {
				t.Fatalf("repeated prompt: %+v", e)
			}
		}
	}
	if prompts != 1 {
		t.Fatalf("late hub prompts: %+v", effects)
	}
}

func TestSupervisionAutomationOffConsumesNudgeWithoutHubAndKeepsEscalation(t *testing.T) {
	now := time.Unix(1000, 0)
	b := couchmessage.Binding{Slot: "pair:7"}
	s := newSupervisionState()
	s.report(b, "report", now, 0, true)
	f := supervisionFacts{binding: b, known: true, settled: true, resting: true, quiet: time.Minute, automationOff: true}
	fx := s.tick(now.Add(31*time.Minute), []supervisionFacts{f}, false)
	if countSupervision(fx, supervisionNudge) != 0 || countSupervision(fx, supervisionEscalate) != 1 || !s.notified(b) {
		t.Fatalf("off did not retire nudge and preserve escalation: %+v", fx)
	}
	f.automationOff = false
	if fx := s.tick(now.Add(32*time.Minute), []supervisionFacts{f}, true); countSupervision(fx, supervisionNudge) != 0 || countSupervision(fx, supervisionEscalate) != 0 {
		t.Fatalf("on replayed off occurrence: %+v", fx)
	}
	s.report(b, "new-report", now.Add(32*time.Minute), 0, true)
	if fx := s.tick(now.Add(47*time.Minute), []supervisionFacts{f}, true); countSupervision(fx, supervisionNudge) != 1 {
		t.Fatalf("new report remained muted: %+v", fx)
	}
}
