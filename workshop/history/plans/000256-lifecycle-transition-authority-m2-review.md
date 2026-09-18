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

---

## Re-review — 2026-09-17T14:33:47-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2d137941e1ccc9a28b1f83683e41e9ba2295a963..c5f55f31a30ec8f1ee253da9e15694488f5011d8 |
| command | sdlc milestone-close --issue 256 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-17T14:33:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2's substance is strong and I verified it adversarially rather than by reading: every guard the round-1 fixes introduced goes red when reverted (`switchableWhenNothingRuns`' receipt replacement, its session refusal, the `isBindingDiagnostic` carry in `ResumeContextWith`), `clearLifecycleDebris` is the right ARCH-DRY consolidation with screen-before-write intact, archive's read-before-write ordering plus the triple-look comparison is the best part of the diff, and the new partial-write crash test is real ARCH-ORDER evidence rather than a comment. What blocks SHIP is that round 1's C1 was fixed as the **instance, not the class**: `PrepareAgentSwitch` still disagrees with the classification for 2 of the 4 reachable producers of `ThreadParked`/live-without-incarnation, and in one direction it now fails **open** — I measured `PrepareAgentSwitch` accepting a row couch is hosting *right now* (classified `live`), where the pre-fix guard refused it. The class guard added this round cannot catch that, because its producer list is hand-written (two cells) and its "totality" assertion compares that list to itself; the classifier's own shape table already lists three parked producers and I demonstrated a fourth. Secondarily, I2's rule was written down but the `git grep` the rule prescribes was never run — an operator-facing `resume` summary and the atlas's own statement of the `parked` rule still state the retired referent.

## 1. Strengths

- `cmd/internal/couchcore/lifecycledebris.go:52` — the four-round-hardened rule as one function that both resume and archive call, with screening complete before the first write and each write authorized by a probe of the entity it acts on. The `clearStart bool` split (`:66-70`) removes the nonce/flag coupling round 1 flagged and cites why rather than restating a validator clause.
- `cmd/internal/couchcore/detach.go:268-281` and `:330-336` — `RecoverySessionRefusal` asked first with nothing written, and the final recheck compared against the **first** look as well as the reconciler's. I reverted the `first`-look arm in a scratch tree and it reds `TestRecoveryArchiveRefusesSessionAppearingBeforeStop`.
- `cmd/internal/couchcore/archivedebris_test.go:349` — `TestClearingDebrisResumesSafelyAfterACrashBetweenItsWrites` reproduces the interleaving through the real `AfterTarget` seam and *guards its own fixture* (`:389`, "this seam no longer splits the two"). That is the ARCH-ORDER partial-progress claim moved from a comment to an oracle.
- `cmd/internal/couchcore/resume.go:454-465` — carrying the binding diagnostic to `DecideResume` instead of bailing at `resumeEvidence` is the right read of "guidance at the consumer". Reverting it to `if err != nil` reds `TestEveryResumeDiagnosticCodeIsReachableFromProduction` with `resume-binding-unbound` instead of `resume-tombstoned` — verified.
- `cmd/internal/couchcore/recovery.go:80-90` — extracting the session half is behaviour-preserving (I traced every `Presence` value through both shapes) and the extraction is what makes archive's ordering fix possible without a second derivation.
- Core-concepts tables: I grepped all 21 rows plus the integration table. Every row now matches the tree, and all eight new M2 production symbols have one. C2 is genuinely closed.

## 2. Critical findings

**C1 — `PrepareAgentSwitch` is a third authority, and the class it was supposed to close is 2 of 4 producers. One direction now fails OPEN.** `cmd/internal/couchcore/switchagent.go:60-95`

> **This is the 5th finding in family `classification-not-authority`.** Earlier rounds fixed instances. Do NOT fix these instances — state the rule that covers all of them, and fix that.

The fix replaced one receipt check with a *fresh session probe*, which is neither the classification nor the live evidence couch already holds. Three measured disagreements, all through the production gather path in a `git archive` scratch tree (working tree untouched):

| Producer | `ClassifyThread` | `PrepareAgentSwitch` |
|---|---|---|
| park receipt, session absent | `parked` | accepted ✓ |
| ledger only, session absent (M2's new producer) | `parked` | accepted ✓ (the fix) |
| **driverless start claim + resolvable ledger** | `parked` | **refuses permanently**: `switch-agent: occupied thread: park requires exactly one identified live or unknown incarnation` |
| park receipt + `SessionUnresolved` (the asymmetry the classifier keeps *deliberately*) | `parked` | refuses: `its session state could not be checked` — and the **pre-M2 guard admitted this row**, because the receipt was the admission |
| hosted pty child, no incarnation, no session binding | `live` | **ACCEPTED** — and `SwitchAgent` parks the source only `if hasOccupiedIncarnation` (`switchagent.go:267`), so a fresh agent launches beside the one couch is hosting: two agents on one tree, the exact cost the guard's own comment names |

The last row is a regression introduced in this window: I reverted the `else if` to `record.VerifiedPark == nil` and the same probe refused. The third row is reachable exactly as often as the producer M2 shipped for — the issue Log records #273 minting one orphaned claim per attempt — and round 1 named this arm explicitly ("The #272 shape hits the same wall by the other arm"), so the fix skipped a site the finding pointed at.

**The rule.** An action guard must **consume** the classification (state + reason + the live evidence already gathered), the way M3 Task 8 plans `ArchivableState` — not re-derive a parallel predicate in the IO shell. Where a strict re-observation is genuinely needed at the action, the *policy* belongs in a pure predicate beside `RecoverySessionRefusal` (which this same diff got right) and must agree with the classifier's own reading of the same fact: the classifier calls an absent session-name binding `SessionUnresolved` and fails closed; `switchableWhenNothingRuns:71-76` calls it proof of absence and fails open, against `ErrPairSessionBindingAbsent`'s own doc ("It does not prove session absence", `artifactcollision.go:15-17`).

**And the enumeration must be derived, not listed.** `parkedproducers_test.go:38-60` hand-writes two producers, and its totality check (`:111`, `len(classified) != len(producers)`) compares a map filled from that same literal — it cannot fail when a producer is added, which is precisely the gap it claims to close. Its sibling `TestEveryResumeDiagnosticCodeIsProducedBySomeSite:295` already shows the shape: derive the set from the declaration. Derive the producer set from `everyThreadShape` (which already carries three parked rows and a totality test, and is itself missing the driverless-claim-with-ledger row) and cross it with `menuActionItems`, so `offered ⇒ permitted` holds by construction for `switch-agent`, `resume` **and** `archive` together.

## 3. Important findings

**I1 — the referent sweep's own mechanical check was never run.** `cmd/internal/couchcore/ops.go:381`

> **This is the 6th finding in family `stale-wording-after-referent-change`.** Do NOT fix these four sites — the rule is the deliverable.

The Revisions entry states the rule correctly ("the homes are atlas + plan + the function comment + every exported doc comment naming it + `README.md`, with `git grep` of the old referent string as the mechanical check") and then the grep was not executed. `git grep -n 'verified.park'` still returns, among historical uses that are fine, four statements of the retired referent:

| Site | Says | Why it is wrong now |
|---|---|---|
| `cmd/internal/couchcore/ops.go:381` | `resume` — "Reattach a detached work thread, or resume a **verified-parked** one" | The operation catalog the TUI and the advisor both render (`ops.go:92-94`). Identical class to `README.md:470`, which *was* swept. |
| `atlas/couch.md:35` | "`parked` when **verified park exists** with no active park transaction, reservation, or incarnation" | The atlas's own statement of the classification rule, contradicted by M2 (ledger authority) and by M1 (park/incarnation unread) |
| `atlas/couch.md:498` | "Enter … resumes an exact **verified-park** row" | |
| `atlas/couch.md:863-871` | "Resume accepts **verified park** or proved detachment… The occupied-incarnation refusal is unchanged" | `DecideResume` reads neither (`resume.go:118`, "the park transaction and the incarnation are NOT read here") |

The rule needs the step, not the sites: make the grep a *checked* item of the boundary close (the same treatment C2 just gave the Core-concepts tables), or the enumeration will be re-derived and re-missed next milestone. `launch_existing.go:33` is a fifth, lower-confidence site (unexported comment).

**I2 — two fixtures were retuned to preserve their old verdict, and the new behaviour they used to cover has no test.** `cmd/internal/couchcore/startup_test.go:328`, `cmd/internal/couchcmd/run_test.go:406`

Both `…StartsNewWhenNoSessionSurvives` tests were changed from `BindingEstablished` to `BindingUnbound` so they keep asserting "startup creates a NEW thread". That is honest and documented — but the behaviour the issue Log advertises as *"what the operator will notice"* ("startup adopts it rather than starting a second thread in the same tree") now has **no** test at the startup level. I restored the established binding in a scratch tree and confirmed the adoption path is taken: startup selected the cold row and reported *"couch could not resume the thread in this tree … and will not start a second one"* after a 15 s registration wait. So the widening extends `TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot`'s deliberate no-fallback policy to a new class of rows, and neither direction is pinned — the positive (adopt, resume succeeds) nor the negative (adopt, resume fails, couch declines to start in that tree at all).

The rule: **when a fixture is retuned so an existing test keeps its old verdict under new behaviour, the new behaviour gets its own test in the same commit.** The retune is the signal that a branch just changed owner, and the commit that moves it is the only one that knows.

## 4. Minor findings

- `cmd/internal/couchcore/actionableinventory.go:412-416` — the late `evidence.Session.State == SessionUnresolved` guard is **unreachable by construction**: `:395` already returns for `Unresolved && VerifiedPark == nil`, and `:405` returns for every remaining receipt-holder. I replaced its body with a panic and ran the whole `couchcore` suite: zero hits. Delete it, or move the receipt exception so the distinction its comment defends is actually decided there. (4th in `fail-closed-guard-untested` — the rule's dual: a branch subsumed by an earlier predicate is a guard no test can reach, and its comment is a claim nothing checks.)
- `cmd/internal/couchcore/resume_test.go:373` — `TestEveryResumeDiagnosticCodeIsReachableFromProduction` asserts **one** code (`ResumeTombstoned`); its own comment admits it. Derive the set from the declaration as its sibling does, or name it for the one code. (5th in `test-name-contradicts-assertion`.)
- `cmd/internal/couchcore/resume.go:426-437` — `DetachedSessions` now runs on every resume including cold ones, and an observation **error** aborts a cold resume that consumes no session evidence. Fail-closed and loud, but it is a newly widened dependency that the Log records only as a cost.
- `cmd/internal/couchcore/resume_test.go:396` — the new reachability test `t.Skipf`s if the store rejects its fixture. It passes today (verified), but a silent skip is how a reachability guard stops guarding.

## 5. Test coverage notes

Mutations I ran, all red as claimed: restore `record.VerifiedPark == nil` in `PrepareAgentSwitch` → `…ParkedProducer…/ledger_only`; drop `binding.Present` → `…RowWhoseSessionSurvives/session_still_up`; revert the `isBindingDiagnostic` carry → `…ReachableFromProduction`. Suite state: `couchcore`, `couchtty`, `couchcmd`, `artifactpath`, `threadrecord` — every failure is `ptychild: … operation not permitted` (sandbox PTY restriction), no logic failures.

Gaps: (a) nothing crosses the classifier's output with `menuActionItems` **and** the guards behind it — that is what C1 needs, and it must derive both sides (§C1); (b) the `parked` producer set is under-enumerated in `everyThreadShape` too — the driverless-claim-with-ledger row has no cell, so `TestClassifyThreadIsTotalOverEveryRecordShape`'s totality claim does not cover the shape C1 breaks on; (c) `PrepareAgentSwitch` has no row for a `live` record carrying no incarnation, which is the shape M1 deliberately admits and the one that now fails open; (d) ARCH-MOCK — the fake/production error-shape conformance is still per-site (`TestSessionPresenceAnswersThroughTheProductionChecker` covers presence; refusal shapes have no shared contract table). Round 1 raised (d) as a note; it is unchanged and is the 3rd in `production-seam-only-tested-through-fake`, so it belongs in M3's plan rather than another per-site fix.

## 6. Architectural notes

- **ARCH-DRY — pass.** `clearLifecycleDebris` and `RecoverySessionRefusal` both consolidate. The one new duplication is C1's: `switchableWhenNothingRuns` re-derives "is anything running", which the classification already answers.
- **ARCH-PURE — flag (C1).** Inconsistent within one diff: the archive session refusal was extracted as a pure predicate; the switch-agent one was written inline in the IO shell, so its policy is only testable through a fake.
- **ARCH-PURPOSE — flag (C1, I1, I2).** Two shadow-sweeps stopped one level short of their own rule: the `parked` consumer sweep fixed the site the finding named, and the referent sweep wrote the grep without running it.
- **ARCH-MOCK — pass with a gap.** The sentinel wrap is correct and mutation-pinned; production wraps the same sentinel (`artifactcollision.go:223`), and the fake's `SessionAbsent` default is faithful to production's readable-scope-no-row rule (`artifactcollision.go:370-385`). No conformance guard against the next divergence.
- **ARCH-CONSTRAINTS — pass.** Both widened costs are measured into `## Log` (ledger reads 2/6 → 4/6, 0.62 ms per round), and the unbounded cold side is named as a known gap rather than left implicit. C1's fix adds one exact session observation per `switch-agent` — strict at the action, correct.
- **ARCH-SECURE — pass.** The validator-accepted domain is the untrusted-input boundary (`lifecycledebris.go:41`), every probe is an exact `{PID, Identity}`. One soft spot: C1's fail-open reads a missing index row as proof rather than as the absence of evidence.
- **ARCH-ORDER — pass.** Read-before-write in archive, the triple-look comparison, and the crash-between-writes test are all genuine. The remaining unreproducible interleaving is C1's TOCTOU (probe, then launch), which is inherent to the action path.
- **ARCH-FUNERAL — pass.** No new artifact family; `AbandonPark`'s tombstone growth is bounded per park attempt and the decision to stop vetoing on tombstones is the removal-path reasoning, written down.

## 7. Plan revision recommendations

1. **C1 is not closed — record the class and the enumeration (`## Revisions`).** State that `ThreadParked` has **four** reachable producers (receipt+absent, ledger-only+absent, receipt+unresolved via the kept asymmetry, driverless claim + resolvable ledger), not two; correct the Core-concepts bullet and `atlas/couch.md:1478` ("`parked` has two producers now"), both of which currently assert two. Add the missing producer row to `everyThreadShape`. Move `switch-agent`'s admission into M3 Task 8/8a's `offered ⇒ permitted` predicate rather than a second session probe, and record the fail-open on a `live` record with no incarnation as the regression this round introduced.
2. **Make the two derived views checked steps, not stated rules.** C2 wrote "re-derived by grep at each boundary close" for the Core-concepts tables and I1 wrote the same for the referent homes; neither is a checkbox in any task. Add both to each milestone's final task, with the exact grep.
3. **Task 6b's startup consequence.** Record that the widening changes startup adoption for session-gone/ledger-resolving rows, that the two `…StartsNewWhenNoSessionSurvives` fixtures were retuned to keep their old verdict, and that the new behaviour (and its no-fallback failure mode) needs its own test — this is also the closest thing in the suite to M2's deferred operator-verification item 3.
4. **Note the dead branch.** `ClassifyThread`'s late unresolved-session guard is unreachable after the M2 reordering; say which way it is resolved so the comment stops defending a distinction decided elsewhere.

```findings
findings:
  - id: new
    severity: Critical
    family: classification-not-authority
    title: |
      switch-agent re-derives "nothing runs" instead of consuming the classification: 2 of 4 parked producers refuse, and a hosted `live` row now fails OPEN
    detail: |
      This is the 5th finding in family classification-not-authority; fix the rule, not
      these sites. Measured through the production gather path in a git archive scratch
      tree: a driverless start claim with a resolvable ledger classifies `parked` and
      PrepareAgentSwitch refuses permanently ("occupied thread: park requires exactly one
      identified live or unknown incarnation"); the receipt+SessionUnresolved producer the
      classifier keeps deliberately is refused where the pre-M2 guard admitted it; and a
      record couch HOSTS with no incarnation and no session binding classifies `live` and is
      now ACCEPTED, while SwitchAgent parks the source only if hasOccupiedIncarnation
      (switchagent.go:267) -- two agents on one tree. Reverting the else-if to
      record.VerifiedPark == nil refuses that row, so it is a regression in this window.
      The rule: an action guard consumes the classification (state, reason, live evidence)
      as M3 Task 8 plans for archive; where a strict re-observation is needed the policy is
      a pure predicate beside RecoverySessionRefusal and must read the same world fact the
      same way (switchagent.go:71 treats ErrPairSessionBindingAbsent as proof of absence
      against its own doc at artifactcollision.go:15). And the offered-implies-permitted
      enumeration must be DERIVED from ClassifyThread/everyThreadShape, not hand-listed:
      parkedproducers_test.go:111 compares a map filled from its own two-cell literal, so
      its totality assertion cannot fail, and everyThreadShape is itself missing the
      driverless-claim-with-ledger row.
  - id: new
    severity: Important
    family: stale-wording-after-referent-change
    title: |
      I2's rule was written down but the git grep it prescribes was never run -- four sites still state the retired `parked` referent
    detail: |
      This is the 6th finding in family stale-wording-after-referent-change; the rule is the
      deliverable, not the sites. The Revisions entry names the homes (atlas + plan +
      function comment + every exported doc comment + README.md) and "git grep of the old
      referent string as the mechanical check"; the check was not executed. Remaining:
      cmd/internal/couchcore/ops.go:381, the operator/advisor-facing `resume` summary
      ("resume a verified-parked one"), the same class of surface as README.md:470 which was
      swept; atlas/couch.md:35, the atlas's own statement of the classification rule
      ("parked when verified park exists ... or incarnation"); atlas/couch.md:498; and
      atlas/couch.md:863-871 ("Resume accepts verified park or proved detachment ... The
      occupied-incarnation refusal is unchanged"), which DecideResume contradicts at
      resume.go:118. Make the grep a checked step of the boundary close, the way C2 just did
      for the Core-concepts tables.
  - id: new
    severity: Important
    family: fixture-retuned-to-preserve-old-verdict
    title: |
      Two fixtures were retuned to keep their old verdict and the new startup behaviour they used to cover has no test
    detail: |
      startup_test.go:328 and couchcmd/run_test.go:406 both changed BindingEstablished ->
      BindingUnbound so "startup creates a NEW thread" still passes. The behaviour the issue
      Log advertises as what the operator will notice -- startup adopts a session-gone row
      whose ledger resolves rather than starting a second thread in the same tree -- has no
      test at the startup level in either direction. Restoring the established binding in a
      scratch tree shows the adoption path IS taken: startup selected the cold row and
      reported "couch could not resume the thread in this tree ... and will not start a
      second one" after a 15 s registration wait, extending
      TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot's deliberate no-fallback
      policy to a new class of rows. The rule: when a fixture is retuned so an existing test
      keeps its old verdict under new behaviour, the new behaviour gets its own test in the
      same commit -- the retune is the signal that a branch changed owner.
  - id: new
    severity: Minor
    family: fail-closed-guard-untested
    title: |
      ClassifyThread's late unresolved-session guard is unreachable after the M2 reordering
    detail: |
      actionableinventory.go:412 is subsumed: :395 returns for Unresolved with no receipt and
      :405 returns for every remaining receipt-holder, so the branch is dead by construction.
      Replacing its body with a panic and running the whole couchcore suite produced zero
      hits. 4th in this family, as the dual of "a guard nothing pins is not a guard": a
      branch subsumed by an earlier predicate is a guard no test can reach, and the comment
      defending its load-bearing distinction is a claim nothing checks. Delete it, or move
      the receipt exception so the distinction is decided there.
  - id: new
    severity: Minor
    family: test-name-contradicts-assertion
    title: |
      TestEveryResumeDiagnosticCodeIsReachableFromProduction asserts exactly one code
    detail: |
      resume_test.go:373 pins ResumeTombstoned only; its own comment admits it. Its sibling
      TestEveryResumeDiagnosticCodeIsProducedBySomeSite:295 derives the identifier set from
      the declaration so it cannot be satisfied by forgetting a row -- do the same here, or
      name the test for the one code. It also t.Skipf's if the store rejects its fixture
      (:396); it passes today, but a silent skip is how a reachability guard stops guarding.
```

---

## Re-review — 2026-09-17T15:05:41-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2d137941e1ccc9a28b1f83683e41e9ba2295a963..8db22bef23530045f2f3435ba93a03f7dcc3c3e5 |
| command | sdlc milestone-close --issue 256 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-17T15:05:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 2's fix is the right *shape* — `SwitchableState` as a pure predicate both the switcher and the guard call, `classifyForAction` running the production evidence pass, and a derived `AllThreadStates() × AllThreadReasons()` table — and I mutation-verified it goes red under the original defect. But BR-33 is **not closed**: the class moved one layer down rather than closing. `PrepareAgentSwitch` now *admits* the driverless-claim-with-ledger producer, and `SwitchAgent` then **fails on it**, because the commit's own execution path still branches on `hasOccupiedIncarnation(thread)` — the bookkeeping the classifier deliberately stopped reading — instead of the state it just consumed. I reproduced it end-to-end through a fully-wired `PairLifecycle` in a pinned scratch tree at HEAD: `outcome="park-incomplete"`, `err=switch-agent: park did not complete; use park retry/recover/abandon: park requires exactly one identified live or unknown incarnation` — the same sentence BR-33 quoted, relocated from the preflight to the commit, and now with guidance ("park retry/recover/abandon") for a park that never existed. The class guard cannot see it because it stops at `PrepareAgentSwitch`. Secondarily, the two derived views the milestone committed to re-deriving mechanically were not: the plan's Integration table still lists `switchableWhenNothingRuns`, a symbol this very commit deleted, and the atlas still says `parked` has "two producers" and names the deleted guard as current.

> Note: my measurements are against the pinned commit `8db22bef`. The working tree had uncommitted edits during the review (including, late in the session, a `state ActionableThreadState` field on `PreparedAgentSwitch` citing "#256 M2, round 3"); none of that is in the reviewed range.

## 1. Strengths

- `cmd/internal/couchcore/actionableinventory.go:319-352` — `SwitchableState` as a pure `(state, reason)` predicate is the correct read of "consume, don't re-derive", and its doc comment records *both* failed attempts rather than presenting the third as obvious. Mutating `case ThreadLive, ThreadParked` → `case ThreadLive` reds `TestSwitchAgentOfferedImpliesPermitted` at `parked/` — verified.
- `cmd/internal/couchtty/switch_agreement_test.go:23` — the domain is genuinely derived from `AllThreadStates() × AllThreadReasons()`, and the `(state == Unusable) != (reason != "")` filter keeps it to rows the projector can actually produce. This is the shape round 2 asked for, and it lives in `couchtty` so the import direction is right.
- `cmd/internal/couchcore/parkedproducers_test.go:156-166` — the totality check now takes its domain from `everyThreadShape` instead of the literal it is checking, and the fix *found* two producers while doing so (receipt+unresolved, driverless+ledger). Adding the driverless row to `everyThreadShape` (`classify_test.go:287`) is the honest completion of that.
- `cmd/internal/couchcore/startup_test.go:344` — the new adoption test is a real regression test for the M2 behaviour: reverting the ledger gate to `record.VerifiedPark != nil` reds it, along with 2 of 4 producer rows. Verified by mutation.
- `cmd/internal/couchcore/actionableinventory.go:462-467` — deleting the subsumed late unresolved-session branch, and replacing it with a comment that says *where* the distinction is decided rather than defending a dead one, is exactly the right disposal of BR-36.

## 2. Critical findings

**C1 — `SwitchAgent` re-derives the park decision from bookkeeping, so switch-agent still always fails on BR-33's own producer.** `cmd/internal/couchcore/switchagent.go:285`

> **This is the 6th finding in family `classification-not-authority`.** Do NOT fix this site. The rule below is the deliverable.

`PrepareAgentSwitch` consumes the classification and admits the driverless-claim-with-ledger row (`parked`). `SwitchAgent` then discards it: `if hasOccupiedIncarnation(thread)` is true for an `IncarnationCreating`, so it calls `ParkExpected`, and `soleParkableIncarnation` accepts only `Live`/`Unknown` with a PID. Measured at HEAD in a `git archive` scratch tree with a production-shaped `PairLifecycleController`:

```
SwitchAgent outcome="park-incomplete"
err=switch-agent: park did not complete; use park retry/recover/abandon:
    park requires exactly one identified live or unknown incarnation
after: park=<nil> verifiedPark=false incarnations=[{State:creating Start:…}]   # no durable damage
retry PrepareAgentSwitch err=<nil>                                             # admitted again
```

That is BR-33's verbatim symptom — an offered action that always fails — with the refusal moved later and the guidance now actively wrong (there is no park to retry, recover or abandon). The `clearLifecycleDebris` call the commit message credits for this producer sits *after* the park block (`:303`), so it never runs. The same defect admits a second shape: an `unusable/session-gone` row carrying a stale `live` incarnation is now let through `PrepareAgentSwitch` entirely (the `state == ThreadLive` gate skips the "source is not a verified live actor" check that refused it before round 2) and fails at `ParkExpected` with `exact Pair session binding is absent` — measured; API-reachable only, since `menuActionItems` gives that row `{archive,name,describe}`.

**The rule.** *The classification an action was admitted on is the value its execution branches on.* Carry it (e.g. on `PreparedAgentSwitch`) and decide park-vs-clear from the state — `live` ⇒ park the source, anything `SwitchableState` admits ⇒ clear debris and proceed — never from `hasOccupiedIncarnation`/`record.Incarnations`, which is precisely the bookkeeping M1 removed from the classifier. I validated the direction: gating the park block on `soleParkableIncarnation(thread) == nil` makes the driverless producer reach `outcome="started"`, with no other couchcore test disturbed.

**And the enumeration must cross producers × ACTIONS EXECUTED, not producers × admission guards.** `parkedproducers_test.go:130` calls `PrepareAgentSwitch`; `:136` calls the pure `DecideResume`. Only archive (`:143`) drives the real action. A table that stops at the preflight cannot catch a preflight/commit disagreement — which is the whole of this finding, and the reason round 2's guard went green over a broken action. The fake stack *can* drive `SwitchAgent` end to end (my scratch test did, with `pairlifecycletest` + `fakeControllerLifecycle`), so there is no seam excuse.

**C2 — the Core-concepts tables contradict the tree, in the same commit that promised they are re-derived.** `workshop/plans/000256-lifecycle-transition-authority-plan.md:252`, `:182`, `:160`

> **This is the 5th finding in family `plan-code-divergence`.** Do NOT fix these rows. The rule is that the re-derivation must be *executed*, not asserted.

| Row / claim | Tree at HEAD |
|---|---|
| `:252` `switchableWhenNothingRuns` \| `switchagent.go` \| new \| M2 | **does not exist** — deleted by this same commit |
| `:182` `AllThreadStates` \| new \| **M3** | landed in **M2** (`actionableinventory.go:299`); Task 8 Step 3 (`:847`) is still `- [ ]` |
| `SwitchableState`, `classifyForAction` | new M2 production symbols with **no row in either table** |
| `:160` "Last re-derived: the M2 boundary, 2026-09-17." | false as of this commit |

Round 2's own review recommendation #2 asked for this to become *a checked step, not a stated rule*, and it did not. The plan already states the exact mechanical check at `:156`; the gap is that nothing runs it. **The rule:** a derived view is either machine-checked or it is prose. This repo already owns the machinery — `couchcore/plan_contract_test.go` pins #151's plan tables against pinned source — so the fix is to extend that pattern to #256's two tables (symbol exists at the stated path; every new production symbol in `git diff <prev boundary>..HEAD -- '*.go'` has a row), not to hand-edit these four cells and re-assert the date.

## 3. Important findings

**I1 — the atlas states the retired guard and the retired producer count as current.** `atlas/couch.md:1488`, `:1498`

> **This is the 4th finding in family `atlas-contradicts-code`.** Same rule as C2, different artifact.

`:1488` "**`parked` has two producers now, and its consumers are the enumeration.**" — the code and `everyThreadShape` now enumerate **four**, and round 2's plan-revision #1 named this exact line ("correct … `atlas/couch.md:1478` … which currently asserts two"). `:1498` "The guard now asks the session instead (`switchableWhenNothingRuns`)" — that function was deleted in this commit; the paragraph five lines below then explains why asking the session was wrong, so the atlas contradicts itself in adjacent paragraphs. The narrative rewrite landed; the corrections the finding actually asked for did not. Same deliverable as C2: the atlas paragraph describing a mechanism cites a symbol, and a boundary check greps that the symbol exists.

## 4. Minor findings

- `cmd/internal/couchcore/switchagent.go:150` — the new `!hasOccupiedIncarnation` refusal is **not pinned**. Deleting it leaves `TestSwitchAgentRefusesAThreadCouchHostsWithNoIncarnation` green (verified), because `soleParkableIncarnation` refuses the same record two lines later. The outcome is covered; the guard and its operator-facing message are not. (5th in `fail-closed-guard-untested` — the rule: *a guard's test must discriminate that guard's own exit*, by code or message, not merely assert `err != nil`.)
- `cmd/internal/couchcore/startup_test.go:344` — `TestStartInteractiveAdoptsAThreadWhoseConversationStillResolves` never calls `StartInteractive`; it stops at `ActionableThreadInventoryContext` + `SelectResumableRoot`. Filtering `ThreadParked` out of `StartInteractive`'s own `SelectResumableRoot` call reds four sibling tests and leaves this one **green** (verified) — the exact shortcut `startup_test.go:276-283` records a previous review catching. (5th in `test-name-contradicts-assertion`.)
- `cmd/internal/couchcore/switchagent.go:80` — `classifyForAction` runs a *whole* evidence round per `PrepareAgentSwitch`, including the form **prefill**: one host-wide `SessionPresence`, plus `Physical()` and a `CurrentStartTransaction` process probe for **every** record in the store (`ask` narrows only the ledger read). The issue Log's ARCH-CONSTRAINTS table measures the *refresh*, and records the action cost as "one exact session observation". (2nd in `declared-measurement-not-recorded` — the rule: when a change moves work onto an action path, the measurement is taken on that path, not inherited from the refresh's.)
- `cmd/internal/couchcore/launch_existing.go:33` still reads "an exact verified-park resume" — BR-34 named it as its fifth, lower-confidence site and it was not swept while a different fifth was. Unexported comment; noted only so the referent list is complete.

## 5. Test coverage notes

Mutations run at HEAD in `$TMPDIR/br3` (pinned `git archive`, working tree untouched), all as claimed unless noted:

| Mutation | Result |
|---|---|
| `SwitchableState`: drop `ThreadParked` | reds `TestSwitchAgentOfferedImpliesPermitted` at `parked/` ✓ |
| gather gate → `record.VerifiedPark != nil` | reds `…AdoptsAThreadWhoseConversationStillResolves` + 2 producer rows ✓ |
| delete `!hasOccupiedIncarnation` in `PrepareAgentSwitch` | **stays green** ✗ (Minor above) |
| drop `ThreadParked` from `StartInteractive`'s selector input | 4 siblings red, the new adoption test **stays green** ✗ (Minor above) |
| gate the park block on `soleParkableIncarnation` | driverless producer reaches `started`; suite otherwise undisturbed (C1 fix direction) |

Suite state at HEAD: `couchcore`, `couchtty`, `couchcmd` fail only on `ptychild: … operation not permitted` / `open pty` (sandbox PTY restriction) and on `plan_contract_test`'s `exit status 128` (git unavailable in a scratch archive); `artifactpath` ok. No logic failures. Gaps: (a) no end-to-end `SwitchAgent` coverage for any producer except the receipt shape — that is C1's hole; (b) `PrepareAgentSwitch` has no row for the `unusable/{binding-lost,session-gone}` states `SwitchableState` newly permits; (c) ARCH-MOCK fake/production refusal-shape conformance is still per-site — unchanged since round 1, 3rd in `production-seam-only-tested-through-fake`, and belongs in M3's plan rather than another per-site fix.

## 6. Architectural notes

- **ARCH-DRY — flag (C1).** `SwitchableState` is the right consolidation; `clearLifecycleDebris` is shared by three callers. The residual duplication is that `SwitchAgent` re-derives "is something running" from `hasOccupiedIncarnation` when the value is already computed metres away.
- **ARCH-PURE — pass with a flag (C1).** `SwitchableState` is pure and `classifyForAction` is a thin seam. The impurity is structural: `PreparedAgentSwitch` drops the state, forcing the commit path to reconstruct it from a record field.
- **ARCH-PURPOSE — flag (C1, C2, I1).** Three rounds, three shadow-sweeps that stopped one level short: consumers-of-the-field → consumers-of-the-state → *admission* consumers, never execution. And the derived views the milestone promised to re-derive were re-narrated, not re-derived.
- **ARCH-MOCK — pass.** The sentinel wrap (`artifactcollision_fake.go:252`) is correct and mutation-pinned, and the fakes are rich enough to drive the whole switch action — which is why C1 is a test-scope gap, not a seam gap.
- **ARCH-CONSTRAINTS — flag (Minor).** A full evidence round now sits behind an operator keypress (the switch-agent form prefill), unmeasured on that path.
- **ARCH-SECURE — pass.** Every probe is an exact `{PID, Identity}`; `clearLifecycleDebris` takes the validator-accepted domain as its untrusted boundary and says so.
- **ARCH-ORDER — flag (C1).** `SwitchAgent` parks (irreversible, for a live source) *before* screening the debris that can make the park impossible — the same "irreversible step before a revocable check" this milestone already learned twice, now in the switch path. In both shapes I measured the store was left untouched, so the fault is unscreened ordering rather than observed damage; the driverless case is a stable accept→fail→accept loop with no state machine recording that the last attempt failed.
- **ARCH-FUNERAL — pass.** No new artifact family; no writer's per-event growth increased.

## 7. Plan revision recommendations

1. **`## Revisions` — C1: BR-33 is closed at admission and open at execution.** Record that `PrepareAgentSwitch` admits four `parked` producers but `SwitchAgent` parks on `hasOccupiedIncarnation`, so the driverless-claim producer fails `park-incomplete` with park-recovery guidance for a park that does not exist (reproduced, no durable damage, repeatable). State the rule — *the state an action was admitted on is the state its execution branches on* — and that the producer×action table must drive each action to completion, not to its preflight.
2. **`## Revisions` — C2: the derived views were not re-derived.** Correct `:252` (delete `switchableWhenNothingRuns`, add `SwitchableState` and `classifyForAction`), `:182` (`AllThreadStates` landed M2, pulled forward from Task 8 Step 3 — tick `:847` or note the pull-forward), and replace the bare "Last re-derived" line with a *check*, modelled on `couchcore/plan_contract_test.go`'s existing #151 machinery.
3. **`## Revisions` — I1: the atlas.** `:1488` "two producers" → four, enumerated; `:1498` delete the `switchableWhenNothingRuns` sentence so the paragraph pair stops contradicting itself.
4. **Record `SwitchableState`'s widening as a decision, not a side effect.** It permits `unusable/binding-lost` and `unusable/session-gone`, which the menu does not offer — the doc calls that a deliberate superset. Note that the superset removed the `verified live actor` refusal for records carrying an occupied incarnation in those states, and say which guard owns it now.
5. **Move the Minor test-coverage rules into M3's task list as checked steps**, since both are the same shape as Task 8's plan: a guard's test discriminates the guard's own exit, and a test named for a production entry point invokes it.

---

## Re-review — 2026-09-17T15:35:59-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2d137941e1ccc9a28b1f83683e41e9ba2295a963..1857eaefb7b73d19f1ab5ea95d3b53fd76a6e3f5 |
| command | sdlc milestone-close --issue 256 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-17T15:35:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3's two headline fixes are real and pinned: I reverted each in a scratch tree at HEAD and watched the regression go red (reverting `SwitchAgent`'s park predicate to `hasOccupiedIncarnation` reds `TestSwitchAgentCommitAcceptsWhatItsPreviewAccepted`, `…OnTheUnusableStatesItPermits/session-gone…` and the `driverless_start_claim_with_a_ledger/switch-agent` cell; deleting the hosted-no-incarnation guard reds its message-discriminating test; dropping `parked` from `SelectResumableRoot` reds the new `TestStartInteractiveAdopts…`; reverting `startInFlight` to `startClaimed` reds all four worlds of the orphaned-claim shape). BR-33, BR-35, BR-36 and BR-37 are all genuinely closed. What blocks a clean SHIP is BR-34, which is **not addressed**: round 3 swept the `verified park` string but never grepped the *second* retired referent, so six sites still state rules the code deleted — including two production doc comments (`resume.go:239,245`, `relaunch.go:111`) that describe a `DecideResume` occupancy refusal M1 removed, the `ThreadParked` declaration itself claiming **two** producers where the code, the test and the atlas all say four, and the atlas paragraph this window *edited* (`atlas/couch.md:33-38`) still gating `parked` on "no active park transaction … or occupied incarnation" when `everyThreadShape` has two `parked` rows that carry exactly those. That is prose-only — no runtime effect — so it should not cost a fourth round, but the rule has now failed to land seven times and the finding is the rule, not the sites.

## 1. Strengths

- **The enumeration finally crosses the right axis.** `parkedproducers_test.go` runs 4 producers × 3 actions, each driven to *completion* in a fresh env (`:105-145`), and its totality domain is derived from `everyThreadShape` rather than its own literal (`:186-196`). Reverting the classifier to receipt-authority reds 8 tests across the package — the class is now covered from several independent directions.
- **`SwitchableState` (`actionableinventory.go:327-348`) is the right shape**: a pure predicate over `(state, reason)` that both the offer (`couchtty/menu.go`) and the guard call, with `TestSwitchAgentOfferedImpliesPermitted` deriving its domain from `AllThreadStates() × AllThreadReasons()`. Offered-implies-permitted now holds by construction.
- **Archive's revocable-before-irreversible ordering** (`detach.go:253-290`) is correctly built: `RecoverySessionRefusal` is asked with nothing written, `clearLifecycleDebris` screens every precondition before its first write, and the final recheck compares `latest` against *both* earlier looks (`:327-336`) so a session that appears and settles between them cannot look stable.
- **ARCH-SECURE on the new probe is airtight by construction**: `threadrecord/record.go:177` makes `OwnerPID > 0 && OwnerIdentity != ""` a validation requirement, so a persisted claim can never be probed as PID ≤ 0, and `couch.go:944` turns an identity mismatch into `Dead` — pinned by the "pid recycled by another process" row at `startclaim_test.go:52`.
- **The fake now wraps the production sentinel** (`artifactcollision_fake.go:252-256`) and that is pinned, not just asserted: unwrapping it reds `TestSpawnedButNeverBoundThreadIsArchivable`. Good ARCH-MOCK repair.

## 2. Critical findings

None.

## 3. Important findings

- **BR-34 remains open** — see the disposition below. Six sites, two retired referents, one of them never grepped.

## 4. Minor findings

- `warm_failure_test.go:176` — the assertion was widened to `parked || (unusable && session-gone)`; only one route (`3-registration-timed-out`) reaches that branch and it deterministically yields `parked`, so the `session-gone` arm is the **pre-M2 verdict** kept alive as dead slack. Measured: reverting `ClassifyThread` to receipt-authority leaves this test green.
- `switchagent.go:34-41` — `PreparedAgentSwitch.state`, the round-3 C1 fix, has no row in the plan's Core-concepts tables; `TestIssue256PlanTablesMatchTheTree` only checks rows→tree, never tree→rows, which is the half of round 3's C2 finding ("four production symbols with no row") that stayed unchecked.
- `actionableinventory.go:624` — `if observed, presenceErr := …` shadows the outer `observed` live-proof map inside the presence block. Harmless today, confusing in a file where both names mean "what we saw".
- ARCH-CONSTRAINTS: the cold-side ledger read (`actionableinventory.go:707`) is now per-record-without-a-live-session on **every** refresh. `TestWarmRowsAskNoLedgerQuestion` bounds the warm side; nothing bounds the cold side, and the 0.62 ms figure in the `## Log` is 6 records against fakes. Honestly declared as a known gap — noted so M3 does not lose it.

## 5. Test coverage notes

Full-suite run at HEAD (`go test ./cmd/internal/{couchcore,couchtty,couchcmd,artifactpath}/...` with the retention-owner env scrub): every failure is a sandbox `operation not permitted` on a pty child — 23 occurrences, all in `ptychild`/`pty.Open`/`mkdir /tmp/pcnotify-*`. No logic failure. `go vet` clean on both changed packages.

Four independent mutation checks confirmed the new guards are pinned (listed in the summary). The one uncovered direction is the `couchcmd` half of BR-35: `run_test.go:406` was retuned to `BindingUnbound` and no `TestInteractiveLaunch…Adopts…` sibling was added, but the decision it would exercise lives in `StartInteractive`, which the new `couchcore` test drives end-to-end — so this is not a gap worth a round.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** pass — `clearLifecycleDebris` gives the four-site sweep one home that resume, archive *and* switch-agent all call; `RecoverySessionRefusal` is extracted rather than restated at the call site.
- **ARCH-PURE** pass — `ClassifyThread`, `startInFlight`, `SwitchableState`, `RecoverySessionRefusal`, `coldResumeAuthorized` are all pure over `(record, evidence)`; `classify_test.go` and `lifecyclesequence_test.go` run them with no IO at all.
- **ARCH-PURPOSE** flag — the executable half of the purpose is delivered and swept as a class; the documentation half of BR-34's own stated rule is the instance-fix again (one referent grepped, the other not).
- **ARCH-MOCK** pass; **ARCH-ORDER** pass (the four cannot-block events are enumerated and tested as a property, and `StartOwner`'s zero value makes Unknown fail closed by construction — `startclaim_test.go:101` asserts that zero value directly, which is the right way to pin a by-construction claim).
- **ARCH-FUNERAL** pass — no new durable family; `clearLifecycleDebris` *is* a removal path, and `ErrThreadRolledBack` names the case where the record itself goes.
- For **M3 Task 8**: `SwitchableState` + `classifyForAction` is the template to copy for `ArchivableState`. One thing to carry over deliberately — `PrepareAgentSwitch` needed a second, record-level guard (`!hasOccupiedIncarnation`) *after* consuming the classification, because `live` does not imply "there is bookkeeping to act on". Archive will have the same gap in the other direction.

## 7. Plan revision recommendations

- Add a `## Revisions` entry recording that BR-34's grep is a **two-referent** check, and add `PreparedAgentSwitch.state` to the Integration-points table (or state why an unexported field is out of scope, given `ThreadEvidence.StartOwner` is in).
- Extend `TestIssue256PlanTablesMatchTheTree` with the tree→rows direction, or record in the plan that the check is deliberately one-way and the other half stays manual.

```findings
dispose:
  - id: BR-33
    disposition: addressed
    note: |
      Mutation-verified: reverting the commit's park predicate to hasOccupiedIncarnation reds 3 tests; deleting the hosted-no-incarnation guard reds its message-discriminating test; everyThreadShape now carries the driverless-claim row and the producers table derives totality from it.
  - id: BR-34
    disposition: not-addressed
    note: |
      One referent was grepped, the second never was; six sites remain, two of them production doc comments.
  - id: BR-35
    disposition: addressed
    note: |
      TestStartInteractiveAdoptsAThreadWhoseConversationStillResolves drives StartInteractive and reds when ThreadParked leaves SelectResumableRoot's rank (mutation-verified).
  - id: BR-36
    disposition: addressed
    note: |
      The subsumed late unresolved-session branch is deleted; the receipt exception now decides the distinction where it is made.
  - id: BR-37
    disposition: addressed
    note: |
      Renamed TestResumeTombstonedIsReachableFromProduction, t.Skipf is now t.Fatalf, and it drives ResumeContext rather than DecideResume.
findings:
  - id: new
    severity: Important
    family: stale-wording-after-referent-change
    title: |
      BR-34 not addressed: only one of the two retired referents was grepped, and the ThreadParked declaration itself now states the wrong producer count
    detail: |
      This is the 7th finding in family stale-wording-after-referent-change. Do NOT fix
      these six sites one at a time. Round 3 swept the string "verified park" and
      reported the rule as landed, but the sweep covered ONE referent. A SECOND
      referent retired in M1 -- "DecideResume refuses any occupied incarnation" -- was
      never grepped, and a THIRD claim (the producer COUNT of ThreadParked) was fixed
      in the atlas by round 3 and left wrong at the declaration.
      Measured, at HEAD: actionableinventory.go:26-28, the doc comment ON ThreadParked,
      says "Two records therefore produce this state" while the code, everyThreadShape,
      parkedproducers_test.go and atlas/couch.md:1462 all say FOUR. resume.go:239 says
      CheckResumePreconditions exists because "it cannot ask DecideResume, which refuses
      any occupied incarnation", and resume.go:245 says "what stays with DecideResume is
      ... the occupancy refusal"; relaunch.go:111 repeats it verbatim -- M1 deleted that
      refusal, and DecideResume now ADMITS a live relaunch target, so the stated
      rationale for the split is false. atlas/couch.md:881 is BR-34's own fourth named
      site, edited around and left intact. atlas/couch.md:33-38 is BR-34's second named
      site: round 3 replaced "verified park exists" with "its LEDGER resolves" but kept
      "with no active park transaction, reservation or occupied incarnation" (false --
      the "park timed out" and "driverless start claim with a ledger" shapes in
      everyThreadShape classify parked while carrying exactly those) and never touched
      the `live` half, which still states the pre-M1 rule "one durable live PID/start
      identity exactly matches one observed TTY owner" (false -- TestSwitchAgentRefuses
      AThreadCouchHostsWithNoIncarnation builds a live row with no incarnation at all).
      atlas/couch.md:597 names "legacy-unverified records" and :1631 defines a parked
      thread as one with "an exact verified resume handle and no occupied incarnation";
      ResumeLegacyUnverified was deleted in this very window.
      The rule, stated at the level that covers all of them: NO PROSE RESTATES THE
      CLASSIFICATION OR GUARD BRANCH TABLE. Every such passage -- atlas, terminology
      entry, and exported/unexported doc comment alike -- points at ClassifyThread,
      everyThreadShape or the named guard instead of paraphrasing it, which is the
      decision M1 round 5 already made FOR THE PLAN and never applied anywhere else.
      Where a count or a rule must appear in prose, it carries the test that derives it,
      the way TestIssue256PlanTablesMatchTheTree now does for the Core-concepts tables.
      And the boundary close's grep step takes a LIST of retired referents, checked in,
      not the one string the last finding happened to name.
  - id: new
    severity: Minor
    family: fixture-retuned-to-preserve-old-verdict
    title: |
      warm_failure_test's row assertion was widened to keep admitting the pre-M2 verdict, so its only reachable route cannot detect a revert
    detail: |
      This is the 2nd finding in family fixture-retuned-to-preserve-old-verdict, so the
      rule is the deliverable: an assertion must pin the verdict its premise DETERMINES,
      never a disjunction that still admits the verdict the change replaced.
      Measured at warm_failure_test.go:176. Exactly one route reaches the else-branch
      (3-registration-timed-out; the other five set sessionSurvives), and instrumenting
      it shows it deterministically yields state="parked" reason="". The added
      `|| (unusable && session-gone)` arm is therefore unreachable -- and it is exactly
      the pre-M2 answer. Confirmed by mutation: restoring `record.VerifiedPark != nil &&`
      in front of the parkedResumeProofMatches branch of ClassifyThread -- the receipt-
      as-authority defect this milestone exists to remove -- leaves this test GREEN.
      (Eight other tests do red, so nothing ships uncovered; the finding is the
      assertion, not the coverage.) The comment above it claims it was "re-derived from
      the premise rather than loosened until it passed", which is the claim the
      disjunction contradicts.
  - id: new
    severity: Minor
    family: plan-code-divergence
    title: |
      The derived-view check runs rows-to-tree only, so the "production symbols with no row" half of round 3's C2 is still unchecked -- and this window added one
    detail: |
      This is the 5th finding in family plan-code-divergence. TestIssue256PlanTablesMatch
      TheTree (plan_contract_256_test.go:113-165) asserts every landed row's symbol is
      declared (or, for `deleted`, is not) at its stated path. It never walks the other
      way, which is the direction round 3's C2 finding named as "four production symbols
      with no row at all". The plan's own prose still carries that half as a manual step
      ("git diff --stat <prev boundary>..HEAD -- '*.go' for files whose new symbols have
      no row"), and it was not run: PreparedAgentSwitch.state (switchagent.go:34-41) --
      the field that IS round 3's C1 fix -- has no row, while ThreadEvidence.StartOwner,
      an equally structural field, does. The rule: a derived view is machine-checked in
      BOTH directions, or the unchecked direction is written down as deliberately manual
      with the reason, rather than left as prose the check appears to cover.
  - id: new
    severity: Minor
    family: envelope-declared-not-enforced
    title: |
      The cold-side ledger read now scales with store size on every refresh and no test bounds it
    detail: |
      actionableinventory.go:707 asks ResolveEstablished for every resume-shaped record
      whose session is not present, on every refresh -- replacing a gate that fired only
      for park-receipt holders. TestWarmRowsAskNoLedgerQuestion bounds the warm side (a
      hosted row and a detached row pay nothing) and nothing bounds the cold side; the
      recorded figure (0.62 ms, 6 records, fakes) does not establish growth. The issue
      Log declares this honestly as a known gap, which is why this is Minor and not
      Important -- the finding is that a declared envelope needs an enforcing assertion
      the way SessionPresenceQueries()==1 and DetachedQueries()==0 got one in M1, not a
      prose note that a later reader has to find. Same shape one layer out:
      observeRecovery (recovery_execute.go:54-64) now probes the session for record
      shapes that previously short-circuited, and reconcileRecoveryHelper calls it in an
      8-attempt loop where a present session costs a ~250 ms list-clients per pass.
```
