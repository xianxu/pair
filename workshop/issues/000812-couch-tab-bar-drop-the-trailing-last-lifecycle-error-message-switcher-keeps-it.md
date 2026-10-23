---
id: 000812
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours: 1.7435
collaboration-mode: collaboration
card_mirror: '6c14ed88312e6a1c92b942446e47c758decfca3e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T05:01:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it)

## Problem

## Spec

Operator request (10-22, screenshot in ops `assets/session/2026-10-22.13-05-53.800.jpg`): the Couch tab bar's second
row shows the last lifecycle or error message after the slot tabs (e.g. `· pair [couch-dd48472d] exited (1)`). The
operator doesn't find it useful there and wants it removed from the tab bar. It stays in the switcher. Anything that
needs the operator's attention goes through the notification (attention list / `couch --notify`), not the tab bar.

- Remove the trailing last-message text from the tab bar (both one-row and wrapped two-row layouts); the space goes
  back to slot tabs.
- The switcher keeps showing it, unchanged.
- If any path relied on the tab-bar text as its only operator-visible signal, route it to the attention list instead
  and say which.

## Done when

- A tab-bar render test with a pending lifecycle/error message shows no trailing message text; the switcher test
  still shows it.
- README/atlas wording updated.
- Actionable asynchronous failures from the producer audit appear in existing attention; routine progress/success/key feedback does not create obligations. Existing prompt/question/hub-notify/backup attention is not duplicated.
- Global or unavailable-thread attention events open the switcher rather than redirecting to an unrelated focused slot; exact attached-thread selection remains exact. TL approved this navigation detail (receipt db842971-53f5-4660-a836-870b0d14de1a).

## Plan

[Durable implementation plan](../plans/000812-tab-bar-notice-plan.md). TL-approved full-flow plan; enter change-code before implementation.

- [ ] Remove general feed notice from both tab-bar styles; retain latest full notice for switcher display and Ctrl+Y with menu precedence/geometry tests.
- [ ] Route the audited asynchronous actionable notice classes through existing attention; keep progress/success/key feedback feed-only, and qualify exact/global selection and bounded lifetime.
- [ ] Update README/atlas, run targeted/race/inventory and recorded checks, obtain coordinated live acceptance, then close/review/land.


## Log

### 2026-10-22


### 2026-10-23 — claimed and planned from main

Authenticated ops:0 dispatcha5d6fa63-e13d-47f0-a2c9-03f5643533b7 assigns #812 after #791 landed. Claimed pair:1 (claim waited for a peer close's legitimate repository lock); start-plan created this independent branch from mainc0dbb32decc052500172ded01ff36e3fa8e49cc9. Catch-up refreshed the local ariadne dependency toee4dd7fa1e8488e3230dc462971be8796f317e41; seeded workflow refresh was explicitly deferred by the new tooling, no local-gates edit to ship.

Read-only producer audit delegated to812_notice_audit, no code changes. The current feed's latest text reaches StatusModel.Notice through statusModelLocked and both layouts; CapabilityNotice is a separate live control and stays. Contrary to the initial “switcher keeps it” assumption, the general feed is currently only available through Ctrl+Y; renderedMenuNotice draws MenuState.Notice instead. The plan explicitly adds a bounded latest-feed switcher projection without replacing operation errors/progress/copy feedback. ARCH-DRY keeps Feed as history owner; ARCH-PURPOSE covers asynchronous warnings before deleting their tab visibility.

Audit identified unexpected exits, crash/trace failures, hub/cleanup/stale/supervision warnings, continuation failures and async cast/guessing-lockout failures without existing attention. Blanket routing all setNotice calls would promote progress and harmless feedback to false obligations. Recommended explicit producer classification reuses AttentionNotify's cap50/select/dismiss/12-hour expiry. Subjectless events are global Couch items; proposed selection opens switcher, never attributes them to the focused actor. The durable plan records this visible-state decision for TL approval. Existing notify/prompt/question/archive routing remains single-source. No new core enum, persisted format, worker, queue or native-input effect proposed. No estimate or change-code run yet.

Next: send committed plan to TL and await approval, especially the global-selection choice; then run change-code and its plan-quality gate before deriving any estimate or implementing. No live stage or operator-slot action authorized.


### 2026-10-23 — plan approved; enter implementation gate

Authenticated TLdb842971-53f5-4660-a836-870b0d14de1a approves renderer/collapse/wrap removal, latest-feed switcher display with operation/copy precedence, explicit actionable async attention routing, and global/detached-target selection opening the switcher. Extra noise guard: routinely firing normal-operation classes do not belong in attention. Durable plan updated with that constraint; no production changes yet. Next run change-code full flow, answer plan-quality findings and derive estimate only after it clears. Use executing-plans in this warm session and delegate isolated test/producer tasks where context is capturable. Stage/live acceptance still requires a later TL go.



- Plan-quality round1 REWORK (PQ-1, function-level-test-strategy): codebase claims and approved architecture verified, but tests were written as case inventories and the new projection/publication functions were unnamed. Swept the test plan into named risky-function strategies with fuzz/generated and controlled mutex/channel interleaving proofs; no behavior or scope change. No source/test implementation yet. Re-run change-code before deriving estimate.



- Plan-quality round2 CLEAN; PQ-1 addressed, no further findings. Estimate derived only after this result, from v3.1/v2.1/v2 primitive tables and baseline-v3.1 (calibration marked stale/provisional). Existing Feed/AttentionList/renderers cover the work; no greenfield library or external API needed. Small module projection: design0.15×0.2=0.03/impl0.50×0.4=0.20; TUI attention and navigation: design1.0×0.2=0.20/impl1.0×0.4=0.40; cross-cutting producer routing: design0.8×0.2=0.16/impl0.5×0.4=0.20; docs: design0.1×0.2=0.02/impl0.15×0.4=0.06; review/verification: design0.1×0.2=0.02/impl0.5×0.4=0.20; coordinated UX iteration: design0.3×0.2=0.06/impl0.3×0.4=0.12. Design0.49×1.15 + implementation1.18×familiarity1.0 =1.7435h. Exact collapse function anchor corrected before gate rerun.


## Revisions

### 2026-10-23 — executable preservation and attention contract proposed

The operator's removal decision remains unchanged. Discovery shows the switcher does not actually draw the general Feed, so preserving that surface requires explicit projection rather than merely leaving MenuState.Notice alone. Add Done-when criteria for the already-required actionable-signal audit, and propose global/unavailable-target attention selection opening the switcher. The durable plan enumerates affected producer classes and keeps this navigation detail pending TL approval; no implementation scope is silently accepted.

### 2026-10-23 — TL approval

The preceding pending navigation choice is approved, with a producer-level noise exclusion for routine normal operation. No new scope beyond the recorded plan.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration source currently stale, values provisional. Spec resolves decisions (design ×0.2), implementation ×0.4 per v3.1, familiarity1.0; thorough-plan design buffer15%. No external API discovery or cross-repo propagation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: cross-cutting-refactor design=0.16 impl=0.20
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
item: ux-rename-iteration design=0.06 impl=0.12
design-buffer: 0.15
total: 1.7435
```
