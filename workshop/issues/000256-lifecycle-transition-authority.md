---
id: 000256
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-16
estimate_hours: 5.44
started: 2026-09-16T20:08:18-07:00
---

# Enforce lifecycle transition authority and outcome uncertainty

## Problem

Split from #255 on 2026-09-15 when the operator narrowed that issue to terminal abstraction. Preserve the non-terminal findings of the 2026-09-14 audit; baseline was HEAD `5ebb381f` plus then-uncommitted #250 work. Revalidate all cited code against current main before design. These findings do not establish the unexplained disconnect's cause.

2. **Record validation is not transition authority.** `ThreadStore.UpdateExistingThread` (`cmd/internal/couchcore/threadstore.go:264`) accepts arbitrary mutation callbacks. CAS, immutable-field checks and final validation protect coherent records, but do not require an authorized state/event transition. Production continuation recovery mutates lifecycle fields through this door. Direct mutations already existed in HEAD; #250 adds more guarded reconciliation. Model these as explicit transitions rather than assuming every guarded mutation is a bug.
4. **Observation uncertainty is lost in projection.** `ProcOps` defines Live/Dead/Unknown, but `ObserveRecordedProcesses` (`couchcore/actionableinventory.go:568`) drops unknown and identity-read errors. `ThreadEvidence.Live` retains only positive evidence; `ClassifyThread` can therefore present an unobservable incarnation as stale (`:269–283`). #250's new execution path preserves uncertainty better than this diagnostic projection. Unknown must not be treated as confirmed absence.

The same audit found loss of process/transport outcome detail: ptychild/child.go discarded PTY read errors, procutil and launcher reduced termination to integer exit codes, and launcher cleanup outcomes were not uniformly consumed. Coordinate diagnostic evidence with #253. Terminal input/output failure handling and terminal-state consequences belong to #255; this issue owns propagation into attachment/process lifecycle decisions. Avoid separate competing outcome types.

## Spec

Make lifecycle transitions authoritative without flattening thread, native-session, process, incarnation and attachment into one global state machine. Map each resource to actual symbols, exact identity, transition owner, accepted events, effects and termination/recovery rules. Preserve independently surviving resources.

- Replace general lifecycle-field mutation with named transition APIs; validation, revision CAS and storage atomicity do not themselves authorize transitions.
- Separate desired state, observation and operation outcome. Preserve Live/Dead/Unknown and identity-read errors through projection, classification and recovery.
- Carry structured process/attachment outcomes and exact attempt identities; distinguish confirmed success, confirmed failure and unconfirmed outcomes, including partial progress.
- Retain existing reducers, transaction journals, supervisor leases and PID/start identities. Reconcile against completed #250 recovery and #253 telemetry instead of duplicating them.

## Done when

- Canonical resource/ownership mapping explains independent thread, session, process and attachment lifetimes.
- Lifecycle mutations use named transition APIs, with tests proving arbitrary callbacks cannot bypass transition rules.
- Unknown observations cannot become confirmed absence or authorize destructive recovery; stale attempts cannot mutate replacements.
- Process/attachment errors and partial outcomes preserve evidence and lead to defined reconciliation behavior.
- Composed lifecycle tests establish attachment loss does not imply session death and retain existing recovery guarantees; #253 and #255 outcome ownership is explicit.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Derivation, so the numbers can be checked rather than trusted:

- **Step 2 — primitives.** Twenty-two, mapping to the plan's 14 tasks plus
  operator verification and the boundary reviews. Most tasks are
  *smaller-go-module*: they extend machinery that already exists and the plan
  carries the code. Three are *cross-cutting-refactor* — Task 8 and Task 8a are
  the same class applied at two call sites (archive, resume), Task 11 is the
  mutation door across four packages. Two are *tui-screen*: **Task 2** because
  its blast radius is the whole classifier plus every existing
  stale-incarnation/busy test expectation (`couch_test.go` is 1747 lines,
  `plan_contract_test.go` 1750), and Task 10 for the confirmation screen. Task
  10's Step 0 is *scope-pivot*, because it is a measurement whose "no" answer
  makes `#274` a mid-flight dependency.
- **Step 2.5 — library availability.** N/A, stated rather than skipped: this is
  routing between our own packages and querying a binary we already wrap. No
  library short-circuits it.
- **Step 3 — spec-quality discount ×0.2 on design.** Applied to every primitive.
  The plan carries the branch table, file:line for every call site, the consumer
  enumeration for each evidence producer, and full test bodies; it cleared
  plan-quality in two rounds with three blocking findings resolved. Design hours
  here are the cost of *reading* that, not making it.
- **Step 4 — Method B.** Not used; every primitive matched the table.
- **Step 5 — familiarity ×1.0.** Familiar territory: every symbol the plan names
  has been read and verified against HEAD during planning, and both bugs have
  live reproductions in the operator's store.
- **Step 6 — buffer +15%**, the thorough-plan-doc case, not +30%.
- **v3.1 scaling.** Each `impl=` is 40% of the v2 primitive-table implementation
  hours. Design hours are unscaled.

**Five `milestone-review` items for three boundaries.** `pair#265`'s close ran 3
rounds and `#255` ran 5; budgeting one per boundary is the error the ledger keeps
recording. Three boundary reviews plus two expected extra rounds.

**Calibration note — recorded, not adopted.** The last nine pair closes with both
figures run a **median estimate/actual ratio of 0.71** (0.15, 0.46, 0.63, 0.70,
0.71, 0.97, 1.17, 1.27, 1.33), so this method currently lands ~1.4x low here.
`#265`, the nearest v3.1 comparator in this same subsystem, estimated 1.70 and
landed at 3.66 (ratio 0.46). Applying that bias would put this issue near 7.7h. I
have **not** inflated the total to match — back-fitting to a predicted actual is
exactly what the estimate-quality gate exists to catch, and the per-item hours
above are the honest reading of the table. **If this closes above ~7h, the signal
is the model's implementation scale, not this decomposition**; that is the number
to check at close rather than explain away.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.30 impl=0.32
item: smaller-go-module design=0.02 impl=0.12
item: smaller-go-module design=0.02 impl=0.08
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.16
item: cross-cutting-refactor design=0.12 impl=0.16
item: smaller-go-module design=0.04 impl=0.12
item: smaller-go-module design=0.04 impl=0.16
item: tui-screen design=0.16 impl=0.24
item: scope-pivot design=0.06 impl=0.16
item: cross-cutting-refactor design=0.16 impl=0.20
item: atlas-docs design=0.03 impl=0.08
item: ux-rename-iteration design=0.10 impl=0.12
item: ux-rename-iteration design=0.10 impl=0.12
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 5.44
```

## Plan

Rows are the milestones of the durable plan at
`workshop/plans/000256-lifecycle-transition-authority-plan.md`.

- [x] Revalidate the preserved audit findings against current code and coordinate #250/#253/#255. *(Done 2026-09-16: findings 2 and 4 confirmed against HEAD; finding 4's collapse is `actionableinventory.go:582`. Measured the process tree — see Log.)*
- [x] M1 — The classifier reads the session, not the bookkeeping: `Incarnation` and `record.Park` leave the classification path entirely. Fixes #271 and #272 by deletion. *(Done 2026-09-17; the class turned out to have FOUR sites, found one at a time — see Log.)*
- [ ] M2 — Make the operator's rows reachable (a start claim with no living owner, `DecideRecovery`'s park gate, the binding-absent hatch, the ledger as cold-resume authority) and verify against real sessions.
- [ ] M3 — Guards consume the classification; preserve Unknown on the destructive paths; archive confirms before stopping a live agent; close the arbitrary lifecycle-mutation door; atlas + lessons.

Split out, both depending on this issue: **#275** (replace the park transaction
with an ordered idempotent write) and **#276** (surface couch-tagged agents with
no thread record — carries #272's corresponding Done-when).

## Log

### 2026-09-17 — M2: what the tasks turned out to be

Recorded because four of the six tasks changed shape once the code was read, and
in every case the change was the same kind: **the rule already existed and was
unreachable, or the planned edit was unnecessary once an earlier one landed.**

| Planned | Delivered |
|---|---|
| Task 4: delete the dead `ThreadBusy` menu branch | Its premise was false — `ThreadBusy` survived M1 with a new producer. The row needed an **escape**: a start claim whose owner couch is provably dead is not in flight. |
| Task 4a + Task 5: teach archive to clear claims; delete `DecideRecovery`'s gates | One change, not two. The rule was already written as `retireDeadIncarnationBeforeStart`; it moved to `clearLifecycleDebris` and archive calls it. **Nothing was deleted** — each gate protects a real downstream precondition and simply stops being the operator's wall. |
| Task 6: teach `observeRecoverySession` to read an absent binding as absence | Unnecessary. With the debris cleared first, the record reaches the reconciler with no incarnation, which the existing hatch already admits. Two real defects surfaced instead: `observeRecovery` skipped the session probe for bookkeeping-carrying records, and the fake's absent-binding error did not wrap its sentinel, so `errors.Is` could never see it. |
| Task 6b: ask the ledger in the classifier | The **guard had to follow**. Four sites read the park receipt as cold-resume authority, including `Resume` itself, which resolved the binding only for receipt-holders — so a row the new classifier calls `parked` arrived at `DecideResume` with an empty binding and was refused `unbound`. |

Two vocabulary deletions fell out of the last one, both forced by the produced-by
guard: `ResumeLegacyUnverified` lost its producer, and the `ParkHistory`
tombstone scan stopped being a veto — archive abandons orphaned parks routinely
now, so vetoing on one would make "couch crashed mid-park once" a permanent
cold-resume ban.

**One regression I introduced and the rule that caught it.** Clearing debris is a
durable write and I put it ahead of the session observation, so archive refused
for an unanswerable session *after* retiring an incarnation — round 2's rule, an
irreversible step preceding a revocable check. The session half of
`DecideRecovery` is now `RecoverySessionRefusal`, asked first with nothing
written. A quieter break came with it: the extra observation shifted which pair of
looks the mid-flight-session check compared, so the final recheck is now compared
against the first look as well as the reconciler's.

**Behaviour the operator will notice.** A thread whose session is gone but whose
conversation still resolves is now offered a cold resume, and startup adopts it
rather than starting a second thread in the same tree.

**How M2's `--actual` was derived, because it is not a clean measurement.**
`sdlc actual` reports only the CUMULATIVE figure for an issue (7.82h, window
`b98b1192 → HEAD`) and has no window flag, and **M1's close recorded no
increment** — not in this file, the plan, the project file, or brain's
calibration ledger. The engine's own attribution segments end at
`2026-09-17 12:16`, which covers the M1 close and the compaction; M2's work ran
from ~12:25 and is not yet in the flushed transcripts. So 7.82h is essentially
M1-and-earlier, and M2's increment is this session's span: **1.4h**, chosen by
the operator over passing the cumulative. The issue close will adopt the measured
cumulative as usual, so calibration is unaffected either way.

**Verification.** `go test ./...` with the five-variable retention scrub: 88
packages, 71 ok, 17 with no test files, **0 failures**. `make -k test` shows one
failure, `test-changelog`
(`pair-changelog-open: viewer: process target is outside selected owner
directory`, from `validateProcessTarget` in `storagegc/lease.go`), reproduced on
`origin/main` in a throwaway worktree **and** under `env -i` — pre-existing and
unrelated. `make` halts the suite at it, so `-k` is required to see past it.

**M2's declared measurements, recorded rather than asserted.** The plan's
operator-verification item asks for the evidence-round timing against the
ARCH-CONSTRAINTS budget, and M2 widened two costs, so both are measured here:

| Figure | Before M2 | After M2 |
|---|---|---|
| ledger reads per refresh (`couchWithOneRecordOfEveryShape`, 6 records) | 2 (the park receipts) | 4 (every resume-shaped record with no session) |
| `Physical` calls per refresh, same fixture | 4 | 4 (unchanged) |
| host-wide `SessionPresence` calls per refresh | 1 | 1 (unchanged) |
| client-counting (`list-clients`) calls per refresh | 0 | 0 (unchanged) |
| whole evidence round, 6 records, fakes, mean of 20 | — | **0.62 ms** |

The cold side of the resume ACTION also grew: `ResumeContextWith` now observes
`DetachedSessions` on every resume including cold ones, which previously skipped
it — one `list-clients` (~250 ms, #228) for the single thread the operator
pressed Enter on. That is the strict-action half of optimistic inventory and it
does not scale with store size. What is NOT bounded by a test is the ledger read
as the store grows: `TestWarmRowsAskNoLedgerQuestion` bounds the warm side (a
hosted row and a detached row pay nothing), and nothing yet bounds the cold side.
Recorded as a known gap rather than left implicit.

**The ACTION path's own cost, measured on that path.** M2 moved a whole evidence
round behind an operator keypress: `PrepareAgentSwitch` classifies through
`classifyForAction`, and the switcher calls it for the form PREFILL as well as
the commit. Inheriting the refresh's figure would have been the wrong
measurement, so it is taken here:

| Per `PrepareAgentSwitch`, 6-record store, fakes | Count |
|---|---|
| whole call | **0.65 ms** |
| `SessionPresence` (one host-wide `list-sessions`) | 1 |
| `Physical()` | 4 — every resume-shaped record, not just the one asked about |
| `ResolveEstablished` (ledger) | 1 — the `ask` predicate narrows this one |
| `DetachedSessions` (`list-clients`, ~250 ms real) | 0 |

The unnarrowed figure is `Physical()`: `gatherThreadEvidence` applies `ask` AFTER
physicalization on purpose, because startup's predicate compares working paths
and an alias would otherwise miss its own thread. `classifyForAction`'s predicate
compares addresses and would be safe earlier, but moving the gate would change
the shared function for every caller — recorded as the knob to turn if this ever
matters, not turned speculatively.

**M2's operator verification is DEFERRED, owner: the operator.** Three items in
the plan need a live couch and cannot be discharged by the suite:

1. The two `brain` rows archive from the switcher, and a fresh start stops
   minting orphans (#273 observed one per attempt; the count must stop growing).
2. #272's fixture built rather than waited for: kill a couch while a thread runs,
   confirm the row reads `detached` **and that Enter actually reattaches** — a row
   that merely reads `detached` proves nothing.
3. The same for M2's new producer: a thread whose session is gone but whose
   ledger resolves must offer a cold resume that works, and `switch-agent` on it
   must now succeed rather than refuse.

Stated as a deferral with an owner because silence reads as done.

**One regression the package-scoped runs could not see.**
`TestProductionArtifactReferencesAreExactlyClassified` lives in
`cmd/internal/artifactpath` and refused the new `lifecycledebris.go` for being
absent from the exhaustive production-source inventory. Every couch-package run
was green. That is the argument for `go test ./...` at a boundary rather than the
packages you touched.

### 2026-09-17 — M1 boundary: four review rounds, and what each found
- 2026-09-17: closed M1 — make test with the retention-owner env scrub and a non-symlinked TMPDIR: 210 packages ok, exit 0. Round 5 REWORK addressed. BR-27 Critical: the same replacementUnknown validator escape at a different incarnation count -- an open park with ZERO incarnations was never cleared, so CommitStartClaim refused uncoded and couch would not start in the tree. Root cause is that round 2 rule was written into the code as four sites rather than as the predicate "every guard refusing on record.Incarnations or record.Park"; the clearing pass is now total over the shapes validateLifecycle accepts, with screening complete before any write and each write authorized by a probe of the entity it acts on. TestReAdoptionExitsAreTotalAndCoded gained an incarnation-count dimension, mutation-proven against the old bail. BR-28 and BR-19: the disproved premise and the site count re-derived in every home -- atlas, the function comment, the plan -- and the plan now points at ClassifyThread and everyThreadShape instead of restating the branch table, which had moved twice and each time became instructions to undo a boundary fix.; review verdict: FIX-THEN-SHIP

The M1 close took **four** boundary-review rounds. Recorded because the pattern
is the finding, not any single defect: each round found something the previous
round's *fix* introduced or left behind, and the gate's own summary from round 2
onward was *"Not converging: fix rules, not instances."*

| Round | Verdict | What it found |
|---|---|---|
| 1 | REWORK | The class had a **fourth** site: re-adoption made a park-open record reachable, so `RetireIncarnation`'s open-park precondition went live and wedged startup. Plus: one failed `list-sessions` demoted every *parked* row; two guards unpinned; the docs half untouched. |
| 2 | REWORK | Round 1's fixes were site-shaped. Same wedge reproduced through `CommitStartClaim`. An irreversible `AbandonPark` was running before a revocable check. |
| 3 | FIX-THEN-SHIP | Round 2's own Rule 1 was wrong: forcing every producer to carry a code **changed what the code meant** and broke `errors.As`/`Is`/`Unwrap`. `ResumeDiagnosticCode` had no produced-by guard, so deleting `occupiedResumeCode` orphaned `ResumeCreating` unnoticed. |
| 4 | REWORK | My "unrepresentable" claim was **false** — read from a validator's main clause, missing its exception — so the code probed one process and wrote a permanent tombstone about another. |

Rules that outlive the sweep, each now pinned:

- An irreversible step's precondition is proved about **the exact entity the step
  acts on**, and never precedes a revocable check.
- A guard omitted as "unrepresentable" cites the validator clause that makes it
  so, **read including its exceptions**, and is pinned by a test that builds the
  fixture through the real store.
- When a value's **meaning** changes, enumerate every reader and re-derive each
  in the same round.
- A vocabulary has a **produced-by guard**; a value nothing emits is a branch no
  test can reach.
- Guidance belongs at the **consumer** that needs it, not as a marker every
  producer must carry.
- A comment asserting what a path classifies **names the test that pins it**.

**ARCH-CONSTRAINTS, the figures the plan asked for.** The envelope is enforced
structurally rather than by a timing number, which is the stronger evidence:
`SessionPresenceQueries() == 1` and `DetachedQueries() == 0` are asserted at four
sites, and `TestSessionPresenceCountsNoClients` pins it against the production
checker with a stubbed `zellij`. In cost terms the refresh went from *one
`list-clients` per detach candidate* (~250 ms each, #228) to **one host-wide
`list-sessions` and no client query at all** — on the operator's 7-record,
4-scope store that is 0 client queries where the old path made up to 6. The
regression the plan flagged is real but bounded the other way: the snapshot now
runs on every refresh where it previously skipped entirely with nothing
detachable, and `Physical` is called for 4 records rather than 3. Both are off
the keystroke path (the refresh runs in a coalesced worker goroutine), and M2's
operator verification owns the wall-clock figure.

### 2026-09-17 — M1: one class, and it kept having one more site

`ClassifyThread` no longer reads `Incarnation` liveness or `record.Park`. Both
filed bugs fall out of the deletion, and the two shapes are asserted to classify
IDENTICALLY, which is the claim: the zellij server is PPID 1 at birth, so a
couch death kills only the launcher and a clean detach leaves the same external
state a crash does.

**The class had four sites, and each was found by fixing the one before it.**
Three were found by running it; the fourth by a boundary reviewer disproving a
claim of mine. The enumeration — *every guard refusing on `record.Incarnations`
or `record.Park`* — is the deliverable, not any single site, and writing it as a
list rather than a predicate is what let two later shapes through.

1. `ClassifyThread` — the one the plan named.
2. `DecideResume` (`resume.go:98,104`) refused on the incarnation, so a row the
   switcher advertised as `detached` could not resume. This was the
   plan-quality gate's Critical finding; the over-engineering re-cut had
   dissolved it for the classifier only. Pulled forward from M3 — M1 cannot be
   green while a guard contradicts the classification it feeds.
3. `CommitStartClaim` then refused a record still carrying the dead launcher's
   incarnation. Its own comment is right that one-incarnation-at-a-time is a
   store invariant rather than a lifecycle opinion, so the CALLER retires the
   stale claim, gated on confirmed `Dead` — an unobservable process must never
   cause a running agent to be abandoned.

Site 3 is the **re-adoption** task the re-cut dropped, on the reasoning "you do
not re-adopt what you never disowned". That held for the classifier and was false
for every guard downstream of it. Both plan reviewers had flagged re-adoption;
the re-cut talked itself out of it one layer too early.

`ThreadBusy` survives with exactly one producer — a `ThreadStartClaim`, which is
couch's record of its OWN in-flight operation, not a claim about an external
process. Without it, the window between claiming a start and the launcher
acquiring a pid would classify `session-gone`, an archive-eligible reason, for a
thread starting normally.

Retired `stale-incarnation` and `unrecorded-child`: both named a disagreement
between record and observation, and there are no longer two sides to disagree.
`unrecorded-child` returns with #276, which gives it a producer; keeping it as a
placeholder would have silenced the guard that found it.

The refresh stopped counting clients entirely — presence is one host-wide
`list-sessions` — so the optimistic-inventory decision is now in the code.

Scale, for the calibration ledger: ~20 test expectations restated, each with why.
Several encoded the bugs as requirements (*"a stale live incarnation stays
hidden"*, *"an occupied incarnation refuses even with the detached proof"*). None
weakened to pass. Two repo guards fired correctly — the new file had to join
`artifactpath`'s exhaustive inventory, and `occupiedResumeCode` became dead and
was deleted rather than allowlisted.

Verification: `make test` with the retention-owner env scrub and a non-symlinked
`TMPDIR` — **210 packages ok, exit 0, zero failures.**

A note on method: two earlier "failures" in this milestone were **pipe
artifacts**, not code. Piping `make test` into `head`/`tail` SIGPIPEs the run and
make reports an error with no failing package. Redirect to a file and grep the
file.


### 2026-09-16 — Over-engineering audit; re-cut around one rule

Operator observation, which the plan is now built on:

> if couch crash, since we know zellij is not affected, thus all threads'
> essentially live, we should pick the default, that that state is recoverable,
> not relying on clean "shutdown" signal. that shutdown seems to be cosmetic?

Verified in code. `Detach` (`detach.go:91-99`) SIGTERMs the **launcher's** process
group, waits for that pid, and clears the incarnation. The launcher is couch's own
child; the zellij server is PPID 1. So **a clean detach and a couch crash leave
identical external state** — the only difference is whether the bookkeeping ran,
and today that difference is `detached` (recoverable, ranked highest) versus
`stale` (debris). Same world, opposite verdicts. That is the whole family.

The plan inverted from addition to deletion: the classifier stops reading
`Incarnation` and `record.Park`. **One task now fixes both #271 and #272.**
Four milestones → three, 18 tasks → 12. Dropped the re-adoption task (you do not
re-adopt what you never disowned), cut `SessionObservation` from four states to
three (under optimistic inventory the refresh can never emit the fourth), deleted
`startInFlight` rather than narrowing it (a start in flight is couch-local
in-memory knowledge, not durable state), and demoted three-valued liveness from a
foundational milestone to one guard on the destructive paths.

Split out rather than absorbed:

- **#275** — replace the park transaction with an ordered idempotent write. Park's
  one non-cosmetic property is that it is irreversible and ordered; that needs
  ordering and idempotence, not phases, attempts, nonces and tombstones. #256 only
  stops the *classifier* reading it.
- **#276** — surface couch-tagged agents with no thread record. Needs a `repos/*`
  enumeration, a new seam and fake, and a projection field, for a report-only row:
  additive, not corrective. #272's corresponding Done-when transfers there.

### 2026-09-16 — Operator cleanup; #272's primary fixture is gone

The operator archived every thread not live-attached to the running couch: **17
records → 7**, across 4 scopes, 40 archived. Verified no new orphan was created —
`ArchiveThread`'s Quiesce-first ordering held, and every live `pair wrap` on the
host maps to a remaining record, a direct (non-couch) `pair` session, or an
orphan that predates the cleanup.

Consequence for verification: the three muse threads with dead launchers and live
agents — #272's headline evidence — were among the archived, so "eleven records
carrying `recorded: live` with a dead pid" is **no longer reproducible**. That
path now needs a built fixture (kill a couch while a thread runs) rather than an
observed one.

Still live and still recordless, and now the only standing #272 fixture:
`couch-797c45e8e649a9bb` (📁parley-couch, `pair wrap` 84488) and
`couch-2583ed61c0ab6ebe` (📁parley-couch-2, `pair wrap` 1130), both running since
2026-08-30 with intact conversations couch cannot see.

The two `brain` records remain the oddity and are the live fixtures for the M2
work: `couch-3b82bfd593cac896` (dead incarnation, no session binding) and
`couch-e1a31510b7033d08` (orphaned park, then the same wall). Tracing why the
operator could not archive them found a gap the plan had missed — see the plan's
new Task 8.

### 2026-09-16 — Plan reviewed; the re-base alone was a regression

Two fresh-context reviews over disjoint halves of the durable plan converged
independently on one defect, which the plan would otherwise have shipped.

**Re-basing liveness onto the session is only half the fix.** Reclassifying
`#272`'s eleven records to `detached` does not make them reattachable:
`DecideResume` (`resume.go:101`) refuses any record with an occupied incarnation,
and all eleven carry a stale `live` one. Meanwhile `SelectResumableRoot` ranks
`detached` **highest**, so startup auto-selects them, and `ProjectRecoveryChoices`'
gate on `reason == ReasonStaleIncarnation` (`recovery.go:84`) stops matching — so
the one gesture that works today disappears. Net: auto-selected, offered a resume
that always fails, recovery removed. M3 now carries an explicit **re-adoption**
task that retires the stale incarnation once the session proves survival.

**The park branch needs no new observation channel.** `ParkIdentity` is copied
from `soleParkableIncarnation` (`park.go:302`) — the process *being parked*,
already probed by `ObserveRecordedProcesses`. The earlier reading (that
`resumeShaped` starves the park branch of evidence) was wrong: `item.Live` is
populated at `:455`, before that gate. So M1's Unknown plumbing is the entire fix
and the proposed `observeParkOwners` would have duplicated an existing probe.

Also corrected: `RecoverActiveParks` is already wired in production
(`couchcmd/run.go:348`) over the identical record set, so the orphan rule lands
there and the unused `ReconcileActiveParks` is deleted rather than wired; session
*existence* is not `detached` (attached-elsewhere, and server-alive-agent-gone —
`87464` 📁brain-couch-2 — both misclassify); and adding a `ThreadReason`
hard-fails three guards that a task now owns.

Milestones reordered: `#271` moves ahead of the liveness re-base. It is
self-contained, needs only M1, unwedges `brain` soonest, and carries none of M3's
risk. Full delta in the plan's `## Revisions`.

### 2026-09-16 — Measured the lifetime that liveness should key to

`#272` asked for measurement rather than assumption. Taken from the live process
table with couch pid 65018 **still running**:

```
65018   61189    bin/couch
66197   65018      pair resume couch-dbc88727c6378a0f --layout3   ← launcher
66242   66197        zellij (client)
66246       1    zellij --server … 📁brain-couch-26               ← PPID 1 AT BIRTH
66247   66246      pair wrap → 66261 claude --session-id d16964c6-…
```

The zellij **server** is PPID 1 while couch is alive — it daemonized, it was not
reparented. `pair wrap`, `pair term`, nvim and the agent are its children, not
couch's. A couch death therefore kills exactly the launcher, the zellij client
and `pair title`.

So the liveness referent is the **zellij session**, not `pair wrap` as `#272`
hypothesised: it is independent from birth rather than surviving by reparenting,
it dominates `pair wrap` and the agent in lifetime, and it is the same entity
`zellij list-sessions` already reports — which collapses "is the thread alive"
and "is its session alive" into one question with one authority (ARCH-DRY).

Corroborating orphans in the same snapshot: `84487` 📁parley-couch and `1127`
📁parley-couch-2 (both PPID 1, live agents, no store record — `#272`'s
unrecorded-child case); `87464` 📁brain-couch-2 (server alive, agent gone).

Also confirmed the structural root of `#271`: `gatherThreadEvidence`'s
`resumeShaped` gate (`actionableinventory.go:460`) excludes `record.Park != nil`
from **every** evidence branch, so the park branch consults nothing because the
shell gathers nothing for it.

### 2026-09-15 — Scope extracted from #255

Preserved the generic lifecycle authority, observation uncertainty and process/attachment outcome findings while #255 becomes the terminal-abstraction issue. No implementation, diagnosis of the past disconnect, or new blocking dependency is asserted by this split.
