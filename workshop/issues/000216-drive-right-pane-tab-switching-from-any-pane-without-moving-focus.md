---
id: 000216
status: working
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-09
estimate_hours: 1.60
started: 2026-09-09T10:45:14-07:00
---

# drive right-pane tab switching from any pane, without moving focus

## Problem

`Alt+Shift+Left/Right` in the draft pane is not worth its keys. The operator
wants those chords to **switch the right pane's tabs from wherever focus
happens to be, and leave focus alone**:

> this allows me to check on status of right pane, switch tabs, while keep
> typing here in the draft pane (which is the command center).

That last clause is the requirement, not a nicety. The draft pane is where the
operator composes; a chord that switches tabs by *moving focus there and back*
would interrupt exactly the typing it is meant to preserve, and would flicker
the cursor through a pane the operator is not looking at.

Today tab switching is `Alt+Left`/`Alt+Right`, handled by `pair term` —
`handleTerminalChord` → `previousTab()`/`nextTab()` — and it only works when the
right pane already has focus.

## Spec

**`Alt+Shift+Left/Right` become workbench globals that drive the right pane's
tabs from any pane, with focus unchanged.**

Three facts make this small rather than speculative, and each is checked:

1. **zellij can write to a pane it is not focused on.** `zellij action
   write-chars` takes `--pane-id` (`zellij 0.44.3 --help`, verified). So the
   chord can deliver `Alt+Left`/`Alt+Right` straight into the terminal pane's
   pty and let `pair term` handle it exactly as if it were focused. No focus
   move, no round trip through the operator's pane.
2. **The repo already has the seam.** `workbenchshortcut.globalBindings` is a
   table of chords handled by whichever pane has focus, and each entry already
   carries a **`FocusDraft bool`** — so "handle this everywhere and do not move
   focus" is an existing, expressible shape rather than a new concept. The two
   new entries set `FocusDraft: false`.
3. **The terminal pane's id is already resolvable.** `Runtime` exposes
   `LastTerminalPaneID()` and `TerminalPaneIDs()`, used by the existing
   split/focus paths.

The bytes to write are the ones zellij already forwards for the focused case —
`Alt+Left` / `Alt+Right` — so `pair term` needs **no change at all**: it cannot
tell the difference between a chord the operator pressed in that pane and one
written into its pty. That is the property worth preserving; anything that adds
a second entry point into tab switching earns a second thing to keep in sync.

**Must not regress:** `Alt+Left`/`Alt+Right` inside the right pane keep working
unchanged, and the chord must be a no-op (not an error, not a focus change) when
there is no terminal pane — `layout2`, or a `layout3` whose right pane has
exited.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module    design=0.10 impl=0.16
item: smaller-go-module    design=0.15 impl=0.24
item: smaller-go-module    design=0.05 impl=0.14
item: lua-neovim           design=0.15 impl=0.24
item: atlas-docs           design=0.05 impl=0.04
item: milestone-review     design=0.00 impl=0.20
design-buffer: 0.15
total: 1.60
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Rows: chord encodings + the pure delivery
argv; the global-binding shape, its two Go executors and the two doc guards;
the CLI verb plus the terminal pane's local path; the nvim side including
deleting `nav_boundary`; atlas; one close review. Frequency
note, per `#201`'s lesson: this is a deliberate operator gesture, so a
subprocess per press is an acceptable cost and no direction here optimises for
it. (`sdlc estimate-source` reports the calibration doc `[stale]`, #127.)

## Plan

Single pass; no separate review boundaries.

- [x] Establish what `Alt+Shift+Left/Right` do TODAY. **Run, and the first
      answer was WRONG — see PQ-1 and `## Revisions`.** The sweep was done with
      the spelling `<M-S-Left>`, which finds only `nvim/scrollback.lua:505`; the
      draft binds them as **`<S-M-Left>`/`<S-M-Right>`** (`init.lua:3552-3555`)
      to `nav_boundary`, a live feature documented in `pair keys`. Corrected
      findings, all three of which the plan below now depends on:
      1. **A live draft feature IS lost** — `nav_boundary`, deleted by operator
         decision. Handled in the next step.
      2. **`zellij/config.kdl` does not bind them** (it only unbinds `Alt Left`
         / `Alt Right`), so nothing is taken from zellij.
      3. **The scrollback viewer's `<nop>` must survive.** `scrollback.lua:505`
         maps `<M-S-Left>`/`<M-S-Right>` to `<nop>` at BUFFER scope in the
         viewer, and it is load-bearing: terminals send these as `ESC <Arrow>`,
         the two bytes can split past `ttimeoutlen`, and the ESC half then fires
         the viewer's exit binding and silently closes it. So the new global
         must not remove it, and Alt+Shift+arrows stay inert inside the
         scrollback viewer — a transient overlay, not the cockpit.
      *Method note: a keymap sweep must search every modifier spelling nvim
      accepts, not the one the searcher happened to think of first.*
- [x] **Delete `nav_boundary` (PQ-1 — operator decision).** `Alt+Shift+arrows`
      are NOT free: `init.lua:3552-3555` binds `<S-M-Left>`/`<S-M-Right>` in
      normal+insert to `nav_boundary(∓1)`, the Home/End of the draft's
      history↔draft↔queue strip, documented in `pair keys` at
      `keyhelp/catalog.go:58-59`. **Operator chose deletion** over rebinding
      (`## Revisions`); single-step `Alt+←/→` is unaffected. Remove those two
      keymaps, both catalog rows, and `nav_boundary`/`ordered_landmarks` if
      nothing else calls them. This is also a correctness requirement, not just
      tidiness: `init.lua:3532` installs the generated global maps BEFORE line
      3552, so the later `keymap.set` wins — adding the bindings without this
      deletion leaves the draft still running `nav_boundary`, which is exactly
      the pane the issue exists for. `TestEveryCatalogEntryStillExists` must
      stay green on the new spelling.
- [x] **Chords + a sentinel (PQ-3).** Add `ChordAltShiftLeft` /
      `ChordAltShiftRight`, encodings `\x1b[1;4D` / `\x1b[1;4C`. Modifier `4` =
      shift+alt, the family the repo already proves live (`ChordAltUp` is
      `\x1b[1;3A`, `ChordAltShiftEnter` is `\x1b[13;4u`); confirm against the
      real terminal at manual-test time rather than trusting the derivation.
      **Both doc guards currently bound the chord space at `chord <=
      ChordAltShiftEnter`** (`run_test.go:1188`, `shortcut_test.go:473`) — the
      last const in the iota block — so appending new chords after it drops them
      from both loops with no failure. Add a `chordMax` sentinel and derive both
      bounds from it, so the NEXT chord cannot repeat this. (Same shape as
      `#171`'s `observationKindCount`; `workshop/lessons.md` already carries the
      rule about hand-maintained enum bounds.)
- [x] **Pure delivery argv (PQ-9).** `DeliverChordArgs(paneID string, chord
      Chord) ([]string, bool)` returning `zellij action write --pane-id <id>
      <decimal bytes…>`. `write`, not `write-chars`: the payload is an escape
      sequence and `write` takes bytes (`zellij 0.44.3 --help`, verified;
      `--pane-id` supported on both). `ChordEncodings` returns a SLICE —
      `ChordAltLeft` has three (`\x1b[1;3D`, `\x1b[1;9D`, `\x1b[3D`) — so this
      emits `ChordEncodings(chord)[0]`, the canonical first entry, and the test
      asserts that specific vector rather than "an encoding". Pure and
      table-tested, keeping `workbenchshortcut`'s no-IO invariant and the byte
      knowledge in the package that owns it (`ARCH-DRY`).
- [x] **Global binding shape — the Spec's fact 2 needs correcting first.**
      `globalBindings` gives "handled from any pane", but every entry's action
      is a **draft Lua call**: `DecideGlobal` sets `DraftLuaFunction`, and both
      Go executors branch on that field before anything else. Delivering bytes
      to the TERMINAL is a different shape. Add `ActionTerminalPrevTab`/`NextTab`
      and a `GlobalBinding` field marking the entry self-handled, so
      `DecideGlobal` returns the action WITHOUT a `DraftLuaFunction` and the Go
      executors act natively instead of round-tripping through nvim.
      `LuaFunction` stays populated — it is what `RenderLuaGlobalMaps` emits for
      the draft's keymap.
- [x] **Terminal-pane resolution reuses the existing picker (PQ-4).** After
      `Alt+Shift+d` there are two `pair term` panes with independent tab sets.
      `LastTerminalPaneStore` / `LiveTerminalPaneIDsFromEnv` are stores, not a
      picker; `layoutcmd.pickRightTerminal` already encodes the tie-break
      (recorded half > zellij focus > pane order, with the reasoning for why
      zellij's `is_focused` is stale memory). Export it — or lift it to a shared
      home — and reuse it, so `Alt+k` and `Alt+Shift+arrow` cannot land on
      different halves (`ARCH-DRY`).
- [x] **Three executors, one implementation of tab switching (PQ-2).** The right
      pane handles the chord in **`handleTerminalChord`** (`run.go:504`) — the
      only seam holding the mux; `runDecision` has none — calling the same
      `mux.previousTab()`/`nextTab()` that `Alt+Left/Right` already call: one
      implementation, two callers. The agent pane resolves the pane and runs the
      delivery argv from `executeWorkbenchDecision`'s Action switch. The draft's
      Lua function shells out to a new `pair term --switch-tab prev|next`, so
      resolution and encoding stay in Go rather than being restated in Lua.
      **Reconciling the two doc guards:** `TestEveryHandledTerminalChordIsDocumented`
      requires every chord `handleTerminalChord` handles to appear in
      `RoleBindings()`, while `TestRoleBindingsCoverTerminalSwitch` already
      exempts globals because "global chords carry their own Help on
      GlobalBinding". Give the termcmd guard the SAME exemption, so `Help` is
      authored once on the `GlobalBinding` and neither table restates it.
- [x] **zellij config.** Bind or unbind as needed so the chord reaches the
      focused pane as bytes, in the SOURCE `zellij/config.kdl` — the mirror
      under `cmd/internal/runtimebundle/assets/runtime/files/` is regenerated by
      `make test` and an edit there is silently reverted.
- [x] **Tests, one strategy line per risky surface.**
      *Chord scanner (PQ-5):* two rows join `chordSequences`, iterated in order
      by `FindChord`, `DecodeChord`, `DecodeChordPrefix` and `IsChordPrefix`,
      which feed the partial-chunk `held` buffer at `run.go:422-489`. Property
      over the table: no sequence is a proper prefix of another; plus
      split-arrival rows delivering `\x1b[1;4` and `D` in separate chunks.
      *Draft path (PQ-6):* `nvim/workbench_route_test.lua` is the existing Lua
      seam — assert the new key routes to the new function; plus argument-parsing
      rows for `pair term --switch-tab` covering `prev`, `next`, missing and
      unknown values.
      *Pure/decision/executor:* delivery argv for both chords; `DecideGlobal`
      returns the action with no `DraftLuaFunction` from all three roles; the
      agent pane emits the complete normalised argv vector (not a prefix); the
      right pane switches tabs with **no** zellij call; `Alt+Left/Right` in the
      right pane untouched; with no terminal pane the chord reports nothing and
      changes no focus. Generated-asset row: the `workbench_actions.lua` mirror
      matches, since `make test` regenerates it.
- [x] **Ordering and failure, stated (PQ-7).** Delivery is fire-and-forget: the
      pane id is resolved then written in a separate `zellij action`, so a
      terminal pane that exits in between yields a stale id and the write fails
      silently — acceptable, because the alternative is an error toast for a
      pane the operator just closed. A rapid double-press is two independent
      subprocesses with no ordering guarantee between them; tab switching is
      idempotent per press, so the only visible effect is arriving one tab off,
      and nothing rolls back.
- [x] **Non-goals (PQ-8).** Not preserving `nav_boundary` under another chord
      (operator chose deletion). Not addressing per-half tab sets after a split —
      the chord drives whichever half the shared picker selects. No absolute
      `--switch-tab <n>` form. `Alt+Left`/`Alt+Right` are untouched.
- [x] **Atlas + `pair keys`.** `Help:` on both entries, and the workbench-chord
      surface in atlas notes these two are delivered to the right pane rather
      than handled in place.
- [ ] **Manual.** Hold focus in the draft, press the chord, confirm the right
      pane's strip moves its bracket and the draft cursor never leaves; confirm
      the scrollback viewer still exits cleanly on `Esc`; confirm the real
      terminal's byte encoding matches `\x1b[1;4D`/`\x1b[1;4C`.

## Done when

- `Alt+Shift+Left/Right` switch the right pane's tabs from the draft pane, the
  agent pane, and the right pane itself.
- Focus is provably unchanged — the operator can keep typing across the chord.
- The right pane's own `Alt+Left`/`Alt+Right` still work.
- With no terminal pane the chord does nothing and reports nothing.
- `pair keys` / `Alt+h` document the new chords.

## Log

- 2026-09-08 — filed from the operator's request during `#199` M4 acceptance.
  Depends on `#199`'s strip only for the *feedback* (the bracket moving is how
  the operator sees the switch land); the mechanism is independent of it.

## Revisions

### 2026-09-09 — Spec fact 2 corrected: globalBindings is not the whole seam

**Reason.** The Spec says `globalBindings` already expresses "handle this
everywhere and do not move focus", so the change is two table entries. Half of
that is right: the table does give any-pane handling, and `FocusDraft` does
exist. But every entry's *action* is a draft Lua call — `DecideGlobal`
(`shortcut.go:288-298`) returns `DraftLuaFunction`, and both Go executors
(`termcmd/run.go:183`, `wrapcmd`'s `executeWorkbenchDecision`) branch on that
field first and route the chord into nvim. Delivering bytes to the *terminal*
pane cannot be expressed that way, so two table entries would route tab
switching through the draft pane and make it depend on nvim being alive.

**Delta.** Adds a self-handled action shape to `GlobalBinding` + `DecideGlobal`,
and names the three executors explicitly. The Spec's mechanism is unchanged and
still the point: write the `Alt+Left`/`Alt+Right` bytes into the terminal pane's
pty so `pair term` cannot tell a written chord from a pressed one, leaving
exactly one implementation of tab switching. `write` replaces the Spec's
`write-chars` because the payload is an escape sequence.

**Also settled (plan step 1, now checked rather than assumed):** nothing is lost
by rebinding — the chords are unbound in `zellij/config.kdl` and the only nvim
mapping is the scrollback viewer's defensive `<nop>`, which must survive.

### 2026-09-09 — nav_boundary deleted, not rebound (operator decision)

**Reason.** Plan-quality PQ-1 (Critical) caught that `Alt+Shift+arrows` are a
live, documented draft feature — `nav_boundary`, the Home/End of the
history↔draft↔queue strip — which this issue's first plan step had recorded as
free. The step was run, but with the wrong spelling (`<M-S-Left>` rather than
`<S-M-Left>`), so it found only the scrollback viewer's `<nop>` and concluded
nothing was lost. It was wrong, and rebinding versus deleting is the operator's
call rather than the implementer's.

**Delta.** Offered rebind-to-`Ctrl+Alt+arrows`, delete, or rebind-to-
`Alt+Shift+↑/↓`; operator chose **delete**. Single-step `Alt+←/→` is unaffected,
so what is lost is the one-keystroke jump to the ends of the strip. Recorded as
a non-goal so a later reader does not restore it by accident.

### 2026-09-09 — implementation

**Three deviations from the plan, each because the code said so.**

1. **The CLI verb is `pair layout switch-terminal-tab prev|next`, not
   `pair term --switch-tab`.** `layoutcmd` already has the exact `Runtime` this
   needs (`ListPanesJSON` + `LastTerminalPaneID` + `TerminalPaneIDs` +
   `RunZellijAction`) and the picker, and the verb sits next to
   `layout focus-terminal` which resolves the same pane the same way. Putting it
   in `termcmd` would have meant a `termcmd → layoutcmd` dependency for a
   command that never touches the terminal process.
2. **PQ-4 resolved WITHOUT exporting `pickRightTerminal`.** The plan said
   "export it — or lift it to a shared home". Neither was needed: putting
   `SwitchRightTerminalTab` in `layoutcmd` alongside `FocusRightTerminal` lets
   both call the unexported picker in place. Strictly smaller than the plan, and
   it keeps the tie-break reasoning in one file.
3. **No `zellij/config.kdl` change — measured, not assumed.** `zellij setup
   --dump-config` shows the only shift+alt default is `Alt Shift p`
   (`ToggleGroupMarking`); there is no shift+alt arrow binding to unbind, so the
   chord already reaches the focused pane as bytes. Worth stating why this was
   measured rather than "just added an unbind to be safe": `#213` established
   that zellij 0.44.3's config parser **silently ignores unknown keys**, so an
   unbind of a binding that does not exist would validate identically to one
   that does and prove nothing.

**Mutation sweep** — baseline GREEN, every deliverable reddens when removed:

| mutation | result |
|---|---|
| `DecideGlobal` drops the `HandledInPane` branch (routes through the draft) | RED |
| right pane stops handling the chord in `handleTerminalChord` | RED |
| agent pane loses its `executeWorkbenchDecision` case | RED |
| delivery uses `write-chars` instead of `write` | RED |
| picker loses the recorded-half preference | RED |
| a chord appended PAST the `chordMax` sentinel | RED |

That last row is the PQ-3 class itself, and it needed a test written for it:
moving the two guards onto the sentinel does not pin the sentinel, because both
new chords are globals that the guards exempt anyway.
`TestChordMaxFollowsEveryEncodedChord` asserts the sentinel against the encoding
table rather than against the name of the current last chord — the mistake the
old `<= ChordAltShiftEnter` bounds made.

**The sweep harness caught itself first.** Its first run reported baseline RED
along with all six mutations, which is the failure shape `workshop/lessons.md`
already warns about: zsh does not word-split unquoted parameters, so the package
list expanded as a single bogus path and `go test` reported `setup failed` for
everything. The baseline assertion is what made that visible instead of six
false REDs reading as success.

**Evidence.** Full `make test` green unsandboxed (EXIT=0, zero FAIL lines),
including the regenerated `workbench_actions.lua` mirror. The draft path is
covered at the Lua seam (`nvim/workbench_route_test.lua` asserts both new keys
route to `PairTermPrevTab`/`PairTermNextTab` with `focus = false` — the whole
requirement) and at the CLI (`RunSwitchTerminalTab` argument rows: `prev`,
`next`, missing, unknown, too many). Split-arrival rows deliver `\x1b[1;4` and
`D` in separate reads, since a chord lost across a chunk boundary reads as a
dead key.

**Still operator-pending, deliberately unticked:** the live gesture. The byte
encoding `\x1b[1;4D`/`\x1b[1;4C` is derived from the shift+alt family the repo
already proves live (`ChordAltShiftEnter` = `\x1b[13;4u`, `ChordAltUp` =
`\x1b[1;3A`), and nvim's own `<S-M-Left>` mapping resolved from the same bytes
before this change — but derived is not measured, and only a real keypress
confirms it.

### 2026-09-09 — boundary review round 1: FIX-THEN-SHIP, addressed

**BR-1 — the README was the one consumer of the chord surface that does not
derive, and it was not swept.** `keyhelp` derives its wording from
`GlobalBinding.Help`, so the catalog moved with the code; README's chord table
is a hand-maintained narrative restatement, and it still documented the deleted
`nav_boundary`. Removed that row and added one for the new chords beside the
existing terminal-tab row. Same class as `#171`'s BR-17/BR-20: when a surface
changes, the non-deriving restatements are the ones that go stale silently.

**BR-2 — the modifier FAMILY, not just the two new rows.** Terminals disagree
about how to report Alt: bit 2 gives modifier 3, bit 8 ("meta") gives 9, and
adding shift gives 4 and 10. `ChordAltLeft`/`Right` have carried both spellings
since e6eee5a3; the new chords had only modifier 4, so on a meta-style terminal
they would work from the draft (nvim resolves `<S-M-Left>` itself) while being
silently dead in the agent and terminal panes — a partial failure of "from any
pane" that a single-terminal manual test cannot rule out. Added `\x1b[1;10D` /
`\x1b[1;10C`, **and** the missing `\x1b[1;9A` / `\x1b[1;9B` for `ChordAltUp` /
`ChordAltDown`, which had the same gap for the same reason. That is the class:
both modifier families for every arrow chord, rather than per-chord accident.
`TestNoChordSequenceIsAProperPrefixOfAnother` guards the shadowing risk.

**BR-6 — one fact for "prev means Alt+Left".** The Action→chord mapping was
restated in three executors. `workbenchshortcut.TabChordFor` now owns it and all
three route through it, including the CLI's `prev|next` parsing.

**BR-4 — `pos_rank` was orphaned** by the `nav_boundary` deletion; the plan's
deletion step named `nav_boundary`/`ordered_landmarks` but not it. Removed; no
mentions remain. No Lua linter runs in `make test`, so nothing would have caught
this — worth remembering that Lua dead code is unguarded here.

**BR-5 — my comment credited a column that does not exist.** `catalog.go`
justified grouping the two globals under the terminal-tab heading by claiming
"the per-row context column is what distinguishes them". `pair keys` renders no
context column: `Context.String()` exists but nothing displays it, so `Context`
feeds classification only. The distinction really comes from the Help wording,
which does hold. Comment corrected to say that instead. I asserted the column
from the existence of a `String()` method without checking a renderer used it.

**BR-3 — not a defect; recorded rather than "fixed".** The finding says
`DeliverChordArgs` ranges a string and yields runes. It does not:
`ChordEncodings` returns `[][]byte` (`shortcut.go:387`), so `encodings[0]` is
`[]byte` and the loop variable is already a `byte`. `range []byte(encodings[0])`
would be a redundant conversion. Left as is deliberately — making a no-op edit
to close a finding would misreport what happened.

### 2026-09-09 — boundary review round 2: rules, not instances

Round 1's fixes were instances. Round 2 said so, correctly, in three families.

**BR-7 — the four meta encodings were unpinned.** Deleting all four left the
whole suite green, so BR-2 was "fixed" by four table rows that nothing checked.
Replaced by the rule they are instances of, checked mechanically over
`chordSequences`: for every arrow chord encoded `\x1b[1;<m><letter>`, the meta
sibling `\x1b[1;<m+6><letter>` must also be registered (alt 3 → meta 9,
shift+alt 4 → shift+meta 10). Verified to redden — removing `\x1b[1;10D` gives
`"\x1b[1;4D" (21) has no meta sibling "\x1b[1;10D" — the chord is dead on a
meta-style terminal`.

**BR-8 — the deletion's reference set, enumerated rather than chased.** The rule
the finding asks for: a deletion is complete only when the full reference set is
swept — bodies, callers, comments describing the behaviour, `atlas/` — via an
enumeration that is run and recorded, because no compiler and no `make test`
step sees any of it. The enumeration:

```
grep -rniE "nav_boundary|ordered_landmarks|pos_rank|region boundary|landmark" \
  --include="*.lua" --include="*.md" --include="*.go" . | grep -v '^./workshop/'
```

It found **four** stale sites, two of which neither review round named:
`nvim/init.lua` (the Boundary-jump comment block, still describing the deleted
behaviour as live), `atlas/architecture.md:955` (the helper list), and — missed
by both the reviewer's enumeration and mine until this run —
`atlas/architecture.md:886`, a keybinding row still documenting the region-jump.
Also `CHANGELOG.md:193`.

**The CHANGELOG entry is deliberately NOT swept**, which refines the rule: it is
a dated record of what shipped in a past release, not a description of current
behaviour. Rewriting it would falsify history. The enumeration has to
distinguish current-state docs (sweep) from dated records (leave) — otherwise
"sweep everything" turns a changelog into fiction. Re-running the grep at head
returns only deliberate historical references.

**BR-9 — the `runDecision` case had no test.** `handleTerminalChord`
short-circuits these chords inside the right pane, so `pair term
--test-shortcut` is the case's only reachable caller, and
`tests/term-pane-shortcuts-test.sh` — the fake-zellij harness that covers every
other terminal chord — had not been extended. Three rows added, driven from the
DRAFT's focus since that is the point: both directions deliver the right bytes,
and a third asserts **no focus action is emitted at all**.

**BR-10** — `resolveRightTerminal` extracts the four-step preamble both callers
re-typed. The ARCH-DRY guarantee was only half delivered: the picker was shared,
its inputs were re-derived, so a fourth preference signal would have landed at
one site and missed the other.

**BR-11** — my atlas claim was too strong. From inside a split half the pump
switches THAT half regardless of the recorded id, so `Alt+k` and
`Alt+Shift+arrow` can point at different halves there. Scoped the guarantee to
the delivery path, which is where it actually holds.

**BR-12** — `pair_bin()` ends the `PAIR_HOME .. '/bin/pair'` idiom's five
copies. **BR-13** — the draft's argv is now a pure
`switch_terminal_tab_command` pinned by `workbench_route_test.lua`; previously
the key→function mapping was covered and what that function *ran* was not.

**A self-inflicted detour worth recording:** placing `pair_bin()` by script put
it inside an `if` block, and `make test` reported exit 2 with **zero** FAIL
lines — which reads as infrastructure, not syntax. `luac -p nvim/init.lua` named
the file, line and unclosed block instantly. Added to `workshop/lessons.md`.

### 2026-09-09 — boundary review round 3: the chain, not a third instance

**BR-14** — third finding in `untested-executor-branch`. Rounds 1 and 2 fixed
instances (the agent pane's case, then `termcmd`'s). The rule: **a chord is
delivered by a CHAIN, and every process- or language-boundary in that chain
needs a test that CROSSES it, not two tests that stop on either side.** The
draft's chain is Lua → `pair layout switch-terminal-tab` → dispatcher case → Go,
and both boundaries were string contracts nothing type-checks. Measured by the
reviewer: renaming the Dispatch case left `go test ./...` byte-identical to
control, failing only at runtime.

Two mechanical checks, both crossing rather than stopping:

- `TestEveryImplementedFamilyIsRoutable` — every `Status:"implemented"` family
  must have a routing case. Read from **source**, not by calling `Dispatch`,
  because calling it executes the command; a routability check that launches
  `wrap` or `term` is not a test.
- `TestDraftLuaSubcommandsAreDeclaredAndRoutable` — extracts the argv
  `nvim/workbench_route.lua` actually builds and asserts Go both declares and
  routes it. This is the boundary no compiler sees.

**The first cut of the routability test was wrong, and its own failure said so.**
It scanned only `dispatcher.go` and reported eight implemented families as
unroutable — `wrap`, `term`, `hoprtt`, `scribe`, `session-watch`, `session-log
append`/`commit`, `title`. They are routed in `cmd/pair-go/main.go`: the
STREAMING commands never reach the buffered switch. So there are two routers,
and a scan of one reports half the surface as broken. Both are scanned now.
Worth keeping: a source-scanning guard's first failure is as likely to be the
scan's own premise as the code's.

Verified by mutation: renaming the Dispatch case reddens, and so does renaming
the subcommand on the Lua side.
