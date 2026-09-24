---
id: 000312
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T19:34:09-07:00
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

- [ ] Reproduce raw output through the production console dispatcher and OSProvisionIO using a short shell command; retain the existing CLI progress test as the non-console oracle.
- [ ] Route console progress to io.Discard while keeping stderr for non-console callers; run regression and affected package tests/race checks.
- [ ] Update atlas/project smoke-test evidence, build Couch, pass SDLC close review and publish.


## Log

### 2026-09-23

Root cause traced from `runTypedOperationWithConsole` assigning stderr, through
`Couch.WorkspaceProgress`, to `OSProvisionIO` forwarding both setup streams.
The screenshot is operator live evidence; regression uses an isolated subprocess.
