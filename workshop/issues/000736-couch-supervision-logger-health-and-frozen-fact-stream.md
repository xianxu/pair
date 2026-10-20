---
id: 000736
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c7a2c631ab90b441855fc55a3c12911a04ac47b1' # card fields mirrored from issue-cards; edit via sdlc
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

- Inspect and expose logger health before changing recovery behavior.
- Reproduce the observed live failure and apply a tested cause-specific fix.

## Log

### 2026-10-20

- Split from pair#735 by ops:0 approval. Start after #735 lands. Investigation scratch probe used a Go overlay in /Volumes/TestScratch/pair735-log-probe-75gnrui4; ephemeral evidence only, not required for future reproduction.
