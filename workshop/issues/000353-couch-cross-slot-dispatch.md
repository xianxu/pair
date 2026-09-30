---
id: 000353
status: working
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-30
estimate_hours:
card_mirror: '8aad5da495291a2921bcf06e9c1c3a21cf40f3ae' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-30T13:47:42-07:00
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

- [ ] Approve the first-iteration contract and implementation plan.
- [ ] Implement and test bounded actor admission and request/reply rules.
- [ ] Integrate supervisor transport and exact live wrapper registration.
- [ ] Qualify safe Claude/Codex delivery through Pair's input owner.
- [ ] Ship CLI, skill, operator notices, and end-to-end acceptance evidence.

Durable plan: [implementation](../plans/000353-couch-cross-slot-dispatch-plan.md).

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

## Revisions

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
