---
id: 000404
status: done
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: 'd7544b5d17dae350566780fbe48ca3a85d6ee0de' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T11:01:19-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.21
---

# Opt-in Couch live terminal tracing

## Problem

The intermittent Claude turn-end display corruption tracked by #379 cannot yet
be reproduced. Existing logs miss the Zellij-to-Couch and Couch-to-host streams.
Tracing must ship independently so the operator can use regular Couch while
waiting for an occurrence; shipping instrumentation does not resolve #379.

## Spec

Own the opt-in two-boundary capture implementation originally developed under
#379, its tests, runbook, and implementation plan. COUCH_CAPTURE_DIR explicitly
enables capture for one regular Couch process; isolation is optional. Preserve
ordered endpoint bytes/geometry, host write receipts, thread identity, private
files, bounded resources, default-off behavior, and terminal operation on failure.

The first real regular-session capture stopped after 6.88 seconds at 5.35 MiB
with `terminal capture queue limit reached`. Resolve realistic startup/burst
handling and expose failure while Couch is running, rather than only at exit.
Settle and document retention for a long-running wait (including the 256 MiB
limit and whether bounded rolling capture is needed), with replay completeness
explicit. The imported implementation is not yet ready to ship.

## Done when

- Explicit COUCH_CAPTURE_DIR opt-in works in regular and isolated Couch; disabled
  mode (including a limit setting alone) writes nothing and installs no observer.
- Both terminal boundaries retain exact bytes, ordering, geometry and accepted
  write receipts; unknown/truncated/incomplete captures cannot claim complete replay.
- A blocked-writer 4,000-record startup burst fits the bounded admission queue;
  regular and isolated real-PTY Console tests retain 3,800 KiB startup output exactly.
- Recording usage and sticky queue/full/IO failure are visible in actor and switcher
  views without input or child output, through the existing Presenter; capture
  failure does not replace the underlying terminal write result.
- Full-prefix recording uses a finite 256 MiB default disk cap, configurable via
  COUCH_CAPTURE_MAX_MIB from 1 MiB to 32 GiB. A capture directory admits at most
  32 GiB of reserved stream bytes and 64 sessions, without deleting evidence. Invalid enabled settings fail startup;
  both capture settings are cleared from children. Limits, memory overhead,
  stopped-state behavior, per-session retention and streaming extraction are documented.
- Focused tests, race checks and independent close review support shipping tracing
  to main independently; broad-suite baseline failures are stated explicitly.
  Closing #404 does not close #379 or claim its display bug fixed.

## Plan

- [x] Import the existing implementation, tests and runbook from #379 onto a branch based on main.
- [x] Revise the transferred implementation plan for the real overflow finding and long-running capture behavior.
- [x] Fix and verify remaining capture reliability and failure visibility requirements.
- [x] Prepare independent delivery with verified implementation and updated runbook; close/merge gates follow, and #379 remains open.

## Log

### 2026-10-07 — Split tracing delivery from diagnosis
- 2026-10-07: closed — BR1-3 addressed: README, evidence-preserving aggregate reservations, pure lifecycle. Full terminalcapture/couchtty, focused race in recorder/couchcmd/couchtty incl real-PTY startup pass; public Open quota mutation fails; build/diff check pass. Actual 3278-observation replay complete. Broad-suite assertion failures reproduced on main; remaining 126 couchcore tests pass after cumulative timeout; inventory still identical 53 findings. See issue Log. #379 stays open.; review verdict: SHIP
- 2026-10-07: flow upgraded quick → full — 954 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Operator requested a separate implementation ticket and branch; #379 retains the long debugging session and all evidence.
- Original implementation commits: 9ca17478 (capture) and 0b384b45 (regular-session activation). Imported work will preserve their provenance; historical test results do not establish live-capture reliability.
- Real evidence: `/Users/xianxu/.local/share/pair/captures/session-2417847440/events.jsonl`, 5,609,068 bytes, 3,280 records, 6.88 seconds, final incomplete marker `terminal capture queue limit reached`. The capture had already stopped, so it cannot record later incidents. Current queue bounds are 128 records / 8 MiB; disk cap is 256 MiB.
- The underlying display fault remains unproven. No root-cause fix is included in this ticket.

- Transfer completed onto `000404-couch-live-terminal-tracing` from current main. Investigation-only branch: `000379-stray-turn-end-text`; original history: `archive/000379-before-tracing-split`. Implementation plan: [#404 tracing plan](../plans/000404-couch-live-terminal-tracing-plan.md). A single import-block conflict in couchcmd/run.go retained both main's scrollbackcmd and tracing's terminalcapture imports. No tracing behavior changed during the split.

- Split verification: `git diff --cached --check` passed; recorder package tests passed; Capture/Observation-filtered tests passed across terminal, ptychild, couchcmd, couchcore and couchtty (ptychild had no matching tests); explicit `TestPtyRunnerObservesFirstBytesAndGeometryBeforeDelivery` passed. Compared original and transferred production/test/atlas/lessons diff additions and removals: identical, with newer main context retained. #379 branch differs from main only in its issue document. These checks verify transfer integrity, not a fix for live overflow.


## Revisions

- 2026-10-07 — Operator authorized finishing #404. Completion design retains full
  terminal prefix rather than lossy rolling history; make the finite disk budget
  configurable and show persistent usage/failure in Couch. Increase the record
  count bound to fit measured startup bursts while retaining the 8 MiB admitted
  record-cost bound (channel/encoding overhead is additional). The plan records
  exact phases, notification ownership and regression checks. No change to #379's
  diagnosis scope or acceptance. Existing opt-in activation is unchanged.

### 2026-10-07 — Reliability design evidence

- Measured source capture: 3,280 records, 3,722,498 data bytes; largest payload
  18,851 bytes, peak 279 records / 277,881 bytes in 100 ms. A 128-record limit is
  below this observed burst even though its data fits the existing byte budget.
- Fresh-eyes plan review requested precise memory accounting and near-limit
  record coverage; plan updated accordingly. Run loop will consume coalesced
  recorder-status notifications and repaint through Presenter; no separate
  terminal writer or polling worker.

### 2026-10-07 — Reliability implementation and verification

- Recorder admits 8,192 records within the same 8 MiB record-cost bound; explicit
  phases and coalesced status notifications preserve terminal-owner rendering.
  Status shows `REC N%` or persistent queue/full/IO stop. A disk-limit override
  supports finite full-prefix captures from 1 MiB through 1 TiB; default remains
  256 MiB, runbook suggests 4 GiB for a longer wait. Both settings are cleared
  from child environments; a limit alone never activates capture.
- Recorder package/race tests pass, including blocked 4,000-record startup,
  near-limit in-flight allocation, lifecycle/status wakeups, ownership and limits.
  Full couchtty suite passes, including idle actor/switcher status and pre-Run
  failure; deleting the event-loop notification repaint makes the regression fail.
- Actual failed capture replayed locally without delays: 3,278 observations,
  3,722,498 bytes, exact comparison, complete end in 100.7 ms. No private payload
  committed. Regular and isolated real-PTY Console startup tests each preserve
  3,800 KiB exact input plus final host marker, both pass in 5.51 seconds total.
- Targeted race run passed across terminalcapture, terminal, ptychild (no matching
  tests), couchcmd and couchtty; log `/tmp/pair404-race.log`. Build passed via
  `go build -o /tmp/pair404-couch ./cmd/couch`; diff check passed.
- Full repository run `/tmp/pair404-go-all.log` found baseline failures. Fresh
  origin/main b933b5a5 baseline `/tmp/pair404-baseline.UU2Pv4`, with generated
  runtime bundle, has the identical 53 artifact inventory findings and all five
  continuation scope-conflict failures. Cold-resume switcher failed 1/5 baseline
  repetitions: it presses Return on the menu header before async inventory loads
  (inventory unavailable -> no selection -> parked row); its fixture has capture
  nil. These are not marked fixed or silently waived by tracing acceptance.

- Final broad verification: `go test ./...` returned nonzero with five failing
  packages, all named assertion failures reproduced on baseline b933b5a5:
  artifactpath, couchcmd, couchcore, gcruntime and launcher. Additional baseline
  logs `/tmp/pair404-baseline-{couchcore,gcruntime,launcher}.log` match missing
  thread claim, missing slot metadata and five launcher scope/restart failures.
  Couchcore also exhausted its cumulative 10-minute package budget while starting
  a test; the remaining 126 tests were run separately and passed in 25.258 s
  (`/tmp/pair404-core-tail.log`). Other packages, including full terminalcapture,
  terminal, ptychild and couchtty suites, passed. This is not a green full-suite
  claim; known baseline failures are explicitly preserved for review.
- Runbook extraction was executed on a disposable complete fixture: exact bytes,
  partial host acceptance and path-safe endpoint naming passed; missing end marker
  was rejected. Two streaming passes support large captures without full-file RAM.
- Implementation and verification are ready for the single SDLC close review,
  then independent publication. #379 remains working, untouched by this close.

- Close preflight requested refreshed acceptance wording after the reliability
  revision. Done when now states the implemented burst, persistent status,
  configuration and complete-prefix retention contracts explicitly; no bypass.

- Boundary review round 1 requested README coverage, aggregate retention and pure
  lifecycle authority (BR-1–BR-3). Plan revised: reserve session allowances against
  a 32 GiB / 64-session directory budget, preserve all evidence on refusal, narrow
  per-session limit accordingly, and centralize phase changes in a pure model.
  Operator explicitly confirmed full-prefix capture with a visible limit.

- BR-1 addressed: README now documents activation, limits, status and aggregate
  admission with a runbook link. BR-2 addressed: storage admission reserves full
  allowances under a persistent nonblocking flock, bounds bytes/session count,
  refuses unsafe/unknown metadata and never deletes prior evidence. Both reservation
  and legacy-ending readers use shared strictjson decoding, including duplicate-key
  rejection. Open's real quota wiring is covered: replacing it with unbudgeted
  creation makes TestCaptureStorageOpenKeepsCompletedReservation fail.
- BR-3 addressed: lifecycle.go is the authoritative pure transition model; Recorder
  executes returned close/notify effects. Sequence tests cover repeated close,
  queue/IO failure, timeout and late completion, including successful completion
  winning a race against timeout. New lifecycle/storage sources registered.
- Post-review verification: full terminalcapture and couchtty suites passed
  (`/tmp/pair404-review-packages.log`); focused race passed in terminalcapture,
  couchcmd and couchtty, including both real-PTY startup configurations
  (`/tmp/pair404-review-race.log`). Build and diff check passed. Rechecked artifact
  inventory: exactly the same 53 baseline findings, zero additions/removals.
