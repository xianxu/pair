# Boundary Review — pair#366 (whole-issue close)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0c1e56689469666b1aaaac708538cbf95f06f1d..603bb5715a0057ee16cc383b11f3d7292b152558 |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-01T23:21:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The singleton lease and selected-root wiring are well separated, and existing focused tests pass. Three independently reproduced adoption bugs block shipping: ignored slot corruption, ignored surviving incarnations in excluded stores, and publication of selections that subsequent reads reject.

1. **Strengths**
   - Host ownership precedes runtime construction; teardown releases the store lease before the host lease.
   - Selected roots reach messaging, artifact readers, tracing, and both child-launch paths.
   - Tests exercise contention, crash/exec release, stale receipts, publication failure, and lost-acknowledgment retry.
   - README and atlas describe migration, isolation, and compatibility limits.

2. **Critical findings**

   The three findings below each have a failing scratch regression against the pinned HEAD.

```findings
findings:
  - id: new
    severity: Critical
    family: adoption-evidence-completeness
    title: |
      Adoption ignores unreadable numbered-slot state.
    detail: |
      cmd/internal/couchsingleton/inspect.go:210 checks only PreviewSnapshot's returned error and snapshot.Unreadable. Numbered-slot failures are instead carried in snapshot.Slots[].Err (cmd/internal/couchcore/threadstore_snapshot.go:31–53). A corrupt slot thread.json therefore produces READY with no blockers; TestReviewCorruptSlotBlocksAdoption reproduces this. Inspect every slot observation and refuse unresolved state. Include slot-local evidence in the digest and publication revalidation, since hashTree currently covers only the global namespace. ARCH-SECURE, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: migration-liveness-evidence
    title: |
      Exclusion admits stores with surviving live incarnations.
    detail: |
      cmd/internal/couchsingleton/inspect.go:191 observes only the supervisor lease and discards snapshot.Records; model.go:95 then accepts exclusion of a populated store. With a free supervisor lease and a recorded incarnation whose PID/start identity is still live, preview returns READY. TestReviewExcludedLiveWrapperBlocksAdoption reproduces this. This contradicts the issue's requirement to report surviving wrappers in other namespaces as migration conflicts. Inspect incarnation ownership through the existing process/lifecycle seams, preserve unknown outcomes, and block exclusion until absence is established. ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: persisted-writer-reader-contract
    title: |
      Successful adoption can publish an unreadable selection.
    detail: |
      cmd/internal/couchsingleton/manager.go:204–212 publishes serialized Selection without enforcing Read's 65,536-byte limit at line 37. Requests permit up to 4096 exclusions. A valid 600-exclusion request successfully published 139,628 bytes, after which Read refused with "file exceeds 65536 bytes"; TestReviewSelectionWriterReadLimit reproduces this. Share the size contract between writer and reader, reject oversized selections before publication, and test the boundary plus successful round trips. ARCH-CONSTRAINTS, ARCH-SECURE.
```

3. **Important findings:** None additional.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: full `couchsingleton` and `couchidentity` suites under `-race`; focused singleton/runtime/adoption tests; focused supervisor-observation and concurrent-FakeGit tests; pinned-range `git diff --check`.
   - All three new scratch regressions fail as described. They used a temporary Go overlay; repository files were unchanged.
   - Full repository tests were not rerun. Existing migration fixtures do not cover these failure classes.

6. **Architectural notes**
   - **ARCH-DRY — pass:** existing leases, identity decoders, and atomic publication are reused.
   - **ARCH-PURE — pass:** adoption policy is separated from filesystem/process inspection.
   - **ARCH-PURPOSE — flag:** incomplete migration evidence violates the preservation/refusal contract.
   - **ARCH-MOCK — pass:** inspected tests use injected roots, process seams, controlled filesystems, and subprocesses.
   - **ARCH-CONSTRAINTS — flag:** writer and reader size envelopes disagree.
   - **ARCH-SECURE — flag:** slot-local corruption is silently omitted from admission evidence.
   - **ARCH-ORDER — flag:** absence of a supervisor lease is insufficient evidence that another inventory has no surviving actors.
   - **ARCH-FUNERAL — pass:** new authority artifacts use fixed paths; lease lifetime and staging are bounded.

7. **Plan revision recommendations**

   Append a timestamped `## Revisions` entry covering complete slot evidence and revalidation, surviving-incarnation admission rules, and the shared selection-size limit. Add the three regressions to the verification contract before marking those implementation items complete again.
