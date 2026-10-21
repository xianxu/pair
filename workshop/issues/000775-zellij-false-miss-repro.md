---
id: 000775
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3f53eb28efe308678bcbabeb5f56586d12b8439a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T10:56:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "db8c8a6c", done: "1839b87d"}
---

# Standalone Zellij false-miss reproduction

## Problem

The operator needs a small upstream-ready reproduction of Zellij 0.45.1 false session misses from #761, without Pair or Couch.

## Spec

Add scripts/repro/zellij-false-miss.sh and a ten-line README section. Bash plus stock shell utilities and Zellij 0.45.1 only. Create three private background sessions with temporary HOME, XDG, config and socket roots; run N raw reads with configurable parallelism (default1000 reads/four workers), mixing native pane/client/session-list reads as in #761. Count exact exit0 target-not-found plus a listing containing the target, and exit1 no-active-session while target liveness is independently observed. Preserve raw anomalies and print counts; no read retries hide failures. Trap cleanup touches only the private sessions/root. Validate one real run on this machine and report both counts. No product code or global configuration changes. ARCH-PURPOSE keeps this diagnostic separate from product recovery.

## Done when

- Standalone script and ten-line README section committed on an independent main-based quick-flow branch.
- Syntax, invalid-input refusal and one native three-session run verified; report N, parallelism, both counts and cleanup result.

## Plan

- [ ] Extract private native stimulus and strict raw signature counts; document invocation and liveness limits.
- [ ] Run syntax/refusal checks and one real1000-query/four-worker experiment, retain result in Log, then close and prepare PR independently of #744.

## Log

TL c4915544 dispatches this tiny tooling side task while #744 integration suite waits. Use a separate checkout so the retained #744 merge runner sees no branch or working-tree changes. Quick plan follows the exact dispatched contract; proceeding under tl-worker quick-flow authority.
