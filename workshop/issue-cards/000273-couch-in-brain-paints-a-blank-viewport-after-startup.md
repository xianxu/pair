---
id: 000273
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
---

# couch in brain paints a blank viewport after startup

## Problem

Split out of `pair#265`, whose `## Spec` bullet 4 and `## Done when` #2 were
narrowed on 2026-09-16 to the routing crash. This issue owns the half that is
not yet attributed.

Operator report: `couch` in `/Users/xianxu/workspace/brain` starts, the tab bar
appears, and the main viewport stays fully blank. couch then sits there. A
keypress at that point printed a raw escape sequence at the top-left instead of
being handled. Pressing `ctrl+space` exited couch with
`couch: terminal: terminal: no admitted endpoint` — **that exit is `pair#265`
and is not this issue**; this issue is the blank screen before it, and the
escape-sequence leak.

What the `pair#265` inspection established, which narrows this considerably:

- The console **is** running when the blank viewport shows. `beginConsole`
  dispatches the initial attach *before* `Console.Run`, and an attach failure
  returns an error that `runConsole` prints and exits on (`run.go`), so a
  visible tab bar with no exit means the attach committed and `Run` took the
  `c.switchTo(initial, …)` branch. The blank area is therefore a **painted
  frame of a child that has produced no output**, not an unpainted screen.
- It is **not** couch refusing to spawn. `brain` holds no resumable thread (3
  stale + 1 wedged park, `pair#271`), so startup goes through `spawnResolved`
  and mints a fresh thread. The three stale records dated 09:02, 09:29 and 10:00
  on 2026-09-16 are one per attempt, each with `last_active_at` at the zero time
  — spawned, never became active.
- **The fresh-spawn path is not broken in general.** `pair` also holds no
  resumable thread (5 stale + 1 binding-lost) and couch works there. So whatever
  this is, it is not "couch cannot start a new thread".

Open questions, in the order worth asking:

- Does the spawned child ever write? `brain`'s remembered agent is `muse`
  (`last_agent: muse`, argv `["--trust-workspace"]`) while `pair`'s is `codex` —
  but `pair` has run `muse` too (three live `muse` agents in its scope), so the
  agent alone does not explain it.
- Is the child alive and silent, or dead? A dead child fires `onExit`, and a
  last-actor exit with an actor focused ends the console — which is *not* what
  the operator sees, so "alive and silent" is the likelier branch.
- What is the escape sequence at the top-left, and who wrote it — the parent
  (a reply couch failed to consume) or the child (output painted at the wrong
  origin)?
- How long is "hangs"? `pair#218` measured couch thread startup at 8.85s; a cold
  `muse` in a fresh zellij session could plausibly exceed that without being
  broken.
