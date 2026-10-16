# Agent keys and model selection implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the connected driver work; bounded pure parsing or fixture work may be delegated. Steps use checkboxes for tracking.

**Goal:** Let a verified owning hub drive an idle slot’s agent keys and choose its model/effort with honest, screen-confirmed results and session-only scope by default.

**Architecture:** Add one exact-incarnation wrapper drive capability, executed by the existing single stdin writer. A pure bounded recipe reducer chooses named inputs from fresh styled agent screens. Couch owns authorization/receipts; the wrapper owns input exclusion, parsing and lifetime. Reuse the existing terminal translator, tail rendering, console queue and ownership gate.

**Tech stack:** Go, existing Unix-socket transport and PTY terminal model, recorded Codex/Claude screens, stateful harness fakes and isolated live conformance.

## Approved requirements and design decisions

The 10-14 Revision supersedes a recipe that selects by assumed position: model and effort labels must be observed, and Codex’s session-only `s` action must be explicit. `--scope session|default` defaults to session; default scope requires the existing `--confirm` and a supported recipe. No config file is edited by Couch. Claude session selection must be verified in its captured recipe; default scope is refused unless the live recipe proves that capability.

CLI:

```sh
couch --keys pair:7 clear-line text:/model submit wait:200 down enter
couch --set-model pair:7 --model gpt-6.1-sol --effort medium --confirm
couch --set-model pair:7 --model gpt-6.1-sol --effort medium --scope default --confirm
```

`--keys` accepts `text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line`, `wait:MS`. Raw keys have no peer-message chrome and return the final styled tail, with JSON available. `enter` traverses Pair’s normal Return translation: newline in the qualified composer, selection inside a picker; `submit` traverses Alt+Return. A lone ESC is a completed separate step, flushed before a subsequent Return. Raw key success means bytes and final observation confirmed, not that an arbitrary command achieved its goal.

Both operations require a verified current hub caller, a different exact connected slot owned by that hub (#546), positive wrapper Settled and a fresh qualified empty agent composer. Unknown/busy/picker/buffered-paste states refuse with “wait until the agent composer is empty and its turn finishes.” TL clarified that the separate Neovim draft is unaffected and needs no guard. No dirty-Git precondition: these operations do not replace a process or checkout.

Choose the wrapper endpoint over Zellij CLI writes: it already pins Binding plus session incarnation and owns the only PTY writer. Zellij would duplicate pane/session discovery and bypass Pair’s key translation. Peer message delivery is also unsuitable: it adds chrome and intentionally accepts messages during active turns. No fallback to an older wrapper’s arbitrary terminal injection; report “relaunch the slot on the new build.”

## Core concepts

### Pure entities

| Name | Kind | Lives in | Status |
|---|---|---|---|
| `KeySequence`, `ParseKeySequence` | PURE | `cmd/internal/agentdrive/keys.go` | new |
| `ModelRequest`, `ModelScope`, `Recipe` | PURE | `cmd/internal/agentdrive/recipe.go` | new |
| `ScreenEvidence`, `ReadModelScreen` | PURE | `cmd/internal/agentdrive/screen.go` | new |
| `DriveState`, `DriveEvent`, `ReduceDrive` | PURE | `cmd/internal/agentdrive/model.go` | new |
| `OperationReceipt`, `ApplyReceiptEvent` | PURE | `cmd/internal/couchmessage/operation.go` | modified |
| `classifyWait` / model-control confirmation recognition | PURE | `cmd/internal/couchcmd/wait_notice.go` | modified |

`agentdrive` imports neither Couch nor wrapper, avoiding the existing couchmessage→couchcore dependency cycle. One drive owns one parsed request and a bounded observation history. Recipes are one table per agent, selecting parser/state phases rather than hardcoded row numbers. Request equality uses a bounded canonical intent digest in the existing receipt machine; request IDs cannot change text, model, effort or scope. Preserve existing receipt statuses and expiry.

`ScreenEvidence` decodes the existing ‹rev›/‹dim›/‹cursor› grammar, including escaped ‹‹. Match complete normalized model identifiers or their recipe-declared exact display labels, never substring guesses. Require unique selected-row evidence and a recognized picker phase. Navigation uses fresh selected labels and screen generations; reorder/scroll is accepted when recognized, duplicate/ambiguous labels decline. Final confirmation must be a newly painted live status region after the apply key, not old scrollback or a footer that was already present before the action. If effort is omitted, use the picker’s observed selection and verify that observed value.

### Integration points

| Name | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| `handleAgentDrive`, `installAgentDriveService` | INTEGRATION | `cmd/internal/couchcmd/agent_drive_service.go` | new | authenticated console queue and wrapper capability |
| `RemoteEndpoint.Drive`, status/cancel RPC | INTEGRATION | `cmd/internal/couchmessage/drive_endpoint.go` | new | existing bounded Unix socket calls |
| endpoint request/response and drive intent | INTEGRATION | `cmd/internal/couchmessage/endpoint.go`, `protocol.go` | modified | closed typed wire parsing |
| `dispatchAgentDrive` | INTEGRATION | `cmd/internal/wrapcmd/agent_drive.go` | new | existing stdin writer, input admission and terminal snapshots |
| wrapper endpoint admission | INTEGRATION | `cmd/internal/wrapcmd/peer_runtime.go` | modified | exact Binding proof and drive mailbox |
| automatic input owner / command accounting | INTEGRATION | `cmd/internal/wrapcmd/automatic_input.go`, `wrap.go` | modified | shared translator and lifecycle observations |
| CLI adapter | INTEGRATION | `cmd/internal/couchcmd/agent_drive_cli.go`, `agent_drive_messages.go` | new | argv, admission and read-only receipt polling |
| stateful drive harness | INTEGRATION | `cmd/internal/wrapcmd/agent_drive_test.go` | new | existing harnessSessionFake / peerLoopFixture |
| isolated model conformance | INTEGRATION | `cmd/internal/wrapcmd/peer_live_test.go` | modified | fresh real-agent PTYs in disposable repos |

The new wrapper adapter admits one complete bounded sequence/model request and wakes the input loop; RPC goroutines never write the PTY. It uses the same translation logic as ordinary input, with a typed remote origin so recognized model-control submissions do not open a conversational turn. Generic raw `submit` retains ordinary turn accounting. All direct translated writes remain under input admission and the automatic-input lease.

## Operating envelope and lifecycle

Bounds are initial design choices, tested before landing: at most 64 caller-supplied steps, 8 KiB aggregate printable UTF-8 text, 0–1000 ms per explicit wait and 5 seconds aggregate waits. Reject control characters in `text:`; named keys provide them. At most 96 recipe navigation steps, 128 observed labels (128 bytes each), and 200 screen lines; retained result tail capped at 32 KiB. Invalid input names the limit and asks the caller to shorten the sequence.

One absolute 30-second server budget begins at operation admission, including queue wait; reserve its final 3 seconds for safe recipe back-out. CLI observes for at most 36 seconds and existing short RPC deadlines stay unchanged. Polling is read-only and uses fresh screen epochs, not arbitrary sleep assumptions. Timeout returns the last confirmed step, latest bounded tail, and whether prompt recovery was observed. No input effect is automatically retried after transport loss or timeout.

Use the existing canonical per-slot console queue key, so keys/model selection conflict with restart/switch of the same slot. Declare `keys` and `set-model` as process/input effects so #546’s effect gate also orders them against moves; repeat authority and session-token checks inside the gate. The 30-second deadline applies while waiting for that gate. Any owner change queued before the drive is revalidated; owner changes cannot interleave with its effects.

Wrapper has one active drive, sharing automatic input ownership with peer delivery. Competing automatic work gets a named busy refusal; it does not allocate a second scheduler. Existing max 64 operation receipts / five-minute expiry bound result retention (~2 MiB with 32 KiB tails). Wrapper retains at most eight bounded completed drive receipts for five minutes, and removes them on shutdown; the active drive dies with its wrapper/session or absolute deadline. No new durable logs/cache/secret or per-thread model field. Versioned test captures are bounded checked-in fixtures, excluding credentials and identifying paths.

## State and ordering contract (ARCH-ORDER)

`ReduceDrive` owns legal drive transitions; IO performs only its declared write/observe/wait/finish effects. It stores the request, current phase, last observed epoch/selection and confirmed progress, not independent completion booleans.

| State / event | Transition and effects |
|---|---|
| absent + validated admission | reserve drive; queued; duplicate exact ID returns held result, changed intent refuses |
| queued + known-idle/identity proof | running raw sequence or model-command phase; acquire input lease |
| queued + busy/unknown/stale/deadline | refuse without a byte; release reservation |
| running + fresh expected screen | choose one observed-label move or select; wait for a newer epoch |
| running + repeated/stale/incomplete paint | observe within remaining deadline; bounded no-progress refusal |
| model/effort + recognized selection | select only the observed requested label; enter scope phase |
| scope + session recipe evidence | Codex `s`; default scope uses recipe’s explicitly verified default action |
| apply + fresh matching live status | confirmed success with model, effort, scope and bounded tail |
| running + write failure/lost acknowledgement | unconfirmed outcome; retain last confirmed progress; never replay input |
| model drive + unknown label/phase/timeout | safe back-out only from an owned recognized picker or owned `/model` composer; verify prompt; report visible choices |
| any active + operator input | cancel drive before admitting that input; release lease; no later automatic keys or cleanup overwrite the operator |
| any active + active turn/replacement/disconnect | stop input; cancel/drop pending drive; report uncertain if an effect may have occurred |
| terminal + late screen/callback | ignore by drive ID/generation; result remains bounded and immutable |

Broker sends one wrapper drive admission, then polls its result. Lost response never resends admission; operation-status is read-only. Caller disconnect cancels the broker-owned drive context and sends an idempotent cancel by ID; wrapper self-expires even if cancellation acknowledgement is lost. Shutdown joins its owned work. Before every key, verify exact wrapper/session, no human input superseded the lease, and no conversational turn is open; picker-induced unsettled status is expected only inside the recognized owned recipe.

Back-out is limited to at most two separate ESC steps plus clear-line only for text positively known to be the driver’s own `/model`. Never send cleanup into an unknown screen, replacement wrapper or a new operator draft. Success/failure must not claim “back at prompt” without fresh empty-composer evidence. If unreachable or an apply outcome is unconfirmed, show “outcome unconfirmed; inspect with couch --peek repo:N before sending anything else”, with the observed tail. Raw keys leave exactly the caller’s sequence result and have no invented model cleanup.

## False waiting notice correction

A recognized model-control drive is a UI command, not a user conversation. Suppress `ObservationUserSubmission` / turn-opening Return observations only while that drive owns the recognized command/picker phase. Do not suppress generic raw submissions or unrelated slash commands. On observed prompt recovery or confirmed success, ordinary settle evaluation resumes from fresh output.

Add pure screen recognition for Codex’s exact model confirmation shape (“Model changed to … for this session only”, plus captured default confirmation) when judging prompt waits. A completed command confirmation alone must not become waitPrompt or waitQuestion. Preserve genuine picker, permission/question and off-resting idle notices; do not blanket-mute the slot or use a ten-minute suppression timer. Test local and remote confirmation screens with Settled both true/false, and confirmations followed by an actual question/dialog. The reported false notices are a design regression contract, not a broad change to idle escalation.

## Tasks (one atomic close boundary)

### 1. Capture recipe evidence and pure driver — RED/GREEN

Files: create `agentdrive/{keys,recipe,screen,model}.go` and colocated tests; versioned raw fixtures under `cmd/internal/wrapcmd/testdata/drive/{codex,claude}/<version>/`.

- [ ] After approval, capture `/model` → model → effort/scope → status and ESC recovery in fresh isolated Codex and Claude PTYs, using existing live harness. Record versions and normalized fields in fixture metadata. Verify session changes leave shared defaults unchanged; default-scope conformance uses a disposable config. No operator slot is driven. If the actual Claude/Codex shapes contradict the chosen phases/scope, stop and append a plan Revision before building.
- [ ] Write parser/reducer tests first: named key bytes, separate ESC boundary, text/wait/step bounds; unique labels, reordered rows, scroll, incomplete paints, stale epochs, ambiguous choices, omitted effort, session/default scope, final-status mismatch and state/event interruption sequences.
- [ ] Run `go test ./cmd/internal/agentdrive -count=1` and observe RED; implement minimal pure types/parsers/reducer; rerun PASS. Commit.

### 2. Wrapper capability and lifecycle — RED/GREEN

Files: new `wrapcmd/agent_drive.go`, `couchmessage/drive_endpoint.go`; modify `endpoint.go`, `peer_runtime.go`, `automatic_input.go`, translator/lifecycle seams in `wrap.go`; add stateful tests.

- [ ] Drive the real endpoint, stdin loop, translator and terminal model through a stateful fake application whose received inputs paint recipe screens. Tests first cover raw keys without chrome, correct Return/submit, separately flushed ESC, peer exclusion and model phase observations.
- [ ] Add controlled interleavings for operator input, queue competition, active turn, replacement/session reconnect, partial write, lost response, deadline and late callback. Assert no automatic replay and no cleanup over operator text. Result/status identity and retention tests cover duplicates/conflicts/expiry.
- [ ] Run `go test ./cmd/internal/wrapcmd ./cmd/internal/couchmessage -run 'Test(AgentDrive|ModelDrive|DriveEndpoint)' -count=1`; RED then implement and rerun GREEN plus targeted `-race`. Commit.

### 3. Couch CLI, authority, receipts and wait correction — RED/GREEN

Files: new couchcmd drive service/CLI/message adapters; modify `cli.go`, `run.go`, `message_service.go`, `slot_operations.go`, `messages.go`, `couchmessage/{protocol,operation}.go`, `couchcore/{ops,operationdispatch}.go`, `wait_notice.go` and tests.

- [ ] Pin CLI shape, no-args compatibility, scope default, --confirm, unknown/extra fields and all bounds. Wire declared operations and inventory audits; receipt intent includes canonical key/model/effort/scope digest and bounded drive result.
- [ ] Stateful socket tests prove verified owning hub role, exact slot resolution, self/cross-hub refusal, idle/unknown refusal, queued owner/session changes, effect-gate rechecks, caller disconnect and status-only lost-reply recovery. Refusals name one useful next action. No process switching or Git writes.
- [ ] Add false-notice regressions and positive question/picker/idle controls; route command observations through the existing lifecycle instead of a second wait controller.
- [ ] Run `go test ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchmessage -run 'Test(AgentDrive|ModelDrive|Drive|Wait|Hub)' -count=1`; RED/GREEN and focused `-race`. Commit.

### 4. Documentation, conformance and acceptance

- [ ] Update README, `atlas/couch.md`, CLI help/cost map and shipped `cmd/internal/couchcmd/skills/couch/SKILL.md`; register new production files in `artifactpath/manifest.go`. Give the hub the verified recipe for its TL skill (TL owns that separate documentation; obtain the exact path/acknowledgement rather than editing another repo silently).
- [ ] Extend opt-in isolated `TestPeerLiveConformance` with the two agents’ model scenarios and raw named keys, final status and session-scope/default isolation evidence. Use fresh PTYs, repositories and config locations. Keep credentials out of captures. Expose this through existing conformance targets without changing protected CI/gate paths in this issue.
- [ ] Run full affected packages, focused races, inventory, clean `make build`, then `make -k test` through the native machine gate. Run remaining targets even if a timing check fails; record every failure independently. Reconcile current main (including #549 after its protected-path merge) and affected tests. No custom load wait.
- [ ] Commit measured evidence and completed active checklist. `sdlc close --issue 544 --verified 'actual evidence'` owns the fresh review; fix findings. Open PR, require green CI, land only on SHIP, and report the PR plus real merge hash to the hub.

## Architectural traceability

ARCH-DRY: one key/recipe driver and shared identity/translator/receipts, not Zellij scraping or a parallel messenger. ARCH-PURE: typed parsing, screen evidence and reducer are IO-free. ARCH-PURPOSE: raw keys, both required agents, effort/scope confirmation and false wait correction are acceptance work. ARCH-MOCK: the fake paints screens after actual translated inputs; versioned/live conformance pins the external contract. ARCH-CONSTRAINTS: fixed input/result/deadline/navigation/concurrency bounds. ARCH-SECURE: strict per-operation wire shapes, trusted broker role, exact incarnation, no control characters hidden in text, no arbitrary RPC command or config edits. ARCH-ORDER: reducer plus single writer/lease, explicit cancellation and confirmed/unconfirmed effects. ARCH-FUNERAL: bounded in-memory receipts expire and all drive work dies with its owner; no durable runtime state.

## Revisions

- 2026-10-14 — first plan incorporates the issue’s 10-14 verified recipe, session-only default, false wait regression and TL clarification that only the agent composer is guarded. This is a plan for approval; no implementation has started and no estimate has been derived.

### 2026-10-14 — plan-quality round 1 refinements (PQ-1/PQ-2/PQ-3)

**PQ-1 / active operation lifetime.** “Caller disconnect” above means loss or replacement of the caller’s registered wrapper SessionToken, not closure of an admission/status RPC or exit of the short-lived CLI. `couchmessage.Call` normally closes each exchange (`transport.go:235`); handlers normally cancel on return (`transport.go:175`). The operation is owned by the console lifetime (`console_remote.go:62`) and an absolute admission deadline, with a service-owned monitor checking the pinned caller Binding/SessionToken. Admission/status handler contexts never own the drive. A normal RPC close has no effect; actual registry disconnect/reconnect, authority loss, console shutdown or deadline cancels it. A lost CLI response leaves the operation bounded and readable via operation-status. The wrapper mailbox similarly owns its job under peerRuntime’s lifetime (`peer_runtime.go:155`) and the supplied absolute deadline, independently of endpoint RPC contexts. The console waits/joins its monitor before finishing; wrapper shutdown cancels/joins the writer’s job. Cancellation by ID is idempotent; it does not replay admission. This is the operative meaning of lifetime and cancellation throughout the plan.

**PQ-2 / isolated model authentication.** Model scenarios must not invoke `peerLiveLocalAuth`’s Claude native-HOME fallback (`peer_live_test.go:420`). Add `peerLiveIsolatedModelAuth` in the same live-test file: retain every HOME/config/cache/data path beneath the disposable root; copy only existing Codex auth.json or Claude .credentials.json into the disposable corresponding directory at mode 0600, or use an already-supplied auth environment variable. Local probes establish that both source login files exist; no secret values have been printed. Use Claude safe mode with disabled tools/MCP/custom settings; authentication reads the private copy, never restores native HOME/config. Missing/unsupported copied authentication is an explicit conformance blocker, not permission to fall back. Tests enforce path isolation before launching the child, and test cleanup deletes credential copies; fixtures retain only sanitized raw UI captures. Native shared configuration is never opened for writes by these scenarios. Session selection is checked against before/after hashes of disposable defaults; default-scope tests use that same disposable configuration. `peerLiveEnvironment` remains the common isolation source.

**PQ-3 / active test strategy.** The following named-function strategies supersede test-case inventories in Tasks 1–3; those inventories illustrate risk, not an additional procedural checklist. Keep the named production-boundary seams and exact commands.

| Risky function | Adversarial class and mechanical guard |
|---|---|
| `ParseKeySequence` | Fuzz arbitrary UTF-8/invalid bytes and numeric waits against independent byte/count/time bounds; malformed inputs never produce a write effect. |
| `ReadModelScreen` | Fuzz escaped markup, malformed/incomplete paints and repeated labels with versioned real-screen seeds; only unique recognized selected/live-status evidence is actionable. |
| `ReduceDrive` | Generate reproducible event sequences with controlled epochs, time and identity changes; independent invariants forbid replay, stale-screen selection, post-cancel writes and unconfirmed success. |
| `ApplyReceiptEvent` | Property/sequence tests prove exact-intent deduplication, changed-intent refusal, immutable terminal evidence and bounded expiry. |
| `handleAgentDrive` / `dispatchAgentDrive` | Stateful real-socket/input-loop fake drives controlled authority, lease and external-write interleavings; assert bytes/evidence, not mocked call counts. |
| `classifyWait` and model-command accounting | Table/property tests distinguish captured completed-control chrome from genuine input requests; model control cannot open a conversational turn or erase a later genuine wait. |
| `peerLiveIsolatedModelAuth` | Table tests with private temp login files prove every writable path remains disposable and native-path fallback is rejected; no credential bytes enter output/captures. |

Approval already covers the feature and bounds at 44e2e723. These refinements make the existing contract precise without changing its scope. TL skill owner/path is now known: ops/skills-draft/xx-tl/SKILL.md, updated by the hub after landing. Pair’s embedded couch skill and atlas own this issue’s documentation; do not edit the ops skill.

### 2026-10-14 — operator-directed Claude evidence and recipe revision

TL supersedes the isolated-Claude authentication work: **no credential plumbing and no Keychain reuse**. Claude’s local credential file has no usable tokens. The hub provided real read-only captures from idle pair:4, cancelled with ESC; observed header is Claude Code2.1.299. Import sanitized screen fixtures and retain their provenance. Future Claude live conformance is performed through the hub on an idle real slot, with session-only scope, not through a new login/helper. Remove the proposed peerLiveIsolatedModelAuth entity/strategy for Claude; Codex continues using the existing disposable copied-auth harness.

Claude has a **combined** model/effort picker: ❯ indicates selection, ✔ indicates current model; effort is adjusted with left/right on the same screen. Its footer explicitly names Enter=default and s=session, so session default is s for both recipes. The model’s effort default may change when navigating to another model; use fresh observed evidence. Internal recipe-only left/right effects are needed, while the approved raw key grammar stays unchanged. The supplied captures prove picker/navigation/scope controls and ESC recovery; they do not prove successful application. Obtain a hub-captured session apply/confirmation before enabling Claude confirmed-success recognition, and include hub-mediated live driver evidence before close.

This is the required stop-and-revise after observing a different agent’s recipe; the TL supplied/authorized this path in message197b2a5f. No ops TL skill edits. The implementation remains one driver with per-agent phase tables.


### 2026-10-14 09:57 UTC — captured phases, cancellation seams and input proof

**Reason:** Real Codex/Claude captures resolved the picker contract; inspecting the existing writer and RPC transport found the cancellation mechanisms already implemented. These deltas supersede the earlier two-ESC backout limit and any proposal to introduce another writer/transport for this feature. Other approved bounds and the single atomic close boundary stay in force.

**Delta / captured scope and phases (ARCH-MOCK, ARCH-CONSTRAINTS).** Codex 0.162.1 confirms session scope with “Model changed to gpt-6.1-sol low for this session only”; the scratch config hash is unchanged. Enter confirms default scope without that suffix and changes only the disposable config. “More reasoning…” opens **Advanced Reasoning**, with Max/Ultra and the same Enter=default/s=session footer. Its first ESC returns to ordinary reasoning, so full prompt backout from that phase may require **at most three separate ESC steps**: Advanced → reasoning → model → empty composer. Raise the earlier two-step cleanup cap to three; retain the absolute deadline, three-second cleanup reserve and requirement to recognize every owned phase. Never claim prompt recovery without fresh empty-composer evidence. The Advanced capture records only first-ESC recovery to the reasoning picker and child teardown, not a completed prompt backout.

Claude 2.1.299 hub captures now also prove successful session apply: “Set model to Opus 5.5 for this session only”, an empty composer and the medium effort footer. This supersedes the preceding revision's missing session-application evidence. Default persistence is still unverified, so reject Claude default scope before any input. Keep hub-mediated Claude conformance and the prohibition on extra credential plumbing/Keychain reuse.

**Delta / reuse cancellation seams (ARCH-DRY, ARCH-FUNERAL).** Use the existing `ttyio.Writer.WriteContext` for bounded PTY writes and the existing context-aware `couchmessage.Call` for socket exchanges. The latter closes blocked socket I/O on context cancellation and joins its owned cancellation callback. Add no second writer goroutine, transport or persistent model state. Service/console and wrapper lifetimes own the admitted job and absolute deadline independently of short admission/status RPCs; actual caller/session replacement and shutdown cancel it. Uncertain writes/acknowledgements never replay admission or input.

**Delta / screen proof before input (ARCH-ORDER).** Feed's atomic terminal generation is the screen identity for reducer evidence. Stamp each picker-derived write effect with `ScreenEpoch` and its parsed picker evidence. Immediately before input require the exact wrapper/session lease and a fresh complete proof: either the same generation or freshly parsed evidence identical in phase, selection, labels, model, effort and footer. Live conformance found benign full repaints can change generation without changing those decisions; permit that identical semantic proof, but refuse cached actions when recognized picker evidence differs. Unknown/incomplete screens wait without keys within the absolute budget, allowing the driver's own `/model` transient to settle. Initial command and raw effects likewise wait for a coherent frame. Stateful adversarial tests pin redraw between reducer decision and write, split/incomplete paint, operator cancellation of a blocked write and context-cancelled RPC I/O. Final success still requires a newly painted matching confirmation after apply.

### 2026-10-14 — viewport authority correction from live conformance

Live-wrapper regressions showed that retained scrollback can contain a former picker after the current viewport changes, or duplicate headings after an identical repaint. Picker decisions and effect revalidation therefore read only the coherent current viewport; retained history is separate evidence for result tails and fresh confirmation occurrence counts. Unknown current views authorize no input. This preserves the original no-input-on-an-unknown-screen intent without refusing harmless identical repainting (ARCH-ORDER).

### 2026-10-14 — wrapper writer handoff after raw key drives

**Reason:** TL reported that a raw Zellij /status injection on parley.nvim:1 at07:40 bypassed wrapper input ownership: subsequent Couch deliveries expired with “waiting for previous automatic input to clear”, and relaunch refused busy despite an empty composer. Receipt78b7fb14-68ed-4bef-a090-f3b403c89721 verified.

**Delta:** All --keys effects remain on the wrapper’s existing single input writer; never substitute direct Zellij key injection. Add an integration test through that writer which completes a key drive, observes its final empty composer, then completes a normal Couch paste/submit/confirmation. This defends the shared automatic-input lease handoff and ordinary delivery liveness (ARCH-DRY, ARCH-ORDER). Preserve the hold before live Claude acceptance after #528; no staging, restart or close is authorized by this revision.

### 2026-10-14 — failed live smoke: submit boundary and recoverable composer

**Reason:** Verified TL receipt6d4558bb-25e2-449b-ac22-63fd1e9b8d2d reports unconfirmed operation12f7dea1 on relaunched Codex ariadne:5 at the staged dbdde238 build: /model remained with a newline, failure left text behind, and --keys clear-line refused the occupied composer. Claude acceptance is still blocked until #528 fixes connection-before-focus. Stage clearance is requested and queued behind another live SDLC merge lock.

**Delta:** Reproduce submit using the actual production wrapper and terminal protocol; logical submit stays distinct from newline, and picker Enter must reach a positively recognized picker as confirmation. Recorded and live conformance must cover the same receiver/protocol path rather than treating a native-PTY success as proof of every live-pane path.

Recovery-only --keys means a validated nonempty sequence containing only clear-line and esc. It may start on a positively recognized occupied composer when the wrapper proves no active turn, picker, image, buffered input or competing automatic transaction. Model and generic key drives retain the empty-composer precondition. The same fresh exact-wrapper, sequence, owner and caller checks apply at admission and execution; do not trust an unrecognized composer or a wire-supplied ownership claim. Reuse Request.RecoveryOnly, peerComposerText and a typed Recoverable observation across the pure reducer, wrapper and Couch service (ARCH-DRY, ARCH-ORDER).

Reserve min(3seconds, one tenth of the budget remaining at execution start) INSIDE the existing absolute deadline for model failure recovery. Before that deadline, cleanup may act only on this operation’s acknowledged /model composer text or a recognized owned picker, never on human cancellation, a turn, changed unrelated text, possibly-applied input, unknown write progress or an unknown view. Supply the current coherent screen at recovery ticks, proof-pin cleanup keys, cap picker backout at three separate ESCs, and verify a fresh empty prompt before reporting recovery. If the available proof or budget cannot establish recovery, retain unconfirmed and name inspection/manual recovery; never claim the model change failed or cleanup succeeded without evidence (ARCH-CONSTRAINTS, ARCH-FUNERAL). Capture real multiline-command clearing behavior before choosing cleanup bytes.

Add recorded RED/GREEN tests for submit-vs-newline under the real protocol, recoverable-command cleanup within the same budget, no cleanup over changed/human input, recovery-only admission on occupied composers, continued generic/model refusal, and key recovery followed by normal Couch delivery. Re-run isolated live conformance using the production input writer and matching terminal protocol; request another operator smoke only after the fixes and evidence are committed. No peer slot input or restart is authorized for this implementation work.


### 2026-10-14 — measured submit timing and native selection evidence

**Reason:** Actual Run/socket/native-child conformance reproduced the operator failure in both scratch and Pair virtual terminals: `/model` and CR can reach Codex in one rapid input burst. Changing CR to CSI13u did not fix it. A 150ms quiet gap did open the picker. The pinned [Codex 0.162.1 paste-burst implementation](https://raw.githubusercontent.com/openai/codex/rust-v0.162.1/codex-rs/tui/src/bottom_pane/paste_burst.rs) retains Enter suppression for 120ms after fast text, even after the text has rendered. Keyboard negotiation differs between Ghostty and the old scratch environment, but is not the demonstrated cause. Native raw capture also proves selected picker rows may use a unique `›` without reverse-video styling.

**Delta:** Keep submit bytes unchanged. Share the existing 150ms peer-submit delay as a pure effect NotBefore bound after acknowledged Codex text, covering raw and model Enter/submit. The single writer retains pending effects without sleeping or extending the absolute budget. Reserved model recovery preempts an un-emitted delayed key; absolute-clock checks fence input even before a context callback runs. Require fresh post-write empty evidence for automatic cleanup. Codex selection accepts exactly one explicit `›` under the recognized complete heading/footer, with duplicate/unknown/incomplete/stale evidence still refused. Preserve the recorded reverse-video cases as compatible evidence.

Native multiline Ctrl+U capture leaves `/model` behind after the empty second line is removed; it does not establish whole-composer deletion. Automatic cleanup therefore remains limited to the acknowledged exact single-line command or recognized owned picker. Explicit recovery clears the current line. Actual Run/socket conformance covers both scratch and Pair terminal negotiation plus explicit dirty-composer recovery; direct native cases retain session/default/unsupported-effort checks. No live peer input, restart or staging is authorized here; pair:0 remains reserved for #528.


### 2026-10-15 — reconcile landed recovery and stall supervision

**Reason:** TL unblocks #544 after #528 acceptance/landing and the operator's main Couch restart. Integrate main3dcdd38c361e4ceed303374099a16ab668c9553d before renewed verification. Nine conflict sites overlap additive documentation, source inventory, candidate-bound recovery receipts, endpoint validation and wrapper shutdown/state fields.

**Delta:** Preserve both typed drive intent/digest and resume CandidateKey through validation, receipt identity and dispatch; retain main's endpoint context/incarnation safeguards and once-only wrapper shutdown while cancelling active drives. Keep both inventory/docs additions. Main's automatic-input reducer now owns alerted residual waits and a monotonic stall sequence: explicit recovery can take an awaiting-empty or alerted residual lease after positive composer proof, while active owners remain exclusive. Route drive begin/finish through that reducer, preserve its sequence and distinguish confirmed final input/recovery from uncertain outcomes. New integration tests first reproduced alerted recovery refusal and sequence reset; both are now GREEN. No broadened input authority or deadline is introduced (ARCH-DRY, ARCH-ORDER).

Rerun targeted races, isolated native conformance and the recorded full suite on the integrated commit. The new broker operations and wrapper endpoint capability require this branch's Couch plus current wrappers in selected idle slots. Propose staging pair:0 for an operator-batched restart only after those checks; no staging or restart is authorized by this revision itself. Renew session-only Codex/Claude model selection and explicit keys/dirty-composer recovery, followed by ordinary delivery to prove the input lease hands off.


### 2026-10-15 — integrate confirmed recovery before the combined smoke

**Reason:** Verified TL receipt473f6e83-ec19-4a8d-ba41-4bebe74fabde authorizes staging after #556 landed, for one operator restart covering both items. Main also brings #572's trimpath build/test contract and #577's environment-probe correction.

**Delta:** Preserve #556's RecoveryConfirmation token, confirmation offers, receipt identity and prepared-recovery routing alongside #544's typed Drive intent, digest and bounded result. Keep both operation declarations and source classifications. Documentation retains current confirmed-recovery behavior and the drive instructions. Native wrapper admission remains on main's exact-grant seam; rerun actual Run conformance with that seam present (ARCH-DRY, ARCH-ORDER). Record a new suite using main's declared make PAIR_GO_TRIMPATH=-trimpath test command. Stage a clean committed merged HEAD and verify both binaries against it. The operator owns the Couch restart; live drive authorization remains enforced by the owning-hub broker guard.


### 2026-10-15 — final restart batch excludes Qoder repair

**Reason:** Verified TL receipt670622a8-d2a5-48e2-892d-9f0a7bca8494 directs immediate merge/restage after #587; do not wait for #569, which rides a later restart.

**Delta:** Integrate origin/main2156c014cee0af8fa1662064ebfe32d746e6568a, carrying #573/#579/#584/#585/#594/#587. Preserve both resource and drive dispatch, source inventories and skill costs at three additive conflict sites. Main's legacy-message serialization, broker-release distinction and capture buffering remain unchanged by reconciliation. Targeted drive/recovery/protocol/resource/native ownership tests plus inventory and actual native Run conformance verify the integration before staging. The operator still owns the Couch restart and the hub runs live model/key acceptance (ARCH-DRY, ARCH-ORDER). No Qoder repair or new live input is part of this restage.


### 2026-10-15 — boundary inventory reconciliation (BR-2)

**Reason:** The original Core concepts table described prospective names rather than the delivered symbols. This revision supersedes that table and the statement “one table per agent”; earlier design text remains historical evidence.

**Delta:** Keep one shared typed reducer and explicit per-agent screen parsers. A separate Recipe type/table would duplicate the parser and reducer transition authority without a second consumer (ARCH-DRY, ARCH-PURE). Codex/Claude picker phases and scope support are implemented by ReadModelScreen plus ReduceDrive/NewDrive/apply; recorded per-agent fixtures qualify those branches. This preserves the design's purpose: labels and complete recognized screens determine navigation, never assumed row positions. Claude default scope still refuses before input. The following is the current symbol inventory:

| Entity | Kind | Path | Status / ownership |
|---|---|---|---|
| KeySequence, ParseKeySequence | PURE | cmd/internal/agentdrive/keys.go | new; bounded caller keys |
| ModelRequest, ModelScope, Request, Result | PURE | cmd/internal/agentdrive/recipe.go | new; request/result data, no Recipe entity |
| ScreenEvidence, ReadModelScreen, confirmationCounts | PURE | cmd/internal/agentdrive/screen.go | new; explicit Codex/Claude parsing |
| DriveState, DriveEvent, ReduceDrive, NewDrive, apply, choiceFailure | PURE | cmd/internal/agentdrive/model.go | new; shared transitions, agent-specific supported scopes and apply actions; bounded failure choices survive changing tail |
| OperationReceipt, ApplyReceiptEvent | PURE | cmd/internal/couchmessage/operation.go | modified; bounded result/intent ownership |
| classifyWait | PURE | cmd/internal/couchcmd/wait_notice.go | modified; model-control confirmation recognition |
| handleAgentDrive, installAgentDriveService | INTEGRATION | cmd/internal/couchcmd/agent_drive_service.go | new; authenticated console queue |
| RemoteEndpoint.ObserveDrive, StartDrive, DriveStatus, CancelDrive | INTEGRATION | cmd/internal/couchmessage/drive_endpoint.go | new; bounded exact-bound RPC |
| endpoint request/response and drive intent | INTEGRATION | cmd/internal/couchmessage/endpoint.go, protocol.go | modified; typed wire parsing |
| dispatchAgentDrive | INTEGRATION | cmd/internal/wrapcmd/agent_drive.go | new; existing writer and admission |
| wrapper endpoint admission | INTEGRATION | cmd/internal/wrapcmd/peer_runtime.go | modified; binding and mailbox |
| automatic input / lifecycle accounting | INTEGRATION | cmd/internal/wrapcmd/automatic_input.go, wrap.go | modified; shared ownership and translator |
| CLI adapter | INTEGRATION | cmd/internal/couchcmd/agent_drive_cli.go, agent_drive_messages.go | new; argv and read-only receipt polling |
| stateful drive harness | INTEGRATION | cmd/internal/wrapcmd/agent_drive_test.go | new; harnessSessionFake and input-loop fixtures |
| native direct conformance | INTEGRATION | cmd/internal/wrapcmd/agent_drive_conformance_test.go, agent_drive_live_test.go | new; disposable native PTY |
| actual Run/socket native conformance | INTEGRATION | cmd/internal/wrapcmd/agent_drive_virtual_conformance_test.go | new; scratch and Pair virtual terminal |
| peerLiveEnvironment, peerLiveLocalAuth | INTEGRATION | cmd/internal/wrapcmd/peer_live_test.go | reused unchanged; existing disposable Codex auth/environment seam, no new Claude credential flow |

Live fleet set-model/keys/delivery smoke remains post-landing after the next operator restart, owned by TL ops:0 under submitted receipt35fe918a. Recorded/native checks and the normal close review/full-suite gates remain landing prerequisites.


### 2026-10-15 — executable native conformance entrypoint (BR-3)

**Reason:** Boundary round2 accepted BR-1/BR-2 but found the planned opt-in conformance target absent; ordinary test execution skips native-login cases.

**Delta:** Add `make test-agent-drive-live` in Makefile.local, enabling both PAIR_LIVE_AGENT_DRIVE=codex and PAIR_LIVE_AGENT_DRIVE_VIRTUAL=codex and selecting both complete suites. Run this target after drive/parser/writer changes or Codex upgrades, before landing them; retain regular suite isolation and no peer-slot input. Atlas documents prerequisites, both tested paths and that cadence (ARCH-MOCK). Verify the exact target executes all three direct-native cases and both actual Run/socket terminal variants without SKIP.


### 2026-10-15 — native recovery observation/admission/dispatch policy (BR-4)

**Reason:** Native conformance reused an unqualified newer observation and did not handle a legitimate zero-input dispatch refusal. Production refusal remains required when sequence or complete composer proof changes before the first write.

**Delta:** The isolated fixture observes until Recoverable/non-idle for each new request, admits one distinct ID, and polls that ID to terminal. Only exact known stale-observation/composer guard refusals with OutcomeRefused/Step0 allow another independently qualified ID, at most three attempts within the same outer context. Unconfirmed/partial/cancelled/unknown outcomes stop immediately. Test-only boundaryRecoveryDrive owns that sequence behind boundaryRecoveryEndpoint; controlled tests force qualification loss and zero-input refusal, assert no unsafe retry, and actual wrapper regressions cover sequence/screen changes at admission and dispatch with zero bytes and no later replay. No production retry is added (ARCH-ORDER).
