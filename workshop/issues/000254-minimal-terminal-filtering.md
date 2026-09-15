---
id: 000254
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
---

# Minimize terminal escape filtering

## Problem

Codex scrolling has remained inconsistent, and the operator reports flashing
and screen artifacts. Pair currently strips Codex synchronized-update controls
(ESC[?2026h/l) and focus-reporting controls (ESC[?1004h/l) by default. An optional
Codex filter strips specific keyboard negotiation sequences. Claude bypasses
these agent-specific output filters. Couch separately strips Ctrl from vertical
mouse-wheel reports for every agent to prevent Zellij pane resizing.

Screenshot-local investigation in #252 found valid source UTF-8 and confirmed
that the running wrapper stripped synchronized-update boundaries during repeated
redraws. This is a plausible flashing contributor, not an established cause.
The confirmed downstream UTF-8/control-injection defect remains owned by #252.


## Spec

Operator direction: minimize filtering and preserve native terminal behavior.
Audit input rewrites and output filtering across Pair and Couch, documenting
which layer owns each behavior and whether it differs for Claude and Codex.
Prefer passthrough by default; retain only narrowly scoped workarounds supported
by a reproducible current failure, with tests and a clear removal condition.

Evaluate synchronized-update/focus-report passthrough first using the existing
PAIR_CODEX_SYNC_PASSTHROUGH switch. Separate the two controls if evidence shows
they require different policies. Audit the opt-in keyboard-negotiation filter
and stale per-thread marker activation. Check whether the installed/supported
Zellij versions can replace Couch's Ctrl-wheel rewrite with configuration;
do not assume an older workaround remains necessary or a newer option exists.

Compare Claude and Codex under the same controlled scroll/redraw scenarios.
Preserve necessary Couch-owned shortcut and terminal-mode behavior. Treat
control injection and UTF-8 framing as separate from byte filtering: coordinate
with #252 rather than attributing all artifacts to the filters. Record actual
active settings/version and evidence before changing defaults (ARCH-DRY,
ARCH-PURE). No live operator-thread experimentation without a recoverable setup.


## Done when

- A concise inventory names each filter/rewrite, its reason, activation, owner,
  current evidence and removal condition.
- Controlled before/after tests cover scrolling, repeated redraws, focus changes,
  keyboard negotiation, Ctrl-wheel resizing and session attachment stability.
- Unnecessary filters are removed or disabled by default; any retained exception
  has a reproducible regression test and documented scope.
- Claude/Codex comparison and operator smoke establish usable behavior; unknown
  flashing or disconnect causes remain explicitly unproven.
- #252 retains its independent UTF-8 regression coverage and implementation.


## Plan

- [ ] Audit current filters and supported terminal/Zellij behavior.
- [ ] Design and test the smallest justified set of transformations.
- [ ] Implement approved changes, verify comparative smoke, and close via SDLC.


## Log

### 2026-09-14

Created at operator request after TTY investigation; queued, not implemented.
Related: #252 UTF-8 output boundaries, #253 disconnect telemetry. Evidence:
/tmp/pair-display-sync-window.raw and /tmp/pair-display-sync-events.json;
#252 records exact source offsets and timestamp. Existing #250 recovery work
continues; this capture does not reorder the previously agreed #245 follow-up.


### 2026-09-14 — Mouse drag highlight trace: mode restoration is distinct from filtering

Operator reports selection highlight appears only upon release, throughout the
agent and right panes. Read-only /private/tmp/couch-mouse-207.log inspection:
13:08:35 actor output establishes 1003,1006; 15:16:04.382–04.886 Couch emits six
successful click-only 1000,1006 assertions while the panel retains a child whose
tracking becomes false. Actor takeovers at 15:21:39.879 and 15:21:47.618 replay
131072 bytes without mouse-mode setters; no later restoring child-mode event
was found through the inspected trace. Current sole trace writer is Couch20191,
started13:08:32. The trace does not identify every historical writer or include
mouse-input reports, so exact symptom onset/current terminal mode is unproven.

Code: takeover clears/replays without restoring incoming child's recorded mouse
modes; couchMayOwnTheMouse sees that child wants tracking and declines further
writes. Click-only reporting suppresses drag motion, closely matching the
symptom. This is a strong mode-leak diagnosis, separate from Ctrl-wheel modifier
filtering and #252 UTF-8 corruption. scanner=none is not host truth: takeover
resets it and own mouse assertions bypass it. Recent no-output-terminal errors
wrote zero bytes and therefore did not change terminal modes. No runtime or
code changes during inspection. Any minimal-filtering solution must still
restore terminal modes it changes when ownership returns to the child.
