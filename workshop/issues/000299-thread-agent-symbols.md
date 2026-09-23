---
id: 000299
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Show coding-agent symbols on thread rows

## Problem

Thread listings currently show repository and lifecycle state but do not give a
compact visual indication of which coding agent owns the thread. When several
threads are open, this makes it harder to distinguish Claude, Codex, Muse,
Agy, and Qoder at a glance.

## Spec

Add an agent badge as a suffix to every repository/thread label that already
exposes the current agent. For example, render `brainⓞ` and
`parley.nvimⓠ`; the badge reflects the agent currently owning that thread.
Use these symbols and meanings:

| Agent | Badge | Meaning |
| --- | --- | --- |
| `claude` | `ⓐ` | Anthropic |
| `codex` | `ⓞ` | OpenAI |
| `muse` | `ⓕ` | Facebook |
| `agy` | `ⓖ` | Google |
| `qoder` | `ⓠ` | Qoder |

Keep the agent name available anywhere it is needed for disambiguation or
accessibility. Unknown or future agents must retain a readable fallback rather
than being mislabeled; the badge mapping should have one canonical owner so
all thread views render the same symbol.

## Done when

- The five mappings above render consistently in every thread/session listing
  that displays an agent.
- Existing selection, filtering, sorting, status labels, and keyboard behavior
  remain unchanged.
- Unknown agent names remain identifiable and do not inherit a wrong badge.
- Unit or acceptance tests cover all five mappings, fallback behavior, and at
  least one full thread-list rendering path.
- User-facing help or documentation explains the badges if the list has a
  legend or help surface.

## Plan

- [ ] Inventory thread-list renderers and identify the canonical agent field
      and shared formatting boundary.
- [ ] Add the centralized badge mapping and render it without removing the
      existing agent identity.
- [ ] Add mapping, fallback, and integration coverage; update the relevant
      help/README surface.

## Log

### 2026-09-20

Created with the requested Anthropic/OpenAI/Facebook/Google/Qoder badge map.

## Revisions

### 2026-09-20 — Badge placement clarified

The symbol is a suffix on the displayed repository/thread label, not a
replacement for the agent name. Examples: `brainⓞ` for a Codex thread and
`parley.nvimⓠ` for a Qoder thread. The suffix reflects the agent currently
owning that thread.

### 2026-09-21 — color ownership; defer inline model strength

Extend the badge with a stable per-agent color, applying the color to the
symbol rather than changing the label grammar. Keep the agent name available
for accessibility and unknown-agent fallback. Color answers “which agent is
this?” at a glance without adding another character.

Model strength is a distinct, optional dimension introduced by couch slots.
Possible punctuation (`ⓞ` fast, `ⓞ:` balanced, `ⓞ!` top) is compact but quickly
becomes hard to scan when combined with slot labels, for example
`brainⓞ. :1! :2:`. Do not commit to that inline encoding yet. First expose the
effective harness/model in secondary metadata such as a detail view, tooltip,
or legend; promote it into the primary row only if real usage shows that the
extra information is worth the density.

The agent badge and model-strength presentation must remain separate
derivations: changing a model profile must not change thread identity, sorting,
or the canonical agent symbol.
