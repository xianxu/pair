---
id: 000772
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# flake: TestDisposableCouchFleet/codex

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestDisposableCouchFleet/codex: Independent close reviewer on exact head23a5b119d111b985de0cc18bb5cedb42c1cf838c: first native command FAIL103.75s during Codex fixture:3 startup (nonce operation-730874d8fa66046c985836b933151494, attach terminal input has ended), unchanged-head repeat PASS107.43s; both exact cleanups succeeded. Original review session01a15ed9-1912-7043-8dcd-cc74e7537546 tool events126 and182 retained locally. Production couchtty console.go checks InputEnded before reservation and before commit with identical text; only final attach error survives, stage/stream/output/exit cause absent. This establishes intermittent product attach/PTY-lifecycle failure, not a harness-only cause. Need bounded endpoint termination/child-exit/output evidence before rollback; no timeout bump, no blind retry, root cause unresolved. Discovered during pair#744 BR-5; native acceptance sequencing requires TL disposition.
