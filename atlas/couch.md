# couch — the session supervisor

`couch` is a second binary in this repo (`cmd/couch`) that supervises agent
sessions: it registers them, spawns them, and knows what is running where. It
is **not** an extension of `pair`. pair is what the operator sits inside, so a
supervisor bug must not break the ability to fix it; the fallback is always to
launch pair the old way.

Project: `workshop/projects/couch.md`. Registry/spawn shipped in `pair#145`;
the pty console, actor panel, notices, and complete local lifecycle shipped in
`pair#146` M1-M4.

## What exists today

The absolute physical `COUCH_STORE_DIR` is one durable namespace. One Couch
supervisor owns it through a non-inherited advisory lease; another supervisor
refuses with verified PID/process-start identity. `couchcore.ThreadStore` is
the mutable authority for composite thread records and admission incumbency,
using one global store lock, revision-checked record updates, and a recoverable
write-ahead journal for membership or multi-record changes.

`registry.json` remains as a transitional display/handle cache for the shipped
console. It is not an admission authority. On first load, Couch journal-imports
its actors into ThreadStore as conservative unknown legacy incarnations and
marks the cutover. The operation set remains shared by CLI, console, and the
future advisor.

**The operation set is deliberately not listed here.** `couchcore.Operations()`
is the single source, the CLI dispatches through it, and a test asserts the two
are identical -- so any list in prose is a second copy that drifts. It already
did: this file named six operations while seven shipped. Run `couch --help`,
which renders the declared set.

**couch hosts `pair` whole.** The stack is couch → pair → zellij → agent+nvim.
couch starts `pair resume <tag> --layout2` inside a child pty and owns the
operator tty until the console exits. Verified by operator smoke; the
alternative (couch absorbing zellij's role) was considered and rejected because
the agent child is never spawned by Go — zellij spawns it from a KDL layout, and
`entrypoint.ValidRootMarkers` *defines* a valid pair install as having those
layouts.

**`couch start` IS the console (`pair#146` M2).** It allocates a pty per child,
puts the operator's terminal in raw mode, and routes bytes -- so it no longer
hands the child its own stdio and blocks. The mechanism is shared with `pair term`
rather than written twice: `cmd/internal/ptychild` (a child on a pty, its
bounded replay ring, the #127 query deny-list, one scanner over its output) and
`cmd/internal/hostty` (the operator's terminal: size, raw mode, coalesced
resizes, the control constants). See `atlas/architecture.md`, "The terminal
plumbing is shared with couch".

`--no-console` keeps the stdio-inheriting path, and announces itself loudly. It
is not dead code kept for symmetry: it is the fallback if the tty layer
misbehaves, which is why `ExecRunner` stays a live production path with a live
conformance check behind it.

**The pty is a CAPABILITY on a handle, not a second Runner signature.**
`Runner.Start` is unchanged; a handle from `PtyRunner` additionally satisfies
`TerminalHandle`. `ExecRunner`'s does not, and a test asserts that -- a
capability check no runner can fail is vacuous. `Terminal()` returns the
concrete `*ptychild.Child` rather than an interface, because `FakeRunner`'s
double IS one, so a test takes the branch production takes.

## The reserved row: a reservation, not compositing

The child is given a terminal one row shorter and the host's scrolling region is
pinned above the last row (DECSTBM). The child is never told, so this is a
resize rather than compositing.

Three things that design has to survive, each learned the expensive way:

- **Scrolling.** What DECSTBM is for. A child scrolling at the bottom of its own
  screen scrolls inside the region.
- **Erasing.** DECSTBM does *not* cover it. Every full-screen app clears the
  display on startup and that takes the row with it while the region stays
  intact -- which is why the signal is `Screen.TakeRowDirty` (erase, margin
  reset, RIS, alt-screen transition) rather than anything named for the region.
  The console repaints on it.
- **Not corrupting the child.** A pty read boundary falls wherever the kernel
  puts it, so a paint written between two chunks can land inside one of the
  child's escape sequences. Two rules keep that impossible rather than unlikely:
  **`Console.Run` is the only goroutine that writes to the host** (resizes and
  hotkeys are events it drains, not writers), and every console-originated write
  goes through a gate that defers while the CHILD's stream is mid-sequence,
  paying the debt on the next chunk that ends on a boundary.

  Both halves were learned by getting them wrong. Asking the *child* whether it
  was mid-sequence answered about a later chunk, because ptychild's pump feeds
  its scanner before the console has drained the earlier one -- so the tracking
  belongs to the stream the console WRITES. And feeding the console's own
  escapes into that scanner let it frame our bytes together with the child's
  partial and report "safe" precisely when it was not, so the scanner is fed
  child bytes only.

Verified against a real terminal emulator (`vtscreen_test.go`) and against a
real pty child (`console_live_test.go`, `PAIR_LIVE_COUCH=1`), and confirmed by
operator smoke on the full Ghostty -> couch -> pair -> zellij -> claude stack
2026-08-23.

## Navigation

`ctrl-space` is intercepted before the child sees it. It arrives in TWO
encodings and both are recognised: the legacy `0x00`, and CSI-u
`\x1b[32;5u` under the Kitty keyboard protocol, which zellij enables -- so the
legacy byte is the one a real session almost never sends. The interceptor
returns a SPLIT (bytes for the focus being left, bytes for the focus landed on),
because a concatenated buffer cannot say which child the tail belongs to. It
suspends inside a bracketed paste: a pasted NUL that switched actors and ate a
byte would be untraceable data loss.

The focus ladder is deliberately small: a non-root child goes to the root
actor, the root actor goes to couch's panel, and the panel stays put. Liveness
is consulted before going home so a dead root cannot become a frozen landing.
The stdin pump does not treat a `Read` boundary as an event boundary: after it
finds a hotkey, it waits for the Run loop to acknowledge the focus transition
before routing the suffix. The same stream rule holds for legacy Escape in the
panel — a bare ESC is held briefly because it may be the first byte of a split
arrow sequence; the Run loop's ambiguity timer turns it into an Escape key only
when no continuation arrives.

The panel is couch's own screen. It owns input while visible, suppresses
background-child painting, and has one flat interaction language. Printable
input—including colons and digits—is typeahead; arrows move selection; Enter
forces the selected live actor's clear-and-replay attach path or starts a
selected parked worktree; Escape clears the filter or returns. Ctrl-Space from
the panel opens the start-path input, and Ctrl-Space inside that input is inert.
`start` dispatches through `couchcore.Operations()`; its returned `StartResult`
is load-bearing because the console attaches the new terminal child, rebuilds
the list, and selects its worktree without leaving the panel. Failed starts
retain filter and selection and report through the notice feed.

The row state is three-way: a local-live row has a console routing target and
Enter switches to it; a remote-live row is present in the global summary but
has no local target and reports that #147 transport is required; only a
non-live parked row dispatches `start`. Liveness and local routing capability
are deliberately separate facts (ARCH-PURPOSE).

There is no numbered jump or `:` command state. Tab/thread actions are deferred
to #151 after #149 provides the durable work-thread identity those actions
target; the current panel does not advertise Tab.

A panel row carries two identities that must not be conflated: the canonical
worktree feeds the shared human resolver, while the console-local child id is
the deterministic switch and bell target. `Couch.LookupTrees` is the one match
rule for the panel, CLI and future advisor; it searches the displayed repo-name
fallback, operator name/description, and agent-published description. This is
why a row displayed as `pair` is findable by typing `pair`.

Rows themselves always start at `Couch.Summarize(nil)`, the same durable source
as `couch list`; a pure join adds only hosted-child routing IDs and bell state.
That direction matters: building rows from hosted panes would silently omit a
parked tree and leave successful name/description changes stale in the running
panel.

## Exit, detach, and terminal lifecycle

Every hosted pane retains three identities with separate jobs: the pty handle
routes bytes inside this console, `ActorID` addresses registry persistence and
notices, and the canonical worktree drives transitional human resolution.
They are not interchangeable: both real and fake runners mint a handle ID that
differs from the actor ID.

Each attached child publishes its own exit. If the focused child exits while
others remain, the operator lands on the panel; an inactive exit records the
cause without stealing focus. Either way the dead pane is removed and
`Couch.Forget` removes the registry-cache incarnation. A dead Pair client does
not prove its zellij session quiescent, so M1 retains durable occupancy until
#152 supplies whole-incarnation quiescence evidence. Exit and bell notices
share one bounded `Feed`
over `couchcore.Enqueue`: keys include the actor (`exit:<id>`, `bell:<id>`), so
repeated bells from one actor collapse while two actors remain two obligations,
and exit controls are never discarded for capacity.

Detach inside a live console means focus moved, not process stopped. The child
keeps running and filling its bounded replay ring; returning from the panel and
switching between children use the same clear-and-replay attach path. Beyond a
console process, warmth belongs to zellij's server session plus couch's forced
Pair tag: the console hosts a zellij client, so losing the client loses the view
and a new couch deterministically reattaches. `pair#147` transport is not on
that path.

Console teardown has one owner. Normal stop, last-child exit, SIGTERM, and
SIGHUP all reset the scrolling region, clear the reserved row, leave alternate
screen, restore/show the cursor, restore raw mode, stop host event sources,
close the blocking input seam, and join console workers before returning.
`hostty.TerminationHost` is optional because couch consumes process termination
while the other `hostty.Host` consumer, `pair term`, owns lifecycle elsewhere.

## Spawning: `pair resume <opaque-tag> --layout2`

Every new start first atomically claims a final composite address
`{repo_scope, couch-<16 lowercase hex>}`. Couch resolves normalized fleet
policy and commits a `creating` incarnation before it forks; capacity or
provider refusal therefore starts no child. The creating record then gains one
`start-<16 hex>` nonce plus the exact supervisor identity. Couch forks the
internal `pair-launch-helper`, which cannot exec Pair until Couch durably adds
the helper's PID/process-start identity and sends one acknowledgement byte over
an inherited close-on-exec descriptor. EOF, cancellation, or timeout exits the
helper without starting any workspace writer.

After acknowledgement, Pair changes the same composite address claim from
`reserved` to `established`; that is the registration oracle, not PID liveness
or a successful pipe write. Only then does Couch clear the transaction and mark
the incarnation live. Any post-ack error before `Spawn` successfully transfers
the handle—including an acknowledgement error after the byte may already have
been delivered, registration read failure, promotion conflict, or legacy-
registry save failure—treats exec as possible. Both stdio and PTY runners make
the Pair client the leader of one actor-owned process group; Couch-launched
session-watcher and title-poller sidecars inherit that group rather than
detaching. Couch sends TERM and then unconditional KILL to the group, reaps the
client, and proves the group empty. The remaining process class is the zellij
server and its panes: Couch resolves the exact `{scope, tag}` session-name
binding, observes its record and exact server PID set, deletes and escalates,
then requires two stable observations with both absent. Query, deletion, and
kill errors fail closed rather than becoming quiescence evidence. Only after
whole-incarnation quiescence is proven does Couch reconcile durable state: an
unfinished transaction remains creating or becomes conservative unknown,
while an already-promoted exact incarnation is marked unknown. No error return
can leave an unowned workspace writer. A failed cleanup attempt does not return:
the start call stack retains the handle and retries until it proves absence.
Server escalation carries PID plus kernel start identity and reauthorizes the
identity and exact server argv immediately before signalling.

On supervisor restart, the pure `ReconcileStart` decision
uses exact owner/helper identities plus that registration evidence: dead and
unregistered is proven free and rolls back by nonce+revision, established and
live promotes live, established but gone promotes conservative unknown, and
any unknown evidence stays occupied. The ThreadStore is therefore always
occupied or proven free across every interruption point.

Composite allocation and Pair artifacts share one durable address authority:
`thread-claim-<tag>.json` is created with O_EXCL before either Couch commits the
ThreadStore record or native Pair writes a sidecar/session binding. Couch writes
a reserved claim; only the child carrying the exact scope/tag establishes it.
That reserved → established transition writes and fsyncs a sibling temporary
file, atomically renames it, then syncs the directory, so concurrent recovery
readers observe one complete state and a crash cannot leave truncated evidence.
Direct Pair creates an established claim before its first artifact and adopts
historical tags into the same scheme. Within Pair's owned scope directory, a
generic delimiter rule recognizes any filename carrying the exact tag, so Go,
Lua, layout, and future artifact families need no parallel prefix enum. The
global session binding remains a separately strict check; malformed indexes
fail closed.

The child receives the canonical namespace plus `COUCH_THREAD_SCOPE` and
`COUCH_THREAD_TAG`, and launches as `pair resume <opaque-tag> --layout2`.
Distinct starts at one policy-unbounded path therefore use distinct Pair
sessions and artifacts. Layout stays pinned to layout2: couch owns terminal
switching, so layout3's third pane is the layer couch replaces.

On a COLD create, couch asks Pair to use the repo's saved agent-argument default
without opening `runConfigPicker`; no default means no user-configured args.
Pair consumes the temporary `PAIR_USE_REPO_DEFAULT=1` handoff at process entry
and carries only typed launch policy downstream, so sidecars, zellij, and panes
cannot inherit it. Existing live sessions still take the attach path unchanged.
Direct `pair resume <tag>` still owns the saved-config choice, and direct
`pair -- <agent-arguments>` is the current way to replace the repo default.

## The agent-facing operation

`couch publish-description` is run BY a session, inside its own tree, not by the
operator -- which is why it is the one operation the README does not carry. A
spawned child is told `$COUCH_TREE`, so the agent can name what it is working on
in one line, and `Describe` prefers that sidecar over anything the operator
typed. It is a LABEL, not state: a stale one still finds the right tree, which
is why it is allowed to go stale where a published status document would not be
(see the cold-revival experiment in the project file).

## Identity and admission

The durable address is `ThreadAddress{RepoScope, Tag}`. `RepoScope` is Pair's
existing hidden repository scope; `Tag` is the opaque Pair thread tag. The
canonical physical requested path is an attribute and admission input, not the
thread identity, so Brain-style repositories can host several independent
threads in one directory.

Admission has one source: Ariadne's versioned `sdlc fleet policy --path P
--json` result. Pair strictly decodes and persists the normalized repo identity,
admission key, capacity/action, provider version, and declaration digest with
each occupied incarnation. It never parses declarations or infers policy from
a repo name. Reconciliation performs provider IO outside the ThreadStore lock,
then compare-and-swaps the exact cohort snapshot; live, unknown, and creating
incarnations all count. Client PID evidence never releases capacity because the
zellij session can outlive that client; #152 owns verified quiescence. Mixed
policy epochs retry as a cohort and fail closed after three attempts.

Bounded policy returns either `reject` or the typed `provision-worktree` action;
#149 never creates the path (#153 owns that lifecycle). Unbounded policy admits
multiple threads. The former public same-path override and local policy file no
longer exist. A source-level shadow sweep prevents those parallel authorities
from returning (ARCH-DRY, ARCH-PURPOSE).

`Worktree`, `ActorID`, and `registry.json` remain transitional console
identities. `Worktree` locates display rows and `ActorID` routes a hosted child;
neither decides admission or addresses durable thread artifacts.

## Seams

Everything touching the world is injected, so the domain tests without
processes, disk, wall-clock or randomness. The seam set itself lives in
`Couch`'s struct fields in `cmd/internal/couchcore/couch.go` -- read it there
rather than from a list here, for the same reason the operations are not
enumerated.

The property that matters: each seam has a fake, and the fakes that model
*behaviour* rather than data are compared against the real thing by
`conformance_live_test.go`. `PAIR_LIVE_COUCH=1` checks process/git/pty behavior;
`make test-couch-policy-live SDLC_BIN=/path/to/current/sdlc` checks the stateful
policy fake and strict consumer against Ariadne's real provider, including a
policy epoch transition and typed missing-declaration refusal. The latter runs
on resolver changes plus a weekly/manual workflow. The process check found a
real bug -- `Alive()` reporting a zombie as running -- which no test against the
fake could have. `TestSessionQuiescenceLive`, run by both `make test-live` and
the focused `make test-couch-zellij-live`,
creates and deletes an ephemeral real zellij session through the production
observation seam; a separate macOS workflow runs it on relevant changes and
weekly/manual cadence.

`Runner` was genuinely new — pair has no async process-exec seam.
`launcher.ProcOps` is named for pair's own sidecars, and `wrapcmd` spawns its
child inline and unseamed.

## Liveness is recomputed, never stored

Because `couch start` blocks, every read command runs in a **second process**
with no `Handle`. So `ActorRecord` persists `{PID, Identity}` where `Identity`
is `procutil`'s kernel start token, and a reader recompares it: a recycled PID
reports not-live because the token differs.

Within one process, `ExecRunner` reaps its children in a background goroutine
and liveness is a closed channel — **not** `kill -0`, which succeeds for a
zombie and would report an exited-but-unreaped child as running.

## Actor loop — built, unit-tested, not yet instantiated

`Actor` exists and is tested, but **no command starts one**: `Couch.Spawn`
launches a child and returns. It is groundwork for `pair#147`, where messages
between actors begin to exist. Described here because the design constraints are
the interesting part, not because a running couch has one.

The intended shape is one goroutine per actor, holding a bounded mailbox. `Enqueue` is a pure function
(collapse by kind, drop the oldest non-control entry over capacity, never drop
control) so the policy is testable without goroutines. The loop drains
control-first.

It is a mutex-guarded queue rather than two channels with a priority select: the
bounded/collapse policy must apply at insertion, and a buffered channel cannot
collapse a duplicate already in it. Queries are direct calls behind the same
mutex — Go shares memory, so message passing here buys ordering and decoupling,
not fidelity to Erlang.

## Terminology

- **namespace** — one canonical physical Couch store and its single live
  supervisor lease.
- **thread** — one durable composite `{repo_scope, opaque tag}` record.
- **path** — canonical starting/working location and policy input; not identity.
- **incarnation** — one creating/live/unknown run attached to a thread, with
  verified process identity and normalized policy evidence.
- **actor / ActorID** — the transitional hosted-child/cache identity used by
  the current console; later milestones finish deriving it from ThreadStore.
- **parked thread** — durable historical thread with no verified live
  incarnation; #152 owns the proof and age semantics.

## Planned, not built

`pair#149` M2-M5 finish recoverable start, shared inventory/naming, launch
profiles, and artifact migration. `pair#151` adds the hierarchical thread menu;
`pair#152` verified park/age; `pair#153` managed-worktree lifecycle; `pair#147`
cluster transport and queries; `pair#148` brain as advisor. Cross-repo enabler
`ariadne#199` exposes the query API. Ariadne #200's normalized policy provider
is implemented and consumed at the #149 M1 boundary.
