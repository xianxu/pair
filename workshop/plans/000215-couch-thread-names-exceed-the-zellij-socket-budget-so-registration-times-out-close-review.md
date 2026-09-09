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

---

## Re-review — 2026-09-09T00:05:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 215 — couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline |
| repo | pair |
| issue file | workshop/issues/000215-couch-thread-names-exceed-the-zellij-socket-budget-so-registration-times-out.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9bd5250d9d18588e2e720825ad50fd9f80e8fe1a..55e8e66fe87addb9b809869e203cafd03e8856c0 |
| command | sdlc close --issue 215 |
| reviewer | claude |
| timestamp | 2026-09-09T00:05:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 2's five blocking findings are genuinely closed, and I verified each by the artifact rather than the message: BR-11's high-end guard goes red under mutation (`return hi, true` → `TestTheBudgetSearchReportsEitherBoundAsUNMEASURED/longest_probe_accepted` fails); BR-14's argv[0] guard goes red under deletion, with a positive control that would catch a shim that never forwards; BR-13's `--overhead` now runs a real A/B and re-measures at **2.75 ms/call** here against the 2.9 published in SKILL.md; BR-12's atlas passage now derives from the code; BR-15's lessons.md entry landed. What blocks SHIP is one measured gap: **the O(1) probe count — the issue's headline Done-when, and the load-bearing justification written into `pairRegistrationTimeout`'s comment — holds only when the machine's socket budget is *smaller than the longest candidate*.** Discovery is triggered by a rejection, and on a roomier machine no candidate is ever rejected, so every candidate costs a subprocess again. Measured in this checkout: 26 probes at 25 owned suffixes and 61 at 60 with a roomy budget, and this checkout's *real* zellij budget is 57 bytes against a 34-byte longest candidate. Six Minors from round 1 (BR-5, BR-6, BR-8, BR-10, BR-16, BR-17) were never touched by either fix commit and are re-disposed `not-addressed`.

### 1. Strengths

- `discoverSessionNameBudget` (createflow.go:1000-1032) states the rule rather than patching the end the finding named: "a boundary needs both an acceptance and a refusal" makes both exits fall out of one sentence, and `TestTheBudgetSearchReportsEitherBoundAsUNMEASURED` is a table over both ends so a third exit cannot appear unasserted.
- `reportOverhead` (probes/zellijcalls/main.go:114-186) refuses to be quoted when `sh <= d`. An instrument that fails loudly on a physically impossible result is the right response to having published a negative overhead — I re-ran it and got a plausible +2.75 ms.
- `TestTheShimOnlyActsWhenInvokedAsZellij` (probes/zellijcalls/main_test.go) is end-to-end with a positive control. This is the model the other two guards in this window should copy.
- `TestSessionNameBudgetMatchesRealZellijLive` asserts the *boundary* (budget accepted, budget+1 refused) and separately that `sessionNameRejected` still matches zellij's own raw text — the seam #215 made load-bearing. It correctly parses zellij's output rather than pair's wrapper, and the trap is written down.
- The `pairRegistrationTimeout` comment (couch.go:273-289) records the measurement, the condition that makes it valid, and that raising it again is the wrong fix. That is the ARCH-CONSTRAINTS envelope stated, not assumed.

### 2. Critical findings

None.

### 3. Important findings

**`cmd/internal/launcher/createflow.go:955-985` — the O(1) probe count is conditional on the machine's socket budget, and every artifact states it unconditionally.**

`sessionNameAcceptor` only reaches `discoverSessionNameBudget` after a probe *refuses* a candidate. A refusal requires the budget to be smaller than the longest candidate (34 bytes for `📁pair-couch-<16hex>-NN`). Where the budget is larger, no candidate is ever refused, discovery never runs, and the ladder pays one subprocess per suffix — the original defect. Measured with the production acceptor:

| budget | 25 owned | 60 owned |
|---|---|---|
| 24 (macOS interactive) | 9 probes | 9 probes |
| 80 (roomy) | 26 probes | **61 probes** |

This is not hypothetical: `TestSessionNameBudgetMatchesRealZellijLive` measures **57 bytes** in this checkout, above every candidate. `atlas/session-identity.md:185` names Linux (`~/.cache/zellij`) as the roomy case, README:29 says "probably on Linux", and `couch.go:283-288` cites "#215 made it O(1)" as the reason a fixed 15 s deadline is safe. **Fix sketch:** trigger discovery on the acceptor's *second question*, not on a refusal — `AssignSessionName`'s ledger short-circuit asks exactly once, the ladder asks more than once, so a call counter keeps resume at 1 probe and makes the ladder 1+search+arithmetic on every machine. `ARCH-PURPOSE` (the purpose is delivered on one environment class), `ARCH-CONSTRAINTS` (the envelope's precondition is unstated). The reason two rounds missed it: every probe-count test uses `fakeRuntime{maxSessionNameBytes: 24}`, so the fake cannot express the case — add a budget dimension to `TestAssignSessionNameCostsABoundedNumberOfProbes`.

### 4. Minor findings

- **`cmd/internal/launcher/session_name_scheme_test.go:317-323` — the acceptor half of the BR-11 test never enters the branch it claims to cover.** Verified by panic-mutation: replacing the `if measured` body with `panic()` leaves the test green, because the acceptor's first call is a direct probe that succeeds at `maxSessionNameBytes: 100`. **This is the 3rd finding in family `fix-not-pinned-at-its-call-site`** (BR-6, BR-14). Do not fix this instance alone — the rule is *a test that pins a guard must be shown to go red when the guard is removed; reachable is not asserted.* The enumeration for this window is three sites: this one, BR-14's argv[0] guard (done, with a positive control — the model to copy), and BR-6's `launch_existing.go:123-124`, still open. Run the mutation on each. For this one the fixture needs a rejection first (`accepts(strings.Repeat("z", 101))`) so discovery actually runs.
- **`cmd/internal/launcher/createflow.go:714-719` — `promptForTag` can print a blank refusal.** When zellij refuses a candidate but `discoverSessionNameBudget` falls back to 24 and the candidate is ≤ 24 bytes, `sessionNameFits` returns `ok=true, message=""` and the user sees `pair:` followed by `pick a shorter name.` with no numbers — on precisely the machine class BR-1 exists for. **This is the 3rd finding in family `fallback-guess-used-as-oracle`.** The rule covering both use sites: *an unmeasured number may EXPLAIN a refusal but never contradict one — when the arithmetic disagrees with the observation, report the observation.*
- Both `sessionNameAcceptor` and `promptForTag` still hand-roll `func(n string) bool { return rt.ProbeSessionName(n) == nil }`, and `session_name_budget_live_test.go:41-43` is now a third copy of `pad()`. See BR-5/BR-17 dispositions.

### 5. Test coverage notes

- `diagnoseRegistrationFailure`'s third branch — `Present == false, observeErr == nil`, i.e. the literal "NO Pair session is live. Pair never started" message — has no case. `FakeThreadArtifactCollisionChecker.PairSession` returns an error whenever the name is empty, so the existing two cases exercise the live branch and the unreadable branch only; `SetPairSession(addr, "name", false)` would reach the third.
- `cmd/internal/couchcore` cannot be run to completion in this environment (`ptychild: operation not permitted` on pty spawn, and `runtimebundle` assets are generated), so I verified the #215 tests individually rather than via `make test`. `./cmd/internal/launcher` and `./probes/...` pass clean.

### 6. Architectural notes

- **ARCH-DRY** — flag, at the three probe-closure copies and the duplicated `pad()`; already carried by BR-5/BR-17.
- **ARCH-PURE** — pass. `sessionNameFits` and `discoverSessionNameBudget` are pure over an injected `accepts`; the only IO is `rt.ProbeSessionName` behind the Runtime seam.
- **ARCH-PURPOSE** — flag, the Important above: the purpose ("O(1) probes, not O(N)") is delivered under an environment condition nobody wrote down.
- **ARCH-MOCK** — pass, and this is the round's best structural gain: `sessionNameRejected` now has a live conformance check wired into `make test-couch-zellij-live`, and the shim's stateful surface is exercised end-to-end by `main_test.go` with a fake zellij on PATH.
- **ARCH-CONSTRAINTS** — flag (same Important). The 15 s envelope's stated precondition — bounded assignment — is only true on tight-socket machines.
- **ARCH-SECURE** — pass. `discoverSessionNameBudget` parses only zellij's own refusal substring, the diagnosis degrades visibly on an unreadable index rather than fabricating "never started", and BR-9's fixed `/tmp` trace path is gone.
- **ARCH-ORDER** — note, not a finding. `sessionNameAcceptor` carries `budget`/`measured`/`attempted` across calls: 8 representable states, 3 legal, and `budget` holds an untrustworthy value in one of them. The Important's fix touches exactly this state — collapse it to `unasked | probing | unmeasurable | measured(int)` while you are in there, so a future reader cannot use `budget` without destructuring its provenance. The lessons.md claim that a caller "must destructure" is not enforced by `(int, bool)`: `limit, _ :=` at createflow.go:717 discards it in one keystroke.

### 7. Plan revision recommendations

The issue has no `## Plan` file; its plan is inline and needs an appended `## Revisions` section (BR-10, still open), which should now carry:

- **2026-09-09 — `Done when` restated to what shipped.** "Assigning a name … costs O(1) zellij probes" is true where the socket budget is below the longest candidate; measured 26/61 probes at 25/60 owned suffixes on a roomy budget, and 57 bytes measured live in a short-`TMPDIR` checkout. Either narrow the criterion to the measured environment or land the call-count trigger.
- **2026-09-09 — "Over-long candidates are rejected without a subprocess"** is not literally met (one probe plus the discovery search). The trade-off is deliberate and well-argued in the Log; the criterion should say so.
- **2026-09-09 — the loaded-machine criterion** is asserted by test, not by a live run; the Log already says this, the Done-when does not.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      290c5e8c added a comment explaining why the discard is safe, but the duplication stands; there are now three copies of the probe closure (createflow.go:713, 956, live test:32).
  - id: BR-6
    disposition: not-addressed
    note: |
      Round 2 fixed BR-14, the sibling this finding's own enumeration named, and left this site untouched; launch_existing.go:123-124 still has no test, and the Present==false/err==nil diagnosis branch is uncovered too.
  - id: BR-8
    disposition: not-addressed
    note: |
      couch.go:114 still initialises resumeRegistrationTimeout to pairRegistrationTimeout with no comment; launch_existing.go:111-114 still reads as if the paths differ.
  - id: BR-10
    disposition: not-addressed
    note: |
      The issue file still has no "## Revisions" section, and the two inaccurate Done-when bullets stand; a third is now needed for the conditional O(1) claim.
  - id: BR-11
    disposition: addressed
    note: |
      Mutation-verified: restoring `return hi, true` turns the longest_probe_accepted subtest red. Stated as the rule over both bounds, not patched at one end.
  - id: BR-12
    disposition: addressed
    note: |
      atlas/session-identity.md:182-199 now says the budget IS the acceptance oracle and records the measured-vs-fallback fork.
  - id: BR-13
    disposition: addressed
    note: |
      Real A/B through a binary named zellij; re-ran `trace.sh overhead` here and got +2.753ms/call, matching SKILL.md's 2.9, and it now refuses a non-positive result.
  - id: BR-14
    disposition: addressed
    note: |
      Mutation-verified: deleting the argv[0] guard turns TestTheShimOnlyActsWhenInvokedAsZellij red, and the positive control rules out a shim that forwards nothing.
  - id: BR-15
    disposition: addressed
    note: |
      workshop/lessons.md gained the provenance entry with the four-value table and the bounded-vs-cheap rule.
  - id: BR-16
    disposition: not-addressed
    note: |
      trace.sh:12 still reads `probes/zellijcalls/trace.sh disarm` with no eval, so following the header leaves PATH armed; SKILL.md's block still disagrees.
  - id: BR-17
    disposition: not-addressed
    note: |
      session_name_budget_live_test.go:41-43 still re-derives pad() from createflow.go:1001-1003.
findings:
  - id: new
    severity: Important
    family: optimisation-gated-on-unstated-environment
    title: |
      the O(1) probe count holds only where the socket budget is smaller than the longest candidate, and code, atlas and Done-when all state it unconditionally
    detail: |
      sessionNameAcceptor reaches discoverSessionNameBudget only after a probe REFUSES a
      candidate, and a refusal requires budget < 34 bytes (the longest candidate,
      "pair-couch-<16hex>-NN" with the folder glyph). Where the budget is roomier the
      acceptor never goes arithmetic and the ladder pays one subprocess per suffix again.
      Measured with the production acceptor against fakeRuntime: budget 24 gives 9 probes at
      both 25 and 60 owned suffixes; budget 80 gives 26 and 61. Not hypothetical --
      TestSessionNameBudgetMatchesRealZellijLive measures 57 bytes in this checkout.
      atlas/session-identity.md:185 names Linux as the roomy case, and couch.go:283-288 uses
      "#215 made it O(1)" as the ARCH-CONSTRAINTS justification for a fixed 15s deadline.
      Fix: trigger discovery on the acceptor's SECOND question rather than on a refusal --
      the ledger short-circuit asks exactly once and the ladder asks more than once, so a
      call counter keeps resume at 1 probe and makes the ladder bounded on every machine.
      Both rounds missed it because every probe-count test pins maxSessionNameBytes: 24; the
      fixture needs a budget dimension.
  - id: new
    severity: Minor
    family: fix-not-pinned-at-its-call-site
    title: |
      the acceptor half of TestTheBudgetSearchReportsEitherBoundAsUNMEASURED never enters the branch it claims to cover
    detail: |
      This is the 3rd finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14).
      Do NOT fix this instance alone. The rule: a test that pins a guard must be shown to go
      RED when the guard is removed -- reachable is not asserted. Verified by panic-mutation:
      replacing the `if measured` body in sessionNameAcceptor with panic() leaves
      session_name_scheme_test.go:317-323 green, because the acceptor's first call is a direct
      probe that succeeds at maxSessionNameBytes: 100, so discovery never runs. The
      enumeration for this window is three sites -- this one, BR-14's argv[0] guard (done,
      with a positive control, the model to copy) and BR-6's launch_existing.go:123-124, still
      open. Run the mutation on each. For this site the fixture must take a rejection first,
      e.g. accepts(strings.Repeat("z", 101)), so the arithmetic path is actually entered.
  - id: new
    severity: Minor
    family: fallback-guess-used-as-oracle
    title: |
      promptForTag can print a blank refusal when the fallback budget contradicts zellij's own rejection
    detail: |
      This is the 3rd finding in family `fallback-guess-used-as-oracle` (BR-1, BR-11). Do NOT
      fix this instance alone. The rule covering all three: an unmeasured number may EXPLAIN a
      refusal but must never contradict one -- when the arithmetic disagrees with the
      observation, report the observation. At createflow.go:714-719, if zellij refuses a
      candidate while discoverSessionNameBudget falls back to 24 and the candidate is <= 24
      bytes, sessionNameFits returns ok=true with an empty message and the operator sees
      "pair: " then "pick a shorter name." with no numbers -- on exactly the long-socket-dir
      machine BR-1 exists for. Pre-#215 in origin (#130), but the line was edited in this
      window and the acceptance half of the same rule was already fixed here.
```

---

## Re-review — 2026-09-09T00:26:35-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 215 — couch spends 52 zellij subprocesses assigning a name inside a 5s registration deadline |
| repo | pair |
| issue file | workshop/issues/000215-couch-thread-names-exceed-the-zellij-socket-budget-so-registration-times-out.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9bd5250d9d18588e2e720825ad50fd9f80e8fe1a..869ec844d4d7d8eb473bfbd54794151f9b51173e |
| command | sdlc close --issue 215 |
| reviewer | claude |
| timestamp | 2026-09-09T00:26:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3's fix is real and I verified it rather than reading it: `sessionNameAcceptor` (`createflow.go:1013`) replaces "probe until refused, then measure a budget" with a two-sided monotone bracket, and I measured the production acceptor at **3–4 probes, flat**, across budgets 20/24/34/40/57/80/200 and indexes of 0/25/60/95 owned suffixes — BR-18's roomy-budget regression is genuinely gone, and re-introducing it (gating the acceptance cache on `shortestBad > 0`) turns `TestProbeCountIsInvariantInBothBudgetRegimes/generous` red at 26-vs-61 probes. What blocks a clean SHIP is not the mechanism but its evidence: 869ec844 renamed the live conformance test to `TestTheAcceptorAgreesWithRealZellijLive` and did not update `Makefile.local:84`, which still selects `TestSessionNameBudgetMatchesRealZellijLive` — `go test -run` with no match prints `[no tests to run]` and **exits 0**, so `make test-couch-zellij-live` has been passing vacuously for the one test whose entire job is to check the fake against real zellij. Separately, the branch that delivers this issue's headline Done-when — `diagnoseRegistrationFailure`'s "NO Pair session is live. Pair never started" — has no test at all: replacing it with `return ""` leaves the whole `couchcore` package green. Four round-1 Minors (BR-6, BR-8, BR-10, BR-16) are untouched for the third round running, and BR-19's explicitly-stated three-site enumeration was never executed.

## 1. Strengths

- **`createflow.go:1013` — the bracket is smaller than what it replaces and stronger.** Deriving O(1) from monotonicity ("a session name is a socket filename") rather than from a measured constant removes the whole `defaultSessionNameBudget` / `discoverSessionNameBudget` apparatus that produced BR-1 and BR-11. The invariant `shortestBad > longestOK` holds *structurally* — the early return at `n <= longestOK` makes an inconsistent bracket unrepresentable rather than checked. ARCH-ORDER, done well.
- **`session_index_test.go:364` — invariance, not a bound, in both regimes.** The test asserts `few == many` rather than `few < K`, which is the assertion that would have caught BR-18 the first time. Mutation-verified red.
- **`session_index_test.go:321` — `TestTheAcceptorNeverDisagreesWithTheProbe` is the right replacement.** Swapping a probe-count proxy (">= 20 probes, because nothing can be measured") for soundness against a raw oracle at five budgets subsumes BR-1 *and* pins the monotonicity the bracket rests on. Replacing a test rather than deleting it, and saying so in the commit, is the model.
- **`createflow.go:713-731` — the refusal-message ladder can no longer be empty.** Observation first, `measureAcceptedLimit` second, `refusedAt()` third, all narrowing through the *same* acceptor. BR-20 and BR-5 fall out together, which is what fixing a family looks like.
- **`couch.go:273-289` — a constant with a basis and a precondition.** "15s is ~1.7× the measured 8.85s… if anything unbounded moves back inside this window, raising the number is the wrong fix" is an ARCH-CONSTRAINTS envelope, not a magic number — and as of round 3 the bounded-work precondition it depends on is actually true on every machine.

## 2. Critical findings

None.

## 3. Important findings

**`Makefile.local:84` — the live conformance test is de-registered; the target passes without running it.**
```
-run '^(TestSessionQuiescenceLive|TestSessionDetachLive|TestQuitLifecycleLive|TestSessionNameBudgetMatchesRealZellijLive)$$'
```
`869ec844` renamed that test to `TestTheAcceptorAgreesWithRealZellijLive` (`session_name_budget_live_test.go:25`) and left the regex. Verified: `go test ./cmd/internal/launcher -run '^(TestSessionNameBudgetMatchesRealZellijLive)$'` → `ok … [no tests to run]`, exit 0. So round 3's "`test-couch-zellij-live` exits 0" is true and empty, the close-review's evidence table row for it is stale, and `sessionNameRejected` — the single-substring seam that #215 made load-bearing for *every* candidate — has no live check running anywhere. ARCH-MOCK: the fake is the only oracle again, which is precisely the state BR-2 existed to end. Fix: rename in the regex; the durable form is a floor assertion (`go test -list` must match ≥1 per name) so the next rename fails loudly. I checked the other five names in that target — all still resolve.

**`launch_existing.go:234` — the "Pair never started" verdict is unasserted; deleting it leaves the suite green.**
Mutation: replacing the final `return fmt.Sprintf(" (waited %s; NO Pair session is live…")` with `return ""` and running `./cmd/internal/couchcore` → PASS. The table in `registration_diagnosis_test.go:20-39` uses `present` to decide whether to call `SetPairSession` at all, so `present:false` lands on the *error* path (BR-7's case), never on `observeErr == nil && !binding.Present` — which production reaches whenever the name is in the index but no live session matches (`artifactcollision.go:165-174`). That branch is the discriminator this issue's Done-when names first ("distinguishes 'pair never started' from 'pair started and did not register'"). **This is the 4th finding in family `fix-not-pinned-at-its-call-site`** (BR-6, BR-14, BR-19). Do not fix this instance alone. The rule: *a branch or guard whose value is what it says must be shown to go RED when it is removed; "reachable" is not "asserted."* The enumeration BR-19 wrote down and this round found un-executed is now four sites — BR-14's argv[0] guard (done, with a positive control), BR-19's own site (removed with its test rather than swept), BR-6's `launch_existing.go:123-124` call-site wiring (still open), and this branch. Run the mutation on each; the fixture change here is one line (`fake.SetPairSession(address, "📁pair-couch-26", false)`).

## 4. Minor findings

- `createflow.go:310-311` — comment says "one budget measurement shared by every tag… so N tags cost one discovery, not N." There is no discovery any more; the shared thing is a length bracket. **2nd in family `vestigial-branch-reads-as-policy`** — rule: *when you delete a mechanism, grep for the prose that describes it in the same commit*, since `git log -S discoverSessionNameBudget` finds the code but not the sentence.
- `session_name_scheme_test.go` — deleting `TestDiscoverSessionNameBudget` took with it the only assertion that calibration probes are synthetic pads ("`list-clients` SUCCEEDS against a foreign live session, which would read as fits for the wrong reason"). `measureAcceptedLimit` still relies on it; nothing in `cmd/` now references `sessionNameProbeMarker` outside `createflow.go`. Family `property-lost-when-its-test-was-replaced`.
- `createflow.go:379` — `runCreate` still calls `rt.ProbeSessionName(session)` raw, a third encoding of "will zellij take this" outside the acceptor, costing an unconditional subprocess on every create. **3rd in family `duplicated-probe-then-discover`** — the rule (now stated once and worth writing into the acceptor's godoc): *the acceptor is the only encoding of "does this fit"; any site asking that question takes one rather than calling the probe.* ARCH-DRY.
- `osruntime.go:108` — `out, _ := exec.CommandContext(...).CombinedOutput()` discards the exec error, so a zellij that is missing, killed, or past `zjTimeout` (5s — reachable under exactly the load this issue is about) yields empty output → `sessionNameRejected("")` false → **accepted**. The bracket now generalizes that one failed observation to every name of that length or shorter. **2nd in family `inconclusive-observation-stated-as-conclusion`** — same rule BR-7 was fixed under: *a failed observation is not a positive one; degrade visibly.* Pre-#215 in origin, amplified here.
- `workshop/lessons.md` — round 3's lesson is in the commit message and the gate ledger only. The transferable one ("every probe-count fixture pinned `maxSessionNameBytes: 24`, so the fake could not express the environment where the optimisation fails") is the strongest of the issue and isn't in the file. **2nd in family `lesson-recorded-only-in-the-issue`.**

## 5. Test coverage notes

- The two probe-count tests plus `TestResumingAKnownThreadCostsOneProbe` are a good pair of pins: bounded and cheap are separately asserted, and I confirmed the regime mutation reds only the generous subtest, so they are not redundant.
- `TestARefusalMessageIsNeverEmptyEvenWhenNothingCanBeMeasured` re-implements `promptForTag`'s branch ladder rather than calling it, so it pins the *shape* of the decision, not the site. Its `else if` arm is also unreachable-by-construction in the negative direction (a refused candidate always leaves `refusedAt()` true), so the first fallback message is never exercised. Same family as the Important above.
- Coverage gap worth one test while you are in there: `assignLaunchSessionNames` shares one acceptor across tags (`createflow.go:312`). Correct — the bracket keys on length only — but nothing exercises tag B being answered arithmetically from tag A's probes.
- All `cmd/...` failures in this checkout are `ptychild: operation not permitted` (environment, not code); `go build ./...`, `./cmd/internal/launcher`, `tests/paint-gate-consumers-test.sh` and `tests/plan-superseded-facts-test.sh` are clean.

## 6. Architectural notes

- **ARCH-DRY** — flag (`createflow.go:379`, and `session_name_budget_live_test.go:31-32` re-spelling the marker/pad). BR-5 is genuinely closed; the class has one production site left.
- **ARCH-PURE** — pass. `sessionNameFits` / `AssignSessionName` are pure with the limit and the predicate as parameters; the acceptor closes over an injected `Runtime`; `diagnoseRegistrationFailure` takes the observer through the `PairSessionIO` seam. No mocks needed to run any of it.
- **ARCH-PURPOSE** — flag. The mechanism fulfils the purpose (measured, both regimes). The *finding-answering* half does not: BR-19 named the class and wrote the enumeration, round 3 removed the one site the finding pointed at and left the other two, and a fourth surfaced under mutation. That is the instance, not the class.
- **ARCH-MOCK** — flag (the Important above). The stateful fake and the seam are right; the live conformance check exists and no longer runs.
- **ARCH-CONSTRAINTS** — pass, and improved. The envelope at `couch.go:273-289` now has a basis, a bound, and a stated precondition that is actually satisfied. Residual for #218: 4 probes × `zjTimeout` 5s = 20s worst case still exceeds the 15s registration budget if zellij hangs — far better than 52 × 5s, but the per-probe timeout is not part of the written envelope.
- **ARCH-SECURE** — mostly pass. BR-7's absent-vs-unreadable split and BR-9's per-invocation `$$` trace path are both real fixes; the shim's `argv[0]` guard makes the smoke-suite leak impossible rather than remembered. One residual, the discarded exec error above.
- **ARCH-ORDER** — pass, notably. Two ints with a structurally-enforced invariant is the right shape for state carried across probes, and the transition set reads straight off the code. The acceptor is not goroutine-safe and every current caller is sequential; a one-line ownership note would keep that true.

## 7. Plan revision recommendations

The issue still has no `## Revisions` section (BR-10, third round open). It needs one appended entry covering: the root-cause correction; the operator's 15s decision; the 2026-09-08 narrowing to "unblock", with latency split to #218; and the three close-review rounds with their verdicts. Two `## Done when` bullets should be restated to what shipped rather than left standing:

- *"Over-long candidates are rejected without a subprocess"* — not literally met and deliberately so. The first over-long candidate costs one probe (measured: 2 probes at budget 24, owned=0), and that probe is what makes every later one free. Say that.
- *"couch starts a thread on `../pair/` on a loaded machine, not only a quiet one"* — asserted by test (invariance across index sizes in both budget regimes), not by a live run under sustained load. The Log already says this honestly; the Done-when should match it.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      measureAcceptedLimit now narrows through the caller's own acceptor; one encoding of "does this fit".
  - id: BR-6
    disposition: not-addressed
    note: |
      Untouched by rounds 2 and 3; still no assertion that the diagnosis reaches launch_existing.go:123-124.
  - id: BR-8
    disposition: not-addressed
    note: |
      couch.go:114 still assigns pairRegistrationTimeout with no note that the resume branch is a test seam.
  - id: BR-10
    disposition: not-addressed
    note: |
      Still no "## Revisions" section, and both inaccurate Done-when bullets stand unchanged.
  - id: BR-16
    disposition: not-addressed
    note: |
      trace.sh:12 still prints `trace.sh disarm` without eval; SKILL.md:15 still has it. Two runbooks, still disagreeing.
  - id: BR-17
    disposition: addressed
    note: |
      The live test now carries its own marker literal and drives the production acceptor, so it no longer follows production silently.
  - id: BR-18
    disposition: addressed
    note: |
      Mutation-verified: gating the acceptance cache on a prior refusal reds the generous-budget subtest at 26 vs 61 probes. Measured 3-4 probes flat across budgets 20-200.
  - id: BR-19
    disposition: not-addressed
    note: |
      The named test was removed with its subject, but the three-site enumeration the finding demanded was never run; BR-6 is still open and a fourth site surfaced under mutation.
  - id: BR-20
    disposition: addressed
    note: |
      promptForTag now defaults to the observation and only upgrades to a measured or refused number.
findings:
  - id: new
    severity: Important
    family: fake-is-the-only-oracle
    title: |
      the live conformance test was renamed and Makefile.local's -run regex was not, so test-couch-zellij-live passes without running it
    detail: |
      This is the 2nd finding in family `fake-is-the-only-oracle`. 869ec844 renamed
      TestSessionNameBudgetMatchesRealZellijLive to TestTheAcceptorAgreesWithRealZellijLive
      (session_name_budget_live_test.go:25) and left Makefile.local:84 selecting the old name.
      Verified: `go test ./cmd/internal/launcher -run '^(TestSessionNameBudgetMatchesRealZellijLive)$'`
      prints "ok ... [no tests to run]" and exits 0. So round 3's "test-couch-zellij-live exits 0"
      claim is vacuous for that test, and sessionNameRejected -- the single-substring seam #215 made
      load-bearing for every candidate rather than one -- has no live check running anywhere.
      ARCH-MOCK. The rule: a suite selected by a name regex silently passes when the name stops
      matching, so the selection needs a floor that fails when it matches nothing. The other five
      names in that target still resolve; I checked.
  - id: new
    severity: Important
    family: fix-not-pinned-at-its-call-site
    title: |
      diagnoseRegistrationFailure's "NO Pair session is live / Pair never started" branch has no test; deleting it leaves the package green
    detail: |
      This is the 4th finding in family `fix-not-pinned-at-its-call-site` (BR-6, BR-14, BR-19). Do
      NOT fix this instance alone. The rule: a branch or guard whose value is what it says must be
      shown to go RED when it is removed -- reachable is not asserted. Verified by mutation:
      replacing launch_existing.go:234 with `return ""` leaves ./cmd/internal/couchcore passing.
      registration_diagnosis_test.go uses one `present` flag to decide whether SetPairSession is
      called at all, so present:false lands on the error path (BR-7's case) and never on
      observeErr == nil with Present false -- which production reaches whenever the name is in the
      index but no live session matches (artifactcollision.go:165-174). That branch is the first
      discriminator the issue's Done-when names. The enumeration for this window is now four sites:
      BR-14's argv[0] guard (done, with a positive control), BR-19's site (removed with its test
      rather than swept), BR-6's launch_existing.go:123-124, and this branch. Run the mutation on
      each. The fixture change here is one line: SetPairSession(address, "\U0001F4C1pair-couch-26", false).
  - id: new
    severity: Minor
    family: vestigial-branch-reads-as-policy
    title: |
      createflow.go:310 still describes a budget measurement and a per-tag discovery that round 3 deleted
    detail: |
      This is the 2nd finding in family `vestigial-branch-reads-as-policy`. The rule: when you delete
      a mechanism, grep for the prose that describes it in the same commit -- `git log -S` finds the
      code but not the sentence. The shared thing is now a length bracket, not a discovery, and "N
      tags cost one discovery, not N" describes an implementation that no longer exists.
  - id: new
    severity: Minor
    family: property-lost-when-its-test-was-replaced
    title: |
      deleting TestDiscoverSessionNameBudget took with it the only assertion that calibration probes are synthetic pads
    detail: |
      The deleted test asserted every probe carried sessionNameProbeMarker, because
      `list-clients` SUCCEEDS against a foreign live session and would read as "fits" for the wrong
      reason -- making the measured limit depend on whatever else is running. measureAcceptedLimit
      still depends on that property and nothing in cmd/ now references sessionNameProbeMarker
      outside createflow.go. The rule: when a replacement test supersedes an old one, diff their
      assertions, not just their names.
  - id: new
    severity: Minor
    family: duplicated-probe-then-discover
    title: |
      runCreate still calls rt.ProbeSessionName raw, a third encoding of "will zellij take this name" outside the acceptor
    detail: |
      This is the 3rd finding in family `duplicated-probe-then-discover`. Do NOT fix this instance
      alone. The rule, now that the acceptor exists: the acceptor is the ONLY encoding of "does this
      fit", and any site asking that question takes one rather than calling the probe directly.
      createflow.go:379 costs an unconditional subprocess on every create, after assignment has
      already established acceptability. ARCH-DRY. Worth writing the rule into sessionNameAcceptor's
      godoc so the next site inherits it.
  - id: new
    severity: Minor
    family: inconclusive-observation-stated-as-conclusion
    title: |
      ProbeSessionName discards the exec error, so a missing or timed-out zellij reads as ACCEPT -- and the bracket now generalizes that
    detail: |
      This is the 2nd finding in family `inconclusive-observation-stated-as-conclusion`. The rule is
      the one BR-7 was fixed under: a failed observation is not a positive one; degrade visibly.
      osruntime.go:108 does `out, _ := exec.CommandContext(...).CombinedOutput()`, so an absent
      zellij, a killed process, or a call past zjTimeout (5s, reachable under exactly the load this
      issue is about) yields empty output, sessionNameRejected returns false, and the name is
      accepted. Pre-#215 that mis-accepted one candidate; the bracket now writes it to longestOK and
      answers every shorter name from it. ARCH-SECURE.
  - id: new
    severity: Minor
    family: lesson-recorded-only-in-the-issue
    title: |
      round 3's lesson -- every probe-count fixture pinned one budget, so the fake could not express the failing environment -- is not in workshop/lessons.md
    detail: |
      This is the 2nd finding in family `lesson-recorded-only-in-the-issue`. The rule: a review
      round's transferable lesson belongs in lessons.md, not only in the commit message and the gate
      ledger, because that is the file AGENTS.md asks you to read at session start. The existing
      #215 entry covers rounds 1-2 (provenance) and stops there; BR-18's lesson -- an optimisation
      whose precondition is unstated, hidden because every fixture pinned maxSessionNameBytes: 24 --
      is the strongest of the issue and the one most likely to recur in a different file.
```
