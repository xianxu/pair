package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type resourcePersistence struct {
	store               couchcore.ResourceSnapshotStore
	checkpoint          resourceCheckpoint
	history             couchcore.ResourceHistory
	dispatcher          resourceAlertDispatcher
	loadError           error // current read failures, retained for startup diagnostics
	checkpointReadError error
	historyReadError    error
	continuity          []string
}

func newResourcePersistence(dir string, d resourceAlertDispatcher) *resourcePersistence {
	p := &resourcePersistence{checkpoint: resourceCheckpoint{Version: 1}, history: couchcore.ResourceHistory{Version: 1}, dispatcher: d}
	p.store = couchcore.ResourceSnapshotStore{Dir: filepath.Join(dir, "resources"), ValidateCheckpoint: validateResourceCheckpoint, ValidateHistory: couchcore.ValidateResourceHistory}
	cp, h, err := p.store.Load()
	var failures *couchcore.ResourceSnapshotLoadError
	if errors.As(err, &failures) {
		p.checkpointReadError = failures.Checkpoint
		p.historyReadError = failures.History
	}
	p.restoreCheckpoint(cp, p.checkpointReadError)
	p.restoreHistory(h, p.historyReadError)
	p.loadError = errors.Join(p.checkpointReadError, p.historyReadError)
	return p
}

func (p *resourcePersistence) restoreCheckpoint(raw []byte, err error) {
	var corrupt *couchcore.ResourceSnapshotCorruptError
	if errors.As(err, &corrupt) {
		p.continuity = append(p.continuity, "checkpoint continuity lost; old provenance discarded, fresh baselines required, prior alert deduplication unknown")
		p.checkpoint = resourceCheckpoint{Version: 1}
		p.checkpointReadError = nil
		return
	}
	p.checkpointReadError = err
	if err != nil {
		return
	}
	p.checkpoint = resourceCheckpoint{Version: 1}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p.checkpoint)
	}
	p.checkpoint.State = couchcore.ResetResourceBaselines(p.checkpoint.State)
}
func (p *resourcePersistence) restoreHistory(raw []byte, err error) {
	var corrupt *couchcore.ResourceSnapshotCorruptError
	if errors.As(err, &corrupt) {
		p.continuity = append(p.continuity, "aggregate history continuity lost; rebuilding from current observations")
		p.history = couchcore.ResourceHistory{Version: 1}
		p.historyReadError = nil
		return
	}
	p.historyReadError = err
	if err != nil {
		return
	}
	p.history = couchcore.ResourceHistory{Version: 1}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p.history)
	}
	cut := time.Now().Add(-24 * time.Hour)
	frames := p.history.Frames[:0]
	for _, f := range p.history.Frames {
		if !f.At.Before(cut) {
			frames = append(frames, f)
		}
	}
	p.history.Frames = frames
}
func (p *resourcePersistence) continuityError() error {
	if len(p.continuity) == 0 {
		return nil
	}
	return errors.New(strings.Join(p.continuity, "; "))
}

func (p *resourcePersistence) persist(ctx context.Context, state couchcore.ResourceState, view couchcore.ResourceView) (couchcore.ResourceState, couchcore.ResourceCost, error) {
	cost := couchcore.ResourceCost{}
	// A read failure grants no overwrite permission. Retry that artifact only;
	// healthy siblings and their suppression records remain in memory.
	if p.checkpointReadError != nil {
		raw, err := p.store.LoadCheckpoint()
		p.restoreCheckpoint(raw, err)
		p.loadError = errors.Join(p.checkpointReadError, p.historyReadError)
		if p.checkpointReadError != nil {
			return state, cost, fmt.Errorf("resource checkpoint unreadable; alerts paused until safe recovery: %w", p.checkpointReadError)
		}
		// The just-computed sample lacked the restored provenance/suppression state.
		// Return the recovered baseline first; the next pass can safely reduce it.
		return p.checkpoint.State, cost, errors.Join(errors.New("resource checkpoint recovered; observation interval reset"), p.continuityError())
	}
	if p.historyReadError != nil {
		raw, err := p.store.LoadHistory()
		p.restoreHistory(raw, err)
	}
	p.loadError = errors.Join(p.checkpointReadError, p.historyReadError)
	if warning := p.continuityError(); warning != nil {
		view.Errors = append(append([]string(nil), view.Errors...), warning.Error())
	}
	p.checkpoint.State = state
	save := func(cp resourceCheckpoint) error {
		raw, err := encodeResourceCheckpoint(cp)
		if err != nil {
			return err
		}
		if err = p.store.SaveCheckpoint(raw); err != nil {
			return err
		}
		cost.StorageBytes += uint64(len(raw))
		cost.StorageWrites++
		return nil
	}
	err := p.dispatcher.dispatch(ctx, &p.checkpoint, save)
	if err == nil {
		err = save(p.checkpoint)
	}
	historyErr := p.historyReadError
	if ctx.Err() == nil && historyErr == nil {
		next, e := p.history.Append(view)
		if e == nil {
			raw, e := json.Marshal(next)
			if e == nil {
				e = p.store.SaveHistory(raw)
				if e == nil {
					cost.StorageBytes += uint64(len(raw))
					cost.StorageWrites++
					p.history = next
				}
			}
			historyErr = e
		} else {
			historyErr = e
		}
	}
	var deliveryErr error
	for _, a := range p.checkpoint.Attempts {
		if a.State == "unknown" {
			deliveryErr = errors.New("alert delivery unknown; inspect resource episodes")
			break
		}
	}
	return p.checkpoint.State, cost, errors.Join(err, historyErr, deliveryErr, p.continuityError())
}

func (p *resourcePersistence) historySummary() (time.Time, int) {
	if len(p.history.Frames) == 0 {
		return time.Time{}, 0
	}
	return p.history.Frames[0].At, len(p.history.Frames)
}
