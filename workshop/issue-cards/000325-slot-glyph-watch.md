---
id: 000325
status: open
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
github_issue:
---

# Solidify repo state's glyph

## Problem

The slot quick-status glyph (#317, #319) gets its data from polling. Couch
runs `git --no-optional-locks status --porcelain=v2 --branch` in every slot
every 10 s (`defaultSlotGitInterval`, `cmd/internal/couchtty/console_slotgit.go`).
It never touches the network. That causes two problems:

- **Local changes show up late.** A commit, checkout or push appears up to
  10 s after it happens. Shortening the interval would cost N slots × more
  `git status` runs.
- **Remote changes don't show up at all.** The behind count (`-`, `±`) is only
  as fresh as the checkout's last `git fetch`. Someone else pushing to `main`
  is invisible until the operator fetches by hand.
