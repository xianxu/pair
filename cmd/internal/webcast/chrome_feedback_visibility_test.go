package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/cdp"
	core "github.com/xianxu/pair/cmd/internal/submission"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

func feedbackWireFixture() core.FeedbackVisibility {
	return core.FeedbackVisibility{Revision: 1, Scene: sketchimage.PageScene{RecordingID: "rec", SketchSourceFacts: query.SketchSourceFacts{T: 0, RecordIndex: 0, Doc: 1, Tab: "tab", URL: "https://example.test/", Title: "", VP: recording.Viewport{W: 100, H: 100, DPR: 1}}}, MaskScope: "mask", MaskRevision: 0, Sketches: []core.VisibleSketch{{ID: "shape", Sequence: 2, Digest: strings.Repeat("a", 64), Paths: []core.VisiblePath{{ID: "path", Locator: []int{1, 2, 0}, Box: recording.Box{1, 2, 10, 20}, Clip: recording.Box{0, 0, 100, 100}}}}}, Notes: []core.PreviewNote{{ID: "note", Revision: 0}}}
}
func visibilityBinding(session string, world int64, revision string) cdp.Event {
	payload, _ := json.Marshal(map[string]any{"t": "feedback-view", "revision": revision})
	body, _ := json.Marshal(map[string]any{"name": bindingName, "executionContextId": world, "payload": string(payload)})
	return cdp.Event{SessionID: session, Method: "Runtime.bindingCalled", Params: body}
}
func TestFeedbackVisibilityStrictWire(t *testing.T) {
	wire, _ := json.Marshal(feedbackWireFixture())
	if _, err := decodeFeedbackVisibility(wire); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(wire), `"revision":"1"`, `"revision":"01"`, 1), strings.Replace(string(wire), `"mask_revision":"0"`, `"mask_revision":null`, 1), strings.Replace(string(wire), `"record_index":0,`, "", 1), strings.Replace(string(wire), `"revision":"1"`, `"Revision":"1"`, 1), strings.Replace(string(wire), `"notes":[{"id":"note","revision":"0"}]`, `"notes":null`, 1), strings.Replace(string(wire), `"title":""`, `"title":null`, 1), strings.Replace(string(wire), `"mask_scope":"mask"`, `"mask_scope":"mask","mask_scope":"other"`, 1)} {
		if _, err := decodeFeedbackVisibility([]byte(bad)); err == nil {
			t.Fatalf("accepted malformed visibility %s", bad)
		}
	}
}
func TestFeedbackVisibilityBindingExactScopeAndSameRevisionInvalidates(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 99, "1"))
	if c.submissionPreview.visibilityRevision != 0 {
		t.Fatal("page spoofed private context")
	}
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	if c.submissionPreview.visibilityRevision != 1 {
		t.Fatal("trusted revision not bound")
	}
	view := feedbackWireFixture()
	c.submissionPreview.visibility = &view
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	if c.submissionPreview.visibility != nil {
		t.Fatal("same revision invalidation retained cached paint")
	}
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "2"))
	c.submissionPreview.visibility = &view
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	if c.submissionPreview.visibility == nil {
		t.Fatal("stale notification invalidated newer frame")
	}
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "02"))
	if c.submissionPreview.visibilityRevision != 2 {
		t.Fatal("noncanonical revision admitted")
	}
}

type visibilityCaller func(context.Context, string, string, any, any) error

func (f visibilityCaller) Call(ctx context.Context, s, m string, p, out any) error {
	return f(ctx, s, m, p, out)
}
func visibilityReply(out any, raw []byte) error {
	response := []byte(`{"result":{"value":` + string(raw) + `}}`)
	return json.Unmarshal(response, out)
}
func TestFeedbackVisibilityReadFencesLateRevisionAndOwnsResult(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, _, _ := c.SubmissionPreviewHost(context.Background())
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	view := feedbackWireFixture()
	wire, _ := json.Marshal(view)
	call := visibilityCaller(func(_ context.Context, session, method string, params, out any) error {
		if session != "s" || method != "Runtime.evaluate" {
			t.Fatal("untrusted scope")
		}
		return visibilityReply(out, wire)
	})
	got, err := c.readSubmissionVisibility(context.Background(), host, call)
	if err != nil {
		t.Fatal(err)
	}
	got.Sketches[0].Paths[0].Locator[0] = 999
	if c.submissionPreview.visibility.Sketches[0].Paths[0].Locator[0] != 1 {
		t.Fatal("returned visibility aliased authoritative cache")
	}
	// A controlled in-flight binding change must reject a reply from revision1.
	changing := visibilityCaller(func(_ context.Context, _, _ string, _, out any) error {
		c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "2"))
		return visibilityReply(out, wire)
	})
	if _, err = c.readSubmissionVisibility(context.Background(), host, changing); err == nil {
		t.Fatal("late old-revision read accepted")
	}
	if c.submissionPreview.visibility != nil {
		t.Fatal("late read restored invalidated cache")
	}
}
func TestFeedbackVisibilityResolutionPreservesChosenPathOrderAndFencesMutation(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, _, _ := c.SubmissionPreviewHost(context.Background())
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	view := feedbackWireFixture()
	c.submissionPreview.visibility = &view
	item := core.PreviewItem{RecordingID: "rec", SketchID: "shape", Sequence: 2, Digest: strings.Repeat("a", 64)}
	selection := core.PreviewSelection{Items: []core.PreviewItem{item}, Candidates: []core.PreviewSketchCandidate{{Item: item}}, Visibility: &view, Page: &view.Scene, Notes: view.Notes}
	resolver := func(_ context.Context, session string, doc int, tab string, paths [][]int) ([]recording.NodeID, error) {
		if session != "s" || doc != 1 || tab != "tab" || !reflect.DeepEqual(paths, [][]int{{1, 2, 0}}) {
			t.Fatalf("wrong ordered locator resolution %s %d %s %v", session, doc, tab, paths)
		}
		paths[0][0] = 999
		return []recording.NodeID{7}, nil
	}
	got, err := c.resolveSubmissionPreview(context.Background(), host, selection, resolver)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.ResolvedNodes, []recording.NodeID{7}) || view.Sketches[0].Paths[0].Locator[0] != 1 {
		t.Fatal("resolution mutated paths or lost IDs")
	}
	got.ResolvedNodes[0] = 900
	if c.submissionPreview.resolvedSelection.ResolvedNodes[0] != 7 {
		t.Fatal("returned proof aliased cache")
	}
	changing := func(context.Context, string, int, string, [][]int) ([]recording.NodeID, error) {
		c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
		return []recording.NodeID{7}, nil
	}
	if _, err = c.resolveSubmissionPreview(context.Background(), host, selection, changing); err == nil {
		t.Fatal("same-revision invalidation crossed path IO")
	}
	if c.submissionPreview.resolvedSelection != nil {
		t.Fatal("late resolver restored proof")
	}
}
func TestFeedbackVisibilityReadCancellationLeavesNoAuthority(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, _, _ := c.SubmissionPreviewHost(context.Background())
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.readSubmissionVisibility(ctx, host, visibilityCaller(func(context.Context, string, string, any, any) error {
		t.Fatal("cancelled read performed IO")
		return errors.New("unreachable")
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func resolvedVisibilityChromeFixture(t *testing.T) (*Chrome, *loopState, core.PreviewHost, core.PreviewSelection) {
	t.Helper()
	c, s := previewChromeFixture(t)
	b := recording.NewBuilder("rec")
	node := b.El("button", []string{"id", "current"})
	b.Page(recording.SegLive, "https://example.test/", node)
	op := func(seq uint64) recording.SketchOperation {
		return recording.SketchOperation{Operation: strconv.FormatUint(seq, 10), Sequence: seq, Digest: strings.Repeat("a", 64)}
	}
	b.At(10).Add(&recording.SketchBegin{SketchOperation: op(0), ID: "shape", Author: recording.Author{ID: "operator", Color: "#ff0000"}, Owner: "operator", Doc: 1, Tab: "tab1", VP: recording.Viewport{W: 1280, H: 800, DPR: 1}})
	b.At(11).Add(&recording.SketchStrokeBegin{SketchOperation: op(1), Sketch: "shape", ID: "p", At: recording.NoteAt{Node: node.ID, Box: &recording.Box{0, 0, 100, 100}}, Width: 3})
	b.At(12).Add(&recording.SketchPoints{SketchOperation: op(2), Sketch: "shape", ID: "p", Pts: []recording.SketchPoint{{20, 20, .2}, {40, 40, .8}}})
	b.At(13).Add(&recording.SketchStrokeEnd{SketchOperation: op(3), Sketch: "shape", ID: "p"})
	b.At(14).Add(&recording.SketchLabel{SketchOperation: op(4), Sketch: "shape", Label: "saved"})
	for _, r := range b.Recs {
		if err := c.opts.Record.Operator.Put(r); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := c.opts.Record.Operator.Selection()
	if err != nil {
		t.Fatal(err)
	}
	operator := c.opts.Record.Operator.OperatorView()
	view := core.FeedbackVisibility{Revision: 1, Scene: operator.FeedbackContext, MaskScope: operator.FeedbackMask.Scope, MaskRevision: operator.FeedbackMask.Revision, Notes: []core.PreviewNote{}, Sketches: []core.VisibleSketch{{ID: "shape", Sequence: snapshot.Sketches[0].NextSequence, Digest: snapshot.Sketches[0].Digest, Paths: []core.VisiblePath{{ID: "p", Locator: []int{1, 0}, Box: recording.Box{0, 0, 100, 100}, Clip: recording.Box{0, 0, 1280, 800}}}}}}
	c.refreshSubmissionPreviewHost(s)
	host, _, err := c.SubmissionPreviewHost(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	wire, _ := json.Marshal(view)
	if _, err = c.readSubmissionVisibility(context.Background(), host, visibilityCaller(func(_ context.Context, _, _ string, _, out any) error { return visibilityReply(out, wire) })); err != nil {
		t.Fatal(err)
	}
	var candidates []core.PreviewSketchCandidate
	for _, v := range snapshot.Sketches {
		candidates = append(candidates, core.PreviewSketchCandidate{Item: core.PreviewItem{RecordingID: "rec", SketchID: v.ID, Sequence: v.NextSequence, Digest: v.Digest}, Source: v.Source, Finish: v.Finish})
	}
	selection, err := core.SelectVisibleFeedback(candidates, nil, core.DeliveredFeedback{RecordingID: "rec"}, core.PreviewStamp{Board: snapshot.SketchGeneration}, view)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := c.resolveSubmissionPreview(context.Background(), host, selection, func(context.Context, string, int, string, [][]int) ([]recording.NodeID, error) {
		return []recording.NodeID{node.ID}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, s, host, resolved
}
func TestFeedbackVisibilityFreezeRequiresExactResolvedCache(t *testing.T) {
	c, s, host, selection := resolvedVisibilityChromeFixture(t)
	if frozen, err := c.FreezeSubmissionPreview(host, selection); err != nil || len(frozen.Sketches) != 1 || frozen.Current == nil {
		t.Fatalf("valid trusted proof refused %+v %v", frozen, err)
	}
	changed := cloneVisibilitySelection(selection)
	changed.ResolvedNodes[0]++
	if _, err := c.FreezeSubmissionPreview(host, changed); err == nil {
		t.Fatal("mutated returned resolved node accepted")
	}
	changed = cloneVisibilitySelection(selection)
	changed.Visibility.Sketches[0].Paths[0].Box[0]++
	if _, err := c.FreezeSubmissionPreview(host, changed); err == nil {
		t.Fatal("mutated paint metadata accepted")
	}
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	if _, err := c.FreezeSubmissionPreview(host, selection); err == nil {
		t.Fatal("same-revision invalidation retained freeze authority")
	}
}
func TestFeedbackVisibilityReadRejectsRetiredHostAndWrongRecording(t *testing.T) {
	for _, retire := range []bool{false, true} {
		c, s := previewChromeFixture(t)
		c.refreshSubmissionPreviewHost(s)
		host, _, _ := c.SubmissionPreviewHost(context.Background())
		c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
		view := feedbackWireFixture()
		if !retire {
			view.Scene.RecordingID = "other"
		}
		wire, _ := json.Marshal(view)
		_, err := c.readSubmissionVisibility(context.Background(), host, visibilityCaller(func(_ context.Context, _, _ string, _, out any) error {
			if retire {
				c.retireOperatorSketchScopes(s, "s")
				c.refreshSubmissionPreviewHost(s)
			}
			return visibilityReply(out, wire)
		}))
		if err == nil {
			t.Fatal("retired/wrong recording read accepted")
		}
	}
}

func TestFeedbackVisibilityAllFieldsRequireCanonicalPresence(t *testing.T) {
	for _, group := range []struct {
		name   string
		fields []string
	}{
		{"root", []string{"revision", "scene", "mask_scope", "mask_revision", "sketches", "notes"}},
		{"scene", []string{"recording_id", "t", "record_index", "doc", "tab", "url", "title", "vp", "scroll"}},
		{"viewport", []string{"w", "h", "dpr"}},
		{"sketch", []string{"id", "next_sequence", "digest", "paths"}},
		{"path", []string{"id", "locator", "box", "clip"}},
		{"note", []string{"id", "revision"}},
	} {
		for _, field := range group.fields {
			for _, null := range []bool{false, true} {
				t.Run(group.name+"/"+field+"/"+strconv.FormatBool(null), func(t *testing.T) {
					raw, _ := json.Marshal(feedbackWireFixture())
					var root map[string]any
					json.Unmarshal(raw, &root)
					target := root
					switch group.name {
					case "scene":
						target = root["scene"].(map[string]any)
					case "viewport":
						target = root["scene"].(map[string]any)["vp"].(map[string]any)
					case "sketch":
						target = root["sketches"].([]any)[0].(map[string]any)
					case "path":
						target = root["sketches"].([]any)[0].(map[string]any)["paths"].([]any)[0].(map[string]any)
					case "note":
						target = root["notes"].([]any)[0].(map[string]any)
					}
					if null {
						target[field] = nil
					} else {
						delete(target, field)
					}
					raw, _ = json.Marshal(root)
					if _, err := decodeFeedbackVisibility(raw); err == nil {
						t.Fatal("missing/null required field was accepted")
					}
				})
			}
		}
	}
}
func TestFeedbackVisibilityResolutionIOCannotHoldPreviewMutex(t *testing.T) {
	c, s, host, selection := resolvedVisibilityChromeFixture(t)
	arrived, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := c.resolveSubmissionPreview(context.Background(), host, selection, func(context.Context, string, int, string, [][]int) ([]recording.NodeID, error) {
			close(arrived)
			<-release
			return []recording.NodeID{1}, nil
		})
		done <- err
	}()
	<-arrived
	// This same-revision event must synchronously acquire the preview mutex and
	// retire authority while the recorder actor is still held.
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	close(release)
	if err := <-done; err == nil {
		t.Fatal("cancelled locator IO restored authoritative proof")
	}
}
func TestFeedbackVisibilityResolutionUsesChosenOrderNotPaintOrder(t *testing.T) {
	c, s := previewChromeFixture(t)
	c.refreshSubmissionPreviewHost(s)
	host, _, _ := c.SubmissionPreviewHost(context.Background())
	c.handleFeedbackVisibilityBinding(s, visibilityBinding("s", 17, "1"))
	v := feedbackWireFixture()
	other := v.Sketches[0]
	other.ID = "second"
	other.Paths = append([]core.VisiblePath(nil), other.Paths...)
	other.Paths[0].Locator = []int{7, 0}
	v.Sketches = append(v.Sketches, other)
	c.submissionPreview.visibility = &v
	selection := core.PreviewSelection{Visibility: &v, Page: &v.Scene}
	for _, id := range []string{"second", "shape"} {
		item := core.PreviewItem{RecordingID: "rec", SketchID: id, Sequence: 2, Digest: strings.Repeat("a", 64)}
		selection.Items = append(selection.Items, item)
		selection.Candidates = append(selection.Candidates, core.PreviewSketchCandidate{Item: item})
	}
	_, err := c.resolveSubmissionPreview(context.Background(), host, selection, func(_ context.Context, _ string, _ int, _ string, paths [][]int) ([]recording.NodeID, error) {
		if !reflect.DeepEqual(paths, [][]int{{7, 0}, {1, 2, 0}}) {
			t.Fatalf("paint order replaced chosen order %v", paths)
		}
		return []recording.NodeID{9, 7}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
