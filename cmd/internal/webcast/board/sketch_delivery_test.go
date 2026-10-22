package board

import (
	"encoding/json"
	"github.com/xianxu/pair/internal/reporoot"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func sketchDeliveryFixture(t *testing.T) *Board {
	t.Helper()
	b := New(notes.Structural{}, nil)
	build := recording.NewBuilder("delivery")
	build.Page(recording.SegLive, "http://x/")
	for _, r := range build.Recs {
		b.Put(r)
	}
	return b
}
func deliveryOp(seq uint64) recording.SketchOperation {
	return recording.SketchOperation{Operation: "op-" + stringNumber(seq), Sequence: seq, Digest: strings.Repeat("a", 64)}
}
func stringNumber(v uint64) string { p, _ := json.Marshal(v); return string(p) }
func deliveryPut(t *testing.T, b *Board, r recording.Record) {
	t.Helper()
	r.SetTime(b.st.T + 1)
	b.Put(r)
	if b.broken != nil {
		t.Fatal(b.broken)
	}
}
func deliveryBegin(t *testing.T, b *Board) {
	deliveryPut(t, b, &recording.SketchBegin{SketchOperation: deliveryOp(0), ID: "s", Author: recording.Author{ID: recording.OperatorID, Color: "#123456"}, Owner: recording.OperatorID, Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 100, H: 100, DPR: 1}})
}
func deliveryStroke(t *testing.T, b *Board) {
	deliveryPut(t, b, &recording.SketchStrokeBegin{SketchOperation: deliveryOp(1), Sketch: "s", ID: "p", At: recording.NoteAt{Node: 3, Box: &recording.Box{0, 0, 10, 10}}, Width: 3})
}
func TestSketchDeliveryGapDuplicatesAndOwnedFrames(t *testing.T) {
	b := sketchDeliveryFixture(t)
	initial, err := b.OperatorSketchSince(SketchDeliveryCursor{})
	if err != nil || initial.Bootstrap == nil {
		t.Fatalf("initial bootstrap %+v %v", initial, err)
	}
	cursor := SketchDeliveryCursor{Epoch: initial.Bootstrap.Epoch, Seq: initial.Bootstrap.Seq}
	deliveryBegin(t, b)
	deliveryStroke(t, b)
	points := &recording.SketchPoints{SketchOperation: deliveryOp(2), Sketch: "s", ID: "p", Pts: []recording.SketchPoint{{1, 2, .5}}}
	deliveryPut(t, b, points)
	frame, err := b.OperatorSketchSince(cursor)
	if err != nil || frame.Batch == nil || len(frame.Batch.Patches) != 3 {
		t.Fatalf("batch %+v %v", frame, err)
	}
	raw, _ := json.Marshal(frame.Batch)
	if len(raw) > SketchDeliveryBatchBytes {
		t.Fatal("batch bound")
	}
	stats := b.OperatorSketchDeliveryStats()
	b.Put(points)
	again, _ := b.OperatorSketchSince(cursor)
	rawAgain, _ := json.Marshal(again.Batch)
	if string(rawAgain) != string(raw) || b.OperatorSketchDeliveryStats() != stats {
		t.Fatal("duplicate changed journal")
	}
	frame.Batch.Patches[0][0] = '!'
	fresh, _ := b.OperatorSketchSince(cursor)
	if fresh.Batch.Patches[0][0] == '!' {
		t.Fatal("frame aliases retained bytes")
	}
	gap, _ := b.OperatorSketchSince(SketchDeliveryCursor{Epoch: "wrong", Seq: 0})
	if gap.Bootstrap == nil || gap.Bootstrap.Seq != fresh.Batch.To {
		t.Fatal("gap highwater not atomic")
	}
	high := SketchDeliveryCursor{Epoch: fresh.Batch.Epoch, Seq: fresh.Batch.To}
	none, _ := b.OperatorSketchSince(high)
	if none.Batch != nil || none.Bootstrap != nil {
		t.Fatal("equal cursor emitted duplicate data")
	}
}
func TestSketchDeliveryOverflowAndBatchBudgets(t *testing.T) {
	b := sketchDeliveryFixture(t)
	initial, _ := b.OperatorSketchSince(SketchDeliveryCursor{})
	old := SketchDeliveryCursor{Epoch: initial.Bootstrap.Epoch, Seq: 0}
	deliveryBegin(t, b)
	deliveryStroke(t, b)
	deliveryPut(t, b, &recording.SketchPoints{SketchOperation: deliveryOp(2), Sketch: "s", ID: "p", Pts: []recording.SketchPoint{{1, 2, 1}}})
	deliveryPut(t, b, &recording.SketchStrokeEnd{SketchOperation: deliveryOp(3), Sketch: "s", ID: "p"})
	for i := uint64(4); i < 304; i++ {
		if i%2 == 0 {
			deliveryPut(t, b, &recording.SketchUndo{SketchOperation: deliveryOp(i), Sketch: "s", ID: "p"})
		} else {
			deliveryPut(t, b, &recording.SketchRedo{SketchOperation: deliveryOp(i), Sketch: "s", ID: "p"})
		}
	}
	if len(b.sketchDelivery.entries) > SketchDeliveryJournalEntries || b.sketchDelivery.bytes > SketchDeliveryJournalBytes {
		t.Fatal("journal exceeded bound")
	}
	gap, err := b.OperatorSketchSince(old)
	if err != nil || gap.Bootstrap == nil {
		t.Fatal("evicted cursor did not bootstrap")
	}
	first := b.sketchDelivery.entries[0].seq
	frame, err := b.OperatorSketchSince(SketchDeliveryCursor{Epoch: gap.Bootstrap.Epoch, Seq: first - 1})
	if err != nil || frame.Batch == nil || len(frame.Batch.Patches) > 32 {
		t.Fatalf("bounded batch %+v %v", frame, err)
	}
	raw, _ := json.Marshal(frame.Batch)
	if len(raw) > SketchDeliveryBatchBytes {
		t.Fatal("batch bytes")
	}
}
func TestSketchDeliveryN2NAndContextChanges(t *testing.T) {
	for _, n := range []int{64, 128} {
		b := sketchDeliveryFixture(t)
		b.OperatorSketchSince(SketchDeliveryCursor{})
		deliveryBegin(t, b)
		deliveryStroke(t, b)
		base := b.OperatorSketchDeliveryStats()
		seq := uint64(2)
		for i := 0; i < n; i += 64 {
			deliveryPut(t, b, &recording.SketchPoints{SketchOperation: deliveryOp(seq), Sketch: "s", ID: "p", Chunk: uint64(i / 64), Pts: make([]recording.SketchPoint, 64)})
			seq++
		}
		stats := b.OperatorSketchDeliveryStats()
		if stats.Points-base.Points != uint64(n) || stats.Bootstraps != base.Bootstraps {
			t.Fatalf("chunk copied history %+v -> %+v", base, stats)
		}
		deliveryPut(t, b, &recording.Stop{Why: recording.StopPause})
		last := b.sketchDelivery.entries[len(b.sketchDelivery.entries)-1]
		if !strings.Contains(string(last.raw), `"truncated":true`) {
			t.Fatal("contextual partial seal missing")
		}
	}
}
func TestSketchDeliveryWakeAndClose(t *testing.T) {
	b := sketchDeliveryFixture(t)
	updates, cancel := b.SubscribeSketchDelivery()
	defer cancel()
	<-updates
	deliveryBegin(t, b)
	deliveryStroke(t, b)
	if len(updates) != 1 {
		t.Fatal("wake must coalesce")
	}
	b.Close()
	if _, ok := <-updates; !ok {
		return
	}
	if _, ok := <-updates; ok {
		t.Fatal("Close left wake open")
	}
}

func TestSketchDeliveryProjectionMatchesActualBrowser(t *testing.T) {
	b := sketchDeliveryFixture(t)
	initial, _ := b.OperatorSketchSince(SketchDeliveryCursor{})
	cursor := SketchDeliveryCursor{Epoch: initial.Bootstrap.Epoch, Seq: initial.Bootstrap.Seq}
	type step struct {
		Batch *SketchDeliveryBatch `json:"batch"`
		Want  SketchDisplay        `json:"want"`
	}
	transcript := struct {
		Initial *SketchDeliveryBootstrap `json:"initial"`
		Steps   []step                   `json:"steps"`
	}{Initial: initial.Bootstrap}
	follow := func(r recording.Record) {
		t.Helper()
		deliveryPut(t, b, r)
		frame, err := b.OperatorSketchSince(cursor)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Batch == nil {
			return
		}
		cursor.Seq = frame.Batch.To
		view, err := b.OperatorSketchSince(SketchDeliveryCursor{})
		if err != nil {
			t.Fatal(err)
		}
		transcript.Steps = append(transcript.Steps, step{Batch: frame.Batch, Want: view.Bootstrap.Display})
	}
	begin := &recording.SketchBegin{SketchOperation: deliveryOp(0), ID: "s", Author: recording.Author{ID: recording.OperatorID, Color: "#123456"}, Owner: recording.OperatorID, Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 100, H: 100, DPR: 1}}
	follow(begin)
	follow(&recording.SketchStrokeBegin{SketchOperation: deliveryOp(1), Sketch: "s", ID: "p", At: recording.NoteAt{Node: 3, Box: &recording.Box{0, 0, 10, 10}}, Width: 3})
	follow(&recording.SketchPoints{SketchOperation: deliveryOp(2), Sketch: "s", ID: "p", Pts: []recording.SketchPoint{{1, 2, .5}}})
	follow(&recording.SketchStrokeEnd{SketchOperation: deliveryOp(3), Sketch: "s", ID: "p", Truncated: true})
	follow(&recording.SketchUndo{SketchOperation: deliveryOp(4), Sketch: "s", ID: "p"})
	follow(&recording.SketchRedo{SketchOperation: deliveryOp(5), Sketch: "s", ID: "p"})
	follow(&recording.SketchStrokeBegin{SketchOperation: deliveryOp(6), Sketch: "s", ID: "q", At: recording.NoteAt{Node: 3, Box: &recording.Box{0, 0, 10, 10}}, Width: 3})
	follow(&recording.SketchPoints{SketchOperation: deliveryOp(7), Sketch: "s", ID: "q", Pts: []recording.SketchPoint{{3, 4, 1}}})
	follow(&recording.Stop{Why: recording.StopPause})
	follow(&recording.Seg{Seg: 2, Why: recording.SegResume, Tab: recording.Tab{ID: "tab1"}, VP: recording.Viewport{W: 100, H: 100, DPR: 1}})
	follow(&recording.SketchLabel{SketchOperation: deliveryOp(8), Sketch: "s", Label: "Saved partial"})
	follow(&recording.AnnotationClear{Who: recording.Author{ID: recording.OperatorID}})
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("Node is required for the shared delivery contract")
	}
	script := `import fs from 'node:fs';import assert from 'node:assert/strict';import {pathToFileURL} from 'node:url';const {createSketchDisplay}=await import(pathToFileURL(process.argv[1]));const input=JSON.parse(fs.readFileSync(0,'utf8'));const display=createSketchDisplay();display.install(input.initial.display);for(const [i,step] of input.steps.entries()){display.applyBatch(step.batch.patches);assert.deepStrictEqual(display.snapshot(),step.want,'prefix '+i);}`
	command := exec.Command(node, "--input-type=module", "-e", script, filepath.Join(root, "cmd/internal/webcast/web/sketch-render.js"))
	raw, err := json.Marshal(transcript)
	if err != nil {
		t.Fatal(err)
	}
	command.Stdin = strings.NewReader(string(raw))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("real browser projector differs: %v\n%s", err, output)
	}
}

func TestSketchDeliveryOversizePatchAndBootstrapEncodingBound(t *testing.T) {
	b := sketchDeliveryFixture(t)
	initial, _ := b.OperatorSketchSince(SketchDeliveryCursor{})
	old := SketchDeliveryCursor{Epoch: initial.Bootstrap.Epoch, Seq: 0}
	deliveryBegin(t, b)
	b.mu.Lock()
	b.sketchDelivery.append(json.RawMessage(`{"label":"`+strings.Repeat("x", SketchDeliveryBatchBytes)+`"}`), 0)
	b.mu.Unlock()
	frame, err := b.OperatorSketchSince(old)
	if err != nil || frame.Bootstrap == nil {
		t.Fatal("oversize patch did not force authoritative bootstrap")
	}
	// Exercise maximum legal sketch/stroke identity counts; each variable ID
	// appears only in its header, while receipts never enter the bootstrap.
	b = sketchDeliveryFixture(t)
	for sketch := 0; sketch < recording.SketchMaxCount; sketch++ {
		sid := "sketch-" + stringNumber(uint64(sketch))
		deliveryPut(t, b, &recording.SketchBegin{SketchOperation: deliveryOp(0), ID: sid, Author: recording.Author{ID: recording.OperatorID, Color: "#123456"}, Owner: recording.OperatorID, Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 100, H: 100, DPR: 1}})
		sequence := uint64(1)
		for stroke := 0; stroke < recording.SketchMaxStrokes; stroke++ {
			id := "path-" + stringNumber(uint64(stroke))
			deliveryPut(t, b, &recording.SketchStrokeBegin{SketchOperation: deliveryOp(sequence), Sketch: sid, ID: id, At: recording.NoteAt{Node: 3, Box: &recording.Box{0, 0, 10, 10}}, Width: 3})
			sequence++
			deliveryPut(t, b, &recording.SketchStrokeEnd{SketchOperation: deliveryOp(sequence), Sketch: sid, ID: id, Truncated: true})
			sequence++
		}
		deliveryPut(t, b, &recording.SketchLabel{SketchOperation: deliveryOp(sequence), Sketch: sid, Label: strings.Repeat("<", 1024)})
	}
	frame, err = b.OperatorSketchSince(SketchDeliveryCursor{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(frame.Bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	projection := b.st.Sketches.Snapshot()
	ceiling := projection.HistoryBytes + 128*recording.SketchMaxCount*recording.SketchMaxStrokes + 256*recording.SketchMaxCount + 1024
	if len(encoded) > ceiling || len(encoded) > SketchDeliveryBootstrapBytes {
		t.Fatalf("actual bootstrap %d exceeds admitted bound %d", len(encoded), ceiling)
	}
	if len(frame.Bootstrap.Display.Sketches) != recording.SketchMaxCount || len(frame.Bootstrap.Display.Sketches[0].Strokes) != recording.SketchMaxStrokes {
		t.Fatal("maximum bootstrap dropped accepted items")
	}
}

func TestSketchDeliveryBootstrapHighWaterIsAtomicWithConcurrentAppend(t *testing.T) {
	b := sketchDeliveryFixture(t)
	deliveryBegin(t, b)
	deliveryStroke(t, b)
	done := make(chan struct{})
	errs := make(chan error, 1)
	go func() {
		defer close(done)
		for i := uint64(0); i < 64; i++ {
			b.mu.Lock()
			stamp := b.st.T + 1
			b.mu.Unlock()
			r := &recording.SketchPoints{Header: recording.Header{T: stamp}, SketchOperation: deliveryOp(i + 2), Sketch: "s", ID: "p", Chunk: i, Pts: []recording.SketchPoint{{float64(i), 2, .5}}}
			b.Put(r)
			b.mu.Lock()
			err := b.broken
			b.mu.Unlock()
			if err != nil {
				errs <- err
				return
			}
		}
	}()
	for i := 0; i < 100; i++ {
		frame, err := b.OperatorSketchSince(SketchDeliveryCursor{})
		if err != nil {
			t.Fatal(err)
		}
		count := len(frame.Bootstrap.Display.Sketches[0].Strokes[0].Pts)
		if frame.Bootstrap.Seq != uint64(count+2) {
			t.Fatalf("snapshot count %d and high-water %d crossed a writer boundary", count, frame.Bootstrap.Seq)
		}
	}
	<-done
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
}

func TestOperatorSketchDoesNotRebuildUnrelatedNoteViewOnPoints(t *testing.T) {
	r := newRig(t).page(recording.SegLive)
	before := r.bd.operatorRevision
	begin := writerSketchBegin()
	if err := r.w.Emit(begin, 1); err != nil {
		t.Fatal(err)
	}
	if r.bd.operatorRevision != before {
		t.Fatal("sketch rebuilt unrelated note projection")
	}
	if r.bd.OperatorView().SketchCursor != r.bd.OperatorSketchCursor() {
		t.Fatal("view lacks atomic drawing highwater")
	}
}
