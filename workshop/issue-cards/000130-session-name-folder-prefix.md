---
id: '000130'
status: done
started: 2026-07-29T10:51:47-07:00
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 4.31
actual_hours: 1.78
---

# session names: folder prefix, repo token, no redundant tag

## Problem

A zellij session name is a **socket filename**, and on macOS the budget is
**24 bytes** — verified empirically: a 24-character name is accepted, 25 is
rejected with `session name must be less than 0 characters` (zellij computes
its allowance minus the long `~/Library/Caches/org.Zellij-Contributors.Zellij/…`
path and goes negative).

Pair's scheme is `pair-<repo>-<tag>` (`session_index.go:57`), which spends the
budget badly:

- **The prefix costs 5 bytes** (`pair-`) purely as an ownership marker.
- **The repo segment is usually redundant** with the tag, because the create
  flow's name prompt defaults the tag to the cwd basename. Accepting that default
  — the common case — produces `pair-pair-pair`.
- **Overflow is resolved by silent truncation.** `BuildSessionNameCandidates`
  shortens repo and tag a rune at a time until zellij accepts one, so
  `pair-parley_nvim-parley_nvim` (28) becomes **`pair-parley_nv-parley_nv`**
  (exactly 24). The user's reaction on seeing it was "I don't even know why it's
  `parley_nv`" — the truncation is invisible and unexplained.

There is also a latent unit bug: `BuildSessionNameCandidates` truncates by
`[]rune` while zellij's limit is **bytes**. `pair-` is 5 runes and 5 bytes so
they agree today; any non-ASCII component breaks that agreement, and a
multi-byte prefix would make it permanent.

The name is user-visible in `zellij list-sessions`, `pair list`, the cmux
workspace title, and — because zellij composes the terminal title as
`<session> | <focused pane title>` — in the terminal tab title itself. So this
is not an internal identifier the user can ignore.
