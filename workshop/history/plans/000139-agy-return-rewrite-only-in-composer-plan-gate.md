---
gate: plan-quality
issue: 139
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-17T12:12:06-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-08-17T12:12:13-07:00"
      agent: gemini
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-08-17T12:14:32-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Choose one tracker ownership seam and name the real resize function
          detail: The architecture says proxy owns one composerTracker, Task 2 preserves agent wrappers, and Task 3 permits either agyComposer or a unified field. Settle that ownership/API contract before coding. Also replace the nonexistent proxy.resizeWinsize with proxy.setWinsize, which currently lives at cmd/internal/wrapcmd/wrap.go:2000.
          round: 3
        - id: PQ-2
          severity: Important
          title: Define how screen mutations invalidate Agy evidence
          detail: The row-level prompt and border maps do not specify invalidation under ordinary repaint, EL/ECH, scrolling, wrapping, or resize. Define the state transition contract and negative strategy for stale nearby/enclosing chrome so AgyActive cannot violate the positive-detection purpose. ARCH-PURE and ARCH-PURPOSE.
          round: 3
        - id: PQ-3
          severity: Important
          title: Compress enumerated test cases into risky-function strategies
          detail: 'Task 1 Step 1 and Task 3 Step 1 are prohibited prose inventories of cases. Replace them with named functions and one strategy line per risky function: terminalTracker.feed over arbitrary malformed and split terminal bytes with bounded carryover; terminalState.AgyActive against stale or distant chrome with current-screen locality guards; proxy.emitPlainCR across generated gate states with overlay precedence as the invariant.'
          round: 3
        - id: PQ-4
          severity: Important
          title: Declare or resolve the open Muse workstream dependency
          detail: 'Task 2 changes cmd/internal/wrapcmd/muse_composer.go while issue 000140 remains open and issue 000139 declares deps: []. Record the dependency or explicitly establish that 139 owns consolidation after 140, avoiding concurrent ownership of the same seam.'
          round: 3
        - id: PQ-5
          severity: Important
          title: Add the external terminal-protocol fake and conformance plan
          detail: Agy's rendered byte stream is an external-binary contract, but the plan names only hand-authored unit feeds. Identify the shared byte-stream seam, a stateful transcript fake modeling composer and overlay transitions, integration replay through production flow, and a cadence for comparing it with live Agy output. ARCH-MOCK.
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-17T12:16:38-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Proxy ownership is settled on one dedicated agyComposerTracker, and the plan now names the real proxy.setWinsize function at wrap.go:2000.
          round: 4
        - id: PQ-2
          disposition: not-addressed
          note: ECH, scrolling, wrapping, resize reflow, and prompt invalidation under ordinary repaint remain undefined, so stale evidence can still satisfy active() (ARCH-PURE, ARCH-PURPOSE).
          round: 4
        - id: PQ-3
          disposition: not-addressed
          note: The test steps still enumerate commands, conditions, matrices, and lifecycle cases instead of the requested one-line adversarial strategies for feed, active, and emitPlainCR.
          round: 4
        - id: PQ-4
          disposition: addressed
          note: The plan isolates Agy in agy_composer.go, explicitly leaves Muse files untouched, and declares no dependency on open issue 000140.
          round: 4
        - id: PQ-5
          disposition: not-addressed
          note: The seam, stateful fake, and production-flow replay are named, but no live-Agy conformance mechanism or cadence is specified (ARCH-MOCK).
          round: 4
      blocked: false
    - "n": 5
      timestamp: "2026-08-17T15:53:28-07:00"
      agent: codex
      dispose:
        - id: PQ-2
          disposition: addressed
          note: The shared current-screen emulator, atomic snapshots, fail-safe visibility resets, coherent-box locality guards, and stale-state negative strategies define mutation invalidation.
          round: 5
        - id: PQ-3
          disposition: not-addressed
          note: Tasks 1 and 3 still enumerate many individual cases and procedural RED/GREEN steps; compress these into named risky functions with one adversarial-input strategy and mechanical guard per function.
          round: 5
        - id: PQ-5
          disposition: addressed
          note: The plan names proxy.handleChunk as the production seam, a stateful harnessSessionFake, literal fixture replay through production flow, and live conformance during release validation.
          round: 5
      findings:
        - id: PQ-6
          severity: Important
          title: Correct the x/vt reply-pipe shutdown contract
          detail: Task 1 says to retain and close Emulator.InputPipe before joining the drainer, but the installed API returns io.Writer at emulator.go:297, so Close is unavailable as written. Specify a checked io.WriteCloser assertion with failure handling, introduce an owned closable pipe seam, or use another verified shutdown ordering; then align the nonblocking and race tests with that contract (ARCH-PURE).
          round: 5
      blocked: false
    - "n": 6
      timestamp: "2026-08-17T15:56:24-07:00"
      agent: codex
      dispose:
        - id: PQ-3
          disposition: addressed
          note: Tasks 1 and 3 now name risky functions and state one adversarial-input strategy with its mechanical guard.
          round: 6
        - id: PQ-6
          disposition: addressed
          note: Construction now checks InputPipe's dynamic io.Closer capability before starting the drainer, and Close uses the verified writer-close, join, emulator-close ordering with shutdown tests.
          round: 6
      blocked: false
content_hash: a4d0c03ef84bc74f0f1c7c470fc6b2c65d046f5fcebf5257033c351c8e4fd3ea
---

# Gate ledger — pair#139 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-17T12:12:06-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-08-17T12:12:13-07:00 (gemini) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-08-17T12:14:32-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] Choose one tracker ownership seam and name the real resize function
  The architecture says proxy owns one composerTracker, Task 2 preserves agent wrappers, and Task 3 permits either agyComposer or a unified field. Settle that ownership/API contract before coding. Also replace the nonexistent proxy.resizeWinsize with proxy.setWinsize, which currently lives at cmd/internal/wrapcmd/wrap.go:2000.
- **PQ-2** [Important] Define how screen mutations invalidate Agy evidence
  The row-level prompt and border maps do not specify invalidation under ordinary repaint, EL/ECH, scrolling, wrapping, or resize. Define the state transition contract and negative strategy for stale nearby/enclosing chrome so AgyActive cannot violate the positive-detection purpose. ARCH-PURE and ARCH-PURPOSE.
- **PQ-3** [Important] Compress enumerated test cases into risky-function strategies
  Task 1 Step 1 and Task 3 Step 1 are prohibited prose inventories of cases. Replace them with named functions and one strategy line per risky function: terminalTracker.feed over arbitrary malformed and split terminal bytes with bounded carryover; terminalState.AgyActive against stale or distant chrome with current-screen locality guards; proxy.emitPlainCR across generated gate states with overlay precedence as the invariant.
- **PQ-4** [Important] Declare or resolve the open Muse workstream dependency
  Task 2 changes cmd/internal/wrapcmd/muse_composer.go while issue 000140 remains open and issue 000139 declares deps: []. Record the dependency or explicitly establish that 139 owns consolidation after 140, avoiding concurrent ownership of the same seam.
- **PQ-5** [Important] Add the external terminal-protocol fake and conformance plan
  Agy's rendered byte stream is an external-binary contract, but the plan names only hand-authored unit feeds. Identify the shared byte-stream seam, a stateful transcript fake modeling composer and overlay transitions, integration replay through production flow, and a cadence for comparing it with live Agy output. ARCH-MOCK.

## Round 4 — 2026-08-17T12:16:38-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Proxy ownership is settled on one dedicated agyComposerTracker, and the plan now names the real proxy.setWinsize function at wrap.go:2000.
- PQ-2 — not-addressed — ECH, scrolling, wrapping, resize reflow, and prompt invalidation under ordinary repaint remain undefined, so stale evidence can still satisfy active() (ARCH-PURE, ARCH-PURPOSE).
- PQ-3 — not-addressed — The test steps still enumerate commands, conditions, matrices, and lifecycle cases instead of the requested one-line adversarial strategies for feed, active, and emitPlainCR.
- PQ-4 — addressed — The plan isolates Agy in agy_composer.go, explicitly leaves Muse files untouched, and declares no dependency on open issue 000140.
- PQ-5 — not-addressed — The seam, stateful fake, and production-flow replay are named, but no live-Agy conformance mechanism or cadence is specified (ARCH-MOCK).

## Round 5 — 2026-08-17T15:53:28-07:00 (codex) — passed

### Disposed

- PQ-2 — addressed — The shared current-screen emulator, atomic snapshots, fail-safe visibility resets, coherent-box locality guards, and stale-state negative strategies define mutation invalidation.
- PQ-3 — not-addressed — Tasks 1 and 3 still enumerate many individual cases and procedural RED/GREEN steps; compress these into named risky functions with one adversarial-input strategy and mechanical guard per function.
- PQ-5 — addressed — The plan names proxy.handleChunk as the production seam, a stateful harnessSessionFake, literal fixture replay through production flow, and live conformance during release validation.

### Raised

- **PQ-6** [Important] Correct the x/vt reply-pipe shutdown contract
  Task 1 says to retain and close Emulator.InputPipe before joining the drainer, but the installed API returns io.Writer at emulator.go:297, so Close is unavailable as written. Specify a checked io.WriteCloser assertion with failure handling, introduce an owned closable pipe seam, or use another verified shutdown ordering; then align the nonblocking and race tests with that contract (ARCH-PURE).

## Round 6 — 2026-08-17T15:56:24-07:00 (codex) — passed

### Disposed

- PQ-3 — addressed — Tasks 1 and 3 now name risky functions and state one adversarial-input strategy with its mechanical guard.
- PQ-6 — addressed — Construction now checks InputPipe's dynamic io.Closer capability before starting the drainer, and Close uses the verified writer-close, join, emulator-close ordering with shutdown tests.

## Open findings

(none — every finding has been disposed)
