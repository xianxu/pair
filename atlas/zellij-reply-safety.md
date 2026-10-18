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
first evidence terminates the request; there is no observation retry.

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
| Launcher Couch attachment | Hardened in M1 | Existing positive listing fences stop on first unknown; launcher and exported attach regressions exercise actual strict native parsing before attach/launch effects. The selected operation shares one query budget through handoff, independently of the interactive client lifetime. |
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
| `launcher.ZellijSource` / `ListingPoller` | short/full `list-sessions`, per-session `list-clients` | Re-checked inventory; strict rows/headers and exact listing-set agreement. Existing cadence/cache; unknown never proves absence. `zellij_reply_admission_test.go`, `zellij_test.go`. |
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
