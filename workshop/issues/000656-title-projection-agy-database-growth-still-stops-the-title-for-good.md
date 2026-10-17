---
id: 000656
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: 'dfb964bd664cc8853a01aa05a70ff457a03b3716' # card fields mirrored from issue-cards; edit via sdlc
---

# Title projection: agy database growth still stops the title for good

## Problem

## Spec

Found in the pair#653 close review. The title projection (cmd/internal/sessioninventory/title_projection.go, checkArtifacts) requires an unchanged fingerprint for every non-transcript artifact. For agy that is its database, which grows during normal work, so that growth still stops the title for good, the same failure class #653 fixed for Claude/Codex transcripts. Classify same-identity growth of p.other as transient, the same way grewFrom does for the transcript, in both admission and tail.

## Done when

- A test where an agy database grows during admission and during tail fails on current main and passes with the fix; shrink, replacement and same-size mutation stay terminal (TestTitleProjectionAgyDatabaseMutationIsTerminal keeps passing).

## Plan

- [ ]

## Log

### 2026-10-17
