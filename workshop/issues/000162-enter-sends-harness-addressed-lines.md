---
id: 000162
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Enter sends when the draft line is addressed to the harness

## Problem

Sending `/login` from the draft costs Alt+Enter. So does `/clear`, `/model
opus`, `/context`, `/resume`. These are the shortest and most frequent things
typed into the draft, and every one of them pays the two-key send.

The current binding is deliberate, not an oversight:

- `nvim/init.lua:3565` — insert-mode `<CR>` goes through `cr_keys`, whose
  documented contract is "`<CR>` in the draft must ALWAYS break the line when
  the user hasn't picked a completion".
- `nvim/init.lua:3474` — `<M-CR>` is `send_and_clear`; `<S-M-CR>` appends
  without submitting.
- `cmd/internal/wrapcmd/wrap.go:127-143` — pair-wrap's `sendKeymap` pushes the
  *same* inversion down into the agent pane (`PAIR_WRAP_REMAP_RETURN`), for the
  stated reason that the pane-to-pane mismatch is jarring.

So "Enter = newline, Alt+Enter = send" is a cross-pane invariant the workbench
went out of its way to establish. The problem is not that the invariant is
wrong — it is right for prose — but that it is applied to text where a newline
has no meaning at all. There is no two-line `/login`.

## Spec

### The rule

**Enter sends when the line is addressed to the harness. Enter inserts a
newline when the text is addressed to the model.**

That is the general rule, and `/` is only its first instance. Prose in the
draft is addressed to the model, and prose is inherently *growable* — a second
line is always a meaningful thing to want, so Enter must stay a newline.
A sigil-led line is addressed to the *harness*, whose grammar is
line-terminated by construction — the utterance is complete the moment it is
typed, so Enter can commit it. The binding follows the addressee, not a
hardcoded character.

This is worth stating as a rule rather than a special case because it
generalizes without further invention (see "Later, same predicate" below), and
because it tells you what does *not* qualify: `@file.go` is a file reference
*inside* prose, addressed to the model, so it stays on today's binding.

### The predicate

`draft_is_harness_addressed(buffer, agent)` is true when all three hold:

1. The buffer is exactly one line (no embedded newline).
2. Its first character is a **line-committed sigil** for this harness.
3. The line matches that sigil's token grammar — for `/`:
   `^/[A-Za-z][A-Za-z0-9_-]*(\s.*)?$`.

Rule 3 is what keeps the rule safe, and it is the reason the predicate is not
just "starts with `/`". `/Users/xianxu/workspace/brain — look at this repo` is
a completely ordinary opening line of a prompt in this workflow, and it starts
with a slash. An inner `/` in the first token disqualifies it, as does a space
straight after the sigil. A misfire in the other direction is cheap: the worst
case is a single line of command-shaped text reaching the agent, which answers
"unknown command".

Sigils are per-harness data, not a constant — `!` (bash) and `#` (memory) are
Claude-specific, Codex has only `/`. The natural home is beside the existing
per-agent tables in `wrapcmd` (`endOfTurnByAgent`, `spanExtractionAgents`,
`sendKeymap`) rather than a literal in `init.lua`, so bringing up a new harness
declares its sigils in one place — cf.
`atlas/how-to-bring-up-a-new-harness-cli.md`.

### Invariants this must not break

- **Alt+Enter always sends, in every mode.** The rule is strictly *additive*:
  Enter gains a meaning, nothing loses one. No swap, no inversion — a user
  whose fingers already send with Alt+Enter never gets a surprise newline.
- **Predicate false → today's behavior, byte for byte.** Including the whole
  `cr_keys` completion-popup decision table, which stays the outer gate: a
  visible popup still owns `<CR>` first, and only the no-popup branch consults
  the new predicate.
- **A literal-newline escape hatch that ignores the mode** (`<C-j>` is free and
  is already "insert newline" in stock insert mode). Every gate in this fleet
  gets a bypass; this is that gate's.
- **The mode is visible before it fires.** The draft should say so — a winbar
  or statusline hint (`⏎ send`) while the predicate holds. A silent modal Enter
  is the failure mode worth spending a line of UI on.

### Scope

Draft pane only. The agent pane's half of the invariant (pair-wrap's
`sendKeymap` rewriting `\r`) is deliberately **out of scope**: byte-level stdin
rewriting has no view of the composer's contents, and the workflow this issue
is about — typing into the draft — never needs it. That leaves a knowing
asymmetry (Enter on `/login` sends in the draft, still newlines in the agent
pane); record it, don't fix it here.

### Later, same predicate

Named so the rule reads as a rule, not scoped in:

- A numbered menu answer (`1`, `2`) or `y`/`n` while the agent is showing a
  picker is the same shape — a closed utterance addressed to the harness. It
  needs agent *state*, not buffer content, which `wrapcmd`'s `overlayDetector`
  and the composer recognizers already reach toward. Different input to the
  same predicate.

## Done when

- Typing `/login` in the draft and pressing Enter sends it, with no Alt.
- `/model opus` — sigil plus arguments — sends on Enter.
- `/Users/xianxu/workspace/brain is the repo` gets a newline on Enter, not a
  send. So does `/ ` followed by prose, and so does any buffer already
  containing a newline.
- Alt+Enter still sends in every one of the cases above, including the ones
  where Enter now also sends.
- `<C-j>` inserts a literal newline even while the predicate holds.
- The completion-popup decision table in `tests/cr-newline-test.sh` still
  passes unchanged — the popup keeps first claim on `<CR>`.
- The draft shows the mode while the predicate holds.
- The sigil set is per-harness data, and a harness with no entry behaves
  exactly as it does today.

## Plan

- [ ]

## Log

### 2026-09-01

- Filed from a brain session. The operator's ask was the narrow one ("`/login`
  should send on a single Enter"); the issue generalizes it to the
  addressee rule because the narrow version would have hardcoded `/` in
  `init.lua` next to a comment that says `<CR>` must *always* insert a newline.
- Key finding while scoping: the Enter/Alt+Enter inversion is not local to the
  draft. `wrapcmd`'s `sendKeymap` (`wrap.go:127`) pushes it into the agent pane
  on purpose, for cross-pane consistency. Any exception is therefore an
  exception to a stated invariant and has to say what it does about the other
  pane — hence the explicit out-of-scope paragraph rather than silence.
- `/Users/...` as a legitimate slash-leading prose opener is the case that
  forced the token grammar in predicate rule 3; "first char is `/`" would have
  mis-sent it, and in this operator's workflow that line is common.
