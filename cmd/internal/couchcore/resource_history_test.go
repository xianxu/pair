package couchcore

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestResourceHistoryBounds(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	h := ResourceHistory{Version: 1}
	for i := 0; i < 1450; i++ {
		var err error
		h, err = h.Append(ResourceView{SampledAt: base.Add(time.Duration(i) * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(h.Frames) != 1440 {
		t.Fatalf("frames %d", len(h.Frames))
	}
	before := h
	h, err := h.Append(ResourceView{SampledAt: base.Add(72 * time.Hour)})
	if err != nil || len(h.Frames) != 1 || len(before.Frames) != 1440 {
		t.Fatalf("prune/immutability %d %v", len(h.Frames), err)
	}
	for i := 0; i < 5; i++ {
		h, err = h.Append(ResourceView{SampledAt: base.Add(72*time.Hour + time.Duration(i+1)*time.Minute), Errors: []string{strings.Repeat("x", 1<<20)}})
		if err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := json.Marshal(h)
	if len(raw) > ResourceHistoryLimit || len(h.Frames) >= 6 {
		t.Fatalf("byte cap %d %d", len(raw), len(h.Frames))
	}
	if _, err = h.Append(ResourceView{SampledAt: base}); err == nil {
		t.Fatal("accepted regressing clock")
	}
}
