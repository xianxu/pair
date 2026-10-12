package webcast

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestZZDebugResources(t *testing.T) {
	if os.Getenv("ZZ") == "" {
		t.Skip()
	}
	c := liveChrome(t)
	h := liveHarness{c}
	s := h.session(t, "")
	c.call(s, "Page.navigate", map[string]any{"url": "http://localhost:4321/2026/10/ariadne-1-building-blocks-of-ai-coding"}, nil)
	time.Sleep(3 * time.Second)
	var res json.RawMessage
	c.call(s, "Page.getResourceTree", nil, &res)
	var tree struct {
		FrameTree struct {
			Frame struct{ ID string } `json:"frame"`
			Resources []struct{ URL, Type string } `json:"resources"`
		} `json:"frameTree"`
	}
	json.Unmarshal(res, &tree)
	for _, r := range tree.FrameTree.Resources {
		if r.Type == "Image" || r.Type == "Font" || r.Type == "Stylesheet" {
			var body struct{ Content string `json:"content"` }
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Call(ctx, s, "Page.getResourceContent", map[string]any{"frameId": tree.FrameTree.Frame.ID, "url": r.URL}, &body)
			cancel()
			t.Logf("%s %s len=%d err=%v", r.Type, r.URL[:min(100, len(r.URL))], len(body.Content), err)
		}
	}
}
