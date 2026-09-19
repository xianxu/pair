---
id: 000294
status: open
deps: []
github_issue:
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
---

# Screen snapshot for debugging: chord for the operator, pair screen for the agent

## Problem

When pair or couch draws something wrong, the operator has to describe the screen
to the agent in words, and the agent can't check its own rendering changes. Since
#255, pair and couch keep an exact model of each child's screen:
`terminal.Endpoint.Snapshot(now)` (`cmd/internal/terminal/endpoint.go:275`)
returns a `Frame` with every cell and its style, the cursor and the modes. A
screenshot of what pair drew is one call away, and nothing exposes it.

## Spec

**One capture, two ways to trigger it:**
- **The operator:** a chord (choose a free one from the keyhelp catalog)
  snapshots the screen, saves it, and inserts the file path into the draft, so
  the operator only types a question and sends.
- **The agent:** `pair screen` returns a snapshot of the calling session over a
  local socket (`cmd/internal/notifytransport` is the precedent for reaching the
  running session). It's read-only and limited to the calling session. Under
  couch, other threads' screens are only captured when the operator triggers it.

**Output, saved in the session's artifact dir:**
- **`.txt`:** the plain cell grid. Exact and cheap for an LLM; enough for layout,
  wrapping and missing-text bugs.
- **A styled variant:** a compact annotation (which cells are reverse video,
  which colour, where the cursor is), rather than raw escape codes.
- **`.png`, optional:** needs a font renderer; only for visual glitches.

**Limits:**
- The snapshot is what pair *believes* it drew. Bugs in the gap between that
  belief and what Ghostty shows (#262's redraw and flicker class) won't appear in
  it. A stronger mode captures the physical window as well (macOS `screencapture
  -l <window-id>`) plus zellij's view, and reports where they differ.
- In pair, the endpoint wraps the agent pane. The nvim draft pane and zellij's
  UI are drawn by other processes, so capturing the whole window needs zellij or
  the OS.
- One-off events (clipboard, notifications, bells) aren't in a snapshot.

## Done when

- The chord writes `.txt` plus the styled variant and inserts the path into the
  draft.
- `pair screen` returns the same snapshot to an agent in the same session and
  refuses other sessions. A test covers both.
- The text and styled output match a golden frame in tests.
- The Log records whether the physical-capture comparison is worth building,
  based on at least one real rendering bug.

## Plan

- [ ] Frame → `.txt` and the styled annotation (pure functions), with golden
  tests.
- [ ] The chord plus inserting the path into the draft.
- [ ] `pair screen` over the local socket, limited to the calling session.

## Log

### 2026-09-19

Filed from a brain advisor session.
