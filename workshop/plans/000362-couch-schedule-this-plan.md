# Cross-slot scheduling mechanisms (pair#362) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A coordinator slot can send work to another local slot and read where that
work stands, including whether its message actually reached the recipient. The read
uses one new read-only command, `couch --peek repo:N`, plus the existing SDLC
observations.

**Architecture:** Nothing new for sending or for work state. `couch --send-to` and
`--message-status` already send and receipt; `sdlc issue show N --json` (ariadne#279)
already answers ownership, checkpoints (milestone verdicts), completion and landing;
`sdlc help recovery` (ariadne#280) already answers retry semantics.

The one gap is runtime evidence: whether a peer message reached the recipient's
composer and whether the turn was submitted. Neither the receipt nor SDLC can answer
that. `couch --peek repo:N` fills the gap, read-only. It resolves the slot to its
thread, then returns:
- the last N lines of the slot's recorded terminal, rendered as plain text by Pair's
  existing scrollback renderer;
- the paths of the Pair sent-prompt log and the agent's native transcript, resolved
  by the existing `OSSwitchContextResolver`.

Everything is reported with explicit "unavailable" reasons; a failed read is never
reported as empty. The Couch skill then teaches the evidence ladder over these
commands.

**Tech Stack:** Go (`couchcore`, `couchcmd`, `scrollbackcmd`), the vendored `charmbracelet/x/vt`
emulator, `sessioninventory`, Markdown skill shipped in the couch binary.

---

## Context and decisions

- Spec: `workshop/issues/000362-couch-schedule-this.md` (narrowed 2026-10-06 to mechanisms;
  orchestration is pair#396, the smoke-test state is ariadne#297).
- **Reading another slot is allowed** (operator, 2026-10-06). Couch centralizes context, and
  every actor can read its own transcript, so a read-only look at a peer is acceptable. This
  deliberately departs from strict encapsulation.
- **Couch → sdlc only** (`feedback_layer_dependency_direction`). Peek is a couch command and
  never calls sdlc. sdlc never learns about couch. A joined couch+sdlc view is not built
  (it would live in couch if ever needed).
- **Agent-agnostic** (`feedback_pair_dogfood_and_agnostic`).
  - The terminal tail comes from Pair's own recording, which exists for every agent.
  - Native transcripts are returned as paths, never parsed: the reading agent knows
    its own transcript format, and couch must not learn N formats.
- **Out of scope** (recorded in the issue):
  - recording peer deliveries as scrollback events with a byte offset (the
    envelope header `[Couch peer from <slot>; delivery <id>]` is already echoed in
    the rendered text, so a reader can search for it);
  - persisting receipts past the wrapper;
  - a combined view;
  - a bench queue.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RenderLines` | `cmd/internal/scrollbackcmd/scrollbackcmd.go` | new (extracted from `render`) |
| `PeekResult` | `cmd/internal/couchcore/peek.go` | new |
| `peekTail` | `cmd/internal/couchcore/peek.go` | new |

- **RenderLines(rawPath, eventsPath, plain, maxLines) ([]string, []dayMark, error).** Replays a
  recorded `.raw` capture through the emulator: scrollback history then the visible screen,
  with trailing blanks trimmed. `render` keeps its file output, viewport sidecar and date
  markers, and becomes `RenderLines` plus writing.
  - **DRY rationale:** the renderer is the one place that knows how to replay a capture
    (resize segments, the emulator drain goroutine). Peek must not grow a second replay.
  - **Relationships:** one capture yields one line list. Callers are `render` (file) and
    `PeekSlot` (memory).
- **PeekResult.** The JSON-able answer:
  - `{slot, tag, agent, lines []string, sent_prompts, transcripts []string, unavailable []string}`;
  - `lines` is the rendered tail (the visible screen is its last rows, where the composer is);
  - `unavailable` names every evidence source that could not be read, and why.
  - **Future extensions:** a delivery-correlation field, if the exercises show that
    searching for the envelope header in `lines` is not enough.
- **peekTail(lines, n).** The last `n` lines; `n<=0` means the default of 40.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Couch.PeekSlot` | `cmd/internal/couchcore/peek.go` | new | thread store, `artifactpath`, storagegc lease, `OSSwitchContextResolver` |
| `peek` operation + `--peek` CLI | `cmd/internal/couchcore/ops.go`, `cmd/internal/couchcmd/{cli,run}.go` | new | operation dispatch, terminal output |
| Couch skill scheduling section | `cmd/internal/couchcmd/skills/couch/SKILL.md` | modified | agent instructions |

- **Couch.PeekSlot(ctx, ref, lines).** Steps:
  - Resolve `repo:N` the way `--show`/`--resume` do: `ParseWorkspaceReference`, then the
    thread record at that slot.
  - `artifactpath.Resolve{DataDir, RepoScope, Tag}.ScrollbackArtifacts(agent)` gives the
    live `.raw`/`.events.jsonl`.
  - Take the storagegc read lease, exactly as `pair scrollback render` does
    (`acquireRenderLease` via explicit owner).
  - Call `RenderLines(plain=true)`, then `peekTail`.
  - `OSSwitchContextResolver.Resolve(record)` gives `PairLog` and `NativeTranscripts`, and
    its `Unavailable` reasons pass through.
  - **Injected into:** the operation layer. Tests use a real temp data dir with a
    synthetic `.raw` capture (no mocks); the resolver's `Query`/`NativePath` seams already
    exist.
- **peek operation.** `ExecuteDirectStore` (read-only, like `show`), `Effect: none`,
  `--json` for agents, a plain renderer for humans. It must run from any slot, including
  inside a live Couch slot.

## Tasks

### Task 1: Extract `RenderLines` from the scrollback renderer

**Files:** Modify `cmd/internal/scrollbackcmd/scrollbackcmd.go`; Test: `cmd/internal/scrollbackcmd/scrollbackcmd_test.go`.

- [ ] Write a failing test, `TestRenderLinesMatchesTheRenderedFile`: for an existing
      fixture capture (reuse one from the package's tests), `RenderLines(plain)` joined with
      "\n" equals the plain file `render` writes.
- [ ] Extract `RenderLines` (everything in `render` up to the trailing-blank trim). `render`
      calls it, then keeps its timestamp interleave, viewport sidecar and atomic write.
- [ ] Run `go test ./cmd/internal/scrollbackcmd/` and `sh tests/scrollback-open-test.sh`: the
      existing tests pass byte-identically.
- [ ] Commit `#362: scrollback: RenderLines returns the rendered capture in memory`.

### Task 2: `PeekResult`, `peekTail`, `Couch.PeekSlot`

**Files:** Create `cmd/internal/couchcore/peek.go`, `cmd/internal/couchcore/peek_test.go`;
modify `cmd/internal/artifactpath/manifest.go` (inventory) only if the new file's paths need
it, and the dead-symbol allowlist only if required.

- [ ] Failing tests, using a real temp data dir, a thread record at `repo:1`, and a
      synthetic `.raw` capture:
  - `TestPeekShowsTheSlotsRecentTerminal`: a capture containing
    `[Couch peer from pair:0; delivery abc]` plus a composer line yields `lines`
    ending with them; `lines` is capped at the requested count.
  - `TestPeekNamesEveryUnreadableSource`:
    - a missing `.raw` reports `unavailable: terminal recording … (reason)` with an
      empty `lines`, never a silent empty;
    - an unknown slot, or `repo:0` with no thread, is refused with the candidates;
    - an unresolved native transcript passes the resolver's reason through.
  - `TestPeekReturnsTranscriptPaths`: through the resolver's `Query`/`NativePath`
    seams, `transcripts` holds the resolved path and `sent_prompts` the Pair log.
  - `TestPeekIsReadOnly`: a snapshot of the data dir and the thread store before and
    after is byte-identical, apart from the lease file the renderer already manages.
- [ ] Implement `peekTail`, `PeekResult` and `Couch.PeekSlot` as in Core concepts. Reuse
      `ParseWorkspaceReference` and the existing slot → thread lookup (the one
      `--show repo:N` and `--resume repo:N` use); write no new resolver.
- [ ] `go test ./cmd/internal/couchcore -run Peek`, then commit
      `#362: couch: peek a slot's recent terminal and transcript paths`.

### Task 3: The `peek` operation and `couch --peek repo:N [--lines N] [--json]`

**Files:** Modify `cmd/internal/couchcore/ops.go`, `operationdispatch.go`,
`cmd/internal/couchcmd/cli.go`, `run.go`, the usage text; Test: `cmd/internal/couchcmd/peek_cli_test.go`.

- [ ] Failing tests:
  - CLI parsing: `--peek pair:1`, `--peek pair:1 --lines 80 --json`; refusals for a
    missing ref, a bad `--lines`, and a layout flag.
  - Rendering: the plain form prints a header (`slot pair:1  agent claude  tag …`),
    the lines, then `transcript: <path>` / `sent prompts: <path>` / `unavailable: …`;
    `--json` prints `PeekResult`.
  - The operation audit tests (operation names, CLI dispatch identity) pass with the
    new operation declared.
- [ ] Implement, matching the `show` operation's shape (read-only, direct store). Place new
      renderers *above* any existing doc comment (lessons: stacked godoc).
- [ ] Commit `#362: couch: --peek repo:N`.

### Task 4: The scheduling section of the Couch skill

**Files:** Modify `cmd/internal/couchcmd/skills/couch/SKILL.md`, README couch CLI section,
`atlas/couch.md` (messaging/peek), `atlas/index.md` if a file is added.

- [ ] Add `couch --peek pair:1` to the command table.
- [ ] Add a `## Scheduling work on another slot` section:
  - **Dispatch.** The issue exists and is published in its repository; send
    "work on repo#N" to an exact slot or a family; do not restate SDLC rules (claim
    enforces claim-before-work and the owner check).
  - **The evidence ladder**, one command per rung:

    | Rung | Evidence | Command |
    |------|----------|---------|
    | accepted | receipt `queued`/`delivering` | `couch --message-status ID` |
    | submitted | receipt `submitted`; envelope header in the composer or tail | `couch --message-status ID`, `couch --peek repo:N` |
    | claimed | `assignment` owner = the recipient slot | `sdlc issue show N --json` |
    | progressing | branch commits, plan ticks, `checkpoints` (milestone verdicts) | `sdlc issue show N --json` |
    | complete | `completion` | `sdlc issue show N --json` |
    | landed | `landing` | `sdlc issue show N --json` |

    Dirty files and a working card are activity, not progress.
  - **Looking again.** Revisit after an interval (about 30 seconds is an example, not a
    deadline). Absence is not proof of loss.
  - **Deciding.**
    - Wait while the rungs advance.
    - Follow up at the same exact slot when the message is visible but the agent
      did not act.
    - Retry only per the operation's recovery contract (`sdlc help recovery`); never
      resend blindly to another slot after an uncertain submission.
    - An unknown read never authorizes a takeover; reassignment is the operator's
      `sdlc reclaim`.
  - **Receiving a duplicate.** If `sdlc claim` refuses because another workspace owns
    the issue, report the owner to the sender and do not start.
  - **Peek is read-only.** Never type into another slot's terminal; messages go through
    `--send-to`.
- [ ] The skill vocabulary and README-sweep tests pass (`TestSkillDocuments*`, couchcmd
      messages tests).
- [ ] Commit `#362: couch skill: scheduling and the evidence ladder`.

### Task 5: Verification and the live exercises (operator smoke test)

- [ ] Full suite, unsandboxed, with the session environment scrubbed: `make -k test`, the
      scratchpad-TMPDIR `make test-changelog`, `go test ./...`. Compare with main's known
      failures.
- [ ] Live exercises, run by the operator, or by this slot driving two scratch slots
      with the operator's OK:
  1. Two slots receive the same "work on pair#N" for a throwaway issue. Exactly one claim
     wins; the other reports the owner. Observed with `sdlc issue show N --json`.
  2. A message is sent while the recipient is busy (composer occupied). `--peek` shows the
     envelope header waiting or absent; `--message-status` explains it; the coordinator
     waits rather than resending, and later sees it submitted.
  3. With the recipient agent idle or stopped, `--peek` and `sdlc issue show` still answer;
     an unavailable read is reported as unavailable.
- [ ] Record each exercise's commands and outcomes in the issue `## Log`. These are the
      Done-when evidence.

## Plan quality notes

- **ARCH-DRY.** One renderer (`RenderLines`), one transcript resolver
  (`OSSwitchContextResolver`), one slot → thread lookup.
- **ARCH-PURE.** `peekTail`/`PeekResult` are pure, and `RenderLines` is a replay with no
  side effects.
- **ARCH-CONSTRAINTS.** Peek replays the whole live capture, which is truncated at
  every wrapper start. Measure the time on the largest live capture during Task 5.
  If it exceeds about 1 s, cap the replay to the tail of the file (resize events
  allow restarting there) and log it.
- **Dependency direction.** Peek touches only couch and Pair data; no sdlc call.

## Revisions

### 2026-10-06 (a) — the peek latency is in resolving `repo:N`, not in rendering

Measured on the live `ariadne:1` (a 1.5 MB recording):
- `couch --peek ariadne:1` takes 2.3 s.
- `RenderLines` on the same capture takes 85 ms; `couch --list` takes 0.23 s.
- `couch --show ariadne:1` takes 3.7 s.

So the cost is in resolving the slot reference (`resolveSlotInput`: workspace
resolution plus `Slots.Discover`), which every `repo:N` operation pays. The CLI path
pays it twice: once in `runTypedOperationWithConsole`, to derive the repository
scope, and again in `ResolveThreadReference`. The replay itself meets the ~1 s
budget, so no tail cap is needed.

At about 2 s, peek is adequate for a coordinator that looks again about every 30 s.
Speeding up `repo:N` resolution (resolve once per call, and cache Discover)
benefits `--show`, `--resume` and `--reboot` as well, so it is a follow-up issue,
not #362 scope.


### 2026-10-06 (b) — "busy" for delivery means an occupied composer

The operator asked when an actor counts as busy, and the code answers it in two
places:
- **Family routing** (`--send-to repo`) skips a slot unless it is on its resting
  branch, has been quiet for 30 seconds, has a free mailbox and has allowance left.
- **Delivery to an addressed slot** does not observe model execution
  (`wrapcmd/peer_composer.go`). It waits only for a recognized, empty composer,
  paste mode, and 1 second since the last keystroke. A working agent's composer is
  empty, so the message submits and the agent queues it behind its current turn.
  A message still waiting after `DeliveryTimeout` (30 seconds) expires.

Task 5 exercise 2 therefore becomes: a draft sits in the recipient's composer. Peek
shows the draft and no envelope, and the receipt says "waiting for empty composer".
Clearing the draft within 30 seconds lets the message submit; leaving it lets the
message expire undelivered, which is safe to resend.

The skill now states that `submitted` means queued, not acted on, and that an
expired or not-dispatched message is safe to resend (526b1a15). The full suite's
only new failure was the operation-declaration table missing `peek`, which is now
fixed; every other failure matches main.

### 2026-10-06 (c) — Core concepts as built (close-review finding)

The close review found that the Core concepts table no longer matched the code.
Recorded here; the table above is left as the plan of record.

| Name | Lives in | Status | As built |
|------|----------|--------|----------|
| `RenderLines(rawPath, eventsPath, maxLines) ([]string, error)` | `scrollbackcmd/scrollbackcmd.go` | new | Plain lines only; `render` uses an internal `replay` for its marks and viewport. **IO, not PURE:** it reads two files and parks one emulator drain goroutine per call for the life of the process. |
| `RenderOwnedLines(dataDir, scope, tag, agent, maxLines)` | `scrollbackcmd/retention.go` | new (unplanned) | Integration point: finds the owner's live capture under `repos/<scope>` and reads it under the retention lease. |
| `SlotTerminalReader` / `Couch.SlotTerminal` | `couchcore/peek.go`, `couch.go` | new (unplanned) | The injected seam `PeekThread` reads the recording through; production wires it to `RenderOwnedLines` in `couchcmd/run.go`. |
| `Couch.PeekThread(ctx, ref, address, lines)` | `couchcore/peek.go` | new | Planned as `PeekSlot(ctx, ref, lines)`. The `repo:N` lookup is the existing `resolveOperationThread` in the dispatcher, not inside peek. |
| `PeekResult`, `peekTail` | `couchcore/peek.go` | new | As planned, plus `ref` and `working_path`. |

Tests that changed shape:
- `TestPeekReturnsTranscriptPaths` through the resolver's `Query`/`NativePath` seams
  was not written. `TestPeekShowsTheSlotsRecentTerminal` covers the paths through a
  whole-resolver fake, and the production resolver's own seams are covered by its
  existing switch-agent tests.
- `TestPeekIsReadOnly` snapshots the thread store. The data directory is touched only
  through `RenderOwnedLines`, whose lease is the existing renderer's.

Minors taken in the same commit:
- a `--json` encode failure is now printed;
- the local `context` variable no longer shadows the package;
- the `peek` declaration records that it is a CLI operation, because each replay
  parks a goroutine, so the long-running console must not dispatch it;
- the atlas paragraph is re-wrapped.

Left as noted:
- no end-to-end `RunWithRuntime --peek --json` test; the live exercises covered
  that path;
- no direct test of the dispatcher's `--lines` refusal.
