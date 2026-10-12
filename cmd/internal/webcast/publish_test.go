package webcast

import "testing"

func TestPublisherNeverBlocksAndKeepsNewest(t *testing.T) {
	p := newPublisher(0)
	ch, cancel := p.subscribe()
	defer cancel()
	if v := <-ch; v != 0 {
		t.Fatalf("initial %d", v)
	}
	for i := 1; i <= 100; i++ {
		p.publish(i) // nobody reading: must not block
	}
	if v := <-ch; v != 100 {
		t.Fatalf("got %d, want newest 100", v)
	}
	select {
	case v := <-ch:
		t.Fatalf("stale extra value %d", v)
	default:
	}
}

func TestPublisherFinishClosesAndLateSubscribersSeeFinal(t *testing.T) {
	p := newPublisher("a")
	ch, cancel := p.subscribe()
	<-ch
	p.publish("b")
	p.finish("end")
	p.publish("after")
	if v, ok := <-ch; !ok || v != "end" {
		t.Fatalf("got %q ok=%v, want final", v, ok)
	}
	if _, ok := <-ch; ok {
		t.Fatal("channel not closed after final")
	}
	cancel() // after finish: no double close
	late, _ := p.subscribe()
	if v, ok := <-late; !ok || v != "end" {
		t.Fatalf("late got %q ok=%v", v, ok)
	}
	if _, ok := <-late; ok {
		t.Fatal("late channel open")
	}
}

func TestPublisherCancelStopsDelivery(t *testing.T) {
	p := newPublisher(0)
	ch, cancel := p.subscribe()
	<-ch
	cancel()
	cancel()
	p.publish(1)
	if _, ok := <-ch; ok {
		t.Fatal("delivered after cancel")
	}
}
