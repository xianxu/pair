---
id: 000748
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6592cdebea8b78963b3f59d2923f8fff093980f6' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestRecoveryRealHelperAndSessionConformanceLive/warm (zellij list-panes no reply after helper exit) holds the pair landing line

## Problem

`couchcore.TestRecoveryRealHelperAndSessionConformanceLive/warm` in the required `couch-zellij-conformance` job fails
intermittently on GitHub macOS: "recovery session state could not be checked; retry: observe live session owner:
zellij action list-panes --json --command returned no reply" (helper exited before recovery; the final query got no
reply). It failed on pair#706's heads twice on 10-20 (run 39501347923 attempt 1; run 39521409698 job 118965862227,
log SHA256 cd823c3894b2ef291a8c4ae4be3b0c70c062aa9ada5fa3d8b436538e393c4f69) and passed on rerun once; #706 touches
no couchcore, launcher or zellij code. Registered in pair#636. Because conformance is required, each occurrence
holds the pair landing line.

## Spec

Root-cause it (no timeout bump, no retry loop): reproduce with the exact CI geometry and timing (a fresh HOME, pinned
Zellij 0.45.1), find why list-panes gets no reply after the helper exits (session teardown racing the query? a
socket that's gone?), and fix the test or the product, whichever is wrong. Record the capture as a fixture.

## Done when

- A deterministic reproduction exists (red before the fix), the fix passes it, and 20 consecutive local runs of the
  case pass; the pair#636 entry is closed with the cause.

## Plan

- [ ]

## Log

### 2026-10-20
