package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestRender_ViewportSidecar drives the full render() pipeline end-to-
// end and verifies the .viewport sidecar that scrollback.lua uses to
// position the cursor on Alt+/ open.
//
// Sets up a 20x5 emulator and pushes 10 short lines. The first 9 lines
// each end with CRLF; the last has no terminator so the cursor parks on
// row 4 of the visible buffer without triggering an extra scroll. The
// resulting state: lines 1..5 spill into scrollback history (5 rows),
// lines 6..10 fill the visible buffer (5 rows). Total .ansi length = 10,
// viewport_top = 6 (1-indexed first line of the visible buffer).
func TestRender_ViewportSidecar(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "in.raw")
	evPath := filepath.Join(dir, "in.events.jsonl")
	outPath := filepath.Join(dir, "out.ansi")
	vpPath := filepath.Join(dir, "out.viewport")

	var raw strings.Builder
	for i := 1; i <= 10; i++ {
		if i < 10 {
			fmt.Fprintf(&raw, "line %02d\r\n", i)
		} else {
			fmt.Fprintf(&raw, "line %02d", i) // no trailing CRLF → no extra scroll
		}
	}
	if err := os.WriteFile(rawPath, []byte(raw.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	// One initial-size resize at offset 0 — pair-wrap always emits one.
	events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
	if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := render(rawPath, evPath, outPath, false, historyRows); err != nil {
		t.Fatalf("render: %v", err)
	}

	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out.ansi: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 10 {
		t.Errorf(".ansi: got %d lines, want 10\n--- content ---\n%s",
			len(lines), string(body))
	}

	vpBytes, err := os.ReadFile(vpPath)
	if err != nil {
		t.Fatalf("read .viewport: %v", err)
	}
	vp, err := strconv.Atoi(strings.TrimSpace(string(vpBytes)))
	if err != nil {
		t.Fatalf("parse .viewport %q: %v", string(vpBytes), err)
	}
	// 5-row visible buffer in a 10-line output → viewport_top = 6.
	if vp != 6 {
		t.Errorf("viewport_top: got %d, want 6 (lines 6-10 = visible buffer)", vp)
	}
}

// TestRender_Plain drives render() in plain mode (uncapped) over styled input
// and verifies the output carries the visible text with NO SGR escapes — the
// substrate a continuation distills.
func TestRender_Plain(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "in.raw")
	evPath := filepath.Join(dir, "in.events.jsonl")
	outPath := filepath.Join(dir, "out.txt")

	raw := "\x1b[31mred\x1b[0m text" // a colored word + plain text, no trailing newline
	if err := os.WriteFile(rawPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	events := `{"type":"resize","offset":0,"cols":20,"rows":5}` + "\n"
	if err := os.WriteFile(evPath, []byte(events), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := render(rawPath, evPath, outPath, true, -1); err != nil { // plain, uncapped
		t.Fatalf("render: %v", err)
	}
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if strings.Contains(string(body), "\x1b") {
		t.Fatalf("plain render contains SGR escape: %q", string(body))
	}
	if !strings.Contains(string(body), "red text") {
		t.Fatalf("plain render missing visible text; got %q", string(body))
	}
	// plain render must NOT litter a .viewport sidecar
	if _, err := os.Stat(outPath + ".viewport"); err == nil {
		t.Fatalf("plain render wrote a stray .viewport sidecar")
	}
}

func TestResolveMax(t *testing.T) {
	for _, c := range []struct{ in, want int }{
		{-1, math.MaxInt32}, {0, math.MaxInt32}, {5, 5}, {2000, 2000},
	} {
		if got := resolveMax(c.in); got != c.want {
			t.Fatalf("resolveMax(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestRender_MaxLinesCaps pushes many lines through a short terminal so most
// spill to scrollback, then asserts a small --max-lines retains strictly fewer
// total rows than uncapped — exercising the cap's *differentiating* behavior.
func TestRender_MaxLinesCaps(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "in.raw")
	evPath := filepath.Join(dir, "in.events.jsonl")
	var raw strings.Builder
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&raw, "line %02d\r\n", i)
	}
	if err := os.WriteFile(rawPath, []byte(raw.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evPath, []byte(`{"type":"resize","offset":0,"cols":20,"rows":5}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	countLines := func(maxLines int) int {
		out := filepath.Join(dir, fmt.Sprintf("out-%d.txt", maxLines))
		if err := render(rawPath, evPath, out, true, maxLines); err != nil {
			t.Fatalf("render(maxLines=%d): %v", maxLines, err)
		}
		b, _ := os.ReadFile(out)
		return len(strings.Split(strings.TrimRight(string(b), "\n"), "\n"))
	}
	capped, uncapped := countLines(2), countLines(-1)
	if capped >= uncapped {
		t.Fatalf("expected capped (%d) < uncapped (%d)", capped, uncapped)
	}
}

// TestRender_ViewportEmptyHistory covers the edge case where the agent
// hasn't produced enough output to spill anything into history yet:
// viewport_top should still be a valid 1-indexed line number (= 1), not
// 0 or negative.
func TestRender_ViewportEmptyHistory(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "in.raw")
	evPath := filepath.Join(dir, "in.events.jsonl")
	outPath := filepath.Join(dir, "out.ansi")
	vpPath := filepath.Join(dir, "out.viewport")

	// Just 2 lines in a 5-row terminal — no history overflow.
	if err := os.WriteFile(rawPath, []byte("only\r\ntwo\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evPath,
		[]byte(`{"type":"resize","offset":0,"cols":20,"rows":5}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := render(rawPath, evPath, outPath, false, historyRows); err != nil {
		t.Fatalf("render: %v", err)
	}

	vpBytes, err := os.ReadFile(vpPath)
	if err != nil {
		t.Fatalf("read .viewport: %v", err)
	}
	vp, err := strconv.Atoi(strings.TrimSpace(string(vpBytes)))
	if err != nil {
		t.Fatalf("parse .viewport: %v", err)
	}
	// No history rows → viewport top of the visible buffer is line 1.
	if vp != 1 {
		t.Errorf("viewport_top: got %d, want 1 (no scrollback history)", vp)
	}
}
