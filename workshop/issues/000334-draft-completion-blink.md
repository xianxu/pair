---
id: 000334
status: open
deps: []
github_issue:
created: 2026-09-27
updated: 2026-09-27
estimate_hours:
card_mirror: 'a9b9e295ce372d2dafe98dfd605236fd41642541' # card fields mirrored from issue-cards; edit via sdlc
---

# draft nvim: audit as-you-type completion; evaluate blink.cmp

## Problem

The draft nvim (`nvim/init.lua`) does as-you-type completion by hand, with no
plugins: ~350 lines across `path_complete` (~L1667), `word_complete` (~L1840),
`spell_suggest_popup` (z=, ~L1919), the `TextChangedI`/`TextChangedP`
dispatcher (~L3890), and helpers (`plain_items`/`indexed_items`,
`spell_popup_active`, `spell_pick_digit`, `<M-1>..<M-9>` quick-pick). It feeds
`vim.fn.complete()` directly and filters with `matchfuzzy` (paths) or
prefix-anchored scoring (words).

parley.nvim is adopting blink.cmp (app cmdline completion, then moving the
plugin's own completion from nvim-cmp to blink, including spelling). This is
a good time to check whether pair's hand-rolled completers should move to
blink sources, or stay as they are.

## Spec

Audit first, decide second. There is no commitment to migrate.

1. **Inventory** each completer: its trigger, candidate source, matching and
   ranking, key handling, and edge cases it deliberately handles. Worth
   keeping:
   - path vs. word are mutually exclusive at trigger time (explicit `/`, `~`,
     `./` prefix);
   - word candidates include **agent-output spans** from
     `$PAIR_AGENT_OUTPUT_PATH`, filtered by SGR color and ranked by
     `(count + α·picks) · 0.5^(rank/H)` with a `POOL_CAP`;
   - words are matched **by prefix on purpose**: fuzzy matching was tried and
     rejected for false positives (`tel` → anything containing t…e…l);
   - z= spell popup: numbered digit-pick, returns to normal mode on
     `CompleteDone`;
   - avoiding feedkeys reentrancy (the reason `complete()` is called directly).
2. **Map onto blink:** could each completer be a custom blink source
   (`get_completions`), with its scoring kept through a custom `score_offset`
   / sort, and with prefix-only matching per source? Does blink's own `buffer`
   / `path` / a spell source replace any of them outright? Can the z=
   digit-pick UX be kept?
3. **Costs:** the draft nvim is currently plugin-free. blink would add a
   plugin (plus the `lua` fuzzy implementation, to avoid the binary download)
   and a way to install it in pair's nvim profile. Weigh that against
   per-keystroke performance, maintenance of ~350 lines, and consistency with
   parley.
4. **Shared piece:** could pair's agent-span source ship as a blink source
   that parley could reuse too, or the other way round?

## Done when

- A findings section in this issue: the per-completer inventory, the blink
  mapping (replace / wrap as a source / keep), and a recommendation (migrate,
  partial migration, or stay plugin-free) with reasons.
- If the recommendation is to migrate: a follow-up issue filed with the
  concrete migration plan. This issue does not change code.

## Plan

- [ ] Read and inventory the completers in `nvim/init.lua`
- [ ] Prototype one completer (e.g. agent spans) as a blink source in a scratch profile
- [ ] Write findings + recommendation; file a follow-up if migrating

## Log

### 2026-09-27
- Filed alongside parley.nvim issues for blink cmdline completion (app) and
  moving the plugin from nvim-cmp to blink (paths + spelling).
