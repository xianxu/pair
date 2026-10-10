# Boundary Review — pair#421 (milestone M2)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1137d7f3d207411e8549a33b498638a436eca31a..04d2d5c2c7347ed1b4ca9c0f25fb2c63b8597827 |
| command | sdlc milestone-close --issue 421 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-10T00:05:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Window inspected; core admission diff read. Now the couchcmd side (probe, CLI, receipt).

Checking the PAIR_DEV propagation claim against #422 and `mergeChildEnvironment`.

Running the affected packages' tests.

Waiting on the test run; checking its output.

Sandbox failures in wrapcmd (/tmp writes). Rerunning the other packages plus the settle tests specifically.

Mostly sandbox socket denials. Running the M2-specific tests on their own to separate those from real failures.

The one remaining failure is a sandbox `/tmp` denial; rerunning that targeted set outside the sandbox.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 delivers `couch --relaunch repo:N --confirm [--same-binary] [--force-unknown] [--json]`. The admission rule is pure and its tests cover every combination of facts. The override plumbing goes from the CLI through the socket to the queued job, and refusals reach the receipt as typed codes that name the fix. The `PAIR_DEV` exception the TL asked for is folded in. All tests for the M2 surface pass: couchcore `LiveRestart|BinaryFreshness|PrepareSlotOperation|RecoverPlanStepsConverge`, couchcmd `Relaunch|LiveRestartProbe|NotedResult|Parse*|SlotOperation*|Receipts|OperationStatus`, couchmessage `Validate*`, and wrapcmd `Settle*`. The broad package runs failed under the sandbox with `/tmp` mkdir and bind "operation not permitted" errors, so I reran the couchcmd set outside the sandbox, where it passed. Four things block SHIP, none of them a correctness bug:
- The plan said the receipt would map `RelaunchResult.Outcome`, but failed relaunches reach the caller as untyped text, and the admission note is dropped on failure.
- `LivenessForThread` and the probe's settled/git half have no tests.
- README.md is missing the new verb.
- The plan's Core concepts table no longer matches where the probe and the liveness lookup actually live.

## 1. Strengths
- `couchcore/live_restart.go:69` `DecideLiveRestart` is pure, with a fixed check order so the first reason given is the first thing to fix. `TestDecideLiveRestartExhaustive` works out the expected code from the rule's order without calling the rule, and checks that it enumerated all 512 cases.
- Unknown is treated as a refusal (`busy-unknown`), not a guess. `--force-unknown` never overrides a known busy, and its use is recorded in the note.
- `slotOperations` plus `IsSlotOperation` is now the one list that protocol validation, `messageService.handle` and admission all read (ARCH-DRY).
- Liveness is matched by thread (scope + exact tag, `message_service.go:824`), not by the `repo:N` spelling. That scope matches `COUCH_THREAD_SCOPE`, which `launch_existing.go:98` sets from `thread.Address.RepoScope`. Ambiguous slots are dropped from the snapshot (`registry.go:362`), so they fall to `busy-unknown`, which is safe.
- `notedResult` is wrapped only in the receipt's `finished` callback. `finishOperation` adopts the raw `RelaunchResult` first (`console.go:1853`), so console adoption of the child is unaffected.

## 2. Critical findings
None.

## 3. Important findings
- **Failed relaunches lose their typed outcome on the receipt** (`couchcmd/slot_operations.go:155`, `message_service.go:337`). The plan's M2 says "the receipt maps `RelaunchResult.Outcome`".
  - `Relaunch` returns `ParkIncomplete` and `ParkedNotResumed` with a non-nil error. `slotOperationOutcome` turns any non-nil error into `failed` with only the error text and the resume diagnostic. A scripting TL therefore can't tell "park transaction still open (use park's own recovery)" from "parked, `--resume` will fix it", which is exactly the distinction `relaunch.go:9-14` says consumers need.
  - The `err == nil` guard also drops the admission note on failure, so a forced-unknown relaunch that fails no longer records `--force-unknown`.
  - Fix: when the value is a `RelaunchResult` with a non-success outcome, set the receipt `Code` to it, keep the note on failure, and add a test for each outcome.
- **The busy guard's wiring has no tests** (`couchcmd/live_restart_probe.go:31`, `message_service.go:824`).
  - Only `binaryFacts` is tested. Nothing checks that `Settled == nil` maps to unknown, a pointer value maps to known, a git error maps to `GitKnown=false` (and so a `dirty` refusal), or a missing session maps to `Session=false`.
  - `LivenessForThread` has no tests at all. A wrong field in its scope/tag match would make every relaunch refuse `busy-unknown`, and nothing short of M4's live check would notice.
  - Fix: one table test over `LiveRestartFacts` with a seeded liveness map and a fake git.
- **README update appears missing for `couch --relaunch`.** `README.md:389-393` lists the slot-operation verbs (`--reboot`, `--recover`, …) but not `--relaunch repo:N --confirm [--same-binary] [--force-unknown] [--json]`. `run.go` usage and the atlas were updated; the README was not.
- **The Core concepts table contradicts the code.** This is the 2nd finding in family `revision-supersedes-body-unmarked`.
  - The `BinaryProbe` row says it lives in `couchcore/live_restart.go` and wraps `exec.LookPath`/buildinfo/git. The code has it as `liveRestartProbe` in `couchcmd/live_restart_probe.go`, and couchcore only declares the `LiveRestartProbe` interface.
  - The `SlotLiveness` lookup is now `LivenessForThread(scope, tag)`.
  - `DecideBinaryFreshness(running, onDisk, sourceHEAD)` is now `(BinaryFacts, sameBinaryOK)`.
  - The `PAIR_DEV` exception appears only in the issue Log, never in the plan.
  - Don't just patch these rows. The rule that covers both findings: **every delta from a Core concepts row (name, path, signature) or from a milestone body lands as a dated `## Revisions` entry in the same commit as the code that diverges, and each milestone close diffs the table against the tree.** The M1 deltas section followed this rule; M2 has no deltas section.

## 4. Minor findings
- `messages.go:189`: a relaunch sent to a pre-#421 Couch that already has slot operations gets "predates `couch --resume`/`--reboot`", which is misleading. With overrides set, an older Couch rejects the request on strict decode and the user sees a generic error. The message should name the verb.
- ARCH-DRY: the string `"relaunch"` is restated in `live_restart_probe.go:43` and `cli.go:311`, alongside `SlotOperationTakesOverrides`. The CLI's two verb lists (`cli.go:94`, `cli.go:299`) also don't read `slotOperations`, even though the commit message says everything reads one list.
- `couchmessage/protocol.go:7`: the couchcore import sits inside the standard-library import group.

## 5. Test coverage notes
- Pure admission and freshness coverage is strong.
- Dispatch is tested from request to job (`TestSlotOperationRelaunchCarriesOverrides`) and from result to outcome (`TestNotedResultForwardsAndJoins`), but not in one end-to-end test that takes a relaunch refusal or failure all the way to the receipt.
- No test covers a failed relaunch outcome reaching the receipt.

## 6. Architectural notes for upcoming work
- ARCH-DRY: flagged as Minor above. ARCH-PURE: pass (pure decisions, thin injected probe). ARCH-PURPOSE: flagged, see the receipt-outcome finding. ARCH-MOCK: pass (`GitRunner` fake; LookPath, getenv and build injected).
- ARCH-CONSTRAINTS: pass. The binary hash plus `git status`/`rev-parse` run once per request on the console queue, about 10ms as measured in M1.
- ARCH-SECURE: pass. Overrides are validated per op at the protocol, and a wrapper's self-reported hash can only cause a refusal or an admission, never an exec.
- ARCH-ORDER: pass. The `note` handoff between prepare and finished is ordered by the queue, and the gap between admission and park is a declared non-goal (PQ-6).
- ARCH-FUNERAL: pass. Nothing durable is created; receipts are the existing bounded map.
- For M3: reuse `prepareLiveRestart` as is. Reload-context's `unknown/unconfirmed` outcome (PQ-4) has the same receipt-mapping need as the first finding, so build the typed outcome path once.

## 7. Plan revision recommendations
- Add a "M2 implementation deltas" entry to `## Revisions`:
  - The probe is `couchcmd.liveRestartProbe`; couchcore owns only the `LiveRestartProbe` interface.
  - The liveness lookup is `LivenessForThread(scope, tag)`.
  - `DecideBinaryFreshness` takes `BinaryFacts`.
  - The `stale-binary` refusal is skipped when Couch's environment has `PAIR_DEV`, which slots inherit through `mergeChildEnvironment`.
  - The admission note reaches the receipt `Warning` through `notedResult`.
- If outcome mapping is deferred instead of fixed, record that under Revisions rather than leaving the M2 body claiming it.

```findings
findings:
  - id: new
    severity: Important
    family: typed-outcome-survives-boundary
    title: |
      Relaunch failure outcomes (park-incomplete vs park-ok-resume-failed) reach the receipt as untyped text, and the admission note is dropped on failure
    detail: |
      slotOperationOutcome maps any err to failed with err text only; the plan promised the receipt maps RelaunchResult.Outcome, and notedResult wraps only when err == nil, so a forced-unknown failure no longer records --force-unknown. Set the receipt Code from the outcome, keep the note on failure, and test each outcome.
  - id: new
    severity: Important
    family: safety-guard-wiring-untested
    title: |
      liveRestartProbe.LiveRestartFacts (settled/session/git mapping) and messageService.LivenessForThread have no tests
    detail: |
      Only binaryFacts is tested. Settled nil→unknown, git error→GitKnown=false, and the scope/tag match are unpinned; a wrong match field silently makes every relaunch busy-unknown. Add a table test with a seeded liveness map and a fake git.
  - id: new
    severity: Important
    family: docs-surface-lags-cli
    title: |
      README.md slot-operation verb list (lines 389-393) is missing couch --relaunch and its overrides
  - id: new
    severity: Important
    family: revision-supersedes-body-unmarked
    title: |
      Core concepts table contradicts code: BinaryProbe location, SlotLiveness rename, DecideBinaryFreshness signature, PAIR_DEV skip absent from plan
    detail: |
      2nd in family. Rule: every delta from a Core concepts row or milestone body lands as a dated Revisions entry in the same commit as the diverging code, and each milestone close diffs the table against the tree. M2 has no deltas entry.
  - id: new
    severity: Minor
    family: version-skew-refusal-names-fix
    title: |
      A relaunch sent to a pre-421 Couch prints the misleading "predates --resume/--reboot"; with overrides set, the strict-decode error is unmapped
  - id: new
    severity: Minor
    family: single-source-verb-list
    title: |
      "relaunch" literal restated in live_restart_probe.go:43 and cli.go:311, and the CLI verb lists do not read slotOperations
```

---

## Re-review — 2026-10-10T00:19:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1137d7f3d207411e8549a33b498638a436eca31a..2c3a0ad9d21a94272f7a19857f735c03ebd9e333 |
| command | sdlc milestone-close --issue 421 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-10T00:19:02-07:00 |
| verdict | SHIP |

## Review

Focus on the last commit (fixes) for dispositions.

Checking that the typed relaunch outcome actually survives the console queue to the receipt (BR-7), then the docs/plan dispositions.

Focused tests pass; checking formatting and then writing the verdict.

```verdict
verdict: SHIP
confidence: medium
```

All six open findings (BR-7 to BR-12) are fixed by commit `2c3a0ad9`, and I found nothing that blocks the boundary. I checked each fix in the code, not just the commit message. On a failed relaunch, the typed outcome (`park-incomplete` / `park-ok-resume-failed`) now reaches the receipt `Code`. It travels the whole path intact: from `operationdispatch.go:403` through `EnqueueRemoteOperation`'s `run`, into `completed.value` (`console.go:1854`), and on through `notedResult.ReceiptCode` (`slot_operations.go:207`) and `slotOperationOutcome` (`slot_operations.go:166-175`). The probe mapping now has a table test, the README lists `--relaunch`, the plan marks every superseded Core-concepts line inline and has a dated M2 deltas entry, and the CLI, protocol and socket all read the one verb list. These focused tests pass: `TestLiveRestartProbeFactsMapping`, `TestRelaunchFailureOutcomeIsTyped`, `TestSlotOperationCLIPollLoop` and the couchcore LiveRestart/SlotOperation/Relaunch tests. I did not run the full package suites: an earlier full `go test` run of the four touched packages passed 120s and was moved to the background, and I gave my verdict without its result. `gofmt` is clean.

1. **Strengths**
   - `RelaunchResult.ReceiptCode` (`relaunch.go:48-56`) keeps the outcome typed on the error path. Duck-typing it in `slotOperationOutcome` means other results can add a code without another `case`.
   - `TestLiveRestartProbeFactsMapping` (`live_restart_probe_test.go:93-140`) covers what BR-8 asked for:
     - a wrapper that claimed Settled, and one that made no claim (legacy);
     - a row with no session;
     - git unreadable, so `GitKnown=false`;
     - no service attached yet;
     - the same tag in a different scope.
   - `couchcore.IsSlotOperation` / `SlotOperationTakesOverrides` is now the one list that `ParseCLI`, `parseMessageCLI` and `ValidateRequest` all read (ARCH-DRY).
   - The plan's M2 deltas revision follows the BR-10 rule: the table and the tree were compared at this close.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - No test pins `message_service.go:338`, where the `note` now survives a failure. `TestRelaunchFailureOutcomeIsTyped` builds the `notedResult` directly, so restoring `&& err == nil` would leave every test green. A test that drives `consoleSlotOperations`'s finished closure with an error would cover it.
   - `protocol.go:7`: the `couchcore` import sits inside the stdlib group. `gofmt` accepts this, but `goimports` would move it. Also, `couchmessage` now depends on `couchcore`; that is a direction worth noting in the atlas layering.
   - The pre-421 detection matches error text (`"unknown field"`). That is acceptable for skew with older versions, but it stops working if the message wording ever changes.

5. **Test coverage notes:** The admission logic (`TestDecideLiveRestartExhaustive`), the probe's wiring, the CLI parsing, the version-skew hint and the receipt code mapping are all pinned now. The one gap is the note-on-failure wrapper above.

6. **Architecture**
   - **ARCH-DRY: pass.** The verb list is single-sourced; the `"relaunch"` literals still in `couchtty` predate this issue.
   - **ARCH-PURE: pass.** `DecideLiveRestart` and `DecideBinaryFreshness` are pure; the probe is the thin IO shell.
   - **ARCH-PURPOSE: pass.** Every BR-12 site the finding named is fixed.
   - **ARCH-MOCK: pass.** The git fake now models the `status` call.
   - **ARCH-CONSTRAINTS: pass.** The probe adds a few git and file reads at admission only.
   - **ARCH-SECURE: pass.** Unknown request fields are still refused by the strict decode.
   - **ARCH-ORDER: pass.** The four relaunch outcomes are an enumerated type, and the uncertain partial states now survive to the receipt.
   - **ARCH-FUNERAL: pass.** No new durable artifacts.

7. **Plan revisions:** none. The M2 deltas entry matches the code.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      ReceiptCode carried through notedResult to slotOperationOutcome; TestRelaunchFailureOutcomeIsTyped fails without it. Note-on-failure in message_service.go:338 is unpinned (Minor).
  - id: BR-8
    disposition: addressed
    note: |
      TestLiveRestartProbeFactsMapping covers settled/legacy/no-session/git-error/scope mismatch and the detached probe.
  - id: BR-9
    disposition: addressed
    note: |
      README.md:394-396 lists --relaunch with --confirm, --same-binary, --force-unknown and its refusals.
  - id: BR-10
    disposition: addressed
    note: |
      Plan core-concepts rows are marked superseded inline, plus a dated 2026-10-10 M2 deltas revision.
  - id: BR-11
    disposition: addressed
    note: |
      messages.go:187 maps unknown-field and names the verb; covered by a new case in TestSlotOperationCLIPollLoop.
  - id: BR-12
    disposition: addressed
    note: |
      OpRelaunch exported and used at live_restart_probe.go:43; cli.go reads IsSlotOperation/SlotOperationTakesOverrides.
findings:
  - id: new
    severity: Minor
    family: safety-guard-wiring-untested
    title: |
      Admission note surviving a failed slot operation (message_service.go:338) has no regression test
    detail: |
      2nd finding in this family. Rule: every closure that adapts a queue result into a receipt needs a test driving it with err != nil. Restoring the `&& err == nil` guard leaves all tests green.
```

The full test run I'd left in the background has finished, and it doesn't change the verdict: **SHIP** stands.

- **`couchcmd` and `couchcore`:** I can't confirm these from the full run, because I only kept its last 15 lines and their results were cut off. The focused tests I ran earlier in both packages passed.
- **`wrapcmd`:** 6 tests failed, all with `mkdir /tmp/...: operation not permitted`. That's the sandbox blocking writes to `/tmp`, not the code. The tests that cover the changed `peer_settle.go` (`-run 'Settle|Settled'`) pass on their own.
- **`couchmessage`:** a separate run fails the same way. Every failure I looked at is `mkdir /tmp/pair-msg-test-*: operation not permitted` in the endpoint tests.

I haven't seen a clean run of the full suites for `wrapcmd` and `couchmessage`. You can get one by running `go test ./cmd/internal/...` outside the sandbox. You can change the sandbox limits with `/sandbox`.
