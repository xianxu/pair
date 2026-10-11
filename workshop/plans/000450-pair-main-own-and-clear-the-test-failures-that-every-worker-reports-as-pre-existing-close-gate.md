---
gate: boundary-review
issue: 450
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T14:14:33-07:00"
      dispatched: "2026-10-11T14:08:46-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: host-preflight probes only socket+PTY; a sandbox that refuses exec of /bin/ps passes it and still fails tests that look like code bugs
          detail: 'Observed in the review sandbox: host-preflight.sh reports ok, but go test ./cmd/pair-go fails TestPublicPairCommandFamiliesIgnoreCouchStore with "fork/exec /bin/ps: operation not permitted", with or without COUCH_PAIR_DATA_DIR set. Three claims overstate the probe: Plan item 3 ("cannot be mistaken"), atlas/index.md, and the host-preflight.sh header. Fix: add an exec probe (e.g. ps -p $$), or narrow all three to the probed capabilities. ARCH-PURPOSE.'
          family: preflight-claims-exceed-probe
          round: 1
        - id: BR-2
          severity: Minor
          title: host-preflight.sh blames the sandbox for any python3 failure, including python3 missing
          detail: A non-zero exit from the python heredoc, including "command not found", prints "An agent sandbox does this". Check command -v python3 first and report that separately.
          family: preflight-misattributes-cause
          round: 1
        - id: BR-3
          severity: Minor
          title: Listing the preflight first does not stop make -k test from running every other suite
          detail: Makefile.local:213. The Log's baseline uses make -k; under -k the other prerequisites still run and fail after the preflight fails. Document this or make the preflight abort the run.
          family: preflight-ordering-under-keep-going
          round: 1
        - id: BR-4
          severity: Minor
          title: lessons.md says to clear ambient env once per package in TestMain, but couchcmd pins it in a single child env
          detail: continuation_acceptance_test.go:254 pins COUCH_PAIR_DATA_DIR= on one command; launcher uses TestMain. It works (couchcmd passes with the variable set), but it does not follow the lesson written in the same window.
          family: lesson-practice-mismatch
          round: 1
        - id: BR-5
          severity: Minor
          title: A pane with no endpoint on a socket-refusing host shows the "finish and close it before reopening" message, which cannot help
          detail: restore_client.lua:113. On a host that refuses the socket, every reopened pane is also legacy. Consider a host-specific message.
          family: legacy-pane-message-misleads
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-11T14:29:51-07:00"
      dispatched: "2026-10-11T14:28:54-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: host-preflight.sh now probes running ps -p; it fails in this sandbox naming "exec ps"; atlas, header and Makefile comment name the three probed capabilities and say an unprobed refusal still fails as an ordinary test.
          round: 2
        - id: BR-2
          disposition: addressed
          note: command -v python3 is checked first with its own message (host-preflight.sh:13-16).
          round: 2
        - id: BR-3
          disposition: addressed
          note: Makefile.local:216-219 documents that make -k runs past the preflight; plain make (no -j in the makefiles) stops at the first prerequisite.
          round: 2
        - id: BR-4
          disposition: addressed
          note: lessons.md now covers the child-env pin as well as TestMain; the launcher TestMain reset is shown to be needed (5 tests fail without it).
          round: 2
        - id: BR-5
          disposition: addressed
          note: restore_client.lua:105,114 adds the unserved state with a host-specific message; the client test fails at line 123 when it is reverted.
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: b049190c7215d3d51fe7e62826ccadaecf922244
      blocked: false
---

# Gate ledger — pair#450 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T14:14:33-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `preflight-claims-exceed-probe` host-preflight probes only socket+PTY; a sandbox that refuses exec of /bin/ps passes it and still fails tests that look like code bugs
  Observed in the review sandbox: host-preflight.sh reports ok, but go test ./cmd/pair-go fails TestPublicPairCommandFamiliesIgnoreCouchStore with "fork/exec /bin/ps: operation not permitted", with or without COUCH_PAIR_DATA_DIR set. Three claims overstate the probe: Plan item 3 ("cannot be mistaken"), atlas/index.md, and the host-preflight.sh header. Fix: add an exec probe (e.g. ps -p $$), or narrow all three to the probed capabilities. ARCH-PURPOSE.
- **BR-2** [Minor] `preflight-misattributes-cause` host-preflight.sh blames the sandbox for any python3 failure, including python3 missing
  A non-zero exit from the python heredoc, including "command not found", prints "An agent sandbox does this". Check command -v python3 first and report that separately.
- **BR-3** [Minor] `preflight-ordering-under-keep-going` Listing the preflight first does not stop make -k test from running every other suite
  Makefile.local:213. The Log's baseline uses make -k; under -k the other prerequisites still run and fail after the preflight fails. Document this or make the preflight abort the run.
- **BR-4** [Minor] `lesson-practice-mismatch` lessons.md says to clear ambient env once per package in TestMain, but couchcmd pins it in a single child env
  continuation_acceptance_test.go:254 pins COUCH_PAIR_DATA_DIR= on one command; launcher uses TestMain. It works (couchcmd passes with the variable set), but it does not follow the lesson written in the same window.
- **BR-5** [Minor] `legacy-pane-message-misleads` A pane with no endpoint on a socket-refusing host shows the "finish and close it before reopening" message, which cannot help
  restore_client.lua:113. On a host that refuses the socket, every reopened pane is also legacy. Consider a host-specific message.

## Round 2 — 2026-10-11T14:29:51-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — host-preflight.sh now probes running ps -p; it fails in this sandbox naming "exec ps"; atlas, header and Makefile comment name the three probed capabilities and say an unprobed refusal still fails as an ordinary test.
- BR-2 — addressed — command -v python3 is checked first with its own message (host-preflight.sh:13-16).
- BR-3 — addressed — Makefile.local:216-219 documents that make -k runs past the preflight; plain make (no -j in the makefiles) stops at the first prerequisite.
- BR-4 — addressed — lessons.md now covers the child-env pin as well as TestMain; the launcher TestMain reset is shown to be needed (5 tests fail without it).
- BR-5 — addressed — restore_client.lua:105,114 adds the unserved state with a host-specific message; the client test fails at line 123 when it is reverted.

## Open findings

(none — every finding has been disposed)
