package couchcore

import (
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"time"
)

// ResourceFrame retains aggregates and offenders, never full process histories.
type ResourceFrame struct {
	At           time.Time             `json:"at"`
	Slots        []ResourceSlotView    `json:"slots"`
	Orphans      []ResourceProcessView `json:"orphans"`
	Unattributed []ResourceProcessView `json:"unattributed"`
	// OrphanNvim keeps the rows the sweep named, which the view suppresses from
	// Unattributed; without them the frame would forget the offender (#583).
	OrphanNvim []OrphanNvim `json:"orphan_nvim"`
	Errors     []string     `json:"errors"`
	Cost       ResourceCost `json:"cost"`
}
type ResourceHistory struct {
	Version int             `json:"version"`
	Frames  []ResourceFrame `json:"frames"`
}

func (h ResourceHistory) Append(v ResourceView) (ResourceHistory, error) {
	if v.SampledAt.IsZero() {
		return h, errors.New("resource history requires sample time")
	}
	if len(h.Frames) > 0 && !v.SampledAt.After(h.Frames[len(h.Frames)-1].At) {
		return h, errors.New("resource history clock regressed")
	}
	next := ResourceHistory{Version: 1}
	for _, f := range h.Frames {
		if v.SampledAt.Sub(f.At) < 24*time.Hour {
			next.Frames = append(next.Frames, f)
		}
	}
	next.Frames = append(next.Frames, ResourceFrame{v.SampledAt, v.Slots, v.Orphans, v.Unattributed, v.OrphanNvim, v.Errors, v.Cost})
	if len(next.Frames) > 1440 {
		next.Frames = next.Frames[len(next.Frames)-1440:]
	}
	// Encode each frame once so pruning is linear rather than repeated 4MiB encodes.
	encoded := make([]int, len(next.Frames))
	size := len(`{"version":1,"frames":[]}`)
	for i, f := range next.Frames {
		raw, err := json.Marshal(f)
		if err != nil {
			return h, err
		}
		encoded[i] = len(raw) + 1
		size += encoded[i]
	}
	for size > ResourceHistoryLimit && len(next.Frames) > 1 {
		size -= encoded[0]
		encoded = encoded[1:]
		next.Frames = next.Frames[1:]
	}
	if size > ResourceHistoryLimit {
		return h, errors.New("resource frame exceeds history byte limit")
	}
	return next, nil
}

func ValidateResourceHistory(raw []byte) error {
	if len(raw) > ResourceHistoryLimit {
		return errors.New("resource history exceeds limit")
	}
	var h ResourceHistory
	if err := strictjson.Decode(raw, &h); err != nil {
		return err
	}
	if h.Version != 1 || len(h.Frames) > 1440 {
		return errors.New("invalid resource history schema/count")
	}
	var last time.Time
	for _, f := range h.Frames {
		if !frameWithinBounds(f) || !last.IsZero() && !f.At.After(last) {
			return errors.New("invalid resource history frame")
		}
		last = f.At
	}
	return nil
}

func frameWithinBounds(f ResourceFrame) bool {
	return !f.At.IsZero() &&
		len(f.Slots) <= ResourceMaxSlots &&
		len(f.Orphans) <= ResourceMaxProcesses &&
		len(f.Unattributed) <= ResourceMaxProcesses &&
		len(f.OrphanNvim) <= ResourceMaxProcesses
}
