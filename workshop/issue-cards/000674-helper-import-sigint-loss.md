---
id: 000674
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Startup SIGINT swallowed by importlib weakref callback before helper handler

## Problem

The pair#650 integration suite on aa8176700b0e50b230c72eeaf190c9b284342810 failed in LifetimeTests.test_cancellation_during_startup (SIGINT, delay 0.02 s) despite the pair#598 fixture normalization being present. The helper did not exit within the existing 3 s bound; captured stderr reports KeyboardInterrupt ignored in importlib._bootstrap module-lock weakref callback. The caller SIGINT was default_int_handler. This differs from the inherited SIG_IGN cause fixed in #598. No causal link to slug gating is shown: the helper and lifetime test files are unchanged by #650.
