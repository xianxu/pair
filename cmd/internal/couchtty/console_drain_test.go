package couchtty

import (
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
	b := couchcore.DrainBoard{ID: "id", State: couchcore.DrainStuck, Detail: "queue unknown", Rows: []couchcore.DrainRow{{Workspace: "pair:1", State: couchcore.DrainStuck}}, Queue: couchcore.DrainQueueObservation{Owners: []string{"owner"}, Active: []couchcore.DrainActivity{{ID: "a"}}}}
	c.SetDrainBoard(b)
	b.Rows[0].State = couchcore.DrainDrained
	b.Queue.Owners[0] = "other"
	b.Queue.Active[0].ID = "other"
	got := <-c.drainUpdates
	if got.Rows[0].State != couchcore.DrainStuck || got.Queue.Owners[0] != "owner" || got.Queue.Active[0].ID != "a" {
		t.Fatal("mutable publication", got)
	}
	for i := 0; i < 100; i++ {
		b.Detail = "latest"
		c.SetDrainBoard(b)
	}
	if len(c.drainUpdates) != 1 || (<-c.drainUpdates).Detail != "latest" {
		t.Fatal("publication not coalesced")
	}
}

func TestConsoleDrainEventPreservesPaneAndPicker(t *testing.T) {
	f := newFixture(t, 24, 100)
	f.con.mu.Lock()
	active := f.con.active
	focus := f.con.focus
	f.con.menu.Blocked = map[couchcore.ThreadAddress]bool{{RepoScope: "scope"}: true}
	f.con.mu.Unlock()
	f.con.SetDrainBoard(couchcore.DrainBoard{ID: "id", State: couchcore.DrainStuck, Detail: "queue unknown"})
	deadline := time.Now().Add(time.Second)
	for {
		f.con.mu.Lock()
		seen := f.con.drainBoard.ID == "id"
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

func TestDrainStatusStateReasonAndBounds(t *testing.T) {
	m := StatusModel{Drain: "stuck: queue unknown", Actors: []StatusActor{{Label: "pair:1", Active: true, Blocked: true, Drain: couchcore.DrainStuck}, {Label: "pair:2", Drain: couchcore.DrainDrained}}}
	for width := 1; width < 140; width++ {
		r := RenderStatusRow(width, m)
		plain := string(ansi.Strip([]byte(r.Body)))
		if textwidth.Width(plain) > width {
			t.Fatalf("width %d: %q", width, plain)
		}
		if width >= 10 && !strings.Contains(plain, "stuck") {
			t.Fatalf("state hidden width %d: %q", width, plain)
		}
	}
	r := RenderStatusRow(120, m)
	plain := string(ansi.Strip([]byte(r.Body)))
	for _, want := range []string{"stuck: queue unknown", "[pair:1{stuck}]", "pair:2{drained}"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("missing %q in %q", want, plain)
		}
	}
	if !strings.Contains(r.Body, blockedSGR) {
		t.Fatal("picker indicator styling lost")
	}
}

func TestDrainStatusSanitizesReason(t *testing.T) {
	s := drainStatusText(couchcore.DrainBoard{ID: "id", State: couchcore.DrainStuck, Detail: "queue\x1b[2J unknown\nreason"})
	if strings.ContainsAny(s, "\x1b\n") {
		t.Fatalf("unsafe reason %q", s)
	}
	if drainStatusText(couchcore.DrainBoard{}) != "" {
		t.Fatal("inactive drain rendered")
	}
}
