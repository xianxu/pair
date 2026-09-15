---
gate: boundary-review
issue: 245
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T20:48:59-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: An incomplete prefix suppresses complete reserved shortcuts and permits unbounded pending input
          detail: cmd/internal/wrapcmd/wrap.go:1671 and :1594 retain a prefix together with an already-complete chord. A scratch test using text\x1b\x1b[84;4u fails at HEAD with zero actions and passes with only the previous holdback logic restored. Resolve prefixes against all available bytes and test bounded pending state across subsequent reads, timeout, and EOF in both adaptation modes. ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER.
          family: bounded-incremental-input-framing
          round: 1
        - id: BR-2
          severity: Important
          title: Both conformance event filters omit source paths promised by the completed plan
          detail: .github/workflows/couch-zellij-conformance.yml:23 and :64 omit wrapper, Couch keys.go/console.go, and Neovim paths. Add these selectors to both pull_request and push filters and verify representative paths match, as Task 5 requires. ARCH-PURPOSE.
          family: conformance-source-trigger-coverage
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T20:59:54-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The incomplete-prefix regressions pass at HEAD, including race coverage. Removing the prefix-release fix through a temporary Go overlay makes TestReservedShortcutAfterIncompletePrefixEOF fail with zero actions, establishing regression sensitivity.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Both workflow filters now match the promised source families. An independent before/after check confirms five missing representative paths become covered in each event. However, no committed regression test preserves this check; the plan records only an ad hoc run. Under the executable-configuration evidence requirement, add a reproducible test covering both event filters that fails when the added selectors are removed.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-14T21:04:39-07:00"
      agent: codex
      dispose:
        - id: BR-2
          disposition: addressed
          note: Both workflow filters cover the missing sources. The committed regression passes at HEAD; removing the selectors in a scratch copy fails all five previously missing paths in both events.
          round: 3
        - id: BR-1
          disposition: addressed
          note: The prior disposition stands. Prefix disambiguation remains present, and the wrapper suite passes the partition, progress, timeout and EOF regressions.
          round: 3
      blocked: false
---

# Gate ledger — pair#245 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T20:48:59-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `bounded-incremental-input-framing` An incomplete prefix suppresses complete reserved shortcuts and permits unbounded pending input
  cmd/internal/wrapcmd/wrap.go:1671 and :1594 retain a prefix together with an already-complete chord. A scratch test using text\x1b\x1b[84;4u fails at HEAD with zero actions and passes with only the previous holdback logic restored. Resolve prefixes against all available bytes and test bounded pending state across subsequent reads, timeout, and EOF in both adaptation modes. ARCH-PURPOSE, ARCH-CONSTRAINTS, ARCH-ORDER.
- **BR-2** [Important] `conformance-source-trigger-coverage` Both conformance event filters omit source paths promised by the completed plan
  .github/workflows/couch-zellij-conformance.yml:23 and :64 omit wrapper, Couch keys.go/console.go, and Neovim paths. Add these selectors to both pull_request and push filters and verify representative paths match, as Task 5 requires. ARCH-PURPOSE.

## Round 2 — 2026-09-14T20:59:54-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The incomplete-prefix regressions pass at HEAD, including race coverage. Removing the prefix-release fix through a temporary Go overlay makes TestReservedShortcutAfterIncompletePrefixEOF fail with zero actions, establishing regression sensitivity.
- BR-2 — not-addressed — Both workflow filters now match the promised source families. An independent before/after check confirms five missing representative paths become covered in each event. However, no committed regression test preserves this check; the plan records only an ad hoc run. Under the executable-configuration evidence requirement, add a reproducible test covering both event filters that fails when the added selectors are removed.

## Round 3 — 2026-09-14T21:04:39-07:00 (codex) — passed

### Disposed

- BR-2 — addressed — Both workflow filters cover the missing sources. The committed regression passes at HEAD; removing the selectors in a scratch copy fails all five previously missing paths in both events.
- BR-1 — addressed — The prior disposition stands. Prefix disambiguation remains present, and the wrapper suite passes the partition, progress, timeout and EOF regressions.

## Open findings

(none — every finding has been disposed)
