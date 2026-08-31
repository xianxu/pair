package couchtty

// RefreshSchedule is the pure single-flight inventory refresh state. Repeated
// requests while one generation runs collapse into one dirty follow-up.
type RefreshSchedule struct {
	Sequence uint64
	Running  uint64
	Dirty    bool
}

type RefreshScheduleEventKind uint8

const (
	RefreshScheduleUnknown RefreshScheduleEventKind = iota
	RefreshRequested
	RefreshFinished
)

type RefreshScheduleEvent struct {
	Kind       RefreshScheduleEventKind
	Generation uint64
}

type RefreshScheduleEffectKind uint8

const (
	RefreshEffectUnknown RefreshScheduleEffectKind = iota
	RefreshStart
)

type RefreshScheduleEffect struct {
	Kind       RefreshScheduleEffectKind
	Generation uint64
}

// AdvanceRefreshSchedule returns work for a thin asynchronous owner without
// performing I/O. Only the matching terminal completion retires Running.
func AdvanceRefreshSchedule(state RefreshSchedule, event RefreshScheduleEvent) (RefreshSchedule, []RefreshScheduleEffect) {
	switch event.Kind {
	case RefreshRequested:
		if state.Running != 0 {
			state.Dirty = true
			return state, nil
		}
		return startRefresh(state)
	case RefreshFinished:
		if event.Generation == 0 || event.Generation != state.Running {
			return state, nil
		}
		state.Running = 0
		if !state.Dirty {
			return state, nil
		}
		state.Dirty = false
		return startRefresh(state)
	default:
		return state, nil
	}
}

func startRefresh(state RefreshSchedule) (RefreshSchedule, []RefreshScheduleEffect) {
	if state.Sequence == ^uint64(0) {
		return state, nil
	}
	state.Sequence++
	state.Running = state.Sequence
	return state, []RefreshScheduleEffect{{Kind: RefreshStart, Generation: state.Running}}
}
