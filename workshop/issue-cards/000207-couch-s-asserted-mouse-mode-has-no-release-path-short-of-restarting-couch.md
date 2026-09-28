---
id: 000207
status: working
started: 2026-09-12T23:21:00-07:00
created: 2026-09-06
updated: 2026-09-12
estimate_hours: 0.773
github_issue:
---

# couch's asserted mouse mode has no release path short of restarting couch

## Problem

Operator report, 2026-09-06: copy-on-select highlighting in the **agent pane**
stopped working. Three recovery gestures were tried, in order:

| gesture | result |
|---|---|
| `alt+n` → switch to another actor and back | **did not fix** |
| `alt+n` → **relaunch** the actor (`#182`: restart pair in place, keeping the agent conversation) | **did not fix** |
| exit couch, start couch again | **fixed** |

That ordering is the finding. A relaunch mints a **new child** — new pty, new
`ptychild.Screen`, so couch's *belief* about the child returns to `unknown`,
where `#196`'s rule says stand back. It did not help. Only killing couch itself
did.

**So the stale thing is not the belief; it is the assertion.** The likely shape:
couch writes its own mouse mode to the HOST terminal once it believes the child
holds none, and **nothing ever writes the release**. Re-observing the child
cannot undo a sequence already sent to the host. Restarting couch works because
process exit tears down the host's terminal state, not because any code path
reconsidered anything.

If that is right, the four rounds `#172` spent on the *observation* side
(BR-16/22/26/33) were all upstream of this: they made couch decide correctly
*when* to assert, and left the assertion permanent once made.

### How it was triggered, reproducibly

Not a mystery — it was caused on purpose, by accident. A throwaway probe ran in
the right pane with mouse reporting on and, on exit, "restored" the prior state
by writing `?1000l` + `?1006l`. Those bytes travel up through zellij into
couch's child stream, where `Screen` scans DECSETs. couch observed a well-formed
"the child turned mouse OFF", moved to *observed-none*, and asserted.

Nothing malfunctioned. The tri-state answered the question it was asked, and the
question is the problem: **`Screen` reports what the byte stream SAID, not what
the child INTENDS.** A transient in-pane process that tidies up after itself is
indistinguishable from the long-lived child changing its mind.

That gives this class a reproducer for the first time, where the four historical
rounds each had only a symptom.
