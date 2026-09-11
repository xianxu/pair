# Boundary Review — pair#230 (whole-issue close)

| field | value |
|-------|-------|
| issue | 230 — A failed warm reattach deletes the session it was reattaching |
| repo | pair |
| issue file | workshop/issues/000230-a-failed-warm-reattach-deletes-the-session-it-was-reattaching.md |
| boundary | whole-issue close |
| milestone | — |
| window | d8dc14f60276d2dd708fea794e5e390abce9c9c4..74b0de8c43969d85ec5791ab7243cd03c90a99fe |
| command | sdlc close --issue 230 |
| reviewer | claude |
| timestamp | 2026-09-11T11:15:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The headline defect is genuinely fixed and the fix is load-bearing: I independently killed four mutations against the shipped tree (`owns` forced true at routes 1–4 and at 5–6, `AbortStarted` trusting the relayed `StartResult`, and warm/live-record `DurableRetire`→`DurableMarkUnknown`), each dying in the six-route seam table, and I verified the honest fake matches production (`QuiesceThreadSession` leaves the name-index entry intact, so production's `PairSession` reports `Present=false` after a quiesce — exactly what the fake now models). What keeps this off SHIP is one real fail-closed gap I confirmed with a read-only probe — on the warm live-record routes, a retire whose own session re-proof fails returns with **no** durable fallback, leaving a live incarnation behind a dead helper where the pre-#230 tail always reached `MarkIncarnationUnknown` — plus three documentation/structure drifts: the decider's `Quiesce` output is never read by production (mutating it to always-true leaves every seam test green), the plan's Core-concepts tables name an entity that was never built (`applyStartCleanup`) and label an IO helper PURE, and the issue's own Revisions still describe `StartResult.Warm` and a 24-row table that the code does not have — the plan gate's PQ-5, disposed `not-addressed` in three consecutive rounds and then dropped from the close Log's carry-forward list.

## 1. Strengths

- **The class was swept, not the filed site.** The issue was filed about cancellation; the diff covers all six exits that reach `quiescePostAckStart`, and I confirmed by grep that those are the only callers of `failPostAckStart`/`failTrackedPostAckStart` and that `Artifacts.Quiesce` has exactly two callers (`detach.go:246` `ArchiveThread`, out of class and documented). ARCH-PURPOSE, done properly.
- **`artifactcollision_fake.go:250` — the fake now models the deletion.** This is the standout. I checked fidelity against production rather than taking it on trust: `launcher.QuiesceThreadSession` (`thread_claim.go:257`) deletes the session but not the index entry, so `ScopedThreadArtifactCollisionChecker.PairSession` (`artifactcollision.go:149`) finds the name and reports `Present=false` — which is what the fake does. The refusal path (hook errors → no effect) matches the retry loop's premise. A live conformance test for the real behaviour already exists (`launcher/session_quiescence_live_test.go`).
- **`couch.go:579-589` — ownership read from couch's own registry**, and `warm_failure_test.go:276` deliberately zeroes the relayed field so the guard is visible. My M3 mutation (read `start.Record.Warm` instead) dies only on that test; every other route-6 test round-trips the real record and would have passed. That is the rare case of a test written against the guard rather than against the author's mental model.
- **`detach.go:118` — `retireDetachedIncarnation` extracted, not copied**, keeping `Detach`'s `ctx.Err()` interrupt (PQ-7 taken correctly), with the per-attempt-clock rationale preserved in the comment.
- **`workshop/lessons.md`** — both rules added are the real lessons (branch-table verification of "existing behaviour" claims; relayed-struct zero values), and the second one explains *why* the mutation survived the first attempt.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `cmd/internal/couchcore/couch.go:514` — a failed retire on the warm live-record routes has no durable fallback; the record is left `IncarnationLive` behind a dead helper.**
`failPostAckStart`'s warm branch returns `errors.Join(cause, cleanupErr, retireErr)` directly. Every failure mode of `retireDetachedIncarnation` — the session vanishing between `observeSessionPresence` and the retire's own `PairSession` re-proof, 32 exhausted revision conflicts, a `RetireIncarnation` precondition refusal — therefore skips the mark-unknown tail that the decider's own fail-closed rule (and the Spec's "mark the start unknown") prescribes. Pre-#230 this path always ran `reconcileInterruptedStarts` + `MarkIncarnationUnknown`, so the record ended Unknown.
Probe-confirmed (overlay test, no repo writes): warm route 5 with `PairSession` flipping absent on the retire's call leaves `incarnation[0] state="live" pid=1000`, and the switcher renders `state="unusable" reason="stale-incarnation"`. Not destructive — the row is visible and honest, which is why this is Important and not Critical — but it is the `pair#171` stale-live shape reached from an ordinary failure path, and it is untested.
*Fix sketch:* on `retireErr != nil`, fall through to the existing mark-unknown tail instead of returning. Test: `BeforePairSession` hook that reports present once, then absent, asserting the record ends Unknown (not Live).

**I2 — `cmd/internal/couchcore/startcleanup.go:98` — `StartCleanup.Quiesce` is a dead output; production never reads it.**
Measured: with `Quiesce: true` forced for every input, the whole seam suite (`TestAFailedWarmReattachKeepsItsSession`, `…OwningStartStillQuiesces…`, `TestAbortStartedReadsOwnershipFromTheRegistryNotTheCaller`, the resume tests) stays green — only `TestDecideStartCleanupTable` and `…NeverQuiescesABorrowedSession` fail. The quiesce half is decided at three call sites by `shape.OwnsSession()` (`couch.go:503`, `couch.go:507`, `launch_existing.go:191`), never by the decision struct. `atlas/couch.md` and the plan both state the pure function returns "whether to quiesce and which durable action to take", which is half true. The risk is a future edit to the decider's `Quiesce` branch that changes nothing in production while its own table test goes green.
*Fix sketch:* drop the `Quiesce` field and say plainly that `StartShape.OwnsSession()` is the single source the call sites consume (adjusting the property test to assert over `OwnsSession`), or route the quiesce through a shell that consumes the decision. Update the atlas paragraph either way.

**I3 — plan Core-concepts tables contradict the code (`workshop/plans/000230-…-plan.md:72` and `:172`).**
`applyStartCleanup` is listed as a new integration entity in `couch.go` (also in Task 4's file list, `:325`, and Task 4 Step 3, `:333`) and does not exist — the two ad-hoc tails it was meant to replace are still there, each consulting the decider differently. `retireDetachedIncarnation` is listed under **Pure entities** (`:72`) though it observes `PairSession` and writes the thread store; it is an INTEGRATION entity and its tests only run with fakes. The review contract rates a table/code contradiction Critical; I am rating it Important because the divergence is in the plan's description of correct, tested code rather than in shipped behaviour — but it needs a `## Revisions` entry or the refactor before the plan is archived.

**I4 — the issue's Revisions still assert a design the code does not implement (`workshop/issues/000230-…md:113`, `:154`, `:169`).**
`- StartResult carries Warm, so route 6 can read it` (no such field; ownership comes from `ActorRecord.Warm`), `quiescePostAckStart, which is the only caller of Artifacts.Quiesce` (`ArchiveThread` is the second, `detach.go:246`), and the Estimate's `24-row exhaustive table` (the delivered table is 36 rows). The plan gate raised exactly these as **PQ-5** and disposed them `not-addressed` in rounds 2, 3 and 4; the close Log carries PQ-6/7/10/11 forward but drops PQ-5 without a word. AGENTS.md's revise-don't-overwrite rule makes this cheap: one more `## Revisions` entry correcting all three, and a line in the Log accounting for PQ-5.

## 4. Minor findings

- `launch_existing.go:205` — the `DurableRetire` arm of `failTrackedPostAckStart` is unreachable: at claim phase no shape yields `DurableRetire` (spawn returns early, cold/warm claims yield rollback or mark-unknown). It duplicates `couch.go:521`'s retire block, and `ctx := context.WithoutCancel(context.Background())` at `:190` exists only to feed it.
- `context.WithoutCancel(context.Background())` (`launch_existing.go:190`, `couch.go:522`) is a no-op — neither function receives the caller's context, so "runs uncancellable" follows from `Background`, not from `WithoutCancel`. Either pass the caller's ctx through and wrap it (what the plan described) or drop the wrapper.
- `warm_failure_test.go:167` — route 3 asserts only `row.State != ThreadDetached`; the plan promised `ReasonSessionGone`. Any wrong-but-not-detached state passes.
- `couch.go:583` — `AbortStarted` labels a spawn `StartColdResume`. Harmless today (both own), but the name is false; consider recording the shape on `ActorRecord` rather than a warm bool.
- `couch.go:584` — a second full scan of `c.reg.Records()` right after the identity loop found the same record; the plan said "the lookup it needs is one it already performs".
- The owning half of the table drops the spawn rows the plan promised for routes 5–6 (`plan.md` Task 3 Step 1). Coverage is not actually lost — `abort_started_test.go:28` and `couch_test.go:866` cover spawn — so it is the plan text that is stale.
- Plan task checkboxes are all still `- [ ]` at HEAD though the work is done.
- PQ-6's unstated non-goal is the one worth a line: if the borrowed session dies before the child attaches, `pair resume <tag>` may *create* a session that the warm path will then never quiesce — leaked, not destroyed, and consistent with "never answer uncertainty destructively", but it should be named rather than discovered.

## 5. Test coverage notes

- Mutation evidence I produced myself (`go test -overlay`, tree untouched): routes 1–4 `owns=true` → 4 warm rows red; routes 5–6 `owns=true` → warm rows 5, 6 **and** the registry-ownership test red; `AbortStarted` reading the relayed struct → its dedicated test red; warm/live-record retire → mark-unknown → warm rows 5, 6 red. The decider-`Quiesce` mutation survived every seam test (I2).
- Sandbox run: `couchcore` and `couchcmd` failures are all `ptychild: operation not permitted`, i.e. the known sandbox limitation, not regressions; `launcher`, `artifactpath`, `sessioninventory` are green. The unsandboxed 197-ok claim in the Log is not independently verifiable from here.
- Uncovered: I1's retire-failure path. Also worth knowing — `quiescePostAckStart` only returns once the helper is proven quiet, so `HelperDead=false` never reaches the decider in production; the "nothing undone while the helper is unaccounted for" property is enforced by that loop, and the decider's live-helper rows (and all 12 spawn rows) are documentation rather than reachable behaviour. Fine as a total rule, but their green says nothing about production.
- `TestResumeAmbiguousAckRollsBackOnceItsSessionIsGone` is a correct rewrite: I traced production's cold route 1 (quiesce deletes the session → `PairSession` reports absent → rollback) and it matches. The Unknown arm is properly re-pinned by `TestResumeUnobservableSessionKeepsUnknownOccupied`.

## 6. Architectural notes

- **ARCH-DRY** — flag (Minor + I3): the retire block exists twice, one copy dead, because the planned single shell was not built; plus the duplicate registry scan. Pass on the real win, `retireDetachedIncarnation` shared with `Detach`.
- **ARCH-PURE** — flag (I2, I3): the decider is genuinely pure and IO-free, `observeSessionPresence` is a proper thin seam; but one decider output has no consumer, and the plan labels an IO helper PURE.
- **ARCH-PURPOSE** — flag (I2, I4): the route sweep is exemplary; the shadow-sweep residue is that the "one pure rule" has a half nothing derives from, and the issue's Revisions still describe the abandoned `StartResult.Warm` design.
- **ARCH-MOCK** — pass, and the model case: fake made stateful, fidelity checked against production, live conformance already present.
- **ARCH-CONSTRAINTS** — pass. Failure paths only; the warm route adds one extra zellij liveness call (the retire's re-proof), which is also what opens I1's window.
- **ARCH-SECURE** — pass. Ownership is read from state couch owns, the zero value points the safe way, and the guard is pinned by a test that zeroes it. `warm,omitempty` decodes to `false` = today's behaviour for pre-#230 registry files, and cross-process abort is impossible (the handle must come from this process's start).
- **ARCH-ORDER** — partial. The six-route table *is* the `(state, event) → effects` enumeration, and the shape rides couch's own durable record. Flag: the transition "retire fails mid-way" has no named target state, so the record lands outside the rule's vocabulary (I1). For #206's background pass, note that each failed reattach now leaves a recoverable row rather than a deleted session — that is the whole point — but a burst of them will leave several `unusable` rows if I1 is not closed.

## 7. Plan revision recommendations

1. `## Revisions` on the plan: "`applyStartCleanup` was not built — the decision is applied in `failTrackedPostAckStart` and `failPostAckStart` directly"; and move `retireDetachedIncarnation` from Pure entities to Integration points (it observes `PairSession` and writes the store).
2. `## Revisions` on the plan or issue recording the delivered mutation sweep as it actually ran (8 rows, not the 10 listed in Task 5) and which planned rows were not run — `owns` forced true per-route, `HelperDead` ignored, `DurableRetire`→`DurableMarkUnknown`, `WithoutCancel` removed, `ctx.Err()` removed. (I ran four of those myself; three kill, and the `WithoutCancel` one cannot kill anything because the wrapper is a no-op.)
3. `## Revisions` on the issue correcting PQ-5's three stale claims (`StartResult carries Warm`, "only caller of `Artifacts.Quiesce`", "24-row"), plus a Log line disposing PQ-5 explicitly rather than dropping it from the carry-forward list.
4. Plan Non-goals: add the `pair resume` create fall-through (a borrowed session that dies before attach may leave a session no cleanup path owns).

```findings
findings:
  - id: new
    severity: Important
    family: fail-closed-fallback-missing
    title: |
      A failed retire on the warm live-record routes returns with no durable fallback, leaving a live incarnation behind a dead helper
    detail: |
      couch.go:514 returns errors.Join(cause, cleanupErr, retireErr) directly, so every failure of
      retireDetachedIncarnation (session lost between observeSessionPresence and the retire's re-proof,
      exhausted revision conflicts, a RetireIncarnation precondition refusal) skips the mark-unknown tail
      the Spec and the decider prescribe. Pre-pair#230 this path always reached MarkIncarnationUnknown.
      Probe-confirmed via a read-only overlay test: warm route 5 leaves incarnation state "live" and the
      switcher renders unusable/stale-incarnation. Fix: fall through to the mark-unknown tail on retireErr,
      and pin it with a PairSession hook that reports present once then absent.
  - id: new
    severity: Important
    family: unconsumed-single-source
    title: |
      StartCleanup.Quiesce is never read by production; the quiesce half is decided at three call sites
    detail: |
      Measured: forcing Quiesce true for every input (startcleanup.go:98) leaves the entire seam suite green;
      only the decider's own table and property tests fail. Production reads shape.OwnsSession() at couch.go:503,
      couch.go:507 and launch_existing.go:191. The atlas paragraph and the plan claim the pure function decides
      whether to quiesce. Fix: drop the field and name OwnsSession() as the single source the call sites consume,
      or route the quiesce through a shell that consumes the decision; update atlas/couch.md to match.
  - id: new
    severity: Important
    family: plan-table-vs-code-drift
    title: |
      Core concepts tables name applyStartCleanup, which was never built, and classify an IO helper as PURE
    detail: |
      plan.md:172 (and :325, :333) lists applyStartCleanup as a new integration entity in couch.go; it does not
      exist, and the two ad-hoc tails it was to replace remain, each consulting the decider differently.
      plan.md:72 lists retireDetachedIncarnation under Pure entities though it observes PairSession and writes
      the thread store. The review contract rates a table/code contradiction Critical; rated Important here
      because the code is correct and tested and the divergence is in the plan's description. Needs a
      "## Revisions" entry or the refactor before archiving.
  - id: new
    severity: Important
    family: stale-artifact-claim
    title: |
      The issue's Revisions still assert StartResult.Warm, a sole Quiesce caller, and a 24-row table
    detail: |
      issue:169 says StartResult carries Warm (ownership actually comes from ActorRecord.Warm); issue:154 says
      quiescePostAckStart is the only caller of Artifacts.Quiesce (ArchiveThread is the second, detach.go:246);
      issue:113 says 24-row exhaustive table (36 rows shipped). The plan gate raised exactly these as PQ-5 and
      disposed them not-addressed in rounds 2, 3 and 4, and the close Log's carry-forward list drops PQ-5
      silently. Fix: one correcting "## Revisions" entry plus a Log line disposing PQ-5.
  - id: new
    severity: Minor
    family: dead-branch
    title: |
      The DurableRetire arm of failTrackedPostAckStart is unreachable and duplicates couch.go's retire block
    detail: |
      At claim phase no shape yields DurableRetire (spawn returns early; cold and warm claims yield rollback or
      mark-unknown), so launch_existing.go:205 can never execute, and the ctx built at :190 exists only to feed
      it. Additionally context.WithoutCancel(context.Background()) is a no-op at both :190 and couch.go:522 --
      neither function receives the caller's context, so "uncancellable" comes from Background, not the wrapper.
  - id: new
    severity: Minor
    family: weak-assertion
    title: |
      Route 3's warm assertion is weaker than the plan promised
    detail: |
      warm_failure_test.go:167 asserts only that the row is not ThreadDetached; the plan promised
      ReasonSessionGone. Any wrong-but-not-detached state passes.
  - id: new
    severity: Minor
    family: shape-label-mismatch
    title: |
      AbortStarted labels a spawn StartColdResume, and rescans the registry it just searched
    detail: |
      couch.go:583 defaults to StartColdResume for any non-warm start, including a spawn (harmless today since
      both own their session, but the name is false); couch.go:584 runs a second full c.reg.Records() scan
      immediately after the identity loop found the same record, which the plan said would be reused.
```
