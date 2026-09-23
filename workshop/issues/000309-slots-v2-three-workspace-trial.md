---
id: 000309
status: open
deps: [ariadne#244, ariadne#245, ariadne#246, pair#307, pair#308]
github_issue:
created: 2026-09-22
updated: 2026-09-22
estimate_hours:
---

# Slots v2: three-workspace acceptance trial

## Problem

The project is complete only when the UI and SDLC workflow work together through real independent issues in parley.nvim.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Run the operator acceptance trial with parley.nvim primary coordinating and :1/:2 completing two independent, suitable issues through integration. Select trial issues at execution time; do not repurpose historical slot-design issues as this project’s tasks. Exercise creating additional threads, repo-wide parked-thread refusal/activation, grouped switcher/bar selection, independent preferences, dirty park/resume, retained directories, and reuse after landing.

Also branch a destination issue from another workspace’s clean committed SHA and demonstrate later source changes do not propagate. Prove landing leaves the resting baseline unchanged and explicit refresh advances it; starting a subsequent issue does not refresh implicitly. Include dirty/unpublished-work refusals and interrupted-operation recovery in automated fixtures, with the live trial covering normal integration. Record commands, refs, paths, before/after SHAs, and UI evidence, keeping raw logs out of the repo. No timeline or implementation success is assumed by creating this task.

## Done when

- Two independent parley.nvim issues land from :1/:2 while :0 coordinates; all addresses/directories remain identifiable and reusable.
- Live evidence covers parked admission, activation, grouping, separate preferences, and dirty park/resume.
- Captured before/after refs prove exact-commit branching, unchanged resting baseline at land, explicit refresh, and no cross-workspace mutation.
- Automated recovery/refusal evidence is linked; any remaining failure becomes explicit follow-up scope before declaring the project done.
- Atlas/operator guidance and the project record reflect the observed final workflow and remaining limitations.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Prepare the acceptance matrix, choose trial issues, and verify installed tools reflect the delivered code.
- [ ] Run the automated integration matrix and the live three-workspace trial.
- [ ] Record reproducible evidence, resolve acceptance failures, and reconcile project completion.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.
