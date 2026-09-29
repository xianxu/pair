---
id: 000343
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: 'f20d5855ed337732fd6ea90b38f081cfeb4ad3be' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch idle fading: tune colours; verify light theme and NO_COLOR

## Problem

#247 shipped idle fading for live Couch threads (1 day / 3 days, blending
toward the terminal's reported background). The operator's live smoke on
2026-09-28 confirmed distinct shades on their dark scheme and deferred colour
tuning. Four things were left open:

- **Colour tuning.** The blend weights are 0 / 40 / 65 % toward the background
  (`idleBlend`, `cmd/internal/couchtty/idle_shade.go`), a single table.
- **Light theme, unverified live.** On a white background the faded amber is
  about `#fff1a6`, which may be illegible. Tests prove fading moves toward the
  background, but not that it stays readable.
- **`NO_COLOR`, unverified live.** It's covered by unit tests; no live run yet.
- **Probe cost.** Measured at about 114 ms per thread (56 threads in 6.4 s),
  about 2.3 s per pass for 20 threads, over #247's 2 s budget. It runs on a
  background worker once a minute. The planned fix is one native-store listing
  per agent per pass, shared across threads (`threadactivity/os.go` rebuilds
  the session inventory runtime per probe).

Also observed during the smoke: `brain:0` (a codex thread launched 09-27) has
no established session binding, so its transcript isn't a signal and only sends
and the launch count. Direct typing into its agent pane doesn't refresh it.
Threads launched after #329 get their binding again; older unbound ones stay
send-only until relaunched.

## Spec

- Tune `idleBlend` with the operator, live, on their dark and light schemes.
  Consider capping the amber's weight separately if light-theme amber washes
  out.
- Live-check `NO_COLOR=1 couch`: no fade escapes, and today's other styling.
- Decide the probe cost: accept it, or build the shared listing per pass.

## Done when

- The operator accepts the fade on both a dark and a light scheme, and the
  chosen weights are recorded here.
- `NO_COLOR` is checked live.
- The probe-cost decision is recorded, with a measurement if the shared listing
  is built.

## Plan

- [ ]

## Log

### 2026-09-28

- Split from #247's close: the operator passed the smoke on visible shades and
  deferred tuning.
