---
collaboration-mode: discussion
---

# Versioned Couch mock server implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve a project's clickable, versioned mocks with shared fixtures, stable screen identity and browser reload after edits or selection changes.

**Architecture:** A standalone `mockserve` package owns project selection, HTTP serving and one content-scan loop. Pure parsing and HTML transformation sit behind a thin filesystem/network shell. The Couch CLI dispatches before supervisor/runtime preparation.

**Tech Stack:** Go net/http, os.Root, SHA-256, atomic file replacement; golang.org/x/net/html for HTML parsing; plain browser JavaScript fetch polling; existing CDP transport for headless qualification.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| Version | cmd/internal/mockserve/project.go | new |
| Screen | cmd/internal/mockserve/page.go | new |
| RevisionState | cmd/internal/mockserve/revision.go | new |
| Command | cmd/internal/mockserve/cli.go | new |

- **Version:** validated `mock-N` or `mock-Na` name, numeric ordering plus optional lowercase variant. One project has many versions and exactly one selected version. One parser handles pointer, CLI and URL input (ARCH-DRY); future variant naming extends that parser.
- **Screen:** normalized relative HTML path without the extension, with `index.html` mapped to `index`. One version has many screens. HTML transformation parses a document, replaces both root identity attributes, and inserts the reload client once. Test as pure bytes-to-bytes logic, including fragment HTML and existing attributes. Future screen metadata belongs here.
- **RevisionState:** tagged healthy/error observation and a content fingerprint. Pure transition from previous observation plus next scan result; identical observations do not advance the revision. Recovery from an error is observable even when repaired contents match the last healthy contents. This is the only publisher of reload state (ARCH-ORDER).
- **Command:** pure closed argv parser for serve/use/help, with validated port/version. Couch delegates the tail without extracting global layout flags. Future mock commands widen this parser.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Project | cmd/internal/mockserve/project.go | new | root-relative filesystem reads and selection writes |
| Snapshot | cmd/internal/mockserve/scan.go | new | bounded content walk/hash |
| Server | cmd/internal/mockserve/server.go | new | HTTP routes and one cancellable scan loop |
| Run | cmd/internal/mockserve/cli.go | new | loopback listener, signals and output |
| ReloadClient | cmd/internal/mockserve/reload.js | new | browser fetch, navigation and visible scan errors |
| Couch mock dispatch | cmd/internal/couchcmd/cli.go, run.go, cost.go | modified | public CLI |

Project is a portable folder; tests use private real directories and HTTP servers, never operator state. No production browser or external service is required by the server. Headless browser tests qualify the real browser behavior through existing CDP; HTTP clients and private filesystem fixtures are the deterministic integration boundary. No new function-call fake of a stateful external service is introduced (ARCH-MOCK).

## State and operating contract

Actions: serve, edit files, select a version, repair invalid files/selection, interrupt server, revisit URL.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|------|------|------|------|
| Startup has a valid project and selection | printed loopback URL, current mock | revisit URL | none |
| Startup invalid, missing fixtures/version, over limits, or bind fails | command exits with specific error | repair, then serve | no files changed |
| Healthy scan changes content | current page reloads | edit again | transient browser-only state resets |
| Selection changes | version URL and root attributes change together | select prior version | transient browser-only state resets |
| Current screen missing in newly selected version | new version index opens | revisit a linked screen | prior URL screen lost, files retained |
| Runtime scan invalid/unreadable/over limit | browser error banner, state endpoint refuses | repair input | last visible page remains; no success fabricated |
| Runtime scan recovers | banner clears and page refreshes | revisit URL | transient browser-only state resets |
| Pointer write fails before rename | command error, prior selection retained | select again | owned temporary file removed |
| Process dies after pointer rename | selection discoverable on disk | select again or serve | no rollback of confirmed replacement |
| Concurrent use commands | each writes whole pointer; last completed rename wins | select intended version | earlier selection superseded |
| Server interrupted or browser disconnects | listener closes / browser retries bounded requests | serve again | no mock artifacts deleted |

Pointer writes use same-directory temporary file, close then atomic rename; temporary files are cleaned on failure. A write error after publication reports uncertainty and instructs rereading rather than claiming old state. No append-only runtime store or per-launch file is created. `.mock-current` is one bounded project-owned text file, replaced in place, removable with the mock folder. Versions/fixtures are operator-authored project artifacts retained until explicitly removed (ARCH-FUNERAL, ARCH-EVOLVE: pointer is a strict single-value text format, no JSON rewrites or unknown-field loss).

The server owns one context, scan loop and ticker; cancellation joins the loop and shuts down the HTTP server with a bounded shutdown context. Browser request timers belong to the document and stop on pagehide; requests have cancellation timeouts. Repeated failed polls do not accumulate requests or navigation. The pure revision model handles healthy→changed, healthy→error, repeated error and error→healthy; one mutex protects publication and readers only consume copies. No per-browser filesystem watchers or mutable session records.

Envelope (domain-informed initial choices for small local mocks): 1,024 regular files, 32 MiB total and 1 MiB per HTML document, one scan per 500ms at most, at most one pending fetch per browser. Scanner work is O(F+B) for F file entries and B content bytes; client polling is O(C) cached reads for C browsers and never scans. Files exceeding the envelope refuse with the actual limit named. No claim that potentially stalled filesystem IO is cancellable; check context between entries/reads and use bounded buffers. HTTP has header/idle/request limits; local origin only, no CORS, no write endpoint. Credentials and auth are absent by issue design (ARCH-CONSTRAINTS, ARCH-SECURE).

Root handling rejects traversal, encoded separators/invalid version paths and symlink entries. `os.Root` confines actual reads against concurrent path replacement. Root `design.md`, pointer and other files are never HTTP routes. Fixtures must be valid JSON, read-only and bounded, and their bytes are served unchanged; schema remains project-owned. Pages are trusted project code, not sandboxed content. HTML parser handles existing attributes/doctype; renderer escapes injected values rather than concatenating untrusted HTML.

## Test strategy (one authoritative inventory)

| Test | Adversarial behavior / oracle |
|------|------|
| TestVersionSelection | natural numeric order, variants, explicit older version, invalid/truncated pointer, removed target; no fallback for invalid explicit choice |
| TestUseAtomicSelection | valid selection persists across restart, unknown version leaves pointer unchanged, failed publication retains prior value, concurrent whole writes; no temporary residue |
| TestPageIdentity | existing/conflicting attributes replaced, fragment/mixed-case HTML, nested paths, escaped data; rendered root owns exactly one pair of authoritative attributes |
| TestRevisionTransitions | same-content scan no reload, changed content reload, repeated errors stable, repaired tree publishes recovery even with old content |
| TestScanLinearWork | N and 2N files/content, count visits and bytes, reject worse than linear; cached HTTP state reads add zero visits |
| TestScanChangesAndBounds | additions/deletions, same-size restored-timestamp edits, CSS/JS/fixture edits, cap boundary/overflow, symlink refusal, invalid JSON and repair |
| TestHTTPProjectRoutes | root redirect, HTML identity, assets, single shared fixtures across versions, absent screen fallback on switch, private root files/traversal/symlink escape refusal, correct MIME/no-cache |
| TestServerLifecycle | real loopback listener, printed URL after listening, no Couch initialization, cancellation joins watcher and server, bind refusal, no background work after return |
| TestMockCLI | complete/extra/incomplete forms, invalid ports/versions, help, global-layout-like path values opaque, explicit `couch -- mock` still launches directory |
| TestMockBrowserReload | real temporary-profile headless Chrome: click all sample screens and read shared fixture data, DOM attrs, HTML/CSS/JS/fixture edit reload, mock-1→mock-2→mock-1 URLs/attrs, runtime error banner and repair; synchronize on DOM/generation predicates |

Keep production mutation negative controls: remove root tagging and browser reload independently and see their tests fail; make state polling rescan and make visit-count test fail. Use no fixed sleeps as synchronization. Browser qualification is opt-in with the repository's live Chrome convention and run explicitly before close; deterministic HTTP/model tests run in the normal suite.

## Implementation tasks

### 1. Project and pure document contract

Files: new `cmd/internal/mockserve/{project,page,revision}.go` with colocated tests; module files for HTML parser.

- [ ] Write VersionSelection, PageIdentity and RevisionTransitions tests; run focused package tests and retain red evidence.
- [ ] Implement typed version/screen parsing, natural selection, pure HTML transformation and the revision transition owner.
- [ ] Add root-confined reads and atomic pointer selection; implement UseAtomicSelection against private directories. Verify failure/uncertain publication and repeated selection behavior.
- [ ] Run `go test ./cmd/internal/mockserve -count=1`; commit passing core with agent/coauthor trailers.

### 2. Scan, HTTP and reload lifecycle

Files: new `cmd/internal/mockserve/{scan,server}.go`, `reload.js`, colocated tests.

- [ ] Write scan changes/bounds/linear tests and HTTP route tests before implementation; confirm red.
- [ ] Implement bounded context-aware scan and cached revision publication. Inject scan timing/entry observation for deterministic scheduling/count oracles; production does not rewalk for requests.
- [ ] Implement version routes, shared fixtures, transformed HTML and static assets; embed the reload script. Use root-confined reads, GET/HEAD only, no-cache, and no directory listing.
- [ ] Implement one-pending-request browser polling; reload on revision, navigate on selection, show repairable scan errors. Preserve same-screen search/hash; fall back to new index when absent.
- [ ] Add real listener lifecycle and headless browser tests using private profiles/CDP. Use context cancellation for every process, connection and goroutine; join owned work.
- [ ] Run targeted package tests plus race detector and explicitly run browser qualification; apply the listed negative controls. Commit verified server.

### 3. Public CLI, sample, documentation and landing

Files: new `cmd/internal/mockserve/cli.go`; modify `cmd/internal/couchcmd/{cli,run,cost}.go` and tests; new `examples/couch-mocks/` with `design.md`, one `fixtures.json`, two version folders, three linked screen paths plus shared styles/script in each; README and atlas pages/index; artifact/source manifests only where required.

- [ ] Write parser and early-dispatch tests, confirm red, then add `couch mock serve/use` and help before runtime preparation. Use `--port` only after the directory in the closed serve form. `use` needs no running server.
- [ ] Build the sample with three linked screens in both versions. Read the same fixture data from every screen. Document normal/empty/error/long-list scenarios in its design state list and UI, with fixtures matching the future API contract.
- [ ] Document layout, default/current selection, version URLs, commands, live reload/state reset, input limits, shared fixtures and state-list convention; link atlas/index and root README references.
- [ ] Classify every new production file in artifactpath; audit any reader inventory entries individually. Run `make test-affected-inventory`, artifactpath and repolint checks, package tests/race and real sample browser qualification.
- [ ] Commit/push and open PR; run recorded `sdlc local-check run --name affected` (full fallback when declared). Run inventory immediately before close, then `sdlc close --issue 755 --verified '<returned evidence>'`; resolve mandatory review findings and rerun checks required by any code change.
- [ ] Enter `sdlc merge --yes`, preserve runner/order, require integration suite and conformance green. Report actual landed PR/evidence to hub and verify submitted receipt. Then start pair#757, with no unrelated work interleaved.

## Architecture decisions

ARCH-DRY: one selection parser, one fixture file and one server scan, not scans per browser. ARCH-PURE: version/document/revision transformations tested directly, separate IO adapters. ARCH-PURPOSE: sample proves clickable multi-screen iteration, actual reload and revert. ARCH-MOCK: portable folder/HTTP integration and real private browser conformance. ARCH-CONSTRAINTS: enforced project/HTML bounds and counted scan growth. ARCH-SECURE: root-confined reads, explicit public routes and no remote write/auth surface. ARCH-ORDER: atomic selection and single revision transition publisher; uncertain writes retain observable disk state. ARCH-FUNERAL: owned temporary files, context-owned processes/workers and one persistent pointer. ARCH-EVOLVE: no rewriting project fixture schemas or unknown JSON fields.
