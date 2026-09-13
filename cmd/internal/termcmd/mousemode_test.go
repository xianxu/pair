package termcmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// Every held state against every wanted state: tracking is one slot with four
// values, SGR is a bit, so the product is 8 x 8. The oracle feeds the prefix
// to a fresh Screen holding `held` and asks whether it now holds `want` --
// the terminal model ptychild already owns, rather than a second hand-written
// table of expected bytes.
func TestMouseReconcileReachesEveryWantedStateFromEveryHeldState(t *testing.T) {
	var states [][]int
	for _, tracking := range []int{0, 1000, 1002, 1003} {
		for _, sgr := range []bool{false, true} {
			var modes []int
			if tracking != 0 {
				modes = append(modes, tracking)
			}
			if sgr {
				modes = append(modes, 1006)
			}
			states = append(states, modes)
		}
	}
	for _, held := range states {
		for _, want := range states {
			t.Run(fmt.Sprintf("%v->%v", held, want), func(t *testing.T) {
				var terminal ptychild.Screen
				terminal.FeedFraming([]byte(hostty.PrivateModes(held, true)))
				prefix := mouseReconcile(held, want)
				terminal.FeedFraming([]byte(prefix))
				if got := terminal.MouseModes(); fmt.Sprint(got) != fmt.Sprint(want) {
					t.Fatalf("after %q the terminal holds %v, want %v", prefix, got, want)
				}
				if fmt.Sprint(held) == fmt.Sprint(want) && prefix != "" {
					t.Fatalf("equal states wrote %q, want nothing", prefix)
				}
				// One write per axis, never more.
				if n := strings.Count(prefix, "\x1b["); n > 2 {
					t.Fatalf("prefix %q is %d writes, want at most one per axis", prefix, n)
				}
			})
		}
	}
}

// The operator's shape (#240): nvim (parley) in tab 1 holding ?1002;1006, a
// shell in tab 2 that never said a word about the mouse. Switching to the
// shell must DROP nvim's modes from the pane -- zellij otherwise keeps
// forwarding every click to the shell and never selects -- and switching back
// must RAISE them again without relying on nvim's startup bytes still being
// in its replay ring: the reconcile prefix precedes the clear, the replay
// follows it, and that order is what the test reads.
func TestSwitchingTabsReconcilesThePanesMouseModesToTheIncomingChild(t *testing.T) {
	var stdout bytes.Buffer
	nvim := ptychild.NewFakeChild(nil)
	shell := ptychild.NewFakeChild([]byte("$ "))
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: nvim},
			{id: 2, name: "terminal 2", child: shell},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}

	// nvim enables tracking; the bytes pass through the live path, which is
	// what sets the PANE's mode and what hostScan learns it from.
	enable := []byte("\x1b[?1002h\x1b[?1006h")
	nvim.Feed(enable)
	mux.handleChunk(ptyChunk{id: 1, data: enable})
	if got := stdout.String(); !strings.Contains(got, string(enable)) {
		t.Fatalf("live path wrote %q, want nvim's enable passed through", got)
	}
	stdout.Reset()

	mux.nextTab() // to the shell
	got := stdout.String()
	drop := mouseReconcile([]int{1002, 1006}, nil)
	if i, j := strings.Index(got, drop), strings.Index(got, hostty.HomeAndClear); i < 0 || j < 0 || i > j {
		t.Fatalf("switch to the shell wrote %q, want %q BEFORE the clear", got, drop)
	}
	stdout.Reset()

	mux.previousTab() // back to nvim
	got = stdout.String()
	// The prefix is byte-identical to nvim's own startup bytes, which the
	// replay ALSO carries -- so the proof that it was asserted rather than
	// replayed is its position: the prefix precedes the clear, the replay
	// follows it.
	raise := mouseReconcile(nil, []int{1002, 1006})
	if i, j := strings.Index(got, raise), strings.Index(got, hostty.HomeAndClear); i < 0 || j < 0 || i > j {
		t.Fatalf("switch back to nvim wrote %q, want %q BEFORE the clear (asserted), not only after it (replayed)", got, raise)
	}
	stdout.Reset()

	// Silent to silent: no mode bytes at all.
	mux.nextTab()
	stdout.Reset()
	mux.tabs = append(mux.tabs, &terminalTab{id: 3, name: "terminal 3", child: ptychild.NewFakeChild(nil)})
	mux.nextTab()
	if got := stdout.String(); strings.Contains(got, "\x1b[?10") {
		t.Fatalf("silent to silent wrote mouse mode bytes: %q", got)
	}
}

// Closing the nvim tab hands the screen to the survivor through the same
// takeover, so the release must be written there too -- the closed child's
// modes must not outlive it on the pane.
func TestClosingATabReleasesTheDeadChildsMouseModes(t *testing.T) {
	var stdout bytes.Buffer
	nvim := ptychild.NewFakeChild(nil)
	mux := &terminalMux{
		pane: paneWriter{w: stdoutWriter{&stdout}},
		rt:   &fakeRuntime{},
		tabs: []*terminalTab{
			{id: 1, name: "terminal 1", child: nvim},
			{id: 2, name: "terminal 2", child: ptychild.NewFakeChild(nil)},
		},
		active: 0,
		cols:   40,
		rows:   24,
	}
	enable := []byte("\x1b[?1002h\x1b[?1006h")
	nvim.Feed(enable)
	mux.handleChunk(ptyChunk{id: 1, data: enable})
	stdout.Reset()

	mux.removeTab(1)

	if got, want := stdout.String(), mouseReconcile([]int{1002, 1006}, nil); !strings.Contains(got, want) {
		t.Fatalf("closing the nvim tab wrote %q, want %q", got, want)
	}
}
