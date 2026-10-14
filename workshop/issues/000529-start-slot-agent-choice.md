---
id: 000529
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '08b93a22f121af15e84ad91fa9729d9278629a4c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T18:57:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "35bda2ae", done: "ab9c3d5e"}
---

# Couch start-slot form has no selectable agents

## Problem

The operator cannot choose an agent in Couch's start/add-slot form. The production `NewMenuState` constructor leaves `Agents` nil, and no production caller initializes it. `openStartForm` and the Left/Right handlers consume that empty slice, so no choice is possible. A preview may display the resolved default agent, but that does not populate the choice list. Existing tests manually inject `state.Agents`, masking the production omission.

This is distinct from the reported Ariadne Claude-to-Codex switch failure. The switch-agent form already uses `launcher.AgentInventory()` directly. Do not assume fixing this start-form bug diagnoses that launch failure.

## Spec

Use the canonical launcher agent inventory for the start form as the switch-agent form already does (ARCH-DRY). Keep the existing launch resolver responsible for default selection and the existing sticky explicit selection semantics. Do not add a second agent list or change live slots during implementation tests.

## Done when

- A state created by the production constructor offers selectable agents in start and add-slot forms without tests injecting an inventory.
- Selecting Codex remains selected after asynchronous preview and is passed to the start operation; untouched selection still follows the normal launch default.
- Regression tests exercise the production initialization path, including Left/Right and preview completion.

## Plan

- [x] Reuse the canonical agent inventory in start-form initialization/navigation and add the constructor-based regression.
- [x] Verify both start and add-slot paths plus sticky selection across preview updates.

## Log

### 2026-10-13
- 2026-10-13: closed — Production-constructor regressions failed before the fix on both start and add-slot selection, then passed after canonical inventory initialization. Focused start/add-slot tests pass (0.275s), focused race checks pass (1.342s), entire couchtty package passes (34.695s), clean make build and diff checks pass. Tests cover both arrows, stale/current preview completion, explicit Codex launch arguments, untouched resolver defaults and inventory ownership. Full make -k test has been started but remains queued at the machine gate; no full-suite pass claim.; review verdict: SHIP

Read-only investigation requested by ops:0. Confirmed against main-era source: `menu.go` NewMenuState, openStartForm, reduceStartKey and selectStartAgent; production references to `Agents` only read or clone it. `menu_switchagent.go` uses launcher.AgentInventory directly. Existing menu tests set Agents themselves.

Independent temporary Go-overlay diagnostic passed (0.271s): production constructor with RootAgent=claude leaves the start agent empty; Right does nothing and AgentSticky remains false. Positive control supplying the canonical inventory makes the same selector choose Codex with AgentSticky=true. No repository implementation files or live slots were changed. Filed from a scratch checkout outside the numbered-slot layout; leave unclaimed.

### 2026-10-13 — implementation design

Initialize `MenuState.Agents` in `NewMenuState` with `launcher.AgentInventory()`, which already returns an owned copy. This keeps both start and add-slot forms on the canonical inventory without duplicating choices or changing resolver authority (ARCH-DRY). Use constructor-based tests for both entry points: untouched previews omit an explicit agent and accept the resolver default; Left/Right selection requests Codex, rejects stale preview completions, retains the current preview and dispatches Codex. Existing focused tests may still inject restricted inventories when testing unrelated behavior. One production initialization change; no live slot operations, new state lifecycle or API. Run RED/GREEN reducer tests, package tests and full `make -k test`, then close/CI/land.

- Constructor regressions RED on unchanged code: both explicit start/add-slot paths failed to select Codex, and the canonical-inventory assertion saw an empty slice (`/tmp/pair529-red.log`). One-line constructor initialization makes all new and existing start/add-slot focused tests pass (0.275 s, `/tmp/pair529-green.log`). Tests cover Left/Right, stale and current preview completion, visible selection, explicit commit arguments, untouched resolver defaults and independent inventory ownership. Full suite queued; focused race and complete package verification running.

- Verification: focused start/add-slot regression suite passed (0.275 s), focused race run passed (1.342 s), complete couchtty package passed (34.695 s) and clean `make build` passed. Full `make -k test` is queued in `/tmp/pair529-full-suite.log`; no full-suite success claim. No live slot was started or changed.
