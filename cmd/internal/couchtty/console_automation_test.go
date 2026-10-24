package couchtty

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

type automationControlSetter interface {
	SetAutomationControl(func(string, string) (bool, error))
}

func installAutomationControl(t *testing.T, c *Console, fn func(string, string) (bool, error)) {
	t.Helper()
	setter, ok := any(c).(automationControlSetter)
	if !ok {
		t.Fatal("Console has no automation control callback")
	}
	setter.SetAutomationControl(fn)
}

func TestSwitcherAutomationSelectedOwnership(t *testing.T) {
	for _, tc := range []struct {
		name string
		row  couchcore.ActionableThreadSummary
		hub  string
	}{
		{"hub", presentationOrdinary("/src/bench", "b"), "/src/bench"},
		{"claimed worker", presentationSlot("/src/pair", 7), "/src/bench"},
		{"default worker", presentationSlot("/src/pair", 8), "/src/ops"},
		{"main hub", presentationOrdinary("/src/ops", "o"), "/src/ops"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 120}), nil)
			t.Cleanup(c.Stop)
			c.SetHubOwnership(multiHubFixture(t))
			c.mu.Lock()
			c.ensureMenuLocked(couchcore.ThreadAddress{})
			c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{tc.row}})
			before := c.focus
			c.mu.Unlock()
			calls := 0
			enabled := false
			installAutomationControl(t, c, func(hub, action string) (bool, error) {
				// Reentering proves the Console mutex is not held by the callback.
				_ = c.menuSnapshot()
				if hub != tc.hub || action != "toggle" {
					t.Errorf("control(%q,%q), want (%q,toggle)", hub, action, tc.hub)
				}
				calls++
				enabled = !enabled
				return enabled, nil
			})
			for _, state := range []string{"on", "off"} {
				c.onMenuInput([]byte{7})
				if c.menuSnapshot().Notice.Level != MenuNoticeInfo {
					t.Fatal("successful automation toggle reported as error")
				}
				if got := c.menuSnapshot().Notice.Text; !strings.Contains(got, tc.hub) || !strings.Contains(got, "automation "+state) {
					t.Fatalf("notice=%q, want hub and automation %s", got, state)
				}
			}
			if calls != 2 {
				t.Fatalf("calls=%d", calls)
			}
			if c.focus != before || len(c.order) != 0 {
				t.Fatal("automation changed focus or started a process")
			}
		})
	}
}

func TestSwitcherAutomationRefusesUnverifiedOwnership(t *testing.T) {
	for _, kind := range []string{"no hubs", "no selection", "unknown root", "conflict", "controller unavailable", "controller error"} {
		t.Run(kind, func(t *testing.T) {
			c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 120}), nil)
			t.Cleanup(c.Stop)
			ownership := multiHubFixture(t)
			row := presentationSlot("/src/pair", 7)
			if kind == "conflict" {
				hubs := ownership.Hubs()
				hubs[0].Claims = hubs[1].Claims
				var err error
				ownership, err = couchcore.NewHubOwnership(hubs)
				if err != nil {
					t.Fatal(err)
				}
			}
			if kind != "no hubs" {
				c.SetHubOwnership(ownership)
			}
			if kind == "unknown root" {
				row = presentationOrdinary("/src/unknown", "u")
				row.StartingPath = "relative"
			}
			c.mu.Lock()
			c.ensureMenuLocked(couchcore.ThreadAddress{})
			if kind != "no selection" {
				c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{row}})
			}
			c.mu.Unlock()
			calls := 0
			if kind != "controller unavailable" {
				installAutomationControl(t, c, func(string, string) (bool, error) { calls++; return false, errors.New("owner refused") })
			}
			c.onMenuInput([]byte{7})
			want := 0
			if kind == "controller error" {
				want = 1
			}
			if calls != want {
				t.Fatalf("calls=%d want %d", calls, want)
			}
			if got := c.menuSnapshot().Notice.Text; !strings.Contains(got, "automation") {
				t.Fatalf("no actionable refusal: %q", got)
			}
		})
	}
}

func TestSwitcherAutomationWireSplitsAndChildPassThrough(t *testing.T) {
	for _, wire := range []string{"\x07", "\x1b[103;5u", "\x1b[103;5:1u"} {
		for split := 0; split <= len(wire); split++ {
			t.Run(wire+string(rune('A'+split)), func(t *testing.T) {
				calls := make(chan struct{}, 4)
				f := newFixtureBeforeRun(t, 24, 120, func(c *Console) {
					c.SetHubOwnership(multiHubFixture(t))
					installAutomationControl(t, c, func(hub, action string) (bool, error) { calls <- struct{}{}; return true, nil })
				})
				// Actor input follows its existing terminal encoding (legacy here),
				// without acquiring a global automation binding.
				_, _ = f.stdin.Write([]byte(wire[:split]))
				_, _ = f.stdin.Write([]byte(wire[split:]))
				waitFor(t, "Ctrl+G forwarded to actor", func() bool { return bytes.Contains(bytes.Join(f.child.Writes(), nil), []byte{7}) })
				select {
				case <-calls:
					t.Fatal("actor input controlled automation")
				default:
				}
				// Install a verified selected row before opening the real switcher.
				f.con.mu.Lock()
				f.con.ensureMenuLocked(couchcore.ThreadAddress{})
				f.con.menu, _ = ReduceMenu(f.con.menu, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{presentationOrdinary("/src/bench", "b")}})
				f.con.mu.Unlock()
				_, _ = f.stdin.Write([]byte{0})
				waitFor(t, "switcher", func() bool { f.con.mu.Lock(); defer f.con.mu.Unlock(); return f.con.focus.IsPanel() })
				_, _ = f.stdin.Write([]byte(wire[:split]))
				_, _ = f.stdin.Write([]byte(wire[split:]))
				waitFor(t, "automation toggled", func() bool {
					return len(calls) == 1 && strings.Contains(f.con.menuSnapshot().Notice.Text, "automation on")
				})
				if got := f.con.menuSnapshot().Notice.Text; !strings.Contains(got, "/src/bench") || !strings.Contains(got, "automation on") {
					t.Fatalf("notice=%q", got)
				}
				_, _ = f.stdin.Write([]byte("\x1b[103;5:3u"))
				_, _ = f.stdin.Write([]byte("q")) // FIFO barrier after the release
				waitFor(t, "filter input", func() bool { return f.con.menuSnapshot().CurrentFrame().Filter == "q" })
				if len(calls) != 1 {
					t.Fatal("release changed automation")
				}
			})
		}
	}
}

func TestSwitcherAutomationHelp(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	body := RenderMenuView(state, 120, 24, time.Time{}, false).Body
	if !strings.Contains(body, "Ctrl+G hub automation on/off") {
		t.Fatalf("missing automation help: %s", body)
	}
}

func TestSwitcherAutomationTerminalDecoder(t *testing.T) {
	for _, tc := range []struct {
		wire    string
		toggles int
	}{
		{"\x07", 1}, {"\x1b[103;5u", 1}, {"\x1b[103;5:1u", 1}, {"\x1b[103;5:2u", 1}, {"\x1b[103;5:3u", 0},
		{"\x1b[103;69u", 1}, {"\x1b[103;133u", 1}, {"\x1b[103;7u", 0}, {"\x1b[103;6u", 0}, {"g", 0},
	} {
		t.Run(strconv.Quote(tc.wire), func(t *testing.T) {
			for split := 0; split <= len(tc.wire); split++ {
				var d terminal.Decoder
				events, err := d.Feed([]byte(tc.wire[:split]))
				if err != nil {
					t.Fatal(err)
				}
				tail, err := d.Feed([]byte(tc.wire[split:]))
				if err != nil {
					t.Fatal(err)
				}
				events = append(events, tail...)
				toggles := 0
				for _, e := range events {
					keys, _ := DecodePanelKeys(e.Canonical)
					for _, k := range keys {
						if k.Kind == KeyToggleAutomation {
							toggles++
						}
					}
				}
				if toggles != tc.toggles {
					t.Fatalf("split %d: toggles=%d want %d", split, toggles, tc.toggles)
				}
			}
		})
	}
}

func TestSwitcherAutomationAdoptsCurrentOwnership(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 120}), nil)
	t.Cleanup(c.Stop)
	ownership := multiHubFixture(t)
	c.SetHubOwnership(ownership)
	row := presentationSlot("/src/pair", 7)
	c.mu.Lock()
	c.ensureMenuLocked(couchcore.ThreadAddress{})
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{row}})
	// Nested actions retain the same canonical selected row even for slots
	// with no native thread address.
	c.menu.Frames = append(c.menu.Frames, MenuFrame{Kind: MenuFrameActions, RowKey: row.RowKey})
	c.mu.Unlock()
	var hubs []string
	installAutomationControl(t, c, func(hub, action string) (bool, error) { hubs = append(hubs, hub); return true, nil })
	c.onMenuInput([]byte{7})
	moved, err := ownership.Apply(couchcore.HubMove{Slots: []couchcore.HubSlot{{Repository: "/src/pair", Number: 7}}, Caller: "/src/bench", Target: "/src/ops", Revision: ownership.Revision()})
	if err != nil {
		t.Fatal(err)
	}
	c.SetHubOwnership(moved)
	c.onMenuInput([]byte{7})
	if len(hubs) != 2 || hubs[0] != "/src/bench" || hubs[1] != "/src/ops" {
		t.Fatalf("owners=%q", hubs)
	}
}

func TestSwitcherAutomationSelectedHubSlotUsesConfiguredRoot(t *testing.T) {
	row := presentationSlot("/src/pair", 9)
	root := row.Target.Slot.WorktreeRoot
	ownership, err := couchcore.NewHubOwnership([]couchcore.HubDefinition{
		{Root: "/src/ops", Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/ops"}},
		{Root: root, Workspace: "/src", Slot: couchcore.HubSlot{Repository: "/src/pair", Number: 9}},
	})
	if err != nil {
		t.Fatal(err)
	}
	state := NewMenuState([]couchcore.ActionableThreadSummary{row}, couchcore.ThreadAddress{})
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHubOwnership, HubOwnership: ownership})
	got, err := selectedAutomationHub(state)
	if err != nil || got != root {
		t.Fatalf("selected hub=%q, %v; want %q", got, err, root)
	}
}
