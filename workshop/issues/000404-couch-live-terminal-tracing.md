---
id: 000404
status: working
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: '922ac7502752ac906468666cc024f29935e3e26b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-07T11:01:19-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3c01f2a8", done: "77814f77"}
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

- Explicit opt-in works in regular and isolated Couch; disabled mode writes nothing.
- Both terminal boundaries retain exact bytes, ordering, geometry and accepted-write receipts with tested completeness semantics.
- A regression test represents the observed startup burst; a real regular Couch smoke capture survives that workload.
- Queue, disk and write failures are visible during the session without corrupting terminal output; resource limits and long-running retention are documented and tested.
- Tests, review and runbook support shipping tracing to main independently, without closing #379 or claiming its display bug fixed.

## Plan

- [x] Import the existing implementation, tests and runbook from #379 onto a branch based on main.
- [ ] Revise the transferred implementation plan for the real overflow finding and long-running capture behavior.
- [ ] Fix and verify remaining capture reliability and failure visibility requirements.
- [ ] Review and ship instrumentation independently; keep #379 open for reproduction and diagnosis.

## Log

### 2026-10-07 — Split tracing delivery from diagnosis

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
