package couchcmd

import (
	"bytes"
	"context"
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
	refusals := map[string]couchmessage.Receipt{
		"pair:3": {Status: couchmessage.Refused, Detail: "wrapper does not own the current binding"},
		"pair:4": {Status: couchmessage.Refused, Detail: "admission timed out before the recipient accepted it"},
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
		*(response.(*couchmessage.Response)) = couchmessage.Response{Code: "ok", Refusals: map[string]couchmessage.Receipt{"pair:3": {Detail: "x"}}}
		return nil
	}
	if got := listRefusals(broadcastListRuntime(), answer); got["pair:3"].Detail != "x" {
		t.Fatalf("refusals %v", got)
	}
	down := func(context.Context, string, any, any) error { return errors.New("no running couch") }
	if got := listRefusals(broadcastListRuntime(), down); got != nil {
		t.Fatalf("no supervisor: %v", got)
	}
}
