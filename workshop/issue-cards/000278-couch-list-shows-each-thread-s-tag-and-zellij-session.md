---
id: 000278
status: working
started: 2026-09-17T15:54:17-07:00
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
---

# couch --list shows each thread's tag and zellij session

## Problem

Operator friction, hit live on 2026-09-17: wanting to `zellij kill-session` the
ariadne thread, there was no way to get from what `couch --list` prints to the
session name zellij knows. It took reading two on-disk files by hand:

    ~/.local/share/pair/couch/threadstore/records/<scope>/<tag>.json   # tag -> path
    ~/.local/share/pair/repos/<scope>/session-names.jsonl              # tag -> session name

`couch --show <ref>` resolves a tag, a path or an operator-assigned NAME — and
these threads have no name set, so `couch --show ariadne` answers "thread
reference not found". The label column shows a disambiguated repo name, which is
not something any other tool accepts as input.
