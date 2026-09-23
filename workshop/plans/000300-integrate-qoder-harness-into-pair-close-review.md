# Boundary Review — pair#300 (whole-issue close)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e9ec027c7a55bb1ae6d5054bbdeb0b61a82889..be6dc05c151869112a21f6aac5e44e3cfd684a87 |
| command | sdlc close --issue 300 |
| reviewer | codex |
| timestamp | 2026-09-22T20:59:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The committed range delivers Qoder wiring across the launcher, session inventory, TTY profile, slug path, and docs. The remaining blocker is a contradiction in the plan’s Core concepts table: it classifies functions that read external state or launch a subprocess as PURE. The focused Go package tests passed; I did not rerun the full `make test` gate.

### Strengths

- The [resume-form table](/Users/xianxu/workspace/pair/cmd/internal/resumeform/resumeform.go:23) drives extraction, stripping, and validation, with table-ranging regression tests.
- The [agent parity test](/Users/xianxu/workspace/pair/cmd/internal/launcher/agent_parity_test.go:19) now probes ledger acceptance as well as scanner, CLI, and watcher registration.
- The [chunk pump](/Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go:3143) scans OSC data before bounding its carry; Claude and Codex have regression rows.
- README and atlas changes cover the new Qoder surface.

### Critical findings

- **ARCH-PURE — Correct the Core concepts classification.** The [plan’s PURE table](/Users/xianxu/workspace/pair/workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md:30) lists `scanClaudeFamily`, `ScanQoder`, and `runQoder`. The scanner reads through `Runtime`; [runQoder](/Users/xianxu/workspace/pair/cmd/internal/model/model.go:150) executes an external command and is also listed under Integration points. Move the IO entry points to INTEGRATION, leave the record transition and argument decisions under PURE, and append a `## Revisions` entry. The review contract treats a PURE entity requiring a fake or subprocess as Critical.

### Important findings

None.

### Minor findings

None newly raised. The Claude and Muse headless persistence gap is tracked separately as issue #304.

### Test coverage notes

Focused `go test` passed for `resumeform`, `launcher`, `sessioninventory`, `sessionledger`, `sessionwatch`, `wrapcmd`, `model`, and `changelogcmd`. I inspected the regression tests for the prior resume, dispatch, fail-closed, and OSC findings. Read-only review prevented a scratch mutation check.

### Architectural notes

ARCH-DRY **pass**: shared resume forms and scanner transition. ARCH-PURE **flag**: plan classification above. ARCH-PURPOSE **pass**: both host paths are represented in the issue evidence and wiring. ARCH-MOCK **pass**: runtime fake, TTY captures, and live conformance seam. ARCH-CONSTRAINTS **pass**: TTY carry and capture bounds are explicit. ARCH-SECURE **pass**: untrusted numeric timestamps have bounds and diagnostics. ARCH-ORDER **pass**: picker reset has sequence coverage. ARCH-FUNERAL **pass for Qoder**: headless persistence is disabled; the sibling residue has issue #304.

### Plan revision recommendation

Append a dated `## Revisions` entry that reclassifies scanner IO entry points and `runQoder` as INTEGRATION, identifies the pure transitions they call, and removes `runQoder` from the PURE table.

```findings
dispose:
  - id: BR-15
    disposition: addressed
    note: |
      resumeform.Forms drives the consumers; table-ranging tests cover glued and valueless forms.
  - id: BR-16
    disposition: addressed
    note: |
      TestAgentInventoryParityWithSessionTables now probes sessionledger.ParseLedger.
  - id: BR-25
    disposition: addressed
    note: |
      The parity and fail-closed tests and implementation are committed in the pinned range.
  - id: BR-26
    disposition: addressed
    note: |
      TestAdvanceTargetValidationPerAgent now ranges SupportedAgents().
  - id: BR-27
    disposition: addressed
    note: |
      Both default arms use artifactDiagnostic; unknown-agent tests exercise both paths.
  - id: BR-41
    disposition: addressed
    note: |
      The shared pump scans before trimming; the Claude and Codex long-chunk rows exercise it.
  - id: BR-49
    disposition: addressed
    note: |
      The requested out-of-scope follow-up was committed as pair#304; Claude and Muse behavior remains for that issue.
findings:
  - id: new
    severity: Critical
    family: pure-integration-classification-drift
    title: |
      Core concepts labels scanner IO and runQoder as PURE (ARCH-PURE)
    detail: |
      The plan's PURE table lists scanClaudeFamily, ScanQoder, and runQoder, although the scanners consume Runtime and runQoder launches a subprocess. Reclassify these entry points as INTEGRATION and record the correction in ## Revisions.
```
