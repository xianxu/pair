---
id: 000691
status: codecomplete
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '488dd4b9975d62bd950dc32d017277c2024c41e6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T10:56:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 2.79
---

# couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage)

## Problem

The hub reads Claude quota by running /usage in an idle Claude slot with `couch --keys SLOT clear-line text:/usage submit` and reading it with `couch --peek`. The /usage panel then stays open ("Esc to cancel"), and `couch --keys SLOT esc` refuses ("wait until the agent composer is empty and its turn finishes"), so only the operator can close it. On 2026-10-18 two slots (brain:1 since 12:38, parley.nvim:4) sat with open panels, raising Couch's 10-minute waiting notice and needing the operator's Esc.

## Spec

Let `couch --keys` send Esc (only Esc) to a slot whose agent shows a dismissable modal (a slash-command panel recognized by the composer recognizers) even though the composer is not empty, so the sender can close a panel it opened. Keep refusing every other key in that state. Alternatively, or as well, a read-only `couch --usage` that runs /usage, captures the panel, and dismisses it in one operation.

## Done when

- After `--keys SLOT clear-line text:/usage submit`, `--keys SLOT esc` closes the panel and the slot returns to an idle composer (recognizer fixture plus a live check).
- Other keys still refuse while a panel is open.

## Plan

### Current behavior (2026-10-19, pair:2)

`couch --keys` admission (`couchcmd/agent_drive_service.go`) accepts a target only when its wrapper reports `Idle`, or, for a recovery-only request (keys are only Esc or clear-line), `Recoverable`. The wrapper computes both from the viewport (`wrapcmd/agent_drive.go agentDriveScreen`). `Recoverable` needs a recognized composer (`qualifiedComposerText`). An open `/usage` panel replaces the composer, so the screen is neither idle nor recoverable and Esc refuses. The reducer's `EventStart` (`agentdrive/model.go`) applies the same rule again. No recognizer for Claude slash-command panels exists today; Claude's picker detection is its own picker-open OSC, which a panel does not emit.

### Design

- [x] Evidence first: the hub's Claude 2.1.304 capture, `wrapcmd/testdata/drive/claude/2.1.304/` (see Revisions). One authorized capture of a Claude `/usage` panel, as `couch --peek` text (styled tail) plus the Claude version, becomes `wrapcmd/testdata/drive/claude/<version>/usage-panel.screen` with metadata, next to the existing drive fixtures. Also a capture of the idle composer after Esc. Requested from the hub; nothing touches a live slot from this slot.
- [x] Pure recognizer `agentdrive.DismissablePanel(agent, lines)`: true only for a Claude panel whose current viewport ends in the panel's own Esc footer, with no recognized composer and no numbered picker (`couchmessage.ReadPicker` empty). Exact footer wording comes from the capture, never from scrollback, as the drive screen already reads only the viewport.
- [x] Screen gains `Dismissable`, set in `agentDriveScreen` from the recognizer, with the same turn, picker and completeness gates as `Recoverable`, and cleared in `agentDriveScreenLocked` under the same input-ownership conditions. `DriveObservation` carries it.
- [x] Request predicate `DismissOnly()`: exactly the keys `[esc]`. Admission accepts `Idle`, or `RecoveryOnly && Recoverable`, or `DismissOnly && Dismissable`; the guard re-checks the same rule. The reducer's `EventStart` uses one shared admissibility function, so admission and the reducer cannot disagree (ARCH-DRY). Every other key sequence, including clear-line or text, still refuses while a panel is open.
- [x] Outcome: after the Esc is written, the existing final observation reads the screen. The drive reports done when the composer is back (`Idle`); a panel still showing reports unconfirmed, with the tail, and nothing retries.
- [x] Tests:
  - Recognizer over the captured fixture, positive.
  - Recognizer negatives: the composer, the /model picker, a permission prompt, a panel footer only in scrollback.
  - Admission matrix: Esc admits on a panel; clear-line, text, submit and esc+text refuse. Run through the existing drive service fake, not a live Couch.
  - Reducer: Esc then composer returns gives done; panel persists gives unconfirmed.
- [ ] Live check after landing: the hub runs `--keys SLOT clear-line text:/usage submit`, then `--keys SLOT esc`, on an idle Claude slot, and confirms an idle composer.

State design (TL rule):

Actions: `couch --keys SLOT esc`, operator Esc, `couch --peek`.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Panel open, recognized | panel shown; `--keys esc` admitted | `--keys esc` | none |
| Panel open, not recognized (new wording) | refused "wait until the agent composer is empty" (today's behavior) | operator Esc; file a recognizer update | none |
| Esc written, composer returns | idle composer, drive done | none | none |
| Esc written, panel persists or outcome unknown | drive unconfirmed with tail at the drive deadline | `couch --peek`, then operator | none (Esc is idempotent on a closed panel) |
| Turn starts or picker appears mid-drive | drive stops, as today | `couch --peek` | none |

Not covered: other agents' panels (Codex and others), and a `couch --usage` one-shot. Both are separable follow-ups; the Spec offers the latter as an alternative.

ARCH-SECURE: the recognizer reads only the current viewport, so scrollback text cannot authorize a key, as with pickers. ARCH-ORDER: no new persistent state; one more admissible screen class through the existing reducer. ARCH-MOCK: tests use the drive fixtures and the service fake; the live check is the conformance step.

### Revised plan (2026-10-19, pair:2): third delta, so a whole plan

The live check failed (parley.nvim:4, Claude 2.1.304, 94x39). Two measured blockers:

- (a) The panel is taller than the screen. The 39-row viewport has no full-width rule, and the recognizer required one.
- (c) The wrapper counts a turn as active for as long as the panel shows. After the `/usage` submit, Claude sends progress "working" (OSC 9;4;3, offset 29768 of `scrollback-1-parley-nvim-62-claude.raw`) and never "stopped" (9;4;0) while the panel is open. `ObservationWorking` opens a turn with activity seen, so `TurnActive` holds until the 60 s watchdog fires. The panel proof required no active turn, and the drive reducer stops on any active turn. At dismissal, Claude does send 9;4;0 right before its redraw (ariadne:5 recording, about offset 270616). The 250 ms grace then ends the turn.

Simplest alternative, and why it fails: treat a submitted line starting with `/` as a local command that never opens a turn, and recognize the panel by its footer on the last row with no rule. The footer half is kept (step 1). The prefix half does not clear the turn: Claude's own progress signal opens it whatever the submission was. It would also drop turn tracking for prompt-backed slash commands (/compact, skills), which run real turns.

What stays (approved in the original design): exactly `[esc]`, one shared admission rule, done only when the composer returns, unconfirmed at the deadline, no retry.

- [x] 1. Recognizer: the footer `Esc to cancel` is the last nonblank viewport row. The scope is the rows below the last full-width rule if one is on screen, otherwise the whole viewport. A panel taller than the screen fills it. Within that scope there is no `❯` prompt row and no numbered choice. Both captures are pinned as fixtures: the short one (2.1.304, 39 rows) and the tall live-check one (its last 39 rows).
- [x] 2. The panel proof outranks the turn flag, for the lone esc only. The turn flag is an inference from progress OSC. The screen proves the composer is gone and a panel holds the keys. `Dismissable` drops its `!TurnActive` gate and keeps the picker-OSC, complete-frame and no-composer gates. A dismissal drive does not stop on `TurnActive` (the reducer's top-level stop, and the wrapper's turn-active dispatch branch). It still confirms only on `Idle`, which arrives once Claude clears progress and the grace ends. Every other request still stops on an active turn. `Recoverable` keeps its turn gate.
- [x] 3. #716 close advisory: the skill's "five seconds total" is asserted against `MaxKeyTotalWaitMS`.
- [x] 4. Tests:
  - Recognizer: positive on both captures. Negatives: composer, closed panel, picker, permission prompt, panel footer followed by other text.
  - Wrapper: paint the tall capture with `turnActive` true. Expect `Dismissable`, and `[esc]` admitted, written once, then the composer and the turn end, then confirmed.
  - Reducer: a dismissal started with `TurnActive`; turn-active frames after the write keep observing; `Idle` confirms. A non-dismissal request still stops on `TurnActive`. Esc on an occupied composer with an active turn still refuses.
- [ ] 5. Live check (hub, after landing): `--keys SLOT clear-line text:/usage submit`, then `--keys SLOT esc`, on a short terminal (the panel taller than the screen) and on a tall one. Expect done and an idle composer.

State design:

Actions: `couch --keys SLOT esc`, operator Esc, `couch --peek`.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Panel open, turn flag active from the panel's own progress signal | panel; `--keys esc` admitted | `--keys esc` | none |
| Panel taller than the screen | panel; `--keys esc` admitted | `--keys esc` | none |
| Panel open over a genuinely working agent turn (if Claude allows /usage mid-turn) | panel; `--keys esc` admitted | `--keys esc`, then peek | unverified: Claude's dialog should consume the Esc; if it reached the turn, the turn would be interrupted |
| Esc written, composer returns, progress clears | idle composer, done | none | none |
| Esc written, composer returns, progress never clears | unconfirmed at the deadline, with tail | peek | none |
| Panel not recognized (new wording) | refused | operator Esc; recognizer update | none |

Not covered: other agents' panels; a `couch --usage` one-shot. Question for the hub: should the live check also try /usage opened during a working turn, to settle the unverified row?

## Revisions

- 2026-10-19 (pair:2): reopened after the failed live check. This is the third design delta on this mechanism, so a whole revised plan sits above under Plan (Revised plan), instead of another delta. Pending hub approval.
- 2026-10-19 (pair:2): the hub approved the design and delayed the live capture (the machine crashed; opening /usage live leaves a panel only a human can close until this lands). Delta: build against a provisional fixture, `wrapcmd/testdata/drive/claude/provisional/usage-panel.screen`, shaped from the `Esc to cancel` footer Claude paints on its own dialogs; its metadata and the recognizer's footer table are marked for replacement by the capture.
- 2026-10-19 (pair:2): close review BR-1. Delta: the Done-when live check is a post-merge step owned by the hub. Fleet rule: workers never drive a live slot, and the hub verifies after landing. This close is gated on the fixture half (the hub's live 2.1.304 capture through recognizer, wrapper and service tests). The hub records the live run (`--keys SLOT clear-line text:/usage submit`, then `--keys SLOT esc`, idle composer) in this Log after merge, and a failure reopens the issue.
- 2026-10-19 (pair:2): capture in. Delta: the fixture moved to `claude/2.1.304/usage-panel.screen` plus `usage-closed.screen`, the provisional directory was deleted, and the recognizer reads only the rows below the panel rule.
- 2026-10-19 (pair:2): outcome timing. A panel frame after the Esc no longer ends the drive at once: a usage panel repaints when its data loads, and such a frame can be drawn before the agent reads the Esc. Delta: only an empty composer confirms; a panel still showing at the drive deadline reports unconfirmed with its tail. No retry either way.

## Log


- 2026-10-20: closed — Full suite green on a5c7d1cc (sdlc-local/suite). a5c7d1cc fixes the close advisory: the recognizer safety variant now places the footer words on the row above the composer box, asserting the prompt row follows the opening rule; agentdrive tests green. --no-plan-check: the open item is the hub post-merge live check. (fixed-then-ship); review verdict: SHIP
- 2026-10-19 (pair:2): the hub approved the revised plan, with conditions: pin the safety direction as hard as the happy path using real working-turn captures, and do the mid-turn live case only in a throwaway setup. Implemented. New fixtures in claude/2.1.304: usage-panel-tall (the hub live-check capture, last 39 rows), and working-thinking, working-after-usage and working-shell, replayed at 94x39 from the ariadne:5 wrapper recording during working turns. Safety tests: each working screen as captured, with the footer words in its history, and with them just above the composer, refuses esc at the recognizer and through the wrapper with the turn flag set, and writes nothing. A dismissal admitted on a panel writes nothing if a working turn replaces it. Mutations, each red: rule required again (tall panel), turn gate back on Dismissable, reducer stopping the dismissal on the turn, the exemption widened to any recovery drive (safety), footer words anywhere plus no composer gate (safety), and the writer forcing the panel proof. The #716 advisory is done: the skill says 5000 ms total, asserted against MaxKeyTotalWaitMS.

- 2026-10-19: reopened — the hub live check failed (parley.nvim:4, Claude 2.1.304, 94x39; /Volumes/AgentScratch/pair691-usage-capture/livecheck-2026-10-19-fail.md). Esc was refused on an open /usage panel for two measured reasons: the panel is taller than the screen, so no rule is in the viewport, and Claude progress stays working while the panel is open, so the wrapper turn is active. A revised plan follows.

### 2026-10-19
- 2026-10-19: closed — Full suite green on 0127de1a (sdlc-local/suite). Recognizer, wrapper and service tests over the hub captures: short and taller-than-screen /usage panels admit exactly esc, with and without the turn flag, and confirm only after the composer returns and the flag clears. Safety: three real Claude 2.1.304 working-turn screens (as captured, footer words in history, footer words above the composer) refuse esc and write nothing; a dismissal whose panel gives way to a working turn writes nothing. Six mutations red, two on the safety side. --no-plan-check: the open item is the hub post-merge live check (short and tall terminals; mid-turn case only in a throwaway slot).; review verdict: SHIP
- 2026-10-19: closed — Full suite green on 28bd9ba86 (sdlc-local/suite); 4675381be adds only a clarity edit and a comment, targeted agentdrive and wrapcmd drive tests green. Recognizer, wrapper and service tests run over the hub live Claude 2.1.304 /usage capture at 94x39; six mutations each turn red. --no-plan-check: the open Plan item is the hub post-merge live check (Revision BR-1). (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-19: flow upgraded quick → full — 133 added lines in code files (limit 100); an earlier round of this close already ran the full review

- The hub's capture arrived (Claude 2.1.304, ariadne:5, 94x39). Footer `Esc to cancel`. It replaced the provisional fixture and exposed a recognizer bug: at 39 rows the earlier `❯ /usage` prompt row is visible above the panel, and the first recognizer rejected any `❯` row. Fix: the panel is the rows below the last full-width rule. Mutations: whole-viewport scan (red on the capture, agentdrive and wrapcmd) and no rule required (red on the no-rule negative).
- The hub reported that the installed couch had admitted a bare `esc` while the panel showed. Replaying the wrapper recording `scrollback-1-ariadne-46-claude.raw` (94x39, build a040b13ba, drive code identical to main) disproved it. On the panel frame the composer is unknown, so the screen is neither idle nor recoverable. The minute-debounced time events (16:57:33, then 16:58:57) put the dismissal redraw before 16:58:33. The only output at 16:58:57 is a small later frame, Claude's response to Esc on an empty composer, which is what the hub's `--keys esc` confirmed. The hub agreed (the operator likely pressed Esc). Scope unchanged.

- Implemented: `agentdrive.Request.Admits` is the one admission rule (service, guard, wrapper start, wrapper pre-write, reducer). `DismissOnly` is exactly `[esc]`. `agentdrive.DismissablePanel` reads the viewport only. `Screen.Dismissable` and `DriveObservation.Dismissable` carry the proof, cleared under the same input-ownership gates as `Recoverable`. Atlas couch.md, README and the couch skill updated.
- Mutation checks, each red: Admits without the panel term (agentdrive, wrapcmd, couchcmd), wrapper never dismissable (wrapcmd), reducer confirming a panel repaint (agentdrive, wrapcmd), recognizer ignoring numbered choices (agentdrive).

### 2026-10-18
