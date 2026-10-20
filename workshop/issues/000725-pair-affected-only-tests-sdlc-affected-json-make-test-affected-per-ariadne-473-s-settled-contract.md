---
id: 000725
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '74042514b334d812e24e650177339cfe7d0735ab' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T02:22:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract

## Problem

ariadne#473 (test-economy, option B′) lets a close and a landing-train car use an **affected-only** test run as
evidence, with the full suite running once per train. It's off per repo until the repo ships `.sdlc/affected.json`
and a command that runs only the affected tests. pair has neither, so pair keeps running a full suite (~9 min) for
every car, and a docs-only branch (pair#723 on 10-19) can't vouch for a train at all.

## Spec

The contract is settled by ariadne#473 M1 (milestone evidence a6c87575, fake-consumer fixture). Implement pair's side:

- **`.sdlc/affected.json`** (strict: unknown fields and duplicate keys refuse): `version: 1`, `command` (argv),
  `weight`, `timeout_minutes`, `full` (globs that force the full suite: e.g. `Makefile*`, `go.mod`, `go.sum`,
  `tests/lib/**`, the CI workflows), and `groups: [{name, paths (globs, ** ok), go (package dirs like ./cmd/pair whose
  dependency on a changed package selects the group)}]`.
- **The command** (for example `make test-affected`): sdlc runs it at the head worktree root, under the suite slots,
  with `SDLC_AFFECTED_GO_PKGS` (sorted, space-separated import paths: changed packages plus every package importing
  them, test imports included) and `SDLC_AFFECTED_GROUPS` (sorted, space-separated group names). It runs `go test` on
  exactly those packages and each selected group's non-Go tests (scripts, Lua specs), printing the suite's own
  `go test` output so failures attribute. It is never called with both empty (docs-only is recorded by sdlc without
  a run) and never for a full selection (sdlc runs the suite instead).
- **Two rules from ariadne's review:** (1) a file owned by no Go package and claimed by no group forces the full suite,
  so groups must claim the `tests/` and `scripts/` paths pair wants selective; (2) Lua embedded by runtimebundle
  selects that Go package and the lua group.

## Done when

- `affected.json` validates against ariadne's strict parser; a fixture shows a one-package change runs that package
  plus its importers only, a scripts-only change runs only its group, a Makefile change selects full.
- On a real pair branch, the affected run is recorded (`sdlc-local/affected`) and is materially faster than the suite
  (numbers in the Log), and a docs-only branch vouches for a train without running tests.
- It's a local-gates change (`.sdlc/`), so the PR needs the operator's `local-gates-verified` label.

## Plan

See [durable plan](../plans/000725-affected-tests-plan.md), one atomic close boundary.

- [ ] Add declaration, selective command and production entrypoint regressions; qualify upstream strict parser/planner fixtures.
- [ ] Audit current non-Go recipe ownership; document the Lua/runtimebundle exception and full fallbacks.
- [ ] After ariadne#473 M2 lands, record real affected and docs-only/train evidence and compare measured duration to full suite.
- [ ] Run full suite, close review, request operator local-gates label through hub, then merge and verify outcome.

## Log

- Deps: ariadne#473 (M2 adds the gates that consume it; this can land first: absent the gates, the file is inert).

- 2026-10-20: Claimed pair:3 and started from fresh main ab32acea6c97663232623b2881e7e7c91aae8a17. Refreshed provider to current main; preserved unrelated generated workflows in 725-start-plan-generated-workflows-preserved-for-699 (older workflow stashes untouched). TL clarification 5626d3a8-45f9-4ea4-9ff5-48620198f6ab: nvim source selects lua and the runner regenerates/tests runtimebundle; generated embedded Lua selects runtimebundle+lua. Implement fixtures now, measure/record real affected evidence once M2 lands. Five groups reuse existing targets and unclaimed non-Go files force full (ARCH-DRY, ARCH-PURPOSE).

## Revisions

- 2026-10-20: Added executable plan and explicit source-Lua/runtimebundle test exception approved by hub. No upstream schema or environment contract changes. Real recorded affected/train acceptance is pending upstream M2 availability rather than being waived.
