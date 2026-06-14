package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildRender builds the sibling pair-scrollback-render binary so the e2e test
// can exercise the real render→cleaned→distill seam (#59).
func buildRender(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-scrollback-render")
	out, err := exec.Command("go", "build", "-o", bin, "../pair-scrollback-render").CombinedOutput()
	if err != nil {
		t.Fatalf("build render: %v\n%s", err, out)
	}
	return bin
}

// End-to-end (plan-quality finding 2): the render's ⟦pair:ts DATE⟧ markers must
// survive through the real render→cleaned→distill pipe, not just a hand-injected
// cleaned fixture. Build raw + events with two time events, render
// --with-timestamps, then distill — assert the log carries both day headers. This
// guards the render↔distiller marker contract (tsMarkerLine ↔ tsMarkerRe).
func TestEndToEndMarkerSurvival(t *testing.T) {
	render := buildRender(t)
	distill := buildBinary(t)
	fakeClaude(t, "- worked\n\n- more\n")

	dir := t.TempDir()
	rawPath := filepath.Join(dir, "s.raw")
	evPath := filepath.Join(dir, "s.events.jsonl")
	cleanedPath := filepath.Join(dir, "s.cleaned")
	logPath := filepath.Join(dir, "changelog.md")
	anchorPath := filepath.Join(dir, "changelog.anchor")

	// 40 lines so scrollback forms (default 80x24); time events for two days at
	// offsets that land in different scrollback regions.
	var raw strings.Builder
	var off10, off30 int
	for i := 0; i < 40; i++ {
		if i == 10 {
			off10 = raw.Len()
		}
		if i == 30 {
			off30 = raw.Len()
		}
		fmt.Fprintf(&raw, "line %02d\r\n", i)
	}
	mustWrite(t, rawPath, raw.String())
	events := `{"type":"resize","offset":0,"cols":80,"rows":24}` + "\n" +
		fmt.Sprintf(`{"type":"time","offset":%d,"ts":"2026-06-13T09:00:00Z"}`, off10) + "\n" +
		fmt.Sprintf(`{"type":"time","offset":%d,"ts":"2026-06-14T09:00:00Z"}`, off30) + "\n"
	mustWrite(t, evPath, events)

	// Real render step (as bin/pair-changelog-open invokes it).
	if out, err := exec.Command(render, "--plain", "--max-lines", "0", "--with-timestamps",
		rawPath, evPath, cleanedPath).CombinedOutput(); err != nil {
		t.Fatalf("render: %v\n%s", err, out)
	}
	cleaned := readOr(cleanedPath)
	if !strings.Contains(cleaned, "⟦pair:ts 2026-06-13⟧") || !strings.Contains(cleaned, "⟦pair:ts 2026-06-14⟧") {
		t.Fatalf("render did not emit both day markers:\n%s", cleaned)
	}

	// Real distill step over the rendered cleaned.
	if out, err := exec.Command(distill, "--cleaned", cleanedPath, "--log", logPath,
		"--anchor", anchorPath, "--agent", "claude").CombinedOutput(); err != nil {
		t.Fatalf("distill: %v\n%s", err, out)
	}
	log := readOr(logPath)
	i13 := strings.Index(log, "## 2026-06-13")
	i14 := strings.Index(log, "## 2026-06-14")
	if i13 < 0 || i14 < 0 || i13 >= i14 {
		t.Fatalf("markers did not survive into dated headers (13=%d 14=%d):\n%s", i13, i14, log)
	}
	if strings.Contains(log, "⟦pair:ts") {
		t.Fatalf("ts marker leaked into the change log:\n%s", log)
	}
}
