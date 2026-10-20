---
id: 000699
status: done
created: 2026-10-19
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-19T21:32:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 0.05
tracker:
    version: 1
    completion:
        token: close-300f6fda32ca
        repository: github.com/xianxu/pair
        reviewed_head: ec35b5d820a6cb463bea6c1690fef44af5ee81ed
        evidence_commit: fdc9435712453408c16d2d9bf977935ff00d13bc
        landed_commit: 2b4a5d3ad19c23d3c262bacb584d62345042f32f
---

# CI: build weave from a pinned ariadne commit

## Problem

ariadne#459 reproduced the cold generated-vocabulary failure in published weave
0.1.0, before ariadne#422's shipped-output sidecar ownership. This repository's
active GitHub main merge-check installs the published formula without an exact
source pin. Verified 2026-10-19 against [the exact workflow blob](https://api.github.com/repos/xianxu/pair/git/blobs/91cfc2012e8f61ff8cfac85b2e7159e0c085711e).
