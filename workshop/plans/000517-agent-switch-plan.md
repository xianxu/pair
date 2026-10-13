# Agent-guided switching and previous-session access Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Switch a slot's coding agent without losing access to its previous session, delivering a colleague-written brief unchanged or starting an idle slot fresh.

**Architecture:** Reuse Couch's authenticated broker, operation queue, park/start transaction, native-session resolver, retention leases, and wrapper orientation delivery. Add a pure asynchronous switch coordinator and one durable previous-session descriptor. The form and remote CLI use that same coordinator; recording replay stays in a short-lived CLI, never the supervisor.

**Tech Stack:** Go, existing Couch socket protocol and thread store, Pair terminal replay, Markdown procedures.

## Evidence and scope

Measured 2026-10-12, metadata only: 436 parked raw files, 436 events files,
286 capture metadata files, 70 live raw files; latest parked raw approximately
9.4 MB. No transcript contents or live session operations were needed.

`peek_fast.go` serves live tails up to `couchmessage.MaxTailLines` (200).
Larger requests already fall back to the current recording. No path selects a
previous incarnation. `OSSwitchContextResolver.Resolve` identifies the exact
current native transcript before park; `ResolveArchive` resolves the verified
park's exact capture. `starttransaction.go` clears `VerifiedPark` on successful
start. Thus neither a latest-file glob nor keeping the current peek is sufficient.

The original `scrollbackcmd.RenderLines(..., 0)` relaxed only the history row
limit; M1 review found hidden cell/byte caps. Reuse the shared replay with
explicit limits and loss detection for full reads. Replay remains in the
short-lived CLI, one recording per invocation. The local VT reply-writer API
allows direct reply discard and emulator closure without a drain goroutine.
No new transcript parser or summary generator.

pair#414 currently contains only its title (Grok attach refused, orientation
lost), no reproduction. Treat it as a regression target: exercise actual launch
registration and delivery, not only request serialization; retain partial failure
evidence instead of reporting a started process as a delivered handoff.

## Design and alternatives

Recommended: preserve exact evidence, let the outgoing agent author the brief,
and coordinate the asynchronous response in Couch. This uses the existing
park/start and delivery contracts. A form that waits on an arbitrary file then
asks for another click adds file polling and was rejected by the TL. Continuing
the generated orientation prompt would fail the requested handoff behavior.

CLI contract:

```text
couch --switch-agent repo:N --agent NAME [--handoff-file PATH] --confirm [--json]
couch --switch-agent self --agent NAME --handoff-file PATH [--request ID] --confirm [--json]
couch --peek repo:N --previous [--lines N] [--transcripts] [--json]
couch --peek repo:N [--previous] --all [--transcripts] [--json]
```

`--switch-agent` is heavy. Ordinary live peek remains light; transcript discovery
is seconds, and `--all` is heavy because it replays a recording. Full reads accept
exactly one slot; `--all` conflicts with `--lines`. `--previous` never falls back
to current evidence. Missing paths or old records lacking a descriptor say
unavailable. JSON and text name the source incarnation and raw/events paths.

The handoff file is read once by the CLI, at most `orientation.MaxBodyBytes`,
validated as nonempty UTF-8 without terminal controls other than LF/TAB, and sent
as bytes in the request. Preserve whitespace and the final newline exactly; do
not reuse `readMessageFile`, which removes a trailing newline. Refuse CRLF with
the path and actionable error rather than silently rewriting a verbatim brief.
No server-side file-path read, shell interpolation, or generated wrapper around
the authored content. The brief points to `--peek SLOT --previous --transcripts`
for stable archive discovery and names the native transcript when available.

No file on a clean resting branch means a fresh start with no first message.
No file for a non-resting slot means request its outgoing agent's brief. The
form uses this same choice and requires no second click. A supplied file means
use that brief. Dirty or unknown targets refuse before requesting or parking.
The TL can author a brief for a supervised slot; other peers cannot impersonate
the source's self callback.

Self admission returns an acknowledgment before waiting, so the initiating
agent can end its turn. It is bound to the authenticated current scope, tag,
terminal session and launch nonce; callback and execution must still match.
The form/TL brief request includes its opaque parent switch ID and the exact
callback command with `--request ID`. That optional flag is accepted only for
self callbacks with a handoff file. Callback lookup requires BOTH that ID and
the authenticated source binding to match a waiting-brief request. Target agent
must match the original choice; callback content cannot change its argv. A
missing, expired, cancelled, or already-fallback ID refuses, even if the same
incarnation has a newer pending request. Duplicate identical callbacks return
the same acknowledgment; conflicting content refuses. Standalone self calls
omit `--request` and create a new request only when the target has no pending
switch. They never implicitly answer an existing request. The request ID is
correlation, not authorization: every callback still checks broker identity.
An external request initially refuses busy/unknown targets. After Couch asks
for a brief, that exact source may run its handoff-writing turn. The coordinator
waits outside the operation queue, then rechecks settled identity and clean Git
state immediately before enqueue and at the park effect.

Default brief timeout: 3 minutes; default overall settle limit: 10 minutes from
admission. Proposed configuration: `COUCH_SWITCH_BRIEF_TIMEOUT` and
`COUCH_SWITCH_SETTLE_TIMEOUT`, parsed once at startup, positive durations with
brief < settle and settle <= 30 minutes. The TL confirmed that the first limit
selects pointers-only fallback content, never permission to interrupt. At the
second limit, refuse without parking and explain on the notice line. Late
callbacks cannot replace the chosen fallback or retarget a new incarnation.

Fallback is only a neutral evidence pointer message: previous agent/session,
native transcript and exact tty discovery command. It includes no invented
work summary. Authored brief delivery and fallback are distinguished in receipt
and notice text. A fresh idle launch sends no orientation request at all.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| PreviousSession | `cmd/internal/threadrecord/previous_session.go` | new |
| AgentSwitch | `cmd/internal/couchcore/agent_switch_model.go` | new |
| SwitchContent | `cmd/internal/couchcore/agent_switch_model.go` | new |
| PeekSelection | `cmd/internal/couchcore/peek_selection.go` | new |
| History capacity-loss counter | `third_party/vt/scrollback.go` (`Emulator.HistoryCapacityLoss`) | new |
| SwitchAgentRequest | `cmd/internal/couchcore/switchagent.go` | modified |
| Request | `cmd/internal/couchmessage/protocol.go` | modified |

`PreviousSession` is one bounded descriptor per thread: source terminal binding,
agent, native session ID, native transcript paths, capture token/events flag,
capture timestamp, and unavailable reasons. Store it before target launch;
successful launch retains it. A second successful source park supersedes it.
Failed target launch/retry preserves the same source; it never overwrites it
with the failed target's empty session. Existing records without it remain valid.
Validate lengths, absolute transcript paths, source-address agreement, and the
capture token using existing artifactpath validation; clone slices/pointers in
both core/persisted conversions. No arbitrary raw path is trusted as an archive.

`AgentSwitch` owns a single source-bound request and its transition function.
Its states, source binding, selected content and deadlines are private; callers
execute effects and return events rather than mutating its fields. One pending
switch per target, using the existing 64 receipt capacity. `SwitchContent`
distinguishes fresh, authored, and fallback; empty content cannot mean all three.
`PeekSelection` distinguishes current and previous evidence and full/tail reads.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Previous-session persistence | `cmd/internal/couchcore/previous_session.go` (new), `thread.go`, `switchagent.go` (modified) | new/modified | strict thread store, revision checked writes |
| Switch coordinator | `cmd/internal/couchcmd/agent_switch.go` | new | broker identities, clock, Git/settled probe, operation queue |
| Slot request transport | `cmd/internal/couchcmd/cli.go`, `messages.go`, `slot_operations.go` | modified | CLI file read, socket requests/receipts |
| Exact capture reader | `cmd/internal/scrollbackcmd/bounded.go` (new), `scrollbackcmd.go` (modified) | new/modified | unchanged retention lease API and shared replay |
| Peek projection | `cmd/internal/couchcmd/run.go`, `cmd/internal/couchcore/peek.go` | modified | short-lived read-only CLI; existing fast peek remains unchanged |
| Form handoff request | `cmd/internal/couchtty/menu_switchagent.go`, `console_remote.go` | modified | same coordinator and broker delivery seam |
| New-agent delivery | `cmd/internal/couchtty/console_switchagent.go`, `cmd/internal/wrapcmd/orientation.go` | modified | existing guarded paste/submit and ready records |

Use the existing fake Couch environment/runner, portable thread store, fake
process identities and broker sessions. Extend them with controllable settled
observations, current incarnation changes, delivery replies, and a fake clock.
Tests use temp homes/data roots and a real temporary Unix socket where transport
matters. No test can discover or restart real operator sessions.

## State and failure contract (ARCH-ORDER)

| State/event | Next state and effects |
|---|---|
| Admit external, unknown/busy/dirty/stale source | refused; no send, park or launch |
| Admit clean resting source without brief | waiting-settled with fresh content |
| Admit source with authored brief, including verified self | waiting-settled; acknowledge immediately for self |
| Admit non-resting source without brief | waiting-brief; send one agent request and arm deadlines |
| waiting-brief, matching request-ID and authenticated-source callback | waiting-settled with immutable authored bytes |
| waiting-brief, brief deadline | waiting-settled with pointers-only content; publish fallback notice |
| waiting-settled, same source settled and clean | switching; enqueue one existing switch transaction |
| waiting states, source identity changes or process dies | refused; never switch the replacement |
| waiting states, overall deadline or cancellation | refused; drop pending request, no destructive effect |
| switching, park succeeds | preserve exact previous evidence before launch |
| switching, start failure/uncertainty | failed/unconfirmed receipt with parked evidence and recovery advice |
| switching, registration succeeds | delivery pending (or success immediately for fresh content) |
| delivery pending, wrapper confirms submitted | succeeded; release pending resources |
| delivery pending, partial paste/input/exit/timeout | unconfirmed; retain manual-copy evidence, never resend automatically |
| any state, duplicate request | return original receipt; no second effect |
| terminal state, callback/completion for old request | ignore/refuse, no state resurrection |

The coordinator is supervised by Couch's lifetime, wakes through events/timers,
and never sleeps while holding the global console/operation queue. Server exit
cancels pending work; existing in-memory receipt semantics report unknown after
restart. The durable previous descriptor survives. A new request revalidates
the current source; it never resumes a stale pending command automatically.

## Operating envelope and trust

- **ARCH-DRY/PURE:** one switch coordinator for CLI and form, one content
  validator, one previous selector, existing source resolver and park/start.
- **ARCH-PURPOSE:** previous native transcript AND full tty access, authored
  brief, idle fresh start, form, TL procedure, and live feel approval all ship.
- **ARCH-MOCK:** fake time and stateful source/replacement sessions reproduce
  races; live Claude/Codex/Grok checks are scheduled through the TL.
- **ARCH-CONSTRAINTS:** no replay in Couch, no fan-out for full reads, bounded
  16 KiB brief, one pending switch per slot and at most 64 globally. Replay
  remains O(raw bytes + rendered terminal cells), not cheap pagination.
  Proposed initial full-read safety envelope is 64 MiB raw, 8 MiB events,
  100,000 rendered rows and at most 1,000 columns/rows per resize. Following
  M1 review, also cap screen area at 262,144 cells and retained history at
  1,048,576 cells / 128 MiB logical bytes. Check file
  size/dimensions before replay and detect row overflow with a capped history
  plus one; detect every capacity eviction/rejected history row with the VT
  history loss counter, including loss before an explicit erase. Refuse visibly
  rather than return a truncated result as complete. Construct and resize the
  emulator through its checked APIs; intentional history erase is not loss.
  These are design assumptions to confirm in review; 9.4 MB is measured input,
  not a replay latency measurement. Oversize errors still name exact source
  files. Native discovery keeps its existing two-second budget.
- **ARCH-SECURE:** broker binds self to the live caller; target binding rechecked
  at effect time. Typed strict persisted decoding refuses malformed descriptors.
  Regular-file and owner-scoped capture selection avoid FIFOs/symlink redirects.
  No tokens minted; handoff/native paths are local private context, not network
  publication. Historical contents are evidence, not new operator instructions.
- **ARCH-FUNERAL:** one previous descriptor replaces one old descriptor, at most
  32 KiB/thread. Existing parked captures retain their existing owner-level
  retention policy (visible owners protected; inactive eligible after its grace).
  Leases close on CLI exit; pending coordinator state dies on terminal outcome
  or Couch exit; operation receipts expire under existing retention. Couch reads
  but never deletes the user's handoff file; the TL procedure uses a temporary
  file and removes it after acknowledged acceptance.

## Named function test strategies (PQ-2)

| Production function | Adversarial strategy and mechanical guard |
|---|---|
| `PreviousSession.Validate` | Fuzz decoded descriptors with malformed identities/paths and size growth; strict shape, source-address equality and existing token validation refuse before IO. |
| `ClonePreviousSession` | Mutate every reachable reference in the clone; original descriptor must stay unchanged. |
| `ThreadStore.RecordPreviousSession` | Sequence revision-checked writes through real temp stores and failed target starts; exact source binding and revision prevent replacement or rollback of retained evidence. |
| `SelectPeekSource` | Vary current/previous generations independently; previous selection can consume only its saved descriptor and never current resolver output. |
| `Couch.PeekThreadSelection` | Stateful fake source/readers vary availability across selection; chosen source identity stays fixed and full reads bypass live tail. |
| `RenderBoundedLines` | Fuzz event dimensions/offsets and vary input/row bounds with small fixtures; admission bounds precede allocation and overflow yields an error, never a falsely complete tail. |
| `RenderOwnedCaptureLines` | Real temp retention store with interleaved lease/collection; exact selected-owner lease protects capture and releases on every outcome. |
| `ValidateSwitchContent` / `readHandoffFile` | Fuzz arbitrary file bytes with bounded nonblocking regular-file IO; reject invalid content and require exact accepted byte equality. |
| `AdvanceAgentSwitch` | Generate/replay event sequences with duplicate, late and cross-request events; source plus request-ID correlation and private state enforce at-most-once switching. |
| `agentSwitchCoordinator.Admit` / `Callback` / `Observe` | Stateful broker/clock/queue fake controls observation-to-effect races; immutable accepted identity and revalidation prevent replacement switches. |
| `couchmessage.ValidateRequest` / `parseCLI` | Fuzz switch fields mixed with other verb shapes; closed per-operation fields reject ambiguous or unauthorized callback forms. |
| `Couch.SwitchAgent` | Existing stateful runner and lifecycle fake interrupt park/start/registration; retained partial evidence and exact identity govern recovery. |
| `reduceSwitchAgentPreview` / `finishOrientation` | Replay form/operation/delivery events in competing orders; one coordinator and producer/request identity prevent stale completion or focus theft. |
| `AdvanceDelivery` and wrapper delivery integration | Existing captured composers plus actual launch envelope transport; byte equality and matched ready records distinguish registration, paste and submission. |

## M1 — Previous-session evidence and full recording reads

Files: `threadrecord/previous_session.go` and `_test.go` (new),
`threadrecord/record.go`, `couchcore/thread.go`, `couchcore/couch.go`,
`couchcore/previous_session.go` and `peek_selection.go` (new),
`couchcore/switchagent.go`, `couchcore/peek.go`, `couchcore/ops.go`,
`couchcore/operationdispatch.go`, `couchcmd/cli.go`, `couchcmd/run.go`,
`scrollbackcmd/bounded.go` (new), `scrollbackcmd/scrollbackcmd.go` and relevant tests.
All paths are under `cmd/internal/`. Register new source files and the newly
resolved recording consumer in `artifactpath/manifest.go`'s exhaustive inventory.
The bounded renderer also consumes a monotonic capacity-loss counter added to
the local VT module's scrollback implementation, with colocated module tests.

- [ ] Write failing tests using the function strategies below for descriptor
  persistence and previous/full selection.
- [ ] Run `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'TestPreviousSession|TestPeekPrevious' -count=1` and confirm the intended failures.
- [ ] Add typed descriptor, clone/validate/persist it from exact source and park
  evidence before launch; reuse record revision checks. No timestamp globbing.
- [ ] Add full-read/previous CLI parsing and exact parked capture lease reader;
  expose paths and source binding in JSON/text, derive help cost tags. Preserve
  ordinary current peek behavior and the fast path.
- [ ] Test the bounded replay and lease reader using small deterministic
  captures and the strategy below, never load tests.
- [ ] Run `go test ./cmd/internal/scrollbackcmd ./cmd/internal/couchcmd ./cmd/internal/couchcore -run 'Test.*(Peek|PreviousSession|Render|SwitchAgent)' -count=1`.
- [ ] Update README and atlas/couch.md, log measured limits, commit with #517 M1,
  and run `sdlc milestone-close --issue 517 --milestone M1` with evidence.

## M2 — Authenticated asynchronous switch protocol

Files: new `cmd/internal/couchcore/agent_switch_model.go` and `_test.go`, new
`cmd/internal/couchcmd/agent_switch.go` and `_test.go`; existing
`couchmessage/protocol.go`, `couchcmd/cli.go`, `messages.go`, `slot_operations.go`,
`message_service.go`, `couchcore/switchagent.go`, `slot_operation.go`, `ops.go`,
`operationdispatch.go`, `couchtty/console_remote.go` and corresponding tests.

- [ ] Add model and coordinator sequence tests against the function strategies
  below before implementation.
- [ ] Run targeted model tests red; implement the private transition-owned
  model and fake-clock coordinator, then rerun green.
- [ ] Extend closed wire shapes with switch-only agent/content fields; reject
  those fields on unrelated verbs. Resolve self from broker caller identity,
  never untrusted target strings. Bind external target at admission too.
- [ ] Test the CLI read/validation and closed flag shapes against the named
  strategies below.
- [ ] Route CLI/form requests through the shared coordinator and existing
  receipt/queue mechanism with PreserveFocus. Self gets acknowledgment before
  its turn ends; no waiting CLI child keeps the source busy. Expose operation
  status to the supervising TL without requiring the departed incarnation.
- [ ] Change the core switch to accept explicit fresh/authored/fallback content;
  remove generated orientation from normal switching. Recheck settled and source
  identity at park. Carry partial outcomes through the receipt.
- [ ] Extend socket acceptance and existing switch transport test to assert
  byte-for-byte first-message content, no orientation for fresh, and separate
  registered versus submitted outcomes. Ensure work on another slot retains focus.
- [ ] Run `go test ./cmd/internal/couchmessage ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(AgentSwitch|SwitchAgent|SlotOperation|Remote|Protocol)' -count=1`.
- [ ] Update README/atlas, checkpoint Log, commit #517 M2, close M2 with evidence.

## M3 — Form, TL procedure, and live acceptance

Files: `cmd/internal/couchtty/menu_switchagent.go`, `console_switchagent.go`,
associated tests; Couch help/skill text; README.md; atlas/couch.md;
`/Users/xianxu/workspace/ops/skills-draft/xx-tl/SKILL.md` (only authorized ops edit).

- [ ] Add failing form tests against the reducer strategy below. Connect the
  form to the M2 coordinator; keep argv preview and stale-fingerprint
  protections. Do not busy-wait inside the operation queue.
- [ ] Test the new notice and manual-copy recovery using existing delivery
  state; don't label registration or partial paste as confirmed handoff.
- [ ] Use skill-writing instructions before editing the TL skill. Add when to
  switch, commit/brief checklist, previous transcript and tty lookup commands,
  exact switch calls, receipt checks, new-agent pickup verification and temporary
  file cleanup. Ask TL to review that section; preserve unrelated ops edits.
- [ ] Run targeted wrapper captured-composer tests for Claude/Codex/Grok:
  `go test ./cmd/internal/wrapcmd -run 'TestOrientation' -count=1`.
- [ ] Coordinate suite capacity with hub; run required relevant checks and
  `git diff --check`. No more than two full suites machine-wide.
- [ ] TL stages the branch and obtains a fresh Couch/wrapper where needed.
  Schedule an idle-to-Codex switch, mid-issue authored brief switch, and
  Claude/Codex/Grok registration/delivery checks using authorized spare targets.
  Read previous transcript and full tty; verify selected identity and first
  message. Never switch or focus an operator slot without the TL's live-test plan.
- [ ] Operator approves handoff feel through TL. First rollout uses the paused
  ariadne #230/#282/#305 slots, with their owners' committed briefs; timing is
  TL-controlled. Record actual observations and any #414 reproduction/fix.
- [ ] Close M3, then issue with measured actuals and verification. On TL SHIP,
  merge latest main, revalidate, run `sdlc pr` and `sdlc merge --yes` (never
  `--no-validate`), report PR/merge to hub, and stop.

## Approval and outstanding choices

The TL confirmed no-second-click self flow, identity-bound callbacks, two
timeouts preserving the busy guard, and exact previous-descriptor/full-read
architecture. Final plan review must confirm the proposed timeout defaults and
full-read operating envelope before implementation. Estimates are derived only
after the `sdlc change-code` plan-quality gate passes.

## Revisions

- 2026-10-12 — M1 round 2 BR-4/BR-5 exposed hidden VT history limits and
  a geometry mismatch. Extend the explicit replay envelope with screen area
  262,144 cells, history 1,048,576 cells and 128 MiB logical bytes; preserve
  existing byte/row/dimension limits. Add a minimal monotonic capacity-loss
  counter to the local VT scrollback API so eviction or rejected rows cannot
  be concealed by later terminal erase-history commands. Use error-returning
  emulator construction and resize. Tests cover cell and byte loss separately,
  initial and later oversized geometry, and intentional erase versus lost
  history. Bounded event decoding also refuses malformed resize data instead
  of silently dropping it; ordinary viewer replay keeps its tolerant policy.
  Use VT's existing reply-writer seam to discard replies synchronously and
  close the emulator, replacing the obsolete drain-goroutine assumption.
  This is required to fulfill complete-or-visible-refusal semantics,
  not a change to the handoff workflow. Run VT's nested-module tests separately.

- 2026-10-12 — M1 boundary review BR-1/BR-2/BR-3: corrected the core-concepts
  and M1 file map to the implemented owners (previous_session.go,
  peek_selection.go, bounded.go and run.go); unchanged threadstore.go,
  retention.go and peek_fast.go are reused dependencies. Evidence reuse is
  valid only when its source matches the current verified source park;
  separately parking a newer incarnation must supersede the old descriptor.
  Add that sequence and controlled exact-capture lease/collection tests before
  M1 closes. These corrections preserve the approved scope and limits.
  The full suite also exposed missing source classification entries; register
  the new files and run.go's resolved-path consumer in artifactpath/manifest.go.

- 2026-10-12 — TL approved implementation (receipt
  `ebf27554-f0e7-4d4e-8b00-2b46b83c529f`), including three milestones,
  timeout defaults, full-read envelope and transition table. M3 procedure must
  explicitly tell the TL to ask a dirty source to commit first, or use its self
  path (which commits before switching). The TL schedules live checks; local
  Claude/Codex/Grok checks remain targeted. The approval section above records
  the questions presented for review; those choices are now approved.

- 2026-10-12 — plan-quality PQ-1/PQ-2: callback correlation now carries the
  exact parent request in `--request ID`, with standalone/self-callback
  admission explicitly distinct and late/duplicate handling defined across
  form, CLI, broker and coordinator. Compressed prose case inventories into
  named production functions with adversarial strategies and mechanical guards.
  The scope and operator flow are unchanged.


### 2026-10-12 21:29 PDT — M2 projection and recovery review; approval pending

**Reason:** ops:0 relayed the operator's concern about complex internals hiding
unrecoverable outcomes, then clarified that internal states may remain as complex
as the mechanics require (receipt `5a859539-2444-44a0-b156-6402cfb3227a`).
This revision supersedes the user-facing interpretation of the earlier table,
the 64-switch admission limit, and the assumption that switch recovery can be
entirely ephemeral. M1 stands. M2 implementation is paused pending TL approval.
Work already begun under the earlier approval is preserved, explicitly unreviewed,
at `f756c2bf9`; it does not implement this revision.

**Count:** retain 8 internal states; project them to 5 visible outcomes. This is
not a cosmetic renaming: all terminal failures carry one selected recovery
operation, and the same projection supplies the slot row, its primary action,
notice and CLI receipt. Internal step names belong only in diagnostic details.

| Internal state and observed evidence | User-visible outcome | One next action |
|---|---|---|
| waiting-brief, waiting-settled, switching, delivery-pending; operation still within its deadline | Switch in progress (short reason: waiting for handoff / waiting for turn / starting agent) | Wait; the deadline produces one of the outcomes below |
| refused, or failed before stopping the exact source; old agent verified live | Old agent still running | Retry switch |
| failed/unconfirmed with old source stopped and no verified target, incomplete park, or process evidence not yet sufficient to select a survivor | Switch needs recovery | Recover slot |
| unconfirmed delivery with exact new agent verified live | New agent running; handoff needs attention | Copy brief |
| succeeded with exact new agent registered and either fresh content or submitted brief | New agent ready | Continue |

`Recover slot` is ONE operation, not instructions to choose among retry/recover/
abandon. It reconciles the saved exact source and target first, reattaches a
verified survivor, or resumes the saved old thread only after proving no target
is running. An unknown process never authorizes a second launch. On success the
row returns to the corresponding live-agent outcome. On unavailable external
resources it stays visibly recoverable with the same action and a concrete error;
it never becomes a spinner with no exit. `Copy brief` copies the exact saved
bytes; the notice says to inspect any text already in the composer first. It does
not paste or submit again automatically.

The failure rows for the **existing park-then-start architecture** are honest:

| Failure | What remains in the slot | One recovery action | Where shown |
|---|---|---|---|
| Brief never arrives | Old agent remains running until settled; at brief deadline select neutral pointers, then proceed. If the overall deadline expires, old agent still runs. | Retry switch after an overall timeout | Slot row primary action, persistent notice, operation-status |
| Agent stays busy | Old agent still running; never parked by the timeout | Retry switch after its turn ends | Same three surfaces |
| Park fails before source exit | Old agent still running if exact liveness confirms it; no target launched | Retry switch | Same three surfaces |
| Park fails after source exit or completion is uncertain | Old thread may be stopped; target has not started. This is a real gap in the current design. | Recover slot (reconcile, then resume old thread) | Same three surfaces; survives Couch restart |
| Target start fails | Old thread is parked. A partially started target is reconciled before resuming anything; neither agent may currently run. | Recover slot (reattach surviving target, otherwise resume old thread) | Same three surfaces; survives Couch restart |
| Delivery partial/timeout | New agent still running if verified live; retain exact brief and never auto-resend. If it exited, use the recovery row above. | Copy brief | Target row primary action, persistent notice, operation-status |
| Couch restarts mid-switch before park | Old agent remains in its Zellij session; waiting request is cancelled, not replayed | Retry switch | Rebuilt slot row and recovered receipt |
| Couch restarts after park, before target is verified | Old may be stopped and new may be absent or alive; do not guess from a lost receipt | Recover slot | Rebuilt slot row and recovered receipt |
| Couch restarts with verified target but no confirmed delivery | New agent running; exact saved brief available, no automatic repeat submission | Copy brief | Rebuilt slot row and recovered receipt |

**Decision needed, not a false guarantee:** these rows expose two cases that do
NOT satisfy the earlier literal requirement that one agent always be running.
Automatic resume after a failed start cannot guarantee that either: resume can
also fail. The projection above eliminates hidden limbo but cannot turn a stopped
process into a live one by labeling it. Approval must explicitly choose whether
visible, one-action recovery is the intended invariant, or uninterrupted agent
availability is mandatory.

If uninterrupted availability is mandatory, the current park/start transaction
must be replaced: keep the old session alive while a replacement registers in a
separate candidate session; publish a durable choice of active session before
retiring the old one. Candidate startup failure then leaves old running; park
failure leaves the registered replacement running; partial delivery leaves new
running; supervisor restart reconciles the durable choice and surviving sessions.
This requires a separate candidate identity and exclusive input ownership (so two
agents cannot accept work in the same slot), plus restart reconciliation. Current
`launchTrackedThread` binds launch to the one thread identity and its artifact
paths, and `CommitStartClaim` refuses an occupied thread: launching first is NOT
an available reorder of today's calls. It is a substantial scope change that
needs approval and its own executable design, not an untested rollback promise.
Host failure or independent death of both agents still cannot be covered by an
absolute process-availability promise. No such scope change is being built now.

**Concrete delta proposed for the one-action recovery interpretation:**

- Keep the honest internal transition model (ARCH-ORDER). Add pure
  `ProjectAgentSwitch` in `cmd/internal/couchcore/agent_switch_model.go`, returning
  one of the five outcomes above and its sole action from exact liveness and
  transaction evidence. CLI and form consume it (ARCH-DRY, ARCH-PURE).
- Remove the switch-specific/global 64-admission policy. One pending switch per
  canonical slot is the admission rule. Keep one latest switch result per slot,
  superseded by that slot's next request; existing unrelated operation admission
  limits stay owned by their existing code. No new queue-size knob or arbitrary
  global switch count (ARCH-CONSTRAINTS).
- Before park, save one validated `SwitchRecovery` record with request/source
  identity, old launch/resume evidence, accepted target parameters and exact
  selected content. Reuse the thread store's atomic, revision-checked writes;
  `cmd/internal/threadrecord/switch_recovery.go` and
  `cmd/internal/couchcore/switch_recovery.go` own conversion/reconciliation.
  Never infer completion from receipt loss. Replace this record at the next
  admitted switch only after prior recovery/cleanup is complete; drop with its
  thread. One record per slot, bounded body by the existing orientation wire
  contract (ARCH-SECURE, ARCH-FUNERAL).
- `cmd/internal/couchcore/switchagent.go` records destructive progress before
  returning; the recovery operation uses existing park reconciliation, process
  identity and resume services. `agent_switch_service.go` rebuilds the visible
  result on supervisor startup. Recovery never sends a brief implicitly.
- Add controlled fake-runner/store/session tests interrupting every destructive
  boundary, including target startup followed by lost acknowledgment. Assert
  actual surviving process identities, a visible row and exactly one recovery
  action. Restart against the same portable store and prove repeated recovery
  does not launch a duplicate or submit the brief twice (ARCH-MOCK/PURPOSE).
- Test the projection exhaustively across all 8 internal states and known/unknown
  process evidence. Exercise the actual menu action and CLI receipt, not just
  serialization. Register new paths and update atlas at the M2 boundary.

This revision is a reviewable design choice, not implementation approval or a
claim that the no-empty-slot requirement has been met. Wait for the TL's answer
before elaborating the selected recovery design, rerunning plan-quality and
resuming M2. The existing estimate must be revisited after the chosen design
passes that gate; it is not silently reused for a larger launch transaction.


### 2026-10-12 21:34 PDT — Approved recovery invariant and executable M2 delta

TL approval `60b20071-4145-4fd1-a177-11240322a715` selects durable one-action
recovery, not the larger candidate-launch transaction. The preceding decision
is settled. Its 8 → 5 projection and failure rows are approved. Additional
conditions: recovery is idempotent; its own failure retains an action; unknown
presence says what could not be proved and offers **Resume old thread** as the
fallback; restart surfaces **Switch needs recovery** automatically as the row's
primary action plus a notice. That fallback invokes the SAME proof-preserving
recovery operation: it is never permission to start over an unknown live target.
Each failure row must have a test of its rendered outcome and action.

**WIP audit before reuse:** `f756c2bf9` has reusable source-authenticated CLI,
verbatim file validation, acknowledgment-after-write barrier, model sequencing,
fresh-launch nonce independent of orientation, and console delivery/admission
hooks. It does not satisfy this recovery contract. Its ephemeral receipts,
global switch cap, internal-phase-only presentation, missing persistence and
ordinary form refusal for parked sources are not accepted as finished behavior.
No M2 boundary is closed. Keep the tested transport/core pieces only after the
new tests prove them in this flow.

The audit found two ordering requirements beyond a cosmetic projection:
(a) initial admission must be durable BEFORE acknowledgment, so interrupted
waiting requests also have an outcome after restart; (b) partial registration
can change SessionBinding and successful registration changes the launch profile,
so recovery must retain the original profile and exact native ID independently.

#### Additional core concepts and owners

| Name | Lives in | Status |
|---|---|---|
| SwitchRecovery (validated durable source/target/content snapshot) | `cmd/internal/threadrecord/switch_recovery.go` | new |
| AgentSwitchView / ProjectAgentSwitch | `cmd/internal/couchcore/agent_switch_view.go` | new |
| ThreadRecord / Record | `cmd/internal/couchcore/thread.go`, `cmd/internal/threadrecord/record.go` | modified |
| ActionableThreadSummary | `cmd/internal/couchcore/actionableinventory.go` | modified |
| OperationReceipt | `cmd/internal/couchmessage/operation.go` | modified |

| Integration | Lives in | Status | Reuses |
|---|---|---|---|
| Persist and recover saved switch | `cmd/internal/couchcore/switch_recovery.go` | new | ThreadStore CAS, exact session/process observations, existing park/start/resume |
| Persist target and restored-source witnesses atomically | `cmd/internal/couchcore/starttransaction.go`, `threadstore.go` | modified | existing session-bound/helper-recorded/registered events |
| Saved-native-ID resume | `cmd/internal/couchcore/resume.go` | modified | existing cold resume pipeline with pinned native-ID equality |
| Surface recovery before ordinary actions | `cmd/internal/couchcore/recover_action.go`, `slot_operation.go`; `cmd/internal/couchtty/menu_actions.go`, `menu.go`, `menu_render.go` | modified | existing Recover and menu action dispatch |
| Restart-visible recovery and exact copy | `cmd/internal/couchtty/console_menu.go`, `console_switchagent.go`, `menu_reattach.go` | modified | inventory refresh and copy action; no automatic submission |
| Durable admission and status | `cmd/internal/couchcmd/agent_switch_service.go`, `agent_switch.go`, `message_service.go`, `slot_operations.go` | modified | broker auth, existing receipt schema and queue |

`SwitchRecovery` keeps one request ID and admission owner per thread, timestamps,
immutable source process/session/native ID/profile, accepted target profile,
selected content (including explicit waiting-for-brief before selection), exact
launch attempt/binding/helper witnesses, last outcome and error. A restoration
attempt is distinguished from the original target attempt. Use tagged outcomes,
not independent booleans that imply undocumented combinations. Validate the
record on decode, constrain identity/address relationships, clone every nested
slice/pointer, and reuse the existing orientation body contract. Preserve old
record compatibility. No recovery record is process-liveness authority.

The source snapshot is retained until a later resolved switch supersedes it or
the thread is deleted. Unresolved recovery blocks replacement by a new request.
Recovery uses exact source and target session observations; a current slot index
or latest native transcript cannot prove either one absent. Successful recovery
records its selected survivor so a repeat reattaches or returns it. Unknown
presence stays in visible state 3 with Resume old thread and concrete diagnostic;
that action rechecks rather than bypasses proof. Source native-ID equality is
checked again immediately before child effects, including same-agent switches.

#### Ordered M2 implementation and verification

- [ ] Pure data and projection: add strict durable-record validation and clone
  tests; add ProjectAgentSwitch table tests for every approved failure row,
  all eight internal phases, unknown presence, and failed recovery. Then implement
  the record and shared five-outcome projection. Keep diagnostic detail separate
  from its single primary action. Run threadrecord and projection targeted tests.
- [ ] Admission: reserve one canonical slot request, write its initial recovery
  record before acknowledgment, then activate only after successful response
  write. A failed durable write authorizes no park. Preserve immutable argv;
  after the admission CAS updates revision, return a renewed fingerprint only
  after comparing the original accepted configuration. Persist callback/fallback
  content before it can authorize switching. Tests inject write failures,
  duplicate/colliding admissions, stale callbacks and restart before park.
- [ ] Switch transaction: checkpoint selected body and target attempt before
  ParkExpected. Atomically copy the target binding/helper/registration witness
  from existing start events into the recovery record; do the same for an
  authorized restoration attempt. Persist submitted delivery only on actual
  wrapper confirmation; write failure leaves a recoverable/copyable result,
  never fabricated success. Tests kill the coordinator between each existing
  durable step and completion callback, then reload the same store.
- [ ] Recovery: add RecoverAgentSwitch under the existing reentrant thread gate;
  dispatch generic Recover to it before ordinary reboot/reap policy. First prove
  exact survivor/absence. Reuse warm-only attach for a live detached survivor.
  Reconcile a matching open park only after selecting the safe recovery path;
  absent-source/missing-completion must either reconcile with proof or retain the
  same visible action, never tombstone the source to make progress. Restore the
  saved source profile/native identity only after proving target absent, then use
  the existing resume pipeline with required native-ID equality. Tests cover two
  consecutive recovery calls, crash during recovery, stale request IDs, same-agent
  targets, unknown process/session, changed native binding and recovery's own
  failure. Assert actual launch count and exact agent/native ID, not only return
  codes. A missing old native session remains visibly actionable with the exact
  reason; no silent fresh-context substitution.
- [ ] Restart: skip ordinary automatic park/re-attach for unresolved switch
  records. Project their current evidence into inventory immediately, show the
  primary recovery action and a persistent notice on the first refresh, and
  restore copyable exact content without starting a delivery watch or submitting.
  Tests render each failure row after a simulated supervisor restart and activate
  its primary action through the real menu reducer/dispatcher.
- [ ] Receipts: latest result per canonical slot, one pending per slot; remove
  switch dependence on the global 64 generic-operation admission cap and
  five-minute memory-only lifetime. Rebuild status from durable records for the
  authenticated source/owner or hub; other peers cannot read the brief. CLI and
  form display ProjectAgentSwitch outcome/action rather than internal phase.
  Tests exercise Unix-socket auth, duplicate ID after target replacement,
  unauthorized authored brief, status after restart and self acknowledgment.
- [ ] Run targeted commands as each owner lands:
  `go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'Test.*(SwitchRecovery|AgentSwitchView|RecoverAgentSwitch|SwitchAgent)' -count=1`;
  `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/couchmessage -run 'Test.*(Switch|Recovery|Remote|ResponseAfterWrite)' -count=1`;
  then the corresponding concurrency-sensitive selections with `-race`.
  Re-run exact authored-byte and fresh-no-orientation layout/wrapper acceptance.
- [ ] Register new source paths, update README/atlas and issue Log. Run the full
  machine-gated suite before M2 close. Review only through milestone-close, then
  proceed to M3's TL procedure and authorized live acceptance.

This delta uses existing storage, liveness and launch owners (ARCH-DRY), adds
pure projection with a thin orchestration shell (ARCH-PURE), and tests the whole
failure class rather than a single happy-path handoff (ARCH-PURPOSE/MOCK).
Restoring availability is an explicit operator action after restart, never
unbounded background retries. Durability is one record per slot with a named
supersession/deletion lifecycle (ARCH-FUNERAL); immutable accepted configuration,
exact identities and closed decoding enforce trust (ARCH-SECURE/ORDER).

### 2026-10-12 22:12 PDT — Switch only; coordinates are discoverable, never delivered

**Authoritative replacement for all earlier M2/M3 design and tasks.** M1 remains
closed at 37707f73b. The earlier text is historical provenance, not executable
scope. TL receipt d24edf27-e5db-4006-ad0b-8bd7672dbca9 relays the operator:

> Record the last conversation's coordinates (where to find each piece of data);
> the new conversation decides what to do with them. Decouple compaction from
> the agent switch: a switch may not need any context.

**Goal:** Switch agents without delivering anything, preserve the exact previous
conversation's coordinates, and leave failures with one safe recovery action.
**Architecture:** Existing authenticated operation admission and park/start own
the switch. A small durable transaction preserves source evidence and launch
witnesses; the existing thread gate serializes effects. One pure projection
drives receipts and menu actions. No handoff coordinator or content lifecycle.
**Tech stack:** existing Go packages, Couch socket and portable thread store.

#### Revised contract and acceptance

`couch --switch-agent repo:N|self --agent NAME --confirm [--json]` switches only.
Remove switch-specific handoff-file/request-callback flags and wire fields.
No authored brief, fallback text, automatic summary, orientation request,
composer paste, draft seed or CLI prompt is created by a switch. Native fresh
registration/readiness remains required and independent of orientation.
Preserve M1's previous descriptor, exact native transcript and tty coordinates,
park identity/token, source identity, `--previous`, and full reads/retention.

Busy external targets, dirty worktrees and unknown source state refuse before
park. **Proposed self-call detail for TL confirmation:** acknowledge an
authenticated self request before effects, then wait only for that initiating
turn to settle under the existing bounded settle policy; recheck clean/exact
source immediately before park. Dirty self requests refuse too. This is the
minimum allowance for an agent to call self while its tool turn is running;
there is no brief-writing turn or callback. If strict busy refusal includes
self's initiating turn, self will refuse there instead of switching.

Success is exact fresh target registration, not prompt delivery. Old exact
source proved live after failure -> **Old agent still running / Retry switch**.
No single survivor proved live -> **Switch needs recovery / Recover slot**
(resume the old thread). Recovery first reconciles exact source/target evidence:
reattach a surviving target or old source, otherwise resume the saved old native
conversation only after proving no competing process is running. Unknown
presence and recovery failure keep the action plus a diagnostic; neither
authorizes a duplicate launch. Restart shows the same action and notice without
automatically retrying park or start. No claim of uninterrupted availability.

Ship this sentence in both `couch --skill` and `couch --help`:

> After a switch or restart, `couch --peek self --previous` shows where the last
> conversation's transcript and terminal log are; read them only if your task
> needs them.

Ensure the default previous peek actually exposes those coordinates; if any
currently require `--transcripts`, reuse the existing descriptor formatter to
include them. Missing evidence remains explicitly unavailable, never current
session substitution. The hub/operator supplies any context request separately.

Out of scope: compaction/continuation/restart delivery, launch prompts, TL brief
authoring procedure, and changes to ops skills. Do not delete shared orientation
or continuation support used by other operations. The logged launch-prompt
replay risk remains a separate concern, not a new fix inside this plan.

#### Core concepts and integration owners

Statuses below are relative to the paused checkpoint, not a claim it works.

| Pure entity | Lives in | Status |
| --- | --- | --- |
| PreviousSession | `cmd/internal/threadrecord/previous_session.go`, `cmd/internal/couchcore/previous_session.go` | modified: retain M1; allow pre-park coordinates to survive interrupted switch |
| SwitchRecovery / SwitchSourceSnapshot / SwitchLaunchWitness | `cmd/internal/threadrecord/switch_recovery.go` | modified: source/profile/coordinates and attempt witnesses only; remove all content and delivery fields |
| AgentSwitch transition | `cmd/internal/couchcore/agent_switch_model.go` | modified: switch/recovery effects only; remove brief/callback/delivery events and deadlines |
| AgentSwitchView | `cmd/internal/couchcore/agent_switch_view.go` | modified: progress, old-live retry, needs-recovery, target-ready; remove copy-brief |
| SwitchContent | `cmd/internal/threadrecord/switch_recovery.go`, aliases in `cmd/internal/couchcore/agent_switch_model.go` | deleted |

| Integration | Lives in | Status | Reuses |
| --- | --- | --- | --- |
| Authenticated CLI admission and status | `cmd/internal/couchcmd/switch_agent_cli.go`, `agent_switch_service.go`, `agent_switch.go`; `cmd/internal/couchmessage/protocol.go`, `transport.go` | modified | broker identity, operation receipts and after-response-write activation |
| Source capture, park and launch | `cmd/internal/couchcore/switchagent.go`, `previous_session.go`, `starttransaction.go`, `threadstore.go` | modified | M1 evidence resolver, CAS, park/start events |
| Exact old-thread recovery | `cmd/internal/couchcore/switch_recovery.go`, `switch_native.go`, `resume.go`, `launch_existing.go` | modified | liveness, warm attach, native resolver and resume |
| Shared form, menu and restart projection | `cmd/internal/couchtty/console_switchagent.go`, `console_remote.go`, `menu_agent_switch.go`, `menu_reattach.go`; `cmd/internal/couchcore/actionableinventory.go`, `slot_operation.go`, `recover_action.go` | modified | existing operation dispatch and inventory |
| Prompt-free fresh readiness | `cmd/internal/launcher/args.go`, `createflow.go`, `launch_args_policy.go` | modified | retain FreshNonce independently of Orientation |
| Discoverability and peek formatting | `cmd/internal/couchcmd/skills/couch/SKILL.md`, `run.go`, `peek_tail.go` | modified | embedded skill/help and M1 descriptor response |

Keep one latest transaction per thread, one pending operation per canonical
slot. Save original profile/native ID independently of mutable latest binding.
Replace only after resolution by the next accepted switch; deletion follows the
thread. M1 leases remain the retention authority for archives (ARCH-FUNERAL).
Persist evidence before acknowledgement and before destructive effects; record
target/restoration witnesses atomically with existing start events. Durable
records are evidence, never liveness authority (ARCH-ORDER/SECURE).

Use a tagged transaction phase: admitted, switching, recovering, resolved.
Resolved outcome and current liveness drive the view. An effect completion
must match request ID and attempt/source identity. Older/malformed WIP records
cannot cause effects; reject unsupported shapes visibly with retained evidence.
No migration may replay a brief. Existing records without a switch remain valid.

| State/event | Next state and permitted effect |
| --- | --- |
| none + valid admission | admitted; persist immutable request/source snapshot before acknowledgement |
| admitted + response written and settled exact clean source | switching; persist coordinates and park intent, then park |
| admitted + busy/dirty/changed source, cancellation or failed response write | resolved/refused; no park |
| switching + verified park | switching; finalize archive coordinates and record target claim before launch |
| switching + exact target registered | resolved/target; no delivery |
| switching + park/start error or uncertain completion | resolved failure; inspect evidence and project retry/recovery |
| restart with unresolved transaction | no automatic effects; project current evidence and recovery action |
| failure + explicit recover | recovering; select exact survivor or prove absence before old-thread resume |
| recovering + success/error | resolved survivor/failure; repeated recover reuses survivor or retains action |
| any + stale completion/duplicate request | ignore stale completion; identical request observes same operation, conflicting request refuses |

#### Remaining implementation — one M2 boundary, no M3

M2 now includes switch mechanics, recovery, discoverability and acceptance.
Delete the old M3 work commitment; no separate TL authoring procedure is needed.
Do not start implementation until TL reviews this plan and revised change-code
passes. Re-estimate only after that gate; 8.640h remains historical provenance.

- [ ] **Trim paused WIP.** Compare against 37707f73b. Remove brief-file reader,
  callback admission/correlation, content selection, brief/overall handoff timers,
  delivery observation, copy actions and switch orientation hooks in the files
  above and their tests. Keep source authentication, immutable accepted argv,
  response-write barrier and FreshNonce. Keep recovery drafts only as candidates
  for the following tests: they are currently disconnected, not verified reuse.
  Commit the coherent removal, preserving M1 and unrelated consumers.
- [ ] **Test and implement durable source capture.** Extend colocated previous
  session/store and switch-recovery tests with failure before/after park. Capture
  exact native and live tty paths before ParkExpected in the durable transaction;
  preserve the park token when park admission allocates it, before source stop.
  After verified park, finalize the M1 archive descriptor using that same source.
  Before finalization, previous discovery uses the pending source coordinates
  and marks archive fields unavailable; it must not expose an older source as
  this switch's predecessor. A failed write authorizes no destructive effect.
  Run tests red, wire CAS/retention-safe updates, rerun green, commit.
- [ ] **Test and implement switch-only admission/transaction.** Colocated CLI,
  service, protocol, model and core tests cover exact/self address, --confirm,
  unsupported flags, busy/dirty refusal, immutable parameters, concurrent
  requests, source replacement, failed ack, self settle, and no orientation/body
  in launch or queue. Wire the reduced transition function and shared form.
  Atomically retain target/restoration witnesses; no IO under receipt mutex.
  Confirm no callback/delivery dependency remains, rerun targeted tests, commit.
- [ ] **Test and complete recovery/restart.** Use the existing stateful process,
  session/runner fakes and portable thread store. Interrupt before/after park,
  claim, helper start and registration; reopen that store. Assert actual survivor
  identities, action/notice, exact old native ID and launch count. Cover lost
  acknowledgements, unknown liveness, same-agent switch, absent transcript,
  repeated recovery, recovery failure, and interruption during recovery. Wire
  RecoverAgentSwitch before ordinary auto-recovery; test real menu dispatch and
  authenticated receipt reconstruction. Run targeted race tests, commit.
- [ ] **Discoverability and verification.** Update skill/help, README and atlas
  in the same change; register any changed source inventory paths. Test default
  previous-peek coordinates and missing-data behavior. Keep M1 full-read tests.
  Through the existing fake launcher verify all three agent names launch with
  no orientation payload or generated prompt; retain wrapper no-orientation
  readiness acceptance. TL schedules live Claude/Codex/Grok switch checks when
  safe: new conversation starts without delivered text, previous peek locates
  the outgoing evidence, induced failure exposes the selected action.
- [ ] **Close M2 and issue.** Run targeted selections then background `make test`;
  log evidence and update atlas. `sdlc milestone-close --issue 517 --milestone M2`
  owns boundary review. Resolve findings, then close/publish through SDLC after
  required live acceptance. No additional ad-hoc review pass at that boundary.

Targeted commands (expected PASS after each relevant red/green task):

```sh
go test ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'Test.*(Previous|Switch|Recover)' -count=1
go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/couchtty -run 'Test.*(Switch|Previous|ResponseAfterWrite)' -count=1
go test ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'Test.*(FreshNonce|Orientation|LaunchEnv)' -count=1
go test -race ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(Switch|Recover)' -count=1
```

ARCH-DRY/PURE: reuse source/launch/recovery owners and one pure projection.
ARCH-PURPOSE: context is discovered on demand, never delivered by switching.
ARCH-MOCK: same stateful runner/process/store seams exercise interrupted effects.
ARCH-CONSTRAINTS: heavy user-triggered operation, no new global admission cap,
one pending/latest per slot; reuse existing settle/launch bounds and propagate
uncertain outcomes rather than inventing absence. No transcript parsing/replay
or network/model calls enter the supervisor switch path. ARCH-SECURE/ORDER and
ARCH-FUNERAL constraints are specified with the transaction and retention above.

### 2026-10-12 22:17 PDT — Lean TL-coordinated switch; replaces the entire 22:12 plan

**Current plan for approval. Do not execute earlier M2/M3 tasks.** Verified TL
receipts e448b65d-8bba-43fb-86e6-415a7ad41b39 and
8d07c41c-e35f-49f3-bbf3-8bdb4b4d460c supersede the preceding revision:

> A lean switch-agent API just quits the current agent and starts a new one
> with a different coding agent, using the known-good startup config for that
> agent in that repo. Changing the default startup parameters is reserved to
> the human.

**Goal:** Let the TL switch a clean, idle slot using settings the operator has
already established. **Architecture:** a thin authenticated adapter over the
existing switch form's prepare/park/start machinery, with server-resolved argv
and existing park/resume failure handling. **Tech stack:** existing Go packages.

#### Contract

`couch --switch-agent repo:N --agent NAME --confirm [--json]`. No `self` alias;
also refuse an exact address resolving to the caller's own slot. Remote API is
for the authenticated hub/TL; the operator retains the local form. Busy, dirty,
unknown or changed source refuses, with no waiting for a turn to settle.
Resolve the target profile before quitting. A different agent is required.

Use the same lookup as the form: the target agent's successful path preference
(`PathLaunchPreference.ArgvByAgent[NAME]`), then its existing repository-scoped
agent default. `repoLaunchDefault` already resolves the slot's primary root.
Keep this precedence; never borrow another agent's argv or another repo's
settings. Distinguish a present empty argv (valid known configuration) from
missing configuration. Missing/invalid config refuses before park:
**"No known-good startup configuration for NAME in this repository; ask the
operator to set it in the switch form first."** The resolver's current silent
empty-argv fallback is not allowed on this API path.

Agent requests carry no argv, default-write option, handoff body or callback.
The server resolves/copies parameters and fingerprints the accepted source and
configuration; revalidate immediately before effects. Config changes while queued
refuse rather than launching different parameters. The operator's form remains
the only switch surface that edits parameters. Successful switching may retain
the existing successful-launch bookkeeping but cannot supply new defaults or
change any stored per-agent argv. Fresh-argument validation remains in force;
an invalid remembered resume configuration directs the operator to the form.

The TL captures what it needs with existing `couch --peek ... --transcripts`
before switching. Switch creates no coordinate record, brief, prompt, discovery
line, previous-session lookup, handoff lifecycle or new durable recovery record.
Remove all switch-generated orientation from the operator form too. Keep fresh
registration via FreshNonce without orientation. Compaction, continuation and
launch prompts stay out of scope. Shared delivery used elsewhere remains intact.

#### Minimal failures and existing state

Use existing `SwitchAgentResult`, park transaction, start claim and resume path:

| Observed result | Visible next step |
| --- | --- |
| Refusal or quit failure; exact old agent still running | Retry switch after the stated blocker is cleared |
| Exact new agent registered | Switched to NAME |
| Old agent parked; target conclusively failed to start | Resume old thread through existing resume action |
| Partial/unknown quit or startup | Show the existing lifecycle diagnostic; resolve uncertainty before retry/resume, never launch a duplicate |

Do not auto-resume. Retain the existing source profile/binding and verified park
until target registration commits the switch; an attempted target must not
become the old thread's resume authority on failure. Use existing cleanup and
resume guards, not a new switch recovery schema or coordinator. Interrupted
operations use ordinary start/park reconciliation after restart. Tests below
must establish the old-thread resume guarantee on definitive startup failure;
if the current lifecycle cannot express it, stop and report the specific gap
instead of rebuilding the superseded recovery subsystem.

#### Core concepts and integration owners

| Pure entity | Lives in | Status relative to checkpoint |
| --- | --- | --- |
| LaunchProfileResolution | `cmd/internal/couchcore/launchprofile.go` | modified: explicit configured/missing result for the agent API, retaining form behavior |
| SwitchAgentRequest / SwitchAgentResult | `cmd/internal/couchcore/switchagent.go` | modified: no content; server-selected config and minimal outcomes |
| SwitchContent, AgentSwitch handoff model, SwitchRecovery and custom view | `cmd/internal/couchcore/agent_switch_model.go`, `agent_switch_view.go`, `cmd/internal/threadrecord/switch_recovery.go` | deleted: no replacement subsystem |

| Integration | Lives in | Status | Reuses |
| --- | --- | --- | --- |
| CLI and authenticated remote switch | `cmd/internal/couchcmd/switch_agent_cli.go`, `agent_switch_service.go`, `slot_operations.go`; `cmd/internal/couchmessage/protocol.go` | modified | existing slot operation admission, receipt/status and console queue |
| Config lookup and guarded park/start | `cmd/internal/couchcore/switchagent.go`, `launchprofile.go`, `threadstore_location.go` | modified | form lookup, thread gate, exact-source guards, CAS and existing lifecycle |
| Human form and result notice | `cmd/internal/couchtty/console_remote.go`, `console_switchagent.go`, `menu_actions.go` | modified | existing form parameter editing, switch operation and resume action |
| Fresh registration | `cmd/internal/couchcore/launch_existing.go`; `cmd/internal/launcher/args.go`, `createflow.go`, `launch_args_policy.go` | modified | retain FreshNonce without orientation |

#### Remaining tasks — one revised M2, no M3

- [ ] **Apply the chosen M1 disposition after TL decision.** The cost comparison
  is in the issue's newest Revisions entry. Do not remove reviewed M1 functionality
  unilaterally. Recommended option is full reads only; previous persistence and
  `--previous` then leave together, including retention hooks and schema fields.
- [ ] **Remove superseded M2 WIP.** Compare to 37707f73b and keep only remote
  command parsing/authentication, existing queue/receipt integration, immutable
  config/source guards and FreshNonce. Remove custom coordinator, callbacks,
  all content/delivery timers and fields, custom recovery schema/helpers/menu,
  and saved-native recovery plumbing added only for that subsystem. Restore
  unrelated orientation/continuation consumers. Commit the coherent trim.
- [ ] **Known configuration, tests first.** Extend `launchprofile_test.go` and
  `switchagent_test.go`: configured-empty versus missing, per-agent/repo isolation,
  form precedence, malformed/default mismatch, remembered invalid resume args,
  config change before park and failed launch not overwriting defaults. Run red,
  add the narrow server-only selection policy, rerun green, commit.
- [ ] **Lean command, tests first.** Update colocated switch CLI/service/protocol
  tests: exact slot only; own-slot/foreign caller rejection; --confirm required;
  reject self, argv/body/handoff/callback fields; busy/dirty/unknown refusal before
  effects. Bind queue execution to exact source and accepted config. Use existing
  operation queue/status, no switch-specific wait loop. Test refusal/replacement
  while queued and duplicate admission; implement, rerun green, commit.
- [ ] **Lifecycle and form, tests first.** In `switchagent_test.go`, existing
  start/park integration tests and `console_switchagent_test.go`, use stateful
  process/session/runner fakes plus portable thread store. Test clean quit then
  target registration, quit failure with old live, spawn/registration failure,
  old-thread resume, target survives lost acknowledgement, interrupted start and
  unknown liveness. Assert surviving identities and launch counts. Form and API
  produce no orientation/draft/prompt; only form can edit argv. Implement minimal
  wiring and result notices, run targeted race checks, commit.
- [ ] **Docs and acceptance.** Update CLI help, README and atlas for the lean
  verb and missing-config instruction; no previous-discovery sentence or TL
  authoring procedure. Adjust source inventory. Keep whichever M1 tests the TL
  selects. TL schedules live switching among Claude/Codex/Grok with existing
  configured settings, including a definitive failed start followed by resume.
- [ ] Run the full machine-gated `make test` in background. Record evidence,
  close M2 through `sdlc milestone-close --issue 517 --milestone M2`, resolve its
  review, then close/publish through SDLC after acceptance. No extra ad-hoc review.

Targeted red/green checks:

```sh
go test ./cmd/internal/couchcore -run 'Test.*(LaunchProfile|SwitchAgent|Switch|StartTransaction)' -count=1
go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/couchtty -run 'Test.*(Switch|ResponseAfterWrite)' -count=1
go test ./cmd/internal/launcher -run 'Test.*FreshNonce' -count=1
go test -race ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*Switch' -count=1
```

ARCH-DRY/PURE: existing resolver/lifecycle and result, no new persistent switch
noun. ARCH-PURPOSE: TL owns context; API only switches. ARCH-SECURE: server picks
argv, authenticates hub and refuses caller-self; human edits config. ARCH-ORDER:
serialize existing quit/start events; registration is commit, uncertainty is not
absence. ARCH-MOCK: lifecycle fakes model process survival and interrupted IO.
ARCH-CONSTRAINTS: heavy explicit action, existing operation bounds, no new timers
or global caps. ARCH-FUNERAL: existing receipt expiry/park/start cleanup owns
residue; no additional switch records. The previous 8.640h forecast is obsolete;
re-estimate after revised change-code review. Await TL OK before that gate/build.

#### 2026-10-12 22:19 PDT — M1 disposition selected under the operator's delegation

Receipt c8f62ecf-c917-4bc7-ab22-c198f01e35ba authorizes choosing the smaller way
to retain addressable full tty reads. **Choose the existing `--previous` reader,
reduced to tty evidence only.** This is the narrow exception to the preceding
"no coordinate record / previous-session lookup" statement; it does not revive
the discarded switch/handoff/recovery design.

Reason: current raw/events names are slot-and-agent aliases (`artifactpath`
`ScrollbackArtifacts`); parked archives have exact tokens. Printing the live
alias before switch cannot promise the same evidence on a later launch. The
current `--previous` reader already selects the exact park token, leases it and
uses the bounded full replay. Keeping that small pointer avoids inventing a new
stable pre-switch ID, archive mapping or copy/retention lifecycle.

Keep `couch --peek repo:N --all` and `couch --peek repo:N --previous --all`,
including JSON raw/events paths and text rendering. Full reads remain a complete
retained terminal projection or explicit error, not a silently truncated tail.
Keep one previous tty reference: source park identity, capture token/agent/events
presence and capture time, using existing capture validation/retention. Persist
it from the verified park before target launch; replace on the next distinct
source park, keep through retries, remove with the thread. No pre-park native
lookup or source snapshot is added. When unavailable, report unavailable.

Remove previous native session, transcript paths, session binding and accumulated
context diagnostics from new descriptor writes and from switch context capture.
Current `--transcripts` stays unchanged for the TL to capture before switching;
`--previous --transcripts` refuses with guidance to capture native coordinates
before the switch. Previous tty source identity is only for choosing the right
recording, never recovery or launch authority. Document the tty-only command in
ordinary peek help/README; do not add the discarded agent-orientation sentence
to the skill or help.

**Owner/task adjustment:** retain and simplify `PreviousSession` in
`cmd/internal/threadrecord/previous_session.go`, `record.go`,
`cmd/internal/couchcore/previous_session.go`, `thread.go`, `peek_selection.go`,
`peek.go`; retain the M1 bounded reader and CLI wiring. Keep the legacy JSON
decode fields only as compatibility input if persisted WIP records require it;
validate then discard their unused native-context values on subsequent writes.
Tests must read old/full and new/tty-only records without accepting malformed
identity/capture data. No new persistent artifact family.

Before trimming, write targeted tests: current and previous `--all`, matching
park source on retries and successive switches, missing capture, rejection of
previous-native lookup, transcript-context absence from new writes, compatibility,
lease retention during replay and limit/loss errors. Then reduce the descriptor,
remove its native resolver dependency and update docs. Add these packages to
verification: `go test ./cmd/internal/scrollbackcmd ./cmd/internal/threadrecord
./cmd/internal/couchcore ./cmd/internal/couchcmd -run
'Test.*(Previous|Peek|Bounded|Capture)' -count=1` (one shell line).

M1's SHIP verdict records the old scope; the reduced surface must be verified and
reviewed with the final M2 diff. Build remains paused pending TL OK on this plan.
