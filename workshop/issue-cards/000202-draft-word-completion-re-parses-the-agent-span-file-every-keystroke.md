---
id: 000202
status: open
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
github_issue:
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
