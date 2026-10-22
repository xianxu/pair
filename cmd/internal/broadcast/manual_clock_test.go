package broadcast

import (
	"testing"
	"testing/synctest"
	"time"
)

// Drive the helper's request and timer channels independently, forcing the
// ordering that a seeded event list cannot control in Hub.run's select.
func TestManualClockWaitsForTimerConsumption(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := &Hub{reqs: make(chan func()), done: make(chan struct{})}
		defer close(h.done)
		clock := newManualClock()
		timer, _ := clock.after(time.Second)
		h.pointerGraceC = timer
		returned := make(chan struct{})
		go func() { clock.fireGrace(h); close(returned) }()
		// A request may win the select before the timer. Completing it must not
		// let fireGrace claim the timer has been consumed.
		(<-h.reqs)()
		synctest.Wait()
		select {
		case <-returned:
			t.Fatal("fireGrace returned after a request, before timer consumption")
		default:
		}
		<-timer
		h.pointerGraceC = nil
		synctest.Wait()
		select {
		case <-returned:
			t.Fatal("fireGrace returned before the timer handler completion barrier")
		default:
		}
		(<-h.reqs)()
		synctest.Wait()
		select {
		case <-returned:
		default:
			t.Fatal("fireGrace did not return after timer consumption and completion")
		}
	})
}

func TestManualClockEndDuringDelivery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := &Hub{reqs: make(chan func()), done: make(chan struct{})}
		clock := newManualClock()
		timer, _ := clock.after(time.Second)
		h.graceC = timer
		returned := make(chan struct{})
		go func() { clock.fireGrace(h); close(returned) }()
		(<-h.reqs)()
		synctest.Wait() // delivery is pending, with no receiver
		close(h.done)
		synctest.Wait()
		select {
		case <-returned:
		default:
			t.Fatal("ended hub left a timer delivery blocked")
		}
	})
}
