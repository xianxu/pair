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

`scrollbackcmd.RenderLines(..., 0)` already returns uncapped plain text. It reads
the raw file and builds emulator history in memory, and keeps one drain goroutine
until process exit. Reuse it only in a short-lived CLI, one full recording per
invocation. No new transcript parser or summary generator.

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
  100,000 rendered rows and at most 1,000 columns/rows per resize. Check file
  size/dimensions before replay and detect row overflow with a capped history
  plus one; refuse visibly rather than return a truncated result as complete.
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
