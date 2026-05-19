package main

import (
	"fmt"
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

	if err := render(rawPath, evPath, outPath); err != nil {
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

	if err := render(rawPath, evPath, outPath); err != nil {
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
