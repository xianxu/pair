---
id: 000513
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '1ff24e58ad657f94ffaabd8f89f579704bff0fc3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:21:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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
- **TL, from the blog source (~/workspace/xianxu.dev, read-only):** it is not a `<video>`. The post has
  `<div class="cast-embed" data-cast="/casts/parley-nvim-v1.cast" data-poster="npt:0:41"></div>`;
  `src/components/blog/CastEmbed.astro` hydrates it lazily (IntersectionObserver, 600px margin) into
  asciinema-player 3.17.0 with `fit:'width'`, the player's CSS imported dynamically, plus a JS-placed caption
  overlay. The player renders `div.ap-term` = `<canvas>` (background and colour blocks via `getContext('2d')`) +
  `<svg class="ap-term-symbols">` + `<pre class="ap-term-text">` of spans. Check: canvas pixels (never in the DOM),
  the dynamically inserted stylesheet, and the lazy mutation after the first snapshot.

## Revisions

- 2026-10-12 (operator via TL): scope addition. Any content the recorder can't capture (canvas pixels, video frames,
  cross-origin iframes, plugins) shows in the viewer as a placeholder at its live rendered size, updated when the
  size changes, with a small label (e.g. "canvas: not captured"), so layout never collapses and agents know
  something is there (the rrweb/FullStory practice). A low-rate canvas snapshot (toDataURL when not tainted) is a
  follow-up, filed unassigned as #515, not built here.
