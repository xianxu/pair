# Boundary Review — pair#425 (whole-issue close)

| field | value |
|-------|-------|
| issue | 425 — couch: in-memory slot tails with style and cursor marks, plus a multi-slot peek |
| repo | pair |
| issue file | workshop/issues/000425-couch-in-memory-slot-tails-with-style-and-cursor-marks-plus-a-multi-slot-peek.md |
| boundary | whole-issue close |
| milestone | — |
| window | d7ade9b1e57fadb44e50c92224cb975a9190fd67..99abfba0dfb1abff3e5f4e5abdef4ea2a324f223 |
| command | sdlc close --issue 425 |
| reviewer | claude |
| timestamp | 2026-10-10T12:24:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: checking slot resolution, concurrency of the wrapper probe, and scrollback cost.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary:** The design holds up. The tail is a read of the vt emulator the wrapper already keeps, through an identity-free, read-only broker op modeled on `broadcast-status`. When the live read fails, peek falls back to the recording and names the reason. The renderer, reference expansion, protocol validation, broker handler and CLI router all have direct tests, and the targeted tests pass (`couchcore`, `couchmessage`, `wrapcmd` and `couchcmd` with `-race`). Two tests that need `/tmp` sockets fail in this sandbox with "operation not permitted", which is a sandbox limit and not a defect in the code. One thing stands between this and SHIP: the first Done-when item ("returns the tail from the wrapper's memory … in well under a second") was never shown end to end. The Log says the live path needs a Couch restart and relaunched slots, and the only live run fell back to the recording. Everything else is Minor.

### 1. Strengths
- `wrapcmd/terminal_tail.go:204-300`: `renderTail` and `renderTailRow` are pure functions over `[][]uv.Cell`. They are tested on captured bytes (`composer.raw` for the ghost text, `paste-short.raw` for the draft) and on synthetic paints. Escaping `‹` as `‹‹` stops screen text from posing as markup, and a test pins it.
- `couchmessage/protocol.go:76-90`: tail validation is strict and identity-free. It refuses a binding, caller identity or message fields, refuses tail fields on any other op, and caps the length of `TailScope`/`TailTag`. A table test covers it.
- The bounds are explicit and enforced at both ends: `ValidTailLines` (1..200), `BoundTail` at the wrapper and again at `RemoteEndpoint.Tail`, and `MaxPeekSlots` (32) limiting fan-out.
- The router lesson from #424 was applied. `TestMultiSlotPeekRunsThroughTheRouter` drives `RunWithRuntime(argv)` end to end, and the gap it catches is recorded in `lessons.md`.
- Mixed versions are handled. An older Couch's refusal maps to "restart Couch", and an over-long tail is refused before the call so the recording answers with the full count.

### 2. Critical findings
None.

### 3. Important findings
- **Done-when #1 is not evidenced (issue `## Done when`, Log 2026-10-10).** No run has gone CLI → broker `tail` → wrapper endpoint → `terminalModel.Tail` against a live slot, and none has been timed. The tests cover each link separately: a fake `messageCall`, a fake endpoint in `handleTail`, and a real socket only from `RemoteEndpoint` to the wrapper. **Fix:** restart Couch on this build, relaunch three or more slots, and run `time couch --peek pair:A:B:C --lines 10`. Confirm `source live`, the `‹dim›`/`‹cursor›` marks and the timing, then record the result in `--verified` and the Log, or ask the operator to run this smoke test.

### 4. Minor findings
- `couchcore/operationdispatch.go:170`: `a["tag"] == ""` can never be false, because peek declares no `tag` argument. Drop the condition or declare the argument.
- `couchcore/peek.go:591`: `ExpandPeekReferences` splits every ref on `,` and expands `x:1:2`. Peek used to accept a thread path containing a comma, or ending in `:N:M`; such a path is now split or expanded. The atlas or help text should say so.
- `couchcmd/run.go:364`: when `ExpandPeekReferences` fails (for example more than 32 slots), `singleReference` hands the raw ref to `ParseWorkspaceReference`. The user then sees the router's parse error instead of the clearer cap message.
- `couchcmd/peek_tail.go:178`: a wrapper from before this change, connected to a new Couch, refuses the endpoint op `tail` and its `Lines` field. That error reaches the caller raw instead of being mapped to "relaunch the slot", the way the old-Couch case is mapped to "restart Couch". Matching on error text is also fragile in itself.
- `couchmessage/tail.go:990`: when a single line is longer than the byte cap, the byte truncation can cut a markup token (`‹di`). Rare, but readers would misparse it.
- `couchmessage/tail.go:932`: the comment says the 128 KiB cap is "well under MaxFrameBytes", but that holds only before JSON escaping. Go escapes `<`, `>` and `&` as 6-byte sequences, so the claim depends on how the screen is made up. If the frame overflows, the peek falls back visibly, so this is not a hazard.
- `wrapcmd/terminal_tail.go:191`: `agentTail` reads `p.terminal` from the endpoint goroutine without a lock. `settledNow` already does the same, so this is not new, but `tailProbe` adds a second cross-goroutine reader.
- ARCH-DRY: `Tail`'s closed-model branch and `TestTailGhostSuggestionAndTypedDraftDiffer` each slice `snapshot.Cells` into rows by hand. A `snapshotRows(s)` helper would serve both.

### 5. Test coverage notes
- Covered: the renderer (ghost vs. typed draft on captured bytes, reverse video, hidden cursor, cursor shape, scrollback then screen, alt screen, wide glyphs, escaping), `BoundTail`, the endpoint and protocol validation tables, `handleTail` (ok, error, absent wrapper, invalid request), `readSlotTail`'s answer mapping, the live-first preference with fallback, the multi-slot snapshot in request order, and the router.
- Not covered:
  - No test runs `handleTail` → a real `RemoteEndpoint` over a socket. The broker-to-wrapper link is only covered with a fake (ARCH-MOCK, minor).
  - Two wrappers bound to the same thread (`ambiguous`) has no test.
  - An endpoint without a `TailReader` (`unsupported`) has no test.
  - A peek that fails mid-way (the `PeekThread` error branch inside `PeekSlots`) has no test.

### 6. Architectural notes
- ARCH-DRY: pass, apart from the minor row-slicing duplicate.
- ARCH-PURE: pass. The pure core is `renderTail`, `ExpandPeekReferences` and `BoundTail`; the I/O glue is thin and injected (`SlotTailReader`, `messageCall`).
- ARCH-PURPOSE: pass in code; all four Done-when items are delivered in code and docs. Only the live-timing evidence is open, as noted above.
- ARCH-MOCK: minor gap. The fakes are function fakes, and there is no broker-plus-endpoint socket test.
- ARCH-CONSTRAINTS: pass on the declared bounds (200 lines, 128 KiB, 32 concurrent slots, `AdmissionTimeout`). `terminalModel.Tail` renders under `m.mu`, which blocks `Feed` for at most 200 rows of rendering, which is acceptable. The latency claim is unmeasured.
- ARCH-SECURE: pass. No new trust widening, since peek could already read recordings and the store directory is operator-only. Markup spoofing is blocked by the escaping. Inputs from older versions fail visibly.
- ARCH-ORDER: pass. Nothing is held between events; each request renders under the model's lock and returns.
- ARCH-FUNERAL: pass. Nothing durable is created; the tail is rendered per request and dies with the response.

### 7. Plan revision recommendations
- The plan's `## Steps` checkboxes are all still unticked, while the issue's `## Plan` is ticked. Tick the plan's steps, or add a `## Revisions` note that the issue's Plan is the record.
- `cli.go` is listed under Files but was not changed; usage lives in `run.go`. Note this in `## Revisions`.

```findings
findings:
  - id: new
    severity: Important
    family: done-when-unverified-live
    title: |
      Done-when 1 (live tail from wrapper memory, well under a second) never exercised end to end or timed
    detail: |
      The Log says the live path needs a Couch restart; the only live run fell back to the recording. Restart Couch on this build, relaunch 3+ slots, time couch --peek over them, and record source live plus the dim/cursor marks in --verified.
  - id: new
    severity: Minor
    family: dead-condition
    title: |
      operationdispatch.go peek checks a["tag"] == "" but peek declares no tag argument
  - id: new
    severity: Minor
    family: reference-grammar-overload
    title: |
      ExpandPeekReferences splits any comma and expands x:1:2, changing how thread paths with those characters resolve
  - id: new
    severity: Minor
    family: version-skew-error-mapping
    title: |
      A wrapper from before this change refusing endpoint op tail is not mapped to a relaunch hint; old-Couch detection matches error text
  - id: new
    severity: Minor
    family: byte-truncation-splits-token
    title: |
      BoundTail single-line byte truncation can cut a markup token mid-way
  - id: new
    severity: Minor
    family: stateful-fake-coverage
    title: |
      No test runs handleTail against a real RemoteEndpoint socket; ambiguous and unsupported branches untested
```

---

## Re-review — 2026-10-10T12:26:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 425 — couch: in-memory slot tails with style and cursor marks, plus a multi-slot peek |
| repo | pair |
| issue file | workshop/issues/000425-couch-in-memory-slot-tails-with-style-and-cursor-marks-plus-a-multi-slot-peek.md |
| boundary | whole-issue close |
| milestone | — |
| window | d7ade9b1e57fadb44e50c92224cb975a9190fd67..b754b12eab635306f23915fc492ee48ee50b0cef |
| command | sdlc close --issue 425 |
| reviewer | claude |
| timestamp | 2026-10-10T12:26:42-07:00 |
| verdict | SHIP |

## Review

Prior round presumably was at 99abfba0 or so. Let me see the issue and the latest commits.

Disposing the six prior findings now; running the touched packages' tests to confirm they pass.

```verdict
verdict: SHIP
confidence: medium
```

Two things changed since round 1. The live check for Done-when 1 was formally deferred to the TL, and a timed test was added that runs a tail over the real endpoint socket. Both commits were inspected (`99abfba0`, `b754b12e`), and the touched tests pass: `couchmessage`, `couchcore`, `couchcmd` and the `wrapcmd` Tail tests. The `wrapcmd` socket tests needed the sandbox off because of their `/tmp` socket root. The 200-line tail over 12k lines of scrollback ran in about 0.7 ms. Each hop is now proven separately: the wrapper render, the real socket, the broker handler, the CLI broker call and the router. The production `authority.endpoint` returns a `RemoteEndpoint`, which satisfies `TailReader` (`message_service.go:301`), so the type assertion in `handleTail` holds in production. The five Minor findings from round 1 are still open and don't block. Nothing new reaches Important.

1. **Strengths**
   - `wrapcmd/terminal_tail.go`: the tail is rendered on request from the emulator the wrapper already keeps, so no new store or durable artifact appears (ARCH-FUNERAL pass).
   - Doubling a literal `‹` (`terminal_tail.go:159`) stops screen text from faking the markup, and a test pins it.
   - The ghost-suggestion vs typed-draft test uses captured bytes, plus the same cells with faint cleared. It pins exactly the 10-10 misread.
   - When the live tail fails, peek falls back to the recording and names the reason inline (`peek.go` PeekThread). The over-200-line case now refuses instead of silently clamping.
   - The multi-slot peek is capped at `MaxPeekSlots=32`. Fan-out is bounded and lexically scoped by `wg.Wait`.

2. **Critical:** none.

3. **Important:** none new.

4. **Minor:** the remaining round-1 findings (BR-2 to BR-6) are still open. The closed-model path in `terminalModel.Tail` passes a `vt.Cursor` with no style, so a finished slot reports its shape as "default". Not raised as a finding.

5. **Test coverage notes:** the wrapper hop is now exercised over a real Unix socket, with a timing check. The broker handler is still tested only through a fake endpoint. Its `ambiguous` and `unsupported` branches have no fixture (BR-6).

6. **Architectural notes**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass. `renderTail` and `ExpandPeekReferences` are pure; IO stays in `readSlotTail`, `handleTail` and the endpoint.
   - ARCH-PURPOSE: pass, given the documented revision.
   - ARCH-MOCK: pass. The broker call is injected as `messageCall`, and the endpoint has a real-socket test.
   - ARCH-CONSTRAINTS: pass. Lines, bytes and slot count are all bounded, and timing is measured.
   - ARCH-SECURE: pass. The tail request is validated, carries no identity and is read-only. The socket is owner-only.
   - ARCH-ORDER: pass. Nothing is stateful; `agentTail` follows the `settledNow` precedent for reading `p.terminal`.
   - ARCH-FUNERAL: pass. Nothing durable is created.
   - Detecting an old Couch by error text remains fragile. A protocol version or capability field would fix that class (BR-4).

7. **Plan revision recommendations:** the issue's Revisions entry says the issue "isn't counted done until that passes", yet the issue is being closed now. Either soften that sentence to "the TL confirms at rollout" or keep a tracked follow-up, so the close doesn't contradict the issue's own text.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Done-when 1 revised with a documented ops decision deferring the live run to the TL; every hop proven, including the timed real-socket test TestTailOverEndpointSocketIsFast (~0.7 ms). The Revision text still says the issue is not done until the TL check passes, which contradicts the close.
  - id: BR-2
    disposition: not-addressed
    note: |
      operationdispatch.go:171 still checks a["tag"] == "", but peek declares no tag arg; harmless dead condition.
  - id: BR-3
    disposition: not-addressed
    note: |
      ExpandPeekReferences is unchanged; a thread path containing a comma or matching x:1:2 is still split.
  - id: BR-4
    disposition: not-addressed
    note: |
      A pre-425 wrapper refusing op tail still surfaces as "unavailable: <raw error>" from handleTail; old-Couch detection in readSlotTail is still substring matching.
  - id: BR-5
    disposition: not-addressed
    note: |
      BoundTail single-line cut at MaxTailBytes is unchanged; practically unreachable at real pane widths.
  - id: BR-6
    disposition: not-addressed
    note: |
      The wrapper hop now has a real-socket test, but handleTail itself only runs against a fake endpoint, and its ambiguous/unsupported branches have no test.
```
