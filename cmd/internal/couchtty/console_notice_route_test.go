package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestConsoleNoticeHandoffDoesNotProjectOrRepeatFeed(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	var handed []Notice
	c.SetNoticeRouter(func(n Notice) { handed = append(handed, n) })
	c.Warn(Notice{Kind: "trace", Body: "trace file unavailable"})
	if len(handed) != 1 || len(c.feed.Messages()) != 1 || c.attn.list.Len() != 0 {
		t.Fatalf("handoff=%d feed=%d bell=%d", len(handed), len(c.feed.Messages()), c.attn.list.Len())
	}
	c.ApplyNoticeAttention(handed[0], "occurrence")
	c.ApplyNoticeAttention(Notice{Body: "updated diagnostic"}, "occurrence")
	if len(c.feed.Messages()) != 1 || c.attn.list.Len() != 1 {
		t.Fatalf("projection duplicated feed or bell: feed=%d bell=%d", len(c.feed.Messages()), c.attn.list.Len())
	}
	if got := c.attn.list.Items()[0].Body; got != "updated diagnostic" {
		t.Fatalf("same occurrence body = %q", got)
	}
}

func TestConsoleNoticeProjectionStopsWithLifetime(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	c.Stop()
	c.ApplyNoticeAttention(Notice{Body: "late result"}, "late")
	if c.attn.list.Len() != 0 {
		t.Fatal("late route projected after stop")
	}
}
