# Couch launch profiler extraction implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. The source seam is fixed; the main worker owns integration and the SDLC binary owns boundary reviews.

**Goal:** Land the opt-in finite Couch profiler independently so a separately approved operator restart can capture a real profile.

**Architecture:** Selectively transplant the approved #865 diagnostic without its workload or scaled fleet. One pure lifecycle admits effects; one joined worker owns runtime profiling and four private outputs. The existing interactive Console finisher is the only launch adapter.

**Tech Stack:** Go runtime/pprof, context, exclusive local files, existing Couch Console, Go tests and real go tool pprof.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| ProfileRequest / ParseProfileRequest | cmd/internal/couchcmd/profiling.go | new |
| ProfileLifecycle / StepProfile | cmd/internal/couchcmd/profiling_model.go | new |

The request is disabled when empty; otherwise it is a clean absolute destination. The lifecycle is one finite owner: Warmup → Acquiring → Capturing → ReleasingOwned → Finished, with cancellation/error paths through ReleasingUnowned or ReleasingOwned. Only positively acquired CPU profiling authorizes StopCPU. A late event after Finished is inert. The interpreter consumes this model rather than maintaining a parallel policy (ARCH-PURE, ARCH-DRY).

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| ProfileWindow and interpreter | cmd/internal/couchcmd/profiling.go | new | clock, runtime profiler, bounded files |
| runProfiledConsole | same file + run.go finisher | new | existing Console lifetime and exit code |
| Child opt-in removal | singleton.go | modified | existing selected-root environment adapter |

One window owns one worker, one cancellation function, one completion channel and four outputs. Close cancels and joins; it cannot race another cleanup owner to stop profiling. The existing test backend records acquired writer/starts/stops/snapshots and refuses unowned release. The clock controls acquisition and cancellation order. File fault wrappers retain failures hidden by asynchronous runtime writes. A real runtime conformance test parses all three profiles against its exact executable (ARCH-MOCK).

## Fixed extraction boundary

- From a40b597e493d104c76999ad88d1ba3fbc9600d1d: profiling_model.go and profiling_model_test.go only.
- From 88c6b3288e1ac5dadfdf707b36b776e9621c5c13: profiling.go, profiling_test.go; the run.go launch-only finisher hunk; singleton.go child environment hunk and its current-main test adaptation.
- Documentation: README profiler block using the repo's inline edit convention, the original 50-line atlas/couch-profiling.md and atlas/index.md link. Adjust incident references to #871 without copying #865 process artifacts.
- Artifact manifest: only the two profiler NonArtifactSources entries. Audit all changed/new readers against current source and regenerate each named certificate individually.
- Exclude both native acceptance consumers, workload.go, scaled Fleet changes and the scaled atlas suffix. The original native PASS415.469s is peer-provided prior evidence, not a claim of #871 native qualification. Normal-suite runtime conformance is required here.

## Actions and visible states

Actions: launch with a new absolute destination; end the Console normally; inspect the matching binary and files; remove files after investigation; relaunch with a new destination.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Empty opt-in | Normal Console, no profiling | Launch with a destination if profiling is wanted | No capture |
| Invalid path / reserve / create refusal | profile unavailable notice; Console still runs | Relaunch with a new valid destination | No capture; partially created private files may remain |
| Warmup | profile armed, capture after 60s for 30s | Let the finite window finish | Ending Console yields cancelled/partial capture |
| Acquired/capturing | profile capturing | Let the finite window finish | Process death may leave partial files |
| Complete | profile complete; metadata complete | Inspect files with retained executable | Sampled data only, not RSS |
| Busy profiler or IO/cap failure | profile failed/incomplete, cause retained | Relaunch with a new destination after repairing cause | Empty/partial profiles; never overwrite/retry |
| Console ends | Worker cancelled and joined; late errors after terminal restoration | Inspect outcome before using files | Partial capture, normal Console exit preserved |
| Process killed | No successful completion witness | Inspect partial output as incomplete | No joined cleanup on hard process death |

There is no retry, network listener, agent-system message or automation-gate dependency. Notices are local Console output; shutdown diagnostics print only after terminal restoration. No live fleet restart is authorized by #871.

## Operating envelope and ownership

- Off by default and only after interactive start admission. Help, peer messaging and invalid-terminal commands do not start profiling. Descendant actors have COUCH_PROFILE_DIR cleared.
- Operator-approved cadence: one 60s warmup and one 30s CPU interval per opted-in launch, never a repeated loop. No sampling-rate change. Runtime profile overhead is not claimed to be zero.
- One worker and at most one owned process-global CPU profiler. A busy existing profiler remains untouched. Close joins the acquired runtime writer before closing output files; scheduler/IO timing is injected in tests (ARCH-STATE).
- Four files only: cpu.pprof, allocs-before.pprof, allocs-after.pprof capped at 32MiB each, metadata.json capped at 64KiB; new 0700 directory and exclusive 0600 files. No unbounded directory scan or append stream. Every runtime write costs O(bytes presented), with bounded forwarded bytes; counted N/2N writer controls will assert forwarded bytes/call counts and saturation (ARCH-CONSTRAINTS).
- Local explicit destination is trusted to be operator-selected. Do not dump environment, terminal text, DOM or credentials. Build identity is compiled runtime information; retain the exact executable for parsing (ARCH-SECURE).
- The requesting owner is the last reader and removes this fixed bounded set after the investigation. No automatic sweep of user-selected directories and no reuse of existing output. Worker/handles end with the Console; hard death leaves bounded partial artifacts (ARCH-FUNERAL).
- New metadata is write-once diagnostic output, with no production decoder or rewrite path. Existing formats and schemas are unchanged. Do not introduce a lossy read-modify-write adapter (ARCH-EVOLVE).

## Implementation and verification

- [ ] Import the pure/model and adapter tests first; run the focused command to establish missing-profiler RED on main-based #871. Import the selected implementation and reconcile only named current-main launch/env/docs/artifact seams.
- [ ] Keep existing pure transition sequence tests and controlled ownership/fault tests. Add dedicated controls for partial creation failure after an opened output, allocation-after failure, short write and pre-cancelled context; add the counted N/2N bounded-writer case. These fill concrete coverage gaps found during extraction audit, without changing the approved mechanism.
- [ ] Run `go test ./cmd/internal/couchcmd -run '^(TestProfile|TestParseProfileRequest|TestSingletonChildUsesSelectedRoots)' -count=1`. It includes actual CPU/alloc runtime files and three real `go tool pprof -top` reads against os.Executable. Run the model/window/launch cohort with `-race` after deterministic tests pass.
- [ ] Audit launch-only integration on current main, preserve the existing Console finisher and child environment ownership. Validate artifact manifest and changed reader inventory; run make test-affected-inventory and git diff --check. No scaled harness or native conformance file should appear in the final diff.
- [ ] Update issue Log with source provenance, red/green/fault/runtime results, and limits. Commit/push, run `sdlc local-check run --pre-close`, close with the binary-owned review, then promptly enter `sdlc merge --yes` on SHIP. The merge turn owns any later main integration and full suite. No manual chase of main after branch verification.
- [ ] Verify PR, merge ancestry, tracker done/archive, clean retained workspace; send TL the actual pin and invocation `COUCH_PROFILE_DIR="$HOME/new-profile-dir" couch`. The TL/operator separately decides when to restart. Stop after landing.

## Alternatives and exclusions

Selective extraction is the approved choice (ARCH-PURPOSE): it makes real launch profiling available without waiting for scaled synthetic measurement. Landing the entire #865 branch would mix independent harness/optimization work. An HTTP pprof endpoint would add a listener and indefinite lifecycle; external sampling alone would not provide this runtime allocation evidence. Neither is needed here. No CPU-cost fix or complete native workload matrix is claimed.
