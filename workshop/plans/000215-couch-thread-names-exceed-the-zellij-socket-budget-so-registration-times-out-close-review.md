# Boundary Review — pair#215 (whole-issue close)

| field | value |
|-------|-------|
| issue | 215 — couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline |
| repo | pair |
| issue file | workshop/issues/000215-couch-thread-names-exceed-the-zellij-socket-budget-so-registration-times-out.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9bd5250d9d18588e2e720825ad50fd9f80e8fe1a..1436413645785b9599a179d73c9884c508aa0ea8 |
| command | sdlc close --issue 215 |
| reviewer | claude |
| timestamp | 2026-09-08T21:57:43-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The #215 half of this window is unusually well-built: I verified all three claimed fixes by mutation in a scratch copy, and each one goes red without the fix, reproducing the exact numbers the issue quotes (52 probes with the naive per-candidate acceptor, 1→7 with the eager-discovery cut the operator caught, and the error-first ordering in the diagnosis). The lazy-discovery design is the right shape — it is cheaper than both the pre-#215 code and the first cut, and the two-test split ("bounded" and "cheap" are different claims) is exactly the lesson the regression taught. What keeps this from SHIP is one in-diff contract drift: `discoverSessionNameBudget`'s documented-fallback constant (`defaultSessionNameBudget`, "a MESSAGE default only — never an acceptance test") now becomes the arithmetic acceptance budget for every remaining candidate, and on a machine with a tight socket path that turns a recoverable ladder walk into a launch that cannot succeed. Three further Importants concern the measurement/docs surface handed to #218 rather than the fix itself.

Scope note: the pinned window's base (`9bd5250d`) predates #215's first commit, so it also contains the whole of #199 (merged as PR #115) plus #216/#217 — ~4,700 of the ~5,000 changed lines. That work crossed its own close gate over many rounds; I reviewed the #215 range (`36a37b4a..14364136`) on the merits and only sanity-scanned the rest. I verified #215's claim that #199 already fixed the probe-orphan leak (`probes/*/main.go` are `os.Exit(run())` with `defer session.Close()`, `Close` runs `zellij delete-session --force`, and `TestNoProbeExitsPastItsOwnCleanup` enforces it at the AST level) — that claim holds.

Build is clean (`go build ./...`, `go vet`). `go test ./...` fails only in `cmd/couch` and `cmd/internal/couchcmd`, all 11 failures tracing to `operation not permitted` on pty allocation — this review agent's environment, not the diff.

## 1. Strengths

- **`sessionNameAcceptor` (createflow.go:951–967)** — the lazy trigger is the load-bearing insight, and the doc comment states *why* (a rejection is the only evidence a ladder walk is underway) rather than what. Hoisting the acceptor outside the tag loop in `assignLaunchSessionNames` (createflow.go:310) costs nothing precisely because discovery is lazy, so N tags share one budget measurement.
- **The two-test split is real, not decoration.** I reverted to the naive acceptor: `TestAssignSessionNameCostsABoundedNumberOfProbes` reports 52 at 25 owned suffixes and 122 at 60. I reverted to the eager cut: `TestResumingAKnownThreadCostsOneProbe` reports 7. Both numbers match the issue's tables exactly.
- **Asserting O(1) as invariance across index sizes** (session_index_test.go:255) rather than as a bound is the correct oracle — a bound alone passes something that grows slowly, and this index only grows.
- **`TestAssignSessionNameCostsABoundedNumberOfProbes` exercises the production acceptor**, not a hand-rolled one, and says so (session_index_test.go:225–228). Counting `accepts` calls would have measured ladder iterations instead of subprocesses — the wrong quantity.
- **`pairRegistrationTimeout` (couch.go:273–289)** is a model ARCH-CONSTRAINTS entry: measurement (8.85 s), basis (a real pty launch on a calm machine), the ratio chosen, and — the part that usually goes missing — the *validity condition*, naming that raising the number is the wrong fix if unbounded work moves back inside the window.
- **`diagnoseRegistrationFailure` (launch_existing.go:207–230)** picks the one observation couch already had, takes it only after the deadline so the happy path pays nothing, and gets the error/present ordering right for a non-obvious reason (a missing index entry surfaces as an *error*, and it is the most diagnostic case).

## 2. Critical findings

None.

## 3. Important findings

**I1 — `defaultSessionNameBudget` becomes an acceptance test in the fallback path, against its own stated contract** (`cmd/internal/launcher/createflow.go:964`; contract at `:913-916`, ladder contract at `session_index.go:285-288`)

`sessionNameAcceptor` stores whatever `discoverSessionNameBudget` returns and judges every later candidate arithmetically. But that function has a fallback: if `accepts(pad(13))` fails it returns `defaultSessionNameBudget` (24) — a *guess*, documented at `:915` as "a MESSAGE default only — never an acceptance test", and pinned as a fallback by `session_name_scheme_test.go:271`.

Failure scenario: on a machine whose socket directory is long enough that zellij refuses a 13-byte name (macOS leaves ~25; a longer username or a different `TMPDIR` narrows it), the first candidate is probed and rejected, discovery falls back to 24, and every subsequent candidate ≤ 24 bytes is accepted arithmetically though zellij refuses all of them. `AssignSessionName` returns the *longest* remaining rung; `createflow.go:379` then rejects it with "Pick a shorter tag" — advice that cannot work, because the ladder is what shortens. Pre-#215 the probe judged each rung independently and could descend to `minSessionRepoBytes` (≈11 bytes total) and succeed, or return `SessionNameExhausted`. So a machine where pair worked now cannot start a session at all. Nothing covers this branch.

Fix sketch: make `discoverSessionNameBudget` distinguish measured from fell-back (`(int, bool)`, or return 0 on fallback) and have the acceptor keep probing per candidate when the budget is unknown — slow, but correct, and only on the machine that would otherwise be broken. Update the `:913-916` comment either way, since the diff makes it false.

**I2 — the O(1) claim and the arithmetic-vs-zellij agreement are pinned only against `fakeRuntime`** (ARCH-MOCK; `cmd/internal/launcher/createflow_test.go:166-175`, `Makefile.local:82-88`)

The whole fix rests on "a candidate's length is a local fact", but nothing runs it against real zellij. `sessionNameRejected` (`zellijparse.go:48`) matches one substring of zellij's stderr, and #215 makes that seam far more load-bearing: previously a missed rejection mis-accepted one candidate; now one wrong budget decides every remaining candidate without asking. `make test-live` / `test-couch-zellij-live` cover quiescence, detach and quit — nothing covers the name budget.

Fix sketch: add a `PAIR_LIVE_COUCH`-gated conformance test that runs `discoverSessionNameBudget` against the real `OSRuntime` and asserts the boundary — a name at `budget` bytes is accepted and one at `budget+1` is rejected. That is the check that catches a zellij message change before it silently disables the oracle.

**I3 — the zellij tracer injects ~50 ms per call into the startup it exists to measure** (`probes/zellijcalls/shim.sh:41-46`)

Every traced call spawns `python3` twice for a timestamp. Measured on this machine: ~25 ms per `python3 -c` start, so ~50 ms per zellij call plus a bash fork, and the shim drops the `exec` (it runs the real zellij as a child) when armed. `SKILL.md`'s "Reading it" section tells the operator to compare `in-zellij time` against `wall span` — but the python cost lands *outside* each measured duration and *inside* the span, so it inflates exactly the denominator the conclusion turns on. At the call volume #215 already measured (52 probes for one name assignment alone), that is seconds of injected latency in an 8.85 s budget, in a tool whose entire purpose is telling #218 where those 8.85 s went. #215's own method lesson applies: this would refute or confirm a hypothesis with a distorted instrument. It has not been run yet, so it is cheap to fix now.

Fix sketch: use bash's `EPOCHREALTIME` (`${EPOCHREALTIME}`, bash ≥5 — the shebang is `env bash`, so guard and fall back), and state the residual per-call overhead in `SKILL.md` so the report is read with it in view.

**I4 — atlas update appears missing for the shell/PATH-shim probe class** (`atlas/index.md:17-49`; `probes/zellijcalls/`)

The atlas states the probe-home rule and its exceptions at length, and closes with: *"'probes live in `probes/`' is true of every probe that runs with no arguments and needs nothing internal; anything else goes under `cmd/probes/` **with its own target in the same commit**, since nothing will run it otherwise."* `probes/zellijcalls/` fits neither branch — it takes `arm|report|disarm`, needs an interactive shell, has no Go code, and has no target. `test-smoke` (`Makefile.local:59-67`) does not break on it (I confirmed under `/bin/sh`: exit 0), but it prints `==> probes/zellijcalls/ (shared harness, not a probe)` — a category the directory does not belong to. The atlas passage records itself as written *because* #199's first probe landed in the wrong home; this is the next instance.

Fix sketch: one paragraph in `atlas/index.md` naming the third branch (an operator-driven tracer that cannot be auto-run, and why it still lives under `probes/`), and either widen `test-smoke`'s skip message or have it recognise a `SKILL.md`-only directory as operator-driven.

## 4. Minor findings

- `promptForTag` (`createflow.go:713-716`) still hand-rolls "probe, then discover the budget" — the pair `sessionNameAcceptor` now encapsulates (ARCH-DRY). Cost is fine (refusal path only), but there are now two encodings of one idea; an acceptor that can also surrender its measured budget would collapse them.
- Nothing pins that `diagnoseRegistrationFailure`'s output actually reaches the operator: it is unit-tested in isolation, and `launch_existing.go:123-124` (the `%s` that wires it in) has no test. Reachable, not dead — but this issue's own Log records a path regressing precisely because nothing asserted it.
- The `sessions, ok := c.Artifacts.(PairSessionIO)` false branch ("no session observer") is uncovered by `registration_diagnosis_test.go`.
- When `observeErr != nil` for a reason other than an absent binding (a corrupt or unreadable index), the message still asserts "NO Pair session … Pair never started" before quoting the error. Inconclusive observation stated as a conclusion (ARCH-SECURE: degrade visibly rather than substitute).
- `resumeRegistrationTimeout` is now initialised to `pairRegistrationTimeout` (`couch.go:114`), so the resume/cold branch at `launch_existing.go:111-114` is vestigial in production and exists only as a test seam. Worth a one-line comment saying so; the cold path has no equivalent seam, which is why no test can exercise a cold registration timeout without waiting 15 s.
- `trace.sh:15` writes a fixed `${TMPDIR:-/tmp}/pair-zellij-trace.tsv`, truncated with `: >` on arm. On a Linux box with no `TMPDIR` that is a predictable name in a shared directory. `arm` also prepends a repo directory to the interactive shell's `PATH` with only a manual `disarm` — both opt-in, neither stated.
- `trace.sh:32` builds a `grep -v '^$DIR$'` pattern by interpolation; a repo path containing regex metacharacters would over-match on disarm. `shim.sh:45` logs `"$*"`, losing argv quoting, which `trace.sh`'s `kind()` then re-splits on whitespace.
- `sessionNameAcceptor`'s closure carries mutable `budget` across calls with no documented single-goroutine constraint (ARCH-ORDER). Both callers are sequential today.

## 5. Test coverage notes

- Mutation-verified, all three: naive acceptor → `TestAssignSessionNameCostsABoundedNumberOfProbes` red at 52/122; eager discovery → `TestResumingAKnownThreadCostsOneProbe` red at 7; error-first ordering → `TestARegistrationTimeoutSaysWhetherPairStarted/pair_never_started` red on all three substrings. No fix here is unpinned.
- Uncovered: the `discoverSessionNameBudget` fallback reaching `sessionNameAcceptor` (I1 — this is the finding's whole point); the `PairSessionIO` assertion failing; the diagnosis's wiring into `launchTrackedThread`; any live check of the budget model (I2).
- Environment: `cmd/couch` and `cmd/internal/couchcmd` fail here on pty allocation only. The operator should confirm a full `make test` on a pty-capable shell before closing — but nothing in this diff touches those paths.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (Minor above). One duplicated probe-then-discover pair at `createflow.go:713`.
- **ARCH-PURE** — pass. `sessionNameFits` is pure and takes the limit as a parameter; `AssignSessionName` is pure given `accepts`; the only IO is behind the `Runtime` seam, and both new launcher tests run with no IO. `diagnoseRegistrationFailure` makes one IO call inside a formatting function, but through an injected interface with a stateful fake.
- **ARCH-PURPOSE** — largely pass. The operator explicitly rescoped ("#215 is about unblock"), and the latency half went to #218 carrying its measurements and its tracer — a separable extension, not the deferred point. The class was swept rather than the site: all three `AssignSessionName` call sites in the assignment path took the new acceptor. Flag: two "Done when" bullets are now inaccurate (see §7).
- **ARCH-MOCK** — flag (I2). The fake is the only oracle for a claim about a real binary's behavior.
- **ARCH-CONSTRAINTS** — pass on `pairRegistrationTimeout`, which is a model entry. Flag on the tracer (I3): an instrument whose unstated self-cost exceeds the effects already ruled out by measurement (nvim at 117 ms). For #218, the shim's overhead must be subtracted or eliminated before its numbers are quoted as evidence.
- **ARCH-SECURE** — flag (Minor). Fixed trace path in a possibly-shared directory; `PATH` armed into the interactive shell with manual disarm.
- **ARCH-ORDER** — pass. The acceptor's state is one int with two legal states and a one-way transition, readable off the code. The one conflation: "not yet measured" and "measured as zero" are the same value, so a future `discoverSessionNameBudget` returning 0 would silently revert to per-candidate probing. `(int, bool)` fixes that and I1 together.

## 7. Plan revision recommendations

The issue is well-logged, but it was **overwritten rather than revised** — the root-cause correction, the WON'T-DO on the suffix walk, and the operator's scope reduction all landed as `## Log` subsections while the `## Problem` and `## Done when` sections were rewritten in place (commit `4663fb70`, which also clobbered the frontmatter). AGENTS.md asks for an appended `## Revisions` section with timestamp + reason + delta. Recommend adding one to `workshop/issues/000215-*.md` carrying:

- **Root cause corrected (2026-09-08)** — filed as "session name exceeds the socket budget"; refuted by measurement, the real defect is the *cost* of the ladder. Delta: `## Problem` and the H1 rewritten; filename slug deliberately left asserting the refuted cause.
- **Plan item 2 withdrawn (2026-09-08)** — "start the walk from the highest known suffix" made obsolete by item 1; the walk is now 66–144 µs with probes flat at 8. Delta: item ticked as WON'T DO, with the reopen condition (approaching the 100-suffix ceiling).
- **Scope reduced by operator (2026-09-08)** — #215 is the unblock; the unexplained 8.85 s startup and its tracer move to #218. Delta: **amend `## Done when` accordingly** — bullet 3 ("couch starts a thread on `../pair/` on a loaded machine") is asserted by test rather than by a live run, and bullet 4 ("Over-long candidates are rejected without a subprocess") is not literally met: the first over-long candidate still costs one probe plus ~7 for discovery. That trade-off is deliberate and well-argued in `createflow.go:930-950`; the Done-when text should say what shipped (8 probes per cold assignment, 1 per resume, invariant in the index size) rather than a criterion the code intentionally does not meet.

```findings
findings:
  - id: new
    severity: Important
    family: fallback-guess-used-as-oracle
    title: |
      defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
    detail: |
      createflow.go:964 stores whatever discoverSessionNameBudget returns, including its
      documented fallback guess of 24 (":913-916" says "a MESSAGE default only -- never an
      acceptance test"). On a machine whose socket path is tight enough that zellij refuses a
      13-byte name, discovery falls back to 24, every later candidate <= 24 bytes is accepted
      arithmetically though zellij refuses all of them, and AssignSessionName returns the LONGEST
      rung. createflow.go:379 then fails with "Pick a shorter tag" -- advice that cannot work,
      since the ladder is what shortens. Pre-#215 the probe judged each rung independently and
      could descend to minSessionRepoBytes and succeed. Fix: return (int, bool) from
      discoverSessionNameBudget and keep probing per candidate when the budget is unknown; update
      the :913-916 comment either way. No test covers this branch.
  - id: new
    severity: Important
    family: fake-is-the-only-oracle
    title: |
      the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
    detail: |
      ARCH-MOCK. The fix rests on "a candidate's length is a local fact", but nothing runs it
      against real zellij. sessionNameRejected (zellijparse.go:48) matches one substring of
      zellij's stderr, and #215 makes that seam far more load-bearing: a missed rejection used to
      mis-accept one candidate, now one wrong budget decides every remaining candidate without
      asking. make test-live covers quiescence, detach and quit -- nothing covers the name budget.
      Add a PAIR_LIVE_COUCH-gated test asserting the boundary against OSRuntime: budget bytes
      accepted, budget+1 rejected.
  - id: new
    severity: Important
    family: probe-overhead-distorts-measurement
    title: |
      the zellij tracer injects ~50ms per call into the startup it exists to measure
    detail: |
      probes/zellijcalls/shim.sh:41-46 spawns python3 twice per traced call for a timestamp
      (~25ms each, measured here) plus a bash fork, and drops the exec when armed. SKILL.md tells
      the operator to compare in-zellij time against wall span -- but the python cost lands
      outside each measured duration and inside the span, inflating exactly the denominator the
      conclusion turns on. At the call volume #215 already measured this is seconds injected into
      an 8.85s budget, in the instrument #218 will use to find where those seconds go. Not yet
      run, so cheap to fix: use bash EPOCHREALTIME with a fallback, and state the residual
      per-call overhead in SKILL.md.
  - id: new
    severity: Important
    family: new-surface-without-atlas
    title: |
      atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
    detail: |
      atlas/index.md:17-49 states the probe-home rule and closes with "anything else goes under
      cmd/probes/ with its own target in the same commit, since nothing will run it otherwise".
      probes/zellijcalls/ fits neither branch: it takes arm|report|disarm, needs an interactive
      shell, has no Go code and no target. test-smoke does not break on it (confirmed under
      /bin/sh, exit 0) but prints "shared harness, not a probe" -- a category it does not belong
      to. That atlas passage records itself as written because #199's first probe landed in the
      wrong home; this is the next instance.
  - id: new
    severity: Minor
    family: duplicated-probe-then-discover
    title: |
      promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
    detail: |
      createflow.go:713-716. ARCH-DRY. Cost is fine (refusal path only), but there are now two
      encodings of one idea; an acceptor that can also surrender its measured budget collapses
      them, and would also carry the fallback-vs-measured distinction I1 needs.
  - id: new
    severity: Minor
    family: fix-not-pinned-at-its-call-site
    title: |
      nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
    detail: |
      The function is unit-tested in isolation; launch_existing.go:123-124, the %s that wires it
      into the user-visible error, has no test. Reachable, not dead -- but this issue's own Log
      records a path regressing precisely because nothing asserted it. The PairSessionIO
      assertion's false branch is also uncovered.
  - id: new
    severity: Minor
    family: inconclusive-observation-stated-as-conclusion
    title: |
      an unreadable session index is reported as "NO Pair session -- Pair never started"
    detail: |
      launch_existing.go:226-229. When observeErr is a read or corruption failure rather than an
      absent binding, the message still asserts the conclusion before quoting the error.
      ARCH-SECURE: degrade visibly rather than substitute a fabricated answer downstream reads as
      evidence.
  - id: new
    severity: Minor
    family: vestigial-branch-reads-as-policy
    title: |
      resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
    detail: |
      couch.go:114 initialises it to pairRegistrationTimeout, so launch_existing.go:111-114 gives
      both paths 15s in production while reading as if they differ. Worth a comment. The cold path
      has no equivalent seam, so no test can exercise a cold registration timeout without waiting
      15s.
  - id: new
    severity: Minor
    family: predictable-path-in-shared-directory
    title: |
      trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
    detail: |
      trace.sh:15 truncates a predictable name with ": >"; on a Linux box with no TMPDIR that is a
      shared directory. arm prepends a repo directory to the interactive shell's PATH with only a
      manual disarm. Both opt-in developer surface, neither stated. Also trace.sh:32 interpolates
      $DIR into a grep pattern, and shim.sh:45 logs "$*", losing argv quoting that kind() then
      re-splits on whitespace.
  - id: new
    severity: Minor
    family: issue-overwritten-not-revised
    title: |
      the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
    detail: |
      AGENTS.md asks for an appended "## Revisions" section (timestamp + reason + delta) rather
      than overwriting; 4663fb70 rewrote the body in place and clobbered the frontmatter. Two
      "Done when" bullets are now inaccurate: the loaded-machine criterion is asserted by test
      rather than a live run, and "over-long candidates are rejected without a subprocess" is not
      literally met -- the first one costs 1 probe plus ~7 for discovery. That trade-off is
      deliberate and well-argued; the criterion should say what shipped.
```

---

## Re-review — 2026-09-08T23:45:44-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 215 — couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline |
| repo | pair |
| issue file | workshop/issues/000215-couch-thread-names-exceed-the-zellij-socket-budget-so-registration-times-out.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9bd5250d9d18588e2e720825ad50fd9f80e8fe1a..290c5e8c9e2c7ed5dea441eed300b75d95d1a60c |
| command | sdlc close --issue 215 |
| reviewer | claude |
| timestamp | 2026-09-08T23:45:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The four prior Importants (BR-1..BR-4) are genuinely fixed and, where I could check cheaply, pinned by tests that go red when the fix is reverted — I mutation-verified BR-1 (forcing `measured = true` turns `TestAnUnmeasurableBudgetKeepsProbingInsteadOfTrustingTheFallback` red) and the O(1) claim (disabling the arithmetic short-circuit gives 59 probes at 25 owned suffixes / 129 at 60, and both assertions fire). The live conformance test BR-2 asked for exists, is wired into `make test-couch-zellij-live`, and passes here (measured budget 57 bytes on this machine; zellij 0.44.3's refusal text still matches `sessionNameRejected`). What blocks SHIP is that BR-1's fix closed the *instance*, not the *class*: `discoverSessionNameBudget` still returns `measured=true` when its binary search saturates at its own ceiling of 64, so on a machine whose real budget exceeds 64 — the Linux `~/.cache/zellij` case the atlas explicitly names — a name zellij would accept is now refused arithmetically and silently shortened, which is a regression from the pre-#215 per-candidate probe. Alongside that: the one atlas passage that documents this exact contract still asserts the pre-#215 version of it, the tracer's self-overhead report measures nothing (it reported **-518µs/call**; the real cost is **~3.4 ms/call**), and §4's lessons.md step was skipped after a 10-finding review round. Nothing here is Critical and none of it is expensive.

Window note: the pinned range spans the `#199` merge plus `#216`/`#217`, which had their own boundary reviews. I scoped findings to `#215`'s surface and confirmed the rest builds, vets clean, and passes (`go test ./...` failures are all the documented `ptychild: operation not permitted` sandbox class — one of them even self-labels it).

## 1. Strengths

- **`cmd/internal/launcher/createflow.go:955-982`** — the lazy acceptor is the right shape and the comment earns its length: it names both questions that wore one signature, and why answering them eagerly cost the resume path 7×. Sharing one acceptor across tags at `createflow.go:311` (outside the loop) is the correct placement.
- **`cmd/internal/launcher/session_index_test.go:263-299`** — two tests for "bounded" and "cheap" as separate claims, with the second existing *because* satisfying the first broke it. That is the failure mode written into the suite, not just the commit message.
- **`cmd/internal/launcher/session_name_budget_live_test.go:60-72`** — asserting against zellij's **raw** output rather than `ProbeSessionName`'s wrapper, with the trap recorded in the comment. That distinction is exactly what makes this a conformance check instead of a spelling check.
- **`cmd/internal/couchcore/couch.go:273-288`** — `pairRegistrationTimeout` states the measurement, the basis, *and* the condition that invalidates it ("if anything unbounded moves back inside this window, raising the number is the wrong fix"). ARCH-CONSTRAINTS done properly.
- **`probes/zellijcalls/main.go:37-49`** — making the smoke-suite safety structural (argv[0] must be `zellij`) rather than an exemption in a list. Verified: `go run ./probes/zellijcalls` prints and exits 0.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `cmd/internal/launcher/createflow.go:1008` — the binary search reports its own ceiling as a measurement.** *(2nd in family `fallback-guess-used-as-oracle`.)*

`lo, hi := len(sessionNameProbeMarker), 64`. BR-1 added the `measured` bool for the **low** end (shortest probe refused → `false`). The **high** end has no equivalent: when the true budget is ≥ 64 the search saturates and returns `(64, true)`, and the acceptor then judges every candidate against a number that is a search bound, not an observation. Verified:

```
discoverSessionNameBudget(func(n string) bool { return len(n) <= 100 })  →  budget=64 measured=true
acceptor(70-byte name)                                                  →  false   (zellij would accept it)
```

Reachable: `~/.cache/zellij/<version>/` on Linux leaves well over 64 bytes of the 108-byte `sun_path`, and `atlas/session-identity.md:184` already names Linux as the case where the number differs. Consequence is the one the ladder's own design notes forbid — silent shortening ("silent truncation is what produced `pair-parley_nv-parley_nv`"), and a regression from pre-#215, where each candidate was probed and a 70-byte name was accepted. The live test would catch it (`probe(pad(budget+1))` returns true there), but it has only ever run on macOS.

**Do not fix only this site.** The rule BR-1 and I1 are both instances of: *a number that did not come from an observation must not be usable as an acceptance oracle, and the discovery function is the only place that knows which it is — so every way discovery can fail to observe must be reported through the same `measured` channel.* The enumeration is small and closed: `discoverSessionNameBudget` has exactly three exits — refused at `lo` (reports `false`, correct), converged inside `(lo, hi)` (correct), and saturated at `hi` (currently reports `true`, wrong). Sketch: `return lo, lo < hi_initial`, or raise `hi` and return `false` on saturation; either way add the third case to `TestDiscoverSessionNameBudget`, which currently tests only two of the three.

**I2 — `atlas/session-identity.md:185-187` — the atlas states the contract #215 replaced.** *(2nd in family `new-surface-without-atlas`.)*

> "Pair therefore treats zellij's own validator as the acceptance oracle (`ProbeSessionName`); the numeric budget is calibrated lazily, only after a rejection, **purely so the refusal message can quote real numbers**."

After #215 the budget *is* the acceptance oracle for every candidate past the first rejection. The passage is not merely incomplete — it asserts the negation of what shipped, in the one file a reader goes to for this contract.

BR-4 was "new surface with no atlas entry"; this is "changed surface whose atlas entry now lies". Same rule: *at a boundary, the atlas must derive from the code for surface the diff touched — and the sweep is the grep, not the recollection.* The enumeration for this issue is greppable and I ran it: `grep -rn "ProbeSessionName|acceptance oracle|calibrated lazily|24 bytes" atlas/` → one hit block (`session-identity.md:184-187`); `grep -rn "registration" atlas/couch.md` → no stale deadline number. So the sweep is one passage; it should also gain the `measured`-vs-fallback distinction, since that is now a documented behavioural fork.

**I3 — `probes/zellijcalls/main.go:114-140` — `reportOverhead` measures neither arm of what it claims to compare.** *(2nd in family `probe-overhead-distorts-measurement`.)*

Both arms `exec` the real zellij in-process; the only difference is a call to `record(...)`, which `trace.sh:38` explicitly disables by setting `PAIR_ZELLIJ_TRACE=""`. So the "shimmed" arm and the "direct" arm are the same operation, and the shim's actual cost — a second Go process exec plus `realZellij()`'s `EvalSymlinks` walk over every PATH entry — is never on either side of the subtraction. Measured on this machine:

```
probes/zellijcalls/trace.sh overhead   →  direct 10.10ms  shimmed 9.58ms  overhead -518.557µs/call
end-to-end (shim named `zellij` on PATH, 25 calls x2)
                                       →  direct 9.26/8.77ms   shimmed 12.49/12.30ms   ≈ +3.4ms/call
```

A negative overhead is the report telling you it is noise. `SKILL.md` publishes "**under 1ms per call** … below the noise floor of a 10-call sample" from it; the true figure is ~3.4 ms, i.e. 8–17% of a 20–45 ms zellij call, landing (exactly as in BR-3) *outside* each recorded duration and *inside* the wall span — the ratio `## Reading it` turns on. At the ~52 calls #215 measured that is ~0.18 s injected into an 8.85 s budget: small, but the *claim of smallness* is unfounded, and #218 will read this report.

The rule covering BR-3 and I3: *an instrument's overhead must be measured through the same entry point production traffic takes — spawn the built artifact, do not re-run its inner steps in the measuring process — and the measured interval must contain the injected cost, or it is measuring something else.* Concretely: build `$BIN/zellij`, `exec.Command(shimPath, "--version")` for the shimmed arm against `exec.Command(real, "--version")` for the direct arm, with `PAIR_ZELLIJ_TRACE` pointed at a real file so the append is included. Then restate the SKILL.md number.

**I4 — `probes/zellijcalls/main.go:44` — the argv[0] guard, whose stated value is preventing a leaked server per `make test-smoke`, is pinned by nothing.** *(2nd in family `fix-not-pinned-at-its-call-site`.)*

`atlas/index.md` now says this is safe "**by construction, not by exemption**", and the commit message calls it a structurally-closed hazard. Delete the guard and every test in the tree still passes — the smoke loop would run a bare `zellij` and the damage (a leaked server, or an obscure non-tty error) shows up nowhere a suite reads. Note also that `--overhead` is handled at `main.go:34`, *before* the guard, so the guard's position is load-bearing and unstated.

Same rule as BR-6: *a guard whose value is that it prevents something needs a test that goes red when the guard is removed; "reachable" is not "asserted".* The enumeration for this issue is two sites — this one, and BR-6's still-open `launch_existing.go:123-124`. Both are cheap: a `TestShimDoesNothingWhenNotInvokedAsZellij` calling `run()` with a rewritten `os.Args[0]`, and one assertion that `launchTrackedThread`'s registration-timeout error string contains the diagnosis suffix.

**I5 — `workshop/lessons.md` — no entry from #215, after a close review that produced ten findings.** *(family `lesson-recorded-only-in-the-issue`.)*

AGENTS.md §4: "When you run code review, add rules to `workshop/lessons.md` that prevent the mistakes you found." The file gained 251 lines in this window, all from #199 (`git log -S` confirms; the most recent entries are `#199 M3`'s). #215 added none — and it has at least two entries in exactly the house format, currently living only in the issue `## Log`, which archives to `workshop/history/` ("don't read unless asked — low signal"):

- *"Bounded" and "cheap" are different claims, and a fix for either can quietly pay for it out of the other* — with the measured 1→7 resume regression as the case.
- *Verify the artifact the code produces, not the hypothesis you authored* — the wrong root cause, filed twice in one investigation from a hand-composed string when `AssignSessionName` was exported and answered it in six lines.

I1's own root cause is a third candidate: *marking one non-observation as unmeasured does not close the class; enumerate every exit the discovery function has.*

## 4. Minor findings

- `probes/zellijcalls/trace.sh:11` — the header runbook says `probes/zellijcalls/trace.sh disarm` without `eval`, so following the comment prints the `unset`/`export` lines instead of applying them and leaves PATH armed. `SKILL.md` gets it right; the two should agree.
- `session_name_budget_live_test.go:41-43` re-derives `pad()` byte-for-byte from `createflow.go:1005-1007`. A change to the padding scheme takes the conformance test with it silently. ARCH-DRY — export the helper or have the test call it.
- `createflow.go:957-958` — `budget`/`measured`/`attempted` is 3 fields carrying 8 representable states across calls where 3 are legal (`unknown` → `measured(n)` | `unmeasurable`). No bug (single-threaded, extent bounded to one assignment), but ARCH-ORDER's tagged form would make the legal set readable off the type.
- `probes/zellijcalls/main.go:88-92` records argv verbatim, and `zellij action write-chars` carries typed content into the trace file. 0600 and opt-in, so low risk, but `SKILL.md` should say the trace contains what was typed.

## 5. Test coverage notes

Mutation checks I ran, all in a scratch edit and reverted (tree confirmed clean afterward):

| mutation | result |
|---|---|
| `budget, _ = discover(...); measured = true` | `TestAnUnmeasurableBudget…` red — BR-1 genuinely pinned |
| `if false && measured` (disable arithmetic) | `TestAssignSessionNameCostsABounded…` red at 59/129 probes — O(1) claim real |
| `PAIR_LIVE_COUCH=1 … TestSessionNameBudgetMatchesRealZellijLive` | passes, budget 57 bytes, classifier matches zellij 0.44.3 |

Gaps, in priority order: the third exit of `discoverSessionNameBudget` (I1); the shim's argv[0] guard and the diagnosis's call-site wiring (I4/BR-6); and nothing exercises the acceptor when it is *shared across tags* in `assignLaunchSessionNames` — tag A measuring the budget changes how tag B's ledger short-circuit is answered (probe → arithmetic). That is correct under the documented "the ladder resolves LENGTH, never ownership" contract, but it is a new cross-tag coupling introduced at `createflow.go:311` with no test.

## 6. Architectural notes

- **ARCH-DRY** — flag (BR-5, still open): `promptForTag:713-717` hand-rolls the probe-then-discover pair the acceptor now encapsulates. Two encodings of one idea; an acceptor that could surrender its `(budget, measured)` collapses them *and* would carry the distinction I1 needs. Also the duplicated `pad()`.
- **ARCH-PURE** — pass. `sessionNameFits`, `sessionNameLadder`, `discoverSessionNameBudget`, `diagnoseRegistrationFailure` are all pure with the seam injected; the package's tests run without touching zellij except in the explicitly-gated live file.
- **ARCH-PURPOSE** — flag. The shadow-sweep of "the probe is the oracle" is incomplete in both directions: one consumer of the contract in code (I1's saturation exit) and one in prose (I2's atlas passage) still encode the pre-#215 model. This is the finding-answered-at-the-instance pattern the principle names.
- **ARCH-MOCK** — pass, and improved. The fake is no longer the only oracle: the boundary is asserted against `OSRuntime` and against zellij's raw stderr, in a target. This was the right fix for BR-2.
- **ARCH-CONSTRAINTS** — pass on the deadline (envelope stated with basis and invalidation condition, bounded work pinned by an invariance test); flag on I3, where the instrument built to find the remaining 8.3 s cannot see its own footprint.
- **ARCH-SECURE** — pass. BR-7's fix is the principle applied correctly: an unreadable observation now degrades visibly instead of being reported as the absent-binding verdict, and `registration_diagnosis_test.go` asserts the message stops short of one. Trace file is 0600 and per-invocation.
- **ARCH-ORDER** — pass with the Minor above. The acceptor's state is single-threaded with lexically bounded extent; `diagnoseRegistrationFailure` runs after `cancelRegistration()` and only on `DeadlineExceeded`, so a parent-cancelled launch (`context.Canceled`) correctly produces no diagnosis and no "waited 15s" overclaim.

## 7. Plan revision recommendations

Still the BR-10 set, unaddressed. The issue needs an appended `## Revisions` section (timestamp + reason + delta) rather than the three in-place `## Log` subsections, covering: the root-cause correction, the operator's 15 s decision, and the 2026-09-08 scope narrowing to "unblock" with latency split to #218. Two `## Done when` bullets should be restated to what shipped rather than left standing:

- "Over-long candidates are rejected without a subprocess" → the first rejection costs 1 probe plus ~7 for discovery; every candidate *after* it is free. The trade-off is well argued in the Log — the criterion should say it.
- "couch starts a thread on `../pair/` on a loaded machine" → asserted by the O(1) invariance test, not by a live loaded run. The Log already says this honestly; the criterion should match.

Add I1 as an open item or fix it in this round, and note that `estimate_hours:` is still empty in the frontmatter, which `sdlc close` will want.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: forcing measured=true turns TestAnUnmeasurableBudget red. See I1 for the unswept sibling exit.
  - id: BR-2
    disposition: addressed
    note: |
      Live test exists, is in make test-couch-zellij-live, and passes here (budget 57, classifier matches zellij 0.44.3).
  - id: BR-3
    disposition: addressed
    note: |
      python3 clock reads are gone and the shim is Go; the residual overhead CLAIM is a separate defect, raised as I3.
  - id: BR-4
    disposition: addressed
    note: |
      atlas/index.md now names the operator-driven instrument class and why it is safe under the smoke loop.
  - id: BR-5
    disposition: not-addressed
    note: |
      promptForTag:713-717 still hand-rolls probe-then-discover; the acceptor still cannot surrender its budget.
  - id: BR-6
    disposition: not-addressed
    note: |
      The no-observer branch is now covered, but launch_existing.go:123-124 (the wiring) still has no test. See I4.
  - id: BR-7
    disposition: addressed
    note: |
      Unreadable and absent are now distinct states with a test asserting the diagnosis stops short of a verdict.
  - id: BR-8
    disposition: not-addressed
    note: |
      couch.go:114 now reads pairRegistrationTimeout but no comment says the resume branch is a test seam only.
  - id: BR-9
    disposition: addressed
    note: |
      Per-invocation $$ path, exact-match disarm instead of a grep pattern, %q per arg, and PATH persistence stated in SKILL.md.
  - id: BR-10
    disposition: not-addressed
    note: |
      No "## Revisions" section; both overstating Done-when bullets still stand as written.
findings:
  - id: new
    severity: Important
    family: fallback-guess-used-as-oracle
    title: |
      discoverSessionNameBudget reports its own search ceiling of 64 as MEASURED, so a fitting name is refused arithmetically
    detail: |
      2nd in family. BR-1 added the measured bool for the LOW exit only; the HIGH
      exit still returns (64, true) when the search saturates. Verified:
      discoverSessionNameBudget(len<=100) yields budget=64 measured=true, and the
      acceptor then refuses a 70-byte name zellij would take -- a regression from
      the pre-#215 per-candidate probe, and the silent shortening the ladder's own
      design forbids. Reachable on Linux, where ~/.cache/zellij leaves well over 64
      bytes and which atlas/session-identity.md:184 already names as the differing
      case. Do NOT patch just this exit. The rule: a number that did not come from
      an observation must never be usable as an acceptance oracle, and discovery is
      the only place that knows which it is -- so EVERY way discovery can fail to
      observe reports through the same channel. The enumeration is closed at three
      exits (refused at lo; converged inside; saturated at hi) and
      TestDiscoverSessionNameBudget covers two of them.
  - id: new
    severity: Important
    family: new-surface-without-atlas
    title: |
      atlas/session-identity.md still asserts the pre-#215 contract -- that the budget is for the message only
    detail: |
      2nd in family. Lines 185-187 read "the numeric budget is calibrated lazily,
      only after a rejection, purely so the refusal message can quote real
      numbers". After #215 the budget IS the acceptance oracle for every candidate
      past the first rejection, so the passage asserts the negation of what
      shipped, in the file a reader consults for this contract. BR-4 was new
      surface with no entry; this is changed surface whose entry now lies. The
      rule: at a boundary the atlas must DERIVE from the code for surface the diff
      touched, and the sweep is a grep, not a recollection. I ran the enumeration
      -- grep -rn "ProbeSessionName|acceptance oracle|calibrated lazily|24 bytes"
      atlas/ returns exactly this one passage, and atlas/couch.md carries no stale
      deadline number -- so the sweep is one edit, which should also record the
      measured-vs-fallback fork.
  - id: new
    severity: Important
    family: probe-overhead-distorts-measurement
    title: |
      the tracer's self-overhead report compares two identical operations, and SKILL.md publishes a number from it
    detail: |
      2nd in family. reportOverhead (main.go:114-140) execs the real zellij in
      BOTH arms in-process; the only difference is record(), which trace.sh:38
      disables via PAIR_ZELLIJ_TRACE="". The shim's actual cost -- a second Go
      process exec plus realZellij's EvalSymlinks walk of PATH -- is on neither
      side. Measured: trace.sh overhead prints "overhead -518.557us/call" (a
      negative result is the report saying it is noise), while end-to-end through
      a shim named zellij on PATH gives 12.49/12.30ms against 9.26/8.77ms direct,
      about +3.4ms/call, or 8-17% of a 20-45ms zellij call, landing outside each
      recorded duration and inside the wall span -- the same structural defect as
      BR-3, one order of magnitude smaller. SKILL.md states "under 1ms per call,
      below the noise floor" on this basis, in the instrument #218 will use. The
      rule: an instrument's overhead must be measured through the same entry point
      production traffic takes -- spawn the built artifact, never re-run its inner
      steps in the measuring process -- and the injected cost must fall inside the
      measured interval.
  - id: new
    severity: Important
    family: fix-not-pinned-at-its-call-site
    title: |
      the argv[0] guard that prevents a leaked zellij server per make test-smoke has no test
    detail: |
      2nd in family. atlas/index.md now claims probes/zellijcalls is safe under the
      wholesale loop "by construction, not by exemption", and the commit calls the
      hazard structurally closed -- but deleting the guard at main.go:44 leaves
      every test in the tree green while make test-smoke starts running a bare
      zellij. Its position is also load-bearing and unstated: --overhead is handled
      at main.go:34, before the guard. Same rule as BR-6, which is still open: a
      guard whose value is that it prevents something needs a test that goes red
      when the guard is removed; reachable is not asserted. The enumeration for
      this issue is two sites -- this one and BR-6's launch_existing.go:123-124 --
      and both are one short test each.
  - id: new
    severity: Important
    family: lesson-recorded-only-in-the-issue
    title: |
      no workshop/lessons.md entry from #215, after a review round that produced ten findings
    detail: |
      AGENTS.md section 4 asks for lessons.md rules from what a review found. The
      file gained 251 lines in this window, all from #199 (git log -S confirms).
      #215 added none, though its Log already contains at least two in the house
      format: "bounded and cheap are different claims, and a fix for either can
      quietly pay for it out of the other" (the measured 1->7 resume regression),
      and "verify the artifact the code produces, not the hypothesis you authored"
      (the wrong root cause, filed from a hand-composed string when
      AssignSessionName was exported). I1's own cause is a third: marking one
      non-observation as unmeasured does not close the class. The issue Log
      archives to workshop/history, which agents are told not to read.
  - id: new
    severity: Minor
    family: runbook-disagrees-with-itself
    title: |
      trace.sh's header runbook omits the eval on disarm, so following it leaves PATH armed
    detail: |
      trace.sh:11 says `probes/zellijcalls/trace.sh disarm`, which prints the
      unset/export lines instead of applying them. SKILL.md's block uses eval. Two
      runbooks for one procedure that disagree on the step that undoes a PATH
      mutation.
  - id: new
    severity: Minor
    family: duplicated-probe-then-discover
    title: |
      the live conformance test re-derives pad() byte-for-byte from the production padding scheme
    detail: |
      session_name_budget_live_test.go:41-43 duplicates createflow.go:1005-1007. A
      change to the probe marker or padding alphabet takes the conformance test
      with it silently, which is the one test whose job is to disagree with
      production when production is wrong.
```
