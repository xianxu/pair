---
id: 000845
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# Native Couch conformance isolation omits Git upload-pack helper

## Problem

Pair's native Couch conformance bridge fails before readiness on macOS because isolated workspace provisioning cannot execute git-upload-pack for its local private upstream. This blocks ariadne#577 native Couch service conformance evidence; no TL/worker replay or successful service qualification was claimed.
