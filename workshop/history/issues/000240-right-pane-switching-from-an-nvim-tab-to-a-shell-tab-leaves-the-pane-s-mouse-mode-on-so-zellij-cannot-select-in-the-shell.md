---
id: 000240
status: done
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.47
started: 2026-09-12T19:05:00-07:00
actual_hours: 1.47
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

**On every takeover, `pair term` reconciles the pane's MOUSE modes to the
incoming child's.** It is the proxy for its children toward zellij. This issue
scopes to mouse modes, which is what the reported bug needs; the pane also
carries the child's OTHER private modes (focus `?1004`, bracketed paste
`?2004`, cursor-key mode), and reconciling the full set across every takeover
path is the follow-up #241 (BR-1, BR-3). The excluded set here is alt-screen
and cursor-save, which the replay/repaint already own.

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
  to the shell tab and the DECSET twice on the way back: once as the
  reconcile prefix, once as nvim's replayed startup bytes. A build without
  the fix writes nothing on the way there and the DECSET once on the way
  back, and only while the ring still holds it.
- `mouseReconcile` is tested over the full held × want product (4 tracking
  values × SGR bit, squared), with `ptychild.Screen` as the oracle: feed the
  held state, feed the prefix, the Screen must hold the wanted state.
- Unit test on `terminalMux`: outgoing child holding `1002+1006`, incoming
  child silent → pane receives the DECRST before the clear; the reverse
  switch receives the DECSET before the clear (the replay's copy comes
  after it, which is how the test tells asserted from replayed); silent →
  silent writes no mode bytes.
- Closing the nvim tab (`removeTab` takeover to the survivor) releases the
  mode the same way.
- Live: the operator selects text at the shell tab with nvim alive in the
  other tab.

## Plan

- [x] `Screen`: one tracking slot + SGR bit; `MouseModes()` + tests
- [x] `applyTakeover`: `mouseReconcile(hostScan's modes, incoming's)` as the prefix; feed hostScan
- [x] Mux tests for the three transitions + the closed-tab case
- [x] `probes/mousemodesmoke` in tree; run it both directions, then the operator's live check — probe run both directions on this build and on a control from main (Log); the in-pane check is the operator's step after install

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
- 2026-09-12: closed — Mouse reconcile on takeover. Red on main then green: TestSwitchingTabsReconcilesThePanesMouseModesToTheIncomingChild, TestClosingATabReleasesTheDeadChildsMouseModes, TestMouseReconcileReachesEveryWantedStateFromEveryHeldState (full 8x8 product, ptychild.Screen as oracle), TestScreenMouseModesIsOneTrackingSlotPlusEncoding, TestPrivateModesFormatsOneDECSETOrDECRST. Full make test green incl the paint-gate consumer guard. LIVE: probes/mousemodesmoke on real pair term + nvim under a pty writes ?1002l ?1006l on Alt+t to the shell and ?1002h ?1006h back; control from main writes nothing there. Operator confirmed shell selection works after restart (logged, BR-2). SCOPE: narrowed to mouse modes (the reported symptom); the full pane-mode sweep (focus 1004, bracketed paste 2004, cursor-key mode) and the removeTab-rename-open path are filed as #241 per the review dispositions of BR-1/BR-3.; review verdict: SHIP

- Filed from the operator's report. First ruled out today's #234 change
  (a drag through `pair term` reaches nvim live on both builds, mode `v`
  mid-drag), then reproduced this with a tab-switch probe: no DECRST is
  written when the active tab changes from nvim to a shell.
- Fix landed: `ptychild.Screen` holds tracking as one slot (`off|1000|1002|
  1003`) plus the 1006 bit, `MouseModes()` reads it back; `hostty.PrivateModes`
  formats one DECSET/DECRST; `termcmd.mouseReconcile(held, want)` is one
  write per axis, tested over the full 8×8 held×want product with `Screen`
  as the oracle; `applyTakeover` reads `hostScan.MouseModes()` before the
  reset and prefixes the composition with it. The paint-gate consumer guard
  now allows `MouseModes()` as a mode read rather than a gate read.
- `probes/mousemodesmoke`, this build: Alt+t to the shell writes
  `?1002l ?1006l`; Alt+Left back writes `?1002h ?1006h` twice (prefix, then
  nvim's replayed startup). Control build from main: nothing on the way
  there, the DECSET once on the way back. Full `make test` green outside the
  sandbox.
- Plan-quality took three rounds: PQ-1 (diff against `hostScan`, not a new
  field) and PQ-2 (one tracking slot, not a set — `?1002h` then `?1000l` is
  tracking OFF in xterm and zellij) both changed the design for the better
  and are recorded under Revisions.

### 2026-09-12 (close)

- **Operator confirmation (BR-2):** after restarting into this build, the
  operator reported mouse selection works in the right-pane shell — "able to
  restart pair (inside couch) after couch restart ... selection in right
  pane's terminal." The Done-when's live shell-selection check is met.
- **Scope narrowed (BR-3, BR-1):** the close review measured that a switch
  from nvim to a shell releases only `?1002l ?1006l`, leaving nvim's `?1004h`
  (focus), `?2004h` (bracketed paste) and cursor-key mode on the pane, and
  that `removeTab` with a rename open skips the reconcile. Those are the
  general "pane mirrors the active child's FULL mode set across ALL paths"
  work, filed as #241. This issue is deliberately the mouse-on-switch/close
  slice that fixes the reported symptom; the Spec header is narrowed to say so.
