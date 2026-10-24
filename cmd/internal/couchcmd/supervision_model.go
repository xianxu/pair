package couchcmd

import (
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

const supervisionNudgeAfter = 15 * time.Minute
const supervisionEscalateAfter = 30 * time.Minute

type supervisionEffectKind uint8

const (
	supervisionPrompt supervisionEffectKind = iota
	supervisionNudge
	supervisionEscalate
)

type supervisionEffect struct {
	kind    supervisionEffectKind
	binding couchmessage.Binding
	report  string
	age     time.Duration
}
type supervisionFacts struct {
	binding                                 couchmessage.Binding
	known, settled, resting, pending, muted bool
	automationOff                           bool
	quiet                                   time.Duration
	submission                              uint64
}
type reportEpisode struct {
	id                              string
	at                              time.Time
	submission                      uint64
	completion, notified, escalated bool
}

// supervisionState owns only volatile facts. Its methods are pure transitions;
// the coordinator serializes them and performs effects after releasing its lock.
type supervisionState struct {
	reports   map[couchmessage.Binding]reportEpisode
	attempted map[string]bool
}

func newSupervisionState() *supervisionState {
	return &supervisionState{reports: map[couchmessage.Binding]reportEpisode{}, attempted: map[string]bool{}}
}
func (s *supervisionState) report(b couchmessage.Binding, id string, at time.Time, submission uint64, completion bool) {
	if old, ok := s.reports[b]; ok && (old.id == id || at.Before(old.at)) {
		return
	}
	s.reports[b] = reportEpisode{id: id, at: at, submission: submission, completion: completion}
}
func (s *supervisionState) reply(b couchmessage.Binding, initiated time.Time) {
	if r, ok := s.reports[b]; ok && !initiated.Before(r.at) {
		delete(s.reports, b)
	}
}
func (s *supervisionState) unanswered(b couchmessage.Binding) bool { _, ok := s.reports[b]; return ok }
func (s *supervisionState) reportID(b couchmessage.Binding) string { return s.reports[b].id }
func (s *supervisionState) notified(b couchmessage.Binding) bool   { return s.reports[b].notified }
func (s *supervisionState) nudged(b couchmessage.Binding, id string) {
	if r, ok := s.reports[b]; ok && r.id == id {
		r.notified = true
		s.reports[b] = r
	}
}
func (s *supervisionState) tick(now time.Time, facts []supervisionFacts, hub bool, availability ...map[couchmessage.Binding]bool) []supervisionEffect {
	var effects []supervisionEffect
	live := make(map[couchmessage.Binding]bool, len(facts))
	for _, f := range facts {
		b := f.binding
		ownHub := hub
		if len(availability) > 0 {
			ownHub = availability[0][b]
		}
		live[b] = true
		if ownHub && !s.attempted[b.Slot] {
			s.attempted[b.Slot] = true
			effects = append(effects, supervisionEffect{kind: supervisionPrompt, binding: b})
		}
		r, ok := s.reports[b]
		if !ok {
			continue
		}
		if f.submission > r.submission {
			delete(s.reports, b)
			continue
		}
		if f.muted || !f.known || !f.settled || f.pending || f.quiet < 30*time.Second || (f.resting && !r.completion) || now.Before(r.at) {
			continue
		}
		age := now.Sub(r.at)
		if !r.notified && age >= supervisionNudgeAfter {
			if f.automationOff {
				// Suppression consumes this occurrence even while the owning hub is
				// absent. Operator escalation remains independent below.
				r.notified = true
				s.reports[b] = r
			} else if ownHub {
				effects = append(effects, supervisionEffect{kind: supervisionNudge, binding: b, report: r.id, age: age})
			}
		}
		// Even a disconnected hub must not suppress the operator's escalation.
		if !r.escalated && age >= supervisionEscalateAfter {
			r.escalated = true
			s.reports[b] = r
			effects = append(effects, supervisionEffect{kind: supervisionEscalate, binding: b, report: r.id, age: age})
		}
	}
	for b, report := range s.reports {
		// A report can arrive after the tick took its connected snapshot.
		if !live[b] && !report.at.After(now) {
			delete(s.reports, b)
		}
	}
	return effects
}

var completionStart = regexp.MustCompile(`(?i)^(?:(?:[a-z0-9_.-]+)?#[0-9]+[ :—-]*)?(?:landed|merged)(?:\s|:|$)`)

func completionHint(body string) bool {
	line, _, _ := strings.Cut(body, "\n")
	// A conservative hint, never proof of completion. Only an explicit leading
	// claim qualifies; prose discussing someone else's landing does not.
	return completionStart.MatchString(strings.TrimSpace(truncateBytes(line, 256)))
}
