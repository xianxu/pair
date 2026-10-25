package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"testing"
)

func TestNoticeSupervisionRetainsTypedSource(t *testing.T) {
	r, slots := hubMessageRig(t)
	var notices []couchtty.Notice
	output := func(n couchtty.Notice) { notices = append(notices, n) }
	r.s.noticeOutput.Store(&output)
	c := newSupervisionCoordinator(r.s, func(string) { t.Fatal("duplicated legacy feed") })
	c.warn(couchcore.NoticeSupervisionEscalation, slots[2], "needs investigation")
	if len(notices) != 1 || notices[0].Class != couchcore.NoticeSupervisionEscalation || notices[0].Thread != bindingThread(slots[2]) || notices[0].Label != slots[2].Slot {
		t.Fatalf("wrong supervision source: %+v", notices)
	}
}
