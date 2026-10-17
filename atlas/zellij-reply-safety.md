# Zellij reply admission

Zellij 0.45.1 can reuse a numeric client ID before all work for the previous
connection finishes. Pair treats a CLI reply as untrusted evidence. The #647
implementation proceeds through lifecycle admission, Go actions and editor
routing; the table records the current implementation boundary.

`cmd/internal/zellijreply` is the pure strict contract layer. It validates
clients, panes and tab information without changing the tolerant
`zellijpane.Parse` used by advisory consumers. `Confirm` validates the first
observation before starting a second independent invocation, compares the
caller's decision projection and returns the admitted second snapshot. Unknown
first evidence terminates the request; there is no observation retry.

A five-second admission context is shared by Couch admission and owner-proof
observations, capped by earlier caller deadlines. Launcher selected-operation
context plumbing remains an open M1 correction: its context-free runtime can
currently reset the query budget at successive fences. The cold-birth phase
boundary is awaiting the R3 design decision; M1 has not closed. Couch carries query admission separately
from helper lifetime: completing or canceling admission cannot terminate an
established client. Existing one-second restart admission remains the smaller
bound when that caller is migrated.

`cmd/internal/zellijexec` owns synchronous read-only query processes. It creates
a private process group, cancels that group, joins the direct CLI child, bounds
pipe draining and stops owned descendants on every completion path. It is not
used for creating a server, interactive attachment or native mutations.
Descendants deliberately leaving the group are outside that ownership contract.

| Caller | Disposition | Current authority / verification |
|---|---|---|
| Listing poller and event-driven inventory | Re-checked | Existing cadence, cache and query count; malformed expected listing/client evidence is unknown. No projection alone authorizes an effect. |
| Launcher name reuse and cold admission | Hardened in M1 | Strict listing contract plus independent exact-server absence; a live server defeats native listing omission. |
| Launcher name-length probe | Hardened in M1 | Successful replies must be client tables; exact native validator/absence diagnostics retain their limited name-length meaning. |
| Launcher Couch attachment | M1 correction, review pending | Existing positive listing fences stop on first unknown; launcher and exported attach regressions exercise actual strict native parsing before attach/launch effects. Shared operation budget is pending R3. |
| Session owner proof, quit and recovery | Hardened in M1 | Matching actual-command/pane-identity projections, exact server generation bracketing the pair and fresh generation revalidation before effects. |
| Launcher live-layout inference | Hardened in M1 | Strict pane fields and matching command/title/identity/floating facts before recording a detected layout. |
| Ordinary Couch warm attachment | Hardened in M1 | Existing initial, pre-spawn, pre-ack and post-ack listing fences; no additional queries or detached-client requirement. Unknown/negative confirmation cannot trigger helper retry. |
| Explicit continuation adoption | Hardened in M1 | Existing repeated exact address/name/agent detached proof under one admission budget; no nested confirmation pairs. |
| Wait-watcher, PID/start liveness and quiescence | Re-checked | Existing OS identity and absence fences; native unknown cannot prove absence. |
| Layout focus/toggle/split/tab forwarding and terminal shortcut fallback | M2 pending | Strict target/state admission and existing validated local sidecar fast paths. |
| Current-tab layout capture/restoration | M2 pending | Correlated tab/layout facts and partial-progress refusal. |
| Go draft routing, restart role and note anchors | M2 pending | Explicit admitted targets; existing Couch notices. |
| Clipboard auto-quote / paste / query-selected flash | M2 pending, fail-closed | Successful OS copy is separate; uncertain routing must refuse before staging, flash, focus or trigger. |
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
