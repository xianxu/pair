# Boundary Review — pair#206 (whole-issue close)

| field | value |
|-------|-------|
| issue | 206 — reattach detached threads automatically at startup |
| repo | pair |
| issue file | workshop/issues/000206-reattach-detached-threads-automatically-at-startup.md |
| boundary | whole-issue close |
| milestone | — |
| window | f643ca8e890c638c742219c84562cb97c6eb14fd..16609d6d2a8c277d0068ec492a1dff680dce8cf6 |
| command | sdlc close --issue 206 |
| reviewer | claude |
| timestamp | 2026-09-12T13:21:29-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Summary.** The whole-issue window (M1 startup narrowing plus the M2 background reattach pass) delivers every Spec sequence step and every Done-when bullet, or records the operator's decision where it deviates (first-inventory seeding, non-selectable rows instead of queue-jump). I ran `go build ./...`, `go vet` on the three packages, the three package suites unsandboxed (couchtty, couchcore, couchcmd all `ok`), and `tests/plan-superseded-facts-test.sh` (all passed). The sandboxed run failed only on pty spawns ("operation not permitted") plus a cascading deadlock in couchcore, which is the known environment limit, not the code. Of the four open Minors, three are verified addressed: BR-11's fail-closed guard turns `TestDetachedSessionsBindsNothingForAnUnreadableScope` red when reverted in a scratch copy; `lookupSessionName` is gone from every Go source and `PairSession` calls `effectiveBindings`; every "three readers" restatement now points at `startupAsks`. BR-9 is still partly open: Task 2 Step 1 describes a `withOthers` zellij-seam `list-clients` count no test takes, with no Revisions line. Nothing new is above Minor. What remains is plan hygiene in one family that has now recurred a third time, so the recommendation is the class fix (a checked table), not the sites.

**1. Strengths**
- `menu_reattach.go` is a readable transitions table: four-phase enum, `passViewOf` as the one authority, and `menuRows`/`menuThread`/`visibleMenuRows` enforced by the source-parsing guard `TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups`.
- `TestReattachPassInvariantsHoldOverGeneratedSequences` (`menu_reattach_routing_test.go:178`) is the right ARCH-ORDER oracle: seeded RNG, 60 seeds × 120 steps over inventory, results, operator ops, cursor, Enter, Tab, click, so any failing interleaving is reproducible by seed.
- `effectiveBindings` (`artifactcollision.go:160`) is genuinely the single derivation now, its doc states the union rule and why summing was wrong, and `DetachedSessions`' reach comment names the test that pins it. The fake answers through production's `ProjectDetachedSessions` (ARCH-MOCK).
- `warm-only` and `background` are `Implicit` args pinned three ways: direct, through the operation table (`TestWarmOnlyReachesTheResumeThroughTheOperationTable`), and refused from the CLI (`TestWarmOnlyIsUnreachableFromTheCommandLine`). The refusal precedes `resumeEvidence`, and the revision is asserted unchanged.
- The README paragraph and the atlas bullets describe exactly what the code does; `COUCH_TRACE` and `PAIR_PROBE_SAMPLE_SECS` are documented where `COUCH_INPUT_TRACE` is.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- **This is the 3rd finding in family `plan-drift-from-code`.** The Core concepts tables restate the tree and three rows are wrong: `rootStateText` is marked modified at `plan.md:155` but is byte-identical between base and head (the change is `passStateText`/`passSuffix` in `renderRootMenuFrame`); `menuRowSelectable` is placed in `menu.go` at `plan.md:157` but lives at `menu_reattach.go:357`; the `COUCH_TRACE` row at `plan.md:233` and Task 11's file list at `plan.md:631` name `inputtrace.go` while the plumbing is `trace.go`. Add `plan.md:389` ("A measured first frame waits for M2's `COUCH_TRACE`"), stale since Task 12 measured 0.72 s, and BR-9's residual Task 2 Step 1. Do not fix the five sites alone. The rule: a plan table that names entities at paths is a set of claims nothing checks, so make it checked. Extend `tests/plan-superseded-facts-test.sh` (or a sibling) to parse the Core concepts rows and assert each name is declared in its stated file, and each "modified" row's file appears in the issue window's `--name-status`. The prose guard already exists for tokens; this is the same guard for the table. Prevalence this round: 5 sites, all in the plan.
- **Revision entries live under `## Estimate`, not `## Revisions`** (`plan.md:684` onward). The constitution asks for a `## Revisions` section, and `tests/plan-superseded-facts-test.sh:222-224` already had to special-case #206 by anchoring on `## Estimate`. Cheap fix: insert a `## Revisions` heading before the first dated entry and re-anchor the script.
- `runBackgroundOperation` discards `Enqueue`'s `accepted` result (`console_reattach.go:51`). A `(false, nil)` return, a duplicate key, produces no completion, so `Loading` would never clear and the pass would hang. Unreachable today because attempt keys are drawn from a monotonic counter, but the liveness of the pass silently depends on it. A comment, or treating `!accepted` like the error path, makes the assumption explicit (ARCH-ORDER: an error path that drops the in-flight effect).

**5. Test coverage notes**
- Revert-verified this round (scratch copy from `git archive`, no repo mutation): disabling the `readable[scope]` guard in `DetachedSessions` fails `TestDetachedSessionsBindsNothingForAnUnreadableScope` with the exact message the test states. BR-11's claim holds.
- BR-12 verified structurally: `lookupSessionName` appears only in plan and ledger prose; `PairSession` derives via `effectiveBindings` over its one read (`artifactcollision.go:210`); `claimsOf` counts through `claimsFromBindings`.
- Round 4's four revert checks (final paint, `expireAttached`, focus-steal guard, cell 10 hold) stand; I did not repeat them.
- Not run here: full `make test`. The Log claims 197 packages exit 0 at low load; I could not confirm.
- Sandboxed failures were all `ptychild: ... operation not permitted` in couchcore and couchcmd, plus a deadlock that follows from a child that never spawned. Unsandboxed, all three packages pass.

**6. Architectural notes**
- ARCH-DRY: pass in code (`spinnerGlyph`, `placeholderSGR`, `traceFile`, `effectiveBindings`, `claimsFromBindings`). Flag in the plan: the tables are a second copy of the tree (finding 1).
- ARCH-PURE: pass. Reducer pure; `statusModelLocked` split from `paintNow`; `startupAsks` a pure predicate beside its readers.
- ARCH-PURPOSE: pass. Shadow-sweep of the reader-list single-source: `startup.go`, the test header, `atlas/couch.md`, and the plan all point at `startupAsks`' comment; no restated count survives.
- ARCH-MOCK: pass. The fake shares the duplicate-name rule; the zellij stub sits at the same seam production uses; console tests inject the dispatcher; the operator's live smoke is the conformance run and is logged.
- ARCH-CONSTRAINTS: pass. Envelope names both `DetachedSessions` queries (~20 s worst case); measured 0.72 s first frame, 2.34 s pass, `zellij action` ≤35 ms; the status tick is armed only while loading and pinned.
- ARCH-SECURE: pass. Failure text reaches the switcher only through `rowtext.SanitizeAndFit` (pinned); trace records codes, never text; implicit args are unreachable from argv; an undecodable index scope fails closed; the probe bounds its env input.
- ARCH-ORDER: pass, with the `accepted` note above. Extent is `WithCancel(c.lifetime)` and pinned by `TestStopCancelsAnInFlightPassAttemptAndRunsNoMore`; a successful leave ends the pass in the reducer.

**7. Plan revision recommendations**
- A `## Revisions` entry: "Core concepts: `rootStateText` unchanged (the rendering change is `passStateText`/`passSuffix`); `menuRowSelectable` lives in `menu_reattach.go`; the trace plumbing is `trace.go`. Envelope: the first frame was measured at 0.72 s (Task 12), replacing 'waits for M2'. Task 2 Step 1: the count is candidates at the fake seam (`DetachedCandidatesAsked`, `BindingResolutions`), not `list-clients` with `withOthers`. Swept: lines 155, 157, 233, 389, 442-445, 631." Then the table guard from finding 1 so the entry is the last of its kind.
- Move the dated entries under a `## Revisions` heading.

```findings
dispose:
  - id: BR-9
    disposition: not-addressed
    note: |
      Tasks 2-4 are ticked and sessionNameClaims is renamed, but Task 2 Step 1 (plan.md:442-445) still describes withOthers and a zellij-seam list-clients count; the test counts candidates and binding resolutions at the fake seam, and no Revisions line records that.
  - id: BR-10
    disposition: addressed
    note: |
      All six sites point at startupAsks' comment or are gone (no "three readers" in any Go, atlas or plan body; lookupSessionName's comment deleted); the lesson is written. The one residual forward pointer (plan.md:389) is folded into the new plan-drift finding.
  - id: BR-11
    disposition: addressed
    note: |
      The comment states the fail-closed reach and names TestDetachedSessionsBindsNothingForAnUnreadableScope; reverting the readable[scope] guard in a scratch copy turns that test red.
  - id: BR-12
    disposition: addressed
    note: |
      lookupSessionName is deleted; PairSession calls effectiveBindings over its one read; claimsOf counts through claimsFromBindings.
findings:
  - id: new
    severity: Minor
    family: plan-drift-from-code
    title: |
      The Core concepts tables misplace three entities and the envelope still waits for a measurement Task 12 delivered
    detail: |
      This is the 3rd finding in family plan-drift-from-code. rootStateText is marked modified (plan.md:155) but is identical at base and head; menuRowSelectable is placed in menu.go (plan.md:157) but lives in menu_reattach.go:357; the COUCH_TRACE row (plan.md:233) and Task 11's file list (plan.md:631) name inputtrace.go where the plumbing is trace.go; plan.md:389 says the first frame waits for M2's trace after Task 12 measured 0.72 s. Do not fix the sites alone. The rule: a table of entities at paths is a set of claims nothing checks; extend tests/plan-superseded-facts-test.sh to parse the Core concepts rows and assert each name is declared in its stated file and each modified row's file is in the window's name-status. Prevalence: 5 sites, all in the plan.
  - id: new
    severity: Minor
    family: plan-revisions-section-convention
    title: |
      The plan's dated revision entries live under the Estimate heading, not a Revisions section
    detail: |
      plan.md:684 is a one-line Estimate stub followed by every revision entry; the constitution asks for a Revisions section, and tests/plan-superseded-facts-test.sh:222-224 already special-cases #206 by anchoring on Estimate. Insert the heading before the first dated entry and re-anchor the script.
  - id: new
    severity: Minor
    family: attempt-always-completes
    title: |
      runBackgroundOperation discards Enqueue's accepted result, so a refused duplicate key would strand the pass on Loading
    detail: |
      console_reattach.go:51 handles only the error return; a (false, nil) return produces no completion and finishReattach never runs. Unreachable today because attempt keys come from a monotonic counter, but the pass's liveness silently rests on that. Treat !accepted like the error path, or state the assumption at the call.
```
