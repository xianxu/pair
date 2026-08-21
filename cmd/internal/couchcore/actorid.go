package couchcore

import (
	"crypto/rand"
	"encoding/hex"
)

// ActorID identifies an incarnation of an actor, not an address. Worktree is
// the address (Erlang's registered name); ActorID is the pid. The distinction
// earns its keep in #147, where a reply referencing a dead incarnation must be
// droppable rather than misdelivered to its successor.
type ActorID string

type IDGen interface{ NewID() ActorID }

type randomIDGen struct{}

func NewRandomIDGen() IDGen { return randomIDGen{} }

func (randomIDGen) NewID() ActorID {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ActorID("couch-00000000")
	}
	return ActorID("couch-" + hex.EncodeToString(b[:]))
}

// FixedIDGen yields a scripted sequence so registry assertions are
// deterministic.
type FixedIDGen struct {
	seq []string
	i   int
}

var _ IDGen = (*FixedIDGen)(nil)

func NewFixedIDGen(seq ...string) *FixedIDGen { return &FixedIDGen{seq: seq} }

func (f *FixedIDGen) NewID() ActorID {
	if f.i >= len(f.seq) {
		return ActorID("couch-exhausted")
	}
	id := ActorID("couch-" + f.seq[f.i])
	f.i++
	return id
}
