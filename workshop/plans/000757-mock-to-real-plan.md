---
collaboration-mode: discussion
---

# Mock-to-real app template implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute the approved plan with superpowers-executing-plans and bounded independent agents where appropriate. Steps use checkboxes; this is one atomic close boundary.

**Goal:** Provide a copyable Go app that replaces mock screens one at a time and refuses a production build until every declared screen is real.

**Architecture:** A standalone example module owns one typed screen list, real handlers, a development-only HTTP fallback to the existing Couch mock server, and a guarded production build command. Dispatch, progress and completeness derive from the same list. Fixture-shape contracts test real backend HTTP responses independently of mock data delivery.

**Tech Stack:** Go standard library, net/http/httputil, encoding/json, html/template, and golang.org/x/net/html for authoritative development chrome insertion. No browser framework or production Couch dependency.

## Proposed contract — awaiting hub/operator approval

The template lives at examples/mock-to-real with its own go.mod and README. Copy the directory, rename the module imports, and run its tests. Pair's root suite invokes the nested module through a focused integration test, because root go test ./... excludes nested modules.

Development starts the existing host with couch mock serve examples/mock-to-real/mocks, then runs the app with go run ./cmd/app dev --mock-url URL. The URL must be an explicit loopback HTTP origin with no credentials/path/query. Two explicit processes keep the app portable and avoid a hidden child-process supervisor; neither opens a browser. The mock host owns scanning, selection, confinement and reload. The app owns its listener and graceful shutdown. The app does not reproduce those host mechanisms.

One screen list declares index, runs and detail, their canonical .html paths and optional real handlers. Index and runs are real in the checked-in sample; detail is absent and therefore mock-backed. Missing real code is an absent registration, never a handler error. Registering detail makes it real without changing the router or progress logic. Duplicate/invalid IDs, paths or handler registrations refuse startup/build. Unknown paths refuse, rather than becoming unrestricted proxy requests.

The app resolves /, /index.html and selected-version paths /mock-N/screen.html through the same screen list. Every known real screen dispatches locally even when linked from a version URL; every missing screen proxies to the mock host. Proxy only declared screen documents, their confined version assets, fixtures.json and the two host reload endpoints. Forwarded redirects remain on the app origin, so live reload and version navigation cannot escape the hybrid router. On mock HTML, inject an authoritative visible MOCK badge plus the current X of Y screens real count. Real pages show the count and REAL status. Do not place a mock badge on real handler failures or silently substitute a mock after such a failure. Scenario query strings and fragments retain ordinary browser behavior.

The production command is go run ./cmd/build --output PATH. It imports the application's same screen registry, validates completeness before invoking go build -tags production, and names every incomplete screen on refusal. It writes no new artifact on refusal. A pre-existing output remains unchanged and is explicitly identified as an older artifact, not a successful build. Production-specific files contain no proxy, fixture-serving route, mock badge, mock assets or fixtures embedding. The production app also refuses startup if the registry is incomplete, so raw compiler invocation cannot turn missing screens into a mock-backed release. Raw go build is not the documented release command; the guarded command is the production build interface. No partial-release/not-yet mode in this initial template (the issue permits choosing refusal).

The single mocks/fixtures.json contains representative project, runs and error response values. Mock screens read it through the host. Real endpoints use their own application data source, not those bytes. A contract table maps actual endpoint/status to a fixture subtree. Infer required object keys and JSON types recursively; allow additional response object fields, require correct nested shapes and every array element, distinguish null, booleans, strings and numbers. Empty observed arrays are valid against a nonempty array prototype. Empty fixture arrays cannot infer element types and refuse as a contract definition with an actionable request for a representative item. Heterogeneous fixture prototypes refuse rather than silently choosing one element. Numeric values share the JSON number type. JSON and nesting/size limits refuse malformed or oversized contracts. Nothing rewrites fixtures or drops unknown fields.

## Alternatives

1. Recommended: standalone module plus development HTTP fallback to #755. Reuses version/scan/reload behavior and keeps production independent; development needs two explicit commands.
2. Extract #755 into a public embeddable library. Enables one process, but broadens this issue into the mock server's ownership and public API migration.
3. Copy the host implementation into the template. Self-contained one process, but duplicates security, selection and browser lifecycle logic and creates drift.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| ScreenList, ResolveScreen, Progress, RequireComplete | examples/mock-to-real/internal/hybrid/screens.go | new |
| FixtureShape, InferShape, MatchShape | examples/mock-to-real/internal/contract/shape.go | new |
| DecorateDevelopmentPage | examples/mock-to-real/internal/hybrid/page.go | new |

ScreenList is the single authority for routes, fallback choice, progress and production completeness. It validates a finite collection once and resolves via a map; optional handler bindings are validated against declared IDs. Handler invocation is an integration effect, not part of the pure route decision. No persisted registration format or generator is introduced.

FixtureShape is a read-only recursive value derived from representative JSON. Tests use literal decoded JSON, no filesystem or network mocks. MatchShape reports a JSON path and expected/actual kind. Depth/size bounds prevent unbounded inference; object iteration and arrays are linear in visited nodes.

DecorateDevelopmentPage parses bounded HTML, replaces any conflicting template-owned chrome, preserves existing host identity/bootstrap/scripts, and inserts the badge and progress before delivery. Pure byte input/output tests pin mock/real distinction, existing metadata preservation and HTML escaping.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| DevelopmentRouter | examples/mock-to-real/internal/hybrid/dev.go | new | local handlers and HTTP reverse proxy |
| ProductionRouter | examples/mock-to-real/internal/hybrid/production.go | new | real HTTP handlers only |
| Sample app registry/backend | examples/mock-to-real/internal/sample/app.go | new | real application data and HTTP responses |
| App command | examples/mock-to-real/cmd/app/main.go and tagged mode files | new | listener, context, signal/shutdown |
| Guarded production builder | examples/mock-to-real/cmd/build/main.go | new | validated registry and go build subprocess |
| Nested template qualification | cmd/internal/mocktemplate/template_test.go | new | private copied module, Go tool and real #755 host |

Development HTTP transport is injectable; tests use a stateful httptest mock host tracking requests, redirects, failures and fixture changes. Live conformance runs the real #755 server against private sample files through the app origin; it verifies clicks/redirects, fixture access, selected-version navigation and reload endpoint routing. This is a test seam, not a new production dependency. Real backend contracts run httptest against actual sample handlers.

The Go build subprocess is context-owned and joined. Build refuses before spawning it for incomplete registries; actual private-module builds prove refusal and success after a third real registration. Production tests request every declared screen plus fixture, reload, version-asset and unknown routes and require absence/refusal of all mock surfaces. Tests never execute a freshly copied Pair binary: use the shared testpair image if invoking Couch.

## Actions and visible states

Actions: start development, start mock host, edit mock/fixtures, implement/register a screen, retry request, build production, run production, stop process.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Two real, one absent | two REAL screens, detail MOCK; 2 of 3 real | implement/register detail | no durable loss |
| Mock host unavailable | explicit preview unavailable error, real screens still work | start mock host then retry request | unsaved browser state on later refresh |
| Invalid mock or fixtures | host repair error shown, no stale success claim | edit mock/fixtures | browser transient state may reset after repair |
| Real handler errors | real application error, never fallback | repair real handler then retry | request-local state |
| Invalid/duplicate screen list | named startup/build refusal | fix registrations and retry | no output written |
| Production build has absent screens | build refusal naming screens | implement/register each screen | old output may remain; no new artifact |
| All screens real | production build succeeds; no mock routes or assets | run production | operator-owned output replaces prior output |
| Build interrupted/fails | nonzero result, no success claim | retry build | existing output may remain older |
| Response shape drift | test names endpoint and JSON path | repair backend or consciously revise fixture | no fixture rewrite |
| Contract lacks array prototype | actionable contract-definition failure | add representative item | none |
| Stop server/build | owned listeners/children end and join | start development/build again | browser transient state only |

## Cost and lifecycle

Screen lookup is O(1) after one O(S) validation of S screens; progress is computed once from the same immutable list. Page chrome insertion is O(H) for at most1MiB HTML. The proxy streams non-HTML responses with standard bounded buffers; HTTP header/read/write/idle deadlines and bounded upstream response headers apply. A cached progress value introduces no rescan per browser. Contract work is O(J) JSON nodes plus bytes, bounded at1MiB and depth64. Count node visits and page transformations at N/2N, and assert no registry scan on N/2N requests. No watcher, sidecar, cache directory or per-request artifact is created. Lock/staging files remain #755's bounded host-owned family and are ignored by the template's mock-project .gitignore. Build outputs are explicit operator-owned artifacts, one selected output path replaced by Go build; test copies/temp roots are scoped and removed after owned processes join.

## Implementation tasks (after approval and change-code)

### 1. Pure registry, shape and page policy

Files: internal/hybrid/screens.go, screens_test.go, page.go, page_test.go; internal/contract/shape.go, shape_test.go under the nested module.

- [ ] Write failing table tests for route aliases, unknowns, duplicates, real/absent classification, progress and complete/refuse decisions.
- [ ] Run nested go test ./internal/hybrid -run TestScreen -count=1; require expected missing-symbol/behavior failures, then implement minimal immutable registry and resolve functions.
- [ ] Write failing shape tests for missing/nested keys, primitive/null drift, extra fields, all array elements, empty observed arrays, missing fixture prototype, heterogeneity and bounds; implement inference/matching and verify green.
- [ ] Write failing HTML tests for visible badge/count, hostile text escaping and preservation of host bootstrap/identity; implement bounded transformation and verify green.
- [ ] Add counted N/2N work tests and fuzz seeds for path/shape boundaries; commit the green pure core.

### 2. Development/production app and fixture contracts

Files: internal/hybrid/dev.go, production.go and their tests; internal/sample/app.go, app_test.go, contract_test.go; cmd/app/main.go plus tagged mode files; mocks/fixtures.json, mock-1/{index,runs,detail}.html, app.js, tokens.css, design.md, .gitignore; module go.mod/go.sum.

- [ ] Write failing stateful fake-host tests proving local real dispatch, actual missing-screen fallback, app-origin redirects, error ownership, reload/fixture routes and unknown/private route refusal.
- [ ] Implement the minimal router and thin process entry; use the existing host over HTTP, no copied watcher/selection logic.
- [ ] Build a three-screen sample with two real implementations, one absent and one shared fixture source; all links remain clickable across real/mock boundaries.
- [ ] Run real backend HTTP contract tests against independent sample data; mutate field/type and require failure, restore source and verify green.
- [ ] Add production tag separation and runtime completeness refusal; prove no fixture, reload or mock asset route exists.
- [ ] Run nested go test -race ./... -count=1 and commit.

### 3. Guarded build, host conformance and repository checks

Files: cmd/build/main.go, main_test.go in the module; cmd/internal/mocktemplate/template_test.go; cmd/internal/artifactpath/manifest.go; audited reader inventory if needed; README.md; atlas/mock-to-real.md and atlas/index.md.

- [ ] Write failing public-command test: production build refuses the checked-in sample and leaves no new output, naming detail.
- [ ] Implement registry validation before the Go tool. In a private module copy add/register detail; require the same production command to build and start successfully, with no mock markers or endpoints.
- [ ] Add root Go integration coverage that explicitly runs nested tests/build checks and real-host conformance so the standard suite cannot omit the module.
- [ ] Run counted tests and negative controls for bypassed completeness and wrong response shape. Run any bounded private-profile browser qualification needed to prove hybrid clicks and visible status; no visible window.
- [ ] Audit every source reader and production file; run make test-affected-inventory and targeted artifactpath/repolint checks, then update README/atlas and sample design instructions.
- [ ] Push/open PR, run recorded affected/suite, close once with mandatory review, land through sdlc merge, report hub. No independent review boundary or Mx tag.

## Architecture decisions

ARCH-DRY: existing mock host remains the one scan/selection/reload owner; the screen list is one dispatch/progress/build authority. ARCH-PURE: pure route/shape/decoration policy separate from HTTP/process effects. ARCH-PURPOSE: sample crosses real/mock links and actually refuses then succeeds at production build. ARCH-MOCK: stateful host fake plus real-host conformance, independent backend response tests. ARCH-CONSTRAINTS: explicit HTML/JSON/depth/HTTP bounds and counted work. ARCH-SECURE: loopback-only origin and closed routes; production does not compile a proxy or mock assets. ARCH-ORDER: immutable routing state, build validation before child/output effects, context-owned children joined. ARCH-FUNERAL: no unbounded generated family; explicit build output and scoped test roots. ARCH-EVOLVE: fixtures are read-only, extra response fields allowed, shape changes require deliberate fixture revision.
