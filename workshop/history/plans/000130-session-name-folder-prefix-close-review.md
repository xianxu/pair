# Boundary Review — pair#130 (whole-issue close)

| field | value |
|---|---|
| issue | 130 — session names: folder prefix, repo token, no redundant tag |
| repo | pair |
| issue file | workshop/issues/000130-session-name-folder-prefix.md |
| boundary | whole-issue close |
| window | 88afa44c677214089a2d95c47c37ad20dc430a74..HEAD |
| reviewer | codex |
| timestamp | 2026-07-29T13:06:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Verdict

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The review found no Critical correctness bugs. It required ship-before-close
fixes for README coverage of the new public session-name scheme, pasted-name
tests for resume/rename paths, a length-aware zellij-name fake in launcher tests,
and plan wording that still contradicted the implemented detached-session
preservation rule.

## Findings

### Important

1. **README update missing for user-visible session names.**
   The diff changed zellij/cmux/list names and allowed pasted public names, but
   README did not document the `📁repo[-residual]` public name, tag vs public
   name, or pasted-name support.

2. **Length-aware fake missing for zellij name budget.**
   `fakeRuntime.ProbeSessionName` returned a constant error, so the create
   prompt's lazy budget/refuse path was not tested through the runtime seam.

3. **Pasted `📁...` resume/rename paths lacked direct tests.**
   The review asked for indexed and unindexed public-name resume coverage,
   rename-old resolution through the ledger, and rename-new rejection as a public
   session name rather than a bare tag.

### Minor

1. `rename.go` had adjacent duplicate comments for `validateRenameTags`.
2. The checked issue plan still said detached legacy sessions are reclaimed,
   while the implementation preserves them as resumable work.

## Resolution

- Updated README with the public `📁{repo}[-{residual tag tokens}]` scheme, tag
  vs public-name distinction, and pasted public-name resume/rename behavior.
- Added launcher tests for public-name resume resolution and refusal, prompt
  accept/refuse under a fake zellij byte budget, and public-name rename paths.
- Made `fakeRuntime.ProbeSessionName` model a per-name byte ceiling while
  preserving the existing constant-error path.
- Folded the duplicate `validateRenameTags` comment.
- Updated #130 Spec/Plan/Revisions so detached legacy sessions are explicitly
  preserved and only `SessionExited` superseded records are reclaimed.
- Added a lesson covering README plus pasted-name tests for user-visible identity
  changes.

## Verification After Fixes

- `go test ./cmd/internal/launcher -run 'TestRunLaunch(ResumePublicSessionNameResolvesThroughIndex|ResumeUnindexedPublicSessionNameRefuses|PromptRefusesNameOverDiscoveredBudget|PromptAcceptsNameUnderDiscoveredBudget)|TestParseLaunchArgsResumeKeepsPublicSessionNameForLedgerResolution|TestValidateRenameTagsRejectsPublicSessionNameAsNewTag|TestRunRenameOldPublicSessionNameResolvesThroughIndex|TestRunRenameUnindexedOldPublicSessionNameRefuses' -count=1`
- `go test ./cmd/internal/launcher -count=1`
- `go test ./... -count=1`
- `tests/pair-rename.sh`
- `git diff --check`
