---
id: 000422
status: codecomplete
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'ac8097c4b9b8ea1631873f2f46fbd58f96d7708f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-09T23:05:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "9556cadd", done: "397aad23"}
actual_hours: 0.30
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
- 2026-10-09: closed — Re-close after the close-review advisory fix (dev-rebuild.sh: messages say dev-rebuild:, header names couch-dev; comment/echo-text only). tests/dev-rebuild-test.sh all pass (incl. 3 couch-dev cases); keyhelp + runtimebundle bundle tests pass after make build. Prior evidence stands: Go inheritance pins pass, make -k test green in clean env, go test ./... residual fails pre-existing (artifactpath, gcruntime, pair#423 flake). CI conformance red = known runner CPR timeout pair#324 (main identical); TL ops:0 approved merging.; review verdict: SHIP
- 2026-10-09: closed — tests/dev-rebuild-test.sh: 3 new couch-dev cases (PAIR_DEV=1 + args reach couch via a symlinked entry; make -C tree build; failed build still launches) red before bin/couch-dev, green after. Go pins TestExecRunnerChildInheritsPairDev + TestConfiguredRunnerChildEnvKeepsPairDev pass. Install layout test asserts the couch-dev symlink, passes. make -k test green in a clean env (env -i; the two failures under the session env were PAIR/COUCH var leaks). go test ./... in clean env: artifactpath + gcruntime fail identically on main (artifactpath couch-dev entry added; remaining entries pre-existing); couchsingleton passes with default TMPDIR (fixture grows with the long scratchpad path); TestColdResume.../switcher is a 3/6 timing flake unrelated to env/launch scripts (logged). Dogfood: bin/couch-dev --list rebuilt the tree and listed threads. pair:1 confirmed #421 relaunch inherits PAIR_DEV.; review verdict: SHIP

- Claimed from TL ariadne:1 (operator-approved dispatch); labelled the slot per
  the #419 skill step. Asked pair:1 to keep `PAIR_DEV` in #421's relaunch env.
- No Couch propagation code needed: the inheritance already holds and is now
  pinned by two Go tests. The work is the dev entry plus wiring:
  `.gitignore` negation, `SHELL_BINS`, the artifactpath inventory (pre-existing
  failures there are unrelated), and the install-layout test.
- Dogfood: `bin/couch-dev --list` ran `make build` on this tree, then listed
  threads.
- pair:1 confirmed #421: `couch --relaunch` runs `Couch.Relaunch` (park, then
  ResumeContext) through `buildExecCommand` with `mergeChildEnvironment`, so
  `PAIR_DEV` reaches the relaunched pair and `dev_rebuild` runs. When the slot
  has `PAIR_DEV`, #421 skips its stale-binary refusal.
- Unrelated flake found: `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins/switcher`
  fails about 3 in 6 under load. It presses Enter after the "threads" header but
  before the row loads, giving "no selection". It passed on clean main once.
- Close review (SHIP) had 2 advisories:
  - Fixed: `dev-rebuild.sh` named only pair-dev. Its messages now say
    `dev-rebuild:` and "fix, then relaunch", and the header names couch-dev.
  - Declined: the DRY finding about couch-dev's symlink loop. Each entry script
    needs that loop to locate `bin/lib`, so a helper kept in `bin/lib` can't
    supply it.
- Landing: PR #222 CI `conformance` was red on `TestNativeConsoleWrapperZellij`
  (its direct-zellij-baseline subtest times out waiting for CPR). main fails the
  same way: the known runner CPR timeout, pair#324. `merge-check` passed. TL
  ops:0 approved merging despite the red check.
- Process slip, repeated from #419: implemented before `change-code`. A lesson
  was added.
