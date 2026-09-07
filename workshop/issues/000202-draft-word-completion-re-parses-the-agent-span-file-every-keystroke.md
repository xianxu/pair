---
id: 000202
status: open
deps: []
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
---

# draft word completion re-parses the agent span file every keystroke

## Problem

The draft pane's word completion (`nvim/init.lua:1830`) rebuilds its entire
candidate pool from disk on **every** `TextChangedI` — that is, every keystroke
where the token is not path-shaped:

```lua
picks_load()                                     -- file read #1
local f = io.open(agent_output_path(), 'r')      -- file read #2
for l in f:lines() do
  local color, count, span = l:match('^([^\t]+)\t(%d+)\t(.+)$')   -- per line
  ...
end
for i, s in ipairs(spans) do
  s.score = (s.count + PICK_WEIGHT * (picks[s.span] or 0)) * 0.5^(rank/DECAY_HALFLIFE)
end
table.sort(spans, function(a, b) return a.score > b.score end)
```

There is **no mtime guard and no cache**. `$PAIR_AGENT_OUTPUT_PATH` changes only
when the agent emits output — never as a result of the operator typing — so on a
keystroke this work is ~100% redundant: two file reads, up to 1000 pattern
matches, a `pow()` per span, and a sort over the full set before it is trimmed
to `POOL_CAP`.

`wrapcmd` caps the file at 1000 spans (`agentOutputSpansMax`), so the cost is
bounded but reaches its bound and stays there: a fresh session's file is ~700
bytes, and every long-running session in this fleet is pegged at the cap (43–52
KB, 1000 lines). Long-lived sessions are the norm under couch, so this is
effectively always worst-case in practice.

### Measured — and smaller than it looks

Benchmarked against the real 1000-line file (`nvim --headless`, 50 iterations,
worst case with every span passing the colour filter):

```
per keystroke: 0.97 ms   (1000 spans parsed)
```

**Sizing this honestly: ~1 ms per keystroke is not perceptible, and this is NOT
the cause of the operator's reported typing lag.** It was filed after a session
that initially misattributed that lag here; the benchmark above is what
corrected it. The keystroke-latency question remains open — see Log.

So this is a papercut, not a defect with a visible symptom. It is worth fixing
on its own terms: it is pure redundant work on the editor's hot path, the fix is
small and local, and it scales with a file that is always at its cap.

## Spec

**Cache the parsed, scored pool; invalidate on the file's mtime.**

- Keep the pool and the mtime it was built from; rebuild only when the mtime
  advances. The agent writes the file atomically (`wrapcmd` builds an LRU and
  "writ[es] atomically"), so an mtime check is a sound invalidation signal.
- `picks_load()` gets the same treatment — it is a second read on the same path,
  changed only when the operator accepts a completion, which the module already
  knows about locally.
- Scoring depends on `picks`, so the cache key is (span-file mtime, picks
  generation); bumping the picks generation in-process on accept avoids a second
  stat.

Deliberately not in scope: changing the ranking, the colour allowlist, the cap,
or the completion's behaviour in any observable way. This is an
identical-output change — if a test can tell the difference, it is wrong.

## Done when

- The span file is read and parsed at most once per change to it, not once per
  keystroke; asserted by a test that types N characters and counts reads.
- Completion results are **identical** to today's for the same inputs — same
  candidates, same order.
- A stale pool is impossible after the agent emits output: the next keystroke
  reflects new spans.
- Re-benchmark: per-keystroke cost for the cached path reported alongside the
  0.97 ms baseline above.

## Plan

- [ ] Add the mtime-guarded pool cache in `nvim/init.lua`; keep the rebuild path
      byte-identical so results cannot drift.
- [ ] Give `picks` an in-process generation counter; key the cache on both.
- [ ] Test: N keystrokes ⇒ 1 read; agent writes ⇒ next keystroke sees new spans.
- [ ] Re-benchmark and record both numbers in `## Log`.

## Plan note

`nvim/annotate.lua`'s pure-core/thin-IO split (`ARCH-PURE`) is the local
precedent for making this testable: the pool build is already a pure function of
(file contents, picks) and can be unit-tested directly once the IO is lifted out.

## Log

### 2026-09-06

Filed out of an operator report of draft-pane typing lag, but **not as its
cause** — the benchmark refuted that. Recording the sequence because the
correction is the useful part:

1. Code read found the uncached per-keystroke rebuild and it looked like an
   obvious culprit — 1000 lines, a `pow()` each, a full sort, per keystroke.
2. Benchmarking it against the real file gave **0.97 ms**, which is invisible.
3. So the reported lag is still unexplained, and the leading hypothesis is now
   the **hop count** on the input path rather than any single slow component:
   Ghostty → couch → zellij client → «socket» → zellij server → nvim, with the
   render path the same in reverse — roughly 8–10 process wake-ups per visible
   character. At the ~2.5 ms per-wake-up latency measured on this machine at
   load 6, that is 20–25 ms round trip, which is at the edge of perceptible;
   on an idle machine (<1 ms per wake-up) it would be invisible.

That hypothesis is **not yet measured** and should not be treated as settled. The
honest next step is a direct end-to-end keystroke-to-render measurement, which
nothing in this session performed. Filing this issue on its own (small,
self-justifying) rather than as the fix for a symptom it does not explain.

The general lesson, worth carrying: a plausible mechanism found by reading code
is a hypothesis, not a finding. This one survived a code read and died to a
one-command benchmark.
