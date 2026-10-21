package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// A waiting notice (pair#451) tells the hub that a slot has sat waiting for
// someone past a threshold. Pair renders and never classifies (tail.go); the
// judgement is Couch's, as the tail's reader.

// waitFacts is what Couch already holds in memory about one slot.
type waitFacts struct {
	Quiet   time.Duration // since the wrapper's last output or keystroke; behind a picker, since it opened
	Settled *bool         // the wrapper's #421 self-report; nil is unknown
	Resting bool          // on its resting branch
	Pending bool          // a message to it is in flight
	Picker  bool          // its wrapper reports an input-holding picker up (pair#465)
}

type waitReason string

const (
	waitPrompt   waitReason = "on a prompt"
	waitQuestion waitReason = "after asking"
	waitIdle     waitReason = "idle off its resting branch"
	waitPicker   waitReason = "behind a picker"
)

// pickerNoticeAfter is how long a slot sits behind a picker before the hub
// hears of it (pair#465): a picker holds the slot outright, so it does not
// wait out the general threshold.
const pickerNoticeAfter = time.Minute

// needsTail reports whether the slot can be waiting at all, before any IO:
// behind a picker past pickerNoticeAfter, or quiet past the threshold with a
// Settled claim to judge by.
func needsTail(f waitFacts, threshold time.Duration) bool {
	if f.Picker {
		return f.Quiet >= pickerNoticeAfter
	}
	return f.Quiet >= threshold && f.Settled != nil
}

// classifyWait is the one definition of "waiting for someone". An unsettled
// slot that has been byte-quiet that long has something on screen asking for
// input (a dialog or a draft); a settled one waits if its
// turn ended with a question, or if it stopped off its resting branch with
// nothing in flight to it.
func classifyWait(f waitFacts, threshold time.Duration, tail []string) (waitReason, string, bool) {
	if !needsTail(f, threshold) {
		return "", "", false
	}
	if f.Picker {
		return waitPicker, pickerSummary(couchmessage.ReadPicker(tail)), true
	}
	prose := lastProseLine(tail)
	switch {
	case !*f.Settled && !completedModelControl(tail, prose):
		if title := dialogTitle(tail); title != "" {
			return waitPrompt, title, true
		}
		return waitPrompt, prose, true
	case isQuestion(prose):
		return waitQuestion, prose, true
	case !f.Resting && !f.Pending:
		return waitIdle, prose, true
	}
	return "", "", false
}

// completedModelControl is screen evidence, not a timed suppression. Reuse the
// exact captured confirmation/footer/composer grammar: a draft or later prose
// must still receive the normal wait classification even if older control
// confirmation chrome remains visible. Local and remote tails share this form.
func completedModelControl(tail []string, prose string) bool {
	for _, agent := range []string{"codex", "claude"} {
		evidence := agentdrive.ReadModelScreen(agent, agentdrive.Screen{Lines: tail, Complete: true, Idle: true})
		if evidence.Phase == agentdrive.ScreenStatus && evidence.Confirmation == prose {
			return true
		}
	}
	return false
}

// dimRun is a faint run in tail markup.
var dimRun = regexp.MustCompile(`‹dim›.*?(‹/dim›|$)`)

// stripTailMarkup renders one tail line as plain text (couchmessage owns the
// grammar).
func stripTailMarkup(line string) string { return couchmessage.StripTailMarkup(line) }

// dimOnly reports a line whose only text is faint: agent chrome.
func dimOnly(line string) bool {
	return stripTailMarkup(dimRun.ReplaceAllString(line, "")) == ""
}

// dialogTitle is the title of the screen's last picker, or "" without one.
func dialogTitle(tail []string) string { return couchmessage.ReadPicker(tail).Title }

// lastProseLine is the agent's last line of prose above its composer: chrome
// paragraphs (a turn summary ✻, a recap ※, all-faint runs, separators) are
// skipped.
func lastProseLine(tail []string) string {
	end := len(tail)
	for i := len(tail) - 1; i >= 0; i-- {
		if t := stripTailMarkup(tail[i]); strings.HasPrefix(t, "❯") || strings.HasPrefix(t, "›") {
			end = i
			break
		}
	}
	var last string
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		first := stripTailMarkup(para[0])
		chrome := strings.HasPrefix(first, "✻") || strings.HasPrefix(first, "※")
		allDim := true
		for _, l := range para {
			allDim = allDim && dimOnly(l)
		}
		if !chrome && !allDim {
			last = stripTailMarkup(para[len(para)-1])
		}
		para = nil
	}
	for _, raw := range tail[:end] {
		t := stripTailMarkup(raw)
		if t == "" || couchmessage.TailRule.MatchString(t) {
			flush()
			continue
		}
		para = append(para, raw)
	}
	flush()
	return last
}

func isQuestion(line string) bool {
	return strings.HasSuffix(strings.TrimRight(line, " *_`)\"'"), "?")
}

// maxWaitSummary bounds the quoted screen line in a notice, and each of a
// picker's title and detail; maxPickerOptions bounds its options' one line.
const (
	maxWaitSummary   = 200
	maxPickerOptions = 160
)

// pickerSummary is what a picker notice quotes: kind, detail, title, then the
// options on one line (pair#465). Each part has its own bound, so a long
// command never pushes out the question.
func pickerSummary(p couchmessage.Picker) string {
	if p.Kind == "" {
		return "(picker text not recognized)"
	}
	out := "(" + p.Kind + ") "
	if p.Detail != "" {
		out += truncateBytes(p.Detail, maxWaitSummary) + " — "
	}
	out += truncateBytes(p.Title, maxWaitSummary)
	if len(p.Options) > 0 {
		out += " [" + truncateBytes(strings.Join(p.Options, " | "), maxPickerOptions) + "]"
	}
	return out
}

// truncateBytes cuts s to at most n bytes on a rune boundary, marking the cut.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// waitNoticeBody puts the slot and reason first: a truncated send keeps its
// head (pair#211).
func waitNoticeBody(slot string, quiet time.Duration, reason waitReason, summary, optOutPath string) string {
	if reason != waitPicker {
		summary = truncateBytes(summary, maxWaitSummary) // pickerSummary bounds its own parts
	}
	if summary == "" {
		summary = "(no text on screen)"
	}
	return fmt.Sprintf("slot %s waiting %d min %s: %s (couch --peek %s; mute: add %s to %s)",
		slot, int(quiet/time.Minute), reason, summary, slot, slot, optOutPath)
}

func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

// parseWaitOptOut reads the operator's mute list: one slot per line, # comments.
func parseWaitOptOut(text string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if line = strings.TrimSpace(line); line != "" {
			out[line] = true
		}
	}
	return out
}

// waitNoticeThreshold reads COUCH_WAIT_NOTICE_AFTER, a Go duration of at
// least a minute; unset is 10 minutes. Anything else refuses Couch's start,
// so a typo is seen instead of silently becoming the default.
func waitNoticeThreshold(getenv func(string) string) (time.Duration, error) {
	v := getenv("COUCH_WAIT_NOTICE_AFTER")
	if v == "" {
		return 10 * time.Minute, nil
	}
	if d, err := time.ParseDuration(v); err == nil && d >= time.Minute {
		return d, nil
	}
	return 0, fmt.Errorf("COUCH_WAIT_NOTICE_AFTER=%q: want a Go duration of at least 1m (e.g. 10m), or unset it", v)
}

// waitTick is the watcher's pace: a tick reads memory only, and a picker
// notice lands within a tick of pickerNoticeAfter (pair#465).
const waitTick = 15 * time.Second

// waitTailLines is enough screen for a dialog or a turn's last paragraph.
const waitTailLines = 60

// waitWatcher sends the hub one notice per waiting episode (pair#451). It
// reads the service's memory every tick; only a slot quiet past the threshold
// costs IO, one tail read from its own wrapper per episode.
type waitWatcher struct {
	staleTick func() // nonblocking offer to the serialized freshness worker
	svc       *messageService
	threshold time.Duration
	optOut    string // the operator's mute list; Couch only reads it
	newID     func() (string, error)
	// seen is the last judged episode per connected binding, pruned to the
	// listing each tick; it dies with the service.
	seen map[couchmessage.Binding]waitEpisode
	// questions is the question set last handed to the attention list
	// (pair#673); nil until the first tick publishes.
	questions map[couchcore.ThreadAddress]couchcore.AttentionCondition
}

// waitEpisode: an episode is one judged wait — the same reason and the same
// words on screen. Output alone doesn't end it (pair#468): an idle agent
// still prints (Claude Code's update check redraws its footer every 30 min),
// and that moves LastActivity without a turn. A turn, the operator's input or
// a branch change ends it by changing what the slot waits on, or by making it
// not wait at all. notified holds for the whole episode (judged is the notice
// it sent); judgedNot only for the exact facts it judged.
type waitEpisode struct {
	key       waitKey
	notified  bool
	judged    string
	judgedNot bool
}

type waitKey struct {
	activity                          time.Time
	settled, resting, pending, picker bool
}

func newWaitWatcher(svc *messageService, threshold time.Duration, storeDir string) *waitWatcher {
	return &waitWatcher{svc: svc, threshold: threshold, optOut: filepath.Join(storeDir, "wait-notice-off"),
		newID: newRequestID, seen: map[couchmessage.Binding]waitEpisode{}}
}

// run ticks until ctx ends.
func (w *waitWatcher) run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			w.tick(ctx, now)
		}
	}
}

// hubBinding is the connected wrapper in the worktree Couch started from:
// the registry's newest admitted incarnation of that slot, so a replacement
// still settling never makes the choice arbitrary.
func (s *messageService) hubBinding() (couchmessage.Binding, bool) {
	return s.hubBindingAt(s.authority.hub)
}

func (s *messageService) hubBindingFor(from couchmessage.Binding) (couchmessage.Binding, bool) {
	if s.authority.hubs == nil {
		return s.hubBinding()
	}
	s.mu.Lock()
	identity, ok := s.workspaces[from]
	s.mu.Unlock()
	if !ok {
		return couchmessage.Binding{}, false
	}
	return s.hubBindingAt(s.authority.owner(identity))
}

func (s *messageService) hubBindingAt(root string) (couchmessage.Binding, bool) {
	if root == "" {
		return couchmessage.Binding{}, false
	}
	slot := ""
	s.mu.Lock()
	for b, identity := range s.workspaces {
		if identity.WorktreeRoot == root {
			slot = b.Slot
			break
		}
	}
	s.mu.Unlock()
	liveness := s.liveness.Load()
	if slot == "" || liveness == nil {
		return couchmessage.Binding{}, false
	}
	live, ok := (*liveness)[slot]
	if !ok || !s.isConnected(live.Binding) {
		return couchmessage.Binding{}, false
	}
	return live.Binding, true
}

func (s *messageService) isHubBinding(b couchmessage.Binding) bool {
	s.mu.Lock()
	identity, ok := s.workspaces[b]
	s.mu.Unlock()
	return ok && s.authority.isHub(identity)
}

func (w *waitWatcher) tick(ctx context.Context, now time.Time) {
	if w.staleTick != nil {
		w.staleTick()
	}
	coordinator := w.svc.supervision.Load()
	if coordinator != nil {
		defer func() {
			text, _ := os.ReadFile(w.optOut)
			coordinator.tick(ctx, now, parseWaitOptOut(string(text)))
		}()
	}
	hub, ok := w.svc.hubBinding()
	if !ok {
		connected := w.svc.broker.Connected()
		if len(connected) == 0 {
			return
		}
		hub = connected[0]
	}
	rows, err := w.svc.broker.Actors(ctx, hub)
	if err != nil {
		return
	}
	liveness := map[string]couchmessage.SlotLiveness{}
	if m := w.svc.liveness.Load(); m != nil {
		liveness = *m
	}
	listed := map[couchmessage.Binding]bool{}
	// waiting is each slot still quiet past its threshold this tick; one in a
	// question episode judged at this same activity is in the question set.
	type waitingRow struct {
		b        couchmessage.Binding
		activity time.Time
	}
	var waiting []waitingRow
	var muted map[string]bool
	var muteErr string // a mute list that can't be read is said in the notice
	for _, row := range rows {
		b := row.Binding
		listed[b] = true
		if w.svc.isHubBinding(b) || !row.Known {
			continue
		}
		permit := w.svc.captureOwnerHubInputPermit(b)
		reportID := ""
		if coordinator != nil {
			reportID, _ = coordinator.waitState(b)
		}
		since := row.LastActivity
		f := waitFacts{Resting: row.Resting, Pending: row.Pending}
		if live, ok := liveness[b.Slot]; ok && live.Binding == b {
			f.Settled = live.Settled
			f.Picker = live.Picker != nil && *live.Picker
			// A picker can repaint (Claude Code's auto-deny countdown), so
			// the slot is never quiet behind it: its wait runs from when it
			// opened. The episode stays keyed by LastActivity; a repaint
			// re-reads the tail and #468's same-wait rule keeps it one notice.
			if f.Picker && !live.PickerSince.IsZero() {
				since = live.PickerSince
			}
		}
		f.Quiet = now.Sub(since)
		if !needsTail(f, w.threshold) {
			continue
		}
		key := waitKey{activity: row.LastActivity, settled: f.Settled != nil && *f.Settled, resting: f.Resting, pending: f.Pending, picker: f.Picker}
		waiting = append(waiting, waitingRow{b, key.activity})
		if ep, ok := w.seen[b]; ok && ((ep.notified && ep.key.activity == key.activity) || (ep.judgedNot && ep.key == key)) {
			continue
		}
		if muted == nil {
			text, err := os.ReadFile(w.optOut)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				muteErr = fmt.Sprintf(" [mute list unreadable: %v]", err)
			}
			muted = parseWaitOptOut(string(text))
		}
		if muted[b.Slot] {
			continue
		}
		reader, ok := w.svc.authority.endpoint(b).(couchmessage.TailReader)
		if !ok {
			continue
		}
		tctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		tail, err := reader.Tail(tctx, waitTailLines)
		cancel()
		if err != nil {
			continue // unreadable now; the next tick asks again
		}
		reason, summary, waiting := classifyWait(f, w.threshold, tail.Lines)
		if !waiting {
			w.seen[b] = waitEpisode{key: key, judgedNot: true}
			continue
		}
		if coordinator != nil && reason == waitIdle {
			if _, notified := coordinator.waitState(b); notified {
				continue
			}
		}
		judged := string(reason) + "\x00" + summary
		if ep, ok := w.seen[b]; ok && ep.notified && ep.judged == judged {
			w.seen[b] = waitEpisode{key: key, notified: true, judged: judged} // output without a turn
			continue
		}
		id, err := w.newID()
		if err != nil {
			continue
		}
		body := waitNoticeBody(b.Slot, f.Quiet, reason, summary, w.optOut) + muteErr
		// From the waiting slot to the hub's exact slot: the notice is an
		// ordinary message, so it gets the hub's breaker, delivery and ledger.
		// Tail reading may yield while ownership changes. Resolve again at admission.
		hub, hasHub := w.svc.hubBindingFor(b)
		var sendErr error
		if !permit.current(w.svc) {
			sendErr = errHubAutomationOff
		} else if !hasHub {
			continue
		} else if w.svc.generatedHubInputOff(hub) {
			sendErr = errHubAutomationOff
		} else if coordinator != nil {
			_, sendErr = coordinator.sendGenerated(ctx, b, hub, id, body, false, permit)
		} else {
			_, sendErr = w.svc.sendAutomaticSystem(ctx, b, hub, id, body, permit)
		}
		if sendErr != nil && !errors.Is(sendErr, errHubAutomationOff) {
			continue // hub busy or out of allowance: retried next tick
		}
		w.seen[b] = waitEpisode{key: key, notified: true, judged: judged}
		if coordinator != nil {
			coordinator.waitNudged(b, reportID)
		}
	}
	for b := range w.seen {
		if !listed[b] {
			delete(w.seen, b)
		}
	}
	// A turn moves the activity, so the episode ends with it: the explicit
	// end the question set needs.
	questions := map[couchcore.ThreadAddress]couchcore.AttentionCondition{}
	for _, row := range waiting {
		ep, ok := w.seen[row.b]
		if summary, question := questionEpisode(ep); ok && question && ep.key.activity.Equal(row.activity) {
			questions[bindingThread(row.b)] = couchcore.AttentionCondition{Label: row.b.Slot, Title: summary}
		}
	}
	w.publishQuestions(questions)
}
