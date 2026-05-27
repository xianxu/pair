---
id: 000023
status: open
deps: []
created: 2026-05-27
updated: 2026-05-27
related: [cmd/pair-wrap/main.go]
---

# Suspend Enter remap while a Claude blocking overlay is open

## Problem

`pair-wrap` translates the user's plain Enter into `\` + `\r` for the
`claude` agent (`cmd/pair-wrap/main.go:127`). That rewrite is correct
inside Claude's textarea — Claude reads `\<CR>` as "insert newline" —
but it is wrong whenever a **blocking overlay** (the AskUserQuestion
picker or a tool-permission prompt) has focus. The overlay consumes the
`\r` to confirm the highlighted option, and the leading `\` falls
through into the textarea behind it. The user sees a stray `\` appear
in their input box every time they pick an option.

Regression window: started "about 2 weeks back" (≈2026-05-13). Claude
Code v2.1.141 (released 2026-05-13) shipped *"Fixed pressing Enter
while a permission/dialog prompt is open also submitting text in the
input box."* That fix tightened Enter routing in dialogs and exposed
the leftover `\` that pair has always been sending. Before v2.1.141 the
overlay was eating both bytes; after, only the `\r` is consumed and the
`\` leaks.

## Spec

Make pair-wrap's Enter remap **context-aware**: suspend the remap
whenever Claude has a blocking overlay open, restore it once the
overlay closes.

**Open signal — semantic, from agent output stream.** Claude emits
`OSC 777 ; notify ; Claude Code ; <body>` BEL at known moments:

| Body                                  | Context                          | Suspend remap? |
|---------------------------------------|----------------------------------|----------------|
| `Claude is waiting for your input`    | End of turn, textarea active     | No             |
| `Claude needs your permission`        | Picker or tool-permission prompt | **Yes**        |

The "needs your permission" variant is the unified open trigger for
both pickers and tool-permission prompts. Both have the stray-`\` bug,
so handling them together is correct.

**Close signal — from user input.** OSC 777 is one-shot (no closing
counterpart). Picker and permission prompts dismiss on a single Enter
or ESC, so we clear `pickerActive` after the first stdin event that
would have triggered the remap — that Enter goes through as plain `\r`
to confirm the overlay, and the next Enter (now in textarea) gets the
normal `\<CR>` remap again.

**Known limitation.** If the user dismisses the overlay with ESC and
then types text and hits Enter expecting a newline, the first
post-dismissal Enter will pass through as a bare `\r` and submit the
draft instead of inserting a newline. This is the worst-case
regression; impact is one accidental submit per ESC-dismissed overlay.
Acceptable for v1 because (a) ESC-dismiss is rare relative to
Enter-confirm and (b) the symptom is recoverable (resend with newline
edit). If it bites in practice, add output-side close detection (the
textarea hint string `shift+tab to cycle` reappears when the textarea
takes focus back) as v2 hardening.

## Plan

- [ ] **M1: Issue + repro evidence.** This file. Reference the captured
      raw-stream offsets in `/Users/xianxu/.local/share/pair/scrollback-pair-claude.raw`
      that documented the OSC 777 signal (offsets 209730, 307851,
      395083, 857406; only 395083 was a picker — body `needs your
      permission`).

- [ ] **M2: Detection + state.** In `cmd/pair-wrap/main.go`:
      - Add `pickerActive bool` to `proxy` struct (sibling of `sendKM`).
      - In the agent-output read loop (`p.ptmx.Read` at line 1331),
        scan each chunk for `\x1b]777;notify;Claude Code;Claude needs
        your permission\x07`. Use the existing `agentPending` carry-over
        so split-across-reads escapes still match. Set `pickerActive =
        true` on match.
      - Only enable for `agentBasename == "claude"`.

- [ ] **M3: Suspend the remap.** In `translateStdinFrom`
      (called from `translateStdin` at line 739):
      - Before translating plain `\r` into `plainCR`, check
        `p.pickerActive`. If true, emit `\r` verbatim and set
        `p.pickerActive = false`.
      - Leave `altCR` (Alt+Enter) handling unchanged — pickers don't
        interact with Alt+Enter.

- [ ] **M4: Tests.**
      - `update_agent_output_test.go`: feed a chunk containing the
        picker OSC 777, assert `pickerActive` flips. Feed the
        "waiting for your input" variant, assert flag unchanged. Feed
        the OSC split across two reads, assert flag still flips.
      - `translate_stdin_test.go` / `translate_test.go`: with
        `pickerActive=true`, plain `\r` translates to `\r` and the
        flag clears. With `pickerActive=false`, plain `\r` translates
        to `\\\r` (existing behavior).

- [ ] **M5: Manual verification.**
      - Run claude under pair with PAIR_WRAP_LOG enabled.
      - Trigger an AskUserQuestion picker, select an option, hit
        Enter. Confirm no stray `\` in textarea after dismissal.
      - Repeat with a tool-permission prompt.
      - Confirm normal Enter-in-textarea still inserts a newline after
        the picker is dismissed.

- [ ] **M6: Atlas update.** If `atlas/` documents the agent keymap
      remap, add a one-line note that the remap is suspended while a
      blocking overlay is open, with the OSC 777 trigger pattern.

## Log

(empty)
