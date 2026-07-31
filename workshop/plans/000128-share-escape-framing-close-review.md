# Boundary Review — pair#128 (whole-issue close)

| field | value |
|-------|-------|
| issue | 128 — share escape-sequence framing between termcmd and wrapcmd |
| repo | pair |
| issue file | workshop/issues/000128-share-escape-framing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6ef2c1fc234a4d932ccde58a45554002b7d8c6d6..HEAD |
| command | sdlc close --issue 128 |
| reviewer | claude |
| timestamp | 2026-07-30T17:23:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The extraction itself is well-executed and the differential-oracle strategy is the right call — `Frame`'s alternative ordering reproduces the retired regex faithfully (including the non-obvious `]`-is-0x5D fallthrough), the `termcmd` adapters are byte-for-byte identical to the scanners they replace, and the `-1`/`(0,false)` sentinels are preserved exactly where `rename_input.go` depends on them. What blocks SHIP is a correctness regression the oracle cannot see: `ansi.Strip`'s new no-ESC fast path returns the caller's slice **aliased**, and both `wrapcmd` call sites immediately hand that slice to `bytesReplaceAll`, which compacts **in place**. The retired `otherEscRe.ReplaceAll` always allocated a fresh copy, so this is behaviour drift on the one Done-when that promised none — and at `wrap.go:1152` the aliased buffer is `p.captureBuffer`, mutex-protected state owned by another goroutine. Everything else is documentation drift: the plan still describes a two-arg `Frame(buf, mode)` API that the Log says was deliberately collapsed but never revised.

I could not run `make test` — Bash is unavailable in this review environment (`EPERM` on `~/.claude/session-env`). All findings below are from reading the code; the Critical is specifically a class the existing suite cannot catch.

## 1. Strengths

- **The oracle is the right proof.** Keeping `otherEscRe` alive in `ansi/oracle_test.go:17` and fuzzing `SequenceLen`/`Strip` against it converts "behaves identically" from an argument into a measurement, and re-running the seed corpus as a plain table (`oracle_test.go:64`) means the check doesn't depend on anyone starting a fuzz session. This is the pattern to repeat.
- **`Frame`'s ordering discipline.** `ansi.go:115-119` — the OSC arm falling through to the two-byte class is exactly the regex's semantics and exactly what PQ-9 asked for, and `ansi_test.go:33` pins it with the reason inline. Likewise `ansi.go:108-114`: refusing a two-byte fallback for `'['` so `frameCSI`'s `None`-vs-`Incomplete` verdict survives is the subtle call that prevents a malformed CSI being pinned in a pending buffer forever.
- **PQ-5 resolved structurally, not by patching.** Exporting `TerminatorScan` as introducer-independent (`ansi.go:63`) rather than dispatching on `buf[1]` is the correct read of what `csiEnd` always was, and `queries_test.go`'s SS3 pins were written to pass against the *old* code first — so "this changes nothing" is measured rather than asserted.
- **Site 3 got faster, not just shorter.** `wrap.go:1019` replaced `FindIndex(data[i:])` — an O(remaining-buffer) scan discarded unless `loc[0]==0` — with an anchored O(sequence-length) call. Same answer, strictly less work per ESC byte on the agent-output hot path.
- **Policy left alone.** `codexKKPMarkers` / `terminalQueryLiterals` stayed put, and both the package doc (`ansi.go:5-9`) and `atlas/architecture.md:435` say why merging them would be the bug. Correct restraint (ARCH-DRY applied to framing, not policy).

## 2. Critical findings

**C1 — `ansi.Strip`'s fast path aliases its input; both callers then mutate it in place (`ansi.go:174-177`, consumed at `wrap.go:813` and `wrap.go:1152`). ARCH-PURE.**

```go
// ansi.go:174
if !hasESC(buf) {
    return buf        // <- aliases the caller's slice
}
```

`bytesReplaceAll` (`wrap.go:1162-1170`) is an **in-place** compactor:

```go
out := b[:0:len(b)]
for _, x := range b { if x != c { out = append(out, x) } }
```

The retired `otherEscRe.ReplaceAll(raw, nil)` always returned a freshly allocated slice (Go's `replaceAll` appends into a nil `buf` even on the no-match path), so `bytesReplaceAll` only ever wrote into a private copy. That invariant is now gone for any buffer containing no `0x1b`.

Failure scenario at `wrap.go:1152` (`maybeFinalizeEarly`):
1. `handleChunk` appends a chunk into `p.captureBuffer` under `captureMu` (`wrap.go:2478`).
2. `maybeFinalizeEarly` takes `buf := p.captureBuffer` (`wrap.go:1147`) and releases the mutex.
3. `ansi.Strip(buf)` returns `buf` itself — the accumulated capture text has no ESC.
4. `bytesReplaceAll(stripped, '\r')` compacts **`p.captureBuffer`'s backing array in place**. `p.captureBuffer`'s *length* is unchanged, so the tail keeps stale duplicated bytes: `"hello\r\nworld"` (12B) becomes `"hello\nworldd"`.
5. Capture stays active; the next chunk appends *after* the stale tail; `finalizeCapture` writes the corrupted buffer to `p.captureOutPath` (`wrap.go:1127`) — the file nvim reads for Alt+i image paste.

PTY output is `\r\n`-terminated under ONLCR, so "chunk with `\r` and no ESC" is ordinary plain-text output, not a contrived input.

Second defect on the same line: `armCapture` "is called from the SIGUSR1 handler goroutine" (`wrap.go:1085`), so `p.captureBuffer` is cross-goroutine state guarded by `p.captureMu`. Step 4 above writes to it **outside** the lock — a data race `go test -race` would flag once the path is exercised. Before this diff no write to that backing array existed at all.

`wrap.go:813` (`stripTerminalControls`) has the milder form: it mutates the caller's `data` (the raw PTY chunk). Downstream consumers happen to copy (`stdoutBatcher.append`, `*rolling = append(...)`, `scrollbackFD.Write` all run earlier), so the damage is contained to the second `stripTerminalControls(data)` at `wrap.go:1554` seeing an already-compacted buffer and logging a corrupted near-miss snippet — diagnostic-only, but wrong.

Fix sketch — restore the no-alias contract at the source:

```go
// ansi.go:174
if !hasESC(buf) {
    return append([]byte(nil), buf...)
}
```

(or delete the fast path entirely; `ReplaceAll` allocated unconditionally, so neither costs more than the code being replaced). Then invert `ansi_test.go:91-94`, which currently *asserts* the aliasing:

```go
in := []byte("plain text")
if got := Strip(in); &got[0] != &in[0] {
    t.Error("Strip should not copy when there is nothing to strip")
}
```

That test is what would otherwise re-introduce this. Replace it with a no-mutation property (see §5).

## 3. Important findings

**I1 — The plan still specifies a `Frame(buf, mode)` API the code does not have, with no `## Revisions` entry (AGENTS.md §1).**

The Log records the simplification ("`Frame` has no `Mode`… `Mode` lives on `OSCEnd` alone") and the Done-when explicitly blessed it in advance, so the *code* is right. But the plan was overwritten-by-omission rather than revised, and it now contradicts the shipped signature in five places:

- `plan.md:7` (Architecture) — "`Frame(buf, mode) (size, status)` … and a strictness `Mode`"; "`termcmd` keeps `csiEnd`/`oscEnd` as thin sentinel adapters over it (Lenient)" — termcmd never touches `Frame`.
- `plan.md:78` (export table) — `Frame(buf, mode) (int, Status)`.
- `plan.md:93` (Return contract) — `func Frame(buf []byte, mode Mode) (size int, status Status)`.
- `plan.md:124` (Core concepts) — `**Frame(buf []byte, mode Mode) (int, Status)**`.
- `plan.md:385` (Done-when) — "`wrapcmd` via `Frame(…, Strict)`".

Fix: one `## Revisions` entry (timestamp + reason + delta) recording the Mode collapse, and correct those five sites. This is the artifact the next reader trusts.

**I2 — PQ-9 (Important, still open in the gate ledger) is fixed in code but not in the plan.**

The code answers PQ-9 correctly: Strict's OSC arm falls through to the two-byte class, regex-identical (`ansi.go:115-119`, pinned at `ansi_test.go:33-34` and by the oracle). But PQ-9's other half — "two stated pins are wrong" — is untouched:

- `plan.md:166` — `{"unterminated OSC", "\x1b]0;title", 0}` → the shipped answer is **2**.
- `plan.md:167` — `{"OSC with bare ESC", "\x1b]0;a\x1bZ\x07", 0}` → **2**.
- `plan.md:194` `TestFrameModesDifferOnlyOnValidation` — asserts `Frame(…, Strict) == None` for the bare-ESC OSC (now `Complete`/2), and calls `Frame(…, Lenient)` on a CSI arm that no longer exists (it would not compile).

Same `## Revisions` entry as I1 can carry this. Per the plan-gate carry-forward contract, PQ-9 stays a live finding at its original severity until the plan stops asserting what the code doesn't do.

## 4. Minor findings

- `cmd/internal/termcmd/rename_input.go:215` — `isTerminalFinalByte` now has **zero callers** repo-wide (its only caller was the old `csiEnd` body). PQ-8 predicted exactly this outcome; the plan kept it "to avoid touching its other callers," and there are none. Delete it. (ARCH-DRY residue: a one-line alias no one calls.)
- `atlas/architecture.md:449-451` — "`Status` distinguishes not-a-sequence from truncated, **because** `malformedEscapeSize` feeds its result into `input = input[size:]`". `malformedEscapeSize` goes through `TerminatorScan`, which returns `-1`/length and never sees a `Status`. The reasoning is historical (the rejected design); the package doc at `ansi.go:22-24` gets the tense right ("an earlier design"), the atlas does not.
- Shadow sweep (ARCH-PURPOSE): the plan's "Out of scope (declared, not silent)" section names only `oscRe`. Two other hand-rolled framings remain undeclared — `sgrRe` (`wrap.go:190`, capturing, same class as `oscRe`) and `sgrMouseSize` (`rename_input.go:175`, narrow policy). Both are defensibly out of scope; the point is that the section exists precisely so they're named rather than silent.
- `ansi.TerminatorScan` has no precondition guard — `TerminatorScan([]byte("abc"))` returns 3 because `'c'` is in `[0x40,0x7e]`. Identical to the `csiEnd` it replaces (so no drift), and both current callers guarantee `buf[0]==0x1b`, but as a newly *exported* surface a doc line stating the precondition would help the next consumer.

## 5. Test coverage notes

- The suite is strong on value equivalence and blind to aliasing — which is exactly the gap C1 shipped through. The differential fuzzers compare `Strip(buf)` to `ReplaceAll(buf, nil)` by value; neither notices that one aliases and the other doesn't. Three lines in `FuzzStripMatchesRegexReplaceAll` close it generically:
  ```go
  before := append([]byte(nil), buf...)
  got := Strip(buf)
  if !bytes.Equal(buf, before) { t.Fatalf("Strip mutated its input: %q -> %q", before, buf) }
  ```
  Plus a `wrapcmd`-side regression test: call `stripTerminalControls` on a `\r`-bearing ESC-free buffer and assert the caller's slice is byte-identical afterward. That is the assertion that would have failed on this diff.
- `TestMalformedEscapeSizeNeverReturnsZeroOnNonEmptyInput` (`queries_test.go`) is the right shape — a *property* over the decoder-loop invariant rather than a case table. It correctly covers the `len<2`, unterminated, and SS3 branches.
- Writing `TestCsiEndLenientFramingIsPinned` against the pre-change code first is the discipline that makes "zero behaviour change" checkable. Worth naming in `workshop/lessons.md`.
- Not verifiable here: the Log's `make test` exit 0 and the ~20M fuzz executions. Bash was unavailable; the main agent should re-run after fixing C1, with `-race` on `./cmd/internal/wrapcmd/`.

## 6. Architectural notes

- **ARCH-DRY — pass.** One implementation of sequence structure; both packages derive. Policy tables correctly *not* merged, with the opposed `\x1b[>7u` / `\x1b[>1u` case documented at both ends. Residue: the zero-caller `isTerminalFinalByte` (Minor above).
- **ARCH-PURE — flag (C1).** `ansi` is a genuine pure leaf: stdlib-only imports, every test runs without exec/net/fs. But purity at the seam is more than "no side effects inside" — a pure function that returns a value aliasing its argument imposes a mutation contract on every caller, and the two callers here violate it immediately. The fix is to make the value independent, not to document the hazard.
- **ARCH-PURPOSE — pass.** The issue's purpose was sharing the framing between the two packages; the shadow sweep confirms both consumers now *derive* — `wrapcmd` through `Strip`/`SequenceLen`, `termcmd` through `TerminatorScan`/`OSCEnd`/`IsFinalByte` — with no hand-maintained restatement left behind. `otherEscRe` survives only as a test oracle, which is derivation-from-source in the strongest form: the old implementation is the *check*, not a parallel copy. `scrollbackcmd` uses a real VT emulator and is correctly untouched.
- **ARCH-MOCK — pass, vacuously.** No external binary or service; the plan states this rather than leaving it implied, which is the right form.
- For upcoming work: `ansi` is now the natural home for the "real conformance check against emitters" the issue's Log floats. If that lands, the `Mode` split is the seam to watch — `Strict`/`Lenient` currently encode *which caller asked*, not *which terminal behaves how*, and a conformance check would want the second meaning.

## 7. Plan revision recommendations

Append one `## Revisions` section to `workshop/plans/000128-share-escape-framing-plan.md` (timestamp + reason + delta), covering:

1. **`Frame` lost its `Mode` parameter.** Shipped signature is `Frame(buf []byte) (int, Status)`; `Mode` lives on `OSCEnd` alone. Reason: the Lenient CSI arm had no production consumer (the simplification the Done-when pre-approved). Correct `plan.md:7`, `:78`, `:93`, `:124`, `:385`. In particular `:7`'s "`termcmd` keeps `csiEnd`/`oscEnd` as thin sentinel adapters over it (Lenient)" must read `TerminatorScan` / `OSCEnd(…, Lenient)`.
2. **PQ-9 resolution recorded.** Strict's OSC arm falls through to the two-byte class (regex-identical). Correct the pins at `plan.md:166-167` from `0` to `2`, and rewrite `TestFrameModesDifferOnlyOnValidation` (`plan.md:194`) — its Strict-OSC expectation is wrong and its `Frame(…, Lenient)` cases no longer compile.
3. **`isTerminalFinalByte` has no other callers.** `plan.md:301`'s justification ("Keeping the name avoids touching its other callers") is false; the function is dead. Record the deletion.
4. **Out-of-scope section completed.** Add `sgrRe` (`wrap.go:190`) and `sgrMouseSize` (`rename_input.go:175`) alongside `oscRe`, with the same declared-not-silent reasoning.

---

## Re-review — 2026-07-30T17:34:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 128 — share escape-sequence framing between termcmd and wrapcmd |
| repo | pair |
| issue file | workshop/issues/000128-share-escape-framing.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6ef2c1fc234a4d932ccde58a45554002b7d8c6d6..HEAD |
| command | sdlc close --issue 128 |
| reviewer | claude |
| timestamp | 2026-07-30T17:34:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The extraction itself is good work — `ansi` is a genuine pure leaf, the differential oracle turns "behaves identically" into a measurement, the `termcmd` adapters are byte-for-byte equivalent to the scanners they replace, and the shadow sweep confirms both consumers now derive from one framing. What blocks SHIP is that the prior boundary round's **only Critical finding is still present, unaddressed and unrebutted**, in a commit titled "address close-review REWORK": `ansi.Strip`'s no-ESC fast path returns the caller's slice *aliased* (`ansi.go:180`), and both `wrapcmd` call sites immediately hand that slice to `bytesReplaceAll`, which compacts **in place** (`wrap.go:1163`). The retired `otherEscRe.ReplaceAll` allocated unconditionally, so this is drift on the one Done-when that promised none — and at `wrap.go:1152` the aliased buffer is `p.captureBuffer`, mutex-guarded state whose contents get written to the file nvim reads. The test at `ansi_test.go:90-94` actively *asserts* the aliasing, so it is pinned rather than accidental. Everything else the prior round raised (I1's `Frame(buf, mode)` signature sites, PQ-9's pins, the atlas tense, the shadow-sweep completion, the `TerminatorScan` precondition doc, `isTerminalFinalByte`'s deletion) was fixed cleanly; two residual doc-drift spots remain.

I could not run `make test` — Bash is unavailable in this environment (EPERM on `~/.claude/session-env`, with and without the sandbox). Every finding below is from reading the code and tracing call chains.

## 1. Strengths

- **The oracle is the right proof, and it is load-bearing.** `oracle_test.go:17` keeps `otherEscRe` alive as the differential source of truth, and re-running the seed corpus as a plain table (`oracle_test.go:64`) means the check doesn't depend on anyone starting a fuzz session. The Log is honest about what it does and doesn't prove ("a differential oracle proves *consumer-visible* equivalence, not that every internal distinction is right").
- **Alternative ordering reproduced exactly.** `ansi.go:120-124` — the OSC arm falling through to the two-byte class matches Go regexp's leftmost-*first* alternation, which is why `\x1b]0;title` frames as 2, not "incomplete". Hand-checked against the regex on `\x1b[`, `\x1b(`, `\x1b]`, `\x1b\x1b`, `\x1b[\x00A`, `\x1b[3\x1b[31m`: all agree. `ansi.go:113-119` refusing a two-byte fallback for `'['` so `frameCSI`'s `None` verdict survives is the subtle call.
- **The adapters are byte-faithful.** `queries.go:150` → `TerminatorScan` and `:159` → `OSCEnd(…, Lenient)` are line-for-line identical to the deleted bodies, including the `-1` and `(0,false)` sentinels `rename_input.go:193`/`:208` branch on. `malformedEscapeSize("\x1bOX") == 3` still holds because the scan never looks at `buf[1]` (PQ-5 resolved structurally, not patched).
- **Pins written against the old code first.** `queries_test.go:TestCsiEndLenientFramingIsPinned` and the never-returns-zero property make "this changes nothing" checkable rather than arguable. `TestMalformedEscapeSizeNeverReturnsZeroOnNonEmptyInput` is the right shape — a decoder-loop invariant, not a case table.
- **Policy correctly left alone.** `codexKKPMarkers` / `terminalQueryLiterals` untouched; `run.go`'s `parseSGRMousePress`/`findSGRMousePress`/`isSGRMousePrefix` and `rename_input.go:174` `sgrMouseSize` are literal-prefix *policy* parsers and are correctly out of scope; `scrollbackcmd` emits through a VT emulator and is untouched. Site 3 also got strictly faster: `wrap.go:1019` replaced an O(remaining-buffer) `FindIndex` with an anchored O(sequence) call.

## 2. Critical findings

**C1 — `ansi.Strip` returns a slice aliasing its input; both callers then mutate it in place (`ansi.go:179-182`, consumed at `wrap.go:813` and `wrap.go:1152`). ARCH-PURE. Carried over unfixed from the prior boundary round.**

```go
// ansi.go:179
if !hasESC(buf) {
    return buf          // <- aliases the caller's backing array
}
```

`bytesReplaceAll` (`wrap.go:1162`) is an in-place compactor — `out := b[:0:len(b)]` then `append` — so it writes through that alias. `regexp.ReplaceAll` never did: with no match its `buf` starts nil and the final `append(nil, bsrc[lastMatchEnd:]...)` allocates, so the callers have always operated on a private copy.

Verified failure at `wrap.go:1152` (`maybeFinalizeEarly`, called from `handleChunk` on every chunk while capture is armed):

1. `handleChunk` appends the chunk into `p.captureBuffer` under `captureMu` (`wrap.go:2478`).
2. `maybeFinalizeEarly` snapshots `buf := p.captureBuffer` and **releases the lock** (`wrap.go:1145-1148`).
3. `ansi.Strip(buf)` returns `buf` itself when the accumulated capture has no `0x1b`.
4. `bytesReplaceAll(stripped, '\r')` compacts `p.captureBuffer`'s backing array in place. `p.captureBuffer`'s *length* is unchanged, so the tail keeps stale bytes: an 8-byte `"abc\r\ndef"` becomes `"abc\ndeff"`.
5. Capture stays active; later chunks append after the corrupted region; `finalizeCapture` writes the result to `p.captureOutPath` (`wrap.go:1127`) — the file nvim polls for Alt+i image paste.

PTY output is `\r\n`-terminated under ONLCR, so "chunk with `\r`, no ESC" is ordinary plain-text output. Note the write at step 4 lands outside `captureMu`; `armCapture` (SIGUSR1 goroutine) only rewrites the slice header, so this is not a `-race` hit, but mutating mutex-guarded state after unlocking is a latent hazard on top of the corruption.

`wrap.go:813` (`stripTerminalControls`) has the milder form: it mutates the caller's `data` (the raw PTY chunk). The user-visible stdout path is safe — `stdoutBatcher.append` copies at `wrap.go:360`, and `scrollbackFD.Write`/`*rolling = append(...)` both run before the mutation — but two later readers of `data` in the same `handleChunk` pass see the corrupted buffer: the second `stripTerminalControls(data)` at `wrap.go:1554` (near-miss drift snippet) and `indexByte(data, 0x07)`'s BEL snippet at `wrap.go:2572-2581`. Diagnostic-only, but wrong, and it makes the two `stripTerminalControls(data)` calls in one pass disagree.

Fix at the source — restore the no-alias contract the regex had:

```go
if !hasESC(buf) {
    return append([]byte(nil), buf...)
}
```

(keeps the fast path's benefit — no per-byte `SequenceLen` — at exactly the allocation `ReplaceAll` already paid; deleting the fast path outright is equally fine). Then **invert** `ansi_test.go:90-94`, which currently pins the defect:

```go
in := []byte("plain text")
if got := Strip(in); &got[0] != &in[0] {
    t.Error("Strip should not copy when there is nothing to strip")
}
```

Replace with a no-mutation property (see §5). Leaving that test as-is is what would re-introduce this after any future refactor.

## 3. Important findings

**I1 — Core-concepts prose still describes a `Frame` with modes, consumed by `termcmd` (`plan.md:130`, `:131`, `:135`).** The prior round's I1 was fixed at the signature sites (Architecture line, export table, return contract, Done-when) and Revisions Delta 1 claims "Corrected in … Core concepts", but three bullets still say otherwise:

- `:130` — "consumed by `SequenceLen`/`Strip` (Strict) **and by `termcmd`'s two adapters (Lenient)**" — flatly contradicts `:7`'s "never over `Frame`" and the shipped code.
- `:131` — "The `Mode` knob is the same principle one level down" (under `Frame`'s bullet; `Frame` has no `Mode`).
- `:135` — "`SequenceLen(buf []byte) int` — **`Frame(buf, Strict)`** reduced…".

This is precisely PQ-7's stated hazard: an implementer reading these could restore the `buf[1]` dispatch PQ-5 rejected. Cheap fix: correct the three bullets and extend Delta 1's delta list.

**I2 — Entity-table row contradicts the code for `isTerminalFinalByte` (`plan.md:127`, also `:286`).** The row says `rename_input.go:214` — "modified — one-line delegation to `ansi.IsFinalByte`"; the code **deleted** it (correctly — it had zero callers once `csiEnd` became a one-liner, which I confirmed by repo-wide grep). Task 4's Files paragraph at `:286` still carries the false justification "Keeping the name avoids touching its other callers." The checklist's default for a table-vs-code contradiction is Critical; I'm reporting Important because Revisions **Delta 3** states the deletion explicitly, so the plan does document what shipped — a stale row alongside an explicit correction misleads no one about the code. Update the row to `deleted` and strike `:286`'s justification.

**I3 — No test can catch the C1 class, and the site where it corrupts state has no test at all.** Both fuzz oracles compare `Strip(buf)` to `ReplaceAll(buf, nil)` **by value**; neither notices that one aliases and the other doesn't, which is exactly how C1 shipped through a 20M-execution fuzz session. Separately, grep over `cmd/internal/wrapcmd/*_test.go` finds zero references to `maybeFinalizeEarly`, `captureBuffer`, or `stripTerminalControls` — the image-capture early-finalize path this diff touched is untested end to end. Concrete additions in §5.

## 4. Minor findings

- **PQ-6 (carry-forward, still open, Minor).** The three verbatim enumerated case tables remain in the plan (Task 1 Steps 1/2a, Task 4 Step 1) rather than compressing to strategy lines. The wrong literals it also flagged are now fixed; only the verbosity remains. Still a finding at its original severity, non-blocking.
- **`Status`'s three-way split has no production consumer that branches on it.** `SequenceLen` maps `None` and `Incomplete` both to 0, and `termcmd` reaches the package through `TerminatorScan`/`OSCEnd`, which never see a `Status`. The distinction is defensible as the seam contract (PQ-3), but `ansi.go:22-24` and `atlas/architecture.md:449` state the "pin it in a pending buffer" hazard in the present tense as if a live caller depended on it. The atlas already got the tense right for the loop hazard one sentence later — same treatment here.
- **Issue `## Log` records no boundary-review round.** A REWORK verdict was produced and committed (`workshop/plans/000128-share-escape-framing-close-review.md`) and a commit claims to address it, but `## Log` has no entry for the round or its disposition (AGENTS.md §3). Add it before the close commit — including, explicitly, the disposition of C1.
- **`workshop/lessons.md` has no rule for this class** (grep for alias/in-place/`ReplaceAll` finds nothing relevant). Once C1 is fixed, AGENTS.md §4 wants the rule: *a pure helper that may return a slice aliasing its input imposes a mutation contract on every caller — when it replaces something that always allocated, callers that compact in place silently corrupt the source.*

## 5. Test coverage notes

- **Input-immutability property, three lines, closes the generic gap** — add to both `FuzzSequenceLenMatchesRegex` and `FuzzStripMatchesRegexReplaceAll`:
  ```go
  before := append([]byte(nil), buf...)
  got := Strip(buf)
  if !bytes.Equal(buf, before) { t.Fatalf("Strip mutated its input: %q -> %q", before, buf) }
  ```
- **A `wrapcmd`-side regression for the site that persists the damage.** Drive `maybeFinalizeEarly` with `p.captureActive = true` and `p.captureBuffer = []byte("hello\r\nworld")` (no ESC, no image marker), then assert `p.captureBuffer` is byte-identical afterward. That is the assertion that fails on this diff and passes after the fix — and it gives the untested early-finalize path its first test.
- The `ansi_test.go:90-94` aliasing assertion must be inverted, not just left green (see C1).
- Unverifiable here: the Log's `make test` exit 0, ~20M fuzz executions, and #127's 8M `FuzzStripTerminalQueries` execs. After fixing C1, re-run the full suite (`env -u PAIR_SESSION_ID -u PAIR_TAG -u PAIR_PANE_CWD make test`) and add `-race` on `./cmd/internal/wrapcmd/`.

## 6. Architectural notes

- **ARCH-DRY — pass.** One implementation of sequence structure; `wrapcmd` derives via `Strip`/`SequenceLen`, `termcmd` via `TerminatorScan`/`OSCEnd`/`IsFinalByte`. Policy tables correctly not merged, with the opposed `\x1b[>7u`/`\x1b[>1u` case documented at both ends (`ansi.go:5-9`, `atlas/architecture.md:435`). The previous round's residue (`isTerminalFinalByte`) is gone; no other generic framing survives in `cmd/internal`.
- **ARCH-PURE — flag (C1).** `ansi` is a genuine pure leaf — `ansi.go` imports nothing, tests import only `testing`/`bytes`/`regexp`, no exec/net/fs. But purity at a seam is more than "no side effects inside": returning a value that aliases the argument exports a mutation contract, and both callers violate it on the first line after the call. Make the value independent; don't document the hazard.
- **ARCH-PURPOSE — pass.** Shadow sweep run: every framing consumer derives from the source. `otherEscRe` survives only as the test oracle, which is the strongest form of derivation — the old implementation is the *check*, not a parallel copy. The three hand-rolled parsers left behind (`oscRe`, `sgrRe`, `sgrMouseSize`) are capturing or policy-specific and are declared in the plan's "Out of scope" section rather than left silent.
- **ARCH-MOCK — pass, vacuously.** No external binary or service; the plan states this at `:143` rather than leaving it implied, which is the right form.
- For upcoming work: `ansi` is the natural home for the "conformance check against real emitters" the issue's Log floats. If that lands, watch the `Mode` split — `Strict`/`Lenient` currently encode *which caller asked*, not *which terminal behaves how*, and a conformance check wants the second meaning.

## 7. Plan revision recommendations

Extend the existing `## Revisions` section (new dated entry, or amend Delta 1/3 with a note — don't overwrite):

1. **Delta 1 is incomplete.** Correct `plan.md:130` ("consumed by … `termcmd`'s two adapters (Lenient)" → `Frame` has no `termcmd` consumer), `:131` (drop "The `Mode` knob is the same principle one level down" from `Frame`'s bullet — `Mode` belongs to `OSCEnd`), and `:135` (`Frame(buf, Strict)` → `Frame(buf)`).
2. **Delta 3 is not reflected in the entity table.** Change `plan.md:127`'s Status from "modified — one-line delegation" to "deleted (zero callers once `csiEnd` became a delegation)", and strike the false justification at `:286`.
3. **Record `Strip`'s no-alias contract once C1 is fixed** — the plan's `Strip` bullet (`:138`) specifies only which bytes are removed, never that the result must not share storage with the input. That silence is what let the fast path in. State it: *`Strip` never returns a slice aliasing `buf`; callers compact the result in place.*
