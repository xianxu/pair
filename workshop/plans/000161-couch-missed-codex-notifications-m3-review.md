# Boundary Review — 000161-couch-missed-codex-notifications#161 (milestone M3)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | dc43a9e8e67f940eee9dc6daeb789bcdf2b16392..e0eb862004afabeb29121a54617648494b7e3ed8 |
| command | sdlc milestone-close --issue 161 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-01T15:28:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3 establishes a credible transcript-authority path with strict envelope parsing, scanner-authorized binding, durable journal identities, current-launch filtering, and focused regressions. It cannot cross the boundary yet: journal filesystem IO and unbounded suffix processing run directly on the PTY master loop, contradicting the explicit non-blocking operating envelope. The plan’s Core concepts inventory also claims entities and statuses that do not match the tree, and the required live Codex lifecycle conformance was not extended.

```findings
findings:
  - id: new
    severity: Critical
    family: lifecycle-contract-coverage
    title: |
      Journal IO blocks the PTY master loop instead of using the required bounded delivery seam
    detail: |
      This is the 2nd finding in family `lifecycle-contract-coverage`. The master-loop ticker calls Advance synchronously; Advance performs stat, open, seek, an effectively suffix-sized ReadAll, JSON decoding, and processing of every returned record before PTY events resume. This violates the declared separate-goroutine/channel contract and permits journal growth or slow filesystem IO to stall terminal forwarding. Do not patch only the ReadAll site: enforce the class rule that all lifecycle journal IO remains outside the PTY owner loop, with bounded incremental reads and durable-cursor backpressure, while only reducer mutation executes on the master loop.
  - id: new
    severity: Critical
    family: plan-entity-inventory
    title: |
      The Core concepts table does not describe entities that actually exist at M3
    detail: |
      The table marks CodexWorkingRecognizer as new although its M4 file does not exist, and names AuthorizedTranscriptFollower and LifecycleJournal entities that have no matching declarations at their stated paths. Under the Core concepts cross-check this is a code/plan contradiction. Revise statuses to distinguish planned from delivered entities and use the actual declaration names, or implement the declared surfaces.
  - id: new
    severity: Important
    family: regression-test-reachability
    title: |
      The required live Codex lifecycle conformance remains unchanged
    detail: |
      This is the 2nd finding in family `regression-test-reachability`. The plan requires TestLiveNativeSessionShapeConformance to assert task_started/task_complete identity and monotonic offsets, but the pinned range does not modify that test; it still only runs the generic inventory report. State the rule for all production envelopes consumed by notification authority: each must be asserted through the recurring live conformance seam when parser support is introduced.
```

1. Strengths

- Strict parsing requires keyed `task_started`, `task_complete`, and `turn_aborted` envelopes instead of broad fallback matching in [event.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessioninventory/event.go:232).
- The short-turn regression genuinely exercises binding followed by ordered opener/terminal publication in [run_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/run_test.go:55).
- Journal append logic uses a stable launch/generation/transcript-offset identity and reconciles indeterminate writes in [lifecycle_event.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:49).
- The tailer correctly starts at prior EOF, filters launch ordinals, waits for committed lines, and fails closed on ordinary replacement/truncation cases.
- Atlas documentation was updated for the new journal and transcript-authority flow; no new README-run surface was introduced.

2. Critical findings

- [wrap.go:2632](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:2632), [lifecycle_journal.go:51](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:51) — `ARCH-PURE`, `ARCH-CONSTRAINTS`: move tailing and parsing behind the planned goroutine/channel seam, bound each advancement’s bytes/work, and leave reducer updates on the master loop. Add a regression proving blocked/large journal reads cannot delay PTY forwarding.
- [plan:17](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/workshop/plans/000161-couch-missed-codex-notifications-plan.md:17), [plan:74](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/workshop/plans/000161-couch-missed-codex-notifications-plan.md:74) — Core-concept inventory contradicts the code. Correct the entity names and introduce explicit `planned` status for M4-only rows.

3. Important findings

- [conformance_live_test.go:9](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessioninventory/conformance_live_test.go:9), [plan:163](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/workshop/plans/000161-couch-missed-codex-notifications-plan.md:163) — `ARCH-MOCK`: extend the opt-in production conformance to pin the exact lifecycle envelopes and identity/offset invariants now relied upon.

4. Minor findings

None.

5. Test coverage notes

The requested focused packages passed:

`go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/artifactpath -count=1`

`git diff --check` also passed. Existing tests cover short and appended turns, incremental reads, append reconciliation, partial records, launch filtering, replacement/truncation, and canonical OSC emission. They do not exercise the declared PTY non-blocking/backpressure envelope or the live lifecycle protocol.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Transcript selection reuses sessionwatch/sessioninventory authority and notification policy converges on the existing reducer.
- `ARCH-PURE`: Flag. Filesystem tailing and parsing are embedded in the terminal owner loop.
- `ARCH-PURPOSE`: Pass for M3’s transcript slice; short completed turns and later completion are both exercised.
- `ARCH-MOCK`: Flag. Stateful local fakes are good, but the promised recurring live-envelope conformance is absent.
- `ARCH-CONSTRAINTS`: Flag. Poll intervals exist, but the non-blocking, bounded-work delivery mechanism described by the plan is not implemented or measured.

7. Plan revision recommendations

Add entries under `## Revisions` stating:

- M3 journal IO is performed by a dedicated tailing worker with bounded incremental reads and capacity-32 delivery/backpressure; the PTY master loop owns only reducer mutation.
- Core concept statuses distinguish `delivered` from `planned`, and entity names correspond to greppable declarations.
- Every production-native envelope promoted to notification authority is added to `TestLiveNativeSessionShapeConformance` in the same milestone as parser support.

---

## Re-review — 2026-09-01T15:33:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 11db1d2d0aef81e131cbc4850f23734421b35ef7..11db1d2d0aef81e131cbc4850f23734421b35ef7 |
| command | sdlc milestone-close --issue 161 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-01T15:33:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The review window is invalid for an M3 boundary review: base and head are the identical commit `11db1d2d`, with identical tree `f520c8d…`. The mandatory stat, name-status, and full-diff recipes all returned an empty patch. Although the HEAD tree appears to contain candidate fixes for BR-5 through BR-7, none is attributable to or revert-testable within the pinned range. Re-pin the base to the actual pre-fix commit and rerun the review.

1. Strengths

- Required repository object and read-only commands were available and completed successfully.
- The HEAD tree contains a dedicated journal worker and capacity-32 delivery channel at `cmd/internal/wrapcmd/wrap.go:2616`.
- A blocked-IO isolation regression exists at `cmd/internal/wrapcmd/lifecycle_journal_test.go:213`.
- The Core concepts table now distinguishes `planned M4` from delivered milestones at `workshop/plans/000161-couch-missed-codex-notifications-plan.md:20`.
- Live conformance invokes lifecycle validation at `cmd/internal/sessioninventory/conformance_live_test.go:33`.

These are promising HEAD-state observations, not validated changes in the submitted range.

2. Critical findings

- **Review boundary contains no changes** — Base and head are both `11db1d2d0aef81e131cbc4850f23734421b35ef7`. Consequently, no M3 implementation, test, plan, or documentation change is reviewable. This is the **2nd finding in family `submission-boundary-reachability`**. Do not patch an individual test: enforce the rule that every milestone review identifies distinct pre-boundary and post-boundary commits and rejects an empty range before invoking review. Re-run with the actual parent/pre-fix base.

3. Important findings

None beyond prior finding BR-7, disposed as not-addressed below.

4. Minor findings

None.

5. Test coverage notes

Tests visible at HEAD cannot satisfy the claimed-fix rule for this review: the empty range provides no fix to revert and no way to establish that a test added by the reviewed change fails without that fix. In particular:

- BR-5 has a plausible regression at `lifecycle_journal_test.go:213`, but it is outside the empty range.
- BR-6 has no mechanical test in the range enforcing correspondence between the Core concepts table and declarations.
- BR-7 has lifecycle validation at `conformance_live_test.go:33`, but no modification is present in the pinned range.

No test suite was run because passing tests against HEAD would not repair the missing review boundary.

6. Architectural notes for upcoming work

- **ARCH-DRY — flag/unreviewable:** no patch exists in which duplication or consolidation can be assessed.
- **ARCH-PURE — flag/unreviewable:** the HEAD worker looks appropriately separated, but its introduction cannot be reviewed in this range.
- **ARCH-PURPOSE — flag:** an empty submission cannot demonstrate delivery of the M3 purpose or disposition of prior findings.
- **ARCH-MOCK — flag/unreviewable:** the HEAD tree contains fake-driven and live-conformance code, but no reviewed change establishes its reachability.
- **ARCH-CONSTRAINTS — flag/unreviewable:** the plan declares bounded channels and nonblocking PTY behavior at plan lines 112–124, but the pinned patch cannot demonstrate enforcement.

The atlas/README gate is likewise not assessable: neither documentation nor production changes occur in the range.

7. Plan revision recommendations

Add a `## Revisions` entry stating that milestone review ranges must use a distinct pre-change base and post-change head, and that the close command must fail before review when `base^{tree} == head^{tree}`.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Candidate worker isolation and a regression exist at HEAD, but the pinned range is empty, so the fix and its fail-without-fix test cannot be reviewed.
  - id: BR-6
    disposition: not-addressed
    note: |
      The HEAD plan appears corrected, but no plan change or mechanical reachability check exists in the pinned range.
  - id: BR-7
    disposition: not-addressed
    note: |
      HEAD invokes lifecycle conformance validation, but the empty range contains no live-conformance modification attributable to this boundary.
findings:
  - id: new
    severity: Critical
    family: submission-boundary-reachability
    title: |
      The pinned M3 review range contains no changes
    detail: |
      Base and head are the same commit and tree, so stat, name-status, and full patch are empty. This is the 2nd finding in family `submission-boundary-reachability`; enforce a class rule that milestone review rejects identical pre/post boundaries, then rerun with the actual pre-fix base and post-fix head.
```

---

## Re-review — 2026-09-01T15:36:56-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | dc43a9e8e67f940eee9dc6daeb789bcdf2b16392..d82dc7819df470a1cce1044b6445c1a54eaa9fa6 |
| command | sdlc milestone-close --issue 161 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-01T15:36:56-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range is valid, non-empty, and substantially improves M3: journal IO now runs in a bounded worker, lifecycle records are incrementally read, live Codex conformance is extended, atlas documentation is updated, and focused packages pass. The boundary remains blocked because the prior IO-isolation regression does not exercise the production master-loop wiring, the Core concepts inventory still claims undelivered M4 surfaces, and malformed journal records can authorize notifications instead of failing closed.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      The worker is wired in production, but its regression invokes the worker and reducer independently; reverting the master-loop integration would not make this test fail.
  - id: BR-6
    disposition: not-addressed
    note: |
      The table still marks terminalModel modified and Couch projection verified although their M4 work remains unchecked, and the prose names CodexWorkingRecognizer while the planned declaration is RecognizeCodexWorking.
  - id: BR-7
    disposition: addressed
    note: |
      The opt-in live conformance test now scans Codex events and requires a keyed task_started followed by same-root task_complete with timestamps and increasing positions.
  - id: BR-8
    disposition: addressed
    note: |
      The pinned objects exist and the range contains 26 changed files with substantive production and test changes.
findings:
  - id: new
    severity: Critical
    family: lifecycle-contract-coverage
    title: |
      Lifecycle journal validation accepts malformed or unauthorized record semantics
    detail: |
      This is the 3rd finding in family lifecycle-contract-coverage. Advance decodes one JSON value without requiring EOF, and validates only that agent, source, and outcome are non-empty; a line containing a valid record followed by garbage, or a current-launch record for another agent/source with outcome completed, can reach the notification reducer despite the fail-closed contract. Do not patch only trailing JSON: state and enforce the complete accepted-record rule, including exact framing, agent, source, outcome, identity, and timestamp constraints, with production-path regressions that fail when each guard is removed.
```

1. Strengths

- `followLifecycleJournal` performs filesystem work outside the PTY master loop and uses a capacity-32 record channel with stop-aware backpressure ([lifecycle_journal.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:143)).
- Each tail advance is bounded to roughly 64 KiB rather than reading the journal suffix without limit ([lifecycle_journal.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:93)).
- Transcript projection preserves keyed opener-before-terminal ordering and filters by the authorized root.
- Live conformance now checks the production envelopes consumed by notification authority.
- Atlas updates document the new journal, identity model, worker boundary, and failure behavior.

2. Critical findings

- [lifecycle_journal.go:118](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:118): `Decode` is called only once, so trailing non-whitespace data is not rejected. Require a second decode to return `io.EOF`, and centralize strict semantic validation shared with `LifecycleRecord.validate`.
- [lifecycle_journal.go:124](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:124): enforce the exact accepted envelope—Codex agent, transcript source, supported outcomes, non-empty keyed identity, and current launch—before returning any record.

3. Important findings

None newly raised. BR-6 remains open as disposed above.

4. Minor findings

None.

5. Test coverage notes

Focused tests passed:

```text
go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch \
  ./cmd/internal/wrapcmd ./cmd/internal/launcher \
  ./cmd/internal/artifactpath -count=1
```

The full `go test ./... -count=1` run could not complete because generated `cmd/internal/runtimebundle/assets/runtime/files` content is absent; all packages that compiled before the final `FAIL` otherwise passed.

The BR-5 regression at [lifecycle_journal_test.go:213](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal_test.go:213) proves that a separately launched worker does not block an unrelated reducer goroutine. It does not prove that `masterPump` uses that worker. Add a proxy/master-pump seam with an injected blocking advancer, demonstrate PTY forwarding continues, and verify the test fails if `Advance` is moved onto the owner loop.

Add tests that reject:

- A valid JSON object followed by another value or garbage.
- Non-Codex agents and non-transcript sources.
- Unknown or empty outcomes.
- Missing turn IDs for all supported lifecycle outcomes.
- A malformed current-launch record before any notification is emitted.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass overall. Existing session inventory authority and the shared lifecycle reducer are reused. Consolidate writer and reader record validation to avoid parallel contract definitions.
- `ARCH-PURE`: Pass. Lifecycle reduction remains separate from journal IO, and reducer mutation stays on the owner loop.
- `ARCH-PURPOSE`: Flag. Fail-closed journal authority and production-reachable IO isolation are part of M3’s core purpose; the remaining validation and reachability gaps are not separable follow-ups.
- `ARCH-MOCK`: Pass for the watcher/inventory seam and live conformance extension. The BR-5 integration regression still needs to exercise the actual production seam.
- `ARCH-CONSTRAINTS`: Pass in implementation shape: 50 ms polling, bounded reads, bounded channel capacity, and worker isolation are present. Production-loop isolation needs the reachable regression described above.

7. Plan revision recommendations

Add a `## Revisions` entry that:

- Defines the exhaustive accepted lifecycle-record envelope and requires strict single-value JSON framing.
- Requires producer and consumer to share that validation rule.
- Replaces the BR-5 helper-only test with a production master-loop reachability regression.
- Changes `terminalModel` to `planned M4`, Couch projection to `planned/verified M4`, and consistently names the future declaration `RecognizeCodexWorking`.

---

## Re-review — 2026-09-01T15:45:01-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | dc43a9e8e67f940eee9dc6daeb789bcdf2b16392..6ff1a8aef21ba01e1fb9f5ea5cf2c2d33623b711 |
| command | sdlc milestone-close --issue 161 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-01T15:45:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3 now correctly moves journal IO off the PTY loop, closes the record grammar, and repairs the Core concepts inventory. Focused tests pass. One blocking fail-closed gap remains: journal reconciliation accepts any JSON row sharing three identity fields as proof that the intended record committed, even when that row is malformed or semantically different. This can silently suppress the authoritative lifecycle record and lose a completion notification.

## Strengths

- Journal IO and decoding now run in a dedicated worker, while reducer mutation remains on the PTY owner loop ([lifecycle_journal.go:145](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal.go:145), [wrap.go:2616](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/wrap.go:2616)).
- The production-path regression demonstrates PTY forwarding while injected journal IO is blocked ([lifecycle_journal_test.go:279](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/wrapcmd/lifecycle_journal_test.go:279)).
- Reader framing is strict: unknown fields, trailing JSON, oversized records, replacement, and truncation fail closed.
- `ValidateLifecycleRecord` now single-sources the accepted agent/source/outcome/identity grammar for producer and consumer ([lifecycle_event.go:35](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:35)).
- Atlas changes describe the new lifecycle journal and session-identity surface; no new user-typed README surface was introduced.

## Critical findings

- [lifecycle_event.go:168](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:168) — `lifecycleIdentityPresent` uses permissive `json.Unmarshal` and compares only launch ordinal, artifact generation, and transcript offset. An existing row with those fields but an unauthorized agent/source, invalid outcome, different turn, or different event semantics makes `AppendLifecycleRecord` return success through reconciliation without committing the intended record. If the tailer opened after that row, it will never inspect it, so the notification is silently lost.

  **This is the 4th finding in family `lifecycle-contract-coverage`.** Earlier rounds fixed instances. Do not patch only one malformed-row case: state and enforce the class rule that durability reconciliation may succeed only when a complete, strictly decoded, validated committed row represents the exact intended lifecycle observation. Add regressions enumerating malformed framing, unknown fields, invalid authority fields, and same transport identity with differing semantic fields; each must fail without the fix. (`ARCH-PURPOSE`)

## Important findings

None.

## Minor findings

None.

## Test coverage notes

Focused tests passed:

```text
go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/artifactpath -count=1
```

`git diff --check` passed.

`go test ./... -count=1` exercised the remaining packages successfully but ended `FAIL` because the checkout lacks generated `cmd/internal/runtimebundle/assets/runtime/files`, causing setup failures in packages embedding that path. This appears unrelated to the reviewed range, but a fully prepared close environment should rerun the suite.

The new reconciliation defect has no regression test. Existing indeterminate-write tests cover exact valid rows only.

## Architectural notes

- `ARCH-DRY`: Pass. Record validation and lifecycle reduction have shared authorities rather than parallel implementations.
- `ARCH-PURE`: Pass. Reducer mutation remains single-threaded and journal IO is isolated behind an injected advancer.
- `ARCH-PURPOSE`: Flagged. Reconciliation does not enforce the complete fail-closed lifecycle-record rule.
- `ARCH-MOCK`: Pass. Stateful runtime and blocked-IO fakes exercise the production seams; live Codex envelope conformance is wired into the opt-in test.
- `ARCH-CONSTRAINTS`: Pass. Reads are capped at 64 KiB plus a sentinel, delivery uses bounded backpressure, and blocking filesystem work is outside the PTY loop.

## Plan revision recommendations

Add under `## Revisions`:

> Durability reconciliation must strictly decode and validate every candidate committed row and may report the attempted observation committed only when its complete semantic content matches the attempted record. Enumerate malformed framing, unknown fields, unauthorized authority fields, and same stable transport identity with differing lifecycle semantics in producer-path regressions.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Journal IO now runs in a dedicated bounded worker, and a production master-pump regression proves PTY forwarding continues while injected journal IO blocks.
  - id: BR-6
    disposition: addressed
    note: |
      The Core concepts table now distinguishes planned M4 entities from delivered entities and uses declarations present at their stated paths.
  - id: BR-9
    disposition: addressed
    note: |
      Strict EOF framing and a shared closed record validator now reject invalid agent, source, outcome, identity, and timestamp fields, with reader-path regressions.
  - id: BR-7
    disposition: addressed
    note: |
      The opt-in live conformance test invokes the new keyed Codex lifecycle-envelope validator.
  - id: BR-8
    disposition: addressed
    note: |
      The pinned review range now contains the substantive M3 implementation and regression changes.
findings:
  - id: new
    severity: Critical
    family: lifecycle-contract-coverage
    title: |
      Journal reconciliation treats unauthorized identity collisions as committed lifecycle records
    detail: |
      lifecycleIdentityPresent permissively decodes rows and compares only launch, artifact generation, and offset, so a malformed or semantically different row can suppress the intended append and silently lose notification authority. This is the 4th finding in family lifecycle-contract-coverage; enforce strict complete-record equivalence across every reconciliation candidate and pin the enumerable malformed and semantic-mismatch classes with producer-path regressions.
```

---

## Re-review — 2026-09-01T15:49:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 161 — Couch misses Codex completion notifications |
| repo | 000161-couch-missed-codex-notifications |
| issue file | workshop/issues/000161-couch-missed-codex-notifications.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | dc43a9e8e67f940eee9dc6daeb789bcdf2b16392..99fb1bab89ef360275006671de000c07d9f8f4e3 |
| command | sdlc milestone-close --issue 161 --milestone M3 |
| reviewer | codex |
| timestamp | 2026-09-01T15:49:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The M3 boundary is ready. BR-10 is addressed with strict, complete-record reconciliation and a reachable producer-path regression. The focused M3 packages pass, documentation and the Core concepts inventory match the delivered surface, and no new blocking findings were identified.

## Strengths

- Reconciliation first strictly decodes and validates candidates, then compares every lifecycle field, including outcome, turn, message, path, offset, and timestamp ([lifecycle_event.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:38), [lifecycle_event.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event.go:70)).
- The producer-path regression enumerates malformed JSON, unauthorized semantics, and valid semantic mismatches ([lifecycle_event_test.go](/Users/xianxu/workspace/worktree/pair/000161-couch-missed-codex-notifications/cmd/internal/sessionwatch/lifecycle_event_test.go:90)).
- Transcript publication remains rooted in the existing authorized session watcher, with incremental offsets and current-launch filtering rather than a parallel resolver.
- Journal filesystem work runs outside the PTY master loop through a bounded, backpressured channel.
- The new lifecycle journal and transcript-authority flow are documented in both relevant atlas pages.

## Critical findings

None.

## Important findings

None.

## Minor findings

None.

## Test coverage notes

- BR-10 mutation check: replacing complete equality with the former launch/artifact/offset-only comparison made `TestAppendLifecycleRecordDoesNotReconcileDifferentOrInvalidObservation` fail for outcome, turn ID, message, transcript path, and timestamp. The fix is therefore reachable and regression-pinned.
- Passed:
  `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/artifactpath -count=1`
- `git diff --check` passed.
- `go test ./... -count=1` exercised the remaining packages successfully but returned failure because `cmd/internal/runtimebundle/assets/runtime/files` was absent for `go:embed`. This is outside the reviewed diff; the M3 packages themselves passed.

## Architectural notes for upcoming work

- `ARCH-DRY`: Pass — transcript selection reuses the established watcher/inventory authority and lifecycle delivery converges on the existing reducer.
- `ARCH-PURE`: Pass — lifecycle projection and reduction remain pure; filesystem polling and journal IO stay in thin boundary workers.
- `ARCH-PURPOSE`: Pass — short and appended Codex turns publish keyed opener/terminal records, and concurrency/root authorization is preserved.
- `ARCH-MOCK`: Pass — mutable transcript and filesystem behavior use injected stateful runtimes; live lifecycle conformance validates the production envelope.
- `ARCH-CONSTRAINTS`: Pass — transcript polling, journal work, record size, channel capacity, and PTY non-blocking behavior enforce the declared M3 envelope.

## Plan revision recommendations

None.

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      Strict decoding plus complete-record equality now governs reconciliation, and the producer-path regression fails when the former permissive three-field comparison is restored.
```
