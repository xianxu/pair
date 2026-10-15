---
id: 000576
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '74a9808b204bc1becc050b543c890b4e634881b2' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch switcher: lifecycle actions run in the background per slot; the UI never blocks

## Problem

While the operator runs a lifecycle action in Couch's switcher (bring a slot up, relaunch, resume), the switcher holds
the screen until the action finishes or fails: the operator can't switch to another slot or the hub meanwhile
(operator, 2026-10-15). A slot's lifecycle is already modelled as states, and startup brings many slots up in the
background, so a single slot's action needn't block the UI.

## Spec

- **Lifecycle actions run as background jobs per slot.** Picking an action starts the job and returns to the switcher
  at once; the slot's row shows its progress (Starting… / Attaching… / Relaunching… / failed: reason).
- **The operator can switch to any other slot or the hub while jobs run;** switching to a busy slot shows its progress,
  then attaches when it's ready.
- **One job per slot at a time,** guarded by the slot's existing lease: a second action on a busy slot is refused with
  "busy: <job>", never queued silently.
- **Failures land on the row** (and, once pair#574 lands, as a hub lifecycle notice), never as a blocking modal.
- The CLI verbs (`couch --relaunch/--resume/--reboot`) keep their synchronous semantics; only the UI becomes async.

## Done when

- A UI test: start a relaunch on one slot, immediately switch to another and to the hub; both respond while the job
  runs, and the first row shows progress then its result.
- A second action on the busy slot is refused with the busy reason.
- A failing job shows its reason on the row; the switcher stays usable.
- Schedule after pair#556 lands (it reworks the same recovery paths).

## Plan

- [ ]

## Log

### 2026-10-15
