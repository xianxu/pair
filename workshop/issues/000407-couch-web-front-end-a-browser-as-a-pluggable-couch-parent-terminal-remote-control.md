---
id: 000407
status: open
deps: [pair#395]
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'e3020de52510d5981df3877f97b836c16fbe282e' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch web front end: a browser as a pluggable Couch parent terminal (remote control)

## Problem

#395 makes a browser a view-only *output* parent for Couch: the hub taps
the Presenter's composed frames and an xterm.js page renders them. The operator's
framing (2026-10-07): a Couch web front end is just a remote-controlled
Couch. Adding the input half gives remote help (customer service, debugging
someone's setup) and, eventually, a browser as Couch's primary front end.

Overlaps #121 (remote Pair control relay), which has been claimed since
2026-07-26. Reconcile with it before designing.

## Spec

Open questions raised in #395's discussion:

- **Separate endpoint and grant.** Control runs on its own URL (WebSocket), with
  its own separately issued grant and explicit approval from the operator. A #395
  view-only link can never be upgraded to control. A capability link alone is too
  weak, because control means running code as the operator; it needs SSO or
  Cloudflare Access.
- **Geometry.** Couch has one host size. A helper takes the operator's grid
  (auto-fit font, as in #395). A browser that is Couch's primary front end would
  own the size, so the host becomes pluggable rather than a second viewer.
- **Keys.** Couch relies on enhanced keyboard reporting (Alt+d, Ctrl+Return;
  #279). Browsers capture some chords, and xterm.js has limited support for kitty
  keyboard reporting. Measure this before promising parity.
- Reuse the #395 hub. A control transport adds only the input direction.

## Done when

-

## Plan

- [ ]

## Log

### 2026-10-07

- Filed from #395's design discussion.
