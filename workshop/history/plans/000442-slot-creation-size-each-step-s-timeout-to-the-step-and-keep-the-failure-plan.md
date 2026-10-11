# #442 plan — budget slot steps by class, keep the failure

**Goal:** a slot creation never dies on a probe-sized budget for a step that writes a checkout. When creation does fail, the step, command, budget and stderr survive the moment.

**Root cause (confirmed, see the issue Log):** `slotConverger.worktreeAdd` ran `git worktree add` through `WorkspaceProvisioner.git`, i.e. `ProbeTimeout` (5 s). Under fleet load, parley.nvim's checkout (1687 files) passed 5 s. The group SIGTERM made git remove its half-made worktree. That left an empty env dir, and the error text existed only on screen.

## Design

1. **Budget by class** (`provision.go`). Add `WriteTimeout` (2 min) beside Probe/Fetch/Setup. `gitWithin(timeout)` is the one git runner; `git` = probe and `gitWrite` = write. `worktree add` and `worktree repair` use `gitWrite`. Everything else is read-only or a single-ref write and stays a probe; fetch and compile keep their budgets.
2. **Commands name themselves** (`provision.go`). `command()` wraps every failure in `ProvisionCommandError{Command, Timeout, TimedOut, Err}`. `TimedOut` holds only when the command's own budget expired, not the caller's context. `Unwrap` keeps `errors.Is/As` (deadline → retryable classification; `ExitCode` in `configValue`).
3. **Advice names the step** (`slotfailure.go`). `ReconcileFailure.Step` is set by `ClassifyConvergeError`, and `ReconcileAdvice` renders `host (step worktree-add) did not converge: git worktree add …: timed out after 5s: …`. That one text reaches every UI caller.
4. **Keep the failure** (`slotmemo.go`, `slotlayout.go`). `Ensure` writes `SlotFailure` (at, resource, step, class, command, timeout, timed_out, full cause with stderr tail) to `<common>/couch-workspaces/N/last-failure.json` under the creation lease when the outcome blocks. It removes the record when the slot converges. The record is bounded at one per slot, and writing it is best effort: the error still reaches the caller.
5. **Show it** (`slotreport.go`, `couchcmd/run.go`). `SlotReport.LastFailure`; `couch --show repo:N` prints `last failure <time> at <step>: <cause>`.
6. **Retry** needs no code. An env without a host is `ErrSlotNeedsReconcile`, which the allocator reuses and the reconciler re-adds.

## Tests (`cmd/internal/couchcore/provision_budget_test.go`)
- `TestWorktreeAddUsesWriteBudget`: a fake `worktree add` slower than the probe budget completes and receives `WriteTimeout`. Mutation-checked: putting it back on `cv.git` fails.
- `TestFailedCreationIsRecordedAndRetryIsClean`: a timed-out add's error names the step and budget, and the record carries step/command/timeout/stderr. A second `Ensure`, with no manual cleanup, returns `created` and removes the record.
- `TestCommandErrorTimedOutIsTheCommandsOwnBudget`: a caller's deadline is not reported as the command's timeout.

## Verification
- Full `make -k test` + `go test ./...` (see memory: full suite before close).
- Live: rebuild pair:0's binary, then create parley.nvim:3 from Couch (or `couch --reconcile parley.nvim:3`).
