# Boundary Review — pair#692 (whole-issue close)

| field | value |
|-------|-------|
| issue | 692 — Native park helper refuses a live session before attachment |
| repo | pair |
| issue file | workshop/issues/000692-park-helper-live-listing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 04e2216a0b2432d014ceac87eb6c6289bce7e667..25759fed860b7688315102255dfa15baa54f9a9f |
| command | sdlc close --issue 692 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T05:53:40-07:00 |
| verdict | SHIP |

## Review

Targeted tests pass. Now I'm reverting the fix to confirm the regression test goes red.

```verdict
verdict: SHIP
confidence: high
```

The change does what the plan says, and I checked both new behaviours by breaking them. In `requireListedLiveTerminal` (`cmd/internal/launcher/session_handoff_os.go:256-285`), a listing that leaves out a session whose exact server is live now refuses with its own error, `ErrListingOmittedLiveServer`. A failed process scan also refuses, as unknown. Only a real absence still says "not listed live". Only a listing that names the session live lets the attach through. Nothing retries, and it adds no new native query. The fixture now waits for an OS sign that the helper's Zellij client started (a live `zellij` child in the process table) before it asks for a client count. Both regression tests pass. When I disabled the classification (`if false && len(live) != 0`), `TestListedTerminalOmissionOfALiveServerIsUnknown` failed with `omitted class=false want true: terminal "chosen" is not listed live`. When I removed the gate (`if started()` → `if true`), `TestParkReadinessQueriesOnlyAfterTheHelperClientStarts` failed. I put both files back afterwards. Nothing blocks shipping. The remaining findings are Minor: one test gap, one small inconsistency and one unclear diagnostic.

**1. Strengths**
- `cmd/internal/launcher/osruntime.go:146-154`: `liveSessionServers` is split out of `SessionBlocksReuse`. The reuse check and the handoff fence now share one source for the rule "a listing can silently omit a live server" (ARCH-DRY pass).
- `requireListedLiveTerminal` stays pure. Both the listing and the server evidence are passed in as functions, so the five-row table test needs no IO (ARCH-PURE pass).
- The test checks three things: the error class with `errors.Is`, that the listing was queried exactly once (no retry), and that no process scan runs when the session is listed live (`scans != 0`). That keeps the positive fence's cost the same.
- The readiness test pins exact counts (`queries == 1`, `checks == 4`), so the gate can't quietly turn into polling.
- The atlas is updated in both places: the attachment row of the table and the paragraph on fixture ordering.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **Wiring is untested.** Nothing checks that `Check` and `Admit` (`session_handoff_os.go:87,160`) pass `a.rt.liveSessionServers` and not a function that always reports no servers. Swapping one in would keep every test green. Both paths refuse either way, so this only affects the diagnostic.
- **Two process-evidence sources in one method.** `Check` uses `a.rt.liveSessionServers` (which honours `r.Processes` and `r.sessionQuiescence`) on the attach branch, but `(osSessionOwnerIO{}).SessionServers` (nil source) on the create branch at line 89.
- **Unclear timeout message.** If the helper never starts a client, the readiness loop's message says `last query: <nil>`. It doesn't say the client never started. In this window, that loop is the only place the gate affects diagnostics.

**5. Test coverage**
- Both new behaviours are covered by tests that fail without the fix (confirmed above).
- Both states of each Done-when clause are exercised: live server and no server, scan failed, exited, and listed live for the fence; before and after the client starts for the fixture.
- The targeted launcher and couchcore tests pass on the head commit.
- I did not run the live native tests. Their evidence is the implementer's log (5/5 locally), plus the CI attempt, which still has to happen at merge.

**6. Architecture**
- **ARCH-DRY:** pass, apart from the Minor create-branch inconsistency.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass. All three Done-when clauses are covered. Clause 3's CI attempt is honestly deferred to the merge CI run, as the plan's Verify item says.
- **ARCH-EVOLVE:** pass (n/a). No persisted format changes.
- **For later work, not raised as a finding:** `cmd/internal/couchcore/resume_route.go:136-148` reads "not listed" the same way. With `WarmOnly` it refuses as "not listed live"; otherwise it moves on to recovery. The log says #684 and #695 have adopted this classification, so it is out of scope here. When they apply it, they should reuse `ErrListingOmittedLiveServer` and not add a second sentinel.

**7. Plan revisions:** none needed. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: injected-seam-wiring-untested
    title: |
      Check/Admit wiring of a.rt.liveSessionServers into requireListedLiveTerminal has no test
    detail: |
      session_handoff_os.go:87 and :160 (the only two production call sites) could pass a stub that always reports no servers and every test stays green. Refusal still holds; only the diagnostic class is unpinned.
  - id: new
    severity: Minor
    kind: defect
    family: single-process-evidence-source
    title: |
      terminalOSAttempt.Check uses two different process-evidence sources
    detail: |
      The attach branch uses a.rt.liveSessionServers (honours r.Processes/sessionQuiescence); the create branch at session_handoff_os.go:89 uses (osSessionOwnerIO{}).SessionServers with a nil source. Consider using a.rt.liveSessionServers in both.
  - id: new
    severity: Minor
    kind: defect
    family: timeout-diagnostic-names-phase
    title: |
      Readiness timeout reports "last query: <nil>" when the helper client never started
    detail: |
      park_lifecycle_live_test.go waitParkHelperAttachmentWithQuery ctx.Done branch; it should say the client-start gate never opened. This is the only diagnostic in the window that the gate affects.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1809ceb6-5400-4ed3-ba80-b6e338ce0730","repository":"github.com/xianxu/pair","issue_id":"000692","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/903bf7fbe89a8f04","reviewed_head":"25759fed860b7688315102255dfa15baa54f9a9f","slot":"pair:2","requested_at":"2026-10-19T12:51:29.805757Z"},"phase":"finished","pid":90284,"started_at":"2026-10-19T12:51:30.037494Z","finished_at":"2026-10-19T12:53:39.623757Z","session_id":"89e9e193-4384-4b29-8654-2b1f5330ca32","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.678719666s
head: "25759fed860b7688315102255dfa15baa54f9a9f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T06:00:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 692 — Native park helper refuses a live session before attachment |
| repo | pair |
| issue file | workshop/issues/000692-park-helper-live-listing.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc220897b1e430382d56168078569d3841dd3425..0b9fd8bffa0a546943f8d4a6d525f9ca701cf7ad |
| command | sdlc close --issue 692 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T06:00:36-07:00 |
| verdict | SHIP |

## Review

Still on the fix review. The BR-1 wiring test goes red when I swap in a stub; now I'm checking for other process-evidence sources and how BR-3 is used.

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are addressed. None of them was blocking; all were Minor. The changes add no new defect and don't change scope. The only issue-file edits are the status change to `codecomplete`, the `actual_hours` field, the card-mirror hash and a Log entry. No main-side conflicts needed resolving.

- **BR-1 (Check wiring untested):** `TestTerminalAttachCheckUsesTheRuntimeServerEvidence` (`cmd/internal/launcher/session_listing_test.go:183`) passes at head. I swapped the Check attach call site for a stub that always reports no servers, and the test failed with `terminal "chosen" is not listed live`, as expected. Then I restored the file.
- **BR-2 (two process-evidence sources):** `session_handoff_os.go:89` now uses `a.rt.liveSessionServers` on the create branch too. I checked for other copies of the same pattern: no production code outside tests calls `osSessionOwnerIO{}` any more, so the whole class is fixed, not just this one place.
- **BR-3 (timeout message unclear):** the `everStarted` flag in `waitParkHelperAttachmentWithQuery` now makes the timeout say when the helper never started its client. couchcore vets clean, and the `TestParkReadiness*` tests pass.

1. **Strengths**
   - The BR-1 test exercises the real `terminalOSAttempt` through the runtime's own seam (`OSRuntime.sessionQuiescence`), not a hand-built stand-in for it.
   - The BR-2 fix leaves `liveSessionServers` as the only source of process evidence in `Check`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** the test comment says "Check and Admit", but only Check is pinned. The Admit call at `session_handoff_os.go:160` passes the same expression and is not covered on its own. It still refuses either way, as BR-1's own detail noted, so this does not reopen BR-1.

5. **Test coverage**
   - Run at head, the full launcher package fails one test, `TestWorkbenchPerformanceOwnerPassCountsEveryNativeFence`, and fails it with BR-2 reverted too. The error is `bind: invalid argument`, most likely because the Unix socket path under the review checkout is too long. It has nothing to do with this window.
   - Reverting BR-2 turned no other test red. That fits a consistency refactor, since with the default runtime both expressions do the same process scan.

6. **Architecture**
   - **ARCH-DRY:** pass. The BR-2 fix leaves one evidence source.
   - **ARCH-PURE:** pass.
   - **ARCH-PURPOSE:** pass. BR-2's class was checked across the tree.
   - **ARCH-EVOLVE:** n/a. No persisted format changes.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestTerminalAttachCheckUsesTheRuntimeServerEvidence goes red when the Check call site gets a stub that reports no servers; Admit passes the same expression.
  - id: BR-2
    disposition: addressed
    note: |
      Create branch now uses a.rt.liveSessionServers; no production osSessionOwnerIO{} call sites remain.
  - id: BR-3
    disposition: addressed
    note: |
      The everStarted flag makes the ctx.Done branch say when the helper never started its zellij client.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c51b93dd-ee0c-4dd9-b48c-52fd4748fdae","repository":"github.com/xianxu/pair","issue_id":"000692","recipe":"fix-review","round":2,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/4973d1089cc88bb3","reviewed_head":"0b9fd8bffa0a546943f8d4a6d525f9ca701cf7ad","slot":"pair:2","requested_at":"2026-10-19T12:57:26.786875Z"},"phase":"finished","pid":85648,"started_at":"2026-10-19T12:57:26.965263Z","finished_at":"2026-10-19T13:00:36.138429Z","session_id":"16cb733b-2426-4fde-bb86-990ce48a0770","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.045718791s
head: "0b9fd8bffa0a546943f8d4a6d525f9ca701cf7ad"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
