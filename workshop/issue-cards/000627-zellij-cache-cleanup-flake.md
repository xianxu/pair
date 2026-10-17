---
id: 000627
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Live shortcut fixture races Zellij cache removal during TempDir cleanup

## Problem

First observed 2026-10-17T00:28:29Z on pair#604 PR #381 integrated head f8e3e1c7970f0ab6505478e74ec823007d2c3a5e, GitHub Actions run 38986306428, conformance job 117258474451. TestAgentShortcutInputConformanceLive passed every input subtest, then failed during Go TempDir cleanup: unlinkat .../002/Library/Caches/org.Zellij-Contributors.Zellij: directory not empty.

Failed log: /tmp/pair604-integrated-conformance-failure.log. Durable CI evidence: https://github.com/xianxu/pair/actions/runs/38986306428/job/117258474451 . Previous PR-head conformance run 38983105918 passed unchanged clipboard implementation. No repeat of the failed job yet.
