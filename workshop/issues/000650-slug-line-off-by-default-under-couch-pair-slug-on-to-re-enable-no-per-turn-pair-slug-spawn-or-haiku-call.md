---
id: 000650
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '96454768670845bf932713d511de05734bb3b502' # card fields mirrored from issue-cards; edit via sdlc
---

# Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call

## Problem

## Spec

Operator decision, 2026-10-17. The draft's slug line (line 1, a summary of what the slot is doing) is off by default when the wrapper is Couch-hosted. Measured by pair:2: about 70 `pair slug` spawns and about 36 Haiku calls an hour across the fleet, with ~570 MB transient memory peaks per call and about 3% of a core on average. The operator doesn't use the slug under Couch.

- When `launcher.CouchHostedEnv` is true, `wrap.go`'s `maybeSpawnSlug` returns before spawning anything: no `pair slug` process and no model call.
- `PAIR_SLUG=on` turns it back on under Couch. Outside Couch, behaviour is unchanged.
- Help, the README and the atlas say so.

## Done when

- A test: Couch-hosted env spawns nothing; with `PAIR_SLUG=on` it spawns; outside Couch it spawns as before.
- After the next restart and relaunch, no `pair slug` or `claude -p` processes appear across the fleet during normal turns. Checked live.

## Plan

- [ ]

## Log

### 2026-10-17
