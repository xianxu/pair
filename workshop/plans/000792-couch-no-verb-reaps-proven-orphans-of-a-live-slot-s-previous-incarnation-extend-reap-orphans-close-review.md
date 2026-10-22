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

---

## Re-review — 2026-10-22T04:10:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b605b1ee4ec9bda5a4d569708d25e0a857efd2f..c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934 |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T04:10:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the revised Spec and Plan. BR-2 is addressed across historical evidence loss and outstanding invocations: both legacy families now revalidate each signal at the supervisor. Independent mutation tests confirmed that removing either protection makes the regressions fail. No additional findings. The separately required recorded SDLC verification remains pending.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      resources.go:261–272 latches incomplete ownership evidence; reap_legacy_signal.go:49–88 checks that latch under the effect gate through identity rereads and signalling. Sequence and late-loss regressions pass. In disposable pinned-head copies, removing the latch reproduced signals to protected PIDs 11/12; removing the effect-time check reproduced late TERM/KILL signals across both legacy families. Both disposable copies were removed.
  - id: BR-1
    disposition: addressed
    note: |
      Protection still precedes lossy projection in resources_projection.go. Projection-to-CLI regressions pass for both legacy families, including incomplete witnesses and repeated projection.
  - id: BR-3
    disposition: addressed
    note: |
      agentfixture/resources_test.go:41–70 retains a short private socket root independent of TMPDIR, with cleanup. Its deliberately long-TMPDIR regression passes.
```

1. **Strengths**

   - [The legacy effect gate](/Volumes/TestScratch/review-checkouts/0b9d11461d29653b/cmd/internal/couchcmd/reap_legacy_signal.go:49) revokes outstanding plans, including escalation, rather than relying on admission-time evidence.
   - The exact executor uses an explicit pure transition model; uncertain acknowledgments and probes cannot produce success or automatic escalation.
   - Stateful composition tests exercise the actual sampler, service, CLI and signal boundary with controllable ordering.
   - README, atlas and embedded command documentation explain the changed compatibility, authority and recovery behavior.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Full tests passed for `couchcore`, `couchmessage`, `launcher`, `couchcmd` and `testcouch/...`.
   - Focused race tests passed for ownership, startup ordering, exact execution, legacy revocation and protocol behavior.
   - Artifact inventory guard and diff whitespace checks passed.
   - Both independent protection-removal mutations failed the expected regressions.
   - Verified retained Claude/Codex native acceptance logs against their hashes and inspected their assertions. Their tested production source is identical to the pinned head; native acceptance was not rerun here.

6. **Architectural notes**

   | Principle | Result |
   |---|---|
   | ARCH-DRY | Pass — shares ownership classification and executor cadence. |
   | ARCH-PURE | Pass — decisions and transitions have direct tests without IO; effects use injected boundaries. |
   | ARCH-PURPOSE | Pass — protects exact targets and both legacy families across evidence-loss sequences. |
   | ARCH-MOCK | Pass — stateful doubles share production boundaries; native conformance exists. |
   | ARCH-CONSTRAINTS | Pass — bounded censuses, requests and waits; counted growth tests cover added work. |
   | ARCH-SECURE | Pass — caller claims require current authority, provenance and generation checks. |
   | ARCH-ORDER | Pass — explicit transitions, controlled interleavings and synchronized revocation. |
   | ARCH-FUNERAL | Pass — invocation state expires; fixture processes and artifacts have owned cleanup. |
   | ARCH-EVOLVE | Pass — wire additions introduce no persisted-format change. |

   The authoritative Plan revisions reconcile the concept locations and revised legacy contract with the implementation.

7. **Plan revision recommendations:** None. Keep the recorded-verification checklist incomplete until its required check passes.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"044472a5-e968-48cc-8e82-952295e581cd","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0b9d11461d29653b","reviewed_head":"c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934","slot":"pair:2","requested_at":"2026-10-22T11:03:53.822313Z"},"phase":"finished","pid":46193,"started_at":"2026-10-22T11:03:53.972037Z","finished_at":"2026-10-22T11:10:21.302608Z","session_id":"01a16348-22f2-7b62-8548-95b9bb096a14","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.905473042s
head: "c8a9316a9bf50d75c1f9d5ee1ab58d0b9227f934"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T05:28:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | b19ef54ad75547d53ebc0d04b043c00b95dce7a5..e6535a0c2c671bfeb3b1065ed279c269886829b0 |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T05:28:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly repairs two misplaced doc comments and removes an obsolete lint suppression. It introduces no runtime behavior, scope change, or new architectural surface. No open prior findings require disposition; BR-1, BR-2, and BR-3 remain disposed. Required recorded verification remains pending independently of this review.

```findings
{}
```

1. **Strengths**

   - Doc comments now describe their actual declarations: `cmd/internal/couchcmd/reap_orphans.go:64` and `cmd/internal/couchcmd/resources.go:65`.
   - The socket fixture retains its short private root and cleanup while removing only the unmatched suppression.
   - Tracker updates preserve the pending verification requirement.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed both complete affected packages, `internal/repolint` and `cmd/internal/termcmd`, plus the socket fixture’s long-TMPDIR regression. Diff whitespace checks passed. No runtime regression test is needed for the doc-comment moves.

6. **Architectural notes**

   - **ARCH-DRY: Pass** — no duplicated logic introduced.
   - **ARCH-PURE: Pass** — no logic or IO boundaries changed.
   - **ARCH-PURPOSE: Pass** — corrections address the reported lint failures without changing issue scope.
   - **ARCH-EVOLVE: Pass** — no persisted-format changes.

   README and atlas updates are unnecessary for this interdiff.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"37913029-3c6b-4085-b6a2-76647b793532","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"fix-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1d73ccafb3f1cabb","reviewed_head":"e6535a0c2c671bfeb3b1065ed279c269886829b0","slot":"pair:2","requested_at":"2026-10-22T12:27:34.501248Z"},"phase":"finished","pid":39518,"started_at":"2026-10-22T12:27:34.671482Z","finished_at":"2026-10-22T12:28:34.032819Z","session_id":"01a16394-befd-78a0-b55c-2d1b27382538","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.916854166s
head: "e6535a0c2c671bfeb3b1065ed279c269886829b0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T07:26:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 915cca376443ef2c0894af3541139203ede6a209..ea1bfa4cd5dce0acfe9b776a23b1ff71670b8bdf |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T07:26:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff contains documentation and tracker updates only. The atlas conflict resolution preserves main’s navigation and tab-bar description, while #792 remains documented in the linked Couch guide. Spec and Done when are unchanged; appended Revisions record verification and integration without changing scope. No open prior findings require disposition.

```findings
{}
```

1. **Strengths**
   - `atlas/index.md:19` matches integrated main and retains the Couch guide link.
   - `atlas/couch.md:5060` retains #792’s authority, exact-set scope, supervisor gates and partial-outcome documentation.
   - Tracker revisions preserve earlier evidence and append integration history.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Required stat, name-status and full diff inspections completed. `git diff --check` passed. No runtime tests were rerun because executable code is unchanged; recorded suite results were not independently reproduced.

6. **Architectural notes**
   - **ARCH-DRY: Pass** — feature detail remains in the existing guide.
   - **ARCH-PURE: Pass** — no logic or IO boundaries changed.
   - **ARCH-PURPOSE: Pass** — integration preserves feature documentation and approved scope.
   - **ARCH-EVOLVE: Pass** — tracker additions introduce no schema or field-meaning changes.

   No new user-facing surface requires a README update.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"da3a0e4b-db6d-4dd4-8376-300614e9fa25","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/453d1175c7c67afb","reviewed_head":"ea1bfa4cd5dce0acfe9b776a23b1ff71670b8bdf","slot":"pair:2","requested_at":"2026-10-22T14:25:30.084534Z"},"phase":"finished","pid":75287,"started_at":"2026-10-22T14:25:30.255801Z","finished_at":"2026-10-22T14:26:33.259254Z","session_id":"01a16400-b66a-7380-a4c9-e95e65456e13","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.060479083s
head: "ea1bfa4cd5dce0acfe9b776a23b1ff71670b8bdf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T09:19:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 792 — Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans |
| repo | pair |
| issue file | workshop/issues/000792-couch-no-verb-reaps-proven-orphans-of-a-live-slot-s-previous-incarnation-extend-reap-orphans.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0e4395b5246213c721c29c02423acd02ed764552..09b3cf61c67b73f7caf0e4ca9c2f46ef073e9b2b |
| command | sdlc close --issue 792 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T09:19:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly resolves both merge conflicts, preserving the editor endpoint service and orphan-reap process seam, plus both command-guide updates. Spec and Done when are unchanged; appended Revisions record integration and verification without expanding scope. No open prior findings require disposition.

```findings
{}
```

1. **Strengths**
   - `cmd/internal/couchcmd/message_service.go:169` retains both fields with reachable consumers: endpoint startup/shutdown and supervisor reap handlers.
   - `cmd/internal/couchcmd/skills/couch/SKILL.md:77` preserves `--editor-endpoint` and accurately describes TERM/KILL grace periods.
   - Tracker additions preserve previous evidence and record the integration history.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Focused editor endpoint, service lifecycle, proven-orphan, legacy-signal and projection/selection tests passed. `make test-affected-inventory` and range whitespace checks passed. Historical full-suite results were not independently rerun.

6. **Architectural notes**
   - **ARCH-DRY: Pass** — no duplicated behavior introduced.
   - **ARCH-PURE: Pass** — existing injected process-table seam remains intact.
   - **ARCH-PURPOSE: Pass** — both conflict resolutions preserve both sides’ intent.
   - **ARCH-EVOLVE: Pass** — no persisted-format interpretation or reader/writer changes.
   
   No new surface in this interdiff requires atlas or README updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"711166fa-9536-43a0-886c-e8ab0c991bc9","repository":"github.com/xianxu/pair","issue_id":"000792","recipe":"fix-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c1cc88c405c66cb9","reviewed_head":"09b3cf61c67b73f7caf0e4ca9c2f46ef073e9b2b","slot":"pair:2","requested_at":"2026-10-22T16:18:03.640531Z"},"phase":"finished","pid":45280,"started_at":"2026-10-22T16:18:03.819108Z","finished_at":"2026-10-22T16:19:00.924614Z","session_id":"01a16467-c37e-7610-bd3c-1892bb8e5882","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 6.743818958s
head: "09b3cf61c67b73f7caf0e4ca9c2f46ef073e9b2b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
