package submission

import "testing"

func TestPreviewNeedsExactAcknowledgementBeforeFreeze(t *testing.T) {
	host := PreviewHost{ID: "console", Generation: 1}
	selection := PreviewSelection{Stamp: PreviewStamp{Board: 1, Delivered: 1}, Items: []PreviewItem{{RecordingID: "r", SketchID: "s", Sequence: 3, Digest: "d"}}}
	state, _ := NextPreview(PreviewState{}, PreviewHostChanged{Host: host})
	state, _ = NextPreview(state, PreviewCandidatesChanged{Selection: selection})
	state, effects := NextPreview(state, PreviewRefresh{})
	if state.Phase() != PreviewPublishing {
		t.Fatalf("phase=%v", state.Phase())
	}
	publication := effects[0].(PreviewPublish).Publication
	_, effects = NextPreview(state, PreviewPrepare{Host: host, Stamp: selection.Stamp})
	if len(effects) != 1 {
		t.Fatalf("prepare effects=%#v", effects)
	}
	if _, ok := effects[0].(PreviewRefuse); !ok {
		t.Fatalf("unacknowledged prepare=%#v", effects)
	}
	state, _ = NextPreview(state, PreviewPublishAccepted{Publication: publication})
	_, effects = NextPreview(state, PreviewPrepare{Host: host, Stamp: selection.Stamp})
	if len(effects) != 1 {
		t.Fatalf("freeze effects=%#v", effects)
	}
	if _, ok := effects[0].(PreviewFreezeProposal); !ok {
		t.Fatalf("acknowledged prepare=%#v", effects)
	}
}

func previewFixture(t *testing.T) (PreviewState, PreviewPublication) {
	t.Helper()
	s, _ := NextPreview(PreviewState{}, PreviewHostChanged{Host: PreviewHost{ID: "c", Generation: 1}})
	s, _ = NextPreview(s, PreviewCandidatesChanged{Selection: PreviewSelection{Stamp: PreviewStamp{Board: 4, Delivered: 2}, Items: []PreviewItem{{RecordingID: "r", SketchID: "s", Sequence: 8, Digest: "digest"}}, Remaining: 3}})
	s, fx := NextPreview(s, PreviewRefresh{})
	return s, fx[0].(PreviewPublish).Publication
}

func TestPreviewRetiredPublicationCannotEnableReplacement(t *testing.T) {
	for _, event := range []PreviewEvent{
		PreviewHostChanged{Host: PreviewHost{ID: "c", Generation: 2}},
		PreviewCandidatesChanged{Selection: PreviewSelection{Stamp: PreviewStamp{Board: 5, Delivered: 2}}},
		PreviewCandidatesChanged{Selection: PreviewSelection{Stamp: PreviewStamp{Board: 4, Delivered: 3}}},
	} {
		s, old := previewFixture(t)
		s, fx := NextPreview(s, event)
		if len(fx) != 2 {
			t.Fatalf("invalidation effects=%#v", fx)
		}
		if cancel, ok := fx[0].(PreviewCancel); !ok || !samePreviewPublication(cancel.Publication, old) {
			t.Fatalf("cancel=%#v", fx)
		}
		s, fx = NextPreview(s, PreviewRefresh{})
		fresh := fx[0].(PreviewPublish).Publication
		if fresh.Operation() <= old.Operation() || fresh.Revision() <= old.Revision() {
			t.Fatal("publication identity reused")
		}
		for _, late := range []PreviewEvent{PreviewPublishAccepted{old}, PreviewPublishFailed{old}, PreviewFreezeRejected{old}} {
			var effects []PreviewEffect
			s, effects = NextPreview(s, late)
			if s.Phase() != PreviewPublishing || len(effects) != 0 {
				t.Fatalf("late result changed replacement: phase=%v effects=%#v", s.Phase(), effects)
			}
		}
		s, _ = NextPreview(s, PreviewPublishAccepted{fresh})
		if s.Phase() != PreviewShown {
			t.Fatal("fresh result refused")
		}
	}
}

func TestPreviewFailureRetriesOnlyOnExplicitTrigger(t *testing.T) {
	s, p := previewFixture(t)
	s, fx := NextPreview(s, PreviewPublishFailed{p})
	if s.Phase() != PreviewDirty || len(fx) != 1 {
		t.Fatalf("failure phase=%v effects=%#v", s.Phase(), fx)
	}
	if _, ok := fx[0].(PreviewUnavailable); !ok {
		t.Fatalf("failure=%#v", fx)
	}
	s, fx = NextPreview(s, PreviewPublishAccepted{p})
	if s.Phase() != PreviewDirty || len(fx) != 0 {
		t.Fatal("late success revived failed publication")
	}
	s, fx = NextPreview(s, PreviewPrepare{Host: p.Host(), Stamp: p.Stamp()})
	if len(fx) != 2 {
		t.Fatalf("prepare should refuse and request refresh: %#v", fx)
	}
	s, fx = NextPreview(s, PreviewRefresh{})
	if s.Phase() != PreviewPublishing || len(fx) != 1 {
		t.Fatal("explicit retry failed")
	}
}

func TestPreviewPrepareAndFreezeRejectStaleSelection(t *testing.T) {
	for _, prepare := range []PreviewPrepare{
		{Host: PreviewHost{ID: "other", Generation: 1}, Stamp: PreviewStamp{Board: 4, Delivered: 2}},
		{Host: PreviewHost{ID: "c", Generation: 2}, Stamp: PreviewStamp{Board: 4, Delivered: 2}},
		{Host: PreviewHost{ID: "c", Generation: 1}, Stamp: PreviewStamp{Board: 5, Delivered: 2}},
		{Host: PreviewHost{ID: "c", Generation: 1}, Stamp: PreviewStamp{Board: 4, Delivered: 3}},
	} {
		s, p := previewFixture(t)
		s, _ = NextPreview(s, PreviewPublishAccepted{p})
		s, fx := NextPreview(s, prepare)
		if s.Phase() != PreviewDirty || len(fx) != 2 {
			t.Fatalf("mismatch failed to invalidate: %v %#v", s.Phase(), fx)
		}
		for _, effect := range fx {
			if _, ok := effect.(PreviewFreezeProposal); ok {
				t.Fatal("stale freeze")
			}
		}
	}
	s, p := previewFixture(t)
	s, _ = NextPreview(s, PreviewPublishAccepted{p})
	s, fx := NextPreview(s, PreviewFreezeRejected{p})
	if s.Phase() != PreviewDirty || len(fx) != 2 {
		t.Fatalf("atomic freeze rejection=%v %#v", s.Phase(), fx)
	}
}

func TestPreviewSelectionCannotBeMutatedByCaller(t *testing.T) {
	selection := PreviewSelection{Stamp: PreviewStamp{Board: 1}, Items: []PreviewItem{{SketchID: "original"}}}
	s, _ := NextPreview(PreviewState{}, PreviewCandidatesChanged{Selection: selection})
	selection.Items[0].SketchID = "input mutation"
	s, _ = NextPreview(s, PreviewHostChanged{Host: PreviewHost{ID: "c"}})
	s, fx := NextPreview(s, PreviewRefresh{})
	p := fx[0].(PreviewPublish).Publication
	exposed := p.Selection()
	exposed.Items[0].SketchID = "getter mutation"
	if p.Selection().Items[0].SketchID != "original" {
		t.Fatal("publication aliases caller's data")
	}
	s, _ = NextPreview(s, PreviewPublishAccepted{p})
	_, fx = NextPreview(s, PreviewPrepare{Host: p.Host(), Stamp: p.Stamp()})
	frozen := fx[0].(PreviewFreezeProposal).Publication.Selection()
	if frozen.Items[0].SketchID != "original" {
		t.Fatal("effect changed authoritative selection")
	}
}

func TestPreviewHostRemovalAndTerminalDisposal(t *testing.T) {
	s, p := previewFixture(t)
	s, fx := NextPreview(s, PreviewHostChanged{})
	if s.Phase() != PreviewDormant || len(fx) != 1 {
		t.Fatalf("host removal=%v %#v", s.Phase(), fx)
	}
	s, _ = NextPreview(s, PreviewHostChanged{Host: p.Host()})
	if s.Phase() != PreviewDirty {
		t.Fatal("candidate was lost across host removal")
	}
	s, _ = NextPreview(s, PreviewRefresh{})
	s, fx = NextPreview(s, PreviewDispose{})
	if s.Phase() != PreviewStopped || len(fx) != 2 {
		t.Fatalf("dispose=%v %#v", s.Phase(), fx)
	}
	for _, event := range []PreviewEvent{PreviewHostChanged{Host: p.Host()}, PreviewCandidatesChanged{}, PreviewRefresh{}, PreviewPrepare{}, PreviewPublishAccepted{p}, PreviewPublishFailed{p}, PreviewFreezeRejected{p}, PreviewDispose{}} {
		s, fx = NextPreview(s, event)
		if s.Phase() != PreviewStopped || len(fx) != 0 {
			t.Fatalf("stopped revived by %#v", event)
		}
	}
}

// Exercise reordered callbacks through the production transition. The oracle
// records only externally observed publication ownership, independently of the
// private state representation, and never calls a mock policy.
func checkPreviewControlOrder(t *testing.T, data []byte) {
	t.Helper()
	var state PreviewState
	var host PreviewHost
	var stamp PreviewStamp
	var issued []PreviewPublication
	var pending, shown *PreviewPublication
	var lastOperation uint64
	stopped := false
	if len(data) > 4096 {
		data = data[:4096]
	}
	for step, b := range data {
		var event PreviewEvent
		var result PreviewPublication
		if len(issued) > 0 {
			result = issued[(int(b)/11)%len(issued)]
		}
		switch b % 11 {
		case 0:
			host = PreviewHost{ID: "c", Generation: uint64(step + 1)}
			event = PreviewHostChanged{host}
			pending = nil
			shown = nil
		case 1:
			stamp.Board++
			event = PreviewCandidatesChanged{PreviewSelection{Stamp: stamp}}
			pending = nil
			shown = nil
		case 2:
			stamp.Delivered++
			event = PreviewCandidatesChanged{PreviewSelection{Stamp: stamp}}
			pending = nil
			shown = nil
		case 3:
			event = PreviewRefresh{}
		case 4:
			event = PreviewPublishAccepted{result}
			if pending != nil && pending.Operation() == result.Operation() {
				p := *pending
				shown = &p
				pending = nil
			}
		case 5:
			event = PreviewPublishFailed{result}
			if pending != nil && pending.Operation() == result.Operation() {
				pending = nil
				shown = nil
			}
		case 6:
			event = PreviewPrepare{Host: host, Stamp: stamp}
		case 7:
			event = PreviewPrepare{Host: PreviewHost{ID: "retired"}, Stamp: stamp}
			shown = nil
		case 8:
			host = PreviewHost{}
			event = PreviewHostChanged{}
			pending = nil
			shown = nil
		case 9:
			event = PreviewFreezeRejected{result}
			if shown != nil && shown.Operation() == result.Operation() {
				shown = nil
			}
		case 10:
			event = PreviewDispose{}
			stopped = true
			pending = nil
			shown = nil
		}
		var effects []PreviewEffect
		state, effects = NextPreview(state, event)
		if stopped {
			if state.Phase() != PreviewStopped {
				t.Fatalf("step%d: stopped state revived", step)
			}
			for _, effect := range effects {
				switch effect.(type) {
				case PreviewCancel, PreviewStop:
				default:
					t.Fatalf("step%d: stopped effect %#v", step, effect)
				}
			}
			continue
		}
		for _, effect := range effects {
			switch e := effect.(type) {
			case PreviewPublish:
				p := e.Publication
				if p.Operation() <= lastOperation || p.Revision() != p.Operation() || p.Host() != host || p.Stamp() != stamp {
					t.Fatalf("step%d: publication has stale identity %#v", step, p)
				}
				lastOperation = p.Operation()
				issued = append(issued, p)
				pending = &p
				shown = nil
			case PreviewFreezeProposal:
				if shown == nil || shown.Operation() != e.Publication.Operation() || e.Publication.Host() != host || e.Publication.Stamp() != stamp {
					t.Fatalf("step%d: freeze without current acknowledged projection", step)
				}
			}
		}
		if state.Phase() == PreviewShown && shown == nil {
			t.Fatalf("step%d: stale callback enabled Shown", step)
		}
	}
}

func TestPreviewGeneratedControlOrder(t *testing.T) {
	// Avoid early disposal in most traces, then append one terminal event.
	for seed := uint64(1); seed <= 100; seed++ {
		x := seed
		data := make([]byte, 501)
		for i := range data[:500] {
			x = x*6364136223846793005 + 1442695040888963407
			b := byte(x >> 32)
			if b%11 == 10 {
				b--
			}
			data[i] = b
		}
		data[500] = 10
		checkPreviewControlOrder(t, data)
	}
}

func FuzzPreviewControlOrder(f *testing.F) {
	f.Add([]byte{0, 1, 3, 4, 6, 2, 3, 4, 5, 6, 0, 3, 4, 6, 10})
	f.Add([]byte{1, 0, 3, 0, 3, 15, 4, 6, 7, 3, 4, 9, 3, 5, 6})
	f.Fuzz(func(t *testing.T, data []byte) { checkPreviewControlOrder(t, data) })
}

func TestPreviewDisposalAlwaysStopsOwnedWorker(t *testing.T) {
	for _, initial := range []PreviewState{{}, {payload: previewDirty{}}, {payload: previewShown{}}} {
		s, fx := NextPreview(initial, PreviewDispose{})
		if s.Phase() != PreviewStopped {
			t.Fatal("disposal not terminal")
		}
		if len(fx) != 1 {
			t.Fatalf("disposal must request owned-worker stop: %#v", fx)
		}
		if _, ok := fx[0].(PreviewStop); !ok {
			t.Fatalf("missing stop: %#v", fx)
		}
	}
}

func TestPreviewIdenticalInputsDoNotRefreshProjection(t *testing.T) {
	for _, acknowledged := range []bool{false, true} {
		state, publication := previewFixture(t)
		if acknowledged {
			state, _ = NextPreview(state, PreviewPublishAccepted{publication})
		}
		phase := state.Phase()
		for _, event := range []PreviewEvent{PreviewHostChanged{Host: publication.Host()}, PreviewCandidatesChanged{Selection: publication.Selection()}} {
			var effects []PreviewEffect
			state, effects = NextPreview(state, event)
			if state.Phase() != phase || len(effects) != 0 {
				t.Fatalf("identical input disturbed phase %v: phase=%v effects=%#v", phase, state.Phase(), effects)
			}
		}
	}
}

func TestPreviewSameStampWithChangedContentsInvalidates(t *testing.T) {
	for _, change := range []func(*PreviewSelection){
		func(s *PreviewSelection) { s.Items[0].Digest = "new digest" },
		func(s *PreviewSelection) { s.Remaining++ },
	} {
		state, p := previewFixture(t)
		state, _ = NextPreview(state, PreviewPublishAccepted{p})
		selection := p.Selection()
		change(&selection)
		state, effects := NextPreview(state, PreviewCandidatesChanged{Selection: selection})
		if state.Phase() != PreviewDirty || len(effects) != 1 {
			t.Fatalf("changed contents ignored: phase=%v effects=%#v", state.Phase(), effects)
		}
	}
}

// pair#849: a refused prepare says which condition refused.
func TestPreviewRefusalNamesItsCondition(t *testing.T) {
	host := PreviewHost{ID: "console", Generation: 1}
	stamp := PreviewStamp{Board: 5, Delivered: 1}
	refusal := func(state PreviewState, e PreviewPrepare) string {
		t.Helper()
		_, effects := NextPreview(state, e)
		for _, effect := range effects {
			if r, ok := effect.(PreviewRefuse); ok {
				return r.Reason
			}
		}
		t.Fatalf("not refused: %#v", effects)
		return ""
	}
	if got := refusal(PreviewState{}, PreviewPrepare{Host: host, Stamp: stamp}); got != "the current feedback has not been shown yet (no preview is shown)" {
		t.Fatalf("unshown: %q", got)
	}
	state, _ := NextPreview(PreviewState{}, PreviewHostChanged{Host: host})
	state, _ = NextPreview(state, PreviewCandidatesChanged{Selection: PreviewSelection{Stamp: stamp}})
	state, effects := NextPreview(state, PreviewRefresh{})
	state, _ = NextPreview(state, PreviewPublishAccepted{Publication: effects[0].(PreviewPublish).Publication})
	if got := refusal(state, PreviewPrepare{Host: host, Stamp: PreviewStamp{Board: 6, Delivered: 1}}); got != "the feedback changed since the preview was shown (board 5→6, delivered 1→1)" {
		t.Fatalf("moved board: %q", got)
	}
	if got := refusal(state, PreviewPrepare{Host: PreviewHost{ID: "console", Generation: 2}, Stamp: stamp}); got != "the private console host changed since the preview was shown" {
		t.Fatalf("moved host: %q", got)
	}
}
