# Pair #865: live profile and proposed M2

Read-only analysis, 2026-10-24. No source, live configuration or pair:0 binary changed.

## Identity and window

- Metadata status complete; PID28154; profiler started18:38:12.705, CPU capture18:39:12.795 to18:39:43.053 PDT (60s warmup, nominal30s CPU).
- Metadata and go version -m of /Users/xianxu/workspace/pair/bin/couch both report clean vcs.revision9bfd7ab51cef5f9d563b19dea087c9ef084c19c9, go1.27.3. Git independently resolves that object. Binary SHA256 d019621c0579a85ec48af42eb1e87c3d67fa73c342fc8c3e9e598136f33c7f68.
- CPU profile SHA256 b3647a60d1eb5ed8c1b34ea6f68b0bd26df0842125c564bf4d06f6aed9780ff5; before43eef04de4712280f6d51143563c42089cd76ec65b84c64b8af5f00cb3713c3e; after618d9cdf99f2c5662cdd029cb74016e9bc506ba8c33aaffdcac39611a59ca6af; metadata6c88c647d8ddd73b7ac4d1a6ba9f073faf8a22ce5a5e5ca08cbbf4ff163e92c8.
- Workload supplied by TL: about25 slots attaching after restart. This is startup/reattachment evidence, not an aged steady-state profile.

## Findings

CPU:44.80 sampled CPU seconds /30.22 wall seconds =148.24% (1.48 cores). Cumulative owners overlap and must not be summed:

| Owner | CPU seconds | share |
|---|---:|---:|
| ptychild.Child.pump |20.27|45.25%|
| its ingest path |12.41|27.70%|
| terminal.Endpoint.Feed / vt.Emulator.Write |11.25|25.11%|
| ttyio.File.ReadContext |7.88|17.59%|
| couchmessage.Broker.runActor |6.62|14.78%|
| menu refresh / ActionableThreadInventoryContext |5.18|11.56%|
| title/activity polling |3.07|6.85%|
| terminal.Endpoint.capture |3.05|6.81%|
| background GC mark worker |3.50|7.81%|

Flat syscall.rawsyscalln15.49s (34.58%) divides among PTY reads, filesystem inventories/locks and other calls; it is not one optimization owner. Flat memclrNoHeapPointers5.52s (12.32%) is substantial allocation clearing. Focus webcast or image/png matches no CPU samples in this capture; no recorder-dominance claim is supported.

Sampled allocation profile difference (alloc_space, after minus before) is20.86GiB. Endpoint.capture alone9.62GiB /46.11%; of that9.32GiB flows through capturePublication -> NewEndpoint EnableMode callback -> DEC2026 handling. Other flat allocators include JSON decoder fetch1,733MiB, compactSessionNameIndex1,242MiB, cloneCells1,015MiB and Compose539MiB. Menu inventory cumulative3,078MiB, message authority3,105MiB, title updater2,157MiB overlap downstream helpers; do not sum them. These are sampled cumulative allocations with GC-accounting lag, not exact30-second allocation bytes or RSS. After snapshot sampled inuse125.27MiB, including capture35.54MiB; allocation churn is not retained-memory evidence.

Exact-stage source: terminal/endpoint.go141-145 captures a publication at each new DEC2026 synchronized hold;245-251 allocates complete Cells and Rows slices;264-266 publishes that frame and history. The private frame is read only under the endpoint mutex and callers receive clones (Snapshot303, Publication319; Frame.Clone in frame.go39 copies cell/row storage). This supports a bounded private-buffer reuse correction without changing synchronized-output semantics.

Later live CPU observation: same PID and creation identity Sat Oct24 18:37:49 throughout. At18:42:54 CPU4:12.34, RSS535472KiB; at18:43:40 CPU4:31.11, RSS693168KiB. Delta18.77 CPU seconds /46 wall seconds ≈40.8% of one core. This describes that later live workload only: it is lower than startup, not a later CPU profile, not proof of idle equilibrium or of the cause of the original hours-old cost. No fseventsd causality measurement exists.

## M2 approved before code

First correction: reuse the endpoint-owned frame Cells/Rows backing storage for capture at unchanged geometry. Every consumer retains its existing independent Clone. Resize replaces storage for the new geometry and drops the old private allocation; retain at most one current-geometry buffer per endpoint, no global pool. Overwrite every cell and row, explicitly zero absent backend cells to avoid stale glyph/style/link references. Preserve pre-sync complete-state capture, repeated begin behavior, release, timeout, EOF/final history and snapshot isolation under the existing mutex. Do not throttle parsing or weaken publication freshness.

Growth drivers: endpoints E, screen cells C*R, sync cycles K, actual publication requests P and geometry changes G. Existing capture backing allocation grows E*K*C*R; proposed backing allocation grows with initial/current geometry and G, while required cell-copy work remains O(E*K*C*R), and consumer clones remain O(P*C*R). Count actual backing allocations/bytes for K=N and2N at fixed geometry, then geometry N and2N, with first allocation excluded: zero capture Cells/Rows backing reallocations after warmup; at least90% reduction in capture-owned allocated bytes for repeated fixed-geometry sync capture. Explicit consumer clones/history are separate budgets, not hidden exceptions.

Regression proof: failing allocation/count tests before implementation, plus stale erased-cell/link/style, resize grow/shrink, previously retained immutable frames, same-feed pre-sync writes, repeated begin, release/timeout/EOF and frozen history. Reuse existing Endpoint and Child ingest tests/benchmarks, adding the measured DEC2026 envelope to the workload; no new scale harness or native matrix. Compare candidate/baseline on the identical bounded replay after879 lands and865 is reconciled with main. Bench allocation/counts are the deterministic acceptance, CPU benchmark improvement is corroborating. Any later operator live profile requires separate TL/operator timing; claim a whole-process steady-state budget only after a comparable settled profile exists. Inventory/title/JSON and PTY-read work remain separately attributed; no promise this one fix removes their cost.

Commands: go tool pprof -top /absolute/exact/couch cpu.pprof; repeat -cum; alloc_space with -base=allocs-before.pprof and allocs-after.pprof; focused CPU/allocation reports for capture, rawsyscalln, GC and webcast/image/png; git show the exact revision paths. All were read-only.

Approval: hub a1a107d5-8fe8-4fd2-b1df-06ef4938ce7b accepted the analysis and this buffer/clone/allocation-count scope. Hub f6c298b1-71ef-46aa-86fc-c8355104ba25 permits isolated fresh-main implementation while #879 waits, deferring lessons.md until879 lands and prohibiting a second full check while879's runner is active.
