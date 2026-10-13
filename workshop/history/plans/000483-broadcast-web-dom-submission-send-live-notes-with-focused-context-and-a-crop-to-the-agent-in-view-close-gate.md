---
gate: boundary-review
issue: 483
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T21:15:36-07:00"
      dispatched: "2026-10-12T21:11:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Idempotent retries bypass failed publication synchronization
          detail: cmd/internal/submission/store.go:467 returns success for an unchanged delivery state after an earlier rename succeeded but syncDir failed at line 498; Begin can therefore authorize paste without confirmed durable intent. The existing-bundle return at line 400 has the corresponding publication gap. Retry directory synchronization before success and add post-rename sync-failure regression tests across these paths. ARCH-ORDER.
          family: durable-retry-confirmation
          round: 1
        - id: BR-2
          severity: Critical
          title: The Lua coordinator contradicts its PURE classification and lacks a pure transition authority
          detail: workshop/plans/000483-dom-submission-plan.md:36 classifies the coordinator as PURE, but nvim/dom_submission.lua:5 independently mutates pending, ack_id, and busy while invoking RPC, scheduling, and delivery effects; nvim/dom_submission_test.lua:4 requires doubles. Reclassify the adapter as INTEGRATION, extract the explicit state/event transition authority, and add a plan Revisions entry. ARCH-PURE and ARCH-ORDER.
          family: pure-state-authority
          round: 1
        - id: BR-3
          severity: Important
          title: Submission transport duplicates agent-pane discovery
          detail: nvim/dom_transport.lua:25 repeats recursive pane collection and the agent identity predicate already implemented in nvim/pair_poke.lua:26. Consolidate parsing and identification in a shared helper while retaining submission-specific uniqueness and incarnation validation at the caller. ARCH-DRY.
          family: shared-recipient-discovery
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T21:52:34-07:00"
      dispatched: "2026-10-12T21:47:06-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: store.go:416 and :487 confirm publication on retries. Removing those checks made Prepare, Begin, Ack, Complete, and resolved-Ack regression tests fail; restored code passed.
          round: 2
        - id: BR-2
          disposition: addressed
          note: dom_submission_state.lua owns transitions; dom_submission.lua interprets effects. Plan revision at line 190 explicitly supersedes the old PURE classification. Pure sequence tests and adapter tests pass; removing the epoch guard makes the stale-response regression fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: dom_transport.lua and pair_poke.lua both consume pane_identity.lua. Parser, transport, and review-poke tests pass; mutating the shared predicate makes parser and transport tests fail.
          round: 2
      findings:
        - id: BR-4
          severity: Critical
          title: Box-text capture reports non-visible content as visible evidence
          detail: 'cmd/internal/webcast/recorder/box_text.js:11-13,40-48,65-68 checks raw rectangles against the viewport/note box without accounting for ancestor overflow clipping or rejecting an empty intersection. Disposable regression fixtures reproduce overflow-hidden text in visible_text and accessible names for a wholly off-screen region, both with truncated:false. ARCH-PURPOSE: enforce effective visible intersection consistently for text and names, and add clipping and empty-intersection regressions through the DOM seam and real-browser conformance test.'
          family: visible-evidence-intersection
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T22:04:18-07:00"
      dispatched: "2026-10-12T22:00:58-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Publication confirmation remains on unchanged and advancing retries; store failure and lifecycle tests pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: dom_submission_state.lua owns production transitions and effects; the integration adapter interprets them. State and adapter tests pass.
          round: 3
        - id: BR-3
          disposition: addressed
          note: DOM transport and pair_poke share pane_identity.lua; parser and transport tests pass.
          round: 3
        - id: BR-4
          disposition: addressed
          note: box_text.js:13-16 rejects empty intersections and shares effective clipping across text and names. Head fixtures fail against the pre-fix collector at box_text_test.js:67 and pass against pinned head. Browser regressions cover overflow, nested scrolling, partial characters, scaling, slots and offscreen regions; their execution was blocked by Chrome startup failure.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 5f9f9b5b11d452428ec7d7d82407835e89de9676
      blocked: false
---

# Gate ledger — pair#483 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T21:15:36-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `durable-retry-confirmation` Idempotent retries bypass failed publication synchronization
  cmd/internal/submission/store.go:467 returns success for an unchanged delivery state after an earlier rename succeeded but syncDir failed at line 498; Begin can therefore authorize paste without confirmed durable intent. The existing-bundle return at line 400 has the corresponding publication gap. Retry directory synchronization before success and add post-rename sync-failure regression tests across these paths. ARCH-ORDER.
- **BR-2** [Critical] `pure-state-authority` The Lua coordinator contradicts its PURE classification and lacks a pure transition authority
  workshop/plans/000483-dom-submission-plan.md:36 classifies the coordinator as PURE, but nvim/dom_submission.lua:5 independently mutates pending, ack_id, and busy while invoking RPC, scheduling, and delivery effects; nvim/dom_submission_test.lua:4 requires doubles. Reclassify the adapter as INTEGRATION, extract the explicit state/event transition authority, and add a plan Revisions entry. ARCH-PURE and ARCH-ORDER.
- **BR-3** [Important] `shared-recipient-discovery` Submission transport duplicates agent-pane discovery
  nvim/dom_transport.lua:25 repeats recursive pane collection and the agent identity predicate already implemented in nvim/pair_poke.lua:26. Consolidate parsing and identification in a shared helper while retaining submission-specific uniqueness and incarnation validation at the caller. ARCH-DRY.

## Round 2 — 2026-10-12T21:52:34-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — store.go:416 and :487 confirm publication on retries. Removing those checks made Prepare, Begin, Ack, Complete, and resolved-Ack regression tests fail; restored code passed.
- BR-2 — addressed — dom_submission_state.lua owns transitions; dom_submission.lua interprets effects. Plan revision at line 190 explicitly supersedes the old PURE classification. Pure sequence tests and adapter tests pass; removing the epoch guard makes the stale-response regression fail.
- BR-3 — addressed — dom_transport.lua and pair_poke.lua both consume pane_identity.lua. Parser, transport, and review-poke tests pass; mutating the shared predicate makes parser and transport tests fail.

### Raised

- **BR-4** [Critical] `visible-evidence-intersection` Box-text capture reports non-visible content as visible evidence
  cmd/internal/webcast/recorder/box_text.js:11-13,40-48,65-68 checks raw rectangles against the viewport/note box without accounting for ancestor overflow clipping or rejecting an empty intersection. Disposable regression fixtures reproduce overflow-hidden text in visible_text and accessible names for a wholly off-screen region, both with truncated:false. ARCH-PURPOSE: enforce effective visible intersection consistently for text and names, and add clipping and empty-intersection regressions through the DOM seam and real-browser conformance test.

## Round 3 — 2026-10-12T22:04:18-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Publication confirmation remains on unchanged and advancing retries; store failure and lifecycle tests pass.
- BR-2 — addressed — dom_submission_state.lua owns production transitions and effects; the integration adapter interprets them. State and adapter tests pass.
- BR-3 — addressed — DOM transport and pair_poke share pane_identity.lua; parser and transport tests pass.
- BR-4 — addressed — box_text.js:13-16 rejects empty intersections and shares effective clipping across text and names. Head fixtures fail against the pre-fix collector at box_text_test.js:67 and pass against pinned head. Browser regressions cover overflow, nested scrolling, partial characters, scaling, slots and offscreen regions; their execution was blocked by Chrome startup failure.

## Open findings

(none — every finding has been disposed)
