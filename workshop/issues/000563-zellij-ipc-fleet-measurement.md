---
id: 000563
status: open
deps: [562]
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'c6840a55f9c21b83d8d2ec37dcc263729f02fd9c' # card fields mirrored from issue-cards; edit via sdlc
---

# Trace Zellij IPC failures and measure attach parallelism

## Problem

#218 implemented native candidate-only process enrichment and shared initial owner evidence, but the balanced9 warm attach/detach experiment did not complete. Successful Zellij inventories disagreed and another run refused malformed live pane JSON. Partial rows cannot select a new concurrency default. #562 owns the reply/discovery race and independent classification of the CI empty PID sidecar; this follow-up waits for that stable observation contract.

## Spec

First perform a separately authorized passive IPC-frame diagnostic of one unchanged workload. Capture socket path, subprocess identity, descriptor lifetime, connect/send/read outcome and complete decoded response frames. A plain syscall timing trace without received bytes is insufficient. Match timestamps to existing pane-close operations to distinguish rejection of an unblock from attribution of its emitter. Do not install patched upstream software or change plugin workload as part of an unapproved measurement repair.

For an omitted owned session, first response UnblockInputThread establishes the rejection mechanism; connect/read/decode failure identifies another path; Connected contradicts this hypothesis. A run with no omission is inconclusive and does not authorize an automatic retry. Tracing may change timing. Naming --session or switching listing formats still invokes upstream discovery.

After #562 and protocol evidence make the measurement valid, run a separately versioned balanced9 schedule [6,20,40,20,40,6,40,6,20] against21 controlled persistent sessions using production Console workers and LeaveDetach. Preserve every failed setup/row and exact private cleanup evidence; do not filter failures, silently continue after uncertainty or pool harness/source versions. A failed or incomplete schedule remains incomplete. Any continuation-after-failure protocol must be explicitly designed and approved first.

Measure meaningful first terminal frame for every client, exact native client births, attached/post-leave observations, agent/server/sentinel survival and exact-generation cleanup. Attribute parent claim, prepare, spawn, durable record/baseline, registration and promotion phases, plus child phases, Leave and actual native API counters. Durability must be explicit: record+baseline includes durable work but does not isolate fsync. Report Couch CPU, host load/idle, actual worker peak, and overlapping phase durations without summing them into wall time. Identify the source/archive/binary hashes, workload and limits. Operator restart/live check is coordinated by the TL, not a private-fixture substitute.

No concurrency-default change without complete evidence supporting it. Keep historical cold-start attribution separate in #560. Reuse #218's retained fixtures, query/stage observers and cleanup guards; no new ownership cache or authority relaxation.

## Done when

- Passive response-frame evidence classifies a reproduced omission/reply failure, or the one-shot diagnostic is explicitly recorded inconclusive; no unsupported IPC-cause claim.
- A complete, versioned balanced9 measurement records all21 meaningful client frames, attach/adoption/Leave times, phase attribution, native counters, Couch/host load and exact survival/cleanup proofs for each row.
- Failed attempts remain first-class evidence and incomplete runs are never presented as successful comparisons.
- The TL/operator reviews the comparison and any concurrency choice; the production default changes only when the evidence supports it.

## Plan

- [ ] Resolve #562, then agree the bounded passive trace protocol and unchanged workload with the TL.
- [ ] Capture and classify exact IPC responses without retries or guard relaxation.
- [ ] Freeze the accepted source/binaries/harness, run balanced9 and preserve diagnostics before cleanup.
- [ ] Report complete and failed evidence, coordinate the real restart timing, and choose concurrency only from supported results.

## Log

### 2026-10-14

Unassigned follow-up split from #218. MATRIX4 retained6 successful rows before row7 Leave refusal; MATRIX5 stopped sample1 after inconsistent successful inventories. Earlier matrices and limitations stay in #218. No passive tracer installed or run; no new experiment or default change performed at filing.
