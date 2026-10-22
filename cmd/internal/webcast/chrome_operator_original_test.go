package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"testing"
	"time"
)

func TestOperatorOriginalBindingUsesExactDebuggerObjectAndReleases(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprint(accepted), func(t *testing.T) {
			c, p := startPeer(t, func(p *peer) { p.ignore["DOM.resolveNode"] = true; p.ignore["Runtime.callFunctionOn"] = true })
			done := make(chan bool, 1)
			go func() {
				done <- c.bindOperatorOriginal(context.Background(), operatorTarget{session: "S1", context: 17, active: true}, 3, 42, 901)
			}()
			call := p.expect("S1", "DOM.resolveNode")
			var args map[string]any
			json.Unmarshal(call.Params, &args)
			if args["backendNodeId"] != float64(901) || args["executionContextId"] != float64(17) || args["objectGroup"] != "couchcast-operator-originals" {
				t.Fatal(args)
			}
			p.send(fmt.Sprintf(`{"id":%d,"result":{"object":{"objectId":"trusted-node"}}}`, call.ID))
			call = p.expect("S1", "Runtime.callFunctionOn")
			json.Unmarshal(call.Params, &args)
			if args["objectId"] != "trusted-node" {
				t.Fatal(args)
			}
			values := args["arguments"].([]any)
			if values[0].(map[string]any)["value"] != float64(3) || values[1].(map[string]any)["value"] != float64(42) {
				t.Fatal(args)
			}
			p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":%t}}}`, call.ID, accepted))
			p.expect("S1", "Runtime.releaseObjectGroup")
			if got := <-done; got != accepted {
				t.Fatalf("bound=%v", got)
			}
		})
	}
}

func TestOperatorOriginalResolveTimeoutStillReleasesObjectGroup(t *testing.T) {
	c, p := startPeer(t, func(p *peer) { p.ignore["DOM.resolveNode"] = true })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- c.bindOperatorOriginal(ctx, operatorTarget{session: "S1", context: 17}, 3, 42, 901) }()
	p.expect("S1", "DOM.resolveNode")
	p.expect("S1", "Runtime.releaseObjectGroup")
	if <-done {
		t.Fatal("unconfirmed identity accepted")
	}
}

func TestOperatorOriginalCacheBoundsRetriesAndDocumentLifetime(t *testing.T) {
	cache := &operatorOriginals{}
	view := board.OperatorView{Doc: 1, Revision: 1}
	for i := 1; i <= operatorOriginalBatch+1; i++ {
		view.Notes = append(view.Notes, &recording.Note{ID: fmt.Sprint(i), At: recording.NoteAt{Node: recording.NodeID(i)}})
	}
	ids, more := cache.next(view)
	if len(ids) != operatorOriginalBatch || !more {
		t.Fatalf("batch %d more %v", len(ids), more)
	}
	for _, id := range ids {
		cache.nodes[id] = operatorOriginalAttempt{bound: true, revision: 1, tries: 1}
	}
	ids, more = cache.next(view)
	if len(ids) != 1 || more {
		t.Fatalf("cached ack: %v %v", ids, more)
	}
	remaining := ids[0]
	cache.nodes[remaining] = operatorOriginalAttempt{revision: 1, tries: 2}
	ids, _ = cache.next(view)
	if len(ids) != 1 {
		t.Fatal("missing retry")
	}
	cache.nodes[remaining] = operatorOriginalAttempt{revision: 1, tries: 3}
	ids, _ = cache.next(view)
	if len(ids) != 0 {
		t.Fatal("unbounded retries")
	}
	view.Revision++
	ids, _ = cache.next(view)
	if len(ids) != 1 || ids[0] != remaining {
		t.Fatal("new snapshot cannot retry")
	}
	view.Notes = nil
	cache.next(view)
	if len(cache.nodes) != 0 {
		t.Fatal("retired notes retained")
	}
	cache.nodes[1] = operatorOriginalAttempt{bound: true}
	view.Doc = 2
	view.Notes = []*recording.Note{{At: recording.NoteAt{Node: 1}}}
	ids, _ = cache.next(view)
	if len(ids) != 1 || cache.nodes[1].bound {
		t.Fatal("old document identity retained")
	}
}

func TestOperatorOriginalCacheRetainsSketchAnchorsWithoutNotes(t *testing.T) {
	cache := &operatorOriginals{sketchDocument: 1, sketchNodes: []recording.NodeID{77}}
	view := board.OperatorView{Doc: 1, Revision: 1}
	ids, _ := cache.next(view)
	if len(ids) != 1 || ids[0] != 77 {
		t.Fatalf("sketch IDs: %v", ids)
	}
	cache.nodes[77] = operatorOriginalAttempt{bound: true}
	if ids, _ = cache.next(view); len(ids) != 0 {
		t.Fatal("unchanged original rebound")
	}
	cache.sketchNodes = nil
	cache.next(view)
	if len(cache.nodes) != 0 {
		t.Fatal("cleared sketch retained original")
	}
}

func TestOperatorOriginalSketchRevisionRebindsAfterCoalescedClear(t *testing.T) {
	cache := &operatorOriginals{}
	view := board.OperatorView{Doc: 1, Revision: 1, Notes: []*recording.Note{{At: recording.NoteAt{Node: 88}}}}
	cache.acceptSketchAnchors(1, 1, []recording.NodeID{77})
	cache.next(view)
	cache.nodes[77] = operatorOriginalAttempt{bound: true, revision: 1, tries: 1}
	cache.nodes[88] = operatorOriginalAttempt{bound: true, revision: 1, tries: 1}
	// The browser has consumed clear and pruned77, while a new sketch's same
	// anchor is awaiting delivery. Go still has the prior bound77 cache.
	cache.sketchPending = true
	if ids, _ := cache.next(view); len(ids) != 0 {
		t.Fatalf("delivery gap spent binding attempts: %v", ids)
	}
	cache.acceptSketchAnchors(1, 3, []recording.NodeID{77})
	ids, _ := cache.next(view)
	if len(ids) != 1 || ids[0] != 77 {
		t.Fatalf("cleared browser proof was not rebound: %v", ids)
	}
	if !cache.nodes[88].bound {
		t.Fatal("unrelated note proof reset")
	}
	cache.nodes[77] = operatorOriginalAttempt{bound: true, tries: 1}
	cache.acceptSketchAnchors(1, 3, []recording.NodeID{77})
	if ids, _ := cache.next(view); len(ids) != 0 {
		t.Fatalf("unchanged revision rebound: %v", ids)
	}
}

func TestOperatorOriginalSketchPendingDoesNotSpendRetriesAndResetsOldNewAnchors(t *testing.T) {
	cache := &operatorOriginals{}
	view := board.OperatorView{Doc: 1, Revision: 1, Notes: []*recording.Note{{At: recording.NoteAt{Node: 88}}}}
	cache.acceptSketchAnchors(1, 1, []recording.NodeID{77, 88})
	cache.next(view)
	cache.nodes[77] = operatorOriginalAttempt{revision: 1, tries: 2}
	cache.sketchPending = true
	ids, _ := cache.next(view)
	if len(ids) != 1 || ids[0] != 88 {
		t.Fatalf("pending anchors attempted, or note blocked: %v", ids)
	}
	if cache.nodes[77].tries != 2 {
		t.Fatal("pending sketch spent retry")
	}
	cache.nodes[88] = operatorOriginalAttempt{bound: true}
	cache.nodes[99] = operatorOriginalAttempt{bound: true}
	cache.acceptSketchAnchors(1, 2, []recording.NodeID{99})
	ids, _ = cache.next(view)
	if len(ids) != 2 {
		t.Fatalf("old/new attempts not invalidated: %v", ids)
	}
	for _, id := range ids {
		if id != 88 && id != 99 {
			t.Fatalf("retired anchor remained: %v", ids)
		}
	}
	cache.acceptSketchAnchors(2, 2, []recording.NodeID{101})
	view.Doc = 2
	view.Notes = nil
	ids, _ = cache.next(view)
	if len(ids) != 1 || ids[0] != 101 {
		t.Fatalf("document replacement reused old anchor: %v", ids)
	}
}
