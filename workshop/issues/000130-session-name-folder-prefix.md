---
id: 000130
status: working
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours:
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

### Why the ownership prefix cannot simply be dropped

`session_blocks_reuse` calls `zellij delete-session --force` on `EXITED` rows.
Without a prefix distinguishing pair's sessions from foreign ones, a stranger's
abandoned session whose name matched a tag would become a deletion target. The
global list already contains foreign names (`fabulous-aardvark` was present while
scoping this). The prefix is load-bearing; only its cost is negotiable.

## Done when

- `pair-pair-pair` → `📁pair`; `pair-parley_nvim-parley_nvim` → `📁parley-nvim`,
  untruncated.
- A tag that would overflow 20 bytes is refused at the name prompt with the real
  limit quoted, not silently shortened.
- Existing `pair-*` sessions — live, detached, and historical ledger rows — are
  still discovered, attachable, and resumable after the change.
- Truncation logic is byte-denominated; a test with a multi-byte component proves
  it (this fails today).
- `pair list`, the picker, `pair resume`, rename, and cmux ownership all work
  against a `📁` session.

## Plan

Single review boundary — no `Mx` tags.

- [ ] **`PublicSessionName(repoDisplay, tag) string`** — one pure function
      implementing rules 1–4, living beside the existing scheme in
      `launcher/session_index.go`. Composes on top of the existing
      `NormalizeDisplayComponent` (`scope.go:44`) rather than re-deriving
      sanitisation: that already maps `parley.nvim` → `parley_nvim`, so rule 2 is
      "first alphanumeric run of the normalised value" (`ARCH-DRY`).
      Test `TestPublicSessionName` — table over the four spec rows, plus the
      cases where the rules interact: tag equal to repo, tag prefixed by repo,
      tag unrelated, repo with no alphanumerics (normaliser's `"pair"`
      fallback), empty tag.
- [ ] **Byte-based truncation.** `BuildSessionNameCandidates` (`:60`) shortens
      `[]rune` against a byte budget. Convert to bytes. This is a prerequisite,
      not a cleanup: `📁` is 1 rune / 4 bytes, so every candidate would otherwise
      be 3 bytes longer than the generator believes.
      Test: a candidate list for a multi-byte component — fails today.
- [ ] **Refuse early rather than truncate.** With `📁` the budget is 20 bytes.
      When the composed name overflows, the create flow's name prompt refuses and
      quotes the real limit instead of silently shortening.
      Test: the pure decision (name → over/under budget + message), driven
      directly; the prompt loop is the thin IO shell around it (`ARCH-PURE`).
- [ ] **Dual-prefix discovery.** All four filters must accept `pair-` **and**
      `📁` while only `📁` is ever emitted: `zellij.go:27`, `zellijparse.go:60`,
      `legacy_live.go:19`, `lifecycle.go:158`. One predicate
      (`isPairSessionName`) shared by all four, not four edited literals — four
      independent copies of one rule is the divergence shape that caused #127.
      Test: the predicate over `pair-x`, `📁x`, `fabulous-aardvark`, `""`.
- [ ] **Sweep the other `"pair-"` constructors.** `compaction.go:36,73`,
      `decision.go:60`, `lifecycle.go:23`, `rename.go:142` each build
      `"pair-"+tag` as a legacy/fallback shape. Decide per site whether it is a
      *legacy reader* (leave, it is reading old names) or a *writer* (must move).
      Getting this wrong is silent: a stale writer mints a `pair-` name that the
      new scheme never expects.
- [ ] **Verify the destructive path specifically.** `session_blocks_reuse` calls
      `delete-session --force` on `EXITED` rows. Prove with a test that a foreign
      session (no pair prefix of either form) is never a deletion candidate —
      this is the one place where getting the predicate wrong destroys someone
      else's state rather than just confusing pair.
- [ ] `atlas/session-identity.md` — it documents the `pair-<repo>-<tag>` scheme
      and the numeric-suffix rule verbatim; update to the new format, the 24-byte
      budget, and the transition. `atlas/architecture.md` gets the cross-link.
- [ ] Live check: create a session in this repo (expect `📁pair`), confirm
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
