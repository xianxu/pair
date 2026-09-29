---
id: 000348
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: 'cbd2c09a8b5e560e5147d28712d32ba37b1ff0d5' # card fields mirrored from issue-cards; edit via sdlc
---

# Restore exhaustive review artifact ownership coverage

## Problem

The exhaustive artifact-reference test fails on review code already present at merge-base `b622052f`, before #346. After classifying #346's new recovery/handshake sources, all 32 remaining diagnostics exactly match an isolated checkout of that base with its runtime bundle generated. This prevents an honestly green full suite.

## Spec

Classify the new review modules and generated mirrors, and exact diagnostic-string vocabulary from #341. Move the `review-recovery` path construction at `nvim/review.lua` into the existing artifactpath authority. Do not label a real path construction as non-path vocabulary or weaken the exhaustive scanner. Preserve review recovery behavior and generated-runtime parity.

## Done when

- `go test ./cmd/internal/artifactpath -count=1` passes with exhaustive scanning unchanged.
- Review sources and generated mirrors have accurate classifications.
- Review recovery path is supplied by the artifact owner, with integration coverage preserving recovery snapshots.
- Runtime generation and the review integration suite pass.

## Plan

- [ ] Fix ownership and classifications, regenerate runtime, and verify artifact and review suites.

## Log

### 2026-09-29

Discovered during #346 close verification. Current evidence: `/tmp/pair346-artifact-inventory-check.log`; matching generated baseline: `/tmp/pair346-artifact-inventory-baseline-generated.log`. Baseline generated from `git archive b622052f` in `/tmp/pair346-inventory-baseline-3vspad5m`. No production fix made as part of #346.
