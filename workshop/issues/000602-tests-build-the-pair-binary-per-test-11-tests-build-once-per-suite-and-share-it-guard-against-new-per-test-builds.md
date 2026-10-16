---
id: 000602
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '893beb75cc7146dd7cf4fc2c1fc3b191c5b6edd1' # card fields mirrored from issue-cards; edit via sdlc
---

# Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds

## Problem

11 tests in pair run a full `go build` of `pair-go` into their own temp directory on every run, to exercise the
real binary as a subprocess (end-to-end fidelity). Each one links a fresh ~20 MB binary plus intermediates, per test,
per suite, per worktree; on 10-15 the suite's `link` processes wrote ~8 MB/s and `couchcmd.test` ~30 MB/s while
#571's suite ran (ops `tools/disk-writes`). Files: cmd/couch/main_test.go cmd/internal/artifactpath/cross_scope_integration_test.go cmd/internal/couchcmd/continuation_acceptance_test.go cmd/internal/couchcmd/switchagent_acceptance_test.go cmd/internal/couchcore/startup_detach_timing_live_test.go cmd/internal/hoprttcmd/hoprtt_test.go cmd/internal/launcher/agent_command_acceptance_test.go cmd/internal/launcher/couch_launch_prompt_acceptance_test.go cmd/internal/launcher/selected_root_test.go cmd/internal/layoutcmd/fullscreen_chord_live_test.go cmd/pair-go/helper_equivalence_test.go 

Part of ops project test-economy (north star: one MacBook, 10 workers; write and time budgets).

## Spec

- The suite builds `pair` once (`make test` already builds `bin/`), and passes its path to tests in one env
  variable (e.g. `PAIR_TEST_BINARY`). A test needing the real program uses it; if unset (a targeted `go test`), a
  single shared build per test process (sync.Once into one temp dir), never one per test.
- Where a test only needs *a* subprocess, use the test-binary helper-process pattern (as the continuation test's
  `couch` shim already does), with no build at all.
- A guard test (or lint) refuses new `exec.Command("go", "build"` in `_test.go` outside the shared helper.

## Done when

- No test builds `pair-go` itself; the guard enforces it; a full suite's `link` writes drop to the per-package
  test binaries only (measured with ops `tools/disk-writes` before and after).

## Plan

- [ ]

## Log

### 2026-10-15
