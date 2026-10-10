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

## Open findings

- **BR-1** [Critical] `insertion-admission-before-mutation` Entering insert mode permits edits to concealed historical turns
- **BR-2** [Critical] `owned-resource-identity-on-retarget` Replacing the thread scratch buffer leaves stale window ownership
