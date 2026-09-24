---
id: 000314
status: open
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
---

# Support Couch slots without Ariadne tooling or repository setup

## Problem

Couch supports ordinary repositories, but numbered-slot setup currently requires
`sdlc workspace` identity lookup and unconditionally invokes `weave compile`.
This makes Ariadne tooling/setup an unnecessary dependency for non-Ariadne repos.

## Spec

Decouple Couch workspace identity/provisioning from mandatory Ariadne tools.
Regular Git repositories retain numbered worktrees, grouping, .couch state,
preferences and the same thread lifecycle. Run Weave setup only for repositories
configured to require it, without treating setup failures as absence of Ariadne.
Use Git-backed identity for ordinary repos; avoid requiring an sdlc binary just
to create or resume Couch slots. Preserve the existing Ariadne integration for
configured repos. This task is filed only; implementation is not authorized here.

## Done when

- An ordinary Git repo can create, restart, park/resume and replace slot conversations with sdlc/weave absent from PATH.
- Existing configured Ariadne repos still run required setup and surface genuine errors.
- Tests cover both paths and retained directory/preference behavior; docs state optional integrations clearly.


## Plan

- [ ] Inspect identity and readiness dependencies; design the optional integration boundary.
- [ ] Implement and verify ordinary Git and configured Ariadne paths.


## Log

### 2026-09-23

Operator requested filing this separately during the Add slot UI work. No code changes included.
