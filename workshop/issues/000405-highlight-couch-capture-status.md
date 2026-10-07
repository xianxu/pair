---
id: 000405
status: working
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: '1527714f4e2c2a6f2c128ef033e90d6d81115ef4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T14:04:38-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "19233212", done: "8871a7ef"}
---

# Highlight Couch capture status badge

## Problem

The opt-in capture indicator blends into the ordinary Couch status row. The operator requested a more obvious highlight while waiting for #379 to recur.

## Spec

Render every visible capture badge in bold reverse video, using the terminal foreground/background colors. Reuse the status row renderer’s scoped SGR/reset handling (ARCH-DRY); preserve label text, clipping, and click spans. Capture disabled still renders nothing. This is a presentation-only follow-up to #404.

## Done when

- Recording and stopped badges have a contrasting bold inverse highlight, scoped to the badge.
- Existing capture status, clipping, and actor click-span checks pass.

## Plan

- [x] Apply the highlight through the existing row renderer and verify the capture/status tests and build.

## Log

### 2026-10-07
- 2026-10-07: closed — Existing capture/status tests and Couch build pass. Scoped reset, empty badge and zero-width behavior verified by inspection in unchanged appendText. BR-1 withdrawal requested: session instructions prohibit new implementation-mirroring tests for this reversible one-line style change. No new architectural surface.; review verdict: SHIP

- Applied bold reverse video through the existing `appendText` style argument; its reset scopes emphasis to the clipped badge, including stopped states. No new rendering mechanism or terminal write path.
- Verified `go test ./cmd/internal/couchtty -run 'Capture|RenderStatusRow|StatusRow' -count=1`, `go build -o bin/couch ./cmd/couch`, and `git diff --check`. Tests ran with Pair/Couch/Zellij environment removed and a dedicated temporary root.

- Close review BR-1 requests new tests asserting the exact bold/inverse escape and reset. Requested disposition: withdraw. This reversible, one-line presentation tweak intentionally relies on the existing scoped renderer rather than adding implementation-mirroring tests. Session instructions explicitly prohibit writing tests for reversible low-impact changes or tests that mirror implementation. Inspection verifies `appendText` omits style for empty/zero-width output, clips before adding style, and emits its reset immediately after badge text; the existing state/width/click-span tests pass. No rendering logic changed.
