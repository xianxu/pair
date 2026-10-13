package couchtty

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// testLifetime is short because this level must let a REAL timer fire: the
// console arms time.NewTimer, so hand-advancing a fake clock would leave it
// waiting the full lifetime and prove nothing about the arming.
const testLifetime = 60 * time.Millisecond

// The half a pure Feed test cannot see: an idle console must REPAINT when the
// row expires. Nothing else is guaranteed to happen at that moment, and the
// operator's report was precisely that the sentence stayed on screen.
func TestAnIdleConsoleRepaintsWhenItsNoticeExpires(t *testing.T) {
	host := newVTHost(24, 80)
	reader, writer := io.Pipe()
	con := New(host, reader)
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
	})
	con.feed = NewFeed(8, time.Now, testLifetime)
	child := ptychild.NewFakeChild(nil)
	con.attachThreadActor("c1", "brain", menuAddress("brain"), "/w/brain", "brain", child)
	go con.Run()
	waitFor(t, "the console to start", func() bool { return len(child.Resizes()) > 0 })

	// The operator's own gesture: ctrl+backspace with nothing to return to. Sent
	// as BYTES so the notice is pushed from the Run goroutine, which is where
	// the expiry timer is armed -- a test that called setNotice directly would
	// arm nothing and prove nothing.
	if _, err := writer.Write([]byte{couchkeys.PreviousLegacy}); err != nil {
		t.Fatalf("write ctrl+backspace: %v", err)
	}
	// It must reach the SCREEN on its own. The first version of this test forced
	// a repaint here, which hid the regression giving notices a lifetime
	// introduced: on an idle console nothing else repaints, so a refusal that is
	// never painted now expires UNSEEN -- worse than one that overstayed.
	waitFor(t, "the refusal to reach the row with no other event", func() bool {
		return strings.Contains(host.Written(), "nowhere to return to")
	})

	// Now NOTHING happens except time passing. The console must repaint anyway;
	// no other event is coming, which is exactly why the message used to stay on
	// screen indefinitely.
	//
	// Asserting on the LAST painted row rather than resetting the buffer first.
	// A Reset here races the very paint it is waiting for: on a loaded box the
	// poll above can outlast the 60ms lifetime, the expiry paint lands before
	// the Reset, and the test then waits three seconds for a repaint that
	// already happened. A test must not discard the evidence it is waiting for.
	waitFor(t, "the row to repaint without the expired notice", func() bool {
		return !strings.Contains(host.row(24), "nowhere to return to")
	})
}

// And an exit stands: it says why a pane disappeared, which does not stop being
// true just because time passed.
func TestAnExitNoticeSurvivesAnIdleConsole(t *testing.T) {
	host := newVTHost(24, 80)
	reader, writer := io.Pipe()
	con := New(host, reader)
	t.Cleanup(func() {
		con.Stop()
		_ = writer.Close()
	})
	con.feed = NewFeed(8, time.Now, testLifetime)
	child := ptychild.NewFakeChild(nil)
	con.attachThreadActor("c1", "brain", menuAddress("brain"), "/w/brain", "brain", child)
	go con.Run()
	waitFor(t, "the console to start", func() bool { return len(child.Resizes()) > 0 })

	con.publishNotice(ExitNotice("couch-b1", "tools", 1))
	waitFor(t, "the exit notice to reach the row", func() bool {
		return strings.Contains(host.Written(), "exited (1)")
	})

	// Several lifetimes of real time, since the point is that nothing retires it.
	time.Sleep(5 * testLifetime)
	con.repaint()
	waitFor(t, "the row to be repainted after the wait", func() bool {
		return strings.Contains(host.row(24), "exited (1)")
	})
}

func TestSwitcherCopiesFullExpiredNoticeRepeatedly(t *testing.T) {
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

func TestSwitcherCopyNoticeEmptyHistory(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte{0, 0x19})
	waitFor(t, "empty notice feedback", func() bool { return f.con.menuSnapshot().Notice.Text == "No notice to copy." })
	if strings.Contains(f.host.Written(), "\x1b]52;") {
		t.Fatal("empty history replaced clipboard")
	}
}

func TestCopyNoticeChordStillReachesFocusedChild(t *testing.T) {
	f := newFixture(t, 24, 80)
	_, _ = f.stdin.Write([]byte{0x19})
	waitFor(t, "Ctrl+Y child input", func() bool { return bytes.Contains(bytes.Join(f.child.Writes(), nil), []byte{0x19}) })
	if strings.Contains(f.host.Written(), "\x1b]52;") {
		t.Fatal("actor chord copied a notice")
	}
}
