# Boundary Review — pair#528 (whole-issue close)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4780a55286f5a171d64907cf03d9c12c7a2ce656..faa2056195acaec505029ab5fbf7e31431c2857f |
| command | sdlc close --issue 528 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T20:18:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range matches #528’s final approved M1 scope: listed-live attachment, bounded retries, background client attachment, independent messaging, and guarded dead-client repair. README and atlas document the changed behavior. No new findings; no previously disposed finding is reopened. The remaining recovery work is explicitly assigned to #556/#557.

```findings
{}
```

1. **Strengths**
   - Continue bypasses cold-launch and lifecycle-adoption prerequisites while rechecking the exact listed terminal.
   - Automatic broker reads use preview access without replaying pending journals; regression fixtures verify unchanged durable bytes.
   - Dead-client repair rechecks revision and exact process identity before publication.
   - Final fixture corrections use atomic publication and preserve existing assertion deadlines.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** All 15 exercised package suites passed, including couchcore (383.787s), couchcmd, couchtty, launcher, command entry points, and storage/native-evidence packages. Startup and pane-independent reconnect regressions each passed ten runs; shutdown passed twenty. Focused race checks passed. Production/docs whitespace checks passed. Live external conformance and the complete `make test` were not rerun.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared projection, attachment, and metadata-reader paths.
   - **ARCH-PURE — pass:** pure parsing/ranking/reducers remain separate from IO adapters.
   - **ARCH-PURPOSE — pass:** implementation follows the explicitly revised M1 scope.
   - **ARCH-MOCK — pass:** stateful terminal/process doubles exercise shared production seams; live conformance hooks remain.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, retries, scans, and attachment concurrency.
   - **ARCH-SECURE — pass:** persisted inputs receive structural, identity, and regular-file checks.
   - **ARCH-ORDER — pass:** tests exercise cancellation, changed observations, late completions, and effect-time revalidation.
   - **ARCH-FUNERAL — pass:** lock files have a fixed 256-stripe bound; retained recovery evidence uses existing ownership and retention.

7. **Plan revision recommendations:** None. Existing revisions record the scope split and superseded designs.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.893210958s
head: "faa2056195acaec505029ab5fbf7e31431c2857f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
