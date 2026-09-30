---
id: 000211
status: done
created: 2026-09-07
updated: 2026-09-29
estimate_hours:
github_issue:
started: 2026-09-29T20:06:42-07:00
actual_hours: 1.10
tracker:
    version: 1
    completion:
        token: close-3eb6560c8542
        repository: github.com/xianxu/pair
        reviewed_head: 15ab0c79cc6ef1e5abca46b5d77963f6a1fbb2d1
        evidence_commit: 94b7517827bc683d962ad086b6541bac44d86119
        landed_commit: 05425f74c0f161d5d3fcd2bdb3746612685491fc
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
