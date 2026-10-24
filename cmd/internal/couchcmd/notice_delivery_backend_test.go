package couchcmd

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestNoticeOwnerUsesExactHubAndMemoryOnly(t *testing.T) {
	r, slots := hubMessageRig(t)
	before := r.world.counts()
	o := couchcore.NoticeDeliveryOccurrence{Thread: bindingThread(slots[2])}
	owner := r.s.NoticeOwner(o)
	if owner.State != couchcore.NoticeOwnerEnabled || owner.Binding != lifecycleBindingKey(slots[1]) {
		t.Fatalf("wrong owning hub: %+v", owner)
	}
	if got := r.world.counts(); got != before {
		t.Fatalf("owner lookup performed discovery: before=%v after=%v", before, got)
	}
	if _, err := r.s.controlAutomation("/repo1", "off"); err != nil {
		t.Fatal(err)
	}
	if got := r.s.NoticeOwner(o); got.State != couchcore.NoticeOwnerDisabled {
		t.Fatalf("disabled hub treated absent: %+v", got)
	}
	r.s.broker.Disconnect(slots[1])
	if got := r.s.NoticeOwner(o); got.State != couchcore.NoticeOwnerUnavailable {
		t.Fatalf("substituted another live hub: %+v", got)
	}
}

func TestNoticeDeliveryRejectsChangedAutomationEpoch(t *testing.T) {
	r, slots := hubMessageRig(t)
	o := couchcore.NoticeDeliveryOccurrence{ID: "notice-epoch", Thread: bindingThread(slots[2]), Body: "diagnostic"}
	owner := r.s.NoticeOwner(o)
	for _, action := range []string{"off", "on"} {
		if _, err := r.s.controlAutomation("/repo1", action); err != nil {
			t.Fatal(err)
		}
	}
	if got := r.s.DeliverNotice(context.Background(), couchcore.NoticeDeliveryEffect{Occurrence: o, Owner: owner, AdmissionID: o.ID}); got != couchcore.NoticeDeliveryRefused {
		t.Fatalf("stale epoch admitted: %v", got)
	}
	if _, known, _ := r.s.broker.ReceiptSnapshot(o.ID); known {
		t.Fatal("stale effect created broker message")
	}
}

func TestNoticeOwnerDefersRegistryConnectUntilIdentityIsCommitted(t *testing.T) {
	r, slots := hubMessageRig(t)
	source := slots[2]
	r.s.mu.Lock()
	identity := r.s.workspaces[source]
	delete(r.s.workspaces, source)
	r.s.mu.Unlock()
	occurrence := couchcore.NoticeDeliveryOccurrence{Thread: bindingThread(source)}
	if got := r.s.NoticeOwner(occurrence); got.State != couchcore.NoticeOwnerBooting {
		t.Fatalf("connected source with pending identity treated absent: %+v", got)
	}
	r.s.mu.Lock()
	r.s.workspaces[source] = identity
	r.s.mu.Unlock()
	if got := r.s.NoticeOwner(occurrence); got.State != couchcore.NoticeOwnerEnabled {
		t.Fatalf("committed identity did not resolve owner: %+v", got)
	}
}
