---
id: 000161
status: working
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
started: 2026-09-01T13:35:54-07:00
---

# Couch misses Codex completion notifications

## Problem

During Couch dogfood testing, Codex completed work in Pair but Couch did not
surface a completion notification. No notification appeared in either the
status bar or the switcher.

## Spec

- Surface Codex completion notifications for Pair sessions in Couch's status
  bar and switcher.
- Keep the two surfaces consistent so a completed agent is not silently missed.

## Done when

- A Codex completion in Pair produces a visible notification in the Couch
  status bar.
- The same completion is visible in the Couch switcher.
- Automated coverage reproduces the missed-notification path and prevents a
  regression.

## Plan

- [ ] Reproduce and identify where the Codex completion event is lost.
- [ ] Add a failing test for the observed Pair/Codex completion path.
- [ ] Restore notification propagation to the status bar and switcher.
- [ ] Verify both surfaces from completion through display.

## Log

### 2026-09-01

Captured from initial Couch dogfood testing. Codex finished work in Pair, but
no notification appeared in the status bar or switcher. Investigation was
explicitly deferred to issue implementation.
