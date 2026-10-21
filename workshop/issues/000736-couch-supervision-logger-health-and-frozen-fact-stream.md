---
id: 000736
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '17f9ff00b1326f7605c0065cae8f54fc65b9c7be' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T20:02:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch supervision logger health and frozen fact stream

## Problem

During pair#735 investigation, Couch's supervision facts at ~/.local/share/pair/couch/supervision.jsonl were confirmed frozen since 2026-10-19 15:56:04 (26,061,670 bytes). Archived diagnostic segments do exist; there is no replacement per-message readiness-decision log. Existing facts record slot snapshots and completed hub deliveries.

The logger discards its Open error and becomes permanently unavailable for that supervisor lifetime. Write errors increment a dropped counter without exposing the error. Current metadata matches the frozen file. Scratch-copy Open/Write probes pass, both with writer registrations removed and retained, so the historical failure cannot be attributed from disk. Live writer PID/start matches metadata. No live files were mutated during investigation.

## Spec

Expose logger health and a bounded last error through an existing Couch status surface first. Observe the actual live refusal, then fix that cause. Do not add speculative retries. Keep diagnostics optional, bounded, and off the interactive admission path. Per-message holds remain visible in retained receipts; document the logging and receipt locations accurately.

## Done when

- Existing status reports distinguish a working, unavailable, and failed-write supervision logger, with a useful bounded error and no message/prompt secrets.
- Tests cover startup Open failure, subsequent write failure/recovery, and closed state without changing admission outcomes.
- The live failure is observed and recorded, its root cause is corrected, and new supervision facts are verified to reach storage.

## Plan

- [ ] M1 — Expose memory-only logger health in resources; test, review and land instrumentation without staging.
- [ ] M2 — Observe the actual refusal after the hub's coordinated restart; approve a cause-specific revision, reproduce it, fix it and verify fresh live facts.

Durable plan: [implementation plan](../plans/000736-supervision-logger-health-plan.md).

## Log

### 2026-10-20

- Split from pair#735 by ops:0 approval. Start after #735 lands. Investigation scratch probe used a Go overlay in /Volumes/TestScratch/pair735-log-probe-75gnrui4; ephemeral evidence only, not required for future reproduction.

### 2026-10-20 — health-first investigation

Read-only stat confirms supervision.jsonl still has 26,061,670 bytes and the
2026-10-19 15:56:04.739652 timestamp. Metadata names the running Couch PID10525;
resources identifies its start as 2026-10-20 10:32:08.391981. No live storage was
changed. Source confirms one unbuffered diagnosticlog.Open on the existing fact
worker; Open errors disappear, and factLogWriteFailed is a lifecycle no-op.
A bounded read-only subagent review confirmed resources as the existing fresh,
request-time status seam. Historical cause remains unknown.

## Revisions

### 2026-10-20 — separately land health before measured repair

The hub approved two landing boundaries: M1 health instrumentation lands through
normal review and merge, no pair:0 stage. At its coordinated restart after #742,
the hub reads couch --resources and supplies the observed failure. M2's precise
cause-specific implementation is revised only after that evidence, never guessed.
M1 does not close the issue or claim the historical logging failure is repaired.
The hub owns rebuilding pair:0, the restart and the project file.
