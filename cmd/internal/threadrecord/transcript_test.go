package threadrecord

import (
	"testing"
	"time"
)

func TestTranscriptBindingValidate(t *testing.T) {
	const id = "0b5c2a1e-9f1d-4c4e-8d5b-3e2f1a0b9c8d"
	at := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	ok := TranscriptBinding{Agent: "claude", NativeID: id, StorageRoot: "claude-projects",
		RelativePath: "-Users-x-workspace-pair/" + id + ".jsonl", BoundAt: at, Source: TranscriptSourceLedger}
	if err := ok.Validate(); err != nil {
		t.Fatalf("established binding refused: %v", err)
	}
	idOnly := TranscriptBinding{Agent: "claude", NativeID: id, BoundAt: at, Source: TranscriptSourceLegacyLedger}
	if err := idOnly.Validate(); err != nil {
		t.Fatalf("binding that knows only its ID refused: %v", err)
	}
	if idOnly.Established() || !ok.Established() {
		t.Fatal("Established must mean a located file")
	}

	bad := map[string]func(*TranscriptBinding){
		"relative escape":    func(b *TranscriptBinding) { b.RelativePath = "../" + id + ".jsonl" },
		"absolute path":      func(b *TranscriptBinding) { b.RelativePath = "/tmp/" + id + ".jsonl" },
		"unclean path":       func(b *TranscriptBinding) { b.RelativePath = "a//" + id + ".jsonl" },
		"foreign root":       func(b *TranscriptBinding) { b.StorageRoot = "codex-sessions" },
		"mismatched id":      func(b *TranscriptBinding) { b.RelativePath = "p/other.jsonl" },
		"id only in dir":     func(b *TranscriptBinding) { b.RelativePath = id + "/notes.jsonl" },
		"root without path":  func(b *TranscriptBinding) { b.RelativePath = "" },
		"path without root":  func(b *TranscriptBinding) { b.StorageRoot = "" },
		"empty id":           func(b *TranscriptBinding) { b.NativeID = "" },
		"id with separator":  func(b *TranscriptBinding) { b.NativeID = "a/b" },
		"empty agent":        func(b *TranscriptBinding) { b.Agent = "" },
		"other agent's root": func(b *TranscriptBinding) { b.Agent = "codex" },
		"subagent file":      func(b *TranscriptBinding) { b.RelativePath = "p/" + id + "/subagents/agent-a.jsonl" },
		"id as a prefix":     func(b *TranscriptBinding) { b.RelativePath = "p/" + id + "-extra.jsonl" },
		"unknown source":     func(b *TranscriptBinding) { b.Source = "guess" },
		"zero bound time":    func(b *TranscriptBinding) { b.BoundAt = time.Time{} },
	}
	for name, mutate := range bad {
		b := ok
		mutate(&b)
		if err := b.Validate(); err == nil {
			t.Errorf("%s: accepted %+v", name, b)
		}
	}
}
