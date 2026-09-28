---
id: 000189
status: open
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
github_issue:
---

# couch project is executing with no deadline or planned_finish baseline

## Problem

`workshop/projects/couch.md` declares `status: executing` and carries neither
`deadline` nor `planned_finish`. The project datatype requires both for
`committed`/`executing`/`paused` (`construct/datatype/project.md:30`), so the
instance-conformance gate refuses it:

    workshop/projects/couch.md does not conform to #Project:
      - deadline: required field is missing
      - planned_finish: required field is missing

Pre-existing: `status: executing` has been on `main` since `9973206d`
(`pair#170`'s rescope to couch-lite). It surfaced now only because `pair#182`
edited the file to tick its rows, which brought it into the gate's changed-file
scope — the gate validates what a branch touches, so an untouched nonconforming
file stays invisible until someone edits it for an unrelated reason.

The file's own Log still reads:

> Status is `defined` rather than `committed`: the PRD exists but no `deadline`
> or `planned_finish` has been set, and neither was invented. Moving to
> `committed` needs both, via `sdlc project set-status`.

So there are two defects, and the second is the interesting one:

1. The baseline was never set, and the status advanced past `defined` anyway —
   `defined -> committed` is the transition that is supposed to demand it.
2. **The prose and the frontmatter disagree, and have since `pair#170`.** The Log
   describes a `defined` project; the frontmatter says `executing`. Whichever
   moved the status did not update the paragraph explaining why it had not
   moved.
