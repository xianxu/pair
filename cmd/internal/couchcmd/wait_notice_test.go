package couchcmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Tails below are shaped on live `couch --peek` captures (2026-10-11).
var (
	claudePermissionTail = []string{
		"  I'll push the branch now.",
		"",
		"╭──────────────────────────────────────────────╮",
		"│ Bash command                                 │",
		"│   git push origin HEAD                       │",
		"│ Do you want to proceed?                      │",
		"│ ❯ 1. Yes                                     │",
		"│   2. No, and tell Claude what to do (esc)    │",
		"╰──────────────────────────────────────────────╯",
	}
	claudeSelectionTail = []string{
		"  Which layout should the notice use?",
		"",
		"❯ 1. Compact",
		"  2. Verbose",
		"",
		"Enter to select · ↑/↓ to navigate · Esc to cancel",
	}
	claudeQuestionTail = []string{
		"  The fix is on the branch and the tests pass.",
		"  Should I merge it now?",
		"",
		"✻ Cooked for 15m 32s · done 9:15 AM",
		"",
		"※ recap: I'm landing #318; next, you run the commit.",
		"  Then tell me to merge?",
		"",
		"──────────────────────────────────────────────",
		"❯ ‹cursor›",
		"──────────────────────────────────────────────",
		"  ⏵⏵ auto mode on (shift+tab to cycle)",
	}
	claudeDoneTail = []string{
		"  Review: the first close review approved it.",
		"  Merged as PR #247; pair:2 is back on its resting branch.",
		"",
		"✻ Baked for 20m 24s · done 1:01 PM",
		"",
		"──────────────────────────────────────────────",
		"❯ ‹cursor›‹dim›move this branch to :0‹/dim›",
		"──────────────────────────────────────────────",
	}
	codexDoneTail = []string{
		"  ‹dim›│‹/dim› plans/000426-review-pane",
		"  ‹dim›└    Passed seven targeted Lua suites‹/dim›",
		"",
		"‹dim›•‹/dim› Implemented #426: compact comment chains.",
		"",
		"  Not merged. Uses ‹‹markup›› in text.",
		"",
		"  ‹dim›Worked for 28m 1s • 12:46 PM‹/dim›",
		"",
		"",
		"› ‹cursor›‹dim›Ask Codex to do anything‹/dim›",
		"",
		"  GPT-6-Astra medium · ~/workspace/pair",
	}
)

func boolp(b bool) *bool { return &b }

func TestClassifyWait(t *testing.T) {
	const threshold = 10 * time.Minute
	quiet := 11 * time.Minute
	cases := []struct {
		name    string
		f       waitFacts
		tail    []string
		reason  waitReason
		summary string
		waiting bool
	}{
		{"permission dialog names the dialog", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: false}, claudePermissionTail, waitPrompt, "Do you want to proceed?", true},
		{"selection dialog names its question", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: true}, claudeSelectionTail, waitPrompt, "Which layout should the notice use?", true},
		{"unsettled without a dialog quotes the prose", waitFacts{Quiet: quiet, Settled: boolp(false), Resting: true}, claudeDoneTail, waitPrompt, "Merged as PR #247; pair:2 is back on its resting branch.", true},
		{"turn ended with a question", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, claudeQuestionTail, waitQuestion, "Should I merge it now?", true},
		{"normal end on resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, claudeDoneTail, "", "", false},
		{"codex normal end on resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: true}, codexDoneTail, "", "", false},
		{"idle off resting branch", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: false}, claudeDoneTail, waitIdle, "Merged as PR #247; pair:2 is back on its resting branch.", true},
		{"idle off resting with a message in flight", waitFacts{Quiet: quiet, Settled: boolp(true), Resting: false, Pending: true}, claudeDoneTail, "", "", false},
		{"not quiet long enough", waitFacts{Quiet: 9 * time.Minute, Settled: boolp(false)}, claudePermissionTail, "", "", false},
		{"no Settled claim is unknown, never waiting", waitFacts{Quiet: quiet}, claudePermissionTail, "", "", false},
	}
	for _, c := range cases {
		reason, summary, waiting := classifyWait(c.f, threshold, c.tail)
		if reason != c.reason || summary != c.summary || waiting != c.waiting {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q, %v)", c.name, reason, summary, waiting, c.reason, c.summary, c.waiting)
		}
	}
}

func modelConfirmationTail(t *testing.T, agent string) []string {
	t.Helper()
	version := "0.162.1"
	if agent == "claude" {
		version = "2.1.299"
	}
	b, err := os.ReadFile("../wrapcmd/testdata/drive/" + agent + "/" + version + "/session-status.screen")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(b), "\n")
}

func TestClassifyWaitCompletedModelControl(t *testing.T) {
	const threshold = 10 * time.Minute
	for _, agent := range []string{"codex", "claude"} {
		for _, settled := range []bool{false, true} {
			for _, resting := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/settled=%v/resting=%v", agent, settled, resting), func(t *testing.T) {
					tail := modelConfirmationTail(t, agent)
					if agent == "codex" {
						for i, line := range tail {
							tail[i] = strings.ReplaceAll(line, " low", " medium")
						}
					}
					reason, _, waiting := classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(settled), Resting: resting}, threshold, tail)
					if resting {
						if waiting {
							t.Fatalf("completed control reported %q", reason)
						}
					} else if !waiting || reason != waitIdle {
						t.Fatalf("off-resting idle lost: %q %v", reason, waiting)
					}
				})
			}
		}
	}
}

func TestClassifyWaitCompletedModelControlPending(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		for _, settled := range []bool{false, true} {
			reason, _, waiting := classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(settled), Pending: true}, 10*time.Minute, modelConfirmationTail(t, agent))
			if waiting {
				t.Fatalf("%s pending slot reported %q", agent, reason)
			}
		}
	}
}

func TestClassifyWaitModelConfirmationDoesNotHideLaterInput(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		base := modelConfirmationTail(t, agent)
		for _, settled := range []bool{false, true} {
			tail := append([]string{}, base...)
			for i := len(tail) - 1; i >= 0; i-- {
				line := stripTailMarkup(tail[i])
				if strings.HasPrefix(line, "›") || strings.HasPrefix(line, "❯") {
					tail = append(tail[:i], append([]string{"Should I merge it now?", ""}, tail[i:]...)...)
					break
				}
			}
			reason, summary, waiting := classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(settled), Resting: true}, 10*time.Minute, tail)
			if !waiting || summary != "Should I merge it now?" || (settled && reason != waitQuestion) {
				t.Fatalf("%s later question hidden: %q %q %v", agent, reason, summary, waiting)
			}
		}
		tail := append(append([]string{}, base...), claudePermissionTail...)
		reason, _, waiting := classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(false), Resting: true}, 10*time.Minute, tail)
		if !waiting || reason != waitPrompt {
			t.Fatalf("%s later permission hidden", agent)
		}
		reason, _, waiting = classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(true), Resting: true, Picker: true}, 10*time.Minute, tail)
		if !waiting || reason != waitPicker {
			t.Fatalf("%s actual picker hidden", agent)
		}
		tail = append([]string{}, base...)
		for i := len(tail) - 1; i >= 0; i-- {
			line := stripTailMarkup(tail[i])
			if strings.HasPrefix(line, "›") || strings.HasPrefix(line, "❯") {
				tail[i] = "❯ a real unsent draft"
				break
			}
		}
		reason, _, waiting = classifyWait(waitFacts{Quiet: 11 * time.Minute, Settled: boolp(false), Resting: true}, 10*time.Minute, tail)
		if !waiting || reason != waitPrompt {
			t.Fatalf("%s draft hidden", agent)
		}
	}
}

func TestLastProseLineSkipsChrome(t *testing.T) {
	for _, c := range []struct {
		tail []string
		want string
	}{
		{claudeQuestionTail, "Should I merge it now?"},
		{codexDoneTail, "Not merged. Uses ‹markup›› in text."},
		{[]string{"plain", "", "last line"}, "last line"},
		{nil, ""},
	} {
		if got := lastProseLine(c.tail); got != c.want {
			t.Errorf("lastProseLine(%q) = %q, want %q", c.tail, got, c.want)
		}
	}
}

func TestIsQuestionToleratesTrailingMarkdown(t *testing.T) {
	for line, want := range map[string]bool{"Merge now?": true, "**Merge now?**": true, "(merge now?)": true, "Merged.": false, "": false} {
		if got := isQuestion(line); got != want {
			t.Errorf("isQuestion(%q) = %v", line, got)
		}
	}
}

func TestWaitNoticeBodyPutsSlotFirstAndBoundsTheSummary(t *testing.T) {
	body := waitNoticeBody("ariadne:3", 12*time.Minute+40*time.Second, waitPrompt, strings.Repeat("é", 300), "/store/wait-notice-off")
	if !strings.HasPrefix(body, "slot ariadne:3 waiting 12m on a prompt: ") {
		t.Fatalf("body head %q", body[:60])
	}
	if !strings.Contains(body, "é…") || strings.Count(body, "é") != 100 {
		t.Fatalf("summary not bounded on a rune boundary: %q", body)
	}
	if !strings.HasSuffix(body, "(couch --peek ariadne:3; mute: add ariadne:3 to /store/wait-notice-off)") {
		t.Fatalf("body tail %q", body)
	}
}

func TestParseWaitOptOut(t *testing.T) {
	got := parseWaitOptOut("# muted\nariadne:2\n  pair:4  # noisy\n\n")
	if len(got) != 2 || !got["ariadne:2"] || !got["pair:4"] {
		t.Fatalf("opt-out %v", got)
	}
}

// tailEndpoint serves a fixed screen; hubInbox records what reaches the hub.
type tailEndpoint struct {
	serviceEndpointFake
	lines *[]string
	mu    *sync.Mutex
}

func (e tailEndpoint) Tail(context.Context, int) (couchmessage.Tail, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return couchmessage.Tail{Lines: append([]string(nil), *e.lines...)}, nil
}

// The hub itself sits on a dialog, so a watcher that forgot to skip it would
// notify itself in every test.
//
// The broker admits a notice inside the watcher's tick (Reserve runs on the
// Send path) but delivers it later, on the recipient's actor; admitted holds
// those notices, so a test waits for exactly them to land.
type hubInbox struct {
	tailEndpoint
	got      *[]couchmessage.Message
	refuse   *int            // admissions to refuse before accepting
	refusal  *error          // what a refused reservation returns (nil: busy)
	admitted map[string]bool // reservations accepted, less those released undelivered
}

func (e hubInbox) Reserve(ctx context.Context, id string, _ uint64) error {
	e.tailEndpoint.mu.Lock()
	defer e.tailEndpoint.mu.Unlock()
	if *e.refuse > 0 {
		*e.refuse--
		if *e.refusal != nil {
			return *e.refusal
		}
		return couchmessage.ErrRecipientBusy
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	e.admitted[id] = true
	return nil
}

// Release follows every delivery and every refusal too; only an admission
// that never delivered is given back.
func (e hubInbox) Release(_ context.Context, id string) error {
	e.tailEndpoint.mu.Lock()
	defer e.tailEndpoint.mu.Unlock()
	for _, m := range *e.got {
		if m.ID == id {
			return nil
		}
	}
	delete(e.admitted, id)
	return nil
}

func (e hubInbox) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	e.tailEndpoint.mu.Lock()
	*e.got = append(*e.got, m)
	e.tailEndpoint.mu.Unlock()
	return couchmessage.Receipt{Status: couchmessage.Submitted}, ctx.Err()
}

type waitRig struct {
	*serviceRig
	hub, worker couchmessage.Binding
	client      *couchmessage.SessionClient
	watcher     *waitWatcher
	mu          sync.Mutex
	screen      []string
	delivered   []couchmessage.Message
	refuse      int
	refusal     error
	admitted    map[string]bool
}

// newWaitRig runs hub pair:1 and a hello-v2 worker pair:2 showing screen.
func newWaitRig(t *testing.T, settled, resting bool, screen []string, decorate ...func(messageAuthority) messageAuthority) *waitRig {
	w := &waitRig{serviceRig: newServiceRig(t, decorate...), screen: screen, admitted: map[string]bool{}}
	w.world.hub = "/repo1"
	w.restart(nil)
	w.hub = w.world.add(1)
	w.worker = w.world.add(2)
	hubScreen := claudePermissionTail
	w.world.endpoints[w.hub] = hubInbox{tailEndpoint: tailEndpoint{lines: &hubScreen, mu: &w.mu}, got: &w.delivered, refuse: &w.refuse, refusal: &w.refusal, admitted: w.admitted}
	w.world.endpoints[w.worker] = tailEndpoint{lines: &w.screen, mu: &w.mu}
	w.serviceRig.mu.Lock()
	w.slotGit["/repo1"] = couchcore.SlotGitStatus{Branch: "main-slot1"}
	branch := "main-slot2"
	if !resting {
		branch = "000451-issue"
	}
	w.slotGit["/repo2"] = couchcore.SlotGitStatus{Branch: branch}
	w.serviceRig.mu.Unlock()
	w.v2wrapper(w.hub, "h1", false)
	w.waitSettled(w.hub, false)
	w.client = w.v2wrapper(w.worker, "h2", settled)
	w.waitSettled(w.worker, settled)
	w.watcher = newWaitWatcher(w.s, 10*time.Minute, t.TempDir())
	return w
}

// v2wrapper runs a hello-v2 session for b, the kind that reports Settled.
func (w *waitRig) v2wrapper(b couchmessage.Binding, pane couchmessage.PaneHandle, settled bool) *couchmessage.SessionClient {
	c := couchmessage.NewSessionClient()
	c.SetBuild(couchmessage.BuildIdentity{SHA256: strings.Repeat("a", 64)})
	c.Update(couchmessage.Observation{LastActivity: time.Now(), Sequence: 1})
	c.Settle(settled)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Run(ctx, w.regSock, b, couchmessage.DefaultReconnectBackoff, 10*time.Millisecond)
	}()
	w.t.Cleanup(func() { cancel(); <-done })
	w.attach(b, pane)
	w.waitConnected(b, true)
	return c
}

func (w *waitRig) waitSettled(b couchmessage.Binding, want bool) {
	w.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if live, ok := w.s.LivenessForThread(b.Scope, b.Tag); ok && live.Settled != nil && *live.Settled == want {
			return
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("worker never reported Settled=%v", want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// tickAt runs one watcher pass at offset past now and returns every notice
// the hub has received, once every notice the pass admitted has landed. The
// pass admits synchronously, so a notice it wrongly sent is counted before
// tickAt starts waiting; a delivery that never lands is the test binary's
// timeout, not a wall-clock window here.
func (w *waitRig) tickAt(offset time.Duration) []couchmessage.Message {
	w.watcher.tick(context.Background(), time.Now().Add(offset))
	for {
		w.mu.Lock()
		if len(w.delivered) >= len(w.admitted) {
			defer w.mu.Unlock()
			return append([]couchmessage.Message(nil), w.delivered...)
		}
		w.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
}

// #451 Done-when 1: a permission dialog left past the threshold produces
// exactly one hub notice, from the waiting slot, naming the dialog.
func TestWaitNoticeDialogNotifiesTheHubOncePerEpisode(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	if got := w.tickAt(9 * time.Minute); len(got) != 0 {
		t.Fatalf("notice before the threshold: %+v", got)
	}
	got := w.tickAt(11 * time.Minute)
	got = w.tickAt(12 * time.Minute)
	if len(got) != 1 {
		t.Fatalf("want exactly one notice, got %d: %+v", len(got), got)
	}
	m := got[0]
	if m.From != w.worker || m.To != w.hub || !strings.HasPrefix(m.Body, "slot pair:2 waiting 1") || !strings.Contains(m.Body, "on a prompt: Do you want to proceed?") {
		t.Fatalf("notice %+v", m)
	}
	// The slot works again, then waits on a new prompt: a new episode, one
	// more notice.
	w.mu.Lock()
	w.screen = claudeSelectionTail
	w.mu.Unlock()
	w.client.Update(couchmessage.Observation{LastActivity: time.Now().Add(time.Second), Sequence: 2})
	deadline := time.Now().Add(3 * time.Second)
	for len(w.tickAt(11*time.Minute)) != 2 {
		if time.Now().After(deadline) {
			t.Fatal("a new waiting episode sent no notice")
		}
	}
	if got := w.tickAt(13 * time.Minute); len(got) != 2 {
		t.Fatalf("second episode notified %d times", len(got)-1)
	}
}

// #451 Done-when 2: a turn that ended with a question notifies once; one that
// ended normally on the resting branch does not.
func TestWaitNoticeQuestionNotifiesNormalEndDoesNot(t *testing.T) {
	q := newWaitRig(t, true, true, claudeQuestionTail)
	q.tickAt(11 * time.Minute)
	if got := q.tickAt(12 * time.Minute); len(got) != 1 || !strings.Contains(got[0].Body, "after asking: Should I merge it now?") {
		t.Fatalf("question notices %+v", got)
	}
	done := newWaitRig(t, true, true, claudeDoneTail)
	if got := done.tickAt(30 * time.Minute); len(got) != 0 {
		t.Fatalf("normal end on resting branch notified: %+v", got)
	}
}

func TestWaitNoticeIdleOffRestingBranchNotifies(t *testing.T) {
	w := newWaitRig(t, true, false, claudeDoneTail)
	if got := w.tickAt(11 * time.Minute); len(got) != 1 || !strings.Contains(got[0].Body, "idle off resting branch") {
		t.Fatalf("idle notices %+v", got)
	}
}

func TestWaitNoticeMutedSlotAndHubAreSilent(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	if err := os.WriteFile(w.watcher.optOut, []byte("# muted\npair:2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("muted slot notified: %+v", got)
	}
	// With no hub connected there is no one to tell.
	w.world.hub = ""
	w.restart(map[couchmessage.ThreadKey]couchmessage.PaneHandle{w.hub.Thread(): "h1", w.worker.Thread(): "h2"})
	w.waitConnected(w.worker, true)
	w.watcher = newWaitWatcher(w.s, 10*time.Minute, t.TempDir())
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("notice without a hub: %+v", got)
	}
}

func TestWaitNoticeThreshold(t *testing.T) {
	for v, want := range map[string]time.Duration{"": 10 * time.Minute, "2m": 2 * time.Minute, "30s": 0, "junk": 0} {
		got, err := waitNoticeThreshold(func(string) string { return v })
		if got != want || (err != nil) != (want == 0) {
			t.Errorf("threshold %q = %v, %v; want %v", v, got, err, want)
		}
	}
}

// A hub too busy to take the notice holds it in line (pair#492): it arrives
// once, with no resend on a later tick.
func TestWaitNoticeHeldByABusyHubArrivesOnce(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	w.mu.Lock()
	w.refuse = 1
	w.mu.Unlock()
	w.tickAt(11 * time.Minute)
	testwait.Until(t, "the held notice", func() bool { w.mu.Lock(); defer w.mu.Unlock(); return len(w.delivered) == 1 })
	if got := w.tickAt(12 * time.Minute); len(got) != 1 {
		t.Fatalf("held notice resent or lost: %d notices", len(got))
	}
}

// A refused admission (the hub's wrapper unreachable) leaves the episode
// unannounced; the next tick sends it.
func TestWaitNoticeRetriesAfterARefusedAdmission(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	w.mu.Lock()
	w.refuse = 1
	w.refusal = couchmessage.ErrUnavailable
	w.mu.Unlock()
	if got := w.tickAt(11 * time.Minute); len(got) != 0 {
		t.Fatalf("refused admission delivered: %+v", got)
	}
	if got := w.tickAt(12 * time.Minute); len(got) != 1 {
		t.Fatalf("retry sent %d notices, want 1", len(got))
	}
}

// A mute list Couch cannot read mutes nothing, and the notice says so.
func TestWaitNoticeSaysWhenTheMuteListIsUnreadable(t *testing.T) {
	w := newWaitRig(t, false, false, claudePermissionTail)
	if err := os.Mkdir(w.watcher.optOut, 0o700); err != nil {
		t.Fatal(err)
	}
	if got := w.tickAt(11 * time.Minute); len(got) != 1 || !strings.Contains(got[0].Body, "[mute list unreadable: ") {
		t.Fatalf("notices %+v", got)
	}
}

func TestDialogTitleIgnoresNumberedProse(t *testing.T) {
	cursorOnTwo := []string{
		"Plan:",
		"1. run tests",
		"2. push",
		"",
		"Do you want to proceed?",
		"  1. Yes",
		"❯ 2. No, and tell Claude what to do (esc)",
	}
	for _, c := range []struct {
		tail []string
		want string
	}{
		{cursorOnTwo, "Do you want to proceed?"},
		{append([]string{"Plan:", "1. run tests", "2. push", ""}, claudePermissionTail...), "Do you want to proceed?"},
		{[]string{"Plan:", "1. run tests", "2. push"}, ""},
	} {
		if got := dialogTitle(c.tail); got != c.want {
			t.Errorf("dialogTitle(%q) = %q, want %q", c.tail, got, c.want)
		}
	}
}

// pair#468 Done-when: a slot idle off its resting branch for an hour, whose
// wrapper still reports output every 30 minutes (Claude Code's update check
// redraws its footer) with nothing new on screen, is one episode: one
// notice. A turn that leaves new words on screen is a new one.
func TestWaitNoticeOutputWithoutATurnKeepsTheEpisode(t *testing.T) {
	w := newWaitRig(t, true, false, claudeDoneTail)
	start := time.Now()
	if got := w.tickAt(11 * time.Minute); len(got) != 1 {
		t.Fatalf("first notice: %+v", got)
	}
	for _, at := range []time.Duration{30 * time.Minute, 60 * time.Minute} {
		w.client.Update(couchmessage.Observation{LastActivity: start.Add(at), Sequence: uint64(at / time.Minute)})
		w.waitObserved(start.Add(at))
		for _, after := range []time.Duration{11 * time.Minute, 15 * time.Minute} {
			if got := w.tickAt(at + after); len(got) != 1 {
				t.Fatalf("output at +%v without a turn re-notified: %d notices %+v", at, len(got), got)
			}
		}
	}
	w.mu.Lock()
	w.screen = claudeQuestionTail
	w.mu.Unlock()
	w.client.Update(couchmessage.Observation{LastActivity: start.Add(70 * time.Minute), Sequence: 100})
	w.waitObserved(start.Add(70 * time.Minute))
	if got := w.tickAt(81 * time.Minute); len(got) != 2 || !strings.Contains(got[1].Body, "after asking") {
		t.Fatalf("a turn with new words on screen must notify again: %+v", got)
	}
}

// waitObserved waits until the worker's pushed activity reaches at.
func (w *waitRig) waitObserved(at time.Time) {
	w.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		rows, _ := w.s.broker.Actors(context.Background(), w.hub)
		for _, r := range rows {
			if r.Binding == w.worker && !r.LastActivity.Before(at) {
				return
			}
		}
		if time.Now().After(deadline) {
			w.t.Fatalf("the worker's activity never reached %v", at)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// #673: a question episode puts the slot in the attention list's question
// set, a quiet tick republishes nothing, and the next turn (the activity
// moves) takes it out: the explicit end a condition item needs.
func TestWaitNoticeQuestionSetStartsAndEndsWithTheEpisode(t *testing.T) {
	q := newWaitRig(t, true, true, claudeQuestionTail)
	var mu sync.Mutex
	var facts []couchcore.AttentionFact
	q.s.SetAttention(func(f couchcore.AttentionFact) {
		mu.Lock()
		defer mu.Unlock()
		facts = append(facts, f)
	})
	last := func() (int, couchcore.AttentionFact) {
		mu.Lock()
		defer mu.Unlock()
		if len(facts) == 0 {
			return 0, couchcore.AttentionFact{}
		}
		return len(facts), facts[len(facts)-1]
	}
	q.tickAt(11 * time.Minute)
	q.tickAt(12 * time.Minute)
	n, f := last()
	want := couchcore.AttentionCondition{Label: "pair:2", Title: "Should I merge it now?"}
	if f.Kind != couchcore.AttentionQuestion || len(f.Live) != 1 || f.Live[bindingThread(q.worker)] != want {
		t.Fatalf("question set after the episode = %+v", f)
	}
	q.tickAt(13 * time.Minute)
	if again, _ := last(); again != n {
		t.Fatalf("an unchanged question set was republished (%d facts, was %d)", again, n)
	}
	q.client.Update(couchmessage.Observation{LastActivity: time.Now().Add(time.Second), Sequence: 2})
	deadline := time.Now().Add(3 * time.Second)
	for {
		q.tickAt(time.Second)
		if _, f := last(); f.Kind == couchcore.AttentionQuestion && len(f.Live) == 0 {
			return
		}
		if time.Now().After(deadline) {
			_, f := last()
			t.Fatalf("a turn did not end the question: %+v", f)
		}
	}
}
