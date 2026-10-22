package couchcmd

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

type documentDeliveryBarrier struct {
	serviceEndpointFake
	entered chan couchmessage.Message
	release chan struct{}
}

func (e *documentDeliveryBarrier) Deliver(ctx context.Context, m couchmessage.Message, _ func(couchmessage.Receipt)) (couchmessage.Receipt, error) {
	select {
	case e.entered <- m:
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
	select {
	case <-e.release:
		return couchmessage.Receipt{Status: couchmessage.Submitted}, nil
	case <-ctx.Done():
		return couchmessage.Receipt{}, ctx.Err()
	}
}

func TestQueuedSubmitRemainsOwedAfterRenameAndOldPathReuse(t *testing.T) {
	base := time.Now().UTC()
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	now := func() time.Time { return time.Unix(0, nanos.Load()).UTC() }
	barrier := &documentDeliveryBarrier{entered: make(chan couchmessage.Message, 2), release: make(chan struct{})}
	r := newServiceRig(t, func(a messageAuthority) messageAuthority {
		a.now = now
		a.endpoint = func(couchmessage.Binding) couchmessage.DeliveryEndpoint { return barrier }
		return a
	})
	store := r.s.originNamespace
	r.s.recordDeliveries(store)
	r.s.deliveries.Load().now = now
	from, to := r.connect(1), r.connect(2)
	s := sessionsync.New(store)
	s.Now = now
	old := filepath.Join(t.TempDir(), "old.md")
	newPath := filepath.Join(filepath.Dir(old), "new.md")
	if err := os.WriteFile(old, []byte("## Work\n\noriginal\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original, err := s.Register(old, to.Slot)
	if err != nil {
		t.Fatal(err)
	}
	send := func(id, body string) couchmessage.Response {
		t.Helper()
		req := callerRequest(from, "send")
		req.ID = id
		req.Target = to.Slot
		req.Body = body
		response := r.s.handle(context.Background(), req)
		if response.Code != "ok" && response.Code != "accepted" {
			t.Fatalf("send %+v", response)
		}
		return response
	}
	nanos.Store(base.Add(time.Second).UnixNano())
	send("blocking-delivery", "submitted: /unrelated-document.md")
	select {
	case <-barrier.entered:
	case <-testwait.After(t):
		t.Fatal("first delivery did not enter")
	}
	nanos.Store(base.Add(2 * time.Second).UnixNano())
	queued := send("old-path-submit", sessionsync.SubmitPrefix+old)
	if queued.Receipt == nil || queued.Receipt.Status != couchmessage.Queued {
		t.Fatalf("submit was not queued: %+v", queued)
	}
	nanos.Store(base.Add(3 * time.Second).UnixNano())
	if err := os.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Rename(old, newPath); err != nil {
		t.Fatal(err)
	}
	nanos.Store(base.Add(4 * time.Second).UnixNano())
	if err := os.WriteFile(old, []byte("## Work\n\nreplacement\n"), 0600); err != nil {
		t.Fatal(err)
	}
	replacement, err := s.Register(old, to.Slot)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Registration.ID == original.Registration.ID {
		t.Fatal("old-path reuse inherited document identity")
	}
	nanos.Store(base.Add(5 * time.Second).UnixNano())
	barrier.release <- struct{}{}
	select {
	case message := <-barrier.entered:
		if message.ID != "old-path-submit" {
			t.Fatalf("unexpected delivery %+v", message)
		}
	case <-testwait.After(t):
		t.Fatal("queued submit did not promote")
	}
	barrier.release <- struct{}{}
	ledger := &sessionsync.Ledger{Dir: sessionsync.LedgerDir(store)}
	testwait.Until(t, "late submit recorded", func() bool {
		rows, err := ledger.Read(to.Slot)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.ID == "old-path-submit" {
				return true
			}
		}
		return false
	})
	moved, err := s.Status(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Owed) != 1 || moved.Owed[0].ID != "old-path-submit" {
		t.Fatalf("renamed document lost delayed submit: %+v", moved)
	}
	reused, err := s.Status(old)
	if err != nil {
		t.Fatal(err)
	}
	if len(reused.Owed) != 0 {
		t.Fatalf("replacement inherited old submit: %+v", reused)
	}
}
