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

Add an agent badge to every thread row that already exposes the current agent.
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
