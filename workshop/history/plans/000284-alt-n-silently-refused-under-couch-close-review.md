# Boundary Review — pair#284 (whole-issue close)

| field | value |
|-------|-------|
| issue | 284 — Pair's Alt+n in a Couch-hosted thread confirms, then silently does nothing |
| repo | pair |
| issue file | workshop/issues/000284-alt-n-silently-refused-under-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 62e4e9c7781f6d7436f0ae32ad61750df22acb62..10010a05bfffbb666b0833ff09d28a3263933b25 |
| command | sdlc close --issue 284 |
| reviewer | claude |
| timestamp | 2026-09-19T08:20:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff delivers the issue's stated direction: Couch now claims Alt+n / Ctrl+Alt+n at `ScopeEveryPane` and reuses the existing `onRelaunchHotkey` actor arm (no new handler), Pair refuses an in-session restart *before* any marker/quit-intent/kill via one shared rule (`CouchOwnsRestart` + `couchRestartGate`) that `pair keys` also reads for its hosted wording, and the draft's four confirmed lifecycle commands now report a non-zero exit through one helper. I verified the substance rather than the prose: I ran the touched packages' tests (green, except pre-existing pty/tmp sandbox failures unrelated to this diff), and mutation-checked all three gates — reverting the `ScopeEveryPane` change reds `TestActorRelaunchChordsConfirmTheThreadOnScreen` (both encodings time out waiting for the confirmation), and deleting either `couchRestartGate` call reds the launcher suite (with the hosted case actually reaching a real `kill-session`, which is exactly the destructive path the gate exists to stop). I also independently re-ran the class sweeps the plan claims: `WriteRestartMarker` has exactly two in-session callers (both gated), `vim.fn.confirm` has exactly five draft sites (the fifth, `PairConfirmCompact`, shells nothing), `ChordAltN` has no legacy `\x1bn` encoding so no Esc-then-n ambiguity is introduced, and the runtime-bundle mirror carries both the new module and the wired `init.lua`. What keeps this from a bare SHIP is small and cheap: the compaction refusal strands a validated checkpoint while naming an operation that discards it, and the draft wiring that *is* the third Done-when row has no test that goes red if a call site reverts to `vim.fn.system`.

## 1. Strengths

- `cmd/internal/launcher/outerrecord.go:57-84` — the rule is stated once as a pure two-bool predicate and the gate is thin glue over the injected seam. `pair keys` (`keyscmd.go:89`) derives the hosted wording from the same function, so the help cannot promise a reload the gate refuses (ARCH-PURPOSE's shadow sweep actually done, not claimed).
- `cmd/internal/launcher/createflow_test.go:387-397` — `fakeRuntime.OuterPresenter` reads back what `RecordOuterTTY` wrote, plus a `presenterErr` arm. That's a stateful double behind the production seam rather than a stub reasserting the implementation (ARCH-MOCK).
- `cmd/internal/launcher/restart_test.go:156-199` — the five-case table pins the whole rule *and* the "no mutation" property (`markers/quit/killed` all empty), including the fail-closed unreadable-record case and the two cases that must still restart. Mutation-checked: it goes red without the gate.
- `cmd/internal/couchkeys/couchkeys.go:87-101` — the scope became a parameter of the existing `pairChord` constructor, so routing (`actorReserved` reads the declared scope), the help section, and `Claimed`/`keyhelp.Layer` all followed from a one-word table edit with no edit outside the table. `keys_test.go:717` pins scope↔reservation per binding, so a same-action mixed-scope declaration can't silently take the first row.
- `workshop/lessons.md:5592-5606` — the "sweep a chord's wire bytes tree-wide, cross-layer tests sit in the receiving package" rule is the generalization of the `wrapcmd` miss, not a note about the one test that broke.

## 2. Critical findings

None.

## 3. Important findings

**a. `cmd/internal/launcher/compaction.go:105-107` — the compaction refusal strands a validated checkpoint and names an operation that discards it.**
By the time the gate fires, `opts.ContinueCheckpoint` has been read and validated from a durable doc. Every other failure arm in this function names what was retained and how to resume it (`:97` "checkpoint kept at %s", `:122`/`:129` "retry with pair continue --retry %s"); this one prints only `pair: compaction: this session's restarts belong to Couch; relaunch the thread from Couch (Alt+n)`. Couch's Alt+n keeps the conversation and ignores the checkpoint, so the operator who asked to compact is pointed at a different operation and never told where their continuation went. Fix sketch: `fmt.Fprintf(stderr, "pair: compaction: %v; checkpoint kept at %s\n", err, opts.ContinueCheckpoint.SourcePath)` — and consider a compaction-specific tail naming the retry route rather than the relaunch one.

**b. `nvim/init.lua:3088,3098,3194,3217` — the four wiring sites that satisfy Done-when "a failed quit/detach/restart/agent-restart shows its error" are untested.**
`nvim/lifecycle_command_test.lua` tests the module in isolation; nothing asserts that `PairConfirmQuit` / `PairConfirmDetach` / `pair_confirm_restart_impl` / `PairConfirmAgentRestart` actually route through `_G._pair_lifecycle.run`. Revert any one of them to `vim.fn.system` and the suite stays green — the precise regression #284 exists to prevent (a confirmed lifecycle keybind that swallows its failure). The repo already drives the real `init.lua` headlessly (`Makefile.local:282-341`); stubbing `_G._pair_lifecycle.run` there and asserting each entry point calls it is a few lines.

## 4. Minor findings

- `cmd/internal/launcher/compaction.go:105` — the hardcoded `false` for `sessionEnvHosted` is correct only because the `opts.Env.CouchHosted()` arm above always returns; pass `opts.Env.CouchHosted()` so the gate survives that arm gaining a fall-through.
- `cmd/internal/launcher/outerrecord.go:78` — the unreadable-record refusal names no remedy (which artifact, or that reattaching through `pair` rewrites it), so a corrupt record makes standalone Alt+n unfixable-looking.
- `cmd/internal/keyscmd/keyscmd.go:90` — garbled comment: "The rule `pair restart` refuses by (#284), so…".
- `nvim/lifecycle_command.lua:19` — reads the `vim` global for `vim.log.levels.ERROR` while taking `notify` as a dep; the sibling `nvim/confirm_quit.lua` touches no global.
- `atlas/architecture.md:816` — the four-confirm-modals entry still describes them as plain shell-outs; it doesn't mention the new "each reports its own failure" rule or the `nvim/lifecycle_command.lua` seam. (The hosting note at `:320-330` and the paste note at `:487` were updated correctly.)
- `cmd/internal/couchkeys/couchkeys.go:92` — "relaunch this thread" reads oddly for the switcher scope, where the target is the highlighted row rather than a thread on screen.

## 5. Test coverage notes

- Mutation-checked here, not taken on trust: scope revert → `TestActorRelaunchChordsConfirmTheThreadOnScreen` red (both encodings); gate removal → `TestRunRestartRefusesACouchOwnedSessionBeforeMutation`, `TestRunLaunchCompactionRefusesACouchPresentedSession` and `TestCheckpointHostedRestartAndRenameRefuseBeforeMutation` red. The working tree was restored byte-for-byte after each.
- `TestRunLaunchCompactionRefusesACouchPresentedSession` covers only the presented case; the unreadable-record branch is exercised only through `runRestart`. Acceptable (shared helper), noted for completeness.
- `TestNoPairRowSharesAnEveryPaneCouchKey` now reads the layered page — it still catches a `keyhelp.Layer` that fails to drop a claimed row, so the narrowing is sound, not a weakening.
- The new couchtty test deliberately drops the old test's trailing-bytes assertion; that's correct, since interception moves focus to the panel and later bytes must *not* reach the child.
- Environment: `TestNotificationPTYConformance`, `TestCouchProductionSoak`, the `wrapcmd` retention tests and `TestPairHelpShimInvokesPairKeys` fail here with "operation not permitted" (pty-child / `/tmp` restrictions in this shell), not from this diff. I could not reproduce a full `make test`; the Log's green claim rests on the implementor's unsandboxed run.

## 6. Architectural notes

Walked each marker: **ARCH-DRY** pass (scope became a parameter of the existing constructor; one predicate serves gate + help; the relaunch handler reused unchanged — the only nit is two access paths to the presenter fact, both funnelling through `ReadOuterPresenter`). **ARCH-PURE** pass (`CouchOwnsRestart` pure over two bools, IO confined to the `Runtime` seam; minor Lua global noted). **ARCH-PURPOSE** pass with the two Important findings above — the enumerable classes (marker writers, confirm sites) were genuinely swept, and I re-derived both enumerations independently. **ARCH-MOCK** pass (read-back fake + real-seam decode tests). **ARCH-CONSTRAINTS** pass (keystroke path gains a ~7-entry table walk; the file read sits behind a modal, not on input). **ARCH-SECURE** pass (untrusted persisted record, strict parse, fails closed on the destructive path and open on help, both stated and tested; pre-#282 one-line records still decode). **ARCH-ORDER** pass (no new state between events; the relaunch outcome machine reused; scope↔reservation pinned per binding). **ARCH-FUNERAL** pass (nothing durable created; the bundle mirror is regenerated per build).

Forward-looking: the client-side refusal at `createflow.go:162` still fires *after* `runCleanup`, so a restart marker arriving from any writer outside the two gated ones (an older binary, a hand-written marker) still tears the thread down before being refused. This diff correctly gates the sources rather than reordering that, but when #246 (adoption) lands, the adopted case stops being latent and that ordering becomes the last unguarded edge — worth an issue rather than scope creep here.

## 7. Plan revision recommendations

None — the Plan's four implementation rows match the code, and the Spec's design (scopes, the predicate, the `Runtime` seam, the gated sites, the draft helper, the exclusions) is what shipped. If finding (b) is deferred rather than fixed, add a `## Revisions` entry recording that Done-when row 6 is delivered but its wiring is unpinned by tests.

```findings
findings:
  - id: new
    severity: Important
    family: refusal-names-the-recovery-route
    title: |
      Compaction's Couch refusal strands the validated checkpoint and names an operation that discards it
    detail: |
      cmd/internal/launcher/compaction.go:105-107 refuses after ContinueCheckpoint
      is read and validated, but prints only the generic "relaunch the thread from
      Couch (Alt+n)" body. Every other failure arm in the same function names the
      retained checkpoint path and the `pair continue --retry <tag>` route (:97,
      :122, :129). Couch's relaunch keeps the conversation and ignores the
      checkpoint, so the operator is pointed at a different operation and never
      told where their continuation went.
  - id: new
    severity: Important
    family: glue-wiring-untested
    title: |
      The four draft call sites that deliver the notify-on-failure Done-when row have no failing test
    detail: |
      nvim/lifecycle_command_test.lua tests the module in isolation; nothing
      asserts PairConfirmQuit / PairConfirmDetach / pair_confirm_restart_impl /
      PairConfirmAgentRestart route through _G._pair_lifecycle.run
      (nvim/init.lua:3088, 3098, 3194, 3217). Reverting any one site to
      vim.fn.system leaves the suite green -- the exact class of bug this issue
      exists to kill. Makefile.local:282-341 already drives the real init.lua
      headlessly, so stubbing the helper and asserting each entry point calls it
      is a few lines.
  - id: new
    severity: Minor
    family: constant-restates-a-caller-invariant
    title: |
      couchRestartGate is called with a hardcoded false for sessionEnvHosted in runCompaction
    detail: |
      cmd/internal/launcher/compaction.go:105 is correct only because the
      opts.Env.CouchHosted() arm above always returns. Passing
      opts.Env.CouchHosted() costs nothing and keeps the gate correct if that arm
      ever gains a fall-through.
  - id: new
    severity: Minor
    family: refusal-names-the-recovery-route
    title: |
      The unreadable-record refusal names no remedy
    detail: |
      cmd/internal/launcher/outerrecord.go:78 fails closed (correct) but tells the
      operator nothing about which artifact is unreadable or that reattaching
      through `pair` rewrites it, so a corrupt record makes a standalone Alt+n
      look permanently broken.
  - id: new
    severity: Minor
    family: comment-accuracy
    title: |
      Garbled comment in keyscmd's hosted-wording branch
    detail: |
      cmd/internal/keyscmd/keyscmd.go:90 reads "The rule `pair restart` refuses by
      (#284), so the hosted wording is the true one".
  - id: new
    severity: Minor
    family: module-purity-deps
    title: |
      lifecycle_command.lua reads the vim global while taking notify as a dep
    detail: |
      nvim/lifecycle_command.lua:19 uses vim.log.levels.ERROR directly; the
      sibling nvim/confirm_quit.lua touches no global, so the module can only run
      under `nvim -l`.
  - id: new
    severity: Minor
    family: docs-follow-new-surface
    title: |
      atlas's confirm-modals entry does not mention the new failure-reporting rule
    detail: |
      atlas/architecture.md:816 still describes the four modals as plain
      shell-outs, with no mention of the shared nvim/lifecycle_command.lua seam or
      the "every confirmed lifecycle command reports its own failure" rule. The
      hosting note (:320-330) and the paste note (:487) were updated correctly.
```

---

## Re-review — 2026-09-19T08:36:06-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 284 — Pair's Alt+n in a Couch-hosted thread confirms, then silently does nothing |
| repo | pair |
| issue file | workshop/issues/000284-alt-n-silently-refused-under-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 62e4e9c7781f6d7436f0ae32ad61750df22acb62..d6f2821631051bd97d11f0a5d0186fb391bb7511 |
| command | sdlc close --issue 284 |
| reviewer | claude |
| timestamp | 2026-09-19T08:36:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This round fixed six of the seven prior findings, and I checked each against the code, tests and docs rather than the commit message. The main change is sound. Couch now takes Alt+n and Ctrl+Alt+n from every pane. One predicate, `CouchOwnsRestart`, decides both the refusal gate and the hosted wording in `pair keys`. Both code paths that write a restart marker from inside a session are gated before any change is made. A headless test now pins the draft's four lifecycle call sites. One thing blocks a clean SHIP: **BR-1's fix names a recovery command that fails**. The compaction refusal tells the operator to run `pair continue --retry <tag>`. That command needs a saved restart marker, and this refusal fires before any marker is written. A scratch test confirmed it: after the refusal, `prepareContinuationRetry` returns "no retained continuation restart intent". The fix is cheap.

**1. Strengths**
- `launcher/outerrecord.go:57-85`: one rule backs both the enforcement and the help page, so Alt+h cannot promise a reload the gate refuses (ARCH-PURPOSE). An unreadable record refuses a restart but leaves `pair keys` on Pair's own page. That split is deliberate and each side documents it (ARCH-SECURE).
- Both restart-marker writers are gated, and there are only two: `git grep WriteRestartMarker(` finds just `restart.go:46` and `compaction.go:121`. The test table in `restart_test.go:161-199` covers five cases: adopted, created by Couch, unreadable record, terminal-presented and no record. For the refusals it also checks that no marker, quit intent or kill happened.
- BR-2's fix is real. `tests/lifecycle-command-nvim-test.sh` drives the real `init.lua`. I reverted the detach call site in a scratch copy and got 2 FAILs; reverting the restart call site also gave 2 FAILs. It runs as part of `make test`.
- `couchkeys.pairChord` takes the chord bytes from Pair's own table (ARCH-DRY). `TestActorRelaunchChordsConfirmTheThreadOnScreen` covers every encoding with another thread paging, and would fail if the chord were still switcher-only.
- The fake `OuterPresenter` returns what `RecordOuterTTY` last wrote, so the fake keeps state instead of returning a canned answer (ARCH-MOCK).

**2. Critical:** none.

**3. Important**
- **BR-1 is not addressed** (`cmd/internal/launcher/compaction.go:108-111`). The message now names the checkpoint path, which is good. But it offers `pair continue --retry <tag>`, which needs a saved v1 restart marker (`checkpoint_retry.go:40-45`), and this refusal returns before `WriteRestartMarker`. The working route is `pair continue --checkpoint <SourcePath>` once the session is gone. For an unreadable record, "compact from the relaunched thread" doesn't apply at all. This is the third finding in the `refusal-names-the-recovery-route` family. The rule: *a refusal may name only a recovery command whose precondition that arm has already set up.* In `runCompaction`, `--retry` is valid only after the marker write succeeds, which covers the quit-intent and kill-failure arms. Before that, the route is `--checkpoint <path>`. That list is complete: the Couch-request arm names no command. Enforce the rule with a test that runs the named route against the state left after the refusal.

**4. Minor**
- `launcher/lifecycle_test.go:119-124`: this comment still says `pair restart` does not refuse in the adopted case and that Alt+n ends the thread. #284 made both untrue.
- The Spec says Alt+n "joins" the focus-after-prefix tests. `TestLifecycleCandidateUsesFocusAfterPrefix` still sends only Alt+x.

**5. Test coverage**
- These pass at the head commit: `couchkeys`, `workbenchshortcut`, `artifactpath`, the relaunch/actor tests in `couchtty`, the passthrough test in `wrapcmd`, both Lua tests, and the new shell test.
- Four tests fail in my shell with the same results at the base commit, so this diff doesn't cause them. They are `TestCreateLayoutWrapperPreservesAgentCommand`, `TestOSRuntimeResolveContinuation`, `TestLaunchNativeRestartInfersAgentFromScopedDataDir` and `TestPairHelpShimInvokesPairKeys`. Three hit permission errors; the other two fail the same way at base. That means I could not see the runcli restart test pass myself. The evidence for it is the implementor's full `make test`.
- No test checks the text of the compaction refusal.
- Compaction's unreadable-record case isn't tested directly. It goes through the same gate the restart test covers.
- The operator smoke test in Done-when can't be checked from this commit range. It belongs in `--verified`.

**6. Architecture**
- **ARCH-DRY:** pass. One reader (`ReadOuterPresenter`) and one rule.
- **ARCH-PURE:** pass. The predicate takes plain values, and the gate reads through the `Runtime` seam.
- **ARCH-PURPOSE:** pass apart from BR-1. All six testable Done-when rows are delivered.
- **ARCH-MOCK:** flag, but it predates this work. The fake writes restart markers to `writtenMarkers` and reads them from `restartMarkers`, so a write-then-retry sequence can't be tested. That split is why BR-1's route went uncaught.
- **ARCH-CONSTRAINTS:** pass. It's one table lookup on the keystroke path and one file read per restart.
- **ARCH-SECURE:** pass. Decoding is strict, the gate fails closed, and the help page fails open.
- **ARCH-ORDER:** minor flag. The relaunch state machine is reused unchanged, but Ctrl+Space followed by Alt+n in the same read is untested (see Minor).
- **ARCH-FUNERAL:** pass. Nothing new is stored on disk; the outer-tty record already existed.

**7. Plan revisions:** none if the focus-after-prefix test gains Alt+n. Otherwise the Spec's sentence claiming it should get a `## Revisions` entry.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The refusal now names the checkpoint path, but it offers `pair continue --retry <tag>`, which prepareContinuationRetry refuses without a retained v1 restart marker. This arm returns before WriteRestartMarker; a scratch test got "no retained continuation restart intent". Name `pair continue --checkpoint <SourcePath>` once the session is gone, drop "relaunched thread" for the unreadable-record case, and pin the route with a test (3rd in family refusal-names-the-recovery-route; the rule is in the review body).
  - id: BR-2
    disposition: addressed
    note: |
      tests/lifecycle-command-nvim-test.sh drives the real init.lua; reverting the detach or restart call site in a scratch copy gives 2 FAILs each; wired into make test.
  - id: BR-3
    disposition: addressed
    note: |
      compaction.go:108 now passes opts.Env.CouchHosted().
  - id: BR-4
    disposition: addressed
    note: |
      The error names the outer-tty record (or its path from ReadFile) and the remedy (re-attaching with `pair` rewrites it); restart_test's unreadable case asserts the refusal.
  - id: BR-5
    disposition: addressed
    note: |
      keyscmd.go:90-91 now reads coherently.
  - id: BR-6
    disposition: addressed
    note: |
      lifecycle_command.lua takes error_level as a dep and touches no vim global; the Lua test pins it with a sentinel level.
  - id: BR-7
    disposition: addressed
    note: |
      atlas/architecture.md:816 names the shared seam, the report-your-own-failure rule and the wiring test.
findings:
  - id: new
    severity: Minor
    family: comment-accuracy
    title: |
      lifecycle_test.go:119-124 still says `pair restart` does not refuse in the adopted case and Alt+n ends the thread
    detail: |
      This is the 2nd finding in family comment-accuracy. Rule: when an issue changes a behavior, every in-tree statement citing that issue or describing the old behavior is part of the fix; run `git grep -n '#284'` and reread each hit for tense. That sweep over cmd/nvim/tests/atlas/README found 27 hits and only this one stale. Rewrite it to say the gate now refuses first and this client-side refusal is the backstop.
  - id: new
    severity: Minor
    family: design-claim-needs-its-test
    title: |
      The Spec says Alt+n joins the focus-after-prefix tests, but TestLifecycleCandidateUsesFocusAfterPrefix still sends only Alt+x
    detail: |
      onRelaunchHotkey reads focus when the chord is handled, so Ctrl+Space followed by Alt+n in one read should relaunch the highlighted pager row, not the thread on screen. Nothing pins that ordering (ARCH-ORDER). Add Alt+n to that test's chord set, or revise the Spec sentence.
```

---

## Re-review — 2026-09-19T08:55:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 284 — Pair's Alt+n in a Couch-hosted thread confirms, then silently does nothing |
| repo | pair |
| issue file | workshop/issues/000284-alt-n-silently-refused-under-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 62e4e9c7781f6d7436f0ae32ad61750df22acb62..dac272aac51515d9e38b444032a6d86900d8806e |
| command | sdlc close --issue 284 |
| reviewer | claude |
| timestamp | 2026-09-19T08:55:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three open findings are genuinely addressed, each with evidence I verified by running rather than by reading the commit message: `TestCompactionRefusalNamesARouteThatRuns` fails without the BR-1 message change (the regexp finds no `--checkpoint` route in the old text, and the `--retry` assertion inverts), `TestLifecycleCandidateUsesFocusAfterPrefix` now drives Alt+n through both focus arms (BR-9), and the BR-8 comment is corrected with the `#284` sweep confirming it was the last stale hit. The implementation itself holds up under the shadow sweep: `WriteRestartMarker` has exactly two in-tree callers (`restart.go:46`, `compaction.go:124`) and both sit behind `couchRestartGate`, `pair keys` reads the same `CouchOwnsRestart` predicate, and the Couch-side re-ownership is pinned per encoding with zero bytes leaking to the child. Nothing here blocks the boundary; the three findings below are Minor and two of them are rule-level records rather than site fixes.

**Verification run:** `go build ./...` clean; `launcher` (Restart/Compaction/Couch/Checkpoint), `couchtty` (Relaunch/LifecycleCandidate/ActorLifecycle), `couchkeys`, `keyhelp`, `keyscmd` (PresenterAndHosting/NoLayerWidens/CouchPresented) all pass; `nvim -l nvim/lifecycle_command_test.lua` and `bash tests/lifecycle-command-nvim-test.sh` both green (5/5 rows incl. "no direct vim.fn.system call"). Unrelated failures in this shell are all `operation not permitted` on pty/`/tmp` (ptychild, mktemp) — environmental, not from this diff. The full `TMPDIR=<scratchpad> make test` and the operator smoke-test Done-when row remain the close's own evidence obligations.

### 1. Strengths

- `cmd/internal/launcher/outerrecord.go:59` — one two-bool predicate with three consumers (restart gate, compaction gate, `pair keys` wording) is the right shape; the IO lives in `couchRestartGate` and `Runtime.OuterPresenter`, so the rule itself is unit-testable without a runtime.
- `cmd/internal/launcher/restart_test.go:156` — the five-case table (`couch` / hosted / unreadable / terminal / no record) pins both directions of the gate *and* asserts no marker, no quit intent, no kill. That is the destructive-case Done-when row, driven.
- `tests/lifecycle-command-nvim-test.sh` — the second assertion ("nothing reached `vim.fn.system` behind the helper's back") is what makes the Lua unit test load-bearing; reverting any one call site goes red twice over.
- `cmd/internal/couchkeys/couchkeys.go:87` — `switcher()` → `pairChord(scope, …)` keeps scope declared in one table, and `actorReserved` reading that scope means routing and the help page cannot disagree (ARCH-DRY).
- `cmd/internal/launcher/compaction.go:109` — naming the surviving artifact (`--checkpoint <path>`) instead of a route whose precondition the arm deliberately skipped, with a test that parses the advice and re-resolves the path. That is the right shape for "the refusal must run."

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

- `cmd/internal/launcher/outerrecord.go:63` / `cmd/internal/keyscmd/keyscmd.go:90` — both comments claim the help and the gate can never disagree; they do disagree on the unreadable record (keys fails open to Pair's standalone "reload pair" row per `TestUnreadablePresenterRendersPairsPage`, the gate fails closed). 3rd in `comment-accuracy` — see the finding block for the rule, not the site.
- `cmd/internal/launcher/outerrecord.go:76` — `if !sessionEnvHosted && tag != ""` skips the presenter read entirely when the tag is unresolved, so that one state fails *open* where the unreadable record fails closed.
- `workshop/lessons.md:2330` — the existing refusal rule says "a command that exists today"; BR-1's route existed and still could not run. 3rd in `refusal-names-the-recovery-route`.

### 5. Test coverage notes

- Every Done-when row that is machine-checkable now has a test: chord re-ownership per encoding with a paging sibling (`console_relaunch_chord_test.go:438`), Alt+d/Alt+x/Alt+Shift+N still passing through (`:409`, `:107`), the gate before mutation on both writers, the draft's notify-on-failure (unit + wiring), and the help/enforcement agreement (`keyscmd_test.go:155`).
- `fakeRuntime.OuterPresenter` models last-attach-wins and the unreadable record, but not `RecordOuterTTY`'s tty-less **removal** branch (real: `Remove(path)` → later reads say "no record" → restart proceeds). No test can express "attached without a tty" today. Cheap to add if that path ever matters; not worth a round.
- Untested by construction: whether a *stale* `presenter=couch` record can outlive Couch. I convinced myself it cannot — reaching `pair restart` requires an attach, and every attach rewrites the record — so no finding, but it is the assumption the gate rests on.

### 6. Architectural notes (each marker stated)

- **ARCH-DRY — pass.** `pairChord` collapses the two declaration helpers; `CouchOwnsRestart` is one rule with three consumers; `lifecycle_command.lua` absorbs four call sites plus the notify block duplicated in `PairConfirmAgentRestart`.
- **ARCH-PURE — pass.** The predicate is pure; `couchRestartGate` is the thin IO shell; the Lua helper takes `system`/`status`/`notify`/`error_level` as deps and touches no global, so its test needs no vim.
- **ARCH-PURPOSE — pass.** Shadow sweep: two marker writers exist, both gated; the help derives from the enforcement rather than restating it. The deferred "follow-up" here (`#246` adoption) is genuinely separable.
- **ARCH-MOCK — pass.** The fake stores what `RecordOuterTTY` wrote and `presenterErr` models a record that will not decode; production and test share the `Runtime` seam. See the coverage note on the removal branch.
- **ARCH-CONSTRAINTS — pass.** Keystroke path gains one linear scan over ~8 declared chords; the only new IO (one small record read) is on `pair restart`, behind a confirmation modal, not per keystroke.
- **ARCH-SECURE — pass, with the Minor note.** `DecodeOuterRecord` stays strict, the destructive gate fails closed, the help fails open by declared choice. The undecidable-tag path above is the single place the gate neither decides nor refuses.
- **ARCH-ORDER — pass.** No new state between events; relaunch reuses `relaunch.go`'s outcome machine. The actor arm's "thread not in the inventory" case returns a visible notice (`menu.go:635`) rather than the silence this issue was filed about, and the prefix-then-chord interleaving is now pinned for both chords in both focuses.
- **ARCH-FUNERAL — pass.** Nothing durable created; `outer-tty-<tag>` is pre-existing and overwritten per attach.

### 7. Plan revision recommendations

None. The Spec's design paragraphs (Couch scope, the two-member gated class, the four-user draft helper, the ARCH notes) all match what the code delivers, including the focus-after-prefix claim that BR-9 found outstanding.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      compaction.go:109 now names the retained checkpoint and `pair continue --checkpoint <path>`; TestCompactionRefusalNamesARouteThatRuns pins no-marker + route parses + path resolves and validates, and goes red on the old --retry text.
  - id: BR-8
    disposition: addressed
    note: |
      lifecycle_test.go:119 now says the client refusal is the BACKSTOP; I re-ran the #284 sweep (24 code hits plus README/atlas) and found no further stale tense.
  - id: BR-9
    disposition: addressed
    note: |
      TestLifecycleCandidateUsesFocusAfterPrefix is now a chord table carrying Alt+n, asserting the actor arm confirms against the thread the prefix left focused.
findings:
  - id: new
    severity: Minor
    family: comment-accuracy
    title: |
      Two comments claim the help and the restart gate can never disagree; they diverge on the unreadable record
    detail: |
      This is the 3rd finding in family `comment-accuracy`. Do not fix the two
      sites alone. The rule: when two consumers of one predicate have DIFFERENT
      failure policies, a comment may not assert they always agree -- it must
      name the divergence and why. Measured prevalence here: 2 of the 3
      consumers of CouchOwnsRestart carry the overstated claim
      (outerrecord.go:63 "the help cannot promise a reload that the gate
      refuses"; keyscmd.go:90 "appears exactly when Pair's Alt+n cannot
      reload"), while atlas/architecture.md states it correctly. The behavior
      both deny is tested on both sides: keyscmd
      TestUnreadablePresenterRendersPairsPage (fail open, standalone reload row)
      and launcher TestRunRestartRefusesACouchOwnedSessionBeforeMutation's
      "unreadable presenter record" case (fail closed). The Spec chose this
      split deliberately, so the code should say so.
  - id: new
    severity: Minor
    family: undecidable-input-fails-open
    title: |
      couchRestartGate skips the presenter read entirely when the tag is unresolved, so that state fails open
    detail: |
      outerrecord.go:76 guards the read with `!sessionEnvHosted && tag != ""`.
      An unreadable record refuses (fail closed), but an unresolvable tag
      neither decides nor refuses: runRestart proceeds to write the marker,
      the quit intent and the kill. Reached when PAIR_TAG is unset AND
      TagForSessionName misses (a scoped session name absent from the index --
      the `pair-<tag>` legacy prefix always resolves), with Couch presenting
      the client: exactly the thread-ending outcome this issue exists to
      prevent. Narrow, and the comment shows it was considered, but the two
      states are epistemically identical ("cannot tell whether Couch presents
      this session") and get opposite policies. Either refuse there too, with
      the same message, or have the comment say why proceeding is safe rather
      than only that the env half still applies.
  - id: new
    severity: Minor
    family: refusal-names-the-recovery-route
    title: |
      The lessons rule for this family says "a command that exists today", which does not catch BR-1's failure mode
    detail: |
      This is the 3rd finding in family `refusal-names-the-recovery-route`
      (BR-4, BR-1, this). Do not fix an instance -- both are already fixed. The
      rule exists at workshop/lessons.md:2330 (from #146) but is scoped to
      EXISTENCE: `pair continue --retry` exists, is a declared verb, and would
      pass its suggested "assert the suggested command is in the verb set"
      test -- yet it could not run, because the arm returns before writing the
      marker --retry consumes. Sharpen that rule to cover preconditions: a
      remedy must be runnable FROM THE STATE THE REFUSAL LEAVES BEHIND, and the
      test must assert the state the remedy needs (here: the retained
      checkpoint resolves and validates), not merely that the verb is spelled
      correctly. compaction_test.go's TestCompactionRefusalNamesARouteThatRuns
      is the model to cite.
```
