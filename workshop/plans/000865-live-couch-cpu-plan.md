# Attribute and bound live Couch CPU implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) and use superpowers-executing-plans for the session-warm implementation. #860 has landed; this issue starts from refreshed main. Commit design before change-code; M2 needs TL approval after measurement.

**Goal:** Attribute the sustained ~1.3-core live Couch workload with matching-binary CPU/allocation profiles, then remove and mechanically bound the measured redundant work.

**Architecture:** First add one explicitly enabled, finite profiling window to the existing Couch launch lifetime. Save local CPU and allocation profiles with build/workload metadata; expose no network profiling service. Select the optimization and CPU/work budget only after the profile identifies its owner and growth driver.

**Tech Stack:** Go runtime/pprof, existing Couch runtime/Console lifetime and diagnostic log, private local files, existing disposable Couch/native-agent fixtures.

## Evidence and limits

See [initial measurements](../evidence/000865/initial-measurements.md). Raw local samples and profiles are retained outside Git; they are not publication assets. PID4409 used135.69% CPU over25s while five frames were stored. The current cast is a local layout mock, not the earlier echo board. Image functions account for201 of5957 counted symbolized Couch leaf samples (~3.37%); GC/memory, terminal and JSON categories are larger. This is not a CPU-profile attribution. Recording growth was ~9.83KB/s PNG plus506B/s events, against ~180KB/s whole-Couch physical writes. fseventsd attribution is unknown; its native counters/fs_usage require privilege.

The mapped process UUID differs from the current binary at its path. Do not assign that running process the newer file's revision. A profile must retain the exact executable it ran and its runtime build identity.

## Alternatives

1. **Recommended: bounded launch-only local profiles.** One env opt-in, one finite window, no listener, negligible disabled cost; obtaining the first live profile requires a coordinated relaunch.
2. **Local HTTP pprof server.** Supports repeated windows, but adds a long-lived control surface, address/auth/lifecycle and concurrency policy absent today. Not needed for first attribution.
3. **Optimize screenshot handling from sample alone.** Insufficient: the measured image leaf share is small and the actual cast changed. It does not establish the dominant redundant work.

## Core concepts

| Name | Kind | Lives in | Status |
|------|------|----------|--------|
| ProfileLifecycle/StepProfile | PURE | cmd/internal/couchcmd/profiling_model.go | new; sole legal state/event/effect reducer |
| ProfileRequest | PURE | cmd/internal/couchcmd/profiling.go | new; parse an absolute output directory or disabled |
| ProfileWindow | INTEGRATION | cmd/internal/couchcmd/profiling.go | new; own timer, runtime/pprof and local output files |
| Couch launch lifetime | INTEGRATION | cmd/internal/couchcmd/run.go | modified; opt-in only on cliLaunch, join profile cleanup on exit |
| Stateful profile backend double | INTEGRATION test | cmd/internal/couchcmd/profiling_test.go | new; enforce one active CPU profile, completion/failure/cancel ownership |
| Controlled workload | INTEGRATION test | existing cmd/internal/testcouch and native agentfixture seams | reuse; exact measured hot-path test chosen after attribution |

There is no new business entity or persisted product format. Artifacts are local diagnostic outputs, never auto-published. Use existing runtime/Getenv and diagnostic ownership rather than another service supervisor (ARCH-DRY).

## M1 — Obtain authoritative profiles

Planned files: new profiling.go/profiling_test.go; existing run.go launch branch; atlas diagnostic entry and atlas/index.md; launch usage/README only if an operator-facing invocation needs documentation. Review exact integration position after start-plan on fresh main.

- [ ] Write RED controls: disabled and non-launch commands create nothing; opt-in reserves a new private output directory; invalid/relative/existing output refuses without clobber; one CPU profile owner; cancellation before/during capture stops and joins; file/start/write/stop failures are observable.
- [ ] Implement launch-only COUCH_PROFILE_DIR opt-in. Fixed60s warmup, fixed30s CPU window; allocation snapshots immediately before/after, runtime build/workload metadata. Use 0700 directory/0600 files; never overwrite existing profiles. Do not change sampling rates, add a listener, or alter normal recording/viewer behavior. Profile outcome affects diagnostic evidence, not ongoing Couch work after launch.
- [ ] Keep lifetime lexically owned: warmup/capture ends on launch cancellation; StopCPUProfile and file close run once; no worker survives return. Fake clock/backend make the lifecycle tests deterministic. At least one short real profiling test must produce profiles accepted by go tool pprof.
- [ ] Run focused lifecycle tests, package race, lint/inventory and the required checks before the boundary. Add a native diagnostic control using existing disposable Couch fixture infrastructure, with Claude/Codex personalities where agent input is involved.
- [ ] TL approval d515e211-98ad-4923-8896-0b50468764e0: profile a disposable Couch under a reproduced workload first, retaining its exact executable with runtime build identity, profile settings, actor/input rates and cast/viewer state. Reuse the existing testcouch/native-agent infrastructure. Collect bounded controlled windows (cast/viewer on/off and quiet/busy terminal input). If the disposable run cannot reproduce the sustained cost, report that evidence to TL; only the operator may approve relaunching the live Couch, and TL will ask them then. Do not stage/relaunch the live Couch autonomously.
- [ ] Analyze go tool pprof top/cumulative and allocation diffs against the matching binary. Measure screenshot stages separately only if the profile justifies them. Report the dominant owner, cost per input unit, and evidence limits to TL.
- [ ] Close M1 through the binary's mandatory fresh review. Then append the measured M2 revision and obtain TL approval before optimization code.

## M2 — Bound the measured work (design gate after M1)

Do not preselect LowRes, GC tuning, fewer screenshots or an arbitrary total CPU target. M1 must identify whether cost grows with input bytes, panes, DOM events/nodes, screenshots/pixels or operator deliveries. Before M2 code, amend this plan with exact owner/files, the redundant transition, a stated CPU budget for a fixed workload and a counted N/2N work bound.

- [ ] Write a RED deterministic counted regression reproducing the identified redundant work through the existing stateful seam; preserve ordering, privacy, observation and cleanup behavior.
- [ ] Implement the smallest owner-level correction, with no blanket suppression, sleeps or unmeasured rate cap. Name operations per growth driver; reject superlinear amplification at N and2N.
- [ ] Repeat the exact workload/profile with profiling disabled for the final external CPU measurement; compare throughput, CPU and allocation rates against the pinned baseline. Meet the agreed budget and preserve native Claude/Codex behavior. Compare capture/viewer modes separately.
- [ ] Answer the fseventsd question only with a permitted controlled change or bounded privileged trace. Otherwise report the measured recording/whole-process write rates and the attribution limit explicitly; no causal claim.
- [ ] Full verification, stage/live acceptance as TL directs, final review and landing. Retain diagnostic artifacts only for investigation, removed explicitly by their operator owner afterward.

## Actions and visible diagnostic states

Actions: enable for a new directory and launch; inspect profiles/log; retry with a new directory/relaunch; remove retained diagnostics.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| no opt-in | normal launch | enable on next launch | no profile collected |
| waiting60s | armed, output directory named | inspect/wait or cancel launch | no CPU profile yet |
| capturing30s | profiling | wait or cancel launch | partial profile if process is killed |
| completed | completed paths | inspect, then remove diagnostics | none beyond deliberate removal |
| path/start/write error | explicit failed diagnostic | retry new directory/relaunch | incomplete profile; Couch work preserved after launch |
| launch ends during warmup/capture | cancelled/partial paths named | retry new directory/relaunch | incomplete measurement only |

## Constraints, ordering, residue and evolution

Disabled cost is one launch-only environment read. Enabled cost has one worker, one30s window after60s delay and a fixed small artifact set; no ongoing listener or per-event profiling code. Runtime CPU profiler ownership and window lifetime are one owner/resource, not check-then-act observations (ARCH-ORDER). Binary artifacts are capped at32MiB each and metadata at64KiB, with a cap hit reported as failed/incomplete; no unbounded queue.

CPU/capacity acceptance is defined on the measured workload before M2 (ARCH-CONSTRAINTS), with counted scaling rather than timing alone. Artifacts are operator-requested, private, finite, retained only until investigation completes and explicitly removed; no periodic growth (ARCH-FUNERAL). Existing product wire formats are unchanged (ARCH-EVOLVE). No credentials, environment dumps, captured browser content or public profile upload is added (ARCH-SECURE).

## Approval

2026-10-24 TL d515e211-98ad-4923-8896-0b50468764e0 approved M1 as written with disposable reproduction first. Operator live relaunch requires their approval, requested by TL only if disposable cost does not reproduce. M2 returns to TL with measured owner, workload and budget before optimization code. #860 merge must settle before implementation.

## M1 integration detail to carry onto the fresh issue branch

Read-only source inspection locates the launch call at RunWithRuntime's cliLaunch branch and the Console lifetime in runTypedOperationStyle's consoleFinisher. Keep Runtime unchanged: use its existing Getenv only on the forceConsole/start launch path. The finisher can arm the profile immediately before runConsole and defer cancellation/join after it returns; every early failure before a Console lifetime then creates no diagnostic worker. Report through Console.Notify, never raw stderr while the Console owns the terminal. Disabled and non-launch calls must not create files or start timers. Revalidate these integration locations against fresh main before admitting code.

The implementation should reserve the requested leaf directory with one os.Mkdir(0700) under an existing parent; no recursive parent creation and no reuse of an existing leaf. Open its fixed artifact names with O_CREATE|O_EXCL and0600. A bounded first-error writer is required for CPU output: the current Go runtime profileWriter discards the error returned by profileBuilder.build, and StopCPUProfile has no error return. Inspect the writer's retained error after a successful StartCPUProfile followed by StopCPUProfile; never call Stop when Start refused, because another owner may be active. Check profile write and Close errors too. A per-artifact hard cap (32MiB for each binary profile and64KiB for metadata) bounds disk growth; a cap hit is an explicit failed/incomplete diagnostic, never success. The native regular-file adapter is not a promise to interrupt a blocked kernel filesystem operation. Do not add an asynchronous unjoined writer to manufacture a timeout.

Profile metadata should contain runtime/debug.ReadBuildInfo, executable identity/hash captured for the selected candidate, process/start/profile timestamps, fixed settings and final status. Describe workload, actors, rates, cast state and viewer state in the controlled run's external manifest using existing harness observations, instead of adding another production inventory reader. No environment dump, browser DOM or authored terminal text belongs in metadata. Record cumulative allocation snapshots at their actual timestamps without forcing GC or changing MemProfileRate; analyze the difference over the observed window and state sampling limits.

Named adversarial strategies (the executable cases live in tests):

- ParseProfileRequest: malformed path classes → pure accepted-request/disabled/refused invariants, without IO.
- StepProfile: generated legal and adversarial event sequences → independently count acquisition/release, require no release without ownership, retain first failure and permit completion only after cleanup.
- StartProfileWindow/ProfileWindow.Close: controlled clock/backend/file failure ordering → same production reducer, one owner, exactly-once cleanup and joined completion; bounded first-error writers preserve failures hidden by the runtime API.
- RunWithRuntime launch integration: vary command/lifetime admission → only an admitted interactive launch may read the opt-in/create artifacts, and ending Console joins it without altering the exit code.
- Runtime profile adapter: short real capture → go tool pprof accepts CPU/allocation outputs with the exact retained binary; stateful double ownership agrees with this conformance control.

The existing Console control is now recorded in the committed measurement summary. Its allocation concentration justifies including terminal-heavy disposable controls; it does not select the M2 optimization.

## Fresh-main reconciliation and verification commands

#863 is landed at109986901f7410e0120135121f9e906054603634; native Fleet.Start honors genuinely empty history and keeps automation off. Reuse its public Hubs/StartCold/QuitConsole API as appropriate. Existing nonempty/unknown namespaces stay parked. #866 startup panic and #867 Codex idle disagreement remain explicit run limits; a failure with retained roots must be reconciled by its existing owner before another fleet starts.

Targeted lifecycle: `go test ./cmd/internal/couchcmd -run 'Profile' -count=1`, then `go test -race ./cmd/internal/couchcmd -run 'Profile' -count=20`. Run the package's relevant launch controls, `go test ./internal/repolint`, the audited reader inventory (`make test-affected-inventory`) and `git diff --check`. Run short real backend conformance with generated profiles parsed by `go tool pprof -top`. Native workload commands are added to Log with their actual selected candidate, manifest and timeout before running; do not claim skipped cases as qualified. Boundary verification is `sdlc milestone-close --issue865 --milestone M1`, with the mandatory binary-owned review; final pre-close uses the pushed head's `sdlc local-check run --pre-close`. M2 exact test commands follow its approved measured revision.

ARCH-PURE: path validation is pure; runtime profiler/file/timer ownership stays in one thin integration. ARCH-MOCK: the stateful profile backend models exclusive acquisition, output, failure and release; a real pprof conformance control checks it. ARCH-PURPOSE: M1 is not final completion; the issue includes the measured M2 correction. ARCH-SECURE: private local outputs only, no listener, credentials, DOM or text dump. ARCH-CONSTRAINTS and ARCH-FUNERAL: one finite window, fixed bounded file set, no periodic growth; explicitly opted-in artifacts retained for the investigation and removed by their requesting owner. ARCH-EVOLVE: no product wire format changes; diagnostic metadata is write-once, with no read-modify-write path to lose unknown fields.

## Revisions

- 2026-10-24 — PQ-1/PQ-2: enforce lifecycle through StepProfile and compress test inventories into adversarial strategies. Public behavior, profile settings, owner and disposable-first approval remain unchanged.

## Enforced lifecycle core (PQ-1)

`ProfileLifecycle` is PURE; `StepProfile(state,event)` is the sole state/effect reducer. `ProfileWindow.run` is the one serialized event/effect interpreter. It holds the authoritative state as a local value and replaces it only with StepProfile's result; file/backend adapters never mutate it. Concurrent Close calls only cancel the context and await the same done channel. They cannot call backend Stop or close files. External acquisition/IO runs synchronously inside this worker; no unjoined effect worker exists.

| State | Event | Next state / emitted effect |
|---|---|---|
| warmup | elapsed | acquiring / allocation-before then CPU Start |
| warmup | cancel | releasing / close files and finalize cancelled evidence, no CPU Stop |
| acquiring | admitted | capturing / arm finite capture timer |
| acquiring | admitted-cancelled | releasing / Stop owned CPU, finalize cancelled evidence |
| acquiring | failed | releasing / finalize failed evidence without CPU Stop |
| capturing | expiry or cancel | releasing / Stop owned CPU, allocation-after, close/finalize |
| releasing | release-completed (with any retained error) | finished / close done after effects joined |
| finished | late event | finished / no effect |

Cancellation during synchronous acquisition is latched by context, not by concurrently mutating lifecycle state: after allocation-before the worker checks cancellation before calling CPU Start. After successful CPU Start it checks cancellation again and emits admitted-cancelled if needed. That event carries positive acquired ownership so release cannot be skipped. Failure of CPU Start never grants ownership. The pure reducer rejects out-of-state events instead of guessing success. Cancel/expiry racing while capturing are serialized by the worker select: the first emits release, later cancellation only waits for done. A release failure is retained as failed/incomplete; it cannot roll back completed effects or create permission to retry acquisition. No retries exist inside a launch.

The model carries an optional first failure and explicit acquired ownership in releasing state. Cancellation is a visible cancelled outcome unless an IO/backend failure supplies stronger failed/incomplete evidence; the first actual failure stays authoritative. Release does not run twice. Cleanup completes despite cancellation, and Close returns only once the worker and runtime writer are joined. A failed file-close or final-metadata write remains observable through the existing notice owner even if the final file is incomplete.

Applicable events: cancel revokes the pending/active window and releases acquired ownership; expiry completes the window through release; IO/acquisition failure cancels further profiling and preserves completed artifacts as incomplete; a late cancel/expiry after finished is ignored. There is no replacement request, automatic retry, queue, preemption or rollback in one launch. Only timer, context and synchronous IO completion ordering are nondeterministic; the controllable backend/clock drives those same boundaries in tests.

- 2026-10-24 — implementation organization: pure lifecycle lives in dedicated profiling_model.go with colocated tests, leaving IO in profiling.go. Corrected the repolint command to its existing internal/repolint package.
