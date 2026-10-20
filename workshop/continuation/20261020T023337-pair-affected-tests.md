---
type: continuation
slug: pair-affected-tests
agent: codex
created: 2026-10-20T02:33:37
supersedes: native-recovery-candidates
branch: 000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract
worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
issues: [000721, 000725]
---

# Continuation: pair-affected-tests

## NEXT ACTION

Resume pair#725 on the current issue branch in /Users/xianxu/workspace/worktree/pair-slot3/pair. Read its issue and workshop/plans/000725-affected-tests-plan.md, run sdlc state and tl-worker --help. Full-flow change-code has PASSED (accepted-design commit d13ef691e); no implementation or tests have been written yet. Start the plan with TDD: write tests/test-affected-test.py driving the actual make test-affected entrypoint with stateful go/make recorder executables, plus pure parse_selection/go_test_argv tests; run red before creating scripts/test-affected.py or editing Makefile.local. The plan owns all five group mappings and the conservative declaration strategy. Use executing-plans and TDD skills; no new approval is needed to begin.

## State of play

The whole recovery train is landed: #712/#715/#714 and #721 (PR479, merge a0b7ee338a732e4d6bf36e3e99cf3f1006925275). #721 close round2 SHIP, original full suite9m17s and integrated9m4s pass, tracker/archive verified; final report receipt9ba221c4-3186-4a4c-9268-9c49692d7576 submitted. Hub owns next natural recovery calibration; no pair:6 action or staging. Do not redo recovery work.

Hub ops:0 dispatched #725 next. It is claimed/working in pair:3, branch 000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract, based on fresh main ab32acea6c97663232623b2881e7e7c91aae8a17. Current last hand commit09eee7865 checkpoints admission; writer will add its continuation commit/push. Tree was clean. No PR yet. One atomic close boundary; full-flow pinned because declaration/runner will exceed quick shell. Plan-quality PQ-1 was named-function/adversarial-strategy clarity: addressed; rounds2/3 CLEAN. Estimate2.21h derived only after plan-quality using v3.1 and stale calibration caveat; estimate-quality INFO only, passed. Log records verification allowance/residual decisions/time-accounting clarification.

## Thread arc & user model

The operator/hub wants actual fleet time savings after repeated long full-suite queues and hourly landing waits. Consume the settled upstream contract, preserve safety through conservative full fallbacks, and prove selectivity with invocation counts and real timing. Keep moving without redundant permission questions, but local-gates verification belongs to the operator: prepare the tested PR and ask hub for the label, never self-label or bypass CI. User approved fixture/native timing before landing, with first real recorded affected/docs-train proof owned by hub after config reaches main.

## Artifact map

Read workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md first: copied strict schema/env contract plus governing Done when Revision. Then workshop/plans/000725-affected-tests-plan.md: pure TestSelection/parse_selection/go_test_argv, IO main/run_selection, five group-to-existing-target mappings, function-level adversarial strategies and full landing steps. Its gate ledger uses the long canonical issue slug under workshop/plans; no open blocking findings. The two are committed.

Makefile.local owns default full test recipe, environment scrub, private binary wrapper and reusable group targets. Root Makefile includes workflow then local. tests/lib/with-pair-test-binary.sh generates runtime tree, builds one private fresh Pair and exports PAIR_TEST_BINARY for its child scope. cmd/internal/runtimebundle/embed.go embeds ignored generated assets/runtime/manifest.json and files tree; editable sources live in nvim/.

Upstream provider is /Users/xianxu/workspace/worktree/pair-slot3/ariadne. Its main was refreshed to438da630ea1dcdf464df0ed839ebe455ffd339ab, which lacks #473. git fetch origin was run and origin/000473-test-economy-affected-suite-as-close-evidence-the-full-suite-runs-once-in-the-train-or-at-merge-option-b is available, including M1 close a6c87575cccdf71a1b067206a4b91d94527207f7. Read its cmd/sdlc/internal/affected/{config,plan,graph,golist}.go via git show for real ParseConfig/Plan/LoadGoGraph API. Do not edit upstream or change its checkout. A disposable source fixture can use that candidate; installed provider will gain it after M2 lands. AGENTS.local.md read; no MEMORY.md exists. Group.go is a trigger, never adds Go packages. Path ownership and group paths are additive; unclaimed non-Go forces full; testdata/embed ownership is exact. Actual affected run reads config from origin/main, never branch.

## Decisions & dead ends

Authenticated hub receipts, all submitted: dispatch df3fc725-a1d9-4c78-a693-d2bf639e2e10; Lua clarification5626d3a8-45f9-4ea4-9ff5-48620198f6ab; plan approval8505a073-10d2-4bf5-8c07-8b6b8d9f5ab6; acceptance-order approval36f3d275-8e19-4e86-a4b4-10d175c5c020. The latter is governing: before operator label/landing run fixtures and representative native selected-command timing against full suite in Log. After config lands, hub chooses the next real code branch to record sdlc-local/affected and first docs-only branch to vouch for train; hub records test-economy follow-up. #725 closes at landing with this noted in Done when Revision. Never fake or prematurely claim recorded affected/train evidence. #725 changes Makefile/.sdlc and therefore must run full suite itself.

Lua rule: nvim source selects lua; runner regenerates mirror and tests runtimebundle, adding it once if not already in selected Go argv. Generated embedded Lua selects Go package plus lua group naturally. Pin both. Counted leaf-package fixture must invoke go test with exactly leaf plus selected importers, never ./.../unselected packages. Package inputs are import paths, not arbitrary shell strings.

Five groups: lua/review/doctor/build/shell reuse existing non-Go Make targets. Do not duplicate pure-Go test-pair-title. Avoid ./cmd/pair-go as group root (nearly all Go changes would trigger everything); main/dispatcher changes force full. Validate Go roots (./cmd/pair is nonexistent). Do NOT broadly claim tests/**: tests/webcast/**, broadcast-viewer/**, terminal-oracle/** are Go-owned external fixtures, and a paths-only group match does not execute their Go consumer. Leave these/unknown/live-only helpers unclaimed or explicitly full. Exact default-script and scripts/supervision-labels claims keep real script edits selective. Shared tests/lib, Makefiles, .sdlc and CI config force full.

No new scheduler/state/background worker. No production-source Go files planned. Existing source inventory guard still applies if that changes. Python selected runner and upstream consumer fixture qualification need portable scratch/cleanup. Test configuration through upstream parser/planner rather than reimplementing dependency planning. Plan suggests disposable Go fixture module using provider affected/jsonstrict sources, selectable candidate provider before M2; keep this integration portable and tests effective.

## Lessons learned

The #725 brainstorm already learned main-only config authority makes prelanding real affected status impossible; hub explicitly revised acceptance rather than bypassing authority. Keep the original requirement plus its governing revision. Plan-quality requires named risky functions and one adversarial strategy each, not feature areas or prose case inventories.

Keep unrelated generated workflows out of this issue. start-plan weave refresh dirtied .github/workflows/local-gates-guard.yml and merge-check.yml; preserved in new stash725-start-plan-generated-workflows-preserved-for-699. Both older stashes715-start-plan-generated-workflows-preserved and721-start-plan-generated-workflows-preserved-for-699 remain untouched. Do not apply/drop them or include these workflows.

Bare whole-shell couch/sdlc commands only, no pipes/chaining/substitution. Read skills and fresh binary help as required. Agent trailer: Agent: codex/gpt-6.1-sol/medium; Co-Authored-By: GPT-6.1 Sol <noreply@openai.com>. Commit design/work checkpoints. Gate owns fresh review; no duplicate ad-hoc reviewer. Record full suite on pushed head, may overlap close review; pass required before landing. Ask hub for operator local-gates label once PR concrete/tested, then merge normally. Report outcome once and verify receipt submitted. Writer owns restart; do not separately restart.
