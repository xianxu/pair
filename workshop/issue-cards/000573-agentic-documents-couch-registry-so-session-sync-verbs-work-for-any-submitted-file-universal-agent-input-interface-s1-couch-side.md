---
id: 000573
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.704
github_issue:
started: 2026-10-15T10:56:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-cbc210fa0407
        repository: github.com/xianxu/pair
        reviewed_head: 9a428a8590bb98e779e633abcaa63deaa3163b55
        evidence_commit: 2a1d217891de5c883ab9a5d02afcd1d5e652be35
---

# Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side)

## Problem

Project universal-agent-input-interface (ops `teams/devinfra/projects/universal-agent-input-interface/design.md`,
slice S1): Couch's session-sync verbs (`read`, `diff`, `write`, `status`, the owed-event ledger, the CAS write and the
operator-lock refusal) work only for files with `type: session-sync` frontmatter. The operator wants every markdown
file submitted from Couch's right-pane nvim to become an **agentic document** with the same guarantees, without
writing anything into the file.
