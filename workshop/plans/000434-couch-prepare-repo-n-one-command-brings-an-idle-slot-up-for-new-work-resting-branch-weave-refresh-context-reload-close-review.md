# Boundary Review — pair#434 (whole-issue close)

| field | value |
|-------|-------|
| issue | 434 — couch --prepare repo:N: one command brings an idle slot up for new work (resting branch, weave refresh, context reload) |
| repo | pair |
| issue file | workshop/issues/000434-couch-prepare-repo-n-one-command-brings-an-idle-slot-up-for-new-work-resting-branch-weave-refresh-context-reload.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8d7c50ef44e16b94bd5275230dd80fd32430c2c1..b93c18c32ac71c4e41f2e1165b40baf9936251e9 |
| command | sdlc close --issue 434 |
| reviewer | claude |
| timestamp | 2026-10-11T01:27:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing pair#434's diff; checking weave refresh semantics now.

Still reviewing; waiting on the package tests.

Package tests mostly ran; rerunning the prepare-specific tests in isolation.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

`couch --prepare` meets all four Done-when items. Admission is a separate round trip through the running Couch, so a busy, dirty or off-resting slot is refused before anything changes. `weave refresh` runs in the CLI process, not on the console's serial queue. A recheck stops with `refresh-dirty` before the reload, and the reload checks again whether the slot is idle and clean. I confirmed that the real `weave refresh` does what the plan relies on: it fetches `origin/main`, refuses unclean or divergent checkouts, then fast-forwards and compiles (`ariadne/cmd/weave/internal/refresh/refresh.go:110-188`). One Important gap stops this from being SHIP: README.md's couch command listing documents every other live slot verb but not `--prepare`. The rest are Minor.

**1. Strengths**
- **One verb list.** `OpPrepare` is added to `slotOperations` (`couchcore/slot_operation.go:34`), and the parser, protocol validator, socket and `TestEveryLiveOwnerOperationIsDispatched` all read that one list (ARCH-DRY).
- **Exhaustive admission test.** `TestDecideLiveRestartExhaustive` now covers every combination including branch, detached and the prepare op (3·2⁸·3 cases). It also pins the refusal order: idle first, then clean, then resting branch.
- **Shared poll loop.** `exchangeSlotOperation` (`couchcmd/messages.go:89`) is factored out with a typed `slotOperationUncertain`, so prepare and the other slot verbs share one admit-and-poll loop.
- **Realistic `RunPrepare` tests.** They use real git repos: origin, a slot clone on `main-slot1` and a substrate. The fake weave sits behind the production `ProvisionIO` seam and simulates `.gitattributes` dirtying, refresh failure and a reload refusal (the case where a turn starts mid-refresh).
- **Safe against older Couch builds.** An older Couch stops before any step; a Couch that admits but sends no `Path` is caught explicitly (`prepare.go:207`).

**2. Critical:** none.

**3. Important**
- **README update missing for `couch --prepare`.** The command block at `README.md:396-416` lists `--relaunch` and `--reload-context` with their refusals but not `--prepare repo:N --confirm [--force-unknown] [--json]`. Fix: add an entry after `--reload-context` that names its refusals (`not-resting`, `refresh-dirty`).

**4. Minor**
- **Wrong advice when admission is uncertain.** That error says "verify with couch --recover-plan-from-sdlc before resending". For prepare, admission changes nothing, so re-running is safe and the advice is wrong (`messages.go:80` reused by `prepare.go:197`).
- **Untracked files are judged differently.** The recheck uses `--untracked-files=no` (`slot_prepare.go:601`), but admission's `ProbeSlotGit` counts untracked files as dirty. If a refresh creates an untracked file, the run ends as a `reload refused (dirty)` instead of `refresh-dirty`, which is still safe but less clear.
- **Parsing mixed with IO.** The `-z` porcelain parsing sits inline in `RunPrepare`; it could be a pure `dirtyPaths([]byte)` helper with its own test, including the rename/copy skip (ARCH-PURE).
- **Filesystem read bypasses the seam.** `substrateCommits` reads the filesystem through `layergraph.OSFS{}` directly rather than through the injected IO (ARCH-PURE/ARCH-MOCK).
- **Stale doc comment.** The comment on `prepareLiveRestart` (`live_restart.go:245`) still says "admits relaunch or reload-context".
- **Duplicated args.** Prepare's `Operations()` args repeat reload-context's verbatim; a shared args slice would keep them in step.

**5. Test coverage notes**
- `go test ./cmd/internal/couchcore -run 'Prepare|LiveRestart|EveryLiveOwner|…'` passes, and so does `go test ./cmd/internal/couchcmd -run Prepare`.
- Failures I hit that this diff did not cause:
  - **Sandbox `/tmp`:** couchmessage `TestTransportRejectsOversizePayloads` and couchcmd `TestSlotOperationRelaunchCarriesOverrides` fail with `mkdir /tmp/... operation not permitted`.
  - **artifactpath inventory:** the test fails on files outside this window (`wrapcmd/peer_viewport.go`, `nvim/review/*`). The two new couch sources are inventoried correctly.

  The operator's full `make -k test` run should confirm both.

**6. Architectural notes**
- ARCH-DRY: pass (one verb list, one shared exchange); minor duplication of the args spec.
- ARCH-PURE: pass with notes. `DecideLiveRestart` and `PrepareReport` are pure; the porcelain parsing and the substrate filesystem read could move out of `RunPrepare`.
- ARCH-PURPOSE: pass. All five Spec steps are delivered, step 2 through weave's own fast-forward, and the TL flow in the skill is now "prepare, then send". README is the remaining consumer (Important above).
- ARCH-MOCK: pass. weave runs through the `ProvisionIO` seam with a fake that changes state over real git. There is no live conformance check that weave refresh still fast-forwards the resting branch, which is worth a note.
- ARCH-CONSTRAINTS: pass. The refresh stays off the serial console queue and has a 20-minute bound; git probes are bounded at 10 seconds.
- ARCH-SECURE: pass. `Path` comes from a socket owned by the same user and is used as an argv `Dir`, never through a shell.
- ARCH-ORDER: pass. The sequence is linear; a turn that starts mid-refresh is caught by the reload's own admission and tested; uncertain outcomes surface as failed steps rather than successes.
- ARCH-FUNERAL: pass. Nothing durable is created; receipts live in the existing in-memory map, which `ReceiptExpire` sweeps.

**7. Plan revision recommendations**
- The plan's Task 5 should list README.md alongside atlas, SKILL.md and CHANGELOG, or record why it was left out.

```findings
findings:
  - id: new
    severity: Important
    family: user-surface-docs-sweep
    title: |
      README couch command listing omits the new couch --prepare verb
    detail: |
      README.md:396-416 documents --relaunch and --reload-context with their refusals but not --prepare repo:N --confirm [--force-unknown] [--json] (not-resting, refresh-dirty).
  - id: new
    severity: Minor
    family: verb-specific-error-guidance
    title: |
      Uncertain prepare admission tells the user to verify before resending, though admission has no effect
    detail: |
      slotOperationUncertain's message (messages.go:80) is reused for prepare; re-running prepare is safe and converges.
  - id: new
    severity: Minor
    family: consistent-dirty-predicate
    title: |
      Recheck ignores untracked files while admission counts them dirty
    detail: |
      slot_prepare.go:601 uses --untracked-files=no; ProbeSlotGit includes untracked, so a refresh that creates an untracked file ends as reload refused (dirty) instead of refresh-dirty.
  - id: new
    severity: Minor
    family: pure-core-io-shell
    title: |
      Porcelain parsing and substrate FS read are inline in RunPrepare rather than pure helpers or the seam
    detail: |
      Extract dirtyPaths([]byte) as a pure tested function; substrateCommits uses layergraph.OSFS{} directly instead of the injected IO.
  - id: new
    severity: Minor
    family: stale-doc-comment
    title: |
      prepareLiveRestart comment still says it admits relaunch or reload-context only
```

---

## Re-review — 2026-10-11T01:42:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 434 — couch --prepare repo:N: one command brings an idle slot up for new work (resting branch, weave refresh, context reload) |
| repo | pair |
| issue file | workshop/issues/000434-couch-prepare-repo-n-one-command-brings-an-idle-slot-up-for-new-work-resting-branch-weave-refresh-context-reload.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8d7c50ef44e16b94bd5275230dd80fd32430c2c1..4686716e8ea06f41ddcad409b99075e6ec5398be |
| command | sdlc close --issue 434 |
| reviewer | claude |
| timestamp | 2026-10-11T01:42:40-07:00 |
| verdict | SHIP |

## Review

Focus on the last fix commit first, then the core files.

Still reviewing. The test failures come from the sandbox blocking `/tmp` socket dirs, not from the code. Running the prepare-only tests to confirm:

```verdict
verdict: SHIP
confidence: high
```

This round's commit (4686716e) fixes all five prior findings, and BR-1, BR-2 and BR-3 now have working evidence behind them. The recheck after the refresh now uses the admission's own status command (`slotGitStatusArgs`) and parser (`ParseSlotGitStatus`). Prepare's refresh-dirty check and the reload's re-admission can no longer judge the checkout differently. A new test covers the untracked-file case (`TestRunPrepareUntrackedOutputIsRefreshDirty`). Without the fix, `--untracked-files=no` would have reported the checkout clean and the reload would have run, so the test checks `r.reloads == 0` and would fail. The admission whose outcome was lost now gets its own prepare-specific message, with a test that rejects the restart-style "recover-plan" text. The README lists `--prepare` with its flags and the refresh-dirty refusal. The targeted tests pass: `go test -run 'Prepare|SlotGit' ./cmd/internal/couchcore/` and `TestPrepareCLI`, all 5 subtests. The broader `couchmessage` and `couchcmd` runs fail only on `mkdir /tmp/...: operation not permitted`, which is the sandbox blocking socket test directories, not this diff. I found nothing new that blocks.

1. **Strengths**
   - `slot_prepare.go:123-136` uses one dirty predicate, borrowed from the gate the next step will face, instead of restating it. This is ARCH-DRY applied to the family. The new lessons.md rule states it as a rule, not as one site.
   - `slotgit.go:77-98`: `SlotGitChangedPaths` is pure. Its test also asserts that `ParseSlotGitStatus` agrees on the same fixture, which links the reporting list to the predicate. It covers ordinary, rename, unmerged, untracked and ignored entries.
   - `prepare.go:40-44`: the guidance for a lost outcome depends on the verb's effect. Admission changes nothing, so the advice is to run it again.
   - `RunPrepare` keeps its fixed step list and skips the remaining steps after a stop, so the report's shape is the same on every path.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `substrateCommits` still reads through `layergraph.OSFS{}` directly (see the BR-4 disposition). That is acceptable, because the tests run against temp-dir fixtures.
   - Porcelain v2 without `-z` C-quotes unusual paths, so `SlotGitChangedPaths` can show quoted names in the report. This affects display only.

5. **Test coverage:** each behavior fix has a regression test that would fail without it: untracked files lead to refresh-dirty with no reload, and the lost-outcome case gives the prepare-specific message. The changed-paths parser has a pure table test with an agreement check against the parser.

6. **Architecture**
   - ARCH-DRY: pass. The predicate is now shared.
   - ARCH-PURE: pass. Parsing is pure. The FS read remains inside `substrateCommits`, which is minor.
   - ARCH-PURPOSE: pass.
   - ARCH-MOCK: pass. Commands go through the `ProvisionIO` seam, with temp-dir git fixtures.
   - ARCH-CONSTRAINTS: pass. The refresh is bounded by a 20-minute timeout and runs off the console's queue.
   - ARCH-SECURE: pass. Git output is parsed through a closed grammar, and a parse error stops the run as recheck failed.
   - ARCH-ORDER: pass. The run is a linear step machine that stops at the first step that does not succeed. There is no journal, and convergence on re-run is the documented recovery.
   - ARCH-FUNERAL: pass. The run creates nothing durable.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:412-415 now lists couch --prepare repo:N --confirm [--force-unknown] [--json] with the refresh-dirty refusal.
  - id: BR-2
    disposition: addressed
    note: |
      prepare.go:40-44 gives a prepare-specific remedy; the TestPrepareCLI lost-outcome subtest asserts "run couch --prepare again" and no recover-plan text.
  - id: BR-3
    disposition: addressed
    note: |
      The recheck uses slotGitStatusArgs + ParseSlotGitStatus; TestRunPrepareUntrackedOutputIsRefreshDirty fails without the fix (the reload would run).
  - id: BR-4
    disposition: addressed
    note: |
      Porcelain parsing is now the pure, tested SlotGitChangedPaths/ParseSlotGitStatus; the remaining OSFS read in substrateCommits runs on temp-dir fixtures in tests, acceptable.
  - id: BR-5
    disposition: addressed
    note: |
      live_restart.go:245-248 comment now names relaunch, reload-context or prepare.
```
