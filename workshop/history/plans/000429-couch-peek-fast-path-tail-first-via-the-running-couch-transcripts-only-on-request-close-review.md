# Boundary Review — pair#429 (whole-issue close)

| field | value |
|-------|-------|
| issue | 429 — couch --peek fast path: tail first via the running Couch, transcripts only on request |
| repo | pair |
| issue file | workshop/issues/000429-couch-peek-fast-path-tail-first-via-the-running-couch-transcripts-only-on-request.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb4e8bf1a3f5ea9c590fddcf6941047ec9c000ca..822dec82c6b9fd4fb16898e1b33cce9be5422a31 |
| command | sdlc close --issue 429 |
| reviewer | claude |
| timestamp | 2026-10-10T14:17:17-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Let me read the files directly.

Still reviewing — checking peek rendering and waiting on the test run.

Tests are still compiling; running the targeted packages directly.

Socket bind was blocked by the sandbox; re-running that one test outside it.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change does what the Spec asks. `PeekThread` no longer resolves transcripts unless `--transcripts` is passed. The recording fallback now takes the agent from the thread record (`RecordAgent`), which gives the same result as the resolver's old precedence. The broker's `tail` op can now name its thread by slot, resolved through the existing `canonicalTarget`. The CLI fast path answers only when every slot's live tail answers, and falls back to the typed peek for the whole request otherwise. It renders the full answer into a buffer before writing, so a fallback never prints half an answer.

The targeted tests pass: `couchcore`, `couchmessage`, and `couchcmd`'s `Peek|Tail|Arity` set. `TestPeekAnswersFromTheRunningCouch` failed inside the sandbox because it couldn't bind its socket (`bind: operation not permitted`), and passed when run outside it.

One drift needs fixing before ship: the `--json` output of the fast path is not the same shape as the typed path's. Done-when 1, the live timing, is deferred to the TL by a documented Revision. That is accepted ops practice, but the issue is not proven done until that check runs.

1. **Strengths**
   - `cmd/internal/couchcmd/peek_fast.go:76-90`: the answer is rendered before anything is written, so a `false` return leaves stdout empty. The test checks this with `out != ""`.
   - `couchmessage.ResolveTailSlot` reuses `parseSlot` and `canonicalTarget` instead of a new resolver (ARCH-DRY). It is a pure function with its own unit tests.
   - `RecordAgent` is pulled out of `OSSwitchContextResolver.Resolve`, and both the resolver and `PeekThread` use it. There is one source for "which agent the record names".
   - `TestDefaultPeekResolvesNoTranscripts` goes through the real operation dispatch. It counts `Resolve` calls on both the live path and the recording fallback, which pins Done-when 3.
   - `TestPeekAnswersFromTheRunningCouch` runs from argv over a real socket, with a runtime whose `NewCouchWith` fails. That proves the fast path builds no Couch.

2. **Critical:** none.

3. **Important**
   - `peek_fast.go:60` leaves `WorkingPath` unset, but the typed `PeekResult` fills `working_path` (`peek.go:150`). So `couch --peek … --json` drops a field depending on whether a Couch is running and every slot is live. That runtime-dependent shape isn't documented. The text output is unaffected, because `renderPeek` doesn't print the field.
     - Fix options: carry the working path in `TailThread`, where the broker can get it without a store read; or drop the field from the peek contract on both paths; or document that `working_path` appears only on the typed path.
     - Add a test asserting both paths give the same JSON keys (ARCH-PURPOSE: Spec 3 says single- and multi-slot peeks take the same path, and the output should be the same too).

4. **Minor**
   - Plan design item 4 says the recording fallback resolves "only if the record has none". The code never resolves. That's correct, since `Resolve` would only return `RecordAgent` again, but the plan's prose is stale.
   - `peek_fast.go:25-38` parses `--lines` and `--json` from `inv.args` again, a second copy of the operation's arg parsing (ARCH-DRY, small).
   - The fast path accepts aliases and unique prefixes (`pa:1`) through `canonicalTarget`. The typed fallback resolves through `ResolveThreadReference`, which may not accept them. The same ref could then succeed with a Couch running and fail without one.
   - The fan-out is one goroutine per expanded ref, with no limit (ARCH-CONSTRAINTS). The ref list is operator-typed and each request is bounded by `AdmissionTimeout`, so this is acceptable. It's worth one line in the atlas.
   - When one slot fails, the typed fallback asks every slot's broker for its live tail a second time. That's harmless, but it's extra latency on the degraded path.

5. **Test coverage**
   - Covered: the fast path's fallbacks (a failed slot, a non-slot ref, `--transcripts`, over-200 `--lines`, bad `--lines`); validation of the by-slot `tail` request; slot resolution by alias, prefix, ambiguity and missing slot; the broker's error when it can't read the repository names.
   - Not covered:
     - JSON parity between the fast and typed paths (see Important).
     - The fallback when an older Couch refuses the request with `invalid-request` "tail takes only a thread…". It's handled by the generic `result.Code != "ok"` branch, but no test enters it.

6. **Architecture**
   - **ARCH-DRY:** pass. `ResolveTailSlot` reuses `canonicalTarget`, and `RecordAgent` is shared. The only nit is the duplicated arg parsing.
   - **ARCH-PURE:** pass. The resolver is pure, and `fastPeek` takes `call` as an injected seam.
   - **ARCH-PURPOSE:** flagged, because the fast path's output differs from the typed path's (Important).
   - **ARCH-MOCK:** pass. The tests use a real Unix socket with a stateful fake handler on the production `messageCall` seam.
   - **ARCH-CONSTRAINTS:** pass with one gap. The latency claim is unproven live; Done-when 1 is deferred to the TL, and the measured parts are logged.
   - **ARCH-SECURE:** pass. The by-slot form is identity-free like the existing one. The response carries only slot, tag and agent, with no session ID, nonce or PID, and `ValidateRequest` still rejects identity fields.
   - **ARCH-ORDER:** pass. The fast path keeps no state between events. Goroutines are joined with a `WaitGroup` before return, and each has a timeout.
   - **ARCH-FUNERAL:** pass. Nothing durable is created; the lease is closed straight away.
   - **Follow-up worth filing:** resolving a `repo:N` in the typed path costs about 1.5s (from the Log). The fallback still pays it.

7. **Plan revisions**
   - Add a `## Revisions` entry: the recording fallback never calls `Resolve`; when the record names no agent, peek reports "agent is unknown" in `unavailable`.
   - Add a `## Revisions` entry stating whether `working_path` belongs in the peek contract on the fast path.

```findings
findings:
  - id: new
    severity: Important
    family: fast-path-output-parity
    title: |
      Fast-path peek JSON omits working_path that the typed peek returns
    detail: |
      peek_fast.go builds PeekResult without WorkingPath, but PeekThread fills it, so the couch --peek --json shape depends on whether a Couch is running. Carry it in TailThread, or drop or document it, and add a test that both paths give the same JSON keys.
  - id: new
    severity: Minor
    family: plan-prose-matches-code
    title: |
      Plan item 4 says the recording fallback resolves when the record has no agent; the code never resolves
    detail: |
      The behavior is correct, since Resolve would only return RecordAgent again. Add a Revisions entry.
  - id: new
    severity: Minor
    family: single-arg-parser
    title: |
      fastPeek re-parses --lines and --json from inv.args instead of reusing the operation's arg parsing
  - id: new
    severity: Minor
    family: fast-path-output-parity
    title: |
      Fast path accepts alias or prefix slot refs that the typed fallback may resolve differently
    detail: |
      canonicalTarget accepts pa:1, while ResolveThreadReference may not, so the same ref can succeed only when a Couch is running.
  - id: new
    severity: Minor
    family: test-covers-fallback-branch
    title: |
      No test for the fallback when an older Couch refuses a by-slot tail request
```

The full background test run has now finished, and it doesn't change the verdict (FIX-THEN-SHIP).

- **`couchmessage`:** all tests pass.
- **`couchcore`:** three tests fail: `TestPtyRunnerBlockedStart…`, `TestBlockedRunnerCancellationConformance` and `TestFailedUnacceptedPTYCleanup…`. Each fails with `ptychild: start …: operation not permitted`. That is the sandbox refusing to start pty child processes, not a problem with this change, and these tests aren't touched by this diff.
- **`couchcmd`:** no summary line in the run output. Its peek tests passed when run directly; `TestPeekAnswersFromTheRunningCouch` needed to run outside the sandbox to bind its socket.

The findings from my review above still stand, including the one Important item: the fast path's `--json` output leaves out `working_path`, which the regular peek includes.

---

## Re-review — 2026-10-10T14:21:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 429 — couch --peek fast path: tail first via the running Couch, transcripts only on request |
| repo | pair |
| issue file | workshop/issues/000429-couch-peek-fast-path-tail-first-via-the-running-couch-transcripts-only-on-request.md |
| boundary | whole-issue close |
| milestone | — |
| window | bb4e8bf1a3f5ea9c590fddcf6941047ec9c000ca..4cc7d076da3196e3757340c1753029687abc6650 |
| command | sdlc close --issue 429 |
| reviewer | claude |
| timestamp | 2026-10-10T14:21:22-07:00 |
| verdict | SHIP |

## Review

Inspecting the rest of `PeekThread` and running the package tests to verify the BR-1 fix.

```verdict
verdict: SHIP
confidence: high
```

The issue's purpose is met. A slot-only `couch --peek` now goes to the running Couch's broker, one `tail` request per slot, all sent at once, and never builds a Couch. Any other case falls back to the typed peek, which reads transcripts only when `--transcripts` is passed. All five round-1 findings are settled:

- **BR-1 (fixed):** `TestFastPeekMatchesTheTypedPeek` now requires the fast and typed paths to produce byte-identical JSON. I checked that it actually catches the bug: in a scratch copy I removed the line `thread.WorkingPath = record.WorkingPath` (`message_service.go:130`), and the test failed with `working_path` missing from the fast output.
- **BR-5 (fixed):** an "older Couch" test case now exists.
- **BR-2, BR-3, BR-4:** answered with sound reasoning in the issue's Revisions.

What remains is minor: the plan and atlas descriptions of `TailThread` and the agent's source were not updated when BR-1 changed the code. The only live check left is the TL's rollout timing of the "under 200ms / under 100ms" Done-when, which Revisions already defers.

Tests run:
- `couchmessage` and `couchcore`, `-run 'Peek|Tail'`: pass.
- `couchcmd`, `-run 'Peek|Tail'`: passes when run outside the sandbox. Inside it, `TestPeekAnswersFromTheRunningCouch` fails only because the sandbox blocks binding the unix socket.

1. **Strengths**
   - `couchcore.RecordAgent` (`switchcontext.go:883`) is now the one place that decides a thread's agent. The resolver, the typed peek and the broker's `tailThread` all call it (ARCH-DRY).
   - `fastPeek` (`peek_fast.go:255-269`) builds the whole answer before writing anything. When it gives up and returns false, stdout is untouched, and the table test checks `out == ""` in every fallback case.
   - `couchmessage.ResolveTailSlot` is a pure function with its own unit tests. It reuses `canonicalTarget`, the same resolution `--send-to` uses, instead of writing a second slot parser.
   - `ValidateRequest` accepts exactly one way of naming the thread (scope plus tag, or slot) and rejects requests that give both. Both cases are tested.
   - `TestPeekAnswersFromTheRunningCouch` runs from `argv` through a real broker socket with a runtime whose Couch construction always fails. That makes "nothing builds a Couch" a property a test enforces.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The plan's Design items 1 and 4 and `atlas/couch.md` still describe the code as it was before BR-1 (details in the findings block). This is the 2nd finding in family `plan-prose-matches-code`; the rule to fix is in the findings block, not just this instance.
   - The broker now reads the store twice for each by-slot tail request (`families` and `record`). That is fine at the current per-slot cost; if fan-out grows, measure it.

5. **Test coverage notes**
   - The parity test pins agent, working path, cursor and the truncation count. I mutation-checked it.
   - The fallback cases are covered as a table: a failing slot, an older Couch, a reference that isn't a slot, `--transcripts`, `--lines` over 200, and a non-numeric `--lines`.
   - `TestDefaultPeekResolvesNoTranscripts` covers Done-when 3: the default peek and the recording fallback make zero resolver calls, and `--transcripts` makes exactly one.

6. **Architecture**
   - **ARCH-DRY: pass.** `RecordAgent` and `askTail` are shared by both paths, and `canonicalTarget` is reused.
   - **ARCH-PURE: pass.** `ResolveTailSlot` and `ValidateRequest` are pure; `askTail` is the thin IO shell.
   - **ARCH-PURPOSE: pass.** Every Spec item is delivered. The live timing is a declared, operator-approved deferral, not part of the purpose quietly dropped.
   - **ARCH-MOCK: pass.** A real unix socket is tested against a stateful fake; the older-Couch case is modelled.
   - **ARCH-CONSTRAINTS: pass.** Each request is bounded by `AdmissionTimeout` and `--lines` by `ValidTailLines`; fan-out is one request per slot the user names.
   - **ARCH-SECURE: pass.** The by-slot request carries no identity, the store directory remains the access control, and `TailThread` exposes no session, nonce or PID.
   - **ARCH-ORDER: pass.** It holds no state between events: each peek is a stateless request/response, and the one parallel step is a lexically scoped `WaitGroup`.
   - **ARCH-FUNERAL: pass.** It creates nothing durable: no files, caches or processes, only an in-memory result per CLI call.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry to the durable plan file itself, not only the issue, recording two changes:
     - `TailThread` gained `WorkingPath`, and its agent comes from `RecordAgent`, falling back to the binding's agent.
     - Item 4: the order is "sole incarnation, else the latest launch", and the recording fallback never calls the resolver.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TailThread carries WorkingPath; TestFastPeekMatchesTheTypedPeek fails (missing working_path) when message_service.go:130 is removed — mutation-verified.
  - id: BR-2
    disposition: addressed
    note: |
      The issue's Revisions "close round 1 findings" records that the fallback never calls the resolver; the code (peek.go:185-197) matches.
  - id: BR-3
    disposition: withdrawn
    note: |
      The flags arrive normalized from ParseCLI, and any flag fastPeek does not recognize falls back to the typed path, so a mismatch can only cost speed, never give a wrong answer.
  - id: BR-4
    disposition: withdrawn
    note: |
      Accepted on purpose and recorded in Revisions: the fast path resolves slots the way --send-to does, and the fallback names any reference it cannot resolve in unavailable.
  - id: BR-5
    disposition: addressed
    note: |
      The "an older couch" case in TestFastPeekAnswersOnlyWhenEverySlotIsLive answers invalid-request and checks the fallback runs with nothing written.
findings:
  - id: new
    severity: Minor
    family: plan-prose-matches-code
    title: |
      The plan and atlas still describe TailThread and the agent's source as they were before BR-1
    detail: |
      2nd finding in this family. The plan's Design item 1 and atlas/couch.md say TailThread is {slot, tag, agent} (no working_path, and the agent is not said to come from the record). Plan item 4 says "LatestLaunchProfile, else the first incarnation", but RecordAgent prefers the sole incarnation, then the latest launch. Rule: when a fix changes a surface that prose describes, grep every restatement (plan Design, atlas, SKILL) in the same commit, and put a Revisions entry in the durable plan file, not only in the issue.
```
