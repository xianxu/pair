---
id: 000823
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '99292d6c1143f2978a41ac2d63b9f1cbf27d1f29' # card fields mirrored from issue-cards; edit via sdlc
---

# unified-editorial-interface: Couch parses, protects and enforces per-section modes (heading comments, aliases, migration)

## Problem

## Spec

**Settled with the operator (10-22, design chat in brain `session.side/2026-10/2026-10-22-01-editorial-modes.md`):**
- **Uniform names:** `collaboration-mode` (where an answer goes: discussion / collaboration / chat),
  `presentation-mode` (how edits are shown: highlight / direct / propose), `processing-mode` (how an edit is made:
  the review modes in `ariadne/construct/local/universal-review/modes/`: copy-editing, developmental, fact-check,
  free-form, line-editing, proofreading; unique prefixes allowed). The old keys `presentation` and
  `process-instruction` stay readable as aliases. `process-instruction`'s free text splits out into
  `processing-instruction`.
- **Per section:** all three can be set for one section by an HTML comment directly under its heading:
  `<!-- collaboration-mode: chat; processing-mode: copy-editing -->`. Couch protects that line as operator-owned;
  parley conceals it as a badge.
- **Frontmatter:** each key is one value covering the whole document; the per-section map form
  (`collaboration-mode: {chat: ["Operator instructions"]}`) is retired.
- **Precedence:** section comment, then document frontmatter, then the built-in default.
- **Typeahead (required):** on a line next to a heading, `<!-- ` offers the three keys, `key: ` offers that key's
  values, `; ` offers the keys not yet used, and accepting the first key inserts the closing ` -->`. Values come from
  the modules that enforce them (extending parley's `lua/parley/frontmatter_enum.lua`), never a second list.


Project `unified-editorial-interface`; implements the spec in ariadne (dependency). In Couch:
- **Parse** the three keys (and the aliases) from frontmatter as single values, and the per-section heading comments,
  resolving each section's effective modes by the precedence; read legacy per-section maps with a deprecation warning.
- **Protect** the heading-comment line as operator-owned, like the operator-owned frontmatter keys: an agent edit
  that adds, removes or changes it is refused; the operator's edits pass.
- **Enforce** the effective per-section modes in the review round (collaboration routing, presentation of edits,
  processing mode), section by section.
- **Migrate** existing documents' per-section maps (e.g. `ops-echo/session.md`) to heading comments, as an
  operator-run step, since the keys are operator-owned.


**Side-doc layout (decided 10-22):** `<doc>.side/YYYY-MM/YYYY-MM-DD-NN-slug.md`: a year-month subfolder, with the full date kept in the filename so a file still makes sense when linked, grepped or moved. **No migration:** existing flat names stay valid (readers accept both). Reason: `ops/session.side/` already holds about 270 entries, 261 from October alone. For this issue: make this repo's side-doc creator use the new layout and its readers accept both. Applies to the naming rule in the universal-review spec (`collaboration-modes.md` §4) and to every side-doc creator (parley, Couch, and tl-lead's side-doc helpers).

## Done when

- A document with frontmatter defaults and two sections overriding different keys resolves each section's modes
  correctly (test), aliases included.
- An agent edit to a heading comment is refused; an operator edit passes (test).
- A round applies chat in one section and collaboration in another in the same document (test).
- The migration converts a per-section map to heading comments exactly (test).

## Plan

- [ ]

## Log

### 2026-10-22
