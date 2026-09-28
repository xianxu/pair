---
id: '000129'
status: wontfix
started: 2026-07-29T10:04:38-07:00
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 1.83
---

# pane titles carry only the pane role

## Problem

The outer terminal tab title is composed by zellij, not pair:

```rust
// zellij-utils/src/shared.rs — hardcoded, no config option exists
format!("\u{1b}]0;{}{}\u{07}",
    get_session_name().map(|n| format!("{} | ", n)).unwrap_or_default(),
    pane_title)
```

So the title is always `<session name> | <focused pane's title>`. The first half
is the zellij session name, which is a **socket filename** capped at **24 bytes**
(verified: 24 accepted, 25 rejected with "session name must be less than 0
characters" — zellij computes its budget minus the long macOS cache path and goes
negative). That budget is why `pair-parley_nvim-parley_nvim` (28 bytes) gets
truncated by `BuildSessionNameCandidates` down to `pair-parley_nv-parley_nv`
(exactly 24). An identifier constrained to 24 bytes makes a poor window title,
and no zellij option suppresses it — every option enumerated, docs checked,
source read. Upstream has had the request open since 2022 (zellij-org/zellij#1495
and #2088, both still open).

The second half, though, is **entirely pair's**: it is the focused pane's title,
settable at runtime via `zellij action rename-pane`, carrying none of the session
name's constraints (verified — a 70-character title containing `/`, spaces, `[]`,
`·` and an em-dash was accepted verbatim). Pair already drives it in two places
and simply never set it for the draft:

| pane | today | where |
|---|---|---|
| agent (startup) | `claude [~/workspace/pair]` | `launcher.PaneTitle` (`format.go:63`) → `PAIR_PANE_TITLE` → `main-3.kdl:54` |
| agent (steady) | `claude (464k) [~/workspace/pair]` | `titlepoller/run.go:197` |
| right terminal | `terminal 1 [a] terminal 3` | `termcmd/run.go:1037` |
| right terminal (renaming) | `[rename: work│] terminal 3` | `termcmd.renamePaneTitleLocked` (`run.go:1057`) |
| draft | *(nothing — falls through to the layout's `name="draft"`)* | `main-{2,3}.kdl` |

**Four** writers, not three, and there are also **two** cwd abbreviators:
`launcher.TildeAbbrev` (`format.go`) and `titlepoller.abbrevCwd` (`run.go:191`).
