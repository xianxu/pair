---
id: 000606
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '8a31808a7994902f4da928f130eb5e4e5728781e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T12:04:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.38
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.

## Spec

- Attribute processes started by `sdlc local-check run` (and its `make test` / `go test` / test binaries) to the slot
  that started the run: the runner's in-flight record (under the worktree's `.git/sdlc/`) names the worktree, which
  maps to the slot; or the runner records its process-group ID at start.
- Such processes count toward the slot's resources (CPU, memory, writes), not as orphans.
- A true orphan (no live slot or run claims it) still alerts as today.

## Done when

- Fake-backed test: a detached suite's test binary is attributed to its slot and raises no orphan alert; an unclaimed
  process still does. Live: a full suite on one slot produces no orphan alert.

## Plan

Measured the cause live before designing: Claude Code's background shells (and a timed-out tool call's command) reparent
to launchd, so the suite loses the ancestry that tied it to the slot; remembered provenance then turns into a proven
orphan (`len(candidates) == 0`). The suite still carries the slot's `COUCH_THREAD_SCOPE`/`COUCH_THREAD_TAG`, readable
from KERN_PROCARGS2 for user binaries (sdlc, python, test binaries) but withheld by macOS for platform binaries
(`/bin/zsh`, `/bin/sleep`), so the detached root itself is often unreadable while its child is not. The runner's
in-flight record (the spec's first idea) doesn't exist and would only cover sdlc; the environment covers every detached
command of a slot.

- [x] `couchcore.DetachedSlotRoots` (pure): a launchd child whose inherited thread names a live slot becomes that slot's
  root; when it names none, its direct children are asked. Infrastructure (zellij, agents, nvim) is never claimed, so a
  relaunched thread's leftovers stay visible to the #583 orphan/unattributed rules (`ClassifyUnattributed` untouched;
  coordinated with pair:4).
- [x] `procutil`: `NativeResourceReader.Environment`, identity-fenced like `Command`; `ParseEnvironment` keeps only the
  named keys.
- [x] Sampler: applies the roots before selection; caches each generation's thread (one read per process lifetime,
  pruned with the table); new `environment_calls` cost.
- [x] Tests: pure attribution incl. withheld-shell fallback and infrastructure exclusion; reparented suite stays the
  slot's (orphan without the roots); sampler attributes a detached suite and reads each generation once (RED without
  the wiring); live child environment read.
- [x] Atlas: couch.md resource monitoring.

## Log



- 2026-10-16: closed — Post-SHIP fixes for two close-review minors: (1) SelectResourceProcesses hands a remembered dead root over to a live same-thread candidate (agent relaunched under its detached suite) instead of orphaning; TestDetachedSuiteOutlivesItsAgentsRelaunch RED without it. (2) inheritedThread does not cache IdentityUnavailableError (mid-read reparent) or cancellation; sampler test asks again next pass, RED without it. couchcore + couchcmd Resource tests green. (fixed-then-ship); review verdict: SHIP
- 2026-10-16: closed — Unit: couchcore DetachedSlotRoots (withheld-shell child fallback, infrastructure never claimed, input not mutated); reparented suite stays the slot (orphan=true without the roots, false with); sampler attributes a detached suite to pair:6 and reads each generation once across passes (RED without the wiring); procutil live child environment read via NativeResourceReader.Environment. couchcore, procutil, couchcmd Resource tests green. Live probe: 1,181 procs, first pass 275 ms / 1,359 env calls, cached 0.26 ms; real claims include my running local-check sdlc (pid 42438 -> 1-pair-8) under a withheld /bin/zsh. Live no-orphan-alert acceptance needs the running Couch on this build (TL after landing). Recorded suite running on ee39876a5.; review verdict: SHIP
- 2026-10-16: live probe on this machine: 1,181 processes, first pass 275 ms / 1,359 environment calls, cached pass
  0.26 ms. Claims found beyond suites, all genuinely slot-started: `pair title` daemons, my suite's lock-helper python,
  parley's cli-proxy-api from a couch thread, and a python pair:6 leaked on Oct 10 (now visible on its slot instead of
  invisible).

### 2026-10-15
