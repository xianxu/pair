# Boundary Review — pair#541 (whole-issue close)

| field | value |
|-------|-------|
| issue | 541 — couch renicer: --actors reports lowered=N with COUCH_SLOT_NICE=0, and the low-agent notice compares against 0 not Couch's own nice (#502 close advisories) |
| repo | pair |
| issue file | workshop/issues/000541-couch-renicer-actors-reports-lowered-n-with-couch-slot-nice-0-and-the-low-agent-notice-compares-against-0-not-couch-s-own-nice-502-close-advisories.md |
| boundary | whole-issue close |
| milestone | — |
| window | 72892fbd9ee6f1bf080830b041bf11bb2fcfc969..6512f177abaf0595502a1048fd0091aec755cb56 |
| command | sdlc close --issue 541 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-13T17:46:24-07:00 |
| verdict | SHIP |

## Review

The couchcmd package won't build in this checkout until its embedded assets are generated. I'm generating them now (they're gitignored build output) so I can run the renicer tests.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: ship it.** The change fixes both #502 close advisories, does what the Spec says, and meets every `## Done when` clause.

- **Level 0:** `PlanRenice` now puts `never[pid] || level <= 0` in the first switch case, so level 0 counts nothing as lowered or at the level.
- **Notice:** the low-agent notice now compares against Couch's own nice, read from the same snapshot.

**How I checked it:**
- I ran the stat, name-status and full diff commands for the window.
- The `couchcmd` package wouldn't build in this checkout until I generated the gitignored runtime assets. After that, the targeted tests pass in both `couchcore` and `couchcmd`.
- I put back the base versions of `renice.go` and `renicer.go` and re-ran the tests:
  - `TestPlanReniceNever` failed (the level-0 `AtLevel` assertion).
  - `TestRenicerLevelZeroOnlyReports` failed with `AtLevel:2`.
  - `TestRenicerNoticeComparesAgainstCouchsOwnNice` failed: an agent at nice 5 was noticed.

  So each new assertion is a real regression test. I restored the files afterwards and the tracked tree is clean.

The only findings are Minor: two stale comments, one Spec branch with no test, and some README reflow.

**1. Strengths**
- `cmd/internal/couchcore/renice.go:84-85`: the fix sits in the pure planner, so the `--actors` column (`messages.go:385`, Lowered+AtLevel) reads 0 without any change to the display code.
- `cmd/internal/couchcmd/renicer.go:128-133`: Couch's nice comes from the same `table` snapshot as the agent's nice, as the Spec requires. If Couch's pid isn't in the snapshot it falls back to 0.
- Both Done-when clauses are tested in both states: level 0 in the core and in the renicer, and the notice with the agent at Couch's nice (no notice) and above it (one notice).
- The README and atlas were updated in the same range, and their new wording matches the code.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **Stale "below normal" wording:**
  - `renicer_test.go:192` still says "An agent found below normal".
  - `renice.go:24-25`: the `ReniceSlot` doc says AtLevel "counts those already at it or lower" and doesn't mention that it is 0 at level 0. The `PlanRenice` doc comment was updated; this one wasn't.
  - I found no other instances in code, the README or the atlas.
- **No test for the fallback:** the Spec's "missing → 0" branch (Couch's pid not in the snapshot) has no test.
- **README reflow:** `README.md:593-594` leaves an uneven line wrap.

**5. Test coverage notes**
The new tests check the actual behaviour, not mocks. The renicer test uses the existing `slotMachine` fake, as other renicer tests do. The only gap is the missing-pid fallback.

**6. Architecture**
- **ARCH-DRY: pass.** The loop that looks up Couch's own nice is a 4-line pid lookup; I found no existing helper it duplicates.
- **ARCH-PURE: pass, with a note.** Working out Couch's nice is a small pure step inside `pass()`. If the notice rule grows, it could move into `couchcore` (for example, `PlanRenice` returning Couch's nice), so it can be tested without the renicer.
- **ARCH-PURPOSE: pass.** I found two places that read AtLevel: the `--actors` column, which reads 0 at level 0 now that the planner sets AtLevel to 0, and the `renicer.go:147` facts. Neither needed its own fix. The notice comparison happens in one place only (`renicer.go:166`).

**7. Plan revision recommendations**
None. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: doc-claim-drift
    title: |
      Stale comments after the semantics change: renicer_test.go:192 says "below normal"; renice.go:24-25 ReniceSlot doc omits that AtLevel is 0 at level 0
    detail: |
      Family in window: renicer_test.go:192 (TestRenicerNoticesALowAgentOnce doc comment) and renice.go:24-25 (ReniceSlot.AtLevel doc). Code, README and atlas otherwise updated.
  - id: new
    severity: Minor
    family: spec-branch-untested
    title: |
      Spec's "Couch's pid missing from snapshot → own nice 0" fallback has no test
    detail: |
      renicer.go:128-133 defaults own to 0; no fixture omits r.self from the table. Cheap to add next to TestRenicerNoticeComparesAgainstCouchsOwnNice.
  - id: new
    severity: Minor
    family: prose-reflow
    title: |
      README.md:593-594 edited sentence left with a ragged line wrap
```
