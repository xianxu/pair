---
id: 000165
status: open
created: 2026-09-01
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Clear a text input with cmd+delete

## Problem

Emptying a text input means holding backspace. Every text-entry surface in the
workbench should clear on one gesture.

**cmd+delete already reaches pair — pair discards it.** The operator's Ghostty
config maps `super+backspace` to `text:\x15`, so the terminal sends `0x15`
(`^U`). `couchtty/panelkeys.go` decodes `\r\n`→Enter, `\t`→Tab,
`0x7f`/`0x08`→Backspace, and `0x20..0x7e`→Rune; `0x15` falls through `default:`
and is dropped. The key arrives and is thrown away.

**The pair-side contract is therefore `^U`, not "cmd".** Nothing below the
terminal knows about a Command key — the Ghostty binding is the operator's, and
a different terminal or config sends something else. Implement `0x15` and
cmd+delete works as a consequence; implement "cmd+delete" and there is nothing
to implement.

**The panel has four separate text handlers.** `KeyRune`/`KeyBackspace` are
handled at `couchtty/menu.go:364`, `:494`, `:549`, and `:596` (the last inside
`reduceTextKey`). Adding a clear is four edits, and three of them are the ones
that get forgotten — a value cleared in the name field but not the describe
field is worse than the feature's absence, because it teaches a gesture that
sometimes silently does nothing.
