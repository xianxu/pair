---
id: '000023'
status: done
created: 2026-05-27
updated: 2026-05-27
estimate_hours: 2
actual_hours: 1.5
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
