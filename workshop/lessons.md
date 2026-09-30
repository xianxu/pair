# Lessons

This is the compact rulebook distilled from the incident history. Keep rules
that prevent a class of failure; put incident detail, transcripts, and one-off
recipes in the issue or plan that owns them. References in parentheses point to
representative evidence, not an exhaustive index.

## Proof and verification

- A filename absence decision that creates a replacement requires complete
  enumeration. Carry failed/partial probes as unknown through every consumer;
  an empty ID must not silently select a destructive fresh fallback. (#346)

- Documentation edits can break executable contract checks. Run the complete
  relevant suite after the final review fix, including README checks; validate
  every durable ID field that can reach agent argv, not just request fields. (#346)

- Capture external predicate status and both output streams from the running
  client/server before encoding a fake; CLI help alone may describe a different
  transport behavior. Keep false distinct from query failure. (#341)

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
- A loss oracle must first normalize every benign transform between sender and
  receiver — the sender's own stripping, the receiver's re-wrapping and
  boundary insertions — and its comparator needs a unit test on exactly those
  transforms. Otherwise the probe reports a loss that is its own misreading.
  (#211: a paste-boundary blank line split a numbered line, and `===` comment
  stripping read as 37 holes.)
- Locate a loss by hop before fixing it: probe each hop of the path in
  isolation, then diff sender logs against the receiver's own record at scale.
  The theory that survives elimination is not yet the cause. (#211)

## Authority, ownership, and identity

- Model present/absent/unknown at the producer instead of reconstructing it
  from booleans in each consumer. Persistent refusals need the failed resource
  and an explicit recovery action, not only a retry instruction. (#350)

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
- A terminal double must consume what the real terminal answers. An emulator
  that replies to queries (DA, DECRQM, OSC 10/11) wedges its own Write when no
  one drains the reply pipe, so every host fake built on one drains or forwards
  its replies from construction, not after the first hang. (#247)
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
- Revert a mutation from a byte copy taken just before it (`cp` + `cmp`), never
  `git checkout <file>`: that restores HEAD and silently drops the uncommitted
  work under test. (#317)
- When a fix closes a class, enumerate the evasions and mutation-test the guard.
  A carve-out with no instances is not a useful rule, and a check that fires on
  every run becomes background noise. (#209, #221)

- When chrome starts consuming inventory metadata, test publication while an idle
  child keeps focus: updating the model or repainting only the panel leaves tabs
  stale. Await both operation completion and painted geometry between clicks. (#307)

- Validate the enclosing tagged target before its payload: a valid slot nested
  in a contradictory target is still invalid. Normalize fallback identity once
  at ingestion so rendering, selection and dispatch cannot disagree. (#307)

- A resolved argument vector is explicit input to a child, but that does not
  authorize the child to publish it as a repository default. Carry provenance
  through ordinary launch paths as well as special fresh/resume paths. (#308)

- A fake clock whose `Sleep` is atomic hides latency when the test injects its
  event from the wake-up hook: the event lands on the poll grid, and a blind 60 s
  sleep looks instant. Stamp the event at its true time, off the grid, and
  mutation-check the test against the old cadence. (#316)

## Working rule

- When an input rule applies across UI modes, enumerate each mode in the test
  and assert both its text effect and the absence of an unintended transition.
  (#338)

- A lifecycle fix needs a regression through the failing event order and final
  authorization boundary. A separate process-group assertion and a successful
  already-bound smoke do not prove detach-before-binding recovery. (#329)
- In asynchronous acceptance tests, wait for the final contractual outcome,
  not an intermediate ledger publication; preserve the last failure in the
  timeout diagnostic. (#329)

- Optional post-send side effects need boundary tests for missing, failing, and
  blocked executables as well as failed dispatch and retry. Ordinary prompt
  transaction tests cannot prove that tagged prompts publish only after success.
  (#337)

When in doubt, draw the boundary first: who owns the state, what evidence can
prove it, which production path delivers it, and what test fails when that path
is removed. Prefer the smallest explicit authority and the strongest observable
proof; record the surprising case so the next change starts from evidence.

- A fresh conversation may carry a newly reserved address or an established one. Test real claim storage through the launcher; permissive runtime fakes concealed an established-only check that rejected every new slot fresh launch. (#315)

- Readiness tests must connect the nonce sender, launcher and actual readiness reader. A successful fake observer hides a missing nonce handoff even when the child starts correctly. (#315)

- Select launch behavior by conversation identity: a new ID uses ordinary creation even inside an existing durable slot. Reuse the composed ordinary-launch test before extending the same-ID replacement protocol. (#315)

- When reviewing an integrated branch, distinguish pre-existing published changes from the PR delta against fetched remote main. Mark earlier review windows as historical when a later review supersedes them.

- A selection no-op must avoid selection side effects, not merely retain the same active index. Test parent output and external operations for already-selected targets.

- For external numeric fields, parse the entire token and reject duplicates or missing values; formatted scanning can silently accept trailing text and extra signs.

- Before a parent/wrapper requests terminal mouse reporting, name who implements text selection afterwards. Turning reporting on hands every drag to the wrapper, and if nothing below selects, native selection silently disappears (#311 → #326). The same policy was safe in couch only because its child, zellij, selects.

- A multiplexer's outer alternate-screen mode is not evidence of inner-pane focus. Route lifecycle shortcuts from positive pane-role evidence; missing best-effort registry entries mean unknown, not permission to relaunch. (#333)

- Shortcut scope changes also affect generated key help. Verify both hosting and current presenter: reattaching a Couch-owned session outside Couch does not remove its restart restriction. (#333 BR-1)

- A non-draft title does not identify the agent: right terminals satisfy that
  predicate too. For pane return/poke operations, select positive command or
  recorded role identity and test with unrelated panes before the agent. (#340)

- When acceptance changes a return destination, retain separate assertions for
  submission routing: a reordered fixture alone proves nothing unless the host
  checks the target of both body delivery and submit. (#340 BR-1)

- Exit-time recovery must be proven through orderly process exit and a new-process restore. Swap and undo may disappear on normal quit; they are not evidence of durable unsaved text. (#341 PQ-1)
- Branch restoration needs an explicit-selection exception before history identifies a file. An empty first human round advances HEAD without supplying that identity; preserve the authenticated active selection while still rejecting stale caches. (#341)

- Neovim `:qa!` can ignore quit-callback errors. Test ordinary failed-storage quit separately from forced-exit recovery; do not promise an autocmd can prevent explicit discard. (#341)
- A retained inactive editor buffer still owns review state. Gate direct buffer writes and exit recovery for every retained buffer, not only the visible activation. (#341)

- A producer payload remains owned by the producer until the consumer explicitly accepts application or deferral. Test refusal between admission and application, and preserve a replacement that arrives during processing. (#341 BR-1)
- Proactive editor observation must not wait on Git or subprocess history scans in typing callbacks. Coalesce asynchronous observations and retain fresh authority checks at mutation boundaries. (#341 BR-2)
- Repeated buffer activation must replace owned callbacks instead of accumulating them. Assert stable callback counts across return visits. (#341 BR-4)
- Acceptance has an uncertain outcome when callbacks fail after partial effects. Preserve the artifact without automatic replay, and retry failed cleanup without reapplying the accepted work. Subprocess waits can pump editor events, so polling also needs an in-flight guard. (#341 BR-1)
- Predicate CLIs can encode false as a nonzero exit: pin stdout and status together from the real command contract, and make stateful fakes reproduce both. Zellij hidden floating panes are `false` / exit 1. (#341 smoke)
- An async identity observation cannot authorize a later checkout read. Capture data inside the identity validation window and carry those bytes to the callback; test checkout movement after observation but before delivery. (#341 BR-5)
- Git checkout changes the working tree before publishing HEAD. Unchanged branch/HEAD alone is insufficient authority for working-tree snapshots; test with checkout deliberately paused mid-update. (#341 BR-5)
- Binary file lines and editor lines differ: CRLF bytes must be decoded before entering a dos-format buffer. Share decoding across activation/refresh and assert exact saved bytes, not just displayed text. (#341 BR-6)
- Process tests launched from a live Pair session must discard inherited session artifact variables and bind every writable path to fixture storage. Rebinding only PAIR_DATA_DIR is insufficient when explicit *_PATH variables override it; verify external sentinel files survive the actual process tests. (#341 BR-7)

- A recovery command may have authority to restore a binding without authority to rewrite launch configuration. Reusing a live producer with missing argv can erase saved options; keep repair writes scoped to proven authority and test config preservation through actual cold resume. (#346 plan review)

- An isolation sentinel must occupy the actual production storage path. Mutation-test the environment wrapper so leaking ambient roots changes that sentinel; a never-consumed marker cannot prove isolation. New recovery commands must be discoverable in both user documentation and the failing command’s diagnostics. (#346 M1 BR-1–3)

- Recovery and live observation must call one binding decision function, not separately combine matching candidates. Exercise per-message candidate intersections in parity tests. (#346 M2 BR-6)
- Every dispatcher family is either documented for operators or explicitly classified internal; enforce this across the family registry to prevent recurring recovery-command README omissions. (#346 M2 BR-7)
- Identity policy changes must sweep ledger-to-legacy projections as well as direct query callers. Test actual parsed ledger data through restart-marker construction; a fake already carrying a UUID hides a dropped provisional target. (#346 M2)

- When a terminal can be recreated independently of its conversation, sweep
  target-generation and receipt validators as well as launcher naming. Preserve
  historical source identity and validate the new target against its own binding.
  A durable registration marker from the previous launch does not prove that the
  replacement terminal started. (#355)

- Independent terminal lifetimes require exact incarnation authority in admission,
  registration, and cleanup; a conversation's old index cannot authorize any of
  those for its replacement. Test the composed launcher, not only a fake runner.
  Revalidate live ownership after blocking preparation, immediately before the
  attach or destructive effect. (#355 M1 BR-1/BR-2)

- A descriptive label must not constrain resource identity. Generated internal
  names need a safe fallback for Unicode/punctuation-only repository names and
  bounded label length; allocating counters carry uniqueness. (#355 M1 BR-3)

- Checkout containment is not repository membership: nested independent Git
  repositories need their own storage. Carry scope/common-dir identity through
  routing, migration, preferences, and inventory, not only admission inference.
  Permanent reservations need an explicit admission bound when removal is
  deferred. (#355 M2 BR-4/BR-5)

- Repository membership sweeps must include hosted-session observers, not just
  persisted records: a foreign nested actor must neither authorize nor veto
  enclosing-slot actions. Exercise in-memory and durable registries, preserving
  conservative handling of unresolved same-scope actors. (#355 M2 BR-4 round 2)
