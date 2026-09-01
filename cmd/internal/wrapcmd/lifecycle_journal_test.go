package wrapcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

func lifecycleJSONLine(t *testing.T, launch uint64, offset int64) []byte {
	t.Helper()
	record := sessionwatch.LifecycleRecord{Version: 1, Agent: "codex", LaunchOrdinal: launch,
		ArtifactGeneration: "dev:1", Source: "transcript", Outcome: "task_complete",
		TranscriptPath: "/rollout.jsonl", TranscriptOffset: offset, EventTimestamp: time.Unix(offset, 0).UTC()}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func TestLifecycleJournalTailerReadsOnlyCurrentLaunchAfterPriorEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
	if err := os.WriteFile(path, lifecycleJSONLine(t, 6, 1), 0o600); err != nil {
		t.Fatal(err)
	}
	tailer, err := OpenLifecycleJournalTailer(path, 7)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(lifecycleJSONLine(t, 6, 2))
	_, _ = f.Write(lifecycleJSONLine(t, 7, 3))
	_ = f.Close()
	records, err := tailer.Advance()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].LaunchOrdinal != 7 || records[0].TranscriptOffset != 3 {
		t.Fatalf("records = %#v", records)
	}
}

func TestLifecycleJournalTailerWaitsForFirstCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
	tailer, err := OpenLifecycleJournalTailer(path, 7)
	if err != nil {
		t.Fatal(err)
	}
	if records, err := tailer.Advance(); err != nil || len(records) != 0 {
		t.Fatalf("before creation: records=%+v err=%v", records, err)
	}
	if err := os.WriteFile(path, lifecycleJSONLine(t, 7, 3), 0o600); err != nil {
		t.Fatal(err)
	}
	if records, err := tailer.Advance(); err != nil || len(records) != 1 {
		t.Fatalf("after creation: records=%+v err=%v", records, err)
	}
}

func TestLifecycleJournalTailerWaitsForPartialLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tailer, err := OpenLifecycleJournalTailer(path, 7)
	if err != nil {
		t.Fatal(err)
	}
	line := lifecycleJSONLine(t, 7, 3)
	if err := os.WriteFile(path, line[:len(line)-1], 0o600); err != nil {
		t.Fatal(err)
	}
	if records, err := tailer.Advance(); err != nil || len(records) != 0 {
		t.Fatalf("partial: %#v, %v", records, err)
	}
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.Write([]byte{'\n'})
	_ = f.Close()
	if records, err := tailer.Advance(); err != nil || len(records) != 1 {
		t.Fatalf("committed: %#v, %v", records, err)
	}
}

func TestLifecycleJournalTailerFailsClosedOnOversizeTruncateAndReplace(t *testing.T) {
	t.Run("oversize", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal")
		_ = os.WriteFile(path, nil, 0o600)
		tailer, _ := OpenLifecycleJournalTailer(path, 1)
		_ = os.WriteFile(path, []byte(strings.Repeat("x", lifecycleJournalMaxRecord+1)+"\n"), 0o600)
		if _, err := tailer.Advance(); err == nil {
			t.Fatal("oversized record accepted")
		}
	})
	t.Run("truncate", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "journal")
		_ = os.WriteFile(path, lifecycleJSONLine(t, 1, 1), 0o600)
		tailer, _ := OpenLifecycleJournalTailer(path, 1)
		_ = os.Truncate(path, 0)
		if _, err := tailer.Advance(); err == nil {
			t.Fatal("truncation accepted")
		}
	})
	t.Run("replace", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "journal")
		_ = os.WriteFile(path, nil, 0o600)
		tailer, _ := OpenLifecycleJournalTailer(path, 1)
		replacement := filepath.Join(dir, "replacement")
		_ = os.WriteFile(replacement, lifecycleJSONLine(t, 1, 2), 0o600)
		_ = os.Rename(replacement, path)
		if _, err := tailer.Advance(); err == nil {
			t.Fatal("replacement accepted")
		}
	})
}

func TestLifecycleJournalRecordsDriveCanonicalNotification(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "lifecycle.jsonl")
	outer := filepath.Join(dir, "outer")
	sidecar := filepath.Join(dir, "outer-path")
	if err := os.WriteFile(journal, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outer, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, []byte(outer+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tailer, err := OpenLifecycleJournalTailer(journal, 7)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(100, 0).UTC()
	records := []sessionwatch.LifecycleRecord{
		{Version: 1, Agent: "codex", LaunchOrdinal: 7, ArtifactGeneration: "gen:1", Source: "transcript", Outcome: "started", TurnID: "turn-1", TranscriptPath: "rollout.jsonl", TranscriptOffset: 10, EventTimestamp: stamp},
		{Version: 1, Agent: "codex", LaunchOrdinal: 7, ArtifactGeneration: "gen:1", Source: "transcript", Outcome: "completed", TurnID: "turn-1", TranscriptPath: "rollout.jsonl", TranscriptOffset: 20, EventTimestamp: stamp.Add(time.Second)},
	}
	f, err := os.OpenFile(journal, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range records {
		raw, _ := json.Marshal(record)
		_, _ = f.Write(append(raw, '\n'))
	}
	_ = f.Close()
	observed, err := tailer.Advance()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(200, 0)
	p := &proxy{outerTTYFile: sidecar, lastSlug: now, now: func() time.Time { return now }}
	for _, record := range observed {
		p.processLifecycleRecord(record)
	}
	written, err := os.ReadFile(outer)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != "\x1b]777;notify;pair;agent finished working\x07" {
		t.Fatalf("outer=%q", written)
	}
}
