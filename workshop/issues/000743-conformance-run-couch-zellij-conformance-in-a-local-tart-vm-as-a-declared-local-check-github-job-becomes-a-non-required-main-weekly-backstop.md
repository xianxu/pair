---
id: 000743
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9bd068ef97388118ca0169e76e2079433f052dee' # card fields mirrored from issue-cards; edit via sdlc
---

# conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop

## Problem

`couch-zellij-conformance` (real Zellij 0.45.1, nvim and PTYs on a clean macOS runner) is a required PR check for
Couch, terminal, launcher and Zellij paths. On 10-20 GitHub's macOS arm64 capacity cancelled it three times before
any runner started ("not acquired by Runner of type hosted"), gating pair landings (#726, #732, #735) on GitHub's
capacity instead of on our tests. The operator's rule is local-first verification.

## Spec

- **Run it in a Tart VM locally** (pair's `.tart/Makefile` already boots `pair-test` headless and clones the repo in):
  a non-interactive `make conformance-vm` boots the VM, a `.tart/vm-hooks.d/` hook installs the pinned Zellij 0.45.1
  (checksum-verified, as the workflow does) and nvim, runs `make -f Makefile.local test-native-terminal-ci` and
  `test-couch-zellij-live`, copies the logs out (including Zellij's), stops the VM, and exits with the tests' status.
  Refresh the `pair-test` image first (last used four months ago).
- **Declare it as a local check** `conformance` in `.sdlc/local-checks.json`, required only when the diff touches the
  workflow's path list (one source for the list, shared with the workflow), heavy weight (a VM plus the tests), recorded
  on the PR head like `suite`.
- **GitHub job demoted (operator-approved 10-20):** keep it on main pushes and weekly as a clean-machine backstop,
  but no longer required to merge.
- Guard: the VM run can never touch the live Couch: no shared Zellij socket, HOME or Couch store with the host.

## Done when

- `make conformance-vm` passes on current main and fails on a known-bad fixture; its isolation is asserted (no host
  socket, HOME or store visible in the VM).
- `sdlc local-check run --name conformance` records on a PR that touches the paths, and is skipped on one that doesn't.
- Local-gates change: the operator verifies and applies `local-gates-verified` (never the worker).

## Plan

- [ ]

## Log

### 2026-10-20

## Revisions

- 2026-10-20 13:30 (TL, from the operator's decisions in ops side doc `2026-10-20-19-conformance-structure.md`):
  - **Tier 3 gate:** the VM conformance runs **once per landing train** when any car touches the paths, not per
    close; it starts as a **nightly** run (off-hours) and is promoted to blocking after about a week clean. Any flake
    is fixed at its root, never retried.
  - **One VM, via a machine-wide lock:** macOS allows two VMs; tier 3 takes one through a lock like the suite lock,
    leaving the other for ad-hoc runs (which take the lock's other position).
  - **Every agent kind:** agent-facing cases run against Claude and Codex (Qoder when it has credits).
  - **Pair first** (operator); ariadne and parley later.
  - Uses the shared disposable-Couch harness (pair#744) for its live cases.
