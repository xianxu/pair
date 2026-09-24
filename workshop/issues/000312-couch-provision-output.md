---
id: 000312
status: codecomplete
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T19:34:09-07:00
flow: {kind: quick, provenance: inferred, spec: "7e70af2f", done: "2460ade9"}
actual_hours: 0.11
---

# Keep workspace setup output out of the Couch terminal UI

## Problem

First-time pair:1 setup printed Homebrew/weave output over the active Couch switcher. The CLI assigns raw stderr to WorkspaceProgress even when a console owns the terminal.

## Spec

When a Couch console exists, discard raw provisioning progress at the CLI wiring
boundary. Existing operation progress/notices remain the UI; OSProvisionIO still
captures its bounded diagnostic tail and returns failures through normal error
handling. Non-console provisioning keeps streaming diagnostics to stderr and
JSON/results to stdout. Apply the policy to the retained Couch instance so later
create/open/fresh/resume operations use it too. No Homebrew/setup policy changes.
ARCH-DRY/PURPOSE: fix the shared output owner, reusing existing command capture
and UI status/error paths. No new log files, state, timers or goroutines.

## Done when

- Console-bound setup stdout/stderr, including control sequences, cannot write raw terminal diagnostics.
- Setup failures retain diagnostic context for normal Couch error reporting.
- Non-console provisioning still streams diagnostics and produces clean result output.


## Plan

- [x] Reproduce raw output through the production console dispatcher and OSProvisionIO using a short shell command; retain the existing CLI progress test as the non-console oracle.
- [x] Route console progress to io.Discard while keeping stderr for non-console callers; run regression and affected package tests/race checks.
- [x] Update atlas/project smoke-test evidence, build Couch, pass SDLC close review and publish.


## Log

### 2026-09-23
- 2026-09-23: closed — Reproduced raw setup stdout/stderr and screen-clear bytes bypassing the console before fix. Complete couchcmd suite passed; affected CLI/provision subprocess race tests passed; make pair bin/couch and git diff --check passed. Regression proves successful setup emits no raw terminal output, failures retain diagnostic context, and non-console CLI keeps stderr progress with clean JSON stdout.; review verdict: SHIP

Root cause traced from `runTypedOperationWithConsole` assigning stderr, through
`Couch.WorkspaceProgress`, to `OSProvisionIO` forwarding both setup streams.
The screenshot is operator live evidence; regression uses an isolated subprocess.

### 2026-09-23 — fix verified

Regression failed before the fix with literal screen-clear bytes and both setup
streams on stderr. After routing console progress to io.Discard, the complete
couchcmd suite passed (19.963s), affected CLI/provision subprocess race tests
passed (3.108s/7.558s), and `make pair bin/couch` plus `git diff --check` passed.
The failure case retains its diagnostic tail, and the existing non-console CLI
test still proves streamed stderr with clean JSON stdout. No live thread was
restarted or parked. Acceptance review and publication follow.

### 2026-09-23 — accepted

SDLC close returned SHIP with no findings; measured actual 0.11h. Reviewer
confirmed focused provisioning/race tests and the complete couchcmd suite.
Its broader couchcore runs were interrupted and remain inconclusive; no full-core
pass is claimed for this fix. Existing terminal-output ownership guidance in
workshop/lessons.md applies; no duplicate rule added.
