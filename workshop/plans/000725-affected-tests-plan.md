---
collaboration-mode: discussion
---

# Pair affected tests implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use superpowers-executing-plans for the shared configuration/runner work; bounded read-only audits may use subagents.

**Goal:** Run only selected Go packages and declared non-Go test groups for branch evidence, preserving full-suite landing verification.

**Architecture:** Adopt ariadne#473's unchanged version-1 affected declaration and environment contract. Pair's command validates its selection, reuses existing Make targets, and preserves the private test-binary and environment boundaries. The lua group additionally regenerates and tests runtimebundle, because editable Lua is copied into its generated embed tree.

**Tech Stack:** JSON, GNU Make, Python 3 unittest/subprocess, Go, existing Bash/Neovim test targets.

## Core concepts

There are no new pure domain entities: the selection/config parser/Go dependency graph remain owned by ariadne#473. Pair consumes their sorted package/group selection without implementing another dependency planner (ARCH-DRY, ARCH-PURE).

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Affected declaration | `.sdlc/affected.json` | new | upstream strict configuration parser and affected planner |
| Selected test runner | `scripts/test-affected.py` | new | selected `go test` invocation and existing make targets |
| Affected Make entrypoint and group target lists | `Makefile.local` | modified | environment scrub, private test binary, runtime generation and current test recipes |
| Runner regressions | `tests/test-affected-test.py` | new | real entrypoint plus disposable stateful child-command recorder/failure injection |
| Consumer configuration fixtures | `tests/affected-config-test.py` | new | upstream affected/config packages in a disposable Go fixture module |
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

| Owner | Strategy |
|---|---|
| Selected test runner validation/dispatch | Enter actual Make target with stateful go/make recorder: exact argv, groups-only, Go-only, combined, lua augmentation/dedup, invalid inputs before effects, inherited flags, stdout and nonzero propagation; remove validation/dispatch wiring and require regressions fail. Count N=1/2 selections. |
| Affected declaration | Use the upstream strict ParseConfig/Plan in a disposable fixture module (provider directory selectable for prelanding qualification, installed provider after M2). Real Pair config + synthetic changed package/importer graph; scripts-only, full Makefile, unknown file, docs-only, nvim source, embedded Lua and stale group root fixtures. Check actual Go roots with LoadGoGraph after generation. Unknown/duplicate config mutation refuses. No copied planner in production. |
| Test recipe ownership | Guard every current default non-Go target is in exactly one group, every declared group has a runner target and vice versa, each claimed default script is covered; preserve intentionally overlapping source dependency paths. |
| Production smoke/economy | After ariadne#473 M2 lands, run its actual recorded affected check on a pushed real Pair code branch and a docs-only branch; inspect sdlc-local/affected and train-ready evidence. Compare affected execution duration (queue reported separately) against an exact measured full suite, record head/scope/time. #725 itself forces full because Makefiles/.sdlc change. |

## Tasks — one atomic close boundary

- [ ] Write the runner/config fixture regressions first. Run them red, including invalid and failure paths. Qualify the strict upstream consumer fixture against a disposable #473 candidate while its M2 is pending.
- [ ] Implement declaration, selected runner and shared Make group lists; run focused tests and native selected command smoke. Audit every group claim and default non-Go target; mutate key runner guards to establish regressions.
- [ ] Document commands, selection exception for Lua/runtimebundle, fallback/full rules and operator label ownership in README/atlas. Append decisions and evidence to issue Log, preserve unrelated generated workflow stashes.
- [ ] Once #473 M2 is available, measure/record actual affected and docs-only no-run behavior using real branch evidence; do not claim these while unavailable. Run full suite on #725 pushed head, close through mandatory fresh review, and fix blocking findings.
- [ ] Send the concrete tested PR to hub for operator local-gates-verified label. Wait without self-labelling or waiving CI; after label and SHIP use sdlc merge, verify tracker/archive, report receipt-submitted outcome.

## Revisions

- 2026-10-20: Initial plan consumes settled ariadne#473 M1 unchanged. Hub clarification keeps source-Lua/runtimebundle coupling inside Pair's command; generated embed paths remain additive package+group selection. Actual affected status/timing and docs train evidence wait for M2 landing, as explicitly approved by hub.
