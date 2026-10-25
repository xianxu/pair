# Live Couch allocation correction implementation plan

> **For agentic workers:** Follow AGENTS.md §3; the boundary binaries own fresh review. This change uses session-warm ownership reasoning and executes in the current session with superpowers-executing-plans. Do not create a second full check while #879's runner is active.

**Goal:** Remove repeated full-screen backing allocations at synchronized-output capture while retaining the terminal's existing publication semantics.

**Architecture:** Reuse only the endpoint's private published Cells/Rows storage at unchanged geometry, under its existing mutex. Every public consumer keeps an independent clone. No pool, extra lifetime owner, parser throttling or new publication state.

**Tech stack:** Go; existing in-process vt emulator and controlled ttyio transport; focused tests, allocation counts and bounded replay benchmarks.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| Frame / Frame.Clone | cmd/internal/terminal/frame.go | reused: immutable consumer frame with independently copied Cells/Rows |
| Geometry / Geometry.Validate | cmd/internal/terminal/profile.go | reused: existing shape and maximum cell budget |
| ProfileLifecycle / StepProfile | cmd/internal/couchcmd/profiling_model.go | reused from landed #871; attribution evidence only, no change |

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Endpoint.capture | cmd/internal/terminal/endpoint.go | modified | serialized vt screen/cursor/row reads into privately owned storage |
| Endpoint.capturePublication / updatePublication | cmd/internal/terminal/endpoint.go | reused callers | existing sync, geometry and EOF publication sequencing |
| Child.ingest | cmd/internal/ptychild/terminal.go | reused | actual parser/publication path exercised by a bounded benchmark |
| Launch profiler | cmd/internal/couchcmd/profiling.go | reused from #871 | runtime/pprof and private bounded files, unchanged |

Frame/Geometry unit logic stays pure; mutable emulator reads remain in Endpoint. The existing stateful ttyio fake and actual in-process vt backend qualify the integration without a second parser mock or live process. No new external dependency is introduced (ARCH-PURE, ARCH-MOCK, ARCH-DRY).

## Evidence and scope

See [live profile](../evidence/000865/live-profile.md): exact clean9bfd7ab51cef5f9d563b19dea087c9ef084c19c9, startup44.80CPU seconds/30.22wall seconds; allocation delta20.86GiB, capture9.62GiB (46.11%), chiefly DEC2026 begin. Later same-process46s CPU≈0.41cores versus1.48 at startup. Sampled allocation deltas can lag GC and are not RSS or an exact allocation rate. The hours-old1.3-core condition may be an uptime effect worth a later aged profile. fseventsd causality remains unknown.

M1 now reconciles profiler delivery under #871 and closes measured attribution. Earlier [issue](../evidence/000865/pre-m2-issue.md) and [design](../evidence/000865/pre-m2-design.md) remain historical captures, including negative/invalid synthetic results. None of their unlanded scaled-harness code enters this main-based branch. M2 targets the measured allocator only; inventory, JSON, terminal parsing and PTY reads retain separately measured costs.

Hub a1a107d5 approved this buffer/clone/count correction before code. Hub f6c298b1 permits isolated implementation while #879 waits, defers the shared lessons append until #879 lands, requires one targeted test at a time and prohibits a second full check while #879's runner is active. Keep pair:0's staged binary unchanged.

## Ownership, actions and ordering

Actions: Feed, Snapshot, Publication, Resize, EndInput, Close. No new action or internal lifecycle state is added. Existing Endpoint.mu serializes capture, backend mutation and publication; its private published frame never escapes without Clone. InputWriter retains its existing joined close.

| Existing state/event | Visible publication | Safe action / possible loss |
|---|---|---|
| ordinary output / capture | latest complete frame; owned consumer copy | feed or read; no added loss |
| sync begin / mutations / repeated begin | pre-hold frame remains unchanged | read frozen copy or await release/timeout; repeated begin cannot postpone recovery |
| release or timeout | current backend frame replaces private publication | read cloned current frame; previous retained copies remain immutable |
| successful geometry change | hold canceled; next publication uses new geometry | allocate exact new-geometry backing once; already-returned consumer clones remain independent |
| same-geometry resize | existing hold/publication unchanged | no acknowledgement call or backing allocation |
| PTY acknowledgement refused before backend change | prior backend geometry and publication | existing error/retry; do not mutate capture backing on failed acknowledgement |
| EndInput / Close | existing final frame/history and close behavior | read final publication until Close; no lost final output or live child |

The risky interleaving is a retained consumer frame followed by another sync capture or resize. The existing mutex prevents concurrent reuse, and Clone isolates the retained frame. Tests control time and acknowledgement callbacks to reproduce these orderings. This work adds no goroutine, asynchronous retry, IO effect or external uncertain outcome. Process death/late callbacks retain the existing owners and policies (ARCH-ORDER).

## Allocation and work budgets

Growth drivers are endpoints E, cells C×R, sync/capture cycles K, consumer publications P and successful geometry changes G. Existing backing allocations grow with E×K×C×R. After warmup at unchanged geometry, new capture Cells/Rows backing allocations must be **zero**, tested for K=64 and128 at80×24 and160×24 (N/2N captures and N/2N cells). A different geometry gets exact new storage once; no retained spare or global pool. Required copying/validation remains O(E×K×C×R), consumer clone cost O(P×C×R), and history cost stays separate.

Count actual storage changes in the production capture/Feed paths and allocated objects through testing.AllocsPerRun. Capture-owned allocated bytes in the fixed-geometry replay must fall at least90% against baseline; zero post-warmup backing reallocations is the deterministic acceptance, not timing. Capture and public consumer clones are measured separately so ownership copies are not hidden exceptions.

Memory remains bounded by existing Geometry.Validate/MaxCells: one private current-geometry frame per endpoint plus already-owned consumer copies. Overwrite every element, including zero for absent backend cells, to drop stale glyph/style/link references. Different geometry replaces the private slices and lets the previous private allocation become collectible. Endpoint disposal collects its remaining private storage with the existing owner. No durable runtime artifact or cache is created (ARCH-CONSTRAINTS, ARCH-FUNERAL).

Terminal bytes remain untrusted; keep all existing captured-cell repair, cluster/link/style validation and frame checks. Do not trust a reused cell's previous contents. No credential, persistence format, external service or preference authority changes (ARCH-SECURE, ARCH-EVOLVE).

## M1 — attribution boundary

Files: issue, this plan, bounded evidence/000865 summaries; existing profiler/source remain main's #871 implementation.

- [x] Preserve the historical issue/design and native outcome summaries; record the exact operator profile identity, owners, limits and later CPU window.
- [x] Reconcile #871's landed ownership/review and the TL-approved M2 scope; commit this design before any long gate.
- [x] Run baseline focused Endpoint sync/resize/EOF checks; retain results in the issue Log.

After change-code admits the design and its estimate reconciliation, mark the M1 attribution deliverable complete and run `sdlc milestone-close --issue 865 --milestone M1 --verified 'exact evidence'`. Its mandatory fresh-context review verifies attribution, not the already separately reviewed profiler implementation. Fix named findings; no M2 production code before this boundary clears.

## M2 — correction boundary

Files: modify cmd/internal/terminal/endpoint.go; add cmd/internal/terminal/endpoint_capture_test.go; extend cmd/internal/ptychild/ingest_redraw_benchmark_test.go with the DEC2026 envelope; update atlas/terminal.md; add bounded before/after evidence. Append workshop/lessons.md only after #879 lands. Audit only reader files the inventory names.

- [x] Write N/2N allocation/count regression tests first, covering fixed geometry and the doubled cell count. Run the exact new tests against unchanged capture and retain the expected failure.
- [x] Add the controlled function-level regressions below before reuse, using existing endpoint helpers and the actual emulator.
- [x] Implement the minimal change in capture: build fresh frame metadata, reuse private published Cells/Rows only when geometry/length match, otherwise allocate exact current lengths; write every row and every cell, with `Cell{}` for nil backend cells. Preserve repairCapturedRow and all publication call sites/Clone boundaries.
- [x] Run new counted tests and related Endpoint/history tests; race-check relevant controlled interleavings with count at most10. Iterate only on concrete failures.
- [x] Commit the RED tests and DEC2026 benchmark before the production correction, so that commit pins the unchanged capture with the identical test workload. Compare that baseline to the corrected candidate through bounded capture/ChildIngestRedraw replay; report allocation counts/bytes and CPU timing separately. Do not add a native scale matrix or new harness.
- [ ] Update the existing architecture map and evidence with the ownership/budget result. After #879 lands, append the deferred lesson. Individually audit changed readers and run make test-affected-inventory before close.

| Risky function | Adversarial strategy / mechanical guard |
|---|---|
| Endpoint.capture | Alternate populated and erased/absent backend cells and row metadata; compare complete typed results to independent expected values while counting backing changes and allocations at N/2N. |
| Endpoint.Snapshot / Publication | Retain consumer frames/history across later captures and mutate a returned copy; assert prior retained values and the next fresh result stay independent. |
| Endpoint.Feed / updatePublication | Drive same-feed and split synchronized envelopes with controlled time and repeated transitions; assert the expected complete-state publication and recovery deadline, not callback counts. |
| Endpoint.Resize | Inject refused, no-op and acknowledged geometry changes while held; assert acknowledgement, geometry/epoch, hold cancellation only on actual success and backing replacement on the next capture. |
| Endpoint.EndInput | End a controlled synchronized output/history sequence; assert the final current publication, stable retained copy and joined input lifetime through the existing cleanup owner. |

Targeted commands (one at a time): `go test ./cmd/internal/terminal -run 'TestEndpoint.*(Capture|Sync|Resize|Publication|EndInput)' -count=1`; relevant history tests; focused `-race` on the new capture tests and retained-frame ordering; bounded `-bench ... -benchtime=100x -count=1 -benchmem` on the identical selected replay. No whole-repo local suite.

Workflow postconditions: when #879's runner has ended, push the final head and run `sdlc local-check run --pre-close` once; mandatory `sdlc milestone-close` for M2 and final `sdlc close` as required by the issue's two genuine boundaries, then `sdlc merge --yes`. Reuse recorded checks through docs-only descendants; let merge integrate current main. A profile of a later live/aged workload or stage change remains a separate operator/TL decision.

## Revisions

### 2026-10-24 — measured M2 replaces the parked speculative plan

The prior complete plan is retained at pre-m2-design.md, source01bdfbe72710456ee1170dd54c5b86deba50f3e5. M1 profiler implementation landed independently through #871; this branch contains its attribution reconciliation and the now-approved M2. The new active design is based on main64e0a8452388843b1fd1791059368682eec1236e, not the unlanded scaled harness or #879. Scope, causal limits, private ownership and counted acceptance follow the explicit TL approvals; no new authority or publication semantics are introduced.

### 2026-10-24 — plan-quality round1 transition and strategy corrections

PQ-1: inspected the entire existing Resize transition, including same-geometry early return and successful syncState reset. Corrected the state table: actual geometry change cancels the hold and the next publication captures new geometry; retained consumer clones remain independent. No-op and refused resize keep the old hold. PQ-2: replaced the prose semantic case inventory with one adversarial strategy and mechanical guard per risky function. These clarify existing contracts without adding behavior or changing the approved allocation correction.

### 2026-10-24 — canonical final-gate handoff

The approved isolated checkout is an independent ordinary clone because the numbered environment refuses linked dependency checkouts. Its directory alias is pair-m2. Hub264df7f6 approved finishing implementation there, then after879 lands transferring the exact branch and M1 review provenance into canonical pair:9 for all final gates and PR through the shared pair landing line; remove the disposable clone afterwards. This changes placement only, not code, owners, budgets or test requirements.
