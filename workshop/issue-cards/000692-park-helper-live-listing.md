---
id: 000692
status: open
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
github_issue:
severity: medium
started: 2026-10-19T05:14:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Native park helper refuses a live session before attachment

## Problem

macOS conformance run39201615767, job117895590257, attempt1 on f479a6e80c2849d687539c6c4377d31b0fdc12c5 (#687 PR447) failed TestParkLifecycleLiveIntentOnlyMutation before attachment: park_lifecycle_live_test.go:255 reports helper exit3; its PTY says park helper attach returned code=1 error=terminal "pair-park-5220-mut" is not listed live. The test ended after0.62s, before its750ms intent-only observation phase. Runner loadavg16.55/26.44/14.88,3CPUs. No timeout increase or product retry proposed.

CI: https://github.com/xianxu/pair/actions/runs/39201615767/job/117895590257
Retained complete failed job log: /Users/xianxu/workspace/pair/.git/pair-687-evidence/ci-39201615767-attempt1.log
SHA2562245c9b554b27c8e95fc208d69f5252ef603d20c8715d090a7963f762d0fa088. No uploaded artifacts in that run.

Retained recovery capture has12records: park session start returns nil at04:54:30.017792Z. Parent list-clients repeatedly returns client1 on terminal_1 between04:54:30.077899Z and04:54:30.259Z; one intervening query at04:54:30.209798Z exits0 with empty output and is correctly reported as no reply. Child helper's own list-sessions fence output is not captured, so absence versus exited state in that listing remains unknown. The matching refusal comes from launcher.requireListedLiveTerminal (session_handoff_os.go), not retention or the test's completion timer.

Same production code passed this CI family in earlier run39201203710 (3m11s) on230a100ed9601fbb70209de6ad677f19ea8af004; the intervening #687 change only replaces a pure predicate test's fixture and updates process docs. Earlier run39200274828 also passed. Classification: intermittent native park setup/live-session observation refusal, cause not yet proved. This differs from #601's explicit client-disconnect exit4, and from #660's direct reattach symptom; do not assume the upstream ID-reuse race without evidence. #636 retained recovery-query diagnostics provide the next failure seam.
