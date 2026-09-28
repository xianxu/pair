---
id: 000197
status: open
created: 2026-09-06
updated: 2026-09-21
estimate_hours:
github_issue:
---

# actor label: repo-qualified, one derivation for status bar and switcher

## Problem

**The request.** The status bar names an actor by its repo, so every thread
started in a subdirectory of the same repo reads identically. It should carry
the subdirectory when there is one: `repo` at a repo root, `repo:<last segment>`
below it. For `~/workspace/kbench/competition/arc-agi-3` → `kbench:arc-agi-3`.

**What checking it found: the status bar and the switcher already disagree.**
Two independent derivations exist for one concept, and neither knows about the
other:

| surface | source | value for the example |
|---|---|---|
| status bar chip | `couchtty/console.go:1817` passes `start.Record.Args.Worktree.Repo()`; `Repo()` is `filepath.Base(git toplevel)` (`couchcore/worktree.go:22`) | `kbench` |
| switcher row | `threadLabel(name, WorkingPath, tag)` → `filepath.Base(WorkingPath)` (`couchcore/actionableinventory.go:148`) | `arc-agi-3` |

So the same actor is `kbench` on the chip and `arc-agi-3` in the switcher, and
the operator cannot match one to the other. That is a defect independent of the
requested format, and it is the reason to fix this at the derivation rather than
at the status bar (`ARCH-DRY`). `threadLabel`'s own doc comment already uses
`arc-agi-3` as its worked example — the two surfaces were reasoned about
separately and landed on different answers.

Only the switcher's rule is documented (`atlas/couch.md:743`). The status bar's
`Worktree.Repo()` is undocumented and appears to be incidental — it is what
happened to be in hand at the attach call site, not a decision with a recorded
reason.

**The inputs already exist; no plumbing is needed.** `StartArgs`
(`couchcore/startargs.go`) carries both halves, and its `WorkingDir()` already
encodes exactly the distinction the request turns on:

```go
Worktree  Worktree // git toplevel, resolved via rev-parse --show-toplevel
Cwd       string   // set only when the operator named a subdirectory

func (a StartArgs) WorkingDir() string { if a.Cwd != "" { return a.Cwd }; return string(a.Worktree) }
```

`Cwd == ""` *is* "started at the repo root". The attach site already holds the
whole `start.Record.Args`, so it passes a narrower value than it has.
