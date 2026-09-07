---
gate: boundary-review
issue: 208
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-06T23:35:17-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
          detail: |-
            2nd finding in this family, so the rule is the deliverable, not the site: a
            plan may name a repo mechanism only with a file:line AND a check that it is
            reachable from the language and layer the calling code sits in. Prevalence
            2/2 rounds that named a mechanism — PQ-1 failed build-path reachability,
            this fails language reachability. artifactpath is cmd/internal/artifactpath
            with no CLI surface; CaptureRecord lives in nvim/doctor.lua. Use the
            existing idiom, pair_data_dir() at nvim/init.lua:484 plus pair_tag()
            (nvim/init.lua:4012). Sweeping the plan's other named mechanisms under the
            same rule found one more: the binary is named `hoprtt`, but
            Makefile.local:5-7 requires the `pair-` prefix on every GO_BINS entry
            because make install (:84-87) puts each on PATH — the collision that
            comment records already fixing once for `scribe`. Name it `pair-hoprtt`.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: unverified-repo-mechanism
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-06T23:35:17-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-06T23:39:21-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Critical
          title: spawnRTT discards c.Run()'s error, so a failing command is reported as a healthy latency
          detail: 'cmd/pair-hoprtt/main.go:96 uses `_ = c.Run()`. Reproduced: `hoprtt -spawn 10 -- /usr/bin/definitely-not-here` prints `0.599 ... 10` and exits 0, and because `zellij action query-tab-names` exits 1 here ("There is no active session!"), doctor/perf.sh printed `zellij_action_ms=10.183` — a fabricated healthy reading for the exact hypothesis (issue 201/203 round-trip cost) this issue exists to settle. perf.sh:117 guards only `command -v zellij`, not the session-gone case the plan''s own ARCH-ORDER table names. Count failed runs and render `n/a (command failed: <err>)`; re-take M1.4''s zellij number from inside a live session.'
          family: failure-reported-as-measurement
          round: 3
        - id: BR-3
          severity: Critical
          title: delta silently drops a pid alive in both samples but missing one cputime row, contradicting its own contract
          detail: 'nvim/doctor.lua:86-92 — `if ca and cb then` has no else, so a pid present in both procs samples but absent from one cputime sample lands in no bucket. Reproduced: procs A={1,2} B={1,2}, cpu A={1,2} B={1} yields rates=1 vanished=0 started=0 reused=0. Reachable in production because perf.sh:60-65 collects procs and cputime in two separate `ps -A` invocations, which diverge most under a spawn storm — the scenario the counters exist for. Fix both ends: one `ps -Ao pid=,etime=,time=,rss=,comm=` pass in perf.sh, and a counted `no_cputime` bucket in delta with a test asserting every input pid lands in exactly one bucket.'
          family: silent-drop-in-join
          round: 3
        - id: BR-4
          severity: Critical
          title: A 2.9MB Mach-O arm64 binary is committed at the repo root, against this window's own gitignore rule
          detail: c290191c added `pair-hoprtt` (2,896,786 bytes, Mach-O 64-bit arm64) at the tree root; 95f73548 then added `/pair-hoprtt` to .gitignore:53, which has no effect on an already-tracked file. Still tracked at HEAD in a base-layer repo whose files propagate. Prefer rebasing c290191c to drop the blob entirely rather than `git rm --cached` in a follow-up, since the branch is unmerged and the history is still cheap to fix.
          family: build-artifact-committed
          round: 3
        - id: BR-5
          severity: Important
          title: perf.sh collector failures render as values, not n/a, contradicting the file's own stated rule
          detail: 'With ps/top/sysctl unavailable (reproduced in this shell), doctor/perf.sh emitted `host_cores=?`, `load=`, `cpu_idle_pct=`, `process_count=0`, and empty `### cputime` / `### procs` / `## disk` blocks — `process_count=0` being a fabricated claim and the empty values indistinguishable from a real reading. The header (:19-20) promises every probe degrades to `n/a` with a reason. Realistic, not academic: the header advertises standalone re-runs by an agent, and agents here run sandboxed with exactly these tools denied. One `probe()` helper substituting `n/a (<tool> unavailable)`, applied to every collector including both sample blocks.'
          family: failure-reported-as-measurement
          round: 3
        - id: BR-6
          severity: Important
          title: grep -c with `|| echo 0` emits a stray bare 0 line, corrupting the report on any quiet machine
          detail: 'doctor/perf.sh:50-51 — `grep -c` prints 0 AND exits 1, so the `||` fires and appends a second 0. Reproduced: `build_procs=0` followed by a bare `0` line, which breaks the k=v line contract M2''s parser will read. build_procs is zero whenever no build is running, i.e. on the healthy-baseline capture the Done-when requires. Drop the `|| echo 0`, or use `|| true`.'
          family: report-line-contract
          round: 3
        - id: BR-7
          severity: Important
          title: macOS `ps -o comm=` is a path, so `^go$` never matches and the pair pattern over-matches
          detail: 'doctor/perf.sh:50-51 anchors `^go$` against a full path like /opt/homebrew/.../bin/go, silently zeroing issue 203''s build-storm variable (compile$/link$ still match). `pair_family_procs` matches unanchored `pair` against full paths, so every process launched from /Users/.../workspace/pair/ counts. The repo''s own doctor/emitter-health.sh:25-31 documents comm= as a full path. Match on the basename before grepping and anchor. Caveat: `ps` is denied in my shell, so this is reasoned from that path-format evidence rather than executed.'
          family: report-line-contract
          round: 3
        - id: BR-8
          severity: Important
          title: parse_duration's validity guard is unreachable dead code, so malformed input yields a fabricated number
          detail: 'nvim/doctor.lua:67-73 — `parts[#parts+1] = tonumber(piece)` never stores a nil, so `if not v then return nil end` can never fire. Verified: parse_duration(''12:garbage'')==12, (''garbage:12'')==12, (''1:2:3:4:5'')==13403045. The docstring''s promise that an unreadable value returns nil rather than a number is not delivered for any input with at least one numeric field. Assign tonumber to a local and check it before appending, add a field-count bound, and add `parse_duration(''12:garbage'') == nil` to doctor_test.lua — it goes red today.'
          family: failure-reported-as-measurement
          round: 3
        - id: BR-9
          severity: Important
          title: doctor/perf.sh has no test and no make recipe, and delta's tests use literals rather than the promised recorded fixtures
          detail: doctor/doctor_test.sh + `make test-doctor` (Makefile.local:266-267) is the local precedent, created in response to a prior M1 review. The 128-line M1 deliverable has neither. Separately the plan's ARCH-MOCK promises delta is "tested against recorded fixture pairs checked into the repo"; what shipped is hand-written literals asserted against code written from the same mental model, with no parser and nothing pinning that perf.sh emits what delta expects. The stray-0 bug above is exactly what a line-shape smoke test would have caught. Add doctor/perf_test.sh + a test-perf recipe wired into `make test`, and drive delta from a captured real sample.
          family: untested-shell-surface
          round: 3
        - id: BR-10
          severity: Important
          title: doctor/perf.sh is missing from runtimebundlegen.explicitAssetPaths — third instance of the hand-maintained-list family
          detail: 'cmd/internal/runtimebundlegen/generate.go:27-32 is a hand-maintained list; assetDirs walks nvim/ wholesale but not doctor/. Verified against the checked-in manifest: doctor/doctor.sh present, doctor/perf.sh absent. An installed or Homebrew pair therefore extracts the drift half and not the perf half, and M2''s $PAIR_HOME/doctor/perf.sh will point at nothing. pair-hoprtt needs a decision too — the bundle carries no helper binaries since 104 M3. Add "doctor/perf.sh" to explicitAssetPaths and to embed_test.go:25-26, then `make runtimebundle-generate`. Sweep the whole enumeration (GO_BINS, artifactpath, explicitAssetPaths, make recipe) rather than this one site.'
          family: unverified-repo-mechanism
          round: 3
        - id: BR-11
          severity: Important
          title: The 6s budget is declared "enforced, not hoped" but perf.sh has no deadline or skip path
          detail: 'The plan''s ARCH-CONSTRAINTS says perf.sh takes a deadline and reports which probes were skipped when exceeded. perf.sh has no elapsed tracking and no skip logic — only fixed sample counts across `top -l 2` + sleep + `iostat -w 1 -c 2` + 500 pipe hops + 30 fork+execs + 5 zellij round-trips, sequential and unbounded, on the only kind of machine it ever runs on. My 2.19s measurement is not evidence: ps/top/sysctl were denied in that run so most collectors did nothing. Either implement the deadline or revise the plan to say the budget is sized, not enforced.'
          family: unenforced-operating-envelope
          round: 3
        - id: BR-12
          severity: Minor
          title: at_ns holds `date +%s` seconds, a live trap for M2's unwritten parser
          family: report-line-contract
          round: 3
        - id: BR-13
          severity: Minor
          title: The vm_stat awk pipeline is duplicated verbatim at perf.sh:69 and :95; extract swapstat()
          family: duplicated-logic
          round: 3
        - id: BR-14
          severity: Minor
          title: PAIR_HOME unset yields PROBE=/bin/pair-hoprtt and the degradation text misdiagnoses it as "not built"
          family: failure-reported-as-measurement
          round: 3
        - id: BR-15
          severity: Minor
          title: summary() indexes an empty slice; unreachable today but one guard line makes it safe for reuse
          family: unguarded-edge-case
          round: 3
        - id: BR-16
          severity: Minor
          title: perf.sh discards hoprtt's sample count, so a truncated pipe run reads identically to a full one
          family: failure-reported-as-measurement
          round: 3
        - id: BR-17
          severity: Minor
          title: perf.sh:61 awk $4 truncates command paths containing spaces, and comm= emits full paths where the plan said process name
          family: report-line-contract
          round: 3
        - id: BR-18
          severity: Minor
          title: PAIR_PERF_WINDOW flows unvalidated into sleep and awk -v, and is undocumented in atlas/README
          family: unguarded-edge-case
          round: 3
        - id: BR-19
          severity: Minor
          title: 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary; note it so M2's base is not mistaken for the branch point
          family: boundary-hygiene
          round: 3
        - id: BR-20
          severity: Minor
          title: doctor/README.md describes the doctor/ contents and was not updated for perf.sh (atlas/index.md was)
          family: docs-gate
          round: 3
        - id: BR-21
          severity: Minor
          title: Issue Plan M1 is unticked and the Log records no M1.4/M1.5 evidence; given the zellij finding, M1.4's number must be re-taken before it is logged
          family: traceability
          round: 3
      boundary: M1
      blocked: true
---

# Gate ledger — pair#208 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T23:35:17-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unverified-repo-mechanism` M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
  2nd finding in this family, so the rule is the deliverable, not the site: a
  plan may name a repo mechanism only with a file:line AND a check that it is
  reachable from the language and layer the calling code sits in. Prevalence
  2/2 rounds that named a mechanism — PQ-1 failed build-path reachability,
  this fails language reachability. artifactpath is cmd/internal/artifactpath
  with no CLI surface; CaptureRecord lives in nvim/doctor.lua. Use the
  existing idiom, pair_data_dir() at nvim/init.lua:484 plus pair_tag()
  (nvim/init.lua:4012). Sweeping the plan's other named mechanisms under the
  same rule found one more: the binary is named `hoprtt`, but
  Makefile.local:5-7 requires the `pair-` prefix on every GO_BINS entry
  because make install (:84-87) puts each on PATH — the collision that
  comment records already fixing once for `scribe`. Name it `pair-hoprtt`.
  (carried from plan-quality PQ-6, deferred to the boundary review)

## Round 2 — 2026-09-06T23:35:17-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-06T23:39:21-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `failure-reported-as-measurement` spawnRTT discards c.Run()'s error, so a failing command is reported as a healthy latency
  cmd/pair-hoprtt/main.go:96 uses `_ = c.Run()`. Reproduced: `hoprtt -spawn 10 -- /usr/bin/definitely-not-here` prints `0.599 ... 10` and exits 0, and because `zellij action query-tab-names` exits 1 here ("There is no active session!"), doctor/perf.sh printed `zellij_action_ms=10.183` — a fabricated healthy reading for the exact hypothesis (issue 201/203 round-trip cost) this issue exists to settle. perf.sh:117 guards only `command -v zellij`, not the session-gone case the plan's own ARCH-ORDER table names. Count failed runs and render `n/a (command failed: <err>)`; re-take M1.4's zellij number from inside a live session.
- **BR-3** [Critical] `silent-drop-in-join` delta silently drops a pid alive in both samples but missing one cputime row, contradicting its own contract
  nvim/doctor.lua:86-92 — `if ca and cb then` has no else, so a pid present in both procs samples but absent from one cputime sample lands in no bucket. Reproduced: procs A={1,2} B={1,2}, cpu A={1,2} B={1} yields rates=1 vanished=0 started=0 reused=0. Reachable in production because perf.sh:60-65 collects procs and cputime in two separate `ps -A` invocations, which diverge most under a spawn storm — the scenario the counters exist for. Fix both ends: one `ps -Ao pid=,etime=,time=,rss=,comm=` pass in perf.sh, and a counted `no_cputime` bucket in delta with a test asserting every input pid lands in exactly one bucket.
- **BR-4** [Critical] `build-artifact-committed` A 2.9MB Mach-O arm64 binary is committed at the repo root, against this window's own gitignore rule
  c290191c added `pair-hoprtt` (2,896,786 bytes, Mach-O 64-bit arm64) at the tree root; 95f73548 then added `/pair-hoprtt` to .gitignore:53, which has no effect on an already-tracked file. Still tracked at HEAD in a base-layer repo whose files propagate. Prefer rebasing c290191c to drop the blob entirely rather than `git rm --cached` in a follow-up, since the branch is unmerged and the history is still cheap to fix.
- **BR-5** [Important] `failure-reported-as-measurement` perf.sh collector failures render as values, not n/a, contradicting the file's own stated rule
  With ps/top/sysctl unavailable (reproduced in this shell), doctor/perf.sh emitted `host_cores=?`, `load=`, `cpu_idle_pct=`, `process_count=0`, and empty `### cputime` / `### procs` / `## disk` blocks — `process_count=0` being a fabricated claim and the empty values indistinguishable from a real reading. The header (:19-20) promises every probe degrades to `n/a` with a reason. Realistic, not academic: the header advertises standalone re-runs by an agent, and agents here run sandboxed with exactly these tools denied. One `probe()` helper substituting `n/a (<tool> unavailable)`, applied to every collector including both sample blocks.
- **BR-6** [Important] `report-line-contract` grep -c with `|| echo 0` emits a stray bare 0 line, corrupting the report on any quiet machine
  doctor/perf.sh:50-51 — `grep -c` prints 0 AND exits 1, so the `||` fires and appends a second 0. Reproduced: `build_procs=0` followed by a bare `0` line, which breaks the k=v line contract M2's parser will read. build_procs is zero whenever no build is running, i.e. on the healthy-baseline capture the Done-when requires. Drop the `|| echo 0`, or use `|| true`.
- **BR-7** [Important] `report-line-contract` macOS `ps -o comm=` is a path, so `^go$` never matches and the pair pattern over-matches
  doctor/perf.sh:50-51 anchors `^go$` against a full path like /opt/homebrew/.../bin/go, silently zeroing issue 203's build-storm variable (compile$/link$ still match). `pair_family_procs` matches unanchored `pair` against full paths, so every process launched from /Users/.../workspace/pair/ counts. The repo's own doctor/emitter-health.sh:25-31 documents comm= as a full path. Match on the basename before grepping and anchor. Caveat: `ps` is denied in my shell, so this is reasoned from that path-format evidence rather than executed.
- **BR-8** [Important] `failure-reported-as-measurement` parse_duration's validity guard is unreachable dead code, so malformed input yields a fabricated number
  nvim/doctor.lua:67-73 — `parts[#parts+1] = tonumber(piece)` never stores a nil, so `if not v then return nil end` can never fire. Verified: parse_duration('12:garbage')==12, ('garbage:12')==12, ('1:2:3:4:5')==13403045. The docstring's promise that an unreadable value returns nil rather than a number is not delivered for any input with at least one numeric field. Assign tonumber to a local and check it before appending, add a field-count bound, and add `parse_duration('12:garbage') == nil` to doctor_test.lua — it goes red today.
- **BR-9** [Important] `untested-shell-surface` doctor/perf.sh has no test and no make recipe, and delta's tests use literals rather than the promised recorded fixtures
  doctor/doctor_test.sh + `make test-doctor` (Makefile.local:266-267) is the local precedent, created in response to a prior M1 review. The 128-line M1 deliverable has neither. Separately the plan's ARCH-MOCK promises delta is "tested against recorded fixture pairs checked into the repo"; what shipped is hand-written literals asserted against code written from the same mental model, with no parser and nothing pinning that perf.sh emits what delta expects. The stray-0 bug above is exactly what a line-shape smoke test would have caught. Add doctor/perf_test.sh + a test-perf recipe wired into `make test`, and drive delta from a captured real sample.
- **BR-10** [Important] `unverified-repo-mechanism` doctor/perf.sh is missing from runtimebundlegen.explicitAssetPaths — third instance of the hand-maintained-list family
  cmd/internal/runtimebundlegen/generate.go:27-32 is a hand-maintained list; assetDirs walks nvim/ wholesale but not doctor/. Verified against the checked-in manifest: doctor/doctor.sh present, doctor/perf.sh absent. An installed or Homebrew pair therefore extracts the drift half and not the perf half, and M2's $PAIR_HOME/doctor/perf.sh will point at nothing. pair-hoprtt needs a decision too — the bundle carries no helper binaries since 104 M3. Add "doctor/perf.sh" to explicitAssetPaths and to embed_test.go:25-26, then `make runtimebundle-generate`. Sweep the whole enumeration (GO_BINS, artifactpath, explicitAssetPaths, make recipe) rather than this one site.
- **BR-11** [Important] `unenforced-operating-envelope` The 6s budget is declared "enforced, not hoped" but perf.sh has no deadline or skip path
  The plan's ARCH-CONSTRAINTS says perf.sh takes a deadline and reports which probes were skipped when exceeded. perf.sh has no elapsed tracking and no skip logic — only fixed sample counts across `top -l 2` + sleep + `iostat -w 1 -c 2` + 500 pipe hops + 30 fork+execs + 5 zellij round-trips, sequential and unbounded, on the only kind of machine it ever runs on. My 2.19s measurement is not evidence: ps/top/sysctl were denied in that run so most collectors did nothing. Either implement the deadline or revise the plan to say the budget is sized, not enforced.
- **BR-12** [Minor] `report-line-contract` at_ns holds `date +%s` seconds, a live trap for M2's unwritten parser
- **BR-13** [Minor] `duplicated-logic` The vm_stat awk pipeline is duplicated verbatim at perf.sh:69 and :95; extract swapstat()
- **BR-14** [Minor] `failure-reported-as-measurement` PAIR_HOME unset yields PROBE=/bin/pair-hoprtt and the degradation text misdiagnoses it as "not built"
- **BR-15** [Minor] `unguarded-edge-case` summary() indexes an empty slice; unreachable today but one guard line makes it safe for reuse
- **BR-16** [Minor] `failure-reported-as-measurement` perf.sh discards hoprtt's sample count, so a truncated pipe run reads identically to a full one
- **BR-17** [Minor] `report-line-contract` perf.sh:61 awk $4 truncates command paths containing spaces, and comm= emits full paths where the plan said process name
- **BR-18** [Minor] `unguarded-edge-case` PAIR_PERF_WINDOW flows unvalidated into sleep and awk -v, and is undocumented in atlas/README
- **BR-19** [Minor] `boundary-hygiene` 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary; note it so M2's base is not mistaken for the branch point
- **BR-20** [Minor] `docs-gate` doctor/README.md describes the doctor/ contents and was not updated for perf.sh (atlas/index.md was)
- **BR-21** [Minor] `traceability` Issue Plan M1 is unticked and the Log records no M1.4/M1.5 evidence; given the zellij finding, M1.4's number must be re-taken before it is logged

## Open findings

- **BR-1** [Minor] `unverified-repo-mechanism` M2.6 routes the Lua rolling-file write through artifactpath, a Go internal package Lua cannot call
- **BR-2** [Critical] `failure-reported-as-measurement` spawnRTT discards c.Run()'s error, so a failing command is reported as a healthy latency
- **BR-3** [Critical] `silent-drop-in-join` delta silently drops a pid alive in both samples but missing one cputime row, contradicting its own contract
- **BR-4** [Critical] `build-artifact-committed` A 2.9MB Mach-O arm64 binary is committed at the repo root, against this window's own gitignore rule
- **BR-5** [Important] `failure-reported-as-measurement` perf.sh collector failures render as values, not n/a, contradicting the file's own stated rule
- **BR-6** [Important] `report-line-contract` grep -c with `|| echo 0` emits a stray bare 0 line, corrupting the report on any quiet machine
- **BR-7** [Important] `report-line-contract` macOS `ps -o comm=` is a path, so `^go$` never matches and the pair pattern over-matches
- **BR-8** [Important] `failure-reported-as-measurement` parse_duration's validity guard is unreachable dead code, so malformed input yields a fabricated number
- **BR-9** [Important] `untested-shell-surface` doctor/perf.sh has no test and no make recipe, and delta's tests use literals rather than the promised recorded fixtures
- **BR-10** [Important] `unverified-repo-mechanism` doctor/perf.sh is missing from runtimebundlegen.explicitAssetPaths — third instance of the hand-maintained-list family
- **BR-11** [Important] `unenforced-operating-envelope` The 6s budget is declared "enforced, not hoped" but perf.sh has no deadline or skip path
- **BR-12** [Minor] `report-line-contract` at_ns holds `date +%s` seconds, a live trap for M2's unwritten parser
- **BR-13** [Minor] `duplicated-logic` The vm_stat awk pipeline is duplicated verbatim at perf.sh:69 and :95; extract swapstat()
- **BR-14** [Minor] `failure-reported-as-measurement` PAIR_HOME unset yields PROBE=/bin/pair-hoprtt and the degradation text misdiagnoses it as "not built"
- **BR-15** [Minor] `unguarded-edge-case` summary() indexes an empty slice; unreachable today but one guard line makes it safe for reuse
- **BR-16** [Minor] `failure-reported-as-measurement` perf.sh discards hoprtt's sample count, so a truncated pipe run reads identically to a full one
- **BR-17** [Minor] `report-line-contract` perf.sh:61 awk $4 truncates command paths containing spaces, and comm= emits full paths where the plan said process name
- **BR-18** [Minor] `unguarded-edge-case` PAIR_PERF_WINDOW flows unvalidated into sleep and awk -v, and is undocumented in atlas/README
- **BR-19** [Minor] `boundary-hygiene` 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary; note it so M2's base is not mistaken for the branch point
- **BR-20** [Minor] `docs-gate` doctor/README.md describes the doctor/ contents and was not updated for perf.sh (atlas/index.md was)
- **BR-21** [Minor] `traceability` Issue Plan M1 is unticked and the Log records no M1.4/M1.5 evidence; given the zellij finding, M1.4's number must be re-taken before it is logged
