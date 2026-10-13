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

## Open findings

- **BR-1** [Critical] `preserve-observation-uncertainty` Unknown terminal observations become live or actionable relaunch verdicts
- **BR-2** [Critical] `exact-process-generation-authority` Agent usability accepts a recycled agent PID beneath the same server
- **BR-3** [Important] `bounded-untrusted-artifact-reads` Policy and readiness reads can defeat the admission timeout
- **BR-4** [Important] `durable-artifact-lifecycle-bound` Per-terminal coordination files accumulate without an aggregate lifecycle bound
