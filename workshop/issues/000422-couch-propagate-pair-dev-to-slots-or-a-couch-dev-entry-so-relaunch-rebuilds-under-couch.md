---
id: 000422
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: '122bb2a71751d384aa2316d965a79fe2fbd54cbe' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T23:05:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "9556cadd", done: "397aad23"}
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

Design (2026-10-09, pair:2): **a `couch-dev` entry**. No propagation code is
needed: slots already inherit Couch's environment (`couchcore`
`mergeChildEnvironment` starts from `os.Environ()`, and `couchcmd`
`configuredRunner.childEnv` blanks only `PAIR_*_PATH`). The gap is that nothing
starts Couch with `PAIR_DEV` set.

- `bin/couch-dev` mirrors `bin/pair-dev`. It resolves its real directory,
  exports `PAIR_DEV=1`, runs `dev_rebuild` from `bin/lib/dev-rebuild.sh` (so the
  couch binary itself is fresh, ARCH-DRY: one rebuild hook), then execs the sibling
  `couch` with every argument.
- Install wiring: `SHELL_BINS += couch-dev`, plus a `.gitignore` negation so the
  script is tracked (the blanket `bin/*` ignore).
- A deployed `couch` is untouched: no `PAIR_DEV`, so `dev_rebuild` is a no-op.
- Test: `tests/dev-rebuild-test.sh` gains couch-dev cases with fake `make` and
  `couch`: `PAIR_DEV=1` reaches the exec'd couch, args forward, and a build
  failure still execs. A Go test pins that a Couch child inherits `PAIR_DEV`
  through `childEnv`, because that inheritance is load-bearing.
- #421 (`couch --relaunch`, pair:1) relaunches through the launcher's create
  path, so it inherits the env. I asked pair:1 to keep `PAIR_DEV` if they build a
  custom env.
- ARCH-FUNERAL: creates nothing durable. It is a launcher script and an env var.

## Done when

- A Couch started in dev mode launches slots whose Alt+n and `couch --relaunch`
  run `dev_rebuild`; a deployed Couch stays a no-op.

## Plan

- [x] Tests: couch-dev shell cases plus the Go childEnv inheritance pin (red first).
- [x] `bin/couch-dev`, `.gitignore` negation, `SHELL_BINS`.
- [x] README dev note and atlas (architecture dev-mode bullet, couch).

## Log

### 2026-10-09

- Claimed from TL ariadne:1 (operator-approved dispatch); labelled the slot per
  the #419 skill step. Asked pair:1 to keep `PAIR_DEV` in #421's relaunch env.
- No Couch propagation code needed: the inheritance already holds and is now
  pinned by two Go tests. The work is the dev entry plus wiring:
  `.gitignore` negation, `SHELL_BINS`, the artifactpath inventory (pre-existing
  failures there are unrelated), and the install-layout test.
- Dogfood: `bin/couch-dev --list` ran `make build` on this tree, then listed
  threads.
- Process slip, repeated from #419: implemented before `change-code`. A lesson
  was added.
