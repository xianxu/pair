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

---

## Re-review — 2026-09-13T11:19:00-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 243 — regression: from-anywhere M-S-left/right stops switching the right terminal's tab when it shows a full-screen app (#227) |
| repo | pair |
| issue file | workshop/issues/000243-regression-from-anywhere-m-s-left-right-stops-switching-the-right-terminal-s-tab-when-it-shows-a-full-screen-app-227.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec3f63eb163fb2215ad0be4e33676720fc581c4c..bb59dc6c304b198ff100ca884ed23ef601558fec |
| command | sdlc close --issue 243 |
| reviewer | claude |
| timestamp | 2026-09-13T11:19:00-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is right and the pinning tests are real. In a scratch copy of HEAD I deleted the CLI's `case "new"` and both RunSwitchTerminalTab tests went red; I removed `ChordAltShiftT` from the merged `handleTerminalChord` case and the pump test went red with `ops="" want "new-tab"`. The delivered-chord class is pinned by `IsGlobalChord`, the generated keymap and the embedded bundle match the tree (uncached runs), and the shell test passes its three `;4` / `84;4u` delivery rows against `bin/pair`. What keeps this off SHIP is the same thing as last round: the Done-when's live bullet is still unmet while the plan row stays ticked. The new Log entry is honest about that ("the remaining operator step is the in-workbench check"), but honesty in the Log does not tick the row, and the plan gained no `## Revisions` entry. That is an artifact fix plus one operator smoke test, not a code change.

**1. Strengths**

- `cmd/internal/layoutcmd/layoutcmd_test.go:244` asserts the delivered bytes per direction, not just an op count. This is the draft pane's only path and it is now red-without-fix (verified by revert).
- `cmd/internal/termcmd/run.go:604` merges `ChordAltT`/`ChordAltShiftT` the same way Left/ShiftLeft are merged at :615 (ARCH-DRY). `TestEveryHandledTerminalChordIsDocumented` still enumerates from `ChordMax()`, so the new chord was swept by the existing guard without edits.
- `cmd/internal/workbenchshortcut/shortcut_test.go:681` pins the class ("every from-anywhere delivery is a global"), which is what stops the #227 regression recurring for a fourth chord.
- The Log's BR-2 entry records a concrete, reproducible measurement (bare `nvim --clean` under a pty, `\x1b[84;4u` fires `nnoremap <M-T>`), which is real evidence for the NvimKey spelling rather than the "same pattern as `<M-N>`" argument the plan leaned on.
- `atlas/architecture.md:471-479` now states the mechanism (delivered AS the global chord, never passed through) and names the guard test correctly.

**2. Critical findings**

None.

**3. Important findings**

- **BR-2 not-addressed.** Issue Plan row 6 ("… full make test; live check") is `[x]`, Done-when's last bullet ("Live: from the draft with nvim in the right pane, M-S-left/right/t drive it") and the typed-`Alt+←/→`-still-reaches-nvim control are unmet, and the Log itself says so. The plan file has no `## Revisions` section (17 unticked boxes, Task 5 Step 5 unchanged). Do one of: run the in-workbench check and record what was pressed and observed, or untick the row and add a `## Revisions` entry deferring the live step to the operator. Not a code change either way.

**4. Minor findings**

- **BR-3 not-addressed on its second half.** The merge is done. The error is still discarded, and the new comment at `run.go:605-610` justifying that is inaccurate on both counts: `enqueue` is documented non-blocking (`run.go:1237`) and `ChordAltShiftD` already calls `mux.reportError` from this same function at `run.go:627`, so the "writer loop not started" hazard does not exist; and a failed `ptychild.Start` (`run.go:875`) spawns no child, so there is no EOF path to surface it. Either report the error or fix the comment so it does not teach a wrong rule.
- **New (Minor, `hand-maintained-restatement`):** the zellij `WriteChars` byte string for `Alt T` at `zellij/config.kdl:146` is a hand-typed restatement of the `chordSequences` row at `shortcut.go:413`, with no test tying the two. A typo there breaks typed `M-S-t` in every pane with the whole suite green. This is the class for every letter global (`Alt D`, `Alt N`, `Alt x`, …), not this instance: a guard that parses each `WriteChars` bind in `config.kdl` and asserts `DecodeChord` returns a global would cover all of them at once. Pre-existing shape; safe to take as a follow-up row.

**5. Test coverage notes**

- Verified red-without-fix: CLI `new` direction (both layoutcmd tests), pump under full-screen for `ChordAltShiftT`.
- Passing uncached: workbenchshortcut, layoutcmd, keyhelp (incl. `TestEveryGlobalChordIsClassified` over `GlobalBindings()`), runtimebundle (`TestEmbeddedSourcesMatchTree`), `tests/term-pane-shortcuts-test.sh` (14/14).
- Not runnable here: three termcmd tests and the wrapcmd harness-TTY tests fail with `operation not permitted`, the documented sandbox pty class. Nothing in this range touches them.
- Untested but low value: `nvim/workbench_route_test.lua` has no `new` argv row (noted last round, generic passthrough).

**6. Architectural notes**

- ARCH-DRY: pass. Duplicate case merged; one encoding row, one action mapping, all executors derive.
- ARCH-PURE: pass. `TabChordFor`, `DecodeChord`, `IsGlobalChord` tested without IO; delivery stays in the existing Runtime seam.
- ARCH-PURPOSE: pass on the issue's purpose (fix + third chord, from-draft path now tested). Shadow-sweep of the chord's consumers: Lua keymap (guarded), catalog (guarded), embedded bundle (guarded), zellij bind (unguarded, the Minor above).
- ARCH-MOCK: pass. Zellij behind the fake Runtime; shell test's fake `zellij` records delivered bytes.
- ARCH-CONSTRAINTS: pass. One chord per keystroke, nothing added to the hot path.
- ARCH-SECURE: pass. New sequence goes through the existing prefix decoder; the no-legacy-`\x1bT` test closes the ESC-ambiguity hole.
- ARCH-ORDER: pass. No new state between events; the passthrough decision remains a pure predicate plus one screen-state read.
- ARCH-FUNERAL: pass. Nothing durable created; the spawned tab's lifetime is owned by the mux's existing EOF path.
- Documented behavior change worth knowing: `Alt+Shift+t` typed into a right-pane app (or the agent) is now consumed workbench-wide, where before it was forwarded as raw bytes. README says so; fine.

**7. Plan revision recommendations**

- Add `## Revisions` to `workshop/plans/000243-from-anywhere-right-terminal-control-set-plan.md`: Task 5 Step 5's in-workbench live check is deferred to the operator (or record its result there and in the issue Log).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestRunSwitchTerminalTabDeliversTheGlobalBytes asserts bytes for prev/next/new; deleting `case "new"` in a scratch copy fails both RunSwitchTerminalTab tests.
  - id: BR-2
    disposition: not-addressed
    note: |
      Log now records a real pty-level encoding check, but Done-when's in-workbench live bullet and the typed Alt+arrow control are still unmet while plan row 6 stays ticked and the plan has no Revisions entry.
  - id: BR-3
    disposition: not-addressed
    note: |
      Cases merged; error still discarded and the justifying comment at run.go:605-610 is wrong (enqueue is non-blocking, ChordAltShiftD reports from the same function, a failed Start has no EOF path).
  - id: BR-4
    disposition: addressed
    note: |
      Single Log heading; atlas sentence now names the guard test correctly.
findings:
  - id: new
    severity: Minor
    family: hand-maintained-restatement
    title: |
      zellij WriteChars byte strings restate chordSequences by hand with no guard tying them together
    detail: |
      config.kdl:146 `Alt T` -> "\u{1b}[84;4u" duplicates shortcut.go:413; a typo breaks typed M-S-t with the suite green. Class covers every letter global (Alt D/N/x/...). One test parsing WriteChars binds and asserting DecodeChord yields a global covers them all; follow-up acceptable.
```

---

## Re-review — 2026-09-13T11:23:21-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 243 — regression: from-anywhere M-S-left/right stops switching the right terminal's tab when it shows a full-screen app (#227) |
| repo | pair |
| issue file | workshop/issues/000243-regression-from-anywhere-m-s-left-right-stops-switching-the-right-terminal-s-tab-when-it-shows-a-full-screen-app-227.md |
| boundary | whole-issue close |
| milestone | — |
| window | ec3f63eb163fb2215ad0be4e33676720fc581c4c..c4c564c66519cd3e77a054eb235c70474e178bfc |
| command | sdlc close --issue 243 |
| reviewer | claude |
| timestamp | 2026-09-13T11:23:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Wrapping up: all inspections are done, tests run, and the fix was confirmed pinned by reverting it in a scratch copy. Writing the verdict now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Plan row split; Log records the bare-nvim encoding proof; the in-workbench operator check is honestly unticked (close needs the operator's smoke test recorded, or an explicit --no-plan-check).
  - id: BR-3
    disposition: not-addressed
    note: |
      The two newTab cases ARE merged (run.go:604). The error half remains discarded, and the new comment's justification does not hold: a failed ptychild.Start has no child, so no EOF path reports it, and enqueue with a nil output channel is synchronous, so a direct handleTerminalChord test would not hang. One line at the merged site: if err := mux.newTab(); err != nil { mux.reportError(err) }. Minor, pre-existing on ChordAltT.
  - id: BR-5
    disposition: not-addressed
    note: |
      Deferred, which the prior round allowed, but the Log says "filed as a follow-up" and no issue in workshop/issues/ exists for it (grep for WriteChars/config.kdl finds only 000226). File it via sdlc issue new or reword the Log.
findings:
  - id: new
    severity: Minor
    family: naming-convention-consistency
    title: |
      ChordName(ChordAltShiftT) is "Alt+Shift+T" while ChordAltShiftD names "Alt+Shift+d"
    detail: |
      shortcut.go:527 vs :495. Cosmetic, but ChordName keys the pump's subtest names and any future help rendering; pick one casing for shifted letters.
```

The code delivers the issue's purpose as a class fix, not an instance fix: the from-anywhere delivery now returns global chords from the single mapping function, and a guard test pins that every from-anywhere action yields a global. I reverted the two mapping lines in a scratch copy of the head commit and both guard tests went red with the exact old `;3` bytes, so the pin is real. All three consumers of the mapping (the CLI, the agent pane, the pair term test-shortcut path) plus nvim, the generated keymap, zellij config, keyhelp, README, and atlas carry the new action. Nothing blocks SHIP; the remaining items are Minor. Package tests for workbenchshortcut, layoutcmd, and keyhelp pass. The termcmd and wrapcmd failures I saw are all the documented sandbox pty-spawn class ("operation not permitted" from ptychild), unrelated to this diff, and the registry and pump tests touched by this issue pass.

**Strengths**
- `TestTabChordForDeliversGlobalChords` (shortcut_test.go:681) encodes the rule rather than the symptom, so the #227 regression cannot recur silently. Verified red without the fix.
- KKP-only encoding for the new chord (shortcut.go:413) with a test asserting the legacy `ESC T` form does NOT decode. This closes the escape-deadline ambiguity before it exists. ARCH-ORDER pass.
- `TestFromAnywhereChordsDriveTheRightTerminalUnderFullScreen` exercises the real pump under `ownsScreen=true`, and the enumerating `TestEveryChordAgainstBothAltScreenStates` picks the new chord up automatically through `chordMax`.
- The delivery-byte test at layoutcmd_test.go:244 covers the draft's only path, and the shell test now asserts the `;4` bytes end to end.
- Docs are complete: README table row, README layout paragraph, atlas passthrough section, keyhelp catalog row tied to `globalBindings` by the existing drift test.

**Critical findings**
None.

**Important findings**
None.

**Minor findings**
- BR-3 residual: `newTab` error still discarded at run.go:611 and the comment's rationale is inaccurate (see dispose note).
- BR-5: Log claims a follow-up was filed, but no issue exists.
- `ChordName` casing differs between `Alt+Shift+d` and `Alt+Shift+T`.

**Test coverage notes**
- The reverting check confirms BR-1 is pinned by a test that fails without the fix.
- Live conformance for the nvim decode of `\x1b[84;4u` is recorded in the Log only, not automated. That matches the existing `<M-N>` precedent and is acceptable for a keystroke path.
- The in-workbench smoke test is the one unverified Done-when bullet. Ask the operator to press the three chords from the draft with nvim full-screen on the right before merging, then tick the row.

**Architecture notes**
- ARCH-DRY: pass. One mapping (`TabChordFor`), one encoding row, merged `newTab` case.
- ARCH-PURE: pass. The new entities are pure and tested without IO; delivery stays behind the `Runtime` fake.
- ARCH-PURPOSE: pass. Shadow-sweep of consumers finds every one deriving from `globalBindings` or `TabChordFor`. The only hand-maintained restatement is the zellij `WriteChars` byte string, already ledgered as BR-5.
- ARCH-MOCK: pass. zellij delivery runs against the injected fake; nvim decode was checked live.
- ARCH-CONSTRAINTS: pass. One chord per keystroke, no new work on the input path.
- ARCH-SECURE: pass. The only input is chord bytes already parsed by `DecodeChord`.
- ARCH-ORDER: pass. KKP-only avoids the ESC-deadline interleaving.
- ARCH-FUNERAL: pass. Nothing durable is created.

**Plan revision recommendations**
None required. The plan's Core concepts table matches the code. Optionally tick the plan file's own step checkboxes, which are all still `- [ ]` while the issue's Plan rows are ticked.
