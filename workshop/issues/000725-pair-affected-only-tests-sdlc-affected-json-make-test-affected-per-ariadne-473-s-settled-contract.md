---
id: 000725
status: codecomplete
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 2.21
collaboration-mode: collaboration
card_mirror: 'ea4d6b3430ec9761138dbb5b3d1aac57c5e61cc3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T02:22:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.89
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

### Done when Revision — file readers

Hub approval 6c4bd7ed-04db-4159-94a2-ad131ea75534 supersedes script-only and source-Lua-only selectivity: non-Go populations with Go file readers force full. Go selections run a fixed named source-scanner preflight followed by exactly the provider-selected package argv. Tests assert the conservative fallbacks, reader inventory and constant invocation count.

## Plan

See [durable plan](../plans/000725-affected-tests-plan.md), one atomic close boundary.

- [x] Add declaration, selective command and production entrypoint regressions; qualify upstream strict parser/planner fixtures.
- [x] Audit current non-Go recipe ownership; document the Lua/runtimebundle exception and full fallbacks.
- [x] Record fixtures and native selected-command timing against full suite; note hub-owned post-landing recorded affected/docs-vouch follow-up in the Done when Revision.
- [ ] Run full suite, close review, request operator local-gates label through hub, then merge and verify outcome.

## Log


- 2026-10-20: closed — Recorded full suite PASS on bc1e83cd in560.511449334s, queue3116.433279s separate, attempt488fd46fb54e5c917f576b83ac50a91f log3b991bdaa414. Runner11 and native-reader8 PASS; upstream parser/planner separately qualified113e7767 for747 audited file inputs. Exact native selected command87.553s incl19.542s source preflight,84.38 percent less execution than full. BR4 complete reader re-audit, independent source-walker registration, actual affected seeded violations require both scanner failures. no-plan-check retains only pending close/operator label/landing checklist; implementation and suite verified. Main-authoritative recorded affected/docs-train follow-up remains hub-owned after landing.; review verdict: SHIP
- Deps: ariadne#473 (M2 adds the gates that consume it; this can land first: absent the gates, the file is inert).

- 2026-10-20: Claimed pair:3 and started from fresh main ab32acea6c97663232623b2881e7e7c91aae8a17. Refreshed provider to current main; preserved unrelated generated workflows in 725-start-plan-generated-workflows-preserved-for-699 (older workflow stashes untouched). TL clarification 5626d3a8-45f9-4ea4-9ff5-48620198f6ab: nvim source selects lua and the runner regenerates/tests runtimebundle; generated embedded Lua selects runtimebundle+lua. Implement fixtures now, measure/record real affected evidence once M2 lands. Five groups reuse existing targets and unclaimed non-Go files force full (ARCH-DRY, ARCH-PURPOSE).

- 2026-10-20: TL approved plan via 8505a073-10d2-4bf5-8c07-8b6b8d9f5ab6. Added counted exact leaf-plus-importer Go argv fixture, explicitly excluding ./... widening; entering full-flow change-code.

- 2026-10-20: PQ-1 addressed by explicit pure validation/argv functions and named IO dispatch functions with adversarial strategies, rather than feature-level case inventories. Hub confirmed main-only-config acceptance ordering (36f3d275-8e19-4e86-a4b4-10d175c5c020); revised Done when explicitly preserves post-landing test-economy follow-up ownership.

- 2026-10-20: Plan-quality round 2 CLEAN with PQ-1 addressed. Derived estimate only after acceptance, using v3.1 primitive method (2.205h →2.21h), with stale-calibration caveat. No implementation started.

- 2026-10-20: Full-flow change-code passed: plan-quality round 3 CLEAN (PQ-1 addressed), estimate-quality INFO. Estimate verification allowance is included in each component, not excluded: runner owns recorder/entrypoint/failure/mutation tests, Make integration owns recipe audit, consumer fixture owns strict-parser/graph scenarios. Residual design is exact safe recipe-path ownership and portable upstream-source fixture construction. Suite elapsed execution is covered by component verification/one boundary qualification, counted once across overlap; the Log separately reports runtime and idle queue, and sdlc actual determines adopted attribution rather than this estimate text. No implementation started.

- 2026-10-20: Implemented validated pure selection/argv functions, thin child runner and five fixed Make group targets. Runner regressions went red before code (missing command/runner), then passed all 8 test methods, including counted N=1/2 selections, zero effects on invalid input, one private binary lifetime, environment scrub, failure propagation and disjoint default-recipe ownership. Added explicit rejection for Go's embedded ... pattern after a failing regression; input never reaches shell evaluation (ARCH-SECURE).

- 2026-10-20: Stateful recorder mutations detected removal of wildcard rejection, Lua/runtimebundle addition, exact-Go argv isolation and recipe ownership. Actual upstream #473 candidate fixture passed strict nested/top-level unknown/duplicate refusal, leaf/transitive/test-importer unions, script-only group selection, editable/generated Lua attribution and full fallbacks; every declared root validated against real Pair graph. Provider read via Git into disposable source module, never modified. Default unavailable-provider check reports a dependency skip; explicit requested source failure refuses. README and atlas document this pending upstream dependency and operator label ownership.

- 2026-10-20: Actual upstream planner selected dispatcher, slugcmd and cmd/pair-go (no groups, Full=false) for cmd/internal/slugcmd/slug.go. Native make test-affected with exactly those three import paths passed in 13.396s wall time (no queue): package outputs 0.243s, 0.692s and 11.604s respectively. This is command qualification, not recorded sdlc-local/affected status; compare against the pending recorded branch full suite before label/landing. Hub retains post-landing real affected/docs-train follow-up per governing Done when Revision.

- 2026-10-20: Close round 1 REWORK, BR-1: command-family roots missed real integration consumers. Cancelled the queued b44540e1 suite before execution (no pass claimed) and audited every five-group recipe recursively. Added independent per-target command inventory and actual-graph dependency/text-input regressions; both failed before declarations were corrected. All explicit upstream fixtures and 8 runner tests then passed. The class sweep includes reviewapply/retentioncmd/gccmd/pairlog/reviewcmd/hoprttcmd/launcher/slugcmd/generator CLI/opener/wrapcmd/probe, plus Make environment scanner's source-text dependencies. Additive build coverage for cmd/probes Go sources and already-claimed non-Go paths preserves unknown fixture full fallbacks (ARCH-PURPOSE). Original 13.396s timing predates this conservative coverage correction; qualify updated actual selection before comparison. No approval/label or recorded affected/train evidence claimed.

- 2026-10-20: Actual upstream absent-owner probes showed broad cmd/probes .go claims would turn deleted/platform-excluded packages into build-only selection, weakening conservative full fallback. Removed those experimental claims before publication. Proposing fixed environment-inventory preflight (skip when build already owns it), with red recorder tests and restored absent-package full regressions, to hub for approval before runner changes.

- 2026-10-20: Hub approved fixed preflight (delivery 29a53677-8658-419c-84d3-1ef1cc04ff30, replying to submitted request 23093c3f-af65-440f-b2de-7214b1c3924c). Implemented make test-make-env once after selection validation unless build already owns its run. 9 runner tests pass; mutations removing the preflight or duplicating it for build both fail their regressions. Actual upstream source pinned at d6309691af55cb39765a55623ed01f5ab31212e2 passes parser, graph, per-recipe/transitive command coverage and absent/deleted/platform-excluded package full-fallback fixtures. No broad cmd/probes path claims remain.

- 2026-10-20: Fixed guard measured independently at 35.095s wall (PASS, 308 derived knobs preserved, session identities dropped). Updated actual upstream selection for slugcmd/slug.go is the same three Go import paths plus build (install-layout exercises pair-slug). Native make test-affected with that exact selection passed in 86.939s wall, no suite queue: dispatcher 0.265s, slugcmd 0.776s, cmd/pair-go 18.126s, plus all build-group recipes. This supersedes the initial pre-audit 13.396s sample. Full-suite execution/queue comparison pending on final code; no recorded affected status claimed.

- 2026-10-20: Close round 2 REWORK. BR-1 roots addressed (reviewer mutation verified), BR-2 found omitted cross-language inputs: nvim/init.lua and zellij_trace.lua omitted shell suites. Cancelled queued suite before execution. Stopped for a complete coverage re-plan, extending the independent inventory from Go roots to every recursively executed/read/copied file input. Broad nvim ownership is conservative across dynamic Lua imports; shared fixtures retain full unless all consumers covered. Requesting hub approval for this complete rule rather than incremental instance fixes.

- 2026-10-20: Timing decision requested by hub: all five non-Go groups under one private-binary wrapper PASS in 286.506s, no suite queue (includes generation/candidate build). Prior recorded integrated #721 suite e7820805aba7f2d6552865b320a1bca196c02878 is 543.7159215s execution, with 1237.640566s queue separately (attempt 0db406db08acf821356ba02244338ed4). These are separate runs/heads, not a same-run phase split: non-Go is 52.69% of that baseline and residual Go/overhead roughly 257.210s. Hub confirmed complete input rule over always-all-non-Go (deliveries ca2de46d-0519-438e-b160-13e75516c6c8 and c8565b38-1938-4992-a77f-39351cfeb85a; submitted report 27c5d696-1c63-4087-b6d1-59582d4a5d23).

- 2026-10-20: Completed recursive file/fixture/text audit for every grouped recipe, extending independent inventory to go/files. Actual upstream Plan/Pair graph regression went red on omitted consumers, then passed 747 distinct group/file inputs alongside all command dependencies. Added conservative build/shell nvim ownership, build runtime asset inputs, review Zellij config and Lua shared perf fixture; shared build-go-atomic script now forces full. Unclaimed external fixtures and absent/platform-excluded packages still full; no broad cmd/probes/tests path claim. Text-inventory guard remains fixed. Mutations removing shell's nvim coverage or Lua's shared fixture consumer fail. All 9 runner tests pass. Final slugcmd selection remains three Go packages + build with unchanged runner/recipes, so the 86.939s native command sample still qualifies that selection; full-suite comparison on final branch pending.

- 2026-10-20: Close round3 REWORK BR-3 confirmed group.go only triggers non-Go recipes; Go tests reading selected non-Go inputs were omitted. Hub approved conservative full fallback for non-Go file-reader populations, then fixed Go source-scanner preflight for Go changes. This supersedes selective labels.py/source-Lua acceptance; group-free Go argv remains exact. Import-only graph source-reader limitation is recorded here for ariadne#473. Runner regression red: N=1/2 observed one Go call instead of required scanner+selected two; after implementation all 11 runner tests PASS, including child failure before downstream effects. Native fixed named-scanner preflight PASS 29.827s, no suite queue. Prior 86.939s selected timing is superseded; rerun final command and full suite. No passing suite has yet been claimed for #725.

- 2026-10-20: Final representative native command PASS 80.679s wall with actual planner's dispatcher/slugcmd/cmd-pair-go plus build; fixed source preflight 19.753s within that run (standalone prior sample29.827s). No queue. Fixed preflight includes probe cleanup and the couchcore plan256 source contract found in the repository-wide audit. Runner11 tests and reader-inventory6 tests PASS; reader tripwire covers448 lexical matches (25 fixed scanner files,51 non-Go/full fallbacks,372 local/synthetic/same-package fixtures). New/changed/deleted reader mutations and removed termcmd registration refuse. This is a lexical re-audit tripwire, not arbitrary dataflow analysis. Upstream parser/planner qualification pinned113e7767719b5b4b4964ba9a8f5e73fecce825b8 PASS, including747 non-Go audited inputs forcing full and actual selective Go roots/dependencies. Full branch suite and its measured execution/queue comparison pending.

- 2026-10-20: Close round4 REWORK BR-4 verified: deleted-vocabulary and teardown-proof production-source walkers were mistakenly marked local fixtures. Cancelled queued f7a52217 suite before execution. Re-audited all372 local dispositions, tracing helper-mediated reads:126 non-couch/a-l/root files,121 couch files and125 m-z files. Added repository-wide vocabulary/teardown/project/trace readers and explicit sibling-test consumers; reclassified actual non-Go captures/scripts versus generated fixture data. An independent conservative source-walker detector now overrides local classifications for git ls-files, parent-relative direct reads, or outside-package Go-source walkers/parsers. Its tests deliberately mark walkers/readers local and still require every named test in the preflight (stronger than package membership). Corrected inventory449 files =56 fixed scanner,86 non-Go fallback,307 local/own-package/immutable-history; all28 fixed packages/400 named tests run in one fixed call. Native seeded violation regression was red (affected command wrongly exited0 in55.648s), then green after correction: actual affected command reports BOTH vocabulary and teardown scanner failures for a disposable slugcmd source mutation and never enters selected groups. Runner11 tests PASS. Final selected timing/full suite comparison pending after inventory correction; previous80.679s sample superseded.

- 2026-10-20: Final corrected native selection PASS87.553s wall, scanner preflight19.542s included, no queue. Scanner argv unchanged by adding the newly detected helper-only internal/repolint/repository_test.go to inventory (same existing preflight package/catalog tests). Independent ls-files fixture initially exposed the helper-only detection omission; detection and helper-package registration now enforced. Full suite/review requeue follows final focused checks.

- 2026-10-20: Final focused verification: all8 reader-inventory/native-regression tests PASS58.207s; all11 runner tests PASS16.511s. Source-walker cases cover Walk, WalkDir, direct parent ReadDir/ReadFile and git ls-files under an intentionally incorrect local disposition. The native regression requires both independent scanner failures.

- 2026-10-20: Recorded full suite PASS on bc1e83cdce3bfa8c368f5de116bfe932650f8a19, attempt488fd46fb54e5c917f576b83ac50a91f: execution560.511449334s (9m21s), queue3116.433279s reported separately, log SHA2563b991bdaa4148aec1b58d08d1339cdf1ee7a800e8fa23df7ac4ff8e2ef747d1b. Source-reader8 native checks and runner11 pass within suite; automatic upstream fixture explicitly skips because source is absent in disposable checkout, qualified separately against113e7767. Representative selected command87.553s versus full560.511s on the same code revision:15.62% of suite execution,84.38% less (6.40x faster), separate wall runs with no queue in the selected sample. This includes fixed source preflight19.542s. No sdlc-local/affected/train vouch is claimed before main config; hub owns postlanding proof. Earlier close retry was refused before review by missing slot provenance on unrelated train174 attempt; hub restored pair:7 provenance (original preserved), filed ariadne#492, and sdlc actual now measures successfully.

- 2026-10-20: Close round5 SHIP,0 new findings, BR1–BR4 disposed, all architecture markers pass. Reviewer re-ran runner11/reader8 and actual upstream qualification, and removing the BR4 registrations fails the regression as intended. Card codecomplete bound to evidence70d7c719a8df48c6e2e4c1f51163371569c8f38e; measured actual3.89h excludes51.94 queue minutes. Operator local-gates-verified remains the only current CI blocker; request through hub next, never self-label.

## Revisions

- 2026-10-20: Added executable plan and explicit source-Lua/runtimebundle test exception approved by hub. No upstream schema or environment contract changes. Real recorded affected/train acceptance is pending upstream M2 availability rather than being waived.

- 2026-10-20: Acceptance ordering revision approved by hub: prelanding fixture/native timing + full suite; hub-owned actual affected/status and docs-only train proof after config reaches main. Clarified pure runner boundary for PQ-1 without changing behavior or upstream contract.

- 2026-10-20: Governing acceptance revision: non-Go source populations with Go readers force full rather than claiming script/source-Lua-only selectivity. Fixed scanner preflight runs for any nonempty selected Go package list, then the exact selected argv. Hub explicitly approved fallback and this final constant-cost correction; no further design approval needed. Prelanding representative timing/full suite/operator label and postlanding hub-owned recorded affected/docs vouch remain required.

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

- 2026-10-20: BR-1 correction expands group dependencies to all audited real command and source-reading consumers; current selections include additive build coverage for the environment-inventory guard. This supersedes the initial sparse root mapping, with regressions tied to the actual upstream planner and Pair graph.
