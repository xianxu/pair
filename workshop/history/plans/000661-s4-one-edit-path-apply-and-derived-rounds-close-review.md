# Boundary Review — pair#661 (whole-issue close)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | whole-issue close |
| milestone | — |
| window | f8b82353d3d0080ccf3c770df927402865b85c68..ae49e84daafa33158a904563af74269decedee57 |
| command | sdlc close --issue 661 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T06:12:32-07:00 |
| verdict | REWORK |

## Review

Reviewing the M2 diff: Go engine done, now reading `reviewapply` and the nvim pane changes.

Confirmed one bypass in the propose exception. Running the package tests and the Lua test now.

```verdict
verdict: REWORK
confidence: high
```

This window covers M2. It adds the `BuildEdit` engine shared by `couch --session-sync apply` and the new `pair review apply`/`pair review round` verbs, the pane's round watch and operator lock, and an end-to-end test. Most of it is sound. The Go packages (`sessionsync`, `reviewapply`, `artifactpath`, `dispatcher`) and `round_watch_test.lua` pass at head. One defect blocks SHIP. The propose-mode exception for a side-doc answer is much wider than its documented contract. I confirmed it with a probe test in this checkout: a four-line rewrite under `presentation: propose` was written directly, with no markers, because its replacement text ended in a `→ [proposal](…)` link and the old text held a ready marker. Three Important gaps also need fixing:
- a round-artifact write failure is reported as a failed apply after the document has already changed;
- a crashed pane leaves `FILE.lock` behind permanently;
- the plan's readiness `FILE.lock` commitment was not delivered.

**1. Strengths**
- **One engine.** `contentStep`/`BuildEdit` (`cmd/internal/sessionsync/round.go:860-923`) is now the single content step. `Sync.write` delegates to it, and the copy of `resolveBatch` that used to live in `sync.go` is gone (ARCH-DRY pass).
- **Round artifact ordering.** `ApplyFile` (`applyfile.go:267-275`) removes the old artifact, replaces the document, then writes the new artifact. `TestApplyFileNeverLeavesAStaleRoundArtifact` pins this with a real failure (the directory is made read-only).
- **Operator-owned keys.** These are guarded in the shared engine, so `write --frontmatter` is covered too (`TestProposeGuardsOperatorKeysOnEveryPath`).
- **Shared conformance fixtures.** `testdata/edit-records/conformance` is used from the Go side, and `review-applied-round-test.sh` drives a separate `pair` process against a real pane (reload, diagnostics, lock exit 3, `edits-invalid`).
- **Commit body round-trip.** `RoundBody` and `occurrenceAt` match `reconstruct.occurrence_at`. The test checks that `new_occurrence` lands back on the span's position.

**2. Critical findings**
- **`answersSideDoc` lets an arbitrary rewrite bypass propose** (`round.go:983-986`).
  - The rule (a) check only requires three things: `old` contains a ready marker, `new` contains a side-doc link, and `new` has no ready marker. It never checks that the text around the marker is unchanged.
  - Probe result: base `Line one.\nLine two.\nLine three.\nLine four. 🤖[shorten]` with new text `Totally rewritten.\nX\nY\nZ\nW → [proposal](doc.side/x.md)` returned `err=nil`, and the rewrite was written in place with no markers.
  - README, the couch SKILL and the atlas all describe the exception as "the record that replaces the operator's ready marker with a new link". The code allows much more than that.
  - Fix: require `new == old[:i] + link + old[j:]`, where `[i, j)` is the ready marker's byte range (whitespace around it may be allowed). Add the probe as a negative case in `TestProposeOrdinaryAnswerStaysAProposal` and as a refused conformance fixture.

**3. Important findings**
- **A failed round-artifact write after the document was replaced is reported as a failed apply** (`applyfile.go:273-275`).
  - The CLI returns exit 1 with `Code: error`, no `Hash` and no `Body`, even though the file has already changed.
  - `pair review round` then returns a null round, so the commit body is lost. The agent's next apply goes stale (exit 6) with nothing explaining why.
  - This contradicts M1 decision 4 ("round-write failure is a warning") and turns a confirmed write into a reported failure (ARCH-ORDER).
  - Fix: return `res` with `Hash` and a `Warning` (exit 0, or a distinct code), and test it by making only the artifact path unwritable.
- **`FILE.lock` has no removal path if nvim dies** (`nvim/review/oplock.lua:14-28`; ARCH-FUNERAL, ARCH-ORDER).
  - The lock is written into the user's repository and only removed on send or at `VimLeave`. A crash or SIGKILL leaves it in place.
  - The next pane sees it as "someone else's lock": `take` returns false and `drop` never removes it. Every agent apply then exits 3 indefinitely, and nothing tells the operator.
  - Fix: write an owner identity (pid or pane tag) into the lock, and on start reclaim a lock whose owner is dead or is this same review tag. Alternatively, have the pane notify the operator when a foreign lock is present. Test the crash-leftover case.
- **The plan's readiness `FILE.lock` commitment was not delivered** (`cmd/internal/reviewcmd/run.go:277`). This is the 2nd finding in family `plan-core-concepts-drift`.
  - "M2 as built" says readiness's clean check must ignore an untracked `FILE.lock` for the reviewed file, and that "a test pins it". The check is still a plain `git status --porcelain`, and no such test exists.
  - **Rule for the class:** every commitment in "M2 as built" must either exist in the code with a test, or be listed in a `## Revisions` entry as dropped or deferred.
  - Sweep the whole section, not just this item. For example, the planned rewrite of `review-loop-test.sh`/`fake-review-agent.sh` was replaced by `review-applied-round-test.sh`, and no entry says so explicitly.
  - Fix this item by excluding the reviewed file's `.lock` (for example `git status --porcelain -- . ':!<rel>.lock'`) and adding the test.

**4. Minor findings**
- **Lost decorations on a modified buffer.** In `round_watch.watch`, `seen` is advanced before `on_applied_round` runs. When the buffer is modified and the reload is refused, the round's decorations are never re-delivered.
- **Unsynchronised lock check.** `ApplyFile` checks the operator lock under its own data-dir flock, which the pane never takes. An operator keystroke between the check and the replace is only caught afterwards by the pane's "unsaved edits" warning.
- **Unused field.** `Edit.Resolved` is exported but has no production reader outside `contentStep`.
- **Weak assertion.** `TestProposeGuardsOperatorKeysOnEveryPath`'s moved-closer case now only asserts `err != nil`, not that nothing was written.

**5. Test coverage notes**
- The probed bypass has no negative test: all of the "answer" tests use a `new` that is exactly the link.
- No test forces an artifact-write failure after a successful document replace.
- No test covers a leftover `FILE.lock` from a crashed pane.
- The Lua tests cover `decide`/`decorations`/`oplock` purely, and the e2e test exercises one interleaving: the lock taken before the apply. The race of a lock taken during an apply is not exercised.

**6. Architectural notes**
- **ARCH-DRY:** pass, with the engine consolidated as above.
- **ARCH-PURE:** pass. `BuildEdit` and `readyMarkerIn` are pure. `ApplyFile` is a thin shell.
- **ARCH-PURPOSE:** pass with a note. Done-when "nvim never writes the reviewed file" is deferred to pair#664 under the TL's recorded decision (a), and the issue Log says so.
- **ARCH-MOCK:** pass, using a real filesystem and a real `pair` binary.
- **ARCH-CONSTRAINTS:** pass. The 100 ms poll and the 16 MiB read cap are reused.
- **ARCH-SECURE:** flag. The propose exception trusts the agent's `new` text far beyond the marker swap (Critical above). `ReadRoundArtifact` validates its input through `decodeRound`, which is good.
- **ARCH-ORDER:** flag. A confirmed write is collapsed into a reported failure, and there is a lock check-then-act race.
- **ARCH-FUNERAL:** flag for `FILE.lock`. The round artifact and the flock are covered by the artifactpath GC.

**7. Plan revision recommendations**
- Add a `## Revisions` entry that narrows rule (a) to "new equals old with only the ready marker replaced by the link".
- Either deliver the readiness `FILE.lock` exclusion, or record it as deferred. In the same entry, reconcile the "M2 as built" test list (the `review-loop-test` rewrite vs `review-applied-round-test`).
- State the lifecycle of `FILE.lock` after a pane crash (who removes it and how).

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: propose-direct-exception-scope
    title: |
      answersSideDoc rule (a) lets an arbitrary multi-line rewrite apply directly under propose if new ends in a side-doc link
    detail: |
      round.go:983 checks only readyMarkerIn(old), a sideDocLink in new and no ready marker in new; it never requires the text outside the marker to be unchanged. A probe with base "Line one..Line four. 🤖[shorten]" and new "Totally rewritten.\nX\nY\nZ\nW → [proposal](doc.side/x.md)" returned err=nil and wrote the rewrite with no markers, contradicting README, SKILL and atlas ("the record that replaces the operator's ready marker with a new link"). Fix: require new == old[:i] + link + old[j:] where [i,j) is the ready marker's span, and add the probe as a negative test plus a refused conformance fixture.
  - id: new
    severity: Important
    kind: defect
    family: committed-effect-reported-as-failure
    title: |
      ApplyFile returns an error (exit 1, no Hash, no Body) when the round artifact write fails after the document was already replaced
    detail: |
      applyfile.go:273-275: the document has changed, but the CLI reports Code error, `pair review round` returns a null round, the commit body is lost, and the agent's next apply goes stale. M1 decision 4 makes a round-write failure a warning. Return the new Hash with a warning (exit 0 or a distinct code), and test it by making only the artifact path unwritable.
  - id: new
    severity: Important
    kind: defect
    family: owned-artifact-removal-on-owner-death
    title: |
      FILE.lock written by oplock.lua has no removal path after a pane crash; agent applies then exit 3 indefinitely
    detail: |
      oplock.lua removes the lock only on send or VimLeave, and treats any lock it did not take in this process as someone else's: take returns false and drop never removes it. After an nvim crash or SIGKILL, the lock stays in the user's repo, every pair review apply refuses with operator-lock, and the operator is never told. Record an owner (pid or tag) in the lock and reclaim it when that owner is dead or is this same review, or notify the operator; test the crash-leftover case.
  - id: new
    severity: Important
    kind: defect
    family: plan-core-concepts-drift
    title: |
      Plan's "M2 as built" commits readiness to ignore an untracked FILE.lock with a pinning test; neither exists
    detail: |
      This is the 2nd finding in family plan-core-concepts-drift. The rule for the class: every "M2 as built" commitment is either in the code with a test or listed in a Revisions entry as dropped or deferred; sweep the whole section, including the planned review-loop-test/fake-review-agent rewrite that review-applied-round-test.sh replaced. reviewcmd/run.go:277 still uses a plain `git status --porcelain`, so a leftover FILE.lock makes the repo "interact" instead of "new".
  - id: new
    severity: Minor
    kind: defect
    family: round-watch-delivery
    title: |
      round_watch marks a round seen before on_applied_round runs, so a refused reload (modified buffer) loses its decorations for good
    detail: |
      watch sets seen = result_hash before calling cb; on_applied_round returns false on a modified buffer and the round is never re-delivered.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9a20f231-bbda-4ed5-bd53-09019f59e636","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":5,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9abc4d65d8d35136","reviewed_head":"ae49e84daafa33158a904563af74269decedee57","slot":"pair:2","requested_at":"2026-10-18T13:10:02.855147Z"},"phase":"finished","pid":91696,"started_at":"2026-10-18T13:10:03.088334Z","finished_at":"2026-10-18T13:12:30.751791Z","session_id":"f121eaa8-6b70-4e6c-b76b-8838be99e6ea","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.376850917s
head: "ae49e84daafa33158a904563af74269decedee57"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T06:23:20-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | whole-issue close |
| milestone | — |
| window | f8b82353d3d0080ccf3c770df927402865b85c68..6072d90471d52d0ba2f74f9b80e9150468cac04d |
| command | sdlc close --issue 661 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T06:23:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the five prior findings against the fix commit. Next I run the tests and revert each fix to confirm its regression test fails without it.

Now reverting each fix in turn.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

All five open findings are fixed. I checked the code and ran the tests at the pinned head. For BR-9, BR-10 and BR-12 I also reverted the fix in the disposable checkout and re-ran the tests: they fail for BR-9 and BR-10, but reverting BR-12's readiness call site leaves them green. For BR-11 and BR-13 I read the new Lua tests rather than reverting. `go test` passes for sessionsync, reviewapply and reviewcmd, and `round_watch_test.lua` prints `ok`. One problem remains in the BR-12 fix, the readiness lock exemption. It compares the `?? path` line against git's unquoted path, so it misses any document whose name git quotes: names with spaces or non-ASCII characters (I reproduced `?? "docs/my doc.md.lock"`). For those documents a leftover lock still makes readiness report a dirty repo. The readiness call site also has no test. The BR-13 fix brings a smaller problem: the pane now warns about unsaved edits every 100 ms. It also re-delivers a round without checking that the file still matches it. Neither blocks the boundary.

1. **Strengths**
   - `answersSideDoc` rule (a) (`cmd/internal/sessionsync/round.go:362`) now checks for an exact swap: `new` must equal `before + link + after`, and `wholeSideDocLink` must match the middle exactly. The new refused fixture and `TestProposeLinkAnswerCannotCarryARewrite` both fail when it is reverted.
   - In `ApplyFile` (`applyfile.go:99-103`), a failed round-artifact write is now a warning. `Round`, `Body` and `Hash` are still returned, and the old artifact is removed before the document changes. `TestApplyFileRoundWriteFailureWarns` goes red when the warning is reverted.
   - The lock records the owner's pid, and the pane reclaims it at start and on take (`oplock.lua:27-49`). The tests cover dead, live and own-pid locks.
   - `cleanBesidesLock` is a small pure function, tested directly, and both porcelain call sites use it.
   - The plan's Revisions entry sweeps the whole "M2 as built" section as Built, Changed, Replaced or Deferred. That is the class-level answer BR-12 asked for.

2. **Critical:** none.

3. **Important**
   - `cmd/internal/reviewcmd/run.go:257-266`: `cleanBesidesLock` builds an unquoted `?? rel.lock` line. Git quotes paths with spaces or non-ASCII characters (`core.quotepath`), so the lock is never recognised for those documents. Fix: run `git status --porcelain -z` and compare the raw NUL-separated entries. Also add a `gatherGitFacts`/`RunReadiness` test using a fake runtime whose status output is only the lock. Reverting the call site to `TrimSpace(st)==""` currently keeps every test green.

4. **Minor**
   - `nvim/review/round_watch.lua:59` with `init.lua:145-147`: a declined round is retried on every 100 ms tick, and each retry calls `vim.notify` again. The retry also does not check that the file on disk still hashes to `round.result_hash`, so after the operator's save it could decorate content the round no longer describes.
   - The docs weren't updated: the `Warning` reply field and the lock's pid/reclaim aren't in the couch `SKILL.md` or `atlas/review-workbench.md:162-165`.
   - `oplock.lua:23`: `vim.uv.kill` failing with EPERM (a live process owned by another user) counts as dead, so that process's lock gets reclaimed. Only ESRCH should mean dead.

5. **Test coverage:** The BR-9 and BR-10 fixes go red when reverted. BR-11 and BR-13 have Lua tests: a dead/live/own-pid lock matrix and a declined-then-redelivered round. BR-12 tests the helper but not where readiness calls it.

6. **Architecture**
   - **Passes:** ARCH-DRY (`readyMarkerSpan` reuses the marker scanner), ARCH-PURE (`cleanBesidesLock`, `decide` and `decorations` are pure), ARCH-MOCK (git goes through the `rt.Git` seam), ARCH-CONSTRAINTS, ARCH-SECURE (the lock pid is parsed with `tonumber` and falls back to "someone else's"), ARCH-FUNERAL (a dead pane's lock now has a removal path).
   - **Flagged:** ARCH-PURPOSE, the readiness exemption doesn't work for names git quotes. ARCH-ORDER, round delivery acts on a stale observation without checking the file's current hash.

7. **Plan revisions:** add a Revisions line saying readiness reads `--porcelain -z` (after the fix), and that redelivery requires the file on disk to match the round's `result_hash`.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      round.go answersSideDoc now requires an exact swap; TestProposeLinkAnswerCannotCarryARewrite and the conformance fixture fail when round.go is reverted.
  - id: BR-10
    disposition: addressed
    note: |
      applyfile.go returns Hash, Round, Body and Warning; TestApplyFileRoundWriteFailureWarns fails when the warning is reverted to return err.
  - id: BR-11
    disposition: addressed
    note: |
      oplock records a pid and reclaims a dead owner at start and on take; round_watch_test covers dead, live and own-pid locks. EPERM edge noted as a new Minor.
  - id: BR-12
    disposition: addressed
    note: |
      cleanBesidesLock is wired into both porcelain checks with a helper test, and the Revisions entry sweeps M2 as built. Gaps in quoting and call-site coverage are raised as a new finding.
  - id: BR-13
    disposition: addressed
    note: |
      watch marks a round seen only when cb does not return false; the redelivery test covers it. A side effect is raised as a new Minor.
findings:
  - id: new
    severity: Important
    kind: defect
    family: git-porcelain-path-parsing
    title: |
      cleanBesidesLock misses the lock for documents whose path git quotes, and readiness wiring is untested
    detail: |
      run.go:257 compares against an unquoted "?? rel.lock" line, but porcelain v1 prints "?? \"docs/my doc.md.lock\"" for names with spaces or non-ASCII characters (reproduced), so readiness still reports such repos dirty. Fix: use git status --porcelain -z and compare raw entries. Add a RunReadiness or gatherGitFacts test with a lock-only status, because reverting the call site keeps all tests green.
  - id: new
    severity: Minor
    kind: defect
    family: round-watch-delivery
    title: |
      A declined round warns every 100ms and is redelivered without checking the file still hashes to result_hash
    detail: |
      This is the 2nd finding in family round-watch-delivery. The rule: the pane acts on a round only while the file on disk hashes to round.result_hash, and it reports each declined round once. Put the hash check in decide, and debounce the notification per result_hash.
  - id: new
    severity: Minor
    kind: defect
    family: docs-gate-new-surface
    title: |
      The apply reply's Warning field and the lock's pid/reclaim are not documented in the couch SKILL or the atlas
    detail: |
      This is the 2nd finding in family docs-gate-new-surface. The rule: any change to the apply reply or lock contract updates SKILL.md and atlas/review-workbench.md in the same commit. The agent must learn that Warning means "commit Body now, no highlight".
  - id: new
    severity: Minor
    kind: defect
    family: owned-artifact-removal-on-owner-death
    title: |
      oplock alive() treats EPERM as dead, so it reclaims a lock owned by a live process of another user
    detail: |
      vim.uv.kill returns nil,"EPERM" for a live process the caller cannot signal. Only ESRCH should count as dead.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"18ac178c-7b11-4c0f-b7c0-d096998c709b","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":6,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/9c78299f55a9a765","reviewed_head":"6072d90471d52d0ba2f74f9b80e9150468cac04d","slot":"pair:2","requested_at":"2026-10-18T13:20:08.522782Z"},"phase":"finished","pid":49473,"started_at":"2026-10-18T13:20:08.628399Z","finished_at":"2026-10-18T13:23:20.21868Z","session_id":"f0a8de62-ee6c-46c2-b161-d9aab448f78f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.563967167s
head: "6072d90471d52d0ba2f74f9b80e9150468cac04d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-18T06:31:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 661 — S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply |
| repo | pair |
| issue file | workshop/issues/000661-s4-one-edit-path-apply-and-derived-rounds.md |
| boundary | whole-issue close |
| milestone | — |
| window | f8b82353d3d0080ccf3c770df927402865b85c68..3862431a728b78c31f179ac7078bb6b7cc069b16 |
| command | sdlc close --issue 661 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-18T06:31:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the regression tests for the round-2 fixes (EPERM, warn-once), then I'll run the suites.

Now running the tests, plus an EPERM probe against the old and new `alive()`.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

Round 2 fixed the one Important finding still open. Readiness now reads `git status --porcelain -z`, and a new wiring test fails if either the call site or the parser is reverted. So BR-14 is addressed. The docs fix (BR-16) is also correct. The other two fixes, BR-15 and BR-17, work at runtime, but neither has a test that fails without it. I checked BR-17 by hand: `vim.uv.kill(1,0)` returns `nil, EPERM` and `reclaim` now leaves that lock alone. But the only "live owner" test uses the parent pid, which can be signalled, so undoing the fix leaves the suite green. The same is true of BR-15's warn-once debounce. By the claimed-fix rule, both stay `not-addressed`. Both are Minor, so neither blocks. Every test I ran passed: the round_watch/oplock suite and `go test` for reviewcmd, sessionsync and reviewapply.

1. **Strengths**
   - `cmd/internal/reviewcmd/run.go:257-266`: switching to `-z` gets rid of git's path quoting completely, instead of trying to unquote it. Any rename entry or other entry still reads as dirty, which is the correct default.
   - `cmd/internal/reviewcmd/run_test.go` `TestReadinessTreatsTheReviewLockAsClean` drives `gatherGitFacts` through the fake runtime and covers both directions: lock-only reads clean, and lock plus another file reads dirty. Undoing either half turns it red.
   - `nvim/review/round_watch.lua:26` `M.hash` gives the same value as Go's `sessionsync.Hash` (first 16 hex digits of the sha256). It hashes only rounds that already passed `decide`, so idle polls stay cheap. The stale-round test (`'a round the file no longer matches is not delivered'`) fails if the hash check is removed.
   - `nvim/review/oplock.lua:23-26`: treating only ESRCH as "gone" is correct for luv's `nil, msg, name` failure return.
   - The new docs say the right things: atlas/review-workbench.md (Warning, pid/ESRCH reclaim, readiness and the lock) and SKILL.md ("Warning → commit Body anyway").

2. **Critical findings:** none.

3. **Important findings:** none new.

4. **Minor findings**
   - BR-15 (not-addressed): the warn-once debounce at `nvim/review/init.lua:146-150` has no test. Test sketch: call `on_applied_round` twice on a modified buffer with the same `result_hash` and stubbed `vim.notify`, then assert it warned once. Also, `decide`'s `current_hash` argument is never passed in production. `watch` (round_watch.lua:71) compares the hash inline, so that rule has two copies. Pass `M.hash(file)` into `decide` when the hash is needed, or drop the parameter.
   - BR-17 (not-addressed): add a lock with `pid: 1` to round_watch_test.lua and assert that `reclaim` returns false. A non-root user can't signal pid 1, so it gets EPERM.

5. **Test coverage notes:** I ran `nvim -l nvim/review/round_watch_test.lua` (ok) and `go test` for reviewcmd, sessionsync and reviewapply (ok). The `prepare` call site at run.go:338 has no test of its own. It uses the same helper as the tested `gatherGitFacts` path, so the risk is low.

6. **Architectural notes**
   - **ARCH-DRY:** flag (Minor). The stale-round rule lives in two places: `decide(current_hash)` and the inline comparison in `watch`. This is folded into BR-15.
   - **ARCH-PURE:** pass. `cleanBesidesLock` and `decide` are pure; the git calls and the file hashing happen in the thin glue.
   - **ARCH-PURPOSE:** pass. Both readiness call sites were moved to `-z` in the same round, so the whole class is fixed, not just the reported site.
   - **ARCH-MOCK:** pass. Git goes through the `rt.Git` fake seam.
   - **ARCH-CONSTRAINTS:** pass. Hashing runs only for an unseen round of this file, at a 100ms poll.
   - **ARCH-SECURE:** pass. The lock's pid is parsed with `tonumber`, and an unreadable or missing pid means the lock is left alone.
   - **ARCH-ORDER:** pass with a note. A declined round stays unseen and is offered again. The extra hash check stops a stale round from being acted on after the file has changed.
   - **ARCH-FUNERAL:** pass. A dead pane's lock is reclaimed at review start and on the next take. The `warned` field is in-memory and goes away with the session.

7. **Plan revision recommendations:** none. The atlas and SKILL now match the code.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      run.go gatherGitFacts/prepare use status --porcelain -z with NUL split; TestReadinessTreatsTheReviewLockAsClean fails on revert of call site or parser; space-path case added to the unit table.
  - id: BR-15
    disposition: not-addressed
    note: |
      Stale-hash guard is in place and pinned by 'a round the file no longer matches is not delivered'; the per-result_hash warn-once in init.lua on_applied_round has no test, and decide's current_hash parameter is unused in production (watch compares inline).
  - id: BR-16
    disposition: addressed
    note: |
      SKILL.md documents Warning means commit Body anyway; atlas/review-workbench.md documents Warning, pid lock owner, ESRCH reclaim, and readiness ignoring the lock via porcelain -z.
  - id: BR-17
    disposition: not-addressed
    note: |
      alive() now treats only ESRCH as dead, and a manual probe confirms reclaim leaves a pid-1 (EPERM) lock; but no test pins it, since the live-owner case uses the signalable ppid, so reverting stays green. Add a pid 1 lock case.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f3801f9c-16e0-4a10-bb96-0b1d3ab19500","repository":"github.com/xianxu/pair","issue_id":"000661","recipe":"milestone-review","round":7,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/f43433ecbffe7816","reviewed_head":"3862431a728b78c31f179ac7078bb6b7cc069b16","slot":"pair:2","requested_at":"2026-10-18T13:29:31.480867Z"},"phase":"finished","pid":67127,"started_at":"2026-10-18T13:29:31.670453Z","finished_at":"2026-10-18T13:31:13.079715Z","session_id":"344d094f-5cd5-4e2b-8755-da26b54d8334","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.395328084s
head: "3862431a728b78c31f179ac7078bb6b7cc069b16"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
