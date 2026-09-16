# Boundary Review — pair#255 (whole-issue close)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | pair |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | whole-issue close |
| milestone | — |
| window | b11ab67ff1d290386cf12177f2dacdccbc8551c0..78922625e5db4ecf21974ccbe7b0501caa51da0e |
| command | sdlc close --issue 255 |
| reviewer | codex |
| timestamp | 2026-09-15T18:34:01-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-22 is addressed, and the inspected terminal ownership and consumer integration are sound. Focused tests and race tests pass. One Important delivery gap blocks closure: the plan claims native terminal conformance is wired into CI, but the committed workflow never runs those tests.

1. **Strengths**

   - Both consumers use the shared Presenter to order presentation and input admission.
   - Endpoint publications preserve coherent frame/history state; child teardown joins publication and input workers.
   - Notification output preserves framing and accounts for accepted bytes.
   - BR-22’s corrected documentation matches the implemented broker route.

2. **Critical findings**

   None.

3. **Important findings**

   **Native terminal conformance is absent from CI.** [Plan:107](/Users/xianxu/workspace/pair/workshop/plans/000255-terminal-abstraction-plan.md:107) marks this delivered. However, [workflow:108](/Users/xianxu/workspace/pair/.github/workflows/couch-zellij-conformance.yml:108) invokes [Makefile.local:83](/Users/xianxu/workspace/pair/Makefile.local:83), which runs only launcher, couchcore and couchcmd lifecycle suites. The new [native test:30](/Users/xianxu/workspace/pair/cmd/internal/couchtty/terminal_native_test.go:30) requires `PAIR_LIVE_COUCH_NATIVE=1` and a candidate binary; no CI entrypoint supplies them.

   Wire the native terminal tests into CI with dependencies, a freshly built candidate, explicit enablement and relevant source-path triggers. Verify the tests execute rather than skip. **ARCH-PURPOSE, ARCH-MOCK.**

4. **Minor findings**

   None remaining.

5. **Test coverage notes**

   Passed focused terminal, qualification, notification, wrapper and fork tests. Race tests passed for terminal, ttyio, notifytransport, ptychild, termcmd and couchtty. Native conformance and extended soaks were not rerun during this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared terminal semantics across consumers.
   - **ARCH-PURE — pass:** inspected pure concepts match their classifications and have direct tests.
   - **ARCH-PURPOSE — flag:** promised CI delivery remains incomplete.
   - **ARCH-MOCK — flag:** independent/native fixtures exist, but native terminal checks lack automated execution.
   - **ARCH-CONSTRAINTS — pass:** bounded queues/resources; measured performance exceptions remain explicit.
   - **ARCH-SECURE — pass:** inspected notification boundaries validate ownership and input.
   - **ARCH-ORDER — pass:** serialized admission, partial-write handling and joined teardown.
   - **ARCH-FUNERAL — pass:** inspected workers and temporary artifacts have disposal paths.

7. **Plan revision recommendations**

   Append a `## Revisions` entry recording the missing CI wiring and its correction, including executed tests, dependencies and trigger coverage.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      The pinned correction updates atlas/architecture.md:743-748, wrapcmd/wrap.go:10-12 and bin/pair-notify:13 together. Current descriptions name the wrapper broker and serialized pane output; outer-TTY references describe compatibility metadata. notifycmd/run.go:34-40 and wrapcmd/wrap.go:730-736 support those descriptions. This correction changes prose/comments only.
findings:
  - id: new
    severity: Important
    family: conformance-ci-enforcement
    title: |
      Claimed native terminal CI coverage is not wired into any workflow
    detail: |
      workshop/plans/000255-terminal-abstraction-plan.md:107 marks native terminal CI complete, but .github/workflows/couch-zellij-conformance.yml:108 invokes Makefile.local:83-86, which runs only lifecycle suites. cmd/internal/couchtty/terminal_native_test.go:30 skips without PAIR_LIVE_COUCH_NATIVE=1, and no CI entrypoint supplies that flag or PAIR_NATIVE_BINARY. Wire the native terminal suites into CI with dependencies, a freshly built candidate and relevant source triggers; verify actual execution rather than skips. ARCH-PURPOSE and ARCH-MOCK.
```

---

## Re-review — 2026-09-15T18:42:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | pair |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | whole-issue close |
| milestone | — |
| window | c01ec7c67637040feb371c3aeb32d24b6c9ce002..85fded9312e4e5f04a466fe411449ec4221ddf64 |
| command | sdlc close --issue 255 |
| reviewer | codex |
| timestamp | 2026-09-15T18:42:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

BR-23 is addressed: the workflow invokes native conformance with dependencies, a fresh candidate, explicit opt-ins, and enforcement against skipped or missing tests. No new blocking defect was found. Independent verification was partially limited by the sandbox denying `/dev/tty` access; the retained execution log records the complete native pass.

1. **Strengths**
   - CI invokes the new target while retaining lifecycle coverage (`.github/workflows/couch-zellij-conformance.yml:165`).
   - Execution validation requires six named pass events and rejects skips/failures (`tests/native-terminal-ci.py:24`).
   - Mutation verification confirms the validator matters: disabling it produced four failed assertions.
   - README and atlas document the actual entrypoint and dependencies.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None raised.

5. **Test coverage**
   - Five Python gate/cleanup tests passed.
   - Eleven relevant Go packages and the vendored VT suite passed.
   - Terminal, PTY-child, and transport race suites passed.
   - Native nvim, scrolling, and notification PTY tests passed independently.
   - Direct/wrapped Zellij fixtures executed but failed because this sandbox denied opening `/dev/tty`. The inspected `/tmp/pair255-br23-native-ci.log` records all six required pass events and the execution validator’s success. This is retained evidence, not an independently reproduced complete pass.
   - The full pinned-range whitespace check reports Markdown hard-break trailing spaces in an earlier review artifact.

6. **Architecture**
   - **ARCH-DRY — pass:** Couch and Pair term use shared endpoint/presenter ownership.
   - **ARCH-PURE — pass:** view transitions remain pure; backend and transport are explicitly integration components.
   - **ARCH-PURPOSE — pass:** native CI coverage now reaches the promised suites and rejects absent execution.
   - **ARCH-MOCK — pass:** stateful transport tests complement native conformance; the scheduled workflow supplies the live check.
   - **ARCH-CONSTRAINTS — pass:** bounded queues, geometry validation, write deadlines, and CI timeouts are present.
   - **ARCH-SECURE — pass:** native fixtures isolate session state; candidate paths are explicit.
   - **ARCH-ORDER — pass:** presenter state mutation passes through its transition function; forced-order and partial-write coverage exercises ownership.
   - **ARCH-FUNERAL — pass:** endpoint/publication teardown and invocation-owned candidate/evidence cleanup have explicit owners.

7. **Plan revisions:** None required. The appended BR-23 revision matches the delivered CI change.

Earlier disposed findings retain their supplied dispositions; none is reopened.

```findings
dispose:
  - id: BR-1
    disposition: addressed
  - id: BR-2
    disposition: addressed
  - id: BR-3
    disposition: addressed
  - id: BR-4
    disposition: addressed
  - id: BR-5
    disposition: addressed
  - id: BR-6
    disposition: addressed
  - id: BR-7
    disposition: addressed
  - id: BR-8
    disposition: addressed
  - id: BR-9
    disposition: addressed
  - id: BR-10
    disposition: addressed
  - id: BR-11
    disposition: addressed
  - id: BR-12
    disposition: addressed
  - id: BR-13
    disposition: addressed
  - id: BR-14
    disposition: addressed
  - id: BR-15
    disposition: addressed
  - id: BR-16
    disposition: addressed
  - id: BR-17
    disposition: addressed
  - id: BR-18
    disposition: addressed
  - id: BR-19
    disposition: addressed
  - id: BR-20
    disposition: addressed
  - id: BR-21
    disposition: addressed
  - id: BR-22
    disposition: addressed
  - id: BR-23
    disposition: addressed
    note: |
      Workflow lines 140–166 install dependencies and invoke the native target; tests/native-terminal-ci.py:63–83 builds a fresh candidate, enables both native flags, and requires six pass events. Five regression tests pass; disabling validation produces four failures. Retained execution evidence records the complete native pass. Independent rerun passed nvim, scrolling, and notification PTY checks but Zellij fixtures encountered sandbox-denied /dev/tty access.
```
