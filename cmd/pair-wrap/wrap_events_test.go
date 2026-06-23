package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTraceWrapWritesStructuredJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wrap-events-test.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(err)
	}
	p := &proxy{agentBasename: "codex", wrapEventsFD: f}

	p.traceWrap("test-event", map[string]any{
		"raw_len":       len("secret output"),
		"raw_sha256_12": shortSHA256([]byte("secret output")),
	})
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(body[:len(body)-1], &rec); err != nil {
		t.Fatalf("invalid jsonl %q: %v", body, err)
	}
	if rec["component"] != "pair-wrap" || rec["agent"] != "codex" || rec["label"] != "test-event" {
		t.Fatalf("unexpected record: %#v", rec)
	}
	if _, ok := rec["ts"].(string); !ok {
		t.Fatalf("missing timestamp: %#v", rec)
	}
	if rec["raw_len"] != float64(13) {
		t.Fatalf("raw_len = %#v, want 13", rec["raw_len"])
	}
	if rec["raw_sha256_12"] == "" {
		t.Fatalf("missing hash: %#v", rec)
	}
	if string(body) == "secret output" || rec["raw"] != nil {
		t.Fatalf("trace leaked raw body: %s", body)
	}
}

func TestHandleChunkTracesMasterStdoutAndScrollback(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "wrap-events-test.jsonl")
	traceFD, err := os.OpenFile(tracePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(err)
	}
	scrollPath := filepath.Join(dir, "scrollback.raw")
	scrollFD, err := os.OpenFile(scrollPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(err)
	}
	p := &proxy{
		agentBasename: "codex",
		wrapEventsFD:  traceFD,
		scrollbackFD:  scrollFD,
		filterSeen:    make(map[string]bool),
		now:           func() time.Time { return time.Unix(0, 0).UTC() },
	}
	rolling := []byte{}

	p.handleChunk([]byte("a\x1b[?2026hb"), &rolling)
	_ = traceFD.Close()
	_ = scrollFD.Close()

	body, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"label":"master-chunk"`,
		`"label":"stdout-write"`,
		`"label":"scrollback-write"`,
		`"raw_len":10`,
		`"stdout_len":2`,
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("trace missing %s in:\n%s", want, body)
		}
	}
	if strings.Contains(string(body), "a\x1b[?2026hb") {
		t.Fatalf("trace leaked raw terminal bytes: %q", body)
	}
}
