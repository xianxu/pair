---
id: 000623
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '5793f51edca7d4024ce811b24beb27eeb79545a4' # card fields mirrored from issue-cards; edit via sdlc
---

# Review observation positive control intermittently loses external edit reload

## Problem

tests/review-observation-diagnostics-test.py intermittently fails its positive control, ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The nested real editor/Git fixture asserts: matching branch external edit did not reload; snapshot lost no-final-newline state.

Observed during #604's recorded rerun on fe4eb6d6e83bd53b9d42fcf27811a383c348b222, first seen here 2026-10-16 at about 16:01 PDT. Eight diagnostic cases ran in 63.905s; one failed. The full local-check stopped before Go tests, exit 2 in 6m59s after 1239s in the machine queue. Retained log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/ffdc28f65f4aad6e6fdf5f09c7feeb01.log, SHA256 prefix 7c703e6585f2.

The preceding complete make -k suite passed this diagnostic command and all other shell/editor/Go checks, with no production/test changes between attempts. Passing log: /tmp/pair604-keepgoing-suite.log, SHA256 79a1e5cb433c82e9af9207831d7deb750f9b8280a826238059423d03fc5b8624. One pass and one failure do not establish a calibrated rate.

## Spec

Use the retained observation ordering and fresh/wanted identity diagnostics to determine why the matching-branch external edit was not reloaded with its exact no-final-newline bytes. Distinguish a delayed observation from lost or rejected state; do not increase timeouts without diagnosing the ordering.

The hub reported load average about 140 and links likely load pressure to #618's Couch Zellij polling. That is a working hypothesis, not a measured cause of this specific assertion. The #604 clipboard diff is not entered by the review fixture, and no clipboard production/test code is to be changed for this failure. Related independent first failure: #620 (missing branch-restore landed.json acknowledgment); that case passed in this failing run.

## Done when

- A reproducer and diagnostics establish the missed reload/byte-preservation ordering.
- A regression defends the identified ordering while preserving exact no-final-newline behavior.
- Repeated runs report before/after failure counts and host conditions; prior failed attempts remain visible.

## Plan

- [ ]

## Log

- 2026-10-16: Filed unclaimed per the hub's explicit decision, delivery efadcd45-6fb6-487e-9392-7fc79782168e. The hub authorized one additional recorded #604 rerun; another failure must stop and be reported, with no indefinite retry loop. No fix is included in #604.
