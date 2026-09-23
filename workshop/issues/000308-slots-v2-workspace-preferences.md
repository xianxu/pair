---
id: 000308
status: open
deps: [pair#306]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
---

# Slots v2: independent workspace preferences

## Problem

Numbered threads need the same agent and launch-parameter preferences as the primary without leaking settings across slots.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Persist preferences per durable workspace identity, including selected agent and supported launch parameters already available to the primary. Resume uses that workspace’s recorded settings. Define inheritance on first creation from existing repo/default configuration, then preserve independent values for :0/:1/:2. Preserve backward compatibility for existing primary configuration and define preference behavior on thread replacement.

Preferred-model customization is excluded from v2: add no new model picker, slot model-default field, or slot-number-to-model mapping. Existing underlying harness launch semantics remain governed by the current configuration contract. No fixed roles or permissions are implied by slot number. ARCH-PURPOSE: extend the existing preference authority rather than adding a competing settings store.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Keep preferences keyed to the durable main workspace address despite its nested checkout path `/workspace/worktree/<repo>-slotN/<repo>`. Ordinary dependency clones accessed through that thread do not receive additional Couch preference records, inherited agent launches, or numbered identities. Primary and numbered slots retain the same supported preference capabilities.

## Done when

- :0, :1, and :2 retain independently selected agents and supported launch parameters across park/resume and process restart.
- Changing one workspace’s preferences does not mutate another workspace or repo defaults.
- Existing primary preferences migrate/read compatibly; new-slot inheritance and replacement behavior are documented and tested.
- Invalid settings use the existing actionable validation path; no preferred-model customization is introduced.

- Nested path resolution retains stable main-workspace preference keys; acquiring sibling dependency clones creates no extra preference or launch records.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Inspect current preference persistence and specify workspace keying, first-use inheritance, and replacement behavior.
- [ ] Implement persistence/launch wiring with restart and isolation tests.
- [ ] Document the supported per-workspace settings and verify primary compatibility.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Preferences belong to the numbered thread, not each dependency

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — build preference UX on local storage from #306

Reason: `.couch` is the authoritative slot store across conversation replacement.
Delta: #306 routes the existing path preference API to
`<environment>/.couch/preferences.json` and preserves it during fresh conversation.
This issue should reuse that storage and successful-launch publication, finishing
inheritance/selection UX and restart isolation coverage rather than introducing
another store. Primary preferences remain in the global backend. No new preferred
model setting is authorized.
