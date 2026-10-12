---
id: 000469
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'db8f651d138b74e27d2bb177ef4871b5bc86dc2a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T02:35:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "5d5f1410", done: "5a291be4"}
---

# notes: focused-context text includes script bodies and runs adjacent blocks together

## Problem

pair#462 (agent queries), switching onto pair#461's `notes.Structural`, found two defects in `Context.Text` (golden: `cmd/internal/webcast/query/testdata/golden/app.notes.txt`): (1) the landmark text includes `<script>` bodies (app fixture, note app-n3: the text ends with `const r=document.getElementById…`); (2) adjacent blocks run together with no separator between text nodes (`tasksAddShip onboarding flowFix billing`). An agent reading a note's context gets code and run-on words instead of what the user saw.

## Spec

Context text is the visible text a person would read: skip `script`, `style`, `template`, `noscript` and other non-rendered content (and hidden elements where the recording says so); separate block-level boundaries (and `br`, list items, table cells) with whitespace, collapse runs of whitespace the way rendering does. Keep the 500-character window. Regenerate #462's golden output after the fix.

## Done when

- The app fixture's note app-n3 context has no script text, and block boundaries are separated, in a test.
- `notes.RunModelContract` and #462's goldens pass with the corrected text.

## Plan

- [x] `recording.DOMNode.RenderedText()` beside `TextContent` (ARCH-DRY: one walker both models use): skip
  non-rendered tags (script, style, template, noscript, head metadata) and elements the recording marks hidden
  (`hidden` attribute, inline `display:none`/`visibility:hidden`, Tailwind `hidden` with no display variant);
  space-separate HTML-default blocks, table cells, list items, `br`, and the children of a Tailwind flex/grid
  container (blockified); collapse whitespace.
- [x] `Structural.Context` and `FakeModel.Context` take landmark and focus text from it; the 500-rune window stays.
- [x] Model contract: the contract page's article carries a script whose body names the anchor's text, and the
  context must hold no script and keep the heading apart from the paragraph. Mutation-checked: either model
  reverted to `TextContent` fails it.
- [x] Unit table `TestRenderedText`; fixture test `TestContextOfAppReportsIsRenderedText` (app-n3).
- [ ] Merge main once #462 lands; regenerate its goldens (`app.notes.txt` et al.), check they read cleanly.

## Log

### 2026-10-12

- Before: app-n3's context was `Reports Week 10 tasksWeek 2137 tasks…const r=document.getElementById('r');…`.
  After: `Reports Week 1 0 tasks Week 2 137 tasks …`, no script.
- Out of scope, noted: `NoteAt.Text` (the anchor's text, stored in recordings and compared in `Resolve`) still
  uses `TextContent`. Changing it would change stored anchors and their matching; Context is what agents read.
