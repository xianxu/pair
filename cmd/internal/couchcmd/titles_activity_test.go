package couchcmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

func TestCouchTitlesWrapperActivityFeedsTitlesAndFading(t *testing.T) {
	row := couchTitleRow("a")
	work := time.Unix(1000, 0)
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Count: "42k", Activity: work.Add(-time.Hour)}}, polls: map[string]int{}}
	u := newCouchTitleUpdater("unused", "unused", "workspace", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	activity, failed := u.UpdateWithActivity(context.Background(), []couchcore.ActionableThreadSummary{row}, map[couchcore.ThreadAddress]time.Time{row.Address: work})
	if failed[row.Address] || !activity[row.Address].Equal(work) {
		t.Fatalf("fading activity=%v failed=%v", activity, failed)
	}
	if len(f.writes) != 2 || !strings.HasSuffix(f.writes[0], ":codex (42k)") || f.writes[1] != "workspace:"+titlepoller.CmuxWorkspaceTitle(titlepoller.PrefixForAge(0), row.TitleTarget.Session) {
		t.Fatalf("wrapper work did not drive frame and workspace heat: %v", f.writes)
	}
	u.UpdateWithActivity(context.Background(), []couchcore.ActionableThreadSummary{row}, map[couchcore.ThreadAddress]time.Time{row.Address: work})
	if len(f.writes) != 2 {
		t.Fatalf("stable shared activity rewrote titles: %v", f.writes)
	}
}
