# Right-pane ESC deadline Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A bare ESC typed in the right pane reaches the child after one
deadline instead of waiting for the next keystroke, and a chord split across
reads still fires when its tail arrives before that deadline.

**Architecture:** `pumpStdin` (`cmd/internal/termcmd/run.go`) already holds a
chord/mouse prefix in `held` and already owns one timer — but that timer is
armed only by the rename decoder. This plan makes the timer the pump's single
*escape-ambiguity deadline*: whichever component owns the pending bytes owns
the timer (rename session → `decoder.Pending`; otherwise → `held`), and on
expiry the owner resolves its pending bytes (rename → cancel; plain → forward
to the child). The deadline value moves out of `couchtty` into
`workbenchshortcut`, the package that owns the chord table that makes ESC
ambiguous in the first place, so all four framers in the repo read one
constant (ARCH-DRY). No new goroutine, no new channel: the `select` gains no
arms, one arm gains a second branch.

**Tech Stack:** Go. `cmd/internal/termcmd` (the `pair term` mux),
`cmd/internal/workbenchshortcut` (chord table), `cmd/internal/couchtty`
(couch's two framers, consumer of the constant only).

**Relationship to #227.** #227 makes role-scoped chords pass through to a
full-screen child. It does not fix this issue — the hold happens *before*
`Decide` runs — and this issue does not fix #227. They compound in one place:
after this plan, an `ESC`,`j` typed inside 35 ms in nvim still decodes as
`ChordAltJ` and is swallowed by the right-terminal role; #227's passthrough
would deliver those bytes to nvim instead, closing the residual. This plan
leaves that residual and records it; #227 is the next issue.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `EscapeAmbiguity` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `escapeAmbiguity` | `cmd/internal/couchtty/keys.go` | deleted |
| `EscapeTimer` | `cmd/internal/termcmd/run.go` | modified |

- **EscapeAmbiguity** — `35 * time.Millisecond`; how long an ESC-led prefix
  may stay pending before it is resolved as typed bytes. Exported, one
  definition.
  - **Relationships:** read by four framers: couch's input framer and panel
    framer (`couchtty/console.go`), termcmd's rename decoder arm and termcmd's
    main-loop arm (both in `run.go`'s `pumpStdinWithTimer`).
  - **DRY rationale:** replaces `couchtty.escapeAmbiguity` (35 ms) and the
    literal `50 * time.Millisecond` in the rename arm. Two values for one
    question was the bug's cousin.
  - **Future extensions:** none planned; if a framer ever needs a different
    deadline, that is a second named constant with a reason, not a parameter.

- **EscapeTimer** — the renamed `RenameTimer` interface (`C`, `Reset`,
  `StopAndDrain`) plus `realEscapeTimer`. Same shape; renamed because it now
  serves both owners and the old name would lie. Exported name changes; grep
  shows no user outside `termcmd`.

### Timer ownership (ARCH-ORDER)

`pumpStdinWithTimer` carries state across reads: `mode ∈ {plain, rename}` and
one pending buffer (`held` in plain, `rename.decoder.Pending` in rename). The
two are mutually exclusive by construction: `held` is drained into `data`
before any read is processed, and a read that begins a rename `continue`s
without re-filling it. The legal states and the timer's arm rule:

| mode | pending | timer | on expiry |
|---|---|---|---|
| plain | empty | stopped | — |
| plain | non-empty (chord/mouse prefix) | armed | `writeActive(held)`, clear |
| rename | anything but a lone ESC | stopped | — |
| rename | lone ESC | armed | `applyRename(nil, flushEscape=true)` → cancel |

Rule: **the tail of each read handler arms or stops the timer according to
its own pending buffer**, and never touches the other owner's. `applyRename`
already does this for rename (its last four lines). The plain handler gets the
same tail. The expiry arm branches on `rename != nil` to pick the owner.

Events the caller cannot block: EOF (`result.err != nil`) — already flushes
`held` to the child; unchanged. Timer expiry racing a read: the `select` picks
one; if the read wins, `held` is drained into `data` and the tail re-arms or
stops (draining a fired tick), so a stale tick can never flush bytes twice.
The ordering most likely mishandled is *rename begins from a held ESC* (`ESC`
read, then `r`): `applyRename` must own the timer from that read onward, so
the plain tail runs only when `rename == nil` at the end of the read.

**Latency budget (ARCH-CONSTRAINTS).** The fix costs a bare ESC up to 35 ms
before it reaches the child. A child that runs its own escape timeout adds it
on top — nvim's default `ttimeoutlen` is 50 ms — so the worst case from
keypress to mode change is about 85 ms. Basis for accepting that: it is the
deadline couch's two framers already impose on the same keystroke, and 85 ms
is under the ~100 ms threshold at which a keystroke reads as laggy.

Plain arms for **any** non-empty `held`, not only a lone ESC. Couch arms only
for the lone ESC; the difference is deliberate: `pair term`'s stdin is a local
zellij pty, where a multi-byte sequence torn across reads is re-joined within
microseconds, and a longer prefix left pending without a deadline (e.g. a
legacy-terminal Alt+`[`) is the same stuck keyboard this issue fixes.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pumpStdinWithTimer` | `cmd/internal/termcmd/run.go` | modified | stdin reads + `EscapeTimer` + `ptyWriter` |

- **pumpStdinWithTimer** — the stdin arbitration loop.
  - **Injected into:** tests via `EscapeTimer` (fake `firingEscapeTimer`,
    already exists as `firingRenameTimer`) and `ptyWriter` (`fakeMux`). The
    fake timer is the ordering seam: `autoFire=false` = "tail arrived before
    the deadline"; `autoFire=true` = "deadline passed first". Tests can force
    either interleaving.
  - **Future extensions:** none.

---

## Chunk 1: the deadline

### Task 1: Reproduce the misroute at the pump (red)

**Files:**
- Create: `cmd/internal/termcmd/escape_deadline_test.go`
- Modify: `cmd/internal/termcmd/run_test.go` (add a `wrote` channel to `fakeMux`)

- [ ] **Step 1: Give `fakeMux` an observation hook**

In `run_test.go`, add `wrote chan string` to `fakeMux` and signal it from
`writeActive` when non-nil (same pattern as `renameFinished`):

```go
func (f *fakeMux) writeActive(data []byte) {
	f.ops = append(f.ops, "write:"+string(data))
	if f.wrote != nil {
		f.wrote <- string(data)
	}
}
```

- [ ] **Step 2: Write the two explicit regressions**

```go
package termcmd

import (
	"io"
	"strings"
	"testing"
	"time"
)

// A lone ESC is the first byte of every legacy Alt chord, so the pump holds
// it. Without a deadline it was held until the NEXT keystroke, which then
// decided what the ESC became: ESC,ESC reached nvim as two bytes at once
// (insert mode left on the second press), and ESC,j became ChordAltJ (#234).
func TestBareEscapeIsForwardedAfterTheDeadline(t *testing.T) {
	rt := &fakeRuntime{}
	mux := &fakeMux{}
	timer := newFiringEscapeTimer()

	pumpStdinWithTimer(&splitReader{chunks: [][]byte{{0x1b}}}, mux, rt, io.Discard, timer)

	if got := strings.Join(mux.ops, ","); got != "write:\x1b" {
		t.Fatalf("ops = %q, want the ESC forwarded on the deadline, before EOF", got)
	}
}

func TestEscapeThenJAfterTheDeadlineIsTwoKeysNotAltJ(t *testing.T) {
	rt := &fakeRuntime{}
	wrote := make(chan string, 2)
	mux := &fakeMux{wrote: wrote}
	release := make(chan struct{})
	reader := &gatedChunksReader{chunks: [][]byte{{0x1b}, {'j'}}, release: release}
	timer := newFiringEscapeTimer()
	done := make(chan struct{})

	go func() {
		pumpStdinWithTimer(reader, mux, rt, io.Discard, timer)
		close(done)
	}()

	select {
	case got := <-wrote:
		if got != "\x1b" {
			t.Fatalf("first write = %q, want the bare ESC", got)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not forward the held ESC")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pump did not finish")
	}
	if got := strings.Join(mux.ops, ","); got != "write:\x1b,write:j" {
		t.Fatalf("ops = %q, want ESC then j as two keys, not ChordAltJ", got)
	}
}
```

Note `gatedChunksReader` gates the *last* chunk until `release` closes, so the
`j` cannot race the deadline.

- [ ] **Step 3: Run to verify both fail on today's code**

Run: `go test ./cmd/internal/termcmd/ -run 'TestBareEscape|TestEscapeThenJ' -v`
Expected: first fails with `ops = "write:\x1b"` … actually with the ESC only
written at EOF — assert message shows the deadline never fired (`timer.resets
== 0`; add `if timer.resets == 0` check if the ops alone pass). Second fails:
the ESC is never written, the test times out on `wrote`, or ops are empty
(`\x1bj` swallowed as Alt+j).

Adjust the first test to also assert `timer.resets >= 1` so it is red today
(today the ESC is written at EOF with no reset).

- [ ] **Step 4: Log the reproduction in the issue** — tick Plan step 1 with
  "reproduced at the pump: ESC,j → no write (swallowed as Alt+j)". A live
  reproduction in the pane is the manual step at the end.

### Task 2: One constant, one owner

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go` (add const + `time` import)
- Modify: `cmd/internal/couchtty/keys.go` (delete const)
- Modify: `cmd/internal/couchtty/console.go:600-614` (use the exported one; add import)
- Modify: `cmd/internal/termcmd/run.go:454` (rename arm uses the constant)

- [ ] **Step 1: Add to `shortcut.go`** next to `chordSequences`:

```go
// EscapeAmbiguity is how long an ESC-led prefix may stay pending before a
// framer resolves it as typed bytes. Every legacy chord in chordSequences
// begins with ESC, so a lone ESC on any Pair-owned stdin is ambiguous
// between "the user pressed Escape" and "the first byte of a chord whose
// tail is in the next read". The chord table is what creates the ambiguity,
// so it owns the one deadline; couch's two framers, termcmd's rename decoder
// and termcmd's main loop all read it (#234).
const EscapeAmbiguity = 35 * time.Millisecond
```

- [ ] **Step 2: Delete `escapeAmbiguity` from `couchtty/keys.go`** (lines
  47-49) and replace the four uses in `console.go` with
  `workbenchshortcut.EscapeAmbiguity`; add the import.

- [ ] **Step 3: Replace `50 * time.Millisecond` in `run.go`'s `applyRename`**
  with `workbenchshortcut.EscapeAmbiguity`.

- [ ] **Step 4: Build + run couchtty and termcmd tests**

Run: `go build ./... && go test ./cmd/internal/couchtty/ ./cmd/internal/termcmd/ ./cmd/internal/workbenchshortcut/`
Expected: build ok; the two new tests still red; everything else green.

### Task 3: Arm the deadline in the main loop (green)

**Files:**
- Modify: `cmd/internal/termcmd/run.go:354-390` (rename the timer types), `:461-464` (expiry arm), `:540-549` (plain tail)
- Modify: `cmd/internal/termcmd/run_test.go` (rename `firingRenameTimer` → `firingEscapeTimer`, `newFiringRenameTimer` → `newFiringEscapeTimer`)

- [ ] **Step 1: Rename** `RenameTimer` → `EscapeTimer`, `realRenameTimer` →
  `realEscapeTimer`, `newRealRenameTimer` → `newRealEscapeTimer`, and the test
  fake, with a doc comment on the interface:

```go
// EscapeTimer is the stdin pump's one escape-ambiguity deadline. Whoever owns
// the pending bytes owns the timer: a rename session arms it for a lone
// pending ESC (expiry cancels the rename); the plain path arms it for any
// held chord/mouse prefix (expiry forwards the prefix to the child). The two
// owners are exclusive — held is drained before a read is processed and a
// read that begins a rename never refills it — so one timer serves both (#234).
type EscapeTimer interface {
```

- [ ] **Step 2: Expiry arm picks the owner**

```go
case <-timer.C():
	if rename != nil {
		applyRename(nil, true, false)
	} else if len(held) > 0 {
		mux.writeActive(held)
		held = nil
	}
```

- [ ] **Step 3: Plain tail arms or stops**

After the inner `for len(data) > 0 { … }` loop, still inside the
`len(result.data) > 0` block and after the rename-mode `continue`:

```go
// Tail of the plain handler: arm for whatever is still pending, stop
// otherwise (draining a tick that fired while this read was processed).
// Skipped when this read began a rename — applyRename owns the timer then.
if rename == nil {
	if len(held) > 0 {
		timer.Reset(workbenchshortcut.EscapeAmbiguity)
	} else {
		timer.StopAndDrain()
	}
}
```

- [ ] **Step 4: Run the two regressions**

Run: `go test ./cmd/internal/termcmd/ -run 'TestBareEscape|TestEscapeThenJ' -v`
Expected: PASS.

- [ ] **Step 5: Make the existing split-chord test deterministic**

`TestPumpStdinDecodesSplitAltChord` uses the real timer; with a deadline it
depends on the scheduler beating 35 ms. Switch it to
`pumpStdinWithTimer(..., timer)` with `timer := newFiringEscapeTimer();
timer.autoFire = false` and assert `timer.resets == 1 && timer.stops >= 1`
(armed on the held ESC, stopped when `t` completed the chord).

- [ ] **Step 6: Run the package**

Run: `go test ./cmd/internal/termcmd/`
Expected: PASS, including every rename-timer test (they exercise the rename
owner and must be untouched by the plain tail).

- [ ] **Step 7: Commit**

```
#234: pair term: bare ESC is forwarded after one escape-ambiguity deadline
```

### Task 4: Tests generated from the chord table

**Files:**
- Modify: `cmd/internal/termcmd/escape_deadline_test.go`

Per `workshop/lessons.md` (escape decoders: add split-boundary tests where the
final byte arrives in a later read). For every sequence in
`workbenchshortcut.ChordSequences()` and every split point `1..len-1`:

- [ ] **Step 1: Write the generated test**

```go
// Every chord, every split point, both sides of the deadline. Generated from
// the chord table so a chord added later is covered without editing this.
func TestEveryChordSplitAtEveryByteResolvesAgainstTheDeadline(t *testing.T) {
	for _, seq := range workbenchshortcut.ChordSequences() {
		for cut := 1; cut < len(seq); cut++ {
			head, tail := []byte(seq[:cut]), []byte(seq[cut:])
			name := fmt.Sprintf("%q|%q", head, tail)

			t.Run("tail before deadline fires the chord/"+name, func(t *testing.T) {
				mux := &fakeMux{activeName: "work"}
				timer := newFiringEscapeTimer()
				timer.autoFire = false
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, tail}}, mux, &fakeRuntime{}, io.Discard, timer)
				for _, op := range mux.ops {
					if strings.HasPrefix(op, "write:") {
						t.Fatalf("ops = %v: chord bytes reached the child", mux.ops)
					}
				}
				if timer.resets == 0 {
					t.Fatal("held prefix never armed the deadline")
				}
			})

			t.Run("deadline first forwards the prefix as typed/"+name, func(t *testing.T) {
				mux := &fakeMux{}
				timer := newFiringEscapeTimer()
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head}}, mux, &fakeRuntime{}, io.Discard, timer)
				if got := strings.Join(mux.ops, ","); got != "write:"+string(head) {
					t.Fatalf("ops = %q, want the prefix forwarded once, on the deadline", got)
				}
			})

			t.Run("a non-chord byte resolves the prefix as typed/"+name, func(t *testing.T) {
				mux := &fakeMux{}
				timer := newFiringEscapeTimer()
				timer.autoFire = false
				pumpStdinWithTimer(&splitReader{chunks: [][]byte{head, {'q'}}}, mux, &fakeRuntime{}, io.Discard, timer)
				if got := strings.Join(mux.ops, ","); got != "write:"+string(head)+"q" {
					t.Fatalf("ops = %q, want prefix+q forwarded, no chord", got)
				}
			})
		}
	}
}
```

`q` is not the tail of any chord and does not extend any prefix; assert that
in the test with `workbenchshortcut.IsChordPrefix(append(head, 'q'))` being
false and `DecodeChord` failing, so a future chord on `q` fails loudly instead
of silently changing the test's meaning.

"Chord fires" is observed as *the child never sees the bytes*. The guarantee
is structural, not a property of `Decide`'s dispositions: the pump consumes a
recognised chord via `chordRest` before any `writeActive`, and `handleChord`
(`run.go:120`) is never handed the mux, so no disposition — `Pass` included —
can produce a `write:` op. A write of chord bytes is therefore exactly the
misroute. Errors reported through
`reportError` (the fake runtime has no panes) are expected and not asserted.
`ChordAltR` begins a rename, which is also not a write.

Case (b) "deadline first" of the first split (`head == ESC`) is the bare-ESC
regression restated 44 times; it stays because the split point is the axis
being generated, not the chord.

- [ ] **Step 2: Run**

Run: `go test ./cmd/internal/termcmd/ -run TestEveryChordSplit`
Expected: PASS. If a case fails on the rename chord's split, check the
`rename == nil` guard on the plain tail.

- [ ] **Step 3: Commit**

```
#234: pair term: split-boundary tests generated from the chord table
```

### Task 5: Atlas + issue

**Files:**
- Modify: `atlas/architecture.md` (the "`pair term` stream hygiene (#127)" *Input* bullet, ~line 442)
- Modify: `workshop/issues/000234-….md` (Plan ticks, Log)

- [ ] **Step 1: Atlas** — append to the *Input* bullet:

  "A chord or mouse *prefix* (every legacy Alt chord begins with ESC) is held
  for one `workbenchshortcut.EscapeAmbiguity` (35 ms) and then forwarded as
  typed — the same deadline couch's two framers and the rename decoder use.
  Before #234 the main path had no deadline: a bare ESC sat in `held` until the
  next keystroke, so nvim needed two presses to leave insert mode, and
  `ESC`,`j` arrived as Alt+j. The residual — `ESC`,`j` typed inside 35 ms
  still decodes as a chord — is what #227's alt-screen passthrough closes."

- [ ] **Step 2: Full suite**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG make test`
Expected: PASS (parley's `parley_harness_golden` 7/7 is a known pre-existing
failure, per memory).

- [ ] **Step 3: Manual in the right pane** (operator or live session): nvim,
  insert, one ESC → normal mode; `i`, ESC, `j` → normal mode then cursor
  down; Alt+j (held as a chord) → focus does NOT move (swallowed, as before);
  Alt+t → new tab. Record in Log.

- [ ] **Step 4: Tick the issue Plan, write the Log, commit**

```
#234: atlas + log: escape-ambiguity deadline on the pair term main loop
```

Then `sdlc close --issue 234 --verified '<the test names + the manual result>'`.

## Revisions

- **2026-09-12, after close round 1 (FIX-THEN-SHIP).** Delta from the approved
  plan: (1) `probes/escsmoke` added — a live oracle the plan did not have: the
  real `pair term` under a pty with a real nvim child, mode read over nvim's
  RPC socket; the operator's in-pane check stays in Task 5 but is no longer
  the only pane-level evidence. (2) Task 4 case (b) now runs through a
  `forwardedOnTheDeadline` helper that gates EOF behind the observed write,
  because the EOF flush produces the same bytes as the expiry flush and a
  deleted expiry branch stayed green. (3) `TestATornMousePrefixMeetsTheSameDeadline`
  pins the "any held prefix, not only a lone ESC" decision. (4) The latency
  budget was corrected: the 35 ms deadline and nvim's 50 ms `ttimeoutlen` add
  (~85 ms worst case); they do not nest. (5) `/escsmoke` added to `.gitignore`
  for the repo-root binary guard. Reason: close-review findings BR-3..BR-7.
