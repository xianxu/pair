# pair#429 plan: couch --peek fast path

**Goal:** `couch --peek` answers live tails in well under a second by asking the
running Couch over its broker socket, and resolves transcripts only on request.

**Architecture.** The broker already holds every connected wrapper's binding,
and each binding names its slot (`Binding.Slot`, e.g. `pair:1`, the same ref
`--send-to` routes on). So the broker can map a slot reference to the wrapper
and read its tail without the CLI building a Couch: one socket round trip per
slot, read concurrently, as `--message-status` and `--broadcast-list` already do
(0.01s). Today's typed operation stays as the fallback, and it stops resolving
transcripts unless asked.

## Design

1. **Protocol (couchmessage).** The `tail` request names its thread either
   by `TailScope`+`TailTag` (pair#425, used by the typed path) or by `Target`,
   an exact slot `repo:N` whose repository part may be an alias or unique prefix,
   resolved the way `--send-to` resolves (`canonicalTarget`). Exactly one form;
   a family (`pair`) is refused. The response gains `TailThread{Slot, Tag,
   Agent}` so the CLI can print the header without the store. No session nonce
   or PID crosses: the answer stays read-only and identity-free.
   `couchmessage.ResolveTailSlot(target, bindings, families)` is the pure
   resolver (exported for the broker handler, unit-tested).
2. **Broker (couchcmd handleTail).** A `Target` request resolves against the
   connected bindings plus the enrolled families (`authority.families`), then
   reads the tail exactly as the scope+tag form does. Both forms fill
   `TailThread`.
3. **CLI fast path (couchcmd peek_fast.go).** `couch --peek REFS` without
   `--transcripts`: expand the refs (`ExpandPeekReferences`); if every one is
   `repo:N`, ask the broker for each concurrently. All answer → print the same
   `PeekResult` / `PeekSnapshot` (JSON and text) with `source live`. Anything
   else — no Couch, an older Couch, a non-slot ref, any slot failing — runs
   today's typed path for the whole request, which retries the live tail and
   names why it fell back to the recording. Single- and multi-slot peeks take
   the same path (Spec 3).
4. **Transcripts only on request (couchcore).** `peek` gains `--transcripts`.
   `PeekThread` reads the live tail first; it calls `SwitchContext.Resolve`
   only with `--transcripts`. The recording fallback takes the agent from the
   thread record (`LatestLaunchProfile`, else the first incarnation — what the
   resolver itself reads), resolving only if the record has none.

Rejected: a broker `peek` op that runs the whole multi-slot snapshot server-side
(more protocol for no measured gain; per-slot requests are ~1ms and already
concurrent); caching a Couch for the CLI (the cold build is the cost to avoid,
not to amortize).

## Steps

- [x] couchmessage: `Target` form of `tail` (validation), `TailThread`, `ResolveTailSlot` + tests.
- [x] couchcmd handleTail: resolve `Target`, fill `TailThread` + tests (fake endpoint).
- [x] couchcore: `--transcripts` arg; PeekThread transcript-free by default, record agent for fallback; test asserting no `Resolve` call by default, one with `--transcripts`.
- [x] couchcmd fast path + CLI flag; tests: fast path answers without building a Couch; falls back on no Couch / older Couch / failed slot / non-slot ref / `--transcripts`.
- [x] Docs: atlas/couch.md, couch SKILL.md, README peek line.
- [ ] Measure on the live workbench (binary, not shell function) — needs a Couch on the new build, so it is the TL's live check like #425's; record the timed real-socket test result here.

## Revisions

### 2026-10-10 — after close round 1 (BR-1)

- **Design 1:** `TailThread` is {slot, tag, agent, working path}. The broker
  reads agent and working path from the thread record (`authority.record`,
  `couchcore.RecordAgent`) so the fast peek's JSON equals the typed peek's
  (`TestFastPeekMatchesTheTypedPeek`). The wrapper's own agent is used only
  when the record can't be read.
- **Design 4:** `RecordAgent` prefers the thread's only incarnation's agent,
  else its latest launch's: the order the resolver used. The fallback never
  calls the resolver.
- **Design 1 resolver name:** shipped as `couchmessage.ResolveTailSlot`.

