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

- [x] Enumerate exact required protocol/capability and compatibility matrix from production entrypoints and fixtures; define frame timeout, queue limits, normal-screen history and input cancellation semantics.
- [x] Add independent conformance fixtures in cmd/internal/terminal and exercise the pinned candidate, recording failures before fixes.
- [x] Resolve backend selection and maintenance strategy from results; review the detailed M2–M4 implementation plan before production migration. Do not advance with unmet required semantics.
- [x] Close M1 through SDLC with the matrix and qualified backend decision (or re-plan if qualification fails).

### M2 — Shared endpoint and parent presentation

- [x] Implement endpoint/profile/frame/view/presenter under the approved backend decision; use Host/Child seams and lifetime-bound reply transport.
- [x] Add forced-order and partial-IO tests through production event/effect paths; ensure typed doors prevent raw bypass.
- [x] Document actual capability/resource bounds and complete SDLC review before integrating a console.

### M3 — Couch and Pair adoption

- [x] Migrate Couch and Pair term to the same shared abstraction; remove raw replay and state-changing reservation from their display paths.
- [x] Replace independent mode authority and before-queue selection mutations; audit and reconcile pair wrap transformations against the same contract, preserving notification, Return, query/reply, capture and park/resume behavior.
- [x] Verify both consumers, nested Zellij, shell/nvim and agent fixtures; update linked issue dispositions from actual acceptance evidence and close the boundary.

### M4 — Live conformance and publication

- [x] Compare baseline/candidate startup, sustained output CPU/memory, redraw throughput, switch latency and idle mouse movement; verify the provisional budgets established in M1 before rollout; M4 must not be the first point where acceptable bounds are decided.
- [x] Run full Go/race and existing Lua/shell/shortcut/retention suites appropriate to integration; add isolated native terminal conformance to CI.
- [x] Smoke-test an isolated candidate before any operator runtime replacement; document capabilities, failure/recovery and diagnostics in atlas/README.
- [x] Close #255 after both consumers pass qualification and the operator accepts smoke; publication follows through the SDLC merge gate.

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


### 2026-09-15 — M1 review round 2 partition-test correction

Round 2 disposed BR-1 through BR-4 and raised BR-5: partition regression tests counted calls without proving delivered bytes. Added literal partitions for empty, single-byte, multi-byte UTF-8 and CSI inputs, plus independent byte-preservation and every-boundary assertions over all 256 byte values and mixed Unicode/control streams. The production RunCase executor path is checked with the same invariant. Four mutations are detected by failed assertions: repeating whole input, dropping a byte, skipping alternating boundaries and omitting the first boundary. No production implementation changed in this correction. Final full Go suite passed after round 1 corrections; focused race verification covers these additional tests.


### 2026-09-15 — M1 review boundary complete

SHIP after three boundary rounds, five findings disposed. Executable qualification lives in terminalqualify as specified by Chunk 1. The negative adoption decision takes the documented re-plan branch: production budgets, backend maintenance choice and detailed M2–M4 design remain deferred and unchecked. This milestone does not establish that the live display or selection symptoms are fixed.

## Revisions — 2026-09-15 continuation through M4

The operator authorized continued design and implementation through M4, then a pause for their smoke test before merge. This supersedes the earlier additional-approval checkpoint after negative qualification. It does not waive reviews, protocol requirements, tests, or final live acceptance. M4 completes automated/isolated qualification and prepares the candidate; operator smoke acceptance, issue close and merge follow separately. Do not merge or replace the running operator session before that acceptance.

## Chunk 2: M2 repaired core, endpoint and presenter

### Backend choice and maintenance

Use a checked-in local module at `third_party/vt`, retaining `github.com/charmbracelet/x/vt` through a root go.mod replace. Preserve the pinned source, license and upstream tests, with `third_party/vt/PAIR_PATCHES.md` listing origin, authored changes, qualification commands and upstream reconciliation procedure. Module-cache mutation is forbidden. Current upstream a5dee49b28632257cd9a475e8ca36e98a62ff155 fixes the hyperlink swap and whole-write combining but not split clusters, Kitty negotiation or mouse semantics. Updating alone is insufficient. An alternative core would require new packaging and qualification with no established reduction in work. The owned fork is approximately 3924 upstream production lines, with an anticipated 400–800 authored lines for observed defects before integration APIs/tests; this is scope evidence, not a measured time estimate.

The fork remains rejected until the 68 current executable cases and expanded boundary tests pass without weakening their predicates. The 14 integration obligations are discharged by concrete M2–M4 tests; operator sustained-use acceptance remains explicitly pending until the final smoke test.

### Core concepts for this chunk

| Name | Kind | Lives in | Change and ownership |
|---|---|---|---|
| Profile / Geometry / Limits | PURE | cmd/internal/terminal/profile.go | New: one declared capability and resource contract for both compositors |
| Cell / Cursor / Frame | PURE | cmd/internal/terminal/frame.go | New: immutable copied cells, style/link and cursor state, endpoint identity, geometry epoch and publication generation |
| View / Transition | PURE | cmd/internal/terminal/view.go | New: selected identity, admitted focus, pressed destination, presentation state and legal transitions |
| Render / StyledRows | PURE | cmd/internal/terminal/render.go | New: diff composed cells and constrained SGR/link chrome; emits only presenter-owned drawing controls |
| InputDecoder | INTEGRATION | cmd/internal/terminal/input.go | New: bounded incremental framing around ultraviolet EventDecoder, preserving paste and reserved shortcut behavior |
| TerminalEndpoint | INTEGRATION | cmd/internal/terminal/endpoint.go | New: serialized repaired backend, copied publication state and typed effects |
| InputWriter | INTEGRATION | cmd/internal/terminal/transport.go | New: one bounded FIFO for complete encoded input events, paste and query replies per child |
| ParentPresenter | INTEGRATION | cmd/internal/terminal/presenter.go | New: exclusive output, ordered selection/input/resize/release, coalesced frame publication |
| VT backend | INTEGRATION | third_party/vt/*.go | Modified local fork: screen/parser and negotiated child protocol state; no product routing policy |
| ContextWriter / Fake transport | INTEGRATION | cmd/internal/ttyio/{writer,fake}.go | New: bounded partial-write seam and controlled stateful double shared by parent and child adapters |

Profile/Frame/View/Render tests use literal data with no IO mocks. Backend and transport are integrations, not mislabeled pure models. Child lifetime remains in ptychild; endpoint owns terminal state and input serialization; presenter owns physical parent state. Terminal package must not import ptychild or couch/term product packages, avoiding a dependency cycle when Child adopts Endpoint in M3.

### Protocol and ownership decisions

1. Backend exposes a synchronous reply sink so Endpoint collects replies during one serialized operation without io.Pipe or a competing reply goroutine. All encoded output is enqueued to InputWriter under that same operation ordering. Raw output ingestion never writes to the parent. Query replies remain bound to the originating endpoint even while hidden or before UI attachment.
2. Endpoint Feed is called once by the child output pump before sink delivery; it updates mutable backend state and publication generation. The presenter requests a copied Frame only for visible output, at most once per frame interval. Hidden endpoints parse and reply without repeatedly allocating frame copies. Synchronized output retains the last publishable frame until end marker; after 150ms it publishes bounded recovery and records that timeout. Later input continues from the same parser; recovery is not parser reset.
3. `View` states are Ready, Presenting, Failed and Released. Selection and geometry changes enter Presenting; successful complete writes admit input and publish click geometry; partial/error writes enter Failed and admit no ordinary input. A released presenter never writes again. Product logical target can survive a panel, but admitted input focus cannot.
4. Diff rendering owns cursor/margins/styles. Child operations are interpreted only in virtual dimensions excluding chrome. Parent frames never replay child queries, mode setters or one-shot effects. Unknown strings/graphics remain framed and are ignored or explicitly rejected according to Profile, never raw-forwarded. Trusted chrome text accepts printable text, SGR and hyperlinks only; cursor/margin/erase strings cannot escape the composition rectangle.
5. Parent mode policy is separate from endpoint requests. Couch requests SGR all-motion to support chrome and reliable drag delivery, forwarding only events requested by the destination child. Pair term with a mouse-disabled child leaves parent mouse reporting disabled so Zellij retains native selection; when its child requests reporting, parent reporting can cover that request and endpoint encoding filters it. Do not impose blanket mouse tracking on shells and take away their enclosing Zellij selection.
6. An admitted mouse press binds its release/drag destination. A switch/panel/close cancels the gesture by delivering a matching release to the old live endpoint before admitting a new target, then suppresses remainder of the old gesture through physical release. Hover never goes to click-only or drag-only children. Bounds and row translation are applied before encoding.
7. InputDecoder uses the existing ultraviolet key/mouse protocol decoder behind bounded complete-frame detection. Product shortcut policy remains above endpoint delivery; complete key events are encoded from endpoint negotiation. Parent keyboard setup preserves Ctrl-Space/Delete/Return and Alt/Shift shortcuts. Paste is one ordered payload; terminal capability replies are consumed as parent replies and never treated as operator text. ESC timeout and split UTF-8 are tested explicitly.
8. Typed effects carry endpoint identity/output position. Title/cwd/bell/clipboard-write/Pair notification effects are emitted once during ingestion, not Snapshot/Select. Retain current clipboard-write policy; do not introduce clipboard reads. Read/query requests have a deterministic unsupported response. Oversize/truncated OSC effects are rejected rather than executing a truncated payload. Notification focus-at-delivery and outer delivery policy remain product-owned.
9. Profile declares implemented xterm-style text semantics, indexed/truecolor SGR, hyperlinks, application cursor/keypad, SGR/X10 mouse, bracketed paste, focus, supported Kitty keyboard flags, synchronized output and matching queries. No graphics/pixel or ambient clipboard-read support is advertised. Derive DA/XTGETTCAP responses and child environment/terminfo from the same profile; test the actual selected terminfo with shell/nvim/Zellij rather than claiming all of xterm without evidence.

### Initial operating envelope

These are implementation constraints and provisional performance targets, to be measured in M2 and enforced/checked in M4, not claims of existing performance:

| Resource/path | Budget and overload behavior | Basis |
|---|---|---|
| Typical workload | 80x24 and 240x80, 16 retained endpoints, one visible | Representative desktop terminal workload; measure both |
| Screen geometry | Existing absolute 262144-cell ceiling; validate before backend allocation or PTY resize | Existing project safety ceiling |
| Parser strings / params | 64KiB per string, 32 params; reject overflow effects and resynchronize at terminator | Reduce existing eager 4MiB parser per endpoint; bounded adversarial input |
| Grapheme / metadata | One incremental cluster capped at 256 UTF-8 bytes; title/cwd 4KiB, links 2KiB; over-limit input has explicit truncation/failure policy without partial escape execution | Bound untrusted retained payloads |
| History | At most 1000 lines AND 65536 cells; evict oldest lines; clear/shrink releases references | Preserve normal history without unbounded line widths |
| Keyboard stack | 16 entries per required buffer domain; bounded push/pop/reset semantics | Covers nested applications; no unbounded protocol stack |
| Child input | 128 packets and 1MiB queued; paste at most 1MiB; failed admission surfaced, no silent byte drop | Bounded interactive backpressure |
| Parent admission | At most 128 pending events, ordinary input paused during select/write; parent stop remains available | No input delivered to unpresented destination |
| Writes | 2s operation deadline, preserve accepted prefix; zero-progress/error fails transport, no blind replay | Explicit partial-write contract |
| Painting | Coalesce to latest dirty state at 16ms; synchronized recovery at 150ms; no periodic repaint when idle | Interactive targets and bounded sync withholding |
| CPU/memory | Measure 16-endpoint RSS and active output; target <512MiB representative aggregate, p95 input-to-visible <50ms and switch <100ms on development machine | Provisional workload targets; optimize or document measured breach before M4 readiness |

Geometry/payload theoretical ceilings are not a license to allocate that ceiling on every endpoint. Record measured empty/typical/history-saturated endpoint cost and enforce bounded retention. No new durable per-event logs or credentials. Transport diagnostics use existing retention; tests use synthetic bytes and disposable PTYs/data directories.

### M2 tasks and verification

- [x] **Repair backend with regression tests first.** Create third_party/vt from pinned module (including license/tests/go.mod), root replacement and patch ledger. In utf8/emulator, retain one provisional grapheme across Write boundaries; controls/cursor/resize finalize it. Add literal/every-split combining, ZWJ, variation selector/keycap, regional indicators, malformed UTF-8, right-margin width changes and bottom-margin scroll tests. In handlers/csi_mode/key/mouse/osc/screen, fix missing bare CSI-u restore, retained mode47 alternate contents, mode1047/1049 distinctions, Kitty flags/stack/reset and key encodings, tracking-mode replacement/suppression, hyperlink parameter order and semicolons, ordinary DSR and blink callback. Run local-module tests plus the existing qualification matrix unchanged. Expected: every executable case passes; integration obligations still honestly uncovered.
- [x] **Bound the backend and expose owned state.** Add narrow reply-writer, parser capacity/overflow, mode/profile snapshot and typed string/effect callback APIs. Tests feed oversized unterminated OSC/DCS, more than 32 parameters, excessive cluster/link/title input and repeated keyboard pushes. Require bounded retention and no truncated clipboard/notification effects. Test history clear/shrink releases references and enforce line/cell limits. Record exact limits and patches in PAIR_PATCHES.md.
- [x] **Implement Profile/Frame/View/Render from independent fixtures.** Add profile_test.go, frame_test.go, view_test.go, render_test.go. Assert literal A/B cell composition with separate save slots, margins, styles, wide cells and chrome; stale generation rejection; failure/release admission; drag binding/cancellation. Rendered bytes are consumed by a fresh emulator only as a supplementary integration check; literal frame assertions remain the independent oracle. Validate dimensions before allocating. Mutation-test removal of stale-generation and selection-admission checks.
- [x] **Implement ContextWriter and InputWriter with stateful doubles.** Add writer_test.go and transport_test.go. Schedule partial-success then continuation, partial-error, zero-progress, blocked reader, cancellation and close. Preserve packet byte ordering under concurrent query/paste/key producers, bounded queue accounting and joined teardown. Use a context-aware OS fd adapter (nonblocking write/poll with restored descriptor flags under exclusive ownership); no abandoned goroutine to simulate cancellation of an uncancellable Write. `go test -race ./cmd/internal/ttyio ./cmd/internal/terminal -count=1` must pass.
- [x] **Implement Endpoint transaction/publication.** endpoint_test.go exercises output before registration, hidden queries, clipboard/notification once-only behavior, synchronized begin/end/timeout, copied snapshots and clear/resize. Use a fake clock for timeout schedules and fake child transport with the same API as production. Query replies and operator packets must cross the actual writer seam. The parser remains coherent across split output, snapshot, input and resize. Reject invalid resize before changing backend epoch; acknowledge only after actual child resize succeeds.
- [x] **Implement ParentPresenter.** presenter_test.go forces select+typing ordering, delayed stale frame, partial parent write, blocked output, release racing paint/effect, resize failure, panel/no-input state, drag across selection and chrome clipping. It owns a context-aware writer and ordered event stream, coalesces frames, and commits admitted selection/hit geometry only after complete presentation. No exported generic raw-write door. Extend host fake behavior where necessary through the common transport seam.
- [x] **Implement InputDecoder and profile queries.** input_test.go covers every split of existing shortcuts, legacy/application cursor, CSI-u press/repeat/release, focus, bracketed paste with embedded escape bytes, mouse protocol variants, orphaned release and parent reply suppression. Feed production decoder into Endpoint.Send and assert literal wire bytes, not only round-trip equivalence. Add terminfo/profile query tests and record unsupported capabilities explicitly.
- [x] **Integrate qualification evidence.** Add executable adapter cases for M2's previously uncovered endpoint/presenter obligations without deleting original required IDs. Keep wrapper/live obligations uncovered until M3/M4. Update CLI report attribution to distinguish backend from integration cases and preserve bounded expected/observed evidence. Test no profile can qualify by omitting obligations.
- [x] **Measure, document and close M2.** Run local fork tests, focused normal/race, qualification probe, representative benchmarks and git diff --check. Update atlas/architecture.md and README terminal profile/maintenance documentation, issue Log and patch ledger. Commit and run `sdlc milestone-close --issue 255 --milestone M2 --verified '<measured evidence>'`; fix all blocking findings before M3.


## Chunk 3: M3 both compositors and wrapper

### Core concepts changed in M3

| Name | Kind | Lives in | Change |
|---|---|---|---|
| ptychild.Child | INTEGRATION | cmd/internal/ptychild/{child,fake}.go | Own one TerminalEndpoint from before output pump start, with direct context input/resize seam |
| OutputBatch | PURE | cmd/internal/ptychild/child.go | Add endpoint generation/position notifications while retaining capture/event metadata |
| Console | INTEGRATION | cmd/internal/couchtty/{console,console_menu,keyboard}.go | Presenter for all parent output/input admission; retain product policy |
| terminalMux | INTEGRATION | cmd/internal/termcmd/{run,presentation}.go | Same presenter/endpoint model, existing tab/chord policy |
| Wrapper delivery | INTEGRATION | cmd/internal/wrapcmd/{wrap,terminal}.go | Explicit delivered-stream observation and minimal framing-safe transforms |

- [x] **Wire endpoint before child startup output.** Modify ptychild.Start/NewFakeChild/pump/Feed/Close/Resize and PtyRunner.start as needed. One endpoint per child exists before the first byte; batch delivery only announces already-ingested state. Reuse direct PTY transport behind InputWriter; no recursive Child.Write calls. Keep raw ring and notification capture for their existing purposes, with no display authority. Update fake and literal lifecycle tests, including early query before UI attachment and blocked sink without deadlocking query service.
- [x] **Migrate Couch output and selection.** Replace Console.writeChild/takeOverScreen/writeOwn/paintNow parent paths with shared Frame/chrome submissions. Preserve RenderStatusRow chip spans and RenderMenuView cursor/extents through constrained styled rows. switchTo/forceSwitch and ExecuteConsoleOperation must report failed presentation; admitted focus/attention/switch history change only on success. Panel keeps logical active target but disables child input. All attach/exit/continuation/relaunch/park paths use the same transition door. Remove raw replay/nudge/hostScan and parent keyboard/mouse scanner authority from migrated paths. Tests observe semantic frames and state, not obsolete passthrough strings.
- [x] **Migrate Couch input/effects/resize.** Keep Interceptor and panel product actions, route remaining decoded events through presenter/endpoint. Preserve modifier stripping policy for wheel and exact reserved shortcuts. Test continuous Zellij drag forwarding during child redraw, chip/panel cancellation, hidden one-shot notification origin/focusedAtDelivery, no acknowledgement on failed landing, and layout resize while panel visible. Parent terminal setup/release belongs only to presenter.
- [x] **Migrate Pair term.** Refactor run.go writer path into presentation.go; newTab, switchRelative, removeTab, writeActive, inheritSize and closeAll use the shared lifecycle/order. Remove applyTakeover, hostScan/owed/row reservation/replay+nudge display authority. Retain tab labels, rename UI, pane title, role-aware shortcuts, shell/nvim dimensions and native Zellij selection when child mouse mode is off. Route output from every tab into its endpoint; only selected published frame draws. Test A/B saves/modes, repeated resize/switch, shell-to-nvim transitions and UTF-8 split across UI paint.
- [x] **Audit and minimize wrapper transforms.** Trace actual handleChunk -> notificationRewriter -> stdoutChunk -> stdoutPump delivery. Remove Codex-specific synchronized/focus/keyboard stripping that conflicts with the faithful endpoint; retain only documented notification normalization and product Return behavior. Terminal observer used for Return and query tracking consumes the stream whose state it claims to model; raw transcript capture remains separate. Preserve pending partial markers/UTF-8 at all splits. Existing no-regression Return/picker/notification tests plus composed delivery must pass. Update #254 disposition by evidence, not blanket close.
- [x] **Join actual production paths.** New integration tests transport real wrapper output through disposable Zellij into the shared endpoint and each consumer, using temporary sockets/config/data; cover shell, nvim, agent-like fixtures, parent queries, notifications, paste, focus and enhanced keys. No live operator-thread experimentation. Keep capability/origin/protocol assertions at the real seams. Run existing Couch attach/recovery/continuation acceptance, Pair shortcut, Lua and shell suites appropriate to these paths.
- [x] **Shadow sweep and M3 review.** Enumerate every parent write and child-mode reader in Couch/Pair; remove migrated raw bypasses and duplicate authoritative models. Update atlas/couch.md, atlas/architecture.md, README and linked issue dispositions for #207/#241/#252/#254 using actual tests. Preserve unrelated lifecycle #256 scope. Commit, run M3 milestone-close, fix blockers before M4.

## Chunk 4: M4 qualification and smoke-ready candidate

M4 ends with a tested candidate and operator instructions. The operator explicitly requested the manual smoke-test pause after M4 and before merge; that manual acceptance still gates #255 close and publication.

- [x] **Run sustained isolated conformance.** Add cmd/probes/terminalconformance and/or tests/terminal-conformance-test.sh using disposable PTYs/Zellij sessions and synthetic inputs. Exercise at least 30 minutes of continuous mixed Unicode/control output, panel/thread/tab switches, resize, detach/reattach and active drag gestures; compare expected screen/cursor/style and event destinations throughout. Log versions, duration, seed and operation counts, with bounded synthetic captures on failure. Add a shorter deterministic CI target and document a scheduled longer run. The harness must send real events through the production parent-input and child-output paths.
- [x] **Verify resource/latency targets.** Measure startup, active output throughput, idle CPU, 16-endpoint memory/history saturation, input-to-visible and switch latency at 80x24/240x80; compare baseline and candidate in the same isolated environment. Check frame coalescing, bounded queues and sync recovery under sustained load. Resolve material target breaches before calling the candidate ready; record results and limitations without claiming production acceptance.
- [x] **Final automated validation.** Run full Go suite, local fork suite, focused race/integration, relevant Lua/shell/shortcut/retention suites, profile/terminfo and native conformance. Mutation-check the causal Unicode/chrome isolation and mouse drag/mode regressions. No expected-failure labels may hide required production semantics. Record exactly which terminal programs/versions were tested.
- [x] **Prepare operator candidate and close M4.** Build isolated candidate binaries and provide exact launch/revert steps and a smoke checklist: agent and right-pane selection highlights while dragging; active output; Codex/Claude interaction; panel/thread/tab switches; long session; reattachment; normal/alternate screen and clipboard/paste. Keep current installation and sessions intact. Update issue/atlas/README qualification evidence, run M4 milestone-close and fix blockers. Leave operator acceptance/issue close/merge pending.
- [x] **Pause for operator smoke test before merge.** Report M1–M4 evidence, remaining limits and candidate command. Await the operator's result; do not merge, publish or claim the original symptoms resolved before that acceptance.

### M2 API refinements from source audit

`NewEmulatorWithLimits(w,h,Limits)` validates dimensions and parser/history/cluster/stack budgets. Preserve legacy NewEmulator/pipe API for upstream compatibility tests. `SetReplyWriter(io.Writer)` routes every reply/key/paste/focus through one internal writer, with first-error capture exposed by `TakeReplyError()`; Endpoint checks it after void input APIs too. `Mode(ansi.Mode)` and `KeyboardFlags()` expose values, not mutable mode maps. Store recognized modes only so hostile distinct DECSET values cannot grow a map indefinitely. Typed clipboard/notification callbacks remain synchronous and non-reentrant; endpoint closures add identity.

String overflow rejection uses parser capacity StringBytes+1 and rejects any dispatch exceeding StringBytes, including truncated longer strings; apply centrally to every string family before callbacks/logging. Retained-memory accounting conservatively includes parser capacity, both cell arrays/content/links, pending cluster, history, metadata and keyboard stacks; do not call it exact heap usage. History gets both cell and byte ceilings (4MiB), and clear/shrink zero retained references. Snapshot/frame and transport queues are accounted separately. Apply dimension checks to Resize as well as construction. Tests cover exact bound−1/bound/bound+1, long overflow plus recovery, and unknown-mode floods.

### M2 independent review corrections

The chunk reviewer identified two implementation blockers; both are resolved in this design before coding:

- **Descriptor ownership:** do not toggle O_NONBLOCK around individual writes or assume dup isolates it. A ttyio terminal transport acquires the relevant descriptors before either pump starts, caches/restores their original flags for its lifetime, and performs BOTH reads and writes through nonblocking syscall/poll adapters. Child.pump must use that Read adapter (EAGAIN is wait/retry, never EOF); Couch and Pair stdin pumps must consume the Host-owned reader when descriptors are acquired. Parent stdin/stdout may share an open-file description; all application readers/writers of those descriptors are inside this ownership interval. Restore flags on normal/failure release; cancel poll waits and join owned workers before release. Add a real-PTY test with concurrent read plus blocked/cancelled write, verifying continued reading, no busy loop, no false EOF and restored flags. In-memory tests inject the same context-aware transport interface. No uncancellable Write goroutine is allowed.
- **Independent renderer oracle:** use pinned `@xterm/headless@5.5.0` under `tests/terminal-oracle/` with package-lock integrity, a small JSON stdin/stdout driver, and an explicit npm-ci test target. It interprets actual renderer wire output and returns cells, colors/styles and cursor for literal assertions (including chrome, margins, wide CJK cells and A/B switches). It is independent of the repaired Go backend. M2's required verification runs this oracle before migration; Go-only tests do not substitute for it. M4 adds actual Zellij/nvim conformance and known Unicode-profile cases. The npm package version/repository/integrity were verified from npm metadata; Node/npm are available in the development environment. Keep this dependency test-only.

### 2026-09-15 — Native history migration finding

Frame-only cursor-addressed painting does not populate a parent terminal's normal-screen scrollback, even though the endpoint retains history. M3 must explicitly preserve native shell/Zellij scrollback through typed history publication, with identity/clear tracking and cell-derived serialization, before claiming the existing behavior is preserved. It must not restore raw-child replay as display authority. Add independent headless and actual Zellij history assertions, including repeated selection, clear, resize, alternate-screen transitions and reserved chrome. This elaborates the existing normal-history requirement; the M2 library alone does not discharge it.

### 2026-09-15 — M2 implementation evidence

The shared library, checked-in repaired fork, typed transport, bounded decoder/profile, immutable frames and presenter are implemented. Original backend predicates remain unchanged; qualification reports 81 pass and six M3/M4 consumer/live obligations uncovered. The complete root Go suite, focused shared-package race suite, local fork normal/race suite and independent renderer oracle pass. An inherited wrapper test had explicitly required the known split-ZWJ corruption; it now requires literal two-cell cluster content and complete snapshot equivalence at every byte split.

In-session review additionally caught repeated mouse-mode resets during painting, cancellation exposing an actor-owned result slice, hidden output replacing a pending visible refresh, and keyboard cleanup without a confirmed owned push. Regression tests defend these corrections. M2 still requires its SDLC boundary review; M3 production migration has not started.

### 2026-09-15 13:03 PDT — M3 typed native history and soft-wrap refinement

**Reason:** CUP/Frame rendering removes the raw-stream mechanism that currently populates native Zellij scrollback. Exporting independent physical rows would regress selection/copy of wrapped shell output. This refines the existing M3 preservation requirement, under the authorization to continue through M4; it does not add a new product or restore child control-stream replay. ARCH-DRY: both consumers use the same publication and serializer. ARCH-CONSTRAINTS: retained history, synchronized publication and rebuild work remain bounded by the endpoint limits.

**Concepts and ownership:**

| Concept | Kind | Location | Change |
|---|---|---|---|
| Row metadata | PURE | third_party/vt screen/scrollback | New incoming-soft-wrap flag and used-column boundary, moved with cells |
| HistoryCursor / HistoryRow / HistoryUpdate | PURE | cmd/internal/terminal/history.go | New monotonic identity, copied typed rows and append/rebuild decision |
| Publication | PURE | cmd/internal/terminal endpoint/frame | Frame and corresponding history window from one transaction |
| Native history serializer | PURE | cmd/internal/terminal/history_render.go | New cell-derived wire generation, shared by both products |
| History presentation state | INTEGRATION | cmd/internal/terminal/presenter.go | Extend existing parent-writer ownership and successful-write admission |

**Backend metadata and identity.** Maintain row metadata alongside each screen buffer: `Wrapped` means continuation of the preceding physical row; `UsedColumns` records the source row's logical content boundary when an early wide glyph forces a wrap. At an ordinary right-margin wrap, retain the entire source width, including actual trailing spaces; at an early wide wrap, exclude unused padding. Do not apply current unconditional trailing-space trimming to a soft row. Move metadata with full-row insert/delete/scroll operations, initialize newly blank rows, and define explicit LF, erasure and resize behavior against the independent oracle. CUP overwrite alone must not erase wrap identity. Width-changing incremental graphemes must restore/update metadata consistently with their cell rollback. Include metadata in retained-memory estimates.

Add monotonic history row IDs and a clear epoch without exposing mutable backend storage. `TotalPushed` never decreases; `ClearEpoch` advances on explicit history clear even when already empty. Eviction cannot renumber retained rows. Make oversized-row rejection observable as a loss boundary rather than silently joining unrelated logical lines. A concrete implementation may assign an ordinal to every append attempt and retain it on admitted rows; publication must then detect ordinal gaps rather than infer an always-contiguous interval from `Len`. Normal resize retains the original history cells/metadata and IDs; it must not masquerade as an explicit clear. ED2 appends the eligible visible rows with metadata before clearing; ED3 clears only the active history. Verify/reset-document RIS behavior explicitly rather than assuming it clears history. Audit `Screen.DeleteLine` admission: a region below screen row zero must not accidentally become native shell history merely because it spans the full width.

**Atomic endpoint publication.** Add an endpoint publication operation taking the last `HistoryCursor{ClearEpoch, Next}` and returning `Publication{Frame, HistoryUpdate}` under the existing endpoint transaction. History rows carry ordinal, copied cells and wrap metadata. A current cursor receives an append; epoch mismatch, eviction gap, rejection gap, new owner or width change requests a bounded retained-suffix rebuild. Carry an explicit truncation/loss indication internally so a retained suffix starts a new logical boundary instead of attaching to missing text. Synchronized-output withholding captures the history window together with the held frame; later live eviction must not change that held publication. Release/timeout presents the matching pair. Charge any additional held copy to the documented memory estimate; retention is capped by the same line/cell/byte limits and there is no unbounded publication queue.

**Selected owner and alternate screens.** Hidden endpoint output updates only its backend. On selection, publish that child's bounded retained primary history; do not append hidden output to the currently selected child's native history. Presenter tracks the installed owner, history cursor and width, and advances these only after the complete history-plus-frame write succeeds. Selection/clear/loss rebuilds erase parent saved history with CSI 3J before installing the selected suffix; partial write follows the existing failed-presentation path. Alternate-buffer output must never enter primary native history. Preserve the selected child's primary history across alternate entry/exit and test the actual parent buffer policy; a rebuild after leaving alternate mode must not duplicate it. Existing product routing still forwards mouse events only when appropriate and otherwise invokes native Zellij scrolling/selection.

**Native serializer and proven wire behavior.** Serialize only typed cells, styles and links, using the renderer's control-byte sanitization. Temporarily use a top-anchored scrolling region of at least two rows. Paint a history row at row 1 with autowrap enabled; trigger wrap into row 2 for a soft continuation, or LF for a hard boundary; CUP to the region bottom and LF admits exactly the staged top row to native history. Then repaint the viewport with CUP and autowrap disabled and restore owned parent state. Do not emit ED2 between history staging and repaint: the disposable xterm experiment showed it destroys the preserved wrap linkage. Ordinary visual damage can use full cell replacement without erasing row metadata. Bound output production and use the existing cancellable writer; avoid materializing an unbounded joined logical line or a second unlimited wire buffer.

Disposable evidence: `/tmp/pair255-wrap-proof.cjs` retained xterm wrapped history across CUP repaint; `/tmp/pair255-wrap-ed2.cjs` reproduced broken linkage with ED2. `/tmp/pair255-zellij-wrap-proof.py` on Zellij 0.45.1 exported two 12-column rows as one 24-character logical line in `dump-screen --full`, followed by a separate hard line; chrome appeared only in the final viewport. `/tmp/pair255-zellij-wrap-height2.py` repeated that result with one child row plus one chrome row. In this minimum geometry, row 2 is temporarily scratch space and chrome is restored before presentation completes; only row 1 enters history. A host with fewer than two total rows cannot use this staging procedure: follow the existing geometry/admission policy and test the explicit fallback rather than silently claiming preserved history. These temporary experiments are discovery evidence; replace them with committed reproducible oracle fixtures before M3 completion.

**Width and remaining proof obligations.** Rebuild at the current parent width by joining soft rows into bounded logical sequences and reflowing cells without splitting wide graphemes. Clipping historical physical rows is not an acceptable substitute because it loses copyable text. The history-to-first-visible-row continuation must be preserved. The experiments establish ordinary appended-history linkage, not arbitrary viewport wrap restoration: add a follow-up experiment that sets and clears every visible row's incoming-wrap flag through staging/LF, then restores cells, including a first visible row attached to history and the final child/chrome boundary. Do not declare that case solved until both xterm and native Zellij agree. Likewise, Unicode early-wrap padding, explicit trailing spaces, width-changing clusters and width-change rebuilds remain required proofs, not results established by the simple ASCII experiment.

**Implementation and verification sequence (within M3):**

- [x] Add literal backend row-metadata/identity tests first: normal and early-wide wrap, every split of combining/ZWJ/VS clusters at right/bottom margins, LF versus CUP, insert/delete/erase, resize, ED2/ED3, retention eviction/rejection, and primary/alternate isolation. Implement the smallest metadata and history-observation APIs those tests require.
- [x] Add pure history append/rebuild and reflow tests, then atomic endpoint tests with fake time for synchronized withholding, hidden output, selection and eviction while held. Assert copied publications remain unchanged after subsequent writes.
- [x] Extend the pinned headless oracle to retain scrollback and report wrap flags. Commit native-wire fixtures covering append, repeated full/incremental repaint, history-to-viewport continuation, width change, one-child-row geometry, chrome exclusion, clear and owner switches. Include the ED2 failure as a negative-control fixture.
- [x] Run the same typed-output fixtures through a disposable native Zellij session. Assert logical lines using `dump-screen --full`, exercise scroll-up/down and selection/copy behavior where available, and verify no operator session is touched. Add alternate-screen and Unicode/reflow cases before relying on the serializer in either consumer.
- [x] Integrate the shared publication path in both Couch and Pair term, retaining product policy. Run local fork tests, shared terminal normal/race tests, independent oracle, consumer integration tests and qualification; update atlas and record unresolved live-only obligations for M4. Do not cross the M3 boundary with a known copied-logical-line regression.

### 2026-09-15 13:09 PDT — Single-row history proof and oracle baseline precision

Follow-up disposable tests resolve the earlier less-than-two-host-rows exception: with one host row and no chrome, stage the typed row in row 1 and let a single LF (hard boundary) or autowrap via one dummy glyph (soft boundary) scroll that sole row into history. Do not add the two-row staging procedure's extra bottom LF. Both pinned xterm and native Zellij 0.45.1 preserved `ABCD` followed by `EFGH` as one wrapped logical history line using this procedure. Preserve the existing one-row term geometry; no history-dropping fallback is needed. Reproducible scratch drivers are `/tmp/pair255-xterm-oracle.cjs` and `/tmp/pair255-zellij-oracle.py`; commit equivalent fixtures in M3.

Direct-stream baseline comparison also distinguishes oracle behavior: xterm retains actual trailing-space cells in a wrapped row, while Zellij `dump-screen --full` trims those spaces when joining its textual output. The serializer must preserve typed cells; compare each terminal's native copied/dumped logical text to that same terminal's direct-stream baseline, rather than treating this existing inter-terminal difference as a new serializer regression. The remaining Unicode/reflow and arbitrary viewport wrap-restoration experiments are still open.

### 2026-09-15 — M3 transport and lifecycle adapter refinement

The consumer audit found three integration requirements the shared M2 library does not alone provide. Preserve existing one-row hosts by allowing a zero-row chrome layout; regular layouts still reserve one row. All layout-dependent clipping and resize arithmetic must use the admitted layout rather than hard-code `host.Rows-1`. Product diagnostics become typed status/panel content through the same presenter, with no raw stderr drawing into a child region.

`ptychild.Child` owns its Endpoint before starting output pumps, using the caller's pre-minted lifetime ID (or an internally minted unique ID). Add `Endpoint()`, `Endpoint.ID()`, typed `OutputBatch.Terminal`/`Err`, and a PTY-only resize callback distinct from `Child.Resize`, which updates endpoint geometry once. Use cached fd numbers for resize and raw-mode ioctls: calling `os.File.Fd()` after acquiring nonblocking transport can change descriptor flags. Compile the custom terminfo entry into the versioned runtime bundle during generation, and set child TERM/TERMINFO consistently before spawn; no runtime `tic` subprocess.

Separate bounded output publication from the PTY reader so an early blocked UI acknowledgment does not immediately prevent later origin queries from being ingested and answered. The publication callback takes a context and must honor cancellation while enqueueing/waiting for UI acknowledgment; use the same path in fake and real children. Bound queued publication by 128 batches and 1MiB including in-flight data, backpressure explicitly at the bound, and expose a completion barrier for tests/final-output delivery. Cancellation joins the worker instead of abandoning an uncancellable callback. Raw capture remains separate from displayed frames.

Process EOF ends input transport but does not destroy the last readable terminal publication. Distinguish terminal state disposal from input ending; the consumer flushes final presentation, selects a survivor/panel, retires the origin, then disposes the endpoint. Canceling a captured gesture on an input-ended child must not make a healthy successor unusable. Add real/fake conformance tests for early queries before registration, a blocked publication callback, EOF with final output, and cancellation while UI delivery is waiting.

`hostty` exposes context-aware writes and the acquired input reader. Acquire descriptors before read pumps; migrate every real reader to that adapter; cancel/join IO before restoring flags. Fake wrappers that override writes must also override contextual writes so interpreter/failure fixtures remain meaningful. Verify blocked reads/writes, repeated resize without loss of nonblocking state, and teardown in disposable PTYs. This refines the approved ownership model and preserves existing product behavior; it does not move generic process lifecycle authority from #256 into #255.

### 2026-09-15 13:17 PDT — Durable native-history experiments and implementation constraints

**Reason and delta:** preserve the subsequent M3 discovery results before context loss. Reproducible prototypes now live in `tests/terminal-oracle/discovery/`, with commands, oracle versions and explicit limitations in its README. This is preparation only; no M3 production path is enabled. All six documented Python probes were rerun from their repository locations successfully. Typed staging, dirty rebuild, width reflow and one-row probes contain executable direct-baseline assertions.

The experiments resolve the ordinary viewport wrap-setting/clearing question but reveal a terminal difference: autowrap sets a soft link in both oracles; LF clears an existing link in xterm but does not canonicalize an existing Zellij row. DL+IL at a target row below row 1 replaces it with a hard-boundary row in both. The viewport probe sets/clears links, preserves the history-to-first-visible-row connection, and diagnostically scrolls only child rows into history to expose the resulting logical lines; chrome remains outside admitted history. Avoid DL at row 1 when preserving existing history, because Zellij may admit that row. These results supersede the earlier assumption that LF alone suffices to clear arbitrary visible wrap links.

Clean typed-cell history staging now matches the direct-stream baseline in complete xterm cells/wrap flags and native Zellij dumped text at widths 4 and 6, including early-wide gaps, printed trailing spaces and combining-text baseline behavior. Clean scratch rows must be cleared before acquiring their soft link. ECH preserves wrap flags, but Zellij stores erased padding as spaces; using it in an early-wide soft gap introduces a copied extra space. EL2 removes this padding in Zellij but clears xterm's incoming wrap flag. Therefore the incremental serializer needs a checked fast path and a bounded typed rebuild when a dirty wrapped top-row gap cannot be preserved. The rebuild prototype starts from old history and a dirty wrapped viewport, uses CSI 3J and full-region IL to discard that viewport without admitting it, and regenerates the retained typed history plus frame; both oracles match the direct baseline exactly at the tested widths. Measure fallback frequency and latency in production before claiming the interactive targets. This is typed state serialization, never replay of child control bytes.

The width-reflow prototype compares direct output resized from 4 to 6 columns with regenerated typed logical lines at width 6; complete xterm rows and native logical text agree for the fixtures. The one-host-row prototype independently asserts that LF/autowrap can admit the sole staged row, preserving a soft history connection without a two-row region or an extra bottom LF. Retain the earlier one-row support decision.

Blank provenance can use `uv.Cell{Content:"", Width:1}` instead of a separate per-cell bitmap: UV Fill/Set/Insert retain that distinction from a printed space. However, simply changing `blankCell` is insufficient. Initialization/reset/resize/erase must use the representation consistently; UV's legacy String/Render collapses empty-content positions, so those compatibility views need explicit space normalization; partial-wide overwrite cleanup internally creates ordinary spaces and needs corresponding local handling. Keep this change scoped to preserving the actual metadata needed for copy/reflow, with literal blank-versus-printed-space regression tests. The discovery Go prototype records these constraints; it does not modify the fork.

Compare each oracle to its own direct-stream baseline at the same history/viewport boundary. Zellij's full dump serializes those areas separately and trims spaces per physical viewport row; after rows merge into history, internal printed spaces remain. The diagnostic viewport flush is therefore necessary to inspect a history-to-visible soft link through dump text. Native combining-mark limitations and headless default-width differences remain visible baseline differences, not proof of broader Unicode conformance. Dump output also does not replace interactive selection/copy conformance. M3 must promote the prototypes into tests of the real backend publication and serializer, adding arbitrary cursor edits, style/link behavior, alternate transitions, synchronized/cancelled writes, resource bounds and actual selection before closing the milestone.

### 2026-09-15 — M2 cancellation commits before subsequent operations

Second M2 review disposed BR6–BR8 but reproduced BR9: a failed resize kept the old gesture admitted after delivering its synthetic release. Gesture cancellation commits independently of selection or resize success. Enumerate release, failure, selection, panel, negotiation reconciliation and resize, with tests for subsequent failure, interrupted delivery, retry, physical release and a fresh press. Failed resize preserves geometry but cannot restore a canceled gesture. Track an admitted release until delivery settles; retry waits for that delivery rather than enqueueing another release. ARCH-ORDER / ARCH-PURPOSE: one cancellation transition serves the whole caller family.

### 2026-09-15 — M2 discovery artifact ownership

The newly preserved native probes must scope their temporary configuration, sockets, logs and scripts to one invocation. The shared driver removes its directory after process and PTY teardown, including setup, launch and oracle failures. No implicit diagnostic retention is supported; bounded returned/stdout evidence is the diagnostic artifact. Add success/failure cleanup regressions. ARCH-FUNERAL applies to discovery tools as well as production paths.

### 2026-09-15 — M2 parent restoration after interrupted output

BR11 independently reproduced an interrupted paint leaving autowrap disabled after successful Release. Enumerate parent mutations: setup mouse tracking/SGR, focus, paste and owned Kitty push; renderer cursor position/visibility/style, origin, margins, autowrap, rendition and OSC8 link. Release aborts partial control framing, restores normal origin/full margins/autowrap/default rendition/no hyperlink/default cursor style/visible cursor, disables owned input reporting, and pops only a confirmed owned keyboard push. Screen contents and prior cursor position are not reconstructed; permitted title/clipboard effects persist intentionally. Independent xterm tests exercise every accepted ASCII output prefix through production Select/Release and subsequent ordinary text; existing keyboard ownership tests defend the conditional pop. ARCH-ORDER: successful cleanup must establish the documented parent state after any partial write.

### 2026-09-15 — M2 Unicode cell coherence

BR12 reproduced legal zero-width input creating nonempty Width0 cells and permanently failing presentation. Preserve the strict cell invariant: zero-width continuation cells have no content. Contiguous marks, joiners and selectors extend their printable grapheme; controls seal that cluster. An orphan zero-width rune is consumed without cell, cursor or pending-wrap mutation, matching measured native Zellij behavior. Pinned xterm uses a different zero-width-cell representation, so do not claim identical orphan semantics. Apply the same policy to StyledRows. Use complete grapheme segmentation for Frame validation and UI text; ANSI DecodeSequence's ASCII fast path is not a complete-grapheme validator. Require every-byte-split backend and production presentation/input regressions plus independent rendering of intact ASCII-base combining clusters. ARCH-PURPOSE / ARCH-DRY: backend, frame and serializer must agree on what one cell contains.

### 2026-09-15 — M2 closed; M3 begins

M2 SHIP at window29101ebf..214d43e8 after six boundary rounds, BR6–BR12 disposed. Shared terminal, maintained backend, bounds/profile, independent renderer and failure/Unicode regression suites are implemented. Root full Go suite passed; the reviewer's separate full run hit unchanged `TestParkCoordinatorConstructorDoesNotQueryPairSession` at its100msdeadline, then ten isolated repetitions passed. Retain that distinction in final evidence and verify the full suite again after M3/M4. M2 closes library work only; production migration, native history and sustained display/selection acceptance remain pending.

### 2026-09-15 14:29 PDT — Primary viewport reflow preserves native logical lines

**Reason and delta:** production history-oracle tests found that merely clipping/padding the backend viewport on width change left underfull consecutive soft rows that the common xterm/Zellij wire could not serialize without a copied extra space. Disposable native resize baselines show that Zellij reflows the visible primary logical lines independently of already retained history: at one column, `ABC` has history `A` and visible `B`/`C`; widening to eight leaves history `A` and joins the viewport to `BC`. Implement that behavior in the backend, rather than introducing renderer-specific terminal profiles (ARCH-PURPOSE / ARCH-DRY). Reflow respects hard boundaries, explicit spaces and early-wide gaps, remaps the active/saved cursor and pending wrap, and admits rows displaced by narrowing through existing bounded history. Retained IDs and source cells remain stable except for those new admissions; alternate-screen rows retain their clipping behavior. Sparse reflow staging is bounded by old cells and rows, while the final buffer obeys the existing geometry cap.

For the physically unrepresentable one-column/wide-glyph case, preserve the indivisible source glyph privately while exposing a valid blank viewport cell; overwriting clears that retained source, scrolling transfers it to typed history, and widening restores it. The one-column serializer omits indivisible historical glyphs that cannot fit; the retained publication remains intact and width changes rebuild the full glyph. This is an explicit native-copy limitation at width one, not a claim of native Zellij's offscreen-wide-cell equivalence. Normal widths of at least two columns retain complete glyphs. Literal backend tests cover growth/narrowing, hard and soft boundaries, early-wide gaps, cursor continuation, clipping/restore/overwrite/admission; production pinned-xterm and native-Zellij tests cover width-one/three/four growth and exact copied logical strings. The earlier native combining and interactive-selection caveats still apply.

The initial owned full-window publication is intentionally uncached. Saturated-history benchmarks on this machine measured approximately 1.18–2.38ms and 4.69–12.79MB allocated per publication, depending on geometry and whether a new generation must be captured. A representative sixteen-endpoint 240×80 saturated-history probe retained about 287.5MB live Go heap after GC with backend, published history/frame and caller frame (not caller history or RSS). This remains below the provisional 512MiB representative target; M4 must measure the actual Presenter/host workload before introducing cache complexity or claiming the aggregate target satisfied.

### 2026-09-15 14:35 PDT — Remove unnecessary ordinary history retention

**Reason and delta:** the full caller-publication probe reached 559,906,816 bytes maximum RSS, exceeding the representative 512MiB target. Avoid retaining a duplicate history window outside actual synchronized holds and EOF: ordinary Publication returns one owned backend history snapshot, while the endpoint keeps its existing published frame; holds/EOF retain frozen history and clone on access. This preserves atomic publication and caller immutability without a shared mutable cache (ARCH-PURPOSE). The repeat sixteen-endpoint 240×80 saturated probe measured 287,526,792 bytes live heap and 427,704,320 maximum RSS. The qualification revision records commands/evidence and the remaining whole-application M4 obligations. Native ED2 baseline tests also require preservation of blank hard separators and printed-space soft rows through the last printed row; update bounded admission accordingly, explicitly retaining the known xterm ED2 difference.


### 2026-09-15 — M3 implementation complete, boundary review requested

Both production consumers, wrapper passthrough, typed native history, bundled profile and shadow-authority removal are implemented. Detailed M3 work items are checked for completed implementation/verification; the issue-level M3 row remains owned by the SDLC review result. Native held-drag highlight and copied-text checks pass in direct and wrapped Zellij. Parent failure/leave diagnostics are deferred until joined release and raw restoration; release failures affect the exit status. M4 sustained runs, final performance comparison, persistent native reattachment and the operator smoke pause remain outstanding. No deployed binary or operator session was changed.

## Revisions — 2026-09-15 M3 review class sweep (BR-13–BR-15)

The first M3 boundary review returned REWORK. Complete these corrections before retrying the boundary; M4's long runs remain gated.

- **Terminal cell attributes:** erased/unprinted blanks retain rendition independently of printed provenance. Sweep viewport, retained history, incremental append, rebuild, and alternate-screen serialization; compare interior and trailing erased backgrounds through the independent wire oracle. ARCH-PURPOSE.
- **Terminal resource ownership:** every accepted child has a last owner. Final output drains before deselection/retirement; retirement is followed by disposal. Enumerate natural exit (including last visible child), park/relaunch, failed attachment/acknowledgment, replacement, and Console teardown. Tests must prove endpoint/publication disposal and joined workers, not only absence from a pane map. ARCH-FUNERAL / ARCH-ORDER.
- **Build-time dependency:** inject a TerminfoCompiler that writes entries into caller-owned temporary storage. Unit tests use a stateful filesystem fake for platform-specific entry directories, failed/empty compilation and preservation of prior output. Production uses tic; separate native conformance checks decode its generated entry with infocmp. Temporary compiler/staging storage is removed on success and failure. ARCH-MOCK.

M4 harness preparation additionally removes the test runtime's growing action transcript, records bounded once-per-minute progress, and compares term cells/styles/cursor plus normal/alternate buffers. These are qualification changes, not evidence of completed sustained tests.

## Revisions — 2026-09-15 M3 geometry and completion evidence (BR-16–BR-18)

M3 round2 accepted BR-13–BR-15 and found two new blockers plus one stale guide. Geometry and chrome validation must precede allocation/clone in both resize entrypoints, with an audit of Select, Panel and UpdateChrome; invalid dimensions must leave admission, child geometry and parent bytes unchanged (ARCH-CONSTRAINTS / ARCH-SECURE). Teardown probes must join the corresponding Run/Release/closeAll completion before inspecting restored state or writing simulated shell output. Sweep both consumers, including mouse/keyboard cleanup assertions, rather than treating a previously emitted reset sequence as an acknowledgment (ARCH-ORDER / ARCH-PURPOSE). Update the harness guide's removed filtering path and telemetry row.

## Revisions — 2026-09-15 M3 closed; M4 continuous-history policy

M3 SHIP at `c5ec1728..e1a18517` after three rounds; BR-13–BR-18 are disposed. Both consumers and wrapper migration are reviewed. M4 remains qualification work, with operator smoke acceptance and merge explicitly pending.

The saturated-history preflight demonstrates excessive output when retention evicts an old prefix: every changed FirstID currently rebuilds the whole exported window. Refine the export policy for continuous presentation: if owner, width and clear epoch are unchanged and the delivered NextID still has contiguous coverage, append only new rows. The physical parent may retain previously delivered older rows under its own scrollback policy, as with normal terminal output. Endpoint storage/export remain bounded; switch/rebuild restores the retained suffix, and clear or missing coverage must rebuild. This preserves logical copy text while avoiding full-window redraws on routine prefix eviction. Add independent/native regression cases for overlap append versus an actual coverage gap; remeasure the end-to-end budgets without subtracting interpreter cost. ARCH-PURPOSE / ARCH-CONSTRAINTS.

M4 also measures actual sixteen-tab compositor RSS in addition to the backend/publication probe. Sample after each saturated tab and retained-screen switch; label maximum observed wrapper RSS explicitly, excluding child/helper/oracle processes and not claiming an OS peak. Native reattachment must prove an unchanged agent PID plus in-memory counter, then held-drag highlight/copy after reattach.

## Revisions — 2026-09-15 M4 measurement corrections

Extended qualification exposed an observation race in the soak receipt barrier and an idle-output backpressure error in the timing harness. Selected-screen receipt observation, followed by completed presentation and frame comparison, is the readiness condition; queued-publication flush alone cannot acknowledge concurrent ingestion. Idle performance samples must drain the PTY continuously while retaining only bounded interpreted screen state. Preserve initial failed/interrupted evidence and rerun from fresh binaries; do not count partial durations or subtract independent-interpreter time from the latency budget. Redundant default style/link reset sequences may be removed only with explicit preservation tests for colored/linked cells, erased backgrounds, soft wraps, and interrupted output.

## Revisions — 2026-09-15 M4 measured performance disposition

All twenty baseline/candidate timing sessions completed on frozen source, with five trials per geometry and every sample retained. Candidate pooled p95 post-history input is 21.051ms at80×24 and24.011ms at240×80, meeting the50ms target. Post-history switching is101.060ms and115.416ms, exceeding the provisional100ms target by1.060ms and15.416ms. Baseline is187.129ms and215.472ms. No oracle time is subtracted. The seven240×80 switches over100ms all occurred in one trial with identical44,156-byte history rebuilds; the other four trials had none. Dense retained-text output and independent interpretation account for substantial cost, with additional timing variation not causally attributed to one subsystem.

Disposition: preserve the provisional threshold and report this measured miss as an explicit M4 limitation for boundary review and operator smoke, rather than silently treating it as a pass or changing history/copy semantics to meet a synthetic timing number. The earlier budget table permits documenting measured breaches; candidate preparation does not authorize rollout. A larger history serializer redesign is not justified by this qualification alone. Operator acceptance of switching responsiveness remains part of the pre-merge smoke. Source and installed-session behavior remain frozen during the sustained tests.

## Revisions — 2026-09-15 M4 shutdown qualification blocker

Pair term completes its thirty-minute run. Couch completes all operational assertions but fails its final shutdown with exit1. Stop M4 closure, capture the actual terminal diagnostic, reproduce the teardown boundary and fix the responsible class before qualification resumes. Preserve the failed long-run log; do not weaken the required exit code or label it a successful soak.

## Revisions — 2026-09-15 M4 shutdown result ownership

Short real-PTY runs reproduce the long-run exit1 with an empty teardown diagnostic and a Presenter write failure whose sole cause is `context.Canceled`. The Console Stop cancels its operation lifetime before the Presenter owner is released; whichever ready select branch wins must not decide success. Joined teardown now also observes Presenter failure, suppressing only new cancellation-only error trees during shutdown. Already-latched live failures, joined real host errors, deadlines and cleanup errors remain failures. A forced blocked-paint regression proves both expected cancellation and genuine-failure outcomes; live cancellation remains latched. Changes are limited to Couch's Console/terminal error handling. Restart the Couch thirty-minute run and rerun affected full/native suites; retain the completed Pair run and previous performance evidence with the exact source delta disclosed.

## Revisions — 2026-09-15 M4 notification transport ownership

**Reason:** strict native qualification captured `EMIT-fail ... resource temporarily unavailable`. Both `wrapcmd.emitOuter` and `notifycmd` independently open Zellij's outer TTY. Retrying that write does not prevent insertion inside Zellij's UTF-8/control output. Remove both bypasses; preserve best-effort attention without adding another screen authority (ARCH-PURPOSE, ARCH-ORDER).

**Architecture:** hook commands send bounded messages to the live wrapper over a private Unix datagram socket. The wrapper's one output owner serializes canonical OSC777 into its existing Zellij pane stream at complete UTF-8/control boundaries. Zellij0.45.1 converts this to OSC9 `pair: <message>`; the shared Endpoint normalizes that representation into the existing typed Pair notification effect. Zellij remains its outer connection's sole writer. Persistent wrappers retain the broker across client detach/reattach. No outer-TTY fallback.

Native evidence: `/tmp/pair255-notify-grapheme-split.log` compares delayed chunks at widths20/4 through native dumps and independent xterm cells/cursor for acute/ZWJ/VS/keycap/regional indicators. Insertion produces the same native result, including existing Zellij combining limitations. `/tmp/pair255-notify-maxbody.log` proves full4096-byte ASCII and multibyte bodies survive as4107-byte OSC9 envelopes. This qualifies the pinned native path, not every terminal emulator.

### Core concepts for this revision

| Name | Lives in | Status |
|------|----------|--------|
| Notification framing state | `cmd/internal/wrapcmd/notification_output.go` | new |
| Ordered rewrite events | `cmd/internal/wrapcmd/notification_rewriter.go` | modified |
| Mapped Pair notification | `cmd/internal/notifyosc/notification.go` | modified |
| Broker address | `cmd/internal/notifytransport/address.go` | new |

Framing state records only UTF-8/control completeness, never screen state. Reuse the bounded x/ansi parser with strict UTF-8 validation; ambiguous malformed framing permanently disables generated insertion for that stream while preserving original output. Feed the actual normalized output, not observer input. Ordered rewrite events preserve the position of native/progress observations relative to passthrough across arbitrary read partitions. Retain existing lifecycle deduplication/rate limits.

Broker address derives from the exact existing `PAIR_PAIR_WRAP_PID_PATH` binding and strictly positive wrapper PID, using a short hashed name under a UID-private0700 temporary directory. It does not reconstruct tag/scope paths or add a durable artifact family. Binding publication occurs after listener readiness and before child hooks can run. Only the owning wrapper can remove its socket; stale socket cleanup must verify type/ownership and never unlink another live PID's socket. A reused PID in the same exact binding denotes the current wrapper, consistent with existing image-capture PID semantics. Socket storage is ephemeral, removed on joined normal teardown; crash residue is inert and reclaimed only with verified dead-owner evidence. All pathname construction stays in this package.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Notification broker | `cmd/internal/notifytransport/transport.go` | new | Unix datagram socket and existing exact PID binding |
| Wrapper output owner | `cmd/internal/wrapcmd/wrap.go` | modified | child PTY, output batching and broker admission |
| Hook adapter | `cmd/internal/notifycmd/run.go` | modified | exact binding lookup and broker sender |
| Zellij notification adapter | `cmd/internal/terminal/endpoint.go` | modified | registered OSC9 handler |

Limits: sanitized body4096 bytes; datagram receiver uses4097 bytes and rejects oversize rather than truncating. Broker queue32, nonblocking admission with explicit diagnostic on overflow; sender has a200ms deadline and existing exit0-with-warning failure behavior. Main wrapper loop alone consumes messages and emits output. Pending unsafe-boundary insertion is bounded to32 notifications with2s expiry; drop/log at expiry or incomplete EOF, never inject CAN/ST or mutate child bytes to manufacture a boundary. Hook messages bypass turn-completion inference but share output serialization; they must not close a lifecycle generation. Listener Close unblocks and joins receiver; early startup/exec failures also clean up. No unbounded goroutine per notification or notification replay on reattach.

### Implementation and verification

- [x] Add pure mapping/framing/address tests first. Exercise every split of UTF-8, CSI, OSC, DCS, APC/SOS/PM, ST halves, CAN/SUB, huge incomplete controls, invalid continuation bytes and EOF. Original passthrough remains byte-exact; inserted notifications occur only at independently checked boundaries. Ordered event tests prove native/progress positions do not depend on reads. Test4096-byte body plus mapped prefix without changing generic backend limits.
- [x] Implement `notifytransport` with real private-socket integration tests for exact binding isolation, bounded receive/admission, stale/malformed PID, oversize, unavailable reader, close/join, startup failure, ownership-safe cleanup and persistent sender lookup. Use a stateful broker fake at the CLI seam, and actual sockets for transport conformance. Migrate `notifycmd/run_test.go`, preserving legacy argument forms and warnings. Update the artifact source-coverage assertion to the existing exact PID binding.
- [x] Implement wrapper framing/output owner and ordered rewriter, then broker startup before child creation and consumption in `masterPump`. Remove `writeTTY`/outer-TTY emission. Sweep wrapper lifecycle/idle/progress tests to observe production stdout. Handle partial output writes explicitly: continue accepted-prefix writes, latch terminal failure on unrecoverable error/zero progress, and report failure rather than claiming complete notification delivery. No retry from byte zero after partial acceptance.
- [x] Register the mappedOSC9 handler in Endpoint using `notifyosc` shared decoding, preserving ordinary non-Pair OSC9 handling. Pin `host_notification_protocol "osc9"` in Zellij config. Keep domain-specific mapping out of the generic backend fork. Add both consumers' hidden/focused effect tests and no-replay tests.
- [x] Update strict native fixtures: actual wrapper -> Zellij -> Endpoint -> Presenter, hidden origin attention and focused suppression, max body, delayed split Unicode, broker CLI message, and persistent reattach with unchanged PID/nonce/counter and no outer-TTY sidecar rewrite. Every fixture must join Run and assert exit0; private environment clears inherited diagnostic bindings. Rerun native race tests and affected package/full suites.
- [x] Update README, atlas notification routes, runtime bundle and qualification evidence. Preserve earlier failed logs. Attribute completed synthetic30m soaks to their immutable binaries; these do not qualify the new wrapper broker. Add sustained native notification/output stress for the changed path, then rebuild isolated smoke candidate. Re-run M4 SDLC boundary review and fix blockers. Pause for the operator's long-running visual/held-drag smoke before merge.

Commands: `go test ./cmd/internal/notifyosc ./cmd/internal/notifytransport ./cmd/internal/notifycmd ./cmd/internal/wrapcmd ./cmd/internal/terminal ./cmd/internal/couchtty ./cmd/internal/termcmd -count=1`, followed by affected `-race` suites and the existing native qualification flags. Run `go test ./... -count=1`, runtime-bundle drift checks and scoped whitespace verification before the M4 boundary. Success requires behavioral assertions, not merely process startup or absence of stderr.

## Revisions — 2026-09-15 M4 candidate verification complete; boundary pending

Both thirty-minute synthetic production runs pass with exact source attribution
in the qualification record. The final notification revision passes the full Go
suite, fork suite, wrapper full race, consumer race suites and three strict native
repetitions (96 actual hook cycles). Broker publication/removal share a stable
UID-private lock, and the address derivation is pure. Hook messages do not refresh
the slug or change turn lifecycle. Unsafe EOF is passed through exactly while
pending notifications are dropped; output failure joins the reader and kills the
owned child. The historical Codex recording ends inside CSI, and its regression
now proves deferred insertion until the sequence completes.

The isolated smoke candidate is rebuilt and validated by private `couch --list`;
Pair SHA256 matches the final native-tested binary. M4 boundary review, operator
smoke, issue close and merge are still pending. The measured100ms switch-budget
exception remains explicit; no additional threshold or visual acceptance is
claimed. Earlier long-run/performance evidence retains its immutable binary
attribution rather than being relabeled as measurements of this final build.

## Revisions — 2026-09-15 M4 review BR-19 / BR-20

The first M4 boundary returns REWORK with two Important findings. Ownership rule:
the wrapper owns its socket, but reclaimability must not depend on another
subsystem retaining the PID binding. Normal close removes admitted owned inodes
under the namespace lock; after crash, independently deleted bindings, artifact
GC or failed close cleanup, the next namespace admission reclaims only strict
Pair socket names whose same-UID owner PID is provably dead. Live, unknown,
foreign and symlink entries are preserved. Sweep/admission is bounded (at most
1024 entries per scan, with explicit capacity refusal rather than unbounded
growth); socket identity is independently readable from its name. Sidecar
cleanup remains its existing owner's responsibility. The lock inode survives
broker close and is never unlinked as a contention workaround.

Namespace rule: socket paths, publication/cleanup locks and dead-owner sweeps
all use the same injected root. Default production root remains UID-private;
`PAIR_NOTIFY_SOCKET_DIR` selects an absolute private root for subprocess
conformance and the smoke launcher. All real-broker fixtures, including wrapper
startup, CLI, PTY and native Zellij, supply private short `/tmp` namespaces and
propagate them to sender and receiver. Tests may not acquire production locks or
sweep production sockets. Cross-namespace tests must prove a held lock has no
effect on another namespace and cannot route a message across that boundary.

| Name | Kind | Lives in | Status |
|------|------|----------|--------|
| Socket address/owner identity | PURE | `cmd/internal/notifytransport/address.go` | modified |
| Notification framing/ordered events | PURE | `cmd/internal/wrapcmd/notification_output.go`, `notification_rewriter.go` | unchanged in this correction |
| Mapped Pair notification | PURE | `cmd/internal/notifyosc/notification.go` | unchanged in this correction |
| Notification namespace and reclamation | INTEGRATION | `cmd/internal/notifytransport/address.go`, `transport.go` | modified |
| Wrapper/hook and native fixtures | INTEGRATION | `wrapcmd`, `notifycmd`, `couchtty` tests | modified |

Add failing regressions for dead socket after binding deletion, failed cleanup
followed by admission, live/foreign/symlink preservation, bounded capacity, and
isolated contention/routing. Sweep every real-socket test seam, not just the
review's named contention test. Re-run broker/wrapper/consumer affected tests,
full repository verification and freshly rebuilt strict native conformance.
Retain earlier long-run and timing source attribution. Refresh the isolated
smoke candidate, commit corrections and rerun the same M4 gate. Operator
acceptance, issue close and merge stay pending. ARCH-FUNERAL / ARCH-SECURE /
ARCH-MOCK / ARCH-PURPOSE.

## Revisions — 2026-09-15 M4 review BR-21: conformance evidence lifetime

Round2 disposes BR-19/BR-20 but finds implicit retention in both successful native
reattachment and failed native fixture evidence. Apply the existing M2 discovery
ownership rule to the entire conformance writer family (ARCH-FUNERAL / ARCH-PURPOSE):
all configuration, sockets, captures, receipts and oracle JSON belong to one test
invocation, and are removed after joined process/PTY teardown on success and
failure. Diagnostics retained in test output must be bounded; no automatic
unbounded `/tmp/pair255-native-*` evidence family remains. Existing historical
qualification logs are preserved as explicitly recorded session evidence.

Enumerate sibling native/PTY/discovery/performance writers; reuse invocation
storage rather than inventing a second persistent artifact lifecycle. Add cleanup
regressions for both success and failure, verify bounded diagnostics, and rerun
native/nvim/broker race checks using the frozen candidate. This changes test
ownership only; production source, binary hashes and prior soak attribution stay
unchanged. Commit the corrections and rerun M4 review. Operator smoke/merge remain
pending.

BR-21 implementation refinement: remove the failure-artifact family entirely;
failure output retains at most4096 raw bytes per reported capture, escaped by the
test logger. Successful reattachment comparisons use `testing.T.TempDir` scratch
allocated before process cleanup registration, so joined teardown runs first.
The same lifetime holds on a failed invocation. Diagnostic reads/logging cannot
abort mandatory teardown. Private native/PTY root removal errors are reported.
Sibling discovery/performance tools already use `TemporaryDirectory`; existing
PTY socket roots have registered cleanup. No new production surface is introduced.

## Revisions — 2026-09-15 M4 closed; pre-merge smoke handoff

M4 SHIP after three boundary rounds, window `12c301ac..ea98f0c7`; all blocking
findings addressed. The advisory documentation-route finding is corrected across
atlas diagnostics/hook instructions, wrapper overview and compatibility shim.
Rule: current notification descriptions name the wrapper broker and serialized
pane output; outer-TTY references describe compatibility metadata only. These
post-review edits are documentation/comments only. The qualified candidate keeps
its recorded pre-comment source and binary hashes; no behavior has changed.

The prepared operator command/checklist is `000255-terminal-smoke.md`. Pause now
for sustained display/held-selection smoke and responsiveness acceptance. Original
symptoms, issue closure and merge remain pending that result.

## Revisions — 2026-09-15 operator acceptance and merge authorization

Operator accepted the isolated candidate, tested the rebuilt branch in the usual
`~/workspace/pair` setup, then explicitly said to consider smoke passed and merge,
with later discoveries fixed forward. Record this as operator acceptance without
inventing a session duration or claiming every possible interaction was exercised.
The Alt+N confirmation report remains unresolved and is tracked in #259, with no
source changes bundled into #255 for that report. The old mouse-trace warning was
traced to content/metadata divergence after a legacy writer; old evidence was
preserved and a fresh trace path supplied. Existing performance exceptions remain
visible. Proceed through issue close and publish gates.

## Revisions — 2026-09-15 whole-issue close BR-23: enforce native CI coverage

The whole-issue close review disposed BR-22 but found that the M4 checklist
claimed CI delivery while the existing scheduled/PR workflow only ran lifecycle
conformance. Local native passes do not establish CI execution (ARCH-PURPOSE /
ARCH-MOCK). Wire the new terminal native suites through a repeatable target with
an explicitly fresh candidate, strict opt-in flags, native and independent-oracle
dependencies, and triggers covering the terminal/backend/consumer sources. Keep
existing lifecycle coverage. Missing dependencies, missing test execution or
skipped required suites must fail qualification; verify the exact target locally.
This is test/CI delivery only, not a product behavior change. Record actual
execution and then retry issue close before the authorized merge.

## Revisions — 2026-09-15 close accepted; publication gate next

Whole-issue close SHIP after two rounds; all23 findings addressed. The final checklist row distinguishes completed local acceptance from the following publish operation. Operator authorized merge; SDLC owns publication and archival.
