# Boundary Review — pair#149 (milestone M1)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a271432590da8a4177fea6c523607182536861a2^..d16a00003a69850ffd0447dc4b7cd98fd29e9d69 |
| command | sdlc milestone-close --issue 149 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-26T11:42:03-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The storage, locking, policy-decoding, and conformance foundations are strong, and all Go tests pass. The boundary cannot close because admission equates death of the Pair/zellij client PID with death of the entire workspace-writing incarnation, despite the issue and committed probe proving the zellij session survives. Two additional contract gaps block the boundary: opaque allocation does not check tag-scoped artifacts, and the plan’s Core concepts table misclassifies the filesystem-backed namespace as PURE while naming an integration entity that does not exist.

## 1. Strengths

- The policy boundary is unusually defensive: duplicate keys, unknown fields, invalid tagged capacity shapes, stderr/exit disagreement, and oversized responses fail closed in [policyresolver.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/policyresolver.go:101).
- ThreadStore uses revision checks, exact before/after images, atomic replacement, and recoverable journals. The crash and third-state tests in [storejournal_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/storejournal_test.go:21) exercise real persistence logic rather than mocks.
- Provider IO occurs outside the ThreadStore lock, while snapshot images are revalidated before commit in [admission.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission.go:69).
- The local-policy shadow sweep removes the previous `PolicyTable`, repository `Mode`, `policy.json`, and admission bypasses from production Go code.
- The supervisor lease has meaningful subprocess coverage for crash release and close-on-exec behavior in [supervisorlease_subprocess_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/supervisorlease_subprocess_test.go:67).

## 2. Critical findings

### C1 — A dead Pair client incorrectly frees capacity while its zellij session remains live

[admission.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission.go:188) treats `Proc.Exists(incarnation.PID) == Dead` as proof that the incarnation is dead and prunes it. That PID is the Pair/zellij client spawned by Couch; the repository’s own [zellijpark probe](/Users/xianxu/workspace/pair/probes/zellijpark/main.go:1) establishes that the zellij session survives both SIGTERM and SIGKILL of that client. The Spec explicitly says a client exit must not free capacity while a zellij session or agent survives ([issue](/Users/xianxu/workspace/pair/workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md:161), [issue](/Users/xianxu/workspace/pair/workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md:182)).

Consequently, after Couch stops or crashes, a bounded-policy start can prune the old record and admit a second writer into the same checkout. The current test at [admission_reconcile_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission_reconcile_test.go:79) blesses the incorrect model.

Fix sketch: until #152 supplies whole-incarnation quiescence evidence, loss of the client PID must transition/remain conservative Unknown and occupied. Add a stateful test modeling “client dead, server-side session still live”; removing that protection must make the test fail. This flags ARCH-PURPOSE and ARCH-MOCK.

### C2 — Opaque allocation ignores collisions with existing scoped artifacts

The Spec requires an atomic no-replace claim checked against both ThreadStore records and tag-scoped artifacts ([issue](/Users/xianxu/workspace/pair/workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md:315); [plan](/Users/xianxu/workspace/pair/workshop/plans/000149-couch-opaque-tags-and-a-human-naming-layer-plan.md:211)). `AllocateThreadTag` only calls `CreateThread` and retries `ThreadExistsError` ([threadtag.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadtag.go:15)). Its tests cover record collisions only.

A pre-existing `draft-couch-<hex>.md`, ledger, configuration, or detached session without a ThreadStore record can therefore be claimed as a new thread and opened through `pair resume`, violating the durable identity boundary.

Fix sketch: inject a scoped-artifact collision checker and include it in the same allocation decision. Seed a scoped artifact for the first scripted tag, verify allocation retries to the second tag, and verify the old artifact remains untouched. This is ARCH-PURPOSE; the necessary collision enumeration cannot be deferred to M5 after M1 begins minting final identities.

### C3 — The Core concepts table contradicts the implementation

The plan calls `CouchNamespace` PURE and separately lists a `NamespaceResolver` integration ([plan](/Users/xianxu/workspace/pair/workshop/plans/000149-couch-opaque-tags-and-a-human-naming-layer-plan.md:39)). No `NamespaceResolver` entity exists. Instead, `ResolveCouchNamespace` directly performs `MkdirAll` and `EvalSymlinks` ([namespace.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/namespace.go:21)), and every namespace test uses mutable filesystem IO ([namespace_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/namespace_test.go:9)).

Per the Core concepts review contract, this PURE/INTEGRATION contradiction is Critical and requires a plan revision. Either classify `CouchNamespace`/`ResolveCouchNamespace` as INTEGRATION, or split a genuinely pure namespace value/decision from an injected filesystem resolver. This flags ARCH-PURE.

## 3. Important findings

### I1 — README still documents the removed admission authority

[README.md](/Users/xianxu/workspace/pair/README.md:276) says Couch always allows one agent per tree and that `--same-tree` overrides the registry guard. M1 removes that flag and makes normalized provider policy the sole admission authority. README was not updated in the M1 implementation slice.

Fix sketch: document policy-derived admission and remove the nonexistent override. Add a documentation test that rejects removed operator-facing flags, not only one that checks current flags are present. This flags ARCH-PURPOSE.

### I2 — Policy-instability exhaustion does not implement its typed contract

The plan promises at most three cohort retries followed by a typed `policy-unstable` refusal ([plan](/Users/xianxu/workspace/pair/workshop/plans/000149-couch-opaque-tags-and-a-human-naming-layer-plan.md:178)). The code performs four attempts and returns a generic formatted error ([admission.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/admission.go:62)). The declared `errPolicyEpochChanged` is unused.

Fix sketch: introduce the typed refusal, align the attempt count, and test persistent mixed epochs for exact call count, type, rollback, and zero child starts.

### I3 — The documented live-conformance invocation is not runnable

The plan documents:

`make test-couch-policy-live ARIADNE_SDLC_BIN=../ariadne/bin/sdlc`

But the Make target consumes `SDLC_BIN` ([Makefile.local](/Users/xianxu/workspace/pair/Makefile.local:62)). Additionally, a relative binary is passed into `go test`, whose test process runs from the package directory; the relative invocation failed with `lstat ../ariadne: no such file or directory`. Using the absolute path and `SDLC_BIN` passed.

Fix sketch: standardize the variable and canonicalize it with an absolute path inside the target, then use that exact command in the plan.

## 4. Minor findings

- [couch.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/couch.go:125) still says the tag derives from the tree and that the same tree always resumes the same session. The implementation now generates a new opaque tag for every admitted start.
- `git diff --check` over the requested boundary fails on trailing whitespace in [issue 150](/Users/xianxu/workspace/pair/workshop/issues/000150-in-session-continuation-style-compacting.md:13), contradicting the M1 Log’s whole-window verification claim.

## 5. Test coverage notes

Verified:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1` — pass.
- Focused `-race` for both packages — pass.
- `go vet` for both packages — pass.
- `go test ./... -count=1` — pass.
- Live provider conformance with absolute `SDLC_BIN` — pass.
- `git diff --check` over the exact requested boundary — fails on the issue-150 whitespace above.

The green suite does not protect the two main defects: the liveness test models only one client PID, and tag-allocation tests enumerate only ThreadStore collisions.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** Admission decisions derive from the normalized provider result, and the source-level sweep prevents the old local policy implementation from returning.
- **ARCH-PURE — flag.** `Admission.Decide` is a good pure core, but the namespace Core concepts classification contradicts its filesystem-backed implementation.
- **ARCH-PURPOSE — flag.** Client-PID pruning releases capacity without whole-incarnation proof; tag allocation omits half its collision domain; README still presents the removed local authority.
- **ARCH-MOCK — flag.** The provider fake and scheduled live conformance are good. The process model is incomplete where admission depends on it: FakeProcOps/FakeRunner model the Pair client, not the stateful zellij session and surviving workspace writers.

## 7. Plan revision recommendations

Append a `## Revisions` entry recording:

1. Whole-incarnation liveness, not Pair-client PID death, is required before capacity release; M1 remains occupied until #152 supplies that proof.
2. Scoped artifact collision enumeration is part of M1 allocation, with a load-bearing artifact-collision test.
3. `CouchNamespace`/`ResolveCouchNamespace` are INTEGRATION, or the plan names the new pure/injected split; remove the nonexistent `NamespaceResolver` claim.
4. Policy instability uses the actual typed error and retry count.
5. Task 6 includes README and the corrected absolute `SDLC_BIN` conformance command.

```findings
findings:
  - id: new
    severity: Critical
    family: incarnation-quiescence-before-capacity-release
    title: |
      a dead Pair client frees capacity while its zellij session can still write
    detail: |
      admission.go:188 prunes an incarnation when its recorded Pair client PID is dead, but the issue contract and probes/zellijpark establish that zellij and its workspace-writing panes survive that client. A subsequent bounded-policy start can therefore admit a second writer. Retain the incarnation as unknown/occupied until whole-incarnation quiescence is proven, and pin dead-client-plus-live-session behavior with a stateful test (ARCH-PURPOSE, ARCH-MOCK).
  - id: new
    severity: Critical
    family: composite-address-collision-domain
    title: |
      opaque tag allocation checks ThreadStore records but not scoped artifacts
    detail: |
      threadtag.go:15 retries only ThreadExistsError from CreateThread, while the Spec and plan require collision checks against both composite records and tag-scoped artifacts. An orphaned draft, ledger, config, or session can be claimed as a new thread and opened by pair resume. Add the scoped-artifact collision seam and a test that pre-creates an artifact for the first entropy result and requires allocation to retry (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: core-concept-kind-contract
    title: |
      the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
    detail: |
      The plan lists CouchNamespace as PURE and NamespaceResolver as its integration, but no NamespaceResolver exists; ResolveCouchNamespace performs MkdirAll and EvalSymlinks directly, and namespace tests require mutable filesystem IO. Revise the plan to classify the actual entity as INTEGRATION or split a genuinely pure value from an injected filesystem resolver (ARCH-PURE).
  - id: new
    severity: Important
    family: user-facing-policy-docs
    title: |
      README still advertises the removed one-tree guard and --same-tree bypass
    detail: |
      README.md:276 says registry membership enforces one agent per tree and --same-tree overrides it. M1 removes that flag and makes normalized provider policy the sole admission authority. Update the README and add a negative documentation check for removed flags (ARCH-PURPOSE).
  - id: new
    severity: Important
    family: typed-retry-exhaustion-contract
    title: |
      policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
    detail: |
      The plan specifies at most three whole-cohort retries and a typed policy-unstable refusal; admission.go:62 performs four and returns fmt.Errorf. Align the retry count and type, and test exact calls, rollback, and zero forks under persistent mixed epochs.
  - id: new
    severity: Important
    family: live-conformance-target-interface
    title: |
      the plan's live-provider command uses the wrong variable and a nonportable relative binary
    detail: |
      The plan sets ARIADNE_SDLC_BIN, while Makefile.local:62 consumes SDLC_BIN. Passing ../ariadne/bin/sdlc also fails because the test runs from its package directory; the absolute SDLC_BIN invocation passes. Canonicalize the target input and update the recorded command.
  - id: new
    severity: Minor
    family: verification-window-cleanliness
    title: |
      git diff --check fails over the requested boundary
    detail: |
      workshop/issues/000150-in-session-continuation-style-compacting.md:13 has trailing whitespace, despite the M1 Log claiming git diff --check passed.
  - id: new
    severity: Minor
    family: identity-comment-truthfulness
    title: |
      Spawn still comments that tags derive from trees and same-tree starts resume one session
    detail: |
      couch.go:125-134 describes the superseded path-derived tag behavior immediately above code that launches the newly allocated opaque thread tag.
```

---

## Re-review — 2026-08-26T12:10:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a271432590da8a4177fea6c523607182536861a2^..b8a72ecf3bc4ecf043e148d148b6150e112fe3c3 |
| command | sdlc milestone-close --issue 149 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-26T12:10:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The prior lifecycle, README, retry, conformance-command, whitespace, and sequential artifact-collision fixes are substantively present, with most pinned by tests that fail under mutation. The boundary remains blocked because artifact collision checking is only a preflight scan—not the promised atomic claim—and the Core concepts table still misclassifies an effectful `Operation` as PURE without a regression protecting the corrected classification.

## 1. Strengths

- Admission now retains all incarnations until whole-incarnation quiescence is proven. Mutating the occupant count to release them makes `TestAdmissionReconcileRetainsDeadClientWithoutWholeIncarnationProof` fail.
- The scoped-artifact retry is wired into production and load-bearing: removing the collision branch makes `TestAllocateThreadTagRetriesScopedArtifactCollision` fail.
- Policy exhaustion now returns `PolicyUnstableError` after exactly three cohorts; restoring four attempts makes its regression fail.
- The relative `SDLC_BIN` target passes, while the old direct relative invocation fails from the package working directory.
- Provider decoding, ThreadStore journaling, supervisor leasing, README, and atlas coverage are substantial and well tested.

## 2. Critical findings

### BR-3 remains open — Core concepts still contradict the implementation

The namespace row was corrected, but no test protects the plan’s classification. More importantly, [the plan](/Users/xianxu/workspace/pair/workshop/plans/000149-couch-opaque-tags-and-a-human-naming-layer-plan.md:48) now calls `Operation` PURE while [Operation](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops.go:34) embeds effectful `Invoke` closures. Its test invokes `Couch.Spawn`, a temporary ThreadStore, and a runner at [ops_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/ops_test.go:24).

This is the 2nd finding in family `core-concept-kind-contract`. The rule is: every current Core concepts row must describe the entity’s current executable boundary, not its intended later M3 shape. Classify the current operation surface as integration/mixed, or split pure declarations from effectful executors now. Add a plan-contract regression comparable to the existing #146 contract test. `ARCH-PURE`.

### New — Artifact collision checking is not atomic with the ThreadStore claim

[threadtag.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadtag.go:39) calls `Collides`, releases that observation, and only afterward calls `CreateThread`. No lock or reservation is shared with scoped-artifact or session-binding producers. A scratch interleaving test created an artifact after `Collides` observed absence but before `CreateThread`; allocation still succeeded with that address.

This is the 2nd finding in family `composite-address-collision-domain`. Do not add another recheck. State and enforce the class rule: one atomic address-claim authority must serialize ThreadStore records and every current artifact/session producer. The sweep includes scoped filenames, the session-name index, malformed index handling, and future constructors. The hard-coded filename list at [artifactcollision.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/artifactcollision.go:64) should also derive from one artifact-family inventory rather than restating `ScopedPaths`. `ARCH-DRY`, `ARCH-PURPOSE`, `ARCH-MOCK`.

## 3. Important findings

- [README.md](/Users/xianxu/workspace/pair/README.md:271) advertises `couch stop <ref>` as operational, but `stop` is `ExecuteLiveOwner` and [RunWithRuntime](/Users/xianxu/workspace/pair/cmd/internal/couchcmd/run.go:142) tries to acquire the already-held supervisor lease. While a console owns the namespace, an external `couch stop` therefore refuses; no #147 routing exists. Document this limitation or provide an owner-local route and regression. Family: `owner-required-command-reachability`.
- [workshop/projects/couch.md](/Users/xianxu/workspace/pair/workshop/projects/couch.md:166) leaves M1 unchecked while lines 179–180 record an actual and closed date. This contradicts the issue log’s statement that close metadata is added only after milestone acceptance. Family: `milestone-state-truthfulness`.

## 4. Minor findings

- BR-8’s comment is now accurate, but the required red-without-fix regression is absent. A source-level identity-comment guard would dispose it under the stated review protocol.

## 5. Test coverage notes

Fresh verification:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1` — pass.
- `go test -race ./cmd/internal/couchcore -count=1` — pass.
- `go test ./... -count=1` — pass.
- `make test` — pass.
- `make test-couch-policy-live SDLC_BIN=../ariadne/bin/sdlc` — pass.
- Exact-window `git diff --check` — pass.
- Worktree returned clean after verification.

Mutation checks confirmed BR-1, BR-2’s sequential case, and BR-5 are load-bearing. The new concurrent artifact-creation test fails against HEAD.

## 6. Architectural notes for upcoming work

- `ARCH-DRY` — flag: collision filenames duplicate `ScopedPaths`.
- `ARCH-PURE` — flag: current `Operation` values contain effectful closures.
- `ARCH-PURPOSE` — flag: the promised atomic composite claim is currently a preflight check.
- `ARCH-MOCK` — flag: the artifact fake covers sequential results but not producer interleavings through the production boundary.

## 7. Plan revision recommendations

Append a `## Revisions` entry recording:

1. The atomic address-claim authority shared by ThreadStore and all artifact/session producers.
2. The current M1 `Operation` kind, distinct from the planned M3 declaration/executor split.
3. External behavior of owner-required commands before #147 routing.
4. Correction of project milestone state so `closed` cannot coexist with an unaccepted boundary.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The dead-client release path is removed, and mutation of incumbent counting makes the whole-incarnation regression fail.
  - id: BR-2
    disposition: addressed
    note: |
      The requested sequential scoped-artifact collision seam and retry regression are present and fail when the collision branch is removed; a separate atomicity sibling is raised below.
  - id: BR-3
    disposition: not-addressed
    note: |
      The namespace text changed but has no red-without-fix contract test, and the same table still labels the effectful current Operation surface PURE.
  - id: BR-4
    disposition: addressed
    note: |
      README now documents provider-owned admission, and its negative removed-flag test would fail on the prior text.
  - id: BR-5
    disposition: addressed
    note: |
      PolicyUnstableError, three exact cohorts, call count, and rollback are pinned; restoring four attempts makes the regression fail.
  - id: BR-6
    disposition: addressed
    note: |
      The target canonicalizes SDLC_BIN and the documented relative invocation passes; the prior direct relative invocation still demonstrates the original failure.
  - id: BR-7
    disposition: addressed
    note: |
      Exact-window git diff --check now exits successfully.
  - id: BR-8
    disposition: not-addressed
    note: |
      The comment is corrected, but no test fails if the obsolete path-derived identity claim returns.
findings:
  - id: new
    severity: Critical
    family: composite-address-collision-domain
    title: |
      scoped artifact checking is a racy preflight rather than an atomic address claim
    detail: |
      AllocateThreadTag checks artifacts before independently acquiring ThreadStore state. An artifact can appear between those operations and allocation still succeeds. This is the 2nd finding in this family: define one claim rule shared by every record, scoped-artifact, and session-binding producer rather than fixing another individual filename or adding a second scan.
  - id: new
    severity: Important
    family: owner-required-command-reachability
    title: |
      README advertises couch stop although an active supervisor makes the CLI route refuse
    detail: |
      stop requires ExecuteLiveOwner, so a second CLI process cannot acquire the lease held by the running console. Document the pre-147 limitation or route stop through the existing owner, with a held-lease regression.
  - id: new
    severity: Important
    family: milestone-state-truthfulness
    title: |
      the project records M1 closed while its milestone row and boundary remain open
    detail: |
      workshop/projects/couch.md leaves pair#149 M1 unchecked but records actual and closed metadata, contradicting the issue log's acceptance rule.
```

---

## Re-review — 2026-08-26T12:39:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a271432590da8a4177fea6c523607182536861a2^..1587f8efa6fb4ecd62b4f1b4c873bfcdb92022c4 |
| command | sdlc milestone-close --issue 149 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-26T12:39:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared O_EXCL claim resolves BR-9’s race, and the policy, namespace, documentation, and project-state edits are directionally correct. The boundary still cannot close: BR-2’s artifact collision sweep remains incomplete, while BR-11 has no regression test and therefore cannot be disposed as addressed under this gate’s explicit red-without-fix rule.

## 1. Strengths

- The O_EXCL marker now serializes Couch and native Pair creation before either writes artifacts. The concurrent-winner and cross-producer tests are meaningful filesystem-backed coverage in [thread_claim_test.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim_test.go:11).
- BR-3 is pinned: [plan_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/plan_contract_test.go:11) rejects the previous PURE classifications for both namespace resolution and the effectful `Operation` surface.
- BR-8 and BR-10 have regressions containing the exact superseded text, so restoring the prior comment or README command makes them fail.
- Admission remains conservative after client death, provider epochs are reconciled outside the store lock, and live conformance against the real Ariadne provider passes.
- README and atlas comprehensively describe the new namespace, admission, opaque identity, and owner-routing limitations.

## 2. Critical findings

### BR-2 remains open — the scoped-artifact inventory omits active tag-owned families

[OwnsTagArtifact](/Users/xianxu/workspace/pair/cmd/internal/launcher/scoped_paths.go:110) recognizes only constructors represented by `ScopedPaths`, and its coverage test merely iterates that same hand-maintained subset. Production still creates tag-owned artifacts outside it, including:

- `draft-pane-<tag>.json` in [route.go](/Users/xianxu/workspace/pair/cmd/internal/draftroute/route.go:61)
- `image-capture-`, `pair-wrap-pid-`, and `wrap-events-` in [wrap.go](/Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go:448)
- `parked-` and `parked-scrollback-` in [osruntime.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/osruntime.go:764)
- `last-left-pane-`, `last-terminal-pane-`, and `terminal-panes-` in [shortcut.go](/Users/xianxu/workspace/pair/cmd/internal/workbenchshortcut/shortcut.go:420)

A source sweep found at least 18 omitted filename shapes, also including title, layout, quote, slug, and review artifacts. An orphaned omitted artifact therefore still permits opaque allocation to adopt an existing tag address.

This is the 3rd finding in family `composite-address-collision-domain`. Do not add these examples individually. Establish one checked inventory derived by every production constructor, and add a source-level coverage test so introducing any unclassified tag-bearing path fails. This flags ARCH-DRY and ARCH-PURPOSE.

## 3. Important findings

### BR-11 remains open — project-state correction has no load-bearing regression

The erroneous `**closed:**` field was removed from [couch.md](/Users/xianxu/workspace/pair/workshop/projects/couch.md:175), while the M1 row remains open. However, no test or tracker contract fails if closed metadata is restored beside an unchecked milestone. Under the supplied prior-finding protocol, a plausible edit without a red-without-fix test is `not-addressed`.

Fix sketch: add a reusable project-state contract asserting that an unchecked milestone cannot carry accepted/closed metadata. Avoid a one-off pair#149 string check; this is the 2nd finding in family `milestone-state-truthfulness`.

## 4. Minor findings

None.

## 5. Test coverage notes

Fresh verification:

- `go test ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/launcher -count=1` — pass.
- `go test -race ./cmd/internal/couchcore ./cmd/internal/launcher -count=1` — pass.
- `go test ./... -count=1` — pass on rerun.
- `make test-couch-policy-live SDLC_BIN=../ariadne/bin/sdlc` — pass.
- Exact-window `git diff --check` — pass.

The first full-suite run hit a pre-existing launcher fake’s five-second subprocess timeout. The focused test then passed five consecutive runs, and the full rerun passed; no change in this boundary touched that test.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag.** `OwnsTagArtifact` claims to be the single inventory, but numerous production constructors live outside it.
- **ARCH-PURE — pass.** Namespace and operations are now truthfully classified as integration; admission decisions remain pure and directly unit-tested.
- **ARCH-PURPOSE — flag.** Atomic producer serialization landed, but the promised complete scoped-artifact collision domain did not.
- **ARCH-MOCK — pass.** Filesystem claims use real portable directories, process/policy seams have stateful fakes, and the policy seam has passing live conformance.

## 7. Plan revision recommendations

Append a `## Revisions` entry recording:

1. M1’s collision authority must derive from a checked enumeration of every current tag-bearing constructor, including Go and Lua consumers—not only `ScopedPaths`.
2. The enumeration needs a source-level coverage regression so an unclassified constructor fails automatically.
3. Project milestone metadata requires a reusable invariant test: unchecked boundary rows cannot carry accepted/closed state.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      The new claim scans only the ScopedPaths inventory; a source sweep found at least 18 current tag-owned filename shapes outside it, so orphaned artifacts such as draft-pane-, image-capture-, parked-, slug-, and review-* remain claimable. This is the 3rd finding in family composite-address-collision-domain.
  - id: BR-3
    disposition: addressed
    note: |
      The plan classifies namespace resolution and the current effectful Operation surface as integration, and TestIssue149CurrentCoreConceptKinds fails on the prior table.
  - id: BR-8
    disposition: addressed
    note: |
      The comment now describes opaque allocation, and TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract rejects both exact obsolete claims from the prior text.
  - id: BR-9
    disposition: addressed
    note: |
      Couch and native Pair now acquire one O_EXCL marker before creation; concurrent-winner and cross-producer tests fail if that shared atomic authority is removed.
  - id: BR-10
    disposition: addressed
    note: |
      README removes couch stop from the external command list, documents the pre-147 owner-routing limitation, and the regression fails on the prior advertised spelling.
  - id: BR-11
    disposition: not-addressed
    note: |
      The premature closed field was removed, but no regression fails if closed metadata returns beside the unchecked M1 row; the gate contract therefore does not permit an addressed disposition.
```

---

## Re-review — 2026-08-26T12:54:06-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 149 — couch: opaque tags and a human naming layer |
| repo | pair |
| issue file | workshop/issues/000149-couch-opaque-tags-and-a-human-naming-layer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | a271432590da8a4177fea6c523607182536861a2^..085956a40ab38eb9f62ce394ea5bdb35054a877c |
| command | sdlc milestone-close --issue 149 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-08-26T12:54:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both open findings are addressed and protected by load-bearing regressions. The composite-address claim now covers existing, future, and concurrently created artifacts, while project milestone state is checked repository-wide. No new blocking findings surfaced; focused, race, full repository, vet, live-conformance, and diff-cleanliness checks passed.

## 1. Strengths

- The O_EXCL marker is acquired before artifact inspection, closing the prior scan/claim race ([thread_claim.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/thread_claim.go:44)).
- Artifact collision detection uses a general delimiter rule rather than another incomplete prefix inventory ([scoped_paths.go](/Users/xianxu/workspace/pair/cmd/internal/launcher/scoped_paths.go:110)).
- `AllocateThreadTag` claims the shared artifact authority before creating the ThreadStore record and releases it on record-claim failure ([threadtag.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/threadtag.go:41)).
- The project-state regression scans every project rather than special-casing pair#149 ([project_state_contract_test.go](/Users/xianxu/workspace/pair/cmd/internal/couchcore/project_state_contract_test.go:18)).
- Weekly, push, PR, and manual policy-provider conformance are committed ([couch-policy-conformance.yml](/Users/xianxu/workspace/pair/.github/workflows/couch-policy-conformance.yml:3)).

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Verified successfully:

- Focused launcher, couchcore, and couchcmd tests.
- Race tests for launcher and couchcore.
- `go test ./... -count=1`.
- `go vet ./...`.
- `make test`.
- Live Ariadne policy conformance.
- Exact-window `git diff --check`.

Scratch mutation checks:

- Disabling `OwnsTagArtifact` made the scoped, non-scoped, and future-family collision tests fail.
- Restoring M1’s premature `**closed:**` metadata made `TestUncheckedProjectMilestoneHasNoClosedMetadata` fail.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** One structural filename rule covers tag-bearing artifact families without duplicating constructor lists.
- **ARCH-PURE — pass.** `Admission.Decide` remains deterministic and directly unit-tested; filesystem, provider, and store behavior remain in integration seams.
- **ARCH-PURPOSE — pass.** The implementation now protects the complete composite-address collision class, not only previously named filenames.
- **ARCH-MOCK — pass.** Stateful fakes cover policy and artifact behavior, filesystem claims exercise the production boundary, and scheduled live conformance checks the real provider.

## 7. Plan revision recommendations

None. The latest revisions accurately describe the delivered M1 behavior and integration classifications.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      The shared O_EXCL authority now precedes both Couch and native Pair creation, and the structural tag-boundary rule detects current non-ScopedPaths and unknown future artifact families. Disabling that rule makes the production-path collision regressions fail.
  - id: BR-11
    disposition: addressed
    note: |
      Premature closed metadata is absent, and a repository-wide invariant rejects closed metadata beside any unchecked milestone. Restoring pair#149 M1's prior closed line makes the regression fail.
```
