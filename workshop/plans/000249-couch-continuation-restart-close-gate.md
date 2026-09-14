---
gate: boundary-review
issue: 249
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T12:38:59-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Core-concepts table marks unchanged orientation files as modified
          detail: workshop/plans/000249-couch-continuation-restart-plan.md:53 lists console_switchagent.go and switchcontext.go as modified, but neither changes in the pinned range. Append a revision identifying unchanged reuse and the new callers; the explicit Core-concepts contract makes this contradiction blocking.
          family: core-concepts-match-diff
          round: 1
        - id: BR-2
          severity: Important
          title: Continuation completion overrides an intervening operator focus change
          detail: 'console_continuation.go:125 captures PreserveFocus at enqueue time, and console.go:2055 uses it after asynchronous replacement. Selecting another actor while replacement runs is overridden by foreground adoption. ARCH-ORDER: preserve intervening focus changes and test acceptance, operator switch, then completion.'
          family: async-completion-preserves-user-intent
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T12:46:48-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan lines 232–255 supersede the original classifications. Line 254 correctly identifies unchanged orientation dependencies and names their new callers; the pinned diff and source confirm this prose-only correction.
          round: 2
        - id: BR-2
          disposition: withdrawn
          note: console.go:411 changes focus only when active is empty, under the installation mutex. TestContinuationCompletionPreservesInterveningFocus at console_continuation_test.go:216 passes all four event orders under race detection without production changes. The previously alleged override is not supported.
          round: 2
      blocked: false
---

# Gate ledger — pair#249 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T12:38:59-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `core-concepts-match-diff` Core-concepts table marks unchanged orientation files as modified
  workshop/plans/000249-couch-continuation-restart-plan.md:53 lists console_switchagent.go and switchcontext.go as modified, but neither changes in the pinned range. Append a revision identifying unchanged reuse and the new callers; the explicit Core-concepts contract makes this contradiction blocking.
- **BR-2** [Important] `async-completion-preserves-user-intent` Continuation completion overrides an intervening operator focus change
  console_continuation.go:125 captures PreserveFocus at enqueue time, and console.go:2055 uses it after asynchronous replacement. Selecting another actor while replacement runs is overridden by foreground adoption. ARCH-ORDER: preserve intervening focus changes and test acceptance, operator switch, then completion.

## Round 2 — 2026-09-14T12:46:48-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Plan lines 232–255 supersede the original classifications. Line 254 correctly identifies unchanged orientation dependencies and names their new callers; the pinned diff and source confirm this prose-only correction.
- BR-2 — withdrawn — console.go:411 changes focus only when active is empty, under the installation mutex. TestContinuationCompletionPreservesInterveningFocus at console_continuation_test.go:216 passes all four event orders under race detection without production changes. The previously alleged override is not supported.

## Open findings

(none — every finding has been disposed)
