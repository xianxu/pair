package terminal

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func TestHistoryPublicationHeldTogetherAndOwned(t *testing.T) {
	e, err := NewEndpoint("history", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	now := time.Unix(100, 0)
	e.Feed([]byte("ABCDEFGHI"), now)
	before, err := e.Publication(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.History.Rows) != 1 || !before.History.ContinuesToScreen || !before.Frame.Rows[0].Wrapped {
		t.Fatalf("publication %+v", before)
	}
	e.Feed([]byte("\x1b[?2026h\r\nJ\r\nK\x1b[3J"), now)
	held, err := e.Publication(now.Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, held) {
		t.Fatal("history and frame not held atomically")
	}
	held.History.Rows[0].Cells[0].Content = "changed"
	held.Frame.Rows[0].Wrapped = false
	again, _ := e.Publication(now.Add(time.Millisecond))
	if !reflect.DeepEqual(before, again) {
		t.Fatal("publication aliases endpoint")
	}
	after, _ := e.Publication(now.Add(SyncTimeout))
	if len(after.History.Rows) != 0 || after.History.Cursor.ClearEpoch == before.History.Cursor.ClearEpoch {
		t.Fatal("clear not released with sync frame")
	}
}

func TestHistoryPublicationSurvivesEOF(t *testing.T) {
	e, err := NewEndpoint("history", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	now := time.Unix(100, 0)
	e.Feed([]byte("\x1b[?2026hABCDEFGHI"), now)
	e.EndInput()
	p, err := e.Publication(now)
	if err != nil || len(p.History.Rows) != 1 || p.Frame.Cells[0].Content != "E" {
		t.Fatalf("EOF publication %+v %v", p, err)
	}
}

func TestComposeMetadataAndNoChrome(t *testing.T) {
	child := Frame{EndpointID: "one", Geometry: Geometry{2, 1}, Cells: []Cell{{Content: "A", Width: 1}, {Width: 1}}, Rows: []RowMetadata{{Wrapped: true, UsedColumns: 1}}}
	same, err := Compose(child, Geometry{2, 1}, nil)
	if err != nil || !same.Rows[0].Wrapped || len(same.Cells) != 2 {
		t.Fatalf("no chrome %+v %v", same, err)
	}
	chrome, err := Compose(child, Geometry{2, 2}, []Cell{{Content: "C", Width: 1}, {Width: 1}})
	if err != nil || len(chrome.Rows) != 2 || chrome.Rows[1].Wrapped || chrome.Rows[1].UsedColumns != 1 {
		t.Fatalf("chrome %+v %v", chrome, err)
	}
	same.Rows[0].Wrapped = false
	if !child.Rows[0].Wrapped {
		t.Fatal("compose aliases metadata")
	}
}

func TestPublicationRetainsHistoryOnlyForHeldOrEndedOutput(t *testing.T) {
	e, err := NewEndpoint("history-retention", Geometry{4, 2}, ttyio.NewFake())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	e.Feed([]byte("A\r\nB\r\nC"), time.Time{})
	p, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.History.Rows) == 0 {
		t.Fatal("publication missing history")
	}
	if len(e.publishedHistory.Rows) != 0 {
		t.Fatal("ordinary publication retained a duplicate history window")
	}
	e.Feed([]byte("\x1b[?2026hD"), time.Time{})
	held, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(e.publishedHistory.Rows) == 0 {
		t.Fatal("held publication did not retain immutable history")
	}
	held.History.Rows[0].Cells[0].Content = "changed"
	again, err := e.Publication(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if again.History.Rows[0].Cells[0].Content == "changed" {
		t.Fatal("held history aliases caller")
	}
	e.Feed([]byte("\x1b[?2026l"), time.Time{})
	e.Publication(time.Time{})
	if len(e.publishedHistory.Rows) != 0 {
		t.Fatal("released hold retained frozen history")
	}
}
