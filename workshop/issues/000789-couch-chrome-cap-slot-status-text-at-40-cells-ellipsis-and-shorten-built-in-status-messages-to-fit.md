---
id: 000789
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '43e87df2e38780a5782a9d2d74f999d010860cad' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch chrome: cap slot status text at 40 cells (ellipsis) and shorten built-in status messages to fit

## Problem

## Spec

Operator, accepting pair#779 (10-21 19:1x): "one caution is we should limit the status text to be certain length,
otherwise, a long text will wrap into 10 lines, destroy the layout? maybe limit status message to be 40 chars,
maybe another task to clean up messages showing up there to be within that limit".

pair#779 caps the tab bar at two rows, but a slot's status text in the chrome can still be arbitrarily long and
crowd out every other tab. Two parts:
1. **Display cap:** the chrome renders a slot's status text at most 40 display cells (rune-width aware), cut with
   `…`; the full text stays available where it already is (peek, the switcher or hover). One pure function,
   unit-tested, used by every place the chrome draws status text.
2. **Producer clean-up:** list the status-text producers (agent status lines, wait notices, lifecycle labels) and
   shorten their messages to fit 40 cells so the cap rarely bites; a test fails if a built-in status string exceeds
   it.

## Done when

- A 200-character status renders as 40 cells ending in `…`, with layout unchanged (test); full text still reachable.
- No built-in status message exceeds 40 cells (test over the catalog).

## Plan

- [ ]

## Log

### 2026-10-21
