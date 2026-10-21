package couchmessage

import (
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"testing"
)

func TestProtocolManualClosedShapes(t *testing.T) {
	m := &couchschedule.ManualRequest{Intent: couchschedule.ManualIntent{ID: "run-1", Hub: "/repo/hub", Task: "task", Args: []string{"arg"}}}
	if err := ValidateRequest(Request{Op: "schedule-run", Manual: m}); err != nil {
		t.Fatal(err)
	}
	for _, r := range []Request{{Op: "schedule-status", Manual: m}, {Op: "schedule-run", Manual: m, Body: "extra"}, {Op: "schedule-run-status", Manual: m}, {Op: "schedule-run-cancel", Manual: m}, {Op: "schedule-run"}} {
		if ValidateRequest(r) == nil {
			t.Fatalf("accepted %#v", r)
		}
	}
	m.Intent.Args = nil
	if err := ValidateRequest(Request{Op: "schedule-run-status", Manual: m}); err != nil {
		t.Fatal(err)
	}
}
