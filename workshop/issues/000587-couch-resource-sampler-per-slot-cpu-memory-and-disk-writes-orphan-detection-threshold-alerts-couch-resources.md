---
id: 000587
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 4.883
card_mirror: 'e2f429d5d3b1adb11e269c0f705d5efa7f0995c0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T16:54:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch resource sampler: per-slot CPU, memory and disk writes, orphan detection, threshold alerts, couch --resources

## Problem

Every resource incident so far was noticed by a human seeing a slow or full machine: a 22 GB orphan `nvim --embed`,
fseventsd at ~90 GB, #579's 61× write amplification (System Monitor), the scratch volume filling, the 15 s ledger
decode (#584). Nothing measures resource use per slot, so problems surface days late.

Part of the ops project resource-observability (`ops/teams/devinfra/projects/resource-observability/project.md`).

## Spec

- **A sampler in Couch's supervisor:** about once a minute, for each slot's process tree (Couch already knows the
  roots), read per-process CPU time, resident memory and **bytes written to disk** via `proc_pid_rusage`
  (`ri_diskio_byteswritten`, no root), and sum per slot. Also sample processes Couch launched that no live slot owns
  (orphans, e.g. a leftover `nvim --embed`).
- **Thresholds → one hub alert per episode:** sustained disk writes (e.g. > 2 MB/s for 10 min), memory (e.g. one
  process > 8 GB), CPU (a slot pinned > 90% for 15 min while idle), any orphan over a memory floor. Thresholds are
  configuration with defaults; the alert names the slot, process, metric, value and duration, and is not repeated
  until the episode ends.
- **`couch --resources [--json]`** (light): the current per-slot table (CPU %, RSS, write rate, totals since start)
  plus orphans.
- **A small rolling history** (bounded, e.g. last 24 h at 1-minute resolution) so the alert and the verb can show
  rates; registered with pair's storage manifest so retention covers it.
- Cheap by construction: the sample itself must be measured (a counted bound on syscalls per sample) and stay well
  under 1% of one core.

## Done when

- Fake-backed tests: per-slot aggregation over a process tree; an orphan detected; threshold episodes alert once and
  re-arm after recovery; history stays within its bound.
- A live check on this machine: `couch --resources` shows every slot; a deliberate write burst in a scratch slot
  raises one alert at the hub.

## Plan

Durable plan: [Couch resource sampler](../plans/000587-couch-resource-sampler-plan.md).

- [x] Counted Darwin resource reader with real owned-child conformance.
- [x] Pure generation-safe attribution, rates, provenance and alert episodes.
- [x] Authenticated full-slot roots and supervisor-owned sampler lifecycle.
- [x] Bounded restart-safe history and existing broker/hub delivery.
- [x] Cached resources CLI, configuration, docs and storage registration.
- [ ] Targeted/full qualification, measured overhead, operator-coordinated live scratch-slot alert, SDLC review and landing.


## Log

### 2026-10-15


2026-10-15: Claimed and started design. Read ops resource-observability project and issue. Read-only seam review found authenticated Zellij server evidence already available as PairSessionBinding.Owner.Server; wrapper-only ancestry would omit draft editors. Preserve that evidence at admission/lifecycle boundaries rather than re-running Zellij during sampling. Native table currently hides an unbounded ENOMEM retry loop, so syscall accounting must live at the native boundary. Full design and proposed defaults are in the durable plan; awaiting TL approval before change-code. No implementation or live-slot changes made.

## Revisions

### 2026-10-15 — proposed implementation contract for TL approval

Make sampling limits explicit: orphan attribution requires a previously observed generation/provenance; children born and orphaned entirely between samples and pre-upgrade unknown orphans cannot be proven. Include every known slot row, with partial/unavailable coverage where root or counter evidence is missing. Propose 60s cadence; 2 MiB/s for10m writes, 8 GiB process RSS, 90% one-core idle CPU for15m, 256 MiB orphan RSS over two valid samples. History is <=24h/1440 aggregate frames/4 MiB, plus <=2 MiB ownership/episode checkpoint, with shorter coverage disclosed at the byte cap. Count actual native attempts and persistence cost; qualify <0.1% average one-core cost at default cadence. Darwin uses the SDK libproc API through cgo; unsupported builds report that explicitly. These choices require TL plan approval, and live smoke staging remains separately coordinated.


### 2026-10-15 — TL plan approval and unattributed addition

Plan bf764e14 approved (receipt9f23fae9-2182-488f-8025-196b3940fa66). Add report-only unattributed (unproven) large PPID1 processes matching known Couch-launched kinds, including nvim --embed, Zellij and agent CLIs; no alert or slot attribution without provenance. Candidate argv probes are bounded and counted in the cost envelope. Continue to change-code.

2026-10-15: Plan gate round1 raised PQ-1/2/3 (orphan alert sender, immutable retry envelope, named function test strategies). Durable Revisions now specify owning-hub self-addressed system notices, persisted exact envelope with conservative unknown outcomes, and per-function adversarial/mechanical contracts. Re-running plan gate before code.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Derived after plan-quality CLEAN (round3). This is a multi-concern supervisor extension, decomposed into independently testable adapters rather than counted again as a greenfield service.

- Issue/spec: design0.8 + implementation0.12 (v2 implementation0.3 ×0.4).
- Native adapter: API integration, design1.0 ×0.2 =0.2; implementation0.8 ×0.4 ×1.5 native-stack familiarity =0.48. SDK libproc and existing procutil parsing supply the platform seam; no third-party library replaces the counted-call/generation contract, so no speculative library discount.
- Pure attribution/rates/episodes: greenfield module, design1.5 ×0.2 =0.3; implementation0.8 ×0.4 =0.32. No library embodies Couch ownership/episode rules.
- Bounded store/history: greenfield module, design2.0 ×0.5 existing durablefile helper ×0.2 =0.2; implementation0.8 ×0.4 =0.32.
- Worker, authenticated roots and broker integration: API integration, design1.5 ×0.2 =0.3; implementation1.5 ×0.4 =0.6.
- Cached CLI/policy: smaller module, design0.1; implementation0.5 ×0.4 =0.2.
- Docs/storage inventory: design0.04; implementation0.2 ×0.4 =0.08.
- Boundary review: design0.08; implementation0.5 ×0.4 =0.2.
- Real API/cost/live qualification: implementation0.6 ×0.4 =0.24.

Design subtotal2.02, thorough-plan buffer15%=2.323; implementation subtotal2.56; total4.883 focused ship-hours. Familiarity is1.0 except the explicit native-adapter factor above. Coordination waiting and full-suite queue time are not estimated as active work; actual measurement owns their classification.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.12
item: api-integration design=0.20 impl=0.48
item: greenfield-go-module design=0.30 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: api-integration design=0.30 impl=0.60
item: smaller-go-module design=0.10 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.08 impl=0.20
item: real-api-discovery design=0.00 impl=0.24
design-buffer: 0.15
total: 4.883
```


2026-10-15: change-code passed plan-quality CLEAN and estimate-quality INFO (design checkpoint committed by SDLC). Implemented native, pure model, root publication/inventory, bounded storage/history, cached CLI and broker alert seams with targeted regressions. Combined targeted Resource|ProductionArtifact run passed all6packages. Native SDK bulk projection fixes measured cgo overhead; full near-cap persistence measurement and TL-authorized revised<0.2% qualification are recorded in plan Revisions. Estimate issue/spec design covers already-incurred shaping, not remaining work; review and API integration allowances include qualification execution/landing, and actuals determine idle classification. Full suite, smoke, boundary review and landing remain outstanding.

2026-10-15: Near-cap cost requalification after TL budget clarification passed: mean68.414ms/max69.011ms CPU,0.114023% at60s; ~1008hostrows,27usageobservations,58nativeattempts and2publications/~4.19MiB perpass,3passes. Targeted worker race tests passed. Real-broker regressions prove hub self-delivery with the original worker disconnected, queued admission once, persisted-intent-before-send, restart reconciliation, and conservative unknown suppression. Design review corrections also enforce closed resources request fields, generation-unknown retention, runtime episode caps and alert offender names.

### 2026-10-15 — TL sequences live acceptance after close review

Receipt5e56f410-ff46-4aad-b623-660fc4da8ca7: pair:0 remains staged for#544, with#569 awaiting the same restart. TL directs finish recorded suite and close review now, then **hold**. TL will combine#544/#569/#587 for one Couch restart after#584, probably tomorrow morning, with cadence2s/write-duration6s and a designated scratch slot. Only then run all-slot visibility, one write alert, sustained silence, recovery and re-arm. No staging/restart or landing is authorized before that acceptance. The live Done-when remains outstanding; if close requires completed plan checkboxes, use its precise plan-check acknowledgment for this explicitly deferred acceptance, not a fabricated pass.

### 2026-10-15 — close review rework

Round1 returned REWORK (BR-1 through BR-5). Added real-socket maximum-cardinality projection regression (unbounded version fails with response exceeds frame limit); bounded version passes with every slot aggregate retained and exact detail omission counts. Independent checkpoint/history recovery, discovery-barrier/cap evidence and pure delivery transitions have deterministic RED/GREEN regressions, including the real broker. Corrected unsupported adapter traceability. Combined Resource/ProductionArtifact checks passed procutil/core/message/tty/artifactpath; the couchcmd old full-detail expectation was revised to require bounded output plus exact omitted counts, then all couchcmd Resource tests passed. Canceled only this slot's obsolete queued pre-rework suite (exit130 before execution); fresh recorded suite follows this code commit. No full-suite pass claimed.

Near-cap full sampler cost requalified after projection/recovery changes: 3 passes, 1074 host rows, 58 native attempts, two ~4.19MiB snapshot publications; CPU mean84.759ms/max86.328ms,0.141264% of one core at60s, within approved<0.2%. Thread-CPU accounting limitation remains disclosed. Live smoke/staging/restart and landing remain held for TL's combined acceptance.

### 2026-10-15 — second close review

Round2 accepted BR-1..BR-5, but returned REWORK for BR-6 (checkpoint byte envelope) and BR-7 (complete entity-table traceability). The full table sweep is committed in7070d47e5. A production-shaped maximum checkpoint fixture has8192processes,16896episodes/attempts and256escapedworkspaceidentities, using SHA256 IDs/generation tokens; raw JSON is32,895,161bytes. Plain compression still exceeds2MiB, so typed dictionary normalization is required before bounded compression. No alert authority may be evicted. Maximum-cardinality CPU must be measured separately from the already-passing normal-host<0.2% measurement. Canceled only our superseded queued suite before it ran (exit130); no suite pass claimed, no live changes.

### 2026-10-15 — TL-directed simplification, awaiting revised-scope approval

Receipt e97f725a supersedes durable checkpoint/intent requirements: alerts are advisory, episode state stays in memory, restart re-arms, and only bounded best-effort 24h history remains durable. The detailed replacement scope is appended to the plan's Revisions. All checkpoint optimization work stopped; no simplification code yet. The steady sampling budget remains well under1% (latest normal-host test69.730ms mean,0.116217% at60s). After approval delete checkpoint/intent/codec machinery, retain simple broker admission plus pure in-memory suppression, prove history failures cannot block alerts, and report exact net line count before rerunning recorded suite/close. Live smoke and landing remain held.
