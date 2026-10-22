package recorder

import (
	"context"
	"encoding/json"
	"errors"
	rec "github.com/xianxu/pair/cmd/internal/webcast/recording"
	"strings"
	"testing"
	"time"
)

func TestSketchFrameExclusionDescendantsAndRetirement(t *testing.T) {
	h := newHarness(t)
	h.page("top", "https://example.test", "one")
	send := func(method, raw string) {
		h.r.Send(Event{Session: "top", Method: method, Params: json.RawMessage(raw)})
	}
	send("Page.frameAttached", `{"frameId":"private","parentFrameId":"T-top"}`)
	if err := h.r.ExcludeSketchFrame(context.Background(), "private", "top"); err != nil {
		t.Fatal(err)
	}
	send("Page.frameAttached", `{"frameId":"child","parentFrameId":"private"}`)
	send("Page.frameNavigated", `{"frame":{"id":"grandchild","parentId":"child"}}`)
	send("Page.frameAttached", `{"frameId":"sibling","parentFrameId":"T-top"}`)
	for id, frame := range map[int]string{1: "T-top", 2: "private", 3: "child", 4: "grandchild", 5: "sibling"} {
		raw, _ := json.Marshal(map[string]any{"context": map[string]any{"id": id, "name": worldName, "auxData": map[string]any{"frameId": frame}}})
		h.r.Send(Event{Session: "top", Method: "Runtime.executionContextCreated", Params: raw})
	}
	if err := h.r.acknowledged(context.Background(), func() error {
		h.r.top = "top"
		got := h.r.castContexts()
		if len(got) != 2 {
			t.Errorf("normal driving contexts=%v; want top and sibling", got)
		}
		h.r.live = true
		h.r.event(time.Now(), Event{Session: "top", Method: "Network.responseReceived", Params: json.RawMessage(`{"requestId":"secret","frameId":"grandchild","type":"Image","response":{"url":"https://secret.test/private","mimeType":"image/png"}}`)})
		h.r.event(time.Now(), Event{Session: "top", Method: "Network.responseReceived", Params: json.RawMessage(`{"requestId":"public","frameId":"sibling","type":"Image","response":{"url":"https://example.test/public","mimeType":"image/png"}}`)})
		if len(h.r.requests) != 1 {
			t.Errorf("network requests=%v", h.r.requests)
		}
		h.r.live = false
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ok, err := h.r.AdmitRecordingFrame(context.Background(), "grandchild", "oop", "top")
	if err != nil || ok {
		t.Fatalf("private OOP admission=%v %v", ok, err)
	}
	send("Page.frameDetached", `{"frameId":"private","reason":"remove"}`)
	if err := h.r.acknowledged(context.Background(), func() error {
		if len(h.r.sketchFrames) != 0 {
			t.Errorf("retained roots=%v", h.r.sketchFrames)
		}
		if _, ok := h.r.frameParents["child"]; ok {
			t.Error("retained child")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSketchAssetURLExclusionExactAndOwned(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	if err := h.r.ExcludeSketchFrame(ctx, "private", "top"); err != nil {
		t.Fatal(err)
	}
	if err := h.r.ExcludeSketchAssetURL(ctx, "private", "blob:null/private-font"); err != nil {
		t.Fatal(err)
	}
	if err := h.r.acknowledged(ctx, func() error {
		for _, src := range []string{`url("blob:null/private-font")`, `url(blob:null/private-font) format('woff2')`, `blob:null/private-font`} {
			if !h.r.excludedAssetURL(src) {
				t.Errorf("private URL not recognized: %s", src)
			}
		}
		for _, src := range []string{`url(blob:null/private-font-more)`, `url(https://public.test/font)`, `local("blob:null/private-font")`} {
			if h.r.excludedAssetURL(src) {
				t.Errorf("ordinary URL excluded: %s", src)
			}
		}
		h.r.forgetFrame("private")
		if len(h.r.sketchAssetURLs) != 0 {
			t.Error("private URL registry retained after frame retirement")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSketchFrameExclusionRecordedDOMAndFonts(t *testing.T) {
	h := newHarness(t)
	p := h.page("top", "https://example.test", "one")
	private := p.el("iframe")
	private.frame = "private"
	private.content = p.node(9, "", "")
	private.content.kids = []*fnode{p.txt("SECRET SOURCE")}
	public := p.el("p")
	public.kids = []*fnode{p.txt("public sibling")}
	p.body().kids = append(p.body().kids, private, public)
	ctx := context.Background()
	if err := h.r.ExcludeSketchFrame(ctx, "private", "top"); err != nil {
		t.Fatal(err)
	}
	if err := h.r.ExcludeSketchAssetURL(ctx, "private", "blob:null/secret"); err != nil {
		t.Fatal(err)
	}
	h.r.Send(Live{Tab: rec.Tab{ID: "T-top"}, Session: "top"})
	for _, src := range []string{`url("blob:null/secret")`, `url("https://example.test/public.woff2")`} {
		h.r.Send(Event{Session: "top", Method: "CSS.fontsUpdated", Params: raw(map[string]any{"font": map[string]any{"fontFamily": "test", "src": src}})})
	}
	h.settle()
	var all strings.Builder
	for _, r := range h.out.sink.Records() {
		b, err := rec.Encode(r)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(b)
	}
	if strings.Contains(all.String(), "SECRET SOURCE") || strings.Contains(all.String(), "blob:null/secret") {
		t.Fatalf("private data recorded: %s", all.String())
	}
	if !strings.Contains(all.String(), "public sibling") || !strings.Contains(all.String(), "public.woff2") {
		t.Fatalf("ordinary sibling/font missing: %s", all.String())
	}
}

func TestSketchExclusionBlocksObserverStylesAndLateOOPEvents(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	h.page("top", "https://example.test", "one")
	if err := h.r.ExcludeSketchFrame(ctx, "private", "top"); err != nil {
		t.Fatal(err)
	}
	if err := h.r.ExcludeSketchAssetURL(ctx, "private", "blob:null/secret"); err != nil {
		t.Fatal(err)
	}
	h.r.Send(FrameAttached{Session: "private-session", Parent: "top", FrameID: "private"})
	h.r.Send(FrameAttached{Session: "nested-session", Parent: "private-session", FrameID: "nested-not-yet-in-frame-tree"})
	if err := h.r.acknowledged(ctx, func() error {
		h.r.top = "top"
		h.r.live = true
		h.r.contexts["top"] = map[int64]string{1: "private", 2: "T-top"}
		h.fc.mu.Lock()
		before := len(h.fc.calls)
		h.fc.mu.Unlock()
		h.r.observeItems(0, "top", 1, []Item{{K: "hello"}}, "")
		h.r.event(time.Now(), Event{Session: "top", Method: "CSS.styleSheetAdded", Params: json.RawMessage(`{"header":{"styleSheetId":"secret","frameId":"private","isConstructed":true}}`)})
		h.r.event(time.Now(), Event{Session: "top", Method: "Network.responseReceived", Params: json.RawMessage(`{"requestId":"blob","type":"Image","response":{"url":"blob:null/secret"}}`)})
		h.r.event(time.Now(), Event{Session: "top", Method: "CSS.styleSheetAdded", Params: json.RawMessage(`{"header":{"styleSheetId":"blob","sourceURL":"blob:null/secret","isConstructed":true}}`)})
		h.fc.mu.Lock()
		after := len(h.fc.calls)
		h.fc.mu.Unlock()
		if before != after {
			t.Error("excluded observer/style triggered CDP read")
		}
		if !h.r.excludedSession("nested-session") {
			t.Error("OOP child not excluded by parent session")
		}
		h.r.forgetFrame("private")
		h.r.event(time.Now(), Event{Session: "private-session", Method: "Network.responseReceived", Params: json.RawMessage(`{"requestId":"late","type":"Image","response":{"url":"https://secret.test/late"}}`)})
		if len(h.r.requests) != 0 {
			t.Errorf("late private requests=%v", h.r.requests)
		}
		h.r.live = false
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h.r.Send(Detached{Session: "private-session"})
	h.r.Send(Detached{Session: "nested-session"})
	if err := h.r.acknowledged(ctx, func() error {
		if len(h.r.sketchSessions) != 0 {
			t.Errorf("retained detached sessions: %v", h.r.sketchSessions)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSketchExclusionCancellationAndRepeatedScopeRetirement(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.r.ExcludeSketchFrame(ctx, "canceled", "top"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled registration=%v", err)
	}
	ctx = context.Background()
	h.page("top", "https://example.test", "one")
	for n := 0; n < 20; n++ {
		if err := h.r.ExcludeSketchFrame(ctx, "private", "top"); err != nil {
			t.Fatal(err)
		}
		if err := h.r.ExcludeSketchAssetURL(ctx, "private", "blob:null/asset"); err != nil {
			t.Fatal(err)
		}
		h.r.Send(Event{Session: "top", Method: "Page.frameDetached", Params: json.RawMessage(`{"frameId":"private","reason":"remove"}`)})
	}
	if err := h.r.acknowledged(ctx, func() error {
		if len(h.r.sketchFrames)+len(h.r.sketchAssetURLs)+len(h.r.sketchSessions)+len(h.r.frameParents) != 0 {
			t.Error("private scope history grew after disposal")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := h.r.ExcludeSketchFrame(ctx, "private", "top"); err != nil {
		t.Fatal(err)
	}
	if ok, err := h.r.AdmitRecordingFrame(ctx, "unknown", "oop", "top"); ok || err == nil {
		t.Fatalf("unknown ancestry admitted during private rendering: %v %v", ok, err)
	}
}
