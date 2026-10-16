package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type resourcePersistence struct {
	store      couchcore.ResourceSnapshotStore
	checkpoint resourceCheckpoint
	history    couchcore.ResourceHistory
	dispatcher resourceAlertDispatcher
	loadError  error
}

func newResourcePersistence(dir string, d resourceAlertDispatcher) *resourcePersistence {
	p := &resourcePersistence{checkpoint: resourceCheckpoint{Version: 1}, history: couchcore.ResourceHistory{Version: 1}, dispatcher: d}
	p.store = couchcore.ResourceSnapshotStore{Dir: filepath.Join(dir, "resources"), ValidateCheckpoint: validateResourceCheckpoint, ValidateHistory: couchcore.ValidateResourceHistory}
	cp, h, err := p.store.Load()
	p.loadError = err
	if err == nil {
		if len(cp) > 0 {
			_ = json.Unmarshal(cp, &p.checkpoint)
			p.checkpoint.State = couchcore.ResetResourceBaselines(p.checkpoint.State)
		}
		if len(h) > 0 {
			_ = json.Unmarshal(h, &p.history)
		}
		// Load prunes expired aggregate history; it never supplies rate baselines.
		cut := time.Now().Add(-24 * time.Hour)
		frames := p.history.Frames[:0]
		for _, f := range p.history.Frames {
			if !f.At.Before(cut) {
				frames = append(frames, f)
			}
		}
		p.history.Frames = frames
	}
	return p
}
func (p *resourcePersistence) persist(ctx context.Context, state couchcore.ResourceState, view couchcore.ResourceView) (couchcore.ResourceState, couchcore.ResourceCost, error) {
	cost := couchcore.ResourceCost{}
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
	// Fail closed on a lost checkpoint: current measurements can still be
	// displayed, but no restart deduplication or old provenance is fabricated.
	if p.loadError != nil {
		return state, cost, fmt.Errorf("resource history/checkpoint unavailable; alerts disabled: %w", p.loadError)
	}
	err := p.dispatcher.dispatch(ctx, &p.checkpoint, save)
	if err == nil {
		err = save(p.checkpoint)
	}
	var historyErr error
	if ctx.Err() == nil {
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
	return p.checkpoint.State, cost, errors.Join(err, historyErr, deliveryErr)
}

func (p *resourcePersistence) historySummary() (time.Time, int) {
	if len(p.history.Frames) == 0 {
		return time.Time{}, 0
	}
	return p.history.Frames[0].At, len(p.history.Frames)
}
