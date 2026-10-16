package couchmessage

import (
	"context"
	"testing"
	"time"
)

func TestResourceIdleReadsOnlyMemory(t *testing.T) {
	now := time.Unix(1, 0)
	b := NewBroker(context.Background(), func() time.Time { return now }, nil)
	defer b.Close()
	binding := brokerBinding("pair:1")
	if _, known := b.ResourceIdle(binding); known {
		t.Fatal("unknown binding reported")
	}
	if err := b.Register(binding, newFakeEndpoint(now)); err != nil {
		t.Fatal(err)
	}
	if pending, known := b.ResourceIdle(binding); !known || pending {
		t.Fatal("connected idle unknown")
	}
	b.mu.Lock()
	b.actors[binding].reservation = "reserved"
	b.mu.Unlock()
	if pending, known := b.ResourceIdle(binding); !known || !pending {
		t.Fatal("reservation omitted")
	}
	b.Disconnect(binding)
	if _, known := b.ResourceIdle(binding); known {
		t.Fatal("disconnected known")
	}
}
