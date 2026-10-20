package couchcore

import (
	"sort"
	"time"
)

type DrainRow struct {
	Workspace, Identity string
	State               DrainVisibleState
	Reason, Recovery    string
	Ack                 bool
}
type DrainBoard struct {
	ID, Owner, Reason string
	Fast              bool
	Deadline          time.Time
	State             DrainVisibleState
	Detail, Recovery  string
	Rows              []DrainRow
	Queue             DrainQueueObservation
}

// ProjectDrainBoard copies all aggregate values; reading a board performs no IO.
func ProjectDrainBoard(s DrainState, now time.Time) DrainBoard {
	b := DrainBoard{ID: s.ID, Owner: s.Owner, Reason: s.Reason, Fast: s.Fast, Deadline: s.Deadline, State: DrainDraining, Recovery: "wait", Queue: s.Queue}
	b.Queue.Owners = append([]string(nil), s.Queue.Owners...)
	b.Queue.Active = append([]DrainActivity(nil), s.Queue.Active...)
	keys := make([]string, 0, len(s.Members))
	for k := range s.Members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	activityByWorkspace := make(map[string]string, len(s.Queue.Active))
	for _, a := range s.Queue.Active {
		activityByWorkspace[a.Workspace] = a.Kind
	}
	for _, k := range keys {
		m := s.Members[k]
		r := DrainRow{Workspace: m.Workspace, Identity: m.Identity, State: DrainDraining, Recovery: "wait", Ack: m.Ack}
		switch {
		case m.Error != "":
			r.State = DrainStuck
			r.Reason = m.Error
			r.Recovery = "inspect status"
		case m.Notice == DrainUnknown || m.Notice == DrainRefused || m.Interrupt == DrainUnknown || m.Interrupt == DrainRefused:
			r.State = DrainStuck
			r.Reason = "control outcome unconfirmed"
			r.Recovery = "inspect status"
		case m.Notice != DrainDone:
			r.Reason = "checkpoint requested"
		case !m.Observation.Known || !m.Observation.Connected:
			r.State = DrainStuck
			r.Reason = "identity or observation unknown"
			r.Recovery = "inspect status"
		case !m.Ack && m.Observation.Idle && !m.Observation.Picker && !m.Observation.Background:
			r.State = DrainStuck
			r.Reason = "no ack"
			r.Recovery = "checkpoint and ack"
		case !m.Ack:
			r.Reason = "active turn"
		case !m.AckSealed || !m.Observation.Idle || m.Observation.Picker || m.Observation.Background:
			r.Reason = "waiting for idle"
		default:
			r.State = DrainDrained
		}
		if kind, ok := activityByWorkspace[m.Workspace]; ok && r.State == DrainDrained {
			r.State = DrainDraining
			r.Reason = "waiting for " + kind
		}

		b.Rows = append(b.Rows, r)
	}
	switch {
	case s.Phase == DrainFinished:
		b.State = DrainCancelled
		b.Detail = "cancelled"
		b.Recovery = "inspect status"
		return b
	case s.Error != "":
		b.State = DrainStuck
		b.Detail = s.Error
		b.Recovery = "inspect status"
		return b
	case s.Phase == DrainRecovered:
		b.State = DrainStuck
		b.Detail = "recovered drain; cancel before a new drain"
		b.Recovery = "retry cancel"
		return b
	case s.Phase == DrainCancelling:
		b.Detail = "cancelling (dispatch held)"
		b.Recovery = "retry cancel"
		for _, m := range s.Members {
			if m.Revoke == DrainUnknown || m.Revoke == DrainRefused {
				b.State = DrainStuck
				b.Detail = "cancellation unconfirmed"
			}
		}
		return b
	case s.Phase == DrainPreparing:
		b.Detail = "pausing queue"
		if !s.Deadline.IsZero() && !now.Before(s.Deadline) {
			b.State = DrainStuck
			b.Detail = "deadline reached: queue pause unconfirmed"
			b.Recovery = "inspect status"
		}
		if s.Queue.Detail != "" {
			b.State = DrainStuck
			b.Detail = s.Queue.Detail
			b.Recovery = "inspect status"
		}
		return b
	case !drainOwnPause(s):
		b.State = DrainStuck
		b.Detail = "queue pause unknown"
		b.Recovery = "inspect status"
		return b
	}
	all := true
	for _, r := range b.Rows {
		if r.State != DrainDrained {
			all = false
		}
		if r.State == DrainStuck {
			b.State = DrainStuck
		}
	}
	if len(s.Queue.Active) != 0 {
		all = false
		b.Detail = "active suite, merge or landing work"
		for _, a := range s.Queue.Active {
			if a.Workspace == "" {
				b.State = DrainStuck
				b.Detail = "external work identity unknown"
			}
		}
	}
	if all {
		b.State = DrainDrained
		b.Detail = "checkpointed and quiescent"
	} else if !s.Deadline.IsZero() && !now.Before(s.Deadline) {
		b.State = DrainStuck
		b.Detail = "deadline reached"
		b.Recovery = "inspect status"
	}
	return b
}
