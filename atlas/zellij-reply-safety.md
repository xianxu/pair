# Zellij reply admission

Zellij 0.45.1 can reuse a numeric client ID before all work for the previous
connection finishes. Pair treats a CLI reply as untrusted evidence. The #647
implementation proceeds through lifecycle admission, Go actions and editor
routing; the table records the current implementation boundary.

`cmd/internal/zellijreply` owns strict contracts and admission. Its pure decoders validate
clients, panes and tab information without changing the tolerant
`zellijpane.Parse` used by advisory consumers. `Confirm` validates the first
observation before starting a second independent invocation, compares the
caller's decision projection and returns the admitted second snapshot. Unknown
first evidence terminates the request after the narrow read-boundary compatibility policy below; confirmation disagreement never retries the decision.

A five-second admission context is shared by the observations of one decision,
capped by earlier caller deadlines. A selected launcher warm attachment carries
the same context through listing fences and handoff Check. Query admission is
separate from the interactive client/helper lifetime: completing or canceling
admission cannot terminate an established client. Cold creation has three phases:
one pre-create query budget, the existing independent birth/readiness bound, and
a fresh post-birth query budget. A slow healthy birth does not consume the latter.
M1 closed SHIP. Existing one-second restart admission remains the smaller bound
and stays alive through query-derived routing or confirmation dispatch.

`zellijreply.Effect` checks the same operation context immediately before each
new effect, including native mutations and authoritative sidecar writes after
positive local evidence. It wraps one effect at a time; cancellation inside an
issued effect stops the next one without rolling back completed progress or
changing the issued mutation's lifetime. The sink-manifest test covers every
protected dispatch and explicitly identifies cleanup and diagnostic exemptions.
Terminal notes use Couch-owned synchronous frame/layout data only; their former
native query and asynchronous enrichment have been removed by operator decision.

`cmd/internal/zellijexec` owns synchronous read-only query processes. It creates
a private process group, cancels that group, joins the direct CLI child, bounds
pipe draining and stops owned descendants on every completion path. It is not
used for creating a server, interactive attachment or native mutations.
Descendants deliberately leaving the group are outside that ownership contract.
Couch's remaining restart-client query entrypoint uses this owner; native note
geometry lookup has been removed.

| Caller | Disposition | Current authority / verification |
|---|---|---|
| Listing poller and event-driven inventory | Re-checked | Existing cadence, cache and query count; malformed expected listing/client evidence is unknown. No projection alone authorizes an effect. |
| Launcher name reuse and cold admission | Hardened in M1 | Strict listing contract plus independent exact-server absence; a live server defeats native listing omission. |
| Launcher name-length probe | Hardened in M1 | Successful replies must be client tables; exact native validator/absence diagnostics retain their limited name-length meaning. |
| Launcher Couch attachment | Hardened in M1 | Existing positive listing fences stop on first unknown; a listing that omits the session while its exact server is live (OS process scan) refuses as `ErrListingOmittedLiveServer`, unknown rather than "not listed live" (#692); launcher and exported attach regressions exercise actual strict native parsing before attach/launch effects. The selected operation shares one query budget through handoff, independently of the interactive client lifetime. |
| Session owner proof, quit and recovery | Hardened in M1 | Matching actual-command/pane-identity projections, exact server generation bracketing the pair and fresh generation revalidation before effects. |
| Launcher live-layout inference | Hardened in M1 | Strict pane fields and matching command/title/identity/floating facts before recording a detected layout. |
| Ordinary Couch warm attachment | Hardened in M1 | Existing initial, pre-spawn, pre-ack and post-ack listing fences; no additional queries or detached-client requirement. Unknown/negative confirmation cannot trigger helper retry. |
| Explicit continuation adoption | Hardened in M1 | Existing repeated exact address/name/agent detached proof under one admission budget; no nested confirmation pairs. |
| Wait-watcher, PID/start liveness and quiescence | Re-checked | Existing OS identity and absence fences; native unknown cannot prove absence. |
| Layout focus/toggle/split/tab forwarding and terminal shortcut fallback | M2 SHIP | Strict target/state admission and existing validated local sidecar fast paths. |
| Current-tab layout capture/restoration | M2 SHIP | Correlated tab/layout facts and partial-progress refusal. |
| Go draft routing and restart role | M2 SHIP | Explicit admitted targets; cancellation checked at actual dispatch; existing Couch notices. |
| Terminal note anchors | Deterministic Couch capture; native consumer removed in M2 | Slot, box, draw-time text and Couch-owned tab/status anchors; existing PNG capture unaffected. Zero native queries or file/line/section guesses. |
| Clipboard auto-quote / paste / query-selected flash | M2 SHIP, fail-closed | Successful OS copy is separate; uncertain routing must refuse before staging, flash, focus or trigger. |
| Lua workbench routing, poke, DOM and review visibility | M3 implemented, qualification in progress | Shared strict contracts, contextual targets, retained recipient/activation fences and visible refusal. |
| Title poller, wrap dim and opener viewport | Re-checked | Existing next tick/invocation and cosmetic fallback; no action authority. |
| Peer viewport advisory check | Re-checked | Existing 500ms freshness/advisory policy; delivery retains independent recipient/composer authority. |

Native pane identity is `(is_plugin,id)`; plugin 0 and terminal 0 may coexist.
Native shell templates may be null, and command enrichment can be absent.
Missing evidence never becomes a default false/zero value for a protected
selection. Full session listings validate native creation metadata and suffixes,
including foreign names with spaces. Short/full pairs must agree on their exact
session-name sets before classification or client queries, including cached pairs.
Full-row consumers retain exact identities and derive EXITED only from its
validated suffix, rather than name text. Short names are inherently ambiguous;
recognizable JSON/table/diagnostic replies refuse even if someone could use
that text as a native session name.

The stateful external double is `tests/lib/fake-zellij.py`, owned by the
`cmd/internal/zellijtest` adapter. Normal replies derive from sessions, panes,
clients, tab state and focus; ordinal faults change observations/state, and
mutations record exact targets. Private roots and no native processes isolate
these deterministic tests. Go/Lua share `tests/fixtures/zellij-replies.json`.

The protocol exposes no request nonce. Two identical valid stale replies remain
indistinguishable; comparison does not authenticate their authorship. A quiet
native run cannot prove universal safety, and the synthetic misrouting sample
is not a production user-action failure probability. #640 contains the native
diagnosis; #647 owns implementation and native refusal qualification. After a
release includes upstream PR5630, rerun the retained raw reproduction and
healthy/refusal/chord controls before changing compatibility assumptions.

M2 layout decisions use the independently known invoking terminal to select a
single tab; multiple focused panes on other tabs do not select the target.
Normal entry confirms geometry before saving split order. Returning to Normal
confirms identity, role, state and tab without comparing irrelevant pre-embed
geometry; later restoration confirms the layout and any geometry used to move a
split half. A refusal after embedding retains that completed progress and the
return record, and stops before return-focus or record-clear. Supported tab
mutations carry the admitted native tab ID. Query reads remain sequential.

Disposable installed-0.45.1 timing on 2026-10-17 measured median return-to-Normal
294 ms and Focus entry 130 ms after narrowing, versus original-binary medians
192/91 ms in the same declared window (five samples each). An initial hardened
attempt measured 353/171 ms. These host-load-sensitive samples do not isolate a
causal optimization effect. Confirmation's observed interactive overhead is
accepted by the TL; no deadline increased. Native timing controls retain all
layout/focus/geometry assertions and do not qualify fault refusal by themselves.

A matched disposable native clipboard pair measured auto-quote 117 ms after
hardening versus 103 ms for the original helpers executed in process; standalone
paste measured 64 ms versus 84 ms. The original control excludes subprocess/exec
overhead, and each number is one host-sensitive sample. Native observations
changed from two to four on auto-quote and one to two on standalone paste. Both
controls retained actual receiver-byte, private staging and client-focus checks;
clipboard input was private and cosmetic reset scheduling was recorded. These
controls bound healthy interactive cost without claiming a causal speedup,
OS clipboard copying or native fault qualification.

An attempted flash creates an owed reset-scheduling attempt even if applying
the color fails or admission is canceled. That cleanup uses the existing
detached lifetime; cancellation stops subsequent staging, focus or input.
Apply and reset-start failures reach existing refusal diagnostics. Successful
reset scheduling proves child start, not eventual native reset success.

Workbench pane confirmation preserves native row order and compares an explicit
projection of identity, plugin namespace, tab, required state/geometry, launch
template and derived role. Titles matter through role inference only; foreground
command and cwd are cosmetic. Consumers pass their existing registered-terminal
snapshot, so a registered terminal changing its display title keeps its role
without an extra registry read. Title-only role changes still refuse.

## Editor decisions

`nvim/zellij_observation.lua` contains pure strict decoders and the thin budget,
query, confirmation and effect glue. Go and Lua run the same reply corpus.
Lua rejects identities it cannot represent exactly. The first unknown reply or
mismatch ends admission. Query spawn errors return refusal; an exception after
an effect starts retains uncertainty about completed progress.

Workbench/poke fallback confirms a unique role in the independently known
invoking terminal's tab. `pane_identity.select_in_tab` and `context` own that
selection/projection. Valid draft session/live-PID cache remains a zero-query
fast path. Every focus or input uses the same admission token; no mutation is
replayed after failure. Titles, foreground command and cwd do not select roles.

DOM pinning is candidate evidence under the existing transaction lifetime.
First pre-write validation starts one five-second admission budget; recipient
`begin` and strict source/agent checks precede each write/submit. The target's
required effect guard reaches the actual dispatch, including cancellation
inside validation. A terminal refusal cannot become authority on a later call.
Refused submit retains `written`; an uncertain dispatched write remains
`indeterminate`. Preparation and receipt recovery keep their existing bounds.

Review restoration completes birth/readiness/activation first. Visibility then
gets its own five-second decision: two matching pane contexts and, for a toggle,
two matching exact visibility results. Token/session/endpoint/activation checks
run before observations and effects. Show/hide explicitly target the admitted
tab; hide-and-return shares that token and draft identity without a nested
confirmation. Visibility refusal keeps the completed activation and ready target.

## Production query ownership

The table covers executable query sites, grouped where one runtime owns several
callers. Comments and test-only probes are outside the production inventory.
Re-checked means another observation may repair display/advisory state; it does
not authorize an irreversible action or authenticate a same-kind reply.

| Runtime / caller | Queries | Policy and evidence |
|---|---|---|
| `launcher.ZellijSource` / `ListingPoller` | one full `list-sessions`, per-session `list-clients` | Re-checked inventory; strict rows/headers, with exact names and state derived from one complete reply (#711). Existing cadence/cache; unknown never proves absence. `zellij_reply_admission_test.go`, `zellij_test.go`. |
| `OSRuntime.SessionBlocksReuse`, `ProbeSessionNameContext` | full listing; targeted clients | Fail closed against live/unknown independent server identity; exact name-validator diagnostic matrix. `zellij_diagnostic_matrix_test.go`, `query_collection_diagnostic_test.go`. |
| `OSRuntime.ProbeLiveLayout` | targeted command/state panes | Hardened matching layout projection before recording; `zellij_reply_admission_test.go`. |
| `SessionOwnerProbe`, retirement/quit/recovery | targeted command panes plus full listing/server checks | Hardened owner projection and exact process generation fences; `session_owner_test.go`, `recovery_terminal_confirmed_test.go`. |
| `OSRuntime.ListSessions` / `ShowFamilyExisting` | listings and clients | Inventory/display only; downstream action independently admits. Unknown inventory refuses mutation. |
| session handoff / warm Couch attach / continuation adoption | repeated listings and owner/client observations through existing runtimes | Hardened existing fences; zero additional ordinary warm-attach queries, no nested pair or helper retry after unknown. `session_handoff_budget_test.go`, `launch_query_budget_test.go`, `couch_session_test.go`. |
| quiescence / presence / wait-watcher | listing plus independent OS/socket/PID/start evidence | Re-checked; unknown cannot authorize deletion, replacement or absence. `session_quiescence_test.go`, `session_servers_test.go`. |
| `layoutcmd.OSRuntime` via layout/terminal/draft actions | panes; `current-tab-info` | Hardened decision-specific pane/tab projections, local positive sidecar fast paths and per-effect cancellation. `reply_admission_test.go`, `decision_projection_test.go`, `effect_cancellation_test.go`. |
| `clipcmd.OSRuntime` | panes and attached clients | Hardened source/destination correlation before quote staging/flash/focus/input; successful OS copy remains separate. `cli_admission_test.go`, `admission_test.go`, `flash_cleanup_test.go`. |
| `couchcmd.queryShortcutClients` | exact session clients | Hardened restart role under one-second token; shared owned collector. `couch_reply_admission_test.go`, `owned_query_test.go`. |
| `workbench_route`, `pair_poke` | contextual command/state/tab panes | Hardened sequential confirmation; actual CLI faults/bytes/targets in `nvim/route_poke_admission_test.lua` and existing route/poke scripts. |
| `dom_transport` / `draft_send` | candidate, pre-write and pre-submit contextual panes | Hardened existing fences plus recipient checks, required final effect guard. `dom_admission_test.lua`, `dom_admission_cli_test.lua`, `draft_send_test.lua`. |
| `review.restore_client` | contextual panes and tab-addressed visibility | Hardened after activation; exact true/false diagnostics, local activation fence, shared hide token. `restore_client_test.lua`, `tests/review-restore-admission-test.lua`. |
| `titlepoller.SessionAlive`, wrap dim, opener viewport | full listing / screen dump / agent pane lookup | Re-checked cosmetic/advisory consumers; title debounce, next paint/invocation or viewport render can repair a bad display. They supply no launch/ownership/routing authority. |
| `wrapcmd.composerInView` | exact pane `dump-screen` | Existing 500ms advisory freshness; delivery still owns independent recipient/composer authority. `peer_input_test.go` and peer-delivery integration controls. |
| Terminal note capture | none | Native query removed. Deterministic frame/owned-layout data only; zero-query broadcast composition tests. |

## Native sample and mid-toggle refusal

The single granted 2026-10-18 installed-0.45.1 sample observed 31 misrouted first
frames in 830 raw requests: **3.7%** (31/830), all wrong response kinds with no
transport error. Breakdown: 24 exit frames, six Connected responses and one
ListPanes output delivered to the other request kind. This is a measured
synthetic raw-request misroute rate, not a production user-action probability or
proof of independent faults. Retained evidence: /tmp/p647nr-3940343646/result.json
and /tmp/pair647-m3-native-attempt.log. No upstream publication or native rebuild.

The first production toggle succeeded with a PTY marker delivered only to the
expected terminal. During the second, embedding and layout steps completed;
then current-tab-info returned exit1, “There is no active session!” Pair displayed
its fixed refusal, kept the return record and made no mutation after unknown.
The final snapshot showed original tiling and agent focus. That is a permitted
completed-prefix outcome: embedding can change native focus before the explicit
return-focus step, which refusal prevents. There is no rollback or automatic
continuation, and no guarantee that every refusal ends in the same focus state.
The next shortcut observes current state; from Normal it starts Focus anew.

The fixture initially expected the previous Focus mode after this partial
restore and stopped before its final PTY witness. Preserve that attempt as
**stopped/inconclusive for clean-path qualification**, not a clean test pass.
The hub accepted its retained refusal/record/effect-order trace as safety
evidence. Offline oracle corrections model completed prefixes; they cannot
retroactively supply the missing final witness. Clean-path qualification is
assigned to the real-setup check at the next restart; no further native run is
authorized for this sample.

## Native fixture contracts

Private CLI fakes serialize query replies through `tests/lib/zellij_fixture_replies.py`.
Semantic fixture models retain their IDs, roles, geometry and state transitions;
one renderer owns native field names, complete positive-state defaults and exact
stdout/stderr/status envelopes. An observed zero-client table and a missing session
remain distinct. Malformed, partial and collector-failure controls use explicitly
classified fault replies; they are not filled with positive defaults.

The source-specific `tests/lib/zellij_fixture_*_cases.json` catalogs cover named
positive states and negative reply variants. Go fixtures pre-render static replies
through `cmd/internal/zellijtest`, so timed native queries gain no generator process.
The stateful CLI uses the same serializer for dynamic model observations.
`TestEveryZellijFixtureReplyUsesProductionDecoder` enumerates the catalog through
production strict pane/client/tab and launcher session/diagnostic boundaries; it
also discovers unregistered subprocess fake producers and rejects independent
positive serializers. Pure parser inputs and real native passthrough diagnostics
remain separate, explicitly classified test surfaces. The guard is verified by
malformed-positive and unregistered-producer mutation controls.


### Recovery fixture diagnostics (#636)

The warm-recovery and park conformance fixtures explicitly select disposable
HOME/config/data/cache/runtime/socket roots through the existing empty-environment
test boundary. Their nested park helper receives the same selected roots;
outer shell overrides alone are not fixture authority. A poisoned-ambient child
regression and the Pair-local repository lint rule defend that propagation
(see [Go lint catalog](go-lint-catalog.md)).

Recovery admission is tested through the production scoped query adapter and
shared stateful native fake, including exhaustion at both ownership reads and
the detached-client read. Once bounded read recovery is exhausted, a later
healthy response cannot rescue that operation; its durable record and
launch/quiescence effects remain unchanged on refusal.

Live fixture diagnostics reuse the existing context-scoped launcher query
observer. It retains bounded copies of the original query bytes, process status
and timing alongside local lifecycle markers. Query events carry the actual
caller deadline and effective deadline after the existing query cap; shorter
child contexts remain visible. Capture writes the test log after the
measured path and cleanup. Capture is limited to64 records and8MiB of response
bytes with explicit dropped counts. It adds no native queries, retries or timing
policy. Local client wait remains distinct from server detach acknowledgement;
the trace records evidence and does not prove the upstream trigger fixed.

The park conformance fixture waits for OS evidence of the helper's Zellij client
(a live `zellij` child in the process table) before its first client-count query, so
it never connects to the server while the helper's single list-sessions fence runs.
Polling across that fence let the upstream reply race drop the live session from the
helper's listing (#692).

Name validation also recognizes Zellij0.45.1's multi-session missing-target reply:
exit0, the exact requested-name not-found stderr heading, and a complete colored
session listing that excludes that name (#672). Only native SGR metadata is
removed; the existing strict full-listing decoder validates every row. This
proves the CLI accepted the name, never session liveness or ownership. Clients
tables remain strict. Malformed, contradictory or failed/cancelled collections
remain unknown. Native list_sessions exits before its caller's intended exit1.

### Bounded native read compatibility (#748)

`launcher.ZellijSource.runContext` owns a temporary workaround for0.45.1's
client-ID reuse bug (upstream PR5630). Exact allowlisted full unformatted
listings, list-clients and JSON list-panes command/state reads can make at most
three total sequential attempts under their original shared5s/caller deadline.
Only exit-zero empty/whitespace output or a complete validated different reply
kind is reissued. Every attempt retains its own query-observer event. There is
no delay/backoff or mutation replay. Nonzero/collector errors, malformed
same-kind payloads and cancellation stop immediately; caller strict fields,
server-generation checks and independent agreement remain mandatory.

Tests retain the empty/crossed reply schedules, exhaustion without effects,
cancellation, counted8/16 read bounds, and native qualification artifacts in
`workshop/evidence/000748`. The Pair lint catalog rejects new direct calls to
the private one-attempt collector outside that owner. Other query consumers
retain their existing contracts. Remove the workaround after the supported
native release includes PR5630 and the controls are requalified.

### Standalone native lookup reproduction (#775)

`scripts/repro/zellij-false-miss.sh [queries=1000] [workers=4]` is an upstream-shareable
Bash diagnostic requiring only Zellij0.45.1 and stock Unix utilities. It starts
three private background shell sessions under temporary HOME/XDG/config/socket
roots, then issues raw pane, client and session-list queries without measured-read
retries. Its [ten-line guide](../scripts/repro/README.md) describes invocation.

`exit0_target_listed` requires the complete target-not-found stderr response and
an exact target token in the colored native listing. `exit1_alive_confirmed`
requires the complete no-active-session response, a socket before/after the
failure, and a subsequent successful dump-layout response containing a line
matching `^layout[[:space:]]*\{`. Failed queries or responses without that heading
stay `exit1_liveness_unconfirmed`; this recognizes a heading rather than parsing
the complete layout. Other nonzero
results remain separate. These are observations, not continuous liveness proof
or an ownership grant, and zero misses does not rule out the race. Raw anomalous
results print before cleanup. Cleanup addresses only the private namespace;
unconfirmed socket removal retains the named root and exits unsuccessfully.

`python3 tests/zellij-false-miss-test.py` exercises the actual worker with a
stateful CLI double and real private sockets, including whole-stderr byte
matching and confirmed/unconfirmed liveness. Both private executables use the
shared reply serializer and producer catalog in `tests/lib/zellij_fixture_replies.py`;
explicit faults retain malformed bytes and their production-decoder classifications.
The standalone script itself does
not require Python. `make zellij-false-miss` explicitly runs the regressions and
the default native diagnostic; neither is added to `make test`. This manual
diagnostic is separate from Pair's bounded recovery.
