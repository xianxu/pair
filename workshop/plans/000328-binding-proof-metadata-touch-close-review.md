# Boundary Review — pair#328 (whole-issue close)

| field | value |
|-------|-------|
| issue | 328 — Relaunch rejects a binding after metadata-only change |
| repo | pair |
| issue file | workshop/issues/000328-binding-proof-metadata-touch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 76a8d0199d0d7a81689aadfe86c4ae9706bca657..ca6cac2a88c6211e9c7f205391fb09273259de40 |
| command | sdlc close --issue 328 |
| reviewer | claude |
| timestamp | 2026-09-25T10:22:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix does what the issue needs. `proofAllowsFullRevalidation` now allows the from-byte-zero re-read whenever the file is the same one, is not smaller, and has no generation token. The existing identity check after the re-read still decides whether the root validates, so metadata can no longer revoke a binding on its own. The new `binding_stale` diagnostic goes into the structured query result, and I found no UI or CLI code that prints query diagnostics, so the UI text is unchanged as the Spec requires.

I checked this against the code, not the commit messages:
- The package tests pass on HEAD.
- Forcing the fallback off in a scratch copy makes the metadata-touch test and the existing growth test fail.

Two gaps before close:
- **Untested boundary:** nothing tests the "not smaller" cut-off.
- **Unevidenced Done-when clause:** the operator's live confirmation isn't recorded in the Log.

**Strengths**
- `query.go:225-266`: a small, targeted change. It keeps the stable-file-ID and generation-token rules exactly as before, so the one-way safety property (content decides) is preserved.
- `query_test.go:139`: the same-size rewrite test uses a native ID of the same length, so the file size really is identical. It exercises the case the relaxed condition opens up, not a growth case.
- `query.go:145-148`: the diagnostic reuses the existing `diagnosticWithSource` helper and code (ARCH-DRY pass), and the status stays provisional.
- The atlas entry (`atlas/session-identity.md:104-109`) matches the code. It correctly limits the claim to filesystems with no generation token.

**Critical:** none.

**Important**
1. **The "not smaller" boundary is untested** (`query.go:257`). In a scratch copy I removed `|| observation.Entry.Size < artifact.Size`, and no query test failed. The only failures were two concept-registry tests that also fail in a git-archive copy for unrelated reasons. A truncated transcript whose first line still carries the right `session_meta` would then be re-read and established. That breaks the Spec's "not smaller" rule, and nothing would catch it.
   - Fix: add `TestQuerySessionRefusesShrunkTranscriptWithoutGenerationToken`. Use the same stable file ID with a smaller size, content that is a valid shorter prefix, and assert provisional plus `binding_stale`.
   - The other guard clauses in that function are the same family and are also only indirectly covered: stable-file-ID mismatch, and a generation token that appears after the proof had none. Cover them in the same sweep.
2. **Done-when clause not evidenced:** "the operator confirms that alt+n relaunch works after relaunch → detach → reattach". The Log records the scratch-probe result but no operator confirmation, even though the Plan box "Verify live" is ticked. Get the operator's smoke-test result and log it before `sdlc close`.

**Minor**
- Nothing in Couch reads the new diagnostic today; the Log says so openly. The operator asked for "debugging log", and a diagnostic nobody reads only partly meets that. Consider filing the COUCH_TRACE wiring as a follow-up issue so it doesn't live only in a Log line.
- The comment at `query.go:230` names "claude" inside the agent-agnostic `sessioninventory` package. Wording like "a resuming agent" would avoid tying it to one agent.

**Test coverage notes**
- The metadata-touch test fails without the fix, as I confirmed with the scratch revert, so the Done-when clause "fails before the fix" holds.
- The rewrite test's `binding_stale` assertion passes with or without the fallback, because the new diagnostic is added on every validation error. That's fine: its "not established" assertion is the safety check that matters once the fallback is on.
- `sessionwatch/run.go:244` also calls `ValidateBindingProof` and gets the relaxed fallback. That's a benefit, but no watcher-level test covers it.

**Architecture**
- ARCH-DRY: pass. The helper is reused, and no duplicated block was added.
- ARCH-PURE: pass. The size and identity check is a pure function; IO stays in `ValidateTargetWork` behind the runtime interface.
- ARCH-PURPOSE: pass on the main purpose. A relaunch after a metadata touch now establishes. The "debugging log" half is only partly met (see Minor).

**Plan revision recommendations**
- In `## Log`, add the operator's live confirmation, or untick "Verify live" until it arrives.

```findings
findings:
  - id: new
    severity: Important
    family: fallback-guard-boundaries-untested
    title: |
      No test pins the not-smaller boundary of proofAllowsFullRevalidation
    detail: |
      Deleting the Size < artifact.Size guard (query.go:257) makes no query test fail, so a truncated transcript with a valid first line would be re-read and established. Add a shrink test that asserts provisional + binding_stale. Sweep the other guard clauses in that function too: stable-file-ID mismatch, and a generation token appearing where the proof had none.
  - id: new
    severity: Important
    family: done-when-clause-unevidenced
    title: |
      Operator confirmation of alt+n relaunch after detach/reattach not recorded
    detail: |
      Done-when requires the operator to confirm the live relaunch → detach → reattach flow. The Log records only the scratch probe, yet the Plan's Verify-live box is ticked. Get the confirmation and log it before close.
  - id: new
    severity: Minor
    family: diagnostic-without-sink
    title: |
      binding_stale proof-failure diagnostic has no reader yet
    detail: |
      The operator asked for a debugging log, but no Couch sink reads query diagnostics. File the COUCH_TRACE wiring as a tracked follow-up issue instead of leaving it as a Log note.
  - id: new
    severity: Minor
    family: agent-agnostic-wording
    title: |
      Comment in agent-agnostic sessioninventory names claude specifically
    detail: |
      query.go:230 says "a resuming claude bumps ctime"; "a resuming agent" would be clearer.
```

---

## Re-review — 2026-09-25T10:35:32-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 328 — Relaunch rejects a binding after metadata-only change |
| repo | pair |
| issue file | workshop/issues/000328-binding-proof-metadata-touch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 76a8d0199d0d7a81689aadfe86c4ae9706bca657..b28de7e1a289752ad9504bed6919c969c4d2b3fa |
| command | sdlc close --issue 328 |
| reviewer | codex |
| timestamp | 2026-09-25T10:35:32-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core fix is correct and focused, with package tests passing. Boundary review remains blocked because the required live operator confirmation is still absent, and the shrink-guard test does not explicitly assert `BindingProvisional`.

1. Strengths

- `query.go:225-267` correctly permits same-size metadata revalidation only for stable, non-smaller files without generation tokens.
- Same-size rewrite coverage verifies the content identity check rejects another conversation.
- `query.go:144-149` records `binding_stale` through the existing diagnostic helper.
- Follow-up issue #330 now tracks COUCH_TRACE diagnostic consumption.
- Atlas documentation was updated for the new behavior.

2. Critical findings

None.

3. Important findings

- BR-1 remains `not-addressed`: `query_test.go:161-192` covers shrink, stable-ID replacement, and generation appearance, but only asserts “not established” plus `binding_stale`; it does not assert exact `BindingProvisional` status. Keep the family sweep covering all three guards.
- BR-2 remains `not-addressed`: the pinned issue Log records only the scratch probe, not operator confirmation of relaunch → detach → reattach.

4. Minor findings

- `agent-agnostic-wording` repeat: `atlas/session-identity.md:106` still says “a resuming claude.” Use “a resuming agent” for the generic contract. The incident-specific test comment is appropriately concrete.

5. Test coverage notes

`go test ./cmd/internal/sessioninventory/...` passes. `make test` reached the Lua suite but failed at `nvim/scrollback_test.lua` because editor storage protection returned `operation not permitted`; therefore the full Done-when suite is not green in this environment.

6. Architectural notes

- ARCH-DRY: pass; existing diagnostic and validation helpers are reused.
- ARCH-PURE: pass; revalidation policy remains separated from runtime I/O.
- ARCH-PURPOSE: pass for the metadata/content behavior; live verification remains unevidenced.

7. Plan revision recommendations

Add a `## Revisions` entry documenting:

- the three guard cases and explicit `BindingProvisional` assertions;
- the operator-confirmed live relaunch flow;
- the wording correction in atlas;
- the existing COUCH_TRACE follow-up reference `#330`.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The new table covers shrink, stable-file-ID mismatch, and generation appearance, but does not explicitly assert BindingProvisional for the shrink case.
  - id: BR-2
    disposition: not-addressed
    note: |
      The pinned issue Log still records only the scratch probe, not operator confirmation of relaunch → detach → reattach.
  - id: BR-3
    disposition: addressed
    note: |
      Issue #330 is committed in the reviewed range and explicitly tracks carrying binding_stale diagnostics into COUCH_TRACE.
  - id: BR-4
    disposition: addressed
    note: |
      The original query.go comment now says “resuming agent”; the remaining atlas wording is a newly introduced instance.
findings:
  - id: new
    severity: Minor
    family: agent-agnostic-wording
    title: |
      Atlas contract still names one agent in generic revalidation prose
    detail: |
      This is the 2nd finding in family agent-agnostic-wording. The remaining changed instance is atlas/session-identity.md:106, which says “a resuming claude”; change it to “a resuming agent” so the generic inventory contract is not tied to one provider.
```

---

## Re-review — 2026-09-25T10:39:30-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 328 — Relaunch rejects a binding after metadata-only change |
| repo | pair |
| issue file | workshop/issues/000328-binding-proof-metadata-touch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 76a8d0199d0d7a81689aadfe86c4ae9706bca657..844dc32b2f4e8c4aa953f7a800842b9b11556c14 |
| command | sdlc close --issue 328 |
| reviewer | codex |
| timestamp | 2026-09-25T10:39:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation is focused and the session-inventory tests pass. Boundary review remains blocked because the required exact `make test` command fails at `nvim/scrollback_test.lua` with `operation not permitted`; this is not attributable to the diff, but the Done-when requirement is not verified.

Strengths:

1. Same-size metadata changes now trigger content revalidation while preserving stable-ID and generation guards.
2. Same-size rewrites to another conversation are rejected with `binding_stale`.
3. Guard coverage explicitly checks shrink, file replacement, and generation appearance.
4. Atlas documentation and the COUCH_TRACE follow-up are present.

Critical findings:

None.

Important findings:

- `nvim/scrollback_test.lua`: exact `make test` is not green; it fails with `pair: cannot protect editor storage: pair retention: operation not permitted`. This is the 2nd finding in family `done-when-clause-unevidenced`; the family instances are the prior live-smoke evidence gap (BR-2) and this full-suite verification gap. Re-run the exact command in a permitted environment and record the result, or document an approved, reproducible exception.

Minor findings:

None.

Test coverage notes:

- `go test ./cmd/internal/sessioninventory/...` passes.
- `git diff --check` passes.
- `make test` fails at the scrollback Lua test due to environment permissions.

Architectural notes:

- ARCH-DRY: pass; existing validation and diagnostic helpers are reused.
- ARCH-PURE: pass; revalidation policy remains separate from runtime I/O.
- ARCH-PURPOSE: pass for the metadata/content behavior; full Done-when verification remains incomplete.

Plan revision recommendations:

- Add the exact `make test` failure and its environment cause to `## Revisions`/`## Log`, then record a successful rerun before close.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The guard-family table now asserts BindingProvisional and binding_stale for shrink, stable-file-ID replacement, and generation appearance.
  - id: BR-2
    disposition: addressed
    note: |
      The pinned issue Log records operator confirmation of the relaunch → detach → reattach smoke flow.
  - id: BR-3
    disposition: addressed
    note: |
      Issue #330 tracks carrying binding_stale diagnostics into COUCH_TRACE.
  - id: BR-4
    disposition: addressed
    note: |
      The query comment now uses “resuming agent”.
  - id: BR-5
    disposition: addressed
    note: |
      Atlas wording now uses “resuming agent”.
findings:
  - id: new
    severity: Important
    family: done-when-clause-unevidenced
    title: |
      Exact make test verification is not green
    detail: |
      This is the 2nd finding in family done-when-clause-unevidenced. The exact required command fails at nvim/scrollback_test.lua because editor storage protection returns operation not permitted. Re-run successfully in a permitted environment and record the evidence before close.
```
