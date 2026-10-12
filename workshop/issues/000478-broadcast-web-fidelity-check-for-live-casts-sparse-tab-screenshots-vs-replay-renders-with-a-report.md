---
id: 000478
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '8837b07abcdb2cf0059bd9e1a2b7e6667119056d' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: fidelity check for live casts — sparse tab screenshots vs replay renders, with a report

## Problem

Principle 6 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`): fidelity is measured, not assumed. #453's harness and #458/#460 measured fixtures headless; a live cast has no check, and the recording format reserves `frames/` (sparse low-res screenshots of the real tab) for exactly this.

## Spec

- During a live cast the recorder takes sparse low-res screenshots of the tab (on navigation and every few seconds; `frame` records in `frames/`, never streamed), cheap enough not to disturb the page.
- A check renders the replay at the same moments (headless, #460's renderer) and compares, reusing #453's diff; a report per recording (median/max, the worst moments with side-by-side crops).
- Run it from the CLI on any recording and automatically at the end of a cast; desktop Chromium targets exact fidelity; record which moments exceed the tolerance so fidelity bugs become issues.

## Done when

- A live blog cast produces frames and a fidelity report within the stated tolerance; a deliberately broken render is caught.
- Tests; the report format documented in the atlas.

## Plan

- [ ]

## Log

### 2026-10-12
