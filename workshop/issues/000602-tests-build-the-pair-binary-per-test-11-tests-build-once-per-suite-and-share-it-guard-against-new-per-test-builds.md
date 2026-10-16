---
id: 000602
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '31004dfc8b493861cfd55ca749fd26ea09ac50cb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T08:07:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b303d084", done: "d0781105"}
---

# Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds

## Problem

Go acceptance helpers and shell tests repeatedly build the complete Pair binary into private directories. Eleven Go files hide additional builds through loops/shared helpers; four shell scripts add more. Repeated linking writes large executables although the source/flags are identical. Native CI separately builds one fresh candidate.

11 tests in pair run a full `go build` of `pair-go` into their own temp directory on every run, to exercise the
real binary as a subprocess (end-to-end fidelity). Each one links a fresh ~20 MB binary plus intermediates, per test,
per suite, per worktree; on 10-15 the suite's `link` processes wrote ~8 MB/s and `couchcmd.test` ~30 MB/s while
#571's suite ran (ops `tools/disk-writes`). Files: cmd/couch/main_test.go cmd/internal/artifactpath/cross_scope_integration_test.go cmd/internal/couchcmd/continuation_acceptance_test.go cmd/internal/couchcmd/switchagent_acceptance_test.go cmd/internal/couchcore/startup_detach_timing_live_test.go cmd/internal/hoprttcmd/hoprtt_test.go cmd/internal/launcher/agent_command_acceptance_test.go cmd/internal/launcher/couch_launch_prompt_acceptance_test.go cmd/internal/launcher/selected_root_test.go cmd/internal/layoutcmd/fullscreen_chord_live_test.go cmd/pair-go/helper_equivalence_test.go 

Part of ops project test-economy (north star: one MacBook, 10 workers; write and time budgets).

## Spec

Build one fresh plain Pair per normal suite in a scoped temporary wrapper and export its absolute PAIR_TEST_BINARY through shell prerequisites and Go tests. Keep ordinary development bin/pair separate because its defaultPairHome linker setting changes embedded fallback. Centralize construction/placement in tests/lib; cmd/internal/testpair resolves a supplied executable or lazily builds once per Go package for standalone targeted runs, cleaning it at TestMain completion. Preserve existing helper dispatch order and explicit acceptance/live overrides. Private executable-location fixtures use hard links (copy across filesystems), not symlinks that can change os.Executable behavior. Standalone shell tests retain owned fallback builds. The separate native lane retains one fresh candidate through the same builder. No production behavior, assertion, fixture deadline or skip changes.

Add a source guard against Go/table/context and shell/Python full-Pair builds outside the centralized builder; leave unrelated fixture/probe builds intact. Verify it red against current offenders and with semantic mutation controls. Measure a controlled warm-cache eleven-build workload versus one-build/shared-placement workload using the same flags/toolchain, link elapsed/PIDs and ops tools/disk-writes counters. Treat device totals as shared-host observations and sampled short-lived logical writes as lower-bound observations, not a causal full-suite speedup. ARCH-DRY: one builder; ARCH-MOCK: retain real executable integration; ARCH-ORDER/FUNERAL: suite/package lifetime exceeds each consumer and owns cleanup; ARCH-CONSTRAINTS: retain caller build limits.

- The suite builds `pair` once (`make test` already builds `bin/`), and passes its path to tests in one env
  variable (e.g. `PAIR_TEST_BINARY`). A test needing the real program uses it; if unset (a targeted `go test`), a
  single shared build per test process (sync.Once into one temp dir), never one per test.
- Where a test only needs *a* subprocess, use the test-binary helper-process pattern (as the continuation test's
  `couch` shim already does), with no build at all.
- A guard test (or lint) refuses new `exec.Command("go", "build"` in `_test.go` outside the shared helper.

## Done when

- [ ] Normal suite builds one shared plain Pair; all existing real-executable consumers/overrides and standalone routes still work without per-test builds.
- [ ] The guard rejects new direct/context/table-driven and shell/Python per-test full-Pair builds; unrelated fixture builds remain allowed.
- [ ] Before/after link time, invocation count and disk-write measurements are recorded with flags, code pin and sampling limits.
- [ ] Focused consumer/lifetime/guard checks, recorded full suite, CI and mandatory close review pass; land through the line.

- No test builds `pair-go` itself; the guard enforces it; a full suite's `link` writes drop to the per-package
  test binaries only (measured with ops `tools/disk-writes` before and after).

## Plan

- [x] Capture warm-cache repeated-link baseline with the ops disk-writes sampler and link-time/PID trace; preserve a same-flags after comparison.
- [x] Add the shared builder, suite temporary wrapper and lazy package resolver with lifetime cleanup; migrate every consumer while preserving real CLI/relocation/override behavior.
- [x] Observe the source guard red, then green after migration; verify supplied/fallback/invalid binary, once-only construction, cleanup, private aliases and mutation controls.
- [ ] Re-measure one build plus shared placements, run focused consumers and recorded full suite, update atlas, close/review and land. TL owns project records.


## Log

2026-10-16: Dispatched by verified TL receiptbd4bbb53-5298-4048-87ce-4a1452a49984; claim/start-plan completed from fresh main. Small plan may proceed after posting. Read-only independent audit found ordinary build sites plus repeated helper calls and four shell scripts. Native-terminal CI is a separate once-per-lane fresh candidate; centralize its builder without weakening freshness. ops/MEMORY.md is absent; ops charter and tools/disk-writes/SKILL.md read. No implementation yet. Scope fits the quick-flow shell by keeping new non-test code in a small standard-library resolver and reusing test-owned shell lifecycle; the gate owns final sizing/review.

### 2026-10-15

2026-10-16 checkpoint: Baseline at 9edd1f06cf8fdad99da25a8b925cc9857d6c056e, warm cache, GOFLAGS=-p=3 GOMAXPROCS=3, go build -trimpath with identical link tracing: 11 builds took 31.147 s; real linker wall sum 20.483 s; sampled exact linker PIDs wrote 465,285,120 logical bytes. Ops sampler covered 30 s (1.147 s of workload tail outside sample), so bytes are explicitly a lower bound. Shared-host device total 2,938,138,624 bytes is not attributable to this workload. Raw report and trace: local scratch pair602.omXdPs/before-report.json and before-links.jsonl. Consumer migrations and source guard are uncommitted partial implementation; shared helper/wrapper are still pending and compilation is not yet verified. Guard synthetic RED observed across forbidden controls; migrated repository guard passes. Context-aware APIs preserve original 2m/3m caller limits.

2026-10-16 implementation: shared builder, suite wrapper/recursive test target, package resolver and all consumers implemented. Lifetime contract observed RED (missing APIs), then GREEN; shell lifecycle observed RED (missing builder), then GREEN. Guard controls reject direct/context/table/variable/shell/Python builds while preserving unrelated binaries. All eight migrated packages compile. testpair -race passed (8.626 s); make-test-env passed with 224 knobs including the new env. Original opt-in fixture contexts and dispatch order preserved.

Controlled comparison (same production code, warm cache, GOFLAGS=-p=3, GOMAXPROCS=3, -trimpath and identical toolexec): eleven fresh outputs 31.147 s / 20.483 s linker sum / 465,285,120 sampled linker bytes; one output plus ten hard links 2.108 s / 1.306 s linker sum / 43,061,248 sampled linker bytes. Link count 11→1; linker time -93.6%; observed writes -90.7%. Baseline head 9edd1f06cf8fdad99da25a8b925cc9857d6c056e; comparison head e75fd5795d58cbddf8860c739fa78cee96778459 (checkpoint-only difference; in-progress changes are test infrastructure). Before sampler misses final 1.147 s; both samples are lower bounds from short-lived process observation. Device totals over 30 s: before 2,938,138,624, after 241,217,536 bytes, shared-host observations only. No claimed full-suite wall-time reduction.
Raw before-report.json SHA256 618a1c0444b89bef9447219cdc0bdafe4970e0236d30f3ef60a03008d69eab5d.
Raw before-links.jsonl SHA256 389bfed5a46921d9a67dcf4e755d7ae4a8654a76d7fd0948bcaffdeea59af28a.
Raw after-report.json SHA256 878109a5a2b626575e280f5fa1657b3f5975b43b0c63297772790038ffea4fac.
Raw after-links.jsonl SHA256 7a87a46f26d860e1021fdc184a8a71066782d3d3335735693c9a44ba3dc31685.

2026-10-16 focused verification: shared supplied binary passed real executable consumers in couch (24.067 s), artifactpath (0.700 s), couchcmd (20.059 s), hoprttcmd (0.484 s), launcher (3.769 s), pair-go (0.623 s). Standalone hoprtt fallback passed (3.224 s). All four migrated shell suites passed, including embedded-runtime relocation and submission editor integration. Guard expanded to the resolver implementation itself; existing production builders remain out of scope. Reused internal/reporoot for trimpath-safe validated module discovery (ARCH-DRY). Awaiting recorded full suite and close review; no opt-in live sessions driven.
