---
id: 000869
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '395c3054431d9eed5064c621c1503cfd28d1082b' # card fields mirrored from issue-cards; edit via sdlc
---

# Publish cancellation fixture PID readiness atomically

## Problem

TestCLILiveLeaderCancellationJoinsBeforeCompletion intermittently parses an empty PID marker. First observed in #863 recorded suite attempt0c7a9818a957d392fc43e9fb44889545 on d00c828e28d41b317b924f3bde4f2f0a622076f7 (2026-10-24; suite exit2 in18m38s). cli_group_test.go:84: strconv.Atoi: parsing "": invalid syntax. Package duration20.261s. Six other failures in that attempt are #863 stale fixtures, not this bug.

The shell uses printf redirected directly to the final marker path. Its open publishes an empty file before printf writes the PID. waitFixture waits only for os.Stat success, then the test reads once and parses it, so a valid scheduling gap reaches this exact empty-string error. This is a fixture readiness race, not evidence of a cancellation-owner defect. #863 changes none of this package; Git diff from the failed head to the repeated head is empty for cmd/internal/testcouch/agentfixture.

One unchanged focused repeat passed (package0.130s, exit0). That is a targeted repeat, not a passing whole suite on the same commit; no known-flake registration is claimed.

## Spec

Publish the PID atomically or make the readiness observation require the complete valid expected PID before cancellation. Preserve the real leader-death/join assertions and the existing bounded timeout; never fix this by sleeping or increasing a budget. Add an injected hold between marker creation and content publication so the original test fails deterministically.

## Done when

- An injected marker-open/write gap cannot cause empty or partial PID parsing.
- The test still cancels a genuinely live leader and proves joined completion.
- Original timeout and process ownership/cleanup assertions remain.

## Plan

- [ ]

## Log

Failed full log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/0c7a9818a957d392fc43e9fb44889545.log, SHA256 prefix13f839805c08. Exact failure: github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCLILiveLeaderCancellationJoinsBeforeCompletion. Focused repeat command: go test ./cmd/internal/testcouch/agentfixture -run '^TestCLILiveLeaderCancellationJoinsBeforeCompletion$' -count=1 -timeout=60s. Its log is retained in #863 evidence/independent-cli-marker-focused-repeat.log. No production or fixture change folded into #863; unclaimed follow-up.
