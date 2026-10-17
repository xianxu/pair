package couchcmd

import (
	"context"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// listRefusals asks the running supervisor, from memory, which slots' latest
// sends were refused (pair#618). Best effort: no Couch, an older one, or a slow
// answer leaves the list exactly as the store reads it.
func listRefusals(rt Runtime, call messageCall) map[string]couchmessage.RefusalInfo {
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	var result couchmessage.Response
	if call(ctx, socket, couchmessage.Request{Op: "refusals"}, &result) != nil || result.Code != "ok" {
		return nil
	}
	return result.Refusals
}

// receiveStateText is the list's one state and one action for a slot whose
// latest send was refused. A refusal from the slot's own registration (the
// wrapper no longer owns its binding, or its conversation changed) clears only
// with a relaunch; one from Couch's side (a stale zellij view, an admission
// that ran out of time under load) clears by itself, so the action is to send
// again.
// pair:m5-concept pure
func receiveStateText(slot string, refusal couchmessage.RefusalInfo) string {
	action := "send again; if it repeats, check couch --resources"
	if refusal.Class == couchmessage.RefusalRelaunch {
		action = "couch --relaunch " + slot + " --confirm"
	}
	return "can't receive: " + refusal.Reason + " -- fix: " + action
}
