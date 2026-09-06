---
id: 000192
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

## Revisions

### 2026-09-05 — renumbered from `#173` to `#192`

Two files carried `id: 000173`: this one and its namesake. Both were created on
2026-09-02 by concurrent sessions, each allocating what it saw as the next free
ID — one through `issue-sync: update issues`, one through `sdlc issue new`, and
neither could see the other's reservation because the broadcast to `main` fails
in this checkout ("could not find a worktree on branch 'main'"). `sdlc claim
--issue 173` refused with "multiple issue files match", which is
the right failure but a blocking one.

**Why this file moved and the other did not.** The namesake owns commit messages
referencing `#173`, and AGENTS.md §12 makes `git log --grep
"^#173"` the way an agent finds an issue's work. A commit message
cannot be rewritten; prose references and code allowlists can. So the number
follows the immutable claim, and the editable references were updated instead.

