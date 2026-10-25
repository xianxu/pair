---
id: 000879
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'dfe3c3f65bb8fd7fef1b673fc6ee52cbd5d2ad22' # card fields mirrored from issue-cards; edit via sdlc
---

# Reusable upgrade test: candidate build adopts, lists and resumes warm sessions created by main

## Problem

## Spec

**A reusable upgrade test: the candidate build must start against warm sessions created by main (operator request,
10-24).** On 10-24 a combined stage (pair:0 at 93a57a38, with pair#866) couldn't attach to the operator's existing
warm threads: #866 refused to list any Zellij session without its new readiness marker, which every pre-#866 session
lacks, so resume failed before attach ("missing zellij-ready-v1 marker"). The operator had to revert pair:0 by hand.
Every item had passed its own tests; none started the new build against sessions from the old one.

Build a reusable native fixture: a disposable fleet started by the main build (Claude and Codex actors, hubs and
workers), then the candidate build takes over: startup adoption, listing, explicit resume and attach of every session,
plus a persisted-state read (registries, thread store, side-doc records) created by main.

## Done when

- The upgrade fixture exists and runs headless within bounds; it reproduces the 10-24 failure against 93a57a38 (RED)
  and passes against the corrected pair#866.
- Changes to live-session or persisted-state contracts are required to pass it (documented in the AGENTS.local or
  atlas test guidance and referenced by the close review).
- The combined-stage procedure (ariadne#600) can call it as the warm-start check.

## Plan

- [ ]

## Log

### 2026-10-24
