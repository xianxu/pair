---
id: 000587
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'bcb557e6aef0c1b7fb31171256e28df9f80cdedc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T16:54:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

- [ ] Counted Darwin resource reader with real owned-child conformance.
- [ ] Pure generation-safe attribution, rates, provenance and alert episodes.
- [ ] Authenticated full-slot roots and supervisor-owned sampler lifecycle.
- [ ] Bounded restart-safe history and existing broker/hub delivery.
- [ ] Cached resources CLI, configuration, docs and storage registration.
- [ ] Targeted/full qualification, measured overhead, operator-coordinated live scratch-slot alert, SDLC review and landing.

## Log

### 2026-10-15


2026-10-15: Claimed and started design. Read ops resource-observability project and issue. Read-only seam review found authenticated Zellij server evidence already available as PairSessionBinding.Owner.Server; wrapper-only ancestry would omit draft editors. Preserve that evidence at admission/lifecycle boundaries rather than re-running Zellij during sampling. Native table currently hides an unbounded ENOMEM retry loop, so syscall accounting must live at the native boundary. Full design and proposed defaults are in the durable plan; awaiting TL approval before change-code. No implementation or live-slot changes made.

## Revisions

### 2026-10-15 — proposed implementation contract for TL approval

Make sampling limits explicit: orphan attribution requires a previously observed generation/provenance; children born and orphaned entirely between samples and pre-upgrade unknown orphans cannot be proven. Include every known slot row, with partial/unavailable coverage where root or counter evidence is missing. Propose 60s cadence; 2 MiB/s for10m writes, 8 GiB process RSS, 90% one-core idle CPU for15m, 256 MiB orphan RSS over two valid samples. History is <=24h/1440 aggregate frames/4 MiB, plus <=2 MiB ownership/episode checkpoint, with shorter coverage disclosed at the byte cap. Count actual native attempts and persistence cost; qualify <0.1% average one-core cost at default cadence. Darwin uses the SDK libproc API through cgo; unsupported builds report that explicitly. These choices require TL plan approval, and live smoke staging remains separately coordinated.


### 2026-10-15 — TL plan approval and unattributed addition

Plan bf764e14 approved (receipt9f23fae9-2182-488f-8025-196b3940fa66). Add report-only unattributed (unproven) large PPID1 processes matching known Couch-launched kinds, including nvim --embed, Zellij and agent CLIs; no alert or slot attribution without provenance. Candidate argv probes are bounded and counted in the cost envelope. Continue to change-code.
