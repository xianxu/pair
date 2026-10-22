---
id: 000803
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9c8e6b7a4b261b4a3dcf0dac5aa80133dbec59c7' # card fields mirrored from issue-cards; edit via sdlc
---

# Zellij empty-reply workaround covers only the launcher; layout queries fail on the same empty list-panes reply (blocks pair#743 CI)

## Problem

## Spec

pair#743's hosted conformance run 39759978845 failed `TestTerminalRecoveryZellijConformance`
(terminal_recovery_live_test.go:146): native `zellij action list-panes --json --command --state --geometry` returned
exit 0 with empty stdout and stderr after an earlier valid 4996-byte answer, and the strict parser refused it. pair:3
traced production: `layoutcmd.OSRuntime.execZellijContext` runs once and `confirmWorkbenchPanesContext` uses
`zellijreply.Confirm` (exactly two valid observations, no retry); `ParsePanes` rejects empty JSON. Only
`launcher.ZellijSource.runContext` has the bounded 3-attempt empty or crossed-reply workaround (pair#748), and the
atlas explicitly excludes other consumers. So the layout path fails in production on the same Zellij behaviour
(upstream zellij-org/zellij#5751 family; pair#775 evidence).

Fix: one shared read-compatibility seam (#748's bounded retry on an empty or crossed reply, unchanged bounds) used by
every Zellij query consumer, layout first, not a per-consumer copy and not a test-only retry. Update the atlas
exclusion.

## Done when

- The layout consumer retries an empty successful reply within the existing bound, through the shared seam (test with
  a fake that returns empty once).
- The live conformance test passes without any test-only retry; pair#743's hosted run goes green.

## Plan

- [ ]

## Log

### 2026-10-22
