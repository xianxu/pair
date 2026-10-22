# Boundary Review — pair#792 (whole-issue close)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..dfd00c602b36faf734bc87b3964fe139792d376a |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T02:40:16-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The exact-set executor, per-effect authority checks, and uncertainty handling are well separated and covered by focused tests. One blocking safety defect remains: response truncation can remove a proven orphan’s protection while retaining its overlapping legacy entry, allowing tree reaping without the new guards.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: incomplete-evidence-must-not-widen-authority
    title: |
      Response projection can route proven orphans through unguarded legacy tree reaping
    detail: |
      cmd/internal/couchcmd/message_service.go:980 projects the confirmed sweep before CLI admission. projectResourceView independently truncates Orphans and LeftRunning (resources_projection.go:66-72), but provenOrphanAdmission protects only PIDs in the received Orphans rows (reap_orphans.go:340-341). With enough orphan rows to exceed their section budget and a single LeftRunning entry overlapping an omitted orphan, that entry survives projection and reaches reapLeftRunning at line 123. It then uses local expanding-tree reaping without owning-hub or current-ownership checks, potentially signalling a foreign orphan and current or unproven descendants. ARCH-PURPOSE, ARCH-SECURE, and ARCH-CONSTRAINTS require omission to preserve protection. Remove all orphan overlaps from legacy families on the supervisor's complete view before projection, or preserve an independently complete exclusion set and refuse when it cannot fit. Add a regression through actual projection and CLI execution that proves zero legacy signals for an omitted overlapping orphan.
```

1. **Strengths**

   - `couchcore/resource_reap.go` reuses existing provenance and requires exact ownership witnesses.
   - `couchcmd/reap_proven.go` reauthenticates callers, serializes hub moves against signalling, and checks final process identity.
   - `launcher/exact_reap_model.go` explicitly models progress; lost replies and unknown observations cannot authorize escalation.
   - README, atlas, and embedded Couch guidance document authority, exact scope, partial results, and uncertainty recovery.

2. **Critical findings**

   - **Projection bypass**, detailed above, at `cmd/internal/couchcmd/message_service.go:980` and `cmd/internal/couchcmd/reap_orphans.go:115`. Perform overlap exclusion before lossy projection and test the complete service-to-CLI path with an omitted overlapping target.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused core, protocol, executor, service, and CLI tests passed, including race detection.
   - Legacy reaper and resource-projection tests passed.
   - Full artifactpath tests passed.
   - Selected shared-harness tests passed. Agentfixture tests passed with `TMPDIR=/tmp`; the initial default-path run exceeded the Unix socket path limit.
   - Existing overlap regressions cover visible and incomplete witnesses, but omit the projection interaction identified above.
   - Opt-in Claude/Codex native acceptance was inspected, not rerun. Repository files remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared classifier and ownership collection avoid parallel proof logic.
   - **ARCH-PURE — pass:** target decisions and invocation transitions are directly tested without IO.
   - **ARCH-PURPOSE — flag:** truncation bypasses the exact-set safety contract.
   - **ARCH-MOCK — pass:** stateful process worlds exercise production seams; disposable native conformance is provided.
   - **ARCH-CONSTRAINTS — flag:** bounded projection can widen destructive authority.
   - **ARCH-SECURE — flag:** omitted evidence permits a path without the required authority and ownership checks.
   - **ARCH-ORDER — pass:** explicit transitions preserve partial progress and prevent automatic escalation after uncertainty.
   - **ARCH-FUNERAL — pass:** production adds no durable reap state; fixture resources have cleanup ownership.
   - **ARCH-EVOLVE — pass:** no production persisted-format change; protocol additions retain strict operation shapes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring overlap protection from the complete supervisor view before any lossy projection. Extend the CLI composition strategy with a bounded-response case where an omitted orphan overlaps a retained legacy entry.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9d544d22-8f43-4561-97a1-2a6e3417385f","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/60f67f47a9451408","reviewed_head":"dfd00c602b36faf734bc87b3964fe139792d376a","slot":"pair:2","requested_at":"2026-10-22T09:36:55.914965Z"},"phase":"finished","pid":22555,"started_at":"2026-10-22T09:36:56.087931Z","finished_at":"2026-10-22T09:40:13.193829Z","session_id":"01a162f8-84e2-7e13-9ab9-29b95aa08559","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.338816417s
head: "dfd00c602b36faf734bc87b3964fe139792d376a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
