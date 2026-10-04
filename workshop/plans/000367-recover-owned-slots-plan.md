# Recover Owned Slots Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** After a restart, an agent (typically the TL in `:0`) can read one report that joins this machine's SDLC
claims and slot verdicts with Couch's thread states and says, per row, what to do next. It can then run that step
(`resume` / `reboot`) through the running Couch from its own live slot, and verify the result by reading the report
again.

**Architecture:** Couch gets a new read-only operation, `recover-plan` (`couch --recover-plan-from-sdlc`). It runs
`sdlc fleet inventory --json` once per fleet of the repositories Couch has enrolled, through the existing
`ProvisionIO` seam, and reads Couch's actionable inventory the same way `couch --list` does. A pure
`DeriveRecoverPlan` joins the two into one row per slot path, using the union of evidence. Each row gets a class, git, disk and
agent state, and next steps. The resume/reboot steps come from `ActorActions`, which is extracted from the switcher's
action table so the report and the switcher share one source. M2 adds `resume` and `reboot` requests to the Couch
message socket. They accept only the `--send-to` caller, check a declared confirmation, and return a typed in-memory
receipt that the CLI polls. The work itself runs through the console's `operationQueue`, `c.ops` and
`finishOperation`, the same path a switcher keypress takes.

**Tech Stack:** Go 1.26. Existing fakes: couchcore `newTestEnv`, `slotRecoveryOperationFixture`, `FakeProcOps`, the
artifact fake. Also couchcmd `newRT` and `serviceRig`, and the couchtty console fixtures (`New(hostty.NewFakeHost…)`).
One new stateful fake: `FakeFleetSDLC`.

**Source of truth for scope:** `workshop/issues/000367-recover-owned-slots.md`, its `## Done when` (rewritten
2026-10-03), and the three Revisions dated 2026-10-03. Those override every earlier Revision. Out of scope: step 4
(TL scheduling, notifications, preparing N slots, workspace shaping → pair#362), callers from outside Couch, a
one-shot "execute the plan" command, and repairing deleted slots (pair#387; the report only shows them).

---

## Milestones

| Milestone | Boundary | Leaves behind |
|---|---|---|
| **M1** | Read-only report: the sdlc v1 decoder with a golden fixture and live conformance, `SDLCFleetSource` with `FakeFleetSDLC`, `ActorActions` extracted, pure `DeriveRecoverPlan`, the `recover-plan` operation and CLI, the restart acceptance test, docs | An agent can read the plan. Steps name actions only (`resume`/`reboot`/`ask-agent-restore`) and carry no CLI text, because `--resume`/`--reboot` do not exist yet |
| **M2** | Socket primitives: protocol ops, a pure receipt state machine, the console's remote enqueue, `PrepareSlotOperation`, the caller rule, `--resume`/`--reboot` CLI, step `command` text, the skill section, the end-to-end recovery acceptance test | The full LLM-driven loop: read the report, act, read it again |

## Resolved spec ambiguities (operator: confirm or correct)

1. **(Confirmed) Report scope = the fleets of Couch-enrolled repositories.** One `sdlc fleet inventory` runs per
   distinct `fleet_root`, as reported by `sdlc workspace --json`, with rows deduplicated by slot address. The result
   does not depend on cwd.
2. **(Confirmed) JSON only, with no human view.**
3. **Failed sources degrade the report; they do not abort it** (operator, 2026-10-04). The quality of each source
   is recorded per row, and no repository is held because one source is weak.
   - sdlc fails, times out, overflows 1 MiB, reports a `schema_version` ≠ 1, or is an older v1 build missing any of
     `slots`/`machine`/`dangling_claims`/`rows[].claims_state` (#288/#289 added them without a bump): the fleet's state
     becomes `unavailable`/`unsupported`, and it contributes no claim or slot evidence. It never reads as zero slots.
   - Disk evidence for that fleet's slots comes from Couch's own existing probe, `ProbeSlotGit` (`slotgit.go`, one
     `git status --porcelain=v2 --branch` per slot, the function the switcher polls every 10 s). The report runs in
     the CLI process, which has no cached console observation, so it calls that same function. This is reuse, not a
     second scanner. Each probe is bounded by the console's own `slotGitProbeTimeout` (3 s), which moves to
     couchcore as `SlotGitProbeTimeout` so both callers share it. The probe yields branch, detached and dirty, but not
     in-progress operations or ahead-of-main, so those dimensions stay unknown (`partial-evidence` /
     `evidence-unavailable`, never absence).
   - If the Couch store read fails, every row's agent becomes unknown.
   - Exit 0 whenever a report renders.
4. **`claims_state` is per-source quality, not a hold** (operator).
   - `stale`: steps are still suggested, and the row is marked `claims.quality: stale` with sdlc's `claims_error`. The
     agents' own sdlc verbs re-check ownership against the live tracker. sdlc v1 carries no fetch timestamp, so the row
     shows `claims stale: <claims_error>`, with no time and no ariadne change (operator, 2026-10-04).
   - `partial`/`unknown`: the claims that were read are used together with the other evidence, and the row names the
     missing or partial source.
5. **Union of evidence replaces "foreign/unattributed"** (operator). Work evidence in a slot is any of:
   - this machine's claim on one of the slot's paths;
   - a checked-out issue branch (sdlc `issues[]`, `provenance: branch-prefix`);
   - unlanded commits;
   - dirty or untracked files;
   - a Couch thread with a conversation (any joined row except unusable `never-started`).

   The rows that result:
   - **Agreeing** evidence → `agrees`, with a step.
   - **Work but no visible claim, claims fully read** → `claim-likely-lost`: still suggest resume, plus the note
     `claim-repair`. v1 omits other machines' claims and ownerless cards, so the note says: first run
     `sdlc issue show N --json`. If a repair is needed, the **slot's own agent** (not the TL) runs
     `sdlc claim --issue N --adopt` or `sdlc reclaim`, under operator direction. The report runs neither.
   - **Claims not fully read** → `partial-evidence`: resume only.
   - **Conflicting** evidence → `conflict`, with no step and the facts listed:
     - `claim-elsewhere`: #N's branch is here, but its claim sits on another slot path;
     - `issue-terminal`: the branch names a done issue;
     - `claim-branch-mismatch`: a claim for A while B's branch is checked out and B is unclaimed;
     - `claim-workspace`: a host-member claim whose non-empty `claimant.workspace` ≠ the address.

     ariadne `claimant.go:42-66` records an empty workspace for dependency clones and plain clones, so an empty
     workspace is never a mismatch.
6. **Dirty files do not block resume** (operator; primitives never touch disk).
   - Dirty on the active issue branch → resume. If resume is not offered → reboot, with the note
     `inspect-uncommitted-first`.
   - An in-progress git operation or detached HEAD → resume only, never reboot. With no resume offered, the row gets
     `no-safe-step`.
   - Exactly one claim on the resting branch **and dirty** → resume only, with note `resting-branch-dirty`: "no restore
     request: switching branches with uncommitted files can fail or carry the edits across". Operator confirmed.
   - Work on **any** non-issue branch (resting, another branch, or detached) with no claim → `unidentified-work`, no
     step. Example: `42shots:0` on `cast-embed`, dirty.
   - A claimed slot with a detached HEAD → `agrees` with note `detached-head`: resume only, never reboot.
7. **Rows are per slot path only** (operator): `:0` and `:1+`, per Couch's slot layout. Claims attach to the path
   they sit on (any member of the slot). A claim whose conventional slot path no longer exists (an sdlc dangling
   claim that `conventionalSlotFromPath` recognizes) gets that slot's row as `directory-missing`. Everything off-slot
   is ignored and only counted in `ignored` (`off_slot_claims`, `non_slot_threads`): non-slot worktrees, claims on
   them, and ordinary threads that are not the primary row (`IsPrimaryRow`, extracted from `ApplyRepositoryAliases`).
   A slot-target Couch row that the healthy fleet lacks is `directory-missing` if its path is gone, otherwise
   `partial-evidence`.
8. **CLI shape:** `couch --resume repo:N [--json]` and `couch --reboot repo:N --confirm [--json]`. These take an exact
   address only, `:0` included. A repository family is refused because a primitive needs one slot. The repo part
   resolves the way `--send-to` does: name, alias or unique prefix. `--confirm` is required only because the `reboot`
   declaration says `ConfirmRequired`; the handler reads `couchcore.OperationConfirms`.
9. **(Confirmed) The socket admits, then the CLI polls.** Both primitives are #280 class `duplicate-safe-refusal`: once the slot is
   live, a repeat is refused (`not-offered`). One caveat: a reboot whose fresh launch failed leaves the slot
   record-less, so a repeat reboot is offered again and archives nothing more, but it starts a second fresh record. The transport has a hard 2 s deadline per exchange
   (`couchmessage.TransportTimeout`), and a resume can take longer. Admission returns a receipt ID, and the CLI polls
   `operation-status` until the receipt is terminal or 3 minutes pass. Receipts live in memory and are visible only to
   the slot that admitted them. A Couch exit loses them. After a lost or uncertain outcome, the skill says: read the
   report again. Resending is refused harmlessly once the slot is live, because resume and reboot are not offered on
   live rows.
10. **(Confirmed) Admission is checked when the queued job runs, against fresh inventory.** It uses the same `ActorActions` the
    switcher offers from, so whatever the socket accepts, the switcher would also offer on that row. The executors
    re-classify at action time, as they do today.
11. **(Confirmed) A remote start runs in the background; the operator's screen stays put.** It uses `PreserveFocus` with `Attempt == 0`, like continuation
    replacements; no new origin field is added. It never clobbers the operator's in-flight switcher operation, and a
    remote resume clears the row's reattach-failure mark (`clearReattachFailure`) exactly as the switcher's resume
    does.
12. **Multiple claims** (operator). The active claim is the one whose issue branch is checked out, matched by
    branch-name issue prefix. Other claims are listed as inactive (note `inactive-claims`) with no step. A resting
    branch with more than one claim → `ambiguous-claims`, no step. Exactly one claim on a clean resting branch →
    `restore-workspace`: `[resume|reboot] + ask-agent-restore`, or `ask-agent-restore` alone when live. The message is
    a single constant.
13. **Busy = "a start not yet reconciled"** (operator): a persisted start claim from a possibly crashed Couch that
    startup reconciliation could not decide. → `start-unreconciled`, no step, re-read later, never reboot (it could
    duplicate a live agent). Unusable-unknown → `agent-unknown`, no step, re-read later.
14. **A conversation alone** (clean, resting branch, no claim, no issue branch) → `idle`, no step (operator). The row
    still shows the agent state, so the TL may resume it if wanted.

## ARCH-* notes

- **ARCH-DRY:** `ActorActions` moves the resume/reboot half of `menuRowActions` into couchcore. The switcher, the
  report's steps and the socket's admission all read it. `ActorOperationArgs` moves the row → args mapping out of
  `dispatchMenuRow`, including the `warm-only=true` rule for a detached ordinary row. That rule is now in
  `dispatchMenuOperation`, `menu.go:1797`, and moves out of it. `IsPrimaryRow` is extracted from
  `ApplyRepositoryAliases` and used by both it and the join. The sdlc subprocess reuses `ProvisionIO`/`OSProvisionIO` (process group, timeout, 1 MiB cap),
  and the queue path reuses `operationQueue`/`c.ops`/`finishOperation`. Disk verdicts are sdlc's
  (`slots[].verdict`/`reasons`). Couch adds no git scanner of its own. Its only git read here is the existing
  `ProbeSlotGit`, called for an unavailable or unsupported fleet's slots.
- **ARCH-PURE:** `DecodeFleetInventory`, `DeriveRecoverPlan`, `ActorActions`, `SelectSlotRow`,
  `ActorOperationArgs`, `SlotOperationCommand` and `ApplyReceiptEvent` are pure and table-tested with no fakes. The IO
  shells are `SDLCFleetSource.FleetInventory`, `Couch.RecoverPlan`, `Couch.PrepareSlotOperation`,
  `Console.EnqueueRemoteOperation` and the message-service handler.
- **ARCH-PURPOSE:** Done-when asks for "every report row class". The fixtures are checked against the derived union
  `AllRecoverClasses() ∪ AllRecoverHolds() ∪ AllRecoverNotes()`, one fixture per code, so the six-rule classes cannot
  hide behind one fixture. The claim "every emitted step is reachable" is checked by
  parsing each emitted `command` with `ParseCLI` and running each resume/reboot step through `ActorActions`.
- **ARCH-MOCK:** sdlc is an external binary. `FakeFleetSDLC` is a stateful fake behind `ProvisionIO`. It models
  fleets → slots → members (branch, dirty, operation, ahead, issues, claims), dangling claims, machine and
  claims_state, and failure modes. It computes verdicts with sdlc's precedence. A golden fixture captured from real
  output, plus a live conformance test (skipped when there is no `sdlc` on PATH), keep it honest.
- **ARCH-CONSTRAINTS:** The report is a batch CLI call that runs in the **CLI process**, like `--list`, so it never
  touches the console's UI path. It shares `--list`'s evidence gather (`gatherThreadEvidence`), but uses the
  actionable projection (`ActionableThreadInventoryContext(ctx, nil)`), not `ThreadInventoryContext`. It measured 5.7 s for 52 worktrees and 39 slots (2026-10-03). Each fleet gets a 90 s
  timeout, because sdlc's own claim reads are 15 s each with at most 8 concurrent. Fleets run sequentially, normally 1
  and capped at 8, and stdout is capped at 1 MiB per fleet. Socket handlers do only O(1) work inside the 2 s budget;
  the inventory read and the action run on the console queue (capacity 16, one runner). At most 64 receipts are held
  and the 65th admission is refused `overloaded`. The CLI polls every 500 ms.
- **ARCH-SECURE:** sdlc output is untrusted input. It is decoded once, at `DecodeFleetInventory`. The version is
  checked first, unknown verdict and reason strings become `unknown`, and paths are normalized before joining.
  Socket callers are authenticated exactly as for `--send-to` (`Broker.Caller` + `authority.current`). Targets are
  re-resolved inside Couch and never trusted as paths. No credentials are involved.
- **ARCH-ORDER / concurrency:** `slotOperations` owns one `sync.Mutex` guarding the receipt map. Every touch goes
  through its methods, from three goroutines: admit and status from socket handlers, `started` from the queue runner,
  and `finished` from the console loop. Tests run under `-race`. The receipt machine has states `queued → running → succeeded|refused|failed`. Its events are admit,
  duplicate-admit (same ID → same receipt; a different op or target → `id-conflict`), start, finish, expire and
  couch-exit. It is enumerated in Task 2.1 and sequence-tested. Couch-exit drops every receipt, so a later status call
  answers `unknown` → the CLI reports "uncertain; verify with the report".
- **ARCH-FUNERAL:** **The report creates nothing durable.** It writes only stdout. The one side effect is sdlc's own,
  documented in its RECOVERY contract: each tracker fetch updates that repository's remote-tracking ref, and the
  next fetch supersedes it. M2's receipts are in-memory only. Each one is created on admission and removed 5 minutes
  after it turns terminal (swept on every admit and status call), or when Couch exits. The cap is 64, about 1 KiB
  each. Queue entries are removed when they complete. Reboot's archive records keep #363's lifecycle (60-day
  `archive_gc.go`).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `FleetInventory` (+ `FleetRow`, `FleetSlot`, `FleetMember`, `FleetClaim`, `FleetMachine`) / `DecodeFleetInventory` / `FleetSchemaVersion` | `cmd/internal/couchcore/recoverplan_fleet.go` | new |
| `RecoverPlanInput` / `FleetObservation` / `CouchObservation` | `cmd/internal/couchcore/recoverplan.go` | new |
| `RecoverPlan` / `RecoverRow` / `RecoverGit` / `RecoverDisk` / `RecoverAgent` / `RecoverNext` / `RecoverStep` | `cmd/internal/couchcore/recoverplan.go` | new |
| `RecoverHold` / `AllRecoverHolds` (hold-code kinds; parameterized codes are `kind:suffix`) | `cmd/internal/couchcore/recoverplan.go` | new |
| `SlotEvidence` (+ `Offer` dimension) / `slotEvidenceOf` / `classifyRecover` / `consistent` / `RecoverNote` / `AllRecoverNotes` / `RecoverPlan.Ignored` | `cmd/internal/couchcore/recoverplan.go` | new |
| `RecoverClaims` / `RecoverFleet` / `RecoverCouch` / `RecoverIgnored` / `RecoverSlotCandidate` / `RecoverLocalGit` | `cmd/internal/couchcore/recoverplan.go` | new (Revision (c)) |
| `IsPrimaryRow` (extracted; `ApplyRepositoryAliases` uses it) | `cmd/internal/couchcore/actionableinventory.go` | new (extracted) |
| `strictjson.RejectDuplicateKeys` (Decode's structural check, exported for the fleet decoder) | `cmd/internal/strictjson/decode.go` | new (Revision (c)) |
| `RecoverClass` / `AllRecoverClasses` / `DeriveRecoverPlan` | `cmd/internal/couchcore/recoverplan.go` | new |
| `RestoreWorkspaceMessage` | `cmd/internal/couchcore/recoverplan.go` | new |
| `ActorRowFacts` / `ActorRowFactsOf` / `ActorActions` | `cmd/internal/couchcore/actor_actions.go` | new (extracted from `couchtty/menu_actions.go`) |
| `menuRowFacts` / `menuRowFactsOf` / `menuRowActions` (non-live phases delegate to `ActorActions`) | `cmd/internal/couchtty/menu_actions.go` | modified |
| Operation `recover-plan`, `PresentationRecoverPlan`, `ResultRecoverPlan` | `cmd/internal/couchcore/ops.go` | new |
| `SelectSlotRow` / `ActorOperationArgs` / `SlotOperationError` (codes) | `cmd/internal/couchcore/slot_operation.go` | new (M2) |
| `dispatchMenuRow` / `dispatchThreadOperation` (use `ActorOperationArgs`); the warm-only block in `dispatchMenuOperation` is deleted | `cmd/internal/couchtty/menu_slot.go:30`, `menu.go:1775,1797` | modified (M2) |
| `SlotOperationCommand` | `cmd/internal/couchcore/slot_operation.go` | new (M2) |
| `OperationReceipt` / `ReceiptStatus` / `ReceiptEvent` / `ApplyReceiptEvent` | `cmd/internal/couchmessage/operation.go` | new (M2) |
| `Request.Confirmed`, `Response.Operation`, `ValidateRequest` (`resume`, `reboot`, `operation-status`) | `cmd/internal/couchmessage/protocol.go` | modified (M2) |
| `ParseCLI` / `parseMessageCLI` (`--recover-plan-from-sdlc`, `--resume`, `--reboot`) | `cmd/internal/couchcmd/cli.go` | modified |

- **DeriveRecoverPlan** — `(RecoverPlanInput) RecoverPlan`. It joins fleet slots to Couch rows by normalized path. A
  slot row matches when `Target.Slot.WorktreeRoot` equals the host member path. An ordinary row matches `:0` only when
  `IsPrimaryRow(row, hostPath, scopeKey)` holds; `scopeKey` is `launcher.ResolveRepoScope(hostPath)`, a pure hash the
  derivation computes itself (Revision 2026-10-04 (c): no `PrimaryScopes` input). Each row's class is then picked by
  the first match in the table in Task 1.5.
  - **Relationships:** 1 fleet slot → 1 row with 0..N Couch threads (more than one → `ambiguous-threads`). 1 claim → 1
    row: its slot's row, or its own row.
  - **DRY rationale:** It is the only place the two observations are joined. The steps come from `ActorActions`, not
    from a second table.
  - **Future extensions:** The deterministic "execute the plan" command (deferred) consumes `RecoverRow.Next`
    unchanged. pair#387 adds a `repair-slot` step to the same step vocabulary.
- **ActorActions** — the per-kind admission table for `resume`/`reboot` over (kind, state, reason, unfinished
  continuation, recovery verdict). Its body is exactly the current `menuPhaseResumable`/`menuPhaseUnusable` arms.
- **ApplyReceiptEvent** — `(OperationReceipt, ReceiptEvent) (OperationReceipt, error)`, a closed transition table.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `FleetInventorySource` / `SDLCFleetSource` | `cmd/internal/couchcore/recoverplan_source.go` | new | `sdlc fleet inventory --json` via `ProvisionIO` |
| `FakeFleetSDLC` / `FakeFleet` (per-fleet mutators, `SetSchema`) | `cmd/internal/couchcore/recoverplan_fake.go` | new | stateful sdlc fleet model behind `ProvisionIO` |
| `SlotGitProbeTimeout` (moved from couchtty) | `cmd/internal/couchcore/recoverplan_source.go` | moved | — |
| `Couch.Fleet` field, `Couch.RecoverPlan` | `cmd/internal/couchcore/couch.go`, `recoverplan_source.go` | new | thread store + fleet source |
| `DirectStoreExecutor` `recover-plan` case | `cmd/internal/couchcore/operationdispatch.go:141` | modified | — |
| `OSRuntime.NewCouchWith` (sets `c.Fleet`) | `cmd/internal/couchcmd/run.go:169` | modified | — |
| `render` (JSON for `RecoverPlan`) / `usageWith` | `cmd/internal/couchcmd/run.go:800,954` | modified | stdout |
| `Couch.PrepareSlotOperation` | `cmd/internal/couchcore/slot_operation.go` | new (M2) | inventory, `WorkspaceReferencePath` |
| `Console.EnqueueRemoteOperation`, `operationRequest.finished` | `cmd/internal/couchtty/console_remote.go`, `operation_queue.go`, `console.go:1787` | new / modified (M2) | console operation queue |
| `slotOperations` (receipt store + handler) | `cmd/internal/couchcmd/slot_operations.go` | new (M2) | message socket |
| `messageService.handle` / `newMessageService` / `startMessageService` | `cmd/internal/couchcmd/message_service.go:165,291,540` | modified (M2) | — |
| `runMessageCLIWithCall` (admit + poll) | `cmd/internal/couchcmd/messages.go` | modified (M2) | — |
| Couch skill recovery section | `cmd/internal/couchcmd/skills/couch/SKILL.md` | modified (M2) | — |

- **`ProbeSlotGit` reuse** — `cmd/internal/couchcore/slotgit.go:79`, unchanged. `Couch.RecoverPlan` calls it only for slots
  of an unavailable or unsupported fleet; the result is passed in as `RecoverPlanInput.LocalGit`.
- **SDLCFleetSource** — `FleetInventory(ctx, vantage) ([]byte, error)` runs
  `ProvisionCommand{Dir: vantage, Program: "sdlc", Args: ["fleet","inventory","--json","--path",vantage], Timeout:
  90s}`.
  - **Injected into:** `Couch.RecoverPlan`, which passes the bytes to `DecodeFleetInventory`. Tests use `FakeFleetSDLC`
    as the `IO`, so the argv construction is exercised too.
  - **Future extensions:** a `--json` field from sdlc's #288/#289 successors; a schema bump is an explicit decoder
    change.
- **FakeFleetSDLC** — state is `map[fleetRoot]*fakeFleet{slots, offSlotRows, dangling}`, plus `Machine`, `Schema int`,
  and `Fail` (exit error, hang until ctx, or garbage bytes). Its mutators are `AddSlot`, `SetDirty`, `SetOperation`,
  `SetBranch`, `SetAhead`, `Claim`, `Release`, `RemoveSlot` (which turns its claims dangling), `SetClaimsState` and
  `MissingMember`. Verdicts follow sdlc's `JudgeCheckout` precedence and `Worst` fold
  (ariadne `cmd/sdlc/internal/fleet/slots.go:58-115`). `Calls` records argv.

---

## Chunk 1: M1 — the read-only report

Every `go`/`make` command runs with the env scrub as a literal prefix (zsh does not word-split a variable):
`env -u PAIR_DATA_DIR -u PAIR_TAG -u PAIR_RETENTION_PROTOCOL -u PAIR_RETENTION_BACKGROUND -u PAIR_RETENTION_START_ID`.
Below it is abbreviated `SCRUB`.

### Task 1.1: The sdlc v1 decoder, golden fixture and live conformance

**Files:**
- Create: `cmd/internal/couchcore/recoverplan_fleet.go`, `recoverplan_fleet_test.go`,
  `cmd/internal/couchcore/testdata/sdlc_fleet_inventory_v1.json`,
  `cmd/internal/couchcore/testdata/sdlc_fleet_inventory_v1_pre288.json` (a real v1 document from before #288/#289:
  no `slots`, `machine`, `dangling_claims` or `claims_state`; build it from the same capture by deleting those keys)

- [x] **Step 1: Capture the golden fixture.** Run
  `/Users/xianxu/workspace/worktree/pair-slot1/ariadne/bin/sdlc fleet inventory --json > $SCRATCH/fleet.json`. Trim it
  to: one `:0` ready slot, one `:N` holds-work slot with a `claims[]` entry and a dependency member, one
  `needs-recovery` (`dirty`), one `missing` member, a `diagnostics[]` entry, and one `dangling_claims[]` entry (hand
  added, shape from ariadne `fleet/claims.go:78-83`). Also include one claim on a **dependency member** with no
  `workspace` key; this is the real shape per `claimant.go:42-66`. Rewrite paths to `/fleet/...`.
- [x] **Step 2: Write the failing tests.**
  - `TestDecodeFleetInventoryGolden`: `ReadFile` errors are checked. The golden decodes with machine `present`, at
    least one slot, and exactly 1 dangling claim.
  - `TestDecodeFleetInventoryRejectsUnsupported`: each case must return `errors.Is(err, ErrFleetSchemaUnsupported)`,
    not just any error. The cases are `schema_version` 2, missing, or the string `"1"`; v1 without `slots`; v1
    without `machine`; v1 without `dangling_claims`; v1 with a row lacking `claims_state`. Malformed input (`[]`,
    empty, a duplicate key) must return a non-nil error that is *not* `ErrFleetSchemaUnsupported`.
  - `TestDecodeFleetInventoryPre288IsUnsupportedNotEmpty`: the pre-#288 golden → `ErrFleetSchemaUnsupported`.
  - `TestFleetInventoryLiveConformance` (skips without `sdlc` on PATH): runs the real command through `OSProvisionIO`
    and decodes it. Every verdict must be in `knownFleetVerdicts`, and every reason must match `knownFleetReason`.
    This pins the fake's vocabulary to the real producer.
- [x] **Step 3:** `SCRUB go test ./cmd/internal/couchcore -run 'FleetInventory' -count=1` → FAIL (undefined).
- [x] **Step 4: Implement.** First probe `schema_version` alone (`struct{ V *int }`) and return
  `ErrFleetSchemaUnsupported` unless it is exactly 1. Then do a presence probe: top-level `slots`, `machine` and
  `dangling_claims`, and each row's `claims_state`, are `json.RawMessage`/pointer fields. Any one absent →
  `ErrFleetSchemaUnsupported` ("sdlc predates #288/#289; upgrade the slot's sdlc"). Then decode the full struct with `encoding/json`, which accepts
  unknown fields because additive v1 fields are allowed. Use `strictjson.Decode`'s duplicate-key check. The verdict
  vocabulary is `knownFleetVerdicts = {ready, holds-work, unknown, missing, needs-recovery}`. The reason grammar is
  `dirty | detached | missing | unlanded-commits | operation:<x> | open-issue:<ref> | claimed:<ref> | probe:<x>`. An
  unknown verdict decodes but is treated as `unknown` downstream (Task 1.5).
- [x] **Step 5:** Run → PASS. Commit `#367 M1: couchcore: decode sdlc fleet inventory v1`.

### Task 1.2: `ActorActions` extracted from the switcher table

**Files:**
- Create: `cmd/internal/couchcore/actor_actions.go`, `actor_actions_test.go`
- Modify: `cmd/internal/couchtty/menu_actions.go:35-90,125-140`

- [x] **Step 1: Failing test.** Pin a literal table written from #363's Spec (not from the code): `{kind, state,
  reason, unfinished, recover} → want`. Parked/detached → `[resume reboot]`. Unusable/unknown → nil. Unusable
  `path-missing` slot → nil, and primary → `[reboot]`. Unusable slot → `[resume reboot]`. Unusable primary with
  recover or an unfinished request → `[resume reboot]`, otherwise `[reboot]`. Live/busy → nil. Iterate over
  `AllThreadStates() × AllThreadReasons() × {primary, slot} × {"", pending, running, failed} × {false, true}`, and
  fail any combination the literal table does not cover (derived enumeration).
- [x] **Step 2:** Run → FAIL. **Step 3:** Move the resumable/unusable arms into `ActorActions(ActorRowFacts)`, and
  `ResumeOffered`/`DirectoryMissing` derivation into `ActorRowFactsOf(ActionableThreadSummary)`. `menuRowFacts` gains
  `Actor couchcore.ActorRowFacts`, and the two `menuRowActions` arms become `return couchcore.ActorActions(f.Actor)`.
  `menuRowAdviceOf` keeps reading `f.DirectoryMissing` through `f.Actor`.
- [x] **Step 4:** `SCRUB go test ./cmd/internal/couchcore ./cmd/internal/couchtty -count=1` → PASS, including
  `TestActionOfferedImpliesPermitted` and `TestRowAdviceNamesOnlyReachableActions`, unchanged. Mutation: make
  `ActorActions` return `[resume]` for unusable primaries, and confirm both packages fail.
- [x] **Step 5:** Commit `#367 M1: couchcore: ActorActions, the one resume/reboot admission table`.

### Task 1.3: `FakeFleetSDLC` and `SDLCFleetSource`

**Files:**
- Create: `cmd/internal/couchcore/recoverplan_source.go`, `recoverplan_fake.go`, `recoverplan_source_test.go`
- Modify: `cmd/internal/couchcore/couch.go` (field `Fleet FleetInventorySource`),
  `cmd/internal/artifactpath/manifest.go` (`NonArtifactSources` += the five M1 `.go` files)

- [x] **Step 1: Failing tests.**
  - `TestSDLCFleetSourceArgv`: the fake records `{Dir: v, Program: "sdlc", Args: [fleet inventory --json --path v],
    Timeout: 90s}`.
  - `TestFakeFleetSDLCIsStateful`: `AddSlot("pair:1")` gives verdict `ready`; `Claim` makes it `holds-work` with
    `claimed:pair#7`; `SetDirty(1)` makes it `needs-recovery [dirty]` with the claim still listed; `RemoveSlot` moves
    the claim to `dangling_claims`; `Schema=2` makes the output fail to decode.
  - `TestFakeFleetSDLCVerdictsMatchSDLCPrecedence`: a table copied from ariadne `TestJudgeCheckout` cases. Note the
    source commit.
- [x] **Step 2:** Run → FAIL. **Step 3:** Implement. The fake's `Run` refuses any program other than `sdlc` and any
  other argv. It marshals through the same `FleetInventory` types it is decoded with, plus `schema_version` from
  `Schema`.
- [x] **Step 4:** Run → PASS. `SCRUB go test ./cmd/internal/artifactpath -count=1` shows the same 33 pre-existing
  failures as main, with no new failures (if the list differs, compare against main in a scratch worktree). Commit `#367 M1: couchcore: SDLC fleet source and its stateful fake`.

### Task 1.4: Couch observation and `Couch.RecoverPlan`

**Files:** Modify `cmd/internal/couchcore/recoverplan_source.go`; test `recoverplan_source_test.go`

- [x] **Step 1: Failing tests.** `TestRecoverPlanRunsOneInventoryPerEnrolledFleet`: enroll two repositories in one
  fleet and one in another, and expect exactly two fake calls with the first primary of each fleet as the vantage.
  `TestRecoverPlanReadsAFleetOnceFromTwoVantages`: two enrolled primaries in one fleet resolve the same
  `fleet_root`; expect one sdlc call. If the fake is forced to answer both vantages with the same document, every
  slot address must appear in exactly one row (the dedupe backstop, the real defense). A nested primary is NOT a
  case to fixture: `WorkspaceIdentity` validation (`workspace_identity.go:89`) refuses an ordinary identity whose
  `filepath.Dir(PrimaryRoot) != FleetRoot`, so through the real resolver its probe fails. Test that instead: a
  primary whose workspace probe fails yields an `unavailable` fleet observation for it, never zero rows.
  `TestRecoverPlanDegradesPerSource`: one fleet unsupported → its slots come from enrolled primaries and
  `EnumerateSlotCandidates`, carrying `ProbeSlotGit` observations (via `env.Git`, the fake `GitRunner`); the other
  fleet is unaffected. A store error makes `couch.state = unavailable`. Neither error is returned.
- [x] **Step 2:** Run → FAIL. **Step 3:** Implement
  `func (c *Couch) RecoverPlan(ctx context.Context) (RecoverPlan, error)`:
  1. Read `c.Threads.RepositoryNames()` (an error here is the only returned error). For each primary, resolve its
     `fleet_root` with the slot catalog's existing `sdlc workspace --json` probe (`SlotWorkspaceResolver.
     ResolveWorkspace`; `WorkspaceIdentity.FleetRoot`). A primary whose probe fails becomes its own observation with
     `state: unavailable`. Group by fleet root in sorted order, capped at 8 fleets. Each extra fleet becomes an observation with `state: unavailable, error:
     "fleet limit"`.
  2. For each fleet: `c.Fleet.FleetInventory` → `DecodeFleetInventory` → a `FleetObservation`.
  3. `c.ActionableThreadInventoryContext(ctx, nil)` → a `CouchObservation`. It shares `--list`'s evidence gather
     (`gatherThreadEvidence`, positive-only, safe while a console runs), but takes the actionable projection.
  4. Resolve `PrimaryScopes[hostPath]` with `launcher.ResolveRepoScope` for each fleet `:0` host.
  4b. For an unavailable or unsupported fleet only, gather that fleet's `SlotCandidates`: its enrolled primaries plus
      `EnumerateSlotCandidates`. That function does filesystem IO, so it runs here in the shell, never inside
      `DeriveRecoverPlan`. Then, per path, call `ProbeSlotGit` under `context.WithTimeout(ctx, SlotGitProbeTimeout)`
      (moved from `couchtty/console_slotgit.go:22`; the console then uses it) and store the result in
      `LocalGit[path]`. A probe error or timeout is recorded as unknown.
      The cost is bounded by that fleet's slot count, at about 10 ms each.
  5. `return DeriveRecoverPlan(input), nil`. `DeriveRecoverPlan` also deduplicates slots by address, first fleet wins,
     and records a duplicate in that fleet's `error`.
- [x] **Step 4:** Run → PASS. Commit `#367 M1: couchcore: RecoverPlan gathers both observations`.

### Task 1.5: `DeriveRecoverPlan` (pure)

**Files:** Create `cmd/internal/couchcore/recoverplan.go`, `recoverplan_test.go`; extract `IsPrimaryRow(row,
primaryRoot, scopeKey) bool` from `ApplyRepositoryAliases:737`, which then calls it.

**Slot universe.** The universe is the union of:
- fleet `slots[]`;
- for an unavailable or unsupported fleet, `RecoverPlanInput.SlotCandidates`, gathered by the shell (Task 1.4);
- Couch slot-target rows;
- dangling-claim paths that `conventionalSlotFromPath` recognizes.

Rows are deduplicated by address. Couch rows join by `Target.Slot.WorktreeRoot` (`:N`) or `IsPrimaryRow` (`:0`).
Anything else is counted in `ignored`. `DeriveRecoverPlan` does no IO.

**Evidence per slot** (`SlotEvidence`, pure). Every dimension is closed, and `unknown` is a value, never absence.

| Dimension | Values |
|---|---|
| `Dir` | present, missing |
| `Couch` | ok, unavailable |
| `Agent` | none, live, detached, parked, busy, unusable, unusable-unknown |
| `Threads` | 0, 1, many |
| `Branch` | resting, open-issue, terminal-issue, other, detached, unknown (local probe: an issue prefix → open-issue, status unknown) |
| `Claims` | none, one-active (its issue is the branch), one-inactive, many-with-active, many-without-active |
| `Quality` | present, stale, partial, unknown, absent, unsupported |
| `Dirty`, `Unlanded`, `Operation` | yes, no, unknown |
| `Workspace` (host claim workspace ≠ address) | yes, no |
| `Elsewhere` (the branch issue's claim sits on another slot path) | yes, no |
| `GitSource` | sdlc, local-probe, unknown |

A healthy fleet whose member verdict is `unknown` (`probe:*`) sets the unread dimensions to `unknown`.

Rules (first match). `AllRecoverClasses()` follows this order:

| # | Condition | Class | Steps | Hold / notes |
|---|---|---|---|---|
| 1 | `Dir` missing | `directory-missing` | — | hold `directory-missing` (pair#387) |
| 2 | `Couch` unavailable | `agent-unknown` | — | hold `couch-unavailable` |
| 3 | `Agent` busy | `start-unreconciled` | — | hold `start-unreconciled` (re-read later; never reboot) |
| 4 | `Agent` unusable-unknown | `agent-unknown` | — | hold `agent-unknown` (re-read later) |
| 5 | `Threads` many | `ambiguous-threads` | — | hold `threads:<n>` |
| 6 | conflict: `Elsewhere`; `Branch` terminal-issue; `Workspace`; a claim with `Branch` other; claims present and `Branch` open-issue not claimed while `Quality` ∈ {present, stale} | `conflict` | — | hold `conflict:<claim-elsewhere\|issue-terminal\|claim-workspace\|claim-off-branch\|claim-branch-mismatch>` (all that apply) |
| 7 | `Claims` many-without-active | `ambiguous-claims` | — | hold `ambiguous-claims` |
| 8 | no claim, `Branch` ∉ {open-issue}, and any of `Branch`/`Dirty`/`Unlanded`/`Operation` unknown | `evidence-unavailable` | — | hold `git-unknown` |
| 9 | no claim, `Branch` resting, `Dirty`/`Unlanded`/`Operation` all no | `idle` | — | — (a conversation is shown, not acted on) |
| 10 | no claim, `Branch` ∈ {resting, other, detached}, with work (dirty, unlanded, operation, or a non-resting branch) | `unidentified-work` | — | hold `unidentified-work` |
| 11 | claims or `Branch` open-issue, `Agent` none | `no-couch-thread` | — | hold `no-couch-thread` |
| 12 | `Claims` one-inactive, `Branch` resting | `restore-workspace` | rule A + `ask-agent-restore`; dirty → rule A only | note `resting-branch-dirty` (why there is no restore request) |
| 13 | `Claims` one-active or many-with-active; or one claim with `Branch` detached | `agrees` | rule A | notes `inactive-claims`, `detached-head`, `git-unknown` as they apply |
| 14 | `Branch` open-issue, unclaimed, `Quality` ∈ {present, stale}, `GitSource` sdlc, no git dimension unknown | `claim-likely-lost` | rule A | note `claim-repair` |
| 15 | `Branch` open-issue unclaimed (otherwise), or claims present with `Branch` unknown | `partial-evidence` | rule A, but **resume only** | notes `claims-partial`/`claims-unknown`/`claims-absent`/`claims-unsupported`/`git-local-probe`/`git-unknown` |
| 16 | anything else | `no-rule` | — | hold `no-rule` (defensive; the totality test proves it is unreachable) |

**Rule A (actor step).** Read only from `ActorActions(ActorRowFactsOf(row))`:
- Live agent → no step.
- `resume` offered → `[resume]`.
- Otherwise `reboot` offered, every git dimension known, no operation and not detached → `[reboot]`, plus note
  `inspect-uncommitted-first` if dirty.
- Otherwise no step, and the class becomes `no-safe-step` with hold `reboot-unsafe-operation` / `reboot-unsafe-git` /
  `no-actor-action`.
- `Quality` stale adds note `claims-stale` ("claims stale: <reason>") on any class.

`Automatic` is true exactly when there are steps and no hold. `AllRecoverHolds()` and `AllRecoverNotes()` are closed
vocabularies; a parameterized code renders as `kind:suffix`. Every row's `Reason` comes from one function,
`recoverReason`. Rows are sorted by address.

- [x] **Step 1: Failing tests.**

```go
func TestDeriveRecoverPlanCoversEveryClassHoldAndNote(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range recoverPlanCases() { // one or more fixtures per table row and per rule-A branch
		t.Run(c.name, func(t *testing.T) {
			row := findRow(t, DeriveRecoverPlan(c.input), c.address)
			if row.Class != c.want || !slices.Equal(stepActions(row), c.steps) || !slices.Equal(row.Next.Hold, c.hold) || !slices.Equal(row.Next.Notes, c.notes) {
				t.Fatalf("got %s %v %v %v, want %s %v %v %v", row.Class, stepActions(row), row.Next.Hold, row.Next.Notes, c.want, c.steps, c.hold, c.notes)
			}
			seen["class:"+string(row.Class)] = true
			for _, h := range row.Next.Hold { seen["hold:"+codeKind(h)] = true }
			for _, n := range row.Next.Notes { seen["note:"+codeKind(n)] = true }
		})
	}
	for _, k := range derivedVocabulary() { // AllRecoverClasses ∪ AllRecoverHolds ∪ AllRecoverNotes minus no-rule — derived
		if !seen[k] { t.Errorf("no fixture yields %s", k) }
	}
}
func TestDeriveRecoverPlanIsTotalOverTheEvidenceDomain(t *testing.T) {
	// evidenceDomain() crosses every value of every SlotEvidence dimension (each from its own All* list, derived)
	// minus physically impossible combinations named in one consistent() predicate (~10^5 points, pure, < 2 s).
	// For every point: class != no-rule; a non-empty hold ⇒ no steps and Automatic false; every resume/reboot step
	// ∈ ActorActions; never reboot with Operation yes|unknown, Branch detached|unknown, any git dimension unknown,
	// or Agent busy|unusable-unknown; and Dirty|Unlanded|Operation unknown never yields idle.
}
func TestOneWeakSourceNeverHoldsARepo(t *testing.T) {
	// One partial card read in a fleet: the other slots of that repo keep agrees/claim-likely-lost with their steps;
	// only rows lacking a claim become partial-evidence.
}
func TestUnsupportedFleetUsesLocalProbe(t *testing.T) {
	// An unsupported fleet plus a LocalGit observation (dirty on 000012-x, parked thread) → partial-evidence [resume],
	// note git-local-probe, never reboot. A clean resting slot there (unlanded/operation unknown) → evidence-unavailable.
	// A healthy fleet with verdict unknown (probe:base) on an issue branch → partial-evidence with git-unknown.
}
func TestJoinUsesThePrimaryRowDefinition(t *testing.T) {
	// A :0 thread at the root plus a subdirectory thread in one scope: :0 has one thread, the subdirectory thread is
	// counted in ignored.non_slot_threads, and ApplyRepositoryAliases aliases exactly the joined row.
}
func TestClaimsAttachToTheirSlotPath(t *testing.T) {
	// Dependency-member claim (empty workspace) → in its slot's row, no conflict. Host claim with workspace "pair:9"
	// on pair:1 → conflict:claim-workspace. Dangling claim on worktree/pair-slot5/pair → pair:5 directory-missing.
	// Claim on a scratch worktree → ignored.off_slot_claims.
}
```

- [x] **Step 2:** Run → FAIL. **Step 3:** Implement in two pure steps: `slotEvidenceOf` (join plus facts), then a
  single `switch` that follows the table, with rule A as one helper. Step actions come only from `ActorActions` or
  the literal `ask-agent-restore`, and `RecoverStep.Command` stays empty in M1.
  `RestoreWorkspaceMessage(ref, address) = "Recovery (" + address + "): restore the workspace of this slot for " + ref +
  " through sdlc (check out its issue branch); never discard files. Reply with what sdlc issue show reports."`
- [x] **Step 4:** Run → PASS. Mutations, each on inputs two rules share, each must turn a test red:
  - swap 6 and 13 (a host claim on B, with B checked out and a mismatched workspace: `conflict` vs `agrees`);
  - swap 11 and 13 (an active claim with `Agent` none: hold `no-couch-thread` vs `agrees`; rule 11's claims clause
    and rule 13's one-active clause both match it, so the order decides);
  - drop the operation guard in rule A (operation present, resume not offered → `no-safe-step` vs `[reboot]`);
  - swap 3 and rule A (busy with reboot offered);
  - make `IsPrimaryRow` accept subdirectories.

  Commit `#367 M1: couchcore: DeriveRecoverPlan, union of evidence per slot`.

### Task 1.6: The `recover-plan` operation and `couch --recover-plan-from-sdlc`

**Files:** Modify `cmd/internal/couchcore/ops.go` (declaration, `PresentationRecoverPlan`, `ResultRecoverPlan`),
`operationdispatch.go` (`DirectStoreExecutor` case), `cmd/internal/couchcmd/cli.go` (`cliRecoverPlan`), `run.go`
(`RunWithRuntime` switch, `NewCouchWith` sets `c.Fleet = couchcore.SDLCFleetSource{IO: couchcore.OSProvisionIO{},
Timeout: 90 * time.Second}`, `render` JSON-encodes `couchcore.RecoverPlan`, and `usageWith` adds `couch
--recover-plan-from-sdlc`); tests in `cli_test.go`, `run_test.go`.

- [x] **Step 1: Failing tests.**
  - `TestRecoverPlanCLIEmitsTheReport`: `newRT` with a `FakeFleetSDLC` injected through a `recoverPlanRT`
    `NewCouchWith` override (the `provisionRT` pattern). Run `RunWithRuntime([]string{"--recover-plan-from-sdlc"})`.
    Expect exit 0, exactly one JSON document on stdout with `schema_version` 1, empty stderr, no supervisor acquired,
    and no runner ops.
  - `TestRecoverPlanCLIRejectsArguments`: `--recover-plan-from-sdlc x` and `--recover-plan-from-sdlc --layout2` exit 2.
  - `TestPublicHelpListsOnlyPublicSurface`: add `couch --recover-plan-from-sdlc` to the wanted list.
- [x] **Step 2:** Run → FAIL. **Step 3:** Implement. The declaration is `{Name: "recover-plan", Execution:
  ExecuteDirectStore, Effect: EffectRead, Confirmation: ConfirmNone, Result: ResultRecoverPlan, Presentation:
  PresentationRecoverPlan}`, and `operationOwnsLive` stays false. Run whatever audit enumerates
  `Operations()`/presentations and update it per its own message.
- [x] **Step 4:** `SCRUB go test ./cmd/internal/couchcmd ./cmd/internal/couchcore -count=1` → PASS. Commit
  `#367 M1: couch: --recover-plan-from-sdlc`.

### Task 1.7: Restart acceptance through the real report path

**Files:** Create `cmd/internal/couchcore/recoverplan_acceptance_test.go`

- [x] **Step 1: Write the test** `TestRecoverPlanAfterRestart`. Use `slotRecoveryOperationFixture` to get an env with
  slot stores, and give the env's `Couch.Fleet` a `FakeFleetSDLC` whose slot paths are the fixture's real
  `WorktreeRoot`s. Seed a "before restart" world:
  - `:1`: a record (`slotRecordFixture`) with a dead PID and its session present → detached. Claimed `#11`, branch
    `000011-x`.
  - `:2`: parked (`verified_park`), claimed `#12`.
  - `:3`: parked, dirty on its claimed branch `000013-x`.
  - `:4`: claimed `#14`, sitting on its resting branch; its agent is parked, so the steps are
    `[resume, ask-agent-restore]`.
  - `:5`: dirty on its resting branch, no claim.
  - `:6`: branch `000016-x` checked out, no claim, claims `present`, parked.
  - one `:0` thread live (`FakeProcOps` alive) with a ready verdict.
  - a dangling claim `#15` on `worktree/<repo>-slot9/<repo>` (the directory is gone).
  - a claim on a scratch worktree.

  Dispatch `DispatchOperation(OperationExecutors{DirectStore: DirectStoreExecutor(env.Couch)}, OperationCall{Name:
  "recover-plan"})`. Assert `:1`–`:3` → `agrees [resume]`, `:4` → `restore-workspace`, `:5` → `unidentified-work`, `:6` →
  `claim-likely-lost [resume]` with note `claim-repair`, `:0` → `idle` (live agent, clean resting branch, no claim:
  rule 9 and no step), `:9` → `directory-missing`, and
  `ignored.off_slot_claims == 1`. Then mutate the fake (`Release(":1")`) and dispatch again: `:1` →
  `claim-likely-lost`. This proves the plan is recomputed from fresh sdlc output on every call.
- [x] **Step 2:** Run → it must PASS against Tasks 1.1–1.6. Revert `DirectStoreExecutor`'s case → FAIL. Drop rule 10
  → FAIL.
- [x] **Step 3:** Commit `#367 M1: couchcore: restart acceptance through the recover-plan dispatch`.

### Task 1.8: Docs and close M1

- [x] README: under the Couch CLI section, add `--recover-plan-from-sdlc` (what it reads, the row classes, "unknown is
  never absence", "creates nothing"). `atlas/couch.md`: the recover-plan flow, the entities table above, and
  `FakeFleetSDLC`. Keep `atlas/index.md` links valid.
- [x] Full verification (Chunk 3). Paste the summary into `## Log`.
- [ ] `sdlc milestone-close --issue 367 --milestone M1`.

---

## Chunk 2: M2 — resume and reboot through the running Couch

### Task 2.1: Protocol types and the receipt state machine (pure)

**Files:** Create `cmd/internal/couchmessage/operation.go`, `operation_test.go`; modify `protocol.go`
(`Request.Confirmed bool \`json:",omitempty"\``, `Response.Operation *OperationReceipt`, and `ValidateRequest`
cases), `protocol_test.go`

Transitions (anything not listed → error, state unchanged):

| state \ event | admit(same op+target) | admit(other) | start | finish(ok) | finish(refused code) | finish(failed) | expire | status |
|---|---|---|---|---|---|---|---|---|
| (none) | → queued | — | err | err | err | err | — | unknown |
| queued | queued (no new effect) | `id-conflict` | → running | → succeeded | → refused (prepare refusals) | → failed | err (not terminal) | queued |
| running | running | `id-conflict` | err | → succeeded | → refused | → failed | err | running |
| terminal | same receipt | `id-conflict` | err | err | err | err | → removed (≥5 min after terminal) | receipt |

Couch exit drops every receipt (memory only), so `status` answers `unknown`.

- [ ] **Step 1: Failing tests.** `TestApplyReceiptEventTable` iterates every (state, event) pair from
  `allReceiptStatuses() × allReceiptEvents()` and checks it against the literal table. `TestReceiptSequences` covers
  admit → duplicate-admit → start → finish → status → expire → status=unknown. `TestValidateSlotOperationRequests`:
  `resume`/`reboot` need a valid ID, an exact `repo:N` target (`parseSlot`), no Body or Agent, and the caller
  identity. `Confirmed` is allowed only on `resume`/`reboot`. `operation-status` takes an ID only. A family target
  (`pair`) is refused with `ErrInvalidTarget`.
- [ ] **Step 2:** FAIL → **Step 3:** implement → **Step 4:** PASS. Commit
  `#367 M2: couchmessage: slot-operation requests and receipt state machine`.

### Task 2.2: `SelectSlotRow`, `ActorOperationArgs`, `SlotOperationCommand`, `PrepareSlotOperation`

**Files:** Create `cmd/internal/couchcore/slot_operation.go`, `slot_operation_test.go`. Modify
`cmd/internal/couchtty/menu_slot.go:30` (`dispatchMenuRow`) and `menu.go:1775` (`dispatchThreadOperation`) so both
build resume/reboot args with `ActorOperationArgs`. Delete the warm-only block from `dispatchMenuOperation`
(`menu.go:1797-1803`) and keep its `clearReattachFailure` call there.

- [ ] **Step 1: Failing tests.**
  - `SelectSlotRow`: a `:N` row matches by host path. `:0` matches only the row `IsPrimaryRow` accepts; a
    subdirectory thread is never picked. Zero matches →
    `SlotOperationError{Code: "no-thread"}`; two → `"ambiguous"`.
  - `ActorOperationArgs(row, op)`: a slot gives `{"path": WorktreeRoot}`; an ordinary row gives
    `{"repo-scope", "tag"}`; a **detached ordinary** row with `resume` adds `{"warm-only": "true"}`, so a detached
    `:0` is only ever reattached and never cold-started. Assert literal equality with what the switcher's effect
    carried before the change, captured for all three shapes, and that the switcher still emits them (couchtty
    `dispatchMenuRow`/`dispatchThreadOperation` tests). Mutation: drop warm-only from `ActorOperationArgs`; both the
    couchcore and couchtty tests must fail.
  - `SlotOperationCommand("resume", "pair:2") == "couch --resume pair:2"` and reboot gives `"... --confirm"`.
    `TestSlotOperationCommandParses` round-trips each through `couchcmd.ParseCLI`. That test lives in couchcmd
    because couchcmd imports couchcore.
  - `PrepareSlotOperation` on the `slotRecoveryOperationFixture` env:
    - a parked slot gives an `OperationCall{Name: "resume", Args: {"path": …}, Implicit: true}`;
    - a live slot gives `SlotOperationError{Code: "not-offered"}` with a factual detail ("pair:2 is live");
    - an unknown repo gives `"unknown-slot"`.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. `PrepareSlotOperation(ctx, op, target string) (OperationCall, error)`
  runs these steps:
  1. Parse the target with `ParseWorkspaceReference`.
  2. Resolve its path with `c.WorkspaceReferencePath`.
  3. Read `c.ActionableThreadInventoryContext(ctx, nil)`.
  4. Pick the row with `SelectSlotRow`.
  5. Return `not-offered` unless `slices.Contains(ActorActions(ActorRowFactsOf(row)), op)`.
  6. Build the call with `ActorOperationArgs`.

  Now set `RecoverStep.Command = SlotOperationCommand(...)` in `DeriveRecoverPlan`. For `ask-agent-restore`, use
  `couch --send-to <addr> --message '<RestoreWorkspaceMessage>'`, quoted with a shell-quoting helper. Add a test that
  every emitted command parses.
- [ ] **Step 4:** PASS. The couchtty tests stay green. Commit `#367 M2: couchcore: PrepareSlotOperation resolves and
  admits a slot target`.

### Task 2.3: The console's remote enqueue (the same queue as a keypress)

**Files:** Create `cmd/internal/couchtty/console_remote.go`, `console_remote_test.go`; modify `operation_queue.go`
(`operationRequest.finished func(any, error)`, carried into `operationCompletion`) and `console.go:1787`
(`finishOperation` calls `completed.finished(value, finalErr)` once, after adoption, from its existing defer).

```go
// EnqueueRemoteOperation runs a socket-originated operation through the same queue, dispatcher (c.ops) and
// completion path (finishOperation: adoption, menu result) as a switcher keypress. prepare runs ON the queue,
// so admission is judged against the inventory at execution time.
func (c *Console) EnqueueRemoteOperation(key string, prepare func(context.Context) (couchcore.OperationCall, error),
	started func(), finished func(any, error)) error
```

The job does three things, in order: it calls `started()`; it runs `prepare(c.lifetime)`, which returns its error on
failure; and it runs `c.ops(call)` with `Implicit: true, Context: c.lifetime`. Its origin is `{Operation: name,
Address: from the call args, PreserveFocus: true}`, with `Attempt == 0`. No new origin field is needed; Attempt 0 is
what keeps it out of `InFlight`. For a resume, the console also reduces `clearReattachFailure` for the resolved
address on completion, as the switcher's resume does. With no origin field, the completion recognises a remote
resume by exactly: `Operation == "resume"`, `Attempt == 0` and no `ContinuationID`, so a continuation replacement
never matches; a test pins both sides. If `Enqueue` returns `accepted=false`, the method returns
`errRemotePending`; overflow returns `errOperationQueueOverloaded`. **Single outcome owner:** when the method returns
an error, it has not called and never will call `started` or `finished`; the admission path reports that outcome.
When it returns nil, `finished` fires exactly once.

- [ ] **Step 1: Failing tests** on the `continuationConsole`-style fixture:
  - A remote resume whose dispatcher returns a `StartResult` is adopted. `attach` is called with
    `background="true"`, focus is unchanged, and `finished` gets `(value, nil)`.
  - An `attach` failure reaches `finished` as an error.
  - A remote completion while the operator has `InFlight.Attempt=3` leaves `InFlight` intact.
  - A duplicate key gives `errRemotePending`.
  - A prepare error reaches `finished` and no dispatcher call happens.
  - An overloaded or pending enqueue returns an error and **never** calls `started` or `finished` (no double report).
  - A remote resume clears the row's reattach-failure mark.
- [ ] **Step 2:** FAIL → implement → PASS. Mutation: call `finished` before adoption, and confirm the attach-failure
  test fails. Commit `#367 M2: couchtty: remote operations ride the console queue`.

### Task 2.4: The message-service handler and caller rule

**Files:** Create `cmd/internal/couchcmd/slot_operations.go`, `slot_operations_test.go`; modify
`message_service.go` (`newMessageService` takes `slotOps slotOperationRunner`, where
`type slotOperationRunner func(key string, op, target string, started func(), finished func(any, error)) error`;
`startMessageService` wires `console.EnqueueRemoteOperation` with `c.PrepareSlotOperation`; and `handle` intercepts
`resume`/`reboot`/`operation-status` before `couchmessage.Handle`). Add the new file to `NonArtifactSources`.

**Concurrency owner:** `slotOperations` holds one `sync.Mutex` over its receipt map and clock-driven sweep. `admit`,
`status`, `started` (queue goroutine) and `finished` (console loop) all lock it. None of them waits on the queue or the
console while holding it.

Handler order:
1. `ValidateRequest`.
2. `broker.Caller` + `authority.current`. Any failure → `unavailable`, with nothing enqueued.
3. For `reboot` without `Confirmed`, check `couchcore.OperationConfirms(op)` → `confirmation-required`.
4. Canonicalize the target to a queue key from the **resolved** slot: `ResolveRepositoryName` over the stored
   repository names, then `"remote\x00" + primaryKey + ":" + N`. This is a store read, with no git or sdlc call.
   `pair:1` and `pa:1` therefore share one pending key.
5. Under the mutex: `ApplyReceiptEvent(admit)`. A duplicate returns the existing receipt and STOPS here: it never
   calls `slotOps`, so a duplicate can never enqueue a second job (tested). Otherwise call `slotOps(key, …)`.
   `Enqueue` never blocks, so holding the lock here is safe, and a job that starts at once waits briefly in `started`
   until its receipt exists. On an enqueue error, drop the reservation and answer pending → `busy`, or overflow / more
   than 64 receipts → `overloaded`. That is the only report of that outcome.
6. Return `accepted` + the receipt.

`finished` maps errors as follows: `errors.As(*SlotOperationError)` → `refused` with its code; anything else →
`failed`, with `Diagnostic: couchcore.ResumeDiagnosticOf(err)`. On success it records the new tag, the
`RebootResult.Archived` tag and the `Warning()`. `operation-status` shows only receipts admitted by the same
(Scope, Tag).

- [ ] **Step 1: Failing tests** on `serviceRig` with a fake `slotOps` that records calls:
  - `TestSlotOperationCallerRule` covers: no identity, an unknown nonce, a disconnected binding, a binding that is no
    longer current (`TestMessageSendingCallerMustRemainCurrent`'s mechanism), and an ambiguous binding. Each gets
    `unavailable` with zero `slotOps` calls. A connected current caller gets `accepted`.
  - `TestSlotOperationRebootNeedsConfirmation`.
  - `TestSlotOperationTypedOutcomes`: `not-offered`, `no-thread`, `failed` + diagnostic, `succeeded` + tag, `busy`,
    `overloaded`, `id-conflict`, duplicate-admit.
  - `TestOperationStatusIsCallerScoped`.
  - `TestReceiptsExpireAndCap`: inject a clock and check the 5 min expiry and the 64 cap.
  - `TestSlotOperationReceiptsConcurrent`: run `-race` with N goroutines admitting, polling status, and firing
    `started`/`finished` from separate goroutines. No race, and every receipt ends terminal exactly once.
  - `TestSlotOperationAliasSharesKey`: `pair:1` then `pa:1` while the first is pending → `busy`.
- [ ] **Step 2:** FAIL → implement → PASS (`SCRUB go test -race ./cmd/internal/couchcmd -run SlotOperation -count=1`).
  Mutation: skip `authority.current`, and confirm the stale-caller case
  fails. Commit `#367 M2: couch: resume/reboot on the message socket for live slots only`.

### Task 2.5: CLI `--resume` / `--reboot`

**Files:** Modify `cmd/internal/couchcmd/cli.go` (the `parseMessageCLI` forms; add `"--resume", "--reboot"` to
`ParseCLI`'s early switch), `messages.go` (an admit-then-poll branch with injectable `sleep`, 500 ms interval and
3 min budget). Today one `AdmissionTimeout` context covers the whole invocation (`messages.go:52`). The new branch
creates a **fresh `context.WithTimeout(…, AdmissionTimeout)` per call** (admit and each status poll), with the 3 min
budget as an outer deadline, `run.go` `usageWith` (two lines); tests in `cli_test.go`, `messages_test.go`, `run_test.go`
(`TestPublicHelpListsOnlyPublicSurface`: allow the `--resume`/`--reboot` flags and keep refusing the bare internal
names: `start`, `park`, `publish-description`, `--internal`, and `resume` outside `--resume`).

- [ ] **Step 1: Failing tests.**
  - Parse forms: `--resume pair:1`, `--resume pair:1 --json`, `--reboot pair:1 --confirm [--json]`. Refused:
    `--resume`, `--resume pair`, `--resume pair:1 extra`, `--reboot pair:1 --confirm --confirm`.
  - Poll loop with a fake `messageCall`:
    - `accepted` → `running` → `succeeded` gives exit 0 and one line: `pair:1 resume succeeded (tag …)`.
    - `refused not-offered` gives exit 1 with the detail.
    - Polling past the budget gives exit 1: `outcome uncertain; verify with couch --recover-plan-from-sdlc`.
    - A status of `unknown` after Couch restarts gives the same uncertain line.
    - Mid-poll, a dial error (Couch exited, socket gone), an `unavailable` response (the caller's binding no longer
      resolves after a Couch restart), or a per-call timeout gives the same **uncertain** line, exit 1, with no raw
      transport error as the only output.
    - An admit-time transport error gives uncertain plus the printed ID (`printUncertainMessage` pattern).
    - The 6th poll still succeeds after more than 2 s in total, which proves the per-call context.
    - `--json` prints the final `OperationReceipt`.
  - Outside a slot (no `COUCH_STORE_DIR`): "requires a live Couch slot", exit 1.
- [ ] **Step 2:** FAIL → implement → PASS. Commit `#367 M2: couch: --resume and --reboot CLI`.

### Task 2.6: Skill recovery section

**Files:** Modify `cmd/internal/couchcmd/skills/couch/SKILL.md` (frontmatter `description` += "or recovering slots
after a restart"); test `messages_test.go`.

Section "## Recovering slots after a restart". Take the contract wording from
`/Users/xianxu/workspace/worktree/pair-slot1/ariadne/bin/sdlc help recovery`. The section says:

1. Run `couch --recover-plan-from-sdlc` and read every row.
2. Review the rows with the operator before acting, and never run steps without that review.
3. For each row the operator approves that has `automatic: true`, run its `steps[].command` in order.
4. Delegate disk fixes and claim repairs to the slot's own agent through `--send-to` (the `ask-agent-restore` command). Never edit
   another slot's repository.
5. A row with a `hold` gets no primitive unless the operator directs one for that specific row. A `conflict` row
   lists its facts so the TL can inspect them. For a `claim-repair` note, first run
   `sdlc issue show N --json`. If a repair is needed, ask the slot's own agent to run `sdlc claim --adopt` /
   `sdlc reclaim`, only on the operator's explicit instruction (#278), never on the report's suggestion. `start-unreconciled`
   and `agent-unknown` mean: re-read later, and never reboot.
6. Verify by re-running the report, never by a reply or a receipt. A message receipt proves delivery only. Stale or
   unknown is not negative evidence: look again after about 30 s.
7. An uncertain outcome means re-read before resending. A resend is refused harmlessly once the slot is live.
8. Continuing work and scheduling are not part of recovery (step 4).

Also add the two commands to the command table.

- [ ] **Step 1: Failing test** `TestSkillDocumentsRecovery`: `couchSkill` contains `--recover-plan-from-sdlc`,
  `--resume`, `--reboot`, `--confirm`, `--send-to`, and "re-run" verification wording. Every `couch --…` command in
  the skill parses with `ParseCLI`. This is a derived sweep: extract each fenced or backticked `couch …` command,
  then **shell-split it** with a quote-aware splitter (a test helper handling `'…'`, `"…"` and `\'`), so quoted
  `--message` bodies arrive as one argv element. Every class named in
  the skill is in `AllRecoverClasses()`.
- [ ] **Step 2:** FAIL → write → PASS. Commit `#367 M2: couch skill: recovery procedure`.

### Task 2.7: End-to-end recovery acceptance

**Files:** Extend `cmd/internal/couchcore/recoverplan_acceptance_test.go`; create
`cmd/internal/couchcmd/slot_operations_acceptance_test.go`

- [ ] **Step 1: couchcore loop** `TestRecoverPlanStepsConverge`. Start from Task 1.7's world. For each `automatic`
  row: `PrepareSlotOperation(step)` → `DispatchOperation` with `CouchLiveOwnerExecutor(env.Couch)` → mark the started
  process alive in `FakeProcOps` → re-run `recover-plan`. Expect `:1`/`:2`/`:3`/`:6` live with no steps (`:6` keeps note
  `claim-repair`), `:5` untouched (its record is byte-identical), and `:4` → `restore-workspace` with steps reduced
  to `[ask-agent-restore]`.
  Calling `PrepareSlotOperation` again on `:1` → `not-offered`, which proves resends converge.
- [ ] **Step 2: couchcmd socket acceptance.** Use a real `newMessageService` on temp sockets (the `serviceRig`)
  wired to a real `couchtty.Console` fixture, whose dispatcher is `DispatchOperation` over a test Couch. The fixture
  needs no pty: use the `continuationConsole` shape and run its queue goroutine. Run
  `runMessageCLIWithCall(--resume pair:1)` from a connected caller's env and expect exit 0. The same call from a
  non-slot env refuses before anything is enqueued.
- [ ] **Step 3:** Both PASS. Revert the `handle` intercept → socket test FAIL. Revert `finished` → poll times out →
  FAIL. Commit `#367 M2: acceptance: report → resume via socket → report`.

### Task 2.8: Docs and close M2

- [ ] README: `--resume`/`--reboot` (live-slot callers only, `--confirm`, receipts, verify via the report).
  `atlas/couch.md`: the socket slot-operation flow and receipt lifecycle (ARCH-FUNERAL line), and `ActorActions` as
  the shared authority. `atlas/index.md` links.
- [ ] Full verification (Chunk 3).
- [ ] **Ask the operator to smoke-test live** (memory: dogfood live). Run `sdlc move` then `make build` in `pair:0`,
  and check its HEAD. Relaunch Couch. From `:0`'s agent: `couch --recover-plan-from-sdlc`, then park a scratch slot,
  `couch --resume pair:N`, re-read the report, and confirm it is `live`. The new pane must appear without stealing
  focus.
- [ ] `sdlc close --issue 367 --verified '<evidence>'`.

---

## Chunk 3: Verification recipe (every milestone close)

Set `SCRATCH=/private/tmp/claude-501/-Users-xianxu-workspace-worktree-pair-slot1-pair/3d48cc67-1d65-4766-9e3b-e76693ebf9ef/scratchpad`
(use a fresh scratchpad path if the session changed). Redirect output to files, and never pipe to `head` (SIGPIPE
gives a phantom FAIL).

1. `SCRUB make -k test > $SCRATCH/make-test.log 2>&1`. Expected: only `test-changelog` fails.
2. `SCRUB TMPDIR=$SCRATCH/tmp make test-changelog > $SCRATCH/changelog.log 2>&1`. Expected: PASS.
3. `SCRUB go test ./... -count=1 > $SCRATCH/go-test.log 2>&1`. Expected: only
   `TestProductionArtifactReferencesAreExactlyClassified` and `TestCouchReferencesLocalArchiveLocatorRoundTrip` fail
   (both fail on main too). Run `SCRUB go test ./cmd/internal/artifactpath -run
   TestProductionArtifactReferencesAreExactlyClassified -count=1 -v`. Its 33 failing subtests must be exactly main's
   33: diff the `--- FAIL` subtest names against the same command on main, and confirm none names a new #367 file.
4. PTY-dependent packages (`ptychild`, console attach) report "operation not permitted" in the sandbox. Re-run them
   with the sandbox off before calling them failures. The live sdlc conformance test needs network access for
   tracker fetches; run it unsandboxed or accept its skip, and record which happened.
5. Paste the pass/fail summary into `--verified` and `## Log`.

## Revisions

### 2026-10-03 — plan review (coordinator), blocking findings 1–13

Reason: the fresh-eyes plan review. Delta:
- **sdlc input.** Presence-aware decoding with a pre-#288 golden, and typed rejection asserts (1, 2).
- **Join.** `fleet_root` grouping plus an address dedupe (3); `IsPrimaryRow` as the one `:0` definition (4); empty
  dependency-clone claim workspace, verified in ariadne `claimant.go` (5); overlapping mutation pairs (6); per-hold
  coverage (7); `not-in-fleet` vs `outside-fleet` scoped by row kind (8).
- **Socket.** Warm-only moves into `ActorOperationArgs` (9); a per-call poll context, with failures → uncertain (10);
  a mutex owner and `-race` (11); a single owner for the overload outcome (12); no `Remote` origin field (13).
- **Advisories.** All applied.

### 2026-10-04 — operator design decisions

Reason: the operator design session. Delta:
- Ambiguities 1, 2, 9, 10 and 11 are confirmed.
- **Union of evidence** replaces `unattributed`. New classes: `claim-likely-lost` (resume + `claim-repair` note),
  `partial-evidence` (resume only) and `conflict` (facts listed, no step).
- **`claims_state` is per-source quality**, not a hold.
- **Unsupported fleets** reuse `ProbeSlotGit` for disk evidence.
- **Dirty files no longer block resume.** An operation or detached HEAD → resume only, never reboot.
- **Rows are per slot path only.** `off-slot-claim`/`outside-fleet` become `ignored` counts, and a dangling claim on
  a conventional slot path gives `directory-missing`.
- **Multiple claims** pick the active one by branch; more than one on the resting branch → `ambiguous-claims`.
- **Busy** becomes `start-unreconciled`.
- Task 1.5's table, vocabularies and mutations, and the Task 1.7/2.6/2.7 expectations, are re-derived.

### 2026-10-04 (b) — operator answers and table-totality review

Reason: the operator answered the plan's three questions, and the review found that the table was not total. Delta:
- **Operator answers.**
  - A conversation alone → `idle`, no step; `conversation-only` is removed.
  - One claim on a dirty resting branch → resume only, with the `resting-branch-dirty` note explaining why.
  - Stale claims show `claims stale: <reason>`.
- **Totality.** The table is rebuilt over closed evidence dimensions:
  - any non-issue branch with no claim → `unidentified-work`;
  - claimed + detached → `agrees` (resume only);
  - a claim on another branch → `conflict:claim-off-branch`;
  - a `no-rule` catch-all.

  A totality test runs over the derived evidence domain, and unknown git is never idle or absence (rule 8, rule 15
  `git-unknown`).
- **Mutations.** Re-picked so each pair overlaps: 6/13 and 11/13 (13/15 was disjoint by construction; replaced after review round 3).
- **Probes.** `ProbeSlotGit` is bounded by the shared `SlotGitProbeTimeout`. `SlotCandidates` are gathered in the
  shell, so the derivation stays pure.
- **Claim repair.** `claim-repair` → `issue show` first; the slot's own agent repairs, under operator direction.
- **Wording.** The dangling "design problems" references and the ARCH-DRY wording are fixed.

### 2026-10-04 (c) — M1 implementation reconciliation

Reason: M1 delivered; the active mappings above are reconciled with the code (lessons: reconcile entity tables, not
only append). Delta:
- **Order.** Task 1.5 landed before Task 1.4: `Couch.RecoverPlan` returns `DeriveRecoverPlan(input)`, so the pure
  join had to exist first. Each task kept its own red → green commit.
- **No `PrimaryScopes` input.** `launcher.ResolveRepoScope` is a pure hash, so `DeriveRecoverPlan` computes the `:0`
  scope itself; one less shell → pure channel.
- **`SlotEvidence.Offer`** (`none` / `reboot` / `resume-reboot`, from `ActorActions(ActorRowFactsOf(row))` for the
  single joined row) is a dimension, so rule A is a function of the evidence and the totality test checks every step
  against the offer. `consistent()` ties it to `Agent`.
- **New hold `resume-only`.** `partial-evidence` and a dirty `restore-workspace` are resume-only (ambiguity 6, operator
  answer in Revision (b)); when resume is not offered, the row is `no-safe-step` with this hold instead of borrowing
  `no-actor-action`.
- **`restore-workspace` with dirt unknown** is resume-only with note `git-unknown` and no restore request (an unread
  tree is not known clean).
- **A slot-target Couch row the healthy fleet lacks** (path present) has unknown git and no claims, so the total table
  gives `evidence-unavailable` (rule 8), superseding ambiguity 7's older `partial-evidence` wording.
- **Dependency members** fold into the slot: their dirt, unlanded commits and operation count as the slot's; their
  claims attach as inactive claims; claim quality is the host's unless a dependency's read is weaker.
- **`ActorRowFacts`** is `{Slot, State, Reason, ResumeOffered, DirectoryMissing}`.
- **Acceptance fixture.** `recoverAcceptanceFixture` is `slotRecoveryOperationFixture` widened to six real slot
  worktrees. It showed an enrolled slot with no record is still a Couch row (unusable, `never-started`), which the
  join counts as a thread but not as a conversation.
- **`landed`** (coordinator, after review of the above): a clean (no dirt, unlanded commits or operation), unclaimed
  slot on a terminal issue's branch is class `landed`, note `issue-done-branch`, no step, ordered after
  `ambiguous-claims`; `conflict:issue-terminal` applies only with dirt, unlanded commits, an operation or a claim.
- **M1 review (BR-4/BR-5), amending (c)'s "dependency claims attach as inactive claims":** each claim is judged
  against its own member's branch. Dependency claims form the `DepClaims` dimension and never drive a host restore
  or `claim-branch-mismatch`; `RestoreWorkspaceMessage` takes the holding checkout. Dangling claims on a present
  slot's members count toward that slot (unread member → `claim-member-unread`). New conflict fact
  `dependency-claim`; candidate layout errors go on the fleet observation's error; only `fs.ErrNotExist` is missing.
