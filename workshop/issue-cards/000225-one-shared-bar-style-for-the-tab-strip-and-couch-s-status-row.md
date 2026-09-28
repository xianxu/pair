---
id: 000225
status: open
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
github_issue:
---

# one shared bar style for the tab strip and couch's status row

## Problem

Three bars sit at the bottom of the workbench and read as unrelated:

- **draft nvim's statusline** — the `Alt: ← history 115 < * [q=queue] > 0 queued →` bar;
- **`pair term`'s tab strip** in the layout3 right pane (`#199`) — renders `[terminal 1]`
  as plain text;
- **couch's status row** — `tools  parley.nvim  ariadne  [brain]  pair`.

The operator wants the two Go bars to adopt the draft bar's treatment, **with one
style shared between them** rather than two that drift.

### The operator's four rules

1. **Foreground: keep the current colour** (reads white on this machine).
2. **Inactive tabs subdued**, like the draft bar's `history 115`.
3. **Keep couch's notification colour** — it reads well.
4. **Background: keep the current colour**, slightly darker than the draft bar. Note
   the observed inconsistency when the right pane runs a program with its own
   colour scheme (nvim with lualine) — the strip's row does not match what surrounds
   it.

**And it must work on a light scheme.**

### What each bar does today — read from the code

| bar | foreground | inactive | notification | background |
|---|---|---|---|---|
| draft nvim | colorscheme `StatusLine` | *(no distinct style)* | — | colorscheme `StatusLine` |
| couch row (`couchtty/reserve.go:103-107`) | terminal default | brackets mark *active* only | `\x1b[38;5;220m` | terminal default |
| tab strip (`termcmd/strip.go`) | terminal default | *(no distinct style)* | — | terminal default |

One detail changes rule 2's meaning: in the draft bar, `history %d` is **not wrapped in
any highlight group** (`nvim/init.lua:2292` — only `Alt:`, `<-`, `->` use
`%#PairAltKey#`). So "the `history 115` colour" is the **colorscheme's `StatusLine`
foreground**. It lives inside nvim, and the Go bars cannot read it.

### Prior art — `#217` already made the hard decision

`#217` (*dim the right pane's tab strip when the pane loses focus*, open) styles the
**same strip**, and its commit `33b42320` settled the both-schemes question:

> Faint was rejected for the reason this repo keeps paying for: a terminal that
> ignores SGR 2 renders nothing different, silently. A colour either shows or is
> visibly wrong.
>
> …menu_render's 238/240/245/250 age ramp is fixed greys chosen against a dark
> background, and on a light scheme it inverts — 238 becomes more prominent than
> 250, so oldest would read loudest. ANSI 90 (bright black) is proposed instead,
> being the one grey terminals theme themselves, with an OSC 11 fallback only if
> measurement demands it.

**Build on that decision; do not re-derive it.** Both issues edit `strip.go`'s
rendering, so they must coordinate or they will collide.
