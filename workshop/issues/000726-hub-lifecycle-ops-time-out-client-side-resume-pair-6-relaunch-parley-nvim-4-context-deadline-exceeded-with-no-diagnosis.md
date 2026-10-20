---
id: 000726
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.77
collaboration-mode: collaboration
card_mirror: '5f483a555946b5c764a8b97136b14c1aca514c3e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T07:54:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 3.74
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

- Tier A: three concurrent parked-slot CLI resumes succeed through the real broker socket and console, both after startup prewarm and from a cold repository memo, with one admission repository discovery and fresh authority before each effect. Tier B live conformance is a separate unclaimed follow-up per the hub-approved split.

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

- 2026-10-20 (hub-approved scope recorded at checkpoint): repository-wide single-flight warming replaces one discovery per cold target. Prewarm distinct hub repositories at message-service startup, cancelled and joined with that service. Keep fresh authority proofs at queue prepare and effects (ARCH-DRY, ARCH-STATE). Add Tier A socket/CLI batch acceptance here; file gated live Tier B conformance separately, unclaimed and referencing #726. The repository sources are wired where production constructs message authority in `message_service.go`; no separate `run.go` wiring exists.

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

- 2026-10-20 (BR-17): the expanded, hub-approved scope receives a full close review rather than the narrow fix recipe. This revision supersedes M1's original "only the existing fresh resolver fills it" and single-flight-per-reference descriptions: repository discovery now fills verified numbered slots once per repository, and fresh per-target proofs still refresh individual entries at prepare/effect. Startup prewarms distinct hub repositories under the same eight-flight cap and service lifetime. Current strategies: `TestHubTargetMemoRepositoryBatch` counts one discovery at N=4 and 2N=8, including aliases/prefixes and primary fallback; `TestHubTargetMemoPrewarmAndSharedFlightCap` proves deduplication, shared capacity and cancellation/join; failure expiry and late-fresh ordering have separate memo regressions. `TestHubResumeBatchSocketAcceptance` drives three real CLI/socket resumes to live persisted incarnations in prewarmed/cold variants, counts one admission discovery and fresh prepare/effect proofs, and goes red for removal of batching or prewarm. `TestRepositoryHubSlotsAgreesWithReferenceResolution` and `TestRepositoryHubSlotsRejectsUnverifiedRows` qualify the new source. These augment the original M1/M2 tests; they do not weaken the fresh authority invariant. Done when now explicitly includes Tier A; Tier B is the separate live fixture, while the existing hub live acceptance remains after landing.

- 2026-10-20 (BR-18/19, correction of the approved invariants): the one 1.5 s admission deadline belongs to the slot/move request entry, spanning authentication, target resolution, queue-key lookup, status filtering and enqueue. Nested guards/filtering retain that deadline; callbacks returning success after expiry cannot enqueue. Queued effect guards continue on their queue-owned contexts. Socket regressions combine successful 700 ms authentication with blocked target resolution, and successful 700 ms target resolution with blocked queue-name lookup; exact context/deadline and cancellation tests qualify slot/status/move/filter paths. Memo and queue-key selection now reuse the core exact-sibling precedence rule used by fresh resolution, before accepting a prewarmed alias. The service captures the supervisor workspace's FleetRoot at startup; if unavailable it retains per-reference resolution rather than inventing context. Warm reads may stat that exact local sibling but still spawn zero native processes. Six cold/prewarmed cases cover sibling names that also match enrollment prefixes or aliases, including constructor fleet-context wiring and physical queue identity. This corrects BR-18/19 without changing fresh effect authority, storage formats, user actions or recovery policy.

- 2026-10-20 (final re-plan approved by ops:0, delivery 27a17ef1-a145-4b57-a410-83502d344918): admission consumes connected caller proof and the warm memo only. The rejected simpler-looking alternative, passing a 1.5 s context to synchronous native resolution, fails because OSProvisionIO joins a two-second cancellation callback: the real socket reproducer completed its handler around 3.5 s after losing the 2 s reply. Remove both synchronous fallback sites: missing caller workspace refuses with reconnect instructions, missing target memo refuses unavailable. Target discovery remains service-owned, cancelled/joined with that service; fresh workspace/target proofs remain queue/effect-owned. The preview/file/cache sources for authentication, repository names and queue keys spawn no native subprocess; recovery preparation reads the console's cached inventory and queues its fresh checks. Move/status share these sources and the existing request budget. No admission deadline values change again.
  - A pending memo flight produces typed wire `warming`, a positive retry-after (500 ms), and no queued receipt. The CLI retries only that explicit nothing-queued response with the same request ID and immutable intent. Bound the complete admission wait at 30 s and eight attempts, clamp exchanges/sleeps to remaining time, and print waiting progress. Never retry uncertain admission, transport errors, other refusals, malformed hints or a warming reply carrying a receipt. After acceptance, the existing 5-minute status polling is unchanged. This is an additive transient response field, not a persisted format change.
  - Tests: `TestMissingCallerWorkspaceSocketNeverRunsSynchronousResolver` uses the real OSProvisionIO/resolver with an owned slow sdlc script; it first lost the socket response in 3.51 s and now refuses with zero native calls. `TestMissingHubMemoNeverFallsBackToSynchronousTargetResolution` requires zero raw target calls/queued jobs. `TestSlotCLIRetriesExplicitWarmingWithSameIntent`, `TestSlotCLIWarmingDoesNotRetryOtherOutcomes` and `TestSlotCLIWarmingRetryBudgetIncludesCallsAndSleep` qualify retry safety, intent identity and the total bound with a controlled clock. `TestHubResumeBatchSocketAcceptance/cold-restart-held` holds the one repository discovery until all three real CLIs receive actual typed warming responses with no receipts and zero synchronous workspace/raw-target calls; only then releases it and requires all three to succeed and persist live incarnations. Disabling automatic retry fails this fixture. Existing split-stage socket and receipt tests now use memo/connected sources rather than the removed synchronous fallback.
  - Stop rule: if the next boundary review finds another admission path, stop and report to the hub instead of further deadline tuning.

Actions for this re-plan: run the lifecycle command, automatic warming retry, reconnect then retry, inspect the recovery report.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Caller workspace proof missing | refused, reconnect required, nothing queued | reconnect then retry | none |
| Target memo missing | refused, catalog unavailable, nothing queued | reconnect/restart then retry | none |
| Memo flight pending | warming with retry-after; CLI prints waiting | automatic bounded warming retry | none |
| Warming remains after attempt/time limit | refused, nothing queued, retry requested | run the command later | none |
| Warm authority snapshot is stale | admitted job refuses on fresh proof before effect | inspect current ownership and retry | none |
| Admission reply lost or uncertain | existing outcome-uncertain report | inspect recovery report; never automatic resend | an effect may already have happened; no claim of failure |

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

- 2026-10-20 (pair:2, Codex): affected package verification completed: couchcore (213.042 s), couchmessage (25.205 s), artifactpath (10.493 s), and the repaired full couchcmd rerun (60.811 s) all pass; vet passes. Noon close preflight correctly refused the older base, naming newly landed main code, including #725's affected-test runner/configuration. Merged origin/main cleanly with no code conflict; no admission policy or deadline changes. This integration now declares .sdlc/affected.json, so the next recorded pre-close check is `sdlc local-check run --name affected`; the landing gate owns the final full integration suite. No extra CI retry was requested for old heads.

- 2026-10-20 (pair:2, Codex): checkpoint before the next recorded run/review. Final re-plan implementation is complete locally: no synchronous workspace/target fallback, typed warming plus 500 ms retry-after, CLI same-intent retries bounded by 30 s/eight attempts, no uncertain resend. New CLI and native cancellation regressions, all three Tier A variants (race count=3), and repaired fixture regressions pass. Full couchcmd run initially failed only fixtures that depended on old fallback (guard/move/confirmation); those now construct the memo before service startup and supply exact connected workspace facts, preserving/strengthening ownership and stale/generation assertions. Named fixture tests pass under race; full couchcmd rerun and original couchcore/couchmessage/artifactpath package run remain in progress. No active mutations; new suite not yet queued. Review round 9 evidence committed with this fix; next admission-path finding must be reported to hub with no further tuning. Tier B draft remains /Volumes/TestScratch/tmp/pair-slot2/pair-726-followup.ZXzzyr/tier-b.md; file unclaimed after landing PR #487.

- 2026-10-20 (pair:2, Codex): round 9 disposed BR-19 and all other findings but kept BR-18 open: a caller workspace cache miss still entered OSProvisionIO and joined two seconds of cancellation. Stopped incremental budget changes, enumerated the synchronous fallbacks and obtained ops:0 approval for the cache-only admission re-plan plus typed warming CLI retry. Removed caller-workspace and missing-memo synchronous fallbacks; real-runner socket and zero-spawn regressions passed. The held-cold three-CLI acceptance passed race three times (21.597 s), and suppressing warming retry made it fail; all three requests first prove warming/no receipts/zero synchronous authority probes, then succeed after one discovery is released. CLI retry budget/cap mutations fail and are restored. Full affected package tests running before the next single push; cancelled the superseded queued suite. The next review finding another admission path is a hard stop/report to ops:0.

- 2026-10-20 (pair:2, Codex): final BR-18/19 focused race run passes across couchcmd/couchcore (all memo, sibling, deadline/socket, stale admission, move, Tier A, scan-selection and oversized-store regressions); vet passes. `TestHubTargetMemoSiblingDoesNotNeedEnrollment` first failed on an unrelated enrollment error, then passed after exact-sibling selection stopped depending on enrollment and alias hints became optional for an already verified repository snapshot. Queue-name selection uses the same sibling rule. `TestBlockedStatusFilterDoesNotHoldTheReceiptMutex` (rendezvous) also defends the existing receipt snapshot-before-IO rule. All mutations are restored; committing the correction and its review evidence together before the new suite/full close.

- 2026-10-20 (pair:2, Codex): full close round 8 disposed BR-1..17, then found BR-18 (the budget omitted initial authentication and final queue-name lookup) and BR-19 (memo selection skipped sibling precedence). Reproduced both socket drops and expired-success enqueue before moving budget ownership; regressions now refuse inside the complete request budget and enqueue nothing. Reproduced all six cold/prewarmed sibling-versus-prefix/alias mismatches; memo, fresh resolution and queue identity now use the shared core sibling rule. Mutations red: a 3 s entry budget drops both 2 s socket exchanges; removal of expiry fences enqueues one job; removal of constructor fleet-root wiring breaks the sibling cases; disabling the shared sibling rule breaks the existing core precedence regression. Production restored after every mutation. Cancelled the superseded queued suite before code correction. Hub notified of correction plan.
- 2026-10-20 (pair:2, Codex): inspected CI run 39475078416 before the next push per ops:0. The macOS conformance job never acquired a hosted runner: runner_id=0, no steps and no logs. Check annotations name hosted runner acquisition failure and macOS arm64 capacity constraints. No conformance test ran, and no rerun requested. Hub notified; bundle the completed correction into one push.

- 2026-10-20 (pair:2, Codex): narrow re-close returned REWORK solely for BR-17 (boundary-review-scope): repository-wide warming is approved new scope and requires full close. Prior BR-16 wording disposed as addressed. No new implementation defect was raised. Recorded the governing revision and Tier A Done-when, and proceeding through full close as the gate directs; the suite on the unchanged code head remains queued.

- 2026-10-20 (pair:2, Codex): resumed the approved CHECKPOINT. Repository warming now discovers once for a cold batch, fills unambiguous names/aliases, preserves the per-reference fallback, shares the eight-flight cap and existing bounded entries, remembers repository failures, and refuses older results over newer fresh proofs. Startup prewarms distinct hub repository primaries on the service lifetime, joined at Close; production sources are wired in message_service.go. `TestRepositoryHubSlotsAgreesWithReferenceResolution` checks the real three-worktree fixture; `TestRepositoryHubSlotsRejectsUnverifiedRows` also counts one Discover and covers discovery failure/missing catalog. Memo tests count one Discover at N=4 and 2N=8 and cover aliases/prefixes, primary fallback, shared cap, cancellation, failure expiry and late completion.
- 2026-10-20 (pair:2, Codex): Tier A `TestHubResumeBatchSocketAcceptance` drives three concurrent parked-slot resumes through the real broker socket, CLI and console recovery/launch/adoption, in startup-prewarm and cold variants. Both pass under race three times. It requires every CLI success and persisted live incarnation, one admission repository Discover, and fresh authority proofs for each prepare/effect. Both mutations fail: per-target-only source breaks the one-Discover assertion; no startup Prewarm times out the pre-request startup assertion. A test-local mutex protects SlotCatalogFake call bookkeeping under concurrent console jobs. Final focused couchcmd/couchcore race tests pass (memo, admission, move, Tier A, proven/fresh scan selection and oversized store); core/artifactpath source checks pass; core vet passes. Cancelled the prior-session recorded suite on old 44c1e2e3 through local-check cancel; queue the new pushed head next. Tier B remains the separately authorized follow-up after landing.

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
- 2026-10-20: closed — Post-SHIP metadata fix only: audited admission_deadline_test.go has no repository inputs, only owned t.TempDir/bin/sdlc; registered its digest as local_or_literal_fixture. python3 -B tests/affected-go-readers-test.py passes all 8 tests; git diff --check passes. Prior full close SHIP and Go package/race/vet evidence unchanged. Recorded suite on a9afd107 failed solely on the missing inventory entry before Go tests; a new recorded suite must pass before landing. (fixed-then-ship); review verdict: SHIP
- 2026-10-20: closed — Final cache-only admission re-plan approved by ops:0. All affected packages pass: full couchcore/couchmessage/artifactpath and repaired full couchcmd, plus vet. Real OSProvisionIO cancellation socket reproducer first lost response after3.51s, now missing caller workspace refuses with reconnect and zero synchronous native calls; missing memo refuses with zero raw target calls/jobs. Typed warming replies carry retry-after and no receipt. CLI retries only that type, sameID/intent, capped30s/8attempts inclcall/sleep; malformed/uncertain/transport/other replies never resend. Retry clamp/cap mutations fail. Real socket+CLI+console TierA startup-prewarmed/cold/held-cold all pass race count3: held discovery releases only after all3 actual warming replies, zero synchronous authority probes and zero queued receipts; all3 later succeed/persistlive after one repository discovery and fresh prepare/effect proofs. Suppressing retry fails. Shared sibling precedence/queue identity regressions and mutations pass/fail appropriately; stale/move/capture/native-scan regressions pass. Noon gate required clean merge of origin/main; integration adds affected runner, no lifecycle code conflicts. Recorded affected check queued on pushed a9afd1079e95, must pass before landing; landing owns full integrated suite. Per hub: if this review identifies another admission path, stop/report instead of tuning deadlines. TierB and hub live acceptance follow approved landing sequence.; review verdict: SHIP
- 2026-10-20: closed — Full suite green on dd8a8666 (sdlc-local/suite). Diagnosis measured: admission ran a serial Discover of 3.0 s (pair, 27 commands) and 2.7 s (parley.nvim) against a 2 s exchange. M1 (FIX-THEN-SHIP, fixes landed): admission on warm sources in its own 1.5 s budget with typed refusals; TestHubSlotAdmissionSpawnsNothingWhenWarm for resume and relaunch (zero sdlc/git), cold, stale and move tests, memo tests, Discover rendezvous concurrency test; live Discover now 1.0 s and 0.85 s. M2 (FIX-THEN-SHIP, fixes landed): proven and fresh resume skip the native scan; the oversized-store fixture reproduces pair:6 text; equivalence across four screen cases. Recover-plan item does not reproduce; (b) filed as #729 per the hub. Many mutations red across rounds. Live check (pair:6 resume, idle relaunch) is the hub step after landing.; review verdict: FIX-THEN-SHIP
- 2026-10-20: closed M2 — Round 2. BR-13 delivered: TestRecoverySkipCandidatesOnAnOversizedNativeStore (store over RecoveryMetadataLimit, a record over RecoveryRecordLimit; full read incomplete, skip reads zero native bytes counted via BytesRead; its mutation reproduces the production text). Minors: one capture fence site via collectCandidates; resume-candidates wording true whatever the cause. All M2 tests green; couchcore and couchcmd green.; review verdict: FIX-THEN-SHIP
- 2026-10-20: closed M1 — Round 3. BR-7: TestDiscoverVerifiesSlotsConcurrently proves overlap with a rendezvous of slot probes held at once (no elapsed-time assertion); serial Discover fails it. Minors: receipt filter budget, start-time ordering (TestHubTargetMemoOrdersByStartAndSweeps), sweep and cap, plan Revisions. All M1 tests green under -race; mutations red. couchcmd, couchcore, artifactpath green.; review verdict: FIX-THEN-SHIP

- 2026-10-20 12:24 PDT — Recorded suite on a9afd107 stopped in the #725 reader inventory before Go tests: new admission_deadline_test.go was unmapped because its scratch bin directory matches the lexical tripwire. Audited the full file: only owned t.TempDir/bin/sdlc fixture setup; no repository reads. Registered its digest as local_or_literal_fixture; no production behavior or protected gate configuration changed. Re-close and rerun recorded suite before landing.
