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
    - "n": 4
      timestamp: "2026-09-06T23:55:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Rename to pair-hoprtt landed and GO_BINS is correct; plan M2.6 line 387-391 still routes the Lua rolling-file write through artifactpath.
          round: 4
        - id: BR-2
          disposition: addressed
          note: 'Verified live, not from the diff: zellij probe now renders n/a (probe failed) instead of a fabricated 10.183; partial failure emits the 5-field line. No regression test pins it — folded into BR-9.'
          round: 4
        - id: BR-3
          disposition: addressed
          note: 'Revert-verified red (unmeasured counted, not dropped). Residual: perf.sh still runs two separate ps passes (:84,:87), so storm pids land in unmeasured rather than being measured; one ps -Ao pid=,etime=,time=,rss=,comm= pass removes the bucket.'
          round: 4
        - id: BR-4
          disposition: addressed
          note: pair-hoprtt is untracked at HEAD (removed in 0a7e6e27). The 2.9MB blob remains in c290191c; the branch is still unpushed, so dropping it by rebase is cheap now and permanent after merge.
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: 'collect() is defined at perf.sh:32 with ZERO call sites. Re-ran the capture: load=, cpu_idle_pct=, process_count=0, host_cores=?, empty ## disk block. The fix reads as protection while doing nothing.'
          round: 4
        - id: BR-6
          disposition: addressed
          note: 'Revert-verified: re-adding `|| echo 0` to count_comm makes perf_test.sh fail with "report contains a bare value line with no key".'
          round: 4
        - id: BR-7
          disposition: addressed
          note: count_comm now takes the basename via awk -F/ and anchors the pattern. Not executable here (ps denied), reasoned from the comm= path format the fix itself documents.
          round: 4
        - id: BR-8
          disposition: addressed
          note: 'Revert-verified red. Residual (Minor, not re-raised): no field-count or sign bound, so 1:2:3:4:5 -> 13403045, 0x10 -> 16, -1:00 -> -60.'
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: perf_test.sh + make test-perf-capture exist and are wired into make test, but pin only line shape and the missing-binary path; nothing pins a failing probe or a failing collector, and delta is still literals rather than recorded fixture pairs.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: explicitAssetPaths, both artifactpath lists, GO_BINS and the recipe are done and doctor/perf.sh is in the generated manifest. embed_test.go:25-26 was named and skipped, and the pair-hoprtt bundling decision was never made - see the new Critical.
          round: 4
        - id: BR-11
          disposition: not-addressed
          note: over_budget() guards only probe_line (:142). top -l 2, sleep WINDOW and iostat -c 2 are unconditional and are the budget; the total is still unbounded and the skip drops the probes rather than the expensive collectors.
          round: 4
        - id: BR-12
          disposition: not-addressed
          note: perf.sh:97,107 still emit at_ns from date +%s.
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: vm_stat awk pipeline still duplicated verbatim at perf.sh:93 and :119.
          round: 4
        - id: BR-14
          disposition: not-addressed
          note: perf.sh:23 still yields /bin/pair-hoprtt when PAIR_HOME is unset; :161's reason still says "not built".
          round: 4
        - id: BR-15
          disposition: not-addressed
          note: summary (main.go:41-45) still indexes an empty slice with no guard.
          round: 4
        - id: BR-16
          disposition: not-addressed
          note: probe_line's awk still discards $4 on the success path.
          round: 4
        - id: BR-17
          disposition: not-addressed
          note: sample() at perf.sh:85 still uses awk $4; comm= still emits full paths.
          round: 4
        - id: BR-18
          disposition: not-addressed
          note: Now wider - PAIR_PERF_BUDGET joins PAIR_PERF_WINDOW, both unvalidated into sleep/awk -v/$(( )) and undocumented in atlas and README.
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: '4f9365b3 is still in the window, and verdict/note_from_lines landed with it; no ## Log entry records the boundary.'
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: doctor/README.md still does not mention perf.sh.
          round: 4
        - id: BR-21
          disposition: not-addressed
          note: 'Issue Plan M1 (line 133) unticked; ## Log has no M1.4 or M1.5 evidence.'
          round: 4
      findings:
        - id: BR-22
          severity: Critical
          title: perf.sh looks for its probe at $PAIR_HOME/bin/pair-hoprtt, a path that never exists in a shipped pair
          detail: |-
            3rd finding in this family, so the deliverable is the RULE, not the site.
            Rule: a plan or diff may name a repo path or mechanism only alongside a
            check that it resolves in every layout the artifact ships to (dev
            checkout, make install, extracted runtime bundle) AND from the language
            and layer the calling code sits in. Prevalence 3/3 rounds that named a
            mechanism: PQ-1 failed the build path, BR-1 failed the language path,
            this fails the install-layout path. Evidence: the generated
            manifest.json's bin/ entries are exactly bin/lib/adapt-log.sh,
            bin/lib/dev-rebuild.sh, bin/pair-help, bin/pair-notify - no Go binaries
            since 104 M3. This window added doctor/perf.sh to the bundle, so the
            shell ships and its payload does not; every installed pair prints
            probes=n/a and is told to run make build, which does not apply to its
            layout. Sweep the enumeration the rule implies for every new runtime
            file: builds, in the bundle manifest, callers' path expressions resolve
            under all three layouts, a test pins each.
          family: unverified-repo-mechanism
          round: 4
        - id: BR-23
          severity: Important
          title: verdict() turns absent timings into 'fast', and its test asserts that direction
          detail: |-
            6th finding in this family, so state the rule rather than patching the
            site. Rule: every value-producing function in the capture, shell and Lua
            alike, must render an absent or failed measurement as absence (nil, or
            n/a with a reason), never as an in-domain value. Prevalence 6/6: BR-2,
            BR-5, BR-14, BR-16, the dead collect() helper, and this. At
            doctor.lua:129 `tonumber(x) or 0` makes verdict(nil,nil)=='fast', and per
            the Spec (issue lines 77-81) 'fast' is what excludes the whole scheduling
            family (201/203) - so a report where nvim was never timed excludes two
            issues on a measurement that did not happen. doctor.lua:118-124 states
            the correct rule three functions above. doctor_test.lua:52 asserts the
            wrong direction, which is the same-mental-model test the gate warns
            about. Enumeration to sweep in one pass: every kv call site in perf.sh,
            every exported function in doctor.lua's 208 block, and report/summary in
            cmd/pair-hoprtt/main.go - each gets one absent-input case.
          family: failure-reported-as-measurement
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-07T00:17:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The naming half landed pre-window (df283abe); the plan file was not touched in this window and M2.6:389 still names artifactpath, which has no CLI surface — every caller is Go under cmd/.
          round: 5
        - id: BR-5
          disposition: not-addressed
          note: The kv collectors are fixed and revert-verified, but sample()/cputimes() (perf.sh:121-122) bypass collect() and their `|| say "n/a (ps unavailable)"` at :137/:139 is dead — the pipeline exits with awk's status, so a failed ps yields silently empty blocks.
          round: 5
        - id: BR-9
          disposition: addressed
          note: perf_test.sh + test-perf-capture wired into `make test`, and parse_samples now pins the perf.sh -> delta contract against a real recorded capture.
          round: 5
        - id: BR-10
          disposition: addressed
          note: generate.go, embed_test.go and the artifactpath manifest all carry doctor/perf.sh; the regenerated manifest.json contains it and the bundle tests pass.
          round: 5
        - id: BR-11
          disposition: addressed
          note: A stage-level deadline with skip reasons and an emitted elapsed_seconds now exists; the shed ORDER it produces is raised separately.
          round: 5
        - id: BR-12
          disposition: addressed
          round: 5
        - id: BR-13
          disposition: addressed
          round: 5
        - id: BR-14
          disposition: addressed
          round: 5
        - id: BR-15
          disposition: not-addressed
          note: cmd/pair-hoprtt/main.go:41 summary() still indexes samples[0] with no length guard; still unreachable from main, still one line from being safe.
          round: 5
        - id: BR-16
          disposition: not-addressed
          note: probe_line's awk (perf.sh:206) still prints only $1 and $2; the sample count $4 is discarded, so a pipeRTT that broke out early reads identically to a full 500-sample run.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: perf.sh:121 still takes awk $4 of `comm=`, truncating any executable path containing a space, and still emits full paths where the plan said process name.
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: PAIR_PERF_WINDOW and PAIR_PERF_BUDGET still flow unvalidated into sleep, `[ -ge ]` and `awk -v`, and neither appears in atlas/index.md or doctor/README.md.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: 4f9365b3 is still inside the M1 window along with verdict/note_from_lines; no Log entry or plan note records that M2's base is not the branch point.
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: doctor/README.md is not in this window's diff and still describes only the drift half; the atlas entry landed but the README a reader of doctor/ opens did not.
          round: 5
        - id: BR-21
          disposition: not-addressed
          note: Plan M1 is still unticked and the Log still ends at 2026-09-06; the M1.4/M1.5 numbers now exist in the fixture (pipe 0.006ms, fork 1.588ms, zellij 13.323ms, elapsed 5) but are not recorded in the issue.
          round: 5
        - id: BR-22
          disposition: not-addressed
          note: PATH-first fixes `make install` only. ../homebrew-pair/Formula/pair.rb builds solely ./cmd/pair-go and installs bin/nvim/zellij, and PAIR_HOME at runtime is the extracted bundle root, which carries no helper binaries — so a shipped pair still prints probes=n/a with advice that does not apply to its layout. I reverted the PATH-first block in a scratch copy and the whole suite stayed green, so no test pins the fix.
          round: 5
        - id: BR-23
          disposition: not-addressed
          note: 'verdict(nil,nil) is fixed, but verdict(nil, 2) still returns ''fast'' and doctor_test.lua:56 asserts that direction — a half-absent measurement rendered as a full in-domain verdict is the same defect at a different arity. Of the enumeration BR-23 named, only doctor.lua''s exports were swept: perf.sh''s sample blocks still fabricate absence (see BR-5), parse_samples has no absent-input case, and summary/report in cmd/pair-hoprtt still panic on empty input.'
          round: 5
      findings:
        - id: BR-24
          severity: Critical
          title: doctor/perf_test.sh:51's bare `mktemp -d` makes `make test-perf-capture`, and therefore `make test`, fail in a sandboxed agent shell
          detail: |-
            This is the 4th finding in family `unverified-repo-mechanism`, so the rule
            is the deliverable, not the site. Rule: a mechanism this repo names must be
            checked in EVERY environment and layout it will run in, enumerated in one
            place — dev checkout, `make install` PATH, extracted runtime bundle,
            Homebrew formula install, and the sandboxed agent shell that runs `make
            test` before every close. Prevalence 4/4 rounds that named a mechanism:
            PQ-1 failed the build path, BR-1 the language path, BR-22 the install-layout
            path, this the execution-environment path.
            Reproduced: `make test-perf-capture` exits 1 with `mktemp: mkdtemp failed
            on /var/folders/...: Operation not permitted`, before any assertion runs,
            because macOS mktemp with no template reads confstr(_CS_DARWIN_USER_TEMP_DIR)
            and cannot be redirected with TMPDIR. Verified fix and green suite with
            `mktemp -d "${TMPDIR:-/tmp}/perfstub.XXXXXX"`. Note this is the same file
            whose header advertises that it is "routinely run by sandboxed agents".
          family: unverified-repo-mechanism
          round: 5
        - id: BR-25
          severity: Important
          title: The budget sheds the three probes first, so a degraded machine — the only condition this tool targets — yields a capture with no probe rows
          detail: |-
            This is the 2nd finding in family `unenforced-operating-envelope`. Rule,
            not site: a capture's budget must reserve capacity for the measurements the
            issue exists to take and shed contextual collectors before diagnostic ones;
            sequential position in the script is not a priority order. Reproduced with
            `PAIR_PERF_BUDGET=2 sh doctor/perf.sh`: disk, pipe_hop, fork_exec and
            zellij_action all render `n/a (budget exceeded...)` while `top -l 2 -n 60`
            — the single most expensive collector, and unbounded on a loaded host — is
            checked before it starts and therefore always runs. The healthy fixture
            already reports elapsed_seconds=5 of 6. The probes are the only rows with
            published baselines and are Done-when item 1. Fix: order the probes ahead
            of top/iostat, or reserve a probe slice of the budget. Secondary: this makes
            perf_test.sh:36's `[ elapsed -le budget ]` load-dependent, so `make test`
            goes red under exactly the conditions the capture is for.
          family: unenforced-operating-envelope
          round: 5
        - id: BR-26
          severity: Minor
          title: perf_test.sh:23's stray-line check only fires for all-digit lines, so any other unattributable line passes
          detail: |-
            The `*[!0-9]*) continue` arm (there to skip tab-separated ps rows) swallows
            every stray line containing a non-digit. Verified: injecting `say "an
            unattributable stray line"` into a scratch perf.sh left the suite green; a
            bare `0` is correctly caught. Match the ps rows positively (a line with a
            tab, inside a `### ` block) instead of negatively by character class.
          family: untested-shell-surface
          round: 5
        - id: BR-27
          severity: Minor
          title: Four shapes of collect()'s availability/failure/empty ladder in one file, which is how the sample blocks escaped the rule
          detail: |-
            collect() (perf.sh:43) exists to be that ladder, but it can only emit one
            key, so the top block (~:88), the disk block (:171) and probe_line (:202)
            each re-implement it and sample()/cputimes() (:121-122) skip it entirely.
            A `stage()` variant that takes a multi-key emitter would collapse all four
            and is the structural fix behind the BR-5 residual.
          family: duplicated-logic
          round: 5
        - id: BR-28
          severity: Minor
          title: perf.sh:163's swap_rate is shell arithmetic, contradicting the file's own rule 2, and divides by WINDOW unguarded
          detail: |-
            This is the 5th finding in family `report-line-contract`. Rule: every value
            the report emits either comes from a tool verbatim or is computed in tested
            Lua; perf.sh's header rule 2 (line 13) states this and the swap block
            violates it. Same block yields inf at WINDOW=0 and negative rates if a
            counter goes backwards. Either move it into doctor.delta with the pid join
            or amend rule 2 to say "no per-process arithmetic".
          family: report-line-contract
          round: 5
        - id: BR-29
          severity: Minor
          title: A vm_stat that exists but exits non-zero renders `swap=n/a (vm_stat unavailable)`, misnaming the failure
          detail: |-
            This is the 7th finding in family `failure-reported-as-measurement`. The
            rule the family needs, stated once: every emitter distinguishes THREE
            outcomes — tool absent, tool failed, tool returned nothing — and names which
            one it hit, because "unavailable" sends a reader to install something that
            is already installed. collect() already does this correctly; the hand-rolled
            swap block at perf.sh:155-157 does not. Reproduced with a `vm_stat` stub
            that exits 1.
          family: failure-reported-as-measurement
          round: 5
        - id: BR-30
          severity: Minor
          title: Comment drift in nvim/doctor.lua — the fixture path and delta's documented return shape are both wrong
          detail: |-
            This is the 2nd finding in family `docs-gate`. Rule: a comment that names a
            path or a return shape is a claim a test can check; when it names a
            filesystem path, assert it, and when it enumerates fields, enumerate all of
            them. doctor.lua:157 says "The fixture in nvim/fixtures/" (it is in
            doctor/fixtures/, and doctor_test.lua explains why it must not be under
            nvim/); doctor.lua:75's "Returns { rates, vanished, started, reused, rows_a,
            rows_b }" omits `unmeasured`, which the code sets and the test asserts.
          family: docs-gate
          round: 5
        - id: BR-31
          severity: Minor
          title: delta detects a reused pid only through etime, so an unparseable etime lets a reused pid produce a negative cpu_pct
          detail: |-
            This is the 3rd finding in family `unguarded-edge-case`. Rule: a derived
            value with a known-impossible range must be rejected at the point of
            derivation, not only by the proxy signal that usually implies it. At
            doctor.lua:95, if pa.etime or pb.etime is nil the reuse branch is skipped
            and the pid falls into the rate branch, where cb < ca yields a negative
            rate. `cb < ca` is itself sufficient evidence of reuse and is cheaper than
            the etime comparison; the fixture test's `r.cpu_pct >= 0` assertion would
            then be enforced by the code rather than by the fixture's luck.
          family: unguarded-edge-case
          round: 5
        - id: BR-32
          severity: Minor
          title: doctor/fixtures/perf_capture.txt is a real host capture in a public repo, safe only by an unrecorded truncation accident
          detail: |-
            The fixture carries zero `/Users/` paths only because it was cut to the
            first 60 pids, which on this host are all system daemons. Nothing records
            that as the selection rule, and perf.sh emits full executable paths, so the
            next re-capture (a wider window, a busier machine) will commit
            home-directory and project paths. Write the redaction/selection rule next
            to the fixture, or filter comm to its basename in sample().
          family: recorded-fixture-redaction
          round: 5
      boundary: M1
      blocked: true
    - "n": 6
      timestamp: "2026-09-07T00:37:16-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan untouched since the base commit; :389 still routes the Lua write through artifactpath and :79/:98-108/:311 still describe the reversed cmd/pair-hoprtt + GO_BINS design.
          round: 6
        - id: BR-5
          disposition: not-addressed
          note: Residual is the two sample blocks; reproduced live and under perf_test.sh's own stub PATH — `### cputime`/`### procs` render empty because the pipeline's exit status is awk's, not ps's.
          round: 6
        - id: BR-15
          disposition: addressed
          note: Guard at hoprtt.go:53 plus TestSummaryOnEmptyInputDoesNotPanic; note the test asserts med==0, an in-domain value, which teaches the wrong contract even though the path is unreachable from Run.
          round: 6
        - id: BR-16
          disposition: not-addressed
          note: probe_line still prints only $1/$2; the sample count $4 is used solely in the failure branch, so a pipeRTT that broke early still reads as a full run.
          round: 6
        - id: BR-17
          disposition: not-addressed
          note: sample() at perf.sh:130 still takes awk $4 and comm is still a full path; under ARCH-SECURE this is worth more than Minor because the report is designed to leave the machine.
          round: 6
        - id: BR-18
          disposition: not-addressed
          note: 'Widened, not fixed — PAIR_PERF_BUDGET=abc emits three `[: integer expression expected` errors and still reports; PAIR_PERF_WINDOW=''1; echo pwned'' yields window_seconds=1; echo pwned. Still undocumented in atlas/README.'
          round: 6
        - id: BR-19
          disposition: not-addressed
          note: Nothing in the window records that 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary.
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: doctor/README.md is untouched in this window and README.md:607 points readers there for the doctor surface.
          round: 6
        - id: BR-21
          disposition: not-addressed
          note: 'All M1 checkboxes remain `- [ ]` and the Log has no 2026-09-07 entry. Numbers now available to record: pipe 0.006 ms, fork+exec 1.5-1.66 ms, perf.sh 2.26 s of a 6 s budget; zellij unmeasurable in this shell.'
          round: 6
        - id: BR-22
          disposition: addressed
          note: Subcommand + dispatcher registration, pinned by TestProbeIsReachableThroughTheShippedPairBinary which builds ./cmd/pair-go; verified `./bin/pair hoprtt` returns 0.006 ms.
          round: 6
        - id: BR-23
          disposition: addressed
          note: verdict is now asymmetric and doctor_test.lua fails without it; the sweep the finding demanded is incomplete — see the new parse_samples finding.
          round: 6
        - id: BR-24
          disposition: addressed
          note: mktemp fallback at perf_test.sh:54; bare `mktemp -d` still fails in this shell, so the fallback is the live path and `make test-perf-capture` passes.
          round: 6
        - id: BR-25
          disposition: not-addressed
          note: 'Behaviour verified fixed (PAIR_PERF_BUDGET=2 keeps pipe_hop and fork_exec while shedding top/iostat/sample_b), but NO test exercises a squeeze and the probe keys are absent from perf_test.sh''s required-key list, so reverting the shed order stays green. Also the secondary is untouched: elapsed can still exceed budget because no stage is bounded.'
          round: 6
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23 still `*[!0-9]*) continue`.
          round: 6
        - id: BR-27
          disposition: not-addressed
          note: Still four shapes — collect() :51, the top block :101, disk :178, probe_line :211 — with sample()/cputimes() skipping the ladder entirely, which is the structural cause of BR-5's residual.
          round: 6
        - id: BR-28
          disposition: not-addressed
          note: Still awk arithmetic in perf.sh and still divides by WINDOW unguarded; worse now, because when the budget sheds sample_b the `sleep "$WINDOW"` never runs yet the rate is still divided by WINDOW, so the divisor names a window that did not occur.
          round: 6
        - id: BR-29
          disposition: not-addressed
          note: perf.sh:165-166 still says "n/a (vm_stat unavailable)" for a vm_stat that exists and failed.
          round: 6
        - id: BR-30
          disposition: not-addressed
          note: doctor.lua still says "The fixture in nvim/fixtures/" (it is doctor/fixtures/) and delta's Returns list still omits `unmeasured`, which the code sets and the test asserts.
          round: 6
        - id: BR-31
          disposition: not-addressed
          note: delta still detects reuse only via etime; an unparseable etime drops the pid into the rate branch where cb < ca yields a negative cpu_pct.
          round: 6
        - id: BR-32
          disposition: not-addressed
          note: No redaction or selection rule recorded next to doctor/fixtures/perf_capture.txt; it is clean of /Users/ paths only by the same truncation accident.
          round: 6
      findings:
        - id: BR-33
          severity: Critical
          title: The plan still specifies the cmd/pair-hoprtt + GO_BINS design that round 3 reversed, with no "## Revisions" entry
          detail: |-
            This is the 2nd finding in family `traceability`. Rule, not site: a durable
            artifact that describes the work is updated in the SAME round the work
            changes it, via an appended "## Revisions" entry — because the plan is what
            the next milestone is executed from, and a stale one re-introduces the
            finding the change removed. Prevalence 2/2 (BR-21: the issue Plan and Log
            do not record what M1 did; this: the plan does not record what M1 became).
            Evidence: the plan's last commit is df283abe, the review base. Core
            concepts line 79 says `pair-hoprtt` lives in `cmd/pair-hoprtt/main.go`;
            the code is `cmd/internal/hoprttcmd/hoprtt.go` and `cmd/pair-hoprtt/` was
            deleted in c8f2d044. Line 107 still says perf.sh "locates it as
            $PAIR_HOME/bin/pair-hoprtt" — verbatim the expression BR-22 called
            Critical. M1.2b (:311-313) still instructs a GO_BINS entry and a
            `bin/pair-hoprtt` check. M2.6 (:389) still routes the Lua rolling-file
            write through artifactpath, disposed not-addressed twice by the plan gate.
            Fix: one Revisions entry plus edits at :79, :98-108, :283, :311-313, :389,
            :402.
          family: traceability
          round: 6
        - id: BR-34
          severity: Important
          title: The budget is checked between stages but no stage is bounded, so a single slow collector blows it without limit
          detail: |-
            This is the 3rd finding in family `unenforced-operating-envelope`, so the
            deliverable is the rule: a wall-clock budget is enforced only when every
            operation that can exceed it is itself bounded; a check before a stage
            starts does not bound the stage. Prevalence 3/3 — BR-11 (no deadline at
            all), BR-25 (wrong shed order), this (no bound once a stage is entered).
            Evidence: `top -l 2 -n 60` (perf.sh:108), `iostat -d -w 1 -c 2` (:183),
            `sleep "$WINDOW"` (:158) and each probe_line invocation run to completion
            once started. The committed fixture reads elapsed_seconds=5 against
            budget_seconds=6 — one second of slack — and the operator symptom quoted
            in the issue is "top took 10 seconds". pipeRTT's hardcoded 500 samples
            (hoprtt.go:178) is 5 s at the ~10 ms/hop the issue says visible lag would
            require. Because perf_test.sh:36 asserts elapsed <= budget, `make test`
            goes red under exactly the conditions the capture exists for. Enumeration
            to sweep in one pass: wrap each external stage in a bounded runner, and
            derive probe sample counts from remaining budget and the observed first
            sample rather than from constants.
          family: unenforced-operating-envelope
          round: 6
        - id: BR-35
          severity: Important
          title: parse_samples turns a budget-shed sample_b into an empty-but-present sample, so delta reports every process as vanished
          detail: |-
            This is the 8th finding in family `failure-reported-as-measurement`. The
            rule was already stated at BR-23 and its sweep enumeration was written
            ("every exported function in doctor.lua's 208 block"); this is a member of
            that enumeration the hand-sweep missed, which is the family reporting that
            a hand-sweep is the wrong instrument. Reproduced under `nvim -l` with a
            capture containing `## sample_b` / `skipped=budget reserved for probes`:
            delta returns rates=0 vanished=2 started=0 rows_a=2 rows_b=0 — on a real
            host that is vanished=856, a fabricated spawn-storm claim from a
            measurement that never happened, in exactly the squeezed-capture condition
            the tool exists for. The same shape arises from BR-5's residual, where a
            failed ps yields two empty samples that read as "nothing is running".
            Deliverable at this prevalence is mechanical: one table-driven test in
            doctor_test.lua iterating the exported functions of the #208 block and
            asserting each maps absent/failed input to absence, so the next function
            added is covered by construction. Site fix: parse_samples returns nil for
            a block carrying `skipped=` or no `###` sections.
          family: failure-reported-as-measurement
          round: 6
        - id: BR-36
          severity: Important
          title: Nothing tests perf.sh's sample-row grammar, so the perf.sh-to-delta contract the Lua test claims to pin does not exist
          detail: |-
            This is the 3rd finding in family `untested-shell-surface`. Rule: when two
            components are joined by a recorded fixture, the recording pins only the
            CONSUMER; the producer needs its own live-run assertion, or drift on the
            producer side is invisible. Prevalence 3/3 (BR-9 no test for perf.sh at
            all; BR-25's fix unpinned; this). doctor_test.lua asserts "the perf.sh ->
            delta contract is pinned. A change to either side that breaks the other
            now fails here" — that claim is false: it reads the static file
            doctor/fixtures/perf_capture.txt and never invokes perf.sh. Demonstrated
            in a scratch copy: changing sample() from tab-separated to pipe-separated
            output left `sh doctor/perf_test.sh` printing "perf.sh shape tests passed"
            and exiting 0 (perf_test.sh:23's `*[!0-9]*) continue` swallows the rows,
            per BR-26). perf_test.sh's required-key list also omits every probe key,
            which is why BR-25's shed-order fix has no failing test. Fix: assert a live
            run's `### procs` rows against ^[0-9]+\t\S+\t[0-9]+\t.+$ and `### cputime`
            against ^[0-9]+\t\S+$, and add pipe_hop_ms/fork_exec_ms to the key list.
          family: untested-shell-surface
          round: 6
        - id: BR-37
          severity: Minor
          title: perf_test.sh's mktemp fallback writes .perf-test-stub.$$ into the worktree and nothing gitignores it
          detail: |-
            This is the 2nd finding in family `build-artifact-committed`. Rule: any
            path a build or test writes into the worktree is gitignored in the same
            change that introduces the write. Prevalence 2/2 (BR-4: the staged
            pair-hoprtt binary; this). perf_test.sh:54 falls back to
            "$repo/.perf-test-stub.$$" when mktemp fails — and bare `mktemp -d` fails
            in the sandboxed agent shell this file's header names as its target
            reader, so the fallback is the live path there, not the exception. An
            interrupted run leaves a directory of fake ps/top/sysctl executables
            untracked at the repo root.
          family: build-artifact-committed
          round: 6
      boundary: M1
      blocked: true
    - "n": 7
      timestamp: "2026-09-07T00:50:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan :389 still routes M2.6's rolling-file write through artifactpath; verified no CLI surface exists and pair_data_dir() (nvim/init.lua:484) remains the idiom. The GO_BINS/pair- prefix half is moot now the probe is a subcommand.
          round: 7
        - id: BR-5
          disposition: not-addressed
          note: collect() is right for the named collectors, but emit_sample's `cputimes 2>/dev/null || say "n/a (ps unavailable)"` is a pipeline whose status is awk's, so the fallback is unreachable; with ps denied here both sample blocks render empty and delta reads rows_a=0/rows_b=0 as "nothing is running".
          round: 7
        - id: BR-16
          disposition: not-addressed
          note: probe_line's awk (perf.sh:216-221) still emits only $1/$2; the sample count $4 survives only inside the failure message.
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: 'perf.sh:130 unchanged. Also: the plan''s ARCH-SECURE allowlist (name+pid for all, argv for go/compile/link/zellij/pair*/nvim) is not implemented at all.'
          round: 7
        - id: BR-18
          disposition: not-addressed
          note: WINDOW, BUDGET and the newly added PROBE_RESERVE are all unvalidated and undocumented; PROBE_RESERVE > BUDGET makes collectors_done true immediately.
          round: 7
        - id: BR-19
          disposition: not-addressed
          note: No mention of 4f9365b3 or M2.2b landing inside M1's window in the issue or the plan.
          round: 7
        - id: BR-20
          disposition: not-addressed
          note: grep for "perf" in doctor/README.md returns nothing. Root README also unchanged for `pair hoprtt` / `make test-perf-capture`, though its subcommand list is explicitly non-exhaustive.
          round: 7
        - id: BR-21
          disposition: not-addressed
          note: Issue Plan M1 still unticked; Log has only the 2026-09-06 entry. M1.4's zellij ~13ms baseline is still unmeasured — the probe renders `n/a (probe failed)` in this shell.
          round: 7
        - id: BR-25
          disposition: not-addressed
          note: 'Behavior IS fixed and I reproduced it (PAIR_PERF_BUDGET=2 sheds top/iostat/sample_b, probes survive), but nothing pins it: perf_test.sh''s key list carries no probe key and there is no squeezed-budget run, so reverting the shed order leaves make test green.'
          round: 7
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged; the new grammar loop is a separate pass and does not replace it.
          round: 7
        - id: BR-27
          disposition: not-addressed
          note: 'Four ladders still present: top (:101-120), disk (:178-189), probe_line (:211), and sample()/cputimes() (:130-131) bypassing collect() entirely.'
          round: 7
        - id: BR-28
          disposition: not-addressed
          note: perf.sh:172's swap arithmetic is unchanged and still divides by WINDOW unguarded.
          round: 7
        - id: BR-29
          disposition: not-addressed
          note: perf.sh:166 still reports a vm_stat that exists but exits non-zero as "(vm_stat unavailable)".
          round: 7
        - id: BR-30
          disposition: not-addressed
          note: doctor.lua:163 still says "The fixture in nvim/fixtures/"; :75 still omits `unmeasured` from the documented return shape.
          round: 7
        - id: BR-31
          disposition: not-addressed
          note: doctor.lua:90 unchanged; no `cb < ca` guard in the rate branch, so an unparseable etime still lets a reused pid produce a negative rate.
          round: 7
        - id: BR-32
          disposition: not-addressed
          note: No redaction or selection rule recorded; doctor/fixtures/ contains only perf_capture.txt, and sample() still emits full executable paths.
          round: 7
        - id: BR-33
          disposition: addressed
          note: Revisions entry appended recording the reversal, the shed order, verdict asymmetry and the fixture location; per AGENTS.md the append-don't-overwrite convention makes the body edits optional. The M2.6/artifactpath item it also named stays open as BR-1.
          round: 7
        - id: BR-34
          disposition: not-addressed
          note: top -l 2 -n 60 (:108), iostat -d -w 1 -c 2 (:183), sleep "$WINDOW" (:158), probe_line, and pipeRTT's hardcoded 500 samples (hoprtt.go:178) all still run to completion once entered.
          round: 7
        - id: BR-35
          disposition: not-addressed
          note: 'Reproduced verbatim under nvim -l: a capture with `## sample_b` / `skipped=budget reserved for probes` still yields rates=0 vanished=2 started=0 rows_a=2 rows_b=0. dc5a3d03 covers only '''' and "no sample block at all"; the table-driven test the finding asked for was not written.'
          round: 7
        - id: BR-36
          disposition: addressed
          note: Verified by mutation — pipe-separated sample() output fails the new grammar loop once ps produces rows. Its vacuity in a ps-denied shell is raised separately.
          round: 7
        - id: BR-37
          disposition: not-addressed
          note: .gitignore carries no .perf-test-stub pattern; perf_test.sh:54's fallback is unchanged.
          round: 7
      findings:
        - id: BR-38
          severity: Important
          title: perf_test.sh asserts a live run against the ambient system, so the BR-36 grammar pin validates zero rows wherever ps is denied
          detail: |-
            This is the 4th finding in family `untested-shell-surface`. Do NOT fix this
            instance — the rule is: a shell test must CONTROL the environment its
            subject reads; a live run against the ambient system asserts nothing when
            the system denies the tool, and a green run is then a sample of size zero
            reporting as coverage. Prevalence 4/4 (BR-9 no test at all; BR-25's fix
            unpinned; BR-36 the recorded fixture pinning only the consumer; this).
            Evidence: `ps` is denied in this agent shell (`/bin/ps: Operation not
            permitted`), so `### procs` and `### cputime` render as empty sections;
            the loop at perf_test.sh:84-99 iterates zero times, `rows` (:98) is
            computed and never read, and :101-102 assert only that the headers exist.
            I mutated sample() to pipe separators in a scratch copy and
            `sh doctor/perf_test.sh` printed "perf.sh shape tests passed" exit 0;
            the same mutation fails correctly once I put a recorded-output ps stub on
            PATH. Sweep in one pass: promote the existing stub dir (:54-58) from
            stateless `exit 1` doubles to a recorded-output ps/top/vm_stat/iostat
            fake, assert rows > 0, and add a PAIR_PERF_BUDGET=2 run asserting
            pipe_hop_ms/fork_exec_ms survive and cpu_idle_pct sheds — which also
            closes BR-25's missing pin (ARCH-MOCK, ARCH-ORDER).
          family: untested-shell-surface
          round: 7
        - id: BR-39
          severity: Important
          title: swap_rate divides by WINDOW even when sample_b was shed and the sleep never ran, reporting a rate over time that did not pass
          detail: |-
            This is the 9th finding in family `failure-reported-as-measurement`. Do
            NOT fix this instance. The rule this round's instances need, stated once:
            when a stage is SHED or FAILS, every value derived from that stage must
            shed with it — the shed has to propagate through the dependency graph, not
            stop at the stage that was skipped. Evidence: perf.sh:153-160 skips
            `sleep "$WINDOW"` under a squeezed budget, but :162-174 unconditionally
            divides the vm_stat counter difference by WINDOW. Reproduced: the full run
            reports pageins_per_s=40.0; `PAIR_PERF_BUDGET=2` reports
            pageins_per_s=3.5 from two reads microseconds apart, rendered
            identically. The enumeration this rule implies, all live: swap_rate
            (here); parse_samples/delta reading a shed sample_b as a real one (BR-35);
            the sample blocks degrading to silence rather than n/a (BR-5's residual);
            and, on the same axis, delta's cpu_pct dividing by the DECLARED window
            while the measured one (`at_s`, emitted per sample) is discarded.
          family: failure-reported-as-measurement
          round: 7
        - id: BR-40
          severity: Important
          title: parse_samples is billed as the perf.sh-to-delta contract but omits window_seconds, which delta needs and which crashes it as a string
          detail: |-
            doctor.lua:164 returns only sample_a and sample_b, while
            `delta(a, b, window)` also needs the window — which lives in the capture
            as the text `window_seconds=2`. M2's caller must therefore re-parse the
            capture with its own pattern, a second untested copy of the contract this
            function exists to own (ARCH-PURPOSE's hand-maintained restatement), and
            if it forwards the string unconverted doctor.lua:80 raises "attempt to
            compare string with number" — verified under nvim -l. That is a crash on
            the primary path, at the moment the operator invoked the tool because the
            machine was already struggling. Fix: return a third `meta` value carrying
            window_seconds and both at_s values as numbers (which also supplies the
            measured window the preceding finding needs), and coerce with
            tonumber(window) at delta's guard so a bad value degrades to the existing
            empty result rather than throwing.
          family: incomplete-parse-contract
          round: 7
      boundary: M1
      blocked: false
    - "n": 8
      timestamp: "2026-09-07T01:02:36-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'The pair- prefix half is overtaken by the subcommand reversal, but plan M2.6 still routes the Lua write through artifactpath, which has no CLI surface (verified: no reference in cmd/pair-go/main.go or dispatcher.go).'
          round: 8
        - id: BR-5
          disposition: not-addressed
          note: Scalar collectors are fixed and verified, but emit_sample's `cputimes || say n/a` is dead (the || sees awk's status, not ps's), so both sample sections still render as silence under denial.
          round: 8
        - id: BR-16
          disposition: not-addressed
          note: probe_line's awk still prints only $1/$2; the sample count in $4 is discarded.
          round: 8
        - id: BR-17
          disposition: not-addressed
          note: perf.sh:130 still truncates at awk $4 and emits comm= as a full executable path.
          round: 8
        - id: BR-18
          disposition: not-addressed
          note: PAIR_PERF_WINDOW/BUDGET/PROBE_RESERVE all flow unvalidated; none documented outside workshop/.
          round: 8
        - id: BR-19
          disposition: not-addressed
          note: No Log entry or plan note records that 4f9365b3 (the M2.2b join) landed inside the M1 window.
          round: 8
        - id: BR-20
          disposition: not-addressed
          note: grep perf doctor/README.md returns nothing.
          round: 8
        - id: BR-21
          disposition: not-addressed
          note: Issue Plan M1 and every plan M1.x checkbox are unticked; the Log has only the 2026-09-06 filing entry, so M1.4 and M1.5 have no recorded evidence.
          round: 8
        - id: BR-25
          disposition: not-addressed
          note: Shed ORDER is fixed and verified at PAIR_PERF_BUDGET=2, but PROBE_RESERVE only gates collector start; probe_line guards on over_budget, so an 8s top stub yields zero probe rows and elapsed_seconds=8 over a 6s budget.
          round: 8
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
          round: 8
        - id: BR-27
          disposition: not-addressed
          note: 'Four ladder shapes remain: the top block, the disk block, probe_line, and the sample blocks that skip collect() entirely.'
          round: 8
        - id: BR-28
          disposition: not-addressed
          note: perf.sh:179 still does the rate arithmetic in awk and divides by WINDOW without a zero guard.
          round: 8
        - id: BR-29
          disposition: not-addressed
          note: 'Reproduced with a vm_stat stub exiting 1: renders `swap=n/a (vm_stat unavailable)`. Same misnaming at _loadavg, which reports "returned nothing" when sysctl failed, because the pipeline''s status is awk''s.'
          round: 8
        - id: BR-30
          disposition: not-addressed
          note: doctor.lua still says "The fixture in nvim/fixtures/" and still omits `unmeasured` from delta's documented return shape.
          round: 8
        - id: BR-31
          disposition: not-addressed
          note: The reuse branch is still etime-only; no `cb < ca` guard at the point of derivation.
          round: 8
        - id: BR-32
          disposition: not-addressed
          note: The fixture still carries no redaction/selection rule; sample() still emits full paths, so the next re-capture commits home-directory paths.
          round: 8
        - id: BR-34
          disposition: not-addressed
          note: 'Reproduced: a `top` stub taking 8s runs to completion and the capture reports elapsed_seconds=8 against budget_seconds=6, which also makes perf_test.sh:36 go red under exactly the conditions the capture exists for.'
          round: 8
        - id: BR-35
          disposition: addressed
          note: 'Mutation-verified: reverting usable() in a scratch copy produced two failures including "a shed sample must not report every process as vanished".'
          round: 8
        - id: BR-37
          disposition: not-addressed
          note: 'Confirmed live: `mktemp -d` fails in this agent shell, so `$repo/.perf-test-stub.$$` is the taken path, and .gitignore has no entry for it.'
          round: 8
        - id: BR-38
          disposition: not-addressed
          note: The fix asserts rows against the ambient system instead of controlling it, so `sh doctor/perf_test.sh` now FAILS ("only 0 sample rows") wherever ps is denied, turning make test red for the reader the file header names. The stateless stubs were not promoted to a recorded-output fake, and in_sample/rows at :85-98 are now dead subshell locals.
          round: 8
        - id: BR-39
          disposition: not-addressed
          note: The swap_rate instance is fixed but pinned by no test; 2 of the 4 enumerated members remain live - the sample blocks still degrade to silence (BR-5 residual), and delta still divides by the DECLARED window while the measured at_s values are parsed away.
          round: 8
        - id: BR-40
          disposition: addressed
          note: 'Mutation-verified twice: removing tonumber(window) crashes the suite, removing the window_seconds parse fails the assertion. The at_s half of the fix sketch is carried by BR-39.'
          round: 8
      findings:
        - id: BR-41
          severity: Minor
          title: pair hoprtt silently ignores unrecognised arguments and runs the 500-sample pipe probe instead
          detail: |-
            This is the 4th finding in family `unguarded-edge-case`. Do NOT fix this
            instance. The rule covering all four: a value read from outside the
            function - an argv token, an env var, a counter difference - is rejected
            at the point it is read when it is not one of the forms the code
            understands; falling through to a default produces a successful-looking
            measurement of something the caller did not ask for. Evidence:
            hoprttcmd.go:148-152 tests args[0] against exactly "-child" and "-spawn",
            so `pair hoprtt --spawn 5 -- x` runs pipeRTT(500) and exits 0 with a
            pipe-hop number labelled as whatever the caller thought it asked for -
            the same defect TestUsageErrorsRatherThanSilentlyMeasuringNothing exists
            to prevent, one token away from the cases it covers. Prevalence 4/4, all
            live: this, BR-31 (a reused pid detected only via etime, so an
            unparseable etime yields a negative rate), BR-28 (WINDOW=0 divides to
            inf), BR-18 (PAIR_PERF_* unvalidated into sleep and awk -v). The
            enumeration to sweep in one pass is those four sites.
          family: unguarded-edge-case
          round: 8
        - id: BR-42
          severity: Minor
          title: Three pure entities shipped in M1 have no row in the plan's Core concepts table
          detail: |-
            This is the 3rd finding in family `traceability`. Do NOT fix this
            instance. The rule: the Core concepts table is the greppable contract the
            next milestone reads, so every entity that ships gets its row in the SAME
            round it lands - a table that lags is a plan claiming a smaller surface
            than the code delivers, which is the same defect as one claiming a
            surface the code lacks (BR-33). Live members: `parse_samples`
            (nvim/doctor.lua, billed in its own docstring as THE perf.sh-to-delta
            contract), `parse_duration`, and `verdict`/`FRAME_MS` - the last of which
            M2.3 describes in prose with a symmetric contract the code has since
            replaced with an asymmetric one. Prevalence 3/3 with BR-21 (the issue's
            M1 row still names cmd/hoprtt, a path that does not exist) and BR-19 (no
            record that 4f9365b3 landed inside the M1 window). The sweep is one
            `## Revisions` entry adding the rows and correcting the two stale
            references.
          family: traceability
          round: 8
      boundary: M1
      blocked: false
    - "n": 9
      timestamp: "2026-09-07T11:05:44-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan M2.6:389 still says "via artifactpath"; that package has no CLI surface Lua can reach. The GO_BINS/pair- prefix half is overtaken by the subcommand reversal.
          round: 9
        - id: BR-5
          disposition: not-addressed
          note: 'Reproduced with ps denied: ''### cputime'' and ''### procs'' render as empty sections; the ''|| say n/a'' arms at perf.sh:146,148 are unreachable because the pipeline exits with awk''s status.'
          round: 9
        - id: BR-16
          disposition: not-addressed
          note: probe_line's awk at perf.sh:227 still prints only $1/$2; the sample count is discarded.
          round: 9
        - id: BR-17
          disposition: not-addressed
          note: sample() at perf.sh:130 still takes $4 of a full comm path; the report carries home-directory paths against the plan's own ARCH-SECURE contract.
          round: 9
        - id: BR-18
          disposition: not-addressed
          note: 'Sharper evidence: PAIR_PERF_WINDOW=0 and =abc both render the entire ''## swap_rate'' section as nothing at all -- no key, no n/a, no reason. No shell injection; quoting is correct.'
          round: 9
        - id: BR-19
          disposition: not-addressed
          note: Still no record that 4f9365b3 landed inside the M1 window.
          round: 9
        - id: BR-20
          disposition: not-addressed
          note: grep -c perf doctor/README.md = 0; it still describes doctor/ as the drift tool only.
          round: 9
        - id: BR-21
          disposition: not-addressed
          note: Issue Plan M1 row unticked and still naming cmd/hoprtt; no 2026-09-07 Log entry, no M1.4/M1.5 evidence.
          round: 9
        - id: BR-25
          disposition: addressed
          note: 'Verified live: PAIR_PERF_BUDGET=2 sheds top, iostat and sample_b while pipe_hop_ms and fork_exec_ms survive. Behaviour correct but pinned by no test; that residual is tracked in issue 210.'
          round: 9
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23's negative character-class arm is unchanged.
          round: 9
        - id: BR-27
          disposition: not-addressed
          note: collect() is still re-implemented at perf.sh:101-120, :185-196, :218-229 and skipped by sample()/cputimes() -- structurally why BR-5's residual survives.
          round: 9
        - id: BR-28
          disposition: not-addressed
          note: perf.sh:179 is still shell arithmetic against the file's own rule 2, and WINDOW=0 now silently erases the whole section instead of printing inf.
          round: 9
        - id: BR-29
          disposition: not-addressed
          note: perf.sh:172-173 still reports a vm_stat that exists but fails as "vm_stat unavailable".
          round: 9
        - id: BR-30
          disposition: not-addressed
          note: doctor.lua:157 still says nvim/fixtures/; doctor.lua:75's return-shape list still omits unmeasured.
          round: 9
        - id: BR-31
          disposition: not-addressed
          note: doctor.lua:95 still gates reuse on etime alone; cb < ca is not rejected at the point of derivation.
          round: 9
        - id: BR-32
          disposition: not-addressed
          note: Fixture still carries 0 /Users/ paths only by the truncation accident; no selection or redaction rule recorded beside it.
          round: 9
        - id: BR-34
          disposition: addressed
          note: Split to issue 210 by explicit operator decision and filed with a Spec, a mechanism and a Done-when. Treated as settled, not re-litigated at this gate.
          round: 9
        - id: BR-37
          disposition: not-addressed
          note: Confirmed the fallback is the LIVE path here -- mktemp -d fails with "Operation not permitted" in this shell -- and .perf-test-stub.* is still absent from .gitignore.
          round: 9
        - id: BR-38
          disposition: not-addressed
          note: 'The >=10-row guard swapped a vacuous pass for a hard failure: sh doctor/perf_test.sh exits 1 here, and test-perf-capture is in the make test chain. The rule (control the environment) is still unapplied.'
          round: 9
        - id: BR-39
          disposition: not-addressed
          note: 'swap_rate is fixed and verified, but 2 of the 4 sites the finding''s own enumeration named are live: the sample blocks'' silence, and delta dividing by the declared window while the measured at_s delta is emitted and discarded.'
          round: 9
        - id: BR-41
          disposition: not-addressed
          note: hoprttcmd.go:148-152 unchanged; an unrecognised argv token still falls through to pipeRTT(500) and exits 0.
          round: 9
        - id: BR-42
          disposition: not-addressed
          note: 'Core concepts table unchanged: pair-hoprtt still points at cmd/pair-hoprtt/main.go, and parse_samples / parse_duration / verdict still have no rows.'
          round: 9
      findings:
        - id: BR-43
          severity: Minor
          title: hoprttcmd.Run takes injected writers but child() reads os.Stdin and writes os.Stdout, and the package is registered as a streaming subcommand with no stdin
          detail: |-
            Run (hoprtt.go:146) accepts stdout/stderr and report() honours them, but
            child() (hoprtt.go:35-45) touches os.Stdin/os.Stdout directly. The
            dispatcher registers hoprtt with Streaming: true (dispatcher.go:63) while
            cmd/pair-go/main.go:91 passes no stdin, so the signature says "reads no
            stdin" and the code contradicts it. This is a newly-introduced internal
            package that downstream M2 work will call, so the surface is worth
            settling now: a future test adding {"-child"} to
            TestUsageErrorsRatherThanSilentlyMeasuringNothing's table would block on
            the real terminal rather than the injected buffer. It works today only
            because the child is always a re-exec'd process. Take a stdin io.Reader
            like every sibling Run, or document that -child is process-level only and
            keep it out of the injected-writer path (ARCH-PURE).
          family: injected-io-seam-bypassed
          round: 9
      boundary: M1
      blocked: false
    - "n": 10
      timestamp: "2026-09-07T13:26:16-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 11
      timestamp: "2026-09-07T13:50:40-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan Revisions record M2.6 landing via pair_data_dir() not artifactpath, and the probe as the `pair hoprtt` subcommand; both mechanisms verified reachable in the tree.
          round: 11
      findings:
        - id: BR-44
          severity: Critical
          title: 'The completion leg still measures nothing — the gate moved from mode to cursor column, and `editor: fast` is still emitted as grounds for exclusion'
          detail: |-
            11th in this family. The rule was stated in round 10 and applied only to the
            precondition that round named. nvim_buf_call runs complete_now in the autocmd
            window at col 0, so path/word/spell_complete all return at `if col == 0 then
            return end` (init.lua:1644, :1817, :1949). Measured under the real init.lua:
            complete_now takes 0.0040 ms and the row records "editor":"fast", while
            doctor/SKILL.md:79 tells the reader that `fast` means the #202 chain ran inside
            a frame and that #201/#203 are excluded on it. Fix the class: assert each leg's
            precondition where it is timed (completion needs a completable token at the
            cursor; redraw needs a real UI), render n/a with the unmet precondition
            otherwise, and add one wiring assertion that the timed chain reached its
            candidate build. Note vim.fn.complete() raises E785 outside Insert mode, so the
            candidate build must be split off the complete() call the way complete_now was
            split off the gate.
          family: failure-reported-as-measurement
          round: 11
        - id: BR-45
          severity: Important
          title: The single-sourced key set stops at the Lua boundary — perf.sh and perf_test.sh restate it, and a producer rename breaks nothing
          detail: |-
            3rd in this family. Rule: one declaration per fact, and where a second language
            cannot import it, a test asserts the two agree. doctor.lua:398-413 collapsed the
            three Lua sites, but perf.sh:198-200 restates the set in shell and
            perf_test.sh:28 restates a different subset. Verified by renaming
            swapins_per_s to swap_in_rate in a scratch copy of doctor/ — perf_test.sh's
            failure count was unchanged, and doctor_test.lua cannot see it because it builds
            its degraded input from HEADLINE_KEYS itself. The prompt would silently drop the
            swap row, which is the C1(b) defect returning. Fix: perf_test.sh runs the real
            perf.sh and asserts every key read out of doctor.HEADLINE_KEYS/PROBE_KEYS via
            `nvim -l` appears in the output.
          family: duplicated-logic
          round: 11
        - id: BR-46
          severity: Important
          title: The ps-content filter and the buffer-changed interleaving are both unexecuted, though the seams to control each now exist
          detail: |-
            5th in this family. BR-38's rule generalised: a test controls the inputs its
            subject reads — subprocess output and event ordering alike. Two live members.
            (1) perf.sh:143-152's basename + control-byte strip has no test; perf_test.sh's
            fake ps only `exit 1`s, so reverting the awk body to `$4` leaves every suite
            green, and doctor/fixtures/perf_capture.txt:82 still carries full paths the
            current producer no longer emits. (2) init.lua:4157-4168's buffer-changed
            branch — the plan's only data-loss path — is unexecuted; the injected runner
            receives cb, so the test can rewrite the buffer before invoking it. Sweep both:
            recorded-output ps fake plus a regenerated fixture, and the two missing
            interleavings in tests/pair-doctor-test.sh.
          family: untested-shell-surface
          round: 11
        - id: BR-47
          severity: Important
          title: SKILL.md and atlas name perf-capture-latest.txt, which the rework stopped writing
          detail: |-
            3rd in this family. Rule: a doc naming a runtime artifact names the one the code
            produces, checked in the window that changes it — enumerable by grepping docs
            for $PAIR_DATA_DIR/<name> against the writers in init.lua. init.lua:4128 now
            writes perf-capture-<epoch>.txt (verified: perf-capture-1788813769.txt), while
            doctor/SKILL.md:74 and atlas/index.md:40 still name the fixed file and SKILL.md
            instructs the agent to open it — the truncated-send recovery path #211 exists
            for.
          family: docs-gate
          round: 11
        - id: BR-48
          severity: Important
          title: time_editor leaks a scratch buffer per invocation while its comment claims a guaranteed delete
          detail: |-
            New family. Rule: a resource acquired inside a function is released on every
            exit path, and a comment claiming teardown is a claim a test should hold.
            init.lua:3999 creates the buffer; nothing deletes it (grep nvim_buf_delete
            returns nothing) and :4001 asserts otherwise. Measured: three :PairDoctor runs
            took the valid-buffer count from 1 to 4. The plan's ARCH-ORDER table has a row
            requiring the teardown, so the plan claims delivered behaviour the code lacks.
          family: unreleased-resource
          round: 11
        - id: BR-49
          severity: Minor
          title: The M2.6 deferrals are recorded only in the plan, which archives at close — issue 210 records neither
          detail: |-
            4th in this family. The plan's Revisions say the missing window-length field and
            the uncapped perf-captures.jsonl are "deferred to #210"; #210 covers only
            time-bounding plus three re-surfaced M1 findings. A deferral must land in the
            artifact that survives the archive.
          family: traceability
          round: 11
      boundary: M2
      blocked: true
    - "n": 12
      timestamp: "2026-09-07T14:17:21-07:00"
      agent: claude
      dispose:
        - id: BR-44
          disposition: not-addressed
          note: Completion leg verified fixed and pinned; the redraw/input legs named in the same rule still have no precondition or proof-of-work.
          round: 12
        - id: BR-45
          disposition: addressed
          note: Verified by renaming swapins_per_s in a scratch copy — perf-key-conformance-test fails exactly one assertion.
          round: 12
        - id: BR-46
          disposition: addressed
          note: 'Both members verified by revert: the awk redaction and the buffer-equality guard each take a suite red.'
          round: 12
        - id: BR-47
          disposition: addressed
          note: SKILL.md:74 and atlas/index.md:40 now name perf-capture-<epoch>.txt; no perf-capture-latest remains in the tree.
          round: 12
        - id: BR-48
          disposition: addressed
          note: Verified by removing the nvim_buf_delete — the buffer-count assertion goes red.
          round: 12
        - id: BR-49
          disposition: not-addressed
          note: Issue 210 is unchanged since b57cd08a (outside this window) and records neither the missing window-length field nor the uncapped perf-captures.jsonl.
          round: 12
      findings:
        - id: BR-50
          severity: Critical
          title: send_generated_prompt discards send_to_agent's failure, so :PairDoctor clears the operator's note on a failed send and reports nothing
          detail: |-
            nvim/draft_send.lua:43 returns `false, phase, '<label> exited N'` when a zellij
            action fails, and nvim/init.lua:794 returns false with 'no attached UI'.
            nvim/submission.lua:66-69 throws all three away and returns true
            unconditionally. In on_capture (nvim/init.lua:4187) the consume branch keys
            off `raw` alone, so a successful capture plus a failed send reaches
            nvim/init.lua:4223 and clears the buffer with no notify — the operator sees an
            emptied draft and concludes it was sent. The trigger is a failing zellij
            action, which is the degraded machine this capture exists for. The plan (M2.4),
            the code comment at :4218-4222, and plan Revisions #8 all claim "consume only
            on a successful send"; none of the three is implemented, and the repo's own
            draft path already states and honours the rule at nvim/init.lua:1519-1521
            ("Durability is the submission gate … any failure leaves all authored state
            intact"). Same underlying rule as BR-2 in `failure-reported-as-measurement` — a
            discarded error becomes a fabricated success — but in the send domain rather
            than the measurement domain, and here the fabricated success authorises a
            destructive step. Fix: return send_low_level's ok from send_generated_prompt
            (its two other callers ignore the result, so this is additive), gate the
            consume on it, notify on failure, and drive the interleaving with a
            _G.PairTestSendToAgent stub returning false.
          family: discarded-failure-signal
          round: 12
        - id: BR-51
          severity: Important
          title: README.md and doctor/README.md still describe :PairDoctor as a drift-only pointer and never mention that it now consumes the draft buffer
          detail: |-
            This is the 4th finding in family `docs-gate`, so the deliverable is the rule
            and its enumeration, not these two files. Rule: a change to a user-facing
            surface updates every doc that describes that surface, and the enumeration is
            mechanical — `grep -rl '<surface>' --include='*.md'` minus
            workshop/{history,plans,issues}. For `:PairDoctor` that is README.md,
            doctor/README.md, doctor/SKILL.md, atlas/index.md; the last two were updated
            and the first two were not. README.md:605 still says it reads "the session's
            adaptation flight recorder"; doctor/README.md:63-69 still says it hands the
            agent "an instruction to run doctor.sh", and doctor/ 's own README never
            mentions perf.sh. Neither states the user-visible behaviour change that the
            draft buffer is consumed as the note. Measured prevalence of the class: three
            members survived to this round — BR-20 (this same doctor/README.md gap,
            disposed not-addressed twice then demoted), BR-30 (nvim/doctor.lua:180 still
            says the fixture is in nvim/fixtures/ when it is doctor/fixtures/, and :75-76
            still omits `unmeasured` from delta's documented return shape), and this. Each
            round fixed the site named instead of writing the enumeration, which is why the
            class keeps returning.
          family: docs-gate
          round: 12
      boundary: M2
      blocked: true
    - "n": 13
      timestamp: "2026-09-07T14:38:33-07:00"
      agent: claude
      dispose:
        - id: BR-44
          disposition: addressed
          note: 'Mutation-verified: removing the cursor placement turns the work-counter assertion red; live render shows completion 0.8ms, matching #202''s benchmark. Redraw''s precondition re-raised separately as the unswept member.'
          round: 13
        - id: BR-49
          disposition: not-addressed
          note: 'No commit in the window touched issue 210; it still records only time-bounding plus the three M1 findings, and #208''s Log names only "no stage is time-bounded".'
          round: 13
        - id: BR-50
          disposition: addressed
          note: Production behaviour fixed and the init.lua gate is mutation-pinned; the submission.lua half is unpinned, rolled into the new coverage finding.
          round: 13
        - id: BR-51
          disposition: addressed
          note: Enumeration swept and verified — grep over *.md leaves only CHANGELOG (release-scoped) and gitignored runtimebundle copies, which are in sync.
          round: 13
      findings:
        - id: BR-52
          severity: Important
          title: Every test drives the send as failing, so the consume-on-success clear, the wired join, and submission.lua's real-result return are all unexecuted
          detail: |-
            This is the 6th finding in family `untested-shell-surface` — first in Lua rather
            than shell, same rule. The rule that covers all six: a new surface is covered
            only when EVERY reachable branch is executed; a suite that drives one side of a
            gate reports coverage it does not have. Enumeration for this diff, and it is
            mechanical — list the branches on_capture introduces and check each. Unexecuted:
            (1) the clear at nvim/init.lua:4229, because send_to_agent returns false with
            'no attached UI' (nvim/init.lua:741) in headless, so tests 1-4 never reach it and
            test 5's stub is redundant for its own premise; (2) the join at
            nvim/init.lua:4180-4184, because the fixture capture.txt has no `## sample_a`
            section, so `if a and b then` is never entered — this is the same wiring a prior
            round found dead in production; (3) submission.lua:75 — reverting it to
            `return true` leaves pair-doctor-test.sh AND submission_test.lua green
            (mutation-verified). Also unexecuted: the doctor-load failure, PAIR_HOME unset,
            the `capture is already running` guard, perf.sh-not-readable, sidecar-write
            failure, and both n/a notes on the completion leg. Fix the class: set
            _G.PairTestZellijExecutor for one case and assert the buffer IS cleared, give the
            fixture real sample blocks, and add a failing-send_low_level case to
            submission_test.lua.
          family: untested-shell-surface
          round: 13
        - id: BR-53
          severity: Important
          title: The redraw leg is still timed with no precondition asserted — the second member of the enumeration round 11 named
          detail: |-
            12th in this family. Round 11 stated the class and named both members
            ("completion needs a completable token at the cursor; redraw needs a real UI");
            only the completion member was swept. nvim/init.lua:4051-4053 times
            vim.cmd('redraw') unconditionally and feeds the result to a verdict that
            doctor/SKILL.md tells the reader to exclude pair#201/#203 on. Not reachable in
            production today — :PairDoctor always runs with a TUI attached — which is why
            this is Important rather than Critical; it is a regression guard on the last row
            of the enumeration. has_ui() already exists at nvim/init.lua:689, so the fix
            reuses it rather than adding a second UI check (ARCH-DRY).
          family: failure-reported-as-measurement
          round: 13
        - id: BR-54
          severity: Important
          title: The operator's note is written only into the prompt, and the buffer is cleared on send
          detail: |-
            nvim/init.lua:4189-4193 writes compact + rates + raw to the sidecar and omits the
            note; verified against a produced sidecar file. The buffer is then cleared on a
            successful send, and perf-captures.jsonl — the only other copy, and itself
            pcall'd and silent — is never named in the payload. The note is also uncapped, so
            a pasted log produces an arbitrarily long prompt whose middle is precisely the
            region pair#211 measured as dropped (1,025 bytes gone from a 2,447-byte send).
            This contradicts the invariant this milestone recorded in atlas/index.md and
            workshop/lessons.md: nothing of value exists only in the prompt — applied to the
            report but not to the operator's own input, which the same docs call the one
            thing that cannot be re-measured. Fix: prepend the note to the sidecar body.
          family: sole-copy-on-lossy-channel
          round: 13
        - id: BR-55
          severity: Minor
          title: The rolling-log append discards pair_write_data_file's nil return, so the comparative series can stop accumulating silently
          detail: |-
            2nd in family. The rule: a function whose contract is "returns whether the effect
            happened" must have that signal consumed, or surfaced to the operator where the
            caller cannot act on it. Enumeration over the new code: the sidecar write consumes
            it (payload degrades), send_generated_prompt now consumes it, the JSONL write at
            nvim/init.lua:4200 does not. M2.6's whole purpose is making the next investigation
            comparative; a permanently failing append is invisible.
          family: discarded-failure-signal
          round: 13
        - id: BR-56
          severity: Minor
          title: The JSONL row has no schema version and encodes an empty probes table as [] rather than {}
          detail: |-
            2nd in family. Verified on a real row: {"probes":[],...}. A degraded capture
            therefore changes the type of `probes` from object to array, breaking any external
            reader doing .probes.pipe_hop_ms — on a file whose stated purpose is to stay
            legible years later. The same row carries no version field.
          family: incomplete-parse-contract
          round: 13
        - id: BR-57
          severity: Minor
          title: swap_na / disk_na / probes_na are three copies of one loop differing only in the key list
          detail: |-
            4th in family. doctor/perf.sh:203-205. One `na_for <reason> <key>...` helper covers
            all three (ARCH-DRY).
          family: duplicated-logic
          round: 13
        - id: BR-58
          severity: Minor
          title: time_editor's synthetic TextChangedI mutates the live completion debounce state
          detail: |-
            5th in family. nvim/init.lua:4046 fires the real autocmd, which cancels any
            pending completion timer and resets complete_last_fire — so invoking the
            diagnostic can drop a popup the operator was about to get. The comment above
            time_editor argues it cannot touch the draft because it uses a scratch buffer;
            that holds for buffer text but not for the shared debounce state.
          family: unguarded-edge-case
          round: 13
      boundary: M2
      blocked: false
    - "n": 14
      timestamp: "2026-09-07T14:54:43-07:00"
      agent: claude
      dispose:
        - id: BR-5
          disposition: addressed
          note: Verified live with ps/sysctl/top/iostat all denied — every key renders n/a with a reason; no fabricated process_count=0, no bare load=.
          round: 14
        - id: BR-16
          disposition: not-addressed
          note: probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
          round: 14
        - id: BR-17
          disposition: addressed
          note: sample() strips the first three fields and basenames on '/'; perf_test.sh:164 pins "Google Chrome" surviving and :160 pins no /Applications/ reaching the report.
          round: 14
        - id: BR-18
          disposition: not-addressed
          note: No validation and no PAIR_PERF mention in README/atlas/SKILL; measured — WINDOW=0 makes the whole swap_rate section vanish with awk "division by zero".
          round: 14
        - id: BR-19
          disposition: not-addressed
          note: grep for 4f9365b3 across workshop/ and atlas/ hits only prior gate-ledger rounds; neither the issue nor the plan records it.
          round: 14
        - id: BR-20
          disposition: addressed
          note: doctor/README.md:70-88 now links perf.sh and documents the note/clear semantics.
          round: 14
        - id: BR-21
          disposition: addressed
          note: Issue Plan M1 is ticked with the design-reversal note; the M1 close line carries evidence and the zellij number (14.385 ms) is in the M2 baseline table.
          round: 14
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
          round: 14
        - id: BR-27
          disposition: not-addressed
          note: na_for deduped only the key lists; the top block, disk block, probe_line and emit_sample each still re-implement the availability/failure/empty ladder.
          round: 14
        - id: BR-28
          disposition: not-addressed
          note: Sharper than reported — at WINDOW=0 awk aborts and all three swap keys VANISH rather than yielding inf, and perf_test.sh:128/:156 drive that exact value.
          round: 14
        - id: BR-29
          disposition: not-addressed
          note: perf.sh:216-217 still says "vm_stat unavailable" when the first read fails; only the second read (:221) distinguishes failure from absence.
          round: 14
        - id: BR-30
          disposition: addressed
          note: doctor.lua:180 now names doctor/fixtures/ and :75-76 enumerates unmeasured.
          round: 14
        - id: BR-31
          disposition: not-addressed
          note: doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
          round: 14
        - id: BR-32
          disposition: addressed
          note: The finding's own alternative was taken — sample() basenames comm at the emitter, so no path can reach a re-capture; fixture verified free of /Users/ and /home/.
          round: 14
        - id: BR-37
          disposition: not-addressed
          note: perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry.
          round: 14
        - id: BR-38
          disposition: not-addressed
          note: Reproduced — sh doctor/perf_test.sh exits 1 in this sandboxed shell (grammar violation on perf.sh's own n/a line, plus "only 2 sample rows"), and test-perf-capture is in make test.
          round: 14
        - id: BR-39
          disposition: not-addressed
          note: The shed member is fixed (WINDOW_ELAPSED). The declared-vs-measured member is live — at_s is discarded by parse_samples, and SWAP_A is read before sample_a while _b is read after sample_b.
          round: 14
        - id: BR-41
          disposition: not-addressed
          note: hoprtt.go:148-177 unchanged; an unrecognised argument still falls through to pipeRTT(500).
          round: 14
        - id: BR-42
          disposition: addressed
          note: Plan Revisions item 3 enumerates the missing entities and a separate entry reverses every cmd/pair-hoprtt reference; the stale TABLE itself is carried as a plan-revision recommendation rather than re-raised.
          round: 14
        - id: BR-43
          disposition: not-addressed
          note: hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming with no stdin from main.go.
          round: 14
        - id: BR-49
          disposition: not-addressed
          note: Issue 210 now records BR-34/25/38/39 but still not the missing window-length field or the uncapped perf-captures.jsonl.
          round: 14
        - id: BR-52
          disposition: addressed
          note: All three prescribed members mutation-verified red — the clear, the join, and submission.lua's return. Residual unexecuted branches roll into the new coverage finding.
          round: 14
        - id: BR-53
          disposition: not-addressed
          note: Behaviour is correct and reachable (payload renders redraw n/a, verdict unknown), but removing the has_ui() gate in a scratch copy leaves every suite green — and vim.g.pair_test_has_ui already exists as the seam.
          round: 14
        - id: BR-54
          disposition: not-addressed
          note: The note does reach the sidecar, but no test opens the sidecar — reverting the prepend leaves pair-doctor-test.sh and doctor_test.lua both green.
          round: 14
        - id: BR-55
          disposition: not-addressed
          note: The notify is present but no test drives a failing pair_write_data_file, so the branch is never executed.
          round: 14
        - id: BR-56
          disposition: addressed
          note: Mutation-verified — deleting the vim.empty_dict line takes doctor_test.lua red on "an empty probes set encodes as an object, not []".
          round: 14
        - id: BR-57
          disposition: addressed
          note: na_for at perf.sh:206 collapses the three loops; the n/a key assertions under tool denial still pass.
          round: 14
        - id: BR-58
          disposition: addressed
          note: Answered as an explicit accepted trade-off with its reasoning recorded at nvim/init.lua:4048-4053, which is a legitimate resolution for this severity.
          round: 14
      findings:
        - id: BR-59
          severity: Important
          title: Three of the closing commit's six behaviour changes are unpinned — the same rule the commit closed BR-52 on
          detail: |-
            This is the 7th finding in family `untested-shell-surface`. Do NOT fix these three
            instances. The rule covering all seven: a behaviour change lands with a test that
            fails without it, and the check is mechanical — revert the hunk in a scratch copy
            and confirm a suite goes red. The enumeration is the closing commit's own
            behaviour hunks, and I ran it. Pinned (verified red on revert): capture_record's
            schema + empty_dict, the consume-on-success clear, the parse_samples->delta join,
            submission.lua's real return. Unpinned (verified green on revert): the has_ui()
            redraw gate at nvim/init.lua:4062-4069, the operator note prepended to the sidecar
            at nvim/init.lua:4209-4211, and the JSONL-append failure notify at
            nvim/init.lua:4232-4239. The seams to close all three already exist —
            vim.g.pair_test_has_ui (nvim/init.lua:689), PAIR_DATA_DIR in the test harness, and
            the injected capture runner. Prevalence 7/7 with BR-9, BR-25, BR-26, BR-36, BR-38,
            BR-52; the commit that closed BR-52 is itself the newest member, which is the
            signal that the rule was fixed at the instance level and never as a class.
          family: untested-shell-surface
          round: 14
        - id: BR-60
          severity: Minor
          title: doctor.lua and doctor_test.lua still claim "no vim API", and atlas's sidecar enumeration omits the note added in the same commit
          detail: |-
            This is the 5th finding in family `docs-gate`. Do NOT fix these instances — the
            rule: a comment or doc that ENUMERATES something is a claim a grep can check, and
            it is updated in the same change that invalidates it; the enumeration is
            mechanical (grep the changed symbol across *.md and the module headers that
            describe it). Live members: nvim/doctor.lua:2 and nvim/doctor_test.lua:2 both say
            "no vim API here" while capture_record now calls vim.empty_dict and the new test
            calls vim.json; atlas/index.md describes the sidecar as "compact half, joined
            per-process rates, and both raw ps samples" while the same commit prepended the
            operator note to it — under a paragraph whose own subject is the invariant that
            nothing of value exists only in the prompt. Prevalence 5/5 with BR-20, BR-30,
            BR-47, BR-51.
          family: docs-gate
          round: 14
      blocked: true
    - "n": 15
      timestamp: "2026-09-07T15:18:04-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: not-addressed
          note: probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
          round: 15
        - id: BR-18
          disposition: not-addressed
          note: No validation; grep for PAIR_PERF across README.md, atlas/, doctor/README.md and doctor/SKILL.md returns nothing.
          round: 15
        - id: BR-19
          disposition: not-addressed
          note: grep 4f9365b3 across workshop/ and atlas/ still hits only the gate ledger's own rounds.
          round: 15
        - id: BR-26
          disposition: not-addressed
          note: perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
          round: 15
        - id: BR-27
          disposition: not-addressed
          note: The top block, disk block, probe_line and emit_sample each still re-implement collect()'s availability/failure/empty ladder.
          round: 15
        - id: BR-28
          disposition: not-addressed
          note: 'Reproduced at HEAD - PAIR_PERF_WINDOW=0 prints "awk: division by zero" and all three swap keys vanish; swapins_per_s is a HEADLINE_KEY, so a headline row is dropped, not just a value lost.'
          round: 15
        - id: BR-29
          disposition: not-addressed
          note: perf.sh:216-217 still says "vm_stat unavailable" when the first read fails.
          round: 15
        - id: BR-31
          disposition: not-addressed
          note: doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
          round: 15
        - id: BR-37
          disposition: not-addressed
          note: perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry; note the three newer fakes (:98, :141, :165) use $TMPDIR with no worktree fallback, so the file is now inconsistent with itself.
          round: 15
        - id: BR-38
          disposition: addressed
          note: Mutation-verified - the controlled ps at perf_test.sh:98-111 validates 12 rows in this shell where /bin/ps is denied, and a tab-to-pipe change in sample() takes it red.
          round: 15
        - id: BR-39
          disposition: not-addressed
          note: The shed member stays fixed. The declared-vs-measured member is live - parse_samples reads window_seconds and discards at_s, so delta divides by the DECLARED window on exactly the machine where sleep 2 does not take 2s.
          round: 15
        - id: BR-41
          disposition: not-addressed
          note: hoprtt.go:146-152 unchanged; an unrecognised argument still falls through to pipeRTT(500) and exits 0.
          round: 15
        - id: BR-43
          disposition: not-addressed
          note: hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming and main.go:90 passes no stdin.
          round: 15
        - id: BR-49
          disposition: not-addressed
          note: Re-read issue 210 at HEAD - it records BR-34/25/38/39 and still neither the missing window-length field nor the uncapped perf-captures.jsonl.
          round: 15
        - id: BR-53
          disposition: addressed
          note: Mutation-verified - replacing has_ui() with `true` at nvim/init.lua:4060 takes pair-doctor-test.sh red on both the n/a render and the unknown verdict.
          round: 15
        - id: BR-54
          disposition: addressed
          note: Mutation-verified - dropping the note prefix at nvim/init.lua:4210 takes the two sidecar-content assertions red; the test opens the file the payload names.
          round: 15
        - id: BR-55
          disposition: addressed
          note: Mutation-verified - replacing the `if not pair_write_data_file(...)` guard with a bare call takes the notify assertion red.
          round: 15
        - id: BR-59
          disposition: addressed
          note: All three members reverted independently in a scratch tree; each took pair-doctor-test.sh red (2, 2 and 1 failures). The rule was applied as a class, not per-site.
          round: 15
        - id: BR-60
          disposition: not-addressed
          note: nvim/doctor.lua:2 and nvim/doctor_test.lua:3 still claim no vim API while :293 calls vim.empty_dict; atlas/index.md:39 still enumerates the sidecar without the note, and doctor/SKILL.md's "compact report, joined per-process rates, and both raw ps samples" is a third member of the same enumeration.
          round: 15
      findings:
        - id: BR-61
          severity: Important
          title: Four of perf.sh's five external-tool parsers are pinned only by exit-1 stubs, so a wrong field index ships as a plausible headline reading
          detail: |-
            This is the 8th finding in family `untested-shell-surface`. Do NOT fix this
            instance. The rule covering it: a shell parser over external-tool output is
            pinned by a RECORDED-OUTPUT fake of that tool - an `exit 1` stub pins the n/a
            ladder and asserts nothing about field extraction. The enumeration is
            mechanical, every awk in perf.sh that indexes a tool's stdout, and it has five
            members: sample()'s procs row (perf.sh:150, pinned at perf_test.sh:98 and
            :165), cputimes()'s row (perf.sh:162, unpinned), top's CPU-usage line
            (perf.sh:115, unpinned), top's WindowServer row (perf.sh:117, unpinned), and
            iostat's three columns (perf.sh:238, unpinned). Mutation-verified
            independently, full suite green after each: changing cputimes()'s tab to a
            space moves the join from rates=5/unmeasured=0 to rates=0/unmeasured=5, so
            SKILL.md's step 4 and the operator's own Spec addition silently vanish;
            $(NF-1)->$2 makes cpu_idle_pct carry the USER percentage under the idle key, a
            HEADLINE_KEY the prompt carries and SKILL.md instructs the reader on; permuting
            iostat's $1,$2,$3 mislabels all three disk values. The mechanism already exists
            at perf_test.sh:98-111 - the sweep is a recorded-output fake for top, iostat
            and vm_stat plus cputime-row assertions on the existing ps fake (ARCH-MOCK,
            ARCH-PURPOSE). Prevalence 8/8 with BR-9, BR-25, BR-26, BR-36, BR-38, BR-52,
            BR-59.
          family: untested-shell-surface
          round: 15
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

## Round 4 — 2026-09-06T23:55:44-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Rename to pair-hoprtt landed and GO_BINS is correct; plan M2.6 line 387-391 still routes the Lua rolling-file write through artifactpath.
- BR-2 — addressed — Verified live, not from the diff: zellij probe now renders n/a (probe failed) instead of a fabricated 10.183; partial failure emits the 5-field line. No regression test pins it — folded into BR-9.
- BR-3 — addressed — Revert-verified red (unmeasured counted, not dropped). Residual: perf.sh still runs two separate ps passes (:84,:87), so storm pids land in unmeasured rather than being measured; one ps -Ao pid=,etime=,time=,rss=,comm= pass removes the bucket.
- BR-4 — addressed — pair-hoprtt is untracked at HEAD (removed in 0a7e6e27). The 2.9MB blob remains in c290191c; the branch is still unpushed, so dropping it by rebase is cheap now and permanent after merge.
- BR-5 — not-addressed — collect() is defined at perf.sh:32 with ZERO call sites. Re-ran the capture: load=, cpu_idle_pct=, process_count=0, host_cores=?, empty ## disk block. The fix reads as protection while doing nothing.
- BR-6 — addressed — Revert-verified: re-adding `|| echo 0` to count_comm makes perf_test.sh fail with "report contains a bare value line with no key".
- BR-7 — addressed — count_comm now takes the basename via awk -F/ and anchors the pattern. Not executable here (ps denied), reasoned from the comm= path format the fix itself documents.
- BR-8 — addressed — Revert-verified red. Residual (Minor, not re-raised): no field-count or sign bound, so 1:2:3:4:5 -> 13403045, 0x10 -> 16, -1:00 -> -60.
- BR-9 — not-addressed — perf_test.sh + make test-perf-capture exist and are wired into make test, but pin only line shape and the missing-binary path; nothing pins a failing probe or a failing collector, and delta is still literals rather than recorded fixture pairs.
- BR-10 — not-addressed — explicitAssetPaths, both artifactpath lists, GO_BINS and the recipe are done and doctor/perf.sh is in the generated manifest. embed_test.go:25-26 was named and skipped, and the pair-hoprtt bundling decision was never made - see the new Critical.
- BR-11 — not-addressed — over_budget() guards only probe_line (:142). top -l 2, sleep WINDOW and iostat -c 2 are unconditional and are the budget; the total is still unbounded and the skip drops the probes rather than the expensive collectors.
- BR-12 — not-addressed — perf.sh:97,107 still emit at_ns from date +%s.
- BR-13 — not-addressed — vm_stat awk pipeline still duplicated verbatim at perf.sh:93 and :119.
- BR-14 — not-addressed — perf.sh:23 still yields /bin/pair-hoprtt when PAIR_HOME is unset; :161's reason still says "not built".
- BR-15 — not-addressed — summary (main.go:41-45) still indexes an empty slice with no guard.
- BR-16 — not-addressed — probe_line's awk still discards $4 on the success path.
- BR-17 — not-addressed — sample() at perf.sh:85 still uses awk $4; comm= still emits full paths.
- BR-18 — not-addressed — Now wider - PAIR_PERF_BUDGET joins PAIR_PERF_WINDOW, both unvalidated into sleep/awk -v/$(( )) and undocumented in atlas and README.
- BR-19 — not-addressed — 4f9365b3 is still in the window, and verdict/note_from_lines landed with it; no ## Log entry records the boundary.
- BR-20 — not-addressed — doctor/README.md still does not mention perf.sh.
- BR-21 — not-addressed — Issue Plan M1 (line 133) unticked; ## Log has no M1.4 or M1.5 evidence.

### Raised

- **BR-22** [Critical] `unverified-repo-mechanism` perf.sh looks for its probe at $PAIR_HOME/bin/pair-hoprtt, a path that never exists in a shipped pair
  3rd finding in this family, so the deliverable is the RULE, not the site.
  Rule: a plan or diff may name a repo path or mechanism only alongside a
  check that it resolves in every layout the artifact ships to (dev
  checkout, make install, extracted runtime bundle) AND from the language
  and layer the calling code sits in. Prevalence 3/3 rounds that named a
  mechanism: PQ-1 failed the build path, BR-1 failed the language path,
  this fails the install-layout path. Evidence: the generated
  manifest.json's bin/ entries are exactly bin/lib/adapt-log.sh,
  bin/lib/dev-rebuild.sh, bin/pair-help, bin/pair-notify - no Go binaries
  since 104 M3. This window added doctor/perf.sh to the bundle, so the
  shell ships and its payload does not; every installed pair prints
  probes=n/a and is told to run make build, which does not apply to its
  layout. Sweep the enumeration the rule implies for every new runtime
  file: builds, in the bundle manifest, callers' path expressions resolve
  under all three layouts, a test pins each.
- **BR-23** [Important] `failure-reported-as-measurement` verdict() turns absent timings into 'fast', and its test asserts that direction
  6th finding in this family, so state the rule rather than patching the
  site. Rule: every value-producing function in the capture, shell and Lua
  alike, must render an absent or failed measurement as absence (nil, or
  n/a with a reason), never as an in-domain value. Prevalence 6/6: BR-2,
  BR-5, BR-14, BR-16, the dead collect() helper, and this. At
  doctor.lua:129 `tonumber(x) or 0` makes verdict(nil,nil)=='fast', and per
  the Spec (issue lines 77-81) 'fast' is what excludes the whole scheduling
  family (201/203) - so a report where nvim was never timed excludes two
  issues on a measurement that did not happen. doctor.lua:118-124 states
  the correct rule three functions above. doctor_test.lua:52 asserts the
  wrong direction, which is the same-mental-model test the gate warns
  about. Enumeration to sweep in one pass: every kv call site in perf.sh,
  every exported function in doctor.lua's 208 block, and report/summary in
  cmd/pair-hoprtt/main.go - each gets one absent-input case.

## Round 5 — 2026-09-07T00:17:26-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — The naming half landed pre-window (df283abe); the plan file was not touched in this window and M2.6:389 still names artifactpath, which has no CLI surface — every caller is Go under cmd/.
- BR-5 — not-addressed — The kv collectors are fixed and revert-verified, but sample()/cputimes() (perf.sh:121-122) bypass collect() and their `|| say "n/a (ps unavailable)"` at :137/:139 is dead — the pipeline exits with awk's status, so a failed ps yields silently empty blocks.
- BR-9 — addressed — perf_test.sh + test-perf-capture wired into `make test`, and parse_samples now pins the perf.sh -> delta contract against a real recorded capture.
- BR-10 — addressed — generate.go, embed_test.go and the artifactpath manifest all carry doctor/perf.sh; the regenerated manifest.json contains it and the bundle tests pass.
- BR-11 — addressed — A stage-level deadline with skip reasons and an emitted elapsed_seconds now exists; the shed ORDER it produces is raised separately.
- BR-12 — addressed
- BR-13 — addressed
- BR-14 — addressed
- BR-15 — not-addressed — cmd/pair-hoprtt/main.go:41 summary() still indexes samples[0] with no length guard; still unreachable from main, still one line from being safe.
- BR-16 — not-addressed — probe_line's awk (perf.sh:206) still prints only $1 and $2; the sample count $4 is discarded, so a pipeRTT that broke out early reads identically to a full 500-sample run.
- BR-17 — not-addressed — perf.sh:121 still takes awk $4 of `comm=`, truncating any executable path containing a space, and still emits full paths where the plan said process name.
- BR-18 — not-addressed — PAIR_PERF_WINDOW and PAIR_PERF_BUDGET still flow unvalidated into sleep, `[ -ge ]` and `awk -v`, and neither appears in atlas/index.md or doctor/README.md.
- BR-19 — not-addressed — 4f9365b3 is still inside the M1 window along with verdict/note_from_lines; no Log entry or plan note records that M2's base is not the branch point.
- BR-20 — not-addressed — doctor/README.md is not in this window's diff and still describes only the drift half; the atlas entry landed but the README a reader of doctor/ opens did not.
- BR-21 — not-addressed — Plan M1 is still unticked and the Log still ends at 2026-09-06; the M1.4/M1.5 numbers now exist in the fixture (pipe 0.006ms, fork 1.588ms, zellij 13.323ms, elapsed 5) but are not recorded in the issue.
- BR-22 — not-addressed — PATH-first fixes `make install` only. ../homebrew-pair/Formula/pair.rb builds solely ./cmd/pair-go and installs bin/nvim/zellij, and PAIR_HOME at runtime is the extracted bundle root, which carries no helper binaries — so a shipped pair still prints probes=n/a with advice that does not apply to its layout. I reverted the PATH-first block in a scratch copy and the whole suite stayed green, so no test pins the fix.
- BR-23 — not-addressed — verdict(nil,nil) is fixed, but verdict(nil, 2) still returns 'fast' and doctor_test.lua:56 asserts that direction — a half-absent measurement rendered as a full in-domain verdict is the same defect at a different arity. Of the enumeration BR-23 named, only doctor.lua's exports were swept: perf.sh's sample blocks still fabricate absence (see BR-5), parse_samples has no absent-input case, and summary/report in cmd/pair-hoprtt still panic on empty input.

### Raised

- **BR-24** [Critical] `unverified-repo-mechanism` doctor/perf_test.sh:51's bare `mktemp -d` makes `make test-perf-capture`, and therefore `make test`, fail in a sandboxed agent shell
  This is the 4th finding in family `unverified-repo-mechanism`, so the rule
  is the deliverable, not the site. Rule: a mechanism this repo names must be
  checked in EVERY environment and layout it will run in, enumerated in one
  place — dev checkout, `make install` PATH, extracted runtime bundle,
  Homebrew formula install, and the sandboxed agent shell that runs `make
  test` before every close. Prevalence 4/4 rounds that named a mechanism:
  PQ-1 failed the build path, BR-1 the language path, BR-22 the install-layout
  path, this the execution-environment path.
  Reproduced: `make test-perf-capture` exits 1 with `mktemp: mkdtemp failed
  on /var/folders/...: Operation not permitted`, before any assertion runs,
  because macOS mktemp with no template reads confstr(_CS_DARWIN_USER_TEMP_DIR)
  and cannot be redirected with TMPDIR. Verified fix and green suite with
  `mktemp -d "${TMPDIR:-/tmp}/perfstub.XXXXXX"`. Note this is the same file
  whose header advertises that it is "routinely run by sandboxed agents".
- **BR-25** [Important] `unenforced-operating-envelope` The budget sheds the three probes first, so a degraded machine — the only condition this tool targets — yields a capture with no probe rows
  This is the 2nd finding in family `unenforced-operating-envelope`. Rule,
  not site: a capture's budget must reserve capacity for the measurements the
  issue exists to take and shed contextual collectors before diagnostic ones;
  sequential position in the script is not a priority order. Reproduced with
  `PAIR_PERF_BUDGET=2 sh doctor/perf.sh`: disk, pipe_hop, fork_exec and
  zellij_action all render `n/a (budget exceeded...)` while `top -l 2 -n 60`
  — the single most expensive collector, and unbounded on a loaded host — is
  checked before it starts and therefore always runs. The healthy fixture
  already reports elapsed_seconds=5 of 6. The probes are the only rows with
  published baselines and are Done-when item 1. Fix: order the probes ahead
  of top/iostat, or reserve a probe slice of the budget. Secondary: this makes
  perf_test.sh:36's `[ elapsed -le budget ]` load-dependent, so `make test`
  goes red under exactly the conditions the capture is for.
- **BR-26** [Minor] `untested-shell-surface` perf_test.sh:23's stray-line check only fires for all-digit lines, so any other unattributable line passes
  The `*[!0-9]*) continue` arm (there to skip tab-separated ps rows) swallows
  every stray line containing a non-digit. Verified: injecting `say "an
  unattributable stray line"` into a scratch perf.sh left the suite green; a
  bare `0` is correctly caught. Match the ps rows positively (a line with a
  tab, inside a `### ` block) instead of negatively by character class.
- **BR-27** [Minor] `duplicated-logic` Four shapes of collect()'s availability/failure/empty ladder in one file, which is how the sample blocks escaped the rule
  collect() (perf.sh:43) exists to be that ladder, but it can only emit one
  key, so the top block (~:88), the disk block (:171) and probe_line (:202)
  each re-implement it and sample()/cputimes() (:121-122) skip it entirely.
  A `stage()` variant that takes a multi-key emitter would collapse all four
  and is the structural fix behind the BR-5 residual.
- **BR-28** [Minor] `report-line-contract` perf.sh:163's swap_rate is shell arithmetic, contradicting the file's own rule 2, and divides by WINDOW unguarded
  This is the 5th finding in family `report-line-contract`. Rule: every value
  the report emits either comes from a tool verbatim or is computed in tested
  Lua; perf.sh's header rule 2 (line 13) states this and the swap block
  violates it. Same block yields inf at WINDOW=0 and negative rates if a
  counter goes backwards. Either move it into doctor.delta with the pid join
  or amend rule 2 to say "no per-process arithmetic".
- **BR-29** [Minor] `failure-reported-as-measurement` A vm_stat that exists but exits non-zero renders `swap=n/a (vm_stat unavailable)`, misnaming the failure
  This is the 7th finding in family `failure-reported-as-measurement`. The
  rule the family needs, stated once: every emitter distinguishes THREE
  outcomes — tool absent, tool failed, tool returned nothing — and names which
  one it hit, because "unavailable" sends a reader to install something that
  is already installed. collect() already does this correctly; the hand-rolled
  swap block at perf.sh:155-157 does not. Reproduced with a `vm_stat` stub
  that exits 1.
- **BR-30** [Minor] `docs-gate` Comment drift in nvim/doctor.lua — the fixture path and delta's documented return shape are both wrong
  This is the 2nd finding in family `docs-gate`. Rule: a comment that names a
  path or a return shape is a claim a test can check; when it names a
  filesystem path, assert it, and when it enumerates fields, enumerate all of
  them. doctor.lua:157 says "The fixture in nvim/fixtures/" (it is in
  doctor/fixtures/, and doctor_test.lua explains why it must not be under
  nvim/); doctor.lua:75's "Returns { rates, vanished, started, reused, rows_a,
  rows_b }" omits `unmeasured`, which the code sets and the test asserts.
- **BR-31** [Minor] `unguarded-edge-case` delta detects a reused pid only through etime, so an unparseable etime lets a reused pid produce a negative cpu_pct
  This is the 3rd finding in family `unguarded-edge-case`. Rule: a derived
  value with a known-impossible range must be rejected at the point of
  derivation, not only by the proxy signal that usually implies it. At
  doctor.lua:95, if pa.etime or pb.etime is nil the reuse branch is skipped
  and the pid falls into the rate branch, where cb < ca yields a negative
  rate. `cb < ca` is itself sufficient evidence of reuse and is cheaper than
  the etime comparison; the fixture test's `r.cpu_pct >= 0` assertion would
  then be enforced by the code rather than by the fixture's luck.
- **BR-32** [Minor] `recorded-fixture-redaction` doctor/fixtures/perf_capture.txt is a real host capture in a public repo, safe only by an unrecorded truncation accident
  The fixture carries zero `/Users/` paths only because it was cut to the
  first 60 pids, which on this host are all system daemons. Nothing records
  that as the selection rule, and perf.sh emits full executable paths, so the
  next re-capture (a wider window, a busier machine) will commit
  home-directory and project paths. Write the redaction/selection rule next
  to the fixture, or filter comm to its basename in sample().

## Round 6 — 2026-09-07T00:37:16-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan untouched since the base commit; :389 still routes the Lua write through artifactpath and :79/:98-108/:311 still describe the reversed cmd/pair-hoprtt + GO_BINS design.
- BR-5 — not-addressed — Residual is the two sample blocks; reproduced live and under perf_test.sh's own stub PATH — `### cputime`/`### procs` render empty because the pipeline's exit status is awk's, not ps's.
- BR-15 — addressed — Guard at hoprtt.go:53 plus TestSummaryOnEmptyInputDoesNotPanic; note the test asserts med==0, an in-domain value, which teaches the wrong contract even though the path is unreachable from Run.
- BR-16 — not-addressed — probe_line still prints only $1/$2; the sample count $4 is used solely in the failure branch, so a pipeRTT that broke early still reads as a full run.
- BR-17 — not-addressed — sample() at perf.sh:130 still takes awk $4 and comm is still a full path; under ARCH-SECURE this is worth more than Minor because the report is designed to leave the machine.
- BR-18 — not-addressed — Widened, not fixed — PAIR_PERF_BUDGET=abc emits three `[: integer expression expected` errors and still reports; PAIR_PERF_WINDOW='1; echo pwned' yields window_seconds=1; echo pwned. Still undocumented in atlas/README.
- BR-19 — not-addressed — Nothing in the window records that 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary.
- BR-20 — not-addressed — doctor/README.md is untouched in this window and README.md:607 points readers there for the doctor surface.
- BR-21 — not-addressed — All M1 checkboxes remain `- [ ]` and the Log has no 2026-09-07 entry. Numbers now available to record: pipe 0.006 ms, fork+exec 1.5-1.66 ms, perf.sh 2.26 s of a 6 s budget; zellij unmeasurable in this shell.
- BR-22 — addressed — Subcommand + dispatcher registration, pinned by TestProbeIsReachableThroughTheShippedPairBinary which builds ./cmd/pair-go; verified `./bin/pair hoprtt` returns 0.006 ms.
- BR-23 — addressed — verdict is now asymmetric and doctor_test.lua fails without it; the sweep the finding demanded is incomplete — see the new parse_samples finding.
- BR-24 — addressed — mktemp fallback at perf_test.sh:54; bare `mktemp -d` still fails in this shell, so the fallback is the live path and `make test-perf-capture` passes.
- BR-25 — not-addressed — Behaviour verified fixed (PAIR_PERF_BUDGET=2 keeps pipe_hop and fork_exec while shedding top/iostat/sample_b), but NO test exercises a squeeze and the probe keys are absent from perf_test.sh's required-key list, so reverting the shed order stays green. Also the secondary is untouched: elapsed can still exceed budget because no stage is bounded.
- BR-26 — not-addressed — perf_test.sh:23 still `*[!0-9]*) continue`.
- BR-27 — not-addressed — Still four shapes — collect() :51, the top block :101, disk :178, probe_line :211 — with sample()/cputimes() skipping the ladder entirely, which is the structural cause of BR-5's residual.
- BR-28 — not-addressed — Still awk arithmetic in perf.sh and still divides by WINDOW unguarded; worse now, because when the budget sheds sample_b the `sleep "$WINDOW"` never runs yet the rate is still divided by WINDOW, so the divisor names a window that did not occur.
- BR-29 — not-addressed — perf.sh:165-166 still says "n/a (vm_stat unavailable)" for a vm_stat that exists and failed.
- BR-30 — not-addressed — doctor.lua still says "The fixture in nvim/fixtures/" (it is doctor/fixtures/) and delta's Returns list still omits `unmeasured`, which the code sets and the test asserts.
- BR-31 — not-addressed — delta still detects reuse only via etime; an unparseable etime drops the pid into the rate branch where cb < ca yields a negative cpu_pct.
- BR-32 — not-addressed — No redaction or selection rule recorded next to doctor/fixtures/perf_capture.txt; it is clean of /Users/ paths only by the same truncation accident.

### Raised

- **BR-33** [Critical] `traceability` The plan still specifies the cmd/pair-hoprtt + GO_BINS design that round 3 reversed, with no "## Revisions" entry
  This is the 2nd finding in family `traceability`. Rule, not site: a durable
  artifact that describes the work is updated in the SAME round the work
  changes it, via an appended "## Revisions" entry — because the plan is what
  the next milestone is executed from, and a stale one re-introduces the
  finding the change removed. Prevalence 2/2 (BR-21: the issue Plan and Log
  do not record what M1 did; this: the plan does not record what M1 became).
  Evidence: the plan's last commit is df283abe, the review base. Core
  concepts line 79 says `pair-hoprtt` lives in `cmd/pair-hoprtt/main.go`;
  the code is `cmd/internal/hoprttcmd/hoprtt.go` and `cmd/pair-hoprtt/` was
  deleted in c8f2d044. Line 107 still says perf.sh "locates it as
  $PAIR_HOME/bin/pair-hoprtt" — verbatim the expression BR-22 called
  Critical. M1.2b (:311-313) still instructs a GO_BINS entry and a
  `bin/pair-hoprtt` check. M2.6 (:389) still routes the Lua rolling-file
  write through artifactpath, disposed not-addressed twice by the plan gate.
  Fix: one Revisions entry plus edits at :79, :98-108, :283, :311-313, :389,
  :402.
- **BR-34** [Important] `unenforced-operating-envelope` The budget is checked between stages but no stage is bounded, so a single slow collector blows it without limit
  This is the 3rd finding in family `unenforced-operating-envelope`, so the
  deliverable is the rule: a wall-clock budget is enforced only when every
  operation that can exceed it is itself bounded; a check before a stage
  starts does not bound the stage. Prevalence 3/3 — BR-11 (no deadline at
  all), BR-25 (wrong shed order), this (no bound once a stage is entered).
  Evidence: `top -l 2 -n 60` (perf.sh:108), `iostat -d -w 1 -c 2` (:183),
  `sleep "$WINDOW"` (:158) and each probe_line invocation run to completion
  once started. The committed fixture reads elapsed_seconds=5 against
  budget_seconds=6 — one second of slack — and the operator symptom quoted
  in the issue is "top took 10 seconds". pipeRTT's hardcoded 500 samples
  (hoprtt.go:178) is 5 s at the ~10 ms/hop the issue says visible lag would
  require. Because perf_test.sh:36 asserts elapsed <= budget, `make test`
  goes red under exactly the conditions the capture exists for. Enumeration
  to sweep in one pass: wrap each external stage in a bounded runner, and
  derive probe sample counts from remaining budget and the observed first
  sample rather than from constants.
- **BR-35** [Important] `failure-reported-as-measurement` parse_samples turns a budget-shed sample_b into an empty-but-present sample, so delta reports every process as vanished
  This is the 8th finding in family `failure-reported-as-measurement`. The
  rule was already stated at BR-23 and its sweep enumeration was written
  ("every exported function in doctor.lua's 208 block"); this is a member of
  that enumeration the hand-sweep missed, which is the family reporting that
  a hand-sweep is the wrong instrument. Reproduced under `nvim -l` with a
  capture containing `## sample_b` / `skipped=budget reserved for probes`:
  delta returns rates=0 vanished=2 started=0 rows_a=2 rows_b=0 — on a real
  host that is vanished=856, a fabricated spawn-storm claim from a
  measurement that never happened, in exactly the squeezed-capture condition
  the tool exists for. The same shape arises from BR-5's residual, where a
  failed ps yields two empty samples that read as "nothing is running".
  Deliverable at this prevalence is mechanical: one table-driven test in
  doctor_test.lua iterating the exported functions of the #208 block and
  asserting each maps absent/failed input to absence, so the next function
  added is covered by construction. Site fix: parse_samples returns nil for
  a block carrying `skipped=` or no `###` sections.
- **BR-36** [Important] `untested-shell-surface` Nothing tests perf.sh's sample-row grammar, so the perf.sh-to-delta contract the Lua test claims to pin does not exist
  This is the 3rd finding in family `untested-shell-surface`. Rule: when two
  components are joined by a recorded fixture, the recording pins only the
  CONSUMER; the producer needs its own live-run assertion, or drift on the
  producer side is invisible. Prevalence 3/3 (BR-9 no test for perf.sh at
  all; BR-25's fix unpinned; this). doctor_test.lua asserts "the perf.sh ->
  delta contract is pinned. A change to either side that breaks the other
  now fails here" — that claim is false: it reads the static file
  doctor/fixtures/perf_capture.txt and never invokes perf.sh. Demonstrated
  in a scratch copy: changing sample() from tab-separated to pipe-separated
  output left `sh doctor/perf_test.sh` printing "perf.sh shape tests passed"
  and exiting 0 (perf_test.sh:23's `*[!0-9]*) continue` swallows the rows,
  per BR-26). perf_test.sh's required-key list also omits every probe key,
  which is why BR-25's shed-order fix has no failing test. Fix: assert a live
  run's `### procs` rows against ^[0-9]+\t\S+\t[0-9]+\t.+$ and `### cputime`
  against ^[0-9]+\t\S+$, and add pipe_hop_ms/fork_exec_ms to the key list.
- **BR-37** [Minor] `build-artifact-committed` perf_test.sh's mktemp fallback writes .perf-test-stub.$$ into the worktree and nothing gitignores it
  This is the 2nd finding in family `build-artifact-committed`. Rule: any
  path a build or test writes into the worktree is gitignored in the same
  change that introduces the write. Prevalence 2/2 (BR-4: the staged
  pair-hoprtt binary; this). perf_test.sh:54 falls back to
  "$repo/.perf-test-stub.$$" when mktemp fails — and bare `mktemp -d` fails
  in the sandboxed agent shell this file's header names as its target
  reader, so the fallback is the live path there, not the exception. An
  interrupted run leaves a directory of fake ps/top/sysctl executables
  untracked at the repo root.

## Round 7 — 2026-09-07T00:50:44-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan :389 still routes M2.6's rolling-file write through artifactpath; verified no CLI surface exists and pair_data_dir() (nvim/init.lua:484) remains the idiom. The GO_BINS/pair- prefix half is moot now the probe is a subcommand.
- BR-5 — not-addressed — collect() is right for the named collectors, but emit_sample's `cputimes 2>/dev/null || say "n/a (ps unavailable)"` is a pipeline whose status is awk's, so the fallback is unreachable; with ps denied here both sample blocks render empty and delta reads rows_a=0/rows_b=0 as "nothing is running".
- BR-16 — not-addressed — probe_line's awk (perf.sh:216-221) still emits only $1/$2; the sample count $4 survives only inside the failure message.
- BR-17 — not-addressed — perf.sh:130 unchanged. Also: the plan's ARCH-SECURE allowlist (name+pid for all, argv for go/compile/link/zellij/pair*/nvim) is not implemented at all.
- BR-18 — not-addressed — WINDOW, BUDGET and the newly added PROBE_RESERVE are all unvalidated and undocumented; PROBE_RESERVE > BUDGET makes collectors_done true immediately.
- BR-19 — not-addressed — No mention of 4f9365b3 or M2.2b landing inside M1's window in the issue or the plan.
- BR-20 — not-addressed — grep for "perf" in doctor/README.md returns nothing. Root README also unchanged for `pair hoprtt` / `make test-perf-capture`, though its subcommand list is explicitly non-exhaustive.
- BR-21 — not-addressed — Issue Plan M1 still unticked; Log has only the 2026-09-06 entry. M1.4's zellij ~13ms baseline is still unmeasured — the probe renders `n/a (probe failed)` in this shell.
- BR-25 — not-addressed — Behavior IS fixed and I reproduced it (PAIR_PERF_BUDGET=2 sheds top/iostat/sample_b, probes survive), but nothing pins it: perf_test.sh's key list carries no probe key and there is no squeezed-budget run, so reverting the shed order leaves make test green.
- BR-26 — not-addressed — perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged; the new grammar loop is a separate pass and does not replace it.
- BR-27 — not-addressed — Four ladders still present: top (:101-120), disk (:178-189), probe_line (:211), and sample()/cputimes() (:130-131) bypassing collect() entirely.
- BR-28 — not-addressed — perf.sh:172's swap arithmetic is unchanged and still divides by WINDOW unguarded.
- BR-29 — not-addressed — perf.sh:166 still reports a vm_stat that exists but exits non-zero as "(vm_stat unavailable)".
- BR-30 — not-addressed — doctor.lua:163 still says "The fixture in nvim/fixtures/"; :75 still omits `unmeasured` from the documented return shape.
- BR-31 — not-addressed — doctor.lua:90 unchanged; no `cb < ca` guard in the rate branch, so an unparseable etime still lets a reused pid produce a negative rate.
- BR-32 — not-addressed — No redaction or selection rule recorded; doctor/fixtures/ contains only perf_capture.txt, and sample() still emits full executable paths.
- BR-33 — addressed — Revisions entry appended recording the reversal, the shed order, verdict asymmetry and the fixture location; per AGENTS.md the append-don't-overwrite convention makes the body edits optional. The M2.6/artifactpath item it also named stays open as BR-1.
- BR-34 — not-addressed — top -l 2 -n 60 (:108), iostat -d -w 1 -c 2 (:183), sleep "$WINDOW" (:158), probe_line, and pipeRTT's hardcoded 500 samples (hoprtt.go:178) all still run to completion once entered.
- BR-35 — not-addressed — Reproduced verbatim under nvim -l: a capture with `## sample_b` / `skipped=budget reserved for probes` still yields rates=0 vanished=2 started=0 rows_a=2 rows_b=0. dc5a3d03 covers only '' and "no sample block at all"; the table-driven test the finding asked for was not written.
- BR-36 — addressed — Verified by mutation — pipe-separated sample() output fails the new grammar loop once ps produces rows. Its vacuity in a ps-denied shell is raised separately.
- BR-37 — not-addressed — .gitignore carries no .perf-test-stub pattern; perf_test.sh:54's fallback is unchanged.

### Raised

- **BR-38** [Important] `untested-shell-surface` perf_test.sh asserts a live run against the ambient system, so the BR-36 grammar pin validates zero rows wherever ps is denied
  This is the 4th finding in family `untested-shell-surface`. Do NOT fix this
  instance — the rule is: a shell test must CONTROL the environment its
  subject reads; a live run against the ambient system asserts nothing when
  the system denies the tool, and a green run is then a sample of size zero
  reporting as coverage. Prevalence 4/4 (BR-9 no test at all; BR-25's fix
  unpinned; BR-36 the recorded fixture pinning only the consumer; this).
  Evidence: `ps` is denied in this agent shell (`/bin/ps: Operation not
  permitted`), so `### procs` and `### cputime` render as empty sections;
  the loop at perf_test.sh:84-99 iterates zero times, `rows` (:98) is
  computed and never read, and :101-102 assert only that the headers exist.
  I mutated sample() to pipe separators in a scratch copy and
  `sh doctor/perf_test.sh` printed "perf.sh shape tests passed" exit 0;
  the same mutation fails correctly once I put a recorded-output ps stub on
  PATH. Sweep in one pass: promote the existing stub dir (:54-58) from
  stateless `exit 1` doubles to a recorded-output ps/top/vm_stat/iostat
  fake, assert rows > 0, and add a PAIR_PERF_BUDGET=2 run asserting
  pipe_hop_ms/fork_exec_ms survive and cpu_idle_pct sheds — which also
  closes BR-25's missing pin (ARCH-MOCK, ARCH-ORDER).
- **BR-39** [Important] `failure-reported-as-measurement` swap_rate divides by WINDOW even when sample_b was shed and the sleep never ran, reporting a rate over time that did not pass
  This is the 9th finding in family `failure-reported-as-measurement`. Do
  NOT fix this instance. The rule this round's instances need, stated once:
  when a stage is SHED or FAILS, every value derived from that stage must
  shed with it — the shed has to propagate through the dependency graph, not
  stop at the stage that was skipped. Evidence: perf.sh:153-160 skips
  `sleep "$WINDOW"` under a squeezed budget, but :162-174 unconditionally
  divides the vm_stat counter difference by WINDOW. Reproduced: the full run
  reports pageins_per_s=40.0; `PAIR_PERF_BUDGET=2` reports
  pageins_per_s=3.5 from two reads microseconds apart, rendered
  identically. The enumeration this rule implies, all live: swap_rate
  (here); parse_samples/delta reading a shed sample_b as a real one (BR-35);
  the sample blocks degrading to silence rather than n/a (BR-5's residual);
  and, on the same axis, delta's cpu_pct dividing by the DECLARED window
  while the measured one (`at_s`, emitted per sample) is discarded.
- **BR-40** [Important] `incomplete-parse-contract` parse_samples is billed as the perf.sh-to-delta contract but omits window_seconds, which delta needs and which crashes it as a string
  doctor.lua:164 returns only sample_a and sample_b, while
  `delta(a, b, window)` also needs the window — which lives in the capture
  as the text `window_seconds=2`. M2's caller must therefore re-parse the
  capture with its own pattern, a second untested copy of the contract this
  function exists to own (ARCH-PURPOSE's hand-maintained restatement), and
  if it forwards the string unconverted doctor.lua:80 raises "attempt to
  compare string with number" — verified under nvim -l. That is a crash on
  the primary path, at the moment the operator invoked the tool because the
  machine was already struggling. Fix: return a third `meta` value carrying
  window_seconds and both at_s values as numbers (which also supplies the
  measured window the preceding finding needs), and coerce with
  tonumber(window) at delta's guard so a bad value degrades to the existing
  empty result rather than throwing.

## Round 8 — 2026-09-07T01:02:36-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — The pair- prefix half is overtaken by the subcommand reversal, but plan M2.6 still routes the Lua write through artifactpath, which has no CLI surface (verified: no reference in cmd/pair-go/main.go or dispatcher.go).
- BR-5 — not-addressed — Scalar collectors are fixed and verified, but emit_sample's `cputimes || say n/a` is dead (the || sees awk's status, not ps's), so both sample sections still render as silence under denial.
- BR-16 — not-addressed — probe_line's awk still prints only $1/$2; the sample count in $4 is discarded.
- BR-17 — not-addressed — perf.sh:130 still truncates at awk $4 and emits comm= as a full executable path.
- BR-18 — not-addressed — PAIR_PERF_WINDOW/BUDGET/PROBE_RESERVE all flow unvalidated; none documented outside workshop/.
- BR-19 — not-addressed — No Log entry or plan note records that 4f9365b3 (the M2.2b join) landed inside the M1 window.
- BR-20 — not-addressed — grep perf doctor/README.md returns nothing.
- BR-21 — not-addressed — Issue Plan M1 and every plan M1.x checkbox are unticked; the Log has only the 2026-09-06 filing entry, so M1.4 and M1.5 have no recorded evidence.
- BR-25 — not-addressed — Shed ORDER is fixed and verified at PAIR_PERF_BUDGET=2, but PROBE_RESERVE only gates collector start; probe_line guards on over_budget, so an 8s top stub yields zero probe rows and elapsed_seconds=8 over a 6s budget.
- BR-26 — not-addressed — perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
- BR-27 — not-addressed — Four ladder shapes remain: the top block, the disk block, probe_line, and the sample blocks that skip collect() entirely.
- BR-28 — not-addressed — perf.sh:179 still does the rate arithmetic in awk and divides by WINDOW without a zero guard.
- BR-29 — not-addressed — Reproduced with a vm_stat stub exiting 1: renders `swap=n/a (vm_stat unavailable)`. Same misnaming at _loadavg, which reports "returned nothing" when sysctl failed, because the pipeline's status is awk's.
- BR-30 — not-addressed — doctor.lua still says "The fixture in nvim/fixtures/" and still omits `unmeasured` from delta's documented return shape.
- BR-31 — not-addressed — The reuse branch is still etime-only; no `cb < ca` guard at the point of derivation.
- BR-32 — not-addressed — The fixture still carries no redaction/selection rule; sample() still emits full paths, so the next re-capture commits home-directory paths.
- BR-34 — not-addressed — Reproduced: a `top` stub taking 8s runs to completion and the capture reports elapsed_seconds=8 against budget_seconds=6, which also makes perf_test.sh:36 go red under exactly the conditions the capture exists for.
- BR-35 — addressed — Mutation-verified: reverting usable() in a scratch copy produced two failures including "a shed sample must not report every process as vanished".
- BR-37 — not-addressed — Confirmed live: `mktemp -d` fails in this agent shell, so `$repo/.perf-test-stub.$$` is the taken path, and .gitignore has no entry for it.
- BR-38 — not-addressed — The fix asserts rows against the ambient system instead of controlling it, so `sh doctor/perf_test.sh` now FAILS ("only 0 sample rows") wherever ps is denied, turning make test red for the reader the file header names. The stateless stubs were not promoted to a recorded-output fake, and in_sample/rows at :85-98 are now dead subshell locals.
- BR-39 — not-addressed — The swap_rate instance is fixed but pinned by no test; 2 of the 4 enumerated members remain live - the sample blocks still degrade to silence (BR-5 residual), and delta still divides by the DECLARED window while the measured at_s values are parsed away.
- BR-40 — addressed — Mutation-verified twice: removing tonumber(window) crashes the suite, removing the window_seconds parse fails the assertion. The at_s half of the fix sketch is carried by BR-39.

### Raised

- **BR-41** [Minor] `unguarded-edge-case` pair hoprtt silently ignores unrecognised arguments and runs the 500-sample pipe probe instead
  This is the 4th finding in family `unguarded-edge-case`. Do NOT fix this
  instance. The rule covering all four: a value read from outside the
  function - an argv token, an env var, a counter difference - is rejected
  at the point it is read when it is not one of the forms the code
  understands; falling through to a default produces a successful-looking
  measurement of something the caller did not ask for. Evidence:
  hoprttcmd.go:148-152 tests args[0] against exactly "-child" and "-spawn",
  so `pair hoprtt --spawn 5 -- x` runs pipeRTT(500) and exits 0 with a
  pipe-hop number labelled as whatever the caller thought it asked for -
  the same defect TestUsageErrorsRatherThanSilentlyMeasuringNothing exists
  to prevent, one token away from the cases it covers. Prevalence 4/4, all
  live: this, BR-31 (a reused pid detected only via etime, so an
  unparseable etime yields a negative rate), BR-28 (WINDOW=0 divides to
  inf), BR-18 (PAIR_PERF_* unvalidated into sleep and awk -v). The
  enumeration to sweep in one pass is those four sites.
- **BR-42** [Minor] `traceability` Three pure entities shipped in M1 have no row in the plan's Core concepts table
  This is the 3rd finding in family `traceability`. Do NOT fix this
  instance. The rule: the Core concepts table is the greppable contract the
  next milestone reads, so every entity that ships gets its row in the SAME
  round it lands - a table that lags is a plan claiming a smaller surface
  than the code delivers, which is the same defect as one claiming a
  surface the code lacks (BR-33). Live members: `parse_samples`
  (nvim/doctor.lua, billed in its own docstring as THE perf.sh-to-delta
  contract), `parse_duration`, and `verdict`/`FRAME_MS` - the last of which
  M2.3 describes in prose with a symmetric contract the code has since
  replaced with an asymmetric one. Prevalence 3/3 with BR-21 (the issue's
  M1 row still names cmd/hoprtt, a path that does not exist) and BR-19 (no
  record that 4f9365b3 landed inside the M1 window). The sweep is one
  `## Revisions` entry adding the rows and correcting the two stale
  references.

## Round 9 — 2026-09-07T11:05:44-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan M2.6:389 still says "via artifactpath"; that package has no CLI surface Lua can reach. The GO_BINS/pair- prefix half is overtaken by the subcommand reversal.
- BR-5 — not-addressed — Reproduced with ps denied: '### cputime' and '### procs' render as empty sections; the '|| say n/a' arms at perf.sh:146,148 are unreachable because the pipeline exits with awk's status.
- BR-16 — not-addressed — probe_line's awk at perf.sh:227 still prints only $1/$2; the sample count is discarded.
- BR-17 — not-addressed — sample() at perf.sh:130 still takes $4 of a full comm path; the report carries home-directory paths against the plan's own ARCH-SECURE contract.
- BR-18 — not-addressed — Sharper evidence: PAIR_PERF_WINDOW=0 and =abc both render the entire '## swap_rate' section as nothing at all -- no key, no n/a, no reason. No shell injection; quoting is correct.
- BR-19 — not-addressed — Still no record that 4f9365b3 landed inside the M1 window.
- BR-20 — not-addressed — grep -c perf doctor/README.md = 0; it still describes doctor/ as the drift tool only.
- BR-21 — not-addressed — Issue Plan M1 row unticked and still naming cmd/hoprtt; no 2026-09-07 Log entry, no M1.4/M1.5 evidence.
- BR-25 — addressed — Verified live: PAIR_PERF_BUDGET=2 sheds top, iostat and sample_b while pipe_hop_ms and fork_exec_ms survive. Behaviour correct but pinned by no test; that residual is tracked in issue 210.
- BR-26 — not-addressed — perf_test.sh:23's negative character-class arm is unchanged.
- BR-27 — not-addressed — collect() is still re-implemented at perf.sh:101-120, :185-196, :218-229 and skipped by sample()/cputimes() -- structurally why BR-5's residual survives.
- BR-28 — not-addressed — perf.sh:179 is still shell arithmetic against the file's own rule 2, and WINDOW=0 now silently erases the whole section instead of printing inf.
- BR-29 — not-addressed — perf.sh:172-173 still reports a vm_stat that exists but fails as "vm_stat unavailable".
- BR-30 — not-addressed — doctor.lua:157 still says nvim/fixtures/; doctor.lua:75's return-shape list still omits unmeasured.
- BR-31 — not-addressed — doctor.lua:95 still gates reuse on etime alone; cb < ca is not rejected at the point of derivation.
- BR-32 — not-addressed — Fixture still carries 0 /Users/ paths only by the truncation accident; no selection or redaction rule recorded beside it.
- BR-34 — addressed — Split to issue 210 by explicit operator decision and filed with a Spec, a mechanism and a Done-when. Treated as settled, not re-litigated at this gate.
- BR-37 — not-addressed — Confirmed the fallback is the LIVE path here -- mktemp -d fails with "Operation not permitted" in this shell -- and .perf-test-stub.* is still absent from .gitignore.
- BR-38 — not-addressed — The >=10-row guard swapped a vacuous pass for a hard failure: sh doctor/perf_test.sh exits 1 here, and test-perf-capture is in the make test chain. The rule (control the environment) is still unapplied.
- BR-39 — not-addressed — swap_rate is fixed and verified, but 2 of the 4 sites the finding's own enumeration named are live: the sample blocks' silence, and delta dividing by the declared window while the measured at_s delta is emitted and discarded.
- BR-41 — not-addressed — hoprttcmd.go:148-152 unchanged; an unrecognised argv token still falls through to pipeRTT(500) and exits 0.
- BR-42 — not-addressed — Core concepts table unchanged: pair-hoprtt still points at cmd/pair-hoprtt/main.go, and parse_samples / parse_duration / verdict still have no rows.

### Raised

- **BR-43** [Minor] `injected-io-seam-bypassed` hoprttcmd.Run takes injected writers but child() reads os.Stdin and writes os.Stdout, and the package is registered as a streaming subcommand with no stdin
  Run (hoprtt.go:146) accepts stdout/stderr and report() honours them, but
  child() (hoprtt.go:35-45) touches os.Stdin/os.Stdout directly. The
  dispatcher registers hoprtt with Streaming: true (dispatcher.go:63) while
  cmd/pair-go/main.go:91 passes no stdin, so the signature says "reads no
  stdin" and the code contradicts it. This is a newly-introduced internal
  package that downstream M2 work will call, so the surface is worth
  settling now: a future test adding {"-child"} to
  TestUsageErrorsRatherThanSilentlyMeasuringNothing's table would block on
  the real terminal rather than the injected buffer. It works today only
  because the child is always a re-exec'd process. Take a stdin io.Reader
  like every sibling Run, or document that -child is process-level only and
  keep it out of the injected-writer path (ARCH-PURE).

## Round 10 — 2026-09-07T13:26:16-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 11 — 2026-09-07T13:50:40-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Plan Revisions record M2.6 landing via pair_data_dir() not artifactpath, and the probe as the `pair hoprtt` subcommand; both mechanisms verified reachable in the tree.

### Raised

- **BR-44** [Critical] `failure-reported-as-measurement` The completion leg still measures nothing — the gate moved from mode to cursor column, and `editor: fast` is still emitted as grounds for exclusion
  11th in this family. The rule was stated in round 10 and applied only to the
  precondition that round named. nvim_buf_call runs complete_now in the autocmd
  window at col 0, so path/word/spell_complete all return at `if col == 0 then
  return end` (init.lua:1644, :1817, :1949). Measured under the real init.lua:
  complete_now takes 0.0040 ms and the row records "editor":"fast", while
  doctor/SKILL.md:79 tells the reader that `fast` means the #202 chain ran inside
  a frame and that #201/#203 are excluded on it. Fix the class: assert each leg's
  precondition where it is timed (completion needs a completable token at the
  cursor; redraw needs a real UI), render n/a with the unmet precondition
  otherwise, and add one wiring assertion that the timed chain reached its
  candidate build. Note vim.fn.complete() raises E785 outside Insert mode, so the
  candidate build must be split off the complete() call the way complete_now was
  split off the gate.
- **BR-45** [Important] `duplicated-logic` The single-sourced key set stops at the Lua boundary — perf.sh and perf_test.sh restate it, and a producer rename breaks nothing
  3rd in this family. Rule: one declaration per fact, and where a second language
  cannot import it, a test asserts the two agree. doctor.lua:398-413 collapsed the
  three Lua sites, but perf.sh:198-200 restates the set in shell and
  perf_test.sh:28 restates a different subset. Verified by renaming
  swapins_per_s to swap_in_rate in a scratch copy of doctor/ — perf_test.sh's
  failure count was unchanged, and doctor_test.lua cannot see it because it builds
  its degraded input from HEADLINE_KEYS itself. The prompt would silently drop the
  swap row, which is the C1(b) defect returning. Fix: perf_test.sh runs the real
  perf.sh and asserts every key read out of doctor.HEADLINE_KEYS/PROBE_KEYS via
  `nvim -l` appears in the output.
- **BR-46** [Important] `untested-shell-surface` The ps-content filter and the buffer-changed interleaving are both unexecuted, though the seams to control each now exist
  5th in this family. BR-38's rule generalised: a test controls the inputs its
  subject reads — subprocess output and event ordering alike. Two live members.
  (1) perf.sh:143-152's basename + control-byte strip has no test; perf_test.sh's
  fake ps only `exit 1`s, so reverting the awk body to `$4` leaves every suite
  green, and doctor/fixtures/perf_capture.txt:82 still carries full paths the
  current producer no longer emits. (2) init.lua:4157-4168's buffer-changed
  branch — the plan's only data-loss path — is unexecuted; the injected runner
  receives cb, so the test can rewrite the buffer before invoking it. Sweep both:
  recorded-output ps fake plus a regenerated fixture, and the two missing
  interleavings in tests/pair-doctor-test.sh.
- **BR-47** [Important] `docs-gate` SKILL.md and atlas name perf-capture-latest.txt, which the rework stopped writing
  3rd in this family. Rule: a doc naming a runtime artifact names the one the code
  produces, checked in the window that changes it — enumerable by grepping docs
  for $PAIR_DATA_DIR/<name> against the writers in init.lua. init.lua:4128 now
  writes perf-capture-<epoch>.txt (verified: perf-capture-1788813769.txt), while
  doctor/SKILL.md:74 and atlas/index.md:40 still name the fixed file and SKILL.md
  instructs the agent to open it — the truncated-send recovery path #211 exists
  for.
- **BR-48** [Important] `unreleased-resource` time_editor leaks a scratch buffer per invocation while its comment claims a guaranteed delete
  New family. Rule: a resource acquired inside a function is released on every
  exit path, and a comment claiming teardown is a claim a test should hold.
  init.lua:3999 creates the buffer; nothing deletes it (grep nvim_buf_delete
  returns nothing) and :4001 asserts otherwise. Measured: three :PairDoctor runs
  took the valid-buffer count from 1 to 4. The plan's ARCH-ORDER table has a row
  requiring the teardown, so the plan claims delivered behaviour the code lacks.
- **BR-49** [Minor] `traceability` The M2.6 deferrals are recorded only in the plan, which archives at close — issue 210 records neither
  4th in this family. The plan's Revisions say the missing window-length field and
  the uncapped perf-captures.jsonl are "deferred to #210"; #210 covers only
  time-bounding plus three re-surfaced M1 findings. A deferral must land in the
  artifact that survives the archive.

## Round 12 — 2026-09-07T14:17:21-07:00 (claude) — BLOCKED

### Disposed

- BR-44 — not-addressed — Completion leg verified fixed and pinned; the redraw/input legs named in the same rule still have no precondition or proof-of-work.
- BR-45 — addressed — Verified by renaming swapins_per_s in a scratch copy — perf-key-conformance-test fails exactly one assertion.
- BR-46 — addressed — Both members verified by revert: the awk redaction and the buffer-equality guard each take a suite red.
- BR-47 — addressed — SKILL.md:74 and atlas/index.md:40 now name perf-capture-<epoch>.txt; no perf-capture-latest remains in the tree.
- BR-48 — addressed — Verified by removing the nvim_buf_delete — the buffer-count assertion goes red.
- BR-49 — not-addressed — Issue 210 is unchanged since b57cd08a (outside this window) and records neither the missing window-length field nor the uncapped perf-captures.jsonl.

### Raised

- **BR-50** [Critical] `discarded-failure-signal` send_generated_prompt discards send_to_agent's failure, so :PairDoctor clears the operator's note on a failed send and reports nothing
  nvim/draft_send.lua:43 returns `false, phase, '<label> exited N'` when a zellij
  action fails, and nvim/init.lua:794 returns false with 'no attached UI'.
  nvim/submission.lua:66-69 throws all three away and returns true
  unconditionally. In on_capture (nvim/init.lua:4187) the consume branch keys
  off `raw` alone, so a successful capture plus a failed send reaches
  nvim/init.lua:4223 and clears the buffer with no notify — the operator sees an
  emptied draft and concludes it was sent. The trigger is a failing zellij
  action, which is the degraded machine this capture exists for. The plan (M2.4),
  the code comment at :4218-4222, and plan Revisions #8 all claim "consume only
  on a successful send"; none of the three is implemented, and the repo's own
  draft path already states and honours the rule at nvim/init.lua:1519-1521
  ("Durability is the submission gate … any failure leaves all authored state
  intact"). Same underlying rule as BR-2 in `failure-reported-as-measurement` — a
  discarded error becomes a fabricated success — but in the send domain rather
  than the measurement domain, and here the fabricated success authorises a
  destructive step. Fix: return send_low_level's ok from send_generated_prompt
  (its two other callers ignore the result, so this is additive), gate the
  consume on it, notify on failure, and drive the interleaving with a
  _G.PairTestSendToAgent stub returning false.
- **BR-51** [Important] `docs-gate` README.md and doctor/README.md still describe :PairDoctor as a drift-only pointer and never mention that it now consumes the draft buffer
  This is the 4th finding in family `docs-gate`, so the deliverable is the rule
  and its enumeration, not these two files. Rule: a change to a user-facing
  surface updates every doc that describes that surface, and the enumeration is
  mechanical — `grep -rl '<surface>' --include='*.md'` minus
  workshop/{history,plans,issues}. For `:PairDoctor` that is README.md,
  doctor/README.md, doctor/SKILL.md, atlas/index.md; the last two were updated
  and the first two were not. README.md:605 still says it reads "the session's
  adaptation flight recorder"; doctor/README.md:63-69 still says it hands the
  agent "an instruction to run doctor.sh", and doctor/ 's own README never
  mentions perf.sh. Neither states the user-visible behaviour change that the
  draft buffer is consumed as the note. Measured prevalence of the class: three
  members survived to this round — BR-20 (this same doctor/README.md gap,
  disposed not-addressed twice then demoted), BR-30 (nvim/doctor.lua:180 still
  says the fixture is in nvim/fixtures/ when it is doctor/fixtures/, and :75-76
  still omits `unmeasured` from delta's documented return shape), and this. Each
  round fixed the site named instead of writing the enumeration, which is why the
  class keeps returning.

## Round 13 — 2026-09-07T14:38:33-07:00 (claude) — passed

### Disposed

- BR-44 — addressed — Mutation-verified: removing the cursor placement turns the work-counter assertion red; live render shows completion 0.8ms, matching #202's benchmark. Redraw's precondition re-raised separately as the unswept member.
- BR-49 — not-addressed — No commit in the window touched issue 210; it still records only time-bounding plus the three M1 findings, and #208's Log names only "no stage is time-bounded".
- BR-50 — addressed — Production behaviour fixed and the init.lua gate is mutation-pinned; the submission.lua half is unpinned, rolled into the new coverage finding.
- BR-51 — addressed — Enumeration swept and verified — grep over *.md leaves only CHANGELOG (release-scoped) and gitignored runtimebundle copies, which are in sync.

### Raised

- **BR-52** [Important] `untested-shell-surface` Every test drives the send as failing, so the consume-on-success clear, the wired join, and submission.lua's real-result return are all unexecuted
  This is the 6th finding in family `untested-shell-surface` — first in Lua rather
  than shell, same rule. The rule that covers all six: a new surface is covered
  only when EVERY reachable branch is executed; a suite that drives one side of a
  gate reports coverage it does not have. Enumeration for this diff, and it is
  mechanical — list the branches on_capture introduces and check each. Unexecuted:
  (1) the clear at nvim/init.lua:4229, because send_to_agent returns false with
  'no attached UI' (nvim/init.lua:741) in headless, so tests 1-4 never reach it and
  test 5's stub is redundant for its own premise; (2) the join at
  nvim/init.lua:4180-4184, because the fixture capture.txt has no `## sample_a`
  section, so `if a and b then` is never entered — this is the same wiring a prior
  round found dead in production; (3) submission.lua:75 — reverting it to
  `return true` leaves pair-doctor-test.sh AND submission_test.lua green
  (mutation-verified). Also unexecuted: the doctor-load failure, PAIR_HOME unset,
  the `capture is already running` guard, perf.sh-not-readable, sidecar-write
  failure, and both n/a notes on the completion leg. Fix the class: set
  _G.PairTestZellijExecutor for one case and assert the buffer IS cleared, give the
  fixture real sample blocks, and add a failing-send_low_level case to
  submission_test.lua.
- **BR-53** [Important] `failure-reported-as-measurement` The redraw leg is still timed with no precondition asserted — the second member of the enumeration round 11 named
  12th in this family. Round 11 stated the class and named both members
  ("completion needs a completable token at the cursor; redraw needs a real UI");
  only the completion member was swept. nvim/init.lua:4051-4053 times
  vim.cmd('redraw') unconditionally and feeds the result to a verdict that
  doctor/SKILL.md tells the reader to exclude pair#201/#203 on. Not reachable in
  production today — :PairDoctor always runs with a TUI attached — which is why
  this is Important rather than Critical; it is a regression guard on the last row
  of the enumeration. has_ui() already exists at nvim/init.lua:689, so the fix
  reuses it rather than adding a second UI check (ARCH-DRY).
- **BR-54** [Important] `sole-copy-on-lossy-channel` The operator's note is written only into the prompt, and the buffer is cleared on send
  nvim/init.lua:4189-4193 writes compact + rates + raw to the sidecar and omits the
  note; verified against a produced sidecar file. The buffer is then cleared on a
  successful send, and perf-captures.jsonl — the only other copy, and itself
  pcall'd and silent — is never named in the payload. The note is also uncapped, so
  a pasted log produces an arbitrarily long prompt whose middle is precisely the
  region pair#211 measured as dropped (1,025 bytes gone from a 2,447-byte send).
  This contradicts the invariant this milestone recorded in atlas/index.md and
  workshop/lessons.md: nothing of value exists only in the prompt — applied to the
  report but not to the operator's own input, which the same docs call the one
  thing that cannot be re-measured. Fix: prepend the note to the sidecar body.
- **BR-55** [Minor] `discarded-failure-signal` The rolling-log append discards pair_write_data_file's nil return, so the comparative series can stop accumulating silently
  2nd in family. The rule: a function whose contract is "returns whether the effect
  happened" must have that signal consumed, or surfaced to the operator where the
  caller cannot act on it. Enumeration over the new code: the sidecar write consumes
  it (payload degrades), send_generated_prompt now consumes it, the JSONL write at
  nvim/init.lua:4200 does not. M2.6's whole purpose is making the next investigation
  comparative; a permanently failing append is invisible.
- **BR-56** [Minor] `incomplete-parse-contract` The JSONL row has no schema version and encodes an empty probes table as [] rather than {}
  2nd in family. Verified on a real row: {"probes":[],...}. A degraded capture
  therefore changes the type of `probes` from object to array, breaking any external
  reader doing .probes.pipe_hop_ms — on a file whose stated purpose is to stay
  legible years later. The same row carries no version field.
- **BR-57** [Minor] `duplicated-logic` swap_na / disk_na / probes_na are three copies of one loop differing only in the key list
  4th in family. doctor/perf.sh:203-205. One `na_for <reason> <key>...` helper covers
  all three (ARCH-DRY).
- **BR-58** [Minor] `unguarded-edge-case` time_editor's synthetic TextChangedI mutates the live completion debounce state
  5th in family. nvim/init.lua:4046 fires the real autocmd, which cancels any
  pending completion timer and resets complete_last_fire — so invoking the
  diagnostic can drop a popup the operator was about to get. The comment above
  time_editor argues it cannot touch the draft because it uses a scratch buffer;
  that holds for buffer text but not for the shared debounce state.

## Round 14 — 2026-09-07T14:54:43-07:00 (claude) — BLOCKED

### Disposed

- BR-5 — addressed — Verified live with ps/sysctl/top/iostat all denied — every key renders n/a with a reason; no fabricated process_count=0, no bare load=.
- BR-16 — not-addressed — probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
- BR-17 — addressed — sample() strips the first three fields and basenames on '/'; perf_test.sh:164 pins "Google Chrome" surviving and :160 pins no /Applications/ reaching the report.
- BR-18 — not-addressed — No validation and no PAIR_PERF mention in README/atlas/SKILL; measured — WINDOW=0 makes the whole swap_rate section vanish with awk "division by zero".
- BR-19 — not-addressed — grep for 4f9365b3 across workshop/ and atlas/ hits only prior gate-ledger rounds; neither the issue nor the plan records it.
- BR-20 — addressed — doctor/README.md:70-88 now links perf.sh and documents the note/clear semantics.
- BR-21 — addressed — Issue Plan M1 is ticked with the design-reversal note; the M1 close line carries evidence and the zellij number (14.385 ms) is in the M2 baseline table.
- BR-26 — not-addressed — perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
- BR-27 — not-addressed — na_for deduped only the key lists; the top block, disk block, probe_line and emit_sample each still re-implement the availability/failure/empty ladder.
- BR-28 — not-addressed — Sharper than reported — at WINDOW=0 awk aborts and all three swap keys VANISH rather than yielding inf, and perf_test.sh:128/:156 drive that exact value.
- BR-29 — not-addressed — perf.sh:216-217 still says "vm_stat unavailable" when the first read fails; only the second read (:221) distinguishes failure from absence.
- BR-30 — addressed — doctor.lua:180 now names doctor/fixtures/ and :75-76 enumerates unmeasured.
- BR-31 — not-addressed — doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
- BR-32 — addressed — The finding's own alternative was taken — sample() basenames comm at the emitter, so no path can reach a re-capture; fixture verified free of /Users/ and /home/.
- BR-37 — not-addressed — perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry.
- BR-38 — not-addressed — Reproduced — sh doctor/perf_test.sh exits 1 in this sandboxed shell (grammar violation on perf.sh's own n/a line, plus "only 2 sample rows"), and test-perf-capture is in make test.
- BR-39 — not-addressed — The shed member is fixed (WINDOW_ELAPSED). The declared-vs-measured member is live — at_s is discarded by parse_samples, and SWAP_A is read before sample_a while _b is read after sample_b.
- BR-41 — not-addressed — hoprtt.go:148-177 unchanged; an unrecognised argument still falls through to pipeRTT(500).
- BR-42 — addressed — Plan Revisions item 3 enumerates the missing entities and a separate entry reverses every cmd/pair-hoprtt reference; the stale TABLE itself is carried as a plan-revision recommendation rather than re-raised.
- BR-43 — not-addressed — hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming with no stdin from main.go.
- BR-49 — not-addressed — Issue 210 now records BR-34/25/38/39 but still not the missing window-length field or the uncapped perf-captures.jsonl.
- BR-52 — addressed — All three prescribed members mutation-verified red — the clear, the join, and submission.lua's return. Residual unexecuted branches roll into the new coverage finding.
- BR-53 — not-addressed — Behaviour is correct and reachable (payload renders redraw n/a, verdict unknown), but removing the has_ui() gate in a scratch copy leaves every suite green — and vim.g.pair_test_has_ui already exists as the seam.
- BR-54 — not-addressed — The note does reach the sidecar, but no test opens the sidecar — reverting the prepend leaves pair-doctor-test.sh and doctor_test.lua both green.
- BR-55 — not-addressed — The notify is present but no test drives a failing pair_write_data_file, so the branch is never executed.
- BR-56 — addressed — Mutation-verified — deleting the vim.empty_dict line takes doctor_test.lua red on "an empty probes set encodes as an object, not []".
- BR-57 — addressed — na_for at perf.sh:206 collapses the three loops; the n/a key assertions under tool denial still pass.
- BR-58 — addressed — Answered as an explicit accepted trade-off with its reasoning recorded at nvim/init.lua:4048-4053, which is a legitimate resolution for this severity.

### Raised

- **BR-59** [Important] `untested-shell-surface` Three of the closing commit's six behaviour changes are unpinned — the same rule the commit closed BR-52 on
  This is the 7th finding in family `untested-shell-surface`. Do NOT fix these three
  instances. The rule covering all seven: a behaviour change lands with a test that
  fails without it, and the check is mechanical — revert the hunk in a scratch copy
  and confirm a suite goes red. The enumeration is the closing commit's own
  behaviour hunks, and I ran it. Pinned (verified red on revert): capture_record's
  schema + empty_dict, the consume-on-success clear, the parse_samples->delta join,
  submission.lua's real return. Unpinned (verified green on revert): the has_ui()
  redraw gate at nvim/init.lua:4062-4069, the operator note prepended to the sidecar
  at nvim/init.lua:4209-4211, and the JSONL-append failure notify at
  nvim/init.lua:4232-4239. The seams to close all three already exist —
  vim.g.pair_test_has_ui (nvim/init.lua:689), PAIR_DATA_DIR in the test harness, and
  the injected capture runner. Prevalence 7/7 with BR-9, BR-25, BR-26, BR-36, BR-38,
  BR-52; the commit that closed BR-52 is itself the newest member, which is the
  signal that the rule was fixed at the instance level and never as a class.
- **BR-60** [Minor] `docs-gate` doctor.lua and doctor_test.lua still claim "no vim API", and atlas's sidecar enumeration omits the note added in the same commit
  This is the 5th finding in family `docs-gate`. Do NOT fix these instances — the
  rule: a comment or doc that ENUMERATES something is a claim a grep can check, and
  it is updated in the same change that invalidates it; the enumeration is
  mechanical (grep the changed symbol across *.md and the module headers that
  describe it). Live members: nvim/doctor.lua:2 and nvim/doctor_test.lua:2 both say
  "no vim API here" while capture_record now calls vim.empty_dict and the new test
  calls vim.json; atlas/index.md describes the sidecar as "compact half, joined
  per-process rates, and both raw ps samples" while the same commit prepended the
  operator note to it — under a paragraph whose own subject is the invariant that
  nothing of value exists only in the prompt. Prevalence 5/5 with BR-20, BR-30,
  BR-47, BR-51.

## Round 15 — 2026-09-07T15:18:04-07:00 (claude) — BLOCKED

### Disposed

- BR-16 — not-addressed — probe_line (perf.sh:262-273) still reads only $1/$2; hoprtt's sample count $4 is discarded.
- BR-18 — not-addressed — No validation; grep for PAIR_PERF across README.md, atlas/, doctor/README.md and doctor/SKILL.md returns nothing.
- BR-19 — not-addressed — grep 4f9365b3 across workshop/ and atlas/ still hits only the gate ledger's own rounds.
- BR-26 — not-addressed — perf_test.sh:23's `*[!0-9]*) continue` arm is unchanged.
- BR-27 — not-addressed — The top block, disk block, probe_line and emit_sample each still re-implement collect()'s availability/failure/empty ladder.
- BR-28 — not-addressed — Reproduced at HEAD - PAIR_PERF_WINDOW=0 prints "awk: division by zero" and all three swap keys vanish; swapins_per_s is a HEADLINE_KEY, so a headline row is dropped, not just a value lost.
- BR-29 — not-addressed — perf.sh:216-217 still says "vm_stat unavailable" when the first read fails.
- BR-31 — not-addressed — doctor.lua:91 unchanged; cb < ca at :98 is still unguarded, so an unparseable etime yields a negative cpu_pct.
- BR-37 — not-addressed — perf_test.sh:59 unchanged and .gitignore has no .perf-test-stub entry; note the three newer fakes (:98, :141, :165) use $TMPDIR with no worktree fallback, so the file is now inconsistent with itself.
- BR-38 — addressed — Mutation-verified - the controlled ps at perf_test.sh:98-111 validates 12 rows in this shell where /bin/ps is denied, and a tab-to-pipe change in sample() takes it red.
- BR-39 — not-addressed — The shed member stays fixed. The declared-vs-measured member is live - parse_samples reads window_seconds and discards at_s, so delta divides by the DECLARED window on exactly the machine where sleep 2 does not take 2s.
- BR-41 — not-addressed — hoprtt.go:146-152 unchanged; an unrecognised argument still falls through to pipeRTT(500) and exits 0.
- BR-43 — not-addressed — hoprtt.go:35-45 still touches os.Stdin/os.Stdout while dispatcher.go:63 registers hoprtt Streaming and main.go:90 passes no stdin.
- BR-49 — not-addressed — Re-read issue 210 at HEAD - it records BR-34/25/38/39 and still neither the missing window-length field nor the uncapped perf-captures.jsonl.
- BR-53 — addressed — Mutation-verified - replacing has_ui() with `true` at nvim/init.lua:4060 takes pair-doctor-test.sh red on both the n/a render and the unknown verdict.
- BR-54 — addressed — Mutation-verified - dropping the note prefix at nvim/init.lua:4210 takes the two sidecar-content assertions red; the test opens the file the payload names.
- BR-55 — addressed — Mutation-verified - replacing the `if not pair_write_data_file(...)` guard with a bare call takes the notify assertion red.
- BR-59 — addressed — All three members reverted independently in a scratch tree; each took pair-doctor-test.sh red (2, 2 and 1 failures). The rule was applied as a class, not per-site.
- BR-60 — not-addressed — nvim/doctor.lua:2 and nvim/doctor_test.lua:3 still claim no vim API while :293 calls vim.empty_dict; atlas/index.md:39 still enumerates the sidecar without the note, and doctor/SKILL.md's "compact report, joined per-process rates, and both raw ps samples" is a third member of the same enumeration.

### Raised

- **BR-61** [Important] `untested-shell-surface` Four of perf.sh's five external-tool parsers are pinned only by exit-1 stubs, so a wrong field index ships as a plausible headline reading
  This is the 8th finding in family `untested-shell-surface`. Do NOT fix this
  instance. The rule covering it: a shell parser over external-tool output is
  pinned by a RECORDED-OUTPUT fake of that tool - an `exit 1` stub pins the n/a
  ladder and asserts nothing about field extraction. The enumeration is
  mechanical, every awk in perf.sh that indexes a tool's stdout, and it has five
  members: sample()'s procs row (perf.sh:150, pinned at perf_test.sh:98 and
  :165), cputimes()'s row (perf.sh:162, unpinned), top's CPU-usage line
  (perf.sh:115, unpinned), top's WindowServer row (perf.sh:117, unpinned), and
  iostat's three columns (perf.sh:238, unpinned). Mutation-verified
  independently, full suite green after each: changing cputimes()'s tab to a
  space moves the join from rates=5/unmeasured=0 to rates=0/unmeasured=5, so
  SKILL.md's step 4 and the operator's own Spec addition silently vanish;
  $(NF-1)->$2 makes cpu_idle_pct carry the USER percentage under the idle key, a
  HEADLINE_KEY the prompt carries and SKILL.md instructs the reader on; permuting
  iostat's $1,$2,$3 mislabels all three disk values. The mechanism already exists
  at perf_test.sh:98-111 - the sweep is a recorded-output fake for top, iostat
  and vm_stat plus cputime-row assertions on the existing ps fake (ARCH-MOCK,
  ARCH-PURPOSE). Prevalence 8/8 with BR-9, BR-25, BR-26, BR-36, BR-38, BR-52,
  BR-59.

## Open findings

- **BR-16** [Minor] `failure-reported-as-measurement` perf.sh discards hoprtt's sample count, so a truncated pipe run reads identically to a full one
- **BR-18** [Minor] `unguarded-edge-case` PAIR_PERF_WINDOW flows unvalidated into sleep and awk -v, and is undocumented in atlas/README
- **BR-19** [Minor] `boundary-hygiene` 4f9365b3 (M2.2b, the pure join) landed inside the M1 boundary; note it so M2's base is not mistaken for the branch point
- **BR-26** [Minor] `untested-shell-surface` perf_test.sh:23's stray-line check only fires for all-digit lines, so any other unattributable line passes
- **BR-27** [Minor] `duplicated-logic` Four shapes of collect()'s availability/failure/empty ladder in one file, which is how the sample blocks escaped the rule
- **BR-28** [Minor] `report-line-contract` perf.sh:163's swap_rate is shell arithmetic, contradicting the file's own rule 2, and divides by WINDOW unguarded
- **BR-29** [Minor] `failure-reported-as-measurement` A vm_stat that exists but exits non-zero renders `swap=n/a (vm_stat unavailable)`, misnaming the failure
- **BR-31** [Minor] `unguarded-edge-case` delta detects a reused pid only through etime, so an unparseable etime lets a reused pid produce a negative cpu_pct
- **BR-37** [Minor] `build-artifact-committed` perf_test.sh's mktemp fallback writes .perf-test-stub.$$ into the worktree and nothing gitignores it
- **BR-39** [Important] `failure-reported-as-measurement` swap_rate divides by WINDOW even when sample_b was shed and the sleep never ran, reporting a rate over time that did not pass
- **BR-41** [Minor] `unguarded-edge-case` pair hoprtt silently ignores unrecognised arguments and runs the 500-sample pipe probe instead
- **BR-43** [Minor] `injected-io-seam-bypassed` hoprttcmd.Run takes injected writers but child() reads os.Stdin and writes os.Stdout, and the package is registered as a streaming subcommand with no stdin
- **BR-49** [Minor] `traceability` The M2.6 deferrals are recorded only in the plan, which archives at close — issue 210 records neither
- **BR-60** [Minor] `docs-gate` doctor.lua and doctor_test.lua still claim "no vim API", and atlas's sidecar enumeration omits the note added in the same commit
- **BR-61** [Important] `untested-shell-surface` Four of perf.sh's five external-tool parsers are pinned only by exit-1 stubs, so a wrong field index ships as a plausible headline reading
