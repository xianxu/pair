package couchmessage

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func boundaryBinding(t *testing.T, slot string) Binding {
	t.Helper()
	b := binding(slot)
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	// Fill the supported 4 KiB encoded identity budget exactly.
	b.Repository += strings.Repeat("x", MaxBindingBytes-len(raw))
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTransportDomainBounds(t *testing.T) {
	from, to := boundaryBinding(t, "pair:0"), boundaryBinding(t, "pair:1")
	now := time.Now().UTC()
	m := Message{ID: strings.Repeat("<", 128), From: from, To: to, Body: strings.Repeat("<", MaxBodyBytes), Deadline: now.Add(time.Minute)}
	receipt := Receipt{Message: m, Status: Submitted, Detail: strings.Repeat("<", MaxReceiptDetailBytes), RetainUntil: now.Add(time.Hour)}
	actors := make([]Candidate, MaxActors)
	for i := range actors {
		actors[i] = Candidate{Binding: boundaryBinding(t, fmt.Sprintf("pair:%d", i)), Known: true, Resting: true, LastActivity: now, Remaining: InboundAllowance, Supported: true}
	}
	send := Request{Op: "send", Scope: from.Scope, Tag: from.Tag, Session: from.Session, Nonce: from.Nonce, ID: m.ID, Target: to.Slot, Body: m.Body}
	commit := EndpointRequest{Op: "commit", Binding: to, ID: m.ID, Message: &m}
	if err := ValidateRequest(send); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEndpointRequest(commit); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name              string
		request, response any
	}{
		{"send", send, Response{Code: "accepted", Receipt: &receipt}},
		{"endpoint-commit", commit, EndpointResponse{Receipt: &receipt}},
		{"receipt-status", Request{Op: "status", Scope: from.Scope, Tag: from.Tag, Session: from.Session, Nonce: from.Nonce, ID: m.ID}, Response{Code: "ok", Receipt: &receipt}},
		{"actors", Request{Op: "actors", Scope: from.Scope, Tag: from.Tag, Session: from.Session, Nonce: from.Nonce}, Response{Code: "ok", Actors: actors}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			socket := transportSocket(t)
			server, err := StartServer(context.Background(), socket, func(_ context.Context, raw []byte) ([]byte, error) {
				want, _ := json.Marshal(tc.request)
				if string(raw) != string(want) {
					t.Error("request changed in transit")
				}
				return json.Marshal(tc.response)
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			got := reflect.New(reflect.TypeOf(tc.response)).Interface()
			if err := Call(context.Background(), socket, tc.request, got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reflect.ValueOf(got).Elem().Interface(), tc.response) {
				t.Fatal("response changed in transit")
			}
		})
	}
}

func TestTransportOversizedResponseIsExplicit(t *testing.T) {
	socket := transportSocket(t)
	server, err := StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) { return json.Marshal(strings.Repeat("x", MaxFrameBytes)) })
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var response any
	err = Call(context.Background(), socket, struct{}{}, &response)
	if err == nil || !strings.Contains(err.Error(), "response exceeds frame limit") || !strings.Contains(err.Error(), "uncertain") {
		t.Fatalf("expected explicit uncertain response size error; got %v", err)
	}
}

func TestBindingEncodedLimit(t *testing.T) {
	b := boundaryBinding(t, "pair:1")
	b.Repository += "x"
	if err := b.Validate(); err == nil {
		t.Fatal("binding beyond encoded limit accepted")
	}
	b = binding("pair:1")
	b.Repository = strings.Repeat("<", 4096)
	if err := b.Validate(); err == nil {
		t.Fatal("escaped binding beyond encoded limit accepted")
	}
}

func TestRequestIdentityBounds(t *testing.T) {
	r := Request{Op: "actors", Scope: strings.Repeat("<", 4097), Tag: "tag", Session: "session", Nonce: "nonce"}
	if err := ValidateRequest(r); err == nil {
		t.Fatal("oversized identity accepted")
	}
}

func TestReceiptDetailBound(t *testing.T) {
	now := time.Now()
	to := binding("pair:1")
	s := registered(t, to, now)
	m := Message{ID: "id", From: binding("pair:0"), To: to, Body: "body", Deadline: now.Add(time.Minute)}
	s, _ = advance(t, s, Event{Kind: Admit, Binding: to, Message: m, At: now})
	_, fx := advance(t, s, Event{Kind: DeliveryFinished, Binding: to, Message: m, Status: Indeterminate, Detail: strings.Repeat("界", 1024), At: now})
	if len(fx) != 1 || len(fx[0].Receipt.Detail) > 1024 {
		t.Fatalf("unbounded receipt detail: %d", len(fx[0].Receipt.Detail))
	}
	if !utf8.ValidString(fx[0].Receipt.Detail) {
		t.Fatal("detail truncation broke UTF-8")
	}
}

func TestAdmissionRejectsOversizedMessageID(t *testing.T) {
	now := time.Now()
	to := binding("pair:1")
	state := registered(t, to, now)
	m := Message{ID: strings.Repeat("x", 129), From: binding("pair:0"), To: to, Body: "body", Deadline: now.Add(time.Minute)}
	if _, _, err := Advance(state, Event{Kind: Admit, Binding: to, Message: m, At: now}); err == nil {
		t.Fatal("oversized message ID admitted")
	}
}
