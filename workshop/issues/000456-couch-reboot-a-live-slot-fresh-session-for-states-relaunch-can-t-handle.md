---
id: 000456
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '72a7cdcd554a15e10e82b2762d09ad4a30c50a3a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T20:52:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "249f658f", done: "cc4aaf89"}
---

# couch: reboot a live slot (fresh session), for states relaunch can't handle

## Problem

`couch --reboot` refuses a live slot (`not-offered: … is live`), and relaunch (Alt+n, `couch --relaunch`) needs a conversation to resume. Some live slots fit neither: a fresh session that never completed a turn (`resume-binding-unbound`), or a Codex slot on an old wrapper that stays `resume-binding-provisional` even after a completed turn. On 10-11 parley.nvim:0 and parley.nvim:3 hit this during the fleet relaunch; the TL had to send a throwaway turn to one, and the operator relaunched the other by hand. The operator needs a way to restart a live slot from scratch.

## Spec

- **Reboot a live slot:** `couch --reboot repo:N --confirm` also accepts a live slot: stop its agent and wrapper, then start a fresh session on the current binary in the same pane and worktree. The conversation is not resumed (that's relaunch's job); the worktree and branch are untouched.
- **Guards**, the same vocabulary as relaunch: refuse `busy` (a turn running), and `busy-unknown` unless `--force-unknown`. A dirty tree does not block (reboot changes no files), but the result reports it.
- **In Couch:** a reboot action next to Alt+n for the thread shown or highlighted, with a confirmation, since it discards the conversation. The key or menu entry is the operator's choice; propose one that doesn't collide (check keyhelp).
- **Relaunch's refusals point at it:** `resume-binding-unbound` and `resume-binding-provisional` name reboot as the next step.
- Atlas and `couch --help` updated.

## Done when

- A live slot with a never-finished fresh session (Claude and Codex) reboots by `couch --reboot … --confirm` and from Couch; the new session runs on the current binary.
- Busy slots are refused; `--force-unknown` overrides busy-unknown.
- Tests cover the guard decisions; a live check reboots a real slot.
- The operator smoke-tested the Couch action before close.

## Plan

Design: a live reboot is park-then-reboot under the reboot hold. `rebootAdmit`
admits `live` only with an idle verdict (`require-settled`, `RebootAdmitted`)
and has no effect; the fresh-profile preflight resolves; then `rebootStopLive`
re-checks busy (`LiveRestart.ConfirmNotBusy`), parks via `ParkExpected` (park
needs no conversation binding) and reclassifies; then today's parked reboot
(`prepareRetirement`, start) runs (revised at close review, see Revisions). Remote admission reuses `prepareLiveRestart`/`DecideLiveRestart`
(`OpReboot`: dirty only noted). The switcher sends `forced` for a live row.
Calls without a verdict (recover's reboot step) still refuse live rows.

- [x] Reboot of a live slot in couchcore/dispatcher, with guards
- [x] Couch action and confirmation (operator chose `Tab → reboot` on live rows; no new chord)
- [x] Relaunch refusals name reboot (`withRebootHint`)
- [x] Tests, atlas, help, README, skill
- [x] Live check: reboot a real slot; operator smoke test of Tab → reboot

## Log

### 2026-10-11
- Operator chose `Tab → reboot` on live rows (no new key) over Alt+Shift+N (Pair's reload-context in an actor pane) or Ctrl+Alt+r.
- `TestActionOfferedImpliesPermitted` caught the live offer against `RebootableState`; the fix was a named whole-admission predicate (`RebootAdmitted`), not widening archive's rule (ARCH-DRY: archive must still refuse live).
- Sandbox blocks pty/socket tests; ran couchtty/couchcore/couchcmd/couchmessage unsandboxed: all ok.

### 2026-10-11
- Moved to pair:0 at 10c02586 (main merged), built couch/pair (vcs.revision verified). Operator smoke: Tab → reboot on pair:6 (live, never finished a turn) worked. Remote: `couch --reboot pair:6 --confirm` → "pair:6 reboot succeeded (tag 1-pair-42); archived 1-pair-36".
- Sandbox: `couch --reboot` from the agent shell failed `dial unix …sock: connect: operation not permitted`; unsandboxed it worked (evidence for ariadne#334).
- `go test ./...` (unsandboxed, PAIR_* retention env scrubbed): all ok after rewording my help line, which tripped `TestPublicHelpListsOnlyPublicSurface` with the word "start".
- Close review round 1 (FIX-THEN-SHIP): BR-1 live reboot parked inside admission, before the fresh-profile preflight (ARCH-ORDER). Split into effect-free `rebootAdmit` + `rebootStopLive` run after the preflight, just before `prepareRetirement`, on both kinds; `TestLiveRebootProfileFailureStopsNothing` (mutation-checked by moving the stop back: fails). BR-2: `TestRebootLiveSlotByPath` covers the slot path, with and without the verdict (`withParkLifecycle` extracted from `envWithLiveThread`). BR-3: comment on `liveRebootArgs` naming the offer table as recover's guard. BR-4 (Codex): the core path is agent-agnostic (park + fresh profile); the live Codex case was not exercised.
- Close review round 2: BR-5, the atlas and this plan's Design line still stated the pre-fix order. Swept every prose site stating it (atlas paragraph + predicate table, `Reboot`'s ORDER comment, `RebootAdmitted`'s comment, Design line). BR-4: added `TestWithRebootHint` covering the provisional branch too.
- `TestOrientationSessionObservationHonorsCallerDeadline` failed once under full-package load, passed 5/5 alone and on the rerun; untouched by this diff.

## Revisions

### 2026-10-11 — close review
- Live reboot runs the fresh-profile preflight before parking; the park and reclassification sit between the preflight and `prepareRetirement`, so reboot's ORDER contract (never stop then start nothing) holds.

