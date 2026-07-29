---
id: 000133
status: open
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours:
---

# agent pane title drops the cwd suffix

## Problem

zellij composes the outer terminal tab title as `<session name> | <focused pane
title>` (hardcoded in `zellij-utils/src/shared.rs`; no config option, upstream
request open since 2022 — zellij-org/zellij#1495, #2088).

#130 made the session half carry the folder and the tag: `ComposeSessionName`
(`session_index.go:75`) emits `📁{repo}` plus any residual tag tokens. Verified
live — `PAIR_SESSION_NAME=📁pair`, and `zellij list-sessions` shows `📁pair`,
`📁ariadne`, `📁kbench`.

The agent pane still appends `[<tilde-cwd>]`, so the tab reads:

```
📁pair | claude (629k) [~/workspace/pair]
```

The repo is named twice. Three of the five Pair-owned surfaces already match the
target — only the agent pane is off:

| pane | writer | title today | on target? |
|---|---|---|---|
| agent (startup) | `launcher.PaneTitle` `format.go:61` → `PAIR_PANE_TITLE` → `main-{2,3}.kdl` | `claude [~/workspace/pair]` | no |
| agent (steady) | `titlepoller.frameTitle` `titlepoller.go:72` | `claude (629k) [~/workspace/pair]` | no |
| draft | layout `name="draft"` | `draft` | yes |
| right terminal | `termcmd.paneTitleLocked` `run.go:1042` | `terminal 1 [a] terminal 3` | yes |
| right terminal (`Alt+r`) | `termcmd.renamePaneTitleLocked` `run.go:1057` | `[rename: work│] terminal 3` | yes |

**Two** writers emit the suffix, not one. The startup writer matters on its own:
`PAIR_PANE_TITLE` is what the pane is named at launch, so leaving it in place
shows the old shape from launch until the poller's first pass.

## Spec

Target tab title:

```
📁pair        | claude (629k)
📁pair-bugfix | claude (629k)
📁pair        | draft
```

The session half carries the folder and tag, so the pane half carries **only the
pane's own identity**. Anything more is the same fact twice.

- `titlepoller.frameTitle` → `<agent> (<count>)`, or `<agent>` when no count
  resolved. Drop the `[%s]`.
- `launcher.PaneTitle` → `<agent>`. Drop the `[<tilde-cwd>]`.
- **Delete `abbrevCwd`, don't consolidate it.** `titlepoller.abbrevCwd`
  (`titlepoller.go:57`) and `launcher.TildeAbbrev` (`format.go:47`) are
  byte-identical twins, but `abbrevCwd`'s only non-test caller is `run.go:194` —
  the empty-`cwd_display` fallback that exists purely to feed `frameTitle`'s
  `cwdDisp`. Dropping the suffix makes it dead code, so the `ARCH-DRY`
  duplication resolves by **deletion**. `TildeAbbrev` stays in `launcher`: it has
  a live caller at `createflow.go:469` (`PAIR_PANE_CWD`) and keeps its test.
- Decide the now-reader-less cwd chain: `PAIR_PANE_CWD` (`createflow.go:469`) →
  `cwd_display` in `pane-*.json` (`main-2.kdl:45`, `main-3.kdl:54`) →
  `Pane.CwdDisplay` (`run.go:32`, `runtime.go:74`). Keep it as pane-identity
  telemetry or strip the chain — but decide explicitly rather than leaving a
  write-only field.
- Out of scope: the draft pane (already correct), the right terminal (already
  correct), and the session half (zellij's, and #130 settled it).

### Accepted consequence

The pane **frame** renders the same string as the tab title, so the agent frame
goes from `claude (629k) [~/workspace/pair]` to `claude (629k)` — the cwd leaves
the frame you look at while working and lives only in the tab title. Confirm at
the live check; if it reads as a loss, the fallback is to keep the frame long and
accept the duplication in the tab.

## Done when

- Agent pane title is `claude (629k)` at startup AND in steady state; the tab
  reads `📁pair | claude (629k)`, verified live by focusing the pane and reading
  the Ghostty tab title in **both** layouts (2 and 3).
- Draft and right-terminal titles are unchanged, including mid-`Alt+r`.
- `abbrevCwd` and `TestAbbrevCwd` are gone; `TildeAbbrev` and `TestTildeAbbrev`
  remain, and the `cwd_display` chain decision is recorded in `## Log`.
- `go test ./cmd/internal/titlepoller/ ./cmd/internal/launcher/` green, and full
  `make test` exit 0 (run with `env -u PAIR_SESSION_ID -u PAIR_TAG` — the
  in-session env leaks into review-target and changelog tests).

## Plan

Single review boundary — no `Mx` tags: one branch, one close.

- [ ] Drop `[%s]` from `titlepoller.frameTitle` (`titlepoller.go:72`). Update
      `TestFrameTitle` (`titlepoller_test.go`) and the `run_test.go` title
      assertions.
- [ ] Drop the suffix from `launcher.PaneTitle` (`format.go:62`) and update
      `TestPaneTitle` (`format_test.go:61`). This is the startup title exported
      as `PAIR_PANE_TITLE`; without it the tab shows the old shape on every
      launch until the first poll.
- [ ] Delete `abbrevCwd` + `TestAbbrevCwd` and its call site at `run.go:194`,
      leaving `pane.CwdDisplay` unread. Decide and record the `cwd_display`
      chain disposition (keep as telemetry vs strip through the two KDLs).
- [ ] `atlas/architecture.md`: the tab title is `<session name> | <focused pane
      title>`; the session half is zellij's, shaped by #130's `📁{repo}-{tag}`;
      the pane half deliberately carries only the pane role. One cwd abbreviator
      (`launcher.TildeAbbrev`), whose only consumer is `PAIR_PANE_CWD`.
- [ ] Live check in both layouts: focus each pane in turn, read the tab title.

## Log

### 2026-07-29

- Supersedes #129, which is `wontfix`. #129 opened with the inverse spec — put
  `~/workspace/pair [work] · ` in FRONT of all four pane titles — built it
  (`97aacb5`), then reverted (`ac6b879`) once the operator pointed out it
  duplicates what `📁{repo}-{tag}` carries. Its re-scope commit (`33845c7`)
  already described this issue's spec. Branch abandoned rather than rebased;
  history preserved at tag `abandoned/000129-prefix-scope`.
- **Two findings carried forward from #129, both worth not re-deriving:**
  - A pane name **survives `next-swap-layout`** — measured directly by renaming
    the live draft pane, cycling the rung with `Alt+↑`/`Alt+↓`, and reading it
    back. So the six `pane name="draft"` re-declarations in `main-3.kdl` do not
    reset a runtime rename, no draft self-rename is needed, and the
    `nvim/init.lua` `layout_step` fallback #129's plan feared is unnecessary.
  - Consequently `frameCache` (`titlepoller.go:114-124`) needs no invalidation:
    nothing re-tiles a title away, and the poller respawns per session entry, so
    there is no long-lived stale-cache window.
- Why #129's `titlefmt` consolidation is not carried forward: #130 **deleted**
  `cmd/internal/titlefmt` outright when it retired `EmojiTitle` (that function
  was the package's only content). Reviving the package to host `TildeAbbrev`
  would add an indirection for a single caller — hence deletion of the duplicate
  instead.
