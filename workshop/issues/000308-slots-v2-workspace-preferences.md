---
id: 000308
status: open
deps: [pair#306]
github_issue:
created: 2026-09-22
updated: 2026-09-22
estimate_hours:
---

# Slots v2: independent workspace preferences

## Problem

Numbered threads need the same agent and launch-parameter preferences as the primary without leaking settings across slots.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Persist preferences per durable workspace identity, including selected agent and supported launch parameters already available to the primary. Resume uses that workspace’s recorded settings. Define inheritance on first creation from existing repo/default configuration, then preserve independent values for :0/:1/:2. Preserve backward compatibility for existing primary configuration and define preference behavior on thread replacement.

Preferred-model customization is excluded from v2: add no new model picker, slot model-default field, or slot-number-to-model mapping. Existing underlying harness launch semantics remain governed by the current configuration contract. No fixed roles or permissions are implied by slot number. ARCH-PURPOSE: extend the existing preference authority rather than adding a competing settings store.

## Done when

- :0, :1, and :2 retain independently selected agents and supported launch parameters across park/resume and process restart.
- Changing one workspace’s preferences does not mutate another workspace or repo defaults.
- Existing primary preferences migrate/read compatibly; new-slot inheritance and replacement behavior are documented and tested.
- Invalid settings use the existing actionable validation path; no preferred-model customization is introduced.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Inspect current preference persistence and specify workspace keying, first-use inheritance, and replacement behavior.
- [ ] Implement persistence/launch wiring with restart and isolation tests.
- [ ] Document the supported per-workspace settings and verify primary compatibility.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.
