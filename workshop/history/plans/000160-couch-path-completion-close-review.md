# Boundary Review — 000160-couch-path-completion#160 (whole-issue close)

| field | value |
|-------|-------|
| issue | 160 — Couch start path tab completion |
| repo | 000160-couch-path-completion |
| issue file | workshop/issues/000160-couch-path-completion.md |
| boundary | whole-issue close |
| milestone | — |
| window | bf65381d5127ffb4d662a54ae0d34fe929a852dc..f7f72d59ca224bcf9f8d93d29bd62a3b7a121835 |
| command | sdlc close --issue 160 |
| reviewer | codex |
| timestamp | 2026-09-01T13:12:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The production design largely matches the specification: completion remains reducer-owned, filesystem work is asynchronous and bounded, stale results use exact identities, and documentation was updated. Two inexpensive gaps remain: directory-close errors are silently discarded despite the plan’s joined-error contract, and the integration fake does not honor the batched reader contract. The latter leaves several promised error, cancellation, and batching behaviors untested.

1. Strengths

- `SplitCompletionPath` preserves relative, absolute, repeated-separator, dot, and literal-tilde spelling, with direct pure tests.
- `CompletionAccumulator` maintains deterministic lexical top-200 results with bounded retained memory.
- `ReduceMenu` requires both frame instance and generation before applying completion results ([menu.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu.go:749)).
- The generic latest-wins scheduler provides one running and one replaceable pending request, shared with preview behavior.
- Both documentation gates are satisfied through updates to `README.md` and `atlas/couch.md`.

2. Critical findings

None.

3. Important findings

- [console_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/console_completion.go:27): `defer file.Close()` discards the close error. Task 5 explicitly promises “joined terminal errors,” and filesystem failures should not silently become successful completion. Use a named return error and `errors.Join(resultErr, file.Close())`, then pin the behavior with an injected close-capable cursor or narrower opener seam whose failing close makes the test fail.

- [console_completion_test.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/console_completion_test.go:53): the fake ignores `batchSize` and yields every configured entry in one call, so it does not implement the specification’s “same batched contract” (`ARCH-MOCK`). Make it emit chunks no larger than the requested size and retain observable per-call/cancellation state. Add integration cases for reader errors, cancellation, coalesced duplicate requests, stale results/notices, and latest-pending replacement—the behaviors explicitly claimed by the issue and plan.

4. Minor findings

None.

5. Test coverage notes

The focused package tests, race-targeted completion tests, full `go test ./...`, and `git diff --check` passed. Pure path, accumulation, reducer interaction, scheduling, rendering, and live-loop navigation have useful coverage.

Coverage does not presently substantiate all claimed Console-worker behavior. In particular, no test forces directory close failure, and the fake cannot detect violation of the 128-entry batch contract.

6. Architectural notes for upcoming work

- `ARCH-DRY`: Pass. Preview and completion share `advanceLatestSchedule`; no material duplicated scheduling implementation was introduced.
- `ARCH-PURE`: Pass. Path splitting, accumulation, reducer transitions, scheduling, and rendering remain deterministic and IO-free.
- `ARCH-PURPOSE`: Pass. Completion remains advisory and acceptance still flows through existing preview/token authorization.
- `ARCH-MOCK`: Flagged. Production and test paths share the interface, but the fake does not model its bounded-batch contract.
- `ARCH-CONSTRAINTS`: Pass for production. Enumeration is asynchronous, concurrency is one active plus one pending, batches are 128 entries, retained matches are capped at 200, and rendering is height-bounded.

7. Plan revision recommendations

If the implementation is not expanded to match the checked claims, add a `## Revisions` entry correcting Task 5/6 and the integration testing description: the current range does not prove joined close failures, bounded fake batching, local reader errors, or stale notice behavior.

```findings
findings:
  - id: new
    severity: Important
    family: filesystem-terminal-error-propagation
    title: |
      Directory close failures are silently discarded
    detail: |
      OSDirectoryBatchReader defers file.Close without joining its error into the returned scan error, contrary to Task 5's joined-terminal-error contract. Propagate the close failure and add a test that fails when it is swallowed.
  - id: new
    severity: Important
    family: external-double-contract-fidelity
    title: |
      The directory-reader fake does not model bounded batching
    detail: |
      The fake ignores batchSize and yields all configured entries at once, so the integration path cannot enforce the production seam's bounded-batch contract or several claimed cancellation and error behaviors. Make the fake stateful and batch-faithful, then pin those worker behaviors.
```

---

## Re-review — 2026-09-01T13:20:05-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 160 — Couch start path tab completion |
| repo | 000160-couch-path-completion |
| issue file | workshop/issues/000160-couch-path-completion.md |
| boundary | whole-issue close |
| milestone | — |
| window | bf65381d5127ffb4d662a54ae0d34fe929a852dc..31d499d2d03693080edbd65a4a2e832d85683553 |
| command | sdlc close --issue 160 |
| reviewer | codex |
| timestamp | 2026-09-01T13:20:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation substantially delivers the specified asynchronous, bounded directory completion and documents the new surface. BR-2 is demonstrably fixed. BR-1 remains incomplete: cancellation can cause the worker to discard a close error joined with `context.Canceled`. Fix that error classification and add a combined cancellation-plus-close regression test before shipping.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The reader now joins Close failures, but the worker clears the entire joined error whenever errors.Is(err, context.Canceled), silently discarding a simultaneous Close failure.
  - id: BR-2
    disposition: addressed
    note: |
      The stateful fake now emits bounded chunks and configured terminal errors; its direct contract test would fail with the prior all-at-once implementation.
```

## Strengths

- The pure accumulator retains a deterministic lexical top 200 without materializing all matches ([menu_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu_completion.go:65)).
- Exact frame-instance/generation matching makes stale completion results inert, with completion slices cloned for reducer immutability ([menu.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu.go:741)).
- Production enumeration uses bounded `ReadDir` calls and follows symlinks only at the IO boundary ([console_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/console_completion.go:26)).
- README and atlas both describe the new interaction and architecture.
- Existing preview authorization remains the only start-dispatch path.

## Critical findings

None.

## Important findings

- **BR-1 — `filesystem-terminal-error-propagation`: cancellation still swallows joined cleanup failures.** At [console_completion.go:117](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/console_completion.go:117), `errors.Is(err, context.Canceled)` sets the complete error to `nil`. When `ReadDirectoryBatches` returns `errors.Join(context.Canceled, closeErr)`, this erases `closeErr`.

  **This is the 2nd finding in family `filesystem-terminal-error-propagation`.** Apply the class-wide rule: expected cancellation may be suppressed only when it is the sole terminal condition; non-cancellation read/cleanup errors must survive. Add a test combining cancellation with an injected close failure and verify the close failure remains observable. The current close-only test does not cover this path.

## Minor findings

None.

## Test coverage notes

- Targeted race tests passed.
- `go test ./cmd/internal/couchtty -count=1` passed.
- BR-2’s test directly observes 128+3 batching and configured errors.
- The full `go test ./... -count=1` run reached unrelated `cmd/pair-go` tests but failed because the sandbox denied `/bin/ps`; no issue-specific package failed.
- `git diff --check` for the pinned range passed.
- The worktree contains an unrelated untracked `.nvimlog`; it was not inspected as part of the committed range.

## Architectural notes

- `ARCH-DRY`: Pass — preview and completion share the generic latest-wins scheduler.
- `ARCH-PURE`: Pass — query, accumulation, reducer transitions, scheduling, and rendering remain IO-free.
- `ARCH-PURPOSE`: Pass — completion stays advisory and does not bypass preview/token authorization.
- `ARCH-MOCK`: Pass — production and tests share `DirectoryBatchReader`, and the fake is stateful and batch-faithful.
- `ARCH-CONSTRAINTS`: Pass — one active/one pending scan, 128-entry reads, 200 retained candidates, and bounded rendering implement the declared envelope.

## Plan revision recommendations

Add a `## Revisions` entry recording the terminal-error rule: cancellation suppression must preserve any joined non-cancellation read or cleanup failure, with the combined cancellation-plus-close regression test named as its guard.

---

## Re-review — 2026-09-01T13:23:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 160 — Couch start path tab completion |
| repo | 000160-couch-path-completion |
| issue file | workshop/issues/000160-couch-path-completion.md |
| boundary | whole-issue close |
| milestone | — |
| window | bf65381d5127ffb4d662a54ae0d34fe929a852dc..09f6ee1aab146d1d4c49cbf2b5a5b3baa51c9f3e |
| command | sdlc close --issue 160 |
| reviewer | codex |
| timestamp | 2026-09-01T13:23:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the issue’s completion behavior and architectural envelope. The remaining BR-1 fix preserves non-cancellation leaves from joined terminal errors, and its Console-level regression would fail under the previous broad cancellation suppression. BR-2’s bounded stateful fake remains directly pinned. Targeted race and package tests pass; the only full-suite failures are sandbox-denied `/bin/ps` executions in unrelated `cmd/pair-go` tests.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      completionTerminalError removes cancellation leaves while preserving joined close failures; the Console regression exercises the production worker path and fails with the prior errors.Is suppression.
  - id: BR-2
    disposition: addressed
    note: |
      The stateful fake emits requested-size batches and terminal errors, with a direct contract test that fails under the former all-at-once behavior.
```

## Strengths

- Exact frame-instance and generation matching rejects stale results after edits, exits, and later form lifetimes ([menu.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu.go:749)).
- Completion preserves literal editable prefixes, including relative paths and repeated separators ([menu_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu_completion.go:20)).
- The accumulator deterministically retains only the lexical top 200 matches ([menu_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/menu_completion.go:75)).
- The filesystem shell uses bounded reads, injected enumeration, and target-following symlink classification ([console_completion.go](/Users/xianxu/workspace/worktree/pair/000160-couch-path-completion/cmd/internal/couchtty/console_completion.go:26)).
- README and atlas both document the new user-facing and architectural surfaces.

## Critical findings

None.

## Important findings

None.

## Minor findings

None.

## Test coverage notes

- `go test -race ./cmd/internal/couchtty -run '^(TestConsoleCompletion|TestOSDirectoryBatchReader|TestFakeDirectoryBatchReader)' -count=1` passed.
- `go test ./cmd/internal/couchtty -count=1` passed.
- `git diff --check` passed.
- The BR-1 regression reaches the real Console worker and result reducer. Restoring the prior `errors.Is(err, context.Canceled)` suppression would remove the expected notice.
- The full suite passed all issue-relevant packages. `cmd/pair-go` alone failed because the review sandbox prohibits `/bin/ps`, not because of a product assertion.

## Architectural notes for upcoming work

- `ARCH-DRY`: Pass — preview and completion share the generalized latest-wins scheduler.
- `ARCH-PURE`: Pass — path splitting, bounded accumulation, scheduling, reducer transitions, and rendering are IO-free.
- `ARCH-PURPOSE`: Pass — completion fulfills the directories-only workflow while leaving preview-token authorization as the sole start path.
- `ARCH-MOCK`: Pass — production and integration tests use the same batched reader seam, backed by a stateful, batch-faithful fake.
- `ARCH-CONSTRAINTS`: Pass — one active/one pending scan, 128-entry batches, 200 retained candidates, asynchronous IO, and terminal-height clipping enforce the declared envelope.

## Plan revision recommendations

None. The Core concepts table, completed tasks, and existing revisions agree with the code.
