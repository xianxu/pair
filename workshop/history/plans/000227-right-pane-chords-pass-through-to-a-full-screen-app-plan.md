# Right-pane chord passthrough under a full-screen app — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When the active `pair term` tab's child owns the screen (it is on the
alternate screen buffer), the pane-local workbench chords pass through to the
child instead of being intercepted; `pair term` keeps intercepting them at a
shell, and global chords fire in every state.

**Architecture:** All recognised chords funnel through ONE site in the stdin
pump (`run.go`), which today consumes every one and dispatches it (tab action,
focus, swallow) — the child never sees a recognised chord. This plan adds a
single gate there: when the active child owns the screen AND the chord is not
global, forward the chord's raw bytes to the child and dispatch nothing. The
"is it global" test is a pure predicate over the chord table; the "does the
child own the screen" bit is read from the active child's `RepaintModes()`
tri-state. Global chords are resolved by the same predicate and always fire.

**Deviation from the issue's sketch, with rationale.** #227's Spec preferred
adding an alt-screen field to `ShortcutInput` and having `Decide` return
pass-through. That covers only the chords dispatched through `Decide`
(`handleChord`); the tab chords go through `handleTerminalChord` and the rename
through an inline branch, both BYPASSING `Decide`. A `ShortcutInput` field would
therefore leave the handled tab chords (`M-t`, `M-w`, `M-Left/Right`, `M-S-d`,
`M-r`) intercepting under a full-screen app — the exact chords the operator
reported. So the gate lives in the pump, the one place all three dispatch paths
funnel through, keyed on a pure `IsGlobalChord` predicate. `Decide` stays
pure and UNTOUCHED (ARCH-PURE), and the chord classification stays
table-testable in `workbenchshortcut`. This is recorded as a `## Revisions`
entry on the issue.

**Tech Stack:** Go. `cmd/internal/workbenchshortcut` (the pure predicate),
`cmd/internal/termcmd` (the pump gate + the active-child accessor),
`cmd/internal/keyhelp` (help text), `atlas/`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `IsGlobalChord` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `RightTerminalChordPassesThrough` | `cmd/internal/workbenchshortcut/shortcut.go` | new |

- **IsGlobalChord** — `IsGlobalChord(chord) bool`, true iff the chord is a
  workbench-wide global (the ones `DecideGlobal` resolves). Thin, pure wrapper
  over the existing `globalDraftAction` lookup, so "global" has one definition.
  - **DRY rationale:** `DecideGlobal(chord)` already returns `(_, ok)` with ok
    == "is global". `IsGlobalChord` names that boolean so nothing open-codes
    `_, ok := DecideGlobal(chord)` and reads `ok` as a classification.

- **RightTerminalChordPassesThrough** — `RightTerminalChordPassesThrough(chord)
  bool`, the passthrough gate's classifier: true iff a right-terminal chord is
  forwarded to a full-screen child rather than intercepted. It is
  **`!IsGlobalChord(chord) && chord != ChordAltK`**.
  - **Why `M-k` is excluded (PQ-1).** `M-k` in the right terminal is
    `ActionFocusPane` — the ONLY keyboard bridge back to the left stack
    (`shortcut.go:192`), and there is NO global focus-left (checked
    `shortcut.go` global table) and zellij's default MoveFocus keys are all
    unbound (`zellij/config.kdl`). If `M-k` passed through to nvim, the
    operator running nvim+parley on the right — the reported workflow — would
    have no keyboard path back to the agent pane at all. Navigation out of the
    workbench's panes is not a convenience the operator can forgo, unlike tab
    management. So `M-k` always intercepts, in every alt-screen state.
  - **Relationships:** consumed only by the pump gate. Pure, table-tested over
    the whole chord set.
  - **Future extensions:** if a second escape/navigation chord is added, it
    joins the exclusion here — the one place the "must survive a full-screen
    app" set is written.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `activeChildOwnsScreen` | `cmd/internal/termcmd/run.go` | new | active tab's `ptychild.Child` |
| pump passthrough gate | `cmd/internal/termcmd/run.go` | modified | stdin chord dispatch |

- **activeChildOwnsScreen** — `func (m *terminalMux) activeChildOwnsScreen()
  bool`, reads the active tab's `child.RepaintModes()` and returns
  `altScreen && observed`. The #196 tri-state, consumed exactly as #227 asks:
  `RepaintModes()`, not the bare `AltScreen()`, so the unknown case (no child,
  or a child that never spoke about the buffer) decides to INTERCEPT, not to
  "off". The three states:

  | observed | altScreen | result | chord handling |
  |---|---|---|---|
  | yes | on | `true` | pass through |
  | yes | off | `false` | intercept (today) |
  | no | — | `false` | intercept (today) |

  - **Injected into:** the pump gate. Nil-safe: no active tab or nil child →
    `false` (intercept), the no-regression choice.
  - **Future extensions:** none.

- **pump passthrough gate** — in `pumpStdinWithTimer`, right after a chord is
  found and `chordBefore` is flushed, before the rename/tab/handleChord
  dispatch: if `RightTerminalChordPassesThrough(chord) &&
  m.activeChildOwnsScreen()`, write the chord's raw bytes to the active child and
  `continue`; else dispatch as today.
  The raw bytes come from `FindChord`'s third return (currently discarded as
  `_`). Forwarding the raw bytes is also what closes #234's residual: `ESC`,`j`
  typed inside 35 ms decodes as `ChordAltJ` whose raw bytes are `\x1bj`, so
  forwarding them delivers ESC then j to nvim, exactly what the user typed.

### The escape-chord decision (recorded, per #227)

**The full set that passes through under a full-screen app** is the
role-scoped right-terminal chords MINUS `M-k`: handled — `M-t` (new tab), `M-w`
(close tab), `M-r` (rename), `M-S-d` (split), `M-S-Enter` (toggle width),
`M-Left`/`M-Right` (prev/next tab); swallowed — `M-j`, `M-/`, `M-S-c`,
`C-M-c`. **`M-k` (focus-left) does NOT pass through** — it is the keyboard
escape back to the left stack and is excluded (see the entity above).

What the operator loses inside a full-screen app, and the surviving fallback:
- **create / close a tab** (`M-t`/`M-w`): leave the app first. No global
  equivalent; this is the one real restriction.
- **switch tabs** (`M-Left`/`M-Right`): survives — the from-anywhere
  `M-S-Left` / `M-S-Right` are GLOBAL (`HandledInPane`) and fire under a
  full-screen app.
- **focus the left stack** (`M-k`): survives — `M-k` is excluded from
  passthrough, so it still jumps to the agent pane even with nvim up.

**No always-available create-tab escape chord is added** (#227's default):
switching and focus-left both survive as above, so only tab create/close needs
leaving the app, and every always-on chord is keyboard real estate taken from
every app. Add one only if the create-from-fullscreen restriction is felt.
Recorded here and in the issue's `## Revisions`.

### ARCH notes

- **ARCH-DRY:** one definition of "global" (`IsGlobalChord`), one gate for all
  three dispatch paths.
- **ARCH-PURE:** `Decide` untouched and still pure; the new classifier is pure
  and table-tested; the only IO is the `RepaintModes()` read behind the mux.
- **ARCH-ORDER:** the pump carries no NEW state; the gate is a pure function of
  (chord, active child's tri-state) evaluated per chord. The one ordering that
  matters — a chord arriving while the child is mid-transition into/out of the
  alt screen — resolves to whatever `RepaintModes()` reads at that instant, and
  either answer is a legal per-keystroke outcome (the operator gets passthrough
  or interception for that one key, never a wedged state).
- **ARCH-FUNERAL / ARCH-CONSTRAINTS / ARCH-SECURE / ARCH-MOCK:** N/A — creates
  nothing durable, adds no unbounded work (one map lookup + one locked read per
  chord, off the hot byte path), no new trust surface, no external double.

---

## Chunk 1: passthrough

### Task 1: `IsGlobalChord` (pure)

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Modify: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [ ] **Step 1: Failing test** — every `globalBindings` chord is global, a
  sample of role/unknown chords is not:

```go
func TestIsGlobalChordMatchesTheGlobalTable(t *testing.T) {
	for _, b := range GlobalBindings() {
		if !IsGlobalChord(b.Chord) {
			t.Errorf("IsGlobalChord(%v) = false, want true (it is in globalBindings)", ChordName(b.Chord))
		}
	}
	for _, c := range []Chord{ChordAltT, ChordAltW, ChordAltR, ChordAltJ, ChordAltK, ChordAltShiftEnter, ChordUnknown} {
		if IsGlobalChord(c) {
			t.Errorf("IsGlobalChord(%v) = true, want false (role-scoped/unknown)", ChordName(c))
		}
	}
}

func TestRightTerminalChordPassesThrough(t *testing.T) {
	// role-scoped, non-navigation: pass through
	for _, c := range []Chord{ChordAltT, ChordAltW, ChordAltR, ChordAltShiftD, ChordAltShiftEnter, ChordAltLeft, ChordAltRight, ChordAltJ, ChordAltSlash, ChordAltShiftC, ChordCtrlAltC} {
		if !RightTerminalChordPassesThrough(c) {
			t.Errorf("%v should pass through to a full-screen child", ChordName(c))
		}
	}
	// M-k is the keyboard escape back to the left stack: never passes through
	if RightTerminalChordPassesThrough(ChordAltK) {
		t.Error("ChordAltK (focus-left) must NOT pass through — it is the only keyboard bridge to the left stack")
	}
	// globals fire, never pass through
	for _, b := range GlobalBindings() {
		if RightTerminalChordPassesThrough(b.Chord) {
			t.Errorf("global %v must not pass through", ChordName(b.Chord))
		}
	}
}
```

- [ ] **Step 2: Run, expect fail** (undefined). `go test ./cmd/internal/workbenchshortcut/ -run TestIsGlobalChord`
- [ ] **Step 3: Implement** next to `DecideGlobal`:

```go
// IsGlobalChord reports whether chord is a workbench-wide global — one
// DecideGlobal resolves. The passthrough gate (#227) forwards a non-global
// chord to a full-screen child but always lets a global fire; this is the one
// definition of that split, so the gate need not re-derive it.
func IsGlobalChord(chord Chord) bool {
	_, ok := globalDraftAction(chord)
	return ok
}

// RightTerminalChordPassesThrough reports whether a right-terminal chord is
// forwarded to a FULL-SCREEN child rather than intercepted by pair term (#227).
// True for the role-scoped chords EXCEPT ChordAltK: M-k is the only keyboard
// path from the right terminal back to the left stack (ActionFocusPane), and
// there is no global equivalent, so it must keep firing even under a
// full-screen app or the operator running nvim on the right is trapped. Globals
// are never passed through — they are workbench-wide and fire in every state.
func RightTerminalChordPassesThrough(chord Chord) bool {
	return chord != ChordUnknown && !IsGlobalChord(chord) && chord != ChordAltK
}
```

- [ ] **Step 4: Run, expect pass.** Commit.

### Task 2: `activeChildOwnsScreen` (the mux accessor)

**Files:**
- Modify: `cmd/internal/termcmd/run.go`
- Modify: `cmd/internal/termcmd/run_test.go`

- [ ] **Step 1: Failing test** — a fake child on the alt screen (observed) →
  true; off → false; never-observed → false. `NewFakeChild([]byte("\x1b[?1049h"))`
  puts it on the alt screen (the Screen scans it):

```go
func TestActiveChildOwnsScreenTriState(t *testing.T) {
	for _, tt := range []struct {
		name  string
		feed  []byte
		want  bool
	}{
		{"on alt screen", []byte("\x1b[?1049h"), true},
		{"returned to primary", []byte("\x1b[?1049h\x1b[?1049l"), false},
		{"never spoke about the buffer", []byte("plain shell output"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mux := &terminalMux{tabs: []*terminalTab{{id: 1, child: ptychild.NewFakeChild(tt.feed)}}, active: 0}
			if got := mux.activeChildOwnsScreen(); got != tt.want {
				t.Fatalf("activeChildOwnsScreen() = %v, want %v", got, tt.want)
			}
		})
	}
	// No active tab / nil child: intercept.
	if (&terminalMux{active: -1}).activeChildOwnsScreen() {
		t.Fatal("no active tab must not own the screen")
	}
}
```

- [ ] **Step 2: Run, expect fail.**
- [ ] **Step 3: Implement** next to `appMouseMode`:

```go
// activeChildOwnsScreen reports whether the active tab's child is a full-screen
// app — on the alternate screen buffer, observed to be so. Under it, role-
// scoped chords pass through to the child (#227). RepaintModes(), not the bare
// AltScreen(): the unknown case (nil child, or one that never spoke about the
// buffer) must read as "not full-screen" so pair term keeps intercepting — the
// no-regression choice, and the tri-state #196 already enforces.
func (m *terminalMux) activeChildOwnsScreen() bool {
	m.mu.Lock()
	tab := m.activeTabLocked()
	m.mu.Unlock()
	if tab == nil || tab.child == nil {
		return false
	}
	altScreen, observed := tab.child.RepaintModes()
	return altScreen && observed
}
```

- [ ] **Step 4: Run, expect pass.** Commit.

### Task 3: The pump gate (green)

**Files:**
- Modify: `cmd/internal/termcmd/run.go` (the chord branch in `pumpStdinWithTimer`)
- Create: `cmd/internal/termcmd/passthrough_test.go`

- [ ] **Step 1: The test seam.** Add `activeChildOwnsScreen() bool` to the
  `ptyWriter` interface and to `fakeMux` as a settable field `ownsScreen`
  (false zero value, so every existing pump test is a "shell" and is
  unaffected). The gate then calls `mux.activeChildOwnsScreen()` uniformly:
  the real `terminalMux` reads `RepaintModes()`, the fake returns its field.
  The exhaustive chord × state table lives in Task 4; the two explicit
  regressions below are the ones worth spelling out by hand.

- [ ] **Step 2: The two explicit regressions**

```go
// #234's residual: ESC,j typed inside the deadline decodes as ChordAltJ. Under
// a full-screen child its raw bytes \x1bj now reach the child = ESC then j,
// which is what the user typed. At a shell it still fires the focus chord.
func TestEscThenJReachesAFullScreenChildAsTwoKeys(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1bj")}}, mux, &fakeRuntime{}, io.Discard)
	if got := strings.Join(mux.ops, ","); got != "write:\x1bj" {
		t.Fatalf("ops = %q, want the Alt+j bytes forwarded to the full-screen child", got)
	}
	shell := &fakeMux{ownsScreen: false}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1bj")}}, shell, &fakeRuntime{}, io.Discard)
	// At a shell Alt+j is the focus-left chord: swallowed by pair term (its
	// disposition), never written to the child.
	for _, op := range shell.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("shell ops = %v: Alt+j must not reach the child at a shell", shell.ops)
		}
	}
}

// M-k (focus-left) is the keyboard escape and must NOT pass through even under
// a full-screen child, or the operator with nvim focused on the right has no
// keyboard path back to the agent pane (PQ-1).
func TestFocusLeftNeverPassesThroughToAFullScreenChild(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[107;3u")}}, mux, &fakeRuntime{}, io.Discard) // Alt+k
	for _, op := range mux.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("ops = %v: M-k must fire focus-left, never reach the full-screen child", mux.ops)
		}
	}
}


// A global fires even under a full-screen child.
func TestGlobalChordFiresUnderAFullScreenChild(t *testing.T) {
	mux := &fakeMux{ownsScreen: true}
	pumpStdin(&splitReader{chunks: [][]byte{[]byte("\x1b[110;3u")}}, mux, &fakeRuntime{}, io.Discard) // Alt+n restart (global)
	for _, op := range mux.ops {
		if strings.HasPrefix(op, "write:") {
			t.Fatalf("ops = %v: a global must not pass through to the child", mux.ops)
		}
	}
}
```

- [ ] **Step 3: Run, expect fail** (child gets nothing today; every chord consumed).
- [ ] **Step 4: Implement the gate.** Add `activeChildOwnsScreen() bool` to the
  `ptyWriter` interface and `fakeMux` (settable `ownsScreen`); capture
  `chordRaw` from `FindChord`; insert the gate:

```go
chordBefore, chord, chordRaw, chordRest, chordOK := workbenchshortcut.FindChord(data)
...
if chordOK && (!mouseOK || len(chordBefore) <= len(mouseBefore)) {
	if len(chordBefore) > 0 {
		mux.writeActive(chordBefore)
	}
	// #227: under a full-screen child, a role-scoped chord is the app's, not
	// pair term's. Forward its raw bytes and dispatch nothing. Globals are
	// workbench-wide and still fire; the rename/tab/handleChord dispatch below
	// is the "at a shell" path.
	if workbenchshortcut.RightTerminalChordPassesThrough(chord) && mux.activeChildOwnsScreen() {
		mux.writeActive(chordRaw)
		data = chordRest
		continue
	}
	if chord == workbenchshortcut.ChordAltR { ... }   // unchanged, now only reached when intercepting
	...
}
```

- [ ] **Step 5: Run, expect pass.** Run the whole package (the existing chord
  tests use a default `fakeMux{ownsScreen:false}`, so they are unaffected — add
  the field with a false zero value). Commit.

### Task 4: Table test over the full chord set × three states

**Files:**
- Modify: `cmd/internal/termcmd/passthrough_test.go`

- [ ] **Step 1:** Generated from `ChordSequences()`: for each chord, assert
  under `ownsScreen:true` the raw bytes reach the child iff `!IsGlobalChord`,
  and under `ownsScreen:false` the chord dispatches as today (no `write:` for a
  handled/swallowed chord). The "unknown" alt-screen state is the real mux's
  `activeChildOwnsScreen()==false` path, covered by Task 2's tri-state test;
  note that here so the coverage is traceable, not duplicated.
- [ ] **Step 2: Run, expect pass.** Commit.

### Task 5: Help text + atlas + the escape-chord decision

**Files:**
- Modify: `cmd/internal/keyhelp/catalog.go` (or the `RoleBinding.Help` strings) — right-pane chords are CONDITIONAL
- Modify: `atlas/architecture.md`
- Modify: the issue (Plan ticks, Log, `## Revisions` for the design deviation + the escape-chord decision)

- [ ] **Step 1:** `pair keys` / Alt+h must say the right-terminal chords pass
  through to a full-screen app (and that `M-k` still escapes). The one edit site
  is the terminal group HEADING — the `groupTerminal` constant in
  `keyhelp/catalog.go:10` — not the entries, which carry no wording
  (`catalog.go:16`). Append the note there. Check `keyhelp/drift_test.go` still
  passes (it classifies every keymap/chord).
- [ ] **Step 2:** Atlas paragraph under the `pair term` input section: the
  passthrough rule, the alt-screen definition, `RepaintModes` tri-state,
  globals unaffected, and that this closes #234's inside-deadline residual.
- [ ] **Step 3:** Issue `## Revisions` AND rewrite the superseded text in
  place: the Done-when bullet "alt-screen state arrives as an input field" and
  the `## Plan` rows that describe the `ShortcutInput`/`Decide` design must be
  rewritten to the pump-gate design, or the close review fails the issue against
  its own text. Record: the deviation (tab chords bypass Decide → pump gate);
  the `M-k` exclusion; the escape-chord decision (none — switching and
  focus-left both survive).
- [ ] **Step 4: Full suite.** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`.
- [ ] **Step 5: Live.** Rebuild; in the right pane run nvim/parley: `M-t`
  reaches parley's outline, `M-j`/`M-k` reach nvim, ESC+`j` reaches nvim as two
  keys, `M-S-Left`/`M-S-Right` still switch tabs, `M-n` still raises the reload
  confirm; at a shell prompt `M-t` still opens a tab. Record in Log.
- [ ] **Step 6: Commit; `sdlc close`.**
