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

// History is best-effort and never an input to process ownership, baselines or
// alert admission. Only this sampler-owned adapter writes the aggregate ring.
type resourceHistoryPersistence struct {
	store     couchcore.ResourceHistoryStore
	history   couchcore.ResourceHistory
	loadError error
}

func newResourceHistoryPersistence(dir string) *resourceHistoryPersistence {
	p := &resourceHistoryPersistence{store: couchcore.ResourceHistoryStore{Dir: filepath.Join(dir, "resources")}, history: couchcore.ResourceHistory{Version: 1}}
	raw, err := p.store.Load()
	p.loadError = err
	if err == nil && len(raw) > 0 {
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
	return p
}
func (p *resourceHistoryPersistence) persist(ctx context.Context, view couchcore.ResourceView) (couchcore.ResourceCost, error) {
	cost := couchcore.ResourceCost{}
	var continuity error
	if p.loadError != nil {
		continuity = fmt.Errorf("aggregate history continuity lost; started a fresh memory ring: %w", p.loadError)
		view.Errors = append(append([]string(nil), view.Errors...), continuity.Error())
	}
	next, err := p.history.Append(view)
	if err != nil {
		return cost, errors.Join(continuity, err)
	}
	// Save failure never rolls back observed history or gates the alert worker.
	p.history = next
	if err := ctx.Err(); err != nil {
		return cost, errors.Join(continuity, err)
	}
	raw, err := json.Marshal(next)
	if err == nil {
		err = p.store.Save(raw)
	}
	if err == nil {
		cost.StorageBytes = uint64(len(raw))
		cost.StorageWrites = 1
	}
	return cost, errors.Join(continuity, err)
}
func (p *resourceHistoryPersistence) historySummary() (time.Time, int) {
	if len(p.history.Frames) == 0 {
		return time.Time{}, 0
	}
	return p.history.Frames[0].At, len(p.history.Frames)
}
