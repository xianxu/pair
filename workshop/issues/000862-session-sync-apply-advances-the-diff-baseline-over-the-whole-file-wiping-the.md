---
id: 000862
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'ad34b49a65b2a57860ee75ffabe52338c2b94058' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync apply advances the diff baseline over the whole file, wiping the operator's unread edits elsewhere

## Problem

`couch --session-sync apply` is a whole-file write, and it advances the agent's diff baseline to the whole new
document, so it consumes the operator's unread edits everywhere in the file, not only where its records landed.

Evidence (cmd/internal/sessionsync/sync.go):

- `Apply` runs `s.write` with no `Section` and no `Frontmatter` (the records build the body).
- `prepareWrite` then takes `shadow := f.shadowAfter(req.Section, body, next, now)`.
- `shadowAfter` returns `next`, the whole new file, when `section == ""`.

So after an apply, `diff` no longer shows the operator's edits in sections the records never touched. A section
write (`shadowAfter` with a section) and a frontmatter write (`frontmatterShadow`) both advance only their own
scope, so the operator's edits elsewhere stay in `diff` (pair#448, ariadne#426). Apply is the only write that
doesn't keep them.

Found by ariadne#588: `tl-lead instructions link` was going to put the `discussions:` index entry and the answer link
in one apply to make the operation all or none. That would have erased the operator's unread `## Needs you`
edits from the TL's diff, so ariadne#588 kept two scoped writes and an undo instead. ariadne's tl-lead
session-sync fake now models apply advancing the whole-file baseline (TestFakeApplyAdvancesBaseline).

## Spec

A whole-file apply advances the baseline only for the regions its records edited: the shadow takes each edited
range's new bytes, and the operator's unread edits outside them stay in `diff`. The shadow and the file can then
differ at the same offsets they differed before, outside the edited ranges; mapping the records onto the shadow
needs the same resolution against the shadow's text (or a line diff of file before and after, replayed on the
shadow), refusing to advance (keeping the old shadow plus the edited ranges marked) when a record's range doesn't
map cleanly.

## Done when

- An apply whose records touch only one section leaves an operator's unread edit in another section in `diff`
  (test).
- An apply's own edits don't show in `diff` afterwards (test).
- ariadne's tl-lead fake is updated to match, so its `instructions link` can move to one apply if wanted.

## Plan

- [ ]

## Log

### 2026-10-24
