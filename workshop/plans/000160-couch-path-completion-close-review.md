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
