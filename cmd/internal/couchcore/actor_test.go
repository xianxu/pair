package couchcore

import (
	"sync"
	"testing"
	"time"
)

// collect drives an actor and returns the kinds handled, in order, once n have
// arrived or the deadline passes.
func collect(t *testing.T, a *Actor, n int) []string {
	t.Helper()
	var mu sync.Mutex
	var got []string
	done := make(chan struct{})
	a.OnMessage = func(m Message) {
		mu.Lock()
		got = append(got, m.Kind)
		if len(got) == n {
			close(done)
		}
		mu.Unlock()
	}
	go a.Loop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("actor did not handle the expected messages")
	}
	a.Close()
	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), got...)
}

func TestActorDrainsControlBeforeNormal(t *testing.T) {
	// Distinct kinds for the normal messages on purpose: Enqueue collapses by
	// Kind, so three identical ones would become one and the expected count
	// would never arrive. A fixture that fights the policy it sits on top of
	// deadlocks rather than fails.
	a := NewActor(ActorRecord{ID: "couch-a"}, 8)
	a.Send(Message{Kind: "note-1"})
	a.Send(Message{Kind: "note-2"})
	a.Send(Message{Kind: "note-3"})
	a.Send(Message{Kind: "stop", Control: true})

	got := collect(t, a, 4)
	if got[0] != "stop" {
		t.Fatalf("order = %v, want the control message first", got)
	}
}

func TestActorCollapsesRepeatedKindsBeforeDelivery(t *testing.T) {
	// This is what makes the mailbox policy load-bearing rather than
	// decorative: a plain append would deliver "status" three times.
	a := NewActor(ActorRecord{ID: "couch-a"}, 8)
	a.Send(Message{Kind: "status"})
	a.Send(Message{Kind: "status"})
	a.Send(Message{Kind: "status"})
	a.Send(Message{Kind: "sentinel"})

	got := collect(t, a, 2)
	if len(got) != 2 {
		t.Fatalf("handled %v, want the three status requests collapsed to one", got)
	}
}

func TestActorReportsDroppedMessagesLoudly(t *testing.T) {
	// A full mailbox is a bug signal, not flow control: Send must say so.
	a := NewActor(ActorRecord{ID: "couch-a"}, 2)
	var lost []Message
	a.OnDropped = func(m Message) { lost = append(lost, m) }

	if ok := a.Send(Message{Kind: "n1"}); !ok {
		t.Fatal("first send should be clean")
	}
	if ok := a.Send(Message{Kind: "n2"}); !ok {
		t.Fatal("second send should be clean")
	}
	if ok := a.Send(Message{Kind: "n3"}); ok {
		t.Fatal("a send that overflows must report false")
	}
	if len(lost) != 1 || lost[0].Kind != "n1" {
		t.Fatalf("lost = %+v, want the oldest entry named", lost)
	}
}

func TestActorSendAfterCloseIsRejectedNotPanicking(t *testing.T) {
	a := NewActor(ActorRecord{ID: "couch-a"}, 4)
	go a.Loop()
	a.Close()
	if ok := a.Send(Message{Kind: "late"}); ok {
		t.Fatal("a send after Close must be refused")
	}
}

func TestActorLoopExitsOnClose(t *testing.T) {
	a := NewActor(ActorRecord{ID: "couch-a"}, 4)
	exited := make(chan struct{})
	go func() { a.Loop(); close(exited) }()
	a.Close()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("Loop did not return after Close")
	}
}

func TestActorQueryIsADirectCallNotAMessage(t *testing.T) {
	// Go shares memory, so a read does not need to round-trip through the
	// mailbox. Message passing is for ordering and decoupling, not fidelity
	// to Erlang.
	a := NewActor(ActorRecord{ID: "couch-a"}, 4)
	a.Send(Message{Kind: "n1"})
	if got := a.QueueLen(); got != 1 {
		t.Fatalf("QueueLen = %d, want 1", got)
	}
	a.Close()
}
