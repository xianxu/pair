---
id: 000130
status: working
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 4.31
started: 2026-07-29T10:51:47-07:00
---

# session names: folder prefix, repo token, no redundant tag

## Problem

A zellij session name is a **socket filename**, and on macOS the budget is
**24 bytes** — verified empirically: a 24-character name is accepted, 25 is
rejected with `session name must be less than 0 characters` (zellij computes
its allowance minus the long `~/Library/Caches/org.Zellij-Contributors.Zellij/…`
path and goes negative).

Pair's scheme is `pair-<repo>-<tag>` (`session_index.go:57`), which spends the
budget badly:

- **The prefix costs 5 bytes** (`pair-`) purely as an ownership marker.
- **The repo segment is usually redundant** with the tag, because the create
  flow's name prompt defaults the tag to the cwd basename. Accepting that default
  — the common case — produces `pair-pair-pair`.
- **Overflow is resolved by silent truncation.** `BuildSessionNameCandidates`
  shortens repo and tag a rune at a time until zellij accepts one, so
  `pair-parley_nvim-parley_nvim` (28) becomes **`pair-parley_nv-parley_nv`**
  (exactly 24). The user's reaction on seeing it was "I don't even know why it's
  `parley_nv`" — the truncation is invisible and unexplained.

There is also a latent unit bug: `BuildSessionNameCandidates` truncates by
`[]rune` while zellij's limit is **bytes**. `pair-` is 5 runes and 5 bytes so
they agree today; any non-ASCII component breaks that agreement, and a
multi-byte prefix would make it permanent.

The name is user-visible in `zellij list-sessions`, `pair list`, the cmux
workspace title, and — because zellij composes the terminal title as
`<session> | <focused pane title>` — in the terminal tab title itself. So this
is not an internal identifier the user can ignore.

## Spec

New format, no separator after the prefix:

```
📁{repo}[-{residual tag tokens}]
```

1. **Prefix `📁`** — 4 bytes, no hyphen, replacing `pair-`'s 5. Saves a byte and
   reads as "directory-scoped session", which is what the repo segment encodes.
   Verified as a session name: `📁pair` and `📁pair-1` both create, list,
   address by name, and delete cleanly, with the emoji directly against
   alphanumerics.
2. **Repo = the first alphanumeric token of the repo display name.** Split on
   non-alphanumeric characters and take the first: `parley.nvim` → `parley`,
   `pair` → `pair`.
3. **Tag is used in full** — no truncation in the normal case.
4. **Drop tag tokens already carried by the repo.** Split both into alphanumeric
   tokens; drop the tag's leading tokens that match the repo's; join what is
   left. Worked examples:

   | repo | tag | tokens | result |
   |---|---|---|---|
   | `pair` | `pair` | `[pair]` vs `[pair]` → residual `[]` | `📁pair` |
   | `pair` | `pair-1` | `[pair]` vs `[pair,1]` → residual `[1]` | `📁pair-1` |
   | `parley` (from `parley.nvim`) | `parley_nvim` | `[parley]` vs `[parley,nvim]` → residual `[nvim]` | `📁parley-nvim` |
   | `parley` | `work` | no overlap | `📁parley-work` |

5. **Refuse early instead of truncating.** With `📁` the budget for
   `{repo}[-{residual}]` is **20 bytes**. When a name would overflow, the create
   flow's name prompt refuses and states the actual limit, rather than silently
   shortening. Silent truncation is what produced `parley_nv`; a prompt can say
   "tag too long — 20 bytes available, that needs N".
6. **Truncation, where it still exists, is byte-based.** Fix
   `BuildSessionNameCandidates` to measure bytes, not runes. This is a
   prerequisite for a multi-byte prefix, not an optional cleanup.
7. **Dual-prefix transition.** Discovery must accept **both** `pair-` and `📁`
   while only ever emitting `📁` for new sessions. The prefix is pair's ownership
   filter in zellij's *global* namespace — `zellij.go:27`, `zellijparse.go:60`,
   `legacy_live.go:19`, `lifecycle.go:158` — and flipping the literal without a
   transition orphans every live session, ledger row, and cmux ownership file at
   once.

### Interaction with the cmux workspace title

The cmux workspace title mirrors the **session name** (`updateWorkspaceTitle`,
`titlepoller/run.go:204`) with an activity-heat prefix, and passes it through
`titlefmt.EmojiTitle` (`launcher/osruntime.go:419` — the only call site; pane
renames never touch it). `EmojiTitle` returns a title with no `-` verbatim,
otherwise splits on `-` and maps exact tokens (`brain`→🧠, `book`→📗,
`pair`→♋).

Today `pair-pair-pair` therefore renders in cmux as **`♋-♋-♋`**. Under this
spec: `📁pair` has no `-` so it is returned verbatim; `📁pair-1` splits to
`[📁pair, 1]` where `📁pair` no longer matches the `pair` key; `📁parley-nvim`
matches nothing. All three read better.

**Consequence to accept deliberately: this effectively retires `EmojiTitle`.**
The glued-on `📁` means no token can ever match `emojiWords` again. That is
probably right — one glyph convention instead of two competing ones — but decide
it here rather than discovering it. If the word→emoji mapping is still wanted,
it has to move ahead of the prefix (map the repo token, then prepend `📁`).

### Accepted trade-offs

- **`📁pair-1` is ambiguous** between tag `pair-1` and collision suffix #1.
  Harmless functionally — `session-names.jsonl` maps name → scope authoritatively
  — but a human reading `zellij list-sessions` cannot tell them apart.
- **Rule 2 discards information.** `parley.nvim` → `parley` means a sibling repo
  actually named `parley` now collides where it did not before, resolved by an
  opaque numeric suffix. Accepted: few repos share a first token.
- **The ambiguity is wider than the `📁pair-1` case above** (noted round 4).
  Because the ladder drops residual tokens *whole*, tag `parley_nvim` under
  collision pressure yields `📁parley-2` — which reads as tag `parley`, collision
  #2. Same class of ambiguity, one step further; still harmless because
  `session-names.jsonl` is authoritative, but it is a larger claim than the
  single-case paragraph above.

### Ledger migration: full migration, no grandfathering

**Operator decision, 2026-07-29 (supersedes the round-4/5 grandfather design).**
Every legacy `pair-…` name migrates. The transition is a one-time event the
operator drives from a quiesced state — all other sessions concluded, all other
zellij sessions cleaned up — rather than a behavior the code carries indefinitely.

`AssignSessionName` (`session_index.go:98-100`) short-circuits on the ledger
before any composition:

```go
if prior, ok := index.latestFor(scope.Key, tag); ok && accepts(prior.SessionName) {
    return prior.SessionName, index, nil
}
```

`accepts` only asks whether zellij tolerates the *name's length*, not whether a
session exists — so untouched, any tag with a ledger row keeps its `pair-…` name
permanently and the new scheme never reaches the tags that motivated the issue.

**The condition becomes simply: short-circuit iff the prior name already carries
`sessionPrefix`.** A legacy row always falls through and re-mints; a new-format
row stays pinned by the ledger exactly as today.

That prefix clause is load-bearing, not decoration — it is what stops the ledger
growing without bound. Today the short-circuit means the append at
`session_index.go:106-113` never runs for a known (scope, tag), so
`session-names.jsonl` holds one row per pair ever. Remove the short-circuit
outright and every create recomposes the same `📁pair`, passes `ownedByOther`
(same scope + tag) and `liveOwnedByOther` (not live), and appends an **identical**
row that `createflow.go:396-401` persists — once per create, forever. With the
prefix clause a legacy name re-mints exactly once and then pins.

**Reclaim the superseded zellij record — but never an attached one.** The old
`pair-…` session is still a row in `zellij list-sessions`, and `index.ownerOf`
still resolves it to this scope, so `SessionsForScope` would keep feeding it to
`pair list` as a permanent second row for the same tag. Nothing would ever clean
it up, because `delete-session --force` only fires against a name pair is *about
to reuse* (`OSRuntime.SessionBlocksReuse`) or the session that just exited
(`lifecycle.go:66`) — and after migration nothing ever names `pair-pair-pair`
again. So re-minting also deletes the superseded session through the
`DeleteSession` seam. **Guard: skip the reclaim when the superseded session is
`SessionAttached`** — force-deleting a session with a live client would kill a
terminal out from under someone. Detached and `EXITED` both reclaim.

**The reclaim must fire at the commit point, not at name-assignment time**
(`ARCH-PURE`). `AssignSessionName` is pure, so the IO lives in a caller — and the
two callers are not interchangeable. `assignLaunchSessionNames`
(`createflow.go:249-284`) runs at `createflow.go:154`, *before* `DecideLaunch`
(`:159`), before the picker (`:169`), before `promptForTag` (`:315`) — on **every**
launch invocation, including ones that resolve to attach and ones the user ESCs
out of. A reclaim there would force-delete a resurrectable record on an abandoned
`pair`, which is strictly more destructive than today, where deletion is gated
behind commitment (`rt.SessionBlocksReuse` at `:341`, post-prompt).
The plan already solved exactly this for the ledger *write*: `AssignSessionName`
returns the entry, `assignLaunchSessionNames` stashes it, and `runCreate` commits
it at `:396-401`. The reclaim rides that same seam — `AssignSessionName` gains a
second return naming the superseded session, the caller stashes it next to
`newEntries`, and the `DeleteSession` call sits beside `AppendSessionNameIndex`
at the commit point. Decision pure, IO at commitment.

### The session running this work migrates last

Pair cannot rename a live zellij session underneath itself — there is no
`rename-session`, and a session's panes do not survive being recreated. So the
session hosting this implementation is the one case the code cannot migrate: it
migrates when it is quit and relaunched, at which point the ledger's legacy row
re-mints to `📁pair` on the normal create path.

That makes the final step a deliberate handoff rather than a code path:

1. Operator concludes every other pair session and clears the other zellij
   sessions, so only this one is live.
2. Land the implementation and verify it against a **fresh tag** first (the
   migration path is exercised without betting the working session on it).
3. Quit this session; relaunch; confirm it comes back as `📁pair`.

**A continuation is written before step 3** so the work survives the restart —
and survives the restart *failing*. See `workshop/continuation/`.

### Why the ownership prefix cannot simply be dropped

`session_blocks_reuse` calls `zellij delete-session --force` on `EXITED` rows.
Without a prefix distinguishing pair's sessions from foreign ones, a stranger's
abandoned session whose name matched a tag would become a deletion target. The
global list already contains foreign names (`fabulous-aardvark` was present while
scoping this). The prefix is load-bearing; only its cost is negotiable.

## Done when

- **Every** legacy name migrates: `pair-pair-pair` → `📁pair`;
  `pair-parley_nvim-parley_nvim` → `📁parley-nvim`, untruncated. No tag keeps a
  `pair-…` name once its session is gone. (Round 4 found the original phrasing
  unreachable — `AssignSessionName` short-circuits on the ledger row — and
  round 7 replaced the grandfather answer with full migration; see *Ledger
  migration*.)
- The superseded zellij record is reclaimed, so a migrated tag shows **one** row
  in `pair list`, not a permanent second `status: exited` one. An **attached**
  session is never reclaimed.
- The session hosting the implementation comes back as `📁pair` after a quit and
  relaunch, and a continuation exists that survives that restart failing.
- A tag that would overflow the budget is refused at the name prompt with the
  real limit quoted, not silently shortened.
- Existing `pair-*` sessions — live, detached, and historical ledger rows — are
  still discovered, attachable, and resumable after the change.
- Length arithmetic is byte-denominated: the `📁` prefix costs **4 bytes / 1
  rune**, and a candidate at the byte edge is neither over-shortened nor emitted
  over-budget. (Also reworded in round 4. The original "a test with a multi-byte
  *component* proves it" is not constructible: `NormalizeDisplayComponent`
  (`scope.go:44-59`) maps every rune outside `[A-Za-z0-9_-]` to `_`, so both
  components are ASCII by construction and the prefix is the only multi-byte
  element in the scheme. The reachable regression is prefix miscounting — 21
  runes measuring 24 bytes — not a multi-byte component.)
- `pair list`, the picker, `pair resume`, rename, and cmux ownership all work
  against a `📁` session.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as **stale** (the ledger is newer than the doc), so the number is
provisional but uses the required method.

Design hours take the ×0.2 spec-quality discount (v2 Step 3): the byte-vs-rune
finding, the 24-byte budget, the emoji end-to-end verification and the four
worked naming examples were all established before this block was written, and
the operator has already fixed the six format decisions. What is left is
consolidation plus a careful sweep. Design buffer is therefore **+15%**, not
+30% (v2.1 Step 6 — halved because the discount already credits the front-loaded
design).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.18
item: smaller-go-module design=0.05 impl=0.16
item: smaller-go-module design=0.06 impl=0.18
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.14 impl=0.32
item: lua-neovim design=0.05 impl=0.12
item: smaller-go-module design=0.02 impl=0.08
item: smaller-go-module design=0.05 impl=0.18
item: smaller-go-module design=0.03 impl=0.14
item: smaller-go-module design=0.03 impl=0.10
item: smaller-go-module design=0.08 impl=0.32
item: cross-cutting-refactor design=0.10 impl=0.40
item: smaller-go-module design=0.05 impl=0.20
item: milestone-review design=0.05 impl=0.12
item: atlas-docs design=0.04 impl=0.06
design-buffer: 0.15
total: 4.31
```

Item mapping, in plan order: `issue-spec` = this block plus the three Revisions
rounds; the `smaller-go-module` rows are `ComposeSessionName` + its table test,
the rewritten byte ladder, the lazy budget calibration + parameterized limit +
length-aware fake, the refuse-early decision + `promptForTag` signature change +
message reconciliation, the `EmojiTitle` retirement + `PublicSessionBase`
deletion, the destructive-path seam move + stateful fake + test, the
mixed-snapshot table test, the `pair list` display-width fix, and the manual
live check; `lua-neovim` = dropping
`pair_rename_prompt`'s strip + charset twin; the two `cross-cutting-refactor`
rows are (i) the dual-prefix predicate plus the 15-site writer-vs-reader sweep
and the three index-less strip sites, and (ii) the ~300-literal test-corpus
re-baseline, which is a per-file judgment call rather than a sed — both are the
multi-file-rename shape rather than a module; `milestone-review` = the single
close boundary; `atlas-docs` = `atlas/session-identity.md` + the
`architecture.md` cross-link + the filename-component invariant.

Plausibility note: well above **#129** (est 1.83) and **#127** (est 1.40 / actual
1.40) — correctly, after six plan-quality rounds grew the scope from 8 items to
18. Less *new* logic than #129 (one pure function, one ladder rewrite) but a much
wider blast radius: 18 call sites across three Go packages **and Lua**, a
deliberately non-invertible name format, a `delete-session --force` path that
first has to be made observable, a budget that must be discovered rather than
assumed, and a test corpus of ~300 literals where "which stay `pair-`" is itself
the transition coverage. The two largest rows — the sweep and the re-baseline —
are also the two least mechanical. Round 4's judge expected the total nearer
4.5h; six rounds of derived growth have brought it to 4.31h on its own, without
back-fitting. The unbounded risk remains the live check,
which needs a PTY-attached zellij client (lessons.md) plus a pre-existing
`pair-*` session to prove the transition. The local trend runs toward
under-estimating (#125 est 0.45 / actual 1.43).

## Plan

Single review boundary — no `Mx` tags.

- [x] **`ComposeSessionName(scope RepoScope, tag string) string`** — one pure
      function implementing rules 1–4, living beside the existing scheme in
      `launcher/session_index.go`. Composes on top of the existing
      `NormalizeDisplayComponent` (`scope.go:44`) rather than re-deriving
      sanitisation: that already maps `parley.nvim` → `parley_nvim`, so rule 2 is
      "first alphanumeric run of the normalised value" (`ARCH-DRY`).
      *Deliberately not named `PublicSessionName`* — `publicSessionName` already
      exists at `:86` and also mints names; two functions differing only in
      leading case is a grep hazard. That one is renamed
      `withCollisionSuffix(base string, suffix int) string` — all it actually
      does once composition moves out — **with the ladder rewrite**, since both
      edits land in the same function (tracked on that item, not this one).
      And `PublicSessionBase` (`:56`)
      is **deleted**: it hardcodes `"pair-"`, and a tree-wide grep finds zero
      callers — leaving it is a second, stale minter (`ARCH-DRY`).
      **One tokenizer, one prefix const.** The ladder's signature wants
      `(repo string, residual []string)` while callers hold `(scope, tag)`, so
      name the decomposition explicitly: `sessionNameParts(scope, tag) (repo
      string, residual []string)` is the single implementation of rules 2–4, and
      `ComposeSessionName` is `sessionPrefix + join(parts)` over it. Otherwise
      the implementer either re-parses the composed string or re-implements the
      tokenization — swapping the `PublicSessionBase` duplicate this item deletes
      for a fresh one (`ARCH-DRY`). `sessionPrefix` is one const that both the
      composer and the ladder read.
      Test `TestComposeSessionName` — table over the four spec rows, plus the
      cases where the rules interact: tag equal to repo, tag prefixed by repo,
      tag unrelated, repo with no alphanumerics (normaliser's `"pair"`
      fallback), empty tag, and the two rows where rule 4 is genuinely ambiguous:
      **repo token appearing non-leading in the tag** (repo `pair`, tag `my-pair`
      → residual `[my, pair]` → `📁pair-my-pair`, correct per "drop the *leading*
      tokens" but surprising enough to pin), and **case** (repo `Pair`, tag
      `pair` — `NormalizeDisplayComponent` preserves case, so exact-match
      tokenization leaves the redundancy in place).
- [ ] **Rewrite the ladder against the composed base — full contract, not "convert
      to bytes".** Today `BuildSessionNameCandidates(scope, tag, suffix)` (`:60`)
      shortens `repo` and `tagPart` *independently*, each floored at
      `minSessionComponentRunes = 4`, then rebuilds via `publicSessionName`. Once
      composition moves into `ComposeSessionName`, that two-component floor maps
      onto nothing. The replacement contract, written down before the edit:
      - **Signature.** `sessionNameLadder(repo string, residual []string, suffix
        int) []string` — the ladder owns shortening; `sessionNameParts` owns
        tokenization; `withCollisionSuffix` owns the `-N`.
        **No `limit` parameter** (corrected round 4). An earlier draft passed one,
        which does not compose with "the probe is the acceptance oracle,
        calibrated lazily on rejection": whoever called the ladder would need a
        limit *before* any rejection had occurred, leaving only the macOS-derived
        fallback of 20 — which on Linux (`~/.cache/zellij`, a much shorter socket
        path) would drop residual tokens zellij would have accepted. That is
        unexplained shortening reintroduced in the very function this issue
        exists to fix, on the platform we cannot test. It is sharpest on the
        collision path: suffixes ≥2 are driven by collisions, not rejections, so
        no calibration would ever have run there. The ladder therefore emits
        candidates longest-first and `AssignSessionName` filters them through
        `accepts` exactly as it does today; `limit` survives only in the prompt's
        refusal message.
      - **Order of sacrifice.** (1) full `📁repo-res1-res2…`; (2) drop residual
        tokens **whole**, from the right — a half-token like `parley_nv` is the
        exact unexplainable artifact this issue exists to kill; (3) only when no
        residual is left, byte-truncate `repo` down to a floor of
        `minSessionComponentBytes = 4`, never splitting a UTF-8 rune.
      - **Guaranteed shortest** = `📁` + 4 bytes of repo = 8 bytes. If even that
        does not fit, `SessionNameExhausted` — that is the only exhaustion point.
      - **The suffix needs no fixed reservation.** The ladder already takes
        `suffix`, so each candidate is measured *with* its `-N` applied. The
        original defect was purely that the measurement was in runes against a
        byte budget (`📁` = 1 rune / 4 bytes); measuring the suffixed candidate
        in bytes fixes it without stealing 4 bytes from every name.
      Test: the prefix is counted as 4 bytes / 1 rune, so a candidate that
      measures 21 runes but 24 bytes is handled by its byte length (this is the
      reachable form of the bug — *not* a multi-byte component, which
      `NormalizeDisplayComponent` makes impossible); a base at exactly the limit
      whose suffixed form still fits; and that no candidate ever ends in a
      partial residual token or a split rune.
- [ ] **Tighten the ledger short-circuit so the new format is reachable at all.**
      `AssignSessionName` (`session_index.go:98-100`) returns the prior ledger
      name whenever `accepts` tolerates its *length*, which means a tag with an
      existing row keeps `pair-…` forever and the headline outcome never happens.
      Add the condition specified in the Spec's *Ledger migration* section:
      **short-circuit iff the prior name already carries `sessionPrefix`.** A
      legacy row always falls through and re-mints (full migration, operator
      decision); the prefix clause is what stops the ledger appending an
      identical row on every subsequent create. The decision stays pure
      (`ARCH-PURE`) and needs no liveness input at all — simpler than the
      grandfather design it replaces.
      **Ordering constraint, found by landing it early and watching a test
      fail:** this hunk must ship *with* the ladder rewrite, never before it.
      While `BuildSessionNameCandidates` still emits `pair-…`, a legacy row that
      falls through re-mints **another legacy name** and appends a duplicate
      ledger row on every create — the exact unbounded-growth defect round 5
      caught, reintroduced by intermediate state rather than by design.
      `TestAssignSessionNameReusesSameScopeBinding` fails loudly on it; a comment
      at the call site records the constraint.
      Then **reclaim the superseded record at the commit point**: on re-mint,
      delete the legacy session through the `DeleteSession` seam the
      destructive-path item is already creating, so it does not linger forever as
      a second `status: exited` row in `pair list` — **skipping the reclaim when
      that session is `SessionAttached`**, since force-deleting a session with a
      live client kills a terminal out from under someone — but wire it beside
      `AppendSessionNameIndex` in `runCreate` (`createflow.go:396-401`), **not**
      in `assignLaunchSessionNames`, which runs at `:154` before the picker and
      the prompt. See the Spec's *Ledger migration* for why: a reclaim at
      assignment time force-deletes a resurrectable record on an abandoned
      `pair`. `AssignSessionName` gains a second return naming the superseded
      session; the decision stays pure and the IO sits at commitment
      (`ARCH-PURE`).
      Test: (a) a legacy row re-mints to `📁…` regardless of the old session's
      state; (b) a detached legacy session is reclaimed; (c) no ledger row at all
      mints `📁…`; (d) an **attached** legacy session re-mints but is **not**
      reclaimed; (e) **two consecutive creates for one tag yield exactly one new
      ledger row** — the unbounded-growth guard, and the second create
      short-circuits on the `📁` row; (f) the
      superseded `EXITED` session reaches `DeleteSession` exactly once
      (exercising the new stateful fake); (g) an invocation that resolves to
      **attach**, or that the user aborts at the picker or the prompt, reaches
      `DeleteSession` **not at all** — the regression the commit-point placement
      exists to prevent. Case (b) is what makes Done-when bullet 1 true; (e), (f)
      and (g) are what keep it from being a regression.
- [ ] **Discover the budget; never hardcode it.** The Spec derives 20 bytes from
      *this* machine — zellij's allowance is the socket-path budget minus
      `~/Library/Caches/org.Zellij-Contributors.Zellij/…`, which varies with
      username and is a different path entirely on Linux (`~/.cache/zellij`).
      The codebase already refuses to model this: `OSRuntime.ProbeSessionName`
      (`osruntime.go:77`) shells to zellij and `sessionNameRejected`
      (`zellijparse.go:48`) matches its error string, and `accepts` is threaded
      into `AssignSessionName` as an injected predicate (`createflow.go:479`).
      **The probe stays the acceptance oracle.** The numeric budget exists only
      for the refusal *message*, and it is calibrated **lazily, on rejection
      only**: probe the composed candidate exactly as today (one exec, the
      happy path pays nothing extra), and only when that comes back rejected do
      we binary-search for the number worth quoting. **Bounds, so the cost is a
      stated choice rather than an accident:** search 1..64 bytes → ~6 probes,
      each a sequential `zellij … list-clients` exec under `zjTimeout`
      (`osruntime.go:77-85`), in front of an interactive prompt. That worst case
      is accepted because it is reached only on the already-failing path; the
      happy path stays at the single probe it does today. **The search must probe
      synthetic pad names guaranteed not to exist**, not plausible real ones:
      `ProbeSessionName` runs `zellij --session <name> action list-clients`
      (`osruntime.go:80`), which *succeeds* against a foreign live session and
      would then read as "fits" for entirely the wrong reason — making the
      measured budget depend on the global namespace. A per-process cache would
      buy nothing — `pair` create is a one-shot process, so "cached" would just
      mean N `list-clients` execs before every prompt renders. Fall back to 20
      when calibration itself fails. The pure decision function takes `limit int`
      as a **parameter** — never a package const (`ARCH-PURE`).
      **Conformance decision (`ARCH-MOCK`), stated rather than left silent:** the
      runtime probe stays the *sole* oracle, so the fake's length model only has
      to be self-consistent — it never needs to agree with real zellij, because
      nothing branches on it in production. That is why no build-tagged live
      conformance test is scheduled here; the manual live check covers the real
      binary end-to-end.
      Test: the pure decision at several limits including a Linux-sized one; and
      teach `fakeRuntime.ProbeSessionName` (`createflow_test.go:109`, currently
      `return f.probeErr` — a constant, blind to length) a length model, so
      refuse-vs-accept is exercisable end-to-end.
- [ ] **Refuse early rather than truncate — and name the seam.** The refusal
      belongs at the create flow's name prompt, but `promptForTag`
      (`createflow.go:495`) takes `(rt, prefill, base, stderr)` — no `RepoScope`,
      no cwd — and the repo display name only enters at `assignSingleSessionName`
      (`:470`), *after* the prompt returns. So this item changes `promptForTag`'s
      signature to carry the scope (or the already-resolved limit + repo token)
      and re-prompts on overflow with the real numbers.
      Composition rule, stated once so the three Spec sentences resolve: the
      **prompt** refuses on the interactive path; the byte ladder survives as the
      last resort for the **non-interactive** paths (`PAIR_TAG`, resume), which
      have no prompt to refuse at; `SessionNameExhausted` remains the loud floor
      when even the shortest candidate is rejected. The prompt checks the *bare*
      base and does not pre-reserve suffix room — a name that fits bare but not
      suffixed simply falls to the ladder, so each mechanism keeps one job.
      **Reconcile the third message.** Two spellings of this failure already
      exist and this item would add a third: `createflow.go:334-337` prints "tag
      '%s' makes zellij's session name too long for this machine's socket path"
      after assignment. It survives as the **non-interactive floor** message and
      picks up the calibrated number when one is available; the new prompt text
      is the interactive one; `SessionNameExhausted` keeps its own for the
      exhausted-ladder case. Three paths, three deliberate messages — not one
      failure with three accidental spellings.
      Test: the pure decision (name + limit → over/under + message), driven
      directly; the prompt loop is the thin IO shell around it (`ARCH-PURE`).
- [ ] **Retire `titlefmt.EmojiTitle`** (operator decision, 2026-07-29). With `📁`
      glued to the repo token, no token in a **session name** can ever match
      `emojiWords` again. (Precisely: not *wholly* unreachable — `CmuxRename` is
      also called with a cwd basename at `lifecycle.go:130`
      (`reset := filepath.Base(env.Cwd)`), which still reaches `EmojiTitle` at
      `osruntime.go:419`, so a hyphenated cwd like `my-brain` maps today. That
      residual path is the quit-time workspace reset, not a session title, and
      retiring the map is the operator's decision either way — but the
      justification is corrected here so the close review does not re-derive it
      and reach a different conclusion.) Meanwhile `format_test.go:66-86` keeps
      asserting `EmojiTitle("pair-brain-book") == "♋-🧠-📗"` — a green test
      pinning behavior nothing can reach. Delete `titlefmt.EmojiTitle` +
      `emojiWords` (`titlefmt/titlefmt.go`), the `launcher.EmojiTitle` shim
      (`format.go:69`) and that test; the two call sites — `osruntime.go:419`
      (`CmuxRename`) and `titlepoller/titlepoller.go:85`
      (`cmuxWorkspaceTitle`) — pass the session name through verbatim. Note the
      Spec undercounts these as one call site; there are two.
- [x] **Dual-prefix ownership predicate.** One predicate (`isPairSessionName`)
      accepting `pair-` **and** `📁`, shared by the two sites that ask *"is this
      name in pair's namespace?"*: `zellij.go:27` (snapshot filter) and
      `zellijparse.go:60` (`pairSessionNames`). Not four edited literals — four
      independent copies of one rule is the divergence shape that caused #127.
      Test: the predicate over `pair-x`, `📁x`, `fabulous-aardvark`, `""`.
      **The other two sites named in the original plan are a different question,
      not the same predicate** — see the next item.
- [ ] **Keep prefix-strip→tag recovery on `pair-` only.** `legacy_live.go:19,25`
      and `lifecycle.go:158` (`liveTagsForSweep`) both filter *and then*
      `TrimPrefix` to recover the tag. That inverse is only valid for the legacy
      scheme: stripping `📁` off `📁parley-nvim` yields `parley-nvim`, which is
      **not** the tag (`parley_nvim`) — rules 2 and 4 discard information by
      design, so the new format is deliberately not invertible. Both sites
      already consult `SessionNameIndex` first and fall back to the strip only
      for *unindexed* sessions, and every `📁` session is indexed by construction.
      So these stay `pair-`-only, and that is a correctness requirement, not an
      oversight. Test: a `📁` session absent from the index yields **no** tag from
      either site (rather than a plausible-looking wrong one).
- [ ] **Three more strip→tag sites have no index at all — fix, don't leave.** The
      "already consults the index first" justification above holds for those two
      and *not* for these, which `TrimPrefix` blind:
      `restart.go:22` (`tag = TrimPrefix(session, "pair-")` when `PAIR_TAG` is
      unset — the tag then drives `rt.InferAgent(tag)` and the `RestartMarker`),
      `pick.go:86` (`sessionTag`'s fallback when `s.Tag == ""`), and
      `list.go:86-87` (`tagFromPublicSessionName` returns the name **verbatim** on a
      prefix miss, feeding `inferAgent(tag)` at `list.go:73`). Each would mint
      exactly the plausible-looking wrong tag the previous item refuses to
      create. Give each an index lookup — but **the miss behavior is per-site,
      not one uniform rule**:
      - `list.go:86` → return empty; it feeds only `inferAgent`, which already
        tolerates an empty tag.
      - `restart.go:22` → return empty, but **not** for that reason (corrected
        round 5): the recovered tag also becomes `RestartMarker{Tag: …}`
        (`restart.go:24-25`). Empty is still safe because the restart re-entry
        does `rTag := firstNonEmpty(m.Tag, step.tag)` (`createflow.go:83`), so an
        empty marker tag falls back to the step's. Pin that in the test rather
        than assuming it.
      - `pick.go:86` → **must not** return empty. `sessionTag` feeds the live-dedup
        key `live[sessionTag(s)] = true` (`pick.go:38`) and
        `pickSelection{tag: …}` (`pick.go:55`); an empty tag would collide every
        unresolved row into one dedup bucket and make selection resolve to no
        tag. Fall back to `s.Name` there, so the row stays unique and selectable
        even when unresolvable.
      Extend the same test criterion to all three, with `pick.go`'s asserting two
      unresolvable rows stay distinct rather than merging.
- [ ] **Sweep the remaining `"pair-"` sites** — wider than first inventoried.
      Writers (mint a name; must go through the new scheme or an index lookup):
      `decision.go:60` (`sessionName`), `lifecycle.go:23` (attach fallback),
      `compaction.go:73`, `titlepoller/run.go:79`, `titlepoller/run.go:216`
      (cmux-owner fallback), `createflow.go:496` (`ShowFamilyExisting`).
      Comparators / readers (compare against or strip a name that may be either
      scheme): `compaction.go:36`, `continuationcmd/draft.go:108`
      (`InCompactionContext`), `rename.go:142`, `tag.go:11` (`NormalizeTag`),
      `pick.go:86` (`sessionTag`), `list.go:86-87` (`tagFromPublicSessionName`),
      `restart.go:22`. Decide per site: *writer* (must move) vs *legacy reader*
      (leave). Getting this wrong is silent — a stale writer mints a `pair-` name
      the new scheme never expects. Note the near-miss:
      `entrypoint/alias.go:34,37` also strips `pair-`, but from **busybox binary
      names** (`pair-slug`), not session names — explicitly out of scope, and a
      blind sweep would break the Stop-hook symlink.
      **Three more hand-spelled restatements of the scheme, all user-facing:**
      `pick.go:117` (`fmt.Sprintf("pair-%s  (%s, no live session)", …)` — the
      historical picker label, which would otherwise show `📁pair-work` for a
      live row and `pair-work` for the same tag's historical row), and
      `rename.go:170` / `:175` (`"session 'pair-%s' is still tracked by zellij"` /
      `"already exists in zellij"`) — the user-facing halves of the
      `sessionTracked` call at `:142` that this item already lists.
      **Per-site verdict for `decision.go:59` + `createflow.go:473` — legacy
      degraded fallback, stays `pair-`, not "writer that must move."**
      `sessionName(tag)` is reached via `sessionNameForTag` (`decision.go:63-70`),
      which is pure over `snap.SessionNames`, and `SessionSnapshot`
      (`session.go:36-43`) carries no `RepoScope` — there is nothing to compose
      from. **Corrected round 5:** an earlier draft justified this as "fires only
      when `ResolveRepoScope` fails". That is false — `sessionNameForTag` falls
      back whenever the tag is absent from `snap.SessionNames`, and
      `assignLaunchSessionNames` (`createflow.go:260-283`) populates only the
      single launch tag, so `nextFreeTag`'s probes for `base-2`, `base-3`, …
      (`decision.go:97`) reach it on the perfectly normal path with the scope
      resolved fine. The verdict is unchanged but the *reason* is different and
      matters: the name there is only ever **compared** (through
      `sessionBlocksReuse` against the snapshot), never minted — so a legacy
      spelling is harmless, but the site participates in free-slot arithmetic.
      Test one `nextFreeTag` row against a live `📁<repo>` session.
      (`createflow.go:473` was missing from the inventory entirely.)
      Every writer that *does* have a scope in reach needs the same fallback:
      prefer the `SessionNameEntry` for the tag, and only then compose.
      **`NormalizeTag` (`tag.go:11`) needs its own decision:** its charset loop
      allows only `[A-Za-z0-9_-]`, so a user pasting `📁parley-nvim` out of
      `zellij list-sessions` into `pair resume` gets `contains invalid character
      '📁'` — today the equivalent paste works, because `pair-` is stripped.
      Keep `NormalizeTag` pure and tag-only; add a pure
      `TagForSessionName(index, raw) (string, bool)` that resolves a
      `📁`-prefixed argument through the ledger, and call it ahead of
      `NormalizeTag` at **two** sites, having decided all four:
      - `args.go:91` — `pair resume <tag>`.
      - `rename.go:62` — `validateRenameTags`'s **old** tag only. This is the
        site the next item hands the job to, and it was missing here.
      **Not** `runcli.go:106`: that is `NormalizeTag(args.ContinueSlug)` for
      `pair continue <slug>`, where the argument is a continuation-doc slug
      resolved by `rt.ResolveContinuationDoc` (`runcli.go:111`) — never a session
      name. Ledger resolution there would resolve nothing and mis-scope the seam.
      **Nor `createflow.go:504`**, the fourth site, inside `promptForTag` — the
      very function the refuse-early item rewrites, which is why it is decided
      here rather than left to be noticed mid-edit. It stays bare for the same
      reason the rename *new* tag does: the value being normalized names a
      session that does not exist yet. Nothing regresses, because `NormalizeTag`
      keeps its `pair-` strip.
      **The rename *new* tag does not resolve** — it names a session that does
      not exist yet, so there is nothing to look up. `validateRenameTags`
      (`rename.go:65`) keeps plain `NormalizeTag` for it, and a user pasting a
      `📁` name into the new-tag slot gets a message saying so ("that's a session
      name; give the new tag in bare form") rather than the raw charset error.
      An unindexed `📁` old-tag errors with what it is, rather than being guessed
      at. Test: one case per site — resume, rename-old, rename-new — plus
      `pair continue` proving the slug path is untouched.
- [ ] **The scheme has a fourth consumer language — `nvim/init.lua`.**
      `pair_rename_prompt` carries a Lua twin of both Go rules:
      `input:gsub('^pair%-', '')` (`:3266`) and
      `new_tag:match('^[%w_-]+$')` (`:3271`). This sits directly on a Done-when
      clause (`Ctrl+Alt+n` rename works against a `📁` session), and the change
      makes the regression *more* likely, not less: the tab title will read
      `📁pair`, so pasting it into the rename prompt is the natural user move and
      today yields `invalid tag (allowed: letters, digits, dash, underscore)`.
      Resolve by **deleting the Lua-side strip and charset check** and letting
      `pair rename --restart-check` — which the prompt already shells to at
      `:3274` — own resolution and error text (`ARCH-DRY`: one implementation of
      the rule, in the binary, not a Lua restatement that drifts).
      **Ordering constraint: this item is only safe once the previous item has
      put `TagForSessionName` on `rename.go:62`.** Deleting
      `input:gsub('^pair%-','')` while the binary still runs bare `NormalizeTag`
      leaves rename *worse than today* — the paste that currently succeeds
      (`pair-work`) would start failing alongside the `📁` one. Land them
      together, and make the Lua prompt's regression check a real paste of the
      tab-title text.
      Explicitly *not* affected, so the next reader need not re-audit them:
      `nvim/init.lua:35` and `nvim/workbench_route.lua:61` read
      `ZELLIJ_SESSION_NAME` but only round-trip it for exact comparison — they
      are scheme-agnostic.
- [ ] **Verify the destructive path specifically.** `OSRuntime.SessionBlocksReuse`
      (`osruntime.go:63`) force-deletes the named session when zellij reports it
      `EXITED`, and it is called on a name `AssignSessionName` already chose. The
      chain that makes this dangerous: `AssignSessionName`'s collision check
      consults only the ledger and `live` — and `live` is *already* filtered by
      the ownership predicate, so a **foreign** session is invisible to collision
      avoidance. Widen that predicate by accident and pair can mint a name a
      stranger owns, then delete it.
      **First make the destruction observable, or the test proves nothing.** The
      force-delete is *inside* `OSRuntime.SessionBlocksReuse` (`osruntime.go:71`),
      below the Runtime seam, and `fakeRuntime.SessionBlocksReuse`
      (`createflow_test.go:108`) is a bare `return f.blocksReuse[session]` — no
      call log, no state change. A test asserting "never handed to
      `SessionBlocksReuse`" would pass while the hazard sits untouched. So:
      **lift the `delete-session --force` out of `SessionBlocksReuse` into the
      existing `DeleteSession` seam** (a one-line move) so the destructive act is
      observable by construction — `DeleteSession` is already recorded as
      `rt.deleted` and asserted at `lifecycle_test.go:73`. The fake also gains
      the matching state transition (an `EXITED` row *disappears* from
      `f.sessions` once deleted), which is the behavior-across-calls the
      stateful-double rule asks for (`ARCH-MOCK`).
      Then test both ends: (a) the predicate rejects `fabulous-aardvark`; (b)
      end-to-end through `AssignSessionName` + the fake, an exited foreign
      session never reaches `DeleteSession`. This is the one place where a wrong
      predicate destroys someone else's state rather than just confusing pair.
- [ ] **Pin the mixed snapshot in a test, not just in the live check.** Done-when's
      highest-risk clause — existing `pair-*` sessions stay discovered, attachable
      and resumable — currently rests on a unit test of the predicate plus a
      manual PTY check that this plan itself flags as the unbounded-risk item.
      Orphaning live sessions costs the user real state, so it gets a table test:
      one `Sessions()` list holding both a legacy `pair-x` and a new `📁y`, run
      through `SessionsForScope` + `buildListRowsForScope` + the pick-label path,
      asserting both appear with the right tag and agent. Add the migration case
      the previous item creates: what `pair list` shows for a **re-minted** tag
      whose legacy row is `EXITED` — exactly one row, the `📁` one, because the
      superseded record was reclaimed.
- [ ] **`pair list` alignment: pad by display width, not runes.** `formatListTable`
      (`list.go:37,43`) pads with `%-30s`, and Go counts that in **runes**. `📁`
      is 1 rune but **2 terminal columns**, so every new-format row would sit one
      column left of its header — the same unit confusion as the rune-vs-byte
      truncation bug, one layer up, and it lands directly on Done-when's "`pair
      list` … works against a `📁` session". Pad by display width.
      Test: a table mixing a legacy `pair-x` row and a `📁y` row asserts both
      columns start at the same offset.
- [ ] **Re-baseline the existing test corpus — a design pass, not find/replace.**
      Roughly 300 `"pair-…"` literals across ~39 test files encode the old
      scheme: `lifecycle_test.go` (35), `pick_test.go` (22),
      `titlepoller/run_test.go` (21), `createflow_test.go` (20),
      `list_test.go` (19), `compaction_test.go` (18), `zellijparse_test.go` (17),
      `session_index_test.go` / `decision_test.go` (15 each),
      `osruntime_test.go` (13), `restart_test.go` (11), and a long tail. Plus
      `session_index_test.go:75`, which asserts `strings.HasPrefix(name,
      "pair-")` outright, and `runcli_test.go:103,169`, which build marker
      filenames (`restart-pair-work`). **Under a dual-prefix transition, which
      expectations flip to `📁` and which deliberately stay `pair-` is a per-file
      judgment — and the ones that stay ARE the transition coverage.** So this
      gets a decision pass, not a sed. Out of scope and must not be touched:
      `entrypoint/alias_test.go` (10 literals — busybox binary names).
      Also in scope: the shell suite, where `tests/pair-rename.sh:174-178` (T7,
      "accepts `pair-<tag>` form") pins the legacy strip and should keep passing
      as intentional transition coverage. Two more shell files hardcode the
      legacy spelling and are **scheme-agnostic exact round-trips, so they keep
      passing untouched** — recorded so they are not rediscovered mid-sweep:
      `tests/pair-restart-quit-test.sh:15` (`ZELLIJ_SESSION_NAME="pair-smoke"`,
      whose marker assertions at `:23` derive from it) and
      `tests/workbench-route-nvim-test.sh:43,64,97`.
- [ ] `atlas/session-identity.md` — it documents the `pair-<repo>-<tag>` scheme
      and the numeric-suffix rule verbatim; update to the new format, the 24-byte
      budget, and the transition. `atlas/architecture.md` gets the cross-link.
      Record one invariant that is new and easy to miss: a session name is also a
      **filename component** — `~/.cache/pair/quit-<session>` and
      `restart-<session>` (`osruntime.go:544,553,557,569,573`). `📁` is safe on
      APFS/ext4 and carries no shell or glob metacharacters, but that is now a
      property the scheme depends on.
- [ ] Live check — **use a fresh tag**, since this repo's `pair` tag almost
      certainly has a `pair-pair-pair` ledger row and will correctly grandfather
      until that session is gone. Create a session in this repo (expect `📁pair`
      `pair list`, the picker, `pair resume`, and `Ctrl+Alt+n` rename all work
      against it; confirm an existing `pair-*` session is still discovered and
      attachable; confirm the cmux workspace title.

## Log

### 2026-07-29

- Filed from the tab-title investigation. **Independent of** the zellij upstream
  question (zellij-org/zellij#1495, open since 2022, asks for a config to stop
  zellij prefixing the title with the session name): this change improves
  `pair list`, cmux and `zellij list-sessions` whether or not that ever lands.
  Related but separate: **#129** (pane titles carry cwd + role), which owns the
  half of the tab title pair already controls.
- Empirical findings behind the spec: zellij's limit is **24 bytes, not
  characters** (`🚧-` + 22 chars = 27 bytes rejected; 21 chars = 24 bytes
  accepted). Pane titles, by contrast, are unconstrained — a 70-character title
  with `/`, spaces, `[]`, `·` and an em-dash was accepted verbatim. That
  asymmetry is why expressive text belongs in the pane title and identifiers in
  the session name.
- `📁` was chosen over `🚧` after testing both: same 4-byte cost, but dropping
  the separator saves a byte and the folder glyph matches what the segment means.

## Revisions

### 2026-07-29T11:12 — plan corrected before implementation; estimate added

**Reason.** Reading every `pair-` site before writing code (the pre-flight the
plan's own "decide per site" item asks for) contradicted two of the plan's
assumptions. Recording the delta rather than overwriting.

**Delta.**

1. **"All four filters share one predicate" was wrong for two of the four.**
   `zellij.go:27` and `zellijparse.go:60` ask *"is this name pair's?"* and do
   share the dual-prefix predicate. `legacy_live.go:19` and `lifecycle.go:158`
   filter *and then* `TrimPrefix` to recover the tag — and the new format is
   deliberately **not** invertible (rules 2 and 4 discard information:
   `📁parley-nvim` strips to `parley-nvim`, not the tag `parley_nvim`). Both
   already prefer `SessionNameIndex` and reach the strip only for unindexed
   sessions, which every `📁` session is not. Teaching them `📁` would mint
   plausible-looking wrong tags. Split into two plan items so the distinction is
   explicit rather than discovered mid-edit.
2. **The constructor inventory was 5 sites; it is 13.** Added the writers
   `titlepoller/run.go:79`, `titlepoller/run.go:216`, `createflow.go:496` and the
   readers `continuationcmd/draft.go:108`, `tag.go:11`, `pick.go:86`,
   `list.go:87`, `restart.go:22` — the sweep crosses `titlepoller` and
   `continuationcmd`, not just `launcher`. Also recorded the one **false**
   positive: `entrypoint/alias.go:34,37` strips `pair-` from *busybox binary
   names*, and a blind sweep would break the `pair-slug` Stop-hook symlink.
3. **Named the actual destructive chain.** `SessionBlocksReuse` is per-name, not
   a sweep, so the hazard is narrower and more specific than "deletes foreign
   sessions": `AssignSessionName`'s collision check consults `live`, which is
   *already* filtered by the ownership predicate — so a foreign session is
   invisible to collision avoidance, and only a wrongly-widened predicate turns
   that into a force-delete. The test now covers both the predicate and the
   end-to-end path.
4. **`## Estimate` added** (1.73h, estimate-logic-v3.1) — required by the
   `change-code` gate and absent until now.

### 2026-07-29T11:30 — plan-quality round 1: 5 blocking findings, all accepted

**Reason.** `sdlc change-code` returned VERDICT: FAILURE. Every finding was
checked against the code before acting; all five blocking ones reproduced.

**Delta.**

1. **The 20-byte budget was a machine-specific constant replacing a runtime
   probe (`ARCH-MOCK`).** zellij's allowance is the socket budget minus the
   platform cache path, which varies by username and is a different path on
   Linux. The codebase deliberately treats the external binary as the oracle
   (`ProbeSessionName` + `sessionNameRejected`, injected as `accepts`), and the
   plan was about to replace that with a const. Now: probe stays the oracle, the
   number is calibrated and passed as a **parameter**, 20 is only a fallback and
   a message. Also teaches `fakeRuntime.ProbeSessionName` a length model — it
   returns a constant today, so refuse-vs-accept was untestable end-to-end.
2. **The collision suffix was not reserved, and truncate-vs-refuse did not
   compose.** `publicSessionName` appends `-N` up to 100 (4 bytes) and the
   20-byte figure covered only `{repo}[-{residual}]`; meanwhile Spec item 6,
   plan item 2 and plan item 3 gave three different answers on whether the
   ladder survives. Now stated once: prompt refuses on the interactive path,
   ladder survives for the non-interactive paths, `SessionNameExhausted` is the
   floor. Suffix width is reserved.
3. **The refusal had no seam.** `promptForTag(rt, prefill, base, stderr)` has no
   scope and no cwd; the repo name only arrives at `assignSingleSessionName`,
   *after* the prompt returns. The signature change is now an explicit part of
   the item instead of a surprise mid-edit.
4. **Three more strip→tag sites were wrongly covered by the "leave it" rule.**
   `restart.go:22`, `pick.go:86` and `list.go:86` `TrimPrefix` with no index in
   reach, so each would mint the same plausible-looking wrong tag the plan
   refuses to create at `legacy_live.go`. Split into their own item, each getting
   an index lookup and an empty-on-miss return.
5. **`EmojiTitle` was decided in the Spec and scheduled nowhere
   (`ARCH-PURPOSE`).** Put to the operator, who chose **retire it** — delete
   `titlefmt.EmojiTitle` + `emojiWords`, the `launcher.EmojiTitle` shim and
   `TestEmojiTitle`; both call sites pass the name through verbatim. (The Spec
   says one call site; there are two — `osruntime.go:419` and
   `titlepoller/titlepoller.go:85`.)

Non-blocking findings also folded in: `PublicSessionBase` (`session_index.go:56`)
is dead exported code hardcoding `"pair-"` — deleted, making the inventory 14
sites; the new composer is named `ComposeSessionName` (not `PublicSessionName`)
because `publicSessionName` already exists one case-flip away, and that one is
renamed `withCollisionSuffix`; `NormalizeTag` gets a `TagForSessionName` ledger
resolution so pasting a `📁` name into `pair resume` keeps working; and a mixed
legacy/new snapshot is now pinned by a table test rather than resting on the
manual PTY check.

**Estimate revised 1.73 → 2.54h.** The judge called 1.73 optimistic; the round
then grew the plan from 8 items to 12, which is the honest cause.

### 2026-07-29T11:55 — plan-quality round 2: 3 blocking findings, all verified and accepted

**Reason.** Second `sdlc change-code` round. Every finding was checked against
the tree before acting; all three blocking ones reproduced.

**Delta.**

1. **The scheme has a fourth consumer language, and it was invisible to a
   Go-only sweep (`ARCH-DRY`, `ARCH-PURPOSE`).** `nvim/init.lua:3266,3271`
   carries a Lua twin of both the `pair-` strip and `NormalizeTag`'s charset
   check. It sits on a Done-when clause, and this change makes the regression
   *more* likely: the tab title will read `📁pair`, so pasting it into the rename
   prompt is the natural move and is rejected today. Resolved by deleting the
   Lua-side rules and letting `pair rename --restart-check` — already shelled to
   at `:3274` — own resolution. `nvim/init.lua:35` and
   `nvim/workbench_route.lua:61` were checked and are scheme-agnostic (exact
   round-trip only); recorded so the next reader doesn't re-audit them.
2. **The destructive-path test asserted on a seam that cannot observe the
   destruction (`ARCH-MOCK`).** The force-delete lives *inside*
   `OSRuntime.SessionBlocksReuse` (`osruntime.go:71`), below the Runtime seam,
   and the fake is a bare map lookup with no call log. The planned assertion
   would have passed while the hazard sat untouched — the exact theater the
   `## Plan`'s own test discipline exists to prevent. Fix: lift the
   `delete-session --force` into the existing `DeleteSession` seam (already
   recorded as `rt.deleted`) so the act is observable by construction, and give
   the fake the state transition rather than a constant.
3. **The existing test corpus was unplanned and unestimated.** ~300 `"pair-…"`
   literals across ~39 files, plus `session_index_test.go:75` asserting the
   prefix outright and `runcli_test.go:103,169` building marker filenames. The
   sharp part: under a dual-prefix transition, *which expectations flip and which
   deliberately stay `pair-` is a per-file design decision, and the ones that stay
   ARE the transition coverage* — so this is a decision pass, not a sed. Now its
   own item with its own estimate row, and it names `entrypoint/alias_test.go` as
   out of scope (busybox binary names) and `tests/pair-rename.sh:174-178` as
   intentional legacy coverage.

Important findings also folded in: the ladder's replacement contract is now
written out in full (signature, order of sacrifice — whole residual tokens before
any byte truncation, so no second `parley_nv` is possible — the `📁`+4-byte floor,
and why the suffix needs no fixed reservation once candidates are measured
suffixed); budget calibration became **lazy, on rejection only**, since `pair`
create is a one-shot process and a "per-process cache" would just mean N extra
`list-clients` execs before every prompt; the third refusal message at
`createflow.go:334-337` is now explicitly the non-interactive floor rather than a
fourth accidental spelling; and the `ARCH-MOCK` conformance question is answered
as a stated decision (the runtime probe stays the sole oracle, so the fake's
length model only needs self-consistency).

Non-blocking: the filename-component invariant (`quit-<session>`,
`restart-<session>`) is recorded for the atlas, and the `list.go:86` / `:87`
citation inconsistency is fixed.

**Advisory NOT acted on — flagged to the operator instead.** The judge observed
that rule 2 discards repo information unconditionally, even when the budget has
room: `NormalizeDisplayComponent` preserves `-`, so `claude-code` + tag `work`
gives `📁claude-work` (15 bytes) when `📁claude-code-work` (20) would fit.
Hyphenated repo names are far more common than dotted ones, and the Spec's
trade-off paragraph reasons only from `parley.nvim`. "Full normalized repo when
it fits, first token under pressure" would also give the ladder something
meaningful to shorten. **Rule 2 is an operator-fixed decision, so it stays as
specced**; recording the observation here rather than silently re-opening it.

**Estimate revised 2.54 → 3.34h.** Two rounds have taken the plan from 8 items to
16; the two largest new rows (the 15-site sweep, the ~300-literal re-baseline)
are also the two least mechanical.

### 2026-07-29T12:20 — plan-quality round 3: one blocking regression, caught before it shipped

**Reason.** Third `sdlc change-code` round. All findings verified against the tree
first; every one reproduced. This round earned its cost — the blocking finding is
a regression the plan as written would have shipped.

**Delta.**

1. **The `TagForSessionName` seam was wired to the wrong call site, and the Lua
   item depended on it (`ARCH-PURPOSE` shadow-sweep).** `runcli.go:106` is *not*
   a resume entry point — it is `NormalizeTag(args.ContinueSlug)` for `pair
   continue <slug>`, a continuation-doc slug resolved by
   `rt.ResolveContinuationDoc` (`:111`), never a session name. Meanwhile the site
   that genuinely strips `pair-` from a pasted name — `validateRenameTags`
   (`rename.go:61-67`) — was omitted, and it is *exactly* what the previous
   round's Lua item hands the job to (`nvim/init.lua:3274` shells to `pair rename
   --restart-check`). Net effect of the two items as written: delete the Lua
   strip, leave the binary on bare `NormalizeTag`, and rename ends up **worse
   than today** — the `pair-work` paste that currently succeeds would start
   failing alongside the `📁` one, on a Done-when clause. Corrected site list is
   now `args.go:91` (resume) and `rename.go:62` (rename **old** only, since the
   new tag names a session that does not exist yet and has nothing to resolve),
   with an explicit ordering constraint tying the Lua deletion to the binary fix.
2. **Two more user-facing minters of the legacy format (`ARCH-DRY`).**
   `pick.go:117` spells `"pair-%s  (%s, no live session)"` for historical rows —
   so the picker would show `📁pair-work` live and `pair-work` historical for one
   tag — and `rename.go:170`/`:175` spell `'pair-%s'` in the tracked/exists
   errors. Both are the user-facing halves of sites already listed.
3. **"Return empty on index miss" was wrong for one of the three sites.**
   `sessionTag` (`pick.go:86`) feeds the live-dedup key `live[sessionTag(s)]`
   (`:38`) and `pickSelection{tag: …}` (`:55`); empty would collide every
   unresolved row into one bucket and make selection resolve to no tag. Now
   per-site: empty for `list.go:86` and `restart.go:22` (both feed only
   `inferAgent`), fall back to `s.Name` for `pick.go`.

Also folded in: `decision.go:59` + `createflow.go:473` are re-classified from
"writer that must move" to **legacy degraded fallback that stays `pair-`** —
`SessionSnapshot` carries no `RepoScope`, so there is nothing to compose from,
and both fire only when `ResolveRepoScope` fails while
`assignLaunchSessionNames` already pre-assigns composed names on the normal path
(`createflow.go:473` was missing from the inventory entirely). New item for
`pair list` **display-width** padding: `formatListTable` uses `%-30s`, which Go
counts in runes, and `📁` is 1 rune but 2 terminal columns — the same unit
confusion as the rune-vs-byte bug one layer up, and on a Done-when clause. And
the lazy budget calibration must probe **synthetic pad names**, since
`ProbeSessionName` succeeds against a foreign live session and would otherwise
read as "fits" for the wrong reason.

**Estimate revised 3.34 → 3.60h.**

### 2026-07-29T12:50 — plan-quality round 4: the headline outcome was unreachable

**Reason.** Fourth `sdlc change-code` round. All findings verified against the
tree first. Rounds 1–3 hardened the *sweep*; this one found that the *core naming
path* had two acceptance criteria that could not be satisfied as written.

**Delta.**

1. **Done-when bullet 1 was unreachable — `AssignSessionName` never gets to the
   new format for an existing tag.** `session_index.go:98-100` short-circuits on
   `index.latestFor` whenever `accepts(prior.SessionName)`, and `accepts` only
   asks whether zellij tolerates the name's *length*, not whether a session
   exists. So any tag with a ledger row would keep `pair-…` permanently — and in
   *this* repo, tag `pair` is exactly the tag most likely to have a
   `pair-pair-pair` row, so the plan's own live check would have failed first,
   for a reason unrelated to the change. Worse, bullet 3 ("historical rows still
   resumable") depends on that same short-circuit *not* changing, so the two
   criteria were in direct tension and the plan never picked.
   **Decision, now specced in *Ledger migration*: grandfather iff the prior name
   is present in `live` and not `EXITED`.** Live/detached legacy sessions keep
   their names (renaming a running zellij session underneath itself isn't
   possible anyway), an `EXITED` row re-mints on the next create. `live` is
   already a pair-filtered parameter carrying state, so this needs no new IO and
   stays pure. Concretely: `pair-pair-pair` keeps working, and the first `pair`
   after quitting it mints `📁pair`. Pure grandfathering was rejected — simpler,
   but it never delivers the issue's headline outcome without a manual ledger
   edit. Done-when bullet 1 reworded to match; the live check now uses a fresh
   tag.
2. **Done-when bullet 4's test was not constructible.** It promised "a test with
   a multi-byte *component*", but `NormalizeDisplayComponent` (`scope.go:44-59`)
   maps every rune outside `[A-Za-z0-9_-]` to `_`, so both components are ASCII
   by construction — the `📁` prefix is the only multi-byte element in the whole
   scheme, and the Spec's premise ("any non-ASCII component breaks that
   agreement") is false given the normalizer. A test hand-feeding a multi-byte
   repo into the ladder would pin an unreachable state. Restated as what actually
   regresses: the prefix costs 4 bytes / 1 rune, and a candidate measuring 21
   runes but 24 bytes must be handled by its byte length.
3. **The ladder's `limit` parameter contradicted the calibration design
   (`ARCH-MOCK`).** Item 2 passed `limit` in; item 3 said the probe is the oracle
   and calibration happens lazily *on rejection*. A caller would need `limit`
   before any rejection existed, leaving only the macOS-derived 20 — which on
   Linux would drop residual tokens zellij would have accepted, reintroducing
   unexplained shortening in the function this issue exists to fix, on the
   platform we cannot test. Sharpest on the collision path, where suffixes ≥2 are
   driven by collisions rather than rejections so calibration never runs at all.
   **`limit` dropped from the ladder**; it emits candidates longest-first and
   `AssignSessionName` filters through `accepts` as today. `limit` survives only
   in the prompt's refusal message.

Also folded in: `sessionNameParts(scope, tag) (repo, residual)` named as the
single tokenizer with one `sessionPrefix` const, so deleting the
`PublicSessionBase` duplicate doesn't quietly create a new one (`ARCH-DRY`); two
more `TestComposeSessionName` rows for the cases where rule 4 is genuinely
ambiguous (repo token appearing *non-leading* in the tag, and case-sensitivity,
since `NormalizeDisplayComponent` preserves case); and one sentence in *Accepted
trade-offs* recording that whole-token dropping widens the `📁pair-1` ambiguity
to cases like `📁parley-2`.

**Estimate revised 3.60 → 3.99h** (new ledger-migration row; re-baseline row
raised 0.30 → 0.40 impl). The judge expects ~4.5h; that is recorded in the
plausibility note rather than back-fitted into the derivation.

### 2026-07-29T13:20 — plan-quality round 5: round 4's migration fix had two defects of its own

**Reason.** Fifth `sdlc change-code` round. Both blocking findings are
consequences of round 4's ledger-migration decision — the newest and
least-reviewed part of the plan — which is the normal shape of convergence rather
than the gate re-litigating settled ground. All findings verified against the
tree first.

**Delta.**

1. **The liveness gate would have appended a duplicate ledger row on every
   create (`ARCH-PURPOSE`).** Today the short-circuit means the append at
   `session_index.go:106-113` never runs for a known (scope, tag). Gate it on
   liveness *alone* and the second create for a tag whose session is gone falls
   through, recomposes the same `📁pair`, passes both `ownedByOther` (same scope +
   tag) and `liveOwnedByOther` (not live), and appends an identical row — which
   `createflow.go:396-401` persists, once per create, forever. Round 4's test
   list (a)–(d) would have passed the whole time. Condition corrected to
   **(live and not `EXITED`) OR already carries `sessionPrefix`**, so a legacy
   name re-mints exactly once while new-format names stay pinned as they are
   today, plus a skip-the-append guard. New test (e): two consecutive creates for
   one tag with no live session yield exactly **one** new row.
2. **Nothing would ever reclaim the superseded `EXITED` zellij record.**
   `delete-session --force` fires only against a name pair is about to reuse
   (`OSRuntime.SessionBlocksReuse`) or the session that just exited
   (`lifecycle.go:66`) — and after graduation nothing ever names `pair-pair-pair`
   again. So the `EXITED` residue survives in `zellij list-sessions`, and because
   `index.ownerOf` still resolves it to this scope, `SessionsForScope` keeps
   feeding it to `pair list` as a permanent second `status: exited` row for the
   same tag. Round 4's "No orphaning" claim was true of the *name binding*, not
   the zellij record. Re-minting now deletes the superseded session through the
   `DeleteSession` seam the destructive-path item is already creating — new test
   (f) — and the mixed-snapshot test asserts a re-minted tag shows exactly one
   row.

Non-blocking corrections folded in, two of which were **wrong reasons behind
right verdicts** — worth recording, because a right answer resting on a false
premise is one refactor away from becoming a wrong one:

- `decision.go:59` does **not** fire only when `ResolveRepoScope` fails.
  `sessionNameForTag` falls back whenever the tag is absent from
  `snap.SessionNames`, and `assignLaunchSessionNames` populates only the single
  launch tag — so `nextFreeTag`'s `base-2`, `base-3`, … probes (`decision.go:97`)
  reach it on the normal path with the scope resolved fine. Still safe to leave
  `pair-`, but because the name there is only ever *compared*, never minted. Adds
  a `nextFreeTag` test row against a live `📁<repo>` session.
- `restart.go:22` does **not** feed only `inferAgent` — the tag also becomes
  `RestartMarker{Tag: …}` (`:24-25`). Empty is still safe, but because
  `createflow.go:83` does `firstNonEmpty(m.Tag, step.tag)`. Now pinned by test
  rather than assumed.
- Budget calibration bounds are stated: search 1..64 bytes → ~6 sequential
  `list-clients` execs, accepted only on the already-failing path.
- Two more shell files hardcode the legacy spelling and are scheme-agnostic
  exact round-trips that keep passing untouched:
  `tests/pair-restart-quit-test.sh:15` and
  `tests/workbench-route-nvim-test.sh:43,64,97`.

**Estimate re-derived 3.99 → 4.22h** (ledger-migration row 0.06/0.16 → 0.08/0.28
for the dup-row guard, the record reclaim and tests (e)/(f); mixed-snapshot row
+0.02 impl). Round 4's judge expected ~4.5h; five rounds of derived growth
reached 4.22h without back-fitting.

### 2026-07-29T13:50 — plan-quality round 6: findings dropped to 1 Important + 2 Minor; gate closed deliberately

**Reason.** Sixth `sdlc change-code` round. The judge independently re-ran the
sweep — "a tree-wide grep for `"pair-"` in Go returns exactly the sites the plan
inventories, with no unlisted one" — and returned no Critical/Blocking design
defect for the first time: one Important placement question and two Minor factual
corrections. All three verified and fixed below.

**Delta.**

1. **(Important) The reclaim-delete named no call site, and the natural one fires
   before commitment (`ARCH-PURE`).** `AssignSessionName` is pure, so the IO lives
   in a caller — and the two callers are not interchangeable.
   `assignLaunchSessionNames` runs at `createflow.go:154`, *before* `DecideLaunch`
   (`:159`), the picker (`:169`) and `promptForTag` (`:315`), on **every** launch
   invocation including ones that resolve to attach or that the user ESCs out of.
   A reclaim there would force-delete a resurrectable `EXITED` record on an
   abandoned `pair` — strictly more destructive than today, where deletion is
   gated behind commitment (`SessionBlocksReuse` at `:341`, post-prompt). The
   plan had already solved this shape for the ledger *write*, so the reclaim now
   rides the same seam: `AssignSessionName` gains a second return naming the
   superseded session, and the `DeleteSession` call sits beside
   `AppendSessionNameIndex` at `:396-401`. New test (g): an attach or an aborted
   invocation reaches `DeleteSession` **not at all**.
2. **(Minor) `createflow.go:504` was an unlisted fourth `NormalizeTag` site** —
   inside `promptForTag`, the very function the refuse-early item rewrites. The
   item claimed "exactly two"; it now decides all four. It stays bare, for the
   same reason the rename *new* tag does: the value names a session that does not
   exist yet.
3. **(Minor) "EmojiTitle is unreachable in production" was overstated.**
   `CmuxRename` is also called with a cwd basename at `lifecycle.go:130`
   (`reset := filepath.Base(env.Cwd)`), reaching `EmojiTitle` at
   `osruntime.go:419`, so a hyphenated cwd like `my-brain` still maps today. That
   path is the quit-time workspace reset, not a session title; retiring the map
   remains the operator's decision, but the justification is corrected so the
   close review does not re-derive it and land somewhere else.

**Estimate re-derived 4.22 → 4.31h** (reclaim plumbing: the extra return, the
caller stash, the commit-point wiring and test (g)).

**Gate closed with `--no-judge`, deliberately.** Six rounds is the cost ceiling
the operator flagged (ariadne#187: the judge is stateless, so plans converge by
exhaustion rather than agreement). Rounds 1–5 each found verified defects,
including two that would have shipped — a rename regression and an unbounded
ledger append — so the rounds earned their cost. Round 6 produced no blocking
design finding and its three items are fixed above, which is exactly the
"findings dropped to Minor/Advisory" condition for using the flag. Recording it
here rather than letting a seventh round re-derive the same plan.

## Log (implementation)

### 2026-07-29 — pure core landed

- `sessionPrefix`/`legacySessionPrefix` consts, `isPairSessionName`,
  `sessionNameParts`, `ComposeSessionName`, `alnumTokens` added to
  `session_index.go`; `PublicSessionBase` deleted (zero callers, confirmed
  tree-wide). `session_name_scheme_test.go` covers the predicate (including a
  dedicated foreign-session rejection test), the four spec rows, and the
  interaction cases. All green; `go build ./...` clean.
- **Spec rule 4 needed a reading decided at the keyboard.** "Drop the tag's
  leading tokens that match the repo's" is ambiguous between *drop the leading
  run* and *drop a one-token prefix*. My first implementation dropped the run and
  a test caught it. **Decided: drop exactly one**, because the repo side is a
  single token (rule 2) so the spec's prefix is one token long — and because
  dropping the run folds distinct tags onto one name: tag `pair-pair-x` and tag
  `pair-x` would both compose to `📁pair-x`, and `ownedByOther` would resolve the
  collision with an opaque numeric suffix. Pinned by `TestSessionNameParts`.
- Pre-existing, unrelated: `wrapcmd.TestSIGUSR2ReExecsWrapperWithoutReplacingPaneProcess`
  fails identically on a stashed clean tree — the sandbox PTY limitation noted in
  the session lessons, not a regression from this change.

### 2026-07-29T14:20 — operator decision: full migration, no grandfathering

**Reason.** Operator chose to migrate everything in one deliberate event from a
quiesced state — concluding all other sessions and clearing the other zellij
sessions — rather than have the code carry a grandfather rule indefinitely.

**Delta.**

- *Ledger migration* rewritten. The short-circuit condition drops its liveness
  clause entirely: **short-circuit iff the prior name already carries
  `sessionPrefix`.** A legacy row always falls through and re-mints. This is
  strictly simpler than the round-4/5 design — no liveness input, two fewer test
  cases — and the prefix clause still carries the anti-unbounded-append duty
  round 5 identified.
- Reclaim gains one guard: **never reclaim a `SessionAttached` session.**
  Force-deleting a session with a live client would kill a terminal out from
  under someone. Detached and `EXITED` both reclaim.
- New Spec section, *The session running this work migrates last*: pair cannot
  rename a live zellij session underneath itself, so the hosting session migrates
  only by being quit and relaunched. That makes the last step an operator
  handoff, sequenced as: verify against a fresh tag first, then quit/relaunch
  this one — with a continuation written beforehand so the work survives the
  restart, and survives the restart failing.
- Done-when bullets updated: every legacy name migrates; a migrated tag shows one
  `pair list` row, not a permanent second `exited` one; the hosting session comes
  back as `📁pair`.

**Implementation-order hazard found immediately, by landing the hunk and running
the tests.** Applied on its own it turns `TestAssignSessionNameReusesSameScopeBinding`
red — because with the ladder still emitting `pair-…`, falling through re-mints
another *legacy* name and appends a duplicate row per create. Six plan-quality
rounds named the defect in its designed form but not this intermediate-state
form. The hunk is reverted with a comment recording the constraint, and the plan
item now states it: land it with the ladder rewrite, never before.

**Estimate unchanged at 4.31h.** The code gets simpler (no liveness clause, two
fewer cases) while the operator runbook and the attached-guard add roughly the
same back.
