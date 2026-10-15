---
id: 000577
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: high
card_mirror: '756825062ebfa0fd6cd3aa2daedbe62c172ccba7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T13:11:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3f3eb1d4", done: "524a9fa4"}
---

# Trimpath command-line assignment breaks the required suite environment probe

## Problem

After #572, the declared required suite is `make PAIR_GO_TRIMPATH=-trimpath test`. Its Make environment probe fails deterministically before the Go suite: `FAIL: knob PAIR_GO_TRIMPATH did not reach the test recipe`. This blocks #556 PR349 after its mandatory main integration. Bare make test-make-env passes.

## Spec

Preserve #572's trimpath default, additive Go flags, explicit caller overrides and nested Make behavior. Make the environment-isolation test valid when its outer make has a command-line assignment. Do not remove knob coverage or alter machine defaults to obtain green. The test currently creates synthetic environment values for every derived knob, then invokes nested make while inherited MAKEFLAGS/MAKEOVERRIDES still carries the outer command-line PAIR_GO_TRIMPATH assignment. GNU Make correctly gives that assignment precedence over the synthetic environment.

## Done when

- Both `make test-make-env` and `make PAIR_GO_TRIMPATH=-trimpath test-make-env` pass with genuine session-variable scrubbing and derived-knob checks retained.
- Caller overrides, additive GOFLAGS and nested Make tests remain meaningful; add an outer-command-line regression.
- The normal declared recorded suite passes on the corrected main head.

## Plan

- [x] Reproduce the recorded-command failure and add a regression running the real environment probe under bare and command-line-assigned outer Make invocations.
- [x] Clear inherited Make control variables at the synthetic fixture boundary; keep real caller-override and recursive Make assertions. Preserve the recorded command and trimpath defaults (ARCH-DRY: one probe implementation).
- [ ] Run both targeted invocations, the recorded suite, and close/land through the gates; notify pair:2.

## Log

2026-10-15: #556 integrated #572 cleanly apart from an artifact manifest list conflict resolved by keeping both entries. Recorded suite at c80734525e16 stopped before Go packages: exit2 in4m11s after504s queue; log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/f56b8f2ef9ecc3a5638161bf3707978c.log, sha256 prefix aa446f79e62e. All preceding editor/shell tests passed.

Isolated actual command repro: make PAIR_GO_TRIMPATH=-trimpath test-make-env exits2 with knob failure, while bare make test-make-env exits0 and reports198 derived knobs kept. Probe under assignment exposes MAKEFLAGS containing -- PAIR_GO_TRIMPATH=-trimpath, MAKEOVERRIDES=${-*-command-variables-*-}, and PAIR_GO_TRIMPATH=-trimpath; bare probe has no such assignment. No source changes made for this defect in #556. Filed unclaimed for independent work from main; #556 retains its branch and approved code.

2026-10-15: Reproduced the exact command: exit2, only PAIR_GO_TRIMPATH synthetic-knob check fails; additive/recursive Go flags assertions pass. The inherited command-line assignment is authoritative in nested Make. Fix the fixture boundary rather than removing knob coverage or changing the approved recorded command.

2026-10-15: Added a bounded wrapper running the same real assertions through a temporary Makefile under bare and explicit trimpath assignment. Before the fix, the bare case passed and the assigned case failed with the original knob diagnostic. Fixture-local removal of MAKEFLAGS/MAKEOVERRIDES/MFLAGS isolates all synthetic child Make invocations; explicit overrides created inside the Go-flags matrix remain intact. No build configuration or production behavior changes.
