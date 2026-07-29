---
id: 000127
status: working
deps: []
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 1.40
started: 2026-07-28T16:20:34-07:00
---

# right terminal pane corrupts the input stream

## Problem

Two defects reported live in the layout-3 right terminal, both in `pair term`'s
stream handling (`cmd/internal/termcmd/run.go`). They present as one symptom
cluster — "the right pane stops responding and spews escape sequences" — but
have independent root causes.

**A. A mouse release kills the pane's keyboard.** An SGR (1006) mouse event is
`\x1b[<button;col;row` plus a terminator: `M` = press, `m` = RELEASE. Both
`parseSGRMousePressPrefix` and `isSGRMousePrefix` searched only for `'M'`, so a
release matched "sequence not finished yet" and was parked in `pumpStdin`'s
`held` buffer. `held` is prepended to the next read, which re-matched the same
way — so the release *and every keystroke typed after it* accumulated and never
reached the child. Reported as "pressing `a` doesn't do anything".

The child app is simultaneously left holding an unmatched button-press, so nvim
stays in an open mouse drag — visual mode. That is the reported "click to
reposition the cursor becomes a visual selection", and it also explains why the
few bytes that did land looked inert (in visual mode `a` is a pending
text-object, `aw`/`ap`/…).

**B. Tab switching replays capability queries; the replies land on the wrong
tab.** `redrawTab` replays the tab's stored raw output verbatim
(`m.stdout.Write(tab.buffer)`, up to 128 KiB). That buffer still contains the
app's *terminal queries* (DA1, DECRQM, Kitty-keyboard). Replaying re-asks the
host terminal; the host's replies arrive on `pair term`'s stdin and are handed
to `mux.writeActive(...)` — the **now-active** tab's shell — which tries to run
them as a command. Observed live as a shell line reading
`execute: 1e1e/1e1e/1e1e\[?62;4;52c[?2026;2$y[?2031;1$y[?0u[?62;4;52c`
(DA1 reply twice — two replays — plus the synchronized-output,
color-scheme-updates, and Kitty-flags reports).

## Spec

- **Mouse release is a complete event.** Recognize both SGR terminators
  (`M` press, `m` release) in the prefix parser and the partial-sequence test,
  carry the press/release distinction on the event, and forward releases to the
  child. A release is never a wheel tick (the wheel reports press-only), so it
  must not fall into the scroll branch.
- **A tab switch must not re-ask the terminal.** Filter capability queries out of
  the bytes `redrawTab` replays, so switching tabs cannot re-issue them. With no
  spurious query there is no spurious reply, and nothing lands on the wrong
  shell.
- **No pump-side "absorb" layer.** Dropping replies on input was specced as
  defense in depth and is now deliberately NOT built: a reply arriving while its
  own tab is active is *solicited and correct* — that is how nvim learns Kitty
  keyboard support, synchronized output, and DA1. An unconditional absorb would
  silently break capability negotiation, and making it safe would require
  outstanding-query state whose only job is to defend the replay path this issue
  already closes. Simplicity First: close the source, don't filter the symptom.
- Both changes are pure stream-decoding decisions, unit-testable against the
  existing `fakeMux` / `splitReader` harness — no PTY needed (`ARCH-PURE`).
- **No fake for the host terminal, deliberately (`ARCH-MOCK`).** The host is a
  stateful external dependency (we query, it replies later), but this design
  removes the round-trip from our code rather than modelling it: we simply stop
  re-asking. Nothing cross-call remains, so shape matchers plus recorded reply
  samples are the right fidelity. The table is an explicit **best-effort
  deny-list** of what nvim / zsh / fzf actually emit, and its failure mode is
  benign by construction: a query we miss simply degrades to today's behavior.
  It does not need to be exhaustive, and no live conformance check backs it —
  worth a comment on the table itself so a maintainer doesn't go looking.

## Done when

- A click-drag-release in the right terminal leaves the pane's keyboard live,
  and nvim ends the drag instead of staying in visual mode.
- Switching tabs **re-issues** no capability queries, so no replay-caused escape
  text lands on either shell. Residual, accepted and recorded: a query already
  in flight from tab A when the user switches to B still has its reply delivered
  to B — closing that needs the outstanding-query state this issue deliberately
  does not build. The class is narrowed, not closed.
- A live query from the active tab still reaches the host and its reply still
  reaches the querying app (capability negotiation is not regressed).
- Regression tests cover: a lone release; a keystroke after a release; a
  press+release+payload in one read; `stripTerminalQueries` over every table
  row; a derived negative case per matched final byte; and a `redrawTab` replay
  emitting no query bytes.
- `go test -race ./cmd/internal/termcmd/` green; live check in a real session.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional but uses the required method.

**Two deliberate departures from the table, stated rather than hidden.** (1) The
strip row's `impl=0.55` is above every Go-module range in v3.1 (`smaller-go-module`
scales to 0.08–0.20, `greenfield-go-module` to 0.12–0.32). It is kept there on
purpose: the four most recent pair closes all ran long (#123 ratio 0.29, #125
0.31, #124 0.71, #118 0.95), and the sharpest comparable is **#125 — the
immediately preceding issue in this same `termcmd` mouse/selection subsystem —
estimated 0.45 and actual 1.43**. #127 is strictly larger than #125, so pulling
these rows to the table ceiling would reproduce a known local bias. Labeling them
off-table is the honest fix. (2) Test hours are folded back into each module
row per v2 Step 2.5 (tests are inside the primitive), not carried as a separate
row — the earlier separate row was a partial double-count.

Design buffer is **+15%**, not +30%: the ×0.2 spec-quality discount was applied
across the module rows (design=0.05 is ×0.2 of the table midpoint), so per v2.1
Step 6 the buffer halves to avoid double-crediting the same front-loaded design.

Design hours are low because the diagnosis was complete before this block was
written — both root causes were read out of `pumpStdin`/`redrawTab` and confirmed
(the terminator bug by a failing test, the replay bug by decoding the reported
escape bytes as DA1/DECRPM/Kitty *replies*). What remains is implementation.
Design buffer is +30%: the plan lives in this issue, not a separate `workshop/plans/` doc.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.08 impl=0.04
item: smaller-go-module design=0.05 impl=0.15
item: smaller-go-module design=0.20 impl=0.55
item: milestone-review design=0.05 impl=0.12
item: atlas-docs design=0.05 impl=0.05
design-buffer: 0.15
total: 1.40
```

Item mapping: `issue-spec` = this file plus the follow-up issue filed at close;
the three `smaller-go-module` rows are the terminator fix (plus the
`sgrMouseSize` dedup), the `stripTerminalQueries` filter (table + CSI/OSC framing
+ the `redrawTab(buf)` signature change at three call sites), and the test suite
priced on its own row; `atlas-docs` = the `atlas/architecture.md` note on
`pair term` stream hygiene.

## Revisions

**2026-07-28 — estimate 1.26 → 1.69 → 1.38 → 1.05; seam moved; absorb dropped.**

1. *1.26* costed the replay work as two plain module edits.
2. *1.69* priced what that plan actually required: bounded tail-carry state,
   because the filter sat at `appendBuffer`, where a query can straddle a
   4096-byte PTY read.
3. *1.38* — the plan-quality gate showed the seam was the more expensive of two.
   `tab.buffer` has exactly two touch points; filtering at the **replay**
   (`redrawTab`) rather than the **append** makes the buffer contiguous at filter
   time, so straddling cannot arise. That deleted the tail carry, its byte bound,
   its overflow rule, and the `updateMouseMode` ordering constraint, and moved an
   O(chunk) scan off the per-PTY-chunk hot path onto the once-per-tab-switch
   path.
4. *1.05* — the gate then found the pump-side absorb was not merely extra but
   **wrong**: a reply arriving while its own tab is active is solicited, and
   dropping it would break capability negotiation (Kitty keyboard, synchronized
   output, DA1). Making it safe needs outstanding-query state that exists only to
   guard the path the strip already closes. The layer is removed, not deferred —
   with its item, its `escapeSequenceIncomplete` wiring, and its four ordering
   tests. Offsetting additions: OSC/DCS-shaped queries in the table, DSR/DA2/
   XTVERSION rows, the locked buffer snapshot, and a table-derived negative
   suite.

**2026-07-28 — estimate 1.05 → 1.47.** The 1.38→1.05 step credited dropping the
absorb layer but never priced the five additions that came with it (OSC rows,
DSR/DA2/XTVERSION, the locked snapshot, the derived-negative suite). The test
surface is now its own `smaller-go-module` row — roughly 15 cases: one strip case
per table row, derived negatives across six final-byte classes, two bisection
cases, the reply-survives case, two invariant pins, the residual pin, and a
`-race` case needing a mutex-wrapped test writer. Offsetting reductions in the
same pass: the DCS/XTGETTCAP row is dropped (framing mode gone), and the
cross-package `otherEscRe` extraction is scoped out to a follow-up issue rather
than rewriting three `wrapcmd` call sites here.

Also from the gates: `longestSuffixPrefix` was named as the carry primitive but
cannot do the job — its contract (rename_input.go:224) is "longest suffix that
prefixes *this literal*", which cannot express a variable-param shape. It is no
longer needed at all. And the `M1`/`M2` tags were dropped: a five-line terminator
fix landing on the same branch as the replay work does not warrant its own
fresh-context review boundary (AGENTS.md §3).

## Plan

Single review boundary — no `Mx` tags: one branch, one close.

- [x] Recognize the SGR release terminator (`m`) alongside press (`M`) in
      `parseSGRMousePressPrefix` and `isSGRMousePrefix`, carry press/release on
      the event, and forward releases to the child. One shared
      `sgrMouseTerminators` constant drives both sites so they cannot disagree
      with the protocol independently again (`ARCH-DRY` — the bug *was* that
      divergence).
- [ ] Point `sgrMouseSize` (rename_input.go:174-183) at `sgrMouseTerminators`.
      It is the **fourth** consumer of that constant (run.go:581, 596, 622 are
      the others) and the one still hardcoding `'M' || 'm'`. Keep the `i >= 3`
      start offset: scan `input[3:]`, return `idx+3+1`, so the rename path's
      existing tests keep passing (`ARCH-DRY`).

- [ ] **Reconcile with the escape-sequence machinery already in `wrapcmd`
      (`ARCH-DRY`).** This binary already has terminal-capability tables and a
      stripper, and the plan must not silently become a second one:
      - `codexKKPMarkers` (wrap.go:738) — includes **`\x1b[?u`**, the exact
        Kitty-flags query this table adds.
      - `codexSyncOutputMarkers` (wrap.go:731) — `\x1b[?2026h`/`l`, the
        synchronized-output mode the DECRQM row is about.
      - `stripCodexOutputMarkers` (wrap.go:766) with `p.stdoutPending`
        (wrap.go:310) — a byte-level marker stripper *with tail carry*,
        structurally the function being written here.
      - `otherEscRe` (wrap.go:189) — already encodes CSI **plus OSC terminated by
        BEL-or-ST**, i.e. the "second terminator rule" this plan called new work.
      **Decision: keep the framing helper inside `termcmd`; do NOT extract a
      shared package in this issue.** The repo is a flat `cmd/internal/<pkg>`
      layout with no `pkg/`, no root `internal/`, and no ansi package, so
      extraction would create a new package as a side effect of a two-defect
      bugfix. `otherEscRe` is also consumed three different ways — `ReplaceAll`
      in `stripTerminalControls` (wrap.go:812) and in the capture-early path
      (wrap.go:1151), and `FindIndex` **at an offset** in the colored-run walker
      (wrap.go:1018) — so a byte-scanner shaped for `stripTerminalQueries` does
      not drop into them; sharing would mean rewriting three call sites that feed
      scrollback capture and agent-output detection, pinned by
      `stdout_filter_test.go`, `extract_fg_test.go` and
      `update_agent_output_test.go`. That is a separable extension, not this
      issue's purpose, so deferring it is not an `ARCH-PURPOSE` gap.
      The duplication is instead **recorded** two ways: a follow-up issue filed
      at close for the cross-package extraction, and a `## Log` note that the two
      policy tables are deliberately different and in one case **opposed** —
      wrapcmd strips `\x1b[>7u` so codex stops pushing Kitty flags, while this
      issue requires `\x1b[>1u` to SURVIVE the replay filter. Two policy tables
      is intentional; two *framing* implementations is the debt being tracked.

- [ ] **Strip queries where the buffer is replayed, not where it is appended.**
      `tab.buffer` has exactly two touch points — the append (run.go:747) and the
      replay in `redrawTab` (run.go:1088). Filter at the replay:
      - The buffer is contiguous there, so a query straddling two 4096-byte PTY
        reads is already whole; no tail carry, byte bound, overflow rule, or
        `updateMouseMode` ordering constraint is needed (the append site is
        untouched and keeps reading the raw chunk).
      - It moves an O(128 KiB) scan off the hot path (every PTY chunk, under
        `m.mu`) onto the rare path (a tab switch).
      - `stripTerminalQueries(buf) []byte` is pure; `redrawTab` stays the thin IO
        shell (`ARCH-PURE`). The live path is untouched: `copyActiveOutput`
        writes `chunk.data` to stdout separately, so an app's first real query
        still reaches the terminal and gets its answer — only the replay is
        silenced.

- [ ] **Snapshot at the call sites, inside the lock they already hold.**
      `appendBuffer` re-slices `tab.buffer` under `m.mu` from the
      `copyActiveOutput` goroutine while `redrawTab` reads it lock-free; today
      that is one `Write` over a racy slice, and an O(128 KiB) scan widens the
      window enough for `-race` to flag it.
      The obvious fix — have `redrawTab` acquire `m.mu` — would introduce a new
      documented invariant ("no caller may hold `m.mu`") whose violation mode is
      a **deadlock**, strictly worse than today's race and enforced only by a
      comment. Avoid it: all three callers already hold the lock immediately
      before calling (`newTab` unlocks at :698 → calls at :700; `switchRelative`
      at :879 → :881; `removeTab` at :932 → :944). Snapshot one line earlier,
      inside the lock each site already holds, and change the signature to
      **`redrawTab(buf []byte)`**. Three one-line changes, no new lock-ordering
      invariant, and a `redrawTab` that knows nothing about the mutex — a
      genuinely thinner IO shell (`ARCH-PURE`).

- [ ] **Query table = a literal set + three narrow parameterized rules.**
      `tab.buffer` is fed from `readPTY` → PTY master (run.go:706, 747), so it
      is dominated by the **child app's output** — where queries are
      overwhelmingly FIXED byte strings (it is the *replies* whose params vary
      per terminal). Note replies **do** reach this buffer: `pumpStdin` writes a
      reply into the child's PTY, the shell's line discipline echoes it, and it
      returns through `readPTY` — that echo *is* the reported
      `execute: …\[?62;4;52c…` line. So the table must be shaped so that no reply
      form matches any row (verified: DECRPM replies terminate `$y` not `$p`; the
      Kitty reply `\x1b[?0u` is not the `\x1b[?u` literal; `\x1b[?62;4;52c` is
      neither `\x1b[c` nor `\x1b[0c`). A broad "params in `0-9;:$<>=?`" class is
      therefore both unnecessary and actively hazardous: it puts the private-parameter introducers `?`, `>`, `<`, `=` —
      the very bytes that discriminate rows — into the variable part, which is
      how a greedy `u` rule would eat the Kitty push `\x1b[>1u` and a greedy
      `h`/`l` rule would eat DECSET `\x1b[?1006h` and kill `updateMouseMode`
      (run.go:756). Specify instead:
      - **Literals:** DA1 `\x1b[c`, `\x1b[0c`; DA2 `\x1b[>c`; XTVERSION
        `\x1b[>q`; Kitty flags query `\x1b[?u`; **DSR `\x1b[6n`** (emitted by
        zsh/bash prompt machinery, fzf and nvim — at least as likely as the four
        observed, and its reply `\x1b[24;1R` types into the shell identically);
        OSC colour `\x1b]10;?`, `\x1b]11;?` (BEL or ST terminated).
      - **Two parameterized rules, each narrow:** DECRQM `\x1b[?<digits>$p`;
        OSC-4 `\x1b]4;<digits>;?`.
      - **No DCS row.** XTGETTCAP (`\x1bP+q<hex>ST`) would be the only row needing
        DCS framing, and `otherEscRe` does not actually frame DCS — its
        `\x1b[@-Z\\-_]` alternative matches `\x1bP` as a bare two-byte escape and
        never scans to ST. So DCS is new machinery either way, for a query the
        live report never produced. The Spec licenses a best-effort deny-list
        whose failure mode is benign, so the row is dropped until one is
        observed and the whole DCS mode disappears at zero cost
        (Simplicity First).
      This removes the class hazard by construction rather than testing around
      it — Simplicity First — and demotes the negatives below to a backstop.
      Framing covers **CSI and OSC only** and lives in `termcmd`, reusing
      `isTerminalFinalByte` (rename_input.go:220) for the CSI final rather than
      re-rolling it.

- [ ] **Rule for unterminated and bisected sequences.** Contiguous is not the
      same as whole: `appendBuffer` re-slices to the last 128 KiB
      (run.go:748-749), which bisects whatever sequence spans that boundary, so
      the buffer can BEGIN mid-sequence; and the newest chunk can END
      mid-sequence. The rule is explicit — **an unterminated escape at
      end-of-buffer is emitted verbatim; never drop-to-end.** The tempting
      "no final byte found → discard the rest" would silently swallow the tail of
      the replay, i.e. the visible screen. Tests: a buffer ending in `\x1b[?100`,
      and a buffer starting mid-sequence (`6;4;52c\x1b[?1006h…`), both asserting
      the surrounding bytes survive byte-for-byte.

- [ ] **Negatives derived from the table, not hand-listed.** For every final byte
      the table matches, assert one legitimate non-query sequence with that final
      SURVIVES:
      - `h`/`l` → DECSET `\x1b[?1006h`, `\x1b[?1002h`, `\x1b[?2004h` (shares the
        `\x1b[?` prefix with DECRQM; `updateMouseMode` parses these same bytes).
      - `u` → Kitty flags **push `\x1b[>1u` / pop `\x1b[<u`**. Stripping a push
        leaves the host encoding keys in legacy form for an app expecting Kitty —
        a silent rerun of defect A.
      - `q` → **DECSCUSR `\x1b[5 q`**, which nvim emits on every mode change and
        which shares the `q` final with XTVERSION; stripping it leaves the cursor
        shape wrong after a tab switch.
      - `c` → SGR/cursor sequences ending `c`; `n` → DSR *reports* other than 6n;
        OSC finals → a non-query OSC such as a title set `\x1b]0;title\x07`.
      - **Replies must survive**: a DA1 reply `\x1b[?62;4;52c` and a DECRPM report
        `\x1b[?2026;2$y` pass through untouched. This is the negative that
        corresponds to the actually-observed bytes, since shell echo puts them in
        the buffer.

- [ ] **Pin the invariant that justifies dropping the absorb layer.** Done-when
      #3 ("a live query still reaches the host, its reply still reaches the app")
      is currently only checked by a live session. Two cheap automated pins, both
      on the existing harness:
      (a) `copyActiveOutput` writes a query-bearing chunk to `m.stdout`
      **verbatim** (run.go:730-733) — proving the filter is not on the live path;
      (b) `pumpStdin` forwards a reply (`\x1b[?62;4;52c`, `\x1b[24;1R`) to
      `writeActive` unmolested.
      Without these, a later refactor that moves the filter back to
      `appendBuffer` breaks capability negotiation with every test still green.

- [ ] **Pin the accepted residual.** Done-when accepts that a query in flight
      from tab A when the user switches to B still delivers its reply to B. Add a
      test asserting exactly that, so "narrowed, not closed" is pinned behavior
      rather than a latent bug the next reader re-discovers and re-files; note it
      in `## Log` at close.

- [ ] Tests on the existing `fakeMux` / `splitReader` harness: the release cases
      (done); one strip case per table row; the derived negative suite; a
      `redrawTab` replay over a query-bearing buffer emitting no query bytes; and
      the two invariant pins above. The `-race` case wraps the test writer in a
      mutex — `stdoutWriter` (run_test.go:857) is a bare embedded `*bytes.Buffer`
      and `m.stdout` is written unsynchronized from both the output goroutine
      (run.go:732) and the pump goroutine (run.go:1087), so an unwrapped race
      test reports on the *double*, not on `tab.buffer`. In production `m.stdout`
      is an `*os.File` — that is interleaving, not a data race — so do NOT add
      stdout locking to production to silence it (different issue).

- [ ] `atlas/architecture.md`: note that `pair term` owns stream hygiene in two
      places — chord/mouse arbitration on input, query stripping on replay — that
      replies are deliberately left alone on the input path, and that the
      capability-sequence *policy* tables in `termcmd` and `wrapcmd` are
      intentionally separate over one shared framing helper.

## Log

### 2026-07-28

- Filed from a live report while doing v1.24 release prep. Both defects found by
  reading `pumpStdin`; A was confirmed with a failing test before the fix
  (`keystroke after mouse release is not swallowed` → mux ops came out as one
  merged `write:\x1b[<0;8;2ma` at EOF, proving the bytes were held, not sent).
- B's root cause was confirmed by decoding the reported shell line: the bytes are
  DA1 / DECRPM(2026) / DECRPM(2031) / Kitty-flags *replies*, i.e. answers to
  queries, not keystrokes — which points at a replay, not at user input.
- Note `rename_input_test.go` already exercises both `M` and `m` forms, so the
  release terminator was known elsewhere in this file and simply missed in the
  mouse path.

- Implemented. `stripTerminalQueries` + the deny-list live in a new
  `cmd/internal/termcmd/queries.go`; `redrawTab` took the `(buf []byte)`
  signature and all three callers snapshot under the lock they already held, so
  no new lock-ordering invariant was created.
- **Confirmed the judge's correction about replies reaching the buffer.** The
  table is shaped so no reply form matches a row, and
  `TestStripTerminalQueriesPreservesLegitimateSequences` pins DA1 replies,
  DECRPM reports, the Kitty `\x1b[?0u` reply and the `\x1b[24;1R` DSR report as
  survivors — the negative that corresponds to the actually-observed bytes.
- Accepted residual is pinned by `TestReplyGoesToActiveTabNotTheQueryingTab`: a
  query in flight when the user switches tabs still delivers its reply to the new
  tab. Narrowed, not closed; closing it needs outstanding-query state.
- Follow-up filed as **pair#128** for sharing the escape *framing* with
  `wrapcmd`. The *policy* tables stay separate and in one case opposed
  (`wrapcmd` strips `\x1b[>7u`; here `\x1b[>1u` must survive) — recorded in
  `atlas/architecture.md` so it reads as intentional, not drift.
- `side-quest:` two pre-existing test defects surfaced while verifying and were
  fixed here rather than left red:
  - `tests/scrollback-open-test.sh` asserted `'<M-x>'` appeared literally in
    `nvim/scrollback.lua`, which #117 invalidated when it moved global chord
    handling into the shared `workbench_route`/`workbench_actions` pair. `make
    test` had been red on main. The assertion now checks the wiring
    (`install_global_maps`) plus the generated action table, i.e. the behavior
    rather than the old implementation.
  - `TestTerminalMuxChildOutputDoesNotRestoreTitleDuringRename` raced under
    `-race` on its own bare `*bytes.Buffer` double, which the test polls while
    `copyActiveOutput` writes from another goroutine. Swapped to a mutex-wrapped
    writer. Deliberately NOT fixed by locking `m.stdout` in production — there it
    is an `*os.File`, so that is interleaving, not a data race.
- Verified: `go test -race ./cmd/internal/termcmd/` green; full `make test`
  exit 0 (the only `fail` strings in the log are test *names* about
  failure-handling).
