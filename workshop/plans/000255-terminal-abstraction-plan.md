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
| hostty.Host / Fake | INTEGRATION | cmd/internal/hostty/{host,fake}.go | Reused: parent terminal IO and controlled partial/error writes |
| Console / terminalMux | INTEGRATION | cmd/internal/{couchtty/console,termcmd/run}.go | Modified: product policy submits events and chrome, cannot write child drawing bytes to parent |

TerminalEndpoint is not labeled pure: the candidate backend has reply pipes and mutable IO lifecycle. Profile/Frame/View transition tests require no IO. Do not create a parallel parser for each console. Existing wrapcmd observer remains a separate observer of its own connection; it must not become a competing authority for the same endpoint. During migration, old Screen-derived mode getters must delegate to the endpoint or be removed for migrated consumers (ARCH-DRY, ARCH-PURE).

## Behavioral contract

### Terminal profile and effects

The required initial profile preserves current interactive Zellij, nvim, shell and agent behavior: UTF-8 including wide/combining characters; indexed/truecolor styles; cursor/save, erase, scroll regions and origin mode; primary/alternate buffers; application cursor/keypad; bracketed paste; focus; SGR mouse with off/click/drag/all-motion; the extended keyboard negotiation/encoding needed by existing shortcuts; hyperlinks; synchronized drawing; clipboard and Pair notifications. Backend qualification must enumerate exact sequences/queries from these features, advertise only implemented capabilities and provide matching terminfo/environment. Existing required shortcuts cannot be removed to make a deficient backend pass.

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

- [ ] Compare baseline/candidate startup, sustained output CPU/memory, redraw throughput, switch latency and idle mouse movement; approve measured budgets before rollout rather than inventing a performance claim.
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
