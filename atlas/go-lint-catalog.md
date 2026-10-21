# Go lint catalog

Pair's ordinary repository test `internal/repolint.TestTrackedGoFilesPassCatalog`
imports `github.com/xianxu/ariadne/pkg/golint` through the pinned Go module in
`go.mod`. Shared rules are maintained upstream, not copied by weave; update the
module pin to adopt later shared rules. `internal/repolint/local_catalog_test.go`
copies that catalog slice into `pairCatalog` and appends Pair-specific ownership
rules. The repository scanner passes the composed catalog to the shared checker.
The initial adoption also advances the runtime
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

## Wrapper child ownership (#715)

The local single-file rule `wrapper-child-release`, owned by
`internal/repolint/local_catalog_test.go`, rejects calls through a `Process.Release`
selector in package `wrapcmd`, including chained and parenthesized receivers.
Wrapper children must retain direct ownership and be joined before wrapper
return; a released watcher can otherwise write retention state after its caller
has begun removing storage. Other packages and unrelated `Release` methods are
outside this syntax rule's scope.

This rule uses the shared `golint.Check` finding and suppression behavior above.
`pairCatalog` copies the slice before appending, so it never mutates upstream
`golint.Catalog`; `sleepSites` still uses the upstream catalog alone to identify
the exact existing sleep debt. `TestWrapperChildReleaseRule` exercises both
positive and negative examples through the actual repository scanner.

## Diagnostic address composition (#714)

The local single-file rule `lifecycle-failure-address`, owned by
`internal/repolint/lifecycle_failure_address_rule_test.go`, protects the three
diagnostic composition owners in `couchcore`: `lifecycleFailureTargetForCall`,
`OperationAttempt.ObserveCall` and `lifecycleFailureForAttempt`. Address writes
there use `completeFailureAddress`; whole-address overwrites, direct zero erasure
and fieldwise scope/tag changes can discard or mix the exact launch identity.
The rule uses the same composed catalog, shared checker and reasoned suppression
contract as `wrapper-child-release`. Ordinary validated record copies elsewhere
stay outside its scope; the upstream catalog and sleep matcher are unchanged.

## Pair native fixture roots (#666)

The same repository lint test also runs the Pair-local cross-file rule
`native-fixture-roots`, owned by `internal/repolint/native_fixture_roots_test.go`.
`readNativeFixtureRoots` reads the three named couchcore source files once each;
its pure `nativeFixtureRootFindings` consumes those bytes and returns located
findings. Missing, unreadable or malformed required sources fail the check.
This adds no Git or native subprocess query and no suppression mechanism.

The explicit manifest protects recovery and park setup, the park helper's
`cmd.Env` handoff, and all seven root reads in the child environment builder.
It checks the named syntax contract, rather than proving arbitrary data flow or
automatically discovering future native fixtures. Extend the manifest when
adding another protected entrypoint. The shared single-file catalog remains
independent of Pair helper names.

The six original #636 source mutations now run at this lint boundary. Synthetic
source tests also omit each root and source, plus a missing warm entrypoint,
and exercise malformed sources and a bodyless warm declaration. The actual poisoned parent/nested child environment,
cleanup/restoration and Unix socket-budget regression stays in couchcore; it
proves runtime propagation, complementing the source rule. The original duplicate
couchcore matcher was retired after both boundaries passed equivalent mutations.

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

Socket-root exceptions budget the full address each consumer derives. The
couchmessage transport fixture keeps an owned short root because cleanup
consumers replace its RPC leaf with production wrapper/PID/hash names; a
long-TMPDIR regression binds those actual derived endpoints.

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

Shared mechanical mistake classes belong in ariadne's catalog with the incident
and direct matcher tests. Propose cross-repository rules to the hub; changing
Pair's pin then adopts them without a second rule implementation. Pair-specific
package ownership policies belong in the local composed catalog, with the same
incident and matcher tests plus this atlas mapping; they reuse the shared checker
and leave the upstream catalog and sleep-debt matching unchanged.

The Pair-local `terminal-registry-owner` rule (pair#731) rejects direct kill-only `LiveIDs` calls in terminal, layout, clipboard and Couch production functions. Native routing uses `LiveOwnerIDs`; explicit registry storage tests and the shared adapter retain the generic predicate seam.

## Bounded Zellij read ownership (#748)

The local `zellij-read-attempt-bypass` rule rejects direct `runQueryAttempt`
calls in package launcher outside `ZellijSource.runContext`. Callers use the
bounded entry to share its deadline, exact read allowlist and attempt evidence.
It checks call-site syntax, not arbitrary function-value alias dataflow.

## Controlled Zellij cleanup (#764)

The local `controlled-zellij-close` rule is owned by
`internal/repolint/controlled_zellij_close_rule_test.go` and registered in
`pairCatalog`. It rejects discarded `ControlledZellij.Close` results (blank,
bare, deferred or goroutine calls) and checked errors that only emit diagnostics.
Cleanup must fail the test through t/tb Error, Fatal or Fail, or propagate the
error through return or a retained errors.Join result. Nonfatal errors preserve
subsequent exact-server and private-root cleanup checks.

The syntax guard follows the actual fixture import, direct constructors, local
aliases, explicit types and same-file fields, including range elements. It does
not prove arbitrary control flow, reassignments, cross-file declarations, dot
imports, method values or function-return aliases. Unrelated closers stay outside
its scope. Scanner regressions exercise both rejected and accepted patterns.

The fixture itself shares one handle-retirement path between KillClient and
Close: consume the PTY and client before closing/signalling/reaping, and retain
real failures. Startup refusal joins cleanup errors; final cleanup still requires
exact server-generation exit evidence before removing its owned cache root.


## Inherited descriptor proof (#767)

The local `fd-path-permission` rule is registered and implemented in
`internal/repolint/local_catalog_test.go`. It flags literal shell `test`, `[` and
`[[` permission probes (`-r`, `-w`, `-x`) of numeric `/dev/fd/N` paths. The identity
fixture exposed a macOS false negative while its inherited descriptor remained
readable. Check the descriptor itself (read, fstat or fcntl); a pathname's access
permission is a separate observation. The rule is lexical and does not resolve
dynamically assembled shell commands. Its fixture cases live in
`internal/repolint/fd_path_permission_test.go`; finding/suppression behavior is
shared with the catalog above. Other consumers' shared catalog is unchanged.

## Scribe terminal-aware output producer (#766)

The Pair-local `scribe-pty-printf` rule, owned by
`internal/repolint/scribe_pty_printf_rule_test.go`, rejects a bare or absolute
`printf` executable after `--` in inline string arguments to scribecmd's
`runWithPTY` and `runWithPTYReply` test helpers. A short-lived producer can lose
unread output at Darwin's last slave close. The tee fixture instead opens
`/dev/tty`, exchanges a readiness/acknowledgement through the real PTYs, then
emits its final marker. Its native stress regression checks the actual stdout
and log bytes.

This guard checks direct syntax only; argv variables, aliases, computed arguments
and shell protocol correctness are outside its scope. Other commands and helpers
are unaffected. Public scribe's generic-command exposure is tracked separately
in #768; this fixture rule does not claim a production launch-policy fix.

## Manual-clock delivery proof (#771)

The Pair-local `manual-clock-rendezvous` rule is registered in
`internal/repolint/local_catalog_test.go`, with its matcher in
`manual_clock_match_test.go` and scanner cases in `manual_clock_rule_test.go`.
It rejects buffered `chan time.Time` creation in `newManualClock` and the
`manualClock.after` method. An unbuffered send proves the hub selected the timer;
a subsequent request barrier proves its handler finished. A buffered send plus
an unrelated request barrier provides no ordering between the two select cases.

This is a direct syntax guard for the named fixture seam: it accepts omitted
capacity or literal zero and rejects other capacity expressions. It does not
resolve aliases, helpers or computed channel types. Real timers elsewhere are
outside its scope. Findings and suppression use the shared catalog behavior;
other consumers' shared catalog is unchanged.

## Fixture CLI group completion (#744)

The Pair-local `fixture-cli-group-completion` rule is owned by
`internal/repolint/fixture_cli_group_rule_test.go` and registered in pairCatalog.
It rejects Wait calls in agentfixture.runCLI when the owner no longer invokes
cliGroupError. A completed command leader does not prove descendant quiescence.
The accepted owner observes every successfully started completion and retains
explicit uncertainty when the group is live or unknown.

This narrow syntax guard detects deletion of the observation, including ordinary
error paths and misleading comments. It does not prove ordering, control flow,
exec.Cmd aliases or indirect calls; the real completion-path regressions cover
success, signal, cancellation and leader-first descendants with open/closed pipes.
