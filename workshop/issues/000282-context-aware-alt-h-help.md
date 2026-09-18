---
id: 000282
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 5.7
started: 2026-09-18T09:31:36-07:00
flow: {kind: full, provenance: inferred}
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

Design includes what the claim window already holds: the spec revision after the
#245 premise correction, and the operator's two redirects. Impl is 40% of the v2
ranges. The design buffer is +15%, because a thorough plan doc exists.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.0  impl=0.05
item: ux-rename-iteration     design=0.5  impl=0.1
item: ux-rename-iteration     design=0.5  impl=0.1
item: scope-pivot             design=0.3  impl=0.15
item: smaller-go-module       design=0.1  impl=0.2
item: smaller-go-module       design=0.1  impl=0.2
item: greenfield-go-module    design=0.5  impl=0.3
item: cross-cutting-refactor  design=0.3  impl=0.2
item: smaller-go-module       design=0.05 impl=0.15
item: atlas-docs              design=0.1  impl=0.08
item: milestone-review        design=0.0  impl=0.2
design-buffer: 0.15
total: 5.70
```

Rows, in order: spec, plan and three plan reviews; redirect 1 (interception);
redirect 2 (draft-only, Pair's pager); the #245 premise correction; launcher
(`CouchHosted`, outer record); keyhelp + `HostedHelp`; `couchkeys`; the
couchtty derivation + `couch --help`; keyscmd + pane title; README/atlas; the
close boundary review.

## Plan

Durable plan: `workshop/plans/000282-context-aware-alt-h-help-plan.md`.

- [x] Confirm the draft nvim's environment carries `COUCH_THREAD_*`, and what
      `PairOpenHelp` passes to `pair keys`.
- [x] Settle the package direction for sharing couch's binding table: the
      pure `couchkeys` package; Pair imports it, never couchtty (third
      revision).
- [x] `launcher.CouchHosted`: one hosted rule (plan Task 1).
- [x] The attaching client records whether Couch presents it (outer-tty
      record, `PresentedByCouch`, `ReadOuterPresenter`) (Task 2).
- [x] `GlobalBinding.HostedHelp` for Alt+d, Alt+n, Ctrl+Alt+n (Task 3).
- [x] keyhelp: `Binding.Chord`, `HostedSections`, `Layer` (Task 4).
- [x] `couchkeys`: Couch's chord table as data, with scope (Task 5).
- [x] couchtty frames and routes from `couchkeys`; `couch --help` renders it
      (Task 6).
- [x] `pair keys` composes the page; pane title "help" (Task 7).
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
- Fresh-context plan review of that first draft: Issues Found, all fixable. The
  reviewer built every snippet in a scratchpad copy and measured standalone
  output byte-identical and the live/absent pages correct.
- **Operator redirect before approval** (see the second Revisions entry): the
  env records the session's *creator*, so a session Couch adopts later (#246)
  would read as standalone. The new direction: Couch takes Alt+h itself and
  shows its keys over Pair's structured rows in a panel frame (the operator
  chose that over Pair's floating pager). Plan rewritten; the first review's
  findings that still apply are folded in.
- **Second operator redirect** (see the third Revisions entry): Couch must not
  take keys from panes it cannot tell apart. Alt+h stays Pair's, and Pair's
  pager shows the combined page. Measured the per-attach outer-tty record in
  this session: `$PAIR_DATA_DIR/outer-tty-$PAIR_TAG` = `/dev/ttys006`, written
  08:52, the moment this Couch started. Nothing reads the record except cleanup
  and GC. Plan rewritten a third time. The review of the second draft was
  stopped as obsolete, and its scratch worktrees were removed.
- Floating pane title: 'pair help' → 'help' (operator request).
- `sdlc change-code`:
  - Plan-quality: no blocking findings; three advisories, folded into the plan
    (a fuzz test for the record parser, the corrected ARCH-SECURE claim, the
    Alt+Shift+C note, and content anchors).
  - Estimate-quality: info. It says the design rows front-load time already
    spent.
  - Branch `000282-context-aware-alt-h-help`, in place.
- Implemented Tasks 1–8 (`dbc489ba`..`9d0e257c`). Evidence:
  - `TestCouchClientRefusesRestartMarker` pins the client-side Alt+n gate.
    Mutation check: with the `createflow.go` guard disabled it fails with code
    0 instead of 1; the file was restored.
  - The couch-presented attach records `live|true` and a terminal attach
    records `…|false` (fake runtime).
  - A non-tty attach removes the record, tested on the real filesystem.
  - `FuzzDecodeOuterRecord` ran for 10s: 539k execs, no failures.
  - `make -k test`, with the five-var env scrub and unsandboxed: the only
    failure is `test-changelog`, the known pre-existing
    `viewer: process target is outside selected owner directory`.
  - `go test ./... -count=1`, same scrub, unsandboxed: 72 packages ok, exit 0.
    The first run caught `TestNoDeclarationCarriesTwoStackedGodocs` on the
    `Sections` doc; fixed in `9d0e257c`.
  - `bash tests/workbench-route-nvim-test.sh` ok, with the pane-name pin
    updated.
  - Standalone `pair keys` is byte-identical to `origin/main` (48 lines, empty
    stderr), from a throwaway worktree with the bundle generated.
  - `pair keys` takes about 6 ms per run.
  - Simulated `presenter=couch` record: Couch's two sections lead, then Pair's
    rows with the hosted Alt+d/Alt+n wording.
  - In this session the record is still the pre-change one-line form, so the
    page has hosted wording and no Couch section. It needs the operator smoke
    after a relaunch.
- Pending: the operator smoke on a restarted Couch (Task 8 Step 5).
- Third plan review: Issues Found. The reviewer built all 8 tasks in a scratch
  worktree, and standalone output was byte-identical. Blocking findings, all
  folded into the plan:
  - `tests/workbench-route-nvim-test.sh:167` pins the pane name.
  - `pair keys` would print a stderr error outside a session.
  - Alt+n in an adopted, Couch-presented thread ends the thread (see the
    correction in Revisions).
  The reviewer removed its scratch worktrees.

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

### 2026-09-18: Couch intercepts Alt+h; detection is dropped

**Reason:** operator direction, before plan approval. The first Revision kept the
Spec's detection model: env decides *hosted*, the supervisor lease decides *live*.
The env records who *created* the Zellij session, so a session Couch attaches to
after creation (#246) would read as standalone. The operator's alternative:
- Each layer binds its own Alt+h.
- Pair exposes structured key→text rows, not finished text.
- Couch adds its keys and overrides the ones it takes, and displays the result
  itself.

A keystroke Couch intercepted proves Couch is presenting this client, so no
detection is needed. The operator picked a Couch panel frame over Pair's floating
pager.

**Delta to `## Spec`:**
- *Detection* is replaced by interception. Couch claims Alt+h from every pane,
  including the agent pane, and in the switcher. The lease is not read.
- *Content*:
  - Pair's `keyhelp` rows carry the typed workbench `Chord`.
  - `keyhelp.Layer(host, claimed, pair)`, which is host-agnostic, puts a
    host's sections first and drops Pair's rows for claimed chords.
  - Couch's table (`couchtty`, with `KeyScope`) feeds framing, routing,
    `couch --help` and the page.
  - Pair's hosted wording (`HostedHelp`) is chosen by Pair's own rule
    (`launcher.CouchHostedEnv`) on Pair's page, and always on Couch's page.
- *Layering*: Pair never reads Couch's table, so no shared package is needed.
  Couch imports keyhelp.

**Delta to `## Done when`** (replaces the first revision's rows):
1. Under Couch, Alt+h (in any pane, and in the switcher) opens Couch's help
   frame, which shows Couch's keys over Pair's. Couch's part comes from the
   table `couch --help` renders, and a binding added there reaches both
   surfaces.
2. A chord Couch takes replaces Pair's row for it, and no key has two meanings
   in one context. Pair entries whose behavior changes in a Couch-launched
   thread show hosted wording.
3. Standalone Pair's help is unchanged.
4. Couch's keys appear only when Couch presents the thread, which interception
   guarantees. Pair's page in a Couch-launched thread says where they are.
5. `pair keys` prints exactly Pair's own Alt+h page. The combined page lives in
   Couch.

**Known gap, recorded in #284:** for a thread Couch adopted without launching,
Pair's Alt+n is not refused, but Couch's page shows the hosted row. #284 owns
Pair's Alt+n under Couch.

### 2026-09-18: Alt+h stays Pair's; Pair's pager shows Couch's layer

**Reason:** operator direction, before plan approval. Couch must not take keys
from panes it cannot tell apart (the agent pane, the right pane), and it keeps
no inner-focus state (#245). So Alt+h fires in the draft, and in the right
terminal through Pair's existing routing, which the operator chose to keep. The
agent pane keeps receiving it. Pair's pager displays the combined page; the
operator chose this over a Couch panel frame reached through a new Pair→Couch
channel. `pair keys` from a shell is not a supported surface. The pane title
becomes "help".

**Delta to `## Spec`:**
- *Detection*: two facts, each from its own authority.
  - **Presenter**: the attaching Pair client knows who launched it, because
    Couch sets `COUCH_THREAD_TAG` on every client it presents, including warm
    reattaches of sessions it did not create. The client writes that into the
    per-attach outer-tty record (`PresentedByCouch`), and `pair keys` reads it.
    Last attach wins.
  - **Hosting**: Pair's own refusal rule (`CouchHostedEnv`) selects the
    hosted wording, so a row is true of the key.

  The supervisor lease is not read. Couch does not intercept Alt+h.
- *Content*: Couch's table moves to the pure `couchkeys` package. couchtty's
  framing and routing, `couch --help`, and Pair's page derive from it.
  `keyhelp.Layer` drops Pair rows for chords Couch claims from every pane (none
  today; the mechanism is there for future overrides).
- *Layering*: `couchkeys` is the small shared package the Spec anticipated.
  Pair imports it, never Couch's console.

**Delta to `## Done when`** (replaces the second revision's rows):
1. When Couch presents the thread, Alt+h (in the draft, and in the right
   terminal through Pair's routing) shows Couch's keys above Pair's. They come
   from the table `couch --help` renders, and a binding added there reaches
   both.
2. A chord Couch claims replaces Pair's row, and no key has two meanings in one
   context. Pair's rows that hosting changes show hosted wording by Pair's own
   rule.
3. Standalone Pair's Alt+h is unchanged apart from the pane title.
4. Couch's keys appear if and only if the attached client was launched by Couch
   for this thread. That includes adopted sessions and excludes a terminal
   reattach after Couch exits.
5. Couch does not take Alt+h; the agent pane receives it.

**The previous revision's known gap is closed:** hosted wording now follows
Pair's refusal rule rather than "Couch is presenting", so an adopted session
shows "reload pair", which is true.

### 2026-09-18: correction — hosted wording is "Couch launched the session or presents this client"

**Reason:** the third plan review measured that Pair's Alt+n is gated twice:
`pair restart` checks the *session* env (`runcli.go:133`), and the attached
*client* refuses the restart marker when its own env names Couch
(`createflow.go:162`), after it has already run the full quit cleanup
(`createflow.go:129`). The previous entry's claim that "an adopted session shows
'reload pair', which is true" is therefore false. In an adopted,
Couch-presented thread, Alt+n ends the thread without relaunching it.

**Delta:**
- Hosted wording is selected by `CouchHostedEnv(session env) || couchPresents`.
- Alt+n's hosted text warns instead of saying "refused": "does not reload under
  Couch and may end the thread; relaunch from the Couch switcher".
- The client-side refusal gets a test (`TestCouchClientRefusesRestartMarker`).
- The destructive adopted case moves to #284.
- Done-when row 2 now reads "…show hosted wording when Couch launched the
  session or presents the client".
