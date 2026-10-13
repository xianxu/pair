---
id: 000513
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4ca149c8078142a6a0e9277502449f927c0da3d2' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: a video element on the cast page is missing in the viewer (the space around it collapses)

## Problem

The operator, 10-12 ~18:30, casting the parley blog in the cast Chrome and watching in Safari: after "How does it work? One video is worth a thousand words. Take a look." the cast page shows a demo video (a poster of a terminal session, a play button, a caption "navigate away while waiting for answer"). The viewer shows nothing there: the next heading ("What Parley isn't") follows directly, with no gap, so the element is either not recorded, not rebuilt, or rebuilt with zero size. Screenshot: the TL scratchpad image 117 (left: cast Chrome; right: Safari viewer).

## Spec

Find where the video is lost (recorder snapshot or mutation, asset fetch for its poster or source, or the viewer's rebuild), with a test that reproduces it from a page holding a `<video>` with a poster (and, if the blog uses one, the embed's actual markup). Fix so the viewer shows at least the element at its size with its poster frame; playing the video in the viewer is out of scope unless it falls out. Check the other replaced elements the same way (`<iframe>`, `<canvas>`, `<picture>`/`srcset`, `<object>`) and fix or file what the test finds. Measure the real page first (the operator's cast tab, read-only; never steal focus).

## Done when

- The parley blog's demo video shows in the viewer at its place and size, with its poster.
- A test with a video page fails before the fix; the replaced-element sweep is recorded in the Log.

## Plan

- [ ]

## Log

### 2026-10-12
