# Boundary Review — pair#256 (milestone M2)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2d137941e1ccc9a28b1f83683e41e9ba2295a963..921af399c5c9e87ba4bf18c03c8fc86f88264963 |
| command | sdlc milestone-close --issue 256 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-17T13:57:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 is substantively strong work: every guard I mutation-tested went red when reverted (`startInFlight`'s `!= Dead` probe, its `Unknown` fail-closed arm, the `SessionUnresolved && VerifiedPark == nil` asymmetry, archive's first-look comparison, the fake's sentinel wrap, the ledger-gate widening), the consolidation of `retireDeadIncarnationBeforeStart` into `clearLifecycleDebris` is the right ARCH-DRY move, and archive's read-before-write ordering is a genuine ARCH-ORDER improvement. What blocks SHIP is that Task 6b widened the **producer set of the `ThreadParked` state** and re-derived only the consumers of `ThreadEvidence.Parked`. One consumer of the state was missed: the switcher offers `switch-agent` on every `parked` row, and `PrepareAgentSwitch` still demands `record.VerifiedPark != nil` — so a ledger-parked row is offered an action that always fails, with the message "thread is not verified parked". I reproduced this through the production gather path (probe below). Secondarily, `DecideResume`'s new cold-refusal branch — the one the plan and atlas say preserves the tombstone as "a better answer than unbound" — is unreachable from the only production caller, verified by instrumenting the call site and running the whole `couchcore` suite with zero hits.

Verification I ran: `go build ./...` clean; `go test ./cmd/internal/{couchcore,couchcmd,artifactpath,threadrecord}` with the retention-env scrub — every failure is `ptychild: ... operation not permitted` (sandbox pty restriction), no logic failures. Mutation checks were run in a `git archive` scratch tree; the working tree was not modified.

## 1. Strengths

- `cmd/internal/couchcore/lifecycledebris.go:52` — moving the four-round-hardened rule to one function that both resume and archive call, instead of re-deriving it inside `DecideRecovery`, is the correct read of ARCH-DRY, and the screen-everything-before-the-first-write structure survives the move intact.
- `cmd/internal/couchcore/detach.go:268-281` — asking `RecoverySessionRefusal` before `clearLifecycleDebris` is the round-2 rule applied correctly (irreversible step never precedes a revocable check), and the triple-look comparison at `:330-334` is pinned: removing the `first`-look arm reds `TestRecoveryArchiveRefusesSessionAppearingBeforeStop/{absent-to-present,session-replaced}`.
- `cmd/internal/couchcore/artifactcollision_fake.go:252-256` — the fake's error now *wraps* the sentinel rather than merely reading the same. Un-wrapping it in a scratch tree reds `TestSpawnedButNeverBoundThreadIsArchivable`. This is exactly the ARCH-MOCK failure mode (a fake that lies in a way `errors.Is` can see) caught and closed.
- `cmd/internal/couchcore/startclaim_test.go:98` — `TestStartClaimDomainsCoincideAndTheGapFailsClosed` pins the `startClaimed`/`CurrentStartTransaction` domain gap **through the real store** rather than from a reading of `validateLifecycle`, which is precisely the M1 round-4 lesson applied forward. The `Liveness(0) != Unknown` assertion at the top is the belt to that braces.
- `cmd/internal/couchcore/recovery_execute.go:54-63` — deleting the "open transactions need no external session probes" early return removes the same structural defect M1 removed from `gatherThreadEvidence`; the bookkeeping no longer decides whether the world gets asked.

## 2. Critical findings

**C1 — `switch-agent` is offered on ledger-parked rows and always refuses.** `cmd/internal/couchcore/switchagent.go:110-112`

Task 6b widened who produces `ThreadParked`: a record with no `VerifiedPark`, no incarnation and no park now classifies `parked` when its ledger resolves. `cmd/internal/couchtty/menu.go:1256` offers `{resume, switch-agent, archive, name, describe}` for every `ThreadParked` row, and `PrepareAgentSwitch` falls through `hasOccupiedIncarnation` to `else if record.VerifiedPark == nil` and refuses. Reproduced through the production gather path:

```
classified state = "parked" reason = "" VerifiedPark=<nil>
PrepareAgentSwitch -> switch-agent: thread is not verified parked; attach or recover it first
```

The #272 shape hits the same wall by the other arm: a record still carrying a dead-launcher incarnation makes `hasOccupiedIncarnation` true, and the `observeExactProcess(...) != Live` check refuses with "source is not a verified live actor". `menu.go:1254` states the contract this violates in its own words — *"offering an action that always fails is how a switcher teaches an operator to distrust it."* `README.md:470` independently documents the guard's contract as "in a live or **verified parked** thread's actions", confirming the guard was never widened with the state.

**This is the 4th finding in family `classification-not-authority`.** Earlier rounds fixed instances. Do NOT fix this instance alone — state the rule that covers all of them, and fix that. The rule: *when a state's producer set widens, its consumers are the enumeration, not the consumers of the evidence field that widened it.* Task 6b's Log records "**One consumer, re-derived**" for `ThreadEvidence.Parked`/`ParkedStatus`; `ThreadParked` itself changed meaning in the same commit and its readers were never listed. The enumeration is small and I measured it — seven readers: `startup.go:39` (rank, intended), `startup.go:93` (`PathHoldsUsableThread`, intended), `actionableinventory.go:175` (`Resumable`, intended), `menu_render.go:430` (label), `run.go:768` (label), `menu.go:1166` (`menuThreadActionable`, intended), `menu.go:1256` (**broken**). Write that enumeration down and sweep it; a states×actions offered⇒permitted test (M3 Task 8 already sketches one for archive) would make the class mechanically checkable rather than re-derived per milestone.

**C2 — The plan's Core-concepts tables contradict the code at HEAD.** `workshop/plans/000256-lifecycle-transition-authority-plan.md:165` and `:228`

I grepped all 20 rows of the *Pure entities* and *Integration points* tables; two contradict the tree:

| Table row | States | Code at HEAD |
|---|---|---|
| `startInFlight` @ `actionableinventory.go` | `deleted` / M1 | **exists** — re-introduced by M2 Task 4 (`actionableinventory.go:441`) |
| `retireDeadIncarnationBeforeStart` @ `resume.go` | `new` / M1 | **absent** — renamed and moved to `clearLifecycleDebris` @ `lifecycledebris.go:52` |

The prose at `:211` compounds the first (*"`startInFlight` is deleted and replaced by `startClaimed`"* — M2 did the reverse). Four M2 entities have no row at all: `clearLifecycleDebris`, `ErrThreadRolledBack`, `RecoverySessionRefusal`, `coldResumeAuthorized` (plus `ThreadEvidence.StartOwner` and `ResumeStarting`). The `## Revisions` entry for M2 opening does not touch either table.

**This is the 5th finding in family `plan-code-divergence`.** Do NOT fix the two rows. The rule: *the Core-concepts tables are a derived view and must be re-derived by grep at every boundary close, not hand-edited per task.* The plan already asserts this discipline at `:150-152` ("Re-derived against the code at the M1 boundary … every row's name and path grepped") — it just isn't a step in any task. Measured prevalence at this boundary: 2/20 rows false, 6 entities unlisted. Make the re-derivation an explicit step in the milestone-close checklist, or drop the tables and point at the greppable symbols the way Task 2 already does for the branch table.

## 3. Important findings

**I1 — `DecideResume`'s cold-refusal branch, and `ResumeTombstoned`'s only producer, cannot fire in production.** `cmd/internal/couchcore/resume.go:142-161`

`ResumeContextWith` is the sole production caller of `DecideResume`. When `!detached` it calls `c.resumeEvidence` (`resume.go:449`) first, and every `NativeBindingResolver` — production `SessionInventoryNativeBindingResolver` (`resume.go:357-359`), `ScopedThreadArtifactCollisionChecker`, and the fake (`artifactcollision_fake.go:214-217`) — returns `refuseBinding(code)` whenever the diagnostic is non-empty. So the resume returns before `DecideResume` ever sees an unauthorized binding. I confirmed this by inserting a panic guarded on `!detached && !coldResumeAuthorized(binding)` immediately above the `DecideResume` call and running the full `couchcore` suite: **0 hits**, suite green.

Consequence: the behaviour the plan (`:733-741`) and `atlas/couch.md:1478-1482` both commit to — *"the tombstone scan … survives only as the better explanation where there is nothing to resume into"* — never reaches the operator; they get `unbound`, the answer the comment at `resume.go:143` explicitly calls worse. `resume_launch_test.go:270` even asserts the production path does *not* produce `ResumeTombstoned`, so nothing observes the gap.

**This is the 3rd finding in family `vocabulary-entry-without-producer`.** Do NOT just move the tombstone scan. The rule M1 round 3 already wrote — *"a vocabulary has a produced-by guard; a value nothing emits is a branch no test can reach"* — needs its one-level-up form: *a diagnostic code needs a producer reachable from a production entry point, and `ResumeDiagnosticCode` has no produced-by guard at all* (M1 round 3 recorded that absence and it was never added). Adding the guard would have caught this at authoring time. The fix for the instance is to let the cold path carry the resolution to `DecideResume` instead of erroring at `resumeEvidence`, or to delete the branch and stop documenting it.

**I2 — The `parked` referent changed and its user-facing and comment homes were not re-derived.** `README.md:458`

M2 made `parked` mean "the ledger resolves a conversation", not "a verified park receipt exists". Six homes still state the old referent; I enumerated them:

| Site | Says |
|---|---|
| `README.md:457-458` | "Rows expose only proven `live`, exact **verified** `parked`, and proved `detached` states" |
| `README.md:470` | "In a live or **verified parked** thread's actions, **switch coding agent** …" (see C1) |
| `cmd/internal/couchcore/resume.go:365` | exported `Resume` doc: "reoccupies one **verified parked** address" |
| `cmd/internal/couchcore/resume.go:126` | "cold rests on … a **verified park** with resolvable authority" (corrected only by the addendum below it) |
| `cmd/internal/couchcore/startup.go:138` | "a **verified-park** row there classifies `unknown`" — now every session-less resume-shaped row does |
| `cmd/internal/couchcore/actionableinventory.go:491` | same sentence, same staleness |

Separately, `cmd/internal/couchtty/menu.go:1236-1238` still carries the twin the plan's Task 2 explicitly assigned to Task 4 — *"would file a record mid-park … It resolves on its own"* — whose premise M1 disproved (`busy` is never a park) and whose second clause M2 disproved (an `Unknown` owner keeps the row busy indefinitely by design).

**This is the 5th finding in family `stale-wording-after-referent-change`.** Do NOT fix the six sites. The rule, and the gap: M1's round-5 log records re-deriving "in every home -- atlas, the function comment, the plan". **`README.md` and the exported-API doc comments were not in that enumeration of homes, and that is why they are stale now.** Fix the enumeration: the homes of a referent are atlas + plan + the function comment + **every exported doc comment naming it** + **README.md**, and a `git grep` of the old referent string is the mechanical check. This subsumes the Docs update gate for this boundary — README.md is not updated in this range and the surface it describes changed.

**I3 — M2's declared verification is not recorded.** `workshop/issues/000256-lifecycle-transition-authority.md:87`

The milestone is "make the operator's rows reachable **and verify against real sessions**", and the plan's Task 7 closes with three named operator-verification items (`:786-794`): the two `brain` rows archivable with no new orphan minted; #272's fixture *built* — kill a couch mid-thread, confirm `detached` **and that Enter actually reattaches** ("a row that merely reads `detached` proves nothing"); and the startup evidence-round timing. The M1 log explicitly deferred the wall-clock figure to M2 ("M2's operator verification owns the wall-clock figure"), and the plan's ARCH-CONSTRAINTS budget at `:274-280` asks for two figures in `## Log` (startup evidence round, one steady-state refresh, 7 records / 4 scopes). The `## Log` records only `go test ./...` and `make -k test`. None of the four is present.

**This is the 2nd finding in family `declared-measurement-not-recorded`.** The rule: *a plan item that names a measurement or a live verification is a deliverable of that milestone, not a note — it closes by a figure or an observation written into `## Log`, and "the tests are green" does not discharge it.* If the operator verification is deliberately deferred past the review to the close, say so in `## Log` as a deferral with an owner; silence reads as done.

## 4. Minor findings

- `cmd/internal/couchcore/warm_failure_test.go:168` — `!row.Resumable() && row.Reason != ReasonSessionGone` now accepts `ThreadDetached`, which the `sessionSurvives == false` premise excludes; the failure message says "resumable or unusable/session-gone" but the predicate is wider than that. Tighten to `ThreadParked || ReasonSessionGone`. (Family `test-name-contradicts-assertion`, 4th — the rule is that a restated assertion must be *re-derived from the premise*, not loosened until it passes.)
- `cmd/internal/couchcore/lifecycledebris.go:86,131` — `rollback` is a nonce string doing double duty as the "should I write" flag. The screen can decide to roll back and the write silently not happen if the nonce is ever empty; `componentPattern` in `threadrecord/record.go:174` makes that unrepresentable, but the code doesn't cite it, and M1's own rule is that an omitted guard cites the validator clause. A separate `clearStart bool` removes the coupling. (`sentinel-string-as-control-flag`)
- `cmd/internal/couchcore/lifecyclesequence_test.go:120-129` — the "session gone and conversation gone" and "session gone but ledger resolves" worlds declare **identical** `evidence`; the difference is injected inside the loop by `if world.wantState == ThreadParked`. The input is derived from the expectation, so the table no longer reads as a specification. Put `Parked` in the world literal. (`test-input-derived-from-expectation`)
- `cmd/internal/couchcore/lifecycledebris.go:122-126` — "a crash between them leaves a record the next attempt repeats safely" is asserted in a comment with no test. `newThreadStoreWithHooks` already provides an `AfterJournal`/`AfterTarget` seam that could fail the run between `AbandonPark` and `RetireIncarnation`/`DeleteStart`; ARCH-ORDER's at-review lens flags exactly this — the partial-progress path is a claim with a sample size of zero. (`partial-write-recovery-untested`)
- `cmd/internal/couchcore/actionableinventory.go:19-21` — `ThreadParked` is the one state constant with no doc comment, and it is the one whose referent changed this milestone. Its widened meaning currently lives only in the `Parked` field comment and the atlas.

## 5. Test coverage notes

Coverage of the new logic is good and I verified it adversarially rather than by inspection. Five mutations, all red:

| Mutation | Red |
|---|---|
| `startInFlight` → `startClaimed` | `TestClassifyThreadIsTotalOverEveryRecordShape` (2 rows), `TestTheWorldDecidesWhateverTheRecordSaysAboutItself` (4 worlds), `…AcceptsExactlyWhatTheOldProjectorAccepted` |
| `StartOwner != Dead` → `== Live` (drop fail-closed) | `…/start_whose_owner_could_not_be_probed` |
| drop `SessionUnresolved && VerifiedPark == nil` | `…/no_park_receipt_and_an_unaskable_session`, `TestAFailedSessionQueryLeavesTheRowUnknownRatherThanGone`, `TestACouchThatCannotObserveSessionsSaysUnknownRatherThanGone` |
| drop archive's `first`-look comparison | `TestRecoveryArchiveRefusesSessionAppearingBeforeStop` (both subtests) |
| un-wrap the fake's sentinel | `TestSpawnedButNeverBoundThreadIsArchivable` |
| re-gate the ledger read on `VerifiedPark != nil` | `TestEvidencePassAsksOnlyAboutResumeShapedRecords`, `TestSessionAbsentWithResolvableLedgerIsResumable`, `TestAFailedWarmReattachKeepsItsSession/3-registration-timed-out` |

Gaps: (a) no test crosses the classifier's output with the **menu's** action list and the guards behind it — that is what would have caught C1, and M3 Task 8 already sketches the shape for archive; extend it to `switch-agent` and `resume` at the same time. (b) `ResumeContextWith` has no test asserting which diagnostic an unresolvable cold resume produces end-to-end, which is why I1 is invisible. (c) ARCH-MOCK: the fake/production error-shape divergence was found by hand this round and nothing prevents the next one — a shared contract table run against both implementations would. (This is the 3rd in `production-seam-only-tested-through-fake`; the rule is that a fake's *refusals* are part of the seam's contract and need conformance, not just its successes.)

## 6. Architectural notes

- **ARCH-DRY — pass.** `clearLifecycleDebris` and `RecoverySessionRefusal` both consolidate rather than duplicate; the plan's reasoning for not re-deriving inside `DecideRecovery` is correct and load-bearing.
- **ARCH-PURE — pass.** `ClassifyThread`, `startInFlight`, `RecoverySessionRefusal`, `coldResumeAuthorized`, `parkedResumeProofMatches` are pure; IO stays in `gatherThreadEvidence`/`observeRecovery`/`clearLifecycleDebris`. `lifecyclesequence_test.go` runs the classifier with no fakes at all.
- **ARCH-PURPOSE — flag (C1, I1).** The shadow-sweep for `ThreadParked`'s new producer stopped at the evidence field. And a branch the plan commits to as the deliverable ("say WHY in the most useful terms available") is unreachable — the easy win (moving the classifier) landed, the stated purpose (the operator reads a better answer) did not.
- **ARCH-MOCK — pass with a gap.** The sentinel fix is the right one and is mutation-pinned; no conformance guard against the next divergence (§5c).
- **ARCH-CONSTRAINTS — flag (I3).** The ledger read went from 2/6 to 4/6 records per refresh by the guard's own count, and `ResumeContextWith` now pays a `DetachedSessions` (~250 ms `list-clients`) on **every** resume including cold ones that previously skipped it. Both are defensible and the code says why, but the plan asked for two measured figures and neither is in `## Log`. `TestWarmRowsAskNoLedgerQuestion` bounds the warm side well; nothing bounds the cold side as store size grows.
- **ARCH-SECURE — pass.** Records are treated as untrusted at the validator-accepted domain (`lifecycledebris.go:41`), every probe is an exact `{PID, Identity}`, and `Unknown` fails closed at every new decision point — all three verified red under mutation. No credentials in scope.
- **ARCH-ORDER — pass with a gap.** Archive's read-before-write sequencing and the three-look comparison are the strongest part of this diff, and the `(state, event)` reasoning in `lifecyclesequence_test.go`'s header is genuine rather than ceremonial. The gap is the interleaving nobody can reproduce: the crash between `clearLifecycleDebris`'s independent writes (§4), where a seam exists and is unused.
- **ARCH-FUNERAL — pass.** No new artifact family. `AbandonPark` appends a permanent `ParkHistory` tombstone and archive now does this routinely, but the record is archived immediately after; on the resume path the growth is one tombstone per orphaned park, bounded by park attempts. The decision to stop vetoing on tombstones is what keeps that append-only growth from becoming a permanent ban — that is the removal-path reasoning this entry asks for, and it is written down.

## 7. Plan revision recommendations

The plan needs one `## Revisions` entry, dated at this boundary, covering:

1. **Core-concepts tables re-derived at M2 (C2).** `startInFlight` moves from `deleted/M1` to `new/M2` (`actionableinventory.go`), and the prose at `:211` is corrected — M2 replaced `startClaimed` *with* `startInFlight`, the reverse of what it says. `retireDeadIncarnationBeforeStart` @ `resume.go` becomes `clearLifecycleDebris` @ `lifecycledebris.go` (`new`/M2). Add rows for `ErrThreadRolledBack`, `RecoverySessionRefusal`, `coldResumeAuthorized`, `ThreadEvidence.StartOwner`, `ResumeStarting`. Add "re-derive both tables by grep" as an explicit step in every milestone's final task, so this stops being a per-round discovery.
2. **`ThreadParked`'s consumer enumeration (C1).** Record that Task 6b widened the state's producer set and list its seven readers with the verdict for each; add the `switch-agent` repair to M2 (it is the half of Task 6b that makes the widened state usable), and note that M3 Task 8's offered⇒permitted test must cover `switch-agent` and `resume`, not archive alone.
3. **Task 6b's tombstone claim is not delivered (I1).** The Log at `:733-741` and `atlas/couch.md:1478-1482` both state the tombstone survives as the better explanation; record that `resumeEvidence` errors first so the branch never fires in production, and say which way it is being resolved — carry the resolution into `DecideResume`, or delete the branch and the claim together. Add the missing produced-by guard for `ResumeDiagnosticCode` (M1 round 3 identified its absence; nothing has added it).
4. **The referent-staleness enumeration (I2).** Amend the "homes" list the M1 round-5 entry established to include `README.md` and exported doc comments, and record the six sites swept.
5. **M2's operator verification (I3).** Either the four figures/observations, or an explicit deferral naming what is outstanding and who owns it.
