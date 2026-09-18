# Boundary Review — pair#256 (milestone M3)

| field | value |
|-------|-------|
| issue | 256 — Enforce lifecycle transition authority and outcome uncertainty |
| repo | pair |
| issue file | workshop/issues/000256-lifecycle-transition-authority.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | c4cbd1fe2fdc9f1f9234979ce26757f3323f94bc..138d04757ddb512f9cf50491379f8719e1a11e7a |
| command | sdlc milestone-close --issue 256 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-17T17:39:46-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 does what it says at its core: archive's admission moved from the record to the classification (`Couch.ArchiveThread` → `classifyForAction` → `ArchivableState`), `Unknown` now survives the process projection as `ThreadEvidence.Unproven` and fails closed ahead of every durable refusal, the arbitrary-mutation door is shut behind an unexported `updateExistingThread` plus a receiver-scoped AST guard, and the archive confirmation names the agent on a measurement rather than an intention. I mutation-verified the five behaviour-changing pieces — archive admission, the `Unproven` branch, `menuArchiveOffered`, the `RetireUnprovenIncarnation` routing, the confirmation wording — and each goes red without its fix. A base-vs-head full-suite comparison (`go test ./...` at `c4cbd1fe` and `138d0475`, both in pinned scratch trees) shows **zero new failures**; every failure in this environment is the pre-existing pty/exec restriction, identical on both sides. What blocks SHIP is not correctness of the shipped paths but the *edges* the milestone added around them: three new store transitions carry fail-closed preconditions that no test exercises (deleting all three leaves `couchcore` green), the new classifier branch has no shape in the corpus that is declared to be "the cross product, not a sample", four sites still state the occupancy claim M3 retired, and the ARCH-CONSTRAINTS bound recorded in `## Log` is measured on the one archive shape that cannot pay it — I measured the shape that can at 3 `list-clients`, not 0.

## 1. Strengths

- **`cmd/internal/couchtty/action_agreement_test.go:72-78` — the non-vacuity check.** `everOffered` turns a table that is trivially satisfied by offering nothing into one that fails when an item silently leaves the menu. This is the right answer to the M2 round-4 finding, and keeping `menuArchiveOffered` hand-written rather than delegating to `ArchivableState` (`menu.go:1275-1290`) preserves the guard's ability to fail.
- **`cmd/internal/couchcore/fake_refusal_conformance_test.go`** derives the seam/fake pairing from the `X.go`/`X_fake.go` convention and fails when the convention changes (`pairs < 3`), rather than listing the one sentinel that bit. That is the class, not the instance, and it is the strongest new guard in the window.
- **`cmd/internal/couchcore/mutation_door_test.go:52-62,86-100` — receiver-scoped, not file-scoped.** The two rules (no exported `*ThreadStore` method takes a record mutator; inside the package the door is reachable only from a `*ThreadStore` method) plus the `checkedFiles < 20` / `doors == 0` self-checks mean the guard notices when it stops guarding. Verified: it is what the plan's "derive, don't list" asked for, delivered better than planned.
- **`RetireUnprovenIncarnation` (`threadstore.go:553-593`) splits by evidence, not by convenience.** Refusing to widen `RetireIncarnation` and giving archive its own transition is exactly right — and `TestRetirementTransitionsTakeExactlyTheStateTheyName` tables *both* directions so neither becomes a superset of the other.
- **`archiveRefusal` is reachable, not decorative.** Because the menu is built from the previous refresh and `ArchiveThread` re-classifies at press time, every arm (`live`, `busy`, `unusable/unknown`) is a message an operator can actually see through the race. Good call keeping `archived` out of it.

## 2. Critical findings

None.

## 3. Important findings

### I1 — three new store transitions ship fail-closed preconditions with no test
**This is the 5th finding in family `fail-closed-guard-untested`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

Mutation-verified in a pinned scratch tree at `138d0475`: deleting all three guards below and running `go test ./cmd/internal/couchcore/` produces **no new failure**.

- `cmd/internal/couchcore/threadstore.go:620` — `RetireProvedDeadIncarnations`' `noOpenStartClaim`
- `cmd/internal/couchcore/continuation_store.go:47` — `BeginContinuationFromRetiredIncarnations`' `noOpenStartClaim`
- `cmd/internal/couchcore/threadstore.go:602` — `ClearVerifiedPark`'s "thread carries no verified park to clear"

The second is not merely uncovered, it is a **new production refusal**: `prepareAbsentContinuation` (`recovery_execute.go:326`) does not pre-screen start claims the way `RetryContinuation` (`continuation_recovery.go:304-306`) does, so a record reaching absent-continuation recovery with an open claim now fails where it previously wrote. The plan's revision claims this precondition as a deliverable ("plus `RetireProvedDeadIncarnations`' start-claim precondition, which the callback had no way to state") — a claimed behaviour change with no regression evidence.

The rule: **a transition introduced so a precondition has somewhere to live is not delivered until a test drives it into that refusal, and the table's domain is derived from the transition set, not hand-listed.** `TestRetirementTransitionsTakeExactlyTheStateTheyName` is the right shape but hand-lists the two transitions its author was thinking about; M3 added five. Derive the domain the way `mutation_door_test.go` already derives its own (exported `*ThreadStore` methods reachable from the mutator door), assert each names at least one refusal a test enters, and the next transition joins by existing.

### I2 — the new classifier branch has no shape in `everyThreadShape`
**This is the 3rd finding in family `vocabulary-entry-without-producer`.** Same rule read the other way — a producer with no corpus entry rather than a vocabulary entry with no producer. Do NOT fix this instance; fix the bijection.

`ClassifyThread` gained a branch at `cmd/internal/couchcore/actionableinventory.go:500` (`len(evidence.Unproven) != 0 → unusable/unknown`). `evidence.Unproven` is set in **no** case of `everyThreadShape` (`classify_test.go:35-317`; grep confirms `Unproven` appears nowhere in that file). The corpus's own header calls it "the cross product, not a sample — a future branch that forgets to classify something fails here instead of silently vanishing", and four guards drive it (`TestClassifyThreadIsTotalOverEveryRecordShape`, `…AcceptsExactlyWhatTheOldProjectorAccepted`, `TestEveryReasonIsProducedBySomeShape`, `TestProjectionNeverProducesArchived`, plus `parkedproducers_test.go:188`).

The concrete cost: `TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted` is the test that would have forced M3 to *state* that an unprovable probe turns a previously-actionable row unusable — the one actionability change in the milestone — via `wasActionableBefore`/`newlyActionable`. It never saw it. The branch is not unpinned (`TestAnUnprovableRecordedProcessIsNotConfirmedAbsence` covers it end-to-end through `ActionableThreadInventory`), but the corpus that is *declared* to enumerate every branch now does not.

The rule: **`everyThreadShape` is the classifier's producer enumeration, so adding a branch to `ClassifyThread` without adding a shape is the same defect as adding a vocabulary entry with no producer — and it should fail the same way.** A mechanical form exists: a guard that counts distinct `(state, reason)` verdicts the corpus produces against the branch count, or (cheaper and in this file's existing idiom) requires every `ThreadEvidence` field the classifier reads to be non-zero in at least one shape.

### I3 — four sites still state the occupancy claim M3 retired
**This is the 7th finding in family `stale-wording-after-referent-change`** (the atlas site is also the 4th in `atlas-contradicts-code`). Earlier rounds fixed instances. Do NOT fix these four — state the rule.

`archivableRecord` no longer refuses an occupied incarnation (`thread.go:361-372`; `TestStoreArchiveGuardAsksOnlyWhatARecordProves` and `TestStoreArchiveRefusesAnUnfinishedTransaction` both prove the store now *permits* a live incarnation). These still say it does:

- `cmd/internal/couchcore/threadstore.go:1125` — "It refuses a thread that is still LIVE or mid-park." False: it refuses an open park or an outstanding start claim.
- `cmd/internal/couchcore/detach.go:242` — "The store's own guard is what refuses an occupied thread, and it applies the same rule to a record it cannot read".
- `cmd/internal/couchcore/detach.go:388` — "`archivableRecord` needs a decoded record to prove the thread is not live". It proves nothing about liveness now.
- `atlas/couch.md:252` — "It refuses a live/unknown helper or an open start/park transaction" — in the *same file* whose M3 section 1300 lines later correctly says the occupancy half moved out. Two statements, one file, one true.

The machinery to catch this exists and did not fire. `issue256RetiredClaims` (`retired_referents_test.go:35-37`) gained two M3 entries — `"archivableRecord's occupancy"` and `"occupiedIncarnation"` — both keyed to the **identifier the author remembered**, and none of the four sites above spells the claim that way.

The rule: **a retired-claim entry is derived from the homes at the commit that retires the claim, not written from memory.** Before adding an entry, grep every home (production Go, `atlas/`, `README.md`) for the retired predicate's *vocabulary* — here `live`, `occupied`, `hosting` in the neighbourhood of `archive`/`archivableRecord` — and add each surviving phrase as data. The entry is complete only when that grep returns nothing; today it returns four lines.

### I4 — the archive cost bound is measured on the shape that cannot pay it
**This is the 2nd finding in family `envelope-declared-not-enforced`.** Earlier round fixed an instance. Do NOT fix this instance — state the rule.

`cmd/internal/couchcore/action_admission_test.go:293-330` (`TestArchivePaysOneEvidenceRoundAndNoClientQuery`) asserts `list-clients == 0` and the `## Log` records "**1 host-wide `list-sessions`, 0 `list-clients`, 1 ledger read** per `Couch.ArchiveThread`… The `list-clients` zero is the one that matters (~250 ms each, #228)". The fixture is `addresses[5]` — "a saved profile, no incarnation, no park, **no session**". With no session, all three `observeRecovery` calls short-circuit at `binding.Present == false` (`recovery_execute.go:77-93`) and no `DetachedSessions` runs.

Measured on the shape the milestone is actually about — a `detached` row, whose session is live and whose agent the new confirmation warns "may survive" — archiving costs **3 `list-clients`**, i.e. ~750 ms at #228's figure:

```
classification = "detached"/"" archivable=true
archive err=<nil>
list-sessions=1  list-clients=3  ledger=0     (at 138d0475)
list-sessions=0  list-clients=3  ledger=0     (at c4cbd1fe — the 3 are M2's, the 1 is M3's)
```

The 3 are pre-existing (`ArchiveThread` calls `observeRecovery` three times: `first`, inside `reconcileRecoveryHelper`, and `latest`); what is new is the *claim* that the number is zero. The test name also asserts a property of `Couch.ArchiveThread` generally while covering one branch — the `test-name-contradicts-assertion` shape.

The rule: **a cost bound is asserted on the input class that maximises it, and the fixture selection is justified in the test by naming the branch that would pay more and why it cannot.** Here the correct assertion is two rows — the sessionless one at 0, and the detached one at its real figure — so a future change that adds a fourth observation round fails instead of being invisible.

## 4. Minor findings

- `cmd/internal/couchcore/startup.go:35-43` — `rank` still hand-lists `ThreadDetached`/`ThreadParked` beside `ResumableState`; reverting the eligibility test to `rank(row) == 0` in a scratch tree leaves the suite green, so the two lists are still one-and-a-half, not one (ARCH-DRY). Deriving `rank`'s domain from `ResumableState` would make a future third resumable state sort correctly instead of silently at rank 0.
- `cmd/internal/couchcore/actionableinventory.go:910` — `observeExactProcessOrUnknown(c *Couch, …)` takes the whole `Couch` to reach one seam; `(proc ProcOps)` with the nil check at the call site keeps the helper at the seam's altitude.
- `cmd/internal/couchcore/detach.go:1125`-region comment aside, the store's "Second line of defence: `Couch.ArchiveThread` runs **the same guard**" (`threadstore.go:1165`) is now half-true — the couch layer runs `ArchivableState` *and* `archivableRecord`. Task 8b Step 3 is ticked for rewriting exactly this comment; it still claims a symmetry that no longer holds. (Covered by I3's rule.)

## 5. Test coverage notes

- Mutation-verified as **pinned** (each goes red without its fix): archive's `ArchivableState` admission → 4 tests; the `Unproven` classifier branch → 2 tests; `menuArchiveOffered` → 2 tests; `clearLifecycleDebris`'s unproven routing → 2 tests; the confirmation's "may survive" clause → 1 test.
- Mutation-verified as **unpinned**: the three preconditions in I1.
- `SelectResumableRoot`'s switch to `ResumableState` is a behaviour-preserving refactor (mutation leaves the suite green), which is correct — `TestStartupSelectionDerivesFromResumableState` pins the equivalence rather than a change.
- The `DISCRIMINATING:` convention is applied consistently to every refusal assertion M3 added, and it earns its keep: `TestArchiveRefusesEveryOccupiedIncarnationNotJustLive` and `TestArchiveKeepsAnUnprovenIncarnationItCouldNotProveDead` both sit behind three or four guards that would produce a bare error.
- Full-suite evidence: `go test ./...` at head vs base, both in pinned scratch trees, produces identical `--- FAIL` sets after removing scratch artifacts (git-dependent `TestIssue151M3*` and `TestEveryCoreConcept*`). **No new failures.** All remaining failures are `operation not permitted` from pty/exec in this environment.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass with the Minor above; `noOpenStartClaim` shared between two transitions is the right consolidation.
- **ARCH-PURE** — pass. `ArchivableState`, `ResumableState`, `archiveRefusal`, `launchProfileAgent`, `menuArchiveOffered` are all pure over `(state, reason)` / a record and are tested without IO; the evidence gathering stays in `Couch`.
- **ARCH-PURPOSE** — flag, via I1 and I3: both are "fixed the instance the author had in mind, not the class the milestone created". `ResumableState`'s only production consumer is `SelectResumableRoot`, not `DecideResume` — the plan's revision argues that on ARCH-CONSTRAINTS grounds and I accept it, but note the consequence: the resume row in `TestActionOfferedImpliesPermitted` compares the menu against a predicate that no Enter-path guard consumes, so offered⇒permitted for resume rests on M1 having removed the bookkeeping reads from `DecideResume`, not on this table. Worth a sentence in the atlas so the next reader does not assume the table covers it.
- **ARCH-MOCK** — pass, strengthened. `fake_refusal_conformance_test.go` closes the seam/fake sentinel gap structurally, and its behavioural half proves the fake can be *driven* into the state, not just that it wraps the word.
- **ARCH-CONSTRAINTS** — flag, I4.
- **ARCH-SECURE** — pass. `ActionableThreadSummary.Agent` reaches the confirmation only on the `detached` branch, which `ClassifyThread` gates behind `launcher.IsSupportedAgent`, so no arbitrary record text reaches the TUI; `clearLifecycleDebris`'s header correctly treats a foreign-version record as untrusted and takes the validator's accepted set as the domain.
- **ARCH-ORDER** — mostly pass: read-only admission precedes every write in `ArchiveThread`, `clearLifecycleDebris` screens all preconditions before its first (permanent) tombstone, and three-valued liveness now survives the projection. Flag via I2 — the shared shape corpus is where this component's ordering guarantees are tested, and the new branch is outside it. Also note for `#275`: `ThreadEvidence` is now seven independent fields (`Live`, `Unproven`, `Session`, `StartOwner`, `Parked`, `ParkedStatus`, `PathError`) whose legal combinations live only in `ClassifyThread`'s branch order; `Live` non-empty **and** `Unproven` non-empty is reachable and resolved only by that order. The totality tests mitigate it today, but this is the constellation the entry warns about.
- **ARCH-FUNERAL** — pass, and notably *positive*: M3 creates no new durable artifact family, and `RetireUnprovenIncarnation` gives an end to a record shape (`markLiveRecordUnknown` + dead helper) that previously had none.

## 7. Plan revision recommendations

The plan's `## Revisions` already carries the four re-derivations honestly, and `TestIssue256PlanTablesMatchTheTree` passes over all 19 M3 rows (I verified each row's symbol exists or is absent as stated). Two additions:

- **A `## Revisions` entry for Task 8b Step 3.** The step is ticked but its deliverable — "update the `:1093-1095` comment to say what the two layers now each own, rather than claiming a defence it no longer provides" — is not in the tree (`threadstore.go:1125` and `:1165`). Either land the comment rewrite or record the step as partially delivered; a ticked step whose stated artifact is absent is the divergence the tables test exists to prevent, one level up from the tables.
- **A `## Revisions` entry recording the archive cost correction.** The `## Log`'s "0 `list-clients` per `Couch.ArchiveThread`" should be restated as "0 for a row with no session; **3** for a `detached` row (measured at `138d0475`), of which 3 predate M3" — so the next person budgeting this path starts from the real figure rather than re-deriving it.
