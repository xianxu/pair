---
id: 000716
status: codecomplete
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b995c6b70d1863b4218692c3f60fff8a2b754807' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T19:14:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.16
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

Quick flow. One step table drives the parser, the help and the skill check (ARCH-DRY).

- [x] `agentdrive.Steps` (keys.go): one row per caller step (kind, syntax, meaning). `Validate` accepts a named key only if it is a row, so the parser's accepted set is the table by construction. `StepSyntax()` renders the inline list.
- [x] `couch --keys --help` / `-h` (parsed in `parseAgentDriveCLI` before the slot argument): prints the table, the limits from the same constants, the admission rules (idle composer; `clear-line`/`esc` on an occupied composer; exactly `esc` on a recognized panel), and the round trip: `couch --keys SLOT clear-line text:/usage submit`, `couch --peek SLOT --lines 40`, `couch --keys SLOT esc`.
- [x] `couch --help`'s `--keys` line names `StepSyntax()` and points at `couch --keys --help`.
- [x] Skill: a quick-table row "Owning hub: close a slash-command panel it opened (/usage, /status) | `couch --keys pair:7 esc`", and the steps sentence lists every row. A test fails if the skill's steps sentence and the table disagree in either direction.
- [x] Review advisory from #691 (hub asked to fold it in): a refusal while a recognized panel shows names `esc` as the one admitted key. `agentdrive.Admission.Refusal(otherwise)` returns the panel hint when `Dismissable`, else the caller's existing text. Used by the reducer start, the wrapper start and the service. The service's early not-settled refusal of non-recovery keys makes one read-only `ObserveDrive` on that refusal path only, so that `text:` or `submit` on a panel also gets the hint.
- [x] Tests: table and parser agreement (every row parses; an unknown or internal kind refuses); keys help output (steps, limits, both recipes); `couch --help` names the steps; skill agreement; refusal hint at the reducer, wrapper and service levels.

Not a state change: there are no new admissible states. The refusal text changes, and the service makes one extra observation on a path that already refuses.

## Log




- 2026-10-19: closed — Full suite green on 76299b606 (main merged, includes pair#715). 657e99e1e fixes both close advisories: a refused lone esc keeps the generic wording (TestPanelRefusalNamesEsc), and the wait meaning and the skill limits come from the constants (TestSkillKeyStepsMatchTheTable); both mutations red. Earlier failure on 657e99e1 was the pair#715 family (#718); continuation test 20/20 on main under load. (fixed-then-ship); review verdict: SHIP
- 2026-10-19: closed — Full suite green on 5dddd69ef (sdlc-local/suite). TestKeysHelpIsTheVerbsOwnHelp (couch --keys --help and -h print every Steps row, the limits and the /usage open, peek, esc round trip); TestCouchHelpNamesTheKeySteps; TestSkillKeyStepsMatchTheTable (skill steps sentence equals the table both ways, close-panel row present); TestStepsTableIsTheParsersAcceptedSet; panel refusal names esc at reducer (TestPanelRefusalNamesEsc), wrapper (TestAgentDrivePanelAdmitsOnlyEsc) and service (TestDrivePanelAdmitsOnlyEsc, incl. text and submit via the not-settled path). Four mutations each red.; review verdict: SHIP
- 2026-10-19: flow upgraded quick → full — 114 added lines in code files (limit 100)
- 2026-10-19 (pair:2): implemented per Plan; the hub approved it ("proceed"). Mutations, each red: no panel hint (agentdrive, wrapcmd, couchcmd), no early service observe (couchcmd), skill sentence drops a step (couchcmd), table gains a row (agentdrive, couchcmd). The table syntax for text is `text:LITERAL`; the skill now uses the same spelling. start-plan weave rewrote the two CI workflow files from the ariadne base; I restored them, since they belong to pair#699 (hub: keep restoring until it lands).

- 10-19 correction (TL): the original Problem said `--keys esc` closed the panel; it didn't (pair:2's recording
  analysis). Rewritten above; the close recipe depends on pair#691.
- Overlap: pair#691 (Esc admissibility while a panel shows; being re-scoped after this finding). Whichever lands
  second accounts for the other's wording.
