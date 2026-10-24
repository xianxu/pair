package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// A feed publication cannot change geometry, hit spans, or control placement.
// The console supplies the input so this guard survives removal of the former
// StatusModel.Notice field.
func TestConsoleFeedLeavesAllTabBarLayoutsUnchanged(t *testing.T) {
	f := newFixture(t, 24, 80)
	for _, style := range []TabStyle{TabStyleCollapse, TabStyleMultiline} {
		for actors := 0; actors < 25; actors++ {
			f.con.mu.Lock()
			f.con.feed = NewFeed(8, time.Now, NoticeLifetime)
			base := f.con.statusModelLocked()
			base.TabStyle = style
			base.Actors = append(base.Actors[:0:0], hubModel()...)
			for i := 0; i < actors; i++ {
				base.Actors = append(base.Actors, hubActor("repo", "worker", i+1))
			}
			f.con.feed.Push(Notice{Kind: "status", Body: "unrelated feed message"})
			changed := f.con.statusModelLocked()
			changed.TabStyle, changed.Actors = base.TabStyle, base.Actors
			f.con.mu.Unlock()
			for width := 1; width <= 120; width += 7 {
				for rows := 1; rows <= 2; rows++ {
					a, b := RenderStatusBar(width, rows, base), RenderStatusBar(width, rows, changed)
					if !reflect.DeepEqual(a, b) {
						t.Fatalf("style=%v actors=%d width=%d rows=%d: feed changed tab bar: %q => %q", style, actors, width, rows, a.Body(), b.Body())
					}
				}
			}
		}
	}
}

func TestConsoleLatestRetainedFeedAppearsInSwitcher(t *testing.T) {
	now := time.Now()
	f := newFixtureBeforeRun(t, 24, 80, func(c *Console) { c.feed = NewFeed(2, func() time.Time { return now }, NoticeLifetime) })
	f.con.mu.Lock()
	f.con.menu.Notice = MenuNotice{}
	f.con.feed.Push(Notice{Kind: "first", Body: "older feed"})
	f.con.feed.Push(Notice{Kind: "second", Body: "latest retained message"})
	now = now.Add(NoticeLifetime + time.Second)
	f.con.mu.Unlock()
	state := f.con.menuSnapshot()
	if got := RenderMenuView(state, 80, 20, now, false).Body; !strings.Contains(got, "latest retained message") || strings.Contains(got, "older feed") {
		t.Fatalf("latest retained feed missing: %q", got)
	}
	// An operation owns the notice surface until it is cleared; feed updates do
	// not replace its error/progress, and copy feedback remains independently visible.
	for _, notice := range []MenuNotice{errorMenuNotice("operation failed"), progressMenuNotice("operation running"), infoMenuNotice("operation complete")} {
		f.con.mu.Lock()
		f.con.menu.Notice = notice
		f.con.menu.CopyFeedback = CopyFeedback{Kind: CopyFeedbackSuccess, Text: "copy confirmation"}
		f.con.mu.Unlock()
		state = f.con.menuSnapshot()
		got := RenderMenuView(state, 80, 20, now, false).Body
		if !strings.Contains(got, notice.Text) || !strings.Contains(got, "copy confirmation") || strings.Contains(got, "latest retained message") {
			t.Fatalf("operation/copy precedence lost: %q", got)
		}
	}
	f.con.mu.Lock()
	f.con.menu.Notice = MenuNotice{}
	f.con.menu.CopyFeedback = CopyFeedback{}
	f.con.mu.Unlock()
	f.con.onHotkey()
	waitFor(t, "retained feed actually painted in switcher", func() bool { return strings.Contains(f.screenText(), "latest retained message") })
}
