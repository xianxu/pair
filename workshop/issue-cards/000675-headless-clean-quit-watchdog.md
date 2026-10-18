---
id: 000675
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Headless clean-quit watchdog expires under load

## Problem

The clean-exit positive control in tests/run-headless-test.sh timed out under machine load during #672's integrated suite. The test runs nvim --headless -c 'qall!' under a 10-second watchdog and expected exit 0, but received 124. The test does not retain the captured positive-control output on failure, so this evidence proves watchdog expiry, not whether Neovim startup or shutdown consumed the bound.
