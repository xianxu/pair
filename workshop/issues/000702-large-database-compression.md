---
id: 000702
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '41cbaac50cc540f7b6e3dfbd0522dff884c841ee' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T04:24:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Compress large database snapshots faster with measured levels and threads

## Problem

#688's coherent 1,999,003,648-byte history snapshot acquired in 7.31s, but the following existing zstd -19 --single-thread plus gpg pipeline exhausted the acceptance pass's five-minute outer budget. Large daily database compression, rather than snapshot acquisition, now dominates latency. The snapshot reader had already exited and its WAL did not grow.

## Spec

Measure size, wall time and CPU for large database snapshot compression at levels -3, -9 and -19, with multithreading (-T0) compared with the present single-thread path. Choose a bounded kind/size-specific policy that makes daily database backups practical without weakening verified encryption, content hashes, manifests, source preservation or interruption/retry semantics. Keep -19 for small text bundles if the size benefit justifies it. Format/manifest compression-description compatibility and unknown fields must be considered if the format actually changes. Coordinate heavy benchmarking with the hub; do not run an unrestricted multithread benchmark alongside fleet suites.

## Done when

- A retained validated large database snapshot and representative small text bundle have reproducible size/time/CPU measurements for the proposed choices.
- The chosen database policy is enforced through the existing codec seam, with a stateful fake and real CLI qualification, bounded concurrency/cancellation and no orphaned compression children.
- Backup round-trip and retry/integrity tests pass, and operator-facing storage/time tradeoffs are documented.

## Plan

- [ ] Measure retained history snapshot at -3/-9/-19 with -T0, plus a representative tracked-text tar, under both machine-test tokens and one 20-minute outer bound. Record wall/user/sys CPU, bytes, zstd version, load and decompression hash agreement; remove private compressed outputs immediately.
- [ ] Send measured policy and compatibility choice to the hub before code; add failing policy/codec/reader tests, then implement through the shared codec seam with truthful compression metadata.
- [ ] Qualify native round trips and cancellation, document storage/time tradeoffs, run recorded suite, close and land on SHIP.

## Log

- Filed unclaimed at hub request e253ca94-31a2-4866-b377-f8f2aa5c48c5. Independent follow-up; it must not block #688. That item publishes its already validated stage at low priority under an approved longer bound, without another native source read/pin.

- First exclusive measurement, zstd 1.5.7 on 12 CPUs, warmed staged 1,999,003,648-byte stream: -3 -T0 wall 1.074790s, user 3.90s/sys 0.60s, 286,271,640 bytes, peak RSS 149,291,008; -9 -T0 wall 2.206355s, user 16.16s/sys 0.94s, 205,249,063 bytes, peak RSS 413,925,376. Both decompression SHA256 results matched the stage. Load1 around 7.4, both machine-test tokens held, nice 10. Evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair702-bench.jy01Ph/results.json and measure.py. The outer bound interrupted -19 before a valid result; its partial compressed output was removed, not measured, and the process group/tokens released. No native database was opened or acquired.

## Revisions

- 2026-10-19, hub 6c554939-71f7-4028-a1f0-6f5cc948a40d: the first conservative 20-minute clock included 18m55s queue time, so it stopped during -19 after approximately 65s held-token work. Resume only -19 -T0 and the small-text comparisons, with an 18-minute bound starting at acquisition; queue waiting holds no resources and does not count. Keep completed -3/-9 measurements, verify the same staged-input hash, and release immediately after cleanup.

- 2026-10-19, hub 69176f28-7984-488e-a48d-754e02e0e524: both machine-test tokens approved for measurement only, capped at 20 minutes including cleanup and released immediately. Measure before proposing the policy. Compression level is an existing value, not a schema-shape change: if a lower level wins, readers should accept supported zstd levels while writers label the actual level honestly; no speculative codec/schema expansion.
