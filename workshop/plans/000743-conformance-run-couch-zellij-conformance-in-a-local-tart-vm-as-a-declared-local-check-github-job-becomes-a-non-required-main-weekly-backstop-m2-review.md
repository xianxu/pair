# Boundary Review — pair#743 (milestone M2)

| field | value |
|-------|-------|
| issue | 743 — conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop |
| repo | pair |
| issue file | workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1fa99626aa4d16425f36cbec1a9cb50359ae8810..07ae3efaa5c600cd67532916665967915ec5bd7f |
| command | sdlc milestone-close --issue 743 --milestone M2 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T22:53:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2’s strict validator, VM command composition, native evidence and documentation check out. One Important gap remains: lifecycle test selection is duplicated between hosted and VM execution despite the plan’s explicit shared-source contract.

1. **Strengths**

   - The validator requires observed execution and completion under exact package identities; metadata cannot grant success.
   - Host composition preserves literal argv, invokes the generic runner once, and preserves failure status.
   - Replaying the retained native logs confirms all 28 required cases. The known-bad stream correctly rejects the skipped required case.
   - README and atlas document the new command, artifact destination and remaining coverage limits. Required hosted PR conformance remains intact.

2. **Critical findings**

   None.

3. **Important findings**

```findings
findings:
  - id: new
    severity: Important
    family: conformance-consumers-share-selection
    title: |
      Hosted and VM lifecycle selections lack the promised shared-source enforcement
    detail: |
      tests/conformance-vm.py:139-155 and :192-199 independently restate the lifecycle packages, cases and checkpoint exclusion maintained in Makefile.local:135-138, which the hosted workflow invokes. No consumer or drift test connects these selections; tests/conformance-paths.py checks only workflow trigger paths. They currently agree, but changing either population can silently leave the other running different coverage, contrary to workshop/plans/000743-local-vm-conformance-plan.md:48. ARCH-DRY and ARCH-PURPOSE: enumerate both lifecycle consumers and derive their selections/exclusions from one catalog, or enforce equality with a regression that fails when either consumer changes independently.
```

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed `make test-vm-conformance`: 35 Python tests.
   - Passed the targeted socket-root regression and native-fixture lint tests.
   - Passed `make test-affected-inventory` and scoped `git diff --check`.
   - Revalidated the 409 retained native events and known-bad refusal. The qualified source and pinned head have no intervening changes in the reviewed executable paths.
   - No new Tart invocation was performed during this read-only review.

6. **Architectural notes**

   - **ARCH-DRY — flag:** lifecycle selection/exclusion duplication above.
   - **ARCH-PURE — pass:** evidence decisions are isolated from subprocess execution.
   - **ARCH-PURPOSE — flag:** the shared-source promise does not yet cover both lifecycle consumers.
   - **ARCH-MOCK — pass:** host command tests use an executable recording fixture; VM ownership remains behind the existing runner seam.
   - **ARCH-CONSTRAINTS — pass:** streaming validation parses once per event; counted tests exercise N and 2N.
   - **ARCH-SECURE — pass:** literal argv and exact package binding preserve the relevant boundaries.
   - **ARCH-ORDER — pass:** malformed, incomplete, duplicate and invalidly ordered events fail closed.
   - **ARCH-FUNERAL — pass:** temporary scratch is scoped; durable artifacts retain the existing runner’s ownership policy.
   - **ARCH-EVOLVE — pass:** this range introduces no persisted read-modify-write codec; additive event members remain accepted with strict known-field validation.

7. **Plan revision recommendations**

   Add a `## Revisions` entry naming the authoritative lifecycle selection/exclusion catalog, both consumers, and their derivation or drift regression. Preserve the approved M3 deferral for named-check declaration and acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9cbed71a-3d1e-4222-8379-cef87368caf2","repository":"github.com/xianxu/pair","issue_id":"000743","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/c99fe24ed4ff7a46","reviewed_head":"07ae3efaa5c600cd67532916665967915ec5bd7f","slot":"pair:3","requested_at":"2026-10-21T05:50:56.522579Z"},"phase":"finished","pid":54506,"started_at":"2026-10-21T05:50:56.68017Z","finished_at":"2026-10-21T05:53:32.257797Z","session_id":"01a15d03-423d-7d42-8799-24b89a538d90","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.253916583s
head: "07ae3efaa5c600cd67532916665967915ec5bd7f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
