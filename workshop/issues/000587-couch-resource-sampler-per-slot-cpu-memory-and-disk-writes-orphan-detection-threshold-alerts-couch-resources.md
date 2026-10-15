---
id: 000587
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '98275dfa3b900eff074652587ae1c981c76f5bfa' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
