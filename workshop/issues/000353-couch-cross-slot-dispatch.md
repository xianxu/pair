---
id: 000353
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-30
estimate_hours: 8.19
card_mirror: '6b0fa566496540576a7fe48a4819a7486ef8262e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T13:47:42-07:00
flow: {kind: full, provenance: operator}
---

# Live cross-slot dispatch between couch slots

## Problem

Using several Couch slots concurrently still costs operator attention at the
start: the operator must go to each slot and kick work off. An agent in one
slot (say ariadne:1) that has discussed follow-up work with the operator
cannot hand it to an idle live slot (say pair:2) so that work is already
under way when the operator arrives. Slots are bounded, inspectable
concurrency (unlike subagents, the operator has console access), but only if
dispatch is cheap.

## Spec

- **Live, not queued on disk.** Dispatch targets a live, idle Couch slot like
  asking an online coworker. No free slot is the status quo, not an error: the
  issue exists and simply isn't picked up yet. No disk queue (that's Argos,
  ariadne#229).
- **Durable subject, ephemeral transport.** A dispatch names an issue, not a
  free-form brief. The sender files it in the target repo (the sender has the
  operator's context): `sdlc issue new` in a free slot of that repo, then
  `sdlc issue move-detail` so it is claimable. Losing the Couch message loses
  nothing durable.
- **Only concurrent work.** Dispatch only work not blocked by the sender's
  current task (ariadne#272: never start on an unlanded base).
- **Couch routes, pair sequences.** Couch holds a per-slot in-memory queue of
  machine messages; pair interleaves them with human input and injects only
  at a safe point. Pair marks injected lines (e.g. `[couch:ariadne:1] …`);
  the marker's authority comes from pair applying it, never from text an
  agent reads, so a typed or tool-output line starting `[couch:` is not a
  peer message. The skill teaches the format and that a peer message is a
  request, not operator authority.
- **Safe insertion point is the hard part.** Pair's return/alt-return work
  already detects some agent-pane states (e.g. a selection menu accepting only
  certain input), but it cannot see everything: an image pasted from the
  draft leaves an `[Image 1]` placeholder in the agent's input box until sent,
  and injecting then would intermingle the messages. Candidate rule: inject
  only when the agent's input box is visible and empty; fall back to a
  recent-user-activity heuristic where that can't be detected.
- **Discovery:** an environment marker plus the Couch skill tell agents they
  run inside Couch and how to address `repo:N` or `repo` (any free slot).
- Provenance UI is secondary; the operator can ask the slot's agent.

### First iteration — proposed 2026-09-30

This proposal refines the original bullets above. Work and acceptance evidence
stay in Ariadne/repository artifacts. Couch messages are transient delivery
attempts, never another task tracker. No automatic provisioning, replay after
restart, cross-instance routing, broadcast, or worksheet scheduler in this issue.

**Public interface.** `couch --actors` lists live, message-capable slots in this
Couch namespace, including each slot's availability and pending message.
`couch --available on|off` declares the calling slot's willingness to take new
work. New/reconnected actors default off. `couch --send-to pair:1 --message TEXT`
addresses an exact live slot; `--send-to pair` atomically selects and reserves
one available live slot in that family. Explicit addressing can send a
clarification to an occupied slot. Work instructions name a durable issue or
other repository artifact; free-form coordination text is allowed. Couch does
not parse issue contents or run SDLC verbs. The skill owns that convention.

`couch --reply-to ID --message TEXT` permits one terminal reply to a delivered
request, addressed back to its original live sender. Replies cannot themselves
be replied to. `couch --message-status ID` returns the canonical envelope and
current outcome to its participants, and `couch --skill` prints the shipped
skill. These CLI shapes are tentative until plan approval.

**Availability and human grounding.** Availability is an explicit declaration,
not inferred from silence, Git cleanliness, or a completed model turn. A new
human submission or accepted ordinary peer request clears it. Receiving a
reply does not release unfinished work. The skill prohibits advertising
availability while implementation, required review, or human acceptance is
outstanding. After acceptance the agent can advertise again. Runtime input
readiness is a separate Pair observation, never evidence of completed work.

**Runtime.** One broker belongs to the live Couch supervisor lease. It maintains
one actor per live registered slot and one pending delivery per actor. Routing
uses the existing verified family/slot identities, including `:0`; ambiguous
family names refuse. Selection/reservation occurs atomically. The recipient's
actor owns delivery sequencing; Pair's existing single input writer performs
all writes. Status queries are answered by the runtime without an LLM turn.
Sender/recipient bindings include the exact conversation and wrapper launch
identity. A reused slot name cannot inherit a pending message or reply right.

**Admission.** Acceptance returns a message ID and resolved recipient immediately
after reserving the recipient's pending slot, not after model completion.
No free family member returns `not-dispatched` with an explanation and exit 0;
invalid destination, dead explicit recipient, unsupported receiver, queue full,
and malformed input return a typed refusal and nonzero exit. A normal accepted
message clears availability. Delivery status is queued, delivering, submitted,
expired, cancelled, or indeterminate; submitted means bytes submitted, not task
started. Query status instead of automatically retrying an uncertain send.

**Delivery.** Initial receivers are Claude Code and Codex, qualified against
captured fixtures and a live smoke; other agents remain usable normally but
refuse peer delivery. Require a recognized, empty coding composer, a completed
turn or positively qualified fresh-session ready state, no menu/permission
prompt, no image attachment/capture, and no operator input in flight. Unknown
state waits and expires after 30 seconds. There is no inactivity heuristic or
force-send fallback. Once paste begins, operator interference cancels automatic
submit and preserves the visible text; partial writes are indeterminate and
never automatically retried. The original deadline also applies after paste:
if the matching render does not arrive in time, cancel automatic submit,
preserve the visible text, report intervention needed, and ignore late render
events. The CLI/status tells the sender when intervention is needed. Pair must arbitrate paste, image admission, typing, and submit under
one input owner; an image remains pending after capture completes.

**Loop control.** The skill teaches one useful reply at most, no courtesy
acknowledgments, and no opening a fresh request to continue an exhausted reply.
The runtime additionally permits eight accepted inbound peer messages per slot
between genuine operator submissions. Requests and replies both consume this
allowance; peer input, orientation, idle transitions, and availability changes
never replenish it. Exhaustion refuses further messages visibly until an
operator submits input to that slot. This bounds fresh-ID and multi-agent cycles
within the existing live actor set without tracking a conversation graph.
A new supervisor/wrapper binding starts a new ephemeral allowance; agents must
not restart or manufacture operator input to evade the limit.

**Provenance and visibility.** Pair creates the sender/ID envelope from the
broker's record and displays a small delivery notice. The skill reads the
canonical envelope through `--message-status` and uses that returned content,
checking current recipient incarnation, sender, body and submitted outcome
before acting. A copied ID with altered text, unknown record or mismatch
confers no peer provenance; a Couch-looking prefix creates no broker record.
Native harness transcript roles cannot be changed through TTY paste: this is
verified peer provenance, not a new native system-message role. Messages never
confer operator approval. Existing Couch navigation remains the inspection UI;
`--actors`/receipts expose coordination state. No dashboard or TTY restyling.

**Bounds.** Start with one pending message per slot, an 8 KiB UTF-8 body limit,
128 registered actors, a 2-second CLI admission deadline, a 30-second delivery
deadline, and 256 retained receipts with a one-hour lifetime. Reject new
admissions at capacity; never evict a pending request or live reply right to make
room. Finished replies can release their parent receipt; expiry ends reply
rights. These are conservative initial engineering choices, not measurements.
Sockets and their ownership metadata are runtime handles, not durable messages.
Actor/supervisor shutdown cancels pending attempts and removes owned handles;
restart revalidates live wrappers and retains no old messages or availability.

### Approved spec amendments — 2026-09-30

The operator approved the spec with the following simplifications. These
amendments supersede conflicting original/proposed text above and the older
Done when / Plan wording below; the earlier text remains as design history.

- **One free-text send API.** Use `couch --send-to SLOT_OR_FAMILY --message TEXT`
  for instructions, updates, and responses alike. Remove `--reply-to`, parent
  message IDs, reply rights, and runtime enforcement of one terminal reply.
  Agents correlate conversations from sender identity and their own context.
  Couch supplies the sender identity; agents need not supply message IDs to
  converse. IDs remain delivery receipts/deduplication details, not conversation
  threading requirements. Discovery, status, skill, and provenance behavior
  otherwise remain as proposed.
- **Derived availability.** Remove `--available` and its declaration state.
  A family dispatch may select a live supported slot when its checkout is on
  its verified resting branch, there has been no operator input or agent TTY
  output for at least 30 seconds, and there is no pending message/reservation.
  Use operator input (including draft typing without submission) and agent TTY
  traffic, not screen-diff analysis. Couch's own notices/status updates and
  background status queries do not count as activity. Unknown branch/activity
  observations do not establish eligibility; registration starts observation
  rather than assuming the preceding interval was quiet.
- **Deliberate approximation.** Do not require separate proof that no model
  request or tool command is running. Current agents generally animate while
  working; the non-resting branch already excludes ongoing issue work, leaving
  mainly the initial pre-claim/branch-switch window. Accept a silent action as
  a limitation of this first iteration. TTY quiet selects a candidate; it does
  not replace safe agent-composer checks at delivery. An exact-slot send can
  still coordinate with an occupied slot.
- **Draft pane versus agent composer.** Text left in the separate draft pane
  neither makes the slot busy nor blocks delivery. Active typing resets the
  quiet interval; existing draft contents do not. Pair must still preserve human
  input and require a recognized empty, safe agent composer, without menus or
  pending images. Remove the requirement for separate positive execution-idle
  evidence; retain safe insertion, deadline, and post-paste ownership checks.
- **One pending delivery.** A second send to a slot with a queued/delivering
  message is refused immediately as recipient-busy; preserve the first message.
  The reservation lasts until submitted or another terminal delivery outcome.
  An empty mailbox alone does not establish availability. Family selection and
  reservation remain atomic.
- **Breaker and convention.** Every accepted inbound peer message, including a
  response sent with `--send-to`, consumes one of eight allowances per slot.
  Only a genuine operator submission resets the budget; typing, peer input,
  idle time, status queries, and duplicate admission of the same transport ID
  do not replenish it or spend another allowance. No separate reply budget.
  The skill teaches actionable instructions/useful updates, responses when
  needed, and no courtesy acknowledgment loops. It does not impose a one-reply
  protocol. Existing restart/reconnect rules remain unchanged.
- **Human grounding remains.** Required human acceptance is outstanding work;
  keep its issue branch until that work is accepted and returned to rest. Couch
  does not infer acceptance from a completed model turn or implement a second
  work ledger. Durable work and acceptance evidence remain in repository artifacts.
- **Receipt retention.** Remove reply-right retention and parent-receipt release
  rules. Keep bounded receipts for delivery status/provenance/deduplication and
  their existing lifetime; reject admissions at capacity without dropping a
  pending delivery.

Acceptance must cover resting versus issue branches; the 30-second boundary;
draft typing versus an unchanged draft; agent output versus Couch notices;
silent execution without a separate idle signal; exact-slot coordination;
immediate rejection of a second pending message; safe composer/image handling;
responses via ordinary sends; and an A/B/C loop exhausting the finite breaker.
No `--available` or `--reply-to` command is required. The spec is approved;
implementation has not started.

## Done when

- From one slot, an agent dispatches an issue to an idle slot (explicit
  `repo:N` or any free slot of `repo`); the target starts work on it.
- Injection never happens while the target's input box is hidden or
  non-empty; tests cover a menu state and a pending image placeholder.
- A line not injected by Pair never creates authenticated peer provenance;
  the skill verifies its receipt before acting as a peer request.
- No free slot leaves the filed issue unclaimed, with a clear message.

- Explicit and family sends work between two live qualified slots, including
  `:0`; simultaneous family requests cannot reserve the same slot.
- Awaiting human acceptance does not advertise availability. Status queries
  do not invoke an LLM. One reply and the eight-message breaker are enforced.
- Restart, replacement, timeout, queue full, and partial writes cause no silent
  retargeting, replay, message loss masquerading as success, or forced submit.
- The Couch skill is shipped and usable by agents in any repository through
  `couch --skill`; setup and an operator acceptance scenario are documented.

## Plan

- [x] Approve the first-iteration contract and implementation plan.
- [x] Implement and test bounded actor admission and free-text messaging (approved amendment).
- [x] Integrate supervisor transport and exact live wrapper registration.
- [ ] Qualify safe Claude/Codex delivery through Pair's input owner.
- [ ] Ship CLI, skill, operator notices, and end-to-end acceptance evidence.

Durable plan: [implementation](../plans/000353-couch-cross-slot-dispatch-plan.md).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration is flagged stale/provisional.*

Derived after plan-quality passed. In block order: issue/spec brainstorm;
pure actor model; bounded broker; Unix transport; composer/delivery state;
shared input arbitration; supervisor/wrapper integration; image admission Lua
seam; CLI; Couch skill; docs; one close review; Claude and Codex conformance.
Implementation uses upper table values scaled by 0.4 for coupled runtime/harness
work, familiarity 1.0 (existing Go/TTY stack). Implementation-primitive design
uses the thorough-spec 0.2 discount; issue/spec retains 1.5h because it owns the
brainstorm decisions. Transport uses Go net/context and existing strictjson:
base design 2h is halved for library availability then multiplied by 0.2.
New policy models use base 1.5h ×0.2; no library removes their product decisions.
TUI uses 2h ×0.2, refactor 1h ×0.2, integration 2h ×0.2, Lua 1h ×0.2,
CLI 0.3h ×0.2, skill 0.7h ×0.2, docs/review 0.2h ×0.2.
Each harness discovery uses 0.6h ×0.4. Design 3.78h ×1.15 = 4.347h;
implementation 3.84h; total 8.187h, rounded to 8.19h.
Includes verification and acceptance work, excluding idle operator wait.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.5 impl=0.12
item: greenfield-go-module design=0.3 impl=0.32
item: greenfield-go-module design=0.3 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: tui-screen design=0.4 impl=0.4
item: cross-cutting-refactor design=0.2 impl=0.2
item: api-integration design=0.4 impl=0.6
item: lua-neovim design=0.2 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: skill-or-dispatcher design=0.14 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 8.19
```

## Log

### 2026-09-29

Filed from ariadne#272's brainstorm. Companion to pair#352 (`couch --notify`).

### 2026-09-30 — operator design direction

Claimed and entered planning in pair:2. The repository is the durable source
of work, including future worksheets for multiple agents; Couch owns only
ephemeral scheduling and communication between live slots. Messages may address
an exact slot (`pair:1`) or a family (`pair`). The operator wants a Couch skill,
slot-owned delivery sequencing, immediate admission feedback, a bounded reply
convention, and clearly identified peer input. Human acceptance remains part of
completion: a quiet agent awaiting acceptance is not thereby free for more work.
Automated acceptance may replace selected human checks later without changing
this separation. Keep the first iteration small and revisable.

Research compared Gas Town's assignment/nudge/mail separation, Claude Code's
cross-session messaging, and orchestrator/worker systems. Relevant references:
[Gas Town messaging](https://github.com/gastownhall/gastown/blob/main/docs/design/mail-protocol.md),
[Gas Town input delivery](https://github.com/gastownhall/gastown/blob/main/internal/cmd/nudge.go),
[Claude cross-session messaging](https://code.claude.com/docs/en/cross-session-messaging),
and [AutoGen termination](https://microsoft.github.io/autogen/stable/user-guide/agentchat-user-guide/tutorial/termination.html).
These inform the design; they do not add a durable runtime queue or an
autonomous staffing/merge system to this issue.

The initial ticket read used `sdlc issue show`, which intentionally prints only
frontmatter and section headings. The full details were subsequently read from
the file; design work must use the file body.

### 2026-09-30 — document review complete

A fresh-context reviewer approved both the revised Spec and implementation plan
on the second pass. Issue schema validation and whitespace checks pass. No code
or runtime test has been run for this proposal. Implementation qualification must
cover short, multiline and collapsed-paste displays: an arbitrary paste-summary
marker is not proof of the payload. Preserve complete-write and uninterrupted
input-ownership evidence, or decline automatic submission. Awaiting operator
approval of the concrete first-iteration plan before the full change-code gate.

### 2026-09-30 — operator approved simplified spec

The operator accepted TTY traffic as the first-iteration activity approximation,
with resting-branch eligibility and one pending delivery per slot. Clarified that
an unchanged separate draft pane does not block assignment or safe delivery.
Removed explicit availability and reply-ID APIs: all agent conversation uses
free-text `--send-to`, with the eight-message inbound breaker and skill convention
preventing runaway exchanges. Appended controlling spec amendments and aligned
the durable plan's implementation/test deltas (ARCH-DRY, ARCH-PURPOSE). This is
spec approval and a documentation checkpoint, not implementation or acceptance
evidence.

### 2026-09-30 — implementation checkpoint

The change-code plan gate passed and implementation is underway. Added the
in-memory actor broker, bounded local protocol, live Console binding checks,
free-text CLI and skill, and single-writer Pair delivery integration. Tests
reproduced and now prevent insertion over unrendered human input, invisible
whitespace drafts, and the stale screen immediately after human submission.
The unchanged separate draft pane remains independent of composer ownership.

All four affected package suites passed; focused messaging race tests passed.
The additional submission/render-fence regressions pass in the wrapper suite.
Live receiver qualification is unfinished: isolated startup runs exposed auth
and composer-layout barriers. The supported-version map remains empty until
live evidence qualifies receivers. This is a work checkpoint, not a usable
release, acceptance, or completion claim.

### 2026-09-30 — Codex qualification and composed integration

Codex CLI 0.159.2 passed fresh-session startup, short and multiline paste,
word-wrapped paste, draft/menu preservation, and exactly one submission followed
by the native composer clearing. Its version is enabled. Captures and replay
regressions are under `cmd/internal/wrapcmd/testdata/peer/codex/0.159.2/`.
The broker → private Unix endpoint → wrapper → rendered envelope → single submit
→ canonical receipt test passes normally and under the race detector.

Focus reports must not acquire human draft ownership; exact, fragmented and
mixed-input regressions now pass. All four affected package suites and focused
messaging/orientation race tests pass; `make build` passed before the latest
receiver qualification changes. Claude live setup needed the native keychain's
USER identity and a settled temporary-workspace trust picker; receiver checks
are underway. Human Couch acceptance and close review remain outstanding.

## Revisions

### 2026-09-30 — implementation entry and plan gate

Operator instructed continuation after spec approval. The first change-code
plan review requested admission-observation freshness and function-level test
strategies (PQ-1/PQ-2); the plan now names input/output producers, a bounded
conditional reservation, and independent sequence/grammar oracles. PQ-3's minor
conformance-cadence finding is also addressed. No implementation gate pass or
runtime behavior is claimed yet.

### 2026-09-30 — first-iteration proposal

The operator expanded issue-only dispatch to live peer communication while
keeping all durable work in repository artifacts. Proposed deltas: permit
coordination text with artifact references for work; explicitly advertise
availability; one pending delivery and one reply; a finite per-slot breaker;
Claude/Codex receiver qualification first. Remove the unsafe inactivity fallback.
Clarify the provenance criterion as broker verification because terminal paste
cannot create a native harness message role. Original Spec bullets are retained
above as history; the dated first-iteration proposal is the contract to review.

### 2026-09-30 — review clarifications

Fresh-context document review required using the canonical receipt content, not
merely checking a copied ID, and enforcing the delivery deadline after paste as
well as before it. Both are added to the proposal and deterministic test plan.
Ordinary reconnect of the same wrapper within one supervisor retains its spent
allowance; only genuinely new incarnation/supervisor state starts fresh.

### 2026-09-30 — approved simplification

Reason: operator feedback favors a small free-text API and observed TTY activity
over agent-maintained availability or reply bookkeeping. The approved amendments
in Spec replace those earlier proposals and their conflicting acceptance/plan
requirements. Historical proposal and review entries remain intact. Plan updated
with corresponding model, CLI, lifecycle, and regression-test changes.
