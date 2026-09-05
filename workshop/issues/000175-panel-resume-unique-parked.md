---
id: 000175
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Resume a unique parked thread from the panel

## Problem

`pair#167` resumes a unique parked thread on **`couch` startup** — "on
interactive `couch [<repo>]` startup, before preparing a new root" — installing
it as home. Starting an actor from the **panel** takes a different path and
always creates a new thread, even when exactly one verified-parked thread sits
at that same normalized path.

Observed 2026-09-02 in `/Users/xianxu/workspace/tools`, capacity `bounded,
limit 1`:

```
couch-21baa48c3a7f009b  created 09-01 20:34  verified_park 09-02 08:38  no incarnations
couch-64bbe04986164fae  created 09-02 12:27  incarnation: live
```

Identical `starting_path`, `working_path` and `repo_scope`. Admission is correct
to allow it — it counts live/unknown/creating incarnations, and a parked thread
has none, so park genuinely releases the tree. The gap is that the panel never
asks whether the thread being created already exists parked.

**The ratchet.** `#167` resolves ambiguity by creating: "more than one matching
candidate → treat the set as non-resumable and create a new root." Under bounded
capacity that compounds — two parked threads make a third, three make a fourth,
each one making the next start *more* ambiguous. The state that most needs
resolving is the state the rule most reliably worsens.

## Spec

**Extend `#167`'s unique-parked decision to the panel's start path.** That
decision is already pure and fed normalized values (`#167`: "physicalize
candidate paths in the thin inventory/startup shell and feed only normalized
values to the pure unique-candidate decision"). Only the second caller is
missing — reuse it rather than restating the rule (`ARCH-DRY`); a second
unique-candidate rule that disagrees with the first is worse than neither.

- **Exactly one** verified-parked thread at the normalized path, with exact
  Resume authority ⇒ resume it; create no new record.
- **Zero** candidates ⇒ create a new thread. Unchanged.
- **More than one** ⇒ **diverge from `#167`: show the candidates and let the
  operator pick.** `#167` refuses to prompt for a defensible reason — it runs
  during headless startup, before a TUI exists. The panel is the opposite
  context: the operator is already looking at a list and already choosing, so
  "do not prompt" buys nothing there and preserves the ratchet. This is also
  the behavior the project already committed to for the live case — converting
  an invisible failure into a visible decision at the one moment the operator
  has the context to answer it.
- Path normalization must match admission's, so symlink or alias spellings of
  one path cannot split the candidate set (same requirement `#167` states).
- Never infer eligibility from raw ThreadStore records or labels; consume the
  same verified-park and exact native-binding proof Resume already uses.

**Not in scope:** changing whether park releases capacity. It should — verified
park proves quiescence (`#152`), so a parked thread is genuinely not running and
must not hold the tree. Treating park as occupancy would let one stale park lock
a repo permanently.

## Done when

- Starting an actor from the panel at a path with exactly one verified-parked
  thread resumes that thread; the store gains no new record.
- Zero candidates still creates a new thread.
- With two or more candidates the operator is shown them and picks; no silent
  (N+1)th record is created.
- A symlinked spelling of the same path yields the same candidate set as the
  physical spelling.
- A shadow sweep confirms one unique-candidate decision function with two
  callers, not two implementations.

## Plan

- [ ] Lift `#167`'s startup call site so the pure decision is reachable from
      the panel's start path without duplicating normalization.
- [ ] Unique ⇒ resume; zero ⇒ create. Tests for both, including the symlinked
      spelling.
- [ ] Multiple ⇒ present the candidates in the panel and resume the pick.
- [ ] Shadow sweep for a second unique-candidate rule.

## Log

### 2026-09-02

Found by asking why `/Users/xianxu/workspace/tools` had one parked and one live
thread. It is not an admission bug — the two records above show park correctly
releasing capacity — it is `#167`'s rule being wired to one of its two callers.

Fourth instance today of the pattern recorded in `workshop/lessons.md`: a
mechanism built and connected at one end only. The decision function exists, is
pure, and is called from exactly one place.

Adjacent but distinct: `pair#170`'s couch-lite scope adds resuming a **live**
session. This issue is about the **parked** case at start time. They meet in
the panel's start path and should be built with an eye on each other, but
neither blocks the other.
