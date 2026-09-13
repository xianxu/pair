---
id: 000240
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.47
started: 2026-09-12T19:05:00-07:00
---

# right pane: switching from an nvim tab to a shell tab leaves the pane's mouse mode on, so zellij cannot select in the shell

## Problem

Operator report, 2026-09-12: "I can't select at all in a terminal in the
right pane." The operator keeps nvim (parley) in one `pair term` tab and a
shell in another.

Reproduced with a probe that runs the real `pair term` under a pty and
records every mouse DECSET/DECRST it writes to the outer terminal:

| step | bytes written outward |
|---|---|
| tab 1: `nvim --clean` starts | `?1002h`, `?1006h` |
| Alt+t → tab 2 (shell) | **nothing** |
| Alt+Left → tab 1 (nvim) | `?1002h`, `?1006h` (nvim's own bytes, replayed) |
| Alt+Right → tab 2 (shell) | **nothing** |
| `:q!` in tab 1 | `?1002l`, `?1006l` |

Identical on today's build and on a control build from before #234, so this
is not a regression from today's stdin-loop change; it surfaced because the
operator recently started keeping nvim in a right-pane tab (#227, #234).

Mechanism: `pair term` is the pane's only application from zellij's point of
view. The active child's output passes through to zellij live, so nvim's
`?1002h;?1006h` sets the PANE's mouse mode in zellij. A tab switch
(`applyTakeover`, `run.go:993`) composes `HomeAndClear` + the incoming
child's replay and asserts nothing else — `hostty.repaint` says so
deliberately, because couch owns its host's mouse mode itself. The shell
child never said anything about the mouse, so its replay carries no
`?1002l`, and zellij keeps believing the pane wants mouse reports. It then
forwards every click and drag to `pair term` → the shell, instead of doing
its own selection. Selection is dead in that tab for as long as nvim lives
in the other.

The reverse direction works today only by accident: switching back to nvim
re-enables the mode because nvim's startup `?1002h` happens to still be in
its replay ring. A long-running nvim whose ring has dropped it would come
back with mouse OFF after a round trip.

## Spec

**On every takeover, `pair term` reconciles the pane's mouse modes to the
incoming child's.** It is the proxy for its children toward zellij; the
pane's modes must equal the active child's, and nothing else writes them.

- `ptychild.Screen` models mouse tracking as ONE slot (`off | 1000 | 1002 |
  1003`) plus the SGR-encoding bit, matching what xterm and zellij do with
  the bytes: raising one tracking mode replaces another, and a DECRST of any
  of the three turns tracking off. `Mouse()`/`SGRMouse()`/`MouseObserved()`
  keep their meaning; `MouseModes()` returns the codes held. Additive; couch
  is unaffected.
- `applyTakeover` reads the modes `hostScan` holds — it is fed exactly what
  the pane was shown, live chunks and composed takeovers alike, so it IS the
  pane's state — before the scan reset, and prefixes the composed repaint
  with `mouseReconcile(held, want)`: one write per axis (a `l` of the held
  tracking mode or a `h` of the wanted one; a `h`/`l` of 1006), nothing for
  equal states. The prefix is fed to `hostScan` with the rest of the
  composition, which is also what makes the next takeover's read correct.
- The policy lives in `termcmd`, not `hostty.repaint`: the two consoles
  differ here by design (couch asserts its OWN mouse mode on its host;
  `pair term` has none of its own and mirrors its children), and the
  shared primitive stays free of either policy (ARCH-PURE at the seam).
- Out of scope: the separate nvim-in-right-pane symptom (no live highlight
  until release, which-key popup), which the drag probe could not reproduce
  through `pair term` and which points at the couch layer; tracked next.

## Done when

- `probes/mousemodesmoke` (in tree) shows a DECRST of nvim's modes on Alt+t
  to the shell tab and the grouped DECSET on the way back. The grouped form
  is the reconcile prefix; nvim's own replayed startup bytes are two
  separate writes, so the probe distinguishes "asserted" from "replayed".
- `mouseReconcile` is tested over the full held × want product (4 tracking
  values × SGR bit, squared), with `ptychild.Screen` as the oracle: feed the
  held state, feed the prefix, the Screen must hold the wanted state.
- Unit test on `terminalMux`: outgoing child holding `1002+1006`, incoming
  child silent → pane receives the DECRST before the repaint; the reverse
  switch receives the grouped DECSET; silent → silent writes no mode bytes.
- Closing the nvim tab (`removeTab` takeover to the survivor) releases the
  mode the same way.
- Live: the operator selects text at the shell tab with nvim alive in the
  other tab.

## Plan

- [ ] `Screen`: one tracking slot + SGR bit; `MouseModes()` + tests
- [ ] `applyTakeover`: `mouseReconcile(hostScan's modes, incoming's)` as the prefix; feed hostScan
- [ ] Mux tests for the three transitions + the closed-tab case
- [ ] `probes/mousemodesmoke` in tree; run it both directions, then the operator's live check

## Revisions

### 2026-09-12 — plan-quality round 1

**Reason.** PQ-1: diff against `hostScan`, not a new outgoing-child record —
`hostScan` is already fed exactly what the pane was shown, so it is the pane's
state with no race to name. PQ-2: 1000/1002/1003 are one variable in xterm
and zellij; a set-diff would emit wrong bytes after `?1002h` then `?1000l`.
**Delta.** Spec bullets 1–2 rewritten to the one-slot model and the
`hostScan` baseline; the pure function is `mouseReconcile(held, want)`,
tested over the full product with `Screen` as the oracle (PQ-3); the probe is
landed in tree as `probes/mousemodesmoke` (PQ-4); `hostty.repaint`'s comment
now points at the reconciliation (PQ-5).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Design at ×0.2 (mechanism reproduced and the rule stated); impl at 40% of v2; +15% buffer.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.04 impl=0.12
item: smaller-go-module  design=0.06 impl=0.16
item: milestone-review   design=0.00 impl=0.08
design-buffer: 0.15
total: 0.47
```

- `Screen` mode set + accessor — 0.04 / 0.12
- takeover reconciliation + mux tests + probe — 0.06 / 0.16
- close review — 0.00 / 0.08

## Log

### 2026-09-12

- Filed from the operator's report. First ruled out today's #234 change
  (a drag through `pair term` reaches nvim live on both builds, mode `v`
  mid-drag), then reproduced this with a tab-switch probe: no DECRST is
  written when the active tab changes from nvim to a shell.
