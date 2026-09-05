---
id: 000189
status: open
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
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

## Spec

Set a real baseline, or move the status back — but do not invent dates. The
paragraph above is a deliberate stance and it is the right one: a `deadline` is
a commitment about the operator's time, and a forecast nobody made is worse than
an absent one because it silently enters portfolio and calibration views as
though it were a plan.

Then reconcile the Log paragraph with whatever the frontmatter ends up saying,
because a project file that contradicts itself teaches its next reader to trust
neither half.

Worth deciding while here: whether the gate should validate the project file on
every close rather than only when a branch happens to touch it. A conformance
check that fires on edit-proximity rather than on state means a file can be
wrong for two weeks and then blow up in an unrelated merge — which is exactly
how this one arrived.

## Done when

- `workshop/projects/couch.md` conforms: either a real `deadline` +
  `planned_finish` set through `sdlc project set-status`, or a status that does
  not require them.
- The Log paragraph agrees with the frontmatter.
- `sdlc merge` passes the instance-conformance gate on a branch touching the
  file, with no `--no-validate`.

## Plan

- [ ] Get the two dates from the operator, or agree the status should change.
- [ ] Apply via `sdlc project set-status` rather than editing frontmatter by
      hand, so the transition's own rules run.
- [ ] Rewrite the stale Log paragraph.
- [ ] Decide whether project conformance should be checked at close, and file
      that separately if so.

## Log

### 2026-09-04

Filed at the operator's direction during `pair#182`'s merge, which was bypassed
with `--no-validate` rather than inventing a deadline to get past the gate. The
bypass is recorded in that merge; this issue is the debt it names.
