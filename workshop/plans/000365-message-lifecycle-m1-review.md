# Boundary Review — pair#365 (milestone M1)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f42189564ba340eb017d4292dc74a73c8f37e7f5..a7e379c97d138ad876584869a3467c8460d67af8 |
| command | sdlc milestone-close --issue 365 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-01T14:47:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers both things it promised: an in-tree acceptance test that counts probes during an idle minute, and a live baseline that can be reproduced. I ran `TestMessageIdleRunsNoProbes` outside the sandbox, because `/tmp` socket creation is blocked inside it. It skips with "12 launch checks (2 ownership probes each), 12 process checks, 6 git status", which matches the measurements file and the issue Log. The checks that don't need a human all pass: the probe's `go test` and `go vet`, and the skip/baseline output. The live CPU and child-process numbers come from the operator's running Couch, so this review cannot re-check them. Two cheap problems keep this from SHIP:
- **Untracked build output:** `run.sh arm` builds two multi-MB binaries into `probes/messageidle/bin/`, and that path is not gitignored.
- **Unrecorded plan drift:** the test and the probe both differ from the plan, and the plan's `## Revisions` doesn't record it.

### 1. Strengths
- **Counts measured at the real seam.** `cmd/internal/couchcmd/message_idle_test.go:28-48` wraps the production `messageAuthority` fields (`launch`/`process`/`branch`) instead of mocking a pure entity. The skip only fires while the counts are non-zero, so it gives the baseline now and becomes the assertion once M2 lands.
- **The shim records only Couch's own spawns** (`probes/messageidle/main.go:41`). Agents in the slots inherit `PATH` and run `ps` themselves; filtering on the parent process keeps those out. The Darwin side reads the parent name through `sysctl`, so the shim never calls `ps` and can't recurse (`parent_darwin.go`).
- **`make test-smoke` can't start a real session.** The shim does nothing unless it was invoked as `zellij` or `ps` (`main.go:30-35`), which keeps the protection lesson from `zellijcalls`.
- **Honest measurement.** The measurements file states its method's limits ("distinct PIDs; undercounts short-lived ones"), records the binary's `vcs.revision`, and archives commands that can be re-run without relaunching Couch.
- **Small, reusable refactor.** `messageSocketForTest` is a clean extraction that `serviceFixture` and the new test both use (ARCH-DRY pass).

### 2. Critical findings
None.

### 3. Important findings
- **`probes/messageidle/bin/` is not gitignored.** `git check-ignore --no-index probes/messageidle/bin/zellij` prints nothing; `probes/zellijcalls` has its own `.gitignore` containing `bin/`. After `arm`, a `git add -A` would sweep two Go binaries into the ariadne base layer. The root `.gitignore:125-130` describes that exact incident (pair#170). **Fix:** add `probes/messageidle/.gitignore` containing `bin/`, as `zellijcalls` does. Also consider a test like `TestEveryMainPackageIsIgnoredAtTheRepoRoot` for probe `bin/` directories, so the next probe can't repeat this.

### 4. Minor findings
- **Copy-paste instead of reuse (ARCH-DRY).** `realBinary` (`probes/messageidle/main.go:69-88`) is `realZellij` (`probes/zellijcalls/main.go:98-116`) with the name made a parameter. Task 1.2 said to extend the `zellijcalls` shim. Creating a new probe instead is defensible, but `## Revisions` doesn't record it.
- **The test is not what Task 1.1 specified.** Task 1.1 asked for a multi-slot test with three wrappers and no traffic. The delivered test has one wrapper and sends a heartbeat every second. Done-when 1 says "multi-slot". M2 rewrites the test anyway, but the plan should say so.
- **Test name doesn't match its body.** `TestParentNameReadsThisTestsParent` (`main_test.go:8`) reads the test's own pid, not its parent's.
- **Trace file grows without bound while armed (ARCH-FUNERAL).** `$TMPDIR/pair-messageidle.tsv` is truncated only on `arm`. Acceptable for an attended probe, but `SKILL.md` should say so.
- **Cosmetic report keys.** `run.sh`'s summary drops dash-prefixed words, so `ps -axo pid=,...` is reported under the key `ps pid=,ppid=,comm=`. It's readable, but the M3 "after" comparison should use the same version of the script.

### 5. Test coverage notes
- The idle test reaches every production path that spawns a process (`launch`, `process`, `branch`), so it would catch a probe that M2 failed to remove.
- It cannot catch a failing registration being re-admitted, which the live data points to (141 `sdlc workspace` calls in 120 s). M2/M3 should add a case where registration fails and assert that `ResolveWorkspace` and the full check stay bounded.
- The probe has no test for `realBinary` excluding its own directory. It is low risk because the code is copied from `zellijcalls`.

### 6. Architectural notes for upcoming work
- **ARCH-DRY:** Minor, the copied `realBinary` above.
- **ARCH-PURE:** pass. The counting decorator is pure wrapping; the IO stays in the probe.
- **ARCH-PURPOSE:** pass for M1 (a baseline is the milestone's purpose). Carry the failing-registration finding into M2's scope, since it is the most expensive case.
- **ARCH-MOCK:** pass. The test uses the existing stateful `messageAuthorityFake`, and the live run is the conformance check.
- **ARCH-CONSTRAINTS:** pass. CPU-seconds and per-minute spawn counts are measured, not asserted.
- **ARCH-SECURE:** pass. The trace file is created `0600` and the probe touches no credentials.
- **ARCH-ORDER:** N/A for M1. No new state is carried between events; the reducer arrives in M2.
- **ARCH-FUNERAL:** pass with the trace-file Minor above. Ignoring `bin/` is also a residue concern.

### 7. Plan revision recommendations
Add to `## Revisions`:
- "M1: Task 1.1 shipped as a single-wrapper idle test that sends heartbeats (the current protocol's idle traffic), not the three-wrapper, no-traffic version. M2's rewrite delivers the multi-slot form required by Done-when 1."
- "M1: Task 1.2 created a separate `probes/messageidle` instead of extending `zellijcalls`, because it needs a `ps` shim and parent filtering; `realBinary` duplicates `zellijcalls.realZellij`."

```findings
findings:
  - id: new
    severity: Important
    family: probe-build-output-ignored
    title: |
      probes/messageidle/bin/ is not gitignored; run.sh arm drops two Go binaries a git add -A would commit
    detail: |
      zellijcalls ships probes/zellijcalls/.gitignore with bin/; messageidle has none (git check-ignore prints nothing). Add probes/messageidle/.gitignore with bin/; consider a test that every probe bin/ dir is ignored.
  - id: new
    severity: Minor
    family: shared-helper-not-reused
    title: |
      realBinary in probes/messageidle/main.go copies zellijcalls realZellij (ARCH-DRY); plan said extend zellijcalls
    detail: |
      Record the deviation in Revisions or extract the PATH-skip-self lookup into a shared probe helper.
  - id: new
    severity: Minor
    family: plan-revision-drift
    title: |
      Idle test is single-wrapper with heartbeats, not the planned three-wrapper no-traffic test; Revisions does not record it
    detail: |
      Done-when 1 says multi-slot; add a Revisions entry noting M2's rewrite covers multi-slot.
  - id: new
    severity: Minor
    family: test-name-matches-assertion
    title: |
      TestParentNameReadsThisTestsParent reads its own pid, not its parent
  - id: new
    severity: Minor
    family: artifact-removal-path
    title: |
      messageidle trace TSV grows unbounded while armed; only arm truncates it
```

---

## Re-review — 2026-10-01T14:48:44-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 365 — Replace messaging liveness polling with lifecycle events |
| repo | pair |
| issue file | workshop/issues/000365-message-lifecycle.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f42189564ba340eb017d4292dc74a73c8f37e7f5..95ec01c00329773a04a9d12be4ebfe11327adf26 |
| command | sdlc milestone-close --issue 365 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-01T14:48:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M1 delivers what the plan asks for, with two deviations that the plan's `## Revisions` now records. It adds a probe-count acceptance test, skipped until M2, whose skip message reports the measured baseline: 12 launch checks, 12 process checks and 6 `git status` per idle minute for one wrapper. It also adds the `probes/messageidle` shim and an archived live baseline in `workshop/plans/000365-message-lifecycle-measurements.md`, with commands that can be run again. Atlas and the project milestone note are updated. Of the six prior findings, BR-2, BR-3, BR-4 and BR-5 are addressed. BR-2 is fixed for the whole class, not just this probe: one root rule `/probes/*/bin/` covers every probe, and `git check-ignore` confirms it matches `probes/messageidle/bin/zellij`. BR-1 and BR-6 are Minor and still open, so neither blocks the boundary. The `couchcmd` service tests fail inside the sandbox with `mkdir /tmp/...: operation not permitted`. That is the sandbox denying the write, not a regression: unsandboxed, the cached run passes and the idle test SKIPs with the baseline counts. `go test ./probes/messageidle` passes, and `GOOS=linux go vet` is clean.

**Strengths**
- `.gitignore:163` closes the BR-2 gap with one root rule for every probe, so a new probe can't forget its own `.gitignore`.
- `probes/messageidle/parent_darwin.go` reads the parent's name with sysctl instead of running `ps`. Running `ps` there would call the shim again.
- The shim records only calls whose parent is `couch` (`main.go:41`). Without that filter, agents running `ps` inside the slots would inflate Couch's numbers.
- `messageSocketForTest` (`message_service_test.go:87`) is pulled out of `serviceFixture` and shared, not copied (ARCH-DRY).
- `countingAuthority` wraps the existing fake behind the same authority seam, so production and test go through the same boundary (ARCH-MOCK).

**Critical:** none.

**Important:** none.

**Minor**
- BR-1 is still open. Plan Task 2.1 still lists 13 named test cases rather than one strategy line: drive permuted event sequences through `Advance` and assert invariants. Fix it before or during M2 execution.
- BR-6 is still open. `run.sh` only truncates the trace when arming. `disarm` could delete `$LOG`, because `window` already writes its summary to a separate `$out.txt`.
- `run.sh` `window`: if `--session` is the last argument, `a[i+2:]` is just empty, which is harmless. The Python only fails on a malformed timestamp.

**Test coverage**
- The idle test can't fail until M2: it skips whenever the counts are non-zero. That is the intended "skipped red" state. Task 2.4 must rewrite the test to the multi-slot, no-traffic form, not just delete the skip; the Revisions entry records this.

**Architecture**

| Principle | Result |
|---|---|
| ARCH-DRY | Pass. The `realBinary` / `realZellij` mirror is acknowledged in Revisions, and there is no shared probe library to reuse. |
| ARCH-PURE | Pass |
| ARCH-PURPOSE | Pass for M1's scope |
| ARCH-MOCK | Pass |
| ARCH-CONSTRAINTS | Pass. The baseline is measured, not asserted. |
| ARCH-SECURE | Pass. The trace file is created with mode 0600, and the shim never changes what the caller sees. |
| ARCH-ORDER | N/A for M1: the change adds no stateful component. |
| ARCH-FUNERAL | Flagged only as BR-6. |

**Notes for upcoming work**
- The measurement found 141 `sdlc workspace` children in 120 s, which shows that failing registrations are the most expensive case under polling. M2's Registry tests should include a wrapper whose admission keeps failing, and assert that its back-off ends in dormant.

**Plan revisions:** none needed beyond the BR-1 compression.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan Task 2.1 still enumerates 13 named tests; compress to the invariant/permutation strategy line before M2 execution (Minor, non-blocking).
  - id: BR-2
    disposition: addressed
    note: |
      Root .gitignore:163 adds /probes/*/bin/ for the whole class; git check-ignore -v confirms it matches probes/messageidle/bin/zellij.
  - id: BR-3
    disposition: addressed
    note: |
      Plan Revisions (2026-10-01 M1 boundary review) records the separate shim and the realBinary/realZellij mirror, with the reason.
  - id: BR-4
    disposition: addressed
    note: |
      Plan Revisions records the single-wrapper heartbeat form and assigns the multi-slot no-traffic rewrite to Task 2.4.
  - id: BR-5
    disposition: addressed
    note: |
      Renamed TestParentNameReadsAKnownProcess with a comment; name now matches the own-pid assertion.
  - id: BR-6
    disposition: not-addressed
    note: |
      run.sh unchanged; disarm could rm the trace since window writes its summary separately (Minor).
```
