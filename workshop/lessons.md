# Lessons

This is the compact rulebook distilled from the incident history. Keep rules
that prevent a class of failure; put incident detail, transcripts, and one-off
recipes in the issue or plan that owns them. References in parentheses point to
representative evidence, not an exhaustive index.

## Proof and verification

- Splitting an acceptance task into a child issue requires sweeping copied Spec tasks and Done when for parent-only checks. A Log saying the parent still owns console/deployment acceptance does not override an active child Spec clause requiring it. Keep the old clause only as superseded Revisions history. (#723 BR-1)

- A replay benchmark must prove every claimed stage executed inside its CPU bracket. NewFakeChild with nil Sink skips publication delivery entirely; count deliveries/bytes and validations, check batch.Err, and join publication plus reply workers before the ending sample. Retain incomplete-path runs as diagnostics rather than calling them full-path performance evidence. (#723)

- #714 PQ-2: Verify persisted compatibility against the real decoder and its adversarial tests before claiming unknown-field preservation. A schema-unchanged bugfix can retain an existing strict refusal contract; a preservation migration is a separate, explicit change.

- #714: Failure diagnostics compose whole identities, never nonzero structs or fieldwise fragments. Test a scope-only outer dispatch through actual normalization and reopened persisted history; a recorded Save call is not proof of a saved failure. Preserve publication warnings at the concise operator boundary too.
- A configuration-only rollout still needs root README coverage when it introduces an operator-facing setting. Sweep the config path/key, selected value, inspection command and approval owner together; an atlas entry alone leaves the usage contract incomplete. (#709 BR-1)

- #715 BR-1: A local extension to a shared tool updates both its adoption instructions and its rule-maintenance instructions in the existing atlas page. Document the local composition owner, scope, shared error/suppression semantics, and which upstream consumers remain unchanged; updating only the feature's lifecycle map misses the tooling surface.

- #715: A child that survives client detach still needs a lifetime owner. Keep its direct child handle and join it before wrapper return; process liveness polling does not fence startup or deferred retention writes. Gate a real watcher before lease acquisition, finish the native child, and prove no writer survives the wrapper scope. Preserve process-group isolation and measure prompt TERM handling separately from the refusal grace.

- Characterize an external reader on a separate disposable copy when a conforming implementation may create coordination files. Otherwise a diagnostic probe can change the source state before a production preservation test, selecting a different path on another platform. Assert the untouched source image around the production operation only. (#701)
- Preserve failure provenance across a fallback collector: uncertain source evidence may defer, but established unsafe paths/limits and local output failures must retain their rejected category and repair action. Test the categories independently; a blanket conversion can make an unsafe configuration look temporarily busy. (#701 BR-1)
- A fixture's readiness publication must follow every prerequisite the production consumer queries immediately. Observe the real reply synchronously between atomic writes, assert final readiness is absent there, and inject each write failure; a later parent wait cannot close a gap the concurrent admission observer already entered. (#698)

- A native signal fixture must control delivery as well as readiness. Python can swallow KeyboardInterrupt in an ignored-exception callback after a ready banner; mask through that phase, prove the signal is pending, and release it at top level. Check an independent expected exit as well as baseline parity, with a callback self-signal regression and no competing parent signal. Preserve wait-error text explicitly: formatting an unexported struct field can hide it behind a pointer. (#703)

- A display decorator must preserve its input collection's empty-versus-unavailable contract. Cloning by append to nil collapses an owned empty slice; run existing inventory tests as well as the new diagnostic-name subset when adding row fields. (#683 suite qualification)

- Optional terminal publication belongs after adoption on an existing joined worker, never the console reducer. Test blocked storage through actual input and Stop, and verify the publication context is canceled by console lifetime; a responsive callback mock alone misses a missing lifetime wire. (#683 BR-6)

- A shared diagnostic projection must be tested through every public reference for the same slot: slot name, canonical primary-root reference, path and exact tag. Centralize matching before latest-selection and visibility filtering so row/show parity survives missing-cache reference-only evidence. (#683 BR-5)

- Diagnostics must not become lifecycle admission authority. Preserve the original action result when optional target enrichment, nonce allocation or logging fails; test unreadable-record recovery through the public dispatcher before adding diagnostic prerequisites. (#683 BR-1)
- A terminal failure collector stays evidence-only until its enclosing operation finishes; test exact publication counts for standalone and borrowed callers, including final enrichment. (#683 BR-3)
- Run the full artifactpath package before a boundary. A regex selecting source/artifact names can omit Symbol guards and miss orphan exports introduced by the same change. (#683 BR-4)

- A design refinement that moves state or file ownership also updates the current Core concepts map, with the prior mapping preserved in Revisions. A later correct paragraph does not repair a contradictory table; distinguish pure value state from its IO methods explicitly. (#686 BR-1)

- Count every IO lane used by a performance claim, including publication guards outside the main reader runtime. Cache hits before a writer lock do not eliminate the guard's own full reread: reuse exact identity-fenced evidence under that same lock and prove append/replacement invalidation before effects. (#686)
- Derive a multi-item fixture's byte budget from every item it must admit. Equal source lengths do not imply equal archive or metadata lengths: fractional timestamps can change both. Use deliberately unequal fixtures and assert that each fits alone while their sum does not. (#696)

- Put exact-target authorization inside the shared journal replay operation, not only its original caller: reopen, append and maintenance can all replay a deletion. Test a held rotated payload through every replay entry point. Preserve completed effects separately from errors, and report retained blockers without labeling unattempted candidates as removed. (#677 close review)
- Classify scheduler errors at the actual collection boundary: stale declarations, changed certificates, malformed per-log state and blocked recovery are path-local blockers. Preserve them for preview and advance the bounded cursor; test the whole class before a later eligible path. Native acquisition, global inventory, cancellation and coordination failures remain distinct. A missing-parent-only exception does not satisfy this rule. (#677 close review)
- An asynchronous reservation must cover identities that become visible before its effect returns. Keep the original attempt and reservation, derive launch-authoritative aliases for admission and projection alike, and test inventory publication during held adoption. A primary slot is an ordinary scoped conversation; numbered-slot-only fixtures miss it. (#576 BR-1)

- A successful subprocess status does not identify its reply kind. Qualify diagnostics with multiple live sessions as well as zero/one: Zellij list_sessions exits internally, so a missing-target action can return exit0 with a session listing and not-found stderr. Keep that exception at the name probe, with captured bytes and strict whole-reply validation; never relax the clients decoder. (#672)

- Classify a concept by every operation its implementation performs: environment reads, test verdicts, diagnostics and subprocess policy checks belong to integration even when their decision rule is simple. When acceptance is revised, synchronize Done when, task rows and new README test switches in the same documentation pass, retaining the original decision in Revisions. (#659 close review)

- Register every new non-`_test.go` source in artifactpath's exhaustive inventory, including helpers used only by tests. Run that guard before queueing a full suite; a test-support package is still production-shaped source to the inventory. (#659)
- When migrating a temporary root, inspect every helper consumer, including ones that reuse its directory with a longer generated basename. Budget the complete native socket address and qualify the entire affected package under the OS's normal TMPDIR; a focused helper-name regex can omit the consumers that break. (#665 R8)
- An ordering fixture must not require every real subprocess to beat a product
  deadline. Use a stateful double for identity/byte changes and controlled
  completion order; keep native timeout and real backend conformance separate.
  Count synchronous waits for UI blocking assertions rather than measuring
  host speed. Lua empty strings are truthy: a failed process needs a nonempty
  diagnostic fallback even when stderr exists. (#662)

- A subprocess double models kill and owner teardown separately: native kill
  still delivers completion, while fixture teardown may suppress late delivery.
  Verify that distinction against the installed runtime so cancellation cannot
  strand a callback gate in its started state. (#662)

- Before adding text validation to a fixture double, check the existing pure
  document decoder. Reuse its UTF-8/NUL contract rather than copying the
  algorithm into a second implementation. (#662 close advisory)

- Ordinary Go files in test-helper packages still belong to the exhaustive production-source inventory. Classify every new file, including build-tag variants, and keep auxiliary scratch names outside reserved artifact prefixes; do not quarantine a missing inventory entry as a known violation. (#658)

- Adding diagnostic pipes changes subprocess completion: a detached child can hold stderr after a successful tool exit. Separate process success from diagnostic-drain cutoff, retain capture uncertainty, and test inherited descriptors behind a release handshake. (#638)

- Reserve shared banners before a menu frame windows its selected rows. Subtracting them only for some frame kinds lets final clipping hide selected actions and cursors; test every windowed frame at minimum geometry. (#638)

- Keep free-form tool stderr out of metadata-only diagnostic callbacks. A truncated or transformed echo defeats full-input redaction; publish typed status plus stderr length/hash and keep bounded output on the operator screen. (#638 PQ-1)

- A strict trust check and a native discovery rule can intentionally treat aliases differently. Failure to prove one observed discovery candidate makes the whole candidate scope unknown; silently omitting it can select an older conversation. Test the selected identity with a newer candidate behind an unsupported alias. (#571 BR-3)
- SQLite read-only mode can create native WAL-index sidecars. A non-mutating adapter queries an owned coherent private snapshot, and a real WAL fixture must compare the entire native file set and bytes before and after success, failure and cancellation. An integrity check validates recovery, not the coherence of separately copied source files. (#571 BR-4)

- Decode persisted authority into zero values and require its version/collection fields explicitly. Only file absence permits initialization: decoding `{}` over a prefilled default can erase unresolved ownership and admit a second writer. Test missing fields, top-level null and null collections separately from valid empty state. (#556 M2)

- Normalize an advertised choice through its validated key before applying mode-specific admission. A shared candidate route may represent Fresh as well as resume; test explicit and keyed forms against every native-only guard and run the printed command/menu route. (#556 BR-5)

- Never copy a synchronized owner to substitute one dependency for a preview. Pass the read-only store into its existing resolver; construct focused test fixtures from dependencies rather than copying live mutexes. Run copylocks vet after the sweep. (#556 BR-4)

- A getter may replay durable transactions or enroll retention through its lock path. Automatic startup/authentication readers use a shared preview accessor, and regressions preserve a valid pending journal in global and routed slot-local stores; prove explicit lifecycle reads can still replay the same journal. Carry cancellation through routing as well as the final file read. (#528 BR-12)

- Terminal integration assertions read projected cells, not concatenated delta writes: a cursor-positioned suffix can complete a phrase that never occurs contiguously in the byte stream. Await a view-specific marker before releasing concurrent effects; status-bar labels do not prove the switcher opened. Split effect-count and visible-outcome assertions so failures identify the missing observation. (#528 BR-11)

- A process-survival fixture synchronizes after the child installs its signal handler, before the parent exits. Forking does not establish that order; a FIFO handshake avoids false liveness failures under load. (#528)
- Tests that exercise host-wide terminal discovery need globally unique session names even when HOME and socket directories are isolated. Test concurrent independent runs; changing only the socket root does not isolate a name-based ownership query. (#528)

- Before claiming a design extends existing metadata, inspect the actual owner/schema. A path decoder is not a persisted attestation. Name the new authority explicitly and test its publication boundary. (#528 PQ-2)
- Admission locks protect the observable transition, not just process creation. Release after verified ownership/readiness and before waiting for the long-lived client; test both orderings with controlled events. (#528 PQ-1)
- A spawn API can return an error after it created and cleaned up a process. Preserve bounded evidence before that cleanup and carry it through the error; a nil returned handle does not prove nothing started. Test the post-spawn setup refusal, not only exec failure and registration failure. (#531 BR-1)

- A late asynchronous report is checked against whether its source is still live, not against a counter that only
  moves on the next start. #495's rotation callback compared the cast's start attempt, which an ended cast keeps until
  a new one begins, so a report after the end still passed. Test the window where the source has ended and nothing
  has replaced it. (#495 BR-23)
- A writer that turns text into terminal cells measures by grapheme width, never by rune count, in its layout and
  its drawing alike. One wide character (an emoji in a label) otherwise yields a frame that fails validation, and the
  whole paint is dropped silently. Test a drawn surface with a wide character in it. (#495)
- A runbook or atlas page states what was measured, not what was hoped: #496's page first called the scratch volume
  "the fix" although its own series showed no fseventsd CPU drop. Write the measured outcome, including the null result,
  in the same commit as the mechanism. On macOS, `mktemp` without a template ignores `$TMPDIR` and is denied in the
  agent sandbox; always pass `"${TMPDIR:-/tmp}/name.XXXXXX"` and `|| exit 1`. (#496 BR-1, BR-2)
- A change to what a user does (a gesture, a control, what a pin shows) updates README's how-to paragraph in the same
  commit as the atlas. The atlas is the agent's map and README is the operator's; grep README for the old verbs
  ("tap", "drag", "pin") before closing. (#488 BR-1)
- Retiring a class, function or field: grep every consumer, including env-gated live tests (`WEBCAST_LIVE_CHROME`,
  skipped in the default suite) and the atlas index, and run the gated ones once. A skipped test still asserts the
  old shape and is green only because it never ran. (#488 BR-7)
  The grep covers comments and test names too, and in the singular as well as the plural ("mode button"), in the
  same commit that retires the thing. (#488 close)
- Bound every untrusted integer before any arithmetic on it, and compare by subtraction (`w <= cols-col`), never by
  adding the request's own numbers: `col+w <= cols` wraps at 2^63 and lets a box through that then drives a loop
  for ~9e18 iterations. A fuzz oracle that restates the parser's addition shares the bug, so state its bounds
  independently. Pin a refusal table with a deadline. (#488 BR-10)
- An async job carries the exact observation it was made from (the frame, its endpoint, the session), and is
  checked against what is in view when it applies. Never re-read "latest" mutable state at apply time, and key
  dedup sets by every identity that can repeat (ids restart per session). (#488 BR-15, BR-16)
- Decide under a leaf lock, act after releasing it: starting a worker (`GoTracked`) takes other locks and can run
  inline. (#488 BR-17)
- Context gathered for the agent (file names, headings, layout) stays off the viewers' wire unless a page needs it.
  Tag it `json:"-"` and test that the wire lacks it. (#488 BR-18)
- A paint that snapshots state and then publishes it holds one lock from snapshot to commit whenever paints run on
  more than one goroutine. Otherwise an older snapshot can publish last and stay on screen until something else
  repaints, which on an idle screen is never. A test that polls for the new state passes in the gap before the stale
  publish lands, so pin the interleaving with a hook rather than a load soak. (#484)
- A bounded cache or graveyard prunes only after the update it belongs to is applied: decide what is gone from the
  structure as it now is, never between its teardown and its rebuild. #458's id map buried a re-snapshotted tree after
  forgetting it, so every live node looked removed and lost its id once the graveyard filled. Test it above the bound.
  (#458 BR-19)
- When several sources can supply the same thing (a response, a cache, a reference), name one keyed owner that
  decides, test every arrival order, and say what the later sources do. Deduping by "seen" flags set in arrival order
  double-fetches the moment the order flips. (#458 BR-24)
- A measuring tool's own error must be smaller than what it measures: a screenshot diff searched in 200 ms steps on a
  page that changes every 50 ms measured the step. Check the tool against a known baseline (the prototype's recordings)
  before comparing anything with it. (#458 M3)
- A fix that adds a review-requested behavior ships with a test that fails when the fix is reverted; check it by
  reverting. Five of #458's first-round fixes passed the suite reverted. (#458 BR-4..17)

- A filter that keeps records by kind must also keep the context records that give the kept ones their meaning.
  #457's catch-up kept a document's notes but dropped the segment they were counted from, so a derived view
  (notes on the current document) emptied while the raw state matched. Assert equivalence on the derived views
  consumers read, not only on the raw state. (#457 BR-1)
- A generator and its parser are one contract: never store an output the parser can't read back. Check the round trip
  at the point of emission (#461's `Anchor` drops a selector that doesn't find its own element), not only by
  restricting inputs, which misses the cases nobody foresaw. When a fix changes behaviour, grep every prose restatement
  (doc comments, atlas, plan) and update them in the same commit. (#461 BR-1, seam-doc-drift)
- A seam's record channel carries one kind of thing. Name the single writer of every side artifact (bodies,
  screenshots), and make each consumer refuse what it must never receive, with a contract test that checks it.
  (#457 BR-4)

- Budget a subprocess by what it does, not by who calls it: a step that writes
  a checkout is not a probe. A failure the operator saw only on screen should be
  recorded where the owner keeps its records, bounded to the store's own record
  limit, and cleared only after the success has been verified. A test that
  shrinks a budget to force a failure restores it before the step that has to
  succeed. (#442)

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

- An ownership fence must preserve exclusivity as well as the original process
  identity. A direct PID check can miss a new competing claimant while the
  original survives unchanged. Share only initial discovery; freshly enumerate
  candidates around slow IO and before effects. Test that exact interleaving. (#218 PQ-1)
- A PATH timing shim must transparently exec a service's internal server reexec.
  Keeping wrapper and child alive can create two exact argv claimants and change
  the ownership verdict being measured. Assert PID, argv, status and environment
  preservation, and keep blocking service lifetime out of finite query totals. (#218)

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
- Backgrounding a child from `sh -c` in a non-interactive shell redirects the
  child's stdin from /dev/null, so a long-lived child (an embedded nvim) that
  inherited an os.Pipe gets an immediate EOF and exits within ~150 ms even with
  the write end held open. Redirect its stdin explicitly — a FIFO whose write
  end the test holds — to keep it alive past its parent. (#583 live conformance)

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

- When acceptance changes, synchronize Done when itself with the expanded surface
  and any explicit waiver, alongside the append-only revision. A revision alone
  leaves the primary acceptance checklist stale. (#521 BR-1)

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
- A new production source file must be declared in artifactpath's exhaustive
  inventory (SourceClassifications or NonArtifactSources) even when it owns no
  artifact — absence itself is the violation. Targeted runs of the packages you
  touched cannot see it; TestProductionArtifactReferencesAreExactlyClassified
  is a whole-tree claim that only the full suite asserts. (#583 pre-close suite)

## Language and tool sharp edges

- When inserting a declaration before another one by anchoring on its `func`
  line, insert above that function's doc comment, not between the comment and the
  `func`. Godoc then attributes the comment to the new declaration. #387 did this
  twice (renderThreads, recoverReason); `TestNoDeclarationCarriesTwoStackedGodocs`
  catches it. After inserting any declaration, run that guard explicitly in
  cmd/internal/termcmd before queueing a suite; a diagnosticlog package run
  does not own the whole-tree comment guard. (#677 recorded-suite correction)

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
- **Protocol text that crosses repositories names no host repo (#419).** The
  Couch skill is read by agents in every repository, but my dispatch-label step
  hardcoded `pair#N`. Write the generic form (`repo#N`), use a foreign repo in
  the example, and keep a copyable command on one line, never wrapped inside
  its quotes.
- **An op an agent may run gets a human-readable render (#419).**
  `publish-description` fell through `render`'s `%v` default and dumped a raw
  struct; nobody saw it because nvim discards the output. Key a new render
  case on the op name, not the result type (`Detach` also returns a bare
  `ThreadRecord`).

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

- **Round trips do not prove wire compatibility (#426 plan review).** Two
  encoders with overlapping escape alphabets can invert each other while
  disagreeing with every external writer. Test independently authored raw
  protocol fixtures before composing codecs; preserve escape parity until the
  grammar that owns it has interpreted it.

- **Conceal replacement cells need their own role highlight (#426).** Coloring
  a turn's text span did not color the extmark's replacement brackets; only the
  PTY screenattr check caught it. Check actual painted cells when layering
  conceal marks over syntax/highlight spans.

- **Input guards belong before mutation, and handle mode entry (#426 BR-1).**
  CursorMoved/TextChanged cannot protect hidden text when `i` enters insert at
  a visible closing delimiter. Admit Insert/Replace entry and each character,
  including queued keys, and verify through actual editor input events.
- **Window identity includes its current buffer (#426 BR-2).** A float window
  can survive `:enew` after its scratch is wiped. Release ownership on scratch
  departure and check the window/buffer pair before both focus and cleanup;
  the replacement is user-owned.

- InsertCharPre is not a general pre-edit hook: newline, deletion and register
  insertion have distinct input paths. For protected geometry, test a matrix
  of mutation mechanisms after queued cursor movement before declaring the
  admission rule complete.

- **A changed outcome contract sweeps every test that asserts the old one
  (#427).** Making "submitted" mean "confirmed after the submit", not "the
  submit write completed", broke a test in a file no targeted `-run` pattern
  reached (`automatic_input_test.go`). Before the first targeted run, `git grep`
  the status or field across `*_test.go`, and run the whole package once.

- **"Router" means the entry function, not the arg binder (#425).** A new
  argument shape (a multi-slot `--peek` ref) passed `ParseCLI` and `bindArgs`
  in a test, then died on first live run: `runTypedOperationWithConsole`
  pre-parses `ref` as one `repo:N` to derive the repository scope. A test of
  argument shapes must run `RunWithRuntime(argv)` end to end; any stage that
  reads an argument before the executor is part of the router.

- **A fast path answers in the slow path's exact shape (#429).** The fast
  `--peek` built its result from what the broker happened to know (the
  wrapper's agent, no working path), so `--json` changed with whether a Couch
  was running. When a shortcut and a fallback return the same type, derive the
  fields from the same source (here the thread record, `RecordAgent`) and pin
  it with a test that runs both on one input and requires identical output.

- **A load gate's test uses the dominant production shape (#420).** The send
  gate was pinned only by a v2, non-chosen launch. Every fresh Claude launch
  is chosen-id, which took the ungated walk path, so the common idle watcher
  kept walking the store. When a cost fix branches on launch shape, origin or
  agent, list the shapes production actually mints (`createflow` mints
  `--session-id` for every fresh minting agent) and pin each branch with an
  operation-counting test.
- **A count helper's key shape is part of the assertion (#420).**
  `OperationCountForRoot` matches a `root:` prefix, but `ListFiles` records the
  bare root name, so a root-scoped `ListFiles` count is always 0 and the
  assertion passes vacuously. Mutation-check every new "count == 0"
  assertion: revert the fix and watch the test fail.
- **A new seam method on the real runtime gets its own conformance test (#420
  BR-1).** Adding `Runtime.StatFile` with only the fake pinning its contract
  left the `os.ErrNotExist` branch untested against the OS. Callers depended
  on that branch. When a seam gains a method, add a temp-dir test of the real
  implementation, compared against the existing method it mirrors.
- **A degraded mode ends only through the operation that repairs it (#430).**
  The first stall design let any successful probe clear `stalled`, including
  one inside a clipboard copy. The copy then got through and the screen kept
  the abandoned half-frame, because the repaint retry only fires while
  stalled. When a flag stands for "this needs repair", clear it only where the
  repair completes (here, a full paint), and test a side path that gets
  through first.
- **Dropping output is not dropping the event (#430).** Skipping a stalled
  write also skipped `EmitEffects`' bookkeeping, so attention marks for bells
  that write nothing were lost. When a mode suppresses I/O, suppress only the
  bytes; sequence numbers, "delivered" lists and caller-visible state proceed.
- **Classify a timeout by which deadline was binding, not by `ctx.Err()` (#430).**
  A poll can return DeadlineExceeded slightly before the caller's context
  reports `Err()`, so "caller still live" raced. Compare the deadlines up front
  (`ctx.Deadline()` vs. your own budget).

- **A red early CI step hides every later step (#324).** Step 1 of
  `couch-zellij-conformance` failed on every run for weeks, so step 2 never
  executed. It went stale under #355 unnoticed. When a step is red, check which
  later steps were `skipped`, and budget for what they show once it is fixed.

- **Reproduce a CI-only failure by matching the runner's *state*, not just its
  binaries (#324).** The same zellij 0.45.1 passed locally and failed on the
  runner, because a fresh HOME has no `~/.config/zellij`, which zellij treats
  as a first run. Run the failing command under `HOME=$(mktemp -d)` early, and
  have the job dump the tool's own log on failure.

- **A refusal's remedy must be reachable from the verb that refused (#437 BR-1).**
  `release` refused unread edits with "pass --accept-edits", a flag only
  `write` took. When several verbs share a guard, share the acknowledgement
  type too (one `Accepts` struct), and test each verb's refusal by following
  its own instruction.
- **Validate an address with the owner's grammar, never a regex copy (#437 BR-2).**
  A narrower slot regex turned one valid-but-unusual family into a store-wide
  outage. Call the parser that defines the address (`ParseWorkspaceReference`)
  and escape it into filenames rather than restricting it.
- **Degraded state clears only by repair, and a restart is not a repair (#437 BR-3).**
  A lost ledger row stays lost across restarts, so a restart-cleared failure flag
  turns "incomplete" back into "complete". Scope the record to what it taints
  (the slot), persist it, and clear it by explicit acknowledgement or expiry.
- **A paused clock resumes only on evidence that can judge the pause (#435).**
  The first version of the operator hold closed on every dispatch that did
  not see the hold, including the input-settle wait that typing the draft
  causes. That restarted the paste-by clock mid-draft. When a state stops or
  starts a timer, let only the observation that can decide it (here, a full
  composer reading) change it; transient waits leave it as is. Test the side
  path (keystrokes during the hold) that reaches the state machine first.

- **A suite that only passes in a clean env is not green for the agents who run it (#436).** Workers run `make test`
  inside live Couch slots, which export `COUCH_*`/`PAIR_*`. A test that isolates `HOME`/`PAIR_DATA_DIR` goes stale
  the day production starts reading a new variable. When a launcher learns a new env input, grep the tests that
  isolate its siblings and set the new one too, the way production hands it to the child.
- **Releasing a `flock` while another goroutine forks is not a release (#436).** The forked child holds a copy of
  every descriptor, CLOEXEC included, until its exec closes it, so a `LOCK_NB` retake right after `Close` can fail
  for a few ms. Retake with a short bounded settle, and test it with a `dup`ed descriptor closed late.
- **A wall-clock budget in the default lane measures the host (#436).** On a shared machine it flakes with load.
  Run the scenario in the default lane, log the time, and assert the budget in an explicit perf lane.

- **A rule proven in the pure model is not proven on the request path (#433 BR-1).** The self-addressed refill
  passed its model test, but nothing showed the broker would route a send to the sender's own slot, reserve on
  its own endpoint and admit it. When a model rule depends on an input only one caller path produces, add one
  test through that path (here `Broker.Send` from a binding to itself), and mutation-check it.
- **Glue that reads process state gets a seam (#433 BR-2).** The hub came from `os.Getwd` + tree resolution
  inline in `startMessageService`, so no test ran it. Extract a function taking the readers
  (`hubWorktree(getwd, resolve)`) and test it in a checkout, in a subdirectory, and outside any checkout.
- **A re-check before an effect uses the gate's own predicate (#434).** The
  prepare recheck read `git status --untracked-files=no`, while the reload's
  admission (`ProbeSlotGit`) counts untracked files as dirty. A refresh that
  left an untracked file would have passed the recheck and then been refused
  by the reload, with the wrong code and fix. When a step pre-checks a
  condition another gate will judge, call that gate's probe and parser. Do not
  write a second definition of the same predicate.
- **Scrub by owner, not by prefix (#436).** Dropping every inherited `PAIR_*` from test recipes also dropped the
  operator's opt-in knobs (`PAIR_PROBE_SETTLE`, `PAIR_LIVE_*`), which share the prefix. Name the caller-owned
  families explicitly, and derive the regression check's knob list from the docs, so a new knob is covered without
  anyone remembering to add it.
- **A retired verb that still takes an effect's flag must fail when it is passed (#439).** `release` became a no-op
  but still accepted `--fold`, so an old skill would see exit 0 and believe its events were folded. When a verb is
  kept only for compatibility, exit 0 for the arguments it can honor as a no-op and refuse those whose effect it
  silently drops, naming the replacement.
- **Validate before creating companions (#439).** The flock helper created `.<file>.flock` before checking the file
  was a session-sync file, so a wrong path left residue. Run the cheap validity check first, then create sidecars.
- **A new exception to a rule the atlas states is new surface (#441).** The ledger skip looked like a one-line
  bugfix, so close ran with `--no-atlas`, but `atlas/couch.md` enumerates exactly what the ledger skips, and the
  prose went stale. Before passing `--no-atlas`, grep the atlas for the function or rule you changed; if it lists
  the cases, add yours.
- **An unlocked check before a lock can read a torn file (#444, qualifies #439).** The #439 pre-flock validation read
  the file without the lock while another writer rewrote it in place, so the loser refused a valid file instead of
  reporting `stale`. A check done outside the lock may only gate side effects such as creating sidecars. Skip it
  when the sidecar already exists, and when it fails, let the read under the lock decide.
- **Re-measure a constraint a comment states before you design around it (#445).** "In place, not rename: parley
  watches this file's inode" kept the session-sync write non-atomic, but parley polls the path's mtime and nvim
  compares mtime and size, so a rename was always visible to both. A comment that justifies a weaker mechanism by
  pointing at another component's behavior goes stale when that component changes. Read the other side's code (or
  probe it live) before keeping the weaker choice, and when replacing a file someone else edits, resolve symlinks and
  keep its mode, since a rename changes both.
- **A file whose mtime is another rule's fallback is not free to rewrite (#448).** Making `read --all` refresh the
  session-sync shadow looked like a pure cache update, but a sidecar-less file's owed horizon is the shadow's mtime,
  so the first refresh would have silently un-owed every earlier delivery. Before adding a writer to a file, grep for
  readers of its metadata (`Stat`, `ModTime`), not just its content, and pin whatever they derive first.
- **A change to what a verb shows is a change to the agent skill that explains it (#448).** The shadow fix made
  `diff` mean "edits you haven't seen", while `couch/SKILL.md` still said "since your last write". The review caught
  it; grep the shipped skills for the verb whenever its output's meaning moves.
- **Overlay controls share one layout container (#446).** The pointer page's swatches and hint were each
  `position: fixed` at their own corner, so they covered terminal cells and, at phone width, each other. Put
  every fixed control in one wrapping band and size the content area from the band's measured height, then
  check it at 390px and 300px in an iframe, because headless Chrome won't open a window under about 500px.
- **A privacy test that bans an API gets narrowed, not deleted (#446).** The viewer test forbade
  `localStorage` because a view link keeps nothing; the color pick needs one key. Pin the exception exactly:
  one key, one reader and one writer, so a later "just remember this too" still fails the test.
- **A callback a verb triggers can run on the caller's own loop (#447).** `Config.OnNotes` looked like a
  request-goroutine callback, so the console's handler waited on the Run loop, but a 👆 click runs on that loop
  and `DisablePointer` now reports the emptied board: a deadlock no plan review caught, found by the first console
  test that clicked. Before a callback waits on a loop, list every caller of the function that fires it, the
  operator's own input path included, and hand off without waiting when one of them is that loop.
- **An overlay that writes text must widen the row's used columns (#447).** The renderer draws a row's text only
  up to `RowMetadata.UsedColumns` and treats the rest as blank tail, so a 💭 on an empty row vanished while marks
  (background only) showed. A frame edit that adds content past the old text updates the metadata too.
- **A user-visible addition to the tab bar is a README change (#447).** The M1 review blocked on it; grep the
  README for the row's current spelling (`LIVE ⏸ 👆` since #503) whenever the row gains a cell.
- **Rule out the host before bisecting a "regression" (#450).** Four workers called `test-review` a regression since
  #436 and the issue asked for a bisect, but the failure reproduced at #436's own merge: they ran the suite inside the
  agent sandbox, which refuses unix sockets and PTYs, while #436 was verified outside it. Before bisecting, rerun the
  failing test at the "known-good" commit in the same shell; if it fails there too, the variable is the environment.
  A suite that cannot run on a host should say so first (`tests/host-preflight.sh`), not fail 150 tests that read
  like code bugs.
- **A test's hand-pinned env list rots when production gains a new env input (#450).** The launcher tests pinned
  `COUCH_THREAD_*` but not `COUCH_PAIR_DATA_DIR`, added later by #366, so raw `go test ./...` in a Couch shell
  failed. Clear an ambient input that changes every call's behaviour once per package, in `TestMain`, rather than
  in each test; a child process built from `os.Environ()` needs the same pin in its env builder, since `TestMain`
  does not reach a package that only launches the binary.
- **A screen scan for "the dialog" reads from the bottom and keys on the selection cursor (#451).** `dialogTitle`
  took the first `1.` line from the top, so a numbered list in the agent's earlier prose became the dialog's title.
  A live dialog is the newest thing on screen and its selected option carries the cursor glyph; prose lists don't.
  When classifying a terminal tail, anchor on the most recent, most specific marker, and test with distractor
  text above it.
- **A retry-on-failure path needs a test that fails once, then succeeds (#451).** Marking an episode notified
  before `Send` kept every test green and silently dropped notices after one refusal. When a state update must
  follow an effect's success, add a fake that refuses the first attempt, and mutation-check the ordering.
- **An operator knob that fails to parse refuses, it doesn't default (#451).** A silent fallback hides a typo for
  as long as the default is plausible. Validate env knobs before taking resources and refuse the start, naming the
  variable and the accepted form.

- **A stream's end resets everything the stream gave the page, not just the screen (#447 M2 review).** The
  viewer's `end` handler reset the terminal and left notes tappable, Resolve live and comment mode drafting on a
  dead broadcast, because the server's `select` sends end, caps and notes in no fixed order. When a stream feeds
  several derived states, its terminal event puts each back to that page variant's join state through the same
  handlers that set them — never replaying an event the variant doesn't receive (a view page never hears caps) —
  and a test pins each variant.
- **An async answer names the request it answers (#447 M2 review).** The note dialog applied a save's 204 to
  whatever dialog was open when it arrived, so cancel-then-new-draft lost the new text. Tag each open with a
  generation and drop answers from an older one.
- **Test a timer's teardown at once, not eventually (#447 M3 review).** A stale self-rescheduling timer clears
  itself when it next fires, so `waitFor(timer == nil)` passed with the teardown deleted; only an immediate
  assertion after the teardown step caught it. For every timer the console arms, test that it re-arms after firing
  and is gone *immediately* after each way its owner ends, and mutation-check both (drop the reschedule, drop the
  stop).
- **State the envelope at the bound, not per unit (#447 M3 review).** "About 11 repaints per note per 15s" hid
  35 repaints a second to every viewer at the 48-note limit. When a cost scales with a capped count, write the
  total at the cap, and prefer a design (one shared beat) whose cost doesn't scale at all.
- **A "never reaches the child" test needs a child that asks (#447 M4).** Couch forwards mouse reports only to a
  child that requested tracking, so asserting "no report reached the child" with a default fixture child passes
  whatever the routing does. Feed the child its tracking mode first, and pair the negative assertion with a
  positive control (a click outside the menu that does arrive).
- **A hit region lives and dies with its drawing (#447 M4 review).** The notes menu kept its click area from the
  last frame it drew, so under the switcher (a private frame, where it isn't drawn) it still took clicks and copied
  a note nobody could see. Record a hit-test layout only from the frame actually shown and clear it on every frame
  that doesn't show the thing; test it by covering the thing and clicking where it was.
- **Test the wiring, not the state field (#447 M4 review).** `TestNotesMenuOpenHoverCopy` asserted `pnotes.hover`
  was set, so dropping the hover from the drawing style left every test green. For each behavior a Done-when
  claims, assert its visible effect through the console (the drawn cells), with a control that differs only by the
  input, and mutation-check by deleting the wiring line. When the visible path can't show a difference (the
  presenter dropping a stray drag), assert the routing decision directly and say why.
- **A tuning change sweeps every sentence that states the old value (#447 close review).** Smoke tuning changed the
  flash constants and added right-click resolve; the commits updated atlas and Revisions but left the plan's Core
  concepts bullets (1.5s, 10 steps, 11 repaints) and a README sentence ("resolved from a viewer's page") stating the
  old behavior. Before committing a constant or capability change, grep the old number and the verb across README,
  atlas and the plan, and rewrite each hit; state tests in terms of the constants so tuning can't stale them.
- **An exit status is not a diagnosis (#454 M1 review).** Chrome exits 21 both when another Chrome holds the profile
  and when it merely can't create its lock directory (as in a sandbox), so mapping 21 to "profile in use" sent a
  sandboxed run down the wrong path. When classifying a child's failure, match the evidence that names the cause
  (its message), quote the output otherwise, and test with a look-alike failure.
- **A loop's terminal reason is a pure function of what it observed (#454 M1 review).** The Chrome loop called every
  unrequested end "closed by operator", hiding a broken connection. Collect the facts (who asked, the connection's
  error, the exit status) and derive the reason in one pure function with a precedence table, then test each row and
  drive the loop itself through a scripted peer so the facts are the real ones.
- **Classify failures at their source, by kind (#454 M1 review round 2).** Excluding `io.EOF` and `ErrClosed` from
  "connection failures" still let `os.ErrClosed` (our reaper closing the read end) and EPIPE through as "lost Chrome"
  — a deny-list of transport errors is never complete. Mark the class you mean where it originates (`cdp.ErrProtocol`
  for what the peer *sent*) and allow-list it; a pipe closing is what every ending looks like, including your own.
- **Reference counts release through one function (#454 M2 review).** The shared Host checked "nobody wants it"
  inline at open completion and at Unmount, and a cancelled waiter had its own `last` test; the waiter's test
  missed the interleaving where the open completed first, leaking a tunnel 1 time in 300. Every way interest ends
  (an open completing, a waiter giving up, an unmount) calls one `releaseIfUnwanted`; test the exact losing state by
  constructing it, plus a stress loop at the timing that reproduces it.
- **A summary figure is computed from its table, endpoints included (#453 close review).** The write-up said
  "~1.5× the gzipped bytes" over measured ratios of 1.15–2.05×, "at least as faithful on every page" where one
  page had more lagging moments, and "lowest overhead" measured on only one page. When a recommendation restates a
  measurement, quote the range from the table and name the scope it was measured on; don't round toward the
  option being recommended.
- **A new irreversible step joins the operation's ORDER, it doesn't ride admission (#456 close review).** Live
  reboot parked the agent inside `rebootAdmit`, ahead of the fresh-profile preflight, so an unlaunchable agent
  stopped the running one and started nothing while the error said "nothing was stopped". When an operation has
  a stated order (check everything that can fail, then the first irreversible effect), add a new effect at the
  effect boundary, keep admission effect-free, and test the earlier refusal with the agent still running —
  mutation-checked by moving the effect back.
- **A latch needs a close for every way its condition ends (#464).** `pickerActive` armed on Claude's permission
  OSC and cleared only on a plain Enter. Claude Code's unattended auto-deny closed the picker with no key, and the
  stale latch held every peer message "waiting for picker to close". When a flag tracks an external state, list
  each way that state ends (key, timeout, Esc, the other side's own decision) and clear the flag from observed
  evidence (here, the composer returning after the picker hid it), not from the one input you happen to see.
  Read the screen at arming too: the evidence can land before the signal that arms the flag.
- pair#463: a new reserved name on an existing addressing surface (`couch --send-to hub`) left README and atlas
  still describing the old address grammar, and a TL-deferred Done-when clause stayed as written. When a change
  adds a form to a CLI grammar, grep every restatement of that grammar (`--help`, skill, README, atlas) in the
  same commit; when scope moves elsewhere, amend `## Done when` and `## Plan` in the commit that moves it.
- **A new Go source file joins the artifact inventory in the same commit (#460).** Moving `FakeHub` into a new
  `recording/hub.go` failed `TestProductionArtifactReferencesAreExactlyClassified`, which only `make test` runs (the
  package's own tests passed). When a change adds a production `.go` file, classify it in
  `cmd/internal/artifactpath/manifest.go` and run `go test ./cmd/internal/artifactpath` before the close.
- **A state record a reader can start from mid-stream needs a baseline rule (#466).** `ctl` records carry the control
  state after them so a late joiner's catch-up keeps only the latest one, but the reducer checked each against the
  previous state, so the catch-up didn't reduce. When a record is meant to be self-contained for some readers and a
  checked transition for others, write the rule for both in one place (first one seen is the baseline if consistent
  in itself; a whole-recording reader stays strict) and run the catch-up check at every cut point.
- **An allow-list's test must not derive its expectation from the allow-list (#466 close, 3 rounds).** `Confine` lost
  four holes one round at a time (Meta+Y, Shift+Escape, Shift+Delete, modified Enter/Space) because each fix patched an
  instance, and the "exhaustive" test compared `Confine` with the table it calls. Write the rule as one closed table
  with its reason (no Chrome/Blink binding on any platform; an input's keyboard twin is refused with its pointer
  form), and test it against a hand-written expectation plus a cited list of the bindings it must exclude.
- **A contract changed in a fix round moves with every restatement (#460 close review).** BR-1 added a method to
  `recording.Live`, and the atlas still described the old behaviour on the next round. When a fix changes what a
  seam promises, grep the seam's name and its old wording across atlas, plan and doc comments, and rewrite each in
  the same commit. The same holds for a retracted figure (a size bound, a measurement): grep its distinctive wording
  ("1.4 MB") and correct every hit, the plan included, in the fix that retracts it. And the reason a channel closed
  is recorded by the source under its lock, not inferred by consumers from timing.
- **A shared corpus has to include the cases where the implementations differ (#459 close review).** The Go and JS
  CSS scanners were "held to one corpus", but the corpus left out comments, strings and name boundaries, exactly where
  a regex and a tokenizer part ways. When two implementations share a test corpus, write the cases each one's
  technique gets wrong first, and run the corpus against the old version to see it fail. And a per-call resource
  (an image decode at a header-checked ceiling) needs a bound on concurrent calls too, not just on each call.
- **A string two components read has one reader of record (#462 close, round 1).** The agent queries' CSS matcher
  also read the notes model's anchor selectors, by its own rules: `>>>` as a plain descendant and `>` crossing shadow
  and frame boundaries, so an anchor named 6 elements where the model named one, and the test only asked for
  containment. When a second component accepts another's format, call the owner's reader for it (`notes.Match`) and
  fall back to your own grammar only outside it; scope every combinator to a tree as the platform does; test with
  identity against an oracle the owner provides (`Resolve` with no node id), never with "contains".
- **A fixture corpus can leave out the very case a rule exists for (#471).** The catch-up rule keeps assets from before
  the latest segment and drops notes from earlier documents, and no fixture had either. The invariant passed on every
  fixture while exercising nothing. When a port is held to an invariant over fixtures, list the rule's branches first,
  then check that some input reaches each one; add a synthetic recording for any branch no fixture reaches. And look at
  a UI once (a headless screenshot): a layout bug that only shows while paused passed every DOM check.
- **A held port needs a corpus the owner's test regenerates (#471 close, round 1).** #458 changed `recording.CatchUp`
  (hover and ptr: only the latest) while #471's JS port was in flight. The merge was clean and every test passed: no
  fixture had a hover or ptr, and the record-for-record check ran only in live Chrome. When code ports another
  package's rule, check it in as a shared corpus. It needs every branch, plus the owner's answers. The owner's test
  must fail when its rule moves (`TestCatchUpCorpus`), and the port's test must fail until it follows. After merging
  main, grep main's diff for the ported function's name.
- **A "what the user sees" rule has one walker; grep for its siblings (#469 close).** Fixing note context with a new
  `RenderedText` left #462's `spacedText` in `query/render.go`, a second visible-text walker with different
  rules beside it in the same output. When you add a function that defines how something reads (rendered text, a
  selector, a label), grep the codebase for every other function doing the same job (`TextContent`, `textContent`,
  "what a reader sees") and route them through yours in the same change. And don't derive state the source doesn't
  hold: a select's live choice isn't in the DOM, so guessing it from the `selected` attribute contradicted the
  event beside it.

- **A check made before handing a record to an asynchronous writer is re-made by the writer (#470 plan-quality).** The
  board checked a viewer's note against the current document, then emitted it outside its lock into the recorder's
  queue; a navigation ordered in between would have filed the note on the new page, because the reducer gives a note
  the segment current when it is *reduced*. When a reader of a stream validates against state and a different
  writer orders the result, send the checked fact with the record (here the document's first segment) and have the
  writer apply the one statement of the rule (`recording.State.OnDoc`) at ordering time; test the race by ordering
  the competing event inside the emit.
- **Fixtures that share an accident hide every reader that depends on it (#472).** The prototype's fixtures were all
  1280×800 with frames taken while the stream was quiet, so the replay harness and the fidelity tool hard-coded the
  viewport, and the recorder's clamp of a frame's time to the stream's (43–87 ms late on a busy page, against the
  format's "midpoint") never showed. When a reader takes a value the recording carries (a viewport, a time), read it
  from the record, not from what the fixtures happen to share; and when a clamp keeps one stream monotonic, apply it
  only to that stream's records, never to a timestamp another file defines. Add a fixture that varies the shared thing.
- **A test that asserts "at once" needs a sync point with the goroutine doing the work (#473).** Two flakes had the
  same shape: the test read state from its own goroutine while another one was mid-step. couchtty polled the pointer
  phase, which flips before the same Run-loop step tears the timers down; couchcore gave a fake child's reaper 2ms of
  real time after its death to deliver. Before writing an exact assertion (`timer == nil`, `kills == 3`), name the
  event that makes it true and wait on that event: a no-op command through the loop that owns the state, or an
  injected clock (`Couch.after`) that ties the timeout to the fake's state. To prove a suspected window, widen it
  with a temporary sleep in the product or fake and watch the test fail every time.
- **A flag that marks setup done must die with what it set up (#475).** `openDoc` guarded its load listener with a
  property on the document, but `document.open()` erases the document's listeners and keeps its properties, so only
  the first page a frame ever drew redrew on loads, and the replay's numbers depended on test order. When a guard
  says "already done", check that every operation that undoes the work also clears the guard; prefer re-doing an
  idempotent setup each time over a flag. And when a render number changes only with run order, diff the same
  capture twice before suspecting the change under test (Chrome throttles a hidden frame's rendering too).
- **A production bound a test inherits is a real-time window too (#476).** `TestCapturePTYBothBoundaries` never named
  a duration. It failed because `Recorder.Close`'s production 2s drain bound is the default it ran with. And the
  retention lock test timed a whole call, file I/O included, to prove it "didn't block", although the held lock
  already proved that. Before asserting on a call that has a bound (a drain, a deadline, a poll), ask whether the
  bound is the test's subject. If not, lift it through a config seam (`Config.CloseTimeout`) and wait on the event.
  If the state the test holds already proves the property, wait with no window and let the binary's timeout catch
  a hang.
- **Nothing that takes input is rebuilt on the per-frame path (#470 M2, two review rounds).** The notes UI redrew its
  pins and then its swatches with `replaceChildren` every animation frame, so a tap that spanned a frame landed on a
  removed element. Frame work only moves and restyles; elements are created and removed when their data changes (keep
  them in a map by id). Test with a human-length press (100 ms): a back-to-back synthetic press and release fits in one
  frame and lets the per-frame mutant pass.
- **An async mount starts from the state it mounts into (#470 M2).** The notes module mounted after a fetch, began
  with "no document", and took the first record it saw for a navigation, ending the user's stroke after one chunk (4 of
  6 runs). A late joiner of a stream initializes from the model's current values, not from zero; and a flaky e2e is a
  bug to root-cause with state dumps from both sides, not a retry.
- **A test writes loop-owned state on the loop, or through the real producer (#470 close, BR-25).** A Couch test set
  the web-notes menu state directly while the cast's delivery goroutine wrote its first snapshot, failing 10 of 20
  runs under `-race` and 0 of 50 without. When state has a single-writer rule, the test drives the real producer (post
  through the board, wait for delivery) and makes any other change inside `runTerminalCommand`; run the new test with
  `-race -count=20` before calling it done.
- **Every server of a page serves every module the page imports (#470 meets #471).** The notes module imported
  `collab.js`, which only the live server served; the replay server, landed meanwhile, served the same viewer from the
  shared file table, and one missing ES module kills the whole module graph. Serve a page's files from one table both
  servers use, and test the closure: walk the imports from the entry module and fetch each from that table. After a
  merge that brings a second consumer of shared files, run the live (browser) suites, not only the unit ones.

- **A snapshot of a running page is not a fresh load (#478).** The fidelity replay started the blog's entry fade
  when it drew the go-live snapshot, but the tab had finished it seconds before the cast went live, so the first
  frame scored 2.15% for no render fault. When a replay or viewer draws a state captured mid-life (a segment that
  isn't a navigation, a catch-up), ask what the browser's own clocks (animations, transitions) were doing at
  capture, and settle them to the likeliest phase. Look at the worst moment's crop before tuning a tolerance: the
  picture named the cause where the number couldn't.

- **A surface that lets an agent open something public needs the operator to see it, and every surface gets swept
  (#477).** `query link` let an agent open the broadcast tunnel with nothing shown to the operator. Round 1 fixed that
  (a notice, and the ids in `--broadcast-list`), but left the README saying the listing never had more, and pinned
  the listing with a test while the notice had none. When an action can be taken by an agent and is visible to people
  outside the machine, ask, at design time, how the operator learns it happened. When you add a verb, flag, output
  line or JSON key, grep its name across the README (synopsis and prose), `couch --help`, the atlas, SKILL.md and the
  Usage text in the same commit. Give each behavior a fix adds its own test that fails without it.

- **A paused view draws no frames, so every event that changes what it shows must draw (#477 M2).** The replay
  redraws its marks once per frame, and a paused replay (how every `#t=` link opens) has none. A tap's note card
  and a resize waited for a frame that never came. When a view redraws on a loop, list what changes its output
  besides the loop (input, resize, a seek) and draw from each. Test with the loop stopped, after a fixed number of
  frames, not by polling: a poll can be satisfied by a later, unrelated redraw (the first resize test passed with
  the fix removed).
- **A test races a timer it doesn't own; give it the timer (#480).** Thirteen tests raced real 15ms-3s timers or
  slept before asserting. Where the timer is the subject, the product takes it through a seam whose nil is the wall
  clock (`loopTimer` for wrap's idle floor and ESC flush, `Couch.registrationDeadline`, `LaunchOptions.BirthClock`),
  and the test advances or spends it. Where it isn't, wait on the event a sleep stood for (admissions counted at
  Reserve, a semaphore's `len`, `Close` joining a watcher). A causal oracle on the test's own timestamps is not a
  substitute: it fails when the loop, not the writer, stalls. And read the product's order before asserting a second
  effect after a first: the stdin loop writes a chord before disarming its flush, so "disarmed" is its own wait (the
  soak caught one in 200).
- **A component that stacks its own elements must not reach past them (#475 close, round 1).** A `z-index` on the
  held frame lifted it over the shell's overlay, which sits beside it with `z-index: auto`. Order layers by document
  order inside the component, or give the component its own stacking context; never set a z-index the host doesn't
  know about. And keep a lifecycle in one state value (`pair.hold = null | {owner, drawn, cancel}`), not in flags,
  an inline style and a shared object; inject its clock so a test can drive each ordering.
- **A test moved to another fixture keeps its old fixture's moments unless they come from the data (#474).** The
  replay marks test moved from the prototype's `app` to `rec-app` and kept sampling `app`'s note times, so it still
  passed but checked nothing about the new notes. Derive sample moments (and any content a test reads) from the
  fixture's own records; re-deriving then surfaced a case the old fixture never had (an open note whose element was
  removed). After retiring a data source, also sweep the tools that only produced it (`fixturegen -raw`) and any
  inventory that lists them.

- **A client of a wire contract sends through one gate that holds every server limit (#467 M2).** The viewer's
  drive shell capped acts and text runes but not body bytes, didn't clamp coordinates, and didn't pace. A paste over
  8 KiB, or a drag captured past the frame's edge, was refused whole, and with it the button's up. When a server
  parses strictly (sizes, ranges, rates), route everything the client sends through one function that applies all
  of those rules (`batcher`: `wireAct`, `splitText`, a byte bound, pacing). Property-test it against the server's
  constants: every sent body parses.
- **The side that outlives an exit ends what was started (#467 M2, three rounds in one family).** A driver's
  pressed button or held key was ended only by the driver's own later up. That up was lost when control moved, when
  a refusal dropped the queue, when the page changed, and when a late release ran after the next driver's press.
  When input has a down/up pairing, the component that survives every exit (here the cast tab's injector) tracks
  what is held, keyed by identity, and releases it on each exit path. A release names the hold it ends.
- **A real-browser conformance run finds what fakes agree on (#467 M1).** `DOM.getNodeForLocation` with
  `includeUserAgentShadowDOM` returns an `<input>`'s inner editor, which no recording has, so every click on an
  input read as stale. The fake injector hit-tests its own boxes and could never show it. A seam that maps between
  a browser's ids and a model's ids needs one live run before its fake is trusted.
- **A user-facing gesture, flag, notice or output format lands with its README line (#467 M3).** The drive span, the
  Ctrl+click take-back, the Alt+right-click drive link and the new `--broadcast-list` shares reached the atlas but
  not the README, which still described the old flag output and "the viewer link". The atlas maps code for agents;
  the README is what the operator reads. Update both in the commit that adds the surface, not at close.
- **A stopwatch ceiling hides the race it was meant to catch (#479).** `elapsed > 1.5s` passed a TERM-deaf-child
  test whose cancel landed before `sh` set its trap, so the child died by SIGTERM and SIGKILL escalation was never
  exercised; only asserting *how* the child ended (its signal) exposed it under load. Where a test's subject is a
  bound, assert the outcome that proves it (the product's own timeout error, the ending signal, the busy refusal),
  not the wall-clock time; where the subject is an event, wait on it with `testwait` (bounded by the test's
  deadline, not a latency guess). A short real deadline the test arms for every case is the same bug: arm it only
  where the timeout is the subject.
- **A capture at t waits for every asset bound by t, not one kind (#482).** The fidelity replay captured right after
  the real-time window and waited only for a held paint's stylesheets. A font and an avatar that arrived 1 ms before
  rec-blog's first frame were still loading in about 1 run in 8 under load. The diagnosis named the font, a second
  slot's crops named the image, and both were one cause. When a capture flakes, look at the worst crop before fixing
  a cause. Wait on the general readiness (sheets, fonts, images, bounded) after stopping the clocks at t, and pin it
  with an asset bound just before the moment, which fails deterministically, not with a soak.
- **A new input path inherits the channel's validation; apply its policy to the whole input, not its edges (#486
  close).** `--message-file` stripped one trailing newline, then a trailing CRLF, and each round found the next CR:
  the broker refuses every control byte but `\n` and `\t`, so a CRLF file fails wherever its CRs sit, with an error
  that doesn't name the file. When an input source feeds an existing validator, read that validator first and decide
  one rule for the whole input (normalize it, or refuse it with the source named), not a fix at the site a reviewer saw.
- **A retry loop owns each attempt's resource, not a variable the defer closes (#465 close review).** The hello
  fallback reassigned the deferred `conn` from a redial, so a failed redial left it nil and the deferred `Close`
  crashed the wrapper. Give each attempt its own local, close it on failure inside the loop, and hand only the
  accepted one to the defer; test the path where a retry's dial fails. Likewise, key a per-episode cache on an
  episode counter bumped at the transition, not on a reader happening to observe the gap between episodes.
- **A launch's death is its direct child's exit, not its process group's (#489 close).** The registration wait
  first watched `Alive` (kill(-pgid,0)), but a Couch-launched Pair's title poller stays in that group for up to its
  30 s startup grace, so a Pair that died at birth still held the wait open. The fake's `SetExited` ended the whole
  group and could not show it. When "did it die?" decides a wait, ask the process you started (`Handle.Exited`), keep
  group liveness for cleanup, and give the fake a member that outlives the child.
- **Every fixed window over load-dependent work moves with the load (#489).** Removing the 15 s registration budget
  exposed the CLI's 2 s per-poll exchange timeout, which turned one late status answer into an "uncertain" outcome
  while the operation went on. When fixing one stopwatch, grep the path end to end (launch, poll, report) for the
  others; retry an idempotent read inside the outer budget instead of giving up on one slow answer.
- **A measurement that moves a Spec's guess revises the Spec in the same commit (#490 close).** The Spec guessed
  `--prepare` and `--reload-context` at "seconds" and light at "milliseconds"; the timings said heavy and "under a
  second", and only the Log said so. When a Spec names expected values to be measured, append a `## Revisions`
  entry the moment a measurement disagrees, and say in Done-when which values were reasoned rather than measured.
- **Describe a mechanism once; every other site names it (#489 close, two rounds).** The registration wait's exit
  signal was restated in four comments and docs, so changing it (process group → direct child) left three stale
  copies the review found one round apart. Put the rule at its owner (`whileHelperAlive` + one atlas section) and
  have other sites point there. A status line likewise derives what it reports from one fact (when Couch last
  answered), not a counter that a second code path must remember to reset. And a fallback branch (the final look
  after an exit) needs a test that fails with the branch disabled, run once with it disabled to prove it.
- **Every path that unblocks a worker wakes it (#492 close).** The queue's worker skipped promotion while another
  admission attempt held the actor's reservation; that attempt's failure path cleared the reservation and woke no
  one, so a queued message sat until the test deadline. When a worker parks on a condition (a reservation, a
  pending delivery), route every release of that condition through one helper that also signals the worker, and
  test the interleaving deterministically: count the worker's blocked checks and wait on the count, never on a sleep.
  The converse is the same rule's other half: the worker never wakes itself. Its own failed attempt schedules its
  retry, or it spins (round 2's Critical). A fake must refuse the way the real seam refuses (`ErrRecipientBusy`,
  not a bare error), or the tests never exercise the path the product takes.
- **A fixture the product identifies by command must not exec away from that command (#487).** The fake cloudflared
  ran `exec sleep 300`, so its ps command became `sleep 300` a moment after start, and reaping (correctly) spares a
  non-cloudflared process. The test passed only while ps beat the exec, and under load it lost. When a fake stands in
  for a process the code checks by command, PID or identity, wait for the fixture's settled state and assert the
  property there, not the first moment it holds. Wait on the outcome with `testwait`, never a fixed window.
- **A fix round that changes a behaviour rewrites every prose restatement of it (#498 close).** Round 1 changed a
  refused `setpriority` from "cancel the launch" to "keep the inherited niceness"; the code comment, README and Design
  moved, but the atlas, the Plan checklist and a Log line still said "cancels", and round 2 blocked on it. Before
  re-running a close, grep the old behaviour's words (`cancel`, `refuse`, the error text) across atlas/, README,
  the issue file and comments, and rewrite or annotate as superseded every hit in the same commit.

- **A file read while it is written has a torn tail (#493 close).** Making the recording's writer unbuffered let
  readers see records mid-cast, and with that the reader's strict decode (any bad line fails the whole recording)
  became a live hazard: a replay racing a write could fail outright. When a writer's output becomes readable while
  open, the reader takes complete lines only and leaves torn-tail detection to the validator of a finished file.
- **"Disable every action" means listing every surface first (#499 close).** The pause guard disabled the rail's
  pen, swatches and 💭 but left the note dialog's Save and Resolve live, including on a pin opened during the pause;
  only the server's refusal caught them. Before a mode turns user actions off, enumerate every control that posts
  (rail, dialog, pins, keyboard), gate them at one place, and put each in the test.

## A count or abbreviation must be anchored to a drawn member of its own group (pair#491)
- **What went wrong:** the tab bar's narrow-width collapse hid every numbered slot of a group, including its first member when the group had no `:0`. The `+N` then sat after the previous repository's chip and read as its count.
- **Rule:** when compressing a run of related items into a summary (`+N`, `:N`, `…`), keep the run's first item drawn so the summary has an owner on screen; test a run that lacks its usual leader.
- **And:** state a value handed across a package boundary at startup (here the hub root) is tested where it is set *and* where a lazy rebuild could drop it.
- **A variant of a read path wraps the original, never copies it (#507 close).** `getThreadStamped` repeated
  `GetThread`'s validate, route, lock, read and clone steps and dropped its `AfterGetThread` hook, which recovery
  tests use to inject interleavings. When a caller needs the same read plus one more fact (a stamp, a path), make
  the old function call the new one so hooks and checks live in one place. A cache that serves a parsed value
  hands each caller its own copy of anything the caller can reach and mutate, and keys on every input that names
  the source (a root's path, not only its fixed name).
- **An error translation is behavior; each needs a test that fails without it (#505 close).** Mapping `errClosed` to
  `ErrClosed`, a CDP failure to `ErrInput` and a nil recorder to `ErrNotLive` were each one line, and reverting any
  left the suite green: the route test injected a ready-made `ErrInput`. Test the translation at the seam that
  produces it (the recorder's fake tab, the nil-recorder Chrome), and mutation-check it once.
- **A browser event's field is a measurement, not an assumption (#505).** drive.js counted double clicks from
  `pointerup.detail`, and its node tests fed fake events with `detail: 2`, which real Chrome never sends (pointer
  events carry 0). Before a gesture rule reads an event field, log that field from a real browser once, and make
  the fakes match it.

- **A viewer-side rule that stands in for the browser must act like the browser everywhere (#513 close).** The
  live-size rule meant to replace a canvas's UA sizing was author-origin, so it beat the UA's `[hidden]`, and it
  lived in the document sheet, which shadow trees don't see. When a viewer rule substitutes for browser behaviour,
  list the browser's own conditions (hidden, display:none, shadow trees, frames) and make each a test case.
- **Changing a rule means grepping for every statement of it (#516 close; #489's lesson again).** The wheel's stale
  rule was restated in `Act`'s doc, `ErrStale`'s doc and `onTarget`'s comment; updating `Injector.Inject` and the
  atlas left three stale copies. Before committing a rule change, `rg` the rule's nouns (here "stale", "wheel")
  across the package and the atlas, and make every other site name the rule's owner instead of restating it.

- **An incarnation does not identify a request (#517 plan review).** When an
  asynchronous callback can outlive cancellation and retry on the same source,
  carry both the source incarnation and the parent request ID end to end.
  A late reply must never satisfy the next request merely because the process
  stayed alive. Distinguish standalone commands from callbacks explicitly.

- **Retained evidence belongs to a source, not a lifecycle label (#517 M1).**
  A previous-session descriptor survived A→B correctly, but a separate park of B
  followed by B→C reused A's evidence because the source was already parked.
  Compare the saved source identity with the current verified park before reuse;
  test both same-source retries and a different source parked through another path.
  Test retention through the new reader itself: collect while its lease is held,
  then prove release after both successful replay and an error.
- **A plan's owner map follows the actual diff (#517 M1).** When implementation
  splits a concern into a new file, check every core-concepts location and the
  milestone file inventory against the diff; record the correction in Revisions.
- **A full read must detect every capacity loss in its dependency (#517 M1).**
  Raising the emulator's row cap left its cell and byte caps silently evicting
  history, and accepting 1000×1000 geometry exceeded its screen-area limit.
  Read the dependency's complete limit model; use checked construction/resize
  and an explicit loss signal that survives later clears. Test each independent
  limit and loss followed by erase in one input, not only a row-count overflow.
- #494 visual review: when mounting an existing viewer control inside another
  surface, reuse its glyphs, tooltips and dimensions as well as its handlers.
  Text labels substituted into icon-sized buttons can clip and make the two
  surfaces needlessly different; inspect the actual embedded rail before handoff.
## Re-meaning a named surface: sweep for the old term, and sweep again after every merge (pair#506)
- **What went wrong:** #506 split the web cast's "view link" into view-only and pointer links. Three review rounds flagged the old term: first in `atlas/webcast-format.md` and the README prose, then in code comments, then in a README table that #495's merge brought in after the sweep.
- **Rule:** when a change gives an existing name new grants or meaning (a link, a mode, a flag), `git grep` the old term and its synonyms ("view link", "viewer link", "two links") across README.md (tables too), atlas/, production comments and test comments before committing, and fix or date every hit. Re-run that grep after every merge from main: a branch that lands second inherits the other side's uses of the old term.
- **And:** when several places map a kind to a value (a link label to its URL), give the type one accessor (`Update.LinkFor`) before the third copy appears; the copies drift (one unlock helper sent the pointer code to the view-only URL).

### #483 — asynchronous submission ownership

- A scheduler may execute inline during startup/shutdown: release the owner's mutex before invoking it, and test the inline case.
- Cancellation must gate the first side effect as well as publication; a canceled job must not start replay, screenshot capture or note resolution.
- Retry identity includes generated evidence and the recipient, not only authored text. After confirmed dispatch, remaining retries must be acknowledgment/commit-only.

- Idempotent durable writes must retry directory synchronization after a successful rename followed by failed sync. Test repeated retries through a newly opened store, including callers that advance directly to the next transition. Visible files are not proof of durable publication.
- An injected adapter is still integration code: pure state/event transitions own retry state; the adapter interprets effects and ignores stale completions by attempt identity.
- Reuse a shared recipient identity/parser when adding a routing consumer; keep each caller's uniqueness and incarnation policy separate.
- Visible evidence needs a valid, nonempty effective intersection, including ancestor clips. Share that predicate across text and accessibility entries; test wholly off-screen regions and clipped descendants in both the DOM fixture and a real browser.

- pair#523 plan review: retryable cleanup must retain evidence until the external effect is acknowledged. Audit every caller of a reused release helper; clearing local state before CDP replies makes a later retry falsely succeed. Uncertain delivery needs an explicit terminal recovery path.
- pair#523 plan review: describe tests by production function, adversarial input class and mechanical guard. Keep acceptance behavior in the spec; executable case inventories belong in tests.
- pair#523 reducer review: test each stale-completion guard independently. An old completion delivered only after failure exercises the phase guard, not the attempt guard; deliver it while the newer attempt is Running and mutation-check that removing attempt equality fails.
- pair#523 cleanup review: a caller classifying partial sink writes as terminal must wrap lower-layer errors with its own terminal outcome. Propagating an inner Retryable misclassifies an effect the outer operation may already have committed.

- #523: Canceling an HTTP caller does not cancel ownership of a started side effect.
  Retain its settled failure for the authority; distinguish an unstarted queued
  callback from uncertain external delivery with typed evidence.
- #523: Fan-out records are shared read-only values. Encoders must stamp copies,
  never mutate an envelope while concurrent subscribers serialize it.

- #523: A cancellation can finish the waiter before the operation. Preserve an
  unobserved terminal outcome beyond the busy channel lifetime and consume it
  before a later retry declares cleanup complete; do not re-report an outcome
  already surfaced to the operator.

- #523 boundary review: a new recording event needs a consumer inventory covering
  both reducers and replay optimizers. Compare marks as well as DOM at every
  prefix and across seeks; DOM-only parity can miss resurrected annotations.
- #523 boundary review: verify acknowledgments through the production adapter.
  A synchronous fake cannot prove an asynchronous recorder committed a write.
- **Measure screenshot side effects before adding waits (#524).** CDP captureBeyondViewport restarted an already-finished responsive fade by changing render viewport state; waiting longer before capture could never fix it. Compare animation identity and pixels before/after the actual capture, not only readiness before it. A browser animation enumeration is root-scoped: real Chrome excluded open-shadow animations from document.getAnimations, so test the scope with a real shadow fixture before trusting a fake.

- Animation target bounds do not bound its painted influence: inherited opacity/filter/transform can affect overflowing children and pseudo-elements. Prove exclusion conservatively and test zero-size/off-box ancestors with real crop pixels.
- **A lifecycle guard checks every representation of its authority (#517 M2).**
  Before registration, a replacement profile lives in Start.LaunchProfile;
  after registration or unknown promotion it lives on the incarnation. A guard
  reading only the latter can erase a surviving target during cleanup. Enumerate
  interruption points, exercise startup reconciliation then the user action,
  and assert identities as well as errors and launch counts.
- **Audit the whole active owner table against a named baseline (#517 M2).**
  A scope cut moved missing-config policy into switchagent.go while the plan
  still claimed launchprofile.go changed. For each active core-concepts row,
  check the actual owner, pure/integration class and new/modified/reused status
  with git diff; append a replacement table and explicitly supersede old rows.

- **Pure selectors need value-only tests even when integration covers them (#517 M2).**
  Construct records directly for projection, validation and defensive-copy
  cases; a filesystem-backed Couch fixture proves integration, not the pure
  seam's independent contract.
### #494 — publish measured commit identifiers

- A smoke handoff accidentally sent a placeholder commit identifier. Read `git rev-parse --short HEAD` after the commit, then copy that exact output into the report. Never draft a send command with a placeholder hash; omit the hash until it is known.

- **#494 gesture preview:** final click geometry is not a pointer-down preview. Keep the default box out of the preview until release, retain drag intent once its threshold is crossed, and exercise down/move/up on every shared mount surface—not only the final geometry helper.

### #494 — close review identity and event boundaries

- A reconciliation success belongs to its original operation, never the editor currently asking about it. Test cancellation followed by a new draft and transitions between action types; an unsent body cannot receive success.
- A shadow root is not a global keyboard boundary. Focus-preserving commands need a document/window listener for Escape, with explicit guards that leave unavailable/Off page input alone. Test the real page-focused delivery path.
- Classify helpers by what they read, not by the shape of their result: a DOM-to-path function is INTEGRATION even when it returns only numbers.

- **#494 reconciliation lifetime:** operation identity includes the authority’s lifetime, not only request ID/body. Retire browser state when the server’s document ledger retires; test switch-return plus lost reply and late completion, not each lifecycle independently.
- **#494 interactive identity:** a record stream can update between pointer-down and pointer-up. Preserve swatch/control DOM nodes across unrelated updates, and test an interleaved record against the originally pressed element on every shared surface.

### #528 — terminal evidence boundaries

- Keep unknown observations distinct from proven unsafe state through every projection and optimistic UI overlay; test a failed inspection after successful attachment.
- A PID plus ancestry is not a process generation. Pin the child's start identity in readiness, and mutation-test reuse of only that PID while the server remains unchanged.
- A byte limit does not bound open/read on a FIFO. Runtime metadata readers must reject special files, open nonblocking, and validate the opened inode.
- Permanent coordination files require a cardinality bound. Stable hashed stripes preserve exclusion without unsafe unlink races or per-session accumulation.

- **#528 bounded-read family:** fixing named reader sites is insufficient. Enumerate every new persisted-input consumer in the branch diff, including audit/config and capture sidecars, and add consumer tests that fail if they bypass the shared reader.

- **#528 optional artifact fallback:** shared IO error types must preserve the distinction between unavailable optional evidence (raw-only, uncertified capture) and malformed authority (refuse). Exercise fallback consumers, not just the new reader. Deadline tests must admit cancellation before an observer is entered; prove inner timeout separately from caller expiry.
- #523 M2: an asynchronous UI result belongs to the authority generation and
  attempt, not merely the cast instance. Await the actual completion in negative
  clipboard tests; a no-op queue barrier can overtake an unqueued callback.
- #523 M2: clearing authoritative/projection collections must also clear consumer
  transient caches. An empty snapshot alone does not communicate a clear event.
- #523 M2: independent SSE channels may deliver initial permission and catch-up
  control in either order. Retain evidence while unavailable; gate its projection
  and invalidate consent on authority-generation changes.

- #523: A published capability phase is not worker completion. Concurrent test
  operations must await admission or their own completion receipt. Always defer
  release of a held test barrier before an assertion can fail, so failure reports
  remain visible instead of hanging fixture teardown.

- #523 M2 review: ingress is already too late to assign revocable authority.
  Carry request-creation generation across the browser/network boundary and test
  delayed-before-ingress payloads as well as already-queued writes. Audit sibling
  mutation routes (request, release and input) for the same temporal gap.
- For animation settlement, border-box intersection is not a painted-influence proof even on leaves (shadows/filters/outline). Prefer bounded conservative document-wide observation to accumulating incomplete paint exclusions; label uncertainty explicitly.
- **A plan names verification functions and strategies, not a prose copy of test cases (#526 and #687 plan-quality).** For each risky function, name the adversarial input class and mechanical guard. Retention claims must distinguish actual operator deletion and capacity refusal from staging cleanup; a staging remover is not an automatic retained-evidence sweeper.

- **A new production Go file must enter the exhaustive source inventory (#526).** Register its artifact ownership in `cmd/internal/artifactpath/manifest.go` alongside the package's existing sources, then run `TestProductionArtifactReferencesAreExactlyClassified`; targeted package tests alone do not exercise this repository-wide invariant.

- Privacy/permission JSON must validate exact schema keys before Go struct decoding: DisallowUnknownFields still accepts case-folded aliases, including Unicode. Test conflicting aliases at each envelope and nested evidence boundary.

- When fixing permission-decoding ambiguity, enumerate root manifests as well as nested policy/rule codecs. A strict nested decoder cannot defend permissively decoded top-level permission fields; exercise store reload, not only standalone policy parsing.

- #523 headless conformance: DOM element/global existence is not readiness.
  Wait for module completion before submitting forms and await the viewer's
  explicit ready contract before reading its renderer.

- #523 operator fixture: recorded publication precedes browser acknowledgement.
  Before the next gesture, await the editor's acknowledged completion, not only
  the Board record; verify with a deliberately delayed reply.

- #523 close review: strengthening a formerly synchronous enqueue into an
  acknowledged external write changes every caller's latency contract. Enumerate
  all input-handler callers and move waits onto bounded lifetime-owned work;
  carry admission identity across scheduling and test responsiveness with ACKs held.

- #523 drive fixture: controller acknowledgement can precede UI delivery and
  paint. A named chip can move neighboring hit targets; sample the committed
  projection before clicking, never combine a new screen with old coordinates.
  Bound fixture waits independently of the package deadline so one missed menu
  does not consume the entire suite's time budget.
- Fresh review checkouts and reused issue slots must run make runtimebundle-generate before direct Go tests that import embedded runtime assets; make build/test already supply that prerequisite. Distinguish Chrome startup failure from a failed capture assertion and rerun the unchanged live check before changing product code.

- A macOS Seatbelt-fenced reviewer can prevent Chrome GPU startup even with an allow-default profile. Before treating this as failed product conformance, compare the identical test under the fence and in the authorized parent environment, retaining real assertion evidence and the review fence.

### #524 — fresh-send recovery and clean smoke artifacts

- A preparation failure must not lock authored text. Keep draft ownership separate from confirmed-delivery recovery; late acknowledgment belongs to its submission ID, never the current editor epoch.
- Supersession is authority release, not delivery evidence. Define ordering across manifest writes and successor publication, and inject failures through every durability checkpoint.
- Build smoke artifacts after committing, then check embedded VCS revision and modified state. A clean checkout and a successful earlier build do not prove the binary was built from that clean commit; timestamp-based make targets can leave old embedded metadata.

- A full suite that compares embedded assets with source must run against a stable tree. Finish source edits before launching it; if an independently verified final delta lands mid-run, disclose snapshot failures and rerun the affected packages after regeneration rather than calling that run green.

### #528 — recovery transaction integration

- When a start claim owns recovery evidence, every transition that removes the claim must transfer that evidence, including rollback with and without a park receipt. Test each destructive transition, not only registration.
- Recovery must not change saved startup preferences through the shared successful-start path. Assert defaults before and after a real store-backed recovery transaction.
- Pair helper ancestry does not necessarily reach its wrapper after Zellij daemonizes. Use exact readiness nonce and agent process generation when identifying the source; scope/tag alone cannot distinguish competing incarnations.
- When integrating capture availability changes, test a batch containing both visible and off-screen notes. Pixel unavailability does not imply lost document identity; preserve navigation rejection separately.

- **#528 exact native ownership:** a live wrapper plus the newest tag-level ledger entry is not a causal join. Bind the wrapper generation through readiness's exact live agent and launch ordinal; test an older live wrapper alongside a newer same-owner launch. Apply the same association rule to source exemptions and fakes.
- **#528 primary recovery routing:** a new secondary action does not satisfy the primary recovery contract. Test Enter, mouse, explicit Relaunch and equivalent restart gestures against the shared preview, preserving ordinary healthy behavior.

- **#528 recovery conformance:** extending a native format consumer creates a new contract even when the older scanner still parses the file. Exercise the new production interpretation in installed-native conformance, and prove recovery-only drift fails while legacy inventory remains valid.

- **#528 store authority:** coordinators call named typed ThreadStore transitions, never updateExistingThread callbacks. Include TestArbitraryLifecycleMutationHasNoDoor alongside targeted lifecycle integration checks; behavioral tests alone do not cover the authority boundary.
- #535: Keep error text and wrapped causes complete; apply concise summaries only at operator presentation boundaries. Run the full affected package and use make -k after an early prerequisite failure so one known timing failure cannot hide later regressions. Cancellation tests should cancel at the intended observation seam, not race unrelated filesystem work against a tiny deadline.
- #533: An operation deadline cannot bound an editing lock while its synchronous subprocesses are unbounded. Carry one remaining-time budget through every external effect and render progress before entering waits. Verify an actual blocked child and embedded UI redraw ordering, not only model state.

- #533 embedded UI: provide the full production startup environment in private fixture files and strip live-session variables. Capture the rendered screen on RPC timeout: a missing completion path caused a hit-enter prompt, which otherwise looked like an unexplained transport stall.

- #533 close BR-1: nomodifiable only guards buffer writes. Navigation, queue, delete, reset and send handlers may mutate logical identity or files first; operation ownership must reject the whole action before those effects. Test persisted contents after failure/timeout and subsequent navigation, not only edit rejection.

- #533 close BR-1 round2: enumerate actual write sites, not only user commands. Focus/cursor/autocmd/timer decorations can write real bytes and open a blocking prompt under nomodifiable. Preserve deferred cleanup bookkeeping and exercise focus loss/gain in an embedded UI.

- #533 integration of #531: a full make test stopped by the known #514 timing check has not exercised later Go packages. Run remaining targets with make -k and inspect all failures before citing the exception; new production files still require exhaustive artifact-source inventory entries.

- #538: A recorded node ID is document-scoped evidence even if the live recorder usually allocates globally. Before giving a connected ID priority over scored anchors, verify the note belongs to the current document; test a same-tag ID reused after navigation.
- #538: Recovery tests must restart the real lifecycle identity. If a new cast always gets a new recording path, reopening a service with the old path alone misses orphaned save/publication intents; exercise a new cast recovering an inactive prior recording.

- #537 recording review: when record families merge by timestamp plus file rank,
  live writers must advance tied timestamps before a transition to an earlier
  file rank. Test real split/merge equivalence around pause, clear and end.
- #537 UI review: mutually exclusive presentation phases do not erase independent
  authority facts. Test OFF/view-only/paused crossed with active driving, and
  preserve held editor text when newly accessible read-only UI is opened.
- #536: Sharing note-box geometry does not share gesture timing. Keep threshold crossing and drag latching shared too; test each consumer's press, sub-threshold movement, release, and drag back toward its origin.

- #537 verification: adding an authenticated probe or POST route requires updating
  the static same-origin transport inventory as well as runtime behavior tests.
  Stage-aware fleet producers and consumers must agree on the new reason before
  live conformance runs while a smoke checkout is staged.

- #537 close BR-1: pending-input tests must cover creation separately from edits.
  A successful creation consumes its draft identity; prevent further draft input
  until acknowledgment or transition explicitly to the committed note, never
  leave a success eligible to create again. BR-2/3: reconcile final entity paths
  and classifications, and update user-facing README instructions alongside atlas.

## Retiring a feature: sweep its side effects, not just its callers (#522)

Deleting the orientation delivery also deleted the only `automaticRenderEpoch.Add(1)`, which lived inside its terminal observer. Peer delivery's post-submit lease reads that epoch, so the next peer message would have waited forever. Every caller compiled, and the dead-symbol scan was clean. Only a unit test of the surviving consumer, written while retiring the old writer, caught it.
- **Rule:** before deleting a function, list every state write in its body (counters, flags, channels, files). For each one, name its surviving reader and move the write there.
- **Rule:** when a retirement leaves one consumer of a shared mechanism, pin that consumer's behaviour directly before deleting the other.
- #540 admission review: when identity checks precede a workspace lookup, test a
  wrapper replacement during that lookup. Cached connection membership alone
  does not prove the current wrapper still owns the accepted launch; revalidate
  at effect admission. Exercise lost acknowledgments after a real side effect
  through the socket, not only an error-returning client stub.
- #540 verification: when changing a consumer's notification policy, search
  upstream integration tests too. Wrapper-to-Couch tests can encode forwarding
  expectations outside the consumer package. Prove suppression after a positive
  processing barrier; never wait for an intentionally absent notification.
- #540 close BR-1: best-effort output helpers may intentionally collapse dropped
  or timed-out writes into nil. Explicit commands claiming a completed effect
  must retain the actual write outcome through degradation/recovery; test both
  pre-existing stalls and stalls caused by the command, including no replay.
  BR-2: append acceptance criteria when approved revisions expand behavior;
  a detailed plan revision does not update Done-when by itself.
- #519: A successful datagram send establishes kernel acceptance, not consumer delivery. At stream EOF, join the receiver, boundedly drain its kernel backlog, and consume its channel backlog before finishing the sole output writer. Pin both pending-work boundaries with ordered fixtures rather than extending deadlines.
- #538 integration: expanding a list across documents also expands reconnect and edit consumers. Check catch-up equivalence against every retained note (including source segment), and test a historical note through current admission, owner checks, edit, and submission; a current-document-only oracle hides missing list entries.

- #538 close: distinguish trusted connected-original DOM identity from scored replacement evidence on every surface. Test common-text originals and large pages through each adapter; never use a mutable path to grant original identity. Reconcile plan inventories with actual symbols, capabilities and changed-file status before close.

- #514: A collector's cooperative budget is not a hard subprocess deadline. Keep shared-host tests on output and controlled-clock scheduling invariants; retain elapsed-time ceilings in the explicit performance lane and prove both gate modes with the same overrun fixture.

- #529: Selection tests must enter through the production constructor before supplying optional test overrides. Injecting a choice inventory in every test can hide a nil production inventory even when navigation and preview reducers are individually correct.

- #542/#511: A signal-probe permission error is not positive process-group membership evidence on every OS. Confirm ambiguous results through the OS membership boundary and retain ownership on query failure. Survivor fixtures must acknowledge signal-handler readiness before their parent exits; a fork is not a readiness event.

- #511 close review: count repeated test outcomes from the complete log, using exact per-subtest records. A terminal tail or truncated tool result is not evidence of the total failure count.

- #528 M1 BR-9: Test an offered Continue through the real resume executor and
  launcher, including missing profile, incomplete argv, unsupported saved agent,
  and missing original cwd. Projection-only tests do not prove downstream cold
  launch requirements were removed from warm attachment. BR-10: startup and
  primary-action changes must update README usage as well as the atlas.

- #528 M1 BR-9 follow-up: A terminal client does not need lifecycle ownership
  of the agent it views. Sweep executor behavior across all record shapes the
  classifier accepts, and preserve independent start/park records. Persisted
  registry PIDs are not proof of a terminal handle owned by this process.

- #528 M1 BR-9 authority sweep: follow an offered action through its post-ack
  completion and the real launcher. Removing a prerequisite at the first gate
  is insufficient when registration or final handoff imposes it again. Keep
  attachment evidence separate from lifecycle retirement authority end to end.
- #530: When a fixture advances a synthetic clock, first await completion of all
  independently admitted deliveries that affect its rule inputs, not just the
  endpoint whose messages the assertion counts. An in-flight startup prompt
  correctly makes a worker Pending; drain its submitted receipt before testing
  an idle-time rule. Preserve facts newer than the liveness snapshot pruning them.

- #530 close: if the contract distinguishes generated traffic from agent traffic,
  cover every generated-send path, including older reminders. Observe terminal
  broker receipts as well as endpoint callbacks: queued expiry/cancellation may
  never call Deliver. A bounded diagnostic cache must not become an admission
  gate for authoritative facts; retain pending metadata and release it at the
  existing receipt lifecycle boundary instead of building a second history.

- #530 round 2: terminal receipt and completed callback are different facts.
  Disconnect may publish an outcome while endpoint accounting is still running.
  Retain generated-message provenance through the broker job lifetime, checked
  atomically with its receipt; test cancellation before a blocked callback returns
  for every producer, plus genuine-message inclusion and eventual cleanup.

- #543 plan review: best-effort diagnostic writers still need explicit ordering,
  shutdown and late-open ownership. Offline labeling must validate bounded input
  before inference and treat malformed or incomplete coverage as unknown.

- #543 close BR-1: inclusive time windows need coverage strictly beyond their
  endpoint when timestamps can tie. Test truncated tails and sequence gaps at
  the endpoint; an equal-time observation cannot exclude a later equal-time event.

- Native process inspection must distinguish full executable identity, mutable argv[0] and truncated kernel comm. Preserve unavailable rows and fence generation across separate command reads; converting a ps table to short kernel names alone silently drops long runtime names. Bound numeric PIDs before narrowing them to kernel argument types. (#547)
- When replacing a subprocess observation with native IO, sweep shell fixtures as well as Go tests. A PATH fake no longer isolates that native boundary; use an injected observation or a real controlled process topology rather than adding a production test bypass. (#547)
- A process table can retain exited, unreaped processes whose argv is unavailable. Use native zombie/terminal status from that same table to exclude them from live-command observations; never infer death from EINVAL, permission failure or empty argv. Exercise a real exited child before Wait and real shutdown conformance. (#547)

- #547 BR-1: A subprocess timeout only supervises the direct child. Test helpers that reparent commands must own the process group before launch, handshake startup, and clean the group on failure/cancellation. Verify descendant lifetime with a kernel-held resource; preserve intentional success-path sidecars.
- #546 integration: an operation queue may have concurrent workers even when admissions are FIFO. Prove effect ordering with barriers; ownership changes need one shared execution gate, and queued mutations must repeat authority checks after acquiring it. Explicit startup and background recovery must consume the same ownership projection to avoid duplicate starts.
- #546 retry evidence: after a successful move, the original caller may no longer own the target. Authenticate the caller and answer its retained full-intent receipt before checking authority for a new effect. Include persistence mode and destination in deduplication; no-op durable commands must not advance the ownership revision.

- #546 close review: retries that replan from visible filesystem state still need durability confirmation; unchanged bytes do not prove a failed directory sync later succeeded. Admission-time authority evidence must survive delayed delivery, rather than being reconstructed only from the current owner.

- #546 multi-owner changes must enumerate observability consumers as well as routing/guards: delivery facts are part of the behavior contract, and a main-only logging predicate can silently lose confirmed secondary-owner evidence.

- #544 ad-hoc review: a queued screen-driven key needs proof from one coherent terminal generation, rechecked at emission. Fence confirmation after the actual write, not after the earlier decision. Route raw keys through ordinary input accounting even when Return remapping is disabled, and prove interruption through the real input loop.
- #545: Every new production source must join artifactpath's exhaustive source
  inventory (SourceClassifications or NonArtifactSources). Run that package
  when adding the file; focused tests in the new file's own package cannot
  detect a missing repository-wide inventory entry.

- #545: A duplicate of a CLOEXEC lease must itself be duplicated atomically
  with F_DUPFD_CLOEXEC. Plain Dup in a transient-fork fixture creates exec-time
  inheritance the real lease does not have, so unrelated children can keep the
  fixture locked after its scheduled close.
- #550: A launchable harness can still lack peer messaging entirely. Require an explicit receiver policy for every AgentInventory entry and a captured empty-composer case for every enabled receiver. Prove startup registration through the real endpoint and delivery through the broker seam; Return remapping geometry alone is insufficient.

- #550 BR-1: Expanding a supported receiver set updates README's support/evidence paragraph as well as the atlas. Sweep the whole affected paragraph against the current delivery path; retaining an earlier echo-validation contract alongside a new receiver claim misdescribes the operator's flow.
- #547 BR-2: Pipe EOF and unavailable process argv can precede terminal kernel metadata. A process-state fixture must await the state its assertion requires, with a bounded deadline and a deterministic live-then-terminal regression.

- #551: os.NewFile does not establish atomic close-on-exec for a raw descriptor. Set O_CLOEXEC at the raw open, and sweep transient binding reads and directory paging alongside locks. A final FD_CLOEXEC check alone cannot catch a post-open fork/exec race; defend atomic placement over the whole raw-open family.

- #544 smoke: A native PTY conformance pass does not establish behavior under Pair's virtual terminal. Match the production terminal profile, TERM_PROGRAM and keyboard query replies, and observe the actual child input across the Run/socket/writer path. Terminal negotiation can change key interpretation even when the driver emits the same bytes.
- #528 merge verification: a new main guard covers feature-branch sources too.
  When importing a cross-cutting invariant such as native process inspection,
  run that guard against the whole merged package, including files outside
  conflict hunks, before spending a full-suite run.
- #552: An occurrence sequence and its transport ID must both name the exact
  incarnation. PID plus sequence can collide after PID reuse; hash the full
  binding, and reject reordered sequences through a high-water mark retained
  across same-binding reconnects. Define that ordering before promising bounded
  duplicate suppression.
- #552: Cancellation and a queued item can be ready in the same select. Recheck
  cancellation after dequeue, preserve accepted diagnostic evidence, and suppress
  new external effects; test the ordering with a blocked real transport.

- #552: Run the first complete pre-close suite through `sdlc local-check run
  --name suite` on a pushed head. A direct `make -k test` log is valid behavior
  evidence but creates no required sdlc-local/suite status, so landing must run
  the declared suite again. Inspect the declaration before scheduling the run.

- #528 BR-13: Bounding a store lock does not bound persisted-byte reads.
  Sweep every routing/fallback reader reachable from a preview, retain request
  and inspection cancellation, and reuse regular-file/type/size admission.
  Exercise FIFOs at routing, final records, archives and aliases; preserve
  descriptor-relative local opens rather than reopening an admitted pathname.

- #528 BR-14/15: A contextual preview at one consumer is not the full automatic
  path. Trace attachment preflight, optional repair, effect-time rereads and
  refusal branches; warm-only absence must exit before lifecycle bookkeeping.
  After reader refactors, run the production-symbol guard before review and
  remove obsolete wrappers rather than exempting them.

- #528 BR-14 downstream: Follow the launched helper through the handoff, not
  just its parent API. Optional agent presentation must not resolve native
  conversations; a selected attach must not run migration or orphan cleanup.
  Include write-only opens in FIFO probes: opening a draft for writing can
  block just as a ReadFile can.

- #528 final suite: A subprocess readiness file is a publication boundary, not
  merely a pathname. Use the existing atomic writer so a successful read cannot
  see the creation/truncation window. Reproduce scheduling races by stretching
  that window in an overlay; do not hide them with more reader polling.
- #555: For generated-output changes, record both consumer and substrate revisions, before/after blob identity, and repeat-generation result. Disposable review checkouts may lack sibling sources; measured host evidence lets review verify provenance without modifying its environment.

- #528 final fixture sweep: An existence-based waiter needs atomic payload
  publication for socket requests and pane/readiness JSON too. Inspect the
  underlying error (empty Unix bind address) before calling a timeout load-only.
  Keep a shutdown-order assertion independent of idle HTTP client cleanup;
  close fixture idle connections instead of widening its ACK deadline.
- #559: A cache for optional evidence must preserve the uncached fallback when its metadata cannot be observed. A disposable-catalog fingerprint failure cannot become new activity authority or a new requirement for otherwise valid ledger/native proof. Pin parity with the uncached query, and count real decoder inputs separately from filesystem reads so decoding cached bytes cannot hide behind a zero-IO test.

- #559 review: Cache failure tests must recover with unchanged fingerprints and fence an older in-flight loader, not merely reject the failing call. Distinguish a canceled waiter that observed nothing from a failed source observation. Measure retained projections using substantive proof/boundary payloads, and signal cancellation barriers only after the caller actually reaches the wait.

- #562: Exit0 without the requested output is no answer, not an empty inventory or detached client set. Preserve that distinction before monotone caches: failed name probes cannot teach a positive acceptance or a negative length bound. Model the early completion before queued output and require a separately requested observation for recovery. An existence-only readiness check needs atomic value publication when the reader parses contents.
- #528 cancellation: receiving a queued request does not prove it is still
  admitted. Check its lifetime before dispatch; a cancelled predecessor can
  publish completion while Stop is still closing channels. Preserve the
  shutdown assertion and reproduce the scheduling window before changing it.

- #528 live CI: Canonical paths are identities, not always valid transports.
  Expanding /var to /private/var can break a Unix socket near its byte limit.
  Canonicalize the shared lock key while preserving the server's lexical path
  for the client; test that aliases share a lock without rewriting argv/env.
- #218: A native convenience wrapper may erase a successful empty result into a
  generic IO error. Preserve absence/error distinctions at the native boundary;
  reproduce exit between successful argv and identity using an owned child, and
  assert argv succeeded so an older disappearance path cannot make the test pass.
  Keep public generation-token compatibility separate from snapshot liveness.

- #218 follow-through: preserving uncertainty at a producer is insufficient when
  consumers reclassify its errno. Carry typed error provenance through every
  gone classifier; pin launcher absence and diagnostic writer-proof consumers,
  while keeping raw command/environment exit semantics separately tested.

- #218 absence proof: demonstrating absent implies empty does not establish
  empty implies absent. Check kernel filtering before turning missing metadata
  into authority; on Darwin require an independent signal-zero ESRCH, retaining
  alive, permission and other outcomes as uncertainty.

- #218 shutdown: cancellation entry points must remain callable while state
  commits hold their mutex. Cancel the lifetime first; release scoped state at
  joined teardown's existing barrier and invoke cleanup outside the mutex. Test
  Stop with an active pass while holding the state lock, not only idle Stop.

- #218 close review: a semantic retention test must not inherit an incidental
  tiny deadline from a failure-path fixture. Preserve deadline-specific tests
  separately, and capture the failing phase before changing a test budget.
  Isolated rerun passes do not erase a failed broad run or prove its cause.

- #562 fixture sweep: When a read boundary distinguishes an unanswered query from observed absence, update every executable fake that encoded absence as exit0/empty—including shell smoke fixtures, shared helpers, composition fixtures and public-command entrypoints. Search generic launch-only executable stubs as well as literal query strings; a silent default can turn cleanup into an endless wait. A package-only pass cannot cover their separate launcher path. Use the Make wrapper for hosted test-environment isolation.
- #528/#218 integration: When an approved lifecycle path changes, carry
  instrumentation onto its actual steps instead of restoring old lifecycle
  work to satisfy phase assertions. Optional borrowed-client previews should
  be conditional on local ownership; bounded nonblocking previews can starve
  beside a fleet's explicit writes even when each operation is correct alone.
- #554: An emulator reply drainer owns a lifetime even in a one-shot screen fixture. Reuse terminalModel instead of pairing raw Emulator.Close with an unjoined io.Copy; close the reply pipe, join the reader, then dispose the emulator. Pin ordering on real EOF and observable disposal, including an injected shutdown error, rather than sleeps or a probabilistic race-only assertion.

- #564: A semantic test must choose its fixture's time authority. Reusing a failure fixture can silently import a tiny real registration deadline despite a fixed domain clock. Use the existing parent-linked registration context seam and bound the caller by its test budget; pin deadline inheritance and cancellation at observer events. Reproduce expiry before the first poll separately, and distinguish later cleanup probes from completed registration observations.

- #564 integration: when upstream changes a public route, move regression observers to the current route and prove the original hazard still applies before retaining a fix. Assert cancellation propagation at the observer before cleanup cancels the child; a final error alone can be produced by a different parent check.
- #562 close review: enumerate subprocess outcomes before learning monotone bounds. Nonempty diagnostics do not turn a failed/canceled query into acceptance; check cancellation before parsing even validator-looking fragments. Preserve only source-verified complete validator outcomes, with real-subprocess negative and positive controls. Keep Done-when and completed Plan rows at the level of evidence actually delivered: a source-derived fake is not an upstream receiver regression or a captured wire ordering.

- #556: Authenticate a recovery confirmer against the exact live process
  generation as well as broker nonce/session. A PID-file binding alone survives
  PID reuse and cannot authorize stopping another slot.
- #556: Wrapper death does not prove the native transcript has no writer.
  Native admission must retain the child generation through reparenting and
  uncertain readiness; explicit Fresh remains the safe alternative.
- #204 plan/code review: a shared initial process capture is not a total scan budget. Count full enumerations at the reader boundary, including named-snapshot live fences, and propagate one instance through every adapter. Keep ratchets as upper bounds with separate positive connection/success controls so improvements can pass. Consumer regressions must enter actual scheduling/socket/attach paths; cold and mutation controls defend against disconnected observers.

- Recovery under foreign survivors needs read-only readiness on every entry path,
  including retained-record Fresh; a different terminal tag does not isolate a
  shared checkout from workspace reconciliation (pair#556 BR-1).
- Retirement tests must independently assert every advertised process generation
  is absent. A successful server signal or empty server list does not establish
  child/native-owner death; model orphan and reparented survivors (pair#556 BR-2).
- When replacing a public recovery flow, sweep README and older Atlas accounts
  as well as the new feature section so obsolete admission rules do not survive
  beside the current contract (pair#556 BR-3).
- #568: A silent nonzero editor exit can happen after all assertions pass. Capture the return code, pre-exit messages and case-local Neovim log before cleanup. Own filesystem watchers through explicit stop/close before global teardown and invalidate already-queued callbacks; a startup delay or isolated green run does not establish disposal.

- #568 inventory gate: a new native runtime helper needs both its canonical source classification and its generated-mirror classification in artifactpath’s exhaustive manifest. Run that package immediately after adding the file; the embedding build alone does not validate source participation.

- #568 review / #570: Unix socket paths have a byte envelope that includes the caller’s TMPDIR prefix. A subprocess test passing under the normal temp root does not prove it works under a deeper review checkout; preserve the refused endpoint and file a separate demonstrated path-length defect rather than treating an environment-shortened run as universal coverage.

- #573 plan review: removing a routing record can reactivate an older storage authority. Export current folds, gap acknowledgments, snapshots and historical addressing durably before that handoff; a partial export must leave the current authority intact. A size-triggered trim is not an unconditional age expiry, so never prune evidence references solely from its advertised retention age.
- #573 rename ordering: delivery completion time cannot identify the pathname a queued message addressed. Preserve immutable admission time through queue promotion, use it only for addressing, and retain completion time for delivery/status/retention. Reproduce queue, rename, old-path reuse, then late completion through the actual broker/ledger seam.

- #573 wire verification: when adding a time.Time field, audit whole-struct comparisons across JSON/socket boundaries. Assert nonzero/Equal instant preservation before normalizing process-local monotonic metadata; never drop the field from integrity coverage. Run the real wrapper/broker integration as well as producer and recorder tests.

- #573 close review: canonical deduplication must retain every admitted alias needed for authority. Keep aliases with their historical interval, share the derived inventory across locks and legacy handoff, and test rename/reuse/late delivery. Enforce reader envelopes at every metadata producer before an earlier destructive effect: import, whole-read refresh and section-preserving writes can each exceed a limit independently of the document's current size.
- #556 M2 review: classify existing-session selectors from the same context authority used by Fresh validation; a remote/PR selector is not an ordinary launch simply because inventory cannot resolve it. Test both native wrapper entry forms and literal option/prompt controls.
- #556 M2 review: a reserved capability needs revocation as well as consumption. Serialize exact dead-helper revocation with Claim, and test both lock orders: revoke first rejects delayed authority; Claim first retains uncertain spawning ownership. Helper death never proves an already-authorized child dead.

- #556 whole-issue BR-6: an option terminator ends flag recognition, not a native command's positional identity grammar. Parse command/option/positional modes together, including literal -- option values; pin by token identity, not the first raw delimiter. Test canonical IDs after -- through launcher and both wrapper transports, with latest, prompt, fork/new and real value-option controls.
### 2026-10-15 — bound filesystem ancestry in discovery tests (#572 BR-1)

A temporary directory is not necessarily outside the checkout: SDLC review
places TMPDIR inside it. Tests expecting ancestor discovery to fail must
establish an explicit foreign-module boundary. Qualify both inside-checkout
and external temporary roots rather than relying on the host's TMPDIR.

- #577: When changing a declared suite command, run its exact argv directly as
  well as the recorded gate: local-check reads declarations from main, so a
  pre-merge pass may still exercise the old command. Synthetic environment
  fixtures must isolate inherited MAKEFLAGS/MAKEOVERRIDES/MFLAGS; keep explicit
  command-line and recursive Make behavior covered inside the fixture.

- #571: Configuration overrides have their own key grammar. Applying a TOML
  document parser to a literal dotted key can turn comments or newlines into
  a different authority-bearing key. Match the native override grammar and
  prove malformed keys cannot create positive policy evidence.
- #571: Latest-selection pagination must retain the native cursor semantics.
  OFFSET can skip a candidate after an earlier row disappears. Fence every
  inspected transcript, including excluded higher candidates, until the chosen
  identity is pinned; selection authority depends on those exclusions too.
- #571: A pre-open Lstat does not enforce no-follow or bounded reads. Use the
  secure metadata primitive and test a symlink/FIFO swap between check and open.
- #571: Revalidation must accept the final production argv, including flags
  injected after selection. Exercise Couch's shell_snapshot injection through
  the real config reader before Claim; raw argv equality alone misses this.
- #571: A read-only adapter's unknown/error/encoding-limit result must preserve
  ordinary native compatibility without a reservation. Test those failures at
  the launch boundary, alongside strict recovery and exact-ID refusal controls.
- #571: Native Git worktrees and Pair slot worktrees share filename vocabulary
  but different ownership. Keep foreign-path exceptions exact and counted, with
  unrelated-slot controls; run both artifactpath and Couch slot-resource audits
  when adding path literals outside the slot implementation.
- #573 second close: writer/reader compatibility includes every authority handoff's derived fields. Budget the encoded export (history plus mutable state) before accepting growth, then prove forget→re-register at the byte boundary with folds intact. Filesystem tests must create distinct aliases explicitly; canonical paths are already known, regardless of whether TMPDIR happens to use /var or /private/var.

- #573 third close: preflight and effect must consume the same immutable prepared bundle. Checking post-pruning state and then validating an older loaded state during shadow publication can refuse after document mutation. Compile document, shadow, encoded state/horizon and result hash before effects; prove valid gap-acceptance plus pruning succeeds, and ambiguous result sections fail without mutation.

- #569: a successful submit-key write is not proof that native input was consumed. Bound submit-key-only retries by exact unchanged-envelope ownership and human-input admission, then confirm from rendered epochs. Terminal recovery must have an event source before polling stops; use existing render wakes for reconciliation and preserve uncertain text rather than replaying or clearing it.

- #569 process-selection mistake: a list of matching command names is not ownership evidence. Before signalling a test runner, trace its parent chain back to this slot's owned execution shell, match its command line, and inspect its exact cwd and retained log FD. Issue the signal in a separate step only if all prove ownership; prefer an owned job handle/process group. Never batch an unchecked ownership probe and kill.

- #569 close review: operator advice must name the chord that the production wrapper routes to the intended native action. Test the action by following the actual notice through translateChunk; matching notice strings can consistently preserve a wrong key. Sweep README as well as Atlas when confirmation, retry or recovery behavior changes.
- #587 close review: collection count bounds do not imply encoded socket bounds. Exercise maximum cardinality and escaped labels through the real transport; budget every detail section while preserving aggregate coverage and reporting omissions.
- #587 close review: independent snapshot artifacts must fail independently. Preserve valid sibling authority, distinguish corruption from unsafe paths/transient I/O, and test recovery on subsequent passes rather than only the initial error.
- #587 close review: an empty or truncated inventory is not confirmed absence. Publish completeness with the census and test startup with discovery withheld. Route production adapter outcomes through the same pure transition function qualified by unit tests.
- #587 second close: fix a traceability finding by enumerating every table row against actual declarations, not renaming only the cited file. Qualify persistence capacity alongside transport capacity; every pre-admission durability barrier must fit the supported authority cardinalities without evicting suppression records.
- #587 simplification: advisory telemetry does not justify durable alert intent/outcome authority. State the tolerated restart repeat/loss first, keep episode state in memory, and keep best-effort history failures independent from notification. When capacity fixes begin adding codecs and checkpoints, revisit the product guarantee before optimizing that machinery.
- #587 suite: embedded skill tables contain executable command examples. Write separate concrete invocations for optional flags; bracketed usage notation belongs in help syntax. Run the skill-command parser test when adding a command to the embedded reference.
- Process-table metadata and argv snapshots intentionally have different domains:
  zombies and unreadable system services can retain metadata without commands.
  Select possible owners before requiring argv, keep uncertainty for those owners,
  and test new rows arriving between the reads. Fake tables containing only owned
  processes miss fleet-wide false refusals. (#584)

- #584: Before adding a whole-history proof to a foreground gate, measure cold decoding on retained production-sized metadata. Custom JSON field decoders do not inherit unknown-field checks, but need not repeat a whole-row duplicate scan already completed by the outer decoder. Preserve the validation contract with old/new differential tests before optimizing. Independent observations can overlap only with bounded workers, ordered decisions, and cancellation plus joining.

- #584 close review: Classify functions by effects, not their decision-making role: injected IO still makes retry/proof orchestration INTEGRATION; name its deterministic predicates separately. For every new admission-critical external probe, commit a repeatable conformance fixture and connect it to an existing recurring live target, rather than relying only on a one-time host measurement.

- #584: After changing a live Make target, run its workflow-contract tests as well as the new probe. The standalone-checkout dry-run contract has an exhaustive command map; update that consumer with the target and derive its expected count from the same map.

- #571 close: distinguish native config-layer merge from ordered CLI leaf assignment. Repeated table-valued overrides need projection and selected-ID regressions; a scalar-only precedence matrix cannot prove native fidelity.
- #571 close: a transcript field is authoritative only after the native decoder would accept its containing record. Use native-valid positive fixtures and schema-invalid newer-candidate controls; partial JSON extraction is not a protocol eligibility check. Unknown complex optional fields must preserve compatibility rather than acquire selection authority.

- #571 source-inventory correction: protocol map keys are vocabulary, not artifact ownership. Classify exact function/literal/count with moved, duplicate and path-use controls; run the complete artifactpath package so TestProductionArtifactReferencesAreExactlyClassified cannot be omitted by an optimistic name filter.
- #579: when persisted metadata gains optional pointer fields, sweep every equality check on that metadata type. Compare decoded evidence by value; pointer identity differs after JSON replay and can permanently block exact-generation cleanup. Defend both interrupted recovery and eventual expiry.

- #579 BR-1: a pump's EOF and its child process exit are separate lifecycle events. A timer-owned buffer must drain when its scheduling owner stops, before any later child wait; test EOF with the flush clock withheld and exercise flush-error reporting.

- #579 BR-2: a successful rename followed by failed directory sync is an uncertain durability outcome, even when the next read finds identical bytes. Test repeated identical publication with injected sync failures, requiring durability reconfirmation and stable inode/content; healthy no-op tests alone cannot defend this class.

- #579 M2: draining an owner is insufficient if asynchronous producers can still admit after its clock stops. Stop admission atomically with the final drain. Exercise a resize whose successful IO result is delayed across PTY EOF, then require no pending or published late sidecar event.

- #571 mechanism re-plan: repeated fidelity deltas on one foreign selector mean
  reassess the boundary before adding another policy rule. A native database is
  evidence, not a native selection contract. Without a supported read-only
  resolver, preserve indirect native behavior and guard only proven exact IDs.

- #571 reduced close: isolate a native-home failure to its agent when projecting
  a shared ledger; a blocked Codex entry must not discard Claude/Qoder authority.
  A home-routing change must cover recovery discovery and validation together,
  including legacy held-file checks. Test the full candidate handoff, not only
  each helper's root list. Historical working cwd is not startup provenance.
- #594 review: When an approved revision changes certainty guarantees or defers production acceptance, append the revision and reconcile the active Spec and every Done when clause too. A historical Log records the decision but does not supersede contradictory closure criteria; name the pending check's owner and distinguish it from the landing gate.
- #585 BR-1: a reaped process-group leader does not prove its descendants have exited. Preserve owned-group termination across a pending stderr drain; reader interruption alone can hide a surviving child. Regression fixtures must include a real descendant retaining the FD and require both descendant death and joined drain. Reuse the test-budget event helper instead of adding another fixed scheduling guard.

- #595 early review: a known native destination is independent of a resume source. Preserve explicit new/fork IDs in the shared option-position parser and lock that destination before spawn. Best-effort discovery notification must not preempt the existing binding cache/lifecycle publication, and inherited launch environment must never turn a generic wrapped command into a native agent.
- #571 BR6 repeat: enumerate persisted representation shapes before fixing a
  projection. A typed callback cannot establish availability for compatibility-
  only records; apply authority/error policy to every represented agent and
  test the real saved-config/restart consumer, including preserved bytes.
- #597 BR-1: removing a guard at a shared boundary (launcher, wrapper) in favour of an upstream check must cover every upstream path that reaches it. Skipping admission for all Couch-owned launches also stripped it from Couch cold resume, which lacked the registry check. List every entry point (grep the profile builders), give each the replacement check and a refusal test, in the same change.

- #593: A cleanup scan budget must count transient artifacts, not retained final objects. Isolate staging before applying bounds, and test continued writes beyond the listing cap. Whenever a writer gains numbered shards, test that its source inventory includes actual rolled names while excluding locks/staging.

- #593 BR-3: Inventory completeness is per supported producer/backend, not per logical kind. Sweep global, repo-scoped, slot-local, legacy and sharded layouts together; assert coverage with producer-generated paths before declaring the coverage family fixed.

- #589: firing the last retry callback does not prove its asynchronous result reached the terminal state. Keep evidence unhealthy until a state-dependent effect proves that transition occurred, and hold the next successful attempt until the triggering request's refusal is observed. Use the existing authority and retry seams; elapsed sleeps can race both sides of readmission.
- #544 integration with #597: subprocess helpers that intentionally model Couch set their disposable thread identity after TestMain sanitizes inherited live identity. Run opt-in native conformance after TestMain changes; a default-suite skip cannot verify the helper startup path.

- #544 BR-1/BR-2: failure recovery replaces current-screen evidence, so retain actionable observed choices in the bounded result detail and test after completed backout through receipt/CLI output. Reconcile prospective plan symbols with delivered APIs and reused conformance fixtures before close.

- #544 BR-3: opt-in external conformance needs a committed target that enables every required suite, plus a named after-change/upgrade cadence. Run the exact target and confirm cases execute instead of accepting a skipped default-suite result.

- #544 BR-4: conformance must requalify every new observation, then separate admission from dispatch. A known zero-input refusal can justify a distinct fresh fixture request; Step0 alone never authorizes replay of an unconfirmed write. Force both pre-admission and pre-dispatch qualification loss with controlled tests.
- #595 merge: never `git stash` (even a pathspec stash) while a merge is in progress. The stash captured index state beyond the named paths, and applying it reintroduced older versions of unrelated files, some as "Stashed changes" conflicts and some silently. Split commits by staging paths (`git add <paths>`, `git commit`) instead, and after any accidental stash apply, grep the tree for the removed symbols before committing.

- #612: client attachment changes neither session-server liveness nor index ownership. Fake transitions must preserve presence, pane identity and duplicate claims together. Force inventory between host acknowledgement and terminal adoption so an intermittent native-proof query becomes deterministic; test removal from detached inventory before adding a duplicate claimant, since duplicate refusal can mask a no-op transition.

- #611: performance profiles must record effective GOMAXPROCS and go env GOFLAGS, and distinguish package-isolated from idle and from a suite starting mid-run. Parallel parents report elapsed excluding their children; report child workload plus scheduling windows rather than claiming zero-cost tables. When batching identity reads, validate against actual primary Git-file authority, including separate Git storage, instead of assuming .git is a directory. Preserve metadata revision effects when seeding cheaper durable fixtures, and pin ordered domain/stride fingerprints before pruning Cartesian prefixes.

- #602: adding a reusable Go test-support package still adds a non-_test.go source to the exhaustive artifact inventory. Classify its ownership and run the whole affected package before the full suite; a narrow acceptance -run misses inventory contracts.

- #602 BR-1/BR-2: when centralizing build commands, enumerate each caller's cwd and original absolute/relative source semantics. Resolve the shared builder's checkout independently, preserve caller-relative output destinations, and test non-root invocation. Document the complete new test override/lifetime contract in README as well as the atlas.

- #602 BR-3: a migration guard must fail against every original offender before its synthetic controls are considered sufficient. Normalize target paths in their actual working-directory context; root-relative examples alone miss package-relative recipes. Keep compact regression fixtures for the original command forms and verify the full original source bodies once.

- #583 BR-5: a fix commit is verified against every package's tests, not only the packages it edits. Guards that parse production sources (`artifactpath`'s inventory test reads every classified file) live in a package the edited code never imports, so a green run of the touched packages says nothing about them; run the whole `./cmd/internal/...` before a boundary review. And measure a review's suggested remedy before applying it: rewording a message to drop an "artifact token" still tripped the guard here, because the family token sat in the command prefix (`pair-changelog-open:`) — the sanctioned fix was the terse shape plus its entry in the manifest's inventory.

- #583 BR-6: a kill path's attribution must name the *owner*, not the kind. Evidence every instance of a kind shares — a config's base name, an `--embed` flag, a binary path — authorizes nothing once a pid can be recycled: the record a dead owner left behind can name a pid another live owner's process now holds, and an identity fence re-read at signal time does not cover a reuse that predates the snapshot. State the owner-naming predicate as a required positive (`Expect…` flag plus its value) so a candidate that states none is a skip with its own reason instead of a pass by omission, and demote kind-wide evidence to corroboration. Then fix the class, not the site: an existing test had *pinned* the unsafe behaviour here ("review kind expects no scope path" → kill), so a site-level fix would have left the rule unstated and the next kind free to repeat it.

- #583 BR-7: a kill path's attribution must be **bound to the pid at the moment its record is written**, and must not change for that process's lifetime. Mutable session state (a target/selection file a live UI rewrites) may corroborate a decision but cannot be its subject: when the state moves and the process's argv does not, the process's *own* orphan is the thing that gets classified "not ours". And check what a decided negative **deletes** before calling a skip safe — `reapReasonOtherPath` skips *and* drops the record, so for a kind with no argv-only fallback the "safe direction" removes the only name the orphan ever had and reopens the leak the issue exists to close. Two rules follow: state a mismatch against mutable evidence as missing evidence (keep the record), and prefer evidence the subject writes about itself (argv, at publish) over evidence another actor writes about it later.
- #617: A subprocess test inherits artifact authority through environment, even
  when HOME, PAIR_DATA_DIR and PAIR_TAG point at temporary storage. The first
  leased-title fixture passed a temporary raw capture path while inheriting the
  live PAIR_SCROLLBACK_EVENTS_PATH; openWitnessedCapture truncated the live
  events file before publishing its new witness. Build subprocess environments
  from a small host allowlist, then supply every Pair path explicitly. Prove
  isolation with a foreign-path sentinel and a disposable-write control; changing
  HOME alone is not an isolation boundary. Self-exec test helpers must dispatch
  before testing.M.Run so a command invocation cannot accidentally run the suite.
- #617: procutil.Identity and StrictIdentity are different token encodings.
  Compare tokens from the same producer, and observe terminal process state in
  lifecycle tests: Identity deliberately preserves a zombie's start token.

- #617 isolation review: a source guard must prove the environment at each actual
  spawn, in source order and on every predecessor path. A later assignment, a
  conditional clean alias, an unsafe dictionary update, or an unrelated helper
  string must never bless inherited authority. Keep original-consumer audits and
  negative provenance fixtures; use an AST when text matching loses scope/flow.
- #617 isolation review: an argv marker cannot authorize skipping an environment
  boundary. Always enter the empty environment and preserve original arguments
  as data. Verify shell entry metadata against the deployed Bash version; a
  top-level source-stack assumption can differ from a tiny function prototype.
- #617 isolation review: empty HOME also removes Go's default caches. Consume the
  shared test binary before starting timed fake agents; do not run a cache probe
  in the caller's HOME before isolation or compile Pair per fixture afterward.

- #617 title review: an authority fence before IO cannot authorize facts after
  that IO. Recheck invocation ownership after tail reads and before publication;
  inject supersession inside the read boundary to verify the order.
- #617 profiling review: a terminal projection's cheap no-op is not warm work.
  Measurements must fail on terminal errors and require positive activity and
  supported-usage controls before reporting allocation or timing improvements.

- #617 teardown review: apply owned-group termination at every exit boundary,
  including the wrapper's controlled-close fallback. A child unable to run its
  EOF watcher still has commands; fence PID/start and group leadership before
  signaling, and test with a stopped title plus an unrelated-group sentinel.
- #606: a new production .go file must be added to `cmd/internal/artifactpath/manifest.go`, or the recorded suite fails `TestProductionArtifactReferencesAreExactlyClassified` at merge. Run `go test ./cmd/internal/artifactpath` whenever a change adds a source file. Fixing review Minors after a SHIP close needs `sdlc close --fixed-then-ship` before `sdlc merge` will publish.

- #617 fixture isolation: a disposable HOME may acquire Go's read-only module
  cache. Cleanup must restore write permission only on directories within its
  owned tree without following symlinks; test root removal and preservation of
  a linked caller-owned read-only directory, not just the fixture's exit code.

- #617 fixture isolation: include indirect Pair entrypoints such as installed
  Couch conformance tests. Their environment must come from the shared empty
  allowlist too; a legacy denylist can strip the helper needed by nested fixture
  children even when the direct Pair-spawn source guard is satisfied.

- #617 environment guard: indirect fixture activation belongs to the assigned
  environment, not the executable name or entire test function. Propagate that
  explicit activation through the same value analysis, and retain negative
  controls for ordinary Couch, disabled flags, and unrelated build/git calls.

- #617 BR-1: atomic visible bytes do not complete an artifact's lifecycle. A
  process killed between temp creation and rename leaves residue; use the same
  coordinated private staging/recovery owner as the collector recognizes, and
  test both interrupted publishers and protection of live publishers.
- #604 BR-1: Deferring a live smoke does not implicitly defer a separately promised diagnostic deliverable. Enumerate the Spec's independent obligations and explicitly reconcile any approved scope reduction across Spec, Plan, Done-when and atlas before claiming completion; keep unknown causes unknown.
- #604 BR-2: For a changed user-visible action, update its existing README account as well as the atlas. Explain confirmed versus unconfirmed outcomes at the user entry point; internal architecture prose alone does not update the user contract.
- #618 BR-10: an unauthenticated socket op answers with a purpose-built projection, never a stored record. Records carry caller Bindings (session and nonce are credentials) and bodies. Test that the marshalled answer holds neither.
- #618 BR-11: a new writer into a shared durable file states its growth bound in the plan (ARCH-FUNERAL) and enforces it in code. Before trimming in place, check for byte offsets other readers keep into that file; the launch's `PairLogOffset` ruled trimming out here.
- #618 BR-12: a milestone's Live step needs either a dated Log observation or a Revisions entry deferring it before the milestone is ticked. A step that needs the landed build gets that deferral when the close is planned, not when the review asks.

- #629: Open a declaration comment with its own symbol. Ordinary prose can accidentally name another existing declaration (Connected above MaxQueued) and trigger the stacked-godoc guard. Run the termcmd doc-comment guards when rewriting shared declaration comments.
- #627 BR-1: Before treating an omitted process row as exit evidence, inspect every native enumeration backend for silently discarded reads. Preserve listed unreadable PIDs as unknown and test actual stat-read/parse failures, then test both initial ownership capture and exit polling; an injected top-level error alone misses lossy enumeration.
- #625: An idle flag carries a particular admission policy, not a universal
  definition of work. Check a real empty-composer/background-command case
  before treating it as one. Preserve permission to receive messages while
  independent display activity is designed; do not overload the admission flag
  to make a busy visual look right.
- **A new help source changes old documentation claims (#619).** When adding companion-editor metadata to a source-derived help catalog, update the README shortcut table and search for universal claims such as "every row comes from its implementing keymap". Qualify the source boundary as well as listing the feature; atlas coverage alone does not keep the user-facing contract accurate.

- #632 BR-1: Absence of a recognized token is not evidence of absence. For nullable native observations, validate the entire captured grammar before returning known-none; test arbitrary prefixed text, partial tokens and unknown trailing fields as unknown, alongside recorded positive and complete negative controls.
- #632 BR-2: Reusing an existing test does not modify its artifact. Reconcile each Core concepts status against the actual name-status diff before close; list unchanged fixtures as reused validation outside the changed-entity inventory.
- #639 BR-1: A transition change must enumerate tests by the production transition function and action identity, including opt-in conformance helpers with generic names. Searching only old state names missed a three-press rung helper. Run every native layout-cycle family and retain all geometry assertions when shortening the event sequence.
- #624 fixture integration: run new Make fixtures in both a woven checkout and a source archive. Dependency symlinks may be absent by design; mirror the root Makefile's optional include and retain a direct sentinel contract check. Do not expose internal Make-variable overrides as apparent environment knobs in test source: the derived environment inventory treats those names as caller inputs. Satisfy fake-compiler prerequisites inside the private fixture instead.

- #624 BR-1: an archive's sentinel-presence check does not prove an absent dependency scanner respects it. Retain a separate woven-copy measurement using the actual dependency, and remove each new sentinel independently to show the regression detects every duplicate publisher. Record the source hash and reproducible fixture, rather than assuming archive verification covers woven behavior.
- #629: Removing a wait horizon also changes the lifetime of its diagnostic histories. Queue caps do not bound per-message maps of changing wait reasons. Retire histories whose expiry consumer disappeared; keep duration accounting in the bounded phase that still consumes it.

- #629 BR-1: A queued/held receipt describes the latest observation, not absence of input after commit handoff. Classify disconnect uncertainty from the active delivery owner; test both sender and receiver disconnect for every interim receipt status.
- #629 BR-2: Advancing an injected business clock does not advance an existing real context timer. Test each timeout owner independently at its IO boundary and restore each removed deadline separately to prove its regression fails.

- #637 BR-1: A current notice is transient presentation, not operation completion. Independent event deliveries can overwrite it after success. Fixtures synchronize on an admitted operation's completion channel and test completion-before-late-presentation ordering; observing a phase alone can also precede release of the owner's work slot.
- #631: Removing a shared mutex from external cleanup must retain the existing per-recipient reservation until that cleanup returns. Regress cancellation emptying the old queue entry followed by a new admission, and a replacement holding its own reservation while the old cleanup finishes.

- #638: A successful subprocess exit does not prove a complete piped-input operation. exec.ErrWaitDelay can describe either unread inherited stdin or a held-open diagnostic descriptor; preserve success only after verifying the existing input reader is exhausted and context remains valid. Pair the full-read/held-stderr regression with an over-pipe-capacity unread-input negative control.
- #634: Start a request's admission budget before sender checks, then carry its context through registry posting and recipient IO. A timeout regression must include earlier work over the real transport; testing admission alone misses additive budgets. Refusal bookkeeping reuses the routing snapshot rather than starting a second lookup after expiry.

- #635: Resolve human-facing receipt prefixes within the authenticated caller's visible retained set before consulting recovery holders. An unmatched query identifies an unknown lookup, not a delivery outcome; ambiguity lists only visible full IDs, with exact matches taking precedence.
- #643: Distinguish atomic replacement of one file from publication of a document/shadow/state bundle. A plan must name partial persisted outcomes and reconciliation after IO failure; test a controlled publication failure and prove unread baselines and owed events survive.

- #630: A deadline and bounded pipe draining do not collect descendants that close their pipes before a successful parent exits. A command group owned only for a synchronous check ends on every completion, including success. Test both cancellation and a successful parent with an independently live, closed-pipe child.

- #630 BR-1: A registered-only operation selects or rejects its route under the registry lock before entering a generic legacy loader. A post-load authority check cannot prevent fallback IO, side effects or unbounded locks. Regress unknown and disappeared authority while the fallback lock is held, and prove service cancellation still collects the caller.
- #638 BR-1 (advisory): When positive and negative subprocess regressions share a descriptor-holder protocol, give one test helper ownership of its environment, release handshake and bounded cleanup so lifecycle fixes do not diverge between copies.
- #598: A signal-cancellation fixture establishes the child's inherited signal
  disposition before exec. Noninteractive background shells supply SIGINT ignored;
  Python preserves that until its handler is installed. Pin pre-handler delivery
  with a process gate and retain stderr rather than relying only on sleep offsets.

- Chunked JSONL framing must not search a retained newline-free prefix again on each append. Count actual delimiter bytes examined across multi-chunk records, alongside semantic boundary/limit tests; read/decode counters alone miss quadratic delimiter work. Preserve per-probe progress when diagnosing aggregate worker timeouts. (#626)
- #620: Derive an integration fixture's completion bound from the product phases
  it actually runs, including allowed serial IO and in-flight work. A short
  aggregate wait can reject successful bounded calls; prove that ordering with
  controlled latency and retain a publication-failure negative control.

- #647 plan review: a context-aware helper does not bound runtime methods that discard the context. Name the total operation budget and propagate it across all reads and process handoffs. For detached shortcuts, stderr/logging is not visible refusal: identify and test the existing presenter independently of the uncertain transport.

- #647 M1: retain counted process-table fences when adding independent native confirmations; bracket the pair with existing generation checks rather than duplicating a host scan per read. On Darwin, group-signal EPERM can mean an empty group: reuse exact membership evidence and preserve live/unknown failures. A readiness fixture also observes the operation's completion channel so early query refusal fails promptly.

- #647 M1: preserve simultaneous observation and lifecycle errors together. Replacing an authority refusal with a helper-exit error loses its no-retry classification; test the helper exiting inside the failing observation, and assert no second helper or later read is admitted.

- #647 BR-1/BR-2: sweep complete launcher entry paths as well as leaf validators. An outer retry can replace unknown evidence after a strict parser rejects it. Derive names and state from one validated complete listing; independent short/full observations can legitimately disagree during unrelated session churn (#711 corrects the paired-snapshot design). Preserve opaque names and interpret EXITED only as a validated suffix.
- #647 BR-3: upgrade every shared native fixture consumer when tightening a contract, including composed production launch controls. Retain JSON progress for long package runs: an early fixture failure can otherwise consume the package deadline during cleanup and conceal which test caused the delay.
- A readiness counter contract needs a stateful test through the actual readiness decision, not only isolated zero/one counts. Cover unchanged observations before growth and show that both equality-accepting and never-accepting mutations fail, especially when native acceptance is deferred. (#645 BR-1)

- #650 BR-1: When a second call site needs the same env-gate rule, extract the predicate into the owning package in the same round. Two hand-written copies of hosted-and-knob differ silently; one helper (e.g. launcher.SlugEnabled) plus one truth-table test — covering the fail-closed spellings like ON/1/0 — owns the rule for every entry point.
- #650 BR-2: A mid-issue scope fold-in updates ## Done when in the same edit, via ## Revisions. Otherwise the close review judges stale acceptance criteria, and literal wording can turn wrong: a gated Stop-hook path may still start a short-lived process that stops at the gate, so "no process at all" is not the correct contract.
- #647 BR-4: errors.As proves a joined error contains an exit status, not that the subprocess boundary otherwise succeeded. Diagnostic normalization must accept only a pure subprocess exit error; preserve cancellation and collector failures as terminal unknown. Sweep every consumer of the shared runner and inject collector failure alongside otherwise accepted diagnostics.

- #647 native fixture: Darwin Go deliberately excludes FIFO descriptors from runtime polling. Blocking reads can defeat Close/join, while O_NONBLOCK can return EAGAIN immediately; cleanup-only or descriptor-flag checks can pass without transmitting any bytes. Prove actual readiness/data delivery and no-client/idle-reader/unread-buffer teardown. Prefer an ordinary owned Unix stream with standard cancellation over adding FIFO polling or retry logic.

- #647 BR-6: admission cancellation is checked at each actual effect, including authoritative sidecar writes and positive local fast paths. One shared pre-effect gate and an exhaustive sink manifest prevent an instance-only repair; cancel inside effect N and prove effect N+1 never starts. Keep the admission token alive through the final dispatch rather than canceling it when observation returns.
- #647 BR-5: unchanged local frame generation or text cannot authenticate an asynchronously observed native tab. A native switch may precede PTY repaint, or another tab may show identical cells. Missing frame/client/tab provenance requires an explicit behavior decision, not a claim that a stronger snapshot bracket supplied it.
- #647 BR-7/8: owned synchronous query groups end on success and failure, including descendants with redirected output; use the shared collector at every query entrypoint. Temporary effects create cleanup obligations at dispatch: cancellation forbids new work but must not skip owed cleanup. Report cleanup-start failures without claiming that a successful spawn proves eventual reset.
- #647 native evidence: a parser error without retained raw streams cannot classify malformed external output. Disposable conformance fixtures preserve bounded stdout, stderr, byte counts and subprocess errors separately; a later passing sample does not explain the missing earlier body. Keep private short native socket/log roots explicit when reproducing controls.
- #647 BR-5 final decision: when deterministic local capture already supplies the needed note evidence, remove unused native inference rather than infer file/line/role from unrelated rectangles or disk substrings. Audit actual consumers before presenting a UX tradeoff: internal metadata excluded from JSON and absent from menus is not an existing displayed breadcrumb. Associate retained frames with owned endpoint identity, not replacement active state; reuse the existing publication owner for committed local layout.

- #647 BR-9: confirmation compares explicit decision facts, including derived roles, rather than serializing whole external rows. Cosmetic titles must not refuse a valid action when an independent launch template or existing registration preserves the role; retain refusal when title is the only evidence and the role changes. Thread the existing registration snapshot through every consumer without additional reads, and test the actual action as well as the shared projection.

- #647 M3: carry independently known local pane/session context from intent or candidate pin across asynchronous phases; capturing it only at the final query can bless a replacement context. Decorate existing parent tokens without mutating preparation/receipt ownership or starting the native query budget early. When a shared native schema tightens, sweep every positive review shell fixture as well as direct module tests; retain negative fixtures and all old behavior assertions.

- #647 native oracle: a refused multi-step action does not restore its prior whole-operation mode. Derive allowed outcomes from completed mutation results, distinguish intended dispatch from authenticated completion, and retain exact healthy-state assertions. Missing independent witnesses remain missing after an offline correction; never relabel the retained failed run as a clean pass.

- When a Lua test gains a Neovim API dependency, audit every entrypoint, including shell integration aggregators: passing its test-lua invocation does not prove the same file still runs under standalone lua. Keep pure tests standalone and run Neovim integrations with nvim -l. (#647)

- Tightening an external query decoder requires auditing every positive CLI shell fixture against the full requested schema and state transitions, not only Go consumers or the fixtures reached by a targeted test. Separate query observations from effects and keep exact confirmation counts. (#647)

- After tightening an external reply contract, make every CLI test double serialize semantic models through one generator and enumerate producer-specific positive/fault cases against production decoders. A source inventory must catch newly introduced unregistered stubs before the full suite; verify the guard with malformed-positive and new-producer mutations. (#647 R7)
- A composite-literal realignment is where initializers get silently dropped.
  When a rename for gofmt alignment touches a constructor literal, grep every
  field's initializer and pin each constructor default with its own assertion;
  the feature test that sets the field explicitly cannot see a missing default.
  (#628)

- A non-positive `context.WithTimeout` bound returns an already-cancelled
  context: Go cancels synchronously when the deadline has already passed
  (context.go cancels when `dur <= 0`), so every probe fails deterministically
  on any machine, loaded or not. When evidence seems to contradict that
  mechanism, doubt the record first: the "green at the broken head" was a
  misattributed head, not a won race — inventing a scheduling story to
  reconcile the two was the real error. (#628)

- Under `go test -timeout`, a wait bounded by the test deadline consumes the
  package budget and cascades: every later failure is instant with a shrinking
  budget (remaining/2 margin). Diagnose the FIRST failure and its duration; the
  rest is aftermath. For a mid-hang goroutine dump, SIGQUIT the test binary
  (the package alarm fires only after testwait's margin lets the test exit);
  stack dumps miss sub-millisecond workers, so dump state (schedules, channel
  depths, probe outcomes) from inside a diagnostic test. (#628)

- #648: Batch a held receiver's queue at the input owner's readiness point,
  then freeze membership before the first write. Admission-time concatenation
  misses arrivals during the busy turn; per-envelope turn logging breaks exact
  transcript correlation. Retain original receipt IDs while recording the one
  combined turn, and test recovery of a follower through the real socket path.
- #646: Shortcut verification must exercise the terminal decoder and routing boundary with both legacy and enhanced wire input. Calling the action handler directly can prove clipboard IO while missing the unsupported key encoding that prevents the handler from running. Keep a content-free arrival observation separate from operation outcome when diagnosing this boundary.
- #622 M2 trailer audit: keep Agent and Co-Authored-By in the same final commit-message paragraph. A blank line before Co-Authored-By makes Git exclude the earlier Agent line from its trailer block. Verify with git log --format="%(trailers:only)" before publishing; preserve already-reviewed historical commits rather than rewriting measurement provenance.
- A readiness counter contract needs a stateful test through the actual readiness decision, not only isolated zero/one counts. Cover unchanged observations before growth and show that both equality-accepting and never-accepting mutations fail, especially when native acceptance is deferred. (#645 BR-1)

- #651: A new public operation must close every existing declaration surface, including help cost and exhaustive source inventory. Run those guards before queueing the full suite; behavior-focused name filters skip them. Keep convenience entrypoints with only test consumers in test code instead of retaining an unused production API.
- #651 native proof: A helper that reproduces a production handoff must preserve its nonzero-exit policy. Retain bounded post-attach PTY diagnostics before fixture cleanup. A lock held by cleanup can postpone a liveness decision until the owner exits; screen any policy that promises immediate live-owner refusal before entering that lock, and assert independent liveness after refusal.

- #651 suite scheduling: when a full suite finds a branch defect, finish collecting that attempt before queueing the corrected head. Fix the whole failure inventory together. Extend shared derived test domains for new action facts; an isolated new-feature test does not update the existing exhaustive sweep.

- #651 BR-1: A cancellable future bounds the waiter, not the worker. Enumerate every IO reached after admission, including reconciliation locks, fallback reads, finalization and failure bookkeeping; contend those boundaries and assert worker settlement/reservation release while the foreign lock remains held. Preserve publication uncertainty and synchronous owned staging cleanup after cancellation.
- #651 BR-2: A shared formatter only enforces the callers that reach it. Enumerate direct, continuation, source-stop, switch, detach, leave and debris-refusal entrypoints; exercise them through production operations and preserve typed error provenance. Place a general refusal before routing that can mask its required actionable advice.

- #651 full-suite correction: when a shared transaction gains a context-aware route, derive exhaustive transition inventories from the common journal/CAS sink rather than either wrapper. Enumerate every fixture assigning a journal hook from inside BeforeUpdate; install those hooks before backend copies and arm them at the intended transition, preserving independently checked durable interruption effects.
- #649 PQ-1: Read the full issue body before writing its plan: sdlc issue show prints section headings without the requirements beneath them. When replacing a mechanism, reconcile every old Done-when obligation explicitly; a revised Spec does not supersede acceptance text left unchanged.
- #653: To make an error class transient, classify it once, at the boundary, against a baseline taken before the first read. Don't add checks at each site you know about. The validator under the boundary has its own read paths: here, the empty-generation-token fallback re-read the whole file. A per-site check misses them, and only a baseline compared after the failure covers every path. Mutation-check each piece of the rule (classifier, each baseline branch) with fixtures shaped like production: a stale proof, an empty token, a proofless binding.

- #648 BR-1: A new user-facing configuration variable needs README documentation as well as atlas and agent protocol text. Before close, check scope, units, default, accepted range, invalid-value behavior and compatibility against the parser and rollout policy.

- #652 PQ-1: Before submitting a plan-quality gate, replace edge-case inventories with one named-function adversarial-input strategy and independent mechanical guard per risky function. Retain concrete acceptance regressions as anchors; inventories alone repeat the #648 planning failure.

- #623: A fixture's artificial child sleep spends the product's real process deadline. To test delayed observation delivery, hold the completed result at a test-only callback seam; keep native deadlines and timeout failures unchanged. Synchronize on start/result/release/application facts rather than settlement sleeps.
- #623: Arming asynchronous work and reading its ID in separate RPCs creates a gap in which the work can already start. Capture the pre-arm ID and arm atomically; force a start inside that RPC in the regression, rather than hoping host load selects the race.
- #655: A timer-boundary regression must assert which operation is measured, not certify host speed. Use virtual time across varied setup and command durations, retain the shipped-binary smoke without a fixed ceiling, and prove a subprocess-clock mutation fails.

- #574 plan/implementation: adding trusted provenance to a message requires checking every retained-receipt import, not only caller admission and rendering. A wrapper-only record cannot grant server origin after restart. Current Slots state keys by the stable slot across Fresh addresses; cancellation of a frozen but unstarted publication must feed its known-no-write outcome back to the reducer, or replacement leaves the slot blocked.
- #574 close BR-1/BR-2: a shared result type does not identify the completed recovery mode, and a cached inventory row can predate an adopted agent switch. Carry the actual typed outcome at every successful producer and prefer completion-owned metadata in projections. Test all successful constructors and final completion paths with empty/conflicting cached metadata; presentation strings and requested mode are not outcome evidence.
- #574 close BR-3: a pending informational projection needs an explicit ownership-move rule. Prefer the successful move producer and the existing admission/effect gate over polling or migrating old history. Dropping pending information must preserve the restart barrier and exact admitted receipts; fence extracted jobs as well as queued jobs. Tell the new owner to resnapshot Slots and prove before-drain, frozen, admitted and uncertain cases with two stateful receivers.
- #661: The pair repo's guard tests catch new files only in the full suite. A new production source must be classified in `cmd/internal/artifactpath/manifest.go`: in the exhaustive inventory, as a ResolvedConsumer with its own binding when it imports artifactpath, or as a VocabularyConsumer at a permitted call site for a family literal. Its generated runtime-bundle mirror needs an entry too. A shell test that runs pair needs the `pair-test-bootstrap.sh` line as line 2. Before pushing a new file, run `go test ./cmd/internal/artifactpath ./cmd/internal/testpair` alongside the targeted tests.

- #649: Register every new production source in artifactpath SourceClassifications or NonArtifactSources before the first suite; retention IO anchors alone do not cover the exhaustive source inventory.

- #649 BR-1: When multiple display consumers share activity, merge every admitted source before branching into title/fade decisions; test consumers together with stale persisted facts and recent wrapper work.
- #649 BR-2: Resource acceptance tests must instantiate the production state being bounded. A fake returning constant facts cannot prove projection retention or warm-history work; use actual compact projections and counted IO.

- #665 PQ-1: a plan that introduces validation or a ratchet names each function and gives one adversarial strategy and mechanical guard per risky seam. A prose case matrix does not identify who owns parsing, identity or authority checks.

- #665 BR-1: when enforcing an exclusive authorization path alongside an upstream parser, recognize the upstream grammar, including its whitespace normalization. Regress the accepted spelling class against that parser; a literal prefix check can leave alternate valid spellings as a bypass.

- #665 BR-3: check the registry's concrete dependency examples before claiming a local-tool exemption. ARCH-MOCK explicitly includes Git: keep policy above a typed read seam, use a bounded stateful fake there, and run the same contract against disposable real repositories as conformance.
- #660: Read the complete failed CI job, including successful failure-diagnostic steps, before classifying an external process as dead. A filtered failed-step log hid server activity after a missing-session reply. Keep query absence, client exit, socket state and server/actor lifetime as separate observations; a dead replacement child cannot later paint its awaited frame.

- #660 BR-1: Bounded readers do not preserve evidence when the reporting consumer still prints a prefix. Test the actual failure collector with oversized child/parent wire, command output, wrapper log and actor receipts ending in distinct final markers; sweep every event-history consumer while keeping deliberate first-difference frame diagnostics separate.
- #656 BR-1: A transient classification must run after every terminal check it can bypass. Carry the immediate pre-read fingerprint through recovery classification, and bracket any new evidence read with metadata checks for every dependent artifact. Test same-size mutation during both existing reads and replacements during the added recovery read; mutations staged only before the poll cannot expose these gaps.

- #636: An outer private environment file does not survive an empty-environment constructor. Name every native storage/socket root at the fixture's actual setup boundary and again at nested child construction; poison ambient roots and inspect real child reports to catch missing propagation. A local client Wait is not a server detach acknowledgement, so retain the failed query's own bounded bytes/status and lifecycle ordering rather than adding a later diagnostic query as a substitute.

- #636 BR-1: A context-installed observer must receive per-operation deadline facts from the query boundary; closing over its installation context misreports shorter inherited deadlines. Test a shorter child through the actual adapter and separately cover absent, shorter and longer caller deadlines against the adapter cap.

- #666 BR-1: Repository lint may require verified main-ref authority. A disposable reviewer checkout without that authority cannot prove the ordinary lint consumer ran; retain a successful exact-head package run and named-consumer output from the configured checkout, and present it through the measured-evidence dispute path rather than weakening the repository guard.

- #666 BR-2: Match universal coverage claims to actual cases. A missing warm entrypoint plus omitted park setup does not establish individual omission of every protected function; name the exact coverage or add the missing cases.

- #676: A named native-root query must not inherit the cost or completeness requirement of global discovery. Count directory entries as well as files in real filesystem regressions; keep exact-leaf validation independent of unrelated descendants while preserving duplicate identity and parent-path proofs. Directory stability requires descriptor identity plus directory mutation evidence; a comparator that requires regular files cannot prove absence under a parent directory.
- #668 BR-1: New developer invocation surfaces need a root README entry even when the package README and atlas already explain them. Sweep retention flags, replay commands and diagnostic controls together; link the detailed guide and state the evidence/lifetime limits at the root testing entry point.
- #651 integration catalog: timing out a known semantic boundary should spend an injected deadline only after readiness. Reuse the package's existing budget/deadline seam; assert the cue and DeadlineExceeded as well as retained ownership, so removing a tiny timeout cannot silently turn the scenario into a different failure.

- Retained operation receipts and intent conflicts are resolved before fresh repository/alias lookup. A repeated request must remain inspectable while the catalog is busy or its alias has changed; recheck after unlocked IO before allocating a new receipt. (#681)

- #681 BR-1: Before close, compare every Core concepts location/status row with the actual branch name-status diff. When implementation chooses a separate fixture file, correct the row and append the decision under Revisions; a broad planned alternative in the task list does not repair an incorrect table.

- #681 BR-2: A real transport fixture must configure the production service-side authorization as well as effect-side guards. Delay external reads at every pre-admission boundary, including caller and target workspace resolvers; otherwise a downstream fast path can hide the original timeout upstream.
- #678 PQ-1: A remote stat selects one object; it does not prove a unique name. Preserve multiplicity in before/after listings and require one matching entry before recording verification. Match the mechanism to the actual writer model: a serialized flat folder needs two flat listings, not a recursive namespace framework.
- #678 installed conformance: Treat external flags as hypotheses until measured against the installed binary. rclone v1.75.2 local-backend copyto replaced conflicts with --immutable alone; --ignore-existing preserved bytes, and final size/checksum verification detects the skipped conflict. A bounded output writer must not embed bytes.Buffer: its promoted ReadFrom can bypass Write limits.

- #678 BR-1: A per-batch writer cap cannot defend a cumulative remote reader cap. Reserve against the complete observed destination, including unrelated and duplicate entries, and charge only new members of admitted pairs; test near-capacity reruns and receipt recovery at a full destination. Keep externally variable metadata wire limits distinct from object-count admission guarantees.
- #677 PQ-1: Before the first plan gate, cross-check every risky production entry point against a named-function strategy/guard table, including service constructors, CLI dispatch and scheduler replay. A concrete integration assertion does not replace the function-level contract; delete duplicated prose case inventories rather than appending more cases.
- #677: An injected environment boundary must stay injected through construction. Parse configuration once and pass the typed value; a constructor that rereads ambient os.Getenv can silently undo the caller's decision. Test an invalid ambient value against a valid injected environment.
- #677: Count native observation through every proof consumer (current writers and immutable readers), and scope evidence to one sweep. Global process uncertainty is not proof that a closed, unheld payload has a writer.
- #664 BR-1: A "nothing happened" detector needs a positive control taken at the same moment as the negative assertion. `:edit!` detaches an `nvim_buf_attach` listener (on_detach, then silence), so an on_lines count after a reload reads zero whatever happens; re-attach on detach and prove each detector fires on a known operator action after the event under test.

- #576: A new row suffix must preserve the existing label-width floor after all suffix composition, including owner labels. Geometry-only render checks can pass while the slot identity and selection marker disappear; assert those visible fields with oversized running and failed details through the real renderer. Keep full retained diagnostics separate from clipped display text.

- #576: Inserting a declaration above another declaration must not steal its preceding godoc. Keep each comment adjacent to its named declaration and run the existing termcmd declaration-comment guard alongside the source-inventory guards before an expensive suite; that guard covers production files across packages.

- #689 BR-1: An authoritative complete inventory needs destination-level evidence even when its record set is empty. Store the complete observation time and binding atomically with the snapshot; partial historical attestations must obey that evidence before claiming availability. Test initial empty refresh and binding replacement followed by projection of previously unseen receipts, in addition to disappearance after indexed presence.

- #689 M2 BR-2: Derive a decoder's whole-stream budget from the admitted payload plus the shared framing allowance, rather than the format's worst-case payload ceiling. Prove the exact total and one-byte-over on both ordinary and large snapshot kinds; a gigabyte overflow test can hide a much smaller promised bound.
- #689 M2 diagnostic check: Quote untrusted member names in library errors and sanitize the final CLI error boundary with the shared terminal-text owner, because child and filesystem errors can carry controls too. Regress the real reporting seam with OSC, C0, C1 and printable Unicode; whitespace folding alone is not terminal safety.
- #687 close review: a composition regression must enter the production options/constructor path with its stateful inspection seam. Replacing its Proof in the test only verifies the adapter, not the wiring; confirm a controlled wiring removal fails the production-path regression. Freeze prepared issue/plan artifacts while close reviews run, even for verification-only log updates.

- #687 BR-1: pure predicates use literal value fixtures, even when a convenient integration fixture exists. Audit setup as well as assertions: a test for a PURE entity must not create/read metadata or inject a process probe; keep those effects in separate integration tests.

- #683 PQ-1/PQ-2: A terminal recorder must account for failures after effect completion and successes that wrote no durable row. Define admission ordering and the lifetime of supersession evidence, then sweep existing capture writers so nested helpers collect evidence while one terminal owner publishes. Test reverse completion with identical clock values.
- #683 PQ-3/PQ-4: Length bounds do not keep argv secrets out of diagnostics; persist catalog-only process kinds. Plans name each risky parsing/normalization/projection function and its adversarial guard rather than repeating a case inventory.
- #688: An alarm may expire after child readiness but before the observer runs. Do not assert a live lease inside that timing window; hold the native fixture explicitly while checking ownership, then release it and require alarm evidence plus EOF. Readiness/lease checks must survive multiple timer intervals, not just a quiet machine.

- #688 BR-1: Partial-failure isolation must include publication, not just acquisition. Test a first-file publication error followed by a later success and require both accurate final summaries and the aggregate error.
- #688: A read-only collector must reject canonical overlap of source and work/output roots before cleanup or creation, including missing descendants behind symlinks. A final rename can otherwise overwrite the very source whose reads were carefully protected; test unchanged source bytes and zero external calls on refusal.

- #688 BR-2: Isolation between sets is a cross-product invariant: compare every output with every source, not paired labels. Canonicalizing a common output parent does not resolve a destination leaf symlink; exercise a different source kind behind that alias, including when its kind is filtered out.

- When changing an encoder’s level or threading policy, search README and all atlas resource descriptions for both the old level and one-thread promises; update the shared resource contract as well as the feature introduction. (#702 BR-2)

- #695 M1 BR-3: A source-text guard such as "no one marshals this type directly" is dodged by the next writer that spells the call differently. Put the invariant on the type: a preserving encoding belongs in the type's own MarshalJSON, so every encoding keeps it, and pin it with a behavior test that fails when the method is removed.
- #695 M1 BR-5: Tolerant decoding means type-level strictness only. Semantic checks tied to this build's catalogs (root names, path parsers, agent lists) run where the value is used and refuse that use. At decode they refuse the whole record, and they refuse a newer build's values, which defeats additive evolution.
- #695 M1 BR-4: A field computed for an operator surface that nothing reads is a promise with no consumer. Wire it to the surface and test it there, or remove it and name the issue that owns that surface.
- A capped subprocess writer must not embed an io.ReaderFrom implementation that bypasses Write: io.Copy uses promoted ReadFrom. Test the actual exec boundary with oversized output, including a valid JSON prefix plus padding; overflow must refuse the whole report, not grant authority to the retained prefix. (#700)

- Qualify runtime evidence through its real encoder using non-UTC clocks, including a mixed healthy/stale pass. A strict UTC codec plus UTC-only fixtures can hide a production local-time mismatch that discards every database result. Normalize at the observation clock owner. (#700 BR-1)

- #695 M2: Before replacing an untyped exemption with a stronger authority, measure the contract the exempted caller keeps. Fresh recovery promises to stop only what the operator confirmed, and to leave a session another slot shares. A forced teardown would have broken both, so the exemption became a typed, named fourth authority rather than disappearing.
- #695 M2: Anything that kills by a shared key (a lease, a tty, a slot path) must not span records it does not own. Key what a teardown kills by the record being retired, here a per-thread lease, and treat a published device such as a tty as evidence only while its publisher's tree is still up, since the kernel reuses it.
- #695 M2 BR-11: A proof must witness each process kind it claims to have stopped by that process's own exact identity. Treat a derived witness such as "nothing on its tty" as a supplement only: when the derived input is absent (no tty published), the derived witness silently matches nothing and proves a live process gone.
- #695 M2 BR-16: At a milestone close, grep every test name and contract in the plan's governing revision against the tree before running the review. Each mismatch gets code or a Revisions line in the same commit, never silence.
- #708: A current ancestry/FD/environment snapshot cannot prove that a detached job will never reuse a temp path. Use an explicit lifetime owner or a documented conventional age-cleaner contract; do not turn observation into deletion authority.
- #708: Separate a shared volume/container's permissions from private child permissions. Qualify the real parent with a read-only stat, then fixture its observed mode (TestScratch/tmp is 0755); requiring private 0700 on an existing shared parent silently defeats admission. Keep the parent unchanged and enforce 0700 on owned slot/bookkeeping directories.

- #708 BR-1: Optional maintenance must not become interactive admission's lock owner. Enumerate the full path (mutex, advisory lock, and preflight lookup); acquire opportunistically and check availability before expensive resolution. Pause a sweep and require admission to return before release, with zero resolver calls.
- #708 BR-2: Constructor callback-presence assertions do not prove production wiring. Inject native observation and scheduling through the real composition, exercise both admission outcomes, and mutate each startup/worker callsite off to verify tests fail. Use a fixture clock whose outcome differs from the host clock; prove cancellation joins a held worker.
- #708: When adding a slot path or native fixture, run the existing slot-state coverage audit and repository lint catalog alongside feature tests. Reuse EnvName for slot basenames. Native socket fixtures need a derived address budget and a long-TMPDIR regression; use the documented short-root lint exception when inherited temp roots cannot fit. Local feature tests alone do not enforce these repository contracts.

- #712 PQ-1: Before adding a diagnostic phase, exercise the real persisted codec and accepted enum validator. Prefer an existing phase plus typed detail when the operation is a subphase; a pure error test cannot establish storage compatibility.
- #712: Fixture readiness must derive from the producer generation and lifetime, never a querying process environment or a marker that outlives the producer. Inject a permanent observation refusal through the real cleanup path and verify both a total budget and retained exact ownership.

- #712 BR-1: Enumerate metadata locks as IO in a timeout promise. A context passed to native probes does not bound a blocking GetThread, and rereading identity after expiry can recreate the hang. Use the context-aware read-only route, retain observed identity before expiry, and contend the actual store lock while asserting return before release. Bound disposition persistence separately and report a failed commit without discarding the occupied claim.

- #712 BR-1 disposition: A deadline-safe observation followed by an unbounded state write still hangs. Enumerate routing, metadata reads, CAS retry and journal locking under one persistence context; contend both the pre-read and read/write seams. If durable unknown cannot commit, keep the exact occupied claim and report that uncertainty too.
- #721 PQ-1: A named-function adversarial strategy table owns the test plan. Tasks reference it and carry execution commands; do not duplicate it as prose case inventories. Keep unique safety invariants in design/state/format contracts.
- #721: Type-owned preserving codecs must normalize absent unknown-member maps consistently with their clones. Empty allocated maps can break exact durable-intent comparisons after a JSON round trip; test equality through the real profile/record encoding rather than weakening the claim check.

- #721 BR-1: Freeze the exact selected snapshot before leaving acquisition; rebuilding from aliased observations can silently accept a changed candidate. Revalidate exact selected artifacts at effect admission, not only after a claim. Mutate through retirement/nonce hooks and test nil verifier and expiry during IO.
- #721 BR-2/BR-3: Shared IO budgets charge only stages that can run. Preflight using the replay owner’s actual raw/event caps before reserving bytes; a rejected witness must not reserve unread raw data. Observe descriptive file activity independently of companion-file and retained-identity authority, and test early failures as well as late ones.
- #721 BR-4: A user-visible command change updates both root README reference and usage passages alongside atlas. Search all command mentions before close.

- #725 BR-1: Affected-test group roots come from every real command and textual consumer exercised by its recipes, not from command-family names. Audit recursive Lua/Python/shell helpers and dispatch routes; pin every recipe's command roots and transitive dependencies with the actual planner/Go graph. Cross-cutting source scans need additive path coverage even when their packages have no Go import relationship.

- #725 BR-2: A Go dependency audit does not cover files loaded, copied or grepped by Lua/shell/Python tests. Inventory both kinds per recipe and qualify all expanded inputs through the actual planner; broad editor-source coverage handles dynamic Lua imports. Measure the non-Go/full split before paying for a selective inventory: here non-Go was 286.506s against a 543.716s recorded full baseline, making selection worthwhile. Keep global textual scans as fixed preflights and shared/unowned inputs conservative.

- #725 BR-3/source audit: dependency graphs describe imports, not Go tests that read other packages or non-Go repository files. Group trigger roots do not execute their Go tests. Fall back to full for unmapped file-input populations; run fixed named source-scanner tests before exact selected Go argv and fail new/changed lexical reader sites pending audit. Test the actual upstream planner and actual runner composition together.

- #725 BR-4: A file hash inventory only defends changes after its initial audit; a mistaken local-fixture disposition can preserve a silent hole. Trace helper calls to real roots, enforce source-walker registration independently of labels, and seed a violation in an unrelated package through the actual affected command. Require each scanner test to execute, rather than checking only that its package appears beside a filtered test regex.
- #720 BR-1/BR-2: Do not grow an optional host feature into a native settings resolver. When permission fidelity needs repeated fixes, prefer an explicit persistent operator declaration and name its false-declaration limitation. A host write probe cannot prove sandbox access. Keep opt-in filenames outside reserved artifact families instead of weakening path-use guards.

- #720 BR-3: A diagnostic added to a show command must cover its valid non-thread targets too. Test a slot with no thread through dispatch; iterating matched threads alone silently omits machine-wide selection information.

- #724: testing.AllocsPerRun reads process-global Mallocs; GOMAXPROCS(1) does not suspend unrelated goroutines. Isolate strict allocation oracles in a self-exec helper and bind the sample to its actual PID; keep an unbounded-implementation negative control so isolation does not weaken the bound.
- #724: Publishing a native PID is not simultaneous identity availability. A fixture must re-observe unavailable identity until established, then retain that exact generation; caching the PID must not disable all later identity observations. Report both title publication and agent identity on timeout so a real live title is not misdiagnosed as never started.

- #711: Registration identity and observation freshness are different guards. Check actor/registration replacement first; ordinary operator input invalidating an otherwise live reservation belongs to the existing busy queue, not a missing-registration error. Reproduce input between successful Reserve and broker commit, and retain sender/recipient disconnect plus same-binding reconnect negative controls.
- #711: Validate a native snapshot before renewing its cache timestamp. A completed subprocess can still return malformed evidence; preserve the prior good observation and its original age. Count one native query at N/2N sessions and zero native calls per cached reader rather than repairing contradictory sequential snapshots with retries.

- #711 BR-1/BR-3: Current connectivity is not the registration lifetime that began an admission. Capture the owner actor under the initial lock and retain that exact resource through routing IO, successful reservation, and release-to-queue fallback. Test same-binding sender reconnect at every yield boundary as well as recipient replacement; keep all refusal paths side/phase labeled.
- #711 BR-2: Preserve validated values through the final consumer, not only validated keys. Passing exact names onward while reparsing state with Fields/Contains revives ambiguity; runtime/display builders consume the same validated state map. Cover live EXITED-containing names and exited whitespace/tab names, including zero client queries for exited rows.
