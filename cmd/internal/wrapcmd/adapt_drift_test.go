package wrapcmd

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

// TestEmitPlainCR_LogsFiredAndBypass covers aspect 1's telemetry: a normal
// Enter logs `fired`, an Enter consumed while an overlay is active logs
// `bypass`. (The byte-translation behavior itself is covered in
// picker_overlay_test.go; this pins the signal.)
func TestEmitPlainCR_LogsFiredAndBypass(t *testing.T) {
	var buf bytes.Buffer
	p := claudeProxy()
	p.adapt = adapt.New(&buf, "pair-wrap", "claude")

	p.emitPlainCR(nil) // no overlay → fired
	p.pickerActive.Store(true)
	p.emitPlainCR(nil) // overlay active → bypass

	recs := decodeAdapt(t, &buf)
	if len(recs) != 2 {
		t.Fatalf("want 2 lines, got %d: %s", len(recs), buf.String())
	}
	if recs[0]["outcome"] != "fired" || recs[1]["outcome"] != "bypass" {
		t.Errorf("outcomes = %v, %v (want fired, bypass)", recs[0]["outcome"], recs[1]["outcome"])
	}
	for _, r := range recs {
		if r["signal"] != "return-remap" || r["aspect"] != float64(1) {
			t.Errorf("bad signal/aspect: %v", r)
		}
	}
}

// TestOutputFilter_LogsFiredDedupedPerMarker covers aspect 5: stripping a
// codex sync-output marker logs `fired` once per distinct marker (deduped —
// the markers repeat many times per render).
func TestOutputFilter_LogsFiredDedupedPerMarker(t *testing.T) {
	var buf bytes.Buffer
	p := &proxy{agentBasename: "codex"}
	p.adapt = adapt.New(&buf, "pair-wrap", "codex")

	// Same marker twice + a different one, interleaved with normal bytes.
	p.stripCodexOutputMarkers(
		[]byte("a\x1b[?2026hb\x1b[?2026hc\x1b[?1004hd"),
		codexSyncOutputMarkers,
	)

	recs := decodeAdapt(t, &buf)
	if len(recs) != 2 {
		t.Fatalf("want 2 deduped fired lines (one per distinct marker), got %d: %s", len(recs), buf.String())
	}
	for _, r := range recs {
		if r["signal"] != "output-filter" || r["outcome"] != "fired" || r["aspect"] != float64(5) {
			t.Errorf("bad record: %v", r)
		}
	}
}

// TestPromptShapeMultibyteNoPanic is the C1 regression: 'Ⱥ' (U+023A, 2 bytes)
// lowercases to 'ⱥ' (U+2C65, 3 bytes), so the old strings.ToLower path
// produced a match offset past len(visible) and panicked snippetLine. The
// ASCII-fold path keeps offsets aligned; the snippet must come back correct.
func TestPromptShapeMultibyteNoPanic(t *testing.T) {
	in := "Ⱥ heads up\nselect an option:"
	snippet, ok := promptShape(in)
	if !ok {
		t.Fatal("expected a prompt-shape match after the multibyte rune")
	}
	if snippet != "select an option:" {
		t.Errorf("snippet = %q, want %q", snippet, "select an option:")
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
