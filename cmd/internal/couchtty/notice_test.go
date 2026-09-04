package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestExitNoticeIsPerActorControlAndNamesCause(t *testing.T) {
	n := ExitNotice(couchcore.ActorID("couch-b1"), "brain", 17)
	m := n.Message()
	if m.Kind != "exit:couch-b1" {
		t.Fatalf("kind = %q, want exit:couch-b1", m.Kind)
	}
	if !m.Control {
		t.Fatal("exit notice is not control priority")
	}
	for _, want := range []string{"brain", "couch-b1", "17"} {
		if !strings.Contains(m.Body, want) {
			t.Fatalf("body %q does not name %q", m.Body, want)
		}
	}
}

func TestFeedNeverDropsExitUnderCapacityPressure(t *testing.T) {
	f := NewFeed(1, time.Now, NoticeLifetime)
	f.Push(ExitNotice("couch-b1", "brain", 1))
	ok := f.Push(ExitNotice("couch-p2", "pair", 2))
	got := f.Messages()
	if ok {
		t.Fatal("control-only capacity overflow was not reported")
	}
	if len(got) != 2 || got[0].Kind != "exit:couch-b1" || got[1].Kind != "exit:couch-p2" {
		t.Fatalf("messages = %+v, want both exit controls retained", got)
	}
}

// The operator's report: "previous: nowhere to return to" sat on the status row
// long after the keystroke it answered, reading as current state. Nothing
// retired a notice at all -- no timer, no clear on keystroke, no expiry.
func TestATransientNoticeStopsStandingAndAControlOneDoesNot(t *testing.T) {
	clock := time.Unix(1000, 0)
	feed := NewFeed(8, func() time.Time { return clock }, NoticeLifetime)

	feed.Push(Notice{Kind: "status", Body: "previous: nowhere to return to"})
	row := feed.Row()
	if row.Body != "previous: nowhere to return to" {
		t.Fatalf("row = %q", row.Body)
	}
	if row.Expires.IsZero() {
		t.Fatal("a transient notice reported no expiry, so nothing can schedule its removal")
	}

	clock = clock.Add(NoticeLifetime - time.Second)
	if feed.Row().Body == "" {
		t.Fatal("the notice went before its lifetime elapsed")
	}
	clock = clock.Add(2 * time.Second)
	if got := feed.Row(); got.Body != "" {
		t.Fatalf("an expired notice is still standing: %q", got.Body)
	}

	// An exit is an OBLIGATION: it says why a pane disappeared, and it stands
	// until something displaces it.
	feed.Push(ExitNotice("couch-b1", "brain", 1))
	clock = clock.Add(100 * NoticeLifetime)
	row = feed.Row()
	if !strings.Contains(row.Body, "exited (1)") {
		t.Fatalf("a control notice expired: %q", row.Body)
	}
	if !row.Expires.IsZero() {
		t.Fatalf("a control notice reported an expiry: %v", row.Expires)
	}
}

// An older transient is staler than the one that just expired, so it is never a
// better answer -- but a control notice underneath IS, because it is still true.
func TestAnExpiredNoticeUncoversOnlyWhatIsStillTrue(t *testing.T) {
	clock := time.Unix(1000, 0)
	feed := NewFeed(8, func() time.Time { return clock }, NoticeLifetime)

	feed.Push(ExitNotice("couch-b1", "brain", 1))
	feed.Push(Notice{Kind: "status", Body: "previous: nowhere to return to"})
	clock = clock.Add(NoticeLifetime + time.Second)
	if got := feed.Row().Body; !strings.Contains(got, "exited (1)") {
		t.Fatalf("row after the transient expired = %q, want the exit still showing", got)
	}

	feed2 := NewFeed(8, func() time.Time { return clock }, NoticeLifetime)
	feed2.Push(Notice{Kind: "first", Body: "an older refusal"})
	feed2.Push(Notice{Kind: "second", Body: "a newer refusal"})
	clock = clock.Add(NoticeLifetime + time.Second)
	if got := feed2.Row().Body; got != "" {
		t.Fatalf("an older transient was uncovered: %q", got)
	}
}

// Enqueue collapses by Kind, so a replaced notice must inherit the expiry slot
// rather than leaving the old deadline behind to retire the new message early.
func TestAReplacedNoticeGetsAFreshLifetime(t *testing.T) {
	clock := time.Unix(1000, 0)
	feed := NewFeed(8, func() time.Time { return clock }, NoticeLifetime)

	feed.Push(Notice{Kind: "status", Body: "first"})
	clock = clock.Add(NoticeLifetime - time.Second)
	feed.Push(Notice{Kind: "status", Body: "second"})
	clock = clock.Add(2 * time.Second)

	if got := feed.Row().Body; got != "second" {
		t.Fatalf("row = %q, want the replacement still standing on its own lifetime", got)
	}
}

// The bounded queue must not grow an unbounded shadow.
func TestExpiryBookkeepingStaysWithinCapacity(t *testing.T) {
	clock := time.Unix(1000, 0)
	feed := NewFeed(2, func() time.Time { return clock }, NoticeLifetime)
	for i := 0; i < 50; i++ {
		feed.Push(Notice{Kind: fmt.Sprintf("kind-%d", i), Body: "x"})
	}
	if len(feed.expiry) > 2 {
		t.Fatalf("expiry map holds %d entries for a capacity-2 feed", len(feed.expiry))
	}
}
