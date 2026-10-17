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
The note resolver retains each slot/pass context through its final publication,
then releases it; a valid late observation cannot outlive admission.

`cmd/internal/zellijexec` owns synchronous read-only query processes. It creates
a private process group, cancels that group, joins the direct CLI child, bounds
pipe draining and stops owned descendants on every completion path. It is not
used for creating a server, interactive attachment or native mutations.
Descendants deliberately leaving the group are outside that ownership contract.
Both Couch restart-client and note-geometry query entrypoints use this owner.

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
| Layout focus/toggle/split/tab forwarding and terminal shortcut fallback | M2 implemented, review pending | Strict target/state admission and existing validated local sidecar fast paths. |
| Current-tab layout capture/restoration | M2 implemented, review pending | Correlated tab/layout facts and partial-progress refusal. |
| Go draft routing and restart role | M2 implemented, review pending | Explicit admitted targets; cancellation checked at actual dispatch; existing Couch notices. |
| Note anchors | M2 REWORK; operator decision pending | Matching geometry and unchanged frame text do not prove which native tab produced a captured frame. Existing enrichment retained pending an explicit UX decision. |
| Clipboard auto-quote / paste / query-selected flash | M2 implemented, fail-closed; review pending | Successful OS copy is separate; uncertain routing must refuse before staging, flash, focus or trigger. |
| Lua workbench routing, poke, DOM and review visibility | M3 pending | Shared contracts, existing recipient fences and user-visible editor refusal. |
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
