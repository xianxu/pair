# #658 whole-suite device-write baseline

Measured 2026-10-17 on the unchanged implementation at b08a927f6fed942fdbb4aef84b09cf07b2a32f47. This supersedes the initial own-test-process measurement as the scale reference; it does not approve an implementation.

## Method and result

One full normal-cache `make -k WF_TEST_WEIGHT=2 test`, including its fresh shared Pair build, shell/Lua prerequisites and final `go test ./... -count=1 -timeout 20m`. Both ordinary suite tokens were held throughout 30 seconds before, the full Make body, and 30 seconds after. Ambient Pair/Couch/Zellij authority was removed; no machine/cache settings changed. Pair:4 held its rerun for the quiet window and received a release notice afterward. This diagnostic is not the recorded acceptance suite required after an implementation.

**Make exit 0; all 116 final Go packages passed.** Make body: 844.188 seconds. Physical disk0 IOBlockStorageDriver Bytes (Write) rose from 35,275,342,270,464 to 35,295,111,237,632: **19,768,967,168 bytes (19.769 GB), averaging 23.418 MB/s**. The original 1,640,353,792 sampled own-test-process bytes are a partial view, not the suite's device cost.

| Device interval | Bytes | Scope |
|---|---:|---|
| Before final Go recipe | 4,214,910,976 | shared build plus shell/editor prerequisites |
| Final Go recipe through Make completion | 15,554,056,192 | linking, tests, nested builders/children, filesystem work, host background |
| Whole Make | 19,768,967,168 | physical device counter; all host work |

The Go recipe marker is 515.604893 seconds after observation start. Its nearest device sample is 515.805910 seconds (0.201017 seconds late); phase boundaries are approximate and can include delayed filesystem writes. Large preceding recipe intervals include install-layout ~1.183 GB, review-branch-restore ~0.553 GB, session-watch ~0.369 GB and lifecycle-command-nvim ~0.333 GB. These are device intervals, not exact process/path charges.

Quiet-before: 118,431,744 bytes in 30.029 seconds (3.944 MB/s). Quiet-after: 74,702,848 bytes in 30.104 seconds (2.482 MB/s). Do not subtract these slopes as an exact background allowance: other host work remains possible and the post window can include delayed suite flushes.

## Owned descendant counters: a lower bound

Identity-fenced cumulative maxima from 832 discovered process generations total **2,288,795,648 bytes**. Full executable names classify native Go tool processes; macOS short `comm` names alone truncate several of them identically.

| Observed executable class | Bytes |
|---|---:|
| tests | 1,264,066,560 |
| linker | 529,874,944 |
| go driver | 468,422,656 |
| nvim | 9,641,984 |
| git | 7,716,864 |
| other | 5,558,272 |
| pair | 1,830,912 |
| compiler | 1,048,576 |
| shell | 634,880 |

Sampling misses short-lived descendants and final increments before exit. In particular couchcmd's last observed 355,708,928 bytes is below its ~625 MB standalone baseline. Small sampled compiler/Git totals do not establish small actual costs. Writable-FD paths/offsets name leads; they are not per-path byte totals. Lifetime counters grouped by first-observed phase are not incremental phase counters.

## APFS and VM views

Nearest volume samples bracket the Make window with errors -11.732 seconds and +1.291 seconds. These counters overlap and are **not an additive device-byte partition**.

| APFS volume | User-written bytes | Block-device bytes | Metadata bytes |
|---|---:|---:|---:|
| Data (disk3s1) | 3,813,843,727 | 5,301,391,360 | 5,216,157,696 |
| AgentScratch (disk3s7) | 1,358,851,884 | 1,930,911,744 | 373,342,208 |
| VM (disk3s6) | 0 | 0 | 0 |

Data metadata includes 2,281,037,824 omap, 1,736,642,560 fsroot and 970,801,152 blockref bytes. AgentScratch and Data share internal physical store disk0s2; AgentScratch is neither RAM nor another SSD. VM swapouts/pageouts did not increase. There is no evidence of swap growth explaining this run. These counters show substantial metadata traffic but do not identify its exact producer or justify labeling all unaccounted device bytes as metadata.

## Excluded-phase leads and ruled-out shortcuts

- Shell bootstrap forwards explicitly exported GOCACHE/GOMODCACHE only. This parent exported neither; its Go settings come from the user Go env file. Empty environment plus disposable HOME resolves new private caches. Ordinary install-layout and embedded-runtime recipes contain real Go builders. The observed install-layout `go build ./cmd/couch` alone had 280,342,528 sampled own bytes. Existing `tests/with-isolated-pair.sh` demonstrates resolving cache configuration before changing HOME.
- The clean-source runtime generator test intentionally creates an empty compiler cache; preserve its clean-source contract.
- Installed Go 1.27.1 source confirms `-count=1` forces fresh test execution/linking, and final test executables are not put in the ordinary build cache. A read-only mixed historical/running census found 58 `.test` files totaling 698,365,876 bytes; multiplying its mean by 116 gives roughly 1.397 GB of link output, not a matched-suite exact count.
- Ordinary Go test already omits debug/DWARF except special binary/profile cases. Explicit `go test -c -o` retains it, so its 38 MB couchcmd binary is not a representative ordinary-suite output. Adding stripping flags is not a supported new saving.
- The proved ledger clone experiment saves roughly 1.2 GB of payload writes, about 6.1% of the raw device total before any filesystem amplification. Keep it secondary; it cannot by itself explain or eliminate the dominant measured cost.

## Remaining gap and next diagnostic

The largest measured interval is the final Go phase (15.554 GB), but process sampling does not yet assign its dominant device cost to particular paths/operations. Investigate that interval's bursts and filesystem metadata/file-creation producers before choosing the headline change. Do not claim the sampled process sum is complete, add APFS views, or equate device writes with NAND wear. If bounded unprivileged attribution cannot name the producer, request the hub's operator-run 60-second filesystem trace for owned test processes as authorized in the dispatch clarification. No production/test implementation edits or change-code yet.

## Local raw evidence

Diagnostic root: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1`.
`device-suite/result.json`, `device.jsonl`, `phases.jsonl`, `volumes.jsonl`, `analysis.json`, and `make.log` retain the full run. `device_suite.py`, `treeprobe.c`, `treeprobe.dylib`, `volume_watch.py`, and `analyze_device.py` retain the measurement implementation. Device-start epoch anchor: 1792301602.6685076. Raw local evidence is temporary; the exact summary and limitations above are the durable record.

## Follow-up: bounded serial couchcore churn probe

Hub approval receipt 18724da1-b2ea-4b45-94d5-f905718ce94e was verified Submitted. The probe used a freshly linked stripped couchcore executable, a private short TMPDIR, timestamped verbose test events, identity-fenced descendant counters, and an unprivileged native FSEvents observer of that owned root. No product source or machine settings changed. The observer's calibration saw all 100 files and 10 directories created/removed, with no dropped/scan flags; an immediate-shutdown calibration also retained 100/100 file creations after adding a settling grace and callback-queue drain.

**This is a partial diagnostic, not a passing package run.** Its eight-minute serial bound was chosen incorrectly: the existing successful serial baseline took 634.448 seconds. The test process reached that diagnostic deadline while TestWorktreeAddUsesWriteBudget had run for only two seconds; there were no preceding FAIL lines. Retain exit2 and the full log as a harness configuration error, not an established product flake or acceptance evidence. No blind full-package rerun. Observer shut down cleanly; none of its measured owned PID generations remained alive afterward.

In 482.889 seconds the observer recorded 115,221 events with zero dropped/MustScan flags. It observed at least **23,465 distinct file paths and 18,253 directory paths created** under this private root. These are lower bounds because events coalesce, repeated creation of the same name is counted once, and hardcoded temp roots outside TMPDIR are excluded.

| Created file-path class | Distinct paths |
|---|---:|
| Default Git sample hooks | 7,588 |
| Other identified Git metadata | 7,553 |
| Git objects | 2,788 |
| Git refs | 1,323 |
| Other files | 4,213 |

The sample hooks are 14 inert template names across 542 repository copies, comprising about 32.3% of observed file paths. Source confirms real init/commit/bare-clone setup in newProvisionFixture (62 source call sites) and seed/bare-origin/host-clone/substrate setup in newPrepareWorld. No test assertions of Git template/hook contents were found. An empty private Git template is a reduction hypothesis, not an approved change or a proved device-byte saving.

Sampled native descendant bytes total76,378,112 across only28 generations, again incomplete; counters do not assign the metadata amplification or all brief Git subprocesses. Device counters overlap another admitted suite in this run and are not attributed to couchcore. Go's installed Darwin File.Sync implementation normally issues fcntl(F_FULLFSYNC), with fsync only on ENOTSUP, so fsync-only tracing would miss the ordinary flush boundary.

Additional local evidence: core-probe/test.log, test-events.jsonl, fs-events.jsonl, process-samples.jsonl, processes.json, result.json and analysis.json. Temporary observer/calibration sources and logs remain at the diagnostic root. A tiny owned Go/libc interposition calibration observed mkdir/open-create/rename/unlink and both flush APIs; its generic variadic fcntl shim is not approved for arbitrary product tests. Continue with a narrow safe flush-count probe or the operator's bounded fs_usage trace before claiming a causal metadata saving.

## Final pre-code probe: selected root F_FULLFSYNC calls

Hubf6308a17-cd1f-4897-9d24-63e72d8c47cf approved exactly one short owned/narrow flush probe, followed by planning. Completed exit0 in12.650 seconds, all four selected tests passed. Root/private-TMPDIR scope observed274 real F_FULLFSYNC attempts:265 during TestSlotPreferencesIndependentAcrossRestartResumeAndFresh, five during TestLocalJournalRecoversImagesLargerThanPayloadReadLimit, four during TestResourceHistoryStoreFixedStageAndByteBound, and zero during TestRunPrepareRefusedAdmissionChangesNothing. This distinguishes substantial root durability traffic from the Git fixture-creation path; it does not prove an exact metadata byte charge for either.

The arm64 thunk tail-forwards every nonmatching fcntl call unchanged. For F_FULLFSYNC it preserves x0–x8, stack and link register, calls a root-PID/canonical-owned-path logger, restores errno/arguments, then tail-forwards the real fcntl. Calibration preserved F_GETFD/F_SETFD/F_GETPATH arguments and results; five File.Sync calls produced five markers. Child logging is disabled by the inherited original PID. No persistence call was bypassed. Only attempt counts are measured; no new device saving is inferred.

File events for those four tests observed623 distinct created files and349 directories, zero dropped/scan flags. Last native own sample23,777,280 bytes is a lower bound. Neither file-event counts nor these bytes represent the whole package. Raw local evidence is flush-probe/{test.log,fs-events.jsonl,output-events.json,result.json}; the safe temporary thunk source/binary is fcntl-thunk.UNQeGX/. These diagnostic tools do not ship. Pre-implementation measurement is finished.

Installed default sample-hook contents total26,788 bytes across14 files per template copy; the542 copies observed in the partial probe imply14,519,096 logical hook bytes (35,520,512 bytes of source allocated-size equivalents). These are estimates from fixture copy count and installed source sizes, not measured per-path writes; metadata amplification remains unknown. The replacement plan ranks this churn reduction alongside cache reuse and the proved ledger payload saving without adding them into an exact device-byte decomposition.


## Direct cache comparison after implementation

The matched direct-cache diagnostic completed both real shell fixtures, `tests/pair-go-install-layout-test.sh` and `tests/pair-embedded-runtime-test.sh`, before and after the cache-boundary change; both passed in each run. Each run executed six real Go build commands. The before source receipt records HEAD `d025e19ee8b23058ef11cefa2cb918827e4f8489`; the after receipt records HEAD `af5ef64f31f21d84229b7647cb99407ed8a36c79` plus hashes of the uncommitted cache helper changes. The two fixture scripts and `Makefile.local` hashes match across the receipts.

The harness used an invocation-owned Go env file, compiler cache and GOTMPDIR. GOCACHE/GOTMPDIR were unset in the fixture caller, so the real caller-side `go env` resolution was exercised before replacing HOME. Existing module input was explicitly forwarded to avoid introducing downloads. An owned priming run prepared the configured compiler cache before the comparison; the operator's compiler cache was neither inventoried nor copied. Before cleanup, the wrapper inventoried private compiler-cache files and fresh executable outputs; the after run inventoried additions to the same owned configured cache.

| Direct inventory | Before | After |
|---|---:|---:|
| New compiler-cache files | 5,347 in private homes | 22 in configured cache; no private cache bytes created |
| New compiler-cache logical bytes | 530,303,396 | 672,620 |
| New compiler-cache allocated bytes | 549,085,184 | 753,664 |
| Fresh executable output logical bytes | 77,109,896 | 77,109,896 |
| Sum of six build-command durations | 86.077 seconds | 11.832 seconds |

The configured cache started the after run at 3,903 files /471,693,421 logical /485,199,872 allocated bytes and ended at 3,925 files /472,366,041 logical /485,953,536 allocated bytes. The net file creation avoided in this pair of real fixtures is **5,325 files, 529,630,776 logical bytes and 548,331,520 allocated bytes**. All recorded Go invocations exited0. The four fresh output files retained the same individual sizes (19,966,818;32,868,194;4,242,018;20,032,866 bytes), and the fixture scripts retained their real install/runtime assertions.

This measures completed cache-file inventory additions for two normal-cache fixtures. Allocated bytes are filesystem allocation, not physical-device traffic, NAND wear or a complete count of transient build writes. Build-command durations are sums from this run, not a controlled suite-speed benchmark. Warm configured-cache reuse is intentional; clean-source tests that explicitly request a fresh compiler cache keep that override. Neither these bytes nor the ledger/Git observations establish an additive decomposition or a matched reduction of the19,768,967,168-byte whole-device baseline. A whole-suite after measurement remains required.

The portable stateful fake-Go regression separately checks8/16 constructions per boundary (32 fresh HOME directories total): one fixed payload for the shared cache key, one separate payload for a distinct explicit key, literal values/spaces, GOCACHE=off, explicit empty GOTMPDIR, missing Go and resolution-error status37. No real builds run in this regression. Existing bootstrap assertions still cover original argv/$0, status23, read-only cleanup and refusal to follow linked caller trees. Its cleanup-only fixture now explicitly supplies Go tool settings because real `go env` can create telemetry/config state in its synthetic caller HOME; this isolates test inputs without removing any caller-state assertions.

The canonical empty-environment boundary also supplies `GIT_TEMPLATE_DIR=`. Real-Git regressions first reproduced operator hooks materializing through both `testpair.Environment` and `ApplyEnvironment`, then passed with the empty setting even when the fixture HOME's global `init.templateDir` names a populated template. Explicit fixture assignments may still override the default. The empty setting requires no additional directory or cleanup owner and does not inherit the operator's template path.

Raw evidence remains in `direct-cache.95x7rwyt/` under the diagnostic root above: `before.summary.json`, `after.summary.json`, source receipts, configured-start inventory, per-Go records, logs, `goenv` and `run.sh`. These temporary files are not the durable record; the figures and limits in this section are.


## Incomplete after attempt retained: review resolver failure

The first corrected-source after attempt used clean91d46ca2f5c0c3b1d5537fbb5b28c2a95b079bb1 and the same Make body/two-token/quiet-window method. Make exited2 after312.427012s because the review-observation diagnostic’s positive control timed out in its real BufWritePre resolver (2578.387417ms against2500ms, code124/signal9, empty result). The10-test diagnostic enumerated one failure; Make completed other prerequisites but suppressed its final full-Go recipe. Only the earlier targeted titlepoller/procutil packages ran. Filed unclaimed pair#662, cross-referencing #623/#659; the unchanged recorded full suite subsequently passed the diagnostic and all118 final Go packages.

**This is not a matched whole-suite after result.** Device registry4294970125 was stable; endpoints35,612,851,138,560→35,615,196,553,216 yield2,345,414,656 bytes. All488 raw/embedded samples agree and have no counter decreases. Sampled descendant own bytes161,669,120 across437 generations remain a lower bound; the2,183,745,536-byte difference is unexplained, not a causal charge. Quiet windows observed213,852,160/30.044595s before (7.118MB/s) and75,165,696/30.103872s after (2.497MB/s). Do not subtract these exactly or compare the incomplete body’s total as a saving against the complete19.769GB baseline.

Raw evidence and explicit-anchor analysis: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-device-after.SjFHBp/device-suite/{result.json,device.jsonl,make.log,analysis.json,counter-checks.json,body-source.json,supervisor-exit.json}. Supervisor exited2 and reaped all direct children after the post-window observer. Hub approvala26d5339 authorizes exactly one issue-backed unchanged device rerun after the recorded suite, same body/bounds; a repeated pre-Go failure means stop/report. That approved attempt is queued separately at pair658-662-device-attempt2.DViNQs and is not yet a comparison result.


## Complete same-body after comparison

The single hub-approved #662-backed unchanged device rerun completed Make/supervisor exit0,118 final Go package passes plus two earlier titlepoller/procutil runs, no FAIL/panic rows, and all direct children reaped. Source030bdf8a41810d07df15ba68bd5d8295c056b22f was clean immediately before Make; code is unchanged from the corrected package profile/recorded suite. It used the same make -k WF_TEST_WEIGHT=2 test body, normal caches, original native read-only probe, APFS observer and30s quiet windows. Queue1754.536273s; both real suite tokens acquired at epoch1792313068.679295, body source pinned at1792313098.748662, supervisor exit1792313778.691938. Hub startb389d04a/end7c691ddc notices both verified Submitted.

| Whole Make observation | Before | After |
|---|---:|---:|
| Physical disk0 bytes |19,768,967,168|15,098,630,144|
| Body seconds |844.188058|643.772099|
| Average MB/s |23.417729|23.453378|
| Sampled descendant own bytes, lower bound |2,288,795,648|1,038,913,536|
| Discovered process generations |832|825|
| Quiet-before MB/s |3.943917|2.882504|
| Quiet-after MB/s |2.481531|0.889555|

Observed raw device difference is4,670,337,024 bytes (23.624588%), with a200.415959s shorter body. The nearly unchanged write rate, different quiet slopes, normal cache state and incomplete process sampling prevent treating that difference as an isolated payload or causal device saving. No exact background subtraction, additive ledger/cache/device accounting, or NAND-wear claim is made. Matched serial own-test-process evidence separately shows72.824% lower bytes across116 packages, including couchcmd90.949% lower; the two added helper packages are separately reported.

After exact disk0 endpoints are35,673,760,194,560→35,688,858,824,704, registry4294970125 stable. All976 raw samples match embedded samples, counters never decrease, and endpoint subtraction agrees. The matched original116 package names are all present; new testdisk/testgit add two final packages. The earlier incomplete exit2 attempt and its passing unchanged recorded-suite control remain separate evidence above.

Using the original event-boundary method (nearest device sample to the final-Go recipe event, exact Make endpoints):

| Interval | Before bytes | After bytes |
|---|---:|---:|
| Before final Go |4,214,910,976|1,786,322,944|
| Final Go through completion |15,554,056,192|13,312,307,200|

Nearest-sample errors are+0.201017s before/+0.202169s after. The separate later-sample phase-label partition gives a Go bucket of15,640,170,496→13,390,426,112 bytes; that bucket includes earlier targeted Go work and boundary increments. Both are approximations of phase scope; never substitute one method for the other baseline.

After sampled own-process categories are: test398,409,728; link545,906,688; Go73,302,016; Git2,834,432; nvim9,707,520; Pair2,228,224; shell688,128; other5,836,800; discovered compiler0. Missing short processes/final samples make these lower bounds; zero observed compiler bytes is not proof compilation wrote nothing. Device minus discovered own lower bound is14,059,716,608 bytes (93.119%), which is unexplained, not an attributed filesystem/child/host charge. Paths remain sampled writable-FD high-water marks, not per-path byte totals.

After APFS nearest-window errors are−11.024661s/+6.077542s. Data deltas:1,468,684,556 user bytes,2,733,666,304 block bytes,4,515,151,872 metadata bytes. AgentScratch:1,409,522,673 user,1,669,398,528 block,84,103,168 metadata. These overlap and cannot be added to device/process totals. Swapouts/Pageouts unchanged;16 Swapins are a distinct VM observation, not an inferred write charge.

Filed unclaimed **pair#663**, remaining-suite-device-writes, naming the largest remaining observed interval:13,312,307,200 final-Go device bytes, about88.2% of the after total. It requires bounded attribution before another optimization and retains the process/metadata/host uncertainty; no additional probe or durability change is part of658.

Raw after evidence:/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-662-device-attempt2.DViNQs/device-suite/{result.json,device.jsonl,phases.jsonl,volumes.jsonl,make.log,analysis.json,counter-checks.json,body-source.json,supervisor-exit.json}. Original tools/scripts and the failed attempt are preserved. These temporary files are not the durable record; the exact comparison, pass/failure disposition and limits in this report are.

## Revisions

- 2026-10-17 — Append the completed direct real-cache before/after comparison, separate cache allocations from fresh executable outputs and physical-device traffic, and record canonical empty-template behavior plus cleanup-fixture input isolation. Earlier diagnostic results and their original scope remain preserved.

- 2026-10-18 — Append the complete approved unchanged after comparison, distinguish shorter-run raw totals from independently measured payload/counter savings, retain both phase-boundary methods and file unclaimed #663 for the remaining final-Go attribution.
