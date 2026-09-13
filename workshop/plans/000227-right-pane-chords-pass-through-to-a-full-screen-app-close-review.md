# Boundary Review — pair#227 (whole-issue close)

| field | value |
|-------|-------|
| issue | 227 — right-pane chords pass through to a full-screen app |
| repo | pair |
| issue file | workshop/issues/000227-right-pane-chords-pass-through-to-a-full-screen-app.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3acb6790f77e7fa7734983445d512b63099d21fe..61865beaf28af01ddf709571a140ddc3ff4ba234 |
| command | sdlc close --issue 227 |
| reviewer | claude |
| timestamp | 2026-09-13T09:50:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The mechanism is right and is pinned by tests that I verified go red on revert: removing the pump gate in a scratch copy fails `TestEscThenJReachesAFullScreenChildAsTwoKeys` and `TestEveryChordAgainstBothAltScreenStates`; removing the M-k and global exclusions from the predicate fails `TestFocusLeftNeverPassesThroughToAFullScreenChild`, `TestGlobalChordFiresUnderAFullScreenChild` and `TestRightTerminalChordPassesThrough`. The only failing tests at HEAD in the touched packages are the three known sandbox-blocked pty spawns (`ptychild: start sh: operation not permitted`), unrelated to this diff. `Decide` is untouched, the gate sits at the single funnel in `pumpStdinWithTimer`, and the tri-state is consumed through `RepaintModes()` with nil-safe intercept defaults. What keeps this off SHIP is the documentation surface: the new `pair keys` heading now claims that Alt+k and the global Shift+Alt+←/→ pass through to a full-screen app (they are in that group and they do not), README still documents every right-terminal chord as unconditional, and the live verification of the issue's actual purpose (parley's `<M-t>` under nvim) is neither ticked in the Plan nor recorded in the Log.

## 1. Strengths

- **Gate placement is correct and minimal** (`cmd/internal/termcmd/run.go:502-514`). All three dispatch paths (rename, `handleTerminalChord`, `handleChord`) sit below the gate, and `chordBefore` is flushed first so bytes ahead of the chord are never reordered. The plan's deviation from the Spec's `ShortcutInput` sketch was the right call; the Spec's approach would have missed exactly the tab chords the operator reported.
- **Tri-state consumed as the Spec demands** (`run.go:1437-1446`): `RepaintModes()` locked pair, `altScreen && observed`, nil tab / nil child → intercept. `TestActiveChildOwnsScreenTriState` drives a real `ptychild.Screen` through `?1049h` / `?1049l` bytes via `NewFakeChild`, so it exercises the actual parser, not a stub.
- **Table oracle derives from the classifier** (`passthrough_test.go:83-121`): the test asks `RightTerminalChordPassesThrough` for the expectation, so PQ-5 is honoured and the M-k exclusion cannot drift from the test. Both legacy `ESC x` and CSI-u encodings are covered because it iterates `ChordSequences()`.
- **Revert-verified coverage**: every behavioural claim in the diff (forward, exclude M-k, exclude globals, tri-state) has a test that fails without it. Confirmed by mutation in a scratch copy, above.
- **Live conformance probe updated honestly** (`probes/escsmoke/main.go:125-140`): the old "Alt+j is consumed" step was inverted to expect line 2, and the comment records that a pre-#227 build shows line 1. That is the right way to flip a probe rather than delete it.

## 2. Critical findings

None.

## 3. Important findings

- **`pair keys` heading overclaims passthrough for chords that do not pass through** — `cmd/internal/keyhelp/catalog.go:10` vs `catalog.go:76,92-93`. The `groupTerminal` group contains `Alt+k` (role-scoped, explicitly excluded from passthrough and the safety-critical escape) and `Shift+Alt+←/→` (globals, always fire). The new heading "Terminal tabs (right terminal; pass through to a full-screen app)" now tells the operator the opposite of PQ-1's whole point for M-k. The commit body says "the M-k detail stays in its binding row", but the M-k row's Help (`shortcut.go`, "jump back to the left pane you came from") is unchanged in the diff, so nothing in the help says M-k survives. Fix sketch, preferred (ARCH-PURPOSE shadow-sweep, ARCH-DRY): in `keyhelp/sections.go:49-53` where role rows take their Help from `RoleBindings()`, append a derived suffix per row from `workbenchshortcut.RightTerminalChordPassesThrough(rb.Chord)` (e.g. " (a full-screen app gets this key)" / " (always Pair's)"), and revert the heading to its short form. That makes the help a consumer of the predicate rather than a hand-maintained restatement, and sidesteps the `TestRunCentersWhenAsked` width limit that forced the heading to be terse. Cheaper alternative: reword the heading so it is true ("tab chords pass through to a full-screen app; Alt+k and Shift+Alt+←/→ always fire").
- **README update appears missing for the changed chord behaviour** — `README.md:15` and `README.md:118-122`. The layout-3 blurb and the key table document `Alt+t`, `Alt+w`, `Alt+r`, `Alt+Shift+d`, `Alt+←/→`, and `Alt+Shift+Return` in the "layout 3 terminal" scope as unconditional. After this diff they do nothing for pair term while nvim/less/htop owns the pane. One sentence in the layout-3 blurb ("when a full-screen app such as nvim is running in the tab, these chords go to the app; `Alt+k` and `Shift+Alt+←/→` still work") plus a scope note on the table rows is enough. This is the class of gap the docs gate exists to catch before the merge-time specs judge.
- **The issue's stated purpose has no recorded live evidence** — issue `## Plan` last row is `- [ ] Manual: parley <M-t> in right-pane nvim; <M-k> back; ESC+j; <M-t> at the shell`, and `## Log` has no entry after 2026-09-10. The commit body of 3c5a6691 claims an escsmoke run with a real nvim (Alt+j moves the cursor on this build, stays put on an origin/main control; Alt+k never reaches nvim), which is exactly the evidence the Log should carry, but it lives only in a commit message. The Done-when's first bullet (parley's outline opens on `<M-t>`) depends on the outer terminal's Alt encoding reaching nvim through zellij and pair term, which unit tests cannot reach. Before close: run the operator smoke test for `<M-t>` under nvim+parley and at a shell prompt, paste the escsmoke output, and tick the row. The `plan-unchecked` close gate will refuse on this row anyway; do not `--no-plan-check` it, since this row is the purpose.

## 4. Minor findings

- ARCH-DRY: `activeChildOwnsScreen` (`run.go:1437`) repeats `appMouseMode`'s lock / `activeTabLocked()` / unlock / nil-check triple (`run.go:1424`); `activeTabLocked()` is read under the lock at 7 sites. An `activeChild() *ptychild.Child` helper would collapse the two child-mode readers.
- `chord != ChordUnknown` in `RightTerminalChordPassesThrough` is unreachable from the gate (`FindChord` never returns `ChordUnknown` with `ok`), but it is a fair contract for a public predicate and is tested. Keep.
- The durable plan's Task steps are all still `- [ ]` while the issue's Plan is ticked. Harmless, but the plan is archived with the issue; tick or leave a one-line note that the issue's Plan is authoritative.
- `TestEveryChordAgainstBothAltScreenStates` sets `activeName: "work"` without using it; drop it or comment why.

## 5. Test coverage notes

- Unit: 86 passing subtests across the five new/changed tests. The two states × every chord sequence, the tri-state accessor (including nil tab and nil child), ESC-then-j, M-k exclusion, global exclusion.
- Mutation-verified (scratch copy): gate removed → 2 positive tests red; exclusions removed → 3 exclusion tests red. Nothing in the diff is protected only by tests that pass trivially.
- The "unknown" alt-screen state is covered only at the accessor, not through the pump. Acceptable, and the test file says so; the fake's `ownsScreen` is a bool, which is the right seam because the accessor owns the tri-state collapse.
- Live: `probes/escsmoke` builds and is the conformance check for the real nvim behaviour, but its run is not logged (see Important finding 3). It cannot run in this sandbox (pty spawn denied).

## 6. Architectural notes

- **ARCH-DRY: pass**, with the Minor above. One definition of "global" via `IsGlobalChord`; one gate.
- **ARCH-PURE: pass.** `Decide` untouched; predicate pure and table-tested; the only IO is one locked read behind the mux interface.
- **ARCH-PURPOSE: flag** (Important finding 1). The "which chords pass through" fact has one enforced source (the predicate) and two hand-maintained restatements (keyhelp heading, README). The keyhelp one is now wrong for three of its rows; deriving per-row from the predicate closes it for good.
- **ARCH-MOCK: pass.** `NewFakeChild` is the stateful child fake driven by real escape bytes; `fakeMux` is the pump seam shared with every existing pump test; escsmoke is the live conformance check against real nvim.
- **ARCH-CONSTRAINTS: pass.** One map lookup plus one locked read per recognised chord, not per byte; no new blocking on the keystroke path.
- **ARCH-SECURE: pass / N/A.** Forwarded bytes are the terminal's own input, already forwarded verbatim for every non-chord byte; no new trust boundary.
- **ARCH-ORDER: pass with note.** No new carried state. The one interleaving that matters, a chord arriving while the child is mid-transition into or out of the alt screen, resolves per keystroke to whichever the locked read sees; both outcomes are legal and non-wedging, as the plan states. The fake cannot inject that interleaving, but since there is no carried state there is nothing for a second ordering to corrupt.
- **ARCH-FUNERAL: pass.** Nothing durable created.
- **Forward note:** if a second must-survive chord ever joins M-k, the exclusion set in `RightTerminalChordPassesThrough` is the one place, and a keyhelp derivation (finding 1) would make that addition self-documenting.

## 7. Plan revision recommendations

- Add a `## Revisions` entry recording that Task 5 Step 1's "one edit site is the heading" was insufficient: the terminal group contains M-k and the two globals, so a heading-only note misstates them, and the help must annotate per row (or the heading must name the exceptions). The plan currently claims the heading is the whole fix.
- Task 5 lists `atlas/architecture.md` as the docs site but omits README; add README to the Files list so the docs gate is visible in the plan.

```findings
findings:
  - id: new
    severity: Important
    family: help-derives-from-classifier
    title: |
      pair keys "Terminal tabs" heading claims passthrough for Alt+k and Shift+Alt+←/→, which never pass through
    detail: |
      catalog.go:10 heading applies to the whole group, which includes Alt+k (catalog.go:76, the PQ-1 exclusion) and the global Shift+Alt+←/→ (catalog.go:92-93). The M-k row's Help is unchanged, so nothing in the help says it survives. Derive a per-row note from RightTerminalChordPassesThrough in sections.go:49-53, or reword the heading to name the exceptions.
  - id: new
    severity: Important
    family: readme-tracks-user-surface
    title: |
      README update appears missing for right-terminal chords becoming conditional under a full-screen app
    detail: |
      README.md:15 and README.md:118-122 still document Alt+t/w/r, Alt+Shift+d, Alt+←/→ and Alt+Shift+Return as unconditional in the layout-3 terminal. Add one sentence to the layout-3 blurb and a scope note on those rows.
  - id: new
    severity: Important
    family: purpose-verified-live-and-logged
    title: |
      Live verification of the issue's purpose (parley M-t under nvim) is unticked and unlogged
    detail: |
      Issue Plan's Manual row is unchecked and the Log has no 2026-09-13 entry; the escsmoke evidence exists only in commit 3c5a6691's body. Run the operator smoke test (M-t under nvim+parley, M-t at a shell), paste the escsmoke output into the Log, tick the row. Do not bypass plan-unchecked for this row.
  - id: new
    severity: Minor
    family: shared-accessor-helper
    title: |
      activeChildOwnsScreen repeats appMouseMode's lock/activeTabLocked/nil-check triple
    detail: |
      run.go:1437 vs run.go:1424; an activeChild() helper would collapse both child-mode readers (ARCH-DRY).
  - id: new
    severity: Minor
    family: plan-artifact-tracks-progress
    title: |
      Durable plan's task checkboxes are all unticked while the issue Plan is ticked
    detail: |
      workshop/plans/000227-...-plan.md steps remain "- [ ]"; tick them or note the issue Plan is authoritative before archival.
```
