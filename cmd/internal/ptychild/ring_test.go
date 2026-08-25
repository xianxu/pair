package ptychild

import (
	"bytes"
	"testing"
)

func TestRingKeepsOnlyTheTail(t *testing.T) {
	r := NewRing(8)
	r.Append([]byte("abcdef"))
	r.Append([]byte("ghijkl"))

	if got := r.Snapshot(); !bytes.Equal(got, []byte("efghijkl")) {
		t.Fatalf("Snapshot() = %q, want %q", got, "efghijkl")
	}
}

// A single Append larger than the whole ring must keep its TAIL. Keeping the
// head would land the operator on the oldest screen the child ever drew.
func TestRingAppendLargerThanCapacityKeepsTail(t *testing.T) {
	r := NewRing(4)
	r.Append([]byte("0123456789"))

	if got := r.Snapshot(); !bytes.Equal(got, []byte("6789")) {
		t.Fatalf("Snapshot() = %q, want %q", got, "6789")
	}
}

// The switcher hands a snapshot to the repaint while the child's read pump is
// still appending. termcmd copies under its mutex for exactly this reason
// (bufferSnapshotLocked); a snapshot that aliases the ring would repaint bytes
// that arrived after the operator landed.
//
// The capacity is deliberately small enough that the next Append WRAPS. A
// version of this test with spare capacity passed against `return r.data` --
// append wrote past the snapshot's bytes instead of over them, so it asserted
// nothing. Overwriting in place is what the trim's copy() does, and it is the
// only aliasing that can actually corrupt a repaint.
func TestRingSnapshotDoesNotAliasTheRing(t *testing.T) {
	r := NewRing(8)
	r.Append([]byte("abcdefgh"))

	snap := r.Snapshot()
	r.Append([]byte("12345678"))

	if !bytes.Equal(snap, []byte("abcdefgh")) {
		t.Fatalf("snapshot mutated by a later Append: %q", snap)
	}
}

func TestRingUnderCapacityKeepsEverything(t *testing.T) {
	r := NewRing(64)
	r.Append([]byte("a"))
	r.Append([]byte("b"))

	if got := r.Snapshot(); !bytes.Equal(got, []byte("ab")) {
		t.Fatalf("Snapshot() = %q, want %q", got, "ab")
	}
}

// A ring that grew its backing array without bound would still pass every
// assertion above, because Snapshot only reports the window.
//
// Scope note (BR-4): this pins BOUNDEDNESS, which both the copy and the
// re-slice satisfy. It is not a discriminator between them, and the code
// comment no longer claims it is.
func TestRingDoesNotGrowWithoutBound(t *testing.T) {
	r := NewRing(32)
	for i := 0; i < 2000; i++ {
		r.Append([]byte("0123456789abcdef"))
	}
	if got := len(r.Snapshot()); got != 32 {
		t.Fatalf("Snapshot() length = %d, want 32", got)
	}
	if got := r.Allocated(); got > 4*32 {
		t.Fatalf("ring retains %d bytes for a 32-byte window", got)
	}
}
