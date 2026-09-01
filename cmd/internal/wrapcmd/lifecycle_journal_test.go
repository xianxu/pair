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
		ArtifactGeneration: "dev:1", Source: "transcript", Outcome: "completed", TurnID: "turn-1",
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

func TestLifecycleJournalTailerRejectsMalformedAndUnauthorizedRecords(t *testing.T) {
	valid := sessionwatch.LifecycleRecord{
		Version: 1, Agent: "codex", LaunchOrdinal: 7, ArtifactGeneration: "gen:1",
		Source: "transcript", Outcome: "completed", TurnID: "turn-1",
		TranscriptPath: "/rollout.jsonl", TranscriptOffset: 3,
		EventTimestamp: time.Unix(3, 0).UTC(),
	}
	tests := map[string]func(*sessionwatch.LifecycleRecord){
		"wrong version":    func(r *sessionwatch.LifecycleRecord) { r.Version = 2 },
		"wrong agent":      func(r *sessionwatch.LifecycleRecord) { r.Agent = "claude" },
		"missing launch":   func(r *sessionwatch.LifecycleRecord) { r.LaunchOrdinal = 0 },
		"wrong source":     func(r *sessionwatch.LifecycleRecord) { r.Source = "native" },
		"unknown outcome":  func(r *sessionwatch.LifecycleRecord) { r.Outcome = "finished" },
		"missing turn id":  func(r *sessionwatch.LifecycleRecord) { r.TurnID = "" },
		"missing artifact": func(r *sessionwatch.LifecycleRecord) { r.ArtifactGeneration = "" },
		"missing path":     func(r *sessionwatch.LifecycleRecord) { r.TranscriptPath = "" },
		"negative offset":  func(r *sessionwatch.LifecycleRecord) { r.TranscriptOffset = -1 },
		"zero timestamp":   func(r *sessionwatch.LifecycleRecord) { r.EventTimestamp = time.Time{} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			tailer, err := OpenLifecycleJournalTailer(path, 7)
			if err != nil {
				t.Fatal(err)
			}
			record := valid
			mutate(&record)
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if records, err := tailer.Advance(); err == nil || len(records) != 0 {
				t.Fatalf("records=%+v err=%v, want fail closed", records, err)
			}
		})
	}

	t.Run("trailing json garbage", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		tailer, err := OpenLifecycleJournalTailer(path, 7)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(valid)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(raw, []byte(" garbage\n")...), 0o600); err != nil {
			t.Fatal(err)
		}
		if records, err := tailer.Advance(); err == nil || len(records) != 0 {
			t.Fatalf("records=%+v err=%v, want fail closed", records, err)
		}
	})
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

func TestLifecycleJournalTailerBoundsWorkPerAdvance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	tailer, err := OpenLifecycleJournalTailer(path, 7)
	if err != nil {
		t.Fatal(err)
	}
	record := func(offset int64) []byte {
		r := sessionwatch.LifecycleRecord{Version: 1, Agent: "codex", LaunchOrdinal: 7, ArtifactGeneration: "gen:1", Source: "transcript", Outcome: "completed", TurnID: "turn", Message: strings.Repeat("x", 40<<10), TranscriptPath: "rollout.jsonl", TranscriptOffset: offset, EventTimestamp: time.Unix(offset, 0).UTC()}
		raw, _ := json.Marshal(r)
		return append(raw, '\n')
	}
	if err := os.WriteFile(path, append(record(10), record(20)...), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := tailer.Advance()
	if err != nil || len(first) != 1 {
		t.Fatalf("first bounded advance: records=%d err=%v", len(first), err)
	}
	second, err := tailer.Advance()
	if err != nil || len(second) != 1 {
		t.Fatalf("second bounded advance: records=%d err=%v", len(second), err)
	}
}

type blockingLifecycleAdvancer struct {
	entered chan struct{}
	release chan struct{}
}

func (b blockingLifecycleAdvancer) Advance() ([]sessionwatch.LifecycleRecord, error) {
	close(b.entered)
	<-b.release
	return nil, nil
}

func TestMasterPumpForwardsPTYWhileLifecycleJournalIOIsBlocked(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	out := newDrainBuffer()
	p := &proxy{
		ptmx: reader, agentBasename: "codex", stdoutPump: newStdoutPump(out),
		stdoutFlushEvery: 5 * time.Millisecond, captureWindow: defaultCaptureWindow,
		notifyModeActive: notifyModeDefault, now: time.Now,
		lifecycleJournal: blockingLifecycleAdvancer{entered: entered, release: release},
	}
	done := make(chan struct{})
	go func() {
		p.masterPump()
		close(done)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("production journal worker did not enter injected IO")
	}
	if _, err := writer.Write([]byte("still-forwarded")); err != nil {
		t.Fatal(err)
	}
	waitForStdoutBatch(t, 250*time.Millisecond, func() bool {
		return string(out.Bytes()) == "still-forwarded"
	})
	close(release)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	waitForStdoutBatch(t, 250*time.Millisecond, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}
