---
id: 000533
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '078b4d33aa093d135ee2983952992bb901fb31ec' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T14:01:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# draft: show progress while a DOM submission prepares, and hold the draft read-only until it's sent or fails

## Problem

After Shift+Ctrl+Alt+Return the draft looks idle for seconds while Couch prepares note evidence. The operator cannot tell whether the chord worked or what text is being sent.

## Spec

Show immediate visible submission progress in the draft status line, then update from actual work: preparing replay/context, settling and capturing crop i/N, saving the bundle, sending to the pinned slot, and sent or the failure reason. Progress is informational, never proof of delivery.

The initiating draft is read-only only while this operation is active. Success releases the lock before applying today's successful-send clearing rules. Every failure, exception, missing response, or timeout releases it immediately and preserves the authored draft, with a visible reason. A new press after completion creates a fresh submission as in #524; duplicate delivery remains acceptable. While an operation is active, another press must not replace its ownership or extend its deadline. No persistent freeze, discard binding, or confirmation dialog.

Measure capture, animation settlement, bundle preparation/publication, and pane-send phases before implementation; record measured numbers and limits here. Identify the dominant measured phase and whether it can safely be accelerated.

## Done when

- [ ] Phase measurements and their reproducible commands are logged, with dominant-phase assessment.
- [ ] The production chord visibly renders progress before external work and reports actual per-crop and pinned-slot progress.
- [ ] The initiating buffer is read-only only until success/failure/timeout; all terminal paths restore editability and preserve failure text, and stale callbacks cannot clear or lock a successor.
- [ ] Pure model, controlled integration, and headless production chord tests cover progress, lock lifetime, timeout and recovery; affected checks and full suite are recorded.

## Plan

See `workshop/plans/000533-submission-progress-plan.md`. One atomic close boundary.

- [ ] Measure existing phases and finalize design.
- [ ] Carry preparation progress through existing status polling.
- [ ] Render progress and enforce operation-scoped editing/timeout ownership.
- [ ] Verify, document, review, and land.

## Log

### 2026-10-13

- Received operator requirements through ops:0, verified Couch receipt c9b8646e-a7f4-49b3-8397-7e45fbdaa509. #533 was already claimed to pair:6. Started planning after SDLC refreshed this slot and its ariadne dependency.
- Investigation: server preparation has a 30-second context; client preparation currently counts 360 polls with 100ms delays and up to 4 seconds per RPC, so this is not an absolute client timeout. Synchronous list-panes, append/commit and Zellij actions use unbounded vim.fn.system. The plan must bound those DOM-only operations as well as the UI timer (ARCH-CONSTRAINTS, ARCH-ORDER).

- Initial isolated measurements before implementation: calm two-crop capture 194.759ms (settlement 1.198/0.563ms; screenshots 51.050/42.383ms). With 800ms animation, capture 857.531–873.928ms and settlement 780.195–795.264ms; with 900ms animation, capture 952.945–1007.452ms and settlement 886.534–933.857ms. Durable Store.Prepare for a tiny PNG 39.216–78.110ms; supersede plus publish 63.498–75.225ms. Private real Zellij bracketed paste of 1.7KB, 100ms settlement and AltEnter 179.683–233.568ms. These fixture measurements exclude production Couch begin validations and do not claim to time the operator's exact page. Animation settlement dominates the animated crop samples (about 91–94%); shortening it would violate #524's natural-completion capture guarantee. Full command/sample record follows when measurement completes.
