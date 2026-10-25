# Bench slot verbs (pair#847) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A non-main Couch hub manages the worker slots it claims by command — start or restart one fresh with an exact agent and argv, and retire it with proof nothing it started survives — with receipts that carry provenance, while production slots stay out of reach.

**Architecture:** Two new broker slot operations, `bench-start` and `bench-retire`, ride the existing hub slot-operation path (admission check, queued effect guard, console lifecycle queue, receipts). `bench-start` is Reboot with an explicit launch profile threaded through the existing Fresh recovery (one launch path, ARCH-DRY). `bench-retire` is ArchiveThread with a strict teardown policy that refuses when the proof names any survivor. A new ownership predicate, `HubOwnership.Provisions`, admits only a non-main hub acting on a slot it holds by an explicit `.hub` claim, so default (main-hub) ownership — production — never qualifies.

**Tech Stack:** Go; Couch packages `cmd/internal/couchcore`, `couchcmd`, `couchmessage`; native tests in `cmd/couch` on the `cmd/internal/testcouch` harness with fixture agents.

---

## Scope and decisions (TL-approved 2026-10-25)

- Distinct verbs `--bench-start` / `--bench-retire`, sharing Reboot's implementation; no second launch path.
- `bench-retire` keeps the worktree (the bench runner owns checkouts). It must prove the agent and every process it started are gone before reporting the slot vacant, and refuse with the named survivors otherwise.
- Receipt fields are additive (`omitempty`); `.hub` claim files and slot layout are unchanged (transitional compatibility: existing slot registries keep working).
- Existing verbs (`--reboot`, `--switch-agent`, `--set-model`, `--wake`) keep their gates and behavior.
- **How the bench hub comes up (the one hand step left, named):** the bench tool writes the bench root's `.hub` file from the run descriptor, claiming the worker slot(s) by exact `repo:N`; the operator then starts Couch once with the bench root as an extra positional hub path (`couch <main-root> <bench-root>`). Couch reads `.hub` files only for explicitly started hubs, at startup. From then on, every run step is a command from the bench hub: `--bench-start` per run (agent, model and effort are argv), `--bench-retire` after it. No console steps.
- Model switch = another `--bench-start` with the new argv (fresh restart, new generation). `--set-model` stays the live, in-conversation switch and is not changed.
- **bench-start never writes saved launch parameters.** Saved parameters are keyed by repository identity, so bench slots share them with production slots of the same repo. A Fresh recovery launch is already never remembered (`acceptedConfigurationForStart` returns nil for `Recovery`, `repopreferences_adoption.go:21-27`), so the explicit profile applies to that launch only. Asserted in Tasks 2 and 6.
- **A whole-repo `.hub` claim grants `Provisions` over every slot of that repo not pointed elsewhere by an exact claim** — the same reach `authorizeHubMutation` already gives it. Non-goal: narrowing to slots the hub actually started.
- **A refused bench-retire** leaves the session stopped and the thread record kept (not archived); the slot is not reported vacant, and a retry refuses again while any survivor lives.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `HubOwnership.Provisions` | `cmd/internal/couchcore/hub_ownership.go` | new |
| `BenchProfile` | `cmd/internal/couchcore/bench_profile.go` | new |
| `RebootTarget` (gains `Argv`) | `cmd/internal/couchcore/reboot.go` | modified |
| `RecoverySelection` (gains `Argv`) | `cmd/internal/couchcore/recovery_plan.go` | modified |
| `OperationReceipt` / `ReceiptOutcome` (gain `Provenance`) | `cmd/internal/couchmessage/operation.go` | modified |
| `ReceiptProvenance` | `cmd/internal/couchmessage/operation.go` | new |
| `slotOperations` list (+`bench-start`, `bench-retire`) | `cmd/internal/couchcore/slot_operation.go` | modified |

- **HubOwnership.Provisions(caller, target HubSlot-owner)** — the bench authority, a pure function of one ownership snapshot, next to `Wakes`. Returns nil only when: the caller is a hub root (`IsHub`), it is not the main hub (`Main()`), the target is not the caller's own hub slot, `Resolve(target)` names the caller with no `Conflicts`, and the resolution came from the caller's explicit `.hub` claim (exact `repo:N` or whole `repo`) — not from default main-hub ownership and not from a runtime move override. Each refusal is a distinct error naming the reason.
  - **Relationships:** 1 snapshot : N checks; called by the bench guard only.
  - **DRY rationale:** one owner of "may a hub drive this bench slot", read by admission and by the effect gate alike (as `Wakes` is for wake).
  - **Future extensions:** a provenance ledger could later narrow it to slots the hub actually started; the signature stays.
  - To tell claim-derived from override-derived ownership, factor `Resolve` into `resolve(slot) (HubResolution, source)` with a small unexported enum (`ownSlot`, `override`, `claim`, `defaultMain`); `Resolve` keeps its signature.
- **BenchProfile** — `{Agent string; Argv []string}` with `ParseBenchProfile(agent string, argv couchmessage.SwitchArguments-equivalent JSON) (BenchProfile, error)`: agent must be a known launcher agent, argv must be explicit (present, possibly empty) and pass the same validation as switch-agent arguments (`validateSwitchArgumentText` rules, `MaxRepoArgumentBytes`). Lives in couchcore so dispatch and tests share it.
  - **DRY rationale:** the wire form reuses switch-agent's `argv` JSON encoding (`Args["argv"]`, present vs absent preserved), not a new format.
- **RebootTarget.Argv / RecoverySelection.Argv** — `*[]string`: nil keeps today's behavior (saved configuration); non-nil is an explicit launch profile (`ArgvSourceExplicit`) with `Agent` required. Threaded into `freshRecoveryLaunchProfile` and `startFreshSlotWithRecovery`'s profile resolution, so a vacant slot and a live slot launch through the same Fresh transaction. Being a recovery launch, it is never written back to saved parameters.
- **ReceiptProvenance** — `{Root, Tag, Session, Generation string; Agent string; Argv []string; ArgvSource string}` on `OperationReceipt` and `ReceiptOutcome` as `*ReceiptProvenance` (`json:",omitempty"`). Filled only by a succeeded `bench-start` from the `StartResult` (`Record.Args` worktree and session, `Record.Thread.Tag`, the accepted launch profile) and the thread record's start incarnation: Generation is that launch's start nonce (`StartEvent.Nonce`, the value `PAIR_LAUNCH_NONCE` carries), one source only. `ApplyReceiptEvent` copies it on the success transition.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| bench CLI parsing (`--bench-start`, `--bench-retire`) | `cmd/internal/couchcmd/bench_cli.go` | new | argv, broker socket |
| bench guard in hub slot-operation path | `cmd/internal/couchcmd/hub_guards.go` | modified | ownership snapshot, caller identity |
| `bench-start` / `bench-retire` dispatch | `cmd/internal/couchcore/operationdispatch.go` | modified | Reboot, ArchiveThread |
| strict retirement (`RetireBenchSlot`) | `cmd/internal/couchcore/bench_retire.go` | new | quiesce, Teardown proof, archive |
| testcouch `.hub` claims (`Options.HubClaims`) | `cmd/internal/testcouch/fleet.go` | modified | fixture filesystem |

- **bench CLI** — `couch --bench-start repo:N --agent A --agent-args "…" --confirm [--json]` and `couch --bench-retire repo:N --confirm [--json]`; polls with the existing `pollSlotOperation`, prints the receipt (with provenance under `--json`). `--agent-args` is required for bench-start (`""` means explicitly empty argv), split with the same parser as `--switch-agent --agent-args`.
- **bench guard** — in `handleHubSlotOperation`, for the two bench ops: refuse when `authority.hubs == nil` (no hub configuration means no claim can be proven); otherwise the admission and queued guard both run `checkHubSlotOperation` and then `Provisions(callerHubRoot, target)`. A caller that is a worker (not a hub root) refuses.
- **strict retirement** — `RetireBenchSlot(ctx, address)`: `prepareRetirement` (stops the session), then `Teardown`. Survivors are the union of `proof.LeftRunning` and the live entries of `LeftRunningRegistry` (`orphan_registry.go`, `<namespace>/orphans.json`) whose `Thread` is any thread of this slot — teardown computes `LeftRunning` only while the agent is still listed (`teardown.go:188-198, 505-507`), so once a refused retire has stopped the agent, and for orphans of earlier bench-start generations, the registry is the only record. Liveness uses the same `ProcessIdentity` check `Prune` uses (PID alive and start identity unchanged). Any live survivor refuses with a typed `SlotOperationError{Code: "bench-retire-survivors", Detail: <named processes>}` and the record is not archived; otherwise `ArchiveThreadExpected`. A slot with no thread and no live registered orphan is already vacant: succeed with Detail "already vacant". Shares `prepareRetirement`/`Teardown` with `ArchiveThread` (no copy of the stop logic).
- **testcouch `.hub` claims** — `Options.HubClaims map[int][]string` writes `<hub root>/.hub` with the listed claim lines before the console starts, so a native test can make fixture:1 a bench hub claiming fixture:2.

## Tasks

### Task 1: Provisions predicate (pure)

**Files:** Modify `cmd/internal/couchcore/hub_ownership.go`; Test `cmd/internal/couchcore/hub_ownership_provisions_test.go` (new; existing coverage is `hub_ownership_sequence_test.go`).

- [ ] Strategy: one table derived from the ownership-source enum × caller role (main hub, other hub, worker, non-hub); only (other hub, claim-sourced, no conflict) passes, every other cell refuses with its distinct reason.
- [ ] Run `go test ./cmd/internal/couchcore -run TestHubOwnershipProvisions` — FAIL (undefined).
- [ ] Factor `resolve` with an ownership source; keep `Resolve`'s behavior (existing `hub_ownership_test.go` cases stay green); implement `Provisions`.
- [ ] Run the package's hub ownership tests — PASS. Commit `#847: bench ownership predicate`.

### Task 2: Explicit launch profile through Reboot's Fresh path

**Files:** Create `cmd/internal/couchcore/bench_profile.go` (+ `_test.go`); Modify `reboot.go`, `recovery_plan.go`, `recovery_preflight.go`, `recovery_fresh_missing.go`, `slotrecovery.go` (profile resolution only).

- [ ] Test `ParseBenchProfile`: unknown agent refuses; absent argv refuses; explicit empty argv is accepted as `[]`; NUL / oversize refuse.
- [ ] Strategy: `Reboot` with an explicit profile on {live slot, vacant slot}: launches exactly that agent and argv (`ArgvSourceExplicit`) and leaves `repo-preferences.json` byte-identical; `Argv: nil` keeps today's behavior (existing reboot tests unchanged); an explicit argv with no agent refuses before any stop.
- [ ] Run — FAIL. Implement: `RecoverySelection.Argv`; `recoveryLaunchProfileForSelection` and `freshRecoveryLaunchProfile` take the explicit profile and skip saved-configuration lookup when present; `startFreshSlotWithRecovery` gets it through its existing `preflight *LaunchProfileResolution` parameter (build the resolution from the profile) rather than a new parameter.
- [ ] Run couchcore reboot/recovery/launch-profile tests — PASS. Commit `#847: explicit launch profile on Fresh reboot`.

### Task 3: Strict retirement

**Files:** Create `cmd/internal/couchcore/bench_retire.go` (+ `_test.go`); Modify `detach.go` only to share `prepareRetirement` if its visibility needs it.

- [ ] Strategy: drive survivor sources (teardown `LeftRunning`, a live registry entry for this slot from an earlier generation, a dead registry entry, another slot's live entry) × retry (first call, retry with the survivor still alive, retry after it exits). Only no-live-survivor cases archive; a retry with the survivor alive still refuses and keeps the record.
- [ ] Run — FAIL. Implement `RetireBenchSlot` as above. Run — PASS. Commit `#847: strict bench retirement`.

### Task 4: Operations, dispatch, guard, receipts

**Files:** Modify `couchcore/slot_operation.go` (list; `OperationConfirms` true for both), `couchcore/operationdispatch.go` (cases), `couchcmd/hub_guards.go` (bench guard), `couchmessage/operation.go` (`ReceiptProvenance`, copy on success), `couchcmd/slot_operations.go` (`classifySlotOperationOutcome` fills provenance from a bench-start `RebootResult`), `couchtty` only if an op name list needs it.

- [ ] Strategy (in-package message-service fakes as in `hub_move_service_test.go`): the guard is the Task 1 table run through admission and again at the effect, plus two cases only the path has — no hub configuration refuses, and a claim revoked between admission and effect refuses with no effect. Receipts: provenance present only on a succeeded bench-start; nil omitted in JSON; older receipts decode unchanged.
- [ ] Run — FAIL. Implement. Run `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'Bench|Receipt|HubSlot'` — PASS. Commit `#847: bench-start and bench-retire operations`.

### Task 5: CLI

**Files:** Create `cmd/internal/couchcmd/bench_cli.go` (+ `_test.go`); Modify `cli.go` routing, `messages.go` routing, `cmd/internal/couchcmd/skills/couch/SKILL.md` (verbs), help text.

- [ ] Parser tests: `--bench-start fixture:2 --agent claude --agent-args "--model x" --confirm --json` → op/target/agent/argv; missing `--agent-args` refuses; missing `--confirm` refuses; `--bench-retire` takes no agent flags.
- [ ] Run — FAIL. Implement (reuse the `--switch-agent` argv splitting and `pollSlotOperation`). Run — PASS. Commit `#847: bench CLI verbs`.

### Task 6: Native acceptance (claude and codex)

**Files:** Modify `cmd/internal/testcouch/fleet.go` (`Options.HubClaims`); Create `cmd/couch/bench_slots_acceptance_test.go`; add it to `Makefile.local`'s native fleet target regex.

- [ ] Test `TestDisposableCouchBenchSlots` (skips without `PAIR_NATIVE_COUCH_FLEET=1`), per agent: `Workers: 3, ExplicitHub: true, Hubs: []int{1}, HubClaims: {1: {"fixture:2"}}`; `Fleet.Start`; wait for both hubs.
  1. fixture:1 `--bench-start fixture:2 --agent <agent> --agent-args "--fixture-flag one" --confirm --json`: receipt succeeded with provenance (root = fixture:2 worktree, agent, argv `["--fixture-flag","one"]`, source explicit); the fixture's ready record shows that argv.
  2. Again with `two`: new Generation, fixture shows `two`.
  3. `--bench-retire fixture:2 --confirm`: succeeded; fixture:2's agent PID gone; the store shows no live thread for fixture:2; the worktree still exists.
  5. Throughout: `repo-preferences.json` is byte-identical to its value before the first bench-start, and fixture:3's saved launch profile is unchanged.
  4. Refusals: fixture:0 (main hub) `--bench-start fixture:2` refused; fixture:1 `--bench-start fixture:3` (unclaimed production slot) refused and fixture:3 untouched; fixture:1 `--bench-retire fixture:3` refused.
- [ ] Run natively (unsandboxed): `PAIR_NATIVE_COUCH_FLEET=1 go test ./cmd/couch -run '^TestDisposableCouchBenchSlots$' -count=1 -v` — iterate until PASS for both agents. Commit `#847: native bench slot acceptance`.

### Task 7: Docs and close

- [ ] `atlas/couch.md`: a "Bench slots" paragraph (verbs, Provisions rule, strict retire, provenance, the one hand step). Check `atlas/index.md` links.
- [ ] Issue: reword Done-when (bench hub brings up, verifies and retires its claimed worker slots; no Couch restart after the initial start, no console steps); tick Plan; Log.
- [ ] `make test-affected-inventory` (audit each reader), push, `sdlc local-check run --pre-close`, `sdlc close`.

## Verification

- Unit/in-package tests per task; native `TestDisposableCouchBenchSlots` for claude and codex.
- Production untouchable: refusal cases in Tasks 1, 4 and 6.
- Compatibility: existing hub ownership, reboot, switch-agent and receipt tests unchanged and green; `.hub` grammar untouched.
