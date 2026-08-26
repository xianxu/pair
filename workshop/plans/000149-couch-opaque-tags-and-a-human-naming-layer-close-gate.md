---
gate: boundary-review
issue: 149
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-26T11:42:03-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: a dead Pair client frees capacity while its zellij session can still write
          detail: admission.go:188 prunes an incarnation when its recorded Pair client PID is dead, but the issue contract and probes/zellijpark establish that zellij and its workspace-writing panes survive that client. A subsequent bounded-policy start can therefore admit a second writer. Retain the incarnation as unknown/occupied until whole-incarnation quiescence is proven, and pin dead-client-plus-live-session behavior with a stateful test (ARCH-PURPOSE, ARCH-MOCK).
          family: incarnation-quiescence-before-capacity-release
          round: 1
        - id: BR-2
          severity: Critical
          title: opaque tag allocation checks ThreadStore records but not scoped artifacts
          detail: threadtag.go:15 retries only ThreadExistsError from CreateThread, while the Spec and plan require collision checks against both composite records and tag-scoped artifacts. An orphaned draft, ledger, config, or session can be claimed as a new thread and opened by pair resume. Add the scoped-artifact collision seam and a test that pre-creates an artifact for the first entropy result and requires allocation to retry (ARCH-PURPOSE).
          family: composite-address-collision-domain
          round: 1
        - id: BR-3
          severity: Critical
          title: the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
          detail: The plan lists CouchNamespace as PURE and NamespaceResolver as its integration, but no NamespaceResolver exists; ResolveCouchNamespace performs MkdirAll and EvalSymlinks directly, and namespace tests require mutable filesystem IO. Revise the plan to classify the actual entity as INTEGRATION or split a genuinely pure value from an injected filesystem resolver (ARCH-PURE).
          family: core-concept-kind-contract
          round: 1
        - id: BR-4
          severity: Important
          title: README still advertises the removed one-tree guard and --same-tree bypass
          detail: README.md:276 says registry membership enforces one agent per tree and --same-tree overrides it. M1 removes that flag and makes normalized provider policy the sole admission authority. Update the README and add a negative documentation check for removed flags (ARCH-PURPOSE).
          family: user-facing-policy-docs
          round: 1
        - id: BR-5
          severity: Important
          title: policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
          detail: The plan specifies at most three whole-cohort retries and a typed policy-unstable refusal; admission.go:62 performs four and returns fmt.Errorf. Align the retry count and type, and test exact calls, rollback, and zero forks under persistent mixed epochs.
          family: typed-retry-exhaustion-contract
          round: 1
        - id: BR-6
          severity: Important
          title: the plan's live-provider command uses the wrong variable and a nonportable relative binary
          detail: The plan sets ARIADNE_SDLC_BIN, while Makefile.local:62 consumes SDLC_BIN. Passing ../ariadne/bin/sdlc also fails because the test runs from its package directory; the absolute SDLC_BIN invocation passes. Canonicalize the target input and update the recorded command.
          family: live-conformance-target-interface
          round: 1
        - id: BR-7
          severity: Minor
          title: git diff --check fails over the requested boundary
          detail: workshop/issues/000150-in-session-continuation-style-compacting.md:13 has trailing whitespace, despite the M1 Log claiming git diff --check passed.
          family: verification-window-cleanliness
          round: 1
        - id: BR-8
          severity: Minor
          title: Spawn still comments that tags derive from trees and same-tree starts resume one session
          detail: couch.go:125-134 describes the superseded path-derived tag behavior immediately above code that launches the newly allocated opaque thread tag.
          family: identity-comment-truthfulness
          round: 1
      boundary: M1
      blocked: true
---

# Gate ledger — pair#149 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T11:42:03-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `incarnation-quiescence-before-capacity-release` a dead Pair client frees capacity while its zellij session can still write
  admission.go:188 prunes an incarnation when its recorded Pair client PID is dead, but the issue contract and probes/zellijpark establish that zellij and its workspace-writing panes survive that client. A subsequent bounded-policy start can therefore admit a second writer. Retain the incarnation as unknown/occupied until whole-incarnation quiescence is proven, and pin dead-client-plus-live-session behavior with a stateful test (ARCH-PURPOSE, ARCH-MOCK).
- **BR-2** [Critical] `composite-address-collision-domain` opaque tag allocation checks ThreadStore records but not scoped artifacts
  threadtag.go:15 retries only ThreadExistsError from CreateThread, while the Spec and plan require collision checks against both composite records and tag-scoped artifacts. An orphaned draft, ledger, config, or session can be claimed as a new thread and opened by pair resume. Add the scoped-artifact collision seam and a test that pre-creates an artifact for the first entropy result and requires allocation to retry (ARCH-PURPOSE).
- **BR-3** [Critical] `core-concept-kind-contract` the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
  The plan lists CouchNamespace as PURE and NamespaceResolver as its integration, but no NamespaceResolver exists; ResolveCouchNamespace performs MkdirAll and EvalSymlinks directly, and namespace tests require mutable filesystem IO. Revise the plan to classify the actual entity as INTEGRATION or split a genuinely pure value from an injected filesystem resolver (ARCH-PURE).
- **BR-4** [Important] `user-facing-policy-docs` README still advertises the removed one-tree guard and --same-tree bypass
  README.md:276 says registry membership enforces one agent per tree and --same-tree overrides it. M1 removes that flag and makes normalized provider policy the sole admission authority. Update the README and add a negative documentation check for removed flags (ARCH-PURPOSE).
- **BR-5** [Important] `typed-retry-exhaustion-contract` policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
  The plan specifies at most three whole-cohort retries and a typed policy-unstable refusal; admission.go:62 performs four and returns fmt.Errorf. Align the retry count and type, and test exact calls, rollback, and zero forks under persistent mixed epochs.
- **BR-6** [Important] `live-conformance-target-interface` the plan's live-provider command uses the wrong variable and a nonportable relative binary
  The plan sets ARIADNE_SDLC_BIN, while Makefile.local:62 consumes SDLC_BIN. Passing ../ariadne/bin/sdlc also fails because the test runs from its package directory; the absolute SDLC_BIN invocation passes. Canonicalize the target input and update the recorded command.
- **BR-7** [Minor] `verification-window-cleanliness` git diff --check fails over the requested boundary
  workshop/issues/000150-in-session-continuation-style-compacting.md:13 has trailing whitespace, despite the M1 Log claiming git diff --check passed.
- **BR-8** [Minor] `identity-comment-truthfulness` Spawn still comments that tags derive from trees and same-tree starts resume one session
  couch.go:125-134 describes the superseded path-derived tag behavior immediately above code that launches the newly allocated opaque thread tag.

## Open findings

- **BR-1** [Critical] `incarnation-quiescence-before-capacity-release` a dead Pair client frees capacity while its zellij session can still write
- **BR-2** [Critical] `composite-address-collision-domain` opaque tag allocation checks ThreadStore records but not scoped artifacts
- **BR-3** [Critical] `core-concept-kind-contract` the Core concepts table labels a filesystem-backed namespace PURE and names a nonexistent integration
- **BR-4** [Important] `user-facing-policy-docs` README still advertises the removed one-tree guard and --same-tree bypass
- **BR-5** [Important] `typed-retry-exhaustion-contract` policy epoch exhaustion uses four attempts and a generic error instead of the promised typed refusal
- **BR-6** [Important] `live-conformance-target-interface` the plan's live-provider command uses the wrong variable and a nonportable relative binary
- **BR-7** [Minor] `verification-window-cleanliness` git diff --check fails over the requested boundary
- **BR-8** [Minor] `identity-comment-truthfulness` Spawn still comments that tags derive from trees and same-tree starts resume one session
