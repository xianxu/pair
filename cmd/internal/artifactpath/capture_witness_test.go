package artifactpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func witnessFixture() CaptureWitness {
	return CaptureWitness{Type: "capture_witness", Version: 1, Generation: "generation-a", ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 18, Nonce: "start-a", RawFile: CaptureFileID{Device: 1, Inode: 2}, EventsFile: CaptureFileID{Device: 1, Inode: 3}}
}
func TestCaptureWitnessStrictGeneration(t *testing.T) {
	w := witnessFixture()
	raw, _ := json.Marshal(w)
	raw = append(raw, '\n')
	if got, e := DecodeCaptureWitness(append(raw, []byte("{\"type\":\"resize\",\"offset\":0}\n")...)); e != nil || got != w {
		t.Fatalf("%+v %v", got, e)
	}
	for _, bad := range [][]byte{append(append([]byte{}, raw...), []byte(`{"type":"capture_witness","type":"resize"}`)...), append(append([]byte{}, raw...), raw...), append([]byte("{\"type\":\"resize\"}\n"), raw...), bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1), bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"extra":1`), 1)} {
		if _, e := DecodeCaptureWitness(bad); e == nil {
			t.Fatal("accepted mixed/ambiguous witness")
		}
	}
	for _, mutate := range []func(*CaptureWitness){func(w *CaptureWitness) { w.Offset = 1 }, func(w *CaptureWitness) { w.LaunchOrdinal = 0 }, func(w *CaptureWitness) { w.Nonce = "" }, func(w *CaptureWitness) { w.Generation = "" }, func(w *CaptureWitness) { w.EventsFile = w.RawFile }} {
		bad := w
		mutate(&bad)
		if ValidateCaptureWitness(bad) == nil {
			t.Fatal("accepted invalid generation")
		}
	}
	if _, e := DecodeCaptureWitness([]byte("{\"type\":\"resize\",\"offset\":0}\n")); !errors.Is(e, ErrCaptureWitnessAbsent) {
		t.Fatal(e)
	}
}
func FuzzCaptureWitness(f *testing.F) {
	b, _ := json.Marshal(witnessFixture())
	f.Add(b)
	f.Fuzz(func(t *testing.T, b []byte) {
		w, e := DecodeCaptureWitness(b)
		if e == nil && ValidateCaptureWitness(w) != nil {
			t.Fatal("invalid accepted witness")
		}
	})
}

func TestCaptureWitnessV2OpenedAtAndLegacyCompatibility(t *testing.T) {
	legacy := witnessFixture()
	raw, _ := json.Marshal(legacy)
	if bytes.Contains(raw, []byte("opened_at")) {
		t.Fatal("legacy encoding gained invented timestamp")
	}
	got, err := DecodeCaptureWitness(raw)
	if err != nil || !got.OpenedAt.IsZero() {
		t.Fatalf("legacy=%+v err=%v", got, err)
	}
	current := legacy
	current.Version = 2
	current.OpenedAt = time.Date(2026, 10, 13, 12, 0, 0, 123, time.UTC)
	raw, _ = json.Marshal(current)
	got, err = DecodeCaptureWitness(raw)
	if err != nil || !got.OpenedAt.Equal(current.OpenedAt) {
		t.Fatalf("v2=%+v err=%v", got, err)
	}
	for _, bad := range []CaptureWitness{func() CaptureWitness { w := current; w.OpenedAt = time.Time{}; return w }(), func() CaptureWitness { w := legacy; w.OpenedAt = current.OpenedAt; return w }(), func() CaptureWitness { w := current; w.Version = 3; return w }()} {
		if ValidateCaptureWitness(bad) == nil {
			t.Fatal("invalid version/time accepted")
		}
	}
	legacyRaw, _ := json.Marshal(legacy)
	for _, value := range []string{`null`, `"0001-01-01T00:00:00Z"`, `"2026-10-13T12:00:00Z"`} {
		bad := bytes.Replace(legacyRaw, []byte(`"version":1`), []byte(`"version":1,"opened_at":`+value), 1)
		if _, err := DecodeCaptureWitness(bad); err == nil {
			t.Fatal("v1 timestamp field accepted")
		}
	}
}
