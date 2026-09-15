package storagegc

import (
	"testing"
	"time"
)

func TestCaptureExpiresIndependently(t *testing.T) {
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	e := CaptureEvidence{CapturedAt: now.Add(-7 * 24 * time.Hour), Complete: true, Reader: ProcessDead}
	if d := DecideCapture(now, e); d.State != Eligible {
		t.Fatal(d)
	}
	if d := DecideCapture(now.Add(-time.Nanosecond), e); d.State != Grace {
		t.Fatal(d)
	}
	e.Reader = ProcessAlive
	if d := DecideCapture(now, e); d.State != Live {
		t.Fatal(d)
	}
	e.Reader = ProcessUnknown
	if d := DecideCapture(now, e); d.State != Blocked {
		t.Fatal(d)
	}
	e.Reader = ProcessDead
	e.CapturedAt = now.Add(time.Hour)
	if d := DecideCapture(now, e); d.State != Blocked {
		t.Fatal(d)
	}
}

func FuzzCaptureNeverExpiresWithoutEvidence(f *testing.F) {
	f.Add(int64(0))
	f.Fuzz(func(t *testing.T, seconds int64) {
		now := time.Unix(seconds%1000000000, 0)
		for _, e := range []CaptureEvidence{{CapturedAt: now.Add(-7 * 24 * time.Hour), Reader: ProcessDead}, {Complete: true, Reader: ProcessDead}, {CapturedAt: now.Add(-7 * 24 * time.Hour), Complete: true}} {
			if d := DecideCapture(now, e); d.State == Eligible {
				t.Fatal("incomplete capture eligible")
			}
		}
	})
}
