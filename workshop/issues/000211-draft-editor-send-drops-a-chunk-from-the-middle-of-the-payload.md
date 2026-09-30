---
id: 000211
status: working
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-29
estimate_hours:
card_mirror: '47c326aa91c6f47df207ae6537e3724cc7470d8b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-29T20:06:42-07:00
flow: {kind: full, provenance: inferred}
---

# Draft-editor send drops a chunk from the middle of the payload

## Problem

Text sent from the draft editor to the agent pane can arrive with a **contiguous
chunk missing from the middle**. The head and the tail arrive intact, so the
result reads as a plausible message rather than an obviously broken one — which
is the dangerous part: a payload can lose its most valuable half silently.

Observed 2026-09-07 while smoke-testing `:PairDoctor`'s perf capture (#208).

### Measured evidence

The sent payload was reconstructed from the on-disk sidecar and diffed against
what actually arrived:

```
payload sent:   2,447 bytes / 66 lines
cut at byte:      952
resumed at byte: 1,977
missing:        1,025 bytes  (one contiguous run)
head:           intact
tail:           intact
```

Corroborating state from the same capture, both **complete**:

- the sidecar file: 183,779 bytes, terminating correctly with `# end`
- the rolling JSONL row: written, with `hop_ms 0.007, exec_ms 1.506,
  action_ms 13.536, editor "fast"`

So the capture itself, the file write, and the log append all succeeded. **The
loss is entirely in the send path.**

### What this rules out

Three theories were live before the measurement. Two are now dead:

1. **A redraw wiping the head** — refuted. The head arrived intact; the hole is
   in the middle.
2. **A focus race dropping the beginning of the stream** — refuted, same reason.
3. **`zellij write-chars` losing a chunked write** — *consistent with* all of
   the above and currently the only surviving theory. A ~1KB loss boundary is
   suggestive of a buffer/chunk size, though nothing yet confirms the number is
   meaningful rather than incidental.

### What does NOT predict it

**Size does not.** A ~180 KB payload sent minutes earlier in the same session
arrived **complete**. A 2.4 KB payload then lost 1 KB. Any fix premised on
"large payloads are the problem" is aimed at the wrong thing.

## Spec

Find and fix the actual loss. Until then #208 works around it rather than
depending on it: the perf capture now writes everything to a sidecar file and
sends a headline plus a path, with **the path deliberately placed in the first
~60 bytes** so it survives a truncated send.

That workaround is not a fix and does not cover the general case — every other
`alt+return` send from the draft editor is still exposed, including ordinary
prose the operator types, where a missing middle would be far harder to notice
than in a structured report.

Suspects to work through, cheapest first:

- the `write-chars` call site: whether it chunks, and whether a partial write's
  return value is checked or discarded
- whether the loss is at the zellij IPC boundary (socket write) or in the
  receiving pty
- whether a bracketed-paste or mouse-mode sequence is interleaving mid-write
  (see #207 for the mouse-mode release gap, a neighbouring symptom)

## Plan

- [x] Reproduce deterministically with a synthetic payload of known content
      (numbered lines) — `probes/zellijwritechars` (zellij + pair wrap, no
      agent) and `probes/claudedraftsend` (real Claude, opt-in)
- [x] Vary size across the ~1KB boundary — 512 B to 180 KB, fast and
      queue-saturating slow readers, zellij 0.44.3 and 0.45.1: all byte-exact.
      The 1,025 is one macOS tty read (~1 KiB), lost by the agent, not by pair
- [x] Instrument the send call site — not needed: pair-wrap already traces every
      stdin read/write (len + sha); the audit used pair's send log instead
- [x] Fix at the root: the draft body (and review pokes) go as ONE bracketed
      paste (`nvim/draft_send.lua` `frame`); tests pin the framing, the marker
      strip, and the stateful fakes now refuse an unframed write
- [x] Post-ship verification (`scripts/send-audit.py --since <merge date>`
      after a week of use) handed to #354

## Done when

- Pair's own send path (zellij `write-chars`, `pair wrap`'s translator) is shown
  byte-exact across the ~1 KiB boundary up to 180 KB, by a re-runnable probe
- The draft body and review pokes reach the agent pane as ONE bracketed paste,
  with any paste markers in the text stripped, pinned by tests that fail when
  the framing or the strip is removed
- The agent-side effect is verifiable from real use: `scripts/send-audit.py`
  reports lossy sends per agent and read count, and #354 owns the post-ship run
- The #208 headline-plus-path workaround is re-evaluated: kept because it is
  independently good (it keeps the prompt small), not because it is load-bearing

## Log

- 2026-09-07 — filed from the #208 M2 smoke test. Evidence above is measured,
  not inferred; the reconstruction compared the arrived text against
  `perf-capture-latest.txt`, which was complete on disk.

### 2026-09-29
- 2026-09-29: closed — Pair path byte-exact: probes/zellijwritechars 512B-180KB, fast+saturated readers, direct and via pair wrap, zellij 0.44.3 + 0.45.1. Root cause via scripts/send-audit.py over 3,334 matched sends: Claude drops whole ~1KiB middle tty reads of unbracketed sends (3-read 8/11, 2-read 0/34, codex 0/1163). Fix: body framed as one bracketed paste (draft_send.frame, review pokes share it); BR-1: pair-wrap drops the markers for a child without DECSET 2004 (childAcceptsPaste, both translators) so agent-agnostic delivery is unchanged. Tests mutation-checked (framing, marker strip, 3 guard points). Live probe real Claude 2.1.285 on rebuilt binary: 6/6 whole (drop not reproducible on idle Claude; post-ship audit is #354). make test + go test ./...: green except review-window-test (7, passes standalone) and artifactpath classification (32), both failing identically on clean origin/main.; review verdict: SHIP
- 2026-09-29: flow upgraded quick → full — 860 added lines in code files (limit 100); an earlier round of this close already ran the full review

- **Root cause located: the receiving agent, not pair.** `probes/zellijwritechars`
  sends numbered-line payloads through `zellij action write-chars` into a
  raw-mode recorder, directly and through `pair wrap` under the claude profile
  (translator verified active: 420 translated chunks in its own trace). 512 B to
  180 KB, fast reader and a 64 B/20 ms reader that holds the tty queue full, on
  zellij 0.45.1 AND the incident's 0.44.3 (official release binary): every send
  byte-exact.
- **Audit of real history** (`scripts/send-audit.py`: pair's send log vs agent
  transcripts, 3,334 matched sends). Claude: 1-read 0/2123 lossy, 2-read 0/34,
  **3-read 8/11**, 4+-read 1/3. Codex: 0/1163 (only 6 sends of 3+ reads, weak).
  Every hole is one whole ~1,020-byte tty read from the middle (offset ~1022, or
  ~4079 in a 5-read send), on every Claude Code version 2.1.258–2.1.285, most
  recently 2026-09-29 19:38. The first audit pass flagged ~38 more sub-1 KiB
  "holes": pair's own `===` comment strip — the script now mirrors
  `nvim/normalization.lua`.
- **Live reproduction is incomplete.** `probes/claudedraftsend` (fresh idle
  Claude, haiku) delivered 12/12 plain and bracketed sends whole. Its first
  run reported one lost line per plain send — Claude's paste-boundary blank
  line split a numbered line; the comparison now ignores that (unit-tested).
  The drop presumably needs a busy Claude (long transcript, slow render); not
  reproduced on demand.
- **Fix: one bracketed paste per send.** Claude no longer infers a paste from
  read timing; the markers bound it however the bytes are chunked. Same
  convention the orientation prompt already uses for every profiled harness
  (`wrapcmd/orientation.go`) — ARCH-DRY. `frame` strips `ESC[200~`/`ESC[201~`
  from the body so text cannot close the paste early; review pokes
  (`pair_poke.lua`) reuse `frame`; `unframe` is the one inverse the three
  stateful fakes share, and they refuse an unframed write (mutation-checked:
  dropping the framing or the strip fails `draft_send_test`).
- **Done-when status.** Pair's path is byte-exact and pinned by probe + tests.
  That the bracketed send defeats Claude's drop is shown only on an idle Claude
  (it arrived whole; so did plain); the proof against the busy case is the
  post-ship audit in the Plan's last row.
- **#208 workaround re-evaluated:** keep. Headline + path keeps the prompt
  small and the capture on disk; with this fix it is no longer load-bearing.
- **Pre-existing, not this change** (identical on clean origin/main):
  full `make test` fails `review-window-test` (7, passes when the target runs
  alone) and `artifactpath` `TestProductionArtifactReferencesAreExactlyClassified`
  (32 identical complaints). Every other target and `go test ./...` pass.
- **Close review BR-1 (Important), fixed at the root:** framing would have made
  DECSET 2004 a requirement for every agent pair drives. pair-wrap now asks its
  terminal model (`childAcceptsPaste`): markers pass when the child enabled 2004
  (or there is no model), and are dropped otherwise, in both the profiled and
  the pass-through translator, so an agent without bracketed paste gets typed
  input exactly as before. Mutation-checked at all three points. Minors: the
  orientation prompt uses the shared Go paste constants; atlas and the harness
  bring-up guide state the contract. Live probe re-run on the rebuilt binary:
  6/6 whole.

## Revisions

- 2026-09-29 — root cause moved from "zellij write-chars chunk loss" (the only
  surviving theory at filing) to Claude Code dropping middle tty reads of an
  unbracketed burst; measured, see Log. Plan rows re-scoped accordingly; the
  post-ship audit, needed because an on-demand live repro was not achieved,
  is #354.
- 2026-09-29 — `## Done when` restated with the Spec's move: "a send of
  arbitrary size arrives byte-exact" is split into pair's path (proven here)
  and the agent's (framing pinned here, real-use proof in #354).
