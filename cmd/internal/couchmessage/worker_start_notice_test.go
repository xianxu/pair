package couchmessage

import "testing"

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
