package couchcore

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	NoticeDeliveryMaxPending   = 128
	NoticeDeliveryMaxInFlight  = 4
	NoticeDeliveryMaxBodyBytes = 4096
	NoticeDeliveryExpiry       = 2 * time.Minute
	NoticeDeliveryOverflowID   = "notice-delivery-overflow"
)

// NoticeDeliveryOccurrence is an ephemeral routing fact. ID is the preallocated
// canonical broker request ID; producers must not reuse it for another episode.
// Source is an exact condition identity, never a printable ownership alias.
type NoticeDeliveryOccurrence struct {
	ID, Condition, Source, Label, OwnerRoot, SourceRoot, Body string
	SourceSlot                                                *HubSlot
	Thread                                                    ThreadAddress
	Class                                                     NoticeClass
	Origin                                                    NoticeOrigin
	HubHandled                                                bool
}

type NoticeDeliveryOwner struct {
	// Root is verified ephemeral owner identity. Occurrences retain it when
	// their source disappears before the admission/availability decision.
	Root    string
	State   NoticeOwnerState
	Binding string
	Epoch   uint64
}
type NoticeDeliveryPhase uint8

const (
	NoticeDeliveryPending NoticeDeliveryPhase = iota
	NoticeDeliverySending
	NoticeDeliveryAwaitingReceipt
)

type NoticeDeliveryEntry struct {
	Occurrence  NoticeDeliveryOccurrence
	Owner       NoticeDeliveryOwner
	CreatedAt   time.Time
	Token       uint64
	AdmissionID string
	Phase       NoticeDeliveryPhase
}
type NoticeDeliveryState struct {
	Entries       []NoticeDeliveryEntry
	NextToken     uint64
	OverflowCount uint64
	Closed        bool
}
type NoticeDeliveryEventKind uint8

const (
	NoticeDeliveryOccurrenceAdded NoticeDeliveryEventKind = iota
	NoticeDeliveryOwnerChanged
	NoticeDeliverySendResult
	NoticeDeliveryReceiptObserved
	NoticeDeliveryConditionCleared
	NoticeDeliveryTick
	NoticeDeliveryShutdown
)

type NoticeDeliveryResult uint8

const (
	NoticeDeliveryRefused NoticeDeliveryResult = iota
	NoticeDeliveryAdmitted
	NoticeDeliveryUncertain
)

type NoticeDeliveryEvent struct {
	Kind       NoticeDeliveryEventKind
	Now        time.Time
	Occurrence NoticeDeliveryOccurrence
	ID         string
	Owner      NoticeDeliveryOwner
	Token      uint64
	Result     NoticeDeliveryResult
}
type NoticeDeliveryEffectKind uint8

const (
	NoticeDeliveryAdmit NoticeDeliveryEffectKind = iota
	NoticeDeliveryBell
	NoticeDeliveryOverflow
)

type NoticeDeliveryEffect struct {
	Kind        NoticeDeliveryEffectKind
	Occurrence  NoticeDeliveryOccurrence
	Owner       NoticeDeliveryOwner
	Token       uint64
	AdmissionID string
	Count       uint64
}

// AdvanceNoticeDelivery contains no IO. The shell publishes feed before adding
// occurrences, validates the effect's exact binding/epoch at ingress, and owns a
// fixed worker pool (including work cancelled by expiry or condition clearance).
// Receipt observation completes tracking; an uncertain result is never retried.
func AdvanceNoticeDelivery(previous NoticeDeliveryState, event NoticeDeliveryEvent) (NoticeDeliveryState, []NoticeDeliveryEffect) {
	state := previous
	state.Entries = append([]NoticeDeliveryEntry(nil), previous.Entries...)
	if state.Closed {
		return state, nil
	}
	if event.Kind == NoticeDeliveryShutdown {
		state.Entries = nil
		state.Closed = true
		return state, nil
	}
	var effects []NoticeDeliveryEffect
	// Time is an explicit input; removal never generates a fallback bell.
	keep := state.Entries[:0]
	for _, entry := range state.Entries {
		if event.Now.IsZero() || event.Now.Before(entry.CreatedAt.Add(NoticeDeliveryExpiry)) {
			keep = append(keep, entry)
		}
	}
	state.Entries = keep
	switch event.Kind {
	case NoticeDeliveryOccurrenceAdded:
		occurrence := event.Occurrence
		if occurrence.ID == "" {
			break
		}
		duplicate := false
		for _, entry := range state.Entries {
			if entry.Occurrence.ID == occurrence.ID || occurrence.Condition != "" && entry.Occurrence.Condition == occurrence.Condition && entry.Occurrence.Source == occurrence.Source && entry.Occurrence.Class == occurrence.Class {
				duplicate = true
				break
			}
		}
		if duplicate {
			break
		}
		if len(occurrence.Body) > NoticeDeliveryMaxBodyBytes {
			body := occurrence.Body[:NoticeDeliveryMaxBodyBytes]
			for len(body) > 0 && !utf8.ValidString(body) {
				body = body[:len(body)-1]
			}
			// Clone the bounded prefix so a large original backing string is released.
			occurrence.Body = strings.Clone(body)
		}
		route := RouteNotice(NoticeRouteInput{Class: occurrence.Class, Origin: occurrence.Origin, Owner: event.Owner.State, HubHandled: occurrence.HubHandled})
		if route.Bell {
			effects = append(effects, NoticeDeliveryEffect{Kind: NoticeDeliveryBell, Occurrence: occurrence})
			break
		}
		if !route.Hub && !route.Defer {
			break
		}
		if len(state.Entries) >= NoticeDeliveryMaxPending {
			if state.OverflowCount < ^uint64(0) {
				state.OverflowCount++
			}
			effects = append(effects, NoticeDeliveryEffect{Kind: NoticeDeliveryOverflow, Occurrence: NoticeDeliveryOccurrence{ID: NoticeDeliveryOverflowID}, Count: state.OverflowCount})
			break
		}
		state.Entries = append(state.Entries, NoticeDeliveryEntry{Occurrence: occurrence, Owner: event.Owner, CreatedAt: event.Now, AdmissionID: occurrence.ID})
	case NoticeDeliveryOwnerChanged:
		for i := range state.Entries {
			entry := &state.Entries[i]
			if entry.Phase == NoticeDeliveryPending && (event.ID == "" || entry.Occurrence.ID == event.ID) {
				entry.Owner = event.Owner
				if event.Owner.Root != "" {
					entry.Occurrence.OwnerRoot = event.Owner.Root
				}
			}
		}
	case NoticeDeliverySendResult, NoticeDeliveryReceiptObserved:
		for i := range state.Entries {
			entry := &state.Entries[i]
			if entry.Occurrence.ID != event.ID || entry.Token != event.Token || entry.Phase == NoticeDeliveryPending {
				continue
			}
			if event.Kind == NoticeDeliveryReceiptObserved || entry.Phase == NoticeDeliverySending && event.Result == NoticeDeliveryRefused {
				if event.Kind == NoticeDeliverySendResult && entry.Owner.Epoch == event.Owner.Epoch {
					route := RouteNotice(NoticeRouteInput{Class: entry.Occurrence.Class, Origin: entry.Occurrence.Origin, Owner: event.Owner.State, HubHandled: entry.Occurrence.HubHandled})
					if route.Bell {
						effects = append(effects, NoticeDeliveryEffect{Kind: NoticeDeliveryBell, Occurrence: entry.Occurrence})
					}
				}
				state.Entries = append(state.Entries[:i], state.Entries[i+1:]...)
			} else if entry.Phase == NoticeDeliverySending {
				entry.Phase = NoticeDeliveryAwaitingReceipt
			}
			break
		}
	case NoticeDeliveryConditionCleared:
		keep = state.Entries[:0]
		for _, entry := range state.Entries {
			matched := event.ID != "" && entry.Occurrence.ID == event.ID || event.Occurrence.Condition != "" && entry.Occurrence.Condition == event.Occurrence.Condition
			if !matched {
				keep = append(keep, entry)
			}
		}
		state.Entries = keep
	}
	inflight := 0
	for _, entry := range state.Entries {
		if entry.Phase == NoticeDeliverySending {
			inflight++
		}
	}
	keep = state.Entries[:0]
	for _, entry := range state.Entries {
		if entry.Phase != NoticeDeliveryPending {
			keep = append(keep, entry)
			continue
		}
		route := RouteNotice(NoticeRouteInput{Class: entry.Occurrence.Class, Origin: entry.Occurrence.Origin, Owner: entry.Owner.State, HubHandled: entry.Occurrence.HubHandled})
		if route.Bell {
			effects = append(effects, NoticeDeliveryEffect{Kind: NoticeDeliveryBell, Occurrence: entry.Occurrence})
			continue
		}
		if !route.Hub && !route.Defer {
			continue
		}
		if route.Hub && inflight < NoticeDeliveryMaxInFlight {
			state.NextToken++
			entry.Token = state.NextToken
			entry.Phase = NoticeDeliverySending
			inflight++
			effects = append(effects, NoticeDeliveryEffect{Kind: NoticeDeliveryAdmit, Occurrence: entry.Occurrence, Owner: entry.Owner, Token: entry.Token, AdmissionID: entry.AdmissionID})
		}
		keep = append(keep, entry)
	}
	state.Entries = keep
	return state, effects
}
