---
id: 000686
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.63
collaboration-mode: collaboration
card_mirror: '9614e486eab88e0d50f471a2658377401d184e31' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T18:47:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [ ] Follow [the measured bound-watcher plan](../plans/000686-bound-watcher-plan.md): add counted red controls, reuse guarded transcript following, isolate bound work and back off idle polls.
- [ ] Verify shared title/watcher consumers and obtain the hub-coordinated live CPU check.
- [ ] Run recorded suite, close review and gated landing; report the merge.

## Revisions

### 2026-10-18 — measured bound-follow cause; design awaiting approval

Both live PIDs already have correlation bindings. The defect is bound lifecycle follow continuing discovery/validation every100ms, not an unbound wait. Proposed scope isolates that follow path, including ledger/cache/history work; unbound cold discovery remains unchanged and explicitly O(N), not falsely bounded. Reuse the title projection’s guarded tail with256KiB payload budget per pass and up-to5s idle backoff. Cold bootstrap cost,32MiB maximum complete-record decode, event latency and the shared-reader test surface are explicit in the durable plan. The hub must approve these boundaries before code.

## Log

### 2026-10-18


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
