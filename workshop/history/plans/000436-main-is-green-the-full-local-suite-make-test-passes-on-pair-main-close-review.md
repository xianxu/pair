# Boundary Review — pair#436 (whole-issue close)

| field | value |
|-------|-------|
| issue | 436 — Main is green: the full local suite (make test) passes on pair main |
| repo | pair |
| issue file | workshop/issues/000436-main-is-green-the-full-local-suite-make-test-passes-on-pair-main.md |
| boundary | whole-issue close |
| milestone | — |
| window | a4ab80e3a81eb749452f6f8aff5abaf25061c9c0..afbe2866d0b5246e7ad9097427eff5b6375cc5ce |
| command | sdlc close --issue 436 |
| reviewer | claude |
| timestamp | 2026-10-11T02:22:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Confirmed one regression in the Makefile scrub. Now checking the test lanes it affects and the lease test.

I found one Important issue: the Makefile change that scrubs the session environment also silently drops knobs the operator sets on purpose. The rest of the window holds up, so the verdict is FIX-THEN-SHIP.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This branch fixes `test-changelog` at the root: owner paths are now compared in the resolved (physical) path form that `SelectedOwner` uses, through one shared helper. It also finds and fixes a real production race in the host-creation lease, which a forked child could hold briefly, and makes couchcore parallel, cutting it from a serial 1,315s to about 205s. The 100k-entry inventory budget became an injected cap, and the 59 existing artifact violations are held in a shrink-only list owned by pair#348. The Done-when clauses are backed by numbers in the Log: two green runs at 434s and 438s, couchcore under 5 minutes, and one quarantine with an owner. The one thing to fix before shipping is the Makefile scrub. When every make goal is a test, it unexports every PAIR_*/COUCH_*/ZELLIJ* variable that came from the environment. That includes knobs the operator sets on purpose, such as `PAIR_PROBE_SETTLE=0s make test-zellij-repaint`, which `Makefile.local:146` documents. I reproduced this with the exact scrub on GNU Make 3.81: the recipe sees an empty value. The tuning or opt-in setting is silently ignored and the probe still reports success.

1. **Strengths**
   - `storagegc.OwnerFramePath` (`cmd/internal/storagegc/env.go:37`) replaces two private EvalSymlinks copies (scrollbackcmd, pairlog) and covers the two sites that were missing it (changelogcmd, `AcquireRoleProcessTarget`). This is a clean ARCH-DRY consolidation, and `TestRoleTargetThroughSymlinkedDataDirIsRecordedPhysically` is a test that fails without the fix.
   - The lease settle loop (`provision_lock_unix.go:60`) is bounded and leaves real errors alone; only a busy result is retried. Taking the settle window as a parameter makes `TestProvisionLeaseOutlastsATransientForkCopy` independent of scheduler timing. The existing busy tests (`provision_lock_test.go:39,110`) still show that a real peer gets `ErrHostCreationBusy`.
   - The artifactpath list fails both ways: on a NEW violation and on a STALE line. It strips line numbers so unrelated edits don't churn it, and it has its own normalization test.
   - Wall-clock budgets moved to an explicit `test-perf-budgets` lane, with the scenarios still running in the default lane. This is documented in `atlas/couch.md`.

2. **Critical findings:** none.

3. **Important findings**
   - **`Makefile.local:10-15`: the scrub drops operator knobs along with session identity.** Every instance in this tree:
     - `PAIR_PROBE_SETTLE` (`cmd/probes/zellijrepaint/main.go:118`, documented at `Makefile.local:146` and in atlas).
     - `PAIR_PROBE_N` and `PAIR_PROBE_SAMPLE_SECS` (`reattachcost/main.go:105,116`, documented at `Makefile.local:155-156` and in `atlas/couch.md:2309`).
     - `PAIR_PROBE_SAMPLE_TSV` (`reattachcost/main.go:124`).
     - `PAIR_PERF_BUDGETS=1 make test` and `PAIR_ARTIFACT_VIOLATIONS_UPDATE=1 make test`.
     - Any `PAIR_LIVE_*=1 make test-…` opt-in.

     **Fix sketch:** narrow the scrub to session identity rather than the whole namespace, for example `$(filter-out PAIR_PROBE_% PAIR_PERF_% PAIR_LIVE_% PAIR_ARTIFACT_%,…)`, or better, an explicit list of the names Couch and pair export. Add a small make-level check that `PAIR_PROBE_N=1 make test-<print-env target>` keeps the value.

4. **Minor findings**
   - The `OwnerFramePath` comment says the missing rest "cannot be a symlink". A dangling symlink also returns ENOENT from EvalSymlinks, so its name is kept as written. The impact is low; reword the comment or handle that case.
   - The artifact list dedupes after stripping line numbers. A new violation with the same file and message as a known one passes unnoticed. This is acceptable for a list that only shrinks, but worth a header note.
   - `scrollbackcmd.canonicalSource` and `pairlog.managedLogWrite` used to error when the parent directory was missing. They now resolve it, so the later owner-path comparison is the only guard left. That looks intended, but the Log doesn't mention it.

5. **Test coverage notes:** the lease, owner-path, inventory-budget and ratchet changes each have a regression test. The Makefile scrub has only a manual check (`make test-pair-embedded-runtime` with and without the scrub) and nothing automated. That is why the over-scrub above got through.

6. **Architectural notes**
   - ARCH-DRY: pass. The owner-frame resolution is consolidated, and `DefaultMaxEntries` names the constant.
   - ARCH-PURE: pass. `OwnerFramePath` necessarily touches the filesystem. The lease's settle window is injected for tests.
   - ARCH-PURPOSE: flag. This is the Makefile finding: making the suite green from a live slot should not cost the existing opt-in lanes their knobs.

7. **Plan revision recommendations:** after fixing the scrub, add a `## Revisions` entry recording that operator knobs (PAIR_PROBE_*, PAIR_PERF_*, PAIR_LIVE_*) are kept.

```findings
findings:
  - id: new
    severity: Important
    family: test-env-scrub-drops-operator-knobs
    title: |
      Makefile.local test-goal unexport drops operator-set PAIR_* knobs (PAIR_PROBE_SETTLE, PAIR_PROBE_N, PAIR_PERF_BUDGETS, PAIR_LIVE_*)
    detail: |
      The scrub unexports every environment-origin PAIR_/COUCH_/ZELLIJ name whenever all goals are test-*. Reproduced on GNU Make 3.81: PAIR_PROBE_SETTLE=0s make test-x leaves the recipe with an empty value. Affected: the documented PAIR_PROBE_SETTLE, PAIR_PROBE_N and PAIR_PROBE_SAMPLE_SECS (Makefile.local:146,155-156; atlas/couch.md:2309), plus PAIR_PROBE_SAMPLE_TSV, PAIR_PERF_BUDGETS, PAIR_ARTIFACT_VIOLATIONS_UPDATE and PAIR_LIVE_* opt-ins. Narrow the scrub to session-identity names (or filter-out the knob prefixes) and add a make-level regression check.
  - id: new
    severity: Minor
    family: doc-claim-from-memory
    title: |
      OwnerFramePath comment says the missing suffix cannot be a symlink; a dangling symlink also yields ENOENT
    detail: |
      EvalSymlinks on a dangling link returns ErrNotExist, so the loop keeps the link name as written instead of resolving it.
  - id: new
    severity: Minor
    family: ratchet-dedup-masks-new-instance
    title: |
      Artifact violation list strips line numbers and dedupes, so a second identical violation in a known file passes
```

---

## Re-review — 2026-10-11T02:33:56-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 436 — Main is green: the full local suite (make test) passes on pair main |
| repo | pair |
| issue file | workshop/issues/000436-main-is-green-the-full-local-suite-make-test-passes-on-pair-main.md |
| boundary | whole-issue close |
| milestone | — |
| window | a4ab80e3a81eb749452f6f8aff5abaf25061c9c0..ac5fd95d446a3434b372900678d459ae19af2059 |
| command | sdlc close --issue 436 |
| reviewer | claude |
| timestamp | 2026-10-11T02:33:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Prior findings look fixed so far; now running the Go tests and checking whether the ratchet fix has a regression test.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

All three prior findings have fixes. BR-1 and BR-2 each have a regression test, and I confirmed both tests fail without their fix. BR-3's counting change looks correct, but nothing tests it, so it stays open (Minor). `tests/make-test-env-test.sh` passes and goes red when `TEST_ENV_KNOBS` is emptied. `go test ./cmd/internal/artifactpath ./cmd/internal/storagegc` is green.

The BR-1 family is still open, though. The scrub drops everything under the prefixes unless a name is on the allowlist. The allowlist is checked only against knobs documented in README, atlas, Makefile.local and doctor. At least six test opt-in knobs fall outside it. When any of them is passed through a `make test*` goal, it is silently dropped and the test skips. This is the 2nd finding in that family, so the fix should be a rule, not one more prefix.

**Strengths**
- `Makefile.local:12-13`: the scrub only removes environment-origin names, and only when every goal is a test. Workflow targets keep the slot identity.
- `tests/make-test-env-test.sh` builds its knob list from the docs instead of keeping a hand-written list, which is the right shape. Its gap is coverage, not the approach.
- `storagegc/env.go:60-64`: an `Lstat` after `ErrNotExist` cleanly tells a dangling symlink apart from a missing component. The new case in `lease_test.go:238-244` fails without the fix: before the fix, the link name was kept and the call returned success.
- The atlas entry for the perf-budget lane and the parallel tests is current (`atlas/couch.md:2485-2490`).

**Critical**
None.

**Important**
- `Makefile.local:12` and `tests/make-test-env-test.sh:7`: **This is the 2nd finding in family `test-env-scrub-drops-operator-knobs`.** These knobs are read by tests and set by the operator, but are neither on the allowlist nor caught by the docs grep:
  - `PAIR_UPDATE_GROUPED_FIXTURES` (`couchtty/grouped_fixture_test.go:55`)
  - `PAIR_PARK_LATENCY_SMOKE`
  - `PAIR_TERMINAL_SOAK_DURATION`
  - `PAIR_TERMINAL_NATIVE`
  - `PAIR_TERMINAL_RESOURCE_PROBE`
  - `PAIR_MENU_PERF_TARGET`
  
  `PAIR_TERMINAL_RESOURCE_PROBE` is even documented in exactly the `X=1 go test` form, at `terminal/resource_test.go:28`. The grep misses it because it doesn't scan `cmd/`.
  - **Rule:** a name may be scrubbed only if a session exports it, and a test may read only a name the scrub keeps.
  - **Fix:** take the real consumers as the list of names to check: every `os.Getenv("PAIR_…"|"COUCH_…")` in `*_test.go`. Require each one to be either in `TEST_ENV_KNOBS` or set by the test itself (`Setenv`/`cmd.Env`). Or flip the scrub to a denylist built from what the session launcher exports.
  - **Measured prevalence:** about 6 operator knobs outside the allowlist, against 10 found by the docs grep.

**Minor**
- BR-3: the count-based ratchet in `coverage_test.go:240-286` sits inline, takes `*testing.T`, and has no regression test. Moving the fresh/stale calculation into a pure function with a test for two identical violations would close it. The known file currently has no duplicate lines, so this has no effect today.
- The test-goal env scrub is a new developer-facing rule, but it is documented only in `Makefile.local`. One line in atlas would make it findable.

**Test coverage**
- The BR-1 and BR-2 regressions fail without their fixes; I checked BR-1 by emptying `TEST_ENV_KNOBS` and BR-2 by reasoning through the code path. BR-3 has no test.

**Architecture**
- **ARCH-DRY:** pass.
- **ARCH-PURE:** pass. `OwnerFramePath` is a small helper that touches the filesystem, which is acceptable; the ratchet diff could be pure (see Minor).
- **ARCH-PURPOSE:** flag. The env-scrub fix covers the documented instances, not the whole class (see Important).
- **ARCH-MOCK:** N/A. No new external dependency.
- **ARCH-CONSTRAINTS:** pass. Wall-clock budgets moved to their own lane.
- **ARCH-SECURE:** pass. The scrub keeps tests from writing into the operator's session state, and a dangling symlink is now refused instead of trusted.
- **ARCH-ORDER:** N/A. The changes hold no state between events.
- **ARCH-FUNERAL:** pass. The ratchet file can only shrink, and its owner is pair#348.

**Plan revisions**
None.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TEST_ENV_KNOBS filter-out plus tests/make-test-env-test.sh; confirmed red with TEST_ENV_KNOBS emptied (PAIR_PROBE_SETTLE dropped), green as committed.
  - id: BR-2
    disposition: addressed
    note: |
      env.go Lstat refusal; lease_test.go dangling case fails pre-fix because the old loop kept the link name and returned success.
  - id: BR-3
    disposition: not-addressed
    note: |
      Counted comparison looks correct, but no test exercises a duplicate violation; extract fresh/stale into a pure func and test it.
findings:
  - id: new
    severity: Important
    family: test-env-scrub-drops-operator-knobs
    title: |
      Scrub still drops test opt-in knobs outside TEST_ENV_KNOBS; docs-grep check cannot see them
    detail: |
      2nd in family. PAIR_UPDATE_GROUPED_FIXTURES, PAIR_PARK_LATENCY_SMOKE, PAIR_TERMINAL_SOAK_DURATION, PAIR_TERMINAL_NATIVE, PAIR_TERMINAL_RESOURCE_PROBE (documented as X=1 go test at terminal/resource_test.go:28) and PAIR_MENU_PERF_TARGET are read by tests, are not session identity, and get unexported under make test goals, so the test skips silently. Rule: scrub only names a session exports; a test may read only names the scrub keeps. Enforce it from the real consumers: scan os.Getenv of PAIR_/COUCH_ names in *_test.go and require each to be in TEST_ENV_KNOBS or set by the test itself. Or derive the scrub from the launcher's exported-name list instead of prefixes. Prevalence: about 6 operator knobs missed vs 10 found by the docs grep.
```

---

## Re-review — 2026-10-11T02:53:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 436 — Main is green: the full local suite (make test) passes on pair main |
| repo | pair |
| issue file | workshop/issues/000436-main-is-green-the-full-local-suite-make-test-passes-on-pair-main.md |
| boundary | whole-issue close |
| milestone | — |
| window | a4ab80e3a81eb749452f6f8aff5abaf25061c9c0..85403b9e88fbafbe6dd10b17f0902b85e90b320e |
| command | sdlc close --issue 436 |
| reviewer | claude |
| timestamp | 2026-10-11T02:53:31-07:00 |
| verdict | SHIP |

## Review

Focus on the last two fix commits for the open findings.

```verdict
verdict: SHIP
confidence: medium
```

Both open findings are fixed, and I re-checked each one directly. For BR-4, the scrub is now built from the names production code mentions instead of a prefix keep-list. A name that only tests read can no longer be dropped. I ran `make test-make-env-production-names` and none of the six knobs BR-4 named (`PAIR_UPDATE_GROUPED_FIXTURES`, `PAIR_PARK_LATENCY_SMOKE`, `PAIR_TERMINAL_SOAK_DURATION`, `PAIR_TERMINAL_NATIVE`, `PAIR_TERMINAL_RESOURCE_PROBE`, `PAIR_MENU_PERF_TARGET`) is in the production set. I also ran `bash tests/make-test-env-test.sh` against the pinned head: `PASS make-test-env (165 derived knobs kept, session identity dropped)`. For BR-3, `diffKnownViolations` is now a pure function, and `TestDiffKnownViolationsCountsDuplicates` checks it with duplicate counts. The guard reaches the real check through `checkKnownArtifactViolations`. Nothing blocks SHIP.

1. **Strengths**
   - The `Makefile.local` scrub states the rule from BR-4 directly: drop what production names, keep what only tests read. It only drops names that came from the environment, and only when every make goal is a test, so workflow targets still see the slot identity.
   - `tests/make-test-env-test.sh` builds its knob list from the code and the docs instead of keeping one by hand, so a new knob is checked without anyone editing a list. It also fails if a scan comes back empty.
   - `storagegc.OwnerFramePath` (`cmd/internal/storagegc/env.go:38`) is now the one helper for this path handling. It replaced three hand-written `EvalSymlinks(Dir)` copies in pairlog, scrollbackcmd and changelogcmd (ARCH-DRY). It refuses a dangling symlink.
   - The ratchet's counted multiset is a pure function with a direct test.
   - `acquireHostCreationLease` takes its settle window as a parameter, so a test can control the timing.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The scrub's production scan only sees names written literally. A session name built at runtime (for example `"PAIR_" + x`) would still get through. The session half of `make-test-env-test.sh` is a hand-picked list of 11 names, so it only samples the drop side. This is acceptable for now.
   - `COUCH_ISOLATED_ROOT` is documented as a test/diagnostic knob but is also a name production uses, so `make test` scrubs it. That is probably what you want (tests set it themselves), but no comment says so.
   - `TestMessageWrapperExitDisconnectsAndForgetsWorkspace` waits with a fixed 3-second polling loop. That's fine, but it cannot inject an ordering.

5. **Test coverage:** the scrub (both what it keeps and what it drops), the ratchet's duplicate counting, symlink resolution of the owner path (an alias is accepted, an escape is refused, a dangling link is refused), and the lease's settle window are all covered.

6. **Architecture**
   - **ARCH-DRY:** pass. The owner-path logic is now in one helper.
   - **ARCH-PURE:** pass. `diffKnownViolations` and `normalizeArtifactViolation` are pure.
   - **ARCH-PURPOSE:** pass. BR-4 was fixed as a rule, not one knob at a time.
   - **ARCH-MOCK:** no new external seam.
   - **ARCH-CONSTRAINTS:** pass. The 100ms settle window is bounded and based on a 2–9ms measurement. Perf budgets run in their own lane, `test-perf-budgets`.
   - **ARCH-SECURE:** pass. Tests no longer inherit the live session's paths, which shrinks how much real state a test can touch. Symlink escapes and dangling links are refused.
   - **ARCH-ORDER:** pass. The lease retry and the message-service wait are both bounded.
   - **ARCH-FUNERAL:** pass. The known-violations list may only shrink, and stale entries fail the test.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      diffKnownViolations is pure and counts duplicates; TestDiffKnownViolationsCountsDuplicates pins a second identical violation as fresh, and checkKnownArtifactViolations uses it.
  - id: BR-4
    disposition: addressed
    note: |
      The scrub drops only the names production mentions, so test-only names pass through; all six named knobs are absent from the production scan, and make-test-env-test.sh passes with 165 derived knobs.
findings:
  - id: new
    severity: Minor
    family: test-env-scrub-drops-operator-knobs
    title: |
      The production-name scan sees only literal names, and the session-drop check samples 11 hand-listed names
    detail: |
      A session variable whose name is built at runtime would get past the scrub. No such variable exists today, so this is only a note.
```
