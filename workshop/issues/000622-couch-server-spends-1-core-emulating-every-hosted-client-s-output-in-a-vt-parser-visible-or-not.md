---
id: 000622
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 1.509
card_mirror: '7c0b7bf4443d3d7dee4ca642af786ccf17a62a32' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T17:03:46-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.

## Spec

- Measure first: per hosted client, ingest bytes per second and CPU, at idle and during a busy turn; which consumers
  need the emulated screen (console view, peek, waiting/idle detection, resource monitor).
- Then the smallest change that keeps every consumer correct. Options to weigh: emulate only the slot(s) the console
  is showing and keep a cheap raw ring for the rest (replaying it when a slot is shown or peeked); or parse lazily
  (append raw bytes, emulate on demand). Idle/waiting detection must keep working for off-screen slots.
- No new background workers; bounded memory per client.

## Done when

- The Couch server's steady CPU with ~27 hosted clients drops by a measured amount (before/after `sample` and CPU%),
  with the console, `--peek` and waiting notices behaving as before (tests plus a live check).

## Plan

- [ ] M1 — land permanent per-hosted-client ingest-byte diagnostics through read-only resources; no parser-policy change.
- [ ] M2 — after the combined restart, measure natural idle/busy rates and server CPU, choose the smallest correct optimization with TL approval, then implement and verify it.


## Log

### 2026-10-16

### 2026-10-16 — initial measurement and consumer audit

Claimed/started on fresh main b95df08cebf86cc71cd86a27dfdf2fdb96f1b91d. Slot-private Ariadne refresh advanced its clean checkout from 1fbc10d2 to e1bb1ba5 after start-plan's fleet refresh refused the unrelated untracked pensive; the pensive stayed untouched. No feature code changed.

Live supervisor PID71982, launched 14:57:39, has 23 direct hosted resume children at the new observation. Cumulative CPU rose 107:20.99 → 107:40.38 over 30.1086775s, or 64.40% of one core. Identity unchanged. This is a natural mixed-load observation, not an idle-fleet result. Baseline /tmp/pair622-live-baseline.json SHA256 956d9eb679e309e51ab0b874c0f436b812cb2ea95541bb87c50f9dec1c5bd7d4. Read-only 5s native sample at 17:06:26 retained /tmp/pair622-couch-71982-native.sample.txt SHA256 414d34db020424e9ccccfd9c69776a8fae33a4fc03393dea7b80c956ba0101aa. Its call graph includes Child.ingest and VT parsing; no percentage is inferred by dividing inclusive counts or summing overlapping stacks. The binary currently at the path reports 9fc035e718b1, but its mtime is later than server launch; that is not proof of the loaded image's revision.

Read-only consumer audit: selected console frame/history, mouse/key encoding, query replies, effects, resize and EOF need the hosted Endpoint authority. Off-screen waiting/picker notices and peek use wrapper tails/status; resource idle uses wrapper settled/picker/broker state; idle fading uses thread activity. These do not require the hosted screen. The raw diagnostic ring is bounded at 128KiB and discards arbitrary prefixes: existing TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction proves screen/modes predate its tail. Replaying that tail cannot safely replace the authoritative emulator. Lazy parsing must also preserve prompt queries, exactly-once effects, ordered geometry and EOF. Details and tradeoffs are in the durable plan.

No exact per-hosted-client ingress counter exists in this running binary. Existing capture could provide bytes but requires startup opt-in. TL instead directs permanent always-on atomic byte counters as M1, exposed read-only through resources/debug; optional emulator CPU timing only if cheap. They land before the #618/#617 combined restart, then measurement and optimization design follow. The posted direction's canonical sender/body matched receipt 9929f33e; its delivery outcome is indeterminate, even though this turn received the text. No capture restart was performed.


### 2026-10-16 — M1 implementation and targeted evidence

Plan-quality resolved PQ-1 CLEAN; change-code admitted at 242bd40216171712db036069ca5aae4f0c381366. Estimate-quality INFO calls the resources/check budget optimistic; 1.509h is cumulative M1 effort including initial shaping, not a remaining-work or whole-issue forecast. M2 remains unestimated until its measured design exists.

RED checks confirmed the missing counter, Console method and resources types/wiring before implementation. GREEN: fresh fake Child → Console → production resources handler → real socket → JSON/text CLI totals advance independently of an intentionally stale process sample; request leaves cache unchanged. Encoded maximum-detail response remains in MaxFrameBytes and reports hosted omissions without mutating its source. Clone/projection identity/isolation checks pass at 0/1/27/256 rows. Counter tests cover generated chunk partitions, eviction, later refusal/publication failure, independent incarnations and concurrent exact totals. Real Child observer barrier proves the diagnostic snapshot completes while the parser mutex is held. Targeted race command go test -race ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/ptychild -run 'TestResourcesHosted|TestResourceHosted|TestResourceProjection|TestHostedIngest|TestChildIngest|TestResourceInventory' -count=1 passes all three packages. No parser policy, live restart or per-client CPU attribution changed.

Counter benchmark control: exact committed pre-counter baseline 242bd40216171712db036069ca5aae4f0c381366 and current working source ran the identical complete-ingest benchmark serially, GOMAXPROCS=3, GOFLAGS=-p=3, three 300ms repetitions. Median ns/op baseline→current: 1-byte 6170→3115, 64-byte 40004→18244, 4096-byte 1116835→959324. Baseline 1-byte spread 3584–9665ns/op and changing contention prevent credible incremental overhead attribution; apparent speedup is inconclusive. Raw logs and metadata retained in /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-ingest-bench.hqUsTx/. Temporary isolated baseline worktree removed by its owner; no peer runner was signalled. Source inspection confirms the incremental hot path is one atomic addition without new allocation/clock/native call. Counter race rerun after test-budget guards passed (1.229s).

### 2026-10-16 — PR and pending gate evidence

M1 candidate aaf17569ea4005aa59502de3a3e47ecacb01dada published as PR382, https://github.com/xianxu/pair/pull/382. Main integration check was already current. Required recorded suite is queued as sdlc local-check run --name suite on that pushed code head, local runner sdlc-local-2321322240; no suite pass or milestone acceptance claimed yet.

Affected-package run encountered TestWorkbenchPerformanceIdleUITick: activity worker did not finish, test50.43s/package101.425s. Unchanged targeted repeat on the same code head passed30.16s with read/decode accounting intact. This fixture creates no Child and does not request resources; relevant activity/runtime/fixture files are unchanged from pre-counter242bd402. Fixed30s test guard versus sequential20×30MB cold ledger probes is a demonstrated tight boundary, but the original scheduling/progress cause remains unproven. Filed unassigned #626 with both attempts and repeat log. No unrelated fixture/production change or blind full-suite retry was made; the first recorded suite remains queued. Remaining affected packages continue to completion.

### 2026-10-16 — affected-package census complete

The remaining affected packages completed: couchcore passed531.520s and artifactpath passed11.803s. The aggregate command exited1; its only failure was TestWorkbenchPerformanceIdleUITick (#626). Preserve it as a failed run, alongside the unchanged targeted pass, without calling the aggregate green. Full recorded suite is still queued.

### 2026-10-16 — recorded full suite passes

Required sdlc local-check run --name suite passed on pushed code head aaf17569ea4005aa59502de3a3e47ecacb01dada, execution10m46s after queue1671s. Recorded sdlc-local/suite; log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/aaf17569ea4005aa59502de3a3e47ecacb01dada/suite.log, SHA256 prefix e13259c07f61. All shell/editor checks and Go packages completed. couchcmd51.455s, couchcore316.270s, couchtty45.657s, ptychild0.837s, wrapcmd49.269s. The earlier #626 affected-package failure remains retained; the unchanged full recorded run passes, including that activity test. No retry, timeout increase, test exclusion or production change was used. PR382 CI conformance/merge/local-gates checks also passed on the docs-only descendant. Preparing M1 boundary review, keeping M2 unchecked.

The two interim hub outcome reports a523d633 and20027db5 expired waiting for a recognized composer; those queue-state reports are now stale. A fresh final outcome will be sent after the M1 gates/landing; no peer keys or operator stage were used to force delivery.

## Revisions

### 2026-10-16 — measurement milestone before optimization design

TL direction 9929f33e replaces the proposed capture choreography with permanent M1 ingress diagnostics. Original whole-issue CPU reduction and live consumer acceptance remain owed in M2; M1 does not claim them. One atomic byte increment per Child.ingest call; no per-chunk clock/syscall/worker or raw payload retention. CPU attribution uses the live server sample until a cheap accurate per-client CPU seam is established; wall elapsed time will not be labelled CPU. Full-flow milestone review is used because an explicit M1/M2 sequence lies outside the binary's quick-flow shell; the implementation target remains a small diagnostic diff.

### 2026-10-16 — M1 plan posted; baseline controls pass

Concrete M1 plan 044ec5dc278c07223a413253e32ba445e0a59fa9 sent to hub as 9c72573b-6098-4f22-aa21-833ce60f5e42; submitted confirmed. The message names fresh read-only counter snapshots, server/client identity, explicit wire omissions and omission of misleading/expensive CPU timing. Awaiting response before feature code. Existing authority/projection controls pass: go test ./cmd/internal/ptychild ./cmd/internal/couchcmd -run 'TestEndpointRetainsFrameAndModesAfterDiagnosticRingEviction|TestChildNotificationIsTypedOnceIndependentOfRawRetention|TestResourceProjection' -count=1 (ptychild 0.217s; couchcmd 0.356s). No production/test implementation changes yet.

### 2026-10-16 — M1 approved; plan-quality refinement

TL e6511b06-0db1-4dae-a695-ca1c364c46f5 (submitted) approves M1 as planned, explicitly without per-feed CPU timing and with full gates. Plan-quality round 1 raised only Important PQ-1: replace test-case inventories with function-named adversarial strategies. Refined Child.ingest/IngestBytes, Console.HostedIngestSnapshot, projection and request/CLI strategies with independent oracles and mechanical guards. No scope/mechanism change; no implementation started before gate admission.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is flagged stale by estimate-source, so provisional. This estimates the admitted M1 scope only; M2 will be derived after live measurements settle its mechanism, before its implementation gate. It is not a whole-issue optimization estimate.

Decomposition: issue-spec/audit uses design 0.6 ×0.5 (initial mixed-density dispatch) and impl 0.3 ×0.4. Three smaller-go-module extensions are counter, Console snapshot, and resources transport/projection/rendering: each design 0.2 ×0.2 and impl 0.5 ×0.4. Atlas docs use design 0.1 ×0.2 and impl 0.15 ×0.4. Milestone review uses design 0.1 ×0.2 and impl 0.5 ×0.4, including required checks. Existing sync/atomic, fake Child/Console and JSON budget helpers cover the stack; no greenfield/library discovery. Familiarity 1.0, thorough-plan design buffer 15%. Design subtotal 0.46; implementation subtotal 0.98; 0.46×1.15+0.98=1.509 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.12
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 1.509
```
