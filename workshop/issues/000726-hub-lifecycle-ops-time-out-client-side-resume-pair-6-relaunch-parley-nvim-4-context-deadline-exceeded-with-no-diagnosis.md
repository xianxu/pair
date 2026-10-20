---
id: 000726
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.77
collaboration-mode: collaboration
card_mirror: 'b4a97024cf8ebe0ffe456188544b976d5f9a9269' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T07:54:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

- [ ] M1 — admission returns a typed outcome inside the budget
  - **Warm sources for every admission input, so a warm admission spawns no process (PQ-1).**
    - *Caller workspace:* the supervisor's existing per-connection identity, `connectedWorkspace` (`s.workspaces`, set at connect). `authorizeParkAdmission` already uses it (ARCH-DRY). Admission no longer calls `authority.workspace` (that is `ResolveWorkspace` → `sdlc`).
    - *Target slot:* a memo `hubTargetMemo`: canonical target reference → `HubSlot` plus the time it was resolved. Only the existing fresh resolver (`hubTargetResolver`: `WorkspaceReferencePath` → `Discover`, then `ResolveWorkspace`) fills it. Every fresh resolution refreshes the memo, including those in queue prepare and at the effect gate. Admission reads an entry younger than its bound (10 minutes; slot topology changes rarely).
    - *Caller liveness and thread:* `liveCaller` and `authority.thread` read the store and recorded files and spawn nothing. The counted test proves it.
  - **Authority is still proven fresh before any effect.** The queue's prepare step and the hub effect gate keep running the full guard on fresh discovery, as today. A stale memo can at worst admit a job that then refuses with a reason, before any effect.
  - **Cold or expired memo (PQ-2).** Admission starts one fresh resolution for that reference. It is single-flight per reference, runs under the supervisor's lifetime context, and is joined at shutdown. Admission waits for it up to the admission budget. If the resolution finishes in time, admission uses it. Otherwise admission refuses ("slot catalog is warming; nothing was queued; retry in a few seconds"), the resolution keeps running, and its result fills the memo. Test seam: the resolver is injected, so a test can complete a resolution after the refusal it caused and see the next admission admit.
  - **Admission budget.** Slot-op admission runs under its own budget, below the transport's 2 s. On overrun it answers `refused`, naming the phase (caller, thread, target, queue), never silence.
  - **`hubReceiptFilter`** (receipt listing and polling) reads the same memo, so `operation-status` polling spawns nothing.
  - **Discover probes concurrently (PQ-3).** The per-slot `inspectSlot` calls run concurrently, at most `MaxSlotCandidates` at a time, and the results keep their order. Each row keeps its before/after `os.SameFile` replacement check, and any row error still fails the whole inventory, as now.
- [ ] M2 — a proven or fresh resume does not scan native candidates (boundary confirmed by pair:3, the recovery owner)
  - For a direct Proven or Fresh selection, `OSRecoveryEvidenceReader.ReadRequest` skips only the native candidate collection (`NewNativeTranscriptRuntime` / `CollectRecoveryEvidence`). It keeps the whole prefix that builds `Source` (`recovery_evidence_os.go`): retained source, producing-launch witness, activity, capture expiry, screen error and an explicit `request.Source`, including the expiry ScreenUnavailable handling. It also keeps `Agent`, `Generation`, the capture identity, byte caps, freshness and validation, and the Source-nil `validateRecoveryCaptureSnapshot` check. Ranking, native Incomplete and native Reasons are not consumed on this path. The inferred (keyed) path is unchanged, even when it later chooses Fresh. Both proven-binding checks, the ownership checks and the retirement checks stay.
  - `--resume-candidates` keeps offering "Resume from disk (confirmed)" (the binding is established) and says that an incomplete scan affects only the inferred choices.
- [ ] `--recover-plan-from-sdlc` from a directory outside any repository: resolve the fleet from Couch's own scope list, or refuse naming the directory to run from (PQ-6: a plain step, not a milestone).

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

- 2026-10-20 (pair:2): M1 review FIX-THEN-SHIP; both blocking findings fixed by their class. BR-1: a failed resolution is remembered for 30 s (`hubTargetFailureAge`), so admission answers with the real error, not "warming" forever (`TestHubTargetMemoRemembersALateFailure`). BR-2: move-slot admission was the same class (fresh Discover per slot at admission). It now names slots from the memo within the budget, and the queued move re-resolves each fresh and refuses on a difference, because those slots are its payload (`TestHubMoveAdmissionUsesWarmSlotsAndRechecksFresh`: zero fresh resolutions at admission, then refusal when a slot shifted). Minors: Discover fan-out capped at 8 (`slotVerifyConcurrency`, with a 10-slot peak check), background flights capped at 8, canonical `repo:N` keys, and one `checkHubSlotOperation` with fresh or warm sources (DRY). Mutations, each red: failures not remembered, no flight cap, move admission resolving fresh, move prepare skipping the fresh check, and the Discover cap raised to 128.

- 2026-10-20 (pair:2): M1 implemented. Measured live after the concurrent Discover: pair 3.0 s down to 1.0 s, parley.nvim 2.7 s down to 0.85 s (load about 30). Tests: `TestHubSlotAdmissionSpawnsNothingWhenWarm` (zero spawning resolutions when warm; the queued guard re-proves fresh), `TestHubSlotAdmissionColdRefusesWarmingThenAdmits` (typed refusal inside the budget, nothing queued; admits once the late resolution lands), `TestHubSlotAdmissionStaleMemoRefusesBeforeAnyEffect`, the memo tests (single flight, expiry, Fresh refreshes, lifetime join), and `TestDiscoverVerifiesSlotsConcurrently` and `TestDiscoverStillFailsWholeInventoryOnReplacedHost`, under -race. Mutations, each red: admission resolving the target fresh, admission resolving the caller workspace fresh, no admission budget, Fresh answering from the memo, serial Discover, and no SameFile fence. A caller without a connected binding falls back to one fresh workspace resolution; the warm zero-spawn claim is for a connected hub, which is how hub CLI requests arrive.

### 2026-10-20
