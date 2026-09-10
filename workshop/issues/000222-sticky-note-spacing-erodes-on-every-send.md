---
id: 000222
status: open
deps: []
github_issue:
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
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

## Spec

**Interior blank runs survive a send regardless of what non-comment content sat
between the notes.**

The operative requirement is not "preserve whitespace" but **idempotence**:
extracting stickies from a body twice must yield what extracting once yielded.
Today it does not, which is precisely why the layout drifts — each send is a
fresh, slightly lossier pass over the previous send's output.

Decision the implementation must make explicit: when prose sat between two
notes with blanks on **both** sides (`A` / blank / prose / blank / `B`), the
prose is removed — should the result carry one blank or two? Recommended: **one**
— the two runs were separated by content that no longer exists, so they collapse
into a single separator. Whatever is chosen must be idempotent, which the
current behaviour is not.

Keep unchanged:

- blanks before the first `===` and after the last are dropped;
- a run of blanks between two notes with **no** prose between them is preserved
  verbatim (this already works — do not regress it).

## Done when

- **Idempotence**: `comment_lines(comment_lines_as_body(x)) == comment_lines(x)`
  for every case in the table above. This is the test that would have caught the
  bug and the one that prevents its return.
- `=== A` / blank / prose / `=== B` keeps a separator between A and B.
- A note followed by prose does not lose the blank above that prose.
- The no-prose case is byte-identical to today.
- Leading/trailing blanks are still dropped.
- A session-shaped test: apply the extraction repeatedly with prose typed under
  varying notes each round, and assert the sticky layout is **unchanged after
  round 2** — the operator's actual complaint is drift over many sends, not a
  single wrong output.

## Plan

- [ ] Decide the one-blank-vs-two collapse rule; record it in `## Spec`.
- [ ] Rewrite `comment_lines` so prose does not discard held blanks.
- [ ] Table test over the measured cases above, including the no-prose regression.
- [ ] Idempotence test.
- [ ] Repeated-send drift test.

## Log

### 2026-09-09

Operator report: *"we use `===` for the effect of sticky notes; without
preserving white space those sticky notes gradually merge into a single big
blob, user couldn't maintain a constant layout"*.

Diagnosed by extracting `comment_lines` and running it on representative bodies
rather than reading it — the failing case is narrower than "whitespace is not
preserved" (the no-prose case works fine), and locating it precisely is what
makes the fix small and the regression test obvious.

The framing that matters for the fix is **idempotence**, not preservation: the
symptom is drift across many sends, so the invariant to assert is that a second
pass changes nothing.
