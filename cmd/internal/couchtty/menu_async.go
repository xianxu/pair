package couchtty

// PreviewRequest is one immutable start-form generation submitted to the
// asynchronous owner boundary.
type PreviewRequest struct {
	Generation uint64
	Path       string
	Agent      string
}

type PreviewSchedule struct {
	Running         *PreviewRequest
	Pending         *PreviewRequest
	CancelRequested bool
}

type PreviewScheduleEventKind uint8

const (
	PreviewScheduleUnknown PreviewScheduleEventKind = iota
	PreviewRequested
	PreviewFinished
)

type PreviewScheduleEvent struct {
	Kind       PreviewScheduleEventKind
	Request    PreviewRequest
	Generation uint64
}

type PreviewScheduleEffectKind uint8

const (
	PreviewEffectUnknown PreviewScheduleEffectKind = iota
	PreviewStart
	PreviewCancel
)

type PreviewScheduleEffect struct {
	Kind       PreviewScheduleEffectKind
	Request    PreviewRequest
	Generation uint64
}

// AdvancePreviewSchedule enforces one running request and one coalesced latest
// request. Cancellation is only a request; the matching terminal outcome is
// what retires Running and admits Pending.
func AdvancePreviewSchedule(state PreviewSchedule, event PreviewScheduleEvent) (PreviewSchedule, []PreviewScheduleEffect) {
	next := clonePreviewSchedule(state)
	switch event.Kind {
	case PreviewRequested:
		request := event.Request
		if request.Generation == 0 {
			return next, nil
		}
		if next.Running == nil {
			next.Running = &request
			return next, []PreviewScheduleEffect{{Kind: PreviewStart, Request: request}}
		}
		if next.Running.Generation == request.Generation || next.Pending != nil && next.Pending.Generation == request.Generation {
			return next, nil
		}
		next.Pending = &request
		if !next.CancelRequested {
			next.CancelRequested = true
			return next, []PreviewScheduleEffect{{Kind: PreviewCancel, Generation: next.Running.Generation}}
		}
	case PreviewFinished:
		if event.Generation == 0 {
			return next, nil
		}
		if next.Running == nil || next.Running.Generation != event.Generation {
			return next, nil
		}
		next.Running = nil
		next.CancelRequested = false
		if next.Pending != nil {
			request := *next.Pending
			next.Pending = nil
			next.Running = &request
			return next, []PreviewScheduleEffect{{Kind: PreviewStart, Request: request}}
		}
	}
	return next, nil
}

func clonePreviewSchedule(state PreviewSchedule) PreviewSchedule {
	next := state
	if state.Running != nil {
		running := *state.Running
		next.Running = &running
	}
	if state.Pending != nil {
		pending := *state.Pending
		next.Pending = &pending
	}
	return next
}
