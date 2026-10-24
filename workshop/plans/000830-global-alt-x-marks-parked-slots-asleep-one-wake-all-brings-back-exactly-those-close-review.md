# Boundary Review — pair#830 (whole-issue close)

| field | value |
|-------|-------|
| issue | 830 — Global Alt+x marks parked slots 'asleep'; one 'wake all' brings back exactly those |
| repo | pair |
| issue file | workshop/issues/000830-global-alt-x-marks-parked-slots-asleep-one-wake-all-brings-back-exactly-those.md |
| boundary | whole-issue close |
| milestone | — |
| window | aafc7c8c08f9d49b950b38265beecc9ed569b1b2..a520f3a5af36b66ca0a612aee8deaa95f1a0ec34 |
| command | sdlc close --issue 830 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-24T04:23:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: the core and record changes are read, and now I'm going through the CLI, protocol and docs diff.

Tests have finished. I'm checking which failures come from the review sandbox and which are real.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The code does what the issue's Revisions ask for. A full quit's fresh park writes a sleep mark in the same CAS as `BeginPark`, tied to that park's nonce. "Asleep" is computed from the record by `ThreadAsleep`, which is the only reader. The switcher and the `couch --recover-plan-from-sdlc` rows show it, and `couch --wake` from the main hub resumes only the other asleep hubs through the normal resume queue. All the #830-specific tests pass in this checkout, in couchcore, couchcmd, couchmessage, couchtty and threadrecord. The wider package runs failed in large numbers, but every failure I looked at came from this sandbox refusing `git init` outside its directories (sdlc #339), not from this diff. Two cheap paperwork gaps block SHIP. First, the durable plan file still describes the design the Revisions replaced, and its task boxes are all unchecked. Second, `## Done when` was not updated after the Spec changed, and it still claims tests this repo cannot hold. The end-to-end test (T6) is committed but skipped until pair#863 lands, which the TL accepted.

Done-when: stale — the wake gesture became the TL's `couch --wake` from the main hub, the "pending-work first" worker test belongs to ariadne#564 and not to pair, and "survives a full restart and clears on wake" has no end-to-end test running while T6 is skipped.

### 1. Strengths
- **The derived flag is the right design.** `cmd/internal/couchcore/asleep.go:57` says asleep only when the mark's nonce matches the verified park's. Every path that ends or replaces a park leaves the mark inert, and so does a record an older build rewrote. Writing the mark inside `beginPark`'s CAS (`threadstore.go:565-574`) makes it durable before any teardown (ARCH-ORDER).
- **The format change is additive, and the test proves it.** `SleepMark` is a new top-level member and keeps unknown members nested inside it. `TestSleepMarkPersistsAdditively` puts a nested member in, decodes, rewrites, and checks it comes back unchanged. It also shows an empty nonce or cause is refused (ARCH-EVOLVE).
- **The quit tests use the real Leave path.** `TestLeaveParkMarksTheParksItBegins` and `TestLeaveParkRecoverKeepsWhatTheBeginWrote` run Leave over the lifecycle fake. They cover an individually parked slot, the fresh retry after a rollback (new nonce, new mark), and Recover keeping or not keeping a mark.
- **Polling is shared, not copied.** `pollSlotOperation` (`messages.go:208`) was pulled out of `exchangeSlotOperation` so wake reuses the slot-operation polling (ARCH-DRY).
- **Wake's authority is narrow and checked twice.** `HubOwnership.Wakes` decides who may wake what. The effect gate checks again, at execution time, that the caller is still the live main hub and the target is still a hub (`wake.go:331-344`). The rig test shows a worker and a non-main hub are refused, and a repeat wake comes back busy without queuing anything.

### 2. Critical findings
None.

### 3. Important findings
- **The plan file still describes the replaced design.** In `workshop/plans/000830-…-plan.md`:
  - Decisions 6 and 7, T5 and T6 still describe a switcher wake key, `couch --asleep --json` as ariadne#564's contract, an `operationdispatch` `wake` op, `Couch.WakeHubs`, and a resume fan-out bounded by `LifecycleParallelism`.
  - T1 says the cause is validated "in enum", but `sleep.go:709` accepts any non-empty cause of 64 characters or fewer. That is deliberate, so a newer build's cause does not make this build refuse the record.
  - Every task checkbox is unchecked, and the file has no `## Revisions` section. The revisions live only in the issue file.

  Anyone reading the "durable plan", for example ariadne#564's owner, would code against an `--asleep --json` that doesn't exist. Fix: add a `## Revisions` entry to the plan (see section 7) and tick the tasks.
- **`## Done when` no longer matches the Spec.** Bullet 2 promises a pair test of the hubs waking their workers "with ariadne#564's resume". That work lives in another repo. What pair delivers is `asleep_at` in the recover-plan rows, pinned by `TestRecoverPlanCarriesTheAsleepMark`. Bullet 3, "survives a full restart and clears on wake", is only shown piece by piece: the record round trip, the `ThreadAsleep` table, and `ClearVerifiedPark`. The one test that would show it end to end, `TestDisposableCouchFleetWake`, is skipped until pair#863 (`fleet_acceptance_test.go:296`). Fix: reword Done-when so the wake is the main hub's `couch --wake` (rig test), the worker ordering is ariadne#564's against the `asleep_at` contract, and restart survival is unit-proven now with the native proof waiting on #863.

### 4. Minor findings
- **The resume does not re-check that the hub is still asleep** (`wake.go:331`). If the operator plain-parks the hub between admission and the resume running, wake resumes a slot that is now plain parked. The window is narrow (ARCH-ORDER).
- **One 1.5 s budget covers the whole wake** (`wake.go:287`). Authentication, the full snapshot read and N admissions all share `slotAdmissionBudget`. With a large store, later hubs get "not admitted" with a deadline error. That failure is visible and a rerun is safe, but nothing measures it (ARCH-CONSTRAINTS).
- **The rig fakes the asleep set** (`wake_test.go:520`). Until T6 runs, nothing in the portable suite checks the production chain: `Couch.AsleepSlots`, then `Target.Slot.WorktreeRoot`, then `HubOwnership.IsHub`.
- **The `next.Asleep = nil` in `AdvanceStartTransaction` has no test** (`starttransaction.go:164`). It is tidy-up only, since `ThreadAsleep` already reads false without a receipt, so no test fails if it is removed.

### 5. Test coverage notes
- **Targeted runs pass:** the `ThreadAsleep` table, the additive round trip, Leave marking and Recover, `asleepSlotRows`, the `Wakes` table, the recover-plan `asleep_at` row, the refusal enumeration (including `BeginParkAsleep`), the switcher text, wake request validation, and the wake CLI parse, summary and poll tests.
- **Wider runs fail in this sandbox only:** the full couchcore, couchcmd and testcouch runs fail where `git init` is refused under the review sandbox (sdlc #339). I sampled those failures, and every one I read was that refusal, unrelated to this diff.
- **Mutation checks not repeated:** I didn't re-run the implementer's mutation checks. The rig's exact admitted list (`resume bench:0,resume third:0`) would catch removing the target filter or the main-hub check.

### 6. Architectural notes for upcoming work

| Principle | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `pollSlotOperation` is shared; wake reuses resume. |
| ARCH-PURE | pass | `ThreadAsleep`, `asleepSlotRows`, `Wakes` and `wakeSummary` are pure; the IO stays in `handleWake` and `runWakeCLI`. |
| ARCH-PURPOSE | pass | The worker-side ordering properly belongs to ariadne#564, and the contract it reads is delivered. |
| ARCH-MOCK | pass | No new external dependency. The native T6 waits on #863. |
| ARCH-CONSTRAINTS | pass with note | The shared admission budget above. |
| ARCH-SECURE | pass | Strict decode, a capped cause length, and authority checked at admission and again at execution. |
| ARCH-ORDER | pass with note | The nonce-bound derived flag is strong; the minor gap is the missing still-asleep check at resume time. |
| ARCH-FUNERAL | pass | One mark per record, cleared on resume or reattach, replaced by the next begin, gone at archive. |
| ARCH-EVOLVE | pass | Top-level additive, nested members preserved, tolerant cause, `asleep_at` omitted when empty, and an old Couch answers `--wake` with a clear "restart on the new binary" message. |

When #722 (drain) arrives it should pass its own `SleepCause`; the `parkJoined` parameter is already in place for that. When #863 lands, remove the T6 skip and run it before treating the restart and clear-on-wake bullet as proven.

### 7. Plan revision recommendations
Add to the plan file:

```
## Revisions
- 2026-10-23/24: superseded by the issue's Revisions. No switcher wake key, no
  `couch --asleep --json`. Wake is the broker op `wake` (`couch --wake`), main hub
  only (`HubOwnership.Wakes`), which admits one ordinary resume per other asleep
  hub. ariadne#564 reads `asleep_at` from the `--recover-plan-from-sdlc` rows.
  SleepMark accepts any non-empty cause (64 characters or fewer) rather than an
  enum, so a newer cause does not make this build refuse the record. The T4
  counted read test was dropped (no new read). T6 is committed but skipped until
  pair#863.
```

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: plan-doc-tracks-revisions
    title: |
      Plan file still describes the replaced design (switcher key, couch --asleep --json, WakeHubs); no Revisions, all tasks unchecked
    detail: |
      workshop/plans/000830-...-plan.md Decisions 6-7 and T1/T5/T6 contradict the code: the code has no --asleep verb or wake key, and the cause is validated as non-empty, not an enum. The revisions live only in the issue file. Add a plan-level Revisions entry and tick the tasks so ariadne#564 does not code against a contract that does not exist.
  - id: new
    severity: Important
    kind: defect
    family: done-when-tracks-spec
    title: |
      Done when is stale after the Spec revisions: claims a pair test with ariadne#564's resume and an untested restart/clear-on-wake
    detail: |
      The wake is now the TL's couch --wake from the main hub. Pending-first worker waking is ariadne#564's, and pair only delivers asleep_at in the recover-plan rows. The end-to-end restart/clear test (TestDisposableCouchFleetWake) is skipped until pair#863. Reword Done when to match what this repo delivers and proves.
  - id: new
    severity: Minor
    kind: defect
    family: effect-gate-reproves-admission-facts
    title: |
      Wake's effect gate re-checks caller and hub-ness but not that the target is still asleep
    detail: |
      A hub plain-parked between admission and the resume running would still be resumed by wake (wake.go:331-344).
  - id: new
    severity: Minor
    kind: defect
    family: admission-budget-covers-fanout
    title: |
      handleWake spends one 1.5s slotAdmissionBudget on the snapshot read plus every per-hub admission
    detail: |
      On a large store, later hubs come back not admitted with a deadline error. The failure is visible and a rerun is safe, but it is unmeasured.
  - id: new
    severity: Minor
    kind: defect
    family: integration-composition-untested
    title: |
      No portable test checks the production chain from Couch.AsleepSlots to the hub-root match in handleWake
    detail: |
      The wake rig fakes authority.asleep. The real snapshot-to-Target.Slot.WorktreeRoot-to-IsHub path is only covered by the skipped native T6.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1d45c4b8-ed93-48c6-9aa9-a12aa3a09dfd","repository":"github.com/xianxu/pair","issue_id":"000830","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/63f6e911ab27370f","reviewed_head":"a520f3a5af36b66ca0a612aee8deaa95f1a0ec34","slot":"pair:6","requested_at":"2026-10-24T11:01:18.152032Z"},"phase":"finished","pid":90665,"started_at":"2026-10-24T11:01:18.435683Z","finished_at":"2026-10-24T11:23:12.90944Z","session_id":"5f2a0e49-bf34-4b2b-925c-ea4a8f9e4e26","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 40.91312675s
head: "a520f3a5af36b66ca0a612aee8deaa95f1a0ec34"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T04:50:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 830 — Global Alt+x marks parked slots 'asleep'; one 'wake all' brings back exactly those |
| repo | pair |
| issue file | workshop/issues/000830-global-alt-x-marks-parked-slots-asleep-one-wake-all-brings-back-exactly-those.md |
| boundary | whole-issue close |
| milestone | — |
| window | aafc7c8c08f9d49b950b38265beecc9ed569b1b2..6fbb5568d24833bd4ae5c0d965845e5a38adfaab |
| command | sdlc close --issue 830 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-24T04:50:20-07:00 |
| verdict | SHIP |

## Review

Most targeted tests pass. The couchcore failure comes from the sandbox refusing a `git init`. Next I'll check which couchcore tests fail, then revert the gate fix to confirm its test goes red.

Prior fixes check out: the gate mutation makes the new test fail. I'm doing one last check of the archive/restore path for the asleep mark.

```verdict
verdict: SHIP
confidence: high
```

The two Important findings from round 1 are fixed. BR-1: the plan now has a `## Revisions` section that records the move from the wake key and `--asleep` to `couch --wake`, and its tasks are ticked. BR-2: Done when now describes only what this repo delivers, and says plainly that the native T6 test is skipped until pair#863. The Minor gate gap (BR-3) is also fixed, and its regression test is real. I removed the asleep re-check from the guard in `wake.go`, and `TestWakeEffectGateRefusesAHubNoLongerAsleep` failed; with the check restored, the `Wake` tests pass. BR-4 (admission budget) and BR-5 (a portable test of the real chain from `AsleepSlots` to `handleWake`) are untouched and stay open as Minor. I found one new Minor issue: archive and restore bring back the asleep mark, which contradicts a row in the plan's state table. Nothing blocks.

**1. Strengths**
- The derived, nonce-bound predicate (`couchcore/asleep.go:17-25`) is a sound way to make stale marks inert. Every path that ends a park, and an older build's rewrite, leaves a mark that no longer matches. `TestThreadAsleepIsTheMarkOfTheParkInForce` lists every case.
- The mark is written in the same CAS as the begin (`threadstore.go` `beginPark`/`BeginParkAsleep`), so it is durable before teardown starts, as Decision 1 requires.
- `SleepMark` keeps unknown members it doesn't recognise, and the record's new member is top-level, so older builds keep it. Validation accepts any non-empty cause, so a future drain cause won't make this build refuse the record (`threadrecord/sleep.go:68-80`). That is ARCH-EVOLVE done right.
- `pollSlotOperation` was pulled out of `exchangeSlotOperation` and is shared with `--wake` (`messages.go:444`) rather than copied (ARCH-DRY).
- The wake reuses the ordinary resume queue with an effect gate, instead of a new core operation.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **Archive then restore makes a slot asleep again.** `RestoreThread` (`couchcore/retention.go:230`) puts back the archived bytes exactly. Neither archive nor restore clears `Asleep` or `VerifiedPark`, so a restored slot still has a matching nonce and reads asleep. If that slot is a hub, the next `couch --wake` resumes it. The plan's state table says "unarchive → parked". Fix: either clear the mark when archiving, or change the table row to "unarchive → asleep". (I read this from the code; I did not run it.)
- BR-4 and BR-5 are still open (see dispositions).

**5. Test coverage notes**
- The `Wake` tests in `couchcmd` and the `couchcore` asleep, sleep and recover-plan tests pass, as do `threadrecord`, `couchmessage` and `couchtty`.
- The full `couchcore` package fails here. Each failure I traced (TestRecoverPlanAfterRestart, TestRebootLeavesSessionSharedWithAnotherLiveSlot) comes from the review sandbox refusing `git init` ("outside the reviewer's directories", sdlc #339). Those failures say nothing about the diff, but it means I could not see the full package pass.
- The mark clearing on reattach and on rollback-then-retry is tested (`asleep_test.go` Leave tests). I did not find a direct test that `StartRegistered` clears it; the predicate covers that case through "mark without receipt".
- The native end-to-end test `TestDisposableCouchFleetWake` is skipped until pair#863, and Done when says so.

**6. Architectural notes**
- **ARCH-DRY:** pass. The poll loop is shared, and the resume queue is reused.
- **ARCH-PURE:** pass. `ThreadAsleep`, `asleepSlotRows` and `HubOwnership.Wakes` are pure and tested directly. `handleWake` is the IO glue.
- **ARCH-PURPOSE:** pass. Mark, display, recover-plan field and wake are all delivered. Worker ordering belongs to ariadne#564, as the Spec states.
- **ARCH-MOCK:** pass. No new external dependency; the rig uses the existing service and queue fakes.
- **ARCH-CONSTRAINTS:** pass. The one unmeasured part is the shared budget, which is BR-4.
- **ARCH-SECURE:** pass. The mark is validated by the strict decode, and the caller's authority is checked again at the effect gate.
- **ARCH-ORDER:** pass. Asleep is derived rather than held as a separate flag. The gate's re-check is tested by injecting a change between admission and the gate.
- **ARCH-FUNERAL:** pass. Removal is covered by the nonce binding and the clearing sites. The restore gap above is the one exception.
- **ARCH-EVOLVE:** pass.

**7. Plan revision recommendations**
- Fix the state-table row "Asleep record archived → unarchive → parked" so it matches restore's actual behaviour, or change the code to match it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan now has a Revisions section covering the move from the key and --asleep to couch --wake, the string cause, and the skipped T6; tasks ticked (6fbb5568 plan diff).
  - id: BR-2
    disposition: addressed
    note: |
      Done when now names couch --wake from the main hub, gives worker ordering to ariadne#564, and states the T6 skip until pair#863; matches wake_test, asleep_test and recoverplan_test.
  - id: BR-3
    disposition: addressed
    note: |
      wake.go:84-93 re-reads the asleep set in the guard; TestWakeEffectGateRefusesAHubNoLongerAsleep goes red when the check is reverted (verified in the review checkout) and green when restored.
  - id: BR-4
    disposition: not-addressed
    note: |
      handleWake still wraps the snapshot read and every per-hub admission in a single slotAdmissionBudget (wake.go:23); still unmeasured. Minor.
  - id: BR-5
    disposition: not-addressed
    note: |
      The wake rig still fakes authority.asleep; no portable test runs Couch.AsleepSlots through handleWake's hub-root match. Minor.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: asleep-mark-retired-by-every-park-exit
    title: |
      Archive then RestoreThread brings the asleep mark back, contradicting the plan's "unarchive → parked"
    detail: |
      RestoreThread (couchcore/retention.go:230) restores the archived bytes exactly, and nothing clears Asleep or VerifiedPark, so a restored record reads asleep and a restored hub is resumed by the next couch --wake. Either clear the mark when archiving, or change the plan's state-table row to unarchive → asleep.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9dc68a6a-87c5-4238-9eb6-b11b5f6c628e","repository":"github.com/xianxu/pair","issue_id":"000830","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/bacc870bae2c8212","reviewed_head":"6fbb5568d24833bd4ae5c0d965845e5a38adfaab","slot":"pair:6","requested_at":"2026-10-24T11:32:00.361707Z"},"phase":"finished","pid":58990,"started_at":"2026-10-24T11:32:00.520487Z","finished_at":"2026-10-24T11:49:46.298162Z","session_id":"97a74949-9a5d-4362-97ff-61564a6b79b1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.923682417s
head: "6fbb5568d24833bd4ae5c0d965845e5a38adfaab"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
