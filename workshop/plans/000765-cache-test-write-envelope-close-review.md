# Boundary Review — pair#765 (whole-issue close)

| field | value |
|-------|-------|
| issue | 765 — Cache-enabled Go tests exceed physical write envelopes |
| repo | pair |
| issue file | workshop/issues/000765-cache-test-write-envelope.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..7277dc3885c834b9dd3dfb80e9ee6f4b085f820e |
| command | sdlc close --issue 765 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:20:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation enforces the chosen accounting policy at the shared boundary, preserves write caps, and passes the targeted tests. One Important issue prevents close: the new repository-reading test is absent from the required reader inventory.

Done-when: current — the clauses still cover invocation attribution, both supported test modes, and continued enforcement of real-write caps.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: repository-reader-registration
    title: |
      Register the new conformance test before closing the issue
    detail: |
      cmd/internal/testdisk/cache_conformance_test.go:21–28 introduces a repository reader without updating SOURCE_GUARDS or tests/affected-go-readers.json. At the pinned head, make test-affected-inventory exits 2 with “unmapped reader: cmd/internal/testdisk/cache_conformance_test.go”; scripts/test-affected.py also runs this check before affected testing. This is the only unmapped reader reported in this window. Audit its repository/module inputs, register its test in SOURCE_GUARDS, record its audited inventory disposition, and rerun the required checks.
```

1. **Strengths**
   - `run.go:87` applies the policy once in the existing shared wrapper.
   - The conformance test proves actual cache behavior using execution markers, with an unguarded caching control and coverage verification.
   - Native tests retain real excess-write refusal and failure-status checks.
   - README and atlas both document the intentional result-cache tradeoff.

2. **Critical findings:** None.

3. **Important findings:** Reader registration failure above.

4. **Minor findings:** None.

5. **Test coverage notes:** Independently ran `go test -count=1 ./cmd/internal/testdisk`: passed in 10.242s on Darwin+cgo. Diff whitespace checks passed. Inspected retained full-package logs showing core at 100,298,752/134,217,728 bytes and tty at 39,800,832/67,108,864 bytes. Full uncached core/tty qualification and the recorded affected check remain unconfirmed at this boundary.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared wrapper owns policy for all seven budgeted package callers.
   - **ARCH-PURE: pass** — flag mutation stays in the integration boundary; pure accounting remains directly tested.
   - **ARCH-PURPOSE: pass** — trace instrumentation is excluded while real writes remain capped.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

7. **Plan revision recommendations:** None; complete and record the outstanding validation items after repairing registration.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ee647810-7427-4c16-94e3-87b99a18fc1e","repository":"github.com/xianxu/pair","issue_id":"000765","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0115220ea38644f9","reviewed_head":"7277dc3885c834b9dd3dfb80e9ee6f4b085f820e","slot":"pair:7","requested_at":"2026-10-21T10:19:15.394078Z"},"phase":"finished","pid":11978,"started_at":"2026-10-21T10:19:15.516661Z","finished_at":"2026-10-21T10:20:41.463069Z","session_id":"01a15df8-e806-7603-af4a-65b1e863c5f5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.101036166s
head: "7277dc3885c834b9dd3dfb80e9ee6f4b085f820e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
