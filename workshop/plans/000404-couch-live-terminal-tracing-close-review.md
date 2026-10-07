# Boundary Review — pair#404 (whole-issue close)

| field | value |
|-------|-------|
| issue | 404 — Opt-in Couch live terminal tracing |
| repo | pair |
| issue file | workshop/issues/000404-couch-live-terminal-tracing.md |
| boundary | whole-issue close |
| milestone | — |
| window | b933b5a5bfef7fb681c5dffa2fd39c40d8765e7d..340664fd9161b673b956a0b037abde0d2f96f881 |
| command | sdlc close --issue 404 |
| reviewer | codex |
| timestamp | 2026-10-07T11:57:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned implementation passes focused and race tests and delivers the main capture behavior. Three Important gaps remain: README coverage, aggregate retention, and the required pure lifecycle transition model. No Critical correctness defect was established.

1. **Strengths**
   - Endpoint observations share the parser/resize lock and record applied geometry even when subsequent replies fail.
   - Queue accounting includes in-flight records; blocked-writer tests exercise both startup bursts and near-limit allocations.
   - Host capture preserves partial write receipts and underlying errors.
   - Capture failure reaches actor and switcher views through the existing Presenter, including while idle.

2. **Critical findings:** None.

3. **Important findings**
   - **README coverage:** `cmd/internal/couchcmd/capture.go:40` introduces operator-facing settings, but README is unchanged. Add activation, limits, stopped-state behavior, and a runbook link.
   - **Aggregate retention — ARCH-FUNERAL:** `cmd/internal/terminalcapture/recorder.go:106` creates a new directory every launch. The per-file cap does not bound accumulated sessions; `atlas/couch-live-capture.md:90` explicitly delegates cleanup to the operator. Add an enforced aggregate budget with visible admission refusal, or automatic retention that protects incident evidence.
   - **Lifecycle ownership — ARCH-ORDER / ARCH-PURE:** `cmd/internal/terminalcapture/recorder.go:177` mixes transitions with channel effects, while `:299` independently changes phase in the IO worker. Extract the required pure state/event transition component and route admission, failure, close, timeout, and worker completion through it.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused tests passed across terminalcapture, terminal, couchcmd, couchtty, and couchcore.
   - Focused race tests passed across terminalcapture, terminal, couchcmd, and couchtty.
   - Child-environment clearing test and pinned diff whitespace check passed.
   - Existing tests provide useful deterministic blocked-write coverage. Add lifecycle sequence tests against the extracted production model and repeated-launch tests for retention enforcement.
   - I did not rerun the full repository suite or independently establish the documented baseline failures.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Shared configuration validation and existing Presenter reused.
   - **ARCH-PURE — flag:** Lifecycle policy remains inside the IO-owning recorder.
   - **ARCH-PURPOSE — pass:** Both requested boundaries and persistent failure visibility are delivered; no claim that #379 is fixed.
   - **ARCH-MOCK — pass:** Stateful fault sink, fake terminal fixtures, and real-PTY checks exercise the relevant boundaries.
   - **ARCH-CONSTRAINTS — pass:** Admission and file budgets are enforced; memory overhead and filesystem cancellation limitations are documented.
   - **ARCH-SECURE — pass:** Private files, explicit activation, child-setting clearing, and fixed-label failure badges.
   - **ARCH-ORDER — flag:** Explicit phases exist, but authoritative transitions do not pass through the required pure model.
   - **ARCH-FUNERAL — flag:** Per-launch residue remains unbounded across sessions.

   The original Core concepts entities exist at their stated locations; no PURE-versus-INTEGRATION contradiction was established.

7. **Plan revision recommendations**
   - Append a dated `## Revisions` entry replacing operator-only retention with the selected enforced policy and its tests.
   - Append an entry naming the pure lifecycle component, events/effects, integration boundary, and deterministic sequence coverage.
   - Include README coverage in completion tasks.

```findings
findings:
  - id: new
    severity: Important
    family: user-surface-readme-coverage
    title: |
      README omits the new live-capture configuration
    detail: |
      cmd/internal/couchcmd/capture.go:40 introduces COUCH_CAPTURE_DIR and COUCH_CAPTURE_MAX_MIB, but README.md is unchanged in the pinned range. Document activation, limits and stopped-state behavior there, linking to the atlas runbook.
  - id: new
    severity: Important
    family: durable-artifact-aggregate-retention
    title: |
      Per-session caps leave accumulated capture storage unbounded
    detail: |
      cmd/internal/terminalcapture/recorder.go:106 creates a fresh session directory on every enabled launch, and atlas/couch-live-capture.md:90 explicitly requires operator deletion. ARCH-FUNERAL requires more than manual cleanup for per-session growth: enforce an aggregate admission budget or automatic retention with incident-evidence protection, and test repeated launches.
  - id: new
    severity: Important
    family: authoritative-pure-lifecycle-transitions
    title: |
      Recorder lifecycle transitions remain embedded in the IO shell
    detail: |
      cmd/internal/terminalcapture/recorder.go:177 changes phases while closing channels, and :299 independently changes phase from the writer loop. Under ARCH-ORDER and ARCH-PURE, route authoritative state changes through an encapsulated pure state/event transition component; exercise failure, repeated close, timeout and late worker completion through that production model.
```
