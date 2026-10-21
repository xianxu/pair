package couchmessage

import (
	"bytes"
	"testing"
)

func FuzzWorkerCueSessionFrames(f *testing.F) {
	b := registryBinding("pair:1", "a", 7)
	build := BuildIdentity{SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	for _, frame := range []SessionFrame{
		{Op: FrameHelloV5, Binding: &b, Build: &build, WorkerCue: &WorkerCueReport{LaunchOrdinal: 1, State: WorkerCueNeeded}},
		{Op: FrameHelloV4, Binding: &b, Build: &build},
	} {
		raw, err := EncodeSessionFrame(frame)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(raw)
	}
	f.Add([]byte(`{"Op":"hello-v4","WorkerCue":{"LaunchOrdinal":1,"State":"needed"}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		frame, err := DecodeSessionFrame(raw)
		if err != nil {
			return
		}
		encoded, err := EncodeSessionFrame(frame)
		if err != nil {
			t.Fatalf("accepted frame cannot encode: %v", err)
		}
		again, err := DecodeSessionFrame(encoded)
		if err != nil {
			t.Fatalf("canonical frame cannot decode: %v", err)
		}
		canonical, err := EncodeSessionFrame(again)
		if err != nil || !bytes.Equal(encoded, canonical) {
			t.Fatalf("unstable canonical frame: %v", err)
		}
	})
}

func TestWorkerStartNoticeIdentityAndMonotonicReceipt(t *testing.T) {
	b := registryBinding("pair:1", "a", 7)
	id := WorkerStartNoticeID(b, 1)
	if id == "" || id == WorkerStartNoticeID(b, 2) {
		t.Fatal("ordinal must distinguish same-PID exec")
	}
	other := b
	other.Nonce += "changed"
	if id == WorkerStartNoticeID(other, 1) || WorkerStartNoticeID(b, 0) != "" {
		t.Fatal("binding and valid ordinal required")
	}
	report := WorkerCueReport{LaunchOrdinal: 1, State: WorkerCueNeeded}
	m := Message{ID: id, From: b, To: b, Origin: OriginSystem, Body: WorkerStartNoticeBody}
	for _, bad := range []Message{func() Message { x := m; x.Origin = OriginPeer; return x }(), func() Message { x := m; x.Body += "!"; return x }(), func() Message { x := m; x.From = other; return x }(), func() Message { x := m; x.ID = WorkerStartNoticeID(b, 2); return x }()} {
		if got := AdvanceWorkerStartNotice(report, Receipt{Message: bad, Status: Queued}); got != report {
			t.Fatalf("foreign receipt adopted: %+v", got)
		}
	}
	report = AdvanceWorkerStartNotice(report, Receipt{Message: m, Status: Queued})
	if report.State != WorkerCueCommitted || report.ID != id {
		t.Fatalf("commit: %+v", report)
	}
	report = AdvanceWorkerStartNotice(report, Receipt{Message: m, Status: Indeterminate})
	if report.State != WorkerCueUncertain {
		t.Fatalf("uncertainty: %+v", report)
	}
	if got := AdvanceWorkerStartNotice(report, Receipt{Message: m, Status: Queued}); got != report {
		t.Fatal("uncertainty reset")
	}
	report = AdvanceWorkerStartNotice(report, Receipt{Message: m, Status: Submitted})
	if report.State != WorkerCueSubmitted {
		t.Fatal("confirmed receipt not published")
	}
	if got := AdvanceWorkerStartNotice(report, Receipt{Message: m, Status: Indeterminate}); got != report {
		t.Fatal("submitted regressed")
	}
}

func TestWorkerCueReportsStrictAndCopied(t *testing.T) {
	for _, report := range []WorkerCueReport{{}, {LaunchOrdinal: 1, State: "surprise"}, {LaunchOrdinal: 1, State: WorkerCueNeeded, ID: "unexpected"}, {LaunchOrdinal: 1, State: WorkerCueCommitted}} {
		if report.Validate() == nil {
			t.Fatalf("accepted %+v", report)
		}
	}
	report := &WorkerCueReport{LaunchOrdinal: 1, State: WorkerCueNeeded}
	owned := CopyWorkerCueReport(report)
	report.State = WorkerCueUncertain
	if owned.State != WorkerCueNeeded || CopyWorkerCueReport(nil) != nil {
		t.Fatal("report alias")
	}
}

func TestWorkerStartNoticeAdmissionFencesLaunchAndPreservesOrdinaryMessages(t *testing.T) {
	b := registryBinding("pair:1", "a", 7)
	cue := Message{Origin: OriginSystem, From: b, To: b, ID: WorkerStartNoticeID(b, 7), Body: WorkerStartNoticeBody}
	if err := ValidateWorkerStartNotice(cue, 7); err != nil {
		t.Fatalf("current cue refused: %v", err)
	}
	for _, ordinal := range []uint64{0, 8} {
		if err := ValidateWorkerStartNotice(cue, ordinal); err == nil {
			t.Fatalf("launch %d accepted old cue", ordinal)
		}
	}
	for _, ordinary := range []Message{func() Message { m := cue; m.Origin = OriginPeer; return m }(), func() Message { m := cue; m.Body = "ordinary system self message"; return m }(), func() Message { m := cue; m.From = registryBinding("brain:0", "hub", 8); return m }()} {
		if err := ValidateWorkerStartNotice(ordinary, 0); err != nil {
			t.Fatalf("ordinary messaging acquired launch gate: %v", err)
		}
	}
}
