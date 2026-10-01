package couchmessage

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func shortCandidate(slot, repository, agent string, now time.Time) Candidate {
	b := binding(slot)
	b.Repository, b.Agent = repository, agent
	return Candidate{Binding: b, Resting: true, Known: true, LastActivity: now.Add(-time.Minute), Remaining: 8, Supported: true}
}

func TestRoutingResolvesPrefixAliasAndAgent(t *testing.T) {
	now := time.Unix(1000, 0)
	live := []Candidate{
		shortCandidate("parley.nvim:1", "/w/parley.nvim/.git", "claude", now),
		shortCandidate("pair:1", "/w/pair/.git", "claude", now),
		shortCandidate("pair:2", "/w/pair/.git", "codex", now),
		shortCandidate("xianxu.dev:1", "/w/xianxu.dev/.git", "claude", now),
		shortCandidate("brain:0", "/w/brain/.git", "claude", now),
		shortCandidate("brainstorm:0", "/w/brainstorm/.git", "claude", now),
	}
	aliases := map[string]string{"xianxu.dev": "blog"}
	for _, tc := range []struct {
		route Route
		want  string
	}{
		{Route{Target: "parley:1"}, "parley.nvim:1"},
		{Route{Target: "parley"}, "parley.nvim:1"},
		{Route{Target: "blog:1"}, "xianxu.dev:1"},
		{Route{Target: "blog"}, "xianxu.dev:1"},
		{Route{Target: "xianxu.dev:1"}, "xianxu.dev:1"},
		{Route{Target: "pair"}, "pair:1"},
		{Route{Target: "pair", Agent: "codex"}, "pair:2"},
		{Route{Target: "pair", Agent: "claude"}, "pair:1"},
		{Route{Target: "pair:2", Agent: "codex"}, "pair:2"},
		{Route{Target: "brain"}, "brain:0"}, // exact beats being a prefix of brainstorm
	} {
		got, err := ResolveRecipient(tc.route, live, aliases, now)
		if err != nil || got.Slot != tc.want {
			t.Fatalf("%+v: got %q %v, want %s", tc.route, got.Slot, err, tc.want)
		}
	}
	if _, err := ResolveRecipient(Route{Target: "bra"}, live, aliases, now); !errors.Is(err, ErrAmbiguous) || !strings.Contains(err.Error(), "brainstorm") {
		t.Fatalf("ambiguous prefix: %v", err)
	}
	if _, err := ResolveRecipient(Route{Target: "pair", Agent: "gemini"}, live, aliases, now); !errors.Is(err, ErrNoRecipient) || !strings.Contains(err.Error(), "gemini") {
		t.Fatalf("absent agent: %v", err)
	}
	if _, err := ResolveRecipient(Route{Target: "pair:1", Agent: "codex"}, live, aliases, now); !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "runs claude") {
		t.Fatalf("exact agent mismatch: %v", err)
	}
}

func TestRoutingMissKeepsCodeAndListsLiveSlots(t *testing.T) {
	now := time.Unix(1000, 0)
	live := []Candidate{shortCandidate("pair:1", "/w/pair/.git", "claude", now), shortCandidate("pair:2", "/w/pair/.git", "codex", now)}
	_, err := ResolveRecipient(Route{Target: "nope"}, live, nil, now)
	if !errors.Is(err, ErrNoRecipient) || !strings.Contains(err.Error(), "live: pair:1 claude, pair:2 codex") {
		t.Fatalf("family miss: %v", err)
	}
	_, err = ResolveRecipient(Route{Target: "nope:1"}, live, nil, now)
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "pair:1 claude") {
		t.Fatalf("exact miss: %v", err)
	}
	// A known family with the slot missing is not a miss of the repository.
	_, err = ResolveRecipient(Route{Target: "pair:9"}, live, nil, now)
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "live:") {
		t.Fatalf("known family: %v", err)
	}
	var many []Candidate
	for i := 0; i < 128; i++ {
		many = append(many, shortCandidate(strings.Repeat("r", 60)+string(rune('a'+i%26))+":"+string(rune('1'+i%9)), "/w/x/.git", strings.Repeat("agent", 20), now))
	}
	_, err = ResolveRecipient(Route{Target: "zzz"}, many, nil, now)
	if err == nil || len(err.Error()) > 1200 || !strings.Contains(err.Error(), "more") {
		t.Fatalf("unbounded miss (%d bytes): %v", len(err.Error()), err)
	}
}

func TestRoutingAgentFilterNeverHidesAmbiguousFamily(t *testing.T) {
	now := time.Unix(1000, 0)
	live := []Candidate{
		shortCandidate("pair:1", "/a/pair/.git", "claude", now),
		shortCandidate("pair:2", "/b/pair/.git", "codex", now),
	}
	if _, err := ResolveRecipient(Route{Target: "pair", Agent: "codex"}, live, nil, now); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("filter hid ambiguity: %v", err)
	}
}

func TestBrokerDeliversToResolvedShortTarget(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	b := NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, func(context.Context, Binding) (bool, error) { return true, nil })
	defer b.Close()
	b.SetAliases(func(context.Context) (map[string]string, error) { return map[string]string{"pair": "p"}, nil })
	from := brokerBinding("brain:0")
	parley := brokerBinding("parley.nvim:1")
	claude, codex := brokerBinding("pair:1"), brokerBinding("pair:2")
	claude.Agent = "claude"
	endpoints := map[string]*FakeDeliveryEndpoint{}
	for _, v := range []Binding{from, parley, claude, codex} {
		endpoints[v.Slot] = newFakeEndpoint(base)
		if err := b.Register(v, endpoints[v.Slot]); err != nil {
			t.Fatal(err)
		}
	}
	nanos.Store(base.Add(time.Minute).UnixNano())
	expect := func(id string, route Route, slot string) {
		t.Helper()
		r, err := b.Send(context.Background(), from, id, route, "body-"+id)
		if err != nil || r.Message.To.Slot != slot {
			t.Fatalf("%+v: %+v %v", route, r, err)
		}
		select {
		case m := <-endpoints[slot].delivered:
			if m.Body != "body-"+id {
				t.Fatalf("delivered %q", m.Body)
			}
			endpoints[slot].outcomes <- Receipt{Message: m, Status: Submitted}
		case <-time.After(time.Second):
			t.Fatalf("%s did not receive %s", slot, id)
		}
	}
	expect("prefix", Route{Target: "parley:1"}, "parley.nvim:1")
	expect("agent", Route{Target: "p", Agent: "codex"}, "pair:2")
	actors, err := b.Actors(context.Background(), from)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range actors {
		if want := map[bool]string{true: "p"}[a.Binding.Slot == "pair:1" || a.Binding.Slot == "pair:2"]; a.Alias != want {
			t.Fatalf("%s alias %q", a.Binding.Slot, a.Alias)
		}
	}
}

func TestValidateRequestAgentFilter(t *testing.T) {
	send := Request{Op: "send", Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch", ID: "id", Target: "pair", Body: "work", Agent: "codex"}
	if err := ValidateRequest(send); err != nil {
		t.Fatalf("send with agent: %v", err)
	}
	for _, agent := range []string{"co dex", "a:b", "a/b", "\x1b"} {
		r := send
		r.Agent = agent
		if ValidateRequest(r) == nil {
			t.Errorf("agent %q accepted", agent)
		}
	}
	binding := protocolBinding("pair:0")
	for _, r := range []Request{
		{Op: "actors", Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch", Agent: "codex"},
		{Op: "status", Scope: "scope", Tag: "pair:0", Session: "session", Nonce: "launch", ID: "id", Agent: "codex"},
		{Op: "register", Binding: &binding, Agent: "codex"},
	} {
		if ValidateRequest(r) == nil {
			t.Errorf("%s accepted an agent filter", r.Op)
		}
	}
}
