---
id: 000309
status: open
deps: [ariadne#244, ariadne#245, ariadne#246, pair#307, pair#308]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
---

# Slots v2: three-workspace acceptance trial

## Problem

The project is complete only when the UI and SDLC workflow work together through real independent issues in parley.nvim.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Run the operator acceptance trial with parley.nvim primary coordinating and :1/:2 completing two independent, suitable issues through integration. Select trial issues at execution time; do not repurpose historical slot-design issues as this project’s tasks. Exercise creating additional threads, repo-wide parked-thread refusal/activation, grouped switcher/bar selection, independent preferences, dirty park/resume, retained directories, and reuse after landing.

Also branch a destination issue from another workspace’s clean committed SHA and demonstrate later source changes do not propagate. Prove landing leaves the resting baseline unchanged and explicit refresh advances it; starting a subsequent issue does not refresh implicitly. Include dirty/unpublished-work refusals and interrupted-operation recovery in automated fixtures, with the live trial covering normal integration. Record commands, refs, paths, before/after SHAs, and UI evidence, keeping raw logs out of the repo. No timeline or implementation success is assumed by creating this task.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Use `/workspace/worktree/parley.nvim-slot1/parley.nvim` and `/workspace/worktree/parley.nvim-slot2/parley.nvim`, each with its own ordinary sibling `ariadne` clone initially selected from origin/main. Record dependency origins and initial/current SHAs as well as main-worktree refs. Demonstrate that primary dependency activity and explicit changes inside one numbered environment do not change the other's sources or generated-link targets; repeated setup/resume preserves selected revisions and dirty/local work. Ordinary setup/build must leave the machine-wide tool supplier unchanged.

Exercise a small suitable dependency issue/update or code change driven through a numbered parent thread, publishing through that dependency repository's existing SDLC workflow before the parent when required. No automatic multi-repository merge is part of acceptance; this is workflow compatibility, not a new orchestration feature. Prove dependencies remain present through main-feature landing/reuse and receive no automatic Couch entries. The primary's coordination role is a trial convention; coordinated cross-repository development is supported from any slot. Do not manufacture an unnecessary live code change solely for this demonstration; use a suitable task or isolated publication fixture.

## Done when

- New host slots start at an accepted local source commit containing unpublished relevant work; record source address/SHA and prove remote/main tracking does not replace that starting point. Dependency clones independently initialize per their remote contract.
- Two independent parley.nvim issues land from :1/:2 while :0 coordinates; all addresses/directories remain identifiable and reusable.
- Live evidence covers parked admission, activation, grouping, separate preferences, and dirty park/resume.
- Captured before/after refs prove exact-commit branching, unchanged resting baseline at land, explicit refresh, and no cross-workspace mutation.
- Automated recovery/refusal evidence is linked; any remaining failure becomes explicit follow-up scope before declaring the project done.
- Atlas/operator guidance and the project record reflect the observed final workflow and remaining limitations.

- Evidence covers independent origin/main dependency initialization, explicit revision changes, repeat setup, private generated-link targets and unchanged installed-tool supplier.
- Normal dependency-first SDLC work can be driven from a numbered thread; landing/reuse retains dependency work without automatic recursive publication or dependency UI entries.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Prepare the acceptance matrix, choose trial issues, and verify installed tools reflect the delivered code.
- [ ] Run the automated integration matrix and the live three-workspace trial.
- [ ] Record reproducible evidence, resolve acceptance failures, and reconcile project completion.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Exercise nested dependency isolation and ordinary cross-repo work

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — verify local-source host initialization

Reason: operator clarified new slots inherit relevant committed local work.
Delta: acceptance now proves the initial host HEAD/resting branch use the accepted
local source SHA even when it differs from remote main; dependency initialization
remains separately governed by the agreed remote-clone contract.
