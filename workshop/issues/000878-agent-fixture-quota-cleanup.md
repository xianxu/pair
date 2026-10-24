---
id: 000878
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '84f381452df78368b4ad28c498d6a2256d0a4b5e' # card fields mirrored from issue-cards; edit via sdlc
---

# Agent fixture quota exhaustion misses completion and cleanup join

## Problem

The full default local Go set on clean main e2f58424b124c38b4340acee459f21fa009394f2 failed TestRunStopAfterOrdinaryRequestQuotaExhausted in cmd/internal/testcouch/agentfixture. It reported run_test.go:324 fixture observation deadline, then run_test.go:311 fixture failed to join during test cleanup; shortroot.go:28 could not remove owned actor/results because the directory was not empty. Test duration13.45s, package117.146s. This was a once-only main baseline comparison for #866, not a candidate-only finding.

Exact command from the detached scratch Pair root: bash tests/lib/with-pair-test-binary.sh go test ./... -count=1 -timeout20m. GOFLAGS=-p=3 -trimpath; TMPDIR and GOTMPDIR=/Volumes/TestScratch/tmp/pair-slot5. Main source was clean before/after and its exact Git head unchanged.

Complete log /Volumes/TestScratch/pair866-main-go-buzlps33/go.log SHA2569cfce92fe2f6bc9ca95f7e03b478728ddbdff02d0027734b7e08e349f6ec59b9; metadata.json/result.json beside it record exact head/command/exit1. Scratch source checkout was removed after completion; captures remain. The other main failure TestCLILiveLeaderCancellationJoinsBeforeCompletion is already #869 and outside this issue.

## Spec

Investigate ordinary-request quota exhaustion in the retained agent fixture, including its stop transition, final observation, owned goroutine/process joins and filesystem cleanup. Preserve the existing request quota and semantic completion assertions. Do not hide a live fixture behind cleanup suppression or an arbitrary timeout increase.

## Done when

- Exhausting the ordinary request quota reaches the intended stop/completion state and joins all owned work before cleanup returns.
- The existing test passes with deterministic coverage of delayed final observation and cleanup ordering.
- Failure diagnostics identify any unjoined actor/results writer, and the owned short root can be removed without a concurrent writer.

## Plan

- [ ]

## Log

TL233cf35e directs an unclaimed issue, after searching active main issue details for the exact test/contour. No production/test change or flake registration was made under #866 for this independent main failure.
