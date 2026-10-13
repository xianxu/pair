---
id: 000483
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'b925e352b7b3fab473044a4cf28f0a78da66c763' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T08:41:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: DOM submission — send live notes with focused context and a crop to the agent in view

## Problem

Wave 3 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`, Direction). Notes are comments in the DOM's time and space, the counterpart of 🤖[] markers in a markdown file; the operator wants them submitted to the agent in view as a "DOM submission", the integrated form of typing a message and attaching a screenshot. Everything it needs has landed: notes in the viewer (#470), the notes model with focused context and structure-snapped crops (#461, #469), agent queries (#462), replay links (#471, #477 in progress).

## Spec

### Current contract — 2026-10-12, plan proposed in pair:6

The operator's Log and the revisions below supersede the initial proposal retained afterward.
Shift+Ctrl+Alt+Return from the draft nvim submits the draft and all open notes on the cast tab at admission;
an empty draft works when notes exist. One versioned shared Go submission type is persisted as
`<couch store>/webcast/submissions/<id>/submission.json` plus one `crop-N.png` per note, including overlapping
notes. The agent receives the draft plus a short absolute manifest pointer; no clipboard or rendered-copy file.
Terminal submission (#455), ticket creation and ticket assets are separate work.

Each note includes recorded author/text/time/anchor, focused context and compact element text, replay link and
recording path. PNGs come from the cast tab now and carry capture time and document identity. A document
already gone yields explicit `crop unavailable: page changed` with the recorded context retained; navigation
during capture refuses preparation. Selection is frozen before asynchronous work, so later notes are not consumed.

Prepare the bundle, reuse the authored-draft send transaction, then acknowledge delivery to resolve exactly those
notes. Retry freezes bundle, recipient and generated agent text. A failed or uncertain send preserves the draft
and notes; a delivered message whose acknowledgment fails retries acknowledgment only, never delivery.
The durable plan below proposes concrete resource limits and store retention for TL review before building.

### Initial proposal (superseded where the operator's Log or current contract differs)

Operator-shaped (the operator shapes it in the slot); the TL's starting proposal, from the project:
- **Ways in:** a button in the notes menu (`💭N`), a hotkey, and a combined submission from the draft nvim (Shift+Ctrl+Alt+Return: the draft's text plus the pending DOM notes).
- **What goes:** per note, the text, author and time; the element and its focused context (`notes.Model.Context`); a screenshot crop of the notes' cluster, snapped to structure (`NoteCrops`, captured from the operator's tab with `Page.captureScreenshot` and a clip: pixels to the agent only); a link to the moment in replay (`#t=`); and the recording's path for `couch --broadcast-web-query`. Notes far apart become one crop per cluster.
- **To whom:** the agent in view in Couch (the slot focused), delivered like a Couch message; submitted notes resolve on every screen.
- **The wire shape:** `notes.Context` gets a JSON shape here; keep it shared with pair#455 (terminal 💭N submission) so both submissions look alike to an agent.
- **Mock loop:** the agent revises, the page reloads, the cast carries on; live notes clear on reload.

## Done when

- Current acceptance: two viewers leave notes on the blog; the single draft hotkey sends all open notes and
  optional draft text to the agent in view, with one PNG per note and readable recorded context; the agent fixes
  at least one correctly from the bundle alone. Empty draft, safe failure/retry, successful resolution on every
  screen, page-change crop behavior and cast continuation are verified. Operator smoke before close remains required.
- The older button/hotkey/combined wording below is superseded by the operator's single-hotkey decision.
- On the blog, two viewers leave notes; the operator submits them to an agent slot (blog:0) by button, by hotkey and combined with a draft; the agent fixes at least one correctly from the submission alone, without asking for the full page.
- The submission format is documented in the atlas, shared with #455's.
- The operator smoke-tested it before close.

## Plan

- [ ] Shared submission format and pure builder/transition tests.
- [ ] Bounded atomic store, note snapshot, current-tab crops and guarded resolution.
- [ ] Couch prepare/status/ack RPC with fixed recipient and replay links.
- [ ] Draft hotkey using the existing authored-send transaction with frozen retry identity.
- [ ] Docs, bounded verification, hub-scheduled operator smoke, SDLC review and SHIP landing.

Detailed plan: [DOM submission implementation plan](../plans/000483-dom-submission-plan.md).

## Log

### 2026-10-12 — plan approval in pair:6

TL approved commit `ec88640dd` and the proposed operating limits/retention (Couch receipt
`0a3761ed-b2f7-4044-8a01-fb16fabd915f`), directing change-code. Resolve submitted notes after delivery as planned;
TL is confirming that policy with the operator. Architecture/API exploration found that draft retries compare
authored text only; the implementation must freeze the generated bundle pointer too (ARCH-ORDER).

### 2026-10-12

Operator shaping (in pane):
- **Images go as PNG files**, not the clipboard: with many slots running at once, a shared clipboard gets fought over.
- **All open notes are submitted**; the operator resolves the unwanted ones first.
- **One hotkey:** Shift+Ctrl+Alt+Return from the draft nvim sends the draft and the notes. An empty draft still submits when notes exist, because the notes are content too.
- **Target:** the agent in view by default. The draft's text says what to do with it, e.g. "this demonstrates XXX#NNN, update that ticket", and the agent files a new ticket or amends one. Tickets are the durable home, publishable and workable from any slot.
- **Ticket assets (operator idea):** an issue gets a per-ticket asset folder that shares its lifecycle, as parley.nvim's chats do (`assets/<chat-timestamp>/`, keyed by identity not slug; moves and deletes follow the chat).
- pair#488 decision (operator via TL): every note carries a bounding box (drag, or a default 1/10 × 1/10 box centred on a click); the crop is that box plus a small margin. No structure snapping, no clustering in #483's crop.
- **Format (operator):** JSON is the agent-facing shape (agents read it fine). One `submission.json` (shared Go type with #455) plus `crop-N.png`, no rendered text copy.
- **Store first, tickets second (operator):** a submission is written to `<couch store>/webcast/submissions/<id>/` (an absolute path that survives slot switches); the Couch message carries the draft text plus a short pointer to it. Filing or amending a ticket is the agent's step two, not #483's. Ticket assets filed as ariadne#337 (details local, unpublished); not a dependency.
- **Scope (operator):** a web submission takes only the cast tab's open notes; the terminal cast is #455's.
- **#488 seam (TL, from pair:3, #488 M1 on its branch):** `NoteCrops(st, notes)` with no hints; `Crops` and `notes.Hints` are gone. Each note crops its region + 16 px; overlapping crops merge; clipped to the page; every new note has a region. `NoteAt` gains `kind` (element) and `path` (container chain), the shape the terminal anchor will use. Build against it; merge main once #488 lands.
- **pair#493 (TL):** notes and strokes aren't written into recordings yet (queued for pair:6); live notes work. Until #493 lands, take the submission's notes from the live Board, and test against a fixture with notes or the live Board, not a fresh recording.

## Revisions

- 2026-10-12 — pair:6 resumed after the hub released the old shaping claim. The operator's later decisions now
  have an explicit current contract; initial proposal retained for provenance. #488 and #493 are landed, so
  recorded context can be read mid-cast. Plan proposed in `workshop/plans/000483-dom-submission-plan.md`.
- 2026-10-12 — Hub confirms one crop per note by calling `NoteCrops` per note; no historical replay pixels.
  Already-gone document: preserve recorded context and mark crop unavailable. Navigation during capture:
  refuse preparation. Delivery acknowledgment follows confirmed draft dispatch, with immutable retry identity.

- 2026-10-12 — crop source: the note's box (pair#488), not `NoteCrops` snapping/clustering. If #488 hasn't landed when #483 builds, crop from the note's `region` (always set once #488 lands) through the notes seam #488 fills.

- 2026-10-12 — TL: #488's plan settles the crop as each note's box + 16 px margin; structure-snapped crops and per-URL crop hints are retired. #483 builds its crop on the note's box through the notes seam #488 fills.
