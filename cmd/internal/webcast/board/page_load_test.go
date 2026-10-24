package board

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func loadPage(r *rig, why string, load *recording.PageLoad) *rig {
	r.t.Helper()
	r.p1 = r.b.El("p", []string{"id", "one"}, r.b.Txt("First"))
	r.p2 = r.b.El("p", nil, r.b.Txt("Second"))
	r.b.Page(why, "http://same/", r.p1, r.p2)
	for _, record := range r.b.Recs[r.fed:] {
		if seg, ok := record.(*recording.Seg); ok {
			seg.Tab.ID = load.Target
			seg.PageLoad = load
		}
	}
	return r.flush()
}

func TestPageLoadPendingBlocksFreshNotesButPreservesAcceptedResolution(t *testing.T) {
	r := loadPage(newRig(t), recording.SegLive, &recording.PageLoad{Target: "A", Loader: "L", Root: 1})
	id, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := r.bd.Selection()
	if err != nil {
		t.Fatal(err)
	}
	r.bd.BeginPageLoad(1)
	if _, err := r.note(r.p2.ID, 1); !errors.Is(err, ErrStale) {
		t.Fatalf("fresh note on pending load: %v, want stale", err)
	}
	if err := r.bd.ResolveSelection(context.Background(), selected); err != nil {
		t.Fatalf("pending scope blocked admitted exact-version recovery: %v", err)
	}
	if len(r.bd.st.Notes) != 1 || r.bd.st.Notes[0].ID != id || r.bd.st.Notes[0].State != recording.NoteResolved {
		t.Fatalf("wrong historical outcome: %+v", r.bd.st.Notes)
	}
}

func TestPageLoadSelectionPartitionsHistoryAndRestoresSameLoad(t *testing.T) {
	a := &recording.PageLoad{Target: "A", Loader: "loader-A", Root: 1}
	b := &recording.PageLoad{Target: "B", Loader: "loader-B", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, a)
	first, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	original, err := r.bd.Selection()
	if err != nil || len(original.Notes) != 1 {
		t.Fatalf("initial selection: %+v, %v", original, err)
	}
	loadPage(r, recording.SegTab, b)
	second, err := r.note(r.p1.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := r.bd.Selection()
	if err != nil || len(selected.Notes) != 1 || selected.Notes[0].ID != second {
		t.Fatalf("new target selected earlier-load note %s: %+v, %v", first, selected, err)
	}
	loadPage(r, recording.SegTab, a)
	selected, err = r.bd.Selection()
	if err != nil || len(selected.Notes) != 1 || selected.Notes[0].ID != first {
		t.Fatalf("same-load tab return: %+v, %v", selected, err)
	}
	if len(original.Notes) != 1 || original.Notes[0].ID != first {
		t.Fatalf("navigation changed admitted selection: %+v", original)
	}
	state, err := recording.Reduce(r.mem.Records(), 1<<62)
	if err != nil || len(state.Notes) != 2 {
		t.Fatalf("history lost across tab return: %+v, %v", state, err)
	}
}

func TestPageLoadPendingFencesSelectionAndStaleConfirmation(t *testing.T) {
	a := &recording.PageLoad{Target: "A", Loader: "loader-A", Root: 1}
	b := &recording.PageLoad{Target: "B", Loader: "loader-B", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, a)
	first, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	scope, ok := any(r.bd).(interface {
		BeginPageLoad(uint64) bool
		ConfirmPageLoad(uint64, *recording.PageLoad) bool
	})
	if !ok {
		t.Fatal("Board lacks generation-fenced pending/confirmation authority")
	}
	if !scope.BeginPageLoad(1) || scope.BeginPageLoad(1) {
		t.Fatal("pending must advance once")
	}
	selected, err := r.bd.Selection()
	if err != nil || len(selected.Notes) != 0 {
		t.Fatalf("pending selected old page: %+v, %v", selected, err)
	}
	if scope.ConfirmPageLoad(0, a) || scope.ConfirmPageLoad(1, b) {
		t.Fatal("stale or unrecorded identity confirmed")
	}
	if !scope.ConfirmPageLoad(1, a) {
		t.Fatal("matching accepted identity did not confirm")
	}
	selected, err = r.bd.Selection()
	if err != nil || len(selected.Notes) != 1 || selected.Notes[0].ID != first {
		t.Fatalf("confirmed selection: %+v, %v", selected, err)
	}
	if !scope.BeginPageLoad(2) || scope.ConfirmPageLoad(1, a) {
		t.Fatal("successor pending was not preserved")
	}
	loadPage(r, recording.SegTab, b)
	selected, err = r.bd.Selection()
	if err != nil || len(selected.Notes) != 0 {
		t.Fatalf("accepted history bypassed pending: %+v, %v", selected, err)
	}
	if !scope.ConfirmPageLoad(2, b) {
		t.Fatal("new accepted target did not confirm")
	}
}

func TestPageLoadFailedObservationDoesNotRestoreAcceptedMetadata(t *testing.T) {
	load := &recording.PageLoad{Target: "A", Loader: "L", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, load)
	scope, ok := any(r.bd).(interface {
		PageLoadReady() bool
		UnavailablePageLoad(uint64) bool
	})
	if !ok {
		t.Fatal("Board lacks failed-observation/readiness boundary")
	}
	if !scope.PageLoadReady() || !r.bd.BeginPageLoad(1) || scope.PageLoadReady() {
		t.Fatal("readiness did not follow pending scope")
	}
	if !scope.UnavailablePageLoad(1) || scope.PageLoadReady() || r.bd.ConfirmPageLoad(1, load) {
		t.Fatal("failed observation restored old accepted metadata")
	}
	if !r.bd.BeginPageLoad(2) || scope.UnavailablePageLoad(1) || !r.bd.ConfirmPageLoad(2, load) || !scope.PageLoadReady() {
		t.Fatal("old failure changed newer observation")
	}
}

func TestPageLoadRejectsFreshActionsOnEarlierNote(t *testing.T) {
	a := &recording.PageLoad{Target: "A", Loader: "A1", Root: 1}
	b := &recording.PageLoad{Target: "B", Loader: "B1", Root: 1}
	for _, action := range []string{"edit", "state"} {
		t.Run(action, func(t *testing.T) {
			r := loadPage(newRig(t), recording.SegLive, a)
			id, err := r.bd.Note(NoteIn{Viewer: ana, Owner: "owner", Seg: r.seg(), Node: r.p1.ID, Box: recording.Box{1, 2, 3, 4}, Region: recording.Box{1, 2, 3, 4}, Text: "original"})
			if err != nil {
				t.Fatal(err)
			}
			loadPage(r, recording.SegTab, b)
			act := func() error {
				if action == "edit" {
					return r.bd.Edit(EditIn{Seg: r.seg(), ID: id, Text: "changed", Owner: "owner"})
				}
				return r.bd.SetState(StateIn{Viewer: ana, ID: id, State: recording.NoteResolved})
			}
			if err := act(); !errors.Is(err, ErrStale) {
				t.Fatalf("%s on earlier load: %v, want stale", action, err)
			}
			loadPage(r, recording.SegTab, a)
			if err := act(); err != nil {
				t.Fatalf("%s after same-load return: %v", action, err)
			}
		})
	}
}

func TestPageLoadSketchFreshWorkFencedAfterReceiptReconciliation(t *testing.T) {
	r, service, authority := sketchServiceRig(t)
	begin := SketchCommand{Kind: "sketch-begin", Operation: "begin", Epoch: 1, Seg: 1, Color: 0}
	accepted := service.Execute(context.Background(), authority, begin)
	if accepted.Outcome != SketchAccepted {
		t.Fatalf("begin: %+v", accepted)
	}
	r.bd.BeginPageLoad(1)
	if again := service.Execute(context.Background(), authority, begin); again.Outcome != SketchAccepted || again.Sketch != accepted.Sketch {
		t.Fatalf("pending scope lost exact receipt: %+v", again)
	}
	label := SketchCommand{Kind: "sketch-label", Operation: "label", Sequence: 1, Epoch: 1, Sketch: accepted.Sketch, Label: "saved"}
	if got := service.Execute(context.Background(), authority, label); got.Outcome != SketchRefused {
		t.Fatalf("fresh label admitted while scope pending: %+v", got)
	}
	if !r.bd.ConfirmPageLoad(1, r.bd.st.CurrentPageLoad()) {
		t.Fatal("could not restore same load")
	}
	label.Operation = "label-current"
	if got := service.Execute(context.Background(), authority, label); got.Outcome != SketchAccepted {
		t.Fatalf("same-load fresh label: %+v", got)
	}
	r.bd.BeginPageLoad(2)
	if got := service.Execute(context.Background(), authority, label); got.Outcome != SketchAccepted {
		t.Fatalf("pending scope lost accepted label receipt: %+v", got)
	}
}

func TestPageLoadConcurrentConfirmationCannotOvertakePending(t *testing.T) {
	load := &recording.PageLoad{Target: "A", Loader: "L", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, load)
	if _, err := r.note(r.p1.ID, 0); err != nil {
		t.Fatal(err)
	}
	for generation := uint64(1); generation < 65; generation += 2 {
		if !r.bd.BeginPageLoad(generation) {
			t.Fatal("new observation refused")
		}
		start := make(chan struct{})
		var done sync.WaitGroup
		done.Add(2)
		go func() {
			defer done.Done()
			<-start
			r.bd.ConfirmPageLoad(generation, load)
		}()
		go func() {
			defer done.Done()
			<-start
			r.bd.BeginPageLoad(generation + 1)
		}()
		close(start)
		done.Wait()
		selected, err := r.bd.Selection()
		if err != nil || len(selected.Notes) != 0 || r.bd.PageLoadReady() {
			t.Fatalf("old confirmation restored newer pending scope: %+v, %v", selected, err)
		}
	}
}

func TestPageLoadPublishesOnlyCurrentFeedbackAndPendingEmpty(t *testing.T) {
	a := &recording.PageLoad{Target: "A", Loader: "L1", Root: 1}
	b := &recording.PageLoad{Target: "B", Loader: "L2", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, a)
	first, err := r.note(r.p1.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	views, cancel := r.bd.SubscribeOperator()
	defer cancel()
	<-views
	live, stop := r.bd.Subscribe()
	defer stop()
	<-live
	r.bd.BeginPageLoad(1)
	select {
	case view := <-views:
		if len(view.Notes) != 0 {
			t.Fatalf("pending view retained notes: %+v", view.Notes)
		}
	default:
		t.Fatal("pending scope did not publish operator view")
	}
	if notes := r.bd.Live(); len(notes) != 0 {
		t.Fatalf("pending live list retained notes: %+v", notes)
	}
	loadPage(r, recording.SegTab, b)
	if view := r.bd.OperatorView(); len(view.Notes) != 0 {
		t.Fatal("historical commit bypassed pending")
	}
	if !r.bd.ConfirmPageLoad(1, b) {
		t.Fatal("confirm B")
	}
	second, err := r.note(r.p1.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if view := r.bd.OperatorView(); len(view.Notes) != 1 || view.Notes[0].ID != second {
		t.Fatalf("B view: %+v", view.Notes)
	}
	r.bd.BeginPageLoad(2)
	loadPage(r, recording.SegTab, a)
	r.bd.ConfirmPageLoad(2, a)
	if view := r.bd.OperatorView(); len(view.Notes) != 1 || view.Notes[0].ID != first {
		t.Fatalf("same-load return: %+v", view.Notes)
	}
	if got := r.bd.Live(); len(got) != 1 || got[0].ID != first {
		t.Fatalf("same-load live list: %+v", got)
	}
	r.bd.BeginPageLoad(3)
	r.bd.UnavailablePageLoad(3)
	if len(r.bd.OperatorView().Notes) != 0 || len(r.bd.Live()) != 0 {
		t.Fatal("failed load restored history")
	}
}

func TestPageLoadSketchDeliveryResetsAndRestoresMatchingHistory(t *testing.T) {
	b := sketchDeliveryFixture(t)
	deliveryBegin(t, b)
	deliveryStroke(t, b)
	before := b.OperatorSketchCursor()
	wake, stop := b.SubscribeSketchDelivery()
	defer stop()
	<-wake
	if !b.BeginPageLoad(1) {
		t.Fatal("pending")
	}
	select {
	case <-wake:
	default:
		t.Fatal("pending scope did not wake sketch delivery")
	}
	pending, err := b.OperatorSketchSince(before)
	if err != nil || pending.Bootstrap == nil || len(pending.Bootstrap.Display.Sketches) != 0 {
		t.Fatalf("pending delivery: %+v %v", pending, err)
	}
	if !b.ConfirmPageLoad(1, b.st.CurrentPageLoad()) {
		t.Fatal("confirm")
	}
	frame, err := b.OperatorSketchSince(SketchDeliveryCursor{Epoch: pending.Bootstrap.Epoch, Seq: pending.Bootstrap.Seq})
	if err != nil || frame.Bootstrap == nil || len(frame.Bootstrap.Display.Sketches) != 1 || frame.Bootstrap.Display.Sketches[0].Begin.ID != "s" {
		t.Fatalf("restored delivery: %+v %v", frame, err)
	}
	if len(b.st.Sketches.Snapshot().Sketches) != 1 {
		t.Fatal("UI reset mutated canonical sketch history")
	}
}

func TestPageLoadOperatorWireOwnedAndPauseKeepsScope(t *testing.T) {
	load := &recording.PageLoad{Target: "A", Loader: "L", Root: 9223372036854775807}
	r := loadPage(newRig(t), recording.SegLive, load)
	view := r.bd.OperatorView()
	if !view.PageLoadReady || !recording.SamePageLoad(view.PageLoad, load) {
		t.Fatalf("confirmed view: %+v", view)
	}
	view.PageLoad.Loader = "tampered"
	if got := r.bd.CurrentPageLoad(); got.Loader != "L" {
		t.Fatal("view aliases Board scope")
	}
	r.b.Add(&recording.Stop{Why: recording.StopPause})
	r.flush()
	if view := r.bd.OperatorView(); !view.Paused || !view.PageLoadReady || !recording.SamePageLoad(view.PageLoad, load) {
		t.Fatalf("pause lost known load: %+v", view)
	}
	r.bd.BeginPageLoad(1)
	view = r.bd.OperatorView()
	if view.PageLoadReady || view.PageLoad != nil || r.bd.CurrentPageLoad() != nil {
		t.Fatal("pending view claimed known load")
	}
}

func TestPageLoadColorQuotaFollowsSameLoadReturn(t *testing.T) {
	a := &recording.PageLoad{Target: "A", Loader: "L1", Root: 1}
	r := loadPage(newRig(t), recording.SegLive, a)
	for i := 0; i < broadcast.MaxNotesPerColor; i++ {
		if _, err := r.note(r.p1.ID, 0); err != nil {
			t.Fatal(err)
		}
	}
	loadPage(r, recording.SegTab, &recording.PageLoad{Target: "B", Loader: "L2", Root: 1})
	loadPage(r, recording.SegTab, a)
	if _, err := r.note(r.p1.ID, 0); !errors.Is(err, ErrColorFull) {
		t.Fatalf("same-load quota reset on activation: %v", err)
	}
}

func TestPageLoadInitialPendingContributesNoFeedbackWithoutRefusal(t *testing.T) {
	b := newRig(t).bd
	if _, err := b.Selection(); !errors.Is(err, ErrNotLive) {
		t.Fatalf("unstarted Board: %v", err)
	}
	b.BeginPageLoad(1)
	for _, phase := range []string{"pending", "unavailable"} {
		if phase == "unavailable" {
			b.UnavailablePageLoad(1)
		}
		selected, err := b.Selection()
		if err != nil || len(selected.Notes) != 0 || len(selected.Sketches) != 0 {
			t.Fatalf("%s initial capture refused ordinary submission: %+v %v", phase, selected, err)
		}
	}
}

func TestPageLoadUnknownRejectsFreshPageFeedback(t *testing.T) {
	for _, kind := range []string{"note", "pointer"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t)
			p := r.b.El("p", nil, r.b.Txt("Legacy"))
			r.b.Page(recording.SegLive, "http://legacy/", p)
			for _, record := range r.b.Recs {
				if err := r.w.Put(record); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			if kind == "note" {
				_, err = r.note(p.ID, 0)
			} else {
				err = r.bd.Stroke(StrokeIn{Viewer: ana, Seg: 1, ID: "pointer", Color: 0, Pts: [][3]float64{{1, 2, 0}}})
			}
			if !errors.Is(err, ErrStale) {
				t.Fatalf("unknown load admitted fresh %s: %v", kind, err)
			}
		})
	}
}
