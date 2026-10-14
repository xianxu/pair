# Several Hubs in One Couch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose subagent-driven-development or executing-plans. Track the steps below; SDLC owns review boundaries.

**Goal:** Run several explicitly selected hubs in one Couch, with durable per-slot ownership, isolated automatic routing and management, and visible hub grouping.

**Architecture:** One stable slot-to-hub fact feeds pure routing/authorization and presentation. Reuse the current console, operation queue, authenticated caller checks, journal writer and launch/recovery operations; do not fork a supervisor per hub. The no-argument launch path remains the existing path.

**Tech Stack:** Go, existing Couch message broker and terminal renderer, portable JSON persistence through the existing store transaction seam.

> **Superseded draft:** The design and tasks below this notice, through the initial Revisions entry, are retained as history only. The operator replaced the persistence model. Execute only the revised plan under **2026-10-14 — hub-owned claims and specificity** at the end.

## Scope and decisions for approval

The issue's Spec is the requirements authority. This is one atomic implementation and close review, not separate milestone deliveries. No code until the hub approves this plan and `sdlc change-code` passes.

Recommended approach: a **stable slot assignment record**, separate from a conversation record. Today a numbered slot's `thread.json` is replaced on fresh/reboot; putting its hub only there would require copying it through every recovery path. Store `SlotHubRecord` entries in one namespace-owned `slot-hubs.json`, keyed by canonical repository identity plus worktree root, including primary slots. This is the slot's ownership record, not a per-repository setting. A new conversation, tag, agent, or wrapper never changes it.

Alternatives considered: (1) embed ownership in ThreadRecord: initially fewer plumbing changes, but ownership follows the wrong lifecycle and couples to #528; (2) infer hub from repository or process launch directory: smaller, but cannot divide one repository's slots and loses assignments on restart. Neither meets the Spec.

Explicit directory arguments designate the ordered active hub set. Resolve each directory through existing workspace/Git identity; reject duplicates (including symlink aliases), files and invalid roots before launching any child. The first hub is focused initially. Each explicit hub uses existing StartInteractive and attach operations, with no new resume/fresh decision policy. Additional hubs start serially through the existing queue; failure leaves already attached hubs usable and gives the failed hub's cause and existing retry action. No rollback kills an already established session.

**Pending hub clarification:** after using multiple hubs, bare `couch` keeps today's implicit current-directory hub, rather than auto-launching remembered hubs. Persisted assignments to an omitted hub remain assigned and show `hub unavailable`; they never silently become the current hub's. Running `couch . ../bench` restores both active hubs. With no multi-hub metadata, all existing default behavior, output and notification titles remain unchanged. This question was sent before implementation.

Unassigned pre-feature slots inherit the first/default hub when first admitted to the multi-hub model. A newly started slot inherits the initiating slot's hub, or the focused slot's hub for an operator action; capture that identity before queueing. Merely focusing, attaching, resuming, or restarting an already assigned slot never reassigns it. Explicit hub slots own themselves and cannot be moved below another hub. Non-hub rows have a switcher `hub` action: select a declared hub and press Enter; Escape cancels. The row moves only after durable acknowledgment.

Implicit family sends (`--send-to pair`) select within the sender's hub; explicit slot sends (`--send-to pair:2`) may cross hubs. Read-only inspection still sees everything. Preserve existing command admission requirements, then add a common same-hub requirement to remote mutations: switch-agent, relaunch, reload-context, prepare, resume, reboot, reap and set-model when #544 lands. Do not silently make currently worker-available recovery commands hub-only. The physical operator's switcher remains able to manage all slots. Hub-only commands still require a verified hub identity, not a caller-supplied hub field.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| SlotHubRecord / HubAssignmentState | `cmd/internal/couchcore/hub_assignment.go` | new |
| HubProjection | `cmd/internal/couchcore/hub_assignment.go` | new |
| ThreadPresentation / StatusActor | `cmd/internal/couchtty/thread_presentation.go`, `reserve.go` | modified |
| Hub selection reducer | `cmd/internal/couchtty/menu_hub.go` | new |

`SlotHubRecord` contains validated stable slot and hub keys, and a revision. Slot keys use canonical repository identity and worktree root, never public aliases or thread tags. A hub key references a self-owned slot; ownership is direct, never a chain. One slot has exactly one hub; one hub has many slots. `HubProjection` is an immutable view over active hubs and assignments, with known/unavailable/unknown distinctions and pure functions for hub resolution, family candidate eligibility and mutation eligibility. Future growth, if required, is at this boundary, not in individual handlers (ARCH-DRY, ARCH-PURE).

`ThreadPresentation` derives sections from that projection; retain today's per-section repository sorting and indentation. `StatusActor` carries derived other-hub state. With one hub, the renderer emits the existing bytes. With several, other-hub chips (including hub chips and glyphs) receive one modest fade level composed with existing attention/idle styles; preserve reverse-video hub marking, blocked/attention cues and hit spans. Focusing a slot immediately changes the fade projection, without IO.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Slot hub persistence | `cmd/internal/couchcore/hub_assignment_store.go` | new | namespace ThreadStore lock, journal and portable JSON file |
| Explicit hub startup | `cmd/internal/couchcmd/hubs.go`, `cli.go`, narrow hooks in `run.go` | new / modified | existing StartInteractive, console attach and operation queue |
| Hub authority adapter | `cmd/internal/couchcmd/hub_authority.go`, `message_service.go` | new / modified | authenticated workspace identities and current assignment snapshot |
| Routing and management consumers | `couchmessage/broker.go`, `routing.go`; `couchcmd/agent_switch_service.go`, `slot_operations.go`, `notify_service.go` | modified | broker candidate selection, operation admission and execution |
| Supervision consumers | `cmd/internal/couchcmd/wait_notice.go`, `supervision.go`, `supervision_model.go` | modified | existing wait notices, stuck-slot nudges and restart prompts |
| Console projection and assignment action | `cmd/internal/couchtty/console_presentation.go`, `console_menu.go`, `menu.go`, `menu_actions.go`, `menu_render.go`, `console_palette.go` | modified | menu events, async completion and rendered terminal frames |

Use the existing stateful message test world, SlotCatalogFake, operation queue fixtures, real temporary ThreadStore and controlled terminal handles. No new external service or subprocess protocol. Real launch conformance uses the repository's isolated terminal fixture with private stores and stripped live COUCH/PAIR variables; never starts or changes the operator's live slots (ARCH-MOCK).

## Persistence, authority and ordering

Reuse namespace ThreadStore lock/journal to atomically replace the assignment document, bounded to 128 slot records (current broker MaxActors) and 1 MiB encoded bytes; validate both before writing. One record per stable slot is replaced, not appended per launch. At capacity, keep current assignments usable and refuse new assignment with `hub assignment store full; remove an unused slot assignment in the switcher`. Provide that action for disconnected stale entries; never evict a live assignment automatically. The switcher may reset/remove a stale assignment only explicitly. Capacity/cleanup must be tested, not left as advice without an action (ARCH-FUNERAL, ARCH-CONSTRAINTS).

Missing metadata means legacy compatibility, not corruption. Strictly decode version, field keys, unique slot keys, identities, revisions and direct hub references. Corrupt/unsupported metadata leaves the console and read-only inspection usable, displays the path/cause, and refuses automatic routing/mutations that need unknown ownership. Do not reset or fabricate ownership. No credentials are created. Existing authenticated workspace and incarnation checks remain required; JSON/env/UI labels never grant authority (ARCH-SECURE).

State/event contract (ARCH-ORDER), owned by the pure assignment transition function and committed through the store:

| State/event | Result and effects |
|---|---|
| Unassigned + admit under known initiator hub | prepare revision-checked assignment; persist before authorizing or launching |
| Assigned + reopen/reboot/new thread | preserve existing assignment; no ownership write |
| Assigned + operator chooses another active hub | prepare compare-and-swap against current revision; publish new projection only on confirmed write |
| Same assignment repeated | idempotent, no extra row or launch |
| Write fails before commit | keep old projection; show reason and `retry hub change` |
| Commit acknowledgment uncertain | reload exact record under store lock; show saved or unchanged only with evidence; otherwise show `hub assignment unknown; retry after inspecting store` |
| Stale completion / stale selection revision | ignore projection update; refresh current record, no rollback of newer assignment |
| Hub omitted or disconnected | retain assignment; reports say hub unavailable, never fall back to another hub |
| Slot reassigned while a mutation is queued | re-check current owner at execution; refuse if caller no longer owns target |
| Reassignment during a running mutation | serialize ownership commit with that slot's operation gate; do not switch authority halfway through an effect |
| Assignment changes while automatic notice is pending | recompute destination before admission; previously dispatched messages retain their exact recipient/evidence |

Broker sends retain normal receipt, timeout and uncertainty semantics. Explicit cross-hub messages do not become same-hub supervision replies. Track each worker's own hub for report acknowledgment and prompts; one available hub must not trigger another hub's restart prompts. Hub buckets apply to every verified active hub. Notification title prefixes the hub label in multi-hub mode, including custom titles; preserve today's exact single-hub title behavior and existing control stripping/limits.

Snapshot refresh and rendering perform no disk/process work per keystroke. Startup validation/launch is serial and uses existing bounded operations; no extra timer or probe per hub. Normal deployment has a handful of hubs and at most current MaxActors slots. Tests exercise that full bound. CPU/memory scale linearly with those slots, no all-pairs probes. Cast singleton ownership, machine test locks and session-sync ownership are unchanged (ARCH-PURPOSE).

## Implementation steps

### 1. Slot assignment core and durable store

Files: create `couchcore/hub_assignment.go`, `hub_assignment_test.go`, `hub_assignment_store.go`, `hub_assignment_store_test.go`; adapt existing store journal/source inventory only as required.

- [ ] Write pure tests for two hubs sharing one repository, default projection, self-owned hubs, inheritance, same-hub management, unknown ownership and revision transitions. Verify tests fail before implementation.
- [ ] Implement the pure key/state/projection boundary. Do not let handlers write assignment maps directly.
- [ ] Test actual temporary-store round trip, process reconstruction, fresh thread identity, malformed/older files, duplicate keys, size/capacity, CAS conflict, fault at each journal publication point and explicit stale-row removal. Implement using existing durable writer and lock.
- [ ] Run `go test ./cmd/internal/couchcore -run 'TestHub' -count=1`; expected PASS after RED. Commit the coherent core/store change.

### 2. Startup, routing and management

Files: `couchcmd/cli.go`, `cli_test.go`, `hubs.go`, `hubs_test.go`, narrow `run.go` wiring, `hub_authority.go`, `message_service.go`; routing and management files in the table above; colocated tests.

- [ ] Add parser tests for no args, one/multiple dirs, layout flags, `--`, unknown flags and duplicate canonical roots. Pin no-argument invocation equality and existing launch/attach call sequence.
- [ ] Implement multi-directory startup adapter, retaining the existing no-argument branch verbatim. Publish hub configuration before peer routing begins. Preserve single initial attach/background recovery ownership; extra hubs must not trigger extra reattach passes.
- [ ] Test inheritance from an initiating worker, UI focus changing while a start is queued, existing assignment on resume, partially failed hub startup, and restart with changed thread tags. Add only a narrow dispatcher hook that supplies ownership context; do not edit resume proof algorithms.
- [ ] Test same-repository family selection scoped by hub; explicit cross-hub sends; `hub` relative routing; all management verbs at admission and queued execution; reassignment races; hub-only bucket and status access without weakening existing caller proof.
- [ ] Wire one authority lookup to all enumerated consumers. Include #544 set-model if merged; coordinate its same guard if still in flight.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'Test(Hub|Message|SlotOperation|AgentSwitch|Notify|ParseCLI)' -count=1`, then focused races. Commit.

### 3. Supervision and UI

Files: supervision and console files in the table; new `couchtty/menu_hub.go`, `menu_hub_test.go`; extend `hub_section_test.go`, `hub_inversion_test.go`, wait/supervision/notify tests.

- [ ] Add two-hub wait/nudge/report/reply/restart sequences, including unavailable hub, cross-hub reply, reassignment and delayed connection. Keep existing one-hub timelines and message text byte-equal.
- [ ] Implement per-worker hub lookup for generated sends and evidence. Keep one watcher/coordinator and existing bounded fact logs, not one competing scheduler per hub.
- [ ] Add switcher hub action through the declared operation dispatcher, async revision-checked save, failure/retry projection and stale-entry removal. Hubs cannot be assigned below hubs; selecting the current hub is a no-op.
- [ ] Add pure grouping/frame tests: two hubs with interleaved repository slots, focused worker/hub, attention/blocking, overflow, ANSI/grapheme width, click spans and repeated refresh. Assert byte-for-byte single-hub output against existing fixtures.
- [ ] Implement projection/fade and snapshot propagation. Preserve hub configuration across palette reconstruction (`console_palette.go`). Verify terminal/web cast controls still follow focus with one instance.
- [ ] Run `go test ./cmd/internal/couchtty -run 'Test(Hub|MenuHub|Status|Console.*Hub)' -count=1` and affected supervision tests. Commit.

### 4. Integration, docs and shipping evidence

- [ ] Reconcile main and #528/#544. Avoid `couchcore/startup.go`, `slotrecovery.go`, native binding/session proof files. Startup hooks in `couchcmd/run.go` were reported to the hub before code; discuss any wider overlap before editing.
- [ ] Update `atlas/couch.md`, README launch/switcher examples, CLI help and `cmd/internal/couchcmd/skills/couch/SKILL.md`; keep hub-authority prose consistent. Update exact source ownership in `cmd/internal/artifactpath/manifest.go` for new production files. Reconcile the tables above to final symbols/paths.
- [ ] Run all affected packages, targeted race tests, inventory and clean `make build`; run `make -k test` through the normal machine gate. Record every failure independently; no inherited failure hides the remainder.
- [ ] Isolated terminal integration: launch two temporary hub repos, start one worker from each, verify same-repo assignments, report destinations, read-only cross-hub inspection, refused mutation, switcher reassignment, restart persistence, grouping/fade and focus-following singleton controls. Capture actual rendered frames. Compare the no-argument fixture's output/calls against the unchanged baseline.
- [ ] Update issue Log with measured evidence and exact commands; commit. `sdlc close --issue 546 --verified 'actual evidence'` owns the fresh review; fix its findings. Open PR, require green CI, land only on SHIP, report real merge hash to hub.

## Revisions

- 2026-10-14: Initial plan. Separates stable slot assignment from replaceable conversation state to minimize #528 overlap; preserves no-argument path and requests explicit clarification of omitted remembered hubs before code.


### 2026-10-14 — hub-owned claims and specificity (ACTIVE PLAN)

**Reason:** Operator refinement delivered by ops:0 (8624c810-08e2-4b4c-8606-b305400e94c2, dc433d5b-8668-446d-be63-09fd002bc92d, f6911942-31a8-4b9a-b87c-923f890f7e25). This revision supersedes the entire initial assignment design, entity tables and task list above. No code was written from that draft. There is no slot ownership sidecar, thread field, hub-unavailable state, exclusions syntax, or stale-assignment cleanup UI.

**Goal/architecture:** The first CLI hub owns all slots by default, without storing that fact. Each started hub contributes its own `.hub` claims. Resolve ownership purely from active hubs plus their claims and in-memory moves; all routing, supervision, remote management and display consume that one projection. Keep the single-console/single-cast architecture and the current no-argument launch path.

#### Operator contract and implementation choices

1. `couch . ../bench` starts hubs in argument order. Validate/canonicalize all hub directories before child launch, rejecting duplicate physical workspaces and invalid paths. Reuse StartInteractive and console attachment; startup changes stay in a separate `couchcmd/hubs.go` helper plus CLI/run.go wiring. No changes to `couchcore/startup.go`, slot recovery or resume binding policy; hub approved that seam. Extra starts use the existing serial operation queue and one reattach pass. A later hub failure leaves established hubs usable, names the failed hub and offers the existing retry action.
2. Bare `couch` is the current path, with cwd as the sole hub. It does not read other directories' `.hub` files, restore old groups or write ownership. Single-hub rendering, message text and notify titles remain byte-equal. In single-hub mode no claims are needed. In multi-hub mode load claims from each explicitly started hub, including main's explicit exceptions; main's implicit ownership is not a competing claim.
3. `.hub` is UTF-8 text: one relative workspace repository path (`bench-pair`) or exact slot (`parley.nvim:7`, including `:0`) per line; blank lines and `#` comments allowed. No globbing, shell evaluation, absolute paths, `..` escape or `!` syntax. Resolve against existing WorkspaceIdentity.FleetRoot and catalog, not process cwd or display aliases. Claims may name not-yet-started slots/repos; applying them still requires real catalog identity. Missing file means no claims. Malformed/read-failed files produce a path/line diagnostic before any multi-hub child launch, rather than silently granting main ownership from bad input.
4. Ownership precedence is: explicit runtime move; exact slot claim; whole-repository claim; main fallback. At equal claim specificity, first CLI hub wins and warn with slot/repository and both hub names. A broad claim beaten by an exact claim is not a conflict. Hubs themselves remain self-owned and cannot be moved beneath another hub. Omitted hubs contribute no claims, so their slots naturally fall back to applicable active claims or main. Newly appearing slots use the same projection, including future slots under broad claims.
5. `couch --move-slot repo:N[,repo:N...] --to-hub HUB [--persist]` targets exact canonical slots and an active hub (exact hub slot address or unambiguous hub directory). The operator may move any non-hub slot. A verified TL hub may move its own slots or main's, never a third hub's; ordinary workers cannot move slots. Validate the entire batch before any effect. A runtime move updates one in-memory revision atomically, without restarting agents or writing files.
6. `--persist` computes the smallest positive-claim edit: remove obsolete exact claims for the moved slot from started hubs, retain all repository-wide lines, and add an exact line to the target only if removing the override does not already resolve there. Thus moving back removes the exact line. An explicit main-hub exact claim is permitted when needed to override an extra hub's broad claim; bare Couch still needs no file. Only started hubs' files are read or edited, so an omitted hub cannot be found or rewritten speculatively. Comments/unrelated lines stay intact. Repeating an already satisfied move is a no-op.
7. Switcher row action `move to hub` presents active hubs and an optional `persist` toggle (default off), then uses the same operation as CLI. Show current hub on the row. Successful move redraws sections and fading; failure keeps the previous runtime assignment and shows the cause plus retry. The physical operator may manage every row regardless of focused group. A hub row does not offer move.
8. `--send-to hub`, generated wait/stuck notices and restart resend prompts resolve the slot's current owner. Implicit family sends stay within that hub; exact cross-hub sends remain allowed. Every hub gets the existing hub message allowance. Read-only actors/peek remain global. Remote mutations keep current caller proof/role requirements and additionally enforce same-owner targets; this does not remove worker recovery powers inside their own group. Enumerate switch-agent, relaunch, reload-context, prepare, resume, reboot, reap and #544 set-model. Any hub may notify; multi-hub notification title includes its identity even with a custom title, within existing sanitized bounds. No change to per-owner session-sync.

#### Revised core concepts

| Pure entity | File | Status | Responsibility |
|---|---|---|---|
| HubClaim / parsed claim document | `cmd/internal/couchcore/hub_claim.go` | new | repo/exact specificity, typed normalized references, lossless edit locations |
| HubOwnership | `cmd/internal/couchcore/hub_ownership.go` | new | ordered claims plus runtime overrides; sole resolution and move transition authority |
| HubClaimEdit | `cmd/internal/couchcore/hub_claim.go` | new | minimal source/target exact-line edits; preserve broad claims and comments |
| Hub move reducer | `cmd/internal/couchtty/menu_hub.go` | new | selected target and ephemeral/persist choice, pending/error/result events |
| ThreadPresentation / StatusActor | `cmd/internal/couchtty/thread_presentation.go`, `reserve.go` | modified | grouped rows and derived other-hub fade |

A slot has one resolved hub, but stores no assignment. Claims belong to hubs and may describe many slots; an override belongs to this process only. HubOwnership encapsulates mutation: callers submit events, never write its override map. The pure resolver takes the ordered hub list, parsed documents, canonical slot identity and overrides. Identity resolution is IO and is not hidden inside the resolver. Shared projection eliminates separate guesses in handlers and renderers (ARCH-DRY, ARCH-PURE).

| Integration | File | Status | Seam and verification |
|---|---|---|---|
| HubClaimFiles | `cmd/internal/couchcmd/hub_claim_files.go` | new | bounded `.hub` reads, compare-before-write and durablefile atomic replacement; real temporary directories with injected write failures |
| Multi-hub startup | `cmd/internal/couchcmd/hubs.go`, `cli.go`, narrow `run.go` hooks | new/modified | existing StartInteractive/attach; SlotCatalogFake and controlled launch/terminal handles |
| Shared authority adapter | `cmd/internal/couchcmd/hub_authority.go`, `message_service.go` | new/modified | authenticated workspace identity, current HubOwnership snapshot and revisions |
| Move operation | `cmd/internal/couchcmd/hub_move.go`, `cli.go`; existing operation declarations/dispatcher | new/modified | same typed operation for CLI and switcher, existing queue/receipt semantics |
| Routing/management | `couchmessage/broker.go`, `routing.go`; `couchcmd/slot_operations.go`, `agent_switch_service.go`, `notify_service.go` | modified | current stateful message world and delayed queue fixtures |
| Supervision | `couchcmd/wait_notice.go`, `supervision.go`, `supervision_model.go` | modified | one existing coordinator, per-slot hub evidence and generated destinations |
| Console | `couchtty/console_presentation.go`, `console_menu.go`, `menu.go`, `menu_actions.go`, `menu_render.go`, `console_palette.go` | modified | immutable projection and queued move results, no IO at render time |

Paths in the last three rows are relative to `cmd/internal/`. New production files enter artifactpath's exhaustive inventory. No new external binary/service; reuse the current stateful fakes and real temporary storage (ARCH-MOCK). Final entity/path tables will be reconciled to actual symbols before close.

#### Ordering, failures and bounds

ARCH-ORDER: state is parsed startup claims plus revisioned runtime overrides. Startup claims are read once; explicit persistent moves refresh affected files. No watcher. Moves are serialized with existing operation admission; acquire affected slot gates in canonical order. A queued move revalidates caller identity, source ownership, hub set and revision before effects; a queued mutation revalidates current ownership before acting. Running mutations finish before a move commits. UI focus changing cannot change a queued command's captured caller/target.

Transitions: valid ephemeral move -> atomic new override snapshot; invalid/stale batch -> no changes; duplicate -> no-op; persistence confirmed -> updated claim snapshot plus runtime move; persistence failed/uncertain -> retain runtime ownership, identify saved/unsaved file outcomes and offer retry; restart -> discard overrides and recompute only from started hubs' files. Late completions carry the operation/revision and cannot overwrite a newer projection. Disconnection invalidates remote authority, not claims. Generated notices re-resolve the owner immediately before admission; dispatched messages keep their original recipients/receipts and are never relabeled. An explicit cross-hub message is not evidence of an own-hub reply.

For persistent multi-file edits, use existing durablefile atomic replacement per file, expected-content comparison and one operation serialization owner. Target additions precede source exact removals, so interruption can temporarily leave duplicate exact claims; startup's specified tie rule warns and remains deterministic. Do not claim multi-file atomicity. If any write is failed or unconfirmed, report that persistence is incomplete with named files and safe retry; reread before retrying, never restore old bytes over a human edit. Test failure before/after each file write and process reconstruction from the partial state. No durable transaction log or background retry worker is added.

ARCH-SECURE: CLI and `.hub` text are untrusted input; parse into typed references and resolve catalog identity. Reject symlink/non-regular claim files and root escapes, unknown syntax and invalid UTF-8. Preserve existing incarnation checks before admission and effects. The operator-only UI path cannot be impersonated by a JSON field. No credentials or real-user test stores.

ARCH-CONSTRAINTS: at most current broker MaxActors (128) active hubs/slots; at most 64 KiB and 1024 claim lines per `.hub`, checked on reads and writes. Bounds are design limits, not measured performance claims. Oversize input fails with the file and `shorten this .hub file and retry`; a persist operation that would exceed it refuses without a runtime move. Work is linear in loaded claims and visible slots, cached until explicit state changes, with no filesystem work on keypress/render and serial bounded launch operations. Existing operation timeouts/cancellation apply; no new goroutine per hub.

ARCH-FUNERAL: `.hub` is operator-owned configuration, one bounded file per explicitly used hub, replaced in place only by `--persist`. It lives with that hub directory and is removed with that directory or by its owner. Exact override lines are removed on move-back; runtime overrides die at restart. No per-launch append-only evidence or slot-side residue is created. ARCH-PURPOSE: tests cover every ownership consumer, same-repository splitting, conflicts, future slots, omitted hubs and no-argument compatibility; casts and machine test gate remain singletons.

#### Revised execution tasks (replace all initial checkboxes)

- [ ] **Claims and ownership, RED/GREEN:** create the pure files/tests above. Table/property tests cover exact > broad > fallback, equal-specificity CLI order/warnings, same-repo split, missing hubs, future slots, invalid syntax, canonical duplicates, batch validation, caller eligibility and move-back edits. Run `go test ./cmd/internal/couchcore -run 'TestHub' -count=1`; expected RED before implementation, PASS afterward. Commit.
- [ ] **File IO and move operation, RED/GREEN:** add parser/CLI forms and typed dispatch tests; run against real temp `.hub` files. Cover no-write ephemeral moves, source broad-line preservation, target-main exact override, comments, unchanged-file no-op, batch failure, human edit conflict, every partial-write point, repeat after lost reply and restart after partial writes. Add queue/reassignment interleavings with explicit barriers, no timing races. Run `go test ./cmd/internal/couchcmd -run 'TestHub' -count=1`; focused `-race`; commit.
- [ ] **Launch/authority integration:** wire explicit hub startup and common projection; preserve no-args branch. Add stateful message tests for hub/family/exact routes and every mutating command, stale binding and ownership changes. Add per-hub supervision, notify identity/bounds and restart prompts. Run focused couchcmd/couchmessage tests, including current single-hub regression suites; commit.
- [ ] **Switcher and tab projection:** implement move action through shared dispatcher; grouped rows and modest other-hub fade, preserving attention/blocking, reversed hub chips, glyph widths, overflow and hit spans. Carry ownership through palette reset. Snapshot tests assert byte-equal single-hub output and focus-dependent multi-hub output. Run focused/full couchtty tests and races; commit.
- [ ] **Docs/inventory/integration:** update README, atlas/couch.md, CLI help and `cmd/internal/couchcmd/skills/couch/SKILL.md`; register new Go sources in `cmd/internal/artifactpath/manifest.go`. Merge main and reconcile #528/#544; no recovery-policy edits. Isolated terminal fixture launches two temporary hub directories and private stores, verifies report routing, mutation refusals, move/restart claims, conflict warnings, grouping/fading and shared cast focus; compare default launch calls and frame bytes to baseline. No live operator slots are changed.
- [ ] **Verify and close:** all affected packages, focused races, inventory, clean `make build`, full `make -k test` through the machine gate; inspect all failures independently. Log actual results. `sdlc close --issue 546 --verified 'actual evidence'` owns review, then PR/green CI/land on SHIP and report real hashes. No implementation until this revised plan is approved and change-code admits it.

### 2026-10-14 — approved CLI-first scope cut

Ops:0 approved plan a5c4842799590f5dfc3aaa09d93a55757810a603 (receipt 676cf232-af54-4067-b179-874215cf0fba), with the switcher move action and persist toggle removed. Those are now dependent follow-up pair#548. This overrides the active plan’s item 7, Hub move reducer/menu_hub.go entity, menu action changes and move-UI task: do not implement them in #546. Keep per-row hub display, sections and other-hub fading. CLI move-slot and all routing, authority, supervision, startup and persistence behavior remain in scope. The operator UI origin is future work in #548; #546’s remote CLI requires a verified hub TL. Proceed through full change-code review before implementation.


### 2026-10-14 — plan review PQ-1/PQ-2 dispositions

**PQ-1 (persistence-aware idempotency):** Ephemeral satisfaction means current runtime ownership equals the requested target. Persistent satisfaction means freshly read claim files resolve every requested slot to the target without runtime overrides AND the normalized edit plan has no remaining changes. Therefore an ephemeral move followed by the same `--persist` request MUST plan/write claims. A retry after incomplete persistence MUST reread and finish the durable edit even if runtime already matches. Command receipt dedup applies only to the same operation identity and full mode/target/batch intent; a new persist command is not a duplicate of an earlier ephemeral command. Successful persistent completion updates both claims and runtime only after confirmed/reconciled writes. No-op is always mode-specific, never inferred solely from the runtime target.

**PQ-2 (function-test strategy contract):** The following table replaces the testing case inventories in the prior draft and active task prose; those are historical context, not implementation instructions. Use these function contracts and commands, with executable tests owning individual cases. Existing behavior anchors verified by review: hub grouping `couchtty/thread_presentation.go:180`, section separators `couchtty/reserve.go:583`, current authority `couchcmd/message_service.go:213`, per-file publication uncertainty `durablefile/write.go:63` (all under cmd/internal).

| Function under test | Strategy and mechanical guard |
|---|---|
| `ParseHubClaims(raw)` | Fuzz arbitrary bounded UTF-8/bytes; successful parsing yields only normalized repo/exact references within byte/line bounds, otherwise a line-addressed error. |
| `ResolveHub(orderedHubs, claims, overrides, slot)` | Property-test claim specificity and permutation of unrelated claims; only a stronger claim, equal-rank hub reordering or that slot’s override may change its owner. |
| `PlanHubClaimEdits(documents, slots, target)` | Round-trip generated positive-claim documents through edit/parse/resolve; target ownership must hold without overrides while unrelated/future repo members and non-claim text retain meaning. Applying the plan twice yields no edits. |
| `HubOwnership.Apply(event)` | Model-based event sequences with separate runtime and durable satisfaction predicates; revision checks reject stale transitions and persistent satisfaction cannot be established from overrides. |
| `ApplyHubClaimEdits(ctx, expected, edits)` | Real temporary-file backend with a write-stage barrier/fault seam; interrupt at every publication boundary, reread actual bytes and require truthful per-file outcomes and convergent retry without overwriting changed input. |
| `AuthorizeHubMutation(caller, target, projection)` | Cross-product of verified role and owner relationships; changing another hub’s unrelated facts cannot grant permission. Run through each production mutation handler as a wiring guard. |
| `ResolveRecipient` (modified) | Stateful broker fixture with interleaved ownership/candidate changes; implicit routes never admit foreign-hub candidates, exact routes retain existing global behavior and incarnation proofs. |
| `hubBindingFor(slot)` / `supervisionState.tick` | Deterministic event sequences and delayed connections; only the current own hub can satisfy reply evidence or receive generated escalation, with existing single-hub timelines unchanged. |
| `ParseCLI` / `startExplicitHubs` | Stateful launch fixture with controlled fail/attach results; default invocation preserves baseline call/output bytes, explicit roots validate before launch and partial launch preserves established sessions. |
| `PresentThreads` / `RenderStatusRow` | Generated grouped rows and focus/attention inputs; full terminal-cell validation and hit-span correspondence, plus exact baseline frame equality in single-hub mode. |

Run core tests with `go test ./cmd/internal/couchcore -run 'TestHub' -count=1`; adapters with `go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage -run 'Test(Hub|Message|SlotOperation|AgentSwitch|Notify|ParseCLI|Supervision)' -count=1`; presentation with `go test ./cmd/internal/couchtty -run 'Test(Hub|Status|Console.*Hub)' -count=1`. Then affected full packages, focused races, source inventory, clean build, isolated terminal integration and full `make -k test`. Tests are written RED before implementation. Final function names are reconciled before close.

### 2026-10-14 — implementation seam reconciliation

The single console operation queue already serializes runtime moves with all
agent mutations; no additional per-slot lock owner is introduced. The queue’s
existing bounded receipt map owns move outcomes, including destination and
persistence mode in full-intent deduplication. `move-slot` carries a trusted
owner-local typed payload; no UI move controls or JSON caller-role fields exist.
A retained receipt is answered after caller authentication and before new-effect
source checks, so a successful move does not invalidate its own lost-reply retry.

Final symbol mapping: planned ResolveHub is `HubOwnership.Resolve`;
ApplyHubClaimEdits is `hubClaimFiles.Apply`; AuthorizeHubMutation is
`authorizeHubMutation`. CLI move plumbing is in `hub_cli.go` and `hub_move.go`;
shared queued mutation preparation was extracted to `console_slot_operations.go`.
`hub_guards.go` owns the additional same-owner check. The projection lives in
existing couchtty menu/console/presentation files, without menu_hub.go. Explicit
hub starts use declared `start-hub` -> existing StartInteractive. Integration
clocks are injected through messageAuthority.now with the real clock as default;
no test relies on waiting thirty wall-clock seconds for family quietness.

### 2026-10-14 — integration corrects queue execution assumption

A barrier-controlled startup test demonstrated that the existing queue has
multiple workers: FIFO admission is not serial execution. Replace the preceding
implementation note’s queue-only serialization assumption with a cancellable
hub-runtime effect gate around process mutations and move-slot. Nil/default
startup bypasses the gate. Remote preparations retain a trusted guard with the
original typed payload; after acquiring the gate, repeat the caller/ownership
check before the real effect. This gives one ordering owner without extra
per-slot gates or domain recovery policy changes. Explicit hubs are excluded
from the background reattach pass by the same immutable ownership projection,
preventing two successful start/adopt paths for a detached extra hub.
