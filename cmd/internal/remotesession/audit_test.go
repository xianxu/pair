package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func auditFixture(now time.Time) AuditRecord {
	return AuditRecord{At: now, Action: "edit", Principal: strings.Repeat("a", 64), Session: strings.Repeat("b", 64), Epoch: 1, DocumentID: strings.Repeat("c", 32), Sequence: 1, Before: "rev1", After: "rev2", Outcome: "pending"}
}
func TestAuditCountAndRace(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(string(rune('a'+n)), func(t *testing.T) {
			now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
			d := t.TempDir()
			a, err := OpenAudit(filepath.Join(d, "active"), filepath.Join(d, "previous"), func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					r := auditFixture(now)
					r.Sequence = uint64(i + 1)
					if err := a.AppendIntent(r); err != nil {
						t.Error(err)
					}
					r.Outcome = "applied"
					if err := a.AppendResult(r); err != nil {
						t.Error(err)
					}
				}(i)
			}
			wg.Wait()
			raw, err := os.ReadFile(a.activePath)
			if err != nil {
				t.Fatal(err)
			}
			var intents, results int
			for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
				r, err := decodeAuditRecord(line)
				if err != nil {
					t.Fatal(err)
				}
				if r.kind == "intent" {
					intents++
				} else {
					results++
				}
			}
			if intents != n || results != n {
				t.Fatalf("counts %d/%d", intents, results)
			}
		})
	}
}
func TestAuditRotationRetentionAndFutureFields(t *testing.T) {
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	d := t.TempDir()
	active, previous := filepath.Join(d, "active"), filepath.Join(d, "previous")
	r := auditFixture(now)
	r.kind = "intent"
	raw, err := encodeAuditRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]json.RawMessage
	json.Unmarshal(raw, &root)
	root["future"] = json.RawMessage(`{"huge":1e999,"spaced": [ 1, 2 ]}`)
	var meta map[string]json.RawMessage
	json.Unmarshal(root["metadata"], &meta)
	meta["future_meta"] = json.RawMessage(`{"value":9007199254740993}`)
	root["metadata"] = marshalRemoteFields(meta)
	raw = marshalRemoteFields(root)
	os.WriteFile(active, append(raw, '\n'), 0600)
	a, err := OpenAudit(active, previous, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	a.segmentLimit = len(raw) + 2
	if err := a.AppendIntent(auditFixture(now)); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(previous)
	if err != nil {
		t.Fatal(err)
	}
	rr, err := decodeAuditRecord(bytes.TrimSpace(saved))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rr.raw["future"], root["future"]) || !bytes.Equal(rr.metadata["future_meta"], meta["future_meta"]) {
		t.Fatal("future fields changed")
	}
	now = now.Add(31 * 24 * time.Hour)
	if _, err := OpenAudit(active, previous, func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{active, previous} {
		b, err := os.ReadFile(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if len(b) != 0 {
			t.Fatalf("expired segment retained %s", p)
		}
	}
}
func TestAuditFailurePoisonsWriter(t *testing.T) {
	now := time.Now().UTC()
	d := t.TempDir()
	a, err := OpenAudit(filepath.Join(d, "a"), filepath.Join(d, "p"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	a.appendLine = func(string, []byte) error { calls++; return errors.New("disk full") }
	if a.AppendIntent(auditFixture(now)) == nil {
		t.Fatal("intent succeeded")
	}
	if a.AppendIntent(auditFixture(now)) == nil || calls != 1 {
		t.Fatal("poisoned writer retried")
	}
}
func TestAuditRefusesEvidenceWithoutErasing(t *testing.T) {
	now := time.Now().UTC()
	for _, raw := range []string{"{bad}\n", `{"version":2}` + "\n"} {
		d := t.TempDir()
		p := filepath.Join(d, "a")
		os.WriteFile(p, []byte(raw), 0600)
		if _, err := OpenAudit(p, filepath.Join(d, "p"), func() time.Time { return now }); err == nil {
			t.Fatal("accepted malformed evidence")
		}
		b, _ := os.ReadFile(p)
		if string(b) != raw {
			t.Fatal("erased evidence")
		}
	}
	d := t.TempDir()
	p := filepath.Join(d, "a")
	os.Symlink(filepath.Join(d, "target"), p)
	if _, err := OpenAudit(p, filepath.Join(d, "p"), func() time.Time { return now }); err == nil {
		t.Fatal("accepted symlink")
	}
}

func TestAuditResultAndRotationFailuresPoison(t *testing.T) {
	for _, stage := range []string{"result", "previous", "active"} {
		t.Run(stage, func(t *testing.T) {
			now := time.Now().UTC()
			d := t.TempDir()
			a, err := OpenAudit(filepath.Join(d, "a"), filepath.Join(d, "p"), func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			r := auditFixture(now)
			if err = a.AppendIntent(r); err != nil {
				t.Fatal(err)
			}
			if stage == "result" {
				a.appendLine = func(string, []byte) error { return errors.New("result fsync failed") }
			} else {
				a.segmentLimit = a.size + 1
				realPublish := a.publish
				a.publish = func(p string, b []byte) error {
					if stage == "previous" && p == a.previousPath || stage == "active" && p == a.activePath {
						return errors.New("publication failed")
					}
					return realPublish(p, b)
				}
			}
			r.Outcome = "applied"
			if a.AppendResult(r) == nil {
				t.Fatal("result falsely succeeded")
			}
			if a.AppendIntent(r) == nil {
				t.Fatal("poisoned writer admitted mutation")
			}
			b, err := os.ReadFile(a.activePath)
			if err != nil {
				t.Fatal(err)
			}
			if len(b) == 0 {
				t.Fatal("intent evidence disappeared")
			}
		})
	}
}
func TestAuditSegmentsStayBounded(t *testing.T) {
	now := time.Now().UTC()
	d := t.TempDir()
	a, err := OpenAudit(filepath.Join(d, "a"), filepath.Join(d, "p"), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	r := auditFixture(now)
	r.kind = "intent"
	b, _ := encodeAuditRecord(r)
	a.segmentLimit = 2 * (len(b) + 1)
	for i := 0; i < 16; i++ {
		r.Sequence = uint64(i + 1)
		if err := a.AppendIntent(r); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("segment files %d", len(entries))
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > int64(a.segmentLimit) {
			t.Fatal("segment unbounded")
		}
	}
}
func TestAuditRetentionRewritePreservesFutureFields(t *testing.T) {
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	d := t.TempDir()
	active, previous := filepath.Join(d, "a"), filepath.Join(d, "p")
	var lines []byte
	for _, at := range []time.Time{now.Add(-29 * 24 * time.Hour), now} {
		r := auditFixture(at)
		r.kind = "intent"
		b, _ := encodeAuditRecord(r)
		root, _ := remoteObject(b)
		root["future"] = json.RawMessage(`{"big":1e999,"opaque": [ 2, 3 ]}`)
		b = marshalRemoteFields(root)
		lines = append(lines, b...)
		lines = append(lines, '\n')
	}
	if err := os.WriteFile(active, lines, 0600); err != nil {
		t.Fatal(err)
	}
	a, err := OpenAudit(active, previous, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * 24 * time.Hour)
	if err := a.AppendIntent(auditFixture(now)); err != nil {
		t.Fatal(err)
	}
	_, records, err := readAuditSegment(active)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("retained %d", len(records))
	}
	if !bytes.Equal(records[0].raw["future"], []byte(`{"big":1e999,"opaque": [ 2, 3 ]}`)) {
		t.Fatal("retention changed unknown")
	}
}
