# couch --prepare repo:N Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --prepare repo:N --confirm [--force-unknown] [--json]` brings an idle slot up for new work in one command: admit (idle, clean, resting branch), `weave refresh`, re-check clean, reload the agent's context, and report the substrate commits the slot's `sdlc`/`weave` now build from.

**Architecture:** A CLI-side composition of two remote calls through the running Couch around one local `weave refresh`. The refresh runs in the CLI process (as `couch --reconcile` already runs `weave compile` in the calling process), never on the console's serial operation queue, which would block every other console operation for the minutes a refresh takes. The remote half reuses the slot-operation protocol: a new admission-only slot operation `prepare` (its job admits and has no effect) and the existing `reload-context`.

**Tech Stack:** Go; `cmd/internal/couchcore`, `cmd/internal/couchcmd`, `cmd/internal/couchmessage`; real git in tests, a fake `weave` behind the existing `couchcore.ProvisionIO` seam.

---

## Design decisions

- **Why not a Couch-side job.** `couchtty.operationQueue.Run` runs one job at a time. A refresh holding it for minutes would freeze Alt-key operations and every other remote slot operation. Running weave on a side goroutine would need a second receipt lifecycle. The CLI process already has the precedent (`reconcile`, ExecuteDirectStore) and its own poll loop.
- **Why an admission-only remote op.** Only the running Couch knows whether the slot's wrapper reports Settled. The Done-when demands that a busy slot is refused *before anything changes*, so admission must be a separate round trip before the refresh. It is declared like reload-context (ExecuteLiveOwner, ConfirmRequired, `--force-unknown` allowed) so the CLI parser, protocol validator and socket admit it from the one verb list (`slotOperations`, ARCH-DRY). Its dispatched job re-reads nothing and changes nothing; it returns `PrepareAdmission{Address, Path}`.
- **Resting branch.** Admission adds one rule for `prepare` only: the checkout is on `RestingBranch(N)` (`main` for `:0`) and not detached; refusal code `not-resting`. Reload-context keeps its rules.
- **Path to the CLI.** The receipt gains `Path` (omitempty), filled from a result's `ReceiptPath()`. The running Couch's resolution of `repo:N` is the one the refresh then runs in; the CLI resolves nothing itself.
- **`--confirm`.** The spec wrote `couch --prepare repo:N [--json]`; prepare ends the agent's conversation, as reload-context does, so it requires `--confirm` like every confirmed slot operation (spec revision logged).
- **Known trap: refresh dirtying its own checkout.** Compile writes ariadne#320's union-merge block into the tracked `.gitattributes`; until a repo commits it (pair#436 for pair), the checkout is dirty after the refresh. Prepare re-checks the checkout after the refresh and, if dirty, stops before the reload with code `refresh-dirty`, naming the modified paths and the fix (commit the compile output on main). Couch does not special-case `.gitattributes` (weave's internals are not Couch's) and never commits it.
- **Race between admission and reload.** The reload re-admits (idle + clean) at its own effect, so a turn that starts during the refresh refuses the reload; the report says "refreshed, not reloaded" with the reload's code. Re-running prepare converges: refresh is idempotent.
- **Durable state (ARCH resource rule).** Creates nothing durable: the admission receipt lives in the existing bounded in-memory receipt map (swept by `ReceiptExpire`); weave's outputs are weave's.
- **Which weave.** `exec` resolves `weave` on the CLI's PATH, the same resolution `couch --reconcile` uses. The report names the substrate commits after the refresh, which is what the slot's `sdlc`/`weave` shell functions build from.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `OpPrepare`, `LiveRestartNotResting` | `cmd/internal/couchcore/live_restart.go` | new |
| `LiveRestartFacts` (Branch, Detached, Resting) | `cmd/internal/couchcore/live_restart.go` | modified |
| `DecideLiveRestart` | `cmd/internal/couchcore/live_restart.go` | modified |
| `PrepareAdmission` | `cmd/internal/couchcore/slot_prepare.go` | new |
| `PrepareReport`, `PrepareStep`, `SubstrateCommit` | `cmd/internal/couchcore/slot_prepare.go` | new |
| `OperationReceipt.Path`, `ReceiptOutcome.Path` | `cmd/internal/couchmessage/operation.go` | modified |

- **DecideLiveRestart** — gains the resting-branch rule for `prepare`, checked after idle and clean (fixed order: the first thing to fix is reported). Unit-tested in `live_restart_test.go`.
- **PrepareReport** — one row per step (`admit`, `refresh`, `recheck`, `reload`) with status `ok|refused|failed|skipped` and detail, plus the checkout HEAD vs `origin/main` and the substrate commits. Rendered as lines or JSON.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `RunPrepare` | `cmd/internal/couchcore/slot_prepare.go` | new | git, weave (via `ProvisionIO`), remote admit/reload (funcs) |
| `liveRestartProbe.LiveRestartFacts` | `cmd/internal/couchcmd/live_restart_probe.go` | modified | `ProbeSlotGit` (now also branch) |
| `runPrepareCLI` | `cmd/internal/couchcmd/prepare.go` | new | the broker socket (two slot-operation exchanges) |

- **RunPrepare(ctx, PrepareRun{Admit, Reload func, IO ProvisionIO, Progress, Timeout})** — the sequence. Admit returns `PrepareAdmission` or a refusal; refresh = `IO.Run(weave refresh)` streaming to Progress; recheck = `git status --porcelain` in the path; reload; then HEAD/origin and `layergraph.DeclaredSubstrates(path)` HEADs. Tests: real temp git repos (origin + slot clone on `main-slot1`, plus a substrate repo), a fake ProvisionIO that delegates git to `OSProvisionIO` and implements `weave refresh` by fast-forwarding the checkout and substrate (or by dirtying `.gitattributes`), fake Admit/Reload funcs that record calls.
- **runPrepareCLI** — reuses `runSlotOperationCLI`'s admission/poll exchange (factored into `slotOperationExchange(op) (receipt, exit)`), so prepare and reload share one poll loop.

## Plan

### Task 1: admission rule and op declaration
**Files:** `couchcore/live_restart.go`, `couchcore/slot_operation.go`, `couchcore/ops.go`, `couchcore/operationdispatch.go`, `couchcmd/live_restart_probe.go`, `couchmessage/protocol.go` (error text), tests `couchcore/live_restart_test.go`, `couchcore/ops_declarations_test.go` as needed.
- [ ] Failing tests: `DecideLiveRestart(OpPrepare, …)` refuses `not-resting` for a feature branch and for detached; admits on resting; busy/dirty refusals come first; reload-context ignores the branch.
- [ ] Add `OpPrepare` to `slotOperations`, `SlotOperationTakesForceUnknown`; declare it in `Operations()` (ExecuteLiveOwner, ConfirmRequired, Args like reload-context); `prepareSlotOperation` routes it to `prepareLiveRestart`, which sets `Resting = RestingBranch(ref.Number)` and puts `path` in the call args; dispatch returns `PrepareAdmission{Address, Path}` (ReceiptTag, ReceiptPath).
- [ ] Probe fills Branch/Detached from `ProbeSlotGit`.
- [ ] `go test ./cmd/internal/couchcore/ ./cmd/internal/couchmessage/ -run 'LiveRestart|Operation|Protocol'` green; commit.

### Task 2: receipt path
**Files:** `couchmessage/operation.go`, `couchcmd/slot_operations.go` (`slotOperationOutcome`, `notedResult.ReceiptPath`), tests.
- [ ] Failing test: a finished prepare admission's receipt carries `Path`; `ApplyReceiptEvent` copies it on finish.
- [ ] Implement; commit.

### Task 3: RunPrepare
**Files:** create `couchcore/slot_prepare.go`, `couchcore/slot_prepare_test.go`.
- [ ] Failing tests (Done-when 1–3): stale clean idle slot → steps all ok, checkout at origin/main, fake weave ran once in the path, Reload called once, report names the substrate's new HEAD; Admit refusal → no weave, no reload, checkout unchanged; fake weave that dirties `.gitattributes` → `refresh-dirty` naming `.gitattributes`, Reload not called; weave failure → `failed`, reload `skipped`; Reload refusal → refresh ok, reload refused.
- [ ] Implement; tests green; commit.

### Task 4: CLI
**Files:** create `couchcmd/prepare.go`, `couchcmd/prepare_test.go`; modify `couchcmd/messages.go` (route `prepare`, factor the exchange), `couchcmd/run.go` (usage line).
- [ ] Failing test with the fake `messageCall` (as `messages_test`/`run_test` do): admission refused → exit 1, no second request; admitted → reload request sent with `Confirmed` and the same `ForceUnknown`; `--json` prints one `PrepareReport`.
- [ ] Implement; `go test ./cmd/internal/couchcmd/ -run 'Prepare|SlotOperation|Usage|CLI'` green; commit.

### Task 5: docs
- [ ] `atlas/couch.md`: the prepare verb, its four steps, refusal codes, why the refresh runs in the CLI. `cmd/internal/couchcmd/skills/couch/SKILL.md`: the TL dispatch flow is prepare, then send. CHANGELOG entry.
- [ ] Full `make -k test` + `go test ./...` before `sdlc close` (memory: full suite before close).
