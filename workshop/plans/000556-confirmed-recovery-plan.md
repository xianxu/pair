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
| Native admission reservation transitions | new `cmd/internal/launcher/native_admission.go` | new |

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
| Native admission store | new `cmd/internal/launcher/native_admission_os.go` | new | portable bounded reservation file under Pair cache and shared flock |
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

### Concurrent native admission

Source inspection found a material gap: `recovery_native_owner.go` expressly implements an observation, not a lock against cross-thread creates. Two slots can both pass before either wrapper exists. Per-thread holds and terminal leases do not enforce the issue's one-writer requirement.

Add one shared native-admission owner at Pair's common native-resume boundary, including ordinary proven resumes. Under a shared cache flock, reobserve and reserve `(agent, native ID)` for the exact attempt/helper before launching a wrapper. A live or unresolved reservation refuses another native launch, with Fresh still available. Transfer ownership to the exact wrapper before native exec; its independently observable readiness/ledger generation permits reservation removal under the same lock. A helper's death alone cannot release a reservation while its spawned server/wrapper may still start. Unknown effect retains the reservation until the exact attempt tree is proved absent or explicitly retired. A bounded before-exec wrapper check rejects cancelled/fenced reservations.

The store is one versioned, atomically replaced file, capped at 256 active/unconfirmed entries and 1 MiB; no entry per ordinary fresh conversation. New admissions collect only exact proved-dead attempts; explicit retirement clears its exact reservation. Capacity exhaustion refuses disk resume honestly and offers Fresh. Never expire a reservation solely by wall clock. This additional durable state is required for cross-process uniqueness, not confirmation persistence: approvals still die with Couch (ARCH-ORDER, ARCH-FUNERAL). Test simultaneous same-ID admissions with barriers, different IDs independently, helper death with surviving child, missing readiness and stale completion.

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

- [x] Add failing pure tests: three selection kinds; strict inferred capture unchanged; proven without tty; fresh without tty; unsupported future record; schema-3 read without writes.
- [x] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'TestRecoverySelection|TestRecoveryContext|TestSlotRecoverySchema|TestThreadRecord' -count=1`; confirm the new cases fail.
- [x] Implement typed variants, version migration and pure transition; re-run to PASS. Commit with issue reference and agent trailers.

### 2. Shared preparation, exact confirmation and stale Starting

Files: new `couchcore/recovery_confirmation.go`, `recovery_confirmation_test.go`; modify `recovery_preview.go`, `recovery_resume.go`, `recovery_native_owner.go`, `actionableinventory.go`, `starttransaction.go`, `lifecycledebris.go`; extend their colocated tests.

- [x] Add failing sequence tests for expired/replayed confirmation, changed PID/start/nonce, changed candidate, wrong hub/target, cancelled acceptance and stale Starting with live supervisor.
- [x] Run `go test ./cmd/internal/couchcore -run 'TestRecoveryConfirmation|TestRecoveryPlan|TestStartClaim|TestStarting' -count=1` to establish RED.
- [x] Implement preparation/capability ownership and pure expiry projection; keep Continue first and startup read-only. CAS-fence stale claims only on explicit recovery. Re-run to PASS and commit.

### 3. Preservation, retirement and one native launch

Files: `couchcore/recovery_resume.go`, `recovery_evidence_os.go`, `reboot.go`, `slotrecovery.go`, `threadstore.go`, `launch_existing.go`, `couch.go`; `launcher/recovery_terminal.go`, `recovery_terminal_os.go`, `recovery_authority.go`, `recovery_authority_os.go`, `createflow.go`; new `launcher/native_admission.go`, `native_admission_os.go` and tests; `wrapcmd/wrap.go` plus new admission adapter/tests; artifactpath cache resolver/manifest; corresponding recovery/native-owner/start tests.

- [x] Add stateful failing cases: preserve failure before signal; owned damaged wrapper; foreign remnant left byte/process-identical; new descendant/PID reuse mid-retirement; partial retirement; proven resume without screen; fresh incomplete/unsupported scan/missing screen; delayed registration after a real launch.
- [x] Run `go test ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/orientation -run 'TestRecovery|TestResumeCandidate|TestReboot|TestStartTransaction' -count=1` and confirm the new cases fail.
- [x] Implement shared ordering using existing journals, park/exact retirement and launch APIs. Every native resume has empty Prompt; every Fresh uses BuildRecoveryOrientation with available coordinates only. Reobserve uncertain attempts once, never kill/fresh-replace merely because registration timed out. Re-run to PASS and commit.
- [ ] Before claiming uniqueness, run a deterministic two-slot barrier test: the same native ID admits one writer; different IDs progress independently. Add reservation handoff/death/unconfirmed tests and verify the production wrapper cannot execute a fenced attempt. Commit the common admission protocol separately.

### 4. Menu, CLI and authenticated hub parity

Files: `couchcore/ops.go`, `operationdispatch.go`, `slot_operation.go`; `couchcmd/cli.go`, `resume_candidates.go`, `slot_operations.go`, `console_slot_operations.go`; `couchmessage/protocol.go`; `couchtty/menu_recovery.go` and existing confirmation/menu state files; colocated declaration/protocol/hub/menu tests.

- [x] Add failing tests: Fresh command never has empty candidate key; all scan failures and missing tty retain executable Fresh; default/four alternatives stable; owned stop summary shared by menu/CLI; hub ownership changes and duplicate receipts cannot stop or launch.
- [x] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage -run 'TestRecovery|TestResumeCandidates|TestMenuRecovery|TestSlotOperation|TestHubMutation' -count=1` to RED.
- [x] Route choices and confirmations through the existing queue and shared executor; remove M1-only choice suppression for expired/dead-owner Starting. Show actual launch/registration result and retained foreign context. Re-run to PASS and commit.

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

### 2026-10-15T05:38-07:00 — native concurrency audit

A bounded follow-up verified that current native checks do not serialize competing slots. Added an explicit bounded native admission reservation protocol to meet the existing no-two-writers Done-when. This is a design addition for hub review, not implemented code or a claim that the existing gate already provides exclusion.

### 2026-10-15 — approved direction and two review boundaries

Hub receipt `249a1bbd-5f2f-457a-b009-c2581eae3da9` approves implementation with
these superseding changes:

- **M1** is tasks 1–4 plus task 5's M1 headless acceptance, atlas and scratch
  script. It excludes the native admission store/hand-off work in task 3.
  Close with `sdlc milestone-close --issue 556 --milestone M1`.
- **M2** is the native admission reservation protocol, common Pair/wrapper gate
  and deterministic two-slot barrier/death/uncertain-readiness tests. It has its
  own `sdlc milestone-close --issue 556 --milestone M2`, then whole-issue close
  and normal landing on SHIP. M1 does not claim to fix cross-slot exclusion.
- `couch --recovery-confirmation TOKEN` must work **standalone** for the
  authenticated owning hub: the offer binds its target and selection so no UI
  or repeated selection arguments are necessary. The CLI's optional token
  argument on resume/reboot may share this owner; no duplicate execution path.
- Keep user-visible outcomes exactly Continue / Resume from disk (confirmed) /
  Start fresh, plus needs relaunch. Internal capabilities, reservations and
  schemas add no visible state or extra prompt beyond the stop confirmation.
- Keep the proposed five-minute Starting presentation bound and one exact
  reobservation. Supply the final scratch script at M1; live effects still wait
  for the hub/operator's morning restart.

This supersedes the original single-boundary task heading and close instruction.

### 2026-10-15 — plan-quality round 1 dispositions

**PQ-1 (ownership-survives-owner-handoff), addressed:** supersede the reservation
removal-on-readiness rule. The durable native admission entry remains while its
native writer can exist, even after successful readiness. Its tagged states are
reserved (exact Pair helper), spawning (exact wrapper; child identity not yet
known), and running (exact wrapper plus native child PID/start identity).
Before native start, the wrapper commits spawning; after spawn it publishes the
exact child's PID/start identity under the same native key and attempt CAS.
Running reservations are reclaimed only when the recorded native child AND
wrapper are proved dead; wrapper death alone never releases a live or unknown
child. A spawning entry whose wrapper dies before publishing child identity
remains unresolved, including across Couch restarts: no timeout-based release or
native relaunch. Fresh under a different native identity stays reachable; a
bounded store is preferable to inventing death evidence. Reserved entries may be
released only before wrapper execution is possible, or after the exact pending
launch is fenced and its attributed launch tree proved absent. Ownership checks
for pre-upgrade writers remain necessary alongside this common-boundary guard.

`AdvanceNativeAdmission` is the pure transition authority; every mutation of the
portable admission store calls it while holding its global store lock. Sequence
tests control handoff, wrapper death, surviving/reparented child, delayed child
publication and duplicate/late completion. The independent invariant is that
no second admission of the same native identity succeeds while the first
writer's death is unproved. A running entry holds only a bounded identity tuple,
not a transcript or screen; explicit retirement and next admission collect exact
dead generations. This applies to M2, not the M1 confirmation milestone.

**PQ-2 (function-level-test-strategy), addressed:** the test cases in the
original task prose are illustrative only; the following named-function
strategies replace them as the implementation/testing contract:

| Production functions | Adversarial strategy and mechanical oracle |
|---|---|
| `DecideSlotRecovery`, new `DecidePreparedRecovery` | Generate combinations of selection kind, incomplete evidence and owner knowledge; assert Continue requires positive listing, inference is explicit, Fresh remains offered |
| `ValidateRecoveryContext`, `ValidateRecoveryIntent`, `DecodePersisted` | Fuzz strict tagged/versioned records and malformed physical-file references; assert old native authority never weakens and future versions never gain write authority |
| New `AdvanceRecoveryExecution` | Drive all external completion/cancellation/drift orderings through the reducer; assert preservation precedes effects, each attempt launches at most once, and uncertainty never becomes absence |
| New `AdvanceRecoveryConfirmation`, capability store `Issue`/`Consume` | Fake-clock/entropy event sequences and binding substitution; assert expiry, single consumption and exact actor/selection/stop-set match before any signal |
| `startInFlight`, new `DecideStaleStart`, `AdvanceStartTransaction` | Clock and delayed old-driver events around the five-minute bound; assert expiry exposes an action but never kills the supervisor or grants a stale nonce authority |
| `ResumeCandidate` and shared prepared executor | Stateful portable store/process/runner worlds with barriers between preservation, CAS and effects; assert selected native identity, no unaccepted signal and evidence retention under interruption |
| `attributeRecoveryServers`, terminal driver `prepare`/`retire` | Mutate process identity/ancestry and membership at each lease/signal boundary; assert only the accepted set may be signalled and unknown/foreign members remain untouched |
| `ValidateRecoveryAuthority`, `launchTrackedThread` | Delay or lose exact-attempt registration and mutate the claim before native exec; assert no duplicate launch, no uncertain destructive cleanup and no stale authority |
| `ParseCLI`, `ValidateRequest`, `PrepareSlotOperation`, `reduceRecoveryKey` | Closed-shape generation plus actor/target/token substitution; assert menu/CLI dispatch the same prepared owner and Fresh never emits an empty candidate command |
| `AdvanceNativeAdmission` and common Pair/wrapper admission adapter | Two-slot barriers and controlled handoff/death/publication sequences; assert same-ID exclusion, different-ID independence and retention through unknown child liveness |

**PQ-3 (external-conformance-cadence), addressed:** run opt-in scratch lifecycle
conformance after changes to native CLI composition, readiness, Zellij lifecycle
or OS process attribution, before that change's release. The implementing slot
records headless conformance evidence; the hub schedules the operator restart
check. CI's ordinary suite runs the same portable stateful seams without live
agents. Morning acceptance is the first release check, not an unattended cron
or permission to touch working slots.

### 2026-10-15 — M1 integration refinements

Fresh always allocates a new tag and archives the exact retained source through
the existing journal. This preserves ordinary reboot's metadata/native-file
contract and also handles unknown remnants without namespace reuse. Tagged
orientation is read through a bounded, physically validated retained-capture
reader; it never authorizes inference for a different tag. Missing-record Fresh
uses the same #522 Prompt and exact Fresh readiness instead of a recovery-only
delivery route. Saved configuration remains first; Fresh-only missing-profile
fallback uses normal read-only path/default resolution. No defaults are edited.

The existing `DecideSlotRecovery` continues to own choices, with Proven added
to its IO-backed preview. `AdvanceRecoveryExecution` in `recovery_plan.go` owns
the internal preservation/retirement/claim/launch order; confirmation has its
separate pure single-use transition. No parallel durable recovery lifecycle or
second classifier is introduced (ARCH-DRY). Existing claim/park reducers retain
their authority. Native open parks/continuations refuse before any stops; Fresh
can archive their exact retained bytes instead. Host repair, when needed, occurs
only after preservation and confirmed retirement, before the new Fresh claim.

Legacy direct `Couch.Reboot` API coverage retains its archive-only and old
reconciliation contracts. Public menu/CLI recovery dispatch is tested separately
through the shared prepared executor, including strict Fresh registration; legacy
fixtures do not pretend that spawning a client proves new-agent registration.
The live script is `workshop/plans/000556-live-acceptance.md`; rare destructive
corners run in private stateful headless fixtures, not a working live slot.

### 2026-10-15 — full-suite integration correction

The recorded full suite rejected retaining an alternate Reboot implementation
that production no longer called. Replace it with a thin production facade over
prepared Fresh, preserving only the existing primary no-path/unreadable raw
archive exit (no stops or launch, exact-byte CAS, future-schema refusal). This
supersedes the prior direct-legacy-test strategy: tests now exercise the shared
shipped path. Existing explicit Fresh agent choice still resolves ordinary saved
configuration; no argv or default edits enter recovery.

Measured recordless foreign/unknown-owner Fresh now uses distinct-tag admission
and the same claimed launch/readiness pipeline with a read-only verified-checkout
readiness adapter. It never runs workspace convergence beneath those remnants;
notice discloses that setup was not refreshed. Ordinary non-recovery Fresh keeps
its strict absence gate. Resume routing must retain positively listed warm
adoption and typed refusal advice while cold native resume uses only the prepared
owner. Menu requests show only the eventual exact-stop confirmation, removing
the old preliminary generic Fresh prompt. No new user-visible state is added.

### 2026-10-15 — M1 boundary round 1 corrections

The first M1 review returned REWORK (BR-1/2/3). The scope remains M1; no
admission-store work has started. Both retained-record and recordless Fresh must
use the shared read-only checkout readiness whenever previous owner absence is
unproved, including its final recheck. A new tag isolates terminal identity,
not the physical workspace, so it never authorizes normal setup convergence.

Retirement must independently observe absence of every accepted process
identity, including survivors after a server exits. The confirmed launcher
retires only immutable accepted generations, leaf-first, checking ownership
before each signal. Strict legacy retirement gains an absence oracle without
broader kill authority. Settled connected source helpers are named in the same
confirmation and use normal ParkExpected after durable intent/screen preservation;
a failed/uncertain park keeps its transaction and never falls through to a
second retirement strategy. The terminal plan then independently proves its
accepted owners gone. README and older Atlas recovery accounts are updated to
the actual shared path and token surface.

Core concept locations: `recovery_prepared.go` owns the prepared executor;
`recovery_retirement_os.go` bridges normal park and confirmed terminal evidence;
`launcher/recovery_terminal_confirmed.go` owns immutable process attribution
and confirmed retirement. These supersede the initial coarse task path list.

### 2026-10-15 — read-only M2 native-selector audit (policy clarification pending)

No M2 code has started. Installed help measurements (`codex resume --help`,
`claude --help`, `grok --help`) confirm native selectors can denote names,
search/pickers or cwd-latest, rather than an exact native ID. The existing
extractor can even read Codex `resume --last` as the literal ID `--last` and can
miss Muse-specific global prefixes. A reservation must never use those aliases
as distinct keys for one transcript.

Proposed policy for the hub: a normal Pair native-resume launch must provide an
explicit ID that the existing native inventory validates as the canonical root;
ambiguous/indirect selectors refuse before launch with exact-ID or Fresh advice.
Do not guess latest or silently change user arguments/defaults. Reuse the
resumeform selector table and existing option-value/Fresh validation so no
second spelling authority appears. Recovery's proven/selected IDs are already
explicit. This affects direct human Pair CLI selectors, so confirm that policy
with the hub before implementing it. Caller-chosen new-conversation IDs and fork
semantics must remain distinguished from a disk resume; do not treat their
source ID as a second writer without measured evidence.

The wrapper audit confirms one native `pty.Start` site for argv and encoded
launch commands. Admission transitions must enclose that boundary, not rely on
best-effort readiness afterward. Strip old admission capability from native
child/watcher environment and same-PID Fresh exec. Do not independently change
existing readiness nonce semantics without proof that the reservation protocol
needs it; each actual native-resume launch gets its own admission attempt.

### 2026-10-15 — M2 selector policy approved: resolve and pin

Hub receipt `14aba7d7-b325-43dc-a16c-6d8d23907f23` supersedes the preceding
proposal to refuse all indirect selectors. Preserve the human's `--continue`
and `resume --last`: resolve the selector through the existing native inventory
using its own meaning, latest eligible session for the exact cwd and agent.
Require one exact canonical root ID, reserve that ID, then replace the indirect
selector in this invocation with the explicit ID. The native CLI therefore
cannot make a different choice after admission. This is temporary launch
normalization, not an edit to saved startup parameters/defaults.

Refuse with exact-ID/Fresh advice only when resolution cannot yield exactly one
ID (none, ambiguity, or names/search terms the inventory cannot map faithfully).
Never use a name, search string, `--last` or an empty value as an admission key.
Fix the extractor and share its selector/option-value authority rather than
creating a second spelling table. New-conversation `--session-id` and forks
remain distinct from resuming the original transcript.

Add deterministic tests for exact cwd/agent and eligible-root filtering, latest
selection and ties, absent/incomplete/unsupported resolution, unsupported name
mapping, preservation of unrelated argv/prompt, and a newly arriving session
between resolution and execution: the admitted explicit ID must remain pinned.
The same pinned ID must appear in reservation, launch argv and durable launch
observations. No new user-visible state or extra prompt is introduced.

M1 still awaits independent #568's landing, the merged-head recorded suite and
normal boundary re-close. This policy approval authorizes M2's selector behavior;
it does not claim M1 closed or start M2 code prematurely.

### 2026-10-15 — BR-4: keep synchronized owner identity in preview resolution

M1 round 2 accepted the three earlier corrections but identified copying the live
Couch (including mutexes) to substitute a read-only store. Pass the preview store
through the existing profile resolvers instead; keep the original Couch owner,
use an independently constructed focused fixture, and verify copylocks plus the
profile/preflight matrix. No lifecycle semantics, M1/M2 split or approval changes.

### 2026-10-15 — BR-5: normalize every Fresh representation before native guards

Every validated Fresh representation, including a keyed candidate from the menu
or printed CLI command, resolves to Fresh before profile fallback and mode-specific
admission. Native-only unfinished park, continuation and start-claim guards remain
intact for genuine transcript choices. Regression coverage compares explicit Fresh,
ResumeCandidate with the advertised Fresh key, and the declared operation route;
menu/printed-command tests cover the transport edges and keyless Fresh. Invalid or
stale keys still refuse without effects. No M2 work or approval change is included.

### 2026-10-15 — measured latest-selector facts, pending provider policy reply

Read-only preparation only; no M2 implementation or policy override. The hub
clarification is submitted as `32602d27-64b2-42a4-972f-c2e3e7c10cff`.
Existing canonical-root validation covers exact IDs for all six agents. Recovery
inference eligibility/activity is not native latest selection.

- **Claude 2.1.299:** shipped readable JS at
  `~/.local/share/claude/versions/2.1.299`, call chain
  `zTe(undefined) -> PZr -> Vro -> tNe -> xCt`, with `Zls/wbt` eligibility.
  `xCt` at byte 202470269 stats UUID JSONL files; `tNe` at 202477625 orders by
  descending mtime, then creation time. Filters include sidechain/team/daemon,
  inappropriate SDK/loop entrypoints, bookkeeping-only/superseded/background
  sessions and cwd slug collisions. Scope can include aliases/relocations;
  do not silently replace these with max semantic message time.
- **Qoder 1.1.71:** shipped readable JS at
  `~/.qoder/bin/qodercli/qodercli-1.1.71`, call chain
  `continue -> sS -> pB/Gj -> Zy -> JKe`. `JKe` at byte 123692726 uses transcript
  stat mtime as lastUpdated and requires displayable content or worktree-session
  metadata. `mke` at 123802179 deduplicates IDs by newest mtime and sorts descending;
  `sS` at 123807731 reloads the first canonical ID. Native scope can span verified
  same-repository worktrees; the approved exact-cwd scope must be explicit.
- **Codex 0.162.1:** matching rust-v0.162.1 source and read-only
  `~/.codex/state_5.sqlite` schema/query measurements establish archived=false,
  nonempty preview, source cli/vscode by default; include-non-interactive admits
  cli/vscode/exec/mcp (AppServer maps to mcp, not appServer). Order is
  updated_at_ms descending, with no explicit ID tiebreak. Fetch two and refuse
  equal best timestamps under the unique-resolution rule. Effective provider
  filtering depends on invocation/profile/managed/cwd configuration: an absent
  provider field defaults to openai, whereas unsupported config/read means no
  filter. SQLite home precedence is effective sqlite_home, CODEX_SQLITE_HOME,
  then resolved Codex home. Unknown configuration/schema/remote or unsupported
  scope modifiers cannot be guessed. Selected rollout path and ID still need
  existing canonical-root validation; metadata paths never authorize arbitrary
  transcript reads. Native unusable-path scan-and-repair is not a read-only
  resolver operation. Primary sources:
  [thread queries](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/runtime/threads.rs),
  [source filters](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/app-server/src/filters.rs),
  [provider selection](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/app_server_session/provider_selection.rs),
  [configuration](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/core/src/config/mod.rs).
- **Grok 1.0.50 / Muse 1.4.9-R6297.1 / Agy 1.2.16:** installed help establishes
  latest selectors, but a faithful ordering/filter resolver remains unproved.
  Grok's human sessions list and embedded updated_at/last_active_at SQL do not
  establish its continue call chain. Do not infer support or substitute mtime.

Future implementation uses an explicit resolved/none/ambiguous/incomplete/
unsupported inventory result, resolves once, revalidates that exact ID, then
reserves and executes the same ID. A newly arriving session cannot change it.
No transcript bodies/configuration contents were printed, no native conversation
was launched, and no native files/database were changed by these measurements.

### 2026-10-15 — M2 implementation seams after normal M1 SHIP

M1 finalized normally at `69c213c289ac9f98aa7e16a9e9c1b54f76c90d26`, all five
findings disposed. M2 now implements the approved reserved/spawning/running
protocol in launcher (already imported by wrapcmd), with a portable cache store,
opaque grant/claim, full final argv digest and strict process generations.
The shared Runtime adapter reserves after exact selector resolution; the sole
wrapper PTY spawn consumes once before native exec and publishes its strict
child before readiness. Strip authority from child/watcher/Fresh environments.
Cancellation can release only a definitely unexposed reserved grant; uncertainty
retains ownership. No readiness or timeout release, default edit or new UI state.

Read-only audit identified two existing-owner integration obligations: the current
legacy checker discovers live wrappers first and therefore misses a retained
native child after wrapper death; its unknown-ready wrapper refusal also cannot
use a new reservation's known different key. Reuse bounded canonical owner
namespace discovery plus retained readiness/ordinal ledger association to check
known orphan children and reobserve their exact generations. Pass an immutable,
private-field NativeAdmissionOwners view to the pre-upgrade callback under the
same store lock. Only exact different-key admitted wrapper generations may skip
best-effort readiness; running coverage must still match its live child lineage,
so a same-PID Fresh re-exec is not exempted by an obsolete running entry. Keep
unknown pre-claim/legacy ownership conservative, without arbitrary process-name
blockers, new persistent indexes or new argv attestation mechanisms.

M2 tests add these orphan/pre-readiness/Fresh-generation cases to the approved
same-ID and different-ID barrier, delayed wrapper, surviving child, cancellation,
PID reuse, capacity, uncertain publication and replay sequences. This completes
the existing no-two-writers contract; it does not start the separate endpoint
rebuild milestone or change the pending latest-provider policy question.
