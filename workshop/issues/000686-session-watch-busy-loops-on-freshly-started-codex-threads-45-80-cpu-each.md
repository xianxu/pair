---
id: 000686
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c37fc5136d00c10e98d5637ea375219f4a0e5b4d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T18:47:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "129430e6", done: "f3ee9002"}
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
