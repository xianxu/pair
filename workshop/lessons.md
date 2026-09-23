# Lessons

This is the compact rulebook distilled from the incident history. Keep rules
that prevent a class of failure; put incident detail, transcripts, and one-off
recipes in the issue or plan that owns them. References in parentheses point to
representative evidence, not an exhaustive index.

## Proof and verification

- Test the behavior at the production boundary that decides it. A parser,
  helper, or framing test does not prove routing, attachment, scheduling, or
  lifecycle behavior. (#139, #255, #265)
- A test must fail when the code under test is reverted. Assert the guard's own
  outcome and that it caused no forbidden effect; a later refusal or generic
  `err != nil` is not evidence. (#209, #230, #256, #280)
- Mutate the wiring as well as the extracted helper. A pure function can be
  perfectly covered while its call site is dead or bypassed. (#288, #297)
- Prefer positive state transitions over absence scans. When a writer moves,
  repoint the oracle at the new owner's state; an old byte pattern can make a
  negative test vacuous. (#289)
- A claim about every site needs a derived enumeration of every site. Tables
  copied from the same list as the assertion cannot detect omissions. (#206,
  #256)
- Probes distinguish false, true, and inconclusive. A timeout, missing sidecar,
  query error, or crashed observer must not be interpreted as proof of absence.
  Time the observation window from the event being judged. (#139, #223, #262)
- Verify that a mutation actually applied and crossed the intended branch before
  trusting its result. A failed replacement, wrong flag, or invalid command can
  produce a green-looking check. (#183, #262)
- Measure the path production takes, not the function that seems relevant. State
  the input that maximizes a cost and test the competing outcome. (#239, #262)
- Run the whole relevant suite, including race and acceptance boundaries. A
  package pass, `make -k`, or a hidden pipeline failure is not a green release.
  Verify the verification command itself. (#139, #262)

## Authority, ownership, and identity

- Give each fact one production authority and make consumers derive from it.
  Negative greps, duplicate registries, and prose tables drift. If a rule fails
  twice, turn it into an executable check. (ARCH-PURPOSE, #206, #256)
- A receipt, filename, PID, display key, or durable identity locates a record;
  it does not prove the resource's semantic identity or liveness. Store the
  canonical resource ID, incarnation, producer, and failure state needed by the
  decision. (#239, #248, #256)
- The process that presents a resource may differ from the process that created
  it or owns its lifetime. Measure parentage and record presentation context at
  attachment. (#256, #282)
- Revalidate authority after slow I/O and before mutation. A pre-check becomes
  stale while a probe, wait, or external command runs. (#255, #256)
- An action must consume the classification that admitted it; do not re-derive a
  second answer during commit or preview. Enumerate producers × actions through
  completion, not just preflight. (#256)
- Shared keyed state needs producer provenance. A cleanup must delete only the
  entries it owns, and a transition name should encode the proof its caller has
  established. (#206, #280)
- Acknowledgement transfers permission to execute; it does not transfer
  ownership. Keep authority, observation, and presentation as separate concepts.
  (ARCH-PURPOSE, #256)

## Async, concurrency, and process lifetime

- Name the owner of every goroutine, timer, lock, callback, and critical section.
  On cancellation or panic, release it, join it, and restore the visible state.
  Comments are not a lifecycle mechanism. (#209, #239)
- A timeout bounds a phase only when a live owner enforces it. If the owner can
  die, make the deadline observable and recoverable without that owner. (#250,
  #280)
- Process cleanup is one observable transaction: signal the right incarnation,
  wait with a deadline, reap descendants, and report residual ownership. Do not
  confuse a zombie, a dead PID, a detached session, and a missing record.
  (#256, #287, #288)
- Readiness observers must not kill or race the component they wait for. Require
  producer-issued evidence, clear it before launch, and compare against a fresh
  baseline. (#287)
- One-shot setup must cover every scope in which state lives: screen, buffer,
  pane, client, process, and reconnect. Reassert or migrate state at each
  transition. (#279, #245)
- Concurrent transitions need one synchronization owner and an exclusive
  transaction token. Panic recovery must not strand the lock or leave half of a
  resize, replacement, or shutdown visible. (#239, #251)
- Child fakes must model the real API's return values, cancellation races, and
  shutdown behavior. A fake that cannot express the failing interleaving proves
  nothing. (#206, #288)

## Interfaces, schemas, and data

- Reconcile active plan entity tables and task file lists with delivered code;
  appending a revision alone leaves the active mappings false. (#305)

- Treat a command, escape sequence, JSON record, sidecar, and persisted row as a
  closed grammar. Test unknown complete controls, prefixes, missing fields,
  empty fields, malformed records, conflicting evidence, and exact boundaries.
  (#139, #184, #223)
- Validate dimensions and lengths before allocation or indexing. Bounds apply to
  records, not only histories; sparse collections must be iterated by index when
  exact count matters. (#206)
- One schema across languages needs a producer-owned golden fixture and exact
  JSON types. Presence-aware types must distinguish absent, empty, false, and
  unsupported. (#184, #206)
- Durable text framing must represent the writer's entire input domain. Delimited
  formats need one shared codec with escaping and parity-aware extraction; do not
  let each subsystem invent marker parsing. (#184)
- Writes that can race themselves are atomic. Append-only stores expose an
  explicit commit result; readers classify mixed formats before opening them.
  (#206, #255)
- When relocating authoritative storage, audit inventory and archive readers as
  well as mutators. Each backend must recover its journal before an authoritative
  read; previews must instead refuse pending recovery without mutating. Test
  missing global discovery, stale global copies, and creating the second local
  backend after the first has enrolled. All local payload reads, including journal
  replay and restore comparisons, share the guarded path/type/size reader; test
  symlinked current records through lifecycle APIs. (#306)
- Historical compatibility has an immutable source boundary and an explicit
  migration policy. A cache must not survive an authority downgrade, and a
  relocated index needs an overlap-read epoch. (#255)
- Serialize replacement of an identity-bearing resource. Check the canonical
  identity while holding the same ownership boundary that performs replacement.
  (#255)
- A portable projection shares the owner's acceptance contract. Compact and
  diagnostic renderers may differ in detail, but neither may invent identity or
  claim fields its source cannot prove. (#206)

## Terminal, input, and UI boundaries

- Route input according to the currently focused, active surface. A key's bytes,
  pane role, screen, and owner all matter; test every encoding and every layer
  that can intercept it. (#245, #284)
- Pane navigation and mutation are ID-based, never relative. Pass the explicit
  pane ID to Zellij actions, and use a real attached client for live smokes.
  (#255)
- Terminal modes are protocol state. Track mouse, Kitty keyboard, alternate
  screen, cursor, and resize modes per target; reset them on release and migrate
  them when the presenter changes scope. (#251, #279)
- A stream split is not an event boundary. Use framing and a single writer for
  injected bytes; decode prefixes and unknown complete controls separately.
  (#183, #245)
- “Defer to the host” is an explicit disposition, not the absence of a handler.
  Preserve host selection, fallback, and escape hatches when adding an input
  layer. (#245, #283)
- UI text is a public contract. Update help, README, atlas, pasted-name tests,
  and recovery prose whenever a key, identity, or lifecycle behavior changes.
  Sweep retired vocabulary, including retry paths. (#251, #282, #291)
- Restore the prior visible state on cancel, failed async work, and rejected
  actions. A status message must not strand focus or cover the control it
  describes. (#245, #249)

## Planning, review, and repository hygiene

- When reusing lifecycle machinery, name its transition authority and cancellation
  owner explicitly. New durable files also need a final consumer and removal
  policy; tests should name risky functions and their mechanical guard. (#306)

- A plan's entity tables name live symbols and promised cases. Before a boundary,
  reconcile every checkbox, acceptance row, concept table, and revision with
  observed evidence; do not tick a row because the code exists. (#262, #297)
- Classify each plan function by its effects: deterministic transitions are pure;
  filesystem reads, process launches, and mutable proxy state are integration
  points. After a capture version or config scope changes, reconcile active
  plan paths with delivered artifacts and record the delta. (ARCH-PURE, #300)
- A milestone or close commit carries its own review verdict and evidence. Keep
  estimates measured, issue status owned by `sdlc`, and durable docs synced before
  long-running work. (#134, #146, #206)
- A public behavior change and its operator prose land in the same window. Sweep
  callers, tests, comments, atlas links, and generated helpers when renaming or
  removing a symbol; grep rendered output, not only source identifiers. (#183,
  #206, #251)
- Keep generated review artifacts bounded and raw transcripts out of source
  artifacts. Cite durable refs that exist on the published branch. (#223)
- Preserve the current source before `git mv`; stage content edits before moving
  issue files, and verify the target is tracked. (#64, #134)
- Use a real build for dogfooding (`make install` where the launcher needs the
  installed binary). Avoid slow multi-round-trip orchestration in hooks whose
  invoker reaps them. (#60, #207)
- Shell pipelines report the last command's status. Use JSON builders rather than
  hand-written `printf` JSON, and inspect output before assuming a command ran.
  (#183, #262)
- Comments and review claims are claims. Check them against the mutation boundary,
  cite symbols rather than line numbers, and record uncertainty as uncertainty.
  (#221, #223)

## Language and tool sharp edges

- In Lua, `\0` is an empty-position pattern, not a NUL byte; run `luac -p` before
  the suite. In Go, `strings.ToLower` can change byte length, and `gofmt -w`
  accepts a directory wider than the intended diff. (#183)
- In shell, `printf` reuses its format arguments and `jq -s` aborts a whole JSONL
  read on one bad line. Check empty fields and string-keyed table counts; do not
  use `#table` for ID generation. (#183)
- OS liveness needs more than `kill -0`: validate the PID and process identity,
  account for zombies, and treat pseudo-filesystem reads as fallible. (#139)
- A fast path is pinned by its answer, not by the work it skipped. A rate-limited
  count test may only be testing the limiter; an aliasing test must force an
  in-place overwrite. (#206)
- When a fix closes a class, enumerate the evasions and mutation-test the guard.
  A carve-out with no instances is not a useful rule, and a check that fires on
  every run becomes background noise. (#209, #221)

## Working rule

When in doubt, draw the boundary first: who owns the state, what evidence can
prove it, which production path delivers it, and what test fails when that path
is removed. Prefer the smallest explicit authority and the strongest observable
proof; record the surprising case so the next change starts from evidence.
