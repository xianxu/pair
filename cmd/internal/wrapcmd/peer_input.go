package wrapcmd

import "time"

// admitStdin observes one stdin chunk under the admission lock that fences
// automatic writes. released is a chunk the peer hold queued earlier
// (pair#428); it is admitted now, just before it is forwarded.
func (p *proxy) admitStdin(data []byte, released bool) (human, held bool) {
	p.inputAdmission.Lock()
	defer p.inputAdmission.Unlock()
	if p.peer != nil {
		if released {
			human = p.peer.admitReleased(data)
		} else {
			human, held = p.peer.admitInput(data)
		}
	}
	if p.orientation != nil && !held {
		p.orientation.admitOperatorInput(data)
	}
	return human, held
}

// heldInput is the input writer's queue of chunks held behind a delivery.
// Chunk i replays at base + (its arrival - the first arrival), so a draft
// send's paste, settle and submit keep their spacing. It lives in memory
// only, and is bounded by PeerDraftHold plus the replayed gaps.
type heldInput struct {
	chunks      []heldChunk
	base, first time.Time
	timer       *time.Timer
}

type heldChunk struct {
	data []byte
	at   time.Time
}

func (h *heldInput) add(data []byte) {
	h.chunks = append(h.chunks, heldChunk{data: data, at: time.Now()})
}

// next returns the first held chunk once the delivery released the hold and
// the chunk's replay time came. With the queue drained, it ends the hold.
func (h *heldInput) next(d *peerDelivery) ([]byte, bool) {
	if d == nil {
		return nil, false
	}
	if len(h.chunks) == 0 {
		if !h.base.IsZero() {
			h.base = time.Time{}
			d.endHold()
		}
		return nil, false
	}
	if h.base.IsZero() {
		if !d.releaseHold() {
			return nil, false
		}
		h.base, h.first = time.Now(), h.chunks[0].at
	}
	if wait := time.Until(h.base.Add(h.chunks[0].at.Sub(h.first))); wait > 0 {
		h.arm(wait)
		return nil, false
	}
	c := h.chunks[0]
	h.chunks = h.chunks[1:]
	return c.data, true
}

// drain hands back everything still held, for a writer that is exiting.
func (h *heldInput) drain() []byte {
	var rest []byte
	for _, c := range h.chunks {
		rest = append(rest, c.data...)
	}
	h.chunks = nil
	return rest
}

func (h *heldInput) arm(wait time.Duration) {
	if h.timer == nil {
		h.timer = time.NewTimer(wait)
		return
	}
	if !h.timer.Stop() {
		select {
		case <-h.timer.C:
		default:
		}
	}
	h.timer.Reset(wait)
}

func (h *heldInput) timerC() <-chan time.Time {
	if h.timer == nil {
		return nil
	}
	return h.timer.C
}

func (h *heldInput) stop() {
	if h.timer != nil {
		h.timer.Stop()
	}
}
