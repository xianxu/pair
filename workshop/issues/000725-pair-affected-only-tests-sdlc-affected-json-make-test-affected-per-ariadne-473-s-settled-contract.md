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

### Done when Revision — 2026-10-20

The original real-status/train requirement above is qualified by upstream main-only configuration authority. Per hub approval 36f3d275-8e19-4e86-a4b4-10d175c5c020, #725 closes at landing after fixtures and representative native selected-command timing against the full suite, with numbers in the Log. After configuration lands, the hub picks the next real code branch for sdlc-local/affected and the first docs-only branch for train vouch, recording this follow-up in test-economy. No recorded status or train acceptance is claimed before it exists.

## Plan

See [durable plan](../plans/000725-affected-tests-plan.md), one atomic close boundary.

- [ ] Add declaration, selective command and production entrypoint regressions; qualify upstream strict parser/planner fixtures.
- [ ] Audit current non-Go recipe ownership; document the Lua/runtimebundle exception and full fallbacks.
- [ ] Record fixtures and native selected-command timing against full suite; note hub-owned post-landing recorded affected/docs-vouch follow-up in the Done when Revision.
- [ ] Run full suite, close review, request operator local-gates label through hub, then merge and verify outcome.

## Log

- Deps: ariadne#473 (M2 adds the gates that consume it; this can land first: absent the gates, the file is inert).

- 2026-10-20: Claimed pair:3 and started from fresh main ab32acea6c97663232623b2881e7e7c91aae8a17. Refreshed provider to current main; preserved unrelated generated workflows in 725-start-plan-generated-workflows-preserved-for-699 (older workflow stashes untouched). TL clarification 5626d3a8-45f9-4ea4-9ff5-48620198f6ab: nvim source selects lua and the runner regenerates/tests runtimebundle; generated embedded Lua selects runtimebundle+lua. Implement fixtures now, measure/record real affected evidence once M2 lands. Five groups reuse existing targets and unclaimed non-Go files force full (ARCH-DRY, ARCH-PURPOSE).

- 2026-10-20: TL approved plan via 8505a073-10d2-4bf5-8c07-8b6b8d9f5ab6. Added counted exact leaf-plus-importer Go argv fixture, explicitly excluding ./... widening; entering full-flow change-code.

- 2026-10-20: PQ-1 addressed by explicit pure validation/argv functions and named IO dispatch functions with adversarial strategies, rather than feature-level case inventories. Hub confirmed main-only-config acceptance ordering (36f3d275-8e19-4e86-a4b4-10d175c5c020); revised Done when explicitly preserves post-landing test-economy follow-up ownership.

- 2026-10-20: Plan-quality round 2 CLEAN with PQ-1 addressed. Derived estimate only after acceptance, using v3.1 primitive method (2.205h →2.21h), with stale-calibration caveat. No implementation started.

## Revisions

- 2026-10-20: Added executable plan and explicit source-Lua/runtimebundle test exception approved by hub. No upstream schema or environment contract changes. Real recorded affected/train acceptance is pending upstream M2 availability rather than being waived.

- 2026-10-20: Acceptance ordering revision approved by hub: prelanding fixture/native timing + full suite; hub-owned actual affected/status and docs-only train proof after config reaches main. Clarified pure runner boundary for PQ-1 without changing behavior or upstream contract.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The calibration is marked stale against its newer ledger, so the units remain provisional.

Map schema/group ownership to a smaller-module extension: 0.3h design ×0.2 settled-spec discount =0.06h, 0.5h implementation ×0.4 v3.1 scale =0.20h. The new Python execution runner maps to a single-concern greenfield module: existing subprocess/Make/private-binary libraries halve 2.0h design, then ×0.2 =0.20h; 0.8h implementation ×0.4 =0.32h. The upstream Go consumer fixture uses the same greenfield primitive and available parser/graph library: 0.20h design/0.32h implementation. Make recipe integration is another smaller-module extension at 0.06h/0.20h. Docs: 0.2h ×0.2 =0.04h design, 0.2h ×0.4 =0.08h implementation. One close review: 0.2h ×0.2 =0.04h design, 0.5h ×0.4 =0.20h implementation. Issue shaping: 0.5h ×0.2 =0.10h design, 0.2h ×0.4 =0.08h implementation. Familiarity is 1.0 throughout; no novel stack/API discovery or vendor propagation. Design subtotal 0.70h ×1.15 thorough-plan buffer + implementation 1.40h =2.205h, rounded to 2.21h. Suite execution and queue delays are measured separately rather than padded into primitives.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.20 impl=0.32
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
item: issue-spec design=0.10 impl=0.08
design-buffer: 0.15
total: 2.21
```
