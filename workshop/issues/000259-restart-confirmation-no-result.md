---
id: 000259
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Alt+N restart confirmation has no visible result

## Problem

During #255 operator smoke, Alt+N opened a restart confirmation; pressing Y appeared to do nothing. The operator accepted #255 and explicitly deferred this report to fix-forward work.

## Spec

Identify the actual dialog and input route before changing behavior. Couch confirmation uses a Cancel/Relaunch selection; draft nvim confirmation uses Yes/No/Rename. The report did not yet identify which was shown. Trace key delivery, confirmation result and restart-command outcome; do not restart an operator session during diagnosis.

## Done when

- Confirmed restart performs the intended action, or displays an actionable failure rather than silently returning.
- A regression covers the demonstrated failing boundary.

## Plan

- [ ] Reproduce and identify the dialog, key encoding and restart result.
- [ ] Fix the demonstrated input or lifecycle boundary; cover success and failure.
- [ ] Validate with an isolated session and obtain operator confirmation.

## Log

### 2026-09-15

Initial read-only finding: nvim/init.lua pair_confirm_restart_impl calls vim.fn.system(argv) and discards output/status. This is a possible silent-error path, not an established cause. Couch reduceConfirmationKey treats letters as filtering, not Yes shortcuts. No source changes or operator restarts were made.
