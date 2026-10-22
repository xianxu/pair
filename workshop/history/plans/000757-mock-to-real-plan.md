---
collaboration-mode: discussion
---

# Framework-neutral mock-to-real implementation plan — approved revision

> **For agentic workers:** Consult AGENTS.md §3. Use superpowers-executing-plans and bounded independent agents where appropriate. Operator approved this framework-neutral revision at 11:00 PDT on 2026-10-21 (receipt 8d8b6147-e2fb-4e7f-ae90-2d150697de6c). TL/operator priority changed at12:54 PDT: park #769 and start #757 now (receipt2fe48cdc-1a0b-4a7d-a692-e2c3852ba839). Preserve PR524/reviews/suite, then re-enter its merge after the shared cache is cleared.

**Goal:** Grow any HTTP web app screen by screen behind the existing mock host, while production uses the app's own build and never serves mock content.

**Architecture:** `couch mock serve DIR --real http://localhost:3000` optionally fronts an already-running application. One versioned `screens.json` declares real/mock status and navigation routes. The existing host still owns mock versions, fixtures, scanning, confinement, selection and reload. A read-only `couch mock check DIR` reports missing screens. A standalone Go example demonstrates the convention without making Go a requirement of the host or production app.

**Tech Stack:** Existing Go mock server, net/http/httputil, x/net/html for mock-page chrome, JSON; a separate standard-library Go sample and fixture-contract tests. No new framework runtime or application process supervisor.

## Revisions and authority

- 2026-10-21 11:00 PDT: operator APPROVED the framework-neutral draft and all four stated limits (TL receipt 8d8b6147-e2fb-4e7f-ae90-2d150697de6c). Mocks run in Go via Couch; the real app uses any stack and its own command. Production uses only that stack with unfinished-page/popup behavior; strict means no mock content ships, not build refusal. mock check reports; Go is a worked sample. This approval supersedes the pending-confirmation status in the historical entries below. Commit/implementation remain sequenced after #769 landing.

- 2026-10-21 09:06 PDT: operator approval relayed by TL (receipt 9e25f576-8ab1-4752-a2c4-21084fb68b10). Supersedes production refusal: builds succeed with missing screens; missing production routes show a plain `Not implemented yet` page with a way back, links visibly identify missing screens, and no mock code, fixtures or MOCK badge ships. Build prints a missing-screen report. Two-command Go-specific development was initially approved.
- 2026-10-21 09:07 PDT: TL relayed the operator's any-framework requirement and proposed moving development dispatch into `couch mock serve --real`, adding `couch mock check`, retaining Go as one example (receipt d475db2c-2dd9-4cee-8bd4-a20901b900ce). This supersedes the Go-only router/build-wrapper architecture. **Framework-neutral proposal awaits operator confirmation.** No change-code, estimate or implementation yet.
- TL instructed scratch-only drafting while #769's retained runner waits; commit to #757 after #769 lands. If GitHub remains degraded at 10:30 PDT, report to hub for reconsideration (receipt 658d29be-e8b8-4a98-bf42-1212130e50fb).

When committing, preserve the original plan as the explicitly superseded proposal and append this revision/current design; reconcile issue Spec and Done when with a dated Revisions entry. Do not erase the earlier decisions or Log.

## Approved contract

### 1. One screen catalogue

Project root `screens.json`:

```json
{
  "version": 1,
  "screens": [
    {"id":"index", "route":"/", "mock":"index.html", "state":"real"},
    {"id":"runs", "route":"/runs", "mock":"runs.html", "state":"real"},
    {"id":"detail", "route":"/runs/detail", "mock":"detail.html", "state":"mock"}
  ]
}
```

`state` is explicit; neither source-code discovery nor upstream HTTP errors infer it. A declared real screen returning an error stays a real application error. IDs, exact canonical routes and version-relative `.html` paths are unique. Root `/` is the one index screen. Canonical routes contain no query/fragment, traversal, reserved mock prefix or encoded separators. Mock paths use the existing ScreenFromPath rules. Dynamic route-pattern inference is outside this first convention; applications may choose representative concrete screen URLs and own their other routes.

Known fields/types and duplicate keys are strict; unknown additive fields are tolerated at root and nested screen levels. Couch reads but never rewrites this file, so extension fields remain byte-for-byte on disk. Unsupported major versions refuse with a named repair error. Limits: 256 KiB catalogue, 256 screens, bounded route/ID lengths. `screens.json` is project content, counted within existing entry/byte caps and scan revision; it is private over HTTP. Check requires only this file, not fixtures or a mock folder, so production pipelines need not retain mock content.

Without `--real`, preserve #755 behavior: pure versioned mock viewing, no catalogue required and no hybrid route interception. A catalogue file, if present, remains ordinary hashed private project content. With `--real`, catalogue is required and validated before listening. The front continues to require a valid #755 mock project even when every screen is real; retain the version folders/fixtures as design history. Only `check` and the app production build work without mock content. Removing that requirement would be a separate explicit lifecycle change.

### 2. Framework-neutral development front

The application starts with its own existing command. The single Couch command starts the front only; it does not start, restart, build or own that application. `--real` accepts an explicit HTTP loopback origin with a port and no credentials/path/query/fragment. Pin resolution to loopback and reject self-targeting once the front listener is known. Never route to an arbitrary destination taken from a request.

Routing precedence is explicit and decided once per request from one immutable published catalogue/revision snapshot:

1. Reject malformed/noncanonical paths before either disk access or upstream proxying. Reserved private project files, `/__couch_mock/*` unknowns and malformed `/mock-*` namespaces never fall through to the real app.
2. Existing state and reload routes stay owned by Couch. Shared `/fixtures.json` and confined `/mock-N/...` assets stay mock-owned and GET/HEAD-only. Unknown `.html` document aliases in the mock namespace refuse locally instead of being served as undeclared screens; non-document assets remain confined to validated version folders.
3. A declared canonical screen URL or its `/mock-N/<mock path>` alias resolves through the catalogue. Real screens redirect version aliases to the canonical URL and proxy there; mock screens redirect canonical URLs into the selected mock version and use existing confined mock delivery. Existing selected-version redirects preserve queries. Registering a screen means editing only its `state` and implementing that route in the app.
4. All other non-reserved routes proxy to the configured app, including its assets, APIs, normal HTTP methods and WebSocket upgrades used by development tooling. This supports a framework's HTTP interfaces without guessing filename suffixes or duplicating API-prefix configuration. The catalogue is a navigation convention, not an authorization boundary around the real app.

Mock HTML receives an authoritative visible MOCK badge plus `2 of 3 screens real` from the current catalogue; preserve existing data-mock-version/screen, bootstrap and reload. Real responses pass through unchanged, including CSP, compression, cookies and streaming. Print progress on startup/catalogue changes so it is available while viewing a real screen. **Accepted limit:** progress chrome is guaranteed on mock pages, not injected into arbitrary real HTML. Real pages may display catalogue-derived progress in the example.

Known same-upstream absolute redirects are rewritten to the front origin; relative redirects keep ordinary semantics. Strip supplied forwarding headers and set proxy-owned forwarding information. Upstream failures produce an explicit gateway error and never mock fallback. External links remain external; there is no arbitrary HTML/JavaScript URL rewriting. Applications must use front-relative navigation and full document loads when crossing between real and mock screens. SPA transitions without an HTTP request cannot be intercepted by a generic front; the README and sample demonstrate the required navigation boundary.

HTTP body streaming uses bounded buffers and context cancellation, not whole-response buffering. Reuse listener lifecycle and one existing scan/watch loop. Track upgraded connections owned by this front, close them on cancellation and join their copy loops; net/http Shutdown alone does not collect hijacked sockets. Keep ReadHeaderTimeout=5s, IdleTimeout=30s and MaxHeaderBytes=16KiB. Hybrid listener WriteTimeout=0 permits app SSE/streaming; existing standalone listener keeps its 10s write timeout. Hybrid active-request/upgrade admission is capped at64, with cancellation-owned lifetimes. Give the owned upstream transport 3s dial and 5s response-header budgets; no total body timer. Tests gate blocked streams and upgrades on explicit barriers, then cancel and assert joined cleanup. No polling loop beyond the existing scanner or browser reload loop.

### Declared mock delivery integrity

Hybrid mode validates every `state:mock` catalogue row against the selected version at startup and each scan, using the same scan observation's validated entry inventory. Each row must name its exact available HTML document. Missing or unsafe declared documents refuse startup; a version switch, catalogue edit or filesystem change making one unavailable publishes the existing explicit repair/error state until repaired. A successful standalone `mock use` pointer write may therefore produce a hybrid repair state; it does not silently choose another screen or undo that selection. At request time, disappearance or replacement after a healthy scan returns a named503 unavailable-declared-screen response, never a redirect to index, never upstream fallback. Recovery follows the existing scan error/recovery revision. Standalone mock mode keeps its existing missing-screen index fallback (`server.go` ServeHTTP) and index-only scan requirement (`scan.go` Project.Scan). Hybrid uses an explicit strict-document delivery path, not an accidental reuse of that fallback.

### 3. Missing screens in production

`couch mock check DIR` parses the same catalogue and prints a deterministic progress summary plus each `state:mock` screen's ID and route. Missing implementations are informational: exit 0 so the app's normal build proceeds. Invalid/unreadable catalogue is a command error, not a missing-screen report. Check opens no listener, executes no build and reads no fixtures. It writes no files.

Each framework owns its production routing and build. The documented convention is to derive a finite production screen list/status table from `screens.json`; `mock` means the implementation is missing, not permission to serve a mock. A missing route renders only `Not implemented yet` plus a link to the home screen. Links to missing routes carry visible `Not implemented` text. No generated mock URLs, mock scripts, fixture bytes or development badge are included in production artifacts. The generic check reports status; it cannot enforce arbitrary framework bundles. Documentation states that ownership explicitly.

The Go worked sample enforces the convention: its production command invokes its own Go build after the report, even with detail missing. The sample reads the same JSON file as a consumer of its documented language-neutral schema; it has no second status declaration and no production dependency on Pair internals. Its build validates that input and emits a minimal projection. Production imports only real handlers and that generated catalogue projection containing IDs, routes and implementation states; mock paths, mock host/proxy, fixtures and development chrome are excluded from that projection/build. Production navigation and placeholder routing derive from that same projection. A compiled registration marked real but missing its real handler is a named configuration error; normal application handler failures are not placeholders. Add detail + switch its catalogue state to real, rebuild and prove the placeholder/report disappears. A private build output is owned by the build/test scope; no release wrapper lives in Couch.

### 4. Fixture shape contracts remain useful across frameworks

Keep one shared fixtures.json for mocks. The example's real backend uses its own application data, not fixture bytes. Contract tests decode real HTTP responses and compare required recursive JSON shapes with representative fixture subtrees. Required/nested fields, primitive/null types, all array items and size/depth limits are enforced; extra response fields allowed. Empty response arrays valid against a nonempty prototype; empty or heterogeneous fixture arrays refuse as ambiguous contract definitions. Never mutate fixture content. Document the language-neutral algorithm with the Go helper as a worked example; do not introduce a cross-language contract runner in this issue.

## Alternatives and choice

1. **Recommended:** optional front in existing Couch mock host; read-only check and documented production convention; one Go worked sample. Reuses ownership/security/reload and serves any HTTP app.
2. App-specific middleware for each framework. Better integration with client routers, but requires adapters and expands the supported-framework matrix before evidence.
3. Inject global navigation interception and chrome into arbitrary real pages. Appears transparent but breaks CSP/streaming/framework ownership and cannot prove all client-router behavior. Excluded; document full-document cross-boundary links instead.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Catalogue, ParseCatalogue, ResolveRoute, ResolveMock, Progress, Report | cmd/internal/mockserve/screens.go | new |
| frontRoute / decideFrontRoute | cmd/internal/mockserve/front_route.go | new |
| TransformDevelopmentPage | cmd/internal/mockserve/page.go | modified |
| Shape, Infer, Match | examples/mock-to-real/internal/contract/shape.go | new |
| app.Screen and prepared navigation policy | examples/mock-to-real/internal/app/app.go | new |

Catalogue owns immutable route/alias maps and one computed progress value; no duplicated app-side Go-only declaration. frontRoute / decideFrontRoute yields mock document/asset, internal/private/refusal or real proxy without IO. Existing version selection remains its sole owner. Mock decoration remains a pure bounded HTML transform, shared with existing bootstrap work. Contract/projection tests use literal values and no IO mocks. Extensions to the catalogue are additive/read-only; future route patterns would be an explicit widened contract.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Project catalogue observation | cmd/internal/mockserve/scan.go, server.go | modified | confined reads and existing scanner |
| developmentFront and upstream lifetime | cmd/internal/mockserve/front.go | new | real app HTTP/WebSockets |
| CLI serve/check | cmd/internal/mockserve/cli.go, check.go; couchcmd help/tests | modified/new | command IO, no Couch runtime |
| Sample catalogue consumption | examples/mock-to-real/internal/catalogue/catalogue.go | new | bounded io.Reader consumption and IO errors |
| Go sample real backend and build | examples/mock-to-real/cmd/{app,build}, internal/{app,buildsample} | new | app listener, real handler responses and injected compiler executor |
| Template qualification | cmd/internal/mocktemplate/template_test.go | new | private nested module, actual host and build |

HTTP tests use a stateful local fake app with actual routing, redirects, cookies, errors, APIs and upgraded connections. Live conformance uses the real mock host and sample app on private loopback ports. No external framework installation is necessary; a framework-shaped fixture exercises asset/API/HMR routes without claiming qualification of every framework. Sample production builds/launches are owned and joined in private roots. Existing shared Pair candidate path is used, never a freshly copied Pair executable.

## State design

Actions: start app, start front, edit catalogue/mock/fixtures, implement/register screen, retry request, run check/build, stop front/app.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Plain mock serve without --real | existing versioned preview | edit mock/fixtures | existing reload transient state |
| Valid catalogue, two real/one mock | clickable hybrid; MOCK/count on mock pages | implement/register detail | browser transient state on reload |
| Upstream down | real requests gateway error; healthy mock pages still work | start app, retry request | in-flight app request may have unknown result; no automatic retry |
| Real handler error | unchanged real application response | repair app, retry request | app-defined request effects |
| Invalid catalogue at startup | named refusal before listener | edit catalogue then start front | no durable work lost |
| Invalid catalogue during watch | explicit repair state; no stale routing/progress claim | repair catalogue | browser stays visible; new document traffic fails while invalid |
| Broken mock project | existing visible repair/recovery state; front document/proxy traffic refuses until repaired | repair mock/fixtures | existing preview transient state; no stale routing decision |
| Valid status edit | next scan publishes route/count/revision together | retry/navigate after reload | current mock page reload resets transient state |
| Production has missing screens | build succeeds with report; placeholder and marked links | implement/register then rebuild | no mock data shipped |
| Invalid catalogue or real registration mismatch | named check/build/start error | fix catalogue/registration | old build artifact may remain; no false success |
| Backend shape drift | endpoint + JSON path test failure | repair backend or consciously revise fixture | no fixture rewrite |
| Stop front with active request/upgrade | owned connections/copy loops end | start front again | in-flight requests/stream state |

## Cost, bounds and lifecycle

Reuse existing scan limits (1,024 entries, 32 MiB contents, 1 MiB HTML) and 500ms scan cadence. Catalogue counts as one ordinary file; parsing and immutable indexes are O(S+B), once per scan snapshot, not per request. One mutex publication pairs route catalogue with revision/health. Request route lookup O(1); only mock HTML transformation O(H), bounded at 1 MiB. Count reads/parse work at N/2N screens and assert zero scans/parses on N/2N requests. Existing scanner hashing O(B) and metadata sort O(F log F) stay unchanged; no second watcher. Shape comparisons O(J), <=1 MiB JSON and depth64.

Retain current finite listener/header budgets; bound proxy header bytes and dial/header waits, use bounded stream buffers, and explicitly cap active front proxy/upgrade connections (proposed 64; excess visible 503). Cancellation closes idle transport and owned upgraded connections and joins all work. Do not reuse a global default transport with unowned idle pools. A handler may stream until cancellation; no whole-body aggregation or duration-based retry. The app is external and survives front shutdown.

No per-request/per-scan durable artifact, cache, generated mock family or catalogue writer. Existing `.mock-current` family remains host-owned. Example's generated production projection is one fixed build-local file replaced by that build and removed with its private/staging root; output is one explicitly selected operator-owned artifact. Tests remove roots after joining children. fixtures/catalogue/mock folders are developer-owned git content, no automatic rewrite/removal.

## Verification strategy by function

| Risky function / owner | Adversarial input class | Mechanical guard |
|---|---|---|
| ParseCatalogue | ambiguous or hostile JSON/schema/route identifiers | table-driven classes and seeded fuzzing, strict duplicate/known-field validation and byte/element bounds |
| ResolveRoute / ResolveMock / decideFrontRoute | alias/reserved namespace collisions and malformed URLs | pure decision tables plus fuzzing; upstream spy must observe zero forbidden dispatches |
| Report / Run check | incomplete versus invalid input and command side effects | exact deterministic report/exit assertions with only a catalogue present; filesystem/listener/process effects remain absent |
| Project.Scan / refresh / observation | selected catalogue/document mutation across observation boundaries | barrier-controlled replacement and selection changes, atomic published health/catalogue/revision; strict hybrid missing-document failure and recovery |
| developmentFront.ServeHTTP | real/mock dispatch under upstream errors and mixed HTTP traffic | stateful fake app, exact body/status/origin assertions and request cancellation barriers; no error fallback |
| developmentFront.Close / upgrade ownership | cancellation while streams/upgrades remain active | explicit connection-ready barriers, cancel owner then assert all copy loops/fds joined; overload admission count |
| TransformPage / TransformDevelopmentPage | hostile bounded HTML and conflicting owned metadata | pure transform assertions and fuzz seeds preserving project scripts/identity; no real HTML modification |
| Reload client lifecycle | status/version/repair observation races | existing deterministic JS generation fences plus bounded private-browser full-navigation conformance |
| Infer / Shape.Match | arbitrary nested JSON and ambiguous array prototypes | seeded fuzzing and JSON-path failure assertions under depth/byte bounds; mutation demonstrates backend drift caught |
| app.Screen / app.New / catalogue.Read | drift between catalogue and navigation/handler registration; reader failure | pure prepared-route assertions plus bounded reader integration tests and private partial/complete sample copies |
| buildsample.Build / executor.Run | compiler failure, partial candidate, validation failure and cancellation order | stateful compiler fake observes stage/candidate/publication; cancellation barriers prove join before retirement; actual Go builds retain live conformance and artifact sentinel checks |
| rewriteRealLocation / targetsListener | origin-only, query/fragment, double-slash redirects and distinct loopback addresses | resolve references against a non-root front URL; exact pinned address/port tables reject genuine self-targets only |
| Screen lookup / scan / shape work | N versus2N input/request growth | counted IO/node/transform operations, zero per-request scan/parse work; timing is not the oracle |
| Nested template qualification | root Go suite silently omits nested module | root integration explicitly runs sample tests/build and real mock host against private owned roots |

## Implementation tasks — only after approval and change-code

All implementation follows RED → observed failure → minimal implementation → GREEN for the named strategies above. Tasks below assign files and owners without duplicating executable test cases in prose.

### 1. Catalogue, report and route policy

Files: mockserve/screens.go, screens_test.go, front_route.go, front_route_test.go, check.go/check_test.go; cli.go/cli_test.go and couchcmd/mock_test.go/run.go.

- [x] Implement immutable catalogue, pure dispatch/report policy and their named strategies.
- [x] Extend ParseCommand/Run for --real and mock check, preserving runtime-independent Couch dispatch and documenting command exit behavior.
- [x] Run targeted mockserve/couchcmd checks and commit the verified catalogue/CLI layer.

### 2. Front, shared scan publication and browser behavior

Files: mockserve/front.go/front_test.go; scan.go/scan_test.go, server.go/server_test.go, page.go/page_test.go, browser_test.go; reload.js/reload_test.js only if catalogue-change navigation needs it.

- [x] Integrate catalogue and declared-document validation into the existing confined scan observation; publish health/catalogue/revision together without a new watcher/read owner.
- [x] Implement optional front, exact hybrid document delivery and mock-only chrome; keep upstream response bytes and standalone mock semantics intact.
- [x] Implement context-owned proxy transport, capacity admission and upgraded-connection shutdown under the named lifecycle strategy.
- [x] Run targeted race/JS checks and bounded private-profile browser conformance; commit the verified host feature.

### 3. Worked Go sample, production convention and contracts

Files: examples/mock-to-real/go.mod, cmd/app/main.go, cmd/build/main.go, internal/{app,catalogue,jsonvalue,buildsample,contract}, mocks/{screens.json,fixtures.json,mock-1/*}, README.md and design.md, .gitignore.

- [x] Implement independent real backend, typed shape contracts and single-catalogue sample consumer using their named verification strategies.
- [x] Implement app-owned production build/report/projection, missing-screen page/back link and marked navigation; qualify partial and complete production artifacts in private roots.
- [x] Document any-framework usage and full-document boundary; run nested race checks and commit the worked example.

### 4. Integration and one close boundary

Files: cmd/internal/mocktemplate/template_test.go; artifactpath source classifications/per-reader certificates; README.md, atlas/mock-server.md, atlas/mock-to-real.md and atlas/index.md.

- [x] Integrate nested qualification into the root suite; audit new readers/source classifications and run inventory/artifact/repolint checks.
- [x] Update public docs for --real, report-only check, production responsibility and accepted limits; retain historical refusal only as superseded provenance.
- [ ] Push/PR, recorded affected/suite, close with binary-owned mandatory review, merge and report hub. One atomic close boundary, no Mx tag.

## Architecture lenses

ARCH-DRY: reuse host; one catalogue feeds dispatch/progress/report/sample production projection. ARCH-PURE: parse/route/report/shape/decoration separate from IO. ARCH-PURPOSE: prove arbitrary HTTP app boundaries and both partial/complete production builds, not just sample screenshots. ARCH-MOCK: stateful app fake + real host/sample/browser conformance. ARCH-CONSTRAINTS: fixed file/body/header/concurrency bounds and counted hot paths. ARCH-SECURE: loopback destination, reserved path precedence, confined mock IO, no automatic fallback on real errors. ARCH-ORDER: atomic catalogue/revision publication and scoped request/upgrade ownership. ARCH-FUNERAL: no growing generated state, fixed build projection + scoped cleanup. ARCH-EVOLVE: additive catalogue fields preserved by read-only consumption, strict known fields and no hidden writer.

## Accepted limits (operator approval 2026-10-21 11:00 PDT)

- This is one front command for an app that is already running; it does not supervise the app.
- Any HTTP framework can sit behind it; SPA client-side transitions require full navigation when crossing the boundary.
- Badge/count are guaranteed on mocks; real HTML is passed through unchanged, with progress also printed in the console.
- `mock check` is informational on missing screens and does not certify arbitrary framework production bundles; the Go example enforces and tests the production convention as a worked sample.

## Revisions — priority and cache

- 2026-10-21 12:54 PDT: TL/operator made #757 high priority and explicitly superseded the wait-for-#769 sequence. Park #769, keep PR524/suite/reviews, start #757 now. Its retained runner had already exited1 when train238 held its merge for conflict; no live process was stopped. Re-entry must follow the printed integrate/re-close recovery after the cache is cleared. All Go build/test subprocesses use private GOCACHE at $HOME/.cache/go-build-757 on main SSD, never TestScratch; remove that private cache when done.


## Revisions — implementation layout and verification integration

- 2026-10-21: Aligned conceptual table names with actual focused implementation APIs; sample owns internal/app, catalogue, jsonvalue, contract and buildsample. No new product policy. Root qualification uses suite-owned testbinary.Cache for its distinct sample executable; every process joins before TestMain removes it.
- The nested standalone Go module has audited source-reading tests that the old fixed preflight cannot address as root-module imports. Added explicit finite SOURCE_MODULES grouping: existing root scanner argv remains exact, nested audited packages run from their own module before provider-selected packages, and any preflight failure stops selected effects. Statefully recorded child-executable tests cover commands/failure propagation; no discovery or widening of the selected-package argv. Reader certificates remain per file.
- Final adversarial regressions exposed blocked incomplete-upload cancellation and zero-padded self-origin ports. Normalize origin ports before self-target checks; owner cancellation wakes downstream reads/writes and joins its callback before returning. Both regressions observed RED before fixes.

## Revisions — close round1

- 2026-10-21: BR-1/BR-3 preserve resolved redirect destinations and compare complete pinned loopback endpoints; the same endpoint policy now drives dialing and identity. Regression tables observed behavioral RED for root/query/fragment/double-slash redirects and distinct-address same-port listeners before GREEN.
- BR-2 corrects ARCH-PURE traceability: catalogue.Read consumes io.Reader and belongs to INTEGRATION; app.Screen/prepared navigation remain pure. Active paths name the delivered app/buildsample modules. Historical proposal remains explicitly nonbinding.
- BR-4 adds a private injected compiler executor (ARCH-MOCK). Stateful candidate files and cancellation barriers pin compile/validation failure cleanup, validate-before-publish and worker joining; real compiler tests remain conformance. BR-5 will retain final-head host browser navigation and delivered-progress negative-control evidence before re-close.

## Historical initial proposal — superseded, retained for decision provenance

The following initial Go-only router and production-refusal proposal is nonbinding. The approved framework-neutral design above and dated Revisions replace it.

<details>
<summary>Initial proposal before operator revisions</summary>

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

</details>

## Revisions — plan-quality round1

- 2026-10-21: PQ-1 clarified exact declared-mock document integrity at startup, scan/selection changes and request-time disappearance, preserving standalone index fallback. PQ-2 replaced active prose test inventories with named per-function adversarial strategies and mechanical guards. These refine approved error visibility and test execution without changing product scope.
