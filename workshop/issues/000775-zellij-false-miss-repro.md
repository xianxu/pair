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

- [x] Extract private native stimulus and strict raw signature counts; document invocation and liveness limits.
- [x] Run syntax/refusal checks and one real1000-query/four-worker experiment, retain result in Log, then close and prepare PR independently of #744.

## Log

TL c4915544 dispatches this tiny tooling side task while #744 integration suite waits. Use a separate checkout so the retained #744 merge runner sees no branch or working-tree changes. Quick plan follows the exact dispatched contract; proceeding under tl-worker quick-flow authority.

### 2026-10-21 — standalone native result and slot-only continuation

Measured from /tmp with Bash plus installed Zellij0.45.1 and stock utilities only, three fresh background shell sessions,1000 raw reads/four workers: exit0_target_listed=7, exit1_alive_confirmed=0, exit1_liveness_unconfirmed=0, other_nonzero=0. Independent raw-output classification agrees. Exit1 was not reproduced in this sample; zero is not evidence of absence. Cleanup passed (private sockets absent/root removed); a subsequent exact session-prefix process search found no server. LogSHA256a3a178d1c8660b2ce94a2e2e067d5de1b5700ee9795950f1172e0e432b49d9a9. The run manifest pins the measured script hash; source is unchanged from that run. Syntax and invalid-input refusal pass.

Initial draft setup refused before measured queries because the native socket directory is contract_version_1 rather than the release version; initial-setup-failure.log preserves it and its passing cleanup. No query failure was retried or erased. Corrected native run and ten-line README are retained. Temporary clone planning was stopped on TLbee5ced4's correction; its record was copied to scratch and the clone deleted, with no code there. After #744 landed PR509, this issue resumed through start-plan/change-code in canonical pair:5 on landed main. TL reported the draft counts received. Required suite and close/landing gates remain due; no product or local-gates configuration changed.

## Revisions

### 2026-10-21 11:52 — review corrections BR-1/2/3

Require complete stderr-byte equality for both native signatures, preserving final-newline distinctions; grep line membership is insufficient. Keep the same CLI, native query schedule, counters and private roots. Source-safe main/worker separation allows deterministic actual-worker tests with a stateful executable and real sockets, covering exact positives, extra diagnostic negatives and confirmed/unconfirmed exit1 liveness. Add top-level README and atlas pointers for this new standalone surface, and correct the original manifest's setup-failure filename. Preserve the original7/0 native sample and repeat native qualification on the corrected source. The obsolete queued suite lease199251636 was canceled through SDLC before execution; no suite result is claimed.
