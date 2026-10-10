---
id: 000422
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '8514de997b185b1e345d2fc1d6be11277ec4a66e' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: propagate PAIR_DEV to slots (or a couch dev entry) so relaunch rebuilds under couch

## Problem

Found by TL ariadne:1 on 2026-10-09 during the #418 rollout. Alt+n (relaunch)
rebuilds Pair only when `PAIR_DEV=1` (`bin/pair-dev` exports it, and
`bin/lib/dev-rebuild.sh` runs the gated `make build`). The Couch server (a Go
binary, never launched through `pair-dev`) and every slot it starts run without
`PAIR_DEV`, so a relaunch under Couch never rebuilt. The slots kept the stale
`bin/pair` until a manual `make build`.

## Spec

Either Couch propagates `PAIR_DEV` from its own environment to the slots it
launches, or there is a `couch` dev entry (as `pair-dev` is for `pair`) that
exports it. Relaunch (#421's `couch --relaunch`) then rebuilds through the
existing `dev-rebuild.sh` path.

## Done when

- A Couch started in dev mode launches slots whose Alt+n and `couch --relaunch`
  run `dev_rebuild`; a deployed Couch stays a no-op.

## Plan

- [ ]

## Log

### 2026-10-09
