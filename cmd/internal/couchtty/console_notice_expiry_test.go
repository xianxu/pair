package couchtty

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// testLifetime is short because this level must let a REAL timer fire: the
// console arms time.NewTimer, so hand-advancing a fake clock would leave it
// waiting the full lifetime and prove nothing about the arming.
const testLifetime = 60 * time.Millisecond

// A real input-loop publication stays out of chrome both before and after
// transient expiry, but remains retained and inspectable in the switcher.
func TestConsoleRetainsExpiredGestureNoticeForSwitcher(t *testing.T) {
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.feed = NewFeed(8, time.Now, testLifetime) })
	if _, err := f.stdin.Write([]byte{couchkeys.PreviousLegacy}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "gesture retained", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		messages := f.con.feed.Messages()
		return len(messages) > 0 && strings.Contains(messages[len(messages)-1].Body, "nowhere to return to")
	})
	if strings.Contains(f.screen.row(24), "nowhere to return to") {
		t.Fatal("gesture reached tab bar")
	}
	waitFor(t, "feed transient expiry", func() bool {
		f.con.mu.Lock()
		defer f.con.mu.Unlock()
		return f.con.feed.Row().Body == ""
	})
	if strings.Contains(f.screen.row(24), "nowhere to return to") {
		t.Fatal("expired gesture reached tab bar")
	}
	f.con.mu.Lock()
	f.con.menu.Notice = MenuNotice{} // No provider is installed in this fixture.
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "expired gesture in switcher", func() bool { return strings.Contains(f.screenText(), "nowhere to return to") })
}

// Control notices keep their existing feed lifetime while their presentation
// moves from chrome to the switcher.
func TestAnExitNoticeSurvivesInSwitcher(t *testing.T) {
	now := time.Now()
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.feed = NewFeed(8, func() time.Time { return now }, testLifetime) })
	f.con.publishNotice(ExitNotice("couch-b1", "tools", 1))
	f.con.mu.Lock()
	now = now.Add(5 * testLifetime)
	got := f.con.feed.Row().Body
	f.con.mu.Unlock()
	if !strings.Contains(got, "exited (1)") {
		t.Fatalf("control notice expired: %q", got)
	}
	if strings.Contains(f.screen.row(24), "exited (1)") {
		t.Fatal("exit notice reached tab bar")
	}
	f.con.mu.Lock()
	f.con.menu.Notice = MenuNotice{} // No provider is installed in this fixture.
	f.con.mu.Unlock()
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "exit notice in switcher", func() bool { return strings.Contains(f.screenText(), "exited (1)") })
}

func TestSwitcherCopiesFullExpiredNoticeRepeatedly(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // Exercise fallback without touching the host clipboard.
	clock := time.Now()
	full := "Chrome cast pointer link unavailable: broadcast: cloudflared exited while opening: " + strings.Repeat("diagnostic ", 40) + "\nfinal diagnostic tail"
	f := newFixtureBeforeRun(t, 24, 44, func(c *Console) {
		c.feed = NewFeed(8, func() time.Time { return clock }, NoticeLifetime)
		c.feed.Push(Notice{Kind: "status", Body: full})
	})
	if strings.Contains(f.screen.row(24), "final diagnostic tail") {
		t.Fatal("fixture did not clip notice")
	}
	f.con.mu.Lock()
	clock = clock.Add(NoticeLifetime + time.Second)
	expired := f.con.feed.Row().Body == ""
	f.con.mu.Unlock()
	if !expired {
		t.Fatal("notice did not expire")
	}
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(full)) + "\x1b\\"
	for count := 1; count <= 2; count++ {
		_, _ = f.stdin.Write([]byte{0x19})
		waitFor(t, "complete notice clipboard", func() bool { return strings.Count(f.host.Written(), want) == count })
	}
	if !strings.Contains(f.screenText(), "Ctrl+Y copy last notice") {
		t.Fatal("copy action is not discoverable")
	}
}

func TestSwitcherCopiesFullNoticeToSystemClipboard(t *testing.T) {
	toolDir := t.TempDir()
	clipboard := filepath.Join(toolDir, "clipboard")
	calls := filepath.Join(toolDir, "calls")
	tool := "#!/bin/sh\n/bin/cat > \"$TEST_NOTICE_CLIPBOARD\"\nprintf x >> \"$TEST_NOTICE_CALLS\"\n"
	if err := os.WriteFile(filepath.Join(toolDir, "pbcopy"), []byte(tool), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", toolDir)
	t.Setenv("TEST_NOTICE_CLIPBOARD", clipboard)
	t.Setenv("TEST_NOTICE_CALLS", calls)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	full := "clipboard diagnostic 界\n" + strings.Repeat("unclipped details ", 40) + "\nfinal tail"
	f := newFixtureBeforeRun(t, 24, 44, func(c *Console) {
		c.feed.Push(Notice{Kind: "status", Body: full})
	})
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	for count := 1; count <= 2; count++ {
		f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
		got, err := os.ReadFile(clipboard)
		if err != nil || string(got) != full {
			t.Fatalf("notice did not reach system clipboard: got %q, err %v", got, err)
		}
		n, err := os.ReadFile(calls)
		if err != nil || len(n) != count {
			t.Fatalf("clipboard calls = %q, err %v; want %d", n, err, count)
		}
		if got := f.con.menuSnapshot().CopyFeedback.Text; got != "Full notice copied to clipboard." {
			t.Fatalf("copy feedback = %q", got)
		}
	}
	if strings.Contains(f.host.Written(), "\x1b]52;") {
		t.Fatal("confirmed local copy also emitted an unverifiable OSC 52 request")
	}
}

func TestSwitcherNoticeClipboardFallbackIsUnconfirmed(t *testing.T) {
	for _, reason := range []string{"missing", "failed", "timeout", "remote"} {
		t.Run(reason, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("PATH", dir)
			for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
				t.Setenv(key, "")
			}
			marker := filepath.Join(dir, "called")
			t.Setenv("TEST_NOTICE_TOOL_CALLED", marker)
			body := "exit 7"
			if reason == "timeout" {
				body = "exec /bin/sleep 30"
			} else if reason == "remote" {
				t.Setenv("SSH_CONNECTION", "remote")
				body = `printf unexpected > "$TEST_NOTICE_TOOL_CALLED"`
			}
			if reason != "missing" {
				if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			full := "complete notice\nlast line"
			f := newFixtureBeforeRun(t, 24, 44, func(c *Console) {
				c.feed.Push(Notice{Kind: "status", Body: full})
			})
			_, _ = f.stdin.Write([]byte{0, 0x19})
			want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(full)) + "\x1b\\"
			waitFor(t, "unconfirmed terminal fallback", func() bool {
				return strings.Contains(f.host.Written(), want) && strings.Contains(f.con.menuSnapshot().CopyFeedback.Text, "terminal copy requested; delivery is unconfirmed.")
			})
			if reason == "remote" {
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatalf("remote copy invoked local clipboard: %v", err)
				}
			}
		})
	}
}

func TestSwitcherCopyNoticeEmptyHistory(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte{0, 0x19})
	waitFor(t, "empty notice feedback", func() bool { return f.con.menuSnapshot().CopyFeedback.Text == "No notice to copy." })
	if strings.Contains(f.host.Written(), "\x1b]52;") {
		t.Fatal("empty history replaced clipboard")
	}
}

func TestCopyNoticeChordStillReachesFocusedChild(t *testing.T) {
	toolDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(toolDir, "pbcopy"), []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", toolDir)

	for _, wire := range []string{"\x19", "\x1b[121;5u", "\x1b[121;5:3u"} {
		t.Run(wire, func(t *testing.T) {
			observations := make(chan CopyKeyObservation, 4)
			f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.SetCopyKeyObserver(func(o CopyKeyObservation) { observations <- o }) })
			_, _ = f.stdin.Write([]byte(wire))
			select {
			case o := <-observations:
				if o.Panel || o.Release != (wire == "\x1b[121;5:3u") {
					t.Fatalf("arrival=%+v", o)
				}
			case <-time.After(time.Second):
				t.Fatal("no child-focus arrival")
			}
			if wire != "\x1b[121;5:3u" {
				waitFor(t, "Ctrl+Y child input", func() bool { return bytes.Contains(bytes.Join(f.child.Writes(), nil), []byte{0x19}) })
			}
			if strings.Contains(f.host.Written(), "\x1b]52;") || f.con.menuSnapshot().CopyFeedback.Kind != CopyFeedbackNone {
				t.Fatal("actor chord copied a notice")
			}
		})
	}
}

func TestSwitcherClipboardKeepsNativeFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\nprintf 'pasteboard unavailable' >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.feed.Push(Notice{Kind: "status", Body: "original notice"}) })
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	if text := f.con.menuSnapshot().CopyFeedback.Text; !strings.Contains(text, "pbcopy") || !strings.Contains(text, "pasteboard unavailable") {
		t.Fatalf("native failure was erased: %q", text)
	}
}

func TestSwitcherClipboardRetryKeepsOriginalAndObservesOnce(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	tool := filepath.Join(dir, "pbcopy")
	captured := filepath.Join(dir, "captured")
	t.Setenv("TEST_RETRY_CAPTURE", captured)
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'private original' >&2\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	observations := make(chan CopyObservation, 4)
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.feed.Push(Notice{Kind: "status", Body: "private original"})
		c.SetCopyObserver(func(o CopyObservation) { observations <- o })
	})
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	first := waitCopyObservation(t, observations)
	raw, _ := json.Marshal(first)
	if strings.Contains(string(raw), "private") || first.Native.Tool != "pbcopy" || first.Native.ExitCode == nil || *first.Native.ExitCode != 7 || first.OSC != "unconfirmed" {
		t.Fatalf("unsafe or incomplete observation: %s", raw)
	}
	f.con.mu.Lock()
	f.con.feed.Push(Notice{Kind: "status", Body: "new unrelated notice"})
	f.con.mu.Unlock()
	f.con.onMenuKey(PanelKey{Kind: KeyRune, Rune: 'z'})
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n/bin/cat > \"$TEST_RETRY_CAPTURE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	second := waitCopyObservation(t, observations)
	got, err := os.ReadFile(captured)
	if err != nil || string(got) != "private original" || second.Outcome != "copied" || f.con.menuSnapshot().CopyFeedback.payload != "" {
		t.Fatalf("retry lost its original: %q / %v / %+v", got, err, second)
	}
	select {
	case extra := <-observations:
		t.Fatalf("duplicate observation: %+v", extra)
	default:
	}
}

func TestSwitcherClipboardRetainsBothFailures(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.feed.Push(Notice{Kind: "status", Body: strings.Repeat("x", terminal.MaxStringBytes+1)})
	})
	_, _ = f.stdin.Write([]byte{0})
	waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
	f.con.onMenuKey(PanelKey{Kind: KeyCopyNotice})
	feedback := f.con.menuSnapshot().CopyFeedback
	if feedback.Kind != CopyFeedbackFailed || !strings.Contains(feedback.Text, "local clipboard unavailable") || !strings.Contains(feedback.Text, "terminal fallback:") {
		t.Fatalf("both failures lost: %+v", feedback)
	}
	f.con.onMenuKey(PanelKey{Kind: KeyEscape})
	if f.con.menuSnapshot().CopyFeedback.Kind != CopyFeedbackNone {
		t.Fatal("Escape did not dismiss")
	}
	f.con.mu.Lock()
	panel := f.con.focus.IsPanel()
	f.con.mu.Unlock()
	if !panel {
		t.Fatal("dismissal switched away")
	}
}

func TestSwitcherClipboardShutdownCancelsWithoutLateFeedback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	started := filepath.Join(dir, "started")
	t.Setenv("TEST_COPY_STARTED", started)
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\nprintf x > \"$TEST_COPY_STARTED\"\nexec /bin/sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	observations := make(chan CopyObservation, 1)
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) {
		c.feed.Push(Notice{Kind: "status", Body: "original"})
		c.SetCopyObserver(func(o CopyObservation) { observations <- o })
	})
	_, _ = f.stdin.Write([]byte{0, 0x19})
	waitFor(t, "clipboard child start handshake", func() bool { _, err := os.Stat(started); return err == nil })
	f.con.Stop()
	select {
	case o := <-observations:
		if !o.Native.Canceled {
			t.Fatalf("cancellation lost: %+v", o)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("clipboard child not reaped after cancellation")
	}
	if f.con.menuSnapshot().CopyFeedback.Kind != CopyFeedbackNone {
		t.Fatal("cancelled copy resurrected feedback")
	}
}

func waitCopyObservation(t *testing.T, observations <-chan CopyObservation) CopyObservation {
	t.Helper()
	select {
	case observation := <-observations:
		return observation
	case <-time.After(3 * time.Second):
		t.Fatal("copy observation missing")
		return CopyObservation{}
	}
}
