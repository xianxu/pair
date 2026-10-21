# Boundary Review — pair#757 (whole-issue close)

| field | value |
|-------|-------|
| issue | 757 — mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000757-mock-to-real-app-template-per-screen-real-or-mock-dev-router-refusing-production-build-fixtures-contract-tests-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ff2f5735f544de5b29ee7d8f78a224f424eb83d..ae22a99112f22f9dc85027198aa406c4483fdfcd |
| command | sdlc close --issue 757 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:33:00-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers the main workflow: catalogue-driven dispatch, explicit real-error ownership, read-only reporting and successful partial production builds. Focused tests pass, including race checks and nested sample qualification. Redirect handling has a correctness defect, the Core concepts table contradicts the implementation, and browser qualification remains unverifiable.

1. **Strengths**

   - Routing and progress share an immutable catalogue published with project health/revision (`server.go:71`).
   - Missing declared mock documents fail visibly instead of falling back to index or upstream (`server.go:187`).
   - Production qualification builds partial and completed applications, checks placeholders and scans artifacts for development content (`build_test.go:99`).
   - README and atlas updates cover both new commands, framework ownership and the SPA navigation limit.

2. **Critical findings**

   - `cmd/internal/mockserve/front.go:82–86`: redirect rewriting loses the root-path meaning of origin-only URLs. Preserve the destination when removing the upstream authority.
   - `workshop/plans/000757-mock-to-real-plan.md:97,108`: the Core concepts table classifies a consuming IO function as PURE and names a nonexistent implementation directory. Correct the classification and paths.

3. **Important findings**

   - `cmd/internal/mockserve/front.go:262`: self-target detection compares only ports, rejecting distinct loopback endpoints.
   - `examples/mock-to-real/internal/buildsample/build.go:139`: the compiler executor lacks the injected stateful fake required by ARCH-MOCK.
   - `cmd/internal/mockserve/front_browser_test.go:97`: browser navigation qualification failed repeatedly with CDP EOF; host evidence is needed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Host and root template tests, normally and with `-race`.
   - Couch command and artifact-path tests.
   - All 13 affected-runner tests.
   - Reader inventory and diff whitespace checks.

   Browser qualification failed three times, including an isolated rerun and a rerun using `/tmp`. No full-suite pass is claimed. Existing redirect coverage exercises an absolute URL with a nonempty path, missing the origin-only case.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing scan, confinement, selection and reload owners are reused.
   - **ARCH-PURE — flag:** `catalogue.Read` consumes mutable input through `io.Reader`; its PURE classification is inaccurate.
   - **ARCH-PURPOSE — flag:** redirect semantics and unresolved browser qualification leave part of the HTTP/navigation contract unqualified. Dispatch, reporting and production status otherwise derive from the catalogue.
   - **ARCH-MOCK — flag:** HTTP tests use stateful upstreams through the production seam; production build execution lacks an injected compiler fake.
   - **ARCH-CONSTRAINTS — pass:** bounded catalogue/JSON input, proxy admission, transport budgets and counted traversal checks are present.
   - **ARCH-SECURE — pass:** destinations are pinned to loopback, private namespaces terminate locally and mock reads retain confinement.
   - **ARCH-ORDER — pass:** synchronized publication, admission and shutdown have barrier-controlled stream, upload and upgrade tests.
   - **ARCH-FUNERAL — pass:** staging roots and children have cleanup owners; explicit output paths prevent accumulating implicit artifacts.
   - **ARCH-EVOLVE — pass:** catalogue consumption is read-only; additive fields remain on disk and known fields/duplicate keys stay strict.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting the active Core concepts table: classify bounded reader consumption as INTEGRATION or extract a pure byte parser, and replace `internal/sample` with the delivered `internal/app` and `internal/buildsample` paths. Document the compiler execution seam and its fake/conformance strategy.

```findings
findings:
  - id: new
    severity: Critical
    family: redirect-reference-preservation
    title: |
      Preserve root-path semantics when rewriting upstream redirects
    detail: |
      cmd/internal/mockserve/front.go:82–86 clears Scheme and Host without supplying a root path. An upstream Location of http://HOST:PORT becomes empty; http://HOST:PORT?next=1 becomes ?next=1, which a browser resolves against the current front path instead of /. This violates redirect preservation. Preserve the resolved destination when constructing the front-relative reference and add regression cases for empty paths, queries, fragments and leading-double-slash paths.
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      Reconcile the Core concepts table with actual purity and implementation paths
    detail: |
      workshop/plans/000757-mock-to-real-plan.md:97 lists catalogue.Read as PURE, but examples/mock-to-real/internal/catalogue/catalogue.go:18–20 consumes an arbitrary mutable io.Reader and propagates its IO errors. The integration row at plan line 108 also names internal/sample, which does not exist; implementation lives in internal/app and internal/buildsample. Under the required Core concepts cross-check these contradictions block admission. Reclassify Read as INTEGRATION or extract a pure byte parser, correct the paths, and record a Revisions entry (ARCH-PURE).
  - id: new
    severity: Important
    family: endpoint-identity-validation
    title: |
      Compare complete endpoint identity when rejecting self-targets
    detail: |
      cmd/internal/mockserve/front.go:262 rejects any upstream sharing the front's port, although the front binds specifically to 127.0.0.1. An app bound only to [::1]:3000 and a front at 127.0.0.1:3000 are distinct supported loopback endpoints, yet startup rejects them as self-targeting. Compare the pinned destination address and port with the listener endpoint; test distinct-address acceptance alongside genuine self-target rejection (ARCH-SECURE).
  - id: new
    severity: Important
    family: external-dependency-fake-seam
    title: |
      Inject the production builder's compiler execution boundary
    detail: |
      examples/mock-to-real/internal/buildsample/build.go invokes the concrete run helper for go build and candidate validation; run constructs exec.CommandContext directly at line 140. No injected executor or stateful compiler fake exists, so Build's compiler failure, partial-output and cancellation orderings cannot be driven deterministically through the production boundary. Add that seam and fake, retaining the existing real-compiler tests as conformance evidence (ARCH-MOCK).
  - id: new
    severity: Important
    kind: unverifiable
    family: browser-navigation-qualification
    title: |
      Browser navigation qualification could not complete in the review environment
    detail: |
      WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run '^TestDevelopmentFrontBrowser$' -count=1 -timeout=60s failed twice at front_browser_test.go:97 with cdp: Runtime.evaluate: cdp: read: EOF before validating the first real-page DOM. A third run with TMPDIR=/tmp failed identically. Supply successful pinned-head host evidence for the full navigation test, including the delivered-progress negative control, or repair the browser environment and rerun; these failures do not establish an application defect.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"18741f3c-c1cb-4113-a689-b84b0b3ad3eb","repository":"github.com/xianxu/pair","issue_id":"000757","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e718e163a11db63a","reviewed_head":"ae22a99112f22f9dc85027198aa406c4483fdfcd","slot":"pair:4","requested_at":"2026-10-21T20:29:13.518874Z"},"phase":"finished","pid":49251,"started_at":"2026-10-21T20:29:13.658394Z","finished_at":"2026-10-21T20:32:59.709781Z","session_id":"01a16027-599f-71d2-b910-6a8a77e7815c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.264763208s
head: "ae22a99112f22f9dc85027198aa406c4483fdfcd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T13:46:17-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 757 — mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping) |
| repo | pair |
| issue file | workshop/issues/000757-mock-to-real-app-template-per-screen-real-or-mock-dev-router-refusing-production-build-fixtures-contract-tests-product-shaping.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ff2f5735f544de5b29ee7d8f78a224f424eb83d..0e4d26c70c40ef990988a2748c9bd219a42549ba |
| command | sdlc close --issue 757 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:46:17-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the approved framework-neutral Spec and Plan. All five prior findings are resolved with appropriate evidence; no new blocking findings emerged. Targeted host, sample, race, JavaScript, and affected-runner checks pass. Browser execution remains unavailable here, but the retained host qualification answers that limitation against identical executable code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      front_origin.go preserves root/query/fragment and leading-double-slash destinations. TestRealRedirectPreservesResolvedDestination passes; removing the preservation guards in a disposable copy fails all five affected cases.
  - id: BR-2
    disposition: addressed
    note: |
      The active Core concepts integration table now classifies catalogue.Read as bounded io.Reader consumption and names internal/app and internal/buildsample. These match the implementation; Revisions — close round1 records the correction. The obsolete proposal is explicitly nonbinding.
  - id: BR-3
    disposition: addressed
    note: |
      targetsListener compares the same pinned address and port used for dialing. TestFrontSelfTargetUsesCompleteEndpoint covers genuine self-targets and distinct loopback addresses; restoring port-only comparison fails the IPv6 and alternate-IPv4 cases.
  - id: BR-4
    disposition: addressed
    note: |
      Build delegates through the injected executor. Stateful compiler tests exercise partial output, validation ordering, cancellation joining and cleanup. Bypassing the executor in a disposable copy makes both failure-phase regressions fail; real-compiler conformance also passes.
  - id: BR-5
    disposition: addressed
    note: |
      The local browser command again fails at the initial real-page assertion with CDP EOF. Retained browser-pass.log records full navigation success, while browser-progress-negative-control.log records failure at the intended delivered-progress assertion. Their cc545350 code head differs from pinned head only in issue-log and evidence files, so the supplied host measurements answer the previously missing qualification.
```

1. **Strengths**

   - Routing decisions are pure and catalogue-driven; real application failures retain upstream ownership (`front_route.go:32`, `front.go:90`).
   - Hybrid document delivery refuses missing declared screens without inheriting standalone index fallback (`server.go:181`).
   - Production qualification builds and launches partial and completed applications, checks missing-handler refusal, and inspects artifacts for development content (`build_test.go:95`).
   - Shutdown tests use readiness barriers for streams, upgrades, capacity and incomplete uploads (`front_lifetime_test.go`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed root `mockserve` and `mocktemplate` tests, the complete nested sample suite, mockserve race tests, 13 affected-runner tests and 11 reload JavaScript tests. Three disposable-copy negative controls confirmed the BR-1, BR-3 and BR-4 tests detect removal of their fixes. The disposable copy was removed; the reviewed checkout remains clean. Full recorded suite/CI remain separate workflow gates.

6. **Architectural notes for upcoming work**

   | Marker | Result | Evidence |
   |---|---|---|
   | ARCH-DRY | Pass | Existing scan, selection, reload and delivery owners are reused. |
   | ARCH-PURE | Pass | Parsing, route decisions, reporting and shape logic remain separate from IO; corrected table matches implementation. |
   | ARCH-PURPOSE | Pass | Catalogue consumers cover dispatch, progress, reporting and production projection; partial and complete builds are exercised. |
   | ARCH-MOCK | Pass | Stateful upstream and compiler seams share production boundaries; real host/compiler conformance remains. |
   | ARCH-CONSTRAINTS | Pass | Catalogue limits, request admission, transport budgets and counted-work tests enforce the declared envelope. |
   | ARCH-SECURE | Pass | Destination pinning, complete endpoint checks, reserved-route rejection and confined file access are explicit. |
   | ARCH-ORDER | Pass | Atomic observations retain the existing revision model; barrier tests exercise cancellation and joining. |
   | ARCH-FUNERAL | Pass | Build stages and test processes have scoped cleanup; no growing runtime artifact family appears. |
   | ARCH-EVOLVE | Pass | Catalogue consumption is read-only, tolerates additive fields and strictly validates known fields and duplicate keys. |

   README and atlas updates cover the new commands, catalogue convention, navigation boundary and production responsibility.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e876a452-1a2c-4ada-9600-30ca6dfa5e18","repository":"github.com/xianxu/pair","issue_id":"000757","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/db2187b0849e1b21","reviewed_head":"0e4d26c70c40ef990988a2748c9bd219a42549ba","slot":"pair:4","requested_at":"2026-10-21T20:44:21.532216Z"},"phase":"finished","pid":95785,"started_at":"2026-10-21T20:44:21.761731Z","finished_at":"2026-10-21T20:46:16.240158Z","session_id":"01a16035-351a-7251-9d94-0bfa7e23ae61","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.294402458s
head: "0e4d26c70c40ef990988a2748c9bd219a42549ba"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
