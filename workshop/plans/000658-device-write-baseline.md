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
