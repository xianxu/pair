// Package ptychild owns one child process on a pty: the bytes it writes, the
// window of those bytes kept for a repaint, and what its output says about the
// screen it thinks it is drawing on.
//
// It is the child half of terminal plumbing that `pair term` and `couch` share.
// termcmd's multiplexer already had all of this inline (pty-backed tabs, a
// 128KB replay buffer, redraw-from-snapshot on switch); couch would have been
// the second copy. What deliberately stays with each caller is POLICY -- which
// child is active, what a switch does, what happens when one exits -- the same
// structure/policy split cmd/internal/ansi documents for escape sequences.
package ptychild

// DefaultRingBytes is how much of a child's output is kept so that landing on
// it is not a blank screen. Carried over from termcmd's per-tab buffer, which
// has driven `pair term`'s tab switching at this size since #127.
const DefaultRingBytes = 128 * 1024

// Ring is a bounded window over the tail of a byte stream.
//
// Bounded is the whole point: a child that streams for hours must cost a fixed
// amount of memory, and only the tail can repaint a screen -- the head is a
// screen nobody will ever see again.
type Ring struct {
	capacity int
	data     []byte
}

func NewRing(capacity int) *Ring {
	if capacity < 0 {
		capacity = 0
	}
	return &Ring{capacity: capacity}
}

// Append adds p and drops whatever no longer fits, oldest first.
func (r *Ring) Append(p []byte) {
	if r.capacity == 0 || len(p) == 0 {
		return
	}
	// A write bigger than the window can skip the copy entirely: everything
	// already held is about to be dropped.
	if len(p) >= r.capacity {
		r.data = append(r.data[:0], p[len(p)-r.capacity:]...)
		return
	}
	r.data = append(r.data, p...)
	if len(r.data) > r.capacity {
		// copy() rather than re-slicing, and this is a CLARITY choice, not a
		// bug fix -- a distinction the M1 boundary review had to correct (BR-4).
		// An earlier comment here claimed `r.data = r.data[n:]` grew without
		// bound; measured over 2000 appends into a 32-byte ring, re-slicing
		// peaks at cap=48 and copying sits at cap=64, because re-slicing
		// monotonically shrinks the remaining capacity and so guarantees the
		// next append reallocates. Both are bounded. What copying buys is that
		// the window always starts at the head of the backing array, so the
		// allocation is flat and predictable instead of sawtoothing.
		r.data = r.data[:copy(r.data, r.data[len(r.data)-r.capacity:])]
	}
}

// Snapshot returns an independent copy, so a repaint can hold it while the
// read pump keeps appending.
func (r *Ring) Snapshot() []byte {
	return append([]byte(nil), r.data...)
}

func (r *Ring) Len() int { return len(r.data) }

// Allocated reports the backing array's size.
//
// It pins that memory stays bounded, which Snapshot cannot show -- Snapshot
// reports the window, so a genuinely unbounded ring looks identical from
// outside. It does NOT discriminate copy from re-slice; both pass, and claiming
// otherwise is what BR-4 corrected.
func (r *Ring) Allocated() int { return cap(r.data) }
