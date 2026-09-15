package storagegc

import "time"

// CaptureEvidence concerns one immutable raw/events capture. Tag activity and
// Couch visibility are intentionally absent: neither renews a capture's age.
const CaptureRetentionPeriod = 7 * 24 * time.Hour

type CaptureEvidence struct {
	CapturedAt  time.Time
	Complete    bool
	Reader      Liveness
	BlockReason string
}

func DecideCapture(now time.Time, e CaptureEvidence) RetentionDecision {
	if e.Reader == ProcessAlive {
		return RetentionDecision{State: Live, Reason: "capture is being read"}
	}
	if !e.Complete || e.Reader != ProcessDead {
		return RetentionDecision{State: Blocked, Reason: "capture ownership or reader state is incomplete"}
	}
	if e.BlockReason != "" {
		return RetentionDecision{State: Blocked, Reason: e.BlockReason}
	}
	if now.IsZero() || e.CapturedAt.IsZero() || e.CapturedAt.After(now) {
		return RetentionDecision{State: Blocked, Reason: "invalid capture timestamp"}
	}
	expiry := e.CapturedAt.Add(CaptureRetentionPeriod)
	if now.Before(expiry) {
		return RetentionDecision{State: Grace, Reason: "capture is less than 7 days old", EligibleAt: expiry}
	}
	return RetentionDecision{State: Eligible, Reason: "capture is at least 7 days old", EligibleAt: expiry}
}
