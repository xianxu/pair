---
id: 000366
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '99e83261a6bae077f3240474f2c278840b5aa638' # card fields mirrored from issue-cards; edit via sdlc
---

# Make Couch a local singleton

## Problem

Durable assignment should identify machine + slot without a Couch instance ID. Today namespace-scoped supervisors require an explicit singleton/migration contract before that assumption is valid.

## Spec

Project: pair/workshop/projects/cross-slot-work-scheduling.md. Captured for operator review; no implementation is authorized by this issue creation.

Make Couch a local singleton with one authoritative slot inventory. Proposed boundary is one supervisor per OS user on a machine; validate and settle that boundary in design. Multiple Ariadne operators/machines may independently run their own local Couch and share issue trackers. Multi-machine Couch routing is out of scope.

Define existing-namespace migration/adoption, second-launch behavior, crash restart and slot identity preservation. Reuse supervision/lease mechanisms and ensure tests/isolated diagnostic stores have an explicit supported isolation path. Do not delete existing conversations, reset worktrees or silently adopt conflicting state.

## Done when

- The singleton boundary is explicit; concurrent starts cannot establish two production supervisors inside it.
- A second invocation attaches/routes to the existing supervisor or gives a concrete diagnostic.
- Existing stores and stopped/live slots migrate or are reconciled without losing conversations, preferences or dirty work.
- Restart and isolated-test behavior are verified; durable issue ownership requires no Couch ID.

## Plan

Implementation plan to be designed after issue claim and start-plan; these are requirements, not an approved implementation plan.

## Log

### 2026-10-01

Captured from the performance → messaging guarantees → SDLC ownership/observability → recovery discussion. No implementation started.
