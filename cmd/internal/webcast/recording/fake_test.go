package recording

import (
	"context"
	"testing"
	"time"
)

func TestFakeHubMeetsTheLiveContract(t *testing.T) {
	RunLiveContract(t, func() (Sink, Live) { h := NewFakeHub(64); return h, h })
}

func TestContractRecordingIsValid(t *testing.T) {
	if fs, _ := Validate(filesOf(t, ContractRecording(5))); len(fs) != 0 {
		t.Fatal(fs)
	}
}

func TestFakeHubIsBoundedBySegment(t *testing.T) {
	h := NewFakeHub(8)
	recs := ContractRecording(500)
	for _, r := range recs {
		h.Put(r)
	}
	// The last segment is small; the 500 changes of the first are gone.
	if n := h.Retained(); n > 20 {
		t.Fatalf("the hub retains %d records after a navigation", n)
	}
}

func TestMemSink(t *testing.T) {
	var m MemSink
	recs := ContractRecording(3)
	if err := (FakeRecorder{Records: recs}).Run(context.Background(), &m); err != nil {
		t.Fatal(err)
	}
	if got := m.Records(); len(got) != len(recs) || got[len(got)-1] != recs[len(recs)-1] {
		t.Fatalf("got %d records", len(got))
	}
	if err := m.Put(&Title{Header: Header{T: -1}}); err == nil {
		t.Error("time going back must be refused")
	}
	m.Close()
	if err := m.Put(&Title{Header: Header{T: 1 << 40}}); err != ErrClosed || !m.Closed() {
		t.Errorf("put after close: %v", err)
	}
}

func TestFakeRecorderPacesAndStops(t *testing.T) {
	b := NewBuilder("p")
	b.At(200).Add(&Title{V: "x"})
	var m MemSink
	start := time.Now()
	if err := (FakeRecorder{Records: b.Recs, Speed: 1}).Run(context.Background(), &m); err != nil {
		t.Fatal(err)
	}
	if el := time.Since(start); el < 150*time.Millisecond {
		t.Errorf("real time took %v for a 200ms recording", el)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	b.At(100000).Add(&Title{V: "y"})
	if err := (FakeRecorder{Records: b.Recs, Speed: 1}).Run(ctx, &MemSink{}); err == nil {
		t.Error("a cancelled run must stop")
	}
}

func TestTee(t *testing.T) {
	var a, b MemSink
	s := Tee(&a, &b)
	s.Put(&Title{V: "x"})
	s.Close()
	if len(a.Records()) != 1 || len(b.Records()) != 1 || !a.Closed() || !b.Closed() {
		t.Fatal("tee must forward to every sink")
	}
}
