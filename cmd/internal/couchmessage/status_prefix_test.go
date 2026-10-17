package couchmessage

import (
	"context"
	"strings"
	"testing"
	"time"
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
	for _, prefix := range []string{"unknown0", "abcdef00-hidden"} {
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
