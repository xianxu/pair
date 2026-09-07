---
id: 000211
status: open
deps: []
github_issue:
created: 2026-09-07
updated: 2026-09-07
estimate_hours:
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

- [ ] Reproduce deterministically with a synthetic payload of known content
      (e.g. numbered lines) so the loss boundary is readable directly rather
      than reconstructed
- [ ] Vary size across the ~1KB boundary to test whether 1,025 bytes is a real
      constant or incidental
- [ ] Instrument the send call site to record bytes offered vs bytes written
- [ ] Fix at the root; add a regression test that sends a payload spanning
      several chunk boundaries and asserts byte-exact arrival

## Done when

- A send of arbitrary size arrives byte-exact, with a test pinning it
- The #208 headline-plus-path workaround is re-evaluated: kept because it is
  independently good (it keeps the prompt small), not because it is load-bearing

## Log

- 2026-09-07 — filed from the #208 M2 smoke test. Evidence above is measured,
  not inferred; the reconstruction compared the arrived text against
  `perf-capture-latest.txt`, which was complete on disk.
