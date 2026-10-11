---
gate: boundary-review
issue: 324
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T19:04:47-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: The comment and atlas say the ownership probe "refuses JSON, quotes and $"; splitObservedCommand parses quotes
          detail: 'Sites: quiesce_conformance_live_test.go standInPaneScript doc comment, and atlas/couch.md "Ownership evidence" bullet. The parser refuses unquoted ;|&<>`$ and $/backtick inside double quotes; JSON quotes get read as quoting and split the argv wrongly rather than being refused. Reword both.'
          family: doc-claim-matches-code
          round: 1
        - id: BR-2
          severity: Minor
          title: The shared ControlledZellij fixture still launches without --config-dir, though the atlas says to pin it
          detail: 'Family members: terminal_native_test.go (fixed), launcher.OSRuntime.LaunchSession (already pinned), pairlifecycletest.StartControlledZellijWithOptions (live_zellij.go:48-52, unpinned). Harmless today, since layouts are applied via action new-tab; pin it for consistency, or narrow the atlas wording to --new-session-with-layout launches.'
          family: zellij-fixture-config-dir-pin
          round: 1
        - id: BR-3
          severity: Minor
          title: AddOwnerWrapperPane sets stubDir before setup succeeds, and the cold recovery path leaks the first fixture's stub dir
          detail: 'live_zellij.go:163-174: assign f.stubDir only after Symlink and WriteFile succeed, so a failed setup is retried. recovery_conformance_live_test.go:235 reassigns fixture, so the Cleanup closes only the new one and the first stub dir is left behind (that path is currently excluded under pair#432).'
          family: fixture-resource-cleanup
          round: 1
      recipe: small-diff-review
      reviewed: 04729d0e110766fd19d2afa845d71f70d337e148
      blocked: false
---

# Gate ledger — pair#324 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T19:04:47-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `doc-claim-matches-code` The comment and atlas say the ownership probe "refuses JSON, quotes and $"; splitObservedCommand parses quotes
  Sites: quiesce_conformance_live_test.go standInPaneScript doc comment, and atlas/couch.md "Ownership evidence" bullet. The parser refuses unquoted ;|&<>`$ and $/backtick inside double quotes; JSON quotes get read as quoting and split the argv wrongly rather than being refused. Reword both.
- **BR-2** [Minor] `zellij-fixture-config-dir-pin` The shared ControlledZellij fixture still launches without --config-dir, though the atlas says to pin it
  Family members: terminal_native_test.go (fixed), launcher.OSRuntime.LaunchSession (already pinned), pairlifecycletest.StartControlledZellijWithOptions (live_zellij.go:48-52, unpinned). Harmless today, since layouts are applied via action new-tab; pin it for consistency, or narrow the atlas wording to --new-session-with-layout launches.
- **BR-3** [Minor] `fixture-resource-cleanup` AddOwnerWrapperPane sets stubDir before setup succeeds, and the cold recovery path leaks the first fixture's stub dir
  live_zellij.go:163-174: assign f.stubDir only after Symlink and WriteFile succeed, so a failed setup is retried. recovery_conformance_live_test.go:235 reassigns fixture, so the Cleanup closes only the new one and the first stub dir is left behind (that path is currently excluded under pair#432).

## Open findings

- **BR-1** [Minor] `doc-claim-matches-code` The comment and atlas say the ownership probe "refuses JSON, quotes and $"; splitObservedCommand parses quotes
- **BR-2** [Minor] `zellij-fixture-config-dir-pin` The shared ControlledZellij fixture still launches without --config-dir, though the atlas says to pin it
- **BR-3** [Minor] `fixture-resource-cleanup` AddOwnerWrapperPane sets stubDir before setup succeeds, and the cold recovery path leaks the first fixture's stub dir
