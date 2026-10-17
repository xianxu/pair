# Hosted terminal ingest diagnostics — implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox syntax. This plan admits M1 only; M2 design waits for the measurements and TL approval.

**Goal:** Expose exact cumulative bytes arriving at each hosted terminal without changing parser behavior, then use live measurements to choose #622's optimization.

**Architecture:** Each ptychild.Child owns one atomic byte counter for its lifetime. The existing Console snapshots counters and stable client/thread/process identities; the resources request returns a bounded fresh diagnostic section alongside its existing cached process sample. No new worker, timer, parser or retained output is introduced.

**Tech Stack:** Go, sync/atomic, existing ptychild/Console/resources protocol and native CPU sample tools.

## Scope and approval

TL dispatch 255c314d requires measurements and a plan before optimization. Subsequent direction 9929f33e requests M1 counters first, landing before the combined #618/#617 restart. This is the narrow M1 plan; it does not pre-approve a parser policy. The issue remains working after M1 and does not claim its whole-issue CPU-reduction Done-when. Milestone review and a partial M1 landing follow the explicit TL direction; continuation uses a fresh branch name after that PR merges.

Per-feed wall timing is deliberately absent: lock waits, Go scheduling and output publication are not emulator CPU. Accurate per-thread CPU on Darwin requires thread pinning/native calls on the hot path; do not add that cost just to measure it. Pair the byte rates with server CPU/sample and clearly label attribution limits.

## Core concepts

| Name | Kind | File | Status |
|---|---|---|---|
| Hosted ingest snapshot records | PURE | cmd/internal/couchcore/resource.go | new |
| Child ingress counter | INTEGRATION | cmd/internal/ptychild/child.go, terminal.go | modified |
| Console hosted-ingest snapshot | INTEGRATION | cmd/internal/couchtty/console_resources.go | modified |
| Resources wire projection | PURE | cmd/internal/couchcmd/resources_projection.go | modified |
| Resources handler | INTEGRATION | cmd/internal/couchcmd/message_service.go | modified |
| Human-readable resource rendering | INTEGRATION | cmd/internal/couchcmd/resources_render.go | modified |

Counter ownership is 1:1 with Child. It counts len(p) at ingestion admission, including bytes whose later parsing/publication fails; it is not output acknowledged by the parent or bytes from the agent wrapper's separate scrollback. Empty input adds zero. Atomic reads do not acquire Endpoint's parser lock. The counter dies with Child and is reset only by a new client incarnation. There is no history map of departed counters.

The diagnostic snapshot names server PID/start time, its own sampled-at time, and endpoint ID + thread address + hosted process PID/generation per row. Only compare a row across snapshots when server and client identities match; a disappearing/replaced client is not a zero-byte interval. Keep the timestamp distinct from resources' existing cached sample timestamp. Unknown/older binaries must be distinguishable from a confirmed empty supported snapshot.

## Consumer audit

| Consumer | Authority | Source |
|---|---|---|
| Selected frame/history | hosted Endpoint | couchtty/terminal.go; terminal/presenter.go |
| Input/mouse modes and geometry | hosted Endpoint | terminal/endpoint.go, presenter.go |
| Off-screen protocol replies | hosted VT + sole FIFO input writer | ptychild/terminal.go; terminal/profile_query.go |
| BEL/notification/clipboard/title effects | hosted parser effects + policy | terminal/endpoint.go; couchtty/console.go |
| Resize/EOF/synchronized publication | ordered hosted Endpoint | ptychild/child.go; terminal/endpoint.go |
| Waiting/picker notices | wrapper registry status + tail | couchcmd/wait_notice.go, message_service.go |
| Peek and agent drive | wrapper endpoint observation/tail | couchcmd/peek_tail.go, agent_drive_service.go |
| Resource idle | wrapper status + broker | couchcmd/resources_inventory.go |
| Idle fading | shared thread activity | couchtty/console_activity.go |
| Broadcast/notes | acknowledged selected parent frame | terminal/tap.go; couchtty/console_broadcast.go |

## M1 operating contract

- One atomic Add per incoming chunk; one atomic Load per reported client. No time.Now or native calls inside ingest.
- O(hosted clients) snapshot under the existing Console mutex; no parser work, filesystem IO, subprocess or retained client references escapes that snapshot.
- Reuse the existing hosted/slot census bound (256 rows); identify omitted rows explicitly. Stable ordering follows Console's ordered hosted inventory.
- Reuse resourceDetailPrefix and MaxFrameBytes for the diagnostic wire section. Extend projection budgets so optional process/episode detail cannot crowd it out; any truncation states omitted rows and never masquerades as zero.
- Read-only request obtains fresh counter data, leaving process-resource cache semantics intact. No periodic counter worker, file persistence, alert state or per-ingest callback.
- Text output names cumulative hosted bytes; JSON distinguishes byte totals from rates. Rates are computed from identity-matched snapshots by the measurement procedure.
- No terminal policy, background waiting semantics, key encoding, delivery admission or rendering authority changes in M1.

## M1 tasks

### 1. Pin counter semantics

Files: modify cmd/internal/ptychild/child.go and terminal.go; tests in existing ptychild test files.

- [x] Child.ingest / proposed Child.IngestBytes: generate byte streams and chunk partitions through production fake Child Feed, comparing totals to an independently summed admitted-byte oracle. Concurrent sampling must be monotone and end exact; vary ring capacity and publication failure to guard against deriving totals from retained/published bytes.
- [x] Add the one Child-owned atomic.Uint64 and accessor, incrementing at ingest admission. Run focused normal/race tests; prove the count does not derive from ring length or snapshots.

### 2. Expose a bounded fresh resources section

Files: couchcore/resource.go; couchtty/console_resources.go; couchcmd/message_service.go, resources_projection.go, resources_render.go; colocated tests.

- [x] Proposed Console.HostedIngestSnapshot: drive controlled child replacement/removal and event order through the production fake Console, checking incarnation changes and snapshot isolation against the admitted child inventory. Hold an endpoint parser lock while observing to mechanically guard against parser-lock acquisition.
- [x] messageService.handle plus runResourcesCLI/renderResources: use production fake Child/Console and socket/CLI transport to compare fresh byte totals against the deliberately stale cached resource sample; assert JSON identity/freshness and sanitized cumulative text output.
- [x] projectResourceView/resourceDetailPrefix: generate oversized census/labels and competing optional detail, then encode the actual socket response to enforce MaxFrameBytes, explicit omission accounting and source snapshot isolation. Compare retained identities to the original inventory.
- [x] Run affected package/race/inventory checks. Benchmark counter overhead at the existing Child ingest seam without inventing CPU-time fields.

### 3. Review and land the diagnostic milestone

Files: README.md, atlas/couch.md; issue Log and this plan.

- [x] Document fields, counter reset/identity semantics, read-only freshness, omission behavior and the measurement recipe. Keep atlas/index links current.
- [x] Commit/push, run the required recorded suite on the pushed head, and milestone-close M1 with measured actuals and --no-project because TL owns project records. Binary owns fresh-context review.
- [x] Land only M1 through normal PR/merge gates as explicitly directed; keep #622 working and M2 unchecked. Report PR/build to TL for the combined restart. No worker-initiated :0 stage/restart.

## Measurement after M1 is live

1. Verify actual running server identity/build and that every expected hosted client appears. Preserve census/omission/unavailable status before comparing counters.
2. Take matched resource snapshots across natural 60-second idle and busy intervals; record endpoint identity, server/client incarnation, byte delta, interval duration and wrapper idle/working state. A partial or reset interval is labelled unusable, never zero.
3. Bracket the same interval with cumulative server CPU and a 5-second native sample; record host load, concurrent builds/suites and visible console selection. Do not call a mixed population idle.
4. Report per-client bytes/s and whole-server CPU separately. The parser sample establishes hot paths, not exact per-client CPU attribution. If exact CPU attribution materially changes the choice, propose labelled sampling rather than expensive per-chunk native timing.
5. Compare candidates only after those numbers. TL approves a concrete M2 design before change-code for that scope.

## Optimization alternatives to decide after measurement

| Option | Benefit | Correctness/cost constraint |
|---|---|---|
| Visible-only emulator + existing raw tail | Removes most hidden emulation | Rejected as specified: 128KiB tail loses earlier screen/modes/history and can split sequences; hidden replies/effects cannot wait. Existing ring-eviction regression proves this. |
| Checkpoint + bounded lazy delta parsing | May coalesce hidden redraw work, retaining exact state | Every byte eventually parses; overflow must catch up without loss, resize must keep chronology, queries/effects must stay prompt, selection catch-up needs a latency bound. An independent partial VT parser duplicates authority. Not approved yet. |
| Preserve eager authority; optimize measured VT/publication hot path | No hidden-state reconstruction or protocol delay | Savings depend on measured parsing/copy/grapheme cost and input mix. A narrow source-derived fast path may be smaller than a new lazy state machine. Not approved yet. |

M2 must preserve off-screen notices/idle/peek and all protocol effects, have no new background worker and bound memory per client. Before/after live server sample/CPU and console/peek/waiting probes remain whole-issue acceptance.

## Architectural checks

ARCH-DRY: reuse Console inventory and resource projection, no second terminal parser or metrics service. ARCH-PURE: immutable diagnostics/projection separated from atomic Child state and request IO. ARCH-PURPOSE: M1 only supplies the measurements; it does not claim CPU reduction. ARCH-MOCK: existing fake Child/Console/service share production seams; combined live restart supplies conformance. ARCH-CONSTRAINTS: constant per-chunk work, bounded client/wire snapshots, no hot-path clocks. ARCH-SECURE: expose identities/totals only, no output or credentials. ARCH-ORDER: counters monotone within an explicit incarnation; comparisons reject replacement, incomplete or reordered samples. ARCH-FUNERAL: one scalar dies with Child; response snapshots die with request; no new durable artifacts/workers.

## Revisions

### 2026-10-16 — M1 diagnostics authorized before optimization

TL replaced capture choreography with permanent atomic ingress counters. CPU timing remains optional only if cheap and truthful; M1 omits it. The mechanism comparison follows measured rates after the combined restart.

### 2026-10-16 — approved M1; function-specific test strategy

TL approval e6511b06 confirms the atomic ingress counter, read-only hosted resources section, no per-feed CPU timing and full gates. Plan-quality PQ-1 requested function-named adversarial strategies; task verification now names each risky function, its independent oracle and its mechanical guard instead of listing cases. Scope and mechanism are unchanged.

### 2026-10-16 — measured M2 proposal, awaiting TL approval

M1 landed PR382; TL verified it and restarted at22:22. Dispatch4306f6e3 requests the natural measurements and M2 plan. [Measurement report](000622-live-measurement.md) retains matched per-client deltas, cumulative server CPU, both native samples, census changes, host contention and loaded-image uncertainty. The old M1 design above remains the historical scope; this revision proposes M2 and does not authorize code.

**Goal:** Reduce eager terminal CPU for ordinary ASCII while retaining exact Unicode, hidden-state and protocol behavior. This is not a promise that two fast paths alone meet whole-issue acceptance.

**Architecture:** Keep Endpoint as the sole eager VT authority. Optimize two existing pure computations at their current ownership boundaries. No new worker, lazy buffer, visibility policy, mode or public API. ARCH-DRY/ORDER favor retaining that authority over a protocol-only second parser; ARCH-PURPOSE requires end-to-end measurement rather than treating grapheme microbenchmarks as server savings.

## M2 conceptual inventory

| Name | Lives in | Status |
|---|---|---|
| Printable ASCII boundary/width decision (PURE) | third_party/vt/utf8.go, existing handlePrint | modified |
| Cell grapheme validation (PURE) | cmd/internal/terminal/frame.go, existing validateCell | modified |

These are local decisions, not new domain types. The first belongs 1:1 to the emulator's existing provisional cluster; the second validates each immutable Frame cell. Each bypasses only a known width1 printable ASCII segmentation result. Retain general Unicode behavior as the extension path rather than duplicating it.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Eager emulator state transition | third_party/vt/utf8.go, handleGrapheme | existing/reused | cells, cursor, history, cluster amendment |
| Authoritative feed and publication | cmd/internal/terminal/endpoint.go; cmd/internal/ptychild/terminal.go | existing/reused | parser, protocol replies/effects, snapshots and parent output |

No new IO integration or fake service. Reuse real in-memory VT, existing fake Child/Endpoint and existing reply/effect recorder seams for correctness and complete-ingest benchmarks.

## M2 implementation tasks after approval

### 1. Establish independently checked semantics and cost

Files: third_party/vt/pair_grapheme_test.go, new colocated benchmark/test file if needed; cmd/internal/terminal/frame_test.go and colocated benchmarks; existing ptychild ingest benchmark seam.

- [ ] Add exhaustive95×95 printable-ASCII pair checks against the unchanged Unicode library and current slow transition reference. The expected boundary and width are independently derived; changing the shortcut predicate must break a negative/control case.
- [ ] Build a test-only slow handlePrint reference from the pre-optimization algorithm, with explicit equivalence checks against the unmodified production method before using it as an oracle. Generated streams vary chunk partitions, geometry, cursor/pending-wrap, charset selection, control boundaries and Unicode continuations. Compare cells, cursor, modes, history, pending wrap and replies/effects through actual stateful terminals; not only returned widths.
- [ ] Include combining marks, selectors, keycaps, ZWJ, Unicode Prepend, cluster-size overflow/drop state, split UTF-8, bottom-right width changes, alternate screen, resize and EOF ordering. Pin existing split-grapheme regressions. Fast path must retain the amendable ASCII cluster so the next non-ASCII continuation takes the original path correctly.
- [ ] For validateCell, compare the result of every ASCII width/link/control variant and generated valid/invalid Unicode cells with the unchanged slow validator. Include nonempty width0, wrong width2, broken wide companions, control/link violations and oversize clusters, proving no validation bypass.
- [ ] Establish committed-baseline benchmarks for ASCII redraw, mixed Unicode/control, frame validation and complete Child.ingest including Endpoint publication. Use fixed geometry/payload/chunks, GOMAXPROCS=3 and GOFLAGS=-p=3, serial repetitions with host load and suite census. No counter-only or mocked parser benchmark as the acceptance oracle. Retain logs and all repetitions.

Tests of semantic equivalence should pass before optimization; benchmarks characterize the current cost. Do not manufacture a failing correctness test for unchanged behavior. The independently established oracle must detect deliberate predicate or bookkeeping mutations when qualifying its coverage.

### 2. Implement the smallest measured shortcuts

Files: third_party/vt/utf8.go; cmd/internal/terminal/frame.go.

- [ ] In handlePrint, bypass joined-cluster and new-rune segmentation only when both the existing cluster and incoming rune are single printable ASCII (0x20..0x7e) and bounds permit them. Preserve clusterDropping reset, cluster coordinates/old cells, provisional amendment and handleGrapheme. Empty/non-ASCII predecessors and all extensions retain the slow path. Avoid adding mutable policy or a second public entry point.
- [ ] In validateCell, retain plainText, max size, validLink and width checks; skip only grapheme segmentation when len(Content)==1, byte is0x20..0x7e and Width==1. Preserve continuation and wide-neighbor checks. Do not skip the validator or defer safety to a caller.
- [ ] Run the differential/generated controls and existing VT, terminal, ptychild and console suites. Run targeted races through production Endpoint/Child consumers.
- [ ] Repeat baseline/candidate benchmarks in balanced serial order. Both shortcuts must independently show the intended local effect; retain only changes with repeatable benefit. Require a complete-ingest improvement beyond run-to-run spread and no material mixed-Unicode regression. If not observed, stop and report a revised design; do not add more mechanisms under this approval.
- [ ] Commit small coherent changes with why/model trailers. No clocks or new allocations on the per-byte path; benchmark allocations alongside elapsed time.

### 3. Verify consumers and whole-server effect

Files: README.md, atlas/couch.md if behavior/architecture account needs updating; issue Log and this plan. Reconcile inventory table to actual symbols before review.

- [ ] Pin off-screen query reply bytes/order, exactly-once effects, sync publication, mode/input encoding, resize chronology, EOF and ring-eviction state through existing fake/real Endpoint tests. Waiting/peek use wrapper status/tails and remain independently exercised; no lazy-state flag is introduced.
- [ ] Commit/push; run required sdlc local-check run --name suite on the pushed head. M2 milestone-close performs the mandatory fresh-context review; fix/dispute findings with evidence. No duplicate review dispatch.
- [ ] Ask TL to coordinate candidate live check/stage/restart; worker never stages :0 unasked. Retain before/after counters, matched population, byte mix, cumulative CPU and5s sample under comparable natural load, plus console/peek/waiting checks. Server attribution must state build/selection/environment uncertainty; don't claim CPU reduction from unmatched workload or microbenchmarks.
- [ ] Whole-issue closure requires the measured live CPU reduction and existing consumer checks, or an explicit TL-approved revision of the acceptance/landing order. Do not infer that M1's pre-restart landing exception applies to M2. Report PR/build and measurements to TL; TL owns portfolio edits.

## M2 bounds and stop conditions

Verification commands after approval (run VT commands with third_party/vt as cwd; others from repository root):

```sh
go test ./... -count=1
go test ./cmd/internal/terminal ./cmd/internal/ptychild ./cmd/internal/couchtty -count=1
go test -race ./cmd/internal/terminal ./cmd/internal/ptychild -count=1
go test ./cmd/internal/ptychild -run '^$' -bench 'ChildIngest' -benchmem -benchtime=500ms -count=5
sdlc local-check run --name suite
```

Add concrete benchmark names with the new tests and record their exact baseline/candidate invocations. All correctness commands must pass; benchmarks report allocation and distribution, not a pass/fail speed claim. If a new production Go file becomes necessary, include the artifact inventory check and update its manifest; prefer existing source files for this scope.

No extra per-client buffer/state, worker, visibility-dependent behavior or authority. Local helpers must not become another grapheme implementation: they recognize only proven ASCII cases. A source predicate wider than that, repeated failed optimization, or need for new terminal state is a design delta requiring TL review. Lazy replay and upstream repaint reduction remain deferred alternatives. No exact per-client CPU attribution is claimed; bytes plus whole-server CPU/profile are the agreed measurement boundary.

### 2026-10-16 — repeated benchmark benefit not established; implementation held

The approved stop condition fired after a complete ABBA experiment: narrow shortcuts are built and semantic controls pass, but complete-redraw wall-time benefit did not repeat. [Evidence and proposed measurement revision](000622-m2-benchmark-evidence.md) retains all runs, broad candidate B2 deterioration, mixed/ASCII contrary evidence even before B2, host load, source/binary/log hashes and the failed test-only oracle qualification corrected before production. No full suite, review, stage or live acceptance is claimed.

Proposed next action is measurement-only: fixed identical work counts per case, paired ABBA order per workload and owned-child cumulative CPU outside the hot path, alongside wall time and contention metadata. This distinguishes CPU cost from descheduling without changing the candidate mechanism. Await TL approval; no blind benchmark retry or wider optimization.
