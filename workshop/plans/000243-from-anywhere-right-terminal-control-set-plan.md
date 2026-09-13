# From-anywhere right-terminal control set — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The from-anywhere right-terminal control chords — `M-S-Left`, `M-S-Right`, and a new `M-S-t` — drive the right pane's tabs from any pane, and keep working when the right pane shows a full-screen app. This fixes the #227 regression (Left/Right delivered a role-scoped chord that #227 now passes through) and adds `M-S-t` (create a tab) as the third.

**Architecture:** The from-anywhere chords are GLOBAL. Each pane, on the global chord, delivers a chord's bytes to the right terminal, and pair term acts on them. The bug is that the delivery used the ROLE-SCOPED `Alt+Left`/`Right`, which #227 forwards to a full-screen child. The fix and the feature are one change: `TabChordFor` returns the GLOBAL chord for every from-anywhere action, and pair term handles all three globals directly (never passing a global through, per #227). One new chord/action/binding and one `handleTerminalChord` case add `M-S-t`; the existing route/CLI/argv plumbing carries it unchanged.

**Tech Stack:** Go — `cmd/internal/workbenchshortcut` (chord/action/binding/`TabChordFor`), `cmd/internal/termcmd` (`handleTerminalChord`), `cmd/internal/wrapcmd` (agent pane), `cmd/internal/layoutcmd` (CLI). Lua — `nvim/init.lua`, regenerated `nvim/workbench_actions.lua`. Docs — `keyhelp`, `atlas`, `README`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ChordAltShiftT` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ActionTerminalNewTab` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `TabChordFor` | `cmd/internal/workbenchshortcut/shortcut.go` | modified |

- **ChordAltShiftT** — the `M-S-t` chord. ONE sequence: `\x1b[84;4u` (KKP:
  `T`=84 uppercase + mod 4 = shift+alt), mirroring `ChordAltShiftD`
  (`\x1b[68;4u`) and `ChordAltShiftN` (`\x1b[78;4u`) — uppercase codepoint +
  mod 4 is the codebase's shift-letter convention, and it is what zellij's
  `WriteChars` bind delivers (see Task 3). **No legacy `\x1bT` form** (PQ-3):
  a global chord is never passed through, so a legacy `ESC` then `T` typed
  inside `EscapeAmbiguity` would fire new-tab inside nvim/the agent; every other
  letter global carries the KKP form only. Needs a `ChordName` ("Alt+Shift+T")
  and one `chordSequences` row.
  - **DRY rationale:** the encoding lives once in `chordSequences`; every
    consumer (FindChord, DeliverChordArgs' `ChordEncodings[0]`, DecodeChord)
    derives from it.

- **ActionTerminalNewTab** — the from-anywhere "create a right-terminal tab"
  action, alongside `ActionTerminalPrevTab`/`NextTab`.

- **TabChordFor** — now returns the **GLOBAL** chord for each from-anywhere
  action: `ActionTerminalPrevTab → ChordAltShiftLeft`, `ActionTerminalNextTab →
  ChordAltShiftRight` (the regression fix), `ActionTerminalNewTab →
  ChordAltShiftT` (new).
  - **Why global.** The delivery writes the chord's bytes to the right
    terminal, where pair term acts on them. A role-scoped chord (`Alt+Left`) is
    passthrough-eligible under #227 and gets eaten by a full-screen child; a
    global is never passed through (`RightTerminalChordPassesThrough` is false
    for globals) and is always handled by `handleTerminalChord`. So the delivered
    chord MUST be global. A test pins `IsGlobalChord(TabChordFor(a)) == true` for
    every from-anywhere action, so this class of regression cannot recur.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `handleTerminalChord` | `cmd/internal/termcmd/run.go` | modified | mux tab ops |
| agent global dispatch | `cmd/internal/wrapcmd/wrap.go` | modified | zellij delivery |
| `RunSwitchTerminalTab` | `cmd/internal/layoutcmd/layoutcmd.go` | modified | CLI → delivery |
| `globalBindings` + Lua gen | `workbenchshortcut` + `nvim/*` | modified | nvim keymaps |

- **handleTerminalChord** — add `case ChordAltShiftT: mux.newTab()`. This is what
  makes the delivered `M-S-t` bytes create a tab (as `ChordAltShiftLeft/Right`
  already switch tabs there). It is reached because a global is never passed
  through by #227's gate.
- **agent global dispatch** (`wrap.go`) — add `ActionTerminalNewTab` to the
  `ActionTerminalPrevTab/NextTab` case, so the agent pane's `M-S-t` delivers too.
- **RunSwitchTerminalTab** — accept a third direction `new` →
  `ActionTerminalNewTab`; update the usage string to `prev|next|new`. The rest
  (TabChordFor → SwitchRightTerminalTab) is unchanged.
- **globalBindings + Lua** — add the `ChordAltShiftT` row (`PairTermNewTab`,
  `<M-T>`, `HandledInPane`, Help). `RenderLuaGlobalMaps` regenerates
  `nvim/workbench_actions.lua` (adds `["<M-T>"] = {fn="PairTermNewTab"}`).
  Hand-add `PairTermNewTab() = pair_switch_terminal_tab('new')` in `init.lua`;
  `workbench_route.switch_terminal_tab_command` already passes any direction
  through as argv.

### ARCH notes

- **ARCH-DRY:** the delivered-chord mapping is one function (`TabChordFor`),
  consumed by the CLI and both pane executors; the encoding is one table row.
- **ARCH-PURPOSE:** the fix is the class ("from-anywhere delivery must use a
  chord pair term won't pass through"), pinned by the `IsGlobalChord` test, not
  the instance (Left/Right).
- **ARCH-PURE:** `Decide`/`TabChordFor`/`IsGlobalChord` stay pure; the only IO
  is the existing zellij delivery.
- **ARCH-CONSTRAINTS / SECURE / ORDER / MOCK / FUNERAL:** N/A — one chord per
  keystroke, no new durable state, no new trust surface, no external double.

---

## Chunk 1: the set

### Task 1: `ChordAltShiftT` + `ActionTerminalNewTab` (workbenchshortcut)

- [ ] **Step 1: failing test** — `chordSequences` decodes `\x1b[84;4u` to a new
  `ChordAltShiftT` (and it is the ONLY sequence — no legacy form);
  `ChordName(ChordAltShiftT) == "Alt+Shift+T"`.
- [ ] **Step 2:** add `ChordAltShiftT` to the `Chord` enum (before `chordMax`),
  `ActionTerminalNewTab` to the actions, the ONE `chordSequences` row
  (`{"\x1b[84;4u", ChordAltShiftT}`), the `ChordName` case, and the
  `namedChord` case (`termcmd/run.go:85-115`, `case "alt+shift+t"`) so
  `pair term --test-shortcut alt+shift+t` resolves it. Run, expect pass. Commit.

### Task 2: `TabChordFor` → globals (the fix + the new action)

- [ ] **Step 1: failing tests**

```go
func TestTabChordForDeliversGlobalChords(t *testing.T) {
	for _, a := range []ShortcutAction{ActionTerminalPrevTab, ActionTerminalNextTab, ActionTerminalNewTab} {
		chord, ok := TabChordFor(a)
		if !ok {
			t.Fatalf("TabChordFor(%v) not ok", a)
		}
		// The delivery writes these bytes to the right terminal; a role-scoped
		// chord would be passed through to a full-screen child (#227). Only a
		// global is always handled there. This test is the guard against the
		// #227 regression recurring.
		if !IsGlobalChord(chord) {
			t.Errorf("TabChordFor(%v) = %v, which is NOT global — it would pass through a full-screen child", a, ChordName(chord))
		}
	}
}
```

- [ ] **Step 2:** `TabChordFor`: `ActionTerminalPrevTab → ChordAltShiftLeft`,
  `ActionTerminalNextTab → ChordAltShiftRight`, `ActionTerminalNewTab →
  ChordAltShiftT`. Run, expect pass. Commit.

### Task 3: `globalBindings` row + regenerate the keymap

- [ ] **Step 1:** add to `globalBindings`: `{Chord: ChordAltShiftT, Action:
  ActionTerminalNewTab, LuaFunction: "PairTermNewTab", NvimKey: "<M-T>",
  FocusDraft: false, HandledInPane: true, Help: "new terminal tab in the right
  pane, from any pane"}`.
- [ ] **Step 2:** regenerate `nvim/workbench_actions.lua`
  (`go run ./cmd/internal/workbenchshortcut/generatecmd -out
  nvim/workbench_actions.lua`, or the Makefile target if one exists — check).
  Confirm it gains the `<M-T>` row. Commit.

### Task 4: pair term `handleTerminalChord` + wrap.go + CLI

- [ ] **Step 1: failing pump test** (in `passthrough_test.go` or a new file):
  under `ownsScreen=true`, delivering each global produces the tab op, NOT a
  passthrough write:

```go
func TestFromAnywhereChordsDriveTheRightTerminalUnderFullScreen(t *testing.T) {
	for _, tt := range []struct{ seq, want string }{
		{"\x1b[1;4D", "prev-tab"}, // Alt+Shift+Left
		{"\x1b[1;4C", "next-tab"}, // Alt+Shift+Right
		{"\x1b[84;4u", "new-tab"},  // Alt+Shift+t
	} {
		mux := &fakeMux{ownsScreen: true}
		pumpStdin(&splitReader{chunks: [][]byte{[]byte(tt.seq)}}, mux, &fakeRuntime{}, io.Discard)
		if got := strings.Join(mux.ops, ","); got != tt.want {
			t.Errorf("%q under fullscreen -> ops=%q, want %q (must NOT pass through)", tt.seq, got, tt.want)
		}
	}
}
```

- [ ] **Step 2:** `handleTerminalChord`: add `case ChordAltShiftT: _ =
  mux.newTab(); return true`. Run the pump test, expect pass.
- [ ] **Step 3:** `wrap.go` — add `ActionTerminalNewTab` to the
  `ActionTerminalPrevTab, ActionTerminalNextTab` case. `RunSwitchTerminalTab` —
  add `case "new": action = ActionTerminalNewTab` and update the usage string to
  `prev|next|new`; extend its test. Run, expect pass. Commit.

### Task 4b: zellij captures `Alt+Shift+t` and delivers the KKP bytes (PQ-1)

Every Alt+LETTER global is bound in `zellij/config.kdl` to `WriteChars` fixed
KKP bytes, because zellij would otherwise capture the Alt+letter for itself
(the arrows needed no bind — zellij forwards them natively, which is why the
regression's Left/Right fix needs no config change). `M-S-t` is a letter, so it
needs the same treatment or a TYPED `M-S-t` never reaches any pane.

**Files:** `zellij/config.kdl`, then the regenerated embedded bundle.

- [ ] **Step 1:** in `zellij/config.kdl`, mirror the `Alt D` rows for `Alt T`:
  `unbind "Alt T"` in the `shared` and `shared_except "locked"` blocks, and
  `bind "Alt T" { WriteChars "\u{1b}[84;4u"; }` in `shared_except`.
- [ ] **Step 2:** regenerate the embedded bundle — `make` regenerates
  `assets/` before tests (`terminalborderless_test.go` notes this), and
  `TestEmbeddedSourcesMatchTree` (`keyhelp` / `runtimebundle`) ties the embedded
  copy to the tree source. Run `env -u PAIR_SESSION_ID -u PAIR_TAG make test`
  and confirm that guard passes. Commit.

### Task 5: init.lua + docs + live

- [ ] **Step 1:** `nvim/init.lua` — add `function _G.PairTermNewTab()
  pair_switch_terminal_tab('new') end` next to Prev/NextTab. (workbench_route's
  argv builder already forwards any direction.) The generated
  `nvim/workbench_actions.lua` gets `["<M-T>"] = {fn="PairTermNewTab"}` from
  Task 3, keyed `["<M-T>"]`. The encoding is PROVEN, not guessed: the
  `ChordAltShiftN` row binds `\x1b[78;4u` to NvimKey `<M-N>` and works live
  (`M-Shift-N` restarts the agent), so `\x1b[84;4u` -> `<M-T>` is the same
  uppercase-letter + mod-4 pattern. If a live mismatch ever appeared it would be
  fixed at the NvimKey spelling, never by adding a byte form.
- [ ] **Step 2:** `keyhelp/catalog.go` — add the `<M-T>` (Alt+Shift+t) row to
  `groupTerminal` (and check the `pair keys` centering test still passes; keep
  the heading short).
- [ ] **Step 3:** `atlas/architecture.md` and `README.md` — document the three
  from-anywhere chords and that they survive a full-screen app (unlike the
  role-scoped `Alt+←/→` the operator types INTO the app, which pass through).
- [ ] **Step 4:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`.
- [ ] **Step 5: live** — rebuild `bin/pair`; from the draft with nvim in the
  right pane, `M-S-Left`/`M-S-Right` switch its tabs and `M-S-t` opens a new one;
  typing `Alt+←`/`Alt+→` inside nvim still moves the nvim cursor. Record in Log.
- [ ] **Step 6:** commit; `sdlc close`.
