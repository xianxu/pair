# Confirmed slot recovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks or superpowers-executing-plans for work needing the coordinator's current context. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make disk resume and fresh startup executable after interrupted or damaged slots, with exact-owner confirmation and evidence preserved before retirement.

**Architecture:** Extend the existing Couch recovery coordinator, operation queue, thread journal and Pair launch gates. Keep proven resume, explicitly selected inference and Fresh distinct; share preparation, confirmation and retirement. Preserve #528's Continue/auto-attach behavior and #522's ordinary Prompt delivery.

**Tech Stack:** Go; local JSON thread records/journals; existing native transcript inventory, procutil, Pair lifecycle and Zellij adapters; stateful portable test doubles.

---

## Contract and choices

Sources: [issue](../issues/000556-confirmed-slot-recovery.md), [slot recovery target](../targets/slot-recovery.md), and the teardown/confirmation commitments in [slot continuity](../targets/slot-continuity.md).

1. Positive live listing remains Continue, including busy agents and broken message channels. No recovery execution at startup. Existing named-hub startup and Continue auto-attach stay unchanged.
2. Resume uses an existing proven binding or an explicitly chosen ranked identity. Keep automatic inference off, best plus four alternatives, with activity/score/excerpt evidence. A copied candidate key is evidence selection, never permission to stop anyone.
3. Fresh has its own selection independent of native scanning and capture availability. It always remains in the recovery menu and has an executable CLI command. Missing/expired/unreadable screen is disclosed; branch/issue still orient the new agent. Unsupported *scans* permit Fresh; a future-version authoritative thread record is not permission to overwrite it with an older binary.
4. Before stopping an attributed owner, preview names the exact server/wrapper/agent/helper generations and descendants that will stop. Acceptance is by the person or the currently authenticated owning hub, expires after two minutes, is one-use and lives only in Couch memory. A changed owner or choice requires a new preview. Existing own-slot and hub-routing restrictions remain.
5. No tag/name-only kill authority. Unknown or foreign remnants are left untouched and reported; Fresh allocates a new tag/terminal identity if the old namespace is contested. Native resume is never permitted while any live writer of that transcript remains unresolved; offer Fresh instead.
6. Preserve the selected intent and available evidence before any quit, signal, cleanup or record replacement. Use existing journals and capture retention, never another recovery file family. Native files, branch, issue, dirty work and drafts are not reset or deleted.
7. Launch once through the existing launch owner. Registered, definitively failed and unconfirmed outcomes remain distinct. Missing registration is not proof the agent did not start; no destructive rollback or silent fresh replacement while that launch may be alive.

Considered alternatives: boolean-only confirmation cannot bind the stop set; a new recovery queue duplicates receipts/serialization; making capture-backed native evidence optional weakens an existing authority. Instead, share one prepared execution path and add a separate, typed optional orientation context for Fresh.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| RecoverySelection / prepared recovery decision | `cmd/internal/couchcore/recovery_plan.go` | new |
| RecoveryPreview / RecoveryChoice | `cmd/internal/couchcore/recovery_preview.go` | modified |
| Recovery execution state/event/effect transition | `cmd/internal/couchcore/recovery_plan.go` | new |
| RecoveryIntent / Fresh orientation context | `cmd/internal/threadrecord/recovery.go` | modified |
| StartTransaction / stale-start decision | `cmd/internal/couchcore/starttransaction.go`, `actionableinventory.go` | modified |
| RecoveryTerminalPlan attribution | `cmd/internal/launcher/recovery_terminal.go` | modified |

One prepared decision owns one selection, a record/start generation, an exact attributable stop set, and retained-foreign diagnostics. Selection kinds are proven, inferred and fresh; they cannot be inferred from an empty candidate key. No caller constructs signal authority from display text. The pure transition decides permitted effects; the IO coordinator feeds observed results back through it (ARCH-PURE, ARCH-ORDER).

Keep the legacy capture-backed RecoveryContext validator intact. Extend the existing durable RecoveryIntent as a tagged variant: inferred carries that strict context; proven carries the exact existing binding witness; fresh carries branch/issue and optional validated screen coordinates plus an explicit unavailable reason. No missing capture manufactures native authority. Persist this in the existing recovery/start slots, not a second journal. Version the containing record as schema 4, migrate readable schema 3 in memory, and keep unsupported future versions explicitly refused. Ordinary pre-upgrade records must remain readable without a rewrite at startup.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Recovery coordinator | `cmd/internal/couchcore/recovery_resume.go`, new `recovery_plan.go` | modified | existing selection, holds, preservation, journal and launch |
| Ephemeral confirmation owner | new `cmd/internal/couchcore/recovery_confirmation.go` | new | clock/entropy and a bounded in-memory capability map |
| Recovery evidence/preservation adapter | `cmd/internal/couchcore/recovery_evidence_os.go` | modified | existing capture retention and transcript inventory |
| Confirmed terminal retirement | `cmd/internal/launcher/recovery_terminal.go`, `recovery_terminal_os.go` | modified | procutil, terminal leases, exact process/tree retirement |
| Native ownership gate | `cmd/internal/couchcore/recovery_native_owner.go`, `cmd/internal/launcher/recovery_native_owner.go` | modified | existing wrapper/readiness/ledger observation |
| Recovery start completion | `cmd/internal/couchcore/launch_existing.go`, `starttransaction.go` | modified | existing runner, registration and durable start claim |
| Menu and authenticated CLI | `cmd/internal/couchtty/menu_recovery.go`, `cmd/internal/couchcmd/slot_operations.go`, `resume_candidates.go` | modified | existing operation declarations, hub guards, queue and receipts |

Reuse FakeRunner, FakeProcOps, fake artifact/native-wrapper state, the existing terminal driver double, portable ThreadStore directories and measured transcript fixtures. Extend those doubles with exact generations, stop/launch ordering, partial deaths and delayed registration. No separate fake-only execution path (ARCH-MOCK). Pure decisions have ordinary colocated table tests; IO sequence tests use those stateful seams.

## Operation and state model

Read-only `--resume-candidates repo:N [--json]` returns proven/ranked/Fresh choices, context-loss reasons and any required exact stop confirmation. Existing `--resume repo:N` selects proven resume; `--resume repo:N --candidate KEY` keeps explicit inference. Existing `--reboot repo:N --confirm` is Fresh, converging onto this recovery preparation/execution path, rather than keyless menu fallback to a different contract. Add `--recovery-confirmation TOKEN` to these operations when preview names a stop set. A first execution without required acceptance returns the exact confirmation offer with no lifecycle effects. Menu confirmation uses the same preparation/token; no second UI lifecycle implementation.

Tokens bind the current authenticated confirmer, target, record/start nonce, selected identity/evidence and exact stop-set digest. They are not serialized into intent, launch argv or journals. Limit the map to 256 offers; collect expired/consumed entries, evict oldest unaccepted offers if necessary, and regenerate a preview after eviction. Closing Couch revokes them all. An empty stop set needs explicit selection but no fabricated process confirmation.

| State + event | Next state / effects |
|---|---|
| Observe + positive live listing | Continue; ordinary attach; no destructive recovery |
| Observe + failed/changed/over-budget read | One bounded retry; then preserve uncertainty and expose recoverable choice/Fresh, never Continue without listing |
| Prepare + valid selected evidence | Prepared; preflight profile/workspace identity, exact source attribution and selected native uniqueness |
| Prepared + running attributable stop set | Await acceptance; expose exact identities and expiring token, no lifecycle writes |
| Prepared + accepted unchanged identities | Preserve evidence and CAS durable selection; consume ephemeral acceptance |
| Preserve failure, cancelled before effects or drift | Stop with no retirement; selection/Fresh remains reachable through refreshed evidence |
| Preserved + immediate effect recheck | Retire only accepted attributable owners; park normally when possible, otherwise exact confirmed retirement |
| Partial retirement / cancelled / changed descendant | Retain selection and partial progress; stop further signals; refresh/reconfirm remaining owners, never discover a new kill target |
| Retired + proven absence and valid selection | Non-force exact dead-owned cleanup, fence stale claim, commit normal start claim, launch once |
| Launch + registration proven | Registered; truthful ready/not-ready notice and native identity |
| Launch + definite no-effect failure | Retain intent/evidence; existing safe claim rollback; explicit retry/Fresh |
| Launch + missing/uncertain acknowledgment | Reobserve exact attempt once; retain its claim/identity while unconfirmed; Continue if listed live, otherwise explicit reconciliation or Fresh under retirement confirmation |
| Retry / duplicate receipt / late completion | Reuse operation receipt; no second launch; old nonce cannot advance replacement claim or gain native execution authority |

All store transitions remain CAS-owned by ThreadStore and AdvanceStartTransaction; add only the specific selection/fencing events needed here. A stale start's supervisor PID is never a kill target. At an explicit recovery effect, replace/fence its exact nonce through CAS so an old driver cannot publish or execute native authority for it. Pair's existing second gate validates the new durable selection immediately before native launch.

### Reconciliation and exact write order

| Observation | Offered outcome | Ordered writes after explicit action |
|---|---|---|
| Listed live, including unavailable message channel | Continue | Existing attach/retry only; no recovery teardown |
| Dead owner, proven binding, screen absent | Resume from disk | Persist binding selection; exact dead cleanup; ordinary proven native resume, no Prompt |
| Dead owner, selected ranked transcript | Resume from disk | Preserve strict capture/candidate snapshot; persist selection; cleanup; claim; revalidate candidate and uniqueness; exact native resume, no Prompt |
| Attributed damaged/orphan owner, selected transcript | Resume from disk after named stop confirmation | Preserve; persist; recheck accepted generations; park/exact retirement; prove old writers gone; cleanup; claim and native resume |
| Missing/weak/incomplete scan or missing screen | Start fresh | Persist available orientation and explicit loss reason; confirm any named stop set; retire only those owners; archive old record; allocate new tag if needed; claim; launch with ordinary Prompt |
| Foreign/unknown remnant | Start fresh | Preserve/archive old evidence; leave remnant and sockets/layout alone; allocate fresh tag/session; launch with warning; no native resume if writer ownership is unresolved |
| StartClaim owner dead or claim older than five minutes, no positive listing | Resume/Fresh | Explicit selection/confirmation; preserve; CAS-fence exact claim nonce; retire only attributed children (never Couch owner); follow selected launch path |

Five minutes is a proposed presentation bound, not proof of death and not an automatic cleanup timeout. Younger genuinely active claims can show Starting. Expired claims offer recovery even if the supervisor remains alive; confirmation and fencing guard effects.

For settled live wrappers use ParkExpected and its receipts. The exact terminal driver gains an explicitly confirmed mode requiring wrapper/readiness/native association; do not simply remove its current live-wrapper rejection. Snapshot authorized descendants before effects; disappearing members are success, new/replaced members stop further signals. Delete a stale saved layout only after its exact owned server is dead; never force-delete a session or remove a socket shared with a foreign generation.

Fresh's new tag avoids contested terminal artifacts; do not weaken all OpenSlot/fresh guards globally. Narrow the recovery-specific absence check to accepted source writers and exact address/native collisions, retaining foreign diagnostics. A foreign survivor cannot authorize killing or reusing its namespace.

## Operating envelope, trust and retention

- Interactive explicit operation; no new startup scan or background recovery. Reuse the 30-second preview budget and existing inventory scan byte/file bounds, at most two observations. Admission/effect identity checks remain mandatory even when an observation retry is exhausted (ARCH-CONSTRAINTS).
- One existing per-thread hold and queue; no parallel recovery effects for the same source. Existing terminal leases serialize launch/retirement. Different selected native identities still pass the global owner check immediately before execution.
- Record/capture/transcript/CLI input is untrusted: strict versioned decode; physical cwd, bounded native paths and immutable file identities revalidated. Truncated/unsupported scan becomes incomplete plus Fresh; altered capture never becomes inference authority. A future record version is not clobbered (ARCH-SECURE).
- Cancellation stops unsent effects, not already started native processes. Owned tasks are bounded by the request/registration contexts; retained unresolved attempts are durable facts, not detached cleanup goroutines.
- New confirmation offers expire in two minutes and die with Couch. New durable selection stays in existing recovery/start fields and is superseded/archived through their normal lifecycle. Capture copies use existing retention leases and expiry; record backups retain existing count/byte limits. No new unbounded family or native transcript copies (ARCH-FUNERAL).
- No network/LLM inference, launch-parameter edits, endpoint recreation (#557), message reconciliation (#561), automatic candidate selection or Zellij fork (ARCH-PURPOSE).

## Implementation tasks (one whole-issue review boundary)

### 1. Typed selection and optional fresh context

Files: new `cmd/internal/couchcore/recovery_plan.go`, `recovery_plan_test.go`; modify `cmd/internal/threadrecord/recovery.go`, `record.go`, `recovery_context_test.go`, `record_test.go`; adapt `couchcore/slotrecovery.go` version admission and schema tests.

- [ ] Add failing pure tests: three selection kinds; strict inferred capture unchanged; proven without tty; fresh without tty; unsupported future record; schema-3 read without writes.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'TestRecoverySelection|TestRecoveryContext|TestSlotRecoverySchema|TestThreadRecord' -count=1`; confirm the new cases fail.
- [ ] Implement typed variants, version migration and pure transition; re-run to PASS. Commit with issue reference and agent trailers.

### 2. Shared preparation, exact confirmation and stale Starting

Files: new `couchcore/recovery_confirmation.go`, `recovery_confirmation_test.go`; modify `recovery_preview.go`, `recovery_resume.go`, `recovery_native_owner.go`, `actionableinventory.go`, `starttransaction.go`, `lifecycledebris.go`; extend their colocated tests.

- [ ] Add failing sequence tests for expired/replayed confirmation, changed PID/start/nonce, changed candidate, wrong hub/target, cancelled acceptance and stale Starting with live supervisor.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestRecoveryConfirmation|TestRecoveryPlan|TestStartClaim|TestStarting' -count=1` to establish RED.
- [ ] Implement preparation/capability ownership and pure expiry projection; keep Continue first and startup read-only. CAS-fence stale claims only on explicit recovery. Re-run to PASS and commit.

### 3. Preservation, retirement and one native launch

Files: `couchcore/recovery_resume.go`, `recovery_evidence_os.go`, `reboot.go`, `slotrecovery.go`, `threadstore.go`, `launch_existing.go`, `couch.go`; `launcher/recovery_terminal.go`, `recovery_terminal_os.go`, `recovery_authority.go`, `recovery_authority_os.go`; corresponding recovery/native-owner/start tests.

- [ ] Add stateful failing cases: preserve failure before signal; owned damaged wrapper; foreign remnant left byte/process-identical; new descendant/PID reuse mid-retirement; partial retirement; proven resume without screen; fresh incomplete/unsupported scan/missing screen; delayed registration after a real launch.
- [ ] Run `go test ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/orientation -run 'TestRecovery|TestResumeCandidate|TestReboot|TestStartTransaction' -count=1` and confirm the new cases fail.
- [ ] Implement shared ordering using existing journals, park/exact retirement and launch APIs. Every native resume has empty Prompt; every Fresh uses BuildRecoveryOrientation with available coordinates only. Reobserve uncertain attempts once, never kill/fresh-replace merely because registration timed out. Re-run to PASS and commit.

### 4. Menu, CLI and authenticated hub parity

Files: `couchcore/ops.go`, `operationdispatch.go`, `slot_operation.go`; `couchcmd/cli.go`, `resume_candidates.go`, `slot_operations.go`, `console_slot_operations.go`; `couchmessage/protocol.go`; `couchtty/menu_recovery.go` and existing confirmation/menu state files; colocated declaration/protocol/hub/menu tests.

- [ ] Add failing tests: Fresh command never has empty candidate key; all scan failures and missing tty retain executable Fresh; default/four alternatives stable; owned stop summary shared by menu/CLI; hub ownership changes and duplicate receipts cannot stop or launch.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage -run 'TestRecovery|TestResumeCandidates|TestMenuRecovery|TestSlotOperation|TestHubMutation' -count=1` to RED.
- [ ] Route choices and confirmations through the existing queue and shared executor; remove M1-only choice suppression for expired/dead-owner Starting. Show actual launch/registration result and retained foreign context. Re-run to PASS and commit.

### 5. Headless acceptance, documentation and close

Files: extend `couchcore/recovery_resume_integration_test.go`, `recovery_conformance_live_test.go`, `couchcmd/recovery_acceptance_test.go`; update `atlas/couch.md`, `atlas/session-identity.md`, `atlas/index.md`; add `workshop/plans/000556-live-acceptance.md`.

- [ ] Headless stateful matrix: proven, best and fourth alternative, no candidate, incomplete/unsupported scan, missing/expired tty, attributed owner needing confirmation, foreign survivor, expired claim, cancelled/partial effects and uncertain registration. Assert executed identity, one writer per resumed transcript, one native launch, preserved evidence and no unintended signals/files/default edits.
- [ ] Run targeted recovery packages to PASS, then `sdlc local-check run --name suite` for the final head and `make build`; retain commands/results in the issue Log. Merge main before final verification if it moved; suite evidence must cover the landing head.
- [ ] Map selection/confirmation lifetime, schema migration and uncertain launch outcomes in atlas; link the newer target. Record the short scratch script below with exact final commands and expected notices.
- [ ] Close normally with `sdlc close --issue 556 --verified '<recorded evidence>'`; answer the binary-owned fresh-context review, no duplicate ad-hoc boundary review or inherited #528 waivers. Land on SHIP through normal PR/CI gates. Report landed PR and real merge hash to hub only; no stage mutation or operator restart.

## Morning live script outline (final executable commands supplied before close)

TL/operator only, after morning Couch restart on the landed build. Use a newly allocated scratch slot, never pair:1/pair:6 or another working slot.

1. Verify pre-existing live rows are Continue, attach automatically and still message; no startup recovery effects.
2. In scratch, finish a native turn and cleanly stop it. Run `couch --resume-candidates SCRATCH --json`; `couch --resume SCRATCH --json` resumes the proven identity with no prompt.
3. Use the headless fixture recipe on scratch to expose best/alternative transcript evidence; inspect menu and JSON, choose the exact printed candidate command, and verify the resumed native ID. No automatic inference.
4. Scratch only: remove/withhold *fixture* tty evidence and use no/unsupported scan evidence. `couch --reboot SCRATCH --confirm --json` starts fresh with branch/issue and an honest missing-screen notice.
5. Scratch attributed orphan: inspect the exact named stop set; cancel once (no effects), then confirm via person/menu and via authenticated hub/token. Native ID has one owner afterward.
6. Scratch foreign generation: Fresh leaves it untouched, reports it and uses a fresh terminal name. Clean up only the scratch fixture processes that this script created, using exact identities.

## Revisions

### 2026-10-15T05:33-07:00 — initial design

Built from source audit at `09c5b48dd7586d6ba908a45ea86899190ad3dd4a`, including independent core/surface audits. Proposed for hub approval before change-code. No changes to the targets: the old continuity document's observe-only startup wording is superseded by accepted #528 auto-attach and the newer recovery target; this issue preserves that behavior.
