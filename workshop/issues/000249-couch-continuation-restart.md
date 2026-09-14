---
id: 000249
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-14
estimate_hours: 6.216
started: 2026-09-14T10:30:49-07:00
---

# Fix continuation restart for Couch-hosted Pair threads

## Problem

Couch-hosted Pair can disappear when an agent saves a continuation: the writer
successfully commits the checkpoint and kills the current Zellij session, but
the replacement fails to launch. Couch retains a stale live incarnation.

Observed 2026-09-13 on Pair thread
`e108517d46ab4575/couch-36b623f6869ebaa2` (`📁pair-couch-27`):

- Native Codex binding `01a09ce1-0dae-7251-ae65-6964a7a2a92b` was established
  around 15:26; the session had substantial conversation. Missing transcript
  establishment does not explain this incident.
- 21:01:31 America/Los_Angeles: implementation checkpoint commit `4d18da7f`.
- 21:02:34: `pair continuation --slug pair-storage-retention ...` ran in the
  #239 worktree and committed `b96e1dc3`. The checkpoint is
  `workshop/continuation/20260913T210234-pair-storage-retention.md` there.
- 21:02:36–37: scrollback was preserved and the session ended. Later observation
  found no replacement Zellij session and Couch still recording helper PID 5330
  as live, displayed as `stale — couch exited unexpectedly`. The Couch
  supervisor and the other attached threads remained running.

### Reproduced registration mismatch

`continuationcmd` invokes `pair continue <slug>` after writing the checkpoint.
`launcher.runCompaction` preserves scrollback, writes a restart marker, then
kills the session. The outer `RunLaunch` loop consumes the marker and calls
`planRestart`, replacing `opts.Args` with arguments that have neither
ResumeRequired nor FreshRequired. Couch scope/tag remain in the environment.

Consequently `runCreate` calls `EnsureThreadAddress(..., couchOwned=true)` rather
than `RegisterExistingCouchThread`. That path accepts only a reserved marker;
the thread's existing marker is already established, so it returns
`Pair thread address already claimed` before launching the replacement.

A temporary Go overlay diagnostic exercised the production claim functions and
restart planner using a temporary store: reserve → establish → plan continuation
restart → attempt Couch claim. It confirmed the rejection and that read-only
existing-thread registration accepts the same marker. Command result:
`TestDiagnosticHostedRestartClaim` passed, launcher package 0.362s. This is a
component reproduction, not a full hosted restart reproduction; the original
outer-launcher error output was not recovered. No production code was changed.

The existing `TestRunLaunchContinueReentry` exercises a standalone fake whose
claim methods return canned errors; it does not enforce the Couch marker
lifecycle and therefore misses this interaction.

### Checkpoint lookup crosses worktrees incorrectly

The restart marker carries only a continuation slug. The outer launcher resolves
it under its own git root via `continuationDirPath`. Here it was started from
the main Pair checkout, but the writer saved the checkpoint in the #239 worktree.
The main checkout lacks that file. The current lookup silently leaves
ContinueDoc empty when resolution fails, so repairing registration alone can
still launch a fresh conversation without its handoff.

## Spec

Make continuation restart an explicit supported operation for Couch-hosted
threads. Preserve the same Pair address and correctly register the existing
hosted thread when launching the intended fresh agent conversation. Do not
weaken initial reservation checks or turn missing ownership evidence into
permission to create a session (ARCH-DRY, ARCH-SECURE).

Carry a durable, validated checkpoint reference across the writer/outer-launcher
boundary so a checkpoint authored in another worktree is resolved exactly.
Specify file lifetime and availability, and validate the handoff before stopping
the source where possible. Missing, unreadable, or mismatched checkpoints must
produce an actionable failure, never an unseeded fresh conversation presented
as successful continuation.

Define Couch's lifecycle across the restart, including helper survival or
replacement, registration, child exit, failure, and retry (ARCH-ORDER). After a
failure Couch must retain the checkpoint reference and truthful recoverable
state; it must not strand the thread behind a stale live incarnation or claim
the supervisor crashed when only one actor's restart failed.

The checkpoint remains the recovery source; native transcript binding of the
old conversation is not a substitute for the requested fresh continuation.
Audit adjacent restart-marker consumers for the same ownership/lookup class,
while keeping the work scoped to the continuation/restart contract.

## Done when

- An actual Couch-hosted continuation write restarts into the same managed
  thread with a fresh agent conversation seeded from the exact checkpoint.
- The flow works after warm reattachment and when the checkpoint is written
  in a different worktree from the outer launcher's working directory.
- Stateful integration coverage crosses writer → restart marker → teardown →
  real registration policy → replacement launch → Couch attachment, using
  portable fake processes/stores; no tests mutate real user sessions.
- Tests cover missing/unreadable checkpoints, established versus reserved
  claims, replacement failure, interruption and retry, without duplicate owners
  or silent unseeded launches. Existing standalone restart behavior stays covered.
- Failure leaves an actionable Couch state and durable checkpoint; docs explain
  supported continuation behavior and recovery.

## Plan

- [x] Reproduce hosted restart with stateful ownership and cross-worktree checkpoint fixtures.
- [x] Design restart ownership, checkpoint transport, and failure recovery in a durable plan.
- [x] Implement, verify the full hosted flow and standalone regressions, and update docs.
- [x] Submit the verified implementation to the SDLC closing gate.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Derived after plan-quality accepted round 3;
the calibration source is marked stale, so this remains provisional.

The primitives, in block order, cover issue/design authoring, checkpoint/request
model, persisted slot, writer/standalone transport, owner execution, interrupted
attempt reconciliation, Console state/polling, CLI bootstrap, cross-package
acceptance, live conformance, docs, and close review. Existing park, tracked
fresh start, orientation and queue implementations satisfy the library check;
there is no new service or third-party library to build. Implementation picks
use the upper table allowance where lifecycle coverage spans packages, scaled
to 40% by v3.1. Design uses the plan's 0.2 spec discount for implementation
primitives (not the design authoring itself); familiarity is 1.0 and the
thorough-plan design buffer is 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.12
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.20 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 6.216
```

## Log

### 2026-09-13

Filed at operator request following investigation of the apparent Pair crash.
Native transcript and parked TTY capture place the exit immediately after the
continuation writer call. The capture is
`~/.local/share/pair/repos/e108517d46ab4575/parked-scrollback-couch-36b623f6869ebaa2-20260913T210236.raw`
with sibling events JSONL. Source inspected in launcher compaction, restart
planning, createflow, thread claims, continuation lookup, and Couch launch
registration. This differs from #248's detached-session inventory rejection:
this incident deliberately terminated the source but failed to replace it.
No implementation started; runtime identities above are historical evidence.

### 2026-09-14 — Implementation checkpoint; integration remains open

The durable design is `workshop/plans/000249-couch-continuation-restart-plan.md`.
Fresh-context plan review passed after one revision addressing last-source-exit
ordering. `sdlc change-code` plan-quality passed on the third review round; the
next invocation supplied the estimate and opened this implementation branch.

Implemented, still uncommitted: immutable bounded checkpoint/request model and
persisted slot; exact committed path and digest through the writer/launcher;
Couch publication without inner teardown; durable standalone marker/retry;
Couch-owned park/fresh replacement, receipt reconciliation, timeout and safe
source/target warm recovery; continuation guards and declared operations.
The Console worker, status/retry UI and CLI bootstrap are partly integrated.

Implementing agents reported passing full package tests for checkpoint,
threadrecord, writer, launcher and Couch core, plus focused race tests. This is
component evidence, not end-to-end completion. Root is running the combined
affected-package check with output in `/tmp/pair-249-integration-status.log`.

Known remaining wiring: inject the OS source reader; bind the writer's expected
digest at CLI ingress; execute again after adopting a recovered source; process
healthy request statuses when another slot fails to read. Then finish actual
hosted acceptance, live-conformance coverage, artifact classifications, docs,
full suite/build and the mandatory SDLC closing review. No smoke-ready or
code-complete claim yet; #250 has not started implementation.

Progress reporting lagged while root answered side questions and investigated
the separate display defect (#252). The implementation agents continued, but
root repeatedly ended turns instead of resuming integration and did not update
this log. This checkpoint corrects the record; future progress is recorded here
at each completed integration/verification unit.

### 2026-09-14 — CLI and Console integration regressions

Root wired the OS source reader and exact digest at CLI ingress. New regression
tests first failed for missing digest transport, a corrupt slot blocking healthy
requests, recovered-source execution stalling in receipt polling, and an empty
error result discarding request identity. All four cases now pass. Additional
Console checks verify actual replacement attachment, focus preservation for a
background thread, and no automatic retry of failed requests.

A non-Console continuation result also exposed an owner-lifetime bug: the CLI
printed the result and returned instead of waiting for its new helper. Generalized
rendering to the existing StartedChild interface, with a failing-then-passing
exit-code test. Full UI/CLI verification is running in
`/tmp/pair-249-ui-cli-tests.log`. Acceptance and live-conformance work remain in
progress; this is not the closing verification run.

### 2026-09-14 — Transport and complete handoff acceptance

Task 2 now carries the writer's committed absolute checkpoint path and SHA-256
through the launcher and the declared Couch publication operation. Hosted
continuation leaves source teardown to Couch; hosted restart/rename refuse
before side effects. Standalone continuation retains its immutable snapshot and
fresh argv through failed replacement and explicit retry; marker writes and
quit writes are checked, and acknowledgment is generation-checked after success.
Shared leaf validation bounds the body and requires continuation frontmatter
and a substantive NEXT ACTION. This preserves one validation authority
(ARCH-DRY) and orders durable intent before source teardown (ARCH-ORDER).

`go test ./cmd/internal/launcher ./cmd/internal/continuationcmd ./cmd/internal/checkpoint -count=1`
passed (6.638s / 0.944s / 0.636s), and focused race tests for those packages
passed; commands and output are retained in
`/tmp/pair249-transport-tests.log` and `/tmp/pair249-transport-race.log`.
Observed red tests covered missing checkpoint APIs, writer slug/error handling,
malformed markers, missing kill executable, unbounded body input, wrong installed
Couch executable lookup, NEXT ACTION only inside frontmatter, and missing durable
fresh argv. Each was corrected before the green runs.

`TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` now builds the
real Pair binary, writes and commits in a sibling Git worktree while its outer
working directory is the main fixture checkout, and follows the actual serialized
writer output through the real launcher, CLI dispatch, ThreadStore, and source
ledger reader. Both initial and warm-reattached cases continue through durable
Pair lifecycle request/completion files, verified source death, production
existing-address registration, exact materialized seed and fresh profile,
on-disk readiness waiting/submitted receipts, complete request state, and
Console input/output. Established claim bytes and prompt-history bytes remain
unchanged. The warm case additionally enters the actual retry CLI operation,
checks singleton lease ownership during replacement attachment, and observes
release after its finisher. Missing committed digest, post-launcher file tamper,
and obsolete source generation produce their exact expected errors and leave
the source, claim, and committed checkpoint intact without request authority.

`go test -race ./cmd/internal/couchcmd -run '^TestContinuationWriterPublishesExactCheckpointAcrossWorktrees$' -count=1 -v`
passed all five cases; output is in
`/tmp/pair249-publication-acceptance-race.log`. The non-race command also passed,
with output in `/tmp/pair249-publication-acceptance.log`. One acceptance fixture
red run proved the retry declaration requires `ref`, not `tag`; the corrected
fixture exercises the declared operation rather than a direct callback.
`rg 'WriteRestartMarker|TakeRestartMarker|RestartMarker|planRestart' cmd bin`
was reviewed; its inventory is in `/tmp/pair249-restart-consumer-sweep.log`.
There is no destructive TakeRestartMarker consumer remaining. `git diff --check`
is clean.

The portable acceptance uses real temporary Git/store/claim/protocol files and
simulates external process death, blocked helper execution, and terminal devices.
It does not claim paid-agent composer recognition or real Zellij teardown;
the separately maintained live conformance and operator smoke cover those
boundaries. No production sessions or data were mutated. Code commit, full-suite
integration, operator smoke, and SDLC close remain with the main agent.

### 2026-09-14 — Verification run and remaining closing gate

Full Couch CLI/Console package tests passed. Final focused continuation race
tests passed across checkpoint, writer, launcher, core, Console and CLI;
ThreadRecord's full ordinary tests passed earlier (the final focused race
selector has no ThreadRecord tests). Full `make test-couch-zellij-live` passed,
including detach, verified park and continuation seed/receipt transport.

The first `make test` run stopped at the standalone restart shell fixture:
inherited COUCH_THREAD_SCOPE/TAG made its standalone restart correctly refuse.
The fixture now explicitly clears those two hosted-context variables; its
focused rerun passed. The full rerun is in `/tmp/pair-249-make-test-2.log`.

Additional boundary tests caught and fixed a missing required thread reference
in the Retry menu's declared-operation payload and pending requests incorrectly
being sent to receipt polling after an admission conflict. Queue-overload tests
verify source focus/liveness remains intact until the request is admitted.

Implementation and acceptance are ready for closing verification. Remaining:
finish full suite/build, commit the coherent implementation and checked plan,
run `sdlc close`'s mandatory fresh-context review, address any findings, then
pause for operator smoke. No closing review has run yet.

### 2026-09-14 — Closing verification passed

`env -u PAIR_SESSION_ID -u PAIR_TAG make test` passed on the second full run,
including all Go packages, shell and Lua checks (`/tmp/pair-249-make-test-2.log`).
`make build` passed (`/tmp/pair-249-build.log`). Full live Zellij conformance
passed (`/tmp/pair-249-live-conformance.log`), focused race tests passed
(`/tmp/pair-249-final-race.log`), and `git diff --check` passed. The implementation
is being committed for the mandatory fresh-context closing review; its verdict
is still pending. Operator smoke follows review, before #250 implementation.


### 2026-09-14 — BR-1 core-concepts traceability corrected

The closing gate returned REWORK with BR-1 (plan/diff traceability) and BR-2
(asynchronous focus ordering). BR-1 is addressed: audited all six pure-entity
and seven integration rows against pinned `7800e968..f5fa755b`, then appended a
superseding Core-concepts audit under the durable plan's Revisions. Orientation
files are reused unchanged; the correction names their actual continuation
callers, the new `checkpoint_io.go` implementation location, the mixed new and
modified Console files, and the exported `checkpoint.Request` name. No runtime
code changed for BR-1. Row-by-row diff and pinned-symbol inspection verified the
correction; BR-2 and the mandatory closing-gate rerun remain pending.

### 2026-09-14 — BR-2 reproduction and requested disposition

The first closing review returned REWORK. BR-1 is corrected by the complete
concept-table audit recorded in the plan revision. BR-2's stated focus override
does not reproduce against the unchanged production implementation.

`TestContinuationCompletionPreservesInterveningFocus` exercises the actual
acceptance, switchTo/onHotkey, source exit, finishOperation, declared dispatcher,
and attach installer. Its four cases select another actor, return to the panel,
switch inside attach dispatch, and switch before source exit. All pass with the
race detector (`/tmp/pair-249-br2-reproduction.log`). No production focus changes
were made to obtain that result.

`installObservedThreadActor` checks `c.active == ""` under the same mutex used
to install the new pane and change focus. Selecting another actor sets active;
reopening the panel retains that active actor. `finishOperation` only forces a
switch for resume, not continue-thread or retry-continuation. Thus a foreground
attach does not override an intervening live actor selection. Request that the
next review withdraw BR-2, or provide a counterexample beyond these covered
event orders. The ledger disposition remains the reviewer's responsibility.

## Revisions

### 2026-09-14T10:40:00-07:00 — Shared recovery contract and execution order

Operator approved design and implementation of #248 → #249 → #250, pausing for
smoke after #248. Coordinated contract is recorded in
`workshop/plans/000248-couch-warm-reattach-gate-plan.md`; #249 receives its own
executable plan after that checkpoint. This issue owns durable continuation
request, checkpoint availability, replacement and retry, reused by #250.

Additional source findings: createflow seeds only the checkpoint basename under
the launcher's relative `workshop/continuation/`, so exact lookup must extend
through final prompt delivery. FreshRequired currently clears continuation
fields; setting that flag alone cannot fix registration. Existing tracked
fresh-existing launch and cleanup should be reused, with ownership independent
of conversation mode. Restart marker consumption currently precedes success;
the replacement must retain recoverable intent across failure.

### 2026-09-14T11:25:00-07:00 — Start implementation design after #248 shipment

Operator accepted #248 for shipment and requested the next issue. #248 merged
through PR #134; its coordinated plan is now archived under
`workshop/history/plans/000248-couch-warm-reattach-gate-plan.md`.

Source tracing confirmed that warm attachment retains the outer launcher loop,
but an in-place restart would leave Couch's helper ownership shape unchanged
and bypass its registration transaction. Design #249 around Couch-owned
replacement using existing verified park and fresh-existing start machinery
(ARCH-DRY, ARCH-ORDER). The writer transports an exact checkpoint snapshot;
the Couch owner picks up a durable request even while the panel is closed.
Failed requests retain that snapshot and require explicit retry. Account for
last-actor exit before replacement attachment so Couch itself stays open.

### 2026-09-14 — Make the closing checklist describe its review handoff

The first close invocation stopped before review because its own final "Close"
checkbox was unchecked. Reworded that self-referential checklist item to the
completed submission of verified work. The review verdict remains pending in
Log and issue status stays working; no review or verification gate is bypassed.
