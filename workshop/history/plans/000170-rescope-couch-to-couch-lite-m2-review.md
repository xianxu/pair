# Boundary Review — pair#170 (milestone M2)

| field | value |
|-------|-------|
| issue | 170 — Rescope couch to couch-lite |
| repo | pair |
| issue file | workshop/issues/000170-rescope-couch-to-couch-lite.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | b5aab2370c1e4386e31f073f271f9413ff977b31..6e2793d8164f31da57c75ad0663a57e1ffb1ecad |
| command | sdlc milestone-close --issue 170 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-02T16:05:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 lands a genuinely well-designed detach: `RetireIncarnation` is `FinalizePark`'s removal half with exact-identity authorization, the projector's detached branch requires **zero** incarnations so a crashed couch cannot masquerade as a clean detach, SIGKILL is never sent and that is asserted, and the `DeleteStart` data-loss fix was made as the class rather than the instance and is pinned by a test that reddens without it. What blocks SHIP is not the state machine but the seams around it: mutating `ResumeContext`'s detached observation to `if false && …` leaves the entire suite green, so the milestone's headline capability ("detached sessions reattach") is unpinned at every level — no unit test on the seam, no test of the production `DetachedSessions` reader, and the live zellij conformance case the plan ticks (`Task 11 Step 4`) was never written. Two ticked plan steps are undelivered, `README.md:297` still claims Leave parks, `LeaveResult.Skipped` has no consumer, and detach omits the `expectedExits` bridge park uses — so roughly half of all `alt+d` presses will emit the exit notice Step 5c exists to suppress. All are cheap; none require redesign.

## 1. Strengths

- **`ThreadStore.RetireIncarnation`** (`cmd/internal/couchcore/threadstore.go:414`) — exact `{PID, Identity}` as the authorization, refuses open park/start transactions and non-live incarnations, and deliberately writes no verified park. `retireincarnation_test.go` crosses record shape × identity × revision properly.
- **The `DeleteStart` fix is the class, not the instance** (`threadstore.go:784`). `TestDeleteStartKeepsARecordThatHasEverStarted` fails without it (the unnamed+profile case falls through to `deleteThreadIf` and deletes the record) — verified by reading the pre-image branch.
- **Fail-closed in both ambiguity directions** in `ProjectDetachedSessions` (`detachedsessions.go:44-70`): two addresses claiming one session name, and two zellij rows sharing one name, both yield nothing. Table-driven tests cover each.
- **`TestConsoleRunAltDDetachesWithoutConfirmation`** drives raw `ChordEncodings(ChordAltD)` bytes through the production input path — which is what caught `reduceParkHotkey` returning no effects, per the project note.
- **`selectMenuItem(state, name)`** (`menu_test.go:845`) replaces a dozen positional `KeyDown`-counting fixtures. That is the right response to reordering the action list, not the cheap one.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `alt+d` emits a spurious exit notice about half the time (`cmd/internal/couchtty/console.go:1325`).** `consumeExpectedParkExitLocked` gained `"detach"`/`"leave"` arms, but the `expectedExits[id]` bridge one frame above is still `origin.Operation == "park"` only. That bridge exists precisely because `c.exited` and `c.operationQueue.results` are separate `select` cases (`console.go:651,668`) and Go picks uniformly at random among ready ones; `reduceOperationResult` clears `InFlight` (`menu.go:1145`), so when completion wins the race the exit falls through to `ExitNotice`. Fix: `if (completed.origin.Operation == "park" || completed.origin.Operation == "detach") && err == nil`. `ARCH-PURPOSE` — Step 5c named one site and only that site was fixed; the rule ("a lifecycle operation's child exit is expected in *either* event order") has two halves.

**I2 — the detached-reattach seam has no test at any level (`cmd/internal/couchcore/resume.go:220-231`).** Reverting the wiring to `if false && thread.VerifiedPark == nil` and running `go test ./cmd/internal/couchcore/... ./cmd/internal/couchtty/... ./cmd/internal/couchcmd/...` produces no new failure (verified; only the pre-existing sandbox pty failures remain). `DecideResume` and `ReconcileResumeAdmission` are tested as pure functions with `Detached` hand-fed, which is exactly the "hand-feeds the derived value into the funnel" trap `workshop/lessons.md` gained an entry for *this round*. Fix: a `ResumeContext`-level test with a `FakeThreadArtifactCollisionChecker` whose `SetDetachedSession` is the only thing making the resume admissible.

**I3 — Task 11 Step 4 is ticked but not delivered (`workshop/plans/…-plan.md:362`).** `cmd/internal/launcher/session_quiescence_live_test.go` is untouched in this range (last commit `022b5a96`, `#152`) and contains no detach case. The behaviour detach *rests on* — a zellij session surviving its client's death — has no conformance check against the real binary, and the estimate carries a `real-api-discovery` line for exactly this. `ARCH-MOCK`: the fake models a state nothing confirms.

**I4 — the M2 operating-envelope claim is unmeasured and, as written, wrong (`workshop/plans/…-plan.md:365`, `atlas/couch.md:400`).** Step 4c ("record the numbers in the issue `## Log`") is ticked; the issue Log at HEAD has an M1 envelope block and no M2 measurement. The claim itself does not hold: `ZellijSource.Snapshot` (`launcher/zellij.go:15-38`) runs `list-clients` once per **pair session on the host**, not per candidate — bounding `detachedCandidates` only gates *whether* `Snapshot` runs, never its N. So a couch with 8 attached threads and 1 detached candidate pays 2+8 spawns per refresh, and `ZellijSource.run` (`zellij.go:49`) uses a bare `exec.Command` with no timeout and no ctx, on the periodic refresh worker. `ARCH-CONSTRAINTS`. Fix: measure and record, and correct the "proportional to detached threads" sentence in both plan and atlas.

**I5 — `LeaveResult.Skipped` is written at zero read sites (`cmd/internal/couchcore/park.go:120,167`).** The design's justification is "named rather than silently dropped, so the operator learns about them here instead of discovering an occupied thread later" — but `finishOperation` discards the result and calls `c.Stop()` on leave success (`console.go:1337-1340`), so the operator learns nothing. `Detached`/`Parked` are equally unread. Fix: surface the counts on the way out (a final line before release), or state in the doc comment that it is a return value for tests only.

**I6 — `README.md:297` still documents the pre-M2 Leave.** "Leave Couch parks every active actor and returns to the shell." Step 4b fixed the sentence at `:360-366` and missed this one. It now contradicts both `README.md:363` and `atlas/couch.md:387` inside the same file.

**I7 — a post-teardown `RetireIncarnation` failure has no recovery and no test (`cmd/internal/couchcore/detach.go:98`).** The CAS uses `record.Revision` read before a SIGTERM, a bounded wait and two zellij observations. If it fails (revision moved, or a store IO error), the client is already dead and the record keeps a stale `IncarnationLive` — invisible in the projector and refused by `DecideResume`, i.e. the exact state `#171` describes, reached from an ordinary failure path rather than a crash. The design note "a failed CAS leaves the thread live and occupied" is true only before the signal. Plan Step 3 lists this case; `detach_test.go` does not cover it. Fix: re-read the record after the wait and retry on `*ThreadRevisionError` (the loop shape `MarkIncarnationUnknown` at `threadstore.go:730-756` already uses).

**I8 — the plan's M2 Core-concepts table omits an entire new file.** `ProjectDetachedSessions`, `SessionNameBinding` (`cmd/internal/couchcore/detachedsessions.go`, both new and PURE) and `ActionableThreadSummary.Resumable()` appear in no row. Separately, `ProjectActionableThreads`, `DecideResume` and `Couch.Leave` are marked `new` when the diff shows them modified. Nothing catches this: `couchtty/core_concepts_contract_test.go` now filters #170 rows to `cmd/internal/couchtty/` paths, and couchcore has no counterpart contract.

**I9 — the uncommitted `if false &&` in the working tree (`cmd/internal/couchcore/resume.go:221`).** A scratch mutation-check left in place. Outside the pinned range, but it disables the detached-resume observation and is one `git add -A` from landing.

## 4. Minor findings

- `ARCH-DRY`: the reverse session-name-index scan is copy-pasted between `PairSession` (`artifactcollision.go:146-153`) and `DetachedSessions` (`artifactcollision.go:225-232`) — extract `lookupSessionName(index, address)`.
- `ARCH-DRY`: `DeleteStart`'s verified-park branch (`threadstore.go:770`) and new launch-profile branch (`threadstore.go:784`) are near-identical `UpdateExistingThread` closures differing only by one predicate.
- `atlas/couch.md:415` — the inserted `DeleteStart` paragraph runs into "It atomically records a creating/start claim…", whose subject was Resume. The sentence now reads as a claim about `DeleteStart`.
- Error handling is inconsistent across the diff: `PairSession` raises on an unreadable index, `DetachedSessions` swallows it per scope (`artifactcollision.go:222`), and `ActionableThreadInventoryContext` swallows the whole observation error (`actionableinventory.go:247`) with no notice. A thread silently vanishing from the switcher is the papercut class this issue exists to close.
- The projector's detached branch (`actionableinventory.go:151-156`) does not check `IsSupportedAgent`/`Argv != nil` while `parkedResumeProofMatches` does. Unreachable today because `ActionableThreadInventoryContext` filters candidates, but it is an asymmetry in a function whose contract is "fails closed".
- `README.md:299` still says a resumed thread is "restored as home" — the root-actor/home concept M1 deleted.

## 5. Test coverage notes

Pure-model coverage is strong: `TestProjectDetachedSessions`, `TestProjectActionableThreadsDetached{,DoesNotDisturbOtherStates}`, `TestThreadStoreRetireIncarnation`, `TestDeleteStartKeepsARecordThatHasEverStarted`, `TestDecideResumeAcceptsDetachedWithoutVerifiedPark`, and the two `alt+d` interceptor tests all pin real logic and would redden on inversion. The gaps are all at seams:

- `ScopedThreadArtifactCollisionChecker.DetachedSessions` (~50 new lines: per-scope grouping, index scan, error policy) has **no test** — `artifactcollision_test.go` is listed as "Modify" in Task 7 and is absent from the diff. With I3 also missing, nothing at any level exercises real zellij → detached row.
- `ResumeContext`'s detached observation (I2).
- `Leave`'s `record.Park != nil → Recover` branch, and therefore the only producer of `LeaveResult.Parked`, is untested (plan guard 5).
- `Detach`'s post-teardown CAS failure (I7) and the plan's "session still attached after exit" case are untested. The latter is not currently expressible: `PairSession` returns only `{Name, Present}` with no client count, so detach's after-proof cannot distinguish "survived detached" from "survived still attached". Worth a line in the plan saying so.
- Plan Step 1b asked for a rollback test through `resume.go:233,236,242` and `StartRollback`; the delivered test is store-level. Acceptable, but the link from a post-claim failure to `DeleteStart` is unpinned.

## 6. Architectural notes

- **ARCH-DRY** — flag (Minor). Two duplications noted above; otherwise the reuse story is good: `SignalGroup` lifts `signalOwnedProcessGroup` rather than re-spelling `kill(-pid)`, `alt+d` bytes come from `ChordEncodings`, and the detached rule consumes `launcher.SessionDetached` rather than adding a second zellij authority.
- **ARCH-PURE** — pass. `ProjectDetachedSessions` and the `actionableThreadState` branch are pure and tested with no fakes; observation and teardown stay in the `Couch` shell.
- **ARCH-PURPOSE** — flag (I1, I2, I5). `DeleteStart` was fixed as the class, which is the principle applied well; `expectedExits` and `LeaveResult.Skipped` are the same principle missed — a finding's site fixed, its sibling left, and a field that reads as protection while doing nothing.
- **ARCH-MOCK** — flag (I3). The fake exists and is well-shaped (`FakeThreadArtifactCollisionChecker.DetachedSessions` answers only for requested addresses, deliberately), but the live conformance check that would detect the fake drifting from zellij was claimed and not written.
- **ARCH-CONSTRAINTS** — flag (I4). Unmeasured, and the bound as stated does not hold; an untimed, non-cancellable subprocess fan-out now runs on every refresh with a detached candidate.
- **ARCH-SECURE** — pass. Session names from `zellij list-sessions` reach `exec.Command` as argv elements, never a shell; the persisted `session-names.jsonl` is parsed fail-closed on duplicates and empties; no credential enters a log, argument list or fixture.

**For M3/M4:** `SelectUniqueResumableRoot` will consume `Resumable()`, so pin the `WorkingPath` physicalization change (`actionableinventory.go:219-226`) with a test before M3 depends on exact-string path matching — the comment already names it as load-bearing and nothing asserts it. And M4's `CommitStartClaim` must carry the widened detached precondition forward, or it silently re-breaks I2's path with the same green suite.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000170-rescope-couch-to-couch-lite-plan.md` covering:

1. **Core-concepts table correction (M2)** — add rows for `ProjectDetachedSessions` and `SessionNameBinding` (`cmd/internal/couchcore/detachedsessions.go`, PURE, new) and `ActionableThreadSummary.Resumable` (`actionableinventory.go`, PURE, new); change `ProjectActionableThreads`, `DecideResume` and `Couch.Leave` from `new` to `modified`.
2. **Task 11 Step 4 un-tick** — the live zellij detach conformance case was not added; either write it or record the deferral with its owner.
3. **Task 11 Step 4c un-tick and envelope correction** — no measurement was recorded, and the "2 + N bounded to candidates" claim is wrong because `ZellijSource.Snapshot` runs `list-clients` per host pair session. Restate the real cost and note the untimed `exec.Command`.
4. **Task 9 Step 3 scope note** — the "session still attached (non-zero clients) after exit" case is not expressible against `PairSession`'s `{Name, Present}` shape; say so, or widen the observation.
5. **Task 10 Step 3b** — the delivered test asserts non-established bindings are refused, which is the converse of the step's concern (that a 0-client session still *reaches* `BindingEstablished`). Restate what was actually verified.
