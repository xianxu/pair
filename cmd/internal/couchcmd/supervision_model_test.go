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

func TestSupervisionPromptProvenanceSurvivesQueueAndHold(t *testing.T) {
	c := newSupervisionCoordinator(nil, nil)
	now := time.Unix(1000, 0)
	hub := couchmessage.Binding{Slot: "ops:0"}
	c.remember("prompt", hub, now, true)
	// A queue and then an operator hold can exceed one hour before delivery.
	c.remember("next", hub, now.Add(couchmessage.QueueWait+couchmessage.MaxHold), false)
	if !c.systemMessage(couchmessage.Message{ID: "prompt", From: hub}) {
		t.Fatal("delayed prompt lost supervisor provenance")
	}
	if c.systemMessage(couchmessage.Message{ID: "prompt", From: couchmessage.Binding{Slot: "pair:7"}}) {
		t.Fatal("other sender inherited exemption")
	}
}
