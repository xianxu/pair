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
