package recorder

import (
	"encoding/json"
	"errors"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeControlUI models document-owned suppression tokens. A context belongs to
// the actual fake page object: replacing that page invalidates its confirmations
// immediately, even before the recorder consumes context-destroyed events.
type fakeControlUI struct {
	mu     sync.Mutex
	worlds map[string]*fakeUIWorld
	calls  []string
}
type fakeUIWorld struct {
	id     int
	page   *fakePage
	tokens map[string]int64
	busy   bool
}

func (h *harness) confirmUI(session, frame string, id int) *fakeControlUI {
	if h.ui == nil {
		h.ui = &fakeControlUI{worlds: map[string]*fakeUIWorld{}}
		h.fc.mu.Lock()
		h.fc.ui = h.ui
		h.fc.mu.Unlock()
	}
	h.fc.mu.Lock()
	page := h.fc.pages[session]
	h.fc.mu.Unlock()
	h.ui.mu.Lock()
	old := h.ui.worlds[session]
	h.ui.worlds[session] = &fakeUIWorld{id: id, page: page, tokens: map[string]int64{}}
	h.ui.mu.Unlock()
	if old != nil && old.id != id {
		h.send(session, "Runtime.executionContextDestroyed", map[string]any{"executionContextId": old.id})
	}
	h.ctxCreated(session, frame, id)
	return h.ui
}
func (f *fakeControlUI) more(session, method string, p map[string]any, page *fakePage) (any, error, bool) {
	expr, _ := p["expression"].(string)
	if method != "Runtime.evaluate" || !strings.Contains(expr, "__couchcastUI(") {
		return nil, nil, false
	}
	start := strings.Index(expr, "...")
	end := strings.LastIndex(expr, ")")
	if start < 0 || end < start {
		return nil, errors.New("bad UI command"), true
	}
	var args []json.RawMessage
	if json.Unmarshal([]byte(expr[start+3:end]), &args) != nil || len(args) != 3 {
		return nil, errors.New("bad UI arguments"), true
	}
	var op, token string
	var until int64
	json.Unmarshal(args[0], &op)
	json.Unmarshal(args[1], &token)
	json.Unmarshal(args[2], &until)
	ctx, _ := p["contextId"].(float64)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, op)
	world := f.worlds[session]
	ok := world != nil && world.page == page && int(ctx) == world.id
	if ok {
		switch op {
		case "acquire-background":
			if world.busy {
				ok = false
			} else {
				world.tokens[token] = until
			}
		case "acquire":
			world.tokens[token] = until
		case "check":
			expiry, exists := world.tokens[token]
			ok = exists && (expiry == 0 || time.Now().UnixMilli() < expiry)
		case "release":
			delete(world.tokens, token)
		default:
			ok = false
		}
	}
	return map[string]any{"result": map[string]any{"value": ok}}, nil, true
}
func (f *fakeControlUI) active(session string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	if w := f.worlds[session]; w != nil {
		return len(w.tokens)
	}
	return 0
}

// The captured image may finish suppression checking before navigation, while
// its layout/postback arrives after it. Document identity still owns admission.
func TestRecorderShotDoneRejectsOldDocument(t *testing.T) {
	frames := &frameRecorder{}
	h := &harness{t: t, fc: newFakeChrome(), out: &memOut{}}
	h.r = Start(Options{ID: "late", Call: h.fc, ShotEvery: -1, Out: func() (Opened, error) { return Opened{Sink: &h.out.sink, Frames: frames}, nil }})
	t.Cleanup(func() { h.r.Close() })
	p := h.page("A", "http://a/", "L1")
	h.r.Send(Live{Tab: rec.Tab{ID: p.frameID}, Session: "A"})
	h.settle()
	late := shotDone{at: time.Now(), top: "A", doc: 1, png: []byte{1}, w: 50, h: 40, scale: .5}
	h.page("A", "http://a/new", "L2")
	h.send("A", "DOM.documentUpdated", map[string]any{})
	h.settle()
	h.r.Send(late)
	h.settle()
	frames.mu.Lock()
	count := len(frames.frames)
	frames.mu.Unlock()
	if count != 0 {
		t.Fatalf("old document kept %d frames", count)
	}
	late.doc = 2
	late.w = 100
	h.r.Send(late)
	h.settle()
	frames.mu.Lock()
	defer frames.mu.Unlock()
	if len(frames.frames) != 1 || frames.frames[0].W != 100 {
		t.Fatalf("current document frames %+v", frames.frames)
	}
}
