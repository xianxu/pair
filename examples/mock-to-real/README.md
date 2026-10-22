# Mock to real: one worked Go sample

Couch hosts the development front for **any framework** that serves real HTTP.
Go is one standalone worked sample, with its own module and production build;
it has no Pair package dependency. These commands require Go 1.26.3+ and a
Unix host (the example build owns child process groups on macOS/Linux).

From this directory, start the real app:

```sh
go run ./cmd/app --listen 127.0.0.1:8081
```

In a second terminal, start the Couch development front:

```sh
couch mock serve mocks/ --real http://127.0.0.1:8081
```

Open the URL printed by Couch. `/` and `/runs` are real; `/runs/detail` is a
mock. Links use canonical same-origin routes, so following a link crosses the
boundary through a full document navigation. SPA client routers must allow
these navigations through; Couch does not automatically intercept SPA routing.
The app command starts only the app. Couch owns switching, proxying, version
selection, metadata, and reload. Real server errors remain real errors.

`mocks/screens.json` is the sole screen state catalogue. Each row declares its
`id`, canonical `route`, relative `mock` HTML path, and `state` (`mock` or `real`).
The read-only consumer uses the host contract: at most 256 KiB, 1–256 screens,
unique IDs/routes/mock paths, and one `/` screen using `index.html`. Routes must
be canonical and avoid the host’s reserved namespaces; mock paths must be
normalized relative `.html` paths. Unknown additive fields are accepted.
`mocks/fixtures.json` is the one illustrative data source for all mock pages.
Mock assets in `mock-1/` use body `data-screen`; Couch installs authoritative
`data-mock-screen` and `data-mock-version` on the HTML root.

To implement detail, add its handler under `"detail"` in
`internal/app.RealHandlers`, then change that catalogue row to `"state":"real"`.
Do both before building: a marked-real screen without a handler is a named
configuration error. Keep the mock as a reference during development.

## Production

**NEVER SHIP mock content.** Build through the application's own command:

```sh
go run ./cmd/build --output ./harbor-production
./harbor-production --listen 127.0.0.1:8082
```

The output path must not exist; a build never silently overwrites it. The
provided partial application builds successfully and reports:

```text
Unimplemented: detail (/runs/detail)
Built ./harbor-production
```

This is a report-only completeness check: unfinished screens do not fail a
build. Production detail displays plain “Not implemented yet” with a link back
to `/`; navigation labels it “Not implemented”. Once implemented and marked
real, detail serves the actual handler and disappears from the missing report.

The builder creates a temporary stage containing only `internal/app`, the Go
module file, and a generated main with an `id`/`route`/`state` projection derived
from the catalogue. It compiles and checks that candidate, joins child commands,
and removes the stage on success, failure, or cancellation. No mock paths,
fixtures, mock assets, contract tooling, or development catalogue reader are
production dependencies. Failed publication/reporting removes its own output;
an existing output stays untouched. The resulting binary needs no source tree.

## Verification

```sh
go test ./internal/contract ./internal/catalogue ./internal/app ./internal/buildsample
go test ./internal/contract -fuzz FuzzShape -fuzztime 5s
```

Contract tests infer recursive required shapes from the representative
`fixtures.runs` subtree and compare the independently populated real
`/api/runs` HTTP response. Extra response fields and empty response arrays are
allowed. Every array item must match; empty or heterogeneous fixture arrays
are rejected. Null, boolean, string, and number are distinct. Parsing is bounded
at 1 MiB and depth 64, and duplicate keys are refused. A negative control proves
field/type changes fail rather than comparing the fixture to itself.

Production tests build and launch their own candidate from a private source
copy. They check placeholders, marked links, report-only success, missing real
handler refusal, and actual real detail after completing that private copy.
They scan the artifact for unique fixture/mock sentinels and development package
paths, and verify fixture, mock, reload, and version endpoints return 404.
They also exercise active child cancellation, output preservation, failed report
cleanup, and stage cleanup. Host/browser integration belongs to Couch's tests.
See [design.md](design.md) for the framework-independent boundary.
