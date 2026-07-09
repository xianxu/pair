# Boundary Review — pair#113 (whole-issue close)

| field | value |
|-------|-------|
| issue | 113 — review footnotes should not join last question |
| repo | pair |
| issue file | workshop/issues/000113-review-footnote-question-boundary.md |
| boundary | whole-issue close |
| milestone | — |
| window | 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5..HEAD |
| command | sdlc close --issue 113 |
| reviewer | codex |
| timestamp | 2026-07-08T22:53:45-07:00 |
| verdict | REWORK |

## Review

Reading additional input from stdin...
OpenAI Codex v0.143.0
--------
workdir: /Users/xianxu/workspace/pair
model: gpt-5.5
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR, /tmp] (network access enabled)
reasoning effort: none
reasoning summaries: none
session id: 019f456e-f532-7110-89ec-c44794d6f1de
--------
user
# Code review — the one SDLC boundary review

You are conducting a fresh-context code review at a development boundary —
whole-issue close — in the **pair** repository.

- repository: pair   (root: /Users/xianxu/workspace/pair)
- issue:      pair#113   (file: workshop/issues/000113-review-footnote-question-boundary.md)
- window:     Base: 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5   Head: HEAD

Review the **pair** repo and its tracker — the ariadne base-layer repo itself (changes here propagate to dependent repos). Do not assume any
other repository or apply another repo's conventions.

You have no prior session context — that is the anti-collusion property. Verify
behavior against the issue's documented Spec/Plan and the code itself; do NOT
take the implementor's word in commit messages or docs at face value. Tools are
read-only: report findings precisely; the main agent (which has session context)
applies the fixes, commits, and re-runs.

Read the diff against the issue's Spec + Plan, then work the checklist below.
Categorize every finding by severity — not everything is Critical; a nitpick
marked Critical is noise.

  Critical (must fix before crossing the boundary)
    - correctness bugs; crashes / panics on unexpected input
    - behavior drift from stated contracts (for ports of existing code where
      byte-faithfulness was promised, diff against the source)
    - silent error swallowing where the source raised
  Important (fix before the boundary if cheap)
    - API design of newly-introduced internal packages (downstream work will
      consume them; is the surface stable?)
    - missing test coverage that would catch the kind of bug shipped
    - inconsistent error handling across the diff
  Minor (note for future)
    - style nits, naming, comment density; performance only if hot-path

## Review checklist

Code quality
  - Clean separation of concerns; edge cases handled (empty / nil / unexpected).
  - Proper error handling — no silent swallowing where the source raised.
  - No duplicated logic / copy-paste that should be a shared helper.

Testing
  - Tests pin real logic, not mocks reasserting the implementation.
  - The kind of bug this diff could ship is covered.
  - PURE entities tested without IO; INTEGRATION via injected fakes (see below).

Requirements traceability
  - Every Plan checklist item this boundary claims is actually delivered.
  - Implementation matches the Spec; no undeclared scope creep.
  - Breaking changes documented.

Production readiness
  - Migration / backward-compatibility considered where state or formats change.
  - Docs / atlas updated for new surface (see the Docs update gate).

## Core concepts cross-check (if the plan has a Core concepts table)

The plan should list entities in a greppable table — name, kind
(PURE/INTEGRATION), file location, status (new/modified/deleted). For each row:
  - Verify the entity exists at the stated path (grep the diff or filesystem).
  - PURE: tests run without IO (no exec, net, mutable fs). If tests need mocks
    to run, it isn't really PURE — flag Critical and recommend promoting it to
    INTEGRATION.
  - INTEGRATION: injected into pure callers, not invoked directly from business
    logic.
  - "modified" / "deleted": the diff shows the expected change/removal at the
    stated location.
Any contradiction between table and code = Critical finding, plus a plan-revision
recommendation (a "## Revisions" entry so the plan stops claiming what the code
doesn't deliver).

## Docs update gate (atlas + README, per AGENTS.md §8)

The boundary should update user-facing docs for any new surface introduced:

  - **atlas/** — new architectural surface, flow, or terminology. Scan the diff
    for new entity types, subcommands, conventions, file-tree locations. Any
    present without corresponding atlas/ changes in the same range = Important
    finding ("atlas update appears missing for <surface>").
  - **README.md** — new user-facing surface a reader runs or types: subcommands,
    flags, keybindings, config keys, install/usage steps. If the diff adds or
    changes such surface and README.md is not updated in the same range =
    Important finding ("README update appears missing for <surface>"). This is the
    class of gap that used to surface only at the merge-time `specs` judge (#142);
    catch it here, at the earliest gate, before the close verdict is recorded.

## Architecture (the at-review backstop — these matter most long-term)

Work through each of ARCH-DRY, ARCH-PURE, ARCH-PURPOSE explicitly, applying its at-review lens. The
full principle definitions are delivered in the ARCHITECTURE PRINCIPLES block
right after this prompt — for EACH marker, state pass or flag, and cite the
marker (e.g. ARCH-DRY) in any finding. Architecture is where review has the
least training signal and the longest-delayed payoff, so be deliberate here, not
holistic.

## Verdict + output

Begin your response with this fenced verdict block — the machine-read handoff:

```verdict
verdict: <SHIP | FIX-THEN-SHIP | REWORK>
confidence: <high | medium | low>
```

  SHIP           ready; ship it
  FIX-THEN-SHIP  ship after addressing the findings (non-blocking at the gate)
  REWORK         blocking; needs rework before shipping — fix + re-run

The fenced ```` ```verdict ```` block above is the **authoritative machine-read
handoff** — emit it as the first thing in your response. (A prose
`VERDICT: <TOKEN>` first line still satisfies the legacy contract as a fallback,
but the block is what the binary trusts.)

After the verdict block: a 1-paragraph summary — what worked, what blocks SHIP if
it isn't — followed by:
  1. Strengths: 2-5 specific things done well (file:line where useful). Affirm
     validated approaches so the operator knows what's confirmed-good ground.
     Empty acceptable for trivial boundaries.
  2. Critical findings (file:line + fix sketch); empty if none.
  3. Important findings (same format).
  4. Minor findings (terse one-liners).
  5. Test coverage notes.
  6. Architectural notes for upcoming work.
  7. Plan revision recommendations: specific "## Revisions" entries the plan
     needs (empty if the plan still matches the code).


ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-review` lens; cite the marker (e.g. ARCH-DRY) in any finding.

# Architecture principles (ARCH-*)

Injected architectural taste — the structural decisions whose payoff (or cost)
shows up many turns, often months, down the road. Agents are strong at local
tactics and weak here, so these are checked **at-plan** (when the design is being
made — highest leverage) and **at-review** (backstop, on the diff). Cite the
marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.

This file is the single source; it is embedded into the planning, plan-quality,
and code-review prompts. The human narrative lives in AGENTS.md "Core Design
Principles"; this is its machine-delivered companion.

## ARCH-DRY — Don't Repeat Yourself

- **principle:** Reuse before adding. One source of truth per fact/behavior; no
  duplicated logic, copy-pasted blocks, or parallel functions that should be one
  shared helper.
- **at-plan:** Flag a plan that re-implements something the codebase already has,
  or that will obviously duplicate logic across the new files instead of
  extracting a shared helper. Name the existing thing it should reuse.
- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
  functions in the diff; point at the consolidation (file:line + the shared
  helper they should become).

## ARCH-PURE — Pure core, thin IO shell

- **principle:** The majority of code is pure functions (deterministic, no side
  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
  functions are unit-tested directly; the glue is kept small and injected.
- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
  that will only be testable with heavy mocks (a sign logic isn't separated from
  IO). The plan should name what's pure vs the thin IO seam.
- **at-review:** Flag business logic mixed with IO in the diff; logic that should
  be a pure function injected into a thin caller. If a test needs mocks to run a
  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.

## ARCH-PURPOSE — Serve the issue's actual purpose

- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
  single-source / "compiled to consumers" change is not done until **every
  consumer derives** from the source — the source is *enforced*, not just
  documentation a surface happens to restate; a hand-maintained restatement of the
  model is a deferred consumer, not a finished one. "Follow-up" is for separable
  extensions, never for the thing that is the point. This is the *opposite axis*
  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
  **under**-deliver the purpose you already committed to."
- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
  wires one consumer + enforcement but leaves the consumers that motivated the
  issue as documentation that doesn't derive). Ask: does the plan fulfill the
  purpose, or just the cheap win? Name the deferred purpose.
- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
  confirm each derives from the source, flag any remaining hand-maintained
  restatement of the model. A "follow-up" that is actually the deferred point of
  the issue is a finding, not a deferral.


OUTPUT CONTRACT (machine-read — do not deviate). LEAD your response with the
fenced ```verdict block shown above — that is the authoritative handoff the binary
reads (its `verdict:` value is one of the listed tokens). Everything after the block
is advisory: a non-blocking verdict WITH findings still PASSES the gate. A bare
`VERDICT: <TOKEN>` line is accepted only as a FALLBACK when the block is absent.

Diff:


codex
I’ll use `superpowers-using-superpowers` for the required skill-selection protocol and `sdlc` because this is explicitly an SDLC boundary review. Then I’ll inspect the issue, plan, and diff window read-only.
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/adapted/superpowers-using-superpowers/SKILL.md" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "sed -n '1,220p' /Users/xianxu/workspace/ariadne/construct/local/sdlc/SKILL.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
name: sdlc
description: Use when at an SDLC checkpoint — starting work, closing an issue or milestone, opening/merging a PR, or recovering workflow state after compaction. The `sdlc` binary owns the gates between workflow stages and refuses transitions that lack required evidence.
---

# sdlc — SDLC checkpoint binary

`sdlc` owns the gates between SDLC workflow stages (claim → change-code → pr →
merge, plus close, milestone-close, judge). It requires evidence at each gate,
mutates state, logs the transition, and refuses transitions that lack the
evidence — that is the shape of a "checkpoint guard."

The binary is the single source of truth. This skill is a static pointer and
intentionally carries no copy of the contract, so it can never drift:

- **`sdlc --help`** — the workflow contract: the start-of-work runbook,
  conventions, and the verb list.
- **`sdlc <verb> --help`** — one checkpoint's full contract, flags, and examples.

Read those instead of relying on memory; the binary's help is always current.

 succeeded in 0ms:
---
name: superpowers-using-superpowers
description: Use when starting any conversation - establishes how to find and use skills, requiring Skill tool invocation before ANY response including clarifying questions
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task, skip this skill.
</SUBAGENT-STOP>

<EXTREMELY-IMPORTANT>
If you think there is even a 1% chance a skill might apply to what you are doing, you ABSOLUTELY MUST invoke the skill.

IF A SKILL APPLIES TO YOUR TASK, YOU DO NOT HAVE A CHOICE. YOU MUST USE IT.

This is not negotiable. This is not optional. You cannot rationalize your way out of this.
</EXTREMELY-IMPORTANT>

## Instruction Priority

> **Ariadne note:** AGENTS.md Section 3 governs subagent strategy and overrides skills that mandate subagent-driven-development as the default execution path.

Superpowers skills override default system prompt behavior, but **user instructions always take precedence**:

1. **User's explicit instructions** (CLAUDE.md, GEMINI.md, AGENTS.md, direct requests) — highest priority
2. **Superpowers skills** — override default system behavior where they conflict
3. **Default system prompt** — lowest priority

If CLAUDE.md, GEMINI.md, or AGENTS.md says "don't use TDD" and a skill says "always use TDD," follow the user's instructions. The user is in control.

## How to Access Skills

**In Claude Code:** Use the `Skill` tool. When you invoke a skill, its content is loaded and presented to you—follow it directly. Never use the Read tool on skill files.

**In Gemini CLI:** Skills activate via the `activate_skill` tool. Gemini loads skill metadata at session start and activates the full content on demand.

**In other environments:** Check your platform's documentation for how skills are loaded.

## Platform Adaptation

Skills use Claude Code tool names. Non-CC platforms: see `references/codex-tools.md` (Codex) for tool equivalents. Gemini CLI users get the tool mapping loaded automatically via GEMINI.md.

# Using Skills

## The Rule

**Invoke relevant or requested skills BEFORE any response or action.** Even a 1% chance a skill might apply means that you should invoke the skill to check. If an invoked skill turns out to be wrong for the situation, you don't need to use it.

```dot
digraph skill_flow {
    "User message received" [shape=doublecircle];
    "About to EnterPlanMode?" [shape=doublecircle];
    "Already brainstormed?" [shape=diamond];
    "Invoke brainstorming skill" [shape=box];
    "Might any skill apply?" [shape=diamond];
    "Invoke Skill tool" [shape=box];
    "Announce: 'Using [skill] to [purpose]'" [shape=box];
    "Has checklist?" [shape=diamond];
    "Create TodoWrite todo per item" [shape=box];
    "Follow skill exactly" [shape=box];
    "Respond (including clarifications)" [shape=doublecircle];

    "About to EnterPlanMode?" -> "Already brainstormed?";
    "Already brainstormed?" -> "Invoke brainstorming skill" [label="no"];
    "Already brainstormed?" -> "Might any skill apply?" [label="yes"];
    "Invoke brainstorming skill" -> "Might any skill apply?";

    "User message received" -> "Might any skill apply?";
    "Might any skill apply?" -> "Invoke Skill tool" [label="yes, even 1%"];
    "Might any skill apply?" -> "Respond (including clarifications)" [label="definitely not"];
    "Invoke Skill tool" -> "Announce: 'Using [skill] to [purpose]'";
    "Announce: 'Using [skill] to [purpose]'" -> "Has checklist?";
    "Has checklist?" -> "Create TodoWrite todo per item" [label="yes"];
    "Has checklist?" -> "Follow skill exactly" [label="no"];
    "Create TodoWrite todo per item" -> "Follow skill exactly";
}
```

## Red Flags

These thoughts mean STOP—you're rationalizing:

| Thought | Reality |
|---------|---------|
| "This is just a simple question" | Questions are tasks. Check for skills. |
| "I need more context first" | Skill check comes BEFORE clarifying questions. |
| "Let me explore the codebase first" | Skills tell you HOW to explore. Check first. |
| "I can check git/files quickly" | Files lack conversation context. Check for skills. |
| "Let me gather information first" | Skills tell you HOW to gather information. |
| "This doesn't need a formal skill" | If a skill exists, use it. |
| "I remember this skill" | Skills evolve. Read current version. |
| "This doesn't count as a task" | Action = task. Check for skills. |
| "The skill is overkill" | Simple things become complex. Use it. |
| "I'll just do this one thing first" | Check BEFORE doing anything. |
| "This feels productive" | Undisciplined action wastes time. Skills prevent this. |
| "I know what that means" | Knowing the concept ≠ using the skill. Invoke it. |

## Skill Priority

When multiple skills could apply, use this order:

1. **Process skills first** (brainstorming, debugging) - these determine HOW to approach the task
2. **Implementation skills second** (frontend-design, mcp-builder) - these guide execution

"Let's build X" → brainstorming first, then implementation skills.
"Fix this bug" → debugging first, then domain-specific skills.

## Skill Types

**Rigid** (TDD, debugging): Follow exactly. Don't adapt away discipline.

**Flexible** (patterns): Adapt principles to context.

The skill itself tells you which.

## User Instructions

Instructions say WHAT, not HOW. "Add X" or "Fix Y" doesn't mean skip workflows.

exec
/bin/zsh -lc 'sdlc --help' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
sdlc collects ariadne's SDLC checkpoint guards into one binary. Each subcommand
owns one checkpoint: it requires evidence at the gate, mutates state, logs the
transition, and refuses transitions that lack it. We don't model the SDLC as a
state machine — stages stay prose; we codify the gates between them where drift
recurs. `sdlc` manages the development life cycle; prefer it over `git`/`gh`.

BEFORE WORK
  - `sdlc claim --issue N` — the single start-of-work gesture, a CHEAP LOCK.
    Flips an *open* issue to `working` and publishes the claim to origin/main so
    peer agents see it. No estimate demanded (#113) — claim early, the moment an
    idea crystallizes. `--no-start` suppresses the flip.
  - Do NOT hand-edit an issue's `status:` — let `sdlc claim` or `sdlc issue
    set-status` own that transition (it carries the reopen/`→ done` guards).

ENTER IMPLEMENTATION
  - After plan approval, before editing code, run `sdlc change-code`. It owns the
    branching decision (in-place branch by default; `--worktree=yes` for an
    isolated worktree), the plan-quality check, and the `estimate_hours` gate
    (relocated here from claim, #113). Don't start coding without it.

PUBLISH
  - Publishing goes through a PR: `sdlc pr` → `sdlc merge`. Direct `sdlc push`
    if working directly on main.
  - Publish ONCE at issue close, not per milestone — and do NOT reuse a branch
    name that already has a merged PR. `sdlc merge` refuses (#148) when a branch
    has commits not in main despite a merged PR (a reused name would otherwise
    silently strand the new commits); rename to a fresh branch, `sdlc pr`, retry.

RECOVER
  - After a compaction or session resume, run `sdlc state` to recover where you
    are instead of re-inferring from issue files.

LOCAL REPO TRANSACTION LOCK
  - Mutating verbs take an SDLC-owned repo transaction lock at
    `.git/sdlc.lock` before reading/writing issue state, committing, changing
    branches, or pushing. The lock is local to the Git common dir, so linked
    worktrees of the same repo serialize with each other.
  - Wait messages identify the holder pid and command when metadata is
    available. `close` and `milestone-close` release the lock while the external
    boundary-review subprocess runs, then reacquire before finalization; if HEAD
    or the issue/project file state they prepared changed meanwhile, they refuse
    to finalize and tell you to rerun. `change-code`, `merge`, and `push` can still hold the lock during
    long-running review/ship transactions; wait or retry rather than removing
    the lock while that process is alive.
  - A dead same-host holder is reclaimed automatically; initializing metadata
    is waited through. Other stale/timeout errors tell you how to inspect
    `.git/sdlc.lock`. Remote push/ref races are separate: the local lock
    serializes this checkout, not another machine or clone.

WHEN A VERB ERRORS
  Do NOT route around it with hand-rolled `git`/`gh`. Its errors are next-action
  specs. The fix is one of two things:
    (a) satisfy the precondition it names and re-run the same verb (e.g. `sdlc
        merge` saying "no upstream" → run `sdlc pr` first, then `sdlc merge`); or
    (b) if the error is a genuine gap in `sdlc` itself, fix that edge case in the
        source and re-run. We're still ironing out edge cases.
  Only drop to manual when a verb genuinely cannot express the need — say so.

These gates sit inside a wider prose arc the binary does NOT own: ideation
(parley/pensive) → brainstorm → plan → build → milestone review (`sdlc judge`,
auto-dispatched) → close/ship → postmortem.

CONVENTIONS

  --issue vs --github-issue — `--issue N` always means workshop/issues
  (6-digit ID). `--github-issue N` means a GitHub issue number. Bare `--issue`
  never means a GitHub issue.

  Form vs essence — checkpoint guards (close, milestone-close, push, merge)
  defend against *omission* via required-evidence flags; `sdlc judge` defends
  against *theater* via fresh-context review. Form runs first; judge second.

The verb list + per-verb help (`sdlc <verb> --help`) follow below.

Usage:
  sdlc [flags]
  sdlc [command]

Available Commands:
  claim           Start work: flip an open issue to working + broadcast the claim
  start-plan      Enter planning: deliver the architecture principles to design against (#75)
  change-code     Enter implementation after the structural + plan-quality gates
  issue           Create + manage issues (new / set-status / list / show)
  actual          Compute an issue's focused dev-hours via active-time-v3 (#68)
  active-time     Per-issue active-time attribution table (the v3 engine, standalone)
  close           Close an issue or milestone (ACTUAL + VERIFIED + atlas/project sweep)
  milestone-close Close one milestone + auto-dispatch its review
  pr              Open a pull request from a feature branch
  merge           Merge the PR, archive done issues, clean up
  push            Ship from main (clean tree + pre-merge judges + archive)
  state           Inspect workflow state (branch, working issues, drift)
  resolve         Resolve a symbolic artifact ref (ariadne#11, #15 M4) to its current path(s) — read-only
  open            Resolve a ref and open the primary artifact in $EDITOR
  judge           Run an LLM-judge check against the diff (fresh-context)
  arch-principles Print the ARCH-* architecture principles (single source; pull for non-gate work)
  estimate-source Name the shared estimate method + the repo-local calibration source (pull)
  process-manual  Unroll every injection source into a linked process manual (#153)
  propagate-base  Re-weave every recursive dependent of this repo (foundation-first)
  help            Help about any command

Flags:
  -h, --help   help for sdlc

Use "sdlc [command] --help" for more information about a command.

exec
/bin/zsh -lc "sed -n '1,220p' workshop/lessons.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
# Lessons

## Async buffer requests need live anchors, not saved coordinates

Pair review definitions originally stored the selected line/column range while
the agent produced an answer. If the user inserted text before the selected term
before the result arrived, the response applied to stale coordinates and inserted
the footnote reference into the wrong text.

**Rule.** Any Neovim request that crosses an async boundary and later mutates the
same buffer must anchor the target with an extmark (or re-locate/validate the
target from content) before applying the result. Raw row/column pairs are only a
snapshot. Add an integration regression that mutates text before the target while
the request is pending, then verifies the result follows the target or aborts
cleanly. Caught in #000112 close review.

## Generated review sidecars must stay bounded

`sdlc close` writes a review sidecar, and that sidecar becomes part of later
diffs. If it stores the full raw prompt/transcript, it can bloat the reviewed
diff and carry whitespace-sensitive embedded patches.

**Rule.** Keep committed review sidecars to the durable facts: verdict, window,
findings, verification, and resolution. Avoid committing full prompt/diff
transcripts unless the generator normalizes them and they remain small enough
for future review prompts.

Caught while closing #000108.

## Path precedence contracts need explicit divergent-env tests

#90's embedded runtime implementation documented extraction under
`$PAIR_DATA_DIR/runtime/<digest>/pair-home`, but the first OS-backed
implementation only used the XDG/home resolver. The copied-binary smoke unset
`PAIR_DATA_DIR`, so the bug survived until boundary review tried
`PAIR_DATA_DIR` and `XDG_DATA_HOME` with different roots.

**Rule.** When a feature promises environment-variable precedence, add a test
where the higher-priority and fallback variables are both set to different
directories, then assert the selected path. Also include every Go source file
that can change build output in Make prerequisites; a generated or embedded
artifact path should have a dependency test or an explicit review checklist
entry. Caught in #000090 boundary review.

## Lua patterns: `\0` is empty-position match, not NUL byte

The unescape function in `nvim/scrollback.lua` first attempt used a
placeholder dance: `s:gsub('\\\\', '\0')` to swap `\\` for NUL,
then `gsub('\\(.)', '%1')` to strip remaining `\X`, then
`gsub('\0', '\\')` to restore the NUL → `\`. The result was
absurd: `unescape("plain")` returned `\p\l\a\i\n\` — the NUL pattern
matches between every byte (empty-position match), not the NUL byte
character. Each "match" inserted a `\` between every char.

**Rule.** When you need to match a literal NUL byte in a Lua pattern,
use `%z` or wrap as a character class `[%z]`. But the cleaner answer
is usually to skip patterns entirely for character-by-character
walks: a tiny while-loop with `s:sub(i, i)` is unambiguous and avoids
all the pattern-syntax footguns. Caught in #000018 review.

## Escape on insert, scan-with-parity on extract — for delimited markers

When user-supplied content is embedded in a delimited container
(e.g. `🤖<X>[Y]`), and X or Y can contain the delimiter chars,
the choice is "escape at insert + unescape at extract" vs "find
the closing delimiter cleverly." The first attempt at `🤖<X>[Y]`
parsing tried the latter — find first `>`, peek for `[`, give up
otherwise. Result: any selection with `>` was silently dropped on
extract, since the user couldn't tell the marker had been written
malformed.

**Rule.** Escape the delimiter chars in user-supplied fields at
insert time; have the parser walk byte-by-byte counting backslash
parity to find the *next unescaped* delimiter; unescape the
extracted content. The escape→walk→unescape chain handles every
delimiter-collision case uniformly, including `\\>` (literal `\`
followed by `>`). Don't try to be clever with "find first `>[`
adjacent" patterns — they fail when X contains `>[` literally,
and the failure mode is silent data loss. Caught in #000018 review.

## Shared delimiter codecs beat subsystem-local marker parsing

M4b's review pane added `Alt+q` visual wrapping as `🤖<selection>[]` but initially
embedded the selected text raw, even though annotate already had delimiter escaping for
the same marker family. A selection containing `>` or `]` could truncate the parsed marker
and make accept/reject leave stray syntax in the document.

**Rule.** When a second feature writes the same delimited marker format, reuse or extract
the existing codec before adding parser/writer code. Add tests for delimiter collisions
(`>`, `]`, backslash) at the write path and the consume path. A parser unit test alone is
not enough; the UI wrapper that inserts the marker must also be covered. Caught in #000066
M4b review.

## Shell scripts should use JSON builders, not `printf` JSON

`pair-review-readiness` originally printed JSON with `printf` and unescaped string fields.
A review branch named `review/a"b` produced invalid JSON, even though all the boolean
fields were correct.

**Rule.** In shell seams that emit JSON, use `jq -n --arg/--argjson` (or an equivalent
structured encoder) for every field. Do not hand-build JSON with `printf` unless every
string field is impossible by construction — and then document why. Guard it with a test
using quotes in a branch/path/name. Caught in #000066 M4b review.

## `#table` is 0 on string-keyed tables — never use it for ID generation

Adding nvim/scrollback.lua's hl-group cache: `local name = 'PairScrollback_' .. (#hl_cache + 1)` was meant to give each new (state→hl-group) entry a unique numeric suffix. `hl_cache` is a string-keyed dict (cache key is `(fg|bg|attrs)`), and Lua's `#` on a non-array table returns 0. Result: every group resolved to `PairScrollback_1`, `nvim_set_hl(0, "PairScrollback_1", def)` overwrote on each call, and all extmarks ended up sharing whatever the last-written attrs were. Caught only by an end-to-end test that checked extmark hl_groups against expected fg/bg ints.

**Rule.** When you need monotonic IDs in Lua, use an explicit counter (`local counter = 0; ... counter = counter + 1`). Do not use `#table` unless `table` is provably array-shaped (`{[1]=..., [2]=..., ...}`). The bug is silent — `nvim_set_hl` doesn't error on overwrite, it just wins-last. Filed during #000017 M4.

## Empty fields in delimited parsing — `[^;]+` drops them; semantics may differ

ECMA-48 SGR semantics: an omitted field is `0` (reset). So `\x1b[;1m` = "reset; bold". The first SGR parser pass used `params:gmatch('[^;]+')`, which silently skips empty fields — `[;1m` produced just `1` (bold), and any standing fg/bg/decoration leaked through. Caught in code review of #000017 (no real input from pair-scrollback-render's output would have triggered it, but it's a correctness footgun for any future caller pointing the viewer at non-pair-rendered ANSI).

**Rule.** When the protocol says "empty field has meaning," parse with `([^;]*);` on a `string + ';'` so the trailing-delimiter trick yields every field including empties. Generally true for any delimiter-separated format where omission has semantic value (CSV with empty cells, env-var lists, SGR, etc.).

## Sparse data structures: iterate by index, not by `.keys()`, when count must be exact

pyte's `screen.buffer` is a `StaticDefaultDict` — accessing `buffer[y][x]` lazily creates a default Char, but `buffer.keys()` only contains rows that were *written to*. The renderer originally did `for y in sorted(screen.buffer.keys())`, which silently dropped trailing blank rows when the agent cleared and paused mid-redraw. That shifts every subsequent line number — directly breaking the feature's core promise that `:880` lands where zellij showed line 880. Caught in code review of #000017.

**Rule.** When iterating over a sparse-by-design structure where every slot has a logical existence (even if unwritten), use `range(0, total)` and let the structure's `__getitem__` materialize defaults. `.keys()` is only correct when "absent" really means "doesn't exist." Same shape applies to anything with lazy materialization: defaultdicts, JS Maps with default fallbacks, sparse arrays.

## Atomic write for files a feature can race on its own

`bin/pair-scrollback-render` initially opened `<out.ansi>` with `'w'` (truncate-then-write). Two `Alt+/` presses in quick succession would race on the same path; whichever finished second left a half-interleaved file for nvim to open. Fixed by writing to `<out.ansi>.tmp` and `os.replace()`-ing at the end.

**Rule.** Any output file that a user-triggered keybind (or any concurrently-fireable mechanism) writes to should use the tempfile + atomic rename pattern. The cost is one extra file path; the gain is that readers see only "old complete file" or "new complete file," never "torn file." Apply uniformly even when a race is unlikely — discipline reduces the cognitive load for future readers.

## Verify zellij action and flag names against the installed version

Two bugs in v1 of `bin/pair` and `zellij/config.kdl` came from going off memory of zellij's API:

- Used `TogglePaneFullscreen` for the Alt+u bind. The actual action name in zellij 0.44.1 is `ToggleFocusFullscreen`. Caught by `zellij setup --check --config-dir <pair>/zellij`.
- Used `--layout PATH --session NAME` to "create a new named session with this layout." Zellij's actual semantic: when `--session` is set, `--layout` means "add as tab to that session" and errors if the session doesn't exist. The right flag is `--new-session-with-layout` (`-n`).

**Rule.** Before writing zellij KDL or invoking the zellij CLI:

1. Run `zellij setup --dump-config` to see the canonical action names used in default keybinds.
2. Run `zellij --help`, `zellij attach --help`, `zellij setup --help` against the installed version, and read the flag descriptions in full — they have non-obvious conditional semantics.
3. Always validate config and layout files with `zellij setup --check --config-dir <dir>` and `zellij setup --dump-layout <path>` before committing.

The verification tools are cheap and authoritative. Memory of "I think it's called X" is not.

## Stage content edits before `git mv` when closing an issue

Closing an issue means (a) editing the file (`status: done`, plan checkboxes), then (b) moving it to `workshop/history/`. Done in that order with `Edit` then `git mv`, the rename gets staged but the unstaged content edits do *not* — they stay in the working tree. `make issue-sync` only stages `workshop/issues/`, so the edits silently miss the commit. End state: history file with stale `status: working`.

**Rule.** When closing an issue:
1. Edit the file in place under `workshop/issues/` and `git add` it (or use `git add -u` after editing).
2. Then `git mv` to `workshop/history/` — git carries the staged content into the rename.
3. Or simpler: `git mv` first, edit second, `git add` the new path.

After running `make issue-sync` on a close, verify with `git show HEAD:workshop/history/<file> | grep status:` that the committed file actually has `status: done`. Don't trust the rename alone.

## On cancel, restore the prior visible state

When a confirmation prompt or interactive flow is dismissed, the cancel path must put the UI back exactly how it was — not just "do nothing." Issuing a prompt via `nvim_echo`/`getchar` (or any flow that paints over a region: cmdline, statusline, floating windows, virtual text, highlights) leaves that region in the prompt's state. The proceed branch usually triggers a redraw incidentally (state changes → statusline refresh → cmdline cleared). The cancel branch does not, so the prompt residue lingers until the next user input.

**Rule.** For every interactive surface, the cancel path is responsible for the same restoration the proceed path gets for free:

- Prompts that overdraw the cmdline/statusline → call the same redraw/refresh helper the success path calls (e.g. `refresh_statusline()`), not just `return`.
- Operations that mutated buffer text/cursor/window before asking for confirmation → snapshot first, restore on cancel.
- Highlights, virtual text, floating windows added as part of the flow → tear them down on cancel just like on success.

Treat cancel as an active branch with cleanup duties, not an early return. If you find yourself writing `if ch == 'n' then return end`, ask: what did the proceed branch do that I'm now skipping, and is any of it visual cleanup that cancel also needs?

## Transcript summarization must bias toward USER turns, not a flat tail

`cmd/pair-slug` (#000027) summarized "what is this session about" by feeding
the last N text-bearing transcript turns to a small model. On a tool-heavy
session that window is almost entirely assistant narration: a real Claude
transcript had ~16 genuine user prompts vs ~200 assistant entries (most
`user` entries carry only `tool_result` blocks, correctly dropped as
text-less). Measured: the last 10 text-bearing turns were 10/10 assistant,
0 user. So the slug tracked what the agent was *saying*, not what the user
*asked for* — the orientation signal was pushed out of the window. The unit
tests passed because their fixtures used only text-content messages, never
the dominant `tool_result`-only user shape — green tests masked the bug.
Caught in #000027 M1 review.

**Rule.** When sampling a conversation transcript to infer user intent:
- Don't take a flat tail of turns. Guarantee a minimum number of recent
  *user* turns are in the window (extend backward until satisfied, capped).
- Model test fixtures on the *real* transcript shape, including
  `tool_result`-only user entries and any sidechain/summary types — not the
  clean text-only case. A fixture that can't reproduce the bug can't guard
  against it.

## `gofmt -w <dir>` reformats files you didn't touch

Running `gofmt -w cmd/pair-wrap/` to format M3's edited `main.go` also
rewrote four pre-existing `*_test.go` files (struct-field alignment) that the
milestone never touched, staging unrelated churn into the commit. Caught at
`git status` review before commit; reverted with `git checkout -- <files>`.

**Rule.** Format only the files the change actually touches: `gofmt -w
path/to/file.go` (or `gofmt -w $(git diff --name-only '*.go')`), not the whole
package directory. If a dir-wide gofmt lights up files outside the change,
revert them — don't smuggle repo-wide reformatting into a feature commit.
Caught in #000027 M3.

## Dogfooding a Go-binary change needs `make install`, not just `make build`

M3's pair-wrap trigger "didn't fire" on restart. Trace: pair-slug worked in
isolation, but the running `pair-wrap` (pid via `pair-wrap-pid-<tag>`, binary
via `lsof -p <pid> | awk '$4=="txt"`) was `~/.local/bin/pair-wrap` dated days
earlier — the *installed* copy, with no spawn. I had only `go build -o bin/…`;
the layout (`zellij/layouts/main.kdl`) execs `pair-wrap` by bare name and the
pane's PATH resolved `~/.local/bin` first.

**Rule.** `bin/` is the repo build; `~/.local/bin` (via `make install`) is what
actually runs in a live pair session. To dogfood a change to a Go binary
(pair-wrap/pair-slug/…): `make install`, *then* restart pair. Verifying with
`bin/<binary>` alone proves nothing about the running session. When a "live"
change seems inert, confirm the running binary: `lsof -p $(cat
$PAIR_DATA_DIR/pair-wrap-pid-<tag>) | awk '$4=="txt"{print $NF}'`. Caught in
#000027 M3 dogfood.

## Queue items: resolve by filename key, not display index, across a mutation


codex
I have the workflow contract and local lessons loaded. Next I’m reading the tracker and the diff range so I can verify the implementation against the issue, not against commit claims.
exec
/bin/zsh -lc "sed -n '1,260p' workshop/issues/000113-review-footnote-question-boundary.md" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
---
id: 000113
status: working
deps: []
github_issue:
created: 2026-07-08
updated: 2026-07-08
estimate_hours: 0.81
started: 2026-07-08T22:41:44-07:00
---

# review footnotes should not join last question

## Problem

When the review pane sends a final `💬:` question at the end of a document that
is followed by a Markdown footnote block, the question extraction includes the
footnote definitions. The agent response is then inserted after the footnotes
instead of above them. This makes durable review definitions interfere with the
ordinary question/answer workflow.

## Spec

For review question submission:

- A line beginning with a Markdown footnote definition, matching
  `^%[%^[^%]]+%]:`, starts a footnote section.
- The final question body stops before the first such footnote-definition line.
- Agent answers for that question are inserted above the managed footnote
  footer. If a `---` divider immediately precedes the footnote definitions,
  keep it with the footer instead of splitting the footer.
- Existing behavior is preserved when no footnote-definition line follows the
  final question.

ARCH-PURPOSE: definitions are durable document metadata; they must not become
part of the user's last question or push the answer below the metadata.
ARCH-DRY/ARCH-PURE: put the boundary predicate in the pure review/question
helper that already computes the last question range, rather than duplicating
footnote detection in the UI shell.

## Done when

- A regression reproduces `💬:` followed by `---` and `[^acos]: ...`, proving
  only the question text is submitted.
- A regression proves the answer insertion point is above the managed footnote
  footer.
- Existing review question tests still pass.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.20 impl=0.40
item: milestone-review design=0.00 impl=0.15
total: 0.81
```

## Plan

- [x] Find the pure/helper code that extracts the final review question and
      chooses the answer insertion point.
- [x] Add a failing regression for a final `💬:` followed by Markdown footnotes.
- [x] Teach the range/insertion calculation to treat a leading `[^...]:`
      footnote definition as the trailing metadata boundary.
- [x] Run focused review/question tests and close #113.

## Log

### 2026-07-08
- Created from reported bug: final review question followed by definition
  footnotes gets submitted together with the footnotes, and the answer is
  inserted below the footnote block.
- Root cause is in peer `../parley.nvim`: `chat_parser.parse_chat` finalized a
  trailing open `💬:` question at EOF, so the exchange model counted the
  managed footnote footer as part of the question and inserted the answer after
  it.
- Implemented in `../parley.nvim/lua/parley/chat_parser.lua`: a final open
  question now treats a trailing column-1 `[^...]:` footnote block as metadata,
  and keeps an immediately preceding `---` divider with that footer.
- Added regressions in `../parley.nvim/tests/unit/parse_chat_spec.lua` for
  submitted question content and model insertion point.
- Verified with `nvim --headless --noplugin -u tests/minimal_init.vim -c
  "PlenaryBustedFile tests/unit/parse_chat_spec.lua"`,
  `nvim --headless --noplugin -u tests/minimal_init.vim -c
  "PlenaryBustedFile tests/unit/build_messages_spec.lua"`, and `make test` in
  `../parley.nvim`.

exec
/bin/zsh -lc 'sdlc arch-principles' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
ARCHITECTURE PRINCIPLES — work through each of the 3 entries below explicitly, applying its `at-plan` lens; cite the marker (e.g. ARCH-DRY) in any finding.

# Architecture principles (ARCH-*)

Injected architectural taste — the structural decisions whose payoff (or cost)
shows up many turns, often months, down the road. Agents are strong at local
tactics and weak here, so these are checked **at-plan** (when the design is being
made — highest leverage) and **at-review** (backstop, on the diff). Cite the
marker (e.g. `ARCH-DRY`) in plans, `## Log` entries, and review findings.

This file is the single source; it is embedded into the planning, plan-quality,
and code-review prompts. The human narrative lives in AGENTS.md "Core Design
Principles"; this is its machine-delivered companion.

## ARCH-DRY — Don't Repeat Yourself

- **principle:** Reuse before adding. One source of truth per fact/behavior; no
  duplicated logic, copy-pasted blocks, or parallel functions that should be one
  shared helper.
- **at-plan:** Flag a plan that re-implements something the codebase already has,
  or that will obviously duplicate logic across the new files instead of
  extracting a shared helper. Name the existing thing it should reuse.
- **at-review:** Flag duplicated logic / copy-pasted blocks / near-identical
  functions in the diff; point at the consolidation (file:line + the shared
  helper they should become).

## ARCH-PURE — Pure core, thin IO shell

- **principle:** The majority of code is pure functions (deterministic, no side
  effects); a thin "glue" layer at the boundary touches IO/UI/network/clock. Pure
  functions are unit-tested directly; the glue is kept small and injected.
- **at-plan:** Flag a design that buries business logic inside IO/handlers, or
  that will only be testable with heavy mocks (a sign logic isn't separated from
  IO). The plan should name what's pure vs the thin IO seam.
- **at-review:** Flag business logic mixed with IO in the diff; logic that should
  be a pure function injected into a thin caller. If a test needs mocks to run a
  "pure" entity, it isn't pure — recommend extracting the IO to the boundary.

## ARCH-PURPOSE — Serve the issue's actual purpose

- **principle:** Deliver the issue's stated purpose, not the easy subset of it. A
  single-source / "compiled to consumers" change is not done until **every
  consumer derives** from the source — the source is *enforced*, not just
  documentation a surface happens to restate; a hand-maintained restatement of the
  model is a deferred consumer, not a finished one. "Follow-up" is for separable
  extensions, never for the thing that is the point. This is the *opposite axis*
  from Simplicity-First/YAGNI: not "build for an imagined future," but "don't
  **under**-deliver the purpose you already committed to."
- **at-plan:** Flag a plan whose scope is a strict subset of the issue's stated
  goal / Done-when where the part deferred as "follow-up" *is* the purpose (e.g.
  wires one consumer + enforcement but leaves the consumers that motivated the
  issue as documentation that doesn't derive). Ask: does the plan fulfill the
  purpose, or just the cheap win? Name the deferred purpose.
- **at-review:** Does the diff *fulfill* the purpose or settle for the easy win?
  For a single-source change, run the **shadow-sweep** — enumerate the consumers,
  confirm each derives from the source, flag any remaining hand-maintained
  restatement of the model. A "follow-up" that is actually the deferred point of
  the issue is a finding, not a deferral.


exec
/bin/zsh -lc 'git diff --stat 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
 .../000113-review-footnote-question-boundary.md    | 32 ++++++++++++++++------
 1 file changed, 23 insertions(+), 9 deletions(-)

exec
/bin/zsh -lc 'git diff --name-status 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
M	workshop/issues/000113-review-footnote-question-boundary.md

codex
The diff window in `pair` currently contains only the issue file, while the issue log says the actual implementation and tests are in sibling `../parley.nvim`. I’m checking whether `pair` has any in-repo consumer or vendored surface that should have changed; for this review I’m staying inside the `pair` repository as requested.
exec
/bin/zsh -lc 'git diff --color=never 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5..HEAD -- workshop/issues/000113-review-footnote-question-boundary.md' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
diff --git a/workshop/issues/000113-review-footnote-question-boundary.md b/workshop/issues/000113-review-footnote-question-boundary.md
index 7a0c3f7..088a818 100644
--- a/workshop/issues/000113-review-footnote-question-boundary.md
+++ b/workshop/issues/000113-review-footnote-question-boundary.md
@@ -26,9 +26,9 @@ For review question submission:
 - A line beginning with a Markdown footnote definition, matching
   `^%[%^[^%]]+%]:`, starts a footnote section.
 - The final question body stops before the first such footnote-definition line.
-- Agent answers for that question are inserted above the first footnote
-  definition line, with one blank line separating the answer block from the
-  footnotes.
+- Agent answers for that question are inserted above the managed footnote
+  footer. If a `---` divider immediately precedes the footnote definitions,
+  keep it with the footer instead of splitting the footer.
 - Existing behavior is preserved when no footnote-definition line follows the
   final question.
 
@@ -42,8 +42,8 @@ footnote detection in the UI shell.
 
 - A regression reproduces `💬:` followed by `---` and `[^acos]: ...`, proving
   only the question text is submitted.
-- A regression proves the answer insertion point is above the first
-  footnote-definition line with a blank line before the footnotes.
+- A regression proves the answer insertion point is above the managed footnote
+  footer.
 - Existing review question tests still pass.
 
 ## Estimate
@@ -58,12 +58,12 @@ total: 0.81
 
 ## Plan
 
-- [ ] Find the pure/helper code that extracts the final review question and
+- [x] Find the pure/helper code that extracts the final review question and
       chooses the answer insertion point.
-- [ ] Add a failing regression for a final `💬:` followed by Markdown footnotes.
-- [ ] Teach the range/insertion calculation to treat a leading `[^...]:`
+- [x] Add a failing regression for a final `💬:` followed by Markdown footnotes.
+- [x] Teach the range/insertion calculation to treat a leading `[^...]:`
       footnote definition as the trailing metadata boundary.
-- [ ] Run focused review/question tests and close #113.
+- [x] Run focused review/question tests and close #113.
 
 ## Log
 
@@ -71,3 +71,17 @@ total: 0.81
 - Created from reported bug: final review question followed by definition
   footnotes gets submitted together with the footnotes, and the answer is
   inserted below the footnote block.
+- Root cause is in peer `../parley.nvim`: `chat_parser.parse_chat` finalized a
+  trailing open `💬:` question at EOF, so the exchange model counted the
+  managed footnote footer as part of the question and inserted the answer after
+  it.
+- Implemented in `../parley.nvim/lua/parley/chat_parser.lua`: a final open
+  question now treats a trailing column-1 `[^...]:` footnote block as metadata,
+  and keeps an immediately preceding `---` divider with that footer.
+- Added regressions in `../parley.nvim/tests/unit/parse_chat_spec.lua` for
+  submitted question content and model insertion point.
+- Verified with `nvim --headless --noplugin -u tests/minimal_init.vim -c
+  "PlenaryBustedFile tests/unit/parse_chat_spec.lua"`,
+  `nvim --headless --noplugin -u tests/minimal_init.vim -c
+  "PlenaryBustedFile tests/unit/build_messages_spec.lua"`, and `make test` in
+  `../parley.nvim`.

exec
/bin/zsh -lc "rg -n \"parse_chat|review question|💬:|footnote|\\[\\"'^|parley|chat_parser" . --glob '"'"'!workshop/history/**'"' --glob '"'!node_modules/**'"'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
./atlas/how-to-bring-up-a-new-harness-cli.md:122:    claude = [[^❯]],
./atlas/how-to-bring-up-a-new-harness-cli.md:123:    codex  = [[^›]],
./README.md:62:| **Shift+Alt+d** | review pane (visual) | Define the selected term inline. The pair agent answers through `pair review definition`, and the pane stores the result as a durable footnote. |
./atlas/review-workbench.md:26:- `markers.lua` (M2) — pure 🤖 review-request parser (ported from parley):
./atlas/review-workbench.md:76:  parley): per-buffer snapshots keyed by content hash; on undo/redo restore the
./atlas/review-workbench.md:88:- `resolve.lua` — pure parley §5 accept/reject resolution for `🤖` marker chains;
./atlas/review-workbench.md:204:  text to `term[^id]`, appends or updates a managed final `---` footnote block,
./atlas/review-workbench.md:205:  and rehydrates diagnostics/highlights from the durable footnotes. Definition
./atlas/architecture.md:357:Pane detection: parse `list-panes --json --command` (via `cmd/internal/zellijpane`), find the focused non-plugin/non-floating pane, and match its `terminal_command` — **not** its `title` — against `nvim|draft`. Keying on `terminal_command` is the #copy-on-select-test fix: the agent overwrites its pane title with `claude [<cwd>]`, so a repo path containing `nvim` (e.g. `parley.nvim`) would misclassify the agent pane as the draft and skip the paste; `terminal_command` never embeds the cwd.
./atlas/architecture.md:461:**Comment markers — `Alt+q` in viewer → draft pickup (#000018).** While reading scrollback, `Alt+q` drops a parley-style `🤖[]` marker at the cursor (or `🤖<selection>[…]` in visual mode). The buffer is read-only, so the keymap lifts `modifiable`/`readonly` for the insert and re-locks immediately. (#57: this whole marker subsystem was extracted to the shared `nvim/annotate.lua` — the change-log viewer uses the identical flow; see "Shared 🤖-marker annotation" above.) On viewer exit (`VimLeavePre`), `nvim/annotate.lua` (`M.emit`) walks every line, parses each `🤖<X>?[Y]` marker by literal-byte scan (Lua patterns aren't UTF-8 aware), and writes a formatted block to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`:
./atlas/architecture.md:484:- `<M-i>` (Alt+i, normal+insert) — `attach_image`: capture-driven image attach. 1) Verify the OS clipboard holds image data (macOS: AppleScript `clipboard info` enumerates `PNGf`/`TIFF`/etc.; Linux: `wl-paste --list-types` or `xclip -t TARGETS`) — if not, flash `[no image in clipboard]` as inline virt_text for 1s and bail. 2) Read pair-wrap's pid from `$PAIR_DATA_DIR/pair-wrap-pid-<tag>` (notify+abort if missing/dead, since pair-wrap is the whole agent I/O path). 3) `kill -USR1 <pid>` to arm a ~200ms capture window in pair-wrap, then `zellij action write 22` to send Ctrl+V to the agent pane. 4) Poll `image-capture-<tag>.done` (20ms cadence, 600ms cap); on hit, read `image-capture-<tag>`, strip ANSI, regex `%[Image[ #][^%]]+%]` (matches both claude's `[Image #N]` and agy's `[Image N-M]`) and insert the captured marker verbatim at cursor. The agent is the source of truth for the marker text — no local counter, no per-agent format hardcoded.
./Makefile.local:204:# "claude [<cwd>]", so a repo path containing "nvim" (e.g. parley.nvim) must not
./nvim/changelog.lua:40:      syntax match ChangelogCode      /`[^`]\+`/
./nvim/scrollback.lua:119:  -- "reset + bold" — splitting on `[^;]+` would drop the empty leading
./nvim/scrollback.lua:121:  -- a trailing `;` and matching `([^;]*);` makes every field — empty
./nvim/scrollback.lua:124:  for code in (params .. ';'):gmatch('([^;]*);') do
./nvim/scrollback.lua:384:  claude = [[^❯]],
./nvim/scrollback.lua:385:  codex  = [[^›]],
./workshop/issues/000113-review-footnote-question-boundary.md:12:# review footnotes should not join last question
./workshop/issues/000113-review-footnote-question-boundary.md:16:When the review pane sends a final `💬:` question at the end of a document that
./workshop/issues/000113-review-footnote-question-boundary.md:17:is followed by a Markdown footnote block, the question extraction includes the
./workshop/issues/000113-review-footnote-question-boundary.md:18:footnote definitions. The agent response is then inserted after the footnotes
./workshop/issues/000113-review-footnote-question-boundary.md:24:For review question submission:
./workshop/issues/000113-review-footnote-question-boundary.md:26:- A line beginning with a Markdown footnote definition, matching
./workshop/issues/000113-review-footnote-question-boundary.md:27:  `^%[%^[^%]]+%]:`, starts a footnote section.
./workshop/issues/000113-review-footnote-question-boundary.md:28:- The final question body stops before the first such footnote-definition line.
./workshop/issues/000113-review-footnote-question-boundary.md:29:- Agent answers for that question are inserted above the managed footnote
./workshop/issues/000113-review-footnote-question-boundary.md:30:  footer. If a `---` divider immediately precedes the footnote definitions,
./workshop/issues/000113-review-footnote-question-boundary.md:32:- Existing behavior is preserved when no footnote-definition line follows the
./workshop/issues/000113-review-footnote-question-boundary.md:39:footnote detection in the UI shell.
./workshop/issues/000113-review-footnote-question-boundary.md:43:- A regression reproduces `💬:` followed by `---` and `[^acos]: ...`, proving
./workshop/issues/000113-review-footnote-question-boundary.md:45:- A regression proves the answer insertion point is above the managed footnote
./workshop/issues/000113-review-footnote-question-boundary.md:47:- Existing review question tests still pass.
./workshop/issues/000113-review-footnote-question-boundary.md:61:- [x] Find the pure/helper code that extracts the final review question and
./workshop/issues/000113-review-footnote-question-boundary.md:63:- [x] Add a failing regression for a final `💬:` followed by Markdown footnotes.
./workshop/issues/000113-review-footnote-question-boundary.md:64:- [x] Teach the range/insertion calculation to treat a leading `[^...]:`
./workshop/issues/000113-review-footnote-question-boundary.md:65:      footnote definition as the trailing metadata boundary.
./workshop/issues/000113-review-footnote-question-boundary.md:71:- Created from reported bug: final review question followed by definition
./workshop/issues/000113-review-footnote-question-boundary.md:72:  footnotes gets submitted together with the footnotes, and the answer is
./workshop/issues/000113-review-footnote-question-boundary.md:73:  inserted below the footnote block.
./workshop/issues/000113-review-footnote-question-boundary.md:74:- Root cause is in peer `../parley.nvim`: `chat_parser.parse_chat` finalized a
./workshop/issues/000113-review-footnote-question-boundary.md:75:  trailing open `💬:` question at EOF, so the exchange model counted the
./workshop/issues/000113-review-footnote-question-boundary.md:76:  managed footnote footer as part of the question and inserted the answer after
./workshop/issues/000113-review-footnote-question-boundary.md:78:- Implemented in `../parley.nvim/lua/parley/chat_parser.lua`: a final open
./workshop/issues/000113-review-footnote-question-boundary.md:79:  question now treats a trailing column-1 `[^...]:` footnote block as metadata,
./workshop/issues/000113-review-footnote-question-boundary.md:81:- Added regressions in `../parley.nvim/tests/unit/parse_chat_spec.lua` for
./workshop/issues/000113-review-footnote-question-boundary.md:84:  "PlenaryBustedFile tests/unit/parse_chat_spec.lua"`,
./workshop/issues/000113-review-footnote-question-boundary.md:87:  `../parley.nvim`.
./nvim/review.lua:241:-- 🤖 marker highlight groups. Keep these aligned with parley.nvim's review mode
./nvim/review.lua:242:-- so pair and parley render the marker language consistently across themes.
./nvim/review.lua:285:-- Accept/reject the 🤖 marker on the cursor's line (M4b, parley §5 via resolve.lua):
./nvim/review.lua:549:  local body = define.strip_definition_footnote_footer(buf_content(buf))
./nvim/review.lua:605:  local applied = define.apply_definition_footnote(
./nvim/review.lua:657:    context = define.strip_definition_footnote_footer(buf_content(buf)),
./nvim/review.lua:729:  -- Alt+a/r/q mirror parley.nvim's review-mode shortcuts now that the pane toggle
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:8:💬: is there a way for external process to inject text into TUI (say claude code's input window)?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:39:💬: so zellij works?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:66:💬: so, if I have two pane working in a ghostty terminal (or whatever), one on the top, and one on the bottom. If I run zellij on the top, and then claude inside that zellij. and in the bottom zellij, I run an nvim. I can do some scripting in nvim, so it sends what's in current buffer to the claude instance running in that zellij in the pane above?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:101:💬: Setup B is nice is the sense inside a single zellij, you can:
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:138:💬: [Image: screenshot of working setup. Text:] so this is the designated input window for claude. should support all functionalities, including paste image. the benefit of this setup, is that you can scroll independently. a couple of additional nice things: 1/ nvim you can insert some clean up, so for example, the copy from claude line breaks at terminal width, we can patch up a single line; 2/ we probably can have hot key, when focus is in claude's pane, to send selected text (already in clipboard) to nvim's current cursor.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:167:💬: this should also support all TUI based coding agents, e.g. codex and gemini as well. not sure if they support paste image yet. let's start with claude.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:181:💬: in terms of artifact of such "claude+" setup, where do we need to change? I'm thinking about how to packaging this for other people to use.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:191:💬: a couple of more thoughts:
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:259:💬: ok, I created a ../pair folder for this project, let's go. treat the pensive as description of this issue. make an issue in brain first.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:269:💬: actually, I also setup ../pair as an ariadne repo, move the ticket there as its first issue.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:277:💬: ok, I've committed ../pair, now go work on it
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:294:💬: what are the keys
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:311:💬: ~/workspace/pair (main*) » bin/pair
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:324:💬: > The big one: you're running pair from inside an existing zellij session
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:341:💬: works!
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:349:💬: is the image pasting working as well?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:359:💬: make the draft window smaller, 8 lines.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:367:💬: nvim, make it taller, currently on my mac, it has 4 lines, one line used by status, one line command, then about 2 lines for zellij's frame for nvim
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:375:💬: make it 10
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:383:💬: can you remove the status line in nvim? or the command line, actually merge them
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:391:💬: what's the hot key to insert image in our nvim (alt-i?)
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:401:💬: doesn't work in either insert mode or normal mode
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:420:💬: for image, how can it just work? you do need some additional dance, e.g. after the paste (send ctrl+v to claude), claude would generate a [Image #2] in cursor location as reference. you need to do that as well in the nvim cursor location. you also need to keep track of how many images have we paste, either that, or somehow read from claude's TUI what feedback [Image #?] did claude insert, use that, then clear out that text in claude's window by sending ctrl+u (I think is that?)
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:434:💬: ah, so in this setup, I will (or at least didn't plan to) use claude's own input. everything would be done in nvim, then alt-return. that's why sending ctrl-u to clear claude input's ok, as we never intend to use state there anyway.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:464:💬: no, you shouldn't counter reset. in a single claude session, the counter continues. not per message.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:474:💬: actually I also realize this setup, can support the paradigm of claude running in ssh, and nvim running locally, essentially creating a bridge. right?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:486:💬: now, we also need to support multiple bin/pair being issues, potentially on different agnet (claude, codex, gemini). is this supported today
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:496:💬: dude, what about 3 claude sessions?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:512:💬: when we ctrl-q quit, can you in bin/pair, to run zellij kill-session on that session? or, you should extend the bin/pair to ask your to select which existing sessions to connect to, in an TUI type of thing.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:520:💬: actually, since in zellij paradigm, ctrl-q is not erase everything, merely detact and can be later reattached, let's keep that semantic. since we are implementing the picker as well. basically only implement the 2nd part of my ask.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:528:💬: the bin/picker should just start new session when there's none; when there are detached, show picker to reattach, or one option to start new (e.g. + new session)
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:536:💬: note you need to implement the picker selection based on the agent selected, if bin/picker codex is issued, for example, only look for codex sessions. did you do that?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:551:💬: ok, tested pair without any sessions -> good, new session started.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:562:💬: but your script doesn't work, pair seems always create a new session, despite some seesions have no clients:
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:574:💬: ok, right now, there are three sessions, one is connected, two are not. when I do `pair` now, it connects directly to session 2. in this case, when there are more than 1 unconnected sessions, there should be a picker for user to pick which session to connect to.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:582:💬: will you be able to bind Alt-d to detach, e.g. instead of use the default Ctrl-o+d
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:596:💬: ok, when I created a new session with bin/pair, the title of my terminal changed to the slug name used, e.g. pair-claude-5. however, when reattached to pair-claude-5, that didn't happen, the title of the terminal stayed as bin/pair. why?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:611:💬: cool. can you now fix issue #2, basically allow naming of a session instead of auto assigned claude-pair-5. user should be prompted for new name, whenever you decide a new session needs to be created. default to the existing naming scheme e.g. claude-pair-5.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:636:💬: add help text with bin/pair --help
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:644:💬: bin/pair didn't show the session I created with name blogging, I think it's intentional from you, can you explain how this worked
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:657:💬: i see, the auto picking behavior is not that useful, because in the bin/pair world, claude sessions are long lived. and auto attach to some long running session that user don't know, is not a good mental model. we can remove auto attach behavior, and always show the picker.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:669:💬: bin/pair only shows the names pair-claude* still. in the new world, you should show all. right?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:679:💬: i was wrong, and not understanding the zellij state. do some research to show how many "endpoint" is connected to a zellij session, since inside zellij, it shows with those colorful blocks of multiple connection to a single session, I assume there's a way to figure it out from command line
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:687:💬: ok, zellij --session NAME action list-clients works.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:699:💬: ok, right now, there are three sessions, one is connected, two are not. when I do `pair` now, it connects directly to session 2. in this case, when there are more than 1 unconnected sessions, there should be a picker for user to pick which session to connect to.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:707:💬: I makde some mistake, can you recheck if the following is implemented. my test shows not.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:723:💬: it's my fault, I was moneying on the side and made a wrong operation outside this chat.
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:735:💬: update the blog post with the issues I listed with claude code's input box. actually, can you display it here?
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:745:💬: can you export the raw transcript of the feature we worked togehter starting from my "is there a way for external process to inject text into TUI (say claude code's input window)?"
./workshop/parley/2026-05-02.08-30-00.000_pair-creation.md:747:export to another file, maybe of the ../parley.nvim export style of a chat markdown file?
./nvim/init.lua:165:    vim.cmd([[syntax match markdownPairBracket /\[[^\]\n]\{-1,}\]\%(\s*[[(]\)\@!/]])
./nvim/init.lua:997:  for line in (body .. '\n'):gmatch('([^\n]*)\n') do
./nvim/init.lua:1020:  for line in (body .. '\n'):gmatch('([^\n]*)\n') do
./nvim/init.lua:1053:    for line in (non_comm .. '\n'):gmatch('([^\n]*)\n') do
./nvim/init.lua:1094:local PAIR_IMAGE_MARKER_RE = '%[Image[ #][^%]]+%]'
./nvim/init.lua:1154:  s = s:gsub('\27%][^\7\27]*\7', '')
./nvim/init.lua:1155:  s = s:gsub('\27%][^\7\27]*\27\\', '')
./nvim/init.lua:1438:  for line in (reflowed .. '\n'):gmatch('([^\n]*)\n') do
./nvim/init.lua:1681:  local dir, filter = token:match('^(.*/)([^/]*)$')
./nvim/init.lua:1758:    for c in env:gmatch('[^,]+') do set[c] = true end
./nvim/init.lua:1864:      local color, count, span = l:match('^([^\t]+)\t(%d+)\t(.+)$')
./nvim/init.lua:2125:  local stripped = s:gsub('%%#[^#]*#', '')
./nvim/init.lua:4034:  local cur = read_file(path):match('^[^\n]*') or ''
./nvim/init.lua:4042:  local proposed = read_file(dd .. '/slug-proposed-' .. tag):match('^[^\n]*') or ''
./nvim/init.lua:4148:    for line in out:gmatch('[^\n]+') do
./workshop/continuation/20260701T225328-m4-build.md:53:- **Tests:** Go unit tests (isNvimCommand incl. the parley.nvim-cwd false-positive
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:6:description: parley's one-shot review is amnesiac; the fix is to stop making parley and Claude Code compete as harnesses — let pair's persistent agent be the conversational workbench and an embedded review nvim be the document workbench, with the agent proposing {old,occurrence,new,explain} records that the review pane applies undo-ably and commits (docflow).
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:7:references: [../parley.nvim/lua/parley/skills/review/init.lua, ../parley.nvim/lua/parley/skills/review/journal.lua, ../parley.nvim/lua/parley/skills/review/projection.lua, ../parley.nvim/lua/parley/skills/review/mode.lua, ../parley.nvim/lua/parley/tools/builtin/propose_edits.lua, ../ariadne/construct/skills/fix, ../42shots]
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:12:The thing that's been bugging me about parley's `review` is that it's **amnesiac**.
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:13:It's one-shot: parley assembles a prompt, forces a single `propose_edits` tool call,
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:21:**The eureka is that I've been framing this as parley-the-harness vs. Claude-Code-the-harness,
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:23:cross-repo memory). parley is the better *document* surface (in-editor 2D rendering of
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:36:- **Embed review in pair as inline lua by extracting parley's _consumer half_.** Render,
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:53:  forces "the agent proposes records, parley applies" rather than "the agent writes the file,
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:54:  parley reloads": a reload resets the buffer's undo. Three layers, none collapsible — nvim
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:62:- **Modes come along.** Borrow parley's review modes (aggressiveness / how the LLM edits and
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:82:- **Keep parley's in-process one-shot as a no-pair fallback producer of the same contract, or
./workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md:86:- **Divergence:** extracting (copying) parley's review into pair forks the code. Acceptable for
./workshop/lessons.md:8:the footnote reference into the wrong text.
./workshop/lessons.md:111:## Empty fields in delimited parsing — `[^;]+` drops them; semantics may differ
./workshop/lessons.md:113:ECMA-48 SGR semantics: an omitted field is `0` (reset). So `\x1b[;1m` = "reset; bold". The first SGR parser pass used `params:gmatch('[^;]+')`, which silently skips empty fields — `[;1m` produced just `1` (bold), and any standing fg/bg/decoration leaked through. Caught in code review of #000017 (no real input from pair-scrollback-render's output would have triggered it, but it's a correctness footgun for any future caller pointing the viewer at non-pair-rendered ANSI).
./workshop/lessons.md:115:**Rule.** When the protocol says "empty field has meaning," parse with `([^;]*);` on a `string + ';'` so the trailing-delimiter trick yields every field including empties. Generally true for any delimiter-separated format where omission has semantic value (CSV with empty cells, env-var lists, SGR, etc.).
./bootstrap.sh:91:        [[ "$line" =~ ^[[:space:]]*replace[[:space:]]+[^[:space:]]+([[:space:]]+[^[:space:]]+)?[[:space:]]+=\>[[:space:]]+([^[:space:]]+) ]] || continue
./tests/review-definition-test.sh:3:# managed footnotes and render exact-span diagnostics/highlights.
./tests/review-definition-test.sh:97:  'here is ASIN[^asin] in context',
./tests/review-definition-test.sh:101:  '[^asin]: Amazon Standard Identification Number.',
./tests/review-definition-test.sh:102:}, '\n'), 'definition persisted as managed footnote')
./tests/review-definition-test.sh:106:  'definition highlight spans only term plus footnote ref')
./tests/review-definition-test.sh:121:  'here is ASIN[^asin] in context',
./tests/review-definition-test.sh:125:  '[^asin]: Amazon Standard Identification Number.',
./tests/review-definition-test.sh:133:ok(request_doc2.context == 'here is ASIN[^asin] in context',
./tests/review-definition-test.sh:134:  'definition request context strips managed footnote footer')
./tests/review-definition-test.sh:145:  'here is ASIN[^asin] in context',
./tests/review-definition-test.sh:149:  '[^asin]: Updated definition.',
./tests/review-definition-test.sh:150:}, '\n'), 'redefinition updates existing footnote without duplicate ref')
./tests/review-definition-test.sh:157:  'rehydrate_definitions redraws exact span from durable footnote')
./tests/review-definition-test.sh:160:ok(context_body == 'here is ASIN[^asin] in context',
./tests/review-window-test.sh:422:grep -q '^review-user-hl$' "$RT/r3" && pass "review user marker highlight matches parley" || fail "review user marker highlight"
./tests/review-window-test.sh:423:grep -q '^review-agent-hl$' "$RT/r3" && pass "review agent marker highlight matches parley" || fail "review agent marker highlight"
./tests/review-window-test.sh:424:grep -q '^review-quoted-hl$' "$RT/r3" && pass "review quoted marker highlight matches parley" || fail "review quoted marker highlight"
./tests/review-window-test.sh:425:grep -q '^review-strike-hl$' "$RT/r3" && pass "review strike marker highlight matches parley" || fail "review strike marker highlight"
./nvim/review/define.lua:35:function M.footnote_id(term)
./nvim/review/define.lua:37:  id = id:gsub('[^%w]+', '-')
./nvim/review/define.lua:43:function M.format_footnote_line(id, definition)
./nvim/review/define.lua:46:  return string.format('[^%s]: %s', id, definition)
./nvim/review/define.lua:69:local function is_footnote_line(line)
./nvim/review/define.lua:70:  return trim(line):match('^%[%^[^%]]+%]:') ~= nil
./nvim/review/define.lua:77:      local has_footnote = false
./nvim/review/define.lua:81:          if not is_footnote_line(line) then return nil end
./nvim/review/define.lua:82:          has_footnote = true
./nvim/review/define.lua:85:      if has_footnote then return i end
./nvim/review/define.lua:92:local function parse_footnote_line(line)
./nvim/review/define.lua:93:  local id, definition = trim(line):match('^%[%^([^%]]+)%]:%s*(.-)%s*$')
./nvim/review/define.lua:100:function M.strip_definition_footnote_footer(text)
./nvim/review/define.lua:111:local function replace_or_append_footnote(lines, id, definition)
./nvim/review/define.lua:114:  local footnote_line = M.format_footnote_line(id, definition)
./nvim/review/define.lua:116:    local escaped_id = id:gsub('([^%w])', '%%%1')
./nvim/review/define.lua:119:        out[i] = footnote_line
./nvim/review/define.lua:123:    out[#out + 1] = footnote_line
./nvim/review/define.lua:131:  out[#out + 1] = footnote_line
./nvim/review/define.lua:135:function M.apply_definition_footnote(lines, l1, c1, l2, c2, term, definition)
./nvim/review/define.lua:136:  local id = M.footnote_id(term)
./nvim/review/define.lua:137:  local ref = '[^' .. id .. ']'
./nvim/review/define.lua:145:  out = replace_or_append_footnote(out, id, definition)
./nvim/review/define.lua:167:    if term ~= '' and M.footnote_id(term) == id then
./nvim/review/define.lua:174:function M.footnote_diagnostics(lines)
./nvim/review/define.lua:181:    local id, definition = parse_footnote_line(lines[i] or '')
./nvim/review/define.lua:190:      local ref_start, ref_end, id = line:find('%[%^([^%]]+)%]', search)
./nvim/review/apply.lua:148:-- parley's diag_wrap_width; falls back to 76 with no window.
./workshop/targets/review-protocol.md:177:  mode; `Alt+Shift+Return` opens a send menu (parley's UI shape: mode list plus
./nvim/review/markers.lua:2:-- Ported from parley's review/init.lua (the LLM-invoke half is dropped). No vim
./workshop/continuation/20260619T133818-review-m3.md:31:Started as a *design brainstorm* on parley.nvim's amnesiac one-shot review; converged on the **eureka**: stop making parley and Claude-Code compete as harnesses — pair's persistent agent is the *conversational* workbench, an embedded nvim is the *document* workbench, a review state file is the seam. Then built it milestone-by-milestone (M0 contract → M1 spine → M2 render → M3 window). The work is a deliberate vertical slice of the **42shots human-AI knowledge-work interface** (a doc tip over a brain/repo corpus, agent discovery, human steering by comment).
./nvim/review/seam.lua:73:  return M.normalize_mode(body:match('([^\r\n]+)') or '')
./nvim/review/define_test.lua:35:eq(define.footnote_id('Amazon Standard Identification Number'),
./nvim/review/define_test.lua:37:  'footnote_id slugifies prose')
./nvim/review/define_test.lua:38:eq(define.footnote_id(''), 'definition', 'empty footnote id falls back')
./nvim/review/define_test.lua:45:local applied = define.apply_definition_footnote(
./nvim/review/define_test.lua:52:  'here is ASIN[^asin] in context',
./nvim/review/define_test.lua:56:  '[^asin]: Amazon Standard Identification Number.',
./nvim/review/define_test.lua:57:}, 'apply_definition_footnote inserts inline ref and managed footer')
./nvim/review/define_test.lua:63:}, 'apply_definition_footnote returns exact selected-ref span')
./nvim/review/define_test.lua:65:local redefined = define.apply_definition_footnote(applied.lines, 1, 8, 1, 11, 'ASIN', 'Updated.')
./nvim/review/define_test.lua:67:  'here is ASIN[^asin] in context',
./nvim/review/define_test.lua:71:  '[^asin]: Updated.',
./nvim/review/define_test.lua:79:  'not a footnote',
./nvim/review/define_test.lua:81:eq(define.strip_definition_footnote_footer(ordinary), ordinary,
./nvim/review/define_test.lua:82:  'strip_definition_footnote_footer preserves ordinary trailing divider prose')
./nvim/review/define_test.lua:83:local ordinary_footnotes = table.concat({
./nvim/review/define_test.lua:84:  'main text[^source]',
./nvim/review/define_test.lua:87:  '[^source]: ordinary existing footnote',
./nvim/review/define_test.lua:89:eq(define.strip_definition_footnote_footer(ordinary_footnotes), ordinary_footnotes,
./nvim/review/define_test.lua:90:  'strip_definition_footnote_footer preserves ordinary divider footnotes without managed blank')
./nvim/review/define_test.lua:91:eq(define.strip_definition_footnote_footer(table.concat(redefined.lines, '\n')),
./nvim/review/define_test.lua:92:  'here is ASIN[^asin] in context',
./nvim/review/define_test.lua:93:  'strip_definition_footnote_footer removes only final managed footnote footer')
./nvim/review/define_test.lua:95:eq(define.footnote_diagnostics(redefined.lines), {
./nvim/review/define_test.lua:105:}, 'footnote_diagnostics derives exact span and stored definition')
./nvim/review/define_test.lua:106:local phrase = define.apply_definition_footnote(
./nvim/review/define_test.lua:112:eq(define.footnote_diagnostics(phrase.lines), {
./nvim/review/define_test.lua:122:}, 'footnote_diagnostics rehydrates the full selected phrase span')
./nvim/review/define_test.lua:123:eq(define.footnote_diagnostics({
./nvim/review/define_test.lua:124:  'main text[^source]',
./nvim/review/define_test.lua:127:  '[^source]: ordinary existing footnote',
./nvim/review/define_test.lua:128:}), {}, 'footnote_diagnostics ignores ordinary divider footnotes without managed blank')
./nvim/review/menu.lua:2:-- Shape follows parley.nvim: a mode selector plus an optional instruction buffer.
./nvim/review/projection.lua:2:-- Ported from parley's review/projection.lua; draws via review.apply's
./nvim/review/projection.lua:3:-- snapshot/apply_snapshot (M2) instead of parley's skill_render.
./nvim/review/wrap.lua:4:-- API); ported from parley.nvim's skill_render.M.wrap. Standalone like record.lua
./cmd/internal/clipcmd/clipcmd.go:35:// contains "nvim" (e.g. parley.nvim) would misclassify the agent pane as the
./cmd/internal/wrapcmd/wrap.go:183:	otherEscRe    = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()*+][@-~]|\x1b[@-Z\\-_]`)
./cmd/internal/wrapcmd/wrap.go:184:	imageMarkerRe = regexp.MustCompile(`\[Image[ #][^\]]+\]`)
./cmd/internal/wrapcmd/wrap.go:185:	oscRe         = regexp.MustCompile(`\x1b\](\d+);([^\x07\x1b]*)(?:\x07|\x1b\\)`)
./cmd/internal/zellijpane/zellijpane_test.go:12:     "title":"claude [~/workspace/parley.nvim]",
./cmd/internal/zellijpane/zellijpane_test.go:36:	if agent.Title != "claude [~/workspace/parley.nvim]" {
./cmd/internal/clipcmd/clipcmd_test.go:19:		// title with "claude [~/workspace/parley.nvim]", but the in_nvim gate
./cmd/internal/clipcmd/clipcmd_test.go:22:		{"parley.nvim agent cmd (no cwd)", `sh -c exec pair-wrap --scrollback-log /d/s.raw claude`, false},
./cmd/internal/clipcmd/clipcmd_test.go:82:// title contains "nvim" (parley.nvim cwd) but whose terminal_command is the
./cmd/internal/clipcmd/clipcmd_test.go:88:		{ID: "0", IsFocused: true, Title: "claude [~/workspace/parley.nvim]",
./nvim/review/init.lua:161:  apply.place_definitions(buf, define.footnote_diagnostics(lines))
./tests/copy-on-select-test.sh:6:# /nvim|draft/, so in a repo whose path contains "nvim" (e.g. parley.nvim) the
./tests/copy-on-select-test.sh:51:# Agent pane: title carries the cwd (parley.nvim → contains "nvim"), but
./tests/copy-on-select-test.sh:54:  "title":"claude [~/workspace/parley.nvim]",
./tests/copy-on-select-test.sh:70:# (a) Selection in the AGENT pane while cwd is parley.nvim → must hand off
./tests/copy-on-select-test.sh:76:wait_staged || { echo "FAIL (a) parley.nvim agent-pane selection did not hand off (quote not staged)"; fail=1; }
./cmd/internal/clipcmd/run_test.go:89:	   "title":"claude [~/workspace/parley.nvim]",
./cmd/internal/clipcmd/run_test.go:141:// fixture: a selection in the AGENT pane while cwd is parley.nvim (title contains
./cmd/internal/launcher/scope_test.go:53:		{in: "parley.nvim", want: "parley_nvim"},
./cmd/internal/launcher/scope_test.go:67:	scope, err := ResolveRepoScope("/Users/alice/work/parley.nvim")
./cmd/internal/launcher/config_test.go:39:	if got := encodeCwd("/Users/x/workspace/parley.nvim"); got != "-Users-x-workspace-parley-nvim" {
./cmd/internal/reviewcmd/reviewcmd.go:73:var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

exec
/bin/zsh -lc "rg --files | sed -n '1,220p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
construct/deps
construct/go.sum
construct/base.manifest
atlas/how-to-bring-up-a-new-harness-cli.md
atlas/index.md
atlas/review-workbench.md
atlas/go-migration-inventory.md
atlas/architecture.md
atlas/session-identity.md
zellij/layouts/main.kdl
zellij/config.kdl
bootstrap.sh
README.md
docs/pair-scribe.md
docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md
go.sum
tests/pair-rename.sh
tests/run-headless-test.sh
tests/changelog-notify-test.sh
tests/review-loop-test.sh
tests/pair-restart-quit-test.sh
tests/review-apply-test.sh
tests/pair-session-watch-test.sh
tests/emitter-health-test.sh
tests/review-definition-test.sh
tests/review-reconcile-test.sh
tests/dev-rebuild-test.sh
tests/lib/fake-review-agent.sh
tests/lib/fake-docflow.sh
tests/lib/run-headless.sh
tests/lib/fail-docflow.sh
tests/cr-newline-test.sh
tests/review-window-test.sh
tests/pair-embedded-runtime-test.sh
tests/scrollback-open-test.sh
tests/changelog-session-key-test.sh
tests/zellij-trace-test.sh
tests/review-poke-test.sh
tests/review-handoff-test.sh
tests/review-indicator-test.sh
tests/pair-review-target-test.sh
tests/changelog-open-test.sh
tests/adapt-schema-test.sh
tests/review-readiness-cli-test.sh
tests/review-toggle-test.sh
tests/queue-send-test.sh
tests/review-projection-test.sh
tests/pair-go-install-layout-test.sh
tests/statusline-pos-test.sh
tests/adapt-golden.expected
tests/review-docflow-test.sh
tests/copy-on-select-test.sh
tests/review-resume-test.sh
tests/autopair-test.sh
doctor/emitter-health.sh
doctor/SKILL.md
doctor/README.md
doctor/doctor.sh
doctor/doctor_test.sh
nvim/changelog.lua
nvim/adapt_test.lua
nvim/doctor_test.lua
nvim/scrollback_test.lua
nvim/annotate.lua
nvim/annotate_test.lua
nvim/adapt.lua
nvim/slug.lua
nvim/changelog_test.lua
nvim/review/spinner_test.lua
nvim/review/define.lua
nvim/review/record_test.lua
nvim/review/poke_bodies_test.lua
nvim/review/seam_test.lua
nvim/review/spinner.lua
nvim/review/resolve.lua
nvim/review/seam.lua
nvim/review/markers_test.lua
nvim/review/menu_test.lua
nvim/review/reconstruct_test.lua
nvim/review/define_test.lua
nvim/review/handoff.lua
nvim/review/menu.lua
nvim/review/projection.lua
nvim/review/wrap.lua
nvim/review/readiness.lua
nvim/review/poke_bodies.lua
nvim/review/resolve_test.lua
nvim/review/mode_test.lua
nvim/review/apply.lua
nvim/review/reconcile.lua
nvim/review/wrap_test.lua
nvim/review/markers.lua
nvim/review/reconstruct.lua
nvim/review/docflow.lua
nvim/review/gate.lua
nvim/review/gate_test.lua
nvim/review/mode.lua
nvim/review/reconcile_test.lua
nvim/review/init.lua
nvim/review/record.lua
nvim/review/readiness_test.lua
nvim/review/definition_seam.lua
nvim/scrollback.lua
nvim/doctor.lua
nvim/pair_poke.lua
nvim/zellij_trace.lua
nvim/review.lua
nvim/marker_codec.lua
nvim/init.lua
nvim/slug_test.lua
CHANGELOG.md
go.mod
AGENTS.local.md
LICENSE
bin/pair-help
bin/pair-notify
Makefile.local
bin/pair-dev
bin/lib/adapt-log.sh
bin/lib/dev-rebuild.sh
workshop/issues/000083-pair-workbench-prescribed-way-of-when-to-use-which-agent.md
workshop/issues/000113-review-footnote-question-boundary.md
workshop/issues/000080-pair-retro-the-ability-to-go-through-tty-log-and-identify-workflow-issues.md
cmd/pair-go/main_test.go
cmd/pair-go/pty_proxy_route_test.go
cmd/pair-go/main.go
cmd/pair-go/helper_equivalence_test.go
cmd/pair-go/changelog_seam_test.go
workshop/continuation/20260701T225328-m4-build.md
workshop/continuation/20260702T161415-launcher-m3.md
workshop/continuation/20260702T215853-launcher-m5.md
workshop/continuation/20260619T133818-review-m3.md
workshop/continuation/20260701T180812-go-port-m4.md
workshop/continuation/20260612T002626-compact.md
workshop/continuation/20260621T180406-pair-pair.md
workshop/continuation/20260615T225403-cont-improve.md
workshop/continuation/20260702T120358-launcher-m2.md
workshop/continuation/20260703T104819-launcher-retired.md
workshop/continuation/20260616T214136-make-test-hang.md
workshop/continuation/20260611T163027-port.md
workshop/continuation/20260702T175427-launcher-m4.md
workshop/pensive/2026-07-02-01-pensive-multi-milestone-sdlc-gotchas.md
workshop/pensive/2026-06-15-01-pensive-user-model-throughline.md
workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md
workshop/pensive/2026-07-07-01-pensive-ctxmeter-frame-count-missing.md
workshop/lessons.md
workshop/history/000089-review-mode-should-disable-edit-while-agent-update-the-doc.md
workshop/history/000074-go-dispatcher-skeleton-close-review.md
workshop/history/000059-changelog-tty-timestamps-plan.md
workshop/history/000094-drop-shell-runtime-assets-plan.md
workshop/history/000055-compact-keybind.md
cmd/internal/runtimebundlegen/generate_test.go
cmd/internal/runtimebundlegen/generate.go
workshop/history/000057-changelog-annotate-unify-plan.md
workshop/history/000067-codex-tag-quit-resume-plan.md
workshop/history/000094-drop-shell-runtime-assets.md
workshop/history/000105-altc-deterministic-restart-draft-fold-plan.md
workshop/history/000085-pair-wrap-stdout-batching.md
workshop/history/000078-go-stateful-shell-glue-plan.md
workshop/history/000109-reset-cmux-title-after-pair-quit-close-review.md
workshop/history/000042-prevent-headless-neovim-execution-from-sending-zellij-keystrokes.md
workshop/history/000013-bringing-in-some-command-nvim-setting-in.md
workshop/history/000078-go-stateful-shell-glue-close-review.md
workshop/history/000069-cmux-title-respawn-ownership.md
workshop/history/000050-pair-continue-plan.md
workshop/history/000049-spell-popup-digit-pick.md
workshop/history/000067-codex-tag-quit-resume.md
workshop/targets/review-protocol.md
cmd/internal/slugcmd/slug.go
cmd/internal/slugcmd/slugcmd_test.go
cmd/internal/slugcmd/slugcmd.go
cmd/internal/slugcmd/slug_test.go
cmd/internal/codexsid/codexsid_test.go
cmd/internal/codexsid/codexsid.go
cmd/internal/wrapcmd/translate_test.go
cmd/internal/wrapcmd/keymap_registry_test.go
cmd/internal/wrapcmd/extract_fg_test.go
cmd/internal/wrapcmd/wrap.go
cmd/internal/wrapcmd/osc_test.go
cmd/internal/wrapcmd/wrap_events_test.go
cmd/internal/wrapcmd/stdout_filter_test.go
cmd/internal/wrapcmd/stdout_batch_test.go
cmd/internal/wrapcmd/run_test.go
cmd/internal/wrapcmd/time_event_test.go
cmd/internal/wrapcmd/adapt_drift_test.go
cmd/internal/wrapcmd/translate_stdin_test.go
cmd/internal/wrapcmd/slug_spawn_test.go
cmd/internal/wrapcmd/overlay_test.go
cmd/internal/wrapcmd/picker_overlay_test.go
cmd/internal/wrapcmd/update_agent_output_test.go
cmd/internal/transcript/transcript_test.go
cmd/internal/transcript/transcript.go
workshop/history/000089-review-mode-should-disable-edit-while-agent-update-the-doc-m1-review.md
workshop/history/000022-rename-tag.md
workshop/history/000007-drop-variant-positional-arg.md
workshop/history/000104-single-pair-binary-plan.md
workshop/history/000090-self-contained-pair-binary-with-embedded-runtime-assets-plan.md
workshop/history/000047-doctor-stale-binary.md
workshop/history/000033-sdlc-push-archive-retry-is-not-idempotent.md
workshop/history/000011-outer-tty-passthrough.md
workshop/history/000101-review-nvim-search-should-be-case-insensitive-smartcase-close-review.md
workshop/history/000095-native-nvim-zellij-assets-close-review.md
workshop/history/000044-fix-mouse-click-selection-in-completion-popup-menu.md
workshop/history/000100-copy-on-select-paste-intermittently-dropped-orchestration-runs-inside-zellij-copy-command-hook-and-gets-reaped-close-review.md
workshop/history/000060-make-test-hang.md
workshop/history/000098-context-window-count-on-agent-pane-stuck-e-g-at-151k.md
cmd/internal/opener/opener_test.go
cmd/internal/opener/run_test.go
cmd/internal/opener/opener.go
cmd/internal/opener/runtime.go
cmd/internal/opener/run.go
cmd/internal/opener/runcli.go
cmd/internal/osfs/osfs.go
cmd/internal/osfs/osfs_test.go
cmd/internal/sessionwatch/runcli_test.go
cmd/internal/sessionwatch/sessionwatch.go
cmd/internal/sessionwatch/run_test.go
cmd/internal/sessionwatch/sessionwatch_test.go
cmd/internal/sessionwatch/runtime.go
cmd/internal/sessionwatch/run.go

exec
/bin/zsh -lc 'git log --oneline --decorate -8' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
cf9455f (HEAD -> 000113-review-footnote-question-boundary) #113: record review footnote boundary fix
473bda6 (main) #113: plan review footnote question boundary
d80a22b (origin/main, origin/HEAD) issue-sync: update issues
779bd86 issue-sync: update issues
67bff18 archive completed issues to history
94b9011 Merge pull request #78 from xianxu/000112-review-pane-inline-definitions
c99e1e9 (origin/000112-review-pane-inline-definitions) #112: close review definition work
c44b22c #112: track pending definition selections

codex
The in-repo sweep shows `pair` has footnote-definition helpers, but this review window does not modify them or add the requested regression tests. I’m getting line numbers now so the findings can point to the exact tracker claims that are unsupported by the reviewed diff.
exec
/bin/zsh -lc "nl -ba nvim/review/define.lua | sed -n '60,125p'" in /Users/xianxu/workspace/pair
exec
/bin/zsh -lc "nl -ba workshop/issues/000113-review-footnote-question-boundary.md | sed -n '1,130p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
     1	---
     2	id: 000113
     3	status: working
     4	deps: []
     5	github_issue:
     6	created: 2026-07-08
     7	updated: 2026-07-08
     8	estimate_hours: 0.81
     9	started: 2026-07-08T22:41:44-07:00
    10	---
    11	
    12	# review footnotes should not join last question
    13	
    14	## Problem
    15	
    16	When the review pane sends a final `💬:` question at the end of a document that
    17	is followed by a Markdown footnote block, the question extraction includes the
    18	footnote definitions. The agent response is then inserted after the footnotes
    19	instead of above them. This makes durable review definitions interfere with the
    20	ordinary question/answer workflow.
    21	
    22	## Spec
    23	
    24	For review question submission:
    25	
    26	- A line beginning with a Markdown footnote definition, matching
    27	  `^%[%^[^%]]+%]:`, starts a footnote section.
    28	- The final question body stops before the first such footnote-definition line.
    29	- Agent answers for that question are inserted above the managed footnote
    30	  footer. If a `---` divider immediately precedes the footnote definitions,
    31	  keep it with the footer instead of splitting the footer.
    32	- Existing behavior is preserved when no footnote-definition line follows the
    33	  final question.
    34	
    35	ARCH-PURPOSE: definitions are durable document metadata; they must not become
    36	part of the user's last question or push the answer below the metadata.
    37	ARCH-DRY/ARCH-PURE: put the boundary predicate in the pure review/question
    38	helper that already computes the last question range, rather than duplicating
    39	footnote detection in the UI shell.
    40	
    41	## Done when
    42	
    43	- A regression reproduces `💬:` followed by `---` and `[^acos]: ...`, proving
    44	  only the question text is submitted.
    45	- A regression proves the answer insertion point is above the managed footnote
    46	  footer.
    47	- Existing review question tests still pass.
    48	
    49	## Estimate
    50	
    51	```estimate
    52	model: estimate-logic-v3.1
    53	familiarity: 1.0
    54	item: lua-neovim design=0.20 impl=0.40
    55	item: milestone-review design=0.00 impl=0.15
    56	total: 0.81
    57	```
    58	
    59	## Plan
    60	
    61	- [x] Find the pure/helper code that extracts the final review question and
    62	      chooses the answer insertion point.
    63	- [x] Add a failing regression for a final `💬:` followed by Markdown footnotes.
    64	- [x] Teach the range/insertion calculation to treat a leading `[^...]:`
    65	      footnote definition as the trailing metadata boundary.
    66	- [x] Run focused review/question tests and close #113.
    67	
    68	## Log
    69	
    70	### 2026-07-08
    71	- Created from reported bug: final review question followed by definition
    72	  footnotes gets submitted together with the footnotes, and the answer is
    73	  inserted below the footnote block.
    74	- Root cause is in peer `../parley.nvim`: `chat_parser.parse_chat` finalized a
    75	  trailing open `💬:` question at EOF, so the exchange model counted the
    76	  managed footnote footer as part of the question and inserted the answer after
    77	  it.
    78	- Implemented in `../parley.nvim/lua/parley/chat_parser.lua`: a final open
    79	  question now treats a trailing column-1 `[^...]:` footnote block as metadata,
    80	  and keeps an immediately preceding `---` divider with that footer.
    81	- Added regressions in `../parley.nvim/tests/unit/parse_chat_spec.lua` for
    82	  submitted question content and model insertion point.
    83	- Verified with `nvim --headless --noplugin -u tests/minimal_init.vim -c
    84	  "PlenaryBustedFile tests/unit/parse_chat_spec.lua"`,
    85	  `nvim --headless --noplugin -u tests/minimal_init.vim -c
    86	  "PlenaryBustedFile tests/unit/build_messages_spec.lua"`, and `make test` in
    87	  `../parley.nvim`.

 succeeded in 0ms:
    60	  end
    61	  if #lines > 1 and lines[#lines] == '' then table.remove(lines) end
    62	  return lines
    63	end
    64	
    65	local function is_divider(line)
    66	  return trim(line) == '---'
    67	end
    68	
    69	local function is_footnote_line(line)
    70	  return trim(line):match('^%[%^[^%]]+%]:') ~= nil
    71	end
    72	
    73	local function managed_footer_start(lines)
    74	  for i = #lines, 1, -1 do
    75	    if is_divider(lines[i]) then
    76	      if trim(lines[i + 1] or '') ~= '' then return nil end
    77	      local has_footnote = false
    78	      for j = i + 2, #lines do
    79	        local line = lines[j] or ''
    80	        if trim(line) ~= '' then
    81	          if not is_footnote_line(line) then return nil end
    82	          has_footnote = true
    83	        end
    84	      end
    85	      if has_footnote then return i end
    86	      return nil
    87	    end
    88	  end
    89	  return nil
    90	end
    91	
    92	local function parse_footnote_line(line)
    93	  local id, definition = trim(line):match('^%[%^([^%]]+)%]:%s*(.-)%s*$')
    94	  if not id then return nil end
    95	  definition = trim(definition)
    96	  if definition == '' then definition = '(no definition)' end
    97	  return id, definition
    98	end
    99	
   100	function M.strip_definition_footnote_footer(text)
   101	  local lines = split_text_lines(text or '')
   102	  local start = managed_footer_start(lines)
   103	  if not start then return text or '' end
   104	  while start > 1 and trim(lines[start - 1]) == '' do start = start - 1 end
   105	  local kept = {}
   106	  for i = 1, start - 1 do kept[#kept + 1] = lines[i] end
   107	  while #kept > 0 and trim(kept[#kept]) == '' do table.remove(kept) end
   108	  return table.concat(kept, '\n')
   109	end
   110	
   111	local function replace_or_append_footnote(lines, id, definition)
   112	  local out = copy_lines(lines)
   113	  local footer = managed_footer_start(out)
   114	  local footnote_line = M.format_footnote_line(id, definition)
   115	  if footer then
   116	    local escaped_id = id:gsub('([^%w])', '%%%1')
   117	    for i = footer + 1, #out do
   118	      if trim(out[i]):match('^%[%^' .. escaped_id .. '%]:') then
   119	        out[i] = footnote_line
   120	        return out
   121	      end
   122	    end
   123	    out[#out + 1] = footnote_line
   124	    return out
   125	  end

exec
/bin/zsh -lc "nl -ba nvim/review/define_test.lua | sed -n '70,100p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
    70	  '',
    71	  '[^asin]: Updated.',
    72	}, 'redefining updates the footer without duplicating the inline reference')
    73	
    74	local ordinary = table.concat({
    75	  'main text',
    76	  '',
    77	  '---',
    78	  '',
    79	  'not a footnote',
    80	}, '\n')
    81	eq(define.strip_definition_footnote_footer(ordinary), ordinary,
    82	  'strip_definition_footnote_footer preserves ordinary trailing divider prose')
    83	local ordinary_footnotes = table.concat({
    84	  'main text[^source]',
    85	  '',
    86	  '---',
    87	  '[^source]: ordinary existing footnote',
    88	}, '\n')
    89	eq(define.strip_definition_footnote_footer(ordinary_footnotes), ordinary_footnotes,
    90	  'strip_definition_footnote_footer preserves ordinary divider footnotes without managed blank')
    91	eq(define.strip_definition_footnote_footer(table.concat(redefined.lines, '\n')),
    92	  'here is ASIN[^asin] in context',
    93	  'strip_definition_footnote_footer removes only final managed footnote footer')
    94	
    95	eq(define.footnote_diagnostics(redefined.lines), {
    96	  {
    97	    id = 'asin',
    98	    term = 'ASIN',
    99	    definition = 'Updated.',
   100	    line = 0,

exec
/bin/zsh -lc 'git diff --check 473bda6f47777b6ae8d80a8ac2b032a4ec4ebbd5..HEAD' in /Users/xianxu/workspace/pair
 succeeded in 0ms:
codex
The tracker line numbers confirm the plan is checked off based on peer-repo work, while the `pair` diff has no implementation or test evidence. I’ll run the nearest in-repo footnote test once to separate “existing pair footnote helper is healthy” from “this issue’s requested question-boundary change is absent here.”
exec
/bin/zsh -lc "make -n test | sed -n '1,160p'" in /Users/xianxu/workspace/pair
 succeeded in 0ms:
bash tests/run-headless-test.sh
nvim -l nvim/slug_test.lua
nvim -l nvim/scrollback_test.lua
nvim -l nvim/annotate_test.lua
nvim -l nvim/changelog_test.lua
nvim -l nvim/adapt_test.lua
nvim -l nvim/doctor_test.lua
nvim -l nvim/review/record_test.lua
nvim -l nvim/review/reconstruct_test.lua
nvim -l nvim/review/define_test.lua
nvim -l nvim/review/markers_test.lua
nvim -l nvim/review/seam_test.lua
nvim -l nvim/review/mode_test.lua
nvim -l nvim/review/poke_bodies_test.lua
nvim -l nvim/review/wrap_test.lua
nvim -l nvim/review/readiness_test.lua
nvim -l nvim/review/spinner_test.lua
nvim -l nvim/review/resolve_test.lua
nvim -l nvim/review/menu_test.lua
nvim -l nvim/review/reconcile_test.lua
nvim -l nvim/review/gate_test.lua
bash tests/queue-send-test.sh
bash tests/autopair-test.sh
bash tests/cr-newline-test.sh
bash tests/statusline-pos-test.sh
bash tests/changelog-notify-test.sh
bash doctor/doctor_test.sh
bash tests/adapt-schema-test.sh
bash tests/dev-rebuild-test.sh
bash tests/pair-go-install-layout-test.sh
bash tests/pair-embedded-runtime-test.sh
go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime
bash tests/pair-restart-quit-test.sh
bash tests/emitter-health-test.sh
bash tests/pair-session-watch-test.sh
sh tests/changelog-open-test.sh
sh tests/changelog-session-key-test.sh
sh tests/scrollback-open-test.sh
bash tests/copy-on-select-test.sh
go test ./cmd/internal/titlepoller ./cmd/internal/procutil -count=1
bash tests/zellij-trace-test.sh
bash tests/review-docflow-test.sh
bash tests/review-apply-test.sh
bash tests/review-reconcile-test.sh
bash tests/review-handoff-test.sh
bash tests/review-loop-test.sh
bash tests/review-projection-test.sh
bash tests/review-poke-test.sh
bash tests/review-window-test.sh
bash tests/review-toggle-test.sh
bash tests/pair-review-target-test.sh
bash tests/review-readiness-cli-test.sh
bash tests/review-indicator-test.sh
bash tests/review-resume-test.sh
bash tests/review-definition-test.sh
go test ./... -count=1

codex
The script is not executable, but `make test` invokes it via `bash`; I’m rerunning it the same way the project test target does.
exec
/bin/zsh -lc 'bash tests/review-definition-test.sh' in /Users/xianxu/workspace/pair
 succeeded in 1023ms:
--- results ---
ok visual definition normalizes 1-based inclusive marks to selected term
ok visual result applies
ok visual definition inserts ref immediately after selected term
ok shifted pending result applies
ok pending definition follows inserted text before selection
ok request helper returns request metadata
ok request json records id and term
ok request context is current document text
ok result applies
ok definition persisted as managed footnote
ok definition highlight spans only term plus footnote ref
ok definition diagnostic uses exact span and stored definition
ok undo removes definition text
ok undo clears definition highlights through projection
ok undo clears definition diagnostics through projection
ok redo restores definition text
ok redo restores definition highlight through projection
ok definition request context strips managed footnote footer
ok redefinition applies
ok redefinition updates existing footnote without duplicate ref
ok clear_all removes definition highlights before rehydrate
ok rehydrate_definitions redraws exact span from durable footnote
ok continued-review context artifact strips managed definition footer
definition_test ok
review-definition-test ok

codex
```verdict
verdict: REWORK
confidence: high
```

The reviewed `pair` window only changes the tracker file; it does not include the code or regression tests required by the issue’s Spec/Done-when. The issue log says the implementation lives in sibling `../parley.nvim`, but this boundary is for the `pair` repo, so the claimed completion cannot be verified here and does not satisfy the reviewed repo’s issue.

1. Strengths:
- The issue clearly states the bug and intended boundary behavior in [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:24).
- Existing `pair` managed-footnote footer logic is pure and covered separately; `bash tests/review-definition-test.sh` passed.

2. Critical findings:
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:41): Done-when requires regressions for `💬:` plus `---`/`[^acos]:` and insertion above the footer, but the `pair` diff contains no code or tests, only tracker edits. This fails ARCH-PURPOSE: the boundary claims delivery while the reviewed repo does not deliver the purpose. Fix: either implement and test the behavior in `pair`, or do not close this `pair` issue as the delivery vehicle for peer-repo work.
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:61): Plan items are checked off based on `../parley.nvim` work, outside the reviewed repo/window. Fix: replace these claims with in-repo evidence or move the close/review to the repo containing the actual changes.

3. Important findings:
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:29): The Spec/Done-when were revised from “above first footnote definition with a blank line” to “above managed footer, keeping `---` with footer,” but there is no `## Revisions` section. Fix: append a dated revision entry explaining the contract change and why.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `bash tests/review-definition-test.sh`: passed.
- Did not run full `make test`; the blocking issue is absence of in-window implementation/tests for #113, not a failing in-repo test run.

6. Architectural notes for upcoming work:
- ARCH-DRY: pass for the reviewed diff; no duplicated code was introduced.
- ARCH-PURE: not verifiable for the claimed change because no `pair` implementation exists in the diff.
- ARCH-PURPOSE: fail; the issue’s purpose is deferred to an unreviewed sibling repo.

7. Plan revision recommendations:
- Add `## Revisions` with a `2026-07-08` entry: reason “footer boundary clarified after implementation”; delta “answer insertion target changed from first footnote-definition line plus blank separation to managed footer boundary, keeping immediately preceding `---` with footer.”
- If work remains outside `pair`, revise the Plan/Log to stop claiming `pair` delivery, or create/close the issue in the repo that contains `chat_parser.lua` and its tests.
tokens used
41,995
```verdict
verdict: REWORK
confidence: high
```

The reviewed `pair` window only changes the tracker file; it does not include the code or regression tests required by the issue’s Spec/Done-when. The issue log says the implementation lives in sibling `../parley.nvim`, but this boundary is for the `pair` repo, so the claimed completion cannot be verified here and does not satisfy the reviewed repo’s issue.

1. Strengths:
- The issue clearly states the bug and intended boundary behavior in [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:24).
- Existing `pair` managed-footnote footer logic is pure and covered separately; `bash tests/review-definition-test.sh` passed.

2. Critical findings:
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:41): Done-when requires regressions for `💬:` plus `---`/`[^acos]:` and insertion above the footer, but the `pair` diff contains no code or tests, only tracker edits. This fails ARCH-PURPOSE: the boundary claims delivery while the reviewed repo does not deliver the purpose. Fix: either implement and test the behavior in `pair`, or do not close this `pair` issue as the delivery vehicle for peer-repo work.
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:61): Plan items are checked off based on `../parley.nvim` work, outside the reviewed repo/window. Fix: replace these claims with in-repo evidence or move the close/review to the repo containing the actual changes.

3. Important findings:
- [workshop/issues/000113-review-footnote-question-boundary.md](/Users/xianxu/workspace/pair/workshop/issues/000113-review-footnote-question-boundary.md:29): The Spec/Done-when were revised from “above first footnote definition with a blank line” to “above managed footer, keeping `---` with footer,” but there is no `## Revisions` section. Fix: append a dated revision entry explaining the contract change and why.

4. Minor findings:
- None.

5. Test coverage notes:
- Ran `bash tests/review-definition-test.sh`: passed.
- Did not run full `make test`; the blocking issue is absence of in-window implementation/tests for #113, not a failing in-repo test run.

6. Architectural notes for upcoming work:
- ARCH-DRY: pass for the reviewed diff; no duplicated code was introduced.
- ARCH-PURE: not verifiable for the claimed change because no `pair` implementation exists in the diff.
- ARCH-PURPOSE: fail; the issue’s purpose is deferred to an unreviewed sibling repo.

7. Plan revision recommendations:
- Add `## Revisions` with a `2026-07-08` entry: reason “footer boundary clarified after implementation”; delta “answer insertion target changed from first footnote-definition line plus blank separation to managed footer boundary, keeping immediately preceding `---` with footer.”
- If work remains outside `pair`, revise the Plan/Log to stop claiming `pair` delivery, or create/close the issue in the repo that contains `chat_parser.lua` and its tests.
