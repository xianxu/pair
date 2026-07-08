---
id: 000112
status: working
deps: []
github_issue:
created: 2026-07-08
updated: 2026-07-08
estimate_hours: 3.30
started: 2026-07-08T14:24:21-07:00
---

# review pane inline definitions

## Problem

Pair's review pane is a good place to read and revise unfamiliar documents, but
there is no lightweight way to ask "what does this selected phrase mean here?"
without leaving the document flow. Parley solved the same user need in
parley.nvim#161 as an inline definition, then improved it in parley.nvim#166 and
parley.nvim#167 by persisting definitions as markdown footnotes and highlighting
only the selected term/reference span.

Pair should bring that behavior to the review workbench, adapted to pair's
architecture: the review pane should not embed its own LLM client. It should use
the existing pair agent via the same poke/file-seam style that review handoffs
already use.

## Spec

In a pair review pane, visual-selecting a term and invoking the definition
gesture requests a concise definition from the existing agent, then persists the
answer in the reviewed document as a managed markdown footnote.

- The selected text remains readable and gains an inline reference:
  `term[^term]`.
- The definition is stored in a managed final footer after a standalone `---`
  divider as `[^term]: definition`.
- The managed footer is recognized only when the final `---` block contains
  blank lines and footnote definitions. Ordinary document `---` content must not
  be stripped or treated as managed state.
- Re-defining an already-footnoted term updates the matching footnote instead of
  appending duplicate inline references or duplicate footer lines.
- Opening a review document rehydrates definition diagnostics/highlights from
  the durable footnotes. The diagnostic message is derived from the stored
  footnote text.
- Visible definition highlighting covers exactly the selected text plus the
  appended `[^id]` reference, not the whole paragraph or line.
- Undo/redo projection preserves definition highlights and diagnostics along
  with existing review decorations.
- The definition request goes through pair's existing agent: nvim writes a
  request artifact and pokes the agent; the agent answers by running a
  `pair review definition` helper that writes the result artifact. No Neovim-side
  LLM dispatcher is added.
- Agent review context must exclude the managed definition footer when pair asks
  for continued document review, so definitions do not pollute later review
  prompts.

ARCH-PURE: footnote slugging, footer insertion/update/strip, diagnostics, and
selection-range transforms live in pure Lua helpers. ARCH-DRY: the same helper
drives persistence, rehydration, and prompt/footer stripping. ARCH-PURPOSE: this
is not complete if definitions are only ephemeral, if they bypass the existing
agent seam, or if the managed footer leaks into subsequent review context.

## Done when

- Visual-selecting `ASIN` and requesting a definition can persist
  `ASIN[^asin]` plus a managed `[^asin]: ...` footer through the
  `pair review definition` result seam.
- Re-defining the same term updates the existing footnote and does not duplicate
  the inline reference.
- Review-pane diagnostics/highlights are rehydrated from durable footnotes on
  open and cover only `ASIN[^asin]`.
- Undo/redo restores definition decorations with exact column spans.
- Review agent pokes/context omit the managed definition footer while preserving
  ordinary trailing `---` document content.
- Focused pure Lua, review-pane, Go CLI, and final verification pass.

## Estimate

Produced via `estimate-logic-v3.1` against the repo-local calibration source
reported by `sdlc estimate-source` (stale but canonical for this repo). Method A
only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.20 impl=0.00
item: lua-neovim design=0.75 impl=1.35
item: smaller-go-module design=0.15 impl=0.30
item: atlas-docs design=0.00 impl=0.10
item: milestone-review design=0.00 impl=0.25
design-buffer: 0.20
total: 3.30
```

## Plan

- [x] Add pure definition-footnote helpers and tests under `nvim/review/`.
- [x] Add the `pair review definition` response helper and tests.
- [x] Wire review-pane visual definition request/result handling, durable
      rendering, and projection/reopen rehydration.
- [x] Strip managed definition footers from review agent context/poke bodies.
- [x] Update atlas/runtime bundle and run focused plus full verification.

## Log

### 2026-07-08
- Close review returned REWORK for four issues: visual mark end-column handling,
  continued-review context still exposing the managed footnote footer, ignored
  `pair review definition` atomic-write errors, and missing README docs. Added
  regressions for each, fixed the visual path, wrote a stripped
  `review-context-<tag>.md` artifact into the human-finished poke, surfaced
  definition-result write failures, and documented the command plus the usable
  `Shift+Alt+d` visual binding (avoiding the existing global `Alt+d` detach).
- Second close review returned REWORK because the managed-footer recognizer also
  treated ordinary trailing `---` + Markdown footnotes as pair definitions.
  Tightened the recognizer to the exact managed shape Pair writes (`---`, blank
  line, footnotes) and added pure regressions proving ordinary divider
  footnotes are neither stripped nor rehydrated.
- Third close review returned REWORK because durable rehydration only expanded
  definition spans backward over a single word, so multi-word terms highlighted
  only the final word plus footnote ref. Reworked pure span derivation to find
  the preceding suffix whose `footnote_id` matches the ref id and added a
  multi-word phrase regression.
- Updated `atlas/review-workbench.md` for the definition keybinding,
  request/result seam, durable footnote storage, exact-span rendering, and
  projection behavior. Ran `make runtimebundle-generate`; generated runtime was
  already in sync with the source changes.
- TDD Chunks 3/4: `bash tests/review-definition-test.sh` first failed on
  missing `_G.PairReviewPane.request_definition`, then later on missing
  `rehydrate_definitions`; `nvim -l nvim/review/poke_bodies_test.lua` failed
  until the definition poke named the request artifact. Added the tag-scoped
  definition request/result seam, visual `Shift+Alt+d` request handling, result
  polling/application, exact-span definition decorations, rehydration from
  durable footnotes, and stripped request context. GREEN:
  `bash tests/review-definition-test.sh`, `nvim -l
  nvim/review/poke_bodies_test.lua`, `bash tests/review-apply-test.sh`,
  `bash tests/review-projection-test.sh`, and `make test-lua` passed.
- TDD Chunk 2: `go test ./cmd/internal/reviewcmd ./cmd/internal/dispatcher
  -count=1` first failed on missing `DefinitionOptions`, `definitionDoc`,
  `RunDefinition`, and `review definition` dispatch resolution. Added the
  `pair review definition [--term TERM] <request-id> <definition...>` result
  writer, using the existing review session priority and atomic data-dir seam.
  GREEN: `go test ./cmd/internal/reviewcmd ./cmd/internal/dispatcher -count=1`
  passed.
- TDD Chunk 1: `nvim -l nvim/review/define_test.lua` first failed because
  `nvim/review/define.lua` was missing. Added pure definition helpers for
  selection slicing, footnote id/formatting, managed footer insertion/update,
  footer stripping, and exact-span diagnostic derivation. GREEN:
  `nvim -l nvim/review/define_test.lua` and `make test-lua` passed.
- Created from user request to port the parley definition feature into pair
  review, using parley.nvim#161 for the original inline-definition interaction
  and parley.nvim#166/#167 for the durable footnote and exact-span behavior.
  Design keeps LLM access in pair's existing agent seam rather than adding a
  second Neovim-side model client (`ARCH-PURE`, `ARCH-DRY`, `ARCH-PURPOSE`).
