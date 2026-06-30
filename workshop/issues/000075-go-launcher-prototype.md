---
id: 000075
status: done
deps: [000073, 000074]
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 3.75
started: 2026-06-29T21:55:46-07:00
actual_hours: 0.98
---

# pair Go launcher prototype

## Problem

The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.

## Spec

Prototype the launcher core in Go behind an alternate or guarded path. The prototype should implement a coherent vertical slice of `bin/pair` behavior while the shell launcher remains the public default.

The #73 inventory identifies `bin/pair` as the P0 public launcher surface. For this milestone, the guarded path is `pair-go launch`, a development-only launcher prototype that reaches the decision boundary but never starts or attaches a real zellij session. The public `bin/pair` shell launcher remains the only supported user entrypoint.

The vertical slice is:

- argv parsing for `pair-go launch`, including default agent, `resume <tag>`, optional agent positional, and `--` forwarded args;
- tag normalization and default tag derivation using the same bare-or-`pair-` contract as `bin/pair`;
- data-dir resolution from `XDG_DATA_HOME` / `HOME`;
- a session snapshot model that represents live, detached, exited, and historical tag candidates;
- a pure decision model for forced resume attach/create, direct create, picker-required, and historical create;
- a thin command/filesystem shell that can read fake `zellij` output and filesystem sidecars in tests, then print the selected prototype action.

The prototype must fail explicitly after the decision phase, rather than silently launching or diverging from `bin/pair`. Unsupported launcher behavior stays shell-owned and documented as out of scope: `continue`, `rename`, zellij lifecycle execution, quit/restart markers, orphan cleanup, cmux title ownership, dev rebuild, tag-restart prompt, config migration, and real fzf interaction.

The Go model stores canonical tags as bare names such as `demo`. Zellij session names are derived at the boundary as `pair-<tag>`. `LaunchDecision` should carry both `Tag` and derived `SessionName` when the action needs a zellij session so printouts and comparisons cannot mix the two forms.

The implementation should keep business decisions pure (`ARCH-PURE`) and reuse the dispatcher introduced in #74 instead of creating a parallel command parser (`ARCH-DRY`). The slice must still satisfy the issue purpose (`ARCH-PURPOSE`): it is not enough to port helpers; `pair-go launch` has to exercise a coherent launcher decision surface.

## Done when

- [x] A guarded Go launcher path can exercise a documented subset of launcher behavior.
- [x] Existing `bin/pair` remains the default public launcher.
- [x] Tests cover the ported decision logic and at least one process-level fake for external commands.
- [x] Any behavior not yet ported fails explicitly rather than silently diverging.
- [x] Pair remains usable after merge through the existing public command.

## Plan

- [x] Select the launcher slice from #73.
- [x] Extract pure decision models and tests.
- [x] Add fake-command process tests for the selected slice.
- [x] Implement the guarded Go path.
- [x] Document remaining shell-owned launcher behavior.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: greenfield-go-module design=1.00 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 3.75
```

## Log

### 2026-06-26

Created from #72. This issue is deliberately a prototype/vertical-slice milestone, not the public switch.

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/internal/dispatcher -run 'TestDispatch(Help|Launch)' -count=1; go test ./cmd/pair-go -run 'TestRunLaunch' -count=1; go run ./cmd/pair-go launch reached prototype decision exit 3; go run ./cmd/pair-go help lists launch as prototype; go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check; review verdict: SHIP
- 2026-06-29: closed — go test ./cmd/internal/launcher ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; git diff -- bin/pair empty; rg atlas boundary check; git diff --check; review verdict: REWORK

Claimed #75 after parking #82. Entered planning with `sdlc start-plan --issue 75`; design cites #73's inventory and chooses a guarded `pair-go launch` decision-phase prototype so the shell launcher remains public while the Go path exercises real launcher concepts (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).

Plan-quality gate returned FAILURE: estimate was low for the visible multi-surface scope, and tag/session naming was ambiguous. Revised the estimate to 3.75 and clarified that canonical tags are bare while zellij session names are derived as `pair-<tag>`.

Second plan-quality gate returned FAILURE: the process-level test was ordered before the dispatcher route it needs, and the plan risked duplicating `dispatcher.Result`. Reordered route before process test and made `dispatcher.Result` the sole process-facing result abstraction (`ARCH-DRY`).

Third plan-quality gate returned FAILURE: Task 4 still named a `LaunchResult` return despite the single-result-abstraction rule. Revised the runner contract so launcher returns domain `LaunchOutcome` values and dispatcher alone maps to `dispatcher.Result`; also named the production IO constructor and test runtime seam.

Implemented `cmd/internal/launcher` as a pure decision-phase core plus fakeable zellij/history seams. `pair-go launch` now routes through the #74 dispatcher, returns explicit prototype decisions, and does not mutate zellij or replace `bin/pair`. Updated atlas architecture and the Go migration inventory to record the shell-owned boundary.
