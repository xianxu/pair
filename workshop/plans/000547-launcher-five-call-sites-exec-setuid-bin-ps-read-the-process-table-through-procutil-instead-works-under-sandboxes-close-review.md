# Boundary Review — pair#547 (whole-issue close)

| field | value |
|-------|-------|
| issue | 547 — launcher: five call sites exec setuid /bin/ps; read the process table through procutil instead (works under sandboxes) |
| repo | pair |
| issue file | workshop/issues/000547-launcher-five-call-sites-exec-setuid-bin-ps-read-the-process-table-through-procutil-instead-works-under-sandboxes.md |
| boundary | whole-issue close |
| milestone | — |
| window | b63f4bd76e3e81d1433cfeda544bb3bed8501522..458b66fded9c0015087e4a3370cfe24bf1f757bc |
| command | sdlc close --issue 547 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T01:00:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The native inspection migration is well structured, and the focused production tests passed. One reproducible test-helper lifecycle defect needs correction. Required seatbelt validation could not run here: `sandbox-exec` failed with `sandbox_apply: Operation not permitted`. The broader launcher suite also encountered missing generated runtime assets, so full acceptance remains unverified.

1. **Strengths**

   - `procutil/command.go:44` preserves per-row errors, fences PID reuse, and checks cancellation between reads.
   - Native executable identity stays separate from process-controlled argv; diagnostic inventory also handles Linux’s unlinked-executable suffix.
   - Launcher projections reuse existing server matching and retain snapshot identities. The live argv-spoofed helper test passed.
   - The production no-ps guard passed and detects direct, contextual, absolute-path, and wrapped calls.
   - Atlas and artifact ownership updates cover the new surface. No new user-facing command or flag requires a README change.

2. **Critical findings**

   None found.

3. **Important findings**

   **[tests/without-ancestors.py:36](/Users/xianxu/Library/Caches/sdlc/review-checkouts/0dc75832cb49122f/tests/without-ancestors.py:36) — timeout leaves descendants running (ARCH-ORDER, ARCH-FUNERAL).**

   `subprocess.run(..., timeout=timeout)` kills and waits for its immediate child. The worker then reports failure, and the supervisor cancels its process-group cleanup alarm at line 63. Descendants survive.

   Reproduced with a controlled command that spawned `/bin/sleep 10`: the helper returned exit 1 after approximately 0.37 seconds with a 0.2-second timeout, while the grandchild remained alive. The review harness removed that private group afterward.

   Make failure cleanup terminate the owned process group before returning. Define cleanup when the supervising process is cancelled, too. Add regression tests proving descendants terminate after timeout and supervisor cancellation.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed full suites: procutil, diagnosticlog, gccmd, gcruntime.
   - Passed focused launcher tests: ancestry, server projection, cancellation, native argv discovery, and no-ps guard.
   - Broader launcher tests failed while building fixtures because `assets/runtime/files` was absent.
   - Seatbelt execution failed before tests started; no sandbox acceptance result is inferred from tracker claims.
   - Native Linux execution remains unverified, consistent with the explicit plan revision.
   - The ancestry helper lacks lifecycle regression coverage. Repository files remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared decoders and server projection replace duplicated inspection.
   - **ARCH-PURE — pass:** parsing and projections are pure; native IO uses adapters. The decoder-location revision accounts for the provisional table path.
   - **ARCH-PURPOSE — pass:** the sweep covers all identified production Go ps consumers.
   - **ARCH-MOCK — pass:** stateful snapshot and inventory fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** serial enrichment, bounded parsing, and cancellation checks match the declared envelope.
   - **ARCH-SECURE — pass:** malformed buffers and unavailable candidate evidence fail conservatively.
   - **ARCH-ORDER — flag:** the new helper drops descendant supervision when its immediate command times out.
   - **ARCH-FUNERAL — flag:** the helper’s claimed bounded lifetime does not cover its descendant group.

7. **Plan revision recommendation**

   Add a `## Revisions` entry defining ownership and cleanup of the reparented process group on timeout and supervisor cancellation, with regression tests for both paths.

```findings
findings:
  - id: new
    severity: Important
    family: owned-process-lifetime
    title: |
      Reparented test helper leaves descendants running after timeout
    detail: |
      tests/without-ancestors.py:36 kills only the immediate subprocess on timeout; the worker reports failure and the supervisor cancels its group-cleanup alarm at line 63. A controlled reproduction returned after approximately 0.37 seconds while its sleep grandchild remained alive. ARCH-ORDER and ARCH-FUNERAL require cleanup of the owned process group on timeout and supervisor cancellation, with regression tests proving descendants terminate.
```

---

## Re-review — 2026-10-14T01:10:40-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 547 — launcher: five call sites exec setuid /bin/ps; read the process table through procutil instead (works under sandboxes) |
| repo | pair |
| issue file | workshop/issues/000547-launcher-five-call-sites-exec-setuid-bin-ps-read-the-process-table-through-procutil-instead-works-under-sandboxes.md |
| boundary | whole-issue close |
| milestone | — |
| window | b63f4bd76e3e81d1433cfeda544bb3bed8501522..7c766e0239e34bef57a8ee6c96062bc1ecc5f99c |
| command | sdlc close --issue 547 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T01:10:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

BR-1 is addressed, and this inspection found no new code defects. Native process probes pass in the current sandbox, where `/bin/ps` is denied. The blocking limitation is verification: the prescribed seatbelt invocation failed with `sandbox_apply: Operation not permitted`, and two launcher acceptance tests could not build because generated runtime assets are absent. Under the requested unavailable-command rule, this review cannot authorize SHIP; rerun those checks in a prepared environment.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      tests/without-ancestors.py:30-106 establishes group ownership and a startup handshake, then kills the group on failure or cancellation. All five lifecycle tests pass at head. Against the pre-fix helper, the timeout regression fails with a surviving grandchild; cancellation regressions also fail. The disposable regression fixture was removed.
```

1. **Strengths**

   - All five launcher sites use procutil; the production no-ps guard passes, including its synthetic violation tests.
   - Snapshot enrichment retains unavailable evidence and fences PID reuse. Stateful tests exercise disappearance, recycling, and cancellation.
   - Real Darwin tests confirm zombie metadata remains in `Table` while `Snapshot` excludes terminated rows.
   - Diagnostic cleanup shares the injected inspection boundary across both proof paths; acceptance coverage verifies a legacy writer blocks both.
   - Atlas documents the new process-inspection surface. No new user command, flag, or configuration key requires a README change.

2. **Critical findings**

   None.

3. **Important findings**

   None newly identified. The verification limitations above prevent completion of this review.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: procutil, diagnosticlog, gccmd, gcruntime, artifact inventory, focused launcher process tests, five helper lifecycle tests, Linux cross-compilation, and diff whitespace checks.

   Launcher package failures were `TestCreateLayoutWrapperPreservesAgentCommand` and `TestSelectedRootNativeCreateListResume`: both encountered missing `runtimebundle/assets/runtime/files`. `TestSessionQuiescenceLive` skipped without `PAIR_LIVE_COUCH=1`. Both explicit seatbelt runs failed before executing tests. Native Linux execution remains unverified, as documented in the plan revision.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared native parsers and consumer projections replace duplicated subprocess inspection.
   - **ARCH-PURE — pass:** decoding, ancestry, and server projection are directly tested without IO.
   - **ARCH-PURPOSE — pass:** the production sweep covers the additional procutil and diagnostic consumers.
   - **ARCH-MOCK — pass:** stateful process and writer fixtures use production observation seams; native conformance coverage exists.
   - **ARCH-CONSTRAINTS — pass:** enrichment is serial, bounded, and checks cancellation between reads.
   - **ARCH-SECURE — pass:** PID narrowing, malformed buffers, executable/argv distinction, and generation fencing are covered.
   - **ARCH-ORDER — pass:** controlled tests exercise changing process generations and helper startup cancellation.
   - **ARCH-FUNERAL — pass:** failed helper invocations clean their owned group; successful sidecars retain explicit caller-owned cleanup.

7. **Plan revision recommendations**

   None. Existing revisions reconcile implementation locations, terminal-process handling, helper lifetime, and the accepted Linux verification limitation.
