---
collaboration-mode: discussion
---

# Pair affected tests implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use superpowers-executing-plans for the shared configuration/runner work; bounded read-only audits may use subagents.

**Goal:** Run only selected Go packages and declared non-Go test groups for branch evidence, preserving full-suite landing verification.

**Architecture:** Adopt ariadne#473's unchanged version-1 affected declaration and environment contract. Pair's command validates its selection, reuses existing Make targets, and preserves the private test-binary and environment boundaries. The lua group additionally regenerates and tests runtimebundle, because editable Lua is copied into its generated embed tree.

**Tech Stack:** JSON, GNU Make, Python 3 unittest/subprocess, Go, existing Bash/Neovim test targets.

## Core concepts

The dependency planner/config parser/Go graph remain owned by ariadne#473. Pair introduces only a validated execution selection, without a second dependency planner (ARCH-DRY, ARCH-PURE).

| Pure entity | Lives in | Status |
|---|---|---|
| `TestSelection` (immutable package/group tuples), `parse_selection`, `go_test_argv` | `scripts/test-affected.py` | new |

`parse_selection(package_text, group_text)` validates whitespace-tokenized inputs and returns TestSelection. `go_test_argv(selection)` derives the exact selected argv plus the declared Lua/runtimebundle exception. These read no environment/files and execute nothing; colocated assertions in tests/test-affected-test.py call them with literal values. `main` alone reads environment; `run_selection(selection)` alone executes child processes. No injected function-call mocks are needed for the pure functions.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Affected declaration | `.sdlc/affected.json` | new | upstream strict configuration parser and affected planner |
| Selected test runner | `scripts/test-affected.py` | new | selected `go test` invocation and existing make targets |
| Affected Make entrypoint and group target lists | `Makefile.local` | modified | environment scrub, private test binary, runtime generation and current test recipes |
| Runner regressions | `tests/test-affected-test.py` | new | real entrypoint plus disposable stateful child-command recorder/failure injection |
| Consumer configuration fixtures | `tests/affected-config-test.py`, `tests/affected-consumers.json` | new | upstream affected/config packages in a disposable Go fixture module and independently audited per-recipe command dependencies |
| Testing map | `README.md`, `atlas/index.md` | modified | operator command, group ownership, conservative fallbacks and label owner |

The declaration is static configuration, never rewritten by Pair. Unknown/duplicate fields remain the upstream parser's strict refusal contract: ignoring selection fields could skip required tests (ARCH-EVOLVE). No credentials or production state are consumed. Selection strings are process inputs; split whitespace into argv, reject malformed/foreign package paths, unknown groups, duplicates and empty selection before executing children (ARCH-SECURE). Never evaluate input as shell or Make syntax.

## Design

Reuse the default non-Go test targets in five groups; group target lists live once in Makefile.local and the runner requests the corresponding fixed target. Preserve every current default full-suite prerequisite, including the new runner/configuration guards. Do not run the pure-Go `test-pair-title` target in a non-Go group; selected Go packages already cover it. No default full-suite behavior is removed.

- **lua:** test-lua, test-retention, test-nvim-pidfile, test-lifecycle-command, test-queue, test-submission-transaction, test-bang-tag, test-autopair, test-draft-complete, test-pair-doctor, test-cr and test-statusline. Claims nvim/** and exact scripts/fixtures those targets use. Includes runtimebundle as a group Go root. Adds runtimebundle to the one Go test argv only when not already selected; regenerates before Go tests, per TL clarification 5626d3a8-45f9-4ea4-9ff5-48620198f6ab.
- **review:** test-review. Claims review scripts, overlay return, review target, and shared nvim sources; root reviewcmd.
- **doctor:** test-doctor, test-perf-capture, test-perf-key-conformance, test-emitter-health and test-adapt-schema. Claims their exact tests, doctor/**, adapt-log shim, nvim/adapt.lua and nvim/doctor.lua; command roots adapt/diagnosticcmd.
- **build:** test-build-go-atomic, test-supervision-labels, test-dev-rebuild, test-pair-go-install-layout, test-pair-embedded-runtime, test-make-env, test-host-preflight and test-run-headless. Claims exact tests, scripts/build-go-atomic.sh, scripts/supervision-labels/** and dev shims; roots runtimebundle/generator/entrypoint. A scripts/supervision-labels-only fixture selects this group without the Go suite.
- **shell:** test-session-watch, test-pair-restart-quit, test-pair-rename, test-changelog, test-copy-on-select, test-term-pane-shortcuts, test-zellij-trace, test-plan-superseded and test-paint-gate-consumers. Claims exact tested scripts and supporting zellij/shell paths; specific command roots sessionwatch/pairlifecycle/launcher/changelogcmd/scrollbackcmd/clipcmd/layoutcmd/termcmd/couchtty/ptychild.

Validate every group Go root against the actual `go list` graph. Do not use cmd/pair-go as a group root: it would select all groups for almost any production change. Instead make executable main/dispatcher source changes full-selection paths. Full also includes Makefile*, go.mod/go.sum, .sdlc/**, .github/workflows/**, tests/lib/**, and runtime bundle manifests/build inputs whose cross-cutting semantics require it.

Claim scripts and tests narrowly. In particular tests/webcast/**, tests/broadcast-viewer/**, tests/terminal-oracle/** and optional/live-only helpers have external Go consumers; leave unclassified or explicitly full. Broad tests/** would falsely claim coverage when its selected group does not execute the changed fixture. Match generated embedded Lua under runtimebundle's asset tree to lua, in addition to nvim/**. Package ownership and group selection remain additive.

`make test-affected` enters the existing test-goal environment scrub. Validate selection first, generate the ignored runtime tree when Go tests need it, and run one `go test <selected packages> -count=1 -timeout 20m` with inherited stdout/stderr. Add runtimebundle only for lua when missing. The existing private binary wrapper is used once for any selected non-Go groups; Go-only runs avoid building a whole Pair solely for the runner. Selected packages' existing TestMain/testpair boundaries retain their normal behavior. Group targets contain only the existing non-Go recipes. A failed child returns failure and cannot yield a passing record.

Actions: run affected, repair selection/config/tests and rerun, run full suite.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Valid package/group selection | Selected child test output | Run affected | Disposable test files only |
| Invalid/empty/unknown selection | Error before child effects | Repair and rerun | None |
| Child test/build failure or signal | Nonzero with original test output | Repair and rerun | Completed partial output remains; no passing evidence |
| Unknown path/full glob | Upstream full selection | Run full suite | Time only |
| Docs only | Upstream records no-run affected pass | Run affected | None |
| Upstream M2 unavailable | Qualification pending | Rerun after provider lands | No fabricated record or train vouch |

This is batch development tooling, not a UI hot path. Per invocation: one validation pass O(P+G), one Go test process, at most one binary-wrapper lifetime, one execution per selected group. Tests at one/two package and group selections pin invocation counts and no duplicate runtimebundle run (ARCH-CONSTRAINTS). Timeout/weight use existing 30-minute/one-slot limits; no new scheduler, background worker or persistent state (ARCH-ORDER). Scratch uses TemporaryDirectory under TMPDIR/TestScratch and context-managed cleanup; SDLC owns log/status retention (ARCH-FUNERAL). Stateful executable doubles retain argv/env and injected failure state; real fixtures exercise actual upstream parsing/planning and native make/Go boundaries (ARCH-MOCK). Full declaration is review-gated by the operator; no self-applied local-gates-verified label (ARCH-PURPOSE).

## Function-level verification

| Named owner | Adversarial strategy and mechanical guard |
|---|---|
| `parse_selection` (PURE) | Mutate literal token streams across the allowlist/duplicate/empty boundaries; rejection precedes any execution, pinned by direct unit tests plus actual Make entrypoint tests. |
| `go_test_argv` (PURE) | Compare exact argv with independent leaf-plus-importer expectations at N=1/2; assert no ./... or unrelated package and exactly one Lua runtimebundle addition. |
| `run_selection`, `main` (INTEGRATION) | Stateful child executables retain argv/env/failure state across calls; drive the real make entrypoint and require exact child counts, preserved output/status, one wrapper lifetime and no effects on invalid input. |
| `check_config` in `tests/affected-config-test.py` (INTEGRATION) | Execute upstream strict ParseConfig/Plan against real configuration and controlled ownership/dependency changes; duplicate/unknown mutation refuses, changed package and importer union stays exact, declared and undeclared non-Go ownership stay selective/full respectively. Include both editable nvim source and generated embedded Lua attribution. Validate roots using the actual Pair Go graph. |
| `test_recipe_ownership` in `tests/test-affected-test.py` (INTEGRATION) | Compare current default non-Go Make prerequisites against the disjoint group-target union and each group against declared configuration; missing, duplicate or stale target mutations must fail, preventing silent recipe drift. |
| `test_selected_native_timing` qualification (INTEGRATION) | Before label/landing, run a representative selected command on real Pair code and compare measured execution duration against the branch full suite, reporting queue separately. After landing, hub chooses first real code branch for recorded affected and first docs branch for train vouch as test-economy follow-up. #725 itself forces full. |

## Tasks — one atomic close boundary

- [x] Write the runner/config fixture regressions first. Run them red, including invalid and failure paths. Qualify the strict upstream consumer fixture against a disposable #473 candidate while its M2 is pending.
- [x] Implement declaration, selected runner and shared Make group lists; run focused tests and native selected command smoke. Audit every group claim and default non-Go target; mutate key runner guards to establish regressions.
- [x] Document commands, selection exception for Lua/runtimebundle, fallback/full rules and operator label ownership in README/atlas. Append decisions and evidence to issue Log, preserve unrelated generated workflow stashes.
- [ ] Measure representative native selected-command duration against the full suite before label/landing. Run full suite on #725 pushed head, close through mandatory fresh review, and fix blocking findings. Record hub-owned post-landing affected-status/docs-vouch follow-up in Done when Revision; no early claim of recorded status.
- [ ] Send the concrete tested PR to hub for operator local-gates-verified label. Wait without self-labelling or waiving CI; after label and SHIP use sdlc merge, verify tracker/archive, report receipt-submitted outcome.

## Revisions

- 2026-10-20: Initial plan consumes settled ariadne#473 M1 unchanged. Hub clarification keeps source-Lua/runtimebundle coupling inside Pair's command; generated embed paths remain additive package+group selection. Actual affected status/timing and docs train evidence wait for M2 landing, as explicitly approved by hub.

- 2026-10-20: TL approved full-flow plan (8505a073-10d2-4bf5-8c07-8b6b8d9f5ab6). Strengthened runner count fixture: for a single changed leaf, pass exactly leaf plus selected importers to one go test invocation; explicitly reject ./... or any unselected package. The Lua/runtimebundle exception stays limited to its declared group.

- 2026-10-20: PQ-1: named TestSelection/parse_selection/go_test_argv as pure owners and main/run_selection as the IO shell; replaced the feature-level case inventories with per-function adversarial strategies. Hub confirmation 36f3d275-8e19-4e86-a4b4-10d175c5c020 revises acceptance order because the upstream runner reads affected.json from origin/main: qualify fixtures/native timing before landing, then hub owns first real code affected status and docs-only train vouch in test-economy after #725 lands.

- 2026-10-20: BR-1 identified incomplete non-Go command coverage in the original roots. Governing correction: tests/affected-consumers.json enumerates every grouped recipe's real Pair command or source-reading package roots; empty arrays mean no real Pair command (injected/argv-only/native-tool fixture), with test-make-env's exceptional textual contract checked separately. Actual upstream planner/Pair-graph tests must select the consumer group (or full) for each root and every in-module dependency. Added lua retentioncmd/gccmd/pairlog/reviewcmd; review reviewapply/pairlog; doctor hoprttcmd; build launcher/slugcmd/runtimebundle-generatecmd; shell opener/wrapcmd/probe. Main/dispatcher remains full. Existing paths and group names are unchanged in meaning; coverage is additive.

- 2026-10-20: BR-1 textual-consumer sweep: test-make-env scans cmd/probes Go sources and claimed script/Lua inputs independently of Go imports. The build group therefore also claims cmd/**/*.go, probes/**/*.go and the union of existing non-Go claims. No tests/** or unknown external-fixture claim is added: those remain full. Source Lua now selects build+lua+review. This conservative build addition supersedes the earlier no-group slugcmd timing example; rerun representative timing with the actual updated selection. TestEnvironmentInventoryCoverage enforces the textual input population; recipe ownership requires the command inventory's exact target union.

- 2026-10-20: Proposed correction pending hub approval: broad .go path claims are unsafe because actual upstream Plan classifies an absent/deleted/platform-excluded package as build-only (no Go packages), suppressing the normal unclassified/full fallback. Removed the experimental textual population claims. Keep the audited command-root additions. Instead run fixed make test-make-env once for every valid selection unless build is selected (which already owns that recipe). This preserves Go-only no-private-binary behavior, zero effects on invalid input, one selected Go invocation and one group wrapper. Add a constant O(1) Make guard call, no scheduler/state/schema change. Update recorder/failure/count tests; declaration fixtures must pin absent-package full fallbacks. TestEnvironmentInventoryCoverage is superseded by runner preflight regressions. Native timing must use the final actual selection, including build for slugcmd due the real install-layout alias test.

- 2026-10-20: Hub approved the fixed preflight (29a53677-8658-419c-84d3-1ef1cc04ff30). Implemented once per valid selection except build; removed broad path claims and the experimental planner text-population test. Actual command-root/transitive-dependency inventory and absent-package full fallbacks remain enforced. Source Lua again selects lua+review (and runtimebundle is tested by the runner); no unconditional build-group widening. Measured guard 35.095s; corrected slugcmd selection with build passed in 86.939s on the loaded host. Review round 2 and full suite are next.
