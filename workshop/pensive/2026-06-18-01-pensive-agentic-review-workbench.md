---
type: pensive
date: 2026-06-18
topic: agentic document-review workbench in pair
mode: eureka
description: parley's one-shot review is amnesiac; the fix is to stop making parley and Claude Code compete as harnesses — let pair's persistent agent be the conversational workbench and an embedded review nvim be the document workbench, with the agent proposing {old,occurrence,new,explain} records that the review pane applies undo-ably and commits (docflow).
references: [../parley.nvim/lua/parley/skills/review/init.lua, ../parley.nvim/lua/parley/skills/review/journal.lua, ../parley.nvim/lua/parley/skills/review/projection.lua, ../parley.nvim/lua/parley/skills/review/mode.lua, ../parley.nvim/lua/parley/tools/builtin/propose_edits.lua, ../ariadne/construct/skills/fix, ../42shots]
---

# Pensive: Agentic document-review workbench in pair

The thing that's been bugging me about parley's `review` is that it's **amnesiac**.
It's one-shot: parley assembles a prompt, forces a single `propose_edits` tool call,
and that's the whole loop. No transcript, no memory, no multi-step discovery. Compare
that to ariadne's `fix` skill running inside Claude Code — there's at least a stable
transcript as short-term memory, and because it lives in the repo it can pull in more
memory on demand (a pensive about testing, the relevant atlas, whatever the local
tools surface). The review tip is just the tip; there's a whole corpus that should be
within reach of it, and right now none of it is.

**The eureka is that I've been framing this as parley-the-harness vs. Claude-Code-the-harness,
as if they compete.** They don't. Claude Code is the better *compute* engine (loop, tools,
cross-repo memory). parley is the better *document* surface (in-editor 2D rendering of
state, accept/reject, where a knowledge worker already lives). Stop making them fight:
let one be the conversational workbench and the other the document workbench. And the
punchline — **pair already _is_ that structure**: agent pane on top, nvim on a persistent
draft below, one long-lived `claude --resume` session behind it. I didn't need to invent
a new architecture; I need to plug review into the one pair already has.

## The shape

- **(B) first: delegate the loop to pair's persistent session.** It already has the agentic
  loop, the tools, and native reach into brain / pensives / datatype skills. Building my own
  loop in lua is the productization move for *later*, not now — robust-and-quick beats
  owning-the-stack at this stage.
- **Embed review in pair as inline lua by extracting parley's _consumer half_.** Render,
  journal, projection, diagnosis, marker editing, modes — roughly half ports as-is; the
  whole LLM-invoke path (`run_via_invoke`, dispatcher, provider) is surgically isolated and
  gets dropped, because pair's agent is the producer now. pair's draft nvim loads no external
  plugins by design, but pair already spawns role-specific nvims (scrollback, changelog), so
  the review window is just one more — opened full-screen via `:PairReview <file>` / alt+r,
  alt+r to leave (never bare esc — it's nvim's most-pressed key).
- **The agent proposes, the review pane applies.** The pair agent emits
  `{old, occurrence, new, explain}` records (propose_edits + an occurrence disambiguator,
  so multiple-old_string is just input validation, never a re-anchor problem). The review
  pane applies them to the buffer as **undo-able** ops, drops an extmark that **rides**
  subsequent edits, renders `explain` as the diagnosis, and makes the commit. One record,
  four uses.
- **git is checkpoints, not history.** docflow commits are the round boundaries (pair is
  heavily repo-centric, already autosave-commits per round). But the *real* fine-grained
  history is nvim undo, which must stay continuous across commit boundaries and persist across
  sessions — ideally undo all the way back to the first review. That's the constraint that
  forces "the agent proposes records, parley applies" rather than "the agent writes the file,
  parley reloads": a reload resets the buffer's undo. Three layers, none collapsible — nvim
  undo (continuous reversibility), git commits (round checkpoints), and a review-history doc
  for the per-hunk explains git can't hold.
- **Styling accumulates.** The agent fires off many suggestions, so its highlights must
  *persist* — the human needs the standing visual cue of what's unreviewed. Human typing,
  alt+a (accept), alt+r (reject) all just ride; a human round adds its own styling but never
  clears the agent's. What clears the agent's styling is the **next conversation turn** —
  the agent's next response repaints — and optionally an explicit end-of-human-turn.
- **Modes come along.** Borrow parley's review modes (aggressiveness / how the LLM edits and
  explains); `mode.lua` + the mode briefs are already in the port bucket. fix has a simpler
  version; converge them.

Underneath all of it: this is a **vertical slice of the 42shots interface** — a document (the
tip) floating on a corpus (brain + the repos), an agent doing multi-step discovery to pull the
right slice of that corpus into reach, and a human steering by comment while staying in the
editor. "A word with a harness." The review tool is the small, buildable proof of the larger
human/AI knowledge-work thesis.

## Open questions

- **Cross-session undo: trust nvim's persistent `undofile`, or reconstruct from the
  review-history doc?** Need to read ariadne's **docflow** to see how it already treats
  fine-grained history vs. commit checkpoints before inventing a parallel mechanism.
- **Where do per-hunk explains actually live** — commit message body (line-anchored), git
  notes, or the history-doc sidecar? docflow may already have a convention; adopt it, don't fork.
- **Exactly when to clear agent styling** — next conversation turn vs. explicit end-of-human-turn.
- **The pair→agent poke channel** is `zellij ... write-chars` keystroke injection today. Fine
  for a text instruction, but it's not a clean RPC — is a better channel worth it?
- **Keep parley's in-process one-shot as a no-pair fallback producer of the same contract, or
  commit to single-path through the agent?** One contract (SKILL.md + record/commit format),
  possibly two producers — and that contract is the agnosticism guard: pair/Claude is one
  consumer, not a hard dependency.
- **Divergence:** extracting (copying) parley's review into pair forks the code. Acceptable for
  B-first; revisit a shared dependency-light module only if both keep evolving.
