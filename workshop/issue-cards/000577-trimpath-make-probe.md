---
id: 000577
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: high
---

# Trimpath command-line assignment breaks the required suite environment probe

## Problem

After #572, the declared required suite is `make PAIR_GO_TRIMPATH=-trimpath test`. Its Make environment probe fails deterministically before the Go suite: `FAIL: knob PAIR_GO_TRIMPATH did not reach the test recipe`. This blocks #556 PR349 after its mandatory main integration. Bare make test-make-env passes.
