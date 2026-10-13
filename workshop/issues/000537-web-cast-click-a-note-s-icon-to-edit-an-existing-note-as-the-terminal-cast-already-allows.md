---
id: 000537
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 4.545
card_mirror: '3c2a51180f66e2f0fa4ae71896f82c23070de65f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T16:03:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# web cast: click a note's icon to edit an existing note (as the terminal cast already allows)

## Problem

In a web cast, an existing note's 💭 icon opens a read-only note instead of
an editor. The operator wants the terminal cast's editing interaction in both
the web viewer and the operator overlay inside cast Chrome.

## Spec

Clicking a current note's icon opens the shared note editor with its existing
text when the caller may annotate. Save changes that same note through the
Board as a new revision; ID, original author, anchor and region stay unchanged.
Cancel before Save leaves the note unchanged. View-only links and annotation
OFF show existing notes read-only; neither client hiding nor disabled buttons
replace server admission. Existing OFF clearing behavior stays in force.

Use one explicit note-edit recording event and the existing generation-bound
Board/recorder paths for viewer HTTP and isolated operator commands. Apply
edits immutably in Go/JS reduction and preserve their order in catch-up/replay.
Submission revision guards must notice text edits. The shared dialog owns edit
mode, and pin callbacks/tooltips must use the latest note, not their first value.

Proposed concurrency policy: Board-ordered accepted saves apply in order, with
the latest accepted save winning, as other collaborative note operations do.
No author-only editing restriction is added. Resolved, cleared or old-document
notes cannot be edited or recreated by stale saves. Unknown operator writes
retain their existing request identity for reconciliation; a pre-existing note
ID must never be mistaken for proof that an edit committed.

Failures keep unsaved text and show the existing retry/status recovery; pause
and capability loss prevent mutation. Navigation/removal closes an existing-note
editor; late replies must not close or modify a different editor. Work stays
within the existing note-size, request and single-dialog limits, with no new
background worker, polling loop or durable artifact family.

## Done when

- Viewer and operator-overlay icon clicks open a populated editor when permitted.
- Save updates exactly one note, increments its revision, and preserves author,
  ID and anchoring; Cancel before Save emits no edit.
- Read-only links and annotation OFF cannot edit; stale generations, removed
  notes and retired documents cannot mutate notes or recreate them.
- Current pin text, tooltip, reconnect/catch-up and arbitrary replay seeks show
  the correct revision; editing does not mutate historical creation records.
- Submission resolution cannot consume a note revision edited after selection.
- Viewer and overlay tests cover Save/Cancel/read-only and delayed replies;
  Board/recording tests cover ordering and admission. Targeted, headless and
  full `make -k test` results are recorded, including any known failures.

## Plan

Durable full-flow plan: [own-note editing](../plans/000537-web-note-editing-plan.md).

- [ ] Implement the revision event and authoritative viewer/operator edit paths.
- [ ] Wire shared editor and current pin projections, with regression coverage.
- [ ] Verify, document, close and land through SDLC.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md`
against `baseline-v3.1.md`. Method A only. Calibration is tagged stale by
estimate-source (#127); figures are provisional. Derived after plan-quality
round 2 accepted all findings.

Decomposition: issue/spec design 0.8h plus 0.2h implementation; three small
module extensions (cookie owner, recording/query projection, terminal mutation)
each 0.3h design x0.2 and 0.5h implementation x0.4. Web Board/recorder integration
uses 1.5h design x0.2 and 1.5h implementation x0.4; shared dialog plus adapters
uses tui-screen (stateful UI analogue) at 1.5h x0.2 and 1.0h x0.4. Headless
integration coverage uses 1.0h x0.2 and 1.0h x0.4. Atlas is 0.1h x0.2 plus
0.2h x0.4; one close review is 0.1h plus 0.5h x0.4. Familiarity 1.0; no novel
stack/library work: existing Go SHA-256, HTTP gate, recorder harness and JS
dialog suffice. Implementation values below already include the v3.1 factor.
Design subtotal 1.90h, implementation 2.36h; 15% thorough-plan design buffer
gives 4.545h. Queue/CI/operator idle time is excluded from focused ship-time.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.80 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.30 impl=0.60
item: tui-screen design=0.30 impl=0.40
item: api-integration design=0.20 impl=0.40
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.10 impl=0.20
design-buffer: 0.15
total: 4.545
```

## Log

### 2026-10-13

- Dispatched by ops:0; existing claim confirmed as pair:7. start-plan refreshed
  the slot's dependencies, then created the issue branch on current main.
- Investigation: shared noteDialog supports draft/view only; Board.Note always
  creates a new ID. Operator snapshots and pin callbacks currently retain the
  creation Note. An explicit edit record avoids duplicate notes and preserves
  attribution (ARCH-DRY/ARCH-ORDER). Full-flow plan follows design acceptance.

## Revisions

### 2026-10-13 16:13 PDT — operator ownership rule and terminal scope

- Supersedes the Spec's unrestricted editing proposal and the interim TL
  proposal allowing an operator override: only a note's author may edit it,
  including the operator. Preserve original attribution; show an `edited`
  marker instead of separate editor attribution.
- Include terminal cast editing and server-side ownership tests for both
  casts. Investigation found terminal currently has draft/view/resolve only;
  this needs a text-edit path, not merely a restriction on an existing one.
- Public names and Viewer.ID are self-asserted attribution, not ownership
  proof. A crafted request copying those fields must not edit another note.
  TL approved private browser-held proof and legacy notes remaining read-only;
  investigate reuse of the existing HttpOnly unlock session (ARCH-DRY).
- Existing session cookies survive refresh but are replaced on unlock and
  revoked on code rotation. Asked TL whether ownership must survive those
  transitions before choosing the session identity lifetime. No code yet.

### 2026-10-13 16:15 PDT — existing session lifetime approved

- TL approved the simple existing-cookie reuse: ownership survives refresh,
  but fresh unlock, code rotation, another browser or another link makes old
  notes read-only. Fixed trusted operator ownership stays within the overlay.
  No castlock expansion and no second credential.
- Additional Done when contract: both casts refuse crafted cross-owner edits
  even when public names/IDs are copied; two real unlock-cookie jars prove it.
  Own notes open populated editors; other/legacy notes open read-only; revised
  notes show `edited`. Record tests for the approved session lifetime.

### 2026-10-13 16:20 PDT — resolver attribution, without history UI

- TL adds resolver attribution to both casts' resolve records, including
  operator/submission resolution, so recordings/replay/agent queries retain
  who resolved a note. Resolution remains available to anyone who can annotate.
- The follow-up explicitly excludes resolved-note lists, history UI and reopen
  work from both #537 and #538. Resolved notes still disappear; terminal may
  delete once the resolver is recorded. Investigating its recording seam.
- Plan-quality round 1 asked for function-level test strategies, a pure shared
  dialog transition model, and inherited edit-record storage lifecycle. The
  durable plan now records these refinements; implementation has not begun.

### 2026-10-13 16:22 PDT — terminal has no recording

- TL approved web durable NoteState resolver attribution and terminal typed
  cast-lifetime resolution events only. No journal, history UI or audit follow-up.
  This supersedes the previous wording implying terminal recording/replay/query.
- Additional Done when: resolution remains open to other annotators; web
  recording/replay/query preserves resolver, and terminal records a typed
  bounded in-memory event before deletion. Repeated resolution adds no event.
