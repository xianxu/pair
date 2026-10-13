package artifactpath

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
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
