---
id: 000791
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '88c31a9990e0871bc3f93fdf2342b1d9fe6dcc7a' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch admit: idle Claude slot refused as busy (auto-mode footer / update banner?); name which busy condition fired

## Problem

## Spec

From ops-echo:0 (10-21, 18:5x–20:1x): `couch --prepare echo:3 --confirm` refused three times with
`admit refused (busy) echo:3: the agent has a turn open, the composer holds text, or input was seen in the last few
seconds`, while `couch --peek echo:3` showed Claude idle at an empty `❯` prompt and `couch --actors` showed it
available. The peek footer read `⏵⏵ auto mode on (shift+tab to cycle)   ✔ Update installed · Restart to update`.

Hypothesis (unverified): the admission check doesn't recognise Claude's idle composer under the **auto mode**
footer (other Claude slots show `bypass permissions on`), or the update banner, so it falls back to "busy". Find
which of the three busy conditions fired; the refusal should name the one that did.

## Done when

- A captured screen fixture of an idle Claude composer with the auto-mode footer and the update banner is admitted as
  idle (test); the same with text in the composer is refused (test).
- The busy refusal names which condition fired (turn open / composer text / recent input).

## Plan

- [ ]

## Log
- 10-21 22:0x (ops:0): **a Codex case, possibly the same class.** parley.nvim:3 (Codex, tag 1-parley-nvim-83) idle at an
  empty `› Ask Codex to do anything` composer, but every delivery since the 21:33 Couch restart sits at "waiting for
  recognized composer" (ops:0's 9d16fde1 at position 2, behind the restart notice). Its screen differs from Codex
  slots that receive normally: a **blank line between the composer and the footer**, and the footer reads
  `⚠ 1 warning · f2 to view` (others show `⚠ 2 warnings`). Capture that screen as a second fixture.

- 10-21 ~20:40 (ops-echo:0): **cause pinned: the `✔ Update installed · Restart to update` banner** in Claude's footer
  blocks composer recognition. After the operator restarted echo:3's Claude (banner gone, auto mode still on), a
  resend delivered. While the banner showed, sends also sat at "waiting for recognized composer", `couch --relaunch`
  refused busy, and `couch --reboot` failed at recovery confirmation: "slot zellij session 1-393 kill not confirmed:
  ownership unresolved, unknown pane owner, pane has no unambiguous actual command" (operation
  e630974ef114e3fe95e51db5a22971b0, launch-diagnostics/failures.json); that reboot failure may be its own issue.
  Fixture: footer `⏵⏵ auto mode on (shift+tab to cycle)   ✔ Update installed · Restart to update`.

### 2026-10-21

### 2026-10-22 — recurring hub stalls; investigation dispatch

- Operator report via ops:0: hub deliveries have repeatedly stalled until an operator keystroke, then flushed in a batch, both before and after #735 landed. The hub is where the operator watches; worker slots may have the same symptom, but that is not yet observed evidence. This is not a newly introduced hub-only failure.
- Latest observed ops:0 footer: `⏵⏵ bypass permissions on (shift+tab to cycle)       ✔ Update installed · Restart to update`. Read-only live capture at investigation start confirms an empty `❯` prompt between horizontal rules and that combined footer. Claude version in the authenticated dispatch identity: 2.1.304.
- The update banner is confirmed for this observation and reported for echo:3; its presence on every earlier hub stall is unknown. Treat the earlier “cause pinned” entry as evidence of recovery after restart, not proof that the banner explains all stalls. Reproduce both Claude footer modes with the banner, compare banner-free controls, and investigate a possible second trigger including the Codex blank-line/warning-footer case.
- TL dispatch and correction authenticated as submitted to this slot: 6af1c55a-9494-4f6e-8c48-b43e5a8c4c2a and c13682ef-f911-427f-93cc-cd9fe557d319. Plan goes to TL before production changes. #737 remains parked on its pushed issue branch awaiting a separate explicit stage go.
