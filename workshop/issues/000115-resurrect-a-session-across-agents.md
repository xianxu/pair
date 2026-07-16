---
id: 000115
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-07-16
estimate_hours: 16.70
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
- When an agent is explicitly named, the picker also includes attached
  sessions. A bare `pair` retains the ordinary picker behavior; parsing must
  preserve whether the default agent was implicit. Rows identify the work tag,
  current/last driver, and attached, detached, exited, or recent-inactive state.
  Expired historical tags remain excluded.
- Selecting work already driven by the requested agent attaches or resumes it
  without a handoff prompt.
- Selecting work driven by another agent presents an exclusive-switch
  confirmation that names the tag, source agent, and target agent and explains
  that Pair will preserve tag state, park the source transcript, close the
  source session, and start the target agent. Declining or dismissing aborts
  without mutation.
- A single live session with mutually consistent session-index and agent-file
  evidence is authoritative; otherwise the most recent valid tag/session
  ledger entry supplies the last driver. Conflicting live evidence or multiple
  live sessions for one tag is a corrupt exclusivity state: Pair refuses the
  handoff, lists every conflicting session/evidence source, and asks the user to
  resolve it. It must never guess which transcript has provenance.
- If Pair cannot identify a historical source driver or transcript, it says
  what state is unavailable before asking whether to proceed with the remaining
  tag state.

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
  selected agent once that launch reaches the readiness point defined below.
  The parser records separator presence independently from argument count, so
  an explicit empty `--` deliberately clears the defaults after readiness. An
  unsuccessful or cancelled launch must not change them.
- Launch argument precedence is:
  1. explicit arguments after `--`;
  2. valid saved arguments for the target `(tag, agent)` conversation;
  3. repository-scoped defaults for the target agent; then
  4. no additional arguments.
- When a valid target `(tag, agent)` conversation exists, Pair resumes it. The
  chosen arguments are composed with that agent's canonical resume invocation;
  repeated handoffs must not accumulate duplicate resume tokens.
- A tag/agent config is structurally usable only when it parses, its embedded
  agent matches the requested agent, and its argument vector is valid. Malformed
  or agent-mismatched config is ignored with a warning. A non-empty session ID
  is resumable only when the requested agent recognizes it and its native
  artifact still exists. If the config arguments are usable but its session ID
  is absent, unsupported, or stale, Pair uses those arguments for a fresh native
  conversation and warns that it cannot resume. Explicit arguments still win
  over usable config arguments, while a valid saved session ID is composed with
  whichever arguments win.
- Returning to a previous driver therefore resumes that agent's prior native
  conversation and uses the new continuation to catch it up on work performed
  by the intervening driver.

### Exclusive handoff transaction and lock

Pair performs a handoff in these phases:

1. Acquire an exclusive repo/tag handoff lock by atomic create. Every launcher
   path that can create, attach, resume, rename, or hand off that tag honors the
   lock. A lock contains a transaction ID, owner PID, source session, and start
   time; live owners reject competitors, while a dead owner routes through the
   journal recovery procedure before new work proceeds. Recovery first wins an
   atomic claim sidecar and compare-validates the observed dead-owner record,
   so only one launcher may replay a stale journal.
2. Under the lock, resolve and validate the source evidence, target arguments,
   target command/session name, and source recovery material without changing
   the live session. Source recovery material is valid only when it can compose
   a launch and, when claiming native resume, its session artifact exists. If it
   is not recoverable, the confirmation explicitly says that stopping the
   source is irreversible and requires a second affirmative choice.
3. Create a transaction directory and atomically publish a journal in
   `prepared` state. Before quiescence this journal records only immutable
   intent and preflight data: the tag, agents, public session, launch nonces,
   and recovery inputs. It must not treat a live draft digest, queue manifest,
   or proposed queue key as authoritative.
4. Advance durably to `source-stop-requested`, then stop the source in handoff
   mode, which suppresses normal quit cleanup of tag-scoped and scrollback
   files, and wait for its pair-wrap and draft Neovim processes to exit. That
   exit is the quiescence boundary: only afterward may Pair snapshot the final
   draft, queue, raw transcript, and resize events. Pair then advances to
   `source-stopped`; step 6 publishes the single `snapshot-complete` transition
   after every authoritative backup and manifest field exists. Recovery from
   `source-stop-requested` first observes the exact source: a bounded-stable
   intact source is left running and the handoff rolls back, while a partially
   stopping or quiescent source completes quiescence and relaunches when
   recoverable. This closes the crash gap without turning intent into evidence.
5. Publish a collision-safe immutable transcript bundle by building a temporary
   directory and renaming it into
   `parked/<tag>/<timestamp>-<agent>-<transaction-id>/`. The bundle contains
   `transcript.txt` rendered in plain-text continuation substrate format,
   `scrollback.raw`, `events.jsonl`, and `metadata.json` with tag, agent, native
   session ID when known, public session, cutoff time, and transaction ID. The
   source files are stable because their writers have exited; allocation does
   not rely on second-resolution timestamps alone.
6. Back up the final draft and queue manifest in the transaction directory. If
   the draft is non-empty, add it with the queue store's existing logical
   push-front operation: allocate one unused six-digit key below the current
   minimum (or the canonical middle key for an empty queue). Existing queue
   filenames are never rekeyed, so their stable identities and order remain
   unchanged. Prepare the generated handoff instruction as the new `*` draft.
   The atomic `snapshot-complete` journal transition publishes this
   post-quiescence draft digest, exact stable-key manifest, allocated front key,
   and backup paths as the sole authoritative recovery state; it replaces, and
   never validates against, any pre-stop observation.
7. Commit the input transition as two individually atomic writes under the
   journal: create the new queue item first, advance to `queue-committed`,
   atomically replace the draft second, then advance to `input-committed`.
   Transaction-retained inodes let recovery prove whether either effect landed
   before its following journal write; it never removes a colliding file or
   restores over unrelated draft content. History is never part of the
   mutation.
8. Launch the target under the same tag/public session identity and an exact
   launch nonce. No target starts before the source is quiescent. Pair advances
   to `target-ready` only after receiving the matching readiness signal, then
   persists explicit agent defaults, marks the journal `complete`, and releases
   the lock.

On any Pair launcher entry, an incomplete journal with a dead lock owner is
recovered according to its last durable state before the tag can be used. Thus
the filesystem/process sequence is crash-recoverable, not falsely described as
one atomic operation.

The tag lock serializes decisions, not terminal attachment lifetimes. Ordinary
attach/resume holds it only through stale-journal recovery and selection, then
releases it before blocking on the existing session. Ordinary create holds it
through the matching readiness point, then releases it while the user remains
attached. A handoff holds it from preflight through rollback or target readiness.

### Launch readiness

- Pair-wrap atomically publishes an agent-ready record only after the target PTY
  process has started successfully. It contains the tag, agent, public session,
  launch nonce, and agent PID.
- Before launch Pair removes any stale record, then accepts only a record whose
  nonce and identity match the transaction and whose PID is alive. A bounded
  timeout or child exit before that signal is a failed launch. The blocking
  Zellij handoff must be orchestrated so the launcher can observe readiness
  while the child is running; waiting for detach/exit is not readiness.
- This readiness point is the sole meaning of target or recovery launch
  success. The durable `target-ready` transition is also the handoff's commit
  point: before it, recovery rolls back; at or after it, recovery finalizes
  forward and the target remains the exclusive driver.
- Persisting explicit repo/agent defaults and changing the journal from
  `target-ready` to `complete` are post-commit finalization. If either write
  fails, Pair reports it and retains enough data in the `target-ready` journal
  for the next launcher entry to retry idempotently. It must not stop a ready
  target or relaunch the source merely because finalization failed.

The generated `*` instruction identifies the source tag, source agent, native
session ID when known, and immutable `transcript.txt` path. It tells the target
agent to follow the continuation datatype's dead-agent procedure: draft the
continuation, show the required preview to the user, and finalize the approved
body through `pair continuation --no-restart` so writing the source
continuation does not compact or restart the new driver. It then continues the
work in the current session. The writer retains its normal commit/push behavior.
The instruction must be actionable without relying on the target session's
`PAIR_TAG`, which names the shared work rather than the transcript's source
agent.

### Failure and recovery

- Any failure before the source session is stopped leaves that session and all
  tag-scoped files unchanged, apart from the recoverable transaction journal
  and lock.
- Draft/queue replacement uses a journaled logical push-front with stable
  filename keys; it never reuses a display index across a mutation.
- If a failure occurs after the source stops, Pair restores the original
  draft/queue snapshot and, when recovery material was accepted as valid,
  attempts to relaunch the source driver from its saved `(tag, agent)`
  configuration—but only before the `target-ready` commit point. A target that
  exits or times out before readiness is torn down by its exact launch nonce and
  public session; Pair waits until its agent, pair-wrap, Neovim, and Zellij
  processes are proven gone before restoring files or starting source recovery.
  Source recovery is successful only when the exact recovery launch reaches the
  readiness contract above; a spawned process or Zellij row alone is
  insufficient. Pair reports the target failure, each restored artifact, and
  whether source recovery became ready. If recovery was knowingly unavailable
  or fails because the binary/provider/session artifact is gone, Pair leaves
  the parked transcript and restored tag state intact and prints a concrete
  manual recovery command.
- A failed handoff must not fall back to a sibling tag, lose a queued prompt,
  overwrite the source transcript, or claim success with neither driver live.
- Transcript rendering happens after quiescence. If it fails, Pair restores and
  relaunches the source when possible; proceeding with a tag-state-only handoff
  requires a new explicit confirmation after the rendering error.

### Architecture

- Reuse the existing repo-scoped session snapshot, history cutoff, fzf runtime,
  session-name index, continuation renderer/writer contracts, agent argument
  composition, and queue primitives (ARCH-DRY).
- Keep picker-row construction, driver classification, argument precedence,
  queue-transition planning, journal-state transitions, and handoff/recovery
  planning as deterministic functions. Zellij, locking, filesystem staging,
  rendering, readiness observation, prompts, and process launch remain in the
  runtime shell (ARCH-PURE).
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
- Pre-stop failures are non-mutating; post-stop failures before the durable
  `target-ready` commit restore input state and attempt to recover the source
  driver with a clear outcome. Failures at or after `target-ready` finalize the
  target forward.
- Unit tests cover pure selection, filtering, allocation, argument precedence,
  queue ordering, driver conflicts, journal transitions, and recovery plans; a
  process-level fake covers a complete live Claude-to-Codex handoff plus
  concurrent rejection, mutation/quiescence, readiness timeout, and failures at
  each post-stop boundary.
- Atlas documentation describes tag-as-work identity, exclusive agent drivers,
  handoff state ownership, and repository-scoped agent defaults.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.8 impl=0.08
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: smaller-go-module design=0.06 impl=0.16
item: greenfield-go-module design=0.3 impl=0.28
item: greenfield-go-module design=0.3 impl=0.28
item: greenfield-go-module design=0.3 impl=0.28
item: greenfield-go-module design=0.3 impl=0.28
item: greenfield-service design=3.0 impl=3.2
item: api-integration design=0.4 impl=0.4
item: api-integration design=0.4 impl=0.4
item: tui-screen design=0.4 impl=0.4
item: tui-screen design=0.4 impl=0.4
item: cross-cutting-refactor design=0.1 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: lua-neovim design=0.2 impl=0.4
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
item: milestone-review design=0.08 impl=0.12
design-buffer: 0.15
total: 16.70
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.*

## Plan

- [x] Write the durable implementation plan after the approved spec passes review.
- [ ] M1 — Define explicit launch intent, repo-agent default precedence, driver classification, and picker policy.
- [ ] M2 — Add nonce-bound readiness and wire automatic repo-agent defaults.
- [ ] M3 — Add the crash-safe lock/journal, shared queue push-front, and immutable transcript bundle.
- [ ] M4 — Wire exclusive handoff into the normal picker and prove end-to-end recovery.

## Log

### 2026-07-16

Claimed before design and entered `sdlc start-plan`. The initial
`pair <agent> resurrect` idea was replaced during brainstorming with an
exclusive agent handoff on the same work tag. The design reuses tag-scoped
draft/history/queue state, parks agent-scoped scrollback for continuation
dead-agent mode, resumes returning agents' native conversations, and adds
repository-scoped per-agent launch defaults. ARCH-DRY, ARCH-PURE, and
ARCH-PURPOSE shaped the boundaries above.

## Revisions

### 2026-07-16T13:58:41-07:00 — fresh-eyes spec review

The first review found that “atomic handoff” lacked a real cross-process and
crash boundary. Added the repo/tag lock, source-process quiescence, durable
transaction journal, stable queue-key push-front, immutable plain-text
transcript bundle, nonce-bound agent readiness, explicit config-validity rules,
corrupt-driver refusal, readiness-gated default persistence, and a concrete
`pair continuation --no-restart` dead-agent path. Recovery now distinguishes
restoring files from successfully relaunching a ready source driver.

### 2026-07-16T14:00:40-07:00 — second review state-machine corrections

Made the post-quiescence snapshot manifest authoritative instead of recording
mutable pre-stop observations. Added nonce-scoped failed-target teardown and a
proved-quiescent boundary before rollback. Defined durable `target-ready` as the
handoff commit point: earlier states roll back, while defaults/journal failures
afterward finalize forward without reviving the source. Clarified that tag locks
cover decision/readiness windows, not the full interactive attachment lifetime.

### 2026-07-16T14:01:25-07:00 — third review boundary correction

Removed an accidental early `snapshot-complete` transition: quiescence records
only `source-stopped`, and the authoritative manifest publishes
`snapshot-complete` exactly once. Qualified acceptance recovery as pre-commit;
failures at or after `target-ready` finalize forward.

### 2026-07-16T15:13:48-07:00 — durable implementation plan

Added the reviewed four-milestone implementation plan and calibrated v3.1
estimate. Planning made every effect-before-journal window explicit: source
stop intent is distinct from observed teardown, queue and draft mutations retain
inode evidence for reconciliation, stale recovery has its own atomic claim, and
transcript-unavailable paths require separate confirmation. The plan was
approved chunk by chunk after fresh-context review.

### 2026-07-16T15:53:01-07:00 — code-entry gate refinement

The plan-quality gate found that the first estimate aggregated four milestone
reviews plus the issue-close review into one item and underrepresented the 18
implementation tasks. Re-derived the v3.1 estimate as 9.58 hours with five
explicit review primitives and per-module/UI/Lua/refactor items. It also found
two readiness JSON descriptions; the plan now puts the wire schema and codec in
one shared `cmd/internal/readiness` package consumed by launcher and pair-wrap
(ARCH-DRY).

### 2026-07-16T15:55:14-07:00 — second code-entry refinement

The next gate found that module counting still hid the dominant work: the
journaled multi-process coordinator, OS teardown/recovery, and hermetic crash
matrix form a service-scale subsystem. Re-derived the estimate as 16.70 hours:
four bounded Go modules cover defaults/readiness/queue/transcript, one
greenfield-service primitive covers coordinator+locking+recovery, and two
integration primitives cover Zellij/process and native-agent boundaries. The
gate also identified a missing forward draft-write seam; the plan now adds
store-owned `CommitHandoffDraft`/`ReconcileHandoffDraft` operations and OS/fake
tests so orchestration remains a thin effect interpreter (ARCH-PURE).
