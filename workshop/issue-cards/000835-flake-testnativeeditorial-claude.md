---
id: 000835
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/couch.TestNativeEditorial/claude]
---

# flake: TestNativeEditorial/claude

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestNativeEditorial/claude: Native qualification on branch HEAD 4619aee1f: startup attempt failed after 192.728s, Zellij 0.45.1 server at 04:15:24.884 panicked at zellij-server/src/lib.rs:1462:26, Option::unwrap on None. Owned root pc-1740203248 was fully joined and cleaned. Retry started normally; final native test passed 25.096s, Claude 8.32s and Codex 10.16s, roots pc-1410873069 and pc-1775381185 fully cleaned. Commands: PAIR_NATIVE_EDITORIAL=1 go test ./cmd/couch -run ^TestNativeEditorial$ -count=1 -v -timeout 12m. Qualification test was uncommitted and its expected-exit handling was corrected between attempts; no production startup or Zellij code changed. This proves intermittent native startup independently of editorial assertions, not exact committed-source qualification. Preserve that limitation; root cause unresolved, no retry policy or timeout increase is proposed.
