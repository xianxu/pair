---
gate: boundary-review
issue: 524
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T23:35:53-07:00"
      dispatched: "2026-10-12T23:31:47-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Target-only geometry can falsely label an animated crop settled
          detail: 'cmd/internal/webcast/recorder/capture_ready.js:58 skips animations whose target box misses the crop, even when an overflowing descendant inside the crop inherits the target''s animated opacity. A production-probe fixture returns pending:false for finite, paused, and infinite animations on such a parent. ARCH-PURPOSE: account for descendant and pseudo-element influence within the inspection budget, conservatively report uncertainty where necessary, and add failing regression coverage including browser pixels.'
          family: animation-influence-coverage
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T23:54:29-07:00"
      dispatched: "2026-10-12T23:52:09-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'ARCH-PURPOSE: capture_ready.js:63-71 still excludes off-box leaves whose box shadows or filters paint into the crop. A production-probe fixture returned pending:false with no reason for finite, paused, and infinite animations on such a leaf. Existing child/shadow/pseudo fixtures pass and fail against the pre-fix probe, confirming a real but incomplete correction. Keep family animation-influence-coverage open; establish and test a conservative painted-influence rule rather than adding another isolated exception.'
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T07:53:28-07:00"
      dispatched: "2026-10-13T07:50:58-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: capture_ready.js:44-63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes; substituting the prior probe fails, including the leaf-shadow assertion at capture_ready_test.js:96. Live fixtures cover overflowing children, pseudo-elements, box-shadow and drop-shadow pixels, but Chrome startup failed before those assertions could execute here.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: f5338dbc5ad7996f2ce6cb30d861d662acddb7b2
      blocked: false
    - "n": 4
      timestamp: "2026-10-13T11:28:20-07:00"
      dispatched: "2026-10-13T11:24:37-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: capture_ready.js observes animations without target-geometry exclusions. The BR-1 overflowing-child fixture passes at pinned HEAD and fails against the pre-fix probe with “child running overflow cannot be settled.”
          round: 4
      recipe: milestone-review
      isolation: fenced
      reviewed: a5edec14048bde1abd649e89ab789cd99c1323f9
      blocked: false
    - "n": 5
      timestamp: "2026-10-13T11:36:13-07:00"
      dispatched: "2026-10-13T11:32:27-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: capture_ready.js:44–63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes and fails when the preceding geometry-based probe is restored in a disposable copy.
          round: 5
      recipe: milestone-review
      isolation: fenced
      reviewed: a5edec14048bde1abd649e89ab789cd99c1323f9
      blocked: false
---

# Gate ledger — pair#524 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T23:35:53-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `animation-influence-coverage` Target-only geometry can falsely label an animated crop settled
  cmd/internal/webcast/recorder/capture_ready.js:58 skips animations whose target box misses the crop, even when an overflowing descendant inside the crop inherits the target's animated opacity. A production-probe fixture returns pending:false for finite, paused, and infinite animations on such a parent. ARCH-PURPOSE: account for descendant and pseudo-element influence within the inspection budget, conservatively report uncertainty where necessary, and add failing regression coverage including browser pixels.

## Round 2 — 2026-10-12T23:54:29-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — ARCH-PURPOSE: capture_ready.js:63-71 still excludes off-box leaves whose box shadows or filters paint into the crop. A production-probe fixture returned pending:false with no reason for finite, paused, and infinite animations on such a leaf. Existing child/shadow/pseudo fixtures pass and fail against the pre-fix probe, confirming a real but incomplete correction. Keep family animation-influence-coverage open; establish and test a conservative painted-influence rule rather than adding another isolated exception.

## Round 3 — 2026-10-13T07:53:28-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — capture_ready.js:44-63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes; substituting the prior probe fails, including the leaf-shadow assertion at capture_ready_test.js:96. Live fixtures cover overflowing children, pseudo-elements, box-shadow and drop-shadow pixels, but Chrome startup failed before those assertions could execute here.

## Round 4 — 2026-10-13T11:28:20-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — capture_ready.js observes animations without target-geometry exclusions. The BR-1 overflowing-child fixture passes at pinned HEAD and fails against the pre-fix probe with “child running overflow cannot be settled.”

## Round 5 — 2026-10-13T11:36:13-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — capture_ready.js:44–63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes and fails when the preceding geometry-based probe is restored in a disposable copy.

## Open findings

(none — every finding has been disposed)
