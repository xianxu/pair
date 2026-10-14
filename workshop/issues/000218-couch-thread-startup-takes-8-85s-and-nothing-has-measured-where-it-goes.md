---
id: 000218
status: working
deps: []
github_issue:
created: 2026-09-08
updated: 2026-10-14
estimate_hours: 5.86
card_mirror: '6bb8facf251995327c3a27ddcdf474eab49ab16b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:00:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch thread startup takes 8.85s and nothing has measured where it goes

## Problem

`#215` raised couch's registration deadline from 5s to 15s against a **measured
8.85s** startup for `pair resume <tag> --layout3`, and made name assignment O(1)
zellij probes. Both were right, and neither explains the 8.85s. A deadline
chosen as 1.7x an unexplained number is a guess with a measurement attached.

The operator's felt symptom, after `#215` unblocked couch: *"startup works now.
I guess it's just slow?"*

### What the number is NOT — measured, so these are closed

| suspect | measured | how |
|---|---|---|
| nvim + pair's `init.lua` | **117 ms** | `nvim --startuptime`, full breakdown; slowest single step is markdown syntax at 3.5ms |
| `zellij list-sessions` | **43 ms** at 26 sessions | wall-clock, 5 runs, 42-45ms |
| one `ProbeSessionName` | **~45 ms**, creates no session | `zellij --session X action list-clients` against absent names |
| name assignment | **8 probes**, flat as the index grows | `#215`, was 52 |
| 6 orphaned zellij servers | **0.0% CPU** each | `ps`; they cost RSS and list-sessions entries, not time |

So roughly 0.5s of the 8.85s is accounted for. **~8.3s is unexplained.**

### What is known about the shape

- **The two waits are different.** The cold path (`awaitThreadRegistration`)
  polls the thread-claim file. The resume path (`awaitResumeRegistration`) waits
  for the zellij **session to be live** — so only resume pays for zellij bring-up,
  layout, three panes, nvim and the agent. The claim file for one long-lived
  thread was last written four days before it was next resumed, which is how we
  know the resume path never touches it.
- `pair` spawns `zellij` from **eight** call sites (`osruntime.go` x6,
  `session_quiescence.go` x2). Nobody has counted how many times a startup
  actually calls it.

## Spec

**Explain the 8.85s with a measurement, then decide what to fix — in that order.**
`#215` was first filed against a root cause that measurement refuted, and its
fix then shipped a 7x regression on the resume path that only an operator
noticed. This issue does not get to guess.

**1. Run the instrument that already exists.** `probes/zellijcalls/` traces every
zellij subprocess a startup spawns, via a PATH shim rather than instrumented call
sites — pair spawns zellij from eight places, so instrumenting "the seam" means
instrumenting eight and missing the ninth. Built in `#215`, never run against a
real couch thread start.

```
eval "$(probes/zellijcalls/trace.sh arm)"
couch                      # start a thread, then quit
probes/zellijcalls/trace.sh report
```

The report's own shape answers the first question: **`in-zellij time` close to
the wall span means the call count is the problem; well under it means the time
is NOT in these subprocesses** and the next instrument goes inside pair.

**2. Only if the time is not in zellij, add stage timing to pair's startup.**
Env-gated, off by default, emitting monotonic marks around the phases between
process start and registration. Do not build this before step 1 says it is
needed — that is the guess this issue exists to avoid.

**3. Decide with the number in hand.** 8.85s may be irreducible (an agent process
genuinely takes seconds to start) or may be one dumb loop. The fix, if any, is
out of scope until the breakdown exists. If it IS irreducible, say so in the
`pairRegistrationTimeout` comment, which currently reasons about a bounded
envelope without knowing what fills it.

## Done when

- A recorded breakdown of the 8.85s exists, with each stage named and timed, and
  it accounts for the wall time rather than a fraction of it.
- `pairRegistrationTimeout`'s comment cites that breakdown instead of a bare 1.7x
  multiple.
- Whatever the breakdown shows is either fixed, or recorded as irreducible with
  the measurement that says so.

## Plan

- [ ] Run `probes/zellijcalls/trace.sh` against a real couch thread start (cold)
      and a relaunch; record both reports in `## Log`.
- [ ] Branch on the result: call count -> reduce calls; otherwise add env-gated
      stage timing inside pair's startup and re-measure.
- [ ] Attribute the wall time stage by stage until the residual is small enough
      to name.
- [ ] Fix what the breakdown indicts, or record irreducibility.
- [ ] Update `pairRegistrationTimeout`'s comment to cite the breakdown.

- [ ] M1 — baseline attach/detach timings and native candidate/fencing foundation ([durable plan](../plans/000218-startup-detach-observation-plan.md)).
- [ ] M2 — shared pass capture, 6/20/40 experiment and measured tuning.

## Log

### 2026-09-08

Split out of `#215` at the operator's direction: *"#215 is about unblock so that
I can continue to use (and thus dogfood) couch."* `#215` achieved that — cold
start and relaunch both work — so latency is its own question rather than a
reason to hold that issue open.

The measurements in `## Problem` were taken while closing `#215` and are the
reason this issue is narrow: four plausible suspects are already dead, which is
worth more than an open-ended "make startup faster".

### 2026-10-14 — planning trace

Claimed by pair:5, started from main 9a2aff687 after slot substrate refresh. Operator approved one snapshot per pass; durable implementation plan: [startup/detach observation](../plans/000218-startup-detach-observation-plan.md). Read-only path trace shows two full native snapshots per healthy SessionOwnerProbe, repeated by Couch prelaunch/registration checks and four child Pair checks (approximately twelve scans for healthy single-poll attach), and approximately six on healthy detach. These are path counts, not measurements. Baseline real attach/detach timing and existing zellij PATH trace precede behavior edits. ARCH-DRY reuse ServerStates, ARCH-ORDER exact fresh fences, ARCH-CONSTRAINTS measure 6/20/40 after reducing duplicate observation.

- Baseline instrument first run: existing TestSessionDetachLive failed before its detach assertion because full ZellijSource.Snapshot queried an unrelated host session (📁1-235) and list-clients returned exit1. The controlled fixture was cleaned up. This is not a production detach timing; the complete targeted owned-fleet harness is still required.

- Admission: plan-quality cleared all PQ-1/PQ-2/PQ-3, estimate-quality INFO accepted, full flow admitted. The 0.80h spec item describes already-incurred shaping, not remaining user decisions. Measurement integrations carry harness/adaptation and experiment work; real-api-discovery covers unverified real Zellij client/registration/retirement behavior and PATH-shim overhead reconciliation. Console item covers pass scope/reducer wiring; parent refactor covers checker/Leave/registration, avoiding duplicate allocation. Calibration is provisional; record any measured overrun honestly.
- Initial shim report (unchanged main behavior): six calls, 0.67s wall span, 0.16s aggregate zellij duration (24%), but this run failed before real detach because of an unrelated host row. Three list-sessions calls averaged25.1ms, failing unrelated list-clients25.1ms; fixture setup/cleanup delete calls were included. No attach/detach latency conclusion is inferred from this incomplete sample.

- Component-only baseline on unchanged production (opt-in test harness): native Snapshot rows1412, observations1097, native argv calls1396, identity calls1097; table1.3185ms, argv26.796274ms, identity39.010451ms, wall67.48775ms, residual0.362525ms. Real owner Probe144.697375ms and Revalidate150.190416ms; three SessionServers reads48.956875/67.473917/150.18375ms, socket0.011166ms, pane query27.94325ms. These measurements explain broad-reader cost, not complete attach wall time. Command: PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1 go test ./cmd/internal/procutil -run '^TestStartupNativeSnapshotTimingLive$' -count=1 -v; launcher equivalent TestStartupOwnerTimingLive (final name checked with harness).
- Measurement instrument defect: PATH shim left itself and the native --server child as exact claimants. Deterministic process regression was red (different parent/native PIDs plus server lifetime trace row); direct syscall.Exec passthrough now passes argv/env/PID/stdin/status preservation and client-timing positive control. Probe package/race and Linux cross-build pass. Rebuilt /tmp/pair218-zellij-shim/zellij. No native discovery behavior has changed. Full baseline fixture is rerunning after correcting macOS /tmp physical-root identity; parent return without a client first frame was explicitly rejected as incomplete evidence.


- Complete controlled baseline at d1902f90 (unchanged launcher/procutil): TestStartupManagedAttachDetachTimingLive passed with frozen Pair/helper in /tmp/pair218-baseline-bin. No shim: parent Resume1.130285083s, four owner probes527.621958ms, parent residual602.663125ms; attached client readiness495.562584ms after parent return, startup_total1.638065166s. Production Detach259.071ms: owner before88.174375ms and after80.372041ms, native process calls54.250µs, residual90.470334ms. Owned server remained detached, incarnation retired, unrelated sentinel survived. Command: PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1 PAIR_STARTUP_PERF_BIN_DIR=/tmp/pair218-baseline-bin /tmp/pair218-couchcore-baseline.test -test.run '^TestStartupManagedAttachDetachTimingLive$' -test.count=1 -test.v.
- Separate instrumented successful baseline: parent2.081159125s (owner935.9585ms, residual1.145200625s), readiness1.888310458s later, total3.969583417s; Detach1.176982334s (owner699.485292ms+314.022083ms, residual163.40396ms). Trace /tmp/pair218-baseline-final-zellij.trace includes fixture/readiness calls: list-sessions21/1.409789s, list-clients8/642.044ms, list-panes7/657.372ms, other3/176.755ms. Internal server exec intentionally excluded; blocking attach lifetime row missing. These totals overlap wall time and are not exclusive attribution. Single runs under changing host load cannot assign the entire instrumented/no-shim delta to overhead.
- Three opt-in timing harnesses pass ordinary-suite skip checks; preserved baseline test binaries in /tmp/pair218-{procutil,launcher,couchcore}-baseline.test. Baseline capture is complete enough to begin candidate-reader implementation; parent/child residual attribution remains outstanding. Reuse existing launchSteps on success to name parent stages, preserving an independent baseline executable for subsequent comparisons. No 21-session experiment has run yet.


- Candidate-reader red/green: before selection argv reads were [1 2 3] instead of [2 3]. SnapshotNamed now shares the full-reader implementation, enriches zellij/unknown metadata candidates, and retains gone/recycled/zombie omission, permission uncertainty and cancellation. Full procutil and targeted race, Linux amd64 cross-build, native self inspection and metadata fuzz (195165 executions) pass. Native same-run comparison: full rows1374, argv1356, identities1058, wall75.307167ms; named rows1375, argv48/identity48, wall2.393042ms. This is a discovery measurement, not end-to-end performance evidence.
- Both launcher discovery consumers use SnapshotNamed. Native helper fixtures now copy the test executable as zellij, proving native metadata selection independently of argv authority. Native server/owner tests pass under host Seatbelt with /bin/ps exec explicitly denied. New competing-claimant regression preserves fresh exclusivity around pane IO and before effects while the original generation remains unchanged. Existing #553 gone/error semantics retained.


- Attributed unchanged baseline: disposable archive886000bba654a714719a4469d4257dc13784c02f plus timing-only overlays; procutil command.go blob1691a9087f660547bc34906ca2d4882586aff9b0 unchanged. No shim parent1.649786167s: claim829.194334ms, checkout0.027208ms, prepare288.964625ms, spawn7.247083ms, record+baseline123.67225ms, ack0.011375ms, registration200.045459ms, promotion+registry136.264166ms; sum1.5854265s, residual64.359667ms (operation return/bookkeeping outside launchTrackedThread). Nested owner times are included, never added again. Exact selected-session attached proof at2.079286667s is a query/readiness bound, not first-frame timing. Parent registration overlaps child startup and is not evidence the child has attached.
- Child consecutive deltas through blocking handoff total590.241ms: native-entry0.537ms, native-resolution13.876ms, prelude158.946ms, discovery77.315ms, verification192.335ms, retention60.489ms, setup25.608ms, revalidation57.370ms, remainder3.765ms in ancestry/decision/layout/handoff. Clock starts at launcher package init; earlier Go init/native loader and client readiness after handoff remain outer residual. Prelude includes native nvim inventory sweep, separate from session discovery. These phases explain the current warm path; they do not reconstruct September's historical8.85s under a different runtime/load.
- Production detach490.558917ms: claim0.023917ms, read+owner169.650958ms, signal0.041459ms, process-exit10.82875ms, retirement310.002291ms; sum490.547375ms, residual0.011542ms. Retirement includes owner160.710666ms and approximately149ms scoped store reads/writes. Exact owned server survives detached, incarnation retired and unrelated sentinel survives. Separate shim run passed: parent1.784370666s, attached proof1.923128s, child793.554ms, detach464.142916ms; detach residual0.010791ms. Trace /tmp/pair218-attributed-detach-final-zellij.trace finite calls include fixture/readiness: list-sessions13/290.488ms, clients4/113.589ms, panes8/208.701ms. Blocking attach row remains absent after group SIGTERM; no exclusive call-count claim from this aggregate.
- Frozen attributed source /tmp/pair218-attribution.TZ7J81, Pair/helper /tmp/pair218-attributed-baseline-bin, test /tmp/pair218-attributed-baseline-couchcore.test. Command from source cmd/internal/couchcore: TMPDIR=/tmp PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1 PAIR_STARTUP_PERF_BIN_DIR=/tmp/pair218-attributed-baseline-bin /tmp/pair218-attributed-baseline-couchcore.test -test.run '^TestStartupManagedAttachDetachTimingLive$' -test.count=1 -test.v. Bounded PTY prefix sink preserves child marks without changing production output capture. Ordinary suite skip and stage scope/failure/ordering tests pass. Targeted procutil/launcher/couchcore race, full procutil/launcher/artifactpath tests, make -k build, native Seatbelt and Linux cross-build pass. Linux native remains untested; portable fixture/cross-build only. No runner/VM added. Host load averages during measurement varied substantially (17.93/34.84/39.43 at16:37); do not infer precise end-to-end speedup from single runs.

## Revisions

- 2026-10-14 — add (operator via TL): concrete case. At the operator's 14:33 restart, Couch's background attaches spawned 21 zellij clients between 14:33:37 and 14:33:47 (~6 at a time): roughly 2–3 s per attach, against ~10–20 ms for `zellij list-sessions` and an expected ~100–300 ms floor (list, lease, one record write, spawn client, first frame). Detach ("leave couch") is unmeasured but also inspects each slot's session server. Trace one attach and one detach step by step; the likely waste is a whole process-table read (native argv for every process since #547) per slot. Probably one shared snapshot per startup or detach pass instead.

- 2026-10-14 — add (operator): try much higher attach parallelism (operator suggests 40; today `LifecycleParallelism = NumCPU/2` = 6 on a 12-core Mac). Run the same restart at 6 / 20 / 40 with a temporary override; record total time and peak load; keep what the numbers support. If per-slot whole-process-table reads dominate, fix those first (shared snapshot per pass).

- 2026-10-14 — add (implementation plan): M1 baseline plus native candidate/fencing foundation; M2 shared parent pass and remeasurement/concurrency selection. Child Pair cannot reuse Couch memory, so narrowed standalone discovery is included. Preserve #553 and exact-generation effect checks.

- 2026-10-14 — refine (PQ-1/PQ-2/PQ-3): shared evidence is initial only. Fresh after-pane/effect checks retain name exclusivity via candidate-only enumeration; registration bypasses the pass to observe post-launch births. Compact named-function test strategies supersede procedural case inventories in the durable plan.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration marked stale/provisional. Existing sysctl/procfs APIs, ServerStates, owner/fleet worlds, controlled Zellij fixture, console reducer and PATH shim supply the mechanisms; no novel library needed. Design discount 0.2 for resolved implementation choices, familiarity 1.0 from #547. Two API-integration items cover baseline/full controlled lifecycle measurement and the 21-session comparative experiment. Smaller-Go items cover candidate reader and pass object; refactors cover launcher consumers and parent lifecycle scope. TUI item covers pass-context/reducer wiring and ordering tests, not a new screen. Three review primitives cover the two milestone boundaries plus final issue close. Impl values use v3.1's 0.4 scale, design buffer 0.15.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: api-integration design=0.20 impl=0.60
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.04 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: cross-cutting-refactor design=0.16 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: milestone-review design=0.04 impl=0.20
item: milestone-review design=0.04 impl=0.20
item: milestone-review design=0.04 impl=0.20
item: atlas-docs design=0.04 impl=0.08
design-buffer: 0.15
total: 5.86
```

- 2026-10-14 — add: repair nontransparent tracing of internal Zellij server reexec before relying on live timing. Native server discovery remains unchanged until complete baseline evidence.
