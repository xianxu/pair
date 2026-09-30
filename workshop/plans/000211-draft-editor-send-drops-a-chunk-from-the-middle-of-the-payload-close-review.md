# Boundary Review — pair#211 (whole-issue close)

| field | value |
|-------|-------|
| issue | 211 — Draft-editor send drops a chunk from the middle of the payload |
| repo | pair |
| issue file | workshop/issues/000211-draft-editor-send-drops-a-chunk-from-the-middle-of-the-payload.md |
| boundary | whole-issue close |
| milestone | — |
| window | 76db90172b0d80ffc14875b9ee5a3fcaef2825f7..10ee5c74f05368918644f7c217274ade780c412a |
| command | sdlc close --issue 211 |
| reviewer | claude |
| timestamp | 2026-09-29T21:40:09-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The change is small and well aimed. `nvim/draft_send.lua` gains a pure `frame`/`unframe` pair, and the draft send and the review poke now both write the body as one bracketed paste. The three stateful fakes refuse any write that isn't framed, so dropping the framing or the marker strip turns the tests red. The two probes back up the claim that pair's own path (zellij `write-chars` plus `pair wrap`) delivers byte-exact, and `send-audit.py` gives a way to verify the fix against real sends afterwards, which #354 owns. The Done-when rewrite is honest: it separates what this issue proves from what only real-use evidence can prove. Nothing blocks shipping. The one real gap is that sending paste markers unconditionally adds a new requirement for any agent pair drives (it must have bracketed paste enabled), and neither the atlas nor the harness bring-up guide says so.

1. **Strengths**
   - `nvim/draft_send.lua:16-27`: `frame`/`unframe` are pure and tested without IO. Stripping embedded `ESC[200~`/`ESC[201~` stops text from closing the paste early, and other escapes are deliberately left alone, which a test pins.
   - ARCH-DRY: `pair_poke.lua:19` reuses `draft_send.frame` instead of keeping a second copy of the rule. All three fakes (`draft_send_test`, `submission_integration_test`, `bang_tag_integration_test`) share one `unframe`.
   - The trace now redacts and hashes the bytes actually written (`redact[4] = wire`), and a test pins that. `review-poke-test` updates the `body_len` assertion (29 + 12 marker bytes) and still checks the body doesn't leak into the trace.
   - Both probes are test-only and clean up after themselves. `claudedraftsend` removes only its own `~/.claude/projects/*<tmpbase>*` directory and scrubs the `PAIR_*`/`CLAUDE*`/`ZELLIJ*` environment, so it can't write into the calling session (ARCH-SECURE, ARCH-FUNERAL pass).
   - Both probe comparators have unit tests for the false-positive the first run hit (Claude splitting a line at a paste boundary).

2. **Critical findings:** none.

3. **Important findings**
   - `atlas/architecture.md:1255` still says the plumbing works for "any TUI agent that accepts typed input". It now also needs the agent to have bracketed paste (DECSET 2004) enabled. An agent without it receives raw `ESC[200~`/`ESC[201~` bytes around every send. All five advertised harnesses are profiled, and the orientation prompt already sends them a bracketed paste, so none of them regresses today. The gap is in documentation and the bring-up contract: `atlas/how-to-bring-up-a-new-harness-cli.md` never mentions paste. Fix sketch: state the requirement in both places. Optionally, have pair-wrap watch the child's `?2004h`/`?2004l` output and strip the markers when the mode is off.

4. **Minor findings**
   - Proof that the change works for a busy Claude rests on the #354 post-ship audit, because the live on-demand repro was only on an idle Claude. The issue discloses this and hands it off properly.
   - The paste markers are now defined in both `nvim/draft_send.lua` and Go (`workbenchshortcut/framing.go`, plus a literal in `wrapcmd/orientation.go:146`). Keeping one copy per language is acceptable, but `orientation.go:146` could use `workbenchshortcut.PasteStart`/`PasteEnd`, since the two already live in the same Go codebase.
   - `send-audit.py` reimplements the `===` comment strip from `nvim/normalization.lua`. It's a standalone audit script, so the copy is tolerable, but it will drift if the strip rule changes; worth a comment pointing back to the Lua source (the docstring partly does this).
   - `send-audit.py`'s `reads = (nbytes+1023)//1024` counts bytes before the comment strip, so a send can land in the wrong read-count bucket. It's only a heuristic.

5. **Test coverage notes**
   - The framing is pinned three ways: exact wire bytes, the marker strip, and fakes that refuse unframed writes.
   - A sender-side 70-line (more than 2 KiB) case checks the framed send is byte-exact end to end through the fake. Whether a real agent reassembles the paste is correctly left to the probe; the fake doesn't claim it.
   - The reported pre-existing failures (`review-window-test` inside the full `make test`, `artifactpath`) are disclosed as present on origin/main too. I did not re-run them.

6. **Architectural notes (each principle)**
   - ARCH-DRY: pass. `frame` is shared, with the minor cross-language constant noted above.
   - ARCH-PURE: pass. `frame`/`unframe` are pure, and IO stays in the executor.
   - ARCH-PURPOSE: pass. Both body writers that send text to the agent (draft send and review poke) are framed. The other `write-chars` sites send `:lua` commands into nvim, not agent bodies, so they rightly stay unframed.
   - ARCH-MOCK: pass. Stateful fakes model a paste-aware composer, and the live probes act as conformance checks against real zellij and Claude.
   - ARCH-CONSTRAINTS: pass. The fix adds 12 bytes per send, and the existing ~100 ms settle is unchanged.
   - ARCH-SECURE: pass. Markers inside the body are stripped, the trace redacts the body, and the probe can't reach real session state. The audit script only reads user transcripts, locally.
   - ARCH-ORDER: pass. No new state is carried between events. Paste state in pair-wrap already exists (`submittingReturn`), and inside a paste a CR is treated as content, not a submit.
   - ARCH-FUNERAL: pass. Probe temp directories and Claude project directories are removed, and the probe binaries are gitignored.

7. **Plan revision recommendations:** none required. The Revisions section already records the root-cause move and the restated Done-when.

```findings
findings:
  - id: new
    severity: Important
    family: docs-track-new-contract
    title: |
      Agent-agnostic contract now requires bracketed paste (DECSET 2004); atlas and harness bring-up guide do not say so
    detail: |
      atlas/architecture.md:1255 still claims any TUI agent that accepts typed input works; a harness without 2004 would get raw ESC[200~/ESC[201~ around every draft send and poke. State the requirement in the bring-up guide (optionally strip the markers in pair-wrap when the child has not enabled 2004).
  - id: new
    severity: Minor
    family: single-source-constant
    title: |
      orientation.go:146 hardcodes paste markers instead of workbenchshortcut.PasteStart/PasteEnd
  - id: new
    severity: Minor
    family: single-source-constant
    title: |
      send-audit.py restates the === comment strip rule from nvim/normalization.lua; drift risk
```

---

## Re-review — 2026-09-29T21:55:37-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 211 — Draft-editor send drops a chunk from the middle of the payload |
| repo | pair |
| issue file | workshop/issues/000211-draft-editor-send-drops-a-chunk-from-the-middle-of-the-payload.md |
| boundary | whole-issue close |
| milestone | — |
| window | 76db90172b0d80ffc14875b9ee5a3fcaef2825f7..15ab0c79cc6ef1e5abca46b5d77963f6a1fbb2d1 |
| command | sdlc close --issue 211 |
| reviewer | claude |
| timestamp | 2026-09-29T21:55:37-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This round fixes the Important finding (BR-1) at its root, and BR-2 is fixed too. BR-3 is still open, but it is Minor and doesn't block the close. The core change frames the draft body and review pokes as one bracketed paste through a single `frame` in `nvim/draft_send.lua`, and `pair_poke.lua` reuses it. pair-wrap now asks its terminal model whether the child has turned on bracketed paste (DECSET 2004). If it hasn't, the markers are dropped in both the profiled translator and the pass-through translator, so an agent without bracketed paste gets typed input as before. The targeted Go tests (`go test ./cmd/internal/wrapcmd -run 'Paste|TranslateChunk|StripPaste|Muse|Agy'`) and `nvim/draft_send_test.lua` both pass at HEAD.

1. **Strengths**
   - `childAcceptsPaste` (`cmd/internal/wrapcmd/wrap.go:1336`) reads mode 2004 from the terminal model pair-wrap already keeps, instead of adding a per-harness flag. If there's no model, it keeps the old behavior of passing markers through.
   - `TestProfiledTranslatorHonorsChildPasteMode` expects `"a\\\rb"` in the no-2004 case, and the verbatim path can't produce that output. So the test fails without the fix; it doesn't just restate the implementation. `TestPassThroughHonorsChildPasteMode` also covers a marker split across two reads.
   - All three stateful Lua fakes share one inverse, `unframe`, and reject an unframed write. The framing is enforced by the tests, not only written down.
   - The Log states the root cause as measured: probes show zellij and pair-wrap deliver byte-exact, and the audit found Claude dropping whole middle reads in 8 of 11 three-read sends. The Revisions entries record that the Done-when moved with it.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-3 is still open: `scripts/send-audit.py:94` restates the `===` strip regex from `nvim/normalization.lua:11`. The two agree today (`^\s*===` vs `^[ \t]*===`, which differ slightly on other whitespace), so this is a drift risk only.

5. **Test coverage**
   - Framing, the marker strip, the frame/unframe round trip, a body larger than three 1 KiB reads, poke argv, trace redaction length and the 2004 capability paths are all pinned.
   - Whether Claude reassembles the paste while busy is not tested here. That depends on the external agent and is correctly handed to #354's post-ship audit.
   - A startup race (a send arriving before the child writes `?2004h`) would fall back to typed input, which is safe.

6. **Architecture**
   - ARCH-DRY: pass. One `frame` serves both senders, and orientation now uses the `workbenchshortcut` constants. Lua and Go each strip markers, but in separate layers, which is acceptable. The only remaining restatement is BR-3.
   - ARCH-PURE: pass. `frame`, `unframe`, `stripPasteMarkers` and `submittingReturn` are pure. The capability check is a thin read of the model.
   - ARCH-PURPOSE: pass. It covers both send paths (draft and poke) and both translators.
   - ARCH-MOCK: pass. The zellij fakes are stateful, and the probes act as live conformance checks.
   - ARCH-CONSTRAINTS: pass. Probes cover 512 B to 180 KB on both zellij versions.
   - ARCH-SECURE: pass. Embedded markers are stripped so text can't close the paste early, and the trace redacts the bytes actually written.
   - ARCH-ORDER: pass. Paste state comes from the child's own DECSET/DECRST through the emulator. A child toggling 2004 in the middle of a paste is an edge case, not a shipped bug.
   - ARCH-FUNERAL: pass. The probe binaries are gitignored and nothing durable is added.

7. **Plan revisions:** none. The Plan and Revisions match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      childAcceptsPaste strips markers for non-2004 children in translateChunk and passThroughChunk; paste_capability_test.go expects typed output the verbatim path cannot produce; atlas architecture.md and the bring-up guide state the contract.
  - id: BR-2
    disposition: addressed
    note: |
      orientation.go:147 now uses workbenchshortcut.PasteStart/PasteEnd.
  - id: BR-3
    disposition: not-addressed
    note: |
      scripts/send-audit.py:94 still hand-restates the === strip regex from nvim/normalization.lua:11; Minor, non-blocking.
```
