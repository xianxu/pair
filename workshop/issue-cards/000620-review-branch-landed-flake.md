---
id: 000620
status: codecomplete
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T10:31:09-07:00
actual_hours: 0.41
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
tracker:
    version: 1
    completion:
        token: close-91e745d04086
        repository: github.com/xianxu/pair
        reviewed_head: 12d613dad701c542e87572a66425d5ddebd84f05
        evidence_commit: 6ccce44bac60ac3170c8de88241b8b7f31094e00
---

# Review branch restore intermittently misses landed acknowledgment

## Problem

The recorded full suite for #604 failed in tests/review-branch-restore-test.sh: its review-pane handoff did not publish data/landed.json within the fixture's five-second wait, then read_text raised FileNotFoundError. The fixture printed an empty draft Neovim log; it did not retain the review pane's state at that boundary.

Failed head: fe4eb6d6e83bd53b9d42fcf27811a383c348b222. First observed here: 2026-10-16, during the #604 recorded suite. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log. The local-check record reports exit 2 in 5m38s and SHA256 prefix eee92d770af5. The failure was before go test ./..., so #604's test executables and their private clipboard environment had not run.

One unchanged isolated repeat passed: bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh; log /tmp/pair604-review-branch-restore-repeat.log. The current head differed only by #604's atlas/issue prose; fixture and executable source were unchanged. This is one failure and one passing repeat, not a calibrated rate.
