package couchtty

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestWarningPublicationAndSelectionControlledOrder(t *testing.T) {
	for _, publicationFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(publicationFirst), func(t *testing.T) {
			c, _ := continuationConsole(t)
			at := time.Unix(100, 0)
			c.now = func() time.Time { return at }
			old := Notice{Kind: "old", Body: "old failure"}
			newer := Notice{Kind: "new", Body: "new failure"}
			c.Warn(old)
			c.toggleAttention()
			// Both operations are already waiting at explicit channel boundaries.
			// Complete either one before releasing the other, without a timing guess.
			publish, selectOld := make(chan struct{}), make(chan struct{})
			published, selected := make(chan struct{}), make(chan struct{})
			go func() { <-publish; c.Warn(newer); close(published) }()
			go func() { <-selectOld; c.selectAttention(1); close(selected) }()
			if publicationFirst {
				close(publish)
				<-published
				close(selectOld)
				<-selected
			} else {
				close(selectOld)
				<-selected
				close(publish)
				<-published
			}
			// Reference model: selecting the old ID clears only that event.
			var reference couchcore.AttentionList
			oldItem, _ := noticeAttention(old, 1)
			newItem, _ := noticeAttention(newer, 2)
			reference = reference.Apply(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: oldItem}, at)
			if publicationFirst {
				reference = reference.Apply(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: newItem}, at)
			}
			reference, _, _ = reference.Select(oldItem.ID)
			if !publicationFirst {
				reference = reference.Apply(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: newItem}, at)
			}
			c.attn.mu.Lock()
			got := c.attn.list.Items()
			c.attn.mu.Unlock()
			if !reflect.DeepEqual(got, reference.Items()) {
				t.Fatalf("events=%+v want=%+v", got, reference.Items())
			}
			if msgs := c.feed.Messages(); len(msgs) != 2 || msgs[1].Body != newer.Body {
				t.Fatalf("feed=%+v", msgs)
			}
		})
	}
}

func TestWarningsBoundedCountAndLifetime(t *testing.T) {
	for _, count := range []int{40, 80} {
		c, _ := continuationConsole(t)
		at := time.Unix(100, 0)
		c.now = func() time.Time { return at }
		for i := range count {
			c.Warn(Notice{Kind: fmt.Sprint(i), Body: fmt.Sprintf("failure %d", i)})
		}
		if c.noticeSequence != uint64(count) || c.attentionCount() != min(count, couchcore.AttentionCap) {
			t.Fatalf("count=%d applications=%d retained=%d", count, c.noticeSequence, c.attentionCount())
		}
		at = at.Add(couchcore.AttentionMaxAge)
		c.expireAttention()
		if c.attentionCount() != 0 {
			t.Fatal("warning outlived existing notification expiry")
		}
	}
}

func TestWarningKeepsExistingAttentionPanelGeometry(t *testing.T) {
	c, _ := continuationConsole(t)
	c.statusRows = 1
	for range 30 {
		c.Warn(Notice{Body: "x"})
	}
	c.toggleAttention()
	c.attn.mu.Lock()
	before := c.attn.box
	c.attn.mu.Unlock()
	// Relayout after a warning must use the same screen as opening the panel.
	c.Warn(Notice{Body: "x"})
	c.attn.mu.Lock()
	after := c.attn.box
	c.attn.mu.Unlock()
	if before.Y != after.Y || before.X != after.X || len(before.lines) != len(after.lines) {
		t.Fatalf("warning changed panel bounds: before=%+v after=%+v", before, after)
	}
}

func TestExpectedExitAndShutdownDoNotRaiseWarnings(t *testing.T) {
	c, _ := continuationConsole(t)
	c.expectedExits.mark("source")
	c.onExit(childExit{id: "source"})
	if c.attentionCount() != 0 {
		t.Fatal("expected exit raised attention")
	}
	c.Stop()
	c.Warn(Notice{Body: "late failure"})
	if c.attentionCount() != 0 {
		t.Fatal("shutdown raised attention")
	}
}

func TestAsyncCastStartFailuresRaiseWarnings(t *testing.T) {
	for _, name := range []string{"screen", "web"} {
		t.Run(name, func(t *testing.T) {
			c, _ := continuationConsole(t)
			if name == "screen" {
				c.bcast = broadcastState{attempt: 1, phase: broadcastStarting}
				c.broadcastStarted(1, nil, errors.New("failed start"), nil)
			} else {
				c.wcast = webcastState{attempt: 1, phase: webcastStarting}
				c.webcastStarted(1, nil, errors.New("failed start"), nil)
			}
			if c.attentionCount() != 1 {
				t.Fatalf("%s failure has no attention", name)
			}
		})
	}
}

func TestUnexpectedExitRaisesExactWarning(t *testing.T) {
	f, threads := pagingFixture(t)
	f.con.exited <- childExit{id: "c2", code: 0}
	waitFor(t, "exit removal", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.panes["c2"] == nil })
	if f.attentionLen() != 1 {
		t.Fatalf("unexpected exit produced %d warnings", f.attentionLen())
	}
	f.con.attn.mu.Lock()
	items := f.con.attn.list.Items()
	f.con.attn.mu.Unlock()
	if items[0].Thread != threads[1] || items[0].Body == "" {
		t.Fatalf("exit lost source: %+v", items)
	}
}

func TestContinuationScanWarningIsOneEpisode(t *testing.T) {
	c, _ := continuationConsole(t)
	failure := continuationScanResult{err: errors.New("discovery unavailable")}
	for range 20 {
		c.acceptContinuationRequests(failure)
	}
	if got := c.attentionCount(); got != 1 {
		t.Fatalf("continuing scan error produced %d warnings, want one", got)
	}
	c.clearAttention()
	c.acceptContinuationRequests(failure)
	if got := c.attentionCount(); got != 0 {
		t.Fatalf("dismissed episode immediately returned: %d", got)
	}
	c.acceptContinuationRequests(continuationScanResult{})
	c.acceptContinuationRequests(failure)
	if got := c.attentionCount(); got != 1 {
		t.Fatalf("new outage did not raise: %d", got)
	}
}

func TestGlobalAndDetachedAttentionOpenSwitcher(t *testing.T) {
	for _, address := range []couchcore.ThreadAddress{{}, {RepoScope: "gone", Tag: "gone"}} {
		f, _ := pagingFixture(t)
		f.con.jumpToAttention(couchcore.AttentionItem{Thread: address, Label: "Couch", Body: "inspect failure"})
		f.con.mu.Lock()
		panel := f.con.focus.IsPanel()
		f.con.mu.Unlock()
		if !panel {
			t.Fatalf("warning for %+v left actor focused", address)
		}
	}
}
