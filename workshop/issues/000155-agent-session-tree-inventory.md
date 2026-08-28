---
id: 000155
status: open
deps: []
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
---

# deterministic agent session-tree inventory

## Problem

Pair's session probing is optimized to discover one native root session for one
new launch. It walks the current agent process tree and returns the first
authorized open transcript; fallbacks select a newly created file or the newest
birth-time candidate. After one ID is written to the tag config and ledger, the
watcher exits. Codex and Muse subagent transcripts are deliberately rejected or
ignored.

That is adequate for a happy-path resume token, but it cannot reconstruct the
world after an unclean shutdown. It does not model the agent's complete session
directory, preserve root-to-subagent relationships, explain competing
candidates, or reliably answer which native session tree belongs to which Pair
tag and in what order. Different Pair consumers consequently grow partial
point-lookups and recency heuristics around the same incomplete evidence.

## Spec

**Model native sessions as forests, not a flat list of resumable roots.** For
every supported Pair agent, one deterministic read-only inventory enumerates
all recognized native session nodes under that agent's storage roots. A node
records the native session ID, agent, transcript/artifact paths, root versus
subagent role, native parent ID when available, and stable native timestamps.
Every recognized subagent remains in the model even when only its root is
directly resumable. Missing parents, malformed metadata, unreadable paths, and
schema near-misses are explicit diagnostics rather than silently dropped rows.

Agent-specific scanners translate Claude, Codex, Agy, and Muse storage into one
shared forest model. Tree construction and ordering are pure over scanner
facts. Sibling and root order is deterministic: native chronology first, then
native ID and canonical path as total tie-breakers. Filesystem traversal order,
glob order, PID enumeration order, and `lsof` output order never affect the
result (ARCH-PURE, ARCH-PURPOSE).

**Tag correlation is a separate evidence-ranked pass.** It binds a Pair tag to
a native root only from authoritative evidence, in this order:

1. the exact Pair config or append-only ledger session ID for `{scope, tag,
   agent}`;
2. an identity-authorized live Pair PID tree holding that native root artifact
   open;
3. native parent relationships, which attach descendants to an already bound
   root.

Timestamps order otherwise plausible candidates for inspection but never
authorize a binding. Conflicting authoritative evidence and equally supported
candidates remain explicitly ambiguous and unbound. The result includes every
candidate and the evidence for and against its correlation so operators and
agents can explain the decision instead of inheriting a silent newest-file
guess.

Expose the inventory through Pair's single public binary with structured JSON
for agents and a stable human rendering for diagnosis. The structured contract
contains forests, correlations, ambiguities, and scan diagnostics; it does not
read Couch persistence. Existing session-watch, transcript lookup, recovery,
and context consumers derive their selection from this shared model rather
than maintaining independent first/newest algorithms (ARCH-DRY).

Tests use portable directory fixtures for every supported agent, including
multiple roots, nested subagents, overlapping timestamps, malformed nodes,
unreadable entries, stale Pair bindings, concurrent live evidence, and
conflicts. A live conformance probe captures each installed agent's current
directory shape without requiring an LLM response and compares it with the
scanner contract (ARCH-MOCK).

## Done when

- One command inventories complete root/subagent session forests for every
  supported Pair agent in stable human and JSON forms.
- Native parent-child edges are preserved; subagents are never discarded merely
  because they cannot be resumed as roots.
- Pair tags bind only through explicit ranked evidence, with ambiguous or
  conflicting candidates left unbound and fully explained.
- Identical filesystem facts produce byte-stable tree order regardless of walk,
  glob, process, or `lsof` ordering.
- The current session watcher consumes the shared inventory/correlation model
  and no longer selects a root through an unreported first/newest heuristic.
- Fixtures cover all supported agents and a no-LLM live conformance probe
  detects native directory-shape drift.

## Plan

- [ ] Define the pure session-node, forest, evidence, correlation, ambiguity,
      diagnostic, and deterministic-ordering model.
- [ ] Implement complete agent-specific filesystem scanners for Claude, Codex,
      Agy, and Muse, including native parent/subagent metadata.
- [ ] Correlate Pair configs, ledgers, and identity-authorized live process
      evidence to native roots without chronology-based assignment.
- [ ] Expose structured and human inventory output through the Pair binary.
- [ ] Replace session-watch and transcript point-selection heuristics with the
      shared model and add portable fixtures plus live conformance.

## Log

### 2026-08-28

Split from #152 design. The operator identified that durable repository state
plus the native transcript tree is sufficient for recovery; the missing
foundation is deterministic, explainable reconstruction of every supported
agent's full root/subagent forest and its Pair-tag bindings.
