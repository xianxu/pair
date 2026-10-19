package sessionwatch

import (
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"time"
)

type boundFollowPhase uint8

const (
	followReading boundFollowPhase = iota
	followPublishing
	followStopped
)

type boundCheckpoint struct {
	cursor  sessioninventory.TranscriptCursor
	scanner sessioninventory.ScannerState
}
type lifecyclePublication struct {
	record   LifecycleRecord
	position uint64
}
type boundPrepared struct {
	checkpoint   boundCheckpoint
	publications []lifecyclePublication
	watermark    uint64
}
type boundFollowState struct {
	phase     boundFollowPhase
	committed boundCheckpoint
	watermark uint64
	pending   *boundPrepared
	index     int
}
type boundFollowEvent struct {
	prepared *boundPrepared
	ack      *LifecycleRecord
	failure  error
}

// reduceBoundFollow owns cursor and publication progress. Prepared snapshots
// are immutable; the IO driver only performs the effect selected by phase.
func reduceBoundFollow(s boundFollowState, e boundFollowEvent) boundFollowState {
	if s.phase == followStopped {
		return s
	}
	if e.failure != nil {
		s.phase = followStopped
		s.pending = nil
		s.index = 0
		return s
	}
	switch s.phase {
	case followReading:
		if e.prepared == nil {
			return s
		}
		if len(e.prepared.publications) == 0 {
			s.committed = e.prepared.checkpoint
			s.watermark = max(s.watermark, e.prepared.watermark)
			return s
		}
		s.pending = e.prepared
		s.index = 0
		s.phase = followPublishing
	case followPublishing:
		if e.ack == nil || s.pending == nil || !s.pending.publications[s.index].record.sameObservation(*e.ack) {
			return s
		}
		s.watermark = max(s.watermark, s.pending.publications[s.index].position)
		s.index++
		if s.index == len(s.pending.publications) {
			s.committed = s.pending.checkpoint
			s.watermark = max(s.watermark, s.pending.watermark)
			s.pending = nil
			s.index = 0
			s.phase = followReading
		}
	}
	return s
}

func nextLifecycleDelay(base, previous time.Duration, progress bool) time.Duration {
	const ceiling = 5 * time.Second
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	base = min(base, ceiling)
	if progress || previous <= 0 {
		return base
	}
	if previous >= ceiling/2 {
		return ceiling
	}
	return min(ceiling, max(base, previous*2))
}
