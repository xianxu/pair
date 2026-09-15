# Terminal Abstraction Design and Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach. Steps use checkbox (`- [ ]`) syntax for tracking. This is the architectural proposal; backend qualification and detailed implementation planning must precede production replacement.

**Goal:** Give programs inside Couch and Pair a faithful, explicitly supported terminal whose state cannot be changed by surrounding UI or another child.

**Architecture:** Each child has an authoritative virtual terminal. A shared compositor renders the selected terminal's cells and Couch/Pair chrome into the parent terminal; raw child drawing commands never cross that composition boundary. Input and query replies are encoded for the originating virtual terminal, while one owner sequences view selection and parent output.

**Tech Stack:** Go, existing PTY/Host seams, pinned `charmbracelet/x/vt` and Ultraviolet as candidates, real Zellij/Neovim conformance and an independent terminal test interpreter.

**Status:** Proposed; requires operator design approval. No implementation or backend suitability is claimed. #255 remains open work until both consumers meet the contract. No estimate before plan-quality acceptance.

## Decision and alternatives

Recommend a virtual-terminal endpoint, conditional on backend qualification. Pair already depends on an emulator for observation; promote a qualified backend behind a Pair-owned interface, rather than turn the selective Screen observer into a new emulator.

1. **Virtual-terminal endpoint (recommended):** isolates cursor, margins, save slots, buffers and parser state per child. View switches render snapshots, not historical commands. Cost: Pair must faithfully implement/advertise the terminal protocol and route replies/effects; the current dependency is not sufficient without work.
2. **Stateful passthrough translator:** preserve raw rendering but translate all relevant cursor/margin/buffer/query semantics and reconcile modes. This eventually approaches emulator complexity while retaining insertion hazards. A narrower profile that excludes normal nvim/Zellij behavior would not meet this issue.
3. **Move all UI/multiplexing into Zellij:** use its existing terminal boundary. Potentially fewer terminal engines, but changes Couch's independent multi-session switching and Pair's pane-tab architecture. This is a product/integration redesign, not the selected scope.

## Current evidence

- `ptychild.Child.pump` feeds each child's Screen and ring continuously before sink delivery, including background output. Preserve this single-ingestion point and Child-owned geometry.
- `ptychild.Screen` is a selective framing/mode observer, not a screen model. Its SafeToPaint permits some cursor-save interference in alternate buffers and misses incomplete UTF-8 (#252).
- `hostty.Reservation` changes parent margins and uses cursor-save state shared with the child. A typed writer alone cannot isolate these semantics.
- Couch `switchTo` updates selection before takeover; Pair `switchRelative` similarly changes active before queuing takeover. Pair drops output positions already available in OutputBatch. A mutex around bytes does not order the selection decision.
- Raw replay can begin inside a sequence, omit persistent mode setters, reintroduce historical state and repeat effects/queries. Its notification-safe endpoint is not a general terminal checkpoint.
- Pinned x/vt has screen/cell/cursor/margin/buffer APIs but source inspection identifies extended keyboard TODOs, missing CSI-u cursor restore and legacy buffer handling, OSC8 parameter ordering concerns, synchronous reply-pipe behavior, and a need to test grapheme splits. These are qualification findings, not yet independently reproduced regressions.

## Core concepts

| Entity | Kind | Proposed home | Status and responsibility |
|---|---|---|---|
| TerminalProfile | PURE | cmd/internal/terminal/profile.go | New: supported capabilities, replies and deliberate protocol restrictions |
| TerminalFrame | PURE | cmd/internal/terminal/frame.go | New: immutable cells, cursor, dimensions, frame generation and output position |
| ViewState / ReduceView | PURE | cmd/internal/terminal/view.go | New: selected endpoint, geometry generation, drag destination and ordered effect decisions |
| TerminalEndpoint | INTEGRATION | cmd/internal/terminal/endpoint.go | New: one backend instance and ordered ingestion/reply/effect ownership per child |
| ParentPresenter | INTEGRATION | cmd/internal/terminal/presenter.go | New: exclusive typed parent-output door, renderer and write-outcome handling |
| ptychild.Child | INTEGRATION | cmd/internal/ptychild/child.go | Modified: PTY lifetime/geometry and integration with endpoint ingestion |
| hostty.Host / Fake | INTEGRATION | cmd/internal/hostty/{host,fake}.go | Reused: parent terminal IO and buffered Fake; partial/error behavior requires extension from couchtty mouseTraceHost |
| Console / terminalMux | INTEGRATION | cmd/internal/{couchtty/console,termcmd/run}.go | Modified: product policy submits events and chrome, cannot write child drawing bytes to parent |

TerminalEndpoint is not labeled pure: the candidate backend has reply pipes and mutable IO lifecycle. Profile/Frame/View transition tests require no IO. Do not create a parallel parser for each console. Existing wrapcmd observer remains a separate observer of its own connection; it must not become a competing authority for the same endpoint. During migration, old Screen-derived mode getters must delegate to the endpoint or be removed for migrated consumers (ARCH-DRY, ARCH-PURE).

## Behavioral contract

### Terminal profile and effects

The required initial profile preserves current interactive Zellij, nvim, shell and agent behavior: UTF-8 including wide/combining characters; indexed/truecolor styles; cursor position/style/save, erase, scroll regions and origin mode; primary/alternate buffers; application cursor/keypad; bracketed paste; focus; SGR mouse with off/click/drag/all-motion; the extended keyboard negotiation/encoding needed by existing shortcuts; hyperlinks; synchronized drawing; clipboard and Pair notifications. Backend qualification must enumerate exact sequences/queries from these features, advertise only implemented capabilities and provide matching terminfo/environment. Existing required shortcuts cannot be removed to make a deficient backend pass.

Each child query is answered from its own endpoint, including hidden children. No raw query is forwarded to a parent whose reply could reach a different child. Clipboard, title, bell, cwd and notification effects retain existing product policy and exact origin identity; they are emitted once from ingestion, never from drawing a frame. Clipboard query/read behavior must be specified against current policy before admission; no new ambient clipboard reads are implied. Unknown strings remain framed and handled according to the declared profile, never passed into the parent's drawing parser as an escape hatch. Unsupported graphics/pixel protocols are not advertised; required existing behavior discovered during qualification cannot be silently reclassified as unsupported.

### Rendering, state and switching

Child dimensions exclude chrome. Child scrolling/erase/save operations affect only its virtual screen. Published frames contain cells and cursor state; the presenter composes them with chrome using renderer-owned cursor/margins/styles. Switching therefore cannot replay clipboard writes, queries or stale mode setters. Raw capture/history remains available for its existing purposes but loses display-authority status. A continuously attached child requires no resize nudge to reconstruct an evicted replay.

Endpoint ingestion yields coherent state at an explicit output position; frames published during synchronized output remain unchanged until end-of-frame or a specified bounded timeout. Snapshot retention is bounded; coalesce superseded frames rather than queue unbounded copies. A switch selects one published frame and generation; subsequent output advances that endpoint only. Background endpoints continue parsing and replying, but cannot emit drawing bytes to the parent.

ParentPresenter owns the ordered event stream for select, input admission, output frame publication, resize and release. Product workers request a switch and receive its completion; they do not first mutate the active input destination. Late frames cannot replace a different selected generation. Resize acknowledgment reflects the actual PTY outcome and endpoint geometry epoch, not merely a requested size. A select completion means the view has been presented successfully before ordinary input is admitted to the new destination. While presentation is pending or uncertain, pause ordinary input admission with a bounded queue; reserve a parent-owned escape/stop path. On partial parent writes, retain the known accepted prefix and invalidate the rendered-screen cache; retry a known remaining suffix only when transport outcome permits, otherwise stop parent output and surface failure. Do not reset a parser in memory and assume the physical parent is reset. Released presenters admit no further writes.

### Input

The parent may use1003 plus SGR mouse reporting, while each endpoint receives only events its own profile/state requested. Couch handles clicks on its UI and discards motion there. An admitted press binds its drag/release to the same destination; switching/closing during a drag requires an explicit cancel/release policy verified before implementation. Zellij remains responsible for its own inner panes. Keyboard, focus and paste are decoded once at the parent boundary and encoded for the selected endpoint, preserving negotiated semantics and existing reserved shortcuts. Query replies use origin identity, never current focus. One ordered child-input writer serializes complete encoded operator events, paste payloads and query replies; independent reply-drain goroutines cannot interleave bytes with user input. Host capability replies are consumed by the presenter rather than forwarded as operator input.

### Coverage across Pair

Both compositors must adopt the same contract: Couch hosts a Zellij client plus Couch UI; `pair term` hosts shell/nvim terminals plus Pair tabs. `pair wrap` remains an explicit transformation/observation boundary inside a Zellij pane, not an automatic extra compositor. Audit its `stdoutChunk`/`stripCodexOutputMarkers`, notification rewriting, Return translation and query/reply tracking. Define whether each wrapper observer consumes raw agent output or the transformed stream delivered to its parent; those are distinct observations and cannot substitute for one another. Each retained transform must state its semantic exception and preserve framing and capabilities; remove filters that violate the declared contract through coordinated #254 work. A composed test must transport actual wrapper output through Zellij/terminal consumers, rather than manufacture equivalent bytes independently. Neither a green Couch-only result nor unchanged wrapper tests closes #255.

## Qualification before production changes

Create a bounded fixture matrix and compare observable results to independent expectations and a real terminal/multiplexer where applicable. The candidate backend itself cannot be the sole oracle. Use synthetic fixtures plus minimal captures without prompt content; do not drive live operator sessions.

Required qualification classes:

- Every split of UTF-8, combining/ZWJ graphemes, CSI/OSC/DCS, malformed input and termination: equivalent whole/split behavior and bounded parser state.
- Cursor save/restore variants, alternate-buffer variants, margins/origin, scrolling/erase, resize and normal-screen history: correct cells and cursor after subsequent output, not only an immediate screenshot.
- Keyboard negotiation and existing Couch/Pair shortcut encodings, press/repeat/release where required; all supported mouse states and encodings; focus/paste.
- Origin-bound queries and one-shot effects, including hidden endpoints and reply backpressure; synchronized output begin/end and bounded recovery.
- Compose chrome and switch A/B while each retains distinct cursor/margins/modes; evict raw replay and split output at the switch boundary; force delayed frames and partial writes.

Known candidate gaps must be reproduced, fixed via maintained upstream changes or a narrowly scoped owned adapter/fork, and rerun against this matrix. Do not edit the module cache or add scattered console overrides. If qualification indicates a broad emulator fork is necessary, stop and return with costed alternatives before choosing it. M1 is evidence for a backend decision, not completion of #255.

## Delivery boundaries

### M1 — Qualify the terminal contract and backend

- [ ] Enumerate exact required protocol/capability and compatibility matrix from production entrypoints and fixtures; define frame timeout, queue limits, normal-screen history and input cancellation semantics.
- [ ] Add independent conformance fixtures in cmd/internal/terminal and exercise the pinned candidate, recording failures before fixes.
- [ ] Resolve backend selection and maintenance strategy from results; review the detailed M2–M4 implementation plan before production migration. Do not advance with unmet required semantics.
- [ ] Close M1 through SDLC with the matrix and qualified backend decision (or re-plan if qualification fails).

### M2 — Shared endpoint and parent presentation

- [ ] Implement endpoint/profile/frame/view/presenter under the approved backend decision; use Host/Child seams and lifetime-bound reply transport.
- [ ] Add forced-order and partial-IO tests through production event/effect paths; ensure typed doors prevent raw bypass.
- [ ] Document actual capability/resource bounds and complete SDLC review before integrating a console.

### M3 — Couch and Pair adoption

- [ ] Migrate Couch and Pair term to the same shared abstraction; remove raw replay and state-changing reservation from their display paths.
- [ ] Replace independent mode authority and before-queue selection mutations; audit and reconcile pair wrap transformations against the same contract, preserving notification, Return, query/reply, capture and park/resume behavior.
- [ ] Verify both consumers, nested Zellij, shell/nvim and agent fixtures; update linked issue dispositions from actual acceptance evidence and close the boundary.

### M4 — Live conformance and publication

- [ ] Compare baseline/candidate startup, sustained output CPU/memory, redraw throughput, switch latency and idle mouse movement; verify the provisional budgets established in M1 before rollout; M4 must not be the first point where acceptable bounds are decided.
- [ ] Run full Go/race and existing Lua/shell/shortcut/retention suites appropriate to integration; add isolated native terminal conformance to CI.
- [ ] Smoke-test an isolated candidate before any operator runtime replacement; document capabilities, failure/recovery and diagnostics in atlas/README.
- [ ] Close and publish #255 only when both consumers satisfy the declared terminal contract.

## Bounds, failure and artifacts

Per-endpoint screens are bounded by the declared geometry and scrollback policy; frame queues retain latest publishable state, and non-coalescible replies/effects use bounded backpressure rather than silent dropping. Drain/cancel reply workers with endpoint lifetime; a dead child cannot block other endpoints indefinitely. Qualification must set numeric limits from existing geometry/capture bounds and representative measurements before implementation approval. No new durable per-event trace is planned; use existing diagnostic retention for failure summaries and keep fixture captures bounded. Existing native session/process lifetime stays owned by current lifecycle components; #256 owns generic redesign (ARCH-CONSTRAINTS, ARCH-FUNERAL).

## Architecture review markers

ARCH-DRY: shared endpoint/presenter for both consumers, reuse PTY/Host and qualified libraries. ARCH-PURE: declarative profile/frame/view core, backend and transport honestly integration. ARCH-PURPOSE: full terminal abstraction at both consumers, not only mouse/UTF-8 symptom fixes. ARCH-MOCK: independent stateful interpretation and live conformance. ARCH-CONSTRAINTS: bounded screens, queues, replies and explicit measurement gate. ARCH-SECURE: capability truthfulness and origin-bound side effects, no unknown raw escape passthrough. ARCH-ORDER: one selection/output/input authority with forced scheduling and partial-outcome semantics. ARCH-FUNERAL: endpoint-owned worker lifetime and retained existing diagnostic policy.

## Revisions

### 2026-09-15 — Qualification review refinements

Made Pair wrapper coverage explicit, distinguished raw/transformed observation, required one ordered child-input writer for operator events and query replies, and specified input admission after successful presentation. The proposal is for architectural approval and M1 qualification; detailed M2–M4 implementation plans and numerical limits remain subject to that evidence and a subsequent approval checkpoint.

### 2026-09-15 — Fresh spec review accepted

Fresh-context reviewer terminal_spec_review approved the architectural proposal and M1 qualification with no blocking findings. It did not approve M2–M4 implementation details or establish backend suitability. Advisory input serialization, presentation-failure admission, wrapper stream semantics and M1 budget timing are incorporated; normal-screen history, cursor style, hyperlinks and grapheme fidelity remain explicit qualification cases.

## Chunk 1: M1 qualification implementation (approved phase)

The operator approved the architecture and qualification phase on 2026-09-15. Implement the following bounded diagnostic tool and evidence; no Console, terminalMux, wrapper behavior, dependency version or live installation changes belong to this phase. A failing qualification is a valid M1 result and requires re-planning before production adoption; it does not waive any #255 acceptance property.

### Deliverables and semantics

Add `cmd/internal/terminalqualify` and `cmd/probes/terminalqualify`. `Run` executes the pinned emulator against literal protocol expectations and returns a JSON report with candidate version, case identifier, capability, expected/observed result and pass/fail/not-covered. `Qualified` is true only if all required cases pass and none is not-covered. The probe exits1 on failed/incomplete qualification,2 on invocation/infrastructure failure,0 only on complete qualification. Ordinary Go tests validate the runner and report honestly; they do not turn known backend defects into accepted terminal behavior. Store a compact checked-in report/interpretation under `workshop/plans/000255-terminal-qualification.md`, with exact rerun command and source baseline. Raw output is synthetic, never operator prompt content.

A production compositor does not exist yet. Composition/lifecycle obligations that cannot be established by endpoint experiments must appear as not-covered, not fabricated successes. The decision report names what M2 would need to prove. M1's admission decision for this pinned backend is independent of a future adapter's possible suitability.

| M1 entity | Kind | Home | Test |
|---|---|---|---|
| Case / Observation / Result / Report | PURE | cmd/internal/terminalqualify/{cases,report}.go | literal cases, honest aggregation, bounded mismatch details |
| Candidate | INTEGRATION | cmd/internal/terminalqualify/candidate.go | x/vt instance, reply pipe lifecycle and ordered operations |
| Probe Run | INTEGRATION | cmd/probes/terminalqualify/main.go | JSON output and exit status with injected report runner |

Use the existing pinned backend via a disposable Candidate per case. Observations copy cells/cursor/links and emitted replies; no expected result is generated by feeding the same backend the same input. Whole/split equivalence is a supplementary metamorphic check, not the only oracle. Backends with missing methods are reported not-covered; behavior that can be exercised and is wrong is fail. Expectations cite terminal protocol sources and existing production contracts in case comments.

### Task 1 — Reporting and candidate lifecycle

Files: create `cmd/internal/terminalqualify/report.go`, `report_test.go`, `candidate.go`, `candidate_test.go`.

- [x] Test `Report.Validate`, `Report.Qualified`, `Compare` and `boundedDetail` using malformed/duplicate/incomplete result sets and mismatching observations; qualification requires exact required-ID coverage and rejects every unmet requirement, while only displayed evidence is truncated.
- [x] Run `go test ./cmd/internal/terminalqualify -run Report -count=1` and record the initial missing-implementation failure; implement report types/aggregation, rerun to PASS.
- [x] Test `Candidate.Execute`, `Candidate.Snapshot` and `Candidate.Close` through controlled reply-producing/blocked IO and cancellation schedules; assert literal state, origin isolation, completion within2s and joined workers. Candidate owns one reply reader and one command path, closing pipes to unblock teardown.
- [x] Implement the thin candidate wrapper; make teardown close both reply directions as required by pinned InputPipe. Run focused race tests. Inject a blocked/failing IO double to verify runner timeout/cleanup mechanics independently of x/vt.

### Task 2 — Rendering/framing matrix

Files: create `cmd/internal/terminalqualify/cases.go`, `screen_cases.go`, `screen_cases_test.go`.

- [x] Implement `ScreenCases` for the required rendering/framing capability classes above; `TestScreenCases` enforces unique IDs, non-empty literal expectations and complete class coverage without deriving expected state from the candidate.
- [x] Test `Compare` and `SplitInputs` against hand-built observations and adversarial byte partitions; any changed cell/cursor/style/link or missed split must be detected. Preserve the full protocol matrix in executable fixtures, not repeated prose lists.
- [x] Test `RunCase` with injected correct/incorrect candidate observations, oversize inputs and cancellation; report failure or infrastructure error accurately, never promote an unobservable behavior to pass.

### Task 3 — Input/query/effect matrix and scope gaps

Files: create `cmd/internal/terminalqualify/input_cases.go`, `input_cases_test.go`, `coverage.go`.

- [x] Implement `InputCases` and `Coverage` for the required input/query/effect and deferred-composition classes above. `TestInputCases` and `TestCoverage` mechanically enforce unique identifiers, required class inclusion and explicit not-covered obligations.
- [x] Test `Compare` against independently specified protocol bytes and deliberate suppression/encoding mismatches; `Candidate.Execute` tests prove reply drain and cancellation ordering, separate from candidate conformance.
- [x] Read `wrap.go` raw/transformed paths and document exact integration tests needed for M3; do not change wrapper filters in M1.

### Task 4 — Probe, measured report and backend decision

Files: create `cmd/probes/terminalqualify/main.go`, `main_test.go`; update `workshop/plans/000255-terminal-qualification.md`, `atlas/architecture.md`, issue Log.

- [x] Test probe `run` with an injected report runner across valid, incomplete and infrastructure-failure reports plus failing output writes; parse emitted JSON and enforce exit-status meaning. Implement a2-minute whole-run context, deterministic order and build-version metadata.
- [x] Run `go test ./cmd/internal/terminalqualify ./cmd/probes/terminalqualify -count=1` and `go test -race ./cmd/internal/terminalqualify ./cmd/probes/terminalqualify -count=1`.
- [x] Run `go run ./cmd/probes/terminalqualify > /tmp/pair255-terminal-qualification.json`; expected exit1 while required gaps exist. Inspect every failure and distinguish candidate mismatch from a defective oracle. Correct oracle errors only with explicit evidence and revisions.
- [x] Measure synthetic80x24 and240x80 screen feed/snapshot costs via benchmarks, reporting raw observations. Existing262144-cell dimension bound is the candidate safety ceiling; initial history cap1000lines and report mismatch cap4KiB/case bound qualification memory. These diagnostic limits are not production performance promises. M2 must set provisional production budgets from measured endpoint cost multiplied by representative Couch thread counts, before implementation approval.
- [x] Write the backend decision: suitable unchanged / suitable only with enumerated owned adaptation / reject candidate. Any required fail or not-covered blocks unchanged production adoption. If fixes entail broad backend ownership, stop for the existing re-plan checkpoint; do not silently start that work.
- [x] Verify full `go test ./...` after generated runtime assets if necessary, `git diff --check`, record all evidence and submit the M1 boundary to SDLC. The reviewer assesses the qualification tool/evidence, not a claim that #255 or production migration is complete.

### M1 bounds and independence

No added runtime worker or durable artifact in ordinary Couch/Pair launches. Each case uses one disposable emulator and closed reply transport; execution is sequential and context-bounded. Report evidence caps apply to stored mismatch text, not to the expected predicate. Tests include cancellation and deterministic fake IO behavior. Protocol literals provide an independent oracle; a second terminal implementation/live Zellij remains required before admitting production semantics. Native tests here, if needed to settle an oracle, use isolated sockets/configuration only.

### 2026-09-15 — Plan-quality round1 correction

PQ-1: replaced repeated case prose with function-level adversarial test strategies and mechanical guards; protocol classes remain specified above and exact cases belong in executable fixtures. PQ-2: corrected hostty.Fake capability claim; it buffers writes, while controlled partial/error behavior currently lives in couchtty mouseTraceHost and must be extracted/extended for production presentation tests. No semantics or phase authorization changed.

### 2026-09-15 — Primary live acceptance clarified

Operator requires the ongoing display corruption and missing live selection highlight to go away. M4 must demonstrate continuous drag highlight in agent and right shell/nvim panes across panel/thread/tab switches and reattachment, and investigate all reported display symptoms rather than equating a fixed UTF-8 regression with complete visual recovery. Record terminal/build, duration and exact workflows during sustained use; final closure requires operator acceptance. Qualification, semantic tests and architecture are supporting evidence, not substitutes for this result.


### 2026-09-15 M1 qualification outcome

The executable qualification is implemented under `cmd/internal/terminalqualify`, isolated from production. Snapshot is private and captured atomically by Execute; exposing an independent Snapshot method would allow unordered reads. The report rejects unchanged adoption (41 pass, 15 fail, 14 not-covered). This is the negative-result branch of M1: backend selection and detailed M2–M4 remain at the approved re-plan checkpoint. No production migration is authorized by a successful qualification-tool review. Fixed fixture inputs are capped at 64KiB before constructing partition metadata. See `000255-terminal-qualification.md` for evidence and limits.


### 2026-09-15 review-driven qualification corrections

Reason: first M1 review found sparse split predicates, unobserved style fields and incomplete evidence/documentation. Delta: full-observation split equivalence plus independent literals; attribute/underline fixtures; bounded structured evidence with explicit truncation and comparison kind; README usage. Matrix expands from 70 to 82 obligations. This corrects the qualification instrument without changing production scope or final operator acceptance.
