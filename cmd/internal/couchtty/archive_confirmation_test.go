package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// Archive on a row whose session is still alive stops a RUNNING agent, and the
// confirmation is the only place that can say so: RenderMenuView overwrites
// line 0 with the breadcrumb, so the frame title never reaches the screen.
//
// The wording is decided by a MEASUREMENT, not by what archive intends (#256 M3
// Task 10, Step 0). `zellij delete-session --force` reaps a pane by SIGHUP, and
// a pane process that inherited SIG_IGN survives it -- measured both ways on
// 2026-09-17, same fixture, the only variable being the launching shell's SIGHUP
// disposition. #274's 106 orphaned `pair term` trees are that regime in
// production. So the confirmation must not promise the agent stops.
func TestArchiveConfirmationNamesTheRunningAgent(t *testing.T) {
	address := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0000000000000001"}
	cases := []struct {
		name   string
		row    couchcore.ActionableThreadSummary
		wants  []string
		avoids []string
	}{
		{
			name: "a detached row still has an agent behind its session",
			row: couchcore.ActionableThreadSummary{
				Address: address, Name: "brain", Agent: "claude", State: couchcore.ThreadDetached,
			},
			wants: []string{"archive ", "brain", "claude", "may survive"},
		},
		{
			name: "a parked row has no agent to name",
			row: couchcore.ActionableThreadSummary{
				Address: address, Name: "brain", Agent: "claude", State: couchcore.ThreadParked,
			},
			wants:  []string{"archive ", "brain"},
			avoids: []string{"may survive"},
		},
		{
			name: "a detached row whose profile named no agent still warns",
			row: couchcore.ActionableThreadSummary{
				Address: address, Name: "brain", State: couchcore.ThreadDetached,
			},
			wants:  []string{"archive ", "brain", "may survive"},
			avoids: []string{"  "},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := MenuState{Inventory: []couchcore.ActionableThreadSummary{tc.row}}
			frame := MenuFrame{Kind: MenuFrameConfirmation, Action: "archive", Thread: address}
			items := confirmationMenuItems(state, frame)
			if len(items) != 2 || items[0] != "cancel" {
				t.Fatalf("confirmation items = %q", items)
			}
			for _, want := range tc.wants {
				if !strings.Contains(items[1], want) {
					t.Errorf("confirmation %q does not contain %q", items[1], want)
				}
			}
			for _, avoid := range tc.avoids {
				if strings.Contains(items[1], avoid) {
					t.Errorf("confirmation %q contains %q", items[1], avoid)
				}
			}
		})
	}
}
