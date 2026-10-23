# Disposable native Couch fleets

`cmd/internal/testcouch` is the shared test-only owner for the native fleets used
by #722, #740 and #741. The first consumer is
`cmd/couch/fleet_acceptance_test.go`. Scenario policy remains with those issues;
existing narrow unit fixtures remain useful. New multi-slot native acceptance
should use this helper instead of copying supervisor/environment/cleanup code.

Run portable tests with `go test ./cmd/internal/testcouch/...`. The explicit native
qualification is `make test-couch-fleet` (`PAIR_NATIVE_COUCH_FLEET=1`); it requires
Go, Git, installed SDLC/Weave, native Zellij and nvim. Missing tools fail an opt-in
run. Without opt-in, only the native consumer skips. No real model, credentials,
package installation or network is part of this test.

## Ownership and interfaces

`Build(t)` owns the candidate binaries and records their source head and native
tool versions. `New(t, binaries, Options{Workers: 3, AgentKind: "claude"})` creates
a fresh fleet; `Prepare(ctx)` uses actual Git and public adoption preview/apply,
then real `couch --internal provision-workspace` and `sdlc workspace --json`.
No setup marker, Couch binding or workspace identity is fabricated.

`Start(ctx)` launches a real supervisor with one steward. `StartWorker(ctx, n)`
asks that steward to run the public recordless Fresh CLI. `ActorCLI` runs inside
the selected fixture process with its actual inherited actor environment; the
test driver does not replay credentials. `Actors`, `Agent`, `Accepted`, `Screen`,
`Input` and `Resize` observe or drive the owned actors. `Detach` uses the ordinary
switcher detach-and-leave action, and `RestartSupervisor` starts the same private
fleet again. Callers provide bounded contexts; `Close` is also registered as test
cleanup.

Claude and Codex are two personalities of one deterministic executable engine,
installed as physical executable files. Pinned native composer captures exercise
the real wrapper, and submitted input produces parser-compatible native
conversation files under private agent homes. Working/picker/idle controls and
correlated CLI requests use an owned fixture mailbox. Consumed CLI results retire
explicitly; unread requests/results stay capped at 128, with at most four active
CLI children and a separate shutdown slot. Admission is serialized by one owned
mailbox lock, and atomic scratch names do not count as extra requests. Caller
cancellation revokes its request, joins active CLI work and retires the terminal
result; absolute deadlines include queue time. Ownership snapshots publish
atomically, and publication errors remain visible while in-memory cleanup
authority is retained. A Stop result is not a
shutdown proof: the helper waits for the fixture's exact process generation to
end after its CLI workers and cancellable native TTY reader join. These stand-ins
qualify our transport and lifecycle adapter, not a real installed agent's reaction.

## Isolation and teardown

One fresh root lease owns HOME, all XDG roots, Pair/Couch/identity state, native
Zellij sockets/config/data, fixture control and all three private SDLC queue
controls. The environment starts empty. PATH contains only owned candidate files
and explicit tool links, with no installed-agent fallback. Canonical physical
paths are checked against the ambient live-root deny-set before launch. Short
owned roots keep AF_UNIX addresses within the native limit; TestScratch is
preferred when available. Messaging uses the same explicit isolation selection
for its private listener parent.

The pure root, owned-generation, cohort and agent reducers live next to their
tests. Native adapters own processes, filesystem operations and terminal IO.
Cleanup closes the creation gate, joins current requests and the supervisor,
stops and joins fixture command producers, then takes its final process census.
Run markers are positive evidence; indexed ancestry and private TTY cohorts also
cover platform binaries whose environment macOS withholds. Signals and absence
outcomes are fenced to exact observed generations. Reused or unknown identities
never authorize signalling another process.

After all process generations are absent and direct IO is joined, the owner
removes its roots. A cleanup timeout fails and retains its ownership manifest
and root for diagnosis; the qualification stops after a failed subcase. Inspect
and resolve that exact residue before starting another fleet. A parent SIGKILL or
host crash cannot guarantee cleanup. There is no attach-arbitrary-root API or
global sweeper in this helper.

The initial qualification uses one steward plus three genuine workers, serially
for each personality. The helper accepts 1–8 workers, bounded output and at most
four concurrent fixture CLI children. Build/setup, scenario and cleanup budgets
are separate. Preserve the complete native target log with source/tool/isolation
evidence; a portable or skipped test is not native acceptance. The initial
Claude/Codex three-worker matrix passed with exact cleanup and supplies #747's
post-landing launch proof. #744 retains its exact candidate/log hash; #761 tracks
an earlier native detach ownership refusal separately. Fixture paint also passes
the real wrapper's composer/text classification across personalities, phases,
draft shapes and geometries. These results qualify deterministic fixtures, not
an installed model service.

Every fixture Exec checks group quiescence after its direct child and output join,
including normal exit, signal and cancellation. A live or unreadable group is
explicit cleanup uncertainty: command, result and cancellation evidence stay in
the same bounded mailbox slot and all new fleet mutations refuse. Existing Close
reconciles private-run process generations; only proved cleanup and root removal
clear the failure. A reaped leader never grants numeric process-group kill authority.

Rejected terminal attachment freezes the preflight or precommit failure site,
actual endpoint ID and input-ended state, an observed exit status when already
reaped, and the last 16 KiB of raw output before rollback. The operation recorder
keeps its original attempt identity; these observations grant no cleanup authority.
Fleet failure logs print unknown before/after exits as `unobserved`. Endpoint
termination cause is currently unavailable; an ended endpoint does not prove EOF
or a particular child exit reason (#772).

Failed fleets also snapshot startup evidence after active calls join and before
terminal teardown. The private `PAIR_WRAP_LOG` uses the existing managed bounded
diagnostic writer. Only `STARTUP-identity`, `STARTUP-error` and
`AGENT-READY-write-fail` records enter the retained projection; other debug rows,
argv, prompts and unrelated environment values are excluded. Effective identity
comes from the wrapper's emitted scope/tag/session/nonce and actual PID, with
missing or truncated values explicit. The collector never reconstructs identity
from slot numbers or failure text. A mismatched recorded launch nonce prevents
attribution and pane querying; an unknown nonce labels only the current observed
identity. The final wrapper projection runs again after writers join to capture
buffered terminal records, without another native query.

Failed fleets additionally retain two escaped private Zellij server-log tails:
one before cancellation or teardown effects, one after native producers and IO
join. Each snapshot retains at most 8192 bytes, with same-descriptor size/offset,
truncation and read errors. One-shot phase budgets include failed reads and
repeated cleanup. The path derives from this Fleet's private TMPDIR; ambient
logs, final symlinks, nonregular files and root escapes are refused. The
`RetainStartupServerLog` test option also captures successful diagnostic
controls; other successful fleets leave it off. Shared-log lines need matching
session/launch/generation chronology before attribution. Truncated or unavailable
evidence never proves absence or grants cleanup authority (#806).

At most four exact failure or recently emitted private thread identities are
selected. Each scope census reads at most 129 directory entries, reports its
128-entry scan limit and retains at most 32 readiness sibling names. The exact
expected readiness path uses `artifactpath.Paths.AgentReadyChecked`; present
records decode through the production readiness schema and project only identity
fields, including a comparison with the emitted attempt. Private pane sidecars
select the native `dump-screen` target. Queries use the owned socket environment
and existing `zellijexec` owner and admission bound. All pane queries share one
sub-deadline of at most one quarter of Close's remaining budget, preserving its
original context for actual cleanup. Unavailable panes (including ones already
removed by rollback), read errors and query errors remain explicit. File reads
are confined through an opened private root, reject final symlinks and bound the
wrapper tail to 64 KiB and each pane/native stream to 16 KiB. Pre-exec layout-shell
stderr is not recorded by this collector.

## Fixture identity authorities (#778)

Private socket roots do not isolate a host-wide native server-name census. The
installed-command fixture and `testcouch.allocateRoots` initialize each fresh
identity authority through `testpair.SeedCouchIdentity`, then use the normal
allocator for its one store and its conversation/terminal counters. The helper
exclusively creates the bounded private registry; an existing authority refuses.
Its high-range first C combines the live parent PID with a bounded process-local
sequence, so concurrent fixture processes and fixtures within one process have
distinct names independently of temporary-directory parents. The seed record is
removed by the existing root owner. It is not a permanent identity scheme across
PID reuse: crashed native residue still meets the unchanged fail-closed census.
No production launch budget, retry or ownership rule changes.

## Delayed peer refusal evidence (#791)

`wrapcmd`'s `TestPeerWaitCapture*` tests force the existing pre-paste refusal
through `dispatchPeer` with controlled time and the stateful peer/terminal
fixtures. They pin no capture before 15 seconds, one `PEER-wait` record per active
delivery head at the threshold, exact refusal/cursor/style/frame evidence,
recovery without altered admission, and both input/mailbox locks released before
writer invocation. Repeated polls, a failed sink or disabled diagnostics cannot
change the safe delivery decision. `FuzzEncodePeerWaitCapture` checks arbitrary
styled cells for valid JSON, explicit screen truncation and the complete-record
64 KiB cap.

Records stay in the existing per-slot wrap-events JSONL selected by
`artifactpath.Paths.WrapEvents()`, under the proxy's existing `diagnosticlog`
buffered writer and retention/GC. `PAIR_WRAP_EVENTS=0` disables this diagnostic
stream; disabled/unavailable/failed diagnostics leave delivery safety unchanged.
Captured screens are private local evidence, with no automatic public upload.
`TestPeerVisibleTabWrapNativeResize` uses the status-bar planner and sanitized
native Claude2.1.304 bytes through one/two/one chrome rows, including ordinary
and batch admission. `TestPeerResizeThenSilencePreservesRefusal` deliberately
withholds output after a real resize: it pins cursor-hidden refusal through
300 simulated seconds, one capture with resize timing, then safe admission after
the retained native repaint. This demonstrates a mechanism; the retained native
event log has no resize timestamps and does not prove historical causation.
`TestTerminalResizeEvidenceGeneratedTransactions` and
`TestTerminalDiagnosticSnapshotConcurrentResize` qualify transaction timing and
coherent screen/resize copies. Recognition remains unchanged; #810 owns the
approved follow-up plan for redraw and re-observation.

## External conformance bridge (#801)

`make couch-conformance-bridge` atomically builds `bin/couch-conformance-bridge`.
An external consumer launches its absolute path with `--scratch-parent ABS` and,
optionally, `--couch ABS`. Use an existing short physical parent such as
`/private/tmp` directly; do not add a private wrapper directory. The producer
allocates fresh0700 `pc-*` and `couch-candidate-*` children and removes only
those exact owned children. Neither caller nor producer removes the supplied
parent. Overlong native socket placements refuse without relocation.
The executable finds its checkout from its physical
location, uses the shared `BuildOwned` and `NewOwned` implementations, and starts
one genuine private steward. Its provisioned worker is not started. Native tools,
including nvim, are selected once from invocation PATH; runtime PATH and all
HOME/XDG/socket identities are private. Candidate preparation is offline, with
explicit shared Go caches and no ambient Pair candidate or agent configuration.

The protocol requires pipe stdin, stdout and stderr. It emits one Version1 JSON line
(up to64KiB) after genuine registration and a public editor-endpoint lookup.
Namespace and Socket identify that private service. Environment is a complete
replacement client environment. Documents contains two real, distinct registry
IDs, ordered as parent `artifacts/first.md` and side document
`artifacts/first.side/second.md` (`type: side-doc`, `parent: ../first.md`).
Consumers use these identities rather than creating another owner or launcher.

Startup and lifetime have absolute300s and600s limits from entry. EOF, TERM and
INT converge on one15s joined cleanup through the existing fleet owner; stdin
accepts no commands. Close native editors before EOF. Exit0 certifies owner
cleanup; failure retains its exact evidence root when cleanup is uncertain.
Readiness or cleanup is never automatically retried. SIGKILL cannot promise
cleanup; the consumer must retain the root and report uncertainty rather than
invent a descendant teardown. Diagnostic content is capped at64KiB plus one
bounded final truncation/root marker. An unconsumed readiness pipe is cancellable. Diagnostic writes are nonblocking;
a full or closed stderr pipe fails the invocation after joined cleanup rather than
stalling the owner. A full pipe cannot carry the final root marker.

The portable stateful tests exercise stops during each boundary, failed/late
readiness, malformed authority and pipe cancellation. The opt-in native bridge
check uses this same owner and actual service. Portable tests and a socket stat
alone do not qualify native producer readiness.
