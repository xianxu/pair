---
id: 000764
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0c6880a7456228355b1367684bf105892c09a052' # card fields mirrored from issue-cards; edit via sdlc
---

# Controlled Zellij fixture closes its PTY twice and hides cleanup errors

## Problem

ControlledZellij.KillClient closes its PTY but retains the non-nil pointer. ControlledZellij.Close then joins a second Close error, even after exact server cleanup succeeds. TestStartupManagedAttachDetachTimingLive discards that cleanup error. A #761 scratch acceptance overlay surfaced close /dev/ptmx: file already closed after successful attach/detach and reattach/detach; exact server generation was absent and private roots removed.

## Spec

Make the helper lifecycle own and retire its client/PTY handles exactly once. Preserve real shutdown and observation errors; callers must not discard meaningful cleanup failures. This is fixture cleanup, independent of #761 native lookup recovery.

## Done when

A focused regression executes KillClient then Close and observes successful exact cleanup without a spurious closed-PTY error. A genuine shutdown/observation failure still surfaces. Audit callers that discard Close errors and keep native cleanup proof explicit.

## Plan

- [ ]

## Log

Observed during #761 on 830f29d9ef0892b6fb8d06869a4840eed28f7ad9. Relevant source: cmd/internal/pairlifecycletest/live_zellij.go, KillClient and close; caller startup_detach_timing_live_test.go. Retained log /Volumes/TestScratch/tmp/pair-slot5/pair-761-native-reattach.Id4Vb5/native.log SHA25649fdd7e0a1e461926b7d13edfa342319d278f28531fea2c8d86e36f29bffac92. No fix is stacked into #761.
