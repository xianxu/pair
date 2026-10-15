---
id: 000218
status: codecomplete
deps: []
github_issue:
created: 2026-09-08
updated: 2026-10-14
estimate_hours: 5.86
card_mirror: '3fbe70a802e09dd08f976beccf59135071e99c09' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:00:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 9.40
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

- [x] M1 — baseline attach/detach timings and native candidate/fencing foundation ([durable plan](../plans/000218-startup-detach-observation-plan.md)).
- [x] M2 — shared pass capture, 6/20/40 experiment and measured tuning.

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
- 2026-10-14: closed — Required sdlc-local/suite PASS on b289539f9ce2211ce3b63d2d1bc71d4b5efef4cd (10m33s, logSHA2f571ffec711); subsequent commits Markdown-only. M2 boundary SHIP with zero findings and independent full changed-package/race passes; M1 already SHIP. Native candidate-only discovery, shared initial pass, fresh pane/effect fences, typed identity uncertainty and mutex-free Stop verified. Partial fleet rows and claim/prepare/durable-record contention retained without complete-benchmark or concurrency claim. Received TL landing scope defers IPC/full6/20/40 to unassigned563 after562, cold560, actual operator timing to TL next restart with528; no staging/default change. no-plan-check acknowledges superseded original cold/full-matrix/tuning Plan rows, not false completion.; review verdict: SHIP
- 2026-10-14: closed M2 — Required sdlc-local/suite PASS on b289539f9ce2211ce3b63d2d1bc71d4b5efef4cd (10m33s, logSHA2f571ffec711); later commits Markdown-only. Race procutil/launcher/diagnosticlog and shutdown/pass tests PASS, full couchtty PASS42.055s, make build PASS. Shared initial evidence with fresh pane/effect fences, identity uncertainty and mutex-free Stop covered. Benchmark remains incomplete with partial rows/contention explicitly retained. Received TL scope defers full balanced9/IPC/concurrency to unassigned563 after562, cold560, operator timing to TL next restart with528; no staging/default change. no-plan-check acknowledges superseded original cold/full-matrix/tuning rows, which remain historically unchecked.; review verdict: SHIP
- 2026-10-14: closed M1 — Recorded sdlc-local/suite PASS on71e3380d773ddda553af77a5933dad1884f2aa42 (12m10s, log e43d9c7c917c); only lessons docs changed since. Targeted race, make -k build, Darwin native and Seatbelt denying /bin/ps pass. Same-run native snapshot1356 argv/75.307ms versus named48/2.393ms. Unoptimized real managed attach and production detach attributed in Log; detach residual11.542us; session/incarnation/sentinel assertions pass. Shared initial pass and21-session6/20/40 experiment remain M2. Linux portable procfs fixtures/cross-build pass; native Linux untested, no runner or VM.; review verdict: SHIP

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


### 2026-10-14 — M1 SHIP and M2 integration

M1 closed SHIP with no findings at82c0eea5782ad527d2fcd077af66ac2e9204bc8a.
Recorded suite passed on71e3380d773ddda553af77a5933dad1884f2aa42 in12m10s,
log sha256e43d9c7c917c. The reviewer independently passed targeted/native/race
checks and a live rerun (attached proof2.29s, detach330ms), after a first handoff
failed with No session found. Investigation reproduced one invalid pane-JSON
response in10 diagnostic attempts; exact same server generation/socket remained.
Subsequent raw-output shared-namespace12/12 and isolated-namespace10/10 passed.
Neither confirms a cause or fix. Preserve all attempts; private socket namespace
is containment, not a demonstrated repair. Evidence /tmp/pair218-handoff-investigate.
Native Zellij0.45.1 discovery filters sockets by ConnStatus response, so kernel
server proof and failed CLI name discovery can coexist; do not weaken authority
or silently retry failed samples.

M2 units prove21 concurrent callers share capture1 with42 live reads (after-pane
plus Revalidate), capture errors do not rescan, sibling cancellation is isolated,
and parent cancellation reaches capture/live IO. Checker contexts are bound to
constructor scope and selected root. Registration-bypass mutation went red with
prelaunch absence polling until its deadline. LeaveDetach scope mutation went red
for all21 slots before wiring; all three per-slot observations now inherit the
one pass and completion cancels it. Standalone Detach remains fresh. Console
scope tests passed20 repetitions, targeted race and production wiring tests;
canceled queued workers now recheck cancellation immediately before dispatcher IO.
Actual native diagnostics are opt-in and preserve the default unwrapped reader.
Cold attribution is now the separately published, unassigned #560.

### 2026-10-14 — private fleet smoke and stopped first matrix

Frozen production binaries at4b3ccc608f47322ce3e4f13d408e41700ce5a05d,
without #559's CPU-loop fix. Smoke1 /tmp/pair218-SMOKE-aw4k478w failed
fixture setup (missing positive ThreadRecord revision); smoke2
/tmp/pair218-SMOKE2-nm6wcyu6 refused the default layout3 versus seeded layout2
before child spawn. Both failures retained, fixture fields corrected without
relaxing production guards, and fixture validation/schedule tests passed.
Smoke3 /tmp/pair218-SMOKE3-xgr7yc1s passed32.72s:21 meaningful terminal-model
first frames plus delivery acknowledgments, actual background peak6, root890.20ms,
all frames15400.59ms, adoption15410.72ms, LeaveDetach4681.67ms. Couch self CPU
startup10078.42ms / leave3357.27ms, sampled peak117%; one host sample8.4%idle.
All21 client kernel births retained; server/agent identities and unrelated
sentinel survived Leave; exact private fixture cleanup left no scoped processes.
This is two-pane persistent stand-in workload, not the operator's three-pane
model sessions, and cannot reconstruct the historical8.85s.

Balanced schedule[6,20,40,20,40,6,40,6,20] began separately at
/tmp/pair218-MATRIX-_15pmlgy (source/binary hashes and source snapshot in
source.json). First three samples completed (limit/all21frame/leave seconds:6/22.740/9.565,
20/17.669/3.969,40/14.222/4.058; actual peaks6/20/20). Fourth(limit20) FAILED during
prepare: possible zellij PID23658, process identity unavailable after command
read. Stopped immediately; no samples5–9, no hidden retries, no concurrency
recommendation from incomplete samples. Exact cleanup verified: private root absent, zero scoped process rows, PID23658
absent. Retained client identities/child prefixes do not identify23658; owned
versus unrelated cannot be reconstructed. Failure cause remains unresolved. Raw errors/child output/native counters/host samples
retained, but private launch diagnostic file was removed by fixture cleanup;
copying diagnostics to the evidence directory must precede future cleanup.
Parent record/fsync timing is measured in the single-attach baseline, but the
private context observer is not yet enabled in the couchcmd fleet harness;
fleet fsync attribution remains an explicit gap. Main now contains #559 PR336;
this matrix does not. Integrate it before final measurements and preserve
pre-fix data rather than blending versions. Targeted TestFleet tests pass and
git diff --check passes for smoke fixture corrections.
Main #559 merged; its checker constructor gains index/owner caches while
preserving #218 ownerScope identity. Targeted integration checks pending.


### 2026-10-14 — resumed identity-contract diagnosis

Retained first matrix remains incomplete and unchanged; PID23658 cannot be
attributed from saved evidence. Local Darwin conformance with an owned reaped
child confirms kern.proc.pid succeeds with zero bytes, while x/sys
SysctlKinfoProc converts that response to EIO. Identity collapses this and all
other errors to empty, so snapshotSelected cannot distinguish confirmed absence
from unreadable metadata after a successful argv read. Deterministic native
TestSnapshotDarwinExitAfterSuccessfulCommand is RED: successful command, child
reaped at the boundary, snapshot retains an unavailable-identity row. This proves
a contract defect, not the historical sample's exact cause. ARCH-ORDER preserves
unknown as unknown; use one error-bearing native identity read, classify only
positive absence/terminal evidence as gone, keep malformed/permission/EIO unknown.
No retries, sleeps or effect-fence relaxation. Fleet diagnostics retention and
existing context observer exposure delegated as bounded independent work.

Native contract fix is GREEN: reaped-after-successful-command regression, raw
Darwin absence/error table, generation/error matrix, native zombies and full
procutil pass; race procutil/launcher owner tests pass and Linux cross-build
passes (native Linux unavailable). Fresh ad-hoc review caught two gaps, both
resolved: preserve public Identity/StrictIdentity unreaped tokens, and assert
native argv succeeded before exit. Darwin length-aware SysctlRaw adds a size
query per lookup; native same-run full1371rows/1352argv/1052identity134.167ms,
named1372rows/47argv/47identity3.820ms. These API invocation counters are not
kernel syscall counts. Host load and different timestamps prevent direct
comparison to M1's75ms baseline. Existing parent timing seam now exposed to
fleet contexts, with record+baseline explicitly aggregate durability rather
than isolated fsync. Private regular diagnostics are copied before attempt
cleanup and final fixture cleanup. Artifactpath dead-symbol check found the
cross-package diagnostic-only API; a narrow documented allowlist entry retains
it. No new production flags or lifecycle authority were introduced.

### 2026-10-14 — second experiment setup refusal retained

make -k build passed; frozen source9d5b3ca89b4c888cf3387fc1c91c454e429c7083,
including #559. Separate MATRIX2 /tmp/pair218-MATRIX2.6f1ANH FAILED setup
before sample1 (4.08s), no timing data. Native Zellij rejected the fixture session
name with a zero-character limit. Manifest proves inherited ZELLIJ_SOCKET_DIR
still pointed to the restarted Couch's long /private/var/folders runtime path;
TMPDIR alone did not isolate this explicit override. No authority change or
retries. Complete source archive/binary hashes, failed raw output and diagnostics
retained. Future fleet setup must explicitly bind Zellij socket and XDG runtime
paths inside the short private root before both fixtures and terminal environment
construction. Regression tests must pin inherited environment isolation. This
is fixture containment, not an explanation of the old identity failure.

### 2026-10-14 — third matrix first-sample disagreement retained

MATRIX3 /tmp/pair218-MATRIX3.WMUt1t froze exact source/binaries at
e97653bafc6cad12069ebacdc144182de0bb8d29, including #559 and private runtime
isolation. Failed after sample1/limit6; no samples2–9 and no retries. All21
meaningful frames23.618s, adoption23.628s, root1.663s, exactly one kernel client
per21 tags, Couch CPU12.228s startup/3.655s leave, actual background peak6.
Subsequent SnapshotSessionsContext said session pair-f21-edcfe72b117a-03 was
not attached. Error omitted its precise state/raw query response; no retained
proof of whether parser/query or real disconnect caused the disagreement.
Production Leave completed5.787s; exact server/agent and unrelated sentinel
survival checks passed. Source/hash manifest, complete sample, native counts,
host logs and pre-cleanup fixture diagnostics retained. Parent stage medians
per background operation: claim1842.8ms, prepare861.3ms, record+baseline462.7ms,
registration703.7ms, promotion+registry509.5ms, spawn8.9ms; overlapping workers
must not be summed into wall. Durability-containing stage is material; no
isolated fsync claim. Investigation continues from evidence, not filtering the
failed sample. Native ownership tests also passed under Seatbelt denying /bin/ps.

Query evidence seam implemented and validated without a wrapper process or
extra query: context observer copies bounded64KiB streams/status/timestamps,
while parser receives original bytes. Stateful response tests preserve empty,
header-only, attached and nonzero classifications with exactly four client
queries; mutation/cap/isolation/cancellation tests pass. Fleet now retains actual
attached/postleave rows and one fresh native client snapshot after queries;
missing/duplicate/unreadable client evidence fails measurement. Focused race
launcher/couchcmd and full artifactpath guards pass; independent focused
launcher/couchcmd/procutil rerun passes. Timeout comment now cites measured warm
phases while retaining #489's hang ceiling and #560's cold scope. Host evidence
from failed MATRIX3: sampled Couch peak84.3%, host minidle0%, maxload1 29.31;
startup129parent named snapshots10144argv/10120identity +105child snapshots
(21full/84named)37140argv/30870identity; Leave75named6001argv/5986identity.
These are actual reader API calls; elapsed reader sums overlap workers and child
phases. Request9d15edcb-cd1a-4886-bf25-6620ff2a969f submitted to hub asks whether
a future fixed schedule may continue only after verified safe resets while
retaining failure rows; until answered, stop-on-any-error remains unchanged.

### 2026-10-14 — fourth matrix stopped during guarded Leave

MATRIX4 /tmp/pair218-MATRIX4.8L6fzd froze source/binaries
7dfd7e6315dd67864be6be41c0c78e9e16522a2c. Rows1–6 passed: limits6/20/40/20/40/6,
all21 first frames13.259/11.821/11.702/12.069/11.016/12.885s; successful
Leave3.775/3.074/3.396/3.335/3.057/3.569s. Row7/40 rendered all21 in11.537s
and verified all21 exact client generations after raw attached queries, but
production Leave REFUSED invalid live pane JSON for session...-17/tag...0012.
Its2.951s Leave is partial, not successful detach. Stop-on-error retained;
rows8/9 never ran, no retries or filtered failure. Total run133.42s FAILED.
Raw attached/postleave queries are now retained, but observer context was only
on verification queries, not authority queries inside root/background/Leave;
therefore malformed owner-pane response bytes still missing. Extend the SAME
observer to all parent operation contexts with a synchronized bounded collector
before further experiments. This is an instrumentation coverage class, not a
JSON repair or permission to relax owner proof. Exact fixture root removed;
no operator environment touched. Hub outcome934a9dea-b86b-4e72-a088-03f848d7b7c3
submitted; a failed Leave cannot qualify for proposed safe-reset continuation.

Authority-query observer coverage is now complete for root StartInteractive,
dispatched background operations/Leave and verification, sharing a mutex-protected
collector. It snapshots after Console worker join;4096 records/8MiB streams per
sample, any omitted/truncated evidence explicitly fails measurement. Race
couchcmd/launcher and full artifactpath pass. A production SessionOwnerProbe test
captures exact malformed-success pane bytes without changing its unresolved
verdict. Diagnostic process inspection confirms zero remaining scoped process
rows for MATRIX2/3/4 roots, beyond root-file absence alone.

Read-only upstream0.45.1 investigation found a possible empty-success mechanism:
[CLI individual_messages_client](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-client/src/cli_client.rs#L206-L252)
accepts an unblock before query Log; [server broadcasts some unblocks](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1287)
to all connected clients, including CLI. ClosePane/CloseTab/plugin reload can
generate those broadcasts. ListClients also has a one-second completion wait in
[route.rs](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/route.rs#L49),
so delayed output can lose to its own unblock. ListPanes normally emits its Log
before its own unblock. This supports empty-success as a hypothesis only; it
cannot explain nonempty malformed bytes until the actual failed query is retained.
No upstream binary change or installation proposed. Hub receipt
b3e5ad8c-b2c7-4e11-a766-2ea84a5b22a2 reports this distinction.

### 2026-10-14 — fifth matrix captured inconsistent live discovery

MATRIX5 /tmp/pair218-MATRIX5.vHTGIP source/binaries
9c4ff8ca0ab8953a1cde5fa42dc6895e50b2c9a7 FAILED sample1/6 warm selection
(resume-not-detached),16.48s test. Only10/21 frames;5.330s is the last PARTIAL
frame, not fleet readiness. No Leave timing.155 parent queries retained without
truncation. Failed tag...0010/session...-15: successful --short at20:36:27.400
returned20 names excluding target; immediately following --no-formatting at
20:36:27.450 returned21 including target. Exact bytes/times/status retained in
listing-disagreement.json. No malformed or empty successful authority-query
stdout preceded the failure; subsequent context-canceled queries belong to
teardown. This does NOT prove the earlier malformed JSON cause. Exact fixture
root absent and diagnostic process inspection found zero scoped rows.

Upstream [assert_socket](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L145-L170)
reads one ConnStatus response, excludes any first message other than Connected,
and listing can still exit0 with remaining sessions. This is a concrete mechanism
for silent omission, not proof of the runtime message ordering. Explicit
[--session actions also enumerate globally](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L404-L451),
so bypassing our two listing calls does not remove upstream discovery. Parent warm
resume currently performs four explicit global listings (DetachedSessions twice),
and child prelude/decision performs additional global listings. Shared initial
native evidence does not eliminate this distinct O(fleet-squared) RPC work.
No further experiment without a diagnosed fix or approved protocol/scope change;
all incomplete matrices remain retained and concurrency default unchanged.

Fresh follow-through review found retained identity EINVAL could be dropped by
downstream IsProcessGone (launcher serverProcesses and diagnosticlog proof).
Correcting error provenance centrally while preserving raw command EINVAL exit
semantics, with regression tests through both consumers. Last scope-report send
failed before acceptance: Couch cannot find live caller/recipient registration.
No receipt/no delivery claimed; no operator Couch restart or alternate channel.

Identity-error follow-through fixed: typed IdentityUnavailableError retains
underlying errno but IsProcessGone refuses to turn its uncertainty into absence.
Snapshot identity-EINVAL classifier regression demonstrated RED first; launcher
server and diagnostic writer-proof regressions pass. Raw command/environment
EINVAL/ESRCH/ENOENT still mean gone. All classifier consumers enumerated; race
procutil/launcher/diagnosticlog PASS4.258/2.846/3.832s. No effects or guard
relaxation. Preparing the pushed candidate's required full local suite; M2 and
issue remain open because no complete balanced9 or operator live check exists.

Darwin absence-proof review found the earlier inference too strong: an absent
child returns an empty successful lookup, but XNU iteration can also omit
SIDL/shadow processes. The empty-result path now requires exactly one independent
signal-zero probe returning ESRCH; all other outcomes remain uncertainty. Raw
lookup errors and normal identities do not probe. Deterministic regression was
observed RED before the correction. The queued suite for3872de336 never started
tests and was canceled (exit130) to verify the corrected pushed head instead.

Correction verification: race procutil/launcher/diagnosticlog selected identity,
snapshot, gone, server and writer-proof tests PASS4.601/3.825/1.177s; diff check
clean. Hub scope report attempted after actors became visible, but transport
returned uncertain receipt e138b787-e4d2-4ca9-96d8-acb8583fb37d; status lookup
refused because no connected recipient retains it. Delivery is unproven; no
blind resend or approval inferred.

Corrected candidate fd4a2580dd4a143493e64f0f1a194ce06fd3f6c7 is pushed;
make -k build PASS. Required sdlc-local/suite is queued for that exact commit
(session55962), no full tests started at this checkpoint.

Read-only protocol synthesis recommends an isolated, separately authorized
passive IPC diagnostic on the unchanged workload: associate socket paths and
complete protobuf reads with each actual listing subprocess. An omitted owned
session whose first decoded response is UnblockInputThread would establish the
rejection mechanism; connection/read/decode failure would show another path,
and Connected would contradict this hypothesis. Trace pane-close timestamps
separately to attribute the source of an unblock. A run without omission is
inconclusive, with no automatic retry. Plain syscall timing without received
frame bytes is insufficient; tracing may change timing. No tracer is installed
or run here. Switching listing format, combining inventories, or explicitly
naming an action cannot eliminate upstream discovery. This diagnostic/scope
choice remains for the hub; no approval inferred.

Full local suite on fd4a2580 FAILED: exit2 after15m24s execution and1201s
queue wait, logSHA d0dc10ab0d77 (full retained sdlc local-check log).
Only failing package couchtty: TestConsoleStopDuringAttachRefusesPartialPaneAndKeepsTerminalRestored
timed out at10min. All other Go packages completed, including launcher, procutil,
couchcore and diagnosticlog. Targeted8s rerun reproduces the same self-deadlock:
existing test holds c.mu while calling Stop; M2 added synchronous c.mu acquisition
to release pass fields. Lifetime cancellation already synchronously cancels
production pass/capture/worker descendants; queued workers check cancellation
before dispatcher IO. Move explicit pass cleanup into teardown's existing state
mutex barrier, invoke its callback outside the mutex before joining workers.
Strengthen scoped-pass Stop test to hold c.mu across Stop, assert immediate
context cancellation, then await teardown for exactly-once release/cleared fields.
Read-only peer review confirmed the cancellation graph and correction.

Shutdown correction GREEN: selected attach-held-mutex, scoped Stop, shared-pass,
canceled-queue and in-flight-stop tests PASS under race (2.185s); make -k build
PASS. Both existing attach regression and strengthened scoped-pass test were
observed RED before the correction. Stop synchronously cancels context
descendants without c.mu; explicit cleanup runs outside the teardown mutex
before workers.Wait. No failed-suite result is being treated as a pass; rerun
required on the corrected pushed head.

Corrected shutdown candidate b289539f9ce2211ce3b63d2d1bc71d4b5efef4cd pushed.
Complete targeted couchtty package PASS42.055s (all tests, 3min bound), in
addition to selected race PASS2.185s and rebuild PASS. Required full suite
session91014 is queued for this exact candidate; no passing status claimed.
Hub outcome report 08575386-7a92-4f70-8e6c-38c3af0525d0 confirmed submitted,
including the fixed failure, incomplete benchmark, proposed isolated diagnostic
and need for scope/live-check coordination. Awaiting answer; no scope approval.

### 2026-10-14 — received TL disposition and follow-up publication

Received TL scope instruction in this conversation, matched against canonical
receipt bd68b37f-cf8d-4d9b-bc88-df5509979c1e (ops:0, current pair:5 binding,
exact body). Transport status is indeterminate because rendered turn-start
confirmation was missed; do not claim a submitted receipt. The instruction
actually received here directs landing M1 candidate-only enrichment and M2
shared initial pass after corrected suite and close review, with benchmark
incomplete and no concurrency-default change. Operator live attach/detach timing
is deferred to the TL's next Couch restart batched with #528; do not stage :0.

Filed, released and published unassigned #562 (Zellij reply race, CI evidence
and internal upstream draft) and #563 (passive IPC frames plus balanced6/20/40,
deps562). Both conform to issue schema and remain open/unassigned. CI Atoi
input was an empty source-agent.pid after an existence-only wait, not captured
CLI stdout; its cause is independent/unproven. Historical cold #560 unchanged.
Pair:2 notification038d6f4e-01ac-452f-b3aa-380febc619f6 timed out and receipt
lookup refused; no delivery claimed or automatic resend. Hub report of the
filings4b9930c1-1d64-4214-8606-f35422db66d5 likewise remains unconfirmed.

Partial performance evidence (not a balanced comparison): MATRIX4 source7dfd7e6
completed6 rows at limits6/20/40/20/40/6 with all21 first frames in
13.259/11.821/11.702/12.069/11.016/12.885s and Leave in
3.775/3.074/3.396/3.335/3.057/3.569s. Row7 had21 frames11.537s but Leave
refused invalid live pane JSON; rows8/9 never ran. MATRIX5 source9c4ff8c failed
row1 before complete readiness due inconsistent successful discovery; its
5.330s is only the last partial frame, not fleet readiness. MATRIX3 sourcee976
had21 frames23.618s but later attached proof disagreed. Keep versions separate.

Contention attribution: M1 single warm parent1.650s included claim829ms,
prepare289ms and record/baseline124ms. Under MATRIX3's21-session load, background
per-operation medians were claim1842.8ms, prepare861.3ms and record+baseline462.7ms
(registration703.7ms, promotion509.5ms). These observed intervals include queueing,
CPU and nested work; they overlap across slots and cannot be summed into fleet
wall time or interpreted as isolated lock/fsync costs. Durable record+baseline
includes durability but does not isolate fsync. Host minidle0% and load1 up to29.31
show co-tenant contention. Candidate-only scans and shared initial evidence
remove repeated process enrichment, while distinct global CLI discovery remains.
No reliable concurrency recommendation follows from these incomplete rows.

Corrected suite still queued on b289539f; all subsequent changes, including
merging main's target/follow-up details, are Markdown-only. No review or close
claimed yet.

Corrected full local suite PASS on b289539f9ce2211ce3b63d2d1bc71d4b5efef4cd:
sdlc-local/suite recorded success, exit0 in10m33s after2620s queue wait,
logSHA256 prefix2f571ffec711, retained under the common Git sdlc/local-checks
commit directory. Full couchcore294.995s, couchtty39.722s, launcher24.886s,
procutil0.348s and diagnosticlog73.085s passed, as did the remaining suite.
Original failed fd4a suite remains retained. Later commits are Markdown-only.
M2 dry-run passed mechanical guards and measured actuals without dispatch or
file changes; actual close will adopt the current measurement. Original Plan
rows superseded by received TL scope and follow-ups stay unchecked; use precise
no-plan-check acknowledgment rather than falsely ticking deferred work.

## Revisions

- 2026-10-14 — received TL final scope disposition (canonical bd68b37f body, transport rendering confirmation indeterminate): #218 acceptance is native candidate-only enrichment plus shared initial owner pass, preserved fresh effect fences, corrected suite and close review, and honest partial performance/contended-phase records. Full balanced9, passive IPC diagnostic and measured concurrency selection move to unassigned #563 after #562; cold stays #560. Operator live check is TL-owned at next restart with #528, with no :0 staging here. Supersedes original cold/full-matrix/tuning Plan rows and Done-when requirements; those remain unchecked as historical work, not falsely completed. M2 close covers built shared pass and retained incomplete measurement; the default is unchanged.

- 2026-10-14 — repair M2 shutdown lock regression found by full suite: preserve mutex-free Stop; release pass fields/callback in joined teardown. No lifecycle authority or retry changes.

- 2026-10-14 — correct the earlier empty-Darwin-lookup inference: only an independent kernel ESRCH establishes absence; retain uncertainty otherwise. No retry or effect-authority relaxation.

- 2026-10-14 — add (operator via TL): concrete case. At the operator's 14:33 restart, Couch's background attaches spawned 21 zellij clients between 14:33:37 and 14:33:47 (~6 at a time): roughly 2–3 s per attach, against ~10–20 ms for `zellij list-sessions` and an expected ~100–300 ms floor (list, lease, one record write, spawn client, first frame). Detach ("leave couch") is unmeasured but also inspects each slot's session server. Trace one attach and one detach step by step; the likely waste is a whole process-table read (native argv for every process since #547) per slot. Probably one shared snapshot per startup or detach pass instead.

- 2026-10-14 — add (operator): try much higher attach parallelism (operator suggests 40; today `LifecycleParallelism = NumCPU/2` = 6 on a 12-core Mac). Run the same restart at 6 / 20 / 40 with a temporary override; record total time and peak load; keep what the numbers support. If per-slot whole-process-table reads dominate, fix those first (shared snapshot per pass).

- 2026-10-14 — add (implementation plan): M1 baseline plus native candidate/fencing foundation; M2 shared parent pass and remeasurement/concurrency selection. Child Pair cannot reuse Couch memory, so narrowed standalone discovery is included. Preserve #553 and exact-generation effect checks.

- 2026-10-14 — refine (PQ-1/PQ-2/PQ-3): shared evidence is initial only. Fresh after-pane/effect checks retain name exclusivity via candidate-only enumeration; registration bypasses the pass to observe post-launch births. Compact named-function test strategies supersede procedural case inventories in the durable plan.


- 2026-10-14 — TL scope clarification (receipt e6621a25-d14b-4826-8a8e-edcccd2e2e4a): this issue covers current warm restart attach, detach and the6/20/40 fleet. This supersedes the original cold-start/historical8.85s checklist requirement; cold attribution is unassigned #560. Warm acceptance requires meaningful client first frame, lease/record durability and contention attribution, including Couch CPU and co-tenant load. The approximate twelve75ms scans could explain0.9s, not all of the observed2–3s; do not blame the full interval on scans.

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
