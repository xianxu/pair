---
id: 000462
status: working
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours:
card_mirror: '8c5ca8c4bc38c92712ffdcbd347d7bd5b412e1b0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T01:55:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "825a7141", done: "03943cf9"}
---

# broadcast-web: agent queries — DOM at t, events in a window, notes with context

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's query surface as a CLI agents use on a recording (built on the reference reducer):
- The DOM at time t, optionally under a selector, as compact readable text (not raw JSON by default; `--json` for structure).
- Events between t1 and t2 (navigation, input, scroll, DOM changes summarized).
- Every note with its element, the DOM around it at that time (the notes model's focused context, pair#461, through its #457 interface/fake until it lands), and the user actions in the N seconds before it.
- Command shape and name fit Couch's CLI conventions (`couch --help`); outputs sized for an agent's context.

## Done when

- The three queries run on every #457 fixture with golden-output tests.
- An agent given only a fixture recording and the CLI answers what a fixture note refers to (element, time, preceding action): a scripted check, or a recorded manual run in the Log.

## Plan

Durable plan: `workshop/plans/000462-broadcast-web-agent-queries-dom-at-t-events-in-a-window-notes-with-context-plan.md`. Single pass, one close.

- [x] Querier (real `query.Surface`) passes `RunSurfaceContract`
- [x] Selector subset + `@id`
- [x] Compact page text (state overlay, budgets)
- [x] Event lines (DOM runs summarized, `by` shown)
- [x] Notes text + `query.Main` CLI (`--json`)
- [x] Golden tests on every fixture
- [x] Agent check: scripted + one recorded subagent run
- [x] `couch --broadcast-web-query` wiring, usage, skill, atlas

## Log

### 2026-10-11

### 2026-10-12
- Command: `couch --broadcast-web-query DIR dom|events|notes` (closed form in couchcmd, parsed by `query.Main`; no Couch runtime needed). Name follows the product (`couch broadcast-web`) and the `--broadcast-list` sibling.
- Notes went through `notes.FakeModel` until #461 landed mid-issue; `query.Main` now uses `notes.Structural` (plan Revisions). The golden diff showed two #461 issues in its context text, reported to the TL rather than patched here: the landmark text includes `<script>` bodies (`app` fixture, app-n3) and runs adjacent blocks' words together (`tasksAddShip`).
- Seam additions, all inside #462's own package (no format change): `query.NoteView` gains `Landmark`, `URL`, `Header`, `BeforeLines` (the real surface fills them, the fake leaves them empty); `query.UserAction` now counts `seg` and `stop`, so a note's preceding actions mark a navigation, tab switch or pause instead of folding typing across it (found in the `segments` golden: `1200-5200 input … (2 changes)` across a pause).
- Typing folds: React writes the value twice per key (the `input` property and the `value` attribute); both fold into one line per field and viewer (`4414-5116 input @73 … "Review PR 42" (19 changes)`, 19 checked against the fixture by grep). The fold key carries `by`, so two people's typing never merges.
- Fixture selectors from fixturegen (`h1~Reports`, `img[alt*=issue tracker]`) aren't CSS; notes resolve by node id so they still work, and `Select` refuses them with the grammar if passed back.
- Agent check, recorded run (2026-10-12): a fresh Sonnet subagent, told to use only the built `couch --broadcast-web-query` on a copy of the `app` fixture and not to read the files or source, ran the bare flag (usage) and then `notes --note app-n2`: two commands. It answered: element @108 `<button id=open-modal>` "Delete workspace…" (classes `bg-rose-600 text-white px-4 py-2 rounded-lg`, inside @106 `div.space-y-4.max-w-md`, box 256,92 174x40); t=12500, segment 2, Ana (design), resolved; before it, typing "Review PR 42" in `#draft`, search typed and cleared, checkboxes, a click on `#nav-settings`, scrolls, then focus and click on `#open-modal` at 10757/10758; fix: `bg-rose-700`/`800` for white-text contrast plus a focus ring. All correct against the fixture. It reported nothing confusing; the only remark was that the usage-only run exits 2 (the usage-error convention, kept). The scripted counterpart is `TestAgentCanTellWhatEachNoteRefersTo` (every note of every fixture).
- Close round 1 (FIX-THEN-SHIP, 2 Important, 5 Minor). BR-1/BR-2: `Select` read the notes model's anchors more loosely than notes defines them, and the test asserted containment. Fixed as a class: notes exports `Match` (its own reader, with whether the text is in its grammar; a small additive export in #461's file), `Select` uses it when it names something and is plain CSS otherwise; the CSS combinators are now tree-scoped (a selector starts in the top document, space and `>` stay in one tree, `>>>` enters only the host's own tree); the test asserts exactly the element wherever `Structural.Resolve` with no node id resolves the anchor uniquely, plus the review's two probes. Minors: the fake and the real surface now both report a note's final state, pinned in `RunSurfaceContract`; one `unreadable`/`checkWindow` helper; `--max-lines` is refused with `--json` (JSON answers in full); a "… N more matches" line when matches outrun the budget; control characters scrubbed at the CLI's output writer (every text path, one place); the seam changes reported to the TL. Mutation-checked: scrubber off, tree-scoping off and the notes.Match delegation off each fail their tests.
- Close round 2 (FIX-THEN-SHIP; 7 round-1 findings disposed as addressed). BR-8: the atlas restated the old selector rules by hand; it now says how selectors are read and points to `query.SelectorGrammar` (the one statement, printed by usage and refusals) instead of copying the grammar; the plan gets a Revisions line. The empty-recording gap is now a `RunSurfaceContract` case (no page, no events, no notes), which the fake failed (panic) before its guard. The `notes.Match` doc comment is back on `Match`. `TestCapturePTYBothBoundaries` timed out once under the parallel run and with `-count=3`; it passes alone on this branch and on origin/main: a load-sensitive test this diff doesn't touch.

