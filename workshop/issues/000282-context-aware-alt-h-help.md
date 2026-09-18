---
id: 000282
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T09:31:36-07:00
---

# Alt+h help knows whether it runs under couch, and shows couch's keys there

## Problem

Alt+h (`PairOpenHelp`, `nvim/init.lua:3469`) shows `pair keys`
(`cmd/internal/keyscmd`), which describes pair's workbench and nothing else. Run
under couch, it is the only in-session help the operator has, and it is silent
about the layer they are actually standing in:

- **couch's own keys are absent.** Ctrl+Space (switcher), Ctrl+Backspace
  (previous thread), Ctrl+Return (newest notification), the switcher's lifecycle
  chords, and the terminal-tab keys couch reserves are documented only in
  `couch --help` — a command nobody runs from inside a thread.
- **Some of pair's own entries are wrong under couch**, and this is the part a
  simple "append couch's section" would miss. Couch intercepts chords before pair
  sees them: `couchtty/keys.go:228` keeps alt+d, alt+x, alt+n and ctrl+alt+n as
  couch's lifecycle chords, and
  `TestRelaunchChordsAreInterceptedAndAltShiftNIsNot` pins that alt+n is couch's
  while alt+shift+n still reaches pair. So pair's help currently tells a couch
  user:
  - alt+d — *"detach from the session (re-attach with `pair`)"*: under couch the
    way back is the switcher, not `pair`.
  - alt+n — *"reload pair — kill and re-launch the workbench in place"*: under
    couch this is couch's relaunch of the thread's pair process.

  The help is describing a different program than the one receiving the keys.

## Spec

Alt+h renders help for **the stack the operator is in**: pair alone, or pair
under couch.

### Detection

Two different questions, and the issue should answer both deliberately:

- **Hosted by couch** — the thread was launched by couch. The pair process
  environment already says so: `COUCH_THREAD_SCOPE`, `COUCH_THREAD_TAG`,
  `COUCH_THREAD_RESUME`, `COUCH_TREE`, `COUCH_STORE_DIR` are set in a couch
  thread's panes (verified 2026-09-17 in the agent pane; confirm the draft's
  nvim sees the same before relying on it). The launcher already branches on the
  first two (`launcher/runcli.go:117`).
- **Couch is presenting it right now** — a different fact. The env is written at
  launch and never revised. If couch exits, the zellij session survives (the
  server is PPID 1 at birth, `pair#275`), can be attached directly, and still
  carries the env — so the help would list Ctrl+Space for a couch that is gone.
  That is the same stale-witness shape as `pair#272`, in miniature.

Recommended: env decides *hosted*; the couch supervisor's singleton lease
decides *live*. `pair#278` is already adding a read of that lease to print the
supervisor's pid — reuse that read rather than inventing a second liveness
check. When hosted-but-not-live, say so in one line instead of listing keys that
will not work.

### Content — derive, don't copy

- **Couch's section comes from couch's key table.** `couchtty.CouchNavigationBindings()`
  already carries label + description per binding, and `couch --help` renders
  from it (`couchcmd/run.go:812`). Alt+h must consume the same source, so the two
  surfaces cannot drift. No hand-written copy of couch's keys in pair.
- **Pair's section already derives from its sources** (`keyhelp`, reading
  `nvim/init.lua`, `zellij/config.kdl` and the `workbenchshortcut` table, with
  drift tests). Keep that; don't fork it.
- **Overridden entries are corrected, not merely supplemented.** For each chord
  couch intercepts, pair's entry under couch shows couch's meaning (or is marked
  as couch's), so the page never lists two meanings for one chord. Derive the
  intercepted set from couch's chord table, not a list maintained in pair.
- **The switcher's own chords** (alt+d detach, alt+x park — "in an actor or in
  the switcher alike", `couchcore/park.go:111`) belong in couch's section. Note
  `pair#279`: alt+d is currently dead in the switcher; the help should describe
  the contract, and #279 makes it true.

### Layering

`pair keys` lives in pair and couch is a peer consumer of pair, so this is a
dependency question worth settling rather than inheriting: pair importing
`couchtty` just to read a key table may be the wrong direction. If it is, the
binding table moves to a small shared package both depend on, the way
`workbenchshortcut` already serves both today.

## Done when

- [ ] Alt+h under couch shows couch's keys, rendered from
      `CouchNavigationBindings()` — the same source `couch --help` uses — proven
      by a test that adds a binding and sees both surfaces change.
- [ ] Chords couch intercepts show couch's meaning under couch; no chord appears
      with two meanings. Derived from couch's chord table.
- [ ] Standalone pair's help is unchanged.
- [ ] Hosted-but-couch-not-running is detected and stated, not rendered as live
      couch keys. Decision on the detection signal recorded (env vs lease).
- [ ] `pair keys` from a shell reflects the same context as Alt+h.

## Plan

Durable plan: `workshop/plans/000282-context-aware-alt-h-help-plan.md`.

- [x] Confirm the draft nvim's environment carries `COUCH_THREAD_*`, and what
      `PairOpenHelp` passes to `pair keys`.
- [x] Settle the package direction for sharing couch's binding table.
- [ ] `couchkeys`: Couch's chord table as data, with scope (plan Task 1).
- [ ] couchtty frames and routes from `couchkeys` (Task 2).
- [ ] `GlobalBinding.HostedHelp` for Alt+d, Alt+n, Ctrl+Alt+n (Task 3).
- [ ] `keyhelp.Page` / `CouchSections` / `Presence` (Task 4).
- [ ] `couch --help` renders via `keyhelp.CouchSections` (Task 5).
- [ ] `launcher.CouchHosted`: one hosted rule (Task 6).
- [ ] `keyscmd.ProbePresence` + `RunWith`; hermetic tests (Task 7).
- [ ] README + atlas sweep; full `make test` + `go test ./...`; behavior
      evidence; operator smoke (Task 8).

## Log

### 2026-09-17

- Filed from a brain advisor session at the operator's request: *"it should be
  aware if we are in pair or couch, and when in couch, to also display help for
  couch."* No existing issue; nearest are #232/#233 (terminal-specific Alt
  bindings), unrelated.
- The overridden-entry problem (alt+d, alt+n) and the hosted-vs-live distinction
  were found while scoping; neither was in the request, both follow from it.

### 2026-09-18

- Claimed. Measured in a live Couch thread (this session):
  - The draft nvim carries all five `COUCH_*` vars (`ps eww` on
    `$PAIR_NVIM_DRAFT_PID_PATH`'s pid).
  - `$COUCH_STORE_DIR/supervisor-owner.json` names a running `bin/couch`.
  - `PairOpenHelp` passes nothing: `bin/pair-help` runs
    `pair keys --center <cols>` in a `zellij run` pane, so context arrives only
    through the Zellij server's env.
- **The premise about interception is stale since #245.** `knownSequences`
  still frames alt+d/x/n and ctrl+alt+n, but
  `Console.dispatchInputCandidate` forwards every non-`actorReserved` hit to the
  displayed Pair pane. `TestActorLifecycleCandidatesPassThrough` and
  `TestConsoleRunAltDActorInputDoesNotDispatchDetach` pin that. Those chords
  are Couch's only in the switcher. See Revisions.
- Found `pair#284`, filed from code reading. Since #249, `pair restart` refuses
  when hosted, so Pair's Alt+n in a Couch pane confirms and then does nothing.
  The help documents that truth, and #284 owns the fix.
- `#278` has not landed code, but the lease read it will use already exists as
  `couchcore.VerifiedOwner`. It is reused here (ARCH-DRY).
- Package direction: a new `couchkeys` package (pure data, depends only on
  `workbenchshortcut`). Pair must not import Couch's console.
- Full design and ARCH lenses: in the plan file.

## Revisions

### 2026-09-18: the mechanism behind "overridden entries" changes; the purpose holds

**Reason:** the Spec assumed Couch intercepts alt+d, alt+x, alt+n and ctrl+alt+n
before Pair sees them. #245 (2026-09-14) made those chords pass through to the
displayed Pair pane. Couch acts on them only in its switcher. Couch takes only
Ctrl+Space, Ctrl+Backspace and Ctrl+Return from a Pair pane. So the "wrong under
couch" entries are wrong because *Pair's own* behavior changes when hosted:
- `pair restart` is refused when hosted (#249), so Alt+n and Ctrl+Alt+n do
  nothing (`pair#284`).
- Pair's Alt+d detaches only its Zellij client, while Couch's detach lives in
  the switcher.

Couch is not taking those keys over.

**Delta to `## Spec`:**
- Couch's section has two scopes:
  - every-pane chords (taken before Pair);
  - switcher chords (acted on only in the switcher).

  Each scope renders as its own context. The same key in a Pair pane and in the
  switcher is two rows in two sections, per keyhelp's existing `(key, context)`
  rule.
- The table the help derives from is a new `couchkeys` package: labels, help,
  encodings and scope for all seven chords. It replaces
  `couchtty.CouchNavigationBindings()`, which covered only the three navigation
  chords and had no help text for the switcher chords. couchtty's framing and
  routing read the same table, so help context and routing cannot disagree.
- Overridden entries are corrected through `GlobalBinding.HostedHelp` (Alt+d,
  Alt+n, Ctrl+Alt+n). This is Pair-authored, because it describes Pair's
  behavior. It is keyed on *hosted* (launch env), not on Couch being live,
  because the launcher's refusal is keyed on the env.

**Delta to `## Done when`:**
- Row 1: "rendered from `CouchNavigationBindings()`" → "rendered from
  `couchkeys.Bindings()` through `keyhelp.CouchSections`, which `couch --help`
  also renders".
- Row 2: "No chord has two meanings **in one context**. Couch's every-pane
  chords are disjoint from Pair's (tested by encoding and by display key). Pair
  entries whose behavior changes when hosted show their hosted meaning." The
  intercepted set still derives from Couch's table, via `Scope`.
- Rows 3–5: unchanged.
