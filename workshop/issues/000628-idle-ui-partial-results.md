---
id: 000628
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '45d40d1cf4d57a66b4cd2c0144268c530f6557cd' # card fields mirrored from issue-cards; edit via sdlc
---

# Idle UI invariant intermittently returns only 16 of 20 activity results

## Problem

During pair#619 close review on 2026-10-16 around 18:39 PDT, head 3f5f7fed4f03aa2829be24ae339cb473260bf000, the affected-package run failed TestWorkbenchPerformanceIdleUITick (27.72s): workbench_activity_invariant_test.go:77 reported pass results observed=16 and four failed rows (work-01, work-03, work-04, work-20). Couchtty package failed in 65.202s. This is a different assertion from #626's outer 30-second activity-worker wait; shared causality is not established.

Retained failing tool record: /tmp/pair619-review-activity-failure.json; source review session 01a14781-9960-7871-8ade-430a4a8c28fa. The version-controlled #619 close-review sidecar records the failing package run and unchanged targeted passing repeat. First recorded full suite on the same code (f500a1ccf3b165f070d40a2b5478605b6a02b48b) passed in 12m46s, log SHA256 prefix 6746e0ae12fd. Later changes are documentation only. No code or timeout fix was made in #619.

## Spec

Determine why the real activity fixture returns failed rows under a package run while the unchanged isolated repeat passes. Preserve the zero-read/decode invariant and distinguish transport/query deadlines from missing evidence. Load or per-row deadlines are hypotheses only, not measured causes.

## Done when

- Reproduce the partial-results assertion, identify its cause and add a deterministic regression.
- Prove the correction without weakening the idle-tick accounting or merely increasing a timeout.

## Plan

- [ ]

## Log

The reviewer ran an unchanged isolated repeat alongside the relevant key pass-through/routing tests; all passed. Issue #619 changes only help metadata and documentation. Its SHIP verdict stands; this independent fixture symptom is retained separately for investigation.
