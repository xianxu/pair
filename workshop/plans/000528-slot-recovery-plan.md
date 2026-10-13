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
