---
id: 000642
status: open
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '12df7554958d21ca0eac7cf34147d127a141a32e' # card fields mirrored from issue-cards; edit via sdlc
---

# Changelog Lua test shares reload scratch path across concurrent suites

## Problem

During pair#637 PR #396 recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41, nvim/changelog_test.lua failed `reload replaced buffer with file contents` and `reload put cursor at newest entry`. This happened after a 1466 s queue wait; the test execution stopped after 27 s, before Go tests. The fixture uses the fixed TMPDIR filename pair-changelog-reload-test.md, writes it, reloads it, then removes it. Another concurrent invocation shares/removes that path. Other fixture paths in the same file are fixed too.

## Spec

Give each invocation ownership of its scratch paths and cleanup. Preserve all reload, annotation, read-only and cursor assertions. Add a deterministic concurrent-original control or equivalent collision regression.

## Done when

- Concurrent invocations do not overwrite or remove one another's reload/annotation scratch files.
- Original assertion set passes independently and concurrently.

## Plan

- [ ]

## Log

2026-10-17: No nvim source differs in #637. Original baseline ce783cd4a474d939c174bb079bc687efc2cc457a was extracted with git archive; both test and loaded Lua modules came from it. An isolated current test passed. Forty unsynchronized concurrent-original runs also passed (not evidence of absence).

A controlled original/original interleaving reproduced exactly both failures: invocation A pauses immediately after its real writefile returns; invocation B runs the unmodified original test to completion in the same private TMPDIR and removes the shared file; release A, which fails both reload assertions and exits 1. B exits 0. The gate is a test-launch Lua wrapper around writefile; all original test/product Lua bodies remain unchanged. No sleeps, production edits or foreign paths. This proves a pre-existing filename collision; the specific other remover in the original suite is unknown, so it is consistent with the suite failure rather than a traced attribution to a named peer.

Evidence directory: /tmp/pair637-ed1erwu8. Wrapper hold-original.lua, original-controlled-collision.log, original nvim source in original/nvim, and classification.json retain commands/outcomes. Failed recorded suite: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7e7efa0d33e0f88288f8746ce6e1d8d3.log (SHA256 2ae07bbeb087aa0eb0fb1b1dc0c2635c5260c9a76f28402e1bd541e72a40fdbc).
