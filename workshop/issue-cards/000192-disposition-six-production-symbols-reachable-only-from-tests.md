---
id: 000192
status: open
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Disposition six production symbols reachable only from tests

## Problem

`pair#170` M4 added `TestNoProductionSymbolIsReferencedOnlyByTests`
(`cmd/internal/artifactpath/deadsymbols_test.go`), which fails when a
`couchcore` production declaration has no reference outside its own definition
in non-test code. It found twelve. Two were deleted in that milestone
(`StartArgs.AgentStack`, `ThreadStore.DeleteUnstartedThread`), four are
legitimate seams or fakes, and **six are genuinely unreferenced by production**:

| symbol | why it is reachable only from tests |
|---|---|
| `Couch.PublishDescription` | the `publish-description` op calls `ApplyThreadMetadata` directly (`operationdispatch.go`) |
| `Couch.ReconcileActiveParks` | an explicit reconciliation pass nothing invokes |
| `OperationNames` | the CLI resolves operations by name without it |
| `Registry.Unregister` | registry mutation with no caller |
| `ResumeDiagnosticOf` | diagnostic accessor with no caller |
| `ClassifyThreadReferenceFields` | reached only through `MatchThreadReferenceFields` |

They are parked in that test's allowlist against this issue. That is deliberate:
each one has tests, so deleting it means deleting coverage, which is a judgement
per symbol rather than something to fold into a deletion sweep already spanning
five subsystems.

The allowlist comment claims the debt is "countable". It is only countable if
this issue exists — the M4 boundary review caught that it did not, which is why
this file is here.
