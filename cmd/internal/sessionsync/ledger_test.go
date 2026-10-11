package sessionsync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func TestLedgerRoundTripsPerSlot(t *testing.T) {
	l := &Ledger{Dir: t.TempDir()}
	if err := l.Append("ops:0", Entry{ID: "a", From: "pair:5", At: t0, Head: "hello\nsecond line"}); err != nil {
		t.Fatal(err)
	}
	if err := l.Append("ops:1", Entry{ID: "b", From: "pair:5", At: t0}); err != nil {
		t.Fatal(err)
	}
	rows, err := l.Read("ops:0")
	if err != nil || len(rows) != 1 || rows[0].ID != "a" || rows[0].Head != "hello" || !rows[0].At.Equal(t0) {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if _, err := os.Stat(filepath.Join(l.Dir, "ops%3A0.jsonl")); err != nil {
		t.Fatal(err)
	}
	if rows, err := l.Read("nobody:3"); err != nil || rows != nil {
		t.Fatalf("a slot with no file has no deliveries: %v %v", rows, err)
	}
}

func TestLedgerRefusesAddressesThatEscapeTheDir(t *testing.T) {
	l := &Ledger{Dir: t.TempDir()}
	for _, slot := range []string{"../x:0", "a/b:0", "ops", "ops:0/..", ":0", "ops:01", "ops:-1", " ops:0"} {
		if err := l.Append(slot, Entry{ID: "x"}); err == nil {
			t.Fatalf("%q must refuse", slot)
		}
	}
}

func TestLedgerSkipsATornTail(t *testing.T) {
	l := &Ledger{Dir: t.TempDir()}
	if err := l.Append("ops:0", Entry{ID: "a", At: t0}); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(filepath.Join(l.Dir, "ops%3A0.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(`{"ID":"b","Fr`)
	f.Close()
	rows, err := l.Read("ops:0")
	if err != nil || len(rows) != 1 || rows[0].ID != "a" {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
}

func TestLedgerTrimKeepsNewestYoungRows(t *testing.T) {
	now := t0.Add(30 * 24 * time.Hour)
	pad := strings.Repeat("p", 200)
	fill := func(l *Ledger, at func(i int) time.Time) int {
		n := 0
		for {
			p := filepath.Join(l.Dir, "ops%3A0.jsonl")
			if info, err := os.Stat(p); err == nil && info.Size() > trimBytes-400 {
				break
			}
			if err := l.Append("ops:0", Entry{ID: fmt.Sprint(n), At: at(n), Head: pad}); err != nil {
				t.Fatal(err)
			}
			n++
		}
		return n
	}
	// Old rows, then young rows; the append that crosses the size trims.
	l := &Ledger{Dir: t.TempDir(), Now: func() time.Time { return now }}
	n := fill(l, func(i int) time.Time {
		if i < 600 {
			return t0
		}
		return now.Add(-time.Hour)
	})
	for i := 0; i < 3; i++ {
		_ = l.Append("ops:0", Entry{ID: fmt.Sprint(n + i), At: now, Head: pad})
	}
	rows, _ := l.Read("ops:0")
	if len(rows) == 0 || len(rows) > keepRows || rows[0].At.Equal(t0) {
		t.Fatalf("old rows must go and the cap hold: kept %d, first %v", len(rows), rows[0].At)
	}
	if last := rows[len(rows)-1].ID; last != fmt.Sprint(n+2) {
		t.Fatalf("the append after a trim must land in the new file: last %s", last)
	}
	entries, _ := os.ReadDir(l.Dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestLedgerConcurrentAppendsLoseNothing(t *testing.T) {
	l := &Ledger{Dir: t.TempDir()}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if err := l.Append("ops:0", Entry{ID: fmt.Sprintf("%d-%d", g, i), At: t0}); err != nil {
					t.Error(err)
				}
			}
		}(g)
	}
	wg.Wait()
	if rows, _ := l.Read("ops:0"); len(rows) != 400 {
		t.Fatalf("got %d rows", len(rows))
	}
}

func TestHeadTruncatesButKeepsASubmitPath(t *testing.T) {
	if got := Head(strings.Repeat("x", 300) + "\nrest"); len(got) != headMax {
		t.Fatalf("len %d", len(got))
	}
	long := SubmitPrefix + "/" + strings.Repeat("d/", 400) + "f.md"
	if got := Head(long + "\nrest"); got != long {
		t.Fatalf("a submit path must survive: len %d", len(got))
	}
}

func TestLedgerUsesCouchsSlotGrammar(t *testing.T) {
	l := &Ledger{Dir: t.TempDir()}
	for _, slot := range []string{"parley.nvim:1", "проект:0", "a b:2"} {
		if !ValidSlot(slot) {
			continue // the grammar is couch's; only assert what it accepts
		}
		if err := l.Append(slot, Entry{ID: "x", At: t0}); err != nil {
			t.Fatalf("%q: %v", slot, err)
		}
		if rows, err := l.Read(slot); err != nil || len(rows) != 1 {
			t.Fatalf("%q: %v %v", slot, rows, err)
		}
	}
	if !ValidSlot("проект:0") || !ValidSlot("parley.nvim:1") {
		t.Fatal("couch accepts these families; the ledger must too")
	}
}

func TestLedgerTrimEndsWellUnderTheBound(t *testing.T) {
	now := t0
	l := &Ledger{Dir: t.TempDir(), Now: func() time.Time { return now }}
	long := SubmitPrefix + "/" + strings.Repeat("d", 3000)
	p := filepath.Join(l.Dir, "ops%3A0.jsonl")
	for i := 0; ; i++ {
		_ = l.Append("ops:0", Entry{ID: fmt.Sprint(i), At: now, Head: long})
		if info, _ := os.Stat(p); info.Size() < trimBytes/2+4096 && i > 100 {
			break
		}
		if i > 1000 {
			t.Fatal("the trim never brought the file under half the bound")
		}
	}
}
