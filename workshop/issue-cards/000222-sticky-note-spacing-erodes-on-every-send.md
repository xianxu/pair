---
id: 000222
status: open
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
github_issue:
---

# sticky note spacing erodes on every send

## Problem

The operator uses `===` lines as **sticky notes** — a persistent scratch layout
that survives every send (`init.lua:1003`: *"After any send, the just-sent body's
`===` lines become `*`'s new sticky set"*). Blank lines between them are how that
layout is organised: groups of related notes, separated.

**Those blank lines are eaten a little at a time, so the notes gradually merge
into one blob and no stable layout can be maintained.**

### Measured, by running the function

`comment_lines` (`init.lua:984-1001`) with representative bodies:

| input | output |
|---|---|
| `=== A` / blank / **prose** / `=== B` | **`=== A`, `=== B`** — separator gone entirely |
| `=== A` / blank / **prose** / blank / blank / `=== B` | `=== A`, blank, blank, `=== B` — only the blanks *after* the prose survive |
| `=== A` / blank / `q1` / blank / `=== B` / blank / `q2` / blank / `=== C` | separations halved: two blanks each become one |
| `=== A` / blank / blank / `=== B` (no prose) | preserved correctly |

**The rule in practice: only blank lines immediately preceding a `===` line
survive.** A blank separated from the next note by prose is discarded.

### The line responsible

```lua
for line in (body .. '\n'):gmatch('([^\n]*)\n') do
    if line:match('^%s*===') then
      for _, b in ipairs(pending) do table.insert(out, b) end   -- flush kept blanks
      pending = {}
      table.insert(out, line)
      seen = true
    elseif seen and line:match('^%s*$') then
      table.insert(pending, line)                               -- hold a blank
    else
      pending = {}                                              -- ← prose DISCARDS held blanks
    end
end
```

The `else` branch treats a non-comment line as a reason to forget the blanks
already collected. But the blanks it forgets are *the operator's layout*, and the
prose that triggers the forgetting is exactly what the send is removing.

### Why this erodes rather than being a one-off

The stripped prose is transient; the layout is meant to be durable. And the
operator naturally types **under the note they are thinking about**, which puts
prose directly beneath a `===` line — so on each send that note loses the blank
above the prose. Repeat over a working session and the groups close up one
separator at a time.

The docstring already states the intent this violates:

> preserving blank lines that sit *between* comments so the sticky block keeps
> its spacing across a send. Blank lines before the first comment or after the
> last one are dropped — only interior spacing is structural.

Interior spacing *is* what is being lost. The leading/trailing rule is fine and
should stay.
