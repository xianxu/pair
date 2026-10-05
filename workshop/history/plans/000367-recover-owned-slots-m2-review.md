# Boundary Review — pair#367 (milestone M2)

| field | value |
|-------|-------|
| issue | 367 — Recover local slots from durable issue ownership |
| repo | pair |
| issue file | workshop/issues/000367-recover-owned-slots.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 6f485a034219a19f3fd6820199cf146cb592b0ed..0a3331ce8cfa85fd53bce8c2bdbb6b17032daa7d |
| command | sdlc milestone-close --issue 367 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-04T05:01:30-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

M2 delivers what the issue's Done-when and the 2026-10-04 (d) revision ask for. `resume`/`reboot` are callable through the Couch socket by live slots only: the server checks the caller through the same `broker.Caller` + `authority.current` rule `--send-to` uses. Each request runs on the switcher's own queue and completion path, in the background, and is admitted only when `ActorActions` offers that action on the row at run time. Results are typed in-memory receipts with a closed transition table. The CLI polls on a fresh context per call and turns every lost outcome after admission into one "uncertain" line. The report now emits a parseable `command` for each step, and the skill, README and atlas are updated. Nothing blocks the boundary. The open findings are disposed below, and two new Minor notes are left for later.

On verification: the M2-scoped tests pass. couchmessage passes, couchtty `Remote|Menu|Reattach` passes, the couchcore recover/slot-op tests pass, and the couchcmd slot-op tests pass with `-race` (run unsandboxed because they write under `/tmp`). The other failures I saw were all sandbox pty/`/tmp` "operation not permitted" errors (ptyrunner, terminal soak, the switch-agent orientation helper). Your memory notes already list these as sandbox artifacts. I did not run the full `make test`.

1. **Strengths**
   - `couchmessage/operation.go:96` `ApplyReceiptEvent` is a pure, closed `(state, event)` table. Both production writers (`slotOperations.apply`, `sweepLocked`) go through it, and a duplicate admission returns the held receipt without a second effect (ARCH-ORDER).
   - `console_remote.go` keeps one place responsible for the outcome. `finished` fires from `finishOperation`'s defer after adoption, so an attach failure reaches the caller. An enqueue refusal never reports through `finished` (pinned by `TestRemoteEnqueueRefusalsNeverReport`).
   - `ActorOperationArgs` is now the one row-to-args mapping for both the switcher (`menu.go`, `menu_slot.go`) and the socket (ARCH-DRY). The end-to-end loop found a real bug through it: the switcher's slot resume sent only a path, so it had been refused for a missing `repo-scope` since #306. The lesson is recorded in `workshop/lessons.md`.
   - Recovery is now judged per slot: `slotOperation`, `slotGitUnknown` and `slotDirty` are the only readers of `DepTree`, while host-level judgments still read host facts only. `withRuleA` clones the notes it is given, and `restoreWorkspaceDecision` uses `slices.Concat`, so the earlier slice-alias bug cannot recur.
   - Lifetimes and limits are stated: receipts are capped at 64, swept 5 minutes after turning terminal, dropped when Couch exits, and the poll budget is 3 minutes (ARCH-FUNERAL, ARCH-CONSTRAINTS).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `console_remote.go:69` `remoteResumeCompletion` still recognizes a remote resume by the shape of its fields, and its comment says there is "no origin field of its own". `completion.remote` is now that explicit origin, and the clear only takes effect because `remoteOperationAddress` returns false when `remote` is nil. Gating directly on `completed.remote != nil && completed.name == "resume"` and dropping the shape predicate would make that visible.
   - `atlas/couch.md` step 2 says any caller failure is `unavailable`. The plan's (d) revision says a request with no identity gets `invalid-request`, which is what `handleSlotOperation` returns.
   - `runSlotOperationCLI` handles `result.Code == "uncertain"` at admission, but `slotOperations` never sends that code. It is dead but harmless.

5. **Test coverage**
   - The CLI poll loop and the argv parser are table-driven, one strategy per row (`messages_test.go:246`, `cli_test.go:152`).
   - The caller rule, the scope of status queries, expiry and the cap, concurrency under `-race`, and alias keys are covered in `slot_operations_test.go`.
   - The socket acceptance test drives a detached `:0` and not a `:N` slot. The plan revision records this; the `:N` path is covered only by the couchcore loop.

6. **Architectural notes**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. The table, row selection, args and command text are pure; `PrepareSlotOperation` is the IO shell.
   - ARCH-PURPOSE: pass. Every Done-when bullet for M2 is delivered; the live smoke test is still open in the Plan, as expected.
   - ARCH-MOCK: pass. The tests use the real service on temp sockets, with `SlotCatalogFake` and the fleet fake.
   - ARCH-CONSTRAINTS: pass. Receipts are capped, enqueue never blocks, and each call has its own deadline.
   - ARCH-SECURE: pass. The target is parsed into a typed slot at the boundary, receipts are scoped to the caller, and nothing secret is involved.
   - ARCH-ORDER: pass, with the shape-recognition note above. The lock order is safe: `s.mu` is held over a non-blocking enqueue, and `started` waits for admission to return.
   - ARCH-FUNERAL: pass.
   - Upcoming work: a switcher keypress and a remote request for the same slot use different queue keys. They are serialized only because the queue runs one job at a time and the remote prepare re-checks the row at run time. If the queue ever runs jobs concurrently, that guarantee goes away.

7. **Plan revisions:** none required. Optionally, append to revision (d) that a remote resume is recognized by `completion.remote`, if the Minor fix above lands.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Overtaken by implementation: the poll loop and argv parser landed as table-driven tests, one strategy per row (messages_test.go:246, cli_test.go:152); the plan prose is now history.
  - id: BR-3
    disposition: addressed
    note: |
      The completion now carries an explicit origin (operationCompletion.remote); remoteOperationAddress returns false for nil remote, so a non-remote origin of the same shape clears nothing; TestRemoteResumeRecognitionIsPinnedBothSides pins both producer sides. Residual cleanup noted as Minor in prose.
findings:
  - id: new
    severity: Minor
    family: docs-lag-new-vocabulary
    title: |
      atlas/couch.md says every caller failure is unavailable; a no-identity request is invalid-request
    detail: |
      This is the 2nd finding in family docs-lag-new-vocabulary. Rule: atlas text describing response codes should be checked against the plan's latest Revisions entry for the same flow; here revision (d) and handleSlotOperation both say invalid-request.
```
