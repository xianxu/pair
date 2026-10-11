# pair#428 plan: only real keystrokes interrupt a peer delivery

## Evidence (kink 47, measured 10-10)

The ops:0 wrapper log (`wrap-events-1-ops-33.jsonl`) hashes every stdin read.
Matched against the parley.nvim:2 send (`4d5ff954`, queued 13:05:35.69) and
the ops transcript:

| time | bytes on the agent pane's stdin | meaning |
|---|---|---|
| 13:05:35.7 | (none) | the wrapper pastes; the pre-#427 build then waits |
| 13:05:40.853 | `ESC[I` | focus in (already ignored by peer delivery) |
| 13:05:46.656 | `ESC[13;3u` (sha `185fa23f768a`) | **kitty-encoded Alt+Enter**: the operator submits by hand |
| 13:05:47.364 | `ESC[O` | focus out |

The message reached Claude at 20:05:46.682Z, 26 ms after the Alt+Enter. On
10-10 there were **no mouse reports** on that pane: the ops Claude never
enables mouse tracking (only `?2004h ?1004h ?2031h ?2026h ?25h` in its
output), so zellij handles the wheel and scrolls its own scrollback. So kink
47 was the pre-#427 submit stall (since fixed by #427's fixed submit delay).
The "interrupt" was a real keystroke, and it rightly counted. Two candidate
classes are still live and get fixed here: unsolicited terminal reports
(`?2031h` asks for color-scheme notices `ESC[?997;Nn`) and mouse reports
from agents that do enable tracking. Regression test: the exact kink 47
sequence. Focus never interrupts; the Alt+Enter does.

## Design

Pair classifies its **own input stream** and its own view of the pane,
never the agent's meaning (pair#427).

1. **Keystroke classifier** (`orientation_replies.go`, a peer-only mode
   beside `operatorDataWithoutFocus`). Frame the input and report whether it
   holds a keystroke. Non-keystrokes, whether solicited or not: focus `CSI
   I/O`; mouse SGR `CSI < b;x;y M/m`, urxvt `CSI b;x;y M`, X10 `CSI M` plus
   3 raw bytes; OSC/DCS strings; `CSI ? … c`, `CSI > … c` (DA); `CSI … $y`
   (DECRPM); `CSI ? … u` (kitty flags); `CSI … n` (DSR, color scheme
   `?997;N`); `CSI … t` (window reports). Everything else is a keystroke:
   text, controls, kitty `CSI … u` keys, and bracketed paste. CPR `CSI r;c
   R` stays solicited-only, because modified F3 is `CSI 1;m R`. Solicited
   replies still decrement their query count. Orientation keeps its
   conservative policy unchanged.
2. **Lone ESC no longer stalls** (found in design). A trailing `ESC` sits in
   `replies.input` as an incomplete frame, and dispatch then waits for
   "incomplete terminal input" until the next byte. When the pending-flush
   timer fires, the held partial frame resolves as a keystroke.
3. **The composer must be visible.** Before the paste, the delivery asks
   zellij for the agent pane's viewport (`zellij action dump-screen -p
   terminal_$ZELLIJ_PANE_ID`, plain text). It aligns the viewport with the
   wrapper's own model (scrollback plus screen) by the smallest scroll shift
   `k` whose rows match, with whitespace removed so wide cells and padding
   can't misalign. The composer counts as visible iff the cursor row is
   still in view (`k ≤ H-1-cursorY`). No alignment means "not visible" and
   the delivery waits for the next poll. That covers scrolled away, and a
   race with output. The probe runs off the dispatch locks in a goroutine
   (a zellij subprocess must not hold `inputAdmission`). Its result is used
   only while fresh (≤500 ms), then rerun. With no zellij (`ZELLIJ_PANE_ID`
   unset) the view counts as visible. This works for any agent: the cursor
   row, not a composer reading. Mouse that keeps the cursor row in view
   doesn't block, and with item 1 its bytes don't interrupt either.
4. **A short mutual lock with the draft pane, all wrapper-side.**
   - **Robot waits for draft.** This already holds: an admitted operator
     paste makes the composer occupied and arms the 1 s settle. A test pins
     it.
   - **Draft waits for robot.** While a delivery is `Delivering` (paste
     through confirm), an operator chunk that *starts a bracketed paste* (a
     draft send's `write-chars`) starts a **hold** of at most
     `PeerDraftHold` = 1 s. The held chunk, and every chunk after it, is
     queued in order and not admitted, so it doesn't interrupt. The hold is
     released when the delivery is terminal or the cap passes. The chunks
     are then admitted and forwarded with their **original inter-arrival
     gaps**, so the draft's paste→settle→Alt+Enter timing survives.
   - Typed keystrokes are not held. They still interrupt, as before.
5. **Other panes never block.** That is structural: each zellij pane has
   its own PTY. The draft send's only side effect on this stdin is focus
   in/out (item 1) and its paste (item 4). A test drives focus in/out
   during a delivery, and it submits.

## ARCH notes

- ARCH-FUNERAL: this creates nothing durable. The viewport result and the
  held chunks live in memory and die with the delivery or the 1 s cap.
- ARCH-CONSTRAINTS: at most one zellij subprocess per poll, only while a
  delivery is queued and otherwise ready, at ≤1 Hz. The hold is bounded at
  1 s, plus a replay no longer than the original gaps.

## Tests (TDD, `cmd/internal/wrapcmd`)

- A table test of the classifier: each byte class (SGR/urxvt/X10 mouse,
  focus, DA/DECRPM/kitty-flags/DSR/color-scheme/OSC/DCS/window reports, split
  frames) gives no keystroke. Text, ctrl, kitty keys, legacy Alt+Enter, and
  paste give a keystroke. The same classes go through `admitInput` and
  `dispatchPeer`: a delivery in Delivering survives the non-keystrokes, and a
  keystroke interrupts it.
- Kink 47 regression: `ESC[I`, then `ESC[13;3u`, then `ESC[O` during
  Delivering → interrupted by the Alt+Enter only.
- A lone `ESC` plus the flush → resolved, and the delivery doesn't wait for
  incomplete input.
- Viewport: alignment units (k=0, a small k that keeps the cursor row,
  scrolled past the cursor, no match, wide chars). Dispatch with a fake
  probe: scrolled holds the paste, and scrolling back releases it.
- Draft lock: a paste chunk during Delivering is held. The delivery submits
  and confirms, then the held paste and Alt+Enter are forwarded in order
  with their gap. Both submit, unmixed. Past the cap, the release counts as
  operator input.

## Steps

- [ ] classifier + table tests + kink 47 regression
- [ ] lone-ESC flush resolution + test
- [ ] viewport probe + alignment + dispatch gate + tests
- [ ] draft-paste hold + ordered timed release + tests
- [ ] atlas/couch.md: what counts as operator input for delivery; the
      viewport and draft-hold rules
- [ ] full `make test`, live smoke via ops:0 (scroll up → held, scroll
      back → delivered)
