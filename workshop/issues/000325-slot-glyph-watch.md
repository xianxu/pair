---
id: 000325
status: open
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
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

## Spec

Two independent parts. Both only trigger a re-probe; neither holds state.
`ProbeSlotGit` stays the only producer of `SlotGitStatus`.

### 1. Local: watch git's metadata

- Watch **git's metadata directories, not the working tree**. On macOS,
  kqueue takes one file descriptor per watched file or directory and can't
  watch recursively, so a whole worktree (`node_modules` included) would run
  out of descriptors. Watch these directories:
  - The worktree's own git dir (`.git/worktrees/<name>/`): `HEAD` (branch or
    detached) and `index` (staged changes, commits).
  - The shared git dir (`--git-common-dir`): `refs/heads/**`,
    `refs/remotes/**` and `packed-refs` (ahead and behind).
- Watch **directories, not files**. git updates a ref by writing a `.lock`
  file and renaming it into place. A watch on the old file is lost at the
  rename.
- **Debounce** events (~150 ms), then re-probe the affected slots. An event in
  the shared git dir re-probes every slot of that repository.
- **Keep the poll as a backstop**, raised to about 30–60 s. The watcher can't
  see unstaged edits (the `*` part), and it can drop events when its queue
  overflows.
- Library candidate: `github.com/fsnotify/fsnotify` (uses inotify, kqueue or
  ReadDirectoryChangesW depending on the platform). Before adding it, vet its
  latest release and activity and name the alternatives. FSEvents (recursive,
  and would also cover `*`) needs a cgo wrapper; that's out of scope for now.
- Watch lifetime follows the slot inventory. A slot added to the catalog
  gains a watch, and a removed slot releases it (ARCH-FUNERAL: no watch
  outlives Couch or its slot).

### 2. Remote: one poller per repository

- Slots of one repository are worktrees that share `refs/remotes/*`. So remote
  freshness belongs to the **repository** (keyed by `--git-common-dir`), not
  to each slot: one poller per repository.
- First version: read only, with `git ls-remote origin <upstream branches>`.
  Compare the remote commit IDs to the local tracking refs. If they differ,
  the glyph shows "remote moved". Nothing is written to the shared git dir,
  so there's no ref-lock race with the operator's own `git pull` or `fetch`.
- Whether to follow up with an actual `git fetch` (which gives a real behind
  count) is an open design question: fetch automatically, or have an operator
  action trigger it? If it fetches, the watch from part 1 refreshes every slot
  for free.
- Run non-interactively (`GIT_TERMINAL_PROMPT=0`, `GIT_SSH_COMMAND` with
  `-o BatchMode=yes`) with a timeout. A network or auth failure is display
  evidence ("remote unknown"), never a notice storm.
- Cadence: every few minutes with jitter, plus once when Couch regains focus
  if that signal is available.

## Done when

- A commit, checkout, or push in a slot updates its glyph in under 1 s,
  without waiting for the backstop poll. A test using a fake watcher proves the
  event→re-probe path and the debounce.
- A change to shared refs (for example `git fetch` in any slot) re-probes every
  slot of that repository once.
- Watches are released when a slot leaves the inventory and when Couch exits
  (a test pins this).
- A remote push to a slot's upstream shows up as "remote moved" within one
  remote-poll period. Exactly one poller runs per repository, no matter how
  many slots it has. A test with a fake runner proves this.
- Operator smoke test in a live Couch with ≥2 slots of one repository.

## Plan

- [ ]

## Log

### 2026-09-24

- Filed from a Couch session discussion. It's related to the 10 s slot git
  pass from #317 and the glyph shape from #319.
