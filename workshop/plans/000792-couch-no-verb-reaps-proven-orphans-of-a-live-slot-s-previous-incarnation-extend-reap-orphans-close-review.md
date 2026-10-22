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

---

## Re-review — 2026-10-22T02:54:12-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..03d0c96e4f24b3ce6fcf82df47f334b573929766 |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T02:54:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1’s projection fix is verified, including a failing test with the fix removed. A separate upstream selection limit still discards orphan protection and permits legacy tree signalling. That safety defect blocks shipping. A new socket fixture also fails under the review environment’s long `TMPDIR`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      resources_projection.go filters both legacy families before truncation. The projection-to-CLI regressions pass at head; removing that filter in a disposable checkout makes all four overlap cases signal both the old PID and its current descendant. The disposable checkout was removed.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: incomplete-evidence-must-not-widen-authority
    title: |
      Selection truncation discards orphan protection before legacy admission
    detail: |
      reap_orphans.go:107 derives protection only from returned Orphans. Before projection, SelectResourceProcesses (resource.go:349-351) drops orphan rows when 8192 current-owned rows fill its selection, and AdvanceResources replaces remembered provenance with that truncated selection. A disposable regression through the real sampler, service and confirmed CLI produced zero orphan rows, retained the overlapping LeftRunning row, signalled old PID 11 and unproven child 12 through legacy reaping, and exited 0 despite coverage errors. This is the 2nd finding in family incomplete-evidence-must-not-widen-authority. Earlier rounds fixed instances; do not fix only this instance. Enforce the rule that incomplete evidence never widens signalling authority, including across sampler passes. Sweep table/slot limits and incomplete census, selection truncation/conflicts/unknown classification, cached/RPC projection, CLI admission and both legacy families. Preserve conservative exclusions independently of lossy selection, refusing when protection cannot be retained, and add sequence regressions.
  - id: new
    severity: Important
    kind: defect
    family: fixtures-must-enforce-platform-bounds
    title: |
      New RPC fixture exceeds the Unix socket path limit with long TMPDIR
    detail: |
      agentfixture/resources_test.go:42 creates its isolated root beneath inherited TMPDIR. In this review environment, TestResourceObservationUsesExistingPrivateRPCAndCallerCancellation fails at line 59 with “isolated message socket exceeds 103 bytes”; the same test passes with TMPDIR=/tmp. ARCH-CONSTRAINTS: use a short private fixture root, following couchmessage/transport_test.go:23, retain cleanup, and verify under a long TMPDIR.
```

1. **Strengths**

   - Supervisor effects recheck caller authority, current ownership and exact process identity before each signal.
   - The exact-reap reducer prevents escalation after uncertain replies and retains partial progress.
   - README, atlas and embedded skill document authority, exact scope and recovery after uncertainty.

2. **Critical findings**

   - **ARCH-PURPOSE / ARCH-SECURE / ARCH-CONSTRAINTS:** [legacy admission](/Volumes/TestScratch/review-checkouts/02e5fea098bb3fd7/cmd/internal/couchcmd/reap_orphans.go:107) treats absence from a truncated orphan report as permission for broader cleanup. Protection must survive upstream selection loss as well as response projection.

3. **Important findings**

   - **ARCH-CONSTRAINTS:** [socket fixture root](/Volumes/TestScratch/review-checkouts/02e5fea098bb3fd7/cmd/internal/testcouch/agentfixture/resources_test.go:42) depends on inherited temporary-path length.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Targeted proof, protocol, executor, service, CLI and ownership tests passed, including race runs.
   - Full `couchmessage`, `launcher` and `artifactpath` suites passed.
   - Removing BR-1’s fix produced the expected unsafe signals.
   - The new selection-cap regression failed against pinned head with signals to PIDs 11 and 12.
   - Native Claude/Codex acceptance was inspected but not rerun. Recorded verification remains pending in the tracker.
   - Disposable checkouts were removed; the reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared classifier, ownership collection and cadence reused.
   - **ARCH-PURE — pass:** decisions and invocation transitions are tested without IO; effects are injected.
   - **ARCH-PURPOSE — flag:** selection omission bypasses exact-set protection.
   - **ARCH-MOCK — pass:** stateful process worlds exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** selection overflow widens authority; socket fixture exceeds platform bounds.
   - **ARCH-SECURE — flag:** missing projected ownership evidence permits legacy signalling.
   - **ARCH-ORDER — flag:** truncated provenance replaces prior protection across sampler passes; the exact-reap invocation model itself passes.
   - **ARCH-FUNERAL — pass:** invocation state is scoped; fixture residue belongs to fleet cleanup.
   - **ARCH-EVOLVE — pass:** no production persisted-format change; wire additions are additive.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending the omission invariant across classification, provenance replacement, projection and CLI admission. Require overflow-and-recovery sequence tests proving that protection persists until authoritative evidence establishes the original generation is gone.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5cd3e3bb-e9a6-4903-9dd3-e7a96e39c4df","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/02e5fea098bb3fd7","reviewed_head":"03d0c96e4f24b3ce6fcf82df47f334b573929766","slot":"pair:2","requested_at":"2026-10-22T09:48:06.682902Z"},"phase":"finished","pid":37480,"started_at":"2026-10-22T09:48:06.827115Z","finished_at":"2026-10-22T09:54:11.4431Z","session_id":"01a16302-c0e2-7523-847c-b3a427221d31","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.736437667s
head: "03d0c96e4f24b3ce6fcf82df47f334b573929766"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T03:37:34-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..aeb3e97b256c63c2f6e5abc853c625258631dc2f |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T03:37:34-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The exact-set executor, fresh ownership checks, and conservative provenance tracking are well separated and substantially tested. BR-3 is verified fixed. BR-2 remains blocking: a confirmed invocation can continue legacy signalling after its own exact-effect proof sets the supervisor’s lifetime refusal flag.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Pre-projection overlap exclusion remains present for both legacy families; the projection regressions pass. No reopening.
  - id: BR-2
    disposition: not-addressed
    note: |
      cmd/internal/couchcmd/reap_orphans.go:101 snapshots legacyRefused before executing exact effects at line 126, then uses that stale value to authorize both legacy families at lines 130–131. A disposable pinned-head regression admitted a healthy classification, injected census failure before the exact signal request, and confirmed the real sampler latched legacy refusal. The exact target was refused, but legacy nvim PID 51 and registered PID 52 both received TERM and were reported reaped. ARCH-PURPOSE/ARCH-ORDER/ARCH-SECURE: the rule must cover outstanding invocations, not only classifications requested after evidence loss. Keep this existing family open; enforce revocation at the legacy effect boundary and sweep loss during exact dispatch, waits, and legacy execution across both families.
  - id: BR-3
    disposition: addressed
    note: |
      agentfixture/resources_test.go:41–70 creates and cleans a short private socket root independently of inherited TMPDIR. The regression passes with its deliberately long TMPDIR; replacing the short-root call with t.TempDir in a disposable checkout makes the same test fail with “isolated message socket exceeds 103 bytes”.
```

1. **Strengths**

   - Exact reaping never discovers descendants; supervisor effects recheck authority, provenance, and process identity.
   - The pure transition reducer preserves uncertainty and prevents automatic escalation after unacknowledged TERM.
   - Overflow/recovery sequence tests exercise the real sampler, service, CLI, and both legacy families.
   - README and atlas updates document the new scope, authority rules, refusal, and recovery behavior.

2. **Critical findings**

   - **BR-2 remains open**, at `cmd/internal/couchcmd/reap_orphans.go:101–131`. A historical classification response remains actionable after the supervisor observes provenance loss. Fix the revocation rule across outstanding operations; merely refreshing the flag once would leave another observation-to-effect race.

3. **Important findings**

   - None additional. BR-3 is addressed with demonstrated red/green evidence.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Targeted core, CLI, protocol, executor, ownership, and fixture tests passed.
   - Relevant CLI/executor race tests passed.
   - Complete `couchmessage` and `agentfixture` package tests passed.
   - Disabling the sticky update made existing overflow/recovery regressions fail, confirming meaningful coverage.
   - The added disposable late-loss regression failed on pinned head, demonstrating the remaining BR-2 gap.
   - Disposable checkout removed; reviewed checkout unchanged. Native acceptance was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared classifier and completeness decision.
   - **ARCH-PURE — pass:** proof decisions and invocation transitions are directly tested without IO.
   - **ARCH-PURPOSE — flag:** BR-2 leaves the incomplete-evidence rule unenforced for outstanding invocations.
   - **ARCH-MOCK — pass:** stateful process worlds share production seams.
   - **ARCH-CONSTRAINTS — pass:** bounded observations, counted collection tests, and short socket fixtures.
   - **ARCH-SECURE — flag:** stale admission remains destructive authority after evidence loss.
   - **ARCH-ORDER — flag:** supervisor refusal transitions do not revoke already-returned legacy plans.
   - **ARCH-FUNERAL — pass:** invocation state is transient; fixture artifacts have cleanup.
   - **ARCH-EVOLVE — pass:** additive protocol surface; no new persisted format.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying how lifetime refusal revokes outstanding legacy operations, where that rule is enforced beside effects, and sequence tests covering evidence loss during exact dispatch, waits, and execution of either legacy family.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"15dc2879-fa83-4221-8abd-c5e1be9cca5c","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9f6bb5d4834b8b2c","reviewed_head":"aeb3e97b256c63c2f6e5abc853c625258631dc2f","slot":"pair:2","requested_at":"2026-10-22T10:34:49.00592Z"},"phase":"finished","pid":5207,"started_at":"2026-10-22T10:34:49.137164Z","finished_at":"2026-10-22T10:37:33.456253Z","session_id":"01a1632d-833a-7f00-9c7a-fce83ee674ec","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.419437459s
head: "aeb3e97b256c63c2f6e5abc853c625258631dc2f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
