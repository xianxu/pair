---
id: '000248'
status: done
started: 2026-09-14T10:30:41-07:00
created: 2026-09-13
updated: 2026-09-14
estimate_hours: 1.08
actual_hours: 7.21
---

# Couch switcher blocks warm reattach without a native binding

## Problem

Couch's switcher rejects a warm reattachment to an existing live Zellij session
when Pair has not established a native-agent session binding. It labels the row
`binding lost — repairable`; selecting it reports `its resume binding was lost;
the session may still be running (pair#168)`.

Observed 2026-09-13 for Tools, address
`434128d5ad68b26e/couch-253f4266b649cb01`. Read-only investigation around 16:38
America/Los_Angeles confirmed:

- Zellij session `📁tools-couch-3` was alive; `action list-clients` returned
  only the header (zero attached clients).
- Pair wrapper PID 69567 and Codex launcher PID 69577 were alive, started 16:03:36.
- The thread record (revision 5) retained its Codex launch profile and working
  path but had no incarnations or VerifiedPark.
- Its Pair ledger contained a launch and no native binding; rendered scrollback
  showed Codex's initial prompt. Missing binding is not evidence of process death.
- The operator then attempted reattachment in the switcher and supplied a
  screenshot of the refusal above.

Source inspection: `couchcore/actionableinventory.go:detachedResumeProofMatches`
requires `observation.NativeID != ""`. Failed proof with a detached observation
projects to ThreadUnusable/ReasonBindingLost. In contrast, `DecideResume` in
`couchcore/resume.go` allows warm resume without a native binding, with explicit
coverage in `warmresume_test.go`. The UI rejects the thread before that path
can run. These files must be rechecked at implementation time for concurrent work.
