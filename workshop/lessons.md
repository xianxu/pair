# Lessons

This is the compact rulebook distilled from the incident history. Keep rules
that prevent a class of failure; put incident detail, transcripts, and one-off
recipes in the issue or plan that owns them. References in parentheses point to
representative evidence, not an exhaustive index.

## Proof and verification

- Bound diagnostic queues against measured burst shapes, including record count
  as well as bytes. Report capture loss through the owning UI while it is running;
  a teardown-only error can leave an operator waiting on a recorder that stopped
  minutes earlier. Keep admission budgets distinct from total allocation overhead.
  (#404)

- Optional diagnostic resources need one owner across startup failure and normal
  exit. Report shutdown failures on both paths, and clear process-scoped capture
  activation from child environments so descendants do not silently opt in. (#379)

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
- An effect's arguments are proved by the table that consumes them, not by a
  literal copy of their shape. #367: the switcher's slot resume sent only its
  path while `resume` declares `repo-scope` Required, so DispatchOperation
  refused every slot resume since #306, and the shape test pinned the bug.
  Send each argument shape through the declared operation table.
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

- A judgment about one member of a composite reads that member's own facts;
  a merged view across members is for display only. Keep per-member facts in
  per-member fields so no decision can read the union, and prove it as a
  domain property (vary only the other members' facts, the judgment must not
  move). #367 hit this family twice: dependency claims judged on the host's
  branch, then dependency dirt and commits counted as the host's (M1 review).

- Model present/absent/unknown at the producer instead of reconstructing it
  from booleans in each consumer. Persistent refusals need the failed resource
  and an explicit recovery action, not only a retry instruction. (#350)

- A refusal or notice that names a next step must name an action reachable from
  that row or caller, so choose the text from the same authority that decides
  the offered actions, per row kind. Hand-written advice drifts from the menu
  each time an action is added, removed or narrowed. (#363, three times in one
  issue)
- A sweep for a removed name must match it in prose too, not only as a quoted
  identifier: "retry open-slot" survived #363's quoted-name grep in an error
  string, and comments kept describing archive and name/describe. (#363 M2 BR)
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

- Build an identity that is compared against an external tool's output from that
  tool's own answer, never from a naming convention. Git reports resolved paths and its
  real common directory. #387's `--show` built a slot from `<primary>/.git` plus an
  unresolved symlinked path, so a healthy slot read as "registration absent, host
  mismatched". Resolve first, then ask the tool (`Discover`), and test through a
  symlink.
- Positive evidence is the tool's answer to the question asked, not a structural proxy
  for it. "Is this a checkout of its own?" is `rev-parse --show-toplevel == path`, not
  "the git dir sits at path/.git". A gitfile-backed clone fails the proxy and would have
  been set aside. (#387)

- Classify an outcome by what it means to the caller, not by which component
  failed. #387 first made a slot "blocking" whenever certain resources failed, which
  would have refused to open a working checkout whose resting branch was checked out
  elsewhere. Stating the invariant from the result ("blocking iff no agent could
  work") over the whole domain exposed it.
- Advice must name an action that can succeed from the state that produced it, and
  the named action's own admission must accept that state. #387's hold said "reboot",
  and reboot's first pass refused on the same hold (BR-10). Test the advised action
  from the advising state.
- A decision made in two places (a planner and the step that executes it) is one pure
  function both call. #387's plan and its set-aside step each mapped agent evidence to
  a hold, and diverged on "unknown" (BR-12).

## Async, concurrency, and process lifetime

- Name the owner of every goroutine, timer, lock, callback, and critical section.
  On cancellation or panic, release it, join it, and restore the visible state.
  Comments are not a lifecycle mechanism. (#209, #239)
- Enforce an automatic-input deadline after paste as well as before it; a late
  matching render must never revive an expired submit. (#353 design review)
- A budget has one owner. Nested deadlines take the minimum, so a transport
  or wrapper that adds its own shorter timeout silently overrides the owner's.
  Test the deadline the I/O actually sees through the production path, not the
  constant that names it. (#383)
- A wait added to a shared lock path is a per-caller decision: a caller that holds
  another lock or authority must not wait (yield instead), or lock ordering breaks
  tests and liveness elsewhere. Run every package that takes the lock, not only the
  one being fixed. (#367 smoke-test side-quest broke couchsingleton adoption)
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

- An error on a lookup path is surfaced, never collapsed into "not that kind of thing".
  A failed probe that returns `(false, nil)` silently turns "could not tell" into "no".
  Every error source on the path fails the command, or appears on the result as an
  error line. Write one test per source. (#387 `--show`, two review rounds)

## Interfaces, schemas, and data

- Reconcile active plan entity tables and task file lists with delivered code;
  appending a revision alone leaves the active mappings false. (#305)

- Correlation IDs identify records; they do not authenticate accompanying text.
  Resolve the canonical payload and recipient before acting on relayed content.
  (#353 design review)
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

- When scrolling grouped rows, test a selected group taller than the viewport;
  anchoring to its final child must not hide the owning row. (#371)
- Configure immutable pane identity in fixtures before starting the console;
  locking only the test's later writes cannot synchronize unlocked readers. (#371)

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

- When inserting a declaration before another one by anchoring on its `func`
  line, insert above that function's doc comment, not between the comment and the
  `func`. Godoc then attributes the comment to the new declaration. #387 did this
  twice (renderThreads, recoverReason); `TestNoDeclarationCarriesTwoStackedGodocs`
  catches it, but only in the full suite.

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

- A stronger identity validator must audit the provenance of every supplied
  identity. Conventional slot paths locate storage but do not prove that Git's
  common directory is `primary/.git`; exercise separate-Git-directory enrollment
  and restart through production storage readers. (#355 M2 BR-4 round 3)

- Search precedence must be applied across the complete displayed inventory,
  not separately per row kind. Test exact references against descriptive matches
  in every competing kind and UI view. (#163 BR-1)
- When a new syntax extends an existing one, sweep that feature's docs for sentences that denied the new form: "There is no `!!` escape" sat right under the new `!!` paragraph. Also reuse the command builder a sibling path already has instead of rebuilding the argv. (#358 close review)

- Tracker close bindings name exact commit IDs. Rebasing a closed branch can leave its card pointing to unreachable review/evidence commits even when every patch is unchanged; preserve close ancestry when integrating main. Read current status through SDLC, since details on main retain stale card mirrors until branch publication. (#358)

- An empty terminal snapshot can predate operator input, and whitespace drafts
  can look empty even after repaint. Automatic input needs ownership evidence
  at input admission, retained until genuine submission, then a fresh-render
  fence. A visual empty check alone is insufficient. (#353 integration review)

- Terminal focus notifications do not create human drafts, and a menu-confirming
  Enter is not a model submission. Distinguish both before assigning persistent
  input ownership or replenishing a peer-message allowance. (#353)

- For terminal-delivery failures, retain the guard that blocked progress in the
  expiry receipt; a generic timeout cannot distinguish layout recognition from
  input bookkeeping. The final #353 contract supersedes persistent human-draft
  ownership above: inspect current composer state after input settles, protect
  incomplete input, and poll only while delivery is pending.

- A mutex around automatic input callbacks does not own the composer between
  paste and submission. Share a transaction lease across all automatic writers,
  test both arrival orders, and require a fresh empty repaint before releasing
  abandoned or submitted input to the next writer. (#353 BR-1)
- Derive transport limits from encoded worst-case domain bounds, including JSON
  escaping and aggregate discovery responses; test every request/response family
  at its limits through actual sockets. (#353 BR-2)
- Incarnation-specific runtime handles need dead-owner collection after crashes,
  not only graceful teardown. Preserve unknown/live owners and replacement inodes
  in tests, and make fixtures name the actual listener process. (#353 BR-3)
- Before close, reconcile proposed entity names and file paths with implemented
  symbols, including stateless functions that replaced planned objects. Record
  the final mapping as an explicit plan revision. (#353 BR-4)
- When acceptance promises a fallback “or none,” exercise both populated and empty fallback states through the public command and assert stored as well as displayed values. (#357 BR-3)

- Validate family identity against the full inventory before eligibility filters;
  excluding the caller or a busy candidate must not hide an ambiguous repository.
  Test the broker effects, not only the pure resolver. (#353 BR-5)

- A display override must be tested in every label source it competes with,
  in a fixture without the structure the feature usually rides on. #360's
  alias passed with a repository that had slots (group name) but lost to the
  thread's operator name in the switcher and to the pane label on the tab for
  a repository with only `:0`. Enumerate the label inputs (row name, pane
  label, group name) and assert the override against each. (#360 smoke)

- A read-only listing must not re-establish global state on the request path,
  and periodic verification must name its cost per unit per second. #353's
  `--actors` re-probed every slot serially (zellij and process probes) under a
  2s transport cap equal to the client's, while one-second heartbeats and
  reconciliation ran ~3 full checks per wrapper per second over a binding map
  that never shrank, so the server saturated as slots accumulated. Serve
  listings from what the heartbeat already observed, bound a verification's
  reuse window, give the server a budget inside the client's, and remove dead
  entries. (#360 smoke)

- Prefix resolution must run over the complete namespace a name can belong to,
  not the subset that is currently reachable. #360 BR-1: messaging resolved
  against live slots only, so `brain:0` with brain offline and brainstorm live
  prefix-routed to brainstorm. Include known-but-offline names so an exact name
  resolves to itself and misses, and pin it with a test whose control case
  shows the narrowed namespace rerouting.

- A selective-stall experiment must timestamp child input receipt independently
  of displayed output and probe the control UI while pressure is still active.
  Use a populated menu and real input dispatch for dismissal; an empty menu or
  direct state switch can conceal a broken fixture. (#373 spec/fixture review)

- PTY producer completion on a side pipe and publication Flush do not prove that
  the PTY reader has consumed the final bytes. Require terminal-stream completion
  evidence for every child before reporting recovery. A test deadline must also
  interrupt blocking observations and input writes, not just its polling loop.
  Join emulator readers before closing unsynchronized emulator state. (#373 BR-1–3)

- When a format moves into a shared package, say where else it is mirrored.
  #372: `slugline` called itself "the one definition" of `=== L | R ===` while
  `nvim/slug.lua` re-implements the same recognition, and a package const
  named `close` shadowed Go's builtin. A cross-language format has one
  definition per language; the doc names the mirror so a change touches both.
  Never name an identifier after a Go builtin (`close`, `len`, `new`, `copy`).

- A pure reducer's effects are only as true as their execution. Any effect the
  shell can fail to apply must come back as an event, or the reducer runs ahead
  of the world it models. #365 BR-7: `EffectConnect` dropped a broker
  `Register` refusal, so the registry reported a binding connected that the
  broker never held, and nothing retried. Enumerate the effects, and for each
  say whether it can fail and which event reports that. Test one refusal through
  the real executor.

- A test that warms a cache before exercising the path it names tests the
  cache. #365 BR-12: the lost-receipt test queried status first, which
  recorded the receipt, so the re-send short-circuited in the broker and the
  wire `already-committed` path went unexercised. Order the claimed path first,
  or use a fresh instance per path, and mutation-check by reverting the guard
  the test's comment names.
- A component that answers for others must prove it is entitled to the
  answer. #365 BR-13: status recovery accepted a receipt from whichever wrapper
  replied. Bind each answer to its source (`receipt.To == answering binding`),
  and test a forged answer.

- Pin the supporting roots with an adopted store: its registration alone cannot
  prove which Pair artifact root belongs to it. Keep selected inventory identity
  separate from a held runtime lease, and revoke shared ownership handles on close.
  Exercise both distinctions through the production command boundary. (#366)

- Adoption evidence must cover every inventory backend and every ownership kind:
  global-store success does not resolve per-slot errors, and a free supervisor
  lease does not prove recorded wrappers absent. Hash external slot state under
  its transaction locks and preserve unknown liveness as a refusal (#366 BR-1/2).
- Share persisted payload limits between writer and reader; a successful publish
  must always produce a readable record, including large valid lists (#366 BR-3).
- Verify environment handoff at its consumer, including artifact creation and
  subsequent reads/resumes. A global data root and a repository-scoped artifact
  directory are different contracts even when both variables say data dir; an
  env-dump test alone cannot establish correct storage behavior (#366 BR-4).
- A record that outlives what it describes is a claim, not proof. #378: the
  actor registry is never forgotten on child exit, and archive read its rows as
  "hosting", so a dead agent bricked a parked thread. Probe before trusting, and
  feed unknown answers into the classifier's existing Unproven side rather than
  short-circuiting ahead of its precedence (#378 BR-1).
- A fixture that "is live" because nothing checked liveness proves the bug,
  not the guard. #378: two hosted-actor tests never marked their pid alive and
  passed only through the defect. Set the state a test's name claims.
- Isolation validates derived defaults as well as explicit overrides. A contained
  root does not contain a child path whose existing symlink points elsewhere;
  validate the fallback HOME, temporary and XDG roots before publishing selection
  or creating directories, and export only the validated physical paths (#366 BR-5).
- Two probes are not one observation. `kill(pid,0)` then an identity read can
  straddle a reap, so a process that just exited reads "unknowable". Re-ask the
  cheap probe before reporting Unknown, and keep exactly one implementation of the
  pair (`observeExactProcess`). #389 fixed the race once, and five hand-spelled
  copies kept it (#389 BR-1); grep for the pattern, not just the call site.
- Sweep a constant's dependents across the whole tree, not just the packages
  that obviously own it. #393 retargeted test fixtures in the four retention
  packages, but `workbenchshortcut` also built a diagnosticlog clock from a
  hard-coded 8 days, and main stayed red until #397's `go test ./...` caught it.
  Grep the literal shape (`8 * 24 * time.Hour`) repo-wide, and run `go test ./...`,
  not only `make test`, which runs a subset of Go packages.
- Never end crash capture (or anything a panic must outlive) from a `defer`. Go
  runs defers while a panic unwinds, before the runtime writes the panic, so a
  deferred `SetCrashOutput(nil)` and empty-file removal deleted the very file the
  panic was about to fill (#397 BR-1). Clean up after a normal return in `main`
  instead, and test with a crashing child that has defers pending. A child with
  no defers passes either way.
- A regression test must run through the code path where the bug lived. #397's
  first BR-1 test called a helper below the faulty `defer`, so re-adding the bug at
  the production site stayed green. Mutate the real site, not the test's copy of
  it. In a re-exec crash test, the parent must own every directory the child
  writes: Go's test runner runs `t.Cleanup` (deleting `t.TempDir`) while a panic
  unwinds, before the runtime writes the crash.
