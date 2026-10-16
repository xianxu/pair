---
id: 000598
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Diagnose helper startup cancellation timeout

## Problem

Recorded #586 integrated suite fails before the review fixtures in LifetimeTests.test_cancellation_during_startup for SIGINT with delay0.02. Helper process does not exit within the existing3s wait. No output retained because the fixture redirects both streams to DEVNULL.
