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
