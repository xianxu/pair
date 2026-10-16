---
id: 000620
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Review branch restore intermittently misses landed acknowledgment

## Problem

The recorded full suite for #604 failed in tests/review-branch-restore-test.sh: its review-pane handoff did not publish data/landed.json within the fixture's five-second wait, then read_text raised FileNotFoundError. The fixture printed an empty draft Neovim log; it did not retain the review pane's state at that boundary.

Failed head: fe4eb6d6e83bd53b9d42fcf27811a383c348b222. First observed here: 2026-10-16, during the #604 recorded suite. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log. The local-check record reports exit 2 in 5m38s and SHA256 prefix eee92d770af5. The failure was before go test ./..., so #604's test executables and their private clipboard environment had not run.

One unchanged isolated repeat passed: bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh; log /tmp/pair604-review-branch-restore-repeat.log. The current head differed only by #604's atlas/issue prose; fixture and executable source were unchanged. This is one failure and one passing repeat, not a calibrated rate.
