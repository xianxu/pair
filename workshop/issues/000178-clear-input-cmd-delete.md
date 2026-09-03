---
id: 000178
status: wontfix
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Clear a text input with cmd+delete

## Problem

Emptying a text input means holding backspace. Every text-entry surface in the
workbench should clear on one gesture.

**cmd+delete already reaches pair — pair discards it.** The operator's Ghostty
config maps `super+backspace` to `text:\x15`, so the terminal sends `0x15`
(`^U`). `couchtty/panelkeys.go` decodes `\r\n`→Enter, `\t`→Tab,
`0x7f`/`0x08`→Backspace, and `0x20..0x7e`→Rune; `0x15` falls through `default:`
and is dropped. So the key is arriving and being thrown away.

**The pair-side contract is therefore `^U`, not "cmd".** Nothing below the
terminal knows about a Command key — Ghostty's binding is the operator's, and a
different terminal or a different config sends something else. Implement `0x15`
and cmd+delete works as a consequence; implement "cmd+delete" and there is
nothing to implement.

**The panel has four separate text handlers.** `KeyRune`/`KeyBackspace` are
handled at `couchtty/menu.go:364`, `:494`, `:549`, and `:596` (the last inside
`reduceTextKey`). Adding clear-all is four edits, and three of them are the ones
that get forgotten — a value cleared in the name field but not the describe
field is worse than the feature's absence, because it teaches a gesture that
sometimes silently does nothing.

## Spec

**`^U` (`0x15`) clears the text input that has focus.**

1. **Consolidate first, then add.** Fold the four rune/backspace handlers into
   one text-editing reducer and give the new key one home (`ARCH-DRY`). Adding
   a fifth case in four places is how the next input key gets forgotten too.
   The reducer is pure, so this is testable without a terminal (`ARCH-PURE`).
2. **Decode `0x15` in `panelkeys.go`** as a `KeyClear` kind rather than
   special-casing a byte inside the reducer — the decoder is where every other
   control byte is named.
3. **Enumerate the surfaces; do not fix the one that prompted this.** At
   minimum: couch's panel edit boxes (name, describe, path/start-args, and the
   switcher typeahead), and the nvim draft pane. The `## Done when` list must
   name every surface found, and a surface deliberately excluded must say why.

**Draft pane, two specifics:**

- `^U` in nvim insert mode is already bound: it deletes text entered in the
  current insert session (bounded by `'backspace'`). Making it clear the whole
  draft **overrides a standard vim binding**, which is a deliberate choice, not
  an omission to fix silently.
- Clearing a multi-line composed prompt is the most destructive instance of
  this gesture. It must be a single `u` undo away, and that should be asserted.

## Relationship to `pair#165`

`#165` already specifies `Alt+Delete` in couch edit boxes: clear all in a
single-line box, clear only the **current line** in a multiline box. This issue
specifies clear-**all** everywhere. Two chords, overlapping scope, different
multiline semantics — whoever implements the second one will otherwise find the
first already there and guess.

**Recommendation: keep both, with the split stated in both issues.** `^U` =
clear the whole input; `Alt+Delete` = clear the current line in a multiline box,
which is a genuinely different and useful operation. They should share the
consolidated reducer from step 1. Settle this before either lands, and record
the decision in both files.

## Done when

- `^U` clears the focused input in every surface named in the enumeration,
  each asserted.
- cmd+delete clears the input in Ghostty with the operator's binding — an
  end-to-end check that the byte survives couch's interceptor and reaches the
  focused editor.
- The four `menu.go` text handlers are one; a shadow sweep confirms no second
  rune/backspace path survives.
- `0x15` is named in the decoder, not matched raw inside a reducer.
- In the draft, `^U` clears and a single `u` restores the full prior text.
- `#165`'s `Alt+Delete` semantics are reconciled and recorded in both issues.

## Plan

- [ ] Enumerate every text-entry surface; record the list in `## Log`.
- [ ] Consolidate `menu.go`'s four rune/backspace handlers into one reducer.
- [ ] `KeyClear` in `panelkeys.go` for `0x15`; clear-all in the reducer.
- [ ] Draft pane binding + undo assertion.
- [ ] Reconcile with `#165` and update both issues.

## Log

### 2026-09-02

The Ghostty binding (`keybind = super+backspace=text:\x15`) is the operator's
own configuration, not something pair ships. Worth knowing when this is tested
on another machine: cmd+delete does nothing there until the same binding
exists, while `^U` typed directly works everywhere. If pair should ship that
binding, that is a separate question about pair owning terminal config.

### 2026-09-02 — closed as a duplicate of `pair#165`

Same feature. `#165` was already open for clearing couch's edit boxes; the
operator's call is that the chord is **cmd+delete**, not `Alt+Delete`, and that
it clears **all** text rather than one line of a multiline field. `#165` is now
the canonical issue and carries everything found here — the dropped `0x15`, the
four `menu.go` text handlers, the draft-pane override and its undo requirement.

Closed as `wontfix` because **the status set has no `dup`**
(`ariadne/construct/vocabulary/issue.cue`: `terminal: ["done", "wontfix",
"punt"]`). `wontfix` reads as "rejected", which is wrong — this work is being
done, under another id. Tracked as `ariadne#209`; re-status this issue once
`dup` exists.
