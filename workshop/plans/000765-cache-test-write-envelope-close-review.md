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

---

## Re-review — 2026-10-21T03:32:17-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 765 — Cache-enabled Go tests exceed physical write envelopes |
| repo | pair |
| issue file | workshop/issues/000765-cache-test-write-envelope.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..7999245e29ebddadb321e849b30dba9a2bba8dfe |
| command | sdlc close --issue 765 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:32:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The shared boundary enforces the documented cache policy without changing write caps or failure semantics. BR-1 is verified addressed. No new findings; the checkout remains unchanged.

Done-when: current — the clauses cover attribution, cache-enabled and uncached invocation contracts, and continued refusal of excessive fixture writes.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      SOURCE_GUARDS registers TestGoCacheWriteBudgetConformance and the audited inventory contains its current digest. make test-affected-inventory passes; independently removing either registration in memory makes the existing audit fail with the expected diagnostic. The registered conformance test passes.
```

1. **Strengths**
   - `cmd/internal/testdisk/run.go:87` centralizes policy for all seven budgeted package callers.
   - `cache_conformance_test.go:88–119` verifies actual cache reuse, fresh guarded execution, and retained coverage.
   - Native tests preserve real excess-write refusal and failure status.
   - README and atlas explain the result-cache tradeoff.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Inventory check and diff whitespace check passed.
   - `go test -count=1 ./cmd/internal/testdisk` passed.
   - Replacing `run.go` with its base version through a temporary Go overlay makes the trace regression fail: 5,353,472 bytes against a 1 MiB cap.
   - Uncached core/tty tests passed: 270.586s / 53.470s.
   - Verified retained plain-run results and cold/warm attribution logs against recorded hashes.
   - The full required `make test` suite was not rerun during this review.

6. **Architectural notes**
   - **ARCH-DRY: pass** — one shared policy owner.
   - **ARCH-PURE: pass** — flag mutation stays at the integration boundary; accounting remains directly testable.
   - **ARCH-PURPOSE: pass** — excludes cache instrumentation while retaining workload caps.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9d9b11b2-0933-4769-8b3a-520a9c6f520e","repository":"github.com/xianxu/pair","issue_id":"000765","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/033ef061b6890bad","reviewed_head":"7999245e29ebddadb321e849b30dba9a2bba8dfe","slot":"pair:7","requested_at":"2026-10-21T10:26:35.798994Z"},"phase":"finished","pid":39688,"started_at":"2026-10-21T10:26:35.934892Z","finished_at":"2026-10-21T10:32:17.208198Z","session_id":"01a15dff-a080-7fd3-b7fe-d5d2d4f892c2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.14874225s
head: "7999245e29ebddadb321e849b30dba9a2bba8dfe"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T05:02:12-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 765 — Cache-enabled Go tests exceed physical write envelopes |
| repo | pair |
| issue file | workshop/issues/000765-cache-test-write-envelope.md |
| boundary | whole-issue close |
| milestone | — |
| window | b00543b68e2c2b84de08d453c7451fd684073c51..7a44ba96a8802861c3991ed9791bd55b549ed8cd |
| command | sdlc close --issue 765 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T05:02:12-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff preserves both sides of the inventory conflict without changing audited hashes or behavior. BR-1 remains addressed. Issue revisions record evidence and integration progress, with no scope expansion. No new defects found.

Done-when: current — the criteria still cover invocation accounting, cache behavior, and enforcement of real fixture-write caps.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The conformance reader retains its matching inventory hash and SOURCE_GUARDS registration. The inventory check and registered conformance test pass. Removing either registration in memory makes the audit fail.
```

1. **Strengths**
   - `tests/affected-go-readers.json:60–61` preserves both conflict-side entries and their original hashes.
   - Both readers remain registered in the fixed preflight.
   - Added revision entries distinguish branch validation from integration validation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - `make test-affected-inventory` passed.
   - Removing either reader’s inventory entry or preflight registration in memory triggered the expected rejection.
   - `TestGoCacheWriteBudgetConformance` passed in 9.119s.
   - Diff whitespace check passed. Full suite was not rerun during this narrow review.

6. **Architectural notes**
   - **ARCH-DRY: pass** — existing inventory and preflight owners remain authoritative.
   - **ARCH-PURE: pass** — no new logic or IO coupling.
   - **ARCH-PURPOSE: pass** — the resolution retains both readers’ enforcement.
   - **ARCH-EVOLVE: pass** — schema, existing entries, and field meanings are preserved.
   - No new architectural or user-facing surface requires atlas or README updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"8f812837-dca2-494e-af5e-63a4803ba2fa","repository":"github.com/xianxu/pair","issue_id":"000765","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1b86d6938dddef80","reviewed_head":"7a44ba96a8802861c3991ed9791bd55b549ed8cd","slot":"pair:7","requested_at":"2026-10-21T12:01:03.074385Z"},"phase":"finished","pid":981,"started_at":"2026-10-21T12:01:03.167655Z","finished_at":"2026-10-21T12:02:12.007717Z","session_id":"01a15e56-19be-7a20-8632-8b16eac9de71","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.502564542s
head: "7a44ba96a8802861c3991ed9791bd55b549ed8cd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
