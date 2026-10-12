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

	"github.com/xianxu/pair/cmd/internal/broadcast"
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
	srv := httptest.NewServer(NewReplayServer("tok", replayRoot(t, recording.Fixtures...)))
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
	for _, name := range recording.Fixtures {
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
// moment, against the reducer's notes and strokes at each time. The replay
// draws them with the live viewer's modules (#470's annotate.js, #477):
// liveNotes' pins, and strokesAt's strokes painted on a canvas.
func TestLiveReplayServerMarks(t *testing.T) {
	page := headlessPage(t)
	srv := replayServer(t)
	rec, err := recording.ReadDir(recording.Fixture("app"))
	if err != nil {
		t.Fatal(err)
	}
	page.openReplay(t, srv.URL+"/tok/r/app/#t=0")
	painted := 0
	for _, at := range []int64{5900, 6100, 12200, 12400, 12600, 13900, 14100, 16000, 18700, 19100} {
		page.eval(t, fmt.Sprintf("location.hash = '#t=%d'", at))
		page.evalString(t, drawnJS)
		var got struct {
			Notes   []string
			Strokes string
			Inked   bool
		}
		json.Unmarshal(page.eval(t, `(() => {
			const c = document.querySelector('.replay-marks canvas.strokes');
			const px = c.getContext('2d').getImageData(0, 0, c.width || 1, c.height || 1).data;
			let inked = false;
			for (let i = 3; i < px.length; i += 4) if (px[i]) { inked = true; break; }
			return {
				notes: [...document.querySelectorAll('.replay-marks .pin:not([hidden])')].map((e) => e.dataset.note).sort(),
				strokes: c.dataset.strokes || '',
				inked,
			};
		})()`), &got)
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
		// A stroke shows from its first point through FADE_MS after the last
		// point drawn by now, its chunks joined by id.
		end := map[string]int64{}
		for _, s := range st.Strokes {
			if s.Seg < st.DocSeg || s.T > at {
				continue
			}
			for _, p := range s.Pts {
				if pt := s.T + int64(p[2]); pt <= at && pt >= end[s.ID] {
					end[s.ID] = pt
				}
			}
		}
		var strokes []string
		for id, e := range end {
			if at-e < 1500 {
				strokes = append(strokes, id)
			}
		}
		sort.Strings(strokes)
		gotStrokes := strings.Fields(got.Strokes)
		sort.Strings(gotStrokes)
		if !slices.Equal(got.Notes, notes) || !slices.Equal(gotStrokes, strokes) || got.Inked != (len(strokes) > 0) {
			t.Errorf("t=%d: drew notes %v and strokes %v (inked %v); the reducer has %v and %v", at, got.Notes, gotStrokes, got.Inked, notes, strokes)
		}
		if got.Inked {
			painted++
		}
	}
	if painted == 0 {
		t.Fatal("no moment painted a stroke: the fixture's strokes are never checked")
	}
	// A pin's tap opens its note, read-only.
	page.eval(t, "location.hash = '#t=19100'")
	page.evalString(t, drawnJS)
	// The replay is paused, so no frame follows the tap: the card is placed
	// under its pin by the tap itself.
	var card struct {
		Note, Text string
		DX, DY     float64
	}
	json.Unmarshal(page.eval(t, `(() => {
		const chip = document.querySelector('.replay-marks .pin:not([hidden]) .chip');
		if (!chip) return {};
		chip.click();
		const c = document.querySelector('.replay-note-card');
		const pin = chip.parentElement.getBoundingClientRect(), r = c.getBoundingClientRect();
		return { note: c.hidden ? '' : c.dataset.note, text: c.textContent, dx: r.left - pin.left, dy: r.top - pin.bottom };
	})()`), &card)
	if card.Note == "" || !strings.Contains(card.Text, ": ") || card.DX != 0 || card.DY != 4 {
		t.Errorf("tapping a pin: %+v, want the card 4px under its pin", card)
	}
	// A resize while paused redraws the marks: the strokes canvas follows the
	// box rather than stretching to it.
	page.eval(t, "location.hash = '#t=12400'")
	page.evalString(t, drawnJS)
	page.call(t, page.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": 900, "height": 700, "deviceScaleFactor": 1, "mobile": false}, nil)
	defer page.call(t, page.session, "Emulation.clearDeviceMetricsOverride", map[string]any{}, nil)
	// The window's resize event, then two frames: the shell lays out, and
	// the marks follow. Nothing else draws while paused.
	resized := page.evalString(t, `(async () => {
		await new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok)));
		const c = document.querySelector('.replay-marks canvas.strokes'), l = c.parentElement;
		return [c.width, Math.round(l.clientWidth * devicePixelRatio), c.height, Math.round(l.clientHeight * devicePixelRatio)].join(' ');
	})()`)
	if f := strings.Fields(resized); len(f) != 4 || f[0] != f[1] || f[2] != f[3] {
		t.Fatalf("after a resize while paused, the strokes canvas (width height) vs the box: %s", resized)
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

// A replay share (#477) in headless Chrome: the link a running Couch
// prints, through the shared Host (a LocalOnly tunnel here; the real tunnel
// is the operator's smoke test), opens at its #t= moment and seeks.
//
//	WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveReplayShare -v
func TestLiveReplayShareSeeks(t *testing.T) {
	page := headlessPage(t)
	host := broadcast.NewHost(broadcast.HostConfig{Tunnel: broadcast.LocalOnly{}, ProbeTimeout: 5 * time.Second})
	shares := NewReplayShares(host, replayRoot(t, "segments"))
	defer shares.Close()
	st := awaitShare(t, shares, "segments")
	if st.State != ShareReady || !st.Local {
		t.Fatalf("share: %+v", st)
	}
	rec, err := recording.ReadDir(recording.Fixture("segments"))
	if err != nil {
		t.Fatal(err)
	}
	pins := recording.CanonicalTimes(rec)
	page.openReplay(t, ReplayLink(st.Base, "segments", pins[len(pins)-1]))
	if got, want := page.evalString(t, drawnJS), canonicalAt(t, rec, pins[len(pins)-1]); got != want {
		t.Fatalf("opened: %s", firstDiff(got, want))
	}
	for _, at := range []int64{pins[0], pins[len(pins)/2]} {
		page.eval(t, fmt.Sprintf("location.hash = '#t=%d'", at))
		if got, want := page.evalString(t, drawnJS), canonicalAt(t, rec, at); got != want {
			t.Fatalf("seeked to t=%d: %s", at, firstDiff(got, want))
		}
	}
}
