---
type: target
slug: review-protocol
status: active
issue: 000066
created: 2026-06-19
updated: 2026-10-18
---

# Review Workbench Protocol — the agent ↔ review-nvim state machine

The agentic review workbench is **two workbenches joined by a thin seam**: pair's
persistent agent is the *conversational + compute + git* surface; an embedded review
nvim is the *document* surface. They never share process state — they coordinate
through a small set of files plus the zellij poke channel. This target is the
invariant both sides must honor; drift on either side breaks the loop silently.

## Governing principle (confirmed 2026-06-19)

**The review nvim never writes git.** It renders the doc, reloads and decorates the
rounds the agent applies through `pair review apply`, captures the human's edits,
saves them, and **pokes** the agent. It never writes the reviewed file for an agent
round. ALL
git — the `review/<slug>` branch, the round commits, and `ship` — is the **agent's**,
driven by prose pokes (the agent is "asked", it acts).

Why this split (not nvim-shells-docflow, which M1 scaffolded):
- The agent is the producer/compute surface (the issue's B-first Spec decision). Git
  is compute; it belongs there, not in a thin nvim UI (`ARCH-PURE`).
- The agent runs in a real shell that resolves `docflow`; the review pane's minimal
  `nvim -u review.lua` env does not (this is the M3-smoke ENOENT class — killed for
  good by moving the calls out).
- One `docflow` caller in one environment, taught once (the M4 SKILL) — not two.

## The seam (files + channel)

| # | seam | writer | reader | payload | status |
|---|------|--------|--------|---------|--------|
| 1 | open-state file `$PAIR_DATA_DIR/review-<tag>.open` | review nvim (owning incarnation publishes/removes) | draft nvim (`PairReviewToggle` liveness; review-mode cue) | line 1 PID; line 2 absolute file; line 3 JSON `{version:1, endpoint, token, session, context}` for same-pane RPC | **BUILT** — `review-toggle-test`, `review-window-test` |
| 2 | `pair review apply EDITS FILE --if-hash H [--json]` + round artifact `$PAIR_REVIEW_ROUND_PATH` (pair#661 M2) | agent, through the shared engine (pair writes the file; nvim never does) | review nvim (`round_watch` poll: reload + decorate); agent recovers a lost reply with `pair review round FILE --json` | records `{old,occurrence,new,explain}` all-or-nothing; exits 0 / 1 `edits-invalid`·`propose-*`·`operator-owned` / 3 `operator-lock` (the pane holds `FILE.lock` during the operator's turn) / 6 `stale`; artifact `{file, round, body}`, `body` = the commit body (round as review-records); **the empty round** (pair#690, producer side ariadne#453): the producer always applies, with records `[]` when it has no edits, and the engine writes no file, only an artifact whose round has `base_hash` = `result_hash`, no spans and an empty `body`, so the agent commits nothing; a round is identified by `result_hash` plus `at`, since an empty round keeps the previous hash; a reply `Warning` = written but no artifact (commit `Body` anyway); the lock records `pid:` and a dead owner's lock is reclaimed | **BUILT** — `review-applied-round-test`, `reviewapply` Go tests, `agent_round_rule_test` (nvim never writes the buffer/file for an agent round). The legacy handoff file and landed artifact (old seams 2/2b) were retired by pair#664 |
| 3 | poke channel (nvim → agent) | review nvim (zellij `write-chars`, agent addressed by **absolute pane id**) | agent pane | NL instruction, carrying the **absolute** doc path and exact activation `context` for human/agent/ship effects | **BUILT** — `review-poke-test` (abs-path 2026-06-19) |
| 4 | git: `review/<slug>` branch + round commits | **AGENT** (`docflow`, in the doc's repo) | review nvim **reads** (reconstruct decorations + indicator counts) | `review(<slug>): <side> r<N> — …`, per-hunk explains in body | **read** BUILT; **write** proven via real `pair review apply` rounds (`review-applied-round-test`), producer instructions = shared `xx-fix` (ariadne#121, scoped effects ariadne#268) |
| 5 | mode file `$PAIR_DATA_DIR/review-<tag>.mode` | **AGENT** (on a mode switch from either channel) | review nvim + draft bar (display the `🪄 <Mode>`) | one line: the active mode | **BUILT (pair side, M4c)** — `seam_test`, `review-indicator-test`, `review-window-test` |
| 6 | review-target `$PAIR_REVIEW_TARGET_PATH` | explicit preparation; draft after activation acknowledgment | draft (session selection receipt/cache, never branch authority) | ready/proposed target with conversation scope; explicit preparation records `{repo,branch,file,head}` identity | **BUILT** — `review-toggle-test`, `review-branch-restore-test` |
| 7 | definition request/result `$PAIR_REVIEW_DEFINITION_REQUEST_PATH` / `$PAIR_REVIEW_DEFINITION_RESULT_PATH` | pane / `pair review definition` | producer / pane | request keeps stripped text in `.context` and activation identity in `.review_context`; result echoes it in `.context`, retaining request ID | **BUILT** — definition CLI/window tests |

Session scope resolves from inherited `PAIR_SESSION_ID`, then the shared
inventory's established owner projection. It does not inspect config files,
processes or native rollout files. A fresh conversation may restore committed
branch history explicitly with Alt+c; it does not adopt another conversation's
selection receipt or transient requests.

### Branch restoration and activation context (#341)

Alt+c resolves the current Pair checkout's review branch before target-cache or
pane-visibility decisions. Exact current-slug round subjects and all their
changed paths must identify one tracked regular in-repository document. Missing,
ambiguous, deleted, unsafe, detached or failed Git observations refuse without
cache/pane mutation. No filename is guessed from the slug. A non-review branch
prompts for explicit selection, except for a verified current-session explicit
selection in a different repository; a current review branch always wins over
that peer selection. Before the first path-bearing round only, a
current-session explicit preparation receipt matching repository, branch, file
and prepared HEAD authorizes opening; committed history wins once present. An
authenticated matching live pane can keep its established selection across an
empty human round that advances HEAD without establishing a file path.

A matching pane toggles visibility. A different clean idle document activates
inside the same Neovim process through private RPC, retaining buffers and undo.
Retained clean content is compared with disk and refreshed as one undoable edit
before decoration reconstruction, including different branches of the same file.
Modified buffers, an awaiting agent request and pending definition requests block
retargeting. A live old pane
without the RPC capability refuses safely; it is never killed to replace it.
The draft publishes its target only after a validated activation acknowledgment.
Uncertain RPC outcomes are probed in that same pane; they never authorize a
replacement process. Activation alone sends no review request or Git write.

The wire context is `{repo,branch,file,activation}`: canonical absolute repository,
repository-relative document, exact branch and opaque activation token. HEAD is
only a resolver observation. Scope is checked before consuming a response and
before save/send/ship. The pane acts on a round artifact only for its own file and
only while the file on disk still hashes to the round's result. Definition
responses also retain their request-ID check.

The cooperative producer echoes request context unchanged and checks the live
open metadata plus repository/branch/document immediately before human-round,
agent-round or ship Git effects. The round body from `pair review apply` is
committed verbatim. A mismatch preserves artifacts
and asks the operator to return to the original branch and finish the round or
reissue from the intended activation. The authoritative instructions are the
shared `xx-fix` skill, updated through ariadne#268; arbitrary independent Git
commands remain outside the pane's authority.

If the checkout changes beneath an active pane, saves and review effects suspend.
On exit, unsaved text is preserved in private `review-recovery/` storage beside
the open-state file, keyed by repository/branch/document. Return to the original
branch and use `:PairReviewRecover` to restore it undo-ably, or explicitly discard
it with `:PairReviewDiscardRecovery`. A successful matching save removes only a
snapshot written or restored by that buffer/incarnation. Prior-process snapshots
are never silently overwritten. Storage admits at most 32 entries and 8 MiB per
snapshot; exhaustion refuses preservation and reports the need to recover or
discard existing data rather than evicting it.

## States & transitions

All transitions below are subject to the branch/activation guards above.

```
            Alt+c (resolved identity)             round artifact appears
   ┌─────┐ ──────────────────────► ┌───────────┐ ─────────────────────► ┌──────────┐
   │idle │   :PairReview <file>     │open /     │                        │reloading │
   │     │ ◄──────────────────────  │rendering  │ ◄───────────────────── │(nvim)    │
   └─────┘   VimLeave (close)       └───────────┘   :edit! + decorate     └──────────┘
                                      ▲   │ Alt+Return                         ▲
                                      │   ▼ (save + poke "commit human round") │ pair review apply
                                      │ ┌───────────────┐                 ┌────┴─────────┐
                                      │ │human-editing  │                 │agent applies │
                                      │ └───────────────┘                 │ + commits    │
                                      │   the agent, asked, commits  ────►│ agent round  │
                                      └─── + re-reviews ──────────────────┴──────────────┘
   ship: "ship it" / `:PairReviewShip` → agent `docflow ship` (merge --no-ff + branch delete) (M4)
```

- **idle** — no open-state file. `Alt+c` resolves and restores the current review branch, or prompts for selection off a review branch. **BUILT.**
- **open / rendering** — review nvim open on `<file>`; doc + 🤖 markers rendered; the draft statusline carries the **review indicator**. `Alt+c` ⇄ visibility. **BUILT** (indicator: M3-close item). In review nvim, `Alt+a` accepts, `Alt+r` rejects, and `Alt+q` inserts `🤖[]` or wraps the visual selection as `🤖<selection>[]`. The context poke defaults the agent to **Copy Edit** posture and tells it to resolve `🤖[]` human comments as edits when possible, or punt explicitly when not.
- **agent-proposing** *(M4)* — the SKILL recognizes "please review", uses the prepared review branch and request context, then runs `pair review apply` with its records. This IS the **xx-fix-under-docflow flow** (see *What "review" means here* below) — not a review skill the agent picks by vibe.
- **applying** — `pair review apply` writes the file and the round artifact; review nvim polls the artifact → reloads (`:edit!`) → decorates the spans → clears the awaiting spinner. It writes nothing. **BUILT.** An empty round (no edits) writes only the artifact; the pane reloads nothing and just clears the spinner and the restore pending block, unsaved operator text or not. **BUILT** (pair#690). While the operator holds `FILE.lock` (mid-turn), the apply exits 3 and the agent waits.
- **agent-committing** *(M4)* — the agent commits the agent round with the `Body` from the apply reply (records in body, exactly what landed). `agent-count++`. An empty `Body` (the empty round) commits nothing.
- **human-editing** — the human edits in the review pane. **BUILT.**
- **human-finish** (`Alt+Return`) — review nvim **saves** → pokes "updated, please commit this human round + re-review `<abs>`" in Copy Edit posture, with `🤖[]` comments handled as fulfill-or-punt instructions. **BUILT** (save + poke); the commit is the agent's.
- **human-committing** *(M4)* — the agent commits the human round. `human-count++`.
- **ship** *(M4)* — "ship it" or `:PairReviewShip` → the agent runs `docflow ship` (merge `--no-ff` + branch delete). The review nvim only pokes; it never shells `docflow ship`.

## Concurrent edits — the operator lock (pair#661 M2)

The human may keep editing while the agent produces a round. The pane takes
`FILE.lock` on the first edit of the operator's turn and drops it on send and at
exit; `pair review apply` exits 3 while it is held, so the agent applies only
between turns. A round the pane declines (unsaved operator edits) is warned about
once and redelivered later. #89's in-nvim reconcile engine (`🤖<…>[reconcile — …]`
markers) and apply gate were retired by pair#664; such markers left in old documents
are ordinary `🤖[…]` requests.

Durability: matching-context saves run on exit. A changed checkout blocks the save
and preserves unsaved text in bounded recovery storage instead. Pending work cannot
authorize a different activation; finish it on the original branch or explicitly
reissue it.

## What "review" means here (xx-fix, not doc-review)

The workbench's "review" is the agentic embedding of **ariadne's `xx-fix` skill under
`docflow`**: the agent proposes edits as `{old, occurrence, new, explain}` records (the
programmatic form of xx-fix's `🤖` marker edits), `pair review apply` applies them and the pane reloads and decorates, and
`docflow` commits each round on `review/<slug>`. The round-commit counts in the
indicator ARE those `docflow` rounds.

This is distinct from the **`doc-review` binary** (the `fresh-context-review` skill): a
**read-only** second-vendor agent that fact-checks a doc's claims + references and writes
`<file>-<agent>-check.md`. It **cannot edit the doc** and makes **no** rounds. It is an
*optional input* to the fix flow (xx-fix can dispatch it, then apply the findings as
edits), **never the review itself**.

> **M3-smoke gotcha (the motivating bug for M4):** poked the bare "please review", the
> M3 dumb agent saw a blog post with external claims and ran `doc-review` (fact-check) —
> reasonable in isolation, wrong for the workbench: it edited nothing and made no rounds,
> so the pane/indicator saw no activity. **The M4 SKILL's whole job is to bind "please
> review (from the workbench)" → the xx-fix-under-docflow record flow**, optionally
> running `doc-review` as a fact-check step first. Invariant #6.

## Review-mode bar (draft statusline) — BUILT (M3), mode segment M4

While a review is open, the draft's **statusline** carries the review state (the line-1
`=== review … ===` indicator was wrong — line 1 is the user's to edit; superseded). The
review segment **replaces the rightmost cheatsheet**; the timer-cached counts mean the
hot statusline render never shells git. Counts are **scoped to the active `review/<slug>`
branch's own rounds** — `🤖0/0` off a review branch (M3 render-only), so a repo's history
of *other* docs' shipped reviews never leaks in (the "25/28" bug). Tested: `review-indicator-test`.

Target format (lean — "remove all help text", `-`=history `+`=future):
```
-92 < -3 > +0 • 🪄 Copy Edit • <file> • 🤖N/M       (M4: 🪄 <Mode> from the mode state)
-92 < -3 > +0 • Review • <file> • 🤖N/M             (M3: no mode state yet)
```
The left `-h < pos > +q` is the lean prompt-history position (history total / current /
queue total). 🤖N = agent (robot) rounds, /M = human.

## Modes, voice, switching — M4

**Three editing postures** (mutually exclusive — the active "how the agent edits") + one
orthogonal pass. Form = the ported `mode.lua` (`modes/<name>.md` → `mode.directives()`);
described in the SKILL up front; the agent tracks the active mode as session state.

- **Generate** *(was "brainstorm")* — human supplies a sketch / skeleton / bullets; the
  agent develops the doc, composing in the user's voice. Still goes through records — in
  the limit a single `old` skeleton line → a large `new` block (rarely a blank page).
- **Copy Edit** — user authored most of it; agent makes limited edits + resolves `🤖[]`
  markers, in the user's style. Agent-authored copy-edit changes are **minimal inline
  marker proposals**: prefer word/phrase/sentence anchors, use `🤖<old text>{new text}`
  for replacements, `🤖{new text}` for insertions, and deletion markers for removals.
  Do not replace a whole paragraph to change a few words. (The battle-tested core.)
- **Proofread** — syntax + spelling only (mechanical).
- **Fact-check** — NOT a peer mode; an **orthogonal pass**, free-text-triggered ("do a
  fact check on this"). Dispatches the read-only `doc-review` agent (world knowledge + web
  + repo state); it changes nothing; the main agent integrates the note as edits through
  the record protocol, in whatever posture is active.

### Voice — `voice: <slug>` in the doc frontmatter
Any pass whose doc has a `voice: <slug>` line loads `~/.personal/<slug>-writing-style.md`
(per-doc: blog ≠ book ≠ company; repo/project default as fallback). Generate + Copy Edit
honor it; Proofread + Fact-check are voice-neutral. Loading the voice is part of the skill.

### Switching + display — one source of truth
The mode lives in the **seam** (a `review-<tag>.mode`, agent-written) so both switch
channels and the bar read the same value.
- **draft window** — free text ("now do a copy edit"; fact-check is also just free text,
  keeping the current mode).
- **review nvim** — `Alt+Return` sends the human turn immediately with the current
  mode. The send menu has no keybinding for now; invoke it explicitly with
  `:lua PairReviewPane.open_mode_menu("/absolute/path/to/document.md")` in review
  nvim (parley's UI shape: mode list plus optional multi-line instruction box).
  On confirm it finishes the human turn with the
  selected mode and optional instruction for that single round only.
- **display** — the review bar's `🪄 <Mode>` segment (above).

> **Naming (deferred):** `xx-fix` has outlived its name — it's no longer "fix small things
> from `🤖[instruction]`", it's a collaborative writing assistant. Rename to
> `writing-assistant` eventually (an ariadne-side change; not now).

## Review-start & resume flow — M4a'

`:PairReview` does **not** open the pane — it *proposes* a review target (seam #6); the
local deterministic readiness prepares it; Alt+c opens once `ready` (**manual** — auto-open's
async timing is bad UX). This is the agentic embedding of "is this doc ready to review?"

**Readiness prep (pair-side) — the 4 git cases.** A deterministic function of git
state; pair computes it and performs deterministic setup locally (draft nvim shells
to `pair-review-readiness --prepare`; ARCH-PURE: pure classification, shell seam for
git effects):
- not git-managed → **stop**, ask the operator to create a repo (don't auto-init).
- git-managed, untracked → **track** the file, then start/resume its review branch.
- on a `review/<slug>` branch whose scoped file == the target → **resume** (single file per
  review branch — multi-file is out of scope).
- not on a review branch: clean → **new** `review/<slug>`; dirty → **interact** (the only
  truly interactive case — clean up / choose with the operator).

**Alt+c restoration.** Resolve committed review-branch identity first, then toggle
or activate the matching document. Explicit `:PairReview` preparation supplies the
session receipt for the zero-round exception. A proposed target reports preparation
in progress; stale ready targets cannot override the current branch.

**Resume.** Reconstruct only the selected current-slug/document rounds. A fresh
session restores committed context, while in-process activation retains the buffer
and undo tree. Neither path inherits another document's decorations or requests.

**Agent-running spinner (≤6 cols).** The pane derives "agent working" from the **protocol
state** (no `pair-wrap` flag): set when it pokes the agent, cleared when the next applied
round lands (`after_applied_round`). Braille spinner + compact elapsed: `⠹ 45s` → `⠹ 2m`.

## Invariants to defend from drift

1. **The review pane nvim writes no git.** — **BUILT (pair side, M4a).** `nvim/review/init.lua`
   calls `docflow` nowhere: `on_applied_round` only reloads and decorates (an empty round only ends the turn);
   `human_round` only saves; `review.start` no longer runs `docflow.start`. Draft nvim may
   perform deterministic start-up git (`pair-review-readiness --prepare`: track/start/resume),
   but round commits and ship remain outside the pane. Verified headlessly by
   `review-applied-round-test` (real `pair review apply` rounds) plus readiness CLI/toggle tests for prep.
   A `docflow round`/`ship` call in `nvim/review/*` is now drift.
2. **nvim never writes the reviewed buffer or file for an agent round.** pair writes the
   file; the pane reloads (`:edit!`) and decorates. nvim writes it only for operator
   actions (save on send, `VimLeave` save, marker accept/reject, comment threads,
   definition results, `:PairReviewRecover`). **BUILT** — `agent_round_rule_test`
   (`on_lines`, `BufWritePre` and Lua write spies, positive control on the operator send).
   Undo across operator edits stays continuous via `undofile`.
3. **The agent commits exactly what landed**: the `Body` from the `pair review apply`
   reply (or `pair review round FILE --json` after a lost reply), never its own proposal.
4. **One review pane per session** (the open-state file is the singleton guard).
5. **Pokes carry the absolute doc path and activation context** (the agent's cwd may differ). Both producer Git effects and pane application validate that identity.
6. **The review is the xx-fix-under-docflow record flow** (propose edits → apply → rounds),
   NOT `doc-review` (read-only fact-check) standing in for it. `doc-review` is an optional
   input, never the review. (See *What "review" means here*.)

## Revisions

2026-09-20 — #297 retires the review send-menu shortcut. The menu remains
available through an explicit Neovim command; no replacement keybinding is assigned.

2026-09-28 — #341 makes the current review branch authoritative for Alt+c, adds
same-pane activation and refusal for pending work, scopes all late responses and
producer Git effects, and preserves unsaved text under checkout mismatch in bounded
recovery storage. Corrected obsolete target/session fallback, pane replacement and
unconditional exit-save descriptions; ariadne#268 owns shared producer guidance.

2026-09-28 — #341 integration: retain explicitly selected peer documents when the draft checkout is non-review, and authenticated live selection across empty first rounds. Retained inactive buffers keep ownership and cannot write into a different checkout.

2026-09-28 — Exit failure semantics: ordinary non-bang quit retains modified buffers when snapshot storage fails. Explicit `:qa!` may ignore callback errors and discard unsaved text on failed storage; it cannot be promised recoverable. Successful snapshot writes are tested across forced exit and restart.

2026-10-18 — pair#664: seams 2 (handoff file) and 2b (landed artifact) collapsed into
one seam 2, `pair review apply` + the round artifact; nvim's buffer apply, the
concurrent-edit reconcile and the apply gate are retired, replaced by the operator
lock. Invariant 2 changes from "apply in-buffer, never reload" to "nvim never writes
the reviewed buffer or file for an agent round", now tested by
`agent_round_rule_test`; invariant 3 now names the apply reply's `Body`.

2026-10-19 — pair#690 (pairs ariadne#453, which makes the producer always apply):
seam 2 gains the empty round. A zero-change apply writes no file, only a round
artifact with no spans whose result is its base and whose `body` is empty, and the
agent commits nothing. Rounds are identified by `result_hash` plus `at`. On an empty
round the pane ends the agent's turn (spinner and restore pending block) without
reloading.
