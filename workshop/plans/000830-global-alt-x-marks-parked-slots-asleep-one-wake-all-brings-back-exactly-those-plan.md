# #830 Asleep slots: a full quit marks what it parks; one wake brings back the asleep hubs — Plan

> **For agentic workers:** single pass, plain checkboxes, one `sdlc close`. TDD each task.

**Goal:** A full quit (Alt+x; any later shutdown that goes through `Leave(LeavePark)`, such as #722's drain)
marks each slot it parks as *asleep*. An individual park, and a slot already parked, stay plain *parked*. The
switcher shows asleep distinctly. One explicit wake gesture resumes **only the asleep hubs**; tl-lead (ariadne#564)
reads the asleep set to wake its workers, pending work first. The mark survives a restart and clears on wake.

**Architecture:** the mark is written in the **same CAS as `BeginPark`**, bound to that park's nonce. "Asleep" is
then *derived*: `VerifiedPark != nil && Asleep != nil && Asleep.Nonce == VerifiedPark.Identity.Nonce`. No new
thread state; asleep annotates parked. Wake reuses the existing resume path per hub.

## Decisions (with the alternatives rejected)

1. **Write the mark at BeginPark, not after the verified park.** A second write after the park can fail after the
   agent is already down: the slot would read plain parked and wake would silently miss it. Writing it at begin
   makes it durable before any teardown. `Recover`, run by this Couch or by the next start, then completes under
   the same nonce, so the mark survives a crash mid-park with no extra step (ARCH-ORDER).
2. **Bind to the nonce, derive the flag.** Every path that ends or replaces a park also invalidates the mark,
   without having to remember to clear it:
   - resume's `StartRegistered` clears `VerifiedPark`;
   - `ClearVerifiedPark` (a warm reattach);
   - a new individual `BeginPark` gets a new nonce;
   - rollback of an unpublished park never sets `VerifiedPark`.

   Those sites also clear `Asleep` (hygiene, ARCH-FUNERAL). But the derived predicate is the authority, so a site
   that forgets, or an **older build** that rewrites the record and keeps the field as Unknown (pair#695), cannot
   resurrect a stale mark.
3. **Top-level record field, not a field inside `VerifiedPark`/`ParkTransaction`.** Those nested types are strict.
   A newer field inside them would make an older build **refuse** the record. A top-level member is preserved
   verbatim by older builds (`threadrecord.Record.Unknown`). The nested `SleepMark` is *born additive*: it keeps
   its own unknown members, like `TranscriptBinding` (ARCH-EVOLVE).
4. **Who marks:**
   - Leave's **fresh** drives pass `ParkCauseSleep`, including the fresh retry after a proven rollback.
   - Leave's **Recover** drives add nothing. An open transaction keeps whatever its `BeginPark` wrote: an
     interrupted full quit keeps its mark, and an operator's individual park left open stays plain.
   - `Park`/`ParkExpected` (individual) pass `ParkCausePlain`, which writes `Asleep = nil`.
   - The cause is an explicit Leave-level parameter, so #722's drain chooses it rather than inheriting it by
     accident.
5. **Asleep is an annotation, not an `ActionableThreadState`.** Every resume, switch and archive reader already
   accepts parked (`TestEveryParkedProducerIsAcceptedByResumeSwitchAndArchive`). A new state would have to be swept
   through every one of them for no behavioural gain (ARCH-DRY). The summary gains `AsleepAt *time.Time`. Rows
   render:
   - a parked row as `asleep · since 23:41`;
   - any other state with ` · asleep` appended (e.g. a session found lost: wake will say so).
6. **Wake = resume the asleep hubs, nothing else.** "Hub" means `HubOwnership.IsHub` for this Couch's command line.
   - It runs the existing `ResumeTarget` per hub, bounded by `LifecycleParallelism`, and reports every hub's
     outcome in one summary (the shape of `FormatLeaveParkSummary`).
   - It is idempotent: a woken hub is no longer asleep, so pressing again retries only the failures.
   - Workers are never touched; their order is tl-lead's (ariadne#564).
   - pair#739's no-automatic-resume rule holds: nothing wakes on start.
7. **One core operation, two surfaces (ARCH-DRY):** a `wake` operation in `operationdispatch`:
   - the switcher key (asks first, listing the hubs it will wake);
   - the CLI `couch --wake [--json]`, which forwards it to the live owner like other operations.

   The read surface for tl-lead is `couch --asleep --json`:
   - It returns `[{"address":"pair:3","asleep_at":"…","hub":false,"cause":"full_quit"}]`.
   - It reads the store only (cost class light, no per-slot probe: deterministic commands answer from memory).
   - It is the contract ariadne#564 consumes instead of guessing from issue ownership.

## State design table

**Actions:** full quit (Alt+x), individual park, wake (key or `--wake`), resume one slot (switcher Enter or
`--resume`), archive, quit again.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Full quit's fresh park verified (mark nonce = receipt nonce) | asleep | wake (hub) / tl-lead resume (worker) | none |
| Full quit's park failed, rolled back (mark set, no receipt) | live (quit incomplete summary, #827) | quit again | none (mark inert) |
| Full quit interrupted mid-park (open txn, mark on its nonce); next start's Recover completes it | asleep | wake | none |
| Same, Recover cannot complete (open txn) | parking/unresolved (existing #827 advice) | the advice's recovery, then quit again | mark stays inert until a receipt matches |
| Slot already parked at quit time | parked | resume it individually | none |
| Individual park (plain cause, `Asleep = nil` in the same CAS) | parked | resume it individually | none |
| Asleep hub, wake's resume succeeds (`StartRegistered` clears the receipt and the mark) | live | n/a | none |
| Asleep hub, wake's resume fails before registration (receipt and mark intact) | asleep, with the failure in the wake summary | wake again (only failures retry) | none |
| Wake's resume outcome unknown (Couch dies mid-wake) | asleep, or busy (existing start claim) until `reconcileInterruptedStarts` decides | wake again after start-up reconcile | none: the receipt survives until registration |
| Asleep slot resumed individually / warm reattach | live | n/a | none (mark cleared) |
| Older build resumes an asleep slot (clears the receipt, keeps the `asleep` member as Unknown) | live; on a later park: parked | n/a | none: the nonce mismatch makes the leftover inert |
| Asleep record archived | archived (no asleep) | unarchive → parked | the asleep fact (by design, archive retires it) |
| Second actor wakes the same hub concurrently | one resume wins (`holdWait`); the other reports "already awake" | n/a | none |

**Covered:** every path that ends a park, crash mid-quit, crash mid-wake, older builds, concurrent wake.
**Not covered:** a slot whose session is lost while asleep. Wake reports the existing resume refusal (reboot
advice); it does not reboot on its own. That is pair#739's rule, and a reboot is the operator's call.

## Operating envelope (ARCH-CONSTRAINTS)

- **BeginPark write:** one more small member in an existing CAS. No new IO; the #827 commit timing is unchanged.
- **Switcher render:**
  - Cost: O(1) per row, from the record already loaded for the summary (growth driver: rows).
  - Counted test: summary building reads each record once at N=8 and N=16.
- **`--asleep`:** one store snapshot, O(threads). Light class.
- **Wake:**
  - Bound: ≤ `LifecycleParallelism` concurrent resumes; hubs are 1–3 in practice.
  - Each resume keeps its existing bounds; the wake returns after every admitted resume settles.

ARCH-SECURE: the mark is persisted input. It is decoded through threadrecord's strict typed decode, and a bad mark
refuses the record exactly as any bad member does. No credentials. ARCH-MOCK: no new external dependency; tests
use the existing store and lifecycle fakes and the `testcouch` native harness.

## Tasks

- [x] **T1 Record field.** `threadrecord.SleepMark{Nonce, At, Cause}` (born additive) plus `Record.Asleep`;
  mirrored in `couchcore.ThreadRecord`, with the clone and convert sites; validation (nonce non-empty, cause in
  enum).
  - Tests: round trip; unknown member inside the mark preserved; older-shape rewrite keeps the top-level member;
    a bad cause refuses.
- [x] **T2 Pure predicate.** `ThreadAsleep(record) (time.Time, bool)` with an exhaustive table: no receipt; receipt
  with no mark; nonce mismatch; match.
- [x] **T3 BeginPark cause.** `BeginPark(address, rev, identity, cause)` writes or clears the mark in the same CAS.
  `StartRegistered` and `ClearVerifiedPark` clear it. Thread `ParkCause` through `parkAwaiting`/`park`/`parkExpected`;
  `Park`/`ParkExpected` pass plain, `parkJoined` takes the cause, and Leave passes `ParkCauseSleep` on fresh drives
  only.
  - Tests: Leave over the `fleetLifecycle` fake marks fresh parks asleep, leaves already-parked and
    open-individual-park slots plain, and keeps the mark through rollback then fresh retry (new nonce, new mark).
    An individual park after a failed quit is plain. Recover keeps an interrupted quit's mark.
- [x] **T4 Summary + switcher.** `ActionableThreadSummary.AsleepAt`; the `threadStateText` rendering above; the
  counted read test.
- [x] **T5 Wake operation.** `Couch.WakeHubs(ctx) (WakeResult, error)`: snapshot, filter asleep ∧ hub, bounded
  resume, summary.
  - Dispatch op `wake`; switcher key with a confirm frame; `couch --wake [--json]`; `couch --asleep --json`;
    cost-class entries; help text.
  - Tests: a fake with 2 asleep hubs, 2 asleep workers and 1 parked hub wakes exactly the 2 hubs. A failure is
    reported and retried by the second wake.
- [x] **T6 Native fleet acceptance.** Extend `TestDisposableCouchFleetFullQuit`, or add a sibling, at 4 slots, on
  Claude and Codex:
  1. individually park one worker, then Alt+x;
  2. restart Couch, then `--asleep --json` lists the hub and live workers but not the individually parked one;
  3. wake: the hub is live, the workers still asleep, the parked one still parked;
  4. one worker resumed, then the flag is cleared.
- [x] **T7 Docs.** `atlas/couch.md` (asleep, wake, `--asleep` contract), `atlas/index.md`, lessons; tell ariadne#564's
  owner the `--asleep --json` contract via the hub.

## Revisions

- **2026-10-23 (TL-approved, first revision):** no switcher wake key and no `couch --asleep --json`. The asleep set reaches ariadne#564 as `asleep_at` on each thread row of `couch --recover-plan-from-sdlc` (additive, schema 1). Decision 7's `--asleep` surface is dropped.
- **2026-10-23 (operator, via TL):** waking is **`couch --wake`, run from the live main hub**, not a key. Alt+Shift+x is removed (Open question 1 closed: no key). `couch --wake` is a broker op, and only the main hub may call it (`HubOwnership.Wakes`). It reads the asleep set from the store (`Couch.AsleepSlots`) and admits one ordinary slot resume per other asleep hub, each through an effect gate that re-proves the caller, that the target is a hub, and that the target is still asleep (close review). The CLI polls the receipts to one summary. There is no `Couch.WakeHubs` core operation: the broker handler composes the existing resume queue (ARCH-DRY). Workers are never woken here; each woken hub's tl-lead resume (ariadne#564) wakes its own from `asleep_at`. Decision 6 stands; Decision 7's "two surfaces" became this one. Open question 2 (`--all`): not done.
- **2026-10-23:** the mark's cause is a non-empty string validated at write time, not an enum (T1).
- **2026-10-24 (TL):** T6 is written as `TestDisposableCouchFleetWake` (the harness gains `Options.Hubs`, `StartCold` and `QuitConsole`) and skipped until pair#863 lands. Since #739, an explicit-hub fleet can't start through `Fleet.Start`. It drives the operator's flow instead: start each hub from the switcher, full quit, cold start, Resume from disk, then `couch --wake`.

## Open questions for the TL / operator

1. **The wake key in the switcher.** Alt+w is Pair's. I propose a switcher-scope **Alt+Shift+x**, "wake the asleep
   hubs (asks first)", the mirror of Alt+x. The alternative is a plain `w` in the switcher. Operator's call.
2. **Should `--wake` also take `--all`** (hubs and workers, for an operator without a TL)? I suggest not now:
   tl-lead owns worker order, and a later flag is additive.
