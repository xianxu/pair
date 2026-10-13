# Boundary Review — pair#517 (whole-issue close)

| field | value |
|-------|-------|
| issue | 517 — couch: an agent switches a slot's coding agent (couch --switch-agent repo:N --agent NAME), with a handoff built from durable state |
| repo | pair |
| issue file | workshop/issues/000517-couch-an-agent-switches-a-slot-s-coding-agent-couch-switch-agent-repo-n-agent-name-with-a-handoff-built-from-durable-state.md |
| boundary | whole-issue close |
| milestone | — |
| window | 805031105e60cc0eae0a37adee4b937b39c21f49..489ec0bfc6c07ef2864d9399415e629f9aa10d98 |
| command | sdlc close --issue 517 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T07:41:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the revised lean scope: authenticated hub switching, saved configuration, prompt-free startup, and exact previous/full tty reads. No new blocking defects found. Prior fixes have concrete regression coverage; six mutation checks failed as expected. Repository files remain unchanged.

1. **Strengths**
   - Source identity, idle state and Git cleanliness are rechecked before effects (`agent_switch_service.go`).
   - Previous evidence is tied to the exact verified park and preserved through failed-launch retries (`switchagent.go:375`).
   - Full replay detects capacity loss and rejects invalid geometry instead of returning misleading output.
   - README and atlas document the revised API, recovery behavior and replay limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Targeted tests passed across thread records, core, protocol, launcher, CLI and TUI.
   - Full scrollback and nested VT suites passed.
   - Temporary overlays independently exposed regressions for BR-1/3/4/5/6/9.
   - Missing generated runtime assets initially blocked CLI/TUI compilation; generating them in a temporary pinned-source copy resolved it.
   - Full repository `make test` and live agent switching were not independently rerun.

6. **Architecture**
   - **ARCH-DRY — pass:** reuses configuration resolution, operation receipts and lifecycle owners.
   - **ARCH-PURE — pass:** projection/selection have direct value-only tests; IO uses explicit seams.
   - **ARCH-PURPOSE — pass:** delivers the revised switch-only scope.
   - **ARCH-MOCK — pass:** stateful lifecycle doubles and existing scheduled conformance infrastructure.
   - **ARCH-CONSTRAINTS — pass:** replay bounds and overflow failures are tested.
   - **ARCH-SECURE — pass:** authenticated admission, server-selected argv and validated persisted evidence.
   - **ARCH-ORDER — pass:** interrupted-target tests preserve uncertain effects and exercise late presence.
   - **ARCH-FUNERAL — pass:** bounded previous descriptor, existing archive collection and released replay leases.

7. **Plan revisions:** None required.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Successive-source tests pass; removing source-identity comparison fails both in-switch and separately parked cases.
  - id: BR-2
    disposition: addressed
    note: |
      The authoritative BR-7 inventory supersedes historical tables and identifies implemented owners and classifications.
  - id: BR-3
    disposition: addressed
    note: |
      Lease tests exercise collection during replay and release on success/error; removing lease closure makes both cases fail.
  - id: BR-4
    disposition: addressed
    note: |
      Capacity-loss tests cover rows, cells, bytes and subsequent history clearing; disabling loss checks makes them fail.
  - id: BR-5
    disposition: addressed
    note: |
      Geometry tests pass; replacing checked emulator construction makes the geometry regression test fail.
  - id: BR-6
    disposition: addressed
    note: |
      Interrupted-target sequence tests pass; bypassing pending Start authority causes record-preservation assertions to fail.
  - id: BR-7
    disposition: addressed
    note: |
      The active inventory correctly calls ResolveLaunchProfile unchanged and places known-configuration admission in switchagent.go; comparison against the named M1 baseline confirms this.
  - id: BR-8
    disposition: addressed
    note: |
      Atlas now names reconcileContinuationOrientationLocked, matching the existing pruning implementation and call sites in console_continuation.go.
  - id: BR-9
    disposition: addressed
    note: |
      Direct projection/selection tests use values without IO; removing the selected binding copy fails the aliasing assertion.
```
