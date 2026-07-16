---
id: 000115
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours:
started: 2026-07-16T12:17:57-07:00
---

# Switch the agent driving existing work

## Problem

When an agent provider is degraded, the user cannot smoothly move live work to
another coding agent. Pair treats the live agent session as if it were the work:
its normal picker hides attached sessions, choosing a different agent tends to
allocate a sibling tag, and conversational state remains trapped in the source
agent's transcript. A sibling `*-resurrect` tag would fragment one body of work
across identities and would require copying state that already belongs to the
original tag.

The tag should identify the work; Claude, Codex, or another agent should be an
exclusive, replaceable driver. Switching drivers must retain the draft pane,
sent-prompt history, future queue, native per-agent conversations, and the
human-meaningful context distilled through a continuation.

## Spec

### Work identity and exclusivity

- A Pair tag is the durable identity of one body of work. The selected agent is
  the tag's current driver.
- At most one live Pair session may drive a tag. Pair must never launch two
  sessions that can concurrently mutate the same tag-scoped draft, log, or
  queue.
- An agent handoff keeps the same bare tag and the same canonical repo-scoped
  public session identity. It does not allocate a `*-resurrect` sibling.
- Same-agent selection keeps the existing attach/resume behavior. Only a
  different requested agent enters the handoff flow.

### Picker and confirmation

- `pair <agent>` remains the normal entry point. Its picker uses the existing
  current-repository scope and history-window cutoff.
- When an agent is explicit, the picker also includes attached sessions. Rows
  identify the work tag, current/last driver, and attached, detached, exited, or
  recent-inactive state. Expired historical tags remain excluded.
- Selecting work already driven by the requested agent attaches or resumes it
  without a handoff prompt.
- Selecting work driven by another agent presents an exclusive-switch
  confirmation that names the tag, source agent, and target agent and explains
  that Pair will preserve tag state, park the source transcript, close the
  source session, and start the target agent. Declining or dismissing aborts
  without mutation.
- The active driver is authoritative when a live session exists; otherwise the
  most recent valid tag/session ledger entry supplies the last driver. If Pair
  cannot identify a source driver or transcript, it must say what state is
  unavailable before asking whether to proceed with the remaining tag state.

### State ownership

Tag-scoped state is reused in place rather than copied:

- `draft-<tag>.md` — the active `*` draft;
- `log-<tag>.md` — sent-prompt history; and
- `queue-<tag>/` — the future `+N` prompts.

Agent-scoped state remains separate under `(tag, agent)`:

- the native conversation/session configuration; and
- raw, event, rendered, and parked scrollback artifacts.

This lets each agent retain its own native conversation while every driver sees
the same work-level input state. A continuation transfers the source driver's
human-meaningful context; it does not replace or duplicate the tag-scoped files.

### Repository-scoped agent defaults

- Pair stores the last explicitly supplied launch arguments per `(repository,
  agent)`, separate from tag-specific native-session configuration.
- Bare `pair claude`, `pair codex`, or another `pair <agent>` uses that agent's
  repository-scoped defaults when creating a fresh native conversation.
- Arguments supplied after `--` replace that repository's defaults for the
  selected agent after a successful launch. An unsuccessful or cancelled
  launch must not change the saved defaults.
- Launch argument precedence is:
  1. explicit arguments after `--`;
  2. valid saved arguments for the target `(tag, agent)` conversation;
  3. repository-scoped defaults for the target agent; then
  4. no additional arguments.
- When a valid target `(tag, agent)` conversation exists, Pair resumes it. The
  chosen arguments are composed with that agent's canonical resume invocation;
  repeated handoffs must not accumulate duplicate resume tokens.
- Returning to a previous driver therefore resumes that agent's prior native
  conversation and uses the new continuation to catch it up on work performed
  by the intervening driver.

### Exclusive handoff transaction

Pair performs a handoff in these phases:

1. Resolve the source tag/agent/session and the target agent's launch arguments
   without mutation.
2. Render the source agent's current scrollback to a stable, immutable parked
   transcript suitable for continuation dead-agent mode. The parked artifact
   records the source tag and agent.
3. Compute and validate the complete draft/queue transition and target launch,
   including session-name acceptance, before stopping the source.
4. If `draft-<tag>.md` is non-empty, reserve it as the new `+1` and shift the
   existing future queue back without changing item order. Prepare a generated
   handoff instruction as the new `*` draft.
5. Stop and remove the source Pair session, preserving tag-scoped state and the
   parked transcript. No target session may start before exclusivity is
   established.
6. Atomically install the planned draft/queue state and launch the target agent
   under the same tag and public session identity.

The generated `*` instruction identifies the source tag, source agent, and
parked transcript path. It tells the target agent to follow the continuation
datatype's dead-agent procedure, create a continuation of that source session,
and then continue the work in the current session. The instruction must be
actionable without relying on the target session's `PAIR_TAG`, which names the
shared work rather than the transcript's source agent.

### Failure and recovery

- Any failure before the source session is stopped leaves that session and all
  tag-scoped files unchanged.
- Draft/queue replacement is an atomic plan with stable filename keys; an
  intervening queue index must never identify an item across a mutation.
- If a failure occurs after the source stops, Pair restores the original
  draft/queue snapshot and attempts to relaunch the source driver from its saved
  `(tag, agent)` configuration. It reports both the target-launch failure and
  whether source recovery succeeded.
- A failed handoff must not fall back to a sibling tag, lose a queued prompt,
  overwrite the source transcript, or claim success with neither driver live.
- If the source transcript cannot be rendered, Pair must not stop a live source
  session unless the user explicitly accepts a tag-state-only handoff.

### Architecture

- Reuse the existing repo-scoped session snapshot, history cutoff, fzf runtime,
  session-name index, continuation renderer/writer contracts, agent argument
  composition, and queue primitives (ARCH-DRY).
- Keep picker-row construction, driver classification, argument precedence,
  queue-transition planning, and handoff/recovery planning as deterministic
  functions. Zellij, filesystem, rendering, prompts, and process launch remain
  in the runtime shell (ARCH-PURE).
- The delivered behavior is an actual same-work agent switch, including
  attached-source recovery and all tag-scoped input state, rather than only the
  easier sibling-session launch (ARCH-PURPOSE).

## Done when

- `pair <agent>` lists attached work within the normal repo/history scope and
  distinguishes same-agent attach from different-agent handoff.
- A confirmed different-agent selection parks the source transcript, enforces
  one live driver, and launches the target under the same tag/session identity.
- The active draft is moved to the front of the future queue, the prior future
  queue retains order, history is unchanged, and the generated handoff prompt
  becomes `*` without data loss.
- A returning target agent resumes its prior valid native conversation and is
  instructed to ingest a continuation from the intervening source agent.
- Repository-scoped per-agent defaults are reused on bare launches and replaced
  only by successful explicit `--` launches.
- Pre-stop failures are non-mutating; post-stop failures restore input state and
  attempt to recover the source driver with a clear outcome.
- Unit tests cover pure selection, filtering, allocation, argument precedence,
  queue ordering, and recovery plans; a process-level fake covers a complete
  live Claude-to-Codex handoff.
- Atlas documentation describes tag-as-work identity, exclusive agent drivers,
  handoff state ownership, and repository-scoped agent defaults.

## Plan

- [ ] Write the durable implementation plan after the approved spec passes review.

## Log

### 2026-07-16

Claimed before design and entered `sdlc start-plan`. The initial
`pair <agent> resurrect` idea was replaced during brainstorming with an
exclusive agent handoff on the same work tag. The design reuses tag-scoped
draft/history/queue state, parks agent-scoped scrollback for continuation
dead-agent mode, resumes returning agents' native conversations, and adds
repository-scoped per-agent launch defaults. ARCH-DRY, ARCH-PURE, and
ARCH-PURPOSE shaped the boundaries above.
