package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/textwidth"
	"math/bits"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestStatusBarWrapsWholeGroupsBeforeHidingActors(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{hubActor("alpha", "a", 0), hubActor("alpha:1", "a", 1), hubActor("beta", "b", 0), hubActor("beta:1", "b", 1)}}
	bar := RenderStatusBar(12, 2, m)
	if bar.Height() != 2 || rowPlain(bar.Rows[0]) != "alpha  :1" || rowPlain(bar.Rows[1]) != "beta  :1" {
		t.Fatalf("wanted complete groups on two rows, got %q", bar.Body())
	}
	if strings.Contains(bar.Body(), "+") {
		t.Fatal("actors hidden despite available second row")
	}
}

func TestStatusBarWideRowsRetainExactLegacyBytesAndSpans(t *testing.T) {
	m := StatusModel{Broadcast: BroadcastLive, Pointer: PointerOn, Webcast: WebcastLive, Attention: 2, Actors: hubModel()}
	m.Actors[2].Active = true
	m.Actors[3].Glyph = "*"
	m.Actors[4].Blocked = true
	got := RenderStatusBar(180, 2, m)
	want := RenderStatusRow(180, m)
	if len(got.Rows) != 1 || !reflect.DeepEqual(got.Rows[0], want) {
		t.Fatalf("wide row changed: got %+v want %+v", got, want)
	}
}

func TestStatusBarOversizedGroupContinuesWithFullRepositoryLabel(t *testing.T) {
	m := StatusModel{Actors: []StatusActor{hubActor("pair", "p", 0), hubActor("pair:1", "p", 1), hubActor("pair:2", "p", 2)}}
	b := RenderStatusBar(9, 2, m)
	if b.Body() != "pair  :1\npair:2" {
		t.Fatalf("continuation lost group identity: %q", b.Body())
	}
	if len(b.Chips) != 3 || b.Chips[2].Row != 1 || b.Chips[2].Start != 0 {
		t.Fatalf("bad row spans: %+v", b.Chips)
	}
}

func TestStatusBarOverflowUsesPriorityAndOneWholeFinalCount(t *testing.T) {
	actors := []StatusActor{hubActor("a", "a", 0), hubActor("b", "b", 0), hubActor("c", "c", 0), hubActor("d", "d", 0), hubActor("e", "e", 0)}
	actors[1].Bell = true
	actors[3].Blocked = true
	actors[4].Active = true
	for width := 1; width <= 20; width++ {
		b := RenderStatusBar(width, 2, StatusModel{Actors: actors})
		visible := map[string]bool{}
		for _, chip := range b.Chips {
			visible[string(chip.Thread.Tag)] = true
		}
		if !visible["e"] {
			t.Fatalf("width%d hid active: %q", width, b.Body())
		}
		if (!visible["b"] || !visible["d"]) && (visible["a"] || visible["c"]) {
			t.Fatalf("width%d kept ordinary before attention: %q", width, b.Body())
		}
		if len(b.Rows) > 2 {
			t.Fatalf("too many rows: %q", b.Body())
		}
		for row, r := range b.Rows {
			plain := rowPlain(r)
			if textwidth.Width(plain) > width {
				t.Fatalf("width%d exceeded: %q", width, plain)
			}
			if strings.Contains(plain, "+") && row != len(b.Rows)-1 {
				t.Fatalf("early count: %q", b.Body())
			}
			if strings.HasSuffix(plain, "+") {
				t.Fatalf("partial count: %q", b.Body())
			}
		}
	}
	b := RenderStatusBar(4, 2, StatusModel{Actors: actors})
	if !strings.Contains(b.Body(), "!") {
		t.Fatalf("hidden attention has no indicator: %q", b.Body())
	}
}

func TestStatusBarTinyOneRowActiveWinsControls(t *testing.T) {
	for width := 1; width <= 20; width++ {
		a := hubActor("active", "g", 0)
		a.Active = true
		b := RenderStatusBar(width, 1, StatusModel{Broadcast: BroadcastLive, Webcast: WebcastLive, Attention: 20, Actors: []StatusActor{hubActor("other", "o", 0), a}})
		found := false
		for _, s := range b.Chips {
			if s.Thread == a.Thread && s.End > s.Start {
				found = true
			}
		}
		if !found {
			t.Fatalf("width%d lost active behind controls: %q", width, b.Body())
		}
		if b.Height() != 1 || textwidth.Width(rowPlain(b.Rows[0])) > width {
			t.Fatalf("bad width/height: %q", b.Body())
		}
	}
}

func TestStatusBarBoundsUnicodeAndInputImmutability(t *testing.T) {
	actors := []StatusActor{hubActor("猫🌍", "g", 0), hubActor("猫🌍:1", "g", 1), hubActor("x\x1b[2J", "x", 0)}
	actors[1].Active = true
	before := append([]StatusActor(nil), actors...)
	for _, rows := range []int{-1, 0, 1, 2, 3} {
		for width := 0; width <= 30; width++ {
			b := RenderStatusBar(width, rows, StatusModel{Actors: actors})
			if (width == 0 || rows <= 0) && b.Height() != 0 {
				t.Fatalf("nonempty zero budget: %+v", b)
			}
			if b.Height() > 2 {
				t.Fatalf("uncapped height: %+v", b)
			}
			for _, r := range b.Rows {
				if textwidth.Width(rowPlain(r)) > width {
					t.Fatalf("Unicode overflow width%d %q", width, r.Body)
				}
			}
		}
	}
	if !reflect.DeepEqual(before, actors) {
		t.Fatal("planner mutated input")
	}
}

func TestStatusBarPlannerCountedBound(t *testing.T) {
	for _, mode := range []string{"kept", "groups", "oversized", "active-last"} {
		var previous statusLayoutWork
		for _, n := range []int{64, 128} {
			actors := make([]StatusActor, n)
			bytes := 0
			for i := range actors {
				group := "g"
				if mode == "groups" {
					group = strconv.Itoa(i)
				}
				actors[i] = hubActor("repository:"+strconv.Itoa(i), group, i)
				actors[i].Blocked = mode == "kept"
				if mode == "active-last" && i == n-1 {
					actors[i].Active = true
				}
				bytes += len(actors[i].Label) + len(actors[i].Glyph)
			}
			work := statusLayoutWork{}
			renderStatusBar(24, 2, StatusModel{Actors: actors}, &work)
			if work.Bytes != bytes {
				t.Fatalf("%s: rescanned labels: %+v want %d", mode, work, bytes)
			}
			t.Logf("%s N%d labelBytes=%d actorVisits=%d", mode, n, work.Bytes, work.Visits)
			bound := n * (12 + 4*bits.Len(uint(n)))
			if work.Visits > bound {
				t.Fatalf("%s N%d visits%d bound%d", mode, n, work.Visits, bound)
			}
			if n == 128 && work.Visits > previous.Visits*3 {
				t.Fatalf("%s superlinear beyond binary-search bound: %d to %d", mode, previous.Visits, work.Visits)
			}
			previous = work
		}
	}
}

func FuzzStatusBarBoundsAndActive(f *testing.F) {
	f.Add([]byte{2, 3, 4, 5}, uint8(12), uint8(2))
	f.Add([]byte{255, 0, 1, 5, 7, 9}, uint8(1), uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, w, rows uint8) {
		if len(data) > 128 {
			data = data[:128]
		}
		width := int(w%80) + 1
		maxRows := int(rows%2) + 1
		actors := make([]StatusActor, len(data))
		for i, v := range data {
			actors[i] = hubActor(strings.Repeat("x", int(v%15)+1), strconv.Itoa(int(v%4)), i)
			actors[i].Thread = menuAddress(strconv.Itoa(i))
			actors[i].Blocked = v&16 != 0
			actors[i].Hub = v&32 != 0
			actors[i].Bell = v&64 != 0
		}
		if len(actors) > 0 {
			actors[len(actors)-1].Active = true
		}
		b := RenderStatusBar(width, maxRows, StatusModel{Actors: actors, Broadcast: BroadcastCell(rows % 3), Attention: int(w % 6)})
		if b.Height() > maxRows {
			t.Fatalf("height %d > %d", b.Height(), maxRows)
		}
		found := len(actors) == 0
		for r, row := range b.Rows {
			if textwidth.Width(rowPlain(row)) > width {
				t.Fatalf("overflow %q", row.Body)
			}
			for _, s := range row.Chips {
				if s.Row != r || s.Start < 0 || s.End <= s.Start || s.End > width {
					t.Fatalf("invalid span %+v", s)
				}
				if len(actors) > 0 && s.Thread == actors[len(actors)-1].Thread {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("active hidden width%d rows%d: %q", width, maxRows, b.Body())
		}
	})
}

func TestStatusBarOverflowCountStaysAtEnd(t *testing.T) {
	actors := make([]StatusActor, 30)
	for i := range actors {
		actors[i] = hubActor("repo"+strconv.Itoa(i), strconv.Itoa(i), 0)
	}
	for width := 10; width < 60; width++ {
		b := RenderStatusBar(width, 2, StatusModel{Actors: actors})
		plain := rowPlain(b.Rows[len(b.Rows)-1])
		at := strings.LastIndex(plain, "+")
		if at < 0 {
			t.Fatalf("no count: %q", b.Body())
		}
		if _, err := strconv.Atoi(plain[at+1:]); err != nil {
			t.Fatalf("text followed overflow count: %q", b.Body())
		}
	}
}
