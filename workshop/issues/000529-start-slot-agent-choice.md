---
id: 000529
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '561aeb9dc9c3cc3866ea653f7e57b963b1f3d551' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Reuse the canonical agent inventory in start-form initialization/navigation and add the constructor-based regression.
- [ ] Verify both start and add-slot paths plus sticky selection across preview updates.

## Log

### 2026-10-13

Read-only investigation requested by ops:0. Confirmed against main-era source: `menu.go` NewMenuState, openStartForm, reduceStartKey and selectStartAgent; production references to `Agents` only read or clone it. `menu_switchagent.go` uses launcher.AgentInventory directly. Existing menu tests set Agents themselves.

Independent temporary Go-overlay diagnostic passed (0.271s): production constructor with RootAgent=claude leaves the start agent empty; Right does nothing and AgentSticky remains false. Positive control supplying the canonical inventory makes the same selector choose Codex with AgentSticky=true. No repository implementation files or live slots were changed. Filed from a scratch checkout outside the numbered-slot layout; leave unclaimed.
