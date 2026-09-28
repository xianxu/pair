# Boundary Review — pair#337 (whole-issue close)

| field | value |
|-------|-------|
| issue | 337 — Draft bang line tags the couch thread description |
| repo | pair |
| issue file | workshop/issues/000337-draft-bang-line-sets-couch-description.md |
| boundary | whole-issue close |
| milestone | — |
| window | d4e0538422eaa670a36c595f56994b764000e908..391819d48cc983a89f96f1241957c81e2d6f85b1 |
| command | sdlc close --issue 337 |
| reviewer | codex |
| timestamp | 2026-09-28T11:34:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The core implementation matches the issue: parsing is pure, both operator-send paths share the wrapper, authored text remains in the log, and publishing uses Couch’s existing API. Focused tests pass. Two Important gaps remain: README documentation and regression coverage for the explicitly promised failure behavior.

1. **Strengths**

   - `nvim/bang_tag.lua:10` isolates parsing without IO.
   - `nvim/init.lua:803` preserves the submission transaction and publishes only after confirmed dispatch.
   - Integration tests exercise real process spawning in both Couch and standalone modes.
   - Atlas updates describe the parser, submission flow, and existing persistence owner.

2. **Critical findings**

   None.

3. **Important findings**

   - **Missing user-facing documentation — `README.md:213`, `nvim/init.lua:803`.** The new draft syntax is absent from README. Document the complete surface: single-line `!` stripping, description publication inside Couch, standalone behavior, bare/multiline handling, and intentional loss of draft bash-mode syntax. This is the sole new user-facing syntax family in this window.

   - **Failure contracts lack regression coverage — `tests/bang-tag-nvim-test.sh:12`, `nvim/bang_tag_integration_test.lua:15`.** The Couch stub always succeeds immediately, and every bang-test dispatch succeeds. Existing submission failure tests use ordinary `BODY` prompts, so they cannot detect erroneous description publication. Cover the whole failure family: unavailable Couch executable, nonzero publisher exit, slow publisher, failed dispatch without publication, and successful retry with exactly one publication. Assert that publisher failures preserve successful prompt delivery and that a blocked publisher does not delay submission.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Bang parser tests.
   - Couch/standalone bang integration tests.
   - Submission transaction failure matrix.
   - Workbench route test.
   - Focused Couch CLI, metadata, and persistence tests.

   Full and race suites were not rerun. Their recorded failures remain outside this verification. Repository working tree is clean.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Both operator-send paths reuse one parser/wrapper; Couch retains description ownership.
   - **ARCH-PURE: pass.** Deterministic parsing is separate from thin process-launch glue.
   - **ARCH-PURPOSE: pass.** The implementation delivers tagging through the existing persistence path; no purpose-critical consumer is deferred.

7. **Plan revision recommendations**

   Add regression steps for the failure family above and README documentation. Record any plan expansion under `## Revisions`.

```findings
findings:
  - id: new
    severity: Important
    family: user-facing-surface-documentation
    title: |
      README omits the new draft bang syntax.
    detail: |
      README.md:213 documents draft comments but has no bang-line documentation, and README is unchanged in the pinned range. Document the single new syntax family introduced at nvim/init.lua:803: single-line stripping and Couch tagging, standalone behavior, bare/multiline handling, and the intentional draft bash-mode compatibility change.
  - id: new
    severity: Important
    family: side-effect-failure-contract-coverage
    title: |
      Bang integration tests do not exercise publication or dispatch failures.
    detail: |
      tests/bang-tag-nvim-test.sh:12 always supplies an immediately successful Couch stub, and nvim/bang_tag_integration_test.lua:15 always succeeds at dispatch. Enumerate and test unavailable executable, nonzero publisher exit, slow publisher, failed dispatch with no publication, and successful retry with exactly one publication. Existing BODY-only transaction tests cannot verify these bang-specific guarantees.
```
