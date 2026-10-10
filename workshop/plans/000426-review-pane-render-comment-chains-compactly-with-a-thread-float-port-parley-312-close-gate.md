---
gate: boundary-review
issue: 426
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T12:33:48-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Entering insert mode permits edits to concealed historical turns
          detail: "nvim/review/comment.lua:108-109 omits insertion-entry admission. On \U0001F916[old]{answer}[reply], place the cursor at byte column 8, press i, then X; the result is \U0001F916[oldX]{answer}[reply], even with event-loop turns between keys. Enforce the allowed insertion points before mutation and test real mode-entry and queued-input sequences. ARCH-PURPOSE, ARCH-ORDER."
          family: insertion-admission-before-mutation
          round: 1
        - id: BR-2
          severity: Critical
          title: Replacing the thread scratch buffer leaves stale window ownership
          detail: nvim/review/comment_float.lua:131-134 does not handle scratch-buffer replacement. After :enew in a clean thread, line 88 treats the surviving window as a live thread, so reopening focuses an ordinary buffer and later cleanup closes that replacement window. Release ownership on scratch departure and verify buffer identity before focus/close; test replacement, reopening, and cleanup. ARCH-ORDER, ARCH-FUNERAL.
          family: owned-resource-identity-on-retarget
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-10T12:41:10-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: "nvim/review/comment.lua:116-123 protects mode entry and InsertCharPre, but Enter bypasses the latter. In the real PTY fixture, use \U0001F916[old]{answer}[reply], position at byte column 8, then queue i<Left><Left><Left><CR>X<Esc>. The concealed agent turn splits, leaving the first line \U0001F916[old]{answe. The existing typed-character regressions pass, but this added probe fails. Enumerate insertion mechanisms and enforce admission before every supported mutation, including newline insertion; add real queued-input regressions. ARCH-PURPOSE, ARCH-ORDER."
          round: 2
        - id: BR-2
          disposition: addressed
          note: comment_float.lua checks window/buffer identity before focus and close, and releases ownership on scratch departure. Current replacement/reopen/cleanup tests pass. Running the same regression against the pre-fix controller in a temporary copy fails at “reopen must create a real thread,” confirming meaningful regression coverage.
          round: 2
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-10T12:46:25-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: InsertEnter and the activation-owned on_key callback admit insertion points before processing input. Current PTY regressions pass; reverting comment.lua to f3ebcb63 in a temporary copy makes the queued-newline regression fail by splitting concealed {answer} text.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Scratch departure ends ownership, and focus/cleanup check the exact window-buffer pair. Clean and dirty replacement, rescue, reopening, and replacement-window preservation tests pass.
          round: 3
      recipe: milestone-review
      reviewed: 320c2a53e080eb288c34dabbf131d97f7fc9f895
      blocked: false
---

# Gate ledger — pair#426 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T12:33:48-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `insertion-admission-before-mutation` Entering insert mode permits edits to concealed historical turns
  nvim/review/comment.lua:108-109 omits insertion-entry admission. On 🤖[old]{answer}[reply], place the cursor at byte column 8, press i, then X; the result is 🤖[oldX]{answer}[reply], even with event-loop turns between keys. Enforce the allowed insertion points before mutation and test real mode-entry and queued-input sequences. ARCH-PURPOSE, ARCH-ORDER.
- **BR-2** [Critical] `owned-resource-identity-on-retarget` Replacing the thread scratch buffer leaves stale window ownership
  nvim/review/comment_float.lua:131-134 does not handle scratch-buffer replacement. After :enew in a clean thread, line 88 treats the surviving window as a live thread, so reopening focuses an ordinary buffer and later cleanup closes that replacement window. Release ownership on scratch departure and verify buffer identity before focus/close; test replacement, reopening, and cleanup. ARCH-ORDER, ARCH-FUNERAL.

## Round 2 — 2026-10-10T12:41:10-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — nvim/review/comment.lua:116-123 protects mode entry and InsertCharPre, but Enter bypasses the latter. In the real PTY fixture, use 🤖[old]{answer}[reply], position at byte column 8, then queue i<Left><Left><Left><CR>X<Esc>. The concealed agent turn splits, leaving the first line 🤖[old]{answe. The existing typed-character regressions pass, but this added probe fails. Enumerate insertion mechanisms and enforce admission before every supported mutation, including newline insertion; add real queued-input regressions. ARCH-PURPOSE, ARCH-ORDER.
- BR-2 — addressed — comment_float.lua checks window/buffer identity before focus and close, and releases ownership on scratch departure. Current replacement/reopen/cleanup tests pass. Running the same regression against the pre-fix controller in a temporary copy fails at “reopen must create a real thread,” confirming meaningful regression coverage.

## Round 3 — 2026-10-10T12:46:25-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — InsertEnter and the activation-owned on_key callback admit insertion points before processing input. Current PTY regressions pass; reverting comment.lua to f3ebcb63 in a temporary copy makes the queued-newline regression fail by splitting concealed {answer} text.
- BR-2 — addressed — Scratch departure ends ownership, and focus/cleanup check the exact window-buffer pair. Clean and dirty replacement, rescue, reopening, and replacement-window preservation tests pass.

## Open findings

(none — every finding has been disposed)
