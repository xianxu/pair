---
id: 000068
status: working
deps: []
github_issue:
created: 2026-06-23
updated: 2026-06-23
estimate_hours: 1.5
started: 2026-06-23T09:31:06-07:00
---

# Trace zellij action boundary

## Problem

Codex pair sessions can reproduce a zellij logout where the final visible line
is `Bye from Zellij!` and the zellij server log reports more than 1000
consecutive unknown/empty client messages. The existing logs identify zellij's
disconnect guard but do not show which pair-originated zellij action happened
immediately before the failure.

## Spec

Add opt-out-safe diagnostic tracing around the pair-managed zellij action
boundary. The trace must focus on the user prompt path from the draft nvim pane
to the agent pane, because that is the known 100% repro surface. Records should
be JSONL under `$PAIR_DATA_DIR`, scoped by `$PAIR_TAG`, and should avoid writing
full prompt bodies; body-bearing actions record length and a short hash instead.

Initial instrumentation covers:

- draft `send_to_agent`: relative focus, `write-chars`, submit, and focus-back;
- review `pair_poke`: direct `--pane-id` write and submit, for the parallel
  absolute-pane path.

The wrapper records timestamp, component, label, argv summary, duration,
`vim.v.shell_error`, stdout/stderr byte counts when observable, and redacted
argument metadata.

## Done when

- A clean repro session leaves `$PAIR_DATA_DIR/zellij-actions-<tag>.jsonl`
  showing the final pair-originated zellij actions before zellij disconnects.
- Full prompt text is not written to the trace.
- Headless tests cover redaction/logging and the review poke integration.

## Estimate

```estimate
diagnostic-instrumentation design=0.2 impl=0.8
focused-headless-tests design=0.1 impl=0.4
```

## Plan

- [ ] Add a reusable Lua zellij trace wrapper with JSONL output and body redaction.
- [ ] Route draft prompt send and review poke zellij calls through the wrapper.
- [ ] Add/extend headless tests that fail without trace records and pass with them.
- [ ] Run the focused tests and record the repro command/log location.

## Log

### 2026-06-23

- Created as follow-up to #67 after log audit showed zellij disconnecting the
  client for repeated unknown/empty IPC messages. `sdlc claim --issue 68`
  changed status to working locally, but could not broadcast because no `main`
  worktree is currently available.
