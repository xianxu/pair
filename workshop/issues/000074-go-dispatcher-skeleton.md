---
id: 000074
status: done
deps: [000073]
github_issue:
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 1.39
started: 2026-06-29T17:00:26-07:00
actual_hours: 0.14
---

# pair Go dispatcher skeleton

## Problem

The target architecture needs a primary Go command, but switching the public launcher immediately would be too risky. The first code step should introduce the dispatch shape without changing user-visible behavior.

## Spec

Add a Go dispatcher skeleton that can host Pair subcommands behind an explicit development path. It should establish command parsing, help text shape, version/build metadata if needed, and an internal routing pattern for future subcommands.

The existing `bin/pair` shell launcher remains the public entrypoint for this issue. Any new Go command must be opt-in, for example a new built binary or a hidden/dev invocation, so this can merge without affecting normal sessions.

Design constraints:

- Reuse existing package structure where possible (`ARCH-DRY`).
- Keep command parsing and dispatch decision logic pure enough to unit-test (`ARCH-PURE`).
- Do not port launcher behavior yet. This issue is only the skeleton.

## Done when

- [x] A Go dispatcher command exists and builds in the normal `make build` flow or an explicitly documented dev target.
- [x] Dispatcher help lists the planned command families without claiming unsupported behavior works.
- [x] Public `bin/pair` behavior is unchanged.
- [x] Tests cover dispatch parsing and unsupported-command errors.
- [x] Pair remains usable after merge through the existing `pair` entrypoint.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: skill-or-dispatcher design=0.30 impl=0.25
item: smaller-go-module design=0.20 impl=0.20
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 1.39
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

## Plan

- [x] Choose the command location/name based on #73 inventory.
- [x] Add pure dispatch parsing tests.
- [x] Add the skeleton implementation.
- [x] Wire build/install only in a non-disruptive way.
- [x] Verify existing Pair flows still use the shell launcher.

## Log

### 2026-06-26

Created from #72 as the safe first code-bearing step toward one primary Go command.

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1; make -B pair-go; go test ./... -count=1; bash tests/pair-continue-test.sh; git diff -- bin/pair empty; review verdict: SHIP

Claimed and entered planning. Design uses a new opt-in `pair-go` binary plus pure `cmd/internal/dispatcher` package so command parsing/help/error behavior is unit-testable (`ARCH-PURE`) and `bin/pair` remains unchanged while the skeleton fulfills the issue purpose (`ARCH-PURPOSE`). Durable plan: `workshop/plans/000074-go-dispatcher-skeleton-plan.md`.

Implemented `cmd/internal/dispatcher` and `cmd/pair-go` behind a non-public `pair-go` build target. Wrapper behavior is covered by `cmd/pair-go/main_test.go` rather than subprocess tests because the pure dispatcher tests cover parsing/help/error semantics and the wrapper only writes the returned streams/exit code. Verified with `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache make -B pair-go`, `env GOCACHE=/private/tmp/pair-go-cache GOMODCACHE=/private/tmp/pair-gomod-cache go test ./... -count=1`, `bash tests/pair-continue-test.sh`, and `git diff -- bin/pair` (empty).
