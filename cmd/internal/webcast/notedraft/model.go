// Package notedraft owns private draw-time capture evidence independently of
// the note's current placement. Adapters request transitions; they never own files.
package notedraft

import "fmt"

const (
	MaxStoreBytes int64 = 256 << 20
	MaxPNGBytes   int64 = 8 << 20
	MaxRecords          = 128
	CaptureBusy         = "crop unavailable: capture busy"
	StoreFull           = "capture store full: submit or resolve notes to free space"
)

type Phase string

const (
	Preparing    Phase = "preparing"
	Ready        Phase = "ready"
	Attaching    Phase = "attaching"
	Attached     Phase = "attached"
	Transferring Phase = "transferring"
	Bundled      Phase = "bundled"
	Retired      Phase = "retired"
)

type State struct {
	Phase       Phase  `json:"phase"`
	Unavailable string `json:"unavailable,omitempty"`
	Abandoned   bool   `json:"abandoned,omitempty"`
	// PreviousBundle preserves the source owner through a repeated submission.
	PreviousBundle bool `json:"previous_bundle,omitempty"`
}
type EventKind string

const (
	Begin           EventKind = "begin"
	CaptureComplete EventKind = "capture-complete"
	CaptureFailed   EventKind = "capture-failed"
	Save            EventKind = "save"
	WriteConfirmed  EventKind = "write-confirmed"
	WriteAbsent     EventKind = "write-absent"
	Publish         EventKind = "publish"
	BundleConfirmed EventKind = "bundle-confirmed"
	BundleAbsent    EventKind = "bundle-absent"
	Cancel          EventKind = "cancel"
	Expire          EventKind = "expire"
	Resolve         EventKind = "resolve"
)

type Event struct {
	Kind   EventKind
	Reason string
}
type Effect string

const (
	Capture         Effect = "capture"
	CancelCapture   Effect = "cancel-capture"
	PersistIntent   Effect = "persist-intent"
	EmitNote        Effect = "emit-note"
	ConfirmWrite    Effect = "confirm-write"
	PublishBundle   Effect = "publish-bundle"
	ConfirmBundle   Effect = "confirm-bundle"
	RebindBundle    Effect = "rebind-bundle"
	DeleteSource    Effect = "delete-source"
	DeleteReference Effect = "delete-reference"
)

// Step is the only ownership transition authority. Effects run in listed order,
// after the resulting intent is durable. A timeout is never absence evidence.
func Step(s State, e Event) (State, []Effect, error) {
	n := s
	if s.Phase == Retired {
		if e.Kind == CaptureComplete {
			return s, []Effect{DeleteSource}, nil
		}
		return s, nil, nil
	}
	switch s.Phase {
	case "":
		if e.Kind == Begin {
			n.Phase = Preparing
			return n, []Effect{PersistIntent, Capture}, nil
		}
	case Preparing:
		switch e.Kind {
		case CaptureComplete:
			n.Phase = Ready
			return n, nil, nil
		case CaptureFailed:
			if e.Reason != "" {
				n.Phase = Ready
				n.Unavailable = e.Reason
				return n, nil, nil
			}
		case Cancel, Expire, Resolve:
			n.Phase = Retired
			return n, []Effect{CancelCapture, DeleteSource}, nil
		}
	case Ready:
		switch e.Kind {
		case Save:
			n.Phase = Attaching
			return n, []Effect{PersistIntent, EmitNote}, nil
		case Cancel, Expire, Resolve:
			n.Phase = Retired
			return n, []Effect{DeleteSource}, nil
		}
	case Attaching:
		switch e.Kind {
		case WriteConfirmed:
			n.Phase = Attached
			n.Abandoned = false
			return n, nil, nil
		case WriteAbsent:
			if n.Abandoned {
				n.Phase = Retired
				return n, []Effect{DeleteSource}, nil
			}
			n.Phase = Ready
			return n, nil, nil
		case Cancel, Expire, Resolve:
			n.Abandoned = true
			return n, []Effect{PersistIntent, ConfirmWrite}, nil
		case Save:
			return n, []Effect{ConfirmWrite}, nil
		}
	case Attached, Bundled:
		switch e.Kind {
		case Publish:
			n.PreviousBundle = s.Phase == Bundled
			n.Phase = Transferring
			return n, []Effect{PersistIntent, PublishBundle}, nil
		case Resolve:
			n.Phase = Retired
			if s.Phase == Bundled {
				return n, []Effect{DeleteReference}, nil
			}
			return n, []Effect{DeleteSource, DeleteReference}, nil
		case Cancel, Expire, WriteConfirmed:
			return n, nil, nil
		}
	case Transferring:
		switch e.Kind {
		case BundleConfirmed:
			n.Phase = Bundled
			fx := []Effect{RebindBundle, DeleteSource}
			if n.Abandoned {
				n.Phase = Retired
				fx = append(fx, DeleteReference)
			}
			return n, fx, nil
		case BundleAbsent:
			n.Phase = Attached
			if n.PreviousBundle {
				n.Phase = Bundled
			}
			if n.Abandoned {
				return Step(n, Event{Kind: Resolve})
			}
			return n, nil, nil
		case Resolve:
			n.Abandoned = true
			return n, []Effect{PersistIntent, ConfirmBundle}, nil
		case Cancel, Expire:
			return n, nil, nil
		case Publish:
			return n, []Effect{ConfirmBundle}, nil
		}
	}
	return s, nil, fmt.Errorf("capture: cannot %s while %s", e.Kind, s.Phase)
}

// Capacity counts source records and reserved worst-case bytes. References to
// already published bundles do not consume capture capacity.
func Capacity(records int, bytes int64, addRecords int, addBytes int64) string {
	if records < 0 || bytes < 0 || addRecords < 0 || addBytes < 0 || records > MaxRecords-addRecords || bytes > MaxStoreBytes-addBytes {
		return StoreFull
	}
	return ""
}

// Reconcile uses independently observed durable evidence. Unknown observations
// preserve ownership; absence must be established by the writer/store barrier.
func Reconcile(s State, write, bundle Observation) (State, []Effect, error) {
	if s.Phase == Attaching {
		if write == Confirmed {
			return Step(s, Event{Kind: WriteConfirmed})
		}
		if write == Absent {
			return Step(s, Event{Kind: WriteAbsent})
		}
	}
	if s.Phase == Transferring {
		if bundle == Confirmed {
			return Step(s, Event{Kind: BundleConfirmed})
		}
		if bundle == Absent {
			return Step(s, Event{Kind: BundleAbsent})
		}
	}
	return s, nil, nil
}

type Observation string

const (
	Unknown   Observation = "unknown"
	Confirmed Observation = "confirmed"
	Absent    Observation = "absent"
)
