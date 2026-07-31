# Shared ANSI escape framing — `cmd/internal/ansi`

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One answer to "where does the escape sequence at `buf[0]` end", used by both `wrapcmd` and `termcmd`, with the policy tables left exactly where they are.

**Architecture:** A new leaf package `cmd/internal/ansi` with ONE scanner of escape-sequence structure, `Frame(buf) (size, status)`, exposing three outcomes (`Complete`/`Incomplete`/`None`). `wrapcmd` consumes it through `SequenceLen`/`Strip` (Strict, regex-equivalent); `termcmd` keeps `csiEnd`/`oscEnd` as thin sentinel adapters over `TerminatorScan` / `OSCEnd(…, Lenient)` — never over `Frame` — so its interactive rename decoder is untouched. The load-bearing risk is behavioural drift at three `wrapcmd` call sites feeding scrollback capture and agent-output detection, so the change is anchored by **differential fuzz tests** against the retired regex, kept alive in test code purely as the oracle.

**Tech Stack:** Go stdlib only. No new dependencies; `regexp` disappears from the production path for this concern.

---

## Home decision (Plan step 1, settled)

**New `cmd/internal/ansi`.** The alternatives fail on dependency direction, which is the criterion the issue set:

- Hosting in `termcmd` → `wrapcmd` imports `termcmd`, a PTY/mux package with a `Runtime` seam. Wrong direction and a heavy dependency for a string function.
- Hosting in `wrapcmd` → `termcmd` imports `wrapcmd`, the codex-marker/scrollback package. Same problem mirrored.
- `ansi` is a leaf: imports nothing but stdlib, imported by both. Acyclic, and it matches the flat `cmd/internal/<pkg>` layout (`textwidth`, extracted in #132, set the precedent for a tiny shared-rule package).

## The two implementations differ in TWO axes — and both consumers need their own answer

Verified against the code after the plan-quality gate found my first reading wrong.

### Axis 1 — coverage

`wrapcmd.otherEscRe` (`wrap.go:189`) is four alternatives:

| # | pattern | covers |
|---|---|---|
| 1 | `\x1b\[[0-?]*[ -/]*[@-~]` | CSI: params `0x30-0x3F`, intermediates `0x20-0x2F`, final `0x40-0x7E` |
| 2 | `\x1b\][^\x07\x1b]*(?:\x07\|\x1b\\)` | OSC terminated by BEL or ST |
| 3 | `\x1b[()*+][@-~]` | charset designation (`ESC ( B`) |
| 4 | `\x1b[@-Z\\-_]` | two-byte escapes (`ESC M`) |

`termcmd` frames CSI + OSC (`queries.go:136,147`) and treats **any** other second byte as a 2-byte escape (`queries.go:126` `default: return 2`) — broader than alternative 4, which restricts to `0x40-0x5A`/`0x5C-0x5F`.

### Axis 2 — strictness, and it cannot simply be unified

`termcmd`'s `csiEnd` scans to any final byte **without validating** param/intermediate ranges; `oscEnd` scans *past* a bare ESC where the regex's `[^\x07\x1b]*` stops at one.

**My first plan claimed this difference was unreachable because `csiEnd`/`oscEnd` sit behind a query-literal prefix match. That was wrong** — only the `oscEnd` at `queries.go:102` is on the literal path. The `csiEnd` at `:115` and `oscEnd` at `:121` are the **fallback** arm of `terminalSequenceAt`'s `switch buf[1]`, reached for arbitrary input. So the strictness difference is live, and forcing `termcmd` onto strict framing would change how malformed input is consumed.

That matters most **outside** `queries.go`. `csiEnd` has two further call sites the first plan never found:

- `rename_input.go:193` — `escapeSequenceIncomplete` returns `csiEnd(input) < 0`, i.e. the `-1` sentinel *is* the "incomplete" signal.
- `rename_input.go:208` — `malformedEscapeSize` does `if end := csiEnd(input); end >= 0 { return end }` and otherwise consumes `len(input)`.

`malformedEscapeSize`'s result feeds `input = input[size:]` in the decoder loop (`rename_input.go:117-120`). **A `0` return there is a zero-advance infinite loop**, and strict framing would also make a malformed-but-complete CSI consume the whole buffer instead of just the sequence — swallowing the user's next keystrokes mid-rename.

### Decision: one scanner, an explicit mode, no behaviour change anywhere

What is genuinely duplicated is the *structure* of a sequence (`ESC [ params intermediates final`, `ESC ] … BEL|ST`, charset, two-byte). What legitimately differs is whether out-of-range bytes abort. So the shared helper carries **one implementation of the structure and a documented strictness knob** — the same shape as this issue's existing "share the framing, never the policy" split, one level down:

- `wrapcmd` uses **Strict** (regex-equivalent) → its three call sites are provably unchanged, held by the differential fuzzers.
- `termcmd` uses **Lenient** → `csiEnd`/`oscEnd` become thin sentinel adapters, so `rename_input.go` and the fallback arm keep their exact current behaviour.

This is a smaller DRY win than "one function, one semantics", and it is the honest one: the alternative silently changed how a malformed escape is consumed in an interactive rename path.

### `csiEnd` is misnamed — it is an introducer-independent terminator scan (PQ-5)

Measured, not read: `csiEnd` ignores `buf[1]` entirely and scans from index 2 for the first final byte. `malformedEscapeSize` (`rename_input.go:205`) routes **SS3** through it on purpose (`input[1] != '[' && input[1] != 'O'` → falls through to `csiEnd`), and it works:

```
csiEnd("\x1b[31m")  = 5      csiEnd("\x1bOX") = 3
csiEnd("\x1b[\x00A") = 4      csiEnd("\x1bO@") = 3
```

So an adapter that dispatches on `buf[1]` would classify `'O'` (0x4F) as a two-byte escape, return 2, and the decoder would insert the `X` into the tab name. `TestDecodeRenameInputConsumesUnknownEscapeTerminators` (`rename_input_test.go:150`) covers exactly `"\x1bOX"` and `"\x1bO@"` and would fail — which is how a plan claiming "identical by construction" gets caught by a test that already exists.

**Resolution: the shared package exposes the primitives each consumer actually needs, rather than one call with a mode.** The duplication being removed is byte-level *knowledge*, and each rule still has exactly one implementation:

| `ansi` export | the one rule it owns | consumer |
|---|---|---|
| `IsFinalByte(c) bool` | `0x40-0x7E` | everything below; replaces `termcmd.isTerminalFinalByte` |
| `TerminatorScan(buf) int` | from index 2 to the first final byte, `-1` if none — **introducer-independent** | `termcmd.csiEnd` |
| `OSCEnd(buf, mode) (int, bool)` | BEL or `ESC \`; `Strict` aborts on a bare ESC, `Lenient` scans past | `termcmd.oscEnd`, and `Frame`'s OSC arm |
| `Frame(buf) (int, Status)` | the sequence at `buf[0]` across `wrapcmd`'s four classes | `SequenceLen`, `Strip` |

`Frame` is built on the three primitives, so nothing is implemented twice. What the first two plans got wrong was assuming both consumers wanted the same *entry point*; they want the same *rules*.

### Return contract (PQ-3): three outcomes, not one int

`csiEnd`'s `-1`, the regex's no-match, and `queries.go:126`'s "consume 2" are three *different* answers that consumers branch on. Collapsing them into `0` is what produced the infinite loop, so the API names all three:

```go
type Status int
const (
    None       Status = iota // buf does not start an escape sequence
    Incomplete               // starts one, but it is truncated — caller must carry the tail
    Complete                 // fully framed; Size is its length
)
func Frame(buf []byte) (size int, status Status)
```

Consumer mapping, stated so each is checkable:

| consumer | today | under `Frame` |
|---|---|---|
| `wrapcmd` (3 sites) | regex match / no match | `Complete` → `size`; `Incomplete`/`None` → 0 |
| `termcmd.csiEnd` | length or `-1` | `ansi.TerminatorScan` — same `-1`. **Not** `Frame`: see PQ-5 |
| `termcmd.oscEnd` | `(length, true)` / `(0, false)` | `ansi.OSCEnd(buf, Lenient)` — same pair |
| `terminalSequenceAt` default arm | `2` | unchanged — stays in `queries.go`, not pushed into `ansi` |

### Out of scope (declared, not silent)

`wrapcmd.oscRe` (`wrap.go:191`) is a fifth OSC framing, but it **captures** (`\x1b\](\d+);([^\x07\x1b]*)…`) rather than only framing, and its consumers want the parameter and payload. Extracting a capturing variant is a different job; left alone deliberately so `ARCH-DRY` is answered rather than ignored.

Two more hand-rolled framings stay put, named here so the sweep is complete rather than selective:

- `wrapcmd.sgrRe` (`wrap.go:190`) — same class as `oscRe`: it **captures** the SGR parameter list for `extractFG`, so a framing-only helper does not serve it.
- `termcmd.sgrMouseSize` (`rename_input.go:175`) — narrow *policy*, not framing: it recognises `\x1b[<` SGR mouse reports specifically, which is a decision about which sequences the rename decoder cares about. Merging it into `ansi` would move policy into the shared layer, the exact mistake this issue exists to avoid.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Frame` | `cmd/internal/ansi/ansi.go` | new |
| `IsFinalByte` / `TerminatorScan` / `OSCEnd` | `cmd/internal/ansi/ansi.go` | new |
| `Mode` / `Status` | `cmd/internal/ansi/ansi.go` | new |
| `SequenceLen` | `cmd/internal/ansi/ansi.go` | new (Strict convenience) |
| `Strip` | `cmd/internal/ansi/ansi.go` | new |
| `otherEscRe` | `cmd/internal/wrapcmd/wrap.go` | deleted (survives in `ansi/oracle_test.go` as the differential oracle) |
| `csiEnd` / `oscEnd` | `cmd/internal/termcmd/queries.go` | modified — bodies become one-line adapters over `TerminatorScan` / `OSCEnd`; names and sentinels stay |
| `isTerminalFinalByte` | `cmd/internal/termcmd/rename_input.go` | deleted — zero callers once `csiEnd` became a delegation |

- **Frame(buf []byte) (int, Status)** — the one place that knows what an escape sequence looks like.
  - **Relationships:** consumed by `SequenceLen`/`Strip` only. `termcmd` does NOT consume `Frame` — its adapters go to `TerminatorScan`/`OSCEnd`. No state, no IO.
  - **DRY rationale:** collapses the regex and the two scanners into one description of sequence *structure*. The *policy* tables (`wrapcmd`'s codex marker list, `termcmd`'s `terminalQueryLiterals`) deliberately stay put — they are opposed in at least one case (`\x1b[>7u` stripped by one, `\x1b[>1u` required to survive by the other), and merging them would be the actual bug. `OSCEnd`'s `Mode` knob is the same principle one level down: shared structure, per-consumer strictness. `Frame` itself has no mode.
  - **Why `Status` and not an int:** `csiEnd`'s `-1`, the regex's no-match and `queries.go:126`'s "consume 2" are three different answers consumers branch on. A single `0` conflates them, and in `malformedEscapeSize` → `input = input[size:]` that conflation is a **zero-advance infinite loop**.
  - **Future extensions:** a `Kind` (CSI/OSC/charset/two-byte) if a caller ever needs to branch on class; no caller does today (YAGNI).

- **SequenceLen(buf []byte) int** — `Frame(buf)` reduced to the regex's answer: length when `Complete`, else `0`. The only form `wrapcmd` needs.
  - **Incomplete input returns 0**, which is load-bearing for `p.stdoutPending`'s tail carry (`wrap.go:310`): a scanner that consumed a partial sequence would corrupt every chunk edge.

- **Strip(buf []byte) []byte** — `buf` with every `Complete` Strict sequence removed; an incomplete trailing sequence is preserved for the caller's carry.
  - **CONTRACT: the result never aliases `buf`**, including when nothing is stripped. Both `wrapcmd` callers pipe it into `bytesReplaceAll`, an in-place compactor, so an aliased return rewrites the caller's own buffer — corrupting `p.captureBuffer` outside its mutex. The retired regex allocated unconditionally; this silence in the first draft is exactly what let a "fast path" in.
  - **DRY rationale:** the two `ReplaceAll` sites (`wrap.go:812`, `:1151`) are the same operation.

### Integration points

None. `ansi` is pure, both consumers already own their IO, and no external binary or service is involved — so no new fake (`ARCH-MOCK` is satisfied vacuously, and the plan says so rather than leaving the reader to wonder).

---

## Task 1: `ansi.Frame` + `SequenceLen`/`Strip`, with the regex as oracle

**Files:**
- Create: `cmd/internal/ansi/ansi.go`, `cmd/internal/ansi/ansi_test.go`, `cmd/internal/ansi/oracle_test.go`

- [ ] **Step 1: Write the failing table test**

```go
func TestSequenceLen(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"CSI SGR", "\x1b[31mX", 5},
		{"CSI with intermediate", "\x1b[?1006h", 8},
		{"CSI no params", "\x1b[H", 3},
		{"OSC BEL", "\x1b]0;title\x07rest", 10},
		{"OSC ST", "\x1b]0;t\x1b\\rest", 7},
		{"charset designation", "\x1b(B", 3},
		{"two-byte escape", "\x1bM", 2},
		{"not an escape", "hello", 0},
		{"bare ESC at EOF", "\x1b", 0},
		{"incomplete CSI", "\x1b[31", 0},        // stream boundary: caller must carry
		{"unterminated OSC falls back to two-byte", "\x1b]0;title", 2}, // `]` is 0x5D — see Revisions
		{"OSC with bare ESC falls back too", "\x1b]0;a\x1bZ\x07", 2}, // OSC arm fails, two-byte matches
		{"empty", "", 0},
	}
	for _, c := range cases {
		if got := SequenceLen([]byte(c.in)); got != c.want {
			t.Errorf("%s: SequenceLen(%q) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

func TestStripRemovesSequencesKeepsText(t *testing.T) {
	if got := string(Strip([]byte("\x1b[31mred\x1b[0m done"))); got != "red done" {
		t.Errorf("Strip = %q", got)
	}
	// A trailing incomplete sequence is KEPT, not silently eaten — the caller's
	// tail-carry decides, and dropping it here would lose a byte at every chunk edge.
	if got := string(Strip([]byte("ok\x1b[3"))); got != "ok\x1b[3" {
		t.Errorf("Strip incomplete tail = %q, want it preserved", got)
	}
}
```

- [ ] **Step 2:** Run → FAIL (undefined). `go test ./cmd/internal/ansi/`

- [x] **Step 2a: Pin the ONE genuine mode split — on `OSCEnd`, not `Frame`**

`Frame` has no `Mode` (see Revisions). The only strictness split with two real
consumers is OSC bare-ESC handling, so that is what gets the test: `OSCEnd(…, Strict)`
must refuse to scan past a bare ESC and `OSCEnd(…, Lenient)` must scan past it, with
both agreeing on well-formed input. Truncation must be `Incomplete` in `Frame`, never
`None`, or a caller's tail carry turns into a dropped byte.

- [ ] **Step 3: Implement.** A switch on `buf[1]`: `[` → CSI (params `0x30-0x3F`*, intermediates `0x20-0x2F`*, final `0x40-0x7E`), `]` → OSC (scan to BEL or `ESC \`, abort on a bare ESC), `(`/`)`/`*`/`+` → 3 bytes if the third is `0x40-0x7E`, else `0x40-0x5A` or `0x5C-0x5F` → 2. Anything else, or running out of input → `0`.

- [ ] **Step 4: The differential oracle — this is what makes "behaves identically" a fact**

`oracle_test.go` keeps `otherEscRe` alive in *test* code only:

```go
// The retired regex, preserved as the oracle. wrapcmd's three call sites have a
// "must behave identically" Done-when (#128), and the only honest way to hold that
// is to compare against what they used to run — not to re-read both and reason.
var otherEscRe = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()*+][@-~]|\x1b[@-Z\\-_]`)

func FuzzSequenceLenMatchesRegex(f *testing.F) {
	for _, s := range []string{
		"\x1b[31mX", "\x1b]0;t\x07", "\x1b(B", "\x1bM", "plain",
		"\x1b[", "\x1b]0;unterminated", "\x1b]0;a\x1bZ\x07", "\x1b[>1u", "\x1b[>7u",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		got := SequenceLen(buf)
		loc := otherEscRe.FindIndex(buf)
		want := 0
		if loc != nil && loc[0] == 0 {
			want = loc[1]
		}
		if got != want {
			t.Errorf("SequenceLen(%q) = %d, regex says %d", buf, got, want)
		}
	})
}

// Strip must equal ReplaceAll on the same input, for the two ReplaceAll sites.
func FuzzStripMatchesRegexReplaceAll(f *testing.F) { /* same corpus; compare Strip(b) to otherEscRe.ReplaceAll(b, nil) */ }
```

- [ ] **Step 5:** Run the fuzzers briefly and fix any disagreement — the scanner is wrong, not the oracle.

```bash
go test ./cmd/internal/ansi/ -run Fuzz -fuzz FuzzSequenceLenMatchesRegex -fuzztime 30s
go test ./cmd/internal/ansi/ -run Fuzz -fuzz FuzzStripMatchesRegexReplaceAll -fuzztime 30s
```
Expected: no failures. Commit any crashers found into `testdata/` as permanent cases.

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/ansi/ && git commit -m "#128: ansi.SequenceLen/Strip with the retired regex as differential oracle"
```

## Task 2: repoint `wrapcmd`'s two `ReplaceAll` sites

**Files:** Modify `cmd/internal/wrapcmd/wrap.go:812` (`stripTerminalControls`), `:1151` (capture-early path)

- [ ] **Step 1:** Confirm the three pinned test files pass BEFORE touching anything, so a later failure is attributable: `go test ./cmd/internal/wrapcmd/ -run 'StdoutFilter|ExtractFG|UpdateAgentOutput' -v`
- [ ] **Step 2:** Replace both `otherEscRe.ReplaceAll(x, nil)` with `ansi.Strip(x)`.
- [ ] **Step 3:** Re-run the same three files. Expected: identical pass set.
- [ ] **Step 4: Commit**

## Task 3: repoint the colored-run walker (`wrap.go:1018`)

**Files:** Modify `cmd/internal/wrapcmd/wrap.go` around `:1018`; then delete `otherEscRe` from `wrap.go:189`

The awkward site the issue flagged — but only in shape, not in meaning: `FindIndex(data[i:])` guarded by `loc[0] == 0` **is** "length of the sequence at `data[i]`", which is `SequenceLen`'s exact contract.

- [ ] **Step 1:** Rewrite as:

```go
if n := ansi.SequenceLen(data[i:]); n > 0 {
	// … unchanged placeholder-space comment and body …
	i += n
	continue
}
```
The `loc[1]` that fed `i += loc[1]` becomes `n`. Keep the CUF/placeholder comment verbatim — it documents a real Claude-TUI behaviour and is not part of this change.

- [ ] **Step 2:** Delete `otherEscRe` from the production `var` block. `go build ./...` must pass; if anything else still references it, that call site was missed — find it rather than keeping the regex.
- [ ] **Step 3:** Run the full `wrapcmd` package, not just the three files: `go test ./cmd/internal/wrapcmd/`
- [ ] **Step 4: Commit**

## Task 4: `termcmd` adapters — Lenient mode, sentinels preserved

**Files:** Modify `cmd/internal/termcmd/queries.go` (`csiEnd`, `oscEnd` become adapters) and **exactly one line** of `cmd/internal/termcmd/rename_input.go` — `isTerminalFinalByte` is defined there and `queries.go`'s old `csiEnd` body was its ONLY caller, so once `csiEnd` becomes a delegation the function is dead and is deleted. (An earlier draft kept it as an alias "to avoid touching its other callers" — there are none; Go does not flag an unused package-level func, which is why this needed checking rather than assuming.) No other `rename_input.go` behaviour changes.

The first plan said "delete `csiEnd`/`oscEnd`, they're only reached after a query-literal match". Both halves were wrong: `csiEnd` at `queries.go:115` and `oscEnd` at `:121` are the **fallback** arm of `terminalSequenceAt` (arbitrary input), and `csiEnd` has two further callers in `rename_input.go` whose behaviour depends on its `-1` sentinel. Keeping the names as adapters is what makes this a zero-behaviour-change refactor.

- [ ] **Step 1: Pin today's behaviour BEFORE touching it — including the malformed cases**

```go
// csiEnd is Lenient on purpose: it scans to any final byte without validating the
// param/intermediate ranges. rename_input.go depends on that — malformedEscapeSize
// feeds its result straight into `input = input[size:]`, so a stricter framing would
// consume the whole buffer (swallowing the user's next keystrokes mid-rename) and a
// 0 return would be a zero-advance infinite loop (#128).
func TestCsiEndLenientFramingIsPinned(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"plain CSI", "\x1b[31m", 5},
		{"unterminated", "\x1b[31", -1},
		{"out-of-range param byte still frames", "\x1b[\x00A", 4}, // strict would reject
		{"private-mode query", "\x1b[?1006h", 8},
		// SS3 — csiEnd is introducer-independent and rename_input relies on it.
		{"SS3", "\x1bOX", 3},
		{"SS3 with @ final", "\x1bO@", 3},
	}
	for _, c := range cases {
		if got := csiEnd([]byte(c.in)); got != c.want {
			t.Errorf("%s: csiEnd(%q) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

// The decoder consumes malformedEscapeSize bytes per iteration; a 0 would spin.
func TestMalformedEscapeSizeNeverReturnsZeroOnNonEmptyInput(t *testing.T) {
	for _, in := range []string{"\x1b[", "\x1b[\x00A", "\x1bZ", "\x1b", "\x1b[31m"} {
		if got := malformedEscapeSize([]byte(in)); got <= 0 {
			t.Errorf("malformedEscapeSize(%q) = %d — the decoder loop would not advance", in, got)
		}
	}
}
```

Run: `go test ./cmd/internal/termcmd/ -run 'CsiEndLenient|MalformedEscapeSize' -v`
Expected: **PASS against the current code** — these pin what exists, so a change that breaks them is caught rather than argued about.

- [ ] **Step 2:** Rewrite the two bodies as adapters, keeping the signatures and sentinels:

```go
// NOT ansi.Frame: csiEnd is introducer-independent — malformedEscapeSize routes SS3
// (\x1bO…) through it, and a buf[1] dispatch would frame that as a 2-byte escape and
// leak the final byte into the tab name (#128 PQ-5).
func csiEnd(buf []byte) int { return ansi.TerminatorScan(buf) }

func oscEnd(buf []byte) (int, bool) { return ansi.OSCEnd(buf, ansi.Lenient) }
```

Both must keep their existing doc comments, extended with a line saying the framing now lives in `ansi` and the sentinel mapping is deliberate.

- [ ] **Step 3:** Re-run Step 1's pins plus the whole package: `go test ./cmd/internal/termcmd/`. Expected: identical pass set — this task changes no behaviour by construction.

- [ ] **Step 4:** Run #127's structural guard, which exists for exactly this surface, since over-strip is the asymmetric danger:

```bash
go test ./cmd/internal/termcmd/ -run Fuzz -fuzz FuzzStripTerminalQueries -fuzztime 30s
```

- [ ] **Step 5: Commit**

```bash
git commit -am "#128: termcmd frames via ansi.TerminatorScan/OSCEnd; sentinels preserved, SS3 pinned"
```

## Task 5: atlas + close

- [ ] **Step 1:** `atlas/architecture.md:433` — the opposed-policy note must still read true. Update it to say the *framing* is now shared via `cmd/internal/ansi` while the two policy tables remain deliberately separate and in one case opposed. This is a Done-when item, not optional.
- [ ] **Step 2:** Sweep for stale references to the retired identifiers, **with a flag that exists**: `/usr/bin/grep -rn "otherEscRe\|csiEnd\|oscEnd" --include='*.go' --include='*.md' .` — ugrep's `--no-ignore` is not a valid option and exits 2 silently (see `workshop/lessons.md`). Expect hits only in `ansi/oracle_test.go` and history.
- [ ] **Step 3:** `env -u PAIR_SESSION_ID -u PAIR_TAG -u PAIR_PANE_CWD make test` → exit 0. `termcmd`/`wrapcmd` pty tests need the sandbox disabled.
- [ ] **Step 4:** `sdlc close`.

## Done when

- One implementation of escape-sequence structure (`ansi.Frame`). `otherEscRe` is gone from production code; `csiEnd`/`oscEnd` survive **as one-line adapters** over it — deleting the names would have meant rewriting `rename_input.go`'s sentinel contract, which is out of scope for a refactor whose whole claim is zero behaviour change.
- The three `otherEscRe` call sites behave identically, held by `stdout_filter_test.go`, `extract_fg_test.go`, `update_agent_output_test.go` **plus** the differential fuzzers against the retired regex.
- Both packages get their byte-level rules from `ansi` and neither implements one twice: `wrapcmd` via `Frame`, `termcmd` via `TerminatorScan` + `OSCEnd(…, Lenient)`. `csiEnd` must stay **introducer-independent** (SS3 goes through it) — pinned by `TestCsiEndLenientFramingIsPinned`'s `\x1bOX`/`\x1bO@` cases and by the existing `TestDecodeRenameInputConsumesUnknownEscapeTerminators`.
- `Frame`'s Lenient CSI arm has no production consumer today; it exists so `Frame` has one code path per mode. If that reads as dead weight at review, collapsing `Frame` to Strict-only is the acceptable simplification — **restoring a `buf[1]` dispatch inside `csiEnd` is not**.
- `malformedEscapeSize` never returns 0 on non-empty input (the decoder-loop guard).
- The opposed-policy note in `atlas/architecture.md` still reads true.
- `make test` exit 0.

## Risks

- **Silent over-strip** is the asymmetric failure (issue Log): it removes mouse mode / Kitty encoding / cursor shape, where a missed query degrades benignly. Mitigated by choosing the stricter semantics and by the differential fuzzers.
- **Stream-boundary regression.** `SequenceLen` returning 0 for an incomplete sequence is load-bearing for `p.stdoutPending`'s tail carry. Pinned by the `Strip` incomplete-tail test; a scanner that "helpfully" consumed a partial sequence would corrupt every chunk edge.
- **A missed call site** would leave `otherEscRe` alive; Task 3 Step 2 deletes it outright so the compiler finds them.

## Revisions

**2026-07-30 — `Frame` shipped without a `Mode` parameter; two Strict-OSC pins in
this plan were wrong.** Recorded per AGENTS.md §1 rather than silently overwriting,
because the plan is the artifact the next reader trusts.

**Delta 1 — `Frame(buf, mode)` → `Frame(buf)`.** The Lenient CSI arm had no
production consumer: `wrapcmd` wants Strict, and `termcmd` reaches the shared code
through `TerminatorScan`/`OSCEnd`, never through `Frame`. Keeping a mode parameter
would have shipped a code path nothing exercised. The Done-when pre-approved exactly
this collapse ("collapsing `Frame` to Strict-only is the acceptable simplification —
restoring a `buf[1]` dispatch inside `csiEnd` is not"), and `Mode` now lives on
`OSCEnd` alone, which is the one function with two consumers wanting different
answers. Corrected in the Architecture line, the export table, the return-contract
snippet, Core concepts, and the Done-when.

**Delta 2 — the Strict-OSC pins said `0`; the answer is `2`.** `]` is 0x5D, which
sits **inside** the two-byte escape class `[0x5C-0x5F]`. So when the OSC arm fails —
unterminated, or containing a bare ESC — the regex did not report "no match": it fell
through to its fourth alternative and matched `\x1b]` as a two-byte escape, leaving
the payload as text. The shipped `Frame` reproduces that fall-through, and the
differential oracle is what caught the plan's expectation being wrong. Task 1's table
now pins `2`, and Step 2a was rewritten: its old body asserted `Frame(…, Strict)` is
`None` for the bare-ESC OSC and called `Frame(…, Lenient)`, so it was both wrong and
uncompilable against the shipped API.

**Delta 3 — `isTerminalFinalByte` deleted, not delegated.** Task 4 justified keeping
the name "to avoid touching its other callers". It had none: its only caller was the
old `csiEnd` body, so once that became a one-line delegation the function was dead. A
one-line alias nobody calls is `ARCH-DRY` residue, so it is gone and `rename_input.go`
loses the `ansi` import it briefly gained.

**2026-07-30 (second entry) — `Strip` must not alias; three plan sites still described
the pre-collapse API.** Raised by the close review, which returned REWORK twice.

**Delta 4 — the `Strip` no-alias contract, and the Critical it hid.** The first draft
specified only *which bytes* `Strip` removes, never that the result must not share
storage with the input. That silence let an "obvious" fast path in — return `buf`
untouched when it holds no ESC. Both `wrapcmd` callers pipe `Strip`'s result straight
into `bytesReplaceAll`, an **in-place** compactor, so on ESC-free input that wrote
through to the caller's own buffer: `p.captureBuffer`'s backing array is rewritten
while its length stays put, leaving stale duplicated bytes in the tail
(`"hello\r\nworld"` → `"hello\nworldd"`), and the corrupted capture is what nvim reads
for Alt+i image paste. It also wrote outside `p.captureMu`, racing the SIGUSR1
handler. Under ONLCR, "plain chunk with `\r` and no ESC" is the ordinary case, not a
contrived one. Reproduced against both versions before and after the fix. The contract
is now stated in Core concepts and pinned by a test that mutates `Strip`'s result and
asserts the input survives — replacing an earlier test that asserted the aliasing was
*correct*.

**Delta 5 — Delta 1 was itself incomplete.** Three further sites still described
`Frame` as `termcmd`'s dependency or took a `Strict` argument; corrected.

**Delta 6 — `isTerminalFinalByte` deleted, and the stated reason corrected.** Task 4's
justification for keeping it ("avoids touching its other callers") was false: it had
none.

