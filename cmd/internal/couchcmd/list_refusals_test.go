package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// couch --list shows a refused slot's reason and the one action that clears it
// (pair#618): a relaunch when the slot's own registration is at fault, a resend
// when Couch's side was (stale view, admission out of time).
func TestListShowsAReceiveStateFromTheSupervisorsRefusals(t *testing.T) {
	rows := []couchcore.ThreadSummary{
		{Address: couchcore.ThreadAddress{RepoScope: "s3", Tag: "1-pair-58"}, WorkingPath: "/w/pair-slot3/pair", Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: 3}}, State: couchcore.ThreadLive},
		{Address: couchcore.ThreadAddress{RepoScope: "s4", Tag: "1-pair-60"}, WorkingPath: "/w/pair-slot4/pair", Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: 4}}, State: couchcore.ThreadLive},
	}
	refusals := map[string]couchmessage.RefusalInfo{
		"pair:3": {Reason: "wrapper does not own the current binding", Class: couchmessage.RefusalRelaunch},
		"pair:4": {Reason: "admission timed out before the recipient accepted it"},
	}
	var out bytes.Buffer
	renderThreadRows(&out, rows, false, refusals)
	text := out.String()
	if !strings.Contains(text, "can't receive: wrapper does not own the current binding -- fix: couch --relaunch pair:3 --confirm") {
		t.Fatalf("pair:3 receive state missing:\n%s", text)
	}
	if !strings.Contains(text, "can't receive: admission timed out before the recipient accepted it -- fix: send again") {
		t.Fatalf("pair:4 receive state missing:\n%s", text)
	}
	// A second pair:3 row (a stale record) makes the labels collide and gain a
	// tag tail; the receive state still finds the slot (BR-13).
	stale := rows[0]
	stale.Address.Tag = "1-pair-12"
	out.Reset()
	renderThreadRows(&out, append([]couchcore.ThreadSummary{stale}, rows...), false, refusals)
	if strings.Count(out.String(), "fix: couch --relaunch pair:3 --confirm") != 2 {
		t.Fatalf("collided labels lost the receive state:\n%s", out.String())
	}
	out.Reset()
	renderThreadRows(&out, rows, false, nil)
	if strings.Contains(out.String(), "can't receive") {
		t.Fatal("no supervisor answer must leave the rows unchanged")
	}
}

func TestListRefusalsIsBestEffort(t *testing.T) {
	answer := func(ctx context.Context, _ string, request any, response any) error {
		if r := request.(couchmessage.Request); r != (couchmessage.Request{Op: "refusals"}) {
			t.Fatalf("request %+v", r)
		}
		*(response.(*couchmessage.Response)) = couchmessage.Response{Code: "ok", Refusals: map[string]couchmessage.RefusalInfo{"pair:3": {Reason: "x"}}}
		return nil
	}
	if got := listRefusals(broadcastListRuntime(), answer); got["pair:3"].Reason != "x" {
		t.Fatalf("refusals %v", got)
	}
	down := func(context.Context, string, any, any) error { return errors.New("no running couch") }
	if got := listRefusals(broadcastListRuntime(), down); got != nil {
		t.Fatalf("no supervisor: %v", got)
	}
}

func TestListReceiveActionUsesRefusalClassNotReason(t *testing.T) {
	rows := []couchcore.ThreadSummary{{Address: couchcore.ThreadAddress{RepoScope: "s3", Tag: "1-pair-58"}, Target: couchcore.ThreadTarget{Kind: couchcore.ThreadTargetSlot, Slot: couchcore.SlotIdentity{Repo: "pair", Number: 3}}}}
	for _, tc := range []struct {
		reason, class string
		relaunch      bool
	}{
		{"different wrapper owns this launch", "relaunch", true},
		{"native conversation identity was replaced", "relaunch", true},
		{"ready evidence is unavailable", "relaunch", true},
		{"terminal absent from verified listing", "relaunch", true},
		{"retained working path missing", "relaunch", true},
		{"wrapper does not own the current binding", "", false},
		{"wrapper conversation changed", "", false},
		{"wrapper launch registration unavailable", "", false},
		{"message terminal is not positively listed", "", false},
		{"message thread has no retained checkout", "", false},
		{"wrapper conversation changed", "future-class", false},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"Code": "ok", "Refusals": map[string]any{"pair:3": map[string]string{"reason": tc.reason, "class": tc.class}}})
			if err != nil {
				t.Fatal(err)
			}
			var response couchmessage.Response
			if err := json.Unmarshal(raw, &response); err != nil {
				t.Fatalf("decode structured refusal: %v", err)
			}
			var out bytes.Buffer
			renderThreadRows(&out, rows, false, response.Refusals)
			want := "fix: send again"
			if tc.relaunch {
				want = "fix: couch --relaunch pair:3 --confirm"
			}
			if !strings.Contains(out.String(), tc.reason) || !strings.Contains(out.String(), want) {
				t.Fatalf("receive advice = %q; want reason plus %q", out.String(), want)
			}
		})
	}
}

func TestListRefusalsMixedVersionLeavesOrdinaryListIntact(t *testing.T) {
	legacy := func(_ context.Context, _ string, _ any, response any) error {
		return json.Unmarshal([]byte(`{"Code":"ok","Refusals":{"pair:3":"wrapper does not own the current binding"}}`), response)
	}
	if got := listRefusals(broadcastListRuntime(), legacy); got != nil {
		t.Fatalf("legacy refusal shape must omit annotations: %+v", got)
	}
	var oldResponse struct{ Refusals map[string]string }
	raw := []byte(`{"Refusals":{"pair:3":{"reason":"reworded","class":"relaunch"}}}`)
	if json.Unmarshal(raw, &oldResponse) == nil {
		t.Fatal("fixture no longer models old-client decode failure")
	}
}
