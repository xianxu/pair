package couchtty

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/castlock"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

func warningSourceItems(c *Console) []couchcore.AttentionItem {
	c.attn.mu.Lock()
	defer c.attn.mu.Unlock()
	return c.attn.list.Items()
}

func TestWarningSourcesTraceOpenFailuresRaiseAttention(t *testing.T) {
	for _, kind := range []string{"input", "mouse", "event"} {
		t.Run(kind, func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
			t.Cleanup(c.Stop)
			path := filepath.Join(t.TempDir(), "absent-parent", "trace.log")
			var err error
			var source string
			switch kind {
			case "input":
				err, source = c.SetInputTrace(path), "COUCH_INPUT_TRACE"
			case "mouse":
				err, source = c.SetMouseTrace(path), "COUCH_MOUSE_TRACE"
			case "event":
				err, source = c.SetEventTrace(path, time.Now()), "COUCH_TRACE"
			}
			if err == nil {
				t.Fatal("trace unexpectedly opened its absent parent")
			}
			items := warningSourceItems(c)
			if len(items) != 1 || !strings.Contains(items[0].Body, source) || items[0].Thread != (couchcore.ThreadAddress{}) {
				t.Fatalf("trace failure did not raise one global source warning: %+v", items)
			}
		})
	}
}

func TestWarningSourcesCastGuessingAttentionIsSecretSafeAndGenerationOwned(t *testing.T) {
	for _, kind := range []castKind{castScreen, castWeb} {
		t.Run(kind.name(), func(t *testing.T) {
			var c *Console
			var attempt uint64
			var stop func()
			if kind == castScreen {
				f := broadcastFixture(t, &broadcast.FakeTunnel{}, nil)
				f.startLive(t)
				c, attempt = f.con, f.screenAttempt()
				stop = func() {
					c.stopBroadcast()
					waitFor(t, "screen cast stopped", func() bool { return f.phase() == broadcastOff })
				}
			} else {
				r := newWebcastRig(t, nil)
				r.open(t)
				c = r.con
				c.mu.Lock()
				attempt = c.wcast.attempt
				c.mu.Unlock()
				stop = func() {
					c.endWebcast()
					waitFor(t, "web cast stopped", func() bool { return r.wphase() == webcastOff })
				}
			}
			rotation := castlock.Rotation{Code: "519351", Reason: castlock.ReasonGuessing, Gen: 10, Capped: true}
			c.onCodeRotated(kind, attempt, broadcast.LinkView, rotation)
			c.onCodeRotated(kind, attempt, broadcast.LinkView, rotation)
			rotation.Code, rotation.Gen = "838383", 9
			c.onCodeRotated(kind, attempt, broadcast.LinkView, rotation)
			rotation.Gen = 11
			c.onCodeRotated(kind, attempt+1, broadcast.LinkView, rotation)
			rotation.Reason = castlock.ReasonOperator
			c.onCodeRotated(kind, attempt, broadcast.LinkView, rotation)
			items := warningSourceItems(c)
			if len(items) != 1 || !strings.Contains(items[0].Body, "locked until you rotate") {
				t.Fatalf("duplicate, stale or operator rotation changed warning ownership: %+v", items)
			}
			if strings.Contains(items[0].Body, "519351") || strings.Contains(items[0].Body, "838383") {
				t.Fatalf("attention exposed a cast secret: %+v", items)
			}
			stop()
			rotation.Reason, rotation.Gen = castlock.ReasonGuessing, 12
			c.onCodeRotated(kind, attempt, broadcast.LinkView, rotation)
			if after := warningSourceItems(c); len(after) != 1 || after[0].ID != items[0].ID {
				t.Fatalf("ended cast's retained attempt raised stale attention: %+v", after)
			}
		})
	}
}

func TestWarningSourcesBroadcastFailureAndNormalStopDiffer(t *testing.T) {
	for _, failure := range []bool{false, true} {
		name := "operator-stop"
		if failure {
			name = "tunnel-failure"
		}
		t.Run(name, func(t *testing.T) {
			tunnel := &broadcast.FakeTunnel{}
			f := broadcastFixture(t, tunnel, nil)
			session := f.startLive(t)
			if failure {
				tunnel.Exit()
			} else {
				f.con.stopBroadcast()
			}
			waitFor(t, "broadcast stop settled", func() bool { return f.phase() == broadcastOff })
			if failure {
				waitFor(t, "failed broadcast warning", func() bool { return f.con.attentionCount() == 1 })
			}
			f.con.broadcastEnded(session) // An obsolete completion has no new effect.
			items := warningSourceItems(f.con)
			if !failure && len(items) != 0 {
				t.Fatalf("ordinary stop raised attention: %+v", items)
			}
			if failure && (len(items) != 1 || !strings.Contains(items[0].Body, "Broadcast ended:")) {
				t.Fatalf("failed broadcast lost or duplicated its warning: %+v", items)
			}
		})
	}
}

func TestWarningSourcesWebcastRecoveryRaisesOncePerEpisode(t *testing.T) {
	c, _ := continuationConsole(t)
	// The pointer is the current-cast identity token for update admission. This
	// test drives only updates; no browser operation is invoked on that token.
	current, retired := &webcast.Cast{}, &webcast.Cast{}
	c.wcast = webcastState{cast: current, phase: webcastRunning}
	defer func() { c.wcast = webcastState{} }()
	update := webcast.Update{State: webcast.State{Recovery: "Cast lost its active tab; choose a tab"}}
	c.webcastUpdated(retired, update)
	if c.attentionCount() != 0 {
		t.Fatal("retired cast update raised attention")
	}
	c.webcastUpdated(current, update)
	c.webcastUpdated(current, update)
	if c.attentionCount() != 1 {
		t.Fatal("unchanged recovery state repeated its warning")
	}
	c.webcastUpdated(current, webcast.Update{})
	c.webcastUpdated(current, update)
	if c.attentionCount() != 2 {
		t.Fatal("a later recovery episode did not raise a new warning")
	}
}

func TestWarningSourcesNormalWebcastEndDoesNotRaiseAttention(t *testing.T) {
	for _, byHand := range []bool{false, true} {
		t.Run(map[bool]string{false: "console-stop", true: "browser-close"}[byHand], func(t *testing.T) {
			r := newWebcastRig(t, nil)
			browser := r.open(t)
			if byHand {
				browser.QuitByHand()
			} else {
				r.con.endWebcast()
			}
			waitFor(t, "webcast stop settled", func() bool { return r.wphase() == webcastOff })
			if items := warningSourceItems(r.con); len(items) != 0 {
				t.Fatalf("normal browser end raised attention: %+v", items)
			}
		})
	}
}

// The stateful browser still owns close/subscription/process lifetime; only its
// terminal error differs from an ordinary operator close.
type warningFailedBrowser struct {
	*webcast.FakeBrowser
}

func (b warningFailedBrowser) Err() error {
	if b.FakeBrowser.Err() != nil {
		return errors.New("browser connection lost")
	}
	return nil
}

func TestWarningSourcesAsyncWebcastFailureRaisesAttention(t *testing.T) {
	r := newWebcastRig(t, func(r *webcastRig) {
		r.wrap = func(browser *webcast.FakeBrowser, _ *webcast.RecordOptions) webcast.Browser {
			return warningFailedBrowser{browser}
		}
	})
	browser := r.open(t)
	browser.QuitByHand()
	waitFor(t, "failed browser warning", func() bool { return r.con.attentionCount() == 1 })
	items := warningSourceItems(r.con)
	if !strings.Contains(items[0].Body, "Chrome cast ended: browser connection lost") {
		t.Fatalf("asynchronous browser failure lost its cause: %+v", items)
	}
}
