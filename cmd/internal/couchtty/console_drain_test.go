package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestConsoleSetDrainBoardCopiesAndCoalesces(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	yes := true
	millis := int64(1200)
	b := couchcore.DrainBoard{Rows: []couchcore.DrainRow{{Workspace: "pair:1", Connected: true, IdleMillis: &millis, Settled: &yes, Picker: &yes, Background: &yes}}}
	c.SetDrainBoard(b)
	b.Rows[0].Workspace = "changed"
	yes = false
	millis = 99
	got := <-c.drainUpdates
	if got.Rows[0].Workspace != "pair:1" || !*got.Rows[0].Settled || !*got.Rows[0].Picker || !*got.Rows[0].Background || *got.Rows[0].IdleMillis != 1200 {
		t.Fatal("mutable publication", got)
	}
	for i := 0; i < 100; i++ {
		b.Rows[0].Workspace = fmt.Sprint(i)
		c.SetDrainBoard(b)
	}
	if len(c.drainUpdates) != 1 || (<-c.drainUpdates).Rows[0].Workspace != "99" {
		t.Fatal("publication not coalesced")
	}
}

func TestConsoleDrainEventPreservesPaneAndPicker(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	active, focus := f.con.active, f.con.focus
	f.con.menu.Blocked = map[couchcore.ThreadAddress]bool{{RepoScope: "scope"}: true}
	f.con.mu.Unlock()
	f.con.SetDrainBoard(couchcore.DrainBoard{Rows: []couchcore.DrainRow{{Workspace: "pair:1"}}})
	deadline := time.Now().Add(time.Second)
	for {
		f.con.mu.Lock()
		seen := len(f.con.drainBoard.Rows) == 1
		same := f.con.active == active && f.con.focus == focus && len(f.con.menu.Blocked) == 1
		f.con.mu.Unlock()
		if seen {
			if !same {
				t.Fatal("drain changed selected pane or picker")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("drain event not consumed")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestDrainStatusCompactCountBoundsAndSelection(t *testing.T) {
	for _, count := range []int{8, 16} {
		board := couchcore.DrainBoard{Rows: make([]couchcore.DrainRow, count)}
		m := StatusModel{Actors: []StatusActor{{Label: "pair:1", Active: true, Blocked: true, Thread: couchcore.ThreadAddress{RepoScope: "one"}}, {Label: "pair:2", Thread: couchcore.ThreadAddress{RepoScope: "two"}}}}
		applyDrainStatus(&m, board)
		want := fmt.Sprintf("drain %d", count)
		if m.Drain != want {
			t.Fatalf("badge %q", m.Drain)
		}
		for width := 1; width < 140; width++ {
			row := RenderStatusRow(width, m)
			plain := string(ansi.Strip([]byte(row.Body)))
			if textwidth.Width(plain) > width {
				t.Fatalf("width %d: %q", width, plain)
			}
		}
		row := RenderStatusRow(100, m)
		plain := string(ansi.Strip([]byte(row.Body)))
		if strings.Count(plain, "drain") != 1 || !strings.Contains(plain, want) || strings.ContainsAny(plain, "{}") || !strings.Contains(plain, "[pair:1]") {
			t.Fatal(plain)
		}
		if !strings.Contains(row.Body, blockedSGR) {
			t.Fatal("picker cue lost")
		}
		if len(row.Chips) != 2 {
			t.Fatalf("selection spans lost: %+v", row)
		}
		applyDrainStatus(&m, couchcore.DrainBoard{})
		if m.Drain != "" {
			t.Fatal("cleared flags retain badge")
		}
	}
}
