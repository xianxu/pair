---
id: 000757
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping)

## Problem

product-shaping (ops) grows the real app out of its mock screen by screen. In development, unbuilt screens should
still be clickable as mocks; nothing half-mocked may ship.
