# Lessons

This is the compact rulebook distilled from the incident history. Keep rules
that prevent a class of failure; put incident detail, transcripts, and one-off
recipes in the issue or plan that owns them. References in parentheses point to
representative evidence, not an exhaustive index.

## Proof and verification

- When opt-in diagnostics add configuration, update README as well as the detailed
  runbook. Bound repeated-launch storage through admission or evidence-preserving
  retention, not only individual files. Keep lifecycle phase authority in the pure
  model; IO executes its channel/notification effects. (#404 BR-1–BR-3)

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
- A guard that admits by an earlier observation must never begin new work on it.
  #205 BR-1: `Couch.Park` read "open park transaction" and skipped the thread gate
  to join it, but the transaction could close before acting, and a normal park
  would then begin a fresh one unheld. On the bypass path, map every mode that can
  begin to one that only drives existing work (`Retry` refuses an absent
  transaction), so a stale observation fails closed.
- Moving work off a goroutine moves it out of that goroutine's implicit lock.
  #205 BR-2: running `AbortStarted` via `GoTracked` made it race `Forget` on the
  unlocked actor registry, which the single console goroutine had serialized for
  free. Before moving a writer to a new goroutine, list the shared state it
  writes and lock it at the same change, not a milestone later.
- Undo only what you did. #205 BR-3: a refused continuation cleared every
  expected-exit mark on its thread's panes, including ones a park had set. Record
  the marks an operation adds (`markThreadExitsLocked` returns them) and remove
  exactly those on refusal. Recording "the marks I added" is not enough when the
  mark is shared: a later owner re-marking the same pane loses its mark to your
  undo. A shared mark must count its owners (`exitMarks`): undo decrements, and
  the real event consumes all (#205 M1 round 2).
- A drain's wait is an interleaving cell, so give each waiter its own test.
  #205 BR-4 found `RecoverActiveParks` and `AbortStarted` waiting untested because
  only `Leave`'s wait was. One waiter's test does not cover another.
- `select` does not prefer `ctx.Done()`. When the context is already cancelled
  and another case is ready, Go picks at random, so a "cancelled" loop
  sometimes takes one more step. #205 M2: a cancelled `Leave` occasionally
  started another thread, caught only by an existing pre-cancelled-ctx test
  that flaked. Check `ctx.Err()` before the select, and after it when the other
  case acquires something.
- Test the cancellation claim you write. #205's `Leave` comment said "started
  threads finish", but they share the ctx and stop at their own safe points.
  The test written for the claim failed on day one. A policy sentence about
  cancellation needs a test that cancels at that exact point.
- Operator advice is code: it must be steps that would have worked on the
  documented incident, and its test checks their ORDER, not that words appear.
  #399's startup refusal said "kill the server; its agent goes with it" and then
  "pkill -P …; kill …; pkill -KILL -P …". Both were wrong on 2026-10-06:
  descendants that outlive the server reparent to PID 1, where no child-of-server
  command finds them. List the tree while the server still parents it, kill the
  descendants, and only then the server.
- A test that already fails on main still has to be read, not skipped. #399
  added five production files and each was missing from the artifact inventory;
  TestProductionArtifactReferencesAreExactlyClassified named them, but it was on
  the "known failure" list, so two milestones passed without anyone reading its
  output. Grep a known-failing test's output for your own files at every close.
- Check how a process is spawned before claiming a tree snapshot covers it. #399's
  reaper skipped the title-poller pidfile reaper on the reasoning that helpers live
  under the zellij server. But the poller's parent is the launcher (Couch's hosted
  client, or standalone `pair`), not the server, so it was never in the server's
  tree; when the launcher dies it reparents to PID 1. That is exactly why
  2026-10-06 left stray `pair title` processes. (A first correction blamed
  `Setsid`, which Couch-launched Pair doesn't even apply. Check the actual parent
  in `ps`, not a plausible mechanism.)
- A test may only remove paths it created with its own `t.TempDir()`. Never derive
  a removal root by walking up (`filepath.Dir`) from a path a child process
  reported: on 2026-10-06 an unsandboxed #397 test ran
  `os.RemoveAll(filepath.Dir(filepath.Dir(store)))` on `$TMPDIR/TestX…/001`,
  deleted the real `$TMPDIR`, including zellij's sockets, and orphaned every live
  session (#399). Run unsandboxed tests with every `PAIR_*`/`COUCH_*`/`ZELLIJ*`
  variable unset and `TMPDIR` pointed at a short, dedicated directory. Short,
  because nvim sockets and some size-bounded fixtures break on a long one.

- When copying dependency tests into a narrow fork, audit fixture paths and run imported benchmarks once; preserve licensed fixtures or omit unsupported benchmarks rather than shipping broken benchmark entrypoints. (#379 BR-1)
- Test a projection through the seam that feeds it, with the seam's real
  contract. #214 BR-1/BR-2: the real resolver returns its resolution TOGETHER
  with a typed refusal, but the evidence pass assumed a refusal carried none.
  Two of the three new reasons were unreachable in production while every
  classify test, built from hand-made evidence, stayed green. For each value a
  projector can produce, keep at least one test that drives it through the
  real input contract.
- A property test that runs a lifecycle call at a fixed point only tests that
  ordering. #395 M1's hub test always called `Activate()` before any frame;
  the one ordering it skipped (LIVE already shown, then activate on a quiet
  screen) killed every broadcast after 1s. Place lifecycle calls at random
  steps too, and state in the invariant what the call must not do.
- A property test's observer must not repair the state it checks. #395's
  hub test drained every viewer after every step, so no queue ever
  overflowed, and the test passed against a hub with resync removed. Check
  by inspecting; let the random schedule decide who drains. Then prove the
  test bites with `go test -overlay` mutants of the invariant's code, which
  never touches the tracked file.
- Publish ending state before signalling the end. #395 M2's hub closed
  subscriber queues and only later closed `Done`, which gated `Err()`; a
  consumer that reacted to its closed queue read `nil` and told viewers the
  operator had stopped when the tunnel had died (1 in 300 runs). Store the
  reason first, then close the channels anyone can observe, and test by
  reading the state at the moment the first signal arrives, many times over.
- A client whose connection can close for good must say so on screen. The
  #395 viewer handled `end` and transient errors, but a refused or dead
  connection left the last frame up, undimmed, looking live. Every terminal
  state of a connection needs a visible state, and its test drives a fake
  transport into each one.
- Record a departure from the plan in its `## Revisions` in the same commit
  that makes it. #395's reviews flagged undocumented departures twice
  (an End message that became channel close; a Makefile target that became a
  Go-driven node test), and a third time for a rename the atlas still cited.
  A rename or departure greps the old identifier across code, atlas, plan and
  lessons in that same commit.
- Text that crosses a trust boundary is a closed vocabulary. #395 M2 sent
  `err.Error()` to remote viewers as the end reason; a wrapped `Serve` error
  carried a local address, and in M4 a unix-socket path with a username.
  Map known errors to fixed strings with `errors.Is`, use a generic fallback,
  and test with a wrapped error that contains a path.
- A caller that can stop waiting must not read state written by the work it
  handed off. #395 M3's broadcast start read an `adopted` flag after
  `runTerminalCommand`, which returns early on shutdown while the loop may
  still run the closure, so a race (seen under `-race -count=8`) could stop a
  session the loop then adopted. Decide ownership of a produced resource with
  one claim (an atomic CAS) that both sides attempt.
- Check-then-act on a shared file needs a lock around both halves. #395 M5's
  run records reaped a stale named-tunnel lock and then claimed with
  O_EXCL, each safe alone; two Couches starting together could both judge the
  same stale lock, and the slower one deleted the faster one's fresh claim. A
  flock around reap+claim makes the decision atomic. The window is narrow:
  racing two claimers 1000 times caught the unlocked version in only 2 of 3
  runs. The test holds the window open with a hook between reading a record
  and acting on it (`afterRecordRead`), which makes the bad ordering happen
  every time.
- A pattern that scrapes a URL from a tool's output must not match the
  tool's own hosts. #395's quick-tunnel regex took
  `https://api.trycloudflare.com` out of cloudflared's failure line as the
  tunnel. Anchor on what only a success can produce, and test with the real
  failure text.
- A test hook that holds one goroutine must not use sync.Once. `Once.Do`
  makes concurrent callers wait until the first call returns, so a hook that
  parks the first caller inside `Do` also parks every other caller, and the
  test serializes the race it was written to show. #395's deterministic
  claim-race test passed against the unlocked mutant until the hook used an
  atomic first-caller flag. Mutation-check a race test; an ordering hook can
  silently remove the ordering.
- Bring main into a mid-flight issue branch by merging, not rebasing. #395
  rebased before a slot move: one commit subject starting `#395` was eaten as
  a comment by `rebase --continue`'s message cleanup, the M1/M2
  `Review-Window` trailers came to name pre-rebase IDs, and `sdlc actual`
  (commit dates, plus the transcripts of the slot it runs in) read 1.79h for
  the whole issue, against 1.75h for M1 alone. M3–M5 and the close went
  unmeasured. A later `git merge origin/main` changed nothing that existed.
- Enforce a protected screen region where it is drawn, not only where input
  enters. #412's marks dropped points on the status row at input time, but a
  resize moved an existing mark onto the new last row, where the overlay
  tinted `LIVE ⏸` and would have tripped the broadcast's fail-safe. The
  overlay itself now refuses the last row, whatever put a mark there.
- Every argv reader and editor sees only the flag region before the first
  `--` (`resumeform.FlagRegion`). #410 BR-3/BR-6: the boundary was honored by
  the inserter, ignored by the strippers (they ate prompt text), and ignored by
  `hasFlag` (prompt text `--fork-session` suppressed the session-id mint). One
  helper, every site routed through it, and a test that places every managed
  spelling after `--` for every agent.
- A launcher registry row implies ledger membership and the whole session side
  in the same boundary. #410: every launch encodes a ledger record for its
  agent, so "registered, session side pending" is not a launchable state; a
  known-gap interim that only the parity test tolerates hides that the agent
  cannot start.
- Amend `## Done when` with a `## Revisions` entry the moment an item is
  dropped or moved, not only in Log prose. #410's close review (BR-7) blocked
  on three Done-when items the operator had moved to a follow-up in
  conversation and the Log, because the contract itself still promised them.
  Name where each item went (issue id) and add it to that issue's Done-when.
- A test that drives input through an async gate must wait on the gate's own
  view, not on a neighbour that updates first. #412's pointer tests posted a
  point once the active marker was on the operator's screen, but the hub,
  which admits points, sees that frame a moment later; under `-race` load the
  point was dropped and the test hung. Retry the input until its effect shows
  (a helper re-taps), and make a "dropped" test first prove the path accepts
  input, so its drop can't be the race.

- **When a doc claims "X isn't needed because Y", test the claim on the real
  client, not in theory (#415).** I wrote "no preload needed, every glyph has
  the cell advance" for the broadcast symbol font. The iPad smoke disproved it:
  xterm.js's DOM renderer measures each character on first draw and keeps a
  `letter-spacing` correction, so a `unicode-range` face fetched by that draw
  is measured as its fallback, and the real glyph collapses to zero width. Any
  font xterm.js may draw with must load before the first frame. A
  correction must reach every place that repeated the claim (atlas, CSS
  comment, VENDOR.md), not just the first one found.

- **A byte-stream rewriter frames sequences in every mode and injects only at
  sequence boundaries (#417).** The focus-mode dimmer tracked escape
  sequences only while dimming. A mode flip could land between two halves of
  the agent's own CSI, and that split the sequence, so its remainder printed as
  text. My test pinned the corrupt output as expected. Hold a split sequence's
  tail in both modes, emit a transition only before the held tail, and test
  chunk invariance: for every split point, two feeds equal one.
- **Optional IO on a shared signal loop gets a deadline (#417).** The dim
  observer ran `zellij list-panes` with no timeout on the goroutine that also
  delivers wrap's capture and restart signals. A cosmetic feature must not be
  able to stall those; bound it with a context deadline.
- **A user-visible behavior change updates README too, not just help and atlas
  (#417).** grep the old phrasing ("toggle ... fullscreen") across README,
  atlas and help strings before close.

- **A projection of another program's rendering is a claim about that program;
  capture it, don't assume it (#418).** Peer delivery modeled Claude's composer
  wrapping with `ansi.Wordwrap`. That function also breaks after hyphens, and
  Claude does not, so every long message with a hyphenated path expired. The
  first fix guessed again, declining over-width words, until a live capture
  showed wrap-ansi's hard rule. Pin each rendering rule with a captured
  fixture that asserts it distinguishes the alternatives. State the rule once,
  on the function, and point every doc there.
- **Run `sdlc change-code` the moment the plan is committed (#419, #422).** Twice
  in one session I implemented straight after `start-plan` and ran change-code
  afterwards. The flow is claim, start-plan, plan commit, **change-code**, then
  code. Run it before the first test edit, even on the quick flow.

- **Derived state subscribes to every source transition, not just the noisy
  ones (#421 M1).** The settle check re-armed only on output and input. A turn
  can also open or close silently (watchdog, grace expiry, transcript record),
  which left an idle slot looking busy, or a working one looking settled. When
  state X is derived from Y, list every writer of Y and make each one notify
  X, with a test per direction.

- **Three review families recurred on #421; their rules:**
  - **A guard checked at admission is checked again at the effect.** Anything
    queued between them can change the fact. Re-check just before the
    irreversible step, with a test that flips the fact in between.
  - **Every declared operation is reachable through its executor.** Keep one
    table test over all declarations, so a missing dispatch case fails a
    test rather than shipping.
  - **A plan revision sweeps the body.** For every identifier or path a delta
    replaces, grep the plan and mark each hit superseded in the same commit.
  - **An effect-time re-check reads admission's decision, not a fresh
    default.** Pass what admission concluded (here `known` or `forced`) to
    the effect. Otherwise a fact that turns unknown in between is silently
    treated as the forced case.
  - **A removal sweeps its identifier.** In the commit that deletes or renames
    one, `git grep` it across code and the plan body, then remove or mark every
    hit. The review records are history and stay as written.
  - **An implicit argument is tested from producer to consumer.** A value
    admission writes for the effect to read (#421's `require-settled`) gets one
    table through the whole path: admit, dispatch, then the consumer. The table
    covers every verb and every value, including absent. Testing each end alone
    let a hard-coded `false` pass.
  - **A scope change marks every line that states the scope:** the Done-when,
    the issue's Plan row, and the plan body, all in the same commit.

- **Test the router, not just the handler (#424).** #421's slot-operation tests
  called `runSlotOperationCLI` directly. The CLI router in front of it kept its
  own four-verb list, so `couch --relaunch` broke on first live use. #421's
  "one verb list" sweep also grepped only for `case` lists and missed the `==`
  chain. When adding a verb, drive it from parsed argv through the top-level
  dispatcher in a test that enumerates every declared verb. Sweep every
  spelling of the old list, `switch` cases and `||` chains alike.
