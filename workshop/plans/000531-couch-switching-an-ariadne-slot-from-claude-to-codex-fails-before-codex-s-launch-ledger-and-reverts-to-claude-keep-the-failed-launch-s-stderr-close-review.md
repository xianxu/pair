# Boundary Review — pair#531 (whole-issue close)

| field | value |
|-------|-------|
| issue | 531 — couch: switching an ariadne slot from Claude to Codex fails before Codex's launch ledger and reverts to Claude; keep the failed launch's stderr |
| repo | pair |
| issue file | workshop/issues/000531-couch-switching-an-ariadne-slot-from-claude-to-codex-fails-before-codex-s-launch-ledger-and-reverts-to-claude-keep-the-failed-launch-s-stderr.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1f57a3129e2f44c4abaebe3be2dc72ab6adc66d0..8d44efaf3b4588c3f61cd17fb7724c843eac1110 |
| command | sdlc close --issue 531 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T13:45:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The main capture, retention, and inspection paths are well separated, and the targeted core race tests passed. However, a spawned helper can still lose its diagnostics when blocked-start setup fails. Immediate errors also omit the required attempt identity, and the completion criteria omit the authorized installation changes.

Done-when: stale — it covers diagnostic preservation but not the added install-link behavior or the revised one-line recovery contract.

1. **Strengths**
   - Diagnostics remain independent of thread rollback, with bounded retention and atomic publication.
   - Pre/post-cleanup exit observations remain distinct; the recorder preserves `errors.Is/As`.
   - Real subprocess tests cover stderr forwarding, invalid bytes, PTY labeling, blocked-wrapper forwarding, and descendant-held descriptors.
   - README and atlas updates cover the new surface and installation compatibility implications.

2. **Critical findings**
   - **Spawned-helper evidence can disappear:** `cmd/internal/couchcore/launch_existing.go:183` treats every `StartBlocked` error as an unstarted helper. But `launchhelper.go:57–69` can return an error after spawning and cleaning up a helper—for example, `Setpriority` returns `ESRCH` after an early helper exit. The resulting record incorrectly says `HelperStarted=false`, with no output or exit evidence. Preserve evidence across this runner boundary and test this failure ordering deterministically. **ARCH-PURPOSE, ARCH-ORDER.**

3. **Important findings**
   - **Missing diagnostic correlation:** `cmd/internal/couchcore/launchfailure_capture.go:21–43,94–98` includes neither nonce nor diagnostic location in immediate errors. A persistence failure such as disk-full or lock timeout leaves only an inspection command that may display older failures. Include bounded attempt identity and contextualize persistence errors with the store path; retain error unwrapping.
   - **Incomplete completion criteria:** issue lines 37–43 omit the installation and recovery-message requirements added at line 77. Add acceptance criteria for both.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Targeted `couchcore` race tests passed: **15.825 s**.
   - Installer `--links-only` regression passed; pinned-range `git diff --check` passed.
   - `couchcmd` tests could not compile because generated `assets/runtime/files` was absent. Consequently, rendering tests were inspected but not independently executed.
   - No prior findings required disposition. The spawned-helper setup-error path needs a regression that fails without its correction.

6. **Architectural notes**
   - **ARCH-DRY — pass:** reuses existing output rings, sanitization, and atomic-write primitives.
   - **ARCH-PURE — pass:** retention transformations are separate from storage and process IO; listed entities exist at their documented locations.
   - **ARCH-PURPOSE — flag:** the runner setup-error gap leaves part of the promised failure class uncovered.
   - **ARCH-MOCK — pass:** production and fake runners share the boundary; real subprocess conformance tests exist.
   - **ARCH-CONSTRAINTS — pass:** writer-side bounds, bounded lock acquisition, and stderr-drain limits are implemented.
   - **ARCH-SECURE — pass:** bounded decoding, unsafe-file rejection, private file creation, and inspection sanitization are present.
   - **ARCH-ORDER — flag:** a failed setup operation is incorrectly treated as evidence that no helper started.
   - **ARCH-FUNERAL — pass:** history eviction and interrupted staging-file reclamation bound durable residue.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry covering evidence transfer from helpers that fail inside `StartBlocked`, including deterministic ordering coverage.
   - Clarify immediate attempt correlation and persistence-error location requirements.
   - Extend issue `## Done when` to cover the authorized installation and one-action recovery contracts.

```findings
findings:
  - id: new
    severity: Critical
    family: partial-start-evidence-preservation
    title: |
      Blocked-start setup errors discard evidence from an already-spawned helper
    detail: |
      cmd/internal/couchcore/launch_existing.go:183-184 records all StartBlocked errors with a nil handle, although launchhelper.go:57-69 can fail after spawning and cleaning up the helper. Preserve its output, observed exit, and started status across this boundary; add a deterministic regression for post-spawn setup failure. ARCH-PURPOSE and ARCH-ORDER.
  - id: new
    severity: Important
    family: diagnostic-attempt-correlation
    title: |
      Immediate failures omit attempt identity and persistence-error location
    detail: |
      cmd/internal/couchcore/launchfailure_capture.go:21-43 and 94-98 omit the nonce and diagnostic path required by the Spec and failure protocol. Include bounded attempt correlation and wrap persistence errors with the store path while preserving errors.Is/As; test both successful and failed persistence.
  - id: new
    severity: Important
    family: completion-criteria-scope-coverage
    title: |
      Done-when criteria omit the authorized installation and recovery-message changes
    detail: |
      The issue's Done when section at lines 37-43 remains diagnostic-only despite the scope additions at line 77. Add criteria for installed links following checkout rebuilds and immediate failures providing one sanitized explanation with exactly one recovery action.
```
