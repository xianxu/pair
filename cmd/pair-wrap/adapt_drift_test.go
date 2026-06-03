package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// decodeAdapt parses the flight-recorder buffer into records.
func decodeAdapt(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var recs []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("bad adapt line %q: %v", line, err)
		}
		recs = append(recs, m)
	}
	return recs
}

// TestOverlayDrift_EmitsNearMiss is the regression the frozen-string unit
// tests structurally cannot catch: when a harness renames its picker so that
// NO registered marker matches, the detector goes silent — but the output is
// still prompt-shaped, so we must emit a near-miss carrying the new wording.
// That near-miss is what lets pair-doctor point at the drifted aspect.
func TestOverlayDrift_EmitsNearMiss(t *testing.T) {
	var buf bytes.Buffer
	p := &proxy{agentBasename: "codex"}
	p.adapt = adapt.New(&buf, "pair-wrap", "codex")

	// A plausible future codex picker that matches none of codexPickerMarkers.
	drifted := []byte("Do you want to apply this patch? (y/n)")
	// Sanity: it really doesn't match the frozen markers.
	if ok, _ := detectCodexOverlayText(string(drifted)); ok {
		t.Fatal("test premise broken: drifted string matched a frozen marker")
	}

	p.checkOverlayOpen(drifted, drifted)

	if p.pickerActive.Load() {
		t.Fatal("near-miss must not arm pickerActive — it is diagnostic only")
	}
	recs := decodeAdapt(t, &buf)
	if len(recs) != 1 {
		t.Fatalf("want exactly one near-miss line, got %d: %s", len(recs), buf.String())
	}
	r := recs[0]
	if r["outcome"] != "near-miss" || r["signal"] != "overlay-detect" {
		t.Errorf("wrong signal/outcome: %v", r)
	}
	if d, _ := r["detail"].(string); !strings.Contains(d, "y/n") {
		t.Errorf("detail should carry the drifted wording, got %q", d)
	}
}

// TestOverlayDrift_NearMissDeduped confirms a persistent drifted prompt
// (repainted every rerender) produces one line, not a flood.
func TestOverlayDrift_NearMissDeduped(t *testing.T) {
	var buf bytes.Buffer
	p := &proxy{agentBasename: "codex"}
	p.adapt = adapt.New(&buf, "pair-wrap", "codex")

	drifted := []byte("Do you want to apply this patch? (y/n)")
	for i := 0; i < 5; i++ {
		p.checkOverlayOpen(drifted, drifted)
	}
	if recs := decodeAdapt(t, &buf); len(recs) != 1 {
		t.Fatalf("want 1 deduped near-miss, got %d", len(recs))
	}
}

// TestOverlayKnownMarker_EmitsFiredNotNearMiss is the positive control: a
// marker we still recognize arms the overlay and logs `fired`, never a
// near-miss.
func TestOverlayKnownMarker_EmitsFiredNotNearMiss(t *testing.T) {
	var buf bytes.Buffer
	p := &proxy{agentBasename: "codex"}
	p.adapt = adapt.New(&buf, "pair-wrap", "codex")

	p.checkOverlayOpen([]byte("Press enter to continue"), []byte("Press enter to continue"))

	if !p.pickerActive.Load() {
		t.Fatal("known marker should arm pickerActive")
	}
	recs := decodeAdapt(t, &buf)
	if len(recs) != 1 || recs[0]["outcome"] != "fired" {
		t.Fatalf("want one fired line, got %s", buf.String())
	}
}

func TestPromptShape(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Do you want to apply this patch? (y/n)", true},
		{"Press enter to continue", true},
		{"Select an option:", true},
		{"esc to go back", true},
		{"Here is the refactored function you asked for.", false},
		{"The file compiles cleanly now.", false},
	}
	for _, c := range cases {
		if _, ok := promptShape(c.in); ok != c.want {
			t.Errorf("promptShape(%q) = %v, want %v", c.in, ok, c.want)
		}
	}
}
