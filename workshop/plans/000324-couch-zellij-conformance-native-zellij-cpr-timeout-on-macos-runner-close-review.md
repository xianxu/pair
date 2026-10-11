# Boundary Review — pair#324 (whole-issue close)

| field | value |
|-------|-------|
| issue | 324 — couch-zellij-conformance: native zellij CPR timeout on macOS runner |
| repo | pair |
| issue file | workshop/issues/000324-couch-zellij-conformance-native-zellij-cpr-timeout-on-macos-runner.md |
| boundary | whole-issue close |
| milestone | — |
| window | d81b069e6293ce2e045c2adf32833d51b029290e..04729d0e110766fd19d2afa845d71f70d337e148 |
| command | sdlc close --issue 324 |
| reviewer | claude |
| timestamp | 2026-10-10T19:04:46-07:00 |
| verdict | SHIP |

## Review

I'm reviewing the #324 diff. The core fix and the fixtures look sound, and I'm confirming a few last claims about the parser before giving the verdict.

```verdict
verdict: SHIP
confidence: high
```

The root cause is found and fixed with evidence. Zellij 0.45.1 on macOS reads its config dir from HOME. On a fresh runner HOME it treats the launch as a first run, starts its default layout and ignores `--new-session-with-layout`. `terminal_native_test.go` now pins `--config-dir`, as production `LaunchSession` already does (`cmd/internal/launcher/osruntime.go:145`). The couchcore live fixtures are updated for the #355 ownership evidence through `AddOwnerWrapperPane`, which matches the `pair` + `wrap` branch of `ClassifySessionOwner` (`session_owner.go:76-79`). The three cases that still fail are excluded by name and handed to pair#432 in a dated `## Revisions` entry. The job stays blocking. Nothing blocks SHIP. The findings below are doc-accuracy and fixture-hygiene nits.

1. **Strengths**
   - The fix follows the evidence instead of raising the timeout. The Log records the runner zellij.log (tab-bar/status-bar plugins), a probe run, and a local reproduction with `HOME=$(mktemp -d)`. This meets the Spec's "do not just raise the timeout" rule.
   - `AddOwnerWrapperPane` (`cmd/internal/pairlifecycletest/live_zellij.go:153`) is a well-reasoned seam. It symlinks `/bin/sh` as `pair` and runs a `wrap` script without `exec`, so zellij sees the argv the probe needs. The ownership check itself is not weakened.
   - `standInPaneScript` is one shared helper used by both the continuation and recovery fixtures (ARCH-DRY). Its quoting (`'\''`) is correct.
   - The `-skip` pattern `^TestRecoveryRealHelperAndSessionConformanceLive$/^checkpoint$` matches the real subtest name (`recovery_conformance_live_test.go:49`).
   - I checked that the unanchored `-test.run=^TestRecoveryLiveAgentStandIn` / `^TestContinuationZellijAgentStandIn` prefixes match no other test name.
   - The failure-only "Dump Zellij logs" step keeps the diagnostic that cracked this issue.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **The doc claim about the parser is inaccurate.** `standInPaneScript`'s comment (`quiesce_conformance_live_test.go:103-107`) and `atlas/couch.md` both say the probe "refuses JSON, quotes and `$`". `splitObservedCommand` actually parses quotes. It refuses only unquoted `;|&<>` + backtick + `$`, and `$`/backtick inside double quotes. JSON is not refused: its `"` is read as quoting, so the argv gets split wrongly. Reword both sites to "rejects `$`, backticks and shell operators, and misparses JSON's quotes".
   - **The atlas tells fixtures to pin `--config-dir`, but the shared fixture doesn't.** `StartControlledZellijWithOptions` (`live_zellij.go:48-52`) passes only `--config`. On a fresh HOME it gets the first-run default layout too. That is harmless today: it applies layouts with `action new-tab --layout`, plugin panes are skipped by `IsPlugin`, and CI is green. Still, it is the one remaining zellij launch in this family without the pin. The other members are the native fixture (fixed) and production (already pinned).
   - **A failed stub setup is never retried.** In `AddOwnerWrapperPane`, `f.stubDir` is assigned before `Symlink`/`WriteFile` succeed. If either fails, the half-built dir is kept and the next call skips setup. Assign `stubDir` only after both succeed.
   - **The first recovery fixture leaks its stub.** On the cold recovery path, `fixture` is reassigned (`recovery_conformance_live_test.go:235`), and the `t.Cleanup` closure only closes the new one. The first fixture's new stub dir is never removed (the client leak predates this change). That path is currently excluded by name.
   - **The stub dir can outlive a crashed test.** It comes from `os.MkdirTemp("", …)` rather than a test-scoped dir. That's acceptable because the helper has no `*testing.T`.

5. **Test coverage notes**
   - The fix lives in test fixtures. Its evidence is a local fresh-HOME reproduction that times out without the fix and passes with it, plus CI dispatch run 38102384802 (step 1 and step 2 green).
   - The Done-when clause about the weekly schedule can only be confirmed after merge. The Plan honestly leaves that to the PR check and the first scheduled run.

6. **Architectural notes**
   - ARCH-DRY: **pass.** `standInPaneScript` is used by both stand-in sites, and `AddOwnerWrapperPane` sits on the shared fixture type rather than being copied per test.
   - ARCH-PURE: **pass.** This is test-fixture glue only; no product logic changed.
   - ARCH-PURPOSE: **pass.** Step 1, the issue's actual purpose, is fixed at the root cause. The deferral to #432 is a separable product-semantics question, recorded in a Revision and in #432's Done-when, and it is not the point of this issue. The `--config-dir` sweep note above is the one family sibling still open.
   - For #432: removing the exclusions means deleting the `-skip` and restoring the three names to `-run`. Both the Makefile comment and the workflow comment already say so.

7. **Plan revision recommendations:** none. The Plan and its Revision match the code.

```findings
findings:
  - id: new
    severity: Minor
    family: doc-claim-matches-code
    title: |
      The comment and atlas say the ownership probe "refuses JSON, quotes and $"; splitObservedCommand parses quotes
    detail: |
      Sites: quiesce_conformance_live_test.go standInPaneScript doc comment, and atlas/couch.md "Ownership evidence" bullet. The parser refuses unquoted ;|&<>`$ and $/backtick inside double quotes; JSON quotes get read as quoting and split the argv wrongly rather than being refused. Reword both.
  - id: new
    severity: Minor
    family: zellij-fixture-config-dir-pin
    title: |
      The shared ControlledZellij fixture still launches without --config-dir, though the atlas says to pin it
    detail: |
      Family members: terminal_native_test.go (fixed), launcher.OSRuntime.LaunchSession (already pinned), pairlifecycletest.StartControlledZellijWithOptions (live_zellij.go:48-52, unpinned). Harmless today, since layouts are applied via action new-tab; pin it for consistency, or narrow the atlas wording to --new-session-with-layout launches.
  - id: new
    severity: Minor
    family: fixture-resource-cleanup
    title: |
      AddOwnerWrapperPane sets stubDir before setup succeeds, and the cold recovery path leaks the first fixture's stub dir
    detail: |
      live_zellij.go:163-174: assign f.stubDir only after Symlink and WriteFile succeed, so a failed setup is retried. recovery_conformance_live_test.go:235 reassigns fixture, so the Cleanup closes only the new one and the first stub dir is left behind (that path is currently excluded under pair#432).
```

---

## Re-review — 2026-10-10T19:11:13-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 324 — couch-zellij-conformance: native zellij CPR timeout on macOS runner |
| repo | pair |
| issue file | workshop/issues/000324-couch-zellij-conformance-native-zellij-cpr-timeout-on-macos-runner.md |
| boundary | whole-issue close |
| milestone | — |
| window | 61d85762959a6d39bc80961a379ca09cc8bb3183..c17bc6b33cc376eb4da79cc1960476c870a88d5a |
| command | sdlc close --issue 324 |
| reviewer | claude |
| timestamp | 2026-10-10T19:11:13-07:00 |
| verdict | SHIP |

## Review

Checking the prior BR-3 leak and whether the cold recovery path still runs in CI.

Prior findings checked; one doc-claim instance left to confirm. Writing up the verdict.

```verdict
verdict: SHIP
confidence: high
```

The window does what the issue set out to do. It finds the root cause of the CI-only failure: zellij falls back to its first-run default layout when HOME is fresh. It fixes the native fixture the way production does, by pinning `--config-dir`. It updates the stale couchcore live fixtures so they present the ownership evidence #355 requires: a `pair wrap` stand-in pane, plus a script launcher so JSON env values don't break the argv parser. It also names the three still-failing cases and hands them to pair#432. That handoff is recorded in the issue's `## Revisions`, the Makefile, the workflow and the atlas. Round 1's fixes hold:
- **BR-1:** the prose now matches `splitObservedCommand`.
- **BR-2:** the atlas wording is narrowed.
- **BR-3:** `stubDir` is now assigned only after setup succeeds.

Two Minor items are left: the cold-path stub leak (that path is currently excluded), and one tree site that breaks the atlas's "any launch must" rule. Neither blocks.

1. **Strengths**
   - `terminal_native_test.go:90-98,132`: the fix mirrors production's `--config-dir` pin (`launcher/osruntime.go`) instead of inventing a workaround, and the comment records why.
   - `live_zellij.go:153-182`: the owner stand-in uses `/bin/sh` symlinked as `pair`, reading a `wrap` script, with no `exec`. This keeps the observed argv exactly `pair wrap --scrollback-log <raw>`. It is clever but documented, and setup is now atomic (`errors.Join` plus `RemoveAll` on failure).
   - `standInPaneScript` (`quiesce_conformance_live_test.go:103-119`) is one shared helper used by both continuation and recovery. Its single-quote escaping is correct (`'\''`). (ARCH-DRY pass.)
   - Scope narrowing is honest and enforced in one place: the `-run`/`-skip` in `Makefile.local`. Comments point to pair#432 and say how to restore.
   - The workflow adds a failure-only zellij log dump, which is good for future diagnosability.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **BR-3 cold leak (still open):** at `recovery_conformance_live_test.go:235`, `fixture` is reassigned, but the Cleanup at line 88 captures the variable, so the first fixture's stub dir is never removed. Fix by registering a `t.Cleanup` that removes the first fixture's stub, or by having the helper take a `testing.TB`.
   - **New, family `zellij-fixture-config-dir-pin`:** `cmd/probes/couchnestedrows/main.go:126-129` launches with `--new-session-with-layout` but no `--config-dir`, which contradicts the atlas's "Any launch that passes that flag must pin `--config-dir`."

5. **Test coverage**
   - These are live-fixture changes; the CI run on the branch is the evidence.
   - `standInPaneScript` has no direct unit test. Its quoting is exercised by the recovery path, which passes JSON in env, but that path's `checkpoint` subtest is excluded. The `warm` subtest takes the same `AfterAcknowledge`-less path, so in CI only the continuation case would exercise the script, and that case is also excluded.
   - So in this window the script launcher has no running exerciser in CI. That is acceptable because pair#432 restores those cases, but worth knowing.

6. **Architecture**
   - **ARCH-DRY:** pass. The shared helpers are `AddOwnerWrapperPane` and `standInPaneScript`.
   - **ARCH-PURE:** pass. This is test-fixture IO glue with no business logic.
   - **ARCH-PURPOSE:** pass. The Done-when's native-terminal goal is delivered. The deferred cases are separable product semantics, formally moved to #432 with an owning Done-when, not a hidden easy-subset deferral.

7. **Plan revisions:** none. The `## Revisions` entry already matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Comment (quiesce_conformance_live_test.go:104-106) and atlas/couch.md now say it refuses shell operators and $, and reads JSON quotes as shell quoting; matches splitObservedCommand (session_owner.go:166).
  - id: BR-2
    disposition: addressed
    note: |
      Atlas narrowed to launches passing --new-session-with-layout; StartControlledZellijWithOptions does not pass it (live_zellij.go:48-52).
  - id: BR-3
    disposition: not-addressed
    note: |
      First half fixed (stubDir assigned after setup, RemoveAll on failure). Cold path still reassigns fixture at recovery_conformance_live_test.go:235 while the line-88 Cleanup closes only the latest, so the first stub dir leaks; path is excluded under pair#432, so Minor.
findings:
  - id: new
    severity: Minor
    family: zellij-fixture-config-dir-pin
    title: |
      cmd/probes/couchnestedrows launches --new-session-with-layout without --config-dir, contradicting the atlas "any launch must pin" rule
    detail: |
      This is the 2nd finding in family zellij-fixture-config-dir-pin. Rule: every zellij launch that passes --new-session-with-layout pins --config-dir (alongside --config, as terminal_native_test does, which keeps show_startup_tips false). Enumerated sites: launcher/osruntime.go:147 (pinned), couchtty/terminal_native_test.go:132 (pinned), probes/couchnestedrows/main.go:126-129 (unpinned; its comment rejects --config-dir alone for the startup-tips reason, which passing both avoids). Either pin it there or scope the atlas claim to CI and production launches.
```
