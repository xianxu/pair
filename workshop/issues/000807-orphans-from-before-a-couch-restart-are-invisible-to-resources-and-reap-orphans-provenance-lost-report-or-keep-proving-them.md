---
id: 000807
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5441da742c1924be02be75eb0e17e1cc4ab6153e' # card fields mirrored from issue-cards; edit via sdlc
---

# Orphans from before a Couch restart are invisible to --resources and --reap-orphans (provenance lost); report or keep proving them

## Problem

## Spec

pair#792 (landed 10-22 09:30) reaps proven orphans of a live slot's previous incarnation. Live retry by ops-echo:0 at
~10:56, on Couch built at c95f2421 (contains #792): `couch --reap-orphans` preview reports "none observed", and
`couch --resources` lists **no proven and no unattributed** processes, yet echo:2's 10-21 leftovers are alive:
zellij server 66561 (ppid 1, session 1-392), `pair wrap` 66575 and `pair term` 66576 under it, and Claude 66695 under
66575, started as an echo:2 "This slot restarted fresh" relaunch. On 10-21, before several Couch restarts,
`--resources` did list exactly these as proven orphans of echo:2.

So after a Couch restart, the previous incarnation's provenance is gone and its orphans become **invisible**: not
proven (correctly not reaped) but also not even reported as unattributed, so no one knows they're there.

Fix: either persist the bounded provenance needed to keep proving them across a Couch restart (the existing
identity fence, PID plus start time plus slot and tag), or at minimum report such processes as **unattributed**
(Couch-shaped trees: zellij servers in Couch's socket dir, pair wrap/term, agents with Couch markers) so a TL sees
them and the operator can decide. Never reap unproven processes automatically.

## Done when

- A fixture with a previous-incarnation tree surviving a Couch restart is either still proven (and reapable through
  the #792 path) or listed as unattributed, never invisible (test).

## Plan

- [ ]

## Log

### 2026-10-22
