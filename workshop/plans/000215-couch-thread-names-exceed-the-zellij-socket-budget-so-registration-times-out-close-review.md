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
