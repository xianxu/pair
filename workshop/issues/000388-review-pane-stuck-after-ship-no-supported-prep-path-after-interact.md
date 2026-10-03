---
id: 000388
status: open
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: '4da1e2642e0fbfc9eda82e7e352f7a98c8cce881' # card fields mirrored from issue-cards; edit via sdlc
---

# review pane stuck after ship; no supported prep path after interact

## Problem

Two ways the next review in the same pair session gets stuck after a
docflow-shipped review. Observed 2026-10-02 in a xianxu.dev couch session
(ariadne-1 shipped, then preparing ariadne-2-hot-takes); only relaunching pair
cleared it.

1. **Stale `landed` blocks the next review.** `nvim/review/restore_controller.lua`
   (~L103-119) keeps `self.landed` from the last applied round and clears it only
   after finding the matching agent-round commit via
   `git log refs/heads/<review-branch>`. `docflow ship` merges `--no-ff` and
   deletes that branch, so the lookup fails forever and Alt+c reports
   "applied round not yet committed; finish it on the original branch first",
   even though the round is committed and merged into the base.

2. **No supported prep path after `interact`.** `pair review readiness --prepare`
   refuses on `interact` (dirty tree off a review branch). After the operator
   chooses "carry edits onto the review branch", the agent runs `docflow start`
   and `pair review target <abs> ready`, but that target has no `identity`
   receipt. With no rounds yet, `restore_client.lua` `missing()` falls back to
   probing the live pane, which still carries the previous (shipped) review's
   activation, and fails: "live review selection could not be authenticated;
   finish or reselect it". Committing the draft and re-running `--prepare`
   (`resume`) is the only path that writes a receipt.

Related: #385 (skill still names the pre-#104 helper commands).

## Spec

## Done when

- After `docflow ship` deletes a review branch, a pane whose landed round is
  in the base branch's history (e.g. reachable from the merge) clears and lets
  the next review open without relaunching pair.
- There is a supported way to finish preparing after an `interact` choice
  (e.g. `--prepare --carry` that runs `docflow start` and writes the identity
  receipt), and `pair review target … ready` never produces a receipt-less
  target that the restore path can't authenticate.
- Tests cover ship-then-next-review and interact-then-carry.

## Plan

- [ ]

## Log

### 2026-10-02
