package couchtty

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// A crash report is an obligation, not an event: it must stand on the status
// row until something displaces it, or a slow first paint could expire it
// unseen (#397).
func TestNotifyPublishesAStandingControlNotice(t *testing.T) {
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), nil)
	con.Notify(Notice{Kind: "crash x", Control: true, Body: "previous couch crashed — see x"})
	row := con.feed.Row()
	if row.Body != "previous couch crashed — see x" || !row.Expires.IsZero() {
		t.Fatalf("status row = %+v, want a standing crash notice", row)
	}
}
