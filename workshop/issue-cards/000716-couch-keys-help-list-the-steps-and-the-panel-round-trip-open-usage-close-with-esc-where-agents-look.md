---
id: 000716
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T19:14:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# couch --keys help: list the steps and the panel round trip (open /usage, close with esc) where agents look

## Problem

On 10-19 the TL (ops:0) asked the operator to press Esc by hand to close a `/usage` panel in a slot, believing
`couch --keys` couldn't. `couch --keys ariadne:5 esc` worked first try ("recovery key bytes accepted and current empty
composer observed"). The capability was documented, but only where an agent wouldn't find it in the moment:

- `couch --help` shows just `couch --keys repo:N STEP... [--json]`: no step list, no recipe.
- `couch --keys --help` refuses ("invalid --keys form; use couch --help"), unlike a verb with its own help.
- `couch --skill` lists the steps (`text:…`, `enter`, `submit`, `up`, `down`, `esc`, `clear-line`, `wait:MS`) and the
  recovery-only rule (`clear-line`/`esc` may start on an occupied composer) in prose, but its quick table has no row
  for "close a slash-command panel" and the recovery paragraph speaks of composers, not panels.
- The `/usage` recipe that does appear (`clear-line text:/usage submit`) never says how to close what it opens.
