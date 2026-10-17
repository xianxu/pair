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

func TestResourceHistoryRetainsTheNamedOrphan(t *testing.T) {
	rss := uint64(22 << 30)
	orphan := OrphanNvim{
		Key:      ResourceProcessKey{PID: 4242, Identity: "1760000000.5"},
		Age:      13 * 24 * time.Hour,
		Cwd:      "/fleet/worktree/pair-slot1/pair/cmd",
		Root:     "/fleet/worktree/pair-slot1/pair",
		RSSBytes: &rss,
		Reason:   "unattributed embedded nvim",
	}
	// The sweep suppresses a named orphan from Unattributed, so the frame is the
	// only durable record that the fleet saw it (pair#583).
	h, err := ResourceHistory{Version: 1}.Append(ResourceView{SampledAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), OrphanNvim: []OrphanNvim{orphan}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateResourceHistory(raw); err != nil {
		t.Fatal(err)
	}
	var back ResourceHistory
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Frames) != 1 || len(back.Frames[0].OrphanNvim) != 1 {
		t.Fatalf("lost the sweep: %s", raw)
	}
	got := back.Frames[0].OrphanNvim[0]
	if got.Key != orphan.Key || got.Age != orphan.Age || got.Cwd != orphan.Cwd || got.Root != orphan.Root || got.Reason != orphan.Reason {
		t.Fatalf("changed %+v", got)
	}
	if got.RSSBytes == nil || *got.RSSBytes != rss {
		t.Fatalf("lost rss %+v", got.RSSBytes)
	}
}

func TestResourceHistoryBoundsTheOrphanSweep(t *testing.T) {
	over := make([]OrphanNvim, ResourceMaxProcesses+1)
	for i := range over {
		over[i] = OrphanNvim{Key: ResourceProcessKey{PID: i + 1, Identity: "1.2"}, Reason: "unattributed embedded nvim"}
	}
	h, err := ResourceHistory{Version: 1}.Append(ResourceView{SampledAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), OrphanNvim: over})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateResourceHistory(raw); err == nil {
		t.Fatal("accepted an unbounded sweep")
	}
}
