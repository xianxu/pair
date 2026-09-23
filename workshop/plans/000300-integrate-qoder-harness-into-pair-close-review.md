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

---

## Re-review — 2026-09-22T21:03:51-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e9ec027c7a55bb1ae6d5054bbdeb0b61a82889..7fce0627ed4c7120b02931df95d7b146c83ffa2f |
| command | sdlc close --issue 300 |
| reviewer | codex |
| timestamp | 2026-09-22T21:03:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The BR-50 correction is valid: the plan now classifies the scanner entry points and `runQoder` as INTEGRATION, consistent with their runtime reads and subprocess call. One Core concepts row still points to a fixture directory absent from the pinned head, so the plan and code disagree at this boundary.

### Strengths

- The scanner keeps record validation separate from reads through `Runtime` ([scan_claude.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/scan_claude.go:104)).
- Qoder’s composer uses the shared ruled-box recognizer with a Qoder-specific spec ([composer_recognizers.go](/Users/xianxu/workspace/pair/cmd/internal/wrapcmd/composer_recognizers.go:311)).
- README and atlas changes cover the new user-facing agent.

### Critical findings

- **Core concepts table contradicts the pinned tree — ARCH-PURPOSE.** The Integration points table names `cmd/internal/wrapcmd/testdata/tty/qoder/1.1.59/` as the new capture location ([plan](/Users/xianxu/workspace/pair/workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md:59)). The pinned head contains only `1.1.60/`. The same table names `~/.qoder/settings.json` as the trust configuration ([plan](/Users/xianxu/workspace/pair/workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md:64)), while its later revision says the allowlist moved to repo-local `.qoder/settings.local.json` ([plan](/Users/xianxu/workspace/pair/workshop/plans/000300-integrate-qoder-harness-into-pair-plan.md:892)). Update the table to describe the delivered locations and append a `## Revisions` entry recording the reconciliation. This is the **5th finding in family `plan-prose-restates-diff`**: reconcile *every* Core concepts location against the delivered tree and final decisions, rather than fixing only these two cells.

### Important findings

None.

### Minor findings

None.

### Test coverage notes

`go test ./cmd/internal/sessioninventory ./cmd/internal/model ./cmd/internal/wrapcmd -count=1` passed. BR-50 changes plan classification only, so its evidence is the before/after plan diff and the referenced code; no wording test is needed. The fixture-path finding is established by the pinned tree, not a runtime test.

### Architectural notes for upcoming work

ARCH-DRY **pass**: the scanner and composer reuse shared implementations. ARCH-PURE **pass for BR-50**: the revised labels follow the effect boundary. ARCH-PURPOSE **flag**: the concept table still describes locations that were superseded. ARCH-MOCK **pass**: fake runtime and captured TTY replay cover the new seams. ARCH-CONSTRAINTS **pass**: no new unbounded work found in this review. ARCH-SECURE **pass**: no new credential exposure found. ARCH-ORDER **pass**: overlay consumption has a regression test. ARCH-FUNERAL **pass**: headless Qoder calls disable session persistence.

### Plan revision recommendation

Append a dated `## Revisions` entry stating that the Core concepts location sweep changed the capture row to `1.1.60/` and the trust row to repo-local `.qoder/settings.local.json`.

```findings
dispose:
  - id: BR-50
    disposition: addressed
    note: |
      The pinned plan diff moves scanClaudeFamily, scanClaudeFamilyFile, ScanQoder, and runQoder out of PURE; scan_claude.go reads Runtime and model.go launches qoder. The 2026-09-22 Revisions entry records the taxonomy correction.
findings:
  - id: new
    severity: Critical
    family: plan-prose-restates-diff
    title: |
      Core concepts locations still contradict the delivered capture and settings locations
    detail: |
      The plan table at lines 59 and 64 names qoder/1.1.59/ and user-scope ~/.qoder/settings.json; the pinned tree has captures only under qoder/1.1.60/, and the plan's later revision says the allowlist moved to repo-local .qoder/settings.local.json. This is the 5th finding in family plan-prose-restates-diff. Sweep every Core concepts location against the delivered tree and final decisions, then correct the table and record the sweep in ## Revisions.
```

---

## Re-review — 2026-09-22T21:10:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08e9ec027c7a55bb1ae6d5054bbdeb0b61a82889..2c594b033add86065e7bad1c3ace81cf496167cf |
| command | sdlc close --issue 300 |
| reviewer | codex |
| timestamp | 2026-09-22T21:10:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The remaining finding, BR-51, is addressed. The plan now points to the committed Qoder 1.1.60 captures and distinguishes existing user-level workspace trust from the repo-local command allowlist. I found no new blocking issue in the pinned range.

**Strengths**

- The Core concepts paths match the scanner, launcher, TTY, model, and glyph files in the pinned tree.
- The Claude-family scanner shares a record transition while keeping Qoder’s timestamp and noise rules agent-specific.
- Captured TTY fixtures, resume-form tests, and cross-agent parity tests cover the main integration paths.

**Critical findings:** None.

**Important findings:** None.

**Minor findings:** None.

**Test coverage:** Six focused Go packages passed with `-count=1`: launcher, resumeform, sessioninventory, wrapcmd, model, and changelogcmd. `git diff --check` passed. I did not rerun the full suite or live Qoder conformance in this review.

**Architecture:** ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK, ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER, and ARCH-FUNERAL: pass for this boundary. The shared scanner and resume-form table avoid parallel implementations; pure record decisions are separated from runtime reads; captured replay and focused integration tests exercise the new paths; the inspected changes introduce no unresolved trust, ordering, resource, or durable-residue finding.

**Plan revisions:** None needed.

```findings
dispose:
  - id: BR-51
    disposition: addressed
    note: |
      The pinned plan changes the capture row to qoder/1.1.60/ (line 60), matching the committed fixtures, and separates ~/.qoder/settings.json workspace trust from <repo>/.qoder/settings.local.json permissions.allow (lines 65–66). Task 9 and Task 15 use those locations; the September 22 Revisions entry records the reconciliation. The local-only allowlist exists in the review workspace with the five specified rules; it is intentionally untracked.
```
