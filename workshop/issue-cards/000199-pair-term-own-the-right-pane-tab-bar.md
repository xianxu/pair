---
id: '000199'
status: done
started: 2026-09-06T19:24:15-07:00
created: 2026-09-06
updated: 2026-09-08
estimate_hours: 7.52
actual_hours: 24.41
---

# pair term: own the right pane tab bar

## Problem

`pair term` has tabs but no tab strip. It renders their state by shelling out
to zellij:

```go
// termcmd/run.go:959-963
return m.rt.RunZellijAction("rename-pane", "--pane-id", m.paneID, title)
```

Three costs, and the third is the one that matters:

- A subprocess per title change.
- Appearance is zellij's pane frame, not ours.
- **One string carries all tab state.** `paneTitleLocked` packs the whole tab
  set into a single rename argument because that is the only channel available.

So the operator's complaint — "the way we change tab title is not great" — is
not a polish problem. There is no surface to polish: a tab strip was never
built, and a rename call is being asked to stand in for one.

**The mechanism already exists and is already proven.** couch reserves a row
with DECSTBM and confines its child above it — `couchtty/reserve.go`
(`Reserve`, `PaintRow`) over `hostty.SetRegion` — and repaints when
`ptychild.Screen.TakeRowDirty()` says a child may have wiped it. That is a
status bar, built, working, and hardened by a full mouse round in `#172`.

**And `pair term` is the process that can use it.** couch cannot reach into the
right pane: its child is the whole zellij session, and zellij owns that pane's
rendering. `pair term` *is* the process in the pane, owns its pty, and already
drives the host seam (`hostty.NewOSHost`, `termcmd/run.go:236`). Everything the
reserved row needs is one package away from a caller that already has the other
half.
