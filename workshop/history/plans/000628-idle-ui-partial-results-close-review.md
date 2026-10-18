# Boundary Review — pair#628 (whole-issue close)

| field | value |
|-------|-------|
| issue | 628 — Idle UI invariant intermittently returns only 16 of 20 activity results |
| repo | pair |
| issue file | workshop/issues/000628-idle-ui-partial-results.md |
| boundary | whole-issue close |
| milestone | — |
| window | 433e0f2e0d4166dcd6411703a75a81771bdee9ff..946c2b15c5903794d7f936768784b980b52df013 |
| command | sdlc close --issue 628 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T14:01:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code change is small and correct. `probeTimeout` is a per-Console seam that `New` sets to the unchanged 2s const, and the worker at `console_activity.go:104` reads it. The production deadline and the idle-tick accounting are untouched, so both Done-when clauses hold. Both new tests pass at head. I also ran a revert check: with the `probeTimeout: activityProbeTimeout` initializer removed, `TestNewConsoleDefaultsTheProbeBound` fails, and so does `TestWorkbenchPerformanceIdleUITick`, with all 20 rows failed in 0.21s on an unloaded machine.

That revert result contradicts the explanation this boundary ships. `workshop/lessons.md` and the issue Log say a zero `context.WithTimeout` deadline "does not fail deterministically" because "the timer arms asynchronously". In Go 1.27, `WithDeadlineCause` cancels synchronously when `time.Until(d) <= 0` (`context.go:645-648`). A zero bound therefore fails every probe deterministically. That wrong lesson, plus a load-sensitive 100ms bound in the new regression test, are the two things to fix before shipping.

1. **Strengths**
   - The seam is minimal. It's one unexported field, one initializer and one read site (`console.go:228,298`, `console_activity.go:104`), and production behavior doesn't change.
   - `TestNewConsoleDefaultsTheProbeBound` pins the constructor default separately from the feature test, which sets the field itself. The revert check confirms it catches a dropped initializer.
   - The deadline test covers all three states the Spec asks for: a cold overrun shows as failed with no UI activity, the next healthy pass recovers the row, and an established row keeps its previous evidence. The cold case asserts the partial-results shape directly.
   - The stall in `deadlineProbeRuntime` stops when `ctx.Done()` fires, so the test exercises the real context path rather than a mock.
   - The atlas entry (`atlas/workbench-performance.md:45-53`) documents the new seam and what an overrun leads to.

2. **Critical findings:** none.

3. **Important findings**
   - **Wrong claim in `workshop/lessons.md:2176-2180`** (the second #628 lesson) **and in the issue Log entry "the recorded suite caught my own regression".** Both say a zero deadline is a race the probe can win when the machine is idle. Go cancels a context whose deadline has already passed before `WithTimeout` returns. My revert check showed every probe failing, unloaded, in 0.21s. Fix: rewrite the lesson to say that a zero or negative `WithTimeout` deadline is already cancelled when it returns, so every probe fails deterministically. Add a `## Revisions` entry to the Log noting that the greens reported at `40b2d08a8` cannot have been run against the dropped initializer.
     - Same family: the lesson's advice to "verify deadline fixes under the suite's real contention" rests on the same false premise. The Log sentence "Unloaded runs … won that race and read green" does too.
   - **The new regression test can flake under load** (`console_activity_deadline_test.go:60`). The healthy row must finish inside a 100ms bound against a measured ~40ms floor, about 2.5x headroom. The issue itself attributes the original failure to loaded-host contention. A loaded `make test` could push the healthy row past 100ms and fail the `len(failed)` check. Fix sketch: make the stall block until `ctx.Done()` with no fixed delay, and raise the injected bound to around 1s. The two overruns would add roughly 2s to the test.

4. **Minor findings**
   - In `workshop/lessons.md`, the last #628 entry runs straight into the #645 entry with no blank line, unlike the entries around it.
   - `probeTimeout` is read by the worker goroutine without holding `c.mu`. That is fine today because only tests write it, and only before any worker starts. Saying so in the field comment would help the next editor.

5. **Test coverage notes**
   - New tests pass at head.
   - Revert check: removing the initializer turns both `TestNewConsoleDefaultsTheProbeBound` and `TestWorkbenchPerformanceIdleUITick` red, and deterministically so. The existing invariant test would therefore also have caught the dropped default.
   - Not covered: a wrapper-evidence row that overruns its deadline (the `wrapper != nil` branch). That's acceptable because the Spec doesn't claim it.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The tests reuse `workbenchperftest.NewFixture` and `ThreadActivityProbe`, and nothing in the diff duplicates #626's framing regressions.
   - **ARCH-PURE: pass.** The change is an injected setting on the IO shell. The deadline outcome is computed in the existing worker and fed into the pure `ReduceMenu`.
   - **ARCH-PURPOSE: pass.** The cause is attributed, reproduced with an overlay, and pinned at the Console seam without raising a timeout, which is what the Spec asked for.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry correcting the Log's "scheduling race between the probe and its own zero timer" explanation: a zero bound cancels synchronously and fails every probe. It should also say which earlier green runs could not have exercised the dropped initializer.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: doc-claim-unverified-against-runtime
    title: |
      lessons.md and the issue Log say a zero WithTimeout deadline is a race; Go cancels it synchronously
    detail: |
      In Go 1.27, context.WithDeadlineCause cancels immediately when time.Until(d) <= 0 (context.go:645-648). Reviewer revert check: removing probeTimeout's initializer makes TestWorkbenchPerformanceIdleUITick fail unloaded with all 20 rows failed in 0.21s. Family instances: lessons.md second #628 entry (the "timer arms asynchronously" claim), the same lesson's advice to verify under contention, and the issue Log sentence "Unloaded runs ... won that race and read green". Rewrite all three and add a Revisions entry.
  - id: new
    severity: Important
    kind: defect
    family: timing-bound-test-headroom
    title: |
      The deadline regression's 100ms bound leaves about 2.5x headroom over the 40ms floor and can flake under suite load
    detail: |
      console_activity_deadline_test.go:60 requires the healthy row to finish within 100ms every pass, under the same loaded-suite conditions that caused #628. Make the stall block until ctx.Done() with no fixed delay and raise the injected bound to around 1s so the healthy row has wide headroom. The sizing comment at :58-59 also needs updating.
  - id: new
    severity: Minor
    kind: defect
    family: list-entry-spacing
    title: |
      In lessons.md the last #628 entry runs into the #645 entry with no blank separator
    detail: |
      workshop/lessons.md around line 2189; the neighbouring entries are separated by blank lines.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"dedb212a-656d-41c5-8fa5-6be3114d9b84","repository":"github.com/xianxu/pair","issue_id":"000628","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/c5f0db1f520b193f","reviewed_head":"946c2b15c5903794d7f936768784b980b52df013","slot":"pair:4","requested_at":"2026-10-17T21:00:31.374624Z"},"phase":"finished","pid":11687,"started_at":"2026-10-17T21:00:31.606107Z","finished_at":"2026-10-17T21:01:36.531651Z","session_id":"ec3a7a9e-7efc-4d53-8f11-731ca9cdcf71","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.68783625s
head: "946c2b15c5903794d7f936768784b980b52df013"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-17T14:06:40-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 628 — Idle UI invariant intermittently returns only 16 of 20 activity results |
| repo | pair |
| issue file | workshop/issues/000628-idle-ui-partial-results.md |
| boundary | whole-issue close |
| milestone | — |
| window | 433e0f2e0d4166dcd6411703a75a81771bdee9ff..a8eb5f150003b1371c8b4092f3c7158649a55021 |
| command | sdlc close --issue 628 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T14:06:40-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-2 and BR-3 are fixed, and the code itself is correct. The production deadline is unchanged at `activityProbeTimeout` (2s), and `New` sets it as the default at `console.go:300`. The worker reads it at `console_activity.go:104`. Workers only start inside `advanceActivity`, so the field comment's promise that tests write it "before any worker starts" holds. Both regression tests pass under `-race` at head (3.5s). One problem keeps this from SHIP: BR-1 fixed the three places it named, but one more copy of the same false "race" claim is still in the tree. It's the doc comment on `TestNewConsoleDefaultsTheProbeBound`. Fixing it is a one-line comment edit.

Done-when: current — both clauses still cover what was built. Clause one is met by the overlay reproduction, the cause traced to probe-deadline overruns, and the deterministic Console-seam regression. Clause two is met because there's no production timeout or accounting change: the test lowers only an injected per-Console bound.

1. **Strengths**
   - `console_activity_deadline_test.go:30-38`: the stall now returns on `ctx.Done()`, with the 10s delay as a backstop only. A row that overruns returns exactly at its deadline. The 1s bound is about 25x the ~40ms floor (BR-2).
   - `TestNewConsoleDefaultsTheProbeBound` checks the constructor default on its own. The feature test sets the field directly, so it could never catch a missing default.
   - The regression covers both states of the "keeps previous evidence" clause: a cold overrun leaves no activity, and an overrun on an established row keeps its earlier value, with recovery in between.
   - `workshop/lessons.md`: the zero-deadline lesson now states the synchronous-cancel behaviour correctly. Per the BR-1 note, the BR-1 reviewer's revert check (dropping the initializer) took the invariant test red unloaded in 0.21s; the Revisions entry records a matching implementer check.

2. **Critical findings:** none.

3. **Important findings:** none new. BR-1 stays open; see dispositions below.

4. **Minor findings:** none new.

5. **Test coverage notes:** BR-2 is a fix to how robust a test is, not a behaviour change, so "red without the fix" doesn't apply. I confirmed the bound, the ctx-bound stall and the sizing comment at `:56-60` all agree. BR-3: there is now a blank line between the last #628 entry and the #645 entry in `lessons.md`.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The new test uses the existing `workbenchperftest` fixture and `ThreadActivityProbe` rather than copying them.
   - **ARCH-PURE: pass.** The change is a single injected duration field, kept in the IO shell.
   - **ARCH-PURPOSE: flag (BR-1).** The record correction fixed the three named instances but missed the test comment, which belongs to the same family.

7. **Plan revision recommendations:** none beyond the existing Revisions entry. Once the test comment is fixed, that entry's claim that the race story is superseded will be true everywhere.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      lessons.md and the issue Log/Revisions are corrected, but console_activity_deadline_test.go:41-43 still says a zero deadline means a probe "can lose that race" under load and the missing default "reads as a flake". Fix: rewrite it as "a zero deadline returns an already-cancelled context, so every probe fails deterministically". Rule for the doc-claim-unverified-against-runtime family: when a mechanism claim is corrected, grep the window (comments, lessons, Log, atlas) for every restatement of it (here: race/flake/won/lost) and fix them all in the same commit. A grep for race|flake over the diff's code and docs finds only this one remaining site.
  - id: BR-2
    disposition: addressed
    note: |
      The bound is now 1s at :61, the stall returns on ctx.Done() (10s delay is only a backstop), and the sizing comment at :56-60 is updated. Green under -race at head.
  - id: BR-3
    disposition: addressed
    note: |
      lessons.md now has a blank line between the last #628 entry and the #645 entry.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2b1d5be7-a4f0-4e09-9a92-673432c0b789","repository":"github.com/xianxu/pair","issue_id":"000628","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7df6bdbba6ff1fb9","reviewed_head":"a8eb5f150003b1371c8b4092f3c7158649a55021","slot":"pair:4","requested_at":"2026-10-17T21:05:50.472168Z"},"phase":"finished","pid":73450,"started_at":"2026-10-17T21:05:50.695404Z","finished_at":"2026-10-17T21:06:39.606355Z","session_id":"38429770-c733-474b-be09-ae4a28c88dd1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.596450083s
head: "a8eb5f150003b1371c8b4092f3c7158649a55021"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-17T14:08:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 628 — Idle UI invariant intermittently returns only 16 of 20 activity results |
| repo | pair |
| issue file | workshop/issues/000628-idle-ui-partial-results.md |
| boundary | whole-issue close |
| milestone | — |
| window | 433e0f2e0d4166dcd6411703a75a81771bdee9ff..892463df63d657addb89400f4d1cfd3f2e60dc91 |
| command | sdlc close --issue 628 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-17T14:08:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This window adds the deadline regression for #628, and it holds up. `probeTimeout` gets its default in `Console.New`, and `advanceActivity` reads it. `TestNewConsoleDefaultsTheProbeBound` pins that default. `TestActivityProbeDeadlineSurfacesFailureAndRecovers` checks the three outcomes at the Console seam: a cold overrun shows as failed with no activity, the next healthy pass recovers the row, and an overrun on an established row keeps its previous evidence. Both tests passed at the pinned head (`go test -count=1 ./cmd/internal/couchtty -run …`, 2.3s). The one open finding, BR-1, is fixed. The rewritten lesson matches Go's source: `context.go:646-648` cancels synchronously when `dur <= 0`. The advice to "verify under contention" has been removed. The issue Log's "won that race" sentence is now overridden by a dated `## Revisions` entry, and the earlier entries are kept as history. A sweep of the code, comment and atlas changes in this window found no other claim that a zero deadline races. Nothing blocks SHIP.

Done-when: current — the clauses ask for a reproduction (the overlay entry in the Log), a cause (deadline overruns driven by the pre-#626 framing cost), a deterministic regression (the new tests), and proof that no timeout was raised (production `activityProbeTimeout` is unchanged; only the test shrinks its own bound). The Spec/Revisions edits correct the historical record and don't change the scope.

1. **Strengths**
   - `console_activity_deadline_test.go:30-38`: the stall ends on the context, so an overrunning row returns exactly at its deadline and the 10s delay is only a backstop. The 1s bound is about 25x the ~40ms bootstrap floor.
   - `console_activity_deadline_test.go:44-50`: the constructor default has its own test. This catches the dropped-initializer bug, which the feature test can't see because it sets the field itself.
   - The production bound and accounting are untouched. `console_activity.go:104` only swaps the const for the field, which defaults to that same const.
   - `atlas/workbench-performance.md:45-53` describes the new seam and the three outcomes accurately.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** the earlier Log entries ("greens were lucky", "100ms bound") still carry their old wording. That's acceptable, because the Revisions entry and the close-review Log entry explicitly replace them.

5. **Test coverage notes:** all three deadline outcomes in the Plan are asserted, as is the healthy-row/failed-row split. The default test is the regression for the dropped initializer.

6. **Architectural notes**
   - ARCH-DRY: pass. The test reuses `workbenchperftest.NewFixture` and the production `ThreadActivityProbe` and doesn't duplicate #626's framing oracle.
   - ARCH-PURE: pass. The IO seam is one injected `time.Duration` field plus an injected runtime wrapper, and the test needs no mocks of business logic.
   - ARCH-PURPOSE: pass. The cause is attributed by measurement, the deadline outcome is pinned, and the BR-1 family was cleaned up at every site in this window, not just the one named.

7. **Plan revision recommendations:** none. The new Revisions entry already brings the plan in line with the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      lessons.md:2176-2182 now states synchronous cancel (matches go context.go:646-648 dur<=0); contention advice removed; issue Revisions entry supersedes the Log race sentence; no race claim remains in the window's code, comments or atlas.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"14dfa0d2-01b2-4f10-bfab-428dcc4e3d6a","repository":"github.com/xianxu/pair","issue_id":"000628","recipe":"small-diff-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/e171a99018d0cdca","reviewed_head":"892463df63d657addb89400f4d1cfd3f2e60dc91","slot":"pair:4","requested_at":"2026-10-17T21:07:34.616912Z"},"phase":"finished","pid":48757,"started_at":"2026-10-17T21:07:34.708239Z","finished_at":"2026-10-17T21:08:25.741086Z","session_id":"8951c75d-3ce3-4ac4-830f-f0a83c07bb7f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.396774333s
head: "892463df63d657addb89400f4d1cfd3f2e60dc91"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
