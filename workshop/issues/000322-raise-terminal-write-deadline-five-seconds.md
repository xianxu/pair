---
id: 000322
status: working
deps: []
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
started: 2026-09-24T15:54:11-07:00
flow: {kind: quick, provenance: inferred, spec: "141a9987", done: "cdbf434a"}
---

# Raise terminal write deadline to five seconds

## Problem

## Spec

The shared `terminal.WriteTimeout` used by queued child-input writes and
presenter writes increases from 2 seconds to 5 seconds. No retry, queue-size,
or cancellation behavior changes; a write that still exceeds the deadline
continues to report its accepted prefix and timeout error. This is a bounded
shutdown/terminal-IO adjustment (ARCH-CONSTRAINTS); the existing single writer
remains the authority for delivery (ARCH-DRY).

## Done when

- `WriteTimeout` is five seconds and its focused regression test passes.
- The terminal package tests pass.
- The timeout remains shared by both transport and presenter call sites.

## Plan

- [x] Add the focused timeout contract test, then change the shared constant.
- [x] Run terminal tests and record the result.

## Log

### 2026-09-24

- TDD: the new timeout contract test failed at the old 2s value, then passed
  after changing the shared `WriteTimeout` constant to 5s. `go test
  ./cmd/internal/terminal -count=1` passes.
