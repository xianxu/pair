# Boundary Review — pair#690 (whole-issue close)

| field | value |
|-------|-------|
| issue | 690 — review pane: a zero-record pair review apply completes the round (pairs ariadne#453) |
| repo | pair |
| issue file | workshop/issues/000690-review-pane-a-zero-record-pair-review-apply-completes-the-round-pairs-ariadne-453.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0830075fb39de9e5adf9b1120cd353d6191e3fda..b84d11e53eb79faf17115f68a9a9f5a7bc5b0c39 |
| command | sdlc close --issue 690 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T03:08:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Go and Lua tests pass. The window test can't run in this sandbox: there's no `bin/pair`, and `git init` in /tmp is refused. Next I'm checking that the rule test goes red when `is_empty` is reverted.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code is correct and does what the Spec and Plan ask, and I found no correctness bugs. In the engine (`ApplyFile`), any apply that changes nothing now writes an empty-round artifact. It still goes through the same flock, operator-lock and stale checks first, never touches the document, and returns a Round with no Body. In the pane, a round is now identified by its hash plus its time (`result_hash@at`), and an empty round ends the agent's turn without reloading the file. The Go tests pass (sessionsync, reviewapply) and the Lua tests pass (round_watch, agent_round_rule). I mutated the code to confirm the tests catch the bug: forcing the `is_empty` branch off in `on_applied_round` turns `agent_round_rule_test` red ("an empty round ends the turn (after_applied_round=0)"). The window test could not run here because the sandbox has no `bin/pair` and refuses `git init` in /tmp. I checked its new case by reading it, and the logic holds. One thing stands in the way of a clean close: the last Done-when clause, the live no-edits pane check, is still unchecked, so the issue does not yet meet its own close criteria. There are also two stale doc sentences next to the change.

1. **Strengths**
   - `cmd/internal/sessionsync/applyfile.go:78-115`: the empty round goes through the existing order (flock → lock → hash → remove the old artifact → write the artifact), so all the refusals still apply. `TestApplyFileRefusesAnEmptyApplyAlike` pins exit 3 and exit 6, and checks that a refused apply writes no artifact.
   - The artifact format does not change: telling rounds apart by the existing `at` field avoids a new field (ARCH-EVOLVE passes). An older pane that dedups on the hash simply ignores an empty round, which is the behaviour it has today.
   - `round_watch.key` is now the one identity used by `decide`, `watch` and the declined-round warning (ARCH-DRY passes).
   - `is_empty` checks that `base_hash` equals `result_hash`, so an older span-less fixture with no `base_hash` still reloads, as intended.
   - The Go test checks that the document's mtime is unchanged, which proves no write happened rather than just that the content matches.

2. **Critical:** none.

3. **Important**
   - The Done-when clause "Live no-edits pane check" (Plan item `[ ]`) is not delivered, yet this is the whole-issue close boundary. Either run the check and record the evidence in the Log, or move it out of this issue with a Revisions entry before closing.

4. **Minor**
   - Two doc sentences next to the change still say every applied round reloads, which an empty round no longer does:
     - `atlas/review-workbench.md:103` ("on each applied round `on_applied_round` reloads the file")
     - `workshop/targets/review-protocol.md:263` ("`on_applied_round` only reloads and decorates")
   - `round_watch.is_empty` (`nvim/review/round_watch.lua:25-28`) would raise an error if an artifact with matching hashes had `"spans": null`, because `#` fails on `vim.NIL`. The Go engine always writes `[]`, so this is a hardening point only.
   - The empty-round Warning path (artifact write fails when nothing changed) has no test. The existing `TestApplyFileRoundWriteFailureWarns` covers only a round that changed the file.

5. **Test coverage notes:** the engine covers `[]`/nil, a no-op record and both refusals. The CLI covers both wire forms. round_watch covers the same hash at a new time and the same key. The rule test covers an empty round over unsaved operator text, and I confirmed by mutation that it fails without the fix. The window-test case is unverified in this environment.

6. **Architectural notes:**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. `key` and `is_empty` are pure and unit-tested; `on_applied_round` stays thin glue.
   - ARCH-PURPOSE: pass on the code. The only gap is the live check above.
   - ARCH-EVOLVE: pass. The artifact keeps its shape, and older readers degrade to today's behaviour.

7. **Plan revision recommendations:** if the live check is deferred, add a "## Revisions" entry moving that Done-when clause to a follow-up, or to ariadne#453's close, with a pointer.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: done-when-clause-undelivered-at-close
    title: |
      The live no-edits pane check in Done-when is still unchecked at the whole-issue close
    detail: |
      The Plan item "Live no-edits pane check: the TL holds it for the next Couch restart" is [ ]. It is the only instance in the window. Either record the live evidence in the Log, or add a Revisions entry moving the clause out of this issue before close.
  - id: new
    severity: Minor
    kind: defect
    family: doc-claim-stale-after-behavior-change
    title: |
      Docs still say every applied round reloads, but an empty round no longer does
    detail: |
      Instances: atlas/review-workbench.md:103 ("on each applied round on_applied_round reloads the file") and workshop/targets/review-protocol.md:263 ("on_applied_round only reloads and decorates"). Add "(an empty round only ends the turn)" to both.
  - id: new
    severity: Minor
    kind: defect
    family: defensive-artifact-decode
    title: |
      is_empty errors on spans = JSON null when the hashes match
    detail: |
      nvim/review/round_watch.lua:27 applies # to round.spans, which raises an error for vim.NIL. The Go engine always writes [], so only a foreign or older artifact would trigger it. Guard it with type(round.spans) ~= 'table'.
  - id: new
    severity: Minor
    kind: defect
    family: warning-path-untested
    title: |
      The empty-round Warning branch in ApplyFile has no test
    detail: |
      cmd/internal/sessionsync/applyfile.go:112-114: the "empty round could not be recorded; commit nothing" text is untested. TestApplyFileRoundWriteFailureWarns covers only a round that changed the file.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4c3723bd-5c25-46b0-a8db-728da7771b2f","repository":"github.com/xianxu/pair","issue_id":"000690","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fcf8adfc812f2645","reviewed_head":"b84d11e53eb79faf17115f68a9a9f5a7bc5b0c39","slot":"pair:2","requested_at":"2026-10-19T10:06:45.316577Z"},"phase":"finished","pid":94442,"started_at":"2026-10-19T10:06:45.655176Z","finished_at":"2026-10-19T10:08:33.517992Z","session_id":"ce33cf02-83ed-4374-b4d5-be1919fb5fed","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.220632459s
head: "b84d11e53eb79faf17115f68a9a9f5a7bc5b0c39"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T03:14:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 690 — review pane: a zero-record pair review apply completes the round (pairs ariadne#453) |
| repo | pair |
| issue file | workshop/issues/000690-review-pane-a-zero-record-pair-review-apply-completes-the-round-pairs-ariadne-453.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0830075fb39de9e5adf9b1120cd353d6191e3fda..8e19651f97a513921e4e6ab3cff48759787e275b |
| command | sdlc close --issue 690 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T03:14:31-07:00 |
| verdict | SHIP |

## Review

Go tests pass. Next I'm reverting the BR-3 and BR-4 fixes to confirm their regression tests fail without them.

Now the reverts (disposable checkout):

```verdict
verdict: SHIP
confidence: high
```

This change is ready to ship. All four round-1 findings are addressed, and I checked each one against the code rather than the commit message. For BR-3 and BR-4 I removed the fix in the disposable checkout and confirmed its test fails, then restored the files. With the BR-3 guard removed, `round_watch_test.lua` errors at `round_watch.lua:27`. With the empty-round Warning branch removed, `TestApplyFileEmptyRoundWriteFailureWarns` fails. With both fixes in place, `go test` for `sessionsync` and `reviewapply` and the `round_watch` Lua test all pass. The Spec and every Done-when clause are delivered. Both wire forms of an empty apply (`[]` and `{"records":[]}`) are tested, plus a record that changes nothing. Both refusals are tested on an empty apply: stale (exit 6) and operator lock (exit 3). The rule test covers an empty round arriving while the operator has unsaved text. The window test asserts the spinner and the pending block both clear. I found nothing new.

**Strengths**
- **No format change.** `cmd/internal/sessionsync/applyfile.go:124-133` gives the empty round its identity from the existing `at` field, so the artifact format is unchanged (ARCH-EVOLVE). An older pane, which matches rounds on `result_hash` alone, ignores an empty round, which is today's behavior.
- **One round identity.** `round_watch.key` (`nvim/review/round_watch.lua:14`) is the single identity used by `decide`, `watch` (both its startup seed and the "seen" update) and the declined-round warning in `init.lua`. No site still compares raw hashes.
- **Same refusals as any apply.** An empty apply skips `BuildEdit` but still goes through the flock, operator-lock and hash checks, so it is refused exactly like a normal apply. The Couch path still refuses an empty list.
- **Safe with unsaved operator text.** The `is_empty` check comes before the modified-buffer check in `on_applied_round`. That is safe because an empty round reloads nothing, and `agent_round_rule_test.lua` pins it: no writes, no reload, and the operator's buffer stays modified.

**Critical:** none.

**Important:** none.

**Minor:** none new. `reviewapply_test.go:78` ignores the `json.Unmarshal` error, but the assertions after it would still catch a bad reply.

**Test coverage:** the engine is tested directly (Go, temp-directory fixtures). The pure helpers `key`, `decide` and `is_empty` are tested without IO, including `spans = vim.NIL`. The integration path goes through the headless window test. The implementer's log records that forcing `is_empty` to false turns the window test red. I did not re-run that mutation myself.

**Architecture**
- **ARCH-DRY: pass.** Round identity lives in one helper; there is no duplicated logic.
- **ARCH-PURE: pass.** `key`, `is_empty` and `decide` are pure; the file IO stays in `watch` and `ApplyFile`.
- **ARCH-PURPOSE: pass.** The full purpose is delivered: the engine writes the empty round, the pane ends the turn, and the protocol, atlas and SKILL docs say an empty reply commits nothing. The live check moves out of this issue through a documented Revisions entry. It needs this branch landed and a Couch restart, so it is a post-landing check rather than the issue's purpose.
- **ARCH-EVOLVE: pass.** The artifact has no new field. The meaning of the existing fields is unchanged; `base_hash = result_hash` was already the natural reading of a round that changed nothing. Older readers degrade to today's behavior.

**Plan revisions:** none needed. The plan and its Revisions entry match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Done-when no longer claims the live check; the Plan lists it as a post-landing note held by the TL, with a Revisions entry giving why (it needs this branch landed and a Couch restart).
  - id: BR-2
    disposition: addressed
    note: |
      atlas/review-workbench.md:103 and review-protocol.md:263 now say "an empty round only ends the turn"; on_applied_round matches (init.lua is_empty early return).
  - id: BR-3
    disposition: addressed
    note: |
      round_watch.lua:27 guards non-table spans; round_watch_test.lua:43 tests vim.NIL and errors when the guard is removed (verified).
  - id: BR-4
    disposition: addressed
    note: |
      TestApplyFileEmptyRoundWriteFailureWarns asserts "commit nothing" and an empty Body; it fails when the empty-round Warning branch is removed (verified).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2fd1d9ab-c6f2-4b9d-8071-3dc0024bb7bf","repository":"github.com/xianxu/pair","issue_id":"000690","recipe":"small-diff-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/79964afad28fd987","reviewed_head":"8e19651f97a513921e4e6ab3cff48759787e275b","slot":"pair:2","requested_at":"2026-10-19T10:13:11.858047Z"},"phase":"finished","pid":3877,"started_at":"2026-10-19T10:13:12.047363Z","finished_at":"2026-10-19T10:14:27.187015Z","session_id":"360adbf1-f091-4c30-b33c-19766069e130","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.702029834s
head: "8e19651f97a513921e4e6ab3cff48759787e275b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
