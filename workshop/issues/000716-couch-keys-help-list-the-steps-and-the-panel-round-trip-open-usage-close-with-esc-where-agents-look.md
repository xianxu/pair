---
id: 000716
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd8b3ec2af2e6ae37007fc788dcc5e59d2c09777f' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --keys help: list the steps and the panel round trip (open /usage, close with esc) where agents look

## Problem

On 10-19 the TL (ops:0) could not tell from `couch --help` which keys `--keys` takes or what happens with an open
slash-command panel, and then misread a `--keys esc` receipt ("recovery key bytes accepted and current empty composer
observed") as having closed a `/usage` panel. pair:2 showed from the wrapper recording that the operator had already
closed it; an open panel still refuses every key, including `esc`, until pair#691 lands. The rules exist, but only
where an agent wouldn't find them in the moment:

- `couch --help` shows just `couch --keys repo:N STEP... [--json]`: no step list, no recipe.
- `couch --keys --help` refuses ("invalid --keys form; use couch --help"), unlike a verb with its own help.
- `couch --skill` lists the steps (`text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line`, `wait:MS`) and the
  recovery-only rule (`clear-line`/`esc` may start on an occupied composer) in prose, but its quick table has no row
  for "close a slash-command panel" and the recovery paragraph speaks of composers, not panels.
- The `/usage` recipe that does appear (`clear-line text:/usage submit`) never says how to close what it opens.

## Spec

Make the step list and the panel round trip discoverable from the command line:

1. `couch --keys --help` (and `-h`) prints the verb's own help: the steps, the limits, the empty-composer rule and
   its recovery-only exception, and two recipes: open and read a slash-command panel
   (`couch --keys SLOT clear-line text:/usage submit`, then `couch --peek SLOT --lines 40`) and close it
   (`couch --keys SLOT esc`).
2. `couch --help`'s `--keys` line names the steps inline (`STEP: text:… enter submit up down esc clear-line wait:MS`)
   and points at `couch --keys --help`.
3. `couch --skill` quick table gains a row "Close a slash-command panel (/usage, /status) | couch --keys pair:7 esc",
   and the recovery paragraph says it covers an open panel.
4. One source for the step list (ARCH-DRY): help, skill and the parser's accepted set come from the same table, with
   a test that they agree.

## Done when

- `couch --keys --help` prints the steps and both panel recipes (test on the output).
- `couch --help` names the steps on the `--keys` line.
- The skill's quick table has the close-panel row.
- A test fails if the parser accepts a step that help omits, or the reverse.

## Plan

- [ ]

## Log

- 10-19 correction (TL): the original Problem said `--keys esc` closed the panel; it didn't (pair:2's recording
  analysis). Rewritten above; the close recipe depends on pair#691.
- Overlap: pair#691 (Esc admissibility while a panel shows; being re-scoped after this finding). Whichever lands
  second accounts for the other's wording.
