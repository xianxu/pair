---
id: 000264
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Fix Codex changelog refresh failure

## Problem

The change-log view does not work reliably for Codex sessions. On refresh, the
viewer displayed the visible error:

`change log refresh failed: model: codex exec failed: exit status 1: OpenAI Codex v0.154.0`

The viewer should remain useful even when the model command fails, but the
current path surfaces only the failure and does not produce a changelog.

## Spec

Trace the Codex-specific changelog refresh path from the viewer through the
model runner, including the exact argv, environment, working directory,
authentication/session state, stderr and exit status. Compare it with a working
Claude or other model path. Make the failure actionable and preserve the
existing changelog when refresh fails; do not silently replace a prior log with
empty output. Keep model selection and command construction in their shared
seam rather than adding a Codex-only workaround in the Neovim viewer.

## Done when

- A Codex session can open and refresh its changelog successfully under the
  supported launch environment.
- A model failure leaves the previous changelog visible and shows an actionable
  error with enough diagnostic context to identify the failed boundary.
- Tests cover Codex command construction/result handling, nonzero exit status,
  preserved prior log content, and the viewer refresh behavior.
- A live or equivalent operator smoke check confirms the Codex changelog view
  works.

## Plan

- [ ] Reproduce the failure with the captured Codex session and record the
  model command, environment, stderr and exit status.
- [ ] Compare the Codex and working-model refresh paths and fix the shared model
  runner or command contract at the responsible boundary.
- [ ] Add regression coverage for success, nonzero exit and prior-log
  preservation.
- [ ] Run focused changelog tests and verify the view with Codex.

## Log

### 2026-09-15

Filed from operator screenshot. Codex changelog refresh showed `codex exec`
exiting with status 1 under `OpenAI Codex v0.154.0`; the viewer surfaced the
error instead of producing the change log.
