---
id: 000431
status: open
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '767b4558a4a7c52181555a6b9c273576a73544bc' # card fields mirrored from issue-cards; edit via sdlc
---

# session ledger launch records embed a whole-store baseline: every QuerySession parses MBs

## Problem

Found during pair#420 (pair:3, 2026-10-10). Every launch record in a thread's
session ledger embeds `LaunchArtifactBoundaries`: one entry per artifact in
the agent's whole native store at launch time. The astro thread's launch has
3150 entries, and its ledger is 1.5MB. The ledger is append-only, so it grows
by a snapshot of the entire store on every launch (ARCH-FUNERAL: residue that
grows per launch with no bound).

Every `sessioninventory.QuerySession` parses the whole ledger strictly
(`readOwnerLaunch` → `sessionledger.ParseLedger`). After pair#420 removed the
native-store walks, this parse is about 74% of a `pair title` tick: a pprof of
`SessionActivity` on 1-pair-7 (44 launches) shows `ParseLedger` at 1.67s of
2.37s over five calls, about 0.35s of CPU per query. The title poller,
Couch's idle fading, `ContextCount` and the session watcher's per-pass
`readCurrentLaunch` all pay it.

## Spec

## Done when

- Launch-record size no longer scales with the native store, or the current
  launch is found without decoding every earlier launch's baseline.
- A pprof of a title tick on a many-launch thread no longer shows the ledger
  parse.

## Plan

- [ ]

## Log

### 2026-10-10

- Filed by pair:3 from the pair#420 measurements. Not implemented.
