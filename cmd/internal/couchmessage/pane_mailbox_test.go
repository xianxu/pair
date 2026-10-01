package couchmessage

import (
	"fmt"
	"testing"
)

func TestPaneMailboxCoalescesToLatestWithoutBlocking(t *testing.T) {
	m := NewPaneMailbox()
	a, b := ThreadKey{"s", "a"}, ThreadKey{"s", "b"}
	// Far more posts than any buffer: none may block.
	for i := 0; i < 10000; i++ {
		m.Post(a, PaneHandle(fmt.Sprint(i)))
	}
	m.Post(b, "h1")
	m.Post(b, "")
	<-m.Wake()
	got := m.Drain()
	if len(got) != 2 || got[a] != "9999" || got[b] != "" {
		t.Fatalf("drain %v", got)
	}
	if _, ok := got[b]; !ok {
		t.Fatal("a departure was dropped")
	}
	if again := m.Drain(); len(again) != 0 {
		t.Fatalf("second drain %v", again)
	}
}
