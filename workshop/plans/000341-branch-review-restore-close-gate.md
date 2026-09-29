---
gate: boundary-review
issue: 341
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T21:33:20-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Handoff deletion precedes final application authorization
          detail: nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
          family: consume-after-acceptance
          round: 1
        - id: BR-2
          severity: Important
          title: Routine typing synchronously resolves full Git history
          detail: nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
          family: nonblocking-editor-observation
          round: 1
        - id: BR-3
          severity: Important
          title: README omits branch restoration and recovery commands
          detail: README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
          family: user-surface-documentation
          round: 1
        - id: BR-4
          severity: Minor
          title: Repeated activation accumulates rendering autocmds
          detail: nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).
          family: activation-resource-ownership
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-28T22:00:39-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Handoff consumption follows explicit apply/defer acceptance. The passing production-function regression fails with the old watcher at the final-refusal preservation assertion.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Edit observation is asynchronous and coalesced. Restoring synchronous observation fails the responsiveness test at approximately 1155 ms with six scans. The distinct late-completion safety regression is reported below.
          round: 2
        - id: BR-3
          disposition: addressed
          note: README.md:99 documents branch restoration and blocking conditions; README.md:108 documents recovery/discard commands implemented in nvim/review.lua.
          round: 2
        - id: BR-4
          disposition: addressed
          note: Activation owns a clearable rendering group. The passing observation test fails with callback counts increasing from one to three when cleanup is removed.
          round: 2
      findings:
        - id: BR-5
          severity: Critical
          title: Late asynchronous observation reloads another branch into the active review
          detail: 'nvim/review/recovery_observer.lua:36 accepts a captured matching identity, then nvim/review.lua:843 invokes checktime against the current checkout. A controlled production-pane probe captured review/a, switched to review/b before delivery, and loaded B''s bytes while the pane remained bound to A. This is the 2nd finding in this family: enforce identity-bound observation effects across refresh, preservation, and coalescing decisions rather than patching only this callback. Reload verified snapshot bytes and add a controlled late-completion regression without restoring synchronous editor observation. ARCH-ORDER, ARCH-PURPOSE.'
          family: nonblocking-editor-observation
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-28T22:17:55-07:00"
      agent: codex
      dispose:
        - id: BR-5
          disposition: not-addressed
          note: 'Captured snapshots fix callback-time rereads, but identity.go:209,220-226 can still label B bytes as A during checkout. A deterministic real-Git probe paused checkout A→B in a smudge filter: index.lock existed, HEAD still named review/a, and a.md already contained B bytes. The production resolver returned status resolved, branch review/a, snapshot "B bytes\n". Enforce snapshot authority across concurrent checkout mutation, with fail-closed retry and a controlled in-progress-checkout regression. ARCH-ORDER, ARCH-PURPOSE; existing nonblocking-editor-observation family.'
          round: 3
        - id: BR-1
          disposition: addressed
          note: Consumption follows explicit acceptance; acceptance, uncertainty, replacement, and refusal regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Typing/focus callbacks use coalesced asynchronous observation; delayed-resolver responsiveness tests pass.
          round: 3
        - id: BR-3
          disposition: addressed
          note: README documents branch restoration, blocked switching, and recover/discard commands, consistent with the implemented interfaces.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Activation clears owned rendering callbacks; repeated activation callback-count regression passes.
          round: 3
      findings:
        - id: BR-6
          severity: Critical
          title: Activating a CRLF document corrupts its bytes on subsequent save
          detail: nvim/review/restore_controller.lua:132-146 reads binary lines retaining carriage returns, then inserts them into a buffer whose bufload selected fileformat=dos. A controller probe activating a file containing "B\r\n" produced buffer line "B\r"; writing saved "B\r\r\n". Share byte-to-buffer decoding with the asynchronous refresh path, including fileformat and endofline handling. Add activation-and-save regressions for new and retained buffers, including branch-driven format changes. ARCH-DRY, ARCH-PURPOSE.
          family: document-byte-preservation
          round: 3
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-09-28T22:39:45-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Acceptance precedes consumption; handoff acceptance, refusal, replacement, and uncertainty regressions pass.
          round: 4
        - id: BR-2
          disposition: addressed
          note: Controlled slow-resolver tests pass for nonblocking edit/focus callbacks and coalesced observations.
          round: 4
        - id: BR-3
          disposition: addressed
          note: README.md:99 documents branch restoration, blocked transitions, and recovery commands, matching the implemented controller and recovery commands.
          round: 4
        - id: BR-4
          disposition: addressed
          note: The production observation test verifies stable rendering callback counts across repeated A → B → A activation.
          round: 4
        - id: BR-5
          disposition: addressed
          note: Captured snapshots reach activation and refresh without later filesystem reads. Independently disabling index-generation admission and staged-tree rejection in scratch overlays makes the paused-checkout and index-before-HEAD regressions fail.
          round: 4
        - id: BR-6
          disposition: addressed
          note: Activation and refresh share document_bytes decoding. New/retained-buffer exact-save regressions pass across CRLF, BOM, LF, mixed endings, empty and no-EOL documents; removing CRLF decoding in a scratch copy makes the regression fail.
          round: 4
      findings:
        - id: BR-7
          severity: Important
          title: Fresh-session regression overwrites inherited review context outside its fixture
          detail: 'tests/review-fresh-restore-test.sh:109 inherits os.environ without rebinding PAIR_REVIEW_CONTEXT_PATH. Its finish_human_turn call at line 71 reaches nvim/review.lua:540 and overwrites that caller-owned path. Reproduced with a scratch sentinel outside the fixture: the test passed while replacing its contents with "A reviewed\n". Sanitize inherited Pair session variables, bind every writable artifact to fixture storage, and add a sentinel regression. Sweep the new branch/observation fixtures using the same environment construction. ARCH-SECURE.'
          family: test-environment-isolation
          round: 4
      recipe: milestone-review
      blocked: false
    - "n": 5
      timestamp: "2026-09-28T22:49:39-07:00"
      agent: codex
      dispose:
        - id: BR-7
          disposition: addressed
          note: tests/lib/review_test_env.py:43 sanitizes inherited session variables and binds writable artifacts to fixture storage. Fresh, branch, observation and producer fixtures passed with caller sentinels intact. A scratch mutation restoring the inherited PAIR_REVIEW_CONTEXT_PATH caused the fresh-session regression to fail at the sentinel assertion.
          round: 5
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-09-28T22:58:04-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Handoff consumption follows explicit acceptance; passing acceptance tests cover final authorization refusal, replacement generations, reentry, and uncertain effects.
          round: 6
        - id: BR-2
          disposition: addressed
          note: Typing/focus observation uses the coalescing asynchronous observer; observer and process latency regressions pass.
          round: 6
        - id: BR-3
          disposition: addressed
          note: README documents branch restoration, blocked switching, stable-checkout requirements, and recover/discard commands; controller and recovery implementations support those passages.
          round: 6
        - id: BR-4
          disposition: addressed
          note: Activation rendering callbacks belong to a cleared augroup; the passing observation fixture checks stable callback counts across repeated activation.
          round: 6
        - id: BR-5
          disposition: addressed
          note: Refresh and activation install resolver-captured bytes; passing tests cover delayed delivery, checkout pauses, index publication, and retained-buffer write refusal.
          round: 6
        - id: BR-6
          disposition: addressed
          note: Shared byte decoding preserves line endings and BOM options; exact-save tests and the CRLF/BOM committed-round process regression pass.
          round: 6
        - id: BR-7
          disposition: addressed
          note: All four new process fixtures use the shared sanitized environment and caller-owned sentinels; each passed independently without sentinel changes.
          round: 6
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#341 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T21:33:20-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `consume-after-acceptance` Handoff deletion precedes final application authorization
  nvim/review/handoff.lua:90 deletes before the callback; nvim/review/init.lua:127 can subsequently refuse after checkout movement. A controlled production-function probe confirmed payload loss without application. Consume only after explicit apply/defer acceptance and test this interleaving (ARCH-ORDER, ARCH-PURPOSE).
- **BR-2** [Important] `nonblocking-editor-observation` Routine typing synchronously resolves full Git history
  nvim/review.lua:848 routes TextChanged/TextChangedI through preserve_mismatch and guard to nvim/review/identity.lua:23, synchronously waiting up to 2.5 seconds per resolver call. Make proactive recovery observation asynchronous and test responsiveness with delayed resolution (ARCH-CONSTRAINTS).
- **BR-3** [Important] `user-surface-documentation` README omits branch restoration and recovery commands
  README.md:139 retains the previous target-based Alt+C description. This range introduces PairReviewRecover and PairReviewDiscardRecovery without any README update; document the changed behavior and recovery workflow.
- **BR-4** [Minor] `activation-resource-ownership` Repeated activation accumulates rendering autocmds
  nvim/review.lua:790 registers callbacks on every activation without corresponding cleanup. A real A-to-B-to-A probe increased A's TextChanged callback count from one to two. Use a clearable owned group and assert stable counts (ARCH-FUNERAL).

## Round 2 — 2026-09-28T22:00:39-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Handoff consumption follows explicit apply/defer acceptance. The passing production-function regression fails with the old watcher at the final-refusal preservation assertion.
- BR-2 — addressed — Edit observation is asynchronous and coalesced. Restoring synchronous observation fails the responsiveness test at approximately 1155 ms with six scans. The distinct late-completion safety regression is reported below.
- BR-3 — addressed — README.md:99 documents branch restoration and blocking conditions; README.md:108 documents recovery/discard commands implemented in nvim/review.lua.
- BR-4 — addressed — Activation owns a clearable rendering group. The passing observation test fails with callback counts increasing from one to three when cleanup is removed.

### Raised

- **BR-5** [Critical] `nonblocking-editor-observation` Late asynchronous observation reloads another branch into the active review
  nvim/review/recovery_observer.lua:36 accepts a captured matching identity, then nvim/review.lua:843 invokes checktime against the current checkout. A controlled production-pane probe captured review/a, switched to review/b before delivery, and loaded B's bytes while the pane remained bound to A. This is the 2nd finding in this family: enforce identity-bound observation effects across refresh, preservation, and coalescing decisions rather than patching only this callback. Reload verified snapshot bytes and add a controlled late-completion regression without restoring synchronous editor observation. ARCH-ORDER, ARCH-PURPOSE.

## Round 3 — 2026-09-28T22:17:55-07:00 (codex) — BLOCKED

### Disposed

- BR-5 — not-addressed — Captured snapshots fix callback-time rereads, but identity.go:209,220-226 can still label B bytes as A during checkout. A deterministic real-Git probe paused checkout A→B in a smudge filter: index.lock existed, HEAD still named review/a, and a.md already contained B bytes. The production resolver returned status resolved, branch review/a, snapshot "B bytes\n". Enforce snapshot authority across concurrent checkout mutation, with fail-closed retry and a controlled in-progress-checkout regression. ARCH-ORDER, ARCH-PURPOSE; existing nonblocking-editor-observation family.
- BR-1 — addressed — Consumption follows explicit acceptance; acceptance, uncertainty, replacement, and refusal regressions pass.
- BR-2 — addressed — Typing/focus callbacks use coalesced asynchronous observation; delayed-resolver responsiveness tests pass.
- BR-3 — addressed — README documents branch restoration, blocked switching, and recover/discard commands, consistent with the implemented interfaces.
- BR-4 — addressed — Activation clears owned rendering callbacks; repeated activation callback-count regression passes.

### Raised

- **BR-6** [Critical] `document-byte-preservation` Activating a CRLF document corrupts its bytes on subsequent save
  nvim/review/restore_controller.lua:132-146 reads binary lines retaining carriage returns, then inserts them into a buffer whose bufload selected fileformat=dos. A controller probe activating a file containing "B\r\n" produced buffer line "B\r"; writing saved "B\r\r\n". Share byte-to-buffer decoding with the asynchronous refresh path, including fileformat and endofline handling. Add activation-and-save regressions for new and retained buffers, including branch-driven format changes. ARCH-DRY, ARCH-PURPOSE.

## Round 4 — 2026-09-28T22:39:45-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Acceptance precedes consumption; handoff acceptance, refusal, replacement, and uncertainty regressions pass.
- BR-2 — addressed — Controlled slow-resolver tests pass for nonblocking edit/focus callbacks and coalesced observations.
- BR-3 — addressed — README.md:99 documents branch restoration, blocked transitions, and recovery commands, matching the implemented controller and recovery commands.
- BR-4 — addressed — The production observation test verifies stable rendering callback counts across repeated A → B → A activation.
- BR-5 — addressed — Captured snapshots reach activation and refresh without later filesystem reads. Independently disabling index-generation admission and staged-tree rejection in scratch overlays makes the paused-checkout and index-before-HEAD regressions fail.
- BR-6 — addressed — Activation and refresh share document_bytes decoding. New/retained-buffer exact-save regressions pass across CRLF, BOM, LF, mixed endings, empty and no-EOL documents; removing CRLF decoding in a scratch copy makes the regression fail.

### Raised

- **BR-7** [Important] `test-environment-isolation` Fresh-session regression overwrites inherited review context outside its fixture
  tests/review-fresh-restore-test.sh:109 inherits os.environ without rebinding PAIR_REVIEW_CONTEXT_PATH. Its finish_human_turn call at line 71 reaches nvim/review.lua:540 and overwrites that caller-owned path. Reproduced with a scratch sentinel outside the fixture: the test passed while replacing its contents with "A reviewed\n". Sanitize inherited Pair session variables, bind every writable artifact to fixture storage, and add a sentinel regression. Sweep the new branch/observation fixtures using the same environment construction. ARCH-SECURE.

## Round 5 — 2026-09-28T22:49:39-07:00 (codex) — passed

### Disposed

- BR-7 — addressed — tests/lib/review_test_env.py:43 sanitizes inherited session variables and binds writable artifacts to fixture storage. Fresh, branch, observation and producer fixtures passed with caller sentinels intact. A scratch mutation restoring the inherited PAIR_REVIEW_CONTEXT_PATH caused the fresh-session regression to fail at the sentinel assertion.

## Round 6 — 2026-09-28T22:58:04-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Handoff consumption follows explicit acceptance; passing acceptance tests cover final authorization refusal, replacement generations, reentry, and uncertain effects.
- BR-2 — addressed — Typing/focus observation uses the coalescing asynchronous observer; observer and process latency regressions pass.
- BR-3 — addressed — README documents branch restoration, blocked switching, stable-checkout requirements, and recover/discard commands; controller and recovery implementations support those passages.
- BR-4 — addressed — Activation rendering callbacks belong to a cleared augroup; the passing observation fixture checks stable callback counts across repeated activation.
- BR-5 — addressed — Refresh and activation install resolver-captured bytes; passing tests cover delayed delivery, checkout pauses, index publication, and retained-buffer write refusal.
- BR-6 — addressed — Shared byte decoding preserves line endings and BOM options; exact-save tests and the CRLF/BOM committed-round process regression pass.
- BR-7 — addressed — All four new process fixtures use the shared sanitized environment and caller-owned sentinels; each passed independently without sentinel changes.

## Open findings

(none — every finding has been disposed)
