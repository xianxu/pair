package webcast

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// The replay (#471) in headless Chrome, through ReplayServer: a #t= link
// and scrubbing draw the reducer's canonical state at every pin; the JS
// seek rule takes the records Go's CatchUp takes; notes and strokes show
// at their moments; who drove is on the timeline. Like the other live
// checks it needs a real Chrome outside the agent sandbox:
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveReplayServer -v

func replayServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewReplayServer("tok", replayRoot(t, recording.Fixtures()...)))
	t.Cleanup(srv.Close)
	return srv
}

// openReplay opens a replay link and waits for its records to be drawn.
func (p *chromePage) openReplay(t *testing.T, url string) {
	t.Helper()
	p.call(t, p.session, "Page.navigate", map[string]any{"url": url}, nil)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if p.evalString(t, "String(!!(window.couchViewer && document.querySelector('.replay-track')))") == "true" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s never drew its timeline", url)
}

// drawn is the page's canonical once it has settled.
const drawnJS = `(async () => {
	await couchViewer.ready();
	await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok)));
	const { goJSON } = await import('./model.js');
	return goJSON(couchViewer.canonical());
})()`

func TestLiveReplayServerSeeks(t *testing.T) {
	page := headlessPage(t)
	srv := replayServer(t)
	page.call(t, page.session, "Network.enable", map[string]any{}, nil)
	for _, name := range recording.Fixtures() {
		t.Run(name, func(t *testing.T) {
			rec, err := recording.ReadDir(recording.Fixture(name))
			if err != nil {
				t.Fatal(err)
			}
			pins := recording.CanonicalTimes(rec)
			// A #t= link opens at its moment.
			last := pins[len(pins)-1]
			page.openReplay(t, fmt.Sprintf("%s/tok/r/%s/#t=%d", srv.URL, name, last))
			if got := page.evalString(t, drawnJS); got != canonicalAt(t, rec, last) {
				t.Fatalf("opened at t=%d: %s", last, firstDiff(got, canonicalAt(t, rec, last)))
			}
			// Scrubbing, in an order no replay would take.
			order := slices.Clone(pins)
			rand.New(rand.NewSource(471)).Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
			for _, at := range order {
				page.eval(t, fmt.Sprintf("location.hash = '#t=%d'", at))
				if got, want := page.evalString(t, drawnJS), canonicalAt(t, rec, at); got != want {
					t.Fatalf("scrubbed to t=%d: %s", at, firstDiff(got, want))
				}
			}
			// The JS seek rule is Go's CatchUp, record for record.
			for _, at := range pins {
				var idx []int
				json.Unmarshal(page.eval(t, fmt.Sprintf(`(async () => {
					const { catchUp } = await import('./replay.js');
					const recs = (await (await fetch('records')).text()).split('\n').filter(Boolean).map((l) => JSON.parse(l));
					const upto = recs.filter((r) => r.t <= %d);
					return catchUp(upto).map((r) => upto.indexOf(r));
				})()`, at)), &idx)
				if want := catchUpIndexes(rec.Records, at); !slices.Equal(idx, want) {
					t.Fatalf("t=%d: JS catch-up takes %v, Go's %v", at, idx, want)
				}
			}
		})
	}
	for _, u := range page.requests() {
		// css.js's BLOCKED (about:invalid): a URL with no asset loads nothing.
		if !strings.HasPrefix(u, srv.URL+"/") && !strings.HasPrefix(u, "blob:") && !strings.HasPrefix(u, "data:") && u != "about:blank" && u != "about:invalid" {
			t.Errorf("the replay requested %s, outside its server", u)
		}
	}
}

// catchUpIndexes is which records Go's CatchUp takes from those up to at.
func catchUpIndexes(recs []recording.Record, at int64) []int {
	var upto []recording.Record
	for _, r := range recs {
		if r.Time() <= at {
			upto = append(upto, r)
		}
	}
	var idx []int
	for _, r := range recording.CatchUp(upto) {
		idx = append(idx, slices.Index(upto, r))
	}
	return idx
}

// Notes are pinned while open on their document, strokes drawn for their
// moment, against the reducer's notes and strokes at each time.
func TestLiveReplayServerMarks(t *testing.T) {
	page := headlessPage(t)
	srv := replayServer(t)
	rec, err := recording.ReadDir(recording.Fixture("rec-app"))
	if err != nil {
		t.Fatal(err)
	}
	page.openReplay(t, srv.URL+"/tok/r/rec-app/#t=0")
	for _, at := range []int64{5900, 6100, 12200, 12400, 12600, 13900, 14100, 16000, 18700, 19100} {
		page.eval(t, fmt.Sprintf("location.hash = '#t=%d'", at))
		page.evalString(t, drawnJS)
		var got struct {
			Notes   []string
			Strokes int
		}
		json.Unmarshal(page.eval(t, `({
			notes: [...document.querySelectorAll('.replay-note')].map((e) => e.dataset.note).sort(),
			strokes: document.querySelectorAll('.replay-strokes polyline').length,
		})`), &got)
		st, err := recording.Reduce(rec.Records, at)
		if err != nil {
			t.Fatal(err)
		}
		var notes []string
		for _, n := range st.Notes {
			if n.Seg >= st.DocSeg && n.State == recording.NoteOpen {
				notes = append(notes, n.ID)
			}
		}
		sort.Strings(notes)
		strokes := 0
		for _, s := range st.Strokes {
			since := at - s.T
			if s.Seg >= st.DocSeg && since >= 0 && since <= int64(s.Pts[len(s.Pts)-1][2])+1500 {
				strokes++
			}
		}
		if !slices.Equal(got.Notes, notes) || got.Strokes != strokes {
			t.Errorf("t=%d: drew notes %v and %d strokes; the reducer has %v and %d", at, got.Notes, got.Strokes, notes, strokes)
		}
	}
}

// Who drove is on the timeline, by name.
func TestLiveReplayServerDrivers(t *testing.T) {
	page := headlessPage(t)
	srv := replayServer(t)
	page.openReplay(t, srv.URL+"/tok/r/driven/")
	var titles []string
	json.Unmarshal(page.eval(t, `[...document.querySelectorAll('.replay-drive')].map((e) => e.title)`), &titles)
	if want := []string{"Ana drove", "Bo drove", "Cy drove"}; !slices.Equal(titles, want) {
		t.Errorf("drive bands %v, want %v", titles, want)
	}
	var gaps int
	json.Unmarshal(page.eval(t, `document.querySelectorAll('.replay-gap').length`), &gaps)
	if gaps != 1 {
		t.Errorf("%d gaps, want the one pause", gaps)
	}
}
