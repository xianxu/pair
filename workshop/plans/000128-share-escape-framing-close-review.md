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
