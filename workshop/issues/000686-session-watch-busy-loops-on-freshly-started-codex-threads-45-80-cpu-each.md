---
id: 000686
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.63
collaboration-mode: collaboration
card_mirror: 'c92e5689e937ccc6937f81030ed726812168a2e7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T18:47:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.54
---

# session-watch busy-loops on freshly started Codex threads (45-80% CPU each)

## Problem

On 2026-10-18 at 18:47, the `pair session-watch codex` processes of the two Codex slots that were just started fresh were busy: pid 70442 (`session-watch codex 1-pair-69`, pair:0, started 18:17) at 45–81% CPU and pid 84747 (`session-watch codex 1-parley-nvim-70`, parley.nvim:0, started 18:22) at 34–70% CPU, continuously. Older Codex slots' session-watch processes sit at 0%. The machine load average was 55 at the time (suites and builds too), and slot launches were timing out on zellij pane birth.

Likely shape (unverified): a fresh Codex thread has no bound rollout yet, and session-watch polls or rescans ~/.codex/sessions in a tight loop looking for it.

## Spec

session-watch must idle at about 0% CPU while it waits for a fresh thread's transcript: bounded, backed-off polling or file-system notifications, with work that doesn't grow with the size of ~/.codex/sessions per tick.

After an exact binding is established, lifecycle follow must observe only that authorized transcript, reuse existing identity and mutation guards, retain a compact cursor, and consume only new records. Each pass reads at most256KiB of new payload and backs off idle polling to at most5seconds. Unchanged follow must perform no listings, foreign reads, catalog IO or history decoding. Initial unbound discovery and one restart bootstrap retain their existing cold cost, as approved in Revisions. A fresh-slot live CPU check is coordinated by the hub.

## Done when

- A regression with a fresh thread and a large sessions tree shows bounded work per tick (counted), and no busy loop.
- On the live machine, a freshly started Codex slot's session-watch stays near 0% CPU (ps sample).

## Plan

- [x] Follow [the measured bound-watcher plan](../plans/000686-bound-watcher-plan.md): add counted red controls, reuse guarded transcript following, isolate bound work and back off idle polls.
- [ ] Verify shared title/watcher consumers and obtain the hub-coordinated live CPU check.
- [ ] Run recorded suite, close review and gated landing; report the merge.

## Revisions

### 2026-10-18 — measured bound-follow cause; design awaiting approval

Both live PIDs already have correlation bindings. The defect is bound lifecycle follow continuing discovery/validation every100ms, not an unbound wait. Proposed scope isolates that follow path, including ledger/cache/history work; unbound cold discovery remains unchanged and explicitly O(N), not falsely bounded. Reuse the title projection’s guarded tail with256KiB payload budget per pass and up-to5s idle backoff. Cold bootstrap cost,32MiB maximum complete-record decode, event latency and the shared-reader test surface are explicit in the durable plan. The hub must approve these boundaries before code.

## Log

### 2026-10-18
- 2026-10-18: closed — Affected packages and counted regressions pass; BR-1 corrected current concept locations and scanner/cursor ownership, with prior mapping preserved in Revisions; documentation only. Recorded suite on c0f766ffc193084da608b9a973ac2b07d039db37 remains queued, unchanged code, review overlap allowed by tl-worker and landing held until pass. --no-plan-check acknowledges that pending suite and hub-deferred live CPU acceptance, explicitly owed after landing per 6593f812; no staging or live success claimed. Clean build and focused race repeats passed.; review verdict: SHIP


### 2026-10-18 — read-only live diagnosis

Claimed/start-plan from pair:7. start-plan refreshed the checkout's older ariadne through its own weave catch-up, then required a repeated start-plan; no installer or machine-setting change was chosen. Both live watcher identities matched the dispatched PIDs/commands and remained untouched. Three-second samples at10ms are retained locally at /tmp/pair686-watch-70442.sample.txt and /tmp/pair686-watch-84747.sample.txt. Both show Run -> incrementalWatcherInventorySnapshot -> ValidateTargetWork -> ValidateCodexDelta -> JSON decoding, with metadata enumeration/catalog reconciliation and lifecycle-history sorting also sampled. CPU snapshots: pair watcher84.1% then72.3%, parley watcher63.9% then81.1%; cumulative CPU grew from14:09.98 to15:51.82 and13:58.08 to15:37.65 over135seconds. Their Pair ledgers already contain correlation bindings for launch ordinal2, so the initial unbound-wait hypothesis is false for these processes.

Metadata-only native-tree census:3,549 JSONL files,103 directories,6,467,935,788 bytes. The two scope catalogs are26,583/68,270 bytes. No native transcript body or credential file was read by this diagnosis. The hot production path is bound lifecycle-follow: RunCLI enables FollowLifecycle, default Poll=100ms, and the bound branch still recreates incremental inventory, walks the whole store, selects all post-launch candidates, validates/advances tracked candidates and republishes catalog before sleeping Poll. It should instead follow only its bound authorized transcript. Adjacent work limits must account for ObserveStableArtifact chasing append growth and lifecycle projection copying/sorting retained old events. Existing ObserveNamedArtifacts, fake operation counters and guarded title-transcript projection are reuse candidates; no production changes or final design yet.


### 2026-10-18 — hub approved scope/cadence

Hub approved87b52684bd89dd0295624d4cae2d080dce3a7163 (7f664349-b72a-4c11-a1b2-91baf70f21ba): bound-only own transcript, shared guarded suffix reader,256KiB per pass, idle backoff to5s, counted4k/8k controls with zero discovery/foreign/catalog reads during follow. Cold unbound discovery and one restart bootstrap may remain O(N); retain this as the slot-lifecycle project's follow-up (existing #157 is relevant), for the TL to record in the project. Up to5s idle latency accepted. The hub will arrange a fresh Codex slot on this build for live CPU acceptance when ready; no live intervention authorized by that future arrangement yet.


### 2026-10-18 — PQ-1/PQ-2: enforced transition and function strategies

The authoritative bound-follow production component is pure reduceBoundFollow(state,event), with phases Reading, Publishing and Stopped. It owns the committed cursor/scanner snapshot, acknowledged watermark, at most one prepared batch/candidate cursor and its next publication index. These are internal phases, not extra operator UI states. The IO loop only executes the reducer's next read/publish/stop effect; it cannot independently advance cursor or watermark (ARCH-ORDER).

| Phase / event | Transition and commit |
|---|---|
| Reading + valid batch, owner/process fences passed | Stage candidate cursor/scanner and the bounded new-record batch; enter Publishing. No cursor or watermark advance yet. |
| Reading + valid batch with no lifecycle output | Commit candidate cursor/scanner after the owner/process fence; remain Reading, update cadence from bytes/backlog. |
| Publishing + exact acknowledged append at expected index | Advance acknowledged watermark/index only. Existing journal reconciliation/dedup owns append certainty. |
| Publishing + final acknowledged append | Commit candidate cursor/scanner; discard staged records, return Reading. |
| Publishing + confirmed failure or unconfirmed outcome | Enter Stopped, retain the acknowledged prefix watermark and original committed cursor; do not claim the failed/unconfirmed record committed and do not retry automatically. Already durable journal records remain. |
| Any active phase + changed process/launch/binding, invalid transcript or canceled observation | Discard uncommitted cursor/state/batch, enter Stopped; preserve acknowledged journal progress. |
| Stopped + any late/duplicate result | No effects. Exact mismatched/stale publication identities never advance the watermark. |

An already committed prefix is never rolled back because a later append fails. A failed append which existing journal reconciliation proves committed is an acknowledgment; an uncertain result stops the watcher. Relaunch uses the existing bootstrap/watermark policy, not a persisted retry queue or history replay: a telemetry gap is possible, native work and durable records are preserved. Use the existing current-confirmation fence for the accepted generation around publication, so a changed owner cannot authorize another batch; this reuses the ledger's existing guard, not a new lock. The IO publisher reports one typed result per expected record; pure reduction decides advancement. Add controlled success-prefix/failure/uncertain/reconciled-commit traces through this actual reducer, not a parallel model used only by tests.

This table is the authoritative named-function strategy required by PQ-2; functions touching IO are explicitly tested with stateful runtimes, not called pure:

| Production function | Adversarial input / mechanical guard |
|---|---|
| reduceBoundFollow (pure) | Generated phase/event traces, permuted/stale acks, acknowledged prefix then error/uncertainty and cancellation; only expected acknowledgments advance watermark, only complete acknowledged batches commit the cursor, Stopped emits no effects. |
| nextLifecycleDelay (pure) | Boundary/property tests over base/previous duration and progress/backlog booleans; monotone idle doubling capped at5s, progress resets, no negative/overflow delay. |
| LifecycleRecordsFromEvents (pure existing projection) | Bounded fresh batches with old/foreign/unkeyed/malformed events and reordered positions; exact offset/turn filtering, watermark monotonicity and no dependence on retained old history. |
| NewTranscriptCursor and TranscriptCursor.Next (IO integration) | FakeRuntime append/replace/truncate/mutate/cancel at every guard; maximum four64KiB payload chunks, fixed snapshot under continuous append, partial framing and retry from discarded candidate, no foreign read or unsafe commit. |
| Exact cached owner reader / factored readOwnerLaunchFromLedger (IO integration) | Large ledger cold/warm/mutated/replaced/conflicting inputs plus mid-read drift; unchanged bytes read/decoded=0, exact artifact stats without listing, strict latest-launch authority unchanged. |
| publishLifecycleBatch (IO integration) | Stateful existing journal writer/reconciler acknowledges a prefix then fails/returns uncertain/proves committed; reducer receives exact results once, no automatic retry, retained durable prefix and no duplicate record after reconciliation. |
| followBoundCodex / Run (IO integration) | Fresh correlation into bound follow with4k/8k unrelated trees, explicit fake sleeps and post-binding counts; zero native/Pair listings, foreign/catalog reads and history decode; exact suffix progress and launch/PID shutdown. |
| TitleProjection.tail / PollContext (IO integration) | Complete recorded/native fixtures and existing title/Agy mutation/cancellation permutations through the shared reader; preserve current admission, database and display behavior. |

These refinements specify the already approved owner/cursor commit guarantees and failure behavior. They add no new user action, retry, data format or scope beyond the approved bound follow. Re-run the full plan gate before implementation.

### 2026-10-18 — hub approved publication contract; plan round2

Hub approved the Reading/Publishing/Stopped contract (ae598ecd-9080-4d36-a3a3-6c9736304c0f): staged cursor, acknowledged-prefix watermark, cursor commit only after the whole batch, unresolved publication stops without retry. Plan round2 disposed PQ-1; PQ-2 remained solely because the old procedural inventory duplicated the named-function table. Removed that duplicate from the plan; counted Run, consumer sweeps and live acceptance remain. No implementation yet.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is stale, so these are provisional ship-wall-clock hours. Read v2 and v2.1 procedure before deriving. This is single-repo familiar Go work; no external library replaces the existing security-sensitive guards. Reuse the guarded reader, owner cache and journal reconciliation rather than a new library or protocol.

Primitive derivation: issue authoring/measurement design0.7 plus raw impl0.2; shared cursor extraction is cross-cutting refactor design0.6×0.2, impl0.5; pure commit reducer is greenfield Go design1.0×0.2, impl0.8; bound driver/owner adapter is smaller Go design0.3×0.2, impl0.5; atlas design0.2×0.2, impl0.15; one close review design0.2×0.2, impl0.5; live process discovery/conformance raw impl0.6. All implementation values below apply v3.1's0.40 factor once. Familiarity1.0; thorough approved plan gets15% design buffer. Design subtotal1.16, impl subtotal1.30, total2.634 rounded2.63. No separate milestone review or second full review is budgeted.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.7 impl=0.08
item: cross-cutting-refactor design=0.12 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.04 impl=0.06
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 2.63
```

### 2026-10-18 — active publication owner guard

Narrow review found that existing WithCurrentConfirmation rereads/parses the full Pair ledger under lock for every nonempty lifecycle batch, outside NativeRuntime counters. Hub chose option1 in cd83e66c-02ca-4a93-b3a0-f82ce8d28254: reuse the exact OwnerLaunchCache reader under the same ledger lock, with the same strict ordinal/conflict/root confirmation check. Extract a reader-bearing variant of the existing guard; legacy callers retain the ordinary reader. Only bound lifecycle supplies its cached exact-ledger reader. Full fingerprint identity/size/time mismatch triggers the existing paired full reread before effects, never stale confirmation. Add active unchanged-byte counters including the actual guard path, and append/replacement stale-owner regressions. No new lock/cache/admission policy. This closes the per-event-tick cost rather than narrowing the acceptance to idle only (ARCH-DRY/SECURE/ORDER).

### 2026-10-18 — implementation and measured red/green controls

Full change-code cleared plan round4 CLEAN and estimate INFO. Estimate tests are included in the module/refactor items; review includes recorded-suite/consumer/race/integration/landing overhead, and real-api-discovery is an analogy for fresh-slot build validation and live process sampling, not a new external API. Calibration remains provisional; incurred diagnosis belongs to issue-spec.

Red Run control: seven bound idle passes performed7 listings and28,000 foreign body reads at4k files, and7/56,000 at8k. Green now doubles both unrelated tree and owned prior history and asserts0 native/Pair listings, body reads/decoder inputs, ledger bytes and catalog reads/writes, with exactly7 own stats. Existing prefix-failure publisher returned watermark0 after a successful first append; the reducer now preserves its exact prefix and never retries uncertain publication. Active guard red control found11 legacy whole-ledger reads; the production reader-bearing guard now reports0 unchanged body/decode reads and rejects both append and same-size/mtime replacement after rereading. The cache reader and effect are tested under the existing writer lock.

Shared cursor/title extraction full sessioninventory suite PASS94.166s; cursor/title race×5 PASS11.825s. Exact-owner/cache race×3 PASS5.095s. New focused watcher/reducer/guard controls passed. Sessionwatch+sessionledger complete packages PASS20.429s/2.684s. Artifactpath inventory guard PASS29.703s. The existing later-completion fixture expected2 range reads; it now requires4: initial validation, admission boundary capture, growth boundary verification, suffix payload. This is the approved shared guard addition, not an unexplained expectation update. Stateful watcher appender now mirrors durable records into the authoritative ledger with physical ordinals, instead of keeping them only in an assertion slice; production authority was not relaxed for the fake.

Two narrow peer reviews checked shared guards and follower transactions. The only material finding was active publication's separate full-ledger parse, now addressed under the hub-approved revision. Full recorded suite and hub-coordinated live CPU acceptance remain pending; original watchers and pair:0 remain untouched.

### 2026-10-18 — acceptance sequencing revision

Hub6593f812-a3ff-468a-ad5e-63aa04669518 explicitly changes live acceptance to post-landing: no staging or fresh launch while host load is125 and launches time out. Land on the recorded suite and SHIP close review. The TL then rebuilds pair:0 and relaunches the actual affected pair:0/parley.nvim:0 slots, measuring both new watchers near0% idle and return to idle after one short round. This live acceptance is OWED to the hub; no live success is claimed here. The two original spinning watchers remain untouched by pair:7. Local clean build c0f766ffc193084da608b9a973ac2b07d039db37 verified the same binary vcs.revision with vcs.modified=false. Focused watcher/ledger race×3 passed20.669s/1.240s. Recorded branch suite started on that pushed head and is queued by the shared gate.

### 2026-10-18 — close round1 REWORK, BR-1 documentation corrected

Boundary review found no runtime defect and independently passed sessionwatch/sessionledger/sessioninventory/artifactpath packages. BR-1 is the stale Core concepts table: cadence moved into bound_state.go, and scanner state belongs to boundCheckpoint rather than TranscriptCursor. Updated the delivered entity map and appended a Revision preserving the prior wording/delta. No production/test changes, so the queued c0f766ffc suite remains applicable. Rerun full close for explicit disposition; live acceptance stays owed per the hub.
