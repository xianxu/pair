# Evidence-based slot recovery implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded adapters/tests and superpowers-executing-plans for warm-context lifecycle integration. Steps use checkbox syntax. This is a proposed design awaiting TL/operator approval; no code is authorized yet.

**Goal:** Recover the conversation the operator actually saw, with explicit confirmation when identity is inferred, and never call an empty resurrected terminal live.

**Architecture:** Gather exact terminal/process and native transcript observations through existing adapters. A pure recovery decision projects one primary action; confirmation binds that action to an immutable evidence snapshot, then existing thread holds, start claims, park and resume machinery execute it with revalidation. Heuristics never change the established-binding resolver.

**Tech Stack:** Go, existing sessioninventory/runtime, Couch thread store and operation queue, Zellij and process observers, retained tty renderer, existing one-shot orientation transport.

## Design and alternatives

Recommended: operator-confirmed inference, with a conservative first scoring policy and a fresh tty-oriented fallback. Keep established bindings unchanged. Keep the existing resolver's causal proof independent from recovery suggestions (ARCH-DRY, ARCH-SECURE).

Alternatives rejected: (1) silently promote the newest cwd match into an established binding: timestamp and root originator do not establish identity, as the subagent example demonstrates; (2) retain the current refusal and require manual native-session selection: safe but fails the operator's requested recovery ladder.

### What the operator sees

| Observation | Visible state | One primary action |
|---|---|---|
| Exact current agent process is usable | Running | Open |
| Agent absent/empty resurrected terminal; native identity proven | Needs relaunch | Resume conversation |
| Agent absent; exactly one strong inferred identity | Needs relaunch | Resume inferred conversation… |
| Agent absent; ambiguous/no strong eligible identity | Needs relaunch | Start fresh from previous screen… |
| Agent/process or evidence inspection incomplete | Checking / state unavailable, with reason | Retry inspection |

A confirmation sheet for inference names agent, native ID, exact cwd, last-active time, tty activity delta and tail-match percentage. A fresh confirmation says explicitly that the conversation starts fresh and names the retained tty, branch and issue. Back/cancel is navigation, not another recovery decision. No menu of competing guesses. Running/starting/unknown remain distinct internally. A failed observation never permits stopping a possibly running agent. Existing running-slot restart busy protections remain in force.

### Evidence and ranking

The source of chronology is the launch associated with the retained tty capture, including its ledger ordinal/offset and exact scope/tag. Later empty/recovery helper starts cannot move that boundary. If provenance is missing, show that the scan is incomplete; do not invent a launch time or use the newest record as a substitute.

Native adapters return a validated root ID, cwd history, interactive classification (yes/no/unknown with provenance), creation/last-message time, file mtime, file generation, and a bounded final assistant/user message window. Reuse scanner root/parent/path validation; do not overload first-activity chronology or the causal event representation that drops assistant text.

- Codex: exact cwd, validated root, source=cli, no parent/subagent source; originator is supporting metadata only. Exclude exec/tool/subagent sources even if originator says codex-tui.
- Claude: validated root path, no sidechain/agentId, supported cli entrypoint and exact cwd in the relevant message tail. Reject sdk-cli. Missing/conflicting metadata is unknown. Multiple cwd values require the matching tail segment, not any historical match.
- Grok: validate summary.info.cwd and root identity; explicit headless is excluded. Current local samples provide no positive interactive discriminator. Omitted session_kind is unknown and cannot qualify for inference. Existing exact/proven Grok resumes remain supported; its unproven case takes the explained fresh fallback when absence is proven. Add positive interactive support only with measured schema evidence, not guessed strings.

Require exact physical worktree cwd, modification after the source launch, and a valid resumable root not already authoritatively owned by another live slot. mtime alone cannot prove activity: show semantic last-message time when present and the filesystem timestamp separately if they disagree.

Normalize rendered tty and authored message text by whitespace folding and narrowly specified Markdown/bullet decoration removal; exclude tools, thinking, metadata and startup instructions from candidate message text. Use existing RenderBoundedLines with 100 retained history rows plus viewport, not ANSI regex stripping. Tail match is the fraction of contiguous eight-word windows from the candidate's last complete assistant message occurring in that rendered window. Initial conservative proposal: at least 12 words and 100% match, plus eligible chronology; no fuzzy threshold in this issue. The 12-word minimum is a proposed guard against trivial common text, not a measured universal threshold. The measured pair:6 message has 28 words and 21/21 windows. Short, partial or indistinct matches go to fallback.

Rank eligible candidates by match percentage, then absolute last-activity delta to tty activity, then stable ID for deterministic display only. Exactly one qualifying strong candidate is required; timestamps never break a tie between strong candidates. Show the measured delta, not a probability. Duplicate transcript artifacts for one native root coalesce. Unknown interactive provenance cannot qualify; enumeration/read failure makes the scan incomplete and does not masquerade as zero candidates.

### Confirmation is bounded authority

Preview returns a value containing thread address/revision, source launch/capture identity, selected native ID, transcript generation/digest, algorithm version, evidence summary and proposed action. Keep it in the owning UI operation, not a new persistent suggestion database. The server constructs it; agents cannot supply an arbitrary ID and call it confirmed. Confirmation applies to exactly that preview.

On confirm: acquire existing ThreadGate; reread revision, current native binding, source capture, selected candidate and resource occupancy; recompute the decision. A changed action, candidate, stronger proven binding, capture generation or competing owner invalidates the preview before effects. Cancel/replaced UI operations ignore late scan completions by request generation. Scans have console lifetime cancellation and no unowned background work.

Confirmed inference is explicit human-selected resume authority for this launch, not fabricated BindingEstablished. Reuse TargetExplicitResume and native-ID validation, build the existing exact RequiredSessionID launch profile, and record the requested native identity in existing launch evidence before execution. Do not relax RequireNativeResumeBinding globally: the ordinary proven path retains its current checks; the confirmed path has a typed, revalidated authority consumed only by recovery. Subsequent startup reconciliation must retain that exact requested target even if registration is interrupted. Successful native launch records restore ordinary durable ownership. Unknown launch outcome preserves the existing start claim and forbids duplicate spawn.

### Fresh orientation fallback

Preflight the saved same-agent config, exact retained capture readability and branch/issue before touching a terminal. Use the existing park/recovery cleanup with preserved scrollback and its exact source identity; never manufacture a successful park receipt. An empty resurrected server may be retired only after authoritative empty-pane/dead-agent observation and exact binding recheck. Capture preservation must finish before any action that clears the old terminal.

Start fresh in the same thread/tag using tracked-start machinery, retaining the tty-only PreviousSession reference for --previous --all. This is a dedicated confirmed recovery action, not a relaxation of switch-agent's different-agent admission and not a new compaction protocol. Reuse the same capture publication and source-identity checks used by #517. If the existing cleanup cannot safely produce the needed source receipt for a measured dead/resurrected shape, stop and revise the plan before adding a parallel lifecycle.

Build a short orientation request: fresh conversation; slot address, branch and issue; literal exact previous capture path; `couch --peek repo:N --previous --all`; historical text is context, not fresh operator authorization. Use existing orientation.Request/DeliveryState and wrapper qualified-composer delivery, tied to the new start nonce. Do not put a reusable prompt into saved argv/defaults. Delivery failure or uncertain submission reports the existing exact delivery status and never automatically pastes twice. If the tty itself is unreadable, preserve the old record and offer inspection retry; do not claim the fresh session is oriented. Ordinary switches and restarts gain no unsolicited orientation.

### Live means a usable agent, not a server

Introduce one observed AgentUsability value: running, absent, unknown, plus exact source identity. The thin observer combines admitted wrapper registration with exact agent-child/process and pane evidence. A live Pair helper, surviving Zellij server, stale pid file, or ready marker alone is insufficient. Reuse existing broker/process/session identity seams; require launch nonce and process start identity/parentage where available to prevent PID reuse. Complete pane inspection showing only waiting-to-run panes is positive empty-terminal evidence; a failed query is unknown. A dead old helper does not imply a dead detached agent if exact current agent evidence proves it survives.

Apply this single value to inventory, detached/warm resume admission, effect-time recheck, automatic startup reattachment, and the attached-pass UI overlay. Attach success cannot promote absent/unknown to running. Batch observation per inventory refresh; expensive native inference runs only on selected recovery, never per paint. No persisted thread-state enum is added merely for the display label; if implementation changes a modeled lifecycle, locate/read its vocabulary owner first.

## Core concepts

Names below are proposed new symbols unless marked existing; reconcile this table against the actual diff at each boundary.

| Pure entity | File | Status / purpose |
|---|---|---|
| RecoveryCandidate, InteractiveEvidence | cmd/internal/sessioninventory/recovery_candidate.go | new; native candidate values with provenance, no IO |
| NormalizeRecoveryTail, RankRecoveryCandidates | cmd/internal/sessioninventory/recovery_match.go | new; deterministic text/window match and unique-strong decision |
| AgentUsability, DecideAgentUsability | cmd/internal/couchcore/agent_usability.go | new; derive agent usability from already-observed process/pane facts |
| RecoveryPreview, DecideSlotRecovery | cmd/internal/couchcore/recovery_preview.go | new; one proposed action and confirmation identity |
| ClassifyThread / ThreadEvidence | cmd/internal/couchcore/actionableinventory.go | modified; consume usability instead of helper/server-only live inference |
| DecideResume | cmd/internal/couchcore/resume.go | modified; warm path requires usable-agent authority |
| Recovery orientation builder | cmd/internal/orientation/recovery.go | new; literal coordinates/branch/issue prompt, direct value tests |

| Integration owner | File | Status / wraps |
|---|---|---|
| Native recovery collector | cmd/internal/sessioninventory/recovery_runtime.go | new; existing Runtime metadata/record reads and native validation |
| Native recovery parsers | cmd/internal/sessioninventory/recovery_codex.go, recovery_claude.go, recovery_grok.go | new; versioned raw-record interpretation beside existing scanners; pure parsing functions separately unit tested |
| Agent usability observer | cmd/internal/couchcmd/agent_usability.go | new; broker/process/Zellij observations, injected into couchcore |
| Existing session/warm observation | cmd/internal/couchcore/sessionevidence.go, detachedsessions.go, resume.go | modified; shared usable-terminal proof and selected-target recheck |
| Recovery executor | cmd/internal/couchcore/recovery_resume.go | new; preview/confirm orchestration, existing ThreadGate and tracked start |
| Existing capture/cleanup/start owners | cmd/internal/couchcore/park.go, previous_session.go, launch_existing.go | reused; modify only if integration requires a shared entry point, never duplicate their transition logic |
| Recovery confirmation UI | cmd/internal/couchtty/menu_recovery.go | new; selected-row scan and existing confirmation frame |
| Existing action/reattach consumers | cmd/internal/couchcore/actor_actions.go, resume_route.go; cmd/internal/couchtty/menu_actions.go, menu.go, menu_reattach.go, console_reattach.go | modified; derive action and prohibit attach-only live projection |
| Existing test doubles | sessioninventory Runtime fixtures; couchcore fake process/session/runner/store | extended; complete/failed scans, append/replace, waiting panes, process death, delayed registration |

## Bounds and lifecycle

ARCH-CONSTRAINTS: action-triggered local inspection, not keystroke work. Initial engineering budgets (subject to measurement in M1): 30-second cancellable scan, 10,000 metadata entries, 64 eligible roots, 256 MiB total native/tty reads, 2 MiB individual authored-record limit; exceedance returns incomplete, not a guessed winner. No network/model calls. Keep only bounded final message windows in memory (at most 256 KiB per root); output contains IDs, references and scores, not full transcript text. Reuse renderer limits/leases and report replay loss. Check these budgets on the real pair:6 snapshot and large/oversize fixtures before freezing constants; revise if they exclude the motivating case.

ARCH-FUNERAL: previews die on cancel/selection change/console shutdown; no durable ranking cache. Confirmed launch identity uses existing bounded launch record/retention owners. Preserved capture uses existing archive collector and reader leases. The one previous pointer is replaced with the next exact predecessor. Orientation is one per launch attempt and follows existing delivery-state retention. No new forever-growing log or native transcript copy.

ARCH-ORDER sequence contract:

| State/event | Result / allowed effects |
|---|---|
| inspecting + complete facts | preview one action; no terminal writes |
| inspecting + cancellation/partial read | cancelled/incomplete; preserve source |
| preview + decline or stale generation | discard/recompute; no launch |
| confirmed + exact facts unchanged | acquire lifecycle claim, preserve source, execute one action |
| confirmed + candidate/record/resource changed | refuse before effects; new confirmation required |
| launch pending + supervisor death/late registration | existing start recovery owns exact target; never infer absence from timeout |
| fresh launch + orientation indeterminate | retain target/delivery evidence; no automatic resend |
| success + later refresh/attach | running only with new usable-agent proof |

## Implementation and verification

### M1 — Native evidence, conservative decision, and truthful liveness

- [ ] Add sanitized real-shape fixtures from the measured Codex root/subagent/exec, Claude cli/sdk/sidechain/multi-cwd and Grok headless/unknown records; preserve causal field shapes, not private text. Record version/provenance beside fixtures.
- [ ] Write pure candidate/parser/match tests first. Exact pair:6-shaped fixture must select the root with 100%; later helper cutoff, source.subagent exclusion, tied roots, common/short text, wrapping/Markdown, torn tails, contradictory metadata, unknown Grok and file replacement must fail closed. Implement only after red evidence.
- [ ] Add collector tests through existing stateful Runtime: complete enumeration, unreadable directories, truncation, append, generation change and declared limits. Prove candidate chronology uses original tty launch, not newest helper.
- [ ] Add pure usability and stateful inventory/warm/reattach tests first: live server + empty panes, live helper + dead agent, detached surviving agent, stale marker/PID reuse, unknown query, death between preview and attach. Wire every enumerated consumer to the common value. Assert both display state and absence of forbidden spawn/attach.
- [ ] Run `go test ./cmd/internal/sessioninventory ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Recovery|Usability|Classify|Warm|Reattach' -count=1`; targeted race on new seams. Mutation checks remove source exclusion, change launch cutoff, and restore attach-only promotion; each must fail a relevant test.
- [ ] Read-only conformance measurement of current pair:6-shaped capture and installed agent versions; no real slot recovery without operator request. Update atlas/couch.md, atlas/index.md and native-format fixture docs; close M1 through sdlc.

### M2 — Confirmed resume and fresh tty-oriented fallback

- [ ] Write executor sequence tests for confirmed inferred resume through the real operation/action route. Assert native ID argv, unchanged defaults, source preservation, no silent established-binding promotion, and rejection of forged/stale confirmations. Inject competing candidate/owner and record updates immediately before claim/effect.
- [ ] Implement selected-row preview and confirmation using existing menu frame/queue; block duplicate submissions with current operation ownership. Test cancel, row change, delayed scan, keyboard/mouse primary action and accessible evidence text. CLI/headless callers must not gain silent heuristic resume; their existing behavior remains unless separately approved.
- [ ] Write fallback tests first: preserve exact tty before terminal replacement, same tag and --previous --all path, literal branch/issue orientation, qualified delivery, operator input interference, process death, restart and ambiguous submission. Reuse current lifecycle/transport, never put orientation in defaults. Test no-capture/partial-scan/unknown-agent absence refusals explicitly.
- [ ] Extend stateful fake sequence matrix for cancellation after claim, helper fork failure, late readiness, startup reconciliation, capture retention under an active reader and retry. Verify original recovery authority survives any unconfirmed external effect.
- [ ] Run targeted suites and race for touched seams, `make build`, then full `make test` in the background under suite budget. Mutation-check effect-time candidate revalidation and no-resend behavior. Inspect the core-concepts table against actual diff and update through Revisions.
- [ ] Update README and atlas for one-action recovery wording and confirmed inference limits. TL coordinates operator acceptance: show exact pair:6 candidate and evidence, confirm correct native resume; exercise fresh fallback on a scratch slot, and a resurrected empty-terminal fixture. Do not reboot pair:6 merely to reproduce.
- [ ] Close M2, then issue close/publish using sdlc only after proof and acceptance.

## Approval points

The evidence-anchored cutoff and source/parent correction are approved by TL receipt a9307d75. Remaining design choices for approval: conservative full-tail match (minimum 12 words), unknown-interactivity Grok fallback, and same-tag fresh recovery with one-shot tty orientation. No implementation or live recovery is performed by this plan submission.

## Revisions

### 2026-10-13 09:12 PDT — Live split incarnation is the primary defect; revised ladder

**This revision supersedes the draft design, owner table, scoring rule and milestone details above.** Preserve that draft as decision history, not implementation authority. TL receipts 08e69ac9 and f0c32c35 supply the new requirements and measurement direction. The issue's original unconditional human-confirmation requirement is superseded: calibrated high-confidence inference may auto-select; close calls require confirmation. Selection still happens only in an operator-requested recovery, never as a background launch.

#### Measured facts and the missing association

TL observed old Pair wrapper 41844, Codex 41860 and Zellij server 41843 still alive after restart; helper 9253 then created server 9682 under the same name/socket and showed empty panes. Messaging/peek addressed the old wrapper; the operator attached to the new server. The fix must make that split visible and forbid creating another incarnation. Pair:6 is not a dead-agent fallback example. Do not stop/recover its processes during implementation measurements; live recovery belongs to the operator.

Pair already runs bare `zellij attach NAME`, with no --create. Installed attach help has no no-resurrection flag. Pair's existing session-owner probe enumerates exact server identities but socket observation only lstats the pathname; its final revalidation does not establish which server currently owns that socket. Multiple same-name servers already classify contested in `launcher/session_servers.go`; preserve and apply that fact at every launch/attach boundary.

Capture provenance is also weaker than the first draft assumed: ready record names ordinal 18/nonce/agent PID; raw birthtime and first event agree chronologically, but raw/events contain no launch nonce or ordinal. Several ledger launches share Pair-log offset zero. The current tty is mutable and advanced at 09:02. Therefore ordinal 18 is well-supported historical inference, not an exact capture-generation proof. Never use equal Pair-log offset or newest ledger entry as a proof of capture ownership.

#### M1: One terminal owner and an attachable agent

The first milestone is the duplicate-server fix and truthful liveness. Reuse `SessionOwnershipExpectation`, server enumeration, broker wrapper identity and process start identities to gather a single `TerminalOwnership` observation for the exact thread. It contains expected name/socket, all matching live server instances, reachable socket ownership evidence, wrapper/agent parentage and exact launch identities. Outcomes are coherent-live, absent, conflicting or unknown. PID alone, pathname alone, a server listed by name, and successful attach are insufficient.

At the final effect boundary, hold one Pair-level OS-backed per-thread/session lock across observe → decide → attach/create handoff. All Pair routes (Couch resume, standalone resume, explicit creation and automatic reattachment) use that owner; a Couch-only mutex does not serialize two Pair processes. Name the lock from canonical scope/thread and resolved session identity, not cwd spelling; reuse an existing lock primitive. Lock acquisition is bounded/cancellable and a dead holder releases through OS lifetime, never blind lock-file deletion. Reobserve after waiting. Check old live wrappers/agents and all same-name servers even when the socket/session listing reports absent. Conflict refuses with zero spawn and explains the old live session/new reachable session discrepancy. Do not unlink a live or contested socket, kill a server, or silently start a differently named replacement.

First measure Zellij configuration on isolated scratch names and directories: `session_serialization false`, metadata-disabling options if available, fresh missing session, formerly serialized exited session under old settings, live session, and two Pair contenders. Verify server/process/socket effects, not just CLI exit status. If a config makes bare attach fail without resurrection, apply it to Pair's shared Zellij config and combine it with the lock. Existing cached serialized sessions must be covered; never sweep global Zellij caches. Record exact tested Zellij version and config in a live-conformance fixture. The no-create guarantee is for cooperating Pair paths plus the measured Zellij config; revalidate exact resource identity across external operations and refuse interference.

If no config provides the required attach-only behavior, return to TL with a minimal upstream change and maintenance cost before proposing a fork. Do not implement a precheck and call it atomic. No new Zellij dependency/fork is approved by this plan. Until that question is resolved, do not promise safe automatic warm reattachment.

Visible model: coherent/attachable agent → **Running / Open**; ownership conflict → **Needs relaunch / Inspect recovery** (no launch); proven absence → **Needs relaunch / Recover**; unknown probe → **State unavailable / Retry inspection**. The recovery action below resolves the conversation rung. A live orphan reachable only through the wrapper is not “Running” in the operator's row. One shared observation feeds inventory, warm admission/recheck, startup recovery, restart admission, message targeting where it assumes attachability, and UI attach overlays. Existing uncertain-target protections from #517 stay mandatory.

#### M2: Calibrated native recovery plus the tty tail

Only after terminal ownership is resolved to a safe recovery state does conversation selection run. Proven native binding wins. Otherwise rank candidates from exact worktree cwd, native root/source/parent, last semantic activity, and overlap with a bounded rendered tty tail. Text overlap is the principal ranking evidence; metadata describes provenance and filters established noninteractive/subagent executions. Do not trust the claim “subagents have no tty” as a universal identity proof: their text can be quoted by the root, so copied-summary negatives belong in calibration. Missing interactivity metadata is uncertainty, not proof of a tool run or proof of an interactive one.

Offline measurement so far: three actual capture/native pairs (pair:2/5/6), 45 transcript candidates comprising 5 CLI roots, 19 exec and 21 subagents. Using last three assistant messages of at least 16 words, normalized eight-word-window coverage, winners scored 100% (38/38, 20/20, 22/22), all competing scores were zero. This proves separation for these samples, not a fuzzy threshold. Additional calibration must include clipped tty windows, wrapping/decorations, common boilerplate, copied subagent summaries, concurrent roots, absent positives and an untouched holdout. Record raw scores, selection margins, false selections and abstentions. Do not hardcode 80%/20 points and describe those numbers as calibrated.

Policy has named `automatic`, `close-call` and `no-match` outcomes, selected from a versioned calibration artifact. Auto requires a single eligible best candidate above the calibrated coverage threshold and separation margin, complete enough evidence, and no contradictory ownership. Close calls show candidate/evidence and require operator choice; no-match/ambiguous-without-a-defensible-choice offers fresh tty-oriented startup. Scores describe matching text, not posterior probability. No threshold changes after seeing the holdout without resetting calibration. If the corpus cannot support a defensible auto threshold, report that evidence gap to TL before implementation approval; do not silently replace the requested auto path with always-confirm.

Use the capture-producing launch boundary when explicitly known. For historical captures with no producer metadata, retain an inferred epoch interval based on exact ready identity, file identity/birthtime and event timestamps; disclose its provenance and do not promote it to exact. Inference eligibility must tolerate the verified older-capture/newer-empty-helper case. An inferred epoch by itself cannot auto-authorize a launch. Read-only calibration determines whether transcript and tty evidence together can support an automatic choice on a historical capture. Future capture metadata records the producer launch ordinal/nonce and raw/events identities atomically when that association becomes known; failed publication is unknown. Never use same-offset lookup as a unique join.

“Best resume” combines the chosen native conversation with a short one-shot “where we were” orientation naming the retained tty tail, branch and issue. This now applies to recovered native resumes as well as fresh fallback; ordinary healthy resumes/switches remain unchanged. Reuse the existing qualified-composer delivery state machine, extend its launch-purpose validation explicitly for recovery, and preserve interruption/indeterminate delivery behavior. Never store orientation in reusable argv or resend it automatically after uncertain submission.

#### Resolve the three feasibility gaps before code

1. **Pair's second binding gate:** `launcher/createflow.go` rechecks ResumeRequired independently. Introduce a typed `RecoveryAuthority` carried through Couch's private trusted launch profile and consumed at this existing launcher decision. Ordinary resume still requires its original proven binding. Recovery authority names exact native ID, thread/attempt, decision kind (calibrated or operator-confirmed), evidence fingerprint/version and capture source. Pair validates current native root/cwd/generation and source occupancy before composing exact resume argv; stale authority refuses. Test through the actual create flow, not just runner arguments. A CLI caller cannot pass an arbitrary ID and label it operator-confirmed.

2. **Durable intent before effects:** extend the existing StartClaim with an optional bounded `RecoveryIntent` owned by the threadrecord/starttransaction model. Commit it before helper release; preserve it through unknown promotion and restart reconciliation. It records selected native ID/action and evidence/capture reference, not the whole transcript. Order: validated decision → revision-checked claim+intent → blocked helper → exact session binding → release → Pair launch ledger with requested ID → registration. Crashes before ledger publication must still retain the requested target, never select an old binding by fallback. Reuse existing start states/events; add a new state machine only if those transitions cannot represent the required knowledge, and revise before doing so. New record version/decoder compatibility must be explicit, including legacy reads and unsupported-new-version refusal; do not silently add a semantic contract to old schema 2. #527 owns better generic diagnostic wording, not #528's valid schema/migration tests.

3. **Truthful dead-source capture:** a dead source is not VerifiedPark. Generalize previous tty evidence to a validated source union: existing verified-park source or an absence-backed source with exact thread, capture file identity/generation, old terminal identity and completed absence observation. Do not forge ParkIdentity. Keep legacy park-source decoding; the new discriminator and optional recovery intent belong to the explicit new record schema. A capture-publisher adapter holds the existing retention lease, preserves the exact raw/events bytes and metadata, then publishes the descriptor under thread revision before replacement. Recheck terminal absence immediately before effects. Unknown ownership cannot publish an absence claim. Both source variants feed the same --previous --all selector and retention owner. This narrowly extends the existing descriptor/reader, not a second history system.

#### Active owner map (replaces earlier table)

| Entity / owner | Kind | Location | Planned delta |
|---|---|---|---|
| TerminalOwnership decision | PURE | cmd/internal/launcher/session_owner.go, session_servers.go | modify existing expectations/classification with coherent socket/server/agent observations |
| OS ownership observation and effect lock | INTEGRATION | cmd/internal/launcher/session_owner_os.go, osruntime.go, lifecycle.go, createflow.go | modify; authoritative final observation and serialized attach/create |
| Pair Zellij policy | CONFIG | zellij/config.kdl | modify only to experimentally proven no-resurrection configuration |
| AgentUsability projection | PURE | cmd/internal/couchcore/agent_usability.go | new projection of shared ownership evidence |
| Native recovery candidate/score/policy | PURE | cmd/internal/sessioninventory/recovery_candidate.go, recovery_match.go | new versioned, calibrated decision; direct value tests |
| Versioned native parsing | PURE | cmd/internal/sessioninventory/recovery_codex.go, recovery_claude.go, recovery_grok.go | new byte-to-value parsers reusing scanner validators; do not classify IO as pure |
| Native/tty evidence collection | INTEGRATION | cmd/internal/sessioninventory/recovery_runtime.go; cmd/internal/couchcore/recovery_evidence.go | new adapters over Runtime, ActivityForSession and retained renderer |
| Capture producer metadata | PURE + IO ADAPTER | cmd/internal/artifactpath/capture_metadata.go; existing capture publication callers | modify explicit schema and producer association; locate writer before implementation |
| RecoveryPreview / recovery decision | PURE | cmd/internal/couchcore/recovery_preview.go | new; proven/automatic/close-call/fresh/incomplete result |
| RecoveryAuthority / RecoveryIntent | PURE | cmd/internal/launcher/launch_args_policy.go; cmd/internal/threadrecord/record.go and validators; cmd/internal/couchcore/starttransaction.go | modify existing trusted-launch/start claim contracts and restart transitions |
| Recovery executor | INTEGRATION | cmd/internal/couchcore/recovery_resume.go; resume.go; cmd/internal/launcher/createflow.go | new coordinator + modified final launch gate, exact revalidation |
| Previous tty source union | PURE | cmd/internal/threadrecord/previous_session.go; cmd/internal/couchcore/peek_selection.go | modify validation/selection with legacy compatibility |
| Capture preservation/publication | INTEGRATION | cmd/internal/couchcore/previous_session.go and new recovery_capture.go | extend shared publisher truthfully for proven absence; reuse artifact retention |
| Liveness/action consumers | INTEGRATION | couchcore/actionableinventory.go, sessionevidence.go, detachedsessions.go, resume_route.go, actor_actions.go; couchtty/menu_actions.go, menu_reattach.go, console_reattach.go | modify all to consume the same attachable-agent observation |
| Recovery UI and orientation | INTEGRATION / PURE builder | couchtty/menu_recovery.go; orientation/recovery.go; launcher/launch_args_policy.go | new UI/builder; extend existing delivery validation only for recovery purpose |

Full paths above abbreviated only in the last rows: all packages are beneath cmd/internal/. Existing native binding resolver, generic native argument composition, thread hold, lifecycle cleanup, reader leases and orientation delivery transition remain reused; no parallel proof resolver or transport. Any actual symbol/file split must update this inventory through Revisions before boundary review.

#### Revised implementation boundaries and acceptance

- **M1 — Prevent split terminal incarnations.** Red tests for same-name old-live/new-empty conflict, socket ownership mismatch, bare-attach resurrection, two cooperating Pair launches, PID reuse and death between observation/effect. Scratch conformance must prove selected config fails missing/exited attachment with zero new server/panes, including preexisting serialized data. Add the shared lock at all Pair effect paths; wire all liveness consumers. Existing healthy detached attachment must still work. Publish no speculative reconnect when the old socket was clobbered: refuse without killing or creating. Review through sdlc milestone-close.
- **M2 — Recover native conversation and tty context.** Freeze reproducible calibration and holdout results before implementing auto selection. Add pure parser/score/policy tests and stateful collector tests from the first draft, then typed durable authority across both Couch and Pair gates, capture source union and one-shot resumed/fresh orientation. Tests span restart at every claim/ledger/registration boundary, candidate/capture mutation, contested server appearance, cancel, duplicate/late completion, lease retention and orientation indeterminacy. Test old record decoding/new record round-trip, and an old reader's explicit unsupported version behavior where controllable.
- Full integration tests must traverse the actual launch route and assert process/server counts, session IDs and unchanged old evidence. Mutation tests remove final ownership/lock checks, source exclusions, target revalidation, durable intent retention, and no-resend handling. Each needs a failing regression.
- Operating budgets from the initial draft remain provisional measurement budgets; reject incomplete scans rather than guessing. No retained full native transcript copies; bounded intent/evidence references follow current claim/history/archive cleanup. Explicit capture metadata growth is constant per capture; reader leases remain scoped.
- Update README/atlas at each milestone. Run targeted/race/build then full suite before close. TL/operator acceptance first observes pair:6's conflict without changing it; actual reconnect/native recovery requires their live authorization. Demonstrate automatic correct choice on the calibrated holdout, a close-call confirmation, and a scratch fresh fallback with tty context.

#### Approval status

No code has been written. TL requested the configuration scratch measurement and forbids touching pair:6. Numeric inference policy and safe no-resurrection config are still measurements in progress; submit their results with this revised design. A Zellij fork requires separate operator approval and is not the default solution.

### 2026-10-13 09:16 PDT — Offline calibration proposal

This replaces the earlier pending numerical policy, subject to TL approval and regression fixtures. Slots 2/5 select the proposed values; slot 6 is held out from numerical selection, but earlier slot-6 output was visible, so this is not a blind evaluation. Candidate text is each of the last three complete assistant messages with at least 16 normalized words; take maximum distinct eight-word-window coverage. Tty is existing VT-rendered 100-history-row window plus viewport. Normalize whitespace, Markdown emphasis/backticks and terminal bullet decoration only.

| Variation | pair:2 training | pair:5 training | pair:6 evaluation |
|---|---:|---:|---:|
| Original | 100% | 100% | 100% |
| Last 25% tty rows | 100% | 0% | 100% |
| Last 50%/75% tty rows | 100% | 100% | 100% |
| Rewrap 40/80 columns | 100% | 100% | 100% |
| Last 75% assistant words | 14/21 = 66.67% | 19/27 = 70.37% | 14/21 = 66.67% |
| Last 50% assistant words | 33.33% | 37.04% | 33.33% |

All actual nonwinning candidates scored zero in the 45-transcript pool. Synthetic copied subagent summaries tied at 100%; two synthetic interactive roots sharing visible text tied at 72.41%. Therefore the initial proposed automatic floor is coverage >= 2/3 AND >= 14 matching distinct eight-word windows; additionally no competing eligible root may have matching windows. Any overlapping competitor is a close call until broader calibration establishes a narrower margin. Candidates excluded by explicit subagent/exec provenance do not become eligible merely through copied text. Unknown provenance remains uncertain and cannot silently disappear when it competes. Low/absent evidence offers fresh fallback only after complete enumeration and resolved terminal absence; partial inspection offers retry.

These values encode the explicit acceptance choice “retain correct recovery with 75% of the measured assistant text,” not a universal probability claim. Some clipped positives intentionally abstain. Tests reproduce those scores with sanitized text preserving overlap geometry. The metrics must not change based on the selected answer at runtime. Temporary overlay measurement PASS (0.946s); no repo production or live-slot mutation.

### 2026-10-13 09:18 PDT — Capture metadata owner resolution

The archive producer is `launcher/osruntime.go`'s preserve-scrollback path, which calls `storagegc.Coordinator.PublishCaptureMetadata` after payload syncs under its lease; the pure schema/decoder is `artifactpath/capture_metadata.go`. These are the concrete integration/schema owners for the active map's capture row. Current metadata describes completed archived raw/events, not the live file producer. Add producer attribution at the existing wrapper capture-open/launch-declaration seam and carry it into this archive publisher only when validated against that file generation. A historical capture without attribution stays historical/inferred; no migration invents ordinal 18. Tests include multiple ordinal-zero-offset launches, append versus inode replacement, failure between raw/events sync and metadata publication, and archive after wrapper restart. Runtime scans must never edit source transcripts or live captures to attach inferred provenance.

### 2026-10-13 09:21 PDT — Approved no-fork policy and complete recovery action

TL receipt baf2f67b approves the no-fork option and requires one actionable legacy projection. This is authoritative over the pending-fork/config alternatives above.

**Scratch conformance:** installed Zellij 0.45.1 on macOS. For each of default, `session_serialization false`, `disable_session_metadata true`, and both: missing session refuses; existing live session attaches to the same server; an OLD serialized dead session resurrects into a new server and held command pane. New sessions created with disabling settings leave no resumable layout in these probes and subsequent attach refuses. Fresh XDG_CACHE_HOME or ZELLIJ_CACHE_DIR did not isolate the platform cache. All scratch servers were cleaned; no Pair session was touched. Raw measurement outputs: /tmp/p528-m-m4z6lkud/results.json, /tmp/p528-m-tmbp5gfy/results.json, /tmp/p528-m-yzis1k7a/results.json. Preserve equivalent sanitized fixtures/conformance tests during M1; temporary probe outputs are evidence, not a shipping dependency.

**Shipping policy:** disable serialization for all newly created Pair servers. Extend the existing session-owner metadata (`artifactpath/session_owner.go`, its writer and reader in launcher) with a versioned launch-policy attestation bound to server PID/start identity, exact socket/name, launch nonce and effective config digest. It proves this server was created with the measured nonserializing policy; an absent witness is legacy, never inferred from today's config file. Use both the per-thread hold and a process-wide OS-backed lock keyed by canonical session/socket identity around all Pair launch/attach paths, so aliases/different supervisors cannot race the same terminal. Hold only through handoff completion/verification, not the lifetime of the interactive client. Do not fork Zellij.

Warm attachment requires coherent exact server/socket/agent ownership, the attested policy and absence of a known resurrectable legacy layout for that session. Recheck after acquiring the lock and at handoff; attach failures never retry as create. A failed or lost connection returns a recovery decision. Explicit cold create requires positive absence of all previous wrapper/agent/server incarnations, unique session admission and the same lock. The isolation claim covers cooperating Pair processes; external unowned interference is detected and refused, not treated as a permission to remove resources.

**Visible action:** “This session was saved by an older Pair and cannot be attached safely” leads to **Relaunch**. The same action handles a split/contested terminal with an explanation. It opens the recovery preview, selects proven/calibrated native identity or asks on a close call, and presents the exact tty-context source. It never calls resurrection-capable attach for that legacy terminal. Fresh fallback is explicit in the preview when no native target is defensible.

A Relaunch involving an existing agent first shows that it will retire the old owned terminal while preserving conversation/capture. Existing settled/busy checks, exact wrapper registration, thread revision and process-generation checks apply. Preserve capture BEFORE stopping. Reuse Pair lifecycle and `launcher/session_quiescence.go` only after authorizing each exact owned server/agent; name-only DeleteSessionRecord is not authority for a contested set. Extend its seam to consume the authorized snapshot and refuse newly appearing/unattributable servers. An old live wrapper and the empty replacement may be retired only when both are proved to belong to this thread and the operator authorized that preview. If ownership cannot be established, the Relaunch sheet names the unresolved identity and routes the operator to the specific recovery prerequisite; it does not spawn, silently kill, or turn “retry” into a destructive bypass. Test this path through completion for a fully attributable two-incarnation fixture, plus refusal for a foreign/unattributable server. Pair:6's actual process retirement remains operator-controlled.

This retirement is distinct from the optional one-time machine cleanup below. The product must be able to refuse unsafe attach and offer Relaunch without a global cleanup prerequisite. New terminal policy is attested on the replacement; revalidation and persisted RecoveryIntent survive interruption before it becomes usable. User-visible success requires the selected native identity, an attachable agent and delivery outcome for the “where we were” tty context; failed/indeterminate orientation is visible and is not auto-replayed.

**Optional machine cleanup proposal — approval not granted:** after the operator has safely retired all owners of pair:6's old session and evidence is retained, remove only these inspected legacy serialization files (not the directory recursively):

- `/Users/xianxu/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1/session_info/📁1-218/session-layout.kdl` (4,094 bytes at inspection).
- `/Users/xianxu/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1/session_info/📁1-218/session-metadata.kdl` (3,607 bytes at inspection).

Before any later approved execution, reread identities/content stamps and show changed paths again; a live old server can rewrite them. No socket, raw/events, native transcript or other session path belongs in this cleanup. This plan authorizes no deletion. TL obtains the operator's separate approval. Broader cleanup would require its own enumerated path list, not a wildcard.

The upstream explanation matches the scratch result: v0.45.1 selects resurrection when a saved layout exists and no active session is found, independently of the create flag ([commands.rs, attach branch](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L719-L755)). We are avoiding that legacy branch rather than patching it.

**Final approval request:** approve M1's policy attestation/locking/shared ownership and actionable legacy Relaunch; M2's measured conservative automatic policy with close-call confirmation, durable exact recovery intent, truthful previous capture source and one-shot native/fresh tty context. Approval to implement is still required; the approved no-fork direction alone did not authorize code. Next gate after approval is `sdlc change-code --issue 528`, not live pair:6 recovery.

### 2026-10-13 09:27 PDT — Approved design; expanded calibration and correction action

TL receipt bea98157-2f91-47c4-8cdc-554d62d5f4a4 approves design commit 4cedc4baa with two requirements before change-code. This supersedes the pending approval paragraph and the three-pair automatic threshold above; it does not authorize pair:6 recovery or legacy cache deletion.

1. Build the offline labelled corpus from every retained thread with a PROVEN native binding in registry/park history. Independently join each capture to its producing conversation before scoring. Score all same-cwd candidates, including excluded execution modes as diagnostic negatives; report eligible competitors separately. Record corpus size, per-agent counts, exclusions, right score, best-wrong score and margin in the issue Log. Freeze the selection threshold from these results, not from the three preliminary examples. A label must not be selected by overlap itself. Missing historical capture/launch association is a corpus limitation, not permission to fabricate proof; report any inability to satisfy the requested corpus before implementation. Native events newer than the retained capture must not contaminate its tail comparison.

2. After an automatic inferred resume succeeds, present a notice naming the exact selected session and measured evidence, for example “resumed <session>: last active 07:54, 92% tail match”. Its single correction action is “wrong conversation? → start fresh from previous screen”. Bind this action to the original predecessor capture and branch/issue carried by RecoveryIntent, not to whatever session later becomes previous. Reuse the existing recovery preview/executor and normal busy, current-incarnation, revision and capture checks; the action cannot terminate a newer or busy conversation through a stale notice. Preserve the capture through notice/action lifetime using the existing retention owner. Fresh orientation uses that original screen once; indeterminate delivery remains visible and is not replayed.

Tests extend M2 with successful inferred-resume evidence, action-to-original-capture continuity after the new agent emits output, stale notice after another launch, busy refusal, capture replacement, restart reconstruction and no duplicate orientation. The visible-state table and both-gate revalidation remain unchanged.

### 2026-10-13 09:30 PDT — Shared ranked alternatives replace the single correction action

Verified TL receipt 4c8b05d4-62a3-4fd7-b8fa-d6fc1609aa38 replaces the single undo/correction action above. Recovery actions expose “resume (default)”, followed by “resume (option 1)” through “resume (option 4)” where available, each with agent, native session ID, last active time, tail match percentage and a one-line authored-text excerpt. A clear winner still resumes automatically with the evidence notice. Close candidates remain available before and after a choice, so the operator can attach, inspect and try another. Fresh-from-screen remains the ladder fallback, not the sole correction action.

The same pure RecoveryPreview owns the ordered evidence and eligibility consumed by UI and CLI (ARCH-DRY). Add read-only `couch --resume-candidates repo:N [--json]` and explicit `couch --resume repo:N --candidate K`. Candidate keys are stable, opaque IDs tied to the evidence snapshot and native ID, not freshly recomputed rank numbers; display labels provide human option numbers while CLI listing includes the corresponding key. Thus “option 3” from a prior listing cannot silently select a different transcript after rescoring. Limit alternatives to four after the default; report truncation and incomplete scans. CLI and UI both pass through the same trusted RecoveryAuthority and final gates; explicit selection is authorization to resume that candidate, not to bypass busy/ownership checks. No source/parent exclusion becomes eligible merely because it scores well.

Persist the bounded original evidence snapshot/capture reference with RecoveryIntent so alternatives remain available after automatic resume and restart. Re-selecting is a new explicit recovery action: revalidate selected native generation, source capture and current slot ownership; apply normal settled/busy protections and preserve current evidence before replacement. Orientation continues to reference the original recovery screen. A changed or expired candidate asks for a refreshed preview rather than reinterpreting a key. Tests cover UI/JSON parity, ordered default plus alternatives, stale rank/key, candidate mutation, post-resume alternatives, busy refusal, repeated selections, restart, and original-screen retention. CLI wiring belongs to existing couch command parsing/dispatch; exact files will be located before implementation and recorded in the owner map.

**Read-only socket check:** at 09:29 PDT, original Zellij PID 41843 (started Oct 12 20:13:13) and replacement PID 9682 (Oct 13 07:55:48) both remain alive and name the same socket path in argv and lsof. The current filesystem entry `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/zellij-501/contract_version_1/📁1-218` is a socket, device 16777232, inode 466359361, born/modified Oct 13 07:55:48.344482 PDT. lsof identifies distinct socket objects: original fd5 device 0xca2ba1a935234282, replacement fd5 device 0xe42a2a8d498cad86. The pathname currently exists as a later entry matching the replacement's launch, not the original creation. This supports replacement of the original pathname; the present snapshot cannot identify who removed it or prove macOS temporary cleanup caused the initial loss. No socket connection, unlink, attach or process mutation was performed.

### 2026-10-13 09:32 PDT — Calibration evidence prerequisite unresolved

The expanded audit finds no certified historical capture/conversation joins across 92 retained threads, despite 49 owners having both native binding rows and capture manifests. Exact counts and exclusions are in the issue Log. Right/best-wrong margins and a broader automatic threshold cannot be reported honestly from this corpus. Do not enter change-code by treating the old three examples or an empty scorer run as satisfying the new requirement. Report the limitation to TL: obtain independently labelled historical examples or newly witnessed pairs, or ask whether M1 may proceed while M2 automatic selection remains gated on calibration. The requested candidate alternatives and read-only socket measurement have been folded in; live pair:6 and cleanup remain untouched.

### 2026-10-13 09:31 PDT — Implementation authorized; automatic choice disabled

Verified TL receipt 6f74a1a9 supersedes the calibration prerequisite block: enter change-code for M1 now. M1 includes no-fork launch policy, lock, shared ownership checks, legacy/contested Relaunch projection, and forward capture producer witnessing. M2 builds ranked evidence, menu alternatives and CLI selection, but **automatic inference is disabled**. The default label is “resume (best candidate)”, an explicit operator/TL action. Proven native binding remains ordinary resume. No historical hand-labelling. Earlier automatic-success and numeric-threshold proposals are future activation criteria, not current shipping behavior.

Move producer witnessing into M1: bind raw/events generation to thread scope/tag, producing launch ordinal and nonce at the existing capture-open/launch-declaration seam; preserve that association in archive metadata. A transcript pair becomes certified only when the same ordinal gains native binding proof. A provisional launch or a relaunch by itself is not proof. Reexecutions/appends must record boundaries or refuse certification if multiple native identities share an undelimited capture; never stamp an entire historical file with the latest nonce. Tests cover capture replacement, append/reexec, archive, interrupted publication, and absence of a native binding.

Proposed later activation floor: at least 20 independently witnessed pairs from distinct producing launches for each agent/parser version enabled for automatic selection, across at least five threads, with all same-cwd competitors scored. Keep five pairs held out from threshold selection. Record corpus hashes/counts, right/best-wrong score and margin, derived score/support/margin floors and held-out results. Require zero wrong automatic choices on that recorded corpus; abstention remains permitted. Twenty is a minimum evidence floor, not a confidence claim, and reaching it never self-enables behavior. Enable only in a reviewed change carrying the recorded calibration policy; unsupported agents stay explicit. This issue can ship the explicit ladder without waiting for future usage to produce the corpus.

Current done-when: explicit ranked correct pair:6 candidate with evidence (live recovery operator-owned), safe alternatives/fresh tty fallback, coherent liveness and no second server, forward certified provenance when native proof exists, and automatic selection demonstrably off. Add regressions that even a unique 100% match requires explicit choice and that missing/invalid calibration cannot enable inference.

### 2026-10-13 09:36 PDT — Plan gate PQ-1 through PQ-4: concrete ownership contracts

This section is the authoritative implementation/test contract; preceding case inventories are historical rationale. No production edits preceded the gate. Pair:6 is no longer a test bed (verified TL receipt 75789ac3); all conflict/legacy acceptance uses isolated scratch setups.

**PQ-1 / ARCH-ORDER — terminal admission lock completion.** Add `TerminalHandoff` coordination in launcher/session_handoff.go and an OS adapter in session_handoff_os.go. The runtime seam separates Start, ObserveAdmission, and Wait; the existing blocking LaunchSession/AttachSession public behavior remains start+observe+wait. Acquire the canonical socket-keyed OS lease before the final owner observation. After starting the client, a bounded watcher observes the exact expected server generation and usable Pair pane; create also requires this launch's pane-birth witness and policy publication. Attach requires the same previously observed attested server and socket identity after client start. This event is `AdmissionVerified`: it permits release of the launch lock, not a claim that the user saw or accepted the conversation. Keep waiting for client exit outside the lock. Do not release just because cmd.Start succeeded.

The observing owner is the launcher that holds the lease, with a 10-second admission bound and cancellation-aware polling (existing birth bound). Unknown/timeout/cancel stops and reaps only its own client, joins the watcher, then closes the lease; it never kills the server or retries create. A surviving server without verified policy/ownership is unsafe legacy/unknown on the next attempt, never assumed absent. Launcher death releases the OS descriptor; the next holder reobserves server/agent existence and refuses incomplete evidence. A stateful fake controls client start, server birth, pane witness, observation, deadline, death and exit using channels; tests prove lease release ordering without sleeps. Cold admission still rejects every extant expected-name server, including an unregistered orphan. Canonical socket-keyed lease path is per-user Pair cache, hashed canonical socket directory plus name, stable inode, never unlinked for unlock. Pair scope aliases converge on this key.

**PQ-2 / ARCH-DRY, ARCH-SECURE — new policy attestation, not an existing metadata extension.** `artifactpath/session_owner.go` decodes owner paths only; the earlier metadata reuse claim was wrong. Define a new pure `SessionLaunchPolicy` in artifactpath/session_policy.go and paths through the existing PairCache constructor. Store one bounded version-1 JSON record at Pair cache session-policy/<hash-of-canonical-socket>.json, containing exact socket pathname and file device/inode, session name, server PID/start identity, owner scope/tag, launch nonce, disabled-serialization policy version and effective config digest. The launcher OS adapter publishes atomically under the admission lease after new-server/pane identity verification and before AdmissionVerified. Unknown fields/versions or mismatched generation refuse warm attach. Absence is legacy.

Use explicit Zellij launch options `options --session-serialization false` (local 0.45.1 help exposes this flag), in addition to repository config, so mutable user config cannot negate the launch policy. Scratch conformance must verify the actual composed argv and failure of post-exit attach before attesting this policy version. Effective config digest is audit evidence, not a retroactive claim about legacy launches. Warm validation binds the attestation to current exact server/socket/pane ownership and absence of legacy saved-layout residue. New creation atomically replaces the old record only after positive prior-server absence. Lifecycle cleanup removes only the matching policy generation after server absence; failed/uncertain cleanup retains a harmless bounded stale record that never validates for a new PID/start/socket. Add the artifact to artifactpath manifest and the existing cache cleanup ownership rather than a second sweeper. No legacy layout deletion.

**PQ-3 / ARCH-ORDER, ARCH-FUNERAL — post-start recovery ownership.** Define optional bounded `Record.RecoveryContext` in the new threadrecord schema, alongside StartClaim.RecoveryIntent. It holds the original screen descriptor, up to five candidate identity/evidence summaries, selected native ID, evidence snapshot digest, context generation and expiry. `ApplyStartEvent` atomically transfers the intent to RecoveryContext before clearing Start on both registered and recovered-unknown transitions. Unknown does not imply successful resume; it merely preserves the decision for reconciliation. An alternative selection carries that context forward; an unrelated fresh start/switch supersedes it. Thread deletion removes it with the record.

Expiry equals the source capture's existing CapturedAt + CaptureRetentionPeriod; do not invent a permanent GC pin. Candidate actions disappear at expiry or verified missing/changed capture. Until expiry the existing age rule already retains the archive; preview and execution acquire existing reader leases and revalidate identity, closing the collection race. A context referencing an already expired source cannot be created. UI/CLI say evidence expired and offer a fresh preview when no source is available; they never substitute a different screen. The original context is bounded and can survive Couch restarts without keeping a reader process alive.

Before execution, listing is read-only and creates no durable snapshot store. `EncodeRecoveryCandidateKey` encodes version, exact native ID and source/evidence digest; `ValidateRecoveryCandidateKey` decodes with a strict size bound, recomputes evidence from the explicitly identified candidate and capture, and refuses stale identity. The key is an untrusted selector, never authorization by itself: current local request admission, exact slot scope and busy/ownership gates apply. Option numbers are presentation only. Extend couchcmd/cli.go and messages.go, couchmessage/protocol.go, couchtty/message_service.go and slot_operations.go, and couchcore/operationdispatch.go. Candidate listing is an authenticated read route, not IsSlotOperation; Recover and automatic warm reattachment cannot infer without explicit selection.

**Forward witness seam correction.** wrapcmd/wrap.go opens both raw/events with O_TRUNC. Publish a versioned first events record from `PublishCaptureWitness` after both opens and before accepting child output, naming exact file identities, unique capture generation, owner, ordinal and nonce. Existing native fresh reexec advances ordinal but retains nonce; every open advances capture generation. `DecodeCaptureWitness` rejects missing/conflicting/malformed authority. No append segmentation is required by current behavior. The archive publisher validates and carries this witness into CaptureMetadata version 2 while preserving strict v1 reads; capture identity and native binding proof remain separate until joined by ordinal.

**PQ-4 / ARCH-PURE — active function-level test strategy.**

| Function / owner | Adversarial class and mechanical guard |
|---|---|
| ClassifySessionOwner, ClassifyServers (launcher) | Conflicting/incomplete process and pane observations; pure tables ensure only coherent evidence yields usability. |
| CoordinateTerminalHandoff (launcher/session_handoff.go) | Adversarial lifecycle ordering; channel-controlled stateful fake proves exclusive observe/start/admit and bounded cancellation/reap. |
| AcquireTerminalLease (launcher/session_handoff_os.go) | Concurrent processes and owner death; subprocess conformance proves stable-key exclusion and OS release. |
| DecodeSessionLaunchPolicy, ValidateSessionLaunchPolicy (artifactpath/session_policy.go) | Arbitrary/mismatched serialized authority; fuzz strict size/version/identity validation. |
| PublishSessionLaunchPolicy (launcher adapter) | Interrupted writes and generation replacement; temporary-filesystem integration proves atomic publication under lease. |
| DecodeCaptureWitness, ValidateCaptureWitness (artifactpath) | Malformed/mixed capture provenance; fuzz parser and pure identity/generation matching. |
| PublishCaptureWitness, PublishCaptureMetadata (wrapcmd/storagegc) | Partial IO and overwritten generations; real temporary-file tests prove no false certification. |
| ParseRecoveryCodex, ParseRecoveryClaude, ParseRecoveryGrok (sessioninventory) | Real-shaped malformed/foreign/subagent streams; fuzz byte parsers and validate provenance without IO mocks. |
| ScoreRecoveryTail, RankRecoveryCandidates (sessioninventory) | Ambiguous/copied/clipped text; pure value tests and reproducible offline scorer preserve deterministic score/margin and explicit-only decisions. |
| CollectRecoveryEvidence (sessioninventory adapter) | Incomplete/moving/overbudget sources; stateful filesystem fake proves bounded reads and explicit uncertainty. |
| EncodeRecoveryCandidateKey, ValidateRecoveryCandidateKey (couchcore) | Untrusted/stale/cross-slot selectors; fuzz bounded codec and exact recomputed evidence binding. |
| ApplyStartEvent, ValidateRecoveryContext (couchcore/threadrecord) | Interrupted/reordered transitions; pure event sequences prove transfer before claim clear, explicit authority and bounded expiry. |
| ExecuteRecoveryResume (couchcore) and launcher final gate | Candidate/owner mutation between decisions and effects; stateful launch fake proves both-gate revalidation and no inferred automatic start. |
| BuildRecoveryOrientation (orientation) | Untrusted text/paths and repeat delivery events; pure builder plus existing delivery reducer proves bounded literal context and no replay. |
| ParseCLI, ValidateRequest, recovery menu reducer | Invalid selectors and stale asynchronous previews; pure tests plus service fake prove read/execute separation and UI/CLI parity. |

Existing targeted/race/full-suite and scratch conformance gates remain; numerical auto-policy is intentionally not shipped.


### 2026-10-13 11:10 — M1 boundary REWORK: uncertainty, agent generation and bounded metadata

The first M1 boundary review returned REWORK (BR-1 through BR-4); this revision supersedes the earlier boolean usability and unbounded per-name lock lifecycle. ARCH-ORDER: TerminalSafety is a three-way observation (unknown, usable, unsafe). Inspection failures stay unknown through core projection and menu overlays; known missing/invalid attestation is unsafe. A previous successful attach cannot overwrite either verdict. Only positively usable evidence can be downgraded for a bound-agent mismatch.

Readiness carries the exact agent PID/start identity in addition to the server generation and launch nonce. Legacy readiness is readable but cannot attest a safe attachment. The producing wrapper pins the child identity for its readiness updates; a legitimate native reexec publishes its own new child identity. This avoids confusing PID reuse with continuity without permanently pinning the server policy to the first child.

ARCH-SECURE: all policy/readiness admission reads share a bounded regular-file reader. It checks type/size before open, uses nonblocking/no-follow flags, validates the opened inode and checks for replacement after bounded reading. Context cancellation and OS inspection failures remain uncertainty; malformed known metadata never authorizes admission.

ARCH-FUNERAL: admission leases use 256 permanent stripes under Pair cache session-locks-v2/HH.lock, selected by the first byte of SHA256(canonical socket). Same-key aliases share a stripe; unrelated collisions only serialize. Lock inodes are never unlinked, and total lock artifacts are bounded at 256. Exact socket policy paths remain unchanged and retire after verified absence under the lease. This new lock namespace cannot coordinate with the prior experimental layout; M1 has not been deployed/staged, so no deployed-format migration or cleanup is required. No legacy layouts are deleted.

Actual M1 function owners (correction to the earlier test table): AcquireTerminalLease is in launcher/session_lease.go (integration); readBoundedRegularMetadata is new in launcher/metadata_read.go (integration); ObserveTerminalUsability/ObserveTerminalSafety are in launcher/session_handoff_os.go (integration); AgentBelongsToServer is in launcher/terminal_agent.go (pure); terminalObservationError is in launcher/session_handoff_os.go (pure); agentUsabilityReason is in couchcore/agent_usability.go (pure); overlayPassRow is in couchtty/menu_reattach.go (pure). New/modified tests cover each review defect; no automatic inference is added.


### 2026-10-13 11:39 — BR-5 shared persisted-input rule and consumer sweep

Round 2 disposed BR-1 through BR-4 but found two more members of the bounded-input family. ARCH-PURPOSE/ARCH-DRY: move the IO owner from launcher to existing osfs, as ReadRegularMetadata(ctx,path,maxBytes). Static type/size violations have a typed InvalidMetadataError; inspection/mutation/cancellation stays distinct. All new M1 persisted byte readers now use this owner, with whole-file size limits, regular-file/no-follow/nonblocking opens, opened-inode and post-read revalidation, and context checks. No prefix reader or permissive truncation is introduced.

| Added consumer | Read bound / cancellation owner | Consumer regression |
|---|---|---|
| readSessionPolicy: usability and policy retirement | existing MaxSessionLaunchPolicyBytes; projection currently has no context parameter and uses Background | policy FIFO and oversized file |
| observePolicyAgentWithTable: projected/current readiness | 8 KiB; snapshot API uses Background | readiness FIFO and oversized file |
| terminalOSAttempt.Check: retained readiness across agents | 8 KiB; handoff context | shared reader cases and existing cold admission checks |
| terminalOSAttempt.observeAdmission: new readiness | 8 KiB; handoff context | shared reader cases and real cold/warm conformance |
| readTerminalConfig: effective config digest during admission | 1 MiB; handoff context | config FIFO, oversized config; prior implementation fails both |
| readCaptureWitnessEvents: source preservation and archived publication recheck | existing MaxCaptureWitnessEventsBytes (8 MiB), whole sidecar; explicit context (ParkScrollback remains a synchronous context-free API and supplies Background) | real consumer FIFO, oversized file rejected before decode, cancellation |

The sweep checked every production read added by the M1 branch diff. Existing readStateJSON, reused for capturePublicationTime, already checks regular files, opens nonblocking/no-follow, bounds at 1 MiB and rejects trailing JSON. Existing transferFile is a payload copy, not a new metadata reader; witnessed copies add no-follow/nonblocking and exact regular-file identity checks, while legacy payload copying remains its previous contract. CaptureIdentity/socket Lstat calls inspect metadata without opening byte streams; capture writers/lease descriptors are not input readers. No other new persisted byte reader was found.

The shared helper's direct tests cover FIFO, directory, symlink, oversized input, exact bound, missing file, cancellation and inode replacement. Consumer tests are required in addition to helper tests so a future direct os.ReadFile substitution cannot bypass the rule unnoticed. Full OS integration runs remain required in the development checkout, where process inspection is available (review sandbox denied /bin/ps).

### 2026-10-13 — M2 implementation refinements

TL receipt b15bc8c6 authorizes M2 after M1 SHIP. Automatic selection remains off. Collector IO now requires a bounded contextual Runtime extension: eager ListFiles and blocking ReadAt cannot enforce the approved bounds. The collector charges tty replay bytes against the combined 256 MiB limit and caps retained authored messages at 256 as well as 256 KiB; the count bound prevents tiny messages from producing unbounded object overhead.

ARCH-ORDER: launch ledger records contain ordinals but no wall clock. Capture witness v2 adds OpenedAt, recorded before child output, to anchor a conservative producer cutoff without borrowing the latest recovery launch. Version 1 remains readable but cannot supply that timestamp. Archive publication time controls existing retention; original tty last activity is retained separately so copying does not change ranking or include later native messages.

ARCH-DRY: the recovery snapshot itself supplies the exact archived tty reference for previous-screen reads, independently of a verified park receipt. A stopped current capture is copied through the existing Pair capture publisher before claim; copy identity and original native selection are revalidated. Menu and CLI share bounded pure RecoveryPreview; snapshot keys identify native identity plus evidence digest, never display rank.

ARCH-ORDER refinement: when recovery must retire a source, persist the exact archived screen and selected recovery context with the record revision check before park/retirement. StartClaim subsequently carries that same intent. An interruption before the claim therefore retains the original screen; it cannot switch evidence to a later empty launch. Final snapshot-key validation discounts only this operation's own recorded revision transitions.

ARCH-ORDER refinement: DeleteStart rollback transfers RecoveryIntent to retained RecoveryContext before removing the creating incarnation, using the same helper as Registered/RecoveredUnknown. This applies with and without a verified park. Contextual VT replay reuses the shared bounded regular-file reader and checks cancellation between feed chunks and rows; no timeout goroutine continues reading after cancellation. Native-owner observations are rechecked before effects and at Pair admission; they reject observed competing ownership but do not claim atomic exclusion against an unrelated external client starting the same native conversation concurrently.

### 2026-10-13 — Authoritative M2 implementation owner reconciliation

This table supersedes **all earlier proposed M2 core-concept/function-owner and test-strategy rows**, including references to nonexistent `recovery_runtime.go` and `recovery_codex.go`. Baseline for new/modified/reused is shipped M1 `3ef0ebf54`; the comparison includes the current tracked diff and new working-tree files. Paths below are relative to `cmd/internal/`. New functions inside modified files are identified separately. This is an implementation map, not a new authorization or a claim that every test strategy has passed. M1 ownership remains governed by its prior reconciliations. ARCH-DRY: shared parsers, lifecycle transitions, publication and admission remain their existing owners.

| Actual concept / functions | Class | Actual owner and change from M1 | Verification seam |
|---|---|---|---|
| Native candidate parsing: `ParseRecoveryCandidate`, streaming `recoveryParser` | PURE | **New** `sessioninventory/recovery_candidate.go`; agent-specific parsing is in this file, not separate recovery_codex/runtime files | Colocated fixtures for source/root eligibility, cwd, message tails, timestamps, unsupported/ambiguous modes and bounded retained messages |
| Tail evidence and ordering: `ScoreRecoveryTail`, `RankRecoveryCandidates` | PURE | **New** `sessioninventory/recovery_match.go` | Deterministic authored-message overlap, recency ordering, ties and wrong-source exclusions; scores never authorize automatic launch |
| Bounded collection and exact selected-artifact revalidation: `RecoveryRuntime`, `CollectRecoveryEvidence`, `RevalidateRecoveryCandidate` | INTEGRATION | **New** `sessioninventory/recovery_collect.go`; reuses existing Runtime, roots and JSONL framing | Stateful runtime tests for enumeration/byte/message/deadline limits, incomplete evidence, capture cutoff and before/after fingerprint changes; exact revalidation does no reranking |
| Physical native IO: `OSRuntime.VisitRecoveryFiles`, `ReadRecoveryAt`, `StatRecoveryFile`, `ResolveRecoveryCWD` | INTEGRATION | **New** `sessioninventory/recovery_os.go` and `recovery_filemeta_{darwin,linux,other}.go` | Temporary-file tests for rooted/no-follow access, bounded reads, cancellation, descriptor identity and replacement; unsupported metadata remains unavailable |
| Snapshot choice and opaque keys: `DecideSlotRecovery`, `EncodeRecoveryCandidateKey`, `ValidateRecoveryCandidateKey` | PURE | **New** `couchcore/recovery_preview.go`, `recovery_key.go` | At most five eligible native choices plus explicit fresh; stale revision/capture, expiry, incomplete evidence and key tampering refuse; clock-only refresh preserves keys |
| Preview/execution: `Couch.ResumeCandidates`, `ResumeCandidate`, `prepareRecoveryRetirement`, `confirmRecoveryAbsent` | INTEGRATION | **New** `couchcore/recovery_preview.go`, `recovery_resume.go`; **modified** Couch/operation declarations and dispatch | Stateful runner/store tests for zero-effect refusal, exact selection, busy/unknown source, post-workspace mutation, persist-before-park, successful second alternative using original capture, and orientation notices |
| Exact tty source, archive preservation and branch context: `OSRecoveryEvidenceReader.Read`, `Preserve`, `PrepareTerminal` | INTEGRATION | **New** `couchcore/recovery_evidence_os.go`; reuses capture paths, retention leases, metadata publisher and GitRunner | Real temporary current→archive→read cases, witness and legacy fresh-only paths, owner/path/identity mismatches, original activity retention and same-size archive-byte substitution refusal |
| Contextual tty replay: `RenderBoundedLinesContext`, `feedSegmentsContext`, `replayDataWithLimitsContext` | INTEGRATION read shell; PURE bounded replay | **New entry points in modified** `scrollbackcmd/bounded.go`, `scrollbackcmd/scrollbackcmd.go`; existing context-free entry points delegate; reuses `osfs.ReadRegularMetadata` | Existing full-read bounds plus cancellation before/during chunked VT replay and row materialization; no detached timeout worker |
| Capture producer timestamp and safe publisher copy: witness `OpenedAt`, `ParkScrollbackContext`, `transferFileContext` | PURE witness schema; INTEGRATION producer/copy | **Modified** `artifactpath/capture_witness.go`, `wrapcmd/capture_witness.go`, `launcher/osruntime.go`; new contextual publisher entry point, existing publisher retained | Version compatibility, raw/events generation association, FIFO/symlink/cancellation refusal, finite regular-file copy and source identity recheck; archived bytes independently hashed by recovery adapter |
| Durable bounded snapshot: `RecoveryCapture`, `RecoveryCandidate`, `RecoveryContext`, `RecoveryIntent`, validators/cloners | PURE | **New** `threadrecord/recovery.go`; **modified** `threadrecord/record.go` schema 3 and legacy migration, `couchcore/thread.go` conversion/clone and `slotrecovery.go` schema seam | Strict schema 3, legacy 1/2 migration, bounded fields/lists, timestamps, optional platform generation token, deep-copy and persisted roundtrip tests |
| Start claim transfer: `AdvanceStartTransaction`, `retainStartRecoveryContext`, `ThreadStore.DeleteStart` | PURE transition; INTEGRATION store rollback | **Modified** `couchcore/starttransaction.go`, `threadstore.go`; helper **new** | Recovery survives Registered, RecoveredUnknown and both exact-nonce/revision rollback branches; unrelated start supersedes context; no fabricated VerifiedPark |
| Pair second gate: `RecoveryAuthority`, `BuildCouchRecoveryLaunchProfile`, `validateRecoveryRecord`, `OSRuntime.ValidateRecoveryAuthority` | PURE profile/record validation; INTEGRATION durable/artifact checks | **New** `launcher/recovery_authority.go`, `recovery_authority_os.go`; **modified** `args.go`, `launch_args_policy.go`, `createflow.go` | Exact attempt/helper/intent and reread, expiry, capture metadata/identity and native artifact checks; stateful create-flow refusal before launch; ordinary resume retains its original binding requirement |
| Exact terminal retirement: `PrepareRecoveryTerminal`, `RetireRecoveryTerminal`, attribution and `recoveryTerminalOS` | PURE attribution/comparison; INTEGRATION observation/lease/retirement | **New** `launcher/recovery_terminal.go`, `recovery_terminal_os.go`; reuses terminal admission leases and quiescence | Stateful server observations for exact owner attribution, changed/unknown process or socket, lease ordering and no unauthorized retirement |
| Competing native owner check: `CheckRecoveryNativeOwner`, `checkRecoveryNativeOwners`; core `checkRecoveryNativeOwner` adapter | INTEGRATION snapshot/read shell; PURE ownership comparison | **New** `launcher/recovery_native_owner.go`, `couchcore/recovery_native_owner.go`, `recovery_native_owner_fake.go`; **modified** executor and Pair authority gate call sites | Injected process/ready observations cover competing owners, unknown observations and exact allowed source/helper; recheck before effects and Pair admission. This observation check does not promise atomic exclusion against unrelated external clients |
| One-shot context: `BuildRecoveryOrientation` | PURE | **New** `orientation/recovery.go`; **reused** existing request/delivery state machine | Bounded literal/quoted untrusted context, exact previous-read command, native/fresh wording, historical screen not operator authorization; no saved argv changes or replay after uncertain delivery |
| Previous-screen selection from retained recovery | PURE selection; INTEGRATION existing capture reader | **Modified** `couchcore/peek_selection.go`; existing previous capture reader **reused** | Exact recovery archive selected without inventing park evidence; current/previous reads and refusal of unavailable context |
| Menu/CLI/message surfaces | PURE menu reducers/formatting; INTEGRATION asynchronous dispatch | **New** `couchtty/menu_recovery.go`, `couchcmd/resume_candidates.go`; **modified** existing menu/console, CLI/run/slot operation and `couchmessage` protocol/transport owners | Shared preview/opaque selection, explicit choice even at 100%, stale response handling, CLI validation, protocol roundtrip and no read-side effects |

The native-owner adapter and contextual replay entries include the concurrently integrated audit fixes under their actual source names. Their tests must be included in the final integration run; this reconciliation does not replace that verification or the SDLC boundary review.

### 2026-10-13 — Boundary review before full-suite capacity

TL receipt 4b2480f6 directs starting the boundary review while two other full suites occupy the machine. The successful dry run requires no bypass. This supersedes the earlier sequencing prerequisite that live acceptance precede M2 code review: run the code boundary with precise targeted/build evidence and the #514 capture-performance failure disclosed. Full verification beyond the aborted prerequisite and TL-coordinated operator acceptance remain required before whole-issue close/landing. Do not label the unseen rest of the full suite passing.

### 2026-10-13 — M2 REWORK: exact live launch and primary recovery routing

BR-6 (ARCH-SECURE, ARCH-ORDER, ARCH-MOCK): a stable live wrapper is not proof that the newest tag-level ledger launch belongs to it. Every recovery ownership assertion must associate exact wrapper PID/start with an exact live descendant agent via the existing readiness PID/start, launch ordinal and nonce, then resolve only that ledger ordinal. Missing/contradictory association refuses. Sweep Couch preflight, after-preparation check, Pair admission, both source-exemption paths and the stateful fake; a newer same-owner launch must never hide an older surviving wrapper. No new durable metadata or latest-launch fallback.

BR-7 (ARCH-PURPOSE): the unsafe row's primary recovery action must be the shared evidence chooser. Enter, mouse primary activation, explicit Relaunch and equivalent unsafe-row restart routes must use that same preview; ordinary healthy relaunch keeps its existing behavior. Add reducer/effect regressions for every entry route, rather than testing only the separately appended candidates item.

### 2026-10-13 — BR-8: recovery-specific native conformance

The second M2 review disposed BR-6 and BR-7, but identified that installed-native conformance only exercises legacy inventory scanners. Extend the existing read-only conformance path through the production recovery parser for Claude and Codex, checking provenance, cwd, activity and authored tails; explicitly report unsupported agents (including Grok recovery) and absent samples rather than claiming coverage. Add a regression where recovery-only field drift leaves legacy scanning valid but fails recovery conformance. Keep bounded reads and redacted output. Run with native-contract changes and the existing monthly operator cadence (ARCH-MOCK); no native writes or slot/process manipulation.

### 2026-10-13 — Final verification and scratch acceptance sequencing

TL receipts 4e054fd4 and 6de1fd5b retire the manual load threshold: run `make -k test` through the machine test lock, then `sdlc stage --build`. TL runs scratch acceptance from the existing hub after the operator restarts Couch onto the stage; close only afterward. Do not mutate a working slot. The procedure below uses the normal data root and authentication, not an isolated supervisor whose ownership checks would reject unrelated host wrappers.

## Scratch acceptance script (TL executes after staging)

**Scope:** create a NEW disposable repository named `recovery-528-<unique>` under a fresh temporary directory, and start its `:0` through Couch's normal start form. Never use pair:6 or any existing working repository. The commands below are separate shell calls from the hub. `SCRATCH_SLOT` is the new displayed address; `SCRATCH_REPO` is its physical checkout path. Keep the operator's menu visible. No saved-layout deletion, default-parameter edits, copied thread records, or duplicate Zellij servers.

1. Create an empty Git repository and initial commit under a new temporary directory. Add that repository via Couch's normal start form, selecting Codex with the operator's working configuration. Record the exact displayed address and path. In its agent pane ask for a distinctive complete paragraph (at least 30 words), with no tool work. Wait for the turn to finish and the composer to be empty.
2. From the hub, inspect and retain the exact scratch actor and transcript coordinates:

```sh
couch --actors --json
couch --show "$SCRATCH_SLOT"
couch --peek "$SCRATCH_SLOT" --transcripts --json
```

Select exactly one `.Actors[]` by `.Binding.Slot`; retain `.Binding.Repository`, `.Binding.Scope`, `.Binding.Tag`, `.Binding.Session`, `.Binding.Nonce`, `.Binding.Agent`, `.Binding.PID`, `.Binding.Start`. PID/Start identify the wrapper, not the Couch helper. Verify Repository equals the physical Git common directory reported by `git -C "$SCRATCH_REPO" rev-parse --git-common-dir` (usually the new checkout's `.git`), not the checkout path itself. Copy only this actor object to a scratch evidence file. Do not infer tag or scope from the slot number.

### Proven binding and explicit inferred choices

Use the scratch row's **Park** action, confirm, and await completion. Do not press Alt+x in the switcher: that parks all live threads. Alt+x is safe only while focused inside this scratch actor pane. Verify the parked row, then:

```sh
couch --resume "$SCRATCH_SLOT" --json
couch --operation-status "$OPERATION_ID" --json
couch --peek "$SCRATCH_SLOT" --transcripts --json
```

The first command returns a direct receipt with `.ID`; status JSON wraps it in `.Operation`. Use that operation ID; read until terminal status, never blindly resend. Expect the same native conversation ID. Complete another distinctive paragraph turn and wait idle, then Park this scratch row again; startup output alone need not retain the prior paragraph. List evidence:

```sh
couch --resume-candidates "$SCRATCH_SLOT" --json
couch --resume-candidates "$SCRATCH_SLOT"
```

Expect `automatic:false`, the real scratch native ID among `candidates`, agent/activity/tail score/excerpt, and `fresh`. Candidate K means the opaque `.candidates[K].key`, **not** an integer. Copy a key from the latest preview, then:

```sh
couch --resume "$SCRATCH_SLOT" --candidate "$CANDIDATE_KEY" --json
couch --operation-status "$OPERATION_ID" --json
couch --peek "$SCRATCH_SLOT" --transcripts --json
couch --peek "$SCRATCH_SLOT" --previous --all
```

Expect the selected native ID, retained original screen, evidence notice and one orientation delivery (or an explicit pending/uncertain notice, never repeated paste). The chooser is available even with a proven binding, so this tests the inference executor without fabricating unproven ledger state. To show alternatives, use `/new` in the scratch Codex pane, complete a second distinctive turn in the same cwd, then park and list again. Do not create synthetic resumable IDs. A later capture epoch can exclude earlier conversations, so `/new` does not guarantee multiple alternatives; record the observed preview and never alter transcript timestamps to manufacture eligibility. Keep at most five native choices plus fresh; a zero-match alternative is still evidence, never an automatic choice. A previously copied key after evidence changes must refuse and request a fresh listing.

### Legacy and contested-policy projection

Start/resume the scratch row and wait idle. Refresh actor coordinates. Discover its exact policy by reading JSON files under `$HOME/.cache/pair/session-policy/` and selecting exactly one whose `scope`, `tag`, `name`, `nonce`, and `agent` match the scratch actor. Its `socket` must name this actor's session; confirm `server_pid` / `server_identity` against that scratch Zellij server. Retain the policy path and bytes outside the cache. Never select a file by filename proximity or globally rename the directory.

Use the canonical socket to derive the exact policy path and permanent terminal-lease stripe (artifactpath SessionPolicy/TerminalLease). Acquire that lease, then recheck the exact server PID/start, socket device/inode, scratch actor nonce and unchanged policy bytes before **backing up and withholding only this verified policy file**. Release the lease before any Couch action. Use the guarded fixture helper below; an unguarded shell rename is not the acceptance command. This is the legacy-policy fault injection; do not stop the wrapper or alter readiness. Observe:

```sh
couch --show "$SCRATCH_SLOT"
couch --resume-candidates "$SCRATCH_SLOT" --json
```

Expect `needs-relaunch`, not live. The operator opens this row and uses primary **Relaunch** (also test Enter, cancel, then its explicit Relaunch menu item). It must open the evidence chooser without starting a process. Select the current native key using the menu, or equivalently:

```sh
couch --resume "$SCRATCH_SLOT" --candidate "$CANDIDATE_KEY" --json
couch --operation-status "$OPERATION_ID" --json
```

Recovery should preserve the original screen, park the exact idle source, and start a new terminal generation. Retain the backup as evidence; do not restore old policy over the new generation. If cancelling before any operation, restore the backup only after rechecking that the same socket/server PID/start and actor nonce still exist and no replacement policy appeared.

For a **synthetic contested-generation attestation**, repeat on a newly healthy idle scratch generation, backing up the exact current policy. Under the same lease and exact-generation recheck, atomically replace only `server_identity` with a distinct nonempty test string, leaving every other field unchanged. Expect `needs-relaunch` and the same chooser/action path. This tests conflicting attestation, **not two physical servers**. Physical duplicate-server refusal remains covered by the controlled automated ownership/concurrency fixtures; do not clobber a live socket to simulate it. No launch-policy mutation is performed by ordinary Couch commands; these are deliberate scratch-only fault injections.

### No candidate and tty-oriented fresh fallback

Use a separate new scratch repository/slot, complete one Codex turn, then Park it normally. Peek supplies the current resolved native root, not an exhaustive history. Before this case, read-only enumerate every configured Codex session root (normally `~/.codex/sessions`) and collect all transcripts whose `session_meta.cwd` resolves to this exact new scratch checkout. Record these exact paths and native IDs. Unknown/unreadable scratch metadata stops this setup; never guess cwd from filenames. Confirm there are no live actors for that scratch slot and no still-running wrapper/agent for its captured scope/tag. Read each candidate transcript's `session_meta` and require its cwd to equal the physical scratch checkout. Move **only those verified scratch transcripts** into a backup directory outside all native scanner roots (record source/destination pairs; never change permissions or move a native root). Do not touch capture, ledger, thread record or another cwd's transcripts.

```sh
couch --resume-candidates "$SCRATCH_SLOT" --json
```

Require `incomplete:false`, an empty `candidates` list and a nonempty `fresh.key`. If evidence is incomplete, retain the reason and stop this case: incompleteness is not a successful no-candidate result. Copy `.fresh.key` from that preview:

```sh
couch --resume "$SCRATCH_SLOT" --candidate "$FRESH_KEY" --json
couch --operation-status "$OPERATION_ID" --json
couch --peek "$SCRATCH_SLOT" --previous --all
couch --peek "$SCRATCH_SLOT" --transcripts --json
```

Expect a new native conversation and orientation naming the exact previous screen, scratch branch and issue (none for an unclaimed scratch repo). Ask the fresh agent to read that screen only as historical context. Restore quarantined transcript files by rename only when their original paths are still absent; never overwrite a newly written file. Park the scratch row afterward. Keep scratch evidence until acceptance is recorded; cleanup is separate.

**Record for each case:** address, staged commit, old/new native IDs and terminal names, preview JSON, operation receipt/status, menu observation, orientation outcome. Report any refusal verbatim; do not force unknown/busy ownership. TL supplies acceptance or failure before #528 whole-issue close.

Acceptance-script audit: corrected show syntax (no --json), Git common-directory actor identity, operation receipt envelope and exhaustive same-cwd transcript enumeration; peek alone cannot enumerate earlier scratch conversations.

### Guarded policy fixture commands

The source is `workshop/plans/000528-policy-fixture.go.txt` (test tooling only, not a shipped command). Build from the reviewed checkout using an overlay, leaving repository source unchanged:

```sh
python3 - <<'PYOVERLAY'
import json
from pathlib import Path
repo = Path.cwd().resolve()
source = repo / 'workshop/plans/000528-policy-fixture.go.txt'
assert source.is_file()
Path('/tmp/pair528-policy-fixture-overlay.json').write_text(json.dumps({'Replace': {str(repo / 'cmd/pair528-policy-fixture/main.go'): str(source)}}))
PYOVERLAY
go build -overlay=/tmp/pair528-policy-fixture-overlay.json -o /tmp/pair528-policy-fixture ./cmd/pair528-policy-fixture
```

Save the complete output of the separate `couch --actors --json` call in a scratch file, then extract exactly one scratch `.Actors[]` object (for example, `jq --arg slot "$SCRATCH_SLOT" '.Actors | map(select(.Binding.Slot == $slot)) | if length == 1 then .[0] else error("need one exact scratch actor") end' actors.json > "$ACTOR_JSON"`). Do not wrap the couch call in a pipe or redirect. Set `GLOBAL_PAIR_DATA_ROOT` to the existing hub's global Pair data root, normally `$HOME/.local/share/pair`, never its `repos/<scope>` child. Confirm this is the new idle scratch repo and no operation is running before each fault injection.

```sh
/tmp/pair528-policy-fixture -actor "$ACTOR_JSON" -repo "$SCRATCH_REPO" -data-root "$GLOBAL_PAIR_DATA_ROOT" -mode legacy -backup "$SCRATCH_REPO/policy-backup-1"
couch --show "$SCRATCH_SLOT"
couch --resume-candidates "$SCRATCH_SLOT" --json
```

After observing/cancelling the chooser, restore the unchanged generation with:

```sh
/tmp/pair528-policy-fixture -actor "$ACTOR_JSON" -repo "$SCRATCH_REPO" -data-root "$GLOBAL_PAIR_DATA_ROOT" -mode restore -backup "$SCRATCH_REPO/policy-backup-1"
```

For the synthetic attestation conflict, use a fresh backup directory and `-mode conflict` instead of `legacy`. Restore with its corresponding backup only if cancelling without recovery. After a successful recovery, the old-generation restore must refuse; keep the backup and capture the new actor coordinates before another case. The helper never stops processes, deletes layouts or edits thread/native records. It checks the terminal lease, canonical socket, exact server/wrapper/agent identities and unchanged policy, and preserves exclusive synced backups. It is compile-checked; the injected live behavior is what this acceptance run verifies.

### 2026-10-13 — Operator correction: attach to answering live servers (supersedes policy admission)

**Trigger and authority:** TL receipt 6bf45548 reports the staged build made roughly twenty pre-upgrade live sessions unusable and blocked Couch startup through ScopeHoldsUnsafeTerminal. Missing session-policy metadata was treated as operational danger even though those servers were answering normally. The operator replaces that rule: **an existing live server answers its session socket → attach as before, regardless of age or policy; no answering live server → never resurrect, offer Relaunch.** The earlier policy-withholding acceptance cases and fixture are obsolete, not success criteria.

**Design:** reuse the existing SessionOwnerProbe/Zellij pane query and exact server/socket observation as the attach evidence, not a newly introduced durable attestation. Keep the per-terminal admission lease and serialization-disabled configuration for new servers. Keep established foreign-owner/ambiguous-owner protections; lack of a new policy file is never an ownership conflict. No policy migration, mandatory legacy restart, cache deletion or new per-session proof file. A failed/timed-out probe supplies no authority to launch another server. Live-session attachment does not require new-format readiness/capture metadata; these remain separate evidence for native-conversation recovery.

| Concept / boundary | Owner | Change |
|---|---|---|
| Terminal attach admission | launcher/session_handoff_os.go and session_owner_os.go | Modified: successful bounded live socket query plus existing owner checks; no SessionLaunchPolicy prerequisite |
| Admission ordering | launcher/session_handoff.go and session_lease.go | Retained: lock across probe/start/admission; recheck exact server generation, no creation fallback on attach failure |
| Console row/startup projection | couchcore/artifactcollision.go, sessionevidence.go, agent_usability.go, startup.go | Modified: same live-server observation; legacy metadata absence cannot hide live rows or hold startup |
| Session policy artifact | artifactpath/session_policy.go, launcher policy publisher/retirement, inventory manifests | Delete policy-specific reader/writer/attestation contract once callers are swept; retain stable terminal-lock path constructor and inode lifetime |
| Recovery candidates / capture witness | existing M2 owners | Unchanged; policy files never establish native binding |

**Implementation steps after TL approval:**

1. Write regressions reproducing pre-upgrade live servers with no policy: selected attach succeeds, hosted/detached rows remain usable, and Couch startup is not blocked. Also cover a stale/malformed policy beside a healthy responding server: it is not attach authority and cannot make the session unsafe.
2. Refactor the shared observation/admission path to use live server/socket evidence. Sweep initial resume, warm attach, post-attach admission, startup auto-selection, selected actions, background row projection and fakes. Reuse existing snapshot batching; do not introduce an external pane query per row on every paint (ARCH-DRY, ARCH-CONSTRAINTS).
3. Keep explicit creation separate. A missing socket, socket with no answering server, dead server or saved-layout-only name must refuse ordinary attachment into Relaunch. Never call create/resurrect as an attach retry. Preserve exact owner checks for actual conflicting live generations rather than interpreting missing metadata as a conflict (ARCH-ORDER).
4. Remove now-unused session-policy persistence/retirement and its inventory entries/tests; leave permanent admission lock inodes and capture witnesses intact. Replace the policy-mutating scratch fixture with live-server/no-server cases. No machine cache cleanup.
5. Prove the external boundary on scratch Zellij sessions: answering old-style server with saved layout and no policy attaches; absent/dead server with saved layout never spawns a replacement; concurrent Pair attachment stays serialized. Explicitly inject server death between probe and attach. A Pair lock cannot stop Zellij dying: stock attach must be measured for this case. If it can resurrect after that death, stop and report the remaining existing-only-attach gap with the smallest no-fork option; do not claim a preflight probe closes the race or silently broaden the design (ARCH-MOCK).
6. Run targeted launcher/core/UI/startup tests, then the gated full suite. Update atlas and compatibility acceptance; build/stage only on TL scheduling. TL/operator smoke must include normal Couch restart over pre-upgrade live sessions as well as fresh/recovery scratch cases. Whole-issue close remains pending.

**Alternatives rejected:** grandfathering by fabricated policy keeps unnecessary persisted authority and requires migration; forcing every old live session to relaunch breaks the operator's workflow and is explicitly rejected. The selected design uses operational evidence already available. This revision is a proposal awaiting TL approval; no implementation is included.

### 2026-10-13 — Operator replacement: Zellij liveness, nonblocking Couch startup (PLAN ONLY)

**Authority:** TL receipt 15876eb2-50a4-414f-9e1e-4fbc751ed838 supersedes commit 3cd0aa805 and the earlier policy/socket-probe proposals. This revision requires **operator approval before implementation or build**. No code, runtime state or staged build is changed by this plan.

**Contract:** Couch starts regardless of individual slot state. Zellij owns terminal liveness: listed by `zellij list-sessions` and not `EXITED` means live; `EXITED` or absent means not live. At that slot's attach action, under the existing attach lock: live means attach normally, including old sessions without new metadata. Not live means never call `zellij attach`; call `zellij delete-session NAME` (no force) for that exact stale session name, then offer **Relaunch**. Startup/refresh only observes; it neither attaches nor deletes stale layouts as a side effect. One failed or unhealthy slot never blocks the supervisor or another slot.

**Retained:** layout saving disabled for newly created sessions; per-slot attach lock across final list/decision/attach-or-delete, so cooperating Couch processes cannot race that slot; proven resume, explicit ranked candidate evidence and tty-oriented fresh fallback as built. Native ownership protection for starting a recovery remains separate from whether an existing Zellij session is live.

**Removed:** ScopeHoldsUnsafeTerminal-style supervisor refusal, custom socket/agent-readiness probes as terminal-liveness/attach prerequisites, session-policy attestation persistence/admission/retirement, policy inventory and policy-mutating acceptance fixture. No fabricated legacy policy or mass relaunch/migration. Keep lock path construction/inodes independently of the deleted policy family. Historical files already on disk need no startup migration or machine-wide deletion.

| Surface | Owner(s) | Revised responsibility |
|---|---|---|
| Listed session facts | launcher/zellij.go, zellijparse.go | Reuse Zellij list-sessions and EXITED interpretation; no new liveness authority or per-row socket probing |
| Attach decision/effect | launcher/session_handoff.go, session_handoff_os.go, session_lease.go | Under lock, re-list exact name; live → attach; not live → delete-session without force, return Relaunch; no fallback to creation |
| Supervisor startup and rows | couchcore/couch.go, startup.go, sessionevidence.go, artifactcollision.go, agent_usability.go; existing couchcmd startup integration | Always enter Couch; project per-slot state from listing, keep failures local; no unsafe-primary refusal and no auto-created replacement for an unhealthy recorded slot |
| Retired policy artifacts | artifactpath/session_policy.go, manifest.go; launcher policy readers/writers/retirement and fixtures | Remove policy-specific surface; retain terminal lease constructors/lifetime and unrelated recovery evidence |
| Resume ladder | M2 recovery preview/executor, CLI/menu | Retained; Relaunch continues to the existing explicit recovery choices |

**Implementation checklist after approval:**

- [ ] Add failing regressions for supervisor startup with mixed healthy/EXITED/absent/legacy slots: the console starts, healthy slots remain usable, another slot's error remains local. Include the exact upgrade case: multiple slots launched by pre-#528 Couch, no policy files, restart supervisor onto the new binary without restarting those slots.
- [ ] Use the existing Zellij listing seam for one shared liveness rule. Preserve bounded/contextual command handling. A failed listing is an observation error, not absence: no delete or attach on that error; display it on the selected slot and leave Couch/other slots running (ARCH-DRY, ARCH-ORDER).
- [ ] Change selected attach under the existing lock. Listed non-EXITED → normal attach, with no policy/socket/readiness prerequisite. EXITED/absent → no attach invocation, delete-session NAME without force, show Relaunch. Treat a verified already-absent saved layout as an idempotent cleanup outcome; genuine deletion failure stays visible on that slot and does not authorize attach, forced deletion or supervisor exit. Measure native delete-session exit semantics rather than guessing them.
- [ ] Remove policy persistence and the obsolete scratch injector; update exhaustive artifact inventory/tests and atlas. Keep serialization-off creation and lock. Sweep startup selection, explicit/warm attach, post-attach checks and background projection so no old policy gate survives (ARCH-PURPOSE).
- [ ] Extend the stateful Zellij fake with active/exited/absent listing, saved-layout deletion and server death between list/attach. Test two cooperating attach callers, no attach in not-live cases, no deletion during startup/refresh, no force, no resurrection retry, and no effect after failed listing.
- [ ] Measure on disposable sessions only: pre-upgrade active session attaches despite saved metadata/no policy; EXITED/absent selected session deletes stale layout and offers Relaunch; new nonserializing server dying between list and attach causes attach failure. Record commands, timing and observed process/session identities. The same death race on an old serializing server may still resurrect its saved layout: measure and report honestly, do not claim the lock stops server death and do not add a socket probe/policy workaround without operator direction (ARCH-MOCK).
- [ ] Run targeted startup/launcher/core/menu tests and the gated full suite; stage only after scheduling with TL. TL/operator acceptance must restart Couch over pre-upgrade live slots, then exercise scratch not-live cleanup and the unchanged resume ladder. Close only after that acceptance.

**Remaining race:** list-sessions and attach are separate Zellij operations. The lock serializes cooperating callers, not server death. Newly created sessions have no saved layout, so the expected attach outcome after death is failure; confirm this in scratch conformance. For pre-upgrade sessions with saved layouts, do not label the race eliminated: retain the measured result and report it. No new race measurement was run during this PLAN ONLY revision.


## Revisions (continued)

### 2026-10-13 20:20 PDT — Three outcomes; code-verified state manifest and reconciliation (PLAN ONLY)

**Authority:** operator frame delivered by TL receipt `70291ee7-93d3-4b62-b027-9f8162a08357`. This is the current design, superseding incompatible earlier revisions and the policy-based scratch acceptance script. **Awaiting operator/TL review; no implementation, build, stage, cleanup or live recovery is authorized by this document.** Source audit: issue branch at `53f25c055` (implementation last changed at `e1876500e`); installed Zellij reports 0.45.1. No working session was probed or changed for this audit.

**User model:** the transcript is the asset; terminal, wrapper, transport and registry are rebuildable. Exactly three outcomes, each with one primary action:

| Outcome | Primary action | Meaning |
|---|---|---|
| CONTINUE | Continue | Use the functioning live conversation; attach and reconnect messaging where possible. Missing bookkeeping alone never authorizes stopping it. |
| RESUME FROM DISK | Resume from disk | Preserve evidence, retire only attributed unusable remnants, and launch on the proven or explicitly chosen native transcript. |
| START FRESH | Start fresh | Preserve available context, retire unusable remnants, launch a new conversation with branch/issue/previous-screen orientation. |

Candidate selection is a detail of Resume from disk, not another slot outcome: proven binding first; otherwise an explicit best candidate plus up to four alternatives, with agent, last activity, tail-match percentage and excerpt. Automatic inference remains OFF. A failed/incomplete observation is not proof of death or lost transcript: keep the relevant action pending with its reason and retry observation within that action; no fourth destructive action and no speculative Fresh fallback. A functioning agent can Continue even when its on-disk binding is unproven. A recovery action that would stop a running agent requires explicit confirmation naming that process/conversation and the reason it cannot continue.

#### Deliverable 1 — State manifest

**Path notation (constructors, not guessed machine constants):** `D` = Pair data root from `launcher.ResolveDataDir` (normally `~/.local/share/pair`, honoring XDG); `N` = canonical selected Couch namespace (`COUCH_STORE_DIR`, otherwise `D/couch`, `couchcmd/run.go:109`); `S` = `D/repos/<repo-scope>`; `tag` = work-thread tag, not slot number. `T` = routed thread-store root: global `N/threadstore`, numbered slot `<fleet>/worktree/<repo>-slotN/.couch` beside its host checkout. Scope and tag must be resolved, never inferred from a human repo basename. Exact path ownership is centralized in `artifactpath/paths.go`, `couchcore/slotlayout.go` and `threadstore.go`.

Classification: **RO** means payload observation with no intended writes; **BOOKKEEPING WRITE** includes locks/leases/journal replay; **WRITE** can change runtime, conversation input or durable payload. A query-looking name is not a guarantee. Source references below are relative to `cmd/internal/` unless a URL is given.

| State and exact store/path | How read; classification | Writers and effects on other pieces |
|---|---|---|
| Couch actor registry/names: `N/registry.json` | `Store.Load`, `couchcore/store.go:37`, RO (missing = empty). This disk registry differs from broker's live connections. | `Store.Save`/`mutateRegistry` atomically replace actors/names. Neither creates nor proves a terminal/agent. |
| Thread membership and record: global `T/manifest.json`, `T/records/<scope>/<tag>.json`; numbered slot `T/thread.json` | `ThreadStore.GetThread`/ordinary Snapshot are **BOOKKEEPING WRITE**: locking may replay journal. Use `PreviewSnapshot` for startup; it reports pending journal instead of replay (`threadstore_preview.go`). Strict record decoder in `threadrecord/record.go`. | Named revision-CAS mutations create/advance/remove incarnations, launch claims, bindings, park and recovery evidence. Slot-local membership is derived from thread.json, not a second manifest. |
| Store transaction staging: `T/journal.json`, `T/.thread-store-publication`; store lock | Direct bounded file/stat observation RO; ordinary store reads can recover it (`storejournal.go`). | Journal replay can publish records, membership, preferences and archive, remove staging/journal and fsync. Startup must not invoke it as an incidental read. Explicit selected-slot action owns recovery. |
| Thread identity and native binding: fields `address`, `starting_path`, `working_path`, `incarnations[]` (PID/start/state/nonce/profile), `session_binding`, `latest_launch_profile` in record | Preview and strict decode RO; process incarnation must be independently checked before effects. A recorded live state is not current liveness. | Start claim/registration promotes exact incarnation; native observation publishes binding; stopping a PID does not automatically rewrite record. Never manufacture binding from a registry label. |
| Park and historical state: record `park`, `park_history`, `verified_park`, `previous_session`, `recovery_context`; archived record `T/archive/<scope>/<tag>.json` | Preview/strict decode RO; ordinary archive listing may recover store. | `BeginPark`, `AdvancePark`, `AppendParkAttempt`, `FinalizePark`: matching completion removes exact incarnation, records park proof/history/capture. `ArchiveThread*` journals archive and removes current membership/record. Native transcript remains untouched. |
| Park handshake: `S/lifecycle-<tag>/<nonce>/quit-request-<attempt>.json`, `quit-completion-<attempt>.json`, `lifecycle.lock`; `S/lifecycle-<tag>.jsonl` | Read completion/journal RO; running controller reconciliation is WRITE. Constructors `artifactpath/paths.go:433`. | Controller request, wrapper completion and finalization are separate steps. Clean quit stops native child and records actual outcome; incomplete completion is not success. Retry uses exact nonce/attempt, never blind duplicate quit. |
| Launch selection: `T/path-preferences/<sha256(repoIdentity + NUL + physicalPath)>.json`, or local `T/preferences.json`; `S/config-<tag>-<agent>.json`; `S/agent-default-<agent>.json` | `PreviewPathLaunchPreference` and direct config decode RO. Ordinary preference getter can replay journals. `resolveConfigPath` is WRITE when migrating legacy Codex config (`createflow.go:1150`). | Successful ordinary launch journals preference with record; recovery-intent launch deliberately skips preference publication (`threadstore.go:894`). `writeConfig` writes selected config. Recovery must not edit human defaults. |
| Launch ledger and native-name cache: `S/ledger-<tag>.jsonl`, `S/session-names.jsonl`, `S/session-inventory-catalog.json` | File read/binding resolution and `CatalogStore.Read` RO. Full inventory/catalog refresh may WRITE; use explicit bounded recovery reader. | Sessionwatch prepares launch and appends launch ordinal/native authority; catalog Update/Repair writes derived metadata. Latest tag-level row alone does not identify the current wrapper generation. |
| Wrapper and native child: kernel PID/start/PPID/argv; `S/pair-wrap-pid-<tag>`, `S/agent-pid-<tag>` | `procutil.Table`, exact PID/start comparison, bounded sidecar read RO; enumerate only on disagreement or before attach/send/recovery. PID alone is insufficient. | Wrapper starts native child; exit/clean park/signals affect descendants and readiness. Wrapper messaging startup may fail after child is already running (`wrapcmd/wrap.go:2935`). SIGUSR2 restarts/kills child, so is NOT reconnect (`wrap.go:3022`). |
| Readiness/adaptation: `S/agent-ready-<tag>-<agent>.json`, `S/adapt-<tag>.jsonl`, `S/agent-<tag>`, pane/outer-tty sidecars | Strict readiness plus exact process generation RO; adapt log RO diagnostics. Missing readiness alone does not prove dead terminal or justify teardown. | Wrapper atomically publishes readiness including native PID/start, launch ordinal, nonce, session and orientation status (`wrapcmd/wrap.go:604`); adapt appends diagnostics. New launch overwrites current-generation sidecars, so capture evidence first. |
| Broker/registry sockets: `/tmp/pair-message-<uid>/<first-20-bytes-SHA256(N + NUL + endpoint)>.sock`, endpoint `broker` or `registry` | `couchmessage.SocketPath` pure; observation/discovery RPC RO of slot payload, but accepted RPC/connection changes volatile transport state. | Supervisor creates listeners; registration validates and adds in-memory binding. Restart loses in-memory connections/receipts, wrappers redial. Unlinking a pathname does not kill an already-connected process. Never remove the entire shared directory. |
| Wrapper message socket: same directory, `wrapper-<pid>-<incarnation-hash>.sock`; wrapper receiver/delivery state in memory | `tail`, `observe`, `status` RPC observe; `reserve`, `release` alter delivery state; `commit` can write PTY input. `couchmessage/endpoint.go`, `wrapcmd/peer_runtime.go`. | Listener currently created once. Registry reconnect does NOT recreate an unlinked wrapper endpoint. Cleanup removes only proven-dead endpoints with inode check (`socket_cleanup.go`). Lost commit reply is uncertain delivery; reconnect must not replay input. |
| Registry connection/registration: wrapper's persistent hello/activity stream and broker's in-memory session map | Broker snapshot/status RO of conversation; no connection is not proof native child is dead. | Existing registry client redials with 250 ms–5 s backoff and resends observation (`session_transport.go:243,333`). Idle existing connection has no pathname heartbeat; initial endpoint bootstrap failure has no later retry. These are reconnect gaps, not teardown authority. |
| Zellij live server: kernel process; socket `<socket-root>/contract_version_1/<session-name>` | `zellij list-sessions --short` and `--no-formatting` via `ZellijSource.LivenessContext` (`launcher/zellij.go`); listed non-EXITED = live. **Not strictly RO:** Zellij may unlink a connection-refused stale socket. ps/Lstat RO only when disputed/before effects. | Server creation opens socket/panes/commands; kill terminates session. Socket loss does not prove server/agent died. `ZELLIJ_SOCKET_DIR` overrides root; Pair currently defaults to temp `zellij-UID` while upstream may use platform runtime root. Match the actual namespace, not a reconstructed guess. |
| Zellij saved state: platform ProjectDirs cache `contract_version_1/session_info/<name>/session-layout.kdl` and `session-metadata.kdl`; macOS `~/Library/Caches/org.Zellij-Contributors.Zellij/...`; Linux normally `$XDG_CACHE_HOME/zellij/...` | stat/read RO. Listed EXITED can be a saved layout, not a functioning server. | Serialization writes layouts/metadata. `attach NAME` is WRITE and can resurrect from layout even without --create. `delete-session NAME` recursively removes exact saved-session directory; `--force` also kills server and is excluded. `kill-session` sends KillSession. Never delete native transcript as part of these effects. |
| Pair terminal lock and rejected policy: `~/.cache/pair/session-locks-v2/<first-SHA256-byte-of-canonical-socket>.lock`; `~/.cache/pair/session-policy/<SHA256(socket)>.json` | Lease acquisition is BOOKKEEPING WRITE (mkdir/open/flock); policy decode RO but obsolete as authority. `artifactpath/session_policy.go`, `launcher/session_lease.go`. | Keep permanent lock inodes/serialization-off launch; remove policy publication, reads/admission, retirement and fixtures. Do not sweep historical policy files at startup. Existing `PrepareRecoveryTerminal` acquires leases; it is not a read. |
| Native Claude transcript: `~/.claude/projects/<cwd-slug>/<session-id>.jsonl`; nested `<session>/subagents/` distinguished | `NewOSRuntime`, `CollectRecoveryEvidence`, `scan_claude.go`, `recovery_candidate.go`: bounded no-follow file reads RO. Check payload cwd/entrypoint/root status, not filename alone. | Claude writes transcript. Pair resume passes selected ID; Pair recovery never deletes, renames or rewrites native transcript. Subagents/tool runs are not interactive candidate roots. |
| Native Codex transcript: `~/.codex/sessions/YYYY/MM/DD/rollout-…-<id>.jsonl` | Same bounded reader; `scan_codex.go`/recovery parser verify session_meta cwd/source/parent and semantic activity. originator alone is not interactive proof. RO. | Codex appends native events; resumed native session remains asset. An observed competing owner prevents a second launch on it; no duplicate native owner created by inference. |
| Native Grok transcript: `~/.grok/sessions/<URL-encoded-cwd>/<id>/updates.jsonl` | `scan_grok.go`: JSON-RPC sessionId/timestamp, directory cwd; summary/events/prompt_history are not authoritative transcript. RO. Current inference has no positive interactive discriminator, so cannot call it a qualified inferred candidate. | Grok writes updates. Proven native binding can use supported resume; lack of inferred eligibility must be explained, not reported as physical transcript absence. |
| Current tty evidence: `S/scrollback-<tag>-<agent>.raw`, `.events.jsonl`, derived `.ansi`, `.viewport`, `.openlock`; prompt log `S/log-<tag>.md` | `RenderBoundedLinesContext` replays raw/events; live wrapper tail RPC; `couch --peek repo:N [--transcripts]`, `--previous --all`. Rendering/read commands may acquire retention bookkeeping; file payloads are not changed. | Wrapper writes raw/events and producing witness (ordinal/nonce/opened_at/file IDs). New launch can replace current capture; previous archive must be preserved before that. Viewport/rendered files are derivatives, not native conversation identity. |
| Retained tty evidence: `S/parked-scrollback-<tag>-<token>.raw`, `.events.jsonl`, `.capture.json`; capture reference in record | `recoveryCaptureSource` prioritizes pending recovery intent, retained recovery context, verified previous capture, then current capture. Exact identity checks; RO payload. | `RecoveryPreserve` copies settled evidence and metadata before retirement; `PreserveRecoveryBeforeRetirement` persists context. Original activity/producing launch must survive later empty starts; archive publication time is not original activity. |
| Recovery candidate list and intent | Preview in memory; selected context/key/native ID in thread record and start claim | `ResumeCandidates` has no candidate-cache write but its evidence reader acquires retention lease: BOOKKEEPING WRITE. Keys bind record revision + evidence generation; candidate K means opaque returned key, not rank number. | Selection persists intent before stop, revalidates at Couch and Pair gates. Failed launch/rollback retains intent and tty evidence. No automatic candidate pick or parameter edits. |
| Orientation delivery | Launch request in tracked start; status in readiness; wrapper delivery state | Read status RO; builder pure (`orientation/recovery.go`). | Orientation delivery writes agent input once, with exact branch/issue/capture coordinates. It does not enter reusable launch argv; uncertain submission is never replayed automatically. |
| Retention ownership | `D/.retention/owners/<SHA256(owner-key)>.json` and coordinator lock | `AcquireSelectedProcessTarget`/Close are BOOKKEEPING WRITE even for recovery readers (`storagegc/lease.go`). | Register/release process leases prevents GC during capture reads/preservation. Existing capture retention bounds apply; startup does not invoke candidate scans or per-slot retention enrollment merely to classify a row. |
| Workspace/branch/issue | Host checkout, `.git` pointer + common-dir worktree registrations/refs, `workshop/issues/…`; slot `.couch/saved-work`; `.weave-setup.lock` | Git identity/ref reads and issue read RO (avoid status paths that refresh index); `SlotLayout` enumerates paths. | Recovery uses existing checkout/config. No branch reset/clean, worktree deletion or issue mutation. Workspace repair is separate from stopping an agent and must not occur implicitly in startup projection. |
| UI/workbench residue | `S/draft-<tag>.md`, `queue-<tag>/`, `quote-<tag>`, pane/nvim/title PID files, `wrap-events-<tag>.jsonl`, layout/review/terminal-pane sidecars enumerated in `artifactpath.Paths`/manifest | File read RO; stale editor PID is not agent identity. | Editors/wrapper maintain them; park/relaunch may retire scoped runtime sidecars. Preserve drafts/queued work; do not wildcard-delete scoped directory. Derived review/render caches are not transcript substitutes. |

**Upstream verification:** installed-version source confirms [socket listing/probe and ConnectionRefused unlink](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L136), [attach resurrection selection](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L719), [socket/cache constructors](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/consts.rs#L305), and [kill/delete behavior](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L281). These are source findings, not a live conformance result.

**Startup consequence:** current `couchcore/couch.go:211` reconciles interrupted starts, `couchcmd/run.go:629` starts `RecoverActiveParks`, and ordinary store reads replay journals. The redesign must remove these per-slot effects from startup, not merely remove `ScopeHoldsUnsafeTerminal`. Use preview paths and project pending operations locally. Couch's own namespace/listeners remain necessary control-plane startup writes; they must not become slot repair. There is one explicit review question: accept Zellij's native stale-socket cleanup as part of list-sessions, or require strict zero-filesystem-write observation? The requested command itself cannot provide the latter. Do not hide this exception behind the word “read-only.”

#### Deliverable 2 — Reconciliation table

**Read order:** (1) registry + preview thread snapshot, (2) one shared bounded Zellij listing in the correct namespace, (3) only for disagreement or a selected attach/send, exact process generations and bounded channel observation, (4) on explicit recovery, proven transcript check then bounded candidate/tty scan. No startup transcript crawl, signal, socket removal, journal replay, park completion or saved-layout deletion. Listing errors remain errors, not an empty list. Pending metadata cannot prevent the console from opening or another slot from working.

The table refers to the exact ordered sequences below. All are **proposed behavior**, with current implementation gaps called out; no implied permission to run them now.

| Observations | Outcome and visible action | Ordered writes and safeguards |
|---|---|---|
| Listed live; expected agent functioning; channel registered | CONTINUE → Continue | C. Attach normally; no recovery, defaults edit or transcript scan. Old session without policy is identical. |
| Listed live; agent functioning; broker registration/connection missing or channel pathname deleted | CONTINUE → Continue (detail: reconnecting messaging) | C then B. Keep terminal/native PID unchanged. Existing broker restart redial works; lost wrapper listener requires new reconnect support. Old wrapper still supports terminal use; lack of messaging repair does not convert it to Resume. |
| Listed live; durable registry entry missing, but pane/process/worktree identify functioning agent | CONTINUE → Continue | C; B rebuilds volatile registration from validated wrapper hello. Persist missing routing metadata only on explicit action and exact identity evidence; never synthesize native proof. If identity conflicts, do not send to an arbitrary process. |
| Session listed live but panes are empty/dead; no functioning native child | RESUME FROM DISK if proven/inferable, otherwise START FRESH | R or F. A live terminal server alone is not Continue. Inspect agent process before effects; retire exact empty terminal after evidence selection/preservation. |
| Listed absent/EXITED; recorded wrapper/agent also proven gone | RESUME FROM DISK when transcript usable | R; no running-agent confirmation needed, but candidate choice explicit. Delete only this stale saved layout after fresh listing under lock. Never call attach. |
| Listed absent/EXITED; original server/wrapper/native agent still running but terminal unreachable (split/socket loss) | RESUME FROM DISK, or START FRESH if unrecoverable | R/F **with explicit stop confirmation** naming original process generations. Channel access to old wrapper is not proof operator can Continue in that terminal. No second server until attributed owners gone. Never repair by starting same-name server or transplanting a socket. |
| Multiple servers/incarnations contend; one attachable functioning conversation can be uniquely identified | CONTINUE for that functioning owner | C only after identity match to the listed attach target. Leave other owners untouched; do not choose from process age or registry label. Conflicting ownership of the same native ID blocks new resume. |
| Contested incarnations cannot provide a functioning attach target | RESUME FROM DISK or START FRESH, pending exact ownership and stop confirmation | R/F only for positively attributed remnants; unknown/foreign owners are not kill targets. If evidence cannot name safe targets, the same action remains pending with reason; do not “clean all” to force progress. |
| No usable live agent; binding PROVEN and native artifact validates | RESUME FROM DISK → Resume from disk | R using native ID directly; binding proof does not waive process-owner check. Show resumed ID and previous-screen coordinates. |
| No usable live agent; binding unproven; ranked candidates available | RESUME FROM DISK → Resume from disk | R with explicit best or option1–4; display evidence. CLI lists same choices and accepts returned key. No silent inference regardless of score. |
| No usable live agent; complete scan yields no usable transcript (or operator rejects candidates) | START FRESH → Start fresh | F with retained tty, branch and issue. Do not delete rejected native transcripts. |
| No transcript and tty missing/empty/expired | START FRESH → Start fresh (detail: previous screen unavailable) | F with branch/issue and honest missing-screen notice. Current code requires nonempty capture and absolute raw/events; revise this so missing last-resort evidence does not make fresh start impossible. |
| Listing/process/native scan fails, changes during read, or budget exhausted | Corresponding action remains pending; explain failed evidence check | No teardown/start on uncertainty. Retry bounded observation from the same action. Never turn incomplete candidate scan into “no candidates,” or unresolved liveness into death. |
| Interrupted park/start or selected recovery intent survives restart | Startup projects Resume from disk/Start fresh as appropriate, or Continue if agent actually functions | No startup completion. Selected action first reconciles its exact persisted operation; if a new agent already exists, report/Continue it rather than launch another. Preserve old evidence when no launch is proven. |

**C — Continue / attach:** acquire existing per-thread attach lease (bookkeeping write); freshly list exact session and require current generation-matched usable-agent evidence before attach (reuse a validated wrapper observation or, when absent/disputed, inspect the exact process/pane generation; a broken message socket alone is not failed agent evidence). If agent death is established, re-project R/F; if agent usability is unknown, keep Continue pending. If no longer listed live, do not call attach, release and re-project to R/F without teardown. Otherwise run normal `zellij --config-dir CONFIG attach NAME`. Do not read policy or require it, delete saved layouts, stop native agent or migrate defaults. Attach changes clients and can have the remaining resurrection race described below. A missing channel does not prohibit terminal Continue.

**B — Reconnect without restart:** supervisor listener ownership remains protected by its existing lease; recreate only its own missing listener endpoint and accept authenticated wrapper hello. Existing wrappers already redial after connection failure. Add wrapper-owned endpoint reconciliation in `wrapcmd/peer_runtime.go`: under the same runtime lifetime, rebind only its own missing socket, preserve the receiver's reservations/uncertain delivery state, and reannounce its exact PID/start binding. It must not send a new prompt, reset native identity, relaunch child, republish launch ledger or clear delivery uncertainty. A conflicting endpoint is not unlinked. No external process can repair an old wrapper's unlinked listener by binding on its behalf; an old binary remains Continue by terminal with a truthful messaging-unavailable detail. Upgrading that wrapper later is a separately confirmed relaunch, never automatic channel repair.

**R — Resume from disk (one transaction, exact selected slot):**

1. On explicit action, acquire slot operation hold; preview/recheck record revision, listing and exact process identities. Validate selected proven binding/candidate/native artifact, saved agent configuration and working directory. Any live owner outside the positively identified retiring source blocks duplicate resume.
2. If a functioning attachable conversation is now present, return Continue. Otherwise gather evidence of inability to Continue. If any running agent would be stopped, obtain confirmation tied to exact PID/start/native selection before any stop; changes invalidate it.
3. Acquire retention protection and preserve original tty/branch/issue; persist selected recovery intent by named CAS before retirement. Existing `PreserveRecoveryBeforeRetirement` is the owner. An absent tty is recorded as absent, not fabricated; validated native transcript remains sufficient for R.
4. Recheck ownership/confirmation. If wrapper responds, use normal exact-source park/quit handshake and verify native child exit. If channel is unavailable, use only positively attributed exact process-tree retirement, with signal/wait/recheck bounded by the existing quiescence owner. Never use a PID-only kill, arbitrary pkill, or assume send success proves exit.
5. Verify all attributed old native/wrapper/server processes are gone. Retire leftover exact terminal server (no stopping an unconfirmed newly observed agent). Under attach lease freshly list; if another live server appears, stop the transaction rather than delete it. `zellij delete-session NAME` without force removes the stale layout; already-absent semantics must be verified. Remove only exact dead-owned socket/sidecar generations with inode checks; keep shared broker sockets, lock inodes, native transcripts, drafts and retained captures. Existing cleanup owners must enumerate these effects rather than broad filesystem removal.
6. Commit start claim carrying retained intent; revalidate native owner/evidence at both Couch and Pair gates. Start new nonserializing terminal/wrapper/native agent with selected resume ID and saved config. Registration/readiness must name this launch; don't declare success from spawn alone.
7. Deliver previous-screen orientation once (when supported), show native ID/evidence and outcome; attach only after listing live. If startup fails or is interrupted, keep original recovery intent/coordinates and expose the same Resume action. Do not silently launch Fresh or repeat an uncertain orientation.

**F — Start fresh:** same R ordering and confirmation/retirement safeguards, selecting a fresh intent instead of a native ID. Complete recovery assessment or explicit operator rejection precedes it. Preserve whatever tty evidence exists plus branch/issue, then launch fresh with saved configuration and a one-time orientation. If tty is absent, state that plainly and provide branch/issue; no fake paths and no rejection solely for missing screen. Old transcripts and git work remain intact. A failure keeps the same Fresh action and retained context, not a forced second launch.

**CLI/menu:** keep `couch --resume-candidates repo:N [--json]` and `couch --resume repo:N --candidate KEY [--json]`; ordinary `--resume repo:N` is proven resume. Add/route explicit confirmation for any selected recovery stopping a running native agent; existing ResumeCandidate currently has no such user confirmation parameter, so source-ownership checking alone does not satisfy this design. Proposed confirmation contract for **all R/F routes**, including proven resume and fresh: preflight returns `confirmation_key` binding record revision, exact source PID/start set, selected action and native ID (or fresh), plus evidence digest. Menu confirmation submits that same key; CLI syntax is `couch --resume repo:N --candidate KEY --confirm --confirmation CONFIRMATION_KEY` (omit candidate for proven resume; fresh uses its returned fresh-choice key). A bare --confirm cannot authorize stopping whichever processes a later invocation happens to discover. The executor recomputes the bound facts before effects; changed identity/selection invalidates confirmation. This short-lived key is not a new persisted authority or liveness state. Menu uses one primary outcome action with candidate subchoices; retaining alternatives after inferred resume does not authorize stopping a newly functioning agent without a separate explicit selection and confirmation. Observation endpoints expose evidence/error details without inventing another lifecycle state.

#### Remaining races and review decisions

1. **Native list-sessions side effect:** Zellij 0.45.1 unlinks a stale socket after ConnectionRefused. Proposed precise interpretation: “startup performs no Pair recovery/teardown writes; native Zellij listing may clean connection-refused stale sockets.” This needs operator agreement; strict zero-write startup cannot use that command unchanged. Ordinary store journal replay and automatic park/start recovery are not covered by this exception.
2. **List-to-attach death:** stock attach can resurrect an old saved layout after a successful listing if the server dies in between. Per-thread lock prevents cooperating Pair callers racing, not server death. New sessions disable serialization, so absent saved layout should make attach fail; this still requires disposable conformance measurement. Legacy saved layouts retain a real race. No custom socket probe, policy attestation, preemptive legacy deletion or Zellij fork is proposed as a hidden fix. Operator must accept the documented legacy race or choose a different existing-only attach mechanism after measurement; no “race eliminated” claim.
3. **Reconnect compatibility:** new wrapper can own endpoint repair; already-running older wrapper cannot acquire new code without a restart. Continue by terminal is safe and useful while messaging is unavailable. Confirm this visible limitation rather than presenting channel loss as agent damage.
4. **Unknown ownership is not permission:** three outcomes are the human model, not a promise every unsafe write is always executable. A pending action explains the single missing observation/confirmation; uncertain ownership never grants blanket cleanup.

#### Implementation map and verification after approval

| Entity/integration | Files | Planned change |
|---|---|---|
| Recovery confirmation witness (pure, new) | `couchcore/recovery_preview.go`, existing resume request types | Bind exact source generations/action/native selection to explicit confirmation for both R and F; no new persistent store. |
| Pure three-outcome projection | `couchcore/sessionevidence.go`, `actionableinventory.go`, `agent_usability.go`; relevant vocabulary model before editing status sets | Modify one shared decision from observed facts; menu/CLI/startup consume it (ARCH-PURE, ARCH-DRY). Do not add a second durable liveness authority. |
| Observation-only startup | `couchcore/couch.go`, `startup.go`, `threadstore_preview.go`, `couchcmd/run.go` | Remove unsafe-slot supervisor refusal and per-slot startup recovery; reuse preview reads; errors remain slot-local. |
| Terminal integration | `launcher/zellij.go`, `session_handoff*.go`, `session_lease.go`, `recovery_terminal.go`, `session_quiescence.go` | Listing drives attach admission; preserve lock/nonserialization; exact explicit recovery owns cleanup. Delete policy authority/fixture, not existing runtime files at startup. |
| Channel reconciliation | `wrapcmd/peer_runtime.go`, `couchmessage/session_transport.go`, `couchcmd/message_service.go` | New wrapper repairs only its own endpoint, re-registers without native restart or message replay; preserve legacy limitation. |
| Recovery execution and presentation | `couchcore/recovery_resume.go`, `recovery_preview.go`, `recovery_evidence_os.go`, existing resume CLI/menu and `orientation/recovery.go` | Confirmation before stopping a live source, retained intent/rechecks, missing-tty fresh/proven recovery, same ranked evidence and no auto-pick. |

- [ ] Write failing stateful tests for every table row and identity change at every write gate. Assert exact surviving native PID/transcript in Continue, no attach when not listed live, no kill without identity-bound confirmation, and no defaults/native-file edits.
- [ ] Add startup regression with mixed pre-upgrade live slots (no policy), saved-only slots, missing registry, pending journal/start/park and observation errors. Assert Couch opens and no per-slot reconciliation effects occur. Model the explicit native-listing exception separately from Pair writes.
- [ ] Add channel tests for broker restart, listener unlink, bootstrap failure, foreign endpoint collision and lost commit reply: reconnect preserves native child, queued/delivery uncertainty and transcript; no input replay.
- [ ] Test R/F interrupted before/after capture publication, source retirement, delete-session, start claim, child spawn, registration and orientation. Retry reads retained operation/evidence; never creates duplicate owner. Missing tty cannot block known transcript resume or fresh branch/issue orientation (ARCH-ORDER).
- [ ] Extend existing stateful Zellij fake for live/EXITED/absent, saved layout, listing cleanup, server death between list/attach, two callers and foreign replacement. After approval measure these on disposable names only, including real delete-session already-absent behavior and legacy race; compare with installed 0.45.1 source (ARCH-MOCK).
- [ ] Run targeted packages then gated full suite; update atlas and acceptance runbook. New live acceptance must restart Couch over sessions launched with old binary, verify Continue without any policy, then exercise scratch broken channel, unreachable-live owner with confirmation, proven/inferred/no-transcript recovery and missing tty. Stage only when TL schedules it; close only after operator acceptance.

No new persistent policy or candidate cache is proposed. Existing bounded capture/intent retention and stable lock lifetimes remain owners (ARCH-FUNERAL, ARCH-CONSTRAINTS). The broad effect inventory is necessary to preserve the transcript while making the visible model simpler (ARCH-PURPOSE); implementation must remove the superseded gates rather than layer this projection over them.


### 2026-10-13 — Approved slot-continuity target: conformance review (PLAN ONLY)

**Authority:** verified TL receipt `7a3c544d-15c1-4c7a-b01c-5f36479f40f7`. The operator-approved [slot-continuity target](../targets/slot-continuity.md) was copied verbatim from `/Users/xianxu/workspace/ops/teams/devinfra/drafts/pair-target-slot-continuity.md`; #528 now references it in frontmatter and the atlas links it beside session identity. This target governs the plan. The earlier appendix is preserved as reviewed history; qualifications below identify where it must change before implementation. No target text was normalized or edited.

| Target commitment | Manifest/table assessment | Conflict or required disposition |
|---|---|---|
| Three outcomes, one action each | Continue / Resume from disk / Start fresh and candidate subchoices match. | **Conflict:** the uncertainty/contested rows and Remaining races item 4 permit indefinite pending with explanation alone. That is the limbo the target rejects, not a satisfactory fourth state hidden behind one of three labels. Replace indefinite retries with a bounded observation attempt and an actionable recovery choice; do not claim the existing table already satisfies this. Unknown foreign ownership still cannot authorize killing arbitrary processes. The revised design must specify a safe forward path for an attributed slot without making complete proof a permanent prerequisite. |
| Know which reads write; attach never probes | Manifest labels attach, leases, journal replay, candidate-read bookkeeping and Zellij listing side effects. C attaches only after listing and usable-agent evidence. | Aligned on classification. The legacy list-to-attach resurrection race remains unresolved, and the new target does not implicitly approve it. Record scratch measurement and operator decision before claiming safe existing-only attachment. |
| Startup observes only and always starts | Plan removes automatic interrupted-start/park reconciliation, ordinary recovery-capable store reads and per-slot startup refusal. | **Conflict/open decision:** proposed exception allowing native list-sessions to unlink connection-refused sockets is still a write. The target says startup acts on nothing; its approval is not approval of this exception. Keep this conflict explicit for TL/operator resolution. Own-supervisor listener setup is distinguished from acting on a slot; no per-slot repair may hide there. |
| Teardown only in outcomes 2/3, with evidence and confirmation for a running agent | R/F preserve context first, revalidate attributed source identity, then stop. C/B retain native child. | Aligned in intent. Confirmation can come from **the person or the authorized hub acting for them**. Source-generation binding is an implementation proposal for review, not a requirement that the human personally click every recovery. No extra operator-only gate is introduced. |
| Educated guesses over refusals | Plan ranks evidence, offers best candidate and four alternatives, keeps inferred selection explicit under prior TL calibration direction. | Choosing the primary Resume action may choose the best-supported candidate with evidence; it must not require proving the inferred identity. Autonomous selection before the person/hub picks an action remains off under the earlier instruction. **Tension:** strict provenance/cutoff/eligibility checks currently yield no selectable candidates in some legacy cases. Such missing proof must not strand the row: offer the best honestly supported choice when usable, otherwise Fresh. Whether to relax specific eligibility rules needs design review, not silent removal. |
| Proportionality | Transcript/tty preservation and removal of policy authority serve common paths. | The new confirmation-key API and channel self-repair are proposals, not yet approved necessities. Review whether existing operation admission can bind confirmation without another user-facing token. Do not let rare ownership corners dominate recovery or grow another persistent authority. |
| Start fresh always reachable | Plan covers absent/empty/expired tty and branch/issue-only orientation. | **Conflict:** F currently requires complete assessment or explicit candidate rejection, and incomplete scan returns no choice. Supersede that prerequisite: incomplete/unsupported/weak transcript evidence must still offer Fresh with an honest explanation of what may be lost. Retain available tty/branch/issue; missing capture/witness/configured inference confidence is not grounds to hide Fresh. This does not waive evidence/confirmation before stopping a running agent, nor authorize foreign-process destruction. The action must resolve those prerequisites rather than wait indefinitely for transcript proof. |
| Broken channel + live attachable agent remains Continue, repaired by reconnecting | Existing broker redial meets the common restart case; new wrapper-owned endpoint repair is proposed. | **Remaining implementation gap:** an old live wrapper with an unlinked listener cannot recreate it through current code. Terminal Continue preserves use but is not full channel repair. Keep the gap visible for review; never turn it into automatic agent teardown. |
| Hub may act for the person and reports actions | CLI provides evidence/candidate selection; previous prose did not clearly state delegated confirmation authority. | Clarified: the hub may select Continue, Resume or Fresh and confirm a running-agent stop on the person's behalf within this target. It reports what it did and verifies the resulting conversation/slot. No separate in-pane human approval is required solely because the actor is the hub. This delegation does not authorize implementation of this still-unapproved plan or mutations of current working slots during planning. |

**Review summary:** the target exposes a substantive design conflict, not just wording: indefinite “pending, retry observation” and mandatory complete transcript assessment undermine Fresh as the guaranteed exit. The next plan revision must make bounded evidence gathering lead to the best supported resume or a reachable Fresh action, while preserving exact-scope teardown and person-or-hub confirmation. Native listing writes, the legacy attach race and old-wrapper channel repair remain explicitly unresolved. The manifest's storage facts remain useful; its reconciliation policy is not yet ready for implementation. Awaiting TL/operator review; no code, tests, build or stage changes.

Verification: target bytes match the approved source (SHA256 `cacfebb254abba2db16eb230a9446a4dce570e1b28b6e91290375441c2542ce9`); issue/atlas references resolve. Staged whitespace check flags only the copied source’s trailing space on line 12 and blank EOF line; preserved intentionally to satisfy the verbatim-copy instruction. Other edited documentation passes the whitespace check.


### 2026-10-13 — Operator decisions: bounded observation, accepted gaps, three acceptance milestones (PLAN ONLY)

**Authority:** verified TL receipt `cb59774f-021f-480a-8d44-a7a344416088`. The operator approved the decisions below. They supersede the pending-limbo rules, usable-agent attach prerequisite and unresolved-gap dispositions in the preceding revisions. **This revision must be confirmed by TL before M1 implementation.** The [slot-continuity target](../targets/slot-continuity.md) remains unchanged. This is documentation only.

#### Shared projection and effects contract

Observe cheaply first: preview registry/thread facts and the shared Zellij listing; process/channel checks only on disagreement or before attach/send; bounded transcript evidence when needed. A failed, changed or over-budget observation gets **one bounded retry**, within an explicit overall deadline. No nested/restarting retry loop. After that, always project an action:

| Observation after at most one retry | Outcome / primary action | Additional choice or effect rule |
|---|---|---|
| Exact session listed live/non-EXITED by Zellij | Continue | Attach normally under the existing lock. Missing policy, readiness, broker registration or message endpoint does not hide Continue. Channel observations may explain unavailable messaging; they do not become another terminal-liveness gate. |
| Not listed live; usable transcript, proven or best-supported candidate | Resume from disk | Show native ID and evidence; best candidate is the primary resume choice, alternatives one step away (up to four). Selecting this action is explicit person/hub authorization to try the supported choice; proof of inferred identity is not required. Fresh remains offered. |
| Otherwise, including failed/incomplete/unsupported scans, weak evidence, missing tty or unrecoverable transcript | Start fresh | Explain the uncertainty and what conversation context may be lost; preserve available branch, issue, tty and draft/prompt evidence. No additional proof or successful scan is required to offer Fresh. |

Start fresh remains an available recovery choice even when Resume is offered. It does not imply permission to stop a functioning agent: teardown remains confined to outcome 2/3 with evidence the agent cannot continue, and stopping a running agent needs confirmation by the person **or the hub acting for them**, naming exactly what stops. The hub reports actions and observed results. No automatic background inferred launch is introduced; the primary Resume action chooses the displayed best-supported candidate when selected.

A failed listing is disclosed, not falsely recorded as proof the server died. It nevertheless cannot leave a pending-only row: after the one retry, choose Resume when usable and safe to resume, otherwise Fresh. Effects must not call attach without a positive live listing, nor infer permission to kill from a failed read. An unidentifiable or foreign remnant is left alone; choose a fresh terminal/session name and isolated launch outputs rather than overwriting its socket/current capture. If it might own the proposed native transcript, do not start a duplicate native owner: offer Fresh while retaining/reporting the candidate and remnant. Report leftovers with the action's result; do not make their removal a prerequisite for using the slot. Reuse existing fresh-name allocation and artifact ownership machinery; no new cleanup authority or persistent liveness mechanism.

On explicit recovery, preserve available evidence and selected intent before any attributed retirement. Recheck identities and confirmation before stopping; an identity change returns to this bounded decision path, not an indefinite hold. A short-lived, **non-persistent** confirmation key may bind exact source PID/start identities, action/native choice and record generation; prefer existing operation admission if it supplies the same guarantee. Confirmation is equally usable by the person and authorized hub. Missing tty must not block either proven resume or Fresh. Native transcript, git work, drafts and retained evidence are never deleted to simplify reconciliation.

#### Three accepted limitations — no additional mechanism

1. **Listing writes:** Zellij list-sessions may unlink a connection-refused stale socket. This native side effect is now explicitly accepted. Startup slot observation otherwise performs **no Pair writes**: no journal replay, per-slot retention enrollment, config migration, interrupted-start/park reconciliation, layout deletion or socket repair. Couch's own existing supervisor service setup is not a pretext for per-slot recovery.
2. **Legacy attach race:** a server can die between live listing and stock attach, allowing resurrection from its old saved layout. Accepted; measure on a disposable scratch session and document the result. Keep the cooperative attach lock and serialization-off creation; add no socket-probe gate, policy file, preemptive layout deletion, Zellij fork or alternative race-prevention mechanism. New nonserializing sessions' death behavior is included in scratch conformance.
3. **Old wrapper messaging:** pre-#528 wrappers cannot recreate an unlinked message endpoint. Continue by terminal, with the exact truthful detail **“messaging unavailable until next relaunch”**. This is an accepted compatibility limitation, not permission to restart its child. M3 improves newly launched wrappers only.

These decisions settle the corresponding conflicts in the target-conformance review. The state manifest remains a description of present storage and side effects, not a requirement to preserve the superseded mechanisms.

#### Revised milestone boundaries

The earlier closed M1/M2 entries and SHIP reviews describe the pre-revision implementation and remain historical evidence. They do **not** satisfy these replacement milestones. “M1/M2/M3” below are the operator's revised acceptance sequence; preserve prior review records and ensure each revised boundary reviews the actual new change window rather than treating a historical same-named close as completion.

- [ ] **M1 — Couch startup, attach and three-outcome projection.**
  - Files: `couchcore/couch.go`, `startup.go`, `threadstore_preview.go`, `sessionevidence.go`, `actionableinventory.go`, `agent_usability.go`; `couchcmd/run.go`; `launcher/zellij.go`, `session_handoff*.go`, `session_lease.go`; `artifactpath/session_policy.go`/manifest and affected fixtures; existing menu projection.
  - Pure decision owns the three outcomes and one-retry exhaustion rule; UI/CLI derive from it. Consult the vocabulary model before editing any lifecycle/status set. Reuse the Zellij listing seam and preview reads (ARCH-PURE, ARCH-DRY).
  - Add failing regressions first: mixed old live/no-policy, EXITED, missing session, missing record, pending journal/start/park, failed/changed/over-budget observations. Console opens; one bad slot never blocks it or another slot; every row has an action; startup invokes no Pair recovery writes.
  - Remove `ScopeHoldsUnsafeTerminal` and all policy authority/publication/retirement/fixtures. Keep lock path/inodes independently and nonserializing creation. Remove startup `reconcileInterruptedStarts`, `RecoverActiveParks` and recovery-capable store reads from slot observation; selected recovery belongs to M2, not a hidden background task.
  - Under the attach lock, freshly list: positive live → normal attach; otherwise no attach, project Resume/Fresh. No saved-layout cleanup during startup or an attach probe; explicit R/F execution owns cleanup. No readiness/agent-proof gate overrules listed-live Continue.
  - Extend stateful fake for one retry, list failure, legacy saved layout, server death between list/attach, lock contention and no-policy compatibility. Run targeted startup/launcher/core/menu tests, then required boundary checks; update atlas. Measure the accepted legacy death race and new nonserialization case on scratch names only, after implementation authorization.
  - **Operator acceptance session M1:** TL schedules staging/restart. Mandatory pre-upgrade case: restart Couch onto this build over live slots launched by the old binary, with no policy files; those slots stay Continue and attach/talk normally. Mixed scratch dead/error rows show Resume/Fresh without preventing startup; old broken messaging shows the accepted notice. Record accepted race measurement. M1 acceptance verifies projection and startup, not a claim M2 recovery execution is finished.
  - Checkpoint/review the changed window and record operator acceptance before requesting the next milestone; do not stage or advance automatically from this plan-only revision.

- [ ] **M2 — Execute Resume from disk and always-reachable Start fresh.**
  - Files: existing `couchcore/recovery_resume.go`, `recovery_preview.go`, `recovery_evidence_os.go`, named thread-store transitions, `launcher/recovery_terminal.go`/quiescence, resume CLI/menu and `orientation/recovery.go`.
  - Add failing tests for complete/incomplete/unsupported/weak scans and missing/empty/expired tty: Fresh always offered and executable with available branch/issue context. Resume uses proven or best-supported native evidence; no complete scan prerequisite to Fresh and no manufactured missing-screen paths.
  - Reuse retained-intent transaction: explicit choice → preserve available evidence → person/hub confirmation for exactly named running source → attributed retirement → dead-owned cleanup and exact non-force stale-layout deletion → revalidated launch → registration/readiness result and one-time orientation. Unknown/foreign remnants survive; fresh name avoids collision. Do not resume a native conversation still owned by an unretired process.
  - Confirmation key, if needed, stays short-lived/in-memory; test source identity changes, cancellation, one bounded observation retry and fallback without pending limbo. Preserve delivery uncertainty and recovery evidence through launch failure; no silent Fresh replacement after an uncertain successful Resume.
  - Test interruption at each external effect with stateful fakes and exact ownership checks. No defaults edits, native-file deletion, git reset or draft loss. Targeted tests, required full/boundary verification and atlas update.
  - **Operator acceptance session M2:** TL exercises scratch proven resume, best/alternative inferred candidate, no/weak/incomplete transcript evidence, missing tty, attributed running remnant with person-or-hub confirmation, and an unattributed remnant left intact while fresh-name recovery proceeds. Verify selected native ID or fresh conversation and orientation, with no duplicate owner or unexpected stop.

- [ ] **M3 — New wrappers repair their message endpoint.**
  - Files: `wrapcmd/peer_runtime.go`, `couchmessage/session_transport.go`/endpoint transport, `couchcmd/message_service.go` only where listener ownership/re-registration requires it.
  - Add failing tests for missing own listener, initial bootstrap failure, broker restart, conflicting foreign endpoint and lost commit reply. Repair only the wrapper's own endpoint; retain receiver state; re-register the same binding. No child restart, no new launch ledger entry, no input replay.
  - Keep repair within existing runtime lifetime, bounded retry/cancellation and inode/owner checks. Do not add durable messaging or a persistent repair service. Old wrappers retain M1's truthful compatibility notice.
  - Run targeted channel/wrapper tests and required boundary/full verification; update atlas.
  - **Operator acceptance session M3:** on a newly launched scratch wrapper, lose its message endpoint and observe recreation/re-registration; same native PID/session continues, messaging returns, no duplicate input. Contrast with pre-upgrade wrapper notice; no working-slot endpoint deletion.

Each milestone has its own fresh-context SDLC boundary review and its own scheduled operator acceptance. The next authorized scope after TL confirms this revised plan is **M1 only**. M2/M3 remain planned, not permission to implement them early. No implementation, tests, build, staging or live mutations were performed for this revision.

### 2026-10-13T21:34:22-0700 — M1 executable actions and in-flight start exception

Authority: verified TL receipts 5e7063f1-5997-4597-aa6c-6d6127feee05 and 46c487bc-ee81-4987-abbe-282a0029b2a2. This supersedes the earlier temporary M1 offer of Fresh where its execution would be refused.

- M1 shows only actions with working paths: retain the existing Relaunch/resume recovery path when Fresh requires M2 confirmation; do not offer a guaranteed-refused reboot. Candidate evidence remains readable when native resume is unsafe, but is not a selectable resume action. Restore the offered-implies-permitted regression without a deferred-Fresh exception.
- Positively listed sessions always project Continue, including an agent working on a turn or a stale start marker. A failed listing never projects Continue.
- Narrow approved M1 exception: an unfinished Couch StartClaim with no positive listing shows **Starting**, with a read-only details action. This internal claim is distinct from native agent busyness. No new persistent state or ownership mechanism.
- **M2 obligation:** a start whose owner is dead or whose age exceeds a time bound must exit Starting into Resume from disk / Start fresh through the confirmed retirement path. Do not build that rare-case execution in M1.
- The prior boundary review was interrupted after this steer; rerun the review on the corrected committed window. Operator acceptance/staging remains separately scheduled.

### 2026-10-13T21:48:00-0700 — M1 boundary corrections (BR-9 / BR-10)

The corrected-window review found a real mismatch between listed-live Continue
and the executor: cold-launch profile/path prerequisites could still refuse
warm attachment. Keep M1 scope and remove those prerequisites only from the
exact listed-session attach path, without inventing an agent profile or changing
the recorded working path. Cover the public executor and downstream launcher,
not only classification (ARCH-DRY). Cold creation keeps its existing gates.

Update README startup, attach, recovery-preview and retention instructions.
The action sweep also checks that a blank-key Fresh fallback is offered only
when that row actually offers reboot; a missing numbered-slot directory has no
such executable action. These corrections require another committed boundary
review. Starting/details and M2's bounded-exit obligation remain as approved.

### 2026-10-13T22:14:44-0700 — M1 BR-9: separate terminal attachment from lifecycle adoption

The corrected-window review confirmed the profile/path fix and README change,
but reproduced a listed-live StartClaim being refused by lifecycle cleanup.
This is the same approved Continue contract, not an expansion of the Starting
exception or authorization for M2 retirement. A listed session must attach even
when an independent supervisor owns its start or park record.

- Public Continue uses a helper-only warm attach path before replacement,
  continuation, lifecycle-gate and cleanup/claim logic. Reuse the existing
  blocked Runner, exact hosted terminal intent, launcher attach lock and final
  positive listing, bounded registration observation, and registry-owned handle.
  It does not Commit/Advance/DeleteStart, retire an incarnation, abandon a park,
  modify profiles/defaults, or acquire authority over another owner. No new
  persistent state, schema, policy file or lock mechanism.
- Recheck the selected terminal before releasing the blocked helper. Failed or
  changed listing cannot authorize attachment. Existing scoped identity checks
  remain. Reuse an already owned matching client rather than duplicate it.
- Failure, cancellation, detach and ordinary Couch leave clean only the exact
  locally owned client helper and its registry entry. They must preserve the
  session, agent, foreign helper, start/park records and thread revision.
  Explicit park remains a separate authorized lifecycle action.
- Explicit continuation recovery still needs its existing proved source/target
  and lifecycle adoption; retain that as an internal route. Merely attaching
  to view a session does not claim that a continuation source or target is valid.
- Executor tests sweep the validated record shapes already classified Continue:
  settled incarnation, active park, and unfinished start, with live/dead/unknown
  owners and zero-helper claims. Assert successful attachment and unchanged
  durable records, plus exact helper-only cleanup and failed-list no-effects.
  Cover cross-agent replacement and continuation metadata, not just the newly
  reported StartClaim. Run affected suites and the boundary again (ARCH-PURPOSE,
  ARCH-ORDER). M2 confirmation/retirement and staged acceptance remain deferred.

### 2026-10-13T22:24:06-0700 — preserve explicit lifecycle actions after Continue

The helper-only attach route must not regress ordinary Park/relaunch/switch
following Continue. Those already explicit lifecycle actions may adopt the
exact client owned by this Couch through existing start-transaction transitions,
under the existing lifecycle gate. Their read-only admission may derive the
same local-helper facts; it cannot perform adoption. Existing debris checks
still refuse retirement of a foreign/live/unknown owner. No new confirmation
protocol, persistent state or inferred-native recovery is introduced in M1.

An in-memory map of exact handles, under the existing registry mutex, supplies
local client ownership; restored registry entries cannot authorize signalling.
It is pruned on registry removal. Console routing reuses a real local pane for
repeated ordinary remote resume; if a listed row lacks a local pane, Continue
routes to warm-only resume for a fresh exact-terminal check and normal adoption
of the returned view. Cold creation is impossible on that fallback.

### 2026-10-13T22:36:00-0700 — carry listed-session authority through the whole attach path

The BR-9 correction must hold after helper acknowledgement and inside Pair as
well as at the Couch entry point. Public Continue completion and the launcher
attach-only path use the same exact positive Zellij listing, including sessions
with existing clients. They must not reintroduce lifecycle PID/socket/pane
ownership proof as an attach prerequisite. Keep the existing attach lease and
final listing; absent/failed listing cannot authorize attach, and no create
fallback is added. Cold creation and explicit lifecycle adoption retain their
own proofs. The accepted legacy list-to-attach race remains unchanged.

Production-boundary fixtures cover attached and detached sessions without owner
metadata, recorded-name override, duplicate attribution and failed listing;
executor/launcher tests must cover the complete public route too. This is the
approved M1 authority rule applied end to end, not new recovery execution.

### 2026-10-13T22:50:45-07:00 — preserve existing explicit-operation compatibility

Full-suite verification exposed consumers that had treated every Continue as
durable lifecycle adoption. Keep the new nonmutating thread contract and adapt
those consumers: explicit use still refreshes existing retention activity;
read-only inventory recognizes a proven local client without turning restored
registry entries into local ownership; checkpoint publication may record one
exact live source helper from a separate process, while execution still needs
the owning Couch's handle before adoption. Missing local slot records return
to the existing recovery route. No new recovery confirmation or M2 execution
is introduced. Existing lifecycle tests retain their ownership protection;
plain-view tests assert unchanged thread records and active holds.

### 2026-10-14 — acceptance merge and explicit hub startup clarification

Operator/TL decisions 4d827d80 and 03666e08 supersede blanket preview-only
startup and the earlier recovery delivery protocol. Every CLI hub, including
bare couch’s cwd, is an explicit request: listed live means Continue; otherwise
the normal path resumes a proven conversation or starts fresh. The #546 queued
extra-hub operation follows this same rule. Non-hub slots remain observation
only; no startup reattach sweep, interrupted-start recovery, active-park recovery
or journal replay. ARCH-PURPOSE: hub selection is an action, slot enumeration
is a read. Hub-scoped routing and mutation guards apply at preparation and effect
to candidate recovery as well as ordinary operations.

Use main #522’s single launch Prompt mechanism. Fresh prompts point to the
previous screen, branch and issue; exact retained capture coordinates remain
literal data, and screen text never enters argv. Native resumes receive no
prompt. Remove recovery delivery/status callbacks. No recovery-only mechanism.

The merge of origin/main 962f3c0478ef4b46766f6e29b4f28092b909735f is
covered by the next M2/issue-close review window, not the already closed M1.
Before staging, run the full suite through sdlc local-check run --name suite.
Operator acceptance still owns live pre-upgrade restart verification.

### 2026-10-14 — explicit attachment, independent messaging and narrow dead-client repair

TL decisions 379f4f5d, 4ad3f4a8 and b6a98c4d authorize this M1 delta.
Selecting a row, clicking it, or requesting hub Continue is explicit attachment:
one retry after a short cancellation-aware backoff, fresh positive listing at
each attempt, and a visible reason if both fail. No background reattach sweep.
A failed initial listing gets one bounded read retry; failed/absent listing
never authorizes attach. Read retry and helper retry are separate bounded steps.

Authenticated live wrappers reconnect to the broker independently of panes.
Admission resolves their retained thread/workspace and exact positively listed
terminal, retaining ready tag/agent/nonce and exact process identity checks.
Pane presence is view state, not messaging authority (ARCH-PURPOSE). Exercise
real Unix wrapper endpoints through broker restart with no pane events and
bidirectional delivery. Endpoint recreation remains M3.

On successful explicit Continue, replace only a proven-dead client-helper
incarnation with this Couch's exact locally owned attach helper. Recheck the
old PID/start identity as dead at the write gate; preserve binding, parks,
history and all lifecycle claims. Any open claim, uncertain/live old owner or
changed facts means borrowed attachment with a visible note and no record
write. Never substitute wrapper or agent PID. Use a guarded named store
transition, not lifecycle adoption that clears debris or invents a binding.

Acceptance includes the untouched pair:1 case: positively listed 📁1-217,
server 86286/wrapper 86287/agent 86303 alive, durable client helper 27163 dead.
Continue must attach and repair only that dead client; never select Fresh.
Fixtures cover changed owner identity at effect and open claims. The queued
5594eb85a12fec01e8edb73f57082d45d2987b5f suite predates this runtime delta;
the final runtime head requires its own recorded full local check before stage.


### 2026-10-14 — untouched pair:1 dead-client acceptance procedure

This is an operator/hub-run acceptance case; the implementation agent must not
attach, detach, signal, relaunch or otherwise alter the real pair:1 session.
First record a read-only baseline from the existing inventory, exact session
listing and process/readiness observations: pair:1 maps to positively listed
Zellij session 📁1-217; server 86286, wrapper 86287 and agent 86303 are alive;
its retained client helper PID 27163 is dead by exact PID/start identity.
PIDs are the observed baseline, not reusable authority: if any identity changed,
refresh the observation and do not treat these numbers alone as ownership.
Save the current record and compare its native binding, park history,
incarnations, revision and every lifecycle claim before and after acceptance.

The operator/hub selects pair:1 Continue once in the newly built Couch, then
observes the same running agent conversation and messages. Continue may replace
only the proven-dead helper incarnation with the returned locally owned helper's
exact PID/start identity; retain every unrelated incarnation, binding and park
history. It must never choose Fresh, create a native conversation, restart the
wrapper/agent/server, clear an open claim, or replay lifecycle work. Compare the
same session/server/wrapper/agent identities and native binding after completion.

If an open lifecycle claim, live/unknown old client, incomplete old identity or
changed write-gate proof prevents that narrow replacement, Continue still
attaches. The visible result says BORROWED ATTACH with the precise reason and
that retained client state was left unchanged. Verify both the console notice
and explicit hub receipt warning, and verify byte-unchanged durable state in
this borrowed case. Repeat attachment only under operator direction; repeated
local attachment must retain the accurate borrowed reason. Attachment notices
are ephemeral result data and disappear from persisted/restored registries.

Exact commands for the operator/hub (refresh changed identities first):

```sh
couch --show pair:1 --json
zellij list-sessions --no-formatting
ps -p 86286,86287,86303,27163 -o pid,ppid,lstart,command
cp /Users/xianxu/workspace/worktree/pair-slot1/.couch/thread.json /tmp/pair528-pair1-before.json
```

Then either select Continue in the row, or run the hub action:

```sh
couch --resume pair:1 --json
couch --operation-status RETURNED_OPERATION_ID --json
couch --show pair:1 --json
couch --peek pair:1 --lines 20
```

Read the returned receipt to completion before repeating any action. Compare
/tmp/pair528-pair1-before.json to the slot's thread.json: only the sole client
incarnation's PID/Identity/State and record Revision may change; StartedAt,
launch profile, session binding/nonces, parks/history, capture/native evidence
and all claims stay equal. With an open claim or failed death proof, the whole
record stays equal and the receipt/console show the borrowed reason. These
commands are acceptance instructions, not measurements run by this agent.
