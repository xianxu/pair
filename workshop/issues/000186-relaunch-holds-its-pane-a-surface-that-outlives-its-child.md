---
id: 000186
status: open
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
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

## Spec

A pane that can exist without a live child. While a relaunch owns it, the entry
stays in `c.panes`, keeps its slot in `c.order` and keeps `c.active`, and renders
`relaunching <label>…` driven by the spinner that already animates in
`menu_render.go`. The adopted child then takes the slot back.

**A frozen glyph is not a status page.** The spinner must ADVANCE: a still frame
is indistinguishable from the hang this surface exists to rule out, so the test
asserts motion rather than presence.

Three places assume a pane implies a live child and must distinguish "dead and
being replaced" from "dead and gone", or adoption will refuse the very pane it is
replacing:

- `installObservedThreadActor` (`console.go:346`) refuses a second pane on a
  thread unless the first is `Done()`.
- `switchTargetForAddressLocked` (`console.go:1643`) will not switch to a pane
  whose child is `Done()`.
- `activeChild` returns `p.child` and `route` writes to it; keystrokes during the
  gap need somewhere to go that is not a dead pty.

Consequence 2 falls out for free — no exit, no `Drop` — but gets its own test
rather than being assumed. `pair#182` produced five instances of a case that was
declared and not reachable; "it follows from the change" is exactly the reasoning
that failed there.

## Done when

- The pane, its slot and `c.active` survive the relaunch child's exit.
- The operator sees `relaunching <label>…` rather than a blank screen, and the
  spinner advances.
- `ctrl+backspace` after a relaunch lands where it did before it.
- No exit notice is published for a relaunch the operator asked for, driven
  through the production input path.
- `Alt+n` and `Ctrl+Alt+n` appear in `menuControls` and the README, so no key
  ships undocumented; the atlas carries the holding pane and the
  scope-follows-focus rule.
- Real-stack verification with a rebuilt binary: PID changed, the binary is the
  rebuilt one (mtime), the ledger's native session id is UNCHANGED, and no blank
  screen was seen. Recorded as a measurement in `## Log`, not as "it worked".

## Plan

- [ ] The holding pane: `paneState`, the `onExit` branch that keeps it, and
      `RenderHoldingPane`. Test the property (the pane and slot survive), then
      that the spinner advances.
- [ ] The three live-child assumptions above, each with a test that the
      replacement is adopted into the held slot rather than refused.
- [ ] `previous` not spent, and no exit notice — each pinned, neither assumed.
- [ ] Docs: `menuControls`, README, atlas.
- [ ] Real-stack verification, recorded as a measurement.

## Log

### 2026-09-04

Split out of `pair#182` M2 so that M1 — relaunch as an operation, which is
delivered and smoke-tested — can land on its own. `pair#182`'s branch was 111
commits by then, and carrying a working feature behind an unstarted surface is
how a branch becomes unreviewable.

Already delivered by `pair#182`, so NOT in scope here: the `Alt+n` /
`Ctrl+Alt+n` interception with `Alt+Shift+N` left to Pair, and the six-site
sweep (`TestEveryOfferedActionIsReachableFromEnter`), which M1's round-3 review
pulled forward after the same defect shape appeared five times.

Open questions carried over from `pair#182`'s plan, recorded rather than decided:
whether relaunch should be offered on a DETACHED row (its agent runs but couch
hosts no client, so it means reattach-with-the-new-binary — reattach then
relaunch for now), and whether the holding pane should keep the dead child's last
frame behind the status page (the scrollback belongs to the `Child` being
discarded; deferred until the surface exists and can be looked at).
