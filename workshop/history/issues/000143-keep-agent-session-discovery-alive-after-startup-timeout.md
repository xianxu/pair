---
id: 000143
status: done
deps: []
github_issue:
created: 2026-08-18
updated: 2026-08-19
estimate_hours: 1.00
started: 2026-08-18T22:02:47-07:00
actual_hours: 0.31
---

# Keep agent session discovery alive after startup timeout

## Problem

The asynchronous session watcher gives up after 60 seconds. Agents such as
Codex can create their transcript only after their first interaction, so an
agent left idle for longer than the startup window never gets a persisted
session ID. The context meter then has no transcript to read and the frame omits
context-window usage for the rest of the session.

## Spec

Treat the existing timeout as the end of fast startup discovery, not the end of
the watcher. When a fresh agent PID is available, continue discovery at a
low-frequency 60-second interval while that process is alive. Stop immediately
when a poll observes that the process has exited (within one slow-poll interval),
and preserve the existing bounded timeout when no fresh PID can be established.

Apply the lifecycle behavior uniformly to every asynchronous agent supported by
`sessionwatch` (Codex, Agy, and Muse). Claude supplies its session ID
synchronously and must remain unaffected.

For native launches, define PID freshness against a generation lower bound
captured immediately before either producer spawns the watcher, not against the
later time at which the detached watcher process happens to run. Both whole-
workbench launch/restart and Shift+Alt+N agent-only restart must build the
internal `pair session-watch` command through one shared serializer. The watcher
must compare generation-bound mtimes at full filesystem precision: accept a PID
file written after the launcher bound even when it predates watcher startup,
while rejecting an older PID even within the same wall-clock second. Direct or
legacy watcher invocations without the bound retain the existing watcher-start
rule and same-second compatibility tolerance.

## Done when

- A live supported agent can acquire and persist a session ID after the initial
  discovery timeout.
- Post-timeout discovery polls no more often than once every 60 seconds by
  default.
- The watcher exits within one slow-poll interval after the bound agent process
  exits, and still times out when no fresh PID exists.
- Automated tests cover delayed discovery for Codex, Agy, and Muse plus both
  exit paths.
- A detached watcher that starts after the new agent PID file was written still
  binds that PID and persists the new session; an older live PID remains stale.
- Whole-workbench Alt+N and agent-only Shift+Alt+N use the same generation-bound
  watcher command contract.
- Shift+Alt+N derives asynchronous-agent support from `SpecForAgent`: Codex,
  Agy, and Muse spawn the watcher; synchronous Claude does not.

## Plan

- [x] Test `Run` with the existing fake-clock/stateful-runtime seam, scheduling
  transcript and process-state transitions to guard the fast-to-slow cadence,
  every `AgentSpec`, PID-bound exit, and PID-less timeout.
- [x] Change the watcher loop to transition from fast polling to lifecycle-bound
  slow polling, using the existing injected runtime seam (ARCH-PURE,
  ARCH-MOCK).
- [x] Run focused and repository-wide verification; confirm the synchronous
  Claude launch path is unchanged (ARCH-PURPOSE, ARCH-DRY).
- [x] Pass one launcher-generation lower bound through the existing watcher
  spawn argv and parse it into watcher options (ARCH-DRY, ARCH-PURE).
- [x] Reproduce the delayed-sidecar race with the stateful launcher/watcher
  fakes, then verify focused, integration, and repository-wide suites
  (ARCH-MOCK, ARCH-PURPOSE).

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method. The
work extends established Go, CLI, and shell-test seams, so no library-availability
adjustment applies.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.04
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.05 impl=0.04
item: milestone-review design=0.04 impl=0.12
design-buffer: 0.15
total: 1.00
```

## Log

### 2026-08-18
- 2026-08-18: closed — Judgment actual: 0.10h from the 22:02 claim through the 22:08 verified commit because transcript telemetry is unavailable. Focused sessionwatch tests pass; all Go packages pass with runtime assets generated and inherited Pair session variables cleared; tests/pair-session-watch-test.sh passes; git diff --check clean. No atlas update: watcher scheduling only, with no new user-facing or architectural surface.; review verdict: FIX-THEN-SHIP

- Root cause: the live Codex transcript appeared after the watcher's fixed
  60-second deadline, leaving `config-<tag>-codex.json` absent even though the
  transcript parser supported the current event format.
- Design approved by the operator: retain fast startup polling, then poll every
  60 seconds for the lifetime of the bound agent process. Apply this to all
  asynchronous agent specs rather than special-casing Codex.
- Implemented one shared `Run` schedule for Codex, Agy, and Muse: 100 ms during
  the startup deadline, then 60 seconds while the fresh bound PID remains alive.
  PID-less legacy discovery retains its bounded timeout; Claude remains on its
  existing synchronous session-ID path.
- TDD evidence: the delayed-discovery tests first failed because `SlowPoll` did
  not exist, then passed after the lifecycle loop was implemented. Focused
  package tests, all Go packages (with generated runtime assets and Pair session
  variables cleared), the shell session-watch integration test, and
  `git diff --check` pass.
- Boundary review verdict: FIX-THEN-SHIP. Updated both atlas descriptions that
  still treated the startup deadline as the watcher's terminal lifetime; no
  production-code findings remained.

### 2026-08-19
- 2026-08-19: closed — Fixed launch-generation freshness, root-vs-subagent authorization, process-incarnation checks before discovery and persistence, and positive-decimal PID validation preventing Linux proc aliases from binding the watcher to itself. Verified adversarial procutil tests, Linux amd64 cross-compile, focused sessionwatch/launcher packages, real shell watcher race regression, complete make test, and branch-wide git diff --check.

- Reopened after live Alt+N diagnosis: the launcher spawned the watcher before
  Zellij, but OS scheduling did not run the detached watcher until 08:47:32.
  The new wrapper had already written `agent-pid-parley_nvim` at 08:47:31, so
  `freshPID(..., watchStart)` rejected the correct live PID, legacy discovery
  snapshotted the already-created rollout, and the watcher logged its startup
  timeout while the previous config ID survived.
- Approved design: carry the launcher generation lower bound through the
  existing internal sidecar argv. Keep timestamp capture inside the OS spawn
  seam and the comparison in the injected watcher core (ARCH-PURE); do not add
  a parallel nonce/readiness protocol or an unsafe grace window (ARCH-DRY).
- TDD RED: focused packages failed on absent `CommandArgs`,
  `Options.PIDNotBefore`, `pidFileCurrent`, and fixed-time restart construction.
  GREEN: launcher/sessionwatch/wrapcmd packages pass with inherited Pair launch
  variables cleared.
- Both whole-workbench launch/restart and Shift+Alt+N now serialize the watcher
  through `sessionwatch.CommandArgs`. Native freshness uses the producer's
  RFC3339Nano generation bound; legacy direct calls preserve same-second
  watcher-start tolerance. Shift+Alt+N derives watchability from
  `SpecForAgent`, covering Codex, Agy, and Muse while excluding Claude.
- Real-process regression writes the new PID after the bound but before watcher
  execution, and separately holds an old live PID stale until replacement; both
  pass through the production CLI/filesystem seam.
- Verification: focused packages, `tests/pair-session-watch-test.sh`,
  `go test ./... -count=1`, full `make test`, and `git diff --check` pass. The
  first full-suite attempt exposed only the temporary worktree's broken
  `../ariadne` Makefile link; after providing that external fixture path, the
  suite passed unchanged.
- Shadow sweep updated the watcher lifecycle, generation contract, single-binary
  route, and Codex/Agy/Muse coverage across the current atlas and code comments
  (ARCH-PURPOSE, ARCH-DRY).
- First boundary review returned REWORK on artifact/test rigor, not watcher
  behavior: it found `pidFileCurrent` mislabeled PURE, an inherited-environment
  readiness test, and stale Codex/Agy dispatcher help. Extracted pure
  `pidFileFresh`, made the test hermetic, generalized the help text, and revised
  the plan's entity table before re-review (ARCH-PURE, ARCH-PURPOSE).
- Second boundary review found that indefinite polling revalidated only numeric
  PID liveness, allowing a recycled PID to impersonate the bound agent. Added a
  stable kernel process-start identity to the runtime seam, a stateful PID-reuse
  regression, and direct coverage of the launcher producer helper
  (ARCH-MOCK, ARCH-PURPOSE).
- Third boundary review found a narrower PID-reuse window during the IO-heavy
  discovery call. Reauthorization now occurs immediately before persistence,
  with a stateful `LsofPaths` hook proving a mid-discovery incarnation change
  discards the candidate. The remaining retired watcher/recovery names were
  reconciled across atlas and transcript comments.
- Fourth boundary review accepted runtime behavior and found only plan status
  drift plus two shell-era recovery instructions. Corrected the entity statuses
  and pointed the guide at the native launcher seams.
- Fifth boundary review found that Linux's `/proc/self` alias let a malformed
  nonnumeric pidfile bind the watcher to itself indefinitely. Centralized
  positive-decimal PID validation across native identity implementations and
  added adversarial cases for aliases, zero, negatives, and nonnumeric input
  (ARCH-PURPOSE, ARCH-DRY).

## Revisions

### 2026-08-18 — Plan-quality review

- Clarified that process death is observed at the next 60-second slow poll,
  rather than promising impossible immediate detection during a blocking sleep.
- Recast the test plan as a function-level fake-clock strategy for `Run`.

### 2026-08-19 09:00 PDT — Delayed watcher-start race

- Extended PID freshness from “newer than watcher process startup” to “newer
  than the native launch generation.” This preserves stale-PID rejection while
  covering scheduler delay between sidecar spawn and sidecar execution.
- Added launcher argv/CLI parsing and cross-boundary stateful-fake coverage to
  the implementation scope; root/subagent authorization remains owned by #144.

### 2026-08-19 09:05 PDT — Fresh-context plan review

- Added the Shift+Alt+N watcher producer to the shadow sweep and moved command
  serialization to `sessionwatch` so launcher and wrapper cannot drift
  (ARCH-DRY, ARCH-PURPOSE).
- Required nanosecond comparison for native generation bounds while preserving
  the legacy same-second rule, plus a same-second stale negative regression.
- Added an OS-backed CLI/filesystem regression where the PID is written after
  the bound but before watcher execution, matching the observed scheduler race
  rather than testing only isolated seams (ARCH-MOCK).

### 2026-08-19 09:10 PDT — Async-agent shadow sweep

- Replaced Shift+Alt+N's hardcoded Codex/Agy watcher condition with
  `sessionwatch.SpecForAgent` as the existing source of asynchronous-agent
  support. Required Codex/Agy/Muse positive tests and a Claude negative test so
  Muse cannot fall out of the restart path again (ARCH-DRY, ARCH-PURPOSE).

### 2026-08-19 09:40 PDT — Process incarnation

- Clarified “while that process is alive” as the same process incarnation, not
  any future process assigned the same numeric PID. The watcher must stop when
  the kernel start token changes, including PID reuse within one slow-poll
  interval.

### 2026-08-19 09:50 PDT — Authorization point

- Required incarnation validation both before discovery and immediately before
  session persistence, so PID reuse during descendant/lsof/filesystem IO cannot
  transfer the original watcher's authority.
