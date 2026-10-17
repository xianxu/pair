# Boundary Review — pair#622 (milestone M1)

| field | value |
|-------|-------|
| issue | 622 — Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not |
| repo | pair |
| issue file | workshop/issues/000622-couch-server-spends-1-core-emulating-every-hosted-client-s-output-in-a-vt-parser-visible-or-not.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b95df08cebf86cc71cd86a27dfdf2fdb96f1b91d..177c4febc9aeba5ea2924f7f5c6b9cc803b0e9ed |
| command | sdlc milestone-close --issue 622 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T18:05:54-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the documented M1 diagnostics scope without changing terminal parsing policy. Counters, fresh snapshots, bounded projection, transport, and documentation agree with the plan. No blocking defects found. Whole-issue CPU reduction and live consumer acceptance remain explicitly owed in M2.

```findings
{}
```

1. **Strengths**

   - Counter admission precedes parser locking and failure paths, matching the documented cumulative-byte semantics (`cmd/internal/ptychild/terminal.go:51`).
   - Snapshots preserve client identity, exclude exited children, cap retained rows, and report omissions without acquiring the parser mutex (`cmd/internal/couchtty/console_resources.go:47`).
   - Resource requests attach fresh diagnostics without mutating cached process samples (`cmd/internal/couchcmd/message_service.go:794`).
   - Tests exercise actual Child/Console behavior and socket/CLI transport, including replacement, retention eviction, failed publication, concurrent totals, sanitization, and wire capacity.
   - README and atlas document the new surface, identity requirements, freshness, omissions, and measurement limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Targeted race tests across `couchcmd`, `couchtty`, and `ptychild`.
   - All `TestResource*` tests in `couchcmd` under the race detector.
   - Complete `ptychild` package tests.
   - `git diff --check`.

   The parser-lock barrier provides controlled ordering evidence. Projection and clone tests check identity preservation and isolation. The full repository suite and live fleet measurements were not independently rerun; M2 measurements remain outside this boundary.

6. **Architectural notes for upcoming work**

   | Principle | Result | Evidence |
   |---|---|---|
   | ARCH-DRY | Pass | Reuses existing inventory, process identity, and projection helpers. |
   | ARCH-PURE | Pass | Projection and records remain separate from counters, Console locking, and request IO. |
   | ARCH-PURPOSE | Pass | Delivers authorized M1 measurement infrastructure; preserves M2 acceptance obligations. |
   | ARCH-MOCK | Pass | Stateful Child/Host doubles share production seams; real PTY and socket checks supplement them. |
   | ARCH-CONSTRAINTS | Pass | One atomic addition per ingest, bounded snapshot rows, encoded-byte projection, no added worker or clock call. |
   | ARCH-SECURE | Pass | Adds identities and totals without payload retention; text labels are sanitized. |
   | ARCH-ORDER | Pass | Atomic lifetime totals and existing Console lifecycle synchronization preserve identity/count association; controlled tests cover replacement and contention. |
   | ARCH-FUNERAL | Pass | Counter dies with Child; snapshots are request-scoped; no new persistent history or background lifecycle. |

   Core-concept entities exist at their stated locations and match their classifications. M2 should retain the documented ordering, protocol-effect, and memory constraints when selecting an optimization.

7. **Plan revision recommendations:** None. Outstanding landing tasks appropriately remain unchecked.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c3a74782-1746-4a47-9bb8-2ad4f11d5cd2","repository":"github.com/xianxu/pair","issue_id":"000622","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0bd982d8a0af4713","reviewed_head":"177c4febc9aeba5ea2924f7f5c6b9cc803b0e9ed","slot":"pair:5","requested_at":"2026-10-17T01:04:08.577095Z"},"phase":"finished","pid":13098,"started_at":"2026-10-17T01:04:08.828772Z","finished_at":"2026-10-17T01:05:37.808171Z","session_id":"01a14763-4312-7043-a289-17be55a9c10f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.775006916s
head: "177c4febc9aeba5ea2924f7f5c6b9cc803b0e9ed"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
