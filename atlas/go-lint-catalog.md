# Go lint catalog

Pair's ordinary repository test `internal/repolint.TestTrackedGoFilesPassCatalog`
imports `github.com/xianxu/ariadne/pkg/golint` through the pinned Go module in
`go.mod`. Rules are maintained upstream, not copied by weave. Update the module
pin to adopt later rules. The initial adoption also advances the runtime
`layergraph` dependency; `TestDeclaredDeps*`, `TestRunPrepare*` and
`TestDeclaredDepsLinkedWorktreePrimaryAnchor` qualify Pair's consumers.

The five seed rules detect shortened test deadlines, literal temporary roots,
builds in tests, tight elapsed-time bounds and fixed sleeps. Findings identify
the path, line, rule and incident that taught the rule. This is a syntax catalog,
not type analysis or a promise to detect every timing mistake.

## Scanner boundary

The test lists tracked paths with NUL-delimited Git output, then reads each
eligible Go source once. It includes probes and tracked third-party Go code.
As upstream, workshop prose and testdata are excluded and generated sources
are ignored. Untracked sources do not count until added to Git. A tracked file
removed from the working tree is ignored; other inventory/read/parse failures
and an empty inventory fail the test. The scanner runs in `go test ./...`, so
no extra Make or CI command is needed.

The test's `repositoryRead` boundary exposes tracked paths, main tracking/ref
facts, merge base, historical paths/blobs, path history and ancestry. Scanner
policy uses that typed boundary with either `gitRepository` or its stateful
fake. The fake keeps working files separate from the index, immutable commit
snapshots, parent links, refs and attached/detached HEAD. It derives answers from
state transitions; it does not replay canned Git command responses. A shared
contract runs on the fake and disposable real Git histories on every ordinary
package test run, including unknown refs/blobs and authority changes. Real Git
remains the normal repository-test adapter; no remote service is contacted.

For an intentional semantic exception, place
`//lint:allow <rule> — <specific reason>` on the finding's line or immediately
above it. Upstream rejects unknown, reasonless and unused suppressions. Explain
why the operation is safe: for example, stable UID-private rendezvous sockets
must agree across processes with different TMPDIR, and a suite-owned build is
not a per-test compile. Ordinary temporary storage respects TMPDIR with a short
allocation prefix when Unix socket length matters. Do not hide a readiness race
by increasing its deadline or shortening a sleep below the matcher threshold.

## Temporary sleep debt (#665)

M1's sole sleep exception is `internal/repolint/sleep-debt.json`. Each entry
names a repository path, enclosing function, canonical Sleep call and occurrence
ordinal; incidental line movement does not change identity. Inline sleep
suppressions are refused. Every current listed sleep must still produce its
upstream finding, and every new unlisted sleep fails.

The current file must be a subset of the file at the candidate/main merge base.
Configured main tracking identifies authority; detached CI may use its unique
remote main ref. Missing or ambiguous refs and malformed/missing previously
present debt refuse, rather than authorizing new debt. Initial M1 bootstrap
alone derives the original103sites from the committed pre-code source anchor;
it never admits debt from the candidate under test. Full-history checkout is
already configured in CI.

When replacing a sleep, remove its exact entry in the same change. M2 works
through this list, starting with flake-history packages. Keep the empty file
when complete so the ratchet continues to reject new debt. Run the scanner
locally with `go test ./internal/repolint -count=1 -v`.

## Test fixtures

Timing regressions drive the event under test after readiness: registration
budgets, lock contention, recorder drain timeout and Chrome's last-page grace
have private controllable seams with unchanged production defaults. A negative
check proves the operation completes while the competing actor remains held;
it does not require the host to finish within a few milliseconds.

`internal/testbinary.Cache` owns distinct test executables from the first build
through every test cleanup. Package TestMain closes it after m.Run; concurrent
lookups and repeated test counts share one build attempt, including failures.
Couch, its terminal registration fixture, the startup launch helper and the
Zellij tracing probe keep their own executable identity. Pair itself continues
to use `cmd/internal/testpair` and the existing suite-provided binary.

New mechanical mistake classes belong in ariadne's catalog with the incident
and direct matcher tests. Propose cross-repository rules to the hub; changing
Pair's pin then adopts them without a second rule implementation.
