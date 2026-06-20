---
type: target
slug: review-protocol
status: active
issue: 000066
created: 2026-06-19
---

# Review Workbench Protocol — the agent ↔ review-nvim state machine

The agentic review workbench is **two workbenches joined by a thin seam**: pair's
persistent agent is the *conversational + compute + git* surface; an embedded review
nvim is the *document* surface. They never share process state — they coordinate
through a small set of files plus the zellij poke channel. This target is the
invariant both sides must honor; drift on either side breaks the loop silently.

## Governing principle (confirmed 2026-06-19)

**The review nvim never writes git.** It renders the doc, applies the agent's edit
records *undo-ably*, captures the human's edits, saves, and **pokes** the agent. ALL
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
| 1 | open-state file `$PAIR_DATA_DIR/review-<tag>.open` | review nvim (pid on VimEnter; removed on VimLeave) | draft nvim (`PairReviewToggle` liveness; review-mode cue) | one line: the pane nvim's pid | **BUILT** — `review-toggle-test`, `review-window-test` |
| 2 | handoff file (agent → nvim) | agent | review nvim (`handoff.watch` poll) | `{old, occurrence, new, explain}[]` (`record.lua`; == agent commit body) | **BUILT** — `review-handoff-test`, `review-loop-test` |
| 3 | poke channel (nvim → agent) | review nvim (zellij `write-chars`, agent addressed by **absolute pane id**) | agent pane | NL instruction, carrying the **absolute** doc path | **BUILT** — `review-poke-test` (abs-path 2026-06-19) |
| 4 | git: `review/<slug>` branch + round commits | **AGENT** (`docflow`, in the doc's repo) | review nvim **reads** (reconstruct decorations + indicator counts) | `review(<slug>): <side> r<N> — …`, per-hunk explains in body | **read** BUILT (`review-loop-test`); **write** = **M4-DESIGN** |
| 5 | mode file `$PAIR_DATA_DIR/review-<tag>.mode` | **AGENT** (on a mode switch from either channel) | review nvim + draft bar (display the `🪄 <Mode>`) | one line: the active mode | **M4-DESIGN** |

## States & transitions

```
            Alt+r (no open-state)                handoff records arrive
   ┌─────┐ ──────────────────────► ┌───────────┐ ─────────────────────► ┌──────────┐
   │idle │   :PairReview <file>     │open /     │                        │applying  │
   │     │ ◄──────────────────────  │rendering  │ ◄───────────────────── │(nvim)    │
   └─────┘   VimLeave (close)       └───────────┘   render + save + poke  └────┬─────┘
                                      ▲   │ Alt+Return                         │ poke
                                      │   ▼ (save + poke "commit human round") │ "applied N"
                                      │ ┌───────────────┐                 ┌────▼─────────┐
                                      │ │human-editing  │                 │agent commits │ (M4)
                                      │ └───────────────┘                 │  agent round │
                                      │   the agent, asked, commits  ◄─────┴──────────────┘
                                      └─── + re-reviews (next handoff) ───┘
   ship: "ship it" → agent `docflow ship` (merge --no-ff + branch delete)            (M4)
```

- **idle** — no open-state file. Draft shows the normal pair-slug. `Alt+r` → file-select. **BUILT.**
- **open / rendering** — review nvim open on `<file>`; doc + 🤖 markers rendered; draft line-1 becomes the **review indicator** (slug generation suppressed). `Alt+r` ⇄ visibility. **BUILT** (indicator: M3-close item).
- **agent-proposing** *(M4)* — the SKILL recognizes "please review", does memory discovery, and on the **first** round creates `review/<slug>` **in the doc's repo** (the abs path from poke #3 tells it which repo), then writes the handoff records. This IS the **xx-fix-under-docflow flow** (see *What "review" means here* below) — not a review skill the agent picks by vibe.
- **applying** — review nvim polls the handoff → applies undo-ably → renders → **saves** → pokes "applied N edits to `<abs>`". **BUILT** (apply/render/save); the post-apply poke is the **commit signal**.
- **agent-committing** *(M4)* — the agent commits the agent round (records in body) **only after** the "applied" poke (apply can drop unanchorable records, so the agent must not blind-commit its own proposal). `agent-count++`.
- **human-editing** — the human edits in the review pane. **BUILT.**
- **human-finish** (`Alt+Return`) — review nvim **saves** → pokes "updated, please commit this human round + re-review `<abs>`". **BUILT** (save + poke); the commit is the agent's.
- **human-committing** *(M4)* — the agent commits the human round. `human-count++`.
- **ship** *(M4)* — "ship it" → the agent runs `docflow ship` (merge `--no-ff` + branch delete).

## What "review" means here (xx-fix, not doc-review)

The workbench's "review" is the agentic embedding of **ariadne's `xx-fix` skill under
`docflow`**: the agent proposes edits as `{old, occurrence, new, explain}` records (the
programmatic form of xx-fix's `🤖` marker edits), the pane applies them undo-ably, and
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
  markers, in the user's style. (The battle-tested core.)
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
- **review nvim** — a sticky mode menu (parley's UI) + an optional multi-line free-text
  box; Alt+Return → Return keeps the default. On confirm it pokes `"[free text], updated,
  go <mode>"` (the persisted-session analogue of parley's one-shot packaged prompt — same
  UI, but it injects a small poke instead of dispatching an invoke).
- **display** — the review bar's `🪄 <Mode>` segment (above).

> **Naming (deferred):** `xx-fix` has outlived its name — it's no longer "fix small things
> from `🤖[instruction]`", it's a collaborative writing assistant. Rename to
> `writing-assistant` eventually (an ariadne-side change; not now).

## Invariants to defend from drift

1. **The review nvim writes no git.** (If you find a `docflow round`/`ship`/branch call
   in `nvim/review/*`, it's drift — the agent owns writes.)
2. **Undo is continuous** (nvim `undofile`); never reload-to-refresh a buffer (a reload
   resets the undo tree — the reason records are applied in-buffer, not file-rewritten).
3. **The agent commits a round only after the nvim's "applied" poke** (apply may drop
   records; the committed body must match what actually landed).
4. **One review pane per session** (the open-state file is the singleton guard).
5. **Pokes carry the absolute doc path** (the agent's cwd is pair's, not the doc's repo).
6. **The review is the xx-fix-under-docflow record flow** (propose edits → apply → rounds),
   NOT `doc-review` (read-only fact-check) standing in for it. `doc-review` is an optional
   input, never the review. (See *What "review" means here*.)
