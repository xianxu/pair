---
id: 000862
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
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
