package couchcore

import "sync"

// Actor is one agent's in-process half: identity, a bounded mailbox, and a
// loop that drains it.
//
// It is a mutex-guarded queue rather than the two channels with a priority
// select the issue's Spec suggested. Recorded as a deviation, with a reason:
// the bounded/collapse policy has to be applied AT INSERTION, and a buffered
// channel cannot collapse a duplicate already sitting in it. Putting Enqueue
// behind the mutex keeps the whole decision pure and testable without
// goroutines; the channel version would have pushed that logic into the
// receive loop where it is much harder to test.
//
// Queries are direct calls behind the same mutex, not messages. Go shares
// memory; message passing here is for ordering and decoupling, not fidelity to
// Erlang.
type Actor struct {
	Record ActorRecord

	// OnMessage handles one message. Set before Loop starts.
	OnMessage func(Message)
	// OnDropped is the loud half of a bounded mailbox: a full mailbox is a
	// bug signal, not flow control, so what was lost gets named.
	OnDropped func(Message)

	capacity int

	mu     sync.Mutex
	cond   *sync.Cond
	queue  []Message
	closed bool
}

func NewActor(rec ActorRecord, capacity int) *Actor {
	a := &Actor{Record: rec, capacity: capacity}
	a.cond = sync.NewCond(&a.mu)
	return a
}

// Send enqueues a message. It never blocks: a send is a non-blocking attempt
// that may fail, and failure is the sender's problem -- which is the same
// mechanism as a delivery deadline, not a second one.
//
// Returns false if the message was refused or something was dropped to make
// room.
func (a *Actor) Send(m Message) bool {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return false
	}
	next, dropped, ok := Enqueue(a.queue, m, a.capacity)
	a.queue = next
	onDropped := a.OnDropped
	a.mu.Unlock()
	a.cond.Signal()

	if !ok && dropped.Kind != "" && onDropped != nil {
		onDropped(dropped)
	}
	return ok
}

// QueueLen reports the current depth. Mailbox depth is one of the two
// staleness signals the advisor reads: it says somebody is waiting on this
// agent, where git staleness says the thread has gone cold.
func (a *Actor) QueueLen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.queue)
}

// Loop drains the mailbox until Close. Control messages are taken first, so a
// stop or a deadline is not stuck behind a backlog of notes.
func (a *Actor) Loop() {
	for {
		a.mu.Lock()
		for len(a.queue) == 0 && !a.closed {
			a.cond.Wait()
		}
		if len(a.queue) == 0 && a.closed {
			a.mu.Unlock()
			return
		}
		m := a.takeLocked()
		handler := a.OnMessage
		a.mu.Unlock()

		if handler != nil {
			handler(m)
		}
	}
}

// takeLocked pops the highest-priority message: the first Control entry if any,
// otherwise the oldest.
func (a *Actor) takeLocked() Message {
	idx := 0
	for i, m := range a.queue {
		if m.Control {
			idx = i
			break
		}
	}
	m := a.queue[idx]
	a.queue = append(a.queue[:idx:idx], a.queue[idx+1:]...)
	return m
}

// Close stops the loop once the queue drains and refuses further sends.
func (a *Actor) Close() {
	a.mu.Lock()
	a.closed = true
	a.mu.Unlock()
	a.cond.Broadcast()
}
