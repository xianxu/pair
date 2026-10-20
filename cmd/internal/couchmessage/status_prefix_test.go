package couchmessage

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestBrokerStatusPrefixesRespectVisibilityAndExactIDs(t *testing.T) {
	now := time.Now()
	b := NewBroker(context.Background(), nil, nil)
	defer b.Close()
	caller, peer, outsider := brokerBinding("ops:0"), brokerBinding("pair:1"), brokerBinding("pair:2")
	for _, binding := range []Binding{caller, peer, outsider} {
		_ = b.Register(binding, newFakeEndpoint(now))
	}
	add := func(id string, from, to Binding) {
		b.mu.Lock()
		b.receipts[id] = Receipt{Message: Message{ID: id, From: from, To: to}, Status: Submitted, RetainUntil: now.Add(time.Hour)}
		b.mu.Unlock()
	}
	add("abcdef00-second", peer, caller)
	add("abcdef00-first", caller, peer)
	add("abcdef00-hidden", peer, outsider)
	request := Request{Op: "status", Scope: caller.Scope, Tag: caller.Tag, Session: caller.Session, Nonce: caller.Nonce, ID: "abcdef00"}
	response := Handle(context.Background(), b, request)
	if response.Code != "refused" || !strings.Contains(response.Error, "ambiguous") || !strings.Contains(response.Error, "abcdef00-first, abcdef00-second") || strings.Contains(response.Error, "hidden") {
		t.Errorf("ambiguous visible IDs: %+v", response)
	}
	add("abcdef00", caller, peer)
	if receipt, err := b.StatusContext(context.Background(), caller, "abcdef00"); err != nil || receipt.Message.ID != "abcdef00" {
		t.Errorf("exact ID did not win: %+v %v", receipt, err)
	}
	add("unique00-visible", caller, peer)
	add("unique00-hidden", peer, outsider)
	if receipt, err := b.StatusContext(context.Background(), caller, "unique00"); err != nil || receipt.Message.ID != "unique00-visible" {
		t.Errorf("hidden receipt made visible prefix ambiguous: %+v %v", receipt, err)
	}
	add("expired0-visible", caller, peer)
	b.mu.Lock()
	expired := b.receipts["expired0-visible"]
	expired.RetainUntil = now.Add(-time.Second)
	b.receipts["expired0-visible"] = expired
	b.mu.Unlock()
	for _, prefix := range []string{"unknown0", "expired0", "abcdef00-hidden"} {
		request.ID = prefix
		response = Handle(context.Background(), b, request)
		if response.Code != "refused" || strings.Contains(response.Error, "no connected recipient retains") || response.Receipt != nil {
			t.Errorf("unknown/invisible lookup misreported delivery: %+v", response)
		}
		if prefix == "unknown0" && !strings.Contains(response.Error, "unknown message ID or prefix") {
			t.Errorf("unknown prefix did not identify unknown lookup: %+v", response)
		}
	}
}

func TestBrokerStatusPrefixKeepsQueueProjection(t *testing.T) {
	f := newBusyReceiverFixture(t, Held)
	const id = "queue000-visible"
	if _, err := f.broker.Send(context.Background(), f.from, id, Route{Target: f.to.Slot}, "queued report"); err != nil {
		t.Fatal(err)
	}
	for _, caller := range []Binding{f.from, f.to} {
		receipt, err := f.broker.StatusContext(context.Background(), caller, "queue000")
		if err != nil || receipt.Message.ID != id || receipt.Status != Queued || !strings.Contains(receipt.Detail, "position 4") || !strings.Contains(receipt.Detail, "waiting for recognized composer") {
			t.Fatalf("queue projection for %s: %+v %v", caller.Slot, receipt, err)
		}
	}
}

func TestBrokerStatusKeepsComposerEvidenceForHeadAndFollowers(t *testing.T) {
	f := newBusyReceiverFixture(t, Queued)
	const reason = "waiting for recognized composer: cursor is hidden"
	f.endpoint.interim <- Receipt{Status: Queued, Detail: reason}
	select {
	case <-f.endpoint.progressApplied:
	case <-testwait.After(t):
		t.Fatal("receiver did not publish its wait reason")
	}
	for i, m := range f.messages {
		response := Handle(context.Background(), f.broker, Request{Op: "status", Scope: f.from.Scope, Tag: f.from.Tag, Session: f.from.Session, Nonce: f.from.Nonce, ID: m.ID})
		want := fmt.Sprintf("position %d; %s", i+1, reason)
		if response.Receipt == nil || response.Receipt.Detail != want || response.Receipt.Status.Terminal() {
			t.Fatalf("message-status lost wrapper reason: %+v, want %q", response, want)
		}
	}
}

func TestBrokerStatusPrefixFindsRefusalAndExpiresIt(t *testing.T) {
	var clock atomic.Int64
	clock.Store(time.Now().UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, clock.Load()) }, nil)
	defer b.Close()
	caller, peer := brokerBinding("ops:0"), brokerBinding("pair:1")
	_ = b.Register(caller, newFakeEndpoint(time.Now()))
	_ = b.Register(peer, &failingObserveEndpoint{FakeDeliveryEndpoint: newFakeEndpoint(time.Now()), err: ErrUnavailable})
	const id = "refused0-visible"
	if _, err := b.Send(context.Background(), caller, id, Route{Target: peer.Slot}, "refused report"); err == nil {
		t.Fatal("send unexpectedly admitted")
	}
	if receipt, err := b.StatusContext(context.Background(), caller, "refused0"); err != nil || receipt.Status != Refused || receipt.Message.ID != id {
		t.Fatalf("refusal prefix: %+v %v", receipt, err)
	}
	clock.Add(int64(RefusalRetention))
	if _, err := b.StatusContext(context.Background(), caller, "refused0"); err == nil || !strings.Contains(err.Error(), "unknown message ID or prefix") {
		t.Fatalf("expired refusal still resolved: %v", err)
	}
}
