---
id: 000865
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '763876b2c3f8dec156e13d9f8aa3a8c7c6646208' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T02:51:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
---

# Live Couch sustains ~1.3 cores: attribute and bound it

## Problem

**The live Couch spends about 1.2 cores continuously, mostly in the Chrome cast recorder (ops:0, 10-24 02:15).**
Measured on the operator's Couch (`~/workspace/pair/bin/couch`, up 5.5 h): 115–140% CPU sustained over several
samples, while `fseventsd` also ran at 120–140%. The machine's load average was ~22 on 12 cores, and ariadne's
suite shards ran ~1100 s instead of ~500 s, so Couch's share directly slows every recorded suite.

A 3-second `sample` of Couch, top of stack (Couch frames): `webcast/recorder.LowRes`, `image/png` decoding
(`readImagePass`, `filterPaeth`), `image/draw.DrawMask`, `image.(*NRGBA).SetRGBA64`, plus terminal emulation
(`vt.(*Emulator).handleGrapheme`, `terminal.plainText`, `ptychild.ingest`) and GC (`scanObject`). The Chrome cast of
echo's board was left recording; that page repaints every 100–200 ms (pair#849's finding).

Find the cause and bound it: e.g. decoding and downsampling every captured frame even when nothing changed, or
recording at full rate with no viewer. Also check whether Couch's writes drive the `fseventsd` load.

## Spec

Measure the sustained live Couch cost before choosing an optimization. The original recorder attribution was not supported by the denser sample: image functions were about3.4% of counted symbolized leaf samples, while GC/memory, terminal and JSON were larger. The current cast also differs from the originally observed echo board. Preserve these limits and do not infer fseventsd causality.

M1 adds launch-only COUCH_PROFILE_DIR, disabled by default: one finite CPU window plus allocation snapshots and build identity, private local files, no HTTP listener. Profile an isolated disposable Couch through the existing native Claude/Codex harness first. Only the operator may approve relaunching live Couch; TL asks only if disposable workloads fail to reproduce.

After attribution, M2 returns to TL with the measured owner, workload, growth driver and budget before optimization code. Deliver the actual cost correction, not merely the diagnostic switch. The canonical plan is [000865-live-couch-cpu-plan.md](../plans/000865-live-couch-cpu-plan.md); initial evidence is [measurement summary](../evidence/000865/initial-measurements.md).

## Done when

- CPU and allocation profiles identify the owner under a pinned executable and described workload, with disposable native controls first and measurement limits recorded.
- Profiling is off by default, launch-only, finite, private, non-clobbering and lexically joined; deterministic cancellation/IO/ownership controls and real pprof parsing qualify it.
- A TL-approved M2 correction meets a stated CPU/work budget for the measured workload, with counted scaling regression and before/after evidence.
- The fseventsd question is answered only to the extent supported by measurement; unknown causality stays explicit.
- Operator live relaunch requires their separate approval; no approval is inferred from profiling-plan acceptance.

## Plan

- [ ] M1 — Launch-only bounded profiling, deterministic controls, disposable attribution and evidence; mandatory review.
- [ ] M2 — Append measured owner/workload/budget design, obtain TL approval, implement and verify the cost bound; mandatory review.

## Log

### 2026-10-24

### 2026-10-24 — measurement and approved M1

Read-only attribution while #860 held this checkout measured135.69% CPU over25.00s, five PNGs and180.4KB/s whole-process physical writes. This does not attribute physical writes to the recorder or fseventsd. Matching-binary Console control passed30.17s; CPU profile13.78s/30.28s, sampled cumulative allocation10,396.63MB (test-host and resize/replacement overhead included). Two existing terminal benchmarks also passed; component costs are not live-call-rate evidence.

TL f34c93f4-ed6c-49cc-8ea7-bcc6a7f34e2d respecified attribution first; d515e211-98ad-4923-8896-0b50468764e0 approved M1, disposable first, M2 approval before optimization. #860 landed PR582/train350; #865 starts from refreshed main, without stacking. #863 landed109986901f7410e0120135121f9e906054603634 restores the shared explicit-hub harness; #866 native startup and #867 Codex idle remain known limits, not qualifications. #868 was briefly queued then withdrawn as pre-#825 stale-head evidence; no claim was made. Both claims #860/#865 overlapped while the former awaited landing.

ARCH-DRY uses the existing Runtime/Getenv, Console notice and testcouch owners; ARCH-ORDER requires profile acquisition before Stop ownership and joined cancellation. Source inspection found Go StopCPUProfile cannot report its writer failure, so the bounded output writer retains first failure.

## Revisions

- 2026-10-24 07:12 PDT — TL revised the original recorder hypothesis after the denser measurement: original filing is retained under Problem; active Spec/Done when now require attribution before optimization. M1 approved as above; M2 remains a measured design gate. No production code yet.
