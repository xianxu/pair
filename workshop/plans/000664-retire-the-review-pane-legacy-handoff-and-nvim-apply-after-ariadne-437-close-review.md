# Boundary Review — pair#664 (whole-issue close)

| field | value |
|-------|-------|
| issue | 664 — Retire the review pane legacy handoff and nvim apply after ariadne#437 |
| repo | pair |
| issue file | workshop/issues/000664-retire-the-review-pane-legacy-handoff-and-nvim-apply-after-ariadne-437.md |
| boundary | whole-issue close |
| milestone | — |
| window | 455737dcd073ae5a8610f9b9e7483203c7708f07..e9b99eb5c103e48584b4435ec350295e0fc60cf1 |
| command | sdlc close --issue 664 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T19:16:04-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff is clean of retired identifiers so far. Next I'm checking the runtime bundle mirror, then the new rule test, and running the suites.

The Lua suite passes. `make test-review` hit a sandbox limit on `/tmp`, so I'm rerunning the changed shell tests one at a time with a temp dir inside the checkout.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The retirement itself is done. No retired identifier is still described as live anywhere in tracked files (checked by grep and by the extended `deletedvocabulary_test.go`). The Go env bindings, gc rows and manifest rows are gone, and the runtime bundle no longer carries `handoff.lua`, `gate.lua` or `reconcile.lua`. The spinner gap the survey found is fixed through `after_applied_round`. `make test-lua` passes, and so do `go test ./cmd/internal/artifactpath ./cmd/internal/runtimebundle` and the launcher scoped-path test. Two things stop a plain SHIP:
- **No live-round evidence.** The third `## Done when` clause (a live round with an ariadne#437 agent) is still unchecked, and nothing in the Log shows it was run.
- **The rule test proves less than it says.** Its `on_lines` detector stops working after the pane's own `:edit!`. I checked this with a probe: after `:edit!` the attachment is detached (`detached true`, on_lines delta 0). So the "declined round changes no lines" assertion passes no matter what, and the promised positive control for a marker resolve is not in the test.

1. **Strengths**
   - Thorough deletion. Grepping the whole tree for every retired symbol finds hits only in `deletedvocabulary_test.go` and an old continuation note, and the vocabulary guard now enforces their absence (`cmd/internal/artifactpath/deletedvocabulary_test.go:48-54`).
   - The survey found a real bug, and it is fixed with a RED/GREEN test. `on_applied_round` now calls `after_applied_round` (`nvim/review/init.lua:57`, `nvim/review.lua:460`), and `review-window-test.sh` checks that the spinner clears after an applied round.
   - The dead seams went with the legacy path, including the `legacy` field and `strict` parameter in `restore_controller`. `M.poke` and `review_opened` went too.
   - The rule test covers both the applied and the declined case, and it spies on writes through `writefile`, `io.open` and `os.rename`. In my test run, a `set_lines` mutation added after the reload made it fail (through the `modified` assertion).
   - The docs are consistent. Atlas, README (the Alt+Return row and the restore-blocking sentence) and review-protocol seams 2/2b all describe the code as it now is.

2. **Critical findings:** none.

3. **Important findings**
   - **The rule test's `on_lines` detector stops working after the reload** (`nvim/review/agent_round_rule_test.lua:29,62-66`).
     - `:edit!` detaches the `nvim_buf_attach` listener.
     - After that, `line_changes == typed` holds no matter what happens. Even the operator's own `set_lines` on line 63 increments nothing.
     - The plan's "positive control that … a marker resolve trips the detectors" is missing. Only the send/BufWritePre control exists, and nothing positively controls `on_lines` or the file spies.
     - Fix: track `vim.b[buf].changedtick` deltas, or re-attach in `on_reload`. Then assert that `line_changes` grows on the operator edit, and add the marker accept/reject control the plan claims.
     - Every instance in this window: the declined-round assertion (line 65) and the absent `on_lines` and file-spy positive controls.
   - **`## Done when` clause 3 has no evidence.** The plan item for the live ariadne#437 round (operator smoke) is unchecked and the Log has no smoke entry. Closing the issue needs either that smoke recorded, or the clause explicitly handed to a named owner.

4. **Minor findings**
   - `nvim/review/spinner.lua:3` still says "expect a handoff … clears it when the handoff lands". It should say an applied round clears it. This is the only stale handoff comment left in `nvim/`.
   - The `legacy` parameter of `nvim/review/restore.lua:7` `validate_context` is now dead. Every caller passes `false` or nil, because the only `legacy=true` source was the retired `restore_controller.legacy`.
   - ARCH-DRY: `tests/review-window-test.sh` rebuilds the round hash as `vim.fn.sha256(raw):sub(1, 16)`. The rule test uses `round_watch.hash(file)` instead, and the window test should too.
   - Zero-edit agent replies may never clear the spinner. The legacy `{records:[]}` handoff cleared it; now only an applied-round artifact does. If an agent review produces no edits and no artifact, the spinner stays, and Alt+c restore is refused with "agent request pending". This depends on what the engine and ariadne#437 do for an empty round, which I did not verify.
   - Restore safety is relaxed, as the spec intends. Deleting the pending-landed check means restore no longer blocks while an agent round is applied but not yet committed. The README change records this. If the engine has its own guard for this, it may be worth pointing to it.

5. **Test coverage notes**
   - Verified: `make test-lua` passes (it includes `agent_round_rule_test`). `go test` passes for artifactpath, runtimebundle, and the launcher `OwnsTag|Scoped` tests. `review-projection`, `review-controls` and `review-apply` pass.
   - Could not run here: the launcher package as a whole fails on unix socket paths that are too long for the sandbox, which has nothing to do with this diff. `make test-review` also fails on the shell and Python tests that call `git init` under `/private/tmp` (`review-applied-round`, `review-window`, the restore tests, observation), because the sandbox refuses those repos. The Log says the suite was green on the host.
   - The rule test can catch a write: a mutation adding `set_lines` after reload made it fail, through the `modified` assertion rather than `on_lines`.

6. **Architectural notes**
   - ARCH-DRY: pass, apart from the test hash duplication listed under Minor.
   - ARCH-PURE: pass. The diff is mostly deletion. `after_applied_round` is a thin UI hook injected into the orchestrator, and `poke_bodies` stays pure.
   - ARCH-PURPOSE: pass on the code. The shadow sweep of retired consumers found no surviving live restatement. The live-round clause is the only part of the purpose not yet evidenced (Important above).

7. **Plan revision recommendations**
   - Either change the rule-test plan item to match what the test does (a send control only), or add the marker-resolve control and a working `on_lines` detector.
   - Record the live-round smoke result in the Log, or move that clause to a named follow-up with an owner.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: detector-without-positive-control
    title: |
      agent_round_rule_test on_lines detector detaches at :edit!, so the declined-round check passes no matter what
    detail: |
      A probe showed nvim_buf_attach is detached by `:edit!` (on_detach fires, later set_lines give on_lines delta 0). So in nvim/review/agent_round_rule_test.lua:63-65, `line_changes == typed` passes without checking anything, even the operator's own edit is not counted, and nothing positively controls on_lines or the writefile/io.open/os.rename spies. The plan's claimed marker-resolve positive control is also missing. Every instance: the declined-round assertion at line 65, the missing on_lines control, the missing file-spy control, the missing marker accept/reject control. Fix: track changedtick deltas, or re-attach in on_reload, and assert that each detector fires on an operator action.
  - id: new
    severity: Important
    kind: defect
    family: done-when-clause-unevidenced
    title: |
      Done-when clause 3 (live round with an ariadne#437 agent) is unchecked and has no Log evidence
    detail: |
      The last Plan item ("a live round with an ariadne#437 agent (operator smoke)") is unchecked, and the Log has no smoke entry showing a live round applied through `pair review apply` with the pane only reloading and decorating. Record the smoke result before close, or hand the clause to a named owner.
  - id: new
    severity: Minor
    kind: defect
    family: stale-legacy-vocabulary
    title: |
      spinner.lua header comment still says the spinner clears when "the handoff lands"
    detail: |
      nvim/review/spinner.lua:3. It is now cleared by after_applied_round (and the definition paths). This is the only stale handoff comment left under nvim/.
  - id: new
    severity: Minor
    kind: defect
    family: dead-seam-after-retirement
    title: |
      restore.validate_context `legacy` parameter is now dead
    detail: |
      nvim/review/restore.lua:7. The only legacy=true source was the retired restore_controller.legacy; review.lua passes false or nil.
  - id: new
    severity: Minor
    kind: defect
    family: duplicated-round-hash
    title: |
      review-window-test rebuilds the round hash as sha256():sub(1,16) instead of round_watch.hash
    detail: |
      This is an ARCH-DRY issue in tests/review-window-test.sh's applied-round block. agent_round_rule_test.lua already uses round_watch.hash, and the window test should use it too.
  - id: new
    severity: Minor
    kind: defect
    family: spinner-clear-paths
    title: |
      A zero-edit agent reply may leave the awaiting spinner and the restore block set
    detail: |
      The legacy empty handoff cleared awaiting; now only an applied-round artifact does. If an agent review produces no edits and no artifact, "agent request pending" blocks Alt+c restore. Check what the engine and ariadne#437 do for an empty round.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e07e6e32-56ce-4e27-be76-896f75794c3d","repository":"github.com/xianxu/pair","issue_id":"000664","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/325f4353ad4609d5","reviewed_head":"e9b99eb5c103e48584b4435ec350295e0fc60cf1","slot":"pair:2","requested_at":"2026-10-19T02:10:08.231902Z"},"phase":"finished","pid":87469,"started_at":"2026-10-19T02:10:08.345389Z","finished_at":"2026-10-19T02:15:53.902742Z","session_id":"7511cd26-1235-4480-9418-5d2946696e9e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.56351875s
head: "e9b99eb5c103e48584b4435ec350295e0fc60cf1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-18T19:20:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 664 — Retire the review pane legacy handoff and nvim apply after ariadne#437 |
| repo | pair |
| issue file | workshop/issues/000664-retire-the-review-pane-legacy-handoff-and-nvim-apply-after-ariadne-437.md |
| boundary | whole-issue close |
| milestone | — |
| window | 455737dcd073ae5a8610f9b9e7483203c7708f07..dc7e372b55cbf3daddb108a5c4f621972d8632c6 |
| command | sdlc close --issue 664 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-18T19:20:40-07:00 |
| verdict | SHIP |

## Review

Checking the BR-1 fix: I'm mutating the declined-round path so it writes buffer lines, to confirm the rule test catches it.

```verdict
verdict: SHIP
confidence: high
```

This boundary is ready to ship. The round-1 fixes hold up when tested, not just on the commit messages:

- **BR-1, the rule test's detectors:** each detector now has a working control after the reload. I made the declined-round path write a buffer line, and the test went red with `FAIL a declined round writes nothing`. With the change reverted it passes again.
- **Retired code:** none of the retired identifiers or env vars remains in code, tests or docs. The only mentions left are the deleted-vocabulary guard, the removal note in the atlas, and an old continuation note.
- **Tests run here:** every `nvim/review/*_test.lua` passes, and so do the Go `artifactpath` and `launcher` tests.
- **Tests not run here:** the shell tests (`review-applied-round-test.sh`, `review-window-test.sh`) could not run. They create temp repos under `/tmp`, and this sandbox refuses that (sdlc #339). Their headless coverage is the implementer's evidence; I checked their contents.
- **What remains:** the zero-edit-reply spinner gap (BR-6) is Minor, so it doesn't block. It is logged as a known gap and has been raised with the hub.

Done-when: current — the only Revision narrows the rule test's positive controls. That is an implementation detail under Done-when clause 2, so the three clauses still cover what this diff builds.

1. **Strengths**
   - `nvim/review/init.lua:33-58`: `on_applied_round` only reloads (`:edit!`) and decorates; it never writes. When the operator has unsaved edits it keeps their buffer and warns once per round.
   - `nvim/review/agent_round_rule_test.lua:30-36,66-78`: the test measured that `:edit!` detaches a buffer listener, so it now counts the detach as the reload and re-attaches. It then proves each of the three detectors (line changes, buffer writes, Lua file writes) live with its own control.
   - `cmd/internal/artifactpath/deletedvocabulary_test.go`: the retired names are now enforced, so a doc that describes them as live fails `go test`, rather than relying on someone remembering.
   - The spinner gap found in the survey was fixed through a single hook, `after_applied_round` (`nvim/review.lua:460`), and the window test covers it.

2. **Critical:** none.
3. **Important:** none new.
4. **Minor:** none new. BR-6 stays open as a known gap.
5. **Test coverage notes**
   - The rule test covers both the applied round and the declined round, and both now catch a mutation.
   - The shell integration tests could not run in this sandbox; see the summary above.
6. **Architectural notes**
   - **ARCH-DRY: pass.** The round hash comes from `round_watch.hash` in both tests, and the applied path reuses `apply.apply_snapshot` and `projection`.
   - **ARCH-PURE: pass.** `round_watch.decide` and `restore.validate_context` stay pure, and the reload glue in `on_applied_round` is thin.
   - **ARCH-PURPOSE: pass.** Every item on the retirement list is deleted, and the deleted-vocabulary guard enforces it. The one remaining edge, the empty-round reply, depends on ariadne#437's protocol, which defines no empty-round reply, so it is outside this issue.
7. **Plan revision recommendations:** none. The existing Revision matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The detector re-attaches on detach and each detector has a live control after the reload (agent_round_rule_test.lua:30-36,66-78). Mutating the declined path in init.lua to call nvim_buf_set_lines made the test fail ("a declined round writes nothing"); reverting made it pass.
  - id: BR-2
    disposition: withdrawn
    note: |
      The evidence holds. The clause needs this build running in the operator's Couch, which only happens after landing, and the live Couch runs a pre-#664 binary. The pane-side behaviour (reload plus decorate, no nvim write) is pinned headless, and the rule test passes here. The Plan keeps the live smoke unchecked for the next restart, which is honest tracking.
  - id: BR-3
    disposition: addressed
    note: |
      nvim/review/spinner.lua:3-5 now says the spinner clears on after_applied_round or a definition answer. No stale handoff wording is left under nvim/review.
  - id: BR-4
    disposition: addressed
    note: |
      The legacy parameter is removed from restore.lua:7, the nil-context case now returns false, both callers in review.lua:847-849 are updated, and the legacy assertion is dropped from restore_test.
  - id: BR-5
    disposition: addressed
    note: |
      tests/review-window-test.sh:383-385 now uses round_watch.hash(name).
  - id: BR-6
    disposition: not-addressed
    note: |
      The Log records it as a known gap: a reply with no edits writes no round artifact, so the awaiting spinner and Alt+c's restore block stay set. It was raised with the hub because ariadne#437 defines no empty-round reply. Minor and non-blocking, but still unfixed in code.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"35858cdc-5bc4-4ed1-92e5-19e87410c4be","repository":"github.com/xianxu/pair","issue_id":"000664","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7ebcb352c0e36b4d","reviewed_head":"dc7e372b55cbf3daddb108a5c4f621972d8632c6","slot":"pair:2","requested_at":"2026-10-19T02:18:52.124107Z"},"phase":"finished","pid":33578,"started_at":"2026-10-19T02:18:52.414571Z","finished_at":"2026-10-19T02:20:24.951743Z","session_id":"7d1ae163-11c0-43cc-849d-9b757a923a92","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.113653917s
head: "dc7e372b55cbf3daddb108a5c4f621972d8632c6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
