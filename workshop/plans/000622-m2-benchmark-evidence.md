# #622 M2 — benchmark qualification and stop decision

TL approval4f9a9b31 admitted the two narrow eager shortcuts, with an explicit stop/re-plan if benefit did not repeat. That condition is met: no production acceptance or landing is claimed.

## Candidate and semantic evidence

Canonical branch recreated by start-plan at fresh main per TL6d5eb120; ad-hoc plan branch was removed after patch-equivalence proof. Plan-quality CLEAN; cumulative estimate-quality INFO (verification budget optimistic), implementation admitted. Qualification baseline a9cf0d192590ab9195aecded9ba3f043dca8154c; candidate b721874da1b87dbc731b7b6fc6bb2da4e8bf1e4e. Production diff is12 added lines and one changed condition in utf8.go and frame.go. No extra authority/state/worker/buffer.

VT full module tests passed after the shortcut (3.538s VT,0.195s ansiparser). Generated differential comparisons against frozen slow handlePrint cover both screen cells/metadata/cursors, history, modes, keyboard state, provisional cluster/old cells/drop state, character sets, reply bytes and typed effects across chunk partitions and resize.95×95 ASCII pair boundary/width checks use the existing Unicode library. Frame tests compare an independent slow validator across all256 byte values, widths, links and generated Unicode/neighbor forms, with explicit golden outcomes.

One VT test-only mutant initially did not distinguish its purported broad-boundary error: a standalone zero-width Prepend rune is already discarded by orphan handling. Retained commit21be6d70e contains that failed qualification; a9cf0d192 replaces it with a real keycap-continuation mutant. No production code existed before the now-passing qualification. Targeted frame tests passed after the shortcut. Full Pair suite, races, boundary close and live acceptance have not been run.

## Complete-ingest experiment

Raw directory: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-m2-bench.AMTYLq

Frozen detached baseline source and separate precompiled baseline/candidate test binaries. Sequential ABBA order; GOMAXPROCS=3, GOFLAGS=-p=3; benchmark300ms,count3. No source changes or own heavy suites during the balanced runs. Other fleet work remained uncontrolled. The redraw workload uses actual Child.ingest, parser, diagnostic ring, publication flush, Endpoint.Publication and Frame.Validate; ASCII/mixed80×24 redraws vary64/4096-byte transport chunks. IngestBytes controls exercise existing1/64/4096-byte workloads. This is wall time and allocations; no process CPU attribution was captured.

Median ns/op; all repetitions retained:

| Workload | A1 baseline | B1 candidate | B2 candidate | A2 baseline |
|---|---:|---:|---:|---:|
| ingest1 |2338|2426|2507|2518|
| ingest64 |16950|15474|200242|17377|
| ingest4096 |929828|822513|15012679|959653|
| ASCII redraw/chunk64 |421569|413349|3839065|428621|
| ASCII redraw/chunk4096 |452624|475898|4992697|324481|
| mixed redraw/chunk64 |524248|590257|6643515|524146|
| mixed redraw/chunk4096 |388513|356294|6592703|439889|

B1 ingest4096 improves11.5% relative to A1, but ASCII redraw/chunk64 improves only1.9%, chunk4096 regresses5.1%, and mixed/chunk64 regresses12.6%. B2 deteriorates broadly; its host load rose16.41→35.19 during the run. A2 recovers near A1 for several workloads. This is consistent with material scheduling/co-tenant interference, but that is not proven causal attribution and does not exonerate the candidate. Even excluding B2 would not establish a consistent complete-redraw benefit. Do not delete B2, average it away or select an isolated passing ratio.

Binary hashes: baseline60f3e1d41e7de53ee286ce8be4c5bff70385c629734df30b2027ba44881ee420; candidatebe8a66f6c3a1d92ec9b965c4988150575b325a013543d1c0b426efe169e43277.
Log SHA256:
- A1 e81d8baf169a5e878e1e1e841fffddafdce3623a789b157f3b07717440f0c69f
- B1 d5cbdbda74904205a700a02651c1e466f044ec37a5a00d9ded21b7256285c522
- B2 39d5807bb95bd7ba20b242707616b2f203c34495c41a137df99193802e2ded42
- A2 1829f831989d5ac2eb1af293c216ef24ed00e89c005dd3ed3f4db83f7bdb2794
- metadata bc18b5f19fa89cbaf7c54ace76b0507b34a74f63acb4fe4e9baf8f4f6a384d93
- parsed repetitions ef3f089b95acefd52e500a67e4d59b465a8acbc374be388595bb2d236a91e0d9

Pilot baseline logs are separate and excluded: one checkout changed during branch recovery; another frozen baseline overlapped short correctness runs. Frame-only baseline timings are retained but are not end-to-end evidence. No process was signalled; all owned runners completed.

## Proposed measurement revision — requires TL approval

Hold this candidate; no more mechanisms or landing. The simplest alternative to changing terminal design is to repair the performance experiment: fixed identical work counts with process CPU measured outside the hot path, alongside wall time. Native Go benchmark ns/op cannot distinguish execution cost from descheduling on this machine. Live server CPU is still the final acceptance boundary.

1. Reuse the frozen source and precompiled test binaries. For each workload separately, run baseline/candidate/candidate/baseline with a fixed iteration count (e.g.1000x), rather than duration-based runs whose work counts change under contention.
2. Have the owning Python runner bracket each child with resource.getrusage(RUSAGE_CHILDREN), storing user+system CPU delta, elapsed time, iteration count, exact argv/payload/chunk/geometry fingerprint, load and co-tenant census. Sequential owned children make CPU deltas unambiguous; no per-ingest clocks, pinning, process interference or machine changes.
3. Include the existing byte controls and real complete-redraw workloads; separately diagnose VT and Frame.Validate with their baseline/candidate microbenchmarks. Those diagnostics identify a local gain or miss but cannot substitute for complete-ingest CPU improvement.
4. Retain every case and run. Require repeatable reduction in complete-workload CPU beyond observed variation and no material Unicode/mixed regression. Wall-time slowdowns stay visible. If CPU benefit also fails, shelve the shortcuts and return to measured hot paths; do not add lazy parsing or upstream suppression under this approval.
5. Only after that proof proceed to races/full suite, mandatory reviews and TL-coordinated live acceptance. No operator stage/restart requested yet.

This revision changes measurement qualification, not terminal behavior. It does not claim scheduler interference explains away contrary evidence.

## 2026-10-16 — approved revised attempt stopped on load guard

TL477ed170 approved one fixed-work CPU qualification with one-minute load below about8. The runner admitted at22:59:23 PDT with load7.9238. It completed only ingest1 baseline and first candidate (one million iterations each), then exited1 because load rose to8.3301. No ABBA workload completed and no complete-redraw case ran. This is incomplete evidence, not proof of benefit or of a CPU regression; no retry was made. Candidate remains held and is not eligible for acceptance or landing.

| Control | Child user CPU | Child system CPU | Total CPU | Wall time |
|---|---:|---:|---:|---:|
| one-byte baseline |2.166668s|0.032001s|2.198669s|2.190251s|
| one-byte candidate |2.117125s|0.030177s|2.147302s|2.146359s|

Raw partial attempt: fixed-cpu-once/ under the existing benchmark directory. results.json SHA2567d9756872fe632d79c5084c8297294bf15b74b294f6f8bd76c080927ea4f83f5. Baseline log19d2c825c3f0c5388d4e6948a17e6c703e1184cf9def59fc18d3735232abdf25; candidate log8f947ee283dc3059a8eccfb10f6816d7b1494f99d3450226c07a8e5f18a6fba5. Original runner session25768 is finished; no signal was needed. Existing attempt directory must not be removed to repeat qualification.

At TL rolling-relaunch request2e95b559 (23:39), the slot has no background test/suite/merge. Preserve the partial result, report idle for relaunch and await TL disposition before any further performance experiment. The new thread must inherit the operator's GOFLAGS setting rather than rebuild this experiment with old flags silently.
