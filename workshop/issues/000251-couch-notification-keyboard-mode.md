---
id: 000251
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
started: 2026-09-14T08:57:57-07:00
---

# Keep Ctrl+Return notification jumps working across Couch thread switches

## Problem

On 2026-09-14 the operator reported that Ctrl+Return stopped jumping to the
newest pending notification: another thread remained yellow in the status bar,
no Couch notice appeared, and Return was inserted into the agent. The recent
thread shortcut still worked. The operator confirmed Ctrl+Space then Return
successfully reached the notification. Thus notification targeting works; the
shortcut is failing before its recognized handler.

`newestPageSequence` accepts `ESC[13;5u` only. The existing tests feed those bytes
directly and pass, so they cannot detect loss of terminal keyboard mode.
Couch currently depends on Zellij's output to enable Kitty disambiguation.
`switchTo` replays a bounded raw output tail; it can omit an aged-out enable or
replay an unmatched pop. `ptychild.Screen` tracks no keyboard flags, and
`hostty.RepaintFor` does not restore them. Child resets and primary/alternate
buffer changes can also remove the mode. The symptom is consistent with the
terminal producing legacy CR; the historical triggering bytes were not captured.

## Spec

Couch must establish and maintain the keyboard disambiguation required by its
own shortcuts while it owns a supporting terminal. Add the Kitty disambiguation
bit using `CSI = 1 ; 2 u`: additive setting, preserving other flags, with no
push/pop stack growth. Couch hosts Zellij, which already accepts this encoding;
do not turn this into a general legacy-child input translator.

Apply the same Couch-owned requirement on initial ownership, after each screen
takeover (actor or panel), and after complete active-child output. Keep the
control adjacent to the completed output in one host write where practical.
Reuse the existing incremental framing scanner and its `MidSequence` boundary,
including overlong-string skip state; never splice the control into a partial
CSI/OSC/DCS/APC. Do not use the cursor-save paint gate for a cursor-neutral
keyboard control. Inactive child output cannot change the host keyboard mode.
Teardown retains the existing shell-safe reset and emits no later re-enable.

Keep the existing notification handler and plain Return behavior. Do not
reinterpret CR as Ctrl+Return. Unsupported terminals retain the documented
Ctrl+Space then Return fallback. This establishes a shortcut requirement, not
full virtualization of each child's historical keyboard stack; it must not
clear other feature flags or introduce repeated pushes (ARCH-DRY, ARCH-ORDER).

Use a stateful terminal double behind the existing Host seam. It must consume
actual emitted bytes, track Kitty flags and primary/alternate stacks, and
encode a physical Ctrl+Return from those flags. Drive that resulting input
through the running Console with a pending notification. Cover the transitions
that remove the bit rather than only matching a new output constant
(ARCH-MOCK, ARCH-PURPOSE).

## Done when

- A stateful regression fails on current code because physical Ctrl+Return
  becomes CR, then passes with the notification jump and acknowledgement.
- Startup, actor/panel takeovers, aged-out enable sequences, live reset/set/pop,
  primary/alternate transitions and split output retain the shortcut requirement.
- Tests preserve ordinary Return, other requested keyboard flags, background
  isolation, framing integrity, bounded stack depth and shell-safe teardown.
- Focused tests, affected-package tests and a real supporting-terminal smoke
  pass; the operator verifies the rebuilt Couch shortcut with a yellow thread.
- Atlas documents Couch's keyboard requirement and legacy-terminal fallback.

## Plan

- [ ] Follow `workshop/plans/000251-couch-notification-keyboard-mode-plan.md`: reproduce using a stateful terminal double.
- [ ] Implement Couch-owned disambiguation through existing output/framing boundaries.
- [ ] Verify regressions, document the behavior and validate the installed runtime with the operator.
- [ ] Close through SDLC review and publish.

## Log

### 2026-09-14

Created and claimed at operator request. Read-only investigation found no
keyboard state restoration in Couch's takeover path. Existing tests passed:
`go test ./cmd/internal/couchtty -run 'Test.*(NewestPage|Interceptor.*)' -count=1`.
The running Couch PID 5316 had a mouse trace but no input trace, so the exact
historical mode-loss event remains unknown. An independent read-only review
confirmed the additive requirement approach and the need to use framing rather
than cursor-save eligibility. Implementation has not started.

Protocol reference: https://sw.kovidgoyal.net/kitty/keyboard-protocol/
(progressive enhancement and separate primary/alternate keyboard stacks).

## Revisions

### 2026-09-14 — Preserve event-reporting compatibility

The independent investigation noted that preserving child event-reporting flags
also requires recognizing explicit Ctrl+Return press/repeat encodings. Add the
two exact forms through the existing interceptor table; release must not jump.
The durable plan includes this bounded compatibility case and stateful tests.
