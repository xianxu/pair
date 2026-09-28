---
id: '000227'
status: done
started: 2026-09-13T09:18:29-07:00
created: 2026-09-10
updated: 2026-09-13
estimate_hours: 0.75
actual_hours: 2.29
---

# right-pane chords pass through to a full-screen app

## Problem

The operator runs nvim with parley in the layout3 right pane. Parley binds `<M-t>` to
its outline; `pair term` binds `<M-t>` to "new terminal tab" and takes it first, so
parley never sees it.

The operator's first proposal was to move pair's new-tab chord to `<M-n>`. **That does
not work**, and the reason reframes the issue.

### Why `<M-n>` is unavailable

`workbenchshortcut.Decide` resolves **global** chords before role-scoped ones, for the
right terminal as well (`shortcut.go`):

```go
if in.Role == PaneRoleLeftAgent || in.Role == PaneRoleLeftDraft || in.Role == PaneRoleRightTerminal {
    if decision, ok := DecideGlobal(in.Chord); ok {
        return decision
    }
}
```

`<M-n>` is global — `ChordAltN → ActionRestartPair`, *"reload pair — kill and
re-launch the workbench"*. A new-tab binding on `<M-n>` inside the right-terminal block
would be unreachable: pressing it would raise the *Reload pair?* confirm instead. And
even a free chord would only move the collision to whatever else nvim binds there.

### The collision is wider than `<M-t>`

In the right-terminal role, `pair term` takes these before any child sees them:

- **handled** (turned into pane actions): `<M-t>`, `<M-w>`, `<M-r>`, `<M-S-d>`,
  `<M-k>`, `<M-S-Enter>`, `<M-Left>`, `<M-Right>`;
- **swallowed** (dropped): `<M-j>`, `<M-/>`, `<M-S-c>`, `<C-M-c>`.

So `<M-t>` is the collision the operator noticed. Every full-screen app in the right
pane loses the whole set.
