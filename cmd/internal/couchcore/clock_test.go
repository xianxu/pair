package couchcore

import (
	"strings"
	"testing"
	"time"
)

func TestFixedClockReturnsItsTime(t *testing.T) {
	want := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	if got := (FixedClock{T: want}).Now(); !got.Equal(want) {
		t.Fatalf("Now = %v", got)
	}
}

func TestFixedIDGenAdvances(t *testing.T) {
	g := NewFixedIDGen("ah8d", "b2c1")
	if got := g.NewID(); got != ActorID("couch-ah8d") {
		t.Fatalf("first = %q", got)
	}
	if got := g.NewID(); got != ActorID("couch-b2c1") {
		t.Fatalf("second = %q; the generator must advance", got)
	}
}

func TestRandomIDGenShapeAndUniqueness(t *testing.T) {
	g := NewRandomIDGen()
	a, b := g.NewID(), g.NewID()
	if !strings.HasPrefix(string(a), "couch-") || len(a) != len("couch-")+8 {
		t.Fatalf("id = %q, want couch- plus 8 hex", a)
	}
	if a == b {
		t.Fatal("two ids must differ")
	}
}
