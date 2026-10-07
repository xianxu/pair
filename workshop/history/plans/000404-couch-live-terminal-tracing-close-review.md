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

---

## Re-review — 2026-10-07T12:12:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 404 — Opt-in Couch live terminal tracing |
| repo | pair |
| issue file | workshop/issues/000404-couch-live-terminal-tracing.md |
| boundary | whole-issue close |
| milestone | — |
| window | b933b5a5bfef7fb681c5dffa2fd39c40d8765e7d..2df93faba69b5bc13754b75859bc9db76b4a6dee |
| command | sdlc close --issue 404 |
| reviewer | codex |
| timestamp | 2026-10-07T12:12:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised capture contract and addresses all three prior findings. Capture remains explicitly enabled, bounded, and observable without changing terminal write results. Focused tests and race checks passed. One unrelated mouse-mode test failed intermittently and reproduced against the verified pinned-base snapshot.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:1211–1232 documents activation, configuration limits, stopped-state badges, aggregate admission and the atlas runbook. These match captureSettings, captureBadge and storage admission.
  - id: BR-2
    disposition: addressed
    note: |
      Recorder.Open calls budgeted storage admission. Reservations survive completed sessions; byte/count limits and concurrent admission preserve existing evidence. Tests cover repeated launches, exhaustion, legacy/corrupt metadata and public Open wiring.
  - id: BR-3
    disposition: addressed
    note: |
      lifecycle.go owns authoritative phase/error transitions; Recorder executes its effects. Production-model sequence tests cover repeated close, failure, timeout, completion-before-timeout and late completion, supplemented by blocked-writer integration tests.
```

1. **Strengths**

   - Endpoint observations occur under the parser/resize mutex, preserving byte and geometry ordering.
   - Host capture retains partial-write receipts and preserves the underlying write result.
   - Storage admission conservatively rejects unknown evidence and reserves capacity without deleting recordings.
   - Coalesced status notifications repaint through the existing Presenter, including idle and pre-Run failures.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Capture, observation, lifecycle and real-PTY tests passed across the affected packages.
   - Focused race checks passed in `terminalcapture`, `couchcmd` and `couchtty`.
   - Child-environment clearing passed; range whitespace checks passed.
   - `TestCouchParentCaptureStableAcrossChildMouseModes` failed once, passed ten repetitions, and reproduced on the base snapshot. Its baseline sources were verified byte-for-byte against the pinned base.
   - Regression assertions exercise production admission and lifecycle decisions. No mutation experiment was performed during this read-only review. The full repository suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared configuration validation and existing Presenter/terminal seams are reused.
   - **ARCH-PURE — pass:** lifecycle decisions are separated from channel, clock and filesystem effects.
   - **ARCH-PURPOSE — pass:** both boundaries, startup bursts and persistent failure visibility are delivered; #379 remains unresolved.
   - **ARCH-MOCK — pass:** stateful fault sinks and host fixtures exercise shared production boundaries; real PTYs supplement them.
   - **ARCH-CONSTRAINTS — pass:** queue, stream size, session count and shutdown waiting are bounded and tested.
   - **ARCH-SECURE — pass:** private files, strict metadata parsing, confinement checks and fixed status labels protect the relevant boundaries.
   - **ARCH-ORDER — pass:** authoritative lifecycle transitions and deterministic failure/late-completion sequences are exercised.
   - **ARCH-FUNERAL — pass:** persistent reservations bound repeated-launch residue; explicit archival/removal releases capacity without automatic evidence deletion.

7. **Plan revision recommendations:** None. Appended revisions reconcile the implemented lifecycle, storage admission and configuration limits.
