package storagegc

import (
	"fmt"
	"time"
)

const CaptureRetentionPeriod = 365 * 24 * time.Hour

// CaptureEvidence concerns one immutable raw/events capture. Tag activity and
// Couch visibility are intentionally absent: neither renews a capture's age.
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
		return RetentionDecision{State: Grace, Reason: fmt.Sprintf("capture is less than %d days old", Days(CaptureRetentionPeriod)), EligibleAt: expiry}
	}
	return RetentionDecision{State: Eligible, Reason: fmt.Sprintf("capture is at least %d days old", Days(CaptureRetentionPeriod)), EligibleAt: expiry}
}
