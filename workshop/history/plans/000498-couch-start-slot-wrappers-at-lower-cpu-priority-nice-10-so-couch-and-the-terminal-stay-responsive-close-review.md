# Boundary Review — pair#498 (whole-issue close)

| field | value |
|-------|-------|
| issue | 498 — couch: start slot wrappers at lower CPU priority (nice +10) so Couch and the terminal stay responsive |
| repo | pair |
| issue file | workshop/issues/000498-couch-start-slot-wrappers-at-lower-cpu-priority-nice-10-so-couch-and-the-terminal-stay-responsive.md |
| boundary | whole-issue close |
| milestone | — |
| window | d4235bae80b826533bfcdc8e974b27d08095f4df..31dd6cb9e4c26f45a825b82f423c6fca7dc8892e |
| command | sdlc close --issue 498 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:05:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff is read; the pgid assumption checks out for both runners. Next I'm running the new tests and probing the edge where Couch itself already runs at a higher niceness.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The mechanism is sound. Niceness is a field on the two real runners, so fakes can never reach the kernel. It is applied once, in the shared `startBlockedChild` handshake, while the helper is parked before exec. That means everything the slot forks inherits it with no race. Parsing is a pure function with `getenv` injected, and a bad value is refused both at singleton preflight and when the runner is built. Nothing here blocks the gate, but three things should be fixed:
- The `setpriority` value is absolute. If Couch already runs at a higher niceness than the configured one, every slot launch is refused.
- The plan's Design paragraph still describes a `NicedRunner` decorator that was dropped.
- `## Done when` still requires live measurements that the Revisions entry deferred.

I could not run the two new couchcore kernel tests here: `fork/exec /bin/ps: operation not permitted`, even outside the sandbox. `couchcmd` would not build in this checkout because the generated `runtimebundle` assets are missing. `TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority` passes.

Done-when: stale. It still requires "under a heavy suite Couch stays responsive (measured)" and ps on live slots. The 2026-10-12 Revisions entry moved both until after the operator restarts Couch, and says close rests on the in-repo proof. The clause was never rewritten to match.

**1. Strengths**
- `couchcore/launchhelper.go:55-65`: one authority. Niceness goes on the process group while the helper is parked, and a refused `setpriority` cancels and reaps the helper before the target runs.
- The design change recorded in the Log was the right call. With a decorator, a `FakeRunner`'s fake PIDs could have reniced real process groups. Making `Nice` a field avoids that.
- `couchcore/nice_test.go:18`: the test reads the grandchild's niceness, which proves inheritance through exec rather than just that the call was made. It also uses `ownNice+3`, so it works under a parent that is already niced.
- `couchcmd/nice.go`: a pure parser with injected `getenv`. It is checked at preflight (`singleton.go:113`) and when the runner is built (`run.go:677`), the same pattern as `captureSettings`.
- The pgid == pid assumption holds for both runners: `Setpgid` at `runner.go:124`, and creack `pty.StartWithSize`, which defaults to Setsid.

**2. Critical:** none.

**3. Important**
- **Absolute niceness lower than Couch's own fails every launch** (`launchhelper.go:58`). `setpriority(PRIO_PGRP, pid, nice)` sets an absolute value, and non-root users cannot lower niceness. So if Couch inherits a niceness above `COUCH_SLOT_NICE`, every launch fails with EPERM. Examples: Couch started from a slot shell at nice 10 with `COUCH_SLOT_NICE=5`, or under `nice -n 15` with the default of 10. Your own refusal test (`nice_test.go:55`) shows the failure path. Fix: apply `max(nice, current)` using `unix.Getpriority`, or skip when the current niceness is already at least the target. Then add a test where the requested niceness is below Couch's own and the launch still succeeds. Same family: the README claim "Couch itself keeps normal priority" assumes Couch runs at nice 0.
- **Plan Design paragraph contradicts the code.** It says "A `couchcore.NicedRunner` decorator lowers that group's priority…". There is no such type: niceness is a field on `ExecRunner` and `PtyRunner` (`runner.go:58`, `ptyrunner.go:26`). The Log records the change, but neither the Design text nor `## Revisions` does.
- **`## Done when` is stale** (see the line above). Rewrite it to say: the in-repo ps proof closes this item, and the live ps and `couch --actors` latency check happen after the restart. Or keep the item open until that check runs.

**4. Minor**
- `couchcmd/nice_test.go:43`: the default-runner check reads the real `os.Getenv`, so a developer with `COUCH_SLOT_NICE` set fails it. It also uses an unchecked `runner.(couchcore.ExecRunner)` assertion, which panics instead of failing. Inject an empty getenv and use the comma-ok form.
- `couchcore/nice_test.go:72` (`ownNice`) and the grandchild probe both depend on `/bin/ps`. `ownNice` could use `unix.Getpriority(PRIO_PROCESS, 0)` with no exec. The grandchild probe could re-exec the test binary to read its own priority, which makes the test runnable in restricted environments like this one.
- The README says the range is 0–20. Linux clamps at 19, and it's harmless.

**5. Test coverage notes**
- The Log says the mutation check was done: removing `setpriority` turns both couchcore tests red. I could not repeat it because of the ps restriction.
- The PtyRunner kernel path is covered only by the delegation-string contract test plus the shared `startBlockedChild`, which is acceptable given there is a single authority.
- Missing: a test where the requested niceness is below the inherited one (see Important #1).

**6. Architectural notes**
- ARCH-DRY: pass. There is one parser (`slotNiceSettings`) used at both entry points, and one application site (`startBlockedChild`).
- ARCH-PURE: pass. Parsing is pure with injected `getenv`, and the syscall sits in the existing IO handshake shell.
- ARCH-PURPOSE: pass, with a caveat. Both production runners are wired, and leaving out `taskpolicy` is legitimate because the Spec makes it opt-in after measurement. The live latency check, the measurable part of the purpose, is deferred until the restart; that is covered by the stale Done-when finding. The probe at `probes/couchstartrecovery/main.go:61` builds `ExecRunner` without `Nice`, which is fine for a probe.

**7. Plan revision recommendations**
- Add a `## Revisions` entry saying niceness is a field on `ExecRunner`/`PtyRunner`, applied in `startBlockedChild`, and that `NicedRunner` was dropped (fakes must not reach the kernel). Rewrite the Design paragraph to match.
- Rewrite `## Done when` to match the deferral, or record the live ps and latency results before close.

```findings
findings:
  - id: new
    severity: Important
    family: absolute-nice-below-inherited
    title: |
      setpriority uses an absolute value, so a COUCH_SLOT_NICE below Couch's own niceness makes every slot launch fail
    detail: |
      launchhelper.go:58 calls Setpriority(PRIO_PGRP, pid, nice). Non-root users cannot lower niceness, so with Couch at nice 15 (or nice 10 with COUCH_SLOT_NICE=5) every StartBlocked returns EPERM and cancels. Fix: apply max(nice, current) via unix.Getpriority, or skip when the current niceness is already at least the target; add a test where the requested niceness is below the inherited one. Same family: the README line "Couch itself keeps normal priority" assumes Couch runs at nice 0.
  - id: new
    severity: Important
    family: plan-contradicts-code
    title: |
      Plan Design paragraph still names a couchcore.NicedRunner decorator that does not exist
    detail: |
      The code puts a Nice field on ExecRunner/PtyRunner (runner.go:58, ptyrunner.go:26). Only the Log records the change; the Design text and Revisions do not.
  - id: new
    severity: Important
    family: done-when-stale
    title: |
      Done when still requires a live ps check and measured latency that Revisions deferred until after the restart
    detail: |
      Rewrite Done when so the in-repo grandchild ps proof closes the item and the live verification follows the operator's restart, or record those results before close.
  - id: new
    severity: Minor
    family: test-ambient-env-dependence
    title: |
      couchcmd nice_test reads the real os.Getenv and uses an unchecked type assertion
    detail: |
      nice_test.go:43 fails if COUCH_SLOT_NICE is set in the developer's shell, and panics instead of failing on a type mismatch. Inject an empty getenv and use the comma-ok form.
  - id: new
    severity: Minor
    family: test-ambient-env-dependence
    title: |
      couchcore nice tests need /bin/ps, which cannot run in restricted environments
    detail: |
      ownNice (nice_test.go:72) could use unix.Getpriority with no exec. The grandchild probe could re-exec the test binary to read its own priority.
```

---

## Re-review — 2026-10-12T14:13:41-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 498 — couch: start slot wrappers at lower CPU priority (nice +10) so Couch and the terminal stay responsive |
| repo | pair |
| issue file | workshop/issues/000498-couch-start-slot-wrappers-at-lower-cpu-priority-nice-10-so-couch-and-the-terminal-stay-responsive.md |
| boundary | whole-issue close |
| milestone | — |
| window | d4235bae80b826533bfcdc8e974b27d08095f4df..83d7e45fd07e3de796ec01f092f37469814baac7 |
| command | sdlc close --issue 498 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:13:41-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The BR-1 fix works. I wrote a throwaway test in this checkout that asks for a niceness below Couch's own (`Nice: -1`, ps not needed), and `StartBlocked` launches. When I reverted the `EACCES`/`EPERM` exemption in `launchhelper.go:64`, that test failed, so the fix is reachable and the regression is real. BR-2, BR-3 and BR-4 are fixed in the plan and the test. One thing blocks SHIP: the same fix changed how a refused `setpriority` is handled, but two documents still describe the old behaviour. `atlas/couch.md` says "a refused `setpriority` cancels the launch before exec", and Plan checklist item 1 promises a test of "a kernel-refused niceness that cancels before the target runs". The code now does the opposite. This is a repeat in family `plan-contradicts-code`. Also, neither shipped couchcore nice test could run here, because `/bin/ps` can't be exec'd in this environment even with the sandbox off. That is BR-5 happening for real, so I'm leaving it not-addressed at Minor. I couldn't run the couchcmd tests either: the package fails setup because `runtimebundle/embed.go` needs generated assets that this checkout doesn't have.

1. **Strengths**
   - Applying niceness inside the parked-helper window (`launchhelper.go:60-71`) means there's no race, nothing changes in argv or process identity, and nothing in the group has forked yet. The niceness then reaches the whole subtree because `Setpgid` is set (`runner.go:124`).
   - `Nice` is a field on the two real runners, not a decorator. A `FakeRunner` therefore can't renice a real process group.
   - `slotNiceSettings` is a small pure parser that takes `getenv` as a parameter. It runs both where the runner is built and at singleton preflight (`singleton.go:113`), so a bad value refuses to start.
   - The BR-1 test (`couchcore/nice_test.go:47`) pins the "keep the inherited niceness" behaviour and fails without the fix (I checked this with my ps-free equivalent).

2. **Critical**: none.

3. **Important**
   - **This is the 2nd finding in family `plan-contradicts-code`.** The rule is: when a review-fix round changes a behaviour, grep every prose restatement of it in the same round (plan Design/checklist, atlas, README, doc comments) and rewrite them all. The places in this window:
     - (a) `atlas/couch.md:2370-2371`: "a refused `setpriority` cancels the launch before exec". This is wrong now: `EACCES`/`EPERM` keep the inherited niceness, and only other errors cancel.
     - (b) Plan checklist item 1: "a kernel-refused niceness that cancels before the target runs". The test now asserts that a refused niceness *launches*.
     - (c) The Log line "refused niceness returns a handle" describes the mutation check from before the fix. It's history, but add a note that it was superseded.
     - These are already consistent: the README, the Design paragraph ("kernel refusal is not an error") and the `startBlockedChild` doc comment.

4. **Minor**
   - The non-`EACCES`/`EPERM` cancel branch (`launchhelper.go:65-70`) has no test.
   - The README paragraph has one line that overruns the wrap width ("…since only root can lower it. It applies to…").
   - In `couchcmd/nice_test.go:45`, the third `consoleRunnerFor` call discards its error.

5. **Test coverage notes**
   - The `Done when` clause has two modes (the configured niceness, or Couch's own when that is higher), and both are covered:
     - Configured niceness: tested on a grandchild.
     - Couch's own when higher: tested on the direct child.
   - In environments that can't exec `ps`, both tests fail outright instead of skipping (BR-5).
   - I couldn't run the couchcmd tests (embed-asset setup failure in this checkout, not a code problem).

6. **Architecture**
   - **ARCH-DRY: pass.** The niceness logic lives in one authority, `startBlockedChild`, and parsing happens in one place, `slotNiceSettings`, which is reused at preflight.
   - **ARCH-PURE: pass.** Parsing is pure and tested without IO. The `setpriority` call is a thin seam inside the existing IO function.
   - **ARCH-PURPOSE: pass.** Both real runners derive from the same setting. The `taskpolicy` QoS clamp is legitimately out of scope (the Spec makes it opt-in after measurement). The live check after the operator's restart is hub-owned and was deferred in Revisions with the hub's agreement.

7. **Plan revisions**: add a `## Revisions` entry recording that checklist item 1's "refused niceness cancels" became "refused niceness keeps the inherited one" (BR-1), and correct the checklist text and the atlas line to match.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      EACCES/EPERM now keep inherited nice; a reviewer ps-free test (Nice -1) is green with the fix and red when the exemption at launchhelper.go:64 is reverted.
  - id: BR-2
    disposition: addressed
    note: |
      The Design paragraph now names the Nice field on ExecRunner/PtyRunner applied in startBlockedChild; Revisions records the dropped NicedRunner.
  - id: BR-3
    disposition: addressed
    note: |
      Done when splits into the in-repo grandchild ps proof and a post-restart hub-owned live check; Revisions records the move.
  - id: BR-4
    disposition: addressed
    note: |
      The test injects getenv via consoleTraceConfig and uses comma-ok assertions (couchcmd/nice_test.go:33,40,46).
  - id: BR-5
    disposition: not-addressed
    note: |
      Reproduced here: both couchcore nice tests fail with "fork/exec /bin/ps: operation not permitted". At minimum t.Skip when ps cannot exec; ownNice could use unix.Getpriority with a per-GOOS offset.
findings:
  - id: new
    severity: Important
    family: plan-contradicts-code
    title: |
      atlas and Plan checklist still say a refused setpriority cancels the launch; code now keeps the inherited niceness
    detail: |
      2nd finding in plan-contradicts-code. Rule: a fix round that changes a behaviour must grep and rewrite every prose restatement of it in the same round. Places: atlas/couch.md:2370-2371 ("a refused setpriority cancels the launch before exec"); Plan checklist item 1 ("a kernel-refused niceness that cancels before the target runs"); the Log mutation-check line "refused niceness returns a handle" (pre-fix, annotate as superseded). README, Design, and the startBlockedChild comment already match the code.
  - id: new
    severity: Minor
    family: untested-error-branch
    title: |
      The non-EACCES/EPERM setpriority cancel branch (launchhelper.go:65-70) has no test
    detail: |
      This is the only remaining path that cancels a launch over niceness; no fixture reaches it.
```

---

## Re-review — 2026-10-12T14:14:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 498 — couch: start slot wrappers at lower CPU priority (nice +10) so Couch and the terminal stay responsive |
| repo | pair |
| issue file | workshop/issues/000498-couch-start-slot-wrappers-at-lower-cpu-priority-nice-10-so-couch-and-the-terminal-stay-responsive.md |
| boundary | whole-issue close |
| milestone | — |
| window | d4235bae80b826533bfcdc8e974b27d08095f4df..1d042f8a4552ef596fb9b5abf3defbfe149a7a76 |
| command | sdlc close --issue 498 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:14:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Round 2 fixed BR-6, the only blocking finding left. The atlas (`atlas/couch.md:2367-2373`) now matches the code. A kernel refusal (EACCES/EPERM) keeps the niceness the slot inherited, and any other `setpriority` error still cancels the launch, which is what `launchhelper.go:61-70` does. Plan checklist item 1 has been rewritten with a revision note, and the Log mutation-check line now says the original wording was superseded in close round 1. README, Design and the `startBlockedChild` doc comment already matched the code. I grepped atlas/ and README for nice/cancel statements and found no other stale ones. BR-5 and BR-7 are still open but Minor, so they don't block. No new findings.

1. **Strengths**
   - `startBlockedChild` (`launchhelper.go:60-71`) is the single place that applies niceness, and both real runners pass `r.Nice` to it. The delegation contract test was updated to match (`plan_contract_test.go:820-821`).
   - The niceness is set while the helper is parked before exec, so it is inherited without a race and without touching argv or the process identity. The grandchild test checks that inheritance through the exec, not just a call made on the helper.
   - `slotNiceSettings` is a pure function with an injected getenv. It covers default, off, both limits and malformed input (`couchcmd/nice_test.go:10-22`). It is called both where the runner is built and at singleton preflight, which matches the existing `captureSettings` pattern.
   - The lessons.md entry states the rule for the whole `plan-contradicts-code` family, not just this instance.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** BR-5 and BR-7 are still open; details in the findings block below.

5. **Test coverage**
   - Run in this sandbox, `TestBlockedStartBelowCouchsOwnNicenessKeepsTheInheritedOne` failed with `fork/exec /bin/ps: operation not permitted`. That is BR-5 showing up, not a product bug.
   - The couchcmd tests could not compile here (`runtimebundle` embed assets are missing from the review checkout), so I relied on reading them rather than running them.
   - The grandchild test can't tell a pass from a no-op when its target equals Couch's own niceness, which only happens if Couch already runs at nice 20.

6. **Architecture**
   - **ARCH-DRY: pass.** One application site, and both runners delegate to it.
   - **ARCH-PURE: pass.** Parsing is pure; the kernel call is a thin IO edge inside the existing shell.
   - **ARCH-PURPOSE: pass.** Every slot launch goes through `StartBlocked` and both production runners carry `Nice`. The live `ps` check is deferred to after the hub's Couch restart, as the Revisions record, which is legitimately separate from this code change.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      ownNice (couchcore/nice_test.go:79) still runs /bin/ps; in this sandbox it failed with "fork/exec /bin/ps: operation not permitted". unix.Getpriority(PRIO_PROCESS, 0) would remove the exec for the parent; the grandchild probe still needs ps or a re-exec of the test binary.
  - id: BR-6
    disposition: addressed
    note: |
      atlas/couch.md:2367-2373 now says EACCES/EPERM keeps the inherited niceness and other errors cancel, matching launchhelper.go:61-70. Plan item 1 was rewritten with a close-round-1 note and the Log mutation line is annotated as superseded. README, Design and the doc comment already agreed. A grep of atlas/ and README for nice+cancel found no other stale statements.
  - id: BR-7
    disposition: not-addressed
    note: |
      The non-EACCES/EPERM cancel branch (launchhelper.go:63-70) is still untested. Reaching it needs an injectable setpriority seam, e.g. ESRCH on a helper that has already exited.
```
