# Boundary Review — pair#206 (milestone M2)

| field | value |
|-------|-------|
| issue | 206 — reattach detached threads automatically at startup |
| repo | pair |
| issue file | workshop/issues/000206-reattach-detached-threads-automatically-at-startup.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | e0680a76eafd8f1110fc65fc0acb9cd5a0a26c20..eed2fe312f6b9bc5f79ec1518e968637d3f5f1f7 |
| command | sdlc milestone-close --issue 206 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-12T13:04:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Review complete. Writing it up now.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** The M2 window delivers what the Spec and Plan promise: a pure `ReattachPass` inside `MenuState`, routed through `ReduceMenu`, with the pass view applied at the row lookups and a source-parsing guard; a `warm-only` resume that refuses before any effect; console wiring that never takes the operator's slot or focus; placeholders with no chip span; a start-only arm; the `COUCH_TRACE` trace; and the operator-assisted measurement recorded in the Log. I ran the three touched packages unsandboxed (couchtty, couchcore, couchcmd all `ok`) plus `tests/plan-superseded-facts-test.sh` (all passed), and reverted four claimed fixes in scratch overlays: each named test went red (the stopping tick's final paint, the reducer's `expireAttached` call, the focus-steal `Background` guard, and cell 10's hold). What keeps this off SHIP is one docs-gate gap: README's couch section still describes startup as "returns to the work already in that physical repository path" and says nothing about every other detached thread coming back behind it, or about the greyed placeholders the operator now sees on every restart. That is a cheap paragraph. One plan-drift instance remains in the Core-concepts code block. I did not run the full `make test`; the Log claims 197 packages exit 0 and I could not confirm that here.

**1. Strengths**
- `menu_reattach.go` is genuinely pure and reads as the transitions table: `armReattach`/`seedReattach`/`advanceReattach`/`finishReattach`/`expireAttached` map to cells 1–11, and the four-phase enum (Idle/Armed/Running/Done) makes "a second arm cannot re-seed" structural rather than a flag.
- `passViewOf` + `menuRows`/`menuThread`/`visibleMenuRows` + `TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups` (`menu_inventory_guard_test.go:35`) turns "every reader goes through the view" into a failing test, not a hope. The allowlist is tight (five bookkeeping functions).
- `TestReattachPassInvariantsHoldOverGeneratedSequences` (`menu_reattach_routing_test.go:178`) is the right oracle for ARCH-ORDER: 60 seeds × 120 steps over inventory/results/operator ops/cursor/Enter/Tab/click, asserting disjointness, root exclusion, queue bound, no emit under the operator's slot, and no pending selection. Removing cell 10's hold killed it.
- `warm-only` is pinned through the operation table (`warmresume_test.go` `TestWarmOnlyReachesTheResumeThroughTheOperationTable`) and refused from the CLI (`run_test.go` `TestWarmOnlyIsUnreachableFromTheCommandLine`), and `ResumeContextWith` refuses a verified park before `resumeEvidence` so a refusal writes nothing (revision asserted unchanged).
- `installObservedThreadActor`'s `background` flag reaches the installer only via the declared implicit `attach` arg, and both the unit form and the end-to-end form (`TestABackgroundSuccessWithNothingActiveLeavesTheSwitcherFocused`) exist; the end-to-end one is what would catch `finishOperation` dropping the arg.
- `effectiveBindings` is now the single derivation for `PairSession` and `DetachedSessions` (`lookupSessionName` deleted), and `TestDetachedSessionsBindsNothingForAnUnreadableScope` pins the fail-closed reach the comment claims. Good ARCH-DRY / documented-rule-reach follow-through.

**2. Critical findings** — none.

**3. Important findings**
- **README update appears missing for the startup reattach pass.** `README.md:323-326` still says a bare `couch` "returns to the work already in that physical repository path: detached first, then parked" and stops there. Since M2 a bare start also reattaches every other detached thread in the background, shows greyed placeholders with a spinner on the status row, greys `queued`/`reattaching…` rows in the switcher, and marks failures `reattach failed: <why>`. That is user-facing behaviour every operator sees on every restart. Fix sketch: one paragraph after the "returns to the work" sentence: bare start → cwd thread first → every other detached thread reattached one at a time behind you → placeholders not clickable/selectable until ready → `couch resume <tag>` does not do this → parked threads are never resumed. Env vars stay atlas-only, consistent with `COUCH_INPUT_TRACE`.

**4. Minor findings**
- **This is the 2nd finding in family `plan-drift-from-code`.** The plan's Core-concepts code block (`plan.md:197-203`) declares three `ReattachPhase` constants; `menu_reattach.go` has four (`ReattachDone`), and `Failed`'s comment says "diagnostic code per row" where decision 11 now stores an error's first line. The close-out revision that "read the plan against the code" missed it because it checked prose, not the code block. Do not fix the instance alone: the rule is the one `lessons.md` just added for counts and lists ("point at its one home"), extended to declarations — a plan never restates a type the code owns; the Core-concepts row names the file and the block goes. If the block must stay, register its stale line (`ReattachRunning … // seeded` as the last constant) in `tests/plan-superseded-facts-test.sh` like the other #206 tokens.
- `workshop/projects/couch-slots.md` (213 lines, commit `1cc55d99`) rides this window and has nothing to do with #206. Not code, not blocking; just noting the milestone diff carries an unrelated artifact.
- `advanceReattach`'s `state.OperationSequence == ^uint64(0)` guard ends the pass silently on counter overflow. Unreachable in practice; a comment saying so would stop the next reader wondering what it protects.

**5. Test coverage notes**
- Verified by revert (overlay, no repo mutation): (A) `syncStatusTick` final `repaint()` → `TestTheFrameThePassLeavesBehindIsPainted` fails; (B) reducer's `expireAttached` call → `TestANewerInventoryTakesBackARowThePassAttached` fails; (C) `!completed.origin.Background` on the focus steal → `TestABackgroundSuccessNeverStealsFocus` fails; (D) cell 10 hold → both the hold test and the generated-sequence invariants fail. The claimed fixes are real.
- Task 8 Step 1's eight fixture tests all exist under recognisable names in `console_reattach_test.go`; item 6 (failure drops placeholder, marks row, pass continues) is covered by `TestTheConsoleRunsThePassAsWarmOnlyBackgroundResumes` ("both marked failed") plus placeholders being derived from `pendingPlaceholders`.
- The `zellij action` latency bound (≤35 ms) is inferred from the phased run's window enclosing startup, not from sample mode, and the Log says so honestly. It satisfies "measured, not assumed" as a bound; the raw-sample cut (`PAIR_PROBE_SAMPLE_TSV`) exists now for a future tighter measurement.
- ARCH-MOCK: console tests inject the dispatcher; couchcore tests run the warm-only refusals against the fake runner and fake zellij seam, and the happy path (`TestWarmOnlyResumeStillReattachesADetachedThread`) through the same seam. Live conformance for the whole pass is the operator's smoke on the real stack, which is recorded. Pass.
- Not run here: full `make test`; `TestSwitchAsksTheIncomingChildToRepaint` flake fix (`dad4de14`) is a fixture change with a stated mechanism, and the three switch tests passed in my run.

**6. Architectural notes**
- ARCH-DRY: pass. `spinnerGlyph`, `placeholderSGR`, `traceFile`, `effectiveBindings`, `claimsOf` via `claimsFromBindings` all consolidated in this range.
- ARCH-PURE: pass. Reducer pure; `statusModelLocked` split from `paintNow` so the placeholder model is testable without a terminal.
- ARCH-PURPOSE: pass. Every Done-when bullet is delivered or explicitly re-decided by the operator (first-inventory deviation, "not selectable" replacing queue-jump). The readers-list single-source from M1 was swept through `startup.go`, the test header and the atlas.
- ARCH-MOCK: pass (above).
- ARCH-CONSTRAINTS: pass. Envelope states the ~20 s worst case and the measured ordinary case; the 120 ms tick is armed only while loading (test + mutant). Per-completion `requestMenuRefresh` coalesces through `refreshSchedule`: measured 2 refreshes over a 5-attempt pass.
- ARCH-SECURE: pass. Error text reaches the switcher only through `rowtext.SanitizeAndFit` (`passSuffix`, pinned); the trace writes codes and counts, never text; `warm-only`/`background` are implicit args the CLI cannot send; trace file is 0600 at an operator-supplied path.
- ARCH-ORDER: pass. Legal states are an enum plus a zero-valued `Loading`; interleavings are reproduced by feeding the pure reducer; Stop's extent is `WithCancel(c.lifetime)` and pinned by `TestStopCancelsAnInFlightPassAttemptAndRunsNoMore`. Note for #231: the status tick is the seam its clock should extend, as the plan says.

**7. Plan revision recommendations**
- `## Revisions` entry: "Core concepts restated `ReattachPhase` with three constants; the code has four (`ReattachDone`, so a second arm cannot re-seed) and `Failed` holds a code or an error's first line. The block is replaced by a pointer to `menu_reattach.go`."

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Operating envelope now names both DetachedSessions queries (about 20 s worst case), and Task 12 measured the real distribution: 5 attempts, median 302 ms, and 2 coalesced refreshes during the pass, so the per-completion refresh is bounded by the schedule rather than N squared.
findings:
  - id: new
    severity: Important
    family: readme-tracks-user-facing-surface
    title: |
      README update appears missing for the startup reattach pass and its placeholders
    detail: |
      README.md:323-326 still describes a bare couch as returning only to the cwd thread. M2 reattaches every other detached thread behind it, shows greyed placeholders and non-selectable switcher rows, and marks failures; none of that is in README, and atlas/couch.md alone does not reach a reader who runs couch.
  - id: new
    severity: Minor
    family: plan-drift-from-code
    title: |
      The plan's Core-concepts code block restates ReattachPhase with three constants where the code has four
    detail: |
      This is the 2nd finding in family plan-drift-from-code. The rule, not the instance: a plan never restates a declaration the code owns (the lessons.md "one home" rule extended from counts to types); replace the block with a pointer to menu_reattach.go, or register its stale line in tests/plan-superseded-facts-test.sh.
```
