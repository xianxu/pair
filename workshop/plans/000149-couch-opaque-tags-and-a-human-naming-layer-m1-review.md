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
