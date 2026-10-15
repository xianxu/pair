---
id: 000568
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Review fresh-restore headless Neovim exits without failure diagnostics

## Problem

During pair#556 M1 recorded verification, the unchanged shell test tests/review-fresh-restore-test.sh failed at missing-history-does-not-adopt-old-receipt. The Python assertion saw a nonzero headless Neovim return with empty combined output; its assertion omits the return code, so the retained log cannot distinguish a signal from ordinary exit. The isolated full script immediately passed all five cases on the same code. This is not a confirmed root cause or a #556 recovery regression.
