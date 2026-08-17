---
id: 000115
status: working
deps: []
github_issue:
created: 2026-07-16
updated: 2026-08-16
estimate_hours: 4.41
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

### Current revived scope (2026-08-16)

This revival keeps the original tag-as-work goal but deliberately does not
restart from the abandoned live handoff coordinator. The first deliverables are
the low-risk substrate that `pair <agent>` needs before any cross-agent recovery:
repository-scoped per-agent launch defaults, followed by explicit-agent picker
routing that makes the user's switch intent visible without taking over a live
foreign-agent session.

- `pair <agent> -- <args...>` records `<args...>` as the local default for that
  `(repo, agent)` only after a successful launch readiness point.
- `pair <agent>` uses that agent's repo-scoped default when creating a new
  session and no tag-specific saved config is available.
- `pair <agent> --` with no following args intentionally clears the repo-scoped
  default after launch readiness.
- Tag-specific saved configs (`config-<tag>-<agent>.json`) keep priority over
  repo-agent defaults when resuming an existing tag; native session IDs remain
  tag-specific.
- The storage is local machine state under Pair's repo-scoped data directory,
  not version-controlled project configuration.
- The old live takeover flow remains historical design context until a later
  milestone replaces it with a safer source-quiescent recovery flow.

M2 defines the launcher entry points around that substrate:

- Bare `pair` means choose work in the current repo: attach/resume an available
  row, or create a new Pair session when a new name is used. It does not imply a
  cross-agent continuation.
- `pair <agent>` means choose work for the requested driver. Same-agent live
  rows are attachable, same-agent exited rows resume normally, and different-
  agent exited/recent rows enter the continuation-backed switch path. Different-
  agent live rows are displayed as unavailable so the picker explains why they
  cannot be attached by the requested agent.
- `pair <agent> -- <args...>` is the explicit-parameter extension of
  `pair <agent>`: it keeps the current create/new-or-resume behavior with the
  supplied launch parameters, and records those parameters as the repo-agent
  default only after launch readiness.
- A different-agent switch from an exited/recent row uses an existing
  continuation document for that tag when one exists. If none exists, Pair still
  starts the requested agent under the same tag and seeds an auto-continuation
  draft that points the agent at the tag's persisted draft, log, queue, and any
  parked scrollback. Pair must not allocate a sibling tag just because the
  source agent did not prewrite a continuation, and the generated prompt should
  stay on Pair's TTY/Pair-state continuation substrate rather than agent-native
  transcript files.

### Historical full handoff design (deferred)

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

- Revived M1: repository-scoped per-agent defaults are reused by `pair <agent>`
  on new-session creation, replaced by successful explicit `-- <args>`, and
  cleared by successful explicit empty `--`.
- Revived M1: tag-specific saved configs still win over repo-agent defaults, so
  `pair resume <tag>` and historical tag picks preserve their existing native
  resume behavior.
- Revived M1: tests cover parser intent, default file codec/path, precedence,
  readiness-gated persistence, and no persistence after abort or failed launch.
- Revived M1: README and atlas explain local repo-scoped agent defaults and the
  relationship to tag-specific configs.

Historical full handoff criteria below are retained as deferred context:

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
- Unit and fake-integration tests cover pure selection, filtering, allocation,
  argument precedence, queue ordering, driver conflicts, journal transitions,
  lock/recovery paths, quiescence timeouts, readiness/default failures, and
  post-stop rollback/forward-finalization windows; a process-level fake covers a
  complete live Claude-to-Codex handoff through the public launcher with real
  binary wiring.
- Atlas documentation describes tag-as-work identity, exclusive agent drivers,
  handoff state ownership, and repository-scoped agent defaults.

## Revisions

- 2026-08-16 (revival): replaced the immediate implementation target with a
  safer first milestone. The original live handoff coordinator remains
  historical context because its production quiescence proof and acceptance fake
  were unsound. The revived M1 implements repository-scoped per-agent defaults
  first; later milestones may redesign `pair <agent>` as a work selector and
  source-quiescent recovery flow without importing the abandoned coordinator.

- 2026-07-28 (close-review REWORK): the issue-close boundary review
  (`workshop/plans/000115-...-close-review.md`) found the headline flow does
  not work outside the acceptance harness, so the issue does NOT close here; an
  M5 rework milestone is added. The substrate is sound — the journal/recovery
  model, durability primitives, and M1-M4 hardening all reviewed well — but
  four Critical defects sit between it and a working agent switch, and the
  process-level test that appeared to prove the switch was passing for a fake
  reason (its fake `zellij` wrote the cleanup acknowledgement, which is a
  different `pair` process's effect, not zellij's). Scope for M5:
  - **C1 — live handoff destroys the source, then times out.** Source
    quiescence gates on `handoff-<tag>-<txn>.cleanup-ack`, whose only writer
    (`lifecycle.go:66`) is reachable only via a quit marker that Alt+x/Alt+n
    write and `zellij delete-session --force` does not — so
    `ObserveJournaledSource` returns `partially-stopping` forever. Reordering
    `runCleanup` is not sufficient: a DETACHED source has no launcher process
    to acknowledge at all. Quiescence needs evidence the coordinator can
    observe itself (zellij row absent AND no live recorded pair-wrap/agent/nvim
    PID), with the ack kept only as a fast path.
  - **C2 — a non-live different-driver row** routes to handoff but stops
    `pair-<tag>` instead of the tag's indexed session (wrong public identity,
    against the spec's canonical-identity rule) and aborts because real zellij
    exits 2 for a missing session. Non-live rows should skip the
    stop/quiescence phase entirely and resolve the name through the
    session-name index.
  - **C3 — a leaked recovery claim permanently disabled the tag** (every
    attach/create/rename/handoff failed until the file was deleted by hand).
    FIXED in this branch with mutation-proven regressions; `RecoverTag` should
    still release/quarantine the claim on its error path.
  - **C4 — `queue_push_front` invokes `$PAIR_HOME/bin/pair`**, which does not
    exist in embedded-runtime installs, so the Alt+Enter draft push-front
    silently fails there. `make test-queue` cannot see it because it pins
    `PAIR_HOME` to the checkout. This one is INDEPENDENT of the handoff feature
    and affects released pair — worth splitting into its own issue.
  Plus six Important findings, notably: the handoff never writes
  `config-<tag>-<agent>.json`, so a stale config permanently shadows the minted
  session id and orphans the conversation the handoff just created;
  `GatherDriverSnapshot` derives the scope key with `filepath.Base` instead of
  the canonical `scopeKeyFromDataDir`, silently dropping all live-session
  evidence whenever `PAIR_DATA_DIR` is set; and the acceptance fake models
  behavior the real dependency does not have (ARCH-MOCK), which is what made
  C1 and C2 invisible.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.06 impl=0.16
item: greenfield-go-module design=0.3 impl=0.28
item: greenfield-go-module design=0.4 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.5 impl=0.48
item: cross-cutting-refactor design=0.15 impl=0.2
item: atlas-docs design=0.05 impl=0.05
item: milestone-review design=0.08 impl=0.12
item: atlas-docs design=0.05 impl=0.05
design-buffer: 0.15
total: 4.41
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only.*

This estimate covers the revived M1 scope only. The smaller-module items cover
parser intent and wrap-side readiness publication; the greenfield-module items
cover the repo-agent default codec/policy and the shared nonce-bound readiness
record/matcher; the API-integration item covers create-flow wiring across saved
tag config, repo defaults, launch, and readiness-gated persistence. The
cross-cutting refactor item covers runtime-seam updates. The docs items cover
README plus atlas updates, and the single review item covers the one issue close
boundary for this revived milestone. Historical live handoff, lock/journal,
picker takeover, queue mutation, and transcript parking are deliberately
deferred and are not included here.

## Plan

- [x] Write the durable implementation plan after the approved spec passes review.
- [x] M1 — Define explicit launch intent, repo-agent default precedence, driver classification, and picker policy.
- [x] M2 — Add nonce-bound readiness and wire automatic repo-agent defaults.
- [x] M3 — Add the crash-safe lock/journal, shared queue push-front, and immutable transcript bundle.
- [x] M4 — Wire exclusive handoff into the normal picker and prove end-to-end recovery.
- [ ] M5 — Make the switch work outside the acceptance harness (close-review REWORK).
      DEFERRED 2026-07-28 — see the abandonment note in ## Log.

## Log

### 2026-08-16
- revived — The original same-tag live handoff branch was abandoned unmerged
  because its source-quiescence proof depended on a fake zellij effect and
  still had Critical recovery failures. The renewed direction keeps the tag as
  work identity but avoids live takeover: `pair <agent>` should act as the
  selector over existing work, and a different-agent pick should recover through
  continuation/parked-transcript material after the source is quiescent.
- M1 implementation — Re-derived the estimate for the revived repo-agent
  defaults scope (4.41h) and preserved the abandoned branch by renaming it to
  `abandoned/000115-resurrect-a-session-across-agents-20260728` before starting
  the fresh implementation branch. Implemented parser intent bits, pure
  repo-agent default codec/path, launch-argument precedence, nonce-bound
  readiness records, wrap-side readiness publication, and readiness-gated
  explicit default persistence. Focused checks so far:
  `go test ./cmd/internal/launcher -run TestParseArgs -count=1`,
  `go test ./cmd/internal/launcher -run 'Test(AgentDefault|LaunchArg|ScopedPaths)' -count=1`,
  `go test ./cmd/internal/readiness ./cmd/internal/launcher ./cmd/internal/wrapcmd -run 'Test.*Ready|Test.*Readiness' -count=1`,
  and
  `go test ./cmd/internal/launcher -run 'TestRunLaunch.*(Default|Config|Codex|Resume)' -count=1`.
  Final verification passed:
  `go test ./cmd/internal/launcher -count=1`,
  `go test ./cmd/internal/readiness ./cmd/internal/wrapcmd -count=1`,
  `go test ./... -count=1`, and `git diff --check`.
- M2 implementation — Updated the launcher entry-point split around the
  corrected design. `pair <agent>` without `--` now uses explicit-agent picker
  policy: same-agent live rows attach, different-agent live rows are visible but
  unavailable, and different-agent historical rows use a matching continuation
  doc when present or generate an auto-continuation draft when no doc exists.
  This fixes the Alt+x smoke-test case (`pair codex` in parley.nvim selecting
  the exited Claude `parley_nvim` tag) and satisfies ARCH-PURPOSE by preserving
  the same tag rather than refusing or allocating a sibling. `pair <agent> --
  <args...>` now keys off separator intent rather than
  non-empty args, so repo-agent defaults do not accidentally bypass the picker.
  Smoke follow-up removed the agent-native transcript reference again after
  confirming continuation distillation uses Pair's TTY scrollback / Pair-state
  substrate, not Claude/Codex native transcript files.
  Focused checks passed:
  `go test ./cmd/internal/launcher -run 'TestDecideLaunchExplicit|TestBuildPickRows|TestRunLaunchExplicitAgentDifferentHistorical' -count=1`
  and `go test ./cmd/internal/launcher -count=1`. Final M2 verification passed:
  `go test ./cmd/internal/launcher -count=1`,
  `go test ./... -count=1`, and `git diff --check`.

### 2026-07-28
- Merged current `main` into the branch after a 9-day gap (the branch predated
  #116-#124: layout2/3 selection, frame-title tab rename, KKP forwarding, and
  the floating -> tiled right-terminal pivot). Nine conflicted files resolved;
  the interesting ones were structural rather than textual: the Runtime seam
  (#115's async `StartSession` supersedes main's blocking `LaunchSession`,
  keeping main's `ProbeLiveLayout`), and `createflow`'s attach + create paths,
  which now carry BOTH #115's tag lock / selection revalidation / launch nonce
  AND main's live-layout resolution and `PAIR_WORKBENCH_LAYOUT` export.
- Two defects that merged cleanly but were behaviorally wrong: (1) the handoff
  relaunch hardcoded `zellij/layouts/main.kdl`, a file main deleted when layout
  selection landed — a handoff would have launched from a nonexistent layout;
  it now resolves the tag's layout so the workbench survives the driver switch.
  (2) #115's async launch introduced a pre-ready failure point that did not
  exist when `LaunchSession` blocked: a readiness timeout stopped the session
  but left the `workbench-layout-<tag>` record behind as if it had come up;
  that path now restores the record. At merge time only (2) carried a
  regression — see the close-review entry below, which corrects that and adds
  coverage for (1).
- Post-merge fresh-eyes review (run out-of-band: the binary's dispatcher hit
  `fork/exec: argument list too long`, because this issue's own three review
  sidecars were 6.6 MB of the 7.3 MB close window; trimming them to verdict +
  findings brought the window to 477 KB and restored normal dispatch).
  Verdict FIX-THEN-SHIP; all three Important findings fixed, each proven by
  mutation testing rather than assertion:
  (1) merge fix #1 had NO coverage — reverting it to the hardcoded `main.kdl`
  left the whole launcher package green. Added a handoff-layout regression
  driven through the real `runHandoff`; that mutation now fails it. The Log
  claim that both fixes carried regressions was false and is corrected above.
  (2) the handoff resolved its layout from the RECORD alone while attach used
  `resolveLiveLayout` (record → live probe → persist). A tag predating the
  layout record would be silently downgraded to layout2, destroying a layout3
  right terminal — the same defect class the merge had just fixed once. The
  handoff now shares the one resolver and persists the result, so an explicit
  `--layoutN` on a handoff can no longer desynchronize the record from reality
  (which would have made a later `pair --layoutN` a silent no-op).
  (3) the attach path released the tag lock BEFORE main's layout-conflict
  branch, which is not an attach at all — it deletes the session and recreates
  the workbench, the mutation create/handoff hold that lock for. A concurrent
  handoff could have had its source deleted underneath it. The lock is now held
  across it and released just before `runAttach`.
  Minors also fixed: `PAIR_WORKBENCH_LAYOUT` is now exported on the handoff and
  attach paths (it was create-only, a trap for the first consumer); the fake's
  `ProbeLiveLayout` no longer succeeds-by-default for sessions it doesn't have
  (it hid production's unrecognized-signature error path, ARCH-MOCK); atlas
  drift repaired (`LaunchSession`, `main.kdl` — both deleted upstream) and a
  new "Workbench layout across launch paths" section documents the resolver
  contract; a swallowed blank line before a lessons heading restored.
- Verified: `go build ./...`; `go test ./... -count=1`; full `make test` (only
  the pre-existing #117 `scrollback-open` Alt+x drift fails, identically on
  main); `tests/pair-agent-handoff-test.sh`; `tests/review-handoff-test.sh`;
  term-pane-shortcuts, review-toggle, review-window, embedded-runtime.
  Environment note: the worktree could not run `make` at all (its `Makefile`
  symlink resolves to `../ariadne/Makefile`, absent under `worktree/pair/`);
  fixed by adding the sibling `ariadne` symlink in the worktree parent.

- 2026-07-19: closed M4 — M4 review fixes committed after go test ./cmd/internal/launcher -count=1, make test-agent-handoff, go test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1, go test ./..., full make test, and git diff --check. The handoff path now restores transaction input during snapshot-complete and queue-committed recovery, stops target only when observed during input-committed recovery, replays source recovery launch environment, retains the source-stop-requested journal/marker/tag lock on live source quiescence timeout for stale-owner recovery, captures transcript cutoff after durable source-stopped, refreshes nonce-bound ready evidence and rejects fallback source identities as quiescent proof, plans queue/draft input after source quiescence inside the coordinator, transfers lock ownership before coordinator entry so retryable journals keep stale-lock recovery, requires durable source cleanup acknowledgement before source quiescence, propagates zellij delete-session failure, journals target ownership and replays agent sidecar plus ledger publication before target-ready forward finalization, proves target quiescence before pre-ready rollback restores input/source, reports cleanup-ack failures, revalidates locked attach selections, and preserves the process-level same-tag handoff smoke; review verdict: FIX-THEN-SHIP
### 2026-07-16
- 2026-07-16: closed M2 — wrap/launcher suites and go test ./... pass; nonce readiness, automatic resume precedence, stale explicit resume fallback, readiness-gated defaults, and plan traceability are covered; per-milestone active-time increment unavailable, so no hours guessed; review verdict: SHIP
- 2026-07-16: closed M1 — launcher unit suite and go test ./... pass; manual fresh review of 7c6fa26..HEAD returned FIX-THEN-SHIP and its unsafe agent-path finding is fixed; --no-judge suppresses only the automatic dispatcher's incorrect 2.1 MB historical window; active-time telemetry unavailable, so no hours guessed; review verdict: not-run

Claimed before design and entered `sdlc start-plan`. The initial
`pair <agent> resurrect` idea was replaced during brainstorming with an
exclusive agent handoff on the same work tag. The design reuses tag-scoped
draft/history/queue state, parks agent-scoped scrollback for continuation
dead-agent mode, resumes returning agents' native conversations, and adds
repository-scoped per-agent launch defaults. ARCH-DRY, ARCH-PURE, and
ARCH-PURPOSE shaped the boundaries above.

### 2026-07-16 — M1

Implemented the pure policy foundation with test-first red/green cycles. Raw
argv now distinguishes an implicit default agent from an explicitly named one
and records `--` even when its argument vector is empty. Added the validated
repo-scoped `agent-default-<agent>.json` codec/path and one deterministic
explicit → tag config → repo default precedence function, including canonical
resume composition and warnings for malformed, mismatched, stale, or
unsupported saved sessions (ARCH-DRY, ARCH-PURE).

Added source-labelled driver evidence classification and a separate agent-picker
projection. Consistent live ownership wins over historical ledger evidence;
missing indexes, multiple live sessions, and conflicting agents are disabled
without guessing. The picker policy preserves bare-picker behavior while an
explicit target agent can see attached, detached, exited, and recent-inactive
work and resolve it to attach, same-tag create, or first-class handoff. The live
runtime remains unchanged until the later integration milestone.

The M1 fresh review returned `FIX-THEN-SHIP`: the new default codec/path allowed
separator and traversal-like agent identifiers. Added one shared safe-agent
identifier predicate, applied it at both codec and path boundaries, and covered
slash, backslash, dot-segment, empty, and NUL-adjacent path construction cases
(ARCH-DRY). The automatic review dispatcher initially selected the issue-file's
old `issue-sync` history and exceeded the OS argv limit; the documented manual
judge was run against the actual M1 branch window `7c6fa26..HEAD` instead. M1 is
therefore finalized with `--no-judge` only to suppress that broken duplicate
dispatch; it does not waive the mandatory fresh review. The manual verdict and
its resolved finding are the boundary evidence.

### 2026-07-16 — M2

Added one shared stable readiness record consumed by the launcher and emitted
by pair-wrap only after PTY start and agent-PID publication. Each launch now
mints a nonce independently from any agent-native session ID, removes stale
evidence before process start, and accepts only an exact tag/agent/public-session
match whose recorded PID is alive. Fresh Zellij creation exposes start,
readiness, wait, and teardown as one cached child handle, so the launcher can
observe startup while the user's attached session remains blocking afterward
(ARCH-DRY, ARCH-PURE).

The production create flow now applies the M1 argument policy directly: valid
tag configurations resume automatically, stale native artifacts retain usable
arguments with a warning, malformed configurations fall through to the
repo-agent default, and the retired config picker is gone. Explicit defaults,
including an intentional empty `--`, persist only after readiness; timeout or
early exit stops and reaps the attempted session without changing the default.
A failure writing the default after readiness warns but preserves the usable
target.

Process-level readiness tests cover exact and stale nonces, early child exit,
named teardown, agent reaping, and proof of Zellij absence. The fake Zellij is
the Go test executable itself rather than a temporary shell script, avoiding an
interpreter-open race under full-suite load. The launcher readiness tests passed
20 consecutive runs, and the focused wrap/launcher suites plus `go test ./...`
pass. The M2 close gate could not derive a per-milestone active-time increment;
it exposed only a 0.03 h issue-level window from telemetry base `c726da9e` and
required a manually supplied increment. M2 therefore uses `--no-actual` rather
than guessing a calibration value.

The first M2 boundary review returned `REWORK`: explicit resume syntax for a
missing native artifact was warned about but still forwarded, and README still
described the retired config picker. Explicit vectors are now normalized before
the resumability branch and the canonical resume binding is added back only for
a proven artifact. Pure-policy and create-flow regressions cover Claude, Codex,
and Agy stale explicit forms; README now documents automatic precedence,
fresh-session fallback, and readiness-gated defaults (ARCH-PURPOSE).

The second M2 review found no behavioral defect and passed ARCH-DRY,
ARCH-PURE, and ARCH-PURPOSE, but returned `REWORK` because the durable plan still
located the launch request/OS implementation in obsolete files and named a
standalone `ReadinessOps` seam that the final encapsulated launch handle did not
need. The Core concepts table and Task 7 file list now describe the delivered
structure exactly.

### 2026-07-17 — M3
- 2026-07-17: closed M3 — race/unit suites, same-inode mutation and signed-key regressions, queue/runtimebundle checks, deterministic generation, real queue-pane integration, and go test ./... pass; immutable draft recovery and all first-review findings are resolved; per-milestone active-time measurement is unavailable, so no hours are guessed; review verdict: FIX-THEN-SHIP

Added the versioned handoff journal and pure recovery planner before wiring its
filesystem effects. The OS store now owns exclusive tag locks, stale recovery
claims, monotonic durable journal transitions, post-quiescence snapshot
manifests, retained-inode queue/draft reconciliation, rollback restoration, and
unresolved-journal discovery. This keeps orchestration policy independent of
filesystem mechanics while making collision and effect-before-journal cases
explicit (ARCH-PURE).

Moved logical queue push-front behind the shared `pair queue push-front` route
and updated the Neovim pane to consume it. The same collision-safe allocation
and atomic publication path is now exercised by Go race tests and the headless
queue suite rather than duplicated between Go and Lua (ARCH-DRY). Added
immutable transcript publication that copies quiescent raw/events sources,
renders the copied stream through the existing plain scrollback renderer, and
atomically publishes a metadata-bound four-file parked bundle.

The M3 race matrix exposed two lifetime/publication gaps and one unsafe test
fixture. Exclusive lock/claim records now publish a fully written, fsynced temp
inode with a no-replace hard link, so a losing launcher cannot observe partial
JSON. The scrollback renderer closes its emulator reply writer and joins the
drainer before closing emulator state. Wrap batching tests use a synchronized
sink when polling output concurrently. Transaction-directory creation also
fsyncs its parent before journal publication.

The M3 close gate could not derive a per-milestone active-time increment. It
reported only a 0.10 h whole-issue window from telemetry base `c726da9e` and
required a manually supplied increment, so M3 uses `--no-actual` rather than
guessing a value that would pollute calibration.

The first M3 boundary review returned `REWORK`: a hard-linked draft backup
shared mutable content with the live draft, signed six-character queue keys
passed validation, and README omitted the new runtime queue helper. Draft
snapshots now use immutable copied inodes and reconciliation proves the live
size/digest before replacement; same-inode mutation is a covered conflict.
Queue keys are exactly six ASCII digits at both publication and journal
validation boundaries, and README identifies `pair queue push-front` as an
internal/runtime helper (ARCH-PURPOSE).

The second M3 review returned `FIX-THEN-SHIP` with no code blockers. Its sole
finding was stale atlas wording that still described draft snapshots as retained
source inodes; the atlas now distinguishes immutable draft bytes/digest checks
from inode identity used for transaction-published queue effects.

### 2026-07-17 — M4

Started the live integration milestone by fixing its post-lock contract before
process effects. `ResolveHandoffPreflight` reclassifies copied driver evidence,
refuses conflicts or changed sessions, and carries explicit transcript and
source-recovery availability into staged confirmations. The OS gatherer now
collects scoped session-index/live rows, agent sidecars, ledger history,
transcript paths, and saved configuration without choosing provenance itself.
Generated instructions name immutable source provenance and the exact
`pair continuation --no-restart` dead-agent procedure (ARCH-PURE, ARCH-DRY).

Added the first journaled handoff coordinator slice. The coordinator now writes
the forward journal sequence through `target-ready`/`complete`, publishes the
handoff marker, quiesces the source through an injected process seam, snapshots
input via the existing store, launches the target with readiness proof, persists
explicit defaults after readiness, and waits on the attached target. A
pre-commit target-readiness failure stops and waits the partial target, restores
journal-owned input, relaunches the recoverable source, finalizes rollback, and
releases the lock; stale-owner recovery now interprets the pure recovery plan
for `target-ready` forward finalization and `input-committed` rollback
(ARCH-PURE, ARCH-DRY). Verified with
`go test ./cmd/internal/launcher -run 'Test(RecoverTag|HandoffCoordinator)' -count=1`
and `go test ./cmd/internal/launcher -count=1`. Remaining M4 work includes the
full OS process/quiescence proof, richer restart reconstruction after process
death, picker/tag-lock wiring, and process-level acceptance.

Hardened the handoff-mode cleanup handshake. Source shutdown now publishes a
transaction-bound marker containing tag, agent, session, and transaction ID;
`runCleanup` consumes only a matching marker, writes a durable cleanup
acknowledgement, and skips ordinary quit teardown so tag-scoped state and
scrollback remain available for the handoff coordinator. Mismatched markers
fall through to normal cleanup. OS observation now classifies a marker-only
still-live source as `intact` for rollback, while an acknowledged absent source
is quiescent (ARCH-PURE, ARCH-PURPOSE). Verified with
`go test ./cmd/internal/launcher -run 'Test(OSRuntimeHandoffMarkerAndCleanupAck|OSRuntimeObserveJournaledSourceMarkerOnlyLiveIsIntact|RunLaunchQuitCleanup(HandoffMarker|Ignores))' -count=1`
and `go test ./cmd/internal/launcher -count=1`.

Made journaled launch recovery self-contained enough for stale-owner replay
after the original launcher process is gone. `RecoveryLaunch` now persists the
config dir, layout, and readiness path, the coordinator enriches both target and
recoverable-source launch material before writing `prepared`, and recovery
uses those persisted paths instead of process-local request fields. Added a
post-commit default-persistence regression: once `target-ready` is durable, a
default write failure leaves the target live, the lock/journal retryable, and
does not roll back input or stop the target (ARCH-PURPOSE). Verified with
`go test ./cmd/internal/launcher -run 'Test(Handoff|RecoverTag)' -count=1` and
`go test ./cmd/internal/launcher -count=1`.

Wired explicit-agent normal picker selections into the handoff coordinator.
`pair <agent>` now reroutes through the agent-aware picker when repo-scoped work
exists, includes attached work, resolves different-driver rows to
`ActionHandoff`, and runs the lock/preflight/confirmation/journaled coordinator
without attaching the source. Declining the confirmation releases the tag lock
without journal, launch, or attach mutation. Help text now documents same-tag
driver switches and readiness-gated repository defaults. Verified with
`go test ./cmd/internal/launcher -run 'Test(BuildAgentPickRows|ResolveAgentPickSelection|RunHandoffSelection|LaunchNativeHelp)' -count=1`,
`go test ./cmd/internal/launcher -race -count=1`, and `git diff --check`.

Added the M4 process-level acceptance target and fixed the smoke blockers it
exposed. Production now owns ANSI-row stripping instead of depending on a
test-only helper, scoped live rows infer their driver from the tag ledger/sidecar
when zellij itself has no agent label, and handoff target/recovery launches
export the durable `PAIR_*` environment before starting zellij. The old
agent-args guard now preserves explicit target args for `ActionHandoff` while
still dropping them for non-handoff picked attaches/resumes. `make
test-agent-handoff` builds the real binary, fakes zellij/fzf only at the process
boundary, switches a live scoped Claude tag to Codex, and verifies no resurrect
sibling, source-before-target ordering, parked transcript bundle, draft
push-front, queue/history preservation, handoff instruction provenance, and
readiness-gated Codex defaults. Verified with `go test ./cmd/internal/launcher
-run 'Test(RunHandoffSelection|OSRuntimeConfirmHandoffEnvOverride)' -count=1`
and `make -f Makefile.local test-agent-handoff`.

Completed the pre-boundary M4 verification matrix. Updated the embedded-runtime
fake zellij smoke to publish the same readiness record now required by the
native launcher, and hardened the handoff acceptance cleanup against inherited
dev-mode Go module caches. Verified with `go test ./cmd/internal/launcher
./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`, `make
test-agent-handoff`, `make -f Makefile.local test-queue`, `make -f
Makefile.local test-runtimebundle`, `make -f Makefile.local
runtimebundle-drift-check`, `go test ./...`, full `make test`, and `git diff
--check`.

Resolved the first M4 boundary review's REWORK findings. Target handoff launches
now share the normal target-agent launch-argument policy, including saved
config/default precedence, native resume validation, Claude session-id minting,
and Codex inline-mode args (ARCH-DRY). Source recovery is no longer assumed from
agent identity alone: it requires a matching saved config and existing native
session artifact, otherwise Pair shows the irreversible-source confirmation
before stopping the source (ARCH-PURPOSE). Post-stop/pre-target failures now
rollback, clear markers, and release locks, restoring draft/queue only after
those inputs may have been mutated. Stale handoff locks on explicit-agent
switches run journal recovery before acquiring the new lock, and conflicting
driver evidence now lists its sources. Verified with `go test
./cmd/internal/launcher -count=1`, `go test ./cmd/internal/launcher
./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`, `make
test-agent-handoff`, `go test ./...`, full `make test`, and `git diff --check`.

Resolved the second M4 boundary review's REWORK findings. Added the planned
`TagOperation`/`TagLockPlan` entity and routed attach/resume, create, rename,
and handoff through one stale-lock recovery policy; create holds the lock through
readiness and attach releases before blocking (ARCH-DRY, ARCH-PURPOSE). Source
stop identity now records existing pair-wrap/agent/nvim/ready-record evidence,
and OS quiescence no longer treats a missing zellij row as sufficient while a
matching handoff marker lacks an ack and recorded writer PIDs are still alive
(ARCH-PURE). Transcript render failures after source stop now ask the
post-render confirmation: declining rolls back, affirming continues with an
explicit tag-state-only instruction. The preflight path now shows both
irreversible-source and transcript-unavailable confirmations when both risks are
present. Verified with `go test ./cmd/internal/launcher -count=1`, `make
test-agent-handoff`, `go test ./cmd/internal/launcher ./cmd/internal/queuecmd
./cmd/internal/wrapcmd -race -count=1`, `go test ./...`, full `make test`, and
`git diff --check`.

Resolved the third M4 boundary review's REWORK findings. Handoff target startup
now publishes durable target ownership after `target-ready`: it appends the
target ledger entry, writes `agent-<tag>` to the target agent, and starts the
target session watcher/title poller with the same launch args the target
received (ARCH-DRY, ARCH-PURPOSE). The process-level handoff smoke now asserts
both `agent-work=codex` and a Codex ledger row after Claude-to-Codex switch, so
later resume/picker/cleanup inference cannot silently drift back to the source.
The M4 Core concepts table now names the delivered `DriverDecision` /
`HandoffPreflight` pair instead of the planned-but-not-created
`ResolvedDriver`. Verified with `go test ./cmd/internal/launcher -count=1`,
`make test-agent-handoff`, `go test ./cmd/internal/launcher ./cmd/internal/queuecmd
./cmd/internal/wrapcmd -race -count=1`, `go test ./...`, full `make test`, and
`git diff --check`.

Resolved the fourth M4 boundary review's REWORK findings. Post-stop source
quiescence now continues polling through `SourceIntact` and
`SourcePartiallyStopping` until quiescent or timeout; a non-quiescent timeout
rolls the stop request back by clearing the handoff marker, finalizing rollback,
and releasing the tag lock instead of stranding the transaction. Attach/resume
now revalidates the locked session and inferred driver before releasing the lock
and blocking on zellij, so stale picker rows cannot be acted on after recovery.
The plan's integration table now names the delivered `HandoffProcessOps` seam
instead of `SessionControlOps`. Verified with `go test ./cmd/internal/launcher
-count=1`, `go test ./cmd/internal/launcher ./cmd/internal/queuecmd
./cmd/internal/wrapcmd -race -count=1`, `go test ./...`, full `make test`, and
`git diff --check`.

Resolved the fifth M4 boundary review's REWORK findings. Target-ready ownership
is now journal material with an `OwnershipPersisted` marker, so stale-lock
recovery replays `agent-<tag>` and the target ledger row before finalizing
forward. Pre-ready target rollback stops the live launch handle, then uses the
same journaled process-stop/quiescence seam before restoring input or
relaunching the source (ARCH-PURE, ARCH-PURPOSE). Handoff queue inspection now
fails before source stop on real `ReadDir`/push-front errors, while absent queue
directories remain an empty queue. Handoff cleanup acknowledgement failures are
reported instead of swallowed. Verified initially with `go test
./cmd/internal/launcher -count=1`, `go test ./cmd/internal/launcher
./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`, `make
test-agent-handoff`, `go test ./...`, full `make test`, and `git diff --check`.

Resolved the sixth M4 boundary review's REWORK finding. The handoff marker now
stays durable until the coordinator clears it; source cleanup reads it without
consuming it and writes a transaction-specific cleanup ack. OS quiescence treats
a matching marker without that ack as still stopping even when zellij is absent
and no PID evidence remains, so snapshot/input mutation cannot advance on a
best-effort delete alone. `StopJournaledLaunch` now propagates zellij
delete-session errors before reaping Neovim. The process-level handoff smoke
simulates source cleanup and asserts the cleanup ack lands between source delete
and target start. Verified with `go test ./cmd/internal/launcher -count=1`,
`make test-agent-handoff`, `go test ./cmd/internal/launcher
./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race -count=1`, `go test ./...`,
full `make test`, and `git diff --check`.

Resolved the seventh M4 boundary review's REWORK findings. Queue/draft planning
now happens inside `HandoffCoordinator` after source quiescence, immediately
before `BackupHandoffInput`, so queue keys and draft presence are based on the
post-stop authoritative state rather than pre-stop observations. `runHandoff`
transfers lock ownership before coordinator entry; recoverable coordinator
errors such as target-ready default persistence failure leave the lock/journal
for stale-owner recovery instead of releasing the tag to a fresh non-recovery
operation (ARCH-PURE, ARCH-PURPOSE). Verified with `go test
./cmd/internal/launcher -count=1`, `make test-agent-handoff`, `go test
./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race
-count=1`, `go test ./...`, full `make test`, and `git diff --check`.

Resolved the eighth M4 boundary review's REWORK finding. OS source observation
now refreshes nonce-bound ready evidence from `agent-ready-<tag>.json` and
treats the synthetic `handoff-source` fallback nonce as insufficient process
proof even after cleanup ack. A matching marker can become quiescent only with a
real recorded PID or nonce-bound ready identity; otherwise it remains
`SourcePartiallyStopping` and the coordinator times out/rolls back instead of
snapshotting mutable input without evidence (ARCH-PURE, ARCH-PURPOSE). Verified
with `go test ./cmd/internal/launcher -count=1`, `make test-agent-handoff`, `go
test ./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd
-race -count=1`, `go test ./...`, full `make test`, and `git diff --check`.

Resolved the ninth M4 boundary review's REWORK findings. Snapshot-complete and
queue-committed rollback now restore transaction input before source recovery,
so effect-before-journal queue insertion cannot leave a duplicate queued draft.
Input-committed recovery stops the target only when the runtime observes that
target session, avoiding pre-launch delete failures. Source recovery replay now
reconstructs the persisted launch environment from the journal before starting
the source, and the M4 plan classifies PID-owning tag lock planning as an
integration point instead of a pure entity (ARCH-DRY, ARCH-PURE). Verified with
`go test ./cmd/internal/launcher -count=1`, `make test-agent-handoff`, `go test
./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race
-count=1`, `go test ./...`, full `make test`, and `git diff --check`.

Resolved the tenth M4 boundary review's REWORK finding. Live coordinator
timeout after `source-stop-requested` no longer clears the handoff marker,
finalizes rollback, or releases the tag lock without proved source quiescence;
it returns a retryable failure with the journal left at
`source-stop-requested` for stale-owner recovery. Transcript cutoff metadata is
now captured from the runtime clock immediately after the durable
`source-stopped` transition instead of reusing the launch-entry timestamp
(ARCH-PURE, ARCH-PURPOSE). Verified with `go test
./cmd/internal/launcher -count=1`, `make test-agent-handoff`, `go test
./cmd/internal/launcher ./cmd/internal/queuecmd ./cmd/internal/wrapcmd -race
-count=1`, `go test ./...`, full `make test`, and `git diff --check`.

## Revisions

### 2026-07-19T09:18:00-07:00 — M4 coverage contract reconciliation

The M4 FIX-THEN-SHIP review found the Done-when overstated process-level
acceptance breadth. The delivered proof uses unit and fake-integration tests for
concurrency/failure windows, with one hermetic public-launcher smoke for the
normal Claude-to-Codex switch. Updated the Done-when to match that evidence
instead of claiming every failure window is process-level.

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

### 2026-07-16T16:00:15-07:00 — third code-entry refinement

Made queue publication/reconciliation an injected store effect so coordinator
failure tests need no real filesystem (ARCH-PURE). Made explicit-default
persistence a named `target-ready` recovery step with a durable idempotence
marker, preventing forward finalization from silently dropping requested args
(ARCH-PURPOSE). Readiness publication now reuses `osfs.FS.WriteAtomic`
(ARCH-DRY). The task mapping exposed two separate service-scale systems—the M3
durability substrate and M4 live process coordinator—bringing the calibrated
estimate to 23.35 hours.

### 2026-07-28 — deferred; branch abandoned unmerged

Work stopped here by decision: the close-review REWORK (four Criticals, six
Importants — full agenda in ## Revisions and the close-review sidecar) put the
remaining work at roughly another milestone, and the priority moved elsewhere.
The branch `000115-resurrect-a-session-across-agents` was NOT merged. Its 44
commits stayed local to the authoring machine with no upstream, so nothing of
this reached `main` except this issue file and the review sidecars — kept
deliberately, because the findings are the durable value: they are specific,
reproduced against the built binary, and would otherwise have to be
rediscovered.

Two of the findings are worth remembering independently of this feature:
- **C4 is a branch-local regression, not a shipped bug.** `queue_push_front` on
  `main` is pure Lua and works everywhere; only the branch delegated it to
  `$PAIR_HOME/bin/pair`, which does not exist in embedded-runtime (Homebrew)
  installs. Nothing to fix on main — but the test gap that hid it is real:
  `make test-queue` pins `PAIR_HOME` to the checkout, so no Lua call site is
  ever exercised under a bundle-shaped `PAIR_HOME`.
- **The acceptance fake wrote an effect its real dependency never produces**
  (the handoff cleanup-ack, written by the fake `zellij`), which is what made
  the top two Criticals invisible while the suite stayed green. That is the
  ARCH-MOCK lesson already in `workshop/lessons.md`, earned again the hard way.

If this is picked up later, start from the close-review sidecar rather than the
branch: the substrate reviewed well ("the journal/recovery model is pure and
well-tested, the durability primitives are careful"), but C1 needs a design
answer — a detached source has no launcher process to acknowledge quiescence,
so the coordinator needs evidence it can observe itself.
