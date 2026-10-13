---
gate: boundary-review
issue: 528
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T11:01:13-07:00"
      dispatched: "2026-10-13T10:58:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Unknown terminal observations become live or actionable relaunch verdicts
          detail: cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
          family: preserve-observation-uncertainty
          round: 1
        - id: BR-2
          severity: Critical
          title: Agent usability accepts a recycled agent PID beneath the same server
          detail: cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
          family: exact-process-generation-authority
          round: 1
        - id: BR-3
          severity: Important
          title: Policy and readiness reads can defeat the admission timeout
          detail: cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
          family: bounded-untrusted-artifact-reads
          round: 1
        - id: BR-4
          severity: Important
          title: Per-terminal coordination files accumulate without an aggregate lifecycle bound
          detail: cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.
          family: durable-artifact-lifecycle-bound
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T11:34:12-07:00"
      dispatched: "2026-10-13T11:30:13-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.
          round: 2
      findings:
        - id: BR-5
          severity: Important
          title: Capture-witness and admission-config reads still permit indefinite blocking
          detail: 'cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.'
          family: bounded-untrusted-artifact-reads
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T12:06:16-07:00"
      dispatched: "2026-10-13T12:03:38-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: dfedd894650cbf1f3cc2dbdb5782b8724131f01d
      blocked: false
---

# Gate ledger — pair#528 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T11:01:13-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `preserve-observation-uncertainty` Unknown terminal observations become live or actionable relaunch verdicts
  cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
- **BR-2** [Critical] `exact-process-generation-authority` Agent usability accepts a recycled agent PID beneath the same server
  cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
- **BR-3** [Important] `bounded-untrusted-artifact-reads` Policy and readiness reads can defeat the admission timeout
  cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
- **BR-4** [Important] `durable-artifact-lifecycle-bound` Per-terminal coordination files accumulate without an aggregate lifecycle bound
  cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.

## Round 2 — 2026-10-13T11:34:12-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
- BR-2 — addressed — Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
- BR-3 — addressed — Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
- BR-4 — addressed — TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.

### Raised

- **BR-5** [Important] `bounded-untrusted-artifact-reads` Capture-witness and admission-config reads still permit indefinite blocking
  cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.

## Round 3 — 2026-10-13T12:06:16-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
- BR-2 — addressed — Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
- BR-3 — addressed — Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
- BR-4 — addressed — Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
- BR-5 — addressed — Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.

## Open findings

(none — every finding has been disposed)
