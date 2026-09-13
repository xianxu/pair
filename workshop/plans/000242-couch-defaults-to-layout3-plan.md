# Couch default layout3 implementation plan

> Follow AGENTS.md Section 3 for execution. One atomic change, one close review.

**Goal:** Bare couch uses the operator's layout3 workbench; explicit layout2 remains supported.
**Architecture:** Share a DefaultLayout constant between Couch.New and ParseCLI. Legacy record normalization remains Layout2. Reuse the existing layout-conflict guard and stateful test runtimes.
**Tech Stack:** Go, existing CLI/PTY tests, Markdown docs.

## Core concepts

| Pure entity | Lives in | Status |
| --- | --- | --- |
| DefaultLayout | cmd/internal/couchcore/layout.go | new constant, Layout3 |
| ParseLayout / NormalizeLayout | cmd/internal/couchcore/layout.go | reused; empty remains Layout2 |
| layoutRemedy | cmd/internal/couchcore/layout.go | modified wording; existing guard unchanged |
| ParseCLI | cmd/internal/couchcmd/cli.go | modified default |

| Integration | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| New | cmd/internal/couchcore/couch.go | modified default | existing stateful runtime and store |
| runTypedOperationWithConsole | cmd/internal/couchcmd/run.go | reused | CLI-to-Couch launch, tested through newRT |

ARCH-DRY: one process-default constant, separate from record provenance.
ARCH-PURE / ARCH-MOCK: preserve parser/guard purity and reuse existing fake runtimes.
ARCH-PURPOSE: cover both constructor and CLI plus emitted argv/layout witness.
ARCH-SECURE: absent legacy fields remain layout2; unreadable values still refuse.
ARCH-ORDER: process layout stays immutable; existing startup guard runs before launch.
ARCH-CONSTRAINTS: constant selection adds no IO, loops, latency or concurrency.
ARCH-FUNERAL: creates no new persisted artifacts or background work.

## Plan

- [x] Update cmd/internal/couchcmd/layout_cli_test.go first: bare/path/dash-path default Layout3, explicit Layout2 and Layout3 reach Couch, emitted cold argv and record witness agree. Add constructor-default and legacy-normalization/conflict checks in existing couchcore layout tests; assert conflict remedy offers staying on layout2 or parking.
- [x] Run focused tests and observe failures: go test ./cmd/internal/couchcmd ./cmd/internal/couchcore -run 'Layout|ProjectionNormalizes' -count=1.
- [x] Add DefaultLayout = Layout3 in layout.go; use it in New and ParseCLI. Preserve ParseLayout empty behavior. Reword layoutRemedy's single-known-layout case to explicitly offer keeping that layout or parking before changing.
- [x] Update legacy comments in actionableinventory.go and layout_projection_test.go; help in run.go, README.md, atlas/couch.md. Make old-layout fixtures explicitly request Layout2 where that is the behavior under test; do not rewrite persisted legacy witnesses to the new default.
- [ ] Run focused tests, then full env -u PAIR_SESSION_ID -u PAIR_TAG make test and git diff --check. Build/install couch using the repository build target; verify help identifies layout3 as default. Do not restart the operator's live Couch.
- [ ] Close once with the binary-owned review, fix findings, publish via PR and merge.

## Revisions

### 2026-09-13 — concrete scope from source inspection

The issue named only New, but ParseCLI has a separate default that overrides it.
Fresh-eyes spec review confirmed this omission. Both now derive from DefaultLayout.
The user's request to work on this simple, already specified default flip authorizes
this implementation; no new product choice is introduced.

### 2026-09-13 — PQ-1: function-level test strategies

- ParseCLI: table-driven argv classes (implicit default, explicit override, positional placement, separator) with literal expected layouts and paths; catches an independent stale CLI default.
- New: real constructor through existing newTestEnv, assert literal Layout3 and exercise Spawn to compare exact child argv and stored witness; catches default/launch/persistence disagreement.
- ParseLayout / NormalizeLayout: absent, known, and unknown persisted values; assert absent still means Layout2 and unknown remains a refusal, independent of the new process default.
- ResolveLayoutConflicts / StartInteractive: hold a legacy detached session in the stateful fake, use default Couch, assert refusal before any runner effect; explicitly requesting Layout2 remains compatible.
- layoutRemedy: assert actionable output over single-known, mixed-known, and unreadable conflict classes using existing table tests; the single-known case names retaining its layout or parking before switching.

These strategies replace the case-only description in the first checklist row.
