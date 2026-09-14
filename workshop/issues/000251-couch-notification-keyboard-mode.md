---
id: 000251
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 1.74
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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration is marked stale by sdlc, so
these numbers are provisional. Derived after plan-quality accepted round 2.

Use familiar-Go multiplier 1.0, detailed-plan design discount 0.2 on implementation
primitives, and 15% design buffer. Raw v2 design/implementation selections:
issue/spec 0.5/0.1; new terminal double 0.5/0.6; keyboard policy extension
0.2/0.5; output serialization refactor 0.5/0.5; atlas 0.05/0.1; boundary
review 0/0.4. Scale implementation by 0.4 for v3.1. Existing Host and ANSI
libraries are reused; x/vt lacks Kitty key encoding, so no additional library
availability discount applies to the new protocol double. The issue/spec row
accounts for design authoring; other design rows use the settled-plan discount.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.04
item: greenfield-go-module design=0.1 impl=0.24
item: smaller-go-module design=0.04 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.01 impl=0.04
item: milestone-review design=0 impl=0.16
design-buffer: 0.15
total: 1.74
```

## Plan

- [x] Follow `workshop/plans/000251-couch-notification-keyboard-mode-plan.md`: reproduce using a stateful terminal double.
- [x] Implement Couch-owned disambiguation through existing output/framing boundaries.
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

### 2026-09-14 — Fresh review: terminal ownership and cleanup

Plan review found that operationQueue can also write takeovers, so the fix must
serialize all current terminal write sites with scanner decisions and stop
later writes after cleanup. The plan adds this prerequisite without claiming
to finish #224's broader typed-writer task. Cleanup must also clear keyboard
mode after returning from alternate to main screen. Add deterministic write/
takeover/cleanup interleaving tests and a both-buffer shutdown regression.


### 2026-09-14 — Plan review approved

Fresh-context spec/plan review approved the revised implementation plan after
addressing terminal-write serialization and both-buffer cleanup. The plan is
ready for operator approval; no production code or runtime sessions changed.


### 2026-09-14 — Operator approval; plan-quality refinement

Operator approved the #251 plan and independently authorized improving #207
mouse diagnostics. SDLC plan-quality's PQ-1 requested function-level test
strategies instead of case inventories; the plan now names pure helper,
terminal-double parser/encoder and Console/interceptor targets with generated
input and deterministic interleaving guards. No behavior/design scope changed.


### 2026-09-14 — Reproduced and implemented

The new stateful Host regression failed before production edits: physical
Ctrl+Return encoded CR and reached c1 while c2 was paging. It also reproduced
main-buffer keyboard leakage on release. Implemented additive disambiguation,
explicit press/repeat key forms, output/scanner serialization and final
main-buffer cleanup. Focused keyboard tests and affected packages pass.

Existing notification tests now permit the keyboard control at the complete
sequence boundary; a menu test now waits for its asynchronous visible banner
rather than assuming reducer completion means paint completion. Source confirms
EOF stops only pumpStdin, not Run; tests preserve this behavior and verify
cleanup after the eventual stop. Historical incident trigger remains unobserved;
the new regression proves the supported mode-loss mechanism.


### 2026-09-14 — Validation checkpoint

Affected packages pass (couchtty, hostty, ptychild), as does the focused race
suite. Pure policy fuzzing passed 103,830 executions in three seconds; the
independent terminal model passed 6,988 partition-fuzz executions. Full
`make test` first flagged the keyboard-only framing reads in the paint-gate
source guard; the documented non-paint exception now passes and the full suite
is rerunning. Live supporting-terminal verification remains outstanding.
