package agentfixture

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const WorkloadInterval = 100 * time.Millisecond
const WorkloadLifetime = 15 * time.Minute

type WorkloadAction string

const (
	WorkloadStart WorkloadAction = "start"
	WorkloadStop  WorkloadAction = "stop"
)

type WorkloadRequest struct {
	Action          WorkloadAction `json:"action"`
	ScrollbackLines int            `json:"scrollback_lines,omitempty"`
}

func (r WorkloadRequest) Validate() error {
	if (r.Action != WorkloadStart && r.Action != WorkloadStop) || (r.ScrollbackLines != 0 && r.ScrollbackLines != 2000) || (r.Action == WorkloadStop && r.ScrollbackLines != 0) {
		return errors.New("invalid bounded fixture workload")
	}
	return nil
}

type WorkloadStatus string

const (
	WorkloadRunning WorkloadStatus = "running"
	WorkloadStopped WorkloadStatus = "stopped"
	WorkloadExpired WorkloadStatus = "expired"
)

type WorkloadObservation struct {
	Status          WorkloadStatus `json:"status"`
	Started         time.Time      `json:"started"`
	Finished        time.Time      `json:"finished,omitempty"`
	Ticks           uint64         `json:"ticks"`
	Bytes           uint64         `json:"bytes"`
	ScrollbackLines int            `json:"scrollback_lines"`
	IntervalMS      int            `json:"interval_ms"`
}

// WorkloadState is a value owned exclusively by Run. All mutations pass through StepWorkload.
type WorkloadState struct {
	Observation WorkloadObservation
	Failure     error
	pending     bool
	pendingTick bool
	lastTick    time.Time
}
type WorkloadEventKind uint8

const (
	WorkloadBegin WorkloadEventKind = iota
	WorkloadTick
	WorkloadWritten
	WorkloadEnd
	WorkloadFailed
)

type WorkloadEvent struct {
	Kind    WorkloadEventKind
	At      time.Time
	Request WorkloadRequest
	Bytes   uint64
	Tick    bool
	Err     error
}

// StepWorkload returns whether the caller may synchronously emit one tick patch.
// Written is admitted only after a complete write. Failed writes retain the first error.
func StepWorkload(s WorkloadState, e WorkloadEvent) (WorkloadState, bool, error) {
	switch e.Kind {
	case WorkloadBegin:
		if err := e.Request.Validate(); err != nil || e.Request.Action != WorkloadStart {
			return s, false, errors.New("invalid workload start")
		}
		if s.Observation.Status == WorkloadRunning {
			return s, false, errors.New("fixture workload already running")
		}
		if s.Failure != nil {
			return s, false, s.Failure
		}
		s = WorkloadState{pending: true, lastTick: e.At, Observation: WorkloadObservation{Status: WorkloadRunning, Started: e.At, ScrollbackLines: e.Request.ScrollbackLines, IntervalMS: 100}}
	case WorkloadTick:
		if s.Observation.Status != WorkloadRunning {
			return s, false, nil
		}
		if !e.At.Before(s.Observation.Started.Add(WorkloadLifetime)) {
			s.Observation.Status = WorkloadExpired
			s.Observation.Finished = e.At
			return s, false, nil
		}
		if s.pending || e.At.Sub(s.Observation.Started)/WorkloadInterval <= s.lastTick.Sub(s.Observation.Started)/WorkloadInterval {
			return s, false, nil
		}
		s.pending = true
		s.pendingTick = true
		s.lastTick = e.At
		return s, true, nil
	case WorkloadWritten:
		if s.Observation.Status == WorkloadRunning {
			if !s.pending || e.Tick != s.pendingTick {
				return s, false, errors.New("workload completion lacks pending output")
			}
			if (!e.Tick && e.Bytes > 256<<10) || (e.Tick && (e.Bytes == 0 || e.Bytes > 128)) {
				return s, false, errors.New("workload completed output exceeds byte bound")
			}
			s.pending = false
			s.pendingTick = false
			s.Observation.Bytes += e.Bytes
			if e.Tick {
				s.Observation.Ticks++
			}
		}
	case WorkloadEnd, WorkloadFailed:
		s.pending = false
		s.pendingTick = false
		if s.Observation.Status == WorkloadRunning {
			s.Observation.Status = WorkloadStopped
			s.Observation.Finished = e.At
		}
		if e.Kind == WorkloadFailed && s.Failure == nil {
			s.Failure = e.Err
		}
	default:
		return s, false, errors.New("invalid workload event")
	}
	return s, false, nil
}
func WorkloadPrefill(lines int) string {
	if lines != 2000 {
		return ""
	}
	var b strings.Builder
	b.Grow(240000)
	padding := strings.Repeat(".", 95)
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "fixture scrollback %04d%s\r\n", i, padding)
	}
	return b.String()
}

// WorkloadPatch touches only the Working spinner, preserving the composer's
// cursor and all profile-specific recognition markers. No full-screen repaint.
func WorkloadPatch(ticks uint64) string {
	return fmt.Sprintf("\x1b7\x1b[1;1H%s\x1b8", [...]string{"◦", "•"}[ticks%2])
}

// WorkloadClock controls the actual Run owner, including its wall-clock expiry.
// Tickers have no per-tick goroutine owned by the fixture.
type WorkloadClock interface {
	Now() time.Time
	NewTicker(time.Duration) WorkloadTicker
}
type WorkloadTicker interface {
	C() <-chan time.Time
	Stop()
}
type realWorkloadClock struct{}

func (realWorkloadClock) Now() time.Time { return time.Now() }
func (realWorkloadClock) NewTicker(d time.Duration) WorkloadTicker {
	return workloadTicker{time.NewTicker(d)}
}

type workloadTicker struct{ *time.Ticker }

func (t workloadTicker) C() <-chan time.Time { return t.Ticker.C }
