---
id: 000405
status: open
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'fcf754f35a7cead00035e88f56aa4fbfbc32d10f' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Apply the highlight through the existing row renderer and verify the capture/status tests and build.

## Log

### 2026-10-07
