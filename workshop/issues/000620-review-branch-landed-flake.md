---
id: 000620
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'e9cbc20cb938cd88478dfa5fb68bffb1b65de00e' # card fields mirrored from issue-cards; edit via sdlc
---

# Review branch restore intermittently misses landed acknowledgment

## Problem

The recorded full suite for #604 failed in tests/review-branch-restore-test.sh: its review-pane handoff did not publish data/landed.json within the fixture's five-second wait, then read_text raised FileNotFoundError. The fixture printed an empty draft Neovim log; it did not retain the review pane's state at that boundary.

Failed head: fe4eb6d6e83bd53b9d42fcf27811a383c348b222. First observed here: 2026-10-16, during the #604 recorded suite. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/07f7702637f31ade0443abc6b434b7f2.log. The local-check record reports exit 2 in 5m38s and SHA256 prefix eee92d770af5. The failure was before go test ./..., so #604's test executables and their private clipboard environment had not run.

One unchanged isolated repeat passed: bash tests/lib/with-pair-test-binary.sh bash tests/review-branch-restore-test.sh; log /tmp/pair604-review-branch-restore-repeat.log. The current head differed only by #604's atlas/issue prose; fixture and executable source were unchanged. This is one failure and one passing repeat, not a calibrated rate.

## Spec

Capture the review-pane handoff/landing state at the missed publication boundary and determine why acknowledgment is missing. Distinguish scheduling delay, lost watcher event, rejected context and editor failure. The existing five-second wait followed by an unconditional file read is where the failure is observed, not a proven cause. Do not merely increase that timeout.

The #604 production diff only adds the explicit Couch switcher clipboard action and its clipboard-runtime seam. This fixture runs review readiness/editor handoff operations, without entering that action. Keep diagnosis/fix independent of #604.

## Done when

- A reproducer and bounded diagnostics identify the missed handoff acknowledgment.
- A regression exercises the identified ordering through the real fixture seam.
- Repeated checks report before/after failure counts and conditions without weakening the assertion or hiding failed attempts.

## Plan

- [ ]

## Log

- 2026-10-16: Filed unassigned after one recorded failure and a passing unchanged targeted repeat. #604 retained its failed suite and started make -k to complete all remaining checks; no fixture or timeout change is included there.
