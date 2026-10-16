---
id: 000602
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '93c58d271770e2e926397c1c8cf3e33fe9b7824d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T08:07:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b303d084", done: "d0781105"}
actual_hours: 2.01
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

- [x] Normal suite builds one shared plain Pair; all existing real-executable consumers/overrides and standalone routes still work without per-test builds.
- [x] The guard rejects new direct/context/table-driven and shell/Python per-test full-Pair builds; unrelated fixture builds remain allowed.
- [x] Before/after link time, invocation count and disk-write measurements are recorded with flags, code pin and sampling limits.
- [ ] Focused consumer/lifetime/guard checks, recorded full suite, CI and mandatory close review pass; land through the line.

- No test builds `pair-go` itself; the guard enforces it; a full suite's `link` writes drop to the per-package
  test binaries only (measured with ops `tools/disk-writes` before and after).

## Plan

- [x] Capture warm-cache repeated-link baseline with the ops disk-writes sampler and link-time/PID trace; preserve a same-flags after comparison.
- [x] Add the shared builder, suite temporary wrapper and lazy package resolver with lifetime cleanup; migrate every consumer while preserving real CLI/relocation/override behavior.
- [x] Observe the source guard red, then green after migration; verify supplied/fallback/invalid binary, once-only construction, cleanup, private aliases and mutation controls.
- [x] Re-measure one build plus shared placements, run focused consumers and recorded full suite, update atlas, close/review and land. TL owns project records.


## Log


- 2026-10-16: closed — Recorded suite PASS on 31561018 in 11m11s, log d1e3299210e9; CI green. BR-3 full original source control RED 5/11 then GREEN 11/11 with byte-verified bodies and committed portable recipes; relative/context/table and unrelated-dir negatives pass, helper race passes. BR-1 non-root/relative-output and embedded-runtime pass; BR-2 README complete. Controlled 11 links to 1, 20.483s to 1.306s, sampled 465MB to 43MB; limits in Log. Atlas updated; TL owns ops project.; review verdict: SHIP
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

2026-10-16 integration: PR #374 opened. Main advanced with #614 during initial shell phase; deliberately interrupted stale-head suite before Go phase rather than spending two full Go runs. SDLC parent interruption left its owned make group alive; verified exact group 78790 and interrupted it too, verified descendants gone. Integrated main without conflicts and stamped merge commit 52ce2411717755707753d856bd7defc11a7024fd. Recorded suite restarted on this combined head. This is a deliberate supersession, not a failing test result.

2026-10-16 suite finding: TestProductionArtifactReferencesAreExactlyClassified failed because any non-_test.go source must appear in the exhaustive artifact inventory, including test-support libraries. Reproduced alone RED (1.928 s), classified the compiler fixture alongside testwait as NonArtifactSources (no session-artifact ownership), then GREEN (1.927 s). Did not add to the quarantine. Continuing the full run to collect all remaining failures before the required fixed-head rerun.

2026-10-16 complete diagnostic suite: 52ce2411 finished in 12m26s; only TestProductionArtifactReferencesAreExactlyClassified failed. All other packages passed, including couchcore 373.988 s, testpair 3.083 s and wrapcmd 81.905 s; all shell prerequisites passed. Failure log sha256 prefix 46f7266a14cf at sdlc/test-stats/failed/ca43791148c8cb561c8204a21b4765d3.log. The classifier fix is committed at c5e6d1742fe21eb0e605addd71e87ad49a3d1c83, whole artifactpath package passed in 6.311 s. Fixed-head recorded suite started.

2026-10-16 verification complete: fixed-head recorded suite PASS on c5e6d1742fe21eb0e605addd71e87ad49a3d1c83 in 13m28s (6 s slot queue), log sha256 prefix d9d0a5955149 at sdlc/local-checks/c5e6d1742fe21eb0e605addd71e87ad49a3d1c83/suite.log. No failing tests. couchcore 351.124 s, testpair 1.989 s, artifactpath 7.075 s, wrapcmd 75.647 s. Server conformance, merge-check and both local-gates verdicts PASS. All implementation/verification steps completed; entering mandatory close review and landing. TL retains project bookkeeping.

2026-10-16: First close returned REWORK: BR-1 Critical cwd independence, BR-2 Important missing README. New fake-compiler shell controls reproduce BR-1 RED before central builder correction. Fixed source-root resolution in the sole builder rather than special-casing the consumer; retained caller-relative output paths. README now covers the complete lifecycle. Review sidecars retained for the next boundary pass.

2026-10-16 review fix verification: new non-root and relative-output fake-compiler controls RED before fix, GREEN after. testpair -race passed in 7.512 s. The real embedded-runtime script from tests/ built successfully but inherited the live Couch root; the standalone rerun clears COUCH_PAIR_DATA_DIR, matching the normal Make environment scrub. No production changes or deadline changes.

2026-10-16 R1 final verification: real standalone embedded-runtime script from tests/ PASS after clearing inherited COUCH_PAIR_DATA_DIR. Final recorded suite PASS on 52b3ed54b73720a9aa3aa48f7b508bfbd216c1d9 in 12m05s; log sha256 prefix 1a4df793016e at sdlc/local-checks/52b3ed54b73720a9aa3aa48f7b508bfbd216c1d9/suite.log. No failing tests; couchcore 367.806 s, testpair 2.927 s, wrapcmd 81.490 s. Environment scrub passes 225 derived knobs. Server checks green. BR-1 and BR-2 fixes ready for the next fresh boundary pass.

2026-10-16 close round 2: prior BR-1 and BR-2 disposed. New BR-3 Important: source guard recognizes only five of the eleven complete original Go source bodies; relative package targets used by six consumers are missed. Fix the family by normalizing relative targets and pin all eleven original recipes as regression controls, retaining unrelated-build negatives. No consumer/runtime change required.

2026-10-16 BR-3 qualification: temporary overlay against all eleven complete original Go files at 53c96ded observed RED 5/11 (exact six relative-target misses), then GREEN 11/11 after package-cwd normalization and bounded explicit-command-Dir facts. Portable committed fixtures cover all eleven command recipes, direct/context/table relatives, absolute/module targets, and unrelated relative/explicit-Dir negatives. Targeted guard tests pass; testpair -race PASS 9.553 s. Consumer and runtime code unchanged.

2026-10-16 root independently re-ran the full-source overlay qualification: all eleven subtests PASS (0.108 s), verified each preserved body byte-for-byte against git show 53c96ded and that overlay implementation starts with the current committed guard. Bundle SHA256 46db6b15d260e5e450668cdd7aef3e0297d91e61d77eeef9b6e63334a5ccb969. Local overlay /tmp/pair602-guard-_cqj8nju/overlay.json; portable eleven-recipe regression fixtures remain committed.

2026-10-16 R2 complete verification: recorded suite PASS on 31561018d1dad71c536dbcad0a3804b64147e55e in 11m11s; log sha256 prefix d1e3299210e9 at sdlc/local-checks/31561018d1dad71c536dbcad0a3804b64147e55e/suite.log. No failing tests; couchcore 333.332 s, wrapcmd 75.986 s. Server conformance and both deterministic checks green. All eleven original source bodies detected, portable recipe fixtures and relative/unrelated negative controls pass. Ready for close round 3.

## Revisions

2026-10-16 R1 — close review BR-1 reproduced the shell builder resolving source relative to caller cwd; centralize checkout selection while resolving relative output against the original cwd, with subdirectory and outside-checkout fake-compiler regression controls plus the real embedded-runtime script from tests/. BR-2 requires README coverage of the complete shared/supplied/fallback lifecycle and plain-vs-development defaults. Scope and original assertions unchanged; re-run targeted checks and the recorded full suite before re-close.

2026-10-16 R2 — guard acceptance must include all eleven original Go recipes, including ../../pair-go, sibling ../pair-go and normalized relative forms. Add compact portable recipe fixtures and independently qualify the full original source bodies; preserve unrelated relative build negative controls. Re-run targeted guard/race checks and recorded suite before close.
