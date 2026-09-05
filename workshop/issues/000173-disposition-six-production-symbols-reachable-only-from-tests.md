---
id: 000173
status: open
deps: [pair#170]
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
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

## Spec

For each of the six, choose and record:

1. **Delete it**, with its tests, if the behaviour it covers is genuinely gone.
2. **Wire it up**, if production *should* be calling it and the absence is a
   bug. `ReconcileActiveParks` is the one to look at hardest: an explicit
   reconciliation pass with no caller may be a missing call rather than dead
   code, and the M4 review's own lesson applies — a deleted guarantee leaves no
   compile error.
3. **Keep it as a seam**, if a test genuinely needs the entry point; then move
   it out of the deferral group into the documented-seam group of the allowlist
   with its reason.

The allowlist entries move from "pair#173: …" to a real reason, or the symbol
leaves the tree.

## Done when

- Each of the six has a recorded disposition.
- `deadSymbolAllowlist` contains no entry whose reason is a deferral.

## Plan

- [ ] `ReconcileActiveParks` first — decide whether the missing caller is the bug.
- [ ] Work the remaining five.
- [ ] Update the allowlist and its comment.

## Log

### 2026-09-02

Filed from `pair#170` M4's close. The boundary review found that the allowlist
cited `pair#173` while `workshop/issues/` stopped at `000172` — six permanent
exemptions pointing at nothing. Filing it is the fix; the citation is now real.
