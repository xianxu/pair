---
id: 000831
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: '453c22f7da76fe079e7b1014797b7980033ccfab' # card fields mirrored from issue-cards; edit via sdlc
---

# Symlink TMPDIR breaks lookup fixture

## Problem

TestLookupSocketUsesNativeSymlinkParentTraversal fails both temporary-root cases while constructing its frames.json fixture, before exercising the query. Found during #759 shared-upgrade-only diagnostic suite; a matched original-module control reproduces it, so it is not unique to those upgrades. The isolated Go1.27.3 baseline suite passed; this is environment-sensitive fixture behavior, not established flaky production lookup.

## Spec

Keep native symlink/parent traversal assertions intact. Separate owned fake-file allocation from the TMPDIR value intentionally containing alias/.., and diagnose the mismatch between native traversal and filepath.Join cleaning. Pin fixture setup under this temporary-root topology, with positive source/readiness and exact cleanup. Do not weaken socket-presence or retry-count assertions or alter production path resolution to accommodate a fixture defect.

## Done when

- Both temporary-root subcases construct and read their intended frames and exercise native lookup/retry assertions.
- A regression covers fixture allocation while TMPDIR contains symlink plus parent traversal.
- Explicit, temporary and runtime root cases pass, including positive and negative socket observations, with bounded private cleanup.

## Plan

- [ ]

## Log

2026-10-22: #759 diagnostic on source bfa3f4a22c1e2583e1b1b2eefc118e19ec54d70d, Go1.27.3. Full upgraded-module run: TestLookupSocketUsesNativeSymlinkParentTraversal/temporary/native-present-cleaned-absent and /temporary/native-absent-cleaned-present fail at zellij_lookup_path_test.go93: open /Volumes/TestScratch/s761-.../sockets/TestLookup.../001/frames.json: no such file or directory. One exact-test ORIGINAL-module control, same source and explicit environment, fails identically; explicit/runtime cases pass. No candidate retry or assertion/deadline change.

Source inspection: the test sets TMPDIR to base/alias/../sockets before newEmptyReplyFake calls t.TempDir and filepath.Join(dir, "frames.json"). The topology deliberately makes lexical cleaning differ from native traversal. This is the suspected fixture boundary, not a completed fix. Evidence committed with #759 at workshop/evidence/000759/tsnet-dependency-20261022/: full shared-upgrades-suite.log and launcher-control/{command,result,test.log}. Unclaimed follow-up; no repair folded into #759.
