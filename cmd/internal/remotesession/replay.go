package remotesession

import (
	"bytes"
	"crypto/sha256"
	"errors"
)

type replayKind uint8

const (
	replayNew replayKind = iota
	replayPending
	replayCached
)

type replayAdmission struct {
	kind      replayKind
	result    []byte
	uncertain bool
}
type replayEntry struct {
	digest    [32]byte
	result    []byte
	complete  bool
	uncertain bool
}

// Replay belongs to one session under its owner's admission lock. The high
// water mark survives result eviction, so an expired request can never be new.
// In-flight and uncertain entries are pinned; overflow refuses new effects.
type Replay struct {
	high    uint64
	entries map[uint64]replayEntry
	order   []uint64
}

func (r *Replay) Admit(sequence uint64, payload []byte) (replayAdmission, error) {
	if sequence == 0 || len(payload) > 64*1024 {
		return replayAdmission{}, errors.New("invalid remote request sequence or size")
	}
	digest := sha256.Sum256(payload)
	if entry, exists := r.entries[sequence]; exists {
		if entry.digest != digest {
			return replayAdmission{}, errors.New("remote sequence payload conflict")
		}
		if !entry.complete {
			return replayAdmission{kind: replayPending}, nil
		}
		return replayAdmission{kind: replayCached, result: bytes.Clone(entry.result), uncertain: entry.uncertain}, nil
	}
	if sequence <= r.high {
		return replayAdmission{}, errors.New("remote result expired; inspect before continuing")
	}
	if len(r.entries) >= 256 {
		victim := -1
		for i, seq := range r.order {
			e := r.entries[seq]
			if e.complete && !e.uncertain {
				victim = i
				break
			}
		}
		if victim < 0 {
			return replayAdmission{}, errors.New("remote uncertain request capacity; reconcile before continuing")
		}
		delete(r.entries, r.order[victim])
		r.order = append(r.order[:victim], r.order[victim+1:]...)
	}
	if r.entries == nil {
		r.entries = make(map[uint64]replayEntry)
	}
	r.high = sequence
	r.entries[sequence] = replayEntry{digest: digest}
	r.order = append(r.order, sequence)
	return replayAdmission{kind: replayNew}, nil
}
func (r *Replay) Complete(sequence uint64, payload, result []byte, uncertain bool) error {
	if len(payload) > 64*1024 || len(result) > 1024 {
		return errors.New("remote replay result exceeds bound")
	}
	entry, exists := r.entries[sequence]
	if !exists || entry.digest != sha256.Sum256(payload) {
		return errors.New("remote replay completion identity mismatch")
	}
	if entry.complete && !entry.uncertain {
		if !uncertain && bytes.Equal(entry.result, result) {
			return nil
		}
		return errors.New("confirmed remote result is immutable")
	}
	entry.complete = true
	entry.uncertain = uncertain
	entry.result = bytes.Clone(result)
	r.entries[sequence] = entry
	return nil
}
