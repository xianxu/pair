package recording

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TB is the part of testing.TB a contract suite uses, so this package
// doesn't import testing.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// ContractRecording is a small recording with two documents, a pause and a
// resume, assets and notes: the input of the live contract, and handy for
// other components' tests.
func ContractRecording(changes int) []Record {
	b := NewBuilder("contract")
	b.Add(&Asset{URL: "http://x/a.png", Hash: HashOf([]byte("a")), MIME: "image/png", Size: 1})
	div := b.El("div", nil, b.Txt("0"))
	doc := b.Page(SegLive, "http://x/", div)
	b.Add(&Note{ID: "n1", Author: Author{ID: "u"}, Text: "first", At: NoteAt{Node: div.ID}})
	for i := 0; i < changes; i++ {
		b.At(int64(10 + i)).Add(&Text{ID: div.C[0].ID, V: fmt.Sprint(i + 1)})
	}
	b.Add(&Stop{Why: StopPause})
	b.Add(&Seg{Seg: b.NextSeg(), Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://x/"}, VP: Viewport{W: 1280, H: 800, DPR: 1}})
	b.Snapshot("http://x/", doc.C[1])
	p := b.El("p", nil)
	b.Page(SegNavigate, "http://x/2", p)
	b.Add(&Note{ID: "n2", Author: Author{ID: "u"}, At: NoteAt{Node: p.ID}})
	b.Add(&Stop{Why: StopEnd})
	return b.Recs
}

// DrivenRecording is a small recording in which viewers drive (#466): a
// request and a grant, the driver's attributed clicks and typing, a pause and
// a resume and a navigation while driving, a release, a second driver
// revoked, a denied request, a viewer leaving, and the end. The input of the
// drive package's contracts (its pages are built to test against), and
// handy for any component that reads control. It is no longer a fixture:
// rec-driven, a real driven session (#467), replaced it.
func DrivenRecording() []Record {
	ana, bo, cy := Author{ID: "v1", Name: "Ana", Color: "#e5484d"}, Author{ID: "v2", Name: "Bo", Color: "#3e63dd"}, Author{ID: "v3", Name: "Cy"}
	var c Control
	op := func(o string, who Author) *Ctl {
		next, err := NextControl(c, o, who)
		if err != nil {
			panic(err) // a hand-written sequence: a bug here is the test's
		}
		c = next
		return &Ctl{Op: o, Who: who, Control: next}
	}
	b := NewBuilder("driven")
	q := b.El("input", []string{"id", "q", "type", "search"})
	save := b.El("button", []string{"id", "save"}, b.Txt("Save"))
	link := b.El("a", []string{"href", "/2"}, b.Txt("Next"))
	doc := b.Page(SegLive, "http://app.local/", q, save, link)
	b.At(100).Add(&Layout{Boxes: map[NodeID]Box{q.ID: {16, 16, 300, 32}, save.ID: {332, 16, 80, 32}, link.ID: {16, 64, 40, 20}}})
	b.At(1000).Add(op(CtlRequest, ana))
	b.At(1500).Add(op(CtlGrant, ana))
	b.At(2000).Add(&Click{Source: Source{By: ana.ID}, ID: q.ID, X: 100, Y: 30}).Add(&Focus{Source: Source{By: ana.ID}, ID: q.ID})
	b.At(2400).Add(&Input{Source: Source{By: ana.ID}, ID: q.ID, V: Str("invoice")})
	b.At(2600).Add(&Note{ID: "drv-n1", Author: ana, Text: "Search should keep my query.", At: NoteAt{Node: q.ID, Sel: "#q", Box: &Box{16, 16, 300, 32}}})
	// The operator acts too: unattributed.
	b.At(3000).Add(&Click{ID: save.ID, X: 370, Y: 30})
	b.At(3500).Add(&Stop{Why: StopPause})
	b.At(4000).Add(op(CtlRequest, bo)) // asked while paused
	b.At(5000).Add(&Seg{Seg: b.NextSeg(), Why: SegResume, Tab: Tab{ID: "tab1", URL: "http://app.local/"}, VP: Viewport{W: 1280, H: 800, DPR: 1}})
	b.Snapshot("http://app.local/", doc.C[1])
	b.At(5500).Add(&Click{Source: Source{By: ana.ID}, ID: link.ID, X: 30, Y: 70})
	// The driver's click navigates: a new document, still driven.
	b.At(5600)
	p := b.El("p", nil, b.Txt("Page two"))
	b.Page(SegNavigate, "http://app.local/2", p)
	b.At(6000).Add(&Scroll{Source: Source{By: ana.ID}, ID: b.Recs[len(b.Recs)-1].(*Doc).Doc, Y: 200})
	b.At(6500).Add(op(CtlRelease, ana))
	b.At(7000).Add(op(CtlGrant, bo))
	b.At(7200).Add(&Click{Source: Source{By: bo.ID}, ID: p.ID, X: 20, Y: 20})
	b.At(7500).Add(op(CtlRevoke, bo))
	b.At(8000).Add(op(CtlRequest, cy))
	b.At(8200).Add(op(CtlDeny, cy))
	b.At(8500).Add(op(CtlRequest, cy)).Add(op(CtlGrant, cy))
	b.At(9000).Add(op(CtlLeave, cy))
	b.At(9500).Add(&Stop{Why: StopEnd})
	return b.Recs
}

// RunLiveContract checks a Live implementation. newLive returns a fresh
// fan-out and the Sink that feeds it (often the same value), whose
// subscribers may fall behind by at least 64 records before being dropped.
func RunLiveContract(t TB, newLive func() (Sink, Live)) {
	t.Helper()
	recs := ContractRecording(40)

	// A late joiner first receives the catch-up of everything so far.
	for _, cut := range []int{1, 5, len(recs) / 2, len(recs)} {
		sink, live := newLive()
		for _, r := range recs[:cut] {
			sink.Put(r)
		}
		ch, cancel := live.Subscribe()
		want := CatchUp(recs[:cut])
		for i, w := range want {
			select {
			case got := <-ch:
				if got != w {
					t.Fatalf("cut %d: catch-up record %d is %s@%d, want %s@%d", cut, i, got.Kind(), got.Time(), w.Kind(), w.Time())
				}
			case <-time.After(time.Second):
				t.Fatalf("cut %d: catch-up stopped after %d of %d records", cut, i, len(want))
			}
		}
		cancel()
		cancel() // idempotent
		sink.Close()
	}

	// Every sequential join point: subscribing after k records gives exactly
	// CatchUp(first k) followed by the rest. Deterministic; the concurrent
	// run below is the stress version of the same property.
	for k := 0; k <= len(recs); k++ {
		sink, live := newLive()
		for _, r := range recs[:k] {
			sink.Put(r)
		}
		ch, _ := live.Subscribe()
		for _, r := range recs[k:] {
			sink.Put(r)
		}
		sink.Close()
		var got []Record
		for r := range ch {
			got = append(got, r)
		}
		want := append(CatchUp(recs[:k]), recs[k:]...)
		if !sameRecords(got, want) {
			t.Fatalf("joining after %d records: got %d records, want CatchUp + rest (%d)", k, len(got), len(want))
		}
	}

	// Frames never reach viewers.
	{
		sink, live := newLive()
		ch, cancel := live.Subscribe()
		f := &Frame{File: "frames/0.png", W: 1, H: 1}
		if err := sink.Put(f); err == nil {
			t.Errorf("a live fan-out must refuse a frame")
		}
		sink.Close()
		for r := range ch {
			if r == Record(f) {
				t.Errorf("a frame reached a viewer")
			}
		}
		cancel()
	}

	// Under a concurrent writer, each subscriber sees CatchUp(prefix)
	// followed by the rest: no record lost between the two, none twice.
	sink, live := newLive()
	const subscribers = 8
	streams := make([][]Record, subscribers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range subscribers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			time.Sleep(time.Duration(i) * 50 * time.Microsecond)
			ch, _ := live.Subscribe()
			for r := range ch {
				streams[i] = append(streams[i], r)
			}
		}()
	}
	close(start)
	for _, r := range recs {
		sink.Put(r)
		time.Sleep(20 * time.Microsecond)
	}
	sink.Close()
	wg.Wait()
	for i, got := range streams {
		if !isCatchUpThenRest(got, recs) {
			t.Errorf("subscriber %d: %d records are not a catch-up followed by the rest", i, len(got))
		}
	}

	// A subscriber that never reads is dropped; it doesn't block the writer.
	sink, live = newLive()
	ch, cancel := live.Subscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 20; i++ {
			for _, r := range recs {
				sink.Put(r)
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("a subscriber that doesn't read blocked the writer")
	}
	n := 0
	for range ch {
		n++
	}
	if n >= 20*len(recs) {
		t.Errorf("a subscriber that didn't read got all %d records; it should have been dropped", n)
	}
	// A drop stays a drop, even once the fan-out has ended since: the
	// viewer must reconnect, having missed records.
	if live.Ended(ch) {
		t.Errorf("a dropped subscription reads as the fan-out's end")
	}
	sink.Close()
	if live.Ended(ch) {
		t.Errorf("a subscription dropped before Close reads as the fan-out's end")
	}
	cancel()

	// Close ends every subscription, and says it was the end.
	sink, live = newLive()
	ch, _ = live.Subscribe()
	sink.Close()
	select {
	case _, open := <-drain(ch):
		if open {
			t.Errorf("subscription still open after Close")
		}
	case <-time.After(time.Second):
		t.Errorf("Close didn't end the subscription")
	}
	if !live.Ended(ch) {
		t.Errorf("a subscription closed by Close doesn't read as the end")
	}
}

// drain discards buffered records and reports when ch closes.
func drain(ch <-chan Record) <-chan Record {
	out := make(chan Record)
	go func() {
		for range ch {
		}
		close(out)
	}()
	return out
}

func isCatchUpThenRest(got, all []Record) bool {
	for k := 0; k <= len(all); k++ {
		if sameRecords(got, append(CatchUp(all[:k]), all[k:]...)) {
			return true
		}
	}
	return false
}

func sameRecords(a, b []Record) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CheckCatchUp reports whether joining after recs, at t, gives a viewer
// what the whole recording gives: the same canonical page, assets,
// document start (DocSeg) and all retained notes. Every
// catch-up check uses it, so none compares less than consumers read.
func CheckCatchUp(recs []Record, t int64) error {
	all, err := Reduce(recs, t)
	if err != nil {
		return err
	}
	var upto []Record
	for _, r := range recs {
		if r.Time() <= t {
			upto = append(upto, r)
		}
	}
	cu, err := ReduceCatchUp(CatchUp(upto), t)
	if err != nil {
		return fmt.Errorf("the catch-up doesn't reduce: %w", err)
	}
	switch {
	case !bytes.Equal(all.Canonical(), cu.Canonical()):
		return fmt.Errorf("the page differs")
	case !sameSketchProjection(all.Sketches.Snapshot(), cu.Sketches.Snapshot()):
		return fmt.Errorf("sketch projection differs")
	case !sameAssets(all.Assets, cu.Assets):
		return fmt.Errorf("the asset table differs")
	case all.DocSeg != cu.DocSeg:
		return fmt.Errorf("the document began in segment %d, the catch-up says %d", all.DocSeg, cu.DocSeg)
	case retainedNotes(all) != retainedNotes(cu):
		return fmt.Errorf("retained notes %q, the catch-up has %q", retainedNotes(all), retainedNotes(cu))
	case all.Hover != cu.Hover || !samePtr(all.Ptr, cu.Ptr):
		return fmt.Errorf("hover %d ptr %v, the catch-up has %d %v", all.Hover, all.Ptr, cu.Hover, cu.Ptr)
	case !all.Control.Equal(cu.Control):
		return fmt.Errorf("driver %q pending %v, the catch-up has %q %v",
			all.Control.Driving(), ids(all.Control.Pending), cu.Control.Driving(), ids(cu.Control.Pending))
	}
	return nil
}

func samePtr(a, b *[2]float64) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func sameAssets(a, b map[string]*Asset) bool {
	if len(a) != len(b) {
		return false
	}
	for url, x := range a {
		if y := b[url]; y == nil || y.Hash != x.Hash {
			return false
		}
	}
	return true
}

// retainedNotes covers every note the open-note list can expose.
func retainedNotes(s *State) string {
	var out []string
	for _, n := range s.Notes {
		wire, _ := json.Marshal(struct {
			Note       *Note
			Seg        int
			State      string
			Revision   uint64
			ResolvedBy *Author
		}{n.Note, n.Seg, n.State, n.Revision, n.ResolvedBy})
		out = append(out, string(wire))
	}
	return strings.Join(out, ",")
}

func sameSketchProjection(a, b SketchProjection) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
