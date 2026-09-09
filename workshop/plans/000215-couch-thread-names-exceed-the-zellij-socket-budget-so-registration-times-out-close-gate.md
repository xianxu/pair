---
gate: boundary-review
issue: 215
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-08T21:57:43-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
          detail: |-
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
          family: fallback-guess-used-as-oracle
          round: 1
        - id: BR-2
          severity: Important
          title: the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
          detail: |-
            ARCH-MOCK. The fix rests on "a candidate's length is a local fact", but nothing runs it
            against real zellij. sessionNameRejected (zellijparse.go:48) matches one substring of
            zellij's stderr, and #215 makes that seam far more load-bearing: a missed rejection used to
            mis-accept one candidate, now one wrong budget decides every remaining candidate without
            asking. make test-live covers quiescence, detach and quit -- nothing covers the name budget.
            Add a PAIR_LIVE_COUCH-gated test asserting the boundary against OSRuntime: budget bytes
            accepted, budget+1 rejected.
          family: fake-is-the-only-oracle
          round: 1
        - id: BR-3
          severity: Important
          title: the zellij tracer injects ~50ms per call into the startup it exists to measure
          detail: |-
            probes/zellijcalls/shim.sh:41-46 spawns python3 twice per traced call for a timestamp
            (~25ms each, measured here) plus a bash fork, and drops the exec when armed. SKILL.md tells
            the operator to compare in-zellij time against wall span -- but the python cost lands
            outside each measured duration and inside the span, inflating exactly the denominator the
            conclusion turns on. At the call volume #215 already measured this is seconds injected into
            an 8.85s budget, in the instrument #218 will use to find where those seconds go. Not yet
            run, so cheap to fix: use bash EPOCHREALTIME with a fallback, and state the residual
            per-call overhead in SKILL.md.
          family: probe-overhead-distorts-measurement
          round: 1
        - id: BR-4
          severity: Important
          title: atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
          detail: |-
            atlas/index.md:17-49 states the probe-home rule and closes with "anything else goes under
            cmd/probes/ with its own target in the same commit, since nothing will run it otherwise".
            probes/zellijcalls/ fits neither branch: it takes arm|report|disarm, needs an interactive
            shell, has no Go code and no target. test-smoke does not break on it (confirmed under
            /bin/sh, exit 0) but prints "shared harness, not a probe" -- a category it does not belong
            to. That atlas passage records itself as written because #199's first probe landed in the
            wrong home; this is the next instance.
          family: new-surface-without-atlas
          round: 1
        - id: BR-5
          severity: Minor
          title: promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
          detail: |-
            createflow.go:713-716. ARCH-DRY. Cost is fine (refusal path only), but there are now two
            encodings of one idea; an acceptor that can also surrender its measured budget collapses
            them, and would also carry the fallback-vs-measured distinction I1 needs.
          family: duplicated-probe-then-discover
          round: 1
        - id: BR-6
          severity: Minor
          title: nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
          detail: |-
            The function is unit-tested in isolation; launch_existing.go:123-124, the %s that wires it
            into the user-visible error, has no test. Reachable, not dead -- but this issue's own Log
            records a path regressing precisely because nothing asserted it. The PairSessionIO
            assertion's false branch is also uncovered.
          family: fix-not-pinned-at-its-call-site
          round: 1
        - id: BR-7
          severity: Minor
          title: an unreadable session index is reported as "NO Pair session -- Pair never started"
          detail: |-
            launch_existing.go:226-229. When observeErr is a read or corruption failure rather than an
            absent binding, the message still asserts the conclusion before quoting the error.
            ARCH-SECURE: degrade visibly rather than substitute a fabricated answer downstream reads as
            evidence.
          family: inconclusive-observation-stated-as-conclusion
          round: 1
        - id: BR-8
          severity: Minor
          title: resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
          detail: |-
            couch.go:114 initialises it to pairRegistrationTimeout, so launch_existing.go:111-114 gives
            both paths 15s in production while reading as if they differ. Worth a comment. The cold path
            has no equivalent seam, so no test can exercise a cold registration timeout without waiting
            15s.
          family: vestigial-branch-reads-as-policy
          round: 1
        - id: BR-9
          severity: Minor
          title: trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
          detail: |-
            trace.sh:15 truncates a predictable name with ": >"; on a Linux box with no TMPDIR that is a
            shared directory. arm prepends a repo directory to the interactive shell's PATH with only a
            manual disarm. Both opt-in developer surface, neither stated. Also trace.sh:32 interpolates
            $DIR into a grep pattern, and shim.sh:45 logs "$*", losing argv quoting that kind() then
            re-splits on whitespace.
          family: predictable-path-in-shared-directory
          round: 1
        - id: BR-10
          severity: Minor
          title: the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
          detail: |-
            AGENTS.md asks for an appended "## Revisions" section (timestamp + reason + delta) rather
            than overwriting; 4663fb70 rewrote the body in place and clobbered the frontmatter. Two
            "Done when" bullets are now inaccurate: the loaded-machine criterion is asserted by test
            rather than a live run, and "over-long candidates are rejected without a subprocess" is not
            literally met -- the first one costs 1 probe plus ~7 for discovery. That trade-off is
            deliberate and well-argued; the criterion should say what shipped.
          family: issue-overwritten-not-revised
          round: 1
      blocked: true
---

# Gate ledger — pair#215 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-08T21:57:43-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fallback-guess-used-as-oracle` defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
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
- **BR-2** [Important] `fake-is-the-only-oracle` the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
  ARCH-MOCK. The fix rests on "a candidate's length is a local fact", but nothing runs it
  against real zellij. sessionNameRejected (zellijparse.go:48) matches one substring of
  zellij's stderr, and #215 makes that seam far more load-bearing: a missed rejection used to
  mis-accept one candidate, now one wrong budget decides every remaining candidate without
  asking. make test-live covers quiescence, detach and quit -- nothing covers the name budget.
  Add a PAIR_LIVE_COUCH-gated test asserting the boundary against OSRuntime: budget bytes
  accepted, budget+1 rejected.
- **BR-3** [Important] `probe-overhead-distorts-measurement` the zellij tracer injects ~50ms per call into the startup it exists to measure
  probes/zellijcalls/shim.sh:41-46 spawns python3 twice per traced call for a timestamp
  (~25ms each, measured here) plus a bash fork, and drops the exec when armed. SKILL.md tells
  the operator to compare in-zellij time against wall span -- but the python cost lands
  outside each measured duration and inside the span, inflating exactly the denominator the
  conclusion turns on. At the call volume #215 already measured this is seconds injected into
  an 8.85s budget, in the instrument #218 will use to find where those seconds go. Not yet
  run, so cheap to fix: use bash EPOCHREALTIME with a fallback, and state the residual
  per-call overhead in SKILL.md.
- **BR-4** [Important] `new-surface-without-atlas` atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
  atlas/index.md:17-49 states the probe-home rule and closes with "anything else goes under
  cmd/probes/ with its own target in the same commit, since nothing will run it otherwise".
  probes/zellijcalls/ fits neither branch: it takes arm|report|disarm, needs an interactive
  shell, has no Go code and no target. test-smoke does not break on it (confirmed under
  /bin/sh, exit 0) but prints "shared harness, not a probe" -- a category it does not belong
  to. That atlas passage records itself as written because #199's first probe landed in the
  wrong home; this is the next instance.
- **BR-5** [Minor] `duplicated-probe-then-discover` promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
  createflow.go:713-716. ARCH-DRY. Cost is fine (refusal path only), but there are now two
  encodings of one idea; an acceptor that can also surrender its measured budget collapses
  them, and would also carry the fallback-vs-measured distinction I1 needs.
- **BR-6** [Minor] `fix-not-pinned-at-its-call-site` nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
  The function is unit-tested in isolation; launch_existing.go:123-124, the %s that wires it
  into the user-visible error, has no test. Reachable, not dead -- but this issue's own Log
  records a path regressing precisely because nothing asserted it. The PairSessionIO
  assertion's false branch is also uncovered.
- **BR-7** [Minor] `inconclusive-observation-stated-as-conclusion` an unreadable session index is reported as "NO Pair session -- Pair never started"
  launch_existing.go:226-229. When observeErr is a read or corruption failure rather than an
  absent binding, the message still asserts the conclusion before quoting the error.
  ARCH-SECURE: degrade visibly rather than substitute a fabricated answer downstream reads as
  evidence.
- **BR-8** [Minor] `vestigial-branch-reads-as-policy` resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
  couch.go:114 initialises it to pairRegistrationTimeout, so launch_existing.go:111-114 gives
  both paths 15s in production while reading as if they differ. Worth a comment. The cold path
  has no equivalent seam, so no test can exercise a cold registration timeout without waiting
  15s.
- **BR-9** [Minor] `predictable-path-in-shared-directory` trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
  trace.sh:15 truncates a predictable name with ": >"; on a Linux box with no TMPDIR that is a
  shared directory. arm prepends a repo directory to the interactive shell's PATH with only a
  manual disarm. Both opt-in developer surface, neither stated. Also trace.sh:32 interpolates
  $DIR into a grep pattern, and shim.sh:45 logs "$*", losing argv quoting that kind() then
  re-splits on whitespace.
- **BR-10** [Minor] `issue-overwritten-not-revised` the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
  AGENTS.md asks for an appended "## Revisions" section (timestamp + reason + delta) rather
  than overwriting; 4663fb70 rewrote the body in place and clobbered the frontmatter. Two
  "Done when" bullets are now inaccurate: the loaded-machine criterion is asserted by test
  rather than a live run, and "over-long candidates are rejected without a subprocess" is not
  literally met -- the first one costs 1 probe plus ~7 for discovery. That trade-off is
  deliberate and well-argued; the criterion should say what shipped.

## Open findings

- **BR-1** [Important] `fallback-guess-used-as-oracle` defaultSessionNameBudget becomes an acceptance test in the fallback path, against its own stated contract
- **BR-2** [Important] `fake-is-the-only-oracle` the O(1) probe claim and the arithmetic-vs-zellij agreement are pinned only against fakeRuntime
- **BR-3** [Important] `probe-overhead-distorts-measurement` the zellij tracer injects ~50ms per call into the startup it exists to measure
- **BR-4** [Important] `new-surface-without-atlas` atlas update appears missing for the shell/PATH-shim probe class introduced by probes/zellijcalls
- **BR-5** [Minor] `duplicated-probe-then-discover` promptForTag hand-rolls the probe-then-discover pair that sessionNameAcceptor now encapsulates
- **BR-6** [Minor] `fix-not-pinned-at-its-call-site` nothing asserts diagnoseRegistrationFailure's output actually reaches the operator
- **BR-7** [Minor] `inconclusive-observation-stated-as-conclusion` an unreadable session index is reported as "NO Pair session -- Pair never started"
- **BR-8** [Minor] `vestigial-branch-reads-as-policy` resumeRegistrationTimeout now equals the cold constant, so the resume/cold branch is a test seam only
- **BR-9** [Minor] `predictable-path-in-shared-directory` trace.sh writes a fixed ${TMPDIR:-/tmp}/pair-zellij-trace.tsv and arms PATH with manual disarm
- **BR-10** [Minor] `issue-overwritten-not-revised` the issue records three mid-stream scope changes in Log subsections instead of a Revisions section, and Done-when now overstates
