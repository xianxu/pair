---
id: 000524
status: open
deps: [pair#483]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'cfba515948b319088ac40c5fd017ee3e4be1329b' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture

## Problem

pair#483's smoke test (2026-10-12, the operator's blog): one of three crops (`crop-0.png`, the 'Couch: A Sneak Peek' post card) came out nearly black, text barely visible, while its box text was exact. The blog's cards use AstroWind's scroll-in effect (`intersect-once intersect-quarter motion-safe:md:opacity-0 motion-safe:md:intersect:animate-fade` in src/components/blog/ListItem.astro / GridItem.astro): they start at opacity 0 and fade in once in view. The capture caught the card at or near opacity 0. Bundle: ~/.local/share/pair/couch/webcast/submissions/9dc7b091…/crop-0.png.

## Spec

Find why the card was still faded at capture (measure first: does the CDP capture path re-render or restart the animation; is the cast Chrome throttling animations when not frontmost; is the intersection never reached). Then capture settled pixels: before each crop, wait (bounded) for animations/transitions intersecting the box to finish (`document.getAnimations()`), or force their finished state for the capture, and restore after. Never steal focus. Record in the crop's metadata when a capture couldn't settle.

## Done when

- The operator's blog card crop shows its text at full opacity; a test page with a fade-in on intersection reproduces the dark crop before the fix and passes after; a capture that can't settle is marked.

## Plan

- [ ]

## Log

### 2026-10-12
