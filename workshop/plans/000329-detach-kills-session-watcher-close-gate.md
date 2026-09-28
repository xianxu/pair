---
gate: boundary-review
issue: 329
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-28T12:14:40-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: Required detach-before-first-turn acceptance sequence is untested
          detail: cmd/internal/wrapcmd/launch_watcher_test.go:56 stubs watcher execution; :131 checks a sleep child's PGID. Neither establishes binding or relaunch after detach and reattach, and workshop/issues/000329-detach-kills-session-watcher.md:98–101 explicitly requires that sequence. The recorded smoke detached only after turns. These are the acceptance-evidence gaps in this window. Add a composed regression covering Codex and Claude with stateful native fixtures, asserting relaunch authorization after the first post-reattach turn, and verify it fails under the original ownership arrangement. ARCH-PURPOSE.
          family: acceptance-boundary-coverage
          round: 1
        - id: BR-2
          severity: Minor
          title: Watcher lifetime claims omit continued Codex lifecycle observation
          detail: All instances are atlas/session-identity.md:55–56, cmd/internal/wrapcmd/wrap.go:2370–2371, and workshop/issues/000329-detach-kills-session-watcher.md:86–87. They say the watcher exits when bound, but sessionwatch/runcli.go:144 enables FollowLifecycle and sessionwatch/run.go:185–196 continues Codex observation after binding. Qualify these statements to describe the Codex exception.
          family: lifecycle-documentation-fidelity
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — pair#329 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-28T12:14:40-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `acceptance-boundary-coverage` Required detach-before-first-turn acceptance sequence is untested
  cmd/internal/wrapcmd/launch_watcher_test.go:56 stubs watcher execution; :131 checks a sleep child's PGID. Neither establishes binding or relaunch after detach and reattach, and workshop/issues/000329-detach-kills-session-watcher.md:98–101 explicitly requires that sequence. The recorded smoke detached only after turns. These are the acceptance-evidence gaps in this window. Add a composed regression covering Codex and Claude with stateful native fixtures, asserting relaunch authorization after the first post-reattach turn, and verify it fails under the original ownership arrangement. ARCH-PURPOSE.
- **BR-2** [Minor] `lifecycle-documentation-fidelity` Watcher lifetime claims omit continued Codex lifecycle observation
  All instances are atlas/session-identity.md:55–56, cmd/internal/wrapcmd/wrap.go:2370–2371, and workshop/issues/000329-detach-kills-session-watcher.md:86–87. They say the watcher exits when bound, but sessionwatch/runcli.go:144 enables FollowLifecycle and sessionwatch/run.go:185–196 continues Codex observation after binding. Qualify these statements to describe the Codex exception.

## Open findings

- **BR-1** [Important] `acceptance-boundary-coverage` Required detach-before-first-turn acceptance sequence is untested
- **BR-2** [Minor] `lifecycle-documentation-fidelity` Watcher lifetime claims omit continued Codex lifecycle observation
