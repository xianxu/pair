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

type latestSchedule[T any] struct {
	Running         *T
	Pending         *T
	CancelRequested bool
}

type latestScheduleEventKind uint8

const (
	latestEventUnknown latestScheduleEventKind = iota
	latestRequested
	latestFinished
)

type latestScheduleEvent[T any, K comparable] struct {
	Kind     latestScheduleEventKind
	Request  T
	Identity K
}

type latestScheduleEffectKind uint8

const (
	latestEffectUnknown latestScheduleEffectKind = iota
	latestStart
	latestCancel
)

type latestScheduleEffect[T any, K comparable] struct {
	Kind     latestScheduleEffectKind
	Request  T
	Identity K
}

func advanceLatestSchedule[T any, K comparable](state latestSchedule[T], event latestScheduleEvent[T, K], key func(T) K, valid func(K) bool) (latestSchedule[T], []latestScheduleEffect[T, K]) {
	next := cloneLatestSchedule(state)
	switch event.Kind {
	case latestRequested:
		identity := key(event.Request)
		if !valid(identity) {
			return next, nil
		}
		request := event.Request
		if next.Running == nil {
			next.Running = &request
			return next, []latestScheduleEffect[T, K]{{Kind: latestStart, Request: request}}
		}
		if key(*next.Running) == identity || next.Pending != nil && key(*next.Pending) == identity {
			return next, nil
		}
		next.Pending = &request
		if !next.CancelRequested {
			next.CancelRequested = true
			return next, []latestScheduleEffect[T, K]{{Kind: latestCancel, Identity: key(*next.Running)}}
		}
	case latestFinished:
		if !valid(event.Identity) || next.Running == nil || key(*next.Running) != event.Identity {
			return next, nil
		}
		next.Running = nil
		next.CancelRequested = false
		if next.Pending != nil {
			request := *next.Pending
			next.Pending = nil
			next.Running = &request
			return next, []latestScheduleEffect[T, K]{{Kind: latestStart, Request: request}}
		}
	}
	return next, nil
}

func cloneLatestSchedule[T any](state latestSchedule[T]) latestSchedule[T] {
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

// AdvancePreviewSchedule enforces one running request and one coalesced latest
// request. Cancellation is only a request; the matching terminal outcome is
// what retires Running and admits Pending.
func AdvancePreviewSchedule(state PreviewSchedule, event PreviewScheduleEvent) (PreviewSchedule, []PreviewScheduleEffect) {
	generic := latestSchedule[PreviewRequest]{Running: state.Running, Pending: state.Pending, CancelRequested: state.CancelRequested}
	genericEvent := latestScheduleEvent[PreviewRequest, uint64]{}
	switch event.Kind {
	case PreviewRequested:
		genericEvent = latestScheduleEvent[PreviewRequest, uint64]{Kind: latestRequested, Request: event.Request}
	case PreviewFinished:
		genericEvent = latestScheduleEvent[PreviewRequest, uint64]{Kind: latestFinished, Identity: event.Generation}
	}
	next, genericEffects := advanceLatestSchedule(generic, genericEvent,
		func(request PreviewRequest) uint64 { return request.Generation },
		func(generation uint64) bool { return generation != 0 },
	)
	result := PreviewSchedule{Running: next.Running, Pending: next.Pending, CancelRequested: next.CancelRequested}
	effects := make([]PreviewScheduleEffect, 0, len(genericEffects))
	for _, effect := range genericEffects {
		switch effect.Kind {
		case latestStart:
			effects = append(effects, PreviewScheduleEffect{Kind: PreviewStart, Request: effect.Request})
		case latestCancel:
			effects = append(effects, PreviewScheduleEffect{Kind: PreviewCancel, Generation: effect.Identity})
		}
	}
	return result, effects
}
