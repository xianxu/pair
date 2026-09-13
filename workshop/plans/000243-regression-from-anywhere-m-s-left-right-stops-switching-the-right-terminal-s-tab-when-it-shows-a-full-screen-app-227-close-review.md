# Boundary Review — pair#243 (whole-issue close)

| field | value |
|-------|-------|
| issue | 243 — regression: from-anywhere M-S-left/right stops switching the right terminal's tab when it shows a full-screen app (#227) |
| repo | pair |
| issue file | workshop/issues/000243-regression-from-anywhere-m-s-left-right-stops-switching-the-right-terminal-s-tab-when-it-shows-a-full-screen-app-227.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec3f63eb163fb2215ad0be4e33676720fc581c4c..e2af5b8f38df3f4184b39b8947f2f23edda14be3 |
| command | sdlc close --issue 243 |
| reviewer | claude |
| timestamp | 2026-09-13T11:08:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is the right shape and it is real: `TabChordFor` now returns the three global chords, `handleTerminalChord` acts on the new `ChordAltShiftT`, and the pinned tests fail without it. I verified the pump test under `ownsScreen=true` yields `prev-tab`/`next-tab`/`new-tab` with no passthrough write, the `IsGlobalChord` guard ties delivery to the global set, the generated keymap matches `RenderLuaGlobalMaps`, the embedded bundle matches the tree (uncached run), and a fresh build of HEAD passes `tests/term-pane-shortcuts-test.sh` including the three `;4`/`84;4u` delivery rows. Every row of the plan's Core concepts table exists at the stated path with the stated status; every PQ finding from the plan gate is honored in code (KKP-only sequence, `<M-T>` NvimKey, `Alt T` zellij rows, `namedChord` case). What keeps this off SHIP: the CLI's `new` direction, which is the ONLY path the draft pane uses, has no test (the plan step that promised one was not delivered), and the ticked "live check" has no record in the Log.

**1. Strengths**

- `cmd/internal/workbenchshortcut/shortcut_test.go:681` `TestTabChordForDeliversGlobalChords` pins the CLASS ("delivered chord must be global"), not the instance. Reverting `TabChordFor` to `ChordAltLeft` fails it. This is exactly ARCH-PURPOSE done right.
- `cmd/internal/termcmd/passthrough_test.go:126` exercises the real pump with the real gate (`RightTerminalChordPassesThrough && activeChildOwnsScreen`), so it catches the bug as it actually shipped, not a re-assertion of the mapping.
- `shortcut.go:410-413` KKP-only sequence with the rationale inline; `TestChordAltShiftTDecodesAndNames` asserts the ABSENCE of `\x1bT`, which is the PQ-3 hazard pinned rather than described.
- `wrapcmd/keymap_registry_test.go:39-48` was corrected to expect the delivered chord to equal the pressed one, with the reason. A test that asserted the old `;3` delivery would have protected the regression.
- Docs: README (both the #227 prose and the table), atlas, and `keyhelp` catalog all updated in-range; the `keyhelp` drift tests derive from `GlobalBindings()` so the new row is checked, not just present.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/internal/layoutcmd/layoutcmd.go:198` `case "new"` has zero test coverage.** `TestRunSwitchTerminalTabParsesItsDirection` (`layoutcmd_test.go:248-252`) still lists only `prev`/`next`/error rows; `layoutcmd_test.go` is not in the range. Plan Task 4 Step 3 says "extend its test" and did not. This matters more than a normal gap: the draft pane path is `init.lua:3407 PairTermNewTab → workbench_route argv → pair layout switch-terminal-tab new → RunSwitchTerminalTab`, and that is the operator's primary use case from the issue ("drive the right pane from the draft"). The shell test covers `pair term --test-shortcut` (the `runDecision` path), not this CLI entry. Deleting the `case "new"` line leaves the whole suite green while from-draft `M-S-t` exits 2 into a detached `jobstart` and does nothing. Fix: add `{"new", []string{"new"}, 0, 1}` to the table, and ideally assert the delivered bytes are `\x1b[84;4u` (the table currently counts ops only).
- **Live check ticked without evidence.** The issue's Plan row "… full make test; live check" is `[x]`, and Done-when's last bullet is the live run, but the Log has only the filing entry followed by an empty second `## Log` / `### 2026-09-13` heading. Plan Task 5 Step 5 says "Record in Log". `bin/pair` was rebuilt at 10:58 (between base and head), so a live run was possible, but nothing records that `M-S-left/right/t` drove nvim and that typed `Alt+←/→` still reached nvim. Either record the result (what was pressed, what happened, including the typed-arrow control) or untick the row. This repo's stated value is to have the operator smoke-test before "done".

**4. Minor findings**

- ARCH-DRY: `run.go:604` and `run.go:619` are two identical `_ = mux.newTab()` cases; the file already collapses `ChordAltLeft, ChordAltShiftLeft` into one case at `run.go:608`. Merge into `case ChordAltT, ChordAltShiftT:`.
- `_ = mux.newTab()` discards a `ptychild.Start` error (pre-existing for `Alt+t`, copied here). `mux.reportError` is right there on the interface (`run.go:625` uses it). Cheap to do in the merged case.
- Issue file: duplicate `## Log` heading at the end; fold into one.
- `atlas/architecture.md:477` reads "The test `TabChordFor` returns a chord for which…"; missing "that" or a name, so it parses as prose about a test called `TabChordFor`.
- `nvim/workbench_route_test.lua:72-76` pins `prev`/`next` argv but not `new`. The builder is a generic passthrough, so low value, but a one-line row costs nothing.

**5. Test coverage notes**

- Covered and verified red-without-fix: the delivered-chord class (`IsGlobalChord`), the pump under full-screen for all three globals, the agent-pane delivery table, decode/name/no-legacy for the new chord, generated keymap currency, embedded bundle currency, and the `--test-shortcut` delivery bytes in the shell test.
- Not covered: `RunSwitchTerminalTab("new")` (Important above); `namedChord("alt+shift+t")` is only reached via the shell test, which is fine.
- Full `make test` could not be run here: the pty-child and harness TTY tests fail under the sandbox (`operation not permitted`), the documented sandbox class. All non-pty tests in the five touched packages pass; `keyhelp` and `workbenchshortcut` pass uncached.

**6. Architectural notes** (ARCH pass/flag)

- ARCH-DRY: flag, Minor (duplicate `newTab` cases). Otherwise good: one mapping in `TabChordFor`, one encoding row, three consumers derive.
- ARCH-PURE: pass. `TabChordFor`/`IsGlobalChord`/`DecodeChord` stay pure and are tested without IO; IO remains in the existing delivery seam.
- ARCH-PURPOSE: pass on the fix (class pinned, not instance). Flag, Important, on delivery: the from-draft consumer is the purpose and is the one path without a test.
- ARCH-MOCK: pass. Zellij is behind the existing `Runtime` fake; the shell test's fake `zellij` records the delivered bytes.
- ARCH-CONSTRAINTS: pass. Keystroke path, one chord per event, nothing added to the hot path.
- ARCH-SECURE: pass. No new trust boundary; the new sequence is parsed by the existing prefix decoder and the no-legacy-form test closes the ESC-ambiguity hole PQ-3 named.
- ARCH-ORDER: pass. No new state carried between events; the passthrough decision is a pure predicate over the chord plus one screen-state read, as before.
- ARCH-FUNERAL: pass. Nothing durable is created. `newTab` spawns a child whose lifetime is already owned by the mux's existing EOF path.
- Forward note: `TestEmbeddedSourcesMatchTree` checks `init.lua` and `config.kdl` but not `workbench_actions.lua`, which is also shipped in the bundle. Pre-existing gap, out of scope, worth a follow-up row.

**7. Plan revision recommendations**

- `## Revisions` entry on the plan: Task 4 Step 3's "extend its test" was not delivered in the closing commit. Either add the `new` row (preferred, see Important) and note it, or record why the CLI parser row is deliberately untested.
- Task 5 Step 5 requires the live result in the Log. Add the entry or revise the step and untick the issue Plan row.

```findings
findings:
  - id: new
    severity: Important
    family: consumer-path-untested
    title: |
      RunSwitchTerminalTab's new "new" direction, the only path the draft pane uses, has no test
    detail: |
      layoutcmd_test.go:248-252 still lists prev/next only; plan Task 4 Step 3 promised to extend it. Deleting `case "new"` at layoutcmd.go:198 leaves the suite green while from-draft M-S-t exits 2 silently. Add the row and assert the delivered bytes.
  - id: new
    severity: Important
    family: verification-evidence-recorded
    title: |
      Plan row "live check" is ticked but the Log holds no live-run record
    detail: |
      Done-when's live bullet and plan Task 5 Step 5 ("Record in Log") are unmet; the issue ends in an empty duplicate `## Log` heading. Record what was pressed and observed (including typed Alt+arrows still reaching nvim) or untick the row.
  - id: new
    severity: Minor
    family: duplicate-dispatch-case
    title: |
      ChordAltT and ChordAltShiftT are two identical newTab cases in handleTerminalChord
    detail: |
      run.go:604 and :619; the file already merges Left/ShiftLeft at :608 (ARCH-DRY). Merge, and report the newTab error via mux.reportError instead of discarding it.
  - id: new
    severity: Minor
    family: artifact-hygiene
    title: |
      Issue file has a duplicate empty `## Log` heading; atlas sentence at architecture.md:477 is missing a word
    detail: |
      "The test `TabChordFor` returns a chord for which…" reads as a test named TabChordFor.
```
