# Boundary Review — pair#450 (whole-issue close)

| field | value |
|-------|-------|
| issue | 450 — pair main: own and clear the test failures that every worker reports as pre-existing |
| repo | pair |
| issue file | workshop/issues/000450-pair-main-own-and-clear-the-test-failures-that-every-worker-reports-as-pre-existing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 536ad9671d39bfb86934fdabd4eae78856ff34da..0342d2ef38b78325fc107add08e49691f90e8331 |
| command | sdlc close --issue 450 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T14:14:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Preflight passes here. Now running the Lua test, then reverting the fix to confirm the test goes red.

Found a pair-go failure with the env var set; checking whether it's the same leak or an unrelated failure.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The window does what it says, and I checked the main claims against the code. The restore-controller fix is reachable, and its new test passes at head. With the base `restore_controller.lua` swapped back in, that test fails at line 96 ("refused restore socket aborted review startup"). The launcher `TestMain` unset also holds up: with `COUCH_PAIR_DATA_DIR` set to a real directory, `TestLaunchNative*`/`TestCheckpoint*` pass at head. Restoring the base `session_quiescence_live_test.go` makes 5 of them fail. `restore_client.lua:102-104` does read a pane with no endpoint as `{legacy=true}`, so the atlas claim is correct. Commits e5dd82c4 (#341, 09-28) and 80cc80af (the #436 merge) exist and match the Log. Nothing blocks. One cheap gap is worth fixing: the preflight overstates how much it covers. This review's own sandbox passed `tests/host-preflight.sh` ("host-preflight ok"). Yet `go test ./cmd/pair-go` failed there with `fork/exec /bin/ps: operation not permitted`, with or without the env var set. That is exactly the "reads like a code bug" failure the issue is meant to stop. I did not re-run the whole `make test`, so "make test passes on main" rests on the Log.

**Strengths**
- `nvim/review/restore_controller.lua:42-47`: when `serverstart` fails, the fix removes the private socket dir, sets `socket_dir=nil`, warns with the cause, and still publishes. It reuses the existing legacy-pane contract in `restore_client` instead of adding a new state.
- `restore_controller_test.lua:86-103`: the test covers the real path. It checks that startup survives, that no endpoint is recorded, that the published meta has a token but no endpoint, that the warning names the cause, and that no directory is left behind. It also restores `serverstart` and `notify` afterwards.
- `session_quiescence_live_test.go:21-24`: the env input is cleared once per package in `TestMain`, after the helper-process branch. Tests that need a selected root (`selected_root_test.go:38`) still set it with `t.Setenv`.
- The diagnosis is backed by evidence: the failure reproduces at the "known-good" merge, and the lesson in `workshop/lessons.md` is a reusable rule.

**Critical findings**: none.

**Important findings**
1. **`tests/host-preflight.sh:11-33`: the probe misses sandbox refusals the suite depends on.** It only checks unix-socket bind and PTY open. In this review's sandbox it passed, yet `cmd/pair-go` `TestPublicPairCommandFamiliesIgnoreCouchStore` failed on `fork/exec /bin/ps: operation not permitted`. That sandbox failure looks like a code failure, which is what the issue is supposed to prevent. Three places claim more than the probe covers:
   - Plan item 3: "a sandboxed run cannot be mistaken"
   - `atlas/index.md`: "refuses a host that cannot…"
   - the header comment of `host-preflight.sh`

   **Fix:** add an exec probe for what the suite shells out to (at least `ps -p $$`), or narrow the wording in all three places to "refuses socket/PTY-denying hosts". (ARCH-PURPOSE)

**Minor findings**
- `host-preflight.sh:36`: if `python3` is missing or fails for any other reason, the script still blames the sandbox ("An agent sandbox does this"). Check `command -v python3` first and report that separately.
- `Makefile.local:213`: listing the preflight as the first prerequisite only gives "fail first, once" without `-k`. The Log's own baseline uses `make -k test`, and under `-k` every other prerequisite still runs and fails. Mention this in the comment, or have `test-host-preflight` abort the run.
- `workshop/lessons.md` says to clear the input "once per package, in TestMain, rather than in each test". The couchcmd fix instead pins it in one child env (`continuation_acceptance_test.go:254`). This works: couchcmd passed with the variable set. But it does not follow the lesson written in the same window.
- With no endpoint, `restore_client.lua:113` tells the user to "finish and close it before reopening". On a host that refuses the socket, reopening will not help. Consider showing the host-specific cause.

**Test coverage notes**
- The Lua regression test fails with the fix reverted; I ran it.
- The launcher env fix fails with the fix reverted (5 FAILs); I ran it.
- With a valid `COUCH_PAIR_DATA_DIR` set, `couchcmd` and `couchcore` pass.
- The preflight script has no test of its refusal branch. That is acceptable for a host probe, but its failure output is unverified.

**Architecture**
- **ARCH-DRY: pass.** No duplicated logic. The no-endpoint path reuses the existing legacy contract.
- **ARCH-PURE: pass.** The change is a small IO-shell guard in `serve()`, and the policy code is untouched.
- **ARCH-PURPOSE: flag.** This is Important finding 1: the preflight covers some of the sandbox's refusals, not all of them. Everything else matches the stated purpose. The Plan's coverage decision is right: `make test` already ends in `go test ./... -count=1`.

**Plan revision recommendations**
- Add a `## Revisions` entry that changes Plan item 3 to name the probed capabilities and drop "cannot be mistaken", unless an exec probe is added. Do the same for the atlas line.

```findings
findings:
  - id: new
    severity: Important
    family: preflight-claims-exceed-probe
    title: |
      host-preflight probes only socket+PTY; a sandbox that refuses exec of /bin/ps passes it and still fails tests that look like code bugs
    detail: |
      Observed in the review sandbox: host-preflight.sh reports ok, but go test ./cmd/pair-go fails TestPublicPairCommandFamiliesIgnoreCouchStore with "fork/exec /bin/ps: operation not permitted", with or without COUCH_PAIR_DATA_DIR set. Three claims overstate the probe: Plan item 3 ("cannot be mistaken"), atlas/index.md, and the host-preflight.sh header. Fix: add an exec probe (e.g. ps -p $$), or narrow all three to the probed capabilities. ARCH-PURPOSE.
  - id: new
    severity: Minor
    family: preflight-misattributes-cause
    title: |
      host-preflight.sh blames the sandbox for any python3 failure, including python3 missing
    detail: |
      A non-zero exit from the python heredoc, including "command not found", prints "An agent sandbox does this". Check command -v python3 first and report that separately.
  - id: new
    severity: Minor
    family: preflight-ordering-under-keep-going
    title: |
      Listing the preflight first does not stop make -k test from running every other suite
    detail: |
      Makefile.local:213. The Log's baseline uses make -k; under -k the other prerequisites still run and fail after the preflight fails. Document this or make the preflight abort the run.
  - id: new
    severity: Minor
    family: lesson-practice-mismatch
    title: |
      lessons.md says to clear ambient env once per package in TestMain, but couchcmd pins it in a single child env
    detail: |
      continuation_acceptance_test.go:254 pins COUCH_PAIR_DATA_DIR= on one command; launcher uses TestMain. It works (couchcmd passes with the variable set), but it does not follow the lesson written in the same window.
  - id: new
    severity: Minor
    family: legacy-pane-message-misleads
    title: |
      A pane with no endpoint on a socket-refusing host shows the "finish and close it before reopening" message, which cannot help
    detail: |
      restore_client.lua:113. On a host that refuses the socket, every reopened pane is also legacy. Consider a host-specific message.
```

---

## Re-review — 2026-10-11T14:29:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 450 — pair main: own and clear the test failures that every worker reports as pre-existing |
| repo | pair |
| issue file | workshop/issues/000450-pair-main-own-and-clear-the-test-failures-that-every-worker-reports-as-pre-existing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 536ad9671d39bfb86934fdabd4eae78856ff34da..b049190c7215d3d51fe7e62826ccadaecf922244 |
| command | sdlc close --issue 450 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T14:29:50-07:00 |
| verdict | SHIP |

## Review

Both fixes go red when reverted, and the checkout is restored. Next I'm checking the launcher env pin.

```verdict
verdict: SHIP
confidence: high
```

All five findings from round 1 are fixed, and I checked each against the code, not the commit messages. The preflight now probes for running `ps`, which round 1 called "exec of `ps`". In this review sandbox it fails at once and names the right cause: `exec ps: [Errno 1] Operation not permitted`. Both Lua fixes have regression tests that fail when the fix is reverted. The `COUCH_PAIR_DATA_DIR` reset in launcher's `TestMain` is needed: without it, 5 `TestLaunchNative*`/`TestCheckpoint*` tests fail when the variable leaks in, and with it they pass. Nothing blocks shipping.

Done-when: current. The Spec/Revisions changes after change-code only made the preflight and the pane message more accurate. Both clauses still cover the diff: `make test` and `go test ./...` passing, and the Log naming what broke `test-review` (the Log says no landing did; the cause was the sandbox host).

1. **Strengths**
   - `nvim/review/restore_controller.lua:40-48`: a refused socket is caught, its private directory is removed, and the pane still publishes, so review still works on such a host. The test at `restore_controller_test.lua:86-103` checks the endpoint is nil, the token is still published, the warning names the cause and nothing is left behind. It fails at line 96 without the fix.
   - `nvim/review/restore_client.lua:103-105`: separating `unserved` from `legacy` is a minimal change, and the client test fails at line 123 without it.
   - The investigation rules out the host before bisecting, which avoided a false "regression" bisect. The lesson at `workshop/lessons.md:1022` records that method.
   - `tests/host-preflight.sh` checks for `python3` before blaming the sandbox, and lists each refused capability on its own line.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new. One wording point, not raised as a finding: the lesson says the sandbox "refuses unix sockets and PTYs" but leaves out `ps`. That sentence describes the original investigation, so it is acceptable.
5. **Test coverage**
   - I reverted each fix in the disposable checkout:
     - the restore_controller change made its test fail;
     - the restore_client change made its test fail;
     - removing the launcher `TestMain` reset made 5 tests fail with `COUCH_PAIR_DATA_DIR` set.
   - All three passed again at head. `git status` was clean afterwards.
   - I did not run the couchcmd child-env pin, because this sandbox refuses running `ps`. By inspection it follows the same pattern.
   - I could not check `make test` passing on main here (the sandbox host is the documented blocker). That rests on the implementor's unsandboxed run recorded in the Log.
6. **Architecture**
   - ARCH-DRY: pass. The script header, the atlas and the Makefile comment each list the probed capabilities. That is three copies, all accurate now. If they drift, the script header is the one to keep as the source.
   - ARCH-PURE: pass. The Lua changes are local guards; the preflight is a thin IO probe.
   - ARCH-PURPOSE: pass. Every failure the issue lists is dealt with (fixed, pinned, or explained as the host), and the coverage decision is recorded. The pre-lock hook left for later is genuinely base-layer (ariadne) work, not the purpose of this issue.
7. **Plan revisions:** none. Round 1's `## Revisions` entry already matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      host-preflight.sh now probes running ps -p; it fails in this sandbox naming "exec ps"; atlas, header and Makefile comment name the three probed capabilities and say an unprobed refusal still fails as an ordinary test.
  - id: BR-2
    disposition: addressed
    note: |
      command -v python3 is checked first with its own message (host-preflight.sh:13-16).
  - id: BR-3
    disposition: addressed
    note: |
      Makefile.local:216-219 documents that make -k runs past the preflight; plain make (no -j in the makefiles) stops at the first prerequisite.
  - id: BR-4
    disposition: addressed
    note: |
      lessons.md now covers the child-env pin as well as TestMain; the launcher TestMain reset is shown to be needed (5 tests fail without it).
  - id: BR-5
    disposition: addressed
    note: |
      restore_client.lua:105,114 adds the unserved state with a host-specific message; the client test fails at line 123 when it is reverted.
```
