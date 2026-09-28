---
id: 000186
status: open
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
github_issue:
---

# Relaunch holds its pane: a surface that outlives its child

## Problem

`pair#182` M1 shipped relaunch and it works — the operator smoke-tested it, and
the ledger proves the agent conversation survives while the Pair process and
zellij session are replaced. What it does not do is LOOK like one operation. The
pane vanishes for the seconds Pair takes to boot and then reappears.

The operator named the cost before the code existed: *"pair's boot isn't instant,
and a genuinely blank page for those seconds is indistinguishable from a hang. It
wants to be a status page — 'relaunching <thread>…' — not a blank one."*

Three symptoms, one cause. `Console.onExit` deletes the pane unconditionally
(`console.go:802`), so a child's death takes the pane, the operator's slot
(`c.active`) and their `ctrl+backspace` target with it:

1. **The blank screen.** Focus falls to the switcher and the actor's slot is gone
   until the replacement is adopted.
2. **`previous` is spent.** `onExit` calls `tracker.Drop` unconditionally, so a
   relaunch empties the return target even though the operator never left the
   thread — which contradicts `SwitchTracker`'s own doc comment.
3. **A spurious "exited" notice** is possible for work the operator asked for.
   `endsItsOwnChild` suppresses it today, but only by naming the operation; with
   no exit there is nothing to suppress.
