package agentdrive

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type EventKind string

const (
	EventStart       EventKind = "start"
	EventScreen      EventKind = "screen"
	EventWritten     EventKind = "written"
	EventWriteFailed EventKind = "write-failed"
	EventCancel      EventKind = "cancel"
	EventTick        EventKind = "tick"
)

type DriveEvent struct {
	Kind   EventKind
	At     time.Time
	Screen Screen
	Error  string
}
type EffectKind string

const (
	EffectWrite   EffectKind = "write"
	EffectObserve EffectKind = "observe"
	EffectWait    EffectKind = "wait"
)

type DriveEffect struct {
	Kind    EffectKind
	Key     KeyStep
	Control bool
	Delay   time.Duration
	// NotBefore keeps Enter out of a receiver's rapid-text paste window.
	// The existing writer polls this bound inside the absolute drive deadline.
	NotBefore time.Time
	// ScreenEpoch pins writes derived from picker evidence. Zero denotes a raw
	// caller step or initial command entry, which has no picker observation.
	ScreenEpoch uint64
	proof       ScreenEvidence
}

// TextSubmitDelay also paces ordinary peer submissions. Codex 0.162.1 retains
// a 120ms Enter-suppression window after a rapid text burst has rendered.
const TextSubmitDelay = 150 * time.Millisecond

// MatchesScreen permits a repaint only when the complete parsed picker evidence
// is identical to the evidence supporting this action. Ordered labels, selection
// and footer remain part of the proof; an epoch change alone is not authority.
func (e DriveEffect) MatchesScreen(agent string, screen Screen) bool {
	if !screen.Complete || screen.TurnActive {
		return false
	}
	if e.ScreenEpoch == 0 {
		return true
	}
	if screen.Epoch < e.ScreenEpoch || e.proof.Phase == ScreenUnknown {
		return false
	}
	fresh := ReadModelScreen(agent, screen)
	old := e.proof
	if fresh.Phase != old.Phase || fresh.Selected != old.Selected || fresh.Model != old.Model || fresh.Effort != old.Effort || fresh.Scope != old.Scope || fresh.Footer != old.Footer || fresh.ComposerText != old.ComposerText || len(fresh.Labels) != len(old.Labels) {
		return false
	}
	for i, label := range fresh.Labels {
		if label != old.Labels[i] {
			return false
		}
	}
	return true
}

// Internal recipe directions are excluded from ParseKeySequence's caller syntax.
const (
	KeyLeft  KeyKind = "recipe-left"
	KeyRight KeyKind = "recipe-right"
)

type drivePhase uint8

const (
	phaseNew drivePhase = iota
	phaseWriting
	phaseRaw
	phaseWaiting
	phaseFinal
	phaseModel
	phaseEffort
	phaseAdvanced
	phaseApplied
	phaseBackout
	phaseTerminal
)

type DriveState struct {
	request                              Request
	agent                                string
	deadline                             time.Time
	recoverAt                            time.Time
	lastWriteAt                          time.Time
	textQuietUntil                       time.Time
	writingKey                           KeyKind
	phase, after                         drivePhase
	index, command, navigation, backouts int
	epoch, writeEpoch                    uint64
	waitUntil                            time.Time
	effort                               string
	selectedModel                        string
	evidence                             ScreenEvidence
	result                               Result
	priorConfirmations                   map[string]int
	// dismissal marks a drive admitted only by a recognized panel; its
	// outcome is the composer returning, not just a fresh frame (pair#691).
	dismissal bool
}

func NewDrive(r Request, agent string, deadline time.Time) (DriveState, error) {
	if err := r.Validate(); err != nil {
		return DriveState{}, err
	}
	if deadline.IsZero() {
		return DriveState{}, fmt.Errorf("drive needs an absolute deadline")
	}
	if r.Model != nil && agent != "codex" && agent != "claude" {
		return DriveState{}, fmt.Errorf("unsupported model recipe for %q", agent)
	}
	if r.Model != nil && agent == "claude" && r.Model.EffectiveScope() == ScopeDefault {
		return DriveState{}, fmt.Errorf("Claude default scope is not supported by a verified recipe; use session scope")
	}
	return cloneDrive(DriveState{request: r, agent: agent, deadline: deadline}), nil
}
func cloneDrive(s DriveState) DriveState {
	s.request.Keys = append(KeySequence(nil), s.request.Keys...)
	if s.request.Model != nil {
		m := *s.request.Model
		s.request.Model = &m
	}
	s.evidence.Labels = append([]string(nil), s.evidence.Labels...)
	s.result.Tail = append([]string(nil), s.result.Tail...)
	counts := map[string]int{}
	for key, n := range s.priorConfirmations {
		counts[key] = n
	}
	s.priorConfirmations = counts
	return s
}
func (s DriveState) Terminal() bool { return s.phase == phaseTerminal }
func (s DriveState) Started() bool  { return s.phase != phaseNew }

// RecoveryDue also preempts a queued key whose quiet period would consume the
// cleanup reserve. The wrapper must not replace other pending effects on ticks.
func (s DriveState) RecoveryDue(at time.Time) bool {
	return !s.Terminal() && s.request.Model != nil && !s.recoverAt.IsZero() && !at.Before(s.recoverAt) && s.phase != phaseBackout && !(s.phase == phaseWriting && s.after == phaseBackout)
}
func (s DriveState) Result() Result {
	r := s.result
	r.Tail = append([]string(nil), r.Tail...)
	return r
}
func boundedTail(lines []string) []string {
	out := []string{}
	remaining := MaxResultTailBytes
	for i := len(lines) - 1; i >= 0 && remaining > 0; i-- {
		line := lines[i]
		if len(line) >= remaining {
			line = line[len(line)-(remaining-1):]
			for !utf8.ValidString(line) && len(line) > 0 {
				line = line[1:]
			}
		}
		out = append(out, line)
		remaining -= len(line) + 1
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func finish(s DriveState, out Outcome, detail string) (DriveState, []DriveEffect) {
	s.phase = phaseTerminal
	s.result.Outcome = out
	s.result.Detail = detail
	return s, nil
}
func observe(s DriveState) (DriveState, []DriveEffect) {
	return s, []DriveEffect{{Kind: EffectObserve}}
}
func write(s DriveState, k KeyStep, next drivePhase) (DriveState, []DriveEffect) {
	s.phase = phaseWriting
	s.after = next
	s.writingKey = k.Kind
	if s.epoch > s.writeEpoch {
		s.writeEpoch = s.epoch
	}
	effect := DriveEffect{Kind: EffectWrite, Key: k, Control: s.request.Model != nil}
	if s.agent == "codex" && (k.Kind == KeyEnter || k.Kind == KeySubmit) {
		effect.NotBefore = s.textQuietUntil
	}
	if s.request.Model != nil && s.command >= 3 {
		bindProof(&effect, s)
	}
	return s, []DriveEffect{effect}
}

func bindProof(effect *DriveEffect, s DriveState) {
	effect.ScreenEpoch = s.epoch
	effect.proof = s.evidence
	effect.proof.Labels = append([]string(nil), s.evidence.Labels...)
}

func reservedRecovery(s DriveState, screen Screen) (DriveState, []DriveEffect) {
	detail := "model drive timed out"
	if s.phase == phaseApplied || (s.phase == phaseWriting && s.after == phaseApplied) || !screen.Complete || screen.TurnActive || screen.Epoch == 0 || screen.Epoch < s.writeEpoch {
		return finish(s, OutcomeUnconfirmed, detail+"; outcome unconfirmed; inspect the slot before sending more input")
	}
	s.epoch = screen.Epoch
	s.result.Tail = boundedTail(screenHistory(screen))
	s.evidence = ReadModelScreen(s.agent, screen)
	if screen.Idle {
		if screen.Epoch <= s.writeEpoch {
			return finish(s, OutcomeUnconfirmed, detail+"; fresh empty prompt unconfirmed; inspect the slot")
		}
		return finish(s, OutcomeFailed, detail+"; back at empty prompt")
	}
	switch s.evidence.Phase {
	case ScreenCommand:
		if s.command < 2 {
			return finish(s, OutcomeUnconfirmed, detail+"; command ownership unconfirmed; inspect the slot")
		}
		s.result.Detail = detail
		s, fx := write(s, KeyStep{Kind: KeyClearLine}, phaseBackout)
		bindProof(&fx[0], s)
		return s, fx
	case ScreenModel, ScreenEffort, ScreenAdvanced, ScreenCombined:
		if s.command < 3 {
			return finish(s, OutcomeUnconfirmed, detail+"; picker ownership unconfirmed; inspect the slot")
		}
		return backout(s, detail)
	default:
		return finish(s, OutcomeUnconfirmed, detail+"; prompt recovery unconfirmed; inspect the slot before sending more input")
	}
}
func uncertain(s DriveState) Outcome {
	if s.result.Step > 0 || s.phase == phaseWriting || s.phase == phaseApplied {
		return OutcomeUnconfirmed
	}
	return OutcomeRefused
}
func rawNext(s DriveState, at time.Time) (DriveState, []DriveEffect) {
	if s.index == len(s.request.Keys) {
		s.phase = phaseFinal
		return observe(s)
	}
	key := s.request.Keys[s.index]
	if key.Kind == KeyWait {
		s.index++
		s.phase = phaseWaiting
		s.waitUntil = at.Add(time.Duration(key.WaitMS) * time.Millisecond)
		return s, []DriveEffect{{Kind: EffectWait, Delay: time.Duration(key.WaitMS) * time.Millisecond}}
	}
	return write(s, key, phaseRaw)
}
func commandNext(s DriveState) (DriveState, []DriveEffect) {
	keys := []KeyStep{{Kind: KeyClearLine}, {Kind: KeyText, Text: "/model"}, {Kind: KeySubmit}}
	if s.command == len(keys) {
		s.phase = phaseModel
		return observe(s)
	}
	return write(s, keys[s.command], phaseModel)
}

// Keep actionable choices separate from the changing tail, within the existing
// 1 KiB receipt-detail envelope, including later recovery diagnostics. Whole
// labels retain UTF-8 boundaries; an omitted suffix is reported explicitly.
func choiceFailure(kind string, labels []string) string {
	detail := "requested " + kind + " not among observed choices: "
	for i, label := range labels {
		separator := ""
		if i > 0 {
			separator = ", "
		}
		if len(detail)+len(separator)+len(label) > 800 {
			return detail + "; further choices omitted (diagnostic limit)"
		}
		detail += separator + label
	}
	return detail
}

func backout(s DriveState, detail string) (DriveState, []DriveEffect) {
	s.result.Detail = detail
	if s.backouts >= 3 {
		return finish(s, OutcomeUnconfirmed, detail+"; prompt recovery unconfirmed")
	}
	s.backouts++
	return write(s, KeyStep{Kind: KeyEsc}, phaseBackout)
}

// ReduceDrive is the sole transition owner. IO supplies one acknowledgement per
// emitted write and fresh screen epochs; no admission or effect is retried here.
func ReduceDrive(input DriveState, event DriveEvent) (DriveState, []DriveEffect) {
	s := cloneDrive(input)
	if s.Terminal() {
		return s, nil
	}
	if event.Kind == EventCancel {
		detail := "drive canceled"
		if event.Error != "" {
			detail += ": " + event.Error
		}
		return finish(s, uncertain(s), detail+"; inspect the slot before sending more input")
	}
	if !event.At.Before(s.deadline) {
		if s.dismissal && s.phase == phaseFinal && event.Screen.Dismissable {
			s.result.Tail = boundedTail(screenHistory(event.Screen))
			return finish(s, OutcomeUnconfirmed, "esc written but the panel still shows at the deadline; inspect the slot before sending more input")
		}
		return finish(s, uncertain(s), "drive deadline expired; inspect the slot before sending more input")
	}
	if event.Kind == EventWriteFailed {
		return finish(s, OutcomeUnconfirmed, "input outcome unconfirmed: "+event.Error)
	}
	if event.Screen.TurnActive {
		return finish(s, uncertain(s), "agent turn started; input stopped")
	}
	if event.Kind == EventTick && s.RecoveryDue(event.At) {
		return reservedRecovery(s, event.Screen)
	}
	switch event.Kind {
	case EventStart:
		if s.phase != phaseNew {
			return s, nil
		}
		if !s.request.Admits(event.Screen.Admission()) || !event.Screen.Complete {
			return finish(s, OutcomeRefused, s.request.Refusal(event.Screen.Admission(), "wait until the agent composer is empty and its turn finishes"))
		}
		s.dismissal = event.Screen.Dismissable && !event.Screen.Idle && !event.Screen.Recoverable
		s.epoch = event.Screen.Epoch
		s.result.Tail = boundedTail(screenHistory(event.Screen))
		if !rememberConfirmations(&s, screenHistory(event.Screen)) {
			return finish(s, OutcomeRefused, "confirmation history exceeds observed-label limit")
		}
		if s.request.Model == nil {
			return rawNext(s, event.At)
		}
		reserve := s.deadline.Sub(event.At) / 10
		if reserve > 3*time.Second {
			reserve = 3 * time.Second
		}
		s.recoverAt = s.deadline.Add(-reserve)
		return commandNext(s)
	case EventWritten:
		if s.phase != phaseWriting {
			return s, nil
		}
		s.result.Step++
		if s.writingKey == KeyText {
			s.textQuietUntil = event.At.Add(TextSubmitDelay)
		}
		if event.At.After(s.lastWriteAt) {
			s.lastWriteAt = event.At
		}
		if event.Screen.Epoch > s.writeEpoch {
			s.writeEpoch = event.Screen.Epoch
		}
		s.phase = s.after
		if s.request.Model == nil {
			s.index++
			return rawNext(s, event.At)
		}
		if s.command < 3 && s.phase != phaseBackout {
			s.command++
			return commandNext(s)
		}
		return observe(s)
	case EventTick:
		if s.phase == phaseWaiting && !event.At.Before(s.waitUntil) {
			return rawNext(s, event.At)
		}
		return observe(s)
	case EventScreen:
		if s.request.RecoveryOnly() && s.phase == phaseFinal && !event.At.Before(s.lastWriteAt) && event.Screen.Complete && event.Screen.Idle && event.Screen.Epoch >= s.writeEpoch {
			s.result.Tail = boundedTail(screenHistory(event.Screen))
			return finish(s, OutcomeConfirmed, "recovery key bytes accepted and current empty composer observed")
		}
		if event.Screen.Epoch <= s.epoch || event.Screen.Epoch <= s.writeEpoch || !event.Screen.Complete {
			return observe(s)
		}
		s.epoch = event.Screen.Epoch
		s.result.Tail = boundedTail(screenHistory(event.Screen))
		if s.phase != phaseApplied && !(s.phase == phaseWriting && s.after == phaseApplied) {
			if !rememberConfirmations(&s, screenHistory(event.Screen)) {
				return finish(s, OutcomeUnconfirmed, "confirmation history exceeds observed-label limit")
			}
		}
		if s.phase == phaseWriting || s.phase == phaseNew || s.phase == phaseWaiting {
			return s, nil
		}
		if s.request.Model == nil {
			if s.dismissal {
				// Only the composer's return confirms a dismissal. A panel frame
				// after the write may be its own repaint (a usage panel loads its
				// data) drawn before the agent read the Esc, so it proves nothing.
				return observe(s)
			}
			if s.phase == phaseFinal && s.epoch > s.writeEpoch {
				return finish(s, OutcomeConfirmed, "key bytes and final observation confirmed")
			}
			return observe(s)
		}
		evidence := ReadModelScreen(s.agent, event.Screen)
		s.evidence = evidence
		if s.phase == phaseBackout {
			if event.Screen.Idle {
				return finish(s, OutcomeFailed, s.result.Detail+"; back at empty prompt")
			}
			if evidence.Phase == ScreenModel || evidence.Phase == ScreenEffort || evidence.Phase == ScreenAdvanced || evidence.Phase == ScreenCombined {
				return backout(s, s.result.Detail)
			}
			return finish(s, OutcomeUnconfirmed, s.result.Detail+"; prompt recovery unconfirmed")
		}
		if s.phase == phaseApplied {
			req := s.request.Model
			if evidence.Phase == ScreenStatus && evidence.Model == strings.ToLower(req.Model) && evidence.Effort == s.effort && evidence.Scope == req.EffectiveScope() && event.Screen.Idle && s.epoch > s.writeEpoch && evidence.ConfirmationCount > s.priorConfirmations[evidence.Confirmation] {
				s.result.Model = evidence.Model
				s.result.Effort = evidence.Effort
				s.result.Scope = evidence.Scope
				return finish(s, OutcomeConfirmed, "model, effort and scope confirmed")
			}
			return observe(s)
		}
		if evidence.Phase == ScreenUnknown {
			return observe(s)
		}
		if s.navigation >= MaxNavigationSteps {
			return backout(s, "navigation limit reached")
		}
		target := strings.ToLower(s.request.Model.Model)
		if evidence.Phase == ScreenModel || evidence.Phase == ScreenCombined {
			if s.phase != phaseModel {
				return backout(s, "unexpected model picker")
			}
			selected, targetIndex := -1, -1
			for i, label := range evidence.Labels {
				if label == evidence.Selected {
					selected = i
				}
				if label == target {
					targetIndex = i
				}
			}
			if targetIndex < 0 {
				return backout(s, choiceFailure("model", evidence.Labels))
			}
			if selected != targetIndex {
				s.navigation++
				k := KeyDown
				if selected > targetIndex {
					k = KeyUp
				}
				return write(s, KeyStep{Kind: k}, phaseModel)
			}
			if evidence.Phase == ScreenModel {
				s.selectedModel = evidence.Selected
				return write(s, KeyStep{Kind: KeyEnter}, phaseEffort)
			}
			s.effort = evidence.Effort
			if s.request.Model.Effort != "" && canonicalEffort(s.request.Model.Effort) != s.effort {
				levels := []string{"low", "medium", "high", "xhigh", "max", "ultra"}
				current, want := -1, -1
				for i, l := range levels {
					if l == s.effort {
						current = i
					}
					if l == canonicalEffort(s.request.Model.Effort) {
						want = i
					}
				}
				if current < 0 || want < 0 {
					return backout(s, choiceFailure("effort", []string{s.effort}))
				}
				s.navigation++
				k := KeyRight
				if current > want {
					k = KeyLeft
				}
				return write(s, KeyStep{Kind: k}, phaseModel)
			}
			return apply(s)
		}
		if evidence.Phase == ScreenEffort || evidence.Phase == ScreenAdvanced {
			if (evidence.Phase == ScreenEffort && (s.phase != phaseEffort || evidence.Model != target)) || (evidence.Phase == ScreenAdvanced && (s.phase != phaseAdvanced || s.selectedModel != target)) {
				return backout(s, "unexpected effort picker")
			}
			effort := canonicalEffort(s.request.Model.Effort)
			if effort == "" {
				effort = evidence.Selected
			}
			if evidence.Phase == ScreenEffort && (effort == "max" || effort == "ultra") {
				effort = "more"
			}
			selected, targetIndex := -1, -1
			for i, label := range evidence.Labels {
				if label == evidence.Selected {
					selected = i
				}
				if label == effort {
					targetIndex = i
				}
			}
			if targetIndex < 0 {
				return backout(s, choiceFailure("effort", evidence.Labels))
			}
			if selected != targetIndex {
				s.navigation++
				k := KeyDown
				if selected > targetIndex {
					k = KeyUp
				}
				next := phaseEffort
				if evidence.Phase == ScreenAdvanced {
					next = phaseAdvanced
				}
				return write(s, KeyStep{Kind: k}, next)
			}
			if evidence.Phase == ScreenEffort && effort == "more" {
				return write(s, KeyStep{Kind: KeyEnter}, phaseAdvanced)
			}
			s.effort = effort
			return apply(s)
		}
	}
	return s, nil
}

func rememberConfirmations(s *DriveState, lines []string) bool {
	for key, n := range confirmationCounts(lines) {
		if _, ok := s.priorConfirmations[key]; !ok && len(s.priorConfirmations) >= MaxObservedLabels {
			return false
		}
		if n > s.priorConfirmations[key] {
			s.priorConfirmations[key] = n
		}
	}
	return true
}
func apply(s DriveState) (DriveState, []DriveEffect) {
	key := KeyStep{Kind: KeyEnter}
	if s.request.Model.EffectiveScope() == ScopeSession {
		key = KeyStep{Kind: KeyText, Text: "s"}
	}
	return write(s, key, phaseApplied)
}
