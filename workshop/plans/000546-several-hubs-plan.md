# Several Hubs in One Couch Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to choose subagent-driven-development or executing-plans. Track the steps below; SDLC owns review boundaries.

**Goal:** Run several explicitly selected hubs in one Couch, with durable per-slot ownership, isolated automatic routing and management, and visible hub grouping.

**Architecture:** One stable slot-to-hub fact feeds pure routing/authorization and presentation. Reuse the current console, operation queue, authenticated caller checks, journal writer and launch/recovery operations; do not fork a supervisor per hub. The no-argument launch path remains the existing path.

**Tech Stack:** Go, existing Couch message broker and terminal renderer, portable JSON persistence through the existing store transaction seam.

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
