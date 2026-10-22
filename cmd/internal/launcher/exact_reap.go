package launcher

import (
	"context"
	"errors"
	"syscall"
	"time"
)

// ErrExactReapRefused means the current effect's ownership proof refused it.
var ErrExactReapRefused = errors.New("exact reap proof refused")

// ExactSignal re-proves authority and identity and signals exactly this key.
// An error is final for this invocation, including an uncertain/lost reply.
type ExactSignal func(context.Context, ReapStep, syscall.Signal) error

// ExactReapOutcome reports one deduplicated immutable target.
type ExactReapOutcome struct {
	Step   ReapStep
	Status string
	Detail string
}

// ReapExact executes the pure invocation model. It never discovers descendants
// or signals through ProcessTable; only the caller's guarded effect may signal.
func (r Reaper) ReapExact(ctx context.Context, steps []ReapStep, signal ExactSignal) []ExactReapOutcome {
	state, _ := AdvanceExactReap(ExactReapState{}, exactReapEvent{kind: exactBegin, steps: steps})
	poll := r.Poll
	if poll <= 0 {
		poll = 50 * time.Millisecond
	}
	var deadline time.Time
	for state.phase != exactDone {
		event := exactReapEvent{kind: exactNext}
		if err := ctx.Err(); err != nil {
			event = exactReapEvent{kind: exactCancel, detail: err.Error()}
		}
		next, effects := AdvanceExactReap(state, event)
		state = next
		for _, effect := range effects {
			if err := ctx.Err(); err != nil {
				state, _ = AdvanceExactReap(state, exactReapEvent{kind: exactCancel, detail: err.Error()})
				break
			}
			switch effect.kind {
			case exactSignalEffect:
				err := ErrExactReapRefused
				if signal != nil {
					err = signal(ctx, effect.step, effect.signal)
				}
				event = exactReapEvent{kind: exactSignalReply, token: effect.token, index: effect.index, status: "sent"}
				if err != nil {
					event.status = "failed"
					event.detail = err.Error()
					if errors.Is(err, ErrServerChanged) {
						event.status = "skipped"
					} else if errors.Is(err, ErrExactReapRefused) {
						event.status = "refused"
					}
				}
			case exactProbeEffect:
				if effect.first {
					wait := r.TermWait
					if effect.phase == exactKillWait {
						wait = r.KillWait
					}
					deadline = time.Now().Add(wait)
				} else {
					remaining := time.Until(deadline)
					delay := poll
					if remaining < delay {
						delay = remaining
					}
					if delay > 0 {
						timer := time.NewTimer(delay)
						select {
						case <-ctx.Done():
							if !timer.Stop() {
								select {
								case <-timer.C:
								default:
								}
							}
						case <-timer.C:
						}
					}
				}
				if err := ctx.Err(); err != nil {
					event = exactReapEvent{kind: exactCancel, detail: err.Error()}
					break
				}
				event = exactReapEvent{kind: exactObservation, token: effect.token, observations: r.exactPresence(ctx, state.rows), deadline: !time.Now().Before(deadline)}
			}
			state, _ = AdvanceExactReap(state, event)
		}
	}
	outcomes := make([]ExactReapOutcome, len(state.rows))
	for i, row := range state.rows {
		outcomes[i] = row.outcome
	}
	return outcomes
}

func (r Reaper) exactPresence(ctx context.Context, targets []exactReapRow) []exactPresence {
	result := make([]exactPresence, len(targets))
	if r.Table == nil {
		return result
	}
	rows, err := r.Table.Snapshot(ctx)
	if err != nil || len(rows) > 65536 {
		return result
	}
	// A complete identified snapshot is required to infer absence. Ambiguity
	// anywhere vetoes this observation instead of manufacturing successful exit.
	byPID := make(map[int]string, len(rows))
	for _, row := range rows {
		if row.PID <= 0 || row.Identity == "" {
			return result
		}
		if _, ok := byPID[row.PID]; ok {
			return result
		}
		byPID[row.PID] = row.Identity
	}
	for i, target := range targets {
		if byPID[target.step.PID] == target.step.Identity {
			result[i] = exactAlive
		} else {
			result[i] = exactAbsent
		}
	}
	return result
}
