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

---

## Re-review — 2026-09-11T11:39:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 230 — A failed warm reattach deletes the session it was reattaching |
| repo | pair |
| issue file | workshop/issues/000230-a-failed-warm-reattach-deletes-the-session-it-was-reattaching.md |
| boundary | whole-issue close |
| milestone | — |
| window | d8dc14f60276d2dd708fea794e5e390abce9c9c4..a6b66189265fed26055574b795f34572e6876469 |
| command | sdlc close --issue 230 |
| reviewer | claude |
| timestamp | 2026-09-11T11:39:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The round-2 fixes hold. I checked each one by mutation or by reading the code, not the commit message. Reverting the retire fallback turns `TestAFailedRetireStillLeavesTheRecordRecoverable` red by name: the record keeps a live incarnation. Making the quiesce ignore ownership turns all six warm rows red. Labelling a spawn as a cold resume turns three spawn tests red. `applyStartCleanup` now exists and is the one cleanup path both entry points share. Build and vet pass, and the four affected packages pass unsandboxed. Two things keep this from SHIP:

- **A regression the refactor introduced.** The session-observation error that the old cold-resume code returned is now dropped. A probe test passes at base and fails at HEAD.
- **A second stale-claim finding.** Eight passages still describe decisions this issue reversed, and one of them is in the atlas.

## 1. Strengths

- **`couch.go:515` `applyStartCleanup`.** One cleanup path now serves both entry points (`failTrackedPostAckStart` and `failPostAckStart` are one line each). The pure decider is called once, and a failed retire falls through to `markLiveRecordUnknown` (`couch.go:549`). The prior round's main defect (BR-5) is closed and pinned by a test.
- **`warm_failure_test.go:326-346`.** The retire-failure test counts the session reads and asserts it reached the third one. It also asserts that the returned error is the retire's own error. That makes the race reproducible, and the test cannot pass without entering the branch it is named for. This is the ARCH-ORDER shape done right.
- **`startcleanup.go:44` `OwnsSession`.** It explicitly tests for the two owning shapes, so an empty or unrecognised shape counts as not owning. The table test includes a value no constant names.
- **`warm_failure_test.go:292`.** The test hands back `StartColdResume`, a plausible wrong value, rather than an empty field that would prove nothing. A test like this can actually catch the guard missing.
- **`startcleanup_test.go`.** All 36 rows are written out as literals, with a count check and a duplicate check, and three property tests cover the whole input space.

## 2. Critical findings

None.

## 3. Important findings

**I1: the session-observation error is dropped (`launch_existing.go:191-204`, `couch.go:515-561`).**
- `observeSessionPresence` turns a `PairSession` error, or a missing `PairSessionIO`, into `PresenceUnobserved` and throws the error away. Nothing in `applyStartCleanup` reports it.
- The base code joined both `bindingErr` and `"exact Pair session observer is unavailable"` into the returned error.
- **Measured:** I ran a probe test (cold resume, acknowledgement fails, `BeforePairSession` returns "zellij unreachable"):
  - on a `git archive` of the base, the returned error is `…ack transport closed\nzellij unreachable` and the probe passes;
  - at HEAD, the error is only `…ack transport closed` and the probe fails.
- **Why Important and not Critical:** the operation still fails, and the record still ends up Unknown, which is the safe state. What's lost is the reason it couldn't be rolled back. The new warm live-record branch has the same gap. This goes against ARCH-SECURE's "degrade visibly".
- **Fix:** make `observeSessionPresence` return `(SessionPresence, error)` and join that error in `applyStartCleanup`. Then assert `"zellij unreachable"` in `TestResumeUnobservableSessionKeepsUnknownOccupied`; that assertion fails without the fix.

**I2: stale claims, the 2nd finding in family `stale-artifact-claim`.**
The first round fixed instances, so this finding states the rule instead. **Rule:** a commit that reverses a design decision must remove every restatement of the old decision in the same commit. Grep for the retired names and claims (`Warm`, "zero value", `WithoutCancel`, "whether to quiesce", `StartCleanup`) across code comments, test comments, atlas and plan. Fix each hit, or record it in a plan or issue `## Revisions` entry.

Measured prevalence is **8 sites in 5 files**:
- `atlas/couch.md:679` says the decider returns "whether to quiesce". It returns only a `DurableAction`. This is the part of BR-6 that was not done.
- `atlas/couch.md:686`, `couch.go:613-616` and `couch.go:623-626` say the zero value of the handed-back struct is the destructive branch. The `couch.go` comment is duplicated about ten lines apart. In fact an empty `Shape` counts as not owning, which is the safe side. The real risk is a wrong *non-empty* value, which `warm_failure_test.go:287-292` itself says.
- `warm_failure_test.go:271-276` and `:298` still describe the risk as a zeroed `Warm` field, but the test hands back `StartColdResume`.
- `detach.go:149-151` says cleanup callers pass `context.WithoutCancel`. The only cleanup caller passes `context.Background()` (`couch.go:540`).
- `plan.md:194-200` and the Task 5 mutation row (`:353`) still describe `WithoutCancel`, and no Revisions entry covers that change.

## 4. Minor findings

- `observeSessionPresence` now also runs for a spawn, even though the decider ignores presence for spawns. That adds one zellij query to each spawn failure. It only affects failure paths.
- `applyStartCleanup`'s switch doesn't cover every action and phase pair: `DurableRollback` with `liveRecord` would call `rollbackTrackedStart` with an empty nonce. The decider never produces that pair today.
- The plan's task checkboxes are still `- [ ]`. The issue's Plan checkboxes are ticked.

## 5. Test coverage notes

- **Sandboxed run:** the `pty` variants fail with `ptychild: … operation not permitted`, which is the known sandbox limit. The `stdio` variants pass.
- **Unsandboxed run** (`env -u PAIR_SESSION_ID -u PAIR_TAG`): `couchcore`, `couchcmd`, `couchtty` and `artifactpath` all pass.
- **Mutations** (applied with `go test -overlay`, repository untouched; each confirmed applied before running):

  | Mutation | Tests that went red |
  |---|---|
  | Retire fallback reverted | `TestAFailedRetireStillLeavesTheRecordRecoverable` |
  | Quiesce ignores ownership | all 6 warm rows, `TestPostAckCleanupEndsItsHelperWithoutOwningTheSession`, `TestAbortStartedReadsOwnershipFromTheRegistryNotTheCaller` |
  | Spawn labelled as cold resume | `…NeverLeaveWorkspaceWriter/registration_evidence_failure`, `TestSpawnAcknowledgementFailureCancelsHelperBeforeRollback`, `TestSpawnPossiblyDeliveredAcknowledgementQuiescesBeforeRollback` |

- **Gap:** no test pins the observation error in the returned error (I1). The probe's assertion is ready to use.

## 6. Architectural notes

- **ARCH-DRY: pass.** There is one shared retire, one cleanup path and one registry scan. The only duplication left is the doubled `AbortStarted` comment, which is part of I2.
- **ARCH-PURE: pass.** `DecideStartCleanup` is pure and tested without IO. `applyStartCleanup` gathers the observations, calls the decider once and acts on the result.
- **ARCH-PURPOSE:** pass on the code, flag on the prose.
  - All three places that decide ownership read it from the same source. `quiescePostAckStart` and `DecideStartCleanup` call `OwnsSession`; `AbortStarted` reads the shape from couch's own registry record. All six routes share that path.
  - The prose sweep is I2.
- **ARCH-MOCK: pass.**
  - The fake now models both a successful deletion and a refused one, through the same interface production uses.
  - A live check against the real binary exists: `TestSessionQuiescenceLive` in `launcher/session_quiescence_live_test.go:56`.
- **ARCH-CONSTRAINTS: pass.** The change only touches failure paths.
- **ARCH-SECURE:** flag for I1.
  - The trust side passes. The shape is read from couch's own registry, and an empty or unknown shape counts as not owning.
  - Old registry files decode `shape` as empty, but they can never reach `AbortStarted`, because a handle only comes from a start made by this process.
  - An older binary ignores the new field, since `json.Unmarshal` is used without strict decoding (`store.go:70`).
- **ARCH-ORDER: pass.**
  - The decider table enumerates every state and event.
  - A retire that fails partway now has a defined outcome (mark-unknown), and its race is reproduced deterministically.
  - The decider's live-helper rows can't happen in production, because `quiescePostAckStart` loops until the helper is quiet. They document the rule rather than test reachable behaviour.
- **For #206:** a burst of failed background reattaches will now leave rows that are detached again or Unknown. Until I1 is fixed, Unknown rows caused by an unreachable zellij won't say why.

## 7. Plan revision recommendations

1. Add a `## Revisions` entry saying cleanup passes `context.Background()`, since `applyStartCleanup` never receives the caller's context. It supersedes `:194-200` and the Task 5 `WithoutCancel` mutation row.
2. Add BR-2's missing non-goals:
   - `pair resume` can fall through to creating a session, which a warm cleanup then leaves behind.
   - If couch dies between killing the helper and writing the record, recovery is left to `reconcileInterruptedStarts`.
3. If I1 is fixed by changing `observeSessionPresence`'s signature, add one line recording it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan "The class, enumerated" names both Quiesce callers and why ArchiveThread (detach.go:246) is out of class; the issue's PQ-5/BR-8 Revision corrects the sole-caller claim.
  - id: BR-2
    disposition: not-addressed
    note: |
      The new Non-goals lists three different items; the pair resume create fall-through and couch dying between helper kill and durable write appear nowhere in plan or issue.
  - id: BR-3
    disposition: addressed
    note: |
      retireDetachedIncarnation is shared by Detach (detach.go:97) and applyStartCleanup (couch.go:540); ctx.Err() kept, and cleanup passes Background so cancellation cannot interrupt it.
  - id: BR-4
    disposition: not-addressed
    note: |
      Task 3 still enumerates rows in prose and still promises Spawn rows for routes 5-6 that the table does not have; archival, never blocks.
  - id: BR-5
    disposition: addressed
    note: |
      couch.go:549 falls through to markLiveRecordUnknown; reverting it reddens TestAFailedRetireStillLeavesTheRecordRecoverable by name (record keeps state=live).
  - id: BR-6
    disposition: addressed
    note: |
      Quiesce field gone, OwnsSession is the single source, and the ownership-ignoring mutation reddens all warm rows; the stale atlas sentence is carried in the new stale-artifact-claim finding.
  - id: BR-7
    disposition: addressed
    note: |
      applyStartCleanup exists (couch.go:515) as the single shell, retireDetachedIncarnation sits under Integration points, and the plan Revisions records StartCleanup removed and Shape as a string.
  - id: BR-8
    disposition: addressed
    note: |
      The issue Revision titled three-claims-are-wrong (PQ-5, BR-8) corrects StartResult.Warm, the sole-caller claim and the row count, and names PQ-5.
  - id: BR-9
    disposition: addressed
    note: |
      failTrackedPostAckStart is one line; the retire arm lives only in the shared shell at live-record phase; the WithoutCancel no-op is gone (its detach.go comment is in the new finding).
  - id: BR-10
    disposition: addressed
    note: |
      warm_failure_test.go:168 now requires ThreadUnusable with ReasonSessionGone, and passes.
  - id: BR-11
    disposition: addressed
    note: |
      ActorRecord.Shape records spawn/cold/warm and AbortStarted reads it in the identity loop with no second scan; labelling a spawn cold reddens three spawn tests.
findings:
  - id: new
    severity: Important
    family: swallowed-error-context
    title: |
      The shared cleanup shell drops the session-observation error that the old cold-resume tail returned
    detail: |
      observeSessionPresence (launch_existing.go:191-204) collapses a PairSession error, or a missing PairSessionIO,
      into PresenceUnobserved and discards it; applyStartCleanup (couch.go:515-561) never reports it. The base tail
      joined bindingErr and "exact Pair session observer is unavailable" into the returned error. Overlay probe
      (cold resume, ack fails, BeforePairSession returns "zellij unreachable"): base returns the ack cause plus
      "zellij unreachable" and passes; head returns only the ack cause and fails. The record still lands Unknown,
      hence Important not Critical, but the operator loses why it was not rolled back, and the new warm live-record
      arm inherits the silence (ARCH-SECURE, degrade visibly). Fix: return (SessionPresence, error), join it in
      applyStartCleanup, and assert "zellij unreachable" in TestResumeUnobservableSessionKeepsUnknownOccupied.
  - id: new
    severity: Important
    family: stale-artifact-claim
    title: |
      Eight comment and doc passages still restate design decisions this issue reversed, including the atlas
    detail: |
      2nd finding in family stale-artifact-claim, so this states the rule rather than one site. Rule: a commit that
      reverses a design decision sweeps every restatement of the old one in the same commit; grep the retired names
      and claims (Warm, zero value, WithoutCancel, whether to quiesce, StartCleanup) across code comments, test
      comments, atlas and plan, and fix each hit or record a Revisions line. Measured prevalence, 8 sites in 5 files:
      atlas/couch.md:679 says the decider returns whether to quiesce (BR-6 residue); atlas/couch.md:686,
      couch.go:613-616 and couch.go:623-626 (duplicated) say the relayed struct's zero value is destructive, but an
      empty Shape answers non-owning, the safe side; warm_failure_test.go:271-276 and :298 keep the Warm/zero-value
      framing though the test relays StartColdResume; detach.go:149-151 says cleanup passes context.WithoutCancel
      but it passes context.Background() (couch.go:540); plan.md:194-200 and its Task 5 row still describe
      WithoutCancel with no Revisions entry covering it.
```

---

## Re-review — 2026-09-11T11:57:29-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 230 — A failed warm reattach deletes the session it was reattaching |
| repo | pair |
| issue file | workshop/issues/000230-a-failed-warm-reattach-deletes-the-session-it-was-reattaching.md |
| boundary | whole-issue close |
| milestone | — |
| window | d8dc14f60276d2dd708fea794e5e390abce9c9c4..6c1c640e651be97922736cf3ac404332fb25e858 |
| command | sdlc close --issue 230 |
| reviewer | claude |
| timestamp | 2026-09-11T11:57:29-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code does what the Spec asks. On all six post-acknowledgement failure routes, a warm reattach now ends only its own `pair resume` helper. Its thread goes back to Detached, or to session-gone on route 3, where the session itself died. Spawn and cold resume still quiesce the session they created, and their record handling matches the base code branch for branch. I reverted each key guard in an overlay and each one's test went red. BR-12 is fixed and pinned. What blocks a clean SHIP is BR-13: the round-2 sweep fixed the eight named sites but left seven more in four files. One of them is `StartResult.Warm` in the plan, which was on the rule's own list and which the plan's round-2 Revision says was corrected. All seven are comment fixes. Everything else is Minor.

Heads-up, outside the window: while I was reviewing, a separate session committed `966e31fd` ("probes: measure how long the repaint nudge…", adding only `cmd/probes/zellijrepainttiming/main.go`) onto this issue branch. It is a child of `6c1c640e`, so every test and probe I ran used the reviewed couchcore code. It will ride along in the #230 PR unless someone moves it.

### 1. Strengths
- **The ownership gate is one check in one place.** `couch.go:665` (`if !shape.OwnsSession()`) is the only thing between a failed start and `Artifacts.Quiesce`. Disabling it turned all six warm routes red, plus `TestPostAckCleanupEndsItsHelperWithoutOwningTheSession`.
- **Ownership comes from the registry, not the caller.** `AbortStarted` takes the shape from its own record (`couch.go:617`, `:631`). When I made it trust the relayed shape, `TestAbortStartedReadsOwnershipFromTheRegistryNotTheCaller` failed at `warm_failure_test.go:298`, while the six-route table stayed green. That confirms this test is the only guard, as the Log says.
- **The fake now behaves like production.** `launcher.QuiesceThreadSession` (`thread_claim.go:257-281`) deletes the session but keeps its index entry, so a later `PairSession` returns `Present=false`. The fake's new `Quiesce` does the same (`artifactcollision_fake.go:83-89`). This is ARCH-MOCK done properly, and it is what caught the old ambiguous-ack test passing for the wrong reason.
- **The decision table cannot mirror the code.** It has 36 rows with expected values written out, plus a row-count and duplicate guard (`startcleanup_test.go:824-836`).
- **Detach's rules are reused, not copied.** `retireDetachedIncarnation` (`detach.go:118-175`) keeps Detach's `ctx.Err()` interrupt and its single clock read.
- **The new registry field is backward compatible.** `ActorRecord.Shape` uses `omitempty`, and the registry is read with the lenient `json.Unmarshal` (`store.go:70`), so older and newer binaries can read each other's files. An unrecognised shape counts as not owning the session.

### 2. Critical
None.

### 3. Important
- **BR-13 is still open: seven leftover passages in four files.**
  1. `plan.md:216-217` (ARCH-SECURE): "The one caller-relayed copy (`StartResult.Warm`) is read by nothing destructive." This name was on BR-13's list, and the plan's Revision at `:419-422` says "All corrected".
  2. `startcleanup.go:20-21`: says a spawn "goes to failPostAckStart for BOTH phases (launch_existing.go's `if !resume` arm)". `a6b66189` deleted that arm. A spawn's claim phase now goes `failTrackedPostAckStart` → `applyStartCleanup` → `markLiveRecordUnknown`.
  3. `startcleanup.go:102-103`: "failPostAckStart reconciles against registration evidence".
  4. `startcleanup.go:106-107`: "An owning one keeps failPostAckStart's tail". That work is now `markLiveRecordUnknown`.
  5. `startcleanup_test.go:27-28`: repeats the `if !resume` claim.
  6. `startcleanup_test.go:43-44`: "takes failPostAckStart's tail".
  7. `couch.go:498-501`: says `failPostAckStart` "owns every error exit… before Spawn transfers the handle". It now owns only route 5 and route 6, and route 6 comes *after* the handoff. Routes 1–4 belong to `failTrackedPostAckStart`.

  This family (`stale-artifact-claim`) has now come up three times. The sweep's search list was copied from the finding's examples instead of derived from the diff. **Rule:** build the list from what the range removed or re-scoped: every deleted identifier or branch arm (`git diff -U0 BASE HEAD -- '*.go' | grep '^-'`), plus every function whose callers changed. Then grep code comments, test comments, the atlas and the plan for each one. Here that adds `if !resume` and `failPostAckStart` as the spawn or owning tail, both retired by the shell refactor.

### 4. Minor
- **BR-2 is still open.** The new Non-goals section (`plan.md:223-234`) lists three *different* items. Of BR-2's three, item (2) is covered at `plan.md:54-61`. Two are still unstated anywhere in the plan, issue or atlas:
  - item (1): if the session dies between `confirmStillDetached` (`resume.go:425`) and exec, `pair resume` falls through to `ActionCreate` (`launcher/decision.go:74,103`). The "warm" start then owns a session it made, and non-owning cleanup leaks it rather than deleting it.
  - item (3): couch dying between killing the helper and the durable write.
- **BR-4 is still open, and Task 3 has gone stale again.** Task 3 (`plan.md:294-316`) is still per-row prose. It promises "for routes 5–6 also a `Spawn`" and an owning version of every row. `TestAFailedOwningStartStillQuiescesItsSession` (`warm_failure_test.go:177-205`) covers cold resume only and skips route 3, and no Revisions entry says so.
- **New — cleanup reports the session read even when the decision ignored it** (new family `diagnostic-scoped-to-decision`).
  - `applyStartCleanup` reads the session and joins any read error for every shape (`couch.go:521-522`). That includes spawn, which never uses presence, and a warm claim, which rolls back either way.
  - A probe showed a spawn whose acknowledge fails now makes one `PairSession` call and returns `ack transport closed` followed by `exact Pair session binding is absent for {…}`. The second line is new compared with base, and it explains nothing about the outcome. In production it can also cost a `zellij list-sessions` call.
  - The comments at `couch.go:519-520`, `launch_existing.go:192-195` and `warm_failure_test.go:362-364` call this error "the operator's only account of why the thread was left occupied". But the test that pins it uses a warm claim, which rolls the thread back, and it never checks the record's outcome.
  - The plan said to read the session "when the decider needs it" (`plan.md:183-184`).
  - **Fix:** read the session only for cold claims and warm live records, the cases where the decision uses it. Move the "zellij unreachable" assertion into `TestResumeUnobservableSessionKeepsUnknownOccupied` (`resume_launch_test.go:168`), where the read actually decides the outcome.
- **New — one retire exit still skips the fallback. This is the 2nd finding in family `fail-closed-fallback-missing`.**
  - `couch.go:537-540`: the `DurableRetire` arm returns on a `GetThread` error before trying any recovery. That can leave a live incarnation behind a helper that is already dead.
  - So the atlas claim at `atlas/couch.md:685-686`, "no path leaves a live incarnation behind a dead helper", is too strong.
  - **Rule:** at the live-record phase, any exit that has not retired the incarnation falls through to `markLiveRecordUnknown`. Build that as one fallback after the switch, not arm by arm.
  - One site remains.

### 5. Test coverage notes
- **Sandboxed run** of `couchcore`, `couchcmd` and `couchtty`: every failure is a pty or ptychild "operation not permitted", which the sandbox always causes. The stdio variants pass. I did not rerun the full suite unsandboxed; the Log reports 197 packages passing there.
- **Mutations I ran:**
  - The two BR-12 mutations both went red.
  - Quiesce ignoring ownership: all six warm routes and the helper-only test went red.
  - `AbortStarted` trusting the relayed shape: the registry test went red.
- **`DecideStartCleanup` gets no row for an unrecognised shape.** It treats one as warm: rollback at the claim phase, retire-or-unknown at the live-record phase. Only `OwnsSession` is tested with unknown values, so the atlas line "whole input space is table-tested" is true only for the three named shapes.
- **`TestAFailedRetireStillLeavesTheRecordRecoverable` depends on an exact read count.** It needs exactly three `PairSession` reads to reach the branch it tests. It fails loudly if that changes, but any new read on the warm live-record path will make it fail.

### 6. Architecture principles, one by one
- **ARCH-DRY: pass.** One cleanup shell, and Detach's retire rules are shared rather than copied.
- **ARCH-PURE: pass.** The decider and `OwnsSession` are pure and tested without IO.
- **ARCH-PURPOSE: pass on the routes, flag on the doc sweep (BR-13).** I checked the route class directly:
  - only `quiescePostAckStart` and `ArchiveThread` call `Artifacts.Quiesce`;
  - only `couchcmd/run.go:464` calls `AbortStarted`;
  - all five post-ack exits in `launchTrackedThread` go through the shell.
- **ARCH-MOCK: pass.** The fake matches production. The SIGKILL case is still reasoned rather than tested, and the plan's open question records that.
- **ARCH-CONSTRAINTS: pass.** Only failure paths changed, apart from the extra spawn read above.
- **ARCH-SECURE: pass.** Ownership comes from couch's own state, and the new field decodes leniently.
- **ARCH-ORDER: pass.** Each route's ordering is injected through fake-seam hooks, so it can be reproduced. The retire loop is bounded and runs on `Background`. The one ordering nothing models is couch dying mid-cleanup (BR-2 item 3).

### 7. Plan revision recommendations
- Fix or add a Revisions line for `plan.md:216-217` (`StartResult.Warm`). The round-2 Revision says every instance was corrected.
- For Task 3, add a Revisions line: the owning table is cold-resume only, route 3 is skipped, and there are no Spawn rows (the existing spawn tests cover spawn). Or shrink Task 3 to one strategy line, as BR-4 asked.
- Add BR-2's items (1) and (3) to Non-goals.
- If you scope the session read to the cases that need it, add a Revisions line saying so. That would make the code match `plan.md:183-184` again.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      The new Non-goals (plan.md:223-234) lists other items; (1) the ActionCreate fallthrough after confirmStillDetached (resume.go:425, launcher/decision.go:74,103) and (3) couch dying between helper kill and durable write are still unstated anywhere.
  - id: BR-4
    disposition: not-addressed
    note: |
      Task 3 (plan.md:294-316) is still per-row prose and stale again: it promises Spawn owning rows for routes 5-6 and an owning route 3; the shipped owning table is cold-resume only and skips route 3, with no Revisions line.
  - id: BR-12
    disposition: addressed
    note: |
      Verified by reverting: dropping the error in observeSessionPresence, or the join at couch.go:522, turns TestCleanupSurfacesWhyItCouldNotObserveTheSession red.
  - id: BR-13
    disposition: not-addressed
    note: |
      The 8 named sites are fixed but 7 remain in 4 files: plan.md:216-217 still names StartResult.Warm (on the rule's own list), and startcleanup.go:20-21,102-103,106-107, startcleanup_test.go:27-28,43-44 and couch.go:498-501 describe the deleted if-!resume arm and failPostAckStart as the spawn/owning tail. Rule, 3rd time in this family: derive the grep list from what the diff deleted or re-scoped, not from the finding's examples.
findings:
  - id: new
    severity: Minor
    family: diagnostic-scoped-to-decision
    title: |
      applyStartCleanup joins the session-observation error for shapes whose decision never reads presence
    detail: |
      couch.go:521-522 observes and joins for every shape; probe: a spawn whose acknowledge fails now also returns "exact Pair session binding is absent", which base never produced. The comments at couch.go:519-520, launch_existing.go:192-195 and warm_failure_test.go:362-364 say it explains why the thread was left occupied, yet the pinning test's warm claim rolls back. Observe only where the decider reads presence (cold claim, warm live record), and move the assertion to TestResumeUnobservableSessionKeepsUnknownOccupied.
  - id: new
    severity: Minor
    family: fail-closed-fallback-missing
    title: |
      The DurableRetire arm returns on a GetThread error before any disposition, so the atlas "no path" claim overclaims
    detail: |
      2nd finding in this family. Rule: at the live-record phase, every applyStartCleanup exit that has not retired the incarnation falls through to markLiveRecordUnknown, as one structural fallback after the switch rather than per arm. Measured prevalence: 1 remaining site, couch.go:537-540, which contradicts atlas/couch.md:685-686.
```

---

## Re-review — 2026-09-11T12:25:26-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 230 — A failed warm reattach deletes the session it was reattaching |
| repo | pair |
| issue file | workshop/issues/000230-a-failed-warm-reattach-deletes-the-session-it-was-reattaching.md |
| boundary | whole-issue close |
| milestone | — |
| window | d8dc14f60276d2dd708fea794e5e390abce9c9c4..87fea1c4cc5e3a2bbe219c9233b76865749dc925 |
| command | sdlc close --issue 230 |
| reviewer | claude |
| timestamp | 2026-09-11T12:25:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix does what the issue asks: a failed warm reattach no longer deletes its session. I checked this by running the code, not by reading the commit messages. In a scratch copy, re-enabling the quiesce on the warm path turns four tests red. Making `AbortStarted` trust the caller's shape turns the registry test red. The fake's new quiesce effect matches production: `QuiesceThreadSession` deletes the session but keeps the index entry, and production `PairSession` then reports `Present=false` with the name, not an error. So the rewritten `TestResumeAmbiguousAckRollsBackOnceItsSessionIsGone` pins a branch production really takes. The spawn and cold-resume cleanup paths still behave as before, and the round-3 fixes for BR-13 and BR-14 hold. Nothing blocks. The rest are all Minor: BR-15's fix is structural but no test pins it, the rollback arm still makes one atlas sentence false, BR-2 and BR-4 are still open, and I found four small new items.

## 1. Strengths
- **The pure decider is exhaustively tested.** `DecideStartCleanup` (`startcleanup.go:98-131`) has a 36-row table with the expected values written out, a duplicate-row guard and three property tests. The three-valued `StartShape` keeps the spawn and cold-resume cleanup paths separate.
- **The fake now models the deletion, and it matches production.** `artifactcollision_fake.go:239-270` matches `thread_claim.go:257-281` plus `artifactcollision.go:149-183`. It exposed a test that had been passing for the wrong reason.
- **Ownership is read from couch's own record.** `AbortStarted` takes the shape from the registry (`couch.go:634-659`). The test deliberately relays an owning shape, so it can fail, and it does (M6).
- **The retire-failure test proves it reached its branch.** `TestAFailedRetireStillLeavesTheRecordRecoverable` counts the session reads in order and asserts the retire's own error (M4 killed).
- **BR-14 is pinned in two places**: against the decider for the rule, and at the shell (M1 killed).

## 2. Critical
None.

## 3. Important
None open. BR-13 is addressed; see the dispositions block.

## 4. Minor
- **Rollback arm (3rd finding in `fail-closed-fallback-missing`).**
  - `couch.go:548-549` returns `rollbackTrackedStart`'s error with no fallback. That makes `atlas/couch.md:686-688` ("every exit… falls through") and the comment at `couch.go:561-566` ("structural rather than per-arm") false.
  - **The rule:** only a completed undo may return early, and the docs claim exactly the exits the fallback covers.
  - **Impact is low.** `ReconcileStart` later rolls back or promotes a leftover claim whose recorded helper is dead.
  - **Fix:** scope both claims to the live-record phase.
- **Fake quiesce has no live conformance check (ARCH-MOCK).** The transition "`PairSession` reads `Present=false` with the name kept after quiesce" has no couchcore live test. Launcher's live tests only pin `DeleteSession`'s own post-condition.
- **Error text says "detach" for a failed reattach.** `detach.go:127,130` produce "…during detach" on the cleanup path too, and the test asserts that misleading wording. Make the messages neutral, or have each caller wrap them.
- **Unrelated commit on the branch.** Commit `966e31fd` (the `zellijrepainttiming` probe, plus `manifest.go:729` and `.gitignore`) has nothing to do with #230. It has no `side-quest:` verb and the issue doesn't mention it.

## 5. Test coverage notes
- **Targeted tests** (real repo, sandboxed, `env -u PAIR_SESSION_ID -u PAIR_TAG`): all pass except the pty-child tests, which the sandbox blocks.
- **Reverting each fix in a scratch copy** (`git archive`, compared against a baseline; the real repo was untouched):

| Mutation | Result |
|---|---|
| M1: observe session presence for every shape | killed |
| M2: swallow the observation error | killed |
| M3: retire-arm `GetThread` error returns early (BR-15) | **survived** |
| M4: retire failure returns early | killed |
| M5: warm path quiesces again | killed by 4 tests |
| M6: `AbortStarted` trusts the relayed shape | killed |

- `gofmt` and `go vet` are clean.
- I did not run the unsandboxed `make test`; the issue Log reports 197 packages ok.

## 6. Architecture principles and upcoming work
- **ARCH-DRY: pass.** `retireDetachedIncarnation` is shared by `Detach` and cleanup, and there is one cleanup shell. `startCleanupReadsPresence` restates the decider's branches, but a test derived from the decider keeps the two in step.
- **ARCH-PURE: pass.**
- **ARCH-PURPOSE: pass.** I swept every session-deleting call. `Artifacts.Quiesce` has two callers: `ArchiveThread` (deliberate) and `quiescePostAckStart` (fixed). `TriggerQuit` is the operator's quit gesture. Launcher's `createflow` deletions happen inside the pair child, and the warm argv omits the layout flag that would reach them. No route is missing.
- **ARCH-MOCK: flag (Minor)**, the missing live check above.
- **ARCH-CONSTRAINTS: pass.** Session observation is now scoped, and the retire is bounded at 32 attempts.
- **ARCH-SECURE: pass.** An unrecognised persisted shape answers non-owning, and the registry decode tolerates the new field.
- **ARCH-ORDER: pass, with one gap.** The transition table is written out explicitly, and cleanup runs on `Background` so a cancellation can't strand it. The `GetThread`-failure ordering has no injection seam, because `Threads` is a concrete `*ThreadStore` (this is why BR-15 is unpinned).
- **Upcoming:** #206 M2's background pass will mostly hit route 6. A new `StartShape` defaults to non-owning, so the decider table has to grow with it.

## 7. Plan revision recommendations
- **Task 3:** compress it to one strategy line. Drop the "owning Spawn for routes 5–6" claim, which the tests don't contain. Note that route 3 is warm-only.
- **Non-goals:** add BR-2's items (1) and (3).
- **Rollback arm:** if you scope the claim, add a Revisions line saying the fallback covers the live-record phase only.
- **Issue:** add a Revision correcting "Reachable today" bullet 2 (`issue.md:48-50`), which `plan.md:54-61` refutes.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      A Non-goals section now exists (plan.md:224-235) but lists other behaviours. Item (2) is stated at plan.md:54-61. Items (1), a warm-labelled start owning a session that pair resume created, and (3), couch dying between helper kill and durable write, are still unstated. issue.md:48-50 still claims a slow pair resume reaches the warm registration timeout.
  - id: BR-4
    disposition: not-addressed
    note: |
      Task 3 (plan.md:295-317) still enumerates per-row injections. It also claims an owning Spawn half for routes 5-6 (plan.md:307-308) that TestAFailedOwningStartStillQuiescesItsSession does not contain; spawn is covered by pre-existing tests.
  - id: BR-13
    disposition: addressed
    note: |
      All eight named sites are corrected (atlas/couch.md:677-680 and 692-694, the single AbortStarted comment at couch.go:652-658, the test relaying StartColdResume, detach.go:149-151, plan.md:193-200). The remaining plan-body mentions are covered by the Revisions at plan.md:379-423, and a grep finds no stale code or test comments.
  - id: BR-14
    disposition: addressed
    note: |
      The observation is gated at couch.go:531. M1 (observe for every shape) turns TestASpawnsCleanupNeverAsksAboutTheSession red, and M2 (swallow the error) turns TestResumeUnobservableSessionKeepsUnknownOccupied red.
  - id: BR-15
    disposition: not-addressed
    note: |
      The fallthrough is structural (couch.go:546-547), but reverting only this arm to an early return leaves the package green (M3, scratch copy). No test pins it, and pinning needs a one-shot GetThread fault seam because Threads is a concrete ThreadStore.
findings:
  - id: new
    severity: Minor
    family: fail-closed-fallback-missing
    title: |
      The rollback arm still returns before any disposition, so the atlas claim that every exit falls through remains false
    detail: |
      This is the 3rd finding in the family. The rule: in applyStartCleanup only a completed undo may return early; a failed undo joins its error and reaches the phase's disposition, and the atlas and code comment claim exactly the exits that structure covers. One site remains: couch.go:548-549 returns rollbackTrackedStart's error directly, which contradicts atlas/couch.md:686-688 and the comment at couch.go:561-566 ("structural rather than per-arm"). Impact is low, because ReconcileStart later rolls back or promotes a claim whose recorded helper is dead, and cold resume behaved this way before pair#230. The simplest fix is to scope both claims to the live-record phase and name the claim-phase rollback as left to reconcileInterruptedStarts. Making the return conditional would change cold resume, which the Spec says stays unchanged.
  - id: new
    severity: Minor
    family: fake-effect-needs-live-conformance
    title: |
      The fake's new Quiesce effect has no live conformance row, though the warm table and the rewritten cold test rest on it
    detail: |
      By code reading the fake agrees with production: QuiesceThreadSession deletes the session but keeps the index entry (thread_claim.go:257-281), and PairSession then reports Present=false with the name kept (artifactcollision.go:149-183). Launcher live-tests only DeleteSession's own post-condition. Add a couchcore live row: quiesce a real session, then assert PairSession returns the name with Present=false (not an error) and DetachedSessions returns none. That locks the transition TestResumeAmbiguousAckRollsBackOnceItsSessionIsGone now depends on (ARCH-MOCK).
  - id: new
    severity: Minor
    family: extracted-helper-caller-specific-text
    title: |
      retireDetachedIncarnation's errors say "detach" when the caller is start cleanup
    detail: |
      detach.go:127 and :130 give "observe Pair session after detach" and "lost its Pair session during detach". A failed reattach whose cleanup retire fails shows that message to an operator who never detached, and TestAFailedRetireStillLeavesTheRecordRecoverable asserts the misleading string. Make the messages operation-neutral, or have each caller wrap them, and update the assertion.
  - id: new
    severity: Minor
    family: undeclared-scope-creep
    title: |
      Commit 966e31fd (zellijrepainttiming probe) is unrelated to pair#230 but rides this branch undeclared
    detail: |
      It measures repaint timing for a transition-animation question, and adds manifest.go:729 and a .gitignore line. It has neither a side-quest verb (AGENTS.md section 12) nor a mention in the issue. Move it to its own branch, or record it in the issue Log as a side-quest so the PR does not carry it silently.
```
