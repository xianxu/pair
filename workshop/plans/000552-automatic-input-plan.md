# Automatic input recovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Keep the causality-sensitive gate changes in the main session; bounded transport work may be delegated. Steps use checkbox syntax. One atomic issue, one close review.

**Goal:** Deliver into an empty composer after Codex `/status` and after a previous confirmed delivery, without an operator keystroke or timeout recovery; make any future stuck lease visibly exceptional.

**Architecture:** Correct the existing Codex field parser and consume confirmed empty-composer evidence at delivery completion. Keep the automatic lease under the stdin writer, with an explicit pure transition function for completion, observation and exceptional timeout. A bounded reporter sends exceptional evidence to the supervisor, which owns hub notices and the desktop presenter.

**Tech Stack:** Go, existing terminal emulator and harness fake, Couch Unix-socket request transport, retained diagnostic writer, fake supervisor/host terminal.

## Evidence and alternatives

The preserved parley wrapper's production read-only Observe/Tail endpoint agrees with its 94×39 raw replay: bold prompt, cursor at the input origin, faint suggestion, resolved model footer. `codexComposerActive` is true, but `codexStartupStatus` reports unresolved startup because `/status` prints `Model:` and `Directory:`. Removing only the title row changes Unknown to Empty. A sanitized diagnostic overlay reproduces that failure without live input.

A second overlay uses production dispatch: paste, occupied echo, submit, empty render, Submitted; after the first reservation expires, enqueue another ID without output. Publish has set `fence == epoch == 3`, so the second delivery waits indefinitely. No fallback exists in either reproduction.

Chosen: fix the label comparison and carry the confirmation's own empty evidence into lease completion. Moving every fence to a write boundary would widen concurrency assumptions; unconditional release on any Submitted outcome would weaken turn-only/uncertain-write safety. Timeout-only recovery would conceal both causes. The targeted completion fix preserves those distinctions.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `automaticInputTransaction` and its transition decision | `cmd/internal/wrapcmd/automatic_input.go` | modified |
| `AutomaticInputStall` report and validation | `cmd/internal/couchmessage/automatic_input_stall.go` | new |

The transaction belongs 1:1 to a proxy's stdin writer. Its transition function takes explicit time, completion evidence and a coherent observation; only that function changes its authoritative phase/fence/stall episode. It returns availability and an optional report/release effect. No terminal, clock or socket IO in the transition. Tests directly enumerate event sequences. This is the sole lease model (ARCH-PURE, ARCH-ORDER).

The report is one bounded, typed occurrence, not a message queue or new delivery receipt. It carries episode ID, cause, elapsed wait, fence/current epoch and whether the stale lease was released. Desktop text is composed by the supervisor from verified slot identity and these enums; it accepts no arbitrary notification title/body from a worker. Screen content remains local diagnostic evidence. No new credential.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Codex startup/status field parsing | `cmd/internal/wrapcmd/composer_rows.go` | modified | existing terminal snapshot; bugfix, no new parser authority |
| Delivery completion adapter | `cmd/internal/wrapcmd/peer_delivery.go` | modified | existing confirmation and single stdin owner |
| Tripwire reporter | `cmd/internal/wrapcmd/automatic_input_report.go`, `peer_runtime.go` | new/modified | retained diagnostic writer and existing broker RPC |
| Supervisor stall handler | `cmd/internal/couchcmd/automatic_input_stall.go`, `message_service.go` | new/modified | live caller validation, generated hub notice, desktop callback |

Reuse terminal snapshots, `traceWrap`, strict JSON, `SocketPath`/`Call`, existing authority checks, supervision `sendGenerated`, and the desktop callback used by `couch --notify` (ARCH-DRY). Worker CLI notification remains hub-only. The new internal operation does not grant general worker notification authority or introduce a CLI command.

Integration tests use the existing stateful harness fake, `serviceRig`/message world, private Unix sockets and `hostty.NewFakeHost`. They run with portable temporary storage, never the real Couch store (ARCH-MOCK). The original preserved raw recording is a separate local conformance check, not a committed fixture containing account information.

## Lease transitions and invariants

Use the existing phase enum, adding an alerted-wait phase; keep completion provenance explicit (confirmed-empty, submitted-by-turn-only, uncertain). The adapter passes the actual `cleared` evidence from the same coherent snapshot used to confirm delivery, rather than re-reading it later.

| State/event | Next state and effects |
|-------------|------------------------|
| Idle / begin peer write | Peer; retain exclusive automatic owner |
| Peer / delivery confirmed by post-submit empty render | Idle; that confirmation satisfies the lease, no new fence |
| Peer / turn-only confirmation or uncertain terminal outcome | AwaitingEmpty; retain existing fence and outcome provenance |
| AwaitingEmpty or Alerted / complete post-fence empty observation | Idle; normal release |
| AwaitingEmpty / first eligible blocked attempt | AwaitingEmpty; start a fixed 30-second stall clock |
| AwaitingEmpty / still eligible at 30 seconds | Report once; retire the lease only under the safe-release rule below, otherwise Alerted |
| Alerted / repeat observations | No repeated report; normal release remains available when fresh empty evidence arrives |
| Any / wrapper shutdown | Runtime owns cancellation and joins reporter; no durable pending report |

The 30 seconds measures actual blocked delivery attempts, not time since the previous quiet delivery. Eligibility requires an idle turn, coherent complete frame, no human-input/paste/image/forwarded-input hold, and an empty or unrecognized composer. Known occupied drafts and pickers do not start/continue this clock; normal human holds are not tripwire events.

**Safe exceptional release:** prior delivery has a confirmed Submitted outcome, the current complete snapshot positively reads Empty, no turn is active and all independent input guards are clear. Only the obsolete epoch fence is bypassed. Unknown/occupied composer, partial or uncertain write, incomplete frame, pending/held human input or picker cannot be converted into permission by elapsed time. Such an eligible unknown-state stall alerts once and keeps refusing. Every subsequent write still passes ordinary admission. This qualification is deliberate: the requested tripwire must not authorize overwriting a draft.

The episode budget is never renewed by output/polls; ineligible genuine human activity ends the eligible stall interval. A new automatic transaction creates a new episode. Test clock movement explicitly. Keep production recovery duration injected/defaulted so root-cause tests can disable recovery entirely; do not add an environment bypass.

## Operator projection

| Internal condition | Receipt / notice | One action |
|--------------------|------------------|------------|
| Confirmed empty; lease released normally | Delivery continues; no alert | None |
| Awaiting a post-completion render | `waiting for previous automatic input: no post-completion render` | Wait within the bound |
| Unknown composer prevents release | `waiting for previous automatic input: composer unrecognized` | Inspect the named slot |
| Genuine occupied composer or picker | Existing human hold with its actual cause | Finish or clear that draft/picker normally |
| Safe tripwire release | Desktop alert + hub notice: unexpected automatic-input stall; lease released; diagnostic reference | Hub investigates the regression |
| Unsafe tripwire; no release | Desktop alert + hub notice: automatic-input stall; input still blocked; cause and diagnostic reference | Hub inspects that slot before recovery |

Relaunch's ordinary idle observation must read the corrected shared Codex parser, so `/status` no longer produces the false occupied/unknown refusal. Do not reset any live wrapper as a workaround.

## Reporting, boundaries and budgets

This is an interactive input path on the shared macOS machine. Normal parsing and lease transitions use existing snapshots and constant transaction state; no extra processes, timer goroutine per delivery, network or disk work on the ordinary key path. Existing queued-delivery polling provides at-most-one-second tripwire observation granularity (assumption based on current poll interval). Threshold 30 seconds is a proposed operator choice, below the observed 90-second delivery expiry.

On a firing, copy bounded evidence (dimensions/cursor/frame status/phase, elapsed time, epoch/fence, cause, styled tail up to 40 rows and 16 KiB with explicit truncation) and enqueue one report to the runtime-owned worker. The worker writes one local `automatic-input-stall` diagnostic through the existing capped writer, then makes one RPC with a two-second deadline. It logs report failures and never retries uncertain writes. No disk/network IO while holding input-admission or delivery locks. One worker, one-slot in-memory handoff; each episode produces at most one report. Overflow/cancellation is recorded locally, never interpreted as an admission decision.

Internal broker op `automatic-input-stall` has a closed request shape: current Scope/Tag/Session/Nonce, episode ID, and a size-bounded typed report in Body; reject other fields. Validate the current connected incarnation immediately before admission. Parse malformed/unknown enum/oversized reports as refusals. An older supervisor refuses the new op; the wrapper records that failure without retry or delivery protocol changes. Ordinary old/new send, actors, registration and endpoints retain their existing wire shapes (ARCH-SECURE).

The supervisor independently emits: (1) a persistent-in-console Couch notice, (2) a generated message to the current owning hub via ordinary in-memory delivery policy, and (3) one desktop write through the existing serialized presenter. Use the same occurrence ID to deduplicate, scoped to the exact connected binding. If the origin is the hub, avoid manufacturing a peer self-send; its console notice plus desktop alert name that hub and the retained diagnostic. A failed hub admission does not prevent the desktop attempt. A failed/uncertain desktop write is recorded and not automatically retried.

Duplicate occurrence bookkeeping is at most one retained occurrence per connected binding and removed on disconnect; it dies at Couch restart. Runtime cancellation joins the reporter before closing its diagnostic owner; no detached goroutine or offline/restart queue. Diagnostic entries join existing size-capped rotation; screens have the explicit per-event cap above (ARCH-FUNERAL, ARCH-CONSTRAINTS). Secrets are neither minted nor needed; account-bearing production screen output is never copied into repository fixtures.

## Implementation steps

### 1. Make both root-cause regressions pass with recovery disabled

Files: `composer_rows.go`, `peer_composer_test.go`, `automatic_input.go`, `automatic_input_test.go`, `peer_delivery.go`, `peer_delivery_test.go`, `peer_integration_test.go` under `cmd/internal/wrapcmd/`.

- [x] Transfer the sanitized `/status`/model-changed paint reproduction from the scratch overlay. Assert empty, occupied draft, lowercase and capitalized labels, unresolved/loading model/directory, hidden cursor and picker variants. Add a harness delivery/relaunch-idle observation case using the same snapshot parser.
- [x] Add consecutive real dispatch tests: paste/occupied echo/submit/empty render confirms the first; release/expire its reservation normally, enqueue a distinct second ID, dispatch without any output or human input. Include a running-turn confirmation case and a case where notification output follows confirmation. Recovery is disabled; assert immediate second paste and zero tripwire reports.
- [x] Run `go test ./cmd/internal/wrapcmd -run 'Test.*(StatusCard|ConfirmedEmpty|AutomaticInput|PeerComposer)' -count=1`; record the two expected baseline failures before edits.
- [x] Parse only field labels case-insensitively; retain existing resolved-value, startup and composer safety checks. Do not lowercase or reinterpret operator draft content.
- [x] Pass post-submit empty confirmation evidence into the transaction completion transition. A confirmed empty delivery returns Idle; other outcomes retain the existing fresh-empty requirement. Keep uncertain/partial writes, turn-only confirmation with no empty render, occupied composer and incomplete synchronized frames blocked.
- [x] Run targeted tests and `go test ./cmd/internal/wrapcmd -count=1`. Commit the root fixes and evidence before implementing recovery.

### 2. Add and prove the exceptional tripwire

Files: transaction model/tests above; new `cmd/internal/couchmessage/automatic_input_stall.go` and tests; `protocol.go`/tests; new `cmd/internal/wrapcmd/automatic_input_report.go` and tests; `peer_runtime.go`; new `cmd/internal/couchcmd/automatic_input_stall.go` and tests; `message_service.go`; relevant artifactpath source inventory.

- [x] Write table/sequence tests for boundary-before/at/after 30 seconds, no deadline renewal, eligible/ineligible transitions, safe release, unknown/occupied/uncertain refusal, one firing, eventual normal recovery, cancellation and new episodes. Assert root regressions still pass with recovery disabled.
- [x] Implement the pure transition and detailed cause projection, then run those tests.
- [x] Add strict report/request validation tests (foreign fields, size, identity, enums) before adding the internal broker operation.
- [x] Build the bounded runtime reporter and supervisor adapter through the existing diagnostic, transport, notice and presenter seams. Use no subprocess for notification and do not loosen worker `notify` admission.
- [x] Exercise the real private-socket path with the existing stateful supervisor rig and fake host: one firing yields one diagnostic, one owning-hub message, one console notice and exactly one OSC notification; duplicate calls do not repeat it. Include hub-origin handling, stale/replaced/disconnected caller, absent hub, old supervisor refusal, closed console, uncertain write, hanging RPC deadline, shutdown while reporting and multiple owned hubs. The input writer must remain usable while report IO is blocked.
- [x] Verify new production files join the exhaustive artifact inventory: `go test ./cmd/internal/artifactpath ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/wrapcmd -count=1`.
- [x] Commit the tripwire and test evidence.

### 3. Verify and close the atomic item

- [ ] Re-run both root regressions disabled and enabled; neither normal trace produces a tripwire. Check occupied drafts/menus and unknown/loading screens fail closed.
- [ ] Replay the preserved parley raw bytes locally at its recorded dimensions against the fixed parser and fixed idle/delivery observations. Compare the still-running wrapper read-only if needed; do not type, restart, replace or reattach it. Report the verified result to TL before any live rollout.
- [ ] Update `atlas/couch.md` (and index summary if needed) with completion-evidence release, detailed status and tripwire ownership/bounds. Record accepted review lessons.
- [ ] Run the complete `make -k test` through the machine gate; inspect all failures and cite known flakes, never stop early. Run targeted race tests for reporter/runtime/supervisor ordering where supported.
- [ ] Record actual hashes from `git rev-parse HEAD`, full-suite and headless evidence. No stage/visible window needed unless TL requests it.
- [ ] After TL acceptance, `sdlc close --issue 552 --verified ...`; address mandatory fresh review findings. Merge latest main, create PR, and on SHIP run `sdlc merge --yes` with validation. Report landed PR/hash to hub and stop.

## Architecture decisions

- ARCH-DRY: parser and completion evidence remain shared; notification reuses the supervisor presenter and normal hub delivery, not OSC emitted by the child.
- ARCH-PURE: transaction decisions and report validation are pure; runtime and supervisor adapters own IO.
- ARCH-PURPOSE: both observed root causes and all three tripwire outputs are in this issue; fallback cannot make a root regression pass.
- ARCH-MOCK: existing stateful terminal/supervisor/fake-host seams, private stores, plus read-only local conformance replay.
- ARCH-CONSTRAINTS: 30-second eligible blocked interval, existing poll granularity, one reporter, two-second RPC, capped evidence, no normal-path IO.
- ARCH-SECURE: exact current incarnation, closed report schema, supervisor-composed notification, no arbitrary worker notify or sensitive committed capture.
- ARCH-ORDER: single input owner, explicit completion evidence, pure transitions, no re-read race, no uncertain-write retry; teardown joins report work.
- ARCH-FUNERAL: one occurrence per live binding, disconnect cleanup, capped existing diagnostic retention, no durable pending reports or sidecars.

## Revisions

### 2026-10-14 — plan-quality PQ-1/PQ-2

This revision supersedes the prose test inventories and arbitrary-ID deduplication description above, while retaining both executable root reproductions as acceptance evidence. TL approved the root fixes, 30-second eligible interval and strict safe-release qualification in receipt 323226c7-e429-4a71-97cb-d259e135af11.

**Function-level test strategy (PQ-1).** Use these names in implementation; the rows replace the case lists in Tasks 1–2:

| Risky function | Adversarial input / mechanical guard |
|----------------|--------------------------------------|
| `codexStartupStatus`, `peerComposerState`, `settledNow` | Captured status/startup paints and styled drafts → sanitized terminal replay plus delivery/idle observations, preserving loading/unknown refusal |
| `advanceAutomaticInput` | Completion/observation/time event reorderings → pure sequence tests with a fake clock, asserting no unsafe write authority and at most one report |
| `dispatchPeer`, `advancePeer` | Consecutive real deliveries with no extra output → production harness dispatch; assert second paste with recovery disabled, while uncertain writes stay fenced |
| `ParseAutomaticInputStall`, `ValidateRequest` | Arbitrary/malformed/oversized JSON and foreign request fields → strict typed decoding, bounded schema, fuzz seeds from malformed forms |
| `runAutomaticInputReporter` | Delayed IO and cancellation at each boundary → private socket barriers and real capped diagnostic file, proving input remains usable, one attempt and joined teardown |
| `handleAutomaticInputStall` | Stale callers and reordered/repeated occurrences → stateful service rig, fake host and controlled dispatch barriers, asserting exact notification counts and authority |

**Occurrence ordering (PQ-2).** Each wrapper assigns a monotonically increasing uint64 `Sequence` only when a tripwire fires; the report carries it and the request ID is derived from binding PID plus sequence. Never wrap: exhaustion refuses reporting and records a local diagnostic. Supervisor admission is atomic under its stall-record mutex: retain the exact current binding plus high-water sequence, reject `Sequence <= highWater` with no repeated effects, and reserve a strictly newer sequence before IO. Thus receiving 2 then 1, or duplicating 1 after 2, cannot notify again. Outcomes remain attempted/uncertain, not retryable success.

Keep one small record per slot, with the exact binding inside it. A same-binding reconnect preserves the high-water mark; a verified replacement incarnation supersedes it, and old callers fail current-identity checks. Disconnect alone does not erase duplicate evidence. Bound the map to `couchmessage.MaxActors` (128) slots; do not evict suppression evidence to admit a new slot. At capacity refuse the new report explicitly, retaining its wrapper diagnostic. It dies with the supervisor; this creates no durable/restart queue. This paragraph supersedes immediate disconnect cleanup above (ARCH-ORDER, ARCH-FUNERAL). No broad broker or session protocol change is needed.

### 2026-10-14 — implemented integration details

Both root reproductions passed with recovery absent, then explicitly disabled after the tripwire was added. Empty-confirmation evidence travels on the existing in-process PeerDeliveryEvent, without changing endpoint or registration wire shapes. Terminal snapshots now carry completeness captured under the model mutex, preventing a partial synchronized frame from satisfying completion/recovery. The normal idle/owned lease path takes no additional snapshot.

Occurrence IDs now hash the full binding, not just PID, plus the monotonic sequence. A PID-reuse test demonstrated the old collision and proves both replacement-incarnation hub messages are admitted. This is the exact-incarnation rule applied consistently to message IDs as well as high-water records.

The reporter captures a bounded styled screen without IO, owns one worker and a one-slot handoff, records overflow counts locally, and drains accepted diagnostic evidence without network effects on cancellation. Runtime Close is idempotent and joins reporting before the existing wrap-diagnostic teardown; there is no detached reporter. Targeted race tests cover the actual handoff/transport and supervisor adapters.

### 2026-10-14 — TL rollout decision

TL receipt 1625cf7b-db6e-4d6d-8442-220ff0eacc93 accepts the pre-landing read-only replay and disabled-recovery regressions. After the complete suite and close review, land normally. Live acceptance is explicitly post-landing: TL rebuilds main after #528 releases pair:0, relaunches parley.nvim:1 and verifies one normal delivery. This slot performs no stage and preserves the original wrapper. This supersedes any implication that landing waits for a pre-landing live window.
