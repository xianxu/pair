---
id: 000602
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Tests build the pair binary per test (11 tests): build once per suite and share it; guard against new per-test builds

## Problem

11 tests in pair run a full `go build` of `pair-go` into their own temp directory on every run, to exercise the
real binary as a subprocess (end-to-end fidelity). Each one links a fresh ~20 MB binary plus intermediates, per test,
per suite, per worktree; on 10-15 the suite's `link` processes wrote ~8 MB/s and `couchcmd.test` ~30 MB/s while
#571's suite ran (ops `tools/disk-writes`). Files: cmd/couch/main_test.go cmd/internal/artifactpath/cross_scope_integration_test.go cmd/internal/couchcmd/continuation_acceptance_test.go cmd/internal/couchcmd/switchagent_acceptance_test.go cmd/internal/couchcore/startup_detach_timing_live_test.go cmd/internal/hoprttcmd/hoprtt_test.go cmd/internal/launcher/agent_command_acceptance_test.go cmd/internal/launcher/couch_launch_prompt_acceptance_test.go cmd/internal/launcher/selected_root_test.go cmd/internal/layoutcmd/fullscreen_chord_live_test.go cmd/pair-go/helper_equivalence_test.go 

Part of ops project test-economy (north star: one MacBook, 10 workers; write and time budgets).
