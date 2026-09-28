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
    - "n": 2
      timestamp: "2026-09-28T12:27:52-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The composed Codex/Claude regression exercises real watcher execution, detach, post-reattach binding and relaunch authorization. Independently restoring client-group ownership in a scratch overlay made both cases fail at post-reattach binding.
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/session-identity.md:55–57, wrap.go:2370–2372 and the issue's lifecycle paragraph now explicitly preserve Codex observation after binding, matching sessionwatch/runcli.go:144 and run.go:185–196.
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: Acceptance regression races final relaunch authorization
          detail: 'cmd/internal/wrapcmd/detach_acceptance_test.go:200–210 waits only for binding before a single resolver call. The full suite failed for Codex at line 208 with “session inventory storage root is absent”; diagnostic repetitions reproduced it for Claude against the Pair data root. Both cases share this site and exhaust the family instances in this window. This is the 2nd finding in family acceptance-boundary-coverage: apply the rule that asynchronous acceptance tests await their final contractual outcome, rather than an intermediate publication. Boundedly await successful resolution and relaunch preconditions for both agents, retaining diagnostic errors and the ownership mutation check. ARCH-PURPOSE.'
          family: acceptance-boundary-coverage
          round: 2
      recipe: small-diff-review
      blocked: true
    - "n": 3
      timestamp: "2026-09-28T12:37:38-07:00"
      agent: codex
      dispose:
        - id: BR-3
          disposition: addressed
          note: cmd/internal/wrapcmd/detach_acceptance_test.go:205–228 boundedly retries resolution and relaunch preconditions for both agents, retaining failure diagnostics. Ten race-enabled repetitions per agent passed; restoring the previous test in a scratch overlay reproduced the storage-root failure for both Codex and Claude.
          round: 3
      recipe: small-diff-review
      blocked: false
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

## Round 2 — 2026-09-28T12:27:52-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The composed Codex/Claude regression exercises real watcher execution, detach, post-reattach binding and relaunch authorization. Independently restoring client-group ownership in a scratch overlay made both cases fail at post-reattach binding.
- BR-2 — addressed — atlas/session-identity.md:55–57, wrap.go:2370–2372 and the issue's lifecycle paragraph now explicitly preserve Codex observation after binding, matching sessionwatch/runcli.go:144 and run.go:185–196.

### Raised

- **BR-3** [Important] `acceptance-boundary-coverage` Acceptance regression races final relaunch authorization
  cmd/internal/wrapcmd/detach_acceptance_test.go:200–210 waits only for binding before a single resolver call. The full suite failed for Codex at line 208 with “session inventory storage root is absent”; diagnostic repetitions reproduced it for Claude against the Pair data root. Both cases share this site and exhaust the family instances in this window. This is the 2nd finding in family acceptance-boundary-coverage: apply the rule that asynchronous acceptance tests await their final contractual outcome, rather than an intermediate publication. Boundedly await successful resolution and relaunch preconditions for both agents, retaining diagnostic errors and the ownership mutation check. ARCH-PURPOSE.

## Round 3 — 2026-09-28T12:37:38-07:00 (codex) — passed

### Disposed

- BR-3 — addressed — cmd/internal/wrapcmd/detach_acceptance_test.go:205–228 boundedly retries resolution and relaunch preconditions for both agents, retaining failure diagnostics. Ten race-enabled repetitions per agent passed; restoring the previous test in a scratch overlay reproduced the storage-root failure for both Codex and Claude.

## Open findings

(none — every finding has been disposed)
