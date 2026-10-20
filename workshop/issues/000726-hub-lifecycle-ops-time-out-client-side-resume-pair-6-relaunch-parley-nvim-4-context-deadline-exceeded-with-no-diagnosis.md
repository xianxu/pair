---
id: 000726
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.77
collaboration-mode: collaboration
card_mirror: '6e7b5dcef1ade36f1489b24fac1fa2a0020099bd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T07:54:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.53
---

# Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis

## Problem

After the 10-20 07:49 Couch restart (supervisor on pair:0 staged f3c597f8, which includes #714 and #721), slot
lifecycle operations from the hub CLI end in a client-side timeout with no diagnosis:

- `couch --resume pair:6` → "context deadline exceeded … outcome uncertain" (4 times since 10-19 23:00, across
  three supervisor builds). `--resume-candidates pair:6` says: "Previous screen unavailable: lstat
  …/scrollback-1-pair-59-codex.raw: no such file", "native transcript incomplete: session inventory read exceeds
  limit", "Candidate scan incomplete", yet still offers "Resume from disk (confirmed)". `--reboot pair:6 --confirm`
  also timed out (10-19 23:05).
- `couch --relaunch parley.nvim:4 --confirm` (idle, load ~16) → "context deadline exceeded"; the slot was NOT
  relaunched (wrapper Start still 10-19 20:12).

The CLI's deadline expires while the supervisor-side operation is presumably still working (or wedged), and the user
gets "verify with couch --recover-plan-from-sdlc", which itself fails to resolve the fleet from a non-git cwd.

## Spec

- Find where the lifecycle operation spends its time (pair:6: likely the native inventory read that "exceeds limit";
  parley.nvim:4: unknown) and bound it inside the supervisor, so the CLI gets a typed outcome (refused with reason,
  or running with an operation id to poll) instead of a client deadline.
- `--resume-candidates` must not offer "Resume from disk (confirmed)" when the resume path will fail on the same
  oversized inventory; either the resume path tolerates it (the binding is established) or the menu says why not.
- `--recover-plan-from-sdlc` must work when run from a non-repo cwd (or say which directory to run it from).

## Done when

- A fixture with an oversized native inventory: resume of an established binding completes or refuses with a reason
  within the CLI budget (test). A relaunch of an idle slot completes within the budget (test with a stateful fake).
- pair:6 resumes (or refuses with a reason) live after the fix.

## Plan

### Diagnosis (2026-10-20, pair:2, measured)

- **Both failures are the admission call, not the operation.** `couchmessage.AdmissionTimeout` is 2 s per exchange (`couchmessage/broker.go`, `transport.go`). The CLI's `exchangeSlotOperation` admits once, then polls `operation-status` for up to 5 minutes. "context deadline exceeded … outcome uncertain" can only come from the first admit exchange (`couchcmd/messages.go`). When the handler finishes late, `serve` drops the reply without writing anything, so the CLI cannot tell whether anything was queued.
- **Admission runs the hub guard synchronously** (`handleHubSlotOperation` → `guard`, `hub_guards.go`). The guard resolves the target through `WorkspaceReferencePath` → `OSSlotCatalog.Discover`: `sdlc` on the primary, `git worktree list`, `git for-each-ref`, then per slot `sdlc workspace` plus 2× `git rev-parse`, all serial and uncached.
- **Measured** on this machine (load 12–18), calling the real `Discover` on the live repositories: pair takes **3.0–3.4 s** (27 commands, 8 slots), and parley.nvim takes **2.6–2.7 s** (15 commands, 4 slots). Discover alone is over the 2 s budget, so every hub lifecycle op on these repositories fails admission, whatever the target's state. That covers both the pair:6 resume and the idle parley.nvim:4 relaunch.
- **The admitted job then repeats the guard twice** (queue prepare, and the hub effect gate). For `--resume` and `--reboot` it runs the full native recovery scan (`executePreparedRecovery` → `observe` → `CollectRecoveryEvidence`) even for a proven resume, which uses only the branch, issue, screen and capture fields. That scan walks all of `~/.codex/sessions` (6.6 GB, about 3,900 files) or `~/.claude/projects` (about 11,000 entries, over the 10,000-entry cap), so it always ends incomplete. It reads matching transcripts in full at 64 KiB per reopen, up to 30 s, while holding the one-wide hub effect gate. pair:6's root transcript is 56.8 MB. This is the next bottleneck once admission passes, and the source of "session inventory read exceeds limit".

### Design (approved by the hub; revised for plan review round 1)

- [x] M1 — admission returns a typed outcome inside the budget
  - **Warm sources for every admission input, so a warm admission spawns no process (PQ-1).**
    - *Caller workspace:* the supervisor's existing per-connection identity, `connectedWorkspace` (`s.workspaces`, set at connect). `authorizeParkAdmission` already uses it (ARCH-DRY). Admission no longer calls `authority.workspace` (that is `ResolveWorkspace` → `sdlc`).
    - *Target slot:* a memo `hubTargetMemo`: canonical target reference → `HubSlot` plus the time it was resolved. Only the existing fresh resolver (`hubTargetResolver`: `WorkspaceReferencePath` → `Discover`, then `ResolveWorkspace`) fills it. Every fresh resolution refreshes the memo, including those in queue prepare and at the effect gate. Admission reads an entry younger than its bound (10 minutes; slot topology changes rarely).
    - *Caller liveness and thread:* `liveCaller` and `authority.thread` read the store and recorded files and spawn nothing. The counted test proves it.
  - **Authority is still proven fresh before any effect.** The queue's prepare step and the hub effect gate keep running the full guard on fresh discovery, as today. A stale memo can at worst admit a job that then refuses with a reason, before any effect.
  - **Cold or expired memo (PQ-2).** Admission starts one fresh resolution for that reference. It is single-flight per reference, runs under the supervisor's lifetime context, and is joined at shutdown. Admission waits for it up to the admission budget. If the resolution finishes in time, admission uses it. Otherwise admission refuses ("slot catalog is warming; nothing was queued; retry in a few seconds"), the resolution keeps running, and its result fills the memo. Test seam: the resolver is injected, so a test can complete a resolution after the refusal it caused and see the next admission admit.
  - **Admission budget.** Slot-op admission runs under its own budget, below the transport's 2 s. On overrun it answers `refused`, naming the phase (caller, thread, target, queue), never silence.
  - **`hubReceiptFilter`** (receipt listing and polling) reads the same memo, so `operation-status` polling spawns nothing.
  - **Discover probes concurrently (PQ-3).** The per-slot `inspectSlot` calls run concurrently, at most `MaxSlotCandidates` at a time, and the results keep their order. Each row keeps its before/after `os.SameFile` replacement check, and any row error still fails the whole inventory, as now.
- [x] M2 — a proven or fresh resume does not scan native candidates (boundary confirmed by pair:3, the recovery owner)
  - For a direct Proven or Fresh selection, `OSRecoveryEvidenceReader.ReadRequest` skips only the native candidate collection (`NewNativeTranscriptRuntime` / `CollectRecoveryEvidence`). It keeps the whole prefix that builds `Source` (`recovery_evidence_os.go`): retained source, producing-launch witness, activity, capture expiry, screen error and an explicit `request.Source`, including the expiry ScreenUnavailable handling. It also keeps `Agent`, `Generation`, the capture identity, byte caps, freshness and validation, and the Source-nil `validateRecoveryCaptureSnapshot` check. Ranking, native Incomplete and native Reasons are not consumed on this path. The inferred (keyed) path is unchanged, even when it later chooses Fresh. Both proven-binding checks, the ownership checks and the retirement checks stay.
  - `--resume-candidates` keeps offering "Resume from disk (confirmed)" (the binding is established) and says that the confirmed choice comes from the established binding, not from the scan. The wording holds whatever made the scan incomplete; revised in M2 review round 1.
- [x] `--recover-plan-from-sdlc` from a directory outside any repository: does not reproduce. Closed as (a), and (b) filed as pair#729 (see the Log).

Tests (PQ-4: function under test, then strategy):
- `handleHubSlotOperation` admission: counted ProvisionIO and slot-catalog fakes. A warm memo plus a connected caller spawns zero `git` or `sdlc` commands and admits (the hub's requirement). A cold memo with a resolver slower than the budget refuses inside the budget, with nothing queued, and names the phase. Then the injected resolution completes, and the next admission admits.
- Queue prepare with a stale memo: the fake catalog moves the target to another hub after the memo was filled. Admission admits, and prepare refuses with a reason before any effect.
- `OSSlotCatalog.Discover`: a fake ProvisionIO with per-command latency, at N and 2N slots. Count the commands (unchanged) and the peak concurrency (bounded). A row replaced mid-inspection still fails the whole inventory.
- `ReadRequest` and `executePreparedRecovery` Proven/Fresh (pair:3's matrix): zero transcript reads, counted, on a fixture over the entry cap with an oversized transcript. `Source`, `Generation` and the orientation are identical to today's for valid, expired, missing and retained-source screens. A missing capture never turns into a new Preserve attempt only because the source was omitted. Inferred still scans and reports incomplete.
- `--recover-plan-from-sdlc` from a non-repository directory.

Non-goals (PQ-5): `serve` keeps dropping late replies for other ops; `AdmissionTimeout` stays 2 s; the 5-minute poll budget stays; the native scan's own limits are unchanged.

State design (M1). Actions: run a lifecycle op, `--operation-status ID`, retry.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Warm snapshot, guard passes | accepted with an operation ID, polled to an outcome | none | none |
| Cold or expired snapshot, refresh finishes in budget | accepted as above | none | none |
| Refresh exceeds the admission budget | refused: "slot catalog is warming; nothing was queued; retry" | retry after a few seconds | none |
| Snapshot stale (slot moved), admitted | prepare re-proves on fresh discovery and refuses with a reason | fix ownership, rerun | none (no effect ran) |
| Any other admission overrun | refused, naming the phase that overran | retry; report if persistent | none |

ARCH notes: ARCH-ORDER (fresh authority before the effect; admission is a cheap gate); ARCH-CONSTRAINTS (Discover cost is linear in slots and now concurrent; admission does zero probes when warm); ARCH-MOCK (stateful catalog fake with a clock); ARCH-FUNERAL (the snapshot has a bounded age and is replaced, never accumulated).

## Revisions

- 2026-10-20 (pair:2): M1 review rounds 1 and 2 changed behavior, so the plan design is amended here:
  - A failed resolution answers admission with its own error for 30 s (`hubTargetFailureAge`), then retries.
  - Move-slot admission names its slots from the memo within the budget. The queued move re-resolves each one fresh and refuses on any difference, because those slots are the move's payload.
  - The receipt filter reads the memo within the admission budget.
  - Resolutions are stamped with their start time, so an older one never replaces a newer one.
  - Entries are swept when expired and capped at 256.
  - At most 8 background flights and 8 concurrent Discover rows.
  - M2: the "counted reads" of the Tests bullet are the reader's own `BytesRead` (native bytes) compared with a screen-only read; the routing test shows that Proven and Fresh set the flag.
  - Entries keep their start time for ordering and their finish time for expiry. `Warm` bounds its own wait by the admission budget (round 3).
  - The concurrency test proves overlap with a rendezvous of in-flight slot probes, never with elapsed time (the hub's flake rule: deterministic, no timeouts).
  - State table additions: "Cold resolution failed after the budget" shows as refused with that error for 30 s, recovered by fixing the cause and retrying, with no loss. "Move admitted on a stale slot" shows as the queued move refusing ("resolved differently since admission; nothing was moved"), recovered by retrying, with no loss.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.3 impl=0.24
item: smaller-go-module      design=0.1 impl=0.14
item: smaller-go-module      design=0.2 impl=0.14
item: smaller-go-module      design=0.05 impl=0.12
item: milestone-review       design=0.0 impl=0.14
item: milestone-review       design=0.0 impl=0.14
item: atlas-docs             design=0.05 impl=0.04
design-buffer: 0.15
total: 1.77
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Rows: M1's admission memo, warm sources and budget (greenfield, single concern); M1's concurrent Discover (smaller, mirroring the existing code); M2's proven-resume scan skip (smaller, but a subtle boundary, so more design); the recover-plan working-directory fix (smaller); two milestone reviews; atlas. Impl values are 40% of the v2 midpoints. The plan doc is thorough, so the design buffer is 15%. The calibration doc is flagged stale (#127), so the numbers are provisional.

## Log

- 2026-10-20 (pair:2, Claude): CHECKPOINT for a cold Codex handoff (the operator is moving Claude slots to Codex).
  - State: #726 was closed (FIX-THEN-SHIP, reconciled; card codecomplete), and PR 487 is open with CI green. It has not landed: the train held it because head 44c1e2e3c had no recorded suite, and the queue was paused for the Couch reboot. Then the hub added scope after this morning's recovery dropped 9 of 14 CLI resumes ("context deadline exceeded").
  - Hub-approved design (2026-10-20, ops:0), second option:
    1. Warm per repository, not per target: one fresh Discover fills every slot of that repository, single-flight per repository. Today `hubTargetMemo` resolves per target, so a cold batch starts up to 8 parallel Discovers of the same repo and the rest refuse "warming".
    2. A lifetime-bound prewarm of the hub repositories at supervisor start: the primaries from `authority.hubs.snapshot().Hubs()[i].Slot.Repository`, deduplicated, run in `newMessageService` after the memo is created, cancelled by the service lifetime and joined in `Close`.
    3. Authority is still re-proven fresh before each effect (unchanged: the queued guard and the effect gate).
    4. Tier A test, landing with #726, in couchcmd: the real broker socket and the real CLI path (`runMessageCLIWithCall`) in hub mode, with three parked slots and a fake runner. Three concurrent `couch --resume repo:N` must all exit succeeded, with no "warming", no uncertain and no drop, and exactly one Discover counted. A cold variant without prewarm must also succeed for all three. Building blocks, from an explorer: `slot_operations_acceptance_test.go` (`TestSlotOperationSocketAcceptance`), `cold_resume_origins_test.go` remote subtest (`seedVerifiedPark`), `hubs_startup_test.go` (`newHubStartupRig`, `prepareExplicitHubs`, `wireResolver`), `hub_admission_test.go` (`newHubAdmissionRig`).
    5. Tier B (the gated live conformance with its own live hub caller identity) is filed as its OWN issue after #726, unclaimed and referencing #726. Not yet filed.
  - Done so far in this step (uncommitted until this checkpoint): `couchcore.(*Couch).RepositoryHubSlots(ctx, primary)` in `slotcontext.go`: one Discover, each verified row mapped to `HubSlot{PrimaryRoot, Number}`. It builds. No test yet.
  - Exact next steps:
    1. In couchcore, test that `RepositoryHubSlots` agrees with per-reference resolution (`WorkspaceReferencePath`, then `ResolveWorkspace`) on `recoverAcceptanceFixture(t, 3)`.
    2. Extend `hubTargetMemo` (`cmd/internal/couchcmd/hub_target_memo.go`) with an optional repository source: `names func(ctx) ([]couchcore.RepositoryName, error)` (`c.Threads.RepositoryNamesContext`) and `discover func(ctx, primary) (map[int]couchcore.HubSlot, error)` (`c.RepositoryHubSlots`). `Warm(ref)`: on a live entry return it. Otherwise map `ref`'s repo name to a unique primary (by `Dir` or `Alias`), join or start one repo flight (keyed per primary in the shared flight cap) that remembers `name:N` for every name of that primary, wait within the budget, then answer from the entries. If the entry is still missing (slot 0, an unverified slot or an ambiguous name), fall back to the existing per-target flight. Add `Prewarm(primaries)`, which does not block.
    3. Wire it in `newMessageService` and `run.go`: authority fields for the names and discover sources, set where `authority.hubTarget` is set, and prewarm the hub primaries.
    4. Memo tests: one Discover fills a batch; prewarm; fallback for slot 0; flight cap shared.
    5. Tier A tests as above, then mutations: a per-target-only memo must fail the one-Discover count, and no prewarm must fail the warm variant.
    6. Atlas note (hub admission paragraph in `atlas/couch.md`), a Revisions entry, then the suite (`sdlc local-check run --name suite`), `sdlc close --issue 726 --fixed-then-ship`, `sdlc issue recovery reconcile --issue 726` if FIX-THEN-SHIP, `sdlc pr` and `sdlc merge --yes`.
    7. File Tier B (`sdlc issue new ... --no-claim`, referencing #726) and report to the hub.
  - After #726: the pair#717 Codex interrupt failure (the hub asked to record it in #717's Log, or file a follow-up). `couch --interrupt` refused a working Codex turn, because `NotificationLifecycle.Working()` relies on Claude's OSC 9;4 progress, which Codex does not emit. The fix uses each agent's own working evidence, defined in one place (`workshop/lessons.md` #717 rule).

- 2026-10-20 (pair:2): recover-plan item. It does not reproduce: from a non-repository directory, `couch --recover-plan-from-sdlc` (f3c597f8) returns the workspace fleet present with 56 rows. The Problem's "fails … from a non-git cwd" was a misreading, as the hub confirmed. The failing row is a second fleet for `/Users/xianxu`: the home directory is enrolled as a repository because a thread started there, and its `sdlc workspace` probe fails with "not a git repository". The hub chose (b), an "ignored with reason" row, if small. A filesystem check proved the wrong seam: it bypasses the faked `SlotWorkspaceResolver` and broke three fixtures that enroll real temporary directories. So, per the hub, this is (a) here and (b) is filed as pair#729, with a typed resolver outcome as the design. Also: `TestHubSlotAdmissionSpawnsNothingWhenWarm` now runs for relaunch as well as resume, to match the Done-when.

- 2026-10-20 (pair:2): M2 closed FIX-THEN-SHIP (round 2). The one new finding was that the atlas and the plan's M2 bullet still said "affects only the inferred choices". Both now carry the corrected wording, in this one commit per the protocol.

- 2026-10-20 (pair:2): M2 review round 1. BR-13 is delivered as the plan wrote it: `TestRecoverySkipCandidatesOnAnOversizedNativeStore` builds a native store over `RecoveryMetadataLimit` with a matching transcript whose one record exceeds `RecoveryRecordLimit`. The full read reports incomplete (the control), and the skip reads no native byte, counted through the reader's own `BytesRead` against a screen-only read. Its mutation reproduces pair:6's exact production text: "native transcript incomplete: session inventory read exceeds limit". Minors: the capture fence has one site (the scan moved into `collectCandidates`, guarded by `!SkipCandidates`), and the `--resume-candidates` line now says only what holds whatever the cause ("the confirmed Resume from disk comes from the established binding, not from this scan").

- 2026-10-20 (pair:2): M2 implemented along pair:3's boundary. `SkipCandidates` is set only for direct Proven and Fresh selections, and `ReadRequest` returns after the Source-building prefix plus the Source-nil capture fence. Tests: `TestRecoverySkipCandidatesKeepsEveryNonCandidateFact` covers valid, expired, missing and retained-source screens: the full read ranks the matching transcript (control), the skip ranks none (zero transcript reads), and Source, Generation, Agent, Branch, Issue, Screen and Capture are identical, with no invented reasons. So a missing capture cannot become a new Preserve attempt. `TestPreparedRecoveryScansCandidatesOnlyForAnInferredChoice` (fresh and proven skip, inferred scans) and the `--resume-candidates` wording test. Mutations, each red: the reader ignoring the flag, the flag never set, the flag set for every selection, and a skip before the Source prefix.

- 2026-10-20 (pair:2): M1 closed FIX-THEN-SHIP (round 3). The round-3 fix lands in this one commit, per the protocol. A failed resolution is aged from when it finished, not from when it started, so a resolution slower than 30 s still reports its cause (`TestHubTargetMemoAgesFailuresFromTheirEnd`, which asserts no second resolution). `Warm` now caps its own wait at the admission budget for any caller (`TestHubTargetMemoWarmCapsItsOwnWait`). Mutations: aging from start, and no self-cap, each red. These fixes are covered by the next boundary review.

- 2026-10-20 (pair:2): M1 review round 2: BR-7 (a timing assertion) is fixed by a rendezvous of slot probes held at once. Mutation: serial Discover fails it after the bounded safety, about 20 s, with "all slots in flight together: false". My first rendezvous draft passed under serial twice: it counted the primary's own probe, then cumulative arrivals. Minors fixed: the receipt filter has a budget, start-time ordering, sweep and cap, and the plan's Revisions. Mutations: late result overwrites, no sweep, each red.

- 2026-10-20 (pair:2): M1 review FIX-THEN-SHIP; both blocking findings fixed by their class. BR-1: a failed resolution is remembered for 30 s (`hubTargetFailureAge`), so admission answers with the real error, not "warming" forever (`TestHubTargetMemoRemembersALateFailure`). BR-2: move-slot admission was the same class (fresh Discover per slot at admission). It now names slots from the memo within the budget, and the queued move re-resolves each fresh and refuses on a difference, because those slots are its payload (`TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh`: zero fresh resolutions at admission, then refusal when a slot shifted). Minors: Discover fan-out capped at 8 (`slotVerifyConcurrency`, with a 10-slot peak check), background flights capped at 8, canonical `repo:N` keys, and one `checkHubSlotOperation` with fresh or warm sources (DRY). Mutations, each red: failures not remembered, no flight cap, move admission resolving fresh, move prepare skipping the fresh check, and the Discover cap raised to 128.

- 2026-10-20 (pair:2): M1 implemented. Measured live after the concurrent Discover: pair 3.0 s down to 1.0 s, parley.nvim 2.7 s down to 0.85 s (load about 30). Tests: `TestHubSlotAdmissionSpawnsNothingWhenWarm` (zero spawning resolutions when warm; the queued guard re-proves fresh), `TestHubSlotAdmissionColdRefusesWarmingThenAdmits` (typed refusal inside the budget, nothing queued; admits once the late resolution lands), `TestHubSlotAdmissionStaleMemoRefusesBeforeAnyEffect`, the memo tests (single flight, expiry, Fresh refreshes, lifetime join), and `TestDiscoverVerifiesSlotsConcurrently` and `TestDiscoverStillFailsWholeInventoryOnReplacedHost`, under -race. Mutations, each red: admission resolving the target fresh, admission resolving the caller workspace fresh, no admission budget, Fresh answering from the memo, serial Discover, and no SameFile fence. A caller without a connected binding falls back to one fresh workspace resolution; the warm zero-spawn claim is for a connected hub, which is how hub CLI requests arrive.

### 2026-10-20
- 2026-10-20: closed — Full suite green on dd8a8666 (sdlc-local/suite). Diagnosis measured: admission ran a serial Discover of 3.0 s (pair, 27 commands) and 2.7 s (parley.nvim) against a 2 s exchange. M1 (FIX-THEN-SHIP, fixes landed): admission on warm sources in its own 1.5 s budget with typed refusals; TestHubSlotAdmissionSpawnsNothingWhenWarm for resume and relaunch (zero sdlc/git), cold, stale and move tests, memo tests, Discover rendezvous concurrency test; live Discover now 1.0 s and 0.85 s. M2 (FIX-THEN-SHIP, fixes landed): proven and fresh resume skip the native scan; the oversized-store fixture reproduces pair:6 text; equivalence across four screen cases. Recover-plan item does not reproduce; (b) filed as #729 per the hub. Many mutations red across rounds. Live check (pair:6 resume, idle relaunch) is the hub step after landing.; review verdict: FIX-THEN-SHIP
- 2026-10-20: closed M2 — Round 2. BR-13 delivered: TestRecoverySkipCandidatesOnAnOversizedNativeStore (store over RecoveryMetadataLimit, a record over RecoveryRecordLimit; full read incomplete, skip reads zero native bytes counted via BytesRead; its mutation reproduces the production text). Minors: one capture fence site via collectCandidates; resume-candidates wording true whatever the cause. All M2 tests green; couchcore and couchcmd green.; review verdict: FIX-THEN-SHIP
- 2026-10-20: closed M1 — Round 3. BR-7: TestDiscoverVerifiesSlotsConcurrently proves overlap with a rendezvous of slot probes held at once (no elapsed-time assertion); serial Discover fails it. Minors: receipt filter budget, start-time ordering (TestHubTargetMemoOrdersByStartAndSweeps), sweep and cap, plan Revisions. All M1 tests green under -race; mutations red. couchcmd, couchcore, artifactpath green.; review verdict: FIX-THEN-SHIP
