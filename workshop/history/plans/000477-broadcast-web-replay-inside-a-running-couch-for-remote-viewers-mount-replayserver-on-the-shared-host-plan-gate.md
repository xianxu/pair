---
gate: plan-quality
issue: 477
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-12T05:13:35-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: ReplayShares never says what Close does to a share whose background Mount is still opening
          detail: 'Close is teardown and Mount can run up to 30s. The plan must say: Close cancels a context held by ReplayShares (Host.Mount abandons on ctx cancel, host.go:85), Close waits for the mount goroutines, Share after Close answers failed or closed, and a mount that completes after Close is unmounted at once. Name the states (opening/ready/failed/lost/closed). Add an ordering test on FakeTunnel: Close during opening leaves Opens == Closes and no goroutine. Also test that two Shares for one id while opening give one mount and one token.'
          family: async-lifecycle-close-races-inflight
          round: 1
        - id: PQ-2
          severity: Minor
          title: What query link does when no Couch answers is unspecified
          detail: An agent's command must return, so it can't serve on loopback. Say whether it refuses or prints a local-only note, and test that path. This shapes the Linker seam.
          family: cli-fallback-unspecified
          round: 1
        - id: PQ-3
          severity: Minor
          title: M2 depends on pair 470, but frontmatter deps is empty
          family: undeclared-dependency
          round: 1
        - id: PQ-4
          severity: Minor
          title: The CLI's poll interval, and what it prints at the 45s deadline while the share is still opening, are unstated
          family: operating-envelope-unstated
          round: 1
        - id: PQ-5
          severity: Minor
          title: The Tests section lists cases; reduce it to the functions under test with one strategy line each
          detail: 'For example: the share state machine gets sequence tests with controlled ordering on FakeTunnel; the scoped ReplayServer gets path fuzzing (other id, listing, traversal); headAndEnd gets a truncated or garbage last line.'
          family: plan-test-enumeration
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T05:14:22-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: States are enumerated; Close cancels the owned ctx (host.go:85 confirmed), joins goroutines, unmounts late completions; tests for Close-while-opening and duplicate Share are named.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: query link refuses with exit 1 and a pointer to --broadcast-web-replay; the path is covered in the CLI share-and-poll tests.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'Frontmatter now has deps: [pair#470].'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: 300ms poll, 45s deadline past the 30s DefaultProbeTimeout (session.go:20), still-opening message, exit 1.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Tests are grouped per function with a strategy line for each (ordered FakeTunnel sequences, path table, torn last line).
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-10-12T05:17:36-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-10-12T05:18:08-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          round: 4
        - id: PQ-2
          disposition: addressed
          round: 4
        - id: PQ-3
          disposition: addressed
          round: 4
        - id: PQ-4
          disposition: addressed
          round: 4
        - id: PQ-5
          disposition: addressed
          round: 4
      blocked: false
content_hash: b97842042226457e98f018e772245bab321698e2f7a3d4b04598e39e14b66342
---

# Gate ledger — pair#477 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T05:13:35-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `async-lifecycle-close-races-inflight` ReplayShares never says what Close does to a share whose background Mount is still opening
  Close is teardown and Mount can run up to 30s. The plan must say: Close cancels a context held by ReplayShares (Host.Mount abandons on ctx cancel, host.go:85), Close waits for the mount goroutines, Share after Close answers failed or closed, and a mount that completes after Close is unmounted at once. Name the states (opening/ready/failed/lost/closed). Add an ordering test on FakeTunnel: Close during opening leaves Opens == Closes and no goroutine. Also test that two Shares for one id while opening give one mount and one token.
- **PQ-2** [Minor] `cli-fallback-unspecified` What query link does when no Couch answers is unspecified
  An agent's command must return, so it can't serve on loopback. Say whether it refuses or prints a local-only note, and test that path. This shapes the Linker seam.
- **PQ-3** [Minor] `undeclared-dependency` M2 depends on pair 470, but frontmatter deps is empty
- **PQ-4** [Minor] `operating-envelope-unstated` The CLI's poll interval, and what it prints at the 45s deadline while the share is still opening, are unstated
- **PQ-5** [Minor] `plan-test-enumeration` The Tests section lists cases; reduce it to the functions under test with one strategy line each
  For example: the share state machine gets sequence tests with controlled ordering on FakeTunnel; the scoped ReplayServer gets path fuzzing (other id, listing, traversal); headAndEnd gets a truncated or garbage last line.

## Round 2 — 2026-10-12T05:14:22-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — States are enumerated; Close cancels the owned ctx (host.go:85 confirmed), joins goroutines, unmounts late completions; tests for Close-while-opening and duplicate Share are named.
- PQ-2 — addressed — query link refuses with exit 1 and a pointer to --broadcast-web-replay; the path is covered in the CLI share-and-poll tests.
- PQ-3 — addressed — Frontmatter now has deps: [pair#470].
- PQ-4 — addressed — 300ms poll, 45s deadline past the 30s DefaultProbeTimeout (session.go:20), still-opening message, exit 1.
- PQ-5 — addressed — Tests are grouped per function with a strategy line for each (ordered FakeTunnel sequences, path table, torn last line).

## Round 3 — 2026-10-12T05:17:36-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-10-12T05:18:08-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed
- PQ-2 — addressed
- PQ-3 — addressed
- PQ-4 — addressed
- PQ-5 — addressed

## Open findings

(none — every finding has been disposed)
