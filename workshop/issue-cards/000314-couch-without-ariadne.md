---
id: 000314
status: open
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
github_issue:
---

# Support Couch slots without Ariadne tooling or repository setup

## Problem

Couch supports ordinary repositories, but numbered-slot setup currently requires
`sdlc workspace` identity lookup and unconditionally invokes `weave compile`.
This makes Ariadne tooling/setup an unnecessary dependency for non-Ariadne repos.
